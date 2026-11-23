//go:build linux

package testutil //nolint:testpackage // the tests assert the unexported perf helpers' exact text shape, which an external test package cannot reach.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot returns the repository root from the test package's working
// directory (internal/testutil), so the disk facts are asserted against the
// tree the module lives in.
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd(): %v", err)
	}
	return filepath.Join(wd, "..", "..")
}

// TestHardwareBlock pins the one-line block every perf run logs: the CPU,
// memory, load, build and disk facts the published numbers are conditioned
// on, with the race state and GOMAXPROCS the measurement ran under.
func TestHardwareBlock(t *testing.T) {
	t.Parallel()
	block := HardwareBlock()
	if !strings.HasPrefix(block, "hardware: ") {
		t.Fatalf("HardwareBlock() = %q, want one line starting %q", block, "hardware: ")
	}
	if strings.Contains(block, "\n") {
		t.Fatalf("HardwareBlock() must be one line, got:\n%s", block)
	}
	for _, want := range []string{"CPU=", "logical=", "RAM=", "load=", "go=", "race=", "GOMAXPROCS=", "root-disk=", "rotational="} {
		if !strings.Contains(block, want) {
			t.Fatalf("HardwareBlock() = %q, missing the %q fact", block, want)
		}
	}
}

// TestPerfDumpDirResolvesUnderBase pins the dump-directory guard: with
// SUPKV_PERF_DUMP_DIR set, every call gets its own existing subdirectory
// under the base (so two measurements never share a dump), and the test's
// cleanup owns their removal.
func TestPerfDumpDirResolvesUnderBase(t *testing.T) {
	base := t.TempDir()
	t.Setenv("SUPKV_PERF_DUMP_DIR", base)

	first := PerfDumpDir(t)
	second := PerfDumpDir(t)
	for _, dir := range []string{first, second} {
		if filepath.Dir(dir) != base {
			t.Fatalf("PerfDumpDir() = %q, want a fresh subdirectory under SUPKV_PERF_DUMP_DIR %q", dir, base)
		}
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			t.Fatalf("PerfDumpDir() = %q, want an existing directory (stat error %v)", dir, err)
		}
	}
	if first == second {
		t.Fatalf("PerfDumpDir() returned the same path %q twice, want a fresh subdirectory per call", first)
	}
}

// TestPerfDumpFSType pins the filesystem probe: the repo root resolves to a
// named filesystem, and a tmpfs path (when this host has one) reports tmpfs —
// the fact that makes a memory-backed dump visible and refusable.
func TestPerfDumpFSType(t *testing.T) {
	t.Parallel()
	name := PerfDumpFSType(repoRoot(t))
	if name == "" || name == "unknown" {
		t.Fatalf("PerfDumpFSType(repo root) = %q, want the backing filesystem's name", name)
	}

	const shmPath = "/dev/shm"
	if _, err := os.Stat(shmPath); err != nil {
		t.Logf("skipping the tmpfs assertion: %s is absent (%v)", shmPath, err)
		return
	}
	if got := PerfDumpFSType(shmPath); got != "tmpfs" {
		t.Logf("skipping the tmpfs assertion: %s reports %q on this host, not tmpfs", shmPath, got)
	}
}

// TestPerfDumpDevice pins the media facts: the repo root resolves to a device
// with a rotational flag, and an unresolvable path reports unknown instead of
// failing or returning an empty string.
func TestPerfDumpDevice(t *testing.T) {
	t.Parallel()
	got := PerfDumpDevice(repoRoot(t))
	if got == "" || !strings.Contains(got, ",rotational=") {
		t.Fatalf("PerfDumpDevice(repo root) = %q, want <device>,rotational=<0|1|unknown>", got)
	}
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if got := PerfDumpDevice(missing); got != "unknown" {
		t.Fatalf("PerfDumpDevice(%q) = %q, want %q for an unresolvable path", missing, got, "unknown")
	}
}
