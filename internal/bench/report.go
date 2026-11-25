package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type key struct{ profile, metric, op string }

// Report builds bench/results/summary.md and bench/ENVIRONMENT.md from the
// raw CSVs in rawDir.
type Report struct {
	samples  map[key][]float64 // milliseconds
	order    map[string][]string
	sizes    map[string]map[string]SizeRow // artifact -> profile -> row
	sizeOrd  []string
	profiles []string
}

// BuildReport loads the raw data.
func BuildReport(rawDir string) (*Report, error) {
	samples, sizes, err := LoadRaw(rawDir)
	if err != nil {
		return nil, err
	}
	r := &Report{samples: map[key][]float64{}, order: map[string][]string{}, sizes: map[string]map[string]SizeRow{}}
	seenProfile := map[string]bool{}
	seenOp := map[string]bool{}
	for _, s := range samples {
		k := key{s.Profile, s.Metric, s.Operation}
		r.samples[k] = append(r.samples[k], float64(s.NS)/1e6)
		if !seenOp[s.Metric+"|"+s.Operation] {
			seenOp[s.Metric+"|"+s.Operation] = true
			r.order[s.Metric] = append(r.order[s.Metric], s.Operation)
		}
		if !seenProfile[s.Profile] && s.Profile != "all" {
			seenProfile[s.Profile] = true
			r.profiles = append(r.profiles, s.Profile)
		}
	}
	for _, z := range sizes {
		if r.sizes[z.Artifact] == nil {
			r.sizes[z.Artifact] = map[string]SizeRow{}
			r.sizeOrd = append(r.sizeOrd, z.Artifact)
		}
		r.sizes[z.Artifact][z.Profile] = z
		if !seenProfile[z.Profile] && z.Profile != "all" {
			seenProfile[z.Profile] = true
			r.profiles = append(r.profiles, z.Profile)
		}
	}
	sort.Strings(r.profiles)
	return r, nil
}

func (r *Report) stat(profile, metric, op string) (Summary, bool) {
	xs := r.samples[key{profile, metric, op}]
	s, err := Summarize(xs)
	return s, err == nil
}

func ms(v float64) string {
	switch {
	case v >= 100:
		return fmt.Sprintf("%.1f", v)
	case v >= 1:
		return fmt.Sprintf("%.3f", v)
	default:
		return fmt.Sprintf("%.4f", v)
	}
}

func kib(n int) string { return fmt.Sprintf("%.1f", float64(n)/1024) }

func (r *Report) table(b *strings.Builder, metric, title, note string) {
	ops := r.order[metric]
	if len(ops) == 0 {
		return
	}
	fmt.Fprintf(b, "\n### %s\n\n", title)
	if note != "" {
		fmt.Fprintf(b, "%s\n\n", note)
	}
	b.WriteString("| Operasi |")
	for _, p := range r.profiles {
		fmt.Fprintf(b, " %s median (ms) | %s P95 (ms) | n |", p, p)
	}
	b.WriteString("\n|---|")
	for range r.profiles {
		b.WriteString("---|---|---|")
	}
	b.WriteString("\n")
	for _, op := range ops {
		fmt.Fprintf(b, "| %s |", op)
		for _, p := range r.profiles {
			if s, ok := r.stat(p, metric, op); ok {
				fmt.Fprintf(b, " %s | %s | %d |", ms(s.Median), ms(s.P95), s.N)
			} else {
				b.WriteString(" - | - | - |")
			}
		}
		b.WriteString("\n")
	}
}

// WASM vs native pairs: (native metric, native op, wasm metric, wasm op, label).
var wasmPairs = [][5]string{
	{"keygen", "total", "keygen_wasm", "genKeys", "pembangkitan kunci (params+sk+pk+rlk)"},
	{"client", "encode+encrypt amount", "client_wasm", "encryptScalar amount", "encode+enkripsi nominal"},
	{"client", "encode+encrypt metadata", "client_wasm", "encryptVector metadata", "encode+enkripsi metadata (3 slot)"},
	{"client", "decrypt+decode balance", "client_wasm", "decryptScalar balance", "dekripsi+decode saldo"},
	{"client", "decrypt+decode metadata", "client_wasm", "decryptVector metadata", "dekripsi+decode metadata"},
}

// Markdown renders summary.md.
func (r *Report) Markdown(env *Environment, wasmEnv map[string]any) string {
	var b strings.Builder
	b.WriteString("# Ringkasan benchmark (Tabel 2.7)\n\n")
	fmt.Fprintf(&b, "Dibuat otomatis oleh `go run ./cmd/bench -report` pada %s dari data mentah di `bench/results/raw/`.\n\n", time.Now().UTC().Format(time.RFC3339))
	b.WriteString("**Prosedur.** Untuk setiap profil: 5 pengulangan pemanasan (tidak dicatat) lalu 30 pengukuran dengan masukan dan rangkaian operasi yang sama. ")
	b.WriteString("Waktu diukur dengan jam monotonik (`time.Now`/`time.Since` di Go, `performance.now()` di Node). ")
	b.WriteString("Median = nilai tengah (rata-rata dua nilai tengah bila n genap). ")
	b.WriteString("**P95 nearest-rank** = elemen ke-⌈0,95·n⌉ dari data terurut dengan posisi mulai 1 (n = 30 → elemen ke-29).\n\n")
	if env != nil {
		fmt.Fprintf(&b, "Lingkungan: %s, %d core logis, RAM %.1f GiB, %s %s, %s, %s (detail: `bench/ENVIRONMENT.md`).\n",
			env.CPU, env.Cores, float64(env.RAMBytes)/(1<<30), env.OSVersion, env.Arch, env.GoVersion, env.Lattigo)
	}

	b.WriteString("\n## Waktu\n")
	r.table(&b, "keygen", "Pembangkitan kunci (klien, native Go)", "Dipisahkan dari operasi layanan. `params` = pembentukan parameter (`NewParametersFromLiteral`, termasuk tabel NTT); `total` = jumlah keempat tahap pada run yang sama.")
	r.table(&b, "client", "Kriptografi klien (native Go, tanpa jaringan)", "`encode` dan `encrypt` diukur terpisah pada run yang sama; `encode+encrypt` = keduanya. Dekripsi `balance`/`total` pada level L-2 (hasil transaksi), `metadata` pada level L.")
	r.table(&b, "server", "Evaluasi server (tanpa jaringan dan basis data)", "Jalur kode yang sama dengan server (`internal/server/transact`). `Add`=topup, `Sub`=pengurangan potongan dan saldo, `Mult`=perkalian ciphertext×ciphertext (tensor, derajat 2), `Relin`/`Rescale` terpisah. `encrypt_*` = enkripsi konstanta katalog dengan pk pengguna; `align_*` = penyelarasan level (DropLevel). `topup.total`/`transact.total` = seluruh evaluasi.")
	r.table(&b, "e2e", "End-to-end (enkripsi klien → HTTP → evaluasi + SQLite → dekripsi)", "Klien Go native terhadap biner server nyata (proses terpisah, SQLite) melalui HTTP lokal.")
	r.table(&b, "keygen_wasm", "Pembangkitan kunci (WebAssembly, Node)", "Fungsi jembatan `genKeys` menggabungkan params+sk+pk+rlk dan serialisasi Base64 dalam satu panggilan; diukur gabungan.")
	r.table(&b, "client_wasm", "Kriptografi klien (WebAssembly, Node)", "Fungsi jembatan menggabungkan decode Base64 kunci (di-cache), encode, enkripsi, dan serialisasi Base64 dalam satu panggilan.")

	if len(r.order["keygen_wasm"]) > 0 || len(r.order["client_wasm"]) > 0 {
		b.WriteString("\n### WASM vs native (median)\n\n| Operasi |")
		for _, p := range r.profiles {
			fmt.Fprintf(&b, " %s native (ms) | %s WASM (ms) | rasio |", p, p)
		}
		b.WriteString("\n|---|")
		for range r.profiles {
			b.WriteString("---|---|---|")
		}
		b.WriteString("\n")
		for _, pr := range wasmPairs {
			fmt.Fprintf(&b, "| %s |", pr[4])
			for _, p := range r.profiles {
				n, okN := r.stat(p, pr[0], pr[1])
				w, okW := r.stat(p, pr[2], pr[3])
				switch {
				case okN && okW:
					fmt.Fprintf(&b, " %s | %s | %.2fx |", ms(n.Median), ms(w.Median), w.Median/n.Median)
				case okN:
					fmt.Fprintf(&b, " %s | - | - |", ms(n.Median))
				default:
					b.WriteString(" - | - | - |")
				}
			}
			b.WriteString("\n")
		}
	}

	if len(r.sizeOrd) > 0 {
		b.WriteString("\n## Ukuran artefak (data hasil serialisasi)\n\nUkuran biner = keluaran `MarshalBinary`/`MarshalJSON` Lattigo; Base64 = panjang string di JSON. 1 KiB = 1024 byte.\n\n| Artefak |")
		for _, p := range r.profiles {
			fmt.Fprintf(&b, " %s biner (byte) | %s biner (KiB) | %s Base64 (byte) | %s Base64 (KiB) |", p, p, p, p)
		}
		b.WriteString("\n|---|")
		for range r.profiles {
			b.WriteString("---|---|---|---|")
		}
		b.WriteString("\n")
		for _, a := range r.sizeOrd {
			fmt.Fprintf(&b, "| %s |", a)
			for _, p := range r.profiles {
				z, ok := r.sizes[a][p]
				if !ok {
					z, ok = r.sizes[a]["all"]
				}
				if ok {
					fmt.Fprintf(&b, " %d | %s | %d | %s |", z.Binary, kib(z.Binary), z.Base64, kib(z.Base64))
				} else {
					b.WriteString(" - | - | - | - |")
				}
			}
			b.WriteString("\n")
		}
	}

	if len(r.profiles) >= 2 {
		p1, p2 := r.profiles[0], r.profiles[1]
		b.WriteString("\n## Perbandingan otomatis " + p2 + " terhadap " + p1 + " (rasio median)\n\n| Besaran | Rasio |\n|---|---|\n")
		for _, x := range [][3]string{
			{"keygen", "total", "pembangkitan kunci total"}, {"keygen", "rlk", "relinearization key"},
			{"client", "encode+encrypt amount", "encode+enkripsi"}, {"client", "decrypt+decode balance", "dekripsi saldo"},
			{"server", "mul_1", "Mult"}, {"server", "relin_1", "Relin"}, {"server", "rescale_1", "Rescale"},
			{"server", "transact.total", "evaluasi Transact"}, {"e2e", "transact", "end-to-end Transact"},
		} {
			a, okA := r.stat(p1, x[0], x[1])
			c, okC := r.stat(p2, x[0], x[1])
			if okA && okC && a.Median > 0 {
				fmt.Fprintf(&b, "| %s | %.2fx |\n", x[2], c.Median/a.Median)
			}
		}
		for _, x := range [][2]string{{"public_key", "public key"}, {"relin_key", "relinearization key"}, {"ct_balance_fresh", "ciphertext saldo (level L)"}} {
			a, okA := r.sizes[x[0]][p1]
			c, okC := r.sizes[x[0]][p2]
			if okA && okC && a.Binary > 0 {
				fmt.Fprintf(&b, "| ukuran %s | %.2fx |\n", x[1], float64(c.Binary)/float64(a.Binary))
			}
		}
	}
	if wasmEnv != nil {
		js, _ := json.Marshal(wasmEnv)
		fmt.Fprintf(&b, "\nLingkungan WASM: `%s`\n", js)
	}
	return b.String()
}

// EnvironmentMarkdown renders bench/ENVIRONMENT.md.
func EnvironmentMarkdown(env *Environment, wasmEnv map[string]any) string {
	var b strings.Builder
	b.WriteString("# Lingkungan benchmark\n\nDitulis otomatis oleh `go run ./cmd/bench`. Angka di `bench/results/` hanya berlaku untuk lingkungan ini.\n\n| Komponen | Nilai |\n|---|---|\n")
	if env != nil {
		rows := [][2]string{
			{"Waktu pengukuran (UTC)", env.Timestamp}, {"CPU", env.CPU}, {"Core logis", fmt.Sprint(env.Cores)},
			{"RAM", fmt.Sprintf("%.1f GiB (%d byte)", float64(env.RAMBytes)/(1<<30), env.RAMBytes)},
			{"Sistem operasi", env.OSVersion + " (" + env.OS + "/" + env.Arch + ")"}, {"Bahasa / runtime", "Go " + env.GoVersion + ", GOMAXPROCS=" + fmt.Sprint(env.GOMAXPROCS)},
			{"Pustaka CKKS", env.Lattigo + " (NTT bawaan)"}, {"Commit", fmt.Sprintf("%s (perubahan belum di-commit: %v)", env.GitCommit, env.GitDirty)},
		}
		for _, r := range rows {
			fmt.Fprintf(&b, "| %s | %s |\n", r[0], r[1])
		}
	}
	for _, k := range []string{"node", "v8", "platform", "go_wasm"} {
		if v, ok := wasmEnv[k]; ok {
			fmt.Fprintf(&b, "| WASM %s | %v |\n", k, v)
		}
	}
	b.WriteString("\nPerintah ulang: `make bench` (Linux/macOS/WSL) atau `scripts/run_bench.ps1` (Windows).\n")
	return b.String()
}

// ReadEnv loads a JSON environment file (nil if missing).
func ReadEnv(path string) (*Environment, map[string]any) {
	var e Environment
	b, err := os.ReadFile(path) // #nosec G304 -- benchmark output directory
	if err != nil {
		return nil, nil
	}
	if json.Unmarshal(b, &e) != nil {
		return nil, nil
	}
	return &e, nil
}

// ReadAny loads a JSON object file (nil if missing).
func ReadAny(path string) map[string]any {
	b, err := os.ReadFile(path) // #nosec G304 -- benchmark output directory
	if err != nil {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return nil
	}
	return m
}

// WriteReport writes summary.md (in resultsDir) and ENVIRONMENT.md (in benchDir).
func WriteReport(resultsDir, benchDir string) error {
	r, err := BuildReport(filepath.Join(resultsDir, "raw"))
	if err != nil {
		return err
	}
	env, _ := ReadEnv(filepath.Join(resultsDir, "raw", "environment_native.json"))
	wasmEnv := ReadAny(filepath.Join(resultsDir, "raw", "environment_wasm.json"))
	if err := os.WriteFile(filepath.Join(resultsDir, "summary.md"), []byte(r.Markdown(env, wasmEnv)), 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(benchDir, "ENVIRONMENT.md"), []byte(EnvironmentMarkdown(env, wasmEnv)), 0o600)
}
