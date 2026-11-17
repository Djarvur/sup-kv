---
schema_version: 1
open_count: 2
waived_count: 1
fixed_count: 2
total_count: 5
last_updated: 2026-11-17T12:44:38.187Z
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
  }
]
````
