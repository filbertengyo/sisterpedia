//go:build e2e

package e2e

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/filbertengyo/sisterpedia/pkg/ckks"
)

// Measure is one compared value: raw decrypted CKKS value (before any
// rounding) against the exact reference.
type Measure struct {
	Label    string  `json:"label"`
	Expected string  `json:"expected"`
	Actual   float64 `json:"actual_raw"`
	EAbs     float64 `json:"e_abs"`
	ERel     string  `json:"e_rel"`
	Within   bool    `json:"within_tolerance"`
}

// Record documents one scenario run: input, steps, expected and actual result.
type Record struct {
	ID       string    `json:"id"`
	Category string    `json:"category"`
	Profile  string    `json:"profile"`
	Input    string    `json:"input"`
	Steps    []string  `json:"steps"`
	Expected string    `json:"expected"`
	Actual   string    `json:"actual"`
	Pass     bool      `json:"pass"`
	Measures []Measure `json:"measures,omitempty"`
	Notes    []string  `json:"notes,omitempty"`
}

// ChainRow is one CSV row: a balance change in a transaction chain.
type ChainRow struct {
	Profile  string
	Chain    string
	Step     int
	Op       string
	Expected *big.Rat
	Actual   float64
	Err      ErrResult
}

// ChainSummary describes one chain run.
type ChainSummary struct {
	Profile, Chain, Description         string
	Ops, Topups, NoDiscount, WithDiscnt int
	MaxAbs                              float64
	MaxRel                              float64
	Pass                                bool
}

var rec = struct {
	sync.Mutex
	records   []*Record
	chainRows []ChainRow
	totalRows []ChainRow
	summaries []ChainSummary
}{}

func fmtE(f float64) string { return strconv.FormatFloat(f, 'e', 3, 64) }

// newRecord starts a record that is saved when the test finishes.
func newRecord(id, category, profile, input, expected string) *Record {
	r := &Record{ID: id, Category: category, Profile: profile, Input: input, Expected: expected}
	rec.Lock()
	rec.records = append(rec.records, r)
	rec.Unlock()
	return r
}

func (r *Record) step(format string, args ...any) {
	r.Steps = append(r.Steps, fmt.Sprintf(format, args...))
}
func (r *Record) note(format string, args ...any) {
	r.Notes = append(r.Notes, fmt.Sprintf(format, args...))
}

// measure compares a raw value with the reference and appends it.
func (r *Record) measure(label string, y *big.Rat, yhat float64) Measure {
	e := errOf(y, yhat)
	m := Measure{Label: label, Expected: y.FloatString(2), Actual: yhat, EAbs: e.Abs, ERel: e.RelString(), Within: e.Abs <= ToleranceMoney}
	r.Measures = append(r.Measures, m)
	return m
}

func (r *Record) allWithin() bool {
	for _, m := range r.Measures {
		if !m.Within {
			return false
		}
	}
	return true
}

func writeResults() error {
	if err := os.MkdirAll(env.results, 0o750); err != nil {
		return err
	}
	rec.Lock()
	defer rec.Unlock()
	sort.SliceStable(rec.records, func(i, j int) bool {
		if rec.records[i].ID != rec.records[j].ID {
			return rec.records[i].ID < rec.records[j].ID
		}
		return rec.records[i].Profile < rec.records[j].Profile
	})
	js, err := json.MarshalIndent(map[string]any{
		"generated_at": time.Now().UTC().Format(time.RFC3339), "tolerance_money": ToleranceMoney,
		"tolerance_metadata": ToleranceMetadata, "profiles": env.profiles, "library": ckks.LibraryVersion,
		"records": rec.records, "chains": rec.summaries,
	}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(env.results, "results.json"), js, 0o600); err != nil {
		return err
	}
	if err := writeChainCSV("chains.csv", rec.chainRows); err != nil {
		return err
	}
	if err := writeChainCSV("chains_totals.csv", rec.totalRows); err != nil {
		return err
	}
	if err := samples.write(); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(env.results, "RESULTS.md"), []byte(markdown()), 0o600)
}

func writeChainCSV(name string, rows []ChainRow) error {
	f, err := os.Create(filepath.Join(env.results, name)) // #nosec G304 -- results dir
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"profile", "chain", "step", "op", "expected", "actual", "e_abs", "e_rel"})
	for _, r := range rows {
		_ = w.Write([]string{r.Profile, r.Chain, strconv.Itoa(r.Step), r.Op, r.Expected.FloatString(2),
			strconv.FormatFloat(r.Actual, 'f', 10, 64), fmtE(r.Err.Abs), r.Err.RelString()})
	}
	w.Flush()
	return w.Error()
}

func esc(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, "|", "\\|"), "\n", " ") }

func markdown() string {
	var b strings.Builder
	pass, total := 0, 0
	for _, r := range rec.records {
		total++
		if r.Pass {
			pass++
		}
	}
	fmt.Fprintf(&b, "# Hasil pengujian U01-U12 dan rangkaian transaksi\n\n")
	fmt.Fprintf(&b, "Dibuat otomatis oleh `go test -tags e2e ./tests/e2e/` pada %s.\n\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "- Profil: %s; pustaka %s; Go %s, %s/%s\n", strings.Join(env.profiles, ", "), ckks.LibraryVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&b, "- Server: biner `cmd/server` sebagai proses terpisah, SQLite sementara; klien: `pkg/ckks` (kode yang sama dengan WASM di browser) di proses uji.\n")
	fmt.Fprintf(&b, "- **Toleransi galat uang (ditetapkan sebelum uji): e_abs <= %g**; metadata: |nilai - bulat| <= %g. Galat relatif e_rel = e_abs/|y| dan ditulis `n/a` bila y = 0.\n", ToleranceMoney, ToleranceMetadata)
	fmt.Fprintf(&b, "- Acuan dihitung dengan `math/big` (rasional eksak); nilai aktual adalah hasil dekripsi **sebelum** pembulatan.\n")
	fmt.Fprintf(&b, "- Ringkasan: **%d/%d skenario lulus**.\n\n", pass, total)

	b.WriteString("## Ringkasan\n\n| ID | Profil | Kategori | Diharapkan | Aktual | Lulus |\n|---|---|---|---|---|---|\n")
	for _, r := range rec.records {
		mark := "ya"
		if !r.Pass {
			mark = "**TIDAK**"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n", r.ID, r.Profile, esc(r.Category), esc(r.Expected), esc(r.Actual), mark)
	}

	b.WriteString("\n## Rangkaian transaksi (>= 5 per profil)\n\nSetiap perubahan saldo dibandingkan dengan acuan (`chains.csv`); total tagihan di `chains_totals.csv`.\n\n")
	b.WriteString("| Profil | Rangkaian | Operasi | Topup | Tanpa diskon | Dengan diskon | max e_abs | max e_rel | Lulus |\n|---|---|---|---|---|---|---|---|---|\n")
	for _, s := range rec.summaries {
		mark := "ya"
		if !s.Pass {
			mark = "**TIDAK**"
		}
		fmt.Fprintf(&b, "| %s | %s | %d | %d | %d | %d | %s | %s | %s |\n", s.Profile, s.Chain, s.Ops, s.Topups, s.NoDiscount, s.WithDiscnt, fmtE(s.MaxAbs), fmtE(s.MaxRel), mark)
	}

	b.WriteString("\n## Detail skenario\n")
	for _, r := range rec.records {
		fmt.Fprintf(&b, "\n### %s (%s) — %s\n\n", r.ID, r.Profile, r.Category)
		fmt.Fprintf(&b, "- **Masukan:** %s\n- **Diharapkan:** %s\n- **Aktual:** %s\n- **Lulus:** %v\n", r.Input, r.Expected, r.Actual, r.Pass)
		if len(r.Steps) > 0 {
			b.WriteString("- **Langkah:**\n")
			for i, s := range r.Steps {
				fmt.Fprintf(&b, "  %d. %s\n", i+1, s)
			}
		}
		if len(r.Measures) > 0 {
			b.WriteString("\n| Nilai | Acuan | Dekripsi mentah | e_abs | e_rel | <= toleransi |\n|---|---|---|---|---|---|\n")
			for _, m := range r.Measures {
				fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %v |\n", esc(m.Label), m.Expected, strconv.FormatFloat(m.Actual, 'f', 8, 64), fmtE(m.EAbs), m.ERel, m.Within)
			}
		}
		for _, n := range r.Notes {
			fmt.Fprintf(&b, "\n> %s\n", n)
		}
	}
	b.WriteString("\n## Contoh JSON\n\nContoh permintaan/respons (ciphertext dipotong, panjang aslinya dicatat) ada di `samples/`.\n")
	return b.String()
}
