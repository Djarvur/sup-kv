package repl_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/port"
)

// Shared sync-test constants (the retransmit-test precedent, same test
// package): the egress limiter's per-response spacing at the normalized
// default rate — ⌊1s/500⌋ = 2ms, docs/IDEA.md §Parameters — and the key no
// test store ever holds, REPL-04's silence row.
const (
	responseSpacing = 2 * time.Millisecond
	keyUnknown      = "cfg/never-written"
)

// requestFrom builds a decoded DataRequest datagram from peerID (REPL-04:
// unicast "send me this key").
func requestFrom(peerID uint16, key string) port.InboundDatagram {
	return datagramFrom(peerID, port.DataRequest{Key: key})
}

// unicasts returns every unicast outbound message in drained in emission
// order — broadcast emissions (Hello, DataSync, retransmit
// DataReplication) are filtered out, leaving exactly the reactive unicast
// traffic the egress queue drains.
func unicasts(drained []port.Outbound) []port.Outbound {
	out := make([]port.Outbound, 0, len(drained))
	for _, ob := range drained {
		if !ob.Broadcast {
			out = append(out, ob)
		}
	}
	return out
}

// recordsEqual reports whether a and b are byte-equal in every record field
// — Version, Data, Deleted and Checksum — the REPL-04 exact-record reply
// contract.
func recordsEqual(a, b storage.Record) bool {
	return a.Version == b.Version && a.Deleted == b.Deleted &&
		a.Checksum == b.Checksum && bytes.Equal(a.Data, b.Data)
}

// TestDataRequest_replyAndSilence proves REPL-04 through the egress queue:
// a unicast DataRequest for a key the store holds — live or tombstoned, the
// delete-healing pin — is answered by one unicast DataReplication carrying
// the exact snapshot record; a DataRequest for an absent key enqueues and
// emits nothing, ever. The requester is never Hello'd first: a stranger's
// DataRequest is still protocol traffic from a live node, so the reply goes
// to the datagram's source handle regardless of peer-table membership. One
// Records() snapshot serves each handler call — the discipline the Task 2/3
// sync paths reuse.
func TestDataRequest_replyAndSilence(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"a known live key is answered with one unicast exact-record DataReplication", func(t *testing.T) {
			t.Helper()
			now := at(0)
			core, st := newTestCore()
			var out port.Outbox
			core.Advance(now, &out) // startup Hello (plan 04-01)
			out.Drain()
			_, _, err := st.store.Set(keyAlpha, []byte("live value"))
			if err != nil {
				t.Fatalf("store Set(%q): %v", keyAlpha, err)
			}
			want := st.Records()[keyAlpha]

			core.HandleDatagram(now, requestFrom(peerID1, keyAlpha), &out)
			if got := len(out.Drain()); got != 0 {
				t.Fatalf("the DataRequest handler emitted %d messages directly, want 0 — replies leave only through the egress queue", got)
			}
			if got := core.QueuedResponses(); got != 1 {
				t.Fatalf("after the DataRequest QueuedResponses() = %d, want 1", got)
			}
			core.Advance(now, &out)
			drained := unicasts(out.Drain())
			if len(drained) != 1 || drained[0].To != handleFor(peerID1) {
				t.Fatalf("the reply Advance drained %v, want exactly one unicast to the requester handle %q", drained, handleFor(peerID1))
			}
			reply, ok := drained[0].Msg.(port.DataReplication)
			if !ok {
				t.Fatalf("the reply is a %T, want port.DataReplication", drained[0].Msg)
			}
			if reply.Key != keyAlpha {
				t.Fatalf("reply key = %q, want %q", reply.Key, keyAlpha)
			}
			if !recordsEqual(reply.Record, want) {
				t.Fatalf("reply record = %+v, want the exact snapshot record %+v", reply.Record, want)
			}
			if got := core.QueuedResponses(); got != 0 {
				t.Fatalf("after the drain QueuedResponses() = %d, want 0", got)
			}
		}},
		{"a tombstoned key replies with the tombstone — deletes heal through reconciliation", func(t *testing.T) {
			t.Helper()
			now := at(0)
			core, st := newTestCore()
			var out port.Outbox
			core.Advance(now, &out)
			out.Drain()
			_, _, err := st.store.Set(keyBeta, []byte("doomed"))
			if err != nil {
				t.Fatalf("store Set(%q): %v", keyBeta, err)
			}
			_, _, err = st.store.Delete(keyBeta)
			if err != nil {
				t.Fatalf("store Delete(%q): %v", keyBeta, err)
			}
			want := st.Records()[keyBeta]
			if !want.Deleted {
				t.Fatalf("test setup: key %q is not tombstoned in the snapshot", keyBeta)
			}

			core.HandleDatagram(now, requestFrom(peerID1, keyBeta), &out)
			out.Drain()
			if got := core.QueuedResponses(); got != 1 {
				t.Fatalf("after the DataRequest for the tombstoned key QueuedResponses() = %d, want 1", got)
			}
			core.Advance(now, &out)
			drained := unicasts(out.Drain())
			if len(drained) != 1 || drained[0].To != handleFor(peerID1) {
				t.Fatalf("the tombstone reply Advance drained %v, want exactly one unicast to %q", drained, handleFor(peerID1))
			}
			reply, ok := drained[0].Msg.(port.DataReplication)
			if !ok {
				t.Fatalf("the tombstone reply is a %T, want port.DataReplication", drained[0].Msg)
			}
			if reply.Key != keyBeta || !recordsEqual(reply.Record, want) {
				t.Fatalf("tombstone reply = (%q, %+v), want the exact tombstone record (%q, %+v)", reply.Key, reply.Record, keyBeta, want)
			}
			if !reply.Record.Deleted {
				t.Fatal("reply record is live, want the tombstone (Deleted true)")
			}
		}},
		{"an unknown key stays silent — no enqueue, no emission, ever", func(t *testing.T) {
			t.Helper()
			now := at(0)
			core, _ := newTestCore()
			var out port.Outbox
			core.Advance(now, &out)
			out.Drain()

			// A second requester peer exercises the parameter both ways:
			// silence holds regardless of who asks.
			core.HandleDatagram(now, requestFrom(peerID2, keyUnknown), &out)
			if got := len(out.Drain()); got != 0 {
				t.Fatalf("the unknown-key DataRequest emitted %d messages directly, want 0", got)
			}
			if got := core.QueuedResponses(); got != 0 {
				t.Fatalf("a DataRequest for the unknown key %q enqueued %d responses, want 0", keyUnknown, got)
			}
			// Drive Advance well past a full rate window — five times the
			// 2ms per-response spacing: nothing may ever emit, no deferred
			// surprise (REPL-04's silence contract).
			for off := responseSpacing; off <= 5*responseSpacing; off += responseSpacing {
				core.Advance(now.Add(off), &out)
				if got := len(out.Drain()); got != 0 {
					t.Fatalf("Advance at t0+%v emitted %d messages after an unknown-key DataRequest, want 0", off, got)
				}
				if got := core.QueuedResponses(); got != 0 {
					t.Fatalf("QueuedResponses() = %d at t0+%v, want 0", got, off)
				}
			}
		}},
		{"two DataRequests in one event share the queue and drain in request order", func(t *testing.T) {
			t.Helper()
			now := at(0)
			core, st := newTestCore()
			var out port.Outbox
			core.Advance(now, &out)
			out.Drain()
			_, _, err := st.store.Set(keyAlpha, []byte("a"))
			if err != nil {
				t.Fatalf("store Set(%q): %v", keyAlpha, err)
			}
			_, _, err = st.store.Set(keyBeta, []byte("b"))
			if err != nil {
				t.Fatalf("store Set(%q): %v", keyBeta, err)
			}
			snap := st.Records()

			core.HandleDatagram(now, requestFrom(peerID1, keyBeta), &out)
			core.HandleDatagram(now, requestFrom(peerID1, keyAlpha), &out)
			out.Drain()
			if got := core.QueuedResponses(); got != 2 {
				t.Fatalf("after two DataRequests QueuedResponses() = %d, want 2", got)
			}

			// The first Advance drains the first request's reply immediately;
			// the second follows one spacing later. The limiter's exact
			// cadence is Task 3's assertion — here both replies must simply
			// emit, in request order, nothing more.
			core.Advance(now, &out)
			first := unicasts(out.Drain())
			core.Advance(now.Add(responseSpacing), &out)
			second := unicasts(out.Drain())
			if len(first) != 1 || len(second) != 1 {
				t.Fatalf("the two requests drained as %v then %v, want one unicast reply per Advance", first, second)
			}
			r1, ok := first[0].Msg.(port.DataReplication)
			if !ok {
				t.Fatalf("the first reply is a %T, want port.DataReplication", first[0].Msg)
			}
			r2, ok := second[0].Msg.(port.DataReplication)
			if !ok {
				t.Fatalf("the second reply is a %T, want port.DataReplication", second[0].Msg)
			}
			if r1.Key != keyBeta || !recordsEqual(r1.Record, snap[keyBeta]) {
				t.Fatalf("the first reply = (%q, %+v), want the exact record under %q", r1.Key, r1.Record, keyBeta)
			}
			if r2.Key != keyAlpha || !recordsEqual(r2.Record, snap[keyAlpha]) {
				t.Fatalf("the second reply = (%q, %+v), want the exact record under %q", r2.Key, r2.Record, keyAlpha)
			}
			if first[0].To != handleFor(peerID1) || second[0].To != handleFor(peerID1) {
				t.Fatalf("reply handles = %q, %q, want both unicast to %q", first[0].To, second[0].To, handleFor(peerID1))
			}
			if got := core.QueuedResponses(); got != 0 {
				t.Fatalf("after both drains QueuedResponses() = %d, want 0", got)
			}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			t.Parallel()
			tt.run(t)
		})
	}
}
