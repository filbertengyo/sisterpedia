// Package codec converts library-serialized binary blobs (keys, ciphertexts,
// parameters) to and from standard Base64 (RFC 4648, with padding), which is
// how they travel inside JSON.
package codec

import (
	"encoding/base64"
	"errors"
	"fmt"
)

// DefaultMaxEncodedLen bounds a single Base64 field (16 MiB, the same as the
// server's MAX_BODY_BYTES default).
const DefaultMaxEncodedLen = 16 << 20

var (
	// ErrInvalidBase64 is returned for empty, oversized or malformed Base64.
	ErrInvalidBase64 = errors.New("invalid base64")
	// ErrUnloadable is returned when bytes decode but the library cannot load them.
	ErrUnloadable = errors.New("binary object cannot be loaded")
)

// Encode returns the standard padded Base64 encoding of b.
func Encode(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

// Decode decodes standard padded Base64. Empty input, input longer than
// maxLen (when maxLen > 0) and non-canonical encodings are rejected.
func Decode(s string, maxLen int) ([]byte, error) {
	if s == "" {
		return nil, fmt.Errorf("%w: empty string", ErrInvalidBase64)
	}
	if maxLen > 0 && len(s) > maxLen {
		return nil, fmt.Errorf("%w: %d characters exceeds limit %d", ErrInvalidBase64, len(s), maxLen)
	}
	b, err := base64.StdEncoding.Strict().DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidBase64, err)
	}
	return b, nil
}

// EncodedLen returns the Base64 length of n binary bytes.
func EncodedLen(n int) int {
	return base64.StdEncoding.EncodedLen(n)
}
