# Release & Versioning

## Version Scheme

Stonepad uses `MAJOR.MINOR.PATCH` (semver). The **PATCH** digit alone determines release type:

| PATCH | Release Type | GitHub | Trigger |
|-------|-------------|--------|---------|
| **≠ 0** (e.g. `0.0.3`, `1.2.7`) | Pre-release | 🟠 `prerelease: true` | Auto on push to `main` |
| **= 0** (e.g. `0.5.0`, `1.0.0`) | Stable | 🟢 Full release | Manual `git tag vX.Y.0` |

**No "alpha" labels, no commit SHAs in tags.** The PATCH digit IS the indicator.
`v0.0.3` is a pre-release; `v0.5.0` is stable.

### Decimal positions

| Position | Bump meaning |
|----------|-------------|
| **PATCH** (3rd) | Individual bug fixes, small changes |
| **MINOR** (2nd) | Grouped fixes, new features, dev-branch merges |
| **MAJOR** (1st) | Breaking changes, UX overhauls, major redesigns |

## Docker Images

Images are tagged by version from `server/VERSION`:

```
ghcr.io/hermes-carpet/stonepad-server:0.0.3   # exact release
ghcr.io/hermes-carpet/stonepad-server:0.0      # latest patch on this minor
ghcr.io/hermes-carpet/stonepad-server:latest    # most recent push
```

Pull a specific version:

```bash
docker pull ghcr.io/hermes-carpet/stonepad-server:0.0.3
```

## Pre-releases (Automated)

Every push to `main` with PATCH ≠ 0 automatically publishes a pre-release:

1. CI runs: lint → test → build → Docker push (versioned tags)
2. Pre-release workflow fires: creates git tag, builds binaries/APK, publishes GitHub pre-release

### Server

- **Version source:** `server/VERSION` (bump before committing)
- **Tag:** `v{VERSION}` (e.g., `v0.0.3`)
- **Assets:** `stonepad-server_linux_amd64`, `stonepad-server_linux_arm64`
- **Workflow:** `server-pre-release.yml`

### Client (Android APK)

- **Version source:** `client/pubspec.yaml` (`version:` field)
- **Tag:** `v{VERSION}` (e.g., `v0.0.3`)
- **Assets:** Signed APK (`stonepad-{version}-universal.apk`)
- **Workflow:** `client-pre-release.yml`

## Stable Releases (Manual)

To publish a stable release, push a tag ending in `.0`:

```bash
# Ensure VERSION or pubspec.yaml has PATCH=0
git tag v0.5.0
git push origin v0.5.0
```

This triggers the stable release pipeline:

- **Server:** Multi-arch binaries + Docker image (semver tags: `0.5.0`, `0.5`, `0`)
- **Client:** Signed APK attached to GitHub release

## Path Gating

CI workflows only run when their source directories change:

| Push changes to | CI triggered | Pre-release triggered |
|----------------|-------------|----------------------|
| `server/**` only | Server CI | Server pre-release |
| `client/**` only | Client CI | Client pre-release |
| Both | Both | Both |
| Docs/CI only | Neither | Neither |

## Manual CI Trigger

To manually kick off CI for the latest commit on `main`:

```bash
gh workflow run "Server CI" --ref main
gh workflow run "Client CI" --ref main
```

Pre-releases will fire automatically when CI completes successfully.
