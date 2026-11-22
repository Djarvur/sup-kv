//go:build linux

// The two-process deletion levels of TEST-02 (D-08) on the real binary.
// Level 1 (codec round-trip) is pinned by this plan's
// internal/udprepl/decode_negative_test.go TestWire_deleteRoundTrip; level
// 2 (simulator Cluster.Delete) by plan 02-15's
// TestSim_Delete_replicatesTombstone and the deletion mix plan 02-06
// landed in the chaos scenarios; levels 3 and 4 are HERE: a Delete on node
// A must surface as NotFound on node B with the tombstone durable in B's
// dump (the two-process level), and a node stopped across the delete must
// learn the tombstone after it starts (the node-down-during-delete level).
//
// Both scenarios are green pins of current behavior — no red gate, no
// production change. Every timing bound is derived in-test from the flags
// the nodes run with (the TestStand_emptyDiskRecovery pattern), never a
// magic constant, and the dump-grep assertions ride the JSONL format's
// greppable "deleted":true marker (readFinalDump is the harness seam).

package stand //nolint:testpackage // the deletion scenarios share the harness's unexported helpers (D-03/D-08); testpackage's external-package rule cannot express that pairing.

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"
)

// The deletion scenarios' constants: the two keys, the tombstone marker
// the JSONL dump exposes, the 16-byte version shape every delete response
// carries, and the sync response rate both nodes run with — pinned
// explicitly as a flag so the returning-peer bound derives from a value
// this package sets, not a production default it cannot read.
const (
	standDeleteReplKey = "stand/delete/replicated-key"
	standDeleteDownKey = "stand/delete/while-down-key"

	standDeleteTombstoneMarker = `"deleted":true`
	standDeleteVersionBytes    = 16
	standDeleteSyncRate        = 500
)

// TestStand_deleteReplication is TEST-02 level 3: two nodes converge a
// live value, a Delete on A propagates as the store's NotFound on B within
// the configured-timers bound, B's dump carries the tombstone's
// "deleted":true record line, and the delete-twice named case — a second
// Delete hits the store's ErrNotFound no-op without burning a version —
// leaves B holding exactly one record line for the key. B's final
// confirmed dump after SIGTERM proves the tombstone durable, not merely
// resident.
func TestStand_deleteReplication(t *testing.T) {
	t.Parallel()
	p, nodeA, nodeB := startStandPair(t)
	awaitServing(t, p.grpcA, servingDeadline)
	awaitServing(t, p.grpcB, servingDeadline)

	clientA := standClient(t, p.grpcA)
	clientB := standClient(t, p.grpcB)

	// Baseline: a live write on A that B serves — B's dump will hold the
	// data line the tombstone must replace.
	standSet(t, clientA, standDeleteReplKey)
	waitFor(t, standConvergeDeadline, "the baseline write on node B", func() bool {
		return standServing(t, clientB, standDeleteReplKey)
	})

	// The two-process level: Delete on A is NotFound on B (the API's one
	// mapped absence code, missing OR tombstoned).
	standDeleteOK(t, clientA, standDeleteReplKey)
	waitFor(t, standConvergeDeadline, "node B serving NotFound for the deleted key", func() bool {
		return standAbsent(t, clientB, standDeleteReplKey)
	})

	// The durable half: B's dump's single record line for the key is the
	// tombstone, marker intact.
	waitFor(t, standConvergeDeadline, "node B's dump holding the tombstone line", func() bool {
		return standDumpHoldsTombstone(p.dumpB, standDeleteReplKey)
	})

	// The delete-twice idempotency case at the two-process level: the
	// second Delete is ErrNotFound, the key stays absent, and B still holds
	// exactly one record line — the tombstone — for it.
	standDeleteAgain(t, clientA, standDeleteReplKey)
	if _, found := standGet(t, clientB, standDeleteReplKey); found {
		t.Fatalf("node B serves %q again after the no-op second Delete", standDeleteReplKey)
	}
	line := standDumpSingleLine(t, p.dumpB, standDeleteReplKey)
	if !strings.Contains(line, standDeleteTombstoneMarker) {
		t.Fatalf("node B's dump lost the tombstone after the second Delete: %s", line)
	}

	// B's final confirmed dump (D-19) keeps the single tombstone line: the
	// NotFound B serves is backed by disk, not just memory.
	signalAndWaitExit(t, nodeB)
	line = standDumpSingleLine(t, p.dumpB, standDeleteReplKey)
	if !strings.Contains(line, standDeleteTombstoneMarker) {
		t.Fatalf("node B's final dump does not hold the tombstone for %q: %s", standDeleteReplKey, line)
	}

	signalAndWaitExit(t, nodeA)
	assertNoPanic(t, nodeA, "the deleting node A")
	assertNoPanic(t, nodeB, "the replicating node B")
}

// TestStand_deleteWhileNodeDown is TEST-02 level 4: node B holds the
// pre-delete value and stops cleanly (its final dump keeps the data line);
// the Delete then happens on A while B is down; B restarts and must learn
// the tombstone — Get NotFound — within the returning-peer bound derived
// from the flags, and its dump must show the tombstone line for the key
// (the pre-delete value replaced, not shadowed).
func TestStand_deleteWhileNodeDown(t *testing.T) {
	t.Parallel()
	p := newStandPair(t)
	rateArg := "--sync-response-rate=" + strconv.Itoa(standDeleteSyncRate)
	nodeA := startStandNode(t, p, standNodeAID, rateArg)
	nodeB := startStandNode(t, p, standNodeBID, rateArg)
	awaitServing(t, p.grpcA, servingDeadline)
	awaitServing(t, p.grpcB, servingDeadline)

	clientA := standClient(t, p.grpcA)
	clientB := standClient(t, p.grpcB)

	// The pre-delete value converges onto B; B's clean stop leaves it in
	// B's dump — the sub-case's premise.
	standSet(t, clientA, standDeleteDownKey)
	waitFor(t, standConvergeDeadline, "the pre-delete value on node B", func() bool {
		return standServing(t, clientB, standDeleteDownKey)
	})
	signalAndWaitExit(t, nodeB)
	line := standDumpSingleLine(t, p.dumpB, standDeleteDownKey)
	if strings.Contains(line, standDeleteTombstoneMarker) {
		t.Fatalf("node B's pre-delete dump already holds a tombstone for %q: %s", standDeleteDownKey, line)
	}

	// The delete happens while B is down: A still holds the live value, so
	// the store writes the fresh-version tombstone rather than the
	// ErrNotFound no-op.
	standDeleteOK(t, clientA, standDeleteDownKey)

	nodeBRestart := startStandNode(t, p, standNodeBID, rateArg)
	awaitServing(t, p.grpcB, servingDeadline)

	// The returning-peer bound, derived from the flags both nodes run with:
	// one record round trip at the limiter's floor(1s/rate) spacing, B's
	// startup-gate window, one discovery round plus the startup
	// listing/request/reply ladder at the shrunken sync interval, and real
	// process scheduling slack on a loaded runner. B reloads the pre-delete
	// value from its dump; A's restart branch for B's new run_id ships the
	// listing, and the LWW law makes A's tombstone win.
	paced := time.Second / time.Duration(standDeleteSyncRate)
	margin := standStartupGate + 2*standHelloInterval + 4*standSyncInterval + 2*time.Second
	deadline := paced + margin
	waitFor(t, deadline, "the restarted node B learning the tombstone", func() bool {
		return standAbsent(t, clientB, standDeleteDownKey)
	})

	// The disk half of the sub-case: the tombstone replaced the pre-delete
	// value in B's dump's single record line for the key.
	waitFor(t, standConvergeDeadline, "the restarted node B's dump holding the tombstone line", func() bool {
		return standDumpHoldsTombstone(p.dumpB, standDeleteDownKey)
	})

	signalAndWaitExit(t, nodeBRestart)
	line = standDumpSingleLine(t, p.dumpB, standDeleteDownKey)
	if !strings.Contains(line, standDeleteTombstoneMarker) {
		t.Fatalf("node B's final dump does not hold the learned tombstone for %q: %s", standDeleteDownKey, line)
	}

	signalAndWaitExit(t, nodeA)
	assertNoPanic(t, nodeA, "node A through the delete-while-down scenario")
	assertNoPanic(t, nodeB, "node B's first run through the pre-delete dump")
	assertNoPanic(t, nodeBRestart, "the restarted node B learning the tombstone")
}

// standDelete tombstones key through the real gRPC endpoint and returns
// the response version; a NotFound is handed back to the caller for the
// delete-twice idempotency assertion, any other outcome is the caller's to
// judge.
func standDelete(t *testing.T, client apiproto.SupKVServiceClient, key string) ([]byte, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), rpcTimeout)
	defer cancel()
	resp, err := client.Delete(ctx, &apiproto.DeleteRequest{Key: key})
	if err != nil {
		return nil, err
	}
	return resp.GetVersion(), nil
}

// standDeleteOK fails the test unless the delete succeeds with a 16-byte
// fresh tombstone version.
func standDeleteOK(t *testing.T, client apiproto.SupKVServiceClient, key string) {
	t.Helper()
	ver, err := standDelete(t, client, key)
	if err != nil {
		t.Fatalf("Delete(%q): %v", key, err)
	}
	if len(ver) != standDeleteVersionBytes {
		t.Fatalf("Delete(%q) version = %d bytes, want %d", key, len(ver), standDeleteVersionBytes)
	}
}

// standDeleteAgain asserts the delete-twice idempotency: a Delete of an
// already-tombstoned key is the store's ErrNotFound no-op — the API's
// NotFound — and burns no version.
func standDeleteAgain(t *testing.T, client apiproto.SupKVServiceClient, key string) {
	t.Helper()
	_, err := standDelete(t, client, key)
	if status.Code(err) != codes.NotFound {
		t.Fatalf("second Delete(%q) = %v, want codes.NotFound (the already-tombstoned no-op)", key, err)
	}
}

// standDumpRecordLines returns the dump's record lines carrying the exact
// JSON key marker — the JSONL format's one-line-per-key shape.
func standDumpRecordLines(dump, key string) []string {
	marker := `"key":"` + key + `"`
	var lines []string
	for line := range strings.SplitSeq(dump, "\n") {
		if strings.Contains(line, marker) {
			lines = append(lines, line)
		}
	}
	return lines
}

// standDumpHoldsTombstone reports whether the dump file currently holds
// exactly one record line for key and that line carries the tombstone
// marker — the live dump-grep condition for waitFor (a missing or stale
// generation reads as not-yet).
func standDumpHoldsTombstone(dumpPath, key string) bool {
	dump, err := os.ReadFile(dumpPath)
	if err != nil {
		return false
	}
	lines := standDumpRecordLines(string(dump), key)
	return len(lines) == 1 && strings.Contains(lines[0], standDeleteTombstoneMarker)
}

// standDumpSingleLine returns the dump's exactly-one record line for key,
// failing when the dump holds none or more than one — the one-record-per-key
// contract the delete-twice assertion rests on.
func standDumpSingleLine(t *testing.T, dumpPath, key string) string {
	t.Helper()
	dump := readFinalDump(t, dumpPath)
	lines := standDumpRecordLines(dump, key)
	if len(lines) != 1 {
		t.Fatalf("dump %s holds %d record lines for %q, want exactly one:\n%s", dumpPath, len(lines), key, dump)
	}
	return lines[0]
}
