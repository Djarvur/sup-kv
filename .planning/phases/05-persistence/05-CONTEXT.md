# Phase 5: Persistence - Context

**Gathered:** 2026-11-22
**Status:** Ready for planning
**Mode:** --auto (all gray areas auto-selected; every choice logged below and reviewable in git)

<domain>
## Phase Boundary

Make the dump path honest about its limits and failures: one enforced value limit, visible load outcomes, generation integrity, bounded memory and blocking — so "a confirmed write is on disk" is a tested guarantee rather than a convention. Requirements PERSIST-01…12. Turns the two remaining red rows green: P-01 (stand + persistfile). R-01/R-02/R-03 stay green.

**Depends on:** Phase 4 — the `DumpStore` load-outcome port backs S-02 (totality: a snapshot load broadcasts its result), and the simulator seam exercises dump↔replication.

</domain>

<decisions>
## Implementation Decisions

### The value limit, one source (PERSIST-01 / P-01)
- **D-01:** Exactly one authoritative value limit: `storage.ValueMaxBytes` (default 61 440) is the only place the number is defined. It is passed by configuration into every consumer — `storage.Config.ValueMaxBytes`, the gRPC Set validator, the persistfile codec (encode and decode), and the `--value-max-bytes` / `SUPKV_VALUE_MAX_BYTES` surface. The codec's frozen `maxRecordBytes = 61 440` is removed: a record of exactly the configured limit survives a restart at any accepted flag value. An invalid value (non-positive, or above the datagram ceiling) exits 2 through the existing usage path. The datagram-ceiling relation stays one-way and documented. — [auto] [Value limit] Q: how is the limit made single-source? → Selected: one exported constant, config-passed everywhere, codec reads config (P-01/Q-04). — **Reversibility:** costly — public parameter and dump format contract.

### Load outcomes and generation integrity (PERSIST-02/03)
- **D-02:** `DumpStore.Load` returns an explicit outcome: `Prompt` (fresh start: files absent — no WARN, no ERROR), `Backup` (`.prev` used — exactly one WARN naming path and cause), `Unusable` (both generations unusable — ERROR and a non-zero metric; the unusable files are preserved under a distinct name, e.g. `.corrupt-<UTC>`, and rotation never overwrites them). The load path never deletes evidence. — [auto] [Load outcomes] Q: how are outcomes visible? → Selected: explicit outcome enum through the port + one WARN / ERROR per class (P-02). — **Reversibility:** expensive — port contract.
- **D-03:** The dump format carries a record count or terminator so a line-boundary truncation is recognized as incomplete (a `Backup`-class outcome with a WARN, not silent acceptance), and one corrupted record discards only that record (record-level checksum already exists; the count/terminator check is file-level). Format-version compatibility with the v1.0 dumps is preserved for reading; a missing trailer is treated as a truncated file. — [auto] [Generation integrity] Q: how is truncation detected? → Selected: record count/terminator in the format (P-03). — **Reversibility:** costly — on-disk format.

### Generation lifecycle (PERSIST-04/05)
- **D-04:** After a backup (`.prev`) or empty start the engine is marked dirty, so the first rotation never displaces the last good generation with a corrupt one; `SnapshotCreate` right after such a start decodes back to the in-memory state. This includes the Phase-4 review New-1 fix: every `markDirty()` wakes the dumper (`e.changed` nudge), so an idle clean dumper cannot defer installed records. — [auto] [Dirty start] Q: what is the post-backup/empty start state? → Selected: dirty + first write re-establishes the good generation (P-04). — **Reversibility:** reversible.
- **D-05:** Dump failures decay and log: each failure produces exactly one ERROR entry carrying the underlying error (ENOSPC, EACCES), retries with a growing pause up to a cap, and health behavior under a persistent failure is defined and documented. Decision: health stays `SERVING` (data service continues; the failure is a durability risk, not a serving failure), a metric counter per failure class is published, and the runbook text lands in OPERATIONS; the retry ladder is bounded and never busy-loops. — [auto] [Failure decay] Q: what happens under a persistent dump failure? → Selected: bounded backoff + ERROR/metric, health stays SERVING (P-05). — **Reversibility:** reversible.

### Resources (PERSIST-06/08/09/10)
- **D-06:** A dump cycle allocates at most two state sizes: streaming encode through the existing writer (no full second copy), with the memory model documented (measured, not guessed). — [auto] [Memory] Q: how is the two-state-size bound proven? → Selected: streaming encode + documented model (P-06). — **Reversibility:** reversible.
- **D-07:** Blocking engine calls carry a deadline: a hung fsync cannot block writes, snapshots or shutdown indefinitely. The dump/fsync work stays on the dumper goroutine; the Set confirmation wait gets a bounded timeout and on expiry returns an explicit unconfirmed outcome (class named, metric counted, health documented) instead of blocking; shutdown is bounded and abandons a stuck fsync rather than hanging. — [auto] [Deadlines] Q: how is a hung fsync bounded? → Selected: bounded confirmation wait + bounded shutdown (P-08). — **Reversibility:** expensive — RPC contract.
- **D-08:** Temp-file hygiene: orphan temp files are collected at startup (age-bounded), a newly created directory is synced at its parent, and `fsync_darwin.go` is removed in favour of one fsync path that treats EINTR as retry and ENOTSUP as a documented skip. — [auto] [Temp/darwin] Q: remove or repair the darwin helper? → Selected: remove, one portable fsync path (P-09). — **Reversibility:** reversible.
- **D-09:** Records skipped at start because the configured value limit shrank are counted and logged (one WARN with the count and class), never silently dropped; the previous generation holding them is not overwritten until the normal rotation discipline allows it, and the metric exposes the count. — [auto] [Skipped records] Q: what happens to oversize records on load? → Selected: count + WARN + metric, evidence preserved by rotation (P-10). — **Reversibility:** reversible.

### Structure and seams (PERSIST-11/12)
- **D-10:** `persist.Engine` splits along its responsibilities: loader/composition root, store facade, dump scheduler, confirmation accounting, snapshot orchestration — internal types with the public API unchanged; `port.DumpStore` returns the load outcome and failure cause (backs PERSIST-02 and the Phase-4 totality seam). — [auto] [Engine split] Q: how far to split? → Selected: five internal roles, public surface stable (Q-03/PERSIST-11). — **Reversibility:** reversible.
- **D-11:** `internal/app/persist` timers and `time.Now()` move behind the same seam the repl core uses (clock + after-func injection), so the simulator exercises the dump↔replication-notify interaction, including a dump failure under load and the skip-count path. — [auto] [Timer seam] Q: same seam as repl? → Selected: yes, shared clock/timer contract (Q-06/PERSIST-12). — **Reversibility:** reversible.

### Tie-break (PERSIST-07)
- **D-12:** Equal version with different content is resolved by a **deterministic** tie-break (content-based, not the Spec's random tail), recorded in a new ADR (`docs/adr/0012-equal-version-tiebreak.md`), implemented in the merge path and tested. Rationale: deterministic simulation replay and byte-identical dumps are project invariants; a random tail would break both. The ADR records the deviation and its consequence. — [auto] [Tie-break] Q: random tail or deterministic? → Selected: deterministic, ADR-recorded (P-07). — **Reversibility:** costly — data model.

### Claude's Discretion
- Exact record-count/terminator encoding, the corrupt-file naming scheme, backoff constants, the confirmation timeout default, the orphan-temp age bound, and the internal split's type names.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Decisions and requirements
- `.planning/REQUIREMENTS.md` — PERSIST-01…12 verbatim acceptance
- `.planning/ROADMAP.md` — Phase 5 goal + 5 success criteria
- `docs/SPEC-DEVIATIONS.md` — rows this phase closes (P-01…P-10 family)
- `docs/adr/` — ADR 0005…0011 (phase-1 register); the new ADR 0012 lands here

### Review evidence
- `docs/reviews/2026-11-19-v1.0/REVIEW.md` — §6 P-01…P-10, Q-03/Q-04/Q-06
- `.planning/phases/03-transport/03-REVIEW.md` — WR-02 (`sync_chunk_bytes` is Phase 7's, not this phase's) and IN-06 (doc claims)
- `.planning/phases/04-protocol/04-REVIEW.md` + `04-REVIEW-DISPOSITION.md` — New-1 (markDirty nudge, Phase 5) must be folded into PERSIST-04

### Code
- `internal/app/persist/engine.go` — current Engine (Save/Load/Snapshot/Restore, confirmation accounting)
- `internal/persistfile/dumpfile.go`, `fsops.go`, `snapshot.go` — file operations and rotation
- `internal/app/storage/store.go` — `defaultValueMaxBytes` (the limit to make single-source)
- `cmd/sup-kv/main.go` — `--value-max-bytes`/`SUPKV_VALUE_MAX_BYTES`, `defaultValueMaxBytes`, datagram-ceiling validation
- `internal/stand/stand_red_test.go` — P-01/F15 frozen acceptance (flip via TSV only)
- `internal/persistfile/red_value_limit_test.go` — P-01 frozen acceptance

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Phase 3/4 seams: Outbox/confirmation accounting, DumpStore port, clock/timer seam pattern from the repl core
- Frozen red tests for P-01 and the red runner/expectation table
- Strict lint and depguard boundaries

### Established Patterns
- Core goroutine-free decisions; adapters own I/O and timers
- Errors classified and counted; one log line per class
- Deterministic simulator replay stays byte-identical

### Integration Points
- `scripts/red-expectations.tsv` — the two P-01 rows flip `red → green`
- `internal/persistfile` codec — the frozen 61 440 must become config-driven
- `internal/grpcapi` Set/snapshot paths — confirmation timeout outcome

</code_context>

<specifics>
## Specific Ideas

- P-01 is already pinned by frozen tests in two packages; the phase's job is code until the rows flip, then the TSV change only.
- The `DumpStore` port returning an outcome enum is the same seam S-02 needs; do not invent a second load result type.
- `fsync_darwin.go` exists today; removing it must keep the darwin build/vet gate green.

</specifics>

<deferred>
## Deferred Ideas

- `sync_chunk_bytes` bounds and the four new counters' metrics surface are Phase 7's (OBSV/robustness), per the Phase-4 ledger.
- `TestCore_keyCountCostFlat` ratio flake is Phase 8's.

</deferred>

---

*Phase: 5-Persistence*
*Context gathered: 2026-11-22*
