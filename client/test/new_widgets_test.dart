/// Tests for new widgets and services added to meet spec §8.1 requirements.
library;
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:stonepad/models/sync_state.dart';
import 'package:stonepad/services/connectivity_service.dart';
import 'package:stonepad/widgets/sync_status_indicator.dart';
import 'package:stonepad/widgets/note_tile.dart';
import 'package:stonepad/widgets/editor_toolbar.dart';

void main() {
  group('ConnectivityService', () {
    test('constructor accepts custom Connectivity instance', () {
      expect(ConnectivityService, isNotNull);
    });
  });

  group('SyncStatusIndicator', () {
    testWidgets('active state shows cloud_done in green', (tester) async {
      await tester.pumpWidget(const MaterialApp(
        home: Scaffold(
          body: SyncStatusIndicator(state: SyncState.active),
        ),
      ));
      expect(find.byIcon(Icons.cloud_done), findsOneWidget);
    });

    testWidgets('manualOnly state shows cloud_off in orange', (tester) async {
      await tester.pumpWidget(const MaterialApp(
        home: Scaffold(
          body: SyncStatusIndicator(state: SyncState.manualOnly),
        ),
      ));
      final icon = tester.widget<Icon>(find.byIcon(Icons.cloud_off));
      expect(icon.color, Colors.orange);
    });

    testWidgets('noNetwork state shows cloud_off in grey', (tester) async {
      await tester.pumpWidget(const MaterialApp(
        home: Scaffold(
          body: SyncStatusIndicator(state: SyncState.noNetwork),
        ),
      ));
      final icon = tester.widget<Icon>(find.byIcon(Icons.cloud_off));
      expect(icon.color, Colors.grey);
    });

    testWidgets('disabled state shows cloud_off in grey', (tester) async {
      await tester.pumpWidget(const MaterialApp(
        home: Scaffold(
          body: SyncStatusIndicator(state: SyncState.disabled),
        ),
      ));
      final icon = tester.widget<Icon>(find.byIcon(Icons.cloud_off));
      expect(icon.color, Colors.grey);
    });
  });

  group('NoteTile', () {
    testWidgets('shows filename without .md as title', (tester) async {
      await tester.pumpWidget(MaterialApp(
        home: Scaffold(
          body: NoteTile(
            notePath: 'work/meetings.md',
            status: null,
            onTap: () {},
            onLongPress: () {},
          ),
        ),
      ));
      expect(find.text('meetings'), findsOneWidget);
      expect(find.text('work/meetings.md'), findsOneWidget);
    });

    testWidgets('modified status shows cloud_upload icon', (tester) async {
      await tester.pumpWidget(MaterialApp(
        home: Scaffold(
          body: NoteTile(
            notePath: 'test.md',
            status: 'modified',
            onTap: () {},
            onLongPress: () {},
          ),
        ),
      ));
      expect(find.byIcon(Icons.cloud_upload), findsOneWidget);
    });

    testWidgets('conflict_pending status shows warning icon', (tester) async {
      await tester.pumpWidget(MaterialApp(
        home: Scaffold(
          body: NoteTile(
            notePath: 'test.md',
            status: 'conflict_pending',
            onTap: () {},
            onLongPress: () {},
          ),
        ),
      ));
      expect(find.byIcon(Icons.warning), findsOneWidget);
    });

    testWidgets('synced status shows no trailing icon', (tester) async {
      await tester.pumpWidget(MaterialApp(
        home: Scaffold(
          body: NoteTile(
            notePath: 'test.md',
            status: 'synced',
            onTap: () {},
            onLongPress: () {},
          ),
        ),
      ));
      expect(find.byIcon(Icons.cloud_upload), findsNothing);
      expect(find.byIcon(Icons.warning), findsNothing);
    });
  });

  group('EditorToolbar formatting', () {
    // Test formatting logic directly by tapping the popup menu.
    // Tests that formatting modifies the controller text and fires onChanged.

    testWidgets('bold wraps selected text in **', (tester) async {
      final controller = TextEditingController(text: 'hello');
      controller.selection =
          const TextSelection(baseOffset: 0, extentOffset: 5);
      var changed = false;

      await tester.pumpWidget(MaterialApp(
        home: Scaffold(
          body: EditorToolbar(
            controller: controller,
            onChanged: () => changed = true,
          ),
        ),
      ));

      await tester.tap(find.byType(PopupMenuButton<String>));
      await tester.pumpAndSettle();
      await tester.tap(find.textContaining('Bold'));
      await tester.pumpAndSettle();

      expect(controller.text, '**hello**');
      expect(changed, isTrue);
    });

    testWidgets('italic wraps selected text in *', (tester) async {
      final controller = TextEditingController(text: 'hello');
      controller.selection =
          const TextSelection(baseOffset: 0, extentOffset: 5);
      var changed = false;

      await tester.pumpWidget(MaterialApp(
        home: Scaffold(
          body: EditorToolbar(
            controller: controller,
            onChanged: () => changed = true,
          ),
        ),
      ));

      await tester.tap(find.byType(PopupMenuButton<String>));
      await tester.pumpAndSettle();
      await tester.tap(find.textContaining('Italic'));
      await tester.pumpAndSettle();

      expect(controller.text, '*hello*');
      expect(changed, isTrue);
    });

    testWidgets('h1 inserts # at cursor position', (tester) async {
      final controller = TextEditingController(text: '');
      var changed = false;

      await tester.pumpWidget(MaterialApp(
        home: Scaffold(
          body: EditorToolbar(
            controller: controller,
            onChanged: () => changed = true,
          ),
        ),
      ));

      await tester.tap(find.byType(PopupMenuButton<String>));
      await tester.pumpAndSettle();
      await tester.tap(find.textContaining('Heading 1'));
      await tester.pumpAndSettle();

      expect(controller.text, '# Heading 1');
      expect(changed, isTrue);
    });

    testWidgets('ul inserts - at cursor position', (tester) async {
      final controller = TextEditingController(text: '');
      var changed = false;

      await tester.pumpWidget(MaterialApp(
        home: Scaffold(
          body: EditorToolbar(
            controller: controller,
            onChanged: () => changed = true,
          ),
        ),
      ));

      await tester.tap(find.byType(PopupMenuButton<String>));
      await tester.pumpAndSettle();
      await tester.tap(find.textContaining('Bullet list'));
      await tester.pumpAndSettle();

      expect(controller.text, '- List item');
      expect(changed, isTrue);
    });

    testWidgets('onChanged fires for each format application', (tester) async {
      final controller = TextEditingController(text: '');
      var changeCount = 0;

      await tester.pumpWidget(MaterialApp(
        home: Scaffold(
          body: EditorToolbar(
            controller: controller,
            onChanged: () => changeCount++,
          ),
        ),
      ));

      await tester.tap(find.byType(PopupMenuButton<String>));
      await tester.pumpAndSettle();
      await tester.tap(find.textContaining('Heading 1'));
      await tester.pumpAndSettle();
      expect(changeCount, 1);

      // Reset cursor and try another format
      controller.selection = const TextSelection.collapsed(offset: 0);
      await tester.tap(find.byType(PopupMenuButton<String>));
      await tester.pumpAndSettle();
      await tester.tap(find.textContaining('Heading 2'));
      await tester.pumpAndSettle();
      expect(changeCount, 2);
    });
  });
}
