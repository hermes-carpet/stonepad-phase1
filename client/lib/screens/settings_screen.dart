/// Settings screen — endpoint configuration, sync toggle, diagnostics.
/// See §8.11 of the Stonepad v1 Implementation Plan.
library;
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../state/settings_state.dart';
import '../state/sync_state_notifier.dart';
import '../state/notes_state.dart';
import '../services/sync_service.dart';
import '../models/settings.dart';
import '../constants/strings.dart';
import '../constants/paths.dart';
import 'login_screen.dart';
import 'package:flutter/services.dart';

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key});

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  final _endpointController = TextEditingController();
  final _tokenController = TextEditingController();
  final _accessKeyController = TextEditingController();
  final _secretKeyController = TextEditingController();
  final _workspaceController = TextEditingController();
  final _relayEndpointController = TextEditingController();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _loadSettings();
    });
  }

  void _loadSettings() {
    final settings = context.read<SettingsState>().settings;
    _endpointController.text = settings.serverEndpoint ?? '';
    _tokenController.text = settings.authToken ?? '';
    _accessKeyController.text = settings.s3AccessKey ?? '';
    _secretKeyController.text = settings.s3SecretKey ?? '';
    _workspaceController.text = settings.workspaceId;
    _relayEndpointController.text = settings.relayEndpoint ?? '';
  }

  @override
  void dispose() {
    _endpointController.dispose();
    _tokenController.dispose();
    _accessKeyController.dispose();
    _secretKeyController.dispose();
    _workspaceController.dispose();
    _relayEndpointController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text(StonepadStrings.settingsTitle)),
      body: Consumer3<SettingsState, SyncStateNotifier, NotesState>(
        builder: (context, settingsState, syncState, notesState, _) {
          final settings = settingsState.settings;

          return ListView(
            padding: const EdgeInsets.all(16),
            children: [
              // --- Notes Location (read-only) ---
              _sectionHeader('Notes Location'),
              FutureBuilder<String>(
                future: StonepadPaths.notesDirectory().then((d) => d.path),
                builder: (_, snapshot) => ListTile(
                  title: const Text('Storage path'),
                  subtitle: Text(snapshot.data ?? '...'),
                  trailing: IconButton(
                    icon: const Icon(Icons.copy),
                    onPressed: () {
                      if (snapshot.data != null) {
                        Clipboard.setData(ClipboardData(text: snapshot.data!));
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(content: Text('Path copied')),
                        );
                      }
                    },
                  ),
                ),
              ),

              const Divider(),

              // --- Server Configuration ---
              _sectionHeader('Server Configuration'),
              _buildTextField(
                controller: _endpointController,
                label: 'Server endpoint URL',
                hint: 'https://stonepad.example.com',
                onChanged: (v) => settingsState.setServerEndpoint(v),
              ),

              // Auth mode
              ListTile(
                title: const Text('Authentication mode'),
                subtitle: Text(settings.authMode == 'token'
                    ? 'Shared Token'
                    : settings.authMode == 's3'
                        ? 'S3 Access Keys'
                        : settings.authMode == 'users'
                            ? 'Username / Password'
                            : 'None'),
                trailing: DropdownButton<String>(
                  value: settings.authMode,
                  items: const [
                    DropdownMenuItem(value: 'n' 'one', child: Text('None')),
                    DropdownMenuItem(value: 'token', child: Text('Token')),
                    DropdownMenuItem(value: 's3', child: Text('S3 Keys')),
                    DropdownMenuItem(value: 'users', child: Text('Login')),
                  ],
                  onChanged: (v) {
                    if (v != null) settingsState.setAuthMode(v);
                  },
                ),
              ),

              if (settings.authMode == 'token')
                _buildTextField(
                  controller: _tokenController,
                  label: 'Shared auth token',
                  obscureText: true,
                  onChanged: (v) => settingsState.setAuthToken(v),
                ),

              if (settings.authMode == 'users')
                _buildUsersModeSection(settingsState, settings),

              if (settings.authMode == 's3') ...[
                _buildTextField(
                  controller: _accessKeyController,
                  label: 'Access Key ID',
                  onChanged: (v) => settingsState.setS3Keys(v, _secretKeyController.text),
                ),
                _buildTextField(
                  controller: _secretKeyController,
                  label: 'Secret Access Key',
                  obscureText: true,
                  onChanged: (v) => settingsState.setS3Keys(_accessKeyController.text, v),
                ),
              ],

              _buildTextField(
                controller: _workspaceController,
                label: 'Workspace ID',
                onChanged: (v) => settingsState.setWorkspaceId(v),
              ),

              const Divider(),

              // --- Sync Toggle ---
              _sectionHeader('Sync'),
              SwitchListTile(
                title: Text(settings.syncEnabled ? 'Online' : 'Offline'),
                subtitle: const Text('Spotify-style offline mode'),
                value: settings.syncEnabled,
                onChanged: (v) {
                  settingsState.setSyncEnabled(v);
                  context.read<SyncService>().onSyncToggle(v);
                  syncState.setSyncEnabled(v);
                },
              ),
              ElevatedButton.icon(
                icon: const Icon(Icons.sync),
                label: const Text('Sync now'),
                onPressed: () {
                  context.read<SyncService>().manualSync();
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('Manual sync triggered')),
                  );
                },
              ),

              const Divider(),

              // --- Relay Configuration ---
              _sectionHeader('Relay configuration (optional)'),
              SwitchListTile(
                title: const Text('Enable relay'),
                value: settings.relayEnabled,
                onChanged: (v) => settingsState.setRelayConfig(enabled: v),
              ),
              if (settings.relayEnabled)
                _buildTextField(
                  controller: _relayEndpointController,
                  label: 'Relay endpoint URL',
                  onChanged: (v) => settingsState.setRelayConfig(endpoint: v),
                ),

              const Divider(),

              // --- Diagnostics ---
              _sectionHeader('Diagnostics'),
              ListTile(
                title: const Text('Sync state'),
                subtitle: Text(syncState.state.name),
              ),
              ListTile(
                title: const Text('Notes count'),
                subtitle: Text('${notesState.allPaths.length}'),
              ),
              ListTile(
                title: const Text('Pending changes'),
                subtitle: Text('${notesState.manifest.modifiedPaths.length}'),
              ),
              ListTile(
                title: const Text('Conflicts'),
                subtitle: Text('${notesState.manifest.conflictPaths.length}'),
              ),
              if (syncState.lastSuccess != null)
                ListTile(
                  title: const Text('Last sync'),
                  subtitle: Text(syncState.lastSuccess!.toLocal().toString()),
                ),
            ],
          );
        },
      ),
    );
  }

  Widget _buildUsersModeSection(SettingsState settingsState, StonepadSettings settings) {
    // If logged in, show session status
    if (settings.sessionToken != null && settings.sessionToken!.isNotEmpty) {
      return Padding(
        padding: const EdgeInsets.only(bottom: 8),
        child: Card(
          child: Padding(
            padding: const EdgeInsets.all(12),
            child: Row(
              children: [
                const Icon(Icons.check_circle, color: Colors.green),
                const SizedBox(width: 8),
                const Expanded(child: Text('Signed in')),
                TextButton(
                  onPressed: () async {
                    await settingsState.setSessionToken(null);
                  },
                  child: const Text('Sign Out'),
                ),
              ],
            ),
          ),
        ),
      );
    }

    // Not logged in — show "Sign In" button
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: OutlinedButton.icon(
        icon: const Icon(Icons.login),
        label: const Text('Sign In to Server'),
        onPressed: () {
          Navigator.push(
            context,
            MaterialPageRoute(builder: (_) => const LoginScreen()),
          );
        },
      ),
    );
  }

  Widget _sectionHeader(String title) {
    return Padding(
      padding: const EdgeInsets.only(top: 24, bottom: 10),
      child: Text(
        title.toUpperCase(),
        style: Theme.of(context).textTheme.labelSmall!.copyWith(
          color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.5),
        ),
      ),
    );
  }

  Widget _buildTextField({
    required TextEditingController controller,
    required String label,
    String? hint,
    bool obscureText = false,
    ValueChanged<String>? onChanged,
  }) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: TextField(
        controller: controller,
        obscureText: obscureText,
        decoration: InputDecoration(
          labelText: label,
          hintText: hint,
          border: const OutlineInputBorder(),
        ),
        onChanged: onChanged,
      ),
    );
  }
}
