# Tailscale Deployment (Recommended)

## Prerequisites

- [Tailscale](https://tailscale.com/) installed on your home server and phone
- Both devices connected to the same Tailnet
- Docker installed on the home server

## Setup

1. Create a directory for Stonepad data:

```bash
mkdir ~/stonepad && cd ~/stonepad
```

2. Generate a secure auth token:

```bash
openssl rand -base64 32
```

3. Create `docker-compose.yml`:

```yaml
services:
  stonepad-server:
    image: ghcr.io/hermes-carpet/stonepad-server:latest
    container_name: stonepad-server
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    environment:
      NOTES_LISTEN_ADDR: ":8080"
      NOTES_DATA_DIR: "/data"
      NOTES_STORAGE_MODE: "direct"
      NOTES_AUTH_MODE: token
      NOTES_AUTH_TOKEN: "${NOTES_AUTH_TOKEN}"
      NOTES_LOG_LEVEL: info
```

4. Start the server:

```bash
NOTES_AUTH_TOKEN=your-secret-token docker compose up -d
```

5. Find your server's Tailscale hostname:

```bash
tailscale status
```

Your server will be accessible at `http://<hostname>:8080`.

## Configure the Flutter App

1. Install the Stonepad APK on your phone
2. Open Settings → Server endpoint URL
3. Enter `http://<hostname>:8080` (the Tailscale MagicDNS name)
4. Set auth mode to "Token" and paste your `NOTES_AUTH_TOKEN`
5. Tap "Test connection" or "Sync now"

## Verification

```bash
curl -H "Authorization: Bearer your-secret-token" \
  http://<hostname>:8080/api/v1/manifest
```

Should return `{"version":1,"workspace_id":"default","notes":[]}`.

## Security Notes

- The server is only accessible within your Tailnet. No public exposure.
- Auth tokens travel over WireGuard encryption (Tailscale's default).
- You can use `NOTES_AUTH_MODE=none` if you trust your Tailnet.
