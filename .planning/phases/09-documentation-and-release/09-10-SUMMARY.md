---
phase: 09-documentation-and-release
plan: 10
subsystem: documentation
tags: [changelog, keep-a-changelog, proto-scrub, goreleaser, darwin-demotion, sbom, syft, release-procedure, dead-code]

requires:
  - phase: 09-documentation-and-release (09-08)
    provides: the TestProvenClaims gate the 1.1.0 entry must satisfy and the README half of the ADR 0010 darwin demotion
  - phase: 09-documentation-and-release (09-09)
    provides: the CONTRIBUTING release section that deferred its release-procedure pointer to this plan
  - phase: 09-documentation-and-release (09-07)
    provides: docs/DEPLOYMENT.md and deploy/sup-kv.service — the artifacts the archives' files: contract ships
  - phase: 08-performance
    provides: the perf gate and test names (TestPerfBurstDistinctKeys) the changelog cites
provides:
  - CHANGELOG.md with the hand-written 1.1.0 entry, the seven (Phase N) suffixes scrubbed, [Unreleased] empty and the compare footers updated — the release body's source, proven-claim-checked
  - both protos comment-scrubbed (DOC-21 two-proto acceptance) with the committed supkv.pb.go regenerated as a comment-only diff
  - .goreleaser.yaml with goos:[linux] (ADR 0010/D-06 demotion), the archive files: contract shipping docs/DEPLOYMENT.md and deploy/sup-kv.service, and the sliced release header verified offline against the real CHANGELOG.md
  - docs/setup/release-procedure.md — the seven-step executed-claim procedure with the one-way tag step reserved for 09-11
  - the Q-09 dead-code verification record (D-09/XPORT-13 closure evidence): zero survivors
  - the CONTRIBUTING.md release-procedure pointer 09-09 deferred to this plan
affects: [09-11 (executes this procedure: gates on master, the snapshot re-run, the one-way tag, the release-body verification)]

actuals:
  tokens: 6000
  tasks: 4
  commits: 4

tech-stack:
  added: []
  patterns:
    - "goreleaser release-header slicing: mustReadFile + split (strings.Split, content-first) over the `## [<version>]` heading markers, verified offline with a text/template scratch program before any release relies on it"
    - "archives files: replaces the default license*/readme*/changelog* globs in goreleaser v2 (defaults apply only when files is empty), so defaults are listed explicitly beside deployment artifacts"
    - "machine-external binary provisioning: a checksum-verified GitHub-release download mirroring the CI action's own pin, never a package-manager install and never --skip=sbom"

key-files:
  created:
    - docs/setup/release-procedure.md
  modified:
    - CHANGELOG.md
    - internal/udprepl/replproto/supkv/v1/supkv.proto
    - internal/udprepl/replproto/supkv/v1/supkv.pb.go
    - .goreleaser.yaml
    - CONTRIBUTING.md

key-decisions:
  - "the archive files: list is explicit (README.md, LICENSE, CHANGELOG.md, docs/DEPLOYMENT.md, deploy/sup-kv.service) because defining files: replaces goreleaser v2's default globs — verified in the pinned v2.20.2 archive pipe source (archive.go:83-89); explicit entries keep the archive paths predictable (README/LICENSE/CHANGELOG at the root, the guide under docs/, the unit under deploy/)"
  - "the release header slices the [1.1.0] section via the planning-proven template and the two version markers are documented in the file as the next release's bump points, discharging the old whole-file header's own promise; the offline render over the real CHANGELOG.md produced exactly the 1.1.0 section (recorded below) — the whole-file fallback never needed"
  - "syft v1.54.0 was provisioned before the first snapshot as a checksum-verified GitHub-release download into $HOME/.local/bin (mirroring release.yml's anchore/sbom-action/download-syft@v0.24.3 pin), because the sboms pipe shells out to syft and --skip=sbom is forbidden — checksums.txt hashes the SBOM files"
  - "CONTRIBUTING.md gains the release-procedure pointer although the plan's files_modified omitted it: 09-09's recorded decision states CONTRIBUTING defers the pointer to 09-10 and this plan adds it when the procedure lands (Rule-2 classed deviation, recorded below)"

requirements-completed: [DOCS-01, DOCS-10]

coverage:
  - id: D1
    description: "CHANGELOG 1.1.0 entry in the 1.0.0 prose style with the (Phase N) scrub and updated compare footers"
    requirement: DOCS-10
    verification:
      - kind: unit
        ref: "cmd/sup-kv/proven_test.go#TestProvenClaims"
        status: pass
      - kind: other
        ref: "grep gate: '^## [1.1.0]' present, zero '(Phase [0-9]+)' occurrences, '^[1.1.0]:' footer, '^## [Unreleased]' intact"
        status: pass
    human_judgment: false
  - id: D2
    description: "Both protos free of decision/phase citations with the technical statements kept; supkv.pb.go regenerated as a comment-only diff"
    requirement: DOCS-10
    verification:
      - kind: other
        ref: "grep gate: no D-[0-9]/phase/plan 0/decision 14 in either proto, CRC statement present; make proto + make buf-lint + git diff --check green; non-comment pb.go diff lines == 0"
        status: pass
    human_judgment: false
  - id: D3
    description: ".goreleaser.yaml: goos [linux] demotion with ADR reference, archive files: shipping the deployment artifacts, sliced release header verified offline"
    requirement: DOCS-10
    verification:
      - kind: other
        ref: "go tool goreleaser release --snapshot --clean (exit 0): exactly 2 Linux archives, no darwin, tar tzf lists sup-kv + docs/DEPLOYMENT.md + deploy/sup-kv.service + README/LICENSE/CHANGELOG per archive, checksums.txt verified OK, SBOMs from syft-1.54.0"
        status: pass
      - kind: other
        ref: "offline header render (/tmp/sup-kv-09-10-render.go with mustReadFile+strings.Split): output starts '## [1.1.0]', contains '### Added', no 'Unreleased', no '## [1.0.0]'"
        status: pass
      - kind: other
        ref: "ci.yml darwin legs untouched (grep darwin/macos: macos-15 native + darwin-amd64-cross present)"
        status: pass
    human_judgment: false
  - id: D4
    description: "docs/setup/release-procedure.md — the seven-step executed-claim procedure with the one-way tag marker"
    requirement: DOCS-10
    verification:
      - kind: other
        ref: "grep gate: release-snapshot, ONE-WAY, tar tzf, checksums.txt, gh release view v1.1.0, Unreleased all present"
        status: pass
      - kind: other
        ref: "executed claims recorded from the 2026-11-28 snapshot at fbda859: snapshot exit 0 in 25s, dist/ listing, per-archive tar tzf, shasum -a 256 -c all OK, syft provisioning OK"
        status: pass
    human_judgment: false
  - id: D5
    description: "Q-09 dead-code verification record (D-09/XPORT-13 closure evidence) — the five probes"
    requirement: DOCS-01
    verification:
      - kind: other
        ref: "probe gate: lastLocal 0, ownHandle 0, func Parse in internal/app/persist 0, cover identifier 0 (five prose-comment verb matches per research §8), fsync_darwin.go gone — DEADCODE-GATE-OK"
        status: pass
    human_judgment: false
  - id: D6
    description: "CONTRIBUTING.md release-procedure pointer (the 09-09 deferral discharged)"
    requirement: DOCS-10
    verification:
      - kind: other
        ref: "grep: docs/setup/release-procedure.md linked from CONTRIBUTING.md's Release section; no Go test pins CONTRIBUTING.md content (verified before edit)"
        status: pass
    human_judgment: false

duration: 18 min
completed: 2026-11-28
status: complete
---

# Phase 9 Plan 10: Release Mechanics Summary

**The hand-written 1.1.0 changelog, the two-proto comment scrub with comment-only regeneration, the goreleaser darwin demotion with the deployment-artifact archive contract and the offline-verified sliced release header, the release-procedure document, and the Q-09 zero-survivor verification.**

## Performance

- **Duration:** 18 min
- **Started:** 2026-11-28T01:04:00Z
- **Completed:** 2026-11-28T01:22:00Z
- **Tasks:** 4
- **Files modified:** 6 (1 created, 5 modified)

## Accomplishments

- CHANGELOG.md carries the hand-written `## [1.1.0] - 2026-11-28` entry directly under
  the empty `[Unreleased]`: the spec-compliance/resilience narrative, then
  Added/Changed/Fixed bullets in the 1.0.0 prose style — every proven claim names its
  tests (`TestRed_R01_pacedSync20k`, `TestRed_R02_burst`, `TestRed_R02_unackedDrain`,
  `TestRed_R03_gateClosedUntilFullList`, `TestRed_R03_f14ReplacedNode`,
  `TestRed_P01_valueLimit`, `TestRed_P01_valueLimitEngine`, `TestPerfBurstDistinctKeys`,
  `TestEngine_burstBoundedConfirmation`, `TestBound_*` — all name-verified against the
  tree before writing). The seven `(Phase N)` suffixes are scrubbed from the 1.0.0
  entry; `[Unreleased]` re-targets the compare link at v1.1.0; the `[1.1.0]` footer
  joins `[1.0.0]`.
- Both protos are decision/phase-number free (DOC-21): the public
  `api/supkvapi/v1/supkvapi.proto` verified clean and untouched; the internal
  replproto's two citations removed with the technical statements kept
  ("decoded by the receiver only after the CRC gate passes" / "the adapter maps every
  field onto the port structs"). `make proto` regenerated the committed
  `supkv.pb.go` as a strictly comment-only diff (2 lines; non-comment diff lines: 0);
  `make buf-lint` and `git diff --check` green.
- `.goreleaser.yaml` implements D-05/D-06: `goos: [linux]` (two Linux archives:
  x86_64 + arm64) with the ADR 0010/demotion comment stating that ci.yml keeps both
  darwin legs; the archives block's explicit `files:` ships
  `docs/DEPLOYMENT.md` and `deploy/sup-kv.service` beside the explicitly listed
  README/LICENSE/CHANGELOG defaults; the release header is the slicing template whose
  bump-point comment replaces the old whole-file promise.
- The snapshot gate ran green end to end with syft v1.54.0 provisioned first:
  `go tool goreleaser release --snapshot --clean` exit 0; `dist/` holds exactly the
  two Linux archives + one SPDX SBOM each + checksums.txt + metadata; `tar tzf` of
  both archives lists `sup-kv`, `docs/DEPLOYMENT.md`, `deploy/sup-kv.service`,
  `README.md`, `LICENSE`, `CHANGELOG.md`; `shasum -a 256 -c checksums.txt` from a
  dist/ copy: all four lines OK; SBOM creator `Tool: syft-1.54.0`. No darwin artifact.
- The header slice was verified offline against the REAL CHANGELOG.md before relying
  on it — the recorded render output (starts `## [1.1.0] - 2026-11-28`, carries the
  milestone paragraph and `### Added`/`### Changed`/`### Fixed`, ends at the last
  Fixed bullet, contains neither `[Unreleased]` nor `## [1.0.0]`):
  the template is
  `{{ $full := mustReadFile "CHANGELOG.md" }}{{ $p1 := split $full "## [1.1.0]" }}{{ $p2 := split (index $p1 1) "\n## [1.0.0]" }}{{ printf "## [1.1.0]%s" (index $p2 0) }}`.
- Q-09 dead-code verification recorded (D-09 / XPORT-13 closure evidence) — the five
  probes on the current tree, all zero survivors, verify-then-record with no
  deletions needed (expected per research §8):
  1. `grep -rn 'lastLocal' --include='*.go' .` → 0 matches
  2. `grep -rn 'ownHandle' --include='*.go' .` → 0 matches
  3. `grep -n 'func Parse' internal/app/persist/*.go` → 0 matches
  4. `grep -rn '\bcover\b' internal/app/persist/*.go internal/app/repl/core.go |
     grep -v '_test\|the cover discipline\|coverage'` → 5 matches, all comment prose
     using the verb ("Eight slots cover the", "cover the whole key space under
     buffer loss", "cover the key domain", "cover the line"), zero identifier uses —
     the identifier-vs-prose distinction research §8 states
  5. `ls internal/app/persist/fsync_darwin.go` → No such file or directory
- docs/setup/release-procedure.md lands in the branch-protection executed-claim style
  ("every command below is re-runnable verbatim"): prerequisites with the syft
  v1.54.0 assertion and the checksum-verified provisioning block; gates on master;
  the `make release-snapshot` gate; archive inspection (`ls dist/`, `tar tzf` per
  archive, `shasum -a 256 -c checksums.txt` from a dist/ copy); **the tag marked THE
  ONE-WAY STEP** (`git tag -a v1.1.0 -m "sup-kv v1.1.0" && git push origin v1.1.0`);
  release verification (`gh release view v1.1.0` — assets match step 4's list, body
  has neither `[Unreleased]` nor `(Phase `); post-release link resolution. The doc
  records the executed 2026-11-28 snapshot outputs (at fbda859) and states that the
  v1.1.0-run transcript is filled by the release plan. It also states the serial
  order, the snapshot's re-runnability, and that a failed snapshot leaves `dist/`
  inspectable for diagnosis.

## Task Commits

Each task was committed atomically:

1. **Task 1: CHANGELOG — the 1.1.0 entry, the suffix scrub, and the footer links** - `848a9c3` (docs)
2. **Task 2: Proto comment scrub in both protos with comment-only regeneration** - `fbda859` (docs)
3. **Task 3: .goreleaser.yaml — darwin demotion, archive files:, and the sliced release header** - `f224add` (feat)
4. **Task 4: Q-09 dead-code verification and the release-procedure document** - `6ac33ea` (docs)

**Plan metadata:** committed with the STATE/ROADMAP/REQUIREMENTS update (docs(09-10)).

## Files Created/Modified

- `CHANGELOG.md` — the 1.1.0 entry, the scrub, the footers
- `internal/udprepl/replproto/supkv/v1/supkv.proto` — the two citation removals
- `internal/udprepl/replproto/supkv/v1/supkv.pb.go` — regenerated, comment-only diff
- `.goreleaser.yaml` — demotion, files:, sliced header
- `docs/setup/release-procedure.md` — new, the seven-step procedure
- `CONTRIBUTING.md` — the release-procedure pointer (see Deviations 1)

## Decisions Made

- **Explicit archive `files:` list** (README.md, LICENSE, CHANGELOG.md,
  docs/DEPLOYMENT.md, deploy/sup-kv.service): verified in the pinned goreleaser
  v2.20.2 source that defaults apply only when `files` is empty
  (archive.go:83-89), so the defaults must be listed explicitly — archive paths
  stay predictable and DOC-05's `tar tzf` acceptance reads the real entry set.
- **The sliced header with documented bump points:** the template's two version
  markers are named in the file comment as the next release's edit points,
  discharging the old header comment's own "from the second release on, slice"
  promise; the whole-file fallback was never needed.
- **syft provisioning shape:** a checksum-verified v1.54.0 GitHub-release download
  (tarball + checksums file, `shasum -a 256 -c` verified, `OK`) installed to
  `$HOME/.local/bin`, mirroring release.yml's action pin — the T-09-SC disposition
  (no npm/pip/cargo; `--skip=sbom` never a workaround).
- **The D-06 decision checkpoint was discharged as written:** the decision is
  pre-recorded and user-approved in 09-CONTEXT (darwin development-only per
  ADR 0010; darwin archives leave the asset set; CI keeps the compile/vet legs);
  the plan states no further decision is required and the irreversible publication
  act is gated to 09-11's publication checkpoint. This plan implemented exactly the
  recorded shape.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] CONTRIBUTING.md release-procedure pointer added**
- **Found during:** Task 4 (the release-procedure document)
- **Issue:** the plan's `files_modified` omits CONTRIBUTING.md, but 09-09's recorded
  decision states CONTRIBUTING defers its release-procedure pointer to 09-10 and
  "09-10 lands the procedure and adds the pointer" — skipping it leaves that
  cross-plan contract unfulfilled
- **Fix:** appended the pointer sentence to CONTRIBUTING.md's Release section,
  linking docs/setup/release-procedure.md; verified first that no Go test pins
  CONTRIBUTING.md content (grep over *.go: zero references)
- **Files modified:** CONTRIBUTING.md
- **Verification:** the link resolves to the file committed in the same task commit
- **Committed in:** `6ac33ea` (Task 4 commit)

---

**Total deviations:** 1 auto-fixed (1 missing-critical per the recorded 09-09
deferral). **Impact on plan:** a one-sentence doc pointer closing a sibling plan's
recorded deferral; no scope creep.

## Issues Encountered

- syft was not installed (the plan's declared machine precondition, not a config
  red): provisioned before the first snapshot per Task 3's precondition — the
  checksum-verified GitHub-release download printed `OK` and `syft version`
  reported `Version:       1.54.0`. The provisioning block is recorded verbatim in
  the procedure doc's prerequisites step.
- The post-install `rm -rf "$DL" "$EX"` in the provisioning one-liner left the shell
  cwd deleted (exit 1 after the successful install); harmless — `syft version` had
  already printed the pinned version. The procedure doc's version of the block
  `cd`s out first.

## User Setup Required

None - no external service configuration required. (The syft binary now on PATH is
machine tooling, recorded in the procedure doc.)

## Next Phase Readiness

- Ready for 09-11 (the final plan): the release tooling is verified offline, the
  procedure exists and drives only existing targets, the changelog body is
  proven-claim-checked, and the archive contract is real — 09-11 runs the gates on
  master, re-runs the snapshot, executes THE ONE-WAY tag step behind its publication
  checkpoint, and verifies the release body (the header render output above is the
  shape the body must match).
- DOCS-01/DOCS-10 REQUIREMENTS.md flips are held by the #2388 shared-ID gate:
  09-11 also declares both and has no summary yet, so this plan marks nothing in
  REQUIREMENTS.md (per the recorded #2388 rule and the 09-08 close-out precedent);
  the last declaring plan's close-out re-evaluates.
- Machine notes for 09-11: syft v1.54.0 is on PATH at `$HOME/.local/bin/syft`;
  `dist/` currently holds the fbda859 snapshot artifacts (re-`--clean`ed by the next
  snapshot run); `.planning` red-evidence untracked directories predate this plan.

## Self-Check: PASSED

All task acceptance criteria re-run on the committed tree:

- Task 1: `go test -race -count=1 ./cmd/sup-kv/ -run '^TestProvenClaims$'` PASS;
  grep gate `## [1.1.0]` + zero `(Phase N)` + `[1.1.0]:` footer + `## [Unreleased]` OK
- Task 2: two-proto grep chain OK; `make proto` + `make buf-lint` + `git diff --check`
  green; pb.go non-comment diff lines 0
- Task 3: syft `Version: 1.54.0` OK; snapshot rc=0 with exactly 2 archives, no
  darwin, DEPLOYMENT.md + sup-kv.service in both archives; static grep gate OK
  (`goos: [linux]`, mustReadFile, 1.1.0, files:, ci.yml darwin legs); offline render
  gate OK
- Task 4: dead-code probe gate OK (zero survivors); procedure doc grep gate OK
- Plan-level: `make lint` green on the committed tree (config verify + strict run,
  no issues); `make test` left to 09-11's gate-on-master step (the full race suite),
  with the affected doc gate (TestProvenClaims) run per task here

---
*Phase: 09-documentation-and-release*
*Completed: 2026-11-28*
