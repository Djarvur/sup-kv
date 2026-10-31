---
schema_version: 1
open_count: 1
waived_count: 0
fixed_count: 0
total_count: 1
last_updated: 2026-10-31T15:45:51.348Z
---

# Broken Windows Ledger

> Cross-phase defect register. With `workflow.windows_enforce` enabled, `/gsd-ship` blocks while `open_count > 0`.
> Waive with `gsd-tools windows waive <id> "<reason>"` (reason required).
> Mark fixed with `gsd-tools windows fixed <id>`.

| id | phase | kind | file | line | description | status | reason | recorded_at | resolved_at |
|----|-------|------|------|------|-------------|--------|--------|-------------|-------------|
| 1 | 01 | unmet-truth | .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-03-SUMMARY.md |  | Backstop truth D5: Dependabot's first version-update PR not yet observed — weekly schedule window has not elapsed since dependabot.yml landed (2026-10-31) | open |  | 2026-10-31T15:45:51.348Z |  |

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
  }
]
````
