/// Note Editor Screen — wraps the AppFlowy editor for WYSIWYG markdown.
/// In v1, we use a simple TextField-based editor since AppFlowy requires
/// complex integration. The full AppFlowy integration comes in Phase 4.
/// See §8.9 of the Stonepad v1 Implementation Plan.
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../state/notes_state.dart';
import '../constants/timing.dart';
import 'dart:async';

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
  }

  @override
  void dispose() {
    _debounceTimer?.cancel();
    _saveNow(); // Save on close
    _controller.dispose();
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
            icon: const Icon(Icons.check),
            tooltip: 'Save now',
            onPressed: _saveNow,
          ),
          _buildFormatMenu(),
        ],
      ),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: TextField(
          controller: _controller,
          maxLines: null,
          expands: true,
          textAlignVertical: TextAlignVertical.top,
          style: const TextStyle(fontSize: 16, height: 1.5),
          decoration: const InputDecoration(
            border: InputBorder.none,
            hintText: 'Start writing...',
          ),
        ),
      ),
    );
  }

  Widget _buildFormatMenu() {
    return PopupMenuButton<String>(
      icon: const Icon(Icons.format_bold),
      onSelected: (value) {
        final text = _controller.text;
        final start = _controller.selection.start;
        final end = _controller.selection.end;
        final selected = text.substring(start, end);

        String replacement;
        switch (value) {
          case 'bold':
            replacement = '**$selected**';
            break;
          case 'italic':
            replacement = '*$selected*';
            break;
          case 'heading':
            replacement = '\n## $selected\n';
            break;
          case 'list':
            replacement = '\n- $selected';
            break;
          case 'code':
            replacement = '`$selected`';
            break;
          default:
            replacement = selected;
        }

        _controller.text = text.substring(0, start) + replacement + text.substring(end);
        _controller.selection = TextSelection.collapsed(
          offset: start + replacement.length,
        );
      },
      itemBuilder: (context) => [
        const PopupMenuItem(value: 'bold', child: Text('Bold (**)')),
        const PopupMenuItem(value: 'italic', child: Text('Italic (*)')),
        const PopupMenuItem(value: 'heading', child: Text('Heading (##)')),
        const PopupMenuItem(value: 'list', child: Text('List (-)')),
        const PopupMenuItem(value: 'code', child: Text('Code (`)')),
      ],
    );
  }
}
