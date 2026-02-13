# GophKeeper

GophKeeper is a client-server password manager with end-to-end encryption. The server stores only encrypted payloads while clients handle encryption, decryption, and sync.

## Highlights
1. End-to-end encrypted secrets (server never sees plaintext).
2. Secret types: login/password, text, binary, card.
3. CLI and TUI clients with sync support.
4. OpenAPI specification in `docs/openapi.yaml`.

## Quick Start (Local)
Build binaries:
```bash
go build ./cmd/server
go build ./cmd/client
go build ./cmd/tui
```

Run the server:
```bash
./server
```

Register and login (CLI):
```bash
./client register -user alice -pass secret
./client login -user alice -pass secret
```

Add and list secrets (CLI):
```bash
./client add -type text -text "hello" -user-pass secret
./client add -type login -login "alice" -password "p@ss" -user-pass secret
./client list -user-pass secret
```

Sync:
```bash
./client sync
```

Run the TUI:
```bash
export GOPHKEEPER_SERVER="http://localhost:8080"
export GOPHKEEPER_USER_PASS="secret"
./tui
```

## TUI Keymap
1. `l` login
2. `r` register
3. `s` sync
4. `a` add item
5. `e` edit selected
6. `d` delete selected
7. `enter` view details
8. `f` filter
9. `q` quit

Binary secret input in TUI:
1. `file:/path/to/file`
2. `b64:<base64>`

## Configuration
Server environment variables:
1. `GOPHKEEPER_ADDR` default `:8080`
2. `GOPHKEEPER_DB` default `file:gophkeeper.db?_pragma=busy_timeout(5000)`
3. `GOPHKEEPER_TOKEN_SECRET` default `dev-secret-change-me`
4. `GOPHKEEPER_TOKEN_TTL` default `24h`

Client environment variables:
1. `GOPHKEEPER_SERVER` for CLI/TUI server URL
2. `GOPHKEEPER_USER_PASS` for TUI decryption password

Client local store:
`~/.config/gophkeeper/store.json` (platform-specific config directory).

## CLI Notes
1. `register` and `login` save the session token to the local store.
2. `list` reads from the local cache only.
3. `sync` pulls updates from the server and refreshes the local cache.

## Docker
Build and run:
```bash
docker build -t gophkeeper:dev .
docker run -p 8080:8080 -e GOPHKEEPER_TOKEN_SECRET=change-me gophkeeper:dev
```

Compose:
```bash
docker compose up --build
```

## API (OpenAPI)
The OpenAPI spec is located at `docs/openapi.yaml`.

Validate:
```bash
go run github.com/getkin/kin-openapi/cmd/validate@latest -- docs/openapi.yaml
```

## Build Metadata
Embed version info at build time:
```bash
go build -ldflags "-X gophkeeper/internal/version.Version=1.0.0 -X gophkeeper/internal/version.BuildDate=2026-02-12"
```

## Development
Run tests:
```bash
go test ./...
```

Integration tests (TUI golden + e2e + OpenAPI contract):
```bash
go test -tags=integration ./...
```

Lint:
```bash
golangci-lint run
```
