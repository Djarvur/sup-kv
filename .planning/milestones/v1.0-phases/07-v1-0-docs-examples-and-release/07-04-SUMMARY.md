---
phase: 07-v1-0-docs-examples-and-release
plan: 04
subsystem: infra
tags: [goreleaser, release-engineering, changelog, keep-a-changelog, github-actions, syft, sbom]

requires:
  - phase: 07-v1-0-docs-examples-and-release (07-01)
    provides: goreleaser v2.20.2 go.mod tool directive, the pinned-tool invocation convention
provides:
  - CHANGELOG.md — Keep-a-Changelog 2.0.0 with the curated 1.0.0 entry (the GitHub Release header source)
  - .goreleaser.yaml — 4-target build, checksums.txt, per-archive SBOMs, CHANGELOG.md release header
  - .github/workflows/release.yml — tag (v*) + workflow_dispatch (snapshot) triggered, contents:write, pinned syft action
  - Makefile release / release-snapshot targets; .gitignore /dist/ anchor
  - Local snapshot proof of the full artifact pipeline (D-09 pre-tag gate) — only the tag push remains for 07-05
affects: [07-05 (ship: tag, observe Release assets, --version identity)]

actuals:
  tokens: 1935     # chars/4 over the realized diff (7742 bytes, 5 files) — plan estimated 30000
  tasks: 2
  commits: 2       # MEASURED: git rev-list --count 80bef63..ac14f92
plan_head_before: 80bef63bdb20995c6b61ea540c246807c735edfb1
plan_head_after: ac14f92088141babae74e4a83504356190334497

tech-stack:
  added: []          # no new modules — goreleaser v2.20.2 already pinned by 07-01; syft v1.57.2 was an ad-hoc local one-off (GOBIN=/tmp), not a repo dependency
  patterns:
    - "Release header via template: release.header: '{{ mustReadFile \"CHANGELOG.md\" }}' — the v2.20.2 form (header is a plain template string; the from_file map shape is gone)"
    - "ldflags identity parity: {{ .Tag }} (v-prefix kept) + {{ .ShortCommit }} matches the Makefile's git-describe stamping — one --version shape for make and release builds"

key-files:
  created:
    - CHANGELOG.md
    - .goreleaser.yaml
    - .github/workflows/release.yml
  modified:
    - Makefile
    - .gitignore

key-decisions:
  - "release.header uses the mustReadFile template — goreleaser v2.20.2's schema (verified against the pinned module source) removed the from_file map form the plan text was written against"
  - "{{ .Tag }} identity decision implemented as pinned: release binaries report sup-kv v1.0.0 <shortcommit> go1.27.4, byte-shape-identical to make build at the same commit"
  - "Local snapshot proven WITH syft (ad-hoc v1.57.2, GOBIN=/tmp one-off per the plan's sanctioned option) instead of --skip=sbom — the SBOM half of the pipeline is proven locally too, not only at tag time"
  - "Optional pre-tag gh workflow dispatch proof skipped — the workflow file must exist on the default branch for workflow_dispatch; 07-05 observes the real tag run post-merge"

patterns-established:
  - "Tag-triggered release workflow separate from ci.yml: check-run names are a ruleset contract, release.yml never runs as a PR check (Pitfall 4)"

requirements-completed: [DOCS-03]

coverage:
  - id: D1
    description: "CHANGELOG.md — Keep-a-Changelog 2.0.0 with one curated 1.0.0 Added section over Phases 2-6"
    requirement: DOCS-03
    verification:
      - kind: other
        ref: "grep -Fq suite over CHANGELOG.md (1.0.0, Unreleased, ### Added, 1.0.0 tag link, keepachangelog.com/en/2.0.0)"
        status: pass
    human_judgment: true
    rationale: "Structural elements are machine-proven, but the curation itself is editorial (wording/truthfulness cross-checked against ROADMAP phases 2-6 by the executor); the maintainer should read the 1.0.0 entry before the tag makes it the public Release header"
  - id: D2
    description: "goreleaser config + tag-triggered release workflow + Makefile targets + /dist/ anchor, ci.yml untouched"
    verification:
      - kind: other
        ref: "grep -Fq suites over .goreleaser.yaml and release.yml; git diff --exit-code .github/workflows/ci.yml; PHONY grep; grep -Fxq '/dist/'; git check-ignore dist/; go tool goreleaser check (exit 0)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Snapshot proof: make release-snapshot builds all 4 targets + checksums.txt (+SBOMs) into dist/; ldflags stamping verified via --version"
    verification:
      - kind: other
        ref: "make release-snapshot (exit 0); find dist -name '*.tar.gz' | wc -l == 4; test -f dist/checksums.txt; extracted binary --version prints 'sup-kv v0.0.0 1db9265 go1.27.4'"
        status: pass
    human_judgment: false

duration: 13min
completed: 2026-11-17
status: complete
---

# Phase 07 Plan 04: Release Pipeline Summary

**Keep-a-Changelog 2.0.0 CHANGELOG.md plus the goreleaser v2.20.2 release pipeline (4-target archives, checksums.txt, per-archive syft SBOMs, tag-triggered workflow with CHANGELOG.md as Release header) — the whole pipeline snapshot-proven locally, SBOM pipe included, before any tag exists**

## Performance

- **Duration:** 13 min
- **Started:** 2026-11-17T13:36:26Z
- **Completed:** 2026-11-17T13:49:51Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- CHANGELOG.md: KaC 2.0.0-pinned preamble, empty `[Unreleased]`, one curated `### Added` section covering Phases 2-6 exactly as shipped, oldest-version tag link + Unreleased compare link (no tags exist yet — Pitfall 9 held: `git tag -l` empty)
- The complete release pipeline: `.goreleaser.yaml` (v2; sup-kv build linux+darwin × amd64+arm64, CGO_ENABLED=0, `{{ .Tag }}`/`{{ .ShortCommit }}` ldflags, checksums.txt, per-archive SBOMs, CHANGELOG.md release header), `.github/workflows/release.yml` (push tags v* + workflow_dispatch snapshot, contents:write, checkout@v7 fetch-depth 0, setup-go@v7 go-version-file, pinned download-syft@v0.24.3, `go tool goreleaser` invocation), Makefile `release`/`release-snapshot` targets, `/dist/` gitignore anchor
- ci.yml byte-identical (`git diff --exit-code` green) — the five ruleset check contexts are untouched (Pitfall 4)
- Mandatory snapshot proof green WITH syft: 4 archives + 4 SBOMs + checksums.txt in dist/, and the extracted binary reports the stamped identity — the artifact pipeline is proven end-to-end before the tag (D-09/Pitfall 8)

## Snapshot Run Evidence (the plan's mandated record)

`make release-snapshot` (goreleaser v2.20.2 via go.mod tool directive, syft v1.57.2 ad-hoc on PATH, exit 0, 28s):

```
dist/sup-kv_0.0.0-SNAPSHOT-1db9265_Linux_x86_64.tar.gz
dist/sup-kv_0.0.0-SNAPSHOT-1db9265_Linux_arm64.tar.gz
dist/sup-kv_0.0.0-SNAPSHOT-1db9265_Darwin_x86_64.tar.gz
dist/sup-kv_0.0.0-SNAPSHOT-1db9265_Darwin_arm64.tar.gz
(+ 4 × .sbom.json, checksums.txt [SHA256 over all 8 artifacts], artifacts.json, config.yaml, metadata.json)
```

Recorded `--version` output of the extracted snapshot binary (linux/amd64):

```
sup-kv v0.0.0 1db9265 go1.27.4
```

`v0.0.0` is goreleaser's snapshot pseudo-tag flowing through `{{ .Tag }}` (v-prefixed by design), `1db9265` is `{{ .ShortCommit }}` — the ldflags wiring took effect (no `dev`). On the real v1.0.0 tag this prints `sup-kv v1.0.0 <shortcommit> go1.27.4`, matching `make build` at the same commit — the conscious Pitfall 6 pick recorded in `.goreleaser.yaml`'s header comment.

## Task Commits

1. **Task 1: CHANGELOG.md — Keep-a-Changelog 2.0.0, the curated v1.0.0 entry** - `1db9265` (docs)
2. **Task 2: goreleaser config + tag-triggered workflow + Makefile targets + snapshot proof** - `ac14f92` (chore)

**Plan metadata:** recorded after this file (docs commit below).

## Files Created/Modified

- `CHANGELOG.md` - KaC 2.0.0 changelog; the GitHub Release header source
- `.goreleaser.yaml` - release pipeline config (v2 schema)
- `.github/workflows/release.yml` - tag/dispatch-triggered release workflow
- `Makefile` - + `release`, `release-snapshot` targets and .PHONY entries
- `.gitignore` - `dist/` anchored to `/dist/` (goreleaser output dir)

## Decisions Made

- `release.header` implemented as `header: '{{ mustReadFile "CHANGELOG.md" }}'` — see Deviations (the plan's `from_file` literal was written against a removed API form)
- Local snapshot proven WITH syft rather than `--skip=sbom`: installing syft ad-hoc (`GOBIN=/tmp/syft-bin-07-04 go install github.com/anchore/syft/cmd/syft@v1.57.2`, documented one-off outside the repo, not a dependency) exercises the SBOM pipe locally too, so both plan-sanctioned halves are covered and the stronger one is recorded
- Optional `gh workflow run release` dispatch proof skipped: workflow_dispatch requires the workflow on the default branch and this branch is unmerged; 07-05 observes the real tag run post-merge (WINDOWS ledger carries the open entry)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `release.header.from_file` removed in goreleaser v2.20.2**
- **Found during:** Task 2 (goreleaser config authoring)
- **Issue:** The plan (and RESEARCH Pattern 7, citing older docs) mandates `release.header.from_file: CHANGELOG.md`; the pinned goreleaser v2.20.2 rejects it — `goreleaser check` fails with `cannot unmarshal !!map into string` because `Release.Header` is a plain template string in the v2 schema (verified against the pinned module source, `pkg/config/config.go:708`)
- **Fix:** `header: '{{ mustReadFile "CHANGELOG.md" }}'` — `mustReadFile` is a built-in goreleaser template func (verified in the pinned module's tmpl func map) that reads the file at release time and errors loudly if absent; intent identical (Release header pulls from CHANGELOG.md)
- **Files modified:** `.goreleaser.yaml`
- **Verification:** `go tool goreleaser check` exit 0; the header mechanism grep (`mustReadFile "CHANGELOG.md"`) passes. The plan's literal verify element `grep -Fq 'from_file: CHANGELOG.md'` cannot pass alongside a working config — it is the single verify line superseded by the deviation; every other element of all five verify commands passes as written
- **Committed in:** ac14f92 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — plan text written against a removed goreleaser API form)
**Impact on plan:** Intent fully preserved (Release header pulls from CHANGELOG.md); no scope change. The pipeline is validated by `goreleaser check` plus the green snapshot run, which is stronger evidence than the grep.

## Issues Encountered

None beyond the deviation above. Notes: syft is not on PATH locally (expected — it is a workflow-installed tool); resolved via the plan's sanctioned ad-hoc install rather than `--skip=sbom`, so the SBOM pipe is proven locally as well.

## Authentication Gates

None.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None — the pipeline is fully wired; nothing placeholder.

## Next Phase Readiness

- Everything 07-05 needs exists: CHANGELOG.md (Release header source), the proven pipeline, `make release-snapshot` as the pre-tag gate
- Remaining for 07-05: merge to master, push tag v1.0.0, observe the release workflow publish archives + checksums + SBOMs, spot-check `--version` on a release archive, D-11 ruleset re-apply
- The CHANGELOG's 1.0.0 date (2026-11-17) is the expected tag date; per plan text, 07-05 amends it in the pre-tag check if the actual tag drifts more than a day

## Self-Check: PASSED

- Files exist: CHANGELOG.md, .goreleaser.yaml, .github/workflows/release.yml, Makefile, .gitignore — all FOUND
- Commits are ancestors of HEAD: 1db9265, ac14f92 — both FOUND
- All task acceptance criteria re-run green (see Deviations for the one superseded grep literal)

---
*Phase: 07-v1-0-docs-examples-and-release*
*Completed: 2026-11-17*
