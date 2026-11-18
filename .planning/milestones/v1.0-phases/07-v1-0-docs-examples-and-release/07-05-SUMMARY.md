---
phase: 07-v1-0-docs-examples-and-release
plan: 05
subsystem: infra
tags: [release, git-tag, github-release, goreleaser, sbom, branch-protection, ruleset]

requires:
  - phase: 07-v1-0-docs-examples-and-release (07-04)
    provides: the snapshot-proven release pipeline (goreleaser config, release.yml, CHANGELOG.md header source)
  - phase: 07-v1-0-docs-examples-and-release (07-02, 07-03)
    provides: the merged docs deliverables (README quickstart, EXAMPLES.md, OPERATIONS.md) riding in the tagged tree
provides:
  - Annotated git tag v1.0.0 on master squash 0ead1e8 (the repo's first tag — Pitfall 9)
  - GitHub Release v1.0.0, published (not draft), 9 assets with CHANGELOG.md as the header
  - Shipped-artifact identity proof: release binary --version = "sup-kv v1.0.0 0ead1e8 go1.27.4"
  - D-11 ruleset re-apply attempted and honestly recorded (still plan-gated, re-verified 2026-11-17)
affects: [milestone v1.0 close-out (/gsd-verify-work, /gsd-complete-milestone)]

actuals:
  tokens: 981     # chars/4 over the realized diff (3926 bytes) — plan estimated 20000
  tasks: 2
  commits: 1      # MEASURED: git rev-list --count 07f24a3..5d3c891 (Task 1 produced no repo commit — tag/Release are runtime artifacts)
plan_head_before: 07f24a36c9ad35e87e7a45c2dbc32e51855a4b0d
plan_head_after: 5d3c8917ae169d8fea6e6f5e778893bf7a8567f6

tech-stack:
  added: []
  patterns:
    - "Release evidence as runtime facts: tag SHA + workflow run ID + asset list + --version line recorded verbatim — the release artifacts are the deliverable, not repo files"

key-files:
  created: []
  modified:
    - docs/setup/branch-protection.md
    - .planning/STATE.md

key-decisions:
  - "Tag target is the master squash commit 0ead1e8 (PR #32 merge), never the branch head — the plan's tagging contract, after verifying the branch's non-planning tree is byte-identical to master (empty diff)"
  - "CHANGELOG 1.0.0 date 2026-11-17 equals the tag date — no amendment PR needed (the plan's step 1 contingency did not fire)"
  - "Ruleset 403 recorded honestly: POST verbatim + read-side both return 'Upgrade to GitHub Pro or make this repository public to enable this feature.' — marker re-verified 2026-11-17, CI-03 protection sub-item stays plan-gated"
  - "Enforcement probe (trivial direct push) deliberately NOT run on the 403 path — with no ruleset live it would actually land on master and violate the PR-only contract it exists to prove"

patterns-established:
  - "Publish-directly release (Open Question 3 resolution held): the first tag ran the publish form with no draft, observed to success — flip-to-draft stays available only if formatting problems appear"

requirements-completed: [DOCS-03]

coverage:
  - id: D1
    description: "v1.0.0 shipped end-to-end: annotated tag on a five-green-checks master commit, release workflow success, Release with 9 assets, shipped-binary --version identity proven"
    requirement: DOCS-03
    verification:
      - kind: other
        ref: "gh release view v1.0.0 --json assets --jq '.assets | length' == 9 (4 archives + 4 SBOMs + checksums.txt)"
        status: pass
      - kind: other
        ref: "downloaded sup-kv_1.0.0_Linux_x86_64.tar.gz; ./sup-kv --version | grep -F 'sup-kv v1.0.0 ' — full line: sup-kv v1.0.0 0ead1e8 go1.27.4"
        status: pass
    human_judgment: false
  - id: D2
    description: "D-11 ruleset re-apply attempted verbatim, read-side verified, marker updated honestly in either direction (403 path taken)"
    verification:
      - kind: other
        ref: "gh api POST + read both 403 (exact error captured in branch-protection.md); grep -Eq 'APPLIED|re-verified 2026-' docs/setup/branch-protection.md passes; zero 'Status: APPLIED' claims — marker matches live state"
        status: pass
    human_judgment: false

duration: 88min
completed: 2026-11-17
status: complete
---

# Phase 07 Plan 05: Ship v1.0.0 Summary

**v1.0.0 shipped from a five-green-checks master (annotated tag → squash 0ead1e8; Release with 9 assets; release binary reports `sup-kv v1.0.0 0ead1e8 go1.27.4`) and the D-11 ruleset re-apply honestly recorded as still plan-gated — 403 re-verified 2026-11-17**

## Performance

- **Duration:** 88 min (wall-clock, includes one mid-run session interruption after the tag push)
- **Started:** 2026-11-17T17:32:45Z
- **Completed:** 2026-11-17T19:00:50Z
- **Tasks:** 2
- **Files modified:** 2 (docs/setup/branch-protection.md, .planning/STATE.md)

## Accomplishments

- Precondition re-proven on the NEW master HEAD: PR #32 state=MERGED (squash `0ead1e8e67bb48ae73c078143c3de4f625518f64`), all five ci.yml checks green on that exact SHA (watched run 44137814529 to completion); branch-vs-master non-planning diff verified EMPTY — the squash contains everything release-relevant through 07f24a3, CHANGELOG.md identical, so the tag target is the master SHA per plan
- Full local gate green (`make lint` + `make test`) on the byte-identical tree; no tags existed locally or remotely (Pitfall 9 re-checked), CHANGELOG 1.0.0 date 2026-11-17 = tag date — no amendment PR
- **v1.0.0 shipped**: annotated tag `v1.0.0` (tag object `ea57bfd`, dereferences to `0ead1e8`) pushed; release workflow run 44138907991 concluded **success**; GitHub Release v1.0.0 published (not draft) with the CHANGELOG.md header and **exactly 9 assets**
- **Shipped identity proven on the artifact**: downloaded `sup-kv_1.0.0_Linux_x86_64.tar.gz` from the Release, extracted to a fresh temp dir, ran the binary — output `sup-kv v1.0.0 0ead1e8 go1.27.4` (the Pitfall 6 `{{ .Tag }}` parity decision, byte-shape as pinned in 07-04)
- **D-11 closed honestly**: ruleset POST + read-side both 403 (exact error below); `docs/setup/branch-protection.md` marker updated to "still plan-gated, re-verified 2026-11-17" with the exact error line; both STATE.md decision entries added; CI-03's protection sub-item deliberately NOT touched (stays plan-gated — the plan's prohibition)

## Release Verification Evidence (the plan's mandated record)

**Tag chain:** local `git tag -a v1.0.0 -m "sup-kv v1.0.0" 0ead1e8…` → pushed → `git ls-remote --tags origin`: `ea57bfdbcf9ea446e1eb05df99a3e685bda066ab refs/tags/v1.0.0`, dereferenced `0ead1e8e67bb48ae73c078143c3de4f625518f64 refs/tags/v1.0.0^{}` — the master squash commit, five checks green.

**Workflow:** run `44138907991` (workflow `release`, event `push`, headBranch `v1.0.0`): status completed, conclusion **success**.

**Release assets (9/9, names match the archive template):**

```
checksums.txt
sup-kv_1.0.0_Darwin_arm64.tar.gz
sup-kv_1.0.0_Darwin_arm64.tar.gz.sbom.json
sup-kv_1.0.0_Darwin_x86_64.tar.gz
sup-kv_1.0.0_Darwin_x86_64.tar.gz.sbom.json
sup-kv_1.0.0_Linux_arm64.tar.gz
sup-kv_1.0.0_Linux_arm64.tar.gz.sbom.json
sup-kv_1.0.0_Linux_x86_64.tar.gz
sup-kv_1.0.0_Linux_x86_64.tar.gz.sbom.json
```

**Strongest identity check (performed, not just asset-listed):** downloaded the Linux x86_64 archive via `gh release download`, extracted, executed:

```
$ /tmp/supkv-rel-0705/sup-kv --version
sup-kv v1.0.0 0ead1e8 go1.27.4
```

`v1.0.0` (v-prefixed `{{ .Tag }}`), `0ead1e8` (`{{ .ShortCommit }}` — matches the tag target), `go1.27.4` (workflow toolchain). The archive also ships CHANGELOG.md, LICENSE, README.md alongside the binary.

## Ruleset Verify Output (D-11, the plan's mandated record)

Apply command verbatim from the doc, then read-side verify — both return the org-plan gate:

```
$ gh api --method POST repos/Djarvur/sup-kv/rulesets --input docs/setup/ruleset.json
{"message":"Upgrade to GitHub Pro or make this repository public to enable this feature.",...,"status":"403"}  (exit 1)

$ gh api repos/Djarvur/sup-kv/rulesets --jq '...'
{"message":"Upgrade to GitHub Pro or make this repository public to enable this feature.",...,"status":"403"}  (exit 1)
```

Outcome recorded in `docs/setup/branch-protection.md`: `## Status: DEFERRED (accepted override)` retained, first line now "**The ruleset is NOT yet applied — still plan-gated, re-verified 2026-11-17.**", plus a dated re-apply block carrying the exact error line and the deliberate no-probe rationale. Remediation (per the ratified CLAUDE.md override): GitHub Pro org plan or public repo flip — a maintainer decision, not an executor action.

## Task Commits

1. **Task 1: Phase gate on master, tag v1.0.0, observe the release workflow, verify the Release assets** — **no repo commit** (plan-conformant): CHANGELOG date did not drift (step 1 contingency unfired) and the deliverables are runtime artifacts (the tag and the Release, evidenced above). Tag object `ea57bfd` on master `0ead1e8`.
2. **Task 2: D-11 ruleset re-apply — honest attempt, read-side verify, marker updated** - `5d3c891` (docs)

**Plan metadata:** recorded after this file (docs commit below).

## Files Created/Modified

- `docs/setup/branch-protection.md` - DEFERRED block updated: dated re-verification + exact 403 error line + no-probe rationale
- `.planning/STATE.md` - two decision entries (ruleset outcome; v1.0.0 release facts)
- `.planning/REQUIREMENTS.md` - DOCS-03 flipped Complete (shared-ID gate: this plan was the last declaring plan — `requirements.ready-ids` returned ready)
- `.planning/WINDOWS.md` - entry id 9 (07-04's unrun-verify: dispatch proof deferred to 07-05) resolved `fixed` — the real tag run was observed

## Decisions Made

- Tag the master squash SHA, never the branch head — verified safe by the empty non-planning diff (the plan's expected case; the coordinator's stop-condition did not fire)
- Enforcement probe skipped on the 403 path: a trivial direct push with no ruleset live would SUCCEED and land outside the PR-only contract — the probe is success-path-only by the plan's own wording
- WINDOWS entry 9 closed as `fixed`: the entry's own text ("07-05 observes the real tag run post-merge") is exactly what Task 1 did

## Deviations from Plan

None - plan executed exactly as written. (Task 1's step-1 CHANGELOG amendment and step-4 failure-recovery paths are contingencies that did not fire; the interruption mid-Task-1 cost wall-clock only — the tag/Release state was verified intact before continuing.)

## Issues Encountered

None. One mid-run session interruption after the tag push; the resumed session re-verified every claimed fact first-hand (workflow conclusion, asset list, tag chain) before relying on it.

## Authentication Gates

None. (gh CLI authenticated throughout.)

## User Setup Required

None - no external service configuration required.

## Known Stubs

None.

## Next Phase Readiness

- All 5 of 5 phase-07 plans complete; Phase 07 done → milestone v1.0 is ready for `/gsd-verify-work 7` and `/gsd-complete-milestone`
- Open WINDOWS entries riding to phase verification: 3, 5, 6, 7, 8 (D-16-era residuals, macos-15 named-skip leg, OPERATIONS.md manual UAT, examples interruption residual) — none block this plan's claims
- DOCS-01 remains Pending in REQUIREMENTS.md by design: its macos-15 quickstart leg is a named skip under the option-b verdict (07-02's recorded adjudication point)

## Self-Check: PASSED

- Files exist: docs/setup/branch-protection.md — FOUND; commits are ancestors of HEAD: 5d3c891 — FOUND
- Task 1 acceptance criteria: tag on green master + workflow success ✓; 9 assets incl. checksums.txt ✓; --version line recorded ✓
- Task 2 acceptance criteria: POST run verbatim + read-side output recorded ✓; marker matches live state ✓; both STATE decision entries ✓

---
*Phase: 07-v1-0-docs-examples-and-release*
*Completed: 2026-11-17*
