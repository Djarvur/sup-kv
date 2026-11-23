---
schema_version: 1
open_count: 7
waived_count: 1
fixed_count: 5
total_count: 13
last_updated: 2026-11-23T07:51:08.600Z
---

# Broken Windows Ledger

> Cross-phase defect register. With `workflow.windows_enforce` enabled, `/gsd-ship` blocks while `open_count > 0`.
> Waive with `gsd-tools windows waive <id> "<reason>"` (reason required).
> Mark fixed with `gsd-tools windows fixed <id>`.

| id | phase | kind | file | line | description | status | reason | recorded_at | resolved_at |
|----|-------|------|------|------|-------------|--------|--------|-------------|-------------|
| 1 | 01 | unmet-truth | .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-03-SUMMARY.md |  | Backstop truth D5: Dependabot's first version-update PR not yet observed — weekly schedule window has not elapsed since dependabot.yml landed (2026-10-31) | fixed |  | 2026-10-31T15:45:51.348Z | 2026-11-02T15:55:45.509Z |
| 2 | 01 | unmet-truth | docs/setup/branch-protection.md | 14 | CI-03 protection sub-item deferred (org plan 403, decision defer 2026-10-31): direct pushes to master are not server-side rejected until the ruleset is applied; PENDING marker records the one-command re-apply path | waived | Ratified accepted override 2026-10-31 (maintainer): org plan gates rulesets on this private repo (403); no plan upgrade, no public flip; payload + one-command re-apply kept in docs/setup/branch-protection.md | 2026-10-31T16:00:46.876Z | 2026-10-31T18:59:31.534Z |
| 3 | 05 | stub | internal/grpcapi/server.go |  | Watch/SnapshotCreate/SnapshotLoad intentionally Unimplemented (UnimplementedSupKVServer embed) until plans 05-04/05-05 | open |  | 2026-11-09T10:49:45.242Z |  |
| 4 | 06 | unrun-verify | .planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-09-PLAN.md |  | D-16 pending main-session work: re-run /gsd-execute-phase for phases 1, 4 and 5 to regenerate VERIFICATION.md fingerprints staled by the 2026-11-14 buf zero-waiver refactor (7b55ad1) — plan 06-09 task 3 is a checkpoint:human-action, not subagent-executable | fixed | Resolved 2026-11-16 in the main session: all five stale phases regenerated with fresh fingerprints (1: 72497d7, 4: e47733c, 5: 9b87620, 2: 5e62bc7 toolchain staleness, 3: 8c548d8 phase-6 covered-input staleness), all passed; STATE.md 945f80f; confirmed by the phase-6 verifier | 2026-11-16T01:34:11.794Z | 2026-11-16T12:55:00.000Z |
| 5 | 07 | deviation | .planning/phases/07-v1-0-docs-examples-and-release/07-01-SUMMARY.md |  | Option-b (maintainer 2026-11-17): DOCS-01/D-06 macos-15 leg is a named skip, not a real run — README quickstart ships a CI-verified Linux block plus a marked darwin variant block (probe loud-skips on darwin per the D-12 precedent); D-06/D-07 partially satisfied, must be adjudicated at phase verification (option-c rewording remains available there) | open |  | 2026-11-17T12:44:38.187Z |  |
| 6 | 07 | deviation | cmd/sup-kv/readme_test.go | 53 | Option-b contract (07-02): TestReadmeQuickstart darwin leg is a deterministic named skip — the macOS quickstart variant block is documented but not CI-verified on macos-15 (sendto: no route to host, PR #32 verdict); same D-06/D-07 partial-satisfaction adjudication as WINDOWS id 5 | open |  | 2026-11-17T13:01:07.869Z |  |
| 7 | 7 | unmet-truth | docs/OPERATIONS.md |  | DOCS-02 truth gap (07-03 flagged assumption): TestOperationsDoc is a presence/anchor guard only — no automated check establishes that the five mandated statements are TRUE (trust model, envelope, read semantics, not-a-lock, drift runbook); content truth is 07-VALIDATION.md's manual UAT table and must be verified there | fixed |  | 2026-11-17T13:31:19.251Z | 2026-11-18T16:03:28.566Z |
| 8 | 7 | deviation | cmd/sup-kv/examples_test.go |  | Examples concurrency/interruption residual (07-03 flagged assumption): marked blocks execute sequentially sharing one pair on the reserved 280xx doc band; a mid-walk test abort is backstopped by the PID-file t.Cleanup, but a hard runner cancel (SIGKILL to the go test process) between blocks skips cleanup and leaks the pair — surfaced for phase verification, not silently assumed safe | open |  | 2026-11-17T13:31:19.581Z |  |
| 9 | 07 | unrun-verify | .github/workflows/release.yml |  | 07-04 optional pre-tag gh workflow dispatch proof not run — workflow_dispatch requires release.yml on the default branch and gsd/v1.0-milestone is unmerged; 07-05 observes the real tag run post-merge | fixed |  | 2026-11-17T13:52:09.591Z | 2026-11-17T18:59:46.444Z |
| 10 | 01 | deviation | .planning/phases/01-spec-reference-and-decisions/01-01-PLAN.md |  | Task 1 negative-verify awk /internal/ patterns false-positive on evidence/repro/**/internal/** payload; executor scoped gates to the non-public prefix docs/reviews/2026-11-19-v1.0/internal/ (Rule 1 fix, publication asserts all clean) | open |  | 2026-11-19T14:46:56.007Z |  |
| 11 | 01 | deviation | .planning/phases/01-spec-reference-and-decisions/01-04-SUMMARY.md |  | Task 2 literal /internal/ tracked-path guard false-positives on 22 legitimate evidence/repro module paths; gate evaluated with the plan's own scoped prefix (docs/reviews/2026-11-19-v1.0/internal/) - wave-1 Rule 1 scoping re-affirmed at the phase gate (01-04-SUMMARY deviation 1) | open |  | 2026-11-19T15:49:20.556Z |  |
| 12 | 3 | deviation | .planning/phases/03-transport/03-12-SUMMARY.md |  | 03-12 halted: R-01 (7453-7734/8000 keys at the 20.6s bound) and R-02 burst (979.6-985.7 vs 990 writes/s) stayed red on the power-save host; rows reverted per plan, 03-13 blocked until a full-speed red-runner re-run | fixed | Resolved 2026-11-22: the miss was the powersave governor plus the burst window dividing by send-window-plus-drain; at performance the measurement was aligned with the documented acceptance (6c17db3) and the full runner is 7/7 OK with mutants killed (red-evidence/20261121T225048Z); 03-13's gate re-confirmed it (20261121T225950Z, 20261121T230732Z) | 2026-11-21T22:33:59.283Z | 2026-11-21T22:50:48.000Z |
| 13 | 07 | deviation | .planning/phases/07-observability-and-configuration/07-03-PLAN.md |  | Task 1/2 depguard probe command is masked by golangci-lint uniq-by-line: the literal blank-import probe makes revive:blank-imports report at the same 3:8 position, so the depguard finding never reaches the full-config output (Phase 6 WR-03 saw the same on api/). Rules are proven with a used-import probe (var _ = udprepl.SentByType{}) and with --default=none --enable=depguard on the literal probe; both name the rule. Fix future plans' probe shape. | open |  | 2026-11-23T07:51:08.600Z |  |

````json
[
  {
    "id": 1,
    "kind": "unmet-truth",
    "phase": "01",
    "file": ".planning/phases/01-ci-bootstrap-and-repo-skeleton/01-03-SUMMARY.md",
    "line": null,
    "description": "Backstop truth D5: Dependabot's first version-update PR not yet observed — weekly schedule window has not elapsed since dependabot.yml landed (2026-10-31)",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-10-31T15:45:51.348Z",
    "resolved_at": "2026-11-02T15:55:45.509Z",
    "milestone": "v1.0"
  },
  {
    "id": 2,
    "kind": "unmet-truth",
    "phase": "01",
    "file": "docs/setup/branch-protection.md",
    "line": 14,
    "description": "CI-03 protection sub-item deferred (org plan 403, decision defer 2026-10-31): direct pushes to master are not server-side rejected until the ruleset is applied; PENDING marker records the one-command re-apply path",
    "status": "waived",
    "reason": "Ratified accepted override 2026-10-31 (maintainer): org plan gates rulesets on this private repo (403); no plan upgrade, no public flip; payload + one-command re-apply kept in docs/setup/branch-protection.md",
    "recorded_at": "2026-10-31T16:00:46.876Z",
    "resolved_at": "2026-10-31T18:59:31.534Z",
    "milestone": "v1.0"
  },
  {
    "id": 3,
    "kind": "stub",
    "phase": "05",
    "file": "internal/grpcapi/server.go",
    "line": null,
    "description": "Watch/SnapshotCreate/SnapshotLoad intentionally Unimplemented (UnimplementedSupKVServer embed) until plans 05-04/05-05",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-09T10:49:45.242Z",
    "resolved_at": null,
    "milestone": "v1.0"
  },
  {
    "id": 4,
    "kind": "unrun-verify",
    "phase": "06",
    "file": ".planning/phases/06-hardening-failure-scenarios-observability-benchmarks/06-09-PLAN.md",
    "line": null,
    "description": "D-16 pending main-session work: re-run /gsd-execute-phase for phases 1, 4 and 5 to regenerate VERIFICATION.md fingerprints staled by the 2026-11-14 buf zero-waiver refactor (7b55ad1) — plan 06-09 task 3 is a checkpoint:human-action, not subagent-executable",
    "status": "fixed",
    "reason": "Resolved 2026-11-16 in the main session: all five stale phases regenerated with fresh fingerprints (1: 72497d7, 4: e47733c, 5: 9b87620, 2: 5e62bc7 toolchain staleness, 3: 8c548d8 phase-6 covered-input staleness), all passed; STATE.md 945f80f; confirmed by the phase-6 verifier",
    "recorded_at": "2026-11-16T01:34:11.794Z",
    "resolved_at": "2026-11-16T12:55:00.000Z",
    "milestone": "v1.0"
  },
  {
    "id": 5,
    "kind": "deviation",
    "phase": "07",
    "file": ".planning/phases/07-v1-0-docs-examples-and-release/07-01-SUMMARY.md",
    "line": null,
    "description": "Option-b (maintainer 2026-11-17): DOCS-01/D-06 macos-15 leg is a named skip, not a real run — README quickstart ships a CI-verified Linux block plus a marked darwin variant block (probe loud-skips on darwin per the D-12 precedent); D-06/D-07 partially satisfied, must be adjudicated at phase verification (option-c rewording remains available there)",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-17T12:44:38.187Z",
    "resolved_at": null,
    "milestone": "v1.0"
  },
  {
    "id": 6,
    "kind": "deviation",
    "phase": "07",
    "file": "cmd/sup-kv/readme_test.go",
    "line": 53,
    "description": "Option-b contract (07-02): TestReadmeQuickstart darwin leg is a deterministic named skip — the macOS quickstart variant block is documented but not CI-verified on macos-15 (sendto: no route to host, PR #32 verdict); same D-06/D-07 partial-satisfaction adjudication as WINDOWS id 5",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-17T13:01:07.869Z",
    "resolved_at": null,
    "milestone": "v1.0"
  },
  {
    "id": 7,
    "kind": "unmet-truth",
    "phase": "7",
    "file": "docs/OPERATIONS.md",
    "line": null,
    "description": "DOCS-02 truth gap (07-03 flagged assumption): TestOperationsDoc is a presence/anchor guard only — no automated check establishes that the five mandated statements are TRUE (trust model, envelope, read semantics, not-a-lock, drift runbook); content truth is 07-VALIDATION.md's manual UAT table and must be verified there",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-11-17T13:31:19.251Z",
    "resolved_at": "2026-11-18T16:03:28.566Z",
    "milestone": "v1.0"
  },
  {
    "id": 8,
    "kind": "deviation",
    "phase": "7",
    "file": "cmd/sup-kv/examples_test.go",
    "line": null,
    "description": "Examples concurrency/interruption residual (07-03 flagged assumption): marked blocks execute sequentially sharing one pair on the reserved 280xx doc band; a mid-walk test abort is backstopped by the PID-file t.Cleanup, but a hard runner cancel (SIGKILL to the go test process) between blocks skips cleanup and leaks the pair — surfaced for phase verification, not silently assumed safe",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-17T13:31:19.581Z",
    "resolved_at": null,
    "milestone": "v1.0"
  },
  {
    "id": 9,
    "kind": "unrun-verify",
    "phase": "07",
    "file": ".github/workflows/release.yml",
    "line": null,
    "description": "07-04 optional pre-tag gh workflow dispatch proof not run — workflow_dispatch requires release.yml on the default branch and gsd/v1.0-milestone is unmerged; 07-05 observes the real tag run post-merge",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-11-17T13:52:09.591Z",
    "resolved_at": "2026-11-17T18:59:46.444Z",
    "milestone": "v1.0"
  },
  {
    "id": 10,
    "kind": "deviation",
    "phase": "01",
    "file": ".planning/phases/01-spec-reference-and-decisions/01-01-PLAN.md",
    "line": null,
    "description": "Task 1 negative-verify awk /internal/ patterns false-positive on evidence/repro/**/internal/** payload; executor scoped gates to the non-public prefix docs/reviews/2026-11-19-v1.0/internal/ (Rule 1 fix, publication asserts all clean)",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-19T14:46:56.007Z",
    "resolved_at": null,
    "milestone": "v1.1"
  },
  {
    "id": 11,
    "kind": "deviation",
    "phase": "01",
    "file": ".planning/phases/01-spec-reference-and-decisions/01-04-SUMMARY.md",
    "line": null,
    "description": "Task 2 literal /internal/ tracked-path guard false-positives on 22 legitimate evidence/repro module paths; gate evaluated with the plan's own scoped prefix (docs/reviews/2026-11-19-v1.0/internal/) - wave-1 Rule 1 scoping re-affirmed at the phase gate (01-04-SUMMARY deviation 1)",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-19T15:49:20.556Z",
    "resolved_at": null,
    "milestone": "v1.1"
  },
  {
    "id": 12,
    "kind": "deviation",
    "phase": "3",
    "file": ".planning/phases/03-transport/03-12-SUMMARY.md",
    "line": null,
    "description": "03-12 halted: R-01 (7453-7734/8000 keys at the 20.6s bound) and R-02 burst (979.6-985.7 vs 990 writes/s) stayed red on the power-save host; rows reverted per plan, 03-13 blocked until a full-speed red-runner re-run",
    "status": "fixed",
    "reason": "Resolved 2026-11-22: the miss was the powersave governor plus the burst window dividing by send-window-plus-drain; at performance the measurement was aligned with the documented acceptance (6c17db3) and the full runner is 7/7 OK with mutants killed (red-evidence/20261121T225048Z); 03-13's gate re-confirmed it (20261121T225950Z, 20261121T230732Z)",
    "recorded_at": "2026-11-21T22:33:59.283Z",
    "resolved_at": "2026-11-21T22:50:48.000Z",
    "milestone": "v1.1"
  },
  {
    "id": 13,
    "kind": "deviation",
    "phase": "07",
    "file": ".planning/phases/07-observability-and-configuration/07-03-PLAN.md",
    "line": null,
    "description": "Task 1/2 depguard probe command is masked by golangci-lint uniq-by-line: the literal blank-import probe makes revive:blank-imports report at the same 3:8 position, so the depguard finding never reaches the full-config output (Phase 6 WR-03 saw the same on api/). Rules are proven with a used-import probe (var _ = udprepl.SentByType{}) and with --default=none --enable=depguard on the literal probe; both name the rule. Fix future plans' probe shape.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-23T07:51:08.600Z",
    "resolved_at": null,
    "milestone": "v1.1"
  }
]
````
