package client

import (
	"context"

	"github.com/google/uuid"
)

// API defines server communication for client use cases.
type API interface {
	Register(ctx context.Context, username, password string) (string, error)
	Login(ctx context.Context, username, password string) (string, error)
	UpsertItem(ctx context.Context, token string, req ItemUpsertRequest) (ItemResponse, error)
	GetItem(ctx context.Context, token string, id uuid.UUID) (ItemResponse, error)
	SyncItems(ctx context.Context, token string, since int64) ([]ItemResponse, error)
	DeleteItem(ctx context.Context, token string, id uuid.UUID, version int64) (ItemResponse, error)
}

// Store persists local client state.
type Store interface {
	Load(ctx context.Context) (*StoreState, error)
	Save(ctx context.Context, state *StoreState) error
}

// Crypto provides encryption services for client payloads.
type Crypto interface {
	DeriveKey(password string, salt []byte) ([]byte, error)
	Encrypt(key, plaintext []byte) ([]byte, error)
	Decrypt(key, ciphertext []byte) ([]byte, error)
	RandomBytes(n int) ([]byte, error)
}
