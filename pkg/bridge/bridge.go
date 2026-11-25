// Package bridge implements the browser crypto bridge (docs/CRYPTO_BRIDGE.md
// section 1) as plain Go so it can be tested natively; cmd/wasm only adapts
// it to syscall/js. Every function is pure crypto: no I/O, no storage, no
// network, and arguments are never logged.
package bridge

import (
	"errors"
	"fmt"
	"sync"

	"github.com/filbertengyo/sisterpedia/pkg/ckks"
	"github.com/filbertengyo/sisterpedia/pkg/codec"
	lckks "github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

// MaxB64 bounds every Base64 argument (16 MiB).
const MaxB64 = 16 << 20

// Error codes returned to JavaScript.
const (
	CodeInvalidPayload   = "invalid_payload"
	CodeParamsMismatch   = "params_mismatch"
	CodeKeysMismatch     = "keys_mismatch"
	CodeValueOutOfRange  = "value_out_of_range"
	CodeUnknownProfile   = "unknown_profile"
	CodeInternal         = "internal_error"
	CodeInvalidArguments = "invalid_arguments"
)

// Error is the error object of a failed call.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Result is the envelope returned by every bridge function.
type Result struct {
	OK    bool   `json:"ok"`
	Value any    `json:"value,omitempty"`
	Error *Error `json:"error,omitempty"`
}

// KeyBundle is the result of GenKeys: Base64 of the library serialization.
type KeyBundle struct {
	Profile   string `json:"profile"`
	Params    string `json:"params"`
	SecretKey string `json:"secret_key"`
	PublicKey string `json:"public_key"`
	RelinKey  string `json:"relin_key"`
}

// CiphertextInfo describes a ciphertext without decrypting it.
type CiphertextInfo struct {
	Level     int     `json:"level"`
	Log2Scale float64 `json:"log2Scale"`
	Bytes     int     `json:"bytes"`
}

// ParamsInfo identifies the profile of serialized parameters.
type ParamsInfo struct {
	ckks.ProfileInfo
	Profile string `json:"profile"`
}

// Bridge holds the profile registry and small caches of loaded keys (keys
// are MiB-sized, so re-parsing them on every call would be slow).
type Bridge struct {
	reg *ckks.Registry

	mu  sync.Mutex
	enc map[[2]string]*ckks.Client
	dec map[[2]string]*ckks.Client
}

const cacheSize = 8

// New builds a bridge over the embedded profiles.
func New() (*Bridge, error) {
	reg, err := ckks.DefaultRegistry()
	if err != nil {
		return nil, err
	}
	return &Bridge{reg: reg, enc: map[[2]string]*ckks.Client{}, dec: map[[2]string]*ckks.Client{}}, nil
}

func codeOf(err error) string {
	switch {
	case errors.Is(err, ckks.ErrUnknownProfile):
		return CodeUnknownProfile
	case errors.Is(err, ckks.ErrParamsMismatch), errors.Is(err, ckks.ErrContextMismatch):
		return CodeParamsMismatch
	case errors.Is(err, ckks.ErrKeysMismatch):
		return CodeKeysMismatch
	case errors.Is(err, ckks.ErrValueOutOfRange):
		return CodeValueOutOfRange
	case errors.Is(err, codec.ErrInvalidBase64), errors.Is(err, codec.ErrUnloadable), errors.Is(err, ckks.ErrPrivateKeyMaterial):
		return CodeInvalidPayload
	case errors.Is(err, errInvalidArgs):
		return CodeInvalidArguments
	default:
		return CodeInternal
	}
}

var errInvalidArgs = errors.New("invalid arguments")

// Wrap runs f and converts its result, error or panic into a Result.
func Wrap(f func() (any, error)) (res Result) {
	defer func() {
		if r := recover(); r != nil {
			res = Result{Error: &Error{Code: CodeInternal, Message: fmt.Sprintf("internal error: %v", r)}}
		}
	}()
	v, err := f()
	if err != nil {
		return Result{Error: &Error{Code: codeOf(err), Message: err.Error()}}
	}
	return Result{OK: true, Value: v}
}

func (b *Bridge) params(paramsB64 string) (string, lckks.Parameters, error) {
	bin, err := codec.Decode(paramsB64, 1<<16)
	if err != nil {
		return "", lckks.Parameters{}, err
	}
	return b.reg.MatchParams(bin)
}

// Profiles lists the configured profiles.
func (b *Bridge) Profiles() ([]ckks.ProfileInfo, error) {
	out := []ckks.ProfileInfo{}
	for _, n := range b.reg.Names() {
		info, err := b.reg.Info(n)
		if err != nil {
			return nil, err
		}
		out = append(out, info)
	}
	return out, nil
}

// ParamsInfo returns the profile matching serialized parameters.
func (b *Bridge) ParamsInfo(paramsB64 string) (ParamsInfo, error) {
	name, _, err := b.params(paramsB64)
	if err != nil {
		return ParamsInfo{}, err
	}
	info, err := b.reg.Info(name)
	return ParamsInfo{ProfileInfo: info, Profile: name}, err
}

// GenKeys generates params, sk, pk and the relinearization key for a profile.
func (b *Bridge) GenKeys(profile string) (KeyBundle, error) {
	params, err := b.reg.Params(profile)
	if err != nil {
		return KeyBundle{}, err
	}
	ser, err := ckks.GenKeys(params).Serialize()
	if err != nil {
		return KeyBundle{}, err
	}
	return KeyBundle{
		Profile: profile, Params: codec.Encode(ser.Params), SecretKey: codec.Encode(ser.SK),
		PublicKey: codec.Encode(ser.PK), RelinKey: codec.Encode(ser.RLK),
	}, nil
}

// ValidateKeys checks that pk (and rlk if non-empty) belong to sk.
func (b *Bridge) ValidateKeys(paramsB64, skB64, pkB64, rlkB64 string) (bool, error) {
	_, params, err := b.params(paramsB64)
	if err != nil {
		return false, err
	}
	skBin, err := codec.Decode(skB64, MaxB64)
	if err != nil {
		return false, err
	}
	sk, err := ckks.LoadSecretKey(params, skBin)
	if err != nil {
		return false, err
	}
	pkBin, err := codec.Decode(pkB64, MaxB64)
	if err != nil {
		return false, err
	}
	pk, err := ckks.LoadPublicKey(params, pkBin)
	if err != nil {
		return false, err
	}
	var rlkErr error
	if rlkB64 != "" {
		rlkBin, err := codec.Decode(rlkB64, MaxB64)
		if err != nil {
			return false, err
		}
		rlk, err := ckks.LoadRelinKey(params, rlkBin)
		if err != nil {
			return false, err
		}
		rlkErr = ckks.ValidateKeys(params, sk, pk, rlk)
	} else {
		rlkErr = ckks.ValidateKeys(params, sk, pk, nil)
	}
	if rlkErr != nil {
		return false, rlkErr
	}
	return true, nil
}

func (b *Bridge) encryptor(paramsB64, pkB64 string) (*ckks.Client, error) {
	key := [2]string{paramsB64, pkB64}
	b.mu.Lock()
	c, ok := b.enc[key]
	b.mu.Unlock()
	if ok {
		return c, nil
	}
	_, params, err := b.params(paramsB64)
	if err != nil {
		return nil, err
	}
	pkBin, err := codec.Decode(pkB64, MaxB64)
	if err != nil {
		return nil, err
	}
	pk, err := ckks.LoadPublicKey(params, pkBin)
	if err != nil {
		return nil, err
	}
	c = ckks.NewClient(params, nil, pk)
	b.mu.Lock()
	if len(b.enc) >= cacheSize {
		clear(b.enc)
	}
	b.enc[key] = c
	b.mu.Unlock()
	return c, nil
}

func (b *Bridge) decryptor(paramsB64, skB64 string) (*ckks.Client, error) {
	key := [2]string{paramsB64, skB64}
	b.mu.Lock()
	c, ok := b.dec[key]
	b.mu.Unlock()
	if ok {
		return c, nil
	}
	_, params, err := b.params(paramsB64)
	if err != nil {
		return nil, err
	}
	skBin, err := codec.Decode(skB64, MaxB64)
	if err != nil {
		return nil, err
	}
	sk, err := ckks.LoadSecretKey(params, skBin)
	if err != nil {
		return nil, err
	}
	c = ckks.NewClient(params, sk, nil)
	b.mu.Lock()
	if len(b.dec) >= cacheSize {
		clear(b.dec)
	}
	b.dec[key] = c
	b.mu.Unlock()
	return c, nil
}

// EncryptVector encodes and encrypts values (slot i = values[i]).
func (b *Bridge) EncryptVector(paramsB64, pkB64 string, values []float64) (string, error) {
	c, err := b.encryptor(paramsB64, pkB64)
	if err != nil {
		return "", err
	}
	ct, err := c.EncryptVector(values)
	if err != nil {
		return "", err
	}
	return ckks.MarshalB64(ct)
}

// EncryptScalar encrypts one value in slot 0.
func (b *Bridge) EncryptScalar(paramsB64, pkB64 string, x float64) (string, error) {
	return b.EncryptVector(paramsB64, pkB64, []float64{x})
}

// DecryptVector decrypts and decodes the first n slots (raw values).
func (b *Bridge) DecryptVector(paramsB64, skB64, ctB64 string, n int) ([]float64, error) {
	c, err := b.decryptor(paramsB64, skB64)
	if err != nil {
		return nil, err
	}
	bin, err := codec.Decode(ctB64, MaxB64)
	if err != nil {
		return nil, err
	}
	ct, err := ckks.LoadCiphertext(c.Params(), bin)
	if err != nil {
		return nil, err
	}
	return c.DecryptVector(ct, n)
}

// DecryptScalar decrypts slot 0 (raw value).
func (b *Bridge) DecryptScalar(paramsB64, skB64, ctB64 string) (float64, error) {
	v, err := b.DecryptVector(paramsB64, skB64, ctB64, 1)
	if err != nil {
		return 0, err
	}
	return v[0], nil
}

// CiphertextInfo returns level, log2 scale and binary size of a ciphertext.
func (b *Bridge) CiphertextInfo(paramsB64, ctB64 string) (CiphertextInfo, error) {
	_, params, err := b.params(paramsB64)
	if err != nil {
		return CiphertextInfo{}, err
	}
	bin, err := codec.Decode(ctB64, MaxB64)
	if err != nil {
		return CiphertextInfo{}, err
	}
	ct, err := ckks.LoadCiphertext(params, bin)
	if err != nil {
		return CiphertextInfo{}, err
	}
	return CiphertextInfo{Level: ct.Level(), Log2Scale: ct.Scale.Log2(), Bytes: len(bin)}, nil
}

// Version returns the bridge and library version.
func Version() string {
	return "sisterpedia-bridge/1 " + ckks.LibraryVersion + " " + ckks.FormatVersion
}

// InvalidArgs builds an invalid-arguments error.
func InvalidArgs(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errInvalidArgs, fmt.Sprintf(format, args...))
}
