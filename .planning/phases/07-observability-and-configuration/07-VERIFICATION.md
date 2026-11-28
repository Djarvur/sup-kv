---
phase: 07-observability-and-configuration
verified: 2026-11-23T14:31:31Z
head: 597ac26
base: 31c9fe4
status: passed
score: 62/62 must-haves verified
must_haves:
  total: 62
  passed: 62
  human_needed: 0
  failed: 0
requirements_verified: 8/8 (OBSV-01, OBSV-02, OBSV-03, OBSV-04, OBSV-05, OBSV-06, OBSV-07, OBSV-08)
fingerprint: "v4:sha256:04fb8eb8817a1472a60d33df1d586c822cbc594c4169ebc02ef0f6bd55b18ff3"
fingerprint_files: 99
fingerprint_method: "sorted git blob hashes of every path in 31c9fe4..HEAD (HEAD blob; base blob for the one deleted path internal/udprepl/repl.go) plus every file in .planning/phases/07-observability-and-configuration, excluding this report"
re_verification: no
---

# Phase 7 Verification Report — Observability and Configuration

**Phase Goal (ROADMAP.md:283):** An operator can see what the node is doing and misconfiguration fails fast — lifecycle logs, one validation point, metric definitions that match the binary, distinct exit codes, and a machine-enforced dependency boundary.

**Verified:** 2026-11-23T14:31:31Z on head `597ac26` (phase source-work base `31c9fe4`), clean tree.
**Status:** passed — 62/62 plan-level must-have truths verified, 8/8 requirements verified.
**Mode:** initial verification (no prior `07-VERIFICATION.md`).

Everything below was re-derived from the committed tree with probes run in this session. SUMMARY.md,
`07-VALIDATION.md` and the disposition were read for context only; their claims are not the evidence.

---

## Observable Truths — Must-Haves by Plan

Verdict key: **PASS** = directly evidenced on the committed tree; **PASS\*** = pass with a documented
scope note (no behavioral gap). No truth is FAILED and none needed human-only judgment.

### 07-01 — udprepl decomposition and SentByType relocation (4 truths)

| # | Truth | Probe | Verdict |
|---|-------|-------|---------|
| 1.1 | `internal/udprepl` is split into socket/codec/loop/mirror units in the SAME package, adapter.go holds Adapter/New/doc | `head -3` on all five files: every one is `package udprepl` (`adapter.go` with the package doc); `find internal/udprepl -mindepth 1 -maxdepth 1 -type d` shows only the pre-existing `replproto`/`testdata`; `loop.go` owns `Run`/`flush`/`countSent`, `socket.go` `listenDatagram`/buffers, `codec.go` encode/decode, `mirror.go` atomics, `adapter.go` `Adapter`/`New` | PASS |
| 1.2 | No exported-name/behavior change and no test edits; the pre-existing udprepl suite compiles and passes unchanged | `git show --stat 8903b0f 3e560d3` — the two 07-01 commits touch zero `internal/udprepl/*_test.go`; `go test -race -count=1 ./internal/udprepl/` → ok (1.4 s). API diff base↔HEAD (`go doc -all` in a `git archive 31c9fe4` copy): no removal/rename; only the four additive 07-09 accessors and the mandated `SentByType` alias | PASS\* (the alias + additive accessors are later plans' explicit deliverables; no breaking change) |
| 1.3 | `SentByType` lives in `internal/app/repl`; udprepl keeps BOTH the alias and `Adapter.SentByType()`; metrics no longer imports udprepl | `internal/app/repl/sentbytype.go` defines `type SentByType struct`; `mirror.go:145` `type SentByType = repl.SentByType`; `mirror.go:149` `func (a *Adapter) SentByType() SentByType`; `grep -rn internal/udprepl internal/metrics/ internal/app/ internal/port/` finds no production import (only comments/docs) | PASS |
| 1.4 | The real-socket end-to-end smoke test still passes | Full `internal/udprepl` suite green, including `TestWire_twoNodeHandoff`, `smoke_test.go` and the fuzz/seam suites | PASS |

### 07-02 — One validation point (8 truths)

| # | Truth | Probe | Verdict |
|---|-------|-------|---------|
| 2.1 | Parse/validation/usage live in `internal/config`; `main.go` holds no flag registration, default table or validation branch | `main.go` imports no `flag` package; its body is Parse → help → version → Validate → setupLogging → logStartup → `os.Exit(run(...))`; all 22 flags are registered only in `internal/config/config.go:register`; `config.Parse/Validate/Usage` are the only entry points | PASS |
| 2.2 | One exported `Validate(c Config) error` walks one rule slice, joins every failure and names the parameter; every failure exits 2 through the usage path | `validate.go:71` `Validate` iterates `rules()` and `errors.Join`s; binary probes: missing required, peer≤hello, ack>max, retransmit 1 ns, sync-chunk 70000, drift==lead, positional args, value-max overflow, empty grpc — each exit 2 naming the parameter; `go test ./internal/config/` green (incl. `TestRules` pinning the 12-rule catalog) | PASS |
| 2.3 | The relations are enforced before any bind (peer_timeout>hello_interval, ack_timeout≤ack_timeout_max, max_clock_lead≥drift_warn+1 ms, retransmit≥1 ms, 0<sync_chunk_bytes≤65507, 0<sync_response_rate≤1e9, datagram ceiling) | Direct binary probes for every relation returned exit 2 with the offending flags named (e.g. `peer-vs-hello: --peer-timeout=1s must be greater than --hello-interval=2s`); `TestValidate` table green; `Validate` runs in `main()` before `run`/`wire` | PASS |
| 2.4 | Every numeric knob is positive: 13 durations + value-max-bytes + sync-chunk-bytes + sync-response-rate reject 0/negatives naming the parameter instead of normalizing | `positiveValues()` lists the 13 durations; explicit checks for the three ints; probes: `--startup-gate=0`, `--value-max-bytes=0`, `--sync-response-rate=0`, `--wait-timeout=-1s`, `--sync-chunk-bytes=0` all exit 2 with names; config table green | PASS |
| 2.5 | Positional args rejected naming every token; `--version` exits 0 on a broken env; `--help` exits 0 and lists the five env-only names | Probe: `straytoken other` → exit 2 `unexpected positional arguments: straytoken other`; `SUPKV_NODE_ID=nope --version` → `sup-kv dev dev go1.27.4`, exit 0; `--help` → exit 0 with `Environment-only (no flag)` block naming `SUPKV_LOG_LEVEL/LOG_FORMAT/METRICS_CERT/METRICS_KEY/METRICS_CLIENT_CA`; `TestUsage`/`TestMain_helpListsEnvOnly` green | PASS |
| 2.6 | `max-clock-lead` default is 2 m in binary, core, test model and `--help` | `--help` prints `(default 2m0s)`; `config.defaultMaxClockLead = 2*time.Minute`; `repl.defaultMaxClockLead = 2*time.Minute`; test models use `driftLead = 2*time.Minute`, `defaultMaxClockLead = 2*time.Minute`, `validateLead = 2*time.Minute`; `TestMain_maxClockLead` green | PASS |
| 2.7 | No touched artifact still calls 60 s the max-clock-lead default (core comments, discard model, version doc, help test); `drift_warn`'s own 60 s unchanged | Grep: core constant/comments say 2 m; `discard_test.go:21-26` says 2 m; `version/version.go:99` says 2 m; `main_test.go:469-475` says 2 m; `drift_warn` default remains 60 s in config and help. Only `retransmit_test.go:41 suspendWindow = 60s` remains and is an unrelated ladder-suspension window | PASS |
| 2.8 | `persist.EncodeTo` rejects non-positive/over-ceiling limits before the header with `storage.ErrValueTooLarge` and an empty buffer; doc.go states it | `codec.go:140-143` guard precedes `enc.Encode(header{...})`; test rows assert `errors.Is(err, storage.ErrValueTooLarge)` and `buf.Len()==0`; `doc.go` paragraph states the IN-04 guard; persist codec tests green | PASS |

### 07-03 — Machine-enforced dependency direction (4 truths)

| # | Truth | Probe | Verdict |
|---|-------|-------|---------|
| 3.1 | Three additive depguard rules exist after the frozen body: `sup-kv-adapters-not-in-core`, `sup-kv-metrics-no-adapters`, `sup-kv-grpcapi-no-repl-metrics` | `.golangci.yml:104-150`: all three rule keys present with per-adapter deny lists (app/port, metrics, grpcapi); no upstream line edited | PASS |
| 3.2 | The `grpcapi`→`persistfile`/`*persist.Engine` composition edge is written with its justification | `.golangci.yml:151-157` comment names the two snapshot sentinels and the `*persist.Engine` facade and states the exception is deliberate and visible; no deny entry blocks it | PASS |
| 3.3 | Every new rule fires on a forbidden non-test import | Temp-copy probes (`git archive HEAD` to /tmp, added one probe file per rule, `golangci-lint run --default=none --enable=depguard`): metrics→udprepl reported `sup-kv-metrics-no-adapters`; grpcapi→metrics reported `sup-kv-grpcapi-no-repl-metrics`; app/repl→grpcapi reported `sup-kv-adapters-not-in-core`. The repo tree was not modified | PASS |
| 3.4 | `.golangci.yml` lines 1-64 hash to `a121e046…`; `make lint` green | `sed -n '1,64p' .golangci.yml \| git hash-object --stdin` = `a121e0466bc211e48ca898f6c07434dc50eba07c`; `make lint` (`config verify` + `run ./...`) exit 0 | PASS |

### 07-04 — Exit codes and shutdown ordering (6 truths)

| # | Truth | Probe | Verdict |
|---|-------|-------|---------|
| 4.1 | Distinct, documented codes 0/1/2/3; final-dump failure wins over Serve error; startup failure never dumps | `main.go:62-66` constants; `run` returns 3 before 1 in the switch and returns 1 before `shutdown` on a wire error; direct binary probes: clean SIGTERM 0, fatal UDP bind 1, usage 2, unusable dump path final failure 3; `TestRun_*` green | PASS |
| 4.2 | Signals armed before wire (before dump load and UDP bind) | `run`: `sig := rt.installSignals()` precedes `rt.wire(cfg)`; `wire` calls `startEngine` (dump load) then `newReplicationAdapter` (bind); `TestRun_signalInstalledBeforeWire` green | PASS |
| 4.3 | Serve error logged ERROR and still runs the full shutdown incl. the bounded final dump | `run` logs `Serve failed` then always calls `rt.shutdown`; `shutdown` ends in `engine.FlushTimeout(shutdownGrace)`; `TestRun_serveErrorStillShutsDown` green (and `TestRun_serveAndDumpFailure` proves 3 precedence) | PASS |
| 4.4 | Fatal startup error is one ERROR naming stage and detail; no lifecycle `log.Fatalf` | Direct probe at `SUPKV_LOG_LEVEL=error`: exactly one `ERROR "Startup failed"` with `stage=replication_bind` + detail, exit 1; `grep Fatalf` in lifecycle code finds none; `TestMain_fatalBindLoggedAtError` green | PASS |
| 4.5 | `run(cfg, rt) int` injectable seam; production runtime wires the real functions; `main` = parse → help/version → setupLogging → `os.Exit(run(...))` | `main.go:156-163,216-252`; `run_internal_test.go` drives fakes and asserts call order/codes; `main()` body follows the stated sequence | PASS |
| 4.6 | Process tests assert each code: clean SIGTERM 0, startup bind 1, usage 2, dump failure 3 even as root | `TestMain_lifecycle` (0), `TestMain_fatalBindLoggedAtError` (1), usage rows (2), `TestMain_finalDumpFailureExitsThree` (3, ENOTDIR-based so root cannot bypass); all green in this session; direct binary probes agree | PASS |

### 07-05 — Received-lead drift (7 truths)

| # | Truth | Probe | Verdict |
|---|-------|-------|---------|
| 5.1 | Drift measures the RECEIVED version's lead; `drift_warn+1 ms` ahead positive, within `drift_warn` (either direction) zero | `clock.go:124-131` `Update` computes `driftBetween(remoteL, ptMs, driftWarn)`; `TestClock_driftSignal` boundary table green (exact warn → 0; +1 ms → positive; behind → 0) | PASS |
| 5.2 | Core counts/warns only positive lead; negative never counted; `driftBetween` saturation-safe | `core.go:1637` and `:1806` guard `drift > 0`; `driftBetween` clamps operands and the signed result to ±`driftMsMax`; `TestClock_driftSignalSaturates` and the behind-the-wall merge row green | PASS |
| 5.3 | `Clock.Tick` keeps the local-vs-wall diagnostic and is not the OBSV-05 source | `Tick` exists and is used by `storage.Set` for local writes; the core's OBSV-05 path calls `ApplyRemote`/`Observe` (Update), not Tick | PASS |
| 5.4 | `applyRemote` counts every received record's positive drift; `handleDataSync` counts one per part maximum via `Observe` | `core.go:1626,1637-1643` and `:1805-1808`; `TestMergeStats_counts` (first far-ahead apply = 1, redelivery unchanged), `TestMergeStats_listingAndStaleExcluded`, `TestDriftEvent` (listing part max = one event/count) green | PASS |
| 5.5 | Shipped defaults (drift_warn 60 s, max_clock_lead 2 m) make a +61 s record accepted and counted | Test model uses `testDriftWarn = time.Minute` and `driftLead = 2*time.Minute` — exactly the shipped 60 s/2 m; far-ahead rows count 1 and are accepted; startup relation requires ≥ +1 ms | PASS |
| 5.6 | Every pre-existing test whose expected drift changes was updated | `internal/app/clock/clock_test.go`, `hlc_prop_test.go`, `storage/store_test.go`, `repl/merge_stats_test.go`, `discard_test.go` are in the phase diff; full clock/storage/repl-selected suites green | PASS |
| 5.7 | Test-model artifacts describe 2 m: version `maxLead`+edge window, generator future window, retransmit seam comment | `version/validate_test.go:16 validateLead = 2*time.Minute`; `testutil/gen.go:80` says the generator straddles the 2 m inbound bound; `discard_test.go` uses 2 m; no stale 60 s max-lead wording remains | PASS |

### 07-06 — Honest conflict counting (6 truths)

| # | Truth | Probe | Verdict |
|---|-------|-------|---------|
| 6.1 | `supkv_merges_conflicts_total` counts only a pending (unacked) local write displaced; from `Core.pending`, never `len(storage.Conflict)` | `core.go:1659-1670` walks `c.pending` after a successful apply; the count does not read the store's Conflict slice; metrics Help says "Pending (unacked) local writes displaced…"; `TestMergeStats_counts` green | PASS |
| 6.2 | Outcome matrix: ordinary/confirmed overwrite 0, stale 0, identical redelivery 0, tie-break loss 0; strictly-greater pending displacement 1; equal-version tie-break win displacing pending 1 + Superseded | `merge_stats_test.go` rows for each case; suite green; code uses `cmp := msg.Record.Version.Compare(pk.version)` and `incomingWon` | PASS |
| 6.3 | Outbox contract unchanged: displaced pending emits Superseded with the tracked version; reply verdict (Ack or held-record answer) derived from the stored record | `core.go:1648-1677`: post-apply `Lookup` + `RecordsEqual` decides Ack vs held-record response; `out.Superseded(pk.key, pk.version)`; the conflict rows assert exactly one Superseded with the tracked version | PASS |
| 6.4 | Apply-failure WARN rate-limited per (node_id, apply_failure) and keyless; counters/gauge stay per datagram | `core.go:1627-1635` calls `warnRateLimited(..., warnClassApplyFailure, "…from handle %q…")` with no key; `TestApplyFailure_warnLimiterKeyedByNodeAndClass` and `TestDiscard_warnLimiterKeyedByNodeAndClass` green | PASS |
| 6.5 | The real-socket metrics-by-effect test builds the required conflict and asserts exactly 1 | `go test -race -count=1 ./internal/stand/ -run '^TestStand_metricsByEffect$'` → PASS; log `conflicts=1 bad_crc=1 retransmitted=1` | PASS |
| 6.6 | `storage.Conflict` doc no longer promises a conflict journal; points the counter at the pending set | `store.go:83-90` explicitly says it is not the counter's source and describes pending displacement (ADR 0004/OBSV-03) | PASS |

### 07-07 — Lifecycle catalog (6 truths)

| # | Truth | Probe | Verdict |
|---|-------|-------|---------|
| 7.1 | Startup INFO after setupLogging and before wire, with version/commit/go_version and every effective config value; no cert/key path or content | Direct node run: `INFO "Starting sup-kv"` with version, commit, go_version, node_id, bind/broadcast/dump/grpc/metrics addresses, metrics_tls, all timing knobs, value/sync knobs, log_level/format; no `metrics_cert/key/client_ca` keys; `TestMain_startupLogRecord` green | PASS |
| 7.2 | Exactly one load-outcome record per start; prompt/main INFO with outcome/generation/records/discarded; backup WARN and unusable ERROR carry the same fields, never a second line | `logLoadStats` appends `loadAttrs` to one record per outcome; direct prompt probe logged exactly one `Dump load outcome=prompt`; unusable probe logged one `No usable dump generation` with all four attrs; `TestMain_startupLogLoadOutcome` (prompt then main) green; catalog rows pin backup WARN/unusable ERROR | PASS |
| 7.3 | Gate-open INFO with reason "sync" or "backstop", callback once after CAS and health flip | `grpcapi/gate.go:208-214`: `CompareAndSwap` → `SetServingStatus` → `OnGateOpen`; both reasons have code paths; direct probe logged `Startup gate opened reason=backstop` once; `TestGate_openReason`/`opensOnSync`/`opensOnTimeout` green | PASS |
| 7.4 | Peer appear INFO, restart WARN, evict WARN (reason=peer_timeout, last_seen_age); repeat Hello / alternating run_id / own echo emit nothing | Real-binary asymmetric-pair probe: `INFO Peer appeared`, `WARN Peer restarted` (previous_run_id + run_id), `WARN Peer evicted reason=peer_timeout last_seen_age=1.0008s`; `TestPeerEvents` covers all three silences; `TestLog_jsonDefault` drives the two-node path | PASS |
| 7.5 | Drift WARN with node_id/handle/lead/drift_warn through the shared per-(node_id, clock_drift) limiter; counter per record; nil `OnDrift` falls back to `Log` | `events.go:77-87` limiter then typed event or Log fallback; `main.go:395-401` renders at WARN; `TestDriftEvent` proves two leading records → counter 2, one event, nil fallback one line; catalog row pins WARN | PASS |
| 7.6 | Structured attrs, capitalized static messages, snake_case keys, handles quoted values | Observed JSON in all probes (`"msg":"Peer appeared","handle":"127.0.0.1:7985"`, `"previous_run_id":…`); catalog test asserts static msgs/levels; no dynamic string is a msg | PASS |

### 07-08 — Unbound listing class and catalog levels (5 truths)

| # | Truth | Probe | Verdict |
|---|-------|-------|---------|
| 8.1 | Unbound listing has its own class/counter/WARN without "configuration error" and a zero-state metric sample | `core.go:68-76,1836-1848` defines `DiscardUnboundListing` and increments `UnboundListing`; `metrics.go:434` renders `{class="unbound_listing"}`; live scrape showed the sample at 0; `TestDiscard_unboundListingClass` asserts the WARN omits "configuration error"; `TestSync_unbound`/`forgedRun` green | PASS |
| 8.2 | `duplicate_node_id` remains for the two identity paths and the alternation path; only `listingBound` moved | `core.go:1111,1120,1452` still count DuplicateNodeID; `listingBound` is the only UnboundListing site | PASS |
| 8.3 | The second proto-version branch WARNs exactly like the first | `core.go:1089-1104`: adapter `ProtoVersionMismatch` and decoded `in.ProtoVersion != SupportedProtoVersion` both count BadProtoVersion and call `warnDiscard`; branch test `TestDiscardLog_classes`/`TestDiscard_validationClasses` green | PASS |
| 8.4 | Catalog level matrix: startup/load/gate/peer-appear INFO; restart/evict/drift WARN; dump failure/no-usable-dump ERROR; adapter WARN carries proto_version and duplicate details | `TestCatalogLevels` rows run each named renderer against a captured handler and assert one record at level+msg+detail; green in this session | PASS |
| 8.5 | Discard-conservation/totality sums the new class and every label list includes it | `discard_test.go` sum includes `UnboundListing`; `TestMetrics_values` asserts the rendered sample; docs anchor names the label; no stand label catalog exists to update | PASS |

### 07-09 — Honest metrics (5 truths)

| # | Truth | Probe | Verdict |
|---|-------|-------|---------|
| 9.1 | Sent counts only after the socket accepts; encode/resolve/WriteToUDP failures count nothing; fake packetConn fails then succeeds | `loop.go:504-556`: count only after `a.write` returns nil; `TestFlush_countsOnlySuccessfulWrites` (failed write 0, then success 1; resolve 0; encode 0 + flush continues; Ack 1) green | PASS |
| 9.2 | Four Phase-4 counters have a production surface and render zero when unseen | `metrics.go:206-209,375-405`; `mirror.go:108-136,251-254`; `main.metricsSources` supplies all four; live scrape showed all four at 0; `TestMetrics_families`/`TestMetrics_values`/`TestMirror_listingCounters` green | PASS |
| 9.3 | `process_resident_memory_bytes` is current RSS on Linux (`statm` field 2×page size); other builds report `ru_maxrss` and the Help calls it the development-only peak | `rss_linux.go` (`statmResidentField = 1`, `os.Getpagesize()`); `rss_other.go` uses `ru.Maxrss` with the darwin KiB note; Help says current/peak; live scrape 1.9e7; `TestParseStatm`/`TestCurrentRSS` green | PASS |
| 9.4 | Each definition is stated in Help and guarded by `TestMetrics_definitions` | Helps: conflicts "Pending (unacked)…", sent "successfully handed to the socket", duration "last dump attempt's encode and save", drift "lead local wall time", RSS "Current resident…"; `TestMetrics_definitions` asserts one phrase per definition; engine records duration on success and failure (`recordDump` called before the error branch) | PASS |
| 9.5 | The closed Sources set and every literal construction are updated; a scrape cannot panic on a nil accessor | All constructions (`main.go:725`, `metrics_test.go:224`; the `docs/reviews/.../repro` file is an archived standalone repro, not built) supply every field; `Snapshot` calls them unconditionally; `TestMetrics_families` zero-state green | PASS |

### 07-10 — Operator documentation (6 truths)

| # | Truth | Probe | Verdict |
|---|-------|-------|---------|
| 10.1 | OPERATIONS documents the lifecycle catalog with each entry's level; the anchor test pins one literal per entry | `docs/OPERATIONS.md` `## Lifecycle logs` lists every entry with level and fields (startup, load/fallback/unusable, gate, peer appear/restart/evict, drift, dump failed, adapter proto_version/duplicate/unbound/apply failure); `TestCatalogLevels` pins one literal+level per catalog entry; `TestOperationsDoc` anchors the section and representative messages | PASS\* (per-entry literal+level pinning is the catalog matrix; the operations anchor table pins the section plus the plan's named literals) |
| 10.2 | Exit-code section states 0/1/2/3, Serve still dumps, final dump bounded, signals armed before load/bind; no stale claim remains | `docs/OPERATIONS.md:385-421`; `operationsForbidden` holds `failed-final-dump exit code 1` and `dump open` out; `TestOperationsDoc` green | PASS |
| 10.3 | Drift runbook states exactly the shipped condition and no fictional direction | `docs/OPERATIONS.md:247-276`: received record or DataSync part maximum leading local wall time by > drift_warn, one rate-limited WARN per peer plus the metric; explicitly says a clock-ahead node does not warn itself; matches `clock.go`/`core.go` | PASS |
| 10.4 | Monitoring list carries the four counters, unbound_listing, the definitions and the 2 m default; `TestOperationsDocFamilies` proves every family exists | `docs/OPERATIONS.md:280-333`; `TestOperationsDocFamilies` green (every `supkv_*` name in the doc is declared in metrics.go) | PASS |
| 10.5 | SPEC-DEVIATIONS flips S-HLC-13, S-PAR-16, S-FAIL-17 to implemented and updates S-SEM-07/S-IMP-08; S-HLC-07 and v2 rows untouched | Rows inspected: the three are `implemented` with shipped-code links; S-SEM-07 names pending displacement; S-IMP-08 cites the conflict/drift definitions; `git diff` shows S-HLC-07 unchanged and no v2/unimplemented row flipped | PASS |
| 10.6 | AGENTS Layout names `internal/config`; deviations item 1 names the three additive rules and the written grpcapi exception | `AGENTS.md:20-31,49-57` inspected; matches `.golangci.yml` rule names and the exception | PASS |

### 07-11 — Phase gate (5 truths)

| # | Truth | Probe | Verdict |
|---|-------|-------|---------|
| 11.1 | Frozen red evidence byte-identical: `519789bd…`, `573fed68…`, `add45b81…` | `git hash-object` on all three → exact matches | PASS |
| 11.2 | Red-runner exits 0, 7 rows checked/0 errors, both mutants killed; sed target shapes present | Re-derived: all 7 named tests pass under `SUPKV_RED_TESTS=1 go test -race`; control seed `848612892191557850` passes; both overlay mutants (store.go `out[key] = rec.Version`; core.go periodic-sync gate) compile and FAIL `TestChaos_convergence` on that seed; both sed targets exist (`store.go:253/274`, `core.go:2510`); recorded runner evidence `20261123T134218Z` shows 7/7 OK + both mutants killed | PASS |
| 11.3 | `make lint`, `make test`, `make buf-lint`, `make buf-breaking`, `make client-example` green | `make lint` exit 0; `make buf-lint` exit 0; `buf breaking --against refs/remotes/origin/master` exit 0; client example build+vet+`go mod tidy -diff` clean; `make test`: every package green except `TestCore_unackedCostFlat` (ratio 2.67 > 2.0) — the documented Phase-8-owned wall-clock flake; focused rerun `-run '^TestCore_unackedCostFlat$'` PASS at ratio 1.29 under load 11.4 | PASS\* (per the phase's explicit load-flake carve-out; Phase 8 owns the gate) |
| 11.4 | Darwin cross-compile and vet pass; nested example covered; `.golangci.yml` 1-64 still `a121e046…` | `GOOS=darwin go build ./...` and `GOOS=darwin go vet ./...` both exit 0; example build/vet/tidy clean; hash matches | PASS |
| 11.5 | No runtime dependency added: direct runtime require set unchanged from phase start | `git diff 31c9fe4..HEAD -- go.mod go.sum` is empty; direct requires remain grpc + protobuf (pre-phase) | PASS |

**Score:** 62/62 must-haves verified; 0 human-only items; 0 failed.

---

## Success Criteria (ROADMAP.md:285-292)

| # | Criterion | Evidence that makes it true | Status |
|---|-----------|-----------------------------|--------|
| 1 | The node narrates its lifecycle at the documented levels; fatal bind at `SUPKV_LOG_LEVEL=error` is an ERROR; Spec entries exist; drift runbook matches the binary | Real-binary probes: startup record with version+effective config and no secrets; `Dump load outcome=prompt`; `Startup gate opened reason=backstop`; `Dump failed class=other` + `Final dump failed` on an unwritable path; `No usable dump generation` ERROR; `Peer appeared`/`Peer restarted`/`Peer evicted` from a live two-node pair; one `Startup failed stage=replication_bind` ERROR and exit 1 at error level; two-node `DuplicateNodeID` WARN process test; `TestCatalogLevels` level matrix; `OPERATIONS.md` drift runbook + anchors | PASS |
| 2 | Configuration validated in one place: one function, ranges/relations, exit 2 naming the parameter, table-tested, flags out of main, `--help` lists five env-only vars | `internal/config` is the sole parse/validate/usage point; 12-rule table with every range/relation; direct probes of each rule exit 2 naming the flag; `TestValidate`/`TestRules`/`TestUsage` green; `main.go` has no flag package; `--help` lists the five env-only names | PASS |
| 3 | Metrics mean what they say and every definition has a test | conflicts = pending displacement (code + `TestMergeStats_counts` + stand exact-1); sent = socket-accepted (`TestFlush_countsOnlySuccessfulWrites`); RSS = current (`rss_linux.go` + `TestCurrentRSS` + live 19 MB); dump duration = last attempt (`recordDump` before error branch + Help + engine test); drift = received lead (`TestClock_driftSignal`, `TestMergeStats_counts`); `TestMetrics_definitions` guards every Help phrase | PASS |
| 4 | Exit codes 0/1/2/3 distinct and ordered; Serve error still dumps; final dump bounded; signals before load/bind; `--help` lists five vars | `run`/`shutdown` code; `TestRun_*` and process tests; direct binary probes of 0/1/2/3; `signalAndWaitExit` clean 0; second-signal force test green; `--help` probe | PASS |
| 5 | Boundaries machine-enforced; units decomposed; main is wiring only | Three depguard rules proven to fire with their names on forbidden imports; `make lint` green; udprepl split into socket/codec/loop/mirror/adapter in one package with unchanged tests; `main()` is parse→help/version→logging→run | PASS |

---

## Requirements Coverage

| Requirement | Delivering code | Test / probe | Status |
|-------------|-----------------|--------------|--------|
| OBSV-01 lifecycle logs | `cmd/sup-kv/main.go` renderers + `internal/app/repl/events.go` + `internal/grpcapi/gate.go` | Real-binary startup/load/gate/peer/dump-failure probes; `TestCatalogLevels`, `TestMain_startupLogRecord`, `TestMain_startupLogLoadOutcome`, `TestMain_gateOpenLogged`, `TestPeerEvents` | VERIFIED |
| OBSV-02 one validation point | `internal/config/{config,validate,usage}.go`; wiring-only `main.go` | 12-rule table + direct exit-2 probes for every range/relation/positional/`--version`/`--help`; `TestValidate`, `TestRules`, `TestUsage` | VERIFIED |
| OBSV-03 honest metrics | `internal/app/repl/core.go` (pending conflicts, drift), `internal/udprepl/loop.go` (sent), `internal/metrics/*` (families/RSS), `internal/app/persist/scheduler.go` (duration) | `TestMergeStats_counts`, `TestFlush_countsOnlySuccessfulWrites`, `TestCurrentRSS`, `TestMetrics_definitions`, `TestStand_metricsByEffect`; live scrape | VERIFIED |
| OBSV-04 exit codes/shutdown | `cmd/sup-kv/main.go` (`run`, `shutdown`, `armSignals`) | Direct probes 0/1/2/3; `TestRun_*`, `TestMain_finalDumpFailureExitsThree`, `TestShutdown_secondSignal` | VERIFIED |
| OBSV-05 drift warnings | `internal/app/clock/clock.go`, `internal/app/repl/core.go`, `internal/app/repl/events.go`, `cmd/sup-kv/main.go` | `TestClock_driftSignal`, `TestDriftEvent`, `TestMergeStats_counts`; docs runbook inspected against code | VERIFIED |
| OBSV-06 Spec-mandated entries | `core.go` discard branches, `main.go` load/dump renderers, `logAdapterWarn` | `TestCatalogLevels`; direct unusable-dump/fatal-bind probes; two-node duplicate-node-id process test; `TestDiscardLog_classes` for proto_version | VERIFIED |
| OBSV-07 machine-enforced boundary | `.golangci.yml` three additive rules + exception comment | Temp-copy probes: each rule fired by name; `make lint` green; lines 1-64 hash match | VERIFIED |
| OBSV-08 decomposition | `internal/udprepl/{adapter,socket,codec,loop,mirror}.go`; `internal/app/repl/sentbytype.go`; wiring-only `main.go` | Same-package split confirmed; alias+accessor; metrics no longer imports udprepl; udprepl suite green; exported API diff shows no breaking change | VERIFIED |

---

## Anti-Patterns and Prohibitions

- **Debt markers:** `grep -n -E "TBD|FIXME|XXX"` over the phase-touched production files found none.
- **No `log.Fatalf` on lifecycle paths:** verified (only tests use `t.Fatalf`).
- **No `//nolint` without justification:** `make lint` with `nolintlint.require-explanation` is green; the phase's new suppressions (e.g. `gosec` G706/G304) each carry a reason.
- **No second validation path in `main.go`:** verified.
- **No stale conflict-journal or bidirectional-drift claim:** verified in code comments and `OPERATIONS.md`.
- **Frozen red tests untouched:** hashes match; `scripts/red-expectations.tsv` untouched.
- **No boundary change without config diff:** the only `.golangci.yml` change is the additive rule block (58 lines, hash 1-64 frozen).

## Probe Log (commands run in this session)

- `go build -o /tmp/supkv-verify ./cmd/sup-kv`; `--help`, `--version`, and ten misconfiguration probes.
- Live node runs: normal lifecycle + metrics scrape; fatal bind at `SUPKV_LOG_LEVEL=error`; unwritable dump path (exit 3, `Dump failed` + `Final dump failed`); two-node peer appear/restart/evict.
- `go test -race -count=1` green for `./internal/config/`, `./internal/metrics/`, `./internal/udprepl/`, `./internal/app/clock/`, `./internal/grpcapi/`, `./internal/app/persist/ -run 'TestCodec|TestEncode'`, `./internal/stand/ -run '^TestStand_metricsByEffect$'`, `./internal/app/repl/` phase-relevant selection, `./cmd/sup-kv/` phase-relevant selection (all green).
- Red gate re-derived: 7/7 expectation tests pass under `SUPKV_RED_TESTS=1`; control seed passes; both mutation overlays fail `TestChaos_convergence` on the fixed seed.
- Depguard temp-copy probes: each of the three rules fired by name.
- `make lint` exit 0; `go tool buf lint` exit 0; `go tool buf breaking --against '.git#ref=refs/remotes/origin/master'` exit 0; `GOOS=darwin go build/vet ./...` exit 0; client example build/vet/`go mod tidy -diff` clean.
- Full `make test`: all packages green except the Phase-8-owned `TestCore_unackedCostFlat` (ratio 2.67 under load 12.8); focused rerun PASS (ratio 1.29, load 11.4).

## Not Verified / Out of Scope

- **Two wall-clock ratio gates** (`TestCore_keyCountCostFlat`, `TestCore_unackedCostFlat`) and the virtual-time `TestScenario_dumpNotify` harness flake: load-sensitive, recorded in `deferred-items.md`, owned by **Phase 8**. The ratio gate is explicitly excluded from this phase's must-haves per the verification brief; the focused rerun above was green.
- **Registry citation resweep (New-3 and the 52-doc-mismatch audit):** in-range symbol drift left by this phase's code growth, recorded in `deferred-items.md` and owned by **Phase 9** (its full registry audit).
- **Darwin RSS runtime behavior:** the `ru_maxrss` peak fallback is code-inspected and cross-compiled, not executed (development-only platform per ADR 0010; Linux is the production target).
- **A full in-repo `bash scripts/red-runner` invocation:** its components were re-derived directly (7/7 rows, both mutants, sed targets) without writing a new evidence directory into the repository; the recorded runner evidence at `20261123T134218Z` (7/7 OK, both mutants killed) is consistent.
- **Test-harness port race:** `TestLog_jsonDefault`/`TestLog_formatText` failed once under the fully parallel package run with `bind: address already in use` (freeUDPPort race); each passed individually. This is a test-harness allocation race, not a phase behavior; Phase 8 owns flake work.

---

_Verified: 2026-11-23T14:31:31Z_
_Verifier: Claude (gsd-verifier)_
