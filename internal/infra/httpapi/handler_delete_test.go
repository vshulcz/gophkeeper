package httpapi

import (
	"context"
	"encoding/json"
	"gophkeeper/internal/app/auth"
	"gophkeeper/internal/app/vault"
	"gophkeeper/internal/domain/secret"
	"gophkeeper/internal/infra/persistence"
	"gophkeeper/internal/infra/security"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestHTTPAPI_DeleteItem(t *testing.T) {
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

	srv := NewServer(authSvc, vaultSvc, signer, 24*time.Hour)
	server := httptest.NewServer(srv.Routes())
	defer server.Close()

	token := registerUser(t, server.URL, "del-user")
	itemID := upsertItem(t, server.URL, token, secret.TypeText, []byte("payload"))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodDelete, server.URL+"/items/"+itemID+"?version="+strconv.FormatInt(time.Now().UnixNano(), 10), nil)
	if err != nil {
		t.Fatalf("delete req: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete status: %d", resp.StatusCode)
	}
	var out struct {
		Deleted bool `json:"deleted"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("delete decode: %v", err)
	}
	if !out.Deleted {
		t.Fatalf("expected deleted=true")
	}

	req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/items/"+itemID, nil)
	if err != nil {
		t.Fatalf("get req: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", resp.StatusCode)
	}

	req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/items?since=0", nil)
	if err != nil {
		t.Fatalf("sync req: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("sync status: %d", resp.StatusCode)
	}
	var items []struct {
		ID      string `json:"id"`
		Deleted bool   `json:"deleted"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("sync decode: %v", err)
	}
	if len(items) != 1 || !items[0].Deleted {
		t.Fatalf("expected deleted item in sync")
	}
}
