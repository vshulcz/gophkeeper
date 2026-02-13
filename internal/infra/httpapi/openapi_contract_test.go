//go:build integration
// +build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"gophkeeper/internal/app/auth"
	"gophkeeper/internal/app/vault"
	"gophkeeper/internal/infra/httpapi/dto"
	"gophkeeper/internal/infra/persistence"
	"gophkeeper/internal/infra/security"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

func TestOpenAPI_SpecValid(t *testing.T) {
	doc := loadOpenAPIDoc(t)
	if err := doc.Validate(context.Background()); err != nil {
		t.Fatalf("openapi invalid: %v", err)
	}
}

func TestOpenAPI_Contract(t *testing.T) {
	doc := loadOpenAPIDoc(t)

	ts := newHTTPServer(t)
	defer ts.Close()

	doc.Servers = openapi3.Servers{{URL: ts.URL}}
	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		t.Fatalf("router: %v", err)
	}

	client := ts.Client()
	base := ts.URL

	// /health
	req := newJSONRequest(t, http.MethodGet, base+"/health", nil, "")
	resp, body := doAndValidate(t, doc, router, client, req)
	if resp.StatusCode != http.StatusOK || string(body) != "ok" {
		t.Fatalf("health unexpected: %d %q", resp.StatusCode, string(body))
	}

	// /register
	creds := dto.CredentialsRequest{Username: "alice", Password: "secret"}
	req = newJSONRequest(t, http.MethodPost, base+"/register", creds, "")
	resp, body = doAndValidate(t, doc, router, client, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register status: %d", resp.StatusCode)
	}
	var reg dto.TokenResponse
	if err := json.Unmarshal(body, &reg); err != nil || reg.Token == "" {
		t.Fatalf("register token: %v", err)
	}

	// /login
	req = newJSONRequest(t, http.MethodPost, base+"/login", creds, "")
	resp, body = doAndValidate(t, doc, router, client, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status: %d", resp.StatusCode)
	}
	var login dto.TokenResponse
	if err := json.Unmarshal(body, &login); err != nil || login.Token == "" {
		t.Fatalf("login token: %v", err)
	}

	// /items (POST)
	itemReq := dto.ItemUpsertRequest{
		Type:    "text",
		Payload: []byte("payload"),
		Version: 1,
	}
	req = newJSONRequest(t, http.MethodPost, base+"/items", itemReq, login.Token)
	resp, body = doAndValidate(t, doc, router, client, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("items post status: %d", resp.StatusCode)
	}
	var item dto.ItemResponse
	if err := json.Unmarshal(body, &item); err != nil || item.ID == "" {
		t.Fatalf("items post response: %v", err)
	}

	// /items (GET)
	req = newJSONRequest(t, http.MethodGet, base+"/items?since=0", nil, login.Token)
	resp, body = doAndValidate(t, doc, router, client, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("items get status: %d", resp.StatusCode)
	}
	var items []dto.ItemResponse
	if err := json.Unmarshal(body, &items); err != nil || len(items) == 0 {
		t.Fatalf("items get response: %v", err)
	}

	// /items/{id} (GET)
	req = newJSONRequest(t, http.MethodGet, base+"/items/"+item.ID, nil, login.Token)
	resp, body = doAndValidate(t, doc, router, client, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("item get status: %d", resp.StatusCode)
	}

	// /items/{id} (DELETE)
	req = newJSONRequest(t, http.MethodDelete, base+"/items/"+item.ID+"?version=2", nil, login.Token)
	resp, body = doAndValidate(t, doc, router, client, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("item delete status: %d", resp.StatusCode)
	}
}

func loadOpenAPIDoc(t *testing.T) *openapi3.T {
	t.Helper()
	root := repoRoot(t)
	path := filepath.Join(root, "docs", "openapi.yaml")
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		t.Fatalf("load openapi: %v", err)
	}
	return doc
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found")
		}
		dir = parent
	}
}

func newHTTPServer(t *testing.T) *httptest.Server {
	t.Helper()
	ctx := context.Background()
	dsn := "file:openapi_" + sanitizeName(t.Name()) + "?mode=memory&cache=shared"
	db, err := persistence.OpenSQLite(ctx, dsn)
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
	return httptest.NewServer(srv.Routes())
}

func sanitizeName(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
		switch r {
		case '/', ' ', ':':
			out = append(out, '_')
		default:
			out = append(out, r)
		}
	}
	return string(out)
}

func newJSONRequest(t *testing.T, method, url string, body any, token string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req, err := http.NewRequest(method, url, &buf)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func doAndValidate(
	t *testing.T,
	doc *openapi3.T,
	router routers.Router,
	client *http.Client,
	req *http.Request,
) (*http.Response, []byte) {
	t.Helper()
	route, pathParams, err := router.FindRoute(req)
	if err != nil {
		t.Fatalf("route: %v", err)
	}

	reqInput := &openapi3filter.RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
		Options: &openapi3filter.Options{
			AuthenticationFunc: func(context.Context, *openapi3filter.AuthenticationInput) error {
				return nil
			},
		},
	}
	if err := openapi3filter.ValidateRequest(context.Background(), reqInput); err != nil {
		t.Fatalf("request validation: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	respInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: reqInput,
		Status:                 resp.StatusCode,
		Header:                 resp.Header,
	}
	respInput.SetBodyBytes(body)
	if err := openapi3filter.ValidateResponse(context.Background(), respInput); err != nil {
		t.Fatalf("response validation: %v", err)
	}

	return resp, body
}
