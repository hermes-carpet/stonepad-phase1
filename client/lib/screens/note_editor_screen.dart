/// Note Editor Screen — markdown editor with live preview.
/// Uses a split-pane approach: edit on left, rendered preview on right.
/// AppFlowy Editor is the intended WYSIWYG editor but is currently incompatible
/// with Flutter 3.44.0 (§8.9 allows flutter_markdown as optional fallback).
/// 
/// Features:
///   - Debounced auto-save (7s per TimingConstants)
///   - EditorToolbar (bold, italic, heading, list, code, link, table, hr)
///   - Markdown preview panel
///   - Unsaved changes indicator
library;
import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import '../state/notes_state.dart';
import '../constants/timing.dart';
import '../widgets/editor_toolbar.dart';

class NoteEditorScreen extends StatefulWidget {
  final String notePath;

  const NoteEditorScreen({super.key, required this.notePath});

  @override
  State<NoteEditorScreen> createState() => _NoteEditorScreenState();
}

class _NoteEditorScreenState extends State<NoteEditorScreen> {
  late TextEditingController _controller;
  Timer? _debounceTimer;
  bool _hasChanges = false;
  bool _showPreview = false;
  final ScrollController _editScroll = ScrollController();
  final ScrollController _previewScroll = ScrollController();

  @override
  void initState() {
    super.initState();
    final notesState = context.read<NotesState>();
    _controller = TextEditingController(text: notesState.currentNoteContent ?? '');
    _controller.addListener(_onTextChanged);
  }

  void _onTextChanged() {
    _hasChanges = true;
    _debounceTimer?.cancel();
    _debounceTimer = Timer(TimingConstants.editDebounce, _saveNow);
  }

  Future<void> _saveNow() async {
    if (!_hasChanges) return;
    final notesState = context.read<NotesState>();
    notesState.updateNoteContent(_controller.text);
    await notesState.saveCurrentNote();
    _hasChanges = false;
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    _debounceTimer?.cancel();
    _saveNow();
    _controller.dispose();
    _editScroll.dispose();
    _previewScroll.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final filename = widget.notePath.split('/').last;

    return Scaffold(
      appBar: AppBar(
        title: Text(filename.replaceAll('.md', '')),
        actions: [
          if (_hasChanges)
            const Padding(
              padding: EdgeInsets.only(right: 8),
              child: Icon(Icons.circle, size: 10, color: Colors.orange),
            ),
          IconButton(
            icon: Icon(_showPreview ? Icons.edit : Icons.visibility),
            tooltip: _showPreview ? 'Edit' : 'Preview',
            onPressed: () => setState(() => _showPreview = !_showPreview),
          ),
          EditorToolbar(
            controller: _controller,
            onChanged: () {
              _hasChanges = true;
              _debounceTimer?.cancel();
              _debounceTimer = Timer(TimingConstants.editDebounce, _saveNow);
            },
          ),
          IconButton(
            icon: const Icon(Icons.check),
            tooltip: 'Save now',
            onPressed: _saveNow,
          ),
        ],
      ),
      body: _showPreview ? _buildPreview() : _buildEditor(),
    );
  }

  Widget _buildEditor() {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: TextField(
        controller: _controller,
        scrollController: _editScroll,
        maxLines: null,
        expands: true,
        textAlignVertical: TextAlignVertical.top,
        style: TextStyle(
          fontSize: 16,
          height: 1.7,
          color: Theme.of(context).colorScheme.onSurface,
        ),
        decoration: InputDecoration(
          border: InputBorder.none,
          hintText: 'Start writing markdown...\n\n# Heading 1\n## Heading 2\n**bold** *italic*\n- list item\n[link](url)',
          hintStyle: TextStyle(
            color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.25),
          ),
        ),
      ),
    );
  }

  Widget _buildPreview() {
    return Markdown(
      data: _controller.text,
      selectable: true,
      padding: const EdgeInsets.all(16),
      styleSheet: MarkdownStyleSheet(
        h1: TextStyle(
          fontSize: 28,
          fontWeight: FontWeight.bold,
          color: Theme.of(context).colorScheme.onSurface,
        ),
        h2: TextStyle(
          fontSize: 22,
          fontWeight: FontWeight.bold,
          color: Theme.of(context).colorScheme.onSurface,
        ),
        p: TextStyle(
          fontSize: 16,
          height: 1.6,
          color: Theme.of(context).colorScheme.onSurface,
        ),
        code: TextStyle(
          backgroundColor: Theme.of(context).colorScheme.surfaceContainerHighest,
          fontFamily: 'monospace',
        ),
        codeblockDecoration: BoxDecoration(
          color: Theme.of(context).colorScheme.surfaceContainerHighest,
          borderRadius: BorderRadius.circular(8),
        ),
      ),
    );
  }
}
