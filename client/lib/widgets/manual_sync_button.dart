/// Manual sync button — triggers an immediate sync cycle.
/// Available in all sync states per §8.6.
library;
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../services/sync_service.dart';

class ManualSyncButton extends StatelessWidget {
  const ManualSyncButton({super.key});

  @override
  Widget build(BuildContext context) {
    return IconButton(
      icon: const Icon(Icons.sync),
      tooltip: 'Sync now',
      onPressed: () {
        context.read<SyncService>().manualSync();
      },
    );
  }
}
