# Contributing to Stonepad

## Code Style

- **Go**: `gofmt` all code. Use `go vet`. Keep files under 500 lines where possible.
- **Dart**: `dart format` all code. Use `flutter analyze`.

## Commit Convention

Use [Conventional Commits](https://www.conventionalcommits.org/):

- `feat(server): ...` — new feature for the Go server
- `feat(client): ...` — new feature for the Flutter app
- `fix: ...` — bug fix
- `docs: ...` — documentation only
- `test: ...` — adding tests
- `chore: ...` — CI, build, dependencies

## Local Development

### Server

```bash
cd server
go mod download
go test ./...           # run all tests
CGO_ENABLED=0 go build -o stonepad-server ./cmd/stonepad-server/
NOTES_AUTH_MODE=none NOTES_DATA_DIR=./testdata ./stonepad-server
```

### Client

```bash
cd client
flutter pub get
flutter analyze
flutter test
flutter build linux   # desktop dev build for testing
```

### Docker

```bash
cd server
docker build -t stonepad-server .
docker run -p 8080:8080 -e NOTES_AUTH_MODE=none stonepad-server
```

## PR Template

When opening a pull request:

1. Reference the phase or issue being addressed
2. List what was changed at a high level
3. Include test results (`go test ./...` or `flutter test`)
4. Verify binary size hasn't regressed (server should stay under 20 MB)

## Running End-to-End Tests

```bash
# Start server in background
NOTES_AUTH_MODE=none NOTES_DATA_DIR=/tmp/stonepad-test ./stonepad-server &
sleep 1

# Run integration checks
curl http://localhost:8080/api/v1/health
curl -X PUT http://localhost:8080/api/v1/notes/test.md \
  -H "Content-Type: text/markdown" \
  -d "# Hello Stonepad"
curl http://localhost:8080/api/v1/notes/test.md
```
