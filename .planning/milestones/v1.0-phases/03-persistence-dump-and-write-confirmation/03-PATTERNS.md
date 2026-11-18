# Phase 3: Persistence — Dump and Write Confirmation - Pattern Map

**Mapped:** 2026-11-03
**Files analyzed:** 12 (10 new, 2 modified)
**Analogs found:** 12 / 12 (5 exact, 3 role-match, 2 conventions-only, 2 no role analog — patterns supplied by RESEARCH.md)

All analog paths below are git-TRACKED source (verified via `git ls-files`). The repo has no gitignored mirror paths in play for these files.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/app/persist/doc.go` (name: planner's discretion) | package doc | n/a | `internal/app/storage/doc.go` | exact |
| `internal/app/persist/codec.go` | service (pure codec) | transform | `internal/app/version/version.go` + `internal/app/storage/store.go` (checksum domain) | role-match |
| `internal/app/persist/engine.go` | service (concurrent core) | event-driven | `internal/app/clock/clock.go` (time+mutex state) + `internal/app/storage/store.go` (consuming core) | role-match |
| `internal/app/persist/errors.go` | model (sentinel errors) | n/a | `internal/app/storage/errors.go` | exact |
| `internal/app/persist/codec_test.go` | test (property) | transform | `internal/app/storage/merge_test.go`, `internal/app/clock/hlc_prop_test.go` | exact |
| `internal/app/persist/engine_test.go` | test (synctest + fake port) | event-driven | `internal/app/clock/clock_test.go` + `internal/app/storage/store_test.go:735` (concurrent hammer) | conventions-only (synctest not yet exercised in repo) |
| `internal/port/dump.go` | provider (port interface) | request-response (bytes in/out) | `internal/app/storage/store.go:51-54` (consumer-side `Clock` interface) | role-match (first real port file; `internal/port/doc.go` is a placeholder) |
| `internal/persistfile/dumpfile.go` (adapter; name: planner's discretion) | adapter (file I/O) | file-I/O | none — first `os`-importing package in the repo | none |
| `internal/persistfile/snapshot.go` | adapter (file I/O) | file-I/O | none | none |
| `internal/persistfile/dumpfile_test.go` | test (real fs) | file-I/O | `internal/app/clock/clock_test.go:87-160` (table-test shape) | conventions-only (no fs tests exist yet) |
| `internal/app/internal/testutil/gen.go` (modify — add map/records generators if needed) | test utility | n/a | itself (`RandStore`/`RandRecord`) | exact |
| `docs/IDEA.md` (modify — append the D-04 max-wait parameter) | config/doc | n/a | itself, §Parameters lines 142-148 | exact |

Package names `persist` / `persistfile` are RESEARCH.md suggestions A7 — CONTEXT.md leaves them to the planner.

## Pattern Assignments

### `internal/app/persist/doc.go` (package doc)

**Analog:** `internal/app/storage/doc.go` (37 lines — copy the whole shape)

Package doc contract to replicate: name the package's CRDT/pure role, name the seam it consumes, cite the depguard rule by name, and document side-effect policy. From `internal/app/storage/doc.go:1-8`:

```go
// Package storage is the sup-kv in-memory LWW-Element-Map store: a pure,
// I/O-free library over records versioned by the sibling version package
// and stamped by a clock supplied by the caller through the consumer-side
// Clock interface declared here — ... per D-03, internal/port
// holds no interfaces. The lint gate forbids networking, filesystem,
// syscall and gRPC/protobuf imports here — see the sup-kv-app-no-io
// depguard rule in .golangci.yml.
```

For `persist`: document the dump codec, the write-confirmation watermark/waiter contract, and — go-ultimate requirement flagged by RESEARCH (Pitfall 4) — the dumper goroutine's documented exit condition in the API contract.

---

### `internal/app/persist/codec.go` (service, transform)

**Analogs:** `internal/app/version/version.go` (pure-function codec shape) and `internal/app/storage/store.go` (the record/checksum domain it must NOT re-implement)

**Domain constants + doc style** — copy `internal/app/version/version.go:23-32` (named constants block with per-constant comment) and `store.go:14-18`:

```go
// Design default NewStore substitutes for a zero Config field, and the
// checksum flag bytes of the RecordChecksum domain, from docs/IDEA.md.
const (
	defaultValueMaxBytes = 61440
	notDeletedFlag       = 0
	deletedFlag          = 1
)
```

**Checksum reuse (mandatory — D-07):** the codec must verify with `storage.RecordChecksum`, never a second CRC32. From `internal/app/storage/store.go:104-110`:

```go
// RecordChecksum returns the CRC32 over the key bytes, the 16 version bytes,
// the data bytes and one flag byte for deleted. It is corruption detection
// for the Phase 3 dump, never tamper protection.
func RecordChecksum(key string, ver version.RecordVersion, data []byte, deleted bool) uint32 {
	flag := tombstoneFlag(deleted).checksumByte()
	sum := crc32.Update(0, crc32.IEEETable, []byte(key))
	sum = crc32.Update(sum, crc32.IEEETable, ver[:])
	sum = crc32.Update(sum, crc32.IEEETable, data)
	return crc32.Update(sum, crc32.IEEETable, []byte{flag})
}
```

**Totality discipline for decode** — mirror `version.go:49-58` ("It is total: pure masking arithmetic over every possible bit pattern, it never panics") plus `RecordVersion.Unpack()` used for the watermark:

```go
// Unpack returns the hybrid logical clock reading and node ID packed into v.
// It is total: pure masking arithmetic over every possible bit pattern, it
// never panics.
func (v RecordVersion) Unpack() (l int64, counter uint16, nodeID uint16) {
```

**Defensive-copy contract** — mirror `store.go:240-256` (`Records()` is literally the dump source, tombstones included):

```go
// Records returns a full snapshot of the map, tombstones included, with Data
// defensively copied. This is the Phase 3 dump source; Get cannot see
// tombstones by design.
func (s *Store) Records() map[string]storage.Record {
```

Codec encode skeleton (sorted keys, `json.Encoder` newline-per-value, hex version — the `[16]byte` pitfall) is fully specified in RESEARCH.md Pattern 2; the codebase contributes the record types (`storage.Record`, `store.go:31-36`) and the checksum above.

---

### `internal/app/persist/engine.go` (service, event-driven)

**Analogs:** `internal/app/clock/clock.go` (mutex-guarded time-state machine) + `internal/app/storage/store.go` (concurrent core consuming an injected dependency)

**Constructor-with-seam pattern** — from `internal/app/clock/clock.go:63-65` (the engine likewise takes its dependencies by parameter; `time` is legal inside the app boundary):

```go
func NewClock(now func() time.Time, initialL int64, initialC uint16, driftWarn time.Duration) *Clock {
	return &Clock{l: initialL, c: initialC & counterMask, now: now, driftWarn: driftWarn}
}
```

**Stateful core + concurrency-safety doc** — from `internal/app/clock/clock.go:30-40`:

```go
// Clock is a Hybrid Logical Clock ... Clock is safe for concurrent use by
// multiple goroutines: every method takes the clock's own mutex, independent
// of any Store lock.
type Clock struct {
	mu        sync.Mutex
	l         int64
	c         uint16
	now       func() time.Time
	driftWarn time.Duration
}
```

**Consuming-core + Config normalization** — from `internal/app/storage/store.go:56-87` (the engine wraps a `*storage.Store` the same way the store wraps a `Clock`; `Records()` at line 243 is the dump input):

```go
// Store is the in-memory LWW-Element-Map: ... Store is safe for concurrent use
// by multiple goroutines.
type Store struct {
	mu      sync.RWMutex
	nodeID  uint16
	cfg     Config
	records map[string]Record
	clock   Clock
}

func NewStore(nodeID uint16, clk Clock, cfg Config) *Store {
	return &Store{nodeID: nodeID, cfg: cfg.normalized(), records: make(map[string]Record), clock: clk}
}
```

**Results-not-callbacks (Phase 2 D-01, binding):** every store operation returns values (`Set` returns `(version, drift, error)`, `Merge` returns conflicts) — the engine's wait primitive must resolve by return value / channel close, never callbacks. See `store.go:132-140`:

```go
func (s *Store) Set(key string, data []byte) (version.RecordVersion, time.Duration, error) {
	if len(data) > s.cfg.ValueMaxBytes {
		return version.RecordVersion{}, 0, ErrValueTooLarge
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ver, drift := s.write(key, data, false)
	return ver, drift, nil
}
```

Watermark/waiter/debounce mechanics (channel-close broadcast, single dumper goroutine, bounded max wait) have no codebase analog — use RESEARCH.md Patterns 4 and 5 verbatim.

---

### `internal/app/persist/errors.go` (model, sentinel errors)

**Analog:** `internal/app/storage/errors.go` (19 lines — copy the whole file's shape)

```go
// Sentinel errors returned by the store operations. Callers match them with
// [errors.Is], never identity comparison; Phase 5 maps them 1:1 to RPC codes.
var (
	// ErrNotFound is returned by Get and Delete when the key is absent or
	// tombstoned. Phase 5 maps it to the NotFound RPC code.
	ErrNotFound = errors.New("key not found")
	...
)
```

New sentinel (e.g. `ErrDumpFailed`) follows exactly: `errors.New(...)`, `[errors.Is]` matching, doc comment naming the Phase 5 RPC mapping (`Unavailable`, per D-05).

---

### `internal/port/dump.go` (provider, request-response)

**Analog:** the consumer-side interface declaration at `internal/app/storage/store.go:47-54`:

```go
// Clock is the version source the store consumes: Tick stamps local writes,
// Update absorbs received readings. The concrete clock (internal/app/clock)
// satisfies it; tests may substitute fakes. Declared consumer-side here —
// per D-03, internal/port holds no interfaces.
type Clock interface {
	Tick() (l int64, counter uint16, drift time.Duration)
	Update(remoteL int64, remoteC uint16) (l int64, counter uint16, drift time.Duration)
}
```

Notes for the planner:
- Phase 2's D-03 reading kept `storage.Clock` consumer-side; Phase 3's research assigns the dump seam to `internal/port` (RESEARCH §Architectural Responsibility Map). Either placement has in-repo precedent — the consumer-side pattern above if the engine declares its own, or a new `internal/port/dump.go` replacing the placeholder comment in `internal/port/doc.go:1-9` ("interfaces appear with the phases that need them").
- Hard constraint: `internal/port` is depguard `sup-kv-port-stdlib-only` ($gostd + own module) — the interface sees only `[]byte`, `string`, `func`, `error` (D-02: whole bytes, never records).
- Doc-comment convention: name the implementer, name the fake-substitution test strategy, cite the decision IDs.

---

### `internal/app/persist/codec_test.go` (test, property)

**Analog:** `internal/app/storage/merge_test.go` (exact conventions) and `internal/app/clock/hlc_prop_test.go`

**The seeded property-test loop** — from `internal/app/storage/merge_test.go:126-136`:

```go
// TestMerge_laws proves the CRDT join laws clock-free over 500 seeded random
// store triples with live and tombstoned records mixed: ... (TEST-01, CORE-06).
func TestMerge_laws(t *testing.T) {
	t.Parallel()
	rng := testutil.NewRNG(t)
	for step := range 500 {
		assertMergeLaws(t, step, testutil.RandStore(rng), testutil.RandStore(rng), testutil.RandStore(rng))
	}
}
```

Conventions to copy verbatim: `t.Parallel()` first line; `rng := testutil.NewRNG(t)`; seeded step loop; failure messages end with `(replay with the property seed above)` (see `merge_test.go:86,98,106`); helpers are doc-commented `t.Helper()` functions (`merge_test.go:19-45`).

**Generator reuse** — `internal/app/internal/testutil/gen.go:53-85` already provides `RandRecord`, `RandKey`, `RandStore` (Merge-built stores with arbitrary versions). The codec's round-trip input is `testutil.RandStore(rng).Records()`; the watermark property needs the dump-model construction style of `hlc_prop_test.go:238-251` (watermark record at `(dumpMax, watermarkC)` plus older-l records — the WR-01 hazard shape).

**Round-trip property precedent** — `internal/app/version/version_test.go:81` (`TestVersion_roundTrip`) and the map-comparison helper `merge_test.go:49-61` (`recordsEqual`, deep-comparing version/deleted/checksum/data).

---

### `internal/app/persist/engine_test.go` (test, event-driven, synctest)

**Analog (conventions only):** `internal/app/clock/clock_test.go` for fake-seam helpers; `internal/app/storage/store_test.go:717-734` for concurrency-harness shape. **No synctest usage exists in the repo yet** (RESEARCH §Wave 0 Gaps) — follow RESEARCH's synctest examples and go-ultimate rules: no `t.Parallel`/`t.Run` on the inner bubble `*testing.T`; fake in-memory dump port inside the bubble, never real I/O.

**Fake-seam helper style** — from `internal/app/clock/clock_test.go:36-53`:

```go
// fixedNow returns a now func parked at the given Unix millisecond, built
// with time.UnixMilli so no monotonic reading is involved (D-02).
func fixedNow(ms int64) func() time.Time {
	return func() time.Time { return time.UnixMilli(ms) }
}
```

**Goroutine-safe fake under concurrency** — from `internal/app/storage/store_test.go:717-726`:

```go
// atomicNow returns a goroutine-safe fake now over an atomic millisecond
// counter: ...
func atomicNow(startMs int64) func() time.Time {
	var ms atomic.Int64
	ms.Store(startMs)
	return func() time.Time { return time.UnixMilli(ms.Add(1)) }
}
```

The fake dump port should follow this doc-commented-helper style (record save times/errors in memory; implement the same interface the file adapter implements). `sync.WaitGroup.Go` worker pattern from `clock_test.go:76-103` is the fan-out precedent.

---

### `internal/persistfile/dumpfile.go` (adapter, file-I/O) — no codebase analog

No `os`-importing package exists yet (verified: only `internal/app/*` + `cmd` + `internal/port` packages). First adapter in the repo — AGENTS.md Layout: "Adapter packages under `internal/` appear with the phases that need them; do not create empty adapter directories."

Pattern source: RESEARCH.md Pattern 3 (canonical atomic-write sequence — temp in target dir → write → fsync → rotate `main`→`.prev` → rename → dir fsync) and the anti-patterns list (no `bufio.Scanner`, no system-temp `CreateTemp`, one dump at a time). Structural conventions to still copy from the codebase: package doc comment (`doc.go` analog above), decision-citing doc comments on every exported symbol, sentinel errors via `errors.New` if the adapter adds its own.

### `internal/persistfile/snapshot.go` (adapter, file-I/O) — no codebase analog

Same as `dumpfile.go`; RESEARCH Pattern 6 covers create (force dump, byte-copy) and name validation (allow-list before any `filepath.Join`, Security §V5).

### `internal/persistfile/dumpfile_test.go` (test, file-I/O)

**Analog (table shape only):** `internal/app/clock/clock_test.go:87-130`:

```go
func TestClock_tick(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		initialL int64
		nows     []int64
		want     []reading
	}{
		{
			name:     "wall ahead of l adopts the wall and resets the counter",
			...
```

Real-fs tests are new to the repo: `t.TempDir()` per go-ultimate (RESEARCH §Project Constraints); scenario table maps to RESEARCH's kill-9 residue cases (truncated main / garbage main / main missing / both missing / one torn record). Runs outside synctest bubbles with real time.

### `internal/app/internal/testutil/gen.go` (modify — optional generators)

**Analog:** itself. Existing surface (`rng.go:41-51` `NewRNG` printed-seed contract; `gen.go:33-85` `RandVersion`/`RandRawVersion`/`RandRecord`/`RandKey`/`RandStore`) likely suffices — the codec consumes `map[string]storage.Record` from `RandStore(...).Records()`. If a plain map generator is added, copy the constant-block + doc-comment style of `gen.go:15-22`. Any `//nolint` must name its linter and carry a reason (precedent: `rng.go:35,50`, `gen.go:36-37`).

### `docs/IDEA.md` (modify — D-04 parameter entry)

**Analog:** itself, §Parameters at lines 142-148:

```markdown
`node_id`, `bind_addr`, `broadcast_addr`, `dump_path` (required);
`value_max_bytes` 61440; `sync_chunk_bytes` 1400; `hello_interval` 1s;
...
`startup_gate` 1s; `dump_debounce` 50ms; `drift_warn` 60s.
```

Append the bounded max wait (default 1 s, D-04) to this exact list, keeping the `name value unit` micro-format. Exact parameter name is planner's discretion.

## Shared Patterns

### Sentinel errors + `[errors.Is]` matching
**Source:** `internal/app/storage/errors.go:5-19`
**Apply to:** `internal/app/persist/errors.go` (ErrDumpFailed-style), any adapter errors. Doc comment states the Phase 5 RPC mapping.

### Constructor injecting the seam (no globals)
**Source:** `internal/app/clock/clock.go:63-65` (`now func() time.Time`), `internal/app/storage/store.go:71-78` (`clk Clock`)
**Apply to:** engine constructor (store + dump port + debounce/max-wait config), adapter constructor (dump path). Production wires `[time.Now]`; tests inject fakes. `context.Context` first param where used, never stored in structs (go-ultimate, RESEARCH §Project Constraints).

### Defensive copy at every boundary
**Source:** `internal/app/storage/store.go:114-126` (Get), `:230-238` (install), `:243-256` (Records)
**Apply to:** codec decode output (returned records own their Data), adapter byte returns, engine snapshot handoff to the codec.

### Doc comments cite decisions and requirements
**Source:** every analog — e.g. `store.go:101-103` ("corruption detection for the Phase 3 dump, never tamper protection"), `store.go:240-242` ("This is the Phase 3 dump source"), `errors.go` RPC mappings, `clock.go:42-51` (VERS-04 full-watermark seeding).
**Apply to:** all new exported symbols in persist/port/persistfile. The strict lint config enforces comment coverage; the convention is decision-ID + phase-reference prose.

### Depguard boundary (machine-enforced)
**Source:** `.golangci.yml:68-88` — `internal/app/**` (non-test) denies `net`, `os`, `syscall`, `google.golang.org/`; `internal/port/**` (non-test) allows only `$gostd` + `github.com/Djarvur/sup-kv`.
**Apply to:** codec/engine/errors live under `internal/app` (no `os` — the whole reason the adapter is separate, D-01); `port/dump.go` signatures stay stdlib-typed; all `os`/`path/filepath` imports live only in `internal/persistfile`.

### Property-test contract (seed print + replay)
**Source:** `internal/app/internal/testutil/rng.go:41-51` — `NewRNG(t)` prints `"property seed: %d,%d (replay: go test ./... -run '%s' -args -seed=%d)"`; failure messages append `(replay with the property seed above)` (`merge_test.go:86` etc.).
**Apply to:** `codec_test.go` and any seeded engine tests. `t.Parallel()` on every top-level test and outer subtest.

### Test constants + fixture helpers block
**Source:** `internal/app/storage/store_test.go:24-63`, `internal/app/clock/clock_test.go:16-63` — named const block (`testNowMs`, `testNodeID`, `cmpNewer`...), then doc-commented helpers (`newStore`, `fixedNow`, `seqNow`).
**Apply to:** both new test files; external `_test` package (`package storage_test` / `clock_test` precedent) exercising only the exported surface.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/persistfile/dumpfile.go` | adapter | file-I/O | First `os`-importing package in the repo; atomic-write sequence supplied by RESEARCH.md Pattern 3 |
| `internal/persistfile/snapshot.go` | adapter | file-I/O | Same; RESEARCH.md Pattern 6 + Security §V5 name validation |
| `internal/app/persist/engine.go` (watermark/waiter/debounce internals) | service | event-driven | Concurrent waiters/timers have no in-repo precedent (`clock.go` supplies the mutex/state idiom only); RESEARCH.md Patterns 4-5 are the design source |
| synctest bubbles + real-fs tests (test conventions) | test | event-driven / file-I/O | Neither exercised anywhere yet (RESEARCH §Wave 0 Gaps); go-ultimate + RESEARCH examples govern |

## Metadata

**Analog search scope:** `internal/app/**` (version, clock, storage, internal/testutil), `internal/port/`, `cmd/sup-kv/`, `.golangci.yml`, `docs/IDEA.md`, `.claude/CLAUDE.md`. No `.claude/skills/` or `.agents/skills/` project skills exist.
**Files scanned:** 14 source/test/config files read; all git-tracked.
**Pattern extraction date:** 2026-11-03
