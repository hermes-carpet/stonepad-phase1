/// Connectivity service — wraps connectivity_plus for testability.
/// See §8.5 of the Stonepad v1 Implementation Plan.
library;
import 'package:connectivity_plus/connectivity_plus.dart';

class ConnectivityService {
  final Connectivity _connectivity;

  ConnectivityService({Connectivity? connectivity})
      : _connectivity = connectivity ?? Connectivity();

  /// Returns true if the device has wifi, ethernet, or mobile connectivity.
  Future<bool> isConnected() async {
    final results = await _connectivity.checkConnectivity();
    return results.any((r) =>
        r == ConnectivityResult.wifi ||
        r == ConnectivityResult.ethernet ||
        r == ConnectivityResult.mobile);
  }

  /// Stream of connectivity changes. Emits true/false for connected state.
  Stream<bool> get onConnectivityChanged {
    return _connectivity.onConnectivityChanged.map((results) {
      return results.any((r) =>
          r == ConnectivityResult.wifi ||
          r == ConnectivityResult.ethernet ||
          r == ConnectivityResult.mobile);
    });
  }
}
