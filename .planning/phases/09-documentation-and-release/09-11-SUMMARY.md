---
phase: 09-documentation-and-release
plan: 11
subsystem: documentation
tags: [release, goreleaser, snapshot, archive-inspection, checksums, sbom, syft, git-tag, red-runner]

requires:
  - phase: 09-documentation-and-release (09-10)
    provides: docs/setup/release-procedure.md, the demoted .goreleaser.yaml, the sliced CHANGELOG 1.1.0 header and the checksum-verified syft v1.54.0 provisioning this plan re-asserts
  - phase: 09-documentation-and-release (09-04)
    provides: the M-01…M-52 master assignment the completeness grep consumes
  - phase: 09-documentation-and-release (09-01…09-09)
    provides: every machine check the cmd/sup-kv doc-anchor suite runs together on the final tree
provides:
  - the executed full-gate record on the final tree (lint, race+cover suite, doc-anchor suite, FORMATS guard, buf gates, client-example, red-runner verdict) with the 52/52 M-completeness proof
  - the merge record (PR #37 → master, merge commit 9f7a318, CI green on all six checks) and the ON-MASTER/clean-tree precondition proof
  - the release snapshot built on merged master and inspected against the archive contract (2 Linux archives, no darwin, DEPLOYMENT.md + systemd unit inside, checksums OK, SPDX-2.3 SBOMs from syft-1.54.0)
  - the staged publication: every tag precondition demonstrably true, awaiting only the maintainer's explicit confirmation
affects: [Task 3 continuation (the one-way tag, release verification, phase close), the v1.1 milestone exit]

actuals:
  tokens: 4200
  tasks: 2
  commits: 1

tech-stack:
  added: []
  patterns:
    - "checkpoint-shaped release execution: everything automatable runs to green, the summary records each command with its exit code and observed output, and the single irreversible act (the tag push) is left to the maintainer's explicit confirmation"

key-files:
  created:
    - .planning/phases/09-documentation-and-release/09-11-SUMMARY.md
  modified: []

key-decisions:
  - "red-runner disposition (maintainer): the R-01 row's powersave miss is a host-speed condition (CPU powersave ~1.8 GHz, recorded per 03-12); under the performance governor (maintainer-authorized temporary switch, restored after) scripts/red-runner ran 7/7 green with both mutants killed on 2026-11-28 (~06:00Z). The red-evidence run directories are DELETED by maintainer decision, not committed — the verdict lives in this summary's prose, no evidence directory is fabricated."
  - "the summary record is committed on local master one commit past the release commit 9f7a318; the tag must therefore name a commit on origin/master — land this docs commit via the PR-only flow before tagging, or tag 9f7a318 explicitly — surfaced in the checkpoint state for the orchestrator/maintainer to decide"

requirements-completed: [DOCS-01, DOCS-08, DOCS-10]

coverage:
  - id: D1
    description: "The full gate is green on the committed final tree: make lint, make test (-race, 19 pkgs), the cmd/sup-kv doc-anchor suite (registry gate with zero unclosed rows), the FORMATS byte-identity guard, buf lint+breaking, client-example"
    requirement: DOCS-01
    verification:
      - kind: other
        ref: "make lint && make test && go test -race -count=1 ./cmd/sup-kv/ && go test -race -count=1 ./internal/app/persist/ -run TestFormatsDoc && make buf-lint && make buf-breaking && make client-example — all exit 0 (Task 1 segment, 2026-11-28)"
        status: pass
    human_judgment: false
  - id: D2
    description: "M-completeness: every M-01…M-52 row's disposition appears in exactly one of the ten 09-xx plan summaries"
    requirement: DOCS-01
    verification:
      - kind: other
        ref: "grep gate over .planning/phases/09-documentation-and-release/09-0[1-9]-SUMMARY.md + 09-10-SUMMARY.md — observed marker COMPLETE-52-52"
        status: pass
    human_judgment: false
  - id: D3
    description: "Red-runner verdict matches the frozen expectation table (7/7 rows green, both mutants killed)"
    requirement: DOCS-01
    verification:
      - kind: other
        ref: "scripts/red-runner run of 2026-11-28 ~06:00Z under the performance governor (recorded from the Task 1 segment; evidence directories deleted per maintainer disposition, verdict recorded in this summary's prose)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The release procedure executed through step 4: merged-master precondition (clean tree, HEAD ancestor of origin/master), syft v1.54.0 asserted, make release-snapshot green, archive inspection passed (2 Linux archives, no darwin, DEPLOYMENT.md + sup-kv.service in each, checksums verified, SPDX-2.3 SBOMs)"
    requirement: DOCS-08
    verification:
      - kind: other
        ref: "git merge-base --is-ancestor HEAD origin/master (exit 0) && syft version (1.54.0) && make release-snapshot (exit 0, 29s) && ls dist/*.tar.gz | wc -l (2) && tar tzf inspection && (cd dist && shasum -a 256 -c checksums.txt) — all green, outputs recorded below"
        status: pass
    human_judgment: false
  - id: D5
    description: "The v1.1.0 tag is pushed on merged master and the GitHub Release verified (assets match the inspection, clean 1.1.0 body, binary version identity) — the milestone's terminal artifact"
    requirement: DOCS-10
    verification: []
    human_judgment: true
    rationale: "The tag push is the one irreversible act (D-06/D-08); this plan is autonomous: false and the publication checkpoint requires the maintainer's explicit confirmation before git push origin v1.1.0. Execution is staged and every precondition is demonstrably true; the deliverable is intentionally unexecuted at this stop."

duration: ~25 min (resumed segment; Task 1's gates ran and were recorded earlier this session, red-runner ~06:00Z)
completed: 2026-11-28
status: halted
---

# Phase 9: Documentation and Release — Plan 11 Summary

**v1.1.0 release staged on merged master 9f7a318 — full gate green, snapshot built and inspected against the archive contract, stopped at the publication checkpoint awaiting the maintainer's tag confirmation.**

## Performance

- **Duration:** ~25 min (this resumed segment; Task 1's gates were executed and recorded earlier in the same session, the red-runner verdict at ~06:00Z)
- **Completed:** 2026-11-28T06:25Z (halted at the publication checkpoint — Task 3 intentionally not started)
- **Tasks:** 2 (Task 1 gates + red-runner disposition; Task 2 merge check + snapshot + inspection)
- **Files modified:** 1 (this summary)

## Accomplishments

- The full gate set is green on the final tree (recorded with the Task 1 segment): `make lint`; `make test` (-race, 19 packages); `go test -race -count=1 ./cmd/sup-kv/` (the doc-anchor suite including the registry gate — zero unclosed rows); the FORMATS byte-identity guard (`go test -race -count=1 ./internal/app/persist/ -run TestFormatsDoc`); `make buf-lint && make buf-breaking`; `make client-example`. The M-completeness grep reported **COMPLETE-52-52** over the ten 09-xx summaries.
- The red-runner disposition is RESOLVED by maintainer decision: the R-01 row's miss under powersave is a host-speed condition (CPU powersave ~1.8 GHz, recorded per 03-12). Under the performance governor (maintainer-authorized temporary switch, restored after) `scripts/red-runner` ran **7/7 green with both mutants killed** on 2026-11-28 (~06:00Z). The red-evidence run directories are DELETED (maintainer decision), not committed; the verdict lives in this prose and no evidence directory is fabricated.
- The merge precondition holds: PR #37 (`gsd/v1.1-milestone` → `master`) MERGED with CI green on all six checks (lint, gate ubuntu-latest, gate ubuntu-24.04-arm, gate macos-15, gate darwin-amd64-cross, client-example); the follow-up darwin fix `f6bb477` (udprepl SO_REUSEPORT) is included; origin/master is at merge commit `9f7a318`.
- Task 2 executed green end-to-end on merged master: clean tree, ON-MASTER ancestor test passed, syft v1.54.0 asserted, `make release-snapshot` exit 0 in 29s, and the archive inspection passed every contract item with outputs recorded verbatim below.
- The publication checkpoint is reached with all three preconditions demonstrably true; the one-way act (`git tag -a v1.1.0 -m "sup-kv v1.1.0" && git push origin v1.1.0`) awaits the maintainer's explicit confirmation.

## Task Commits

No task commits — Tasks 1 and 2 execute gates, the merge check and the snapshot, producing no code changes. The sole commit of this plan is this summary record itself (committed immediately after this file is written; the hash is reported in the checkpoint return). The release commit the tag names is `9f7a318`.

## The executed record

### Task 1 — full gate on the committed tree (Task 1 segment, 2026-11-28)

| Command | Result |
|---|---|
| `make lint` | exit 0 — config verify + strict golangci-lint run, no issues |
| `make test` | exit 0 — race+coverage suite green over 19 packages |
| `go test -race -count=1 ./cmd/sup-kv/` | exit 0 — doc-anchor suite incl. the registry gate, zero unclosed rows |
| `go test -race -count=1 ./internal/app/persist/ -run TestFormatsDoc` | exit 0 — FORMATS byte-identity guard |
| `make buf-lint && make buf-breaking` | exit 0 |
| `make client-example` | exit 0 |
| M-completeness grep (M-01…M-52 over the ten summaries) | `COMPLETE-52-52` |
| `scripts/red-runner` | 7/7 green, both mutants killed — under the performance governor (maintainer-authorized, restored after; ~06:00Z). The powersave R-01 miss is a host-speed condition per 03-12. Evidence directories deleted per maintainer disposition. |

### Task 2 — merge check and the release snapshot (this segment)

Merge state — the local checkout moved to merged master:

```
$ git switch master
Switched to branch 'master'
Your branch is behind 'origin/master' by 670 commits, and can be fast-forwarded.
$ git pull --ff-only
Updating d6b2a75..9f7a318
Fast-forward
...
$ git status --porcelain | wc -l | grep -q '^0$' && echo TREE-CLEAN
TREE-CLEAN
$ git log --oneline -3
9f7a318 Merge pull request #37 from Djarvur/gsd/v1.1-milestone
f6bb477 fix(udprepl): WithAddressReuse sets SO_REUSEPORT on darwin
f7c03ea docs(09-10): update state, roadmap and requirements for the release mechanics plan
$ git merge-base --is-ancestor HEAD origin/master && echo ON-MASTER
ON-MASTER
```

Syft precondition re-asserted:

```
$ command -v syft
/home/nil/.local/bin/syft
$ syft version | grep -E '^Version:'
Version:       1.54.0
```

The release snapshot (log at `/tmp/sup-kv-09-11-snapshot.log`, 32 lines):

```
$ make release-snapshot          # exit 0
  • building snapshot...                           version=1.0.0-SNAPSHOT-9f7a318
    • building                                       paths=cmd/sup-kv binaries=sup-kv target=linux_amd64_v1
    • building                                       paths=cmd/sup-kv binaries=sup-kv target=linux_arm64_v8.0
  • archives
    • archiving                                      name=dist/sup-kv_1.0.0-SNAPSHOT-9f7a318_Linux_arm64.tar.gz
    • archiving                                      name=dist/sup-kv_1.0.0-SNAPSHOT-9f7a318_Linux_x86_64.tar.gz
  • software bill of materials
    • cataloging                                     cmd=syft sbom=sup-kv_1.0.0-SNAPSHOT-9f7a318_Linux_arm64.tar.gz.sbom.json
    • cataloging                                     cmd=syft sbom=sup-kv_1.0.0-SNAPSHOT-9f7a318_Linux_x86_64.tar.gz.sbom.json
      • took: 25s
  • calculating checksums
  • writing artifacts metadata
  • release succeeded after 29s
  • thanks for using GoReleaser!
```

The archive inspection — `ls dist/`:

```
$ ls dist/
artifacts.json
checksums.txt
config.yaml
metadata.json
sup-kv_1.0.0-SNAPSHOT-9f7a318_Linux_arm64.tar.gz
sup-kv_1.0.0-SNAPSHOT-9f7a318_Linux_arm64.tar.gz.sbom.json
sup-kv_1.0.0-SNAPSHOT-9f7a318_Linux_x86_64.tar.gz
sup-kv_1.0.0-SNAPSHOT-9f7a318_Linux_x86_64.tar.gz.sbom.json
sup-kv_linux_amd64_v1
sup-kv_linux_arm64_v8.0
$ ls dist/*.tar.gz | wc -l
2
$ ls dist/ | grep -i darwin        # no output — grep exit 1
```

Archive contents — identical six entries in BOTH archives:

```
$ tar tzf dist/sup-kv_*Linux_x86_64.tar.gz
CHANGELOG.md
LICENSE
README.md
deploy/sup-kv.service
docs/DEPLOYMENT.md
sup-kv
$ tar tzf dist/sup-kv_*Linux_arm64.tar.gz
CHANGELOG.md
LICENSE
README.md
deploy/sup-kv.service
docs/DEPLOYMENT.md
sup-kv
```

Checksum and SBOM integrity — the plan's in-dist form and the procedure's copy-based form both verified:

```
$ (cd dist && shasum -a 256 -c checksums.txt)
sup-kv_1.0.0-SNAPSHOT-9f7a318_Linux_arm64.tar.gz: OK
sup-kv_1.0.0-SNAPSHOT-9f7a318_Linux_arm64.tar.gz.sbom.json: OK
sup-kv_1.0.0-SNAPSHOT-9f7a318_Linux_x86_64.tar.gz: OK
sup-kv_1.0.0-SNAPSHOT-9f7a318_Linux_x86_64.tar.gz.sbom.json: OK
$ rm -rf /tmp/sup-kv-release-check && cp -r dist /tmp/sup-kv-release-check
$ cd /tmp/sup-kv-release-check && shasum -a 256 -c checksums.txt   # all four lines: OK
$ grep -o '"spdxVersion":"[^"]*"' sup-kv_1.0.0-SNAPSHOT-9f7a318_Linux_x86_64.tar.gz.sbom.json
"spdxVersion":"SPDX-2.3"
$ grep -o 'Tool: syft-[0-9.]*' sup-kv_1.0.0-SNAPSHOT-9f7a318_Linux_x86_64.tar.gz.sbom.json
Tool: syft-1.54.0
```

The full Task 2 verify chain ran atomically once more and printed `ON-MASTER` then `TASK2-VERIFY-CHAIN: ALL GREEN`.

## Decisions Made

- Red-runner disposition per maintainer decision (recorded above): governor-run verdict accepted as the gate record; evidence directories deleted, not committed; the verdict lives in this prose.
- The summary record is committed one commit past `9f7a318` on local master (the plan's only commit). Consequence surfaced at the checkpoint: before `git push origin v1.1.0`, this docs commit must either land on origin/master through the repo's PR-only flow, or the tag must name `9f7a318` explicitly — the orchestrator/maintainer decides at the checkpoint.

## Deviations from Plan

None — plan executed exactly as written. The red-runner governor re-run is the maintainer's recorded disposition of the host-speed condition, not an execution deviation.

## Issues Encountered

None. The snapshot's `1.0.0-SNAPSHOT-9f7a318` naming is the expected snapshot-mode form (no tag exists yet); the tagged release swaps it for `1.1.0` as the procedure documents.

## User Setup Required

None — no external service configuration required.

## Checkpoint state (PUBLICATION CHECKPOINT — reached, not passed)

The tag is the one irreversible act (D-06/D-08). Its three preconditions are all demonstrably true:

1. **All gates green** — the Task 1 record above (including the red-runner verdict and COMPLETE-52-52).
2. **Phase branch merged to origin/master** — PR #37 merged with CI green on all six checks; HEAD `9f7a318` is an ancestor of origin/master (`ON-MASTER`).
3. **Snapshot archive inspection passed** — 2 Linux archives, no darwin, `docs/DEPLOYMENT.md` + `deploy/sup-kv.service` + README/LICENSE/CHANGELOG + binary in each, checksums verified OK, SBOMs SPDX-2.3 from syft-1.54.0.

Awaiting: the maintainer's explicit confirmation to run `git tag -a v1.1.0 -m "sup-kv v1.1.0" && git push origin v1.1.0` (which fires `.github/workflows/release.yml`), after the local summary commit's placement on origin/master is decided (PR-only flow first, or tag `9f7a318` explicitly). If declined, the release stops here with everything staged — the snapshot is re-runnable and no tag is pushed.

Task 3 (tag, publish, verify the release, and close this summary with the milestone statement) stays untouched until that confirmation arrives via the orchestrator.

---
*Phase: 09-documentation-and-release*
*Completed: 2026-11-28 (halted at the publication checkpoint; Task 3 pending maintainer confirmation)*
