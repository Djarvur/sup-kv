# sup-kv review — replication core, UDP adapter, gRPC API, binary

Scope: `internal/app/repl/core.go`, `internal/udprepl/repl.go` (+ `supkv.proto`),
`internal/grpcapi/{server,gate,wait,watch}.go` (+ `supkvapi.proto`), `cmd/sup-kv/main.go`,
`internal/port/repl.go`. Reviewed at `d6b2a75` (master, clean tree). The repository was not
modified.

Reference documents: `docs/IDEA.md` and the accepted upstream spec
(`SPEC.md`, cited below as **SPEC §section**).

## How the findings were verified

All reproductions live in a scratch copy of the repo (no `.git`):

    evidence/repro/repl

| File (relative to the copy) | What it is |
|---|---|
| `internal/app/repl/review_repro_test.go` | 13 pure-core / simulator tests, package `repl_test`, reuse the existing helpers. Each asserts the required behaviour, so **FAIL = defect reproduced**. |
| `internal/reviewprobe/repl.go` | `internal/udprepl/repl.go` copied verbatim under another package name, so the production codec and socket setup can be exercised without running any `udprepl` test. |
| `internal/reviewprobe/*_test.go` | Codec, sizing, socket and persist probes. Sockets are **ephemeral loopback ports only** (`127.0.0.1:0`); files go to `t.TempDir()`. |

    cd <copy> && go test ./internal/app/repl/ -run 'TestReview_' -count=1 -v     # 13/13 FAIL
    cd <copy> && go test ./internal/reviewprobe/ -count=1 -v

No test under `cmd/sup-kv`, `internal/udprepl` or `internal/grpcapi` was run. Findings in those
packages are marked **Verified** only when the quoted code is unambiguous or when the probe
package reproduced them with the copied production code; otherwise **Suspected**, with what
would confirm them. Host for the socket probes: darwin/arm64, Go 1.27.4.

---

## Architecture assessment

**What holds up**

- The pure core is real. `repl.Core` takes `now` as a parameter, returns results through
  `port.Outbox`, owns no goroutine, lock, timer or socket, and emits in sorted order. The
  simulator (`internal/app/internal/simnet`) drives the *same* handlers the socket adapter
  drives; there is no "simulation mode". This is the strongest part of the code base.
- Cross-goroutine discipline in `udprepl` is clean: one event-loop goroutine mutates the core;
  everything else reads copy-on-write mirrors. `NotifyLocalWrite`'s select + re-check closes the
  exit race properly.
- The Watch hub's exact cut (snapshot taken on the hub goroutine, strictly-dominating versions
  only, disconnect instead of drop) is correct; I found no gap or duplicate path.
- The shutdown order (health → API ctx → GracefulStop with force path → replication → final
  flush → engine) loses no confirmed write, and signal ownership spans the whole shutdown.

**What does not**

1. **The implementation silently departs from the accepted spec in at least eight places**, and
   no ADR records them: the third `DataReplication` outcome (RPL-04), the listing for a
   first-seen peer (RPL-03), HLC `Update` on `DataSync` and the gate condition (RPL-02),
   `SyncID`/`ChunkIndex`/`ChunkCount` missing from the wire schema (RPL-02), create-if-absent on
   tombstones (API-02), snapshot-load deletion and replication (API-04), the dump-failure
   replication promise (API-01), the conflict journal (RPL-18). `v1.0-REQUIREMENTS.md` had
   already lost most of them, so "requirements complete" did not mean "spec implemented".
2. **"Determinism proven by simulation" proves less than the docs claim.** The simulator
   delivers decoded structs with no size limit, no socket buffer and no codec;
   `Cluster.Write` is a synchronous `Set`+`Notify` (never delayed by a dump, never out of
   order); restarts are instantaneous (never longer than `peer_timeout`); the gate lives in
   `grpcapi` and is not simulated at all. Every Critical/High finding below sits in exactly
   those blind spots.
3. **`StateStore` is the wrong shape.** Its only read is `Records()` — a deep copy of every
   value. The core calls it per `DataRequest`, per `DataSync` chunk and per listing (RPL-08),
   and it is the reason pending entries and egress entries carry full record copies.
4. **"Results, never callbacks" was taken one step too far.** The core has no output for
   "entry drained by Ack from X", so `udprepl` reconstructs it from snapshot diffs
   (`prevPending`, `lastAck`, `drained`, 250 ms retention) and `grpcapi/wait.go` polls that
   mirror every 5 ms. That is ~300 lines of the most intricate code in scope, it is approximate
   (API-03), and it costs O(pending) per event (RPL-15). A drain *result* in the Outbox would
   delete all of it.
5. **Proactive traffic is unpaced.** Only reactive unicasts pass the `sync_response_rate`
   limiter; listings, the discovery flush and retransmission rounds are emitted as one burst
   per event-loop iteration into sockets whose buffers are never sized (RPL-01, RPL-07, RPL-10).
6. **File factoring.** `core.go` is 513 code lines + 499 comment lines; the code is cohesive,
   but half the file narrates planning history (`D-15 P4-WR-02`, `probe #9`, `T-05-04`) and
   several comments are stale or wrong (INF-01). `repl.go` mixes socket, event loop, mirror and
   codec — split `codec.go` / `mirror.go`. `main.go` (554 code lines) holds config parsing,
   wiring and lifecycle although `AGENTS.md` says "`main()` holds no testable logic", and it
   validates almost nothing (BIN-01).
7. **Layering is enforced only for imports.** `internal/app/persist` runs its own goroutine,
   real timers and `time.Now()`, so the dump↔notify interplay (RPL-09, API-01) is outside the
   simulated core. `grpcapi` imports the `persistfile` adapter for one sentinel and the concrete
   `*persist.Engine`; `port.repl.go` is a clean seam.

---

## Findings

Severity: Critical = confirmed-write loss / permanent divergence / crash / hang; then High,
Medium, Low, Info.

### Critical

---

#### RPL-01 — The anti-entropy listing is one unpaced burst; the tail of the keyspace is never reconciled

- **Severity:** Critical (permanent divergence at the documented scale)
- **Status:** Verified with the copied production adapter on darwin, receiver `SO_RCVBUF` forced
  to the Linux default (212 992 B). Confirmation on a Linux host is still required (expected
  worse: Linux charges ~2.3 KB of `truesize` per 1.4 KB datagram).
- **Where:** `internal/app/repl/core.go:751-759` (and the unicast twin at `:490-493`),
  `internal/app/repl/core.go:633`, `internal/udprepl/repl.go:270-291`

```go
// core.go:755
c.nextSync = now.Add(c.cfg.SyncInterval)
for _, chunk := range c.listingChunks() {
    out.Broadcast(port.DataSync{Entries: chunk})   // every chunk, same event-loop iteration
}
// core.go:633 — receiver, once per chunk
snapshot := c.store.Records()                      // deep copy of the whole store
```

- **Failure scenario.** 10 000 keys × 1 KiB. The listing is 304 chunks ≈ 444 KB, sent in a
  tight loop. The receiver needs ~2.2 ms per chunk (RPL-08), its inbound channel holds 64
  datagrams and the socket buffer (never sized, RPL-07) about 213 KB. Everything that arrives
  after the buffer fills is dropped — always the same, highest-sorted chunks, every round,
  because order and timing are deterministic. A key in that tail that missed direct replication
  (writer restarted before the Ack, dump-failed write, snapshot load, RPL-09) is **never**
  repaired: the sender logs no error, the receiver counts no discard.
  Measured: 231 of 304 chunks processed per round; the tail key is still missing after 5 full
  rounds in 4 of 4 runs; sender-side send errors 0. With the deep copy removed (control) the
  burst still loses tail chunks in 13 of 24 runs, so pacing and buffer sizing are needed too.
  SPEC §DataSync says a lost chunk "only means its keys are reconciled next time" — that
  assumes loss is not systematic.
- **Fix.** (1) Pace proactive listings (token bucket, or spread chunks over a fraction of
  `sync_interval`); (2) rotate the starting chunk per round so residual loss cannot starve the
  same keys; (3) `SetReadBuffer`/`SetWriteBuffer` at bind and log the effective sizes (RPL-07);
  (4) give `StateStore` a cheap key→version view (RPL-08).
- **Acceptance.** Two real-socket nodes, receiver `SO_RCVBUF` = 212 992 B, 20 000 keys × 1 KiB,
  receiver missing one key in each of the first, middle and last chunk: all three are present
  after at most 2 listing rounds, in 20 of 20 runs.

```go
// internal/reviewprobe/synctail2_probe_test.go (copied production adapter, ephemeral ports)
func TestProbe_antiEntropyTailStarvation(t *testing.T) {
    const keys = 10_000
    missing := map[int]bool{5: true, 5000: true, 9995: true}
    key := func(i int) string { return fmt.Sprintf("svc/v1/key-%05d", i) }
    mk := func(id uint16, skip map[int]bool) *storage.Store {
        st := storage.NewStore(id, clock.NewClock(time.Now, 0, 0, time.Minute), storage.Config{})
        val := make([]byte, 1024)
        for i := range keys {
            if skip[i] { continue }
            _, _, _ = st.Merge(key(i), storage.Record{Version: version.PackVersion(1730000000000, 0, 9), Data: val})
        }
        return st
    }
    rStore, sStore := mk(2, missing), mk(1, nil)
    sink, _ := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
    defer sink.Close()
    r, _ := New("127.0.0.1:0", sink.LocalAddr().String(), 2, memStore{rStore}, repl.Config{SyncInterval: time.Hour})
    _ = r.conn.SetReadBuffer(212992) // Linux rmem_default
    s, _ := New("127.0.0.1:0", r.conn.LocalAddr().String(), 1, memStore{sStore}, repl.Config{SyncInterval: time.Second})
    ctx, cancel := context.WithCancel(context.Background()); defer cancel()
    go func() { _ = r.Run(ctx) }()
    time.Sleep(200 * time.Millisecond)
    go func() { _ = s.Run(ctx) }()
    time.Sleep(4700 * time.Millisecond) // startup listing + 4 timer rounds
    for _, i := range []int{5, 5000, 9995} {
        if _, err := rStore.Get(key(i)); err != nil {
            t.Errorf("key #%d is still missing on the receiver after %d anti-entropy rounds", i, s.SentByType().Sync/304)
        }
    }
}
// Output: key #5 repaired=true, key #5000 repaired=true, key #9995 repaired=false (4/4 runs)
```

---

#### RPL-02 — The startup gate opens on the first DataSync datagram, before anything was synced; DataSync never advances the HLC

- **Severity:** Critical (a confirmed peer write is overwritten; a confirmed local write is
  discarded)
- **Status:** Verified (simulator + unit repro)
- **Where:** `internal/app/repl/core.go:631-644`, `internal/grpcapi/gate.go:138-142`,
  `internal/udprepl/replproto/supkv/v1/supkv.proto:66-68`

```go
// core.go:631
func (c *Core) handleDataSync(now time.Time, source string, entries []port.SyncEntry) {
    c.syncArrived = true              // any chunk, any sender, nothing processed yet
    snapshot := c.store.Records()     // ...and no clock Update anywhere in this handler
// gate.go:139
if s.opts.SyncArrived != nil && s.opts.SyncArrived() { s.openGate(); return }
// supkv.proto:66 — no SyncID / ChunkIndex / ChunkCount
message DataSync { repeated SyncEntry entries = 1; }
```

- **Spec.** §Startup: writes are rejected "until the first full sync from the neighbour **has
  been processed** or `startup_gate` elapsed"; it "closes the window in which the node has not
  yet seen versions written by the neighbour while it was down". §Recovery: "on receiving a
  chunk the server calls `Update` with the chunk's maximum version". §DataSync carries
  `SyncID`, `ChunkIndex`, `ChunkCount`. None of the three is implemented; the code comment
  calls this "the A1 reading".
- **Failure scenario A (create-if-absent — the spec's own use case).** Rolling upgrade. Node B
  holds 300 keys, including one the operator changed. Node A restarts with an empty or older
  disk. B's listing arrives; the gate on A opens 22 ms after restart, when A holds 10 of 300
  records. A's service seeds build defaults with `create_if_absent`; the key is still absent on
  A, so the write succeeds with a fresh version and **replaces the operator's confirmed value
  on both nodes**. A gate that waited for the sync (0.6 s at 500 responses/s, inside the 1 s
  `startup_gate`) would have answered `AlreadyExists`.
- **Failure scenario B (clock skew).** B's clock is 30 s ahead (inside `drift_warn`). A's first
  received chunk announces a version 30 s in A's future; the gate opens; a client `Set` on A is
  stamped with A's wall clock, confirmed on disk, and discarded by LWW when the requested
  record arrives.
- **Fix.** Add `sync_id`, `chunk_index`, `chunk_count` to `DataSync` (new `proto_version`).
  Open the gate when one complete round from one peer has been received **and** the responses
  it generated have been resolved (or `startup_gate` elapsed). Add a version-observation method
  to `StateStore` and call it once per chunk with the chunk's maximum version.
- **Acceptance.** `TestReview_gateOpensBeforeSyncIsProcessed` and
  `TestReview_dataSyncDoesNotAdvanceHLC` pass; in the first, `SetIfAbsent` on any peer-held key
  returns `ErrAlreadyExists` at every instant after the gate signal, while `startup_gate` has
  not elapsed.

```go
func TestReview_gateOpensBeforeSyncIsProcessed(t *testing.T) {
    sched := simnet.NewScheduler(time.UnixMilli(testNowMs))
    cluster := simnet.NewCluster(sched, simRNG(1), 2, repl.Config{}, simFixedDelay(nil))
    quiesceDiscovery(t, sched, cluster)
    const keys = 300
    for i := range keys { cluster.Write(1, fmt.Sprintf("cfg/%04d", i), []byte("user-value")) }
    if !scnQuiesce(sched, cluster) { t.Fatal("setup did not quiesce") }
    cluster.ReplaceNode(0, scnRunID(1, 7), 0)             // node 0 restarts empty, within peer_timeout
    restartAt := sched.Now()
    for !cluster.Core(0).SyncArrived() { sched.RunUntil(sched.Now().Add(time.Millisecond)) }
    sched.RunUntil(sched.Now().Add(20 * time.Millisecond)) // grpcapi gate poll period
    have := len(cluster.Store(0).Records())
    t.Logf("gate open %v after restart; restarted node holds %d of %d peer records", sched.Now().Sub(restartAt), have, keys)
    lastKey := fmt.Sprintf("cfg/%04d", keys-1)
    _, _, err := cluster.Store(0).SetIfAbsent(lastKey, []byte("build-default"))
    if err == nil { // grpcapi.Set's author sequence
        var out port.Outbox
        cluster.Core(0).NotifyLocalWrite(sched.Now(), lastKey, cluster.Store(0).Records()[lastKey], &out)
        for _, ob := range out.Drain() { if ob.Broadcast { cluster.Network().Broadcast(cluster.Handle(0), ob.Msg) } }
    }
    scnQuiesce(sched, cluster)
    peerVal, _ := cluster.Store(1).Get(lastKey)
    if err == nil || string(peerVal.Data) != "user-value" {
        t.Fatalf("create-if-absent right after the gate opened: err=%v, peer now holds %q", err, peerVal.Data)
    }
}
// Output: gate open 22ms after restart; restarted node holds 10 of 300 peer records
//         create-if-absent right after the gate opened: err=<nil>, peer now holds "build-default"

func TestReview_dataSyncDoesNotAdvanceHLC(t *testing.T) {
    core, st := newTestCore() // local wall clock parked at testNowMs
    var out port.Outbox
    core.Advance(at(0), &out); out.Drain()
    peerVer := version.PackVersion(testNowMs+30_000, 0, peerID1)
    core.HandleDatagram(at(0), syncFrom(peerID1, port.SyncEntry{Key: keyAlpha, Version: peerVer}), &out)
    localVer, _, _ := st.store.Set(keyAlpha, []byte("confirmed-after-gate"))
    if localVer.Compare(peerVer) <= 0 {
        t.Fatalf("write accepted after the first sync chunk got version %x, BELOW the announced %x", localVer[:8], peerVer[:8])
    }
}
```

### High

---

#### RPL-03 — A first-seen or returning peer gets no listing; restart detection works only inside `peer_timeout`

- **Severity:** High
- **Status:** Verified (unit repro)
- **Where:** `internal/app/repl/core.go:483-498`

```go
discovery := len(c.peers) == 0
restarted := false
if known, ok := c.peers[in.NodeID]; ok && known.runID != in.RunID { restarted = true }
...
if restarted { /* unicast full listing */ return }
if !discovery { return }                // unknown node: learned, nothing sent
```

- **Spec.** §Hello: "if the neighbour's `run_id` differs from the remembered one **or the
  neighbour is seen for the first time**, the node … immediately sends it the full sync".
- **Failure scenario.** `peer_timeout` is 5 s; an ordinary reboot takes longer. B has evicted
  A, so A's first Hello is "unknown node": B sends nothing. A's own startup listing only
  reveals keys A *has*; keys created on B during the downtime are invisible until B's next
  timer round, up to `sync_interval` = 30 s. A's gate opens on the 1 s backstop with no sync at
  all, which widens RPL-02 scenario A to every key created while A was down. The spec's
  failure table promises "neighbour sends the full sync on the new RunID … recovery: seconds".
  The same gap delays convergence after every healed partition longer than `peer_timeout`.
  The comment's rationale ("the peer's own startup DataSync and the periodic rounds reconcile
  it") is wrong for keys the peer lacks.
- **Fix.** Send the targeted listing for every Hello that creates a peer row, paced per RPL-01
  and rate-limited per node_id per RPL-12.
- **Acceptance.** `TestReview_restartAfterEvictionGetsNoListing` passes; a simulator scenario
  with a node down for `2 × peer_timeout` and 300 peer-only keys converges within 2 s of the
  node's first Hello instead of at the next `sync_interval` round.

```go
func TestReview_restartAfterEvictionGetsNoListing(t *testing.T) {
    core, st := newTestCore()
    seedGraduatedKeys(t, st)
    var out port.Outbox
    core.Advance(at(0), &out)
    core.HandleDatagram(at(0), helloFromRun(runIDFor(1)), &out); out.Drain()
    core.Advance(at(6*time.Second), &out); out.Drain()             // down > peer_timeout: evicted
    core.HandleDatagram(at(6*time.Second), helloFromRun(runIDFor(2)), &out) // back as a new process
    if len(targetedSyncs(t, out.Drain(), handleFor(peerID1))) == 0 {
        t.Fatal("a peer restarted after 6s of downtime received no targeted DataSync listing")
    }
}
```

---

#### RPL-04 — A stale DataReplication is Acked instead of answered: the spec's third outcome is missing

- **Severity:** High
- **Status:** Verified (unit + simulator repro)
- **Where:** `internal/app/repl/core.go:523-543`

```go
conflicts, drift, err := c.store.ApplyRemote(msg.Key, msg.Record)
if err != nil { ...; return }
...
out.Unicast(source, port.Ack{Key: msg.Key, Version: msg.Record.Version})  // newer, equal AND older
```

- **Spec.** §Replication: "received version is lower than the local one: the sender gets a
  unicast `DataReplication` with the local record. **`Ack` is not sent**." §Ack: sent "if the
  received version was applied or already equals the local one".
- **Failure scenario.** B holds a newer version of `k` that A never received and that B no
  longer tracks (B restarted, RPL-09, API-01, API-04). A writes `k` with a lower version. B
  rejects it and Acks. A's pending entry drains, `wait_for_replication` reports `acked=true`
  for a write the peer threw away, and A keeps serving the losing value until the next sync
  round (up to 30 s, or never under RPL-01). The spec bound is one RTT.
- **Fix.** Make `ApplyRemote`'s result distinguish applied / equal / stale (the `Conflict`
  winner already tells). On stale: enqueue a unicast `DataReplication` with the local record on
  the rate-limited egress queue, send no Ack.
- **Acceptance.** Both tests below pass; a simulator scenario with N=4 shows exactly N−1
  unicast replies and zero Acks for one stale broadcast.

```go
func TestReview_staleDataReplicationIsAckedInsteadOfAnswered(t *testing.T) {
    core, st := newTestCore()
    var out port.Outbox
    core.Advance(at(0), &out)
    core.HandleDatagram(at(0), helloFrom(peerID1), &out); out.Drain()
    newer := remoteRecord(testNowMs+10_000, peerID2, "newer")
    _, _, _ = st.store.Merge(keyAlpha, newer)
    older := remoteRecord(testNowMs+1_000, peerID1, "older")
    core.HandleDatagram(at(0), replFrom(peerID1, keyAlpha, older), &out)
    var acks, replies int
    for _, ob := range out.Drain() {
        switch m := ob.Msg.(type) {
        case port.Ack: acks++
        case port.DataReplication:
            if !ob.Broadcast && ob.To == handleFor(peerID1) && m.Record.Version == newer.Version { replies++ }
        }
    }
    if acks != 0 || replies != 1 { t.Fatalf("got %d Ack(s) and %d newer-record replies; spec: 0 and 1", acks, replies) }
}
// Output: got 1 Ack(s) and 0 unicast newer-record replies
// TestReview_staleWriterDrainsOnRejectingPeersAck (simulator): 5 RTTs later node 0 reads
// "A-older", node 1 holds "B-newer", node 0's pending set is empty.
```

---

#### RPL-05 — Received versions are not bounded: one datagram exhausts the version domain, permanently and invisibly

- **Severity:** High (impact is permanent divergence plus silent loss of every later write; it
  needs one CRC-valid malformed datagram — a peer bug, version skew, or a CRC32 collision)
- **Status:** Verified (unit repro + codec probe)
- **Where:** `internal/udprepl/repl.go:843-850` (length is the only check),
  `internal/app/repl/core.go:523-524`; effect in `internal/app/clock/clock.go:119-149`

```go
func recordVersion(b []byte) (version.RecordVersion, bool) {
    var ver version.RecordVersion
    if len(b) != len(ver) { return version.RecordVersion{}, false }
    copy(ver[:], b); return ver, true          // any 16 bytes are a version
}
```

- **Failure scenario.** A `DataReplication` with version `ff…ff` is applied and Acked. `Update`
  pins the HLC at `(2^48−1, 4095)`; the very next local write is stamped year ~10889, counter
  4095, and so is every write after it. Two successive writes to one key carry **the same
  version**; peers treat the second as "equal — no-op" and keep the old value; `DataSync`
  compares versions and sees nothing. The poison record replicates like any other, so every
  node is infected within one sync round, and the dump watermark keeps it across restarts.
  Recovery needs hand-editing every dump with the whole cluster down. With a timestamp of
  `2^48−1` and counter 0 the same end state is reached after 4096 clock events.
  The drift signal does not fire for the poison record itself (computed before the update).
- **Fix.** Validate at the adapter or core, before `ApplyRemote` and before any clock update:
  version nibble 7, variant `10`, and `l ≤ now + max_future` (configurable, hours to days).
  Discard, count under a new class, log at Warn. Apply the same check to `SyncEntry` versions.
- **Acceptance.** `TestReview_farFutureVersionExhaustsVersionDomain` passes (record rejected,
  later versions distinct); a new discard class is asserted in `TestDiscard_classes`.

```go
func TestReview_farFutureVersionExhaustsVersionDomain(t *testing.T) {
    wall := func() time.Time { return time.UnixMilli(testNowMs) }
    mk := func(id uint16) (*repl.Core, *stateStore) {
        st := &stateStore{store: storage.NewStore(id, clock.NewClock(wall, 0, 0, testDriftWarn), storage.Config{})}
        return repl.New(id, testRunID(id), handleFor(id), st, repl.Config{}), st
    }
    coreA, stA := mk(localID); _, stB := mk(peerID1)
    var out port.Outbox
    coreA.Advance(at(0), &out); out.Drain()
    poison := storage.Record{Version: version.PackVersion(int64(1)<<48-1, 0, strangerID), Data: []byte("x")}
    coreA.HandleDatagram(at(0), replFrom(strangerID, "poison", poison), &out); out.Drain()
    for i := range 5000 { _, _, _ = stA.store.Set(fmt.Sprintf("burn/%d", i%7), []byte("b")) }
    vOld, _, _ := stA.store.Set(keyAlpha, []byte("old")); recOld := stA.store.Records()[keyAlpha]
    vNew, _, _ := stA.store.Set(keyAlpha, []byte("new")); recNew := stA.store.Records()[keyAlpha]
    _, _, _ = stB.store.Merge(keyAlpha, recOld); _, _, _ = stB.store.Merge(keyAlpha, recNew)
    gotB, _ := stB.store.Get(keyAlpha)
    if vOld == vNew || string(gotB.Data) != "new" {
        t.Fatalf("two successive writes share version %x; A holds %q, B holds %q", vNew[:8], "new", gotB.Data)
    }
}
// Output: two successive writes share version ffffffffffff7fff (equal=true); node A holds "new", node B holds "old"
```

---

#### RPL-06 — Key length and `value_max_bytes` are unbounded: a legal, confirmed write can be impossible to replicate

- **Severity:** High (permanent non-replication of a confirmed record; needs keys ≥ 4 KB with
  large values, or a raised `value_max_bytes`)
- **Status:** Verified (sizes measured with the production codec)
- **Where:** `internal/grpcapi/server.go:135-137`, `cmd/sup-kv/main.go:311-312`,
  `internal/app/repl/core.go:780-815`, `internal/udprepl/repl.go:555-560`

```go
// server.go:135 — the only key validation in the system
if req.GetKey() == "" { return nil, status.Error(codes.InvalidArgument, "key must not be empty") }
// repl.go:556 — what happens to the datagram
_, err := a.conn.WriteToUDP(datagram, addr)
if err != nil { a.log(fmt.Sprintf("udprepl: send to %s: %v", addr, err)) }
```

- **Measured.** `DataReplication` overhead is 68 B. With a 61 440 B value the largest key that
  fits one datagram is **3 998 B**; `Set(key=4096 B, value=61440 B)` yields 65 605 B > 65 507.
  With a 66 000 B key, Ack, DataRequest, DataSync and DataReplication are all unsendable.
  `--value-max-bytes=99999999` is accepted without complaint.
- **Failure scenario.** The client gets `OK` (the dump succeeded). Every broadcast, every
  retransmission (forever, each `ack_timeout_max`) and every `DataRequest` reply fails with
  `EMSGSIZE`. Peers see the key in every listing, request it, and never receive it. The only
  signals are a Warn line per attempt and a stuck unacked gauge. If the node dies, the write is
  gone. The same key sizes break the spec rule that every non-`DataReplication` datagram fits
  `sync_chunk_bytes`: Ack(key=1400 B) is 1 457 B.
- **Fix.** Enforce a key limit at the API (`InvalidArgument`) and on receive, e.g. 1 024 B;
  reject at startup any `value_max_bytes` above `65507 − 68 − key_max`; refuse at
  `NotifyLocalWrite`/encode time what cannot be sent and surface it as a metric.
- **Acceptance.** `Set` with a key over the limit returns `InvalidArgument`; a table test proves
  `len(encodeDatagram(DataReplication{max key, max value})) ≤ 65507` and every other message
  with a max key ≤ `sync_chunk_bytes`; startup with `--value-max-bytes=70000` exits 2.

```go
// internal/reviewprobe/probe_test.go
over := len(enc(t, port.DataReplication{Key: strings.Repeat("k", 4096), Record: rec(61440)}))
if over > 65507 { t.Errorf("a legal Set (key 4096 B, value at value_max_bytes) produces a %d B DataReplication", over) }
// Output: 65605 B; largest key that still fits with a 61440 B value = 3998 B
```

---

#### RPL-07 — Socket buffers are never sized: on darwin no record above ~9.1 KiB ever leaves the node

- **Severity:** High on darwin (a release target: `goos: [linux, darwin]`); Medium on Linux
  (burst loss, and a precondition of RPL-01)
- **Status:** Verified on darwin with the copied production `New()`; Linux part Suspected
- **Where:** `internal/udprepl/repl.go:270-291` — `SO_REUSEADDR` and `SO_BROADCAST` are set,
  `SO_SNDBUF`/`SO_RCVBUF` are not; `SetReadBuffer`/`SetWriteBuffer` appear nowhere in the repo.
- **Failure scenario.** darwin's default UDP send limit is `net.inet.udp.maxdgram` = 9 216 B.
  A `Set` with a 20 KiB value is confirmed; its `DataReplication` fails with `sendto: message
  too long` on every attempt, including `DataRequest` replies, so the record never reaches a
  peer. `supkv_replication_datagrams_sent_total{type=data}` still counts it (RPL-18). On Linux
  the default 212 992 B receive buffer holds about three maximum-size datagrams.
- **Fix.** `SetWriteBuffer` (≥ 1 MiB) and `SetReadBuffer` (≥ 4 MiB) in `New`; read the
  effective values back and log them; fail startup if the send buffer is below the maximum
  datagram size.
- **Acceptance.** The probe below passes on darwin and Linux; startup logs both effective sizes.

```go
// internal/reviewprobe/socket_probe_test.go — production New()/Run()/NotifyLocalWrite, ephemeral ports
for key, size := range map[string]int{"small-1KiB": 1 << 10, "mid-20KiB": 20 << 10, "max-60KiB": 61440} {
    _ = a.NotifyLocalWrite(key, storage.Record{Version: ver, Data: make([]byte, size)})
}
// Output (GOOS=darwin): received DataReplication: map[small-1KiB:1024]
//   adapter log: udprepl: send to 127.0.0.1:56199: ... sendto: message too long   (x2)
//   only 1 of 3 legal records left the production socket; sent-by-type(data)=3
```

---

#### RPL-08 — `StateStore.Records()` deep-copies the whole store for every DataRequest, every DataSync chunk and every listing

- **Severity:** High (the single event loop stalls at the documented scale; feeds RPL-01)
- **Status:** Verified (measured)
- **Where:** `internal/app/repl/core.go:606`, `:633`, `:772`; interface at `:178-187`

```go
rec, ok := c.store.Records()[key]     // :606 — one record wanted
snapshot := c.store.Records()         // :633 — ~33 versions wanted
snapshot := c.store.Records()         // :772 — key→version wanted
```

- **Measured** (10 000 keys × 1 KiB): one `DataRequest` allocates **10.9 MiB** and takes
  2.3 ms; one `DataSync` chunk the same. Empty-node recovery is 10 000 requests ≈ 106 GiB of
  copying, at 500 requests/s that is more than one core; one sync round is ~304 chunks ≈ 0.7 s
  of loop time per peer. At 10 000 × 60 KiB each datagram copies 600 MB. During the copy the
  store's `RLock` is held, so writers and the replication loop itself stall; Hellos queue
  behind it and peers time out.
- **Fix.** Replace `Records()` in `StateStore` with `Lookup(key) (Record, bool)` (tombstones
  included) and `Versions() map[string]RecordVersion` (or a sorted iterator). Let retransmission
  and egress re-read the record by key at send time instead of holding copies (this also halves
  pending-set memory while a peer is down, and SPEC §Retransmission describes the set as
  key-version pairs).
- **Acceptance.** `TestReview_recordsDeepCopyPerDatagram` passes (< 64 KiB allocated per
  datagram on a 10 MiB store); a benchmark for `handleDataRequest` is O(record), not O(store).

```go
func TestReview_recordsDeepCopyPerDatagram(t *testing.T) {
    const keys, valueSize = 10_000, 1024
    core, st := newTestCore()
    val := make([]byte, valueSize)
    for i := range keys { _, _, _ = st.store.Set(fmt.Sprintf("svc/v1/key-%05d", i), val) }
    var out port.Outbox
    core.Advance(at(0), &out); out.Drain()
    var before, after runtime.MemStats
    runtime.GC(); runtime.ReadMemStats(&before)
    for i := range 50 { core.HandleDatagram(at(0), requestFrom(peerID1, fmt.Sprintf("svc/v1/key-%05d", i)), &out) }
    runtime.ReadMemStats(&after)
    if per := (after.TotalAlloc - before.TotalAlloc) / 50; per > 64<<10 {
        t.Fatalf("one DataRequest allocates %d B on a %d-key store", per, keys)
    }
}
// Output: DataRequest (1 record) 10.89 MiB allocated, 2.27ms per datagram
```

---

#### API-01 — A write whose dump failed is never handed to replication

- **Severity:** High (spec promise broken exactly when the node is most likely to die)
- **Status:** Verified (unambiguous from code)
- **Where:** `internal/grpcapi/server.go:142-150` (same shape in `Delete`, `:178-182`)

```go
ver, _, err = s.engine.Set(req.GetKey(), req.GetValue())   // record already installed in memory
...
if err != nil { return nil, mapError(err) }                // ErrDumpFailed returns here
s.notifyLocalWrite(req.GetKey(), ver, req.GetValue(), false)
```

- **Spec.** §Create and Update: on dump failure the client gets `Unavailable`; "the record
  stays in memory **and is replicated to the neighbour, therefore it is not lost while a
  neighbour exists**".
- **Failure scenario.** Disk full or failing. The record is visible to `Get`, already delivered
  to Watch subscribers, but not broadcast and not tracked as unacked. It reaches the peer only
  through the next listing (up to 30 s; never under RPL-01). A node with a dying disk that
  crashes inside that window takes the write with it. `engine.Set` returns the stamped version
  together with `ErrDumpFailed`, so the handler has everything it needs.
- **Fix.** On `persist.ErrDumpFailed` call `notifyLocalWrite` before returning `Unavailable`.
- **Acceptance.** A `grpcapi` test with a failing `DumpStore` and a recording `LocalNotify`:
  `Set` returns `Unavailable` and the notify seam received `(key, version)`.

### Medium

---

#### RPL-09 — `NotifyLocalWrite` replaces the pending entry unconditionally; out-of-order notifications drop tracking of the newer version

- **Severity:** Medium · **Status:** Verified (unit repro)
- **Where:** `internal/app/repl/core.go:689-709`; the comment says "A newer local write …
  REPLACES the entry", the code does not check "newer".

```go
c.pending[key] = pendingEntry{ rec: own, version: own.Version, ... }
```

- **Failure scenario.** Two concurrent `Set` RPCs on one key are confirmed by the same dump and
  their handler goroutines race to `notify` — normal under "bursts up to ~1000 writes/s on hot
  keys". If the older version is notified second, it replaces the newer entry. When the newer
  broadcast is lost, the peer Acks the older one, the entry drains, and the newer version is
  never retransmitted; the loss bound degrades from `ack_timeout_max` to `sync_interval`. The
  same happens when a remote newer version was merged while the handler waited for the dump.
- **Fix.** Ignore a notification whose version is not greater than the tracked one; better,
  pass only the key and let the core read the current record from the store (removes the record
  rebuild in `grpcapi.notifyLocalWrite` too).
- **Acceptance.** `TestReview_outOfOrderNotifyReplacesNewerPending` passes.

```go
core.NotifyLocalWrite(now, keyAlpha, rec2, &out)   // v2 first
core.NotifyLocalWrite(now, keyAlpha, rec1, &out)   // then the older v1
core.HandleDatagram(now, ackFrom(peerID1, keyAlpha, v1), &out)
// 20 s of Advance: v2 is never broadcast again and nothing is pending
```

---

#### RPL-10 — The discovery flush and every retransmission round emit the whole pending set at once

- **Severity:** Medium · **Status:** Verified (unit repro)
- **Where:** `internal/app/repl/core.go:499-506`, `:872-887`
- **Failure scenario.** A node serves writes alone for an hour (5 000 keys × 8 KiB pending).
  The peer's first Hello triggers 5 000 broadcasts (39 MiB) in one event-loop iteration, and
  5 000 more 200 ms later because the flush resets every ladder to the same deadline. While the
  loop is busy sending it reads nothing; the peer's Acks overflow the receive buffer, are
  lost, and the next round resends almost everything. `sync_response_rate` exists "so that
  recovery does not overflow socket buffers" (SPEC §Recovery) and this path bypasses it.
- **Fix.** Emit flush and retransmissions through a paced queue; stagger ladder deadlines.
- **Acceptance.** `TestReview_discoveryFlushIsOneUnpacedBurst` passes (≤ rate-limit datagrams
  per second for any pending size).

---

#### RPL-11 — The egress queue is unbounded, not de-duplicated, one FIFO for all peers, and pins popped records

- **Severity:** Medium · **Status:** Verified (unit repro)
- **Where:** `internal/app/repl/core.go:652-657`, `:668-678` (`c.egress = c.egress[1:]`)
- **Failure scenario.** The same 30-entry chunk delivered 1 000 times queues 30 000
  DataRequests for 30 keys — 60 s of backlog. Duplicate listings are routine (startup + restart
  + timer listings, N peers, RPL-12). One peer's backlog delays every other peer's
  `DataRequest` reply. "Newer" responses hold full record copies, and popping by re-slicing
  keeps them reachable until the next reallocation. SPEC §Recovery expects the opposite:
  "what was not sent in this round will be detected in the next".
- **Fix.** Key the queue by (handle, key, kind); cap its length and drop the oldest; queue keys
  and read records at send time; nil the slot on pop.
- **Acceptance.** `TestReview_egressQueueUnboundedNoDedup` passes; queue length ≤ configured cap
  under a replayed-listing fuzz.

---

#### RPL-12 — A duplicate `node_id` seen from a third node is a perpetual restart: one full listing per Hello, never classified

- **Severity:** Medium · **Status:** Verified (unit repro)
- **Where:** `internal/app/repl/core.go:405`, `:486-494`
- **Failure scenario.** Nodes X1 and X2 share node_id 2; node C sees their run_ids alternate.
  Each Hello is "run_id differs from the remembered one", so C sends a full targeted listing —
  19 listings in 10 s in the repro — and counts zero `duplicate_node_id` discards and logs
  nothing. Combined with RPL-11 the receivers' queues grow without bound. The handler comment
  ("the flap case is bounded by this idempotence") is false for two live runs.
- **Fix.** Limit targeted listings to one per node_id per `peer_timeout`; when a row's run_id
  changes while its previous `lastHello` is younger than `2 × hello_interval` and the handle
  differs, classify as duplicate and log.
- **Acceptance.** `TestReview_duplicateNodeIDFlapsRestartOnThirdNode` passes.

---

#### RPL-13 — `proto_version` is checked after the body decode, and version skew is logged only at Debug

- **Severity:** Medium · **Status:** Verified (codec probe)
- **Where:** `internal/udprepl/repl.go:749-757`, `:262-264`; `internal/app/repl/core.go:376-390`

```go
err := proto.Unmarshal(env.GetBody(), &body)
if err != nil { return port.InboundDatagram{Status: port.Undecodable} }   // envelope fields dropped
msg, ok := portMessage(&body)
if !ok { return port.InboundDatagram{Status: port.Undecodable} }
```

- **Failure scenario.** During a rolling upgrade a v2 node sends a body the v1 schema does not
  know. The v1 node counts `undecodable`, not `bad_proto_version`. Even a correctly classified
  mismatch is logged only through `slog.Debug`, so at the default level the operator sees
  nothing while the two nodes run independently. The peer never enters the table, so
  `wait_for_replication` answers `acked=true` (API-03). SPEC §failure table: "datagrams are
  dropped **with a log entry**".
- **Fix.** Check `proto_version` right after the envelope unmarshal, before CRC and body.
  Log `bad_proto_version` and `duplicate_node_id` at Warn, rate-limited per (node_id, class).
- **Acceptance.** `TestProbe_futureProtoVersionMisclassified` passes; a log-capture test sees
  one Warn record for the first mismatching datagram at the default level.

---

#### RPL-14 — The read loop exits permanently and silently on any read error

- **Severity:** Medium · **Status:** Verified (unambiguous from code); the trigger is Suspected
- **Where:** `internal/udprepl/repl.go:494-497`

```go
n, src, err := a.conn.ReadFromUDP(buf)
if err != nil { return } // the socket closed — Run's exit path.
```

- **Failure scenario.** A transient error (`ENOMEM`/`ENOBUFS` under memory pressure) ends the
  goroutine. `Run` keeps going: the node still broadcasts Hello and its own writes, so peers
  consider it alive, but it never receives again. No log, no metric, no exit; `main` discards
  `Run`'s result anyway.
- **Fix.** Return only on `net.ErrClosed`; otherwise log, count, back off and continue — or
  propagate the error out of `Run` and exit non-zero.
- **Acceptance.** A fake-conn unit test: a non-closed read error is logged and the loop reads
  the next datagram.

---

#### RPL-15 — Every event-loop iteration copies and sorts the whole pending set

- **Severity:** Medium · **Status:** Verified (measured at the core seam; adapter part by code)
- **Where:** `internal/udprepl/repl.go:336-339`, `:567-613`; `internal/app/repl/core.go:1026-1038`
- **Failure scenario.** With the peer down, the pending set holds one entry per written key.
  At 10 000 entries the snapshot + clone costs 3.6 ms per iteration, i.e. at most ~276 events/s.
  Every local write is one event, so above that rate the 1 024-slot notify queue fills and
  `Set` handlers block after their write was confirmed.
- **Fix.** Emit drain results from the core (architecture note 4) and keep an incremental
  mirror; drop the per-iteration `PeerSnapshot()` used only for its length.
- **Acceptance.** `TestReview_perEventMirrorCostWithLargePending` passes (< 200 µs per event at
  10 000 pending).

---

#### API-02 — `create_if_absent` on a tombstoned key returns `AlreadyExists`; the spec says it writes

- **Severity:** Medium · **Status:** Verified (code vs spec; cross-boundary with `storage`)
- **Where:** `internal/grpcapi/server.go:142-143`, `supkvapi.proto:58-60`,
  `internal/app/storage/store.go:193-195`
- **Spec.** §Create and Update: "if the key exists **and is not deleted**, `AlreadyExists` is
  returned, otherwise an ordinary write is performed."
- **Failure scenario.** `Get(k)` answers `NotFound`, `Set(k, create_if_absent)` answers
  `AlreadyExists`. Tombstones are kept forever, so a deleted default can never be re-seeded by
  the build.
- **Fix.** Follow the spec, or record the deviation in an ADR and in `OPERATIONS.md`.
- **Acceptance.** API test: Set, Delete, Set(create_if_absent) → `OK`, `Get` returns the value.

---

#### API-03 — `wait_for_replication` reports `acked=true` with no peer, counts instead of identities, and stalls 500 ms on hot keys

- **Severity:** Medium · **Status:** Verified (unambiguous from code)
- **Where:** `internal/grpcapi/wait.go:95-105`, `:113-130`, `:158-180`

```go
if peerCount == 0 { return true, 0, 0 }                                  // :101
if info, ok := s.opts.PendingCopy()[key]; ok && info.Version == ver {    // :114 — the only early exit
```

- **Failure scenarios.**
  1. Two-node cluster, peer dead longer than `peer_timeout` (or on another `proto_version`):
     `acked=true, total_peers=0`. The flag is true exactly when no second copy exists. The
     spec says the flag tells "whether it [the Ack] was received".
  2. Concurrent writes to one key: only the last-notified version stays in the mirror. Every
     other waiter sees a version mismatch, polls the full 500 ms `waitAppear`, and returns
     `acked=false` although its version may have been Acked.
  3. The success test is `min(len(Acked), total) == total`. An Ack from a node outside the
     write-time peer set substitutes for a missing one; a response can carry `acked=false,
     acked_count == total_peers`.
- **Fix.** Return `acked=false` when `total_peers == 0`. Snapshot the awaited node ids when the
  entry is created and intersect. Return at once when a greater version is tracked. Best:
  deliver drain results from the core to a waiter channel.
- **Acceptance.** Table test over fake seams: zero peers → `(false, 0, 0)`; stranger Ack never
  yields `acked=true`; superseded waiter returns in < 20 ms.

---

#### API-04 — `SnapshotLoad` is not announced to replication and does not delete keys absent from the snapshot

- **Severity:** Medium · **Status:** Verified (code; cross-boundary with `persist.Restore`)
- **Where:** `internal/grpcapi/server.go:243-252`, `internal/app/persist/engine.go:572-591`
- **Spec.** §Snapshots: load "performs a write with a new `RecordVersion` for every snapshot
  key; **for keys present in the map and absent from the snapshot a delete is performed**.
  Thanks to this the load is replicated to the second node in the regular way."
- **Failure scenario.** `Restore` writes through `store.Set`/`Delete`; the handler never calls
  `notifyLocalWrite`. Peers learn of the restore only from the next listing (up to 30 s, or
  never for tail keys under RPL-01), and nothing is tracked as unacked. Keys created after the
  snapshot survive the "restore", so the result is not the snapshot state.
- **Fix.** Have `Restore` return the replayed `(key, record)` list and notify each; implement
  the delete half or document the deviation.
- **Acceptance.** Two-node test: after `SnapshotLoad` on A, B equals the snapshot within one
  RTT, including deletion of a post-snapshot key.

---

#### API-05 — `SnapshotCreate` returns `OK` with an unloadable snapshot after a `.prev` fallback start

- **Severity:** Medium · **Status:** Verified (probe with `persist` + `persistfile` in a temp dir;
  the defect is in `persist`, reported here because the RPC contract is in scope)
- **Where:** `internal/grpcapi/server.go:227`, `internal/app/persist/engine.go:212-220`, `:544-550`
- **Failure scenario.** The main dump is torn; startup loads `.prev`. The engine is clean, so
  `Snapshot` skips the forced dump and byte-copies the corrupt main file. The RPC answers `OK`;
  a later `SnapshotLoad` fails with `Internal`.
- **Fix.** Mark the engine dirty when startup did not load the main generation, or validate the
  bytes with the codec before copying.
- **Acceptance.** `TestProbe_snapshotAfterPrevFallbackCopiesCorruptMain` passes.

```go
eng2, _, stop2 := openEngine(t, path)        // main corrupt, state served from .prev
_ = eng2.Snapshot("backup")                  // returns nil
// snapshot bytes = "{ this is not a dump"; eng2.Restore("backup") => persist: dump header unreadable
```

---

#### API-06 — `List` deep-copies every value, has no pagination, and its size comment is wrong

- **Severity:** Medium · **Status:** Verified (code; grpc defaults read from the pinned module)
- **Where:** `internal/grpcapi/server.go:191-212`
- **Failure scenario.** `s.engine.Records()` copies all values to return key names (600 MB at
  the envelope's edge) under the store read lock. The comment says the response "fits the gRPC
  4 MiB default send limit" and names `grpc.MaxSendMsgSize` as the knob. In grpc v1.85 the
  server's default send limit is `math.MaxInt32`; 4 MiB is the *client's* default receive
  limit. 45 000 keys of 100 B (4.6 MB) are sent and then rejected by every default client with
  `ResourceExhausted`; the server knob cannot fix that.
- **Fix.** List from a key/version view; add `page_size`/`page_token` (or stream); correct the
  comment.
- **Acceptance.** A default client lists 100 000 keys of 100 B successfully; `List` allocates
  O(keys), not O(values).

---

#### BIN-01 — Configuration is not validated: out-of-range values are silently replaced, one typo panics, small values spin

- **Severity:** Medium · **Status:** Verified (binary run with an unbindable address; core repro)
- **Where:** `cmd/sup-kv/main.go:381-393`, `:691-706`; normalisation in `core.go:136-168`
- **Evidence.** This command passes validation and only fails at the (deliberately invalid)
  bind: `--hello-interval=-5s --peer-timeout=1ns --ack-timeout=10s --ack-timeout-max=1ms
  --sync-chunk-bytes=-7 --value-max-bytes=99999999 --sync-response-rate=2000000000
  --retransmit-interval=1ns stray-positional-arg`.
- **Consequences.**
  - Negative or zero values silently become defaults; a rate above 1e9 silently becomes 500.
  - `--sync-chunk-bytes=1125899906842624` panics at the first `Advance`
    (`makeslice: cap out of range`, `core.go:799`); values above 65 507 make every chunk
    unsendable and kill anti-entropy with only Warn lines.
  - `--retransmit-interval=1ns` or `--hello-interval=1ns` turn the event loop into a busy loop
    (and a broadcast flood for Hello).
  - `peer_timeout ≤ hello_interval` makes every peer flap; `ack_timeout > ack_timeout_max`
    makes the ladder go down; `value_max_bytes` above the datagram limit is RPL-06.
  - Positional arguments are ignored.
- **Fix.** One `validate(cfg)` with explicit ranges and relations (`peer_timeout ≥ 2 ×
  hello_interval`, `ack_timeout ≤ ack_timeout_max`, `128 ≤ sync_chunk_bytes ≤ 65000`,
  intervals ≥ 1 ms, `flag.NArg() == 0`), exit 2 with the offending knob named.
- **Acceptance.** Table test over `finalizeConfig`: each bad value returns an error naming the
  flag; `TestReview_hugeSyncChunkBytesPanics` passes.

---

#### BIN-02 — Fatal errors are logged at INFO and disappear at `SUPKV_LOG_LEVEL=warn`; startup, load source and dump failures are never logged

- **Severity:** Medium · **Status:** Verified (binary run; log call sites enumerated)
- **Where:** `cmd/sup-kv/main.go:231`, `:241`, `:246`, `:259-266`, `:414-422`
- **Evidence.**

      $ sup-kv ... --bind-addr 256.0.0.1:1 ...
      {"level":"INFO","msg":"sup-kv: bind replication socket: ... no such host"}   exit=1
      $ SUPKV_LOG_LEVEL=warn sup-kv ... --bind-addr 256.0.0.1:1 ...
      (no output)                                                                   exit=1

  `slog.SetDefault` routes the `log` package through the handler at INFO, so `log.Fatalf` and
  `log.Printf` are filtered at warn/error: bind failures, "final dump failed", "discarded N
  undecodable dumped records", forced-stop notices.
- **Never logged at any level:** version and effective configuration at start; which dump
  generation was loaded (`persist.Open` drops `Load`'s `ok`, so an unreadable or mistyped
  `--dump-path` is an indistinguishable silent empty start — SPEC §Dump: "writes an error to
  the log and sets a metric"); dump failures and their cause (the `ENOSPC`/`EACCES` is
  discarded, clients see only "dump failed"); gate opening and why; peer discovered, evicted,
  restarted; drift warnings.
- **Fix.** Use `slog.Error` + `os.Exit` for fatal paths; log a startup record; return the load
  outcome from `persist.Open`; give the engine a logger or an error callback.
- **Acceptance.** With `SUPKV_LOG_LEVEL=error` a bind failure prints one `level=ERROR` record; a
  start with two unreadable generations prints an error naming the path; a failing dump prints
  the underlying error once per failure.

### Low

---

#### RPL-16 — The receive path accepts well-formed garbage, and the CRC protects only the body

- **Severity:** Low · **Status:** Verified (codec probe)
- **Where:** `internal/udprepl/repl.go:797-838`, `:715`, `:745`
- **Observed.** A `DataReplication` with an **empty key** is stored and Acked; the API can then
  neither read nor delete it, yet `List` and Watch show it. A tombstone carrying a payload is
  stored as is. The wire `checksum` is transported and ignored (`0xDEADBEEF` is accepted and
  recomputed). `crc32` covers `body` only: a datagram with a flipped `node_id` or `run_id`
  passes as `Decoded`, which can create a phantom peer that joins every new wait-set until it
  expires.
- **Fix.** Reject empty and over-long keys and tombstones with data as `undecodable`; verify or
  drop the record checksum field; extend the CRC over the envelope header in the next
  `proto_version`.
- **Acceptance.** `TestProbe_wellFormedGarbageAccepted` and
  `TestProbe_crcDoesNotCoverEnvelopeIdentity` pass.

---

#### RPL-17 — Socket setup traps: `SO_REUSEADDR` in production, bind to a specific address, no timer jitter

- **Severity:** Low · **Status:** `SO_REUSEADDR` Verified by code; bind behaviour Suspected
- **Where:** `internal/udprepl/repl.go:270-291`, `cmd/sup-kv/main.go:297-300`
- **Details.**
  - `SO_REUSEADDR` exists for same-host tests. In production a second instance started by
    mistake binds the same port without `EADDRINUSE` and takes over the unicast traffic (Acks,
    requests, replies).
  - Suspected: `--bind-addr 192.168.1.10:7900` (a natural choice) receives no broadcasts on
    Linux or BSD, so discovery silently never happens. Confirm with two hosts; if so, reject or
    warn for a non-wildcard bind address. Nothing checks that the broadcast port matches the
    peers' bind port either.
  - Hello and sync timers have no jitter: nodes powered on together emit their listings in the
    same instant every 30 s.
  - `ownHandle` (`core.go:285`) is stored and never read.
- **Fix.** Make address reuse a test-only option; validate or warn on the bind address; add
  ±10 % jitter; delete the dead field.
- **Acceptance.** Starting a second instance on the same bind address fails with
  `address already in use`.

---

#### RPL-18 — Observability gaps in the replication path

- **Severity:** Low · **Status:** Verified (code)
- **Where:** `internal/udprepl/repl.go:522-533`, `internal/app/repl/core.go:531-541`,
  `internal/grpcapi/server.go:143-145`
- **Details.** `countSent` runs before the send, so failed sends are counted as sent and there
  is no send-error counter (RPL-07 shows `data=3` for one delivered datagram). Send errors are
  logged per datagram without rate limit. `Conflicts` adds every displaced or rejected version,
  so ordinary remote updates and stale redeliveries count as conflicts. The one real conflict —
  a pending local entry superseded by a remote version (`core.go:539`) — is neither logged nor
  journalled, although SPEC §Semantics requires the losing version in a conflict journal
  (`OPERATIONS.md` declares "there is no conflict journal", a documented deviation). Drift
  warnings are counted, never logged; the drift result of a local write is discarded.
- **Fix.** Count after a successful send and add `send_errors_total{errno}`; count and log
  conflicts only at the supersession site; log drift warnings rate-limited.
- **Acceptance.** Metrics test: a failed send increments the error counter only; a concurrent
  write scenario logs exactly one conflict with the losing version.

---

#### API-07 — Unmapped errors become `Internal` and leak server paths

- **Severity:** Low · **Status:** Verified (code)
- **Where:** `internal/grpcapi/server.go:306-309`, `internal/persistfile/snapshot.go:59-75`
- **Failure scenario.** `SnapshotLoad` of an unknown name returns `Internal: internal error:
  open /var/lib/…/dump.json.snapshots/x: no such file or directory`. `SnapshotCreate` on a node
  that has never dumped returns `Internal: … no dump on disk to snapshot`.
- **Fix.** Map `fs.ErrNotExist` → `NotFound`, `ErrNoDump` → `FailedPrecondition`, decode errors
  → `DataLoss`; log the detail server-side and return a fixed message.
- **Acceptance.** API test: unknown snapshot → `NotFound` with no path in the message.

---

#### API-08 — Watch: clean `OK` on server shutdown, full deep copy on the hub goroutine

- **Severity:** Low · **Status:** Verified (code); the resubscribe storm is Suspected
- **Where:** `internal/grpcapi/watch.go:350-355`, `:390`, `:263-283`
- **Details.** On shutdown the hub closes subscriber channels with a nil error and the handler
  returns it, so a watch ends with status `OK`; clients cannot tell it from normal completion
  and standard retry policies do not fire. Use `Unavailable`. Each subscribe deep-copies the
  whole store on the hub goroutine, blocking fan-out; after `disconnectAll` all clients
  resubscribe together, which can overflow the 4 096-slot ingress again. A subscriber still
  streaming a large snapshot is shed after 256 live events and will be shed again on retry
  under steady writes.
- **Acceptance.** A watch client sees `Unavailable` on server shutdown; subscribe cost on the
  hub goroutine is independent of value sizes.

---

#### API-09 — gRPC server runs with bare defaults; health can flip back to SERVING during shutdown

- **Severity:** Low · **Status:** Verified (code)
- **Where:** `cmd/sup-kv/main.go:588`, `:645-654`
- **Details.** `grpc.NewServer()` has no keepalive enforcement, no `MaxConcurrentStreams`, no
  panic-recovery interceptor (a handler panic kills the process without the final dump) and no
  error logging. In `shutdown`, health is set to `NOT_SERVING`, then the metrics listener is
  drained for up to 10 s, and only then `apiCancel()` stops `RunGate`; a SIGTERM inside the
  startup gate lets the gate open in between and set `SERVING` again.
- **Fix.** Add the options; call `health.Server.Shutdown()` (later status changes are ignored)
  or cancel the gate before the health flip.
- **Acceptance.** Test: SIGTERM at t = 0.5 × `startup_gate` — health never reports `SERVING`
  afterwards.

---

#### BIN-03 — Exit codes and lifecycle edges

- **Severity:** Low · **Status:** Verified (code, binary run)
- **Where:** `cmd/sup-kv/main.go:229-249`, `:601-605`, `:676`
- **Details.** Every startup failure exits 1, which `OPERATIONS.md` defines as "the final dump
  failed"; a port conflict therefore loops under `Restart=on-failure`. A premature `Serve`
  error calls `log.Fatalf` and skips `shutdown`, so remote merges since the last dump are not
  flushed. Signals are armed only after `wire()` (dump load, binds). `SIGHUP` is not handled.
  `engine.Flush()` has no deadline, and by then further signals are swallowed, so a hung disk
  blocks shutdown until SIGKILL. `--help` does not list the five env-only knobs
  (`SUPKV_LOG_LEVEL`, `SUPKV_LOG_FORMAT`, `SUPKV_METRICS_CERT/KEY/CLIENT_CA`). A malformed
  `SUPKV_*` value aborts startup even when its flag overrides it.
- **Fix.** Distinct exit codes (2 usage, 3 startup, 1 dump); run `shutdown` on the `Serve`
  error path; arm signals first; bound the final flush.
- **Acceptance.** Exit-code table test; a `Serve` failure still produces the final dump.

### Info

---

#### INF-01 — Comments that contradict the code

| Location | Comment | Reality |
|---|---|---|
| `server.go:196-200` | List "fits the gRPC 4 MiB default send limit"; `MaxSendMsgSize` is the knob | Server send default is `MaxInt32`; 4 MiB is the client receive default (API-06) |
| `core.go:686` | "A newer local write … REPLACES the entry" | Any notification replaces it (RPL-09) |
| `core.go:443-445` | "the flap case is bounded by this idempotence" | Unbounded for two live runs (RPL-12) |
| `core.go:440-441` | the peer's startup DataSync and periodic rounds reconcile a new peer | Not for keys the peer lacks (RPL-03) |
| `core.go:175-177`, `store.go:73-74` | "internal/port holds no interfaces" | `port.DumpStore` and `port.Message` are interfaces |
| `core.go:328-331` | `ownHandle` "kept for the later own-echo … work" | That work is done and does not use it |
| `repl.go:52-55` | a full inbound queue "backpressences the segment" | UDP has no backpressure; datagrams are dropped (RPL-01) |
| `supkv.proto:4-6` | core "consumes only Hello, DataReplication and Ack" | All five are implemented |
| `supkvapi.proto:3-5` | "the unimplemented RPCs inherit Unimplemented" | All seven are implemented; `.planning/WINDOWS.md` id 3 is still open for it |
| `AGENTS.md:34-39` | `main()` holds no testable logic; `internal/app` has three packages | `main.go` has 554 code lines; `internal/app` has five packages |
| `docs/OPERATIONS.md:40-48` | convergence bounds "proven in simulation" | The simulator has no buffers, sizes or dump delay (architecture note 2) |

---

#### INF-02 — Notes for the reviewer of `storage` / `persist` / `persistfile`

Seen while tracing calls across the boundary; not reviewed in depth.

- `persistfile.Load` treats every read error (including `EACCES`, `EIO`) as "try `.prev`", and
  the next two dumps rotate the unreadable-but-valid main file out of existence.
- The dump trigger is a trailing-edge debounce reset by every write, bounded by
  `dump_max_wait` = 1 s. Under a steady 1 000 writes/s each write waits up to 1 s; the spec
  describes a minimum pause after the previous dump (latency ≤ `dump_debounce` + dump time).
  Replication of a local write starts only after that dump.
- `version.PackVersion` uses zero filler where the spec says random bits; an exact
  `(l, c, node_id)` repeat after a lost disk plus a backward clock is then an equal version with
  different data.
- `clock.Update` computes the drift signal before absorbing the remote reading, so the first
  far-ahead version is never reported.
- `persist.Open` installs dump records through `store.Merge`, which fires the Watch tap for
  every record before the hub runs (harmless; the 4 096-slot ingress overflows on larger dumps).
