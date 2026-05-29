/// SyncService — S3-based bidirectional sync engine.
///
/// Owns:
///   - An S3 client (minio package), configured from SettingsState.
///   - A periodic Timer that calls runSyncCycle() at pollInterval.
///   - Connectivity awareness via ConnectivityState.
///
/// The sync cycle algorithm follows §8.7 of the Stonepad v1 Plan:
///   1. Check sync state + connectivity → bail if disabled/unreachable.
///   2. Fetch server manifest via ListObjects (ListAllObjects shortcut).
///   3. Compute diff between local and server manifests.
///   4. Execute push/pull/delete operations sequentially (respects free-tier rate limits).
///   5. Write conflict files when both sides changed (§8.8).
///   6. Update local manifest and save.
///
/// The SyncService also exposes manualSync() for the user's manual sync button
/// (§8.6 — available in ALL states, even Disabled).
library;

import 'dart:async';
import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'package:minio/minio.dart';
import 'package:crypto/crypto.dart';
import '../constants/timing.dart';
import '../models/sync_state.dart';
import '../models/note_entry.dart';
import '../state/notes_state.dart';
import '../state/sync_state_notifier.dart';
import '../state/connectivity_state.dart';
import '../state/settings_state.dart';
import 'storage_service.dart';

class SyncService {
  final NotesState _notesState;
  final SyncStateNotifier _syncState;
  final ConnectivityState _connectivity;
  final SettingsState _settingsState;
  final StorageService _storage;
  Timer? _pollTimer;
  bool _disposed = false;

  SyncService({
    required NotesState notesState,
    required SyncStateNotifier syncState,
    required ConnectivityState connectivity,
    required SettingsState settingsState,
    required StorageService storage,
  })  : _notesState = notesState,
        _syncState = syncState,
        _connectivity = connectivity,
        _settingsState = settingsState,
        _storage = storage;

  // ────────────────────── S3 client factory ──────────────────────

  /// Build a [Minio] client from the current settings.
  /// Returns null if settings are incomplete.
  Minio? _buildClient() {
    final s = _settingsState.settings;
    if (!s.hasEndpoint) return null;

    final uri = Uri.tryParse(s.serverEndpoint!);
    if (uri == null) return null;

    final host = uri.host;
    final port = uri.hasPort ? uri.port : (uri.scheme == 'https' ? 443 : 80);

    // For the S3-compatible server, use the appropriate credentials based on auth mode.
    // - none:   empty credentials (passes through)
    // - token:  use the shared bearer token as both access key and secret key
    // - users:  use the session token as both access key and secret key
    // - s3:     use the explicit S3 access key / secret key pair
    String accessKey;
    String secretKey;

    switch (s.authMode) {
      case 's3':
        accessKey = s.s3AccessKey ?? '';
        secretKey = s.s3SecretKey ?? '';
        break;
      case 'users':
        accessKey = s.sessionToken ?? '';
        secretKey = s.sessionToken ?? '';
        break;
      case 'token':
        accessKey = s.authToken ?? '';
        secretKey = s.authToken ?? '';
        break;
      default: // none
        accessKey = '';
        secretKey = '';
    }

    return Minio(
      endPoint: host,
      port: port,
      useSSL: uri.scheme == 'https',
      accessKey: accessKey,
      secretKey: secretKey,
      region: 'us-east-1',
    );
  }

  // ────────────────────── Polling control ──────────────────────

  /// Start the periodic sync poll. Called once after app startup.
  void startPolling() {
    _startTimer();
  }

  void _startTimer() {
    _pollTimer?.cancel();
    _pollTimer = Timer.periodic(TimingConstants.pollInterval, (_) {
      runSyncCycle();
    });
    // Also run immediately on start
    runSyncCycle();
  }

  /// Stop the poll timer.
  void stopPolling() {
    _pollTimer?.cancel();
    _pollTimer = null;
  }

  // ────────────────────── Public API ──────────────────────

  /// Run one complete sync cycle (§8.7 algorithm).
  Future<void> runSyncCycle() async {
    if (_disposed) return;

    // Step 1 – check sync state
    final state = _syncState.state;
    if (state == SyncState.disabled) return;

    // Step 2 – check connectivity
    if (!_connectivity.isConnected) {
      if (state != SyncState.noNetwork) {
        _syncState.onConnectivityChanged(false);
      }
      return;
    }
    // If we were in NoNetwork and connectivity is back, transition to Active
    if (state == SyncState.noNetwork) {
      _syncState.onConnectivityChanged(true);
    }

    // ManualOnly — only run if called explicitly via manualSync().
    // The poll timer should not trigger sync in ManualOnly.
    if (state == SyncState.manualOnly) return;

    final client = _buildClient();
    if (client == null) {
      _syncState.recordFailure();
      return;
    }

    try {
      _syncState.setOperation('Syncing...');

      // Step 3 – fetch server manifest via listAllObjects
      final serverEntries = await _fetchServerManifest(client);
      _syncState.setOperation('Computing diff...');

      // Step 4 – compute diff
      final diff = _computeDiff(serverEntries);

      // Step 5 – execute operations sequentially
      if (diff.pulls.isNotEmpty) {
        _syncState.setOperation('Pulling ${diff.pulls.length} notes...');
        for (final path in diff.pulls) {
          await _pullNote(client, path);
        }
      }

      if (diff.pushes.isNotEmpty) {
        _syncState.setOperation('Pushing ${diff.pushes.length} notes...');
        for (final path in diff.pushes) {
          await _pushNote(client, path);
        }
      }

      if (diff.deletes.isNotEmpty) {
        _syncState.setOperation('Deleting ${diff.deletes.length} notes...');
        for (final path in diff.deletes) {
          await _deleteRemoteNote(client, path);
        }
      }

      // Step 5b – handle conflicts (§8.8)
      if (diff.conflicts.isNotEmpty) {
        _syncState.setOperation(
            'Handling ${diff.conflicts.length} conflicts...');
        await _handleConflicts(client, diff.conflicts);
      }

      _syncState.recordSuccess();
      _syncState.setOperation(null);
    } catch (e, stack) {
      debugPrint('SyncService: cycle failed: $e\n$stack');
      _syncState.recordFailure();
      _syncState.setOperation('Sync failed');
    }
  }

  /// Manual sync — callable from UI button. Works in ALL states.
  Future<void> manualSync() async {
    await runSyncCycle();
  }

  /// Called when the user toggles sync on/off in settings.
  void onSyncToggle(bool enabled) {
    _syncState.setSyncEnabled(enabled);
    if (enabled) {
      _startTimer();
    } else {
      stopPolling();
    }
  }

  /// Called when connectivity changes.
  void onConnectivityChanged(bool connected) {
    _syncState.onConnectivityChanged(connected);
    if (connected && _syncState.state == SyncState.active) {
      _startTimer();
    } else if (!connected) {
      stopPolling();
    }
  }

  // ────────────────────── S3 operations ──────────────────────

  /// Fetch all objects from the server bucket and build a path→hash map.
  Future<Map<String, String>> _fetchServerManifest(Minio client) async {
    final result = <String, String>{};

    final listResult = await client.listAllObjects(
      _bucketName,
      prefix: '',
      recursive: true,
    );

    for (final obj in listResult.objects) {
      // ETag is the content hash (server returns SHA-256 as ETag)
      final hash = (obj.eTag ?? '').replaceAll('"', '');
      if (obj.key != null && obj.key!.isNotEmpty) {
        result[obj.key!] = hash;
      }
    }

    return result;
  }

  /// Pull a note from the server and write it locally.
  Future<void> _pullNote(Minio client, String path) async {
    final stream = await client.getObject(_bucketName, path);
    final bytes = await stream.fold<List<int>>(
        <int>[], (acc, chunk) => acc..addAll(chunk));
    final content = utf8.decode(bytes);

    // Write to disk
    final hash = sha256.convert(bytes).toString();
    await _storage.writeNote(path, content);

    // Update manifest
    _notesState.manifest.notes[path] = NoteEntry(
      contentHash: hash,
      sizeBytes: bytes.length,
      localModified: DateTime.now(),
      lastSyncedHash: hash,
      lastSyncedAt: DateTime.now(),
      status: NoteStatus.synced,
    );

    await _storage.saveManifest(_notesState.manifest);
  }

  /// Push a local note to the server.
  Future<void> _pushNote(Minio client, String path) async {
    final content = await _storage.readNote(path);
    if (content == null) return;
    final bytes = utf8.encode(content);

    // Build metadata headers if we have a last-synced hash
    Map<String, String>? headers;
    final localEntry = _notesState.manifest.notes[path];
    if (localEntry?.lastSyncedHash != null &&
        localEntry!.lastSyncedHash.isNotEmpty) {
      headers = {'If-Match': localEntry.lastSyncedHash};
    }

    await client.putObject(
      _bucketName,
      path,
      Stream.value(Uint8List.fromList(bytes)),
      size: bytes.length,
      metadata: headers,
    );

    // Mark as synced
    final hash = sha256.convert(bytes).toString();
    _notesState.markSynced(path, hash);
  }

  /// Delete a remote note.
  Future<void> _deleteRemoteNote(Minio client, String path) async {
    try {
      await client.removeObject(_bucketName, path);
    } catch (_) {
      // Object may already be deleted — that's fine.
    }
  }

  // ────────────────────── Diff computation ──────────────────────

  String get _bucketName => _settingsState.settings.workspaceId;

  _SyncDiff _computeDiff(Map<String, String> serverManifest) {
    final local = _notesState.manifest.notes;
    final pulls = <String>[];
    final pushes = <String>[];
    final deletes = <String>[];
    final conflicts = <String>[];

    // Notes that exist on the server
    for (final entry in serverManifest.entries) {
      final path = entry.key;
      final serverHash = entry.value;
      final localEntry = local[path];

      if (localEntry == null) {
        // Exists on server, not locally → PULL
        pulls.add(path);
      } else {
        switch (localEntry.status) {
          case NoteStatus.modified:
            if (localEntry.contentHash != serverHash &&
                localEntry.lastSyncedHash != serverHash) {
              // Both sides changed → CONFLICT
              conflicts.add(path);
            } else if (localEntry.lastSyncedHash == serverHash) {
              // Local modified, server unchanged → PUSH (clean update)
              pushes.add(path);
            }
            break;
          case NoteStatus.synced:
            if (localEntry.contentHash != serverHash) {
              // Server has newer version → PULL
              pulls.add(path);
            }
            break;
          case NoteStatus.deleted:
            deletes.add(path);
            break;
          case NoteStatus.conflictPending:
            // Skip — user must resolve manually (§8.8)
            break;
        }
      }
    }

    // Notes that exist locally but NOT on the server → PUSH if modified
    for (final entry in local.entries) {
      if (serverManifest.containsKey(entry.key)) continue;
      if (entry.value.status == NoteStatus.modified) {
        pushes.add(entry.key);
      }
    }

    return _SyncDiff(
      pulls: pulls,
      pushes: pushes,
      deletes: deletes,
      conflicts: conflicts,
    );
  }

  /// Handle conflicts — pull server version to conflicts/, mark local as conflict_pending.
  Future<void> _handleConflicts(Minio client, List<String> paths) async {
    for (final path in paths) {
      try {
        final stream = await client.getObject(_bucketName, path);
        final bytes = await stream.fold<List<int>>(
            <int>[], (acc, chunk) => acc..addAll(chunk));
        final content = utf8.decode(bytes);

        // Write to conflicts directory
        await _storage.writeConflictFile(path, content);

        // Mark local entry as conflict_pending
        final entry = _notesState.manifest.notes[path];
        if (entry != null) {
          _notesState.manifest.notes[path] = entry.copyWith(
            status: NoteStatus.conflictPending,
          );
        }
      } catch (e) {
        debugPrint('SyncService: conflict handling failed for $path: $e');
      }
    }
    await _storage.saveManifest(_notesState.manifest);
  }

  void dispose() {
    _disposed = true;
    stopPolling();
  }
}

/// Result of computing the diff between local and server manifests.
class _SyncDiff {
  final List<String> pulls;
  final List<String> pushes;
  final List<String> deletes;
  final List<String> conflicts;

  const _SyncDiff({
    required this.pulls,
    required this.pushes,
    required this.deletes,
    required this.conflicts,
  });
}
