/// Editor toolbar — markdown formatting toolbar for the note editor.
/// Inserts markdown syntax into the text controller at the cursor position.
/// See §8.9 of the Stonepad v1 Implementation Plan.
library;
import 'package:flutter/material.dart';

class EditorToolbar extends StatefulWidget {
  final TextEditingController controller;
  final VoidCallback onChanged;

  const EditorToolbar({
    super.key,
    required this.controller,
    required this.onChanged,
  });

  @override
  State<EditorToolbar> createState() => _EditorToolbarState();
}

class _EditorToolbarState extends State<EditorToolbar> {
  TextSelection _capturedSelection = const TextSelection.collapsed(offset: -1);

  static const _formatItems = <_FormatItem>[
    _FormatItem('bold', 'Bold (**text**)', Icons.format_bold),
    _FormatItem('italic', 'Italic (*text*)', Icons.format_italic),
    _FormatItem('strikethrough', 'Strikethrough (~~text~~)', Icons.format_strikethrough),
    _FormatItem('code', 'Inline code (`text`)', Icons.code),
    _FormatItem('h1', 'Heading 1 (# )', Icons.title),
    _FormatItem('h2', 'Heading 2 (## )', Icons.text_fields),
    _FormatItem('h3', 'Heading 3 (### )', Icons.subtitles),
    _FormatItem('ul', 'Bullet list (- )', Icons.format_list_bulleted),
    _FormatItem('ol', 'Numbered list (1. )', Icons.format_list_numbered),
    _FormatItem('task', 'Task list (- [ ] )', Icons.checklist),
    _FormatItem('quote', 'Blockquote (> )', Icons.format_quote),
    _FormatItem('codeblock', 'Code block (```)', Icons.terminal),
    _FormatItem('link', 'Link ([text](url))', Icons.link),
    _FormatItem('table', 'Table', Icons.table_chart),
    _FormatItem('hr', 'Horizontal rule (---)', Icons.horizontal_rule),
  ];

  @override
  Widget build(BuildContext context) {
    return PopupMenuButton<String>(
      icon: const Icon(Icons.format_bold),
      onSelected: (value) => _insertFormatting(value),
      onOpened: () {
        // Capture selection before the popup steals focus
        _capturedSelection = widget.controller.selection;
      },
      itemBuilder: (context) => _formatItems
          .map((item) => PopupMenuItem(
                value: item.type,
                child: Text(item.label),
              ))
          .toList(),
    );
  }

  void _insertFormatting(String type) {
    final text = widget.controller.text;
    final selection = _capturedSelection.isValid && _capturedSelection.start >= 0
        ? _capturedSelection
        : TextSelection.collapsed(offset: text.length);
    final start = selection.start;
    final end = selection.end;
    final selected = text.substring(start, end);

    String replacement;
    int cursorOffset = 0;

    switch (type) {
      case 'bold':
        replacement = '**${selected.isEmpty ? 'text' : selected}**';
        cursorOffset = selected.isEmpty ? 2 : 0;
      case 'italic':
        replacement = '*${selected.isEmpty ? 'text' : selected}*';
        cursorOffset = selected.isEmpty ? 1 : 0;
      case 'strikethrough':
        replacement = '~~${selected.isEmpty ? 'text' : selected}~~';
        cursorOffset = selected.isEmpty ? 2 : 0;
      case 'code':
        replacement = '`${selected.isEmpty ? 'code' : selected}`';
        cursorOffset = selected.isEmpty ? 1 : 0;
      case 'h1':
        replacement = '# ${selected.isEmpty ? 'Heading 1' : selected}';
        cursorOffset = replacement.length;
      case 'h2':
        replacement = '## ${selected.isEmpty ? 'Heading 2' : selected}';
        cursorOffset = replacement.length;
      case 'h3':
        replacement = '### ${selected.isEmpty ? 'Heading 3' : selected}';
        cursorOffset = replacement.length;
      case 'ul':
        replacement = '- ${selected.isEmpty ? 'List item' : selected}';
        cursorOffset = replacement.length;
      case 'ol':
        replacement = '1. ${selected.isEmpty ? 'List item' : selected}';
        cursorOffset = replacement.length;
      case 'task':
        replacement = '- [ ] ${selected.isEmpty ? 'Task' : selected}';
        cursorOffset = replacement.length;
      case 'quote':
        replacement = '> ${selected.isEmpty ? 'Quote' : selected}';
        cursorOffset = replacement.length;
      case 'codeblock':
        replacement = '```\n${selected.isEmpty ? 'code' : selected}\n```';
        cursorOffset = selected.isEmpty ? 4 : 0;
      case 'link':
        replacement = '[${selected.isEmpty ? 'link text' : selected}](url)';
        cursorOffset = selected.isEmpty ? 1 : 0;
      case 'table':
        replacement = '\n| Col 1 | Col 2 | Col 3 |\n| --- | --- | --- |\n| A | B | C |\n';
        cursorOffset = 1;
      case 'hr':
        replacement = '\n---\n';
        cursorOffset = 0;
      default:
        replacement = selected;
    }

    final newText = text.substring(0, start) + replacement + text.substring(end);
    final newPos = start + replacement.length - (selected.isNotEmpty ? 0 : cursorOffset);
    widget.controller.text = newText;
    widget.controller.selection = TextSelection.collapsed(
      offset: newPos.clamp(0, widget.controller.text.length),
    );
    widget.onChanged();
  }
}

class _FormatItem {
  final String type;
  final String label;
  final IconData icon;
  const _FormatItem(this.type, this.label, this.icon);
}
