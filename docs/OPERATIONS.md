# Operations

## Server Configuration
Environment variables:
1. `GOPHKEEPER_ADDR` default `:8080`
2. `GOPHKEEPER_DB` default `file:gophkeeper.db?_pragma=busy_timeout(5000)`
3. `GOPHKEEPER_TOKEN_SECRET` default `dev-secret-change-me`
4. `GOPHKEEPER_TOKEN_TTL` default `24h`

## Running Locally
```bash
go build ./cmd/server
./server
```

## Docker
Build:
```bash
docker build -t gophkeeper:dev .
```

Run:
```bash
docker run -p 8080:8080 -e GOPHKEEPER_TOKEN_SECRET=change-me gophkeeper:dev
```

## Docker Compose
```bash
docker compose up --build
```

## Backups
SQLite lives at `/data/gophkeeper.db` when using Docker. Back up the volume.
