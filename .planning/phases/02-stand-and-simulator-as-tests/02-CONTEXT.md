# Phase 2: Stand and Simulator as Tests - Context

**Gathered:** 2026-11-19
**Status:** Ready for planning
**Mode:** --auto (all gray areas auto-selected; every choice logged below and reviewable in git)

<domain>
## Phase Boundary

Turn the v1.0 review's verdict into machine-reproducible evidence: a real-socket test stand in CI (F9, F9b, F14, the 1000-writes/s × 15 s burst over 10 000 keys, F2c), a hardened simulator (datagram budgets, real-encoder size checks, virtual dump delay, restart-with-state, deletions, stable chaos), red tests for the four Criticals (R-01, R-02, R-03, P-01) that demonstrably fail against current code, seam-pinning tests (filesystem seam, fuzzing, deletion replication at every level, Watch branches, per-decision 14.x tests, property shrinking), and the scheduled heavy machinery. Requirements TEST-01…TEST-16. Phase 2 writes TESTS and harnesses only — it does NOT fix the underlying transport/protocol/persistence defects (that is Phases 3–5); red stays red here by design.

</domain>

<decisions>
## Implementation Decisions

### Red-test gating mechanism
- **D-01:** Critical-scenario red tests (R-01 paced sync over 20 000 keys, R-02 unacked-set cost and burst drain, R-03 closed-until-list interface, P-01 single value limit) live in the normal test packages but **skip unless `SUPKV_RED_TESTS=1`** is set. The scheduled CI workflow runs them with the variable set and **asserts the named tests FAIL** (a small runner script greps the test output for each expected-red test name in the FAIL section; absence of an expected red is an error). Red runs are recorded as phase evidence (test output saved under `.planning/phases/02-…/red-evidence/`). Phases 3–5 flip the assertion to "must pass" when the fix lands (the flip is a one-line change in the runner script's expectation table). — [auto] [Red-test gating] Q: how do intentionally-red tests merge without breaking CI? → Selected: env-gated skip + scheduled job asserting red (recommended default: keeps `make test` green, keeps red visible and machine-checked, no build-tag binary duplication). — **Reversibility:** reversible — flipping an expectation entry per fix.
- **D-02:** `TestChaos_convergence` mutation sensitivity: two named mutants ("sync without tombstones", "periodic sync disabled") are implemented as **code-level mutation probes in the runner script** (sed-edited copies of the target functions built in a temp overlay via `go build -overlay`), asserted to make the test fail on the fixed seed corpus; the production tree is never mutated in place. — [auto] [Mutation sensitivity] Q: how are mutants realized safely in CI? → Selected: `go build -overlay` temp overlays (recommended default: no repo mutation, no git state juggling).

### Real-socket stand
- **D-03:** The stand lives in a new package `internal/stand` (harness helpers: node spawn via the real `cmd/sup-kv` binary, temp dirs, broadcast socket setup, log capture, convergence polling) with scenario tests in `internal/stand/stand_*_test.go` guarded by `//go:build linux`. On non-Linux the package carries a single test that reports darwin replication untested loudly (TEST-05: no silent skips — a Linux CI skip is an error; the macOS cell states the limitation). Existing two-process exec patterns from v1.0's failure-scenario suite are the starting point, moved/extended rather than duplicated. — [auto] [Stand placement] Q: where does the real-socket harness live? → Selected: internal/stand package + linux build tag (recommended default: mirrors the repo's package-per-concern layout; keeps exec/process plumbing reusable).
- **D-04:** Scenario coverage on the stand: F9 (empty-disk peer, 10 000 × 1 KiB written offline), F9b (pure resync, 11 740 keys — scaled variant at 10 000), F14 (replaced node + `create_if_absent` seeding), burst 1000 writes/s × 15 s over 10 000 distinct keys (≥990 writes/s, unacked set empty 5 s after end), F2c (70 000-char key → `OK` then `sendto: message too long`, never replicated). Timing assertions use the review's stated bounds (F9b ≤ 2 × `sync_interval` + N/`sync_response_rate` + margin; burst ≥990/s and set-drain ≤5 s). — [auto] [Scenario set] Q: which scenarios and bounds? → Selected: the review's section-15 set verbatim with its measured bounds (recommended default: acceptance criteria are carried verbatim from the review).

### Simulator hardening
- **D-05:** simnet extensions land in `internal/app/internal/simnet` as additive API: per-node incoming-datagram budget (delivery refused + counted past budget), datagram size checks by the real encoder (`replproto` encode → length), record announcement delayed by a virtual dump latency, restart-with-preserved-state, and `Delete` in the cluster API. Chaos and scenario tests gain deletions in the mix. — [auto] [Simulator shape] Q: where do budget/size/dump-delay/restart/deletion capabilities go? → Selected: extend simnet additively (recommended default: keeps the deterministic core; existing sim_*_test.go continue to run unchanged).
- **D-06:** Chaos stability: `TestChaos_convergence` runs a **fixed seed corpus checked into `testdata/chaos-seeds.txt`** (regenerable by a generator test) plus fresh seeds each run; nightly-scale runs use the scheduled workflow. Combined-failure and long-partition (≥ `peer_timeout` + margin) cases become scenario tests with the documented bound (`not before first post-heal sync, not later than the bound`, with the no-fresh-loss qualifier). — [auto] [Chaos corpus] Q: how is 10 000-seed stability proven economically? → Selected: fixed corpus + fresh seeds per run locally, full-scale on schedule (recommended default: fast feedback in PR CI, scale nightly).

### Seams and pinning tests
- **D-07:** Filesystem seam under `persistfile`: a small `fsops` interface (write/fsync/rotate/rename/dir-fsync) with recording and failing fakes; a test asserts the exact operation ORDER and injects a failure at each step; a blocking `Save` fake proves `Set` does not return while the dump stands. `persist.Open` clock seeding is proven on the real composition path (`Open` with injected time earlier than dump max → first local write dominates). — [auto] [Seam shape] Q: how is "confirmed write is on disk" pinned? → Selected: fsops interface + order-asserting fakes (recommended default: TEST-03's acceptance wording maps 1:1 onto it).
- **D-08:** Negative decoder tests: corrupted-datagram table (bad CRC, truncated, wrong proto_version, bad run_id, oversize, empty key, tombstone-with-data) each asserting its discard class + counter; `FuzzDecode` and `FuzzDumpDecode` with in-repo `testdata/` corpora. Deletion replication pinned at four levels: codec round-trip (`Deleted: true` per message), simulator `Cluster.Delete`, two-process (Delete on A → `NotFound` on B, `"deleted":true` in B's dump), node-down-during-delete. Watch cut/backpressure: deterministic snapshot-then-live cut + slow-subscriber disconnect + `Send` error branch. Metrics by observed effect: `/metrics` after a conflict+retransmit+discard scenario asserts exact values. Per-decision tests reference 14.x by name in their comments/doc (TEST-13). Property shrinking: generators draw half the versions from a small `(l, c, node)` space; the printed replay command is executed by a test. — [auto] [Pinning set] Q: breadth of TEST-02/07/08/12/13/14/15 mechanics? → Selected: all as specified by TEST acceptance wording (recommended default: the requirements ARE the spec here).

### Scheduled heavy machinery
- **D-09:** New `.github/workflows/scheduled.yml` (cron nightly): 1000-seed chaos (`-shuffle=on -count=3`), bench smoke (`-bench . -benchtime=1x`), red-expected Critical run (D-01), `go test -race -count=20` over `cmd/sup-kv`, `udprepl` and affected packages (green ×3 across runs is the flakiness bar — a failing flake class that needs a Phase 3–5 fix carries its red test forward), and the nightly real-socket run at 10 000 keys × 1 KiB with the section-15 scenarios. PR CI keeps: lint, unit, race unit, Linux socket tests (mandatory, non-skippable). Benchmarks report p50/p99 without self-pacing on the real `persistfile` (no fake disk, default params). — [auto] [CI topology] Q: where does heavy machinery run? → Selected: one scheduled workflow + tightened ci.yml (recommended default: single place to look; nightly cost off the PR path).

### Claude's Discretion
- Exact helper APIs, test names (dot-consistent with existing suite), seed-corpus file format, runner-script implementation details, how the red-evidence directory is structured.
- Which existing v1.0 two-process tests move into `internal/stand` vs stay.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Review inputs (committed, public)
- `docs/reviews/2026-11-19-v1.0/REVIEW.md` — §3/§4 (stand findings F2c/F9/F9b/F14 + burst evidence), §5–§7 (R-01…R-16, P-01…P-10, S-xx), §11 (T-01…T-16 — the requirement source), §14 (decisions the tests reference by name), §15 (scenario list and phase composition)
- `docs/reviews/2026-11-19-v1.0/evidence/RESULTS.md` and `evidence/stand/` — the reviewer's measured numbers every stand test must reproduce within its stated bounds
- `docs/reviews/2026-11-19-v1.0/evidence/repro/README.md` — the reviewer's repro layout (reference only; the subtree is a foreign Go module — never move it into the main module's package graph)
- `docs/SPEC-DEVIATIONS.md` — decision names/ADR references TEST-13 tests cite (Phase 1 output)
- `docs/adr/0005-interface-closed-until-full-key-list.md`, `docs/adr/0001…0011` — the 14.x decisions tests reference by name

### Planning anchors
- `.planning/ROADMAP.md` — Phase 2 goal + 5 success criteria
- `.planning/REQUIREMENTS.md` — TEST-01…TEST-16 acceptance wording (verbatim source of every gate)
- `.planning/phases/01-spec-reference-and-decisions/01-CONTEXT.md` — locked Phase 1 decisions (registry/ADR names)
- `.planning/PROJECT.md` — constraints (quality gates, trusted-LAN model, scale envelope)
- `.planning/STATE.md` — decision note: red tests precede fixes; gating mechanism is THIS phase's to choose (D-01 answers it)

### Code (existing seams to extend)
- `internal/app/internal/simnet/simnet.go` — the deterministic virtual network (D-05 target)
- `internal/app/repl/sim_chaos_test.go`, `sim_bound_test.go`, `sim_scenario_test.go` — chaos/bounds/scenario tests to stabilize and extend
- `internal/grpcapi/` tests, `internal/persistfile/` tests — Watch/persist seams (D-07/D-08 targets)
- `.github/workflows/ci.yml` — the CI file scheduled.yml complements (D-09)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/app/internal/simnet` — goroutine-free discrete-event virtual net with loss/duplication/reordering/partition; the hardening target and the model for budget/size/delay extensions
- v1.0 failure-scenario suite (`cmd/sup-kv` two-process exec tests) — process spawn/kill/restart plumbing to move into `internal/stand`
- `internal/app/repl` sim tests — seeded, replayable chaos harness (`SUPKV_SIM_SEED`) with convergence-bound assertions to extend
- The reviewer's repro tests (`docs/reviews/2026-11-19-v1.0/evidence/repro/`) — worked red-test examples for R-01/R-02/R-03/P-01 semantics (foreign module; consult, do not import)

### Established Patterns
- Deterministic simulation runs the production repl core — new simulator features must keep that property
- Tests are table-driven with named subtests; quality gates (strict lint, race) from the first commit
- `make lint` / `make test` are the pre-push gates; CI matrix Linux/macOS × amd64/arm64

### Integration Points
- `.github/workflows/scheduled.yml` (new) + `ci.yml` (tightened)
- `testdata/` corpora (chaos seeds, fuzz) — in-repo, committed
- `internal/stand` — new package wired into existing CI via `go test ./internal/stand/`

</code_context>

<specifics>
## Specific Ideas

- The reviewer's measured numbers are the bounds to encode: F9b ≈ 83 s bound breached to ~1000 s (the red R-01 evidence), F9 14.5-min retransmit storm (red R-02), burst 465/s achieved vs 990 required, F14 default-seed displacement in 3 s. Stand tests assert the BOUND, red tests assert the CURRENT failure.
- macOS loud-skip: the darwin leg of the stand package states "darwin replication untested" as a named test result (TEST-05), continuing the ratified option-b pattern.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 2-Stand and Simulator as Tests*
*Context gathered: 2026-11-19*
