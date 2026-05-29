/// Sync state enum for the sync state machine.
/// See §8.6 for the full state transition diagram.
enum SyncState {
  /// Default — polling every 7s and pushing/pulling changes.
  active,

  /// Polling stopped after 3 consecutive failures; only manual button works.
  manualOnly,

  /// No connectivity detected; no sync attempts until connectivity restored.
  noNetwork,

  /// User toggled offline; auto-sync disabled. Manual button still available.
  disabled,
}
