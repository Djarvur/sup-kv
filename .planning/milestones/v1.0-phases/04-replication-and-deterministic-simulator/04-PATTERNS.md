# Phase 4: Replication and Deterministic Simulator - Pattern Map

**Mapped:** 2026-11-03
**Files analyzed:** 14 (9 new, 5 modified)
**Analogs found:** 12 / 14 (2 involve first-of-kind artifacts with no in-repo analog: the `.proto` schema and generated `.pb.go`; the smoke test's socket mechanics also have no analog, only its structure does)

All analog paths below are git-tracked source, verified via `git ls-files`. The repo is small (26 tracked `.go` files, ~6 000 lines) — every analog was read in full this session; excerpts carry exact line numbers.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/port/repl.go` (new) | port (seam types) | event-driven (message structs) | `internal/port/dump.go` | exact |
| `internal/port/doc.go` (modified) | doc | — | `internal/port/doc.go` + `internal/app/persist/doc.go` | exact |
| `internal/app/repl/` core `.go` (new; name planner's discretion) | service (pure decision core) | event-driven (single event loop, D-04) | `internal/app/storage/store.go` + `internal/app/clock/clock.go` + `internal/app/persist/engine.go` | role-match (composite; see concurrency caveat) |
| `internal/app/repl/` unit tests (new) | test | — | `internal/app/storage/store_test.go`, `merge_test.go` | exact |
| `internal/app/repl/` bound + chaos sim tests (new) | test (property/integration) | event-driven | `internal/app/clock/hlc_prop_test.go` | exact |
| `internal/app/internal/simnet/` (new, test-only normal package) | test utility (discrete-event scheduler + virtual network) | event-driven | `internal/app/internal/testutil/rng.go` + `gen.go` | role-match (see testing-import caveat) |
| `internal/udprepl/` adapter `.go` (new) | adapter | request-response (datagram I/O) | `internal/persistfile/dumpfile.go` + `snapshot.go` | exact |
| `internal/udprepl/replproto/supkv.proto` (new) | schema/IDL | — | none | no analog |
| `internal/udprepl/replproto/supkv.pb.go` (new, generated, committed) | generated model | — | none (lint-exempt by path pattern) | no analog |
| `internal/udprepl/` smoke test (new) | test (integration/smoke, real sockets) | request-response | `internal/persistfile/dumpfile_test.go` (structure) | role-match (socket/probe logic: no analog) |
| `internal/app/persist/engine.go` (modified — `ApplyRemote`) | service | CRUD/merge | same file: `markDirty` (lines 535-546) + `await`'s seam comment (384-386) | exact (in-file, the seam is pre-documented) |
| `internal/app/persist/engine_test.go` (modified — ApplyRemote test) | test | — | same file: `fakeDumpPort` (246-346) | exact (in-file) |
| `go.mod` (modified — tool directive + promote protobuf) | config | — | same file: existing `tool` block | exact (in-file) |
| `Makefile` (modified — `proto` target) | config/build | — | same file: `lint`/`test` targets | exact (in-file) |

## Pattern Assignments

### `internal/port/repl.go` (port, event-driven message structs)

**Analog:** `internal/port/dump.go` (40 lines — read in full)

This is the exact seam-precedent class Phase 4 D-01 names ("same contract class as Phase 3 D-02"). Copy: interface/type declarations at top, long doc comments citing decisions, explicit concurrency contract, and the `nolint` with specific linter + reason where the lint gate cannot know the intent.

**Seam declaration pattern** (`internal/port/dump.go:3-21`):

```go
// DumpStore is the whole-dump persistence seam the pure persistence core
// (internal/app/persist) saves through and the one file adapter
// (internal/persistfile) implements; tests may substitute in-memory fakes
// that record timestamps and inject failures. The seam carries whole dump
// bytes, never records — D-01/D-02: the adapter knows nothing about record
// structure, the engine knows nothing about files, and the signatures see
// only standard-library types (the sup-kv-port-stdlib-only depguard rule).
//
// Concurrency: Save serializes a single dumper goroutine's dumps — the
// engine's contract — while Load, CreateSnapshot and ReadSnapshot may be
// called independently at startup (Open) and on operator request.
//
//nolint:iface // port seams exist for other packages: the engine consumes and the adapter implements DumpStore, so no same-package usage is possible or wanted (D-01).
type DumpStore interface {
```

Phase 4 divergence (deliberate, per D-01): the seam carries **decoded structs, not bytes** — so `repl.go` is mostly plain structs (Hello/DataReplication/Ack/DataRequest/DataSync field types, inbound-datagram + outbound-message types) referencing `version.RecordVersion`/`storage.Record` where useful (own module path is allowed — `.golangci.yml:81-88`). Write the same style of "who implements / who consumes / what crosses and what never does" doc comment, citing D-01/D-02/D-03.

**Interface placement precedent (choose consciously):** two shapes exist in-repo. `port.DumpStore` lives in `port`; but `storage.Clock` is declared consumer-side in the consumer package with an explicit note (`internal/app/storage/store.go:47-54`):

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

The research's recommended structure puts message **structs** in `port/repl.go` (no interface needed — the adapter imports the concrete `repl.Core` and calls its handlers; the "Outbox" accumulator is a struct). Follow the research recommendation; cite whichever precedent the final shape matches.

---

### `internal/app/repl/` core (service, event-driven — the replication state machine)

**Analogs (composite):** `internal/app/storage/store.go` (package shape), `internal/app/clock/clock.go` (time seam), `internal/app/persist/engine.go` (timer/dirty discipline). RESEARCH.md Pattern 1 already sketches the handler signatures (`HandleDatagram`/`NotifyLocalWrite`/`Advance` with an `Outbox` accumulator) — the excerpts below are the in-repo code those sketches derive from.

**CRITICAL concurrency caveat — the store is the anti-analog on locking.** `storage.Store` uses `sync.RWMutex` on every method (`store.go:59-65, 114-126`). Phase 4 D-04 forbids that shape in the repl core: single event-loop goroutine, **no locks, no goroutines, no `time.Now`/`time.Sleep` inside the core**. Copy from `store.go` only: the Config/defaults shape, defensive copying, package doc, error style.

**Config + design-defaults pattern** (`internal/app/storage/store.go:12-27, 80-87`) — repl's Config will carry the IDEA.md parameters (`hello_interval`, `peer_timeout`, `ack_timeout`, `ack_timeout_max`, `retransmit_interval`, `sync_interval`, `sync_chunk_bytes`, `sync_response_rate`):

```go
// Design default NewStore substitutes for a zero Config field, and the
// checksum flag bytes of the RecordChecksum domain, from docs/IDEA.md.
const (
	defaultValueMaxBytes = 61440
	...
)

// Config holds the store tuning parameters; it is immutable once handed to
// NewStore. A zero or negative ValueMaxBytes is normalized to the design
// default of 61440. ...
type Config struct {
	ValueMaxBytes int
}

// normalized returns cfg with a zero or negative ValueMaxBytes replaced by
// the design default.
func (c Config) normalized() Config {
	if c.ValueMaxBytes <= 0 {
		c.ValueMaxBytes = defaultValueMaxBytes
	}
	return c
}
```

The engine does the identical dance for timing parameters (`internal/app/persist/engine.go:17-43`: `defaultDebounce = 50 * time.Millisecond`, `defaultMaxWait = time.Second`, same `normalized()` idiom) — that is the closer analog for a Config full of durations.

**Time-seam pattern** (`internal/app/clock/clock.go:3-10, 47, 79-81`) — the D-02 `now` seam the simulator relies on; repl's handlers take `now time.Time` as a parameter (research Pattern 1) rather than holding a `now func`, but the doc-comment discipline is the model:

```go
// ... The wall clock enters through exactly one seam — the now func() [time.Time] constructor parameter of
// NewClock (D-02): production passes [time.Now], tests pass a controlled
// function driving the adversarial scenarios. ...
type Clock struct {
	...
	now       func() time.Time
```

**Explicit author call, no callbacks** (D-05) — engine's write path is the precedent ("store, then confirm" → repl's "store, then broadcast call") (`internal/app/persist/engine.go:345-354`):

```go
func (e *Engine) Set(key string, data []byte) (version.RecordVersion, time.Duration, error) {
	ver, drift, err := e.store.Set(key, data)
	if err != nil {
		return ver, drift, err
	}
	return ver, drift, e.await(ver)
}
```

**Single-driver + timers pattern for the production adapter's event loop** (`internal/app/persist/engine.go:140-181`, excerpts) — Run is the one goroutine that mutates engine state; the adapter's read/timer goroutines feeding `repl.Core`'s queue mirror this shape, while the core itself stays synchronous:

```go
// Run is the single dumper goroutine the caller launches. It serializes
// every dump, debounces bursts at cfg.Debounce and forces a dump start no
// later than the oldest unconfirmed write plus cfg.MaxWait ...
func (e *Engine) Run(ctx context.Context) error {
	var debounce, deadline *time.Timer
	for {
		var fired bool
		select {
		case <-ctx.Done():
			stopTimer(debounce)
			...
			return ctx.Err()
		case <-e.changed:
		case <-timerChan(debounce):
			debounce = nil
			fired = true
		case <-timerChan(deadline):
			deadline = nil
			fired = true
		case reply := <-e.flushCh:
			reply <- e.flushDump()
		}
		if fired {
			_ = e.runDump() // the error belongs to the waiters; nobody is left to report to.
		}
		debounce, deadline = e.rearm(debounce, deadline)
	}
}
```

Also copy `timerChan`/`stopTimer` nil-timer helpers (`engine.go:328-343`) if the adapter uses `*time.Timer` in selects, and the Go 1.23+ timer semantics note (no drain selects).

**Backoff/growth-table precedent** — the clock's counter overflow and saturation rules (`internal/app/clock/clock.go:100-109`) are the in-repo model for repl's documented backoff ladder (200ms doubling capped at 5s): explicit constants, every step of the ladder spelled out in the doc comment, saturate-never-wrap.

**Sorted-map-iteration determinism (mandatory in every emit path)** — the exact idiom, with its reason documented (`internal/app/persist/engine.go:465`):

```go
	for _, key := range slices.Sorted(maps.Keys(records)) {
```

and the same discipline in the codec with the full rationale (`internal/app/persist/codec.go:56-61, 70`):

```go
// Encode writes records as the JSONL dump (PERS-01, D-06): the header
// object on line 1, then one record object per line sorted by key — the
// sort is mandatory because Go map iteration order is random and byte
// stability across generations of the same state is the contract.
```

**Defensive copying** (`internal/app/storage/store.go:228-238`) — applies to any record data the core buffers in pending entries:

```go
func (s *Store) install(key string, rec Record) {
	data := bytes.Clone(rec.Data)
	s.records[key] = Record{
		Version:  rec.Version,
		Data:     data,
		...
	}
}
```

**Merge/Ack trigger surface** — the store APIs the core drives (`store.go:189-205, 243-256`): `Merge(key, rec) (conflicts, drift, err)` is idempotent/commutative (duplicate/reordered datagrams are no-ops — the protocol's inherent defense), and `Records() map[string]Record` (tombstones included) is the DataSync compare source.

---

### `internal/app/repl/` unit tests (test)

**Analog:** `internal/app/storage/store_test.go` (table/black-box) + `merge_test.go` (property). Both read in full.

**Black-box test file shape** (`internal/app/storage/store_test.go:1-63`) — external test package, shared constants block with a comment naming each constant's origin, `t.Parallel()` everywhere, helper constructors:

```go
package storage_test

import (
	...
	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/internal/testutil"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// Shared test constants: a fixed wall-clock millisecond (the D-02 seam), the
// test node ID, the default value limit from docs/IDEA.md parameters, ...
const (
	testNowMs      = int64(1730000000000)
	testNodeID     = uint16(0xABCD)
	...
)

// fixedNow returns a now func parked at the fixed test wall-clock
// millisecond, built with time.UnixMilli so no monotonic reading is involved
// (D-02).
func fixedNow() func() time.Time {
	return func() time.Time { return time.UnixMilli(testNowMs) }
}
```

Note `fixedNow` built via `time.UnixMilli` (no monotonic reading) — duplicated in `testutil/gen.go:27-29`; the virtual-clock construction for simulator nodes follows it.

**Property-test shape** (`internal/app/storage/merge_test.go:126-136`):

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

Failure messages end with the replay reminder: `"step %d ... (replay with the property seed above)"` (`merge_test.go:86` etc.). Reuse `testutil.RandVersion`/`RandRawVersion`/`RandRecord`/`RandKey`/`RandStore` for classification and chunking tests; `RandRawVersion` (`gen.go:43-49`) is the specifically-cited precedent for adversarial 16-byte inputs (untrusted-datagram classification must never panic on any bit pattern — see also `persist/codec.go:126-131` decodeRecord's totality comment).

---

### `internal/app/repl/` bound + chaos simulator tests (test, property over virtual time)

**Analog:** `internal/app/clock/hlc_prop_test.go` (312 lines — read in full).

The named bound tests are adversarial walks against a **model function** — exactly `expectedDrift` (`hlc_prop_test.go:17-25`):

```go
// expectedDrift is the documented drift convention as a model: evaluated on
// the pre-event state, the exact signed divergence when l and the wall clock
// are apart beyond warn in either direction, zero within tolerance (positive
// when l runs ahead, negative on a forward wall-clock jump).
func expectedDrift(lBefore, ptBefore int64, warn time.Duration) time.Duration {
```

driving a long randomized walk with per-step assertions (`hlc_prop_test.go:35-67`, the 2 000-step `TestClock_monotonic`). The repl bound tests are the same shape: a documented-bound model (`one RTT`, `≤ ack_timeout_max`, `≤ 2×sync_interval + response`) asserted per virtual-time step over the simnet scheduler. Note the walk's wall clock is a local variable advanced by rng (`nowMs = nextWallTime(rng, nowMs)` then `now := func() time.Time { return time.UnixMilli(nowMs) }`, lines 40-42) — the manual-virtual-clock idiom the simulator generalizes.

Chaos test = `TestMerge_laws` shape (seeded loop, replay-line failure messages) plus coverage assertions like `TestClock_driftUnderChaos`'s both-signs-observed check (`hlc_prop_test.go:304-311`: `observedPositive = observedPositive || drift > 0` ... then fail unless both fired) — useful to prove the fault injector actually dropped/duplicated/partitioned during the run.

**Seeding/replay:** reuse `testutil.NewRNG(t)` verbatim (see Shared Patterns). RESEARCH Open Question 3 (flag vs env var) — the `-seed` flag at `rng.go:35` is the precedent D-06's own text points to; whatever the planner picks, the binding requirement is the printed copy-paste re-run command (`rng.go:48-49`).

---

### `internal/app/internal/simnet/` (test utility, discrete-event scheduler + virtual network)

**Analog:** `internal/app/internal/testutil/` (`rng.go` 51 lines, `gen.go` 85 lines — both read in full).

**Test-only normal package precedent** (`rng.go:1-14`) — package doc explains exactly why a non-`_test.go` package may exist inside `internal/app` and how the lint gate permits it:

```go
// Package testutil holds the property-test generators shared by the
// internal/app packages: the seeded random source behind the printed-seed
// and -seed replay contract (D-04), plus version, record, key and store
// generators. It is a normal package linked only into test binaries; the
// no-test-libs-in-prod lint rule exempts the testutil directory, which is
// what allows the testing import here.
package testutil
```

**Load-bearing caveat the planner must know:** the `no-test-libs-in-prod` depguard exemption matches directories `**/*test/**` and `**/test*/**` (`.golangci.yml:52-60`). `testutil` qualifies **by name**; `simnet` does **not**. Therefore a normal package at `internal/app/internal/simnet/` must NOT import `testing` (or any test-only lib) in its non-test files — keep simnet testing-free (rng and config passed in as parameters; assertions live in `_test.go` files), or name/place the directory so the glob matches. `net`/`os`/`syscall`/`google.golang.org/*` remain denied there regardless (`sup-kv-app-no-io`, `.golangci.yml:68-80`) — a virtual network needs none of them.

**Seeded rng + printed replay line** (`rng.go:31-51`) — the D-06/D-07 discipline, quote:

```go
var seedFlag = flag.Int64("seed", 0, "replay a property test with a fixed seed") //nolint:gochecknoglobals // test-only flag; package-level registration is the standard -seed replay hook (D-04)

func NewRNG(t *testing.T) *rand.Rand {
	t.Helper()
	seedNano := *seedFlag
	if seedNano == 0 {
		seedNano = time.Now().UnixNano()
	}
	hi, lo := uint64(seedNano>>seedHalfBits), uint64(seedNano)&(1<<seedHalfBits-1)
	t.Logf("property seed: %d,%d (replay: go test ./... -run '%s' -args -seed=%d)",
		hi, lo, t.Name(), seedNano)
	return rand.New(rand.NewPCG(hi, lo)) //nolint:gosec // G404: deterministic seeded randomness is the D-04 contract, not a crypto secret
}
```

**Generator style** (`gen.go:13-39`) — tuning constants block with per-constant comments, rng-parameter constructors, `//nolint:gosec // G115:` on every narrowing conversion with the bound spelled out. Simnet's fault-draw helpers (`Drop? Duplicate? Delay?`) belong in this style.

**Fake port/recordings precedent for the sim's driver side** — `fakeDumpPort` (`internal/app/persist/engine_test.go:246-346`): a mutex-guarded recorder with injectable failures (`failSaves`) and a parking `gate` channel. The virtual network's per-node inboxes/fault counters mirror this accessor style (`saveCount()`, `save(i)`, `snapshot(name)` helpers, `t.Helper()` not needed since they are type methods).

---

### `internal/udprepl/` adapter (adapter, datagram request-response)

**Analog:** `internal/persistfile/dumpfile.go` (174 lines) + `snapshot.go` (106 lines) — both read in full. The single existing exemplar of "adapter outside the app boundary".

**Adapter package doc** (`internal/persistfile/dumpfile.go:1-19`) — names the depguard-justified role ("the only os-importing persistence code"), cites decisions, documents the locking/serialization contract:

```go
// Package persistfile is the sup-kv dump file adapter: the only
// os-importing persistence code, implementing port.DumpStore with atomic
// writes, .prev rotation and the fallback chain (D-01/D-02). ...
package persistfile

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/Djarvur/sup-kv/internal/port"
)
```

**Interface-compliance assertion** (`dumpfile.go:45-46`) — udprepl asserts the same way against whatever seam type it implements (or against the core's driver contract):

```go
// FileStore implements the whole-dump persistence seam.
var _ port.DumpStore = (*FileStore)(nil)
```

**Constants block with decision refs** (`dumpfile.go:22-31`): mode/suffix constants each with a one-line rationale. Repl's adapter: socket buffer 65 507 (the truncation pitfall), `SO_REUSEADDR`/`SO_BROADCAST`, broadcast handle, supported proto version.

**Sentinel errors** (`internal/persistfile/snapshot.go:19-28`) — errors-as-values with doc comments, matched by `errors.Is`:

```go
// Sentinel errors of the snapshot half of the adapter.
var (
	// ErrNoDump is returned by CreateSnapshot when no main dump exists on
	// disk to copy.
	ErrNoDump = errors.New("no dump on disk to snapshot")
	...
)
```

**Validate-before-touch + nolint discipline** (`snapshot.go:96-106, 65`): every `//nolint` names its linter and carries the reason (AGENTS.md deviation 3 — `nolintlint.require-explanation`/`require-specific` enforce it):

```go
func validateName(name string) error {
	if name == "" || len(name) > maxSnapshotName || name == "." || name == ".." {
		return ErrInvalidSnapshotName
	}
	...
}
```

```go
	return os.ReadFile(snapshotPath) //nolint:gosec // G304: the joined path is a validated name under the configured snapshots dir
```

The adapter's mechanical decode path (CRC → envelope → body, own-echo source handle attached, per D-01/D-02) has no in-repo codec analog outside `internal/app` — the pure-codec shape to mirror mechanically is `internal/app/persist/codec.go` (see Shared Patterns), but with `proto.Marshal`/`proto.Unmarshal` since the JSON purity trick does not transfer.

---

### `internal/udprepl/` smoke test (test, real sockets)

**Analog:** `internal/persistfile/dumpfile_test.go:1-80` for structure only — external `package persistfile_test`, constants block with rationale comments, plain helpers with `t.Fatalf`, `t.Parallel()`, `t.TempDir()`. The socket capability probe, subnet-broadcast fallback, and documented `t.Skipf` (RESEARCH Pitfall 3) have **no in-repo analog** — take them from RESEARCH.md Pattern 3/Pitfall 3 (probe-verified this session). Do not assert unicast delivery between same-port nodes (Pitfall 2).

**White-box internal-test option** (`internal/persistfile/dumpfile_internal_test.go:1-32`): the `*_internal_test.go` filename is the configured testpackage exemption (`.golangci.yml:426-427`: `testpackage: skip-regexp: .*_internal_test\.go`); its header comment explains why the test is white-box "on purpose". Use only if the smoke test must reach adapter internals.

---

### `internal/app/persist/engine.go` modification — `ApplyRemote` (service, CRUD/merge)

**Analog:** the modification site itself — the seam is pre-documented in the file.

**The reserved seam comment** (`internal/app/persist/engine.go:383-386`, inside `await`'s doc):

```go
// The dirty-mark path is generic — any state change dirties; only local writes
// register waiters, and Phase 4's remote merges will reuse the dirty path
// without waiters.
```

**`markDirty` — the exact body ApplyRemote's dirty half copies** (`engine.go:535-546`):

```go
// markDirty marks the engine dirty from a fresh epoch without registering
// a waiter: the restore replay's store writes are unconfirmed until the
// final flush's covering dump — the cover discipline of await without the
// per-write blocking.
func (e *Engine) markDirty() {
	e.mu.Lock()
	if !e.dirty {
		e.oldest = time.Now()
	}
	e.dirty = true
	e.mu.Unlock()
}
```

`ApplyRemote(key, rec)` = `e.store.Merge(key, rec)` + `e.markDirty()`, no waiter, errors surfaced unchanged (the `Restore` path at `engine.go:456-473` shows the merge-then-markDirty-then-flush composition; ApplyRemote skips the flush — the running `Run` picks up the dirty flag). Note `markDirty` uses `time.Now()` — legal here because the engine is production-side, not the repl core; do not copy `time.Now` into `internal/app/repl`.

### `internal/app/persist/engine_test.go` modification — ApplyRemote test

**Analog:** the same file's `fakeDumpPort` (`engine_test.go:246-346`, excerpted above under simnet) — the test asserts "remote merge triggers a dump" by counting `saveAttempts()`/`saveCount()` after `ApplyRemote`, exactly how the existing suite observes dump behavior.

### `go.mod` modification

**Analog:** the existing tool block (`go.mod:5-8`):

```
tool (
	github.com/golangci/golangci-lint/v2/cmd/golangci-lint
	golang.org/x/vuln/cmd/govulncheck
)
```

Add `google.golang.org/protobuf/cmd/protoc-gen-go` as one line; `go mod tidy` promotes `google.golang.org/protobuf v1.36.12 // indirect` to direct (probe-verified byte-diff: exactly that, go.sum unchanged — RESEARCH "Installation").

### `Makefile` modification

**Analog:** target convention (`Makefile:1-20`): `.PHONY` line lists all targets, each target preceded by a `## name:` explanation comment with decision/pitfall references. The `proto` target from RESEARCH's Code Examples follows it (build plugin via `go build -o`, run system protoc with `--plugin` + `paths=source_relative`).

## Shared Patterns

### Decision-referencing doc comments
**Source:** every file cited above — e.g. `internal/port/dump.go:7-8` ("D-01/D-02"), `internal/app/persist/engine.go:384-386`, `internal/persistfile/dumpfile.go:3`.
**Apply to:** all new files. Every exported (and most unexported) declarations carry a doc comment naming the governing decision/requirement IDs (D-01..D-08, REPL-01..10, TEST-03) and, where relevant, the IDEA.md parameter it encodes. This is the project's strongest and most uniform convention.

### Purity boundary (machine-enforced)
**Source:** `.golangci.yml:68-88` (quoted verbatim in RESEARCH).
**Apply to:** `internal/app/repl` and `internal/app/internal/simnet` — no `net`, `os`, `syscall`, `google.golang.org/*` (`!$test` exempts `_test.go` files only); `internal/port/repl.go` — `$gostd` + `github.com/Djarvur/sup-kv` only; protobuf lives only under `internal/udprepl/replproto/` (the `**/internal/app/**` glob would deny its own imports if placed under app). Plus `.golangci.yml:52-60` `no-test-libs-in-prod` — no `testing` import outside `_test.go` unless the directory matches `test*/`.

### Sentinel errors + errors.Is
**Source:** `internal/app/storage/errors.go:5-19`, `internal/app/persist/errors.go:5-18`, `internal/persistfile/snapshot.go:19-28`.
**Apply to:** adapter send/decode failures; core discard classes are **counters, not errors** (D-02) — model them as an enum + counter set, not sentinel errors.

### Config struct + normalized() design defaults
**Source:** `internal/app/storage/store.go:80-87`, `internal/app/persist/engine.go:33-43`.
**Apply to:** repl core Config (IDEA.md §Parameters defaults) — zero/negative fields normalize to design defaults, constants named `default*` with docs citing IDEA.md.

### Deterministic iteration (sorted maps/slices) in every emit path
**Source:** `internal/app/persist/engine.go:465`, `internal/app/persist/codec.go:70` (`slices.Sorted(maps.Keys(...))`).
**Apply to:** repl core everywhere it produces output (DataSync chunking, retransmit batches, rate-limited queues, per-peer fan-out) — unsorted map iteration makes the chaos test irreproducible (RESEARCH Pitfall 1). Scheduler ties break on `(virtualTime, seq)`.

### Seeded rng + printed replay command
**Source:** `internal/app/internal/testutil/rng.go:31-51`.
**Apply to:** simulator seeding (D-06/D-07) — `testutil.NewRNG(t)` verbatim; failure output prints the copy-paste re-run line.

### now seam / virtual clocks built with time.UnixMilli
**Source:** `internal/app/clock/clock.go:4-6` (seam), `internal/app/internal/testutil/gen.go:27-29` and `internal/app/storage/store_test.go:45-50` (fake clocks, no monotonic reading).
**Apply to:** repl core handlers take `now time.Time`; simulator nodes' stores get fake `now` funcs advanced by the scheduler; production adapter passes `time.Now()`.

### nolint discipline
**Source:** `internal/app/internal/testutil/rng.go:35,50`, `internal/persistfile/dumpfile.go:76`, `internal/persistfile/snapshot.go:65`.
**Apply to:** any suppression — specific linter name + human-readable reason, always (AGENTS.md deviation 3).

### Defensive copying at ownership boundaries
**Source:** `internal/app/storage/store.go:230-238` (`install`), `store.go:243-256` (`Records`).
**Apply to:** port structs carrying `[]byte` record data crossing the core/adapter seam — clone on the way in and out; datagram receive buffers are reused per read, so the adapter must copy bytes it hands to the core.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/udprepl/replproto/supkv.proto` | schema/IDL | — | First protobuf schema in the repo; take the Envelope/Body shape from RESEARCH Pattern 2 (probe-verified round-trip) |
| `internal/udprepl/replproto/supkv.pb.go` | generated code | — | First generated file; committed, carries the standard `// Code generated by protoc-gen-go. DO NOT EDIT.` header which `generated: strict` + the `issues.exclusions` path pattern `\.[\w-]+\.go$` (`.golangci.yml:492,519,521`) recognize — the filename must match that pattern |
| smoke-test socket mechanics (within `internal/udprepl/` smoke test) | test | request-response | No real-socket code exists in-repo; capability probe + subnet-broadcast fallback + loud `t.Skipf` come from RESEARCH Pattern 3 / Pitfalls 2-3 (live-probe-verified) |
| virtual-network fault injection (within `simnet`) | test utility | event-driven | No discrete-event scheduler exists in-repo; `(virtualTime, seq)` queue + send-time fault draws per RESEARCH Pattern 4; nearest in-repo spirit is `fakeDumpPort`'s injectable failures |

## Metadata

**Analog search scope:** entire repo (`internal/`, `cmd/`, `Makefile`, `go.mod`, `.golangci.yml`); `.gomodcache/` excluded (untracked tool mirror). All 26 tracked `.go` files enumerated; 13 read in full, 1 (engine_test.go, 1 284 lines) read via targeted range.
**Tracked-source check:** every analog path above verified non-empty under `git ls-files -- <path>`.
**Pattern extraction date:** 2026-11-03
