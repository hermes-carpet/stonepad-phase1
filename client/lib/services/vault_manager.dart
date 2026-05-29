/// Split-platform vault path manager.
///
/// **Android**: Forces the user to pick a folder via the Storage Access Framework
///   on first launch. Saves the chosen path to SharedPreferences so subsequent
///   launches skip the picker. The picked folder lives on public storage (e.g.,
///   `/storage/emulated/0/Documents/MyNotes/`) — no root/scoped storage issues.
///
/// **iOS**: Automatically uses `getApplicationDocumentsDirectory()`. With
///   `UIFileSharingEnabled` and `LSSupportsOpeningDocumentsInPlace` in
///   Info.plist, this path is visible and accessible in the iOS Files app.
///
/// **Desktop**: Uses `$HOME/.local/share/stonepad/` — no picker, no prefs.
library;

import 'dart:io';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:path_provider/path_provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

class VaultManager {
  static const _keyVaultPath = 'vault_path';

  /// Call when the app starts up. Returns the vault path, or null if Android
  /// needs the user to pick a folder first.
  static Future<String?> getOrCreateVaultPath() async {
    final prefs = await SharedPreferences.getInstance();
    final savedPath = prefs.getString(_keyVaultPath);

    // Path already configured — use it.
    if (savedPath != null && savedPath.isNotEmpty) {
      return savedPath;
    }

    // Desktop: fixed path, no user interaction needed.
    if (!Platform.isAndroid && !Platform.isIOS) {
      final home = Platform.environment['HOME'] ?? '/tmp';
      final path = '$home/.local/share/stonepad';
      final dir = Directory(path);
      if (!await dir.exists()) {
        await dir.create(recursive: true);
      }
      await prefs.setString(_keyVaultPath, path);
      return path;
    }

    // iOS: app documents directory — visible in Files app automatically.
    if (Platform.isIOS) {
      final directory = await getApplicationDocumentsDirectory();
      await prefs.setString(_keyVaultPath, directory.path);
      return directory.path;
    }

    // Android: return null → UI must show the folder picker.
    return null;
  }

  /// Call when the Android user taps "Choose Folder". Returns the chosen path
  /// or null if the user cancelled.
  static Future<String?> pickAndroidFolder() async {
    try {
      final selectedDirectory = await FilePicker.platform.getDirectoryPath(
        dialogTitle: 'Choose where to store your notes',
      );

      if (selectedDirectory != null && selectedDirectory.isNotEmpty) {
        final prefs = await SharedPreferences.getInstance();
        await prefs.setString(_keyVaultPath, selectedDirectory);
        return selectedDirectory;
      }
    } catch (e) {
      debugPrint('VaultManager: folder picker failed: $e');
    }
    return null;
  }

  /// Reset the vault path (e.g., when changing folders).
  static Future<void> clearVaultPath() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_keyVaultPath);
  }
}
