package hashid

import (
	"strings"
	"testing"
)

func TestEncode_NonEmpty(t *testing.T) {
	// Encode(1) returns non-empty string
	if s := Encode(1); s == "" {
		t.Errorf("Encode(1) returned empty string, want non-empty")
	}
	// Encode(0) returns non-empty string
	if s := Encode(0); s == "" {
		t.Errorf("Encode(0) returned empty string, want non-empty")
	}
}

func TestEncodeDecode_RoundTrip(t *testing.T) {
	// Decode(Encode(n)) == n round-trip test
	values := []int{0, 1, 42, 100, 99999}
	for _, n := range values {
		encoded := Encode(n)
		if encoded == "" {
			t.Fatalf("Encode(%d) returned empty string", n)
		}
		got, err := Decode(encoded)
		if err != nil {
			t.Fatalf("Decode(%q) returned error: %v, want nil", encoded, err)
		}
		if got != n {
			t.Errorf("Decode(Encode(%d)) = %d, want %d (encoded=%q)", n, got, n, encoded)
		}
	}
}

func TestEncode_DifferentInputsProduceDifferentOutputs(t *testing.T) {
	// Different inputs should not produce the same output (avoid hashid degradation)
	if Encode(1) == Encode(2) {
		t.Errorf("Encode(1) == Encode(2), want different outputs")
	}
	if Encode(100) == Encode(99999) {
		t.Errorf("Encode(100) == Encode(99999), want different outputs")
	}
}

func TestDecode_EmptyStringReturnsError(t *testing.T) {
	if _, err := Decode(""); err == nil {
		t.Errorf("Decode(\"\") returned nil error, want non-nil")
	}
}

func TestDecode_InvalidStringReturnsError(t *testing.T) {
	// Completely invalid characters
	if _, err := Decode("invalid!@#"); err == nil {
		t.Errorf("Decode(\"invalid!@#\") returned nil error, want non-nil")
	}
	// Spaces only
	if _, err := Decode("    "); err == nil {
		t.Errorf("Decode(\"    \") returned nil error, want non-nil")
	}
	// Single invalid character
	if _, err := Decode("!"); err == nil {
		t.Errorf("Decode(\"!\") returned nil error, want non-nil")
	}
}

func TestDecode_InvalidHashidsStringReturnsError(t *testing.T) {
	// A valid hashid string should not contain non-alphanumeric characters.
	// Strings containing separators should fail (hashids does not produce such output).
	if _, err := Decode(strings.Repeat("a", 5) + " "); err == nil {
		t.Errorf("Decode(trailing space) returned nil error, want non-nil")
	}
}
