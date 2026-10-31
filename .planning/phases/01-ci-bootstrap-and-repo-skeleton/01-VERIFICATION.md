---
phase: 01-ci-bootstrap-and-repo-skeleton
verified: 2026-10-31T16:35:20Z
status: gaps_found
score: 13/17 must-haves verified
covered_files:
  - .github/dependabot.yml
  - .github/workflows/ci.yml
  - .github/workflows/vulncheck.yml
  - .gitignore
  - .golangci.yml
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-01-PLAN.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-01-SUMMARY.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-02-PLAN.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-02-SUMMARY.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-03-PLAN.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-03-SUMMARY.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-04-PLAN.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-04-SUMMARY.md
  - AGENTS.md
  - Makefile
  - cmd/sup-kv/main.go
  - docs/setup/branch-protection.md
  - docs/setup/ruleset.json
  - go.mod
  - go.sum
  - internal/app/app_test.go
  - internal/app/doc.go
  - internal/port/doc.go
covered_digest: "v4:sha256:c1762d8c220ae94b56e916574c64d7042c5b9de12f30c622efb51e85a6502c15"
behavior_unverified: 1
overrides_applied: 0
gaps:
  - truth: "Direct pushes to master are rejected by branch protection (roadmap SC-2) and a red gate blocks the merge (SC-1 enforcement half, CI-03 protection sub-item)"
    status: failed
    reason: "The 'protect master' ruleset is NOT applied — the GitHub rulesets API is plan-gated on this private org repo (live 403 'Upgrade to GitHub Pro or make this repository public' re-verified during this verification). The defer decision (2026-10-31, auto-selected at the plan 01-04 blocking checkpoint) is recorded in docs/setup/branch-protection.md's PENDING marker, but no later milestone phase covers the application, so master has no server-side push protection and no required checks: a red PR could currently be merged and direct pushes would be accepted."
    artifacts:
      - path: docs/setup/branch-protection.md
        issue: "Status marker reads PENDING (accurate) — payload scripted and re-appliable in one command, but enforcement absent on the GitHub side"
      - path: docs/setup/ruleset.json
        issue: "Payload complete and field-verified, but never POSTed (403)"
    missing:
      - "Human billing decision: upgrade the Djarvur org plan to a tier allowing rulesets on private repos (or flip the repo public per the deferred decision), then run the apply command from docs/setup/branch-protection.md and update the PENDING marker to APPLIED"
      - "OR ratify the deferral as an accepted override (see override suggestion in this report) — the auto-selected 'defer' has not had explicit human acceptance"
  - truth: "Dependabot is active (updates AND alerts) — roadmap SC-3 / CI-02 alerts half"
    status: failed
    reason: "Dependabot ALERTS require a GHAS/Code Security license unavailable on this private org tier; alerts are off. Version updates are configured and live (both ecosystems), and the caveat is documented in 01-03-SUMMARY.md, STATE.md and .github/dependabot.yml's header comment — but the requirement text includes alerts, which are not active, and REQUIREMENTS.md marks CI-02 'Complete' without the annotation CI-03 carries."
    artifacts:
      - path: .github/dependabot.yml
        issue: "Updates active; alerts GHAS-gated off (external billing constraint, documented)"
      - path: .planning/REQUIREMENTS.md
        issue: "CI-02 checkbox marked Complete while the alerts half is off — traceability overstates (contrast CI-03's honest 'Partial' annotation)"
    missing:
      - "GHAS license (org billing) or the deferred public flip to unlock alerts — OR ratify the documented caveat as an accepted override"
      - "Annotate CI-02 in REQUIREMENTS.md as 'Partial — alerts GHAS-gated' mirroring the CI-03 pattern, so the traceability record matches reality"
behavior_unverified_items:
  - truth: "Dependabot opens its first version-update PR within the weekly schedule window (plan 01-03 backstop truth)"
    test: "After Monday 2026-11-02 06:00 UTC (first cron window), list PRs authored by app/dependabot"
    expected: "At least one gomod or github-actions update PR (grouped minor+patch) exists against master"
    why_human: "insufficient_spec abstention — the weekly window had not elapsed at verification time (config merged 2026-10-31, Saturday); live check shows zero Dependabot PRs so far, which proves nothing either way; presence+wiring of the config is verified but the first-PR event is a future observation"
overrides:
  - must_have: "Direct pushes to master are rejected by branch protection; a red gate blocks the merge"
    reason: "Rulesets API 403 on private org repo (plan-gated); payload + one-command re-apply recorded in docs/setup/branch-protection.md; defer decision taken 2026-10-31 at the plan 01-04 blocking checkpoint ; ratified by the maintainer as an accepted override 2026-10-31 (no org plan upgrade, no public flip)"
    accepted_by: "onokonem"
    accepted_at: "2026-10-31T18:41:25Z"
  - must_have: "Dependabot is active (updates and alerts)"
    reason: "Alerts GHAS-gated on private repo tier; version updates + govulncheck delivered and green; alerts unlock at the deferred public flip ; ratified by the maintainer as an accepted override 2026-10-31."
    accepted_by: "onokonem"
    accepted_at: "2026-10-31T18:41:25Z"
---

# Phase 1: CI Bootstrap and Repo Skeleton Verification Report

**Phase Goal:** Quality gates and the enforced repo skeleton exist before any feature code, so every subsequent change lands inside the architecture and behind a red-light-blocking gate
**Verified:** 2026-10-31T16:35:20Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Mode Note (MVP discrepancy — human decision requested)

ROADMAP.md marks Phase 1 `Mode: mvp`, but the goal is outcome-shaped, not the required
"As a [role], I want to [capability], so that [outcome]" user story —
`gsd-tools query user-story.validate` returns `valid: false`. Per
`gsd-core/references/verify-mvp-mode.md`, the MVP user-flow framing does not apply to a
non-story goal; this is an infrastructure phase with no user-facing flow regardless
(infra-phase UAT scoping applies). Verification therefore proceeded with the standard
goal-backward methodology. **Discrepancy for the developer:** run `/gsd:mvp-phase 1` to
set a proper User Story goal, or drop the `mvp` mode marker — the current combination
produces a low-quality User Flow Coverage section by construction. The plans themselves
record the goal as outcome-shaped "per the no-invention rule", so this appears to be a
deliberate project choice awaiting formalization.

## Goal Achievement

### Observable Truths

Merged set: 4 ROADMAP success criteria (split into their independently-verifiable
halves) + deduplicated PLAN frontmatter truths. Live GitHub state re-verified via `gh`
during this pass; all local gates re-run by the verifier (not trusted from SUMMARYs).

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC-1a: A PR runs the full gate matrix (strict lint, go vet, race tests) across Linux/macOS × amd64/arm64 | ✓ VERIFIED | `gh api commits/f9110ee/check-runs` (verifier-run): exactly 5 check-runs — lint, gate (ubuntu-latest), gate (ubuntu-24.04-arm), gate (macos-15), gate (darwin-amd64-cross) — all conclusion success; ci.yml inspected (4-cell include matrix, vet+race+tidy native, build+vet cross); D-05 amended interpretation (3 native cells + darwin/amd64 cross) recorded in 01-CONTEXT.md |
| 2 | SC-1b: A red gate blocks the merge | ✗ FAILED | Red path proven (probe PR #4: lint check failed citing sup-kv-app-no-io, PR CLOSED unmerged — state re-verified live) but no server-side required checks exist: the ruleset that would enforce them is not applied (403). Merging a red PR is currently possible; blocking is by discipline only. → Gap 1 |
| 3 | SC-2a: Direct pushes to master are rejected by branch protection | ✗ FAILED | `gh api repos/Djarvur/sup-kv/rulesets` → 403 "Upgrade to GitHub Pro or make this repository public" (re-verified live by verifier). PENDING marker + defer decision 2026-10-31 in docs/setup/branch-protection.md. → Gap 1 |
| 4 | SC-2b: Every change (code and docs) lands through a pull request | ✓ VERIFIED | origin/master history: all phase changes behind 10 "Merge pull request #" commits (#1-#11); PR states re-verified live (MERGED except probe #4 CLOSED). Warning: 2 local docs commits (d7a5054, bfcc1eb — review reports) sit on local master ahead of origin, outside the PR flow; pushed directly they would break the practiced contract |
| 5 | SC-3a: Dependabot is active (updates and alerts) | ✗ FAILED | Updates: config live and shape-verified (gomod + github-actions, weekly, grouped minor+patch, prefixes). Alerts: GHAS-gated OFF on this private tier — documented caveat (01-03-SUMMARY, dependabot.yml header, STATE.md) but the requirement's alerts half is not delivered. → Gap 2 |
| 6 | SC-3b: The scheduled govulncheck job runs green against the Go vulnerability database | ✓ VERIFIED | `gh run list --workflow=vulncheck.yml` (verifier-run): run 41216077968, event workflow_dispatch, headBranch master, conclusion success. Cron `0 6 * * 1` + go.mod/go.sum paths trigger + dispatch all present in vulncheck.yml; pinned `go tool govulncheck` invocation (no latest-tag run). First *scheduled* occurrence not yet elapsed — dispatch is the plan's declared proof trigger |
| 7 | SC-4: Dependency boundary machine-enforced from the first feature commit (internal/app importing net/os/grpc, internal/port beyond stdlib, fails the gate) | ✓ VERIFIED | Verifier re-ran the red/green probe (sanctioned re-runnable by plan 01-01): probe importing `net` in internal/app → exit 1, `import 'net' is not allowed from list 'sup-kv-app-no-io'`; `golang.org/x/mod/semver` in internal/port → `not allowed from list 'sup-kv-port-stdlib-only'`; after removal → exit 0. Deny list covers net, os, syscall, google.golang.org/ (grpc); prealloc finding confirms the wider strict config live |
| 8 | .golangci.yml is the vendored powerman 2.15.0 config byte-unmodified with exactly two additive sup-kv rules (D-10 amended) | ✓ VERIFIED | Verifier fetched pristine upstream and diffed: single pure insertion (64a65,88 — banner + 2 rules); zero upstream lines changed; Origin header verbatim on line 1; each rule key occurs exactly once |
| 9 | go.mod pins golangci-lint v2.15.0 + govulncheck v1.9.0 as tool directives with zero direct runtime requires (D-03/D-04) | ✓ VERIFIED | `tool (` block present; require lines carry v2.15.0 (line 94) and v1.9.0 (line 216) both `// indirect`; awk scan of the require block: 215 requires, zero without `// indirect` |
| 10 | The strict lint gate is live and green on the skeleton; `config verify` exits 0 (QUAL-01) | ✓ VERIFIED | Verifier-run: `go tool golangci-lint config verify` → 0; `go tool golangci-lint run ./...` → 0; package doc comments present in all three packages |
| 11 | The race gate is not vacuous (CI reports ok for internal/app) | ✓ VERIFIED | TestAppSkeleton exists on master (table-driven, t.Run subtests); verifier-run `go test -race ./...` → `ok github.com/Djarvur/sup-kv/internal/app`; gate cells whose scripts run `go test -race` concluded success. (The specific CI log line is SUMMARY-quoted; Actions logs no longer served via API — local equivalent run is first-hand) |
| 12 | Matrix cells are independent (fail-fast disabled) | ✓ VERIFIED | ci.yml `fail-fast: false`; PR #3's first run observed three red native cells while lint + cross cells completed — structural + observed evidence |
| 13 | Push and PR events for the same ref do not cancel each other; superseded runs cancel | ✓ VERIFIED | Concurrency group `ci-${{ github.event_name }}-${{ github.ref }}` embeds the event name (code-enforced, not incidental); push run on master + PR runs both observed completing; `cancel-in-progress: true` present |
| 14 | docs/setup/ruleset.json encodes the full master protection payload with contexts matching the recorded check-run names verbatim (D-08) | ✓ VERIFIED | Verifier-run `python3 -m json.tool` → valid; 4 rule types (pull_request 0-approval, required_status_checks strict, non_fast_forward, deletion); the 5 context strings equal the live check-run names from merge SHA f9110ee character-for-character |
| 15 | delete_branch_on_merge enabled on the repository | ✓ VERIFIED | Verifier-run `gh api repos/Djarvur/sup-kv --jq '.delete_branch_on_merge'` → true (also default_branch master, private true) |
| 16 | docs/setup/branch-protection.md records apply/verify/re-apply commands, zero-approval rationale, context-coupling rule, and the PENDING state | ✓ VERIFIED | File inspected: POST apply, read-side verify, PUT update + DELETE re-apply paths, PATCH delete_branch_on_merge, context-coupling rule, no-path-exemptions note, zero-approval rationale, `## Status: PENDING` with "Deferral decision 2026-10-31: defer" |
| 17 | Backstop: Dependabot opens its first version-update PR within the weekly window | ⚠ ABSTAINED (insufficient_spec) | Config merged Sat 2026-10-31; first cron window Mon 2026-11-02 06:00 UTC — not elapsed. Verifier-run `gh pr list --author app/dependabot` → none yet (proves nothing either way). Abstains per honest-verifier rules → human review item, never a silent pass |

**Score:** 13/17 truths verified (3 failed, 1 abstained-insufficient-spec)

### Gaps Summary

Two root causes, both external billing constraints on the private org tier, both
explicitly decision-gated at a sanctioned blocking checkpoint — but neither ratified by
a human, and neither covered by any later milestone phase (Step 9b check: Phases 2-7
goals mention neither rulesets nor GHAS):

1. **Branch protection not applied (SC-2a, SC-1b, CI-03 protection sub-item)** — the
   rulesets API 403s; payload + one-command re-apply sit ready in docs/setup/. Master's
   protection is convention-only today. The auto chain selected `defer`; per ADR-550 the
   judgment-tier deferral still needs explicit human resolution.
2. **Dependabot alerts off (SC-3a / CI-02 alerts half)** — GHAS license unavailable;
   updates active, caveat documented in three places, but CI-02 is checkbox-marked
   "Complete" in REQUIREMENTS.md without an annotation (unlike CI-03's honest Partial).

Both failures look intentional (documented, decision-gated). **Override suggestions** —
to accept either deviation, add to this file's frontmatter and re-verify:

```yaml
overrides:
  - must_have: "Direct pushes to master are rejected by branch protection; a red gate blocks the merge"
    reason: "Rulesets API 403 on private org repo (plan-gated); payload + one-command re-apply recorded in docs/setup/branch-protection.md; defer decision taken 2026-10-31 at the plan 01-04 blocking checkpoint"
    accepted_by: "{name}"
    accepted_at: "{ISO timestamp}"
  - must_have: "Dependabot is active (updates and alerts)"
    reason: "Alerts GHAS-gated on private repo tier; version updates + govulncheck delivered and green; alerts unlock at the deferred public flip"
    accepted_by: "{name}"
    accepted_at: "{ISO timestamp}"
```

Alternatively, close gap 1 for real: upgrade the org plan, run the apply command, flip
the marker to APPLIED.

### Required Artifacts

gsd-tools `verify.artifacts`: 13/13 pass across the 4 plans (exists, substantive, no issues).

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| go.mod | module + tool directives + go 1.27 | ✓ VERIFIED | module github.com/Djarvur/sup-kv; go 1.27.1; tool block; 215 indirect requires |
| cmd/sup-kv/main.go | Command doc comment, minimal main | ✓ VERIFIED | `// Command sup-kv` comment, `func main() {}` |
| internal/port/doc.go | Package comment, stdlib-only target | ✓ VERIFIED | `// Package port` doc present |
| internal/app/doc.go | Package comment, pure-core target | ✓ VERIFIED | `// Package app` doc present |
| internal/app/app_test.go | Real race target (D-01) | ✓ VERIFIED | TestAppSkeleton, table-driven, t.Run, t.Parallel |
| .golangci.yml | Vendored strict config + boundary | ✓ VERIFIED | Origin header line 1; 535 lines; rules once each; pure-insertion diff vs upstream |
| Makefile | lint/test/generate/build via go tool | ✓ VERIFIED | .PHONY targets inspected; `go tool golangci-lint run ./...` |
| AGENTS.md | Conventions + 3 deviations + D-03 reading | ✓ VERIFIED | All three deviations + dependency policy section present |
| .github/workflows/ci.yml | lint + 4-cell matrix + hygiene | ✓ VERIFIED | All keys inspected; no pull_request_target; no github.event.* in run scripts |
| .github/workflows/vulncheck.yml | cron + paths trigger + dispatch, pinned scan | ✓ VERIFIED | All three triggers; `go tool govulncheck ./...` |
| .github/dependabot.yml | gomod + github-actions updates | ✓ VERIFIED | version 2; both ecosystems; groups; prefixes; limit 10 |
| docs/setup/ruleset.json | Ruleset payload, 4 rule types, 5 contexts | ✓ VERIFIED | JSON valid; contexts == live check-run names |
| docs/setup/branch-protection.md | Apply/verify/re-apply + PENDING marker | ✓ VERIFIED | All command forms + rationale + defer decision present |

.gitignore (modified, not in must_haves.artifacts): verified merged — original `/sup-kv`, `coverage.*` retained plus `.gocache/`, `dist/`, `*.exe`, `*.coverprofile`, `.cache/`, `.gomodcache/`, `*.swp`, `*~`, `Thumbs.db`, `.env`.

### Key Link Verification

gsd-tools `verify.key-links`: 8/9 automated-verified; 1 tooling limitation, verified manually with live evidence.

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| Makefile | go.mod | `go tool golangci-lint` | ✓ WIRED | Pattern found (automated) |
| .golangci.yml | internal/app | quoted `**/internal/app/**` glob | ✓ WIRED | Pattern found; probe proved the glob matches (red fired) |
| .golangci.yml | internal/port | `**/internal/port/**` strict list-mode | ✓ WIRED | Pattern found; probe proved live |
| ci.yml | go.mod | `version: v2.15.0` + `go-version-file` | ✓ WIRED | Both present; equals go.mod tool pin |
| ci.yml | .golangci.yml | lint job runs vendored config | ✓ WIRED | golangci-lint-action on default config path |
| ci.yml (gate job display names) | docs/setup/ruleset.json | contexts verbatim | ✓ WIRED (manual) | Automated checker cannot parse composite from-field; verifier cross-checked live check-run names on f9110ee against ruleset.json contexts — character-for-character equal |
| vulncheck.yml | go.mod | `go tool govulncheck` | ✓ WIRED | Pattern found; pinned, no latest-tag run |
| dependabot.yml | repo root | `directory: /` both ecosystems | ✓ WIRED | Pattern found |
| ruleset.json | ci.yml check-run names | contexts == display names | ✓ WIRED | Pattern found + live cross-check |
| branch-protection.md | GitHub rulesets API | re-runnable gh api commands | ✓ WIRED | POST/GET/PUT/PATCH/DELETE forms present; verifier re-ran the GET (403 as recorded) |

### Data-Flow Trace (Level 4)

No UI/rendered data in this phase — the data-flow equivalent for infrastructure: every
gate consumes a real source. lint job ← .golangci.yml (real, schema-verified); setup-go
← go.mod (go-version-file); govulncheck ← go.mod tool graph; check-runs ← real matrix
execution (5/5 success observed on the merge SHA). No static/hollow values found. N/A
for render-style flowing data.

### Behavioral Spot-Checks

All commands verifier-run in a fresh shell; probe files created and removed by the
verifier (plan-sanctioned re-run), tree restored to prior state afterward.

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Build + vet + race tests | `go build ./... && go vet ./... && go test -race ./...` | exit 0; `ok ... internal/app` | ✓ PASS |
| Config schema-valid (silent-no-op guard) | `go tool golangci-lint config verify` | exit 0 | ✓ PASS |
| Strict gate green | `go tool golangci-lint run ./...` | exit 0, zero findings | ✓ PASS |
| Depguard RED — net in internal/app | probe file + `go tool golangci-lint run ./internal/...` | exit 1, `not allowed from list 'sup-kv-app-no-io'` | ✓ PASS (red as required) |
| Depguard RED — third-party in internal/port | same run | `not allowed from list 'sup-kv-port-stdlib-only'` | ✓ PASS (red as required) |
| Gate GREEN after probe removal | `go tool golangci-lint run ./...` | exit 0; no probe paths in `git ls-files` | ✓ PASS |
| Matrix green on merged PR | `gh api commits/f9110ee/check-runs` | 5 names present, 5× success | ✓ PASS |
| Red probe PR disposition | `gh pr view 4` | CLOSED, mergedAt null | ✓ PASS |
| Rulesets endpoint state | `gh api repos/Djarvur/sup-kv/rulesets` | 403 plan-gated (matches recorded deferral) | ✓ PASS (state as recorded) |
| delete_branch_on_merge | `gh api repos/Djarvur/sup-kv` | true | ✓ PASS |
| vulncheck green | `gh run list --workflow=vulncheck.yml` | run 41216077968 success | ✓ PASS |
| First Dependabot PR | `gh pr list --author app/dependabot` | empty (window not elapsed) | ? SKIP → human item |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist in this repo (no scripts directory). The
phase's probes are inline lint runs — executed by this verifier directly (see Behavioral
Spot-Checks, rows 4-6). SUMMARY probe claims were reproduced first-hand, not trusted.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| QUAL-01 | 01-01 | Strict golangci-lint config + go-ultimate conventions govern every package; red gate blocks merge | ✓ SATISFIED | Vendored config byte-verified vs pristine upstream; gate green (verifier-run); boundary red/green re-proven; zero nolint suppressions |
| CI-01 | 01-01, 01-02 | Actions runs quality gates on every push and PR, Linux/macOS, amd64/arm64 | ✓ SATISFIED | push+PR triggers, 4 cells, 5/5 green on merge SHA (live); red teeth via probe PR #4; push-event run on master also green. Merge-blocking *enforcement* is CI-03's protection scope (gap 1) |
| CI-02 | 01-03 | Dependabot current (updates and alerts) + scheduled govulncheck green; no open critical vulns | ✗ PARTIAL — NEEDS HUMAN | Updates configured + live; govulncheck dispatch green (live); alerts GHAS-off (gap 2); first update PR pending (backstop abstention); REQUIREMENTS.md marks Complete — overstated, recommend Partial annotation |
| CI-03 | 01-04 | PR-only development — branch protection on master, direct pushes forbidden | ✗ PARTIAL (annotated in REQUIREMENTS.md) | PR-only practiced and verified (10 PR merges); server-side protection deferred → gap 1; payload + re-apply scripted and verified |

Orphaned requirements: none — REQUIREMENTS.md maps exactly CI-01, CI-02, CI-03, QUAL-01 to Phase 1; all four appear in plan frontmatter requirements.

### Decision Coverage

`check.decision-coverage-verify`: 14/14 trackable CONTEXT.md decisions honored by
shipped artifacts (including D-05 amended — the 3-native + 1-cross matrix
interpretation — and D-10 amended — single-file vendoring after `extends:` was
disproven). Non-blocking gate; no findings.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| .golangci.yml | 155 | `FIXME` string | ℹ️ Info | Upstream vendored content (gocritic's own tag list), not phase debt; the byte-unmodified prohibition forbids "fixing" it |
| .github/dependabot.yml | 7-19 | WR-01: Dependabot gomod PRs will silently desync the 3-way linter pin (go.mod / action version input / vendored config) | ⚠️ Warning | Open review finding (01-REVIEW-DISPOSITION.md): first weekly gomod PR can merge green while breaking local/CI parity; fix = `ignore: dependency-name: github.com/golangci/golangci-lint/v2` |
| AGENTS.md | 5 | IN-01: quotes `go 1.27`, go.mod pins 1.27.1 | ℹ️ Info | Open; harmless self-referential pointer |
| .claude/CLAUDE.md | 17 | IN-02: asserts branch protection as enforced while it is deferred | ⚠️ Warning | Open; actively misleading to agents trusting the generated context — master is protected by convention only |
| ci.yml / vulncheck.yml | — | IN-03: actions pinned by mutable major tags, not SHAs | ℹ️ Info | Open; hardening advice (bounded by contents:read, no secrets, no pull_request_target) |
| .golangci.yml | 68-80 | IN-04: `!$test` exempts app tests from the no-I/O rule; x/sys not in deny list | ℹ️ Info | Open; deliberate per inline comments, documented |
| local master (d7a5054, bfcc1eb) | — | 2 docs commits on local master ahead of origin, outside the PR flow | ⚠️ Warning | PR-only is convention-only (the enforcement gap); pushing master directly would break the practiced contract — route through a PR |
| .planning/REQUIREMENTS.md | CI-02 row | Checkbox Complete while alerts half off | ⚠️ Warning | Traceability overstates; mirror CI-03's Partial annotation pattern |

Debt-marker gate: zero unreferenced TBD/FIXME/XXX in phase files (the single FIXME match is upstream vendored content). No stubs, no empty implementations, no console-log-only code, no hardcoded-hollow data — the repo is 16 tracked implementation files, all substantive.

## Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| internal/app/app_test.go | QUAL-01 (gate target) | 1 | 0 | 0 | Value (`got != want` → t.Errorf) | OK — deliberately trivial per D-01 (walking-skeleton race target, not feature proof); engine property tests are Phase 2 scope |

No disabled tests, no circular fixtures (nothing writes expected values), provenance N/A (no comparison requirement in this phase).

## Human Verification Required

Infrastructure/foundation phase — no user-facing UAT steps invented. Items below are
evidence-gap and ratification items, not UI walkthroughs:

### 1. Ratify or close the branch-protection deferral (Gap 1)

**Test:** Decide: (a) upgrade the Djarvur org plan, run `gh api --method POST repos/Djarvur/sup-kv/rulesets --input docs/setup/ruleset.json`, verify per the doc, flip PENDING→APPLIED; or (b) accept the deferral via the override in this report's frontmatter.
**Expected:** Either master is verifiably protected (direct-push attempt rejected) or the deviation carries explicit human acceptance (accepted_by/at).
**Why human:** The blocker is a billing decision no agent can take; the 2026-10-31 `defer` was auto-selected by the autonomous chain and per ADR-550 needs explicit human resolution.

### 2. Ratify the GHAS-gated Dependabot-alerts caveat (Gap 2)

**Test:** Accept (or reject) that Dependabot alerts stay off until the deferred public flip / GHAS license, with updates + govulncheck as the delivered CI-02 scope.
**Expected:** An override entry, or a GHAS license purchase; CI-02's REQUIREMENTS.md row annotated to match.
**Why human:** External paid-service constraint; conscious risk acceptance is a human call.

### 3. First Dependabot update PR observation (backstop abstention)

**Test:** After Mon 2026-11-02 06:00 UTC, check `gh pr list --author app/dependabot`.
**Expected:** ≥1 grouped gomod or github-actions update PR within the weekly window.
**Why human:** Future-dated observation — the window had not elapsed at verification time; presence checks cannot see a schedule that has not fired.

### 4. Unverified prohibitions — human review recommended (7 items)

All plan must_haves.prohibitions were authored `status: unverified` (spec-less fallback).
Non-authoritative LLM-judge verdicts with verifier-collected evidence — none appear
violated, but each requires human concurrence:

| Plan | Prohibition | Evidence found | Verdict |
|------|-------------|----------------|---------|
| 01 | Vendored config never edited in place | Pristine-fetch diff = pure insertion 64a65,88, zero upstream lines changed | Not violated |
| 01 | No suppressions without written justification | `grep -rn nolint --include="*.go"` → zero directives in repo | Not violated |
| 02 | No gate theater / silently no-oped config | `config verify` exit 0; red probe fires (gate CAN fail); action verify on by default | Not violated (WR-01 records a future desync vector) |
| 02 | Same linter build local and CI (v2.15.0) | ci.yml `version: v2.15.0` == go.mod tool pin v2.15.0 | Not violated currently (WR-01 risk) |
| 03 | No runtime dependency via update PRs | go.mod: 215 requires, all `// indirect`, zero direct | Holds today; future PRs need diff review |
| 03 | No suppressed vulnerability findings | vulncheck.yml runs bare `go tool govulncheck ./...`, no exclusion flags | Not violated |
| 04 | No path-based exemptions in protection | ruleset.json conditions: ref_name only; no path filters | Not violated |

### 5. MVP goal-format discrepancy

**Test:** Run `/gsd:mvp-phase 1` to set a User Story goal, or remove the `Mode: mvp` marker.
**Expected:** Goal format and mode marker agree.
**Why human:** Project-level roadmap decision; `user-story.validate` currently returns false.

## Gaps Summary

The delivered half of Phase 1 is genuinely delivered and adversarially confirmed: the
strict gate is live, green, and proven to have teeth (verifier re-ran the red/green
probe); the 4-cell matrix ran 5/5 green on the merged PR (live API evidence); the
vendored config is byte-pure upstream plus exactly two additive boundary rules (pristine
diff); the skeleton builds/vets/race-tests with a real target; tools are pinned with
zero direct runtime requires; vulncheck is green on demand; Dependabot updates are
configured; delete_branch_on_merge is on; all 13 artifacts and 10 key links check out;
14/14 context decisions honored.

What keeps the goal from fully holding is exactly its enforcement clause — "behind a
red-light-blocking gate": the server-side ruleset is not applied (org-plan 403), so
neither direct-push rejection nor merge-blocking required checks exist, and Dependabot
alerts are GHAS-off. Both are documented, decision-gated deferrals awaiting human
ratification (or funding), not silent failures — but as long as they stand unratified,
roadmap SC-1's second half, SC-2's first half, and SC-3's alerts half are not true, and
the phase goal's red-light-blocking guarantee rests on discipline rather than
enforcement.

---

_Verified: 2026-10-31T16:35:20Z_
_Verifier: Claude (gsd-verifier)_
