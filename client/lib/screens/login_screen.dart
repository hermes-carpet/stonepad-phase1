/// Login screen for users auth mode.
/// Collects endpoint URL, username, and password, then attempts login
/// via the server's POST /api/v1/auth/login endpoint.
/// On success, stores the session token and navigates to the main app.
library;

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../services/api_client.dart';
import '../state/settings_state.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _endpointController = TextEditingController();
  final _usernameController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _loading = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    // Pre-fill from existing settings
    final settings = context.read<SettingsState>().settings;
    _endpointController.text = settings.serverEndpoint ?? '';
  }

  @override
  void dispose() {
    _endpointController.dispose();
    _usernameController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Sign In')),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.lock_outline, size: 64, color: Colors.blue),
              const SizedBox(height: 16),
              Text(
                'Sign in to your Stonepad server',
                style: Theme.of(context).textTheme.titleLarge,
              ),
              const SizedBox(height: 24),

              // Endpoint
              TextField(
                controller: _endpointController,
                decoration: const InputDecoration(
                  labelText: 'Server URL',
                  hintText: 'https://stonepad.example.com',
                  prefixIcon: Icon(Icons.link),
                  border: OutlineInputBorder(),
                ),
                keyboardType: TextInputType.url,
                textInputAction: TextInputAction.next,
              ),
              const SizedBox(height: 16),

              // Username
              TextField(
                controller: _usernameController,
                decoration: const InputDecoration(
                  labelText: 'Username',
                  prefixIcon: Icon(Icons.person),
                  border: OutlineInputBorder(),
                ),
                textInputAction: TextInputAction.next,
              ),
              const SizedBox(height: 16),

              // Password
              TextField(
                controller: _passwordController,
                decoration: const InputDecoration(
                  labelText: 'Password',
                  prefixIcon: Icon(Icons.key),
                  border: OutlineInputBorder(),
                ),
                obscureText: true,
                textInputAction: TextInputAction.done,
                onSubmitted: (_) => _doLogin(),
              ),
              const SizedBox(height: 24),

              // Error
              if (_error != null)
                Padding(
                  padding: const EdgeInsets.only(bottom: 16),
                  child: Text(
                    _error!,
                    style: const TextStyle(color: Colors.red),
                    textAlign: TextAlign.center,
                  ),
                ),

              // Login button
              SizedBox(
                width: double.infinity,
                height: 48,
                child: FilledButton(
                  onPressed: _loading ? null : _doLogin,
                  child: _loading
                      ? const SizedBox(
                          width: 20,
                          height: 20,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Text('Sign In'),
                ),
              ),
              const SizedBox(height: 12),

              // Skip
              TextButton(
                onPressed: _loading
                    ? null
                    : () => Navigator.pushReplacementNamed(context, '/notes'),
                child: const Text('Skip for now'),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _doLogin() async {
    final endpoint = _endpointController.text.trim();
    final username = _usernameController.text.trim();
    final password = _passwordController.text;

    if (endpoint.isEmpty) {
      setState(() => _error = 'Server URL is required');
      return;
    }
    if (username.isEmpty) {
      setState(() => _error = 'Username is required');
      return;
    }
    if (password.isEmpty) {
      setState(() => _error = 'Password is required');
      return;
    }

    // Normalize the URL
    String url = endpoint;
    if (!url.startsWith('http://') && !url.startsWith('https://')) {
      url = 'https://$url';
    }
    if (url.endsWith('/')) {
      url = url.substring(0, url.length - 1);
    }

    setState(() {
      _loading = true;
      _error = null;
    });

    final client = ApiClient(baseUrl: url);
    final result = await client.login(username, password);
    client.dispose();

    if (!mounted) return;

    if (result.success) {
      final settingsState = context.read<SettingsState>();
      await settingsState.setServerEndpoint(url);
      await settingsState.setAuthMode('users');
      await settingsState.setSessionToken(result.token);
      await settingsState.setSyncEnabled(true);

      if (mounted) {
        Navigator.pushReplacementNamed(context, '/notes');
      }
    } else {
      setState(() {
        _loading = false;
        _error = result.error;
      });
    }
  }
}
