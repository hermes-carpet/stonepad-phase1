# Cloudflare Tunnel Deployment

## Prerequisites

- A domain name managed by Cloudflare
- Docker installed on the home server
- [Cloudflare Tunnel](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/) (cloudflared)

## Setup

1. Create a directory:

```bash
mkdir ~/stonepad && cd ~/stonepad
```

2. Generate a secure auth token:

```bash
openssl rand -base64 32
```

3. Create a Cloudflare Tunnel via the Zero Trust dashboard and get your tunnel token.

4. Create `docker-compose.yml`:

```yaml
services:
  stonepad-server:
    image: ghcr.io/hermes-carpet/stonepad-server:latest
    restart: unless-stopped
    expose:
      - "8080"
    volumes:
      - ./data:/data
    environment:
      NOTES_AUTH_MODE: token
      NOTES_AUTH_TOKEN: "${NOTES_AUTH_TOKEN}"

  cloudflared:
    image: cloudflare/cloudflared:latest
    restart: unless-stopped
    command: tunnel --no-autoupdate run --token ${CLOUDFLARE_TUNNEL_TOKEN}
```

5. Start:

```bash
CLOUDFLARE_TUNNEL_TOKEN=your-token NOTES_AUTH_TOKEN=your-secret-token \
  docker compose up -d
```

## Configure the Flutter App

1. Set the endpoint to `https://stonepad.yourdomain.com`
2. Use auth mode "Token" with your `NOTES_AUTH_TOKEN`

## Security Notes

- All traffic is encrypted via Cloudflare's edge.
- Add Cloudflare Access policies for additional authentication layers.
