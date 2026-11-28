---
phase: 09-documentation-and-release
plan: 08
subsystem: documentation
tags: [readme, examples, proven-claims, doc-anchor-tests, darwin-demotion, create-if-absent, m01-m05, m17-m21, docs-09, tdd]

requires:
  - phase: 09-documentation-and-release (09-01…09-07)
    provides: the green cmd/sup-kv doc-anchor harness (readme_test.go, examples_test.go), docs/API.md's per-RPC error and snapshot tables the EXAMPLES prose cites, docs/DEPLOYMENT.md the install section links, and the OPERATIONS.md proven-claim fixes (09-05) the checker's passing shape rests on
  - phase: 08-performance
    provides: the shipped bounds the named tests back (sim_bound_test.go, stand_red_test.go, storage/version property tests)
provides:
  - cmd/sup-kv/proven_test.go — TestProvenClaims, the DOCS-09 claim gate: every word-boundary prove/proves/proven/proving occurrence in README.md, docs/OPERATIONS.md and CHANGELOG.md must name a backticked `Test…` identifier or a _test.go file inside its blank-line-delimited paragraph, with a ≥3-occurrence vacuity floor; a future bare claim fails make test
  - docs/EXAMPLES.md — M-17 (quoted "SERVING" wait grep), M-18 (the four-service listing), M-19 (the Create-if-absent block: create_if_absent → AlreadyExists with the Unavailable/ResourceExhausted error prose pointing at docs/API.md), M-20 (the full-rollback SnapshotLoad text incl. the whole-restore no-op), M-21 (the Linux-verified platform note with the ADR 0010 darwin statement)
  - cmd/sup-kv/examples_test.go — the bidirectional assertion table with the quoted status shapes and the new Create-if-absent row carrying the state-model ordering rationale
  - README.md — M-01 (## Install from a release: v1.1 archive contract with the v1.0.0 archives as the published example, checksums.txt/SBOM verification), M-02 (both quickstart blocks rm the dumps — the cleanup claim is true), M-03 (the derive line's en0/DHCP behavior and silent fallback stated), M-04 doc half (the archive reader's docs path), M-05 README half + M-31 (every claim names its bound tests; the crossed demo ports named as the same-host trick, the CI darwin cross-compile cell stated), 14.10 (the darwin development-only demotion per ADR 0010's Consequences wording)
  - CHANGELOG.md — the two 1.0.0 proven bullets name the storage/version property tests and the sim_bound tests (the (Phase N) suffix scrub stays 09-10's)
affects: [09-09 (ARCHITECTURE must keep the README's claims consistent), 09-10 (the goreleaser change makes the install section's v1.1 archive contract true; its CHANGELOG scrub edits the same bullets this plan fixed), 09-11 (milestone verification greps the master M-assignment this plan closes rows in)]

actuals:
  tokens: 5300
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "proven-claim gate: word-bounded case-sensitive claim regex + blank-line paragraph scope + backticked `Test[A-Z]…`/`_test.go` evidence extraction + a vacuity floor, so a bare claim fails make test and a scrubbed corpus fails loudly instead of passing vacuously"
    - "block placement by the walkthrough's own state machine: the create_if_absent block sits after the write that makes its AlreadyExists refusal deterministic and stores nothing, so later blocks' state is unchanged — the ordering rationale lives on the assertion-table row"

key-files:
  created:
    - cmd/sup-kv/proven_test.go
    - .planning/phases/09-documentation-and-release/red-evidence/09-08-task1-red.json
  modified:
    - README.md
    - docs/EXAMPLES.md
    - cmd/sup-kv/examples_test.go
    - CHANGELOG.md

key-decisions:
  - "The checker's paragraph-scoped scan is more complete than the plan's two-site RED gloss: it names five bare sites, not two — README.md:21 (proving) and README.md:134 (prove) are real bare claims the docs-audit evidence section had already listed (audit line 260), and both are fixed by naming their bound tests per the plan's own 'every current bare spot' must-have"
  - "Both SERVING assertion rows flip to the quoted shape, not just the Start row: audit M-17 flagged examples_test.go's demo-pair AND check-health rows as NOT_SERVING-satisfiable (:285,:295), so fixing one would leave the weakness the threat model (T-09-23) mitigates"
  - "M-31 is closed here although the plan's task text omitted it: the master M-assignment (09-04-SUMMARY, the table 09-11 greps) assigns M-31 to 09-08, and the README edit set already owned the quickstart prose — the crossed demo ports are named as the same-host trick pointing at DEPLOYMENT.md's one-port real-segment shape, and the demotion paragraph states CI's darwin cells are cross-compile legs"
  - "CHANGELOG.md is edited despite its absence from files_modified: the must_haves and assumptions mandate the 1.0.0 proven-bullet fixes, and the (Phase 2)/(Phase 4) suffixes are deliberately kept for 09-10's scrub — only the evidence names were added"
  - "TestProvenClaims runs t.Parallel() (the strict paralleltest gate): it is a pure doc-file scan with no process state, unlike the serialized doc-block walkthroughs whose nolint justification does not apply"

patterns-established:
  - "claim-evidence adjacency as a CI gate: the claim word and its named test must share one blank-line-delimited paragraph — matching the audit's own reading of a proven claim, and reusable for any future prose surface"

requirements-completed: [DOCS-09]
# DOCS-09 is genuinely complete with this plan: its every clause has landed (proven
# claims name their tests here; the health waits and the macOS pair are green here;
# TestOperationsDoc's binary verification and the rename-coupling landed with 09-05,
# its only sibling declarer, whose summary exists). DOCS-01 and DOCS-10 stay Pending
# on the #2388 shared-ID gate: siblings 09-09/09-10/09-11 declare both and have no
# summaries yet — the ready-ids tool reported all three ready, but the documented
# gate rule (no Complete while a declaring sibling lacks a summary) wins; the
# premature flip was reverted by hand (checkboxes + phase table) and recorded in
# the deviations below. DOCS-01's M rows M-01…M-05/M-17…M-21 and M-31 close here
# while the remaining M rows belong to siblings; DOCS-10's README half (14.10) is
# landed while the archive half is 09-10's.

coverage:
  - id: proven-claim-gate
    description: "TestProvenClaims: word-bounded case-sensitive scan of README/OPERATIONS/CHANGELOG where every claim paragraph must name a backticked Test… identifier or a _test.go file, with the ≥3-occurrence vacuity floor; RED state was five bare sites (README 21/30/134, CHANGELOG 18/28), all fixed"
    requirement: DOCS-09
    verification:
      - kind: unit
        ref: "tests/cmd/sup-kv/proven_test.go#TestProvenClaims"
        status: pass
    human_judgment: false
  - id: examples-m17-21
    description: "EXAMPLES.md closes M-17 (quoted \"SERVING\" wait grep + quoted assertion rows), M-18 (four-service listing), M-19 (Create-if-absent block with AlreadyExists output + Unavailable/ResourceExhausted error prose), M-20 (full-rollback SnapshotLoad text incl. the whole-restore no-op), M-21 (Linux-verified platform note with the ADR 0010 darwin statement); the bidirectional assertion table holds with the new row"
    requirement: DOCS-01
    verification:
      - kind: integration
        ref: "tests/cmd/sup-kv/examples_test.go#TestExamplesDoc"
        status: pass
      - kind: other
        ref: "grep: '\"SERVING\"', create_if_absent, AlreadyExists, Unavailable, ResourceExhausted, deleted all present in docs/EXAMPLES.md"
        status: pass
    human_judgment: false
  - id: readme-m01-m05-demotion
    description: "README closes M-01 (Install from a release section with checksums.txt/SBOM steps), M-02 (rm cleanup in both marked blocks), M-03 (derive-line behavior), M-04 doc half, M-05 README half, M-31 (crossed ports + CI darwin cross-compile cell) and the 14.10 darwin development-only demotion per ADR 0010; the macOS quickstart pair keeps its option-b loud-skip contract"
    requirement: DOCS-10
    verification:
      - kind: integration
        ref: "tests/cmd/sup-kv/readme_test.go#TestReadmeQuickstart"
        status: pass
      - kind: other
        ref: "grep: 'in preparation' absent; 'Install from a release', 'rm -f supkv-a.json', 'development-only', 'ADR 0010', 'checksums.txt' present"
        status: pass
    human_judgment: false
  - id: install-section-fact-accuracy
    description: "The install section's factual claims (v1.0.0 archive contents as the published example; SPDX SBOMs produced by syft; the v1.1 archive contract stated as the going-forward shape ahead of 09-10's goreleaser change) are prose grounded in the audit's archive listing and .goreleaser.yaml's sboms/checksum blocks — no executable block asserts them"
    requirement: DOCS-10
    verification: []
    human_judgment: true
    rationale: "The section describes the v1.1 archive contract that 09-10's goreleaser change must make true before the release lands; its accuracy against the eventual release artifact is checkable only once that artifact exists (09-10 verify-only + the release procedure's own archive listing)."
  - id: changelog-proven-bullets
    description: "The CHANGELOG 1.0.0 entry's two proven bullets name the storage/version property tests and the sim_bound tests; the (Phase N) suffixes are deliberately left for 09-10's scrub"
    requirement: DOCS-09
    verification:
      - kind: unit
        ref: "tests/cmd/sup-kv/proven_test.go#TestProvenClaims"
        status: pass
    human_judgment: false

# Metrics
duration: 40min
completed: 2026-11-28
status: complete
---

# Phase 9: Documentation and Release — Plan 08 Summary

**The proven-claim rule is now a CI gate (TestProvenClaims), and the two pages a newcomer reads are honest: README's install/cleanup/macOS/demotion fixes (M-01…M-05, M-31, 14.10) and EXAMPLES' health/state/platform/restore fixes (M-17…M-21) with the new create_if_absent walkthrough.**

## Performance

- **Duration:** 40 min
- **Started:** 2026-11-28T00:05:00Z
- **Completed:** 2026-11-28T00:45:00Z
- **Tasks:** 3
- **Files modified:** 5 (plus the RED evidence record)

## Accomplishments

- TestProvenClaims (cmd/sup-kv/proven_test.go) makes DOCS-09's claim rule executable: every word-boundary prove/proves/proven/proving occurrence in README.md, docs/OPERATIONS.md and CHANGELOG.md must carry a backticked `Test…` name or a _test.go reference in its own blank-line-delimited paragraph, with a ≥3-occurrence vacuity floor. RED ran against the live tree and named five bare sites; the doc fixes turned it green, and any future bare claim fails make test.
- docs/EXAMPLES.md: the health wait greps the quoted `"SERVING"` form and both SERVING assertion rows expect the quoted answer shape, so NOT_SERVING can satisfy neither (M-17); the service-count sentence states the actual four entries (M-18); a new marked Create-if-absent block after the Write-and-read block refuses demo/hello with AlreadyExists (deterministic by the walkthrough's own state, stores nothing) and the page's error prose names Unavailable and ResourceExhausted with pointers to docs/API.md's tables (M-19); the SnapshotLoad text states the full-rollback contract — keys absent from the snapshot are deleted, the result broadcasts, and this example's immediate reload is a whole-restore no-op — citing docs/API.md (M-20); the platform note at the top of the executable section states the CI-verified Linux scope and the darwin development-only status per ADR 0010 (M-21).
- README.md: the stale "release artifacts are in preparation" sentence is replaced by an ## Install from a release section (GitHub releases page; the v1.1 Linux-archive contract stated as the going-forward shape with the v1.0.0 archives — binary + CHANGELOG + LICENSE + README, darwin included — named as the published example; checksums.txt via shasum -c and the per-archive syft SBOM's purpose); both marked quickstart blocks now rm the dump files so "The block cleans up after itself" is true with the claim kept; the macOS derive-line description states the hard-coded en0 DHCP query and its silent 127.255.255.255 fallback; the darwin development-only demotion follows ADR 0010's Consequences wording (build/unit tests supported, replication untested on darwin, archives Linux-only, CI darwin cells are cross-compile legs) while the macOS quickstart pair keeps its option-b loud-skip contract; the docs-links paragraph gives the archive reader their path; every proven claim names its bound tests; the crossed demo ports are named as the same-host trick (M-31).
- CHANGELOG.md: the 1.0.0 entry's two proven bullets name the storage/version property tests (`TestMerge_laws`, `TestMerge_lawsTombstones`, `TestProp_mergeSmallSpace`, `TestVersion_byteOrder`) and the sim_bound tests; the (Phase N) suffixes stay for 09-10's scrub.

## Task Commits

Each task was committed atomically:

1. **Task 1: RED — the proven-claim checker** - `4018225` (test) — TestProvenClaims + the vacuity guard, failing naming the five bare sites; RED evidence record classified RED_EVIDENCE_OK
2. **Task 2: EXAMPLES M-17…M-21 and the create_if_absent walkthrough** - `013bff7` (docs) — docs/EXAMPLES.md + the bidirectional examplesAssertions updates; TestExamplesDoc green under -race
3. **Task 3: README install/cleanup/macOS/demotion + every proven claim named (GREEN)** - `d45a957` (feat) — README.md + CHANGELOG.md + the t.Parallel() lint fix in proven_test.go; TestProvenClaims, TestReadmeQuickstart and the full package green

**Plan metadata:** see the docs(09-08) complete commit (SUMMARY + STATE + ROADMAP + REQUIREMENTS).

_Note: the plan's TDD cycle spans Tasks 1 and 3 — the RED commit (test) is followed by the GREEN commit (feat) whose "implementation" is the doc fixes the checker demands._

## Files Created/Modified

- `cmd/sup-kv/proven_test.go` — TestProvenClaims: the DOCS-09 claim gate (word-boundary scan, paragraph scope, backticked-Test/_test.go evidence, vacuity floor)
- `docs/EXAMPLES.md` — M-17…M-21: quoted health wait, four-service listing, the Create-if-absent block and error prose, full-rollback restore text, the Linux platform note
- `cmd/sup-kv/examples_test.go` — the assertion table: quoted `"status": "SERVING"` shapes, the Create-if-absent row with the state-model ordering rationale
- `README.md` — M-01…M-05, M-31, 14.10: install-from-release section, block cleanup, derive-line truth, darwin demotion, named evidence
- `CHANGELOG.md` — the 1.0.0 proven bullets name their tests
- `.planning/phases/09-documentation-and-release/red-evidence/09-08-task1-red.json` — the classified RED record

## Decisions Made

See the key-decisions frontmatter: the checker's five-site RED state (two sites beyond the plan's gloss), both SERVING rows quoted, M-31 folded in per the master M-assignment, CHANGELOG edited despite the files_modified omission with the phase suffixes preserved for 09-10, and t.Parallel() on the pure-scan checker.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Two additional bare proven-claim sites beyond the plan's RED state**
- **Found during:** Task 1 (the RED run)
- **Issue:** The plan's RED description named README.md:30 and the CHANGELOG 1.0.0 entry; the spec'd scan also found README.md:21 (the Status paragraph's "proving") and README.md:134 (the macOS paragraph's "prove") — both bare, both already listed in the docs-audit evidence section (audit line 260)
- **Fix:** Both paragraphs name their bound tests (stand_red_test.go / sim_bound_test.go for the status suite; readme_test.go for the macOS CI claim) in Task 3
- **Files modified:** README.md
- **Verification:** TestProvenClaims green; the RED log named all five sites
- **Committed in:** d45a957 (Task 3 commit)

**2. [Rule 1 - Bug] The "Check health" assertion row was also NOT_SERVING-satisfiable**
- **Found during:** Task 2 (examplesAssertions update)
- **Issue:** Audit M-17 flagged both SERVING rows (demo-pair and check-health) as substring-satisfiable by NOT_SERVING; the plan's task text named only the Start row
- **Fix:** Both rows expect the quoted `"status": "SERVING"` shape
- **Files modified:** cmd/sup-kv/examples_test.go
- **Verification:** TestExamplesDoc green
- **Committed in:** 013bff7 (Task 2 commit)

**3. [Rule 2 - Missing critical] M-31 had no task despite the master M-assignment's 09-08 row**
- **Found during:** Task 3 planning (the read-first pass over 09-04-SUMMARY's master table)
- **Issue:** 09-04's master M-assignment assigns M-31 (README topology wording: crossed demo ports as the same-host trick; the CI darwin cross-compile cell) to 09-08, and 09-11's completeness check greps that table; no plan task mentioned it
- **Fix:** Folded into Task 3's README edits — the quickstart explanation names the crossed-ports trick with a pointer to DEPLOYMENT.md's one-port real-segment shape, and the demotion paragraph states the CI darwin cross-compile legs
- **Files modified:** README.md
- **Verification:** TestReadmeQuickstart green; the README wording present
- **Committed in:** d45a957 (Task 3 commit)

**4. [Rule 2 - Missing critical] CHANGELOG.md missing from files_modified while the must_haves mandate its fix**
- **Found during:** Task 3
- **Issue:** The plan frontmatter lists four files, but the must_haves and assumptions mandate naming tests in the CHANGELOG 1.0.0 proven bullets (M-05's closure), and Task 3's verify requires TestProvenClaims green
- **Fix:** The two bullets gained their bound test names; the (Phase 2)/(Phase 4) suffixes are untouched for 09-10's scrub
- **Files modified:** CHANGELOG.md
- **Verification:** TestProvenClaims green
- **Committed in:** d45a957 (Task 3 commit)

**5. [Rule 1 - Bug] The ready-ids tool flipped shared requirements prematurely**
- **Found during:** plan close-out (update_requirements step)
- **Issue:** requirements.ready-ids reported DOCS-01 and DOCS-10 ready although siblings 09-09/09-10/09-11 declare both and have no summaries — contradicting the #2388 shared-ID gate's documented rule (no Complete until every declaring plan has finished)
- **Fix:** requirements.mark-complete ran on the tool's verdict, then the DOCS-01/DOCS-10 flips were reverted by hand (checkboxes and phase table back to Pending); DOCS-09 stayed Complete (its only sibling declarer 09-05 has a summary, and its clauses are all landed)
- **Files modified:** .planning/REQUIREMENTS.md
- **Verification:** git diff shows exactly the DOCS-01/DOCS-10 rows reverted; DOCS-09 remains [x]
- **Committed in:** the plan-metadata commit

---

**Total deviations:** 5 auto-fixed (3 bug, 2 missing-critical; all plan-text or tooling gaps against the plan's own must_haves, the committed master assignment, or the documented shared-ID rule)
**Impact on plan:** All five fixes close gaps the plan's own must_haves, the 09-04 master table or the #2388 rule demand; no scope creep — every edit lands in the plan's declared deliverable surfaces.

## Issues Encountered

None — the RED state was exactly diagnosable, the create_if_absent refusal was deterministic at the chosen block position, and both full gates (make lint, make test) passed on the committed tree.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Ready for 09-09 (ARCHITECTURE/ADR index/simulator guide/CONTRIBUTING/SECURITY): the README and EXAMPLES pages this plan fixed are stable targets for its cross-links
- 09-10 must land the goreleaser change that makes the install section's v1.1 archive contract true (binary + docs/DEPLOYMENT.md + sup-kv.service + docs, Linux-only) and owns the CHANGELOG (Phase N) scrub over the bullets this plan fixed
- 09-11's completeness check can grep the master M-assignment: M-01, M-02, M-03, M-31 and the M-04/M-05/M-17/M-18/M-19/M-20/M-21 halves named as 09-08's are closed by commits 4018225, 013bff7 and d45a957

---
*Phase: 09-documentation-and-release*
*Completed: 2026-11-28*
