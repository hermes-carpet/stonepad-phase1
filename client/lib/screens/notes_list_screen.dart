/// Notes List Screen — Apple Notes-style folder hierarchy browser.
/// Shows folders first, then notes alphabetically. Supports create, rename, delete.
/// See §8.10 of the Stonepad v1 Implementation Plan.
library;
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/sync_state.dart';
import '../state/notes_state.dart';
import '../state/sync_state_notifier.dart';
import '../services/storage_service.dart';
import '../services/sync_service.dart';
import 'note_editor_screen.dart';

class NotesListScreen extends StatefulWidget {
  const NotesListScreen({super.key});

  @override
  State<NotesListScreen> createState() => _NotesListScreenState();
}

class _NotesListScreenState extends State<NotesListScreen> {
  String _currentFolder = '';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<NotesState>().loadManifest();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Consumer<NotesState>(
      builder: (context, notesState, _) {
        final allPaths = notesState.allPaths;
        final folders = StorageService.subFolders(allPaths, _currentFolder);
        final notes = StorageService.notesInFolder(allPaths, _currentFolder);

        return Scaffold(
          appBar: AppBar(
            title: _currentFolder.isEmpty
                ? const Text('Stonepad')
                : Text(_buildBreadcrumb()),
            actions: [
              // Sync status indicator + manual sync button
              Consumer<SyncStateNotifier>(
                builder: (context, syncState, _) {
                  return Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      // Sync state icon
                      _syncStateIcon(syncState.state),
                      // Manual sync button
                      IconButton(
                        icon: const Icon(Icons.sync),
                        tooltip: 'Sync now',
                        onPressed: () {
                          final syncService = context.read<SyncService>();
                          syncService.manualSync();
                        },
                      ),
                    ],
                  );
                },
              ),
              IconButton(
                icon: const Icon(Icons.create_new_folder_outlined),
                tooltip: 'New Folder',
                onPressed: () => _createFolder(),
              ),
              IconButton(
                icon: const Icon(Icons.note_add),
                tooltip: 'New Note',
                onPressed: () => _createNote(),
              ),
              IconButton(
                icon: const Icon(Icons.settings),
                onPressed: () => Navigator.pushNamed(context, '/settings'),
              ),
            ],
          ),
          body: ListView(
            children: [
              if (_currentFolder.isNotEmpty)
                ListTile(
                  leading: const Icon(Icons.arrow_back),
                  title: const Text('..'),
                  onTap: () => _navigateUp(),
                ),
              ...folders.map((f) => ListTile(
                    leading: const Icon(Icons.folder, color: Colors.amber),
                    title: Text(f.split('/').last),
                    trailing: const Icon(Icons.chevron_right),
                    onTap: () => setState(() => _currentFolder = f),
                    onLongPress: () => _showFolderActions(f),
                  )),
              ...notes
                .where((p) => !p.endsWith('/.folder'))
                .map((notePath) {
                final entry = notesState.manifest.notes[notePath];
                if (entry == null) return const SizedBox.shrink();
                final statusIcon = _syncStatusIcon(entry.status.name);
                return ListTile(
                  leading: const Icon(Icons.description, color: Colors.grey),
                  title: Text(notePath.split('/').last.replaceAll('.md', '')),
                  subtitle: Text(notePath),
                  trailing: statusIcon,
                  onTap: () => _openNote(notesState, notePath),
                  onLongPress: () => _showNoteActions(notesState, notePath),
                );
              }),
              if (folders.isEmpty && notes.isEmpty)
                Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 64),
                  child: Center(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(
                          Icons.auto_stories,
                          size: 48,
                          color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.15),
                        ),
                        const SizedBox(height: 16),
                        Text(
                          'No notes yet',
                          style: Theme.of(context).textTheme.titleMedium?.copyWith(
                            color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.4),
                          ),
                        ),
                        const SizedBox(height: 8),
                        Text(
                          'Tap + to create your first note.',
                          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                            color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.3),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
            ],
          ),
        );
      },
    );
  }

  Widget? _syncStatusIcon(String status) {
    switch (status) {
      case 'modified':
        return const Icon(Icons.cloud_upload, size: 16, color: Colors.orange);
      case 'conflict_pending':
        return const Icon(Icons.warning, size: 16, color: Colors.red);
      default:
        return null;
    }
  }

  /// Visual indicator for the sync state machine state.
  Widget _syncStateIcon(SyncState state) {
    switch (state) {
      case SyncState.active:
        return const Icon(Icons.cloud_done, size: 18, color: Colors.green);
      case SyncState.manualOnly:
        return const Icon(Icons.cloud_off, size: 18, color: Colors.orange);
      case SyncState.noNetwork:
        return const Icon(Icons.cloud_off, size: 18, color: Colors.grey);
      case SyncState.disabled:
        return const Icon(Icons.cloud_off, size: 18, color: Colors.grey);
    }
  }

  String _buildBreadcrumb() {
    if (_currentFolder.isEmpty) return 'Stonepad';
    final parts = _currentFolder.split('/');
    return parts.join(' / ');
  }

  void _navigateUp() {
    final parts = _currentFolder.split('/');
    if (parts.length <= 1) {
      setState(() => _currentFolder = '');
    } else {
      setState(() => _currentFolder = parts.sublist(0, parts.length - 1).join('/'));
    }
  }

  void _openNote(NotesState notesState, String path) {
    notesState.openNote(path).then((_) {
      if (!context.mounted) return;
      Navigator.push(
        context,
        MaterialPageRoute(
          builder: (_) => ChangeNotifierProvider.value(
            value: notesState,
            child: NoteEditorScreen(notePath: path),
          ),
        ),
      );
    });
  }

  Future<void> _createNote() async {
    final nameController = TextEditingController();
    final result = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('New Note'),
        content: TextField(
          controller: nameController,
          autofocus: true,
          decoration: const InputDecoration(
            hintText: 'Note name (e.g. shopping-list)',
            suffixText: '.md',
          ),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, nameController.text),
            child: const Text('Create'),
          ),
        ],
      ),
    );

    if (result != null && result.isNotEmpty) {
      if (!context.mounted) return;
      final folderPrefix = _currentFolder.isEmpty ? '' : '$_currentFolder/';
      final path = '$folderPrefix$result.md';
      final notesState = context.read<NotesState>();
      await notesState.createNote(path);
      if (mounted) {
        _openNote(notesState, path);
      }
    }
  }

  Future<void> _createFolder() async {
    final nameController = TextEditingController();
    final result = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('New Folder'),
        content: TextField(
          controller: nameController,
          autofocus: true,
          decoration: const InputDecoration(hintText: 'Folder name'),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, nameController.text),
            child: const Text('Create'),
          ),
        ],
      ),
    );

    if (result != null && result.isNotEmpty) {
      if (!context.mounted) return;
      final folderPrefix = _currentFolder.isEmpty ? '' : '$_currentFolder/';
      final folderPath = '$folderPrefix$result';
      // Create a .folder marker so the folder appears in subFolders().
      // The marker is filtered from the notes list display but keeps
      // the folder visible in the hierarchy.
      final notesState = context.read<NotesState>();
      await notesState.createNote('$folderPath/.folder', content: '');
      setState(() {});
    }
  }

  void _showFolderActions(String folderPath) {
    showModalBottomSheet(
      context: context,
      builder: (ctx) => Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          ListTile(
            leading: const Icon(Icons.edit),
            title: const Text('Rename'),
            onTap: () => Navigator.pop(ctx),
          ),
          ListTile(
            leading: const Icon(Icons.delete, color: Colors.red),
            title: const Text('Delete', style: TextStyle(color: Colors.red)),
            onTap: () => Navigator.pop(ctx),
          ),
        ],
      ),
    );
  }

  void _showNoteActions(NotesState notesState, String path) {
    showModalBottomSheet(
      context: context,
      builder: (ctx) => Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          ListTile(
            leading: const Icon(Icons.open_in_new),
            title: const Text('Open'),
            onTap: () {
              Navigator.pop(ctx);
              _openNote(notesState, path);
            },
          ),
          ListTile(
            leading: const Icon(Icons.delete, color: Colors.red),
            title: const Text('Delete', style: TextStyle(color: Colors.red)),
            onTap: () {
              Navigator.pop(ctx);
              notesState.deleteNote(path);
            },
          ),
        ],
      ),
    );
  }
}
