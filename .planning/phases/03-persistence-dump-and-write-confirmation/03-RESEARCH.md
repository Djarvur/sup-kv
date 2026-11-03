# Phase 3: Persistence — Dump and Write Confirmation - Research

**Researched:** 2026-11-03
**Domain:** Go stdlib crash-consistent file persistence + concurrent write-confirmation (watermark/waiter) design
**Confidence:** HIGH

<user_constraints>

## User Constraints (from CONTEXT.md)

### Locked Decisions

*(copied verbatim from `.planning/phases/03-persistence-dump-and-write-confirmation/03-CONTEXT.md` §Implementation Decisions)*

**Architecture split (pure core vs file adapter)**
- **D-01:** The dump codec — JSON assembly and parsing, per-record checksum verification, and computing the startup watermark (max `l` plus max counter at that `l`, VERS-04) — is a pure component inside `internal/app`, property-tested without disk. The file machinery (temp file, fsync, rename, parent-directory fsync, `.prev` rotation) lives in an adapter outside the app boundary, per the depguard rule `sup-kv-app-no-io`. — **Reversibility:** costly — the port interface is what Phases 4–5 code against.
- **D-02:** The port seam carries whole dump bytes. Saving: the core hands the adapter one complete byte block and the adapter's atomic-write interface knows nothing about records. Loading: the adapter resolves the fallback chain main → `.prev` → empty start and returns the first generation that reads as bytes (a truncated or unparseable file counts as unreadable → next generation); the codec then validates per-record checksums and discards bad records (PERS-02). — **Reversibility:** costly — same contract consumers.

**Write confirmation (PERS-04)**
- **D-03:** The confirmation mechanism lives in the core: a confirmed-version watermark plus a wait primitive ("a completed dump has covered version X") that unblocks writers. No callbacks, no events — callers observe results, mirroring Phase 2 D-01.
- **D-04:** Bounded max wait defaults to **1 second**: even under a continuous write stream a dump must start no later than 1 s after the oldest unconfirmed write, so waiters are confirmed within ~1 s plus dump duration (PERS-03; success criterion 3). Exact parameter name is planner's discretion; `docs/IDEA.md` §Parameters gains the entry.
- **D-05:** On dump failure all current waiters immediately receive the error (Unavailable at the API layer, Phase 5); the write stays in memory and replicates later (PERS-04). The next dump starts on the next change or the debounce timer — no auto-retry loops holding writers hostage.

**Dump file format (PERS-01)**
- **D-06:** One JSON object: a header carrying the format version, then the records one per line, sorted by key — readable by eye, greppable per record, and byte-stable across generations of the same state.
- **D-07:** Per-record checksum is CRC32 — uniform with the replication integrity story, within the accepted trust model (docs/IDEA.md §Trust model).

**Snapshots (PERS-05)**
- **D-08:** Named snapshots live beside the dump in `<dump_path>.snapshots/<name>` — no extra launch parameter.
- **D-09:** Restore replays only the snapshot's keys as ordinary writes/deletes with fresh versions (Deleted → Delete); keys absent from the snapshot are untouched — their newer versions win by normal LWW. Literal "replay as ordinary writes", not a full state replacement.
- **D-10:** Creating a snapshot first forces the current dump to complete (so the copy contains every confirmed write), then copies the finished dump byte-for-byte under the name; snapshots read back with the same codec.

### Claude's Discretion

- Adapter package name and location (an `internal/` package appearing with this phase per AGENTS.md layout rules), the exact interface names and signatures in `internal/port`, JSON field spellings beyond PERS-01's named fields, the max-wait parameter name, temp-file naming, and the concurrency internals of the waiter primitive — planner decides per go-ultimate conventions.

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.

Additional out-of-scope markers from the phase boundary: no replication, no gRPC/API surface, no metrics (Phases 4–6); snapshot create/load ship as library-level operations, gRPC wiring arrives in Phase 5; macOS F_FULLFSYNC durability is a Phase 6 verification item, not this phase's blocker.

</user_constraints>

## Project Constraints (from AGENTS.md / .claude/CLAUDE.md)

- Go version pinned in `go.mod` (`go 1.27.1`); do not raise without an ADR. Local toolchain verified 1.27.3. [VERIFIED: go.mod line 3; `go version`]
- `make lint` (`go tool golangci-lint config verify && go tool golangci-lint run ./...`) and `make test` (`go test -race -cover ./...`) before pushing; CI runs the same. [VERIFIED: Makefile]
- Depguard boundary: `internal/app` files must not import `net`, `os`, `syscall`, `google.golang.org/*` (rule `sup-kv-app-no-io`, with `!$test` exemption for `*_test.go`); `internal/port` restricted to `$gostd` + own module path (rule `sup-kv-port-stdlib-only`). Any boundary change is a visible `.golangci.yml` diff. [VERIFIED: .golangci.yml:68-88 — deny list verbatim: `pkg: net`, `pkg: os`, `pkg: syscall`, `pkg: google.golang.org/`; allow list verbatim: `$gostd`, `github.com/Djarvur/sup-kv`]
- No `//nolint` without naming the linter and a written reason (`nolintlint.require-explanation`/`require-specific`). Phase 2 achieved zero nolint in `internal/app` except documented test-flag case. [VERIFIED: AGENTS.md deviation 3; testutil/rng.go:35]
- Zero direct runtime requires; adapters appear with the phases that need them; do not create empty adapter directories. [VERIFIED: AGENTS.md Dependency policy / Layout]
- go-ultimate (v0.15.0) conventions govern: `t.Parallel()` by default; `t.TempDir()` for scratch dirs; **`testing/synctest` for anything involving time or goroutine coordination** — "`time.Sleep` in a test is a smell: on Go 1.25+ reach for `synctest`"; every goroutine needs a documented exit condition ("Do not start goroutines from inside library code unless the API contract explicitly documents the lifecycle"); `context.Context` first param, never stored in structs; assertion libraries forbidden (vanilla `testing` + table tests). [CITED: go-ultimate references/engineering-policy.md §Concurrency, references/testing.md §Testing concurrent code]
- `.claude/CLAUDE.md` additionally mandates (from project stack research): keep the `encoding/json` **v1 API** for on-disk format stability — "v1 API is stable and deterministic: map keys are sorted, `[]byte` is base64 automatically — exactly the dump format the design specifies"; do not chase json/v2 semantics mid-project. [CITED: .claude/CLAUDE.md §Technology Stack]

<phase_requirements>

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PERS-01 | Human-readable JSON dump of the full map (format version + records with base64 Data, RecordVersion, Deleted, Checksum), greppable | JSONL codec pattern (Pattern 2); `[]byte`→base64 automatic; `[16]byte`→number-array pitfall; sorted-key byte-stable encoding |
| PERS-02 | Atomic dumps (temp in target dir, fsync, rename, parent-dir fsync), `.prev` kept, fallback main → `.prev` → empty start, bad-Checksum records discarded | Canonical atomic-write sequence (Pattern 3); fallback-chain port seam options (Open Question 1); per-record verify via `storage.RecordChecksum` |
| PERS-03 | Debounced dumps (`dump_debounce` 50 ms) with bounded max wait (1 s, D-04) — bursts cannot starve completion | Debounce loop with oldest-change deadline (Pattern 5); synctest burst test design (Validation §REQ-PERS-03) |
| PERS-04 | Write returns OK only after a dump containing it completed (version watermark); dump failure → error to waiters, write stays in memory and replicates | Channel-close broadcast waiter (Pattern 4); watermark publish; sentinel error following `storage/errors.go` precedent |
| PERS-05 | Named snapshots (copies of the dump); restore replays as ordinary writes/deletes with fresh versions | Snapshot flow (Pattern 6); `Store.Delete` no-op semantics make Deleted-replay safe; name validation (Security §V5) |
| VERS-04 (persistence half) | Restart seeds the dump's full watermark (max `l` + max counter at that `l`) into `clock.NewClock` initialL/initialC | Watermark computation (Pattern 2 / Code Example 4); `NewClock` signature verified |

</phase_requirements>

## Summary

Phase 3 is a pure-Go, stdlib-only persistence layer with three cooperating pieces: (1) a pure codec + confirmation engine inside `internal/app` (the depguard gate forbids `os`/`net` there, and `time` is allowed), (2) a dump port interface declared in `internal/port` carrying whole dump bytes (D-02), and (3) a file adapter under `internal/` implementing atomic writes (temp file in the target directory → fsync → rename → directory fsync), `.prev` rotation, the fallback chain, and the `<dump_path>.snapshots/<name>` directory. No external packages are installed — the phase's entire surface is Go standard library plus the existing Phase 2 packages (`storage`, `clock`, `version`, `internal/testutil`).

The three externally-verifiable technical facts that shape the implementation were confirmed against pkg.go.dev this session: (a) under the `encoding/json` v1 API "`[]byte` encodes as a base64-encoded string" but fixed-size arrays like the `[16]byte` `RecordVersion` encode as **JSON number arrays** — the version must be converted explicitly (hex recommended) for a greppable dump; (b) `json.Encoder.Encode` "writes the JSON encoding of v to the stream, followed by a newline character" — one `Encode` call per record yields the one-record-per-line format D-06 requires, and v1 map marshaling is deterministic, though records should still be sorted explicitly because the source is a Go map; (c) the Go 1.23+ timer rework (sup-kv's `go 1.27.1` module activates it) makes `Timer.Reset` safe without channel draining, which simplifies the debounce loop that PERS-03 lives on.

The write-confirmation engine is the novel design work: a confirmed-version watermark, per-waiter channels resolved by close (the `sync` docs themselves recommend channel-close over `sync.Cond` for broadcast, and `Cond` has no error-fan-out), a single dumper goroutine serializing dumps (only one dump in flight — prevents temp-file and `.prev` rotation races), and a debounce that resets on change but is capped by the oldest-unconfirmed-change deadline (D-04's 1 s). Deterministic tests of that engine run under `testing/synctest` with a fake dump port — real file I/O inside a bubble stalls it, so adapter tests stay outside bubbles with `t.TempDir()` and real time.

**Primary recommendation:** one new pure package `internal/app/persist` (codec + confirmation engine, property-tested via `testutil`), one port interface in `internal/port` for save/load/snapshot bytes, one adapter package (e.g. `internal/persistfile`) owning every `os` call; engine tests under `synctest` with a fake port, adapter tests against `t.TempDir()`, and a documentation task appending the max-wait parameter to `docs/IDEA.md` §Parameters (D-04).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Dump codec (JSONL encode/decode, checksum verify/discard, format version) | `internal/app` (pure core) | — | D-01: pure, property-tested without disk; `encoding/json` is stdlib-legal inside the app boundary |
| Startup watermark computation (max `l` + max counter at that `l`) | `internal/app` (pure core) | — | D-01; feeds `clock.NewClock` (VERS-04) |
| Confirmation watermark + waiter primitive | `internal/app` (pure core) | — | D-03: "lives in the core"; `time`/`sync` are not denied by the depguard rule |
| Debounce scheduling + bounded max wait + dump orchestration | `internal/app` (pure core) | — | Pure logic over timers; the dumper goroutine's lifecycle must be documented API contract (go-ultimate) |
| Dump port interface (whole bytes save/load/snapshot) | `internal/port` | — | Phase 2 D-03: phase 3 declares the interface it needs; stdlib + own module only — the interface sees `[]byte`, never records (D-02) |
| Atomic write, `.prev` rotation, fallback-chain resolution, temp-file hygiene | `internal/` file adapter (os allowed) | — | D-01/D-02: all `os` machinery outside the app boundary; adapter contains no business decisions (go-ultimate adapter discipline) |
| Snapshot directory + byte-copy create + read back | `internal/` file adapter | engine (forces dump first) | D-08/D-10: `<dump_path>.snapshots/<name>` derived from dump path, no new parameter |
| Snapshot restore (replay-as-writes) | `internal/app` engine | — | D-09: core decision logic (which keys, fresh versions, Deleted→Delete) |
| Unavailable sentinel error | `internal/app` engine package | — | Follows `storage/errors.go` sentinel pattern; Phase 5 maps to the `Unavailable` RPC code |
| gRPC exposure of snapshots / Unavailable mapping | Phase 5 (out of scope) | — | Phase boundary: library-level operations only |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `encoding/json` (v1 API) | Go 1.27.1 toolchain | JSONL dump encode/decode | Decided stack: v1 API is deterministic and stable for the on-disk contract; `[]byte`→base64 automatic; `Encoder.Encode` emits newline-terminated values (JSONL-native) [CITED: pkg.go.dev/encoding/json; .claude/CLAUDE.md §Technology Stack] |
| Go stdlib `os` (+ `path/filepath`) | Go 1.27.1 | Temp file, fsync, rename, dir fsync, snapshots dir | The entire file-adapter surface; `CreateTemp` 0600 + pattern semantics verified [CITED: pkg.go.dev/os] |
| Go stdlib `sync` | Go 1.27.1 | Engine mutex, channel-close broadcast for waiters | Zero-dep policy; docs recommend channels over `Cond` for broadcast [CITED: pkg.go.dev/sync] |
| Go stdlib `time` | Go 1.27.1 | Debounce/max-wait timers | Go 1.23+ timer semantics active (module `go 1.27.1`): `Reset`/`Stop` safe without draining, unreferenced timers GC-able [CITED: go.dev/doc/go1.23] |
| Go stdlib `testing/synctest` | Go 1.27.1 | Deterministic engine tests (fake clock bubble) | Stable since 1.25; go-ultimate mandates it for time/goroutine tests [CITED: pkg.go.dev/testing/synctest; go-ultimate testing.md] |
| `github.com/Djarvur/sup-kv/internal/app/storage` | in-repo | Record type, `Records()` dump source, `RecordChecksum`, `Merge` for load/restore, sentinel-error precedent | Reuse — the checksum domain must be identical to replication's (D-07) [VERIFIED: internal/app/storage/store.go] |
| `github.com/Djarvur/sup-kv/internal/app/clock` | in-repo | `NewClock(now, initialL, initialC, driftWarn)` watermark seeding | VERS-04 restart seam already delivered by plan 02-04 [VERIFIED: internal/app/clock/clock.go:63] |
| `github.com/Djarvur/sup-kv/internal/app/version` | in-repo | `Unpack`/`Compare` for watermark and waiter comparisons | `Unpack` is total over any bit pattern — decoded file bytes can't panic it [VERIFIED: internal/app/version/version.go:53-58] |
| `github.com/Djarvur/sup-kv/internal/app/internal/testutil` | in-repo | Seeded generators + printed-seed replay for codec properties | D-04 (phase 2) contract: `NewRNG` prints "property seed:" [VERIFIED: internal/app/internal/testutil/rng.go:41-51] |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `encoding/hex` | stdlib | RecordVersion → greppable hex string in the dump line | Recommended over base64 for the version field: UUID-shaped, human-scannable (planner's choice of exact spelling) |
| `slices` / `maps` (`slices.Sorted(maps.Keys(m))`) | stdlib | Deterministic record ordering | Encode path: Go-map iteration order is random; D-06 demands sorted-by-key, byte-stable output |
| `bytes` | stdlib | Build the whole dump block (D-02) in memory | `bytes.Buffer` + `json.NewEncoder`; decode via `json.NewDecoder(bytes.Reader)` — no line-splitting, no bufio 64 KiB line-limit trap |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| JSONL (header line + record lines) | Single pretty-printed JSON object | A single object with nested records is NOT greppable per record; D-06's "records one per line" forces JSONL [CITED: 03-CONTEXT D-06] |
| In-repo atomic write (~50 lines) | A durability library (e.g. write-file-atomic equivalents) | Forbidden by zero-dep policy; the canonical sequence is 5 syscalls — nothing to gain from a dependency |
| Channel-close waiter broadcast | `sync.Cond` | `Cond` has no timeout/context and no error fan-out; Go docs themselves say channels are usually better ("Broadcast corresponds to closing a channel") [CITED: pkg.go.dev/sync] |
| `testing/synctest` engine tests | Real-time sleeps with margins | Flaky by construction; go-ultimate calls `time.Sleep` in tests a smell; synctest turns D-04's deadline into an exact assertion [CITED: go-ultimate testing.md] |

**Installation:** none — zero new dependencies; `go.mod` must not gain a single direct require (Phase 1 D-03).

**Version verification:** no registry packages this phase; stdlib + in-repo only. Toolchain verified locally: `go version go1.27.3 linux/amd64`, `go build ./...` OK.

## Package Legitimacy Audit

**Not applicable — this phase installs zero external packages.** The runtime graph stays empty (stdlib + own module only); the Package Legitimacy Gate was therefore not run. The only new module-level artifact is in-repo source.

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.

## Architecture Patterns

### System Architecture Diagram

Write-confirmation path (PERS-03/PERS-04):

```
caller ──Set/Delete/SetIfAbsent──▶ engine (internal/app/persist)
                                     │ 1. store.Set → fresh version v (store lock orders versions)
                                     │ 2. mark dirty (engine lock), register waiter for v
                                     │ 3. BLOCK on waiter channel  ─ ─ ─ ─ ─ ─ ─ ─ ─ ┐
                                     ▼                                             │
                      dumper goroutine (single, serialized dumps)                  │
                        │ debounce 50ms reset per change                           │
                        │ OR oldest-unconfirmed-change + 1s deadline (D-04)        │
                        ▼                                                          │
                        store.Records() snapshot (tombstones incl.)                │
                        ▼                                                          │
                        codec.Encode → ONE complete byte block (D-02)              │
                        ▼                                                          │
                        port.SaveDump(bytes)  ──▶ file adapter:                     │
                            temp(dir) → write → fsync → rename(main→.prev first)   │
                            → rename(tmp→main) → fsync(dir)                         │
                        ▼                                                          │
                        on success: watermark = max version in snapshot             │
                        on failure: error to ALL current waiters (D-05)             │
                        └── close(waiter.done) ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ┘
        ◀── version, drift, nil  (write confirmed on disk)
        ◀── version, drift, ErrDumpFailed  (write STAYS in memory, replicates later)
```

Startup load path (PERS-02 + VERS-04):

```
adapter.LoadDump(parse oracle)  ── try main ──▶ bytes? parse ok? ──no──▶ try .prev ──no──▶ EMPTY START
        │ first generation that parses
        ▼
codec.Decode: per-record RecordChecksum verify ──bad──▶ discard record (count it)
        │ valid records only
        ├──▶ watermark (initialL, initialC) = max l, max counter at that l
        │        └──▶ clock.NewClock(now, initialL, initialC, driftWarn)
        └──▶ store.Merge(key, rec) per record (versions preserved; checksum recomputed)
```

Snapshot path (PERS-05, D-08/D-09/D-10):

```
Create(name): flush (wait for dump covering all current state) ──▶ adapter byte-copies
              finished dump → <dump_path>.snapshots/<name> (atomic, validated name)
Load(name):   adapter reads snapshot bytes ──▶ same codec.Decode ──▶ engine replays
              each key as ordinary Set/Delete with FRESH versions (Deleted→Delete,
              ErrNotFound skip); keys absent from snapshot untouched
```

### Recommended Project Structure

```
internal/
├── app/
│   └── persist/            # NEW (name: planner's discretion) — pure, no os/net
│       ├── doc.go          #    package contract + lifecycle docs
│       ├── codec.go        #    JSONL encode/decode, checksum verify/discard, watermark
│       ├── engine.go       #    confirmation watermark, waiters, debounce/max-wait dumper
│       ├── errors.go       #    ErrDumpFailed-style sentinel (Phase 5 → Unavailable)
│       ├── codec_test.go   #    seeded property tests (round-trip, discard, watermark)
│       └── engine_test.go  #    synctest tests (fake port): confirm/fail/burst/snapshot
├── port/
│   └── dump.go             # NEW — SaveDump/LoadDump/snapshot bytes interface(s)
└── persistfile/            # NEW (name: planner's discretion) — the ONLY os-importing code
    ├── dumpfile.go         #    atomic write, .prev rotation, fallback chain, temp hygiene
    ├── snapshot.go         #    .snapshots dir, validated names, byte-copy create/read
    └── dumpfile_test.go    #    real t.TempDir() tests: rotation, chain, truncation, perms
```

Note: codec and engine may also be two `internal/app` sub-packages per the phase 2 package-per-concern precedent — planner's call; one package keeps the watermark type private to both halves.

### Pattern 1: Port seam carries whole bytes (D-02)

**What:** The engine hands the adapter one complete `[]byte`; the adapter's save interface knows nothing about records. Loading inverts: the adapter walks the fallback chain and returns the first generation that parses.
**When to use:** Exactly here — this is the locked contract Phases 4–5 code against.

Recommended shape (names are planner's discretion):

```go
// Sketch — internal/port/dump.go. Exact names/signatures: planner decides.
type DumpStore interface {
    // Save durably writes dump as the new main generation, rotating the old main to .prev.
    Save(dump []byte) error
    // Load returns the first generation (main → .prev) whose bytes pass parse
    // (a truncated or unparseable file counts as unreadable → next generation,
    // D-02); ok=false is the empty start. The engine supplies the codec's
    // parse as the readability oracle, keeping the adapter record-blind.
    Load(parse func([]byte) error) (dump []byte, ok bool, err error)
    // CreateSnapshot byte-copies the CURRENT on-disk dump under name (D-10);
    // ReadSnapshot returns a snapshot's bytes for codec decoding.
    CreateSnapshot(name string) error
    ReadSnapshot(name string) ([]byte, error)
}
```

The parse-oracle parameter is what makes D-02's wording literal ("the adapter resolves the fallback chain … a truncated or unparseable file counts as unreadable") without the adapter importing the codec. See Open Question 1 for the simpler two-byte-slices alternative and the tradeoff.

### Pattern 2: Byte-stable JSONL codec (PERS-01, D-06, D-07)

**What:** Line 1 is the header object (format version only — anything volatile like a timestamp would break byte-stability); lines 2..N+1 are record objects sorted by key. Same state ⇒ identical bytes.
**When to use:** Every dump generation and every snapshot.

```go
// Source: encoding/json semantics verified against pkg.go.dev (see Sources);
// field spellings beyond PERS-01's named ones are planner's discretion.
type header struct {
    FormatVersion int `json:"format_version"`
}

type recordLine struct {
    Key      string `json:"key"`
    Version  string `json:"record_version"` // hex of version.RecordVersion — [16]byte
    Data     []byte `json:"data"`           // []byte → base64 automatically; nil → null
    Deleted  bool   `json:"deleted"`
    Checksum uint32 `json:"checksum"`       // fits float64 exactly (< 2^53)
}

func Encode(records map[string]storage.Record) ([]byte, error) {
    var buf bytes.Buffer
    enc := json.NewEncoder(&buf)
    enc.SetEscapeHTML(false) // greppability: keys containing &, <, > stay literal
    if err := enc.Encode(header{FormatVersion: dumpFormatVersion}); err != nil {
        return nil, err // Encode's error MUST be handled — errcheck + errchkjson gate
    }
    for _, key := range slices.Sorted(maps.Keys(records)) { // map order is random; sort!
        rec := records[key]
        if err := enc.Encode(recordLine{
            Key:      key,
            Version:  hex.EncodeToString(rec.Version[:]),
            Data:     rec.Data,
            Deleted:  rec.Deleted,
            Checksum: rec.Checksum,
        }); err != nil {
            return nil, err
        }
    }
    return buf.Bytes(), nil
}
```

Decode is the mirror: `json.NewDecoder` streams successive values (no line handling, no bufio limits), first `Decode(&header)`, then a record loop until `io.EOF`; any decode error = generation unreadable (→ chain falls back); a per-record checksum mismatch = discard that record and continue (PERS-02). Verify with the existing `storage.RecordChecksum(key, ver, data, deleted)` [VERIFIED: internal/app/storage/store.go:104-110 — signature verbatim] — do NOT re-implement a second checksum: the implemented domain is CRC32 IEEE over `[]byte(key)`, `ver[:]`, `data`, one flag byte (0 live / 1 deleted), via `crc32.Update(0, crc32.IEEETable, …)`.

### Pattern 3: Canonical atomic write (PERS-02)

**What:** temp file in the TARGET directory → write → fsync file → rotate `main`→`.prev` → rename temp→main → fsync parent directory.
**When to use:** Every dump save and snapshot create.

```go
// Sketch — order is the contract; planner refines error/close paths.
// Source: pkg.go.dev/os + community crash-consistency consensus (Sources).
func (a *Adapter) saveAtomically(path string, data []byte) error {
    tmp, err := os.CreateTemp(filepath.Dir(path), "dump-*") // SAME dir: rename never crosses filesystems
    if err != nil { return err }
    tmpName := tmp.Name()
    defer func() { // no temp litter on any failure path (CreateTemp is caller-managed)
        if err != nil { _ = tmp.Close(); _ = os.Remove(tmpName) }
    }()
    if _, err = tmp.Write(data); err != nil { return err }
    if err = tmp.Sync(); err != nil { return err }  // data durable BEFORE the rename can be durable
    if err = tmp.Close(); err != nil { return err }
    if _, statErr := os.Stat(path); statErr == nil {
        if err = os.Rename(path, path+".prev"); err != nil { return err } // rotate old generation first
    }
    if err = os.Rename(tmpName, path); err != nil { return err } // atomic replace on Unix
    dir, err := os.Open(filepath.Dir(path))
    if err != nil { return err }
    defer dir.Close()
    return dir.Sync() // fsync the directory: the rename itself becomes durable
}
```

Why each step exists [CITED: pkg.go.dev/os; MEDIUM: community consensus — see Sources]: `File.Sync` "commits the current contents of the file to stable storage"; `os.Rename` on Unix atomically replaces an existing non-directory target; skipping the pre-rename fsync risks a durable rename pointing at unflushed data; skipping the directory fsync risks losing the rename itself. Rotation order matters: rotate-then-rename leaves a crash window where `main` is absent — exactly the case the fallback chain absorbs (main missing → `.prev` serves).

### Pattern 4: Confirmation watermark + channel-close waiters (PERS-04, D-03, D-05)

**What:** The engine tracks the highest version covered by a completed dump; writers block on a per-waiter channel that dump completion closes.
**When to use:** The write-confirmation path.

```go
// Sketch — concurrency internals are planner's discretion (CONTEXT).
// Broadcast by close: wake ALL waiters of a completed dump; on failure the
// same close carries the error (D-05). sync.Cond cannot do error fan-out.
publish := func(cover version.RecordVersion, dumpErr error) {
    e.mu.Lock()
    if dumpErr == nil && cover.Compare(e.watermark) > 0 {
        e.watermark = cover
    }
    resolved := e.takeWaitersUpTo(e.watermark, dumpErr) // waiters with version <= watermark
    e.mu.Unlock()
    for _, w := range resolved {
        close(w.done) // exactly-once wake; w.err set under e.mu
    }
}
```

Key mechanics: local versions are strictly monotonic (Phase 2 locked the store write paths across `Tick`, so version order equals commit order — STATE.md plan 02-01), so a version watermark is well-ordered for local writes; only one dump runs at a time (single dumper goroutine — serializes temp files and rotation); waiters register BEFORE returning from the write path; the error sentinel returned to waiters is the Phase-5 `Unavailable` source. Phase 4 note: remote merges will mark dirty without waiting (REPL-03 Acks don't wait for dumps) — keep the dirty-mark path generic (any state change dirties), only local writes register waiters.

### Pattern 5: Debounce with bounded max wait (PERS-03, D-04)

**What:** The debounce timer resets on each change; an independent deadline = oldest-unconfirmed-change + 1 s forces a dump start regardless of continuing changes.
**When to use:** The dumper loop — this is the anti-starvation mechanism success criterion 3 tests.

```go
// Sketch. Go 1.23+ timer semantics apply (go.mod go 1.27.1): Reset is safe
// without draining; unreferenced timers are GC-able. Do NOT cargo-cult the
// pre-1.23 drain idiom.
for {
    select {
    case <-ctx.Done():
        return // documented goroutine exit condition (go-ultimate)
    case <-e.changed:
        if !dirty {
            oldest = time.Now()
            dirty = true
        }
        debounce.Reset(e.dumpDebounce) // coalesce: 50 ms quiet window
    case <-debounce.C:
    case <-time.After(remainingTo(oldest.Add(e.maxWait))): // bounded: dump starts
        // no later than oldest+1s even under continuous writes (D-04)
    }
    if dirty {
        e.runDump(ctx) // snapshot → encode → port.Save → publish (Pattern 4)
        dirty = e.stillDirty() // writes that landed during the dump re-arm immediately
    }
}
```

After a failed dump (D-05): current waiters got the error; the state stays dirty; the next attempt starts on the next change or the debounce timer — no tight retry loop. Whether the debounce timer re-arms after failure without a new change is a planner call (D-05's wording permits a timer-triggered next attempt).

### Pattern 6: Snapshot create/load (PERS-05, D-08, D-09, D-10)

**What:** Create = flush current state to disk, then byte-copy the finished dump under a validated name. Load = decode with the same codec, replay as ordinary writes.
**When to use:** Snapshot operations (library-level this phase).

Restore replay sketch — the Phase 2 store already provides the exact semantics D-09 needs:

```go
for _, key := range slices.Sorted(maps.Keys(records)) { // deterministic replay order
    rec := records[key]
    if rec.Deleted {
        if _, _, err := e.store.Delete(key); errors.Is(err, storage.ErrNotFound) {
            continue // missing/tombstoned: nothing to beat — Delete is a no-op there anyway
        } else if err != nil { return err }
        continue
    }
    if _, _, err := e.store.Set(key, rec.Data); err != nil { return err } // fresh version: wins LWW
}
```

[VERIFIED basis — internal/app/storage/store.go:179-183: "Delete writes a fresh-version tombstone for key that merges like any write; a missing key and an already-tombstoned key are both `ErrNotFound` and write nothing, so no version is burned on a no-op."] Snapshot names MUST be validated before touching the filesystem (Security §V5).

### Anti-Patterns to Avoid

- **Hand-rolled second checksum:** any CRC32 polynomial/table other than `storage.RecordChecksum` breaks D-07's "uniform with the replication integrity story" and fails round-trip against Phase 2 records.
- **`bufio.Scanner` over dump lines:** default 64 KiB line limit; a 60 KiB value base64-encodes to ~80 KiB — silently truncated lines. Use `json.Decoder` streaming.
- **Two concurrent dumps:** racing temp files and rotations corrupt the `.prev` chain. One dumper goroutine, period.
- **Holding the engine mutex across adapter I/O:** blocks write-path dirty-marking, and under synctest a mutex is not durably blocking → bubble deadlock detection trips [CITED: pkg.go.dev/testing/synctest].
- **`sync.Cond` for waiters:** no timeouts, no context, no error broadcast [CITED: pkg.go.dev/sync].
- **Writing the dump pretty-printed as one object:** kills per-record greppability (D-06) and makes 10k-key diffs useless.
- **Timestamp/counter fields in the header:** breaks byte-stability across generations of the same state (D-06).
- **Temp file via `os.CreateTemp("", …)`** (system temp dir): rename onto the dump path can cross filesystems and fail `EXDEV` — temp must live in the target directory (PERS-02 says so explicitly).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Per-record integrity | A dump-specific checksum | `storage.RecordChecksum(key, ver, data, deleted)` | Same CRC32-IEEE domain as replication (D-07); already property-tested |
| JSONL parsing | Line splitting + `json.Unmarshal` per line | `json.NewDecoder` value loop | Handles concatenated values, truncation (`io.ErrUnexpectedEOF`), no line-length limits |
| Broadcast to many waiters | Callback/event system (forbidden by Phase 2 D-01) | Per-waiter channel closed by the publisher | Results-not-callbacks; channel close is the doc-blessed broadcast |
| Deterministic timer tests | Sleep-and-margin tests | `testing/synctest` bubble + fake port | Exact deadline assertions; go-ultimate-mandated; free goroutine-leak detection |
| Restart watermark | ad-hoc "max version" scan | max `l` + max counter at that `l` via `version.Unpack` | VERS-04 wording is authoritative (CONTEXT §Specifics); `NewClock` takes both parameters |
| Atomic file replacement | O_TRUNC + write in place, or rename without fsyncs | Canonical temp→fsync→rotate→rename→dir-fsync sequence | Crash windows: in-place writes tear; unsynced renames vanish |

**Key insight:** every hard part of this phase (checksum domain, version order, watermark seeding, merge-on-load) already exists in Phase 2 code — the phase's real work is composition plus the two genuinely new mechanisms (atomic file sequence, watermark/waiter engine).

## Common Pitfalls

### Pitfall 1: `[16]byte` does not base64-encode
**What goes wrong:** `version.RecordVersion` is `[16]byte`; the v1 `encoding/json` API encodes it as a JSON array of 16 numbers, not a string — "`Array and slice values encode as JSON arrays, except that `[]byte` encodes as a base64-encoded string`" [CITED: pkg.go.dev/encoding/json].
**Why it happens:** Assumption that "bytes are base64" covers arrays; it covers slices only.
**How to avoid:** Convert explicitly — `hex.EncodeToString(rec.Version[:])` (recommended: greppable UUID-shaped) or `[]byte(ver[:])` (base64).
**Warning signs:** Dump lines showing `"record_version":[15,178,...]`; grep for a version fails.

### Pitfall 2: Debounce starvation under sustained bursts
**What goes wrong:** Every write resets the 50 ms timer; with writes every ~1 ms (1000/s) the quiet window never arrives → no dump ever starts → waiters hang (exactly what success criterion 3 forbids).
**Why it happens:** Naive debounce resets on every change with no floor.
**How to avoid:** D-04's deadline: dump starts no later than oldest-unconfirmed-change + 1 s regardless of new changes (Pattern 5).
**Warning signs:** Burst test's oldest waiter latency grows without bound; dump count ≪ expected under load.

### Pitfall 3: Lost durability from skipped syncs or wrong temp location
**What goes wrong:** rename survives but data doesn't (no pre-rename fsync), or the rename itself vanishes (no directory fsync), or `os.TempDir()` temp → `EXDEV` rename failure.
**How to avoid:** Follow Pattern 3's order verbatim; temp file in the target directory (PERS-02 wording).
**Warning signs:** kill −9 recovery tests losing the newest generation; sporadic `*LinkError` on rename.

### Pitfall 4: Waiter deadlock/lifecycle bugs
**What goes wrong:** waiters registered after a failed dump never resolve; engine goroutine leaks past test end; `Set` blocks forever on Close.
**How to avoid:** every waiter resolves exactly once (success close, error close, or ctx cancel); the dumper goroutine has a documented exit (ctx/done) — go-ultimate requires the API contract to document the lifecycle; synctest fails the test if a bubble goroutine survives.
**Warning signs:** `-race`/synctest hangs; `TestMain` leaks.

### Pitfall 5: CreateTemp permissions and temp litter
**What goes wrong:** `os.CreateTemp` "creates the file with mode 0o600 (before umask)" [CITED: pkg.go.dev/os] — the dump inherits 0600 (operator readability of the "human-readable dump" becomes a chmod decision); failed saves litter `dump-*` temp files.
**How to avoid:** Decide the dump mode explicitly (see Open Question 3); `defer` remove-on-error for the temp file.
**Warning signs:** `ls -l` shows 0600 dumps operators can't read; temp files accumulate after injected failures.

### Pitfall 6: Real I/O inside a synctest bubble
**What goes wrong:** The bubble's clock only advances when every goroutine is durably blocked; file I/O and mutex acquisition are NOT durably blocking → deadlock panic [CITED: pkg.go.dev/testing/synctest].
**How to avoid:** Engine tests use a fake in-memory dump port inside the bubble; the real adapter is tested outside bubbles with `t.TempDir()` and real time. Also: `T.Parallel`/`T.Run` must not be called on the inner `*testing.T`.
**Warning signs:** `synctest` deadlock panics pointing at adapter reads.

### Pitfall 7: Misreading the fallback chain's three failure classes
**What goes wrong:** Confusing (a) missing/unreadable file → next generation, (b) unparseable generation → next generation, (c) bad-checksum records → discard record, KEEP the generation. Treating (c) as (b) discards a whole good generation for one torn record; treating (b) as (c) loads garbage.
**How to avoid:** Encode the distinction in the port/codec split (D-02): parse failure = generation-level; checksum = record-level. Property-test both classes separately.
**Warning signs:** Single corrupted record nukes the map to empty start.

### Pitfall 8: Load path that can panic or accept absurd input
**What goes wrong:** Hand-edited/hostile dump: giant lines, duplicate keys, unknown format version, values over `ValueMaxBytes`. A panic in decode takes the node down at startup.
**How to avoid:** `version.Unpack` is already total [VERIFIED: version.go:50-58 — "It is total: pure masking arithmetic over every possible bit pattern, it never panics"]; keep decode error-returning everywhere; decide unknown-format-version = unreadable generation (recommended) and oversized-value = discarded record (Open Question 5); `Merge` rejects oversized values with `ErrValueTooLarge` [VERIFIED: store.go:196-199].

### Pitfall 9: Obsolete timer-drain cargo cult
**What goes wrong:** Copying pre-1.23 `Stop`-drain idioms adds dead code and confuses reviewers.
**Why:** Go 1.23+ guarantees "for any call to a Reset or Stop method, no stale values prepared before that call will be sent or received after the call" and GC of unreferenced timers [CITED: go.dev/doc/go1.23]; sup-kv's `go 1.27.1` activates this.
**How to avoid:** Plain `Reset` in the debounce loop; no drain selects.

### Pitfall 10: Snapshot path traversal
**What goes wrong:** `name = "../../etc/cron.d/x"` escapes `<dump_path>.snapshots/`.
**How to avoid:** Strict allow-list validation (no separators, no `..`, e.g. `[A-Za-z0-9._-]{1,N}`) at the entry point — see Security Domain.
**Warning signs:** any snapshot name reaching `filepath.Join` unvalidated.

## Code Examples

### Watermark computation (VERS-04 — full watermark, not max l alone)

```go
// Source: VERS-04 + CONTEXT §Specifics ("the dump loader computes the FULL
// watermark — max l plus max counter at that l"); NewClock signature verified:
//   func NewClock(now func() time.Time, initialL int64, initialC uint16, driftWarn time.Duration) *Clock
//   [VERIFIED: internal/app/clock/clock.go:63]
func Watermark(records map[string]storage.Record) (initialL int64, initialC uint16) {
    for _, rec := range records {
        l, c, _ := rec.Version.Unpack()
        switch {
        case l > initialL:
            initialL, initialC = l, c
        case l == initialL && c > initialC:
            initialC = c
        }
    }
    return initialL, initialC // empty map → (0,0) → NewClock starts from the wall clock
}
```

Compute from VALID (checksum-passing) records only — a torn record's version is untrusted.

### Startup wiring: load → watermark → clock → store

```go
// Source: phase 2 seams; store.Merge preserves foreign versions and recomputes
// checksums on install [VERIFIED: store.go:195-205, 230-238].
dump, ok, err := dumpStore.Load(codec.Parseable) // chain main → .prev → empty (D-02)
records, discarded, err := codec.Decode(dump)     // per-record checksum discard (PERS-02)
initialL, initialC := persist.Watermark(records)
clk := clock.NewClock(time.Now, initialL, initialC, driftWarn) // VERS-04
store := storage.NewStore(nodeID, clk, cfg)
for key, rec := range records {
    _, _, _ = store.Merge(key, rec) // versions preserved; watermark already dominates
}
```

### Burst confirmation test under synctest (PERS-03, success criterion 3)

```go
// Source: go-ultimate testing.md ("Use testing/synctest for anything involving
// time or goroutine coordination") + pkg.go.dev/testing/synctest.
// Fake port records virtual save-times; no real I/O inside the bubble (Pitfall 6).
func TestEngine_burstEveryWriteConfirmedBounded(t *testing.T) {
    t.Parallel()
    synctest.Test(t, func(t *testing.T) { // NO t.Parallel/T.Run in here
        e, fake := newTestEngine(t, Debounce: 50*time.Millisecond, MaxWait: time.Second)
        ctx := startDumper(t, e)
        for i := 0; i < 1000; i++ { // ~1 write/ms virtual, sustained 1 s
            go writeAndTrackConfirmTime(t, e, fake) // engine.Set blocks; fake records time
            synctest.Sleep(time.Millisecond)        // deterministic virtual cadence
        }
        synctest.Wait() // let the final dump + waiters settle
        // Assert: all 1000 confirmed; firstDumpStart <= oldestWrite+1s (D-04);
        // fake.saves << 1000 (debounce coalesced); dumps serialized.
    })
}
```

### Kill −9 recovery simulation (success criterion 2, library level)

Real SIGKILL harnesses are Phase 6 (TEST-04); the library-level equivalent covers every crash residue the chain must absorb:

```go
// t.TempDir()-driven table test in the adapter package (real fs, real time):
//   {"main truncated mid-line",   truncate(dump, n-40)}                    → serves .prev
//   {"main garbage",              write(dump, "not json at all")}          → serves .prev
//   {"main missing",              remove(dump)}                            → serves .prev
//   {"both missing",              remove(dump, prev)}                      → empty start, ok=false
//   {"one torn record in main",   flipOneByte(dataOf(recordK))}            → serves main,
//                                                                   recordK discarded, others load
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `sync.Cond` broadcast | Channel-close broadcast for waiter fan-out | Long-standing Go guidance | Error fan-out (D-05) and ctx support need channels; `Cond` can't |
| Timer Stop/Reset drain idiom | Safe Reset, GC-able timers | Go 1.23 | Debounce loop simplifies; old idiom is dead code on `go 1.27.1` |
| Sleep-based concurrency tests | `testing/synctest` fake-clock bubbles | Go 1.25 (stable) | D-04's 1 s bound is an exact, instant assertion |
| json v1 engine | v2 engine default under the v1 API | Go 1.27 | Keep v1 API: on-disk stability; v2 is stricter (duplicate keys, UTF-8) but machine-written dumps don't care [CITED: .claude/CLAUDE.md §Technology Stack] |

**Deprecated/outdated:** nothing in this phase's stdlib surface is deprecated; do not adopt `encoding/json/v2` API mid-project (project decision).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | D-06's "One JSON object: a header…, then the records one per line" means JSONL: header object on line 1, one record object per line after (not a single nested object). Reading forced by "greppable per record", but the header-on-its-own-line detail is interpretation | Patterns 2, Examples | Low — any one-object-per-line layout satisfies PERS-01; planner should confirm header line shape |
| A2 | The engine's write path blocks inside `Set`/`Delete`/`SetIfAbsent` until confirmed (rather than returning a confirm-handle callers await) — implied by PERS-04 "a write returns OK only after a dump containing it completed"; API shape itself is planner's discretion | Patterns 4–5 | Low-medium — changes engine method signatures; both satisfy the requirement |
| A3 | Dump/snapshot file mode: 0600 (CreateTemp default) kept for values' privacy, accepting operators read as root/owner; alternative 0644 favors the "inspectable with standard tools" principle | Pitfall 5, Open Q 3 | Low — one chmod decision; confirm with user if operators are non-owner |
| A4 | Unknown `format_version` in a dump = unreadable generation (chain falls back), not best-effort load | Pitfall 8 | Low — forward-compat policy; planner/user confirm |
| A5 | Oversized decoded record (`len(Data) > ValueMaxBytes`, e.g. after a config lower) = discarded record (mirrors checksum discard), not a load failure | Pitfall 8, Open Q 5 | Low — rare; affects only loads after config shrink |
| A6 | Recommended port shape `Load(parse func([]byte) error)` (parse-oracle) — D-02-literal; the simpler two-generations-return alternative is acceptable if the planner prefers a dumber seam | Pattern 1, Open Q 1 | Medium — interface is D-01/D-02 "costly reversibility"; decide once, deliberately |
| A7 | Package names `internal/app/persist` and `internal/persistfile` as concrete suggestions (explicitly planner's discretion per CONTEXT) | Project Structure | None — naming only |
| A8 | Debounce timer MAY re-arm after a failed dump without a new change (one attempt per debounce interval), reading D-05's "or the debounce timer" literally | Pattern 5 | Low — retry cadence detail; either reading satisfies "no auto-retry loops" |

## Open Questions (all five RESOLVED — settled by plans 03-01/03-02, back-annotated 2026-11-03)

1. **Fallback-chain seam shape (D-02-literal vs simpler)**
   - What we know: D-02 assigns chain resolution to the adapter; "unparseable" requires the codec.
   - What's unclear: parse-oracle callback (A6) vs the port returning both generations' bytes with the core iterating.
   - Recommendation: parse-oracle `Load` — one place owns chain logic; engine-test fakes implement the interface trivially either way. Decide in planning; it's the costly-to-reverse seam.
   - **(RESOLVED)** Parse-oracle Load, per the recommendation (A6 accepted): `port.DumpStore.Load(parse func([]byte) error) (dump []byte, ok bool)` with the engine supplying `codec.Parse`, keeping the adapter record-blind. Settled in plan 03-01 Task 1 (reversibility note and the port interface action).
2. **Snapshot file `.prev`/rotation policy**
   - What we know: D-10 says byte-copy under the name; overwrite semantics unspecified.
   - Recommendation: reuse the atomic write; no `.prev` for snapshots (overwrite in place atomically). Low stakes.
   - **(RESOLVED)** Atomic overwrite in place, no `.prev` for snapshots: `saveAtomically(..., rotate=false)` — snapshots are operator-named recovery points, not crash generations; the dump's own `.prev` chain is the crash mechanism. Settled in plan 03-01 Task 1 (snapshot.go action); ratified in plan 03-02 flagged assumption 2.
3. **Dump/snapshot file mode (A3):** 0600 vs 0644 — one-line decision, confirm with user if operator-readability as non-owner matters.
   - **(RESOLVED)** 0600 for dump and snapshot files (CreateTemp default kept — values privacy over non-owner readability) and 0700 for the `.snapshots` directory; greppability is preserved because grep runs as the owner. Settled in plan 03-01 Task 1; carried as flagged assumption 2 there for phase-verification ratification.
4. **Restore replay and confirmation:** replay through the ordinary blocking write path (each restored write confirmed; debounce coalesces to a few dumps) vs an internal batch path with one final dump. Recommendation: ordinary path — simpler, PERS-04-consistent; revisit only if restore benchmarks hurt (they won't at v1 scale).
   - **(RESOLVED)** Semantics-level reading adopted: each key replays through the store's ordinary write path (`store.Set`/`store.Delete`, fresh versions — never `Merge`), but the whole batch is confirmed by ONE final flush rather than per-write blocking (identical per-key semantics, stronger batch confirmation; sequential per-write blocking would serialize N writes at one debounce each). Settled in plan 03-02 Task 1 (Restore action); carried as flagged assumption 1 there.
5. **Oversized-record-on-load policy (A5).** Recommendation: discard-and-count, mirroring PERS-02's checksum class.
   - **(RESOLVED)** Discard-and-count, mirroring the checksum class: decode-side records whose `Data` exceeds `maxRecordBytes` (61440, the docs/IDEA.md design envelope) are discarded and counted; Open additionally skips `storage.ErrValueTooLarge` on merge so a config-shrunk limit never aborts startup. Settled in plan 03-01 Task 1 (codec.go action).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | everything | ✓ | 1.27.3 linux/amd64 (>= pinned 1.27.1) [VERIFIED: local `go version`] | — |
| Module build | adapter + engine compile | ✓ | `go build ./...` OK [VERIFIED: local run] | — |
| Real filesystem for adapter tests | PERS-02 tests | ✓ | `t.TempDir()` (go-ultimate standard) | — |
| `testing/synctest` | engine timing tests | ✓ | stdlib, stable since 1.25 [CITED: pkg.go.dev] | — |
| golangci-lint v2.15.0 gate | `make lint` | ✓ | `go tool golangci-lint` via go.mod tool directive [VERIFIED: go.mod] | — |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** none.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (+ `testing/synctest`), `-race` mandatory [VERIFIED: Makefile] |
| Config file | none needed — existing Makefile targets + CI (`.github/workflows`) |
| Quick run command | `go test -race ./internal/app/persist/... ./internal/persistfile/...` (final names per planner) |
| Full suite command | `make lint && make test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PERS-01 | Round-trip: full map incl. tombstones; greppable JSONL; byte-stable; base64 Data | property (seeded via `testutil.NewRNG`) + golden | `go test -race -run TestCodec ./internal/app/persist/` | ❌ lands in-phase |
| PERS-02 | Atomic write order, `.prev` rotation, chain main→.prev→empty, bad-checksum discard | integration (real `t.TempDir()`) + property (codec) | `go test -race ./internal/persistfile/` | ❌ in-phase |
| PERS-03 | Debounce coalesces; bounded start (oldest+1 s); 1000 writes/s all confirmed | unit under `synctest` (fake port) | `go test -race -run TestEngine_burst ./internal/app/persist/` | ❌ in-phase |
| PERS-04 | OK only after covering dump; failure → error to all waiters, write stays in memory; restart recovers | unit (synctest) + integration (restart via reload) | `go test -race -run 'TestEngine_confirm\|TestEngine_dumpFail' ./internal/app/persist/` | ❌ in-phase |
| PERS-05 | Snapshot create forces dump then copies bytes; load replays fresh-version writes; absent keys untouched; name validation | unit + integration | `go test -race -run TestSnapshot ./...` | ❌ in-phase |
| VERS-04 (half) | Watermark = max l + max counter at that l; seeds NewClock; post-restart versions dominate dump | property (seeded) | `go test -race -run TestWatermark ./internal/app/persist/` | ❌ in-phase |

### Sampling Rate
- **Per task commit:** `go test -race ./internal/app/... ./internal/persistfile/...` (affected packages)
- **Per wave merge:** `make lint && make test` (full gate, same as CI)
- **Phase gate:** full suite green before `/gsd:verify-work`

### Wave 0 Gaps
None — existing test infrastructure (Makefile, CI, `testutil` generators, `-race`) covers all phase requirements; new test files land with their packages. Note the two test-file conventions that are NOT yet exercised anywhere: synctest bubbles (inner `t` forbids `Parallel`/`Run`) and real-fs adapter tests — both are pure additions, no framework work.

## Security Domain

ASVS L1 (`security_enforcement: true`, `security_asvs_level: 1`, `security_block_on: high`).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | none this phase (trusted-segment trust model, docs/IDEA.md §Trust model) |
| V3 Session Management | no | — |
| V4 Access Control | yes (file level) | Dump/snapshot file permissions (0600 vs 0644, A3); snapshots dir mode |
| V5 Input Validation | yes | Snapshot-name allow-list (no separators/`..`); dump decode hardened against arbitrary bytes (error-returning, `Unpack` total, size-bounded, unknown-format-version rejected); record discard classes |
| V6 Cryptography | no | CRC32 is explicitly NOT cryptographic — integrity only within the accepted trust model (D-07); never present it as tamper protection |
| V8 Data Protection | partial | Dump stores values in plaintext — accepted, documented trust model; permissions are the only at-rest control |
| V12 File Execution | no | — |

### Known Threat Patterns for Go file-persistence layer

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal via snapshot name (write/read outside `.snapshots/`) | Tampering, Elevation | Strict charset allow-list `[A-Za-z0-9._-]`, reject separators and `..` before any path join |
| Hostile/corrupt dump at startup (operator edit, torn writes) | Tampering | Parse-never-panic; per-record checksum discard; unknown format version → unreadable generation; value-size bound |
| Torn/crashed dump generations | Tampering (accidental) | Atomic write + `.prev` chain (the phase's core purpose) |
| Local symlink swap on dump path | Tampering | Trusted-host trust model; `CreateTemp`+`rename` replaces symlinks atomically rather than following them — document, don't defend beyond |
| Checksum confusion | Spoofing | CRC32 detects corruption, not forgery — keep the trust-model wording in docs |

## Sources

### Primary (HIGH confidence)
- In-repo (Read this session): `internal/app/storage/store.go` (Record, RecordChecksum CRC32-IEEE domain, Records, Merge/Delete semantics), `internal/app/storage/errors.go` (sentinel pattern), `internal/app/clock/clock.go` (NewClock signature + seeding contract), `internal/app/version/version.go` (Pack/Unpack/Compare, Unpack totality), `internal/app/internal/testutil/{gen,rng}.go` (seeded generators, printed-seed replay), `internal/port/doc.go`, `.golangci.yml` (depguard rules verbatim), `Makefile`, `go.mod`, `cmd/sup-kv/main.go`, `docs/IDEA.md`, `03-CONTEXT.md`, `02-CONTEXT.md`, `.planning/REQUIREMENTS.md`, `.planning/STATE.md`, `.claude/CLAUDE.md`
- pkg.go.dev/encoding/json — []byte→base64 vs [16]byte→number-array; Encoder.Encode newline; SetEscapeHTML default; map-key sorting (fetched 2026-11-03)
- pkg.go.dev/os — CreateTemp 0600 + caller-remove contract; Rename atomic-on-Unix; File.Sync guarantee (fetched 2026-11-03)
- pkg.go.dev/sync — Cond broadcast/Wait rules, no-timeout, channels-recommended note (fetched 2026-11-03)
- pkg.go.dev/testing/synctest — bubble semantics, durably-blocking list, fake clock, restrictions (fetched 2026-11-03)
- go.dev/doc/go1.23 — timer Reset/Stop guarantees, GC-able timers (fetched 2026-11-03)
- go-ultimate skill v0.15.0 references — engineering-policy.md §Concurrency, testing.md §Testing concurrent code, architecture.md §Adapter discipline

### Secondary (MEDIUM confidence)
- WebSearch (2026-11-03): community crash-consistency consensus on temp→fsync→rename→dir-fsync — [Atomic writes with respect to unexpected power offs (Stack Overflow)](https://stackoverflow.com/questions/35060793/), [How to do atomic file write & fsync to stable storage on Linux (Stack Overflow)](https://stackoverflow.com/questions/72891357/), [Without fsync() before rename() (Hacker News)](https://news.ycombinator.com/item?id=31047131), [Durability of writes between fsync() calls (Stack Overflow)](https://stackoverflow.com/questions/44543283/), [Atomic File Replacement on NTFS Is Not rename + fsync](https://luth.io/blog/2026/08/ntfs-atomic-file-replacement-not-rename-fsync) (Windows caveat — irrelevant to Linux/macOS targets)
- Local toolchain probes: `go version go1.27.3`, `go build ./...` OK

### Tertiary (LOW confidence)
- None — no claim in this document rests on an unverified single web source.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — stdlib only; every contested behavior verified against pkg.go.dev this session; zero new dependencies eliminates the legitimacy surface entirely
- Architecture: HIGH — every seam is locked by CONTEXT decisions D-01..D-10 and machine-enforced by depguard; the only open architecture choice is the Load-seam shape (Open Question 1), presented with a recommendation
- Pitfalls: HIGH — eight of ten pitfalls verified against primary sources (json/os/sync/synctest/go1.23 docs) or in-repo code read this session; the remaining two (starvation math, waiter lifecycle) follow directly from locked decisions D-04/D-05

**Research date:** 2026-11-03
**Valid until:** 2026-12-03 (stable stdlib domain; decisions are locked and do not expire)
