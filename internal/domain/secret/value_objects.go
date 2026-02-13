package secret

import "errors"

// Value represents a typed secret payload.
type Value interface {
	Type() Type
	Validate() error
	Data() map[string]string
}

// LoginPassword stores login and password pair.
type LoginPassword struct {
	Login    string
	Password string
}

// Type returns secret type.
func (v LoginPassword) Type() Type { return TypeLoginPassword }

// Validate validates the value.
func (v LoginPassword) Validate() error {
	if v.Login == "" || v.Password == "" {
		return errors.New("login and password required")
	}
	return nil
}

// Data returns the data map.
func (v LoginPassword) Data() map[string]string {
	return map[string]string{
		"login":    v.Login,
		"password": v.Password,
	}
}

// Text stores arbitrary text.
type Text struct {
	Text string
}

// Type returns secret type.
func (v Text) Type() Type { return TypeText }

// Validate validates the value.
func (v Text) Validate() error {
	if v.Text == "" {
		return errors.New("text required")
	}
	return nil
}

// Data returns the data map.
func (v Text) Data() map[string]string {
	return map[string]string{"text": v.Text}
}

// Binary stores base64-encoded data.
type Binary struct {
	Base64 string
}

// Type returns secret type.
func (v Binary) Type() Type { return TypeBinary }

// Validate validates the value.
func (v Binary) Validate() error {
	if v.Base64 == "" {
		return errors.New("binary data required")
	}
	return nil
}

// Data returns the data map.
func (v Binary) Data() map[string]string {
	return map[string]string{"file": v.Base64}
}

// Card stores bank card data.
type Card struct {
	Number string
	Expiry string
	Holder string
	CVV    string
}

// Type returns secret type.
func (v Card) Type() Type { return TypeCard }

// Validate validates the value.
func (v Card) Validate() error {
	if v.Number == "" || v.Expiry == "" || v.Holder == "" || v.CVV == "" {
		return errors.New("card fields required")
	}
	return nil
}

// Data returns the data map.
func (v Card) Data() map[string]string {
	return map[string]string{
		"number": v.Number,
		"expiry": v.Expiry,
		"holder": v.Holder,
		"cvv":    v.CVV,
	}
}
