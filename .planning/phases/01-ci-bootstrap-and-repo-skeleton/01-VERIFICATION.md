---
phase: 01-ci-bootstrap-and-repo-skeleton
verified: 2026-11-02T16:07:36Z
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
    - "Dependabot first-PR backstop (prior truth 17, WINDOWS.md item 1, UAT item 1): window elapsed Mon 2026-11-02 and resolved by recorded maintainer decision 2026-11-02 (UAT Test 1, option A) — no version-update PR exists because nothing is updatable (golangci-lint v2.15.0 IS the latest upstream release, deliberately ignored per WR-01; weekly machinery proven firing by the on-schedule vulncheck cron run 2026-11-02, success). All three factual predicates re-verified live by this verifier this pass; ledger item 1 now fixed (open_count 0); UAT complete 3/3"
    - "Stale-report regeneration trigger: PR #28 (307f252, merged 2026-11-02T15:34:33Z) superseded the internal/app walking skeleton (doc.go + app_test.go deleted) with the phase-2 engine packages (internal/app/{version,clock,storage} + internal/app/internal/testutil) and updated AGENTS.md Layout prose — changed covered inputs invalidating the prior fingerprint"
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
    reason: "New-scope finding on phase-2 engine code that landed inside this phase's covered paths via PR #28; no failing test evidences it (the review's own disposition records it open, severity warning); phase-2 verification already closed the adjacent ceiling-saturation class (IN-02 fixed, TestClock_ceilingSaturation). Resolution: triage in phase-2 scope (01-REVIEW-DISPOSITION.md WR-01 open)"
    evidence_status: "none provided"
  - finding: "WR-02 (2026-11-02 review increment): NewClock never clamps initialL to the 48-bit domain — the one unclamped input silently defeats the documented wrap protection (internal/app/clock/clock.go:63-65)"
    category: other
    reason: "Same class as WR-01: new-scope phase-2 code, no deterministic evidence, tracked open in 01-REVIEW-DISPOSITION.md. Resolution: triage in phase-2 scope (belongs to the VERS-02/VERS-03 contract, not any Phase 1 truth)"
    evidence_status: "none provided"
---

# Phase 1: CI Bootstrap and Repo Skeleton Verification Report

**Phase Goal:** Quality gates and the enforced repo skeleton exist before any feature code, so every subsequent change lands inside the architecture and behind a red-light-blocking gate
**Verified:** 2026-11-02T16:07:36Z
**Status:** passed
**Re-verification:** Yes — sanctioned regeneration of a stale report (prior pass human_needed 24/25, verified 2026-10-31T22:56:10Z; invalidated by PR #28 changing covered inputs: internal/app skeleton superseded by the phase-2 engine, AGENTS.md Layout prose updated)

## Regeneration Scope

Covered-file diff since the prior verified timestamp (git log --since 2026-10-31T22:56:10Z): exactly
one covered-input commit — 307f252 (PR #28, merged 2026-11-02T15:34:33Z), which deleted the
`internal/app` walking skeleton (`doc.go`, `app_test.go`) in favor of the real engine packages
(`internal/app/{version,clock,storage}` + `internal/app/internal/testutil`), rewrote AGENTS.md's
Layout bullet to describe them (deviations and conventions untouched), and updated planning-side
REQUIREMENTS/ROADMAP/STATE. Every implementation covered file that still exists is UNCHANGED since
the prior pass (verifier-run git log --since over the whole covered set: empty except as above) —
ci.yml, Makefile, .golangci.yml, vulncheck.yml, dependabot.yml, ruleset.json, branch-protection.md,
PROJECT.md/.claude/CLAUDE.md parentheticals, go.mod/go.sum, cmd/sup-kv/main.go, internal/port/doc.go.
The superseded skeleton paths were re-verified at full level through their replacements: the same
`internal/app` prefix now carries the real engine, and every gate (build, vet, race, strict lint,
both depguard red probes) was re-run live by this verifier on the phase-2 tree. All evidence below
was collected by this verifier on this pass — live GitHub state and local gates, nothing trusted
from SUMMARYs. The prior report's one open human item (Dependabot backstop) is closed by a recorded
maintainer decision with every factual predicate re-verified live (see truth 17). An uncommitted
working-tree edit that had flipped the prior report's frontmatter status field (human_needed →
passed, no body change) is superseded by this sanctioned regeneration.

## Goal Achievement

### Observable Truths

Same 25-truth contract as the prior report (4 ROADMAP success criteria split into
independently-verifiable halves + deduplicated PLAN truths + the 8 gap-closure recording truths
G1-G8). Skeleton-scope truths are read through their phase-2 supersession: the paths still carry
compile/vet/race/lint targets, now the real engine instead of the placeholder.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC-1a: A PR runs the full gate matrix (strict lint, go vet, race tests) across Linux/macOS × amd64/arm64 | ✓ VERIFIED | CHANGED TREE, full re-verify: current ci.yml re-read (push(master)+pull_request, 4 cells, native cells run vet+race+tidy, cross cell GOOS/GOARCH build+vet, standing `config verify` step). Verifier-run `gh api commits/307f252/check-runs` (origin/master HEAD = PR #28): exactly 5 check-runs — lint, gate (ubuntu-latest), gate (ubuntu-24.04-arm), gate (macos-15), gate (darwin-amd64-cross) — all success. `gh pr checks 28`: 5/5 pass (lint 2m20s incl. the verify step), MERGED 2026-11-02T15:34:33Z |
| 2 | SC-1b: A red gate blocks the merge | ✓ PASSED (override) | Red path itself proven (historical probe PR #4: lint check failed citing sup-kv-app-no-io, PR CLOSED unmerged — live evidence stands; the boundary teeth re-proven red on the current tree this pass, truth 7). Server-side merge-blocking absent by ratified override entry 1 (accepted_by onokonem, 2026-10-31T18:41:25Z) |
| 3 | SC-2a: Direct pushes to master are rejected by branch protection | ✓ PASSED (override) | Verifier re-ran `gh api repos/Djarvur/sup-kv/rulesets` this pass → 403 "Upgrade to GitHub Pro or make this repository public" — the override's factual predicate still holds (repo: private=true, default=master confirmed via repo API). Deviation ratified via override entry 1; markers re-verified this pass (REQUIREMENTS.md CI-03 suffix, branch-protection.md DEFERRED heading, WINDOWS.md item 2 waived, STATE.md decision log) |
| 4 | SC-2b: Every change (code and docs) lands through a pull request | ✓ VERIFIED | All feature/code changes landed through PRs, verifier-checked through #28 (squash-merged with (#28) subject; merges #1-#27 in history). Exception class unchanged and disclosed: the gsd workflow's own planning-doc commits (60db0fb prior report landing, 0ca46c9 review docs, and 3 currently-unpushed local docs commits 0598747/adddd0b/ff6e24f) sit outside the PR flow — see Anti-Patterns (⚠️ Warning, carried forward) |
| 5 | SC-3a: Dependabot is active (updates and alerts) | ✓ PASSED (override) | Updates: config live and shape-verified this pass (gomod + github-actions, weekly, grouped, prefixes; WR-01 hardening ignore for golangci-lint/v2 intact with its lockstep justification). Alerts: GHAS-gated off — ratified override entry 2; REQUIREMENTS.md CI-02 honestly Partial/unchecked with the ratified annotation (re-read this pass) |
| 6 | SC-3b: The scheduled govulncheck job runs green against the Go vulnerability database | ✓ VERIFIED | STRENGTHENED this pass: the first scheduled cron occurrence is now observed — verifier-run `gh run list --workflow=vulncheck.yml`: run 41466437895, event **schedule**, conclusion **success**, created 2026-11-02T14:54:44Z (the Monday of the `0 6 * * 1` window; GitHub may delay scheduled starts — the schedule event proves the scheduler fired, plus dispatch run 41216077968 success from 2026-10-31). vulncheck.yml unchanged since prior pass |
| 7 | SC-4: Dependency boundary machine-enforced (internal/app importing net/os/grpc, internal/port beyond stdlib, fails the gate) | ✓ VERIFIED | CHANGED TREE, full re-verify: verifier re-fired BOTH red probes this pass on the phase-2 engine layout — `net` import in internal/app/clock → `import 'net' is not allowed from list 'sup-kv-app-no-io'` (depguard); `golang.org/x/mod/semver` in internal/port → `not allowed from list 'sup-kv-port-stdlib-only'`. Probes removed; full gate re-ran green; `git status internal/ cmd/` clean; go.mod/go.sum diff-clean |
| 8 | .golangci.yml is the vendored powerman 2.15.0 config byte-unmodified with exactly two additive sup-kv rules | ✓ VERIFIED | Regression: file unchanged since the prior pristine-diff pass (verifier-run git log --since 2026-10-31T22:56:10Z — not in the changed set); Origin header line 1 re-asserted this pass; exactly one occurrence each of sup-kv-app-no-io / sup-kv-port-stdlib-only |
| 9 | go.mod pins golangci-lint v2.15.0 + govulncheck v1.9.0 as tool directives with zero direct runtime requires | ✓ VERIFIED | Regression: unchanged since prior pass; tool block re-read (both directives present; requires carry golangci-lint/v2 v2.15.0 + x/vuln v1.9.0) and awk scan re-run this pass: 213 require lines, every one `// indirect` — the only `//`-less lines are the `require (` / `)` delimiters. Zero direct runtime requires |
| 10 | The strict lint gate is live and green; `config verify` exits 0 (QUAL-01) — a STANDING gate in Makefile and CI | ✓ VERIFIED | CHANGED TREE, verifier-run this pass: `make lint` → exit 0 (runs `go tool golangci-lint config verify` THEN `run ./...`; both green on the phase-2 code). ci.yml lint job carries the same standing verify step before golangci-lint-action@v9 (version: v2.15.0); both files unchanged since the prior pass that added the gate (PR #26) |
| 11 | The race gate is not vacuous (CI reports ok for internal/app) | ✓ VERIFIED | CHANGED TREE, verifier-run this pass with cache bypass: `go test -race -count=1 ./...` → `ok github.com/Djarvur/sup-kv/internal/app/clock 1.029s`, `ok .../internal/app/storage 1.677s`, `ok .../internal/app/version 1.016s` — the skeleton's single TestAppSkeleton is superseded by three real test suites (~2200 test lines), a strictly stronger race target; matrix race step green live on 307f252 |
| 12 | Matrix cells are independent (fail-fast disabled) | ✓ VERIFIED | Current ci.yml re-read: `fail-fast: false`; unchanged since prior pass (PR #3 observed evidence stands) |
| 13 | Push and PR events for the same ref do not cancel each other; superseded runs cancel | ✓ VERIFIED | Current ci.yml re-read: concurrency group `ci-${{ github.event_name }}-${{ github.ref }}` (code-enforced), `cancel-in-progress: true`; unchanged since prior pass |
| 14 | docs/setup/ruleset.json encodes the full master protection payload with contexts matching the check-run names verbatim | ✓ VERIFIED | Regression: file unchanged since prior pass; verifier-run python compare THIS pass against live check-runs on 307f252: 5 contexts == 5 live check-run names character-for-character (match: True); 4 rule types (pull_request, required_status_checks, non_fast_forward, deletion); strict policy true; enforcement active |
| 15 | delete_branch_on_merge enabled on the repository | ✓ VERIFIED | Regression, verifier-run `gh api repos/Djarvur/sup-kv` this pass → delete_branch_on_merge true, default_branch master, private true |
| 16 | docs/setup/branch-protection.md records apply/verify/re-apply commands, zero-approval rationale, context-coupling rule, and the current ratified state | ✓ VERIFIED | Regression, re-greped this pass: `## Status: DEFERRED (accepted override)` exactly once, zero PENDING occurrences, Deferral decision 2026-10-31 line exactly once, POST apply command exactly once |
| 17 | Backstop: Dependabot opens its first version-update PR within the weekly window | ✓ PASSED (override) | Window ELAPSED (Mon 2026-11-02 06:00 UTC). Verifier-run live: `gh pr list --author app/dependabot --state all` → 0 (the literal event did not occur); golangci-lint latest release = **v2.15.0** (the pin IS upstream-latest, and it is deliberately ignored per WR-01 — nothing else in the graph is updatable); weekly machinery proven firing by vulncheck run 41466437895 (event schedule, success, 2026-11-02). Maintainer decision 2026-11-02 (UAT Test 1, option A) recorded → override entry 3; WINDOWS.md item 1 fixed (open_count 0, verifier-run `gsd-tools windows status`); UAT complete 3/3 |
| G1 | (01-05 T1) overrides: frontmatter carries the two ratified entries verbatim from the prior report's own suggestion block, attributed accepted_by "onokonem" with real UTC accepted_at | ✓ VERIFIED | Both entries carried into this regenerated report's frontmatter verbatim (accepted_at "2026-10-31T18:41:25Z"); the exactly-two condition held at the 01-05 gap-closure moment (PR #17 diff evidence stands in history); the living report now additionally carries entry 3 — a sanctioned post-UAT addition, attributed the same way |
| G2 | (01-05 T2) Gap-closure edits left the report byte-unchanged outside the overrides block | ✓ VERIFIED | Historical: PR #17 diff showed the overrides block as the only change. The body was since legitimately rewritten by verifier action (PR #23, the 2026-10-31 regeneration, and this sanctioned regeneration) — the verifier's own record-keeping anticipated by plan 01-05's verification section, not tampering; verifier-computed keys recomputed each time |
| G3 | (01-05 T3) REQUIREMENTS.md CI-02 annotated Partial: checkbox unchecked, row reads Partial with the ratified alerts deferral | ✓ VERIFIED | Regression, re-read this pass (PR #28 touched the file for phase-2 rows only): `- [ ] **CI-02**` bullet; traceability row "Partial — version updates + scheduled govulncheck delivered and green; alerts half deferred (GHAS-gated on private tier), ratified accepted override 2026-10-31; unlocks at GHAS license or the deferred public flip" intact; CI-01/QUAL-01 rows Complete and untouched |
| G4 | (01-05 T4) REQUIREMENTS.md CI-03 row records the ratification of its deferral | ✓ VERIFIED | Regression, re-read this pass: row carries "ratified 2026-10-31 as accepted override (no org plan upgrade, no public flip)" after decision `defer`, re-apply path mention intact |
| G5 | (01-06 T1) branch-protection.md status states DEFERRED (accepted override) with ratification paragraph, decision line and every command verbatim | ✓ VERIFIED | Regression, re-greped this pass (see truth 16): heading + paragraph + decision line + POST/verify/PATCH/PUT/DELETE trail all present; zero pending-titled headings; file unchanged since prior pass |
| G6 | (01-06 T2) PROJECT.md and .claude/CLAUDE.md Process constraints no longer assert server-side enforcement — identical deferral parenthetical in both (IN-02 closed at source) | ✓ VERIFIED | Regression, re-verified this pass: `(server-side ruleset deferred 2026-10-31 — …)` found exactly once in each; diff of the extracted parentheticals → character-identical (mirror == source); both files unchanged in the window |
| G7 | (01-06 T3) WINDOWS.md item 2 waived with the override reason; ledger consistent; item 1 resolved | ✓ VERIFIED | Verifier-run `gsd-tools windows status` this pass: open_count 0, waived_count 1, fixed_count 1 — item 1 status **fixed** (resolved_at 2026-11-02T15:55:45Z, the recorded `windows fixed 1` after the UAT observation); item 2 status waived with the ratified-override reason naming the 403 and the re-apply path |
| G8 | (01-06 T4) STATE.md decision log carries one entry per ratified override, citing 2026-10-31 and gap-plan provenance | ✓ VERIFIED | Regression, re-greped this pass (PR #28 rewrote STATE.md for phase-2 progress; the decision entries survived): exactly one "Verification gap 1 ratified 2026-10-31" and one "Verification gap 2 ratified 2026-10-31" entry |

**Score:** 25/25 truths verified (4 via maintainer-ratified overrides; 0 present-but-behavior-unverified)

### Override Ratification Audit (Step 3b)

Three maintainer-ratified overrides carry in the frontmatter. Entries 1-2 (accepted_by "onokonem",
accepted_at "2026-10-31T18:41:25Z") carry forward verbatim — their factual predicate re-verified
live this pass: the rulesets endpoint still returns 403 and the repo is still private on the plan-gated
tier. Entry 3 is new this pass, recording the 2026-11-02 maintainer decision (UAT Test 1, option A)
that resolved the last open backstop; every factual predicate in its reason was re-verified live by
this verifier (0 dependabot PRs; golangci-lint latest = v2.15.0; on-schedule vulncheck cron success).
Token-overlap matching maps entry 1 to truths 2+3 (SC-1b + SC-2a), entry 2 to truth 5 (SC-3a), and
entry 3 to truth 17: 4 truths PASSED (override). Every marker the gap-closure plans established still
tells the same one story (REQUIREMENTS rows, DEFERRED status, ledger waiver + fix, decision log,
constraint parentheticals — all re-verified this pass, truths G3-G8).

### Deferred Items

None — no Phase 1 gap is scheduled into a later milestone phase; the original two gaps are closed by
ratified override (entries 1-2), and the backstop is closed by override entry 3. The roadmap's later
phases (3-7) cover none of the deferred halves (ruleset application, GHAS alerts) — those remain
one-command / one-billing-decision away, tracked in WINDOWS.md (waived) and docs/setup/, not in a
later phase.

### Advisory (New Scope, Unevidenced)

Re-verification ran; Step 7's scan of all files changed since the prior verification (the phase-2
engine files under internal/app/, AGENTS.md) found zero Step-7 blockers, zero unreferenced debt
markers, zero stub patterns, and zero unjustified suppressions. Two WARNING-severity findings from
the concurrent 2026-11-02 code-review increment (landed during this verification pass) flag phase-2
clock code and are reported here as advisory — they are new-scope relative to the Phase 1 contract
(no Phase 1 truth touches clock arithmetic), evidenced by no failing test, and already tracked open
in 01-REVIEW-DISPOSITION.md for phase-2 triage.

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| 1 | WR-01: driftBetween int64 overflow at the HLC domain ceiling (internal/app/clock/clock.go:144) | other | new-scope (phase-2 code via PR #28), no deterministic evidence; tracked open in 01-REVIEW-DISPOSITION |
| 2 | WR-02: NewClock does not clamp initialL to the 48-bit domain (internal/app/clock/clock.go:63-65) | other | new-scope (phase-2 code), no deterministic evidence; tracked open in 01-REVIEW-DISPOSITION |

### Required Artifacts

gsd-tools `verify.artifacts`: 18/20 pass across all 6 plans. The 2 reported misses are the superseded
walking-skeleton paths — `internal/app/doc.go` and `internal/app/app_test.go` (plan 01-01
declarations), deleted by PR #28 in favor of the real engine. Their declared purpose — doc-commented
packages under internal/app and a real race-test target (D-01) — is fulfilled by the replacements,
verified live this pass: strict lint green (package-comments enforced by the gate), `go test -race`
ok for internal/app/{clock,storage,version}. Supersession, not a gap.

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| go.mod | module + tool directives + go 1.27 | ✓ VERIFIED | unchanged; tool block + 213/213 indirect re-verified |
| cmd/sup-kv/main.go | Command doc comment, minimal main | ✓ VERIFIED | unchanged |
| internal/port/doc.go | Package comment, stdlib-only target | ✓ VERIFIED | unchanged; port red probe re-fired this pass |
| internal/app/doc.go | Package comment, pure-core target | ✓ SUPERSEDED (purpose fulfilled) | deleted by PR #28; internal/app packages now documented per-package (storage/doc.go etc.); lint's package-comments gate green |
| internal/app/app_test.go | Real race target (D-01) | ✓ SUPERSEDED (purpose fulfilled) | deleted by PR #28; replaced by ~2200 lines of real tests in clock/storage/version — `ok` under -race this pass, a strictly stronger target |
| .golangci.yml | Vendored strict config + boundary | ✓ VERIFIED | unchanged; boundary probes re-fired red this pass |
| Makefile | lint/test/generate/build via go tool | ✓ VERIFIED | unchanged; `make lint` verifier-run exit 0 (verify + run) |
| AGENTS.md | Conventions + 3 deviations + D-03 reading | ✓ VERIFIED | CHANGED (PR #28): Layout bullet now describes version/clock/storage + testutil; all three deviations, the D-03 reading, and the tool-invocation rule byte-intact (diff inspected) |
| .github/workflows/ci.yml | lint + 4-cell matrix + hygiene | ✓ VERIFIED | unchanged; 5/5 live check-runs green on 307f252 |
| .github/workflows/vulncheck.yml | cron + paths trigger + dispatch, pinned scan | ✓ VERIFIED | unchanged; scheduled run observed green 2026-11-02 |
| .github/dependabot.yml | gomod + github-actions updates | ✓ VERIFIED | unchanged; WR-01 ignore with lockstep justification intact |
| docs/setup/ruleset.json | Ruleset payload, 4 rule types, 5 contexts | ✓ VERIFIED | unchanged; contexts == live check-run names re-confirmed this pass |
| docs/setup/branch-protection.md | Apply/verify/re-apply + ratified DEFERRED marker | ✓ VERIFIED | unchanged; marker re-greped |
| .planning/REQUIREMENTS.md | CI-02 Partial annotation + CI-03 ratification suffix | ✓ VERIFIED | re-read post-PR #28; phase-1 rows intact |
| .planning/WINDOWS.md | ledger item 2 waived; item 1 fixed | ✓ VERIFIED | open_count 0 / waived 1 / fixed 1 (tooling + file) |
| .planning/STATE.md | decision-log entries for both ratified overrides | ✓ VERIFIED | exactly two gap-ratification entries (survived PR #28 rewrite) |
| .planning/PROJECT.md | Process constraint stating the deferral honestly | ✓ VERIFIED | parenthetical present exactly once |
| .claude/CLAUDE.md | generated mirror in sync (IN-02 closed) | ✓ VERIFIED | parenthetical character-identical to source |

### Key Link Verification

gsd-tools `verify.key-links`: 9/13 automated-verified; the 4 "failures" are all the known tooling
limitation (descriptive `from:` fields that are not relative file paths — the tool reports
"from: must be a relative file path"); each verified manually with direct evidence this pass,
as in prior passes.

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| Makefile | go.mod | `go tool golangci-lint` | ✓ WIRED | Automated (01-01: 3/3) |
| .golangci.yml | internal/app | `**/internal/app/**` deny rule | ✓ WIRED | Automated; red probe re-fired this pass on the phase-2 subtree |
| .golangci.yml | internal/port | `**/internal/port/**` strict list-mode | ✓ WIRED | Automated; red probe re-fired this pass |
| ci.yml | go.mod | `version: v2.15.0` + `go-version-file` | ✓ WIRED | Automated |
| ci.yml | .golangci.yml | lint job runs vendored config + standing verify step | ✓ WIRED | Automated |
| ci.yml (gate display names) | docs/setup/ruleset.json | contexts verbatim | ✓ WIRED (manual) | Verifier-run python compare this pass: 5 contexts == 5 live check-run names on 307f252, character-for-character |
| vulncheck.yml | go.mod | `go tool govulncheck` | ✓ WIRED | Automated (01-03: 2/2) |
| dependabot.yml | repo root | `directory: /` both ecosystems | ✓ WIRED | Automated |
| ruleset.json | ci.yml check-run names | contexts == display names | ✓ WIRED | Automated (01-04: 2/2) |
| branch-protection.md | GitHub rulesets API | re-runnable gh api commands | ✓ WIRED | Automated; GET re-run live this pass (403 as recorded) |
| 01-VERIFICATION.md overrides entries | ratified override state | verbatim + ratification clause | ✓ WIRED (manual) | Entries 1-2 carried verbatim; entry 3 added this pass from the recorded UAT maintainer decision; all markers re-verified |
| REQUIREMENTS.md CI-02 row | CI-03 row | same Partial-with-explanation pattern | ✓ WIRED (manual) | Both rows re-read this pass — same shape |
| .claude/CLAUDE.md Process line | PROJECT.md Process constraint | identical parenthetical | ✓ WIRED (manual) | diff of extracted parentheticals → identical |
| WINDOWS.md item 2 reason | ratified override 2026-10-31 | reason names ratification + re-apply path | ✓ WIRED (manual) | Ledger reason quotes both (windows status output read this pass) |

### Data-Flow Trace (Level 4)

No UI/rendered data in this phase. Infrastructure equivalent re-confirmed: lint job ←
.golangci.yml (schema-verified twice — standing `config verify` step + the action's own verify);
setup-go ← go.mod; govulncheck ← go.mod tool graph (pinned v1.9.0); check-runs ← real matrix
execution (5/5 success observed live on current origin/master HEAD 307f252); ledger counters ←
gsd-tools windows status reading the actual file (open 0 / waived 1 / fixed 1). No static/hollow
values.

### Behavioral Spot-Checks

All commands verifier-run in a fresh shell this pass; probe files created and removed by the
verifier; trees verified clean after.

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full local gate incl. standing verify | `make lint` | exit 0 (config verify + run ./... both green) | ✓ PASS |
| Build + vet + race tests (cache bypassed) | `go build ./... && go vet ./... && go test -race -count=1 ./...` | exit 0; ok clock/storage/version (+cmd, port, testutil: no test files) | ✓ PASS |
| Depguard RED — net in internal/app (non-test file) | probe + lint run | `import 'net' is not allowed from list 'sup-kv-app-no-io'` | ✓ PASS (red as required) |
| Depguard RED — third-party in internal/port | probe + lint run | `not allowed from list 'sup-kv-port-stdlib-only'` | ✓ PASS (red as required) |
| Gate GREEN after probe removal | `go tool golangci-lint run ./...` + `git status internal/` + `git diff go.mod go.sum` | exit 0; tree clean; no module churn | ✓ PASS |
| Matrix green on current origin/master HEAD | `gh api commits/307f252/check-runs` | 5 names, 5× success | ✓ PASS |
| PR #28 (the stale-making change) green | `gh pr checks 28` + `gh pr view 28` | 5/5 pass, MERGED 2026-11-02T15:34:33Z | ✓ PASS |
| Scheduled vulncheck fired green | `gh run list --workflow=vulncheck.yml` | run 41466437895: event schedule, success, 2026-11-02 | ✓ PASS |
| Dependabot PR inventory | `gh pr list --author app/dependabot --state all` | 0 — nothing updatable (see override 3) | ✓ PASS (state as recorded) |
| golangci-lint upstream latest | `gh api repos/golangci/golangci-lint/releases/latest` | v2.15.0 == the pin | ✓ PASS |
| Rulesets endpoint state | `gh api repos/Djarvur/sup-kv/rulesets` | 403 plan-gated — matches the ratified override's predicate | ✓ PASS (state as recorded) |
| delete_branch_on_merge | `gh api repos/Djarvur/sup-kv` | true (default_branch master, private true) | ✓ PASS |
| Commit hash validity | `gsd-tools query verify.commits` (11 hashes: 307f252, 0598747, 60db0fb, 0ca46c9, 8c33bc0, 515cf2f, 0dce266, ae09dc7, 4a2e173, acf1215, full 0598747a…) | all_valid: true | ✓ PASS |
| Windows ledger state | `gsd-tools windows status` | open_count 0, waived_count 1, fixed_count 1 | ✓ PASS |
| Ruleset contexts == live check-run names | python3 compare vs check-runs on 307f252 | match: True (5/5) | ✓ PASS |
| Decision coverage | `gsd-tools query check.decision-coverage-verify` | 14/14 honored, 0 not honored | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist (no scripts directory). The phase's probes are inline
lint runs — executed directly by this verifier (Behavioral Spot-Checks rows 3-5), re-fired on the
phase-2 tree because internal/ changed since the prior pass. SUMMARY probe/gate claims were
reproduced first-hand via file reads, live gh queries, and tooling queries, not trusted.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| QUAL-01 | 01-01 | Strict golangci-lint config + go-ultimate conventions govern every package; red gate blocks merge | ✓ SATISFIED | `make lint` verifier-run green on the full current tree (standing config verify in Makefile + CI); boundary red/green re-proven both lists; zero suppressions without specific-linter + written justification (7 nolint in testutil, all compliant with D-12); vendored config byte-unmodified; merge-blocking half subject to the ratified override (CI-03 scope) |
| CI-01 | 01-01, 01-02 | Actions runs quality gates on every push and PR, Linux/macOS, amd64/arm64 | ✓ SATISFIED | push+PR triggers, 4 cells re-read; 5/5 green live on origin/master HEAD 307f252 and on PR #28 (verifier-run); standing config-verify step in the lint job |
| CI-02 | 01-03, 01-05 | Dependabot current (updates and alerts) + scheduled govulncheck green | ✓ SATISFIED-BY-RATIFICATION (Partial) | Updates configured + live (WR-01 ignore hardening intact); scheduled govulncheck observed green on its first cron occurrence (2026-11-02); alerts half deferred under ratified override entry 2; REQUIREMENTS.md honestly Partial/unchecked; first-PR backstop resolved by override entry 3 (maintainer decision 2026-11-02, UAT complete) |
| CI-03 | 01-04, 01-05, 01-06 | PR-only development — branch protection on master, direct pushes forbidden | ✓ SATISFIED-BY-RATIFICATION (Partial) | PR-only practiced and verified (all feature/code changes through PRs #1-#28); server-side protection deferred under ratified override entry 1 (403 predicate re-verified live); payload + one-command re-apply scripted and verbatim-preserved; DEFERRED marker + ledger waiver + decision log all in place |

Orphaned requirements: none — REQUIREMENTS.md maps exactly CI-01, CI-02, CI-03, QUAL-01 to
Phase 1; all four appear in plan frontmatter requirements (01-05/01-06 claim CI-02/CI-03 for the
ratification recording with requirements-completed deliberately empty — honest, not an omission).

### Anti-Patterns Found

Debt-marker gate over all files changed since the prior verification (internal/app engine files,
AGENTS.md, planning docs): zero unreferenced TBD/FIXME/XXX in implementation files. The single
vendored FIXME remains .golangci.yml line 155 (upstream powerman content the byte-unmodified
prohibition forbids touching). ROADMAP.md `**Plans**: TBD` lines for unplanned phases 3-7 remain
the planning template's honest state (each names its own formal follow-up phase). No stubs, no
empty implementations, no skipped tests; every `//nolint` names its linter and carries a reason
(D-12 compliant — verified this pass in testutil).

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| local master (0598747, adddd0b, ff6e24f) | — | 3 unpushed docs commits on local master ahead of origin, outside the PR flow | ⚠️ Warning | Carried-forward class (prior round flagged the identical condition); the gsd tooling's own docs commits are its sanctioned record-keeping mode, but pushing master directly extends the disclosed exception — route or push consciously |
| internal/app/clock/clock.go | 144 | WR-01 (2026-11-02 review increment): driftBetween int64 overflow at domain ceiling | ⚠️ Warning | New-scope phase-2 finding, no failing test; tracked open in 01-REVIEW-DISPOSITION (see Advisory) |
| internal/app/clock/clock.go | 63-65 | WR-02 (2026-11-02 review increment): initialL not clamped to 48-bit domain | ⚠️ Warning | New-scope phase-2 finding, no failing test; tracked open in 01-REVIEW-DISPOSITION (see Advisory) |
| .planning/ROADMAP.md | 63+ | `**Plans**: TBD` for unplanned phases | ℹ️ Info | Planning-template placeholders naming their own follow-up phase |
| .golangci.yml | 155 | upstream FIXME (vendored, byte-unmodified rule) | ℹ️ Info | Carried forward; touching it would violate the stronger prohibition |
| AGENTS.md | 5 | IN-01: quotes `go 1.27`, go.mod pins 1.27.1 | ℹ️ Info | Open; harmless self-referential pointer |
| ci.yml / vulncheck.yml | — | IN-03: actions pinned by mutable major tags | ℹ️ Info | Open hardening advice (bounded by contents:read, no secrets, no pull_request_target) |
| docs/setup/branch-protection.md | — | IN-05: "Apply command above" direction nit | ℹ️ Info | Open review Info; doc contains exactly one Apply command |
| ci.yml lint job | — | IN-06: schema gate and lint run separately provisioned, both v2.15.0 | ℹ️ Info | Version parity holds (verifier-checked both pins this pass); WR-01 ignore guards drift |

IN-02: FIXED (source + mirror parentheticals verified identical this pass). WR-01 (dependabot
ignore): FIXED (config re-read this pass). The new review round's WR-01/WR-02 above reuse IDs from
the review's own increment sequence — they are distinct findings, correctly tracked in
01-REVIEW-DISPOSITION.md.

### Test Quality Audit

The phase's original race target (app_test.go/TestAppSkeleton) is superseded; the current
requirement-linked test files (internal/app/clock/{clock_test,hlc_prop_test}.go,
internal/app/storage/{store_test,merge_test}.go, internal/app/version/version_test.go) are active,
not skipped (zero t.Skip / testing.Short matches), no circular generation (testutil generators are
independent PCG-based constructors, not system-output re-runs), and assertion level is value/
property-level (seeded property suites + Kulkarni unit tables) — appropriate and strong for the
compile/liveness purpose phase 1 declared. Gate-teeth proofs are lint-probe based (red/green),
exercised directly this pass.

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
result pass; 01-UAT.md now complete 3/3; WINDOWS.md ledger item 1 fixed, open_count 0). The 7
judgment-tier prohibitions carry recorded maintainer concurrence (01-UAT.md Test 2: all 7 reviewed
and accepted — QUAL-01 ×2, CI-01 ×2, CI-02 ×2, CI-03), and this verifier's independent scan found
no prohibition violated (vendored config byte-unmodified; every nolint justified; no gate-theater
mechanism — the standing config-verify gate catches the no-op class; local/CI linter versions
identical at v2.15.0; no runtime dependency entered the graph — 213/213 requires indirect; no
vulnerability suppressions in vulncheck.yml; no path-based protection exemptions in ruleset.json).

## Gaps Summary

No gaps remain. The regeneration trigger — PR #28 superseding the walking skeleton with the real
engine inside this phase's covered paths — was itself the phase goal working as designed: the
first feature code landed through the PR gate (5/5 checks green, verifier-observed live), inside
the depguard boundary (both red probes re-fired on the new tree), with the strict gate green
(make lint verifier-run, zero findings). Every previously verified artifact that still exists is
byte-unchanged since the prior pass, and the superseded skeleton's purpose is fulfilled more
strongly by its replacements (three real race-tested packages where one placeholder test stood).
The two maintainer-ratified overrides carry forward verbatim with their 403/GHAS predicates
re-verified live, and the last open backstop closed via a third recorded override whose every
factual claim this verifier re-confirmed independently (zero updatable dependencies; golangci-lint
v2.15.0 is upstream-latest; the weekly cron demonstrably fired on schedule and green). The phase
closes at 25/25 with no open ledger items (open_count 0) and UAT complete 3/3. Two warning-severity
review findings on phase-2 clock code are reported as advisory and tracked in 01-REVIEW-DISPOSITION
for phase-2 triage; they touch no Phase 1 truth.

---

_Verified: 2026-11-02T16:07:36Z_
_Verifier: Claude (gsd-verifier)_
