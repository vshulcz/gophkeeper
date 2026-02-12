package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"gophkeeper/internal/app/auth"
	"gophkeeper/internal/app/vault"
	"gophkeeper/internal/infra/persistence"
	"gophkeeper/internal/infra/security"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newRequest(t *testing.T, method, url string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, url, body)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	return req
}

func doRequestStatus(t *testing.T, req *http.Request) int {
	t.Helper()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode
}

func TestHTTPAPI_Errors(t *testing.T) {
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

	// Method not allowed
	req := newRequest(t, http.MethodGet, server.URL+"/register", nil)
	status := doRequestStatus(t, req)
	if status != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405")
	}

	// Health
	req = newRequest(t, http.MethodGet, server.URL+"/health", nil)
	status = doRequestStatus(t, req)
	if status != http.StatusOK {
		t.Fatalf("expected 200")
	}

	// Invalid json
	req = newRequest(t, http.MethodPost, server.URL+"/register", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	status = doRequestStatus(t, req)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400")
	}
	// Missing fields
	req = newRequest(t, http.MethodPost, server.URL+"/login", bytes.NewReader([]byte("{\"username\":\"u\"}")))
	req.Header.Set("Content-Type", "application/json")
	status = doRequestStatus(t, req)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400")
	}

	// Unknown fields
	req = newRequest(t, http.MethodPost, server.URL+"/register", bytes.NewReader([]byte("{\"username\":\"u\",\"password\":\"p\",\"x\":1}")))
	req.Header.Set("Content-Type", "application/json")
	status = doRequestStatus(t, req)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400")
	}

	// Missing token
	req = newRequest(t, http.MethodGet, server.URL+"/items", nil)
	status = doRequestStatus(t, req)
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401")
	}

	// Bad auth scheme
	req = newRequest(t, http.MethodGet, server.URL+"/items", nil)
	req.Header.Set("Authorization", "Token abc")
	status = doRequestStatus(t, req)
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401")
	}

	// Invalid token
	req = newRequest(t, http.MethodGet, server.URL+"/items", nil)
	req.Header.Set("Authorization", "Bearer bad")
	status = doRequestStatus(t, req)
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401")
	}

	// Invalid since
	req = newRequest(t, http.MethodGet, server.URL+"/items?since=bad", nil)
	req.Header.Set("Authorization", "Bearer "+registerUser(t, server.URL, "bob2"))
	status = doRequestStatus(t, req)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400")
	}

	// Invalid item id
	req = newRequest(t, http.MethodGet, server.URL+"/items/bad", nil)
	req.Header.Set("Authorization", "Bearer "+registerUser(t, server.URL, "bob3"))
	status = doRequestStatus(t, req)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400")
	}
	// Method not allowed on item
	req = newRequest(t, http.MethodPost, server.URL+"/items/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+registerUser(t, server.URL, "bob3x"))
	status = doRequestStatus(t, req)
	if status != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405")
	}

	// Method not allowed on items
	req = newRequest(t, http.MethodPut, server.URL+"/items", nil)
	req.Header.Set("Authorization", "Bearer "+registerUser(t, server.URL, "bob4"))
	status = doRequestStatus(t, req)
	if status != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405")
	}

	// Invalid item type
	token := registerUser(t, server.URL, "bob")
	body, _ := json.Marshal(map[string]any{"type": "bad", "payload": []byte("x"), "version": 1})
	req = newRequest(t, http.MethodPost, server.URL+"/items", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	status = doRequestStatus(t, req)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400")
	}

	// Invalid item id in body
	body, _ = json.Marshal(map[string]any{"id": "bad", "type": "text", "payload": []byte("x"), "version": 1})
	req = newRequest(t, http.MethodPost, server.URL+"/items", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	status = doRequestStatus(t, req)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400")
	}
}

type fakeSigner struct{}

func (fakeSigner) Sign(_ string, _ time.Time) (string, error) { return "token", nil }
func (fakeSigner) Verify(_ string) (string, time.Time, error) { return "not-a-uuid", time.Now(), nil }

func TestHTTPAPI_InvalidUserID(t *testing.T) {
	ctx := context.Background()
	db, err := persistence.OpenSQLite(ctx, "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	userRepo := persistence.NewUserRepo(db)
	itemRepo := persistence.NewItemRepo(db)
	hasher := security.NewBcryptHasher(10)
	signer := fakeSigner{}
	authSvc := auth.NewService(userRepo, hasher, signer, time.Now, 24*time.Hour)
	vaultSvc := vault.NewService(itemRepo, time.Now)

	srv := NewServer(authSvc, vaultSvc, signer, 24*time.Hour)
	server := httptest.NewServer(srv.Routes())
	defer server.Close()

	req := newRequest(t, http.MethodGet, server.URL+"/items", nil)
	req.Header.Set("Authorization", "Bearer token")
	status := doRequestStatus(t, req)
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401")
	}
}

func TestHTTPAPI_InternalError(t *testing.T) {
	rr := httptest.NewRecorder()
	writeDomainError(rr, errors.New("boom"))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500")
	}
}

func TestHTTPAPI_ExpiredToken(t *testing.T) {
	ctx := context.Background()
	db, err := persistence.OpenSQLite(ctx, "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	userRepo := persistence.NewUserRepo(db)
	itemRepo := persistence.NewItemRepo(db)
	hasher := security.NewBcryptHasher(10)
	signer := security.NewHMACTokenSigner([]byte("secret"))
	expiredClock := func() time.Time { return time.Now().Add(-2 * time.Hour) }
	authSvc := auth.NewService(userRepo, hasher, signer, expiredClock, time.Hour)
	vaultSvc := vault.NewService(itemRepo, time.Now)

	srv := NewServer(authSvc, vaultSvc, signer, time.Hour)
	server := httptest.NewServer(srv.Routes())
	defer server.Close()

	token := registerUser(t, server.URL, "expired-user")
	req := newRequest(t, http.MethodGet, server.URL+"/items", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	status := doRequestStatus(t, req)
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", status)
	}
}
