package security

import "testing"

func TestBcryptHasher_HashCompare(t *testing.T) {
	h := NewBcryptHasher(10)
	hash, err := h.Hash("pass")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := h.Compare(hash, "pass"); err != nil {
		t.Fatalf("compare: %v", err)
	}
	if err := h.Compare(hash, "wrong"); err == nil {
		t.Fatalf("expected error")
	}
}
