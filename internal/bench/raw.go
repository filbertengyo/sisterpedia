package bench

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Sample is one timed run (raw CSV row: profile,metric,operation,run,ns).
type Sample struct {
	Profile   string
	Metric    string // keygen | client | server | e2e (native) or *_wasm
	Operation string
	Run       int // 1-based, warm-up runs are not recorded
	NS        int64
}

// SizeRow is one artifact size (raw CSV: profile,artifact,binary_bytes,base64_bytes).
type SizeRow struct {
	Profile  string
	Artifact string
	Binary   int
	Base64   int
}

// Recorder accumulates samples for one profile and metric.
type Recorder struct {
	Profile, Metric string
	Samples         []Sample
}

// Add records a duration for operation at run (1-based).
func (r *Recorder) Add(operation string, run int, d time.Duration) {
	r.Samples = append(r.Samples, Sample{Profile: r.Profile, Metric: r.Metric, Operation: operation, Run: run, NS: d.Nanoseconds()})
}

// WriteSamples writes raw/<profile>_<metric>.csv.
func WriteSamples(dir string, samples []Sample) error {
	if len(samples) == 0 {
		return nil
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	name := filepath.Join(dir, fmt.Sprintf("%s_%s.csv", samples[0].Profile, samples[0].Metric))
	f, err := os.Create(name) // #nosec G304 -- benchmark output directory
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"profile", "metric", "operation", "run", "ns"})
	for _, s := range samples {
		_ = w.Write([]string{s.Profile, s.Metric, s.Operation, strconv.Itoa(s.Run), strconv.FormatInt(s.NS, 10)})
	}
	w.Flush()
	return w.Error()
}

// WriteSizes writes raw/<profile>_sizes.csv.
func WriteSizes(dir, profile string, rows []SizeRow) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, profile+"_sizes.csv")) // #nosec G304 -- benchmark output directory
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"profile", "artifact", "binary_bytes", "base64_bytes"})
	for _, r := range rows {
		_ = w.Write([]string{r.Profile, r.Artifact, strconv.Itoa(r.Binary), strconv.Itoa(r.Base64)})
	}
	w.Flush()
	return w.Error()
}

func readCSV(path string) ([][]string, error) {
	f, err := os.Open(path) // #nosec G304 -- benchmark output directory
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	var rows [][]string
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		rows = append(rows, rec)
	}
	return rows, nil
}

// LoadRaw reads every timing and size CSV in dir.
func LoadRaw(dir string) ([]Sample, []SizeRow, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	var samples []Sample
	var sizes []SizeRow
	names := []string{}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".csv") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, n := range names {
		rows, err := readCSV(filepath.Join(dir, n))
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", n, err)
		}
		if len(rows) == 0 {
			continue
		}
		switch strings.Join(rows[0], ",") {
		case "profile,metric,operation,run,ns":
			for _, r := range rows[1:] {
				run, _ := strconv.Atoi(r[3])
				ns, _ := strconv.ParseInt(r[4], 10, 64)
				samples = append(samples, Sample{Profile: r[0], Metric: r[1], Operation: r[2], Run: run, NS: ns})
			}
		case "profile,artifact,binary_bytes,base64_bytes":
			for _, r := range rows[1:] {
				b, _ := strconv.Atoi(r[2])
				b64, _ := strconv.Atoi(r[3])
				sizes = append(sizes, SizeRow{Profile: r[0], Artifact: r[1], Binary: b, Base64: b64})
			}
		}
	}
	return samples, sizes, nil
}
