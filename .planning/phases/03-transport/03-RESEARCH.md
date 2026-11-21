# Phase 3: Transport - Research

**Researched:** 2026-11-21
**Domain:** Go UDP transport hardening (pacing, buffer sizing, input validation, unacked-ledger cost, ack seam, store read views)
**Confidence:** HIGH for stack/API facts and pitfalls; MEDIUM-HIGH for architecture (design grounded in current code + the committed review, some choices left to the planner)

**Provenance:** `[VERIFIED: source]` = read from a source-of-truth file or tool output **in this session**; `[CITED: source]` = official documentation fetched this session; `[ASSUMED]` = design judgment not confirmed here (see the Assumptions Log). In-repo citations carry the current file and line range; the files were opened with the Read tool this session.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Paced sync listing (XPORT-01 / R-01)
- **D-01:** DataSync listing sends go through a **token-bucket pacer at `sync_response_rate`** shared by all outgoing listing traffic (per-send pacing, not per-chunk burst); the sending **start chunk rotates across sync rounds** (a rotating offset persisted per peer) so repeated rounds cover the whole key space under buffer loss. Convergence bound: 2 × `sync_interval` + N / `sync_response_rate`. — [auto] [Sync pacing] Q: how is the listing paced and made loss-robust? → Selected: token-bucket at sync_response_rate + rotating start chunk (recommended default: the review's own R-01 acceptance describes exactly this shape). — **Reversibility:** reversible.
- **D-02:** `SO_RCVBUF`/`SO_SNDBUF` are set at socket setup (Go `SetReadBuffer`/`SetWriteBuffer` with a double read-back), the **actual values are logged at startup**, and startup **fails when the send buffer cannot hold a maximal datagram**; darwin's ~9 KiB send limit is documented, not fixed (14.10). — [auto] [Socket buffers] Q: configure+report buffers or document limits only? → Selected: configure + log + fail-fast (recommended default: R-08's acceptance wording). — **Reversibility:** reversible.

#### Unacked set and retransmits (XPORT-02 / R-02, XPORT-12)
- **D-03:** The owner-side **unacked ledger replaces snapshot diffing**: the core tracks per-peer unacked records in a map keyed by (key, version), updates it from acked results, and retransmit scanning iterates the ledger in place (O(unacked), no store copies, no `mirrorState` deep scans). Iteration time at 10 vs 10 000 unacked ≤ 2×. — [auto] [Unacked cost] Q: how does event-loop cost stop growing with the unacked set? → Selected: per-peer unacked ledger keyed (key,version) (recommended default: Q-02's stated remedy). — **Reversibility:** costly — touches core loop + adapter mirror.
- **D-04:** "Record acknowledged by peer X" travels as an **`Outbox` result** from the core; the adapter forwards it to waiters (`wait_for_replication`) — removing the 5 ms polling and snapshot diffing in `wait.go`. This is the same seam Phase 4/5 build on. — [auto] [Ack seam] Q: how do waiters learn about acks? → Selected: Outbox result + waiter channel (recommended default: Q-02/XPORT-12). — **Reversibility:** costly — the wait API contract rides it.
- **D-05:** Retransmits are **paced with jitter** on the existing `retransmit_interval` ladder (per-peer scheduling, no whole-set storm); a peer returning to a node holding 10 000 unacked drains within N / `sync_response_rate` + 10 s with ≤ 3N retransmits; the open-loop 1000 writes/s × 15 s burst over 10 000 distinct keys achieves ≥ 990 writes/s with the set empty 5 s after the end. — [auto] [Retransmit pacing] Q: how are retransmits bounded? → Selected: per-peer jittered pacing on the existing backoff ladder (recommended default: R-02 acceptance). — **Reversibility:** reversible.

#### Input validation and read loop (XPORT-03/04/08/10)
- **D-06:** Received versions are **validated before any clock call**: `ver`/`var` fields sane, `l` within `now + max_clock_lead` (new configurable bound, default 60 s), invalid → discarded with its own class (`bad_version`), WARN-logged rate-limited; the same rule applies to versions inside DataSync. Empty key, tombstone-with-data, unverified checksum get their own discard counters (extending the Phase-2 negative table). Key-length and `value_max_bytes` ceilings enforced (`InvalidArgument`; exit 2 when the largest record cannot fit a datagram; any accepted record encodes to ≤ 65 507 bytes). — [auto] [Validation] Q: validate before clock or trust the sender? → Selected: validate before clock, own classes (recommended default: R-07/R-16 acceptance). — **Reversibility:** reversible.
- **D-07:** The read loop **exits only on `net.ErrClosed`**; any other read error is logged and counted (or the process exits deliberately) — a node never keeps sending `Hello` while deaf; a node that received no Hello for `peer_timeout` after start WARNs. Bind-address misuse is loud: non-wildcard `--bind-addr` warns or is rejected; second instance fails with "address already in use". — [auto] [Read loop] Q: what may terminate the read loop? → Selected: only net.ErrClosed; everything else counted/logged (recommended default: R-14/R-15 acceptance). — **Reversibility:** reversible.

#### Seams and cleanup (XPORT-05/06/07/11/13)
- **D-08:** The store gains a **cheap key→version view with tombstone visibility** + point reads (shallow iteration under RLock, no deep copies); DataRequest replies and sync-chunk handling use it — making them independent of foreign record reconstruction. A **bounded, deduplicated sync-answer queue** (keyed by (key, version), capacity-capped) ensures 1000 missing keys answered twice produce 1000 answers, not 2000. Dead code from Q-09 (`lastLocal`/`cover`, exported `persist.Parse`, `ownHandle`, `fsync_darwin.go`) is **deleted or justified in place** (fsync_darwin.go removal coordinates with Phase 5's P-09 — if Phase 5 intends to rework it, delete here and note the Phase-5 follow-up). — [auto] [Seams] Q: key→version view + queue + dead code? → Selected: shallow view + dedupe queue + delete-with-justification (recommended default: Q-01/R-11/Q-09 acceptance). — **Reversibility:** costly — the store API is consumed across phases.

### Claude's Discretion
- Exact pacer implementation (token bucket vs timer window), jitter distribution, `max_clock_lead` default (60 s unless research finds a better-grounded value), queue capacities, rotation persistence details.

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

## Summary

Phase 3 turns the two red transport scenarios green (R-01 paced sync, R-02 unacked/retransmit cost + burst) by fixing the four structural causes the v1.0 review measured: (1) the DataSync listing is emitted as one unpaced burst and the same sorted tail is lost every round; (2) the adapter re-copies and re-sorts the whole pending set after **every** event loop iteration while the retransmit scanner walks all pending entries per tick and the discovery flush emits the entire pending set in one iteration; (3) the store exposes only a whole-map deep copy (`Records()`), so every `DataRequest`, every sync chunk and every listing copies the world; (4) "record acknowledged by peer X" never leaves the core as a result, so `wait.go` polls a snapshot mirror every 5 ms.

The current code was read end to end this session; line numbers below are from the current tree (they differ from the review's citations, which predate Phase 2). [VERIFIED: repo-wide reads of `internal/app/repl/core.go`, `internal/udprepl/repl.go`, `internal/app/storage/store.go`, `internal/app/persist/engine.go`, `internal/grpcapi/wait.go`] No new dependencies are needed: everything is in the Go standard library (`net`, `syscall`, `container/heap`, `time`) plus the already-present protobuf module for one size calculation.

**Primary recommendation:** implement the phase as a set of seam-preserving changes in this order — (1) store read view (`Version`/`Lookup`/`Versions`) and the core's switch to it; (2) record-free per-peer unacked ledger with a due-heap and O(1) gauges; (3) Outbox result events (`Ack`/`Drained`/`Superseded`/`WaitSetEmpty`) plus an adapter tracker replacing `PendingCopy` and the 5 ms poll; (4) two paced send queues (listing, record) at `sync_response_rate` with deterministic jitter + rotating listing start; (5) socket buffers + bind/reuse policy + read-loop hardening; (6) core-side wire validation with its own discard classes; (7) dead-code removal; then flip the three `scripts/red-expectations.tsv` rows and add the 20-run R-01 evidence job. Keep the red test files untouched.

**Locked-decision interpretations the planner needs (the only places this research reads a locked decision through `Claude's Discretion`):**

- **D-01 "token-bucket pacer … per-send pacing, not per-chunk burst":** the recommended form is the discretion-allowed timer-window/spacing limiter already proven by `drainEgress` — at most one datagram per `floor(1s/rate)` when late, no burst capacity, so the `N / sync_response_rate` arithmetic is exact. A token bucket with capacity is the alternative if timer-wakeup cost ever matters.
- **D-02 "double read-back":** read back **both** `SO_RCVBUF` and `SO_SNDBUF` after setting them; because the kernel doubles each value (man7), the log and the fail-fast compare the **returned** values, never the requested ones.
- **D-05 "per-peer scheduling … paced with jitter":** realized as the per-peer ledger + per-entry ladders (which preserve the documented `loss ≤ ack_timeout_max` bound) + a global record pacer whose spacing is jittered. Jittering the ladder deadlines themselves would delay the first retry past `ack_timeout_max` and break `TestBound_lossWithinAckTimeoutMax`.
- **D-08 "queue keyed by (key, version)":** correct for `DataReplication` replies; `DataRequest` items carry no version, and the destination handle is mandatory (the review's RPL-11 fix says `(handle, key, kind)`). The recommended identity is `(handle, kind, key)` with the record re-read at send time; this still yields "two listings with the same 1 000 missing keys leave 1 000 answers".

## What the phase must not break

- The three red tests must stay byte-untouched and flip red→green only through the TSV (CONTEXT: "no test edits needed"). They assert public behavior (gRPC, `/metrics`, SIGTERM), so the internal redesign is free — but `scripts/red-runner`'s **mutation stage** sed patterns target exact `core.go` source lines and will break (see Pitfall 1).
- The deterministic simulator must still replay same-seed identically: no `math/rand`/crypto randomness in the core, no wall clock, no map-iteration emission order.
- The depguard boundary: `internal/app/**` may not import `net`, `os`, `syscall` or `google.golang.org/*`; `internal/port` allows stdlib + the own module only. All new socket code stays in `internal/udprepl`; all validation decisions stay in `internal/app/repl`/`version`.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|---|---|---|---|
| Listing/retransmit pacing schedule, rotation, jitter | `internal/app/repl` (pure core) | `internal/udprepl` (socket + timer wakeups) | The core owns every timer and every emission decision (D-04 of v1.0); the adapter only sleeps and sends. |
| Socket buffer sizing, bind, read loop | `internal/udprepl` (adapter) | `cmd/sup-kv` (bind-address WARN, startup exit 2) | Only the adapter may touch `net`/`syscall`; config validation is `main`'s until Phase 7 extracts it. |
| Key→version view, point reads, tombstone visibility | `internal/app/storage` | `internal/app/persist` (Engine forwards) | The store is the single owner of record state; the Engine is a facade. |
| Received-version/key/checksum validation + counters | `internal/app/repl` | `internal/app/version` (layout/bound helper) | Decisions must run in the simulator too; protobuf stays adapter-side. |
| "Acked by peer X" result → waiters | `internal/app/repl` (emit results) | `internal/udprepl` (tracker) + `internal/grpcapi` (thin wait) | Results-not-callbacks is the core contract; waiting is an adapter-side goroutine concern. |
| New counters/families | `internal/app/repl` + `internal/udprepl` (atomics) | `internal/metrics` (rendering) | The exporter is a pure formatter over accessors; no registry. |
| New config knob (`max_clock_lead`) | `cmd/sup-kv` (flag/env) | `internal/app/repl` (Config + normalization) | Same flag/env twin pattern; Phase 7 later centralizes validation. |

## Standard Stack

No external packages are added in this phase. The relevant APIs were verified against the local Go 1.27.4 toolchain this session.

### Core
| Library/API | Version | Purpose | Why Standard |
|---|---|---|---|
| `net.UDPConn.SetReadBuffer` / `SetWriteBuffer` | Go 1.27.4 stdlib | Set `SO_RCVBUF`/`SO_SNDBUF` after bind | [VERIFIED: `go doc net.UDPConn.SetReadBuffer`] "SetReadBuffer sets the size of the operating system's receive buffer associated with the connection." |
| `net.UDPConn.SyscallConn` + `syscall.GetsockoptInt` | stdlib | Read the *effective* buffer sizes back | [VERIFIED: `$(go env GOROOT)/src/net/udpsock.go:125`] `func (c *UDPConn) SyscallConn() (syscall.RawConn, error)`; [VERIFIED: `src/syscall/syscall_unix.go:312`] `func GetsockoptInt(fd, level, opt int) (value int, err error)` under `//go:build unix` (includes darwin). |
| `net.ErrClosed` | stdlib | The only read-loop exit | [VERIFIED: `go doc net.ErrClosed`] "This may be wrapped in another error, and should normally be tested using errors.Is(err, net.ErrClosed)." |
| `container/heap` | stdlib | Due-time priority queue for retransmits | [VERIFIED: `internal/app/internal/simnet/simnet.go:80-120`] the simulator's deterministic `(at, seq)` `eventHeap` is the same shape. |
| `time.Timer` with `Reset`/`Stop`, no drain | stdlib (Go 1.23+ semantics) | Driver wakeups | [VERIFIED: `internal/udprepl/repl.go:668-696`] existing `armTimer`/`stopTimer` pattern. |
| `google.golang.org/protobuf/proto` | already in `go.mod` | One-shot "does the largest accepted record fit a datagram" check + the size property test | [VERIFIED: `internal/udprepl/repl.go:702-723`] the adapter already encodes/marshals with it; no new module. |

### Supporting
| API | Purpose | When to Use |
|---|---|---|
| `syscall.SetsockoptInt(SOL_SOCKET, SO_REUSEADDR, 1)` via `ListenConfig.Control` | Same-host port sharing | **Test-only**: an explicit adapter option used by `internal/udprepl/smoke_test.go`; never the production default (see Pitfall 12). |
| `slices.Sorted`/`maps.Keys` | deterministic listing order | Already the repo's only iteration-to-emit idiom. |

**Installation:** none. No `go.mod` change is required (verify with `go mod tidy && git diff --exit-code go.mod go.sum` at the phase gate).

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|---|---|---|
| Strict per-send spacing at `floor(1s/rate)` | Token bucket with capacity | A token bucket permits small bursts (fewer timer wakeups) but changes the `N/rate` convergence arithmetic; the existing egress limiter already proves the spacing style and its integer arithmetic is pinned by tests. |
| Custom indexed min-heap | `container/heap` with lazy stale entries | Lazy stale entries leak one heap item per re-arm (≈ retry count). Use a small custom heap that tracks each entry's index and supports `Remove`/`Fix`; ~60 lines, deterministic. |
| Adapter tracker created at notify (`track bool`) | Adapter polls a mirror / lazy tracker + orphan-ack ledger | Polling is what the phase removes; the orphan ledger needs retention and can under-report. The `track` flag is exact and costs one internal signature change. |
| `Records()` deep copy | `Versions()` + `Lookup()` + `Version()` | The deep copy is O(store) per datagram and was measured at 2.3 ms / 10.9 MiB; the view is O(1)/O(chunk). |
| Removing `SO_REUSEADDR` outright | Detect a second instance some other way | Not detectable: with `SO_REUSEADDR` the second bind succeeds and the kernel splits unicast traffic. Removal is the only honest fix. |

**Version verification:** `go version` → `go1.27.4 linux/amd64`; `go.mod` pins `go 1.27`; `go tool golangci-lint --version` → `2.15.0 built with go1.27.4`. `GOOS=darwin GOARCH=arm64 go build ./...` passes on the current tree.

## Package Legitimacy Audit

**No external packages are installed in this phase.** The audit is not applicable: no npm/PyPI/crates/Go module additions, no `go get`, no tool additions. The existing Go module graph is unchanged; the gate `go mod tidy && git diff --exit-code go.mod go.sum` is the phase check.

## Project Constraints (from CLAUDE.md / AGENTS.md)

Actionable directives the planner must honor:

1. **Go version** stays as pinned in `go.mod` (`go 1.27`); no raise without an ADR.
2. **Architecture boundary is lint-enforced** (`.golangci.yml`, project-additive keys inside `linters.settings.depguard.rules`):
   - `sup-kv-app-no-io`: `**/internal/app/**` may not import `net` (prefix match catches `net/http`), `os`, `syscall`, `google.golang.org/`. Exempts `$test`.
   - `sup-kv-port-stdlib-only`: `**/internal/port/**` may import `$gostd` + `github.com/Djarvur/sup-kv` only.
3. **No lint suppressions** without a specific linter name and a human-readable reason (`nolintlint.require-explanation`, `require-specific`). New `//nolint` requires justification in the plan.
4. **Vendored lint config is byte-unmodified** for golangci-lint exactly 2.15.0; project rules are additive keys only. Any boundary change is a visible `.golangci.yml` diff.
5. **Zero direct runtime requires** for Phase 1 remains the dependency-policy baseline: gRPC/protobuf are already present; do not add modules.
6. `make lint` and `make test` (`go test -race ./...`) are the pre-push gates; CI runs the same. CI matrix: `ubuntu-latest`, `ubuntu-24.04-arm`, `macos-15`, plus a darwin/amd64 cross-compile cell (`go build ./... && go vet ./...`).
7. Tests are vanilla `testing`, table-driven; no assertion libraries. Test-only infrastructure lives in `*_test.go` or the `internal/stand` / `internal/testutil` packages.
8. Product constraints: whole state in RAM; values ≤ 60 KiB by default; trusted LAN; CRC32 is integrity only (no auth); convergence must not depend on wall-clock sync (this phase adds a *rejection* bound, not a sync protocol).
9. darwin is development-only (ADR 0010); the ~9 KiB send limit is documented, not fixed.

<phase_requirements>
## Phase Requirements

| ID | Description (from REQUIREMENTS.md) | Research Support |
|----|-----------------------------------|------------------|
| XPORT-01 | Sync listing paced and robust to buffer loss; 20 000 keys × 1 KiB converge ≤ 2×sync_interval + N/sync_response_rate, 20/20; start chunk rotates; buffers set+logged; cheap key→version view | Listing pacer + rotation design (§Patterns 1), buffers (§Pattern 7), store view (§Pattern 4); red test `TestRed_R01_pacedSync20k` (`internal/stand/stand_red_test.go:276-364`); 20× evidence job (§Validation). |
| XPORT-02 | Unacked cost flat (10 vs 10 000 ≤ 2×); 10 000-unacked drain within N/rate + 10 s, ≤ 3N retransmits; 1000 writes/s × 15 s ≥ 990/s, empty 5 s after | Record-free per-peer ledger + due-heap + O(1) gauges (§Pattern 3); paced record queue with ladder re-arm (§Pattern 2); red tests `TestRed_R02_unackedDrain`, `TestRed_R02_burst` (`:381-450`, `:470-576`). |
| XPORT-03 | Key length and `value_max_bytes` ceilings; `InvalidArgument`; exit 2 when largest record cannot fit; accepted record ≤ 65 507 B | `storage.MaxKeyBytes` single constant, API + receive checks, startup check through the real codec (§Pattern 6); measured overhead 52–61 B (§Sizing facts). |
| XPORT-04 | Received versions validated before any clock call; `l ≤ now + configurable bound`; own class + WARN; same inside DataSync | `version.RecordVersion.Validate` + core validation pipeline (§Pattern 6); `repl.Config.MaxClockLead`; new `bad_version` class. |
| XPORT-05 | Socket buffers configured and reported; startup fails if send buffer < max datagram; 1/20/60 KiB deliver on Linux; darwin documented | §Pattern 7 (kernel doubling/clamping verified); stand test for 1/20/60 KiB; ADR 0010 for darwin. |
| XPORT-06 | DataRequest/sync-chunk handling independent of foreign records; 1 000 vs 50 000 keys within 2× | `Version(key)` per listing entry and `Lookup(key)` for replies (§Pattern 4); benchmarks. |
| XPORT-07 | Sync-answer queue bounded and deduplicated; 2×1000 keys → 1000 queued | §Pattern 9 (identity `(handle, kind, key)`, cap, drop-oldest). |
| XPORT-08 | Read loop exits only on `net.ErrClosed`; other errors logged+counted | §Pattern 8 + injectable `packetConn` seam. |
| XPORT-09 | Non-wildcard bind rejected/WARNed; second instance EADDRINUSE; no-Hello WARN; timers jitter | §Pattern 10 (REUSEADDR removal verified), peer-timeout WARN, deterministic jitter (Open Question O3). |
| XPORT-10 | Empty key, tombstone-with-data, unverified checksum discarded with own counters; CRC coverage documented | §Pattern 6; `internal/udprepl/decode_negative_test.go:190-270` rows flip; CRC-coverage doc is DOCS-06 (Phase 9). |
| XPORT-11 | Store exposes key→version view and point reads with tombstone visibility | §Pattern 4 (`Store.Version`/`Lookup`/`Versions`, Engine forwarding, tests). |
| XPORT-12 | "Record acked by peer X" as an `Outbox` result to adapter and waiters; removes snapshot diffing and 5 ms polling | §Pattern 5 (result kinds, adapter tracker, thin `wait.go`). |
| XPORT-13 | `lastLocal`/`cover`, exported `persist.Parse`, `ownHandle`, `fsync_darwin.go` gone or justified | §Pattern 11 (all four verified; exact refs below). |
</phase_requirements>

## Architecture Patterns

### System Architecture Diagram

```
             UDP socket (SO_BROADCAST default; SNDBUF/RCVBUF set + read back + logged)
                      │
        read goroutine │ ReadFromUDP into one 65 507 B buffer
                      │   net.ErrClosed ──► exit
                      │   other error ──► count ++ / rate-limited WARN / brief backoff ──► continue
                      ▼
              inbound chan (64) ──► EVENT LOOP (single goroutine, Run)
                                        │
                     decode() ──► InboundDatagram {Status, envelope, Msg}
                                        │
                     core.HandleDatagram(now, in, out)
                        ├─ framing/proto/own-echo/dup-node-id classes (unchanged)
                        ├─ NEW wire validation (before any store/clock touch)
                        │    empty_key | key_too_long | tombstone_with_data |
                        │    bad_checksum | bad_version   → OnDiscard + counters
                        ├─ Hello        → peer table; discovery flush → record queue
                        ├─ DataReplication → ApplyRemote → out.Ack(msg) + outbox Ack result
                        ├─ Ack          → ledger drain → Ack / Drained results
                        ├─ DataRequest  → store.Lookup(key) → egress queue
                        └─ DataSync     → per-entry store.Version(key) → egress queue
                                        │
                     core.Advance(now, out)   (fires due timers, returns next deadline)
                        ├─ hello/sync timers (deterministic jitter; startup immediate)
                        ├─ syncListing: store.Versions() → chunk → rotate start → LISTING PACER
                        ├─ evictPeers: per-peer ledger removal, O(peer unacked)
                        ├─ due-heap retransmits → RECORD PACER (first send arms the ladder)
                        ├─ drain listing pacer        (≤1 per interval when late)
                        ├─ drain record pacer         (≤1 per interval when late)
                        └─ drain egress pacer         (existing)
                                        │
                     flush(out): encode + WriteToUDP (broadcast/unicast) + SentByType
                     drainResults(out): adapter tracker map → waiter channels (WaitReplicated)
                     mirrorStats(): O(1) atomics (UnackedStats O(1), PeerCount O(1), counters)
                                        │
                                  /metrics (in-repo text exporter)
```

### Recommended Project Structure (files this phase touches)

```
internal/app/version/version.go        # + Validate(now, maxLead)
internal/app/repl/core.go              # ledger, heap, pacers, validation, results, rotation, WARN
internal/app/repl/bench_test.go        # NEW: event-loop / DataRequest / DataSync benchmarks
internal/app/storage/store.go          # + Version / Lookup / Versions / MaxKeyBytes
internal/app/persist/engine.go         # forward the three reads; remove lastLocal/cover; remove Parse
internal/app/persist/codec.go          # delete Parse
internal/persistfile/fsops.go          # syncDurable → File.Sync
internal/persistfile/fsync_darwin.go   # DELETE
internal/persistfile/fsync_other.go    # DELETE (File.Sync is portable, incl. darwin F_FULLFSYNC)
internal/port/repl.go                  # + result events on Outbox
internal/udprepl/repl.go               # buffers, conn seam, read loop, trackers, options; -mirror
internal/metrics/metrics.go            # new families (Source additions)
internal/grpcapi/wait.go               # thin wait over the new seam
internal/grpcapi/gate.go               # ServerOptions: PendingCopy → WaitReplicated
internal/grpcapi/server.go             # key-length check; notify carries `track`
cmd/sup-kv/main.go                     # max-clock-lead flag/env, bind WARN, startup size check
internal/app/internal/simnet/simnet.go # nodeStore new methods; Cluster.Write error+Lookup (WR-02)
scripts/red-runner                     # mutation sed re-point (NOT the expectation flip)
scripts/red-expectations.tsv           # R-01/R-02 rows red → green
```

### Pattern 1: Paced listing with rotating start chunk

**What:** `syncListing` / restart listings build chunks from a cheap `Versions()` snapshot and append them to a **listing queue**, drained at `floor(1s/SyncResponseRate)` per datagram. Exactly one in-flight listing per destination (broadcast or peer handle); a trigger that arrives while one is queued is dropped and `nextSync` still advances (otherwise a short `sync_interval` livelocks). The broadcast round's `start` chunk rotates by a monotone round counter; a targeted (restart) listing rotates a per-peer counter stored in the peer row. **"Persisted per peer" means kept in core state across rounds (the peer row), not written to disk**; a peer restart re-creates the row and resets its rotation, which is harmless (the broadcast round counter keeps covering the space). `Versions()` carries no `Deleted` flag, so the listing is tombstone-inclusive by contract; the red-runner's "sync without tombstones" mutant must therefore be re-pointed at the store's `Versions` (Pitfall 1).

**Why:** the review measured systematic loss of the sorted tail because every round sends the same chunk order into a buffer that overflows. Rotation guarantees that any fixed tail is at the head next round; pacing removes the overflow itself.

**Verified anchors:** current burst emission [VERIFIED: `internal/app/repl/core.go:751-759`] `for _, chunk := range c.listingChunks() { out.Broadcast(port.DataSync{Entries: chunk}) }`; the existing spacing arithmetic [VERIFIED: `core.go:342`] `syncResponseInterval: time.Second / time.Duration(cfg.SyncResponseRate)` and [VERIFIED: `core.go:668-678`] `drainEgress`; chunk builder [VERIFIED: `core.go:771-778`] `listingChunks`.

**Sketch (recommended shape, not yet in repo):**
```go
// exactly one in-flight broadcast listing; skip (but advance nextSync) if busy
func (c *Core) syncListing(now time.Time, out *port.Outbox) {
    if !c.nextSync.IsZero() && now.Before(c.nextSync) { return }
    c.nextSync = now.Add(c.cfg.SyncInterval)   // advance even when the enqueue is skipped
    if c.listingBusy { return }
    chunks := c.listingChunks()                // Versions() snapshot → entries → chunks
    start := c.syncRotation % len(chunks)
    for i := range chunks {
        c.enqueueListing(now, "", chunks[(start+i)%len(chunks)]) // dst=="" means broadcast
    }
    c.syncRotation++
    c.listingBusy = true
}
```
Note the first chunk of an idle queue emits in the same `Advance` (`nextSendAt = now` clamp), which keeps `TestSync_triggers`' startup expectation for a one-chunk store.

### Pattern 2: Record-send pacer + ladder re-arm

**What:** One FIFO of `(key, version)` record sends, deduplicated by `(key, version)`, paced at `sync_response_rate`; fed by the discovery flush and by the retransmit scanner. At emission the core resolves the record with `Lookup` (skip if the pending entry is gone), emits one broadcast, and **arms that entry's ladder from the emission time** (`nextRetry = now + backoff`), not from the enqueue time. The ladder deadline is never jittered earlier/later (positive ladder jitter would violate "loss ≤ ack_timeout_max"); jitter belongs to pacer spacing and the hello/sync timers.

**Why:** the discovery flush currently broadcasts 10 000 datagrams inside one `handleHello` call ([VERIFIED: `core.go:499-506`] `for _, key := range slices.Sorted(maps.Keys(c.pending)) { ... out.Broadcast(port.DataReplication{...}) }`) and every ladder step rebroadcasts the whole set ([VERIFIED: `core.go:872-887`]); the receiver's Acks overflow and the set never shrinks. Arming ladders at enqueue would make every backed-up entry overdue the moment the pacer reaches it — a second storm.

**Sketch:**
```go
type recordSend struct{ key string; ver version.RecordVersion; retry bool }

func (c *Core) drainRecordSends(now time.Time, out *port.Outbox) {
    for len(c.recordQueue) > 0 && !now.Before(c.nextRecordAt) {
        rs := c.recordQueue[0]; c.recordQueue = c.recordQueue[1:]
        e, ok := c.pending[pendingKey{key: rs.key, version: rs.ver}]
        if !ok { continue }                       // drained/superseded after enqueue
        rec, ok := c.store.Lookup(rs.key)
        if !ok || rec.Version != rs.ver { continue }
        out.Broadcast(port.DataReplication{Key: rs.key, Record: rec})
        if rs.retry { c.retransmitted++ }
        e.nextRetry = now.Add(e.backoff)          // ladder starts at the real send
        c.pending[e.pk] = e
        c.reheap(e)
        if now.After(c.nextRecordAt) { c.nextRecordAt = now }
        c.nextRecordAt = c.nextRecordAt.Add(c.pacedInterval())
    }
}
```

### Pattern 3: Record-free per-peer unacked ledger + due-heap + O(1) gauges

**What:** `pending` becomes `map[pendingKey]pendingEntry` where `pendingKey{key string, version version.RecordVersion}` and the entry holds **no record Data** — only version, `waitSet map[uint16]struct{}`, `acked []uint16`, `nextRetry`, `backoff`, `createdAt`, and a list element. Add `peerUnacked map[uint16]map[pendingKey]struct{}` (the per-peer ledger D-03 locks) so peer eviction is O(that peer's unacked) and no whole-pending scan. Add a custom indexed min-heap of `(nextRetry, seq, pendingKey)` so `Advance` touches only due entries and `nextDeadline` is O(1). Keep `PendingSnapshot`/`PendingInfo` (tests depend on it) but stop calling it per event from the adapter. Replace the adapter's `PeerSnapshot()`-for-length with a new `PeerCount()`; make `UnackedStats` O(1) with an insertion-ordered list (`count = len(pending)`, `oldest = front.createdAt`; deletion removes the list node) — the current `UnackedStats` walks all pending (`core.go:1059-1073`) and the adapter calls it every event (`internal/udprepl/repl.go:575-577`).

**Why:** the review's CPU profile was 52% `mirrorState` + 20% `scanRetransmit`; the mirror is removed by Pattern 5, the scan by the heap. Removing the retained record also halves pending memory (review RPL-08/Q-02) and lets `Lookup` be the single record source.

**Verified anchors:** [VERIFIED: `core.go:265-273`] `pendingEntry` currently retains `rec storage.Record`; [VERIFIED: `core.go:872-887`] `scanRetransmit` sorts and walks all keys; [VERIFIED: `core.go:831-845`] `evictPeers` loops all pending per expired peer; [VERIFIED: `core.go:999-1005`] `PeerSnapshot` allocates a map per event call.

### Pattern 4: Cheap store read view (tombstone-visible)

**What:** add to `storage.Store` (and forward from `persist.Engine`):
```go
func (s *Store) Version(key string) (version.RecordVersion, bool)         // RLock, no Data copy
func (s *Store) Lookup(key string) (Record, bool)                         // RLock, one record incl. tombstone
func (s *Store) Versions() map[string]version.RecordVersion               // RLock, shallow key→version
```
Then shrink `repl.StateStore` to `ApplyRemote`, `Version`, `Lookup`, `Versions` (drop `Records`). `handleDataRequest` uses `Lookup`; `handleDataSync` uses `Version` per entry (never `Versions`, which would be O(store) per chunk) and `Lookup` only for the strictly-newer reply; `listingChunks` uses `Versions`.

**Why:** [VERIFIED: `internal/app/storage/store.go:274-287`] `Records()` deep-copies every `Data` (`out[key] = Record{... Data: bytes.Clone(rec.Data) ...}`), measured at 2.3 ms / 10.9 MiB per datagram at 10 000 × 1 KiB (review RPL-08). [VERIFIED: `store.go:141-153`] `Get` hides tombstones (`if !ok || rec.Deleted { return Record{}, ErrNotFound }`), so it cannot serve reconciliation; `Lookup` is the tombstone-visible point read XPORT-11 asks for. Per-entry `Version` keeps `handleDataSync` O(chunk), which is what XPORT-06's 1 000 vs 50 000 benchmark measures.

**Note:** `Engine.Get` stays live-only for the API; `Engine.Lookup` is the new tombstone-visible method.

### Pattern 5: Outbox result events + adapter tracker (XPORT-12)

**What:** `port.Outbox` gains a result stream (separate from `Drain()` so no existing test changes meaning):
```go
type Outbox struct { outbound []Outbound; results []Result }
func (o *Outbox) Ack(key string, ver version.RecordVersion, peer uint16)
func (o *Outbox) Drained(key string, ver version.RecordVersion)
func (o *Outbox) Superseded(key string, ver version.RecordVersion)
func (o *Outbox) WaitSetEmpty(key string, ver version.RecordVersion)
func (o *Outbox) DrainResults() []Result
```
The core emits: `Ack` in `handleAck` for every matching Ack (including strangers — the WR-01 counting behavior), `Drained` when the wait-set empties and peers remain (the deletion point), `Superseded` from `applyRemote`'s strictly-greater branch ([VERIFIED: `core.go:539-541`] `if entry, ok := c.pending[msg.Key]; ok && msg.Record.Version.Compare(entry.version) > 0 { delete(c.pending, msg.Key) }`) and from local replacement in `NotifyLocalWrite`, `WaitSetEmpty` when eviction removes the last waiter but the entry stays suspended ([VERIFIED: `core.go:831-845`]). The adapter keeps `map[trackerKey]tracker` (mutex-guarded), creates a tracker **before enqueueing the notification** when the write requested `wait_for_replication` (change `LocalNotify` to `func(key string, rec storage.Record, track bool) error`), applies results under the lock, and exposes:
```go
func (a *Adapter) WaitReplicated(ctx context.Context, key string, ver version.RecordVersion,
    total uint32, timeout time.Duration) (acked bool, ackedCount uint32)
```
Success is `resolvedOK && min(len(acked), total) == total`; `Superseded`/`WaitSetEmpty` resolve false; timeout returns the count. This is behavior-identical to today's `waitForDrain` (`internal/grpcapi/wait.go:144-196`) without the 5 ms poll or the 250 ms `drainedRetention` machinery.

**Why:** D-04 locks "results, never callbacks"; the current mirror clones/sorts the whole pending set after every event ([VERIFIED: `internal/udprepl/repl.go:594-666`] `mirrorState`/`retainDrained`) and `wait.go` polls it at 5 ms with a 500 ms appearance phase ([VERIFIED: `internal/grpcapi/wait.go:34-53`] `waitPoll = 5 * time.Millisecond`, `waitAppear = 500 * time.Millisecond`; `:94-131`). Deleting the mirror removes the measured 52% CPU site; the tracker-at-notify removes the appearance race the CR-01 retention worked around.

**Pitfall:** `Run` reuses one `port.Outbox` across iterations; `DrainResults` **must** be called after every handler (and in `simnet`'s `deliver`/`drive`) or results accumulate for the process lifetime.

### Pattern 6: Wire validation pipeline (XPORT-03/04/10)

Where: in the core, after the existing framing/proto/own-echo/duplicate-node-id classification and **before** `handleMessage`, so the simulator exercises the exact production path and no store/clock call can precede the verdict. New tokens and counters:

```go
const (
    DiscardEmptyKey          = "empty_key"
    DiscardKeyTooLong        = "key_too_long"
    DiscardTombstoneWithData = "tombstone_with_data"
    DiscardBadChecksum       = "bad_checksum"
    DiscardBadVersion        = "bad_version"
)
```

Pinned order per message (tests will pin it): DataReplication → empty key, key too long, tombstone-with-data, checksum (`storage.RecordChecksum(key, ver, data, deleted) != rec.Checksum`), version; DataRequest/Ack → empty key, key too long; DataSync → every entry: empty key, key too long, version (one invalid entry discards the datagram); Hello → nothing. Details passed to `OnDiscard` must **not** include key names (the T-6-05 metadata contract).

Version validation comes from a pure helper in `internal/app/version`:
```go
func (v RecordVersion) Validate(now time.Time, maxLead time.Duration) error
// layout: v[6]&0xF0 == 0x70 (UUIDv7), v[8]&0xC0 == 0x80 (variant 10)
// future: nowMs := now.UnixMilli(); if l >= nowMs && l-nowMs > maxLeadMs → future
```
The subtraction form avoids overflow for absurd `maxLead`. `repl.Config` gains `MaxClockLead time.Duration` normalized to 60 s (the `drift_warn` default, so peers inside the documented drift tolerance stay accepted). What the review's poison scenario gets: a `ff…ff` or `now+1h` version is rejected before `ApplyRemote`, so `clock.Update` never sees it (a counting-clock fake proves "no clock call" in the test).

**Sizing facts** [VERIFIED: local probes marshaling the generated schema this session]: a `DataReplication` datagram overhead is **52–61 B** with zero-valued envelope fields and **66–75 B** at worst-case varint widths (max `proto_version`, `node_id`, `crc32` and checksum fields); the review's 68 B sits in that range. With key 1 024 + value 61 440 → **62 539 B** (fits); key 1 024 + value 64 408 → **65 507 B** (exactly the ceiling); key 1 024 + value 64 415 → **65 514 B** (over); key 4 096 + value 61 440 → **65 611 B** (over, matching the review's 65 605 modulo varint widths). Hence with a 1 024-byte key limit the largest accepted value is **64 408 B**, and the default 61 440 passes with ~3 KiB slack.

**Verified anchors:** envelope `recordVersion` accepts any 16 bytes ([VERIFIED: `internal/udprepl/repl.go:843-850`] `if len(b) != len(ver) { return version.RecordVersion{}, false }` and nothing else); the only `Clock.Update` is `storage.Merge` ([VERIFIED: `store.go:226-228`] `remoteL, remoteC, _ := rec.Version.Unpack(); _, _, drift = s.clock.Update(remoteL, remoteC)`); `applyRemote` is [VERIFIED: `core.go:523-543`].

### Pattern 7: Socket buffers — set, read back, log, fail-fast

Verified kernel semantics ([CITED: man7.org/linux/man-pages/man7/socket.7.html], fetched this session): for both `SO_RCVBUF` and `SO_SNDBUF` "the kernel doubles this value" and "this doubled value is returned by getsockopt(2)"; the ceiling is `rmem_max`/`wmem_max`. [VERIFIED: local probe on this Linux host] (`rmem_max=4194304`, `wmem_max=4194304`): requesting 1 MiB → getsockopt 2 097 152; requesting 4 MiB → 8 388 608; requesting 65 507 → 131 014. So the "actual" number in the log is **twice what was requested** (or twice the clamp), which is exactly why the phase must read it back instead of logging the request.

Recommended: `SetReadBuffer(4<<20)` and `SetWriteBuffer(1<<20)` after bind, then `SyscallConn().Control` + `syscall.GetsockoptInt(SOL_SOCKET, SO_RCVBUF/SO_SNDBUF)`, `slog.Info("Socket buffers", "recv_buf", rcv, "send_buf", snd)`, and a small `SocketBuffers() (recv, send int)` accessor for unit tests; fail `New` when the actual send buffer `< datagramBufSize` (65 507) or any set/read-back errors. Also verified in the Go 1.27 toolchain: `net` already sets `SO_BROADCAST=1` for every datagram socket ([VERIFIED: `$(go env GOROOT)/src/net/sockopt_linux.go:21`] `syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)`; same at `sockopt_bsd.go:36`), so the current `Control` is redundant for broadcast. darwin's ~9 KiB `net.inet.udp.maxdgram` is unaffected by `SO_SNDBUF`; it is documented (ADR 0010), not fixed, and the Linux 1/20/60 KiB delivery test is Linux-tagged. [ASSUMED: darwin maxdgram value — taken from committed review evidence, not measured here]

### Pattern 8: Read loop that cannot silently die

```go
func (a *Adapter) readLoop() {
    buf := make([]byte, datagramBufSize)
    for {
        n, src, err := a.conn.ReadFromUDP(buf)
        if err != nil {
            if errors.Is(err, net.ErrClosed) { return }        // the only exit
            a.readErrors.Add(1)
            a.logReadError(err)                                // rate-limited
            time.Sleep(readErrorBackoff)                       // never a spin
            continue
        }
        ... // clone, queue, a.done select (unchanged)
    }
}
```
To test this deterministically, introduce a small unexported `packetConn` interface (`ReadFromUDP`, `WriteToUDP`, `Close`) on `Adapter.conn` and an unexported test constructor; `New` keeps the real `*net.UDPConn`. The acceptance's "fake-conn unit test: a non-closed read error is logged and the loop reads the next datagram" maps 1:1.

### Pattern 9: Bounded, deduplicated sync-answer queue

Identity: `(handle, kind, key)` — `kind` separates a `DataRequest` from a `DataReplication` reply; the destination handle is mandatory or one peer's answers collapse into another's (the review's RPL-11 fix wording is exactly "(handle, key, kind)"). CONTEXT D-08's "(key, version)" reads naturally for DataReplication replies; DataRequests carry no version, so the recommended identity keeps the version out of the key and re-reads the record at send time (`Lookup`), which also frees the queue from pinning records. Cap entries (`maxEgressEntries`, recommend 32 768 — the documented 20 000-key scale plus headroom; smaller caps still converge by re-requesting next round but eat the R-01 margin) and drop the **oldest** on overflow, counting the drop. Keep `QueuedResponses()` as the egress length (tests depend on it) and add a total-queue accessor for quiescence/metrics.

**Verified anchor:** [VERIFIED: `core.go:250-253`] `type queuedUnicast struct { handle string; msg port.Message }`; the queue is `egress []queuedUnicast` ([VERIFIED: `core.go:297`]) and popping re-slices, keeping records reachable ([VERIFIED: `core.go:668-678`] `c.egress = c.egress[1:]`).

### Pattern 10: Bind-address and address-reuse policy

Verified in the Go toolchain: `listenDatagram` does **not** call `setDefaultListenerSockopts` ([VERIFIED: `$(go env GOROOT)/src/net/sock_posix.go:178-215`] the only `setDefaultListenerSockopts` caller is `listenStream` at `:149`), so `net.ListenPacket("udp4", …)` does not set `SO_REUSEADDR` by itself. Empirical Linux check this session: a second bind of the same wildcard UDP address fails with `Address already in use` without `SO_REUSEADDR`, and succeeds with it (reproducing the v1.0 defect). [VERIFIED: local Python socket probe] Therefore:
- Production `New` sets no socket options at all (SO_BROADCAST is a Go default) — a second instance fails with EADDRINUSE.
- Add a test-only functional option `udprepl.WithAddressReuse()`; only `internal/udprepl/smoke_test.go` (which binds two adapters to the same wildcard port) needs it.
- `main` WARNs once, after `setupLogging`, for a non-wildcard `--bind-addr` host (`net.SplitHostPort`; wildcard = empty, `0.0.0.0`, `::`), naming the broadcast-reception limitation. Do not reject: a unicast-only deployment remains usable.
- The no-Hello WARN lives in the core: record the first `Advance` time; if no Hello arrived by `firstAdvance + PeerTimeout`, call `cfg.Log` once.

### Pattern 11: Dead-code removal (verified references)

| Target | Current state (opened this session) | Required change |
|---|---|---|
| `lastLocal` / `cover` / "above cover" | [VERIFIED: `internal/app/persist/engine.go:72`] `lastLocal version.RecordVersion`; [VERIFIED: `:457-459`] `if ver.Compare(e.lastLocal) > 0 { e.lastLocal = ver }`; [VERIFIED: `:294`] `cover := e.lastLocal`; [VERIFIED: `:347-368`] `if w.ver.Compare(cover) > 0 { ... e.waiters = append(e.waiters, w); continue }`; [VERIFIED: `engine_internal_test.go:73+`] `TestConfirmWaiters_aboveCoverInvariant` pins the unreachable branch. Proof it is dead: `await` raises `lastLocal` and appends the waiter **in the same critical section**, and every captured record is installed before `Encode(e.store.Records())`, so every captured waiter is covered. | Delete the field, the `cover` parameter, the above-cover branch + its invariant assertion and test. `confirmWaiters(captured)` resolves every captured waiter. |
| exported `persist.Parse` | [VERIFIED: `internal/app/persist/codec.go:158-161`] `func Parse(dump []byte) error { _, _, err := Decode(dump); return err }`; only `codec_test.go` uses it ([VERIFIED: `codec_test.go:357,403,512,520`]); `Open` uses its own Decode closure; [VERIFIED: `engine.go:122`] mentions it in a comment. | Delete `Parse`; tests call `persist.Decode` and assert its error. Fix the `engine.go` comment. |
| `ownHandle` | [VERIFIED: `core.go:285`] `ownHandle string` field, set from `conn.LocalAddr().String()` ([VERIFIED: `core.go:332-337`] `func New(nodeID uint16, runID [16]byte, ownHandle string, ...)` / `ownHandle: ownHandle`); never read anywhere else ([VERIFIED: repo-wide grep returned only these three occurrences]). | Remove field and constructor parameter. There are **12 core `repl.New(...)` call sites** (verified count; the 12 `udprepl.New(...)` adapter calls are a different constructor and are unaffected), mostly tests; update them mechanically. |
| `fsync_darwin.go` | [VERIFIED: `internal/persistfile/fsync_darwin.go:23-30`] `func syncDurable(f *os.File) error { _, _, errno := syscall.Syscall(syscall.SYS_FCNTL, f.Fd(), uintptr(syscall.F_FULLFSYNC), 0) ... }`; `syncDurable` is called twice in [VERIFIED: `fsops.go:97,151`]. Verified in the Go 1.27.4 toolchain that `os.File.Sync` on darwin already issues `F_FULLFSYNC` with `ignoringEINTR` and an `ENOTSUP`→`Fsync` fallback ([VERIFIED: `$(go env GOROOT)/src/internal/poll/fd_fsync_darwin.go:1-35`]), so the hand-rolled helper is strictly worse. | Delete `fsync_darwin.go` **and** `fsync_other.go`; replace `syncDurable(...)` in `fsops.go` with `File.Sync()`; update the `fsops.go` comments and `docs/SPEC-DEVIATIONS.md` S-PERS-03 (it currently says "F_FULLFSYNC on darwin"). This also discharges Phase 5 P-09's alternative. |

### Anti-Patterns to Avoid

- **Pacing by sleeping in the core or adapter send path:** core is I/O-free; the deadline is returned and the adapter sleeps. Sleeps belong only in the read-error backoff.
- **Building a second listing implementation for restart:** reuse `listingChunks`; only the destination and rotation counter differ.
- **Jittering ladder deadlines:** positive jitter breaks the documented `loss ≤ ack_timeout_max` bound (`TestBound_lossWithinAckTimeoutMax`); put jitter on Hello/sync timers and pacer spacing.
- **Keeping `Records()` on `StateStore` "for compatibility":** it re-creates the O(store) trap; the core has no legitimate use after this phase.
- **Mirroring pending state "just for metrics":** the three gauges are O(1) accessors; a full snapshot per event is the R-02 cost.
- **Logging every discard at info:** existing `logDiscard` is Debug by design; new WARNs need a per-class interval.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---|---|---|---|
| Due-time scheduling | A sorted slice re-sorted per tick, or a timing wheel | `container/heap` with index tracking (`Remove`/`Fix`) and a `(time, seq)` tie-break | Deterministic and O(log n); the repo already uses this shape in `simnet`'s scheduler. |
| Socket option read-back | Parsing `/proc/sys/net/...` or assuming the requested value | `SyscallConn().Control` + `syscall.GetsockoptInt` | The kernel doubles and clamps; `/proc` gives limits, not the socket's effective value. |
| Datagram-size ceiling arithmetic | Hand-written per-field varint math | Encode a max-size fixture with `replproto` + `proto.Marshal`/`proto.Size` | Field sizes depend on varint widths; the real codec is the only honest oracle. The measured overhead is 52–61 B, not a constant. |
| Bounded queue | A linked list without a bounds map, or a slice with O(n) dedupe | FIFO slice + `map[identity]struct{}` + identity stored per item | O(1) dedupe/eviction; nil the popped slot or keep only keys. |
| Rate-limited WARN | A global log throttle utility | A tiny per-class `lastWarn time.Time` field in the core (event `now`) | Deterministic in the simulator, bounded cardinality, no dependency. |
| Priority scheduling with randomness | `math/rand` in the core | A deterministic hash of `(runID, counter, salt)` | Same-seed replay must stay byte-identical. |
| Version layout checks | Re-deriving masks in the core | `version.RecordVersion.Validate` beside the packing constants | One source for the RFC 9562 layout; the version package already owns `uuidVerBits`/`uuidVarBits`. |
| Send pacing | `time.Ticker` per queue | `Advance`-driven `nextSendAt` (existing `drainEgress` shape) | The simulator has no real timers; one deadline drives both production and simulation. |

## Runtime State Inventory

> Not a rename/refactor phase: no stored string, key space, node identity or package path changes. The inventory is still answered explicitly because the phase deletes code and adds one knob:

| Category | Items Found | Action Required |
|---|---|---|
| Stored data | None. No key/collection/ID rename; the dump format is untouched. Records above a new key limit can exist only in pre-v1.1 dumps (written by an older binary or a modified file); Phase 3 does not rewrite them. | None in this phase; Phase 5 PERSIST-10 owns load-time accounting. |
| Live service config | None in git-external systems. | None. |
| OS-registered state | None (no systemd/pm2/launchd changes). | None. |
| Secrets/env vars | New env twin `SUPKV_MAX_CLOCK_LEAD` (flag `--max-clock-lead`, default 60s). Existing deployments need no change; the default preserves current acceptance of any 16-byte version. | Document in `docs/OPERATIONS.md`/Phase 9 DOCS-02; no secret rotation. |
| Build artifacts | The darwin build loses `internal/persistfile/fsync_darwin.go`; `fsync_other.go` is replaced by plain `File.Sync` (the platform fsync path already uses F_FULLFSYNC on darwin). No package/import path changes. | `GOOS=darwin go build ./...` + CI cross cell; update `docs/SPEC-DEVIATIONS.md` S-PERS-03. |

## Common Pitfalls

1. **The red-runner mutation stage breaks on the `core.go` rewrite.** [VERIFIED: `scripts/red-runner`] the script mutates two exact source shapes: `entries = append(entries, port.SyncEntry{Key: key, Version: snapshot[key].Version})` (sync-without-tombstones) and the `syncListing` timer gate (periodic-sync-disabled), and it errors `sed edit was a NO-OP` when the target no longer matches. A rewritten `listingChunks` using `Versions()` has no `Deleted` flag to test, so that mutant cannot be expressed as a sed on `core.go`. Re-point the tombstone mutant at `internal/app/storage/store.go` (`Versions` skipping tombstones) and update the periodic-sync sed; generalize `MUTANT_SOURCE`/overlay to a per-mutant source; re-verify the committed kill seeds. The no-op failure is loud, so this cannot silently pass.
2. **`-count=20` R-01 evidence is not free.** The acceptance is 20/20 at 20 000 keys with default timers; one nightly run is ~70–90 s plus seeding, so the 20× job costs 25–40 min. Put it in a dedicated scheduled job with its own timeout rather than stretching `red-run` (60 min) — or record a local-scale 20× as the PR-adjacent evidence and the nightly 20× as the release gate. Do not edit the red test to loop.
3. **Coalescing listings must still advance `nextSync`.** If a skip leaves `nextSync` in the past, every `Advance` re-triggers, and with `sync_interval=200ms` in the stand the loop never sleeps. Same trap the mutation runner documented for the mutant.
4. **Ladder re-arm at emission, not enqueue.** Otherwise the 10 000-entry discovery flush arms every retry at `helloAt+200ms`; by the time the pacer reaches entry 5 000 the whole ladder is overdue and a self-sustaining storm returns.
5. **Skip queued sends whose entry is gone or whose stored version changed.** Otherwise a drained or superseded record is still broadcast (and a stale record could be sent).
6. **One pacer queue is not enough, and three are not a bug.** Listing and reactive responses each need the full `sync_response_rate` to meet the R-01 bound; merging them halves the response stream. Keep separate `nextSendAt`s and take the min in `nextDeadline`.
7. **`UnackedStats` must become O(1).** It currently walks all pending (`core.go:1059-1073`) and `mirrorStats` calls it every event (`udprepl/repl.go:575-577`) — leaving it is enough to fail XPORT-02(1) even after deleting `mirrorState`. Use an insertion-ordered list (exact deletion) rather than a lazily popped FIFO (which grows with total writes).
8. **`PeerSnapshot()` for `len()` is O(peers) allocation per event.** Add `PeerCount()`; keep `PeerSnapshot` for tests/observability.
9. **Dedupe identity must include the destination handle and kind.** Keying by key alone collapses one peer's replies into another's; keying replies by (key, version) lets a record update leave two queued sends (harmless but wasteful). Re-read the record at send time so the queue never pins Data.
10. **Queue cap and dedupe map must be updated together.** Evicting the oldest FIFO item without deleting its map entry makes a later identical request look "already queued" and silently starves it.
11. **Outbox results accumulate if not drained.** `Run` reuses one Outbox across iterations; `DrainResults` after every handler. Same in `simnet` (`deliver`/`drive`), even though simulation ignores them.
12. **Removing `SO_REUSEADDR` breaks `TestSmoke_twoNodesUDP` unless the test opts in.** It binds two production adapters to the same wildcard port; the exported test option is the minimal fix. All other adapter tests use distinct ports (verified: `repl_test.go`, `scenario_test.go`, `grpcapi` bench/integration).
13. **Kernel buffer values are doubled and clamped.** Do not assert an exact read-back in tests; assert it is logged and that the send check passes. On a default Linux host (rmem_max 212 992) `SetReadBuffer(4 MiB)` reports 425 984, i.e. ~2× default — the pacing, not the buffer, is the R-01 fix.
14. **`os.File.Sync` on darwin already uses F_FULLFSYNC.** The hand-rolled helper loses EINTR and ENOTSUP handling; deleting it is both the Q-09 and the P-09 fix (verified in `internal/poll/fd_fsync_darwin.go`).
15. **`lastLocal` deletion is safe only because of the mutex discipline.** Prove it in the plan/summary: `await` raises the watermark and appends the waiter under one lock; `runDump` captures the waiter list before `Encode(Records())`; every captured record is therefore in the snapshot. Do not delete the invariant without stating this.
16. **Version validation will break three simulator/scenario families.** `TestScenario_farAhead` uses a 24 h offset (`sim_scenario_test.go:210`), `TestScenario_clockJumps` jumps 2 h forward (`:127`), and `TestDiscard_totality` draws versions across the whole 48-bit `l` domain (`discard_test.go:343-355`, `testutil.RandVersion`). Give the scenarios an explicit `repl.Config.MaxClockLead` large enough to keep their intent, and teach the totality model the new classes. `RandRecord` produces zero checksums (`internal/app/internal/testutil/gen.go:50-63`), so the totality model must either compute checksums in the generator or count them as `bad_checksum`.
17. **`handleDataSync` must not call `Versions()`.** O(store) per chunk fails XPORT-06's 2× at 50 000 keys; use per-entry `Version(key)`.
18. **`engine.Lookup` must be tombstone-visible, unlike `Get`.** A tombstone reply is how a local delete heals a peer's live copy.
19. **The startup `value_max_bytes` check must accept 64 000.** P-01's red tests run `--value-max-bytes=64000` and must stay red (they fail on the dump loader, Phase 5); the Phase 3 worst-case ceiling with a 1 024-byte key is 64 408, so 64 000 passes with 408 bytes to spare. Rejecting it here would turn P-01 into an unrelated startup failure. Use the worst-case varint widths in the check (the zero-filled probe undercounts by up to 14 B).
20. **Key limit choices are coupled:** the API check, the receive check, the startup ceiling and the documented limit must read one constant (`storage.MaxKeyBytes`), or the property "accepted record ≤ 65 507 B" fails at one of the edges. `Set` validates writes; leave `Get` permissive for migrating pre-upgrade long keys (Open Question O2).
21. **Jitter must be deterministic and positive-only where it affects bounds.** Core has no rng; derive offsets from `(runID, counter, salt)`. Positive-only timer jitter keeps `TestSync_triggers`' "never before the interval" assertion (adapt it to an upper bound) and does not shrink the R-01 margin.
22. **`sync_response_rate` clamp stays load-bearing.** The normalized rate clamps to ≤1e9 because `time.Second/rate` truncates to zero (`core.go:59-66`, `:158-160`); the new pacers must reuse the normalized interval, never recompute from a raw config value.
23. **The simulator's quiescence predicate must include the new queues.** `chaosQuiescent` (`sim_chaos_test.go:449-459`) checks `QueuedResponses`; a queued listing/record send would let a scenario declare quiescence while datagrams are still unsent. Add a total-queue accessor (or extend `QueuedResponses`' contract carefully — existing tests assert it means the egress queue).
24. **Strict-lint churn is real.** `mnd`, `gocyclo`/`gocognit`, `funlen`, `godoclint` (97-char doc lines), `errcheck`, `gosec` G115 conversions and `wsl` will all fire on a rewrite of this size. Budget lint iterations; every `//nolint` needs its reason.
25. **`make race-soak` currently excludes `internal/stand` (WR-06).** Adding it (`go test -race -count=20 ./internal/stand/`) matches the Makefile comment's own rationale and is cheap because red tests skip without `SUPKV_RED_TESTS`; measure the wall time before committing to 20.

## Code Examples

### Socket buffers: set, read back, enforce the send minimum
```go
// Source: go doc net.UDPConn.SetReadBuffer/SetWriteBuffer + man7 socket(7)
// (SO_RCVBUF/SO_SNDBUF: "the kernel doubles this value ... returned by getsockopt(2)")
const (
    socketRecvBuffer = 4 << 20
    socketSendBuffer = 1 << 20
)
if err := conn.SetReadBuffer(socketRecvBuffer); err != nil { return fmt.Errorf("set rcvbuf: %w", err) }
if err := conn.SetWriteBuffer(socketSendBuffer); err != nil { return fmt.Errorf("set sndbuf: %w", err) }
rcv, err := socketOption(conn, syscall.SO_RCVBUF)
if err != nil { return fmt.Errorf("read rcvbuf: %w", err) }
snd, err := socketOption(conn, syscall.SO_SNDBUF)
if err != nil { return fmt.Errorf("read sndbuf: %w", err) }
slog.Info("Socket buffers", "recv_buf", rcv, "send_buf", snd)
if snd < datagramBufSize {
    return fmt.Errorf("send buffer %d B < maximal datagram %d B", snd, datagramBufSize)
}

func socketOption(conn *net.UDPConn, opt int) (int, error) {
    raw, err := conn.SyscallConn()          // net/udpsock.go:125
    if err != nil { return 0, err }
    var value int
    var opErr error
    if err := raw.Control(func(fd uintptr) {
        value, opErr = syscall.GetsockoptInt(int(fd), syscall.SOL_SOCKET, opt) // syscall_unix.go:312
    }); err != nil { return 0, err }
    return value, opErr
}
```
Measured this session on Linux: requested 1 MiB → 2 097 152; 4 MiB → 8 388 608; 65 507 → 131 014.

### Read loop: only `net.ErrClosed` exits
```go
// Source: go doc net.ErrClosed ("should normally be tested using errors.Is")
for {
    n, src, err := a.conn.ReadFromUDP(buf)
    if err != nil {
        if errors.Is(err, net.ErrClosed) { return }
        a.readErrors.Add(1)
        a.logReadError(err)         // rate-limited
        time.Sleep(readErrorBackoff)
        continue
    }
    ...
}
```

### Store read view (tombstone-visible)
```go
// Mirrors the locking discipline of storage.Get (store.go:141-153) and Records (store.go:274-287).
func (s *Store) Version(key string) (version.RecordVersion, bool) {
    s.mu.RLock(); rec, ok := s.records[key]; s.mu.RUnlock()
    return rec.Version, ok
}
func (s *Store) Lookup(key string) (Record, bool) {
    s.mu.RLock(); rec, ok := s.records[key]; s.mu.RUnlock()
    if !ok { return Record{}, false }
    return Record{Version: rec.Version, Data: bytes.Clone(rec.Data), Deleted: rec.Deleted, Checksum: rec.Checksum}, true
}
func (s *Store) Versions() map[string]version.RecordVersion {
    s.mu.RLock(); defer s.mu.RUnlock()
    out := make(map[string]version.RecordVersion, len(s.records))
    for k, rec := range s.records { out[k] = rec.Version }
    return out
}
```

### Version validation (before any store/clock call)
```go
// Source: version layout constants, version.go:20-40 ("uuidVerBits = 0x70",
// "uuidVarBits = 0x80", "l = int64(binary.BigEndian.Uint64(v[0:8]) >> lPadding)")
func (v RecordVersion) Validate(now time.Time, maxLead time.Duration) error {
    if v[6]&0xF0 != uuidVerBits || v[8]&0xC0 != uuidVarBits { return ErrVersionLayout }
    l, _, _ := v.Unpack()
    nowMs := now.UnixMilli()
    lead := maxLead.Milliseconds()
    if lead < 0 { lead = 0 }
    if l >= nowMs && l-nowMs > lead { return ErrVersionFuture }
    return nil
}
```

### Startup ceiling check through the real codec
```go
// Source: local probe against the generated schema this session. Overhead is 66-75 B at
// worst-case varint widths; key=1024 + value=64408 = exactly 65 507 B.
// IMPORTANT: fill every envelope/record varint with its worst case, or the check undercounts.
func MaxRecordDatagramBytes(keyBytes, valueBytes int) int {
    body := &replproto.Body{Msg: &replproto.Body_DataReplication{DataReplication: &replproto.DataReplication{
        Key: strings.Repeat("k", keyBytes), Version: make([]byte, 16), // 16 B fixed
        Data: make([]byte, valueBytes), Checksum: math.MaxUint32,      // 5-byte varint
    }}}
    b, _ := proto.Marshal(body)
    env, _ := proto.Marshal(&replproto.Envelope{
        ProtoVersion: math.MaxUint32, NodeId: math.MaxUint32, // worst-case varints
        RunId: make([]byte, 16), Crc32: math.MaxUint32, Body: b,
    })
    return len(env)
}
```

### Pacer deadline arithmetic (existing, reuse verbatim)
```go
// Source: Core.drainEgress, core.go:668-678 — at most one emission per Advance when late.
for len(queue) > 0 && !now.Before(c.nextSendAt) {
    item := queue[0]; queue = queue[1:]
    out.Broadcast(item.msg)
    if now.After(c.nextSendAt) { c.nextSendAt = now }
    c.nextSendAt = c.nextSendAt.Add(c.syncResponseInterval)
}
```

### simnet `Cluster.Write` error handling (WR-02)
```go
// Source: review 02-REVIEW.md WR-02; mirror Delete's error path (simnet.go:741-754).
func (c *Cluster) Write(i int, key string, data []byte) storage.Record {
    c.checkIndex(i, "Write")
    now := c.sched.Now()
    if _, _, err := c.nodes[i].store.Set(key, data); err != nil {
        panic(fmt.Sprintf("simnet: Cluster.Write: Set %q on node %d: %v", key, i, err))
    }
    rec, _ := c.nodes[i].store.Lookup(key)   // cheap point read, not Records()[key]
    var out port.Outbox
    c.nodes[i].core.NotifyLocalWrite(now, key, rec, &out)
    c.route(i, out.Drain())
    c.drive(i, now)
    c.writes = append(c.writes, LoggedWrite{Key: key, Rec: rec})
    return rec
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|---|---|---|---|
| Retransmit scanner sorts+walks all pending every `retransmit_interval` | Due-heap touch-only-due entries | This phase | O(due) instead of O(unacked); satisfies XPORT-02(1). |
| Adapter clones/sorts the pending set after every event to serve `wait_for_replication` | Core emits ack/drain/supersede results; adapter forwards to a tracker | This phase | Removes the measured 52% `mirrorState` CPU site and the 5 ms poll. |
| Whole-store deep copy per datagram | `Version`/`Lookup`/`Versions` view | This phase | O(record)/O(chunk) instead of O(store); feeds Phase 6's `List`. |
| Unpaced listing/retransmit bursts | Per-send pacing at `sync_response_rate`, rotating listing start, jittered timers | This phase | Removes the systematic tail loss and the retransmit storm. |
| Assumed socket buffer sizes | Requested + kernel-reported (doubled/clamped) values logged; send-buffer floor enforced | This phase | Makes R-08 observable; avoids "we set it" fiction. |

**Deprecated/outdated:**
- `SO_REUSEADDR` in production: replaced by test-only opt-in.
- `persist.Parse` (unused exported oracle), `ownHandle`, `lastLocal`/`cover`, `fsync_darwin.go`/`fsync_other.go`: deleted.
- The adapter's `drainedRetention`/`lastAck`/`prevPending` mirror: deleted with the result seam.
- `ServerOptions.PendingCopy`: replaced by the adapter's `WaitReplicated` seam (Phase 6 APIS-01 may refine the report, not the seam).

## Assumptions Log

> Every row below is `[ASSUMED]` unless its "Claim" cell already carries a tag. The planner and discuss-phase use this table to identify decisions that need user confirmation before execution.

| # | Claim | Section | Risk if Wrong |
|---|---|---|---|
| A1 | `max_clock_lead = 60 s` is the right default (equals `drift_warn`, so peers inside the documented drift tolerance are accepted) | Pattern 6 | A stricter/looser bound changes which clock-skew scenarios pass; Phase 7 can revisit validation. |
| A2 | Key limit = 1 024 B (review's suggested "e.g. 1024 B"); max value with that key = 64 408 B (measured worst case) | Pattern 6 / Pitfall 20 | A different limit changes the API contract and the startup ceiling; 4 096 was the review's earlier example and leaves only ~3 500 B for values. |
| A3 | darwin `net.inet.udp.maxdgram` = 9 216 B | Pattern 7 | Taken from the committed review evidence/ADR 0010, not measured here (no darwin host); not acted on — documented only. |
| A4 | CI Linux runners' `rmem_max`/`wmem_max` allow a ≥ 65 507 B send buffer | Pattern 7 | If a runner clamps lower, startup would fail; the send check should report both requested and actual so it is diagnosable. |
| A5 | The 20× R-01 nightly job fits a 60–90 min slot | Validation | At ~90 s/run it is 30 min; slower runners could exceed. Use a dedicated job with a 120-min timeout. |
| A6 | Egress cap 32 768 keeps the R-01 bound comfortable | Pattern 9 | A smaller cap still converges by re-requesting next round but consumes the bound's margin. |
| A7 | Removing `PendingCopy` now is acceptable because Phase 6 APIS-01 owns the final report semantics | Pattern 5 | If Phase 6 needs a snapshot for identity-based semantics, the tracker can be extended; the seam remains. |
| A8 | Non-red tests may be edited freely; only the three red test files stay untouched | Summary | If the phase insists on zero edits everywhere, several timing assertions cannot be updated. |
| A9 | Deterministic hash jitter satisfies "timers jitter" without `math/rand` | Pattern 2/10 | Same-seed replay must remain identical; any rng would break Phase 2's determinism gate. |
| A10 | Per-peer ledger memory (`O(peers × unacked)`) is acceptable at the documented scale (2–a handful of nodes) | Pattern 3 | For many nodes the ledger duplicates keys; still O(unacked) for fixed N. |

## Open Questions

1. **Exact key limit (1 024 vs 4 096 vs unbounded with a per-record ceiling).**
   - What we know: measured DataReplication overhead 66–75 B at worst-case varint widths; with 1 024 B keys the default 61 440 value fits with ~3 KiB slack; with 4 096 B keys and 61 440 values the datagram is 65 611 B (over).
   - What's unclear: the review says "e.g. 1024"; the docs' earlier example used 4 096.
   - Recommendation: 1 024 B, one constant `storage.MaxKeyBytes`; document in `docs/OPERATIONS.md` and the SPEC-DEVIATIONS S-DM-03 row.
2. **Write-only vs all-RPC key validation.**
   - What we know: XPORT-03 only requires `Set`; receive-side validation is needed for the datagram property.
   - What's unclear: blocking `Get` on a pre-upgrade long key could obstruct migration.
   - Recommendation: validate `Set`/`Delete` + inbound datagrams; keep `Get` permissive; note in docs.
3. **Timer jitter scope.**
   - What we know: jitter is in XPORT-09's title, not its acceptance text; retransmit-ladder jitter conflicts with the documented `loss ≤ ack_timeout_max` guarantee; `TestSync_triggers` asserts exact intervals.
   - What's unclear: whether the planner treats the title as binding.
   - Recommendation: implement deterministic positive-only jitter on Hello/sync deadlines and pacer spacing; do **not** jitter ladder deadlines; adapt `TestSync_triggers` to an upper bound. If descoped, record the justification.
4. **Jitter magnitude.** Recommendation: ±10% for pacer spacing (average = rate), 0–10% for Hello/sync deadlines (never earlier).
5. **20/20 R-01 evidence placement.** Dedicated nightly job vs an extension of `red-run`; and whether the local 8 000-key scale may carry the 20× at PR time.
6. **Whether to fix RPL-18 now** ("a failed send is counted as sent"). XPORT does not require it; OBSV-03 (Phase 7) does. Recommendation: leave it, but ensure the new pacing counter does not compound the lie by counting enqueues.
7. **WR-02 / WR-06 scope.** WR-02 (simnet `Write` swallows `Set`'s error) is a small, exactly-in-class fix that also removes a full-store copy per simulated write — worth doing here. WR-06 (add `internal/stand` to `race-soak`) is a one-line Makefile change; measure the ×20 cost first.
8. **`SendStats` metric names.** CONTEXT says pacing counters land now, OBSV-03 finalizes semantics; treat names as provisional (`supkv_replication_read_errors_total`, `supkv_replication_datagrams_paced_total`, `supkv_replication_send_queue`, `supkv_replication_egress_dropped_total`).
9. **Whether `version.Validate` should reject the 46 filler bits.** RFC 9562 permits random filler; rejecting it would break any future conforming implementation. Recommendation: no.
10. **`StateStore.Version` vs making `Lookup` return a borrowed slice.** Recommendation: keep the Data clone (safe contract, O(record)); the phase's memory goal is the *retained* pending record, which Pattern 3 removes.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|---|---|---|---|---|
| Go toolchain | everything | ✓ | go1.27.4 linux/amd64 | — |
| golangci-lint (pinned via `go tool`) | `make lint` gate | ✓ | 2.15.0 built with go1.27.4 | — |
| Linux loopback broadcast (`127.255.255.255` fan-out) | stand tests | ✓ (Phase 2 CI + stand harness probe; local host assumed) | kernel 7.3.4 | A probe miss is a FATAL on Linux per TEST-05, never a skip. |
| `rmem_max`/`wmem_max` ≥ 65 507 | send-buffer startup floor | ✓ on this host (4 194 304) | kernel sysctls | Cannot be raised unprivileged; the check reports actual values. |
| Docker | none (not needed this phase) | — | — | — |
| Network access | none at runtime/test | — | — | — |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** darwin runtime (not available here) — pattern 7's darwin claim is documented, not measured; the macOS CI cell covers build + unit tests.

## Validation Architecture

> `workflow.nyquist_validation` is `true` in `.planning/config.json` (key present), so this section is included.

### Test Framework
| Property | Value |
|---|---|
| Framework | `go test -race` (vanilla `testing`); `internal/stand` real-process suite (`//go:build linux`); `internal/app/internal/simnet` deterministic simulator; `scripts/red-runner` expectation gate |
| Config file | `Makefile` (`lint`, `test`, `bench`, `bench-smoke`, `race-soak`, `red-run`); no separate test config |
| Quick run command | `go test -race ./internal/app/repl/ ./internal/app/storage/ ./internal/app/persist/ ./internal/udprepl/ ./internal/grpcapi/ ./cmd/sup-kv/` |
| Full suite command | `make lint && make test` then `bash scripts/red-runner` (red-evidence + mutation stage) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|---|---|---|---|---|
| XPORT-01 | Paced listing, rotation, convergence bound | unit + stand | `go test -race ./internal/app/repl/ -run 'TestSync_(pacing\|rotation)'`; `SUPKV_RED_TESTS=1 go test -race ./internal/stand/ -run '^TestRed_R01_pacedSync20k$'` | ❌ new unit tests; red test exists (`stand_red_test.go:276`) |
| XPORT-01 | 20/20 evidence | scheduled repeat | `SUPKV_RED_TESTS=1 SUPKV_STAND_NIGHTLY=1 go test -race -count=20 ./internal/stand/ -run '^TestRed_R01_pacedSync20k$'` | ❌ new scheduled job |
| XPORT-02 | Flat event-loop cost | benchmark | `go test -run '^$' -bench 'BenchmarkCoreEventLoop' -benchtime=1x ./internal/app/repl/` | ❌ new `bench_test.go` |
| XPORT-02 | Drain ≤ N/rate+10 s, ≤ 3N; burst ≥ 990/s, empty 5 s | stand | `SUPKV_RED_TESTS=1 go test -race ./internal/stand/ -run 'TestRed_R02_(unackedDrain\|burst)'` | ✅ (`stand_red_test.go:381`, `:529`) |
| XPORT-03 | `Set` key limit → InvalidArgument; startup exit 2; ≤ 65 507 property | unit + exec | `go test ./internal/grpcapi/ -run TestServer_keyLimit`; `go test ./cmd/sup-kv/ -run TestMain_valueMaxDatagram`; `go test ./internal/udprepl/ -run TestDatagram_maxRecordSize` | ❌ new |
| XPORT-04 | Version rejected before clock; own class; DataSync too | unit | `go test ./internal/app/version/ -run TestValidate`; `go test ./internal/app/repl/ -run 'TestValidate\|TestDiscard'` | ❌ new; extend `discard_test.go` |
| XPORT-05 | Buffers logged; send floor; 1/20/60 KiB deliver | unit + stand | `go test ./internal/udprepl/ -run TestSocketBuffers`; `go test ./internal/stand/ -run TestStand_socketBuffers` | ❌ new |
| XPORT-06 | 1 000 vs 50 000 keys ≤ 2× | benchmark | `go test -run '^$' -bench 'BenchmarkHandle(DataRequest\|DataSync)' ./internal/app/repl/` | ❌ new |
| XPORT-07 | 2×1000 → 1000 queued; cap drops oldest | unit | `go test ./internal/app/repl/ -run TestEgress_boundedDeduped` | ❌ new |
| XPORT-08 | Only `net.ErrClosed` exits; errors counted | unit (fake conn) | `go test ./internal/udprepl/ -run TestReadLoop` | ❌ new |
| XPORT-09 | Non-wildcard WARN; second instance EADDRINUSE; no-Hello WARN; jitter | exec + unit | `go test ./cmd/sup-kv/ -run 'TestMain_(bindAddrWarn\|duplicateBind)'`; `go test ./internal/app/repl/ -run TestNoPeerWarn` | ❌ new |
| XPORT-10 | Empty key / tombstone+data / bad checksum classes; negative table flips | unit | `go test ./internal/udprepl/ -run TestDecode_negativeTable`; `go test ./internal/app/repl/ -run TestDiscard` | ✅ rows exist (`decode_negative_test.go:190-270`), flip them + add core rows |
| XPORT-11 | Version/Lookup/Versions tombstone visibility | unit + property | `go test ./internal/app/storage/ -run 'TestStore_(version\|lookup\|versions)'` | ❌ new |
| XPORT-12 | Result seam; no polling; wait behaviors preserved | unit + integration | `go test ./internal/grpcapi/ -run 'TestWait'`; `go test ./internal/grpcapi/ -run TestWaitIntegration` | ✅ (`wait_test.go`, `wait_integration_test.go`) — rewrite fakes |
| XPORT-13 | Dead code gone | build + grep gates | `go build ./...`; `! grep -rn 'lastLocal\|ownHandle\|func Parse' internal/` ; `GOOS=darwin go build ./...` | ✅ |
| All | Red flips | gate | `bash scripts/red-runner` (R-01, R-02 rows `green`) | ✅ TSV exists |

### Sampling Rate
- **Per task commit:** the task's own command above (unit packages are sub-second to a few seconds).
- **Per wave merge:** `go build ./...` + targeted `go test -race` + `go vet ./...`; after the core rewrite, `bash scripts/red-runner` (stand reds are minutes).
- **Phase gate:** `make lint && make test`, `bash scripts/red-runner`, `GOOS=darwin GOARCH=arm64 go build ./... && GOOS=darwin GOARCH=arm64 go vet ./...`, `make bench-smoke`.

### Wave 0 Gaps
- [ ] `internal/app/repl/bench_test.go` — event-loop (10 vs 10 000 pending), `HandleDataRequest` and `HandleDataSync` at 1 000 vs 50 000 keys (XPORT-02(1), XPORT-06).
- [ ] `internal/app/repl/egress_test.go` (or extend `sync_test.go`) — bounded/deduped queue.
- [ ] `internal/app/version/validate_test.go` — layout/future/edge (exactly `now+lead` valid).
- [ ] `internal/app/storage/view_test.go` — tombstone visibility, consistency with `Records()`, concurrency.
- [ ] `internal/udprepl/socket_internal_test.go` — fake `packetConn` read-loop test; buffer read-back via a real loopback socket.
- [ ] `internal/stand/` — buffer sizes logged + 1/20/60 KiB delivery; rewrite `TestStand_oversizeKeyF2c`; second-instance test (or `cmd/sup-kv`).
- [ ] `cmd/sup-kv/main_test.go` — bind-address WARN; `--value-max-bytes` exit 2; `--max-clock-lead` env twin.
- [ ] Metrics tests: `internal/metrics/metrics_test.go` family table/zero lines, `cmd/sup-kv/main_test.go` family list, and any `stand_metrics_test.go` exact-value assertions.
- [ ] `scripts/red-runner` mutation sed update + a per-mutant source generalization; re-verify the corpus kill seeds.
- [ ] Scheduled workflow: the 20× R-01 job (or a documented alternative); `Makefile` `race-soak` gains `./internal/stand/` (WR-06) after measuring cost.

## Security Domain

> `workflow.security_enforcement` is `true`, `security_asvs_level` 1. The trust model is unchanged: trusted LAN segment, CRC32 for integrity, no authentication (AUTH-01 deferred to v2).

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---|---|---|
| V2 Authentication | no | Out of scope by design (trusted segment; AUTH-01 v2). |
| V3 Session Management | no | No sessions. |
| V4 Access Control | no | No authorization model; the startup gate is availability, not access control. |
| V5 Input Validation | **yes** | Version layout + `now+max_clock_lead` bound, key emptiness/length, tombstone-with-data, record checksum, datagram-size ceiling; all rejected before store/clock; counters per class; no key names in logs/counter details. |
| V6 Cryptography | no new crypto | Existing CRC32 is corruption detection only (accepted limitation); do not hand-roll anything; `crypto/rand` stays only for `runID`. |

### Known Threat Patterns for the UDP/Go stack
| Pattern | STRIDE | Standard Mitigation |
|---|---|---|
| Far-future version (`ff…ff`) poisons the HLC, freezing version ordering cluster-wide (review R-07) | Tampering / DoS | Validate `ver`/`var` nibbles and `l ≤ now + max_clock_lead` **before** `ApplyRemote`; `bad_version` class; rate-limited WARN. |
| Oversized key/value creates a confirmed but never-sendable record (EMSGSIZE forever) | DoS / integrity | Key limit at the API + on receive; startup exit 2 when `key_max + value_max + overhead > 65507`; property test through the real codec. |
| Empty key, tombstone-with-data, forged record checksum | Spoofing / Tampering | Reject with own classes/counters before the store; the store recomputes the checksum at install. |
| Log flooding via crafted/malformed datagrams | DoS | Per-datagram discard logging stays `slog.Debug` (existing); new WARNs rate-limited per class; counters are the reliable signal. |
| Second local process hijacks the node's port (SO_REUSEADDR) | Spoofing / Availability | Remove `SO_REUSEADDR` from production; test-only opt-in; EADDRINUSE at startup. |
| Deaf node keeps broadcasting Hello (read loop died silently) | Availability / integrity | Only `net.ErrClosed` exits; other errors counted + logged + backoff; `/metrics` exposes the counter. |
| Unbounded pending mirror/queues → CPU/memory exhaustion under load | DoS | O(1) mirror, record-free ledger, bounded/deduped egress, paced sends. |
| Non-wildcard bind → silent one-way replication | Misconfiguration (Repudiation) | Explanatory WARN at startup (or rejection), documenting broadcast reception. |
| Mis-set `max_clock_lead` (absurdly large) | Misconfiguration | Phase 7's one-point validation; Phase 3 normalizes non-positive to 60 s and documents the relation to `drift_warn`. |
| Failed datagram sends counted as sent (metric lie) | Repudiation | Out of scope here (OBSV-03, Phase 7); noted so the phase does not add another pre-send counter. |

## Sources

### Primary (HIGH confidence)
- Go 1.27.4 toolchain source, read this session:
  - `net/sockopt_linux.go:21` and `net/sockopt_bsd.go:36` — Go sets `SO_BROADCAST` by default for datagram sockets.
  - `net/sock_posix.go:178-215` — `listenDatagram` does **not** set `SO_REUSEADDR`.
  - `net/udpsock.go:125` — `UDPConn.SyscallConn`.
  - `syscall/syscall_unix.go:312` — `GetsockoptInt` (`//go:build unix`, includes darwin); `zerrors_darwin_arm64.go:974,980` — `SO_RCVBUF`/`SO_SNDBUF` exist on darwin.
  - `internal/poll/fd_fsync_darwin.go:1-35` — `os.File.Sync` uses `F_FULLFSYNC` + `ignoringEINTR` + `ENOTSUP` fallback.
- `go doc net.UDPConn.SetReadBuffer`, `SetWriteBuffer`, `net.ErrClosed`, `syscall.GetsockoptInt` — output read this session.
- Empirical probes on this Linux host (Go 1.27.4, kernel 7.3.4):
  - `SO_RCVBUF`/`SO_SNDBUF` read-back: requested 1 MiB → 2 097 152; 4 MiB → 8 388 608; 65 507 → 131 014; sysctls `rmem_max=wmem_max=4 194 304`.
  - UDP double-bind: without `SO_REUSEADDR` → `[Errno 98] Address already in use`; with it → succeeds (v1.0 defect reproduced).
  - Datagram size probe through the generated schema: overhead 52–61 B with zero-valued envelope fields and 66–75 B at worst-case varint widths; key 1024 + value 61440 → 62 539 B (fits); key 1024 + value 64 408 → 65 507 B (ceiling); key 4096 + value 61440 → 65 611 B (over).
- man7 `socket(7)` (fetched): SO_RCVBUF/SO_SNDBUF doubling and `rmem_max`/`wmem_max` ceilings.
- In-repo code (read this session, current line numbers): `internal/app/repl/core.go`, `internal/udprepl/repl.go`, `internal/app/storage/store.go`, `internal/app/persist/engine.go`/`codec.go`, `internal/persistfile/fsops.go`/`fsync_darwin.go`/`fsync_other.go`, `internal/grpcapi/wait.go`/`gate.go`/`server.go`, `internal/metrics/metrics.go`, `internal/app/internal/simnet/simnet.go`, `internal/stand/stand.go`/`stand_red_test.go`/`stand_test.go`/`stand_metrics_test.go`, `internal/udprepl/decode_negative_test.go`, `cmd/sup-kv/main.go`, `internal/app/version/version.go`.
- In-repo review inputs: `docs/reviews/2026-11-19-v1.0/REVIEW.md` §5 R-01/R-02/R-06–R-09/R-11/R-14–R-16, §10 Q-01/Q-02/Q-09, §14 (14.10/14.11), §15; `docs/reviews/2026-11-19-v1.0/evidence/review-repl-api.md` RPL-01…RPL-18; `evidence/review-engine.md` ENG-18; `evidence/stand/RESULTS.md` F9/F9b/F17.
- In-repo decisions: `docs/adr/0010-darwin-development-only.md`, `docs/adr/0011-value-limit-60kib-kept.md`; `.planning/phases/03-transport/03-CONTEXT.md`; `.planning/REQUIREMENTS.md` XPORT-01…13; `.planning/ROADMAP.md` Phase 3.

### Secondary (MEDIUM confidence)
- `.planning/phases/02-stand-and-simulator-as-tests/02-REVIEW.md` WR-02/WR-06 + `02-REVIEW-DISPOSITION.md`.
- `scripts/red-runner` / `scripts/red-expectations.tsv` (mutation seds, expected-red rows).
- `.github/workflows/ci.yml`, `scheduled.yml`, `Makefile` (job budgets, gate commands).

### Tertiary (LOW confidence)
- The darwin ~9 KiB `maxdgram` figure is taken from the committed review evidence, not measured in this session; it is documented, not fixed, per ADR 0010.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every API and kernel behavior used here was verified in the local toolchain source, `go doc`, man7, or a local probe.
- Architecture: MEDIUM-HIGH — the seams and failure modes are read from the current code; the pacing/ledger/tracker shapes are recommendations with explicit alternatives and locked-decision interpretations.
- Pitfalls: HIGH — most were verified by reading the exact code/tests that will break, including the red-runner mutation seds, the simulator clock-skew scenarios, and the P-01 flag value.
- Validation: MEDIUM — the mapping is concrete, but the 20×/nightly timing and jitter-related test adaptations need measured confirmation.

**Research date:** 2026-11-21
**Valid until:** ~2026-12-21 for the in-repo facts (fast-moving phase); re-verify Go/kernel behavior only if the toolchain or CI image changes.
