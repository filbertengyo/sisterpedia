// Package service implements the five shop services on top of the store and
// the homomorphic evaluator. Every sensitive value is a ciphertext; the
// service never sees a secret key and has no way to decrypt.
package service

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/filbertengyo/sisterpedia/internal/server/catalog"
	"github.com/filbertengyo/sisterpedia/internal/server/store"
	"github.com/filbertengyo/sisterpedia/internal/server/transact"
	"github.com/filbertengyo/sisterpedia/pkg/ckks"
	"github.com/filbertengyo/sisterpedia/pkg/codec"
	"github.com/oklog/ulid/v2"
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	lckks "github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

// Error codes of the REST contract (docs/API_CONTRACT.md).
const (
	CodeInvalidJSON        = "invalid_json"
	CodeInvalidPayload     = "invalid_payload"
	CodeContextMismatch    = "context_mismatch"
	CodePrivateKeyRejected = "secret_key_rejected"
	CodeEvaluationFailed   = "evaluation_failed"
	CodeAccountNotFound    = "account_not_found"
	CodeProductNotFound    = "product_not_found"
	CodeDiscountNotFound   = "discount_not_found"
	CodeAccountExists      = "account_exists"
	CodeInternal           = "internal_error"
)

// Error is a domain error carrying a contract error code. Detail never
// contains ciphertext or key material.
type Error struct {
	Code   string
	Detail string
}

func (e *Error) Error() string { return e.Code + ": " + e.Detail }

func fail(code, format string, args ...any) error {
	return &Error{Code: code, Detail: fmt.Sprintf(format, args...)}
}

// UserIDPattern is the accepted user_id format.
var UserIDPattern = regexp.MustCompile(`^[a-z0-9_-]{3,32}$`)

// ValidUserID reports whether id matches UserIDPattern.
func ValidUserID(id string) bool { return UserIDPattern.MatchString(id) }

// Service is the shop application layer.
type Service struct {
	reg     *ckks.Registry
	store   *store.Store
	catalog *catalog.Catalog
	maxB64  int
	now     func() time.Time

	locksMu sync.Mutex
	locks   map[string]*sync.Mutex

	cache *evalCache
}

// New builds a service. reg must contain only the allowed profiles.
func New(reg *ckks.Registry, st *store.Store, cat *catalog.Catalog, maxB64 int) *Service {
	return &Service{
		reg: reg, store: st, catalog: cat, maxB64: maxB64,
		now:   func() time.Time { return time.Now().UTC() },
		locks: map[string]*sync.Mutex{},
		cache: newEvalCache(16),
	}
}

// Catalog returns the public catalog.
func (s *Service) Catalog() *catalog.Catalog { return s.catalog }

// lock serializes all state-changing operations of one account so that
// concurrent requests cannot lose updates (read-evaluate-write).
func (s *Service) lock(userID string) func() {
	s.locksMu.Lock()
	m, ok := s.locks[userID]
	if !ok {
		m = &sync.Mutex{}
		s.locks[userID] = m
	}
	s.locksMu.Unlock()
	m.Lock()
	return m.Unlock
}

func (s *Service) decode(field, b64 string) ([]byte, error) {
	b, err := codec.Decode(b64, s.maxB64)
	if err != nil {
		return nil, fail(CodeInvalidPayload, "%s: %v", field, err)
	}
	return b, nil
}

func mapLoadErr(field string, err error) error {
	switch {
	case errors.Is(err, ckks.ErrPrivateKeyMaterial):
		return fail(CodePrivateKeyRejected, "%s contains secret key material; the secret key must never leave the client", field)
	case errors.Is(err, ckks.ErrContextMismatch):
		return fail(CodeContextMismatch, "%s: %v", field, err)
	default:
		return fail(CodeInvalidPayload, "%s: %v", field, err)
	}
}

func (s *Service) loadCiphertext(params lckks.Parameters, field, b64 string, fresh bool) (*rlwe.Ciphertext, error) {
	b, err := s.decode(field, b64)
	if err != nil {
		return nil, err
	}
	ct, err := ckks.LoadCiphertext(params, b)
	if err != nil {
		return nil, mapLoadErr(field, err)
	}
	if err := ckks.RequireDefaultScale(params, ct); err != nil {
		return nil, mapLoadErr(field, err)
	}
	if fresh && ct.Level() != params.MaxLevel() {
		return nil, fail(CodeContextMismatch, "%s must be a fresh ciphertext at level %d (got %d)", field, params.MaxLevel(), ct.Level())
	}
	return ct, nil
}

// ActivateInput is the Activate request (Base64 fields).
type ActivateInput struct {
	UserID, Params, PublicKey, RelinKey, EncBalance string
}

// Activate registers an account with its public artifacts and Enc(0).
func (s *Service) Activate(ctx context.Context, in ActivateInput) (string, error) {
	unlock := s.lock(in.UserID)
	defer unlock()

	if exists, err := s.store.Exists(ctx, in.UserID); err != nil {
		return "", fail(CodeInternal, "store: %v", err)
	} else if exists {
		return "", fail(CodeAccountExists, "account %q is already activated", in.UserID)
	}

	paramsBin, err := s.decode("params", in.Params)
	if err != nil {
		return "", err
	}
	profile, params, err := s.reg.MatchParams(paramsBin)
	if err != nil {
		return "", fail(CodeContextMismatch, "params do not equal an allowed profile %v", s.reg.Names())
	}

	pkBin, err := s.decode("public_key", in.PublicKey)
	if err != nil {
		return "", err
	}
	if _, err := ckks.LoadPublicKey(params, pkBin); err != nil {
		return "", mapLoadErr("public_key", err)
	}
	rlkBin, err := s.decode("relin_key", in.RelinKey)
	if err != nil {
		return "", err
	}
	if _, err := ckks.LoadRelinKey(params, rlkBin); err != nil {
		return "", mapLoadErr("relin_key", err)
	}
	ct, err := s.loadCiphertext(params, "enc_balance", in.EncBalance, false)
	if err != nil {
		return "", err
	}
	balBin, err := ct.MarshalBinary()
	if err != nil {
		return "", fail(CodeInternal, "marshal: %v", err)
	}

	err = s.store.CreateAccount(ctx, store.Account{
		UserID: in.UserID, Profile: profile, Params: paramsBin,
		PublicKey: pkBin, RelinKey: rlkBin, EncBalance: balBin, CreatedAt: s.now(),
	})
	if errors.Is(err, store.ErrExists) {
		return "", fail(CodeAccountExists, "account %q is already activated", in.UserID)
	}
	if err != nil {
		return "", fail(CodeInternal, "store: %v", err)
	}
	return codec.Encode(balBin), nil
}

type account struct {
	params lckks.Parameters
	ev     *ckks.Evaluator
	rec    store.Account
}

// account loads an account and its (cached) evaluator.
func (s *Service) account(ctx context.Context, userID string) (*account, error) {
	rec, err := s.store.GetAccount(ctx, userID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, fail(CodeAccountNotFound, "account %q is not activated", userID)
	}
	if err != nil {
		return nil, fail(CodeInternal, "store: %v", err)
	}
	params, err := s.reg.Params(rec.Profile)
	if err != nil {
		return nil, fail(CodeContextMismatch, "account profile %q is no longer allowed", rec.Profile)
	}
	if ev, ok := s.cache.get(userID); ok {
		return &account{params: params, ev: ev, rec: rec}, nil
	}
	pk, err := ckks.LoadPublicKey(params, rec.PublicKey)
	if err != nil {
		return nil, fail(CodeInternal, "stored public key: %v", err)
	}
	rlk, err := ckks.LoadRelinKey(params, rec.RelinKey)
	if err != nil {
		return nil, fail(CodeInternal, "stored relinearization key: %v", err)
	}
	ev := ckks.NewEvaluator(params, pk, rlk)
	s.cache.put(userID, ev)
	return &account{params: params, ev: ev, rec: rec}, nil
}

func (s *Service) storedBalance(a *account) (*rlwe.Ciphertext, error) {
	ct, err := ckks.LoadCiphertext(a.params, a.rec.EncBalance)
	if err != nil {
		return nil, fail(CodeInternal, "stored balance: %v", err)
	}
	return ct, nil
}

// Balance returns the stored balance ciphertext (Base64).
func (s *Service) Balance(ctx context.Context, userID string) (string, error) {
	b, err := s.store.GetBalance(ctx, userID)
	if errors.Is(err, store.ErrNotFound) {
		return "", fail(CodeAccountNotFound, "account %q is not activated", userID)
	}
	if err != nil {
		return "", fail(CodeInternal, "store: %v", err)
	}
	return codec.Encode(b), nil
}

// Topup adds the encrypted amount to the encrypted balance.
func (s *Service) Topup(ctx context.Context, userID, encAmount string) (string, *ckks.Trace, error) {
	unlock := s.lock(userID)
	defer unlock()

	a, err := s.account(ctx, userID)
	if err != nil {
		return "", nil, err
	}
	amount, err := s.loadCiphertext(a.params, "enc_amount", encAmount, false)
	if err != nil {
		return "", nil, err
	}
	bal, err := s.storedBalance(a)
	if err != nil {
		return "", nil, err
	}
	tr := &ckks.Trace{}
	nb, err := transact.Topup(a.ev, bal, amount, tr)
	if err != nil {
		return "", nil, fail(CodeEvaluationFailed, "%v", err)
	}
	nbBin, err := nb.MarshalBinary()
	if err != nil {
		return "", nil, fail(CodeInternal, "marshal: %v", err)
	}
	if err := s.store.UpdateBalance(ctx, userID, nbBin, s.now()); err != nil {
		return "", nil, fail(CodeInternal, "store: %v", err)
	}
	return codec.Encode(nbBin), tr, nil
}

// TransactInput is the Transact request.
type TransactInput struct {
	UserID, ProductID, DiscountCode, EncQuantity, EncMetadata string
}

// TransactOutput is the Transact response.
type TransactOutput struct {
	TransactionID string
	EncTotal      string
	EncBalance    string
	Trace         *ckks.Trace
}

// Transact computes the bill homomorphically, debits the balance and appends
// the history entry atomically.
func (s *Service) Transact(ctx context.Context, in TransactInput) (TransactOutput, error) {
	unlock := s.lock(in.UserID)
	defer unlock()

	a, err := s.account(ctx, in.UserID)
	if err != nil {
		return TransactOutput{}, err
	}
	product, err := s.catalog.Product(in.ProductID)
	if err != nil {
		return TransactOutput{}, fail(CodeProductNotFound, "product %q is not in the catalog", in.ProductID)
	}
	discount, err := s.catalog.Discount(in.DiscountCode)
	if err != nil {
		return TransactOutput{}, fail(CodeDiscountNotFound, "discount %q is not in the catalog", in.DiscountCode)
	}
	cq, err := s.loadCiphertext(a.params, "enc_quantity", in.EncQuantity, true)
	if err != nil {
		return TransactOutput{}, err
	}
	cm, err := s.loadCiphertext(a.params, "enc_metadata", in.EncMetadata, false)
	if err != nil {
		return TransactOutput{}, err
	}
	bal, err := s.storedBalance(a)
	if err != nil {
		return TransactOutput{}, err
	}

	tr := &ckks.Trace{}
	res, err := transact.Transact(a.ev, bal, cq, transact.Terms(product, discount), tr)
	if err != nil {
		return TransactOutput{}, fail(CodeEvaluationFailed, "%v", err)
	}
	totalBin, err := res.Total.MarshalBinary()
	if err != nil {
		return TransactOutput{}, fail(CodeInternal, "marshal: %v", err)
	}
	balBin, err := res.Balance.MarshalBinary()
	if err != nil {
		return TransactOutput{}, fail(CodeInternal, "marshal: %v", err)
	}
	metaBin, err := cm.MarshalBinary()
	if err != nil {
		return TransactOutput{}, fail(CodeInternal, "marshal: %v", err)
	}
	t, err := s.store.ApplyTransaction(ctx, in.UserID, balBin, store.Transaction{
		ID: ulid.Make().String(), EncMetadata: metaBin, EncTotal: totalBin, CreatedAt: s.now(),
	})
	if err != nil {
		return TransactOutput{}, fail(CodeInternal, "store: %v", err)
	}
	return TransactOutput{TransactionID: t.ID, EncTotal: codec.Encode(totalBin), EncBalance: codec.Encode(balBin), Trace: tr}, nil
}

// HistoryItem is one GetTransact entry (Base64 ciphertexts).
type HistoryItem struct {
	TransactionID string
	EncMetadata   string
	EncTotal      string
	CreatedAt     time.Time
}

// History returns the account's transactions from oldest to newest.
func (s *Service) History(ctx context.Context, userID string) ([]HistoryItem, error) {
	exists, err := s.store.Exists(ctx, userID)
	if err != nil {
		return nil, fail(CodeInternal, "store: %v", err)
	}
	if !exists {
		return nil, fail(CodeAccountNotFound, "account %q is not activated", userID)
	}
	txs, err := s.store.ListTransactions(ctx, userID)
	if err != nil {
		return nil, fail(CodeInternal, "store: %v", err)
	}
	out := make([]HistoryItem, 0, len(txs))
	for _, t := range txs {
		out = append(out, HistoryItem{
			TransactionID: t.ID, EncMetadata: codec.Encode(t.EncMetadata), EncTotal: codec.Encode(t.EncTotal), CreatedAt: t.CreatedAt,
		})
	}
	return out, nil
}

// evalCache is a small LRU of per-account evaluators (the relinearization key
// is several MiB, so reloading it on every request would be wasteful).
type evalCache struct {
	mu    sync.Mutex
	cap   int
	ll    *list.List
	items map[string]*list.Element
}

type cacheEntry struct {
	key string
	ev  *ckks.Evaluator
}

func newEvalCache(capacity int) *evalCache {
	return &evalCache{cap: capacity, ll: list.New(), items: map[string]*list.Element{}}
}

func (c *evalCache) get(key string) (*ckks.Evaluator, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.ll.MoveToFront(el)
		return el.Value.(*cacheEntry).ev, true
	}
	return nil, false
}

func (c *evalCache) put(key string, ev *ckks.Evaluator) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		el.Value.(*cacheEntry).ev = ev
		c.ll.MoveToFront(el)
		return
	}
	c.items[key] = c.ll.PushFront(&cacheEntry{key: key, ev: ev})
	for c.ll.Len() > c.cap {
		last := c.ll.Back()
		c.ll.Remove(last)
		delete(c.items, last.Value.(*cacheEntry).key)
	}
}
