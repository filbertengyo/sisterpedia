package ckks

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/filbertengyo/sisterpedia/pkg/codec"
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/ring"
	"github.com/tuneinsight/lattigo/v6/ring/ringqp"
	lckks "github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

var (
	// ErrContextMismatch is returned when an object loads but does not fit the
	// account's parameters (wrong level, scale, dimensions or NTT form).
	ErrContextMismatch = errors.New("object does not match the parameters")
	// ErrPrivateKeyMaterial is returned when a payload field carries a secret key.
	ErrPrivateKeyMaterial = errors.New("payload contains private key material")
)

// Lattigo's deserializers allocate slices whose length is read from the
// input, so feeding them untrusted bytes can trigger huge allocations. Before
// calling UnmarshalBinary we therefore check that a blob has exactly the
// binary structure (all length prefixes and flags) of an object of the
// expected shape for the parameters. The structure is learned from two
// serializations of a template whose coefficients are filled with two
// different patterns: bytes that differ are coefficient data, bytes that are
// equal are structure.
type shape struct {
	size     int
	tmpl     []byte
	ranges   [][2]int // [start, end) byte ranges that must equal tmpl
	skipFrom int      // metadata region not compared (ciphertexts only)
	skipTo   int
}

func (s *shape) matches(b []byte) bool {
	if s == nil || len(b) != s.size {
		return false
	}
	for _, r := range s.ranges {
		start, end := r[0], r[1]
		if start < s.skipTo && end > s.skipFrom { // overlap with skipped region
			if start < s.skipFrom && !bytes.Equal(b[start:s.skipFrom], s.tmpl[start:s.skipFrom]) {
				return false
			}
			if end > s.skipTo && !bytes.Equal(b[s.skipTo:end], s.tmpl[s.skipTo:end]) {
				return false
			}
			continue
		}
		if !bytes.Equal(b[start:end], s.tmpl[start:end]) {
			return false
		}
	}
	return true
}

func buildShape(a, b []byte, skipFrom, skipTo int) (*shape, error) {
	if len(a) != len(b) {
		return nil, errors.New("template serializations differ in length")
	}
	s := &shape{size: len(a), tmpl: a, skipFrom: skipFrom, skipTo: skipTo}
	start := -1
	for i := range a {
		if a[i] == b[i] {
			if start < 0 {
				start = i
			}
		} else if start >= 0 {
			s.ranges = append(s.ranges, [2]int{start, i})
			start = -1
		}
	}
	if start >= 0 {
		s.ranges = append(s.ranges, [2]int{start, len(a)})
	}
	return s, nil
}

func fillPoly(p ring.Poly, v uint64) {
	for i := range p.Coeffs {
		for j := range p.Coeffs[i] {
			p.Coeffs[i][j] = v
		}
	}
}

func fillQP(p ringqp.Poly, v uint64) {
	fillPoly(p.Q, v)
	fillPoly(p.P, v)
}

const (
	patternA = 0x0101010101010101
	patternB = 0x0202020202020202
)

func shapeOf(fill func(v uint64), marshal func() ([]byte, error), skipFrom, skipTo int) (*shape, error) {
	fill(patternA)
	a, err := marshal()
	if err != nil {
		return nil, err
	}
	fill(patternB)
	b, err := marshal()
	if err != nil {
		return nil, err
	}
	return buildShape(a, b, skipFrom, skipTo)
}

type shapes struct {
	sk  *shape
	pk  *shape
	rlk *shape
	ct  map[int]*shape // by binary size -> shape (degree-1 ciphertext)
	lvl map[int]int    // by binary size -> level
}

var shapeCache sync.Map // key: normalized literal string -> *shapes

func shapesFor(params lckks.Parameters) (*shapes, error) {
	lit, err := normalizedLiteral(params.ParametersLiteral())
	if err != nil {
		return nil, err
	}
	key := string(lit)
	if v, ok := shapeCache.Load(key); ok {
		return v.(*shapes), nil
	}
	s := &shapes{ct: map[int]*shape{}, lvl: map[int]int{}}

	sk := rlwe.NewSecretKey(params)
	if s.sk, err = shapeOf(func(v uint64) { fillQP(sk.Value, v) }, sk.MarshalBinary, 0, 0); err != nil {
		return nil, err
	}
	pk := rlwe.NewPublicKey(params)
	if s.pk, err = shapeOf(func(v uint64) {
		for i := range pk.Value {
			fillQP(pk.Value[i], v)
		}
	}, pk.MarshalBinary, 0, 0); err != nil {
		return nil, err
	}
	rlk := rlwe.NewRelinearizationKey(params)
	if s.rlk, err = shapeOf(func(v uint64) {
		for i := range rlk.Value {
			for j := range rlk.Value[i] {
				for k := range rlk.Value[i][j] {
					fillQP(rlk.Value[i][j][k], v)
				}
			}
		}
	}, rlk.MarshalBinary, 0, 0); err != nil {
		return nil, err
	}
	for level := 0; level <= params.MaxLevel(); level++ {
		ct := lckks.NewCiphertext(params, 1, level)
		metaEnd := 1 + ct.MetaData.BinarySize()
		sh, err := shapeOf(func(v uint64) {
			for i := range ct.Value {
				fillPoly(ct.Value[i], v)
			}
		}, ct.MarshalBinary, 1, metaEnd)
		if err != nil {
			return nil, err
		}
		s.ct[sh.size] = sh
		s.lvl[sh.size] = level
	}
	actual, _ := shapeCache.LoadOrStore(key, s)
	return actual.(*shapes), nil
}

// guard converts a panic inside Lattigo deserialization into an error.
func guard(f func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", codec.ErrUnloadable, r)
		}
	}()
	return f()
}

// IsPrivateKeyBlob reports whether b has the binary structure of a secret
// key for params. Server code uses it to reject payloads carrying an sk.
func IsPrivateKeyBlob(params lckks.Parameters, b []byte) bool {
	s, err := shapesFor(params)
	if err != nil {
		return false
	}
	return s.sk.matches(b)
}

// CheckNotPrivate returns ErrPrivateKeyMaterial if b is a serialized secret key.
func CheckNotPrivate(params lckks.Parameters, b []byte) error {
	if IsPrivateKeyBlob(params, b) {
		return ErrPrivateKeyMaterial
	}
	return nil
}

// LoadPublicKey loads a serialized public key after a structural check.
func LoadPublicKey(params lckks.Parameters, b []byte) (*rlwe.PublicKey, error) {
	s, err := shapesFor(params)
	if err != nil {
		return nil, err
	}
	if s.sk.matches(b) {
		return nil, ErrPrivateKeyMaterial
	}
	if !s.pk.matches(b) {
		return nil, fmt.Errorf("%w: not a public key for this parameter set (%d bytes, want %d)", codec.ErrUnloadable, len(b), s.pk.size)
	}
	pk := rlwe.NewPublicKey(params)
	if err := guard(func() error { return pk.UnmarshalBinary(b) }); err != nil {
		return nil, fmt.Errorf("%w: %w", codec.ErrUnloadable, err)
	}
	return pk, nil
}

// LoadRelinKey loads a serialized relinearization key after a structural check.
func LoadRelinKey(params lckks.Parameters, b []byte) (*rlwe.RelinearizationKey, error) {
	s, err := shapesFor(params)
	if err != nil {
		return nil, err
	}
	if s.sk.matches(b) {
		return nil, ErrPrivateKeyMaterial
	}
	if !s.rlk.matches(b) {
		return nil, fmt.Errorf("%w: not a relinearization key for this parameter set (%d bytes, want %d)", codec.ErrUnloadable, len(b), s.rlk.size)
	}
	rlk := rlwe.NewRelinearizationKey(params)
	if err := guard(func() error { return rlk.UnmarshalBinary(b) }); err != nil {
		return nil, fmt.Errorf("%w: %w", codec.ErrUnloadable, err)
	}
	return rlk, nil
}

// LoadSecretKey loads a serialized secret key (client side only).
func LoadSecretKey(params lckks.Parameters, b []byte) (*rlwe.SecretKey, error) {
	s, err := shapesFor(params)
	if err != nil {
		return nil, err
	}
	if !s.sk.matches(b) {
		return nil, fmt.Errorf("%w: not a secret key for this parameter set (%d bytes, want %d)", codec.ErrUnloadable, len(b), s.sk.size)
	}
	sk := rlwe.NewSecretKey(params)
	if err := guard(func() error { return sk.UnmarshalBinary(b) }); err != nil {
		return nil, fmt.Errorf("%w: %w", codec.ErrUnloadable, err)
	}
	return sk, nil
}

// LoadCiphertext loads a serialized degree-1 ciphertext and checks that it
// fits params: level within the modulus chain, NTT form, full slot
// dimensions and a sane, positive scale. Use RequireDefaultScale to also
// demand the exact default scale.
func LoadCiphertext(params lckks.Parameters, b []byte) (*rlwe.Ciphertext, error) {
	s, err := shapesFor(params)
	if err != nil {
		return nil, err
	}
	if s.sk.matches(b) {
		return nil, ErrPrivateKeyMaterial
	}
	sh, ok := s.ct[len(b)]
	if !ok || !sh.matches(b) {
		return nil, fmt.Errorf("%w: not a ciphertext for this parameter set (%d bytes)", codec.ErrUnloadable, len(b))
	}
	ct := lckks.NewCiphertext(params, 1, s.lvl[len(b)])
	if err := guard(func() error { return ct.UnmarshalBinary(b) }); err != nil {
		return nil, fmt.Errorf("%w: %w", codec.ErrUnloadable, err)
	}
	if ct.MetaData == nil || ct.Degree() != 1 || ct.Level() != s.lvl[len(b)] {
		return nil, fmt.Errorf("%w: unexpected degree or level", ErrContextMismatch)
	}
	if !ct.IsNTT || ct.IsMontgomery {
		return nil, fmt.Errorf("%w: ciphertext must be in NTT, non-Montgomery form", ErrContextMismatch)
	}
	if ct.LogDimensions != params.LogMaxDimensions() {
		return nil, fmt.Errorf("%w: unexpected slot dimensions", ErrContextMismatch)
	}
	l2 := ct.Scale.Log2()
	if math.IsNaN(l2) || math.IsInf(l2, 0) || l2 < 1 || l2 > 200 {
		return nil, fmt.Errorf("%w: invalid scale", ErrContextMismatch)
	}
	return ct, nil
}

// RequireDefaultScale returns ErrContextMismatch unless ct has exactly the
// default scale. All ciphertexts of the protocol (fresh client inputs,
// balances, totals) keep this exact scale, so additions never mix scales.
func RequireDefaultScale(params lckks.Parameters, ct *rlwe.Ciphertext) error {
	if !ct.Scale.Equal(params.DefaultScale()) {
		return fmt.Errorf("%w: scale 2^%.4f, expected 2^%d", ErrContextMismatch, ct.Scale.Log2(), params.LogDefaultScale())
	}
	return nil
}

// Marshal serializes a Lattigo object with its own binary format.
func Marshal(m interface{ MarshalBinary() ([]byte, error) }) ([]byte, error) {
	return m.MarshalBinary()
}

// MarshalB64 serializes a Lattigo object and encodes it as standard Base64.
func MarshalB64(m interface{ MarshalBinary() ([]byte, error) }) (string, error) {
	b, err := m.MarshalBinary()
	if err != nil {
		return "", err
	}
	return codec.Encode(b), nil
}
