package codec

import (
	"errors"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	in := []byte{0, 1, 2, 250, 251, 252, 253, 254, 255}
	s := Encode(in)
	out, err := Decode(s, 0)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(in) {
		t.Fatalf("round trip mismatch: %v != %v", out, in)
	}
	if EncodedLen(len(in)) != len(s) {
		t.Fatalf("EncodedLen = %d, want %d", EncodedLen(len(in)), len(s))
	}
}

func TestDecodeRejects(t *testing.T) {
	cases := map[string]string{
		"empty":       "",
		"bad chars":   "@@@@",
		"no padding":  "QQ",
		"url-safe":    "-_-_",
		"bad padding": "QQ=A",
	}
	for name, in := range cases {
		if _, err := Decode(in, 0); !errors.Is(err, ErrInvalidBase64) {
			t.Errorf("%s: want ErrInvalidBase64, got %v", name, err)
		}
	}
	if _, err := Decode("QUJD", 3); !errors.Is(err, ErrInvalidBase64) {
		t.Errorf("oversized input not rejected: %v", err)
	}
}
