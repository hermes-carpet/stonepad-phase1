/// Onboarding screen — shown on first launch when settings.json doesn't exist.
/// Offers three paths: Use without sync, Set up sync, Skip for now.
/// See §8.12 of the Stonepad v1 Implementation Plan.
///
/// Per taste skills: clean editorial layout, warm monochrome palette,
/// generous whitespace, max 2 primary CTAs.
import 'package:flutter/material.dart';
import '../constants/strings.dart';

class OnboardingScreen extends StatelessWidget {
  const OnboardingScreen({super.key});

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
              // Logo mark — simple, not Material icon
              Container(
                width: 72,
                height: 72,
                decoration: BoxDecoration(
                  color: theme.colorScheme.onSurface.withValues(alpha: 0.06),
                  borderRadius: BorderRadius.circular(18),
                ),
                child: Icon(
                  Icons.auto_stories,
                  size: 36,
                  color: theme.colorScheme.onSurface.withValues(alpha: 0.7),
                ),
              ),
              const SizedBox(height: 32),
              Text(
                StonepadStrings.welcomeTitle,
                style: theme.textTheme.headlineLarge,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 12),
              Text(
                StonepadStrings.welcomeMessage,
                style: theme.textTheme.bodyMedium,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 56),
              // Primary CTA — Use without sync
              SizedBox(
                width: double.infinity,
                child: FilledButton.icon(
                  icon: const Icon(Icons.arrow_forward, size: 18),
                  label: const Text(StonepadStrings.useWithoutSync),
                  onPressed: () {
                    Navigator.pushReplacementNamed(context, '/notes');
                  },
                ),
              ),
              const SizedBox(height: 16),
              // Secondary — Set up sync
              SizedBox(
                width: double.infinity,
                child: OutlinedButton.icon(
                  icon: const Icon(Icons.sync, size: 18),
                  label: const Text('Set up sync'),
                  onPressed: () {
                    Navigator.pushReplacementNamed(context, '/settings');
                  },
                ),
              ),
              const SizedBox(height: 20),
              // Tertiary — Skip
              TextButton(
                onPressed: () {
                  Navigator.pushReplacementNamed(context, '/notes');
                },
                child: Text(
                  StonepadStrings.skipForNow,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.onSurface.withValues(alpha: 0.4),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
