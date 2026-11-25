//go:build windows

package monoclock

import (
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	qpc      = kernel32.NewProc("QueryPerformanceCounter")
	qpf      = kernel32.NewProc("QueryPerformanceFrequency")
	freq     int64
)

func init() {
	_, _, _ = qpf.Call(uintptr(unsafe.Pointer(&freq))) // #nosec G103 -- documented Win32 out-parameter
	if freq <= 0 {
		freq = 1
	}
}

func now() int64 {
	var c int64
	_, _, _ = qpc.Call(uintptr(unsafe.Pointer(&c))) // #nosec G103 -- documented Win32 out-parameter
	// split to avoid int64 overflow of c * 1e9
	return (c/freq)*1_000_000_000 + (c%freq)*1_000_000_000/freq
}
