/// Central state for notes — owns the in-memory manifest and
/// the currently-opened note. Notifies listeners on changes.
library;
import 'package:flutter/foundation.dart';
import '../models/manifest.dart';
import '../models/note_entry.dart';
import '../services/storage_service.dart';

class NotesState extends ChangeNotifier {
  final StorageService _storage;
  Manifest _manifest = Manifest();
  String? _currentNotePath;
  String? _currentNoteContent;

  NotesState(this._storage);

  Manifest get manifest => _manifest;
  String? get currentNotePath => _currentNotePath;
  String? get currentNoteContent => _currentNoteContent;
  List<String> get allPaths => _manifest.notes.keys.toList()..sort();

  /// Load manifest from disk.
  Future<void> loadManifest() async {
    _manifest = await _storage.loadManifest();
    notifyListeners();
  }

  /// Open a note for editing. Loads its content from disk.
  Future<void> openNote(String path) async {
    _currentNotePath = path;
    _currentNoteContent = await _storage.readNote(path);
    notifyListeners();
  }

  /// Update the in-memory note content (debounced save is handled elsewhere).
  void updateNoteContent(String content) {
    _currentNoteContent = content;
    notifyListeners();
  }

  /// Persist the current note and update manifest.
  Future<void> saveCurrentNote() async {
    if (_currentNotePath == null) return;

    final hash = await _storage.writeNote(_currentNotePath!, _currentNoteContent ?? '');
    final now = DateTime.now();

    final existing = _manifest.notes[_currentNotePath!];
    if (existing != null && existing.contentHash == hash) {
      return; // No change — skip write
    }

    _manifest.notes[_currentNotePath!] = NoteEntry(
      contentHash: hash,
      sizeBytes: (_currentNoteContent ?? '').length,
      localModified: now,
      lastSyncedHash: existing?.lastSyncedHash ?? '',
      lastSyncedAt: existing?.lastSyncedAt,
      status: NoteStatus.modified,
    );

    await _storage.saveManifest(_manifest);
    notifyListeners();
  }

  /// Create a new note at the given path.
  Future<void> createNote(String path, {String content = ''}) async {
    final hash = await _storage.writeNote(path, content);
    final now = DateTime.now();

    _manifest.notes[path] = NoteEntry(
      contentHash: hash,
      sizeBytes: content.length,
      localModified: now,
      lastSyncedHash: '',
      status: NoteStatus.modified,
    );

    await _storage.saveManifest(_manifest);
    _currentNotePath = path;
    _currentNoteContent = content;
    notifyListeners();
  }

  /// Delete a note from disk and manifest.
  Future<void> deleteNote(String path) async {
    await _storage.deleteNote(path);
    _manifest.notes.remove(path);
    if (_currentNotePath == path) {
      _currentNotePath = null;
      _currentNoteContent = null;
    }
    await _storage.saveManifest(_manifest);
    notifyListeners();
  }

  /// Delete an entire folder and all notes within it.
  Future<void> deleteFolder(String folderPath) async {
    final prefix = '$folderPath/';
    final toDelete = _manifest.notes.keys.where((p) => p.startsWith(prefix)).toList();
    // Also delete the .folder marker
    toDelete.add('$folderPath/.folder');
    
    for (final path in toDelete) {
      await _storage.deleteNote(path);
      _manifest.notes.remove(path);
      if (_currentNotePath == path) {
        _currentNotePath = null;
        _currentNoteContent = null;
      }
    }
    await _storage.saveManifest(_manifest);
    notifyListeners();
  }

  /// Mark a note as synced in the manifest.
  Future<void> markSynced(String path, String serverHash) async {
    final entry = _manifest.notes[path];
    if (entry == null) return;

    _manifest.notes[path] = entry.copyWith(
      lastSyncedHash: serverHash,
      lastSyncedAt: DateTime.now(),
      status: NoteStatus.synced,
    );

    await _storage.saveManifest(_manifest);
    notifyListeners();
  }
}
