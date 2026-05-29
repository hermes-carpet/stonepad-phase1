/// ApiClient — native REST API client for Stonepad server.
///
/// Handles non-S3 endpoints:
///   - POST /api/v1/auth/login  (username + password → session token)
///   - POST /api/v1/auth/logout (invalidate session token)
///   - GET  /api/v1/health      (connection test)
///
/// Auth headers are set based on the configured auth mode:
///   - none:   no Authorization header
///   - token:  Authorization: Bearer ${NOTES_AUTH_TOKEN}
///   - users:  Authorization: Bearer ${session_token}
///
/// S3 operations are handled by SyncService via the minio package.
library;

import 'dart:convert';
import 'package:http/http.dart' as http;

class ApiClient {
  final String baseUrl;
  String? _bearerToken;
  final http.Client _httpClient;

  ApiClient({
    required this.baseUrl,
    String? bearerToken,
    http.Client? httpClient,
  })  : _bearerToken = bearerToken,
        _httpClient = httpClient ?? http.Client();

  /// Set the bearer token for authenticated requests.
  void setBearerToken(String? token) {
    _bearerToken = token;
  }

  /// Build common headers including auth.
  Map<String, String> _headers() {
    final headers = <String, String>{
      'Content-Type': 'application/json',
    };
    if (_bearerToken != null && _bearerToken!.isNotEmpty) {
      headers['Authorization'] = 'Bearer $_bearerToken';
    }
    return headers;
  }

  // ────────────────────── Auth endpoints ──────────────────────

  /// Login with username/password. Returns a session token.
  /// Only works when the server is in `users` auth mode.
  Future<LoginResult> login(String username, String password) async {
    final uri = Uri.parse('$baseUrl/api/v1/auth/login');
    try {
      final response = await _httpClient.post(
        uri,
        headers: {'Content-Type': 'application/json'},
        body: jsonEncode({
          'username': username,
          'password': password,
        }),
      );

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body) as Map<String, dynamic>;
        final token = data['token'] as String;
        _bearerToken = token;
        return LoginResult.ok(token);
      }

      if (response.statusCode == 401) {
        return LoginResult.error('Invalid username or password');
      }

      if (response.statusCode == 500) {
        return LoginResult.error(
            'Server does not support users auth mode');
      }

      return LoginResult.error(
          'Login failed (HTTP ${response.statusCode})');
    } catch (e) {
      return LoginResult.error('Connection failed: $e');
    }
  }

  /// Logout — invalidates the current session token on the server.
  Future<void> logout() async {
    if (_bearerToken == null) return;
    final uri = Uri.parse('$baseUrl/api/v1/auth/logout');
    try {
      await _httpClient.post(uri, headers: _headers());
    } catch (_) {
      // Best effort — clear token locally even if server is unreachable
    }
    _bearerToken = null;
  }

  // ────────────────────── Health / test ──────────────────────

  /// Test the connection to the server.
  /// Returns true if the health endpoint responds with 200 OK.
  Future<bool> testConnection() async {
    final uri = Uri.parse('$baseUrl/api/v1/health');
    try {
      final response = await _httpClient.get(uri).timeout(
        const Duration(seconds: 2),
      );
      return response.statusCode == 200;
    } catch (_) {
      return false;
    }
  }

  /// Dispose the underlying HTTP client.
  void dispose() {
    _httpClient.close();
  }
}

/// Result of a login attempt.
class LoginResult {
  final bool success;
  final String? token;
  final String? error;

  const LoginResult._({required this.success, this.token, this.error});

  factory LoginResult.ok(String token) =>
      LoginResult._(success: true, token: token);

  factory LoginResult.error(String message) =>
      LoginResult._(success: false, error: message);
}
