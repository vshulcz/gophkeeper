package client

import (
	"encoding/base64"
	"gophkeeper/internal/domain/secret"
	"testing"

	"github.com/google/uuid"
)

type fixedCrypto struct{}

func (fixedCrypto) DeriveKey(string, []byte) ([]byte, error) { return nil, nil }
func (fixedCrypto) Encrypt([]byte, []byte) ([]byte, error)   { return nil, nil }
func (fixedCrypto) Decrypt([]byte, []byte) ([]byte, error)   { return nil, nil }
func (fixedCrypto) RandomBytes(n int) ([]byte, error)        { return make([]byte, n), nil }

func TestStoreState_EnsureSalt(t *testing.T) {
	state := &StoreState{}
	salt, err := state.EnsureSalt(fixedCrypto{})
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if len(salt) != 16 {
		t.Fatalf("expected 16 bytes")
	}
	decoded, err := base64.StdEncoding.DecodeString(state.Salt)
	if err != nil || len(decoded) != 16 {
		t.Fatalf("expected base64 salt")
	}

	state2 := &StoreState{Salt: state.Salt}
	salt2, err := state2.EnsureSalt(fixedCrypto{})
	if err != nil {
		t.Fatalf("ensure2: %v", err)
	}
	if len(salt2) != 16 {
		t.Fatalf("expected 16 bytes")
	}
}

func TestStoreState_UpdateOrInsert(t *testing.T) {
	id := uuid.New()
	state := &StoreState{}
	state.UpdateOrInsertItem(LocalItem{ID: id, Type: secret.TypeText, Version: 1})
	state.UpdateOrInsertItem(LocalItem{ID: id, Type: secret.TypeText, Version: 2})
	if len(state.Items) != 1 || state.Items[0].Version != 2 {
		t.Fatalf("expected update")
	}
}

func TestStoreState_RemoveItem(t *testing.T) {
	id := uuid.New()
	state := &StoreState{Items: []LocalItem{{ID: id, Type: secret.TypeText, Version: 1}}}
	state.RemoveItem(id)
	if len(state.Items) != 0 {
		t.Fatalf("expected removal")
	}
}

func TestStoreState_RequireToken(t *testing.T) {
	state := &StoreState{}
	if err := state.RequireToken(); err == nil {
		t.Fatalf("expected error")
	}
	state.Token = "t"
	if err := state.RequireToken(); err != nil {
		t.Fatalf("unexpected error")
	}
}

func TestPayloadEncoding(t *testing.T) {
	data := []byte("hello")
	enc := EncodePayload(data)
	dec, err := DecodePayload(enc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(dec) != "hello" {
		t.Fatalf("unexpected decode")
	}
}
