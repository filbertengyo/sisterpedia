// Package bench contains the benchmark statistics, raw-data format,
// environment capture and summary report used by cmd/bench (Table 2.7 of
// the assignment).
package bench

import (
	"errors"
	"math"
	"sort"
)

// ErrEmpty is returned for statistics of an empty sample.
var ErrEmpty = errors.New("empty sample")

func sorted(xs []float64) []float64 {
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	return s
}

// Median returns the median (mean of the two middle values for even n).
func Median(xs []float64) (float64, error) {
	if len(xs) == 0 {
		return 0, ErrEmpty
	}
	s := sorted(xs)
	n := len(s)
	if n%2 == 1 {
		return s[n/2], nil
	}
	return (s[n/2-1] + s[n/2]) / 2, nil
}

// NearestRankIndex returns the 1-based nearest-rank position ceil(p*n).
func NearestRankIndex(p float64, n int) int {
	k := int(math.Ceil(p * float64(n)))
	if k < 1 {
		k = 1
	}
	if k > n {
		k = n
	}
	return k
}

// Percentile returns the p-th percentile with the nearest-rank method: the
// element at 1-based position ceil(p*n) of the sorted sample.
func Percentile(xs []float64, p float64) (float64, error) {
	if len(xs) == 0 {
		return 0, ErrEmpty
	}
	if p <= 0 || p > 1 {
		return 0, errors.New("percentile must be in (0, 1]")
	}
	s := sorted(xs)
	return s[NearestRankIndex(p, len(s))-1], nil
}

// P95NearestRank returns the 95th percentile (nearest rank). For n = 30 it is
// the 29th smallest value.
func P95NearestRank(xs []float64) (float64, error) { return Percentile(xs, 0.95) }

// Summary of a sample.
type Summary struct {
	N      int
	Median float64
	P95    float64
	Min    float64
	Max    float64
	Mean   float64
}

// Summarize computes median, nearest-rank P95, min, max and mean.
func Summarize(xs []float64) (Summary, error) {
	if len(xs) == 0 {
		return Summary{}, ErrEmpty
	}
	s := sorted(xs)
	med, _ := Median(s)
	p95, _ := P95NearestRank(s)
	sum := 0.0
	for _, x := range s {
		sum += x
	}
	return Summary{N: len(s), Median: med, P95: p95, Min: s[0], Max: s[len(s)-1], Mean: sum / float64(len(s))}, nil
}
