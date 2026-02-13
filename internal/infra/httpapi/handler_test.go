package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"gophkeeper/internal/app/auth"
	"gophkeeper/internal/app/vault"
	"gophkeeper/internal/domain/secret"
	"gophkeeper/internal/infra/persistence"
	"gophkeeper/internal/infra/security"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPAPI_RegisterLoginAndItems(t *testing.T) {
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

	token := registerUser(t, server.URL, "alice")
	_ = loginUser(t, server.URL, "alice", "pass")

	itemID := upsertItem(t, server.URL, token, secret.TypeText, []byte("payload"))
	getItem(t, server.URL, token, itemID)
	syncItems(t, server.URL, token)
}

func registerUser(t *testing.T, baseURL, user string) string {
	payload, _ := json.Marshal(map[string]string{"username": user, "password": "pass"})
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, baseURL+"/register", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("register req: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register status: %d", resp.StatusCode)
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("register decode: %v", err)
	}
	return out.Token
}

func loginUser(t *testing.T, baseURL, user, pass string) string {
	payload, _ := json.Marshal(map[string]string{"username": user, "password": pass})
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, baseURL+"/login", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("login req: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status: %d", resp.StatusCode)
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("login decode: %v", err)
	}
	return out.Token
}

func upsertItem(t *testing.T, baseURL, token string, typ secret.Type, payload []byte) string {
	body := map[string]any{
		"type":    string(typ),
		"payload": payload,
		"version": time.Now().UnixNano(),
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, baseURL+"/items", bytes.NewReader(b))
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

func getItem(t *testing.T, baseURL, token, id string) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/items/"+id, nil)
	if err != nil {
		t.Fatalf("get req: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get status: %d", resp.StatusCode)
	}
}

func syncItems(t *testing.T, baseURL, token string) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/items?since=0", nil)
	if err != nil {
		t.Fatalf("sync req: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("sync status: %d", resp.StatusCode)
	}
}
