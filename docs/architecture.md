# Architecture Overview

## Design Philosophy

Stonepad follows five core principles that inform every architectural decision:

1. **Files are the source of truth.** Notes live as plain `.md` files on disk. Metadata and indexes are derived from those files and can be rebuilt at any time by rescanning. If the SQLite database is lost, the notes are still there.
2. **Battery life is the #1 constraint.** Every decision that affects the mobile client is evaluated against battery cost. No background sync. Aggressive debouncing. Minimal network requests.
3. **Disk writes are minimized.** Notes are saved on a debounce timer (7 seconds after last edit), not on every keystroke.
4. **The relay is optional.** The default path is phone ↔ home server. Cloud relay is opt-in for users who want global edge latency.
5. **Fully open source.** Only Dart/Flutter and Go. No other languages.

## Component Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Flutter Client (Dart)                 │
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐     │
│  │ UI Layer  │  │  State    │  │   Services         │     │
│  │ (Screens) │◀─│ (Provider)│──│ (Storage, Sync,    │     │
│  │           │  │           │  │  Auth, Lifecycle)  │     │
│  └──────────┘  └──────────┘  └────────┬──────────┘     │
│                                       │                 │
│                          ┌────────────▼───────────┐     │
│                          │  S3 Client (minio)      │     │
│                          │  Native API (http)      │     │
│                          └─────────────────────────┘     │
└─────────────────────────────────────────────────────────┘
                           │
                           │ HTTPS (S3 protocol)
                           ▼
┌─────────────────────────────────────────────────────────┐
│                    Go Server (stonepad-server)           │
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐     │
│  │ HTTP     │  │  Auth    │  │   Storage          │     │
│  │ Router   │──│ (none/   │──│   (direct/tmpfs)   │     │
│  │ (net/http)│ │  token/  │  │                    │     │
│  │          │  │  users)  │  └───────────────────┘     │
│  └──────────┘  └──────────┘                             │
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐     │
│  │ S3       │  │ Metadata │  │  Relay             │     │
│  │ Endpoint │  │ (SQLite) │  │  (R2 polling)      │     │
│  └──────────┘  └──────────┘  └───────────────────┘     │
└─────────────────────────────────────────────────────────┘
```

## Sync Protocol

Stonepad uses **content-hash-based sync**, not byte-level diffing:

1. Client computes SHA-256 of each note and stores it in a local `manifest.json`.
2. Server returns a manifest of all notes (path → SHA-256 hash).
3. Client compares local vs. server manifests.
4. Notes that differ are pushed or pulled as full files.
5. Conflict files are preserved; nothing is silently overwritten.

The sync cycle frequency is 7 seconds (foreground only). The server speaks the **S3 protocol** — any S3-compatible storage (R2, MinIO, Ceph, AWS S3) can serve as a relay.

## Storage Layout

### Client

```
{app-storage}/
├── notes/default/       ← .md files in folder hierarchy
├── conflicts/           ← conflicting server versions
├── manifest.json        ← metadata + sync state
└── settings.json        ← user preferences
```

### Server (direct mode)

```
{NOTES_DATA_DIR}/
├── notes/{workspace_id}/  ← .md files
├── meta.db                ← SQLite (notes table, auth, audit log)
└── credentials.txt        ← S3 access keys (0600)
```

### Server (tmpfs mode)

```
{NOTES_DATA_DIR}/  (tmpfs, in RAM)
├── notes/...
├── meta.db...
        │
        │ periodic snapshot (every 300s)
        ▼
{NOTES_TMPFS_PERSIST_DIR}/  (persistent disk)
├── notes/...
├── meta.db
└── last_snapshot.txt
```

## Affordances for Future Versions

These are deliberately left open in v1:

| Future Feature | v1 Affordance |
|---|---|
| E2EE | Server never parses note content — encryption wrapper in v2 stays transparent |
| Teams | `workspace_id` and `user_id` columns present in all SQLite tables |
| Custom storage | `Storage` interface allows swapping filesystem backends |
| OAuth/LDAP | `Authenticator` interface allows adding new auth providers |
| Attachments | Path validation already allows arbitrary file extensions on S3 endpoint |

## Database Schema

The SQLite database (`meta.db`) contains:

- `notes` — path, content_hash, size_bytes, modified_at, workspace_id
- `users` — user_id, username, password_hash (Argon2id), created_at
- `auth_tokens` — token_hash (SHA-256), user_id, expires_at
- `audit_log` — workspace_id, user_id, action, path, timestamp

## Binary Size

The Go server is built as a single static binary with:
- `CGO_ENABLED=0` (pure Go, no C dependencies)
- `-ldflags="-w -s"` (strip debug info and symbol table)
- Pure-Go SQLite driver (`modernc.org/sqlite`)
- Standard library HTTP (`net/http`)
- Target: under 20 MB. Current: ~11 MB.

The Docker image uses `FROM scratch` — no base OS, just the binary and CA certificates.
