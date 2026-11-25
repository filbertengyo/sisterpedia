// Package shopclient is the native (Go) counterpart of the browser shop
// module (web/lib/crypto/shop.ts): it generates keys, encrypts inputs,
// calls the REST API and decrypts results. It runs in a trusted process
// separate from the server (end-to-end tests, benchmarks) and uses the same
// pkg/ckks code that is compiled to WebAssembly for the browser.
package shopclient

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/filbertengyo/sisterpedia/internal/apiclient"
	"github.com/filbertengyo/sisterpedia/pkg/ckks"
	"github.com/filbertengyo/sisterpedia/pkg/codec"
	"github.com/filbertengyo/sisterpedia/pkg/monoclock"
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
)

// MetadataTolerance is the maximum distance from an integer for a decrypted
// metadata slot to be considered certain.
const MetadataTolerance = 0.25

// Timings records client-side crypto durations of the last operation.
type Timings struct {
	Encrypt time.Duration
	Decrypt time.Duration
	Total   time.Duration
}

// User is one account on the client side (keys never leave this struct
// except through SaveKeys).
type User struct {
	ID      string
	Profile string
	Keys    ckks.KeySet
	Crypto  *ckks.Client
	API     *apiclient.Client
	Catalog apiclient.Catalog
	Last    Timings
}

// New generates a fresh key set for profile and returns an unactivated user.
func New(api *apiclient.Client, reg *ckks.Registry, id, profile string) (*User, error) {
	params, err := reg.Params(profile)
	if err != nil {
		return nil, err
	}
	ks := ckks.GenKeys(params)
	return fromKeys(api, id, profile, ks), nil
}

func fromKeys(api *apiclient.Client, id, profile string, ks ckks.KeySet) *User {
	return &User{ID: id, Profile: profile, Keys: ks, Crypto: ckks.NewClient(ks.Params, ks.SK, ks.PK), API: api}
}

// Key file names of the export bundle (Table 2.3 of the assignment).
const (
	FileParams = "params.json"
	FileSK     = "secret.key"
	FilePK     = "public.key"
	FileRLK    = "relin.key"
)

// SaveKeys writes params.json, secret.key, public.key and relin.key to dir.
func (u *User) SaveKeys(dir string) error {
	s, err := u.Keys.Serialize()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	for name, data := range map[string][]byte{FileParams: s.Params, FileSK: s.SK, FilePK: s.PK, FileRLK: s.RLK} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			return err
		}
	}
	return nil
}

// LoadKeys reloads a user from a key directory written by SaveKeys
// (no new key generation).
func LoadKeys(api *apiclient.Client, reg *ckks.Registry, id, dir string) (*User, error) {
	var s ckks.Serialized
	for name, dst := range map[string]*[]byte{FileParams: &s.Params, FileSK: &s.SK, FilePK: &s.PK, FileRLK: &s.RLK} {
		b, err := os.ReadFile(filepath.Join(dir, name)) // #nosec G304 -- test/bench key directory
		if err != nil {
			return nil, err
		}
		*dst = b
	}
	profile, ks, err := ckks.LoadKeySet(reg, s)
	if err != nil {
		return nil, err
	}
	if err := ckks.ValidateKeys(ks.Params, ks.SK, ks.PK, ks.RLK); err != nil {
		return nil, err
	}
	return fromKeys(api, id, profile, ks), nil
}

// ActivateRequest builds the Activate payload (no secret key in it).
func (u *User) ActivateRequest() (apiclient.ActivateRequest, error) {
	s, err := u.Keys.Serialize()
	if err != nil {
		return apiclient.ActivateRequest{}, err
	}
	zero, err := u.Crypto.EncryptScalar(0)
	if err != nil {
		return apiclient.ActivateRequest{}, err
	}
	zb, err := ckks.MarshalB64(zero)
	if err != nil {
		return apiclient.ActivateRequest{}, err
	}
	return apiclient.ActivateRequest{
		UserID: u.ID, Params: codec.Encode(s.Params), PublicKey: codec.Encode(s.PK),
		RelinKey: codec.Encode(s.RLK), EncBalance: zb,
	}, nil
}

// Activate registers the account with Enc(0) and returns the decrypted
// initial balance.
func (u *User) Activate(ctx context.Context) (float64, error) {
	req, err := u.ActivateRequest()
	if err != nil {
		return 0, err
	}
	resp, err := u.API.Activate(ctx, req)
	if err != nil {
		return 0, err
	}
	if cat, err := u.API.Catalog(ctx); err == nil {
		u.Catalog = cat
	}
	return u.DecryptB64(resp.EncBalance)
}

// LoadCatalog fetches the public catalog.
func (u *User) LoadCatalog(ctx context.Context) error {
	cat, err := u.API.Catalog(ctx)
	if err != nil {
		return err
	}
	u.Catalog = cat
	return nil
}

// EncryptB64 encrypts a vector and returns its Base64 serialization.
func (u *User) EncryptB64(values ...float64) (string, error) {
	ct, err := u.Crypto.EncryptVector(values)
	if err != nil {
		return "", err
	}
	return ckks.MarshalB64(ct)
}

// LoadB64 decodes and loads a Base64 ciphertext for this user's parameters.
func (u *User) LoadB64(b64 string) (*rlwe.Ciphertext, error) {
	b, err := codec.Decode(b64, 0)
	if err != nil {
		return nil, err
	}
	return ckks.LoadCiphertext(u.Keys.Params, b)
}

// DecryptB64 decrypts slot 0 of a Base64 ciphertext (raw value).
func (u *User) DecryptB64(b64 string) (float64, error) {
	v, err := u.DecryptVectorB64(b64, 1)
	if err != nil {
		return 0, err
	}
	return v[0], nil
}

// DecryptVectorB64 decrypts the first n slots of a Base64 ciphertext.
func (u *User) DecryptVectorB64(b64 string, n int) ([]float64, error) {
	ct, err := u.LoadB64(b64)
	if err != nil {
		return nil, err
	}
	return u.Crypto.DecryptVector(ct, n)
}

// Balance calls CheckBalance and decrypts the result.
func (u *User) Balance(ctx context.Context) (float64, error) {
	start := monoclock.Now()
	resp, err := u.API.Balance(ctx, u.ID)
	if err != nil {
		return 0, err
	}
	d := monoclock.Now()
	v, err := u.DecryptB64(resp.EncBalance)
	u.Last = Timings{Decrypt: monoclock.Since(d), Total: monoclock.Since(start)}
	return v, err
}

// Topup encrypts amount, calls Topup and decrypts the new balance.
func (u *User) Topup(ctx context.Context, amount float64) (float64, error) {
	start := monoclock.Now()
	enc, err := u.EncryptB64(amount)
	if err != nil {
		return 0, err
	}
	tEnc := monoclock.Since(start)
	resp, err := u.API.Topup(ctx, u.ID, enc)
	if err != nil {
		return 0, err
	}
	d := monoclock.Now()
	v, err := u.DecryptB64(resp.EncBalance)
	u.Last = Timings{Encrypt: tEnc, Decrypt: monoclock.Since(d), Total: monoclock.Since(start)}
	return v, err
}

// TransactResult holds the decrypted outcome of a transaction.
type TransactResult struct {
	TransactionID string
	Total         float64 // raw
	Balance       float64 // raw
}

func (u *User) numericIDs(productID, discount string) (float64, float64, error) {
	if discount == "" {
		discount = "NONE"
	}
	prod, disc := -1, -1
	for _, p := range u.Catalog.Products {
		if p.ProductID == productID {
			prod = p.NumericID
		}
	}
	for _, d := range u.Catalog.Discounts {
		if d.Code == discount {
			disc = d.NumericID
		}
	}
	if prod < 0 || disc < 0 {
		return 0, 0, fmt.Errorf("unknown product %q or discount %q in local catalog", productID, discount)
	}
	return float64(prod), float64(disc), nil
}

// Transact encrypts quantity and metadata, calls Transact and decrypts the
// total and the new balance.
func (u *User) Transact(ctx context.Context, productID string, quantity int, discount string) (TransactResult, error) {
	if len(u.Catalog.Products) == 0 {
		if err := u.LoadCatalog(ctx); err != nil {
			return TransactResult{}, err
		}
	}
	prod, disc, err := u.numericIDs(productID, discount)
	if err != nil {
		return TransactResult{}, err
	}
	start := monoclock.Now()
	encQ, err := u.EncryptB64(float64(quantity))
	if err != nil {
		return TransactResult{}, err
	}
	encM, err := u.EncryptB64(prod, float64(quantity), disc)
	if err != nil {
		return TransactResult{}, err
	}
	tEnc := monoclock.Since(start)
	req := apiclient.TransactRequest{ProductID: productID, EncQuantity: encQ, EncMetadata: encM}
	if discount != "" {
		req.DiscountCode = &discount
	}
	resp, err := u.API.Transact(ctx, u.ID, req)
	if err != nil {
		return TransactResult{}, err
	}
	d := monoclock.Now()
	total, err := u.DecryptB64(resp.EncTotal)
	if err != nil {
		return TransactResult{}, err
	}
	bal, err := u.DecryptB64(resp.EncBalance)
	if err != nil {
		return TransactResult{}, err
	}
	u.Last = Timings{Encrypt: tEnc, Decrypt: monoclock.Since(d), Total: monoclock.Since(start)}
	return TransactResult{TransactionID: resp.TransactionID, Total: total, Balance: bal}, nil
}

// HistoryEntry is a decrypted GetTransact entry.
type HistoryEntry struct {
	TransactionID string
	CreatedAt     string
	ProductID     string
	Quantity      int
	DiscountCode  string
	MetadataRaw   []float64
	Uncertain     bool
	Total         float64 // raw
}

// History calls GetTransact and decrypts metadata and totals. Metadata slots
// are rounded to the nearest integer and mapped back through the catalog;
// Uncertain is set if a slot is more than MetadataTolerance from an integer.
func (u *User) History(ctx context.Context) ([]HistoryEntry, error) {
	if len(u.Catalog.Products) == 0 {
		if err := u.LoadCatalog(ctx); err != nil {
			return nil, err
		}
	}
	resp, err := u.API.History(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	out := make([]HistoryEntry, 0, len(resp.Transactions))
	for _, t := range resp.Transactions {
		meta, err := u.DecryptVectorB64(t.EncMetadata, 3)
		if err != nil {
			return nil, err
		}
		total, err := u.DecryptB64(t.EncTotal)
		if err != nil {
			return nil, err
		}
		e := HistoryEntry{TransactionID: t.TransactionID, CreatedAt: t.CreatedAt, MetadataRaw: meta, Total: total}
		r := make([]int, 3)
		for i, v := range meta {
			r[i] = int(math.Round(v))
			if math.Abs(v-float64(r[i])) > MetadataTolerance {
				e.Uncertain = true
			}
		}
		e.Quantity = r[1]
		e.ProductID, e.DiscountCode = "?", "?"
		for _, p := range u.Catalog.Products {
			if p.NumericID == r[0] {
				e.ProductID = p.ProductID
			}
		}
		for _, d := range u.Catalog.Discounts {
			if d.NumericID == r[2] {
				e.DiscountCode = d.Code
			}
		}
		out = append(out, e)
	}
	return out, nil
}

// IsCode reports whether err is an API error with the given status and code.
func IsCode(err error, status int, code string) bool {
	var e *apiclient.Error
	return errors.As(err, &e) && e.Status == status && e.Code == code
}
