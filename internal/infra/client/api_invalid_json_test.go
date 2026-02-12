package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIClient_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("{"))
	}))
	defer ts.Close()

	api := NewAPIClient(ts.URL)
	_, err := api.Register(context.Background(), "u", "p")
	if err == nil {
		t.Fatalf("expected error")
	}
}
