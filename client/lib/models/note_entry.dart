/// Entry in the manifest for a single note.
/// Tracks local content hash and sync status. See §8.4.
enum NoteStatus { synced, modified, deleted, conflictPending }

class NoteEntry {
  final String contentHash;
  final int sizeBytes;
  final DateTime localModified;
  final String lastSyncedHash;
  final DateTime? lastSyncedAt;
  final NoteStatus status;

  const NoteEntry({
    required this.contentHash,
    required this.sizeBytes,
    required this.localModified,
    required this.lastSyncedHash,
    this.lastSyncedAt,
    this.status = NoteStatus.synced,
  });

  factory NoteEntry.fromJson(Map<String, dynamic> json) {
    return NoteEntry(
      contentHash: json['content_hash'] ?? '',
      sizeBytes: json['size_bytes'] ?? 0,
      localModified: DateTime.parse(json['local_modified']),
      lastSyncedHash: json['last_synced_hash'] ?? '',
      lastSyncedAt: json['last_synced_at'] != null
          ? DateTime.parse(json['last_synced_at'])
          : null,
      status: _parseStatus(json['status']),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'content_hash': contentHash,
      'size_bytes': sizeBytes,
      'local_modified': localModified.toUtc().toIso8601String(),
      'last_synced_hash': lastSyncedHash,
      if (lastSyncedAt != null)
        'last_synced_at': lastSyncedAt!.toUtc().toIso8601String(),
      'status': status.name,
    };
  }

  static NoteStatus _parseStatus(dynamic s) {
    final str = (s ?? 'synced').toString();
    return NoteStatus.values.firstWhere(
      (e) => e.name == str,
      orElse: () => NoteStatus.synced,
    );
  }

  /// Creates a copy with updated fields (used for marking as modified, etc.).
  NoteEntry copyWith({
    String? contentHash,
    int? sizeBytes,
    DateTime? localModified,
    String? lastSyncedHash,
    DateTime? lastSyncedAt,
    NoteStatus? status,
  }) {
    return NoteEntry(
      contentHash: contentHash ?? this.contentHash,
      sizeBytes: sizeBytes ?? this.sizeBytes,
      localModified: localModified ?? this.localModified,
      lastSyncedHash: lastSyncedHash ?? this.lastSyncedHash,
      lastSyncedAt: lastSyncedAt ?? this.lastSyncedAt,
      status: status ?? this.status,
    );
  }
}
