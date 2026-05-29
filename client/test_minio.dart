import 'package:flutter/foundation.dart';
import 'package:minio/minio.dart';

void main() async {
  final minio = Minio(
    endPoint: 'localhost',
    port: 8080,
    useSSL: false,
    accessKey: 'STNP6f2c15475a542546',
    secretKey: '35c1e6c5c62d64a0a2fdaea4ee3e0312',
    region: 'us-east-1',
  );

  try {
    final result = await minio.listAllObjects('default', recursive: true);
    debugPrint('Objects: ${result.objects.length}');
    for (final obj in result.objects) {
      debugPrint('  key=${obj.key}, eTag=${obj.eTag}, size=${obj.size}');
    }
  } catch (e, stack) {
    debugPrint('ERROR: $e');
    debugPrint('\$stack'
  }
}
