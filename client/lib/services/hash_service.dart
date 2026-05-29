/// Hash service — pure functions for SHA-256 hashing. No state.
library;
import 'dart:convert';
import 'package:crypto/crypto.dart' as crypto;

class HashService {
  /// Compute SHA-256 of a string, returned as lowercase hex.
  static String compute(String input) {
    final bytes = utf8.encode(input);
    final digest = crypto.sha256.convert(bytes);
    return digest.toString();
  }

  /// Compute SHA-256 of raw bytes.
  static String computeBytes(List<int> bytes) {
    final digest = crypto.sha256.convert(bytes);
    return digest.toString();
  }
}
