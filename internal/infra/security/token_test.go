package security

import (
	"errors"
	"testing"
	"time"
)

func TestHMACTokenSigner_SignVerify(t *testing.T) {
	signer := NewHMACTokenSigner([]byte("secret"))
	okUntil := time.Now().Add(1 * time.Hour)
	token, err := signer.Sign("user-1", okUntil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	userID, _, err := signer.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("unexpected userID: %s", userID)
	}
}

func TestHMACTokenSigner_Expired(t *testing.T) {
	signer := NewHMACTokenSigner([]byte("secret"))
	token, err := signer.Sign("user-1", time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	_, _, err = signer.Verify(token)
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected expired token, got %v", err)
	}
}
