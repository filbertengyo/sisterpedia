//go:build !windows

package monoclock

import "time"

var base = time.Now()

func now() int64 { return int64(time.Since(base)) }
