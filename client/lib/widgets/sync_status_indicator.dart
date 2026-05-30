/// Sync status indicator — small icon in the app bar showing sync state machine state.
/// See §8.6 of the Stonepad v1 Implementation Plan.
library;
import 'package:flutter/material.dart';
import '../models/sync_state.dart';

class SyncStatusIndicator extends StatelessWidget {
  final SyncState state;
  final double size;

  const SyncStatusIndicator({
    super.key,
    required this.state,
    this.size = 18,
  });

  @override
  Widget build(BuildContext context) {
    switch (state) {
      case SyncState.active:
        return Icon(Icons.cloud_done, size: size, color: Colors.green);
      case SyncState.manualOnly:
        return Icon(Icons.cloud_off, size: size, color: Colors.orange);
      case SyncState.noNetwork:
        return Icon(Icons.cloud_off, size: size, color: Colors.grey);
      case SyncState.disabled:
        return Icon(Icons.cloud_off, size: size, color: Colors.grey);
    }
  }
}
