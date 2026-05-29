/// Sync state machine notifier.
/// Manages transitions between Active, ManualOnly, NoNetwork, and Disabled.
/// See §8.6 for the full state transition diagram.
import 'package:flutter/foundation.dart';
import '../models/sync_state.dart';
import '../constants/timing.dart';

class SyncStateNotifier extends ChangeNotifier {
  SyncState _state = SyncState.disabled;
  int _consecutiveFailures = 0;
  DateTime? _lastAttempt;
  DateTime? _lastSuccess;
  String? _currentOperation;

  SyncState get state => _state;
  int get consecutiveFailures => _consecutiveFailures;
  DateTime? get lastAttempt => _lastAttempt;
  DateTime? get lastSuccess => _lastSuccess;
  String? get currentOperation => _currentOperation;

  /// Transition to a new state.
  void transitionTo(SyncState newState) {
    if (_state == newState) return;
    _state = newState;
    notifyListeners();
  }

  /// Record a successful sync operation.
  void recordSuccess() {
    _consecutiveFailures = 0;
    _lastAttempt = DateTime.now();
    _lastSuccess = DateTime.now();
    notifyListeners();
  }

  /// Record a failed sync operation. Transitions to ManualOnly if threshold reached.
  void recordFailure() {
    _consecutiveFailures++;
    _lastAttempt = DateTime.now();
    if (_state == SyncState.active &&
        _consecutiveFailures >= TimingConstants.failureBackoffThreshold) {
      _state = SyncState.manualOnly;
    }
    notifyListeners();
  }

  /// Set a human-readable description of the current sync operation.
  void setOperation(String? description) {
    _currentOperation = description;
    notifyListeners();
  }

  /// Handle connectivity change.
  void onConnectivityChanged(bool isConnected) {
    if (_state == SyncState.disabled) return;

    if (isConnected) {
      if (_state == SyncState.noNetwork) {
        _state = SyncState.active;
        _consecutiveFailures = 0;
        notifyListeners();
      }
    } else {
      if (_state != SyncState.noNetwork) {
        _state = SyncState.noNetwork;
        notifyListeners();
      }
    }
  }

  /// User toggled sync on/off.
  void setSyncEnabled(bool enabled) {
    if (enabled) {
      _state = SyncState.active;
      _consecutiveFailures = 0;
    } else {
      _state = SyncState.disabled;
    }
    notifyListeners();
  }
}
