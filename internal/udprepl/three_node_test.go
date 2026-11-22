package udprepl_test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/udprepl"
)

// smokeNodeC is the third node id of the N=3 broadcast-convergence test; the
// two-node smoke fixture names the other two.
const smokeNodeC = uint16(3)

// waitServing polls store every pollEvery until it serves key with exactly
// ver and want — the two-node smoke test's deadline and hard-deadline clamp,
// plus the last observed record on expiry, which is the state the N=3
// convergence verdict needs to name. It fails at the earlier of
// visibilityDeadline and the test's own deadline minus hardDeadlineMargin.
func waitServing(t *testing.T, store *storage.Store, key string, ver version.RecordVersion, want []byte, what string) {
	t.Helper()
	deadline := time.Now().Add(visibilityDeadline)
	if d, ok := t.Deadline(); ok {
		if hard := d.Add(-hardDeadlineMargin); hard.Before(deadline) {
			deadline = hard
		}
	}
	var observed string
	for time.Now().Before(deadline) {
		got, err := store.Get(key)
		switch {
		case err != nil:
			observed = fmt.Sprintf("record absent: %v", err)
		case got.Version != ver:
			observed = fmt.Sprintf("version %x, want %x", got.Version, ver)
		case !bytes.Equal(got.Data, want):
			observed = fmt.Sprintf("data %q, want %q", got.Data, want)
		default:
			return
		}
		time.Sleep(pollEvery)
	}
	t.Fatalf("timed out after %v waiting for %s: %s", visibilityDeadline, what, observed)
}

// TestSmoke_threeNodeBroadcastConvergence is PROTO-09's three-node
// real-socket proof: three stores, three cores and three adapters — the
// production adapter, codec, read loop and pacing, never a fake socket —
// share ONE wildcard loopback port through [udprepl.WithAddressReuse]. The
// pre-flight capability probe is the two-node smoke test's: a host without
// same-host broadcast fan-out skips loudly with the probed address and the
// platform reason, and never passes vacuously.
//
// A write originated by each adapter in turn becomes byte-equal visible in
// the other two stores over genuine broadcast datagrams, each node alone in
// learning its peers from broadcast Hellos (discovery counts two known peers
// per adapter). A same-key double write then pins LWW at N=3: node A writes
// the key, every node converges on it, node B writes the same key with a
// later store-stamped version, and every node converges on B's value.
//
// The assertions cover broadcast-delivered flows only. Unicast to a port
// shared by SO_REUSEADDR sockets is delivered to exactly ONE socket, the
// kernel's choice (research Pitfall 2), so unicast correctness at N=3 is not
// asserted here: the deterministic simulator (plans 04-02..04-05) and the
// two-port real-adapter integration harness cover it, and the adapter's doc
// comment records the same limitation.
func TestSmoke_threeNodeBroadcastConvergence(t *testing.T) {
	t.Parallel()

	port := freeUDPPort(t)
	loopback := &net.UDPAddr{IP: net.IPv4(127, 255, 255, 255), Port: port}
	broadcast := probeBroadcastAddr(t, port, loopback)

	bind := fmt.Sprintf("0.0.0.0:%d", port)
	cfg := repl.Config{Log: func(msg string) { t.Logf("%s", msg) }}
	ids := []uint16{smokeNodeA, smokeNodeB, smokeNodeC}
	stores := make([]*storage.Store, len(ids))
	adapters := make([]*udprepl.Adapter, len(ids))
	for i, id := range ids {
		stores[i] = storage.NewStore(id, clock.NewClock(time.Now, 0, 0, time.Minute), storage.Config{})
		adapter, err := udprepl.New(bind, broadcast.String(), id, smokeStore{store: stores[i]}, cfg, udprepl.WithAddressReuse())
		if err != nil {
			t.Fatalf("node %d adapter: %v", id, err)
		}
		adapters[i] = adapter
	}

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	for _, adapter := range adapters {
		go func() { _ = adapter.Run(ctx) }()
	}

	// Discovery: three nodes on one segment learn each other from broadcast
	// Hellos alone (REPL-01 — zero configuration beyond the broadcast
	// address); each adapter's own echo is discarded, so each sees two.
	waitFor(t, discoveryDeadline, "mutual discovery (each of the three known-peer counts == 2)", func() bool {
		for _, adapter := range adapters {
			if adapter.KnownPeers() != 2 {
				return false
			}
		}
		return true
	})

	// writeAndNotify is the production author sequence: write through the
	// store, read the stamped record back, then announce it through the
	// adapter's notification seam (track=false — this test observes stores,
	// not wait_for_replication).
	writeAndNotify := func(i int, key string, value []byte) version.RecordVersion {
		t.Helper()
		ver, _, err := stores[i].Set(key, value)
		if err != nil {
			t.Fatalf("node %d store Set(%q): %v", ids[i], key, err)
		}
		rec, err := stores[i].Get(key)
		if err != nil {
			t.Fatalf("node %d store Get(%q): %v", ids[i], key, err)
		}
		if rec.Version != ver {
			t.Fatalf("node %d store Get(%q) version %x, want the just-stamped %x", ids[i], key, rec.Version, ver)
		}
		err = adapters[i].NotifyLocalWrite(key, rec, false)
		if err != nil {
			t.Fatalf("node %d NotifyLocalWrite(%q): %v", ids[i], key, err)
		}
		return ver
	}

	// Every origin writes a key of its own; the broadcast DataReplication
	// must surface it byte-equal in both other stores.
	for origin := range ids {
		key := fmt.Sprintf("three/origin-%d", ids[origin])
		value := []byte(fmt.Sprintf("value-from-node-%d", ids[origin]))
		ver := writeAndNotify(origin, key, value)
		for peer := range stores {
			if peer == origin {
				continue
			}
			waitServing(t, stores[peer], key, ver, value,
				fmt.Sprintf("node %d holding node %d's %q", ids[peer], ids[origin], key))
		}
	}

	// Same-key LWW at N=3: node A's write converges everywhere, then node B's
	// later store-stamped write replaces it everywhere. The second write is
	// later by construction: B merged A's version before writing, so its own
	// clock stamps a strictly greater one.
	sharedKey := "three/shared"
	firstValue := []byte("first-writer")
	firstVer := writeAndNotify(0, sharedKey, firstValue)
	for peer := range stores {
		waitServing(t, stores[peer], sharedKey, firstVer, firstValue,
			fmt.Sprintf("node %d holding the first shared write", ids[peer]))
	}

	secondValue := []byte("second-writer")
	secondVer := writeAndNotify(1, sharedKey, secondValue)
	for peer := range stores {
		waitServing(t, stores[peer], sharedKey, secondVer, secondValue,
			fmt.Sprintf("node %d holding node %d's later shared write", ids[peer], ids[1]))
	}
}
