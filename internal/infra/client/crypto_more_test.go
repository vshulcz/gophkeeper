package client

import "testing"

func TestCryptoService_RandomBytes(t *testing.T) {
	crypto := NewCryptoService()
	b, err := crypto.RandomBytes(16)
	if err != nil {
		t.Fatalf("rand: %v", err)
	}
	if len(b) != 16 {
		t.Fatalf("expected 16 bytes")
	}
}
