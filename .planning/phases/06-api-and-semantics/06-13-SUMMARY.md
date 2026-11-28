---
phase: 06-api-and-semantics
plan: 13
subsystem: phase-gate-verification
tags:
  - apis-01
  - apis-02
  - apis-03
  - apis-04
  - apis-05
  - apis-06
  - apis-07
  - frozen-content-pin
  - red-runner
  - mutation-stage
  - transport-envelope
  - standing-gates
  - status-complete
requires:
  - phase: 06-api-and-semantics (06-01 … 06-12)
    provides: every phase behavior (public schema, honest wait, error mapper, paginated List, durable Watch, transport envelope, rollback, docs) this gate re-verifies
  - phase: 02-stand-and-simulator-as-tests
    provides: the frozen red tests, scripts/red-expectations.tsv and scripts/red-runner (including its mutation stage)
provides:
  - "the recorded phase-gate result: the three frozen paths still hash to their planning-time literals, and the red-runner reports 7/7 with 0 errors plus two killed mutants"
  - "make lint, make test, make buf-lint, make buf-breaking and make client-example green on the final phase tree"
  - "the corrected TestStand_oversizeKeyF2c assertion that names the layer rejecting each bad key — transport ResourceExhausted over the envelope, handler InvalidArgument inside it"
affects:
  - phase-verification
  - phase-07-observability-and-configuration
actuals:
  tokens: 1803
  tasks: 2
  commits: 1
  plan_head_before: 176f7571a1423a6607ad8839fc7df9e788a15659
  plan_head_after: 930382c6130f59d7f7562faf150677a766110754
tech-stack:
  added: []
  patterns:
    - "Phase gate as content, not ref: three `git hash-object` literals pin the frozen evidence, so the check reproduces on any branch, in any commit, and for an untracked file."
    - "A rejection assertion names its layer: an over-envelope message expects ResourceExhausted from the transport and a key over the bound (inside the envelope) expects InvalidArgument from the handler, so the code proves which layer spoke."
key-files:
  created:
    - .planning/phases/06-api-and-semantics/06-13-SUMMARY.md
  modified:
    - internal/stand/stand_test.go
    - internal/stand/stand.go
key-decisions:
  - "The gate compares content hashes captured from the phase-start tree; no merge-base is involved because the frozen paths do not exist at the milestone base — a mounting path that never existed at the ref is irrelevant."
  - "The red-runner is run without flags so the mutation stage executes; `--skip-mutants` remains local-iteration only."
  - "TestStand_oversizeKeyF2c's InvalidArgument assertion predated 06-07's explicit transport envelope: a 70 000-byte key now exceeds MaxRecvBytes (a maximum valid Set plus the envelope), so the documented verdict is ResourceExhausted at the transport (docs/API.md, Transport). The handler's key bound is kept at process level with a key one byte over storage.MaxKeyBytes (inside the envelope), and the handler-level proof also stays pinned by TestServer_keyLimit in internal/grpcapi."
  - "The rejected key's live absence is read through List, not Get: a Get of the oversize key is itself over the receive envelope and would prove nothing about the store, while List's small request reports the live key set."
  - "No frozen file, expectation-table row or pinned literal was edited; the three literals are the planning-time content and a red row is a regression to report, never a table edit."
requirements-completed: [APIS-01, APIS-02, APIS-03, APIS-04, APIS-05, APIS-06, APIS-07]
coverage:
  - id: D1
    description: "The three frozen paths hash to the planning-time literals: internal/grpcapi/red_gate_test.go = 519789bd…, internal/stand/stand_red_test.go = 573fed68…, scripts/red-expectations.tsv = add45b810…."
    requirement: "APIS-01 … APIS-07 (phase gate)"
    verification:
      - kind: process
        ref: "a=$(git hash-object internal/grpcapi/red_gate_test.go) && b=$(git hash-object internal/stand/stand_red_test.go) && c=$(git hash-object scripts/red-expectations.tsv) && test \"$a\" = 519789bd1f7abb2af09053b81c26e0ce45d9bdaf && test \"$b\" = 573fed68d2f1a039b8ac023a342446103064273b && test \"$c\" = add45b810d36d69fd574ca83cb575dab5722a55b -> exit 0; printed red_gate=519789… stand_red=573fed… tsv=add45b8…"
        status: pass
    human_judgment: false
  - id: D2
    description: "The full red-runner reports exactly 7 expectation rows checked with 0 errors and the mutation stage kills both mutants on the final committed tree."
    requirement: "APIS-01 … APIS-07 (phase gate)"
    verification:
      - kind: integration
        ref: "SUPKV_RED_TESTS=1 bash scripts/red-runner -> exit 0; summary \"red-runner: 7 expectation row(s) checked, 0 error(s)\"; mutants sync-without-tombstones and periodic-sync-disabled both killed, control passed; evidence .planning/phases/02-stand-and-simulator-as-tests/red-evidence/20261123T021142Z/"
        status: pass
    human_judgment: false
  - id: D3
    description: "The strict lint gate is green on the final tree (config verify plus the full run)."
    requirement: "APIS-01 … APIS-07 (phase gate)"
    verification:
      - kind: process
        ref: "make lint -> 'go tool golangci-lint config verify' and 'go tool golangci-lint run ./...' both exit 0"
        status: pass
    human_judgment: false
  - id: D4
    description: "The full race-detector suite is green across every package the root module lists, including internal/stand on the real binary."
    requirement: "APIS-01 … APIS-07 (phase gate)"
    verification:
      - kind: integration
        ref: "make test (go test -race -cover ./...) -> exit 0 in 2m49s; internal/stand ok 7.250s; every other package ok"
        status: pass
    human_judgment: false
  - id: D5
    description: "The wire-compatibility gates are green with no waiver: buf lint passes the STANDARD rule set and buf breaking runs the real FILE-policy comparison against origin/master (the proto is present there, so the D-08 deferral does not apply)."
    requirement: "APIS-01 … APIS-07 (phase gate)"
    verification:
      - kind: process
        ref: "make buf-lint -> exit 0; make buf-breaking -> exit 0 (git cat-file -e refs/remotes/origin/master:internal/udprepl/replproto/supkv/v1/supkv.proto succeeds, so go tool buf breaking ran)"
        status: pass
    human_judgment: false
  - id: D6
    description: "The standalone consumer example still builds, vets and keeps its go.mod/go.sum tidy."
    requirement: "APIS-06"
    verification:
      - kind: build
        ref: "make client-example -> cd examples/client && go build ./... && go vet ./... && go mod tidy && git diff --exit-code -- go.mod go.sum -> exit 0"
        status: pass
    human_judgment: false
  - id: D7
    description: "The F2c stand scenario proves both rejection layers on the real binary and that the rejected keys never reach the peer or either dump."
    requirement: "APIS-05"
    verification:
      - kind: integration
        ref: "go test -race -count=1 ./internal/stand/ -run '^TestStand_oversizeKeyF2c$' -> PASS (1 025-byte key InvalidArgument, 70 000-byte key ResourceExhausted, List absence on A, full-window List absence on B, boundary record replicated and in B's dump)"
        status: pass
    human_judgment: false
duration: 20min
completed: 2026-11-23
status: complete
---

# Phase 06 Plan 13: Phase Gate — Frozen Evidence and Standing Gates Summary

**The phase's frozen red evidence is byte-identical to its phase-start pin (three content hashes), the red-runner is 7/7 with 0 errors and both mutants killed, and lint, the full race suite, buf lint/breaking and the Go client example are green on the final tree.**

## Performance

- **Duration:** 20min
- **Started:** 2026-11-23T01:53:57Z
- **Completed:** 2026-11-23T02:13:30Z
- **Tasks:** 2
- **Files modified:** 2 (plus this summary); `git diff 176f757..930382c` is 68 insertions / 20 deletions across `internal/stand/stand.go` and `internal/stand/stand_test.go`

## Task Commits

1. **Task 1 (deviation found by the gate): align the F2c stand assertion with the transport envelope** - `930382c` (fix)
   - The gate itself produced no production commit; this is the Rule 1 fix it forced.
2. **Task 2: standing lint/test/buf/client gates** - no commit (verification only)

**Plan metadata:** see the final `docs(06-13)` commit.

## Gate Results

| Check | Command | Result |
|-------|---------|--------|
| Frozen content pin | three `git hash-object` values compared to the planning-time literals | PASS — `red_gate=519789bd1f7abb2af09053b81c26e0ce45d9bdaf`, `stand_red=573fed68d2f1a039b8ac023a342446103064273b`, `tsv=add45b810d36d69fd574ca83cb575dab5722a55b` |
| Red runner | `SUPKV_RED_TESTS=1 bash scripts/red-runner` (no flags) | PASS — `red-runner: 7 expectation row(s) checked, 0 error(s)`; both mutants killed; evidence `20261123T021142Z` |
| Strict lint | `make lint` | PASS — config verify + `go tool golangci-lint run ./...` |
| Race suite | `make test` | PASS — 2m49s; `internal/stand` ok 7.250s; all packages ok |
| Buf lint | `make buf-lint` | PASS — STANDARD rule set, zero waivers |
| Buf breaking | `make buf-breaking` | PASS — the real FILE-policy comparison against `origin/master` ran (the proto is present there; no D-08 deferral) |
| Client example | `make client-example` | PASS — build, vet and tidy-proof |

The runner's evidence directories are intentionally untracked (`red-evidence/20261123T015556Z` = the real F2c failure, `20261123T020338Z` = the convergence flake, `20261123T020555Z` and `20261123T021142Z` = the green runs). None was deleted or rewritten.

## Decisions Made

- **The gate compares content, not a ref.** The three literals were captured from the phase-start tree; the milestone merge-base cannot serve because the frozen paths do not exist there. A mismatch prints the actual hashes instead of deriving an expectation from a changing ref.
- **The F2c assertion now names the rejecting layer.** The 1 025-byte key stays inside the transport envelope and answers `InvalidArgument` from the handler's key bound; the 70 000-byte key is over `MaxRecvBytes` and answers `ResourceExhausted` at the transport, exactly as `docs/API.md` (Transport) documents. The handler half is also pinned by `TestServer_keyLimit` in `internal/grpcapi`.
- **Live absence is read with `List`, not `Get`.** A `Get` of the oversize key is itself over the receive envelope, so it would be rejected before any handler and prove nothing about the store; `List`'s small request reports the live key set, so the "not served on A" and "never replicated to B" claims stay real.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] TestStand_oversizeKeyF2c asserted the pre-06-07 error precedence**

- **Found during:** Task 1, first full `bash scripts/red-runner` run (`20261123T015556Z`).
- **Issue:** the stand scenario Set a 70 000-byte key and wanted `InvalidArgument` from the handler key bound, but the deterministic answer was `rpc error: code = ResourceExhausted desc = grpc: received message larger than max (70007 vs. 63846)`. Plan 06-07's explicit transport envelope (`MaxRecvBytes = MaxKeyBytes + effective value limit + 1 382`) deliberately rejects an over-envelope message before any handler, and `docs/API.md` (Transport) records that verdict. The assertion predated the envelope; it was not a production regression.
- **Fix:** the test now asserts both layers — a 1 025-byte key (`storage.MaxKeyBytes+1`, inside the envelope) answers `InvalidArgument` from the key bound, and the 70 000-byte key answers `ResourceExhausted` from the transport. The rejected key's absence is read through a new `standListPrefix` helper (`List` with the `f2c-` prefix) for both the on-A and full-window on-B checks, because a `Get` of that key is itself over the envelope. The final dump assertions are unchanged.
- **Files modified:** `internal/stand/stand_test.go`, `internal/stand/stand.go`.
- **Verification:** `go test -race -count=1 ./internal/stand/ -run '^TestStand_oversizeKeyF2c$'` → PASS; final red-runner 7/7 with the mutant stage green; `make test` green.
- **Commit:** `930382c`.

**2. [Rule 1 - Bug] `make lint` found a govet shadow after the first fix**

- **Found during:** Task 2, the first `make lint` on the fix.
- **Issue:** `internal/stand/stand_test.go:650` shadowed the outer `listed` inside the B-absence poll (`govet` shadow), so the strict lint gate failed.
- **Fix:** renamed the poll-local variable to `polled`; behaviour unchanged. Folded into the same commit via `--amend` so the plan keeps one coherent fix commit.
- **Files modified:** `internal/stand/stand_test.go`.
- **Verification:** `make lint` → exit 0.
- **Commit:** `930382c`.

No frozen file, expectation-table row or pinned literal was edited, and no gate was narrowed (`--skip-mutants` was not used).

## Issues Encountered

- **Documented stand flake fired once.** The second red-runner run (`20261123T020338Z`) failed `TestStand_killNineRestart` at its 6 s baseline-convergence wait under the parallel `-race` stand load; the immediate re-run (`20261123T020555Z`) was green, and the final run (`20261123T021142Z`) was green again on the committed tree. This is the load-sensitive stand class already recorded in `deferred-items.md`; no bound, test or expectation row was weakened.
- **ROADMAP v1.0-row collision recurred.** `roadmap.update-plan-progress 06` again targets the v1.0 phase-6 row by number. Repaired by hand (v1.0 row restored to `9/9 | Complete | 2026-11-16`; v1.1 row set to `13/13 | Complete`), matching the earlier 06-01/06-03 entries in `deferred-items.md`.
- **Out-of-order wave execution left the plan counter stale** (`Current Plan: 11` while 12 plans were already closed). The phase's plans executed in wave order (…, 11, 4, 5, 6, 12, 7, 8, 9, 10, 13), so the counter and the plan IDs diverged; the final position was reconciled to 13/13 by hand.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Phase 6 is complete: 13/13 plans.** All seven APIS requirements are marked complete, and the gate record above is the phase-level proof that the moving parts (Watch watermark, List budget, error mapper, rollback, public proto) left the frozen red evidence and the standing lint/test/wire contract intact.
- **Ready for phase verification.** Phase 7 (Observability and Configuration) is the next phase in v1.1.
- **Carried-forward items:** the pre-existing v1.1 Phase 4/5 progress-table drift (`0/?`) and the documented stand/port flakes remain with their owners in `deferred-items.md`; neither blocks verification.

## Self-Check: PASSED
