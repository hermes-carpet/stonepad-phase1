/// Filesystem path helpers for the client storage layout.
/// Centralizes platform-specific path resolution so the rest of the code
/// doesn't branch on platform. See §8.3 of the Stonepad v1 Plan.
library;
import 'dart:io';
import 'package:path_provider/path_provider.dart';

class StonepadPaths {
  StonepadPaths._();

  /// Returns the base directory where Stonepad stores all data.
  ///
  /// Android: external app-specific storage
  ///   (/storage/emulated/0/Android/data/io.stonepad.app/files/)
  /// iOS: application documents directory
  /// Linux (dev): $HOME/.local/share/stonepad/
  static Future<Directory> getBaseDirectory() async {
    if (Platform.isAndroid) {
      final dir = await getExternalStorageDirectory();
      if (dir == null) {
        throw StateError('External storage unavailable on Android');
      }
      return dir;
    }
    if (Platform.isLinux || Platform.isWindows || Platform.isMacOS) {
      final home = Platform.environment['HOME'] ?? '/tmp';
      final dir = Directory('$home/.local/share/stonepad');
      if (!await dir.exists()) {
        await dir.create(recursive: true);
      }
      return dir;
    }
    return getApplicationDocumentsDirectory();
  }

  /// Returns the notes directory within the base directory.
  static Future<Directory> notesDirectory() async {
    final base = await getBaseDirectory();
    final dir = Directory('${base.path}/notes/default');
    if (!await dir.exists()) {
      await dir.create(recursive: true);
    }
    return dir;
  }

  /// Returns the conflicts directory.
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
