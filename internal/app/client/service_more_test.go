package client

import (
	"context"
	"errors"
	"gophkeeper/internal/domain/secret"
	"testing"
	"time"
)

type errorStore struct{}

func (errorStore) Load(context.Context) (*StoreState, error) { return nil, errors.New("load") }
func (errorStore) Save(context.Context, *StoreState) error   { return errors.New("save") }

type saveErrorStore struct{}

func (saveErrorStore) Load(context.Context) (*StoreState, error) { return &StoreState{}, nil }
func (saveErrorStore) Save(context.Context, *StoreState) error   { return errors.New("save") }

type errorCrypto struct{}

func (errorCrypto) DeriveKey(string, []byte) ([]byte, error) { return nil, errors.New("derive") }
func (errorCrypto) Encrypt([]byte, []byte) ([]byte, error)   { return nil, errors.New("encrypt") }
func (errorCrypto) Decrypt([]byte, []byte) ([]byte, error)   { return nil, errors.New("decrypt") }
func (errorCrypto) RandomBytes(int) ([]byte, error)          { return nil, errors.New("rand") }

func TestService_LoadSaveErrors(t *testing.T) {
	api := &fakeAPI{registerToken: "t1"}
	svc := NewService(api, errorStore{}, fakeCrypto{}, time.Now)
	if err := svc.Register(context.Background(), "u", "p"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestService_CryptoErrors(t *testing.T) {
	api := &fakeAPI{registerToken: "t1"}
	store := &memoryStore{state: &StoreState{Token: "t", Salt: "c2FsdA=="}} // "salt"
	svc := NewService(api, store, errorCrypto{}, time.Now)
	_, err := svc.Add(context.Background(), AddInput{Value: secret.Text{Text: "x"}, UserPassword: "p"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestService_SaveError(t *testing.T) {
	api := &fakeAPI{registerToken: "t1"}
	svc := NewService(api, saveErrorStore{}, fakeCrypto{}, time.Now)
	if err := svc.Register(context.Background(), "u", "p"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestService_ListNoItems(t *testing.T) {
	api := &fakeAPI{registerToken: "t1"}
	store := &memoryStore{state: &StoreState{Token: "t"}}
	svc := NewService(api, store, fakeCrypto{}, time.Now)
	items, err := svc.List(context.Background(), "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if items != nil {
		t.Fatalf("expected nil items")
	}
}
