package client

import "testing"

func TestCryptoService_EncryptDecrypt(t *testing.T) {
	crypto := NewCryptoService()
	salt := []byte("1234567890abcdef")
	key, err := crypto.DeriveKey("password", salt)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	plain := []byte("hello")
	ciphertext, err := crypto.Encrypt(key, plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	got, err := crypto.Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(got) != string(plain) {
		t.Fatalf("unexpected plaintext: %s", got)
	}
}
