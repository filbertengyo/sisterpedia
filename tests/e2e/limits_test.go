//go:build e2e

package e2e

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/filbertengyo/sisterpedia/internal/apiclient"
	"github.com/filbertengyo/sisterpedia/internal/shopclient"
	"github.com/filbertengyo/sisterpedia/pkg/ckks"
	"github.com/filbertengyo/sisterpedia/pkg/codec"
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
)

// U06: a bill above the balance is still subtracted; the client detects the
// negative balance after decryption.
func TestU06InsufficientBalance(t *testing.T) {
	data := functional(t)
	forProfiles(t, func(t *testing.T, profile string) {
		id := uid("u06", profile)
		x := data.U06Transaction
		r := newRecord("U06", "Saldo tidak mencukupi", profile,
			fmt.Sprintf("topup %s; beli %s x%d %s (tagihan di atas saldo)", data.U06Initial, x.ProductID, x.Quantity, x.DiscountCode),
			"Server tetap mengurangi saldo (tanpa pemeriksaan homomorfik); klien mendeteksi saldo negatif setelah dekripsi")
		u := activeUser(t, id, profile)
		_ = u.LoadCatalog(ctx())
		if _, err := u.Topup(ctx(), amount(data.U06Initial)); err != nil {
			t.Fatal(err)
		}
		p, d, k := catalogTerms(t, u.Catalog, x.ProductID, x.DiscountCode)
		bill := refBill(p, x.Quantity, d, k)
		res, err := u.Transact(ctx(), x.ProductID, x.Quantity, x.DiscountCode)
		if err != nil {
			t.Fatalf("transaction above balance was rejected: %v", err)
		}
		want := new(big.Rat).Sub(rat(data.U06Initial), bill)
		r.step("POST transaksi diterima (201) meskipun tagihan %s > saldo %s", bill.FloatString(0), data.U06Initial)
		r.measure("tagihan", bill, res.Total)
		r.measure("saldo akhir", want, res.Balance)
		negative := math.Round(res.Balance*100)/100 < 0
		r.step("Klien mendekripsi saldo: %.6f -> negatif terdeteksi: %v (GUI menampilkan 'Saldo tidak mencukupi')", res.Balance, negative)
		r.Actual = fmt.Sprintf("transaksi diterima, saldo %.6f, negatif terdeteksi: %v", res.Balance, negative)
		r.Pass = r.allWithin() && negative
		if !r.Pass {
			t.Errorf("U06 %s: %s", profile, r.Actual)
		}
	})
}

// U07: chained multiplications beyond the modulus chain without
// bootstrapping, isolated evaluation (library level).
func TestU07DepthLimit(t *testing.T) {
	forProfiles(t, func(t *testing.T, profile string) {
		params, _ := env.reg.Params(profile)
		L := params.MaxLevel()
		r := newRecord("U07", "Batas kedalaman perkalian", profile,
			fmt.Sprintf("x = 1.5 dikalikan berulang dengan Enc(1.5) (ciphertext x ciphertext + relin + rescale) sebanyak %d kali; level maks %d", L+1, L),
			fmt.Sprintf("Perkalian ke-1..%d berhasil dalam toleransi; perkalian ke-%d melewati kapasitas: pustaka menolak rescale atau hasil di luar toleransi", L, L+1))
		ks := ckks.GenKeys(params)
		cli := ckks.NewClient(params, ks.SK, ks.PK)
		ev := ckks.NewEvaluator(params, ks.PK, ks.RLK)
		ct, err := cli.EncryptScalar(1.5)
		if err != nil {
			t.Fatal(err)
		}
		want := rat("1.5")
		r.step("Enc(1.5): level %d, skala 2^%.4f", ct.Level(), ct.Scale.Log2())
		okSteps, rejected, outOfTol := 0, false, false
		for step := 1; step <= L+1; step++ {
			level := ct.Level()
			m, err := ev.EncryptConst([]float64{1.5}, L, ev.PrimeScale(level))
			if err != nil {
				t.Fatal(err)
			}
			if m, err = ev.DropTo(m, level); err != nil {
				t.Fatal(err)
			}
			want.Mul(want, rat("1.5"))
			var tr ckks.Trace
			next, err := ev.MulRelinRescale(ct, m, &tr)
			for _, s := range tr.Steps {
				r.step("perkalian #%d %s -> level %d, skala 2^%.4f", step, s.Op, s.Level, s.Log2Scale)
			}
			if err != nil {
				rejected = true
				r.step("perkalian #%d DITOLAK pustaka: %v", step, err)
				// what remains without rescaling: decrypt the relinearized product
				prod, perr := ev.Mul(ct, m)
				if perr == nil {
					lin, lerr := ev.Relinearize(prod)
					if lerr == nil {
						v, _ := cli.DecryptScalar(lin)
						mm := r.measure(fmt.Sprintf("perkalian #%d tanpa rescale (level %d, skala 2^%.1f)", step, lin.Level(), lin.Scale.Log2()), want, v)
						outOfTol = !mm.Within
						r.step("tanpa rescale, skala 2^%.1f melebihi q0 (%d bit): dekripsi %.6g, acuan %s", lin.Scale.Log2(), params.LogQLvl(0), v, want.FloatString(6))
					}
				}
				break
			}
			v, _ := cli.DecryptScalar(next)
			mm := r.measure(fmt.Sprintf("perkalian #%d (level %d)", step, next.Level()), new(big.Rat).Set(want), v)
			if mm.Within {
				okSteps++
			} else {
				outOfTol = true
			}
			ct = next
		}
		r.note("Tanpa bootstrapping, setiap perkalian+rescale menghabiskan satu level; profil %s mendukung tepat %d perkalian beruntun (tagihan butuh 2).", profile, L)
		r.Actual = fmt.Sprintf("%d perkalian berhasil dalam toleransi; perkalian ke-%d: ditolak pustaka=%v, hasil di luar toleransi=%v", okSteps, L+1, rejected, outOfTol)
		r.Pass = okSteps == L && (rejected || outOfTol)
		if !r.Pass {
			t.Errorf("U07 %s: %s", profile, r.Actual)
		}
	})
}

func rawPost(t *testing.T, path, body string) (int, string) {
	t.Helper()
	var e *apiclient.Error
	status, err := env.api.Do(ctx(), http.MethodPost, path, []byte(body), "application/json", nil)
	if errors.As(err, &e) {
		return e.Status, e.Code
	}
	if err != nil {
		t.Fatal(err)
	}
	return status, ""
}

// U08: malformed JSON/types, invalid Base64 and unloadable binaries are
// rejected with 400 without changing data; a byte-flipped ciphertext that
// still loads is accepted (CKKS has no integrity protection).
func TestU08InvalidPayloads(t *testing.T) {
	forProfiles(t, func(t *testing.T, profile string) {
		id := uid("u08", profile)
		r := newRecord("U08", "Muatan dan ciphertext tidak valid", profile,
			"JSON rusak, tipe salah, field tak dikenal, Base64 tidak valid, biner tak dapat dimuat, ciphertext profil lain, skala salah, secret key sebagai ciphertext; lalu ciphertext yang diubah 1 byte",
			"Kasus a-c: HTTP 400 dengan kode sesuai kontrak dan saldo/riwayat tidak berubah; kasus d: ciphertext termodifikasi tetap diterima (tidak terdeteksi) dan saldo menjadi salah")
		u := activeUser(t, id, profile)
		if _, err := u.Topup(ctx(), 1000); err != nil {
			t.Fatal(err)
		}
		if _, err := u.Transact(ctx(), "P005", 1, "NONE"); err != nil {
			t.Fatal(err)
		}
		before, _ := env.api.Balance(ctx(), id)
		histBefore, _ := env.api.History(ctx(), id)

		good, _ := u.EncryptB64(1)
		otherProfile := "p2"
		if profile == "p2" {
			otherProfile = "p1"
		}
		other := newUser(t, uid("u08o", profile), otherProfile)
		foreign, _ := other.EncryptB64(1)
		ev := ckks.NewEvaluator(u.Keys.Params, u.Keys.PK, u.Keys.RLK)
		odd, _ := ev.EncryptConst([]float64{1}, u.Keys.Params.MaxLevel(), rlwe.NewScale(math.Exp2(30)))
		oddB64, _ := ckks.MarshalB64(odd)
		skBin, _ := u.Keys.SK.MarshalBinary()
		goodBin, _ := codec.Decode(good, 0)
		truncated := codec.Encode(goodBin[:len(goodBin)/2])

		topup := "/accounts/" + id + "/topup"
		tx := "/accounts/" + id + "/transactions"
		cases := []struct {
			name, path, body string
			status           int
			code             string
		}{
			{"a1 JSON rusak", topup, `{"enc_amount":`, 400, "invalid_json"},
			{"a2 tipe salah (angka)", topup, `{"enc_amount": 1000}`, 400, "invalid_json"},
			{"a3 field tak dikenal (amount)", topup, `{"enc_amount":"` + good + `","amount":1000}`, 400, "invalid_json"},
			{"a4 field wajib hilang", tx, `{"product_id":"P001","enc_quantity":"` + good + `"}`, 400, "invalid_json"},
			{"b1 Base64 tidak valid", topup, `{"enc_amount":"@@@@####"}`, 400, "invalid_payload"},
			{"b2 Base64 kosong", topup, `{"enc_amount":""}`, 400, "invalid_payload"},
			{"c1 biner acak", topup, `{"enc_amount":"` + codec.Encode([]byte("bukan ciphertext CKKS sama sekali")) + `"}`, 400, "invalid_payload"},
			{"c2 ciphertext terpotong", topup, `{"enc_amount":"` + truncated + `"}`, 400, "invalid_payload"},
			{"c3 ciphertext profil " + otherProfile, topup, `{"enc_amount":"` + foreign + `"}`, 400, "invalid_payload"},
			{"c4 skala 2^30 (konteks salah)", topup, `{"enc_amount":"` + oddB64 + `"}`, 400, "context_mismatch"},
			{"c5 secret key sebagai ciphertext", topup, `{"enc_amount":"` + codec.Encode(skBin) + `"}`, 400, "secret_key_rejected"},
		}
		allOK := true
		for i, c := range cases {
			if i == 0 && profile == env.profiles[0] {
				samples.expect("06-error-invalid-json")
			}
			if c.code == "invalid_payload" && strings.HasPrefix(c.name, "c1") && profile == env.profiles[0] {
				samples.expect("07-error-invalid-payload")
			}
			status, code := rawPost(t, c.path, c.body)
			ok := status == c.status && code == c.code
			allOK = allOK && ok
			r.step("%s -> HTTP %d %s (diharapkan %d %s) %s", c.name, status, code, c.status, c.code, map[bool]string{true: "OK", false: "SALAH"}[ok])
		}
		after, _ := env.api.Balance(ctx(), id)
		histAfter, _ := env.api.History(ctx(), id)
		unchanged := after.EncBalance == before.EncBalance && len(histAfter.Transactions) == len(histBefore.Transactions)
		bal, _ := u.DecryptB64(after.EncBalance)
		r.measure("saldo setelah semua penolakan", rat("-4000"), bal)
		r.step("Ciphertext saldo byte-per-byte sama: %v; jumlah riwayat %d -> %d", after.EncBalance == before.EncBalance, len(histBefore.Transactions), len(histAfter.Transactions))

		// d: tampered but loadable ciphertext on a separate account
		id2 := uid("u08d", profile)
		v := activeUser(t, id2, profile)
		if _, err := v.Topup(ctx(), 1000); err != nil {
			t.Fatal(err)
		}
		enc, _ := v.EncryptB64(500)
		bin, _ := codec.Decode(enc, 0)
		bin[len(bin)-100] ^= 0x01
		resp, err := env.api.Topup(ctx(), id2, codec.Encode(bin))
		accepted := err == nil
		var tampered float64
		if accepted {
			tampered, _ = v.DecryptB64(resp.EncBalance)
		}
		e := errOf(rat("1500"), tampered)
		r.step("d: akun terpisah %s, Enc(500) diubah 1 byte (koefisien) -> diterima server: %v; saldo didekripsi %.6g (acuan 1500)", id2, accepted, tampered)
		r.note("Validasi format hanya memeriksa struktur, level, dan skala; perubahan isi ciphertext yang tetap valid tidak terdeteksi karena CKKS tidak menyediakan autentikasi/integritas (diperlukan MAC/tanda tangan di luar CKKS).")
		r.Actual = fmt.Sprintf("a-c: %d/%d kasus sesuai, data tidak berubah: %v; d: diterima=%v, saldo salah (e_abs %s)", countOK(r.Steps), len(cases), unchanged, accepted, fmtE(e.Abs))
		r.Pass = allOK && unchanged && r.allWithin() && accepted && e.Abs > 1
		if !r.Pass {
			t.Errorf("U08 %s: %s", profile, r.Actual)
		}
	})
}

func countOK(steps []string) int {
	n := 0
	for _, s := range steps {
		if strings.HasSuffix(s, " OK") {
			n++
		}
	}
	return n
}

// U09: relinearization key of another key pair in an isolated evaluation,
// plus the same mistake made through the server.
func TestU09ForeignRelinKey(t *testing.T) {
	forProfiles(t, func(t *testing.T, profile string) {
		r := newRecord("U09", "Kunci evaluasi berbeda", profile,
			"Evaluasi tagihan P001 x2 COMBO (acuan 22000) dengan pk akun A tetapi rlk dari pasangan kunci B",
			"Pustaka tidak dapat mendeteksi rlk yang salah; hasil dekripsi dengan sk A tidak sesuai acuan (dicatat nilainya)")
		params, _ := env.reg.Params(profile)
		a := ckks.GenKeys(params)
		b := ckks.GenKeys(params)
		cli := ckks.NewClient(params, a.SK, a.PK)
		evWrong := ckks.NewEvaluator(params, a.PK, b.RLK)
		evRight := ckks.NewEvaluator(params, a.PK, a.RLK)
		terms := ckks.BillTerms{Price: []float64{12500}, Factor: []float64{0.9}, Fixed: []float64{500}}
		cq, _ := cli.EncryptScalar(2)
		right, err := evRight.Bill(cq, terms, nil)
		if err != nil {
			t.Fatal(err)
		}
		vRight, _ := cli.DecryptScalar(right)
		r.measure("kontrol: rlk A", rat("22000"), vRight)
		wrong, err := evWrong.Bill(cq, terms, nil)
		libErr := err != nil
		var vWrong float64
		if err == nil {
			vWrong, _ = cli.DecryptScalar(wrong)
		}
		e := errOf(rat("22000"), vWrong)
		r.step("Evaluasi terisolasi dengan rlk B: galat pustaka=%v; hasil dekripsi sk A = %.6g (e_abs %s)", libErr, vWrong, fmtE(e.Abs))

		// through the server: activation with pk A and rlk B is accepted (format valid)
		id := uid("u09", profile)
		u := newUser(t, id, profile)
		other := newUser(t, uid("u09b", profile), profile)
		req, _ := u.ActivateRequest()
		otherReq, _ := other.ActivateRequest()
		req.RelinKey = otherReq.RelinKey
		_, actErr := env.api.Activate(ctx(), req)
		r.step("Aktivasi server dengan pk A + rlk B: diterima=%v (format valid tidak membuktikan pasangan kunci)", actErr == nil)
		var serverTotal float64
		if actErr == nil {
			_ = u.LoadCatalog(ctx())
			if _, err := u.Topup(ctx(), 50000); err != nil {
				t.Fatal(err)
			}
			res, err := u.Transact(ctx(), "P001", 2, "COMBO")
			if err == nil {
				serverTotal = res.Total
			}
			r.step("Topup tetap benar (tanpa relinearization); transaksi P001 x2 COMBO -> total didekripsi %.6g", serverTotal)
		}
		es := errOf(rat("22000"), serverTotal)
		r.note("Lattigo tidak memverifikasi bahwa rlk berasal dari sk yang sama; hanya pemeriksaan di klien (ValidateKeys sebelum aktivasi/impor) yang dapat mencegahnya.")
		r.Actual = fmt.Sprintf("terisolasi: %.6g (e_abs %s); via server: aktivasi diterima=%v, total %.6g (e_abs %s)", vWrong, fmtE(e.Abs), actErr == nil, serverTotal, fmtE(es.Abs))
		r.Pass = r.allWithin() && (libErr || e.Abs > 1) && (actErr != nil || es.Abs > 1)
		if !r.Pass {
			t.Errorf("U09 %s: %s", profile, r.Actual)
		}
	})
}

// U10: services for an id that was never activated return 404 account_not_found.
func TestU10UnknownAccount(t *testing.T) {
	forProfiles(t, func(t *testing.T, profile string) {
		id := uid("u10", profile)
		r := newRecord("U10", "Akun belum aktif", profile, "user_id="+id+" (tidak pernah diaktifkan)",
			"CheckBalance, Topup, Transact, GetTransact -> HTTP 404 account_not_found (pilihan kontrak: 404)")
		u := newUser(t, id, profile)
		_ = u.LoadCatalog(ctx())
		enc, _ := u.EncryptB64(1000)
		meta, _ := u.EncryptB64(1, 1, 0)
		calls := []struct {
			name string
			f    func() error
		}{
			{"CheckBalance", func() error { _, err := env.api.Balance(ctx(), id); return err }},
			{"Topup", func() error { _, err := env.api.Topup(ctx(), id, enc); return err }},
			{"Transact", func() error {
				_, err := env.api.Transact(ctx(), id, apiclient.TransactRequest{ProductID: "P001", EncQuantity: enc, EncMetadata: meta})
				return err
			}},
			{"GetTransact", func() error { _, err := env.api.History(ctx(), id); return err }},
		}
		ok := 0
		for i, c := range calls {
			if i == 0 && profile == env.profiles[0] {
				samples.expect("08-error-account-not-found")
			}
			err := c.f()
			match := shopclient.IsCode(err, 404, "account_not_found")
			if match {
				ok++
			}
			r.step("%s -> %v", c.name, err)
		}
		r.Actual = fmt.Sprintf("%d/%d panggilan mengembalikan 404 account_not_found", ok, len(calls))
		r.Pass = ok == len(calls)
		if !r.Pass {
			t.Errorf("U10 %s: %s", profile, r.Actual)
		}
	})
}

// U11: value bound. A coefficient of the plaintext polynomial must satisfy
// |c| < q0/2 at level 0. When every slot holds x, coefficient 0 equals
// x * scale, so |x| < q0/(2*scale) (about 524287) is the bound that holds for
// any slot pattern; beyond it CKKS wraps around silently. A value in a single
// slot spreads over the coefficients (about x*scale*2/N each) and only
// overflows much later. The client enforces the conservative bound
// (max_value = 500000) for every value.
func TestU11ValueBound(t *testing.T) {
	forProfiles(t, func(t *testing.T, profile string) {
		params, _ := env.reg.Params(profile)
		limit := ckks.MaxValue(params)
		slots := params.MaxSlots()
		r := newRecord("U11", "Batas nilai (overflow)", profile,
			fmt.Sprintf("batas aman |x| <= q0/(2*skala) = %.0f (semua pola slot); penjaga klien 500000; nilai diturunkan ke level 0 lalu didekripsi", limit),
			"Penjaga klien menolak > batas; di bawah batas hasil benar; di atas batas (vektor penuh 600000, skalar 1e10) hasil salah TANPA pesan galat")
		ks := ckks.GenKeys(params)
		cli := ckks.NewClient(params, ks.SK, ks.PK)
		ev := ckks.NewEvaluator(params, ks.PK, ks.RLK)
		_, guardErr := cli.EncryptScalar(600000)
		r.step("Penjaga klien EncryptScalar(600000): %v", guardErr)

		fill := func(x float64, n int) []float64 {
			v := make([]float64, n)
			for i := range v {
				v[i] = x
			}
			return v
		}
		cases := []struct {
			label     string
			values    []float64
			wantRight bool
		}{
			{"skalar 400000 (slot 0)", []float64{400000}, true},
			{fmt.Sprintf("vektor penuh 400000 (%d slot)", slots), fill(400000, slots), true},
			{"skalar 600000 (slot 0)", []float64{600000}, true},
			{fmt.Sprintf("vektor penuh 600000 (%d slot)", slots), fill(600000, slots), false},
			{"skalar 1e10 (slot 0)", []float64{1e10}, false},
		}
		pass := errors.Is(guardErr, ckks.ErrValueOutOfRange)
		for _, c := range cases {
			ct, err := cli.EncryptUnchecked(c.values)
			if err != nil {
				t.Fatal(err)
			}
			low, _ := ev.DropTo(ct, 0)
			v, _ := cli.DecryptScalar(low)
			e := errOf(ratF(c.values[0]), v)
			within := e.Abs <= ToleranceMoney || (c.values[0] >= 1e9 && e.Rel < 1e-9)
			r.Measures = append(r.Measures, Measure{Label: c.label + " pada level 0", Expected: fmt.Sprintf("%.2f", c.values[0]), Actual: v, EAbs: e.Abs, ERel: e.RelString(), Within: within})
			r.step("%s, level 0 -> slot 0 didekripsi %.6g (%s)", c.label, v, map[bool]string{true: "benar", false: "SALAH, tanpa galat"}[within])
			pass = pass && within == c.wantRight
		}
		r.note("Overflow terjadi pada koefisien polinomial: |koefisien| >= q0/2 membungkus modulo q0 tanpa pesan galat. Untuk vektor penuh koefisien ke-0 = x*skala, sehingga batas %.0f berlaku umum; nilai di satu slot tersebar ke banyak koefisien (~x*skala*2/N) sehingga baru overflow jauh di atas batas. Klien memakai batas konservatif 500000 untuk saldo, nominal, dan subtotal.", limit)
		r.Actual = fmt.Sprintf("penjaga menolak: %v; vektor penuh 600000 -> %.6g; skalar 1e10 -> %.6g", errors.Is(guardErr, ckks.ErrValueOutOfRange), r.Measures[3].Actual, r.Measures[4].Actual)
		r.Pass = pass
		if !r.Pass {
			t.Errorf("U11 %s: %s", profile, r.Actual)
		}
	})
}

// U12: concurrent requests on one account: no lost update, consistent history.
func TestU12Concurrency(t *testing.T) {
	forProfiles(t, func(t *testing.T, profile string) {
		id := uid("u12", profile)
		const topups, txs = 10, 3
		r := newRecord("U12", "Permintaan serentak", profile,
			fmt.Sprintf("saldo awal 20000; %d topup 1000 dan %d transaksi P005 x1 NONE dikirim bersamaan", topups, txs),
			"Semua permintaan berhasil; saldo akhir = 20000 + 10000 - 15000 = 15000; riwayat berisi 3 transaksi (tidak ada update yang hilang)")
		u := activeUser(t, id, profile)
		_ = u.LoadCatalog(ctx())
		if _, err := u.Topup(ctx(), 20000); err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		errs := make(chan error, topups+txs)
		for i := 0; i < topups; i++ {
			enc, _ := u.EncryptB64(1000)
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := env.api.Topup(ctx(), id, enc)
				errs <- err
			}()
		}
		for i := 0; i < txs; i++ {
			q, _ := u.EncryptB64(1)
			m, _ := u.EncryptB64(5, 1, 0)
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := env.api.Transact(ctx(), id, apiclient.TransactRequest{ProductID: "P005", EncQuantity: q, EncMetadata: m})
				errs <- err
			}()
		}
		wg.Wait()
		close(errs)
		failed := 0
		for err := range errs {
			if err != nil {
				failed++
			}
		}
		b, _ := u.Balance(ctx())
		h, _ := u.History(ctx())
		r.step("%d permintaan paralel selesai, %d gagal; server menyerialkan operasi per akun (mutex per user_id) dan menyimpan saldo+riwayat dalam satu transaksi SQL", topups+txs, failed)
		r.measure("saldo akhir", rat("15000"), b)
		r.Actual = fmt.Sprintf("gagal %d, saldo %.6f, riwayat %d entri", failed, b, len(h))
		r.Pass = failed == 0 && r.allWithin() && len(h) == txs
		if !r.Pass {
			t.Errorf("U12 %s: %s", profile, r.Actual)
		}
	})
}
