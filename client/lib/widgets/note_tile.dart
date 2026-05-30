/// Note tile — displays a note entry in the notes list with sync status icon.
/// See §8.10 of the Stonepad v1 Implementation Plan.
library;
import 'package:flutter/material.dart';

class NoteTile extends StatelessWidget {
  final String notePath;
  final String? status;
  final VoidCallback onTap;
  final VoidCallback onLongPress;

  const NoteTile({
    super.key,
    required this.notePath,
    required this.status,
    required this.onTap,
    required this.onLongPress,
  });

  @override
  Widget build(BuildContext context) {
    final displayName = notePath.split('/').last.replaceAll('.md', '');
    return ListTile(
      leading: const Icon(Icons.description, color: Colors.grey),
      title: Text(displayName),
      subtitle: Text(notePath),
      trailing: _buildStatusIcon(),
      onTap: onTap,
      onLongPress: onLongPress,
    );
  }

  Widget? _buildStatusIcon() {
    if (status == null) return null;
    switch (status) {
      case 'modified':
        return const Icon(Icons.cloud_upload, size: 16, color: Colors.orange);
      case 'conflict_pending':
        return const Icon(Icons.warning, size: 16, color: Colors.red);
      default:
        return null;
    }
  }
}
