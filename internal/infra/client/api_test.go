package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIClient_Register(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/register" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "t"})
	}))
	defer ts.Close()

	api := NewAPIClient(ts.URL)
	tok, err := api.Register(context.Background(), "u", "p")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if tok != "t" {
		t.Fatalf("unexpected token")
	}
}

func TestAPIClient_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "fail"})
	}))
	defer ts.Close()

	api := NewAPIClient(ts.URL)
	_, err := api.Login(context.Background(), "u", "p")
	if err == nil {
		t.Fatalf("expected error")
	}
}
