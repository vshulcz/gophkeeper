package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIClient_ErrorNoJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("oops"))
	}))
	defer ts.Close()

	api := NewAPIClient(ts.URL)
	_, err := api.Login(context.Background(), "u", "p")
	if err == nil {
		t.Fatalf("expected error")
	}
}
