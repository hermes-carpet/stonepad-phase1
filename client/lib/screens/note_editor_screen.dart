/// Note Editor Screen — markdown editor with live preview.
/// Uses a split-pane approach: edit on left, rendered preview on right.
/// AppFlowy Editor is the intended WYSIWYG editor but is currently incompatible
/// with Flutter 3.44.0 (§8.9 allows flutter_markdown as optional fallback).
/// 
/// Features:
///   - Debounced auto-save (7s per TimingConstants)
///   - Format toolbar (bold, italic, heading, list, code, link, table, hr)
///   - Markdown preview panel
///   - Unsaved changes indicator
library;
import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import '../state/notes_state.dart';
import '../constants/timing.dart';

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
          _buildFormatMenu(),
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

  Widget _buildFormatMenu() {
    return PopupMenuButton<String>(
      icon: const Icon(Icons.format_bold),
      onSelected: (value) => _insertFormatting(value),
      itemBuilder: (context) => [
        const PopupMenuItem(value: 'bold', child: Text('Bold (**text**)')),
        const PopupMenuItem(value: 'italic', child: Text('Italic (*text*)')),
        const PopupMenuItem(value: 'strikethrough', child: Text('Strikethrough (~~text~~)')),
        const PopupMenuItem(value: 'code', child: Text('Inline code (`text`)')),
        const PopupMenuDivider(),
        const PopupMenuItem(value: 'h1', child: Text('Heading 1 (# )')),
        const PopupMenuItem(value: 'h2', child: Text('Heading 2 (## )')),
        const PopupMenuItem(value: 'h3', child: Text('Heading 3 (### )')),
        const PopupMenuDivider(),
        const PopupMenuItem(value: 'ul', child: Text('Bullet list (- )')),
        const PopupMenuItem(value: 'ol', child: Text('Numbered list (1. )')),
        const PopupMenuItem(value: 'task', child: Text('Task list (- [ ] )')),
        const PopupMenuItem(value: 'quote', child: Text('Blockquote (> )')),
        const PopupMenuItem(value: 'codeblock', child: Text('Code block (```)')),
        const PopupMenuDivider(),
        const PopupMenuItem(value: 'link', child: Text('Link ([text](url))')),
        const PopupMenuItem(value: 'table', child: Text('Table')),
        const PopupMenuItem(value: 'hr', child: Text('Horizontal rule (---)')),
      ],
    );
  }

  void _insertFormatting(String type) {
    final text = _controller.text;
    final start = _controller.selection.start;
    final end = _controller.selection.end;
    final selected = text.substring(start, end);

    String replacement;
    int cursorOffset = 0;

    switch (type) {
      case 'bold':
        replacement = '**${selected.isEmpty ? 'text' : selected}**';
        cursorOffset = selected.isEmpty ? 2 : 0;
        break;
      case 'italic':
        replacement = '*${selected.isEmpty ? 'text' : selected}*';
        cursorOffset = selected.isEmpty ? 1 : 0;
        break;
      case 'strikethrough':
        replacement = '~~${selected.isEmpty ? 'text' : selected}~~';
        cursorOffset = selected.isEmpty ? 2 : 0;
        break;
      case 'code':
        replacement = '`${selected.isEmpty ? 'code' : selected}`';
        cursorOffset = selected.isEmpty ? 1 : 0;
        break;
      case 'h1':
        replacement = '# ${selected.isEmpty ? 'Heading 1' : selected}';
        cursorOffset = replacement.length;
        break;
      case 'h2':
        replacement = '## ${selected.isEmpty ? 'Heading 2' : selected}';
        cursorOffset = replacement.length;
        break;
      case 'h3':
        replacement = '### ${selected.isEmpty ? 'Heading 3' : selected}';
        cursorOffset = replacement.length;
        break;
      case 'ul':
        replacement = '- ${selected.isEmpty ? 'List item' : selected}';
        cursorOffset = replacement.length;
        break;
      case 'ol':
        replacement = '1. ${selected.isEmpty ? 'List item' : selected}';
        cursorOffset = replacement.length;
        break;
      case 'task':
        replacement = '- [ ] ${selected.isEmpty ? 'Task' : selected}';
        cursorOffset = replacement.length;
        break;
      case 'quote':
        replacement = '> ${selected.isEmpty ? 'Quote' : selected}';
        cursorOffset = replacement.length;
        break;
      case 'codeblock':
        replacement = '```\n${selected.isEmpty ? 'code' : selected}\n```';
        cursorOffset = selected.isEmpty ? 4 : 0;
        break;
      case 'link':
        replacement = '[${selected.isEmpty ? 'link text' : selected}](url)';
        cursorOffset = selected.isEmpty ? 1 : 0;
        break;
      case 'table':
        replacement = '\n| Col 1 | Col 2 | Col 3 |\n| --- | --- | --- |\n| A | B | C |\n';
        cursorOffset = 1;
        break;
      case 'hr':
        replacement = '\n---\n';
        cursorOffset = 0;
        break;
      default:
        replacement = selected;
    }

    setState(() {
      _controller.text = text.substring(0, start) + replacement + text.substring(end);
      final newPos = start + replacement.length - (selected.isNotEmpty ? 0 : cursorOffset);
      _controller.selection = TextSelection.collapsed(offset: newPos.clamp(0, _controller.text.length));
    });
    _hasChanges = true;
    _debounceTimer?.cancel();
    _debounceTimer = Timer(TimingConstants.editDebounce, _saveNow);
  }
}
