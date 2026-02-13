//go:build integration
// +build integration

package client

import (
	"context"
	"gophkeeper/internal/app/auth"
	"gophkeeper/internal/app/vault"
	"gophkeeper/internal/domain/secret"
	"gophkeeper/internal/infra/httpapi"
	"gophkeeper/internal/infra/persistence"
	"gophkeeper/internal/infra/security"
	"strings"
	"testing"
	"time"

	appclient "gophkeeper/internal/app/client"
)

type memoryStore struct {
	state *appclient.StoreState
}

func (m *memoryStore) Load(_ context.Context) (*appclient.StoreState, error) {
	if m.state == nil {
		m.state = &appclient.StoreState{}
	}
	return m.state, nil
}

func (m *memoryStore) Save(_ context.Context, state *appclient.StoreState) error {
	m.state = state
	return nil
}

func TestInfraClient_E2E(t *testing.T) {
	ctx := context.Background()
	db, err := persistence.OpenSQLite(ctx, "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("db: %v", err)
	}

	userRepo := persistence.NewUserRepo(db)
	itemRepo := persistence.NewItemRepo(db)
	hasher := security.NewBcryptHasher(10)
	signer := security.NewHMACTokenSigner([]byte("secret"))
	authSvc := auth.NewService(userRepo, hasher, signer, time.Now, 24*time.Hour)
	vaultSvc := vault.NewService(itemRepo, time.Now)
	httpSrv := httpapi.NewServer(authSvc, vaultSvc, signer, 24*time.Hour)

	ts := newTestServer(httpSrv.Routes())
	defer ts.Close()

	api := NewAPIClient(ts.URL)
	store := &memoryStore{}
	crypto := NewCryptoService()
	svc := appclient.NewService(api, store, crypto, time.Now)

	if err := svc.Register(ctx, "bob", "pass"); err != nil {
		t.Fatalf("register: %v", err)
	}

	value := secret.Text{Text: "hello"}
	id, err := svc.Add(ctx, appclient.AddInput{Value: value, Meta: map[string]string{"site": "ex"}, UserPassword: "pass"})
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	items, err := svc.List(ctx, "pass")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item")
	}

	_, err = svc.Update(ctx, appclient.UpdateInput{ID: id, Value: secret.Text{Text: "updated"}, Meta: map[string]string{}, UserPassword: "pass"})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	if err := svc.Delete(ctx, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.Sync(ctx); err != nil {
		t.Fatalf("sync: %v", err)
	}
	items, err = svc.List(ctx, "pass")
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items after delete")
	}
}

func TestInfraClient_E2E_LargePayload(t *testing.T) {
	ctx := context.Background()
	db, err := persistence.OpenSQLite(ctx, "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("db: %v", err)
	}

	userRepo := persistence.NewUserRepo(db)
	itemRepo := persistence.NewItemRepo(db)
	hasher := security.NewBcryptHasher(10)
	signer := security.NewHMACTokenSigner([]byte("secret"))
	authSvc := auth.NewService(userRepo, hasher, signer, time.Now, 24*time.Hour)
	vaultSvc := vault.NewService(itemRepo, time.Now)
	httpSrv := httpapi.NewServer(authSvc, vaultSvc, signer, 24*time.Hour)

	ts := newTestServer(httpSrv.Routes())
	defer ts.Close()

	api := NewAPIClient(ts.URL)
	store := &memoryStore{}
	crypto := NewCryptoService()
	svc := appclient.NewService(api, store, crypto, time.Now)

	if err := svc.Register(ctx, "large", "pass"); err != nil {
		t.Fatalf("register: %v", err)
	}

	largeText := strings.Repeat("a", 128*1024)
	value := secret.Text{Text: largeText}
	id, err := svc.Add(ctx, appclient.AddInput{Value: value, Meta: map[string]string{"site": "large"}, UserPassword: "pass"})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	env, err := svc.Get(ctx, id, "pass")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got := env.Data["text"]; got != largeText {
		t.Fatalf("unexpected payload length: %d", len(got))
	}
}
