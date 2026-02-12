package client

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStore_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")
	store := NewFileStoreWithPath(path)

	state, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	state.Username = "alice"
	if err := store.Save(context.Background(), state); err != nil {
		t.Fatalf("save: %v", err)
	}

	state2, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load2: %v", err)
	}
	if state2.Username != "alice" {
		t.Fatalf("expected username")
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file")
	}
}

func TestNewFileStore(t *testing.T) {
	store, err := NewFileStore()
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if store == nil {
		t.Fatalf("expected store")
	}
}
