# Phase 5: Persistence - Research

**Researched:** 2026-11-22
**Domain:** Go 1.27 stdlib persistence core and file adapter (JSONL dump format, durability, write confirmation, LWW tie-break)
**Confidence:** HIGH on the in-repo current state (every claim read from source this session); MEDIUM on the newly designed contracts (they are recommendations to be pinned by the phase's tests); LOW only where marked `[ASSUMED]`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### The value limit, one source (PERSIST-01 / P-01)
- **D-01:** Exactly one authoritative value limit: `storage.ValueMaxBytes` (default 61 440) is the only place the number is defined. It is passed by configuration into every consumer — `storage.Config.ValueMaxBytes`, the gRPC Set validator, the persistfile codec (encode and decode), and the `--value-max-bytes` / `SUPKV_VALUE_MAX_BYTES` surface. The codec's frozen `maxRecordBytes = 61 440` is removed: a record of exactly the configured limit survives a restart at any accepted flag value. An invalid value (non-positive, or above the datagram ceiling) exits 2 through the existing usage path. The datagram-ceiling relation stays one-way and documented. — [auto] [Value limit] Q: how is the limit made single-source? → Selected: one exported constant, config-passed everywhere, codec reads config (P-01/Q-04). — **Reversibility:** costly — public parameter and dump format contract.

#### Load outcomes and generation integrity (PERSIST-02/03)
- **D-02:** `DumpStore.Load` returns an explicit outcome: `Prompt` (fresh start: files absent — no WARN, no ERROR), `Backup` (`.prev` used — exactly one WARN naming path and cause), `Unusable` (both generations unusable — ERROR and a non-zero metric; the unusable files are preserved under a distinct name, e.g. `.corrupt-<UTC>`, and rotation never overwrites them). The load path never deletes evidence. — [auto] [Load outcomes] Q: how are outcomes visible? → Selected: explicit outcome enum through the port + one WARN / ERROR per class (P-02). — **Reversibility:** expensive — port contract.
- **D-03:** The dump format carries a record count or terminator so a line-boundary truncation is recognized as incomplete (a `Backup`-class outcome with a WARN, not silent acceptance), and one corrupted record discards only that record (record-level checksum already exists; the count/terminator check is file-level). Format-version compatibility with the v1.0 dumps is preserved for reading; a missing trailer is treated as a truncated file. — [auto] [Generation integrity] Q: how is truncation detected? → Selected: record count/terminator in the format (P-03). — **Reversibility:** costly — on-disk format.

#### Generation lifecycle (PERSIST-04/05)
- **D-04:** After a backup (`.prev`) or empty start the engine is marked dirty, so the first rotation never displaces the last good generation with a corrupt one; `SnapshotCreate` right after such a start decodes back to the in-memory state. This includes the Phase-4 review New-1 fix: every `markDirty()` wakes the dumper (`e.changed` nudge), so an idle clean dumper cannot defer installed records. — [auto] [Dirty start] Q: what is the post-backup/empty start state? → Selected: dirty + first write re-establishes the good generation (P-04). — **Reversibility:** reversible.
- **D-05:** Dump failures decay and log: each failure produces exactly one ERROR entry carrying the underlying error (ENOSPC, EACCES), retries with a growing pause up to a cap, and health behavior under a persistent failure is defined and documented. Decision: health stays `SERVING` (data service continues; the failure is a durability risk, not a serving failure), a metric counter per failure class is published, and the runbook text lands in OPERATIONS; the retry ladder is bounded and never busy-loops. — [auto] [Failure decay] Q: what happens under a persistent dump failure? → Selected: bounded backoff + ERROR/metric, health stays SERVING (P-05). — **Reversibility:** reversible.

#### Resources (PERSIST-06/08/09/10)
- **D-06:** A dump cycle allocates at most two state sizes: streaming encode through the existing writer (no full second copy), with the memory model documented (measured, not guessed). — [auto] [Memory] Q: how is the two-state-size bound proven? → Selected: streaming encode + documented model (P-06). — **Reversibility:** reversible.
- **D-07:** Blocking engine calls carry a deadline: a hung fsync cannot block writes, snapshots or shutdown indefinitely. The dump/fsync work stays on the dumper goroutine; the Set confirmation wait gets a bounded timeout and on expiry returns an explicit unconfirmed outcome (class named, metric counted, health documented) instead of blocking; shutdown is bounded and abandons a stuck fsync rather than hanging. — [auto] [Deadlines] Q: how is a hung fsync bounded? → Selected: bounded confirmation wait + bounded shutdown (P-08). — **Reversibility:** expensive — RPC contract.
- **D-08:** Temp-file hygiene: orphan temp files are collected at startup (age-bounded), a newly created directory is synced at its parent, and `fsync_darwin.go` is removed in favour of one fsync path that treats EINTR as retry and ENOTSUP as a documented skip. — [auto] [Temp/darwin] Q: remove or repair the darwin helper? → Selected: remove, one portable fsync path (P-09). — **Reversibility:** reversible.
- **D-09:** Records skipped at start because the configured value limit shrank are counted and logged (one WARN with the count and class), never silently dropped; the previous generation holding them is not overwritten until the normal rotation discipline allows it, and the metric exposes the count. — [auto] [Skipped records] Q: what happens to oversize records on load? → Selected: count + WARN + metric, evidence preserved by rotation (P-10). — **Reversibility:** reversible.

#### Structure and seams (PERSIST-11/12)
- **D-10:** `persist.Engine` splits along its responsibilities: loader/composition root, store facade, dump scheduler, confirmation accounting, snapshot orchestration — internal types with the public API unchanged; `port.DumpStore` returns the load outcome and failure cause (backs PERSIST-02 and the Phase-4 totality seam). — [auto] [Engine split] Q: how far to split? → Selected: five internal roles, public surface stable (Q-03/PERSIST-11). — **Reversibility:** reversible.
- **D-11:** `internal/app/persist` timers and `time.Now()` move behind the same seam the repl core uses (clock + after-func injection), so the simulator exercises the dump↔replication-notify interaction, including a dump failure under load and the skip-count path. — [auto] [Timer seam] Q: same seam as repl? → Selected: yes, shared clock/timer contract (Q-06/PERSIST-12). — **Reversibility:** reversible.

#### Tie-break (PERSIST-07)
- **D-12:** Equal version with different content is resolved by a **deterministic** tie-break (content-based, not the Spec's random tail), recorded in a new ADR (`docs/adr/0012-equal-version-tiebreak.md`), implemented in the merge path and tested. Rationale: deterministic simulation replay and byte-identical dumps are project invariants; a random tail would break both. The ADR records the deviation and its consequence. — [auto] [Tie-break] Q: random tail or deterministic? → Selected: deterministic, ADR-recorded (P-07). — **Reversibility:** costly — data model.

### Claude's Discretion
- Exact record-count/terminator encoding, the corrupt-file naming scheme, backoff constants, the confirmation timeout default, the orphan-temp age bound, and the internal split's type names.

### Deferred Ideas (OUT OF SCOPE)
- `sync_chunk_bytes` bounds and the four new counters' metrics surface are Phase 7's (OBSV/robustness), per the Phase-4 ledger.
- `TestCore_keyCountCostFlat` ratio flake is Phase 8's.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PERSIST-01 | The value limit is defined in one place and enforced (P-01, Q-04) | Current three copies located with quotes; single-source design (`storage.ValueMaxBytes` + `Config.ValueLimit` + `Store.ValueMaxBytes`) in D-01; frozen tests' exact expectations mapped; non-positive exit-2 gap found |
| PERSIST-02 | Dump-load outcomes are visible and never destroy evidence (P-02) | `Load`'s error-swallowing path quoted; `port.LoadResult` design with Prompt/Main/Backup/Unusable; preservation ordering proof that keeps `.prev` alive across the first rotation; logging/metric split |
| PERSIST-03 | Generation integrity is explicit (P-03) | `json.Decoder` poisoning verified by probe; per-line decode + header `record_count` design; exact truncation rules; v1.0 legacy read path; `DecodeStats` class split |
| PERSIST-04 | After a backup or empty start the engine is marked dirty (P-04) | Engine's `dirty` initialization and `markDirty` nudge gap quoted (Phase-4 New-1); dirty-on-Prompt/Backup/Unusable rule; evidence that preservation (not dirty alone) is what saves `.prev` |
| PERSIST-05 | Dump failures decay and log (P-05) | `failWaiters` re-dirtying + debounce re-arm quoted (the 50 ms ladder); backoff design; port-level failure-class enum; `OnDumpFailure`; health is gate-only (verified) so no health code change |
| PERSIST-06 | A dump cycle allocates at most two state sizes (P-06) | `Records()` copy + `Encode` buffer + `Save([]byte)` quoted; `SaveStream(func(io.Writer) error)` + `EncodeTo` design; measurement strategy and the residual lock-hold note |
| PERSIST-07 | Equal version with different content is resolved deliberately (P-07) | `join`'s equal branch and the replication core's equal-version short-circuit quoted; content tie-break law; `MergeChanged` + `RecordsEqual`; ADR 0012 outline; tests that pin the old no-op |
| PERSIST-08 | Blocking engine calls carry a deadline (P-08) | `await`'s unbounded `<-w.done`, `flush`'s two unbounded waits, `shutdown`'s unbounded `Flush` + `<-engineDone` quoted; waiter-claim protocol; `ErrUnconfirmed`; `FlushTimeout`; RPC totality ripple |
| PERSIST-09 | Temp-file hygiene and the darwin fsync helper are fixed (P-09) | `saveAtomically`/`fsops` quoted; `fsync_darwin.go` already gone (verified); GOROOT-verified EINTR/ENOTSUP facts; sweep/parent-sync/ENOTSUP-on-directory design |
| PERSIST-10 | Records skipped at start are accounted (P-10) | Mixed `discarded` counter and `mergeRecords` skip quoted; class-split design; the test that asserts `discarded == 0` and must change; WARN/metric/evidence plan |
| PERSIST-11 | `persist.Engine` is split along its responsibilities (Q-03) | Engine's full method inventory; five-role split with public signatures preserved (required by the frozen P-01 engine test); port contract change backs the load outcome |
| PERSIST-12 | `internal/app/persist` timers and `time.Now()` sit behind the same seam as the repl core (Q-06) | Every `time.*` call in the engine located; repl's `now`-parameter seam and the adapter's `time.Now` quoted; `Config.Now`/`Config.After` design; simnet `Scheduler`/`Network`/`AnnounceDelay` evidence and the additive `NotifyInstalled` seam needed for the sim test |
</phase_requirements>

## Summary

The phase closes twelve findings against a persistence path that is small, well-factored at the file level and under-instrumented at its boundaries. Three facts shape everything below.

First, the value limit exists **three times** (storage, codec, main) and the codec's copy is the silent data-loss bug: the store accepts a value up to the configured limit, the encoder writes it, `Set` confirms it, and the decoder then discards it as oversize. The fix is not just deleting two constants; the effective limit must become a single runtime value that the store owns and the codec reads, because the decoder runs *before* the store exists (its clock is seeded from the decoded records). The design below threads that with one exported constant plus a `Config.ValueLimit()` helper, and keeps `persist.Open`'s signature byte-compatible because a frozen test compiles against it.

Second, the load path currently cannot tell "no files" from "both files corrupt": `FileStore.Load` returns `(dump, ok)` and swallows every read and parse error; `Open` ignores the boolean entirely. The same absence of evidence-preservation is what makes the `.prev` generation disposable: the first dump after a backup start rotates the corrupt main over the good `.prev`. Making the outcome explicit **and moving corrupt generations out of the rotation paths at load time** are one design, not two — without the preservation, marking the engine dirty actually accelerates the loss.

Third, the governance constraints are unusually tight in ways that a naive implementation would break: `persist.Open`'s `(*Engine, int)` signature is compiled by two frozen red tests that may not be edited; the engine's public API is exercised by hundreds of assertions; the codec is a fuzz target with a committed corpus; and the strict lint config (`noinlineerr`, `mnd`, `gochecknoglobals`, depguard) constrains the shape of the new helpers. Every recommendation below is written to keep those contracts stable.

**Primary recommendation:** implement in five dependency-ordered waves — (1) single-source limit + codec format/limit/per-line decoding, (2) port `LoadResult`/`SaveStream` + adapter outcomes/preservation/sweep, (3) engine dirty/backoff/confirmation-timeout/timer-seam + the internal split, (4) the deterministic tie-break including the replication core's equal-version branch, (5) wiring, process-level proofs, docs and the red-row flip. Do not start with the Engine split; the split is a no-behavior refactor best performed once the new contracts exist.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| The value limit as a single runtime value | API/Backend (`internal/app/storage`) | Binary edge (`cmd`, flag/env) | The store is the one component that already rejects oversize writes at every entry point; exposing its effective limit read-only makes disagreement impossible |
| Dump grammar, per-record validation, record census | API/Backend (`internal/app/persist` codec) | — | Pure bytes-in/records-out; no I/O, no platform knowledge (depguard) |
| Value-limit enforcement on decode/encode | API/Backend (codec, parameterized) | — | Same domain as the grammar; the number is *passed*, never defined |
| Atomic generation write, fsync, rotation, rename, dir-sync | Adapter (`internal/persistfile`) | — | Only package allowed to import `os`/`syscall`; the fsops seam already serializes the five steps |
| Load outcome classification and evidence preservation | Adapter (`internal/persistfile`) | API/Backend (`persist.Open` turns it into a report) | Only the adapter knows which files exist/read; only the engine knows what was decoded |
| Generation census outcome → WARN/ERROR/metric | Binary wiring (`cmd`) + metrics adapter | API/Backend (report type) | The core stays silent (the repl `OnDiscard` precedent); log policy is process policy |
| Dirty state, dump scheduling, retry backoff | API/Backend (`internal/app/persist`) | — | Decision logic; no I/O; timers behind the injected seam |
| Write confirmation and its deadline | API/Backend (engine) | API adapter (`grpcapi` mapping/totality) | Confirmation is the core's contract; the RPC layer must map the new outcome honestly |
| Failure classification (ENOSPC/EACCES) | Adapter (errno knowledge) | Port (enum) | `syscall` is forbidden in the core; the port enum is the shared vocabulary |
| Streaming encode | API/Backend (codec + engine) | Adapter (writer source) | The encoder writes to the adapter-supplied writer; neither buffers the dump |
| Deterministic equal-version resolution | Database/Storage (store merge) | Replication core (`applyRemote`) | The store owns the join law; the core owns the wire behavior that must reach that law |
| Orphan-temp sweep, parent-dir sync, fsync tolerance | Adapter | — | Filesystem semantics only |
| Clock and timer seam | API/Backend consumes injected funcs; adapter/test supplies | Simulator harness | Same discipline as the repl core: the core owns no wall clock |

## Project Constraints (from AGENTS.md)

No `CLAUDE.md` and no project skills directories exist in this repo; `AGENTS.md` is the project instruction file. Its actionable directives, which the plan must honour:

| Directive | Consequence for this phase |
|-----------|----------------------------|
| Go version pinned in `go.mod` (`go 1.27.4`); do not raise without an ADR | All designs use Go 1.27 stdlib only; `testing/synctest` is available and already used |
| `make lint` (`go tool golangci-lint run ./...`) and `make test` (`go test -race ./...`) before pushing | Every wave must end lint-clean; no `//nolint` without a named linter and a written reason |
| Depguard `sup-kv-app-no-io`: `internal/app` may not import `net`, `os`, `syscall`, `google.golang.org/*` | Codec/engine stay pure; errno classification, temp sweep, fsync tolerance live in `internal/persistfile`; the failure-class enum must live in `internal/port` (stdlib-only) so the app can consume it |
| Depguard `sup-kv-port-stdlib-only`: `internal/port` may import stdlib + own module only | `port.LoadOutcome`, `port.FailureClass`, `port.LoadResult` and the `DumpStore` shape must be stdlib-only |
| `.golangci.yml` is byte-unmodified powerman strict config for golangci-lint 2.15.0; additions only under `linters.settings.depguard.rules` | No new linter settings; solution shapes must fit `default: all` (notably `noinlineerr`, `mnd`, `gochecknoglobals`, `err113`, `testpackage`, `paralleltest`, `recvcheck`) |
| Runtime graph has zero direct runtime requires; tools via `go tool` | No new module dependencies; this phase adds none |
| `cmd/sup-kv/main.go` — "`main()` holds no testable logic (flags/env/signals live there from Phase 5, nothing else)" | New wiring (load logging, metric sources, bounded shutdown) stays thin; validation predicates live in testable helpers (Phase 7 will consolidate all validation — do not pre-empt OBSV-02) |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go standard library | go1.27.4 (pinned in `go.mod`) `[VERIFIED: go.mod:3]` | Everything this phase touches: `encoding/json`, `bufio`, `io`, `os`, `syscall`, `errors`, `time`, `testing/synctest` | The repo carries zero direct runtime requires by policy (`AGENTS.md`); persistence is deliberately stdlib-only |
| `testing/synctest` | Go 1.27 stdlib `[VERIFIED: go doc testing/synctest.Test]` | Virtual-time engine tests (already used by `engine_blocking_test.go` and the engine suite) | The existing engine suite is built on it; the new confirmation/backoff tests should follow rather than invent a second clock fake |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `internal/app/internal/simnet` | in-repo | Virtual scheduler, network, cluster | PERSIST-12's dump↔notify interaction test |
| `internal/app/internal/testutil` | in-repo | Seeded property generators (`RandRecord`, `RandStore`, `RandKey`) | Tie-break property tests and codec round-trips |
| `scripts/red-runner` + `scripts/red-expectations.tsv` | in-repo | The expected-red gate | Flipping the two P-01 rows after the fix lands |
| `go tool golangci-lint` | v2.15.0 via `go.mod` tool directive `[VERIFIED: go.mod:5-12]` | Lint gate with the vendored strict config | Every wave |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Content tie-break in the store | Spec's random 46-bit tail in the version | Breaks byte-identical dumps and deterministic simulator replay; D-12 rejects it, ADR 0012 records why |
| Header record count | Trailer terminator line | Trailer is detected more simply, but a v1.0 binary downgraded onto a new dump would count the trailer as one discarded record, and the "missing trailer = truncated" rule would classify every v1.0 dump as corrupt at upgrade. See D-03 analysis |
| Header record count, new format version 2 | Keep `format_version: 1` + additive field | Version bump rejects the file on downgrade (old binary treats it as unsupported), a rollback data-loss path; keeping 1 with an additive field is forward/backward compatible. See D-03 analysis |
| `SaveStream(func(io.Writer) error)` | `SaveFrom(io.Reader)` with a pipe goroutine | A reader forces an extra goroutine per dump and a pipe-error protocol; a fill callback runs on the dumper goroutine with direct error return |
| A `DumpStore` method returning an errno class | `port.FailureClass` enum + `ClassedError` | The core may not import `syscall`; the enum must cross the boundary in a stdlib-only package |
| New Engine methods for time | `Config.Now`/`Config.After` injection | The repl seam is "time arrives as an argument"; for a goroutine-owning loop the equivalent is injected funcs — same discipline, no wall-clock reads inside `internal/app` |

### Package Legitimacy Audit

No external packages are added by this phase. The runtime graph stays at zero direct requires; the only "dependencies" are Go stdlib and in-repo packages. `npm view`/`pip index`/`cargo search` checks and the registry verdict table are therefore **not applicable** — there is nothing to legitimately check.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| — (none added) | — | — | — | — | — | — |

**Packages removed due to [SLOP] verdict:** none — no package was proposed.
**Packages flagged as suspicious [SUS]:** none — no package was proposed.

## Current-State Map (evidence for every requirement)

Every quote below was read from the working tree this session; line ranges are exact for commit `4f81fd7`. The rule applied: a discrete in-repo value (constant, enum, error, path) is quoted verbatim beside its citation.

### PERSIST-01 — the value limit exists three times and only two of them are configuration

The store's own copy, defaulted on any non-positive config:

```go
// internal/app/storage/store.go:14-18
const (
	defaultValueMaxBytes = 61440
	notDeletedFlag       = 0
	deletedFlag          = 1
)
```
`[VERIFIED: internal/app/storage/store.go:14-18]`

```go
// internal/app/storage/store.go:118-128
func (c Config) normalized() Config {
	if c.ValueMaxBytes <= 0 {
		c.ValueMaxBytes = defaultValueMaxBytes
	}
```
`[VERIFIED: internal/app/storage/store.go:118-124]`

The codec's frozen copy — decode-only, and the actual data-loss site:

```go
// internal/app/persist/codec.go:22-25
// maxRecordBytes is the design envelope from docs/IDEA.md — the codec's
// decode-side size bound: a record whose Data exceeds it is discarded, the
// config-shrink class (research Open Question 5).
const maxRecordBytes = 61440
```
`[VERIFIED: internal/app/persist/codec.go:22-25]`

```go
// internal/app/persist/codec.go:139-141
	if len(line.Data) > maxRecordBytes {
		return storage.Record{}, false
	}
```
`[VERIFIED: internal/app/persist/codec.go:139-141]`

The binary's third copy and the only validation next to it:

```go
// cmd/sup-kv/main.go:145
	defaultValueMaxBytes    = 61440
```
`[VERIFIED: cmd/sup-kv/main.go:145]`

```go
// cmd/sup-kv/main.go:357-358
	flag.IntVar(&cfg.valueMaxBytes, "value-max-bytes", envNum(envValueMaxBytes, defaultValueMaxBytes),
		"maximum value size in bytes (env SUPKV_VALUE_MAX_BYTES)")
```
`[VERIFIED: cmd/sup-kv/main.go:357-358]`

`checkDatagramCeiling` treats a non-positive value as "the default", so `--value-max-bytes=0` and negatives start today:

```go
// cmd/sup-kv/main.go:517-522
func checkDatagramCeiling(cfg config) error {
	valueBytes := cfg.valueMaxBytes
	if valueBytes <= 0 {
		valueBytes = defaultValueMaxBytes
	}
	probe := min(valueBytes, udprepl.MaxDatagramBytes)
```
`[VERIFIED: cmd/sup-kv/main.go:517-522]`

The only wiring point, where the engine and the store receive the flag value:

```go
// cmd/sup-kv/main.go:552-556
	dumps := persistfile.New(cfg.dumpPath)
	engine, discarded := persist.Open(cfg.nodeID, time.Now, cfg.driftWarn, dumps, storage.Config{
		ValueMaxBytes: cfg.valueMaxBytes,
		OnChange:      hub.Ingest,
	}, persist.Config{Debounce: cfg.dumpDebounce, MaxWait: cfg.dumpMaxWait})
```
`[VERIFIED: cmd/sup-kv/main.go:552-556]`

There is **no gRPC-side value check**: the Set/Delete handlers validate only the key (`internal/grpcapi/server.go:193-198`, `:253-255` — quoted under Security), and an oversize value surfaces from the store as `ErrValueTooLarge` → `InvalidArgument` via `mapError` (`internal/grpcapi/server.go:435-436`). D-01's "the gRPC Set validator" is therefore already the store's configured limit; adding a second check would create the very duplicate the decision removes (see Open Question 2).

The frozen tests that define acceptance:

- `internal/stand/stand_red_test.go:890-894` — sizes `redP01ValueMax = 64000`, `redP01ReviewBytes = 63000`, key `redP01KeyFmt = "red/p01/value-%d"` `[VERIFIED: internal/stand/stand_red_test.go:890-894]`; the scenario starts a node with `--value-max-bytes=64000`, Sets 63 000 and exactly 64 000 bytes, asserts the record is in the dump, SIGKILLs, restarts the same node, and asserts `Get` returns the full value (`:929-958`).
- `internal/persistfile/red_value_limit_test.go:18-20` — `redP01ValueMaxBytes = 64000`, `redP01ValueBytes = 63000`, `redP01EngineKey = "red/p01/engine-value"` `[VERIFIED: internal/persistfile/red_value_limit_test.go:18-20]`; the engine test calls `persist.Open(seedNodeID, seedFixedNow(seedNowMs), seedDriftWarn, persistfile.New(dumpPath), cfg, persist.Config{Debounce: seedDebounce})` and destructures `(eng, discarded)` (`:51-57`) `[VERIFIED: internal/persistfile/red_value_limit_test.go:51-57]`.
- `scripts/red-expectations.tsv` rows: `TestRed_P01_valueLimit	red` and `TestRed_P01_valueLimitEngine	red` `[VERIFIED: scripts/red-expectations.tsv:29-30]`.

**Consequence for the planner:** the frozen engine test compiles against `persist.Open`'s current return shape and against `persist.Config`/`storage.Config` keyed literals. `Open` MUST keep returning `(*Engine, int)`. The codec limit must be derivable from `storage.Config` alone (before a store exists), and the flag value 64 000 must remain accepted by the datagram-ceiling check.

### PERSIST-02 — `Load` cannot distinguish absent from corrupt, and reports nothing

```go
// internal/port/dump.go:26-30
	// Load resolves the fallback chain main then .prev and returns the
	// first generation whose bytes pass the caller-supplied parse
	// oracle; a truncated or unparseable file counts as unreadable and
	// advances the chain (D-02). ok false is the empty start. parse is
	// supplied by the engine (codec.Parse) so the adapter stays
	// record-blind, and Load itself never reports an error: every
	// per-generation fault is chain business.
	Load(parse func([]byte) error) (dump []byte, ok bool)
```
`[VERIFIED: internal/port/dump.go:26-30]`

```go
// internal/persistfile/dumpfile.go:82-94
func (f *FileStore) Load(parse func([]byte) error) (dump []byte, ok bool) {
	for _, candidate := range []string{f.path, f.path + prevSuffix} {
		data, err := os.ReadFile(candidate) //nolint:gosec // G304: candidate derives from the configured dump path, never caller input
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
`[VERIFIED: internal/persistfile/dumpfile.go:82-94]`

Every fault is discarded: a read error, a permission error, a truncated file and a format-version mismatch are indistinguishable, and nothing is preserved or logged. `Open` then ignores the boolean:

```go
// internal/app/persist/engine.go:133-145
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
```
`[VERIFIED: internal/app/persist/engine.go:133-145]`

The only startup log is the discarded-record count, and it says nothing about which generation was used:

```go
// cmd/sup-kv/main.go:557-559
	if discarded > 0 {
		log.Printf("sup-kv: discarded %d undecodable dumped records at startup", discarded)
	}
```
`[VERIFIED: cmd/sup-kv/main.go:557-559]`

Rotation is rotate-before-rename, so the next dump overwrites the loaded `.prev` with whatever main holds:

```go
// internal/persistfile/dumpfile.go:119-127
	err = ops.rotate(path, prevPath) // TEST-03 step 3: rotate main → .prev
	if err != nil {
		return err
	}
	err = ops.rename(path) // TEST-03 step 4: rename temp → main
```
`[VERIFIED: internal/persistfile/dumpfile.go:119-127]`

`Snapshot` copies the main file with no dirty check of its own beyond `flush`'s clean fast path:

```go
// internal/persistfile/snapshot.go:50-54
	data, err := readMainDump(f.path)
	if err != nil {
		return err
	}
	return saveAtomically(f.ops, filepath.Join(snapshotDir, name), "", data)
```
`[VERIFIED: internal/persistfile/snapshot.go:50-54]`

The existing fallback-chain test asserts bytes only (`internal/persistfile/dumpfile_test.go:97-154`) and has no case for preservation, outcomes, or logging.

### PERSIST-03 — the format has no census, and one syntax corruption kills the whole generation

```go
// internal/app/persist/codec.go:17-20
// dumpFormatVersion is the on-disk dump format this package writes and
// reads; a generation announcing any other version is unreadable and the
// fallback chain advances (research A4).
const dumpFormatVersion = 1
```
`[VERIFIED: internal/app/persist/codec.go:17-20]`

```go
// internal/app/persist/codec.go:39-41
type header struct {
	FormatVersion int `json:"format_version"`
}
```
`[VERIFIED: internal/app/persist/codec.go:39-41]`

`Encode` writes the header and then sorted record lines; nothing records how many lines were written:

```go
// internal/app/persist/codec.go:66-70
	err := enc.Encode(header{FormatVersion: dumpFormatVersion})
	if err != nil {
		return nil, err
	}
	for _, key := range slices.Sorted(maps.Keys(records)) {
```
`[VERIFIED: internal/app/persist/codec.go:66-70]`

`Decode` streams with `json.Decoder`; any non-EOF error aborts the whole generation:

```go
// internal/app/persist/codec.go:108-116
	for {
		var line recordLine
		err = dec.Decode(&line)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return records, discarded, nil
			}
			return nil, 0, fmt.Errorf("persist: dump record unreadable: %w", err)
		}
```
`[VERIFIED: internal/app/persist/codec.go:108-116]`

I verified the consequence with a probe against this Go toolchain (a typed record struct with a `[]byte` field, three lines, the middle one corrupted):

- A base64 error in the middle line does **not** poison `json.Decoder`; the next `Decode` returns the following record.
- A raw JSON syntax error **does** poison it permanently: every subsequent `Decode` returns the same error.
- Line-wise `json.Unmarshal` recovers in both cases, losing only the bad line.

`[VERIFIED: local Go 1.27.4 probe, /tmp/jsonprobe/main.go output]` — the exact output was `stream after-error: ... invalid character 'n' looking for beginning of object key string` after a syntax error, versus the third record decoding cleanly with line-wise unmarshalling. This is why P-03's "44 % of single-byte corruptions reject the generation" happens, and why only a line-oriented decoder can satisfy "one corrupted record discards only that record".

Truncation today: a cut mid-line yields `unexpected end of JSON input`-class errors (generation fault, tested at `codec_test.go:494-513` with `tornTailCut = 10`), but a cut exactly at a line boundary is indistinguishable from a complete file.

The tests that pin current behaviour and will need updating:
- `codec_test.go:370-394` builds a hand-joined duplicate-key dump from two `Encode` outputs; a census rule must tolerate or the fixture must be rebuilt with a correct count.
- `codec_test.go:494-513` expects a generation-level error for a mid-line cut.
- `fuzz_dump_test.go:49-50` pins the literal `fuzzDumpHeader = "{\"format_version\":1}\n"` and the `Decode(dump) (records, discarded, err)` shape `[VERIFIED: internal/app/persist/fuzz_dump_test.go:49-50]`.

### PERSIST-04 — the engine starts clean after any load, and `markDirty` does not wake the dumper

```go
// internal/app/persist/engine.go:101-110
func New(store *storage.Store, dumps port.DumpStore, cfg Config) *Engine {
	return &Engine{
		store:   store,
		dumps:   dumps,
		cfg:     cfg.normalized(),
		changed: make(chan struct{}, 1),
		flushCh: make(chan chan error, 1),
		runDone: make(chan struct{}),
	}
}
```
`[VERIFIED: internal/app/persist/engine.go:101-110]` — `dirty` is the zero value, so `Open` (which returns `New(...)`) starts clean regardless of what was loaded or skipped.

The Phase-4 New-1 gap, verbatim — no nudge from `markDirty`:

```go
// internal/app/persist/engine.go:785-792
func (e *Engine) markDirty() {
	e.mu.Lock()
	if !e.dirty {
		e.oldest = time.Now()
	}
	e.dirty = true
	e.mu.Unlock()
}
```
`[VERIFIED: internal/app/persist/engine.go:785-792]`

`ApplyRemote` carries its own nudge after calling it (`engine.go:496-500`), and `await` nudges after registering (`engine.go:449-452`); `replaySnapshot`'s mid-replay error path (`engine.go:656-661`) does **not**, which is exactly New-1:

```go
// internal/app/persist/engine.go:656-661
		ver, replayed, err := replayRecord(e.store, key, records[key])
		if err != nil {
			if len(restored) > 0 {
				e.markDirty()
			}
			return restored, err
		}
```
`[VERIFIED: internal/app/persist/engine.go:656-661]`

Why the current code loses the last good generation after a backup start: `Open` loads `.prev` into memory and starts clean; `Snapshot` sees clean, skips the flush, and `CreateSnapshot` copies the corrupt main; the first later dump rotates the corrupt main onto `.prev`, destroying the generation that was loaded. Marking dirty fixes the snapshot half but *accelerates* the rotation half unless the corrupt main has been moved out of the way (see D-02/D-04).

### PERSIST-05 — failures retry at the debounce cadence, unlogged and unclassified

```go
// internal/app/persist/engine.go:359-374
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
`[VERIFIED: internal/app/persist/engine.go:359-374]`

Re-arming then uses `cfg.Debounce` for the next attempt:

```go
// internal/app/persist/engine.go:228-245
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
`[VERIFIED: internal/app/persist/engine.go:228-245]`

With `defaultDebounce = 50 * time.Millisecond` (`engine.go:22`), a persistent failure retries every ~50 ms and can never reach a cap. The loop discards the error entirely:

```go
// internal/app/persist/engine.go:202-204
		if fired {
			_ = e.runDump() // the error belongs to the waiters; nobody is left to report to.
		}
```
`[VERIFIED: internal/app/persist/engine.go:202-204]` and `internal/app/persist/engine.go:319-325` counts only a total (`dumpErrors`). There is no logger in the package and no class notion. Health is gate-only: `SERVING` is set solely when the startup gate opens (`internal/grpcapi/gate.go:177`), never touched by dump outcomes, so "health stays SERVING" is a documentation+test obligation, not a code change `[VERIFIED: internal/grpcapi/gate.go:177]`.

`TestEngine_dumpFailureFansOut` (`engine_test.go:498-563`) pins that exactly one failed and one recovered save occur — it does not pin the interval or growth; the new backoff test can extend it.

### PERSIST-06 — one dump cycle materializes the state, the bytes and the copy

```go
// internal/app/persist/engine.go:295-307
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
```
`[VERIFIED: internal/app/persist/engine.go:295-307]`

Peak live heap is the `Records()` deep copy plus the `bytes.Buffer` in `Encode` (which grows by doubling), and `Save([]byte)` holds the full byte slice for the entire I/O; `Records()` also clones every Data value while holding the store's RLock (`internal/app/storage/store.go:369-382`), which is the writer pause P-06 measured at 74 ms. The port takes whole bytes:

```go
// internal/port/dump.go:16-21
type DumpStore interface {
	// Save durably writes dump as the new main generation, rotating the
	// old main generation to .prev. A non-nil error means the generation
	// did not become durable; the caller treats every such error as a
	// dump failure (D-05).
	Save(dump []byte) error
```
`[VERIFIED: internal/port/dump.go:16-21]`

`BenchmarkDump10kKeys` (`internal/app/persist/bench_test.go:73-97`) is the existing measurement seam; it calls `persist.Encode(store.Records())` then `dumps.Save(block)` and reports `bytes/op` and `dump-bytes`.

### PERSIST-07 — equal version with different content is a silent no-op, on both the store and the wire

```go
// internal/app/storage/store.go:324-343
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
`[VERIFIED: internal/app/storage/store.go:324-343]`

The replication core short-circuits equal versions before the store is ever consulted:

```go
// internal/app/repl/core.go:1477-1488
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
```
`[VERIFIED: internal/app/repl/core.go:1477-1488]`

Therefore a content-based tie-break in `storage.join` alone is **dead code in production**: two replicas holding different content at the same version will Ack each other forever. P-07's fix touches the core.

`ApplyRemote`'s dirty heuristic also cannot survive equal versions, because at equal versions both `Winner` values are the same bytes:

```go
// internal/app/persist/engine.go:490-495
	displaced := len(conflicts) > 0 && conflicts[0].Winner == rec.Version
	if !displaced && getErr == nil {
		// Rejected stale redelivery or equal-version redelivery over the
		// live record the probe saw: the store did not change.
		return conflicts, drift, nil
	}
```
`[VERIFIED: internal/app/persist/engine.go:490-495]`

Tests that pin the old law and must be rewritten (not frozen):
- `internal/app/storage/store_test.go:510-530` — subtest `"equal version is a no-op with no conflicts"`, seeding data `"a"` and merging the same version with data `"x"`, asserting `conflicts == nil` and the stored `"a"` unchanged. This is literally the P-07 pair.
- `internal/app/storage/merge_test.go:222-261` — `TestMerge_losersProperty` asserts `case cmpSame: want = nil` and that no Conflict ever pairs equal versions.
- `TestMerge_duplicateDelivery` (`merge_test.go:194-220`) must stay green (identical content stays a no-op).
- Registry row `S-ALG-REP-03` currently says "An equal version is acknowledged without triggering a dump" — `[VERIFIED: docs/SPEC-DEVIATIONS.md:125]`; and `S-HLC-07` records the zero filler as an accepted deviation `[VERIFIED: docs/SPEC-DEVIATIONS.md:53]`.

### PERSIST-08 — no engine call has a deadline

```go
// internal/app/persist/engine.go:449-454
	select {
	case e.changed <- struct{}{}:
	default:
	}
	<-w.done
	return w.err
```
`[VERIFIED: internal/app/persist/engine.go:449-454]` — unbounded.

```go
// internal/app/persist/engine.go:763-778
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
		case err := <-reply: // the exit sweep answered us after all
			return err
		default:
			return ErrClosed
		}
	}
```
`[VERIFIED: internal/app/persist/engine.go:763-778]` — the first send can block behind a queued flush, the reply wait is unbounded, and `Run` cannot service anything while parked inside `runDump`→`Save`.

Shutdown compounds it:

```go
// cmd/sup-kv/main.go:828-837
	err := node.engine.Flush()
	if err != nil {
		return fmt.Errorf("final dump failed: %w", err)
	}

	node.engineCancel()
	runErr := <-node.engineDone
	if !errors.Is(runErr, context.Canceled) {
		log.Printf("sup-kv: engine Run returned %v", runErr)
	}
```
`[VERIFIED: cmd/sup-kv/main.go:828-837]` — a hung fsync parks both waits forever, so SIGKILL is the only exit.

The frozen blocking proof (`internal/app/persist/engine_blocking_test.go:88-126`) requires that an in-flight `Save` keeps `Set` blocked; its windows are `notReturnedWindow = 200ms`, `releaseWait = 1s` `[VERIFIED: internal/app/persist/engine_blocking_test.go:18-25]`. A confirmation timeout well above one second keeps it green.

The RPC layer's totality branch only recognises `ErrDumpFailed`:

```go
// internal/grpcapi/server.go:217-220
		if errors.Is(err, persist.ErrDumpFailed) {
			s.notifyLocalWrite(req.GetKey(), ver, req.GetValue(), false, false)
		}
		return nil, mapError(err)
```
`[VERIFIED: internal/grpcapi/server.go:217-220]` — the same shape at `:263-266` for Delete.

### PERSIST-09 — no sweep, no parent sync, and the darwin helper is already gone

```go
// internal/persistfile/dumpfile.go:27-31
const (
	prevSuffix  = ".prev"
	tempPattern = "dump-*"
	dirMode     = 0o700
)
```
`[VERIFIED: internal/persistfile/dumpfile.go:27-31]`

```go
// internal/persistfile/dumpfile.go:105-128
func saveAtomically(ops fsops, path, prevPath string, data []byte) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, dirMode)
	if err != nil {
		return err
	}
	err = ops.write(dir, data) // TEST-03 step 1: create temp + write payload
	if err != nil {
		return err
	}
	err = ops.fsync() // TEST-03 step 2: fsync the temp, close it
	...
	err = ops.rename(path) // TEST-03 step 4: rename temp → main
	if err != nil {
		return err
	}
	return ops.syncDir(dir) // TEST-03 step 5: fsync the directory
}
```
`[VERIFIED: internal/persistfile/dumpfile.go:105-128]` — `MkdirAll` is called for every save and nothing syncs the *parent* of a newly created directory; no code enumerates `dump-*` leftovers; the snapshot directory is created the same way (`snapshot.go:45-48`).

`fsync_darwin.go` does not exist in the tree `[VERIFIED: find . -name 'fsync*' → no results]`; the fsops doc already claims the stdlib path:

```go
// internal/persistfile/fsops.go:20-26
// The platform durability detail deliberately lives in the standard
// library, not here: [os.File.Sync] on darwin goes through internal/poll's
// darwin Fsync, which issues F_FULLFSYNC with EINTR retry and an ENOTSUP
// fallback to plain fsync — a strict superset of the [syscall.Fsync] path a
// hand-rolled helper would reach. The seam therefore calls [os.File.Sync]
// directly and carries no platform file of its own (REVIEW ENG-18, P-09;
// Phase 5's P-09 absorbs the remaining orphan-temp/parent-dir work).
```
`[VERIFIED: internal/persistfile/fsops.go:20-26]`

That claim is correct, and I verified the mechanism in the toolchain source:

- Linux/posix: `func (fd *FD) Fsync() error { ... return ignoringEINTR(func() error { return syscall.Fsync(fd.Sysfd) }) }` `[VERIFIED: GOROOT src/internal/poll/fd_fsync_posix.go:20-27]`
- Darwin: `_, err := unix.Fcntl(fd.Sysfd, syscall.F_FULLFSYNC, 0)`, then `if err != nil && errors.Is(err, syscall.ENOTSUP) { err = syscall.Fsync(fd.Sysfd) }`, all inside `ignoringEINTR` `[VERIFIED: GOROOT src/internal/poll/fd_fsync_darwin.go:20-31]`

So EINTR is already retried for both file and directory sync (`syncDir` calls `d.Sync()` → the same `FD.Fsync`), and darwin ENOTSUP already falls back. The remaining gap is a **directory** sync returning `ENOTSUP`/`EINVAL` on filesystems that cannot fsync a directory; `os.File.Sync` returns that error verbatim and today it fails the whole save after the generation is already placed (the `syncDir` step is last, so the data is durable; the error is an unnecessary failure signal). Whether a given filesystem returns `ENOTSUP` or `EINVAL` for a directory fsync is **not** verified in this session — `[ASSUMED]`, and the reason it must be a documented skip with both errnos tolerated.

### PERSIST-10 — skipped oversize records share one counter with corruption and vanish from disk

`Decode` lumps every per-record fault into one `discarded` integer (`internal/app/persist/codec.go:97-124`), and `Open` returns that integer as its only diagnostic (`engine.go:125-151`). `mergeRecords` silently skips a store rejection:

```go
// internal/app/persist/engine.go:157-164
func mergeRecords(store *storage.Store, records map[string]storage.Record) {
	for key, rec := range records {
		_, _, err := store.Merge(key, rec)
		if errors.Is(err, storage.ErrValueTooLarge) {
			continue
		}
	}
}
```
`[VERIFIED: internal/app/persist/engine.go:157-164]`

`TestEngine_configShrunkValueLimits` (`engine_test.go:1679-1735`) pins today's split: a 64-byte record is written at the default limit, then reopened with `ValueMaxBytes: 16`; the test asserts `discarded == 0` with the comment "the skip is at merge, not codec" (`:1707-1709`) `[VERIFIED: internal/app/persist/engine_test.go:1707-1709]`. Under D-01/D-09 the codec will apply the configured limit, so this assertion legitimately changes to one skipped-value-limit record — the planner must not treat that test edit as a regression.

Evidence preservation today is accidental: the record is dropped at merge, but the loaded file remains on disk until the next dump rotates it; after two dumps it is gone. No count, no class, no WARN, no metric.

### PERSIST-11 — `Engine` currently owns five responsibilities in one 792-line file

`internal/app/persist/engine.go` holds: startup composition (`Open`, `mergeRecords`), the store facade (`Get`, `Lookup`, `Version`, `Versions`, `Observe`, `KeyCount`, `Generation`, `Records`, `ApplyRemote`), the dump scheduler (`Run`, `runDump`, `flushDump`, `rearm`, timers), confirmation accounting (`waiter`, `await`, `confirmWaiters`, `failWaiters`, `sweepClosed`, `DumpStats`) and snapshot orchestration (`Snapshot`, `Restore`, `replaySnapshot`, `restoreChanges`, `replayRecord`), plus `Flush`/`flush`/`markDirty`. The public method set is consumed by `internal/grpcapi`, `internal/udprepl`, `cmd/sup-kv`, `persistfile` tests and the frozen P-01 test; all of those signatures must survive the split.

The port's diagnostic gap is the other half:

```go
// internal/port/dump.go:30
	Load(parse func([]byte) error) (dump []byte, ok bool)
```
`[VERIFIED: internal/port/dump.go:30]` — no outcome, no cause, no path.

### PERSIST-12 — the engine reads the wall clock directly while the repl core takes time as an argument

Every wall-clock site in the engine: `time.NewTimer`/`Reset` and `time.Until` in `rearm` (`engine.go:237-244`), `time.Now` in `runDump` (`:302`), `time.Since` (`:307`), `time.Now` in `failWaiters` (`:366`), `time.Now` in `await` (`:444`). The repl core by contrast owns no clock and no timer: every handler takes `now time.Time` (`internal/app/repl/core.go:1007` `HandleDatagram(now time.Time, ...)`, `:2244` `Advance(now time.Time, ...)`) `[VERIFIED: internal/app/repl/core.go:1007,2244]`, and the adapter supplies `time.Now()` at its own loop (`internal/udprepl/repl.go:558-587`) `[VERIFIED: internal/udprepl/repl.go:558-587]`.

The simulator already models the production ordering: `Policy.AnnounceDelay` adds extra virtual latency to `DataReplication` deliveries "mirroring production's dump-then-broadcast order" (`internal/app/internal/simnet/simnet.go:155-168`) `[VERIFIED: internal/app/internal/simnet/simnet.go:155-168]`. But no `persist.Engine` is part of the simulator today: `Cluster` builds its own stores and cores (`simnet.go:533-556`), each store's clock reads `sched.Now()` plus a per-node offset. `Cluster.Store(i)` returns the store pointer, so an engine can share a cluster node's store; what is missing is a way to notify the core about a record the *engine* installed (as opposed to `Cluster.Write`, which re-Sets the key). See D-11.
## Design Options and Recommendations (D-01 … D-12)

Each decision below states the options considered, the recommendation, the concrete contract, and the acceptance tests that must exist. The recommendation is what the planner should encode; the alternatives are recorded so a later reviewer can see the trade-off without re-deriving it.

### D-01 (PERSIST-01) — one value limit, owned by the store, read by the codec

**The problem with "pass a limit everywhere":** the decoder runs before the store is constructed (`Open` decodes first, derives the watermark, then builds the clock and store: `engine.go:133-149`), so the limit cannot be read from a store that does not exist yet, and two configuration inputs (`storage.Config.ValueMaxBytes` and a new `persist.Config.ValueMaxBytes`) can disagree — which is exactly how P-01 happened.

**Options:**

| Option | Shape | Verdict |
|--------|-------|---------|
| A. Store owns the effective limit; a `Config` helper normalizes it for the pre-store decode | `storage.ValueMaxBytes` exported; `func (c Config) ValueLimit() int`; `func (s *Store) ValueMaxBytes() int` | **Recommended** — one source, no way to disagree, and the pre-store decode gets the same normalized value the store will use |
| B. `persist.Config.ValueMaxBytes` alongside `storage.Config.ValueMaxBytes` | two keyed fields, main passes both | Rejected — two inputs; a future wiring mistake reintroduces P-01 |
| C. A `persist.Codec{ValueMaxBytes}` value threading through everything | struct receiver on Encode/Decode | Rejected — more surface for the same guarantee; the engine already has a config seam |

**Recommended contract (sketches):**

```go
// internal/app/storage/store.go — the ONLY definition of the number.
const ValueMaxBytes = 61440

// ValueLimit returns the effective limit: ValueMaxBytes when the configured
// value is non-positive, otherwise the configured value. NewStore uses this
// same helper, so the pre-store decode and the store can never disagree.
func (c Config) ValueLimit() int

// ValueMaxBytes reports the store's effective limit (immutable after NewStore).
func (s *Store) ValueMaxBytes() int

// internal/app/persist/codec.go — the codec's frozen constant is deleted.
func Encode(records map[string]storage.Record, valueMaxBytes int) ([]byte, error)
func EncodeTo(w io.Writer, records map[string]storage.Record, valueMaxBytes int) error
func Decode(dump []byte, valueMaxBytes int) (map[string]storage.Record, DecodeStats, error)
```

- `persist.Open` computes `limit := storeCfg.ValueLimit()` before decoding; `New` captures `limit := store.ValueMaxBytes()` for its encoder. There is no third path.
- `cmd/sup-kv/main.go` deletes `defaultValueMaxBytes` and uses `storage.ValueMaxBytes` for the flag default and for `checkDatagramCeiling`. The literal `61440` then exists in exactly one production file (`storage/store.go`), plus one pinning test.
- Non-positive validation: add a small predicate called from `finalizeConfig` (keep it a named helper so Phase 7's one-point validation can absorb it):

```go
func checkValueMaxBytes(cfg config) error {
	if cfg.valueMaxBytes <= 0 {
		return fmt.Errorf("%w: --value-max-bytes=%d", errValueMaxBytesRange, cfg.valueMaxBytes)
	}
	return nil
}
```

  Then `checkDatagramCeiling`'s `valueBytes <= 0` branch becomes unreachable for the binary (keep it defensively or simplify; do not change the ceiling arithmetic). `--value-max-bytes=0` and negatives exit 2 through the existing `finalizeConfig` → `os.Exit(exitUsage)` path (`main.go:233-238`).
- **No new gRPC validator.** The store's configured limit is the enforcement point; `mapError` already maps `ErrValueTooLarge` to `InvalidArgument` (`grpcapi/server.go:435-436`). Adding a handler-side check would create a second copy of the limit and a new way for the two to disagree. Document this reading of D-01 in the plan (Open Question 2).
- Encode-side use of the limit is a tripwire, not a filter: an oversize record in the map means the store invariant was violated, so `EncodeTo` returns an error (never skips). It is unreachable through `Set`/`Merge`/`Open`/`Restore`, all of which reject oversize. Codec tests that need to build an oversize *decode* fixture pass a larger encode limit than decode limit (see Pitfall 12).

**Acceptance tests:** the two frozen P-01 tests (flip TSV only), a new `TestMain_valueMaxNonPositive` (0 and −1 → exit 2 naming the flag), a new pin `TestStorage_valueLimitIs61440` asserting `storage.ValueMaxBytes == 61440` and `Config{}.ValueLimit() == 61440`, and a restart test at two accepted flag values (64000 and a small value such as 8) proving the exact-limit boundary in both directions.

### D-02 (PERSIST-02) — explicit load outcomes, and preservation as part of loading

**Options for the outcome contract:**

| Option | Shape | Verdict |
|--------|-------|---------|
| A. Struct result through the port, enum + per-candidate failures | `LoadResult{Dump, Outcome, Generation, Failures, Sweep}` | **Recommended** — carries path, cause and preservation result, which D-02 requires for the WARN/ERROR text |
| B. Enum + error return | `Load(...) (dump []byte, outcome LoadOutcome, cause error)` | Rejected — cannot report two failed generations, their distinct paths, or preservation errors |
| C. Keep `(dump, ok)` and add a separate `Diagnostics()` call | stateful adapter read after Load | Rejected — racy and awkward; the information is produced by Load |

**Recommended contract:**

```go
// internal/port/dump.go — stdlib only.
type LoadOutcome uint8

const (
	LoadPrompt   LoadOutcome = iota // no generation file existed at all
	LoadMain                        // the main generation loaded
	LoadBackup                      // .prev loaded; main absent or unusable
	LoadUnusable                    // a generation existed, none loaded
)

type Generation uint8

const (
	GenMain Generation = iota
	GenPrev
)

type GenerationFailure struct {
	Candidate Generation
	Path      string
	Cause     error  // read error or the parse oracle's decode error
	Preserved string // the .corrupt-<stamp> path when the rename succeeded
	PreserveErr error
}

type LoadResult struct {
	Dump       []byte
	Outcome    LoadOutcome
	Generation Generation // meaningful for LoadMain / LoadBackup
	Failures   []GenerationFailure
	Swept      int   // orphan temps removed
	SweepErr   error
}

type DumpStore interface {
	SaveStream(fill func(io.Writer) error) error
	Load(now time.Time, parse func([]byte) error) LoadResult
	CreateSnapshot(name string) error
	ReadSnapshot(name string) ([]byte, error)
}
```

**Why `now` is a parameter:** the preserved name is time-derived (`<path>.corrupt-<UTC>`), and the engine already owns the startup clock (`Open`'s `now func() time.Time`). Passing `now()` into Load keeps `persistfile` clock-free and makes the name deterministic in tests (the frozen P-01 test injects `seedFixedNow`, so a test can assert the exact suffix). Alternative: derive the stamp from the file's mtime — rejected because it reports when the file was last written, not when it was found corrupt, and two candidates could share an mtime.

**Preservation rules (the core of D-02):**

1. For each candidate in order `main`, `main+".prev"`:
   - `os.ReadFile`: `fs.ErrNotExist` → record "absent" (not a failure); any other error → failure with cause, then try to preserve.
   - `parse(data)`: error → failure with cause, then preserve.
   - success → this generation is the loaded one; stop (do **not** preserve it).
2. Preserve = rename the failed candidate to `<candidate path> + ".corrupt-" + stamp`, where `stamp = now.UTC().Format("20060102T150405Z")`; if the target exists, append `-1`, `-2`, … until free (probe with `os.Lstat`; the directory is 0700, so the TOCTOU window is owner-only). Only regular files are renamed; a directory or device at the dump path is a failure with cause and no rename. After a successful rename, best-effort `syncDir(parent)`.
3. Outcome: loaded main → `LoadMain`; loaded prev → `LoadBackup`; nothing loaded and at least one candidate existed (readable or not) → `LoadUnusable`; no candidate existed → `LoadPrompt`.
4. **Preservation is what keeps `.prev` alive.** After a `LoadBackup`, the corrupt main is gone from `main`, so the first dump's `rotate` finds no main and is a no-op (`fileFSOps.rotate` guards on `targetExists(main)`, `fsops.go:122-125`), leaving the loaded `.prev` intact. Without step 1 the first rotation would put the corrupt file where the good generation lives — the P-04 loss. The plan must assert this ordering explicitly (a test that snapshots `.prev` bytes across the first post-backup dump).

**Engine report (no signature change):** `persist.Open` keeps returning `(*Engine, int)` (frozen-test requirement). It stores a rich report on the engine:

```go
type DecodeStats struct {
	Checksum   int // checksum mismatch
	Version    int // undecodable/wrong-length record_version
	ValueLimit int // Data over the configured limit
	Malformed  int // line failed to parse as a record object
}
func (s DecodeStats) Total() int

type LoadStats struct {
	Outcome    port.LoadOutcome
	Generation port.Generation
	Failures   []port.GenerationFailure
	Decoded    DecodeStats
	Swept      int
	SweepErr   error
}

func (e *Engine) LoadStats() LoadStats // zero for engines built by New; set by Open
```

`Open`'s second return stays `stats.Total()` of the accepted generation, preserving the existing log line and every current test's expectation (except the shrunken-limit test, see D-09).

**Logging policy (in `cmd`, one line per class, never duplicated):**

| Outcome | Log | Metric |
|---------|-----|--------|
| `LoadPrompt` | none | `supkv_dump_load_total{outcome="prompt"} 1` |
| `LoadMain` | none (existing INFO for discarded > 0 stays) | `{outcome="main"} 1` |
| `LoadBackup` | exactly one `slog.Warn("Dump generation fallback", "main_path", …, "cause", …)` naming the main path and why; if main was merely absent the cause is "main generation absent (crash residue)" | `{outcome="backup"} 1` |
| `LoadUnusable` | exactly one `slog.Error("No usable dump generation", "main", …, "prev", …)` carrying both causes | `{outcome="unusable"} 1` (non-zero by definition) |

The WARN/ERROR text must never contain the values' bytes (only paths and error strings). `mapError`'s "no server paths in RPC messages" rule (`API-03`) applies to RPCs, not logs; logs are where paths belong.

**Acceptance tests:** `persistfile` unit tests for each outcome and preservation (including both-unusable, main-absent/prev-good, unreadable-with-EACCES via chmod, and a directory at the path), the rotation-survival test, an engine `LoadStats` test (the report maps the port result and the decode stats of the accepted generation), and a stand test: pre-seed `.prev` good + main garbage → one WARN with the path, a `.corrupt-*` file exists, `.prev` unchanged after a write+shutdown, server serves `.prev`'s records; both corrupt → one ERROR, nonzero `supkv_dump_load_total{outcome="unusable"}`, both files preserved, node starts empty; no files → no WARN/ERROR lines.

### D-03 (PERSIST-03) — an additive record census plus line-oriented decoding

**Options:**

| Option | Truncation detection | v1.0 reads | Downgrade on new dumps | Verdict |
|--------|---------------------|------------|------------------------|---------|
| A. `format_version: 2` + `record_count` in the header; accept 1 and 2 | yes, for v2 | yes (v1 legacy) | **no** — a v1.0 binary rejects v2 as unsupported ⇒ starts empty and (without preservation) loses the path | Rejected for operations, though the most "versioned" |
| B. Trailer line with the count | yes (missing trailer) | **no** — every v1.0 dump looks truncated at upgrade | n/a | Rejected: contradicts "v1.0 dumps preserved for reading" |
| C. Keep `format_version: 1`, add optional `record_count` to the header | yes, when present | yes (absent ⇒ legacy) | **yes** — v1.0 ignores the unknown field and reads the records | **Recommended** |

**Recommended format (C), with the completeness rule that survives corruption:**

```json
{"format_version":1,"record_count":3}
{"key":"a","record_version":"<32 hex>","data":"<base64>","deleted":false,"checksum":123}
...
```

- Writer: `header{FormatVersion: 1, RecordCount: &n}` where `n = len(records)`. Byte stability holds (sorted keys, no timestamps).
- Reader: parse the header with `RecordCount *int`. Absent ⇒ legacy v1.0 generation: read all lines, no census (this is the documented compatibility path; a line-boundary truncation of a legacy file is undetectable — record it in the memory/format docs). Present ⇒ census enforced.
- **Completeness rule (chosen to satisfy both "truncation is incomplete" and "one corrupt record loses only itself"):**
  1. Count every non-empty line after the header as a record line (`L`).
  2. If the final line is unterminated **and** fails to parse as a record object ⇒ generation-level error (torn tail; matches `TestCodec_truncatedGenerationUnreadable`).
  3. If `record_count` is present and `L < record_count` ⇒ generation-level error ("incomplete generation": cut at a line boundary, or a truncated tail whose fragment was dropped).
  4. Otherwise accept: each line is parsed independently; a line that fails to parse is one discarded record (`Malformed`), a structurally valid line with a bad version/oversize/checksum is that class's discard. `L > record_count` is tolerated (hand edits, duplicate keys), so the `codec_test.go:370-394` last-wins fixture keeps working without a count rewrite.
- Note the residual: a bit flip that lowers `record_count` below `L` is undetectable. It is a header-only corruption in a file whose record checksums still protect the payload; document it rather than adding a header checksum in this phase.

**Line-oriented decode implementation (the P-03 fix):**

- Use `bufio.NewReaderSize(bytes.NewReader(dump), maxDumpLineBytes(limit)+1)` and `ReadSlice('\n')`. `ReadSlice` bounds a line by the buffer size, returns `bufio.ErrBufferFull` for an overlong line (treat as a generation-level error — memory safety beats per-record isolation for a line that cannot be a legal record), and reports `io.EOF` on an unterminated final line.
- `maxDumpLineBytes(limit)` = base64 expansion of the limit + the key ceiling + fixed JSON overhead, e.g. `(limit+2)/3*4 + storage.MaxKeyBytes + 256`. Name the constant and test the boundary: a record at exactly the limit parses; one byte over is discarded as `ValueLimit`, not as `Malformed`.
- Parse each line with `json.Unmarshal` into `recordLine` (not a stream `json.Decoder`). A syntax-corrupt line is isolated; the probe above proved the stream decoder cannot do this.
- Header: first line, `json.Unmarshal` into `header`; unknown fields ignored (forward compatibility); empty input ⇒ generation-level error (existing test).
- Duplicate keys: last-wins (preserve the existing documented behaviour).

**Why the census fixes the line-boundary case:** `Encode` writes every record line terminated by `\n`; a cut at a boundary leaves `L = N − k < N` ⇒ incomplete. A cut mid-line leaves an unterminated fragment that fails to parse ⇒ torn-tail error. A syntax-corruption that injects a newline makes `L = N + 1`, the two fragments fail to parse (both counted as `Malformed`), and the rest loads — the corrupted record is lost, no other record is.

**Acceptance tests:** header round-trip and byte stability with the count; legacy v1.0 header without count loads; v1.0 line-boundary truncation is accepted (documented limitation, an explicit test naming it); v1.1 line-boundary truncation ⇒ generation error; mid-line cut ⇒ generation error; a single base64/syntax-corrupt line ⇒ `Malformed == 1` and all siblings load; overlong line ⇒ generation error; duplicate-key hand edit ⇒ last-wins; fuzz invariants updated (`len(records)+stats.Total() <= L`, no panics, generation faults return nil records and a zero stats value).

### D-04 (PERSIST-04) — dirty after Prompt/Backup/Unusable, and the nudge belongs to `markDirty`

**Rule:** `Open` marks the engine dirty when the outcome is `LoadPrompt`, `LoadBackup` or `LoadUnusable`; it leaves it clean on `LoadMain` — including a main load that discarded records. Rationale:

- `LoadPrompt`: there is no dump at all, so `Snapshot` must not return `ErrNoDump`; dirty forces the flush before the copy.
- `LoadBackup`: memory came from `.prev`; the main generation must be re-established before anything can displace `.prev`. Combined with D-02's preservation, the first dump writes a fresh main and the rotate is a no-op.
- `LoadUnusable`: same, with an empty memory state.
- `LoadMain` with `ValueLimit`/checksum discards: the *loaded* bytes are on disk and a snapshot of them decodes to the same in-memory state (the same decoder discards the same lines), so clean is correct; marking dirty here would rewrite the file immediately and shorten the evidence window PERSIST-10 wants.

**Nudge (Phase-4 New-1):** move the non-blocking `e.changed` send into `markDirty` itself, under no lock after the unlock (the existing pattern). Then delete the duplicate send in `ApplyRemote` (`engine.go:497-500`) and rely on `markDirty` everywhere; `replaySnapshot`'s mid-error path gains the nudge for free. `await` keeps its send (it registers and marks dirty in one critical section, then nudges; it can also simply call the same helper — either is fine, but only one send per write).

**Acceptance tests:** synctest test — `Open` over a fake port returning `LoadBackup` starts dirty and a flush produces a dump without any write; the same for `LoadPrompt`; a `LoadMain` with a discarded record starts clean (no dump until a write); an internal-package test calls `markDirty` on an idle clean engine and observes a dump at `Debounce` with no other activity (proves the nudge); a stand test proves `SnapshotCreate` immediately after a backup start succeeds and the snapshot decodes to the loaded records; the existing `TestEngine_redundantDirty` (equal-version redelivery leaves a clean engine clean) must stay green.

### D-05 (PERSIST-05) — a bounded retry ladder, one ERROR per attempt, class counters

**Backoff design (constants are Claude's discretion; these are the recommended values):**

```go
const (
	defaultDebounce   = 50 * time.Millisecond // existing
	defaultMaxWait    = time.Second           // existing
	defaultRetryBase  = 100 * time.Millisecond
	defaultRetryCap   = 5 * time.Second
)
```

- State: `retryDelay time.Duration` in the scheduler, zero while healthy.
- On a failed `runDump`: `retryDelay = next(retryDelay)` (first failure 100 ms, then 200, 400, 800, 1.6 s, 3.2 s, 5 s cap); on success `retryDelay = 0`.
- Scheduling: while `retryDelay > 0`, the next attempt is armed at `failureInstant + retryDelay` through the injected `After`, and the debounce/deadline timers are not armed (they would otherwise fire earlier: `MaxWait` is 1 s < a 5 s cap and would defeat the ladder). New writes during backoff still register waiters and are covered by the next attempt; if the wait exceeds `ConfirmTimeout` they return the D-07 unconfirmed outcome. Recovery (a successful attempt) clears the ladder and immediately covers the waiters.
- One ERROR per failed attempt: `Config.OnDumpFailure func(err error, class port.FailureClass)` invoked once from `runDump` on the dumper goroutine; it must not block. `cmd` wires it to `slog.Error("Dump failed", "class", class.String(), "detail", err.Error())` — a static message plus attributes, per sloglint.
- Failure classes live in the port (stdlib-only) and are attached by the adapter, which is the only package allowed to know errnos:

```go
// port
type FailureClass uint8
const (
	FailureOther FailureClass = iota
	FailureDiskFull
	FailurePermission
)
func (c FailureClass) String() string // "other", "disk_full", "permission"
type ClassedError interface { error; FailureClass() FailureClass }
func FailureClassOf(err error) FailureClass // errors.As, default FailureOther

// persistfile (Save / SaveStream return classified errors)
switch {
case errors.Is(err, syscall.ENOSPC): class = port.FailureDiskFull
case errors.Is(err, syscall.EACCES), errors.Is(err, fs.ErrPermission): class = port.FailurePermission
}
```

- `DumpStats` gains `DiskFull`, `Permission` and `Unconfirmed` counters (keep `Errors` as the total so the existing metric and its tests stay meaningful). Metrics: keep `supkv_dumps_errors_total`; add `supkv_dumps_failures_total{class="disk_full"|"permission"|"other"}` and `supkv_dumps_unconfirmed_total`. Phase 7's OBSV-03 confirms the final catalog; this phase must publish a nonzero sample per class.
- Health: no code change; `gate.go:177` is the only `SetServingStatus(SERVING)` site. Document in OPERATIONS that a persistent dump failure leaves health `SERVING` and surfaces as `Unavailable` on writes (via `ErrDumpFailed`/`ErrUnconfirmed`) plus the ERROR log and counters.
- OPERATIONS text: the retry ladder, the cap, the "one ERROR per attempt" rule, the `ENOSPC`/`EACCES` runbook (free space / permissions; the node keeps serving reads and eventually retries), and the confirmation-timeout interaction.

**Acceptance tests:** synctest backoff test with a fake port that fails N times, asserting attempt instants at +100 ms, +300 ms, +700 ms … and the 5 s cap, then a success resets to the debounce cadence; `OnDumpFailure` called exactly once per failed attempt with the injected error and class; class classification unit tests (`ENOSPC`, `EACCES`, other); a stand test with the dump directory chmod'd to 0500 after the first write: Set returns `Unavailable`, stderr carries an ERROR naming permission denied, health stays `SERVING`, the attempt rate over a window is bounded (count the ERROR lines), chmod back → the next Set succeeds. Skip-as-root guard (modes do not bind).

### D-06 (PERSIST-06) — streaming encode through the existing writer

**Options:**

| Option | Peak live heap per cycle | Verdict |
|--------|--------------------------|---------|
| A. `SaveStream(fill func(io.Writer) error)` on the port; adapter supplies a buffered temp-file writer; engine's fill is `EncodeTo` | Records copy (1×) + one record line | **Recommended** |
| B. `SaveFrom(io.Reader)` with an `io.Pipe` goroutine | same, plus a goroutine and pipe error plumbing per dump | Rejected — needless concurrency in the confirmation path |
| C. Keep `Save([]byte)` but encode into a pooled buffer | ~1.4× encoded state still live | Rejected — does not meet "no full second copy" |

**Port and adapter shape:**

```go
// port
SaveStream(fill func(io.Writer) error) error

// persistfile
func (f *FileStore) SaveStream(fill func(io.Writer) error) error
// kept for byte fixtures and existing tests, implemented via SaveStream:
func (f *FileStore) Save(dump []byte) error

// fsops step 1 becomes:
write(dir string, fill func(io.Writer) error) error
```

The adapter creates the temp (0600, in `dir`), wraps it in a `bufio.Writer` (64 KiB), calls `fill(bufw)`, returns any fill error after removing the temp, else flushes/closes, fsyncs, rotates, renames, dir-syncs — the five TEST-03 steps unchanged. The engine:

```go
records := e.store.Records() // the one unavoidable copy, taken before any I/O
err := e.dumps.SaveStream(func(w io.Writer) error {
	return EncodeTo(w, records, e.valueMaxBytes)
})
```

**Memory model to document (measured, not guessed):**

- Live: one `map[string]storage.Record` deep copy (Data cloned under the store RLock) + one record line's base64 buffer (≤ ~1.34 × value limit) + the adapter's 64 KiB `bufio` + page cache (kernel, not heap).
- Transient allocation per cycle: the `Records()` clone (≈1× state) plus one encoded line's worth per record (≈1.4× state *total allocated* over the cycle, but each line is garbage immediately). The acceptance metric is **peak live**, which drops from ~5× to ~1×; the benchmark must report `B/op` and a sampled peak (see below) so the claim is measured.
- Residual, must be stated in the doc: `Records()` still holds the store's read lock while cloning every Data value (`store.go:369-382`), so writers are paused for O(state) at snapshot time (the review's 74 ms figure). Eliminating that pause needs a copy-on-write view and is out of scope for this phase.

**Measurement:** extend `BenchmarkDump10kKeys` (or add `BenchmarkDumpCycle`) to report `B/op`, `bytes/op` and a peak sample taken with `runtime.ReadMemStats` after each record write in a `discardWriter` run; keep thresholds out of `make test` (shared-runner noise) and rely on the documented model + the nightly `bench-smoke` job for evidence. Add a deterministic structural test: `EncodeTo` into a writer that records the largest single `Write` must never see a write larger than roughly one line (`maxDumpLineBytes(limit)`), proving the dump is not materialized in one buffer.

**Acceptance tests:** `EncodeTo` output byte-identical to `Encode` for the same map; the max-single-write bound; the port contract test that a fill error removes the temp and surfaces classified; benchmark numbers recorded in the phase summary/OPERATIONS.
### D-07 (PERSIST-08) — bounded confirmation, bounded flush, bounded shutdown

**Recommended defaults and sentinel:**

```go
// persist.Config
ConfirmTimeout time.Duration // 0 ⇒ 10s

// persist/errors.go
// ErrUnconfirmed is returned to a write whose covering dump did not complete
// within ConfirmTimeout: the record is installed in memory (and will be
// covered by a later successful dump) but is not yet durable. It is NOT a
// dump failure — the dump's outcome is unknown — and it must replicate like
// a dump-failed write (PROTO-08 totality).
ErrUnconfirmed = errors.New("write not confirmed before the confirmation timeout")
```

Ten seconds is chosen to sit far above the frozen blocking test's 200 ms window and above the documented dump worst case (`MaxWait` 1 s + dump time) while still bounding a stuck fsync to a client-visible `Unavailable` within one RPC wait. It stays a `persist.Config` field, not a CLI flag, to avoid pre-empting Phase 7's configuration catalog; the plan should note the option to expose `--confirm-timeout` later (Open Question 3).

**Waiter claim protocol (race-free, no double close):** add a `claimed bool` to `waiter`, guarded by `Engine.mu`. Whoever claims first writes `err` and owns the close:

- `runDump`/`failWaiters`/`sweepClosed`: under the mutex (at capture for the captured slice, again for the current slice), set `claimed = true` on every unclaimed waiter, set its error, collect it, then close outside the lock.
- `await`: register, nudge, then `select { case <-w.done: return w.err; case <-cfg.After(e.cfg.ConfirmTimeout): }`; on the timeout branch lock and
  - if `w.claimed` (a dump resolved it concurrently) unlock, `<-w.done`, return `w.err`;
  - else set `claimed = true`, `w.err = ErrUnconfirmed`, remove the waiter from `e.waiters` (linear scan; the slice is bounded by in-flight writes), unlock, count `Unconfirmed`, return `ErrUnconfirmed`.
  The removal is what keeps memory bounded under a permanently hung dumper; the record itself is already dirty in the store.

**Flush deadline:** `flush()` must bound both waits (the `flushCh` send and the reply) with the same deadline. Add:

```go
func (e *Engine) Flush() error                        // FlushTimeout(0) → ConfirmTimeout
func (e *Engine) FlushTimeout(d time.Duration) error  // bounded; ErrUnconfirmed on expiry
```

`Snapshot` and `Restore` use `flush()` (hence `ConfirmTimeout`); a timed-out flush must return `ErrUnconfirmed` (wrapped with context) and, for `Restore`, must still return the partial replay list so the caller's totality notify covers installed records (the existing IN-04 contract).

**Shutdown (`cmd`):** `engine.FlushTimeout(shutdownGrace)` and then a bounded wait for `engineDone`:

```go
err := node.engine.FlushTimeout(shutdownGrace)
node.engineCancel()
select {
case runErr := <-node.engineDone:
	...
case <-time.After(shutdownGrace):
	slog.Warn("Engine did not stop within the shutdown grace", "grace", shutdownGrace.String())
	// abandon the goroutine; the process is exiting
}
```

A timed-out flush returns an error → the existing exit code 1 (`main.go:266-269`); OBSV-04 (Phase 7) may refine the taxonomy, but Phase 5 must not hang. Document that abandoning a goroutine blocked in a kernel fsync at process exit is the deliberate choice.

**RPC ripple (must not be missed):** a timed-out write is installed and unconfirmed, so it must replicate. Extend the two totality branches (`grpcapi/server.go:217-220`, `:263-266`) to `if errors.Is(err, persist.ErrDumpFailed) || errors.Is(err, persist.ErrUnconfirmed)`, and add `case errors.Is(err, persist.ErrUnconfirmed): return status.Error(codes.Unavailable, persist.ErrUnconfirmed.Error())` to `mapError`. Without this, a timed-out write would never be announced — a PROTO-08 regression.

**Acceptance tests (synctest, fake port; no real I/O in a bubble):** a port whose `SaveStream` never completes ⇒ `Set` returns `ErrUnconfirmed` at exactly the virtual timeout, the waiter is removed (no growth over many writes), `Unconfirmed` == number of writes; a port that completes after the timeout ⇒ the later success resolves nothing (the waiter is gone) and the record is still dumped; `Snapshot` with a hung port returns `ErrUnconfirmed`; `FlushTimeout` with a hung port returns `ErrUnconfirmed`; the frozen `TestEngine_blockingSaveHoldsSet` stays green (release at 200 ms << 10 s); an RPC-level test (or the existing harness) proves `ErrUnconfirmed` → `Unavailable` and the notify call.

### D-08 (PERSIST-09) — startup sweep, parent sync, directory-fsync tolerance

**Sweep (age-bounded, startup only, visible):** perform it at the start of `Load`, before the fallback chain, over both the dump directory and `<dump>.snapshots` (when it exists): remove files whose base name matches `dump-*` and whose `ModTime` is older than `defaultOrphanTempAge` (recommend `time.Hour`). Report `Swept` and `SweepErr` in `LoadResult`; `cmd` logs a WARN when `SweepErr != nil` and an INFO with the count when `Swept > 0`. Never touch `.prev`, `.corrupt-*` (different prefixes) or the configured main path. The age bound protects a concurrently running second instance's in-flight temp (a misconfiguration, but the sweep must not be the thing that corrupts it).

**New-directory parent sync:** keep `MkdirAll` in `saveAtomically` but wrap it in an `ensureDir(ops, dir)` pre-step (before the five recorded steps): when `dir` did not exist, `MkdirAll` it and then call the existing `ops.syncDir(filepath.Dir(dir))`. Two consequences:
- The five-step TEST-03 order is unchanged for an existing directory — the existing order tests use `filepath.Join(t.TempDir(), "dump.json")`, whose directory already exists, so they observe exactly write → fsync → rotate → rename → dir-fsync.
- The parent sync is observable through the *existing* recording fake as a `syncDir` call with the parent argument before `write`, so no new fsops method is needed; a dedicated test uses a nested, non-existent path and asserts `[syncDir(parent), write, fsync, rotate, rename, syncDir(dir)]`.

A failure to sync the parent aborts before any temp exists (step-1 hygiene, nothing to clean). The snapshots directory created by `CreateSnapshot` gets the same treatment. Note `filepath.Dir("dump.json") == "."` — syncing `"."` is valid and cheap.

**Directory-fsync tolerance:** `syncDir` wraps its `d.Sync()` in a helper that returns nil for `errors.Is(err, syscall.ENOTSUP)` and `errors.Is(err, syscall.EINVAL)`, documented as "this filesystem cannot fsync a directory; the rename's durability is then at the filesystem's discretion" — and **only** for the directory step. File-data fsync must keep every error (ENOSPC on data is not survivable silently). EINTR needs no manual loop: `os.File.Sync` already routes through `ignoringEINTR` on both Linux and darwin (GOROOT evidence above); the helper's doc must say so, citing the stdlib, rather than re-implementing a retry loop. `fsync_darwin.go` is already deleted; the phase's remaining P-09 obligation is to keep the darwin build green and to correct the registry row/comment if the "only" claim changes (the current fsops comment is accurate — do not rewrite it as if new work were done).

**Acceptance tests:** sweep removes an old `dump-*`, keeps a young one, ignores `.prev`/`.corrupt-*` and the snapshots' non-temp files; sweep error is reported; a fresh nested dump path creates the directory and the recording fake observes the parent `syncDir` before `write` (arguments distinguish parent vs dump dir); an existing directory triggers no parent sync; `syncDirTolerant` returns nil for ENOTSUP/EINVAL and the error for anything else (pure helper, table test with synthetic errnos).

### D-09 (PERSIST-10) — the skip class is counted, logged once, and left on disk

Implement D-03's `DecodeStats` with the four classes. `Open` copies the accepted generation's stats into `LoadStats.Decoded`; `cmd` adds one line:

```go
if n := stats.Decoded.ValueLimit; n > 0 {
	slog.Warn("Dumped records skipped by the configured value limit",
		"count", n, "value_max_bytes", cfg.valueMaxBytes)
}
```

plus a metric `supkv_dump_records_discarded_total{class="value_limit"|"checksum"|"version"|"malformed"}` (all four samples always rendered). The existing INFO for the remaining classes stays for continuity.

**Do not mark the engine dirty on skips.** A clean engine means the next dump happens only on a write or shutdown; the skipped record's line survives in main until the first dump rotates it to `.prev` and the second displaces it — "the normal rotation discipline" of D-09, and the longest evidence window the design allows. A test must pin this: after a shrunk-limit load with zero writes, the main file still contains the record's key; after one write the record is in `.prev`; after two dumps it is gone.

**Test change to expect:** `TestEngine_configShrunkValueLimits` currently asserts `discarded == 0` ("the skip is at merge, not codec"). With the codec limit configured it becomes `discarded == 1` and `LoadStats.Decoded.ValueLimit == 1`; the `Get("fat") == ErrNotFound` assertions stay. `mergeRecords`'s `ErrValueTooLarge` skip becomes an unreachable belt (keep it, update the doc to say it is a belt over the store's invariant).

### D-10 (PERSIST-11) — five internal roles, public signatures frozen

**Split shape (internal types, no new interfaces, no exported names beyond `Engine`):**

| Role | Contents today | Recommended home |
|------|----------------|------------------|
| Loader / composition root | `Open`, `mergeRecords`, decode closure, load report construction | `loader.go` (`openEngine` builds store+engine and attaches `LoadStats`) |
| Store facade | `Get`, `Lookup`, `Version`, `Versions`, `Observe`, `KeyCount`, `Generation`, `Records`, `ApplyRemote`, `Set`, `Delete`, `SetIfAbsent` | `facade.go` — thin forwards and the write-then-await shape |
| Dump scheduler | `Run`, `runDump`, `flushDump`, `rearm`, timers, backoff, `recordDump`, `DumpStats` | `scheduler.go` with an internal `scheduler` value |
| Confirmation accounting | `waiter`, `await`, `confirmWaiters`, `failWaiters`, `sweepClosed`, `Unconfirmed`/class counters | `confirm.go` with an internal `confirmSet` (own mutex, waiters, dirty/oldest, `changed`, closed, `runDone`) |
| Snapshot orchestration | `Snapshot`, `Restore`, `replaySnapshot`, `restoreChanges`, `replayRecord` | `snapshot.go` |

`Engine` becomes the composition of those values and keeps every current method signature exactly (the frozen P-01 engine test and the existing `persist`, `grpcapi`, `udprepl`, `cmd` test suites compile against them). The split is behaviour-preserving except for the changes decided above; the plan should land it as a separate task **after** the behavioural waves so its diff is reviewable as a pure move. Do not introduce interfaces for the roles (`iface` lint) and keep each file's functions within `funlen` (move helpers, don't compress).

**Acceptance:** the whole existing suite green; `Engine`'s exported API unchanged (a compile-time assertion file is unnecessary — the tests are the assertion); `LoadStats`, `FlushTimeout` are additive; the port's new methods are the only interface change.

### D-11 (PERSIST-12) — `Config.Now` / `Config.After`, and the sim seam

**Seam shape:**

```go
type Config struct {
	Debounce       time.Duration
	MaxWait        time.Duration
	ConfirmTimeout time.Duration
	Now            func() time.Time                 // nil ⇒ time.Now
	After          func(time.Duration) <-chan time.Time // nil ⇒ time.After
}
```

`normalized()` fills the nil funcs and the timeout (no package-level globals — `gochecknoglobals`). Replace every `time.Now`/`time.NewTimer`/`time.Until`/`time.Since` in the package:

- `Run`'s timer variables become `<-chan time.Time`; `rearm` arms them via `cfg.After`; `stopTimer` is deleted (a channel cannot be stopped). Abandoning an armed `time.After` channel when re-arming is safe — nobody selects on the old channel afterwards — and costs one timer per re-arm, bounded by the loop's wake-ups; document it in the memory model.
- `runDump` measures with `cfg.Now()`; `failWaiters`/`await`/the backoff use `cfg.Now()`; the confirmation timeout uses `cfg.After`.
- Semantics unchanged: the deadline is still armed from the clean→dirty transition and never reset by later writes; the debounce still resets on every change.

**Sim adapter (in the test):**

```go
func virtualAfter(sched *simnet.Scheduler) func(time.Duration) <-chan time.Time {
	return func(d time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		sched.Schedule(sched.Now().Add(d), func() { ch <- sched.Now() })
		return ch
	}
}
```

**Additive simnet seam needed:** `Cluster` re-Sets a key in `Write` (`simnet.go:739-775`), so an engine-driven write would be stamped twice. Add `func (c *Cluster) NotifyInstalled(i int, key string) (storage.Record, error)`: `Lookup` the already-installed record, run the same `NotifyLocalWrite` + `route` + write-log capture as `Write`, but never `Set`. `Cluster.Store(i)` already returns the store pointer, so the engine can share it: `persist.New(cluster.Store(0), fakeDump, persist.Config{Now: sched.Now, After: virtualAfter(sched), ...})`. This is test-harness-only, additive, and follows the Phase-2 additive-extension precedent.

**Sim test (PERSIST-12's acceptance), outline:**
1. Two-node `Cluster` with a `Policy{AnnounceDelay: …}` (the existing dump-latency model), deterministic seed.
2. Give node 0 a `persist.Engine` over `cluster.Store(0)` and a fake `DumpStore` that completes `SaveStream` at `sched.Now()+dumpDelay` (or fails on demand), recording attempt instants.
3. `eng.Set(key, value)` in a goroutine; drive `sched.RunUntil` past `dumpDelay`; assert the confirmation arrives at the virtual dump instant, then `cluster.NotifyInstalled(0, key)` and assert the peer's store holds the record after the network delay — the announcement is not delayed by the dump beyond the modeled `AnnounceDelay`.
4. Fail the dump for a window: assert `Set` returns `ErrDumpFailed`, the notification still reaches the peer (mirroring the RPC totality branch), and the fake's attempt instants follow the backoff ladder.
5. The skip-count path: build a fake `Load` result for a "restarted" node whose dump contains an over-limit record with a shrunk `storage.Config`; assert `LoadStats.Decoded.ValueLimit == 1` and that the peer still converges.
6. Determinism: run the same scenario twice with the same seed and assert identical attempt instants and identical `TraceDigest`-style ordering (no rng in the tie-break or backoff).

**Fallback if the goroutine/virtual-time interleaving proves brittle:** keep steps 1–2 and 4–5 (which only need the scheduler-driven engine loop) and move the pure engine timing assertions to `synctest`; PERSIST-12's acceptance is satisfied by the simulator exercising the interaction, not by every assertion living there.

### D-12 (PERSIST-07) — deterministic equal-version resolution, end to end

**Law:** the order is `(version, content)`, lexicographic, where content is `(Deleted, Data)`:

1. Different versions → the greater full 16-byte version wins (unchanged).
2. Equal versions, identical `Deleted` and identical `Data` bytes → full no-op (idempotent redelivery; no conflict, no dump).
3. Equal versions, different content → the record with `Deleted == true` wins; if both are live or both are tombstones, the lexicographically greater `Data` (Go `bytes.Compare`) wins. Report a `Conflict` in both directions with the (equal) versions; the new `changed` result is authoritative for "the incoming record became the stored record".

**Why tombstone-wins:** it is the standard remove-wins choice, it keeps a confirmed delete durable against a resurrected same-version value, and it is explainable in the ADR. The honest consequence to record: when the same version carries a delete and a value, one of two confirmed operations is lost either way; remove-wins loses the value. This is a pathological clock-rollback/lost-disk state, not a normal conflict.

**Store contract (avoid a 62-call-site signature ripple):**

```go
// New: the merge carrying the installation decision.
func (s *Store) MergeChanged(key string, rec Record) (conflicts []Conflict, changed bool, drift time.Duration, err error)
// Merge stays the three-result wrapper every existing caller uses.
func (s *Store) Merge(key string, rec Record) ([]Conflict, time.Duration, error)
// The content equality production needs (version, deleted, data).
func RecordsEqual(a, b Record) bool
```

`join` gains the tie-break; `install` already bumps the generation and fires `OnChange`, so a tie-break install is visible to Watch and the listing generation without extra wiring. `Engine.ApplyRemote` switches its `displaced` heuristic to `MergeChanged`'s `changed` (its signature is unchanged).

**Replication-core branch (the part that makes the law reachable):**

```go
// core.applyRemote, replacing the cmp == 0 early Ack
case cmp == 0:
	held, ok := c.store.Lookup(msg.Key)
	if ok && storage.RecordsEqual(held, msg.Record) {
		out.Unicast(source, port.Ack{Key: msg.Key, Version: held}) // identical content: the old no-op
		return
	}
	// fall through to ApplyRemote; the store decides
```
after the merge, decide the reply from the stored record:
```go
after, _ := c.store.Lookup(msg.Key)
if storage.RecordsEqual(after, msg.Record) {
	out.Unicast(source, port.Ack{Key: msg.Key, Version: msg.Record.Version}) // incoming won
} else {
	c.enqueueResponse(now, source, egressDataReplication, msg.Key) // stored won: answer like PROTO-03, no Ack
}
```
The stale branch (`cmp > 0`) and all other paths are unchanged. Clock updates still happen inside `ApplyRemote`; `mergeStats.Conflicts` counts the tie-break conflict; the tracked-write supersession loop is untouched (equal versions never superseded before, and still do not — the tracked version is still the stored version).

**ADR 0012 outline (`docs/adr/0012-equal-version-tiebreak.md`, matching 0005–0011's slim-MADR shape):**
- Frontmatter: `adr: 0012`, `review-decision: "P-07 (equal-version tie-break)"`, `informs: [PERSIST-07]`, `date: 2026-11-22`, `status: accepted`.
- Context: the Spec's random tail (L243) was replaced by zero filler (S-HLC-07, accepted deviation), which makes same-version-different-content reachable after disk loss + clock rollback; v1.0 silently no-ops it on merge and Acks it on the wire; deterministic replay and byte-identical dumps are project invariants.
- Decision: the `(version, content)` law above; where it applies (store join + replication equal-version branch); what is explicitly *not* changed (version layout, the zero filler, per-prefix delete semantics — PFXD-01 remains v2).
- Consequences: convergence under the pathological state; one of two same-version operations is dropped; the wire answers with the stored record when it wins; conflict accounting counts the tie-break; `S-ALG-REP-03` amended and `S-HLC-07` links to this ADR.
- Links: REVIEW P-07; SPEC-DEVIATIONS S-HLC-07, S-ALG-REP-03; informs PERSIST-07.

**Tests:** store-level table tests for every branch of the law (live-vs-live both directions, tombstone-vs-live both directions, identical content), commutativity/associativity/idempotence property over `testutil.RandRecord` *plus deliberately equal versions with different data drawn from the small space*, `MergeChanged` reporting, `TestMerge_losersProperty` rewritten to the reference join (allow equal-version conflicts; assert the content winner), `store_test.go:510-530` rewritten, `TestMerge_duplicateDelivery` untouched, an internal core test for the equal-version wire branch (incoming wins → Ack only; stored wins → exactly one unicast DataReplication of the stored record and no Ack), an `ApplyRemote` dirty test (equal-version content displacement dirties; identical redelivery does not), and a same-seed sim replay check.
## Architecture Patterns

### System Architecture Diagram (target data flow)

```
                          STARTUP
                             |
                    cmd: finalizeConfig
              (value-max-bytes >0 and <= datagram ceiling)
                             |
                 +-----------+-----------+
                 |                       |
        persistfile.Load(now, parse)     (config error -> exit 2)
                 |
     sweep orphan dump-* (age-bounded, both dirs)
                 |
        for main then .prev:
           read -> absent? -> next
           parse oracle (persist.Decode at the configured limit)
              ok -> remember generation, stop
              fail -> record {path, cause}
                      preserve: rename -> <path>.corrupt-<UTC>[-N]
                      sync dir (tolerate ENOTSUP/EINVAL)
                 |
        LoadResult{Outcome Prompt|Main|Backup|Unusable, Failures, Dump}
                 |
        persist.Open: decode accepted bytes -> DecodeStats
                 |        -> clock seeded from FULL watermark
                 |        -> storage.NewStore(nodeID, clk, storeCfg)
                 |        -> merge records (oversize belt)
                 |        -> Engine{LoadStats, dirty if Prompt/Backup/Unusable}
                 |
        cmd: one WARN (Backup) / one ERROR (Unusable) / nothing (Prompt/Main)
             + load-outcome and discarded-class counters

                           WRITE PATH
   gRPC Set/Delete --> Engine.Set --> Store.Set (value limit, fresh version)
                           |
                     await(): register waiter + markDirty + nudge changed
                           |
                    select on waiter.done vs ConfirmTimeout
                    /                         \
              resolved                     timeout
        nil: on disk                    ErrUnconfirmed (+1 counter)
        ErrDumpFailed: dump failed      record stays in memory/dirty
                    \                         /
                     gRPC: notify replication on failure/timeout (PROTO-08)

                           DUMPER (single goroutine)
   Run: select changed | debounce | deadline | retry | flush request
        |
        runDump: capture waiters; dirty=false
        |        records := Store.Records()            (one deep copy)
        |        dumps.SaveStream(func(w) {            (streaming)
        |            EncodeTo(w, records, valueLimit)  (one line at a time)
        |        })
        |            persistfile: temp(0600) -> bufio -> fill -> flush
        |                         -> fsync -> rotate -> rename -> dir fsync
        |        success: confirm waiters, retryDelay=0
        |        failure: fail waiters, classify (disk_full|permission|other),
        |                 OnDumpFailure (one ERROR), retryDelay=next(), re-arm
        |
        backoff armed: no debounce/deadline timers until it fires (cap 5s)

                       REPLICATION APPLY
   core.handleDataReplication --> equal version?
        no  -> ApplyRemote -> Store.MergeChanged -> dirty + nudge
        yes -> Lookup: identical content -> Ack, no dump
                      different -> Store.MergeChanged (content tie-break)
                                   -> incoming won: Ack
                                   -> stored won: unicast stored record, no Ack
```

### Recommended Project Structure (only the files this phase adds/moves)

```
internal/port/
├── dump.go          # LoadOutcome/Generation/LoadResult/GenerationFailure, DumpStore
│                    # (SaveStream, Load(now, parse), snapshots), FailureClass enum
internal/app/persist/
├── engine.go        # Engine struct, New, Flush/FlushTimeout/flush, Config
├── loader.go        # Open, load-report mapping, mergeRecords
├── facade.go        # store-facing methods (incl. ApplyRemote)
├── scheduler.go     # Run, runDump, rearm, backoff, timers, DumpStats
├── confirm.go       # waiter/confirmSet, await + timeout, confirm/fail/sweep
├── snapshot.go      # Snapshot, Restore, replay helpers
├── codec.go         # Encode/EncodeTo/Decode, header+record_count, DecodeStats
├── errors.go        # + ErrUnconfirmed
└── doc.go           # updated contract text (timers, timeout, load stats)
internal/persistfile/
├── dumpfile.go      # SaveStream, Load(now, parse)->LoadResult, preservation, sweep
├── fsops.go         # write(dir, fill), mkdir-synced, syncDirTolerant
├── saveerror.go     # errno classification -> port.FailureClass
└── snapshot.go      # unchanged copy path (uses the new saveAtomically)
cmd/sup-kv/main.go   # load logging, failure callback wiring, FlushTimeout, value check
docs/adr/0012-equal-version-tiebreak.md
docs/OPERATIONS.md   # retry ladder, health, load outcomes, confirm timeout, memory model
docs/SPEC-DEVIATIONS.md
```

### Pattern 1: adapter classifies, core reports, binary narrates

**What:** the file adapter is the only place that knows errnos and file existence; the port carries a small stdlib-only vocabulary (`LoadOutcome`, `FailureClass`, `LoadResult`); the engine stores a report; `cmd` owns the slog lines and the metrics wiring.
**When to use:** every new load/failure signal in this phase.
**Why:** depguard forbids `syscall` in `internal/app`, and the project's established rule is "the core is silent; adapters log" (`repl.Config.Log`/`OnDiscard` precedent). It also keeps one WARN per class: only `cmd` decides, from one report.

### Pattern 2: line-oriented codec with a completeness census

**What:** split the file into bounded lines, parse each independently, count lines, compare against the header's `record_count`, and treat only whole-generation conditions (header unreadable, unsupported version, unterminated unparseable tail, too few lines, overlong line) as generation faults.
**When to use:** any future field added to the dump must follow the same additive rule (unknown fields ignored, absence = legacy).
**Why:** it is the only shape that satisfies both "truncation is detected" and "one corrupt record loses one record" (verified probe: `json.Decoder` cannot recover from syntax errors).

### Pattern 3: injected now/after loop

**What:** the scheduler loop owns no wall clock; `Config.Now` and `Config.After` are injected, defaulting to `time.Now`/`time.After`.
**When to use:** every timer in `internal/app/persist` (debounce, deadline, backoff, confirmation, flush).
**Why:** mirrors the repl core's "time arrives from outside" discipline and lets `simnet.Scheduler` drive the engine deterministically; `synctest` also intercepts the defaults, so existing tests need no change.

### Pattern 4: single-claim waiter resolution

**What:** a waiter is resolved exactly once by whichever party claims it first under the engine mutex (dump success, dump failure, shutdown sweep, or the awaiting caller's timeout); the claimer writes the error and closes the channel.
**When to use:** the confirmation timeout and every existing fan-out path.
**Why:** channels cannot be closed twice; a timeout must not leave an unreachable waiter that later panics on close, and must not let a late dump write `w.err` concurrently with the caller reading it.

### Anti-Patterns to Avoid

- **Storing the limit in `persist` as a package constant or a second config field.** It is the exact class of bug P-01 is.
- **A second `DumpStore` result type in `persist`.** The port's `LoadResult` is the seam; the engine's `LoadStats` is a report, not a parallel contract.
- **Preserving corrupt files *after* the first save instead of at load.** By then `.prev` is already gone.
- **A `//nolint` for a new lint finding.** Fix the shape or record the deviation in AGENTS.md (three deviations are already enumerated; do not quietly add a fourth).
- **Marking dirty unconditionally on every load.** It rewrites a healthy file on every restart and shortens the PERSIST-10 evidence window.
- **Using `json.Decoder` for records with a per-record error-recovery expectation.** Verified impossible for syntax faults.
- **Buffering the dump in a second `bytes.Buffer` "for simplicity".** PERSIST-06's acceptance is the streaming property.
- **A random tie-break tail.** Breaks byte-identical dumps and same-seed replay.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| EINTR handling on fsync | A retry loop around `syscall.Fsync` | `os.File.Sync` (already `ignoringEINTR` on Linux and darwin; darwin falls back from `F_FULLFSYNC` on ENOTSUP) | GOROOT-verified; a hand-rolled loop is where the deleted `fsync_darwin.go` went wrong |
| Temp file naming and 0600 mode | `fmt.Sprintf("dump-%d", pid)` | `os.CreateTemp(dir, "dump-*")` | Collision-free, 0600, in the target directory (no EXDEV) |
| Atomic replacement | Copy-then-truncate | `os.Rename` (replaces an existing non-directory target) | POSIX atomic replace; documented in `os.Rename` |
| Base64 / JSON spelling of bytes | Hand-written encoder | `encoding/json` with `[]byte` (base64) and `hex` for the version | The format is already pinned by tests and docs; hand-rolling invites drift |
| Errno classification | String matching on error text | `errors.Is(err, syscall.ENOSPC/EACCES)` + `port.ClassedError` | Wrapping (`*os.PathError`, `*fs.PathError`) preserves the errno for `errors.Is` |
| Writing numbers twice | Two config fields for one limit | `storage.ValueMaxBytes` + `Config.ValueLimit()` | The three-copy bug |
| A test clock | A new fake clock type per test | `testing/synctest` for unit timing; `Config.Now/After` + `simnet.Scheduler` for the sim | Two mechanisms already exist and are proven |
| Line reading with unbounded memory | `bufio.Scanner` default buffer or `ReadString` on hostile input | `bufio.NewReaderSize` + `ReadSlice`, bound from the configured limit | A crafted dump must not allocate a gigabyte line |
| Preservation name uniqueness | `os.Rename` and hope | probe free names (`Lstat` + `-N` suffix) before renaming | `os.Rename` silently replaces an existing target; evidence would be lost |

## Runtime State Inventory

This phase changes on-disk state and metric family names, so the inventory applies even though it is not a rename.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | Main dump and `<dump>.prev` files in the v1.0 grammar (header `format_version:1` + record lines, no count); snapshots under `<dump>.snapshots/<name>` in the same grammar | **No migration.** The reader accepts the legacy grammar; the first successful dump rewrites the main in the new grammar (additive `record_count`); `.prev` and snapshots may remain legacy until overwritten. A code edit, not a data migration |
| Stored data (new artifacts) | `<dump>.corrupt-<UTC>[-N]` files created at load | Documented; never rotated or swept; operator deletes manually after inspection (OPERATIONS) |
| Stored data (garbage) | Orphan `dump-*` temp files from crashed saves, in the dump dir and `<dump>.snapshots` | Swept at startup when older than the age bound; reported |
| Live service config | None — sup-kv has no external service state; flags/env only | None |
| OS-registered state | None found (no systemd unit shipped in-repo yet; no pm2/launchd/task registrations) | None |
| Secrets/env vars | None new; `SUPKV_VALUE_MAX_BYTES` semantics tighten (non-positive now exits 2) | Document in OPERATIONS; DOCS-02 (Phase 9) owns the flag table |
| Build artifacts | None; no generated code touched | None |
| Observability state | Existing family `supkv_dumps_errors_total` kept; new families are additive (`supkv_dumps_failures_total{class}`, `supkv_dumps_unconfirmed_total`, `supkv_dump_load_total{outcome}`, `supkv_dump_records_discarded_total{class}`) | External dashboards referencing existing names keep working; new alerts are Phase 7/9 documentation work |
| Simulator/model state | Models use `storage.Store`, so the tie-break reaches them automatically; no stored sim state | None |

**Canonical question — after every file is updated, what runtime system still holds the old string/format?** Only the dump files themselves, and they are read-only-compatible; the next dump replaces the main, and `.prev`/snapshots age out naturally.

## Common Pitfalls

### Pitfall 1: changing `persist.Open`'s signature breaks a frozen test that must not be edited

**What goes wrong:** the P-01 engine test destructures `(eng, discarded)` (`internal/persistfile/red_value_limit_test.go:51-57`). Any "cleaner" `Open` returning a report struct fails to compile the frozen test, and the red gate cannot flip.
**Why:** the test is frozen by D-01/plan 02-12's contract ("only the expectation-table row changes").
**How to avoid:** keep `Open(...) (*Engine, int)`; expose `LoadStats()` on the engine. Treat this as a hard API constraint in every task.
**Warning sign:** a plan task says "change Open to return LoadReport".

### Pitfall 2: the shrunken-limit test asserts the old diagnostic split

**What goes wrong:** `TestEngine_configShrunkValueLimits` asserts `discarded == 0` "the skip is at merge, not codec" (`engine_test.go:1707-1709`). With the codec limit configured, the correct value is 1 and the class is `ValueLimit`.
**How to avoid:** update the assertion and add the class assertion; do not "fix" the codec to keep the old test green.
**Warning sign:** a task that leaves the codec using a default when the store says 16.

### Pitfall 3: the old equal-version tests pin exactly the behaviour P-07 replaces

**What goes wrong:** `store_test.go:510-530` seeds `"a"` and merges the same version with `"x"`, expecting no change and no conflicts; `TestMerge_losersProperty` expects equal versions to yield no conflict and no conflict ever pairing equal versions. Both fail under the new law.
**How to avoid:** rewrite both to the reference `(version, content)` join, keep `TestMerge_duplicateDelivery` (identical content) unchanged, and state the equal-version-conflict semantics in the `Conflict` doc.
**Warning sign:** someone "preserves backward compatibility" by only applying the tie-break to tombstones.

### Pitfall 4: the tie-break is dead code unless the replication core stops short-circuiting equal versions

**What goes wrong:** `core.applyRemote` Acks any equal version before the store (`core.go:1483-1487`). A store-only tie-break passes all store tests and diverges in production/sim.
**How to avoid:** implement the core branch (content-aware `Lookup`; Acknowledge only on identical content or an incoming win; unicast-reply on a stored win) and test it at the core level with two records sharing a version.
**Warning sign:** a plan that lists only `storage/store.go` under PERSIST-07.

### Pitfall 5: `ApplyRemote`'s `Winner == rec.Version` test is ambiguous at equal versions

**What goes wrong:** at equal versions `Winner` and `rec.Version` are the same bytes whichever side won, so the engine would dirty on a rejected tie-break (redundant dump) or skip dirtying on a win depending on the branch.
**How to avoid:** use `MergeChanged`'s `changed` result inside `ApplyRemote`.
**Warning sign:** a test that passes because both sides have the same content.

### Pitfall 6: the `DumpStore` doubles ripple further than expected

**What goes wrong:** changing the interface breaks four implementations that must all be updated in the same wave: `internal/persistfile/dumpfile.go`, `internal/app/persist/engine_test.go` (`fakeDumpPort`), `internal/app/persist/engine_blocking_test.go` (`blockingDumpPort`), `internal/grpcapi/server_test.go` (`fakeDumpStore`). The copies under `docs/reviews/.../evidence/` are separate historical modules and are not built.
**How to avoid:** change the port and all four in one task; give the fakes a shared helper style (a `recordingWriter` fill) so `SaveStream` captures bytes for `Decode` assertions.
**Warning sign:** `go build ./...` passes while `go test ./...` fails to compile a test package.

### Pitfall 7: the datagram ceiling bounds how far the value limit can be tested

**What goes wrong:** `--value-max-bytes=64000` is accepted; the exact largest accepted value is 64 408; 64 409 exits 2 (`main_test.go:509-577`, `startupBoundaryValue`/`startupBoundaryOverValue`). A test that raises the limit to probe the codec's limit will hit the ceiling first.
**How to avoid:** test the single-source limit at 64 000 (accepted, the frozen value) and at small values (8, 16) where the ceiling is irrelevant; never raise `--value-max-bytes` past the boundary.
**Warning sign:** a test failing with `errDatagramCeiling` instead of the expected value-limit behaviour.

### Pitfall 8: a timed-out write must still replicate

**What goes wrong:** `ErrUnconfirmed` is a new error class; the gRPC handlers only notify on `ErrDumpFailed`, so a timed-out write would be invisible to peers although it is in memory — violating PROTO-08's totality.
**How to avoid:** extend both handler branches and `mapError` (D-07). Add an RPC-level assertion.
**Warning sign:** the `Set` failure-path test lists only `ErrDumpFailed`.

### Pitfall 9: the strict lint config rejects idiomatic new code

**What goes wrong:** `noinlineerr` forbids `if err := f(); err != nil`; `mnd` flags bare `10*time.Second`/`100`; `gochecknoglobals` flags package-level default funcs; `err113` forbids dynamic errors; `recvcheck` requires consistent receivers; `testpackage` requires the `_internal_test.go` exemption pattern for white-box tests; `paralleltest` requires `t.Parallel()` or a documented nolint; `godot` wants terminal punctuation on comments.
**How to avoid:** declare every duration/threshold as a named constant near the code that uses it; split `err := ...; if err != nil`; use sentinels plus `%w`; add new white-box tests with the existing `//nolint:testpackage // reason` shape; run `make lint` at the end of every wave, not at the end of the phase.
**Warning sign:** a plan task that budgets lint at the end.

### Pitfall 10: classification must not leak `syscall` into the core

**What goes wrong:** importing `syscall` or `os` into `internal/app/persist` fails depguard (`sup-kv-app-no-io`) and, worse, would be a quiet architecture change.
**How to avoid:** the port enum + `ClassedError`; classify in `persistfile`; the engine calls `port.FailureClassOf`.
**Warning sign:** a `.golangci.yml` diff in the plan.

### Pitfall 11: encoding an oversize fixture through the limit-checked encoder

**What goes wrong:** codec tests build an oversize record *via* `Encode` to test decode-side discard; with an encode-side limit error the fixture cannot be built.
**How to avoid:** call `Encode(records, limit+oversizeBy)` for the fixture and `Decode(dump, limit)` in the assertion; keep the encode-side check as a tripwire (tested separately with a direct oversize map).
**Warning sign:** a test that skips the oversize case because "Encode refuses".

### Pitfall 12: `json.Decoder` cannot recover from a syntax fault

**What goes wrong:** keeping the streaming decoder and adding a count still collapses the whole generation on a syntax-corrupt line (verified). P-03's "one corrupted record" stays unfixed.
**How to avoid:** line-oriented `ReadSlice` + `json.Unmarshal`.
**Warning sign:** `Decode` still contains `json.NewDecoder`.

### Pitfall 13: bounded line reading, not bounded ambition

**What goes wrong:** `bufio.Scanner`'s default 64 KiB token limit silently stops on a 61 440-byte value's base64 line (≈81.9 KiB) — a legal dump becomes unreadable; `ReadString` with no bound allocates a hostile line.
**How to avoid:** `bufio.NewReaderSize` sized from `maxDumpLineBytes(limit)`; `ErrBufferFull` ⇒ generation error; a boundary test at exactly the limit and one byte over.
**Warning sign:** a default `bufio.Scanner`.

### Pitfall 14: the writer pause is not fixed by streaming

**What goes wrong:** `Records()` clones every value under the store's RLock; the 74 ms pause remains, and a claim that P-06 fixed it would be false.
**How to avoid:** document the residual in the memory model; measure the lock hold only if a test is added; do not add a COW store in this phase.
**Warning sign:** OPERATIONS claiming writers are never paused.

### Pitfall 15: `os.Rename` over an existing `.corrupt-*` destroys evidence

**What goes wrong:** two loads in the same second (tests, or an operator restart loop) produce the same stamp; the second rename silently replaces the first preserved file.
**How to avoid:** probe for a free name and append `-1`, `-2`; assert two preserves of the same source both survive in a test (with an injected fixed `now`).
**Warning sign:** preservation tests all use distinct seconds.

### Pitfall 16: the sweep deletes live files

**What goes wrong:** matching `dump*` instead of `dump-*` would delete the main dump (`dump.json`) or `.prev`; deleting young temps can break a concurrent save.
**How to avoid:** match the exact `tempPattern` base-name prefix (`dump-`), skip anything not a regular file, enforce the age bound, and exclude the snapshots directory's non-temp files by the same rule.
**Warning sign:** a sweep test only with one temp file.

### Pitfall 17: `time.After` channels cannot be stopped, and a stale one must not fire a dump

**What goes wrong:** with the channel seam, a re-arm creates a new channel; code that still selects on an old one can trigger an early dump; a loop that recreates timers every iteration can accumulate them.
**How to avoid:** keep the timer values in local variables and replace them on re-arm (the old channel is then unreachable), and only arm when the value is nil (the existing discipline). Assert the debounce/deadline behaviour at the virtual instants in a synctest test.
**Warning sign:** a `time.After` call inside the select statement itself (a new channel every iteration).

### Pitfall 18: real I/O inside a synctest bubble deadlocks

**What goes wrong:** the blocking test documents that real filesystem I/O inside `synctest.Test` is not durably blocking and can deadlock the bubble (`engine_blocking_test.go:27-34`). New timeout/backoff tests that use `persistfile` inside a bubble will hang.
**How to avoid:** fakes inside bubbles; real files in ordinary tests; the sim test uses the scheduler, not synctest.
**Warning sign:** a synctest test constructing `persistfile.New`.

### Pitfall 19: health is gate-only, so "health stays SERVING" needs no code but does need a test

**What goes wrong:** an implementer adds a health transition on dump failure, contradicting D-05.
**How to avoid:** assert `SERVING` in the EACCES stand test and document the decision in OPERATIONS.
**Warning sign:** `SetServingStatus` in a dump-failure path.

### Pitfall 20: root ignores directory modes

**What goes wrong:** the EACCES stand test passes as root because chmod does not bind the owner.
**How to avoid:** `os.Geteuid() == 0 → t.Skip`, matching `TestFileStore_saveFailureHygiene`.
**Warning sign:** the failure never triggers in a local root shell.

### Pitfall 21: metric sources with nil accessors panic in the zero-state test

**What goes wrong:** `metrics.Snapshot(Sources{...})` calls every source unconditionally; a new field left nil panics `TestMetrics_families`.
**How to avoid:** update `zeroSources()` and the `families` table in the same task as the family; keep every new sample rendered at zero.
**Warning sign:** a metrics diff without a test diff.

### Pitfall 22: no randomness or map order in anything replayable

**What goes wrong:** the tie-break, backoff or preservation naming must be a pure function of inputs; a map iteration or `math/rand` would break same-seed replay and byte-identical dumps.
**How to avoid:** `bytes.Compare` over explicit fields; constants for delays; an injected `now` for names; the sim test compares two same-seed runs.
**Warning sign:** `for key := range records` deciding a winner.

### Pitfall 23: `Encode`'s error path must not leave a temp behind

**What goes wrong:** with a fill callback, an encode error mid-stream leaves a partially written temp; the adapter must remove it and return the encode error (classified as `other`).
**How to avoid:** the `write` step's error hygiene (close+remove) covers a fill error; add a failure-matrix row for "fill returns an error".
**Warning sign:** the order test's failure rows only cover the five seam steps.

## Code Examples

Illustrative shapes, consistent with the contracts above (not verbatim production code).

### Dump decode with a completeness census

```go
func Decode(dump []byte, valueMaxBytes int) (map[string]storage.Record, DecodeStats, error) {
	var stats DecodeStats
	reader := bufio.NewReaderSize(bytes.NewReader(dump), maxDumpLineBytes(valueMaxBytes)+1)

	headerLine, err := readLine(reader, valueMaxBytes)
	if err != nil || len(headerLine) == 0 {
		return nil, DecodeStats{}, fmt.Errorf("persist: dump header unreadable: %w", err)
	}
	var h header // {FormatVersion int; RecordCount *int `json:"record_count,omitempty"`}
	if err := json.Unmarshal(headerLine, &h); err != nil {
		return nil, DecodeStats{}, fmt.Errorf("persist: dump header unreadable: %w", err)
	}
	if h.FormatVersion != dumpFormatVersion {
		return nil, DecodeStats{}, errUnsupportedFormat
	}

	records := make(map[string]storage.Record)
	lines := 0
	for {
		line, terminated, err := readRecordLine(reader, valueMaxBytes)
		if err != nil {
			return nil, DecodeStats{}, err // ErrBufferFull -> too long, io error
		}
		if line == nil {
			break
		}
		lines++
		var rec recordLine
		if json.Unmarshal(line, &rec) != nil {
			if !terminated { // torn tail
				return nil, DecodeStats{}, fmt.Errorf("persist: dump record truncated")
			}
			stats.Malformed++
			continue
		}
		decoded, ok, class := decodeRecord(rec, valueMaxBytes)
		if !ok {
			stats.add(class)
			continue
		}
		records[rec.Key] = decoded // duplicate keys: last-wins
	}
	if h.RecordCount != nil && lines < *h.RecordCount {
		return nil, DecodeStats{}, fmt.Errorf("persist: dump incomplete: %d of %d record lines", lines, *h.RecordCount)
	}
	return records, stats, nil
}
```

### Preservation at load

```go
func preserve(now time.Time, path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("dump path %s is not a regular file", path)
	}
	base := path + ".corrupt-" + now.UTC().Format("20060102T150405Z")
	for n := 0; ; n++ {
		candidate := base
		if n > 0 {
			candidate = fmt.Sprintf("%s-%d", base, n)
		}
		if _, err := os.Lstat(candidate); errors.Is(err, fs.ErrNotExist) {
			if err := os.Rename(path, candidate); err != nil {
				return "", err
			}
			_ = syncDirTolerant(filepath.Dir(path)) // best-effort evidence durability
			return candidate, nil
		}
	}
}
```

### Confirmation with a single-claim waiter

```go
func (e *Engine) await() error {
	w := &waiter{done: make(chan struct{})}
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return ErrClosed
	}
	e.markDirtyLocked()
	e.waiters = append(e.waiters, w)
	e.mu.Unlock()
	e.nudge()

	select {
	case <-w.done:
		return w.err
	case <-e.cfg.After(e.cfg.ConfirmTimeout):
	}

	e.mu.Lock()
	if w.claimed { // a dump resolved the race first
		e.mu.Unlock()
		<-w.done
		return w.err
	}
	w.claimed, w.err = true, ErrUnconfirmed
	e.waiters = slices.DeleteFunc(e.waiters, func(x *waiter) bool { return x == w })
	e.mu.Unlock()
	e.unconfirmed.Add(1)
	return ErrUnconfirmed
}
```

### Deterministic equal-version join

```go
func contentCompare(a, b Record) int { // +1 when a wins
	if a.Deleted != b.Deleted {
		if a.Deleted {
			return +1 // remove-wins
		}
		return -1
	}
	return bytes.Compare(a.Data, b.Data)
}

func (s *Store) join(key string, rec Record) (conflicts []Conflict, changed bool) {
	stored, ok := s.records[key]
	if !ok {
		s.install(key, rec)
		return nil, true
	}
	switch cmp := rec.Version.Compare(stored.Version); {
	case cmp > 0:
		s.install(key, rec)
		return []Conflict{{Key: key, Winner: rec.Version, Loser: stored.Version}}, true
	case cmp < 0:
		return []Conflict{{Key: key, Winner: stored.Version, Loser: rec.Version}}, false
	default:
		if RecordsEqual(stored, rec) {
			return nil, false // idempotent redelivery
		}
		if contentCompare(rec, stored) > 0 {
			s.install(key, rec)
			return []Conflict{{Key: key, Winner: rec.Version, Loser: stored.Version}}, true
		}
		return []Conflict{{Key: key, Winner: stored.Version, Loser: rec.Version}}, false
	}
}
```

### Backoff in the scheduler loop

```go
// after a failed runDump
e.retry = nextDelay(e.retry) // 100ms -> 200 -> ... -> cap 5s
e.conf.reDirty(failureInstant)

// rearm order: a pending retry wins over the debounce/deadline
if e.retry > 0 {
	if retry == nil {
		retry = e.cfg.After(e.retry)
	}
	return nil, nil, retry
}
```
## Validation Architecture

`workflow.nyquist_validation` is `true` in `.planning/config.json` (absent counts as enabled; this is explicitly true), so this section is required.

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go 1.27 `testing`, `go test -race`, `testing/synctest` for virtual-time unit tests |
| Config file | `Makefile` targets `lint`, `test`, `red-run`, `bench-smoke`; `scripts/red-expectations.tsv` for the red gate |
| Quick run command | `go test -race -count=1 ./internal/app/persist/ ./internal/persistfile/ ./internal/app/storage/` |
| Full suite command | `make lint && make test && SUPKV_RED_TESTS=1 bash scripts/red-runner` |
| Process-level command | `go test -race -count=1 ./internal/stand/ -run 'TestStand_persist'` (builds the real binary; ~10–60 s per scenario) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| PERSIST-01 | Exact-limit record survives restart at accepted flag values; non-positive exits 2; one constant | process + engine + unit | `SUPKV_RED_TESTS=1 go test -race ./internal/persistfile/ -run '^TestRed_P01_valueLimitEngine$'`; `SUPKV_RED_TESTS=1 go test -race ./internal/stand/ -run '^TestRed_P01_valueLimit$'`; `go test ./cmd/sup-kv/ -run 'TestMain_valueMax'`; `go test ./internal/app/storage/ -run 'TestStorage_valueLimit'` | Frozen tests exist; `cmd`/storage pins ❌ Wave 0 |
| PERSIST-02 | Prompt/Main/Backup/Unusable visible; one WARN/ERROR; evidence preserved; rotation leaves `.prev` | adapter unit + engine unit + process | `go test ./internal/persistfile/ -run 'TestFileStore_load'`; `go test ./internal/app/persist/ -run 'TestEngine_loadStats'`; `go test ./internal/stand/ -run 'TestStand_persistLoad'` | ❌ Wave 0 |
| PERSIST-03 | Census detects line-boundary truncation; one corrupt line loses one record; v1.0 dumps read | codec unit + property + fuzz | `go test ./internal/app/persist/ -run 'TestCodec_'`; `go test ./internal/app/persist/ -run '^$' -fuzz FuzzDumpDecode -fuzztime 30s` (nightly) | Partially (existing codec tests to extend) ❌ Wave 0 |
| PERSIST-04 | Dirty after Prompt/Backup/Unusable; snapshot decodes to memory; `markDirty` nudges | engine unit (synctest) + process | `go test ./internal/app/persist/ -run 'TestEngine_startDirty|TestEngine_markDirtyNudge'`; `go test ./internal/stand/ -run 'TestStand_snapshotAfterBackup'` | ❌ Wave 0 |
| PERSIST-05 | One ERROR + class counter per failure; growing bounded pauses; health SERVING | engine unit (synctest) + process | `go test ./internal/app/persist/ -run 'TestEngine_backoff|TestEngine_dumpFailureClasses'`; `go test ./internal/stand/ -run 'TestStand_dumpFailure'` | ❌ Wave 0 |
| PERSIST-06 | Streaming encode; ≤ two state sizes; model documented | codec unit + benchmark | `go test ./internal/app/persist/ -run 'TestEncodeTo_'`; `go test -run '^$' -bench 'BenchmarkDump' -benchmem ./internal/app/persist/` (evidence, not a gate) | ❌ Wave 0 |
| PERSIST-07 | Deterministic tie-break in store and on the wire; ADR; convergence | store unit/property + core unit + sim replay | `go test ./internal/app/storage/ -run 'TestMerge_'`; `go test ./internal/app/repl/ -run 'TestCore_equalVersion'`; `go test ./internal/app/repl/ -run 'TestScenario_tiebreak'` | ❌ Wave 0 (existing tests rewritten) |
| PERSIST-08 | Write/snapshot/flush/shutdown bounded; explicit unconfirmed outcome; RPC mapping/totality | engine unit (synctest) + RPC + process | `go test ./internal/app/persist/ -run 'TestEngine_confirmationTimeout|TestFlush_timeout'`; `go test ./internal/grpcapi/ -run 'TestServer_unconfirmed'`; `go test ./internal/app/persist/ -run '^TestEngine_blockingSaveHoldsSet$'` (stays green) | ❌ Wave 0 (blocking test exists) |
| PERSIST-09 | Orphan sweep; parent-dir sync; ENOTSUP/EINVAL dir tolerance; darwin build | adapter unit + `GOOS=darwin go build ./...` | `go test ./internal/persistfile/ -run 'TestSweep|TestSyncDir'`; `GOOS=darwin go build ./...` | ❌ Wave 0 |
| PERSIST-10 | Skipped oversize records counted/logged; evidence survives rotation | codec + engine + process | `go test ./internal/app/persist/ -run 'TestEngine_configShrunkValueLimits'` (updated); `go test ./internal/stand/ -run 'TestStand_shrunkLimit'` | Partially (existing test to update) |
| PERSIST-11 | Five internal roles; public API stable; port returns outcome/cause | compile + existing suite + load tests | `go build ./... && go test ./internal/app/persist/ ./internal/grpcapi/ ./internal/udprepl/ ./cmd/sup-kv/` | ✅ (suite is the assertion) |
| PERSIST-12 | Same clock/timer seam; sim exercises dump↔notify incl. failure and skip path | sim + unit | `go test ./internal/app/persist/ -run 'TestEngine_injectedClock'`; `go test ./internal/app/repl/ -run 'TestScenario_dump'` | ❌ Wave 0 |

### Red-Expectation Rows and Mutants

- Flip exactly the two P-01 rows in `scripts/red-expectations.tsv` from `red` to `green` after both frozen tests pass with `SUPKV_RED_TESTS=1`: `TestRed_P01_valueLimit` and `TestRed_P01_valueLimitEngine` `[VERIFIED: scripts/red-expectations.tsv:29-30]`. Nothing else in the file, the runner or the tests changes (`scripts/red-runner` header: the flip "IS this one-line change — never the runner script, never the test").
- The red-runner's mutant stage targets chaos (`storage/store.go`'s tombstone view and `repl/core.go`'s periodic sync) — unrelated to this phase and must stay green. The tie-break change touches `storage/store.go`; verify the tombstone mutant's `sed` anchor still exists after Phase 5 edits (`bash scripts/red-runner` with mutants).
- The red-runner's package list already includes `./internal/stand/` and `./internal/persistfile/` — no script extension needed.
- Evidence: the runner writes to `.planning/phases/02-stand-and-simulator-as-tests/red-evidence/<UTC>/`; the phase's gate evidence should quote the 7/7-row all-green run.

### Sampling Rate

- **Per task commit:** the task's own command from the map above (≤ 30 s for unit, longer for stand).
- **Per wave:** `go build ./... && make test` plus `make lint` at the last task of the wave.
- **Phase gate:** `make lint && make test && SUPKV_RED_TESTS=1 bash scripts/red-runner` green, plus the stand persist scenarios; record the red-run evidence.

### Wave 0 Gaps

- [ ] `internal/app/persist/load_stats_test.go` — `LoadStats` mapping and the dirty-on-outcome rules.
- [ ] `internal/app/persist/timeout_test.go` — confirmation timeout, flush timeout, waiter removal (synctest).
- [ ] `internal/app/persist/backoff_test.go` — ladder, cap, reset, `OnDumpFailure` (synctest).
- [ ] `internal/app/persist/tiebreak_test.go` (or storage) — the `(version, content)` law and property.
- [ ] `internal/app/persist/encode_stream_test.go` — `EncodeTo` equivalence and max-single-write bound.
- [ ] `internal/persistfile/load_outcome_test.go` — outcomes, preservation, collision suffixes, rotation survival.
- [ ] `internal/persistfile/sweep_test.go` — age bound, both dirs, `.prev`/`.corrupt-*` exclusion, parent sync.
- [ ] `internal/persistfile/saveerror_test.go` — ENOSPC/EACCES/other classification.
- [ ] `internal/app/repl/sim_dump_test.go` — the sim scenario and `virtualAfter`; `Cluster.NotifyInstalled` helper.
- [ ] `internal/stand/persist_test.go` (Linux) — load outcomes, EACCES failure decay + health, shrunk-limit evidence.
- [ ] `cmd/sup-kv/main_test.go` additions — non-positive `--value-max-bytes`.
- [ ] Fake updates: `fakeDumpPort`, `blockingDumpPort`, `grpcapi.fakeDumpStore` for `SaveStream`/`LoadResult`; `zeroSources` for the new metric source.
- [ ] No framework install; no new module.

### Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Disk-full (ENOSPC) process behaviour | PERSIST-05 | CI cannot reliably fill a filesystem without root/loop devices; EACCES is the automated stand proxy and exercises the same code path and class wiring | On a small tmpfs (`mount -t tmpfs -o size=1M`), point `--dump-path` at it, write until `Unavailable`, confirm the ERROR and the class label, free space, confirm recovery |
| Memory model on target hardware | PERSIST-06/PERF-04 | Shared-runner allocation numbers are noisy; the model is a measured claim, PERF-04 re-measures at Phase 8 | Run `make bench` on the target host, record `B/op`, peak sample and RSS in the phase summary/OPERATIONS |
| Real kernel fsync latency under a hung mount | PERSIST-08 | A genuinely hung fsync (NFS/SMB) is not reproducible in CI | Optional: point the dump at an NFS mount, suspend the server, observe `Unavailable` at the confirm timeout and the bounded shutdown; record as an operational note |

## Security Domain

`security_enforcement` is `true`, ASVS level 1. The dump file is the primary trust boundary of this phase: it is local, but it can be corrupted, hand-edited, or left over from an older/newer writer, and the loader runs before any serving path.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | No auth surface in persistence |
| V3 Session Management | no | None |
| V4 Access Control | yes | Dump directories 0700 and files 0600 (`os.CreateTemp` default), snapshot-name allow-list (`validateName`, `snapshot.go:96-106`) — preserved unchanged; new `.corrupt-*` files inherit the source mode (0600) |
| V5 Input Validation | yes | Dump grammar is validated line by line with bounded lines, a completeness census, per-record checksum/version/size checks, and total (panic-free) parsers; fuzzed with an in-repo corpus |
| V6 Cryptography | no (deliberate) | CRC32 is corruption detection, not tamper protection; do not present it as cryptographic integrity. No new crypto |
| V12 Files and resources | yes | Temp files created in the target directory, 0600, removed on every failure path; orphan sweep age-bounded; no symlink following (Lstat for preservation; CreateTemp in an owner-0700 dir) |

### Known Threat Patterns for the Go/stdlib persistence stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Corrupted or hand-edited dump (bit flips, truncation, injected newlines) | Tampering | Per-record CRC32 + per-line structural validation + completeness census; a bad record is isolated, a bad generation falls back and is preserved |
| Hostile dump line causing unbounded allocation (DoS at startup) | Denial of Service | `ReadSlice` bounded by `maxDumpLineBytes(valueMaxBytes)`; overlong line ⇒ generation fault; no `ReadString` on untrusted bytes |
| Fsync error silently treated as success | Tampering / Repudiation | File-data fsync keeps every error; only the directory step tolerates ENOTSUP/EINVAL, documented |
| Evidence destruction by rotation | Tampering | Unusable generations are renamed out of `main`/`.prev` before any save; preserved names are never rotated or swept |
| Path traversal via snapshot names | Tampering | Existing `validateName` (`[A-Za-z0-9._-]`, 1–64, no `.`/`..`) runs before any filesystem touch; unchanged |
| Denial of service via symlink at the dump/corrupt path | Tampering | `Lstat` before preservation; rename replaces the link itself; the directory is 0700 |
| A rogue same-version record forcing divergence (P-07) | Tampering / DoS | Deterministic content tie-break on every replica; the wire answers with the stored record; no randomness a peer can exploit to flip outcomes |
| Timing side channels on disk-full/permission classification | Information disclosure | Class labels are operator-facing metrics; no attacker-observable difference on the RPC path (both map to `Unavailable`) |

### Security-proof obligations for the plan

- Fuzz `FuzzDumpDecode` must keep running on the new decoder (corpus updated) and must include: header with/without count, count larger than lines, overlong line, injected newline, base64 corruption, torn tail.
- A test must prove no panic and a bounded allocation for an overlong line (feed a 2× bound line; expect a generation error, and assert the read did not consume unbounded memory by construction of the reader size).
- A test must prove a corrupt main is never copied into a snapshot after a backup start (D-04) — the snapshot's bytes decode to the loaded state.
- The EACCES stand test doubles as the "fsync error is not silently success" proof: writes fail, the error is logged and counted.

## State of the Art

| Old Approach | Current Approach (this phase) | When Changed | Impact |
|--------------|-------------------------------|--------------|--------|
| Frozen 61 440 decode bound independent of the flag | Configured limit owned by the store, passed to encode/decode | Phase 5 (v1.1) | P-01 class of silent loss eliminated; exactly-once constant |
| Silent fallback with `(dump, ok)` | Explicit outcome + per-generation cause + evidence preservation | Phase 5 | Data-losing starts are distinguishable from first starts; `.prev` survives the first rotation |
| Streaming `json.Decoder` over the whole file | Line-oriented decode with a header census | Phase 5 | Line-boundary truncation detected; one corrupt record loses one record |
| 50 ms fixed retry, no log | Bounded exponential backoff, one ERROR per attempt, class counters | Phase 5 | No busy loop; operator-visible causes; health semantics explicit |
| Whole dump materialized twice | `SaveStream` + `EncodeTo` | Phase 5 | Peak live heap ~1 state size; streaming proven structurally |
| Unbounded blocking calls | Confirmation/flush/shutdown deadlines with an explicit unconfirmed outcome | Phase 5 | A hung fsync cannot hang clients or shutdown |
| Silent equal-version no-op | Deterministic `(version, content)` tie-break, ADR 0012 | Phase 5 | Convergence under lost disk + clock rollback; deterministic replay preserved |
| Wall-clock reads inside the dumper | `Config.Now`/`Config.After`, sim-driven | Phase 5 | The dump↔notify interaction becomes simulatable |
| One 792-line engine file | Five internal roles, public API frozen | Phase 5 | Reviewable seams for Phase 6/7 |

**Deprecated/outdated:** `persist.maxRecordBytes` (deleted), `storage.defaultValueMaxBytes`/`cmd.defaultValueMaxBytes` (replaced by `storage.ValueMaxBytes`), the port's `Load(parse) ([]byte, bool)`, `DumpStore.Save([]byte)` as the dump path (kept only as a FileStore convenience), and the `fsync_darwin.go` helper (already deleted in Phase 3).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | A directory fsync can return `ENOTSUP` or `EINVAL` on some filesystems, and tolerating both is the right "documented skip" | D-08 | If wrong, a real durability error could be masked; mitigate by restricting tolerance to the directory step only and documenting it (the plan should keep the file-data path strict) |
| A2 | `encoding/json` ignores unknown object keys, so a v1.0 reader tolerates the new `record_count` field in a downgrade | D-03 | If wrong, a downgrade rejects the file; the phase does not require downgrade compatibility, but the claim must not be used as a safety argument in docs without a test |
| A3 | A 10 s confirmation timeout is safely above every legitimate dump in the documented envelope, and above the frozen blocking test's windows | D-07 | Too low would fail snapshots on slow media; too high delays the unconfirmed answer; tunable via `persist.Config` later |
| A4 | A 100 ms → 5 s backoff ladder is an acceptable "growing pause up to a cap" and keeps recovery within the confirm timeout | D-05 | If the operator expectation is a faster retry, the cap is a one-line constant change; document the ladder in OPERATIONS |
| A5 | A one-hour orphan-temp age bound is safe for any real save | D-08 | A save in flight for over an hour is already broken; the bound only protects a second process's temp |
| A6 | `TestEngine_configShrunkValueLimits` is not frozen and may be updated | D-09 / Pitfall 2 | If treated as frozen, the codec cannot adopt the configured limit and P-01 cannot be fixed |
| A7 | The CI/stand environment runs non-root, so chmod-based EACCES scenarios work; root runs must skip | Pitfall 20 | If CI runs as root, the EACCES stand scenario skips (acceptable; the unit-level classification test still covers the wiring) |
| A8 | The `.corrupt-<UTC>` naming has no operator contract beyond "distinct and never overwritten" | D-02 | If operators need a stable name/format, it is a doc change; the collision suffix is already stable |
| A9 | The test-count estimates (≈39 `Encode`/`Decode` call sites, 62 `Merge` sites, ≈20 `persistfile.New` sites) are accurate enough for planning | Summary / Pitfalls | Counts were taken this session with grep; exact numbers may shift by a few |

**If a claim is wrong:** A1 and A2 change only implementation details (both are guarded by tests); A3/A4/A5 are constants fixed in `persist.Config` or `persistfile`; A6 is the one that would block a requirement if misread — resolve it in the first planning pass.

## Open Questions

Each question is answered with a recommendation; the planner should treat the recommendation as the resolution unless the user says otherwise.

1. **Header count vs trailer vs version bump (D-03's "missing trailer" wording).**
   - What we know: D-03 allows "a record count or terminator"; it also says "format-version compatibility with the v1.0 dumps is preserved for reading" and "a missing trailer is treated as a truncated file".
   - The tension: a *trailer* implementation makes every v1.0 dump look truncated at upgrade; a version bump to 2 makes a downgrade reject new dumps.
   - **Recommendation:** header `record_count` in `format_version: 1`, absent = legacy v1.0 read path; document that a legacy generation has no completeness census (the "missing trailer" clause is satisfied for the new format by the missing/mismatched count, and for the legacy format by the documented compatibility exemption). If the user reads D-03 strictly, the fallback is `format_version: 2` + count (accept 1) and accept the downgrade-rejection consequence — decide before the format task starts.

2. **D-01's "the gRPC Set validator".**
   - What we know: there is no handler-side value check today; the store rejects with `ErrValueTooLarge` and `mapError` maps it to `InvalidArgument`.
   - **Recommendation:** no second validator. Document the store as the single enforcement point; add a handler test that an oversize value returns `InvalidArgument` (already covered by the stand metrics test) and nothing else.

3. **Should `ConfirmTimeout` be a CLI flag now?**
   - What we know: D-07 leaves the default to discretion; adding a flag means `--help`, README and Phase 9's flag-table test all change.
   - **Recommendation:** keep it a `persist.Config` field defaulting to 10 s this phase; Phase 7's OBSV-02/DOCS-02 can expose it deliberately. Note it in OPERATIONS as a non-configurable bound.

4. **Tombstone vs value at an equal version.**
   - What we know: either direction is deterministic; one operation is lost whichever wins.
   - **Recommendation:** remove-wins (`Deleted == true` sorts last), recorded in ADR 0012 with the consequence.

5. **`MergeChanged` vs changing `Merge`'s signature.**
   - What we know: a `changed` result is required for `ApplyRemote`'s dirty decision; changing `Merge` costs ~62 mechanical test edits, a new method costs one extra exported surface.
   - **Recommendation:** `MergeChanged` + `Merge` delegating; `Merge`'s doc says the equal-version tie-break can install and reports the conflict, so both paths share one law. If a reviewer insists on one method, the signature change is mechanical but must be budgeted as its own task.

6. **Where the load report lives.**
   - What we know: `Open`'s signature is frozen by a test; the report is startup-only.
   - **Recommendation:** `Engine.LoadStats()` returning a value copy, populated by `Open`, zero for `New`-built engines. No new package-level function.

7. **Preservation timestamp source.**
   - What we know: the engine has a `now` func; the adapter has none.
   - **Recommendation:** pass `now` into `Load`; deterministic tests and a name that reports when the corruption was found. Alternative (mtime) is recorded in D-02.

8. **Sweep location and errors.**
   - What we know: startup is the only safe time; the adapter has no logger.
   - **Recommendation:** sweep at the top of `Load`, report `Swept`/`SweepErr` in `LoadResult`, log in `cmd`.

9. **Should a legacy (no-count) generation log anything?**
   - **Recommendation:** no WARN — it is a normal upgrade state; the next dump writes the census. Optionally a DEBUG line; do not add a metric class for it.

10. **How to prove "at most two state sizes".**
    - What we know: peak-live is the right metric but noisy to assert in `make test`.
    - **Recommendation:** structural streaming test (max single write) + benchmark with `-benchmem` and a sampled peak, recorded in OPERATIONS/phase summary; no CI threshold this phase (PERF-04 owns the measured model at Phase 8).

11. **The sim's persistence seam.**
    - What we know: `Cluster.Write` re-Sets; the engine needs to share the store and notify an installed record.
    - **Recommendation:** additive `Cluster.NotifyInstalled(i, key)` (Lookup + NotifyLocalWrite + route + write-log capture, no Set); reuse `Cluster.Store(i)` for the engine. No change to existing sim semantics.

12. **Do skipped records mark the engine dirty?**
    - **Recommendation:** no (PERSIST-10's evidence window); the record is already absent from memory and the file is left until normal rotation.

13. **Which SPEC-DEVIATIONS rows flip?**
    - **Recommendation:** `S-PAR-05` → `implemented` (PERSIST-01); `S-PERS-08` → `implemented` (PERSIST-02 adds the fallback WARN); `S-PERS-09` → `implemented` (PERSIST-02's ERROR+metric; keep the OBSV-06 link for the rest of lifecycle logging); `S-ALG-REP-03` summary amended for the tie-break; `S-HLC-07` keeps its accepted-deviation status and links ADR 0012; `S-DM-04`/`S-DM-08`/`S-DM-05` links updated to the new constant/merge. Preamble sentence "ADR numbers 0001–0011" updated to mention 0012.

14. **What if preservation of the corrupt main fails on a Backup start?**
    - **Recommendation:** start anyway (the `.prev` data is good), surface `PreserveErr` in the report → a WARN naming the failure, and document that a later successful save could rotate the corrupt main over `.prev` in that pathological case (an unwritable directory usually prevents the save too, so the hazard is narrow). Do not silently retry preservation on every dump.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| Go toolchain | build, test, lint | ✓ | go1.27.4 linux/amd64 | — |
| `testing/synctest` | engine timing tests | ✓ | stdlib Go 1.27 | A manual clock fake (not recommended) |
| `python3` | `scripts/red-runner` JSON parsing | ✓ | 3.15.2 | — |
| `git` | phase branch/commits, red-runner evidence path | ✓ | 2.43.0 | — |
| `make` | `make lint/test/red-run` | ✓ | GNU Make 4.3 | direct `go` commands |
| `go tool golangci-lint` | lint gate | ✓ (module cache/tool directive) | v2.15.0 | — |
| Linux (loopback broadcast) | stand scenarios | ✓ | linux 7.3.4 | non-Linux: the stand reports/skips per TEST-05 policy |
| Network research providers | external docs | ✗ (no Context7/Exa/Brave MCP in this session) | — | GOROOT stdlib source + `go doc` (used; sufficient — the phase is stdlib-only) |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** the documentation provider; all statements it would have answered were verified locally from GOROOT and the repo.

## Sources

### Primary (HIGH confidence)

- In-repo source, read this session with line citations throughout: `internal/app/persist/engine.go`, `codec.go`, `errors.go`, `doc.go`; `internal/persistfile/{dumpfile,fsops,snapshot}.go`; `internal/app/storage/store.go`; `internal/port/dump.go`; `internal/app/repl/core.go` (the `applyRemote` branch); `internal/udprepl/repl.go` (adapter time); `cmd/sup-kv/main.go`; `internal/grpcapi/server.go`, `gate.go`; `internal/metrics/metrics.go`; `internal/app/internal/simnet/simnet.go`; the frozen tests and `scripts/red-expectations.tsv`; `.planning/phases/05-persistence/05-CONTEXT.md`; `.planning/REQUIREMENTS.md`; `.planning/ROADMAP.md`; `docs/SPEC-DEVIATIONS.md`; `docs/adr/0001`, `0003`, `0005`, `0011`.
- Go 1.27.4 toolchain source: `src/internal/poll/fd_fsync_posix.go` (EINTR retry via `ignoringEINTR`), `src/internal/poll/fd_fsync_darwin.go` (F_FULLFSYNC + ENOTSUP→fsync + EINTR), `src/os/file_posix.go` (`File.Sync` → `pfd.Fsync`).
- `go doc os.Rename` (replace semantics), `go doc os.CreateTemp` (0600, in-dir), `go doc testing/synctest.Test`.
- Local probe `/tmp/jsonprobe/main.go` against Go 1.27.4: stream `json.Decoder` is permanently poisoned by a syntax error and recovers from a base64 error; line-wise `json.Unmarshal` recovers from both.

### Secondary (MEDIUM confidence)

- `docs/reviews/2026-11-19-v1.0/REVIEW.md` §6 (P-01…P-10) and the Q-03/Q-04/Q-06 rows — the finding text and measured figures (18 attempts/s, 74 ms pause, ~5 state sizes). Standard of evidence: the review's own measurements, not re-run this session.
- `.planning/phases/04-protocol/04-REVIEW.md` (New-1, IN-04) and `04-REVIEW-DISPOSITION.md` (deferrals) — the Phase-4 ledger is authoritative for what was deferred here.
- `.planning/research/PITFALLS.md` (the POSIX durability recipe) — confirms the temp→fsync→rename→dir-sync order this phase preserves.

### Tertiary (LOW confidence)

- The specific errnos returned by directory fsync on non-ext4 filesystems (`ENOTSUP` vs `EINVAL`) — marked `[ASSUMED]` in A1; not reproduced in this session.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — stdlib only, pinned toolchain, no new dependencies; the GOROOT mechanisms were read, not recalled.
- Architecture: HIGH for the current-state map (every quote read this session); MEDIUM for the newly proposed contracts (they are designs to be pinned by the phase's tests, not observed behaviour).
- Pitfalls: HIGH for the frozen-test/lint/ripple items (counts and quotes taken from the tree); MEDIUM for the operational constants (timeouts, ladders, age bounds) which are discretionary and tunable.

**Research date:** 2026-11-22
**Valid until:** 2026-12-22 (30 days) — the repo moves one phase at a time; re-check the frozen-test and TSV rows if Phase 5 planning spans a commit that touches `internal/stand`, `internal/persistfile` or `scripts/`.
