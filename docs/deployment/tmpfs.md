# tmpfs (RAM-Backed) Storage Mode

Run the server entirely in RAM for maximum performance. Data is periodically snapshotted to disk so it survives restarts.

## Prerequisites

- Docker with sufficient RAM (notes + metadata must fit)
- Persistent storage for snapshots (SSD or HDD)

## Setup

```yaml
services:
  stonepad-server:
    image: ghcr.io/hermes-carpet/stonepad-server:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    tmpfs:
      - /data:size=512M
    volumes:
      - ./persist:/data/persist
    environment:
      NOTES_STORAGE_MODE: tmpfs
      NOTES_DATA_DIR: "/data"
      NOTES_TMPFS_PERSIST_DIR: "/data/persist"
      NOTES_TMPFS_SNAPSHOT_INTERVAL: "300"
      NOTES_AUTH_MODE: token
      NOTES_AUTH_TOKEN: "${NOTES_AUTH_TOKEN}"
```

## How It Works

1. **On startup**: The server copies the last snapshot from `./persist/` into the tmpfs `/data/`.
2. **During operation**: All reads and writes go to RAM (tmpfs).
3. **Every 300 seconds**: A snapshot copies the RAM state to `./persist/` using atomic rename to prevent corruption.
4. **On shutdown**: A final synchronous snapshot runs before the process exits.

## Snapshot Verification

Check the snapshot timestamp:

```bash
cat ./persist/last_snapshot.txt
```

## Recovery Test

1. Create a note via the API
2. Wait 300 seconds for a snapshot
3. Restart the container:

```bash
docker compose restart
```

4. The note should still be there — verify with:

```bash
curl -H "Authorization: Bearer your-token" \
  http://localhost:8080/api/v1/manifest
```
