---
schema_version: 1
open_count: 2
waived_count: 0
fixed_count: 0
total_count: 2
last_updated: 2026-10-31T16:00:46.876Z
---

# Broken Windows Ledger

> Cross-phase defect register. With `workflow.windows_enforce` enabled, `/gsd-ship` blocks while `open_count > 0`.
> Waive with `gsd-tools windows waive <id> "<reason>"` (reason required).
> Mark fixed with `gsd-tools windows fixed <id>`.

| id | phase | kind | file | line | description | status | reason | recorded_at | resolved_at |
|----|-------|------|------|------|-------------|--------|--------|-------------|-------------|
| 1 | 01 | unmet-truth | .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-03-SUMMARY.md |  | Backstop truth D5: Dependabot's first version-update PR not yet observed — weekly schedule window has not elapsed since dependabot.yml landed (2026-10-31) | open |  | 2026-10-31T15:45:51.348Z |  |
| 2 | 01 | unmet-truth | docs/setup/branch-protection.md | 14 | CI-03 protection sub-item deferred (org plan 403, decision defer 2026-10-31): direct pushes to master are not server-side rejected until the ruleset is applied; PENDING marker records the one-command re-apply path | open |  | 2026-10-31T16:00:46.876Z |  |

````json
[
  {
    "id": 1,
    "kind": "unmet-truth",
    "phase": "01",
    "file": ".planning/phases/01-ci-bootstrap-and-repo-skeleton/01-03-SUMMARY.md",
    "line": null,
    "description": "Backstop truth D5: Dependabot's first version-update PR not yet observed — weekly schedule window has not elapsed since dependabot.yml landed (2026-10-31)",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-10-31T15:45:51.348Z",
    "resolved_at": null,
    "milestone": "v1.0"
  },
  {
    "id": 2,
    "kind": "unmet-truth",
    "phase": "01",
    "file": "docs/setup/branch-protection.md",
    "line": 14,
    "description": "CI-03 protection sub-item deferred (org plan 403, decision defer 2026-10-31): direct pushes to master are not server-side rejected until the ruleset is applied; PENDING marker records the one-command re-apply path",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-10-31T16:00:46.876Z",
    "resolved_at": null,
    "milestone": "v1.0"
  }
]
````
