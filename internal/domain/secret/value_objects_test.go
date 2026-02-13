package secret

import "testing"

func TestValueObjects_Validate(t *testing.T) {
	cases := []struct {
		name  string
		value Value
		ok    bool
	}{
		{"login ok", LoginPassword{Login: "u", Password: "p"}, true},
		{"login bad", LoginPassword{}, false},
		{"text ok", Text{Text: "hi"}, true},
		{"text bad", Text{}, false},
		{"bin ok", Binary{Base64: "Zm9v"}, true},
		{"bin bad", Binary{}, false},
		{"card ok", Card{Number: "1", Expiry: "12/30", Holder: "A", CVV: "123"}, true},
		{"card bad", Card{}, false},
	}
	for _, c := range cases {
		err := c.value.Validate()
		if c.ok && err != nil {
			t.Fatalf("%s: expected ok, got %v", c.name, err)
		}
		if !c.ok && err == nil {
			t.Fatalf("%s: expected error", c.name)
		}
	}
}

func TestValueObjects_Data(t *testing.T) {
	if (LoginPassword{Login: "u", Password: "p"}).Data()["login"] != "u" {
		t.Fatalf("login data")
	}
	if (Text{Text: "t"}).Data()["text"] != "t" {
		t.Fatalf("text data")
	}
	if (Binary{Base64: "b"}).Data()["file"] != "b" {
		t.Fatalf("binary data")
	}
	if (Card{Number: "1", Expiry: "2", Holder: "3", CVV: "4"}).Data()["number"] != "1" {
		t.Fatalf("card data")
	}
}

func TestValueObjects_Type(t *testing.T) {
	if (LoginPassword{}).Type() != TypeLoginPassword {
		t.Fatalf("type login")
	}
	if (Text{}).Type() != TypeText {
		t.Fatalf("type text")
	}
	if (Binary{}).Type() != TypeBinary {
		t.Fatalf("type binary")
	}
	if (Card{}).Type() != TypeCard {
		t.Fatalf("type card")
	}
}
