//go:build linux

// Process-level persistence-load scenarios (PERSIST-02, D-02/D-04): the
// real binary started over a pre-seeded dump directory must report every
// load outcome exactly once — one WARN naming the main path and cause for a
// .prev fallback, one ERROR carrying each failing candidate for an unusable
// chain, nothing at all for a fresh start — while serving the records the
// accepted generation holds. The dump files are seeded by running the real
// binary (never hand-built bytes), so the scenarios and the loader agree on
// the production format and checksums by construction.
package stand //nolint:testpackage // the persistence scenarios share the harness's unexported node launchers and log capture (D-03); testpackage's external-package rule cannot express that pairing.

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The persistence-load scenario constants: one key every seeded generation
// carries, a generation-level garbage payload (its first line is not a JSON
// object, so the decoder rejects the whole generation before considering any
// record line), and the two static log messages the load classes emit.
const (
	persistLoadKey = "stand/persist/load-key"

	persistGarbage = "not a dump generation\n"

	persistBackupMsg   = "Dump generation fallback"
	persistUnusableMsg = "No usable dump generation"
)

// seedPersistDump runs one node over the pair's dump path, writes key with
// its scenario value, and SIGKILLs the node after Set returned — the dump is
// exactly what the client's write was confirmed against (PERS-04), so the
// kill cannot be blamed for a loss.
func seedPersistDump(t *testing.T, p *standPair, key string) {
	t.Helper()
	node := startStandNode(t, p, standNodeAID)
	awaitServing(t, p.grpcA, servingDeadline)
	standSet(t, standClient(t, p.grpcA), key)
	killAndReap(t, node)
	mainDump := readFinalDump(t, p.dumpA)
	if !strings.Contains(mainDump, `"key":"`+key+`"`) {
		t.Fatalf("the seeded dump %s does not carry %q:\n%s", p.dumpA, key, mainDump)
	}
}

// persistWriteFile overwrites path with the test fixture bytes.
func persistWriteFile(t *testing.T, path, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), dumpFileMode)
	if err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// persistRenameMainToPrev rotates the seeded main generation to .prev.
func persistRenameMainToPrev(t *testing.T, p *standPair) {
	t.Helper()
	err := os.Rename(p.dumpA, p.dumpA+".prev")
	if err != nil {
		t.Fatalf("rename %s to .prev: %v", p.dumpA, err)
	}
}

// persistRemovePrev removes the seeded .prev generation when the scenario
// needs an absent one; a missing file is the same state.
func persistRemovePrev(t *testing.T, p *standPair) {
	t.Helper()
	err := os.Remove(p.dumpA + ".prev")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("remove %s.prev: %v", p.dumpA, err)
	}
}

// persistMessageLines returns the process log's records carrying msg — the
// structured-log JSON default spells the static message as "msg":"...".
func persistMessageLines(output, msg string) []string {
	marker := `"msg":"` + msg + `"`
	var lines []string
	for line := range strings.SplitSeq(output, "\n") {
		if strings.Contains(line, marker) {
			lines = append(lines, line)
		}
	}
	return lines
}

// awaitPersistMessage waits until msg has been logged and returns its first
// record — the load logging happens synchronously at startup, so this only
// absorbs the exec pump's delivery latency.
func awaitPersistMessage(t *testing.T, node *standNode, msg string) string {
	t.Helper()
	var line string
	waitFor(t, servingDeadline, "the "+msg+" log record", func() bool {
		lines := persistMessageLines(node.stderr.String(), msg)
		if len(lines) == 0 {
			return false
		}
		line = lines[0]
		return true
	})
	return line
}

// persistAssertMessageCount asserts the exact number of msg records after
// the process exited, so the complete log is observed.
func persistAssertMessageCount(t *testing.T, node *standNode, msg string, want int) {
	t.Helper()
	if got := len(persistMessageLines(node.stderr.String(), msg)); got != want {
		t.Fatalf("%q appeared %d times in the process log, want %d:\n%s", msg, got, want, node.stderr.String())
	}
}

// persistCorruptFiles returns the preserved .corrupt-* names beside the
// configured dump path.
func persistCorruptFiles(t *testing.T, p *standPair) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(p.dumpA))
	if err != nil {
		t.Fatalf("read the dump directory: %v", err)
	}
	prefix := filepath.Base(p.dumpA) + ".corrupt-"
	var names []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			names = append(names, entry.Name())
		}
	}
	return names
}

// TestStand_persistLoadBackup starts over a good .prev and a corrupt main:
// exactly one WARN names the main path and its cause, the corrupt generation
// is preserved under .corrupt-*, and the .prev record is served.
func TestStand_persistLoadBackup(t *testing.T) {
	t.Parallel()
	p := newStandPair(t)
	seedPersistDump(t, p, persistLoadKey)
	persistRenameMainToPrev(t, p)
	persistWriteFile(t, p.dumpA, persistGarbage)

	node := startStandLoggedNode(t, p, standNodeAID)
	awaitServing(t, p.grpcA, servingDeadline)
	line := awaitPersistMessage(t, node, persistBackupMsg)
	if !strings.Contains(line, `"main_path":"`+p.dumpA+`"`) {
		t.Fatalf("the fallback WARN does not name the main path %s:\n%s", p.dumpA, line)
	}
	if !strings.Contains(line, `"cause":"`) || strings.Contains(line, `"cause":""`) {
		t.Fatalf("the fallback WARN carries no cause text:\n%s", line)
	}
	if !strings.Contains(line, `"preserve_error":""`) {
		t.Fatalf("the fallback WARN reports a preservation failure for the successful preserve:\n%s", line)
	}

	value, found := standGet(t, standClient(t, p.grpcA), persistLoadKey)
	if !found || value != standValue(persistLoadKey) {
		t.Fatalf("Get(%q) after the .prev fallback = (%q, %t), want the .prev record", persistLoadKey, value, found)
	}
	if corrupt := persistCorruptFiles(t, p); len(corrupt) != 1 {
		t.Fatalf("preserved corrupt generations = %v, want exactly the corrupt main", corrupt)
	}

	signalAndWaitExit(t, node)
	persistAssertMessageCount(t, node, persistBackupMsg, 1)
	persistAssertMessageCount(t, node, persistUnusableMsg, 0)
	assertNoPanic(t, node, "the .prev fallback node")
}

// TestStand_persistLoadBackupAbsentMain starts over a good .prev and no
// main: the one WARN still names the main path, with the fs.ErrNotExist
// cause, and the .prev record is served.
func TestStand_persistLoadBackupAbsentMain(t *testing.T) {
	t.Parallel()
	p := newStandPair(t)
	seedPersistDump(t, p, persistLoadKey)
	persistRenameMainToPrev(t, p)

	node := startStandLoggedNode(t, p, standNodeAID)
	awaitServing(t, p.grpcA, servingDeadline)
	line := awaitPersistMessage(t, node, persistBackupMsg)
	if !strings.Contains(line, `"main_path":"`+p.dumpA+`"`) {
		t.Fatalf("the absent-main WARN does not name the main path %s:\n%s", p.dumpA, line)
	}
	if !strings.Contains(line, `"cause":"`+fs.ErrNotExist.Error()+`"`) {
		t.Fatalf("the absent-main WARN does not name the fs.ErrNotExist cause %q:\n%s", fs.ErrNotExist.Error(), line)
	}
	if !strings.Contains(line, `"preserve_error":""`) {
		t.Fatalf("the absent-main WARN reports a preservation failure for an absent file:\n%s", line)
	}

	value, found := standGet(t, standClient(t, p.grpcA), persistLoadKey)
	if !found || value != standValue(persistLoadKey) {
		t.Fatalf("Get(%q) after the absent-main fallback = (%q, %t), want the .prev record", persistLoadKey, value, found)
	}
	if corrupt := persistCorruptFiles(t, p); len(corrupt) != 0 {
		t.Fatalf("an absent main left preserved corrupt generations %v, want none", corrupt)
	}

	signalAndWaitExit(t, node)
	persistAssertMessageCount(t, node, persistBackupMsg, 1)
	persistAssertMessageCount(t, node, persistUnusableMsg, 0)
	assertNoPanic(t, node, "the absent-main fallback node")
}

// TestStand_persistLoadUnusable starts over two corrupt generations: exactly
// one ERROR carries each failing candidate's path and cause, both files are
// preserved, and the node starts empty (serving).
func TestStand_persistLoadUnusable(t *testing.T) {
	t.Parallel()
	p := newStandPair(t)
	seedPersistDump(t, p, persistLoadKey)
	persistRenameMainToPrev(t, p)
	persistWriteFile(t, p.dumpA, persistGarbage)
	persistWriteFile(t, p.dumpA+".prev", persistGarbage)

	node := startStandLoggedNode(t, p, standNodeAID)
	awaitServing(t, p.grpcA, servingDeadline)
	line := awaitPersistMessage(t, node, persistUnusableMsg)
	for _, fragment := range []string{
		`"main_path":"` + p.dumpA + `"`,
		`"main_cause":"`,
		`"main_preserve_error":""`,
		`"prev_path":"` + p.dumpA + `.prev"`,
		`"prev_cause":"`,
		`"prev_preserve_error":""`,
	} {
		if !strings.Contains(line, fragment) {
			t.Fatalf("the unusable ERROR lacks %s:\n%s", fragment, line)
		}
	}
	if strings.Contains(line, `"main_cause":""`) || strings.Contains(line, `"prev_cause":""`) {
		t.Fatalf("the unusable ERROR carries an empty candidate cause:\n%s", line)
	}

	if value, found := standGet(t, standClient(t, p.grpcA), persistLoadKey); found {
		t.Fatalf("Get(%q) after an unusable load = (%q, true), want an empty store", persistLoadKey, value)
	}
	if corrupt := persistCorruptFiles(t, p); len(corrupt) != 2 {
		t.Fatalf("preserved corrupt generations = %v, want both candidates", corrupt)
	}

	signalAndWaitExit(t, node)
	persistAssertMessageCount(t, node, persistUnusableMsg, 1)
	persistAssertMessageCount(t, node, persistBackupMsg, 0)
	assertNoPanic(t, node, "the unusable-load node")
}

// TestStand_persistLoadUnusableSingle starts over one corrupt generation and
// no other: the one ERROR names that candidate and its cause, with no
// phantom second entry, and the node starts empty (serving).
func TestStand_persistLoadUnusableSingle(t *testing.T) {
	t.Parallel()
	p := newStandPair(t)
	seedPersistDump(t, p, persistLoadKey)
	persistRemovePrev(t, p)
	persistWriteFile(t, p.dumpA, persistGarbage)

	node := startStandLoggedNode(t, p, standNodeAID)
	awaitServing(t, p.grpcA, servingDeadline)
	line := awaitPersistMessage(t, node, persistUnusableMsg)
	if !strings.Contains(line, `"main_path":"`+p.dumpA+`"`) || !strings.Contains(line, `"main_cause":"`) {
		t.Fatalf("the single-candidate ERROR does not name the corrupt main and its cause:\n%s", line)
	}
	if strings.Contains(line, "prev_path") || strings.Contains(line, p.dumpA+".prev") {
		t.Fatalf("the single-candidate ERROR carries a phantom .prev entry:\n%s", line)
	}

	if value, found := standGet(t, standClient(t, p.grpcA), persistLoadKey); found {
		t.Fatalf("Get(%q) after an unusable load = (%q, true), want an empty store", persistLoadKey, value)
	}
	if corrupt := persistCorruptFiles(t, p); len(corrupt) != 1 {
		t.Fatalf("preserved corrupt generations = %v, want exactly the single corrupt main", corrupt)
	}

	signalAndWaitExit(t, node)
	persistAssertMessageCount(t, node, persistUnusableMsg, 1)
	persistAssertMessageCount(t, node, persistBackupMsg, 0)
	assertNoPanic(t, node, "the single-candidate unusable-load node")
}

// TestStand_persistLoadPrompt starts over an empty directory: neither a WARN
// nor an ERROR is emitted.
func TestStand_persistLoadPrompt(t *testing.T) {
	t.Parallel()
	p := newStandPair(t)
	node := startStandLoggedNode(t, p, standNodeAID)
	awaitServing(t, p.grpcA, servingDeadline)
	signalAndWaitExit(t, node)
	persistAssertMessageCount(t, node, persistBackupMsg, 0)
	persistAssertMessageCount(t, node, persistUnusableMsg, 0)
	assertNoPanic(t, node, "the fresh-start node")
}
