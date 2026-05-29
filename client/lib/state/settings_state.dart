/// Settings state — persists user settings to settings.json.
import 'dart:convert';
import 'dart:io';
import 'package:flutter/foundation.dart';
import '../constants/paths.dart';
import '../models/settings.dart';

class SettingsState extends ChangeNotifier {
  StonepadSettings _settings = StonepadSettings();

  StonepadSettings get settings => _settings;

  /// Load settings from disk, or create defaults.
  Future<void> load() async {
    final file = await StonepadPaths.settingsFile();
    if (await file.exists()) {
      try {
        final json = jsonDecode(await file.readAsString());
        _settings = StonepadSettings.fromJson(json);
      } catch (_) {
        _settings = StonepadSettings();
      }
    }
    notifyListeners();
  }

  /// Persist settings to disk.
  Future<void> save() async {
    final file = await StonepadPaths.settingsFile();
    final tmp = File('${file.path}.tmp');
    await tmp.writeAsString(jsonEncode(_settings.toJson()));
    await tmp.rename(file.path);
  }

  // Individual setters
  Future<void> setServerEndpoint(String? value) async {
    _settings.serverEndpoint = value;
    await save();
    notifyListeners();
  }

  Future<void> setAuthMode(String mode) async {
    _settings.authMode = mode;
    await save();
    notifyListeners();
  }

  Future<void> setAuthToken(String? value) async {
    _settings.authToken = value;
    await save();
    notifyListeners();
  }

  Future<void> setS3Keys(String? accessKey, String? secretKey) async {
    _settings.s3AccessKey = accessKey;
    _settings.s3SecretKey = secretKey;
    await save();
    notifyListeners();
  }

  Future<void> setWorkspaceId(String value) async {
    _settings.workspaceId = value;
    await save();
    notifyListeners();
  }

  Future<void> setSyncEnabled(bool value) async {
    _settings.syncEnabled = value;
    await save();
    notifyListeners();
  }

  Future<void> setRelayConfig({
    bool? enabled,
    String? endpoint,
    String? accessKey,
    String? secretKey,
  }) async {
    if (enabled != null) _settings.relayEnabled = enabled;
    if (endpoint != null) _settings.relayEndpoint = endpoint;
    if (accessKey != null) _settings.relayAccessKey = accessKey;
    if (secretKey != null) _settings.relaySecretKey = secretKey;
    await save();
    notifyListeners();
  }
}
