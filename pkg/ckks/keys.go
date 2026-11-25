package ckks

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/filbertengyo/sisterpedia/pkg/monoclock"
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	lckks "github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

// KeySet is a full key set of one account. Only the client ever holds SK.
type KeySet struct {
	Params lckks.Parameters
	SK     *rlwe.SecretKey
	PK     *rlwe.PublicKey
	RLK    *rlwe.RelinearizationKey
}

// KeyTimings records the duration of each key generation stage.
type KeyTimings struct {
	Params    time.Duration
	SecretKey time.Duration
	PublicKey time.Duration
	RelinKey  time.Duration
}

// Total is the sum of all stages.
func (t KeyTimings) Total() time.Duration {
	return t.Params + t.SecretKey + t.PublicKey + t.RelinKey
}

// ErrKeysMismatch is returned when keys of a set do not belong together.
var ErrKeysMismatch = errors.New("keys do not belong to the same key pair")

// GenKeys generates sk, pk and the relinearization key for params.
func GenKeys(params lckks.Parameters) KeySet {
	ks, _ := genKeys(params)
	return ks
}

func genKeys(params lckks.Parameters) (KeySet, KeyTimings) {
	var t KeyTimings
	kgen := lckks.NewKeyGenerator(params)

	start := monoclock.Now()
	sk := kgen.GenSecretKeyNew()
	t.SecretKey = monoclock.Since(start)

	start = monoclock.Now()
	pk := kgen.GenPublicKeyNew(sk)
	t.PublicKey = monoclock.Since(start)

	start = monoclock.Now()
	rlk := kgen.GenRelinearizationKeyNew(sk)
	t.RelinKey = monoclock.Since(start)

	return KeySet{Params: params, SK: sk, PK: pk, RLK: rlk}, t
}

// GenKeysTimed builds the parameters from a profile literal and generates
// the key set, timing every stage separately (benchmark metric "keygen").
func GenKeysTimed(p Profile) (KeySet, KeyTimings, error) {
	start := monoclock.Now()
	params, err := lckks.NewParametersFromLiteral(lckks.ParametersLiteral{
		LogN: p.LogN, LogQ: p.LogQ, LogP: p.LogP, LogDefaultScale: p.LogDefaultScale,
	})
	tParams := monoclock.Since(start)
	if err != nil {
		return KeySet{}, KeyTimings{}, fmt.Errorf("profile %q: %w", p.Name, err)
	}
	ks, t := genKeys(params)
	t.Params = tParams
	return ks, t, nil
}

// Serialized is the binary form of a key set: params.json, secret.key,
// public.key and relin.key.
type Serialized struct {
	Params []byte
	SK     []byte
	PK     []byte
	RLK    []byte
}

// Serialize marshals every element with Lattigo's own formats.
func (ks KeySet) Serialize() (Serialized, error) {
	var s Serialized
	var err error
	if s.Params, err = MarshalParams(ks.Params); err != nil {
		return s, err
	}
	if ks.SK != nil {
		if s.SK, err = ks.SK.MarshalBinary(); err != nil {
			return s, err
		}
	}
	if s.PK, err = ks.PK.MarshalBinary(); err != nil {
		return s, err
	}
	if s.RLK, err = ks.RLK.MarshalBinary(); err != nil {
		return s, err
	}
	return s, nil
}

// LoadKeySet loads a serialized key set against an allowed registry.
// rlk may be nil (not needed for decryption).
func LoadKeySet(reg *Registry, s Serialized) (string, KeySet, error) {
	name, params, err := reg.MatchParams(s.Params)
	if err != nil {
		return "", KeySet{}, err
	}
	ks := KeySet{Params: params}
	if ks.SK, err = LoadSecretKey(params, s.SK); err != nil {
		return "", KeySet{}, fmt.Errorf("secret key: %w", err)
	}
	if ks.PK, err = LoadPublicKey(params, s.PK); err != nil {
		return "", KeySet{}, fmt.Errorf("public key: %w", err)
	}
	if len(s.RLK) > 0 {
		if ks.RLK, err = LoadRelinKey(params, s.RLK); err != nil {
			return "", KeySet{}, fmt.Errorf("relinearization key: %w", err)
		}
	}
	return name, ks, nil
}

// ValidateKeys checks that pk (and rlk when non-nil) were generated from sk:
// a value encrypted under pk must decrypt correctly with sk, and a product
// relinearized with rlk must decrypt to the expected product.
func ValidateKeys(params lckks.Parameters, sk *rlwe.SecretKey, pk *rlwe.PublicKey, rlk *rlwe.RelinearizationKey) error {
	c := NewClient(params, sk, pk)
	const a, b = 3.25, 7.5
	ca, err := c.EncryptScalar(a)
	if err != nil {
		return err
	}
	got, err := c.DecryptScalar(ca)
	if err != nil {
		return err
	}
	if math.Abs(got-a) > 1e-3 {
		return fmt.Errorf("%w: public key does not match secret key", ErrKeysMismatch)
	}
	if rlk == nil {
		return nil
	}
	cb, err := c.EncryptScalar(b)
	if err != nil {
		return err
	}
	ev := NewEvaluator(params, pk, rlk)
	prod, err := ev.MulRelinRescale(ca, cb, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrKeysMismatch, err)
	}
	got, err = c.DecryptScalar(prod)
	if err != nil {
		return err
	}
	if math.Abs(got-a*b) > 1e-3 {
		return fmt.Errorf("%w: relinearization key does not match secret key", ErrKeysMismatch)
	}
	return nil
}
