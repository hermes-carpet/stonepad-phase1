/// Service for reading/writing notes and manifest to local disk.
/// All writes use atomic semantics: write to temp file, then rename.
/// See §8.5 and §9.4 of the Stonepad v1 Implementation Plan.
library;
import 'dart:convert';
import 'dart:io';
import 'package:crypto/crypto.dart';
import '../constants/paths.dart';
import '../models/manifest.dart';

class StorageService {
  Future<Manifest> loadManifest() async {
    final file = await StonepadPaths.manifestFile();
    if (!await file.exists()) {
      return Manifest();
    }
    final content = await file.readAsString();
    if (content.trim().isEmpty) {
      return Manifest();
    }
    return Manifest.fromString(content);
  }

  Future<void> saveManifest(Manifest manifest) async {
    final file = await StonepadPaths.manifestFile();
    final tmp = File('${file.path}.tmp');
    await tmp.writeAsString(manifest.toString());
    // Atomic rename
    await tmp.rename(file.path);
  }

  Future<String?> readNote(String path) async {
    final notesDir = await StonepadPaths.notesDirectory();
    final file = File('${notesDir.path}/$path');
    if (!await file.exists()) return null;
    return file.readAsString();
  }

  /// Writes note content atomically. Returns the SHA-256 hash.
  Future<String> writeNote(String path, String content) async {
    final notesDir = await StonepadPaths.notesDirectory();
    final file = File('${notesDir.path}/$path');

    // Ensure parent directory exists
    final parent = file.parent;
    if (!await parent.exists()) {
      await parent.create(recursive: true);
    }

    // Compute SHA-256
    final hash = sha256.convert(utf8.encode(content)).toString();

    // Atomic write: temp file then rename
    final tmp = File('${file.path}.tmp');
    await tmp.writeAsString(content);
    await tmp.rename(file.path);

    return hash;
  }

  Future<void> deleteNote(String path) async {
    final notesDir = await StonepadPaths.notesDirectory();
    final file = File('${notesDir.path}/$path');
    if (await file.exists()) {
      await file.delete();
    }
  }

  /// List all .md files under the notes directory.
  Future<List<String>> listNotePaths() async {
    final notesDir = await StonepadPaths.notesDirectory();
    if (!await notesDir.exists()) return [];

    final paths = <String>[];
    await for (final entity in notesDir.list(recursive: true)) {
      if (entity is File && entity.path.endsWith('.md')) {
        // Get relative path from notes directory
        final relPath = entity.path.substring(notesDir.path.length + 1);
        paths.add(relPath);
      }
    }
    paths.sort();
    return paths;
  }

  /// Write a conflict file. See §8.8.
  Future<void> writeConflictFile(String originalPath, String content) async {
    final conflictsDir = await StonepadPaths.conflictsDirectory();
    final timestamp = DateTime.now().toUtc().toIso8601String().replaceAll(':', '-');
    final safePath = originalPath.replaceAll('/', '_');
    final file = File('${conflictsDir.path}/$timestamp-$safePath');
    await file.writeAsString(content);
  }

  /// Extract folder hierarchy from a list of note paths.
  static List<String> extractFolders(List<String> paths) {
    final folders = <String>{};
    for (final path in paths) {
      final parts = path.split('/');
      for (int i = 1; i < parts.length; i++) {
        folders.add(parts.sublist(0, i).join('/'));
      }
    }
    final sorted = folders.toList()..sort();
    return sorted;
  }

  /// Get notes directly in a folder (not subfolders).
  static List<String> notesInFolder(List<String> paths, String folder) {
    final prefix = folder.isEmpty ? '' : '$folder/';
    return paths.where((p) {
      if (!p.startsWith(prefix)) return false;
      final remainder = p.substring(prefix.length);
      return !remainder.contains('/');
    }).toList();
  }

  /// Get immediate subfolders of a folder.
  static List<String> subFolders(List<String> paths, String folder) {
    final prefix = folder.isEmpty ? '' : '$folder/';
    final subFolders = <String>{};
    for (final path in paths) {
      if (!path.startsWith(prefix)) continue;
      final remainder = path.substring(prefix.length);
      final parts = remainder.split('/');
      if (parts.length > 1) {
        final subFolder = folder.isEmpty ? parts[0] : '$folder/${parts[0]}';
        subFolders.add(subFolder);
      }
    }
    final sorted = subFolders.toList()..sort();
    return sorted;
  }
}
