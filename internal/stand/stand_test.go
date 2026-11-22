//go:build linux

// The stand scenarios: the two-process half of the failure-scenario suite
// (TEST-04's exec scenarios) moved wholesale from cmd/sup-kv's
// scenario_test.go (D-03: moved/extended, never duplicated) and renamed to
// the stand namespace — kill -9 one node then restart with convergence
// and no operator action, both-nodes-down restart with state preserved
// through a corrupted dump record, SIGTERM's final confirmed dump
// provably covering replicated writes, an empty-disk node recovering from
// a live peer bounded by the sync response rate, and the NEW F2c
// oversize-key pin. These are the only scenarios that prove the artifact
// itself — process lifecycle, real signal delivery, real dump files on
// disk — so they extend the stand harness (stand.go: build-once, launch/
// reap, signalAndWaitExit/assertExitCode, awaitServing/supKVConn), never
// rebuild it.
//
// Every scenario rides the ASYMMETRIC-PORT topology (the empirically
// load-bearing finding): node A binds P1 and broadcasts to
// 127.255.255.255:P2, node B binds P2 and broadcasts to 127.255.255.255:P1
// — every broadcast and every unicast (the Ack and DataRequest answers
// the shared-port smoke topology cannot carry) crosses to exactly the
// peer, so convergence assertions are deterministic. The one platform
// capability that buys — loopback broadcast fan-out — is probed up front
// and, per TEST-05's Linux branch, a probe miss is a FATAL naming the
// probe, never a skip (stand.go's probe contract; the darwin report is
// the same-wave plan 02-14 named test).

package stand //nolint:testpackage // the plan mandates internal tests so the scenarios share the harness's unexported helpers (D-03); testpackage's external-package rule cannot express that pairing.

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
)

// standNightly is the scale-selection hook: SUPKV_STAND_NIGHTLY=1 selects
// the full documented scale for the nightly run (plan 11 wires the
// selection); the default scale keeps every scenario CI-viable (the
// shrunken-timer pattern below) exactly as the source suite ran.
var standNightly = os.Getenv("SUPKV_STAND_NIGHTLY") == "1"

// TestMain wires the build-once binary eagerly: a failed build is
// surfaced here and by every scenario through standBinary's loud fatal —
// never a silent pass (TEST-05). It announces the nightly scale mode for
// the run log and removes the build temp dir after the suite.
func TestMain(m *testing.M) {
	if standNightly {
		fmt.Fprintln(os.Stderr, "sup-kv stand: nightly scale selected (SUPKV_STAND_NIGHTLY=1)")
	}
	_, buildErr := buildStandBinary(context.Background())
	if buildErr != nil {
		fmt.Fprintf(os.Stderr, "sup-kv stand: build failed: %v\n", buildErr)
	}
	code := m.Run()
	cleanupStandBinary()
	if code != 0 {
		fmt.Fprintf(os.Stderr, "sup-kv stand: tests failed with code %d\n", code)
	}
}

// Scenario constants: the timer-derived convergence bound, the per-
// scenario keys, and the empty-disk scale — the source suite's scenario
// constants moved verbatim under the stand namespace.
const (
	// standConvergeDeadline bounds one write-to-visible cycle on either
	// endpoint, derived from the shrunken timers rather than wall-clock
	// precision: one hello round for discovery, a few sync rounds for the
	// listing/request/replay ladder, and real-process scheduling slack on
	// a loaded runner — twenty of each interval keeps every contribution
	// covered with the harness's generous-deadline discipline (a healthy
	// pair converges in well under one second).
	standConvergeDeadline = 20*standHelloInterval + 20*standSyncInterval

	standKeyKillA = "stand/kill9/key-on-a"
	standKeyKillB = "stand/kill9/key-on-b"

	standKeyBothA = "stand/bothdown/key-on-a"
	standKeyBothB = "stand/bothdown/key-on-b"

	standKeySigterm = "stand/sigterm/key-on-b"

	// standDiscardLine is the startup line main prints when the loader
	// discarded undecodable records (startEngine's discarded-count log) —
	// the proof a corrupted record took the per-record discard path, not
	// the whole-generation .prev fallback.
	standDiscardLine = "discarded 1 undecodable dumped records at startup"

	// The empty-disk scenario's default scale: N keys on the seeded node
	// and the shrunken sync response rate both nodes run with — the
	// documented recovery bound is N/Rate seconds of paced egress
	// (floor(1s/Rate) per datagram, the repl core's limiter arithmetic),
	// small enough that the whole scenario stays far inside CI deadlines.
	emptyDiskKeys = 20
	emptyDiskRate = 50

	// The documented nightly scale (plan 11's SUPKV_STAND_NIGHTLY=1
	// selection): the review's F9 row — "Empty-disk B, A holds 10k × 1 KiB
	// written while B was down"
	// (docs/reviews/2026-11-19-v1.0/evidence/stand/RESULTS.md, F9) — run
	// with the documented default flags the reviewer ran: hello 1 s, sync
	// 30 s, sync_response_rate 500 (cmd/sup-kv main.go's defaults; the
	// startup gate's default 1 s is the harness's own standStartupGate).
	// The bound stays the derived-from-flags expression below, scaling
	// with N.
	emptyDiskKeysNightly  = 10000
	emptyDiskRateNightly  = 500
	emptyDiskValueBytes   = 1024
	emptyDiskHelloNightly = time.Second
	emptyDiskSyncNightly  = 30 * time.Second

	// The nightly seeding pool mirrors the review's preload profile — 64
	// workers round-robin over 8 connections (RESULTS.md §2: ~700 writes/s
	// at the 10 000-key state with dump confirmation); a sequential loop
	// would spend a quarter-hour on the seeding alone.
	// emptyDiskPollEvery throttles the nightly full-listing poll: waitFor's
	// 10 ms cadence is far too hot for a 10 000-key List.
	emptyDiskSeedWorkers    = 64
	emptyDiskSeedConns      = 8
	emptyDiskSeedRPCTimeout = 30 * time.Second
	emptyDiskPollEvery      = 500 * time.Millisecond

	// standEmptyKeyFmt derives the seeded keys; standEmptyKeyPrefix scopes
	// the nightly listing poll; standKeyFromB is the recovered node's own
	// post-recovery write.
	standEmptyKeyFmt    = "stand/emptydisk/key-%02d"
	standEmptyKeyPrefix = "stand/emptydisk/"
	standKeyFromB       = "stand/emptydisk/from-b"
)

// The F2c scenario's constants: the review's 70 000-character key, which
// the API now rejects before it can reach the wire, and the design-default
// value limit the boundary record pairs with a maximal key (the stand nodes
// run without --value-max-bytes, so 61 440 is their accepted maximum).
const (
	standOversizeKeyLen     = 70000
	standBoundaryValueBytes = 61440
)

// TestStand_killNineRestart is the flagship TEST-04 scenario on the real
// binary: two nodes converge a write each way, node A dies to a kill -9
// (the SIGKILL exit shape, never SIGTERM's), node B keeps serving both
// keys through the peer's death, and A restarted with the SAME dump path
// and ports — a fresh process and run id — converges back to both keys
// with no operator action beyond restarting the process: the dump
// restored what it held and the REPL-07 resync healed the rest from the
// peer.
func TestStand_killNineRestart(t *testing.T) {
	t.Parallel()
	p, nodeA, nodeB := startStandPair(t)
	awaitServing(t, p.grpcA, servingDeadline)
	awaitServing(t, p.grpcB, servingDeadline)

	clientA := standClient(t, p.grpcA)
	clientB := standClient(t, p.grpcB)

	// Baseline convergence: one write on each node, both keys visible on
	// both endpoints through real gRPC.
	standSet(t, clientA, standKeyKillA)
	standSet(t, clientB, standKeyKillB)
	waitFor(t, standConvergeDeadline, "the baseline pair converged (both keys on both endpoints)", func() bool {
		return standServing(t, clientA, standKeyKillA) && standServing(t, clientA, standKeyKillB) &&
			standServing(t, clientB, standKeyKillA) && standServing(t, clientB, standKeyKillB)
	})

	// kill -9 node A: the hard death — no dump, no goodbye.
	killAndReap(t, nodeA)

	// Node B stays available through the peer's death: both keys readable,
	// its own endpoint unaffected.
	for _, key := range []string{standKeyKillA, standKeyKillB} {
		if value, found := standGet(t, clientB, key); !found || value != standValue(key) {
			t.Fatalf("node B after the peer's kill -9: Get(%q) = (%q, %t), want the pre-kill value", key, value, found)
		}
	}

	// Restart node A: same dump path, same ports — a fresh process with a
	// fresh run id (the REPL-07 path).
	nodeARestart := startStandNode(t, p, standNodeAID)
	awaitServing(t, p.grpcA, servingDeadline)

	// Convergence with no operator action: both pre-kill keys readable via
	// A's own endpoint — whatever the dump restored plus whatever the
	// resync healed from B, the union is the pre-kill state.
	waitFor(t, standConvergeDeadline, "the restarted node A serving both pre-kill keys", func() bool {
		return standServing(t, clientA, standKeyKillA) && standServing(t, clientA, standKeyKillB)
	})

	// Clean teardown of the survivors proves the restarted node is a full
	// participant, and no process left a panic behind.
	signalAndWaitExit(t, nodeARestart)
	signalAndWaitExit(t, nodeB)
	assertNoPanic(t, nodeA, "the kill -9 node A")
	assertNoPanic(t, nodeARestart, "the restarted node A")
	assertNoPanic(t, nodeB, "node B through the peer's death and restart")
}

// TestStand_bothNodesDown is the corrupted-dump-records scenario inside a
// both-nodes-down restart: both nodes exit cleanly through the D-19
// SIGTERM contract, ONE record of node A's dump is corrupted on disk
// while both are down, and the restart proves the two halves separately —
// A restarted alone serves every uncorrupted record from its own disk but
// NOT the corrupted one (the loader discarded it, and with no peer
// nothing can heal it), then B's restart heals the discarded record back
// onto A through replication alone. No operator action beyond restarting
// the processes.
func TestStand_bothNodesDown(t *testing.T) {
	t.Parallel()
	p, nodeA, nodeB := startStandPair(t)
	awaitServing(t, p.grpcA, servingDeadline)
	awaitServing(t, p.grpcB, servingDeadline)

	clientA := standClient(t, p.grpcA)
	clientB := standClient(t, p.grpcB)

	// Baseline: one local write per node, converged both ways.
	standSet(t, clientA, standKeyBothA)
	standSet(t, clientB, standKeyBothB)
	waitFor(t, standConvergeDeadline, "the baseline pair converged (both keys on both endpoints)", func() bool {
		return standServing(t, clientA, standKeyBothA) && standServing(t, clientA, standKeyBothB) &&
			standServing(t, clientB, standKeyBothA) && standServing(t, clientB, standKeyBothB)
	})

	// Both nodes down, cleanly: the D-19 exit on each (assertExitCode 0 is
	// signalAndWaitExit's contract).
	signalAndWaitExit(t, nodeA)
	signalAndWaitExit(t, nodeB)

	// While both are down, corrupt A's own local write in A's dump: one
	// checksum bit of one record, file otherwise valid.
	corruptDumpRecord(t, p.dumpA, standKeyBothA)

	// Restart A ALONE first: with B still down, nothing can heal the
	// discarded record, so A's own endpoint proves the load's two halves —
	// the corrupted record gone, every other record served from disk.
	nodeARestart := startStandLoggedNode(t, p, standNodeAID)
	awaitServing(t, p.grpcA, servingDeadline)
	waitFor(t, servingDeadline, "the discard line in restarted A's stderr", func() bool {
		return strings.Contains(nodeARestart.stderr.String(), standDiscardLine)
	})
	if _, found := standGet(t, clientA, standKeyBothA); found {
		t.Fatalf("the corrupted record %q survived the load — the discard path did not run", standKeyBothA)
	}
	if value, found := standGet(t, clientA, standKeyBothB); !found || value != standValue(standKeyBothB) {
		t.Fatalf("the uncorrupted record %q did not survive on disk: (%q, %t)", standKeyBothB, value, found)
	}

	// Bring B back: B's own dump kept the replicated copy of the discarded
	// record, and replication heals A — no operator action beyond the
	// restarts.
	nodeBRestart := startStandNode(t, p, standNodeBID)
	awaitServing(t, p.grpcB, servingDeadline)
	waitFor(t, standConvergeDeadline, "the corrupted record healed back from node B", func() bool {
		return standServing(t, clientA, standKeyBothA) && standServing(t, clientA, standKeyBothB) &&
			standServing(t, clientB, standKeyBothA) && standServing(t, clientB, standKeyBothB)
	})

	signalAndWaitExit(t, nodeARestart)
	signalAndWaitExit(t, nodeBRestart)
	assertNoPanic(t, nodeA, "the first-run node A")
	assertNoPanic(t, nodeB, "the first-run node B")
	assertNoPanic(t, nodeARestart, "the restarted node A through the corrupted dump")
	assertNoPanic(t, nodeBRestart, "the restarted node B")
}

// TestStand_sigtermFinalDump pins the D-19 final-dump contract's
// replicated half end to end: a write made on node B and replicated to
// node A is in A's FINAL confirmed dump after SIGTERM (the final Flush
// covers remote-merged dirtiness, not just local writes), the exit is 0
// (a clean exit is reserved for a confirmed dump), and a LONE restart of
// A — B stopped before the restart, so no peer interaction can supply
// the record — serves the replicated key purely from the final dump on
// disk. The exit-code negatives are already pinned by the cmd/sup-kv
// suites and are not duplicated here.
func TestStand_sigtermFinalDump(t *testing.T) {
	t.Parallel()
	p, nodeA, nodeB := startStandPair(t)
	awaitServing(t, p.grpcA, servingDeadline)
	awaitServing(t, p.grpcB, servingDeadline)

	clientA := standClient(t, p.grpcA)
	clientB := standClient(t, p.grpcB)

	// A replicated write on A: written on B, converged across.
	standSet(t, clientB, standKeySigterm)
	waitFor(t, standConvergeDeadline, "the replicated write visible on node A", func() bool {
		return standServing(t, clientA, standKeySigterm)
	})

	// SIGTERM node A: exit 0 only after the final confirmed dump.
	signalAndWaitExit(t, nodeA)

	// The final dump on disk contains the REPLICATED record — the
	// dump-grep precedent.
	dump := readFinalDump(t, p.dumpA)
	if !strings.Contains(dump, standKeySigterm) {
		t.Fatalf("the final dump at %s does not contain the replicated key %q:\n%s", p.dumpA, standKeySigterm, dump)
	}

	// Stop B before the restart: with the peer gone, the only possible
	// source for the record on a restarted A is the final dump itself.
	killAndReap(t, nodeB)

	nodeARestart := startStandNode(t, p, standNodeAID)
	awaitServing(t, p.grpcA, servingDeadline)
	if value, found := standGet(t, clientA, standKeySigterm); !found || value != standValue(standKeySigterm) {
		t.Fatalf("the lone restarted node A does not serve the replicated record from its final dump: (%q, %t)",
			value, found)
	}

	signalAndWaitExit(t, nodeARestart)
	assertNoPanic(t, nodeA, "the SIGTERM node A")
	assertNoPanic(t, nodeB, "the stopped node B")
	assertNoPanic(t, nodeARestart, "the lone restarted node A")
}

// TestStand_emptyDiskRecovery is the empty-disk scenario on the real
// binary: a live node A seeded with N keys, then node B started against
// it with a genuinely empty dump directory — B's whole state can only
// come from the live peer. Recovery — from B's process start to B serving
// all N keys through its own gRPC endpoint — must complete within the
// documented bound, derived in-test from the flags (never a magic
// constant): N records at worst ride N limiter-paced datagrams at
// floor(1s/rate) each, plus a margin for everything else the path spends.
// After recovery, B's own write replicates to A: the recovered node is a
// full participant, not a read-only mirror.
//
// Scale selection (plan 11): SUPKV_STAND_NIGHTLY=1 runs the review's
// documented F9 shape — 10 000 keys × 1 KiB with the reviewer's default
// flags (RESULTS.md's F9 row) — while the default keeps the
// shrunken-timer 20-key scale the PR gate runs, unchanged.
func TestStand_emptyDiskRecovery(t *testing.T) {
	t.Parallel()
	p := newStandPair(t)

	keys := emptyDiskKeys
	rate := emptyDiskRate
	hello, syncInterval := standHelloInterval, standSyncInterval
	var nodeExtras, nodeBExtras []string
	if standNightly {
		keys = emptyDiskKeysNightly
		rate = emptyDiskRateNightly
		hello, syncInterval = emptyDiskHelloNightly, emptyDiskSyncNightly
		// The documented default flags override the harness's shrunken
		// timer defaults on BOTH nodes (later flags win in main.go's
		// flag parsing), reproducing the reviewer's stand configuration.
		documented := []string{
			"--hello-interval=" + hello.String(),
			"--sync-interval=" + syncInterval.String(),
			"--sync-response-rate=" + strconv.Itoa(rate),
		}
		nodeExtras = documented
		nodeBExtras = documented
	} else {
		nodeBExtras = []string{"--sync-response-rate=" + strconv.Itoa(rate)}
	}

	// Node A first, alone: the seeded node.
	nodeA := startStandNode(t, p, standNodeAID, nodeExtras...)
	awaitServing(t, p.grpcA, servingDeadline)
	clientA := standClient(t, p.grpcA)

	keyList := make([]string, keys)
	for i := range keyList {
		keyList[i] = fmt.Sprintf(standEmptyKeyFmt, i)
	}
	seedEmptyDiskKeys(t, p.grpcA, keyList)

	// The recovery bound, derived from the flags the nodes run with: the
	// paced egress term is N * floor(1s/rate) — the limiter's own spacing
	// arithmetic (core.go's time.Second / SyncResponseRate) times the
	// record count. The margin covers, generously rather than precisely
	// (the documented bound is asserted, never wall-clock precision): B's
	// startup gate window (an empty disk selects the empty backstop, which
	// the harness derives from nodeArgs' standStartupGate, so both timers
	// share this one window; the gate opens on the peer's sync arrival or
	// the timeout, and B serves nothing before it opens), discovery plus one
	// listing/request/reply ladder round at
	// each timer (B's own requests ride B's limiter too), and
	// real-process scheduling slack on a loaded CI runner.
	paced := time.Duration(keys) * (time.Second / time.Duration(rate))
	margin := standStartupGate + 2*hello + 2*syncInterval + 2*time.Second
	deadline := paced + margin

	// Node B against the live peer, empty disk: its dump directory is its
	// own fresh temp dir — nothing to load, every key must arrive over
	// the wire.
	start := time.Now()
	nodeB := startStandNode(t, p, standNodeBID, nodeBExtras...)
	awaitServing(t, p.grpcB, servingDeadline)
	clientB := standClient(t, p.grpcB)

	waitFor(t, deadline, fmt.Sprintf("the empty-disk node B serving every one of the %d keys within the paced bound (%v + margin %v)", keys, paced, margin),
		emptyDiskRecovered(t, clientB, keyList))
	t.Logf("empty-disk recovery: %d keys in %v (paced bound %v, margin %v)", keys, time.Since(start), paced, margin)

	// The documented scale's values are the review's 1 KiB records: once
	// the listing proves presence, verify every record's value once (the
	// nightly List probe checks presence only, so a stale or truncated
	// record cannot hide).
	if standNightly {
		for _, key := range keyList {
			value, found := standGet(t, clientB, key)
			if !found || value != emptyDiskValue(key) {
				t.Fatalf("the recovered node B serves %q with a wrong record: found=%t value-bytes=%d want=%d",
					key, found, len(value), emptyDiskValueBytes)
			}
		}
	}

	// The recovered node participates in replication immediately: its own
	// write crosses to the seeded peer. The wait is derived from the flags
	// the nodes run with — the documented 30 s sync cadence at the nightly
	// scale (a new write is announced on the periodic round), the
	// shrunken-timer expression otherwise.
	converge := standConvergeDeadline
	if standNightly {
		converge = 2*syncInterval + 2*hello + 3*time.Second
	}
	standSet(t, clientB, standKeyFromB)
	waitFor(t, converge, "the recovered node's write visible on node A", func() bool {
		return standServing(t, clientA, standKeyFromB)
	})

	signalAndWaitExit(t, nodeA)
	signalAndWaitExit(t, nodeB)
	assertNoPanic(t, nodeA, "the seeded node A")
	assertNoPanic(t, nodeB, "the empty-disk recovered node B")
}

// emptyDiskValue is the scenario value for one key: the harness's derived
// value at the default scale and a deterministic 1 KiB value at the
// documented nightly scale (the review's F9 shape — every record carries
// a 1 KiB value). It stays a pure function of the key, so the nightly
// value pass can assert exact bytes rather than mere presence.
func emptyDiskValue(key string) string {
	if !standNightly {
		return standValue(key)
	}
	if len(key) >= emptyDiskValueBytes {
		return key[:emptyDiskValueBytes]
	}
	return key + strings.Repeat(".", emptyDiskValueBytes-len(key))
}

// seedEmptyDiskKeys writes the scenario keys through the real gRPC
// endpoint. The default scale keeps the direct sequential writes it
// always used; the nightly 10 000-key scale uses the review's preload
// profile — 64 workers round-robin over 8 connections (RESULTS.md §2's
// measured ~700 writes/s at the 10 000-key state) — because
// dump-confirmed sequential writes cost tens of milliseconds each at that
// scale. Every write must succeed: a failed seed invalidates the
// scenario, so the first error fails the test.
func seedEmptyDiskKeys(t *testing.T, addr string, keys []string) {
	t.Helper()
	if !standNightly {
		client := standClient(t, addr)
		for _, key := range keys {
			standSetWith(t, client, key, emptyDiskValue(key))
		}
		return
	}
	clients := make([]apiproto.SupKVServiceClient, emptyDiskSeedConns)
	for i := range clients {
		clients[i] = standClient(t, addr)
	}
	next := make(chan string, len(keys))
	for _, key := range keys {
		next <- key
	}
	close(next)
	var (
		wg       sync.WaitGroup
		errMu    sync.Mutex
		firstErr error
	)
	for w := range emptyDiskSeedWorkers {
		client := clients[w%len(clients)]
		wg.Go(func() {
			for key := range next {
				ctx, cancel := context.WithTimeout(t.Context(), emptyDiskSeedRPCTimeout)
				_, err := client.Set(ctx, &apiproto.SetRequest{Key: key, Value: []byte(emptyDiskValue(key))})
				cancel()
				if err != nil {
					errMu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("seed Set(%q): %w", key, err)
					}
					errMu.Unlock()
				}
			}
		})
	}
	wg.Wait()
	if firstErr != nil {
		t.Fatalf("seeding %d keys: %v", len(keys), firstErr)
	}
}

// emptyDiskRecovered returns the recovery poll's condition. At the
// default scale every key is polled through its serving value (the
// harness's standServing shape). At the nightly 10 000-key scale a single
// prefix List is the presence probe — 10 000 individual Gets per poll
// would drown the recovering node — throttled to emptyDiskPollEvery; the
// values are then verified once after the bound.
func emptyDiskRecovered(t *testing.T, client apiproto.SupKVServiceClient, keys []string) func() bool {
	t.Helper()
	if !standNightly {
		return func() bool {
			for _, key := range keys {
				if !standServing(t, client, key) {
					return false
				}
			}
			return true
		}
	}
	var last time.Time
	return func() bool {
		if now := time.Now(); now.Sub(last) < emptyDiskPollEvery {
			return false
		}
		last = time.Now()
		ctx, cancel := context.WithTimeout(t.Context(), rpcTimeout)
		defer cancel()
		resp, err := client.List(ctx, &apiproto.ListRequest{Prefix: standEmptyKeyPrefix})
		if err != nil {
			return false
		}
		return len(resp.GetKeys()) == len(keys)
	}
}

// TestStand_oversizeKeyF2c pins review finding F2c at process level against
// the fixed behavior (XPORT-03): node A Sets a 70 000-character key and the
// API rejects it locally with InvalidArgument, because the one key-length
// ceiling (storage.MaxKeyBytes) runs before the store. The original finding
// — Set OK locally, then every replication datagram refused by the kernel
// with EMSGSIZE forever — is fixed by the key bound rather than by kernel
// error handling; the scenario keeps its name for traceability. The key
// therefore never reaches node B: over the derived window (2×sync_interval
// plus the discovery, gate and slack margin) B's endpoint serves nothing
// under it, and neither node's final dump carries a record for it. The
// positive half proves the bound admits the documented maximum: a key of
// exactly storage.MaxKeyBytes bytes with a 61 440-byte value replicates to
// B and lands in B's final confirmed dump.
func TestStand_oversizeKeyF2c(t *testing.T) {
	t.Parallel()
	p := newStandPair(t)
	nodeA := startStandNode(t, p, standNodeAID)
	nodeB := startStandNode(t, p, standNodeBID)
	awaitServing(t, p.grpcA, servingDeadline)
	awaitServing(t, p.grpcB, servingDeadline)

	clientA := standClient(t, p.grpcA)
	clientB := standClient(t, p.grpcB)

	// The F2c key: standOversizeKeyLen characters of a repeating marker,
	// so the dump-grep assertions stay readable.
	oversizeKey := strings.Repeat("f2c-", standOversizeKeyLen/len("f2c-"))

	// The fixed half: the API rejects the key locally, before the store, so
	// there is no confirmed-but-unsendable record.
	ctx, cancel := context.WithTimeout(t.Context(), rpcTimeout)
	_, err := clientA.Set(ctx, &apiproto.SetRequest{Key: oversizeKey, Value: []byte("f")})
	cancel()
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Set with a %d-byte key: want InvalidArgument from the key bound, got %v", len(oversizeKey), err)
	}

	// The rejection is real on A: its own endpoint must not serve the key.
	if _, found := standGet(t, clientA, oversizeKey); found {
		t.Fatalf("node A serves the rejected %d-byte key", len(oversizeKey))
	}

	// The never-replicated half: the key must stay absent from node B for
	// the full window — 2×sync_interval plus the discovery, gate and slack
	// margin, derived from the flags the nodes run with. waitFor cannot
	// express "the absence holds for the whole window", so the poll's
	// condition is the elapsed window AND continued absence — an early
	// appearance fails the same waitFor loudly at its deadline instead of
	// passing silently.
	window := 2*standSyncInterval + standStartupGate + 2*standHelloInterval + 2*time.Second
	start := time.Now()
	waitFor(t, window+checkTimeout, "the rejected oversize key to stay absent from node B for the full window", func() bool {
		return time.Since(start) >= window && standAbsent(t, clientB, oversizeKey)
	})
	t.Logf("oversize key stayed unreplicated for the full %v window (2x sync_interval + margin)", window)

	// The positive half: the documented maximum — a 1 024-byte key and a
	// 61 440-byte value (the default limit) — passes the API and the
	// datagram ceiling, replicates to B, and survives B's final dump.
	boundaryKey := strings.Repeat("b", storage.MaxKeyBytes)
	boundaryValue := strings.Repeat("v", standBoundaryValueBytes)
	standSetWith(t, clientA, boundaryKey, boundaryValue)
	waitFor(t, standConvergeDeadline, "the 1 024-byte-key / 61 440-byte-value record on node B", func() bool {
		value, found := standGet(t, clientB, boundaryKey)
		return found && value == boundaryValue
	})

	// The disk-level truth after B's clean stop (the D-19 final dump): the
	// boundary record is there and the oversize key is not.
	signalAndWaitExit(t, nodeB)
	dumpB := readFinalDump(t, p.dumpB)
	if !strings.Contains(dumpB, boundaryKey) {
		t.Fatalf("node B's final dump at %s lacks the boundary key that replicated:\n%s", p.dumpB, dumpB)
	}
	if strings.Contains(dumpB, oversizeKey) {
		t.Fatalf("node B's final dump at %s carries the rejected oversize key:\n%s", p.dumpB, dumpB)
	}

	signalAndWaitExit(t, nodeA)
	assertNoPanic(t, nodeA, "the key-bound node A")
	assertNoPanic(t, nodeB, "the boundary-replicating node B")
}

// The startup-gate-empty stand proofs' knobs: the lone node's explicit pair
// (the empty gate far past the normal one), the closed-side probes, the
// derivation gate, and the two-empty-node backstop with its much shorter
// serving bound (a third of the backstop, so only the complete-list exchange
// can satisfy it).
const (
	standLoneNormalGate    = 200 * time.Millisecond
	standLoneEmptyGate     = 2 * time.Second
	standLoneClosedProbe   = 600 * time.Millisecond
	standDerivedGate       = 3 * time.Second
	standDerivedServeWait  = 5 * time.Second
	standFrozenClosedProbe = 2 * time.Second
	standTwoEmptyBackstop  = 30 * time.Second
)

// awaitHealthReachable waits until the endpoint's health service answers at all
// (a transport-level reply, whatever its status) — the closed-side probes'
// precondition, so their timed check never races the listener's startup.
func awaitHealthReachable(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	waitFor(t, timeout, "health answering on "+addr, func() bool {
		_, err := standHealthStatus(t, addr)
		return err == nil
	})
}

// TestStand_startupGateEmptyLoneNode pins the harness yield rule together with
// the lone empty node's wait: with an explicit --startup-gate-empty=2s in the
// extras the node is still NOT_SERVING at 600 ms — past the 200 ms normal gate
// — and then opens on the explicit empty gate. A harness that appended a
// derived empty gate after the extras would overwrite the 2 s with 200 ms and
// serve by the 600 ms probe (ADR 0006: the two parameters are separate, and a
// scenario's explicit empty gate always wins).
func TestStand_startupGateEmptyLoneNode(t *testing.T) {
	t.Parallel()
	p := newStandPair(t)
	started := time.Now()
	node := startStandNode(t, p, standNodeAID,
		"--startup-gate="+standLoneNormalGate.String(),
		"--startup-gate-empty="+standLoneEmptyGate.String(),
	)
	awaitHealthReachable(t, p.grpcA, servingDeadline)
	if elapsed := time.Since(started); elapsed < standLoneClosedProbe {
		time.Sleep(standLoneClosedProbe - elapsed)
	}
	got, err := standHealthStatus(t, p.grpcA)
	if err != nil || got != healthpb.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("the lone empty node's health at >= %v = (%v, err=%v), want NOT_SERVING: the explicit --startup-gate-empty=%v must govern, not the derived %v",
			standLoneClosedProbe, got, err, standLoneEmptyGate, standLoneNormalGate)
	}
	awaitServing(t, p.grpcA, standDerivedServeWait)
	signalAndWaitExit(t, node)
	assertNoPanic(t, node, "the lone empty-gate node")
}

// TestStand_startupGateEmptyDerivedFromExtras pins the harness rule that keeps
// the frozen R-03 test deterministic (RESEARCH Pitfall 1, T-4-13): the harness
// derives --startup-gate-empty from an explicit --startup-gate in the scenario
// extras, so the explicit normal gate governs BOTH timers. The first row is
// the executable proof — without the derivation the 60 s production default
// would keep the lone node closed past the 5 s wait; the second row pins the
// frozen 30 s shape closed at 2 s without waiting the timer out.
func TestStand_startupGateEmptyDerivedFromExtras(t *testing.T) {
	t.Parallel()

	t.Run("the explicit normal gate governs the derived empty backstop", func(t *testing.T) {
		t.Parallel()
		p := newStandPair(t)
		started := time.Now()
		node := startStandNode(t, p, standNodeAID, "--startup-gate="+standDerivedGate.String())
		// A lone node hears no peer, so only the derived backstop can open
		// the gate; if the harness did not derive it, the 60 s production
		// default governs and this wait fails.
		awaitServing(t, p.grpcA, standDerivedServeWait)
		if elapsed := time.Since(started); elapsed < standDerivedGate {
			t.Fatalf("the derived empty gate opened %v after launch, want at least the explicit --startup-gate=%v",
				elapsed, standDerivedGate)
		}
		signalAndWaitExit(t, node)
		assertNoPanic(t, node, "the derived-empty-gate node")
	})

	t.Run("the frozen 30 s shape stays closed at 2 s", func(t *testing.T) {
		t.Parallel()
		p := newStandPair(t)
		node := startStandNode(t, p, standNodeAID, "--startup-gate="+redR03StartupGate.String())
		awaitHealthReachable(t, p.grpcA, servingDeadline)
		time.Sleep(standFrozenClosedProbe)
		got, err := standHealthStatus(t, p.grpcA)
		if err != nil || got != healthpb.HealthCheckResponse_NOT_SERVING {
			t.Fatalf("the frozen %v shape's health after %v = (%v, err=%v), want NOT_SERVING (the R-03 closed-side window)",
				redR03StartupGate, standFrozenClosedProbe, got, err)
		}
		signalAndWaitExit(t, node)
		assertNoPanic(t, node, "the frozen-shape node")
	})
}

// TestStand_startupGateEmptyTwoEmptyNodes pins ADR 0006's first-start rule: two
// empty nodes exchange complete empty listings and both open well before the
// 30 s empty backstop — the 10 s serving bound is a third of the backstop, so a
// broken complete-listing path leaves both closed past the deadline instead of
// passing on the timeout. The explicit --startup-gate-empty in the extras also
// re-proves the harness yield rule on the pair's shared extras.
func TestStand_startupGateEmptyTwoEmptyNodes(t *testing.T) {
	t.Parallel()
	p, nodeA, nodeB := startStandPair(t, "--startup-gate-empty="+standTwoEmptyBackstop.String())
	started := time.Now()
	deadline := started.Add(servingDeadline)
	awaitServing(t, p.grpcA, time.Until(deadline))
	awaitServing(t, p.grpcB, time.Until(deadline))
	t.Logf("both empty nodes served %v after launch, well inside the %v empty backstop",
		time.Since(started), standTwoEmptyBackstop)
	signalAndWaitExit(t, nodeA)
	signalAndWaitExit(t, nodeB)
	assertNoPanic(t, nodeA, "the first empty node")
	assertNoPanic(t, nodeB, "the second empty node")
}
