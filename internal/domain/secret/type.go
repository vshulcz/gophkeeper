// Package secret contains secret aggregate types.
package secret

import (
	"gophkeeper/internal/domain"
	"strings"
)

// Type defines a supported secret data type.
type Type string

const (
	// TypeLoginPassword stores login and password pairs.
	TypeLoginPassword Type = "login_password"
	// TypeText stores arbitrary text data.
	TypeText Type = "text"
	// TypeBinary stores arbitrary binary data (base64 encoded).
	TypeBinary Type = "binary"
	// TypeCard stores bank card data.
	TypeCard Type = "card"
)

// IsValid reports whether the secret type is supported.
func (t Type) IsValid() bool {
	switch t {
	case TypeLoginPassword, TypeText, TypeBinary, TypeCard:
		return true
	default:
		return false
	}
}

// ParseType validates and returns the secret type.
func ParseType(value string) (Type, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "login":
		normalized = string(TypeLoginPassword)
	case "loginpassword":
		normalized = string(TypeLoginPassword)
	case "login-password":
		normalized = string(TypeLoginPassword)
	}
	t := Type(normalized)
	if !t.IsValid() {
		return "", domain.ErrInvalidItemType
	}
	return t, nil
}
