// Package ckks wraps the Lattigo v6 CKKS scheme for the shop: parameter
// profiles, key generation, encryption/decryption (client side), homomorphic
// evaluation (server side) and safe loading of serialized objects.
//
// The same package is compiled natively (server, tests, benchmarks) and to
// WebAssembly (browser bridge), so serialized objects are byte-compatible.
// Serialization always uses Lattigo's own binary/JSON formats.
package ckks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"sync"

	"github.com/filbertengyo/sisterpedia/config"
	lckks "github.com/tuneinsight/lattigo/v6/schemes/ckks"
	"gopkg.in/yaml.v3"
)

// LibraryVersion is the pinned Lattigo version (see go.mod).
const LibraryVersion = "github.com/tuneinsight/lattigo/v6 v6.2.0"

// FormatVersion identifies the serialization format of keys/ciphertexts
// produced by this package (Lattigo v6 MarshalBinary, params as Lattigo JSON).
const FormatVersion = "sisterpedia-ckks/1 (lattigo v6 binary)"

// MinLevel is the minimum multiplicative depth a profile must support: the
// bill needs two chained ciphertext-ciphertext multiplications.
const MinLevel = 2

// heStandard128 is the maximum log2(QP) giving 128-bit classical security with
// a uniform ternary secret (HomomorphicEncryption.org security standard).
// Lattigo validates parameters but does not enforce this bound, so we do.
var heStandard128 = map[int]float64{10: 27, 11: 54, 12: 109, 13: 218, 14: 438, 15: 881, 16: 1761}

// Profile is a named CKKS parameter set loaded from config/profiles.yaml.
type Profile struct {
	Name            string `yaml:"-" json:"name"`
	LogN            int    `yaml:"log_n" json:"logN"`
	LogQ            []int  `yaml:"log_q" json:"logQ"`
	LogP            []int  `yaml:"log_p" json:"logP"`
	LogDefaultScale int    `yaml:"log_default_scale" json:"logDefaultScale"`
}

// ProfileInfo summarizes a profile for clients and reports.
type ProfileInfo struct {
	Name            string  `json:"name"`
	LogN            int     `json:"logN"`
	MaxLevel        int     `json:"maxLevel"`
	Slots           int     `json:"slots"`
	LogDefaultScale int     `json:"logDefaultScale"`
	LogQP           float64 `json:"logQP"`
	MaxValue        float64 `json:"maxValue"`
}

// Registry holds validated profiles and their Lattigo parameters.
type Registry struct {
	names    []string
	profiles map[string]Profile
	params   map[string]lckks.Parameters
	literal  map[string][]byte // normalized JSON literal for matching
}

var (
	// ErrUnknownProfile is returned for a profile name that is not configured.
	ErrUnknownProfile = errors.New("unknown profile")
	// ErrParamsMismatch is returned when serialized parameters match no allowed profile.
	ErrParamsMismatch = errors.New("parameters do not match an allowed profile")
)

// ParseProfiles parses a profiles YAML document and validates every profile:
// Lattigo's own parameter checks, at least MinLevel levels, and the 128-bit
// RLWE security bound on log2(QP).
func ParseProfiles(data []byte) (*Registry, error) {
	var doc struct {
		Profiles map[string]Profile `yaml:"profiles"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse profiles: %w", err)
	}
	if len(doc.Profiles) == 0 {
		return nil, errors.New("parse profiles: no profiles defined")
	}
	r := &Registry{
		profiles: map[string]Profile{},
		params:   map[string]lckks.Parameters{},
		literal:  map[string][]byte{},
	}
	for name, p := range doc.Profiles {
		p.Name = name
		params, err := lckks.NewParametersFromLiteral(lckks.ParametersLiteral{
			LogN:            p.LogN,
			LogQ:            p.LogQ,
			LogP:            p.LogP,
			LogDefaultScale: p.LogDefaultScale,
		})
		if err != nil {
			return nil, fmt.Errorf("profile %q rejected by Lattigo: %w", name, err)
		}
		if params.MaxLevel() < MinLevel {
			return nil, fmt.Errorf("profile %q: max level %d < %d (two chained multiplications are required)", name, params.MaxLevel(), MinLevel)
		}
		bound, ok := heStandard128[p.LogN]
		if !ok || params.LogQP() > bound {
			return nil, fmt.Errorf("profile %q: log2(QP)=%.1f exceeds the 128-bit security bound for LogN=%d", name, params.LogQP(), p.LogN)
		}
		lit, err := normalizedLiteral(params.ParametersLiteral())
		if err != nil {
			return nil, err
		}
		r.names = append(r.names, name)
		r.profiles[name] = p
		r.params[name] = params
		r.literal[name] = lit
	}
	sort.Strings(r.names)
	return r, nil
}

var (
	defaultOnce sync.Once
	defaultReg  *Registry
	defaultErr  error
)

// DefaultRegistry returns the profiles embedded from config/profiles.yaml.
func DefaultRegistry() (*Registry, error) {
	defaultOnce.Do(func() {
		defaultReg, defaultErr = ParseProfiles(config.ProfilesYAML)
	})
	return defaultReg, defaultErr
}

// LoadRegistry loads profiles from path, or the embedded defaults when path is empty.
func LoadRegistry(path string) (*Registry, error) {
	if path == "" {
		return DefaultRegistry()
	}
	data, err := os.ReadFile(path) // #nosec G304 -- path comes from operator configuration
	if err != nil {
		return nil, fmt.Errorf("read profiles: %w", err)
	}
	return ParseProfiles(data)
}

// Restrict returns a registry containing only the named profiles.
func (r *Registry) Restrict(names []string) (*Registry, error) {
	out := &Registry{profiles: map[string]Profile{}, params: map[string]lckks.Parameters{}, literal: map[string][]byte{}}
	for _, n := range names {
		if _, ok := r.profiles[n]; !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnknownProfile, n)
		}
		if _, dup := out.profiles[n]; dup {
			continue
		}
		out.names = append(out.names, n)
		out.profiles[n] = r.profiles[n]
		out.params[n] = r.params[n]
		out.literal[n] = r.literal[n]
	}
	sort.Strings(out.names)
	return out, nil
}

// Names returns the profile names in sorted order.
func (r *Registry) Names() []string { return append([]string(nil), r.names...) }

// Params returns the Lattigo parameters of a profile.
func (r *Registry) Params(name string) (lckks.Parameters, error) {
	p, ok := r.params[name]
	if !ok {
		return lckks.Parameters{}, fmt.Errorf("%w: %q", ErrUnknownProfile, name)
	}
	return p, nil
}

// Profile returns the configured literal of a profile.
func (r *Registry) Profile(name string) (Profile, error) {
	p, ok := r.profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("%w: %q", ErrUnknownProfile, name)
	}
	return p, nil
}

// Info returns a summary of a profile.
func (r *Registry) Info(name string) (ProfileInfo, error) {
	params, err := r.Params(name)
	if err != nil {
		return ProfileInfo{}, err
	}
	return ProfileInfo{
		Name:            name,
		LogN:            params.LogN(),
		MaxLevel:        params.MaxLevel(),
		Slots:           params.MaxSlots(),
		LogDefaultScale: params.LogDefaultScale(),
		LogQP:           params.LogQP(),
		MaxValue:        MaxValue(params),
	}, nil
}

// MarshalParams serializes parameters with Lattigo's JSON format (params.json).
func MarshalParams(params lckks.Parameters) ([]byte, error) {
	return params.MarshalJSON()
}

// MatchParams decodes serialized parameters (Lattigo JSON) and returns the
// allowed profile they are exactly equal to. Untrusted input is only decoded
// as a literal and compared; no ring is ever built from it.
func (r *Registry) MatchParams(data []byte) (string, lckks.Parameters, error) {
	if len(data) == 0 || len(data) > 1<<16 {
		return "", lckks.Parameters{}, fmt.Errorf("%w: size %d", ErrParamsMismatch, len(data))
	}
	var lit lckks.ParametersLiteral
	if err := json.Unmarshal(data, &lit); err != nil {
		return "", lckks.Parameters{}, fmt.Errorf("%w: %w", ErrParamsMismatch, err)
	}
	norm, err := normalizedLiteral(lit)
	if err != nil {
		return "", lckks.Parameters{}, fmt.Errorf("%w: %w", ErrParamsMismatch, err)
	}
	for _, name := range r.names {
		if bytes.Equal(norm, r.literal[name]) {
			return name, r.params[name], nil
		}
	}
	return "", lckks.Parameters{}, ErrParamsMismatch
}

// normalizedLiteral returns a canonical JSON form of the fields that define
// the parameter set, used for exact comparison.
func normalizedLiteral(l lckks.ParametersLiteral) ([]byte, error) {
	canon := struct {
		LogN            int
		Q               []uint64
		P               []uint64
		Xs              any
		Xe              any
		RingType        any
		LogDefaultScale int
	}{l.LogN, l.Q, l.P, l.Xs, l.Xe, l.RingType, l.LogDefaultScale}
	return json.Marshal(canon)
}

// MaxValue is the largest |x| that still decodes correctly at level 0:
// |x| * scale < q0 / 2. Larger values overflow silently (wrong results, no error).
func MaxValue(params lckks.Parameters) float64 {
	q0 := float64(params.Q()[0])
	return math.Floor(q0 / 2 / params.DefaultScale().Float64())
}
