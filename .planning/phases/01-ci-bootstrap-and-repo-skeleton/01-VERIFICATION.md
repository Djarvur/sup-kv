---
phase: 01-ci-bootstrap-and-repo-skeleton
verified: 2026-11-02T16:20:24Z
status: passed
score: 25/25 must-haves verified
covered_files:
  - .claude/CLAUDE.md
  - .github/dependabot.yml
  - .github/workflows/ci.yml
  - .github/workflows/vulncheck.yml
  - .gitignore
  - .golangci.yml
  - .planning/PROJECT.md
  - .planning/REQUIREMENTS.md
  - .planning/STATE.md
  - .planning/WINDOWS.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-01-PLAN.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-01-SUMMARY.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-02-PLAN.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-02-SUMMARY.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-03-PLAN.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-03-SUMMARY.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-04-PLAN.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-04-SUMMARY.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-05-PLAN.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-05-SUMMARY.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-06-PLAN.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-06-SUMMARY.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-UAT.md
  - AGENTS.md
  - Makefile
  - cmd/sup-kv/main.go
  - docs/setup/branch-protection.md
  - docs/setup/ruleset.json
  - go.mod
  - go.sum
  - internal/app/clock/clock.go
  - internal/app/clock/clock_test.go
  - internal/app/clock/hlc_prop_test.go
  - internal/app/internal/testutil/gen.go
  - internal/app/internal/testutil/rng.go
  - internal/app/storage/doc.go
  - internal/app/storage/errors.go
  - internal/app/storage/merge_test.go
  - internal/app/storage/store.go
  - internal/app/storage/store_test.go
  - internal/app/version/version.go
  - internal/app/version/version_test.go
  - internal/port/doc.go
covered_digest: "v4:sha256:907fbffae6513c4badfa2558f93a8da986c1843db30a294efb1029f68b74f54c"
behavior_unverified: 0
overrides_applied: 4
re_verification:
  previous_status: human_needed
  previous_score: 24/25
  gaps_closed:
    - "Dependabot first-PR backstop (prior truth 17, WINDOWS.md item 1, UAT item 1): window elapsed Mon 2026-11-02 and resolved by recorded maintainer decision 2026-11-02 (UAT Test 1, option A) — no version-update PR exists because nothing is updatable (golangci-lint v2.15.0 IS the latest upstream release, verifier-confirmed live via gh api this pass, and deliberately ignored per the WR-01 lockstep rule in dependabot.yml; the weekly machinery demonstrably fires — scheduled vulncheck run 41466437895, event schedule, success, 2026-11-02, verifier-observed). Ledger item 1 now fixed (open_count 0, verifier-run windows status); UAT complete 3/3"
    - "Stale-report regeneration trigger: PR #28 (307f252, merged 2026-11-02T15:34:33Z) superseded the internal/app walking skeleton (doc.go + app_test.go deleted) with the phase-2 engine packages and updated AGENTS.md Layout — changed covered inputs invalidating the prior fingerprint"
    - "Concurrent-process note: a prior regeneration pass (commit 821338a, verified 2026-11-02T16:07:36Z, passed 25/25) landed while this verification was running. This verifier did NOT trust it — every predicate below was independently re-derived against the current tree in this pass (local gates re-run, probes re-fired, live gh state re-queried, vendored config re-diffed against a fresh upstream fetch), and the covered-input fingerprint was recomputed (reproducing the same 43-file digest, confirming covered inputs are unchanged since that pass)"
  gaps_remaining: []
  regressions: []
overrides:
  - must_have: "Direct pushes to master are rejected by branch protection; a red gate blocks the merge"
    reason: "Rulesets API 403 on private org repo (plan-gated); payload + one-command re-apply recorded in docs/setup/branch-protection.md; defer decision taken 2026-10-31 at the plan 01-04 blocking checkpoint ; ratified by the maintainer as an accepted override 2026-10-31 (no org plan upgrade, no public flip)"
    accepted_by: "onokonem"
    accepted_at: "2026-10-31T18:41:25Z"
  - must_have: "Dependabot is active (updates and alerts)"
    reason: "Alerts GHAS-gated on private repo tier; version updates + govulncheck delivered and green; alerts unlock at the deferred public flip ; ratified by the maintainer as an accepted override 2026-10-31."
    accepted_by: "onokonem"
    accepted_at: "2026-10-31T18:41:25Z"
  - must_have: "Dependabot opens its first version-update PR within the weekly schedule window"
    reason: "Window elapsed (Mon 2026-11-02 06:00 UTC cron) and no version-update PR exists because nothing is updatable: golangci-lint v2.15.0 is the latest upstream release (verifier-confirmed live via the GitHub releases API this pass) and is deliberately ignored in dependabot.yml (WR-01 — must move in lockstep with the vendored 2.15.0-exact strict config); the weekly machinery demonstrably fires — the vulncheck cron ran on schedule 2026-11-02 with conclusion success (verifier-observed run 41466437895). Maintainer decision 2026-11-02 (UAT Test 1, option A): the backstop's substance — Dependabot wired and healthy — is satisfied; a version-update PR will appear when an upstream release actually exists."
    accepted_by: "onokonem"
    accepted_at: "2026-11-02T15:55:45Z"
advisory:
  - finding: "WR-01 (2026-11-02 review increment): driftBetween overflows int64 at the HLC domain ceiling (~292-year delta), corrupting the drift signal exactly when the clock is pathological (internal/app/clock/clock.go:144)"
    category: other
    reason: "Phase-2 engine code that landed inside this phase's covered paths via PR #28; no failing test evidences it (01-REVIEW-DISPOSITION.md records it open, severity warning); phase-2 verification owns the clock contract. Resolution: triage in phase-2 scope"
    evidence_status: "none provided"
  - finding: "WR-02 (2026-11-02 review increment): NewClock never clamps initialL to the 48-bit domain — the one unclamped input silently defeats the documented wrap protection (internal/app/clock/clock.go:63-65)"
    category: other
    reason: "Same class as WR-01: phase-2 clock code, no deterministic evidence, tracked open in 01-REVIEW-DISPOSITION.md. Resolution: triage in phase-2 scope (VERS-02/VERS-03 contract, not any Phase 1 truth)"
    evidence_status: "none provided"
---

# Phase 1: CI Bootstrap and Repo Skeleton Verification Report

**Phase Goal:** Quality gates and the enforced repo skeleton exist before any feature code, so every subsequent change lands inside the architecture and behind a red-light-blocking gate
**Verified:** 2026-11-02T16:20:24Z
**Status:** passed
**Re-verification:** Yes — sanctioned regeneration of a stale report (prior pass human_needed 24/25, verified 2026-10-31T22:56:10Z; invalidated by PR #28 changing covered inputs: internal/app skeleton superseded by the phase-2 engine, AGENTS.md Layout prose updated)

## Regeneration Scope

Covered-input diff since the prior verified timestamp (git log --since 2026-10-31T22:56:10Z): exactly
one implementation-side commit — 307f252 (PR #28, merged 2026-11-02T15:34:33Z) — which deleted the
`internal/app` walking skeleton (`doc.go`, `app_test.go`) in favor of the real engine packages
(`internal/app/{version,clock,storage}` + `internal/app/internal/testutil`), rewrote AGENTS.md's
Layout bullet (deviations and conventions untouched — re-read this pass), and touched
REQUIREMENTS.md (phase-2 rows only; phase-1 rows intact) and PROJECT.md (transition commit 5609f2e;
the deferral parenthetical survived). Every other implementation covered file is UNCHANGED since the
prior pass (verifier-run git log --since over the covered set): ci.yml, Makefile, .golangci.yml,
vulncheck.yml, dependabot.yml, ruleset.json, branch-protection.md, go.mod/go.sum,
cmd/sup-kv/main.go, internal/port/doc.go. The superseded skeleton paths were re-verified at full
level through their replacements: the same `internal/app` prefix now carries the real engine, and
every gate (build, vet, race, strict lint, both depguard red probes, vendored-config pristine diff)
was re-run live by this verifier on the phase-2 tree. Nothing was trusted from SUMMARYs or from the
concurrently-landed prior regeneration (821338a) — see the re_verification note. The prior report's
one open human item (Dependabot backstop) is closed by a recorded maintainer decision with every
factual predicate re-verified live (truth 17).

Working-tree observations (planning-side, uncommitted; the orchestrator handles commits):
WINDOWS.md carries the recorded `windows fixed 1` ledger update (item 1 → fixed, open_count 0) and
STATE.md carries a concurrent workflow's progress-metadata reset ("Phase 01 executing") — the
G8 decision-log entries survive in it, and both files were hashed in this state by the fingerprint
(reproducing the prior pass's digest, so nothing moved since).

## Goal Achievement

### Observable Truths

Same 25-truth contract as the prior report (4 ROADMAP success criteria split into
independently-verifiable halves + deduplicated PLAN truths + the 8 gap-closure recording truths
G1-G8). Skeleton-scope truths are read through their phase-2 supersession: the paths still carry
compile/vet/race/lint targets, now the real engine instead of the placeholder. All evidence below
was collected by this verifier on this pass.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC-1a: A PR runs the full gate matrix (strict lint, go vet, race tests) across Linux/macOS × amd64/arm64 | ✓ VERIFIED | ci.yml re-read in full this pass: push(master)+pull_request triggers, permissions contents:read, 4 cells (ubuntu-latest, ubuntu-24.04-arm, macos-15 native vet+race+tidy; darwin-amd64-cross GOOS/GOARCH build+vet), lint job prefixed by the standing `go tool golangci-lint config verify` step. Verifier-run `gh api commits/307f252/check-runs` (origin/master HEAD = PR #28 squash): exactly 5 check-runs — lint, gate (ubuntu-latest), gate (ubuntu-24.04-arm), gate (macos-15), gate (darwin-amd64-cross) — all success; PR #28 MERGED 2026-11-02T15:34:33Z |
| 2 | SC-1b: A red gate blocks the merge | ✓ PASSED (override) | Red path proven twice over: historical probe PR #4 (lint failed citing sup-kv-app-no-io, CLOSED unmerged) and this pass's live re-fired red probes (truth 7). Server-side merge-blocking absent by ratified override entry 1 (accepted_by onokonem, 2026-10-31T18:41:25Z) |
| 3 | SC-2a: Direct pushes to master are rejected by branch protection | ✓ PASSED (override) | Verifier re-ran `gh api repos/Djarvur/sup-kv/rulesets` this pass → 403 "Upgrade to GitHub Pro or make this repository public" — the override's factual predicate still holds (repo private=true, default=master confirmed via repo API). Deviation ratified via override entry 1; all markers re-verified this pass (REQUIREMENTS.md CI-03 suffix, branch-protection.md DEFERRED heading, WINDOWS.md item 2 waived, STATE.md decision log) |
| 4 | SC-2b: Every change (code and docs) lands through a pull request | ✓ VERIFIED | Verifier-run first-parent analysis of origin/master: every implementation change (internal/, cmd/, .github/, Makefile, .golangci.yml, go.mod) landed through PRs #1-#28 (merge commits or squash-suffixed); the only first-parent non-PR commits are the gsd tooling's own .planning/docs record-keeping commits plus the pre-phase repo seed (b82514b, .gitignore/LICENSE/README only). Exception class disclosed and carried as a Warning: 4 such docs commits (0598747, adddd0b, ff6e24f, 821338a) currently sit unpushed on local master — see Anti-Patterns |
| 5 | SC-3a: Dependabot is active (updates and alerts) | ✓ PASSED (override) | Updates: dependabot.yml re-read this pass (version 2, gomod + github-actions at directory /, weekly, grouped minor+patch, prefixes go-deps/ci, WR-01 ignore for golangci-lint/v2 with the lockstep justification). Alerts: GHAS-gated off — ratified override entry 2; REQUIREMENTS.md CI-02 honestly Partial/unchecked with the ratified annotation (re-read this pass) |
| 6 | SC-3b: The scheduled govulncheck job runs green against the Go vulnerability database | ✓ VERIFIED | Verifier-run `gh run list --workflow=vulncheck.yml`: run 41466437895, event **schedule**, conclusion **success**, created 2026-11-02T14:54:44Z (first scheduled cron occurrence observed green — GitHub may delay scheduled starts past the 06:00 cron), plus dispatch run 41216077968 success 2026-10-31. vulncheck.yml re-read: cron "0 6 * * 1", go.mod/go.sum paths trigger, workflow_dispatch, `go tool govulncheck ./...` |
| 7 | SC-4: Dependency boundary machine-enforced (internal/app importing net/os/grpc, internal/port beyond stdlib, fails the gate) | ✓ VERIFIED | Verifier re-fired BOTH red probes this pass on the phase-2 engine layout: `net` import in internal/app/clock → `import 'net' is not allowed from list 'sup-kv-app-no-io'` (depguard); `golang.org/x/mod/semver` in internal/port → `not allowed from list 'sup-kv-port-stdlib-only'` (exit 1). Probes removed; full gate re-ran green; `git status internal/ cmd/` clean; `git diff --exit-code go.mod go.sum` clean |
| 8 | .golangci.yml is the vendored powerman 2.15.0 config byte-unmodified with exactly two additive sup-kv rules | ✓ VERIFIED | STRENGTHENED this pass: verifier fetched the pristine upstream config live (raw.githubusercontent.com, 511 lines) and diffed — single hunk 64a65,88 (the two sup-kv rules + banner), zero upstream lines deleted or changed, diff exit clean; Origin header line 1 asserted; exactly one occurrence each of sup-kv-app-no-io / sup-kv-port-stdlib-only; file unchanged since the prior pass |
| 9 | go.mod pins golangci-lint v2.15.0 + govulncheck v1.9.0 as tool directives with zero direct runtime requires | ✓ VERIFIED | Tool block re-read (both directives present); awk scan re-run this pass: direct_count=0 — every require line carries `// indirect`; file unchanged since prior pass |
| 10 | The strict lint gate is live and green; `config verify` exits 0 (QUAL-01) — a STANDING gate in Makefile and CI | ✓ VERIFIED | Verifier-run this pass: `make lint` → exit 0 (target runs `go tool golangci-lint config verify` THEN `run ./...`; both green on the phase-2 code). T-01-01 teeth re-proven in /tmp this pass: `run --config <copy with bogus top-level key>` exits 0 (silent no-op — the threat class is real) while `config verify` on the same file exits 3 naming `bogus_top_level_key_that_noops`; repo config untouched |
| 11 | The race gate is not vacuous (CI reports ok for internal/app) | ✓ VERIFIED | Verifier-run with cache bypass: `go test -race -count=1 ./...` → ok internal/app/clock 1.035s, internal/app/storage 1.596s, internal/app/version 1.017s (2214 test lines — the skeleton's single TestAppSkeleton superseded by three real suites, a strictly stronger race target); build + vet green; matrix race step green live on 307f252 |
| 12 | Matrix cells are independent (fail-fast disabled) | ✓ VERIFIED | ci.yml re-read: `fail-fast: false` with the rationale comment; unchanged since prior pass |
| 13 | Push and PR events for the same ref do not cancel each other; superseded runs cancel | ✓ VERIFIED | ci.yml re-read: concurrency group `ci-${{ github.event_name }}-${{ github.ref }}` (event name in the key), `cancel-in-progress: true`; unchanged since prior pass |
| 14 | docs/setup/ruleset.json encodes the full master protection payload with contexts matching the check-run names verbatim | ✓ VERIFIED | Verifier-run python compare against live check-runs on 307f252: 5 contexts == 5 live check-run names, match True character-for-character; rule types [pull_request, required_status_checks, non_fast_forward, deletion]; strict_required_status_checks_policy true; enforcement active; conditions ref_name exactly refs/heads/master |
| 15 | delete_branch_on_merge enabled on the repository | ✓ VERIFIED | Verifier-run `gh api repos/Djarvur/sup-kv` → delete_branch_on_merge=true, default_branch=master, private=true |
| 16 | docs/setup/branch-protection.md records apply/verify/re-apply commands, zero-approval rationale, context-coupling rule, and the current ratified state | ✓ VERIFIED | Re-greped this pass: `## Status: DEFERRED (accepted override)` exactly once, zero PENDING occurrences, `Deferral decision 2026-10-31` line once, `Ratified 2026-10-31: accepted override` paragraph present, POST apply command exactly once; file unchanged since prior pass |
| 17 | Backstop: Dependabot opens its first version-update PR within the weekly window | ✓ PASSED (override) | Window ELAPSED (Mon 2026-11-02 06:00 UTC). Verifier-run live: `gh pr list --author app/dependabot --state all` → 0; `gh api repos/golangci/golangci-lint/releases/latest` → **v2.15.0** == the pin (nothing updatable — and deliberately ignored per the WR-01 lockstep rule); weekly machinery proven firing by scheduled vulncheck run 41466437895 (event schedule, success). Maintainer decision 2026-11-02 (UAT Test 1, option A, result pass) → override entry 3; WINDOWS.md item 1 fixed (open_count 0, verifier-run windows status); UAT complete 3/3 |
| G1 | (01-05 T1) overrides: frontmatter carries the ratified entries verbatim from the prior report's own suggestion block, attributed accepted_by "onokonem" with real UTC accepted_at | ✓ VERIFIED | Entries 1-2 carried in this report's frontmatter verbatim (accepted_at "2026-10-31T18:41:25Z"); the exactly-two condition held at the 01-05 gap-closure moment (PR #17 diff evidence stands in history); the living report additionally carries entry 3 — a sanctioned post-UAT addition recording the 2026-11-02 maintainer decision, attributed the same way |
| G2 | (01-05 T2) Gap-closure edits left the report byte-unchanged outside the overrides block | ✓ VERIFIED | Historical: PR #17 diff showed the overrides block as the only change. The body was since legitimately rewritten by verifier action (PR #23, the 2026-10-31 regeneration, the concurrent 821338a regeneration, and this sanctioned regeneration) — the verifier's own record-keeping anticipated by plan 01-05's verification section, not tampering; verifier-computed keys recomputed each time (this pass's fingerprint reproduces the same digest) |
| G3 | (01-05 T3) REQUIREMENTS.md CI-02 annotated Partial: checkbox unchecked, row reads Partial with the ratified alerts deferral | ✓ VERIFIED | Re-read this pass: `- [ ] **CI-02**` bullet; traceability row "Partial — version updates + scheduled govulncheck delivered and green; alerts half deferred (GHAS-gated on private tier), ratified accepted override 2026-10-31; unlocks at GHAS license or the deferred public flip" intact; CI-01/QUAL-01 rows Complete and untouched |
| G4 | (01-05 T4) REQUIREMENTS.md CI-03 row records the ratification of its deferral | ✓ VERIFIED | Re-read this pass: row carries "ratified 2026-10-31 as accepted override (no org plan upgrade, no public flip)" after decision `defer`, re-apply path mention intact |
| G5 | (01-06 T1) branch-protection.md status states DEFERRED (accepted override) with ratification paragraph, decision line and every command verbatim | ✓ VERIFIED | Re-greped this pass (see truth 16): heading + paragraph + decision line + POST/verify/PATCH trail all present exactly once; zero pending-titled headings; file unchanged since prior pass |
| G6 | (01-06 T2) PROJECT.md and .claude/CLAUDE.md Process constraints no longer assert server-side enforcement — identical deferral parenthetical in both (IN-02 closed at source) | ✓ VERIFIED | Re-verified this pass on the post-5609f2e file: `server-side ruleset deferred 2026-10-31` exactly once in each; diff of the extracted parentheticals → character-identical (mirror == source); 01-REVIEW-DISPOSITION records IN-02 fixed |
| G7 | (01-06 T3) WINDOWS.md item 2 waived with the override reason; ledger consistent; item 1 resolved | ✓ VERIFIED | Verifier-run `gsd-tools windows status` this pass: open_count 0, waived_count 1, fixed_count 1 — item 1 status **fixed** (resolved_at 2026-11-02T15:55:45Z, the recorded `windows fixed 1` after the UAT decision); item 2 status waived with the ratified-override reason naming the 403 and the re-apply path |
| G8 | (01-06 T4) STATE.md decision log carries one entry per ratified override, citing 2026-10-31 and gap-plan provenance | ✓ VERIFIED | Re-greped this pass in both HEAD and the (concurrently modified) working tree: exactly one "Verification gap 1 ratified 2026-10-31" and one "Verification gap 2 ratified 2026-10-31" entry, plus the plan 01-05 provenance entry — the decision log survived both the PR #28 rewrite and the working-tree state reset |

**Score:** 25/25 truths verified (4 via maintainer-ratified overrides; 0 present-but-behavior-unverified)

### Override Ratification Audit (Step 3b)

Three maintainer-ratified overrides carry in the frontmatter. Entries 1-2 (accepted_by "onokonem",
accepted_at "2026-10-31T18:41:25Z") carry forward verbatim per the regeneration briefing — their
factual predicates re-verified live this pass: the rulesets endpoint still returns 403 and the repo
is still private on the plan-gated tier. Entry 3 records the 2026-11-02 maintainer decision (UAT
Test 1, option A, result pass) that resolved the last open backstop; every factual predicate in its
reason was re-verified live by this verifier (0 dependabot PRs; golangci-lint latest release =
v2.15.0; scheduled vulncheck cron fired and green). Token-overlap matching maps entry 1 to truths
2+3 (SC-1b + SC-2a), entry 2 to truth 5 (SC-3a), and entry 3 to truth 17: 4 truths PASSED
(override). Every marker the gap-closure plans established still tells the same one story
(REQUIREMENTS rows, DEFERRED status, ledger waiver + fix, decision log, constraint parentheticals —
all re-verified this pass, truths G3-G8).

### Deferred Items

None — no Phase 1 gap is scheduled into a later milestone phase; the original two gaps are closed by
ratified override (entries 1-2), and the backstop by override entry 3. Verifier grep over the
Phase 3-7 roadmap sections: none mentions rulesets, branch protection, GHAS, or Dependabot alerts —
the deferred halves (ruleset application, GHAS alerts) remain one-command /
one-billing-decision away, tracked in WINDOWS.md (waived) and docs/setup/, not in a later phase.

### Advisory (New Scope, Unevidenced)

Re-verification ran; Step 7's scan of all files changed since the prior verification (the phase-2
engine files under internal/app/, AGENTS.md, REQUIREMENTS.md, PROJECT.md) found zero Step-7
blockers, zero unreferenced debt markers, zero stub patterns, and zero unjustified suppressions.
Two WARNING-severity findings from the 2026-11-02 code-review increment flag phase-2 clock code
that landed inside this phase's covered paths via PR #28; both are reported as advisory — phase-2
scope (no Phase 1 truth touches clock arithmetic), evidenced by no failing test, and both are
recorded open in 01-REVIEW-DISPOSITION.md for phase-2 triage. Per the convergence evidence gate
they do not revert a completed must-have: the clock code is phase 2's completed, separately
verified work entering this report's covered set by path supersession, not a phase-1 gap-closure
product.

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| 1 | WR-01: driftBetween int64 overflow at the HLC domain ceiling (internal/app/clock/clock.go:144) | other | phase-2 code via PR #28, no deterministic evidence; tracked open in 01-REVIEW-DISPOSITION |
| 2 | WR-02: NewClock does not clamp initialL to the 48-bit domain (internal/app/clock/clock.go:63-65) | other | phase-2 code, no deterministic evidence; tracked open in 01-REVIEW-DISPOSITION |

### Required Artifacts

gsd-tools `verify.artifacts`: 18/20 pass across all 6 plans. The 2 reported misses are the superseded
walking-skeleton paths — `internal/app/doc.go` and `internal/app/app_test.go` (plan 01-01
declarations), deleted by PR #28 in favor of the real engine. Their declared purpose — doc-commented
packages under internal/app and a real race-test target (D-01) — is fulfilled by the replacements
and was verified live this pass: strict lint green (package-comments enforced by the gate —
storage carries its own doc.go), `go test -race -count=1` ok for internal/app/{clock,storage,version}
with 2214 test lines. Supersession, not a gap.

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| go.mod | module + tool directives + go 1.27 | ✓ VERIFIED | unchanged; tool block + direct_count=0 re-verified |
| cmd/sup-kv/main.go | Command doc comment, minimal main | ✓ VERIFIED | unchanged since prior pass |
| internal/port/doc.go | Package comment, stdlib-only target | ✓ VERIFIED | unchanged; port red probe re-fired this pass |
| internal/app/doc.go | Package comment, pure-core target | ✓ SUPERSEDED (purpose fulfilled) | deleted by PR #28; internal/app packages documented per-package (storage/doc.go); lint's package-comments gate green |
| internal/app/app_test.go | Real race target (D-01) | ✓ SUPERSEDED (purpose fulfilled) | deleted by PR #28; replaced by 2214 lines of real tests in clock/storage/version — ok under -race this pass |
| .golangci.yml | Vendored strict config + boundary | ✓ VERIFIED | unchanged; LIVE pristine-diff vs fresh upstream fetch this pass: single pure insertion, zero upstream lines touched; boundary probes re-fired red |
| Makefile | lint/test/generate/build via go tool | ✓ VERIFIED | unchanged; `make lint` verifier-run exit 0 (verify + run) |
| AGENTS.md | Conventions + 3 deviations + D-03 reading | ✓ VERIFIED | CHANGED (PR #28): Layout bullet now describes version/clock/storage + testutil (matches the tree, verifier-listed); all three deviations, the D-03 reading, and the tool-invocation rule byte-intact (re-read in full this pass) |
| .github/workflows/ci.yml | lint + 4-cell matrix + hygiene | ✓ VERIFIED | unchanged; re-read in full; 5/5 live check-runs green on 307f252 |
| .github/workflows/vulncheck.yml | cron + paths trigger + dispatch, pinned scan | ✓ VERIFIED | unchanged; scheduled run observed green 2026-11-02 |
| .github/dependabot.yml | gomod + github-actions updates | ✓ VERIFIED | unchanged; WR-01 ignore with lockstep justification intact |
| docs/setup/ruleset.json | Ruleset payload, 4 rule types, 5 contexts | ✓ VERIFIED | unchanged; contexts == live check-run names re-confirmed this pass (python, match True 5/5) |
| docs/setup/branch-protection.md | Apply/verify/re-apply + ratified DEFERRED marker | ✓ VERIFIED | unchanged; markers re-greped |
| .planning/REQUIREMENTS.md | CI-02 Partial annotation + CI-03 ratification suffix | ✓ VERIFIED | re-read post-PR #28; phase-1 rows intact |
| .planning/WINDOWS.md | ledger item 2 waived; item 1 fixed | ✓ VERIFIED | open_count 0 / waived 1 / fixed 1 (tooling + file) |
| .planning/STATE.md | decision-log entries for both ratified overrides | ✓ VERIFIED | exactly two gap-ratification entries (survived PR #28 rewrite and the working-tree reset) |
| .planning/PROJECT.md | Process constraint stating the deferral honestly | ✓ VERIFIED | parenthetical present exactly once (post-5609f2e) |
| .claude/CLAUDE.md | generated mirror in sync (IN-02 closed) | ✓ VERIFIED | parenthetical character-identical to source |

### Key Link Verification

gsd-tools `verify.key-links`: 9/13 automated-verified; the 4 "failures" are all the known tooling
limitation (descriptive `from:` fields that are not relative file paths — the tool reports "from:
must be a relative file path"); each was verified manually with direct evidence this pass.

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| Makefile | go.mod | `go tool golangci-lint` | ✓ WIRED | Automated (01-01: 3/3) |
| .golangci.yml | internal/app | `**/internal/app/**` deny rule | ✓ WIRED | Automated; red probe re-fired this pass on the phase-2 subtree |
| .golangci.yml | internal/port | `**/internal/port/**` strict list-mode | ✓ WIRED | Automated; red probe re-fired this pass |
| ci.yml | go.mod | `version: v2.15.0` + `go-version-file` | ✓ WIRED | Automated |
| ci.yml | .golangci.yml | lint job runs vendored config + standing verify step | ✓ WIRED | Automated |
| ci.yml (gate display names) | docs/setup/ruleset.json | contexts verbatim | ✓ WIRED (manual) | Verifier-run python compare this pass: 5 contexts == 5 live check-run names on 307f252, match True |
| vulncheck.yml | go.mod | `go tool govulncheck` | ✓ WIRED | Automated (01-03: 2/2) |
| dependabot.yml | repo root | `directory: /` both ecosystems | ✓ WIRED | Automated |
| ruleset.json | ci.yml check-run names | contexts == display names | ✓ WIRED | Automated (01-04: 2/2) |
| branch-protection.md | GitHub rulesets API | re-runnable gh api commands | ✓ WIRED | Automated; GET re-run live this pass (403 as recorded) |
| 01-VERIFICATION.md overrides entries | ratified override state | verbatim + ratification clause | ✓ WIRED (manual) | Entries 1-2 carried verbatim; entry 3 records the UAT maintainer decision with every predicate re-verified live; all markers re-verified |
| REQUIREMENTS.md CI-02 row | CI-03 row | same Partial-with-explanation pattern | ✓ WIRED (manual) | Both rows re-read this pass — same shape |
| .claude/CLAUDE.md Process line | PROJECT.md Process constraint | identical parenthetical | ✓ WIRED (manual) | diff of extracted parentheticals → identical |
| WINDOWS.md item 2 reason | ratified override 2026-10-31 | reason names ratification + re-apply path | ✓ WIRED (manual) | Ledger reason quotes both (windows status output read this pass) |

### Data-Flow Trace (Level 4)

No UI/rendered data in this phase. Infrastructure equivalent re-confirmed: lint job ←
.golangci.yml (schema-verified twice — the standing `config verify` step re-proven this pass to
catch a bogus key that `run` silently ignores, plus the action's own verify); setup-go ← go.mod
(go-version-file); govulncheck ← go.mod tool graph (pinned v1.9.0); check-runs ← real matrix
execution (5/5 success observed live on origin/master HEAD 307f252); ledger counters ← gsd-tools
windows status reading the actual file (open 0 / waived 1 / fixed 1). No static/hollow values.

### Behavioral Spot-Checks

All commands verifier-run in a fresh shell this pass; probe files created and removed by the
verifier; trees verified clean after.

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Build + vet + race tests (cache bypassed) | `go build ./... && go vet ./... && go test -race -count=1 ./...` | exit 0; ok clock 1.035s / storage 1.596s / version 1.017s | ✓ PASS |
| Full local gate incl. standing verify | `make lint` | exit 0 (config verify + run ./... both green) | ✓ PASS |
| Config schema-valid (repo) | `go tool golangci-lint config verify` (via make lint) | exit 0 | ✓ PASS |
| T-01-01 teeth: run silently ignores bogus key | `run --config /tmp/copy-with-bogus-key` | exit 0 — the threat class is real | ✓ PASS (as predicted) |
| T-01-01 teeth: config verify catches bogus key | `config verify --config /tmp/copy-with-bogus-key` | exit 3: "additional properties 'bogus_top_level_key_that_noops' not allowed" | ✓ PASS |
| Depguard RED — net in internal/app (non-test file) | probe + lint run | `import 'net' is not allowed from list 'sup-kv-app-no-io'` | ✓ PASS (red as required) |
| Depguard RED — third-party in internal/port | probe + lint run | `not allowed from list 'sup-kv-port-stdlib-only'` | ✓ PASS (red as required) |
| Gate GREEN after probe removal | `go tool golangci-lint run ./...` + `git status internal/ cmd/` + `git diff go.mod go.sum` | exit 0; tree clean; no module churn | ✓ PASS |
| Vendored config pristine vs upstream | curl upstream 2.15.0 config + diff | single hunk 64a65,88 (pure insertion), zero `<` lines | ✓ PASS |
| Matrix green on current origin/master HEAD | `gh api commits/307f252/check-runs` | 5 names, 5× success | ✓ PASS |
| PR #28 (the stale-making change) | `gh pr view 28` | MERGED 2026-11-02T15:34:33Z | ✓ PASS |
| Scheduled vulncheck fired green | `gh run list --workflow=vulncheck.yml` | run 41466437895: event schedule, success, 2026-11-02 (+ dispatch 41216077968 success) | ✓ PASS |
| Dependabot PR inventory | `gh pr list --author app/dependabot --state all` | 0 — nothing updatable (see override 3) | ✓ PASS (state as recorded) |
| golangci-lint upstream latest | `gh api repos/golangci/golangci-lint/releases/latest` | v2.15.0 == the pin | ✓ PASS |
| Rulesets endpoint state | `gh api repos/Djarvur/sup-kv/rulesets` | 403 plan-gated — matches the ratified override's predicate | ✓ PASS (state as recorded) |
| delete_branch_on_merge | `gh api repos/Djarvur/sup-kv` | true (default_branch master, private true) | ✓ PASS |
| Ruleset contexts == live check-run names | python3 compare vs check-runs on 307f252 | match: True (5/5); strict policy true | ✓ PASS |
| Commit hash validity | `gsd-tools query verify.commits` (13 hashes: 307f252, 0598747, 821338a, ff6e24f, adddd0b, 60db0fb, 0ca46c9, 8c33bc0, 515cf2f, 0dce266, ae09dc7, 4a2e173, acf1215) | all_valid: true | ✓ PASS |
| Windows ledger state | `gsd-tools windows status` | open_count 0, waived_count 1, fixed_count 1 | ✓ PASS |
| Non-PR commits never touch code | first-parent file scan on origin/master | only .planning//docs/ paths (plus the pre-phase seed commit) | ✓ PASS |
| Decision coverage | `gsd-tools query check.decision-coverage-verify` | 14/14 honored, 0 not honored | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist (no scripts directory). The phase's probes are inline
lint runs — executed directly by this verifier (Behavioral Spot-Checks rows 3-7), re-fired on the
phase-2 tree because internal/ changed since the prior pass. SUMMARY probe/gate claims were
reproduced first-hand via file reads, live gh queries, and tooling queries, not trusted.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| QUAL-01 | 01-01 | Strict golangci-lint config + go-ultimate conventions govern every package; red gate blocks merge | ✓ SATISFIED | `make lint` verifier-run green on the full current tree (standing config verify in Makefile + CI); vendored config byte-unmodified (LIVE pristine diff this pass); boundary red/green re-proven both lists; every one of the 7 `//nolint` suppressions names its linter and carries a written justification (D-12, all in testutil); merge-blocking half subject to the ratified override (CI-03 scope) |
| CI-01 | 01-01, 01-02 | Actions runs quality gates on every push and PR, Linux/macOS, amd64/arm64 | ✓ SATISFIED | push+PR triggers, 4 cells re-read; 5/5 green live on origin/master HEAD 307f252 (verifier-run); standing config-verify step in the lint job |
| CI-02 | 01-03, 01-05 | Dependabot current (updates and alerts) + scheduled govulncheck green | ✓ SATISFIED-BY-RATIFICATION (Partial) | Updates configured + live (WR-01 ignore hardening intact); scheduled govulncheck observed green on its first cron occurrence (2026-11-02, verifier-run); alerts half deferred under ratified override entry 2; REQUIREMENTS.md honestly Partial/unchecked; first-PR backstop resolved by override entry 3 (maintainer decision 2026-11-02, UAT complete 3/3) |
| CI-03 | 01-04, 01-05, 01-06 | PR-only development — branch protection on master, direct pushes forbidden | ✓ SATISFIED-BY-RATIFICATION (Partial) | PR-only practiced and verified (every implementation change through PRs #1-#28, first-parent scan); server-side protection deferred under ratified override entry 1 (403 predicate re-verified live); payload + one-command re-apply scripted and verbatim-preserved; DEFERRED marker + ledger waiver + decision log all in place |

Orphaned requirements: none — REQUIREMENTS.md maps exactly CI-01, CI-02, CI-03, QUAL-01 to
Phase 1; all four appear in plan frontmatter requirements (01-05/01-06 claim CI-02/CI-03 for the
ratification recording with requirements-completed deliberately empty — honest, not an omission).

### Anti-Patterns Found

Debt-marker gate over all files changed since the prior verification (internal/app engine files,
AGENTS.md, REQUIREMENTS.md, PROJECT.md, planning docs): zero unreferenced TBD/FIXME/XXX in
implementation files. The single vendored FIXME remains .golangci.yml line 155 (upstream powerman
content the byte-unmodified prohibition forbids touching). ROADMAP.md `**Plans**: TBD` lines for
unplanned phases remain the planning template's honest state (each names its own formal follow-up
phase). No stubs (internal/port/doc.go's "skeleton placeholder" line is the package's honest doc
comment — port interfaces arrive in later phases by design), no skipped tests (0 t.Skip /
testing.Short matches), every `//nolint` justified (7/7, D-12 compliant).

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| local master (0598747, adddd0b, ff6e24f, 821338a) | — | 4 unpushed docs commits on local master ahead of origin, outside the PR flow | ⚠️ Warning | Carried-forward class (prior rounds flagged the identical condition); the gsd tooling's own docs commits are its sanctioned record-keeping mode, but pushing master directly extends the disclosed exception — route or push consciously |
| .planning/STATE.md, .planning/WINDOWS.md | — | Uncommitted working-tree edits: the recorded `windows fixed 1` ledger fix (load-bearing for G7/override 3) and a concurrent workflow's STATE progress reset ("Phase 01 executing" — contradicts the completed phases 01-02) | ⚠️ Warning | Planning bookkeeping only; decision-log entries survive; commit or reconcile deliberately |
| internal/app/clock/clock.go | 144 | WR-01 (2026-11-02 review increment): driftBetween int64 overflow at domain ceiling | ⚠️ Warning | Phase-2 scope, no failing test; tracked open in 01-REVIEW-DISPOSITION (see Advisory) |
| internal/app/clock/clock.go | 63-65 | WR-02 (2026-11-02 review increment): initialL not clamped to 48-bit domain | ⚠️ Warning | Phase-2 scope, no failing test; tracked open in 01-REVIEW-DISPOSITION (see Advisory) |
| .planning/ROADMAP.md | 63+ | `**Plans**: TBD` for unplanned phases | ℹ️ Info | Planning-template placeholders naming their own follow-up phase |
| .golangci.yml | 155 | upstream FIXME (vendored, byte-unmodified rule) | ℹ️ Info | Carried forward; touching it would violate the stronger prohibition |
| AGENTS.md | 5 | IN-01: quotes `go 1.27`, go.mod pins 1.27.1 | ℹ️ Info | Open; harmless self-referential pointer |
| ci.yml / vulncheck.yml | — | IN-03: actions pinned by mutable major tags | ℹ️ Info | Open hardening advice (bounded by contents:read, no secrets, no pull_request_target — re-read this pass) |
| internal/app/internal/testutil | — | IN-01 (review increment): NewRNG doc comment overstates the printed-seed contract | ℹ️ Info | Open review Info on phase-2 code; tracked in 01-REVIEW-DISPOSITION |
| docs/setup/branch-protection.md | — | IN-05: "Apply command above" direction nit | ℹ️ Info | Open review Info; doc contains exactly one Apply command |
| ci.yml lint job | — | IN-06: schema gate and lint run separately provisioned, both v2.15.0 | ℹ️ Info | Version parity holds (verifier-checked both pins this pass); WR-01 ignore guards drift |

IN-02: FIXED (source + mirror parentheticals verified identical this pass; disposition row
consistent). WR-01 (dependabot ignore): FIXED (config re-read this pass). The review increment's
WR-01/WR-02 above reuse IDs from the review's own increment sequence — distinct findings, correctly
tracked in 01-REVIEW-DISPOSITION.md.

### Test Quality Audit

The phase's original race target (app_test.go/TestAppSkeleton) is superseded; the current
requirement-linked test files (internal/app/clock/{clock_test,hlc_prop_test}.go,
internal/app/storage/{store_test,merge_test}.go, internal/app/version/version_test.go) are active,
not skipped (verifier grep: 0 skip/Short matches), with no circular generation (testutil generators
are independent PCG-based constructors, not system-output re-runs — gen.go/rng.go re-read this
pass) and assertion level is value/property-level (seeded property suites + unit tables) —
appropriate and strong for the compile/liveness purpose phase 1 declared. Gate-teeth proofs are
lint-probe based (red/green), exercised directly this pass.

### Decision Coverage

14/14 phase CONTEXT.md decisions honored (0 not honored) — gsd-tools
`check.decision-coverage-verify` run this pass; D-03 (AGENTS.md dependency policy), D-08
(branch-protection.md audit trail), D-10/D-11/D-12 (.golangci.yml + AGENTS.md deviations) all
present in shipped artifacts.

## Human Verification Required

None remaining. Infrastructure/foundation phase — no user-facing UAT steps, and no
behavior-unverified truths. The prior report's single open human item (first Dependabot PR
observation) is closed: the observation window elapsed, the observation was made (no PR — nothing
updatable), and the maintainer recorded the resolution decision on 2026-11-02 (01-UAT.md Test 1,
result pass; UAT complete 3/3; WINDOWS.md ledger item 1 fixed, open_count 0). The 7 judgment-tier
prohibitions carry recorded maintainer concurrence (01-UAT.md Test 2: all 7 reviewed and accepted —
QUAL-01 ×2, CI-01 ×2, CI-02 ×2, CI-03), and this verifier's independent scan found no prohibition
violated: vendored config byte-unmodified (live pristine diff), every nolint justified and specific
(7/7), no gate-theater mechanism (the standing config-verify gate demonstrably catches the no-op
class — re-proven this pass), local/CI linter versions identical at v2.15.0, no runtime dependency
entered the graph (direct_count 0), no vulnerability suppressions in vulncheck.yml, no path-based
protection exemptions in ruleset.json.

## Gaps Summary

No gaps remain. The regeneration trigger — PR #28 superseding the walking skeleton with the real
engine inside this phase's covered paths — was itself the phase goal working as designed: the first
feature code landed through the PR gate (5/5 checks green, verifier-observed live on origin/master
HEAD), inside the depguard boundary (both red probes re-fired on the new tree with the exact
finding strings), with the strict gate green (make lint verifier-run, zero findings, config verify
standing). Every previously verified artifact that still exists is byte-unchanged since the prior
pass, and the vendored config was re-proven byte-unmodified against a fresh upstream fetch. The
superseded skeleton's purpose is fulfilled more strongly by its replacements (three real
race-tested packages, 2214 test lines, where one placeholder test stood). The two
maintainer-ratified overrides carry forward verbatim with their 403/GHAS predicates re-verified
live, and the last open backstop closed via a third recorded override whose every factual claim
this verifier re-confirmed independently (zero updatable dependencies — golangci-lint v2.15.0 is
upstream-latest; the weekly cron demonstrably fired on schedule and green). The phase closes at
25/25 with no open ledger items (open_count 0) and UAT complete 3/3. Two warning-severity review
findings on phase-2 clock code are reported as advisory and tracked open in 01-REVIEW-DISPOSITION
for phase-2 triage; they touch no Phase 1 truth.

---

_Verified: 2026-11-02T16:20:24Z_
_Verifier: Claude (gsd-verifier)_
