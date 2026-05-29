/// Stonepad — Self-hostable markdown notes application.
/// Mobile-first Flutter client with optional sync.
/// See §8 of the Stonepad v1 Implementation Plan.
library;
import 'dart:io';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:window_manager/window_manager.dart';
import 'constants/strings.dart';
import 'services/storage_service.dart';
import 'services/sync_service.dart';
import 'services/lifecycle_service.dart';
import 'state/notes_state.dart';
import 'state/sync_state_notifier.dart';
import 'state/settings_state.dart';
import 'state/connectivity_state.dart';
import 'screens/notes_list_screen.dart';
import 'screens/settings_screen.dart';
import 'screens/onboarding_screen.dart';
import 'screens/login_screen.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Linux/macOS/Windows desktop: lock to portrait phone size for dev testing.
  // See §8.13 — this exists exclusively for the AI-driven dev feedback loop.
  if (Platform.isLinux || Platform.isWindows || Platform.isMacOS) {
    try {
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
    } catch (e) {
      // window_manager may fail in headless/XWayland environments.
      // Continue without window sizing — the app still runs.
      print('Window manager unavailable (running headless): $e');
    }
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
  late final ConnectivityState _connectivityState;
  late final SyncService _syncService;
  late final LifecycleService _lifecycle;

  @override
  void initState() {
    super.initState();
    _notesState = NotesState(widget.storageService);
    _syncState = SyncStateNotifier();
    _connectivityState = ConnectivityState();

    // Initialize sync state from settings
    if (widget.settingsState.settings.syncEnabled) {
      _syncState.setSyncEnabled(true);
    } else {
      _syncState.setSyncEnabled(false);
    }

    _syncService = SyncService(
      notesState: _notesState,
      syncState: _syncState,
      connectivity: _connectivityState,
      settingsState: widget.settingsState,
      storage: widget.storageService,
    );

    // Start polling if sync is enabled and endpoint is configured
    if (widget.settingsState.settings.hasEndpoint &&
        widget.settingsState.settings.syncEnabled) {
      _syncService.startPolling();
    }

    // Wire connectivity changes to SyncService
    _connectivityState.addListener(() {
      _syncService.onConnectivityChanged(_connectivityState.isConnected);
    });

    _lifecycle = LifecycleService(
      notesState: _notesState,
      syncState: _syncState,
    );
    _lifecycle.register();
  }

  @override
  void dispose() {
    _syncService.dispose();
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
        ChangeNotifierProvider.value(value: _connectivityState),
        Provider.value(value: _syncService),
      ],
      child: MaterialApp(
        title: StonepadStrings.appName,
        debugShowCheckedModeBanner: false,
        theme: ThemeData(
          // Warm monochrome palette with muted slate accent — not generic Material blue.
          // Per taste skills: avoid AI-default blue/purple, use off-black text, warm bone canvas.
          colorScheme: ColorScheme.fromSeed(
            seedColor: const Color(0xFF475569), // Slate — muted, professional
            brightness: Brightness.light,
          ),
          useMaterial3: true,
          brightness: Brightness.light,
          scaffoldBackgroundColor: const Color(0xFFFBFBFA), // Warm bone
          cardColor: const Color(0xFFFFFFFF),
          dividerColor: const Color(0xFFEAEAEA),
          appBarTheme: const AppBarTheme(
            backgroundColor: Color(0xFFFBFBFA),
            foregroundColor: Color(0xFF1A1A1A),
            elevation: 0,
            scrolledUnderElevation: 0.5,
          ),
          textTheme: const TextTheme(
            headlineLarge: TextStyle(fontSize: 28, fontWeight: FontWeight.w700, letterSpacing: -0.5, color: Color(0xFF1A1A1A)),
            headlineMedium: TextStyle(fontSize: 24, fontWeight: FontWeight.w600, letterSpacing: -0.3, color: Color(0xFF1A1A1A)),
            titleLarge: TextStyle(fontSize: 20, fontWeight: FontWeight.w600, color: Color(0xFF1A1A1A)),
            titleMedium: TextStyle(fontSize: 16, fontWeight: FontWeight.w500, color: Color(0xFF1A1A1A)),
            bodyLarge: TextStyle(fontSize: 16, height: 1.6, color: Color(0xFF555555)),
            bodyMedium: TextStyle(fontSize: 14, height: 1.5, color: Color(0xFF666666)),
            labelSmall: TextStyle(fontSize: 11, fontWeight: FontWeight.w600, letterSpacing: 0.5, color: Color(0xFF475569)),
          ),
          filledButtonTheme: FilledButtonThemeData(
            style: FilledButton.styleFrom(
              backgroundColor: const Color(0xFF1A1A1A),
              foregroundColor: Colors.white,
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
              padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
            ),
          ),
          inputDecorationTheme: InputDecorationTheme(
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: Color(0xFFEAEAEA)),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: Color(0xFFEAEAEA)),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: Color(0xFF475569), width: 1.5),
            ),
            contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
          ),
          cardTheme: CardThemeData(
            elevation: 0,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(10),
              side: const BorderSide(color: Color(0xFFEAEAEA)),
            ),
          ),
          listTileTheme: const ListTileThemeData(
            contentPadding: EdgeInsets.symmetric(horizontal: 16, vertical: 2),
          ),
        ),
        darkTheme: ThemeData(
          colorScheme: ColorScheme.fromSeed(
            seedColor: const Color(0xFF94A3B8), // Light slate for dark
            brightness: Brightness.dark,
          ),
          useMaterial3: true,
          brightness: Brightness.dark,
          scaffoldBackgroundColor: const Color(0xFF111111),
          appBarTheme: const AppBarTheme(
            backgroundColor: Color(0xFF111111),
            elevation: 0,
            scrolledUnderElevation: 0.5,
          ),
          dividerColor: const Color(0xFF2A2A2A),
          filledButtonTheme: FilledButtonThemeData(
            style: FilledButton.styleFrom(
              backgroundColor: const Color(0xFFFBFBFA), // Warm bone (not stark white)
              foregroundColor: const Color(0xFF111111),
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
              padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
            ),
          ),
          outlinedButtonTheme: OutlinedButtonThemeData(
            style: OutlinedButton.styleFrom(
              foregroundColor: const Color(0xFF94A3B8), // Slate accent now visible
              side: const BorderSide(color: Color(0xFF94A3B8)),
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
              padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
            ),
          ),
          inputDecorationTheme: InputDecorationTheme(
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: Color(0xFF2A2A2A)),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: Color(0xFF2A2A2A)),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: Color(0xFF94A3B8), width: 1.5),
            ),
            contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
          ),
        ),
        home: _buildHome(),
        routes: {
          '/notes': (_) => const NotesListScreen(),
          '/settings': (_) => const SettingsScreen(),
          '/login': (_) => const LoginScreen(),
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
