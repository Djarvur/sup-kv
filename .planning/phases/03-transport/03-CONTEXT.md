# Phase 3: Transport - Context

**Gathered:** 2026-11-21
**Status:** Ready for planning
**Mode:** --auto (all gray areas auto-selected; every choice logged below and reviewable in git)

<domain>
## Phase Boundary

Make the UDP transport meet the review's stated bounds on real sockets: paced and bounded sending (sync listing + retransmits), socket buffers configured and reported, received input validated before it touches the clock, cheap key→version reads instead of store copies, an honest Outbox "acked" seam, a bounded/deduplicated sync-answer queue, a read loop that cannot silently die, and the dead code from Q-09 gone or justified. Requirements XPORT-01…XPORT-13. This phase turns the Phase-2 red tests R-01 and R-02 GREEN; it does NOT touch protocol reconciliation semantics (Phase 4) or persistence (Phase 5).

</domain>

<decisions>
## Implementation Decisions

### Paced sync listing (XPORT-01 / R-01)
- **D-01:** DataSync listing sends go through a **token-bucket pacer at `sync_response_rate`** shared by all outgoing listing traffic (per-send pacing, not per-chunk burst); the sending **start chunk rotates across sync rounds** (a rotating offset persisted per peer) so repeated rounds cover the whole key space under buffer loss. Convergence bound: 2 × `sync_interval` + N / `sync_response_rate`. — [auto] [Sync pacing] Q: how is the listing paced and made loss-robust? → Selected: token-bucket at sync_response_rate + rotating start chunk (recommended default: the review's own R-01 acceptance describes exactly this shape). — **Reversibility:** reversible.
- **D-02:** `SO_RCVBUF`/`SO_SNDBUF` are set at socket setup (Go `SetReadBuffer`/`SetWriteBuffer` with a double read-back), the **actual values are logged at startup**, and startup **fails when the send buffer cannot hold a maximal datagram**; darwin's ~9 KiB send limit is documented, not fixed (14.10). — [auto] [Socket buffers] Q: configure+report buffers or document limits only? → Selected: configure + log + fail-fast (recommended default: R-08's acceptance wording). — **Reversibility:** reversible.

### Unacked set and retransmits (XPORT-02 / R-02, XPORT-12)
- **D-03:** The owner-side **unacked ledger replaces snapshot diffing**: the core tracks per-peer unacked records in a map keyed by (key, version), updates it from acked results, and retransmit scanning iterates the ledger in place (O(unacked), no store copies, no `mirrorState` deep scans). Iteration time at 10 vs 10 000 unacked ≤ 2×. — [auto] [Unacked cost] Q: how does event-loop cost stop growing with the unacked set? → Selected: per-peer unacked ledger keyed (key,version) (recommended default: Q-02's stated remedy). — **Reversibility:** costly — touches core loop + adapter mirror.
- **D-04:** "Record acknowledged by peer X" travels as an **`Outbox` result** from the core; the adapter forwards it to waiters (`wait_for_replication`) — removing the 5 ms polling and snapshot diffing in `wait.go`. This is the same seam Phase 4/5 build on. — [auto] [Ack seam] Q: how do waiters learn about acks? → Selected: Outbox result + waiter channel (recommended default: Q-02/XPORT-12). — **Reversibility:** costly — the wait API contract rides it.
- **D-05:** Retransmits are **paced with jitter** on the existing `retransmit_interval` ladder (per-peer scheduling, no whole-set storm); a peer returning to a node holding 10 000 unacked drains within N / `sync_response_rate` + 10 s with ≤ 3N retransmits; the open-loop 1000 writes/s × 15 s burst over 10 000 distinct keys achieves ≥ 990 writes/s with the set empty 5 s after the end. — [auto] [Retransmit pacing] Q: how are retransmits bounded? → Selected: per-peer jittered pacing on the existing backoff ladder (recommended default: R-02 acceptance). — **Reversibility:** reversible.

### Input validation and read loop (XPORT-03/04/08/10)
- **D-06:** Received versions are **validated before any clock call**: `ver`/`var` fields sane, `l` within `now + max_clock_lead` (new configurable bound, default 60 s), invalid → discarded with its own class (`bad_version`), WARN-logged rate-limited; the same rule applies to versions inside DataSync. Empty key, tombstone-with-data, unverified checksum get their own discard counters (extending the Phase-2 negative table). Key-length and `value_max_bytes` ceilings enforced (`InvalidArgument`; exit 2 when the largest record cannot fit a datagram; any accepted record encodes to ≤ 65 507 bytes). — [auto] [Validation] Q: validate before clock or trust the sender? → Selected: validate before clock, own classes (recommended default: R-07/R-16 acceptance). — **Reversibility:** reversible.
- **D-07:** The read loop **exits only on `net.ErrClosed`**; any other read error is logged and counted (or the process exits deliberately) — a node never keeps sending `Hello` while deaf; a node that received no Hello for `peer_timeout` after start WARNs. Bind-address misuse is loud: non-wildcard `--bind-addr` warns or is rejected; second instance fails with "address already in use". — [auto] [Read loop] Q: what may terminate the read loop? → Selected: only net.ErrClosed; everything else counted/logged (recommended default: R-14/R-15 acceptance). — **Reversibility:** reversible.

### Seams and cleanup (XPORT-05/06/07/11/13)
- **D-08:** The store gains a **cheap key→version view with tombstone visibility** + point reads (shallow iteration under RLock, no deep copies); DataRequest replies and sync-chunk handling use it — making them independent of foreign record reconstruction. A **bounded, deduplicated sync-answer queue** (keyed by (key, version), capacity-capped) ensures 1000 missing keys answered twice produce 1000 answers, not 2000. Dead code from Q-09 (`lastLocal`/`cover`, exported `persist.Parse`, `ownHandle`, `fsync_darwin.go`) is **deleted or justified in place** (fsync_darwin.go removal coordinates with Phase 5's P-09 — if Phase 5 intends to rework it, delete here and note the Phase-5 follow-up). — [auto] [Seams] Q: key→version view + queue + dead code? → Selected: shallow view + dedupe queue + delete-with-justification (recommended default: Q-01/R-11/Q-09 acceptance). — **Reversibility:** costly — the store API is consumed across phases.

### Claude's Discretion
- Exact pacer implementation (token bucket vs timer window), jitter distribution, `max_clock_lead` default (60 s unless research finds a better-grounded value), queue capacities, rotation persistence details.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Review inputs (committed)
- `docs/reviews/2026-11-19-v1.0/REVIEW.md` — §5 R-01, R-02, R-06–R-09, R-11, R-14–R-16 (findings + acceptance), §15 (phase composition), §10 Q-01/Q-02/Q-09
- `docs/reviews/2026-11-19-v1.0/evidence/RESULTS.md` + `evidence/stand/` — the measured numbers (F9b pacing, F9 storm, burst 465/s) the phase must beat
- `docs/SPEC-DEVIATIONS.md` + `docs/adr/0005…` — decision names tests cite
- `.planning/phases/02-stand-and-simulator-as-tests/02-SUMMARY.md`… (17 summaries) — the red tests that must turn green: `TestRed_R01_pacedSync20k`, `TestRed_R02_unackedDrain`, `TestRed_R02_burst` in `internal/stand/stand_red_test.go`; runner table `scripts/red-expectations.tsv`
- `scripts/red-runner` + `scripts/red-expectations.tsv` — the flip mechanism: rows flip red→green per fix (D-01 contract)

### Planning anchors
- `.planning/ROADMAP.md` — Phase 3 goal + 5 success criteria
- `.planning/REQUIREMENTS.md` — XPORT-01…XPORT-13 verbatim acceptance
- `.planning/phases/01-spec-reference-and-decisions/01-CONTEXT.md` — registry/ADR names
- `.planning/PROJECT.md` — constraints, parameter defaults table
- `.planning/phases/02-stand-and-simulator-as-tests/02-CONTEXT.md` — the Phase-2 decisions this phase turns green

### Code (targets)
- `internal/app/repl/core.go` — the pure repl core (unacked/mirror state, retransmit scan, merge stats); D-03/D-04 target
- `internal/udprepl/repl.go` — the socket adapter (read loop, send path, buffers); D-01/D-02/D-07 target
- `internal/app/storage/store.go` — the LWW store (Records() deep copy); D-08 key→version view target
- `internal/grpcapi/wait.go` — the 5 ms polling this phase removes (D-04)
- `.golangci.yml` — depguard boundary (app stays I/O-free)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- The Port/Outbox seam from v1.0 (core returns results, adapter applies side effects) — D-03/D-04 extend it, never bypass
- `replproto` encoder for the datagram-size bound (D-06)
- Phase-2 stand harness (`internal/stand`) + red runner — the verification loop for every transport change
- simnet (with Phase-2 extensions: budgets, SizeOf, dump delay, deletions) — fast regression net while iterating

### Established Patterns
- Core stays goroutine-free and I/O-free (depguard-enforced); adapter owns sockets/timers
- Deterministic simulator runs the production core — every core change must keep seed replay byte-identical (Phase-2 test)
- Strict lint (no suppressions without justification); table-driven tests; `make lint`/`make test` green

### Integration Points
- `internal/stand/stand_red_test.go` red tests flip green as fixes land (no test edits needed — full acceptance already asserted)
- `scripts/red-expectations.tsv` row status flips red→green per fix (the runner asserts the new status)
- Metrics: new discard classes + pacing counters land alongside the existing in-repo exporter (OBSV-03 semantics are Phase 7's to finalize)

</code_context>

<specifics>
## Specific Ideas

- The measured red numbers are the targets to beat: R-01 local 2 691/8 000 at bound (nightly 2 976/20 000); R-02 drain never within 35 s, burst 25–52 writes/s; F9 retransmit storm 949 552 retries / 14.5 min. The Phase-2 red-runner harness re-measures automatically.
- The walkthrough of R-02's acceptance names exact numbers (10 000 unacked, ≤ 3N retransmits, set empty 5 s after burst end) — those are the tests' assertions already committed.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 3-Transport*
*Context gathered: 2026-11-21*
