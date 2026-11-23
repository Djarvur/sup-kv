# Phase 7: Observability and Configuration - Context

**Gathered:** 2026-11-23
**Status:** Ready for planning
**Mode:** --auto (all gray areas auto-selected; every choice logged below and reviewable in git)

<domain>
## Phase Boundary

An operator can see what the node is doing and misconfiguration fails fast: lifecycle logs, one validation point, metric definitions that match the binary, distinct exit codes, and a machine-enforced dependency boundary. Requirements OBSV-01…08. Also folds every deferral prior phases routed here: the four Phase-4 counters' metrics surface, the unbound-listing discard class, the Phase-5 library-level value-limit guard, Phase-3's `sync_chunk_bytes` bounds and the unrate-limited apply-failure WARN.

**Depends on:** Phase 6 (public API and validation surface).

</domain>

<decisions>
## Implementation Decisions

### Lifecycle logging (OBSV-01/05/06)
- **D-01:** One lifecycle log catalog, emitted through `slog` with stable levels and fields: a startup entry carrying the binary version and the effective configuration (secrets-free); one entry each for the loaded dump generation, a dump failure, gate opening, peer appear/disappear/restart and drift warnings. A fatal bind error at `SUPKV_LOG_LEVEL=error` produces an ERROR entry and a non-zero exit, never a silent exit 1. The Spec-mandated entries (dump failure, start without usable dumps, foreign proto_version, duplicate node_id) sit at the documented levels. — [auto] [Lifecycle logs] Q: what does the catalog cover? → Selected: startup+config, load generation, dump failure, gate, peers, drift at stable levels (O-01/S-07). — **Reversibility:** reversible.
- **D-02:** Clock-drift warnings match the Spec: a received version leading system time by more than `drift_warn` yields one WARN plus the drift metric; the runbook text describes exactly this condition in the shipped binary (no fictional conditions). — [auto] [Drift] Q: does the runbook match the binary? → Selected: WARN + metric at `drift_warn`, runbook tied to the code path (S-06). — **Reversibility:** reversible.

### One validation point (OBSV-02)
- **D-03:** One config package and one validation function own ranges and relations: non-positive values, the `sync-chunk-bytes` overflow (the Phase-5 IN-04 library-level guard lands here too), the `retransmit-interval=1ns` busy loop, `peer_timeout ≤ hello_interval`, `ack_timeout > ack_timeout_max`, and ignored positional arguments. Every failure exits 2 naming the parameter through the usage path; the table test covers every rule. Flag parsing moves out of `main.go` (which stays wiring). — [auto] [Validation] Q: where does validation live? → Selected: one config package + one function, exit 2, table-tested (O-02/Q-07). — **Reversibility:** costly — public flags/env.
- **D-04:** Fold the deferrals into this validation catalog: `sync_chunk_bytes` gains bounds (Phase-3 WR-02) and `value_max_bytes` keeps its datagram-ceiling relation with the library-level codec guard added (Phase-5 IN-04); the apply-failure WARN becomes rate-limited per peer/class (Phase-3 IN-01); the unbound-listing case gets its own discard class (Phase-4 New-2) and joins the metrics family. — [auto] [Deferrals] Q: do the routed deferrals land here? → Selected: yes, in the validation/metrics catalog (prior ledgers). — **Reversibility:** reversible.

### Metrics and exits (OBSV-03/04)
- **D-05:** Metric definitions are honest and each is test-confirmed: the conflict counter counts only the displacement of a local unconfirmed record by another value; a failed send is not counted as sent; `process_resident_memory_bytes` reports current RSS (not peak); the dump-duration and drift-counter semantics are stated; the four Phase-4 counters (`ListingDropped`, `SyncStreamDropped`, `ListedKeysEvicted`, `NotificationsRegressed`) get their production surface. — [auto] [Metrics] Q: how are definitions confirmed? → Selected: one documented definition per metric + a test per definition (O-03/14.4). — **Reversibility:** reversible.
- **D-06:** Exit codes are distinct and ordered: usage errors 2, startup failure 1, final-dump failure a distinct code (3); a `Serve` error still runs the bounded final dump; the signal handler is installed before dump load and bind; `--help` lists the five env-only variables. — [auto] [Exit codes] Q: which codes? → Selected: 2 usage / 1 startup / 3 final dump, ordered shutdown (O-05). — **Reversibility:** costly — operator contract.

### Boundaries and decomposition (OBSV-07/08)
- **D-07:** The dependency direction is machine-enforced with additive depguard rules: adapters may not be imported by `app`/`port`, and the observed cross-imports (`metrics`→`udrepl`, `grpcapi`→`persistfile`/`*persist.Engine`) are removed or explicitly allowed with a written justification; any boundary change is a visible `.golangci.yml` diff (the vendored body stays byte-unmodified; project rules are additive keys only). — [auto] [Boundaries] Q: which imports are ruled? → Selected: direction + the observed cross-imports (Q-05). — **Reversibility:** reversible.
- **D-08:** `internal/udprepl` splits socket, event loop, mirror and codec into separate units; `cmd/sup-kv/main.go` stays wiring only (config parsing extracted per D-03, shutdown testable). Public behavior and tests stay unchanged; the depguard direction holds after the split. — [auto] [Decomposition] Q: how far to split? → Selected: four units in udprepl + wiring-only main (Q-07). — **Reversibility:** reversible.

### Claude's Discretion
- Exact catalog field names/levels, the final-dump exit code choice (3 unless research grounds another), rule names in `.golangci.yml`, and the split file names in `udprepl`.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Decisions and requirements
- `.planning/REQUIREMENTS.md` — OBSV-01…08 verbatim acceptance
- `.planning/ROADMAP.md` — Phase 7 goal + 5 success criteria
- `docs/adr/` — the phase-1 register (14.4 metrics, S-06/S-07 log entries)
- `docs/SPEC-DEVIATIONS.md` — rows this phase closes

### Review evidence and routed deferrals
- `docs/reviews/2026-11-19-v1.0/REVIEW.md` — §9 O-01…O-05, Q-05/Q-07, §10 Q-09 (dead code is Phase 9)
- `.planning/phases/03-transport/03-REVIEW-DISPOSITION.md` — WR-02 (`sync_chunk_bytes`), IN-01 (apply-failure WARN)
- `.planning/phases/04-protocol/04-REVIEW-DISPOSITION.md` — IN-03 (four counters' surface), New-2 (unbound-listing class)
- `.planning/phases/05-persistence/05-REVIEW-DISPOSITION.md` — IN-04 (library-level value-limit guard)
- `.planning/phases/06-api-and-semantics/06-REVIEW-DISPOSITION.md` — nothing routed here; confirm

### Code
- `cmd/sup-kv/main.go` — flag/env parsing, lifecycle, logging, exit codes, signal install
- `internal/udprepl/repl.go` — socket/loop/mirror/codec mix
- `internal/metrics/metrics.go` — definitions and sources
- `.golangci.yml` — depguard project rules (additive only; vendored body byte-unmodified)
- `internal/grpcapi/limits.go`, `internal/app/persist/` — the value-limit and WARN surfaces

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- The phase-4/5/6 discard classes, per-class WARN limiter, validation surfaces and metrics families
- Phase-6 `LimitsFor` and the config package groundwork
- Doc-anchor tests (`TestOperationsDoc`, `TestAPIDoc`, `TestExamplesDoc`)

### Established Patterns
- One catalog per concern; levels named once; metrics with a documented definition and a test
- Additive `.golangci.yml` project keys over the vendored strict body

### Integration Points
- `main.go` exit paths and shutdown ordering
- `udprepl` split ↔ depguard rules
- Metrics source wiring in `main.go`

</code_context>

<specifics>
## Specific Ideas

- The exit-code contract should be one table: usage 2, startup 1, final dump 3 — with process tests asserting each.
- The lifecycle catalog and the runbook should be cross-anchored so a fictional drift condition cannot re-enter.
- The decomposition must not change any behavior: `go test` unchanged, only tests proving the new units.

</specifics>

<deferred>
## Deferred Ideas

- Dead-code deletion (`lastLocal`/`cover`, exported `Parse`, `ownHandle`) is Phase 9 per Q-09 routing; do not absorb it.
- Performance thresholds and the flake work are Phase 8.

</deferred>

---

*Phase: 7-Observability and Configuration*
*Context gathered: 2026-11-23*
