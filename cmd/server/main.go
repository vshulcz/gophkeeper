// Command gophkeeper-server provides the HTTP server.
package main

import (
	"context"
	"errors"
	"gophkeeper/internal/app/auth"
	"gophkeeper/internal/app/vault"
	"gophkeeper/internal/infra/httpapi"
	"gophkeeper/internal/infra/persistence"
	"gophkeeper/internal/infra/security"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	if err := runServer(context.Background(), func(s *http.Server) error {
		return s.ListenAndServe()
	}); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func runServer(ctx context.Context, serve func(*http.Server) error) error {
	addr := envOrDefault("GOPHKEEPER_ADDR", ":8080")
	dsn := envOrDefault("GOPHKEEPER_DB", "file:gophkeeper.db?_pragma=busy_timeout=5000")
	secret := envOrDefault("GOPHKEEPER_TOKEN_SECRET", "dev-secret-change-me")
	tokenTTL := durationOrDefault("GOPHKEEPER_TOKEN_TTL", 24*time.Hour)

	db, err := persistence.OpenSQLite(ctx, dsn)
	if err != nil {
		return err
	}

	userRepo := persistence.NewUserRepo(db)
	itemRepo := persistence.NewItemRepo(db)

	hasher := security.NewBcryptHasher(12)
	signer := security.NewHMACTokenSigner([]byte(secret))

	authSvc := auth.NewService(userRepo, hasher, signer, time.Now, tokenTTL)
	vaultSvc := vault.NewService(itemRepo, time.Now)

	srv := httpapi.NewServer(authSvc, vaultSvc, signer, tokenTTL)

	server := &http.Server{
		Addr:              addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := serve(server); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func durationOrDefault(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
