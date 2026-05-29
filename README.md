# Stonepad

**Self-hostable markdown notes with mobile sync.** Notes are plain `.md` files on your phone. Sync them to your own server — or don't. Works standalone.

[![CI](https://github.com/hermes-carpet/stonepad-phase1/actions/workflows/server-ci.yml/badge.svg)](https://github.com/hermes-carpet/stonepad-phase1/actions)

---

## Quick Start

### Run the server (Docker)

```bash
docker run -d \
  -p 8080:8080 \
  -v ./stonepad-data:/data \
  -e NOTES_AUTH_MODE=token \
  -e NOTES_AUTH_TOKEN=your-secret-token \
  ghcr.io/hermes-carpet/stonepad-server:latest
```

The server now listens on `http://localhost:8080`. See [Deployment](#deployment) for production setups.

### Build the Flutter app

```bash
cd client && flutter pub get && flutter build apk
```

Install the APK on your phone and configure the server endpoint in Settings.

---

## Features

- **Plain Markdown files** — Notes live as `.md` files in a folder you can browse with any file manager. No proprietary database, no lock-in.
- **Works standalone** — Sync is optional. Use the app entirely offline if you want.
- **Self-hosted sync** — Run your own sync server on a Raspberry Pi, NAS, VPS, or old laptop. No subscription required.
- **Optional cloud relay** — Use Cloudflare R2's free tier as a global sync relay. Phone writes to R2, your home server polls it periodically.
- **Content-hash sync** — Only changed notes are transferred. Conflict files are preserved; nothing is silently overwritten.
- **Apple Notes-style UI** — Folder hierarchy, breadcrumb navigation, dual-pane editor with live Markdown preview.
- **Three auth modes** — None (behind Tailscale), shared token, or username/password with Argon2id hashing.
- **tmpfs mode** — Run the server entirely in RAM with periodic disk snapshots. Survives restarts with zero data loss.
- **S3-compatible API** — Works with `rclone`, `mc` (MinIO Client), and `aws s3` CLI.
- **Under 20 MB Docker image** — Static binary, `FROM scratch`, zero CGo.

## Non-Features (v1)

These are explicitly out of scope for v1. See [docs/architecture.md](docs/architecture.md) for the affordances that make them possible in future versions.

- End-to-end encryption (notes are plaintext on the server)
- Team / multi-user workspaces (the schema supports it; v1 uses fixed `default` workspace)
- Image attachments (plain Markdown only)
- iOS build (Android + Linux dev build only in v1)
- Push notifications / background sync
- Drag-and-drop editor reordering

## Architecture

```
┌──────────────┐     S3 protocol      ┌──────────────────┐
│  Flutter App  │ ──────────────────▶ │  Stonepad Server  │
│  (Android)    │ ◀─ ─ ─ ─ ─ ─ ─ ─ ─ │  (Go, Docker)     │
│               │     (optional)       │                   │
│  .md files    │                      │  .md files on disk │
│  on device    │                      │  SQLite metadata   │
└──────┬───────┘                      └────────┬─────────┘
       │                                       │
       │          Cloudflare R2                 │
       └──────────── (relay) ──────────────────┘
```

**Sync flow:** Phone ↔ Server via S3 protocol (PUT/GET/DELETE/LIST over HTTP). Optional relay: Phone → R2 → Server (polling).

## Documentation

| Document | Description |
|---|---|
| [docs/architecture.md](docs/architecture.md) | Architecture overview, design philosophy, affordances |
| [docs/configuration.md](docs/configuration.md) | All environment variables and defaults |
| [docs/deployment/tailscale.md](docs/deployment/tailscale.md) | Tailscale deployment (recommended) |
| [docs/deployment/cf-tunnel.md](docs/deployment/cf-tunnel.md) | Cloudflare Tunnel deployment |
| [docs/deployment/direct.md](docs/deployment/direct.md) | Direct exposure with reverse proxy |
| [docs/deployment/lan.md](docs/deployment/lan.md) | LAN-only deployment |
| [docs/deployment/tmpfs.md](docs/deployment/tmpfs.md) | RAM-backed tmpfs storage mode |
| [docs/relay/cloudflare-r2.md](docs/relay/cloudflare-r2.md) | Cloudflare R2 relay setup |
| [docs/api.md](docs/api.md) | API reference (native + S3) |
| [docs/contributing.md](docs/contributing.md) | Contributing guide |

## License

Stonepad is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0). See [LICENSE](LICENSE) for the full text.
