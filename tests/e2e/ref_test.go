//go:build e2e

package e2e

import (
	"math/big"
	"testing"
)

// Reference arithmetic with exact rationals (math/big), so measured
// differences reflect only the CKKS approximation error.

func rat(s string) *big.Rat {
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		panic("bad rational " + s)
	}
	return r
}

func ratF(f float64) *big.Rat { return new(big.Rat).SetFloat64(f) }

func ratI(i int) *big.Rat { return new(big.Rat).SetInt64(int64(i)) }

// refBill computes t = p * q * (1 - d) - k exactly.
func refBill(p *big.Rat, q int, d, k *big.Rat) *big.Rat {
	one := big.NewRat(1, 1)
	t := new(big.Rat).Mul(p, ratI(q))
	t.Mul(t, new(big.Rat).Sub(one, d))
	return t.Sub(t, k)
}

// ErrResult holds e_abs = |y - y_hat| and e_rel = e_abs / |y| (undefined for y = 0).
type ErrResult struct {
	Abs        float64
	Rel        float64
	RelDefined bool
}

// errOf compares the raw decrypted value with the exact reference.
func errOf(y *big.Rat, yhat float64) ErrResult {
	diff := new(big.Rat).Sub(ratF(yhat), y)
	abs, _ := new(big.Rat).Abs(diff).Float64()
	if y.Sign() == 0 {
		return ErrResult{Abs: abs}
	}
	rel, _ := new(big.Rat).Quo(new(big.Rat).Abs(diff), new(big.Rat).Abs(y)).Float64()
	return ErrResult{Abs: abs, Rel: rel, RelDefined: true}
}

func (e ErrResult) RelString() string {
	if !e.RelDefined {
		return "n/a"
	}
	return fmtE(e.Rel)
}

func TestReferenceArithmetic(t *testing.T) {
	// spec example: 12500 x 2 x (1 - 0.1) - 500 = 22000
	if got := refBill(rat("12500"), 2, rat("0.1"), rat("500")); got.Cmp(rat("22000")) != 0 {
		t.Fatalf("refBill = %s", got.FloatString(6))
	}
	e := errOf(rat("0"), 1e-6)
	if e.RelDefined || e.RelString() != "n/a" || e.Abs != 1e-6 {
		t.Fatalf("y=0: %+v", e)
	}
	e = errOf(rat("22000"), 22000.0022)
	if !e.RelDefined || e.Rel < 9.9e-8 || e.Rel > 1.01e-7 {
		t.Fatalf("rel: %+v", e)
	}
}
