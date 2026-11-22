---
phase: 06-api-and-semantics
plan: 03
subsystem: public-api
tags: [apis-06, client-example, nested-module, public-api, consumer-zone, adr-0007, make-target, ci-job, status-complete]
requires:
  - phase: 06-api-and-semantics
    plan: 01
    provides: "the public client wire schema at github.com/Djarvur/sup-kv/api/supkvapi/v1, including the List page fields"
provides:
  - "a standalone examples/client Go module (require + replace ../..) that compiles only against the public package — the consumer-importability proof for APIS-06"
  - "a make client-example target and a dedicated client-example CI job that build, vet and tidy-prove the nested module"
  - "a README documenting the RPC walkthrough and the last-read-value consumer-library boundary (ADR 0007, 14.7)"
affects: [06-13-phase-gate, phase-9-docs, apis-06-acceptance]
actuals:
  tokens: 3643
  tasks: 3
  commits: 3
  plan_head_before: fc4ab655e6b08e3e04c927c771fa7379eca5709b
  plan_head_after: 149d7993dbdc4b34b2cf728414c38baee7dca36b
tech-stack:
  added: []
  patterns:
    - "A nested Go example module (own go.mod, `require github.com/Djarvur/sup-kv v0.0.0` + `replace => ../..`) is the cheapest compile-time proof that the public package is importable from another module: `go build` fails on any accidental internal/ dependency."
    - "One Makefile target is the single definition of the nested module's gate; the CI job invokes exactly that target (build + vet + tidy diff), so local and CI behavior cannot drift."
    - "The example is a demonstration, not a client library: the last-read-value cache is stated as consumer-library territory per ADR 0007 (14.7) and is deliberately absent."
key-files:
  created:
    - examples/client/go.mod
    - examples/client/go.sum
    - examples/client/main.go
    - examples/client/README.md
  modified:
    - Makefile
    - .github/workflows/ci.yml
    - .gitignore
key-decisions:
  - "go mod tidy normalized google.golang.org/protobuf to an indirect requirement (18-line go.sum, matching the research probe): main.go never imports protobuf directly, and tidy's canonical form is the committed one. The module still requires exactly the root module (replace), gRPC and protobuf — the plan's dependency boundary holds; only the direct/indirect classification is tidy's."
  - "The example writes one key, exactly as the plan specifies, so the List page loop usually completes in one page on a fresh node; the loop is the demonstrated pattern (page_size=2, follow next_page_token to the empty end signal) and paginates against a populated prefix."
  - "The plan-head ledger .git/gsd-plan-head-before-06-03 was stale (eaf88c00, a prior session's HEAD) and was reset to this plan's true pre-plan HEAD fc4ab65 before measuring commits/tokens — the 06-01 precedent."
  - "APIS-06's REQUIREMENTS.md checkbox was deliberately not flipped: requirements.ready-ids reports it blocked because sibling plans 06-09, 06-10 and 06-13 also declare APIS-06 and have no SUMMARY yet (#2388 shared-ID gate). The SUMMARY records APIS-06 as this plan's requirement; the checkbox flips when the last sibling completes."
patterns-established:
  - "Nested-module gate: a separate module is excluded from the root graph by construction, so it needs its own single target and its own CI job; `go list ./...` is the guard that the exclusion holds."
  - "Consumer-zone documentation: the README names the boundary (ADR 0007 / 14.7) and lists what the example intentionally omits, so an example can never be mistaken for a supported client library."
requirements-completed: [APIS-06]
coverage:
  - id: D1
    description: "examples/client is a separate Go module (module github.com/Djarvur/sup-kv/examples/client) requiring the root module through replace ../.. and building only against the public package."
    requirement: "APIS-06"
    verification:
      - kind: process
        ref: "cd examples/client && go build ./... && go vet ./... -> exit 0"
        status: pass
      - kind: process
        ref: "grep -q 'replace github.com/Djarvur/sup-kv => ../..' examples/client/go.mod -> exit 0"
        status: pass
      - kind: process
        ref: "grep -q 'github.com/Djarvur/sup-kv/api/supkvapi/v1' examples/client/main.go -> exit 0; grep -c 'github.com/Djarvur/sup-kv/internal' examples/client/main.go -> 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "The example exercises Set with wait_for_replication, Get, a List page loop following next_page_token to the empty end signal, and one bounded Watch event — all type-checked against the generated client."
    requirement: "APIS-06"
    verification:
      - kind: process
        ref: "source proofs (the plan's test is the build itself): WaitForReplication: true, WaitTimeoutMs: 1000, PageSize: 2, GetNextPageToken() loop exit, context.WithTimeout(ctx, watchWindow), stream.Recv() — all present; `go build` compiles them against supkvapi.SupKVServiceClient"
        status: pass
    human_judgment: false
  - id: D3
    description: "make client-example builds, vets, tidies and proves the committed go.mod/go.sum are tidy (no diff)."
    requirement: "APIS-06"
    verification:
      - kind: process
        ref: "make client-example -> exit 0 (cd examples/client && go build ./... && go vet ./... && go mod tidy && git diff --exit-code -- go.mod go.sum)"
        status: pass
      - kind: process
        ref: "h1=$(git hash-object go.mod go.sum); go mod tidy; test \"$h1\" = \"$(git hash-object go.mod go.sum)\" -> exit 0 (tidy idempotent)"
        status: pass
    human_judgment: false
  - id: D4
    description: "CI has a dedicated client-example job that runs exactly the make target, with checkout/setup-go pinning and a 10-minute timeout."
    requirement: "APIS-06"
    verification:
      - kind: process
        ref: "grep -q 'client-example' .github/workflows/ci.yml -> exit 0; the job's single step is `run: make client-example` (ci.yml:110)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The root go test ./... and go list ./... never include the nested module; the root gate is unchanged."
    requirement: "APIS-06"
    verification:
      - kind: process
        ref: "test -z \"$(go list ./... | grep examples/client)\" -> exit 0"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./... -> exit 0 on rerun; the first attempt hit the documented pre-existing cross-package UDP port-contention flake in cmd/sup-kv (TestLog_formatText), which passes standalone (`go test -race -count=1 ./cmd/sup-kv/` -> ok 5.893s)"
        status: pass
    human_judgment: false
  - id: D6
    description: "The README documents the run command, flags, each demonstrated RPC (List follows next_page_token; Watch is bounded), and the last-read-value/consumer-library boundary with ADR 0007 and a docs/API.md pointer; it contains no internal package path."
    requirement: "APIS-06"
    verification:
      - kind: process
        ref: "grep -q 'last-read-value' && grep -q 'consumer' && grep -qi 'consumer-library' && grep -q 'next_page_token' && grep -q 'ADR 0007' examples/client/README.md -> exit 0"
        status: pass
      - kind: process
        ref: "test \"$(grep -c 'github.com/Djarvur/sup-kv/internal' examples/client/README.md)\" -eq 0 -> exit 0"
        status: pass
      - kind: process
        ref: "make client-example -> exit 0 after the README-only edit"
        status: pass
    human_judgment: false
duration: 7min
completed: 2026-11-22
status: complete
---

# Phase 06 Plan 03: Standalone Client Example Module Summary

**A separate `examples/client` Go module now compiles in CI against the public `api/supkvapi/v1` package, demonstrating Set-with-wait, Get, a paginated List loop and one bounded Watch event, gated by a single `make client-example` target and a dedicated CI job.**

## Performance

- **Duration:** 7min
- **Started:** 2026-11-22T22:50:00Z
- **Completed:** 2026-11-22T22:57:26Z
- **Tasks:** 3
- **Files modified:** 7 (4 created, 3 modified)

## Accomplishments

- **The consumer-importability proof exists as a real module.** `examples/client/go.mod` declares `module github.com/Djarvur/sup-kv/examples/client`, requires `github.com/Djarvur/sup-kv v0.0.0` with `replace => ../..`, and pins `google.golang.org/grpc v1.85.0` / `google.golang.org/protobuf v1.36.13-0.20261016172727-e9ee2a76ab8e` exactly as the root `go.mod` does. `go build` therefore fails if the public package ever grows an `internal/` dependency.
- **The example exercises the four RPC patterns a consumer uses.** `Set` sends `wait_for_replication=true` + `wait_timeout_ms=1000` and prints version/acked/acked_count/total_peers; `Get` reads the same key; `List` loops with `page_size=2` following `next_page_token` until it is empty, printing each page's records and token; `Watch` runs under a 3-second `context.WithTimeout`, prints exactly one received `Record` (the snapshot guarantees one for the written key) and treats the window closing as a clean end.
- **One target is the nested module's whole gate.** `make client-example` runs `cd examples/client && go build ./... && go vet ./... && go mod tidy && git diff --exit-code -- go.mod go.sum`; `go mod tidy` is byte-idempotent (`git hash-object` comparison both before and after the commit).
- **CI invokes exactly that target.** A dedicated `client-example` job (`ubuntu-latest`, `timeout-minutes: 10`, `actions/checkout@v7`, `actions/setup-go@v7` with `go-version-file: go.mod`) has one step: `run: make client-example`. It neither regenerates protos nor touches the root `go.mod`/`go.sum`.
- **The root graph is untouched.** `go list ./...` prints no `examples/client` package; `go test -race -count=1 ./...` is green on rerun (the one failure was the documented pre-existing UDP port-contention flake in `cmd/sup-kv`, which passes standalone).
- **The consumer-zone boundary is stated where a consumer will read it.** The README names the separate module and why the replace proves importability, the exact run command and flags, the per-RPC walkthrough (including `next_page_token` as the end-of-list signal and the bounded Watch), the last-read-value cache as consumer-library territory per ADR 0007 (14.7), a `docs/API.md` pointer, and a "What this example does not do" list (no auth, no retries, no local cache, no snapshot RPCs).

## Task Commits

1. **Task 1: standalone client module (go.mod, go.sum, main.go, README)** — `c3cd554` (feat)
2. **Task 2: Makefile target and CI job** — `4a356b3` (chore)
3. **Task 3: README walkthrough and consumer-boundary tightening** — `149d799` (docs)

**Plan metadata:** the `docs(06-03)` completion commit (this commit).

## Files Created/Modified

- `examples/client/go.mod` (created) — the nested module declaration, the root-module require + replace, and the gRPC/protobuf pins copied from the root `go.mod`.
- `examples/client/go.sum` (created) — 18 lines, matching the research probe's tidy result.
- `examples/client/main.go` (created) — `package main`; `-addr` (default `127.0.0.1:28011`) and `-prefix` (default `example/`) flags; `grpc.NewClient` with insecure credentials; the Set/Get/List/Watch sequence with non-zero exit on the first error; no `internal/` import and no cache.
- `examples/client/README.md` (created, tightened in Task 3) — run command, flags, RPC table, consumer-library boundary, and the explicit non-goals list.
- `Makefile` (modified) — `client-example` added to `.PHONY`; the single recipe is the nested module's gate.
- `.github/workflows/ci.yml` (modified) — the dedicated `client-example` job; required check names of existing jobs unchanged.
- `.gitignore` (modified) — ignores `/examples/client/client`, the executable `go build ./...` writes inside the nested module.

## Decisions Made

- **`go mod tidy` owns the require classification.** The plan listed protobuf as a direct require, but `main.go` never imports it; tidy moved it to the indirect block (and the generated code's own import keeps it in the module graph). The plan's boundary — root module (replace) + gRPC + protobuf only — holds; the committed `go.mod` is tidy's canonical form, which is what the idempotency gate must pin.
- **One written key, exactly as the plan specifies.** The List loop therefore usually completes in one page against a fresh node; it still demonstrates the pattern (bounded `page_size`, token following, empty-token end signal) and paginates against a prefix that already holds more keys.
- **The stale plan-head ledger was corrected.** `.git/gsd-plan-head-before-06-03` existed from a prior session and pointed at `eaf88c00` (three weeks of history). It was reset to this plan's true pre-plan HEAD `fc4ab65` before measuring, so `commits: 3` and `tokens: 3643` describe this plan — the same fix 06-01 documented.
- **APIS-06 is not flipped in REQUIREMENTS.md yet.** `requirements.ready-ids` reports it blocked: sibling plans 06-09, 06-10 and 06-13 also declare APIS-06 and have no SUMMARY, so the #2388 shared-ID gate defers the checkbox until the last sibling finishes. The SUMMARY records the requirement as this plan's; no premature `Complete` row is written.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `go build ./...` wrote an executable into the example module**
- **Found during:** Task 2's first `make client-example` run
- **Issue:** the plan's recipe (`go build ./...` inside `examples/client`) writes the `main` package's binary to `examples/client/client`, leaving an untracked 16 MB artifact in the tree on every local run (it must never be committed).
- **Fix:** added a scoped `.gitignore` entry (`/examples/client/client`) rather than changing the plan's literal recipe or redirecting the build output; the artifact is now ignored and the tree stays clean after the target runs.
- **Files modified:** `.gitignore`
- **Verification:** `git check-ignore -v examples/client/client` matches the new line; `make client-example` leaves `git status --short` clean.
- **Commit:** `4a356b3`

**2. [Rule 1 - Bug] The plan-head ledger was stale and would have mis-measured the SUMMARY**
- **Found during:** the close-out measurement
- **Issue:** `.git/gsd-plan-head-before-06-03` predated this run and pointed at `eaf88c00`; measuring from it counted 409 commits and the whole v1.1 planning history.
- **Fix:** reset the ledger to `fc4ab655` (the commit this plan's first task commit sits on) and re-measured: `commits: 3`, `tokens: 3643` (chars/4 over `git diff fc4ab65..HEAD`).
- **Files modified:** `.git/gsd-plan-head-before-06-03` (local `.git` metadata, not a tracked file)
- **Verification:** `git rev-list --count fc4ab65..HEAD` = 3; the diff stat lists exactly the 7 plan files.
- **Commit:** none (not a repository file)

**3. [Rule 1 - Bug] The ROADMAP progress updater wrote into the v1.0 phase-6 row again**
- **Found during:** the metadata close-out, reading the resulting `ROADMAP.md`
- **Issue:** `gsd_run query roadmap.update-plan-progress 06` matched the **v1.0** phase-6 row (`6. Hardening … | v1.0 | 9/9 | Complete | 2026-11-16`) and rewrote it as `3/13 | In Progress`, leaving the v1.1 row at `2/13`. The phase-detail section and the 06-03 checkbox were updated correctly; only the summary-table row collided. This is the exact defect 06-01 logged in `deferred-items.md`.
- **Fix:** restored the v1.0 row byte-for-byte and set the v1.1 row to `3/13 | In Progress`; recorded the recurrence in the phase deferred-items ledger.
- **Files modified:** `.planning/ROADMAP.md`, `.planning/phases/06-api-and-semantics/deferred-items.md`
- **Verification:** `grep -n "^| 6\." .planning/ROADMAP.md` shows v1.0 at `9/9 | Complete | 2026-11-16` and v1.1 at `3/13 | In Progress`.
- **Commit:** this metadata commit

## Issues Encountered

- **The documented cross-package port-contention flake fired once.** The first `go test -race -count=1 ./...` failed `cmd/sup-kv`'s `TestLog_formatText` with `bind: address already in use` on a UDP probe port — the pre-existing `internal/stand` fixed-port contention across parallel packages recorded in 06-01's SUMMARY. The package passes standalone (`ok 5.893s`) and the full rerun was green in every package. This plan cannot have caused it: `examples/client` is not in the root package graph and binds no sockets.
- **`go mod tidy` reclassified protobuf as indirect.** Expected and recorded as a decision above; the second tidy run leaves `go.mod`/`go.sum` byte-identical.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Ready for 06-04.** The example module, its Makefile target and its CI job are committed; the public-package consumer proof is in place for the phase gate.
- **Runtime proof is deliberately out of scope.** CI builds, vets and tidy-proves the example; it does not start a node and run the example against it. The plan's stated test is the build itself, and the RPC sequence is type-checked against the generated client.
- **APIS-06 completes when its sibling plans do.** The requirement is shared with 06-09, 06-10 and 06-13; this plan's SUMMARY is one of the inputs the shared-ID gate waits on.

## Self-Check: PASSED

- Created files exist: `examples/client/{go.mod,go.sum,main.go,README.md}`, this SUMMARY — all FOUND.
- Task commits are ancestors of HEAD: `c3cd554`, `4a356b3`, `149d799` — all FOUND.
