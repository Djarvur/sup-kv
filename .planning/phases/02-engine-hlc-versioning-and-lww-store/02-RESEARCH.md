# Phase 2: Engine — HLC Versioning and LWW Store - Research

**Researched:** 2026-11-02
**Domain:** Pure Go library: Hybrid Logical Clock, UUIDv7 (RFC 9562) version packing, LWW-Element-Map CRDT store, stdlib-only property-based testing
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**D-01 — Reporting events from the pure core:** "The store reports merge losers by return value: `Merge` returns the list of losing records (the conflict-journal source, CORE-07), and the HLC methods return a drift signal when the wall clock runs ahead beyond `drift_warn` (VERS-05). No callback handlers, no events accumulated inside the library, no hidden side effects — callers (Phases 4/6) decide where losers and drift warnings are recorded." (Reversibility: costly)

**D-02 — HLC time source:** "The clock receives a `now func() time.Time` in its constructor. Production passes `time.Now`; tests pass a controlled function whose clock jumps backwards/forwards and sticks, driving the adversarial scenarios of VERS-03/TEST-02. Method signatures stay clean (`Tick()` / `Update(remote)`); the injection point is exactly one."

**D-03 — Interfaces in internal/port:** "Phase 2 declares no interfaces in `internal/port` — concrete types only in `internal/app`. Each later phase (3 dump, 4 replication, 5 gRPC) declares in `internal/port` the interface it actually needs… and Go's consumer-side interface practice." (Reversibility: reversible)

**D-04 — Property-test tooling:** "Property tests are written with hand-rolled deterministic generators on the standard library only (TEST-01/TEST-02). Generators print their seed; a failing run is replayed exactly by re-running with the same seed; counterexample minimization is manual (halving). No new dependencies — `go.mod` keeps zero direct runtime requires, consistent with Phase 1 D-03 and the project's no-unnecessary-dependencies principle."

> Note: D-04 **overrides** the project STACK.md suggestion of `pgregory.net/rapid` as a test-only dependency. Research follows the locked decision: no property-test library, no `google/uuid` — hand-rolled packing verified against RFC 9562 in tests.

### Claude's Discretion
"Exact API signatures and type placement within `internal/app` (one package vs subpackages), the `Conflict` record fields, sentinel errors vs other error representation (the future gRPC mapping in Phase 5 stays unaffected either way), and the choice of deterministic PRNG for the generators — planner decides following go-ultimate conventions."

### Deferred Ideas (OUT OF SCOPE)
"None — discussion stayed within phase scope."
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CORE-01 | Get: missing or tombstoned key returns NotFound | Sentinel-error pattern (lint-probe verified); store semantics table below |
| CORE-02 | Set creates or updates under LWW | HLC Tick + pack (VERS findings); version-generation flow |
| CORE-03 | create-if-absent: AlreadyExists for any existing record incl. tombstone | Tombstone visibility pitfall (Pitfall 1); sentinel errors |
| CORE-04 | Delete writes fresh-version tombstone | Same Tick path as Set with `Deleted: true` |
| CORE-05 | Values over `value_max_bytes` (default 60 KiB) rejected InvalidArgument | Config-struct pattern; sentinel error Phase 5 maps to InvalidArgument |
| CORE-06 | Merge laws: per key greater RecordVersion wins; commutative, associative, idempotent | CvRDT join-semilattice theory (Shapiro); merge-law property-test design below |
| CORE-07 | Every displaced version recorded (merge exposes loser) | D-01 return-value contract; Conflict record shape |
| VERS-01 | Byte-wise UUID comparison equals `(l, c, NodeID)` order | RFC 9562 field layout (verified, §5.7/§6.2/§6.11); ordering-equivalence property test |
| VERS-02 | Unique per write by construction | Tick strict-increase property test; NodeID tie-break |
| VERS-03 | Monotonic under backward/forward jumps, stuck clock, counter overflow | Kulkarni Tick/Update rules (quoted below); adversarial clock generator design |
| VERS-04 | Init at startup from dump's max `l`, not wall clock | Constructor-with-initial-`l` seam; restart property test |
| VERS-05 | Drift beyond `drift_warn` (60s) → warning | D-02 drift signal; pure core returns signal, callers log (scoping note below) |
| TEST-01 | Property tests prove merge commutative/associative/idempotent | Hand-rolled generator + seed-replay pattern (D-04) |
| TEST-02 | HLC property tests: monotonicity, post-receive ordering, uniqueness, counter-reset, restart-from-dump-max | Adversarial clock generator; math/rand/v2 PCG seeding |
</phase_requirements>

## Summary

Phase 2 builds the entire decision core as one pure Go package (`internal/app`) with **zero new dependencies** — every algorithm is tens of lines against a published, verifiable spec. The three technical pillars are all fully specified by authoritative sources: (1) the HLC Tick/Update rules from Kulkarni et al. 2014, (2) the UUIDv7 field layout from RFC 9562 §5.7 which *explicitly blesses* a dedicated counter in `rand_a` (§6.2 Method 1: counter "MUST be positioned immediately after the embedded timestamp", "SHOULD be at least 12 bits but no longer than 42 bits" — sup-kv's 12-bit HLC counter `c` is exactly the floor of that range), and (3) the LWW-Element-Map merge as a join-semilattice (Shapiro et al. 2011), whose per-key max-version join is commutative/associative/idempotent *by construction* — the property tests prove the implementation didn't break the construction.

The one design-derived subtlety is **counter overflow**: a 12-bit counter holds 4096 values, so more than 4096 versioned writes inside one wall-clock millisecond (stuck clock, or ~1000 writes/s bursts compressed below 1 ms resolution) must bump `l` by 1 ms and reset `c` to 0 rather than wrap — wrapping would break both uniqueness and monotonicity. This is drivable by a fake `now func() time.Time` (D-02) and is a required TEST-02 case.

Tooling-wise the phase is stdlib-only by locked decision (D-04): `math/rand/v2` with `rand.New(rand.NewPCG(seed1, seed2))` for deterministic, replayable generators (the package-level functions are auto-seeded and NOT replayable — a local `*rand.Rand` instance is mandatory), table-driven vanilla `testing` in the external `app_test` package, and sentinel errors — which I empirically verified pass the vendored strict golangci-lint 2.15.0 gate this session (probe: `var ErrNotFound = errors.New("not found")` produced no `gochecknoglobals` finding).

**Primary recommendation:** single `internal/app` package with four production files (`clock.go`, `version.go`, `store.go`, `errors.go`), concrete types only (D-03), Merge applies externally-versioned records (remote records keep *their* versions — this makes merge-law property tests clock-free and fully deterministic), and the property suite seeds every generator via a printed seed with a `-seed`-style replay hook.

## Project Constraints (from .claude/CLAUDE.md + AGENTS.md)

- **Tech stack**: "HLC, UUIDv7 packing, LWW map, UDP reliability layer all implemented in-repo (each is tens of lines against a known algorithm) — product principle: no invented consensus, correctness from proven components" [VERIFIED: .claude/CLAUDE.md Constraints]
- **Quality**: strict golangci-lint + go-ultimate conventions from first feature commit; red gate blocks merge. `make lint` and `make test` (`go tool golangci-lint run ./...`, `go test -race -cover ./...`) before pushing [VERIFIED: AGENTS.md Conventions, Makefile]
- **Go version**: "as pinned in `go.mod` (`go 1.27`). Do not raise it without an ADR" [VERIFIED: AGENTS.md Conventions]; go.mod says `go 1.27.1` [VERIFIED: go.mod:3]; local toolchain go1.27.3 (compatible, newer patch)
- **Depguard boundary (machine-enforced)**: `internal/app` non-test code may not import `net`, `os`, `syscall`, `google.golang.org/` [VERIFIED: .golangci.yml:68-80]; `internal/port` restricted to `$gostd` + own module [VERIFIED: .golangci.yml:81-88]. `time`, `sync`, `bytes`, `hash/crc32`, `math/rand/v2` are all legal in `internal/app`. Test files are exempt from the boundary rules (`'!$test'`) [VERIFIED: .golangci.yml:71]
- **math/rand v1 is denied**: `pkg: math/rand$` / "use math/rand/v2" [VERIFIED: .golangci.yml:34-35]
- **No lint suppressions without a written, linter-specific justification** (nolintlint) [VERIFIED: AGENTS.md deviation 3]
- **Layout**: `cmd/sup-kv/` holds no testable logic; adapters appear only with the phases that need them; do not create empty adapter directories [VERIFIED: AGENTS.md Layout]
- **Go version gating**: go-ultimate — use features up to the go.mod version (`any`, `errors.Is/As`, `slices`/`maps`/`cmp`, `sync.WaitGroup.Go` on 1.25+)
- **go-ultimate testing**: vanilla `testing`, external `package app_test`, `t.Parallel()` by default, table-driven, no assertion library, helpers use `t.Helper()` + `t.Cleanup`, no `defer` cleanup in helpers, verify behavior not implementation
- **go-ultimate errors**: sentinel errors via `errors.New` "for expected error conditions callers must handle", "keep sentinels few and stable — they are public API"; never `==` comparison, always `errors.Is`; `error` last return value; wrap with `%w` when the sentinel is contract
- **go-ultimate public API discipline**: document every exported identifier; doc comments must state concurrency safety, ownership/aliasing, sentinel errors to match, and non-obvious parameter constraints
- **PR-only landing** (CI-03): every change lands through a PR from a feature branch

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| HLC timekeeping (Tick/Update, drift) | Library core (`internal/app`) | — | Pure state arithmetic over injected `now func() time.Time` (D-02); no I/O |
| RecordVersion pack/compare (UUIDv7) | Library core (`internal/app`) | — | Bit manipulation on `[16]byte` against RFC 9562 |
| LWW-Element-Map store + tombstones | Library core (`internal/app`) | — | In-memory map + RWMutex; merge = per-key max-version join |
| Conflict-journal recording | Later phases (4/6 adapters) | Library core (returns losers) | D-01: core returns losers by value; callers decide persistence |
| Drift-warning logging | Later phases (4/6 adapters) | Library core (returns signal) | D-01: no hidden side effects in the pure core |
| Dump serialization of records | Phase 3 (`internal/` adapter) | — | Record fields must stay reachable/exported for JSON |
| Error → RPC code mapping | Phase 5 (gRPC adapter) | — | Sentinel errors chosen so mapping is unaffected |
| Property-test generators | Test files in `internal/app` | — | Stdlib-only, seeded, replayable (D-04); live in `_test.go`, never production |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib (`time`, `sync`, `bytes`, `errors`, `fmt`, `hash/crc32`) | go 1.27.1 [VERIFIED: go.mod:3] | The entire engine | Zero-dependency principle; every algorithm is tens of lines against a published spec |
| `math/rand/v2` (PCG) | stdlib (Go 1.22+) | Deterministic PRNG for property-test generators | [CITED: go.dev/blog/randv2] `rand.New(rand.NewPCG(seed1, seed2))` — local instance, two uint64 seeds, exact replay; PCG is the fastest stdlib generator [CITED: lemire.me 2025 benchmark] |
| `testing` (vanilla) | stdlib | Unit + property tests | go-ultimate mandate; `-race` in CI and locally |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `testing/synctest` | stdlib (stable 1.25+) | Virtual-time bubbles | Not needed in Phase 2 (no timers/sleeps in the pure core); becomes relevant Phases 3–4 |
| `hash/crc32` | stdlib | Record Checksum field | Only if the planner includes the checksum helper now (see Open Question 3); validation itself is Phase 3 (PERS-02) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled generators (D-04, locked) | `pgregory.net/rapid` v1.3.0 | rapid gives integrated shrinking, but D-04 locks stdlib-only; not an option this phase |
| Hand-rolled UUIDv7 pack | `github.com/google/uuid` v1.6.0 | `NewV7` does not implement counter-in-`rand_a` HLC packing; adding it would create a direct require, violating D-04's "no new dependencies"; not an option |
| `slices.Compare`/`bytes.Compare` on `[16]byte` | Manual field-by-field compare | stdlib compare is correct and faster; field compare only as the *reference* inside the ordering property test |

**Installation:**
```bash
# Nothing. Zero new dependencies this phase (D-04).
# go.mod keeps exactly its current two tool directives and zero direct runtime requires.
```

**Version verification:** No packages to verify — the phase adds none. Existing pins confirmed on disk: golangci-lint 2.15.0 (`go tool golangci-lint version` → "2.15.0 built with go1.27.3"), Go toolchain go1.27.3 against `go 1.27.1` directive. [VERIFIED: local probe this session]

## Package Legitimacy Audit

**This phase installs zero external packages.** D-04 locks property testing to the standard library and D-03 (project-level, Phase 1) keeps `go.mod` at zero direct runtime requires. The two stack-research candidates that touch this phase (`pgregory.net/rapid`, `github.com/google/uuid`) are **deliberately excluded by locked decision**, not by legitimacy verdict.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| (none) | — | — | — | — | — | No installs; gate not applicable |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
                       injected at construction (D-02)
        ┌────────────────────────── now func() time.Time ─────────────────────┐
        │                                                                    │
        ▼                                                                    │
┌───────────────┐  Tick()  ┌──────────────┐  (l, c)  ┌───────────────────┐   │
│ Wall clock    │◄─────────│ HLC Clock    │─────────►│ RecordVersion     │   │
│ (prod=time.Now│          │ (l, c) state │          │ pack (UUIDv7)     │   │
│  test=fake)   │          └──────▲───────┘          │ [16]byte          │   │
└───────────────┘                 │ Update(remote)  └─────────┬─────────┘   │
                                  │ (l_msg, c_msg)            │ (l,c,NodeID)
                                  │                          ▼
                            ┌─────┴────────┐   Set/SetIfAbsent/Delete
                            │ unpack from  │   (fresh local version)
                            │ RecordVersion│──────────────┐
                            └──────────────┘              │
                                                          ▼
   Merge(key, rec) ──── per key: greater full version wins ──► ┌─────────────────────┐
   (remote records keep                                        │ Store               │
    THEIR version — no re-stamping)                            │ map[key]Record      │
        │                                                      │ + RWMutex           │
        │ losers (displaced versions) — D-01 return value      │ Record: Version,    │
        ▼                                                      │ Data, Deleted,      │
   Conflict{key, loser, winner} ──► Phase 4/6: journal         │ Checksum            │
   drift signal (|l-pt| > drift_warn) ──► Phase 4/6: log       └─────────────────────┘
```

A local write flows: `now()` → `Tick()` → `(l, c)` → pack with own NodeID → record stored. A remote write flows: `Merge(key, rec)` → compare full 16-byte versions → winner kept, loser returned → `Update(rec.Version)` advances the clock. Get/Set/Delete never touch I/O; every side effect (journal, logging) is a return value.

### Recommended Project Structure
```
internal/app/
├── doc.go            # rewritten: real package doc — contracts, concurrency, semantics
├── clock.go          # HLC: state (l, c), constructor (now func, driftWarn, initial l), Tick, Update
├── version.go        # RecordVersion [16]byte: pack(l, c, nodeID, filler), unpack, Compare, NodeID()
├── store.go          # Store: Config, Get/Set/SetIfAbsent/Delete, Merge; Record struct
├── errors.go         # sentinel errors: ErrNotFound, ErrAlreadyExists, ErrValueTooLarge
├── app_test.go       # existing walking-skeleton test REMOVED (superseded)
├── version_test.go   # unit: bit layout vs RFC 9562, round-trip, ordering equivalence
├── clock_test.go     # unit: jump/overflow/restart/drift-signal cases
├── store_test.go     # unit: CORE-01..05 table tests
├── merge_test.go     # property: TEST-01 merge laws (clock-free — records carry versions)
└── hlc_prop_test.go  # property: TEST-02 monotonicity/post-receive/uniqueness/overflow/restart
```

One package, no subpackages (discretion area): a single `app` package keeps the cross-talk between clock, version, and store internal (unexported), which a subpackage split would force into exported API — go-ultimate "start simple; grow deliberately".

### Pattern 1: HLC with injected clock and signal returns (D-01/D-02)
**What:** Clock holds `(l int64 ms, c uint16)`; constructor takes `now func() time.Time`, `driftWarn time.Duration`, and an initial `l` (the VERS-04 seam — Phase 3 passes the dump's max `l`, zero/absent means start from wall clock).
**When to use:** always in this phase; the single `now` seam is where every adversarial test plugs in.
**Example:** see Code Examples — the Kulkarni rules verbatim.

### Pattern 2: Merge applies externally-versioned records
**What:** Local writes (Set/SetIfAbsent/Delete) call `Tick()` and stamp a fresh version; `Merge` accepts a record that already carries its version and joins by comparing full 16-byte versions.
**When to use:** always — remote records must keep their origin version to preserve the CRDT laws; as a bonus the merge-law property tests are clock-free and fully deterministic (they construct versions directly).
**Example:**
```go
// Store merges a remote record; D-01: loser returned by value.
// only a strictly greater version displaces — equal version is a no-op (idempotence)
```

### Pattern 3: Seeded generator with printed seed and replay hook (D-04)
**What:** every property test creates `rng := rand.New(rand.NewPCG(seedHi, seedLo))` from a seed derived from `time.Now().UnixNano()` unless overridden; the seed is printed with `t.Logf` on failure (or unconditionally under `-v`); re-running with the same seed replays exactly.
**When to use:** every property test in this phase and the Phase 4 simulator later.
**Example:** see Code Examples.

### Pattern 4: Sentinel errors (discretion area — recommended resolution)
**What:** `var ErrNotFound, ErrAlreadyExists, ErrValueTooLarge = errors.New(...)`; callers branch with `errors.Is`; Phase 5 maps them 1:1 to NotFound/AlreadyExists/InvalidArgument RPC codes. **Empirically verified this session**: a probe package with `var ErrNotFound = errors.New("not found")` produced **no** `gochecknoglobals` finding under the vendored strict config (gochecknoglobals has a built-in `errors.New` exception) — the pattern is lint-clean with no `//nolint` needed. [VERIFIED: local golangci-lint 2.15.0 probe this session]

### Anti-Patterns to Avoid
- **Re-stamping remote records on Merge** — destroys the CRDT laws (a record's version is its identity across nodes).
- **Comparing only `(l, c)` in merge** — two nodes can produce the same `(l, c)`; the full 16-byte compare (NodeID tie-break) is what makes the winner deterministic and identical on every replica (VERS-01).
- **Dropping tombstoned keys from the map** — a tombstone must persist forever (out-of-scope GC); dropping it lets an older Set resurrect the key.
- **Logging inside `internal/app`** — D-01 forbids hidden side effects; drift returns as a value. `log/slog` imports are technically legal under depguard but contradict the locked decision.
- **Global `rand.Uint64()`-style calls in generators** — package-level `math/rand/v2` functions are auto-seeded; only a local `rand.New(rand.NewPCG(...))` instance replays. [CITED: go.dev/blog/randv2]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Deterministic PRNG | Custom LCG/xorshift | `math/rand/v2` `rand.NewPCG` | Statistical quality, replay guarantee, stdlib; depguard *denies* hand-picking v1 `math/rand` anyway |
| Byte comparison of `[16]byte` | Field-by-field comparators in production code | `bytes.Compare(a[:], b[:])` (or `slices.Compare`) | Correct big-endian lexicographic order; field compare belongs only in the property test as the reference model |
| Error taxonomy | Typed error structs / string matching | Sentinel errors + `errors.Is` | Few and stable (go-ultimate); direct 1:1 Phase 5 RPC mapping; `err113` in the strict config discourages dynamic errors |
| Concurrency safety | Lock-free map, sharding | `sync.RWMutex` + plain map | Design says exactly this; envelope is tens of thousands of keys; measure before optimizing (never, at this scale) |
| Monotonic version scheme | Sequence numbers, vclocks | HLC (Kulkarni) packed into UUIDv7 | Locked design; causality + sortability + uniqueness in one 16-byte value |

**Key insight:** the algorithms themselves (HLC, UUIDv7 packing, LWW join) ARE the in-repo deliverable per the product principle — "don't hand-roll" applies to everything *around* them (PRNG, comparison, errors, locking), not to the three core algorithms, which are each tens of lines against a quoted spec.

## Common Pitfalls

### Pitfall 1: Two visibility rules over tombstones
**What goes wrong:** Get must treat a tombstoned key as NotFound (CORE-01) while create-if-absent must treat the *same* record as existing → AlreadyExists (CORE-03: "an existing key returns AlreadyExists", including a tombstoned one).
**Why it happens:** implementers unify on a single "exists" predicate.
**How to avoid:** explicit semantics table — `Get`: `!ok || rec.Deleted → ErrNotFound`; `SetIfAbsent`: `ok → ErrAlreadyExists` regardless of `Deleted`; `Set`: unconditional replace; `Delete`: fresh-version tombstone.
**Warning signs:** a `Delete`-then-`SetIfAbsent` test succeeding when it should return AlreadyExists.

### Pitfall 2: Counter overflow breaks uniqueness (12-bit counter)
**What goes wrong:** `c` lives in `rand_a` — 12 bits = values 0..4095. More than 4096 writes inside one wall-clock millisecond (stuck clock; sub-ms burst compression) overflows. Wrapping `c` back to 0 emits `(l, 0)` after `(l, 4095)` — a *smaller* byte-wise version: monotonicity and uniqueness both break silently, and the merge laws survive only because duplicates no-op — the real damage is two different writes sharing a version.
**Why it happens:** the HLC paper treats overflow as practically impossible (16-bit counter, c resets whenever pt advances) [CITED: notes.andymatuschak.org/zNyYBEFKUqVLXYnH9i75cM1] — but a 12-bit counter at ~1000 writes/s bursts is 4× closer to the edge.
**How to avoid:** on `c == 4095` at a Tick/Update increment, set `l = l + 1` (1 ms ahead of physical time) and `c = 0` — exactly IDEA.md's "counter overflow bumps `l` by 1 ms" [VERIFIED: docs/IDEA.md:66]. Property test: stuck fake clock + 10,000 Ticks → all versions unique and strictly increasing.
**Warning signs:** versions with equal `l` repeating `c` values; `drift` signal firing during sustained overflow (l runs ahead of pt — that is correct behavior, not a bug).

### Pitfall 3: Merge records a "conflict" on duplicate delivery
**What goes wrong:** retransmitted datagram → equal version → if Merge reports the stored record as a loser, the conflict journal fills with noise and idempotence visibly fails.
**Why it happens:** treating `>=` as displacement.
**How to avoid:** only a **strictly greater** incoming version displaces; equal → no-op, no loser; lesser → incoming record is the loser (CORE-07: "the merge exposes the loser" — including the rejected older record, which IS a displaced version).
**Warning signs:** journal entries with loser.version == winner.version.

### Pitfall 4: []byte aliasing between store and caller
**What goes wrong:** Set stores the caller's slice; caller mutates it afterwards → store state corrupted mid-flight; -race sees nothing (same goroutine can corrupt; dump serializes mutated data).
**Why it happens:** avoiding a copy "for performance".
**How to avoid:** defensive copy of `Data` on the way in (Set/Merge) and on the way out (Get returns a copy) — ≤60 KiB values at thousands of keys; document ownership in doc comments (go-ultimate "ownership and cleanup" rule).
**Warning signs:** property test mutating the input buffer after Set and observing store contents change.

### Pitfall 5: Unseeded or unprinted generator randomness
**What goes wrong:** flaky CI property test nobody can reproduce — the opposite of D-04's contract ("a failing run is replayed exactly").
**Why it happens:** using package-level `rand` functions (auto-seeded), or seeding but not printing the seed on failure.
**How to avoid:** every generator takes a local `*rand.Rand`; seed = `time.Now().UnixNano()` split into two uint64s; print both halves in the failure message (`t.Fatalf`/`t.Errorf`); allow a fixed seed override so `go test -run X -seed ...` replays. [CITED: bitfieldconsulting.com/posts/random-testing]
**Warning signs:** "it passed on re-run" reports.

### Pitfall 6: Restart from wall clock instead of dump max (VERS-04)
**What goes wrong:** node restarts while its clock has jumped backward (or a far-ahead peer had bumped its l before the dump) — initializing the HLC from `time.Now()` lets the node emit versions *older* than the ones in its own dump; its own persisted writes then beat its new ones.
**Why it happens:** `l = now()` in the constructor is the intuitive default.
**How to avoid:** constructor takes an explicit initial `l`; Phase 3 passes the dump's max; the first Tick's `max(l, pt)` reconciles forward jumps naturally. Property test: init with `l` far ahead of a backward-jumped fake clock → all new versions still greater than `l`.
**Warning signs:** clock constructor reading `now()` during initialization beyond taking the max with the provided initial `l`.

### Pitfall 7: Concurrency seam between Clock and Store
**What goes wrong:** Tick (inside Set) and Update (inside Merge) both mutate `(l, c)`; if the clock is separately callable while the store holds only the map lock, races appear only under concurrent test load.
**Why it happens:** splitting the lock domains of two co-stateful components.
**How to avoid:** one decision: either the Store owns the Clock and is the only caller (Clock methods then need no own mutex but keep one anyway if the Clock type is exported), or the Clock carries its own `sync.Mutex` and both remain independently safe. Document concurrency safety per go-ultimate. Property/unit tests must include at least one `-race`-parallel hammer (paralleltest linter in the strict config enforces `t.Parallel()` discipline anyway).
**Warning signs:** `go test -race` failures appearing only after Phase 4 wires the real network.

### Pitfall 8: Adversarial fake clock with monotonic readings
**What goes wrong:** `time.Now()`-derived values carry monotonic clock readings; hand-built fake times (`time.Unix(ms, 0)`) don't. Comparisons via `.After/.Before` behave subtly differently across the two kinds; `==` on times never matches expectations.
**Why it happens:** Go's dual clock readings.
**How to avoid:** store `l` as `int64` milliseconds (via `t.UnixMilli()`) inside the HLC — all comparisons are plain integer arithmetic on `l`; fake clocks build values with `time.Unix(ms, 0)`. The design already speaks of `l` in ms (UUIDv7 `unix_ts_ms`). [VERIFIED: docs/IDEA.md:66-67 "48-bit `unix_ts_ms` = `l`"]
**Warning signs:** any `time.Time` stored as struct state in the clock.

### Pitfall 9: Success-criterion "logged as a warning" (VERS-05) misread as a core obligation
**What goes wrong:** adding `log/slog` to `internal/app` to satisfy the criterion literally — contradicting D-01 ("no hidden side effects").
**Why it happens:** ROADMAP criterion 4 says "drift beyond `drift_warn` is logged as a warning".
**How to avoid:** Phase 2 delivers the *signal* (returned boolean/struct from Tick/Update when `|l − pt| > drift_warn`) and tests assert it fires; the log emission is Phases 4/6 wiring (per D-01: "callers decide where... drift warnings are recorded"). The planner should word the task as "drift signal returned and tested", not "drift logged".
**Warning signs:** any logging import in the pure core.

## Code Examples

### HLC Tick and Update — the algorithm this phase implements
```
Source: singhajit.com/distributed-systems/hybrid-clock (summary of Kulkarni et al. 2014,
cse.buffalo.edu/tech-reports/2014-04.pdf — primary paper; PDF fetch unsupported, rules
cross-checked against the summary and the paper's abstract)

Local / Send Event (Tick):                    Receive Event (Update), message (l.m, c.m):

    old_l = l.j                                   old_l = l.j
    l.j = max(old_l, pt.j)                        l.j = max(old_l, l.m, pt.j)
    if l.j == old_l:
        c.j = c.j + 1                             if l.j == old_l == l.m:
    else:                                             c.j = max(c.j, c.m) + 1
        c.j = 0                                   elif l.j == old_l:
                                                     c.j = c.j + 1
                                                 elif l.j == l.m:
                                                     c.j = c.m + 1
                                                 else:
                                                     c.j = 0
```
sup-kv additions per locked design: `pt` comes from the injected `now func() time.Time` (D-02); overflow of the 12-bit `c` bumps `l` by 1 and sets `c = 0` [VERIFIED: docs/IDEA.md:66 "counter overflow bumps `l` by 1 ms"]; the constructor accepts an initial `l` from the dump [VERIFIED: docs/IDEA.md:71-72 "initialized at startup from the dump's max `l`"]; every Tick/Update also evaluates the drift signal against `drift_warn` (default `60s` [VERIFIED: docs/IDEA.md:148]).

### UUIDv7 layout this phase packs (RFC 9562, verified against datatracker.ietf.org)
```
// Source: RFC 9562 §5.7 (Figure 11), §6.2, §6.11 — fetched this session.
// Field quotes verbatim:
//   unix_ts_ms: "Occupies bits 0 through 47 (octets 0-5)."  48-bit big-endian ms = l
//   ver:        "Occupies bits 48 through 51 of octet 6."   0b0111
//   rand_a:     "Occupies bits 52 through 63 (octets 6-7)." 12-bit HLC counter c
//   var:        "Occupies bits 64 and 65 of octet 8."       0b10
//   rand_b:     "Occupies bits 66 through 127 (octets 8-15)." 62 bits:
//               16-bit NodeID in the high bits + 46 random filler bits
//               [VERIFIED: docs/IDEA.md:67-69 "62-bit rand_b = 16-bit NodeID in the
//                high bits + random filler"]
// §6.2 Method 1: "the rand_a section (or a subset of its leftmost bits) of UUIDv7 is
//   used as a fixed bit-length dedicated counter that is incremented for every UUID
//   generation"; counter "SHOULD be at least 12 bits but no longer than 42 bits".
// §6.11: UUIDv7 "sort as opaque raw bytes without the need for parsing".
//
// Byte-wise packing (derived arithmetic; ver/var are constants so byte order == (l, c, NodeID, filler) order):
//   octets 0-5: l, big-endian
//   octet 6:    0x70 | byte(c>>8)
//   octet 7:    byte(c)
//   octet 8:    0x80 | byte(nodeID>>10)
//   octet 9:    byte(nodeID >> 2)
//   octet 10:   byte(nodeID&0x3)<<6 | top 6 filler bits
//   octets 11-15: remaining 40 filler bits
```

### Seeded property-test generator (D-04 pattern, math/rand/v2)
```go
// Source: go.dev/blog/randv2 (rand.NewPCG) + bitfieldconsulting.com/posts/random-testing
// (seed printing/replay pattern). Depguard denies math/rand v1 [VERIFIED: .golangci.yml:34-35].
package app_test

import (
    "fmt"
    "math/rand/v2"
    "testing"
    "time"
)

func newRNG(t *testing.T) *rand.Rand {
    t.Helper()
    // fixed seed when set via the replay hook; time-derived otherwise
    seed := time.Now().UnixNano()
    hi, lo := uint64(seed>>32), uint64(seed&0xFFFFFFFF)
    t.Logf("property seed: %d,%d (replay: SEED=%d go test -run %s)", hi, lo, seed, t.Name())
    return rand.New(rand.NewPCG(hi, lo))
}

// adversarial clock: random walk with jumps, sticks, and bursts below 1 ms
func nextWallTime(rng *rand.Rand, prev int64) int64 {
    switch rng.IntN(10) {
    case 0: // big backward jump (hours)
        return prev - rng.Int64N(4*60*60*1000)
    case 1: // big forward jump
        return prev + rng.Int64N(4*60*60*1000)
    case 2, 3: // stuck clock — same millisecond
        return prev
    default: // small step, sometimes within the same ms
        return prev + rng.Int64N(3)
    }
}
```

### Merge-law property skeleton (TEST-01, clock-free)
```go
// Records carry their versions: the laws are tested without any clock,
// which keeps them exactly deterministic. State-based CvRDT join-semilattice:
// merge = per-key max version. [CITED: Shapiro et al. 2011, inria.hal.science/inria-00555588]
func TestMerge_laws(t *testing.T) {
    t.Parallel()
    rng := newRNG(t)
    for range 500 {
        a, b, c := randStore(rng), randStore(rng), randStore(rng) // incl. tombstones
        // commutativity: merge(a,b) == merge(b,a)
        // associativity: merge(merge(a,b),c) == merge(a,merge(b,c))
        // idempotence:   merge(a,a) == a   (no loser reported on duplicate)
        // losers: every displaced version returned exactly once with correct winner
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| RFC 4122 UUIDv1 (timestamp + MAC) | RFC 9562 UUIDv7 (ms timestamp + counter guidance) | 2024 (Proposed Standard, obsoletes 4122) | Sortable, counter-in-`rand_a` explicitly designed-for — sup-kv's packing is RFC-blessed [VERIFIED: RFC 9562 §6.2] |
| `math/rand` v1 with `rand.Seed` | `math/rand/v2` with `rand.New(rand.NewPCG(a, b))` | Go 1.22, 2024 | Deterministic replay requires a local RNG instance; `Seed` is gone [CITED: go.dev/blog/randv2] |
| `testing/quick` for properties | Frozen (stdlib, no new features); hand-rolled seeded generators or rapid | frozen for years | D-04's hand-rolled approach is the standard no-dep route [CITED: go.dev/src/testing/quick + bitfieldconsulting] |
| Physical timestamps for LWW | HLC (l, c) | Kulkarni et al. 2014 | Correctness independent of clock sync; skew only affects which concurrent write wins — deterministically, identically on all replicas |

**Deprecated/outdated:**
- `math/rand` v1 — denied by this repo's depguard; use `math/rand/v2`.
- UUIDv1/v6 for this design — RFC 9562 §5.7: "implementations SHOULD utilize UUIDv7 instead of UUIDv1 and UUIDv6 if possible".

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `time.Time.UnixMilli()` (Go 1.17+) is the intended ms extraction for `l` | Pitfall 8, code examples | Trivial — any ms extraction works; planner picks |
| A2 | Filler bits (46) may be zero-filled per node instead of random per write — uniqueness rests on (NodeID, l, c), not filler | Architecture | If random-per-write is preferred, only filler generation changes; ordering/uniq unaffected |
| A3 | `Delete` of a missing or already-tombstoned key returns `ErrNotFound` (fresh tombstone only written for a live key) | Open Question 1 | If chosen otherwise, CORE-04 test expectations change; no structural impact |
| A4 | Drift signal fires on `\|l − pt\| > drift_warn` in either direction (not only wall-clock-ahead) | Open Question 2 | Single-direction check would miss the "far-ahead node" TEST-04 scenario wiring |

## Open Questions (RESOLVED)

All four questions carry adopted dispositions as of plan finalization (2026-11-02); one-line status per question below, citing the deciding plan task.

1. **Delete semantics for missing/tombstoned keys** — RESOLVED: plan 02-01 Task 2 adopts `ErrNotFound` for missing and already-tombstoned keys — no version is burned on a no-op (assumption A3).
   - What we know: CORE-04 requires a fresh-version tombstone on Delete; Get defines NotFound for missing/tombstoned; nothing states Delete's behavior on an absent key.
   - What's unclear: error (recommended — `ErrNotFound`, symmetric with Get and maps to the same RPC code) vs. idempotent silent success (always writes a tombstone).
   - Recommendation: `ErrNotFound` for missing and already-tombstoned; keeps versions from being burned on no-ops. Planner decides.
2. **Drift signal direction** — RESOLVED: plan 02-01 Task 1 (clock.go spec) implements the signed both-directions signal of assumption A4 — positive when the HLC runs ahead beyond `drift_warn`, negative on a forward wall-clock jump beyond it; carried into testing as flagged assumption VERS-05 in plan 02-02.
   - What we know: D-02 says "return a drift signal when the wall clock runs ahead beyond `drift_warn`"; ROADMAP criterion 4 says "drift beyond `drift_warn`"; TEST-04 (Phase 6) includes "a node's clock far ahead".
   - What's unclear: whether the signal covers only `pt > l + drift_warn` (forward) or both directions.
   - Recommendation: signal on `|l − pt| > drift_warn` with the sign available — both directions are operationally meaningful (backward jump = timestamps in the future = unfair LWW wins; forward jump = NTP correction), and the cost is one comparison. Planner decides.
3. **Record Checksum helper now or in Phase 3** — RESOLVED: plan 02-01 Task 1 ships the exported `RecordChecksum` (CRC32 over key+version+data+deleted) alongside the Record type now; enforcement stays deferred to Phase 3 (PERS-02).
   - What we know: IDEA.md defines Checksum as a record field ("CRC32 over key+version+data+deleted" [VERIFIED: docs/IDEA.md:54]); Phase 2's boundary text does not list checksum computation; PERS-02 (Phase 3) validates it on dump load.
   - What's unclear: whether Phase 2 ships the ~10-line `hash/crc32` helper alongside the Record type.
   - Recommendation: define the field in Phase 2 (the Record type is Phase 2's deliverable; Phase 3 serializes it) and include the tiny constructor/check function now so records are self-validating from the start — but defer *enforcement* to Phase 3. Planner decides; strict-minimal alternative (field only) is also defensible.
4. **Store constructor shape for the `now` seam** — RESOLVED: plan 02-01 Task 1 — `NewStore(nodeID, cfg, now)` builds its internal clock from `now`, `cfg.InitialL` and `cfg.DriftWarn` (one composition point, one seam; Clock stays an implementation detail, independently exported and testable).
   - What we know: D-02 puts `now func() time.Time` in the *clock's* constructor; the store needs a clock; Phase 3 needs to inject the dump's max `l` through the same construction path.
   - What's unclear: `NewStore(nodeID, cfg, now)` (store builds its clock) vs. exposing the Clock so callers compose it.
   - Recommendation: store accepts `now` (and cfg carries `DriftWarn` plus an optional initial `l`) and builds its internal clock — one composition point, one seam, matches "accept functions, return structs" and keeps Clock an implementation detail while still independently exported/testable. Planner decides.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | everything | ✓ | go1.27.3 linux/amd64 (go.mod: `go 1.27.1`) | — |
| golangci-lint (via `go tool`) | `make lint` | ✓ | 2.15.0 (built with go1.27.3) | — |
| govulncheck (via `go tool`) | CI only (not this phase) | ✓ (pinned v1.9.0 in go.mod tool directives) | v1.9.0 | — |
| Network access for modules | nothing — zero new deps | not required | — | — |

**Missing dependencies with no fallback:** none
**Missing dependencies with fallback:** none — the phase is deliberately dependency-free (D-04).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vanilla `testing` stdlib; external `package app_test`; table-driven units + seeded property tests; no assertion library |
| Config file | none (Go default; `Makefile` targets `lint`/`test`/`build` [VERIFIED: Makefile]) |
| Quick run command | `go test -race ./internal/app/` |
| Full suite command | `make test` (= `go test -race -cover ./...`) plus `make lint` (= `golangci-lint config verify` + `run ./...`) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CORE-01 | Get NotFound (missing + tombstoned) | unit, table | `go test -race ./internal/app/ -run TestStore_Get` | ❌ Wave 0 (`store_test.go`) |
| CORE-02 | Set create/replace under LWW | unit + property | `go test -race ./internal/app/ -run TestStore_Set` | ❌ Wave 0 |
| CORE-03 | SetIfAbsent AlreadyExists incl. tombstone | unit, table | `go test -race ./internal/app/ -run TestStore_SetIfAbsent` | ❌ Wave 0 |
| CORE-04 | Delete fresh-version tombstone | unit | `go test -race ./internal/app/ -run TestStore_Delete` | ❌ Wave 0 |
| CORE-05 | Value > 60 KiB → ErrValueTooLarge | unit, boundary (61440/61441) | `go test -race ./internal/app/ -run TestStore_valueLimit` | ❌ Wave 0 |
| CORE-06 | Merge commutative/associative/idempotent incl. tombstones | property (TEST-01) | `go test -race ./internal/app/ -run TestMerge_laws` | ❌ Wave 0 (`merge_test.go`) |
| CORE-07 | Every displaced version returned as loser | unit + property | `go test -race ./internal/app/ -run TestMerge_losers` | ❌ Wave 0 |
| VERS-01 | bytes.Compare == (l, c, NodeID) order | property | `go test -race ./internal/app/ -run TestVersion_byteOrder` | ❌ Wave 0 (`version_test.go`) |
| VERS-02 | Unique per write, strictly increasing | property | `go test -race ./internal/app/ -run TestVersion_uniquePerWrite` | ❌ Wave 0 |
| VERS-03 | Monotonic under adversarial clock | property (TEST-02) | `go test -race ./internal/app/ -run TestClock_monotonic` | ❌ Wave 0 (`hlc_prop_test.go`) |
| VERS-03 | Counter overflow bumps l (stuck clock, >4096/ms) | property | `go test -race ./internal/app/ -run TestClock_counterOverflow` | ❌ Wave 0 |
| VERS-04 | Restart init from dump max, not wall clock | unit + property | `go test -race ./internal/app/ -run TestClock_restartFromDumpMax` | ❌ Wave 0 |
| VERS-05 | Drift signal beyond drift_warn (both directions recommended) | unit, table | `go test -race ./internal/app/ -run TestClock_driftSignal` | ❌ Wave 0 (`clock_test.go`) |
| TEST-02 | Post-receive ordering (Tick after Update beats remote) | property | `go test -race ./internal/app/ -run TestClock_postReceive` | ❌ Wave 0 |
| QUAL-01 | Lint gate green | gate | `make lint` | ✓ exists |

### Sampling Rate
- **Per task commit:** `go test -race ./internal/app/` + `make lint` (both fast; lint on the touched package via `go tool golangci-lint run ./internal/app/...`)
- **Per wave merge:** `make test` + `make lint` (full repo, same as CI)
- **Phase gate:** full suite + lint green before `/gsd:verify-work`; CI runs the identical targets on the 4-cell matrix

### Wave 0 Gaps
- [ ] `internal/app/version_test.go` — VERS-01/02 unit+property (bit-layout assertions against RFC 9562: ver/var bits, round-trip, ordering equivalence)
- [ ] `internal/app/clock_test.go` — VERS-03/04/05 unit cases (jumps, stuck, overflow boundary at 4095→4096, restart, drift table)
- [ ] `internal/app/store_test.go` — CORE-01..05 table tests (boundary values 61440/61441 bytes; tombstone visibility matrix)
- [ ] `internal/app/merge_test.go` — TEST-01/CORE-06/07 property suite (seeded, incl. tombstones and duplicate delivery)
- [ ] `internal/app/hlc_prop_test.go` — TEST-02 property suite (adversarial clock generator, uniqueness, post-receive, counter-reset)
- [ ] Remove `TestAppSkeleton` from `app_test.go` (superseded — CONTEXT.md: "the walking-skeleton test is superseded by real engine tests")
- [ ] Shared generator helpers (`newRNG`, `randStore`, `nextWallTime`) — either one helpers file per test file or a single `props_test.go`; they live in `_test.go` files only

## Security Domain

ASVS level 1 (`security_enforcement: true`, `security_block_on: high`). This phase is a pure in-memory library with no attack surface beyond its inputs; the replication trust model (no auth/encryption in v1) is a documented accepted limitation [VERIFIED: docs/IDEA.md:150-154 "Trust model (accepted limitation)"].

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | nothing authenticates in this phase; trust model is documented future work (SEC-01, v2) |
| V3 Session Management | no | no sessions |
| V4 Access Control | no | no callers beyond future adapters |
| V5 Input Validation | yes | `value_max_bytes` enforcement at the single entry point (Set/SetIfAbsent) — reject, never truncate; input dimensions (key length, value size) validated before allocation/copy |
| V6 Cryptography | no | CRC32 is an integrity checksum against corruption, **not** a security control — must not be presented as tamper protection (trust model: trusted segment) |
| V12 File Handling | no | no filesystem in this phase (Phase 3) |

### Known Threat Patterns for pure in-memory CRDT store

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Oversized-value DoS (memory exhaustion via huge values) | DoS | `value_max_bytes` (61440) check before any copy or storage (CORE-05) |
| Forged/future timestamps to win all merges | Tampering/Elevation | Out of scope for the core (trusted segment); HLC drift signal (VERS-05) makes the condition observable; HMAC is SEC-01 (v2) |
| Checksum mistaken for integrity protection | Tampering | Document CRC32 as corruption-detection only (dump integrity), never as auth |
| Panic-based DoS via malformed records (Merge input) | DoS | Merge treats input records as data; unpack of malformed versions must not panic (property tests feed random 16-byte versions) |

## Sources

### Primary (HIGH confidence)
- RFC 9562 (datatracker.ietf.org/doc/html/rfc9562) — §5.7 field layout, §6.2 counter Method 1, §6.11 byte-wise sorting; fetched this session, quoted verbatim in Code Examples
- In-repo, read this session with Read tool: `docs/IDEA.md` (HLC/UUIDv7/LWW design, parameters), `.planning/phases/02-engine-hlc-versioning-and-lww-store/02-CONTEXT.md` (D-01..D-04), `.planning/REQUIREMENTS.md` (CORE/VERS/TEST), `.planning/ROADMAP.md` §Phase 2, `.golangci.yml` (depguard rules), `go.mod`, `internal/app/doc.go`, `internal/port/doc.go`, `internal/app/app_test.go`, `Makefile`, `AGENTS.md`, `.claude/CLAUDE.md`
- Local empirical probe: golangci-lint 2.15.0 on a scratch package with a sentinel error global — clean (gochecknoglobals does not flag `errors.New` vars)
- Go 1.27.3 toolchain + `go tool golangci-lint version` verified on this machine

### Secondary (MEDIUM confidence)
- singhajit.com/distributed-systems/hybrid-clock — Tick/Update pseudocode quoted verbatim (cross-checked against Kulkarni et al. 2014 abstract; the paper PDF itself at cse.buffalo.edu/tech-reports/2014-04.pdf rejects fetch due to content type)
- Shapiro et al., *A Comprehensive Study of Convergent and Commutative Replicated Data Types* (2011, inria.hal.science/inria-00555588 / dsf.berkeley.edu/cs286/papers/crdt-tr2011.pdf) — state-based CvRDT join-semilattice
- go.dev/blog/randv2 — `rand.NewPCG`, auto-seeded global functions, removal of `Seed`
- bitfieldconsulting.com/posts/random-testing — seed-print/replay property-testing pattern in Go
- gobyexample.com/random-numbers, gosamples.dev — `rand.New(rand.NewPCG(...))` usage
- lemire.me/blog (Oct 2025) — PCG fastest Go generator benchmark

### Tertiary (LOW confidence)
- notes.andymatuschak.org/zNyYBEFKUqVLXYnH9i75cM1 — HLC counter overflow "practically impossible" reasoning (directionally sound; superseded here by the 12-bit-specific arithmetic anyway)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero dependencies, all stdlib components verified present; lint behavior empirically probed
- Architecture: HIGH — package shape follows locked decisions D-01..D-04 and existing Phase 1 skeleton; merge-keeps-remote-version is the one recommended structural choice (well-precedented CvRDT practice)
- Pitfalls: HIGH — counter-overflow arithmetic derived from verified 12-bit `rand_a` + design's overflow rule; aliasing/visibility pitfalls are standard Go/CRDT knowledge reinforced by in-repo design constraints
- HLC algorithm details: MEDIUM — quoted from a secondary summary of the Kulkarni paper (PDF not directly fetchable); rules match the paper's published structure and the design doc's own description; the property tests themselves verify the implementation independently

**Research date:** 2026-11-02
**Valid until:** 2027-11-02 (stable domain — published algorithms and frozen stdlib; no fast-moving dependencies)
