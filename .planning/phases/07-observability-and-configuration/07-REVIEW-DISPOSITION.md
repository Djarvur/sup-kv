---
phase: 07-observability-and-configuration
updated: 2026-11-23T13:05:00Z
total: 10
---

# Code Review Disposition — Phase 07

One row per finding from `07-REVIEW.md`. Edit the **Disposition** column by hand when triaging
(`open` / `fixed` / `skipped` / `deferred`); put the reason in **Source**.

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | fixed | An explicitly empty `--grpc-addr` bypassed validation and `net.Listen` bound the unauthenticated API (reflection on) to the wildcard address — the opposite of the documented loopback default. `a030a4b`: `checkAddressShape` rejects an empty `--grpc-addr` (only the metrics endpoint keeps the documented "no listener" opt-in); table row added and a process-level relation row proves exit 2 before any socket work. Re-probe: `--grpc-addr=` exits 2 naming the parameter. |
| WR-02 | warning | fixed | The address rule accepted ports outside 0..65535 (validated, then exit 1 at `grpc_listen`) and port 0 on the broadcast destination (a node that can never send). `963aa0c`: `validHostPort` range-checks 0..65535; port 0 stays legal for listeners and is rejected for `--broadcast-addr`; table rows for `:99999`, `:-1`, `:65536`, broadcast `:0` and a positive listener `:0`; process rows prove exit 2 before bind; the runbook text was aligned. |
| WR-03 | warning | fixed | The exit-1 table listed a "dump open" startup failure that cannot exist (`persist.Open` returns no error; a bad dump path is a load outcome, and failure to write surfaces as a final-dump exit 3). `d719670`: the exit-1 row now names only the real stages; the runbook states the as-built dump-path behavior; a forbidden-phrase guard in `TestOperationsDoc` keeps `dump open` from re-entering. |
| WR-04 | warning | fixed | The registry cited the deleted `internal/udprepl/repl.go` (seven rows) and pre-extraction `main.go` line ranges (the parameter table). `212d27a`: every `repl.go` citation repointed to the shipped split files with the symbol/line, and the `S-PAR-*`/`S-AVL-*`/`S-IMP-07` rows repointed to `internal/config/{config,validate,usage}.go`; each replacement symbol was grep-verified for the rows this fix touched. The re-verification found that the registry as a whole still carries pre-existing drift outside those rows (New-1/New-2/New-3 below), so the "all in range" reading was an overclaim — corrected here. |
| IN-01 | info | fixed | `checkDriftVsLead` added 1 ms before comparing, so a `drift_warn` near the duration maximum wrapped negative and an invalid relation validated. `7a4d8c6`: subtraction form (`MaxClockLead-DriftWarn < 1ms`), table row at `DriftWarn = math.MaxInt64`; a deliberate revert to the additive form makes the new row fail, so the test discriminates both directions. |
| IN-02 | info | fixed | The `StateStore.Observe` contract claimed its drift was not counted, while `handleDataSync` counts exactly that return. `fce532b`: the interface doc now states the positive return is counted in `MergeStats.DriftWarnings` and reported through `OnDrift`/`Log` under the shared limiter; the file's line count is unchanged, so no registry citation shifted. |
| IN-03 | info | fixed | `AGENTS.md` still said flags/env/signals live in `main()` and that it holds no testable logic, contradicting the config extraction and the tested `run` seam. `a982d19`: the bullet now describes `main()` as parse → help/version → logging → `run`, with signals and shutdown order in `run`/`shutdown`. |
| IN-04 | info | fixed | `TestMirror_listingCounters` compared two counters that were zero on both sides, so a missing or swapped mirror store passed. `7a1f8af`: each of the four counters is driven with a distinct non-zero sample (core `ListingDropped=9`, `SyncStreamDropped=4`); a deliberate swap in `mirrorStats` fails the test naming both values. |
| IN-05 | info | fixed | The refused-apply row stamped its record inside the drift window, so `DriftWarnings == 0` was guaranteed by the stamp rather than by the refusal. `40c6bbd`: the record is stamped at `test_now + drift_warn + 1s`; an Observe-on-refusal inversion fails at the new stamp and passes at the old one, proving the old row was vacuous. |
| IN-06 | info | fixed | The `Replication adapter` catalog bullet attributed the whole stream to the core's rate-limited output, while adapter-side send failures (not rate-limited) and read errors (rate-limited) share the message. `ca79720`: the bullet names all three record sources and their limiter status; the catalog level test and the doc anchors stay green. |
| New-1 | info | fixed | (Re-verification) S-PERS-11 cited `internal/app/persist/snapshot.go:291-308`; the file has 307 lines and `replayRecord` ends at 307. Fixed inline by the orchestrator in `9fb66ed`: the citation is `291-307`, and the registry-wide range check now passes for every cited `.go` path. |
| New-2 | info | fixed | (Re-verification) S-HLC-15 cited `docs/OPERATIONS.md:87-108` (record-limits text) for the time-synchronisation claim. Fixed inline: the row cites `docs/OPERATIONS.md:264-268`, the clock-drift runbook's action that states correctness is independent of wall-clock sync. |
| New-3 | info | deferred | (Re-verification) Pre-existing in-range symbol drift in rows the WR-04 fix did not touch: S-HLC-04/S-HLC-10 (`core.go:1510-1543`, `store.go:295-299`), S-DM-04/S-DM-08 (`store.go:252-273`, `:306-328`, `:376-415`, `:178-186`), S-ALG-REP-02/03 (`core.go:1489-1539`, `:1487-1530`), S-ALG-CU-07 (`grpcapi/server.go:433-454`), S-ALG-CU-08 (`metrics.go:405-445`), S-DM-05/S-PAR-05 (`store.go:148-152`). Same class as WR-04; the Phase-9 registry audit owns the full resweep (recorded in `deferred-items.md`). |

**Legend:** `open` — seen, untriaged · `fixed` — resolved (record the commit) · `skipped` — will
not fix (say why) · `deferred` — postponed (name the phase).

**Fix gates (2026-11-23, fixer run on the committed tree):** `make lint` clean (twice at the end);
focused `-race` suites green for `internal/config`, `internal/udprepl`, `internal/app/repl`,
`cmd/sup-kv` and the doc anchors; `go build ./...` and the darwin build/vet green; full `make test`
run 1 failed only on the documented Phase-8 wall-clock flake (`TestCore_keyCountCostFlat`, DataSync
4.17) and run 2 was fully green (repl 246.5 s); frozen hashes, the `.golangci.yml` 1-64 hash and the
`go.mod` require-block hash all unchanged.

**Post-fix gates (2026-11-23, final tree `9fb66ed`):** `make lint` clean; full `make test` green on the
retried run (`internal/app/repl` 500 s, exit 0 — the first run failed only the recorded Phase-5
`TestScenario_dumpNotify` virtual-time flake, focused rerun green); `make buf-lint`, `make
buf-breaking` and `make client-example` green; `GOOS=darwin go build/vet` green; `bash
scripts/red-runner` exit 0 — 7 expectation rows, 0 errors, both mutants killed (evidence
`20261123T134218Z`); frozen hashes `519789bd…`/`573fed68…`/`add45b81…`, `.golangci.yml` 1-64
`a121e046…` and the first `go.mod` require block `9679f143…` all unchanged.

---

**Re-verification verdict (same reviewer, read-only, independent reproducers, 2026-11-23):**

Head `44cce0d`, isolated `/tmp` copy from `git archive HEAD`; every reproducer was re-run from source, not read from the fix commits. **All ten findings are FIXED; none is still open.** Three new registry defects were found (New-1 is a pre-existing out-of-range citation that the WR-04 fix narrowed but did not close; New-2 and New-3 are pre-existing and were missed by my round-1 audit); none is a runtime regression.

| Finding | Verdict | Independent proof |
|---|---|---|
| WR-01 | FIXED | `--grpc-addr=` exits 2 with `address-shape: ... --grpc-addr must not be empty (the default is 127.0.0.1:7901)`; no socket work, `ss` shows no listener. |
| WR-02 | FIXED | `--grpc-addr=127.0.0.1:99999`, `:-1`, `:65536` each exit 2 naming the flag and `outside 0..65535`; `--broadcast-addr=127.255.255.255:0` exits 2 with `can never be sent to`; a node with `--bind-addr`, `--grpc-addr` and `--metrics-addr` all at port 0 starts, serves (`ss` shows both TCP listeners, gate opened) and exits 0 on SIGTERM. |
| WR-03 | FIXED | Exit-1 names only the three real stages (`replication_bind`/`grpc_listen`/`metrics_listen`); "dump open" is absent and held out by `TestOperationsDoc`; an unwritable `--dump-path` probe starts (`Dump load outcome=prompt`), logs `Final dump failed`, and exits 3, exactly as the corrected paragraph states. |
| WR-04 | FIXED (named citations) | Zero `internal/udprepl/repl.go` references remain; 352 numeric citation segments resolve to existing files, with one out-of-range (New-1); the S-PAR/S-AVL/S-IMP-07 rows now cite `internal/config/{config,validate}.go` and each replacement symbol was spot-checked at its cited lines. Residual pre-existing drift filed as New-2/New-3. |
| IN-01 | FIXED | `--drift-warn=2562047h47m16.854775807s --max-clock-lead=2m` exits 2 naming both flags; `MaxClockLead-DriftWarn < 1ms` is overflow-free; the boundary pair `--drift-warn=2m --max-clock-lead=2m1ms` still starts. |
| IN-02 | FIXED | The `Observe` contract now states a positive return is a received lead past `drift_warn`, counted once per part in `MergeStats.DriftWarnings` and reported through `OnDrift`/`Log` under the shared limiter — exactly `handleDataSync`'s `if drift > 0 { count; emitDrift }` and `emitDrift`'s `allowWarn`/`Log` fallback. |
| IN-03 | FIXED | The `AGENTS.md` bullet now says `main()` parses via `internal/config`, handles help/version, installs logging and calls `run`, with signals/shutdown in `run`/`shutdown` — matching `main()`'s actual body. |
| IN-04 | FIXED | Deliberate mirror swap fails `TestMirror_listingCounters` naming `ListingDropped() = 4, want the core's 9` and `SyncStreamDropped() = 9, want the core's 4`; restored, it passes. Both core counts are non-zero and distinct. |
| IN-05 | FIXED | Observe-on-refusal inversion fails `MergeStats() = {Conflicts:0 DriftWarnings:1} after a refused apply, want no counters`; with the pre-fix `+5000 ms` stamp the same inversion passes (proving the old row was vacuous); restored, `TestMergeStats_counts` passes. |
| IN-06 | FIXED | The bullet names all three sources and their limiter status; code matches: core `Log` WARNs are `warnRateLimited` (proto_version, duplicate node_id, apply failure, unbound_listing); `write` logs one `udprepl: send to ...` per failed datagram with no limiter; `logReadError` is limited to one per 1 s with the loop continuing. |

**New defects found:**

- **New-1 (info, pre-existing; `212d27a` narrowed it but did not close it).** `docs/SPEC-DEVIATIONS.md:169` (S-PERS-11) cites `internal/app/persist/snapshot.go:291-308`; the file has 307 lines and `replayRecord` ends at 307. The round-1 citation was `291-310` (also out of range); the WR-04 fix reduced it to `291-308`, still one line past EOF. Failure: the ledger's own "all in range" checker rejects the registry, and an auditor following the range reads past the file. Fix: `291-307`.
- **New-2 (info, pre-existing; missed in round 1).** `docs/SPEC-DEVIATIONS.md:63` (S-HLC-15) cites `docs/OPERATIONS.md:87-108` for "time synchronisation is recommended practice, never a correctness requirement"; that range is the record-limits/configuration text. The substantiating sentence is `docs/OPERATIONS.md:264-268` ("correctness never depends on wall-clock synchronization"). Failure: the registry cannot substantiate the row from the cited lines. Fix: cite 264-268 (the range S-IMP-07 now uses).
- **New-3 (info, pre-existing at the round-1 head; caused by the phase's own code growth, not by the fix commits).** In-range citations now pointing at the wrong symbol: S-HLC-04/S-HLC-10 cite `internal/app/repl/core.go:1510-1543` (`handleDataSync` → 1794) and `internal/app/storage/store.go:295-299` (`Observe` → 377); S-DM-04/S-DM-08 cite `store.go:252-273` (`Set` → 283), `:306-328` (`MergeChanged` → 349), `:376-415` (`join` → 417) and `:178-186` (`RecordsEqual` → 194); S-ALG-REP-02/S-ALG-REP-03 cite `core.go:1489-1539`/`:1487-1530` (`applyRemote` → 1608); S-ALG-CU-07 cites `grpcapi/server.go:433-454` (`mapError` → 493); S-ALG-CU-08 cites `metrics.go:405-445` (`dumpFamilies` → 457); S-DM-05/S-PAR-05 cite `store.go:148-152` (`Store.ValueMaxBytes` → 161). Failure: an auditor following these rows lands on unrelated code — the same class as WR-04. Phase 9 owns the full registry audit per 07-10; the ledger's "every replacement symbol was grep-verified" holds for the rows the fix touched, not for the registry as a whole.

**Regression sweep (head `44cce0d`, isolated copy):**

- `golangci-lint config verify && golangci-lint run ./...`: exit 0.
- `go test -race -count=1`: green for `internal/config`, `internal/udprepl`, `internal/grpcapi`, `internal/metrics`, `cmd/sup-kv`, `internal/stand`.
- `internal/app/repl`: two package runs failed only `TestCore_keyCountCostFlat` (DataRequest ratio 2.39, then 4.89) — one of the two recorded Phase-8 wall-clock ratio gates — while `TestCore_unackedCostFlat` passed in both; a standalone `-run TestCore_keyCountCostFlat` then passed (ratios 0.22/0.20) at host load average 16.49 on 16 cores. Classified **environmental, not a finding**: the fix commits change no executable code in that package (`fce532b` is comment-only; `40c6bbd`/`7a1f8af` are tests), and round 1 ran the same package green (104.9 s) on a calmer host.
- Frozen red evidence `519789bd…`, `573fed68…`, `add45b81…`: all match. `.golangci.yml` lines 1-64 `a121e046…`: matches. First `go.mod` require block `9679f143…`: matches.
- `GOOS=darwin go build ./...` and `GOOS=darwin go vet ./...`: green. Host `go build ./...`: green.
