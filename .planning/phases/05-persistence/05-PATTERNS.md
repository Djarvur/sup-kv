# Phase 5: Persistence - Pattern Map

**Mapped:** 2026-11-22
**Files analyzed:** 47 (24 shipped/docs + 23 test files)
**Analogs found:** 47 / 47 (44 exact or role-match; 3 partial with the gap named)

All analog paths below were verified git-tracked with `git ls-files -- <path>`; no
gitignored mirror/install path is cited.

---

## Frozen-contract guardrails (read before using any pattern below)

These are the contracts the planner must encode as hard constraints on every task.
They are not "preferences"; breaking any one either fails a frozen red test or breaks
the depguard architecture gate.

| Contract | Evidence | Consequence |
|----------|----------|-------------|
| `persist.Open(...)` MUST keep returning `(*Engine, int)` | `internal/persistfile/red_value_limit_test.go:51-57` destructures `(eng, discarded)`; frozen by D-01/plan 02-12 | The load report rides `Engine.LoadStats()`, never a changed return type |
| `persist.Config` and `storage.Config` keyed literals in frozen tests | `red_value_limit_test.go:50-55`, `open_seed_test.go:199-203`, `engine_test.go:1683-1686` | Adding fields is safe; renaming/removing fields is not |
| `Engine`'s exported method set is compiled by `grpcapi`, `udprepl`, `cmd`, `persistfile` tests | `internal/app/persist/engine.go:393-558`; `internal/grpcapi/server.go:207-268,342` | The five-role split is internal-only; no exported signature changes |
| Depguard `sup-kv-app-no-io` | `.golangci.yml:68-80` | `internal/app/**` may not import `net`, `os`, `syscall`, `google.golang.org/*`; errno classification and file work stay in `persistfile` |
| Depguard `sup-kv-port-stdlib-only` | `.golangci.yml:81-88` | `internal/port` may import only stdlib + own module; every new enum/result lives there |
| Only the two P-01 rows in `scripts/red-expectations.tsv` flip | `scripts/red-expectations.tsv:29-30`; `scripts/red-runner:1-40` | Never edit the runner or frozen tests; flip `red` → `green` only |
| `.golangci.yml` is vendored byte-unmodified | `AGENTS.md` deviation 2 | No new lint settings; fix shapes, never add `//nolint` without a named linter + reason |
| `fsync_darwin.go` does not exist | `find . -name 'fsync*'` → no results | P-09's remaining work is sweep/parent-dir/tolerance, not deleting a file |

---

## File Classification

### Shipped and documentation files

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/app/storage/store.go` (mod) | model/store | CRUD + merge | itself (current) | exact |
| `internal/app/persist/codec.go` (mod) | codec | transform, streaming | itself (current) | exact |
| `internal/app/persist/engine.go` (mod) | service/engine | event-driven, request-response | itself (current) | exact |
| `internal/app/persist/loader.go` (new) | service/loader | file-I/O + transform (startup) | `persist.Open` + `mergeRecords` in `engine.go:112-164` | exact (move) |
| `internal/app/persist/facade.go` (new) | service/facade | request-response | pass-throughs + `ApplyRemote` in `engine.go:393-558` | exact (move) |
| `internal/app/persist/scheduler.go` (new) | service/scheduler | event-driven | `Run`/`rearm`/`runDump`/stats in `engine.go:166-391` | exact (move) |
| `internal/app/persist/confirm.go` (new) | service/confirmation | event-driven, request-response | `waiter`/`await`/`confirm*`/`fail*`/`sweepClosed` | exact (move) |
| `internal/app/persist/snapshot.go` (new) | service/snapshot | file-I/O, batch | `Snapshot`/`Restore`/`replay*` in `engine.go:560-725` | exact (move) |
| `internal/app/persist/errors.go` (mod) | utility | — | itself; `storage/errors.go` twin | exact |
| `internal/app/persist/doc.go` (mod) | config/doc | — | itself; `storage/doc.go` | exact |
| `internal/port/dump.go` (mod) | port/interface | request-response | itself; `port/repl.go` enums | exact |
| `internal/persistfile/dumpfile.go` (mod) | adapter | file-I/O | itself | exact |
| `internal/persistfile/fsops.go` (mod) | adapter | file-I/O | itself | exact |
| `internal/persistfile/saveerror.go` (new) | adapter/utility | file-I/O (error class) | `readMainDump` errno→sentinel (`snapshot.go:68-76`) | partial |
| `internal/persistfile/snapshot.go` (mod) | adapter | file-I/O | itself | exact |
| `cmd/sup-kv/main.go` (mod) | config/wiring | startup, batch | itself | exact |
| `internal/grpcapi/server.go` (mod) | controller | request-response | itself (`Set`/`Delete`/`mapError`) | exact |
| `internal/app/repl/core.go` (mod) | service/domain | event-driven | itself (`applyRemote`, `Advance`) | exact |
| `internal/app/internal/simnet/simnet.go` (mod) | test harness | event-driven | itself (`Cluster.Write`/`drive`/`route`) | exact |
| `internal/metrics/metrics.go` (mod) | adapter/exporter | request-response | itself (`Sources`/`dumpFamilies`) | exact |
| `docs/adr/0012-equal-version-tiebreak.md` (new) | documentation | — | `docs/adr/0011-value-limit-60kib-kept.md` | exact |
| `docs/OPERATIONS.md` (mod) | documentation | — | itself (limits/monitoring/lifecycle) | exact |
| `docs/SPEC-DEVIATIONS.md` (mod) | documentation | — | itself (row flips) | exact |
| `scripts/red-expectations.tsv` (mod) | config/test gate | — | itself | exact |

### Test files

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/app/persist/engine_test.go` (mod) | test (engine) | event-driven | itself: `fakeDumpPort`, `startDumper`, synctest suite | exact |
| `internal/app/persist/engine_blocking_test.go` (mod) | test (blocking) | event-driven | itself | exact |
| `internal/app/persist/codec_test.go` (mod) | test (codec) | transform | itself: `rewriteLine`, discard table | exact |
| `internal/app/persist/fuzz_dump_test.go` (mod) | test (fuzz) | transform | itself | exact |
| `internal/app/persist/bench_test.go` (mod) | test (benchmark) | streaming | itself: `BenchmarkDump10kKeys` | exact |
| `internal/app/persist/load_stats_test.go` (new) | test (engine) | startup | `engine_test.go` fakes + `open_seed_test.go` | role-match |
| `internal/app/persist/timeout_test.go` (new) | test (engine) | event-driven | `engine_blocking_test.go` | role-match |
| `internal/app/persist/backoff_test.go` (new) | test (engine) | event-driven | `TestEngine_dumpFailureFansOut` (`engine_test.go:492-563`) | role-match |
| `internal/app/repl/retransmit_test.go` (mod) | test (wire law) | request-response | itself: `TestReplication_equalVersionAcked` `:1346-1373` | exact |
| `internal/app/persist/encode_stream_test.go` (new) | test (codec) | streaming | `codec_test.go` property helpers | role-match |
| `internal/app/storage/store_test.go` (mod) | test (store) | CRUD | itself: equal-version subtest `:510-530` | exact |
| `internal/app/storage/merge_test.go` (mod) | test (merge) | CRUD | itself: `TestMerge_losersProperty` `:222-261` | exact |
| `internal/persistfile/load_outcome_test.go` (new) | test (adapter) | file-I/O | `dumpfile_test.go:93-154` fallback table | role-match |
| `internal/persistfile/sweep_test.go` (new) | test (adapter) | file-I/O | `dumpfile_test.go` + `fsops_test.go` helpers | partial |
| `internal/persistfile/saveerror_test.go` (new) | test (adapter) | file-I/O | `fsops_test.go` table/failure fakes | partial |
| `internal/persistfile/dumpfile_test.go` (mod) | test (adapter) | file-I/O | itself | exact |
| `internal/persistfile/fsops_test.go` (mod) | test (adapter) | file-I/O | itself: order + failure fakes | exact |
| `internal/app/repl/sim_dump_test.go` (new) | test (sim) | event-driven | `internal/app/repl/sim_scenario_test.go` + `simnet_ext_test.go` | role-match |
| `internal/app/repl` core equal-version tests (new/mod) | test (core) | event-driven | `repl` core scenario tests | role-match |
| `internal/stand/persist_test.go` (new, Linux) | test (process) | file-I/O | `stand_red_test.go:884-962` + `stand_metrics_test.go` | role-match |
| `cmd/sup-kv/main_test.go` (mod) | test (binary) | startup | `TestMain_valueMaxDatagram` `:509-577` | exact |
| `internal/grpcapi/server_test.go` (mod) | test (RPC) | request-response | `fakeDumpStore:61-109` | exact |
| `internal/metrics/metrics_test.go` (mod) | test (exporter) | request-response | `zeroSources`/`families` `:196-283` | exact |

---

## Pattern Assignments

### `internal/app/storage/store.go` (model/store, CRUD + merge)

**Analog:** itself — the phase edits this file in place.

**Imports pattern** (lines 3-10):

```go
import (
	"bytes"
	"hash/crc32"
	"sync"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/version"
)
```

**Single-source constant block** (lines 12-18) — replace `defaultValueMaxBytes` with the
one exported limit; keep the checksum flag bytes beside it:

```go
// Design default NewStore substitutes for a zero Config field, and the
// checksum flag bytes of the RecordChecksum domain, from docs/IDEA.md.
const (
	defaultValueMaxBytes = 61440
	notDeletedFlag       = 0
	deletedFlag          = 1
)
```

**Config normalization pattern** (lines 118-128) — `ValueLimit()` must be the same helper
`normalized` and the pre-store decode use:

```go
// normalized returns cfg with a zero or negative ValueMaxBytes replaced by
// the design default and a nil OnChange replaced by a no-op.
func (c Config) normalized() Config {
	if c.ValueMaxBytes <= 0 {
		c.ValueMaxBytes = defaultValueMaxBytes
	}
	if c.OnChange == nil {
		c.OnChange = func(_ string, _ Record) {}
	}
	return c
}
```

**Write-side enforcement** (lines 217-225) — the model for `Store.ValueMaxBytes()`
(field is immutable after `NewStore`, lines 109-116):

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

**The join to extend with the `(version, content)` law** (lines 324-343):

```go
func (s *Store) join(key string, rec Record) []Conflict {
	stored, ok := s.records[key]
	if !ok {
		s.install(key, rec)
		return nil
	}
	cmp := rec.Version.Compare(stored.Version)
	switch {
	case cmp > 0:
		s.install(key, rec)
		return []Conflict{{Key: key, Winner: rec.Version, Loser: stored.Version}}
	case cmp < 0:
		return []Conflict{{Key: key, Winner: stored.Version, Loser: rec.Version}}
	default:
		return nil
	}
}
```

**Content-equality precedent for `RecordsEqual`** — `restoreChanges`
(`internal/app/persist/engine.go:685-693`) already compares exactly the tuple the new
helper must expose:

```go
func restoreChanges(current, snap map[string]storage.Record) bool {
	for key, rec := range snap {
		stored, ok := current[key]
		if !ok || stored.Version != rec.Version || stored.Deleted != rec.Deleted || !bytes.Equal(stored.Data, rec.Data) {
			return true
		}
	}
	return false
}
```

**Install convergence point** (lines 345-364) — a tie-break install automatically bumps
the generation and fires `OnChange`, so no extra wiring:

```go
func (s *Store) install(key string, rec Record) {
	data := bytes.Clone(rec.Data)
	stored := Record{
		Version:  rec.Version,
		Data:     data,
		Deleted:  rec.Deleted,
		Checksum: RecordChecksum(key, rec.Version, data, rec.Deleted),
	}
	s.records[key] = stored
	s.generation++
	s.cfg.OnChange(key, stored)
}
```

**`MergeChanged` shape** — the existing `Merge` (lines 271-287) is the wrapper:
`MergeChanged` returns `(conflicts, changed, drift, err)`; `Merge` delegates to it and
drops `changed`. Do not change `Merge`'s three-result signature (62 call sites per
research A9).

**Testing analog:** `internal/app/storage/store_test.go:510-530` (`"equal version is a
no-op with no conflicts"`) and `internal/app/storage/merge_test.go:222-261`
(`TestMerge_losersProperty`) are the two tests that must be rewritten. Keep
`TestMerge_duplicateDelivery` (`merge_test.go:194-220`) green.

---

### `internal/app/persist/codec.go` (codec, transform + streaming)

**Analog:** itself.

**Format constants and error sentinel** (lines 17-34) — `maxRecordBytes` is deleted; the
limit becomes a parameter:

```go
const dumpFormatVersion = 1
const maxRecordBytes = 61440 // DELETE: becomes the configured valueMaxBytes parameter

var errUnsupportedFormat = errors.New("persist: dump format version unsupported")
```

**Header and record line** (lines 34-54) — add `RecordCount *int` to `header`; keep the
record field tags byte-stable (the fuzz corpus pins them):

```go
type header struct {
	FormatVersion int `json:"format_version"`
}
type recordLine struct {
	Key      string `json:"key"`
	Version  string `json:"record_version"`
	Data     []byte `json:"data"`
	Deleted  bool   `json:"deleted"`
	Checksum uint32 `json:"checksum"`
}
```

**Deterministic writer** (lines 62-85) — `Encode` becomes `Encode(records, valueMaxBytes)`
and `EncodeTo(w, records, valueMaxBytes)`; the sorted-key walk is the contract:

```go
func Encode(records map[string]storage.Record) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(header{FormatVersion: dumpFormatVersion})
	if err != nil {
		return nil, err
	}
	for _, key := range slices.Sorted(maps.Keys(records)) {
		rec := records[key]
		line := recordLine{ /* ... */ }
		err = enc.Encode(line)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}
```

**Current decoder to replace** (lines 97-124) — this is exactly the `json.Decoder` shape
Pitfall 12 forbids:

```go
func Decode(dump []byte) (records map[string]storage.Record, discarded int, err error) {
	dec := json.NewDecoder(bytes.NewReader(dump))
	var h header
	err = dec.Decode(&h)
	if err != nil {
		return nil, 0, fmt.Errorf("persist: dump header unreadable: %w", err)
	}
	if h.FormatVersion != dumpFormatVersion {
		return nil, 0, errUnsupportedFormat
	}
	records = make(map[string]storage.Record)
	for {
		var line recordLine
		err = dec.Decode(&line)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return records, discarded, nil
			}
			return nil, 0, fmt.Errorf("persist: dump record unreadable: %w", err)
		}
		rec, ok := decodeRecord(line)
		if !ok {
			discarded++
			continue
		}
		records[line.Key] = rec
	}
}
```

The new decoder is line-oriented: `bufio.NewReaderSize` + `ReadSlice('\n')` +
`json.Unmarshal` per line, header `record_count` census, four-class `DecodeStats`.
The research sketch is `05-RESEARCH.md` "Dump decode with a completeness census"
(lines ~1572-1621); the real codebase constraints come from `decodeRecord`
(lines 132-151) — keep the four validation steps (hex length, value limit, checksum)
and their order:

```go
func decodeRecord(line recordLine) (storage.Record, bool) {
	raw, err := hex.DecodeString(line.Version)
	if err != nil || len(raw) != versionBytes {
		return storage.Record{}, false
	}
	var ver version.RecordVersion
	copy(ver[:], raw)
	if len(line.Data) > maxRecordBytes { // becomes the passed valueMaxBytes
		return storage.Record{}, false
	}
	if storage.RecordChecksum(line.Key, ver, line.Data, line.Deleted) != line.Checksum {
		return storage.Record{}, false
	}
	// ...
}
```

**Testing analog:** `codec_test.go:58-77` (`rewriteLine`) already splits the dump into
lines for tamper fixtures — the new census tests can build on it. `codec_test.go:494-513`
pins the mid-line truncation error; `fuzz_dump_test.go:44-99` pins the exact invariants
and the header literal `fuzzDumpHeader = "{\"format_version\":1}\n"` (`:49`).

---

### `internal/app/persist/engine.go` (service/engine, event-driven + request-response)

**Analog:** itself. The file shrinks to `Engine`, `New`, `Config`, `Flush`,
`FlushTimeout`, `flush`.

**Config + normalization** (lines 20-46) — add `ConfirmTimeout`, `Now`, `After`; fill
nil funcs here (no package-level globals, `gochecknoglobals`):

```go
const (
	defaultDebounce = 50 * time.Millisecond
	defaultMaxWait  = time.Second
)

type Config struct {
	Debounce time.Duration
	MaxWait  time.Duration
}

func (c Config) normalized() Config {
	if c.Debounce <= 0 {
		c.Debounce = defaultDebounce
	}
	if c.MaxWait <= 0 {
		c.MaxWait = defaultMaxWait
	}
	return c
}
```

**Nil-func normalization pattern** — copy the repl core's
`internal/app/repl/core.go:337-343`:

```go
if c.Log == nil {
	c.Log = func(_ string) {}
}
if c.OnDiscard == nil {
	c.OnDiscard = func(_, _ string) {}
}
```

**Waiter ticket** (lines 48-58) — add `claimed bool` for the single-claim protocol:

```go
type waiter struct {
	done chan struct{}
	err  error
}
```

**Engine struct** (lines 60-87) — the composition root after the split; keep the three
stats atomics and the four channels:

```go
type Engine struct {
	mu      sync.Mutex
	store   *storage.Store
	dumps   port.DumpStore
	cfg     Config
	waiters []*waiter
	dirty   bool
	oldest  time.Time
	changed chan struct{}
	flushCh chan chan error
	runDone chan struct{}
	closed  bool

	dumpNanos  atomic.Int64
	dumpTotal  atomic.Uint64
	dumpErrors atomic.Uint64
}
```

**Startup composition to move to `loader.go`** (lines 112-164):

```go
func Open(...) (eng *Engine, discarded int) {
	var records map[string]storage.Record
	decode := func(dump []byte) error {
		decoded, count, err := Decode(dump)
		if err != nil {
			return err
		}
		records, discarded = decoded, count
		return nil
	}
	dumps.Load(decode)
	if records == nil {
		records = make(map[string]storage.Record)
	}
	initialL, initialC := Watermark(records)
	clk := clock.NewClock(now, initialL, initialC, driftWarn)
	store := storage.NewStore(nodeID, clk, storeCfg)
	mergeRecords(store, records)
	return New(store, dumps, engineCfg), discarded
}
```

The new loader computes `limit := storeCfg.ValueLimit()` before `Decode`, switches to
`LoadResult`, builds `LoadStats`, and marks the engine dirty for
`Prompt`/`Backup`/`Unusable`.

**Timer loop to move to `scheduler.go`** (lines 181-246) — the loop shape stays; timers
become `<-chan time.Time` armed through `cfg.After`:

```go
func (e *Engine) Run(ctx context.Context) error {
	var debounce, deadline *time.Timer
	for {
		var fired bool
		select {
		case <-ctx.Done():
			stopTimer(debounce)
			stopTimer(deadline)
			e.sweepClosed()
			close(e.runDone)
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
			_ = e.runDump()
		}
		debounce, deadline = e.rearm(debounce, deadline)
	}
}
```

```go
func (e *Engine) rearm(debounce, deadline *time.Timer) (nextDebounce, nextDeadline *time.Timer) {
	e.mu.Lock()
	dirty, oldest := e.dirty, e.oldest
	e.mu.Unlock()
	if !dirty {
		stopTimer(debounce)
		stopTimer(deadline)
		return nil, nil
	}
	if debounce == nil {
		debounce = time.NewTimer(e.cfg.Debounce)
	} else {
		debounce.Reset(e.cfg.Debounce)
	}
	if deadline == nil {
		deadline = time.NewTimer(time.Until(oldest.Add(e.cfg.MaxWait)))
	}
	return debounce, deadline
}
```

Pitfall 17: with the channel seam, a non-nil old channel must be replaced and become
unreachable; never call `cfg.After` inside the `select`.

**Dump cycle to move to `scheduler.go`** (lines 278-335) — the capture-before-snapshot
proof is load-bearing:

```go
func (e *Engine) runDump() error {
	e.mu.Lock()
	captured := e.waiters
	e.waiters = nil
	e.dirty = false
	e.mu.Unlock()

	start := time.Now()
	block, err := Encode(e.store.Records())
	if err == nil {
		err = e.dumps.Save(block)
	}
	e.recordDump(time.Since(start), err)
	if err != nil {
		e.failWaiters(captured)
		return err
	}
	e.confirmWaiters(captured)
	return nil
}
```

New shape: `records := e.store.Records()` once, then
`e.dumps.SaveStream(func(w io.Writer) error { return EncodeTo(w, records, e.valueMaxBytes) })`;
measure with `cfg.Now()`; on failure classify, invoke `cfg.OnDumpFailure` once, advance
the backoff ladder.

**Failure fan-out to extend** (lines 359-374) — `failWaiters` re-dirties from a fresh
epoch; the new version claims waiters and uses `cfg.Now()`:

```go
func (e *Engine) failWaiters(captured []*waiter) {
	e.mu.Lock()
	stranded := make([]*waiter, 0, len(captured)+len(e.waiters))
	stranded = append(stranded, captured...)
	stranded = append(stranded, e.waiters...)
	e.waiters = nil
	e.dirty = true
	e.oldest = time.Now()
	for _, w := range stranded {
		w.err = ErrDumpFailed
	}
	e.mu.Unlock()
	for _, w := range stranded {
		close(w.done)
	}
}
```

**Confirmation to move to `confirm.go`** (lines 428-455) — the register-under-mutex
shape stays; the timeout select is new:

```go
func (e *Engine) await() error {
	w := &waiter{done: make(chan struct{})}
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return ErrClosed
	}
	if !e.dirty {
		e.oldest = time.Now()
	}
	e.dirty = true
	e.waiters = append(e.waiters, w)
	e.mu.Unlock()
	select {
	case e.changed <- struct{}{}:
	default:
	}
	<-w.done
	return w.err
}
```

Single-claim protocol: `w.claimed` under `e.mu`; whoever claims first writes `w.err` and
closes; the timeout branch re-checks `claimed` and removes the waiter with
`slices.DeleteFunc`.

**Dirty nudge to centralize in `markDirty`** (lines 781-792) — then delete the duplicate
send at `engine.go:497-500`:

```go
func (e *Engine) markDirty() {
	e.mu.Lock()
	if !e.dirty {
		e.oldest = time.Now()
	}
	e.dirty = true
	e.mu.Unlock()
}
```

**ApplyRemote to move to `facade.go` and switch to `MergeChanged`** (lines 457-502):

```go
func (e *Engine) ApplyRemote(key string, rec storage.Record) ([]storage.Conflict, time.Duration, error) {
	_, getErr := e.store.Get(key)
	conflicts, drift, err := e.store.Merge(key, rec)
	if err != nil {
		return conflicts, drift, err
	}
	displaced := len(conflicts) > 0 && conflicts[0].Winner == rec.Version
	if !displaced && getErr == nil {
		return conflicts, drift, nil
	}
	e.markDirty()
	select {
	case e.changed <- struct{}{}:
	default:
	}
	return conflicts, drift, nil
}
```

Replace the `displaced` heuristic with `MergeChanged`'s `changed` (Pitfall 5); drop the
probe once `changed` is authoritative.

**Flush to bound** (lines 755-779) — both waits are unbounded today:

```go
func (e *Engine) flush() error {
	reply := make(chan error, 1)
	e.mu.Lock()
	closed := e.closed
	e.mu.Unlock()
	if closed {
		return ErrClosed
	}
	select {
	case e.flushCh <- reply:
	case <-e.runDone:
		return ErrClosed
	}
	select {
	case err := <-reply:
		return err
	case <-e.runDone:
		select {
		case err := <-reply:
			return err
		default:
			return ErrClosed
		}
	}
}
```

`FlushTimeout(d)` wraps an `e.cfg.After(d)` case around both waits and returns
`ErrUnconfirmed` on expiry. `Flush()` delegates with `ConfirmTimeout`.

**Snapshot/Restore to move to `snapshot.go`** (lines 571-725) — keep the flush-first,
partial-restore-return contracts byte for byte; `flush()` now can return
`ErrUnconfirmed` and `Restore` must still return the partial list.

**Testing analogs:**
- `engine_test.go:248-374` — `fakeDumpPort` (records saves, `failSaves`, `gate`, snapshots)
  is the fake all synctest tests build on; extend `Save` to `SaveStream` and `Load` to
  `LoadResult`.
- `engine_test.go:376-391` — `startDumper` stop-func pattern; every new synctest test
  must defer it inside the bubble.
- `engine_test.go:492-563` — `TestEngine_dumpFailureFansOut` is the backoff test's base.
- `engine_test.go:1679-1735` — `TestEngine_configShrunkValueLimits` is the one
  non-frozen assertion that legitimately changes (`discarded` becomes 1).
- `engine_blocking_test.go:14-126` — the blocking proof; its 200 ms/1 s windows must
  stay below the 10 s `ConfirmTimeout`.
- `engine_replay_internal_test.go:1-93` — the `_internal_test.go` white-box pattern
  (`testpackage` exemption) for testing unexported helpers such as `markDirty`/`await`.

---

### `internal/app/persist/loader.go`, `facade.go`, `scheduler.go`, `confirm.go`, `snapshot.go` (new internal roles)

**Analog:** the corresponding sections of `engine.go` listed above. These are pure moves;
land them as one behaviour-preserving task after the behavioural waves (research D-10).

Rules observed from the existing code:
- No new interfaces (`iface` lint); plain structs/values.
- Keep each function's comment with the moved code (the comments are the contract).
- `Engine` remains the exported composition, so every existing test compiles unchanged.

---

### `internal/app/persist/errors.go` + `doc.go` (utility/doc)

**Analog:** `errors.go` itself (lines 5-18) and `storage/errors.go`:

```go
var (
	ErrDumpFailed = errors.New("dump failed; write not confirmed")
	ErrClosed = errors.New("persist engine closed")
)
```

Add:

```go
// ErrUnconfirmed is returned to a write whose covering dump did not complete
// within ConfirmTimeout: the record is installed in memory and will be covered
// by a later successful dump, but is not yet durable.
ErrUnconfirmed = errors.New("write not confirmed before the confirmation timeout")
```

`doc.go` pattern: each package contract is a prose section (lines 8-42); update the
timing guarantee and add the load-outcome/confirm-timeout paragraphs.

---

### `internal/port/dump.go` (port/interface, request-response)

**Analog:** itself — full current file is lines 1-40. The doc-comment discipline is the
pattern: every method states the contract, the concurrency rule, and the consumer/
implementer split. Keep the `//nolint:iface` justification comment (line 15) because the
seam is still consumed cross-package.

**Enum pattern to copy:** `internal/port/repl.go:8-34` (`DecodeStatus`):

```go
type DecodeStatus int8

const (
	Decoded DecodeStatus = iota
	BadCRC
	Undecodable
	ProtoVersionMismatch
)
```

Apply the same shape to `LoadOutcome` (`LoadPrompt`/`LoadMain`/`LoadBackup`/
`LoadUnusable`), `Generation` (`GenMain`/`GenPrev`) and `FailureClass`
(`FailureOther`/`FailureDiskFull`/`FailurePermission`), plus `String()` and
`FailureClassOf(err) (errors.As)`.

**New interface shape** (research D-02/D-06):

```go
type DumpStore interface {
	SaveStream(fill func(io.Writer) error) error
	Load(now time.Time, parse func([]byte) error) LoadResult
	CreateSnapshot(name string) error
	ReadSnapshot(name string) ([]byte, error)
}
```

`Save([]byte)` may stay on `FileStore` as a convenience but leaves the interface.
`internal/port` may import `time` and `io` — both stdlib, allowed by
`sup-kv-port-stdlib-only` (`.golangci.yml:81-88`).

---

### `internal/persistfile/dumpfile.go` (adapter, file-I/O)

**Analog:** itself.

**Package doc** (lines 1-11) — update the concurrency paragraph to name the sweep and
preservation rules.

**Constants and FileStore** (lines 21-59) — `tempPattern = "dump-*"` is what the sweep
must match; `prevSuffix` is what preservation must never touch:

```go
const (
	prevSuffix  = ".prev"
	tempPattern = "dump-*"
	dirMode     = 0o700
)

type FileStore struct {
	path string
	ops  fsops
	mu   sync.Mutex
}
```

**Save** (lines 61-74) — becomes `SaveStream`; `Save` delegates:

```go
func (f *FileStore) Save(dump []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return saveAtomically(f.ops, f.path, f.path+prevSuffix, dump)
}
```

**Current Load to replace** (lines 76-94) — every fault is swallowed today:

```go
func (f *FileStore) Load(parse func([]byte) error) (dump []byte, ok bool) {
	for _, candidate := range []string{f.path, f.path + prevSuffix} {
		data, err := os.ReadFile(candidate) //nolint:gosec // ...
		if err != nil {
			continue
		}
		if parse(data) != nil {
			continue
		}
		return data, true
	}
	return nil, false
}
```

New `Load(now, parse) LoadResult`: sweep first, then for each candidate distinguish
`fs.ErrNotExist` (absent, not a failure) from other read errors and oracle failures,
preserve every failed candidate as `<path>.corrupt-<UTC>[-N]`, and classify the outcome.
`now` is the engine's startup clock; the adapter stays clock-free.

**Preservation collision probe** — no in-repo precedent; use the research sketch
(`05-RESEARCH.md` "Preservation at load", lines ~1624-1650): `os.Lstat` the candidate,
require `Mode().IsRegular()`, probe `<base>`, `<base>-1`, … with `os.Lstat`, then
`os.Rename`. Cite `os.Rename`'s replace semantics ("Don't Hand-Roll" table) in the
comment.

**Rotation-survival mechanism** — `fileFSOps.rotate` already guards on the main path
existing (`fsops.go:122-132`), so once preservation has renamed the corrupt main away,
the first post-backup dump's rotate is a no-op and `.prev` survives. The new test must
assert `.prev` bytes are unchanged across the first dump.

**`saveAtomically`** (lines 96-128) — the five-step order must not change; add an
`ensureDir` pre-step that syncs the parent only when the directory was created:

```go
func saveAtomically(ops fsops, path, prevPath string, data []byte) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, dirMode)
	if err != nil {
		return err
	}
	err = ops.write(dir, data) // step 1
	if err != nil {
		return err
	}
	err = ops.fsync() // step 2
	if err != nil {
		return err
	}
	err = ops.rotate(path, prevPath) // step 3
	if err != nil {
		return err
	}
	err = ops.rename(path) // step 4
	if err != nil {
		return err
	}
	return ops.syncDir(dir) // step 5
}
```

**Testing analog:** `dumpfile_test.go:93-154` (`TestFileStore_fallbackChain`) is the table
to extend with outcomes/preservation; `dumpfile_test.go:156-185` has the root-skip guard
for the EACCES class:

```go
if os.Geteuid() == 0 {
	t.Skip("root ignores directory modes - the read-only residue cannot be produced")
}
```

---

### `internal/persistfile/fsops.go` (adapter seam, file-I/O)

**Analog:** itself.

**Seam doc** (lines 5-26) — keep it; it is the authoritative statement that
`os.File.Sync` already handles EINTR and darwin ENOTSUP, and that
`fsync_darwin.go` is not needed. The phase adds directory-fsync tolerance, not a platform
file.

**Interface** (lines 27-63) — change only step 1:

```go
type fsops interface {
	write(dir string, fill func(io.Writer) error) error
	fsync() error
	rotate(main, prev string) error
	rename(main string) error
	syncDir(dir string) error
}
```

**write implementation pattern** (lines 81-97) — temp creation/naming and hygiene stay:

```go
func (o *fileFSOps) write(dir string, data []byte) error {
	tmp, err := os.CreateTemp(dir, tempPattern)
	if err != nil {
		return err
	}
	o.tmp, o.tmpName = tmp, tmp.Name()
	_, err = tmp.Write(data)
	if err != nil {
		o.closeRemoveTemp()
		return err
	}
	return nil
}
```

New: `bufio.NewWriterSize(tmp, 64*1024)`, call `fill(bufw)`, on fill/flush error
`closeRemoveTemp` and return the fill error (Pitfall 23).

**Hygiene helpers** (lines 162-189) — reuse, do not duplicate:

```go
func (o *fileFSOps) closeRemoveTemp() {
	if o.tmp != nil {
		_ = o.tmp.Close()
		o.tmp = nil
	}
	o.removeTemp()
}

func (o *fileFSOps) removeTemp() {
	if o.tmpName != "" {
		_ = os.Remove(o.tmpName)
		o.tmpName = ""
	}
}
```

**`syncDir` + new tolerance helper** (lines 148-160):

```go
func (*fileFSOps) syncDir(dir string) error {
	d, err := os.Open(dir) //nolint:gosec // G304: ...
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	return d.Sync()
}
```

`syncDirTolerant(dir)` wraps `syncDir` and returns nil only for
`errors.Is(err, syscall.ENOTSUP)` / `syscall.EINVAL`; the file-data path keeps every
error. Put the helper in `saveerror.go` or `fsops.go`; it imports `syscall` legally
because this is the adapter.

**Testing analog:** `fsops_test.go:42-96` (recording fake delegating to the real ops) and
`:98-148` (failure-matrix fake reusing the production hygiene helpers); the order test
`:187-219` asserts one ordered slice and the argument paths. The new parent-dir sync is
observable through the same recording fake: `syncDir(parent)` must precede `write` only
when the directory was created.

---

### `internal/persistfile/saveerror.go` (new; adapter error classification)

**Analog:** partial. The nearest in-repo error-classification pattern is
`snapshot.go:68-76`:

```go
func readMainDump(path string) ([]byte, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: ...
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNoDump
	}
	return data, err
}
```

Use the same `errors.Is` discipline, but classify into `port.FailureClass`:

```go
switch {
case errors.Is(err, syscall.ENOSPC):
	class = port.FailureDiskFull
case errors.Is(err, syscall.EACCES), errors.Is(err, fs.ErrPermission):
	class = port.FailurePermission
}
```

Return a `port.ClassedError` (`error` + `FailureClass()`); the engine calls
`port.FailureClassOf`. No `syscall` may appear in `internal/app`.

**Testing analog:** `fsops_test.go`'s table style + synthetic errnos; no real ENOSPC
needed (wrapped `*os.PathError`/`syscall.Errno` values suffice).

---

### `internal/persistfile/snapshot.go` (adapter, file-I/O)

**Analog:** itself.

- `CreateSnapshot` (lines 30-55): keep validate-first, mutex, `MkdirAll` + snapshot-dir
  parent sync, `readMainDump`, `saveAtomically(..., "", data)`. The empty `prevPath`
  skips rotation — the snapshot policy.
- `ReadSnapshot` (lines 57-66): unchanged.
- `readMainDump` (lines 68-76): unchanged; it is also the `fs.ErrNotExist` analog above.
- `validateName` (lines 92-106): unchanged security allow-list.

---

### `cmd/sup-kv/main.go` (config/wiring, startup + batch)

**Analog:** itself.

**Sentinel and exit-code pattern** (lines 56-76):

```go
var (
	errDatagramCeiling   = errors.New("value-max-bytes exceeds the datagram ceiling")
	// ...
)

const (
	exitDumpFailed = 1
	exitUsage      = 2
)
```

Add `errValueMaxBytesRange = errors.New("value-max-bytes must be positive")`.

**Defaults** (lines 134-167) — delete `defaultValueMaxBytes` (line 145); use
`storage.ValueMaxBytes` in the flag registration (lines 357-358) and in
`checkDatagramCeiling`:

```go
flag.IntVar(&cfg.valueMaxBytes, "value-max-bytes", envNum(envValueMaxBytes, storage.ValueMaxBytes),
	"maximum value size in bytes (env SUPKV_VALUE_MAX_BYTES)")
```

**Validation pattern** (lines 427-533) — add `checkValueMaxBytes` and call it from
`finalizeConfig` before `checkDatagramCeiling`:

```go
func finalizeConfig(cfg *config, nodeID uint64, envErr error) error {
	// ...
	err = checkStartupGates(*cfg)
	if err != nil {
		return err
	}
	err = checkDatagramCeiling(*cfg)
	if err != nil {
		return err
	}
	return checkSyncChunkBytes(*cfg)
}
```

Mirror `checkStartupGates` (lines 492-500) for the error text and the `%w` sentinel; the
`main` caller already maps any `finalizeConfig` error to `exitUsage` (lines 233-238).

**Engine wiring** (lines 547-564) — replace `defaultValueMaxBytes`, read `LoadStats`, log
one WARN/ERROR per load outcome, wire `OnDumpFailure`, keep `Open`'s return shape:

```go
func startEngine(cfg config, hub *grpcapi.WatchHub) (*persist.Engine, context.CancelFunc, <-chan error) {
	dumps := persistfile.New(cfg.dumpPath)
	engine, discarded := persist.Open(cfg.nodeID, time.Now, cfg.driftWarn, dumps, storage.Config{
		ValueMaxBytes: cfg.valueMaxBytes,
		OnChange:      hub.Ingest,
	}, persist.Config{Debounce: cfg.dumpDebounce, MaxWait: cfg.dumpMaxWait})
	if discarded > 0 {
		log.Printf("sup-kv: discarded %d undecodable dumped records at startup", discarded)
	}
	// ...
}
```

Log policy (research D-02): `Prompt`/`Main` silent; `Backup` exactly one
`slog.Warn("Dump generation fallback", "main_path", …, "cause", …)`; `Unusable` exactly
one `slog.Error("No usable dump generation", …)`; `Swept > 0` INFO; `SweepErr` WARN.
Static message + attributes (sloglint), never dynamic text as the message.

**Metrics sources** (lines 673-689) — add the new dump/load accessors as method values:

```go
func metricsSources(adapter *udprepl.Adapter, engine *persist.Engine) metrics.Sources {
	return metrics.Sources{
		// ...
		DumpStats: engine.DumpStats,
		// ...
	}
}
```

**Shutdown** (lines 777-839) — replace the unbounded `Flush` + `<-engineDone` pair
(lines 828-837) with `FlushTimeout(shutdownGrace)` and a bounded `select` on
`engineDone`/`time.After(shutdownGrace)`; abandon the goroutine with a WARN on expiry.

**Testing analog:** `main_test.go:509-577` — `TestMain_valueMaxDatagram` is the exact
template for `TestMain_valueMaxNonPositive` (run the binary to completion, assert exit 2
and the message naming the flag). `runToCompletion`/`startLoggedNode`/`assertExitCode`
are the helpers.

---

### `internal/grpcapi/server.go` (controller, request-response)

**Analog:** itself.

**Set failure path** (lines 211-221) — extend the totality branch:

```go
if err != nil {
	if errors.Is(err, persist.ErrDumpFailed) {
		s.notifyLocalWrite(req.GetKey(), ver, req.GetValue(), false, false)
	}
	return nil, mapError(err)
}
```

New condition: `errors.Is(err, persist.ErrDumpFailed) || errors.Is(err, persist.ErrUnconfirmed)`.
Same at Delete (lines 259-266). Comment update: the unconfirmed record is installed and
must replicate (PROTO-08 totality; research D-07).

**mapError** (lines 414-448) — add one case, keep the doc table current:

```go
case errors.Is(err, persist.ErrUnconfirmed):
	return status.Error(codes.Unavailable, persist.ErrUnconfirmed.Error())
```

**Snapshot handlers** — `SnapshotCreate` (lines 305-323) and `SnapshotLoad`
(lines 326-357) already map errors; a `Snapshot` returning `ErrUnconfirmed` flows through
`mapError` unchanged. No new handler-level value check (research Open Question 2); the
store is the single enforcement point, and `ErrValueTooLarge` already maps at line 435.

**Testing analog:** `server_test.go:61-109` (`fakeDumpStore`) must implement
`SaveStream`/`Load(now, parse)`; its snapshot comment (lines 56-59) says Phase 5 would
stage snapshots. The `notifyCapture` pattern (lines 111-124) is how the totality tests
assert the notify call.

---

### `internal/app/repl/core.go` (service/domain, event-driven)

**Analog:** itself.

**Config/normalization** (lines 252-344) — the model for `persist.Config.Now/After`
normalization, including nil-callback → no-op:

```go
func (c Config) normalized() Config {
	// ...
	if c.MaxClockLead <= 0 {
		c.MaxClockLead = defaultMaxClockLead
	}
	if c.Log == nil {
		c.Log = func(_ string) {}
	}
	if c.OnDiscard == nil {
		c.OnDiscard = func(_, _ string) {}
	}
	return c
}
```

**The equal-version short-circuit to replace** (lines 1476-1514):

```go
func (c *Core) applyRemote(now time.Time, source string, msg port.DataReplication, out *port.Outbox) {
	if held, present := c.store.Version(msg.Key); present {
		switch cmp := held.Compare(msg.Record.Version); {
		case cmp > 0:
			// The sender is behind: answer with the held record, never Ack.
			c.enqueueResponse(now, source, egressDataReplication, msg.Key)
			return
		case cmp == 0:
			// Already held at this exact version: Ack and emit no reply.
			out.Unicast(source, port.Ack{Key: msg.Key, Version: held})
			return
		}
	}
	conflicts, drift, err := c.store.ApplyRemote(msg.Key, msg.Record)
	// ...
}
```

New `cmp == 0` branch: `Lookup` the held record; `storage.RecordsEqual(held, msg.Record)`
→ keep the Ack; otherwise fall through to `ApplyRemote`, then decide the reply from the
stored record: incoming won → Ack; stored won → `enqueueResponse(..., egressDataReplication, msg.Key)`.

**Time-as-argument seam** — every handler takes `now time.Time`
(`HandleDatagram` line 1007, `Advance` line 2244); the adapter supplies `time.Now()`
(`internal/udprepl/repl.go:558,571,580,582,587`). This is the pattern `persist.Config.Now`
generalizes for a goroutine-owning loop.

**Testing analog:** `internal/app/repl/sim_scenario_test.go:19-107` — the seeded
scenario/quiescence/replay skeleton; core-level equal-version tests follow its
table-driven shape, and `sim_chaos_test.go` supplies `chaosQuiescent`/model-store helpers.

---

### `internal/app/internal/simnet/simnet.go` (test harness, event-driven)

**Analog:** itself.

**Scheduler/Policy** (lines 86-182) — `Scheduler.Schedule`/`RunUntil`/`Now` is the
virtual-time source; `Policy.AnnounceDelay` (lines 160-168) already models the
dump-before-broadcast delay. The new engine test injects:

```go
func virtualAfter(sched *simnet.Scheduler) func(time.Duration) <-chan time.Time {
	return func(d time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		sched.Schedule(sched.Now().Add(d), func() { ch <- sched.Now() })
		return ch
	}
}
```

**Additive `NotifyInstalled` seam** — copy `Cluster.Write` (lines 739-757) minus the
`store.Set`:

```go
func (c *Cluster) Write(i int, key string, data []byte) storage.Record {
	c.checkIndex(i, "Write")
	now := c.sched.Now()
	_, _, err := c.nodes[i].store.Set(key, data)
	if err != nil {
		panic(fmt.Sprintf("simnet: Cluster.Write: Set %q on node %d: %v", key, i, err))
	}
	rec, ok := c.nodes[i].store.Lookup(key)
	if !ok {
		panic(fmt.Sprintf("simnet: Cluster.Write: Lookup %q on node %d after a successful Set: record missing", key, i))
	}
	var out port.Outbox
	c.nodes[i].core.NotifyLocalWrite(now, key, rec, &out)
	c.route(i, out.Drain())
	out.DrainResults()
	c.drive(i, now)
	c.writes = append(c.writes, LoggedWrite{Key: key, Rec: rec})
	return rec
}
```

`NotifyInstalled(i, key)` is the same body without `Set`: `Lookup`, `NotifyLocalWrite`,
`route`, `DrainResults`, `drive`, and the write-log append. Use
`Cluster.Store(i)` (line 801) to share the store with `persist.New`.

---

### `internal/metrics/metrics.go` (adapter/exporter, request-response)

**Analog:** itself.

**Sources** (lines 176-197) — add the new accessors as fields:

```go
type Sources struct {
	Discards      func() repl.DiscardStats
	KnownPeers    func() uint64
	Unacked       func() repl.UnackedStats
	Sent          func() udprepl.SentByType
	Retransmitted func() uint64
	MergeStats    func() repl.MergeStats
	DumpStats     func() persist.DumpStats
	// ...
}
```

The `DumpStats` value gains `DiskFull`, `Permission`, `Unconfirmed`; add
`LoadStats`/`RecordsDiscarded` sources for the load-outcome and discarded-class families.

**dumpFamilies** (lines 343-369) — the exact family-construction shape:

```go
func dumpFamilies(stats persist.DumpStats) []Family {
	return []Family{
		{
			Name: "supkv_dumps_duration_seconds", Type: Gauge,
			Help: "Duration of the last dump's encode and save in seconds.",
			Samples: []Sample{{Value: stats.DurationSeconds}},
		},
		// ...
	}
}
```

New labeled families use `[]Label{{Name: "class", Value: "..."}}` with every enum value
always rendered (the discard-family pattern, metrics.go:306-341).

**Testing analog:** `metrics_test.go:196-283` — the `families` table and `zeroSources`
must be updated in the same task; `TestMetrics_families` counts each TYPE line exactly
once and asserts every zero sample. Pitfall 21: a nil new source panics the zero-state
test.

---

### `docs/adr/0012-equal-version-tiebreak.md` (new documentation)

**Analog:** `docs/adr/0011-value-limit-60kib-kept.md` (closest: a PERSIST requirement and
a single decision) and `docs/adr/0002-create-if-absent-blocked-by-tombstone.md`
(deviation shape).

**Frontmatter pattern** (0011:1-7):

```markdown
---
adr: 0011
review-decision: "14.11 (The ~60 KiB single-value limit stays)"
informs: [PERSIST-01]
date: 2026-11-19
status: accepted
---
```

New ADR: `adr: 0012`, `review-decision: "P-07 (equal-version tie-break)"`,
`informs: [PERSIST-07]`, `date: 2026-11-22`, `status: accepted`.

**Body sections** — exactly `## Context`, `## Decision`, `## Consequences`, `## Links`
(see 0011:11-43). The research D-12 outline supplies the content: Spec random tail
replaced by zero filler (S-HLC-07), same-version-different-content reachable after disk
loss + clock rollback, the `(version, content)` law (tombstone wins; else
`bytes.Compare`), where it applies (store join + core equal branch), what is not changed,
and the lost-operation consequence.

---

### `docs/OPERATIONS.md` (modification)

**Analog:** itself. The sections to extend:

- `## Record and datagram limits` (lines 62-85) — note that the value limit is now the one
  `storage.ValueMaxBytes` / configured value and that non-positive exits 2; keep the
  64 408 boundary.
- `## Monitoring` (lines 141-171) — add the new family names and the label catalogs
  (`supkv_dumps_failures_total{class}`, `supkv_dumps_unconfirmed_total`,
  `supkv_dump_load_total{outcome}`, `supkv_dump_records_discarded_total{class}`); keep the
  "every class rendered at zero" rule.
- `## Lifecycle and exit codes` (lines 173-189) — document the retry ladder, "health
  stays SERVING under a dump failure", the confirmation timeout (`Unavailable` on writes),
  and bounded shutdown/abandoned goroutine.
- Add a `## Load outcomes` section: Prompt/Main/Backup/Unusable, one WARN/ERROR per class,
  `.corrupt-<UTC>` files are operator-inspected and never auto-deleted, sweep age bound.
- Add a memory-model paragraph: one `Records()` copy + one line buffer + 64 KiB bufio;
  the residual writer pause while `Records()` clones under the store read lock.

Pattern: static headings, bullet lists, exact family names in backticks, no Spec quotes.

---

### `docs/SPEC-DEVIATIONS.md` (modification)

**Analog:** itself.

- Preamble line 25: change "ADR numbers 0001–0011 … 14.1–14.11" to include 0012.
- Row updates (research Open Question 13):
  - `S-PAR-05` (line 230): `fixed-in-v1.1(PERSIST-01)` → `implemented`, links to
    `storage.ValueMaxBytes` + codec.
  - `S-PERS-08` (line 164): `implemented`, links updated to the new `LoadResult` path.
  - `S-PERS-09` (line 165): `fixed-in-v1.1(OBSV-06)` → `implemented`, keep the OBSV-06
    link for the rest of lifecycle logging.
  - `S-ALG-REP-03` (line 125): summary amended for the deterministic tie-break; link
    ADR 0012.
  - `S-HLC-07` (line 53): keep `accepted-deviation`, add the ADR 0012 link.
  - `S-DM-04`/`S-DM-05`/`S-DM-08` (lines 36-40): update the code links for the new
    constant/merge method.

The table format is fixed: `| ID | Summary (own words) | Status | Links |`; status
vocabulary at lines 10-20 is a closed enum.

---

### `scripts/red-expectations.tsv` (modification)

**Analog:** itself. Rows 29-30 are the only edits:

```tsv
TestRed_P01_valueLimit	red
TestRed_P01_valueLimitEngine	red
```

Flip both to `green` only after `SUPKV_RED_TESTS=1` passes for both. Do not touch the
header, any other row, or `scripts/red-runner` (its header at lines 1-40 states the flip
"IS this one-line change"). The runner's package list already includes
`./internal/stand/` and `./internal/persistfile/` (red-runner:70-76).

---

## Test Pattern Assignments

| New/Modified Test | Analog | Patterns to copy |
|-------------------|--------|------------------|
| `internal/app/persist/load_stats_test.go` | `engine_test.go:248-391`, `open_seed_test.go:22-110` | fake port construction, `startDumper`, keyed `persist.Config`, real `persistfile.New(t.TempDir())` for composition |
| `internal/app/persist/timeout_test.go` | `engine_blocking_test.go:14-126` | synctest bubble, parking fake, `waitParked`, release channel, assert at virtual instants; **never construct `persistfile` inside a bubble** (Pitfall 18) |
| `internal/app/persist/backoff_test.go` | `engine_test.go:492-563` | `failSaves` counter, `saveAttempts()`/`save(i).at`, synctest virtual instants, assert exactly N attempts |
| `internal/app/repl/retransmit_test.go` | `retransmit_test.go:1346-1373` | explicit `DataReplication` equal-version table and same-seed Ack/reply trace; no convergence claim (that proof is `internal/app/repl/sim_dump_test.go` in plan 05-07) |
| `internal/app/persist/encode_stream_test.go` | `codec_test.go:19-116`, `bench_test.go:61-97` | property steps, `sealed`/`baseRecords` fixtures, a recording `io.Writer` for max-single-write |
| `internal/persistfile/load_outcome_test.go` | `dumpfile_test.go:93-185` | temp-dir table, `writeFile`, marker oracle, root-skip for EACCES, fixed `now` for `.corrupt-<UTC>` names |
| `internal/persistfile/sweep_test.go` | `dumpfile_test.go`, `fsops_test.go:172-185` | `fsopsEntries` sorted base-name oracle, `os.Chtimes` for the age bound, both directories |
| `internal/persistfile/saveerror_test.go` | `fsops_test.go:23-148` | sentinel injected error + `errors.Is`, table over synthetic wrapped errnos |
| `internal/persistfile/dumpfile_test.go` (mod) | itself | update `Load` call sites to `Load(now, parse)` and assert `LoadResult` |
| `internal/persistfile/fsops_test.go` (mod) | itself `:42-219` | update `write` fake to the fill callback; keep the exact `fsopsOrder` slice; add the parent-sync row |
| `internal/app/repl/sim_dump_test.go` | `sim_scenario_test.go:19-107`, `simnet_ext_test.go:380-490` | seeded rng before events, `scnQuiesce`, `virtualAfter`, `Policy.AnnounceDelay`, same-seed replay comparison |
| `internal/app/repl` equal-version tests | `sim_scenario_test.go` + core table tests | two records sharing a version, incoming-wins Ack vs stored-wins unicast |
| `internal/stand/persist_test.go` | `stand_red_test.go:884-962`, `stand_metrics_test.go:1-50` | `newStandPair`, `startStandNode(..., extra flags)`, `awaitServing`, gRPC client helpers, `readFinalDump`, SIGKILL/restart, metrics scrape over HTTP, root-skip |
| `cmd/sup-kv/main_test.go` (mod) | `TestMain_valueMaxDatagram:509-577`, `TestMain_startupGateEmptyRange:579-...` | `smokeBinary`, `runToCompletion`, `assertExitCode`, assert the flag name in stderr |
| `internal/grpcapi/server_test.go` (mod) | `fakeDumpStore:61-124` | `SaveStream` capture, `Load` returning `LoadResult`, `notifyCapture` assertions |
| `internal/metrics/metrics_test.go` (mod) | `families` table `:196-211`, `zeroSources:212-234` | add every new family + zero sample line in the same task |
| `internal/app/storage/store_test.go` (mod) | `:510-530` | rewrite the equal-version subtest to the content law |
| `internal/app/storage/merge_test.go` (mod) | `:222-261` | rewrite the `cmpSame` expectation; assert equal-version conflicts are allowed and name the content winner |
| `internal/app/persist/engine_test.go` (mod) | itself | extend `fakeDumpPort` for `SaveStream`/`LoadResult`; change `configShrunkValueLimits` to `discarded == 1` |
| `internal/app/persist/engine_blocking_test.go` (mod) | itself | update the fake to `SaveStream`; keep the 200 ms/1 s windows |
| `internal/app/persist/codec_test.go` (mod) | itself | update `Encode`/`Decode` call sites with the limit; rebuild the duplicate-key fixture with a correct count |
| `internal/app/persist/fuzz_dump_test.go` (mod) | itself | update the `Decode` signature; add census/overlong/injected-newline corpus seeds; assert `len(records)+stats.Total() <= L` |
| `internal/app/persist/bench_test.go` (mod) | itself | `EncodeTo`/`SaveStream` path; report `B/op` + a sampled peak |

---

## Shared Patterns

### 1. Depguard-safe layering (adapter classifies, core reports, binary narrates)

**Source:** `.golangci.yml:68-88`; precedent `repl.Config.OnDiscard` /
`persistfile` `fsops` doc (`internal/persistfile/fsops.go:5-26`).

- `internal/persistfile` is the only package that imports `os`/`syscall`; it classifies
  errnos into `port.FailureClass` and reads file existence.
- `internal/port` carries the stdlib-only vocabulary (`LoadOutcome`, `Generation`,
  `LoadResult`, `FailureClass`, `ClassedError`).
- `internal/app/persist` stores a report (`LoadStats`) and calls
  `port.FailureClassOf`; it never knows a path or an errno.
- `cmd/sup-kv/main.go` owns the one slog line per class and the metric wiring.

Apply to: every new load/failure signal (PERSIST-02/03/05/09/10).

### 2. Sentinel errors + `errors.Is`, never identity

**Source:** `internal/app/storage/errors.go:5-19`,
`internal/app/persist/errors.go:5-18`, `internal/grpcapi/server.go:414-448`.

Add `ErrUnconfirmed` as a sentinel beside `ErrDumpFailed`; map it in `mapError`; match
with `errors.Is` everywhere. Never construct dynamic errors in the core (`err113`).

### 3. One log line per class, static message + attributes

**Source:** `cmd/sup-kv/main.go:669` (`Log: func(msg string) { slog.Warn("Replication adapter", "detail", msg) }`),
`main.go:721` (`slog.Error("Metrics endpoint", "detail", ...)`).

The core is silent; the binary logs. New lines: one WARN for `LoadBackup`, one ERROR for
`LoadUnusable`, one ERROR per failed dump attempt, one WARN for the shrunk-limit skip
count, one WARN for `SweepErr`. Message is static; paths/classes/counts are attributes.

### 4. Counted classes, never strings

**Source:** `internal/app/repl/core.go:406-434` (`DiscardStats` one field per class),
`internal/metrics/metrics.go:306-341` (labeled family with all classes).

`DecodeStats` gets four integer fields (`Checksum`, `Version`, `ValueLimit`, `Malformed`)
and `Total()`; `DumpStats` gains `DiskFull`, `Permission`, `Unconfirmed`. Metrics families
render every enum value at zero.

### 5. Fake-port discipline

**Source:** `internal/app/persist/engine_test.go:257-374`,
`engine_blocking_test.go:35-71`, `internal/grpcapi/server_test.go:61-109`,
`internal/persistfile/fsops_test.go:58-148`.

All four `DumpStore` implementations must change in one task (research Pitfall 6):
`FileStore`, `fakeDumpPort`, `blockingDumpPort`, `fakeDumpStore`. Fakes record bytes and
timestamps, inject failures, and never touch the filesystem inside a synctest bubble.
For `SaveStream`, capture by giving the fill a `bytes.Buffer` writer.

### 6. synctest for core timing; simnet for the dump↔notify interaction

**Source:** `engine_test.go:376-391` and `:492-563`; `engine_blocking_test.go:88-126`;
`sim_scenario_test.go:19-107`.

Unit timing uses `synctest.Test` + `synctest.Sleep`/`Wait` with fake ports. The
cross-component test uses `simnet.Scheduler` + `Config.Now/After`; never mix real
`persistfile` I/O into a bubble.

### 7. Determinism rules

**Source:** `codec.go:56-85` (sorted keys), `simnet.go:1-30` (seeded rng, no wall clock),
`restoreChanges` (explicit tuple comparison).

- Dump byte stability: sorted keys, no timestamps, no map iteration in output.
- Tie-break: `bytes.Compare` over `(Deleted, Data)`; no rng, no map order.
- Preservation names: injected `now`, UTC format.
- Backoff: named constants, no jitter.
- Same-seed sim runs must be byte-identical.

### 8. Frozen test-adjacent edits

**Source:** `internal/persistfile/red_value_limit_test.go:24-46`,
`internal/stand/stand_red_test.go:896-922`.

The two P-01 tests are edited **not at all**. The only file change is the TSV row flip.
Tests that may change: `store_test.go` equal-version subtest, `merge_test.go`
losers property, `engine_test.go` shrunk-limit expectation, codec call sites, fakes.

---

## No Analog Found

| File | Role | Data Flow | Reason / fallback |
|------|------|-----------|-------------------|
| `internal/persistfile/saveerror.go` | adapter/utility | file-I/O | No errno-classification helper exists. Nearest pattern is `snapshot.go:68-76` (`errors.Is(fs.ErrNotExist)` → sentinel). Use RESEARCH D-05's `port.FailureClass` + `ClassedError` design; keep `syscall` in this adapter only. |
| `internal/persistfile` corrupt-generation preservation | adapter | file-I/O | No rename-with-collision-probe code exists. Use RESEARCH D-02's `Lstat` + `-N` probe sketch; cite `os.Rename` replace semantics and the 0700 dir in the comment. |
| `internal/persistfile` orphan-temp sweep | adapter | file-I/O | No directory-walk code exists. Use `tempPattern = "dump-*"` (`dumpfile.go:29`), `os.ReadDir`, `ModTime` age bound; test oracle pattern is `fsopsEntries` (`fsops_test.go:172-185`). |
| `internal/app/persist/backoff_test.go` | test | event-driven | No backoff exists today; the failure fan-out test (`engine_test.go:492-563`) pins "exactly 2 attempts" only. Extend it to assert attempt instants and the cap. |
| `internal/app/repl/sim_dump_test.go` | test | event-driven | Simulator exists, but no `persist.Engine` was ever part of a cluster. Use the scenario skeleton and add `Cluster.NotifyInstalled`; the research D-11 fallback (steps 1-2, 4-5) applies if virtual-time/goroutine interleaving is brittle. |
| `internal/stand/persist_test.go` | test | file-I/O | Stand harness exists, but no persist-outcome/EACCES scenario. Use the P-01 scenario's node lifecycle and metrics-test scrape; skip as root. |

---

## Metadata

**Analog search scope:** `internal/app/{storage,persist,repl}`, `internal/app/internal/{simnet,testutil}`,
`internal/{port,persistfile,grpcapi,udprepl,metrics}`, `cmd/sup-kv`, `docs/adr`,
`docs/{OPERATIONS,SPEC-DEVIATIONS}.md`, `scripts/`.
**Files scanned:** 47 target files + ~40 analog files read this session.
**Pattern extraction date:** 2026-11-22.
**Tracked-source gate:** every analog path verified with `git ls-files -- <path>`;
no gitignored mirror paths cited.

**Highest-risk pattern gaps for the planner:**
1. `persist.Open`'s frozen `(*Engine, int)` return — every loader task must preserve it.
2. The four `DumpStore` implementations changing in one task.
3. `json.Decoder` must disappear from `codec.Decode` (line-oriented decode is the only
   shape that satisfies PERSIST-03).
4. `ErrUnconfirmed` must reach both gRPC totality branches and `mapError` in the same
   task, or a timed-out write silently skips replication.
5. `MergeChanged` + the repl equal-version branch are one law; a store-only change is
   dead code in production.
