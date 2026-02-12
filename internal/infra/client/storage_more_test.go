package client

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	appclient "gophkeeper/internal/app/client"
)

func TestFileStore_LoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	store := NewFileStoreWithPath(path)
	_, err := store.Load(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestFileStore_SaveBadPath(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	path := filepath.Join(file, "store.json")
	store := NewFileStoreWithPath(path)
	state := &appclient.StoreState{}
	if err := store.Save(context.Background(), state); err == nil {
		t.Fatalf("expected error")
	}
}
