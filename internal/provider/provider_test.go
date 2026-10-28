package provider

import (
	"errors"
	"testing"
)

func TestRedact(t *testing.T) {
	secret := "sk-test-secret-value"
	err := Redact(errors.New("auth failed for "+secret), secret)
	if err == nil || stringsContains(err.Error(), secret) {
		t.Fatalf("secret leaked: %v", err)
	}
	if !stringsContains(err.Error(), "[REDACTED]") {
		t.Fatalf("%v", err)
	}
	if Redact(nil, secret) != nil {
		t.Fatal("nil")
	}
}

func stringsContains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && (s == sub || len(s) > 0 && contains(s, sub))
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
