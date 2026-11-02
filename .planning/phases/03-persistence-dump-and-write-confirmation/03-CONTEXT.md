# Phase 3: Persistence — Dump and Write Confirmation - Context

**Gathered:** 2026-11-02
**Status:** Ready for planning

<domain>
## Phase Boundary

State survives restarts and a client-confirmed write is always on disk: the human-readable JSON dump of the full LWW map (tombstones included), atomic writes with the `.prev` fallback chain, debounced dumps with the write-confirmation watermark, and named snapshots with restore-as-writes. Covers PERS-01..PERS-05 and the persistence half of the VERS-04 restart contract (the dump loader feeds `clock.NewClock`). No replication, no gRPC/API surface, no metrics — those belong to Phases 4–6. Snapshot create/load ship as library-level operations here; the gRPC wiring arrives in Phase 5.

</domain>

<decisions>
## Implementation Decisions

### Architecture split (pure core vs file adapter)
- **D-01:** The dump codec — JSON assembly and parsing, per-record checksum verification, and computing the startup watermark (max `l` plus max counter at that `l`, VERS-04) — is a pure component inside `internal/app`, property-tested without disk. The file machinery (temp file, fsync, rename, parent-directory fsync, `.prev` rotation) lives in an adapter outside the app boundary, per the depguard rule `sup-kv-app-no-io`. — **Reversibility:** costly — the port interface is what Phases 4–5 code against.
- **D-02:** The port seam carries whole dump bytes. Saving: the core hands the adapter one complete byte block and the adapter's atomic-write interface knows nothing about records. Loading: the adapter resolves the fallback chain main → `.prev` → empty start and returns the first generation that reads as bytes (a truncated or unparseable file counts as unreadable → next generation); the codec then validates per-record checksums and discards bad records (PERS-02). — **Reversibility:** costly — same contract consumers.

### Write confirmation (PERS-04)
- **D-03:** The confirmation mechanism lives in the core: a confirmed-version watermark plus a wait primitive ("a completed dump has covered version X") that unblocks writers. No callbacks, no events — callers observe results, mirroring Phase 2 D-01.
- **D-04:** Bounded max wait defaults to **1 second**: even under a continuous write stream a dump must start no later than 1 s after the oldest unconfirmed write, so waiters are confirmed within ~1 s plus dump duration (PERS-03; success criterion 3). Exact parameter name is planner's discretion; `docs/IDEA.md` §Parameters gains the entry.
- **D-05:** On dump failure all current waiters immediately receive the error (Unavailable at the API layer, Phase 5); the write stays in memory and replicates later (PERS-04). The next dump starts on the next change or the debounce timer — no auto-retry loops holding writers hostage.

### Dump file format (PERS-01)
- **D-06:** One JSON object: a header carrying the format version, then the records one per line, sorted by key — readable by eye, greppable per record, and byte-stable across generations of the same state.
- **D-07:** Per-record checksum is CRC32 — uniform with the replication integrity story, within the accepted trust model (docs/IDEA.md §Trust model).

### Snapshots (PERS-05)
- **D-08:** Named snapshots live beside the dump in `<dump_path>.snapshots/<name>` — no extra launch parameter.
- **D-09:** Restore replays only the snapshot's keys as ordinary writes/deletes with fresh versions (Deleted → Delete); keys absent from the snapshot are untouched — their newer versions win by normal LWW. Literal "replay as ordinary writes", not a full state replacement.
- **D-10:** Creating a snapshot first forces the current dump to complete (so the copy contains every confirmed write), then copies the finished dump byte-for-byte under the name; snapshots read back with the same codec.

### Claude's Discretion
- Adapter package name and location (an `internal/` package appearing with this phase per AGENTS.md layout rules), the exact interface names and signatures in `internal/port`, JSON field spellings beyond PERS-01's named fields, the max-wait parameter name, temp-file naming, and the concurrency internals of the waiter primitive — planner decides per go-ultimate conventions.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Design and Requirements
- `docs/IDEA.md` — §Persistence (dump contents, atomic write, `.prev` fallback, debounce, confirmed writes, snapshots), §Node lifecycle (startup load, HLC init from dump max), §Parameters (`dump_path` required, `dump_debounce` 50 ms, plus the new bounded max wait, D-04), §Trust model (CRC32 accepted)
- `.planning/REQUIREMENTS.md` — PERS-01..PERS-05 are this phase's requirements; VERS-04 fixes the restart watermark contract
- `.planning/ROADMAP.md` §Phase Details → Phase 3 — goal and the four success criteria (the verification contract)
- `.planning/PROJECT.md` — Context (decided design), Constraints (zero-dependency principle, quality gates)

### Mandated Conventions
- `AGENTS.md` — layout rules (adapters appear with the phases that need them), dependency policy, depguard boundary
- `.planning/phases/02-engine-hlc-versioning-and-lww-store/02-CONTEXT.md` — Phase 2 decisions still binding: D-01 (results, not callbacks), D-02 (`now` seam), D-03 (port interfaces appear with phases)
- go-ultimate skill (v0.15.0, available locally) — engineering and testing policy

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/app/storage` — the LWW-Element-Map store the dump serializes; `Merge`/`Set`/`Delete` produce the records; the store consumes the `storage.Clock` interface
- `internal/app/clock` — `NewClock(now, initialL, initialC, driftWarn)` is the watermark seed target (VERS-04); the loader feeds it max `l` plus max counter at that `l`
- `internal/app/version` — `RecordVersion` (UUID v7) with `Compare`; powers both the dump's version fields and the watermark computation
- `internal/app/internal/testutil` — deterministic seeded generators for the codec property tests (round-trip, checksum discard, watermark laws)

### Established Patterns
- Depguard gate (`sup-kv-app-no-io`) machine-enforces where the file adapter must NOT live
- Strict golangci-lint + go-ultimate from first commit; `make lint` / `go test -race` red blocks merge
- Phase 2's package-per-concern split (`version`, `clock`, `storage`) sets the naming precedent for this phase's packages

### Integration Points
- Phase 4 (replication) triggers dumps on remote merges and keeps serving after dump failure (D-05)
- Phase 5 (gRPC) maps dump failure to `Unavailable` and exposes snapshot create/load
- Phase 6 (hardening) verifies crash/durability scenarios, including the macOS F_FULLFSYNC policy already recorded in STATE.md's decision log

</code_context>

<specifics>
## Specific Ideas

- The dump loader computes the FULL watermark — max `l` plus max counter at that `l` — not max `l` alone (VERS-04, REQUIREMENTS wording is authoritative)
- Success criterion 3 is a test target: ~1000 writes/s sustained bursts, every write confirmed within bounded latency
- macOS F_FULLFSYNC durability policy is a Phase 6 crash-scenario verification item, not this phase's blocker

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 3-Persistence — Dump and Write Confirmation*
*Context gathered: 2026-11-02*
