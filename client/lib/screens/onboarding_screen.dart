/// Onboarding screen — shown on first launch when settings.json doesn't exist.
/// Offers three paths: Use without sync, Set up sync, Skip for now.
/// See §8.12 of the Stonepad v1 Implementation Plan.
import 'package:flutter/material.dart';
import '../constants/strings.dart';

class OnboardingScreen extends StatelessWidget {
  const OnboardingScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(32),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Icon(Icons.auto_stories, size: 80, color: Colors.blue),
              const SizedBox(height: 24),
              Text(
                StonepadStrings.welcomeTitle,
                style: Theme.of(context).textTheme.headlineMedium,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 16),
              Text(
                StonepadStrings.welcomeMessage,
                style: Theme.of(context).textTheme.bodyLarge,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 48),
              SizedBox(
                width: double.infinity,
                child: FilledButton.icon(
                  icon: const Icon(Icons.wifi_off),
                  label: const Text(StonepadStrings.useWithoutSync),
                  onPressed: () {
                    Navigator.pushReplacementNamed(context, '/notes');
                  },
                ),
              ),
              const SizedBox(height: 12),
              SizedBox(
                width: double.infinity,
                child: FilledButton.tonalIcon(
                  icon: const Icon(Icons.sync),
                  label: const Text('Set up sync'),
                  onPressed: () {
                    Navigator.pushReplacementNamed(context, '/settings');
                  },
                ),
              ),
              const SizedBox(height: 12),
              SizedBox(
                width: double.infinity,
                child: OutlinedButton.icon(
                  icon: const Icon(Icons.login),
                  label: const Text('Sign in with username/password'),
                  onPressed: () {
                    Navigator.pushReplacementNamed(context, '/login');
                  },
                ),
              ),
              const SizedBox(height: 12),
              SizedBox(
                width: double.infinity,
                child: TextButton(
                  onPressed: () {
                    Navigator.pushReplacementNamed(context, '/notes');
                  },
                  child: const Text(StonepadStrings.skipForNow),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
