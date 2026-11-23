//go:build !linux

package testutil

import (
	"fmt"
	"runtime"
	"testing"
)

// HardwareBlock returns the documented non-Linux block: the build, race and
// GOMAXPROCS facts, with the /proc and disk facts marked unavailable. Linux
// is the only perf target (the stand harness is linux-only; darwin stays
// development-only per 14.10), so this block exists to keep the helper pair
// compiling, never to measure anything.
func HardwareBlock() string {
	return fmt.Sprintf(
		"hardware: CPU=unknown; logical=%d; RAM=unknown; load=unknown; root-disk=unknown; go=%s/%s %s; module=unknown; race=%s; GOMAXPROCS=%d; proc=unavailable",
		runtime.NumCPU(),
		runtime.GOOS,
		runtime.GOARCH,
		runtime.Version(),
		raceLabel(),
		runtime.GOMAXPROCS(0),
	)
}

// PerfDumpDir resolves the perf run's dump directory: the same SUPKV_PERF_DUMP_DIR
// rule as Linux, without a filesystem probe (there is no perf target here).
func PerfDumpDir(tb testing.TB) string {
	tb.Helper()
	return perfDumpDir(tb, PerfDumpFSType, PerfDumpDevice)
}

// PerfDumpFSType returns perfUnknown: Linux is the only perf target (14.10
// keeps darwin development-only), so no filesystem probe exists here.
func PerfDumpFSType(string) string { return perfUnknown }

// PerfDumpDevice returns perfUnknown: Linux is the only perf target (14.10
// keeps darwin development-only), so no mount/rotational probe exists here.
func PerfDumpDevice(string) string { return perfUnknown }
