package ports

import "time"

// PasswordHasher hashes and verifies passwords.
type PasswordHasher interface {
	Hash(password string) ([]byte, error)
	Compare(hash []byte, password string) error
}

// TokenSigner signs and verifies auth tokens.
type TokenSigner interface {
	Sign(userID string, expiresAt time.Time) (string, error)
	Verify(token string) (userID string, expiresAt time.Time, err error)
}
