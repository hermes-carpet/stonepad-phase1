# Direct Exposure Deployment

For users who want to expose Stonepad publicly with a reverse proxy.

## Prerequisites

- A domain name with DNS pointing to your home server
- Docker installed
- A reverse proxy (Caddy, Traefik, or Nginx)

## Setup with Caddy

1. Create `~/stonepad/docker-compose.yml`:

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

  caddy:
    image: caddy:2-alpine
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy_data:/data
      - caddy_config:/config

volumes:
  caddy_data:
  caddy_config:
```

2. Create `~/stonepad/Caddyfile`:

```
stonepad.yourdomain.com {
    reverse_proxy stonepad-server:8080
}
```

3. Start:

```bash
NOTES_AUTH_TOKEN=your-secret-token docker compose up -d
```

## Verification

```bash
curl -H "Authorization: Bearer your-secret-token" \
  https://stonepad.yourdomain.com/api/v1/health
```

## Security Notes

- Caddy automatically obtains and renews Let's Encrypt certificates.
- Always use HTTPS in production.
- Consider adding rate limiting.
