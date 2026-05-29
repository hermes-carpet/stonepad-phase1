// Package auth implements username/password authentication with Argon2id hashing.
// Uses session tokens stored in the auth_tokens table.
// On first startup with empty users table, generates a random admin password.
// See §7.7 for the full spec.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters — MUST be exactly these values for v1.
// See §14.3 of the Stonepad v1 Implementation Plan.
const (
	argonTime        = 3
	argonMemory      = 64 * 1024 // 64 MiB in KiB
	argonParallelism = 2
	argonKeyLen      = 32
	argonSaltLen     = 16
)

// UsersAuth implements username/password authentication backed by SQLite.
type UsersAuth struct {
	db *sql.DB
}

// NewUsersAuth creates a new UsersAuth backed by the given database connection.
func NewUsersAuth(db *sql.DB) *UsersAuth {
	return &UsersAuth{db: db}
}

// GenerateInitialAdmin creates an admin user with a random password on first run.
// Returns the generated password (which should be logged ONCE and not stored).
// This should only be called when the users table is empty.
func (a *UsersAuth) GenerateInitialAdmin() (string, error) {
	password := generateRandomPassword(16)
	hash, err := HashPassword(password)
	if err != nil {
		return "", fmt.Errorf("hashing admin password: %w", err)
	}

	_, err = a.db.Exec(
		`INSERT OR IGNORE INTO users (user_id, username, password_hash, created_at)
		 VALUES (?, ?, ?, ?)`,
		"admin", "admin", hash, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return "", fmt.Errorf("inserting admin user: %w", err)
	}

	return password, nil
}

// Authenticate validates a session token from the Authorization header.
func (a *UsersAuth) Authenticate(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	token, ok := strings.CutPrefix(authHeader, "Bearer ")
	if !ok {
		return "", fmt.Errorf("invalid Authorization header format: expected 'Bearer <token>'")
	}

	tokenHash := hashToken(token)

	var userID string
	var expiresAt sql.NullString
	err := a.db.QueryRow(
		`SELECT user_id, expires_at FROM auth_tokens WHERE token_hash = ?`,
		tokenHash,
	).Scan(&userID, &expiresAt)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("invalid token")
	}
	if err != nil {
		return "", fmt.Errorf("querying token: %w", err)
	}

	// Check expiry
	if expiresAt.Valid {
		expiry, err := time.Parse(time.RFC3339, expiresAt.String)
		if err == nil && time.Now().UTC().After(expiry) {
			// Token expired — delete it
			a.db.Exec(`DELETE FROM auth_tokens WHERE token_hash = ?`, tokenHash)
			return "", fmt.Errorf("token expired")
		}
	}

	return userID, nil
}

// ValidateLogin checks username/password and returns a session token.
func (a *UsersAuth) ValidateLogin(username, password string) (string, error) {
	var userID, storedHash string
	err := a.db.QueryRow(
		`SELECT user_id, password_hash FROM users WHERE username = ?`,
		username,
	).Scan(&userID, &storedHash)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("invalid username or password")
	}
	if err != nil {
		return "", fmt.Errorf("querying user: %w", err)
	}

	if !VerifyPassword(password, storedHash) {
		return "", fmt.Errorf("invalid username or password")
	}

	// Generate session token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	tokenHash := hashToken(token)
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339)

	_, err = a.db.Exec(
		`INSERT INTO auth_tokens (token_hash, user_id, created_at, expires_at)
		 VALUES (?, ?, ?, ?)`,
		tokenHash, userID, time.Now().UTC().Format(time.RFC3339), expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("storing token: %w", err)
	}

	// Update last login
	a.db.Exec(
		`UPDATE users SET last_login_at = ? WHERE user_id = ?`,
		time.Now().UTC().Format(time.RFC3339), userID,
	)

	return token, nil
}

// InvalidateToken removes a session token from the database.
func (a *UsersAuth) InvalidateToken(token string) error {
	tokenHash := hashToken(token)
	_, err := a.db.Exec(`DELETE FROM auth_tokens WHERE token_hash = ?`, tokenHash)
	return err
}

// HashPassword hashes a password using Argon2id with the v1 fixed parameters.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonParallelism, argonKeyLen)

	// Format: base64(salt):base64(hash)
	encoded := base64.RawStdEncoding.EncodeToString(salt) + ":" +
		base64.RawStdEncoding.EncodeToString(hash)

	return encoded, nil
}

// VerifyPassword checks a password against an Argon2id hash.
func VerifyPassword(password, encoded string) bool {
	parts := strings.SplitN(encoded, ":", 2)
	if len(parts) != 2 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	actualHash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonParallelism, argonKeyLen)

	return subtle.ConstantTimeCompare(expectedHash, actualHash) == 1
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}

func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			// Fallback to simple rand on error (should never happen)
			b[i] = charset[i%len(charset)]
			continue
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}
