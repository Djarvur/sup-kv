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
| WR-04 | warning | fixed | The registry cited the deleted `internal/udprepl/repl.go` (seven rows) and pre-extraction `main.go` line ranges (the parameter table). `212d27a`: every `repl.go` citation repointed to the shipped split files with the symbol/line, and the `S-PAR-*`/`S-AVL-*`/`S-IMP-07` rows repointed to `internal/config/{config,validate,usage}.go`; a line-range checker over every cited `.go` path reports all in range, and each replacement symbol was grep-verified. |
| IN-01 | info | fixed | `checkDriftVsLead` added 1 ms before comparing, so a `drift_warn` near the duration maximum wrapped negative and an invalid relation validated. `7a4d8c6`: subtraction form (`MaxClockLead-DriftWarn < 1ms`), table row at `DriftWarn = math.MaxInt64`; a deliberate revert to the additive form makes the new row fail, so the test discriminates both directions. |
| IN-02 | info | fixed | The `StateStore.Observe` contract claimed its drift was not counted, while `handleDataSync` counts exactly that return. `fce532b`: the interface doc now states the positive return is counted in `MergeStats.DriftWarnings` and reported through `OnDrift`/`Log` under the shared limiter; the file's line count is unchanged, so no registry citation shifted. |
| IN-03 | info | fixed | `AGENTS.md` still said flags/env/signals live in `main()` and that it holds no testable logic, contradicting the config extraction and the tested `run` seam. `a982d19`: the bullet now describes `main()` as parse → help/version → logging → `run`, with signals and shutdown order in `run`/`shutdown`. |
| IN-04 | info | fixed | `TestMirror_listingCounters` compared two counters that were zero on both sides, so a missing or swapped mirror store passed. `7a1f8af`: each of the four counters is driven with a distinct non-zero sample (core `ListingDropped=9`, `SyncStreamDropped=4`); a deliberate swap in `mirrorStats` fails the test naming both values. |
| IN-05 | info | fixed | The refused-apply row stamped its record inside the drift window, so `DriftWarnings == 0` was guaranteed by the stamp rather than by the refusal. `40c6bbd`: the record is stamped at `test_now + drift_warn + 1s`; an Observe-on-refusal inversion fails at the new stamp and passes at the old one, proving the old row was vacuous. |
| IN-06 | info | fixed | The `Replication adapter` catalog bullet attributed the whole stream to the core's rate-limited output, while adapter-side send failures (not rate-limited) and read errors (rate-limited) share the message. `ca79720`: the bullet names all three record sources and their limiter status; the catalog level test and the doc anchors stay green. |

**Legend:** `open` — seen, untriaged · `fixed` — resolved (record the commit) · `skipped` — will
not fix (say why) · `deferred` — postponed (name the phase).

**Fix gates (2026-11-23, fixer run on the committed tree):** `make lint` clean (twice at the end);
focused `-race` suites green for `internal/config`, `internal/udprepl`, `internal/app/repl`,
`cmd/sup-kv` and the doc anchors; `go build ./...` and the darwin build/vet green; full `make test`
run 1 failed only on the documented Phase-8 wall-clock flake (`TestCore_keyCountCostFlat`, DataSync
4.17) and run 2 was fully green (repl 246.5 s); frozen hashes, the `.golangci.yml` 1-64 hash and the
`go.mod` require-block hash all unchanged.
