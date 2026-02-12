package security

import "testing"

func TestHMACTokenSigner_InvalidBase64(t *testing.T) {
	signer := NewHMACTokenSigner([]byte("secret"))
	_, _, err := signer.Verify("%%%")
	if err == nil {
		t.Fatalf("expected error")
	}
}
