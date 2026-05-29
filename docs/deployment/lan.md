# LAN-Only Deployment

For users who only want sync on their home WiFi network.

## Prerequisites

- A device on your home network to run the server (Raspberry Pi, NAS, old laptop)
- Docker installed

## Setup

1. Create a directory:

```bash
mkdir ~/stonepad && cd ~/stonepad
```

2. Create `docker-compose.yml`:

```yaml
services:
  stonepad-server:
    image: ghcr.io/hermes-carpet/stonepad-server:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    environment:
      NOTES_AUTH_MODE: none
      NOTES_DATA_DIR: "/data"
```

3. Start:

```bash
docker compose up -d
```

4. Find your server's LAN IP:

```bash
hostname -I | awk '{print $1}'
```

## Configure the Flutter App

1. Set the endpoint to `http://<lan-ip>:8080`
2. Auth mode: "None" (no token needed on trusted LAN)

## Security Notes

- Only accessible from your local network. No public exposure.
- Use `NOTES_AUTH_MODE=none` for convenience on trusted LAN.
- If your WiFi has guests, consider using `NOTES_AUTH_MODE=token` instead.
