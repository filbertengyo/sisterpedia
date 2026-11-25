package monoclock

import (
	"testing"
	"time"
)

func TestMonotonicAndFine(t *testing.T) {
	a := Now()
	time.Sleep(2 * time.Millisecond)
	if d := Since(a); d < time.Millisecond || d > time.Second {
		t.Fatalf("Since = %v", d)
	}
	// resolution: consecutive readings must eventually differ by < 100us
	minStep := time.Hour
	prev := Now()
	for i := 0; i < 100000 && minStep > 100*time.Microsecond; i++ {
		cur := Now()
		if cur < prev {
			t.Fatal("clock went backwards")
		}
		if cur > prev && time.Duration(cur-prev) < minStep {
			minStep = time.Duration(cur - prev)
		}
		prev = cur
	}
	if minStep > 100*time.Microsecond {
		t.Fatalf("resolution too coarse: %v", minStep)
	}
}
