//go:build e2e

package e2e

import (
	"fmt"
	"math"
	"math/big"
	"testing"
)

type chainOp struct {
	Op           string `json:"op"`
	Amount       string `json:"amount,omitempty"`
	ProductID    string `json:"product_id,omitempty"`
	Quantity     int    `json:"quantity,omitempty"`
	DiscountCode string `json:"discount_code,omitempty"`
}

type chainDef struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Ops         []chainOp `json:"ops"`
}

// TestChains runs the five fixed transaction chains of tests/data/chains.json
// on every profile and compares the decrypted balance with the exact
// reference after every balance change (C2.4).
func TestChains(t *testing.T) {
	var data struct {
		Chains []chainDef `json:"chains"`
	}
	loadJSON(t, "chains.json", &data)
	if len(data.Chains) < 5 {
		t.Fatalf("need >= 5 chains, have %d", len(data.Chains))
	}
	forProfiles(t, func(t *testing.T, profile string) {
		for ci, ch := range data.Chains {
			t.Run(ch.Name, func(t *testing.T) {
				u := activeUser(t, uid(fmt.Sprintf("ch%d", ci+1), profile), profile)
				if err := u.LoadCatalog(ctx()); err != nil {
					t.Fatal(err)
				}
				sum := ChainSummary{Profile: profile, Chain: ch.Name, Description: ch.Description, Ops: len(ch.Ops), Pass: true}
				bal := rat("0")
				for step, op := range ch.Ops {
					var actual float64
					var label string
					switch op.Op {
					case "topup":
						sum.Topups++
						label = "topup " + op.Amount
						b, err := u.Topup(ctx(), amount(op.Amount))
						if err != nil {
							t.Fatalf("step %d: %v", step+1, err)
						}
						bal.Add(bal, rat(op.Amount))
						actual = b
					case "transact":
						code := op.DiscountCode
						if code == "" {
							code = "NONE"
						}
						if code == "NONE" {
							sum.NoDiscount++
						} else {
							sum.WithDiscnt++
						}
						label = fmt.Sprintf("transact %s x%d %s", op.ProductID, op.Quantity, code)
						p, d, k := catalogTerms(t, u.Catalog, op.ProductID, code)
						bill := refBill(p, op.Quantity, d, k)
						res, err := u.Transact(ctx(), op.ProductID, op.Quantity, code)
						if err != nil {
							t.Fatalf("step %d: %v", step+1, err)
						}
						bal.Sub(bal, bill)
						actual = res.Balance
						te := errOf(bill, res.Total)
						rec.Lock()
						rec.totalRows = append(rec.totalRows, ChainRow{Profile: profile, Chain: ch.Name, Step: step + 1, Op: label, Expected: bill, Actual: res.Total, Err: te})
						rec.Unlock()
						if te.Abs > ToleranceMoney {
							sum.Pass = false
						}
					default:
						t.Fatalf("unknown op %q", op.Op)
					}
					e := errOf(bal, actual)
					rec.Lock()
					rec.chainRows = append(rec.chainRows, ChainRow{Profile: profile, Chain: ch.Name, Step: step + 1, Op: label, Expected: new(big.Rat).Set(bal), Actual: actual, Err: e})
					rec.Unlock()
					sum.MaxAbs = math.Max(sum.MaxAbs, e.Abs)
					if e.RelDefined {
						sum.MaxRel = math.Max(sum.MaxRel, e.Rel)
					}
					if e.Abs > ToleranceMoney {
						sum.Pass = false
						t.Errorf("%s step %d (%s): expected %s, got %.10f (e_abs %s)", ch.Name, step+1, label, bal.FloatString(2), actual, fmtE(e.Abs))
					}
				}
				rec.Lock()
				rec.summaries = append(rec.summaries, sum)
				rec.Unlock()
			})
		}
	})
}
