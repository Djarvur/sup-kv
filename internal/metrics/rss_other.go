//go:build !linux

package metrics

import (
	"runtime"
	"syscall"
)

// currentRSS returns the non-Linux fallback for process_resident_memory_bytes:
// [syscall.Getrusage]'s ru_maxrss, which is the process's PEAK resident set,
// not its current one (OBSV-03, D-05). Linux is the production target and
// reports the current size from /proc/self/statm (rss_linux.go); darwin is a
// development-only platform per ADR 0010, so the peak is the documented
// fallback there and the family Help says so — the value is never presented
// as current. ru_maxrss is bytes on darwin and kibibytes elsewhere,
// normalized here to bytes.
func currentRSS() float64 {
	var ru syscall.Rusage
	err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	if err != nil {
		return 0
	}
	rss := float64(ru.Maxrss)
	if rss > 0 && runtime.GOOS != "darwin" {
		rss *= 1024
	}
	return rss
}
