---
phase: 07-observability-and-configuration
reviewed: 2026-11-23T12:40:34Z
base: 31c9fe4
head: 6fdecfbd5a1df0e20020ba8547c3080a0a470a0e
depth: deep
files_reviewed: 54
files_reviewed_list:
  - .golangci.yml
  - AGENTS.md
  - cmd/sup-kv/catalog_internal_test.go
  - cmd/sup-kv/main.go
  - cmd/sup-kv/main_test.go
  - cmd/sup-kv/operations_test.go
  - cmd/sup-kv/run_internal_test.go
  - docs/04-MULTI-NODE.md
  - docs/OPERATIONS.md
  - docs/SPEC-DEVIATIONS.md
  - internal/app/clock/clock.go
  - internal/app/clock/clock_test.go
  - internal/app/clock/hlc_prop_test.go
  - internal/app/internal/testutil/gen.go
  - internal/app/persist/codec.go
  - internal/app/persist/codec_test.go
  - internal/app/persist/doc.go
  - internal/app/repl/core.go
  - internal/app/repl/discard_test.go
  - internal/app/repl/events.go
  - internal/app/repl/events_test.go
  - internal/app/repl/listedkeys_internal_test.go
  - internal/app/repl/merge_stats_test.go
  - internal/app/repl/retransmit_test.go
  - internal/app/repl/sentbytype.go
  - internal/app/repl/sim_scenario_test.go
  - internal/app/repl/warn_test.go
  - internal/app/storage/store.go
  - internal/app/storage/store_test.go
  - internal/app/version/validate_test.go
  - internal/app/version/version.go
  - internal/config/config.go
  - internal/config/config_test.go
  - internal/config/usage.go
  - internal/config/usage_test.go
  - internal/config/validate.go
  - internal/config/validate_test.go
  - internal/grpcapi/gate.go
  - internal/grpcapi/gate_test.go
  - internal/metrics/metrics.go
  - internal/metrics/metrics_definitions_test.go
  - internal/metrics/metrics_test.go
  - internal/metrics/rss_linux.go
  - internal/metrics/rss_linux_internal_test.go
  - internal/metrics/rss_other.go
  - internal/stand/stand_metrics_test.go
  - internal/udprepl/adapter.go
  - internal/udprepl/codec.go
  - internal/udprepl/decode_negative_test.go
  - internal/udprepl/loop.go
  - internal/udprepl/mirror.go
  - internal/udprepl/mirror_internal_test.go
  - internal/udprepl/scenario_test.go
  - internal/udprepl/socket.go
findings:
  critical: 0
  warning: 4
  info: 6
  total: 10
  necessity: 0
status: issues_found
---

# Phase 7: Code Review Report

**Reviewed:** 2026-11-23T12:40:34Z (UTC)
**Base:** `31c9fe4` · **Head:** `6fdecfb`
**Depth:** deep
**Files Reviewed:** 54 changed paths (55 paths changed since the base; `internal/udprepl/repl.go` was deleted and was reviewed at the base commit)
**Status:** issues_found

## Summary

Deep, adversarial review of the phase's committed work against its own contract (`07-CONTEXT.md` D-01…D-08 plus the research resolutions, `07-VALIDATION.md`, the 07-02/05/06/08/09/10/11 plans, `AGENTS.md`, and the operator docs the phase wrote). Every changed source file was read in full; the deleted `internal/udprepl/repl.go` was reconstructed from the base commit and compared function by function with its successors. Findings were re-derived from source, not from the phase's summaries.

**No host findings of the critical tier were found.** No `<structural_findings>` block and no `<external_reviewer_evidence>` block were supplied, so the narrative section below is the whole of the review. The necessity rubric was read and applied: no new branch in this phase guards a provably impossible state (the new renderer/enum fallbacks, the `RunGate` deadline normalization and the RSS platform branch are either reachable through an exported API, a trust boundary or a portability target), so `necessity: 0`.

The phase's core engineering is sound: the `internal/config` rule table is genuinely single-point and boundary-tested; the `udprepl` split is behavior-preserving except for the intended send-accounting change; the conflict counter now walks the pending set and is proven against that definition; the drift signal moved to the received reading (both in the clock and in the core) and is counted and rate-limited as documented; the `unbound_listing` class, the four Phase-4 counters, the RSS split and the exit-code precedence all landed. `make lint` on an isolated copy of the committed tree is green, the depguard rules fire when probed, the frozen hashes and the no-new-dependency hash match, the darwin build/vet pass, and every package test I ran under `-race` is green. Passing tests were not taken as correctness evidence; the defects below were reproduced with process-level probes.

The four warnings are: a newly added address rule that does not close two holes it claims to cover (an explicitly empty `--grpc-addr` binds the unauthenticated gRPC API to every interface; ports outside 0..65535 and port 0 on the broadcast address pass validation and surface as exit 1 or a mute node); one false claim in the exit table (`dump open` as an exit-1 startup failure); and a registry left citing the deleted `repl.go` and pre-extraction `main.go` lines. Six info findings record an overflow edge in the new drift/lead rule, three stale contract statements the phase's own behavior invalidates, and two tests whose assertions cannot discriminate the behavior they claim.

## Narrative Findings (AI reviewer)

| ID | Severity | Title | File:line | Evidence | Failure scenario |
|----|----------|-------|-----------|----------|------------------|
| WR-01 | warning | Empty `--grpc-addr` passes validation and binds the unauthenticated API to all interfaces | `internal/config/validate.go:286-320`; `cmd/sup-kv/main.go:794-804`; `docs/OPERATIONS.md:17-21` | `checkAddressShape` skips every empty value; `checkRequiredKnobs` does not list `--grpc-addr`; `net.Listen("tcp", "")` binds the wildcard address on an ephemeral port. Probe: `--grpc-addr=` started with startup record `"grpc_addr":""`, `ss -lntp` showed `LISTEN *:37889`, health answered `SERVING` and `Get` executed over the host LAN IP `192.168.7.117:37889`. | A unit file or script writes `--grpc-addr=$GRPC_ADDR` with the variable unset (or a typo leaves the value empty). The node starts cleanly and serves reads/writes/deletes/snapshot RPCs, plus gRPC reflection, to every host that can scan the ephemeral port — the opposite of the documented "default `--grpc-addr` binds loopback only". The failure is silent: no WARN, exit 0, and the startup record shows an empty string. |
| WR-02 | warning | Address-shape rule accepts ports outside 0..65535 and port 0 on the broadcast destination | `internal/config/validate.go:290-320`; `docs/OPERATIONS.md:382-388` | `validHostPort` accepts any `strconv.Atoi`-parsable port. Probes: `--grpc-addr=127.0.0.1:99999` / `:-1` / `:65536` validated, then exited 1 with `Startup failed stage=grpc_listen`; `--broadcast-addr=127.255.255.255:0` started and emitted `Replication adapter ... sendto: invalid argument` on every Hello. | An operator mistypes a port (`--grpc-addr=127.0.0.1:79011`). The documented contract says a bad flag value exits 2 with the usage text; instead the node does socket work first and exits 1, so a systemd `Restart=on-failure` unit restart-loops on a permanent typo instead of failing fast. A broadcast port of 0 (or any port the kernel refuses) starts a node that can never send a datagram, with one non-rate-limited WARN per Hello/retransmit and no validation error. |
| WR-03 | warning | Exit-1 row lists `dump open` as a startup failure that does not exist | `docs/OPERATIONS.md:382-386`; `internal/app/persist/loader.go:40-101`; `cmd/sup-kv/main.go:216-241` | `persist.Open` returns `(*Engine, int)` and no error; load failures become the `prompt`/`backup`/`unusable` outcome with their own log records, and `wire` has only three fatal stages (`replication_bind`, `grpc_listen`, `metrics_listen`). Probe: `--dump-path=/proc/supkv-probe-dump.json` started normally (`Dump load outcome=prompt`), and on SIGTERM logged `Final dump failed ... open /proc/.tmp~...: no such file or directory` and exited 3. | An operator hitting a bad or unwritable dump path reads the runbook expecting exit 1 and a supervisor restart; the real behavior is a running node with a one-line ERROR at load and, on shutdown, exit 3. Incident response is aimed at the wrong code and the wrong remediation, and `Restart=on-failure` never fires because the node stays up. |
| WR-04 | warning | Registry cites the deleted `internal/udprepl/repl.go` and pre-extraction `main.go` line ranges | `docs/SPEC-DEVIATIONS.md:69,70,72,81,96,97,213`; also `:219-220,228-243,255` | Seven rows (`S-PROTO-01/02/04/13`, `S-ALG-ST-03/04`, `S-FAIL-18`) cite `internal/udprepl/repl.go`, which commit `8903b0f` deleted. The `S-PAR-*`, `S-AVL-*` and `S-IMP-07` rows cite `cmd/sup-kv/main.go` line numbers (e.g. `main.go:294-296,693-695`) that the config extraction invalidated; those line ranges now name `logStartup`, `candidateFailure` and the startup attribute list. | An auditor following the compliance registry — the phase's own T-07-31 "overclaimed closure" guard — lands on a nonexistent file for seven rows, and on unrelated code for the whole parameter table. The five rows the phase flipped are current; the surrounding registry is now less traceable than before the phase. |
| IN-01 | info | `checkDriftVsLead` integer-overflow bypass | `internal/config/validate.go:260-269` | `DriftWarn+time.Millisecond` overflows int64 for a `drift_warn` near the duration maximum. Probe: `Validate` with `DriftWarn = math.MaxInt64` and `MaxClockLead = 2m` returns `nil`, though the documented relation is violated. Reachable from the CLI: `--drift-warn=2562047h47m16.854775807s --max-clock-lead=2m` passes. | An operator (or a generated config) sets an extreme `drift_warn`. Startup accepts a relation the runbook says it enforces, and the reachability invariant the rule exists for (`max_clock_lead >= drift_warn + 1ms`) is false for the accepted configuration. |
| IN-02 | info | `StateStore.Observe` contract says its drift is not counted, but the core counts it | `internal/app/repl/core.go:425-429` vs `:1801-1810` | The interface comment ends "The returned drift is the clock's own signal; MergeStats does not count it this phase", while `handleDataSync` increments `MergeStats.DriftWarnings` and emits `OnDrift` from exactly that return. | A maintainer trusting the interface contract removes the `Observe` return from the drift accounting as out-of-contract, silently reverting OBSV-05 for listing-part maxima with no failing test at the interface boundary. |
| IN-03 | info | `AGENTS.md` layout still says flags/env/signals live in `main()` and that it holds no testable logic | `AGENTS.md:49-50` | The same file's new `internal/config/` bullet says parsing/validation moved out, and `cmd/sup-kv/run_internal_test.go` now tests the lifecycle seam. | An architect reading the layout contract concludes `main.go` still owns flag/env registration and that there is no testable lifecycle logic, contradicting D-08 and the file's own next bullet; reviewers re-open "where does config live" and may reintroduce registration in `main`. |
| IN-04 | info | `TestMirror_listingCounters` cannot detect a missing or swapped mirror for two of the four counters | `internal/udprepl/mirror_internal_test.go:294-308` | The loop compares each adapter accessor with the core accessor, but `ListingDropped` and `SyncStreamDropped` are zero on both sides in this test. A `mirrorStats` that never stored them, or stored them into each other's atomics, passes unchanged; the comment at `:236` claims "a missing or swapped mirror store cannot pass". | The exact regression IN-03 routed to this phase — a mirror store wired to the wrong core accessor — ships green, and `supkv_replication_listings_dropped_total` / `supkv_replication_sync_streams_dropped_total` silently swap meanings until an operator notices a counter that never moves. |
| IN-05 | info | The refused-apply test row's drift assertion cannot discriminate the refused-apply drift behavior | `internal/app/repl/merge_stats_test.go:208-232` | The row stamps the refused record at `testNowMs+5000`, inside the 60 s `drift_warn` window, so `DriftWarnings == 0` holds whether or not a refused apply would otherwise count. Only the conflict half of the row is meaningful. | A change that starts (or stops) counting drift for records the store refuses passes this row; the definitional edge — a leading version refused by the value limit — stays unpinned in either direction. |
| IN-06 | info | The `Replication adapter` catalog entry under-describes the records sharing that message, and one class is not rate-limited | `docs/OPERATIONS.md:366-370`; `internal/udprepl/loop.go:514-537,562-568`; `cmd/sup-kv/main.go:415-417` | The bullet says the record carries "the core's rate-limited adaptation and discard details", but the same `logAdapterWarn` renderer also carries adapter-side send failures (unbounded: one per failed `WriteToUDP`) and rate-limited read errors. Probe: a refused broadcast destination immediately produced two `Replication adapter` WARNs with `sendto: invalid argument`. | An operator greps "Replication adapter", reads the runbook, and concludes the stream is rate-limited core output; a broken broadcast destination instead produces a steady WARN stream from a path the runbook does not describe, drowning the rate-limited events it does describe. |

### WR-01: Empty `--grpc-addr` passes validation and binds the unauthenticated API to all interfaces

**File:** `internal/config/validate.go:286-320`, `cmd/sup-kv/main.go:794-804`, `docs/OPERATIONS.md:17-21`

**Issue.** `checkAddressShape` deliberately skips an empty value, and its comment names only two legitimate empty cases — `*required-knobs*`' finding and the metrics endpoint's documented "no listener" opt-in. `--grpc-addr` is neither: it has a design default (`127.0.0.1:7901`), it is absent from `checkRequiredKnobs`, and an explicitly empty flag value reaches `listenGRPC` unchanged. `net.ListenConfig.Listen(ctx, "tcp", "")` binds the wildcard address on an ephemeral port.

**Failure scenario.** `--grpc-addr=$GRPC_ADDR` with `GRPC_ADDR` unset (or any wrapper that emits `--grpc-addr=`) starts a node whose gRPC API — no authentication, no authorization, reflection on — is reachable from every host that can reach the ephemeral port, while the runbook promises a loopback default and asks for a deliberate wider bind. Reproduced: startup record `"grpc_addr":""`, `ss` showing `*:37889`, health `SERVING`, and a `Get` answered over `192.168.7.117:37889`.

**Fix.** Reject an empty `--grpc-addr` (it can never mean "no listener"; the metrics address is the only opt-in), e.g. in `checkAddressShape`:

```go
if c.GRPCAddr == "" {
    errs = append(errs, fmt.Errorf("%w: --grpc-addr must not be empty (the default is %s)", errAddressShape, defaultGRPCAddr))
}
```

and add a `TestValidate` row plus a `TestMain_validationRelations` row for it. This behavior predates the phase (the old `main.go` had no address rule at all), but the new rule is the place it is now owned, and the rule's own comment currently claims the empty surface is covered.

### WR-02: Address-shape rule accepts out-of-range ports and port 0 on the broadcast destination

**File:** `internal/config/validate.go:290-320`, `docs/OPERATIONS.md:382-388`

**Issue.** `validHostPort` only proves `net.SplitHostPort` succeeds and the port parses as an `int`. Negative ports, ports above 65535 and port 0 all pass. The broadcast address is a *destination*, so port 0 there can never work.

**Failure scenario.** (a) `--grpc-addr=127.0.0.1:79011` (or `:-1`, or `:65536`) passes validation, does the engine start and both socket binds, then exits 1 through `Startup failed stage=grpc_listen` — while `docs/OPERATIONS.md` says a bad flag value exits 2 with the usage text. Under `Restart=on-failure` this converts a permanent typo into a restart loop. (b) `--broadcast-addr=127.255.255.255:0` starts a node whose every send fails (`sendto: invalid argument`), so the node never replicates, one WARN per Hello and per retransmission. Reproduced with both inputs.

**Fix.** Validate the numeric range in `validHostPort` and reject the unusable destination port explicitly:

```go
p, err := strconv.Atoi(port)
if err != nil || p < 0 || p > 65535 {
    return false
}
// a listener may legitimately use 0; a broadcast destination may not
```

plus a `checkBroadcastPort` rule (or an extension of `checkAddressShape`) rejecting port 0 for `--broadcast-addr`, with table rows. If port 0 is to stay legal for listeners, make that an explicit carve-out rather than an accident of `Atoi`.

### WR-03: Exit-1 row lists `dump open` as a startup failure that does not exist

**File:** `docs/OPERATIONS.md:382-386`, `internal/app/persist/loader.go:40-101`, `cmd/sup-kv/main.go:216-241`

**Issue.** `persist.Open` cannot fail: `port.DumpStore.Load` reports an outcome (`prompt`/`main`/`backup`/`unusable`) with per-candidate failures, and `startEngine` always returns an engine. The only fatal wiring stages are the UDP bind, the gRPC listen and the metrics listen. A dump-path problem is a load outcome plus (for a non-`main` outcome) a dirty engine.

**Failure scenario.** With `--dump-path` on an unwritable filesystem the node starts, logs one ERROR `No usable dump generation`/`Dump load`, and at SIGTERM logs `Final dump failed` and exits 3. The runbook's exit-1 cause ("dump open") is unreachable, so the documented exit-code table disagrees with the shipped binary on a case the phase's own acceptance (OBSV-04) says must match.

**Fix.** Drop "dump open" from the exit-1 bullet and state the real behavior: a load or dump-path problem logs its catalog record and the node starts; failure to write surfaces as a per-attempt ERROR, and a failed final dump exits 3. Alternatively implement a startup writability probe and make it a stage — but then it must appear in the stage list and the process tests.

### WR-04: Registry cites the deleted `repl.go` and pre-extraction `main.go` line ranges

**File:** `docs/SPEC-DEVIATIONS.md:69,70,72,81,96,97,213`; also `:219-220,228-243,255`

**Issue.** The phase's 07-01 decomposition deleted `internal/udprepl/repl.go` and 07-02 moved the whole flag/default/validation table out of `cmd/sup-kv/main.go`. The phase updated the five rows its plan named and left the rest, so seven rows now point at a file that does not exist and the S-PAR/S-AVL/S-IMP rows point at line numbers that now name unrelated code (`logStartup`'s attribute list, `candidateFailure`, `signalBuffer`). I verified each cited range against the head tree.

**Failure scenario.** The registry is the audited spec-traceability artifact (each row is a compliance claim with a link). An auditor or a Phase-9 registry audit following `S-FAIL-18` or `S-PROTO-01` finds no `repl.go`; following `S-PAR-01` finds `logStartup` where the node-id rule is supposed to live. The phase's own T-07-31 threat ("an overclaimed deviation closure") is realized for rows it did not flip.

**Fix.** Point the seven `repl.go` citations at the split files (`codec.go`, `socket.go`, `loop.go`, `mirror.go`, `adapter.go`) and the parameter rows at `internal/config/config.go`/`validate.go` (symbols rather than line ranges where practical). If the plan's "touch only five rows" scope is to be honoured, record the remainder in `deferred-items.md` with Phase 9 as the owner — currently neither is done.

### IN-01: `checkDriftVsLead` integer-overflow bypass

**File:** `internal/config/validate.go:260-269`

**Issue.** The rule computes `c.DriftWarn+time.Millisecond` before comparing. Both operands are user-controlled `time.Duration` values; near `math.MaxInt64` the addition wraps negative, and the comparison `MaxClockLead < negative` is false, so an invalid pair validates.

**Failure scenario.** `--drift-warn=2562047h47m16.854775807s --max-clock-lead=2m` starts (probe: `Validate(...) = nil`), violating the relation the startup log and OPERATIONS claim is enforced.

**Fix.** Use the subtraction form, which cannot overflow for two positive durations: `if c.MaxClockLead-c.DriftWarn < time.Millisecond`. Add a test row at `DriftWarn = math.MaxInt64`.

### IN-02: `StateStore.Observe` contract says its drift is not counted

**File:** `internal/app/repl/core.go:425-429` (contract) vs `:1801-1810` (behavior)

**Issue.** The interface doc still ends "The returned drift is the clock's own signal; MergeStats does not count it this phase", a sentence 07-05 invalidated when `handleDataSync` began counting the return.

**Failure scenario.** A maintainer or a future refactor trusts the interface text and removes the counting as an out-of-contract side effect; OBSV-05 for listing-part maxima silently disappears and no interface-boundary test objects.

**Fix.** Rewrite the `Observe` doc to state that a positive return is counted in `MergeStats.DriftWarnings` and reported through `OnDrift`/`Log` by `handleDataSync` under the shared limiter.

### IN-03: `AGENTS.md` layout still says flags/env/signals live in `main()`

**File:** `AGENTS.md:49-50`

**Issue.** The bullet reads "`main()` holds no testable logic (flags/env/signals live there from Phase 5, nothing else)", but this phase moved flags/env to `internal/config` and extracted the lifecycle into the tested `run` seam. The file's own next bullet and `run_internal_test.go` contradict it.

**Failure scenario.** Architecture reviews and new plans cite the layout contract; the false sentence invites reintroducing registration in `main` and hides that `run` is the tested seam.

**Fix.** Replace with: "`main()` parses (via `internal/config`), handles help/version, installs logging and calls `run`, the tested lifecycle seam; signals and the shutdown order live in `run`/`shutdown`."

### IN-04: `TestMirror_listingCounters` cannot detect a missing or swapped mirror for two counters

**File:** `internal/udprepl/mirror_internal_test.go:294-308`

**Issue.** Only `NotificationsRegressed` and `ListedKeysEvicted` are driven non-zero. For `ListingDropped` and `SyncStreamDropped` the adapter and core values are both zero, so the equality assertion is vacuous and the comment's "a missing or swapped mirror store cannot pass" is false.

**Failure scenario.** `a.listingDropped.Store(a.core.SyncStreamDropped())` (or omission of one store) passes the whole suite; the two published counters then swap or freeze with no test failure.

**Fix.** Drive one non-zero sample for each (a `listingFits`-refused trigger for `ListingDropped`, an accumulator-bound eviction for `SyncStreamDropped`), or assert the atomics through distinct sentinel values injected into the core; if that is impractical, delete the claim from the comment so the gap is not overstated.

### IN-05: The refused-apply row's drift assertion cannot discriminate

**File:** `internal/app/repl/merge_stats_test.go:208-232`

**Issue.** The row's record is stamped `testNowMs+5000`, inside the 60 s window, so `DriftWarnings == 0` is guaranteed by the stamp, not by the refusal path. The row title claims both counters.

**Failure scenario.** A change that counts drift before the apply outcome (or after, for refused applies) passes this row in either direction; the definitional edge is unpinned.

**Fix.** Stamp the refused record at `testNowMs + drift_warn + 1s` (still inside `MaxClockLead`), so the assertion actually distinguishes "refused ⇒ no count" from "received ⇒ count".

### IN-06: The `Replication adapter` catalog entry under-describes the records sharing that message

**File:** `docs/OPERATIONS.md:366-370`; `internal/udprepl/loop.go:514-537,562-568`; `cmd/sup-kv/main.go:415-417`

**Issue.** The catalog bullet attributes the whole `Replication adapter` stream to "the core's rate-limited adaptation and discard details". `logAdapterWarn` is also the sink for adapter-side send failures (`flush`/`write`, one record per failed `WriteToUDP`, no limiter) and for rate-limited read errors.

**Failure scenario.** A broadcast destination the kernel refuses produces a permanent WARN stream under a message the runbook describes as rate-limited core output; an operator cannot tell from the doc which of the two streams they are seeing or where to look.

**Fix.** Extend the bullet to name the adapter-side send-failure and read-error details, state that send failures are not rate-limited, and keep the core's rate-limited classes as the documented subset they are.

## Verified clean

Each item below was probed directly, not inferred from the phase's summaries.

- **Depguard boundary actually fires.** In an isolated copy I imported `internal/udprepl` from a non-test file under `internal/app/**` and `internal/metrics` from a non-test file under `internal/grpcapi/**`: `golangci-lint run --enable-only depguard` reported exactly the three expected rules (`sup-kv-adapters-not-in-core`, `sup-kv-metrics-no-adapters`, `sup-kv-grpcapi-no-repl-metrics`). The same `udprepl` import in a `_test.go` file produced no finding, confirming `!$test` works. Full `golangci-lint run ./...` on the committed tree: exit 0.
- **Vendored lint body and dependency policy.** `sed -n '1,64p' .golangci.yml` hashes to `a121e0466bc211e48ca898f6c07434dc50eba07c`; the `.golangci.yml` diff is purely additive inside `linters.settings.depguard.rules`; the first `require` block of `go.mod` hashes to `9679f1431d1815116871a4f7af79e727283a2875`. No adapter imports exist in non-test files under `internal/app` or `internal/port`.
- **Frozen red evidence.** `git hash-object` for `internal/grpcapi/red_gate_test.go`, `internal/stand/stand_red_test.go` and `scripts/red-expectations.tsv` matches the three pinned literals.
- **Darwin portability.** `GOOS=darwin go build ./... && GOOS=darwin go vet ./...` pass; the `rss_linux.go`/`rss_other.go` split compiles on both sides.
- **Test suite.** `go test -race -count=1` green for `internal/config`, `internal/app/clock`, `internal/metrics`, `internal/app/persist`, `internal/app/repl`, `internal/udprepl`, `internal/grpcapi`, `internal/persistfile`, `internal/stand`, `api/supkvapi/v1`, `cmd/sup-kv`.
- **Exit table.** Source (`run`'s switch) plus the five `TestRun_*` rows plus process probes: clean SIGTERM exit 0; staged wire failure exit 1 with no shutdown; `Serve` error still shuts down; dump failure exit 3 and precedence over 1; the real binary exits 1 on a duplicate bind at `SUPKV_LOG_LEVEL=error` with exactly one ERROR record and exits 3 on a blocked final dump.
- **Signal ordering and bounded shutdown.** `run` arms signals before `wire`; the fake-wire test fails if wire runs first; the second-signal force path keeps the final `FlushTimeout` after `GracefulStop`; `awaitEngineStop` abandons a stuck dumper with one WARN.
- **`udprepl` split parity.** `Run`, `NotifyLocalWrite`/`NotifyReplayedWrite`, `enqueueLocalWrite`, the tracker lifecycle, `WaitReplicated`, `readLoop`, `logReadError`, `mirrorStats`, `decode`/`encodeDatagram`, and every socket helper were compared with the base `repl.go`. The only behavioral delta is the intended one: `SentByType` now increments after `WriteToUDP` returns nil. `WriteToUDP` has exactly one production call site.
- **Conflict counter.** The code walks `Core.pending` and counts only a pending entry the applied record beats (strictly greater version, or an equal-version tie-break the incoming record won); the nine-row matrix covers confirmed overwrite, remote-to-remote, both tie-break directions, redelivery, stale, refused apply, and the two displacement directions; `TestStand_metricsByEffect` constructs the pending-write displacement on real sockets and asserts the exact value.
- **Drift signal.** `Clock.Update` compares `remoteL` against the wall (positive only when the received reading leads); the adversarial property test models `Tick` and `Update` separately; `applyRemote` counts only `drift > 0`; `handleDataSync` counts the part maximum once per part; the WARN/event is rate-limited per `(node_id, clock_drift)` while the counter stays per record; the default pair (60 s / 2 m) makes a +61 s record accepted and counted.
- **`unbound_listing`.** Counter, class token, `OnDiscard`, WARN text without "configuration error", metric label, `04-MULTI-NODE.md`, the conservation/totality sum (now including `bad_sync_header`) and the discard-label completeness all agree; the alternating-run and duplicate-identity paths still use `duplicate_node_id`.
- **Foreign `proto_version`.** Both branches (adapter verdict and decoded-envelope defence in depth) count, fire `OnDiscard` and emit one rate-limited WARN; the core-level test proves the actual message text and the limiter behaviour.
- **Config rule table.** Twelve named rules in one ordered slice, pinned by a meta-test; 13 durations plus the three numeric knobs reject 0 and negatives; boundaries at 1 ms, `repl.MaxSyncResponseRate`, `repl.MaxSyncChunkBytes`, the 64 408/64 409 datagram edge and `drift_warn+1ms`; `--version` survives a broken environment; positional tokens are rejected by name.
- **Metrics definitions and wiring.** Help text matches behavior for conflicts (pending), sent (socket-accepted), drift (received lead), dump duration (last attempt) and RSS (current on Linux); one test per definition; the closed `Sources` set has exactly two literal constructions (production and the test helper) and both set all accessors, so a scrape cannot panic.
- **Gate narration.** `OnGateOpen` fires inside the single CAS branch, after the health flip; both reasons are asserted, and a second callback is proven absent.
- **Necessity.** No new branch guards an impossible state: the empty-value skip in `checkAddressShape` is a real reachable case (the defect above), the `RunGate` deadline normalization serves zero-value library callers, the RSS platform branch is reachable on non-darwin `!linux` builds, and the loader/renderer fallbacks are total-function defaults rather than guards on a proven-absent state.

## Claims checked against code

| # | Claim (where) | Code evidence | Verdict |
|---|---------------|---------------|---------|
| 1 | Exit 0 means clean shutdown with the final dump confirmed (`OPERATIONS.md:381`) | `run`'s switch (`main.go:233-240`); `shutdown` returns nil only after `FlushTimeout` succeeds; probe exited 0 | verified |
| 2 | Exit 1 covers a startup failure (bind, listen, metrics, **dump open**) or a Serve error (`OPERATIONS.md:382-386`) | `startupError` stages `replication_bind`/`grpc_listen`/`metrics_listen`; `persist.Open` returns no error | **partially falsified** — startup/Serve true; `dump open` unreachable (WR-03) |
| 3 | Exit 2 means a malformed flag/env value, naming the parameter (`OPERATIONS.md:387-388`) | `config.Parse`/`config.Validate` → `usageError` → `os.Exit(2)`; 12 rules and env errors joined | **partially falsified** — out-of-range ports exit 1 (WR-02); empty `--grpc-addr` starts (WR-01) |
| 4 | Exit 3 is the failed final dump, including on the Serve-error path, and wins over 1 (`OPERATIONS.md:389-391`) | `run`'s switch order; `TestRun_dumpFailureCode`, `TestRun_serveAndDumpFailure`; probe exited 3 | verified |
| 5 | A Serve error still runs the bounded final dump (`OPERATIONS.md:383-385`) | `run` calls `shutdown` on both paths; `TestRun_serveErrorStillShutsDown` | verified |
| 6 | Signals are armed before dump load and bind (`OPERATIONS.md:374`) | `run`: `installSignals()` precedes `wire`; `TestRun_signalInstalledBeforeWire` fails if wire runs first | verified |
| 7 | A second signal forces the stop without skipping the final flush (`OPERATIONS.md:378-379`) | `shutdown`'s select forces `gs.Stop()` then still runs `replCancel` and `FlushTimeout`; `TestShutdown_secondSignal` | verified |
| 8 | `--help` exits 0 and lists the five env-only knobs (`OPERATIONS.md:393-395`) | `usage.go:12-18`, `main.go:166-169`; `TestUsage`, `TestMain_helpListsEnvOnly` | verified |
| 9 | One validation point owns every range and relation; every failure exits 2 naming the parameter (`OPERATIONS.md:98-104`) | `config.Validate` joins 12 rules; 13+3 positivity rows; relation boundaries tested | **partially falsified** — WR-01/WR-02 are configuration failures that do not exit 2 |
| 10 | `retransmit>=1ms`, `peer>hello`, `ack<=ack_max`, `sync_chunk` in 1..65507, `lead>=drift_warn+1ms` (`OPERATIONS.md:106-115`) | `validate.go:168-269`; boundary tests at each edge | verified for realistic inputs; **overflow edge** (IN-01) |
| 11 | `max_clock_lead` defaults to 2 m in the binary, core, help and tests (`OPERATIONS.md:89-96`) | `config.go:106`, `repl/core.go:129`, `TestMain_maxClockLead`, `TestValidate` | verified |
| 12 | Drift means a received version (record or DataSync part maximum) leading wall time by more than `drift_warn`; one rate-limited WARN per peer plus the counter (`OPERATIONS.md:239-268`, `:360-363`) | `clock.go:124-153`; `core.go:1637-1643,1801-1810`; `events.go:77-87`; `clock_test.go`, `events_test.go`, `merge_stats_test.go` | verified |
| 13 | The drift counter counts every offending record (`OPERATIONS.md:246-248`) | per-record in `applyRemote`, per-part in `handleDataSync`, `drift > 0` only | verified for applied records/parts; a store-refused record counts neither counter (definitional edge, not filed) |
| 14 | `supkv_merges_conflicts_total` counts only pending-local displacement (`OPERATIONS.md:303-306`) | `core.go:1659-1670` walks `c.pending`; nine-row matrix; `TestStand_metricsByEffect` | verified |
| 15 | `sent_total` counts only datagrams the socket accepted (`OPERATIONS.md:296-301`) | `loop.go:514-537` counts after `write` returns nil; `TestFlush_countsOnlySuccessfulWrites`; single `WriteToUDP` site | verified |
| 16 | `process_resident_memory_bytes` is current RSS on Linux, peak on dev-only darwin (`OPERATIONS.md:322-324`) | `rss_linux.go:42-52` reads `statm[1] * pagesize`; `rss_other.go` reads `ru_maxrss`; tests | verified |
| 17 | The four Phase-4 counters are published and mirrored (`OPERATIONS.md:302-313`) | `metrics.go:375-405`, `mirror.go:112-138`, `main.go:739-742` | verified in code; two mirror paths are test-vacuous (IN-04) |
| 18 | `unbound_listing` is its own class and metric label, and the WARN is not a "configuration error" (`OPERATIONS.md:288-295`) | `core.go:1840-1846`, `mirror.go:24`, `metrics.go:434`, `discard_test.go#TestDiscard_unboundListingClass` | verified |
| 19 | A foreign `proto_version` warns on both branches (`OPERATIONS.md:366-369`, `SPEC-DEVIATIONS.md` S-FAIL-17) | `core.go:1084-1104`; `TestProtoVersionWarns` | verified |
| 20 | The lifecycle catalog levels are the operator contract (`OPERATIONS.md:343-370`) | `catalog_internal_test.go#TestCatalogLevels` plus process tests for startup/load/gate/fatal-bind/final-dump | verified; the adapter bullet is not exhaustive (IN-06) |
| 21 | Load outcomes: prompt/main one INFO `Dump load`, backup one WARN, unusable one ERROR (`OPERATIONS.md:117-147`) | `main.go:481-579`; `TestMain_startupLogLoadOutcome` | verified |
| 22 | The startup record carries the effective config and no certificate/key material (`OPERATIONS.md:347-351`) | `main.go:286-317`; `TestMain_startupLogRecord` asserts the three secret-bearing keys absent | verified |
| 23 | `main()` holds no testable logic; flags/env/signals live there (`AGENTS.md:49-50`) | flags/env live in `internal/config`; the lifecycle is the tested `run` seam (`run_internal_test.go`) | **falsified** (IN-03) |
| 24 | The five registry rows are current and no other row was touched (`07-10-PLAN.md`) | the five rows cite shipped code; the untouched rows cite the deleted `repl.go` and stale `main.go` lines | five rows verified; surrounding registry **falsified** (WR-04) |
| 25 | Depguard machine-enforces the adapter direction and the vendored body is byte-unchanged (`AGENTS.md:12-37`) | isolated-copy probes fire all three new rules; `!$test` honoured; lines 1-64 hash matches; additive-only diff | verified |
| 26 | No runtime dependency was added (`07-11-PLAN.md`) | first `require` block hash matches `9679f14...` | verified |
| 27 | The frozen red evidence is byte-identical (`07-11-PLAN.md`) | all three `git hash-object` values match | verified |
| 28 | Darwin cross-build and vet pass (`07-11-PLAN.md`) | `GOOS=darwin go build ./... && go vet ./...` exit 0 | verified |

---

_Reviewed: 2026-11-23T12:40:34Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
