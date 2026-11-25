//go:build e2e

package e2e

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/filbertengyo/sisterpedia/internal/apiclient"
	"github.com/filbertengyo/sisterpedia/internal/shopclient"
)

type txInput struct {
	ProductID    string `json:"product_id"`
	Quantity     int    `json:"quantity"`
	DiscountCode string `json:"discount_code"`
	Kind         string `json:"kind"`
}

type functionalData struct {
	U04Topups       []string  `json:"u04_topups"`
	U05Initial      string    `json:"u05_initial_topup"`
	U05Transactions []txInput `json:"u05_transactions"`
	U06Initial      string    `json:"u06_initial_topup"`
	U06Transaction  txInput   `json:"u06_transaction"`
}

func functional(t *testing.T) functionalData {
	var d functionalData
	loadJSON(t, "functional.json", &d)
	return d
}

// catalogTerms returns the exact rational price, discount and fixed reduction.
func catalogTerms(t *testing.T, cat apiclient.Catalog, productID, code string) (*big.Rat, *big.Rat, *big.Rat) {
	t.Helper()
	if code == "" {
		code = "NONE"
	}
	var p, d, k *big.Rat
	for _, x := range cat.Products {
		if x.ProductID == productID {
			p = rat(fmt.Sprint(x.BasePrice))
		}
	}
	for _, x := range cat.Discounts {
		if x.Code == code {
			d, k = rat(fmt.Sprint(x.Percent)), rat(fmt.Sprint(x.Fixed))
		}
	}
	if p == nil || d == nil {
		t.Fatalf("unknown product %s or discount %s", productID, code)
	}
	return p, d, k
}

func amount(s string) float64 {
	f, _ := new(big.Rat).SetString(s)
	v, _ := f.Float64()
	return v
}

// U01: activation with Enc(0), then CheckBalance decrypts to ~0.
func TestU01ActivationZeroBalance(t *testing.T) {
	forProfiles(t, func(t *testing.T, profile string) {
		id := uid("u01", profile)
		r := newRecord("U01", "Aktivasi dan saldo awal", profile, fmt.Sprintf("user_id=%s, saldo awal Enc(0)", id),
			fmt.Sprintf("201 Created, CheckBalance didekripsi mendekati 0 (e_abs <= %g, e_rel n/a)", ToleranceMoney))
		u := newUser(t, id, profile)
		if profile == env.profiles[0] {
			samples.expect("01-activate")
		}
		b0, err := u.Activate(ctx())
		if err != nil {
			t.Fatal(err)
		}
		r.step("Klien membangkitkan sk, pk, rlk (%s) dan mengenkripsi 0; POST /accounts tanpa secret key -> 201", profile)
		r.measure("saldo dari respons Activate", rat("0"), b0)
		if profile == env.profiles[0] {
			samples.expect("02-checkbalance")
		}
		b, err := u.Balance(ctx())
		if err != nil {
			t.Fatal(err)
		}
		r.step("GET /accounts/%s/balance -> ciphertext; dekripsi di klien", id)
		m := r.measure("saldo dari CheckBalance", rat("0"), b)
		r.Actual = fmt.Sprintf("saldo mentah %.10g (e_abs %s, e_rel %s)", b, fmtE(m.EAbs), m.ERel)
		r.Pass = r.allWithin()
		if !r.Pass {
			t.Errorf("U01 %s: %s", profile, r.Actual)
		}
	})
}

// U02: keys exported to files, server process restarted, keys re-imported
// (no new key generation): balance and history still decrypt.
func TestU02PersistenceRestart(t *testing.T) {
	forProfiles(t, func(t *testing.T, profile string) {
		id := uid("u02", profile)
		r := newRecord("U02", "Persistensi kunci dan sesi", profile,
			fmt.Sprintf("user_id=%s; topup 75000; beli P001 x2 COMBO; ekspor kunci ke params.json/secret.key/public.key/relin.key", id),
			"Setelah server dan klien dijalankan ulang serta kunci lama diimpor: saldo 53000 dan 1 riwayat tetap terdekripsi; tidak ada pembangkitan kunci baru")
		u := activeUser(t, id, profile)
		if _, err := u.Topup(ctx(), 75000); err != nil {
			t.Fatal(err)
		}
		if _, err := u.Transact(ctx(), "P001", 2, "COMBO"); err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(t.TempDir(), "keys")
		if err := u.SaveKeys(dir); err != nil {
			t.Fatal(err)
		}
		files, _ := os.ReadDir(dir)
		names := []string{}
		for _, f := range files {
			names = append(names, f.Name())
		}
		r.step("Aktivasi, topup 75000, transaksi P001 x2 COMBO; ekspor kunci ke berkas: %s", strings.Join(names, ", "))
		pkBefore, _ := u.Keys.PK.MarshalBinary()

		oldPID := env.server.PID()
		if err := env.server.Restart(); err != nil {
			t.Fatal(err)
		}
		r.step("Server dihentikan (PID %d) dan dijalankan ulang (PID %d) dengan basis data SQLite yang sama", oldPID, env.server.PID())
		u = nil // the client "closes": all in-memory keys are dropped

		u2, err := shopclient.LoadKeys(env.api, env.reg, id, dir)
		if err != nil {
			t.Fatal(err)
		}
		pkAfter, _ := u2.Keys.PK.MarshalBinary()
		same := string(pkBefore) == string(pkAfter)
		r.step("Klien dijalankan ulang: kunci dimuat dari berkas (validasi pasangan kunci), tanpa keygen; public key identik: %v", same)
		b, err := u2.Balance(ctx())
		if err != nil {
			t.Fatal(err)
		}
		r.measure("saldo setelah restart", rat("53000"), b)
		h, err := u2.History(ctx())
		if err != nil {
			t.Fatal(err)
		}
		okHist := len(h) == 1 && h[0].ProductID == "P001" && h[0].Quantity == 2 && h[0].DiscountCode == "COMBO"
		if len(h) == 1 {
			r.measure("total riwayat setelah restart", rat("22000"), h[0].Total)
		}
		r.step("CheckBalance dan GetTransact setelah restart; riwayat: %d entri", len(h))
		r.Actual = fmt.Sprintf("saldo %.6f, riwayat %d entri (%v), kunci sama: %v", b, len(h), okHist, same)
		r.Pass = r.allWithin() && okHist && same && oldPID != env.server.PID()
		if !r.Pass {
			t.Errorf("U02 %s: %s", profile, r.Actual)
		}
	})
}

// U03: two accounts with different keys stay isolated; account B's secret
// key does not recover account A's values.
func TestU03Isolation(t *testing.T) {
	forProfiles(t, func(t *testing.T, profile string) {
		ida, idb := uid("u03a", profile), uid("u03b", profile)
		r := newRecord("U03", "Isolasi antarpengguna", profile,
			fmt.Sprintf("akun A=%s topup 12345.67; akun B=%s topup 500 lalu beli P005 x1", ida, idb),
			"Saldo A dan B terpisah dan benar dengan kuncinya masing-masing; sk B pada ciphertext A tidak menghasilkan 12345.67")
		a := activeUser(t, ida, profile)
		b := activeUser(t, idb, profile)
		if _, err := a.Topup(ctx(), 12345.67); err != nil {
			t.Fatal(err)
		}
		if _, err := b.Topup(ctx(), 500); err != nil {
			t.Fatal(err)
		}
		if _, err := b.Transact(ctx(), "P005", 1, ""); err != nil {
			t.Fatal(err)
		}
		r.step("Dua akun dengan pasangan kunci berbeda; transaksi masing-masing")
		ba, _ := a.Balance(ctx())
		bb, _ := b.Balance(ctx())
		r.measure("saldo A dengan sk A", rat("12345.67"), ba)
		r.measure("saldo B dengan sk B", rat("-4500"), bb)

		resp, err := env.api.Balance(ctx(), ida)
		if err != nil {
			t.Fatal(err)
		}
		wrong, err := b.DecryptB64(resp.EncBalance)
		if err != nil {
			t.Fatal(err)
		}
		e := errOf(rat("12345.67"), wrong)
		r.step("Ciphertext saldo A didekripsi dengan sk B: hasil %.6g", wrong)
		r.note("Dekripsi dengan kunci yang salah tidak menimbulkan galat dari pustaka; hasilnya angka acak (e_abs %s terhadap 12345.67).", fmtE(e.Abs))
		r.Actual = fmt.Sprintf("A=%.6f, B=%.6f; sk B pada ciphertext A -> %.6g (salah, tanpa pesan galat)", ba, bb, wrong)
		r.Pass = r.allWithin() && e.Abs > 1
		if !r.Pass {
			t.Errorf("U03 %s: %s", profile, r.Actual)
		}
	})
}

// U04: repeated top ups (integer and decimal) compared at every step.
func TestU04RepeatedTopups(t *testing.T) {
	data := functional(t)
	forProfiles(t, func(t *testing.T, profile string) {
		id := uid("u04", profile)
		r := newRecord("U04", "Topup berulang", profile, "topup berturut-turut: "+strings.Join(data.U04Topups, ", "),
			fmt.Sprintf("Saldo hasil dekripsi pada setiap langkah = jumlah acuan (e_abs <= %g)", ToleranceMoney))
		u := activeUser(t, id, profile)
		sum := rat("0")
		for i, s := range data.U04Topups {
			if i == 0 && profile == env.profiles[0] {
				samples.expect("03-topup")
			}
			b, err := u.Topup(ctx(), amount(s))
			if err != nil {
				t.Fatal(err)
			}
			sum.Add(sum, rat(s))
			r.measure(fmt.Sprintf("saldo setelah topup #%d (+%s)", i+1, s), new(big.Rat).Set(sum), b)
		}
		r.step("%d kali POST /accounts/%s/topup dengan Enc(nominal); dekripsi saldo setelah setiap langkah", len(data.U04Topups), id)
		last := r.Measures[len(r.Measures)-1]
		r.Actual = fmt.Sprintf("saldo akhir %.8f (acuan %s), max e_abs %s", last.Actual, sum.FloatString(2), fmtE(maxAbs(r.Measures)))
		r.Pass = r.allWithin()
		if !r.Pass {
			t.Errorf("U04 %s: %s", profile, r.Actual)
		}
	})
}

func maxAbs(ms []Measure) float64 {
	m := 0.0
	for _, x := range ms {
		m = math.Max(m, x.EAbs)
	}
	return m
}

// U05: percentage discount, fixed reduction, both, and none; bills, balances
// and the full history through GetTransact.
func TestU05TransactionsAndHistory(t *testing.T) {
	data := functional(t)
	forProfiles(t, func(t *testing.T, profile string) {
		id := uid("u05", profile)
		var desc []string
		for _, x := range data.U05Transactions {
			desc = append(desc, fmt.Sprintf("%s x%d %s (%s)", x.ProductID, x.Quantity, x.DiscountCode, x.Kind))
		}
		r := newRecord("U05", "Transaksi dan riwayat", profile, "topup "+data.U05Initial+"; "+strings.Join(desc, "; "),
			"Tagihan t=(p*q*(1-d))-k dan saldo sesuai acuan di setiap langkah; GetTransact memuat semua transaksi dengan metadata (produk, qty, diskon) dan total yang benar")
		u := activeUser(t, id, profile)
		if err := u.LoadCatalog(ctx()); err != nil {
			t.Fatal(err)
		}
		bal := rat(data.U05Initial)
		if _, err := u.Topup(ctx(), amount(data.U05Initial)); err != nil {
			t.Fatal(err)
		}
		var totals []*big.Rat
		for i, x := range data.U05Transactions {
			p, d, k := catalogTerms(t, u.Catalog, x.ProductID, x.DiscountCode)
			want := refBill(p, x.Quantity, d, k)
			if i == 3 && profile == env.profiles[0] {
				samples.expect("04-transact")
			}
			res, err := u.Transact(ctx(), x.ProductID, x.Quantity, x.DiscountCode)
			if err != nil {
				t.Fatal(err)
			}
			bal.Sub(bal, want)
			totals = append(totals, want)
			r.measure(fmt.Sprintf("tagihan %s x%d %s", x.ProductID, x.Quantity, x.DiscountCode), want, res.Total)
			r.measure(fmt.Sprintf("saldo setelah transaksi #%d", i+1), new(big.Rat).Set(bal), res.Balance)
		}
		r.step("Topup lalu %d transaksi (POST /accounts/%s/transactions; product_id dan discount_code terbuka, kuantitas & metadata terenkripsi)", len(data.U05Transactions), id)
		if profile == env.profiles[0] {
			samples.expect("05-gettransact")
		}
		h, err := u.History(ctx())
		if err != nil {
			t.Fatal(err)
		}
		okMeta := len(h) == len(data.U05Transactions)
		for i := range h {
			if i >= len(data.U05Transactions) {
				break
			}
			x := data.U05Transactions[i]
			if h[i].ProductID != x.ProductID || h[i].Quantity != x.Quantity || h[i].DiscountCode != x.DiscountCode || h[i].Uncertain {
				okMeta = false
			}
			r.measure(fmt.Sprintf("riwayat #%d total (%s x%d %s, metadata mentah %.6f/%.6f/%.6f)", i+1, h[i].ProductID, h[i].Quantity, h[i].DiscountCode,
				h[i].MetadataRaw[0], h[i].MetadataRaw[1], h[i].MetadataRaw[2]), totals[i], h[i].Total)
		}
		r.step("GET /accounts/%s/transactions: %d entri, metadata dibulatkan ke bilangan bulat terdekat (toleransi %g)", id, len(h), ToleranceMetadata)
		r.Actual = fmt.Sprintf("%d transaksi tercatat, metadata cocok: %v, max e_abs %s", len(h), okMeta, fmtE(maxAbs(r.Measures)))
		r.Pass = r.allWithin() && okMeta
		if !r.Pass {
			t.Errorf("U05 %s: %s", profile, r.Actual)
		}
	})
}
