//go:build !linux

// The non-Linux half of the stand package (TEST-05's second branch, D-03):
// the two-process real-socket harness and scenarios are linux-only — the
// macOS CI cell cannot run them, and darwin is development-only per owner
// decision 14.10 / ADR 0010. This file carries the limitation as ONE named
// PASSING test: the darwin cell executes it and prints the statement as a
// green named result, so the TEST-05 macOS-cell acceptance is carried by a
// visible test outcome, never a silent skip. A skip directive is forbidden
// here — the test must pass while reporting.

package stand

import (
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// darwinUntestedStatement is the loud statement the darwin CI cell prints
// as a green named test result (TEST-05's macOS-cell wording).
const darwinUntestedStatement = "darwin replication is untested: " +
	"the two-node real-socket scenarios in this package are linux-only " +
	"(decision 14.10 / ADR 0010, TEST-05) — CI's darwin cell reports here " +
	"instead of silently skipping socket tests"

// TestPlatform_darwinReplicationUntested is the single loud non-Linux
// report. It asserts nothing about sockets and always passes: reaching its
// body already proves the package compiled on this platform without the
// linux-tagged harness and scenarios. It also asserts the build-tag
// boundary directly — every other .go file in the package must be excluded
// for darwin — so a scenario file that leaks across the tag fails here with
// a named error instead of compiling untested socket code.
func TestPlatform_darwinReplicationUntested(t *testing.T) {
	t.Logf("%s", darwinUntestedStatement)

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller could not locate this test's source file")
	}
	pkgDir := filepath.Dir(thisFile)
	darwin := build.Default // value copy: the mutations below stay local
	darwin.GOOS = "darwin"
	darwin.GOARCH = "amd64"

	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatalf("read stand package dir %s: %v", pkgDir, err)
	}
	thisName := filepath.Base(thisFile)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || name == thisName {
			continue
		}
		matches, err := darwin.MatchFile(pkgDir, name)
		if err != nil {
			t.Fatalf("match build constraints of %s for darwin: %v", name, err)
		}
		if matches {
			t.Fatalf("%s compiles on darwin: a stand package file leaked across "+
				"the build-tag boundary — every other .go file here must be "+
				"linux-tagged, because the harness and scenarios are linux-only "+
				"(decision 14.10 / ADR 0010)", name)
		}
	}
}
