package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrInvalidToken indicates token is malformed or signature mismatch.
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken indicates token is expired.
	ErrExpiredToken = errors.New("expired token")
)

// HMACTokenSigner signs tokens using HMAC-SHA256.
type HMACTokenSigner struct {
	secret []byte
}

// NewHMACTokenSigner creates a signer with the given secret.
func NewHMACTokenSigner(secret []byte) *HMACTokenSigner {
	return &HMACTokenSigner{secret: secret}
}

// Sign creates a signed token for user ID and expiry.
func (s *HMACTokenSigner) Sign(userID string, expiresAt time.Time) (string, error) {
	payload := fmt.Sprintf("%s|%d", userID, expiresAt.Unix())
	sig := s.sign([]byte(payload))
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// Verify validates the token and returns user ID and expiry.
func (s *HMACTokenSigner) Verify(token string) (string, time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", time.Time{}, ErrInvalidToken
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", time.Time{}, ErrInvalidToken
	}
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", time.Time{}, ErrInvalidToken
	}
	if !hmac.Equal(sigBytes, s.sign(payloadBytes)) {
		return "", time.Time{}, ErrInvalidToken
	}

	payload := string(payloadBytes)
	fields := strings.Split(payload, "|")
	if len(fields) != 2 {
		return "", time.Time{}, ErrInvalidToken
	}
	expUnix, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return "", time.Time{}, ErrInvalidToken
	}
	exp := time.Unix(expUnix, 0)
	if time.Now().After(exp) {
		return "", time.Time{}, ErrExpiredToken
	}
	return fields[0], exp, nil
}

func (s *HMACTokenSigner) sign(payload []byte) []byte {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write(payload)
	return mac.Sum(nil)
}
