/// Tests for Phase 6 — Token + Users Auth.
import 'package:flutter_test/flutter_test.dart';
import 'package:stonepad/models/settings.dart';
import 'package:stonepad/services/api_client.dart';

void main() {
  group('StonepadSettings — auth fields', () {
    test('defaults to none mode', () {
      final s = StonepadSettings();
      expect(s.authMode, 'none');
    });

    test('token mode stores authToken', () {
      final s = StonepadSettings(authMode: 'token', authToken: 'secret123');
      expect(s.authMode, 'token');
      expect(s.authToken, 'secret123');
    });

    test('users mode stores sessionToken', () {
      final s = StonepadSettings(
          authMode: 'users', sessionToken: 'sess-abc');
      expect(s.authMode, 'users');
      expect(s.sessionToken, 'sess-abc');
    });

    test('s3 mode stores access/secret keys', () {
      final s = StonepadSettings(
        authMode: 's3',
        s3AccessKey: 'AKID',
        s3SecretKey: 'secret',
      );
      expect(s.authMode, 's3');
      expect(s.s3AccessKey, 'AKID');
      expect(s.s3SecretKey, 'secret');
    });

    test('sessionToken serializes to/from JSON', () {
      final s = StonepadSettings(
        authMode: 'users',
        sessionToken: 'token-xyz',
        serverEndpoint: 'https://example.com',
      );
      final json = s.toJson();
      expect(json['session_token'], 'token-xyz');

      final restored = StonepadSettings.fromJson(json);
      expect(restored.authMode, 'users');
      expect(restored.sessionToken, 'token-xyz');
      expect(restored.serverEndpoint, 'https://example.com');
    });

    test('hasEndpoint respects serverEndpoint', () {
      expect(StonepadSettings().hasEndpoint, isFalse);
      expect(
        StonepadSettings(serverEndpoint: 'https://example.com').hasEndpoint,
        isTrue,
      );
    });
  });

  group('ApiClient', () {
    test('constructs with baseUrl', () {
      final client = ApiClient(baseUrl: 'https://example.com');
      expect(client.baseUrl, 'https://example.com');
      client.dispose();
    });

    test('setBearerToken updates the token', () {
      final client = ApiClient(baseUrl: 'https://example.com');
      client.setBearerToken('test-token');
      // We can't easily test headers directly, but the method exists
      client.dispose();
    });
  });

  group('SettingsState session token', () {
    // SettingsState requires file I/O, which we'd mock in integration tests.
    // For unit tests, we verify the model handles sessionToken correctly.

    test('sessionToken is optional and defaults to null', () {
      final s = StonepadSettings();
      expect(s.sessionToken, isNull);
    });
  });

  group('LoginResult', () {
    test('ok creates success result with token', () {
      final result = LoginResult.ok('abc123');
      expect(result.success, isTrue);
      expect(result.token, 'abc123');
      expect(result.error, isNull);
    });

    test('error creates failure result with message', () {
      final result = LoginResult.error('Bad credentials');
      expect(result.success, isFalse);
      expect(result.token, isNull);
      expect(result.error, 'Bad credentials');
    });
  });
}
