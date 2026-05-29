/// Filesystem path helpers for the client storage layout.
///
/// All paths are rooted under the user's vault directory, managed by
/// [VaultManager]. On Android this is a user-picked public folder; on iOS
/// it's the app documents directory (visible in Files); on desktop it's
/// `$HOME/.local/share/stonepad/`.
///
/// See §8.3 of the Stonepad v1 Implementation Plan.
library;

import 'dart:io';
import '../services/vault_manager.dart';

class StonepadPaths {
  StonepadPaths._();

  /// Returns the base vault directory.
  static Future<Directory> getBaseDirectory() async {
    final path = await VaultManager.getOrCreateVaultPath();
    if (path == null) {
      throw StateError(
        'Vault path not configured. Call VaultManager.pickAndroidFolder() '
        'on Android before accessing StonepadPaths.',
      );
    }
    final dir = Directory(path);
    if (!await dir.exists()) {
      await dir.create(recursive: true);
    }
    return dir;
  }

  /// Notes directory within the vault.
  static Future<Directory> notesDirectory() async {
    final base = await getBaseDirectory();
    final dir = Directory('${base.path}/notes/default');
    if (!await dir.exists()) {
      await dir.create(recursive: true);
    }
    return dir;
  }

  /// Conflicts directory.
  static Future<Directory> conflictsDirectory() async {
    final base = await getBaseDirectory();
    final dir = Directory('${base.path}/conflicts');
    if (!await dir.exists()) {
      await dir.create(recursive: true);
    }
    return dir;
  }

  /// Manifest file path.
  static Future<File> manifestFile() async {
    final base = await getBaseDirectory();
    return File('${base.path}/manifest.json');
  }

  /// Settings file path.
  static Future<File> settingsFile() async {
    final base = await getBaseDirectory();
    return File('${base.path}/settings.json');
  }
}
