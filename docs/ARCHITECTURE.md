# Architecture

## Overview
GophKeeper is a client-server password manager with end-to-end encryption. Clients encrypt data locally and the server stores only ciphertext and metadata needed for synchronization.

## Components
1. HTTP server (`cmd/server`)
2. CLI client (`cmd/client`)
3. TUI client (`cmd/tui`)

## Package Layout
```
cmd/
  server/           HTTP API entrypoint
  client/           CLI entrypoint
  tui/              TUI entrypoint
internal/
  app/
    auth/           auth use-cases
    cli/            CLI use-cases
    client/         client domain logic, sync, crypto envelope
    tui/            TUI state machine and views
    vault/          secret storage use-cases
  domain/
    secret/         secret types and validation
    user/           user domain types
  infra/
    client/         API client, local store, crypto implementation
    httpapi/        HTTP transport and handlers
    persistence/    SQLite repositories
    security/       password hashing and token signing
  ports/            interfaces for external deps
```

## Data Flow
1. Client derives an encryption key from user password and a per-user salt.
2. Secret data is wrapped in a payload envelope and encrypted locally.
3. The server stores ciphertext and metadata only.
4. Sync uses a monotonic `version` to exchange deltas and resolve conflicts.

## Conflict Resolution
Updates carry a `version` (timestamp). The server rejects stale writes with a conflict. Clients surface conflicts and recommend a sync before retrying.

## Storage
1. Server: SQLite (configurable DSN).
2. Client: JSON store at `~/.config/gophkeeper/store.json` (or platform equivalent).

## API
OpenAPI specification is available at `docs/openapi.yaml`.
