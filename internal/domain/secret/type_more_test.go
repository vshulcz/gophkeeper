package secret

import "testing"

func TestType_IsValid(t *testing.T) {
	if !TypeText.IsValid() {
		t.Fatalf("expected valid")
	}
	if Type("bad").IsValid() {
		t.Fatalf("expected invalid")
	}
}
