/// Stonepad — Self-hostable markdown notes application.
/// Mobile-first Flutter client with optional sync.
/// See §8 of the Stonepad v1 Implementation Plan.
import 'dart:io';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:window_manager/window_manager.dart';
import 'constants/strings.dart';
import 'services/storage_service.dart';
import 'services/lifecycle_service.dart';
import 'state/notes_state.dart';
import 'state/sync_state_notifier.dart';
import 'state/settings_state.dart';
import 'state/connectivity_state.dart';
import 'screens/notes_list_screen.dart';
import 'screens/settings_screen.dart';
import 'screens/onboarding_screen.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Linux/macOS/Windows desktop: lock to portrait phone size for dev testing.
  // See §8.13 — this exists exclusively for the AI-driven dev feedback loop.
  if (Platform.isLinux || Platform.isWindows || Platform.isMacOS) {
    await windowManager.ensureInitialized();
    const windowOptions = WindowOptions(
      size: Size(412, 915),
      minimumSize: Size(412, 915),
      maximumSize: Size(412, 915),
      center: true,
      title: StonepadStrings.desktopTitle,
    );
    await windowManager.waitUntilReadyToShow(windowOptions, () async {
      await windowManager.show();
      await windowManager.focus();
    });
  }

  // Initialize services
  final storageService = StorageService();
  final settingsState = SettingsState();
  await settingsState.load();

  runApp(StonepadApp(
    storageService: storageService,
    settingsState: settingsState,
  ));
}

class StonepadApp extends StatefulWidget {
  final StorageService storageService;
  final SettingsState settingsState;

  const StonepadApp({
    super.key,
    required this.storageService,
    required this.settingsState,
  });

  @override
  State<StonepadApp> createState() => _StonepadAppState();
}

class _StonepadAppState extends State<StonepadApp> {
  late final NotesState _notesState;
  late final SyncStateNotifier _syncState;
  late final LifecycleService _lifecycle;

  @override
  void initState() {
    super.initState();
    _notesState = NotesState(widget.storageService);
    _syncState = SyncStateNotifier();
    _lifecycle = LifecycleService(
      notesState: _notesState,
      syncState: _syncState,
    );
    _lifecycle.register();
  }

  @override
  void dispose() {
    _lifecycle.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: _notesState),
        ChangeNotifierProvider.value(value: _syncState),
        ChangeNotifierProvider.value(value: widget.settingsState),
        ChangeNotifierProvider(create: (_) => ConnectivityState()),
      ],
      child: MaterialApp(
        title: StonepadStrings.appName,
        debugShowCheckedModeBanner: false,
        theme: ThemeData(
          colorSchemeSeed: Colors.blue,
          useMaterial3: true,
          brightness: Brightness.light,
        ),
        darkTheme: ThemeData(
          colorSchemeSeed: Colors.blue,
          useMaterial3: true,
          brightness: Brightness.dark,
        ),
        home: _buildHome(),
        routes: {
          '/notes': (_) => const NotesListScreen(),
          '/settings': (_) => const SettingsScreen(),
        },
      ),
    );
  }

  Widget _buildHome() {
    // If the user hasn't configured anything, show onboarding.
    if (!widget.settingsState.settings.isConfigured) {
      return const OnboardingScreen();
    }
    return const NotesListScreen();
  }
}
