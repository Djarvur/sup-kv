//go:build linux

package testutil

import "testing"

// HardwareBlock returns the perf run's one-line hardware description (RED
// stub: the real /proc reads land in GREEN).
func HardwareBlock() string { return "" }

// PerfDumpDir resolves the perf run's per-test dump directory (RED stub).
func PerfDumpDir(tb testing.TB) string {
	tb.Helper()
	return tb.TempDir()
}

// PerfDumpFSType names the filesystem backing path (RED stub).
func PerfDumpFSType(string) string { return "" }

// PerfDumpDevice renders path's device and rotational flag (RED stub).
func PerfDumpDevice(string) string { return "" }
