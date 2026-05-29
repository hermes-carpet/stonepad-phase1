# Configuration Reference

All server configuration is via **environment variables**. No config files in v1.

## Required Variables

| Variable | Default | Description |
|---|---|---|
| `NOTES_AUTH_TOKEN` | (none) | Shared bearer token. **Required** when `NOTES_AUTH_MODE=token`. |

## Server Settings

| Variable | Default | Description |
|---|---|---|
| `NOTES_LISTEN_ADDR` | `:8080` | Address and port the HTTP server listens on |
| `NOTES_DATA_DIR` | `/data` | Root directory for note storage and metadata |
| `NOTES_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `NOTES_WORKSPACE_ID` | `default` | Workspace identifier (single-tenant in v1) |
| `NOTES_USER_ID` | `owner` | Default user ID for single-tenant operation |

## Storage Mode

| Variable | Default | Description |
|---|---|---|
| `NOTES_STORAGE_MODE` | `direct` | `direct` (disk) or `tmpfs` (RAM with snapshots) |
| `NOTES_TMPFS_SNAPSHOT_INTERVAL` | `300` | Seconds between tmpfs-to-disk snapshots |
| `NOTES_TMPFS_PERSIST_DIR` | `/data/persist` | Persistent directory for tmpfs snapshots |

## Authentication

| Variable | Default | Description |
|---|---|---|
| `NOTES_AUTH_MODE` | `none` | `none`, `token`, or `users` |
| `NOTES_AUTH_TOKEN` | (empty) | Shared bearer token for `token` mode |

### Auth modes

- **`none`**: No authentication. All requests pass through. Use behind Tailscale or on LAN.
- **`token`**: Shared bearer token. Every request must include `Authorization: Bearer <token>`. Set `NOTES_AUTH_TOKEN`.
- **`users`**: Username/password with Argon2id hashing and session tokens (30-day expiry). On first startup, a random admin password is generated and printed to logs.

## API Endpoints

| Variable | Default | Description |
|---|---|---|
| `NOTES_S3_ENDPOINT_ENABLED` | `true` | Enable the S3-compatible endpoint at `/s3/` |
| `NOTES_NATIVE_ENDPOINT_ENABLED` | `true` | Enable native REST API at `/api/v1/` |

## Limits

| Variable | Default | Description |
|---|---|---|
| `NOTES_MAX_NOTE_SIZE_BYTES` | `5242880` (5 MB) | Maximum size of a single note |
| `NOTES_MAX_NOTES_PER_WORKSPACE` | `100000` | Maximum number of notes in a workspace |

## Relay (Cloudflare R2)

| Variable | Default | Description |
|---|---|---|
| `NOTES_RELAY_ENABLED` | `false` | Enable R2 relay polling |
| `NOTES_RELAY_ENDPOINT` | (empty) | R2 endpoint URL |
| `NOTES_RELAY_ACCESS_KEY` | (empty) | R2 access key ID |
| `NOTES_RELAY_SECRET_KEY` | (empty) | R2 secret access key |
| `NOTES_RELAY_BUCKET` | (empty) | R2 bucket name |
| `NOTES_RELAY_POLL_INTERVAL` | `300` | Seconds between R2 polls |

### Example: Docker Compose with relay

```yaml
services:
  stonepad-server:
    image: ghcr.io/hermes-carpet/stonepad-server:latest
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    environment:
      NOTES_AUTH_MODE: token
      NOTES_AUTH_TOKEN: "${NOTES_AUTH_TOKEN}"
      NOTES_RELAY_ENABLED: "true"
      NOTES_RELAY_ENDPOINT: "https://<account>.r2.cloudflarestorage.com"
      NOTES_RELAY_ACCESS_KEY: "${R2_ACCESS_KEY}"
      NOTES_RELAY_SECRET_KEY: "${R2_SECRET_KEY}"
      NOTES_RELAY_BUCKET: "stonepad-relay"
```
