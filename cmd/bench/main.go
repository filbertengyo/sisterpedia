// Command bench measures the metrics of Table 2.7 of the assignment for each
// CKKS profile with the same inputs and operation sequence: key generation,
// client-side crypto, server-side evaluation (no network, no DB), end-to-end
// Topup/Transact over HTTP against the real server binary, and serialized
// artifact sizes (binary vs Base64). Each metric uses 5 warm-up runs and 30
// measured runs with a monotonic clock; raw data goes to bench/results/raw.
//
//	go run ./cmd/bench                 # both profiles, then the summary
//	go run ./cmd/bench -report         # only regenerate summary.md/ENVIRONMENT.md
//	node bench/wasm/bench.mjs          # WASM client metrics (uses fixtures written here)
package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/filbertengyo/sisterpedia/internal/apiclient"
	"github.com/filbertengyo/sisterpedia/internal/bench"
	"github.com/filbertengyo/sisterpedia/internal/server/catalog"
	"github.com/filbertengyo/sisterpedia/internal/server/transact"
	"github.com/filbertengyo/sisterpedia/internal/serverproc"
	"github.com/filbertengyo/sisterpedia/internal/shopclient"
	"github.com/filbertengyo/sisterpedia/pkg/ckks"
	"github.com/filbertengyo/sisterpedia/pkg/codec"
	"github.com/filbertengyo/sisterpedia/pkg/monoclock"
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
)

type options struct {
	profiles []string
	warmup   int
	runs     int
	out      string
	server   string
	skipE2E  bool
	fixtures string
}

// Same inputs for every profile.
const (
	topupAmount = 100000.0
	quantity    = 2.0
)

var (
	metadata = []float64{1, 2, 3} // [P001, qty 2, COMBO]
	product  = catalog.Product{ProductID: "P001", NumericID: 1, BasePrice: 12500}
	discount = catalog.Discount{Code: "COMBO", NumericID: 3, Percent: 0.1, Fixed: 500}
)

func main() {
	var o options
	profile := flag.String("profile", "all", "profile to measure: p1, p2 or all")
	flag.IntVar(&o.warmup, "warmup", 5, "warm-up runs (not recorded)")
	flag.IntVar(&o.runs, "runs", 30, "measured runs")
	flag.StringVar(&o.out, "out", "bench/results", "results directory (raw CSVs in <out>/raw)")
	flag.StringVar(&o.server, "server", "", "base URL of a running server for e2e (default: build and start cmd/server)")
	flag.BoolVar(&o.skipE2E, "skip-e2e", false, "skip the end-to-end metric")
	flag.StringVar(&o.fixtures, "fixtures", filepath.Join(os.TempDir(), "sisterpedia-bench"), "directory for WASM benchmark fixtures (TEST ONLY keys)")
	reportOnly := flag.Bool("report", false, "only regenerate the summary from raw data")
	flag.Parse()

	benchDir := filepath.Dir(filepath.Clean(o.out))
	if *reportOnly {
		if err := bench.WriteReport(o.out, benchDir); err != nil {
			log.Fatal(err)
		}
		log.Printf("wrote %s and %s", filepath.Join(o.out, "summary.md"), filepath.Join(benchDir, "ENVIRONMENT.md"))
		return
	}

	reg, err := ckks.DefaultRegistry()
	if err != nil {
		log.Fatal(err)
	}
	o.profiles = reg.Names()
	if *profile != "all" {
		o.profiles = strings.Split(*profile, ",")
	}
	raw := filepath.Join(o.out, "raw")
	if err := os.MkdirAll(raw, 0o750); err != nil {
		log.Fatal(err)
	}
	env := bench.CaptureEnvironment()
	if err := bench.WriteJSON(filepath.Join(raw, "environment_native.json"), env); err != nil {
		log.Fatal(err)
	}
	log.Printf("environment: %s, %d cores, %s, %s", env.CPU, env.Cores, env.OSVersion, env.GoVersion)

	var api *apiclient.Client
	if !o.skipE2E {
		url := o.server
		if url == "" {
			srv, stop, err := startServer()
			if err != nil {
				log.Fatal(err)
			}
			defer stop()
			url = srv.URL
		}
		api = apiclient.New(url)
	}

	for _, p := range o.profiles {
		if err := runProfile(reg, p, o, raw, api); err != nil {
			log.Fatalf("%s: %v", p, err)
		}
	}
	if err := wasmSizes(raw); err != nil {
		log.Printf("wasm size: %v (run `make wasm` first)", err)
	}
	if err := bench.WriteReport(o.out, benchDir); err != nil {
		log.Fatal(err)
	}
	log.Printf("done: %s", filepath.Join(o.out, "summary.md"))
}

func startServer() (*serverproc.Server, func(), error) {
	repo, err := serverproc.RepoRoot()
	if err != nil {
		return nil, nil, err
	}
	tmp, err := os.MkdirTemp("", "sisterpedia-bench-")
	if err != nil {
		return nil, nil, err
	}
	bin, err := serverproc.Build(repo, tmp)
	if err != nil {
		return nil, nil, err
	}
	srv, err := serverproc.New(repo, bin, tmp)
	if err != nil {
		return nil, nil, err
	}
	if err := srv.Start(); err != nil {
		return nil, nil, err
	}
	log.Printf("server process started at %s (pid %d)", srv.URL, srv.PID())
	return srv, func() { _ = srv.Stop(); _ = os.RemoveAll(tmp) }, nil
}

// repeat runs f warmup+runs times and records only the measured runs.
func repeat(o options, f func(run int) error) error {
	for i := 0; i < o.warmup; i++ {
		if err := f(0); err != nil {
			return err
		}
	}
	for i := 1; i <= o.runs; i++ {
		if err := f(i); err != nil {
			return err
		}
	}
	return nil
}

func runProfile(reg *ckks.Registry, profile string, o options, raw string, api *apiclient.Client) error {
	prof, err := reg.Profile(profile)
	if err != nil {
		return err
	}
	params, _ := reg.Params(profile)
	log.Printf("[%s] keygen", profile)

	// 1. key generation (client), separate from the services
	kg := &bench.Recorder{Profile: profile, Metric: "keygen"}
	if err := repeat(o, func(run int) error {
		_, t, err := ckks.GenKeysTimed(prof)
		if run > 0 {
			kg.Add("params", run, t.Params)
			kg.Add("sk", run, t.SecretKey)
			kg.Add("pk", run, t.PublicKey)
			kg.Add("rlk", run, t.RelinKey)
			kg.Add("total", run, t.Total())
		}
		return err
	}); err != nil {
		return err
	}
	if err := bench.WriteSamples(raw, kg.Samples); err != nil {
		return err
	}

	ks := ckks.GenKeys(params)
	cli := ckks.NewClient(params, ks.SK, ks.PK)
	ev := ckks.NewEvaluator(params, ks.PK, ks.RLK)
	terms := transact.Terms(product, discount)

	// realistic inputs: balance after one transaction (level L-2), its total
	bal0, _ := cli.EncryptScalar(0)
	amt, _ := cli.EncryptScalar(topupAmount)
	bal1, err := ev.Topup(bal0, amt, nil)
	if err != nil {
		return err
	}
	cq, _ := cli.EncryptScalar(quantity)
	res, err := transact.Transact(ev, bal1, cq, terms, nil)
	if err != nil {
		return err
	}
	cmeta, _ := cli.EncryptVector(metadata)

	// 2. client crypto, no network
	log.Printf("[%s] client crypto", profile)
	cr := &bench.Recorder{Profile: profile, Metric: "client"}
	enc := func(name string, values []float64, run int) error {
		t0 := monoclock.Now()
		pt, err := cli.Encode(values)
		if err != nil {
			return err
		}
		t1 := monoclock.Now()
		if _, err := cli.EncryptPlaintext(pt); err != nil {
			return err
		}
		t2 := monoclock.Now()
		if run > 0 {
			cr.Add("encode "+name, run, monoclock.Between(t0, t1))
			cr.Add("encrypt "+name, run, monoclock.Between(t1, t2))
			cr.Add("encode+encrypt "+name, run, monoclock.Between(t0, t2))
		}
		return nil
	}
	dec := func(name string, ct *rlwe.Ciphertext, n, run int) error {
		t0 := monoclock.Now()
		pt, err := cli.DecryptPlaintext(ct)
		if err != nil {
			return err
		}
		t1 := monoclock.Now()
		if _, err := cli.Decode(pt, n); err != nil {
			return err
		}
		t2 := monoclock.Now()
		if run > 0 {
			cr.Add("decrypt "+name, run, monoclock.Between(t0, t1))
			cr.Add("decode "+name, run, monoclock.Between(t1, t2))
			cr.Add("decrypt+decode "+name, run, monoclock.Between(t0, t2))
		}
		return nil
	}
	if err := repeat(o, func(run int) error {
		if err := enc("amount", []float64{topupAmount}, run); err != nil {
			return err
		}
		if err := enc("quantity", []float64{quantity}, run); err != nil {
			return err
		}
		if err := enc("metadata", metadata, run); err != nil {
			return err
		}
		if err := dec("balance", res.Balance, 1, run); err != nil {
			return err
		}
		if err := dec("total", res.Total, 1, run); err != nil {
			return err
		}
		return dec("metadata", cmeta, 3, run)
	}); err != nil {
		return err
	}
	if err := bench.WriteSamples(raw, cr.Samples); err != nil {
		return err
	}

	// 3. server evaluation, same code path as the server, no network/DB
	log.Printf("[%s] server evaluation", profile)
	sv := &bench.Recorder{Profile: profile, Metric: "server"}
	if err := repeat(o, func(run int) error {
		var trT ckks.Trace
		start := monoclock.Now()
		if _, err := transact.Topup(ev, res.Balance, amt, &trT); err != nil {
			return err
		}
		totalT := monoclock.Since(start)
		var trX ckks.Trace
		start = monoclock.Now()
		if _, err := transact.Transact(ev, res.Balance, cq, terms, &trX); err != nil {
			return err
		}
		totalX := monoclock.Since(start)
		if run > 0 {
			for _, s := range trT.Steps {
				sv.Add("topup."+s.Op, run, s.Duration)
			}
			sv.Add("topup.total", run, totalT)
			for _, s := range trX.Steps {
				sv.Add(s.Op, run, s.Duration)
			}
			sv.Add("transact.total", run, totalX)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := bench.WriteSamples(raw, sv.Samples); err != nil {
		return err
	}

	// 4. sizes of serialized artifacts
	if err := sizes(raw, profile, ks, cli, res, cmeta, amt, cq); err != nil {
		return err
	}

	// fixtures for the WASM benchmark (TEST ONLY keys, written to a temp dir)
	if err := writeFixtures(o.fixtures, profile, ks, res, cmeta); err != nil {
		log.Printf("fixtures: %v", err)
	}

	// 5. end-to-end over HTTP + SQLite
	if api != nil {
		log.Printf("[%s] end-to-end", profile)
		if err := e2e(reg, api, profile, o, raw); err != nil {
			return err
		}
	}
	return nil
}

func b64Size(b []byte) int { return codec.EncodedLen(len(b)) }

func sizes(raw, profile string, ks ckks.KeySet, cli *ckks.Client, res transact.Result, cmeta, amt, cq *rlwe.Ciphertext) error {
	ser, err := ks.Serialize()
	if err != nil {
		return err
	}
	rows := []bench.SizeRow{
		{Profile: profile, Artifact: "params", Binary: len(ser.Params), Base64: b64Size(ser.Params)},
		{Profile: profile, Artifact: "secret_key", Binary: len(ser.SK), Base64: b64Size(ser.SK)},
		{Profile: profile, Artifact: "public_key", Binary: len(ser.PK), Base64: b64Size(ser.PK)},
		{Profile: profile, Artifact: "relin_key", Binary: len(ser.RLK), Base64: b64Size(ser.RLK)},
	}
	add := func(name string, ct *rlwe.Ciphertext) error {
		b, err := ct.MarshalBinary()
		if err != nil {
			return err
		}
		rows = append(rows, bench.SizeRow{Profile: profile, Artifact: name, Binary: len(b), Base64: b64Size(b)})
		return nil
	}
	fresh, _ := cli.EncryptScalar(0)
	for _, x := range []struct {
		n  string
		ct *rlwe.Ciphertext
	}{
		{"ct_balance_fresh", fresh}, {"ct_topup_amount", amt}, {"ct_quantity", cq}, {"ct_metadata", cmeta},
		{"ct_total", res.Total}, {"ct_balance_after_transact", res.Balance},
	} {
		if err := add(x.n, x.ct); err != nil {
			return err
		}
	}
	// request payloads as sent (JSON)
	zero, _ := ckks.MarshalB64(fresh)
	act, _ := json.Marshal(apiclient.ActivateRequest{UserID: "bench-user", Params: codec.Encode(ser.Params), PublicKey: codec.Encode(ser.PK), RelinKey: codec.Encode(ser.RLK), EncBalance: zero})
	amtB64, _ := ckks.MarshalB64(amt)
	top, _ := json.Marshal(map[string]string{"enc_amount": amtB64})
	metaB64, _ := ckks.MarshalB64(cmeta)
	code := "COMBO"
	cqB64, _ := ckks.MarshalB64(cq)
	tx, _ := json.Marshal(apiclient.TransactRequest{ProductID: "P001", DiscountCode: &code, EncQuantity: cqB64, EncMetadata: metaB64})
	totB64, _ := ckks.MarshalB64(res.Total)
	balB64, _ := ckks.MarshalB64(res.Balance)
	txResp, _ := json.Marshal(map[string]string{"transaction_id": "01JXXXXXXXXXXXXXXXXXXXXXXX", "enc_total": totB64, "enc_balance": balB64})
	for _, x := range []struct {
		n string
		b []byte
	}{{"json_activate_request", act}, {"json_topup_request", top}, {"json_transact_request", tx}, {"json_transact_response", txResp}} {
		rows = append(rows, bench.SizeRow{Profile: profile, Artifact: x.n, Binary: len(x.b), Base64: len(x.b)})
	}
	return bench.WriteSizes(raw, profile, rows)
}

func wasmSizes(raw string) error {
	repo, err := serverproc.RepoRoot()
	if err != nil {
		return err
	}
	b, err := os.ReadFile(filepath.Join(repo, "web", "public", "crypto.wasm")) // #nosec G304 -- fixed path inside the repository
	if err != nil {
		return err
	}
	var gz bytes.Buffer
	w, _ := gzip.NewWriterLevel(&gz, gzip.BestCompression)
	_, _ = w.Write(b)
	_ = w.Close()
	return bench.WriteSizes(raw, "all", []bench.SizeRow{
		{Profile: "all", Artifact: "crypto.wasm (raw / gzip)", Binary: len(b), Base64: gz.Len()},
	})
}

type fixture struct {
	Profile    string `json:"profile"`
	Note       string `json:"note"`
	Params     string `json:"params"`
	SecretKey  string `json:"secret_key"`
	PublicKey  string `json:"public_key"`
	CtBalance  string `json:"ct_balance"`
	CtTotal    string `json:"ct_total"`
	CtMetadata string `json:"ct_metadata"`
}

func writeFixtures(dir, profile string, ks ckks.KeySet, res transact.Result, cmeta *rlwe.Ciphertext) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	ser, err := ks.Serialize()
	if err != nil {
		return err
	}
	f := fixture{Profile: profile, Note: "TEST ONLY benchmark keys, generated for this run", Params: codec.Encode(ser.Params), SecretKey: codec.Encode(ser.SK), PublicKey: codec.Encode(ser.PK)}
	if f.CtBalance, err = ckks.MarshalB64(res.Balance); err != nil {
		return err
	}
	if f.CtTotal, err = ckks.MarshalB64(res.Total); err != nil {
		return err
	}
	if f.CtMetadata, err = ckks.MarshalB64(cmeta); err != nil {
		return err
	}
	return bench.WriteJSON(filepath.Join(dir, profile+".json"), f)
}

func e2e(reg *ckks.Registry, api *apiclient.Client, profile string, o options, raw string) error {
	ctx := context.Background()
	id := fmt.Sprintf("bench-%s-%d", profile, time.Now().UnixNano()%1_000_000)
	u, err := shopclient.New(api, reg, id, profile)
	if err != nil {
		return err
	}
	if _, err := u.Activate(ctx); err != nil {
		return err
	}
	if err := u.LoadCatalog(ctx); err != nil {
		return err
	}
	rec := &bench.Recorder{Profile: profile, Metric: "e2e"}
	// Topup 1000 (35 runs -> balance 35000), then Transact P005 x1 (5000):
	// balances stay within the safe range on every profile.
	if err := repeat(o, func(run int) error {
		start := monoclock.Now()
		_, err := u.Topup(ctx, 1000)
		d := monoclock.Since(start)
		if run > 0 {
			rec.Add("topup", run, d)
			rec.Add("topup.client_encrypt", run, u.Last.Encrypt)
			rec.Add("topup.client_decrypt", run, u.Last.Decrypt)
		}
		return err
	}); err != nil {
		return err
	}
	if err := repeat(o, func(run int) error {
		start := monoclock.Now()
		_, err := u.Transact(ctx, "P005", 1, "NONE")
		d := monoclock.Since(start)
		if run > 0 {
			rec.Add("transact", run, d)
			rec.Add("transact.client_encrypt", run, u.Last.Encrypt)
			rec.Add("transact.client_decrypt", run, u.Last.Decrypt)
		}
		return err
	}); err != nil {
		return err
	}
	return bench.WriteSamples(raw, rec.Samples)
}
