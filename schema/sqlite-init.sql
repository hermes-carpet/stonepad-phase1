-- Stonepad Server SQLite Schema
-- Version 1
-- Reference schema for the server's SQLite database.

-- Schema version tracking (allows future migrations)
CREATE TABLE schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL
);
INSERT INTO schema_version (version, applied_at) VALUES (1, datetime('now'));

-- Workspaces (v1 has exactly one row: 'default')
CREATE TABLE workspaces (
    workspace_id TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    created_at TEXT NOT NULL
);

-- Users (v1 has exactly one row: 'owner' unless AUTH_MODE=users)
CREATE TABLE users (
    user_id TEXT PRIMARY KEY,
    username TEXT UNIQUE,
    password_hash TEXT,           -- Argon2id hash; NULL when AUTH_MODE != 'users'
    created_at TEXT NOT NULL,
    last_login_at TEXT
);

-- Note metadata (index over filesystem files)
CREATE TABLE notes (
    workspace_id TEXT NOT NULL,
    path TEXT NOT NULL,            -- relative path within workspace, e.g. 'work/meetings/2026-04-22.md'
    content_hash TEXT NOT NULL,    -- SHA-256 of file content (lowercase hex)
    size_bytes INTEGER NOT NULL,
    modified_at TEXT NOT NULL,     -- ISO 8601 UTC
    created_at TEXT NOT NULL,
    PRIMARY KEY (workspace_id, path),
    FOREIGN KEY (workspace_id) REFERENCES workspaces(workspace_id)
);

CREATE INDEX idx_notes_workspace_modified ON notes(workspace_id, modified_at DESC);

-- Audit log (v1 records writes but doesn't expose them via API yet)
CREATE TABLE audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    action TEXT NOT NULL,          -- 'create', 'update', 'delete'
    path TEXT NOT NULL,
    content_hash TEXT,             -- NULL for delete
    occurred_at TEXT NOT NULL,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(workspace_id),
    FOREIGN KEY (user_id) REFERENCES users(user_id)
);

CREATE INDEX idx_audit_workspace_time ON audit_log(workspace_id, occurred_at DESC);

-- Tokens (used when AUTH_MODE=users for session tokens)
CREATE TABLE auth_tokens (
    token_hash TEXT PRIMARY KEY,   -- SHA-256 of the token value
    user_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    expires_at TEXT,
    FOREIGN KEY (user_id) REFERENCES users(user_id)
);
