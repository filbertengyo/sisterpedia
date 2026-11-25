// Package monoclock is a high-resolution monotonic clock for benchmarks and
// operation traces. On Windows, time.Now's monotonic reading follows the
// system interrupt time (0.5-15.6 ms steps), which is too coarse for
// sub-millisecond CKKS operations, so QueryPerformanceCounter is used there;
// elsewhere the Go monotonic clock already has nanosecond resolution.
package monoclock

import "time"

// Now returns a monotonic timestamp in nanoseconds (arbitrary origin).
func Now() int64 { return now() }

// Since returns the time elapsed since a timestamp returned by Now.
func Since(start int64) time.Duration { return time.Duration(now() - start) }

// Between returns end - start as a duration.
func Between(start, end int64) time.Duration { return time.Duration(end - start) }
