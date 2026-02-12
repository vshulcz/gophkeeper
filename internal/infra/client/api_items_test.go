package client

import (
	"context"
	"encoding/json"
	"gophkeeper/internal/domain/secret"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appclient "gophkeeper/internal/app/client"

	"github.com/google/uuid"
)

func TestAPIClient_GetAndSync(t *testing.T) {
	id := uuid.New()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/items/"+id.String():
			_ = json.NewEncoder(w).Encode(appclient.ItemResponse{ID: id, Type: secret.TypeText, Payload: []byte("p"), Version: 1, UpdatedAt: time.Now()})
		case r.Method == http.MethodGet && r.URL.Path == "/items":
			_ = json.NewEncoder(w).Encode([]appclient.ItemResponse{{ID: id, Type: secret.TypeText, Payload: []byte("p"), Version: 1, UpdatedAt: time.Now()}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	api := NewAPIClient(ts.URL)
	_, err := api.GetItem(context.Background(), "t", id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	_, err = api.SyncItems(context.Background(), "t", 0)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
}
