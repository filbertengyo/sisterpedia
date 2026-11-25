package ckks

import (
	"errors"
	"fmt"
	"time"

	"github.com/filbertengyo/sisterpedia/pkg/monoclock"
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	lckks "github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

// ErrEvaluation is returned when the library rejects a homomorphic operation
// (level exhausted, wrong degree, scale mismatch, ...).
var ErrEvaluation = errors.New("homomorphic evaluation failed")

// Step records one homomorphic operation: its name, the level and log2
// scale of its output, and how long it took.
type Step struct {
	Op        string        `json:"op"`
	Level     int           `json:"level"`
	Log2Scale float64       `json:"log2_scale"`
	Duration  time.Duration `json:"duration_ns"`
}

// Trace collects the steps of an evaluation. A nil *Trace records nothing.
type Trace struct {
	Steps []Step
}

func (t *Trace) add(op string, ct *rlwe.Ciphertext, d time.Duration) {
	if t == nil || ct == nil {
		return
	}
	t.Steps = append(t.Steps, Step{Op: op, Level: ct.Level(), Log2Scale: ct.Scale.Log2(), Duration: d})
}

// Total returns the summed duration of all steps.
func (t *Trace) Total() time.Duration {
	if t == nil {
		return 0
	}
	var d time.Duration
	for _, s := range t.Steps {
		d += s.Duration
	}
	return d
}

// Evaluator performs server-side homomorphic operations for one account.
// It holds only public material: the public key (to encrypt catalog
// constants) and the relinearization key. It is not safe for concurrent use.
type Evaluator struct {
	params lckks.Parameters
	ecd    *lckks.Encoder
	enc    *rlwe.Encryptor
	eval   *lckks.Evaluator
}

// NewEvaluator builds an evaluator from an account's public artifacts.
func NewEvaluator(params lckks.Parameters, pk *rlwe.PublicKey, rlk *rlwe.RelinearizationKey) *Evaluator {
	return &Evaluator{
		params: params,
		ecd:    lckks.NewEncoder(params),
		enc:    lckks.NewEncryptor(params, pk),
		eval:   lckks.NewEvaluator(params, rlwe.NewMemEvaluationKeySet(rlk)),
	}
}

// Params returns the evaluator's parameters.
func (e *Evaluator) Params() lckks.Parameters { return e.params }

// EncryptConst encrypts values with the account's public key at the given
// level and scale (server-side encryption of catalog constants).
func (e *Evaluator) EncryptConst(values []float64, level int, scale rlwe.Scale) (*rlwe.Ciphertext, error) {
	if level < 0 || level > e.params.MaxLevel() {
		return nil, fmt.Errorf("%w: level %d out of range", ErrEvaluation, level)
	}
	pt := lckks.NewPlaintext(e.params, level)
	pt.Scale = scale
	if err := e.ecd.Encode(values, pt); err != nil {
		return nil, fmt.Errorf("%w: encode: %w", ErrEvaluation, err)
	}
	ct, err := e.enc.EncryptNew(pt)
	if err != nil {
		return nil, fmt.Errorf("%w: encrypt: %w", ErrEvaluation, err)
	}
	return ct, nil
}

// PrimeScale returns the scale equal to the prime q_level. Encoding an
// operand with this scale before a multiplication at that level makes the
// following rescale return exactly to the other operand's scale.
func (e *Evaluator) PrimeScale(level int) rlwe.Scale {
	return rlwe.NewScale(e.params.Q()[level])
}

// DropTo returns ct lowered to level (modulus switching without rescaling;
// the scale is unchanged). ct itself is never modified.
func (e *Evaluator) DropTo(ct *rlwe.Ciphertext, level int) (*rlwe.Ciphertext, error) {
	if level > ct.Level() || level < 0 {
		return nil, fmt.Errorf("%w: cannot drop from level %d to %d", ErrEvaluation, ct.Level(), level)
	}
	if level == ct.Level() {
		return ct, nil
	}
	return e.eval.DropLevelNew(ct, ct.Level()-level), nil
}

// Align brings two ciphertexts to the same (lowest) level and checks that
// their scales are exactly equal, as required before Add/Sub.
func (e *Evaluator) Align(a, b *rlwe.Ciphertext) (*rlwe.Ciphertext, *rlwe.Ciphertext, error) {
	if a.Degree() != 1 || b.Degree() != 1 {
		return nil, nil, fmt.Errorf("%w: operands must be degree 1", ErrEvaluation)
	}
	level := min(a.Level(), b.Level())
	var err error
	if a, err = e.DropTo(a, level); err != nil {
		return nil, nil, err
	}
	if b, err = e.DropTo(b, level); err != nil {
		return nil, nil, err
	}
	if !a.Scale.Equal(b.Scale) {
		return nil, nil, fmt.Errorf("%w: scale mismatch 2^%.4f vs 2^%.4f", ErrEvaluation, a.Scale.Log2(), b.Scale.Log2())
	}
	return a, b, nil
}

// Add returns a (+) b after alignment.
func (e *Evaluator) Add(a, b *rlwe.Ciphertext, tr *Trace) (*rlwe.Ciphertext, error) {
	start := monoclock.Now()
	a, b, err := e.Align(a, b)
	if err != nil {
		return nil, err
	}
	tr.add("align", a, monoclock.Since(start))
	start = monoclock.Now()
	out, err := e.eval.AddNew(a, b)
	if err != nil {
		return nil, fmt.Errorf("%w: add: %w", ErrEvaluation, err)
	}
	tr.add("add", out, monoclock.Since(start))
	return out, nil
}

// Sub returns a (-) b after alignment.
func (e *Evaluator) Sub(a, b *rlwe.Ciphertext, tr *Trace, op string) (*rlwe.Ciphertext, error) {
	if op == "" {
		op = "sub"
	}
	start := monoclock.Now()
	a, b, err := e.Align(a, b)
	if err != nil {
		return nil, err
	}
	tr.add("align_"+op, a, monoclock.Since(start))
	start = monoclock.Now()
	out, err := e.eval.SubNew(a, b)
	if err != nil {
		return nil, fmt.Errorf("%w: sub: %w", ErrEvaluation, err)
	}
	tr.add(op, out, monoclock.Since(start))
	return out, nil
}

// Mul multiplies two degree-1 ciphertexts at the same level, producing a
// degree-2 ciphertext (no relinearization, no rescaling).
func (e *Evaluator) Mul(a, b *rlwe.Ciphertext) (*rlwe.Ciphertext, error) {
	if a.Degree() != 1 || b.Degree() != 1 {
		return nil, fmt.Errorf("%w: operands must be degree 1", ErrEvaluation)
	}
	if a.Level() != b.Level() {
		return nil, fmt.Errorf("%w: operands at levels %d and %d", ErrEvaluation, a.Level(), b.Level())
	}
	out, err := e.eval.MulNew(a, b)
	if err != nil {
		return nil, fmt.Errorf("%w: mul: %w", ErrEvaluation, err)
	}
	return out, nil
}

// Relinearize maps a degree-2 ciphertext back to degree 1 with the rlk.
func (e *Evaluator) Relinearize(ct *rlwe.Ciphertext) (*rlwe.Ciphertext, error) {
	out := lckks.NewCiphertext(e.params, 1, ct.Level())
	if err := e.eval.Relinearize(ct, out); err != nil {
		return nil, fmt.Errorf("%w: relinearize: %w", ErrEvaluation, err)
	}
	return out, nil
}

// Rescale divides by the last prime of the current level (consumes a level).
func (e *Evaluator) Rescale(ct *rlwe.Ciphertext) (*rlwe.Ciphertext, error) {
	out := lckks.NewCiphertext(e.params, ct.Degree(), ct.Level())
	if err := e.eval.Rescale(ct, out); err != nil {
		return nil, fmt.Errorf("%w: rescale: %w", ErrEvaluation, err)
	}
	return out, nil
}

// MulRelinRescale computes a (x) b: multiplication, relinearization and
// rescaling, each recorded in tr under the given suffix.
func (e *Evaluator) MulRelinRescale(a, b *rlwe.Ciphertext, tr *Trace, suffix ...string) (*rlwe.Ciphertext, error) {
	sfx := ""
	if len(suffix) > 0 {
		sfx = suffix[0]
	}
	start := monoclock.Now()
	prod, err := e.Mul(a, b)
	if err != nil {
		return nil, err
	}
	tr.add("mul"+sfx, prod, monoclock.Since(start))

	start = monoclock.Now()
	lin, err := e.Relinearize(prod)
	if err != nil {
		return nil, err
	}
	tr.add("relin"+sfx, lin, monoclock.Since(start))

	start = monoclock.Now()
	out, err := e.Rescale(lin)
	if err != nil {
		return nil, err
	}
	tr.add("rescale"+sfx, out, monoclock.Since(start))
	return out, nil
}

// BillTerms are the plaintext catalog values the server knows: unit price p,
// discount factor (1 - d) and fixed reduction k, one entry per slot.
type BillTerms struct {
	Price  []float64
	Factor []float64
	Fixed  []float64
}

// Bill evaluates t = (p (x) q (x) (1-d)) (-) k on an encrypted quantity
// (Algorithm 1 of the assignment):
//
//	c_p = Enc_pk(p)      at level L, scale q_L
//	c_u = c_p (x) c_q    mul + relin + rescale -> level L-1, scale exactly Delta
//	c_d = Enc_pk(1-d)    at level L, scale q_{L-1}, dropped to level L-1
//	c_v = c_u (x) c_d    mul + relin + rescale -> level L-2, scale exactly Delta
//	c_k = Enc_pk(k)      at level L, scale Delta, dropped to level L-2
//	c_t = c_v (-) c_k
//
// Encoding each multiplicand at the scale of the prime that the following
// rescale removes keeps every result at exactly the default scale Delta, so
// later additions/subtractions never need scale adjustment. The quantity
// ciphertext must be fresh (level L, scale Delta). No secret key is used.
func (e *Evaluator) Bill(cq *rlwe.Ciphertext, terms BillTerms, tr *Trace) (*rlwe.Ciphertext, error) {
	L := cq.Level()
	if L < MinLevel {
		return nil, fmt.Errorf("%w: quantity ciphertext at level %d, need >= %d", ErrEvaluation, L, MinLevel)
	}
	if err := RequireDefaultScale(e.params, cq); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEvaluation, err)
	}

	start := monoclock.Now()
	cp, err := e.EncryptConst(terms.Price, L, e.PrimeScale(L))
	if err != nil {
		return nil, err
	}
	tr.add("encrypt_price", cp, monoclock.Since(start))

	start = monoclock.Now()
	cd, err := e.EncryptConst(terms.Factor, L, e.PrimeScale(L-1))
	if err != nil {
		return nil, err
	}
	tr.add("encrypt_discount", cd, monoclock.Since(start))

	start = monoclock.Now()
	ck, err := e.EncryptConst(terms.Fixed, L, e.params.DefaultScale())
	if err != nil {
		return nil, err
	}
	tr.add("encrypt_fixed", ck, monoclock.Since(start))

	cu, err := e.MulRelinRescale(cp, cq, tr, "_1")
	if err != nil {
		return nil, err
	}

	start = monoclock.Now()
	if cd, err = e.DropTo(cd, cu.Level()); err != nil {
		return nil, err
	}
	tr.add("align_discount", cd, monoclock.Since(start))

	cv, err := e.MulRelinRescale(cu, cd, tr, "_2")
	if err != nil {
		return nil, err
	}

	return e.Sub(cv, ck, tr, "sub_fixed")
}

// Topup returns c_b (+) c_amount. The amount must be at the default scale.
func (e *Evaluator) Topup(cb, camount *rlwe.Ciphertext, tr *Trace) (*rlwe.Ciphertext, error) {
	if err := RequireDefaultScale(e.params, camount); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEvaluation, err)
	}
	return e.Add(cb, camount, tr)
}

// Debit returns c_b (-) c_t (balance update of a transaction).
func (e *Evaluator) Debit(cb, ct *rlwe.Ciphertext, tr *Trace) (*rlwe.Ciphertext, error) {
	return e.Sub(cb, ct, tr, "sub_balance")
}
