# sup-kv — engine review (version, clock, storage, persist, persistfile, port, metrics)

Reviewed at `master` d6b2a75, Go 1.27.4, darwin/arm64 (Apple M3 Max, NVMe).
Scope read line by line: `internal/app/{version,clock,storage,persist}`, `internal/persistfile`, `internal/port`, `internal/metrics`.
Callers (`cmd/sup-kv`, `internal/grpcapi`, `internal/app/repl`, `internal/udprepl`) were read only to see how the scope is driven; findings that cross that boundary are marked **cross-boundary**.

Baseline: `go test -race` is green for all six scoped packages; the repository was not modified.
Every repro below lives in a copy of the repo at
`evidence/repro/engine`
(files named `review_*_test.go`). A repro "fails" when the defect is present; the quoted output is from those runs.
Spec references (`README:NNN`) are line numbers in `SPEC.md`.

Status vocabulary: **Verified** = reproduced by a test here, or unambiguous from the quoted code. **Suspected** = the code path is clear but the end effect was not executed (stated per finding).

---

## 1. Architecture assessment

**What holds up**

- `version`, `clock`, `storage` are small, pure and correct against the algorithm they claim. Pack/Unpack round-trips; byte order equals `(l, c, NodeID)` for well-formed versions; Tick/Update match Kulkarni's rules and the spec's four cases; counter overflow bumps `l`; the restart watermark is `(max l, max c at that l)`, not `max l` alone.
- Store locking is simple and right: one `RWMutex`, the clock ticked under the write lock so version order equals commit order, data cloned on the way in and out, stored records never mutated in place.
- The confirmation discipline in `persist.Engine` is sound. A waiter is registered only after its record is installed, the waiter list is swapped under the engine mutex before the snapshot is taken, so "confirmed" implies "contained in the bytes handed to `Save`". I found no path that confirms on error, no stranded waiter, and no deadlock in the `Run`/`flush`/exit-sweep interplay.
- The happy-path atomic write is correct: temp file in the target directory, data fsync, rename, directory fsync, `Save` serialized against snapshot copy. No crash point in a healthy sequence leaves both `main` and `.prev` invalid.
- The exporter emits valid 0.0.4 text (HELP/TYPE once per family, sorted, correct escaping, `ParseFloat`-compatible values) and reads only atomics.

**What does not**

- **The store's read API is one method too narrow, and everything pays for it.** `Records()` — a deep copy of every value — is the only tombstone-visible read, so it is used for the dump, the sync listing, every received sync chunk, a *single-key* `DataRequest`, `List`, the Watch snapshot and `Restore`. Cost is O(total bytes) per call (ENG-04, ENG-12).
- **`persist.Engine` carries five jobs**: startup loader and composition root (`Open` builds the clock and the store), store facade, dump scheduler, confirmation tracker, snapshot/restore orchestrator. The visible costs: the Watch tap has to be threaded through `persist.Open` and re-attached later (`hub.attach`), `Restore` writes around the replication notify path, and the replication core has no way to hand a version to the clock (ENG-03).
- **`port.DumpStore` hides what the core needs in order to decide.** `Load(parse)` inverts control (the adapter drives the fallback policy with a caller oracle), returns no diagnostics, and `Open` discards even the `ok` it does return. `CreateSnapshot` means "copy whatever file is at the main path", which bakes a file-level assumption into the port (ENG-02, ENG-06).
- **Failures are computed and then dropped at the boundary.** `internal/app` logs nothing by design ("results only"), but `Open` returns a bare count, `Run` discards dump errors, `ErrDumpFailed` does not wrap its cause and `Load` swallows every error. The durability subsystem can degrade with no log line and no metric (ENG-02, ENG-08).
- **Limits are duplicated as constants instead of flowing from configuration**: 61440 appears independently in `storage`, `persist/codec.go` and `main.go` (ENG-01).
- **Layer enforcement is a stdlib deny-list.** It stops `os`/`net` imports in `internal/app` but not `internal/app -> adapter` or `internal/port -> adapter` imports, and `internal/metrics` imports `internal/udprepl` for one struct (ENG-25).
- **Defensive code and comments have outgrown the behaviour.** `lastLocal`/cover and the "above cover" branch are unreachable; `persist.Parse` and the darwin fsync shim are redundant; several doc comments are now false (ENG-18, ENG-23, ENG-24). The doc comments are decision logs (D-xx, WR-xx, plan numbers) more than contracts, which is why they drift.

---

## 2. Findings

| ID | Severity | Status | Title |
|----|----------|--------|-------|
| ENG-01 | Critical | Verified | With `--value-max-bytes` above 61440, confirmed writes are discarded at the next restart |
| ENG-02 | High | Verified | Unreadable, unsupported or corrupt dump silently becomes an empty/stale start, and the next dumps destroy the files |
| ENG-03 | High | Verified | HLC is never advanced by `DataSync`; a write accepted after the gate opens is stamped below a known peer version (cross-boundary) |
| ENG-04 | High | Verified | `Records()` is the only bulk/tombstone-visible read: O(whole state) deep copy per `DataRequest` and per sync chunk |
| ENG-05 | Medium | Verified | One damaged byte in one record rejects the whole generation; a dump truncated at a line boundary is accepted as complete |
| ENG-06 | Medium | Verified | After a fallback start the rejected main is rotated over the last good `.prev`; the engine is "clean" while disk differs from memory |
| ENG-07 | Medium | Verified | Equal version with different payload is a silent no-op; zero filler instead of the spec's random tail makes it reachable |
| ENG-08 | Medium | Verified | Failed dump: retried every 50 ms forever, cause never logged or returned, failed write never broadcast |
| ENG-09 | Medium | Verified | Debounce is a resettable quiet window: median confirmation 0.5 s under sustained writes; spec bound is `dump_debounce` + dump time |
| ENG-10 | Medium | Verified | `Restore` does not delete keys absent from the snapshot, reports success on a partly discarded snapshot, and bypasses replication notify |
| ENG-11 | Medium | Verified | Drift signal fires after any idle period > `drift_warn` and stays silent on the event that absorbs a far-future version |
| ENG-12 | Medium | Verified | One dump cycle allocates ~5x the live state and blocks writers for the copy (3.4 GiB, 74 ms stall at 10k x 60 KiB) |
| ENG-13 | Medium | Verified (limit absent) / Suspected (effect) | No key-size bound: a record that cannot fit one datagram is confirmed and can never replicate |
| ENG-14 | Medium | Verified | `SetIfAbsent` returns `AlreadyExists` for a tombstoned key; the spec says it must write |
| ENG-15 | Low | Verified | Blocking engine calls take no context: a hung fsync blocks writers, snapshot, restore and shutdown forever |
| ENG-16 | Low | Verified | `process_resident_memory_bytes` reports peak RSS (`ru_maxrss`) |
| ENG-17 | Low | Verified | Orphaned temp files are never collected; new directories are not made durable |
| ENG-18 | Low | Verified | `fsync_darwin.go` re-implements what `os.File.Sync` already does, on a false premise, and loses EINTR/ENOTSUP handling |
| ENG-19 | Low | Verified (store) / Suspected (wire) | Version bits are not validated at trust boundaries: one malformed version poisons ordering or exhausts the domain |
| ENG-20 | Low | Verified | "Conflict" is every overwrite and every stale redelivery; metric HELP and `Merge` doc say otherwise |
| ENG-21 | Low | Verified (package level) | A key that is not valid UTF-8 is accepted, mangled by the codec and discarded at reload |
| ENG-22 | Low | Verified | Records skipped by a shrunk value limit vanish without count or log, then from disk |
| ENG-23 | Info | Verified | Dead and over-defensive code; startup load goes through `Merge` |
| ENG-24 | Info | Verified | Stale or false comments |
| ENG-25 | Info | Verified | Depguard boundary does not enforce dependency direction |
| ENG-26 | Info | Verified | Checksum domain has no length prefixes (theoretical only) |

---

### ENG-01 — Critical — Verified
**With `--value-max-bytes` above 61440, confirmed writes are discarded at the next restart**

`internal/app/persist/codec.go:25,139-141`, `cmd/sup-kv/main.go:311,417`

```go
// codec.go
const maxRecordBytes = 61440
...
	if len(line.Data) > maxRecordBytes {
		return storage.Record{}, false
	}
// main.go — no upper bound anywhere
flag.IntVar(&cfg.valueMaxBytes, "value-max-bytes", envNum(envValueMaxBytes, defaultValueMaxBytes), ...)
```

`Encode` has no size check, `Decode` has a hard-coded one, and the store limit is a free tunable. Precondition: the operator raises the documented tunable above the default (62000 still fits one UDP datagram).

Scenario: `--value-max-bytes=65000`; `Set(big, 62000 bytes)` returns OK after the dump. Restart with the same configuration: the record is dropped by `Decode`, the only trace is `discarded 1 undecodable dumped records at startup`. The engine is clean, so the next dump writes a main without the key and the one after that rotates it out of `.prev`. A live peer re-delivers it by sync, and it is lost again at every restart; when both nodes restart together it is gone.

```go
func TestReview_valueMaxAboveCodecEnvelopeLosesConfirmedWrite(t *testing.T) {
	dumpPath := filepath.Join(t.TempDir(), "dump.json")
	cfg := storage.Config{ValueMaxBytes: 65000}
	eng, _ := persist.Open(1, time.Now, time.Minute, persistfile.New(dumpPath), cfg, persist.Config{})
	stop := startRun(t, eng)
	if _, _, err := eng.Set("big", bytes.Repeat([]byte("x"), 62000)); err != nil {
		t.Fatalf("Set(big) = %v, want nil (confirmed)", err)
	}
	stop()
	eng2, discarded := persist.Open(1, time.Now, time.Minute, persistfile.New(dumpPath), cfg, persist.Config{})
	if _, err := eng2.Get("big"); err != nil {
		t.Errorf("CONFIRMED WRITE LOST: discarded=%d err=%v", discarded, err)
	}
}
// output: CONFIRMED WRITE LOST: Set(big) returned nil, restart discarded it (discarded=1, err=key not found)
```

Fix: remove the size check from `decodeRecord` (the store limit is the authority; `Open` already has `mergeRecords` for it) or pass the configured limit into `Decode`. Independently, validate `--value-max-bytes` at startup against what one datagram can carry and refuse larger values. See ENG-22 for the sibling case.

Acceptance: the test above passes for any `ValueMaxBytes` the binary accepts; `finalizeConfig` rejects a value that cannot be replicated in one datagram with exit code 2.

---

### ENG-02 — High — Verified
**Unreadable, unsupported or corrupt dump silently becomes an empty/stale start, and the next dumps destroy the files**

`internal/persistfile/dumpfile.go:74-86`, `internal/port/dump.go:30`, `internal/app/persist/engine.go:141-144`, `cmd/sup-kv/main.go:416-422`

```go
// dumpfile.go
for _, candidate := range []string{f.path, f.path + prevSuffix} {
	data, err := os.ReadFile(candidate)
	if err != nil {
		continue            // ENOENT, EACCES, EIO, EMFILE: all the same
	}
	if parse(data) != nil {
		continue            // truncated, syntax fault, errUnsupportedFormat: all the same
	}
	return data, true
}
return nil, false
// engine.go
dumps.Load(decode)          // both return values discarded
if records == nil {
	records = make(map[string]storage.Record)
}
```

"No dump exists" (fresh node) is indistinguishable from "a dump exists and could not be used". `main.go` logs only `discarded > 0`. There is no log line and no metric for "fell back to `.prev`" or "both generations unusable", which the spec requires (README:330-331 discarded records are logged; README:501-503 empty start [spec text]).

Scenarios, all ending the same way:
1. Main dump restored from backup with the wrong owner/mode (EACCES), or a transient EIO at boot.
2. Rollback to an older binary after a future format bump: both generations return `errUnsupportedFormat`.
3. One bad byte in each generation (see ENG-05 for how little it takes).

The node starts empty, serves `NotFound`, opens for writes after `startup_gate`, and the first two dumps rotate the intact-but-unusable files out of existence. On a single node, or when both nodes start cold, every confirmed write is gone and nothing was reported. With a live peer the data flows back. By code reading (not executed, and cross-boundary): the write gate opens on the first sync chunk (`gate.go:137-139`), so for up to N/`sync_response_rate` seconds a `create_if_absent` on the recovering node sees "absent" and stamps build-time defaults with fresh versions that can beat the peer's user values.

```go
func TestReview_unreadableMainIsSilentEmptyStartThenDestroyed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dump.json")
	fs := persistfile.New(path)
	_ = fs.Save([]byte("GOOD 10000 confirmed records"))
	_ = os.Chmod(path, 0o000)
	fs2 := persistfile.New(path)
	loaded, ok := fs2.Load(parse) // parse accepts anything starting with "GOOD"
	t.Logf("Load: %q ok=%t", loaded, ok)
	for _, gen := range []string{"GOOD 1 record", "GOOD 2 records"} {
		_ = fs2.Save([]byte(gen))
	}
	main, _ := os.ReadFile(path)
	_ = os.Chmod(path+".prev", 0o600)
	prev, _ := os.ReadFile(path + ".prev")
	if !bytes.Contains(main, []byte("10000")) && !bytes.Contains(prev, []byte("10000")) {
		t.Errorf("the unreadable-but-intact generation was rotated out and destroyed")
	}
}
// output: Load with unreadable main: "" ok=false
//         after two dumps: main="GOOD 2 records" .prev="GOOD 1 record"
```

Fix:
- Change the port so the core sees outcomes: e.g. `Generations() []Generation{Name string; Data []byte; Err error}`; `Open` returns a `LoadReport{Source: main|prev|none; MainErr, PrevErr error; Discarded, Skipped int; DiscardedKeys []string}`.
- `main.go` logs the report at ERROR when `Source != main` or anything was discarded, and exports it (`supkv_dump_load_source{source=...}`, `supkv_dump_load_discarded_records`).
- A generation that exists but was not used is never rotated or overwritten: move it aside (`dump.json.unusable-<unixts>`) before the first save.
- Treat `errUnsupportedFormat` as a startup error, not as a chain step.

Acceptance: with `chmod 000` on main (and separately with `format_version: 2` in both files) the node either refuses to start or starts with an ERROR log naming the file and the errno, the metric is set, and after two dumps the original bytes still exist on disk under the quarantine name. A node with no files at all starts silently as today.

---

### ENG-03 — High — Verified — cross-boundary
**HLC is never advanced by `DataSync`; a write accepted after the gate opens is stamped below a known peer version**

`internal/app/storage/store.go:219-229` (the only `clock.Update` call site in production code), `internal/app/repl/core.go:178-187,631-644`

```go
// store.go — the single place the clock absorbs a remote reading
conflicts = s.join(key, rec)
remoteL, remoteC, _ := rec.Version.Unpack()
_, _, drift = s.clock.Update(remoteL, remoteC)
// core.go — StateStore has ApplyRemote and Records only; handleDataSync:
c.syncArrived = true
snapshot := c.store.Records()
for _, entry := range entries { ... enqueue DataRequest / DataReplication ... }
```

The spec's clock rule 2 (README:224-225): `Update(m)` is called [spec text], and the startup gate exists to close [spec text] (README:337-338). Neither `Store` nor `Engine` exposes a way to hand a version to the clock without merging a record, so the replication core cannot implement the rule.

Scenario: peer B's clock runs 90 s ahead (unsynchronized clocks are in the design envelope). A restarts or a partition heals. B's listing arrives carrying `(k, l = now+90s)`; `syncArrived` opens A's write gate. A client writes `k` on A and gets OK. The version is stamped from A's wall clock, below the version A was just told about. When B's record arrives through the rate-limited `DataRequest` path, B's older value displaces A's confirmed write. The window lasts until the peer's newest record has been fetched: milliseconds for a small delta, up to N/500 s for an empty or long-absent node. It also breaks the spec property [spec text] (README:251-253).

```go
func TestReview_dataSyncDoesNotAdvanceHLC(t *testing.T) {
	wall := time.UnixMilli(1_700_000_000_000)
	store := storage.NewStore(1, clock.NewClock(func() time.Time { return wall }, 0, 0, time.Minute), storage.Config{})
	eng := persist.New(store, nopDump{}, persist.Config{})
	core := repl.New(1, repl.NewRunID(), "self", eng, repl.Config{})
	peerVersion := version.PackVersion(wall.Add(90*time.Second).UnixMilli(), 0, 2)
	var out port.Outbox
	core.HandleDatagram(wall, port.InboundDatagram{
		Source: "peer", Status: port.Decoded, ProtoVersion: repl.SupportedProtoVersion, NodeID: 2, RunID: repl.NewRunID(),
		Msg: port.DataSync{Entries: []port.SyncEntry{{Key: "k", Version: peerVersion}}},
	}, &out)
	local, _, _ := store.Set("k", []byte("written after the listing was received"))
	if local.Compare(peerVersion) <= 0 {
		t.Errorf("the write stamped AFTER receiving the peer's version sorts below it")
	}
}
// output: peer version seen in the listing: 018bcfe6c79070008000800000000000
//         local version stamped afterwards:  018bcfe5680070008000400000000000
```

Fix: add `Store.Observe(v version.RecordVersion) time.Duration` (clock `Update` under the store lock, no record), expose it on `Engine`, add it to `repl.StateStore`, and call it once per received `DataSync` chunk with the chunk's maximum version before the gate flag is set.

Acceptance: the test above passes; a simulator scenario with one node's clock +90 s shows no local write accepted after a received listing ever losing to a version contained in that listing.

---

### ENG-04 — High — Verified
**`Records()` is the only bulk/tombstone-visible read: O(whole state) deep copy per `DataRequest` and per sync chunk**

`internal/app/storage/store.go:274-287`; callers `internal/app/repl/core.go:606,633,772`, `internal/app/persist/engine.go:301,581`, `internal/grpcapi/server.go:202`, `internal/grpcapi/watch.go:264`

```go
// store.go
for key, rec := range s.records {
	out[key] = Record{Version: rec.Version, Data: bytes.Clone(rec.Data), ...}
}
// core.go:606 — one key wanted
rec, ok := c.store.Records()[key]
```

`Get` hides tombstones, so every consumer that needs a tombstone, a version or a listing takes the full deep copy under the read lock. Stored records are immutable (`install` replaces wholesale), so the clone protects nothing for read-only consumers.

Measured (M3 Max; a management-board CPU will be several times slower):

| Operation | 10k x 1 KiB | 10k x 60 KiB |
|---|---|---|
| `Store.Records()` | 2.0 ms, 11.4 MB | 52.7 ms, 656 MB |
| one inbound `DataRequest` through `repl.Core` | 2.57 ms, 12.1 MB | 40.0 ms, 689 MB |
| `Store.Get` (same key, for scale) | 325 ns, 1 KB | — |
| worst `Set` latency while snapshots run | 2.4 ms | 74 ms |

Consequences at the stated envelope:
- Empty-node recovery: the healthy peer receives `sync_response_rate` = 500 `DataRequest`/s. At 10k x 1 KiB that is 1.3 CPU-seconds and 6 GB of allocation per second on the single replication goroutine; it cannot keep up, the socket buffer overflows, and recovery time is no longer "N / rate". At 60 KiB values it is 20 CPU-seconds per second.
- Every sync round: a 10k-key listing is ~300 chunks (33 entries per 1400-byte chunk with 16-byte keys), and the receiver snapshots once per chunk: 0.6 s CPU and 3.4 GB of garbage per round per peer every 30 s at 1 KiB; 16 s CPU per round at 60 KiB.
- Each snapshot holds the read lock for its duration; a waiting writer also blocks new readers.

Repro: `internal/app/storage/review_bench_test.go`, `internal/app/repl/review_bench_test.go` (`go test -bench BenchmarkReview -benchtime 20x`).

Fix: give the store the reads its consumers need and stop cloning for read-only use:
- `Lookup(key) (Record, bool)` — tombstone-visible, O(1), one value clone.
- `Versions() map[string]version.RecordVersion` (or `Range(func(key string, rec Record) bool)` under RLock, no clone) for listings and chunk comparison.
- `Snapshot()` returning records that share the immutable `Data` slices, documented read-only, for the dump.
Extend `repl.StateStore` accordingly; keep `Records()` only for tests if at all.

Acceptance: `BenchmarkReview_DataRequest_*` allocates O(one value) independent of key count; handling a `DataSync` chunk allocates O(entries in the chunk); `Set` latency under continuous sync traffic at 10k x 60 KiB stays below 1 ms on the reference machine.

---

### ENG-05 — Medium — Verified
**One damaged byte in one record rejects the whole generation; a dump truncated at a line boundary is accepted as complete**

`internal/app/persist/codec.go:97-124`

```go
dec := json.NewDecoder(bytes.NewReader(dump))
...
err = dec.Decode(&line)
if err != nil {
	if errors.Is(err, io.EOF) {
		return records, discarded, nil          // any clean EOF is "complete"
	}
	return nil, 0, fmt.Errorf("persist: dump record unreadable: %w", err)   // any other fault kills the generation
}
```

(a) The format is one record per line, but decoding is a single JSON stream, so a fault that breaks JSON or base64 syntax inside one record is a generation-level error. The doc promise "one torn record never collapses its generation" holds only for faults that keep the line parseable. Census over every single-bit flip in one 1 KiB record line: **44 % reject the whole generation**, 55 % discard the record, none is silently wrong.

```
single-bit flips in one 1491-byte record line: total=11928
  whole generation rejected=5229 (44%)  record discarded=6609 (55%)  harmless=90  silently wrong=0
```

Effect: one flipped bit in the main dump makes `Load` fall back to `.prev`, discarding every write confirmed by the last dump, although all other records in main are intact. If `.prev` has its own bad bit the node starts empty (ENG-02).

(b) There is no trailer, record count or whole-file check. A file cut at a line boundary decodes as a smaller, valid generation:

```
truncated to 400 of 1000 records: records=400 discarded=0 err=<nil>
```

The atomic write prevents this for the adapter's own crashes, but not for an interrupted copy, a restored backup or a filesystem that lost the tail. The 600 missing keys are simply absent and the next dump makes that permanent locally.

Repro: `TestReview_decodeGenerationFaults`, `TestReview_bitFlipCensus` in `internal/app/persist/review_repro_test.go`:

```go
corrupt := bytes.Clone(dump)
corrupt[dataIdx] ^= 0x40                         // one bit inside one base64 payload
_, _, err := persist.Decode(corrupt)             // err: invalid character '\x14' in string
truncated := bytes.Join(bytes.SplitAfter(dump, []byte("\n"))[:401], nil)
got, _, err := persist.Decode(truncated)         // len(got)=400, err=nil
```

Fix: decode line by line (`bytes.Split` on `\n`, `json.Unmarshal` per line; an unparseable line is `discarded++`). Append a trailer line (`{"records":N}`) and treat its absence or a count mismatch as a generation-level fault. Better still, load *both* generations and join them by LWW: both are valid CRDT states, tombstones are never dropped, so `main ⊔ prev` is always safe and recovers a record damaged in main from `.prev`.

Acceptance: the census reports 0 generation-level rejections for single-bit faults inside a record line; the truncated dump is rejected (or reported as incomplete); a main with one damaged record plus an intact `.prev` yields the full key set.

---

### ENG-06 — Medium — Verified
**After a fallback start the rejected main is rotated over the last good `.prev`; the engine is "clean" while disk differs from memory**

`internal/persistfile/dumpfile.go:149-157`, `internal/app/persist/engine.go:212-220`, `internal/persistfile/snapshot.go:50-54`

```go
// dumpfile.go
if prevPath != "" && targetExists(path) {
	err := os.Rename(path, prevPath)     // unconditional: also for a main that Load just rejected
	...
}
return os.Rename(tmpName, path)
// engine.go
if !dirty {
	return nil                           // "nothing unconfirmed on disk" — assumed, not known
}
```

Two consequences of the same gap (the engine never learns that the loaded state is not what is at the main path):

1. **Double-fault window.** Main is corrupt, `.prev` is the only good generation, the node loads `.prev`. The first save renames the corrupt main over `.prev`. A crash, or an EIO on the second rename (plausible on the disk that just corrupted main; `saveAtomically` then also deletes the temp file), leaves no loadable generation.

```
restart: Load -> "GOOD generation 1" ok=true (fell back to .prev, silently)
crash between the renames: files=[dump-crash dump.json.prev]  .prev="ROTTEN generation 2"  Load -> "" ok=false
```

2. **Snapshot of garbage.** After the same start the engine is not dirty, so `Snapshot` skips the dump and byte-copies the corrupt main, returning success. The recovery point is unusable; if main is absent instead, `Snapshot` fails with `ErrNoDump` (mapped to `codes.Internal`) although the state is in memory.

```
fallback start: discarded=0 (no signal that .prev was used); Get(b) err=key not found  records=1
Snapshot() err=<nil>; snapshot decodes: persist: dump record unreadable: unexpected EOF
```

Repro: `TestReview_corruptMainRotatedOverGoodPrev` (`internal/persistfile/review_repro_test.go`), `TestReview_snapshotAfterFallbackCopiesCorruptMain` (`internal/app/persist/review_repro_test.go`).

Fix: with the load report from ENG-02, `Open` marks the engine dirty whenever the loaded state is not exactly the main generation (fallback, discards, skips), and the adapter quarantines a rejected main instead of rotating it. Removing the main-absent window altogether is cheap: `os.Remove(prev); os.Link(main, prev); os.Rename(tmp, main)` keeps main present at every instant and makes the Save/CreateSnapshot mutex rationale unnecessary. Longer term, snapshot by encoding the in-memory state (`SaveSnapshot(name, bytes)`) so its validity does not depend on a file the engine did not write in this run.

Acceptance: after a fallback start, killing the process at any point of the first save leaves at least one loadable generation containing the pre-crash state; `Snapshot` after a fallback start produces a file that `Decode` accepts and that equals the in-memory state.

---

### ENG-07 — Medium — Verified
**Equal version with different payload is a silent no-op; zero filler instead of the spec's random tail makes it reachable**

`internal/app/version/version.go:19-22,45`, `internal/app/storage/store.go:240-249`, `internal/app/persist/codec.go:168-179`

```go
// version.go — "the remaining 46 rand_b bits are zero filler"
v[10] = byte((nodeID & nodeLsbMask) << fillerShift)
// store.go
default:
	return nil            // equal version: full no-op, payload not compared
```

The spec (README:243) fills the rest of `rand_b` with random bits and relies on that for [spec text]. With zero filler a version is fully determined by `(l, c, NodeID)`, so uniqueness across restarts rests entirely on the dump watermark, which cannot cover versions that never reached a dump but did reach a peer (a failed dump, or a kill before the next dump, with the record picked up through `DataSync`). When the HLC runs ahead of the wall clock at restart (peer clock ahead, or local clock stepped back) the node re-issues `(l, c)` values. If one lands on the same key with different data, `Merge` and `DataSync` both see "equal" and stay silent forever.

The repro needs the counters to line up (after restart the clock sits at watermark + one increment per loaded record), so this is rare; when it happens it is permanent and invisible, and the second write was confirmed to its client.

```go
// internal/app/persist/review_repro_test.go, TestReview_equalVersionDivergenceAfterRestart (essentials)
storeB.Set("other", []byte("b"))                         // B's clock is 60 s ahead of A's
engA.ApplyRemote("other", storeB.Records()["other"])     // A's HLC now runs ahead of A's wall clock
engA.Set("k", []byte("v1"))                              // dumped
dumpsA.setFail(true)
for i := range 3 { v2, _, _ = engA.Set("k", fmt.Appendf(nil, "lost-%d", i)) }   // ErrDumpFailed x3
storeB.Merge("k", engA.Records()["k"])                   // reaches B through the periodic sync
stop(); dumpsA.setFail(false); wall = wall.Add(5 * time.Second)                 // A restarts 5 s later
engA2, _ := persist.Open(nodeA, now, 0, dumpsA, storage.Config{}, persist.Config{})
v3, _, err := engA2.Set("k", []byte("v3"))               // err == nil, v3 == v2
engA2.ApplyRemote("k", storeB.Records()["k"]); storeB.Merge("k", engA2.Records()["k"])
// output:
// v2=00dc6ad0966070058000400000000000 (never dumped, held by B)
// v3=00dc6ad0966070058000400000000000 (confirmed after restart)
// after exchange: A="v3" B="lost-2"  conflicts A=0 B=0
// PERMANENT DIVERGENCE: same version ..., A holds "v3", B holds "lost-2"; no conflict reported
```

A related hole in the watermark: records discarded for a checksum fault are excluded from it, although their version field decoded fine (`TestReview_watermarkIgnoresDiscardedRecords`: watermark `(5000,1)` while the discarded record carried `(5000,2)`), so the first post-restart tick can re-issue exactly the discarded record's version.

Fix:
- Fill the 46 spare bits with random data per version, as the spec says (they sort below `(l, c, NodeID)`, so the order contract is unchanged). A re-issued `(l, c)` then differs in the tail and LWW resolves it identically on every node.
- In `join`, when versions are equal and the checksums differ, do not return silently: break the tie deterministically (for example by checksum, then data) and count it (`supkv_merges_equal_version_mismatch_total`).
- Compute the watermark over every record whose version field parses, including checksum-discarded ones.

Acceptance: the repro converges (both nodes hold the same value); a property test "two stores that exchanged all records are equal, including equal-version/different-payload inputs" passes; `Watermark` of a dump with a checksum-faulty newest record dominates that record's version.

---

### ENG-08 — Medium — Verified
**Failed dump: retried every 50 ms forever, cause never logged or returned, failed write never broadcast**

`internal/app/persist/engine.go:202,292-312,374-389`, `internal/app/persist/errors.go:12`, `internal/grpcapi/server.go:147-150`

```go
_ = e.runDump() // the error belongs to the waiters; nobody is left to report to.
...
for _, w := range stranded {
	w.err = ErrDumpFailed          // sentinel only; the errno is gone
}
```

- **No backoff.** `failWaiters` re-dirties, `rearm` arms the 50 ms debounce, and the cycle repeats with no writer present. One write on a broken disk: `Save attempts=203` in 10 idle seconds. Each attempt is a full `Records()` copy plus encode (ENG-12), i.e. the node burns CPU and memory at 20 Hz for as long as the disk is full or read-only.
- **Cause lost.** Waiters get `ErrDumpFailed` without the wrapped error (`errors.Is(err, cause)=false`); `Run` discards the return value; `main.go` logs dump errors only for the final shutdown flush. When the dirtiness came from remote merges there is no waiter at all, so nothing but `supkv_dumps_errors_total` moves. The spec asks for a log entry (README:357-358). An unwritable dump directory at startup is not detected either: the node starts normally and every write returns `Unavailable` with no reason anywhere.
- **Cross-boundary:** on `ErrDumpFailed` the gRPC handler returns before `notifyLocalWrite`, so the record that [spec text] (README:358-359) is not broadcast; it reaches the peer only through the next periodic sync, and after the retry succeeds nobody notifies either.

```go
func TestReview_failingDumpRetryCadence(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dumps := &memDump{fail: true}
		eng := persist.New(storage.NewStore(1, clock.NewClock(time.Now, 0, 0, time.Minute), storage.Config{}), dumps, persist.Config{})
		stop := startRun(t, eng); defer stop()
		_, _, err := eng.Set("k", []byte("v"))
		synctest.Sleep(10 * time.Second)
		t.Logf("attempts=%d errors.Is(err, cause)=%t", dumps.attempts(), errors.Is(err, errDiskFull))
	})
}
// output: Set error: dump failed; write not confirmed; errors.Is(err, cause)=false
//         after 10 idle seconds on a broken disk: Save attempts=203, DumpStats={... Total:203 Errors:203}
```

Fix: wrap the cause (`fmt.Errorf("%w: %w", ErrDumpFailed, err)`); give `Config` an `OnDumpError func(error)` (or return a last-error accessor) that `main.go` logs with rate limiting; back off exponentially from `Debounce` to a few seconds while consecutive dumps fail, resetting on success or on a new waiter; probe the dump directory for writability in `Open`.

Acceptance: at most ~10 attempts in the 10 s scenario; `errors.Is(err, errDiskFull)` is true; one ERROR log record per failure streak carrying the errno and path; a read-only dump directory is reported at startup.

---

### ENG-09 — Medium — Verified
**Debounce is a resettable quiet window: median confirmation 0.5 s under sustained writes; spec bound is `dump_debounce` + dump time**

`internal/app/persist/engine.go:227-245`

```go
if debounce == nil {
	debounce = time.NewTimer(e.cfg.Debounce)
} else {
	debounce.Reset(e.cfg.Debounce)       // every change restarts the window
}
if deadline == nil {
	deadline = time.NewTimer(time.Until(oldest.Add(e.cfg.MaxWait)))
}
```

Spec: [spec text] (README:490-491), `dump_debounce` = [spec text] (README:609), and for a 1000 writes/s burst the client delay is [spec text] (README:565). That is a minimum interval between dumps. The implementation is a trailing quiet window with a 1 s cap (`dump_max_wait`, a parameter the spec does not have): an isolated write always waits 50 ms before its dump starts, and under any cadence faster than 50 ms the dump starts only when the 1 s deadline fires.

```
5 s @ 1000 writes/s, dump_debounce=50ms, instantaneous dump: saves=5  p50=501ms  p99=991ms  max=1s  mean=500.5ms
```

Ten to twenty times the spec's bound, on exactly the load class the spec sizes the parameter for.

Repro: `TestReview_sustainedLoadLatency` (`internal/app/persist/review_repro_test.go`): 5000 `Set` calls at a 1 ms virtual cadence against the suite's `fakeDumpPort`, latency per call collected and sorted.

Fix: schedule `next = max(now, lastDumpEnd + Debounce)` when the engine turns dirty; one timer, no reset on later changes, no `MaxWait`. This also removes a timer, a config knob and the stale-deadline cases in `rearm`. If the quiet-window behaviour is wanted, the spec needs the amendment, not the code a comment.

Acceptance: in the same test p99 <= `Debounce` + one dump duration; an isolated write after an idle period longer than `Debounce` starts its dump immediately; dumps never start closer than `Debounce` after the previous one finished.

---

### ENG-10 — Medium — Verified
**`Restore` does not delete keys absent from the snapshot, reports success on a partly discarded snapshot, and bypasses replication notify**

`internal/app/persist/engine.go:572-591,617-630`, `internal/grpcapi/server.go:243-252`

```go
records, _, err := Decode(data)        // discarded count dropped
...
for _, key := range slices.Sorted(maps.Keys(records)) {
	err = replayRecord(e.store, key, records[key])   // only the snapshot's keys; store.Set / store.Delete directly
```

1. **Spec deviation.** README:510-512: [spec text]. The code leaves such keys untouched, and the project's own test pins it (`engine_test.go:1047-1056`, "post-key: absent from the snapshot — its post-snapshot state stands"). Result: after loading a snapshot the store is not in the snapshot's state. Rolling back a bad configuration push that *added* keys leaves those keys live on every node.
2. **Partial restore reported as success.** A snapshot record that fails its checksum is dropped by `Decode`, and `Restore` returns nil:
   ```
   Restore err=<nil>; hot="changed-after-snapshot"      // TestReview_restoreIgnoresDiscardedSnapshotRecords
   ```
3. **Cross-boundary: no replication notify.** The replay writes through `store.Set`/`store.Delete`, and `SnapshotLoad` never calls `notifyLocalWrite`. Restored records are not broadcast and not tracked for Ack; peers learn of them only through the next periodic `DataSync` (up to `sync_interval` plus N/500 s). [spec text] holds only in the eventual sense.
4. Any difference re-stamps every snapshot key ("ANY difference replays in full"), so restoring a 10k-key snapshot that differs in one key produces 10k new versions, 10k Watch events (enough to overflow the hub's 4096-slot ingress, which disconnects every subscriber; not executed) and 10k `DataRequest` round trips per peer.

Fix: implement the spec's second half (tombstone every live key not in the snapshot); return an error, or at least the count, when `discarded > 0`; replay only keys whose record differs (the comparison already exists in `restoreChanges`); route the replayed writes through the same notify seam as `Set`/`Delete`.

Acceptance: after `Restore(s)` the live key set and values equal the snapshot's; a snapshot with a damaged record makes `Restore` fail before any write; a second node converges within one RTT per restored key without waiting for a sync round; restoring an identical-but-one-key snapshot issues exactly one new version.

---

### ENG-11 — Medium — Verified
**Drift signal fires after any idle period > `drift_warn` and stays silent on the event that absorbs a far-future version**

`internal/app/clock/clock.go:90-91,123-125,159-176`

```go
ptMs := c.now().UnixMilli()
drift = driftBetween(c.l, ptMs, c.driftWarn)     // PRE-update local l against the wall clock
newL := max(c.l, remoteL, ptMs)
```

The signal compares the clock's stale `l` with the wall clock; the remote reading is not part of it. Spec (README:259-260): warn [spec text]. Three wrong outcomes:

```
(a) synced peer after 61 s idle: drift=-1m1s          <- false positive
(b) absorbing a version 1 h ahead of the wall clock: drift=0s   <- missed
(c) next synced remote version: drift=1h0m0s          <- counted on every later merge
```

(a) `l` only moves on events, so after 61 s without a write or merge any remote record reports "-61 s drift" although all clocks agree. For the low-rate data classes ([spec text]) nearly every merge increments `supkv_clock_drift_warnings_total` on a healthy cluster. (b) The one event that actually carries the far-ahead timestamp reports zero. (c) From then on every merge counts, whoever sent it, so the counter measures traffic volume while the HLC is ahead. `docs/OPERATIONS.md:89-92` and the HELP text ("Remote timestamps absorbed beyond the drift-warn threshold") describe the spec's behaviour, not this one. Nothing is logged for drift anywhere, and the `Tick` drift is discarded by every caller.

```go
func TestReview_driftSignalSemantics(t *testing.T) {
	wall := time.UnixMilli(1_700_000_000_000)
	clk := clock.NewClock(func() time.Time { return wall }, 0, 0, 60*time.Second)
	clk.Tick()
	wall = wall.Add(61 * time.Second)
	_, _, drift := clk.Update(wall.UnixMilli(), 0)                      // want 0, got -61s
	_, _, drift = clk.Update(wall.Add(time.Hour).UnixMilli(), 0)        // want +1h, got 0
	_, _, drift = clk.Update(wall.UnixMilli(), 0)                       // got +1h
}
```

Fix: in `Update`, report `remoteL - ptMs` when it exceeds `driftWarn` (the spec's definition), and nothing else. Drop the negative ("wall clock jumped forward") branch or make it a separate, explicitly named signal that excludes idle time. Log the first occurrence per peer.

Acceptance: (a) returns 0, (b) returns +1 h, (c) returns 0; a simulator run with synchronized clocks and one write per 2 minutes leaves the counter at 0.

---

### ENG-12 — Medium — Verified
**One dump cycle allocates ~5x the live state and blocks writers for the copy**

`internal/app/persist/engine.go:300-304`, `internal/app/persist/codec.go:62-85`, `internal/persistfile/dumpfile.go:118-144`

```go
block, err := Encode(e.store.Records())   // deep copy of all data, then base64+JSON into one bytes.Buffer
if err == nil {
	err = e.dumps.Save(block)             // then one Write of the whole buffer
}
```

Three full materializations per cycle: the deep-copied map, the growing `bytes.Buffer` (1.37x for base64 plus doubling slack), and the returned slice kept alive through `Save`.

| State | Dump file | Records / Encode / Save | Allocated per cycle | Peak heap |
|---|---|---|---|---|
| 10k x 1 KiB (14 MiB live) | 14 MiB | 4 / 19 / 18 ms | 61 MiB | 65 MiB (4.7x live) |
| 10k x 60 KiB (627 MiB live) | 782 MiB | 69 / 843 / 996 ms | 3444 MiB | 3092 MiB (4.9x live) |

The upper row is the product of both documented maxima; a few hundred 60 KiB values land in between. On a memory-constrained board the transient decides whether the process is OOM-killed, and every burst repeats it (once per `dump_max_wait` under load, 20 times per second while a dump is failing, ENG-08). During `Records()` writers are locked out (74 ms worst `Set` latency at the large size) and, because a waiting writer blocks new readers, so are `Get` calls.

Repro: `TestReview_dumpCycleAtEnvelope` (`internal/app/persist/review_envelope_test.go`), `TestReview_writerStallDuringRecords` (`internal/app/storage/review_bench_test.go`).

Fix: take a shallow snapshot (share the immutable `Data` slices), and stream the encoding into the temp file through a `bufio.Writer`: change the port to `Save(write func(io.Writer) error) error`. Extra heap per cycle becomes the map of headers plus one buffered line. `CreateSnapshot` and `Load` can stream the same way.

Acceptance: peak heap during a dump <= 1.3x live at 10k x 60 KiB; allocated bytes per cycle independent of total value bytes; `Set` latency during a dump < 1 ms on the reference machine.

---

### ENG-13 — Medium — Verified (limit absent) / Suspected (end effect)
**No key-size bound: a record that cannot fit one datagram is confirmed and can never replicate**

`internal/app/storage/store.go:159-162,187-190,219-222`

```go
if len(data) > s.cfg.ValueMaxBytes {
	return version.RecordVersion{}, 0, ErrValueTooLarge
}
```

Only the value is bounded. With the default 61440-byte limit a datagram has 65507 - 61440 = 4067 bytes left for the envelope and the key; nothing in the store, the engine, the gRPC handlers or `main.go` limits key length (grep for `len(key)`/`len(req.GetKey` finds only the sync chunk estimator). The spec's premise is [spec text] (README:106).

Scenario: `Set` with a 5 KiB key and a 60 KiB value is confirmed locally. Its `DataReplication` exceeds the UDP payload limit; the send fails and is retried forever; a `DataSync` entry for a key above ~1.3 KiB is shipped as its own oversized chunk. The peer never gets the record: a confirmed write that exists on one node only and is lost with that node.

Suspected part: the EMSGSIZE send failure itself was not executed (udprepl tests are out of bounds for this review); the absence of any bound is verified.

Fix: `storage.Config{KeyMaxBytes, RecordMaxBytes}` with `ErrKeyTooLarge`, enforced in `Set`/`SetIfAbsent`/`Merge`, with defaults derived from the datagram budget; map to `InvalidArgument`.

Acceptance: a write whose encoded `DataReplication` would exceed one datagram is rejected with `InvalidArgument` before any state change; a property test over random key/value sizes shows every accepted record encodes within the datagram limit.

---

### ENG-14 — Medium — Verified
**`SetIfAbsent` returns `AlreadyExists` for a tombstoned key; the spec says it must write**

`internal/app/storage/store.go:193-195`

```go
if _, ok := s.records[key]; ok {
	return version.RecordVersion{}, 0, ErrAlreadyExists
}
```

Spec (README:361-362): [spec text]. The code rejects on any record, tombstones included, deliberately (`storage/doc.go:15-19`). Tombstones are kept forever and replicate, so a key that was deleted once can never again be created through create-if-absent on any node. The feature exists for build-time defaults: "delete the key and restart to get the default back" does not work, and a default that was removed by mistake stays missing for the lifetime of the cluster unless someone issues an unconditional `Set`.

Either reading is defensible; the two documents disagree and the consequence is user-visible.

Fix: follow the spec (`ok && !rec.Deleted` rejects), or amend the spec and document the reset procedure in `docs/OPERATIONS.md`.

Acceptance: a decision recorded in the spec; a test for `Delete` followed by `SetIfAbsent` asserting the chosen outcome on both nodes after replication.

---

### ENG-15 — Low — Verified
**Blocking engine calls take no context: a hung fsync blocks writers, snapshot, restore and shutdown forever**

`internal/app/persist/engine.go:411-472,544-550,572-591,643-684`, `cmd/sup-kv/main.go:676`

`Set`, `SetIfAbsent`, `Delete`, `Snapshot`, `Restore`, `Flush` have no `context.Context` and no deadline; `await` ends only on a dump result or on `Run` exiting, and `Run` cannot exit while it is inside `Save`. A stalled device (fsync can block indefinitely) parks every write handler with its request payload, ignores client cancellation and deadlines, and `shutdown` hangs in `node.engine.Flush()` after the signal channel has stopped being read, so only SIGKILL ends the process. In the healthy case a cancelled RPC still holds its goroutine for up to `dump_max_wait` plus one dump.

Fix: accept a context in the blocking calls and select on `ctx.Done()` in `await`/`flush` (the write stays in memory and is reported unconfirmed, exactly like `ErrDumpFailed`); bound the shutdown flush with a timeout and exit non-zero.

Acceptance: with a `DumpStore` whose `Save` blocks, `Set(ctx)` returns `ctx.Err()` at the deadline and the process exits within the shutdown grace period.

---

### ENG-16 — Low — Verified
**`process_resident_memory_bytes` reports peak RSS**

`internal/metrics/metrics.go:356-360,369-381`

```go
rss := float64(ru.Maxrss)       // getrusage: MAXIMUM resident set size
```

`ru_maxrss` is the high-water mark. The family uses the conventional name for the *current* resident set, so dashboards and alerts built on it never see memory return after the dump-time spike (ENG-12).

```
gauge before=13 MiB, with 512 MiB live=527 MiB, after free+scavenge=527 MiB     // TestReview_residentMemoryGaugeIsPeak
```

Fix: read the current RSS (`/proc/self/statm` on Linux; `os` is allowed in this adapter) and, if the peak is wanted, publish it under its own name.

Acceptance: the test above shows the gauge falling after the memory is released on Linux.

---

### ENG-17 — Low — Verified
**Orphaned temp files are never collected; new directories are not made durable**

`internal/persistfile/dumpfile.go:97-113,118-134`, `internal/persistfile/snapshot.go:45-49`

- A crash between `CreateTemp` and the rename leaves `dump-<random>` in the dump directory (and in `.snapshots`). Nothing removes them at startup or later; each is a full dump in size.
  ```
  after Load + 3 Saves: [dump-111 dump-222 dump-333 dump.json dump.json.prev]
  ```
- `os.MkdirAll(dir)` is followed by an fsync of `dir` itself but not of its parent. On a fresh deployment (first dump) or the first snapshot, a power loss can drop the new directory entry together with the confirmed dump or snapshot inside it.
- `CreateSnapshot` creates the `.snapshots` directory before it knows a main dump exists, so a failed call still leaves the directory behind.

Fix: remove `dump-*` leftovers in `persistfile.New`/first `Load`; after `MkdirAll` creates a directory, fsync its parent; create the snapshot directory after `readMainDump` succeeds.

Acceptance: `TestReview_orphanTempFilesNeverCollected` passes; a test asserts the parent directory is synced when the dump directory is created.

---

### ENG-18 — Low — Verified
**`fsync_darwin.go` re-implements what `os.File.Sync` already does, on a false premise, and loses EINTR/ENOTSUP handling**

`internal/persistfile/fsync_darwin.go:10-29`

The file's rationale: "the stdlib binds Fsync to plain libc_fsync and never sends F_FULLFSYNC, verified in the Go 1.27 toolchain source". That is true of `syscall.Fsync`, not of `os.File.Sync`, which goes through `internal/poll` (Go 1.27.4, `src/internal/poll/fd_fsync_darwin.go`):

```go
// Fsync invokes SYS_FCNTL with SYS_FULLFSYNC because
// on OS X, SYS_FSYNC doesn't fully flush contents to disk.
func (fd *FD) Fsync() error {
	...
	return ignoringEINTR(func() error {
		_, err := unix.Fcntl(fd.Sysfd, syscall.F_FULLFSYNC, 0)
		// ... SMB mounts where fcntl will fail with ENOTSUP. In those cases fallback to fsync.
		if err != nil && errors.Is(err, syscall.ENOTSUP) {
			err = syscall.Fsync(fd.Sysfd)
		}
		return err
	})
}
```

The hand-rolled `syscall.Syscall(SYS_FCNTL, f.Fd(), F_FULLFSYNC, 0)` therefore adds nothing and subtracts the EINTR retry (a spurious `ErrDumpFailed`), the ENOTSUP fallback (permanent failure on filesystems without F_FULLFSYNC) and fd reference counting. Darwin is not the deployment target, hence Low.

Fix: delete `fsync_darwin.go` and `fsync_other.go`; call `f.Sync()`.

Acceptance: `persistfile` has no build-tagged files and no `syscall` import; tests stay green on darwin and linux.

---

### ENG-19 — Low — Verified (store level) / Suspected (wire reachability)
**Version bits are not validated at trust boundaries: one malformed version poisons ordering or exhausts the domain**

`internal/app/version/version.go:53-65`, `internal/app/persist/codec.go:133-136`, `internal/app/storage/store.go:219-229`

`Compare` orders all 16 bytes; `Unpack`/the clock see only `(l, c, NodeID)`. Nothing checks the version nibble, the variant bits or a plausible `l` when a version enters from a dump (`decodeRecord` checks length only) or from `Merge`.

```
// TestReview_malformedVersionBeatsLaterLocalWrites: version nibble 0xF instead of 0x7
stored remote 018bcfe56800f0008000800000000000
later local   018bcfe5680070028000400000000000  compare=-1
// TestReview_ceilingVersionExhaustsDomain: one merged version with l = 2^48-1
write #4199 ffffffffffff7fff8000400000000000
write #4200 ffffffffffff7fff8000400000000000
```

In the first case every later local write to that key sorts below the stored record: confirmed locally, rejected by peers, then overwritten by the old record on the next exchange. In the second, 4095 writes after the merge every write of the node carries the same version, so peers see "equal" and never take the new data (ENG-07), and the state persists in every dump. Reaching either through the shipped binary needs a datagram or dump line with such bits and a valid CRC32, which the trust model treats as out of scope; the cost of the check is two byte comparisons. Whether `udprepl` filters such versions was not examined (Suspected).

Fix: `func (v RecordVersion) Valid() bool` (nibble 7, variant 10); reject invalid versions in `decodeRecord` and `Merge`; reject or clamp a remote `l` beyond `wall + N * drift_warn`.

Acceptance: both tests pass (the merge is refused with a typed error and a counter increments).

---

### ENG-20 — Low — Verified
**"Conflict" is every overwrite and every stale redelivery; metric HELP and `Merge` doc say otherwise**

`internal/app/storage/store.go:213-250`, `internal/app/persist/engine.go:501-519`, `internal/metrics/metrics.go:243-248`

- `join` returns a `Conflict` for any displacement, so an ordinary sequential update `v1 -> v2` arriving at the peer is a "conflict". The spec's conflict is narrower: an *unacknowledged local* record replaced by a different remote one (README:530-531). `supkv_merges_conflicts_total` therefore tracks remote update volume.
- The doc says each displaced or rejected version is returned "exactly once"; a retransmitted stale record returns the same `Conflict` on every delivery.
- HELP reads "Remote merges that displaced a stored record"; the counter also includes rejected stale deliveries.
- `Merge` does not say whether it installed anything, so `ApplyRemote` probes with `store.Get` first (cloning up to 60 KiB to learn "live or not") and infers. The inference is safe but cannot see tombstones, so an equal-version or stale redelivery over a tombstone triggers a full redundant dump (acknowledged in the comment).

Fix: `Merge` returns `(changed bool, displaced *Conflict, ...)`; `ApplyRemote` uses `changed`; the conflict counter is fed only for displacement of a pending local version (the replication core knows the unacked set); correct HELP and doc text.

Acceptance: a replicated sequential update leaves the conflict counter unchanged; a tombstone redelivery does not start a dump; `ApplyRemote` performs no `Get`.

---

### ENG-21 — Low — Verified (package level)
**A key that is not valid UTF-8 is accepted, mangled by the codec and discarded at reload**

`internal/app/storage/store.go:159-167`, `internal/app/persist/codec.go:70-82`

The store accepts any Go string. `encoding/json` replaces invalid UTF-8 with U+FFFD while the checksum was computed over the original bytes, so the record fails its own checksum on every load:

```
dump line: {"key":"bad�key","record_version":"0000...","data":"dg==","deleted":false,"checksum":2218801238}
Encode accepted the key, Decode dropped it: records=0 discarded=1 err=<nil>      // TestReview_invalidUTF8KeyLostOnReload
```

Not reachable through the shipped surfaces today: both `.proto` files declare keys as proto3 `string`, which protobuf-go validates. It is a trap for any caller that bypasses them (tests, an embedding, a future `bytes` key).

Fix: reject keys failing `utf8.ValidString` in the store (same error class as ENG-13), or have `Encode` return an error.

Acceptance: `Set("bad\xffkey", ...)` returns a typed error; `Encode` never emits a line that `Decode` discards.

---

### ENG-22 — Low — Verified
**Records skipped by a shrunk value limit vanish without count or log, then from disk**

`internal/app/persist/engine.go:156-163`

```go
_, _, err := store.Merge(key, rec)
if errors.Is(err, storage.ErrValueTooLarge) {
	continue
}
```

After lowering `--value-max-bytes`, every stored value above the new limit is dropped at startup. It is not part of `discarded` (the project test asserts `discarded == 0` for this case), nothing is logged, the engine is clean, and the next dump writes a main without those keys. No tombstone is written, so a peer that still holds them keeps re-offering records the node keeps refusing.

Fix: apply the limit to new writes only and load existing records regardless; or count and log the skipped keys through the load report (ENG-02) and refuse to drop them silently.

Acceptance: restart with a lower limit keeps every previously confirmed record readable, or fails startup with the list of offending keys.

---

### ENG-23 — Info — Verified
**Dead and over-defensive code; startup load goes through `Merge`**

- `Engine.lastLocal`, the `cover` capture, `waiter.ver` and the "above cover" branch of `confirmWaiters` (`engine.go:72,294,347-368,457-459`) are unreachable: `await` raises `lastLocal` in the same critical section that appends the waiter, so every captured waiter is at or below the cover. The comment says "unreachable in practice", and `TestConfirmWaiters_aboveCoverInvariant` exists to exercise it by writing private fields. The captured list is the covered set; ~40 lines and one internal test can go.
- `persist.Parse` (`codec.go:158-161`) is exported and used only by tests; `Open` passes its own closure.
- `mergeRecords` loads the dump through `Merge` (`engine.go:156-163`): N clock updates (with the wall clock behind the watermark the counter advances once per record, 10k records burn two milliseconds of `l` per restart) and N `OnChange` callbacks into a Watch hub that is not running yet (more than 4096 records overflow its ingress and set the disconnect flag before the first subscriber exists). A dedicated `Store.Load(records)` avoids both.
- `decodeRecord` clones `line.Data` although `json` already allocated it, and `install` clones it again: three copies of every value during startup.
- `Run` may be started once only; a second call panics on `close(e.runDone)`. Not guarded, not stated in the doc.

---

### ENG-24 — Info — Verified
**Stale or false comments**

- `storage/store.go:73-74`, `storage/doc.go:5-6`, `repl/core.go:176-177`: "per D-03, internal/port holds no interfaces" — `port.DumpStore` and `port.Message` are interfaces, and `port/doc.go` describes the package as "the interface definitions".
- `persist/doc.go:15`: "No callbacks, no events — results only" — `storage.Config.OnChange` is a callback threaded through `persist.Open`. `storage/doc.go:21` "The package has no side effects" has the same problem.
- `port/dump.go:27`: "parse is supplied by the engine (codec.Parse)" — it is a closure in `Open`.
- `storage/store.go:216`: "returned exactly once" (ENG-20).
- `persistfile/fsync_darwin.go:13-16`: the stdlib claim (ENG-18).
- `version/version.go:22`: "zero filler" is accurate for the code and contradicts `docs/IDEA.md:68-69` and the spec ("random filler").
- `metrics/metrics.go:245,252` HELP texts (ENG-11, ENG-20); `docs/OPERATIONS.md:89-92` drift runbook.
- `AGENTS.md:36-41`: `internal/app` "split into three packages"; there are five plus `internal/`.

---

### ENG-25 — Info — Verified
**Depguard boundary does not enforce dependency direction**

`.golangci.yml:68-88`. `sup-kv-app-no-io` denies `net`, `os`, `syscall`, `google.golang.org/`; it does not deny `github.com/Djarvur/sup-kv/internal/{persistfile,udprepl,grpcapi,metrics}`, so `internal/app` can import an adapter (and its I/O transitively) without tripping the rule. `sup-kv-port-stdlib-only` allows the whole module path, so `internal/port` can import adapters too. `internal/metrics` imports `internal/udprepl` for the `SentByType` struct, pulling the UDP adapter and protobuf into the exporter's graph. Suggested: add the adapter paths to the app deny-list, narrow the port allow-list to `internal/app/{storage,version}`, move `SentByType` to `internal/app/repl` or `internal/port`.

---

### ENG-26 — Info — Verified
**Checksum domain has no length prefixes (theoretical only)**

`internal/app/storage/store.go:131-137`: CRC32 over `key || version(16) || data || flag(1)`. Two different `(key, version, data)` triples with different key lengths can produce the same byte stream. For the dump this cannot be reached by corruption: the three fields are separately delimited and differently encoded in JSON, so a shifted boundary would require consistent edits in three places. No action needed; if the checksum is ever reused for another purpose, prefix the key length.

---

## 3. Repro helpers

Shared by the engine-level repros (`internal/app/persist/review_repro_test.go`):

```go
type memDump struct {           // port.DumpStore that survives a simulated restart
	mu   sync.Mutex
	last []byte
	fail bool
	n    int
}
var errDiskFull = errors.New("ENOSPC (injected)")
func (m *memDump) Save(dump []byte) error {
	m.mu.Lock(); defer m.mu.Unlock()
	m.n++
	if m.fail { return errDiskFull }
	m.last = bytes.Clone(dump)
	return nil
}
func (m *memDump) Load(parse func([]byte) error) ([]byte, bool) {
	m.mu.Lock(); defer m.mu.Unlock()
	if m.last == nil || parse(m.last) != nil { return nil, false }
	return m.last, true
}
func (*memDump) CreateSnapshot(string) error         { return nil }
func (*memDump) ReadSnapshot(string) ([]byte, error) { return nil, errDiskFull }

func startRun(t *testing.T, eng *persist.Engine) (stop func()) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- eng.Run(ctx) }()
	return func() { cancel(); <-done }
}
```

Run everything:

```
cd evidence/repro/engine
go test ./internal/app/clock ./internal/app/storage ./internal/app/persist ./internal/app/repl ./internal/persistfile ./internal/metrics -run TestReview -count=1 -v
go test ./internal/app/storage ./internal/app/repl -run XXX -bench BenchmarkReview -benchtime 20x
```

Not examined: `internal/udprepl` and `internal/grpcapi` internals beyond the call sites quoted, `cmd/sup-kv` beyond wiring and shutdown, and any behaviour on Linux (all runs were on darwin; the persistence findings do not depend on the platform, ENG-16's numbers will differ in unit but not in kind).
