# Cloudflare R2 Relay Setup

Use Cloudflare R2's free tier (10 GB storage, no egress fees) as a globally-distributed sync relay. Your phone writes directly to R2, and your home server polls R2 periodically for changes.

## Prerequisites

- A Cloudflare account (free tier works)
- Your home server running Stonepad with Docker

## Step 1: Create an R2 Bucket

1. Log in to the [Cloudflare Dashboard](https://dash.cloudflare.com/).
2. Navigate to **R2** → **Create bucket**.
3. Name: `stonepad-relay` (or any name you prefer).
4. Leave default settings.

## Step 2: Generate API Token

1. In the Cloudflare Dashboard, go to **R2** → **Manage R2 API Tokens**.
2. Click **Create API Token**.
3. Set permissions:
   - **Object Read & Write** — required for sync.
4. Scope: select the bucket you created.
5. Copy the **Access Key ID** and **Secret Access Key**. Store them securely.

## Step 3: Configure the Home Server

Add relay configuration to your `docker-compose.yml`:

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
      NOTES_AUTH_MODE: token
      NOTES_AUTH_TOKEN: "${NOTES_AUTH_TOKEN}"

      # Relay configuration
      NOTES_RELAY_ENABLED: "true"
      NOTES_RELAY_ENDPOINT: "https://<account-id>.r2.cloudflarestorage.com"
      NOTES_RELAY_ACCESS_KEY: "${R2_ACCESS_KEY}"
      NOTES_RELAY_SECRET_KEY: "${R2_SECRET_KEY}"
      NOTES_RELAY_BUCKET: "stonepad-relay"
      NOTES_RELAY_POLL_INTERVAL: "300"
```

Replace `<account-id>` with your Cloudflare account ID (visible in the R2 dashboard URL).

## Step 4: Configure the Flutter App

In the Stonepad app settings:

1. **Server endpoint URL**: `https://<account-id>.r2.cloudflarestorage.com`
2. **Auth mode**: "S3 Keys"
3. **Access Key ID**: Paste the R2 Access Key ID
4. **Secret Access Key**: Paste the R2 Secret Access Key
5. **Workspace ID**: `default`

The app will now write directly to R2. Your home server polls R2 every 5 minutes and pulls in new notes.

## Step 5: Verify End-to-End

1. Create a note in the Flutter app
2. Wait for the sync interval (or tap "Sync now")
3. Check the R2 bucket in the Cloudflare dashboard — the note file should appear
4. Check your home server:

```bash
# After the poll interval, the server should have pulled the note
curl -H "Authorization: Bearer your-server-token" \
  http://localhost:8080/api/v1/manifest
```

## Architecture

```
Phone ──(S3 PUT/GET)──▶ Cloudflare R2 ◀──(periodic poll)── Home Server
                              │
                        Global edge caching
                        (free tier: 10 GB)
```

## Troubleshooting

**"relay: list objects failed" in server logs**
- Verify the endpoint URL includes `https://` and your account ID
- Check that the access key and secret are correct
- Ensure the bucket name matches exactly

**Notes not appearing on the server**
- The default poll interval is 300 seconds (5 minutes). Wait for the next poll cycle.
- Check server logs for relay errors.
