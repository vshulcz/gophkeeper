package security

import (
	"testing"
	"time"
)

func TestHMACTokenSigner_InvalidSignature(t *testing.T) {
	signer := NewHMACTokenSigner([]byte("secret"))
	token, err := signer.Sign("user-1", nowPlus())
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	// Tamper token
	tampered := token + "x"
	_, _, err = signer.Verify(tampered)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func nowPlus() time.Time {
	return time.Now().Add(1 * time.Hour)
}
