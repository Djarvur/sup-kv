# Phase 8: Performance - Context

**Gathered:** 2026-11-23
**Status:** Ready for planning
**Mode:** --auto (all gray areas auto-selected; every choice logged below and reviewable in git)

<domain>
## Phase Boundary

Performance claims become measurements with documented conditions: the write-latency model for
both profiles, the 1000 writes/s burst as a standing tested bound, disk-write amplification on
target-grade media (or the gap recorded), the memory model from stand measurements, absent-peer
latency proven near live-peer latency, and `make perf` with thresholds over the stand suite.
Requirements PERF-01…PERF-06. Also owns every flake class routed here by phases 3-7: the
wall-clock ratio gates, the virtual-time dump-notify harness, the UDP pick-then-bind race, the
port-contention and lifecycle flakes, the `killAndReap` diagnostic gap, the paced-replay double
copy (04 New-3) and the slice-sizing test teeth (04 New-5).

**Depends on:** Phase 7 (all fixes in; `make perf` builds on the stabilized stand).

</domain>

<decisions>
## Implementation Decisions

### Latency model (PERF-01)
- **D-01:** One documented latency model covers both profiles with their measurement conditions.
  The default profile: under sustained write flow the quiet window never arrives, so the dump
  runs on `dump_max_wait` and write latency reaches `dump_max_wait` + dump time by design; a
  single sequential client does at most ~14 writes/s because `dump_debounce` waits for quiet.
  The fast-response profile (`Debounce` 1 ms / `MaxWait` 50 ms) is documented with its disk cost
  and carries the "~10 ms under 1000 writes/s" claim, tied to the exact benchmark conditions it
  was measured under (`internal/grpcapi/bench_test.go`'s parameters), never stated standalone.
  — [auto] [Model] Q: where do the two profiles live? → Selected: one model section, both
  profiles, each number with its conditions (S-FAIL-15, 14.3). — **Reversibility:** reversible.

### Burst bound (PERF-02)
- **D-02:** The 1000 writes/s burst over distinct keys becomes a standing test with a bound, not
  a folklore number: the XPORT-02(3) scenario runs in the standing suite, the docs state what is
  guaranteed (dump cadence, one datagram per write, no loss of confirmed writes) and what is not
  (distinct-key bursts can storm retransmissions; only hot-key bursts hold the rate today), and
  the `S-FAIL-15` registry row is closed against the shipped test. — [auto] [Burst] Q: stand the
  burst up or document it? → Selected: standing test + explicit guarantee/non-guarantee text
  (R-02, PERF-03). — **Reversibility:** reversible.

### Amplification and memory (PERF-03, PERF-04)
- **D-03:** Disk-write amplification is measured with a documented method (bytes written by the
  dumper per logical byte written by clients, on the available media, with the state size and
  cadence named) and published with a hardware description, or the gap is recorded honestly
  against Spec open question #2 — the design is not changed: decision 14.3 (full-dump policy)
  stands, recorded in ADR 0003. — [auto] [Amplification] Q: measure or record the gap? →
  Selected: measure on target-grade media where available, else record the gap with the method
  and the missing hardware (PERF-04). — **Reversibility:** reversible.
- **D-04:** The memory model is documented from stand measurements: one dump cycle allocates a
  small multiple of the state size, RSS tracks the state plus that transient, and the measured
  points (state size → RSS, one dump cycle) are published with their hardware description and
  tied to the streaming-encode semantics delivered by PERSIST-06 (Phase 5). — [auto] [Memory]
  Q: formula or points? → Selected: formula + measured points with conditions (P-06, PERF-05).
  — **Reversibility:** reversible.

### Absent-peer latency (PERF-05)
- **D-05:** Write latency with an absent peer and a growing unconfirmed set is proven near
  live-peer latency by a benchmark on the same fix as XPORT-02; the benchmark states the unacked
  set size, the peer state and the ratio bound, and the paired measurement at a live peer is
  stored in the same test so a regression fails by name. While here, 04 New-3 is fixed: the
  paced replay path must not copy `rec.Data` twice before discarding it. — [auto] [Absent peer]
  Q: benchmark shape? → Selected: paired benchmark, absent vs live peer, ratio bound (PERF-06,
  04-New-3). — **Reversibility:** reversible.

### make perf (PERF-06)
- **D-06:** `make perf` runs the stand suite with thresholds defined in one place, and every
  published number carries a hardware description block (CPU, RAM, disk, load). Thresholds are
  documented bounds (not stored profiles, not self-pacing), and CI keeps running the threshold-
  free smoke target as today; `make perf` is the named operator-facing target. — [auto] [Perf
  target] Q: one target or profiles? → Selected: one `make perf` with central thresholds and
  hardware blocks (D-13/D-14 stay for CI smoke). — **Reversibility:** reversible.

### Flake classes routed here (phases 3-7)
- **D-07:** The routed flake classes are closed, not muted: the two wall-clock ratio gates
  (`TestCore_keyCountCostFlat`, `TestCore_unackedCostFlat`) lose their load sensitivity (drop
  `t.Parallel` where it contaminates, or replace the wall-clock ratio with a deterministic
  operation-count assertion), `TestScenario_dumpNotify` gets deterministic sequencing, the
  `freeUDPPort` pick-then-bind class is structurally removed, the port-contention/lifecycle
  flakes get a serialized or ephemeral-port cure, and `killAndReap` reports an already-exited
  child with its captured stderr/exit status. 04 New-5 strengthens
  `TestChunkSyncEntries_capacityFollowsEntries` to pin the slice-sizing claim. — [auto] [Flakes]
  Q: fix or record? → Selected: fix the classes with deterministic assertions; a bound is never
  relaxed to green (04 New-3/New-5, 05/06/07 deferred-items). — **Reversibility:** reversible.

### Claude's Discretion
- Where the performance model lives (new `docs/PERFORMANCE.md` vs a section of OPERATIONS), the
  exact threshold values and their basis, benchmark names and the shape of the hardware block,
  and the mechanism for deterministic ratio gates (operation counters vs serialized measurement).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Decisions and requirements
- `.planning/REQUIREMENTS.md` — PERF-01…PERF-06 verbatim acceptance
- `.planning/ROADMAP.md` — Phase 8 goal + 5 success criteria
- `docs/adr/0003-dump-policy-unchanged.md` — decision 14.3 (full-dump policy stays; the phase
  documents and measures it, never changes it)
- `.planning/PROJECT.md:43,80,82,97` — the "~10 ms under 1000 writes/s" claim and the target
  workloads; the claim must be tied to its measurement conditions
- `docs/SPEC-DEVIATIONS.md` — S-FAIL-15 (burst bound) and the rows this phase closes

### Review evidence
- `docs/reviews/2026-11-19-v1.0/REVIEW.md` — §10 PERF-01…PERF-06 (the measurements and their
  conditions) and §13 stand findings R-01/R-02
- `.planning/phases/04-protocol/deferred-items.md` — ratio-gate flake (IN-03), freeUDPPort,
  grpcurl walkthrough stall; `04-REVIEW-DISPOSITION.md` New-3 (paced-replay double copy) and
  New-5 (slice-sizing test teeth)
- `.planning/phases/05-persistence/deferred-items.md` — dumpNotify virtual-time harness
- `.planning/phases/06-api-and-semantics/deferred-items.md` — port-contention and lifecycle flakes
- `.planning/phases/07-observability-and-configuration/deferred-items.md` — ratio gates repeat,
  dumpNotify repeat, `killAndReap` diagnostic gap

### Code
- `internal/app/repl/bench_test.go` — the two wall-clock ratio gates to fix
- `internal/app/persist/bench_test.go`, `internal/grpcapi/bench_test.go` — dump-10k and
  confirmation-latency benchmarks; the fast-profile parameters
- `internal/udprepl/` — the benchmark surface for the wire path
- `internal/stand/` — the stand suite `make perf` wraps, and the scenario harnesses
- `Makefile` — `bench`, `bench-smoke`, `race-soak`, `red-run`; `make perf` is new
- `internal/app/persist/engine.go` (dump cadence and byte accounting), `internal/config/`
  (Debounce/MaxWait knobs)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- The stand suite (`internal/stand/`) with real binaries, metrics and scenarios — `make perf`
  wraps it rather than inventing a second harness
- Existing benchmarks behind `make bench`/`make bench-smoke` (persist, udprepl, grpcapi), and
  the phase-4/5 core benchmarks
- Phase-7 metrics (`supkv_dump_duration_seconds`, dump counters, RSS) as the measurement sources
- Doc-anchor tests (`TestOperationsDoc`) as the shape for test-enforcing published numbers

### Established Patterns
- Thresholds in one place; CI keeps a threshold-free smoke target (D-13/D-14)
- Deterministic simulator scenarios over wall-clock sleeps; virtual time in the core
- A benchmark's claim is pinned by a test that fails by name with both measured values

### Integration Points
- `Makefile` targets and the CI workflow
- The dump engine's byte accounting ↔ the amplification method
- The core's unacked/pending accounting ↔ the absent-peer benchmark

</code_context>

<specifics>
## Specific Ideas

- The published-numbers rule: every number in docs sits next to a hardware block and the command
  that produced it, so a reader can rerun it.
- `make perf` should print the hardware block it ran on and fail with the threshold name, not a
  bare number.
- The ratio-gate fix should keep the *claim* (inbound handling is flat in stored keys; the event
  loop is flat in unacked size) while removing the wall-clock dependence.

</specifics>

<deferred>
## Deferred Ideas

- Dead-code deletion and the registry resweep (`New-3` from the Phase-7 re-verification) stay in
  Phase 9.
- Any dump-policy redesign is out of scope: decision 14.3 stands (ADR 0003); only documentation
  and measurement change here.

</deferred>

---

*Phase: 8-Performance*
*Context gathered: 2026-11-23*
