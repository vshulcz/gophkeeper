package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"gophkeeper/internal/app/auth"
	"gophkeeper/internal/app/vault"
	"gophkeeper/internal/infra/persistence"
	"gophkeeper/internal/infra/security"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPAPI_ConflictOnStaleUpdate(t *testing.T) {
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

	token := registerUser(t, server.URL, "conflict-user")

	// Create item with version 10.
	itemID := upsertItemWithVersion(t, server.URL, token, "text", []byte("payload"), 10)

	// Try to update with stale version.
	body, _ := json.Marshal(map[string]any{
		"id":      itemID,
		"type":    "text",
		"payload": []byte("payload2"),
		"version": 5,
	})
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL+"/items", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("upsert req: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upsert stale: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}

	// Delete with stale version should also conflict.
	req, err = http.NewRequestWithContext(context.Background(), http.MethodDelete, server.URL+"/items/"+itemID+"?version=5", nil)
	if err != nil {
		t.Fatalf("delete req: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete stale: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 on delete, got %d", resp.StatusCode)
	}
}

func upsertItemWithVersion(t *testing.T, baseURL, token, typ string, payload []byte, version int64) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"type":    typ,
		"payload": payload,
		"version": version,
	})
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, baseURL+"/items", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("upsert req: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upsert status: %d", resp.StatusCode)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("upsert decode: %v", err)
	}
	return out.ID
}
