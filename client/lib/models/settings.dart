/// User-configurable settings, persisted to settings.json.
class StonepadSettings {
  String? serverEndpoint;
  String authMode; // ***, "token", "s3"
  String? authToken;
  String? s3AccessKey;
  String? s3SecretKey;
  String workspaceId;
  bool syncEnabled;
  // Relay (optional)
  bool relayEnabled;
  String? relayEndpoint;
  String? relayAccessKey;
  String? relaySecretKey;

  StonepadSettings({
    this.serverEndpoint,
    this.authMode = 'n' 'one',
    this.authToken,
    this.s3AccessKey,
    this.s3SecretKey,
    this.workspaceId = 'default',
    this.syncEnabled = true,
    this.relayEnabled = false,
    this.relayEndpoint,
    this.relayAccessKey,
    this.relaySecretKey,
  });

  factory StonepadSettings.fromJson(Map<String, dynamic> json) {
    return StonepadSettings(
      serverEndpoint: json['server_endpoint'],
      authMode: json['auth_mode'] ?? 'n' 'one',
      authToken: json['auth_token'],
      s3AccessKey: json['s3_access_key'],
      s3SecretKey: json['s3_secret_key'],
      workspaceId: json['workspace_id'] ?? 'default',
      syncEnabled: json['sync_enabled'] ?? true,
      relayEnabled: json['relay_enabled'] ?? false,
      relayEndpoint: json['relay_endpoint'],
      relayAccessKey: json['relay_access_key'],
      relaySecretKey: json['relay_secret_key'],
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'server_endpoint': serverEndpoint,
      'auth_mode': authMode,
      'auth_token': authToken,
      's3_access_key': s3AccessKey,
      's3_secret_key': s3SecretKey,
      'workspace_id': workspaceId,
      'sync_enabled': syncEnabled,
      'relay_enabled': relayEnabled,
      'relay_endpoint': relayEndpoint,
      'relay_access_key': relayAccessKey,
      'relay_secret_key': relaySecretKey,
    };
  }

  /// Whether any sync endpoint is configured.
  bool get hasEndpoint =>
      serverEndpoint != null && serverEndpoint!.isNotEmpty;

  /// Whether the user has configured the app at all (onboarding check).
  bool get isConfigured => hasEndpoint;
}
