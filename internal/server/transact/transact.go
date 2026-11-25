// Package transact implements the homomorphic business logic of the shop on
// ciphertexts only: Topup (c_b (+) c_amount) and Transact (bill evaluation
// with two ciphertext-ciphertext multiplications, then c_b (-) c_t).
// It never has access to a secret key and never decrypts.
package transact

import (
	"github.com/filbertengyo/sisterpedia/internal/server/catalog"
	"github.com/filbertengyo/sisterpedia/pkg/ckks"
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
)

// Terms converts a product and a discount rule into the plaintext bill terms
// the server encrypts under the user's public key: p, (1 - d) and k.
func Terms(p catalog.Product, d catalog.Discount) ckks.BillTerms {
	return ckks.BillTerms{
		Price:  []float64{p.BasePrice},
		Factor: []float64{1 - d.Percent},
		Fixed:  []float64{d.Fixed},
	}
}

// Topup returns the new balance c_b (+) c_amount.
func Topup(ev *ckks.Evaluator, balance, amount *rlwe.Ciphertext, tr *ckks.Trace) (*rlwe.Ciphertext, error) {
	return ev.Topup(balance, amount, tr)
}

// Result is the outcome of a transaction evaluation.
type Result struct {
	Total   *rlwe.Ciphertext // c_t
	Balance *rlwe.Ciphertext // c_b' = c_b (-) c_t
}

// Transact evaluates t = (p * q * (1 - d)) - k on the encrypted quantity and
// subtracts it from the encrypted balance. There is no sufficiency check:
// the balance may become negative and only the client can see it.
func Transact(ev *ckks.Evaluator, balance, quantity *rlwe.Ciphertext, terms ckks.BillTerms, tr *ckks.Trace) (Result, error) {
	total, err := ev.Bill(quantity, terms, tr)
	if err != nil {
		return Result{}, err
	}
	nb, err := ev.Debit(balance, total, tr)
	if err != nil {
		return Result{}, err
	}
	return Result{Total: total, Balance: nb}, nil
}
