/// LifecycleService — triggers save + sync burst on app lifecycle events.
/// Uses AppLifecycleListener. On paused, hidden, or detached,
/// saves the current editor state. See §8.5.
import 'package:flutter/material.dart';
import '../state/notes_state.dart';
import '../state/sync_state_notifier.dart';

class LifecycleService {
  final NotesState notesState;
  final SyncStateNotifier syncState;
  AppLifecycleListener? _listener;

  LifecycleService({
    required this.notesState,
    required this.syncState,
  });

  /// Register lifecycle callbacks. Must be called after widget tree is ready.
  void register() {
    _listener = AppLifecycleListener(
      onStateChange: (state) {
        if (state == AppLifecycleState.paused ||
            state == AppLifecycleState.hidden ||
            state == AppLifecycleState.detached) {
          _onBackground();
        }
      },
    );
  }

  /// Called when the app goes to background.
  /// Saves the current note immediately (no debounce delay).
  Future<void> _onBackground() async {
    // Save immediately — bypass debounce
    if (notesState.currentNoteContent != null) {
      await notesState.saveCurrentNote();
    }

    // Trigger a final sync burst if online
    // (sync service will be wired in Phase 5)
  }

  void dispose() {
    _listener?.dispose();
  }
}
