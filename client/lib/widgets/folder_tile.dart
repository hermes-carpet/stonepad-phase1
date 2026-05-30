/// Folder tile — displays a folder in the notes list hierarchy.
/// See §8.10 of the Stonepad v1 Implementation Plan.
library;
import 'package:flutter/material.dart';

class FolderTile extends StatelessWidget {
  final String folderPath;
  final VoidCallback onTap;
  final VoidCallback onLongPress;

  const FolderTile({
    super.key,
    required this.folderPath,
    required this.onTap,
    required this.onLongPress,
  });

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: const Icon(Icons.folder, color: Colors.amber),
      title: Text(folderPath.split('/').last),
      trailing: const Icon(Icons.chevron_right),
      onTap: onTap,
      onLongPress: onLongPress,
    );
  }
}
