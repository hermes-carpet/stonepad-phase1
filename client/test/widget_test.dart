import 'package:flutter_test/flutter_test.dart';
import 'package:stonepad/models/manifest.dart';
import 'package:stonepad/models/note_entry.dart';

void main() {
  test('Manifest serialization round-trip', () {
    final manifest = Manifest(
      workspaceId: 'default',
      notes: {
        'hello.md': NoteEntry(
          contentHash: 'abc123',
          sizeBytes: 100,
          localModified: DateTime(2026, 4, 22),
          lastSyncedHash: 'abc123',
          status: NoteStatus.synced,
        ),
      },
    );

    final json = manifest.toJson();
    final restored = Manifest.fromJson(json);

    expect(restored.version, 1);
    expect(restored.workspaceId, 'default');
    expect(restored.notes.length, 1);
    expect(restored.notes['hello.md']!.contentHash, 'abc123');
    expect(restored.notes['hello.md']!.status, NoteStatus.synced);
  });

  test('Manifest rejects unknown version', () {
    expect(
      () => Manifest.fromJson({'version': 99, 'notes': {}}),
      throwsFormatException,
    );
  });

  test('NoteEntry status parsing', () {
    final entry = NoteEntry.fromJson({
      'content_hash': 'abc',
      'size_bytes': 10,
      'local_modified': '2026-04-22T00:00:00.000Z',
      'last_synced_hash': '',
      'status': 'modified',
    });
    expect(entry.status, NoteStatus.modified);

    // Unknown status defaults to synced
    final entry2 = NoteEntry.fromJson({
      'content_hash': 'abc',
      'size_bytes': 10,
      'local_modified': '2026-04-22T00:00:00.000Z',
      'last_synced_hash': '',
      'status': 'unknown',
    });
    expect(entry2.status, NoteStatus.synced);
  });
}
