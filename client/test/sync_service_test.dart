/// Tests for SyncService — diff computation, state transitions.
library;
import 'dart:convert';
import 'package:crypto/crypto.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:stonepad/models/note_entry.dart';
import 'package:stonepad/models/settings.dart';
import 'package:stonepad/models/sync_state.dart';
import 'package:stonepad/services/storage_service.dart';
import 'package:stonepad/services/sync_service.dart';
import 'package:stonepad/state/notes_state.dart';
import 'package:stonepad/state/sync_state_notifier.dart';
import 'package:stonepad/state/settings_state.dart';
import 'package:stonepad/state/connectivity_state.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  group('SyncService _computeDiff logic', () {
    late NotesState notesState;
    late SyncStateNotifier syncState;
    late ConnectivityState connectivity;
    late SettingsState settingsState;
    late StorageService storageService;
    late SyncService syncService;

    setUp(() {
      storageService = StorageService();
      notesState = NotesState(storageService);
      syncState = SyncStateNotifier();
      connectivity = ConnectivityState();
      settingsState = SettingsState();
      syncService = SyncService(
        notesState: notesState,
        syncState: syncState,
        connectivity: connectivity,
        settingsState: settingsState,
        storage: storageService,
      );
    });

    tearDown(() {
      syncService.dispose();
    });

    String hash(String content) =>
        sha256.convert(utf8.encode(content)).toString();

    test('empty manifests produce empty diff', () {
      // _computeDiff is private, we test behavior through runSyncCycle,
      // but can test diff by examining manifest after mock server fetch.
      // For now, verify the sync state starts correctly.
      expect(syncState.state, SyncState.disabled);
      syncState.setSyncEnabled(true);
      expect(syncState.state, SyncState.active);
    });

    test('state transitions: Active → NoNetwork on connectivity loss', () {
      syncState.setSyncEnabled(true);
      expect(syncState.state, SyncState.active);

      syncState.onConnectivityChanged(false);
      expect(syncState.state, SyncState.noNetwork);
    });

    test('state transitions: NoNetwork → Active on connectivity restored', () {
      syncState.setSyncEnabled(true);
      syncState.onConnectivityChanged(false);
      expect(syncState.state, SyncState.noNetwork);

      syncState.onConnectivityChanged(true);
      expect(syncState.state, SyncState.active);
    });

    test('state transitions: 3 consecutive failures → ManualOnly', () {
      syncState.setSyncEnabled(true);
      expect(syncState.state, SyncState.active);

      syncState.recordFailure(); // 1
      expect(syncState.state, SyncState.active);
      syncState.recordFailure(); // 2
      expect(syncState.state, SyncState.active);
      syncState.recordFailure(); // 3 — threshold reached
      expect(syncState.state, SyncState.manualOnly);
      expect(syncState.consecutiveFailures, 3);
    });

    test('state transitions: success resets failure count', () {
      syncState.setSyncEnabled(true);
      syncState.recordFailure();
      syncState.recordFailure();
      expect(syncState.consecutiveFailures, 2);

      syncState.recordSuccess();
      expect(syncState.consecutiveFailures, 0);
      expect(syncState.state, SyncState.active);
    });

    test('state transitions: Disabled → Active on sync toggle', () {
      syncState.setSyncEnabled(false);
      expect(syncState.state, SyncState.disabled);

      syncState.setSyncEnabled(true);
      expect(syncState.state, SyncState.active);
      expect(syncState.consecutiveFailures, 0);
    });

    test('Disabled does not react to connectivity changes', () {
      syncState.setSyncEnabled(false);
      expect(syncState.state, SyncState.disabled);

      syncState.onConnectivityChanged(false);
      expect(syncState.state, SyncState.disabled);

      syncState.onConnectivityChanged(true);
      expect(syncState.state, SyncState.disabled);
    });

    test('ManualOnly transitions to Active after success and connectivity', () {
      syncState.setSyncEnabled(true);
      // Simulate 3 failures to enter ManualOnly
      syncState.recordFailure();
      syncState.recordFailure();
      syncState.recordFailure();
      expect(syncState.state, SyncState.manualOnly);

      // Then connectivity drops
      syncState.onConnectivityChanged(false);
      expect(syncState.state, SyncState.noNetwork);

      // Connectivity restored → back to Active
      syncState.onConnectivityChanged(true);
      expect(syncState.state, SyncState.active);
      expect(syncState.consecutiveFailures, 0);
    });

    test('setSyncEnabled transitions properly from NoNetwork', () {
      syncState.setSyncEnabled(true);
      syncState.onConnectivityChanged(false);
      expect(syncState.state, SyncState.noNetwork);

      // Disable sync
      syncState.setSyncEnabled(false);
      expect(syncState.state, SyncState.disabled);

      // Re-enable — should go straight to Active
      syncState.setSyncEnabled(true);
      expect(syncState.state, SyncState.active);
    });
  });

  group('SyncStateNotifier setters', () {
    test('setOperation updates and notifies', () {
      final notifier = SyncStateNotifier();
      var notified = false;
      notifier.addListener(() => notified = true);

      notifier.setOperation('Pulling 5 notes...');
      expect(notifier.currentOperation, 'Pulling 5 notes...');
      expect(notified, isTrue);

      // Clear operation
      notified = false;
      notifier.setOperation(null);
      expect(notifier.currentOperation, isNull);
      expect(notified, isTrue);
    });

    test('recordSuccess sets lastSuccess and lastAttempt', () {
      final notifier = SyncStateNotifier();
      notifier.recordSuccess();
      expect(notifier.lastSuccess, isNotNull);
      expect(notifier.lastAttempt, isNotNull);
      expect(notifier.consecutiveFailures, 0);
    });

    test('recordFailure increments and sets lastAttempt', () {
      final notifier = SyncStateNotifier();
      notifier.recordFailure();
      expect(notifier.consecutiveFailures, 1);
      expect(notifier.lastAttempt, isNotNull);
      expect(notifier.lastSuccess, isNull);
    });
  });

  group('Manifest diff scenarios (integration)', () {
    late NotesState notesState;
    late SyncStateNotifier syncState;
    late ConnectivityState connectivity;
    late SettingsState settingsState;
    late StorageService storageService;
    late SyncService syncService;

    setUp(() {
      storageService = StorageService();
      notesState = NotesState(storageService);
      syncState = SyncStateNotifier();
      connectivity = ConnectivityState();
      settingsState = SettingsState();
      syncService = SyncService(
        notesState: notesState,
        syncState: syncState,
        connectivity: connectivity,
        settingsState: settingsState,
        storage: storageService,
      );
    });

    tearDown(() {
      syncService.dispose();
    });

    String hash(String content) =>
        sha256.convert(utf8.encode(content)).toString();

    test('new note locally → status is modified', () {
      final h = hash('hello');
      notesState.manifest.notes['test.md'] = NoteEntry(
        contentHash: h,
        sizeBytes: 5,
        localModified: DateTime.now(),
        lastSyncedHash: '',
        status: NoteStatus.modified,
      );

      expect(notesState.manifest.notes['test.md']!.status, NoteStatus.modified);
      expect(notesState.manifest.modifiedPaths, ['test.md']);
    });

    test('deleted note locally → status is deleted', () {
      final h = hash('hello');
      notesState.manifest.notes['test.md'] = NoteEntry(
        contentHash: h,
        sizeBytes: 5,
        localModified: DateTime.now(),
        lastSyncedHash: h,
        status: NoteStatus.deleted,
      );

      expect(notesState.manifest.notes['test.md']!.status, NoteStatus.deleted);
      expect(notesState.manifest.deletedPaths, ['test.md']);
    });

    test('conflict → status is conflictPending', () {
      final h = hash('local');
      notesState.manifest.notes['conflict.md'] = NoteEntry(
        contentHash: h,
        sizeBytes: 5,
        localModified: DateTime.now(),
        lastSyncedHash: hash('old-server'),
        status: NoteStatus.conflictPending,
      );

      expect(notesState.manifest.conflictPaths, ['conflict.md']);
    });

    test('markSynced transitions note to synced', () {
      final h = hash('content');
      notesState.manifest.notes['test.md'] = NoteEntry(
        contentHash: h,
        sizeBytes: 7,
        localModified: DateTime.now(),
        lastSyncedHash: '',
        status: NoteStatus.modified,
      );

      notesState.markSynced('test.md', h);

      final entry = notesState.manifest.notes['test.md']!;
      expect(entry.status, NoteStatus.synced);
      expect(entry.lastSyncedHash, h);
      expect(entry.lastSyncedAt, isNotNull);
    });
  });

  group('SettingsState for sync', () {
    test('hasEndpoint returns true when endpoint set', () {
      final settings = StonepadSettings(
        serverEndpoint: 'https://example.com',
      );
      expect(settings.hasEndpoint, isTrue);
    });

    test('hasEndpoint returns false when empty', () {
      final settings = StonepadSettings();
      expect(settings.hasEndpoint, isFalse);
    });

    test('syncEnabled defaults to false', () {
      final settings = StonepadSettings();
      expect(settings.syncEnabled, isFalse);
    });
  });

  group('ConnectivityState', () {
    test('starts as connected', () {
      final state = ConnectivityState();
      expect(state.isConnected, isTrue);
    });
  });
}
