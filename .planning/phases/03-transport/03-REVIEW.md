---
phase: 03-transport
reviewed: 2026-11-21T23:38:34Z
depth: standard
files_reviewed: 54
files_reviewed_list:
  - cmd/sup-kv/main.go
  - cmd/sup-kv/main_test.go
  - docs/OPERATIONS.md
  - docs/SPEC-DEVIATIONS.md
  - .github/workflows/scheduled.yml
  - internal/app/internal/simnet/simnet_ext_test.go
  - internal/app/internal/simnet/simnet.go
  - internal/app/internal/testutil/gen.go
  - internal/app/persist/codec.go
  - internal/app/persist/codec_test.go
  - internal/app/persist/engine.go
  - internal/app/persist/engine_test.go
  - internal/app/repl/bench_test.go
  - internal/app/repl/core.go
  - internal/app/repl/discard_test.go
  - internal/app/repl/egress_internal_test.go
  - internal/app/repl/merge_stats_test.go
  - internal/app/repl/peertable_test.go
  - internal/app/repl/restart_test.go
  - internal/app/repl/retransmit_test.go
  - internal/app/repl/sim_chaos_test.go
  - internal/app/repl/sim_scenario_test.go
  - internal/app/repl/sync_test.go
  - internal/app/repl/warn_test.go
  - internal/app/storage/store.go
  - internal/app/storage/view_test.go
  - internal/app/version/validate_test.go
  - internal/app/version/version.go
  - internal/grpcapi/bench_test.go
  - internal/grpcapi/gate.go
  - internal/grpcapi/server.go
  - internal/grpcapi/server_test.go
  - internal/grpcapi/wait.go
  - internal/grpcapi/wait_integration_test.go
  - internal/grpcapi/wait_test.go
  - internal/metrics/metrics.go
  - internal/metrics/metrics_test.go
  - internal/persistfile/fsops.go
  - internal/port/repl.go
  - internal/stand/stand_buffers_test.go
  - internal/stand/stand_test.go
  - internal/udprepl/decode_negative_test.go
  - internal/udprepl/maxdatagram.go
  - internal/udprepl/maxdatagram_test.go
  - internal/udprepl/readloop_internal_test.go
  - internal/udprepl/repl.go
  - internal/udprepl/repl_internal_test.go
  - internal/udprepl/repl_test.go
  - internal/udprepl/scenario_test.go
  - internal/udprepl/smoke_test.go
  - internal/udprepl/socket_internal_test.go
  - internal/udprepl/syncsize_internal_test.go
  - Makefile
  - scripts/red-expectations.tsv
findings:
  critical: 2
  warning: 4
  info: 7
  total: 13
  necessity: 0
status: issues_found
---

# Phase 3: Code Review Report

**Reviewed:** 2026-11-21T23:38:34Z
**Depth:** standard
**Files Reviewed:** 54
**Status:** issues_found

## Summary

Reviewed the complete Phase 3 (03-transport) change scope: the replication core rewrite (record-free unacked ledger, indexed due-heap, result stream, three pacers, listing FIFOs, wire validation), the UDP adapter (tracker lifecycle, read loop, socket-buffer policy, encode/decode), the persistfile/engine dead-code removals, the gRPC `wait_for_replication` seam, the metrics families, the docs, the Makefile/red-gate machinery and the new scheduled job.

Gates run for this review: `go build ./...` (clean), `go tool golangci-lint run ./...` (clean), and `go test -race -count=1` over `internal/app/repl`, `internal/udprepl`, `internal/grpcapi`, `internal/metrics`, `internal/app/persist`, `internal/persistfile`, `internal/app/internal/simnet`, `internal/app/version`, `internal/app/storage`, `internal/stand` and `cmd/sup-kv` (green, except one flake — see WR-04). The design intent checks out on the main points I was asked to verify: validation runs strictly between the identity classifier and `handleMessage`, and the only store-with-clock path is `applyRemote → ApplyRemote → storage.Merge → clock.Update` (the only production caller outside the receive path is startup seeding in `persist.mergeRecords`; restore re-enters through `Set`/`Delete`, never `Merge`); the due-heap, `PeerCount` and `UnackedStats` are O(1)/O(log n) with no store copies; listing chunking packs to the budget, rotates its start chunk and drains one datagram per jittered slot; jitter is deterministic FNV over the run id (no `math/rand` in the core); the key/64 408/65 507 ceiling arithmetic is exact (verified: key 1 024 + value 64 408 + the Body/Envelope framing is exactly 65 507); the read loop exits only on `net.ErrClosed`, `SO_REUSEADDR` is test-only, and the dead code (`lastLocal`/`cover`, `persist.Parse`, `ownHandle`, `fsync_darwin/other`) is gone with no dangling references.

Two availability defects in the new queue machinery survive that review and must be fixed before this ships: the restart-listing queue has no bound and is keyed by an attacker-chosen source handle (CR-01), and the "bounded deduplicated egress" still materializes and retains a full record copy per queued answer, contrary to its own contract (CR-02). Both were reproduced with a throwaway white-box overlay test (no tree modification): 100 spoofed restart Hellos left 96 listing FIFOs and 4 800 queued chunks while 200 paced Advances drained only 200; 100 queued answers pinned 6 000 000 bytes of Data (60 000 per entry), extrapolating to ~1.9 GiB at the 32 768-entry cap with the default 61 440-byte value limit.

**Red-gate state (measured).** The table holds 7 rows: 3 green (`TestRed_R01_pacedSync20k`, `TestRed_R02_unackedDrain`, `TestRed_R02_burst`) and 4 red (`TestRed_R03_gateClosedUntilFullList`, `TestRed_R03_f14ReplacedNode`, `TestRed_P01_valueLimit`, `TestRed_P01_valueLimitEngine`). That matches 03-CONTEXT (R-01/R-02 flip in Phase 3; R-03/P-01 stay red for Phases 4/5) and 03-13-SUMMARY ("R-03 x2, P-01 x2 remain red; the three transport rows are green"), but it does **not** match the "5 green / 2 red" figure quoted in this review's focus and in `.planning/phases/03-transport/03-VALIDATION.md:20`. The table itself is internally consistent with the repository; the 5/2 wording appears stale. `.planning` is outside this review's file scope, so this is recorded as context, not a numbered finding.

**Measurement alignment (6c17db3).** I judge the change still enforces the documented XPORT-02(3) acceptance. `REQUIREMENTS.md:56` states two independent clauses — "at least 990 writes/s achieved" and "set empty 5 s after the end". The test now divides successful ops by the 14.999 s open-loop schedule window and asserts the unacked set empty within `sendEnd + 5 s + 2 s` poll margin (`internal/stand/stand_red_test.go:537-590`). That is the documented reading; the earlier denominator silently added the in-flight drain to the rate window, demanding a ≤151 ms drain, which the acceptance never required. The drain bound still catches a run whose send loop or RPCs lag, because the deadline is measured from the pre-burst `sendEnd`, not from the end of `redRunBurst`.

**Known accepted flake.** `TestConfig_parametersByEffect/peer_timeout_evicts_silent_peer` is timing-sensitive under load, but I found no structural cause: with the stand's 100 ms hello interval and the 500 ms configured peer timeout, the earliest possible eviction after the kill is ~400 ms, safely above the 250 ms lower bound, and the 3 s upper bound is 6× the timeout. Treat the observed single failure as load sensitivity, not a defect. Separately, a *different* flake reproduced during this review: `cmd/sup-kv` `TestLog_formatText` failed once with `bind: address already in use` and passed 3/3 reruns — that one has a structural cause and is filed as WR-04.

## Narrative Findings (AI reviewer)

### Critical Issues

#### CR-01: Restart-listing queue is unbounded and keyed by an attacker-controlled source handle

**File:** `internal/app/repl/core.go:537-541` (fields), `:948-963` (restart branch), `:1380-1399` (`enqueueRestartListing`), `:1405-1417` (`enqueueListingChunk`), `:1513-1520` (`listingChunks`)
**Issue:** A Hello from a **known** `node_id` carrying a different `run_id` is unconditionally treated as a restart and enqueues a full chunked store listing into `listingByDst[handle]` — a map keyed by the datagram's source handle. Nothing bounds the number of destinations or the total queued chunk count: `listingByDst`, `listingOrder` and `listingCount` grow with every new handle, and the shared pacer drains only 500 chunks/s (`drainListings`). There is no analogue of the egress `maxEgressEntries` cap for listings. One Hello is ~40 bytes; a full listing of a 20 000-key store is ~500-625 chunks (~0.8-1 MB of retained `SyncEntry` slices), so two spoofed datagrams (learn, then a second run id from a new source port) buy ~1 MB of permanently-until-drained memory, and every further source port adds another full listing. The retrigger is also O(store): each restart Hello calls `listingChunks()` → `store.Versions()` (full map copy) + sort. A hostile segment host can therefore drive the event loop to OOM or starve it on CPU at will. Reproduced: 100 restart Hellos with fresh handles left 96 FIFOs / 4 800 queued chunks, while 200 paced Advances (0.6 s of virtual time) drained 200 chunks. The base commit emitted the restart chunks straight to the Outbox (no retention), so the unbounded queue is introduced by this phase.
**Fix:** Bound it the same way egress is bounded, and stop materialising per-destination listings:
```go
// core.go: one global cap, counted like maxEgressEntries.
const (
    maxListingChunks       = 4096 // a small multiple of one store round
    maxListingDestinations = 64
)

func (c *Core) enqueueRestartListing(now time.Time, peerID uint16, handle string) {
    if _, busy := c.listingByDst[handle]; busy {
        return
    }
    // Coalesce per node id and refuse when the queue is at its bound.
    if len(c.listingByDst) >= maxListingDestinations || c.listingCount >= maxListingChunks {
        c.listingDropped++
        return
    }
    ...
}
```
Additionally: key the FIFO/cooldown by `node_id` rather than by handle and rate-limit the restart reaction per peer (e.g. at most one targeted listing per `sync_interval`), so a flapping or hostile `run_id` cannot mint a new listing per datagram; and build chunks lazily (chunk index into the sorted listing at drain time) so a queued destination costs O(chunk), not O(store). A drop counter belongs on the same observability surface as `EgressDropped`.

#### CR-02: The egress queue pins a full record copy per answer (and allocates one per duplicate), contradicting "the queue never pins Data"

**File:** `internal/app/repl/core.go:369-372` (`queuedUnicast.msg`), `:1146-1162` (`enqueueResponse`), `:1088-1094` (`handleDataRequest`), `:1116-1133` (`handleDataSync`), `:1226-1235` (`resolveQueuedResponse`); cap at `:97-101`
**Issue:** `egressKey`'s doc says "a DataReplication answer re-reads the stored record at drain time, so the queue never pins Data" and `resolveQueuedResponse` does re-read through `Lookup`. But both producers first call `Lookup` and store the returned record (a **cloned** `Data` slice) inside `queuedUnicast.msg`; the drain never uses that copy. Consequences: (a) retention — the queue can hold `maxEgressEntries` (32 768) record copies, i.e. up to 32 768 × `value_max_bytes` ≈ 1.9 GiB at the 61 440-byte default, on top of the store's own data; this is remotely triggerable because a DataSync listing carrying stale-but-valid versions (layout-valid, `l = 0`) makes the node answer with a full record per key, and broadcast listings reveal every key to any segment host; (b) allocation amplification — the `Lookup` happens **before** `enqueueResponse`'s dedupe check, so a repeated listing (or a flood of tiny DataRequests for existing keys) allocates and discards a full value-sized copy for every request; a ~65-byte DataRequest against a 60 KiB value forces ~60 KiB of garbage, and a 2 KB DataSync with 30 entries forces ~1.8 MB. Reproduced: 100 queued answers pinned 6 000 000 bytes (60 000 per entry). The phase's own threat table lists "unbounded pending mirror/queues → memory exhaustion" as the DoS to mitigate, and this queue still carries the data it says it does not.
**Fix:** Carry only the identity in the queue; resolve everything at drain time (where the code already does the right thing):
```go
// queuedUnicast carries the identity only; the message is built at drain time.
type queuedUnicast struct {
    id egressKey // (handle, kind, key) — no Data
}

func (c *Core) handleDataRequest(now time.Time, source, key string) {
    if _, ok := c.store.Version(key); !ok { // O(1) presence probe, no Data copy
        return
    }
    c.enqueueResponse(now, source, egressDataReplication, key)
}

// handleDataSync's local-newer branch: the point Version probe already proved
// the record exists, so enqueue the same identity without the second Lookup.
c.enqueueResponse(now, source, egressDataReplication, entry.Key)

// resolveQueuedResponse builds the message at drain time:
//   egressDataRequest     -> port.DataRequest{Key: id.key}
//   egressDataReplication -> Lookup(id.key), skipping when the key is gone
```
The dedupe map stays; the queue then really is key-only, and the retention bound becomes `32 768 × (key + small struct)` instead of `32 768 × value_max_bytes`.

### Warnings

#### WR-01: A tracked write leaks its tracker forever when `WaitReplicated` is nil while the notify seam creates trackers

**File:** `internal/grpcapi/wait.go:66-70`; `internal/udprepl/repl.go:604-636`, `:638-669`; `internal/grpcapi/wait_test.go:522-547`
**Issue:** `NotifyLocalWrite(key, rec, track)` creates the tracker synchronously for `track == true` (by design, before the enqueue). `waitForReplication` then returns `(true, 0, 0)` immediately when `s.opts.WaitReplicated == nil` — without calling the release path — so the just-created tracker is never deleted. `applyResults` resolves trackers but never deletes them, so every write with `wait_for_replication` adds a permanent map entry (unbounded, keyed by every tracked write). The shipped `main.go` wires both seams, so this is latent; but `ServerOptions`'s zero value is documented legal, `TestWait_trackerNilIsSingleNode` pins a nil seam with a non-zero `KnownPeers`, and nothing enforces "nil `WaitReplicated` ⇒ notify must not track". `TestWait_zeroPeers` (fake tracker) hides the real adapter's tracker creation because the test harness passes `notify == nil`.
**Fix:** Enforce the invariant at construction, so the leak is unrepresentable:
```go
// NewServer: a server that cannot wait must never ask the adapter to track.
if opts.WaitReplicated == nil && notify != nil {
    inner := notify
    notify = func(key string, rec storage.Record, _ bool) error {
        return inner(key, rec, false) // no tracker can ever be released
    }
}
```
(An alternative is a `Release` seam, but forcing `track=false` matches the documented single-node semantics.)

#### WR-02: `sync_chunk_bytes` is unbounded: a typo can OOM the first listing or produce permanently unsendable chunks

**File:** `internal/app/repl/core.go:219-221` (only `<= 0` normalizes), `:1534-1556` (`chunkSyncEntries`, capacity `budget/entryOverheadBytes`), `cmd/sup-kv/main.go:348-349` (flag registered with no validation)
**Issue:** `Config.SyncChunkBytes` accepts any positive `int`. `chunkSyncEntries` pre-allocates `budget/entryOverheadBytes` entries, so `--sync-chunk-bytes=1000000000` requests ~41.7 M `SyncEntry` values (~1.3 GB) on the first listing, and a merely large (e.g. 100 000) budget builds chunks whose encoded DataSync exceeds the 65 507-byte UDP payload and can never be sent (EMSGSIZE, logged per send). Every other size knob in this phase is bounded — `value_max_bytes` has the exit-2 startup ceiling — but the chunk budget has neither a normalization clamp nor a startup check, even though it is the direct input to the same datagram ceiling.
**Fix:** Clamp in `Config.normalized()` and keep the startup check symmetric with `checkDatagramCeiling`:
```go
if c.SyncChunkBytes <= 0 || c.SyncChunkBytes > maxSyncChunkBytes { // e.g. 65507
    c.SyncChunkBytes = defaultSyncChunkBytes
}
```
and reject an explicit out-of-range `--sync-chunk-bytes` at startup with exit 2 rather than silently defaulting, naming the flag and the 65 507-byte ceiling.

#### WR-03: The new `red-r01-x20` evidence step prints a stale committed summary on every fresh checkout

**File:** `.github/workflows/scheduled.yml:169-183`
**Issue:** The step asserts "no red-evidence directory is expected in a fresh checkout", but 30 red-evidence files are tracked in git (`.planning/phases/02-stand-and-simulator-as-tests/red-evidence/*/summary.txt`, latest committed `20261121T230732Z`). `compgen -G` therefore always succeeds on a clean runner, and `ls -1dt ... | head -n 1` prints a previous `red-runner` run's summary — a different command at a different scale — under the label "20x R-01 evidence summary". The job's real verdict is its exit status; the printed artifact is misleading evidence from another run.
**Fix:** Delete the step (the job's exit status is the evidence, as the comment says), or select only evidence produced by this job (e.g. a fresh temp directory passed to the test, or a `find -newermt "$GITHUB_RUN_STARTED_AT"` guard), and write the job's own verdict to `$GITHUB_STEP_SUMMARY`.

#### WR-04: Pick-then-bind UDP port race in the test harness — reproduced during this review

**File:** `cmd/sup-kv/main_test.go:139-148` (`freeUDPPort`), `:1069-1090` (`TestLog_formatText` and its probe); same helper in `internal/udprepl/*_test.go`
**Issue:** `freeUDPPort` binds `0.0.0.0:0`, closes the socket and returns the number; the test then binds it later. Between the two, a parallel test — or the kernel reusing the just-freed ephemeral port — can take it. Observed: `go test -race -count=1 ./cmd/sup-kv/` failed once with `probe rx bind: bind udp4 0.0.0.0:43437: ... address already in use`, then passed 3/3 identical reruns. The phase adds two more parallel tests that hold `:0`-picked UDP ports (`TestMain_bindAddrWarn`, `TestMain_duplicateBind`), widening the existing window as `deferred-items.md` already notes. This is a test-reliability defect: the CI `race-flake`/`red-run` jobs will fail spuriously.
**Fix:** Allocate from a package-level registry (`sync.Mutex` + `map[int]struct{}`) released via `t.Cleanup`, and wrap node start in a bounded retry on `EADDRINUSE` (re-place the node with a freshly picked port). The registry removes same-process collisions; the retry covers the kernel-reuse/cross-process case that no allocation scheme can prevent.

### Info

#### IN-01: The apply-failure WARN names the key and has no rate limit (remote-triggerable log flood)

**File:** `internal/app/repl/core.go:993-998` (apply failure); pre-existing duplicate-node sites at `:719-731`
**Issue:** An oversize-value DataReplication passes wire validation and makes `ApplyRemote` return `ErrValueTooLarge`; the core then logs one WARN per datagram containing the attacker-chosen **key name** — the T-6-05 metadata contract this phase enforces elsewhere says details never carry key names, and the new discard WARNs are per-class rate-limited (`discardValidated`) while these sites are not. A segment host can flood the log at line rate, and the duplicate-node-id Log calls in `HandleDatagram` are likewise one line per datagram. `%q` quoting prevents line injection but not the flooding or the key disclosure.
**Fix:** Route both through the same per-class rate limiter (or one shared `logRateLimit` helper) and drop the key from the message: log `class`/`source handle`/`err` only, or a stable operation name. The counters/`OnDiscard` remain the reliable per-datagram signal.

#### IN-02: XPORT-02(1)'s flatness gate does not cover the paths that still scan the whole pending set

**File:** `internal/app/repl/bench_test.go:260-298` (gate); `internal/app/repl/core.go:1012-1017` (`applyRemote` supersession scan), `:2019-2026` (`removeSupersededLocal`)
**Issue:** The gate measures only an idle `Advance` + `UnackedStats` + `PeerCount` iteration at 10 and 10 000 pending records, which is genuinely O(1). But every inbound DataReplication iterates **all** pending entries to find superseded versions, and every local write does the same in `removeSupersededLocal`; at the documented 20 000-key scale that is O(20 000) per datagram/write, i.e. exactly the "event-loop cost grows with the unacked set" shape XPORT-02 rules against. The executable acceptance carefully avoids those two handlers (the XPORT-06 handler benchmark runs with an empty pending set), so a regression there cannot fail any test. (Performance is otherwise out of v1 scope; this is flagged as an acceptance-coverage gap, not a threshold request.)
**Fix:** Add a per-key index (`map[string]map[pendingKey]struct{}`, maintained beside `peerUnacked`) so supersession touches only the entries under that key, and extend the flatness gate with an inbound-DataReplication sub-benchmark against the same 10/10 000 pending fixtures.

#### IN-03: `TestCore_keyCountCostFlat` is a wall-clock ratio gate marked `t.Parallel`

**File:** `internal/app/repl/bench_test.go:160-194`
**Issue:** The sibling `TestCore_unackedCostFlat` is deliberately sequential because "a parallel sibling benchmark would contaminate the measurement", yet this 2× ratio gate runs `t.Parallel()` (so it measures alongside every other parallel test in the repl package). Local runs measured ratios 0.96-1.22 against the 2.0 bound; under a loaded 2-core runner the margin can narrow.
**Fix:** Drop `t.Parallel()` (the file already serializes the other ratio gate) or, better, replace the wall-clock ratio with a deterministic operation-count assertion (instrumented `Versions()`/`Lookup()` call counts in the fixture) so the gate cannot flake.

#### IN-04: The 50 000-key benchmark fixture packs counters outside `PackVersion`'s documented domain

**File:** `internal/app/repl/bench_test.go:61-64`
**Issue:** `version.PackVersion(testNowMs, uint16(i), localID)` is called with `i` up to 49 999, but `PackVersion`'s contract is counter ≤ 4095: the high bits of the 16-bit counter collide with the version nibble (`v[6] |= 0x70`): for example, counter 0 and counter 4096 pack to identical 16-byte versions, and every `i ≥ 32768` corrupts the version nibble entirely. The benchmark's timing is unaffected (the map key is the string), but the fixture is not a wire-real store and would silently mask any version-order behavior in a future test reusing it.
**Fix:** Vary the timestamp (`testNowMs + int64(i)`) with counter 0, as `benchUnackedFixture` already does, or use `uint16(i % (1 << 12))` with distinct `l`.

#### IN-05: `wait_for_replication`'s "write-time set" is not what the core awaits

**File:** `internal/grpcapi/wait.go:44-84`; `internal/app/repl/core.go:1251-1261`; `internal/udprepl/repl.go:730-754`
**Issue:** The doc says `total` is captured once "so peers discovered during the wait never extend the awaited set". The clamp makes the *reported count* fixed, but resolution is decided by the core's `waitSet`, which is seeded from the live peer table when the event loop processes the notification — strictly later than the caller's `KnownPeers()` snapshot. A peer whose Hello is processed between those two moments joins the wait-set but not `total`; if it never Acks, `ResultDrained` never fires and a write fully covered by the write-time peers reports `acked=false` at timeout. This is the safe direction (never a false success) and the timeout bounds it, but the documented guarantee is stronger than the code.
**Fix:** Pass the write-time peer set with the notification (`NotifyLocalWrite(..., waitSet []uint16)`) and seed `pendingEntry.waitSet` from it, or document the race explicitly in `waitForReplication` and in the API docs.

#### IN-06: `storage.MaxKeyBytes`'s doc claims `Set`/`Delete` enforce the ceiling; the store methods do not

**File:** `internal/app/storage/store.go:20-27`
**Issue:** The constant's comment says "Validation is write-side only: … Set/Delete and the receive path enforce this constant", but `storage.Store.Set`/`SetIfAbsent`/`Delete` check only `ValueMaxBytes`; the key ceiling lives in `grpcapi` and `repl.validate`. A future caller of the store (a library user, `Engine.Restore`'s replay, `simnet`) reading this comment would believe the store is safe, while an over-limit key is installable there. The current shipped paths are fine (API and receive enforce it), so this is a documentation/robustness gap, not a live bypass.
**Fix:** Reword to name the actual gates (`grpcapi.Set/Delete` and `repl.keyClass`), or enforce the ceiling in `Store.Set`/`Delete` too (returning a sentinel) so the comment becomes true.

#### IN-07: `supkv_replication_datagrams_sent_total` counts failed sends

**File:** `internal/udprepl/repl.go:951-970` (`flush`), `:975-988` (`countSent`)
**Issue:** `countSent` runs after encode but before `WriteToUDP`, so a write the kernel rejects (EMSGSIZE on darwin's ~9 KiB limit, `ENETUNREACH`, buffer pressure) still increments the per-type counter whose help text is "Replication datagrams sent, by message type". `pacedSends` explicitly defers RPL-18's send-failure accounting to Phase 7, but `SentByType` carries no such note, so a broken egress path can look healthy on `/metrics`.
**Fix:** Count at the `write` success (move `countSent` after a nil error), or rename/annotate the family as attempted sends and add a failed-send counter now that `flush` already sees the error.

---

_Reviewed: 2026-11-21T23:38:34Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
