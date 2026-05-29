/// Vault setup screen — shown on Android first launch when no folder has been
/// picked yet. Uses the Storage Access Framework (SAF) via file_picker to let
/// the user choose a public folder for their notes.
///
/// On iOS and desktop, this screen is never shown — the vault path is set
/// automatically.
library;

import 'dart:io';
import 'package:flutter/material.dart';
import '../services/vault_manager.dart';

class VaultSetupScreen extends StatefulWidget {
  /// Called after the user successfully picks a folder.
  final VoidCallback onVaultReady;

  const VaultSetupScreen({super.key, required this.onVaultReady});

  @override
  State<VaultSetupScreen> createState() => _VaultSetupScreenState();
}

class _VaultSetupScreenState extends State<VaultSetupScreen> {
  bool _picking = false;
  String? _error;

  Future<void> _pickFolder() async {
    setState(() {
      _picking = true;
      _error = null;
    });

    try {
      final path = await VaultManager.pickAndroidFolder();
      if (!mounted) return;

      if (path != null && path.isNotEmpty) {
        // Verify the picked folder is writable
        try {
          final testFile = File('$path/.stonepad_test');
          await testFile.writeAsString('ok');
          await testFile.delete();
        } catch (e) {
          setState(() {
            _error = 'The selected folder is not writable. '
                'Please choose a different folder.';
            _picking = false;
          });
          return;
        }

        widget.onVaultReady();
      } else {
        setState(() {
          _error = null;
          _picking = false;
        });
      }
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = 'Could not access the selected folder. '
            'Please try again or choose a different folder.';
        _picking = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 48),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              // Icon
              Container(
                width: 72,
                height: 72,
                decoration: BoxDecoration(
                  color: theme.colorScheme.onSurface.withValues(alpha: 0.06),
                  borderRadius: BorderRadius.circular(18),
                ),
                child: Icon(
                  Icons.folder_open,
                  size: 36,
                  color: theme.colorScheme.onSurface.withValues(alpha: 0.7),
                ),
              ),
              const SizedBox(height: 32),
              Text(
                'Choose a Notes Folder',
                style: theme.textTheme.headlineLarge,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 12),
              Text(
                'Stonepad stores your notes as plain Markdown files. '
                'Pick a folder on your device where they will live — '
                'you can access them from any file manager.',
                style: theme.textTheme.bodyMedium,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 12),
              Text(
                'Tip: create a new folder like "My Notes" in your '
                'Documents directory.',
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.onSurface.withValues(alpha: 0.5),
                  fontStyle: FontStyle.italic,
                ),
                textAlign: TextAlign.center,
              ),
              if (_error != null) ...[
                const SizedBox(height: 16),
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: theme.colorScheme.error.withValues(alpha: 0.1),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Row(
                    children: [
                      Icon(Icons.error_outline,
                          size: 18, color: theme.colorScheme.error),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text(
                          _error!,
                          style: theme.textTheme.bodyMedium?.copyWith(
                            color: theme.colorScheme.error,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
              const SizedBox(height: 48),
              SizedBox(
                width: double.infinity,
                child: FilledButton.icon(
                  icon: _picking
                      ? const SizedBox(
                          width: 18,
                          height: 18,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            color: Colors.white,
                          ),
                        )
                      : const Icon(Icons.create_new_folder, size: 18),
                  label: Text(_picking ? 'Opening picker…' : 'Choose Folder'),
                  onPressed: _picking ? null : _pickFolder,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
