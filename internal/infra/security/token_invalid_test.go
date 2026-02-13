package security

import "testing"

func TestHMACTokenSigner_InvalidFormat(t *testing.T) {
	signer := NewHMACTokenSigner([]byte("secret"))
	_, _, err := signer.Verify("badtoken")
	if err == nil {
		t.Fatalf("expected error")
	}
}
