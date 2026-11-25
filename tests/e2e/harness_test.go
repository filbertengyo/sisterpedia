//go:build e2e

// Package e2e runs the assignment's test scenarios (U01-U12 and the
// transaction chains) against the real server binary started as a separate
// process with a temporary SQLite database. The client side (key generation,
// encryption, decryption) runs in this test process through the same
// pkg/ckks code that the browser runs as WebAssembly: a trusted environment
// separate from the server. Reference values use exact rational arithmetic.
//
//	go test -tags e2e -count=1 -v ./tests/e2e/
//
// Env: E2E_PROFILES (default "p1,p2"), E2E_RESULTS_DIR (default tests/results).
package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/filbertengyo/sisterpedia/internal/apiclient"
	"github.com/filbertengyo/sisterpedia/internal/serverproc"
	"github.com/filbertengyo/sisterpedia/internal/shopclient"
	"github.com/filbertengyo/sisterpedia/pkg/ckks"
)

// ToleranceMoney is the accepted absolute error |y - y_hat| for money values
// (balances and bills), fixed before running the tests. docs/API_CONTRACT.md
// starts at 1e-2 and allows tightening to 1e-3 only once the measured error is
// well below 1e-3. It is not: CKKS error grows with the magnitude of the
// multiplied values and with N, and repeated runs of the large-value chain on
// p2 (C3-nilai-besar, subtotals up to ~1e5) reached ~1e-3 (relative ~1e-8),
// while a single earlier run happened to stay at 1.6e-4. 1e-2 (one cent) is
// therefore kept.
const ToleranceMoney = 1e-2

// ToleranceMetadata is the maximum distance from an integer for decrypted
// metadata slots (product id, quantity, discount id) before rounding.
const ToleranceMetadata = shopclient.MetadataTolerance

var env struct {
	repo     string
	server   *serverproc.Server
	api      *apiclient.Client
	reg      *ckks.Registry
	profiles []string
	results  string
	started  time.Time
}

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	var err error
	env.started = time.Now()
	if env.repo, err = serverproc.RepoRoot(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	env.results = os.Getenv("E2E_RESULTS_DIR")
	if env.results == "" {
		env.results = filepath.Join(env.repo, "tests", "results")
	}
	env.profiles = []string{"p1", "p2"}
	if v := os.Getenv("E2E_PROFILES"); v != "" {
		env.profiles = strings.Split(v, ",")
	}
	if env.reg, err = ckks.DefaultRegistry(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	tmp, err := os.MkdirTemp("", "sisterpedia-e2e-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	bin, err := serverproc.Build(env.repo, tmp)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if env.server, err = serverproc.New(env.repo, bin, tmp); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := env.server.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = env.server.Stop() }()
	env.api = apiclient.New(env.server.URL)
	env.api.OnExchange = samples.capture

	code := m.Run()

	if err := writeResults(); err != nil {
		fmt.Fprintln(os.Stderr, "write results:", err)
		if code == 0 {
			code = 1
		}
	}
	if err := checkServerLogs(env.server.Logs()); err != nil {
		fmt.Fprintln(os.Stderr, "server log check:", err)
		if code == 0 {
			code = 1
		}
	}
	return code
}

// checkServerLogs verifies that the server log never contains bodies: no
// Base64 blobs and no JSON.
func checkServerLogs(logs string) error {
	for _, line := range strings.Split(logs, "\n") {
		if len(line) > 400 || strings.Contains(line, "{\"") || strings.Contains(line, "enc_") {
			return fmt.Errorf("suspicious log line: %.120s", line)
		}
	}
	return os.WriteFile(filepath.Join(env.results, "server_access.log"), []byte(logs), 0o600)
}

func ctx() context.Context { return context.Background() }

// newUser creates a fresh client-side user with its own key pair.
func newUser(t *testing.T, id, profile string) *shopclient.User {
	t.Helper()
	u, err := shopclient.New(env.api, env.reg, id, profile)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// activeUser creates and activates a user.
func activeUser(t *testing.T, id, profile string) *shopclient.User {
	t.Helper()
	u := newUser(t, id, profile)
	if _, err := u.Activate(ctx()); err != nil {
		t.Fatalf("activate %s: %v", id, err)
	}
	return u
}

// uid builds a per-profile, per-run unique user id.
func uid(prefix, profile string) string {
	return fmt.Sprintf("%s-%s-%d", prefix, profile, env.started.UnixNano()%1_000_000)
}

func loadJSON(t *testing.T, name string, v any) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(env.repo, "tests", "data", name)) // #nosec G304 -- test data
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatal(err)
	}
}

func forProfiles(t *testing.T, f func(t *testing.T, profile string)) {
	for _, p := range env.profiles {
		t.Run(p, func(t *testing.T) { f(t, p) })
	}
}
