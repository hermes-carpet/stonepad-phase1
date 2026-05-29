/// Manifest model — tracks sync state of all notes in the workspace.
/// Format is versioned; code must check the version field and refuse
/// to load unknown versions. See §8.4.
library;
import 'dart:convert';
import 'note_entry.dart';

class Manifest {
  final int version;
  final String workspaceId;
  DateTime? lastSyncAttempt;
  DateTime? lastSyncSuccess;
  final Map<String, NoteEntry> notes;

  Manifest({
    this.version = 1,
    this.workspaceId = 'default',
    this.lastSyncAttempt,
    this.lastSyncSuccess,
    Map<String, NoteEntry>? notes,
  }) : notes = notes ?? {};

  factory Manifest.fromJson(Map<String, dynamic> json) {
    final version = json['version'] as int? ?? 1;
    if (version != 1) {
      throw FormatException('Unknown manifest version: $version');
    }

    final notesJson = json['notes'] as Map<String, dynamic>? ?? {};
    final notes = <String, NoteEntry>{};
    for (final entry in notesJson.entries) {
      notes[entry.key] = NoteEntry.fromJson(entry.value);
    }

    return Manifest(
      version: version,
      workspaceId: json['workspace_id'] ?? 'default',
      lastSyncAttempt: json['last_sync_attempt'] != null
          ? DateTime.parse(json['last_sync_attempt'])
          : null,
      lastSyncSuccess: json['last_sync_success'] != null
          ? DateTime.parse(json['last_sync_success'])
          : null,
      notes: notes,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'version': version,
      'workspace_id': workspaceId,
      if (lastSyncAttempt != null)
        'last_sync_attempt': lastSyncAttempt!.toUtc().toIso8601String(),
      if (lastSyncSuccess != null)
        'last_sync_success': lastSyncSuccess!.toUtc().toIso8601String(),
      'notes': notes.map((k, v) => MapEntry(k, v.toJson())),
    };
  }

  /// Load a manifest from a JSON string.
  factory Manifest.fromString(String jsonStr) {
    return Manifest.fromJson(jsonDecode(jsonStr));
  }

  /// Serialize to a JSON string.
  @override
  66|  String toString() => jsonEncode(toJson());

  /// Notes with local changes pending sync.
  List<String> get modifiedPaths =>
      notes.entries.where((e) => e.value.status == NoteStatus.modified).map((e) => e.key).toList();

  /// Notes pending deletion from server.
  List<String> get deletedPaths =>
      notes.entries.where((e) => e.value.status == NoteStatus.deleted).map((e) => e.key).toList();

  /// Notes with unresolved conflicts.
  List<String> get conflictPaths =>
      notes.entries.where((e) => e.value.status == NoteStatus.conflictPending).map((e) => e.key).toList();
}
