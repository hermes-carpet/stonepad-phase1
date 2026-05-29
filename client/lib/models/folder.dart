/// Folder model for hierarchy display.
/// Folders are derived from note paths — the filesystem hierarchy IS the data model.
class Folder {
  final String name;
  final String path;
  final List<Folder> subFolders;
  final int noteCount;

  const Folder({
    required this.name,
    required this.path,
    this.subFolders = const [],
    this.noteCount = 0,
  });

  @override
  String toString() => 'Folder($path)';
}
