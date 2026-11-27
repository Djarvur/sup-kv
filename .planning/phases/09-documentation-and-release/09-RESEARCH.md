# Phase 9 Research: Documentation and Release

**Researched:** 2026-11-27
**Grounding:** every claim below was verified against the working tree at `4cf390a`
(branch `gsd/v1.1-milestone`), the live GitHub API (ruleset queries), and the actual
test/config/goreleaser sources. Locked decisions D-01…D-10 are taken as fixed; this
document grounds them in the current repo state and resolves the planner's open
questions with evidence.

---

## 1. The registry: 22 `fixed-in-v1.1(...)` rows (D-01's flip list)

`docs/SPEC-DEVIATIONS.md` (261 lines) carries **167 table rows** — the review's full
section-13 trace — with this status distribution (grep-verified):

| Status | Rows |
|---|---|
| `implemented` | 126 |
| `fixed-in-v1.1(<REQ>)` | **22** |
| `superseded-by-amendment` | 12 |
| `consumer-zone` | 4 |
| `accepted-deviation(...)` | 2 (S-HLC-07, S-IMP-08) |
| `deferred-v2(SNAP-01)` | 1 |

The 22 open rows, with the REQ that closed them in Phases 2–8 (all fixes landed; the
flips were deferred to this phase per 08-CONTEXT / 07 deferred-items New-3 routing):

| Row | REQ | Fix evidence to cite at flip time |
|---|---|---|
| S-PROTO-02 | XPORT-03 | key length + value/datagram ceiling enforced (`maxdatagram.go`, `checkDatagramCeiling`) |
| S-PROTO-03 | XPORT-03 | same ceiling work |
| S-PROTO-04 | OBSV-06 | per-class discard counters + rate-limited WARNs (Phase 7) |
| S-PROTO-10 | PROTO-04 | first-seen Hello → immediate targeted listing (Phase 4) |
| S-PROTO-14 | PROTO-08 | dump-failed writes and snapshot loads broadcast (Phase 4) |
| S-PROTO-15 | PROTO-03 | stale DataReplication → one unicast reply, no Ack (Phase 4) |
| S-PROTO-16 | PROTO-03 | Ack only for applied/equal versions (Phase 4) |
| S-ALG-CU-08 | OBSV-06 | `logDumpFailure` ERROR + class counters (Phase 7) |
| S-ALG-CU-09 | PROTO-08 | dump-failed write stays tracked + replicates (Phase 4) |
| S-ALG-CU-11 | PROTO-03 | displaced version → Superseded, honest wait report (Phase 6) |
| S-ALG-REP-04 | PROTO-03 | the unicast-reply branch exists (`applyRemote`, Phase 4) |
| S-ALG-RC-09 | XPORT-07 | bounded, deduplicated sync-answer queue (Phase 3) |
| S-ALG-RC-10 | XPORT-02 | paced listing/outbox, cost-flat (Phase 3) |
| S-SCN-03 | PROTO-03 | same as S-PROTO-15 |
| S-SCN-04 | PROTO-04 | same as S-PROTO-10 |
| S-PERS-13 | PROTO-08 | SnapshotLoad notify loop (Phase 6) |
| S-SEM-08 | PROTO-08 | visibility bounds now hold incl. dump-failed/snapshot paths |
| S-FAIL-03 | PROTO-03 | same as S-PROTO-15 |
| S-FAIL-07 | PROTO-04 | same as S-PROTO-10 |
| S-FAIL-09 | XPORT-01 | paced empty-node recovery, 20/20 nightly job |
| S-FAIL-14 | OBSV-05 | received-lead drift counter + WARN (Phase 7) |
| S-FAIL-15 | PERF-02 | burst bound: `TestRed_R02_burst`, `TestPerfBurstDistinctKeys`, `TestEngine_burstBoundedConfirmation` |

**REQ distribution:** PROTO-03 ×6, PROTO-08 ×4, PROTO-04 ×3, XPORT-03 ×2, OBSV-06 ×2,
XPORT-01/02/07, OBSV-05, PERF-02 ×1 each.

**Two coupled sweeps ride the same audit pass (D-01):**

1. **Stale line citations (Phase-7 re-verification New-3).** The
   `07-observability-and-configuration/deferred-items.md` row names the in-range
   citations that resolve to the wrong symbols after core growth: S-HLC-04/S-HLC-10
   (`core.go:1510-1543` — actual `handleDataSync` ~1794; `store.go:295-299` — actual
   `Observe` ~377), S-DM-04/S-DM-08 (`store.go:252-273/:306-328/:376-415/:178-186`),
   S-ALG-REP-02/03 (`core.go:1489-1539/:1487-1530` — actual `applyRemote` ~1608),
   S-ALG-CU-07 (`grpcapi/server.go:433-454` — actual `mapError` ~493),
   S-ALG-CU-08 (`metrics.go:405-445`), S-DM-05/S-PAR-05 (`store.go:148-152`). The
   resweep must re-derive **every** Links cell line-by-line (the 06-10 precedent: a
   pure path swap cited neighbouring declarations).
2. **Gate shape.** The preamble's own words: "Phase 9 greps it for unclosed rows".
   Beware: the vocabulary preamble line itself contains the token
   `fixed-in-v1.1(<REQ-ID>)` — the gate must match the **table-status form**
   (`| fixed-in-v1.1(` at a cell) or exclude the preamble, or it can never pass.
   Same for `deferred-v2(` (S-IMP-04 is a legitimate permanent row) — the exit
   criterion is zero `fixed-in-v1.1(` rows, not zero non-`implemented` rows. The
   gate is a test in the doc-anchor package (machine-checked, per D-01).

---

## 2. M-01…M-52 closure starting state (D-02)

No consolidated ledger exists; Phases 3/5/6/7 recorded M mentions in their plan
files but there is no per-row disposition table. The plan's sweep must re-derive
closure per row. Verified current state (grep/read, 2026-11-27) — a head start, not
a substitute for the sweep:

### Verified still OPEN (fix rides this phase)

- **M-01** — `README.md:22-23` still says release artifacts "in preparation";
  no install-from-release section. Tag `v1.0.0` exists (only tag).
- **M-02** — `README.md:92-93` "The block cleans up after itself"; the marked block
  has no `rm`, leaves `supkv-a.json`/`supkv-b.json` in cwd; second-run false positive.
- **M-03** — `README.md:99` ipconfig/en0 derive line; `:132-134` still claims it
  "derives the segment broadcast address from the primary interface instead".
- **M-04** — archives contain only CHANGELOG/LICENSE/README/binary
  (`.goreleaser.yaml` has no `files:` entries); README links `docs/*` dead in archive.
- **M-05** — "proven" without a test name: `README.md:30`,
  `docs/OPERATIONS.md:40,257`; CHANGELOG 1.0.0 body too.
- **M-06** — `OPERATIONS.md:44-45` still the unconditional
  "bounds recovery at ack_timeout_max (default 5s)" claim.
- **M-16** — no systemd unit file anywhere in the repo (confirmed: no `*.service`).
- **M-17** — `EXAMPLES.md:61-62` wait loop uses unquoted `grep -q SERVING`
  (matches `NOT_SERVING`); assertion table `{title: "Start the demo pair",
  expect: []string{"SERVING"}}` is also NOT_SERVING-satisfiable (substring).
  README uses the correct quoted form.
- **M-18** — `EXAMPLES.md:93` "The listing names the three services" followed by a
  four-entry listing (Health, reflection v1, reflection v1alpha, SupKVService).
- **M-19** — `create_if_absent`, `AlreadyExists`, `Unavailable`,
  `ResourceExhausted` appear **zero** times in docs/EXAMPLES.md (grep-verified).
- **M-21** — docs/EXAMPLES.md has no platform note at all (zero linux/macos/darwin
  mentions).
- **M-23** — `docs/IDEA.md:145,:169` still promise a conflict journal (IDEA.md is
  under the HISTORICAL banner — plan must decide banner-covers vs scrub);
  `.claude/CLAUDE.md:21` also lists "conflict journal" as an inspectable artifact.
- **M-24** — `docs/IDEA.md:110` still describes the dump as one JSON document
  (banner may cover; FORMATS.md will be the as-built truth).
- **M-32…M-37** — AGENTS.md stale: "go 1.27" vs go.mod `go 1.27.4`; lint/test/CI
  description differs from Makefile/ci.yml; `internal/app` named as "three packages"
  (actually five: clock, persist, repl, storage, version, plus simnet/testutil);
  adapters "appear with the phases" (four exist: grpcapi, udprepl, persistfile,
  metrics — plus config and stand); "Phase 1 carries zero direct runtime requires"
  (grpc+protobuf are direct now); "two tool directives" (seven exist).
- **M-38…M-44** — `.claude/CLAUDE.md` stack section still pre-implementation
  research: `pgregory.net/rapid` v1.3.0 (not in module); simulator "goroutines
  inside a synctest bubble" (it is a discrete-event scheduler); `json.MarshalIndent`
  (code uses one-line Encoder); `ReadBuildInfo` (ldflags); `x/sync v0.23.0` (actual
  v0.25.0 indirect), `version: v2.15` (actual v2.15.0), protoc-gen-go plugin/runtime
  claims; "Conventions not yet established" / "Architecture not yet mapped".
  (M-40's SetReadBuffer advice became accurate when Phase 3 shipped XPORT-05 —
  verify, don't assume.) Note: the stack section is delimited by
  `<!-- GSD:stack-start source:research/STACK.md -->` … `stack-end` markers.
- **M-48** — `internal/app/storage/store.go:100` and `internal/app/storage/doc.go:6`
  still say "internal/port holds no interfaces" while `port/dump.go`/`port/repl.go`
  define them.
- **M-49** — Makefile `buf-breaking` and ci.yml buf-gate comments still describe the
  dead guard ("self-destructs the moment the proto lands there"); the proto IS on
  origin/master since Phase 6 (07-11 ran the real comparison).
- **M-50** — `.github/dependabot.yml` comment still says "GHAS-gated on this private
  repo… stay off until the deferred public flip"; repo flipped public 2026-11-18.
- **M-51** — **executed during this research:** the doc's verify command
  (`gh api repos/Djarvur/sup-kv/rulesets --jq '.[] | select(.name == "protect
  master") | …'`) fails with `cannot iterate over: null` — the list endpoint's
  objects have keys `_links, created_at, enforcement, id, name, node_id, source,
  source_type, target, updated_at` (no `rules`). The by-id form
  `gh api repos/Djarvur/sup-kv/rulesets/27576625 --jq '{name, enforcement,
  rules: [.rules[].type]}'` **works today and prints exactly the documented shape**
  (executed: `{"enforcement":"active","name":"protect master","rules":["pull_request",
  "required_status_checks","non_fast_forward","deletion"]}`). The fix is known-good;
  the plan executes it once and records the result.
- **M-52** — no `help` target (`.PHONY` list has none; `## target:` comments exist).

### Verified CLOSED by Phases 3–8 (sweep confirms, flips disposition)

- M-08 (single value limit + exit 2; `## Record and datagram limits` documents it),
  M-09 (conflict counter = pending-local displacement; HELP text matches),
  M-10 (drift = received leads; runbook rewritten), M-11 (current-RSS documented),
  M-12 (exit codes 0/1/2/3; startup errors at ERROR), M-13 (lifecycle catalog
  pinned by `TestCatalogLevels`), M-14 (load outcomes: Backup WARN / Unusable
  ERROR / "Dump failed" ERROR), M-22 (zero-peer `(false, 0)` documented; EXAMPLES
  asserts `ackedCount`/`totalPeers`), M-45/M-46 (proto phase references gone —
  grep clean in both protos for "phase"/"plan 0"), M-47 (caller-bounded wait).

### Banner-route candidates (DOC-12's "historical banner OR replace" was chosen)

M-24…M-31 (IDEA.md inaccuracies) — `docs/IDEA.md` opens with the loud HISTORICAL
banner that names the as-built trio and says "The as-built architecture description
and docs/ARCHITECTURE.md land by Phase 9 of milestone v1.1." The sweep's per-row
disposition must state whether the banner covers each (M-25…M-28 plausibly yes) or
whether the row needs a fix (M-29/M-30/M-31 partially superseded by API.md work —
but API.md's own "## What Phase 9 adds" section names the key-prefix convention and
write-latency model as **pending Phase 9** deliverables).

---

## 3. Document inventory (what exists / what is new)

Existing (docs/): `04-MULTI-NODE.md`, `API.md` (325 lines; ends with "## What Phase 9
adds" naming the pending per-RPC failure table, write-latency model, key-prefix
convention), `EXAMPLES.md`, `IDEA.md` (bannered historical), `OPERATIONS.md`
(425 lines), `PERFORMANCE.md` (270 lines; Phase-8 hardware-block ledger),
`SPEC-DEVIATIONS.md`, `adr/0001…0012` (12 ADRs, **no index**), `setup/branch-protection.md`
+ `ruleset.json`, `reviews/2026-11-19-v1.0/`. Root: `README.md`, `CHANGELOG.md`,
`AGENTS.md`; `.claude/CLAUDE.md`.

**New files this phase (fixed targets per REQUIREMENTS/D-05):**
`docs/CONFIGURATION.md`, `docs/DEPLOYMENT.md` (+ `sup-kv.service` — path planner's
choice; `deploy/sup-kv.service` or `docs/` both defensible), `docs/FORMATS.md`,
`CONTRIBUTING.md`, `SECURITY.md`, `docs/ARCHITECTURE.md` (the as-built description;
IDEA.md's banner already promises it), an ADR index, a simulator guide (new file or
ARCHITECTURE section), a release-procedure doc (new file or CONTRIBUTING section).

OPERATIONS.md section map today (headings in order): Trust model → What you are
operating → Scale envelope → Record and datagram limits → Configuration validation →
Load outcomes → Memory model → Read semantics → Not a lock service → Clock drift
runbook → Monitoring → Lifecycle logs → Lifecycle and exit codes. D-03 extends it
with: the 18-row failure-behavior table, seven new runbooks, the metrics reference
(the Monitoring prose becomes the typed/labels/alert table), and the full
log-message catalog (Lifecycle logs already carries ~13 entries; main.go has ~22
slog call sites).

### The 18 failure-table rows

S-FAIL-01…S-FAIL-18 in the registry are exactly the Spec's 18 failure rows — the
registry summaries are the symptom/behavior glosses; the table adds recovery, bound,
metric and log line per row. Seven runbooks: dump failure/full disk, missing peer,
growing unconfirmed set, write conflict, incompatible versions, corrupted dump,
duplicate `node_id` (clock drift already exists as the eighth).

### Log-message surface (main.go, grep-verified)

INFO: `Starting sup-kv`, `Dump load`, `Startup gate opened`, `Peer appeared`,
`Socket buffers`, `Orphan temp sweep removed files`. WARN: `Bind address is not a
wildcard`, `Peer restarted`, `Peer evicted`, `Clock drift`, `Replication adapter`,
`Dumped records skipped by the configured value limit`, `Dump generation fallback`
(×2 sites), `Orphan temp sweep failed`, `Engine did not stop within the shutdown
grace`. ERROR: `Startup failed`, `Serve failed`, `Final dump failed`, `Dump failed`,
`No usable dump generation`, `Metrics endpoint`. DEBUG: `Bind address is not
host:port`. Plus the usage path's plain-text errors and `logDiscardedRecords`' INFO.
The catalog cross-check can grep these literals in `cmd/sup-kv/main.go` (the
`TestOperationsDocFamilies` precedent) or render them via the named renderers
(`TestCatalogLevels` precedent).

---

## 4. Configuration surface (CONFIGURATION.md input)

22 registered flags (`internal/config/config.go` `register`/`registerDurationFlags`):
`--version` (special), `--node-id`, `--bind-addr`, `--broadcast-addr`, `--dump-path`,
`--grpc-addr` (default 127.0.0.1:7901), `--metrics-addr` (no default = no listener),
`--startup-gate` (1s), `--startup-gate-empty` (60s), `--wait-timeout` (1s),
`--value-max-bytes` (61440 = `storage.ValueMaxBytes`), `--sync-chunk-bytes` (1400),
`--sync-response-rate` (500), `--hello-interval` (1s), `--peer-timeout` (5s),
`--ack-timeout` (200ms), `--ack-timeout-max` (5s), `--retransmit-interval` (100ms),
`--sync-interval` (30s), `--max-clock-lead` (2m), `--dump-debounce` (50ms),
`--dump-max-wait` (1s), `--drift-warn` (60s).

Five env-only knobs: `SUPKV_LOG_LEVEL`, `SUPKV_LOG_FORMAT`, `SUPKV_METRICS_CERT`,
`SUPKV_METRICS_KEY`, `SUPKV_METRICS_CLIENT_CA`.

Validation relations (`internal/config/validate.go`, one 12-rule catalog): required
knobs; node-id 16-bit; positivity rules (exit 2 naming the parameter);
`sync-chunk-bytes` ≤ datagram ceiling; `value-max-bytes` fits 65 507 with the key
limit; `peer_timeout > hello_interval`; `ack_timeout ≤ ack_timeout_max`;
`dump-debounce ≤ dump-max-wait`; `drift_warn < max_clock_lead`; `max_clock_lead`
normalization. "Must match on all nodes": value-max-bytes, sync-chunk-bytes,
max-clock-lead (documented in OPERATIONS today), node-id uniqueness.

Iterator grounding: `TestMain_helpListsEnvOnly` (cmd/sup-kv/main_test.go:539) already
asserts `--help` lists the five env-only variables — the binary's `--help` output is
the parseable registered-flag source for the table iterator (default values printed
per flag), keeping the check binary-anchored end to end.

---

## 5. Metrics surface (metrics reference + cross-check input)

`internal/metrics/metrics.go` — `Snapshot(Sources)` renders the complete family set
at zero state ("complete-catalog rule": every labeled class/outcome renders even at
zero). Families (24 supkv_ + 5 runtime):

- Unlabeled counters/gauges: `supkv_replication_peers`, `..._unacked_records`,
  `..._unacked_oldest_age_seconds`, `..._datagrams_retransmitted_total`,
  `..._datagrams_paced_total`, `..._send_queue`, `..._egress_dropped_total`,
  `..._read_errors_total`, `..._listings_dropped_total`,
  `..._sync_streams_dropped_total`, `..._listed_keys_evicted_total`,
  `..._notifications_regressed_total`, `supkv_merges_conflicts_total`,
  `supkv_clock_drift_warnings_total`, `supkv_dumps_duration_seconds` (gauge),
  `supkv_dumps_total`, `supkv_dumps_errors_total`, `supkv_dumps_unconfirmed_total`.
- Labeled: `supkv_replication_datagrams_sent_total{type: hello|data|ack|data_request|sync}`;
  `supkv_datagrams_discarded_total{class: bad_crc, undecodable, bad_proto_version,
  own_broadcast, duplicate_node_id, unbound_listing, empty_key, key_too_long,
  tombstone_with_data, bad_checksum, bad_version, bad_sync_header}` (12 classes);
  `supkv_dumps_failures_total{class: disk_full|permission|other}`;
  `supkv_dump_load_total{outcome: prompt|main|backup|unusable}`;
  `supkv_dump_records_discarded_total{class: checksum|version|value_limit|malformed}`.
- Runtime: `go_goroutines`, `go_memstats_alloc_bytes`, `go_memstats_sys_bytes`,
  `process_cpu_seconds_total`, `process_resident_memory_bytes` (current RSS on Linux).

The structured cross-check is straightforward: `metrics.Snapshot(metrics.Sources{})`
yields the whole name/type/label vocabulary without a live node — the DOCS-07 test
can iterate it against the reference table in both directions (exported-not-documented
fails; documented-not-exported fails — the existing `TestOperationsDocFamilies`
covers only doc→exporter by string grep; the reverse direction and label/type
checking are new).

---

## 6. Doc-anchor test infrastructure (D-04's pattern)

Five precedents live in `cmd/sup-kv/` (package `main_test`), all reading repo docs by
relative path (never fixtures):

| Test | File | Mechanism |
|---|---|---|
| `TestReadmeQuickstart` | readme_test.go | extracts `<!-- quickstart-verify -->` blocks, runs verbatim `bash -e` with built binary + grpcurl shim; option-b darwin loud-skip constant `quickstartBroadcastOption` |
| `TestExamplesDoc` | examples_test.go | 10 marked blocks in doc order, shared dir, title-keyed assertion table `examplesAssertions` |
| `TestOperationsDoc` / `TestOperationsDocFamilies` | operations_test.go | anchor literals + forbidden phrases (whitespace-normalized) + doc→exporter grep |
| `TestAPIDoc*` | api_docs_test.go | one literal per mandated API statement |
| `TestPerformanceDoc*` | performance_test.go | headings in order, thresholds formatted from `internal/testutil` constants (never hand-typed), per-section hardware block + command; also pins the S-FAIL-15 registry row |

Harness in main_test.go: build-once binary (`buildSmokeBinary`), grpcurl tool shim,
port bands ([27900, 28100) reserved for doc blocks; testutil allocator bands above),
`docBlockBudget` deadline clamp. New machine checks extend this package per D-04,
except the FORMATS encode test which belongs beside the codec it pins
(`internal/app/persist`, reading `../../docs/FORMATS.md`; `EncodeTo(w, records,
valueMaxBytes)` is deterministic over fixed `storage.Record` values — versions are
plain structs, so a fixed record set encodes byte-identically without clock
injection).

Exit codes for the ops-doc verification: `exitStartup=1`, `exitUsage=2`,
`exitDumpFailed=3` (main.go:63-65); precedence final-dump > Serve > OK.

---

## 7. Release mechanics (D-05/D-06/D-07/D-08)

- `.goreleaser.yaml`: `goos: [linux, darwin] × goarch: [amd64, arm64]` (4 archives);
  **no `files:` config** — archives carry binary + CHANGELOG/LICENSE/README defaults
  only. D-06: drop darwin (`goos: [linux]`) → 2 archives; D-05: add `files:` for
  DEPLOYMENT.md + the unit (DOC-05 acceptance: `tar tzf` of a snapshot archive lists
  docs/ and the unit). The release `header:` currently embeds the whole CHANGELOG.md
  including `## [Unreleased]` — its own comment says "from the second release on,
  slice the current-version section"; **v1.1.0 is that second release**, so the
  header template must be changed alongside the release procedure (DOC-24: release
  body must not contain "[Unreleased]" or phase suffixes).
- `.github/workflows/release.yml`: tag-triggered `go tool goreleaser release --clean`;
  workflow_dispatch runs `--snapshot`. Unchanged by the demotion.
- Makefile targets: `release`, `release-snapshot` (the mandatory pre-tag gate), plus
  `lint`, `test`, `bench`, `bench-smoke`, `perf`, `race-soak`, `red-run`, `generate`,
  `build`, `client-example`, `proto`, `buf-lint`, `buf-format`, `buf-breaking`. No
  `help` target (M-52). CI (`ci.yml`) does not call make for lint/test — it runs the
  tools directly; AGENTS.md's "CI runs the same targets" phrasing is part of M-33.
- `ci.yml` gate matrix: ubuntu-latest, ubuntu-24.04-arm, macos-15 (full native gate),
  darwin-amd64-cross (build+vet). D-06 keeps both darwin legs (ADR 0010: "building
  and unit tests on macOS remain supported") — only the goreleaser matrix changes.
  The five check-run contexts (`lint`, four `gate (…)` cells) are recorded in
  `docs/setup/ruleset.json` — **no CI matrix change is planned, so no ruleset change
  is needed**; the ruleset-coupling rule ("every gate-matrix change is a ruleset
  change") still constrains any accidental matrix edit.
- Tags: only `v1.0.0` exists; `v1.1.0` lands as the phase's final plan (D-08) after
  `make release-snapshot` verification. PR-only flow: tag is cut on merged master.
- Unit-in-CI options (planner discretion): `systemd-analyze verify <unit>` (offline
  syntax + ExecStart binary existence, no root) and/or running the unit's exact
  `ExecStart` arguments as a smoke node. DOC-05's acceptance wording is "the unit
  file is in .goreleaser.yaml archives and starts the binary in a CI smoke test" —
  the minimal honest shape is systemd-analyze verify + an ExecStart-args smoke run;
  a full `systemctl start` on the runner VM is possible (passwordless sudo) but
  brittle.

---

## 8. Q-09 dead code (D-09): verify-then-delete → everything already gone

Grep-verified against non-test code (2026-11-27):

- `lastLocal` — zero matches anywhere (test code included).
- `ownHandle` — zero matches anywhere.
- `persist.Parse` — no `func Parse` in internal/app/persist (Phase 3 removed it;
  `waiter.ver`/cover branch deleted same phase).
- `cover` — no identifier; only prose comments ("the cover discipline of await").
- `fsync_darwin.go` — file does not exist (only ADR 0010 and
  `stand_nondarwin_test.go` match "*darwin*").

XPORT-13's "gone or justified" acceptance is already met by deletion. D-09's plan
still records the verification (the four greps) as its evidence; the "deletion" task
becomes a no-op confirmation unless the sweep finds survivors.

---

## 9. Stack-audit grounding (D-10 / DOCS-08)

`go list ./...` (19 packages): `api/supkvapi/v1`, `cmd/sup-kv`, `internal/app/clock`,
`internal/app/internal/simnet`, `internal/app/internal/testutil`, `internal/app/persist`,
`internal/app/repl`, `internal/app/storage`, `internal/app/version`, `internal/config`,
`internal/grpcapi`, `internal/grpcapi/apiproto/supkvapi/v1`, `internal/metrics`,
`internal/persistfile`, `internal/port`, `internal/stand`, `internal/testutil`,
`internal/udprepl`, `internal/udprepl/replproto/supkv/v1`. Nested module
`examples/client` (module `example.com/sup-kv-client`, root-module invisible by
design). Direct requires: `google.golang.org/grpc v1.85.0`,
`google.golang.org/protobuf v1.36.13-0.20261016172727-…`. Tool directives (7): buf,
grpcurl, golangci-lint, goreleaser, govulncheck, protoc-gen-go-grpc, protoc-gen-go.
Go directive: `go 1.27.4`.

Machine-checkable audit shape: a test (or lint step) comparing the AGENTS.md/CLAUDE.md
package list against `go list ./...`, the Go directive against go.mod, and the tool
list against the `tool (…)` block — the audit DOC-17 acceptance names exactly this.
CLAUDE.md's stack section sits between `GSD:stack-start/stack-end` markers
(source: research/STACK.md) — the plan should decide whether to edit inside the
markers or replace the section and update the markers' source reference.

---

## 10. Proto comment scrub (D-07)

- `api/supkvapi/v1/supkvapi.proto` — grep for `phase|D-[0-9]|plan 0|decision 14`:
  **clean** (Phase 6's 06-10 scrub landed). The plan verifies and keeps it clean.
- `internal/udprepl/replproto/supkv/v1/supkv.proto` — two comment references remain:
  line 16 "decoded by the receiver only after the CRC gate passes (D-02)" and line 26
  "the adapter maps every field onto the port structs (D-01)". The audit's DOC-21
  acceptance reads "no match for `D-[0-9]`, `phase`, `plan 0` in **either** .proto" —
  so the scrub should cover these two lines (then `make proto` regenerates the
  committed code; CI's buf gates confirm nothing else moved). If the planner scopes
  DOCS-10's "proto comments" to the public proto only, the deviation must be stated
  in the plan — the stricter two-proto reading is the audit's own acceptance.

---

## 11. Validation Architecture (how each artifact proves itself)

The phase's property: **every document is either executed, cross-checked against the
binary, or anchored by literal — never prose-only.** The mechanism inventory:

1. **Registry gate (test):** zero table-cell `fixed-in-v1.1(` rows in
   SPEC-DEVIATIONS.md; additionally every flipped row's Links cell must resolve
   (file exists + symbol name present at/after the cited range) — the citation
   resweep's permanent guard. Runs in `cmd/sup-kv` doc-anchor package (D-04) so
   `make test`/CI carries it.
2. **CONFIGURATION.md iterator (test):** parse the built binary's `--help` output
   (name + default + env twin per flag — the registered surface, binary-anchored),
   require each flag and each of the five env-only knobs as a table row with the
   matching default; range/relation columns covered by the validate.go catalog as
   anchors. A new flag or changed default without a doc edit fails.
3. **Metrics reference cross-check (test):** `metrics.Snapshot(metrics.Sources{})`
   gives the complete family/type/label vocabulary; require every exported family
   and every label value documented (both directions), replacing/extending
   `TestOperationsDocFamilies`' one-directional grep. A metric rename or label
   change without a doc update breaks CI (DOCS-09).
4. **Log-message catalog cross-check (test):** extract quoted message literals from
   the catalog and grep them in `cmd/sup-kv/main.go` (and require the level matrix
   via the existing `TestCatalogLevels` renderers); the failure table's log-line and
   metric columns checked the same way — every row references something that exists.
5. **FORMATS.md byte-identical encode (test):** `persist.EncodeTo` over a fixed
   record set reproduces the doc's example dump byte-for-byte; the documented
   checksum recipe recomputes the example's checksums. Lives in
   `internal/app/persist` beside the codec (D-04).
6. **"Proven" claim checker (test):** scan README/OPERATIONS/CHANGELOG for
   proven/proves/proving occurrences; each must carry a named test (backticked
   `Test…` reference) in the same sentence/paragraph or a claimed-in-doc table.
   Mechanism (regex + adjacency vs whitelist) is planner discretion.
7. **Executable docs (existing harness):** README quickstart pair keeps running
   verbatim on Linux (the macOS pair's "works" = the two blocks stay green + the
   darwin leg loud-skips); EXAMPLES blocks gain the create_if_absent/AlreadyExists
   block and the corrected service count; DEPLOYMENT.md gains marked executable
   blocks where they prove topology claims (the two-board demo) — the marked-block
   contract is opt-in, so only blocks meant to be CI-enrolled get markers.
8. **Unit + archive checks:** `systemd-analyze verify` (+ ExecStart-args smoke) in
   CI; `make release-snapshot` then `tar tzf dist/*.tar.gz` lists DEPLOYMENT.md and
   the unit, and (post-demotion) contains no darwin archive; checksums + SBOMs
   appear. This is the release-procedure doc's executed claim.
9. **Exit codes / log levels:** existing process tests (run_internal_test.go,
   shutdown_internal_test.go, `TestCatalogLevels`) already exercise 1/2/3 and the
   level matrix; the ops doc's exit-code table anchors to them by name.
10. **Ruleset command (executed claim, not a test):** the fixed by-id command is run
    once, output recorded in branch-protection.md (this research already proved the
    fixed form works).

**Gate placement:** all new tests join `go test -race ./...` (the `make test` gate)
— no new CI job needed; the release steps stay in release.yml + the procedure doc.

---

## 12. Pitfalls

1. **The registry gate must exclude the preamble token** — `fixed-in-v1.1(<REQ-ID>)`
   appears in the status-vocabulary line; a naive count never reaches zero.
2. **`deferred-v2(SNAP-01)` is permanent** — the milestone criterion is zero
   *unclosed* rows (`fixed-in-v1.1(`), not zero non-`implemented` rows.
3. **Citation re-derivation, not path swaps** — Phase-6's 06-10 lesson: five SPEC
   links cited neighbouring declarations after a mechanical swap. Re-derive every
   Links cell line-by-line with symbol verification; 22 rows are being edited at
   once, the highest-blast-radius edit of the phase.
4. **The `--help` output is stderr**, exit 0 (audit Appendix A) — the iterator must
   capture stderr or use the flag set programmatically; and the usage text is
   rendered by `config.Usage` (usage.go) which resolves defaults from the process
   environment — the iterator should run with a clean env so env twins don't skew
   printed defaults (mirroring the `SUPKV_MAX_CLOCK_LEAD=90s --help` behavior).
5. **EXAMPLES block state carries across blocks** (shared directory, doc order) —
   inserting the create_if_absent block changes later blocks' state (a
   create_if_absent after an existing write returns AlreadyExists by design);
   place and account for ordering deliberately.
6. **The examples assertion table is bidirectional** — an added block without a
   table row fails ("marked block the table does not know"); every new block needs
   its entry.
7. **`grep -q SERVING` matches NOT_SERVING** — M-17's fix is the quoted
   `"SERVING"` form; the assertion-table entry must be quoted too
   (`"status": "SERVING"`), not just the doc text.
8. **goreleaser header slicing** — the v2 schema's `header:` is a template string;
   slicing "the current-version section" of CHANGELOG.md inside the template is
   non-trivial (mustReadFile + string ops); verify the template against the pinned
   v2.20.2 func map before committing to a shape, or ship the whole file and accept
   DOC-24 partially. The safer route: keep the release body produced by the tag
   process aligned by editing CHANGELOG so `[Unreleased]` is empty and the 1.1.0
   section sits first — but the comment's own promise ("slice") should be honored
   or explicitly discharged.
9. **Proto comment scrub regenerates code** — editing replproto comments requires
   `make proto` (committed pb.go changes) and passes through buf lint/format/
   breaking; keep comment-only diffs clean.
10. **CHANGELOG scrub is over existing prose** — the 1.0.0 entry carries
    `(Phase 2)`…`(Phase 6)` suffixes on seven bullets; the scrub edits published
    text of a released version — acceptable here (Keep-a-Changelog hygiene, D-07
    explicitly mandates it) but note the GitHub Release body for v1.0.0 is
    historical and stays as published.
11. **IDEA.md banner vs per-row closure** — the M-sweep dispositions must be honest
    about what "closed" means for bannered-historical rows: the audit's DOC-12
    acceptance ("README's description of the file matches its banner") is the bar,
    not rewriting IDEA.md content.
12. **AGENTS.md edits have lint-visible consequences** — AGENTS.md documents the
    depguard rules and the vendored-config provenance; keep deviations 1–3 and the
    origin header intact while refreshing the stale layout/policy prose (the
    config-provenance rule forbids editing upstream lines, but AGENTS.md prose is
    free to update).
13. **Doc tests read repo files by relative path** — new tests in cmd/sup-kv must
    keep the `../../docs/...` path idiom; the FORMATS test in internal/app/persist
    uses `../../../docs/FORMATS.md` (three levels up).
14. **Release ordering** — the tag is the final plan: every gate (lint/test/
    red-runner/registry gate) must be green on merged master before the tag;
    `make release-snapshot` first, verify archive contents, then the real tag.
    Publishing is D-06's irreversible step.
15. **Port bands** — DEPLOYMENT.md demo blocks (if marked) must use the reserved
    [27900, 28100) doc band, not collide with the testutil allocator bands
    (28100/29100/30100/31100) — the examples_test.go header documents the reserve.
16. **`go tool grpcurl` shim, not PATH** — any new executable doc block resolves
    grpcurl through the existing shim; do not document a bare `grpcurl` dependency
    the harness does not provide.

---

## 13. Open questions resolved with evidence

| Question | Resolution |
|---|---|
| How many registry rows flip? | 22 (grep-verified list in §1); 167 rows total; exit criterion is zero `fixed-in-v1.1(` cells |
| Do the Q-09 deletions have targets? | No — all four candidates plus fsync_darwin.go are already gone (§8); plan is verify-and-record |
| Does the ruleset verify command fail today? | Yes, executed: `cannot iterate over: null`. By-id fix proven working with the exact documented output (§2, M-51) |
| Is the public proto already scrubbed? | Yes (Phase 6); the internal replproto still has D-01/D-02 on lines 16/26 — DOC-21's acceptance covers both protos (§10) |
| Do archives ship docs today? | No — no `files:` in .goreleaser.yaml; v1.0 archive had 4 entries (§7) |
| Is the macOS quickstart pair CI-proven? | Option-b: both README blocks run verbatim on Linux; darwin leg loud-skips (`quickstartBroadcastOption`); macos-15 native cell runs the full suite (§7) |
| What anchors TestOperationsDoc today? | ~60 anchor literals + 3 forbidden phrases + doc→exporter family grep; upgrade path to binary-verification is the iterator + structured metrics check (§6, §11) |
| Where do new machine checks live? | `cmd/sup-kv` main_test idiom (D-04), except FORMATS encode beside the codec (§6) |
| Is CHANGELOG still phase-suffixed? | Yes — seven `(Phase N)` suffixes in the 1.0.0 entry; `[Unreleased]` empty (§2/§12) |
| Is `make help` still missing? | Yes (M-52) — the `## target:` comments are present, so the target is a small addition |
| Does PERFORMANCE.md satisfy the benchmark-results requirement? | Largely — Phase 8 landed the hardware-block ledger with producing commands and commit stamps; the plan cross-checks coverage against DOCS-10's wording |
| Is there an ADR index / ARCHITECTURE.md / CONTRIBUTING / SECURITY? | No — all new (docs/adr/ has 12 unindexed ADRs; IDEA.md's banner already promises ARCHITECTURE.md by Phase 9) |

## 14. Remaining planner discretion (from CONTEXT + evidence)

- Exact OPERATIONS.md vs docs/METRICS.md split if the operator reference outgrows
  one file (D-03 default: one file).
- Machine-check test names, and the metrics/label coupling mechanism (structured
  `Snapshot` iteration recommended over string grep — §5/§11).
- systemd unit file path and the CI smoke shape (systemd-analyze verify +
  ExecStart-args vs full systemctl).
- goreleaser archive `files:` set (DEPLOYMENT.md + unit minimally; DOC-05
  acceptance says "lists docs/ and the unit" — decide whether all of docs/ ships).
- Whether the M-dispositions get one consolidated summary across plans (CONTEXT
  leaves it open; D-02 fixes per-plan tables as the source of truth).
- Whether bannered IDEA.md rows close via the banner or need content edits (§2).
- Whether the proven-claim checker uses adjacency regex or a claimed-tests table.
- CLAUDE.md marker handling (edit inside GSD markers vs rewrite section + markers).
- The goreleaser release-header slicing shape (§12.8).

---

*Sources: working tree at 4cf390a (all file paths absolute in the body above are
relative to `/home/nil/DiskD/W/Djarvur/sup-kv/`); live GitHub API queries executed
2026-11-27 (ruleset list + by-id); `go list ./...`; grep/read of every cited file.*
