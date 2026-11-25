package ckks

import (
	"errors"
	"fmt"
	"math"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	lckks "github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

// ErrValueOutOfRange is returned for values that are not finite or exceed
// MaxValue (they would overflow silently at the lowest level).
var ErrValueOutOfRange = errors.New("value out of range")

// Client performs the user-side operations: encode+encrypt inputs with pk and
// decrypt+decode results with sk. It must only ever run on the user's device
// (browser WASM, or a trusted test/benchmark process).
type Client struct {
	params lckks.Parameters
	ecd    *lckks.Encoder
	enc    *rlwe.Encryptor
	dec    *rlwe.Decryptor
}

// NewClient builds a client. sk may be nil for an encrypt-only client.
func NewClient(params lckks.Parameters, sk *rlwe.SecretKey, pk *rlwe.PublicKey) *Client {
	c := &Client{params: params, ecd: lckks.NewEncoder(params)}
	if pk != nil {
		c.enc = lckks.NewEncryptor(params, pk)
	}
	if sk != nil {
		c.dec = lckks.NewDecryptor(params, sk)
	}
	return c
}

// Params returns the client's parameters.
func (c *Client) Params() lckks.Parameters { return c.params }

// CheckRange validates that every value is finite and |v| <= MaxValue.
func CheckRange(params lckks.Parameters, values []float64) error {
	limit := MaxValue(params)
	for i, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > limit {
			return fmt.Errorf("%w: slot %d = %g (limit %g)", ErrValueOutOfRange, i, v, limit)
		}
	}
	return nil
}

// Encode encodes values into a plaintext at the max level and default scale.
func (c *Client) Encode(values []float64) (*rlwe.Plaintext, error) {
	if len(values) == 0 || len(values) > c.params.MaxSlots() {
		return nil, fmt.Errorf("%w: %d values for %d slots", ErrValueOutOfRange, len(values), c.params.MaxSlots())
	}
	pt := lckks.NewPlaintext(c.params, c.params.MaxLevel())
	if err := c.ecd.Encode(values, pt); err != nil {
		return nil, err
	}
	return pt, nil
}

// EncryptPlaintext encrypts an encoded plaintext with the public key.
func (c *Client) EncryptPlaintext(pt *rlwe.Plaintext) (*rlwe.Ciphertext, error) {
	if c.enc == nil {
		return nil, errors.New("client has no public key")
	}
	return c.enc.EncryptNew(pt)
}

// EncryptVector range-checks, encodes and encrypts values (slot i = values[i]).
func (c *Client) EncryptVector(values []float64) (*rlwe.Ciphertext, error) {
	if err := CheckRange(c.params, values); err != nil {
		return nil, err
	}
	return c.EncryptUnchecked(values)
}

// EncryptUnchecked encodes and encrypts without the overflow guard. It exists
// only to demonstrate silent overflow in tests (U11).
func (c *Client) EncryptUnchecked(values []float64) (*rlwe.Ciphertext, error) {
	pt, err := c.Encode(values)
	if err != nil {
		return nil, err
	}
	return c.EncryptPlaintext(pt)
}

// EncryptScalar encrypts a single value in slot 0.
func (c *Client) EncryptScalar(x float64) (*rlwe.Ciphertext, error) {
	return c.EncryptVector([]float64{x})
}

// DecryptPlaintext decrypts a ciphertext with the secret key.
func (c *Client) DecryptPlaintext(ct *rlwe.Ciphertext) (*rlwe.Plaintext, error) {
	if c.dec == nil {
		return nil, errors.New("client has no secret key")
	}
	return c.dec.DecryptNew(ct), nil
}

// Decode decodes the first n slots of a plaintext (real parts).
func (c *Client) Decode(pt *rlwe.Plaintext, n int) ([]float64, error) {
	if n <= 0 || n > c.params.MaxSlots() {
		return nil, fmt.Errorf("%w: cannot decode %d slots", ErrValueOutOfRange, n)
	}
	values := make([]float64, c.params.MaxSlots())
	if err := c.ecd.Decode(pt, values); err != nil {
		return nil, err
	}
	return values[:n], nil
}

// DecryptVector decrypts and decodes the first n slots (raw CKKS values).
func (c *Client) DecryptVector(ct *rlwe.Ciphertext, n int) ([]float64, error) {
	pt, err := c.DecryptPlaintext(ct)
	if err != nil {
		return nil, err
	}
	return c.Decode(pt, n)
}

// DecryptScalar decrypts slot 0 (raw, unrounded).
func (c *Client) DecryptScalar(ct *rlwe.Ciphertext) (float64, error) {
	v, err := c.DecryptVector(ct, 1)
	if err != nil {
		return 0, err
	}
	return v[0], nil
}
