/// Note model — a markdown note identified by its path within the workspace.
/// The path IS the canonical identifier (no UUIDs). See §4.
class Note {
  final String path;
  final String content;
  final String contentHash;
  final int sizeBytes;
  final DateTime modifiedAt;

  const Note({
    required this.path,
    required this.content,
    required this.contentHash,
    required this.sizeBytes,
    required this.modifiedAt,
  });

  /// Returns the note's filename (last component of path).
  String get filename => path.split('/').last;

  /// Returns the parent folder path, or empty string for root.
  String get parentPath {
    final parts = path.split('/');
    if (parts.length <= 1) return '';
    return parts.sublist(0, parts.length - 1).join('/');
  }

  @override
  String toString() => 'Note($path)';
}
