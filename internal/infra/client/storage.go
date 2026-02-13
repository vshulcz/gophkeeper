package client

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	app "gophkeeper/internal/app/client"
)

// FileStore persists client state to disk.
type FileStore struct {
	path string
}

// NewFileStore creates a FileStore with default location.
func NewFileStore() (*FileStore, error) {
	path, err := StorePath()
	if err != nil {
		return nil, err
	}
	return &FileStore{path: path}, nil
}

// NewFileStoreWithPath creates a FileStore with a custom path.
func NewFileStoreWithPath(path string) *FileStore {
	return &FileStore{path: path}
}

// Load loads local store from disk or returns empty state.
func (s *FileStore) Load(_ context.Context) (*app.StoreState, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &app.StoreState{}, nil
		}
		return nil, err
	}
	var state app.StoreState
	if err := json.Unmarshal(b, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// Save persists local store to disk.
func (s *FileStore) Save(_ context.Context, state *app.StoreState) error {
	state.UpdatedAt = time.Now()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o600)
}

// StorePath returns the default client store path.
func StorePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gophkeeper", "store.json"), nil
}
