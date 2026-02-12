package security

import (
	"encoding/base64"
	"testing"
)

func TestHMACTokenSigner_InvalidExpiry(t *testing.T) {
	signer := NewHMACTokenSigner([]byte("secret"))
	payload := []byte("user|bad")
	sig := signer.sign(payload)
	token := base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(sig)
	_, _, err := signer.Verify(token)
	if err == nil {
		t.Fatalf("expected error")
	}
}
