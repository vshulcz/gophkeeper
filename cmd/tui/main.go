// Command gophkeeper-tui provides the terminal UI client.
package main

import (
	"context"
	"gophkeeper/internal/app/tui"
	"gophkeeper/internal/infra/client"
	"gophkeeper/internal/version"
	"log"
	"os"
	"time"

	appclient "gophkeeper/internal/app/client"

	"github.com/google/uuid"
)

func main() {
	server := envOrDefault("GOPHKEEPER_SERVER", "http://localhost:8080")
	password := envOrDefault("GOPHKEEPER_USER_PASS", "")

	api := client.NewAPIClient(server)
	store, err := client.NewFileStore()
	if err != nil {
		log.Fatal(err)
	}
	crypto := client.NewCryptoService()
	svc := appclient.NewService(api, store, crypto, time.Now)

	model := tui.New(&tuiService{svc: svc}, password, tui.Deps{
		Now:      time.Now,
		ReadFile: os.ReadFile,
		Version: func() (string, string) {
			return version.Version, version.BuildDate
		},
	})
	if err := tui.Start(model); err != nil {
		log.Fatal(err)
	}
}

type tuiService struct {
	svc *appclient.Service
}

func (t *tuiService) List(ctx context.Context, userPassword string) ([]appclient.ListItem, error) {
	return t.svc.List(ctx, userPassword)
}

func (t *tuiService) Register(ctx context.Context, username, password string) error {
	return t.svc.Register(ctx, username, password)
}

func (t *tuiService) Login(ctx context.Context, username, password string) error {
	return t.svc.Login(ctx, username, password)
}

func (t *tuiService) Get(ctx context.Context, id uuid.UUID, userPassword string) (appclient.PayloadEnvelope, error) {
	return t.svc.Get(ctx, id, userPassword)
}

func (t *tuiService) IsLoggedIn(ctx context.Context) (bool, error) {
	return t.svc.IsLoggedIn(ctx)
}

func (t *tuiService) Logout(ctx context.Context) error {
	return t.svc.Logout(ctx)
}

func (t *tuiService) Add(ctx context.Context, input appclient.AddInput) (uuid.UUID, error) {
	return t.svc.Add(ctx, input)
}

func (t *tuiService) Update(ctx context.Context, input appclient.UpdateInput) (uuid.UUID, error) {
	return t.svc.Update(ctx, input)
}

func (t *tuiService) Delete(ctx context.Context, id uuid.UUID) error {
	return t.svc.Delete(ctx, id)
}

func (t *tuiService) Sync(ctx context.Context) (int, error) {
	return t.svc.Sync(ctx)
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
