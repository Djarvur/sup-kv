# Phase 2: Engine — HLC Versioning and LWW Store - Context

**Gathered:** 2026-11-02
**Status:** Ready for planning

<domain>
## Phase Boundary

The pure, I/O-free decision core as a Go library: the Hybrid Logical Clock, the `RecordVersion` packed into UUID v7, and the LWW-Element-Map store (Get/Set/Delete/create-if-absent, tombstones, 60 KiB value limit, per-key merge by greater version) — plus the property tests that prove the correctness laws (merge commutative/associative/idempotent; HLC monotonic under adversarial clock input; version uniqueness per write). Covers CORE-01..CORE-07, VERS-01..VERS-05, TEST-01, TEST-02. No networking, no disk, no gRPC — those belong to Phases 3–5. The package boundary from Phase 1 applies: `internal/app` imports no `net`/`os`/`syscall`/`google.golang.org/*` (depguard rule `sup-kv-app-no-io`).

</domain>

<decisions>
## Implementation Decisions

### Reporting events from the pure core
- **D-01:** The store reports merge losers by return value: `Merge` returns the list of losing records (the conflict-journal source, CORE-07), and the HLC methods return a drift signal when the wall clock runs ahead beyond `drift_warn` (VERS-05). No callback handlers, no events accumulated inside the library, no hidden side effects — callers (Phases 4/6) decide where losers and drift warnings are recorded. — **Reversibility:** costly — changes the signature every later consumer of `Merge`/`Tick`/`Update` codes against.

### HLC time source
- **D-02:** The clock receives a `now func() time.Time` in its constructor. Production passes `time.Now`; tests pass a controlled function whose clock jumps backwards/forwards and sticks, driving the adversarial scenarios of VERS-03/TEST-02. Method signatures stay clean (`Tick()` / `Update(remote)`); the injection point is exactly one.

### Interfaces in internal/port
- **D-03:** Phase 2 declares no interfaces in `internal/port` — concrete types only in `internal/app`. Each later phase (3 dump, 4 replication, 5 gRPC) declares in `internal/port` the interface it actually needs, per the note already recorded in `port/doc.go` ("interfaces appear with the phases that need them") and Go's consumer-side interface practice. — **Reversibility:** reversible — interfaces can be declared later without touching the core.

### Property-test tooling
- **D-04:** Property tests are written with hand-rolled deterministic generators on the standard library only (TEST-01/TEST-02). Generators print their seed; a failing run is replayed exactly by re-running with the same seed; counterexample minimization is manual (halving). No new dependencies — `go.mod` keeps zero direct runtime requires, consistent with Phase 1 D-03 and the project's no-unnecessary-dependencies principle.

### Claude's Discretion
- Exact API signatures and type placement within `internal/app` (one package vs subpackages), the `Conflict` record fields, sentinel errors vs other error representation (the future gRPC mapping in Phase 5 stays unaffected either way), and the choice of deterministic PRNG for the generators — planner decides following go-ultimate conventions.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Design and Requirements
- `docs/IDEA.md` — authoritative, self-contained design document: HLC algorithm (Tick/Update, counter overflow, init from dump max), UUID v7 packing (48-bit `unix_ts_ms` = `l`, 12-bit `rand_a` = counter, `NodeID` in the high bits of `rand_b`), LWW-Element-Map semantics, checksum, parameter defaults
- `.planning/REQUIREMENTS.md` — CORE-01..CORE-07, VERS-01..VERS-05, TEST-01, TEST-02 are the fourteen requirements this phase covers
- `.planning/ROADMAP.md` §Phase Details → Phase 2 — goal and the four success criteria (the verification contract)
- `.planning/PROJECT.md` — Context (decided design), Constraints (zero-dependency principle, quality gates)

### Mandated Conventions
- go-ultimate skill (github.com/Djarvur/go-ultimate, v0.15.0; available locally as the `go-ultimate` skill) — engineering policy and testing conventions governing all implementation
- `AGENTS.md` — layout rules (adapters appear with the phases that need them), dependency policy, lint gate deviations

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/app/doc.go`, `internal/port/doc.go` — Phase 1 placeholders documenting where this phase's code lands and the boundary rules
- `internal/app/app_test.go` — the established test pattern: external `app_test` package, `t.Parallel()`, table-driven; the walking-skeleton test is superseded by real engine tests

### Established Patterns
- Depguard gate (`sup-kv-app-no-io`, `sup-kv-port-stdlib-only`) in `.golangci.yml` machine-enforces the boundary this phase must live inside
- Strict golangci-lint + go-ultimate from first commit; `make lint` / `make test -race` red blocks merge

### Integration Points
- Phase 3 (dump) serializes records — record fields need to be reachable for JSON encoding
- Phase 4 (replication) merges remote records and consumes `Merge`'s returned losers (D-01)
- Phase 5 (gRPC) maps store errors to RPC codes

</code_context>

<specifics>
## Specific Ideas

- The `now` constructor parameter is the single seam for wall-clock input; every adversarial-clock property test goes through it (D-02)
- Failing property runs must be exactly reproducible: seed printed, seed re-runnable (D-04)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 2-Engine — HLC Versioning and LWW Store*
*Context gathered: 2026-11-02*
