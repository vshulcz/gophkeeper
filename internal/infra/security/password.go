package security

import "golang.org/x/crypto/bcrypt"

// BcryptHasher hashes and verifies passwords using bcrypt.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher creates a BcryptHasher with the provided cost.
func NewBcryptHasher(cost int) *BcryptHasher {
	return &BcryptHasher{cost: cost}
}

// Hash hashes a password string.
func (h *BcryptHasher) Hash(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), h.cost)
}

// Compare verifies a password against a hash.
func (h *BcryptHasher) Compare(hash []byte, password string) error {
	return bcrypt.CompareHashAndPassword(hash, []byte(password))
}
