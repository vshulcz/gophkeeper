package client

import (
	"context"
	"errors"
	"gophkeeper/internal/domain/secret"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestService_AddErrors(t *testing.T) {
	api := &fakeAPI{registerToken: "t1"}
	store := &memoryStore{}
	crypto := fakeCrypto{}
	svc := NewService(api, store, crypto, time.Now)

	// Not logged in
	_, err := svc.Add(context.Background(), AddInput{Value: secret.Text{Text: "x"}, UserPassword: "p"})
	if err == nil {
		t.Fatalf("expected error")
	}

	_ = svc.Register(context.Background(), "alice", "pass")
	_, err = svc.Add(context.Background(), AddInput{Value: nil, UserPassword: "p"})
	if err == nil {
		t.Fatalf("expected error")
	}
	_, err = svc.Add(context.Background(), AddInput{Value: secret.Text{}, UserPassword: "p"})
	if err == nil {
		t.Fatalf("expected error")
	}
	_, err = svc.Add(context.Background(), AddInput{Value: secret.Text{Text: "x"}, UserPassword: ""})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestService_GetErrors(t *testing.T) {
	api := &fakeAPI{registerToken: "t1"}
	store := &memoryStore{}
	crypto := fakeCrypto{}
	svc := NewService(api, store, crypto, time.Now)

	_ = svc.Register(context.Background(), "alice", "pass")
	_, err := svc.Get(context.Background(), fakeID(), "")
	if err == nil {
		t.Fatalf("expected error")
	}

	_, err = svc.Get(context.Background(), fakeID(), "pass")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestService_ListDecryptError(t *testing.T) {
	api := &fakeAPI{registerToken: "t1"}
	store := &memoryStore{}
	crypto := badCrypto{}
	svc := NewService(api, store, crypto, time.Now)

	_ = svc.Register(context.Background(), "alice", "pass")
	_, _ = svc.Add(context.Background(), AddInput{Value: secret.Text{Text: "x"}, Meta: map[string]string{}, UserPassword: "pass"})
	items, err := svc.List(context.Background(), "pass")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 || items[0].Err == nil {
		t.Fatalf("expected decrypt error")
	}
}

func TestService_GetDecryptError(t *testing.T) {
	api := &fakeAPI{registerToken: "t1"}
	store := &memoryStore{}
	crypto := fakeCrypto{}
	svc := NewService(api, store, crypto, time.Now)

	_ = svc.Register(context.Background(), "alice", "pass")
	value := secret.Text{Text: "hello"}
	id, _ := svc.Add(context.Background(), AddInput{Value: value, UserPassword: "pass"})

	bad := NewService(api, store, badCrypto{}, time.Now)
	_, err := bad.Get(context.Background(), id, "pass")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestService_SyncNotLoggedIn(t *testing.T) {
	api := &fakeAPI{registerToken: "t1"}
	store := &memoryStore{}
	crypto := fakeCrypto{}
	svc := NewService(api, store, crypto, time.Now)

	_, err := svc.Sync(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
}

type badCrypto struct{}

func (badCrypto) DeriveKey(_ string, _ []byte) ([]byte, error) { return []byte("key"), nil }
func (badCrypto) Encrypt(_, plaintext []byte) ([]byte, error)  { return plaintext, nil }
func (badCrypto) Decrypt(_, _ []byte) ([]byte, error)          { return nil, errors.New("decrypt") }
func (badCrypto) RandomBytes(n int) ([]byte, error)            { return make([]byte, n), nil }

func fakeID() uuid.UUID {
	id, _ := uuid.Parse("00000000-0000-0000-0000-000000000001")
	return id
}
