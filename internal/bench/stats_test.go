package bench

import (
	"errors"
	"math/rand/v2"
	"testing"
)

func seq(n int) []float64 {
	xs := make([]float64, n)
	for i := range xs {
		xs[i] = float64(i + 1)
	}
	// shuffle: functions must sort internally
	r := rand.New(rand.NewPCG(1, 2)) // #nosec G404 -- deterministic shuffle for a test
	r.Shuffle(n, func(i, j int) { xs[i], xs[j] = xs[j], xs[i] })
	return xs
}

func TestP95NearestRankThirtyRuns(t *testing.T) {
	// n = 30: ceil(0.95 * 30) = ceil(28.5) = 29 -> the 29th smallest value
	if k := NearestRankIndex(0.95, 30); k != 29 {
		t.Fatalf("index = %d, want 29", k)
	}
	p, err := P95NearestRank(seq(30))
	if err != nil || p != 29 {
		t.Fatalf("P95 = %v (%v), want 29", p, err)
	}
}

func TestPercentileEdges(t *testing.T) {
	cases := []struct {
		n    int
		want float64
	}{{1, 1}, {2, 2}, {10, 10}, {20, 19}, {100, 95}, {35, 34}}
	for _, c := range cases {
		if p, _ := P95NearestRank(seq(c.n)); p != c.want {
			t.Errorf("n=%d: P95 = %v, want %v", c.n, p, c.want)
		}
	}
	if _, err := P95NearestRank(nil); !errors.Is(err, ErrEmpty) {
		t.Error("empty sample must fail")
	}
	if _, err := Percentile([]float64{1}, 0); err == nil {
		t.Error("p=0 must fail")
	}
}

func TestMedian(t *testing.T) {
	if m, _ := Median(seq(30)); m != 15.5 {
		t.Errorf("median of 1..30 = %v", m)
	}
	if m, _ := Median(seq(5)); m != 3 {
		t.Errorf("median of 1..5 = %v", m)
	}
	s, err := Summarize([]float64{5, 1, 3})
	if err != nil || s.N != 3 || s.Median != 3 || s.Min != 1 || s.Max != 5 || s.Mean != 3 || s.P95 != 5 {
		t.Errorf("summary %+v %v", s, err)
	}
}
