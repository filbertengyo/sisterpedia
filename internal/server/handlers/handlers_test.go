package handlers_test

import (
	"context"
	"io"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/filbertengyo/sisterpedia/internal/apiclient"
	"github.com/filbertengyo/sisterpedia/internal/server/catalog"
	"github.com/filbertengyo/sisterpedia/internal/server/handlers"
	"github.com/filbertengyo/sisterpedia/internal/server/service"
	"github.com/filbertengyo/sisterpedia/internal/server/store"
	"github.com/filbertengyo/sisterpedia/internal/shopclient"
	"github.com/filbertengyo/sisterpedia/pkg/ckks"
	"github.com/filbertengyo/sisterpedia/pkg/codec"
)

const origin = "http://localhost:3000"

func newServer(t *testing.T) (*httptest.Server, *ckks.Registry) {
	t.Helper()
	reg, err := ckks.DefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(t.TempDir(), "shop.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	cat, err := catalog.Load("../../../data/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	svc := service.New(reg, st, cat, 16<<20)
	h := handlers.New(svc, handlers.Options{AllowedOrigins: []string{origin}, MaxBodyBytes: 16 << 20, Logger: log.New(io.Discard, "", 0)})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv, reg
}

func near(a, b float64) bool { return math.Abs(a-b) <= 1e-2 }

func TestFullFlow(t *testing.T) {
	srv, reg := newServer(t)
	ctx := context.Background()
	api := apiclient.New(srv.URL)
	u, err := shopclient.New(api, reg, "alice", "p1")
	if err != nil {
		t.Fatal(err)
	}
	b0, err := u.Activate(ctx)
	if err != nil || !near(b0, 0) {
		t.Fatalf("activate: %v %v", b0, err)
	}
	if _, err := u.Activate(ctx); !shopclient.IsCode(err, 409, "account_exists") {
		t.Fatalf("double activation: %v", err)
	}
	if b, err := u.Topup(ctx, 100000); err != nil || !near(b, 100000) {
		t.Fatalf("topup: %v %v", b, err)
	}
	r, err := u.Transact(ctx, "P001", 2, "COMBO")
	if err != nil || !near(r.Total, 22000) || !near(r.Balance, 78000) {
		t.Fatalf("transact: %+v %v", r, err)
	}
	h, err := u.History(ctx)
	if err != nil || len(h) != 1 || h[0].ProductID != "P001" || h[0].Quantity != 2 || h[0].DiscountCode != "COMBO" || !near(h[0].Total, 22000) {
		t.Fatalf("history: %+v %v", h, err)
	}
	if b, err := u.Balance(ctx); err != nil || !near(b, 78000) {
		t.Fatalf("balance: %v %v", b, err)
	}
}

func post(t *testing.T, url, ctype, body string) (int, string) {
	t.Helper()
	resp, err := http.Post(url, ctype, strings.NewReader(body)) // #nosec G107 -- test server
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestRejectionsLeaveDataUnchanged(t *testing.T) {
	srv, reg := newServer(t)
	ctx := context.Background()
	api := apiclient.New(srv.URL)
	u, _ := shopclient.New(api, reg, "bob", "p1")
	if _, err := u.Activate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := u.Topup(ctx, 5000); err != nil {
		t.Fatal(err)
	}
	before, _ := api.Balance(ctx, "bob")

	enc, _ := u.EncryptB64(1)
	cases := []struct {
		name, path, ctype, body string
		status                  int
		code                    string
	}{
		{"malformed json", "/accounts/bob/topup", "application/json", `{"enc_amount":`, 400, "invalid_json"},
		{"wrong type", "/accounts/bob/topup", "application/json", `{"enc_amount": 123}`, 400, "invalid_json"},
		{"unknown field", "/accounts/bob/topup", "application/json", `{"enc_amount":"` + enc + `","amount":1}`, 400, "invalid_json"},
		{"missing field", "/accounts/bob/topup", "application/json", `{}`, 400, "invalid_json"},
		{"trailing data", "/accounts/bob/topup", "application/json", `{"enc_amount":"` + enc + `"}{}`, 400, "invalid_json"},
		{"wrong content type", "/accounts/bob/topup", "text/plain", `{"enc_amount":"` + enc + `"}`, 400, "invalid_json"},
		{"empty base64", "/accounts/bob/topup", "application/json", `{"enc_amount":""}`, 400, "invalid_payload"},
		{"invalid base64", "/accounts/bob/topup", "application/json", `{"enc_amount":"@@@"}`, 400, "invalid_payload"},
		{"unloadable binary", "/accounts/bob/topup", "application/json", `{"enc_amount":"` + codec.Encode([]byte("not a ciphertext")) + `"}`, 400, "invalid_payload"},
		{"unknown product", "/accounts/bob/transactions", "application/json", `{"product_id":"NOPE","enc_quantity":"` + enc + `","enc_metadata":"` + enc + `"}`, 404, "product_not_found"},
		{"unknown discount", "/accounts/bob/transactions", "application/json", `{"product_id":"P001","discount_code":"NOPE","enc_quantity":"` + enc + `","enc_metadata":"` + enc + `"}`, 404, "discount_not_found"},
		{"unknown account", "/accounts/nobody/topup", "application/json", `{"enc_amount":"` + enc + `"}`, 404, "account_not_found"},
		{"bad user id", "/accounts/UPPER/topup", "application/json", `{"enc_amount":"` + enc + `"}`, 400, "invalid_json"},
	}
	for _, c := range cases {
		status, body := post(t, srv.URL+c.path, c.ctype, c.body)
		if status != c.status || !strings.Contains(body, `"error":"`+c.code+`"`) {
			t.Errorf("%s: got %d %s, want %d %s", c.name, status, body, c.status, c.code)
		}
	}
	after, _ := api.Balance(ctx, "bob")
	if after.EncBalance != before.EncBalance {
		t.Fatal("balance changed after rejected requests")
	}
	h, _ := api.History(ctx, "bob")
	if len(h.Transactions) != 0 {
		t.Fatal("history changed after rejected requests")
	}
}

func TestActivateRejectsPrivateKeyAndForeignParams(t *testing.T) {
	srv, reg := newServer(t)
	ctx := context.Background()
	api := apiclient.New(srv.URL)
	u, _ := shopclient.New(api, reg, "carol", "p1")
	req, _ := u.ActivateRequest()
	ser, _ := u.Keys.Serialize()

	withSK := req
	withSK.PublicKey = codec.Encode(ser.SK)
	if _, err := api.Activate(ctx, withSK); !shopclient.IsCode(err, 400, "secret_key_rejected") {
		t.Errorf("sk as public_key: %v", err)
	}
	withSK = req
	withSK.RelinKey = codec.Encode(ser.SK)
	if _, err := api.Activate(ctx, withSK); !shopclient.IsCode(err, 400, "secret_key_rejected") {
		t.Errorf("sk as relin_key: %v", err)
	}
	bad := req
	bad.Params = codec.Encode([]byte(`{"LogN":12}`))
	if _, err := api.Activate(ctx, bad); !shopclient.IsCode(err, 400, "context_mismatch") {
		t.Errorf("foreign params: %v", err)
	}
	// p2 keys with p1 params
	u2, _ := shopclient.New(api, reg, "carol", "p2")
	req2, _ := u2.ActivateRequest()
	mixed := req
	mixed.PublicKey = req2.PublicKey
	if _, err := api.Activate(ctx, mixed); !shopclient.IsCode(err, 400, "invalid_payload") {
		t.Errorf("p2 public key with p1 params: %v", err)
	}
	if _, err := api.Balance(ctx, "carol"); !shopclient.IsCode(err, 404, "account_not_found") {
		t.Errorf("rejected activation must not create the account: %v", err)
	}
	// unknown JSON field in Activate (e.g. a secret_key field) is rejected
	status, body := post(t, srv.URL+"/accounts", "application/json", `{"user_id":"carol","secret_key":"AAAA"}`)
	if status != 400 || !strings.Contains(body, "invalid_json") {
		t.Errorf("secret_key field: %d %s", status, body)
	}
}

func TestConcurrentTopups(t *testing.T) {
	srv, reg := newServer(t)
	ctx := context.Background()
	api := apiclient.New(srv.URL)
	u, _ := shopclient.New(api, reg, "dave", "p1")
	if _, err := u.Activate(ctx); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	const n = 8
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			enc, _ := u.EncryptB64(100)
			if _, err := api.Topup(ctx, "dave", enc); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if b, err := u.Balance(ctx); err != nil || !near(b, n*100) {
		t.Fatalf("balance after %d concurrent topups = %v (%v)", n, b, err)
	}
}

func TestCORS(t *testing.T) {
	srv, _ := newServer(t)
	for _, tc := range []struct {
		origin string
		allow  bool
	}{{origin, true}, {"http://evil.example", false}} {
		req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/accounts", nil)
		req.Header.Set("Origin", tc.origin)
		req.Header.Set("Access-Control-Request-Method", "POST")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		got := resp.Header.Get("Access-Control-Allow-Origin")
		if tc.allow && (got != origin || resp.StatusCode != http.StatusNoContent) {
			t.Errorf("allowed origin: status %d header %q", resp.StatusCode, got)
		}
		if !tc.allow && got != "" {
			t.Errorf("foreign origin got CORS header %q", got)
		}
	}
}
