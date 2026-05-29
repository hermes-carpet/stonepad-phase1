# API Reference

## Native REST API

Base path: `/api/v1`. All endpoints require authentication per `NOTES_AUTH_MODE`.

### Health Check

```
GET /api/v1/health

Response 200:
{
  "status": "ok"
}
```

No authentication required.

### Manifest

```
GET /api/v1/manifest
Authorization: Bearer <token>

Response 200:
{
  "version": 1,
  "workspace_id": "default",
  "generated_at": "2026-04-22T15:30:00Z",
  "notes": [
    {
      "path": "work/meetings/2026-04-22.md",
      "content_hash": "a1b2c3d4e5f6...",
      "size_bytes": 1843,
      "modified_at": "2026-04-22T14:12:00Z"
    }
  ]
}
```

### Get Note

```
GET /api/v1/notes/{path}
Authorization: Bearer <token>

Response 200:
Content-Type: text/markdown; charset=utf-8

{raw markdown content}
```

### Put Note (Create/Update)

```
PUT /api/v1/notes/{path}
Authorization: Bearer <token>
Content-Type: text/markdown
If-Match: <optional-hash>     ← optimistic concurrency

{raw markdown content}

Response 200:
{
  "path": "work/meetings/note.md",
  "content_hash": "new_hash",
  "size_bytes": 1893,
  "modified_at": "2026-04-22T15:30:00Z"
}
```

### Delete Note

```
DELETE /api/v1/notes/{path}
Authorization: Bearer <token>

Response 204: (no content)
```

### Login (users mode only)

```
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password"
}

Response 200:
{
  "token": "base64-encoded-session-token"
}

Response 401:
{
  "error": {
    "code": "unauthorized",
    "message": "invalid username or password"
  }
}
```

### Logout (users mode only)

```
POST /api/v1/auth/logout
Authorization: Bearer <session-token>

Response 204: (no content)
```

## Error Format

All errors return a consistent JSON structure:

```json
{
  "error": {
    "code": "not_found",
    "message": "Note not found at path 'foo/bar.md'"
  }
}
```

Error codes:

| HTTP Status | Code |
|---|---|
| 400 | `bad_request` |
| 401 | `unauthorized` |
| 403 | `forbidden` |
| 404 | `not_found` |
| 409 | `conflict` |
| 412 | `precondition_failed` |
| 413 | `payload_too_large` |
| 500 | `internal_error` |

## S3-Compatible Endpoint

Base path: `/s3`. Single bucket named after `NOTES_WORKSPACE_ID` (default: `default`).

Authentication: AWS Signature V4. Credentials are generated on first startup and stored in `{NOTES_DATA_DIR}/credentials.txt`.

### Supported Operations

| Operation | Method | Path | Notes |
|---|---|---|---|
| ListBuckets | GET | `/s3/` | Returns single bucket entry |
| ListObjectsV2 | GET | `/s3/{bucket}?list-type=2` | Supports `prefix` and `continuation-token` |
| HeadObject | HEAD | `/s3/{bucket}/{key}` | Returns ETag (SHA-256) and metadata |
| GetObject | GET | `/s3/{bucket}/{key}` | Returns raw note content |
| PutObject | PUT | `/s3/{bucket}/{key}` | Creates/updates note |
| DeleteObject | DELETE | `/s3/{bucket}/{key}` | Deletes note |

### Unsupported S3 Operations (returns 501 Not Implemented)

- Multipart uploads
- Object versioning
- Bucket policies / ACLs
- CORS configuration
- Object tagging
- Server-side encryption

### Testing with rclone

```bash
# Read credentials from server's credentials.txt
export AWS_ACCESS_KEY_ID=<access-key>
export AWS_SECRET_ACCESS_KEY=<secret-key>

rclone ls :s3,provider=Other,endpoint=http://localhost:8080:default
rclone copy ./local-notes :s3,provider=Other,endpoint=http://localhost:8080:default
```
