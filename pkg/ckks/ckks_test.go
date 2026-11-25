package ckks

import (
	"errors"
	"math"
	"testing"

	"github.com/filbertengyo/sisterpedia/pkg/codec"
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
)

func registry(t *testing.T) *Registry {
	t.Helper()
	reg, err := DefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func keys(t *testing.T, profile string) KeySet {
	t.Helper()
	params, err := registry(t).Params(profile)
	if err != nil {
		t.Fatal(err)
	}
	return GenKeys(params)
}

func TestProfilesLoaded(t *testing.T) {
	reg := registry(t)
	names := reg.Names()
	if len(names) < 2 || names[0] != "p1" || names[1] != "p2" {
		t.Fatalf("profiles = %v", names)
	}
	for _, n := range names {
		info, err := reg.Info(n)
		if err != nil {
			t.Fatal(err)
		}
		if info.MaxLevel < MinLevel {
			t.Errorf("%s: max level %d", n, info.MaxLevel)
		}
		// q0 = 60 bits, scale 2^40 => about 2^19
		if info.MaxValue < 500000 || info.MaxValue > 600000 {
			t.Errorf("%s: MaxValue = %v", n, info.MaxValue)
		}
	}
	if _, err := reg.Params("nope"); !errors.Is(err, ErrUnknownProfile) {
		t.Errorf("unknown profile: %v", err)
	}
}

func TestParseProfilesRejectsInsecureAndShallow(t *testing.T) {
	insecure := []byte("profiles:\n  x:\n    log_n: 12\n    log_q: [60, 40, 40]\n    log_p: [61]\n    log_default_scale: 40\n")
	if _, err := ParseProfiles(insecure); err == nil {
		t.Error("LogN=12 with 201-bit QP must be rejected (128-bit bound is 109)")
	}
	shallow := []byte("profiles:\n  x:\n    log_n: 13\n    log_q: [60, 40]\n    log_p: [61]\n    log_default_scale: 40\n")
	if _, err := ParseProfiles(shallow); err == nil {
		t.Error("a profile with one level must be rejected")
	}
}

func TestMatchParams(t *testing.T) {
	reg := registry(t)
	for _, n := range reg.Names() {
		params, _ := reg.Params(n)
		b, err := MarshalParams(params)
		if err != nil {
			t.Fatal(err)
		}
		got, _, err := reg.MatchParams(b)
		if err != nil || got != n {
			t.Fatalf("MatchParams(%s) = %q, %v", n, got, err)
		}
	}
	if _, _, err := reg.MatchParams([]byte(`{"LogN":13}`)); !errors.Is(err, ErrParamsMismatch) {
		t.Errorf("partial literal accepted: %v", err)
	}
	if _, _, err := reg.MatchParams([]byte(`not json`)); !errors.Is(err, ErrParamsMismatch) {
		t.Errorf("garbage accepted: %v", err)
	}
	only, _ := reg.Restrict([]string{"p1"})
	p2, _ := reg.Params("p2")
	b, _ := MarshalParams(p2)
	if _, _, err := only.MatchParams(b); !errors.Is(err, ErrParamsMismatch) {
		t.Errorf("p2 accepted by p1-only registry: %v", err)
	}
}

// The reference example of the assignment: p=12500, q=2, d=0.1, k=500 gives
// t=22000; balance 100000 -> 78000.
func TestReferenceTransaction(t *testing.T) {
	for _, profile := range []string{"p1", "p2"} {
		t.Run(profile, func(t *testing.T) {
			ks := keys(t, profile)
			cli := NewClient(ks.Params, ks.SK, ks.PK)
			ev := NewEvaluator(ks.Params, ks.PK, ks.RLK)
			L := ks.Params.MaxLevel()

			cb, err := cli.EncryptScalar(0)
			if err != nil {
				t.Fatal(err)
			}
			camt, _ := cli.EncryptScalar(100000)
			if cb, err = ev.Topup(cb, camt, nil); err != nil {
				t.Fatal(err)
			}
			cq, _ := cli.EncryptScalar(2)
			var tr Trace
			ct, err := ev.Bill(cq, BillTerms{Price: []float64{12500}, Factor: []float64{0.9}, Fixed: []float64{500}}, &tr)
			if err != nil {
				t.Fatal(err)
			}
			if ct.Level() != L-2 {
				t.Errorf("bill level %d, want %d", ct.Level(), L-2)
			}
			if err := RequireDefaultScale(ks.Params, ct); err != nil {
				t.Errorf("bill scale drifted: %v", err)
			}
			if cb, err = ev.Debit(cb, ct, &tr); err != nil {
				t.Fatal(err)
			}
			total, _ := cli.DecryptScalar(ct)
			bal, _ := cli.DecryptScalar(cb)
			if math.Abs(total-22000) > 1e-2 || math.Abs(bal-78000) > 1e-2 {
				t.Fatalf("total=%v balance=%v", total, bal)
			}
			want := []string{"encrypt_price", "encrypt_discount", "encrypt_fixed", "mul_1", "relin_1", "rescale_1",
				"align_discount", "mul_2", "relin_2", "rescale_2", "align_sub_fixed", "sub_fixed", "align_sub_balance", "sub_balance"}
			if len(tr.Steps) != len(want) {
				t.Fatalf("trace has %d steps: %+v", len(tr.Steps), tr.Steps)
			}
			for i, s := range tr.Steps {
				if s.Op != want[i] {
					t.Errorf("step %d = %s, want %s", i, s.Op, want[i])
				}
			}
			// repeated transactions keep the balance at level L-2 (no extra depth)
			for i := 0; i < 3; i++ {
				cq, _ := cli.EncryptScalar(1)
				ct, err := ev.Bill(cq, BillTerms{Price: []float64{5000}, Factor: []float64{1}, Fixed: []float64{0}}, nil)
				if err != nil {
					t.Fatal(err)
				}
				if cb, err = ev.Debit(cb, ct, nil); err != nil {
					t.Fatal(err)
				}
				topup, _ := cli.EncryptScalar(1000.5)
				if cb, err = ev.Topup(cb, topup, nil); err != nil {
					t.Fatal(err)
				}
			}
			if cb.Level() != L-2 {
				t.Errorf("balance level %d after repeats, want %d", cb.Level(), L-2)
			}
			bal, _ = cli.DecryptScalar(cb)
			if math.Abs(bal-(78000-15000+3001.5)) > 1e-2 {
				t.Errorf("balance after repeats = %v", bal)
			}
		})
	}
}

func TestSerializationRoundTripAndLoaders(t *testing.T) {
	reg := registry(t)
	ks := keys(t, "p1")
	ser, err := ks.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	name, ks2, err := LoadKeySet(reg, ser)
	if err != nil || name != "p1" {
		t.Fatalf("LoadKeySet: %q %v", name, err)
	}
	if err := ValidateKeys(ks2.Params, ks2.SK, ks2.PK, ks2.RLK); err != nil {
		t.Fatalf("ValidateKeys: %v", err)
	}

	cli := NewClient(ks.Params, ks.SK, ks.PK)
	ct, _ := cli.EncryptScalar(42.5)
	b, _ := Marshal(ct)
	ct2, err := LoadCiphertext(ks.Params, b)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := NewClient(ks2.Params, ks2.SK, nil).DecryptScalar(ct2); math.Abs(v-42.5) > 1e-3 {
		t.Errorf("decrypted %v", v)
	}

	// a secret key in place of a public key, relin key or ciphertext is detected
	if _, err := LoadPublicKey(ks.Params, ser.SK); !errors.Is(err, ErrPrivateKeyMaterial) {
		t.Errorf("sk as pk: %v", err)
	}
	if _, err := LoadRelinKey(ks.Params, ser.SK); !errors.Is(err, ErrPrivateKeyMaterial) {
		t.Errorf("sk as rlk: %v", err)
	}
	if _, err := LoadCiphertext(ks.Params, ser.SK); !errors.Is(err, ErrPrivateKeyMaterial) {
		t.Errorf("sk as ciphertext: %v", err)
	}
	if !IsPrivateKeyBlob(ks.Params, ser.SK) || IsPrivateKeyBlob(ks.Params, ser.PK) {
		t.Error("IsPrivateKeyBlob misclassifies")
	}

	// garbage / truncated / corrupted headers are rejected without panics
	bad := [][]byte{[]byte("hello world"), b[:len(b)-1], append([]byte{}, b...)}
	bad[2][1] ^= 0xFF // breaks the metadata JSON
	huge := append([]byte{}, ser.PK...)
	for i := 0; i < 8; i++ {
		huge[i] = 0x7F // absurd length prefix
	}
	for i, x := range bad {
		if _, err := LoadCiphertext(ks.Params, x); err == nil {
			t.Errorf("bad ciphertext %d accepted", i)
		}
	}
	if _, err := LoadPublicKey(ks.Params, huge); !errors.Is(err, codec.ErrUnloadable) {
		t.Errorf("pk with absurd length prefix: %v", err)
	}
	// a p2 public key does not load as p1
	p2 := keys(t, "p2")
	pkp2, _ := Marshal(p2.PK)
	if _, err := LoadPublicKey(ks.Params, pkp2); err == nil {
		t.Error("p2 public key accepted for p1")
	}
}

func TestByteFlipStillLoads(t *testing.T) {
	ks := keys(t, "p1")
	cli := NewClient(ks.Params, ks.SK, ks.PK)
	ct, _ := cli.EncryptScalar(1000)
	b, _ := Marshal(ct)
	b[len(b)-100] ^= 0x01 // a coefficient byte: still structurally valid
	ct2, err := LoadCiphertext(ks.Params, b)
	if err != nil {
		t.Fatalf("flipped ciphertext should still load (CKKS has no integrity): %v", err)
	}
	v, _ := cli.DecryptScalar(ct2)
	t.Logf("decrypt after 1-byte flip: %v (original 1000)", v)
}

func TestValidateKeysDetectsMismatch(t *testing.T) {
	a := keys(t, "p1")
	b := keys(t, "p1")
	if err := ValidateKeys(a.Params, a.SK, b.PK, nil); !errors.Is(err, ErrKeysMismatch) {
		t.Errorf("foreign pk: %v", err)
	}
	if err := ValidateKeys(a.Params, a.SK, a.PK, b.RLK); !errors.Is(err, ErrKeysMismatch) {
		t.Errorf("foreign rlk: %v", err)
	}
}

func TestRangeAndScaleChecks(t *testing.T) {
	ks := keys(t, "p1")
	cli := NewClient(ks.Params, ks.SK, ks.PK)
	if _, err := cli.EncryptScalar(1e7); !errors.Is(err, ErrValueOutOfRange) {
		t.Errorf("1e7: %v", err)
	}
	if _, err := cli.EncryptScalar(math.NaN()); !errors.Is(err, ErrValueOutOfRange) {
		t.Errorf("NaN: %v", err)
	}
	ev := NewEvaluator(ks.Params, ks.PK, ks.RLK)
	odd, err := ev.EncryptConst([]float64{1}, ks.Params.MaxLevel(), rlwe.NewScale(math.Pow(2, 30)))
	if err != nil {
		t.Fatal(err)
	}
	cb, _ := cli.EncryptScalar(0)
	if _, err := ev.Topup(cb, odd, nil); !errors.Is(err, ErrEvaluation) {
		t.Errorf("topup with foreign scale: %v", err)
	}
	if _, err := ev.Bill(odd, BillTerms{Price: []float64{1}, Factor: []float64{1}, Fixed: []float64{0}}, nil); !errors.Is(err, ErrEvaluation) {
		t.Errorf("bill with foreign scale: %v", err)
	}
}
