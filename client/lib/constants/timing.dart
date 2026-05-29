/// All sync/debounce timing constants in one place for easy tuning.
///
/// No other file in the client codebase may declare a timing duration.
/// All timing references must use TimingConstants.X.
/// See §8.2 and §14.1 of the Stonepad v1 Implementation Plan.
class TimingConstants {
  TimingConstants._();

  /// How often the app polls the server for changes while foregrounded.
  static const Duration pollInterval = Duration(seconds: 7);

  /// How long to wait after the last edit before saving to local disk.
  /// Default 7s per plan — adjustable for testing.
  /// EDIT THIS VALUE to experiment with different debounce durations.
  static const Duration editDebounce = Duration(seconds: 7);

  /// Maximum time to wait for a sync network request before giving up.
  static const Duration syncTimeout = Duration(seconds: 2);

  /// How long the manual sync button stays disabled after a tap (prevents spam).
  static const Duration manualSyncCooldown = Duration(seconds: 3);

  /// Number of consecutive sync failures before backing off to ManualOnly state.
  static const int failureBackoffThreshold = 3;

  /// How long to wait on app close before giving up on a final sync burst.
  static const Duration closeFinalSyncTimeout = Duration(seconds: 1);

  /// Connectivity check timeout (should be very quick).
  static const Duration connectivityCheckTimeout = Duration(milliseconds: 500);
}
