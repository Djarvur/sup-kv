---
status: testing
phase: 07-v1.0 — Docs, Examples and Release
source: [07-VERIFICATION.md]
started: 2026-11-17T19:30:00Z
updated: 2026-11-17T19:30:00Z
---

## Current Test

number: 1
name: DOCS-01 macOS-leg adjudication — accept option-b (complete-with-recorded-partial) or option-c (reword the requirement)
expected: |
  The maintainer decides how the macOS quickstart claim stands: option-b keeps DOCS-01 as complete
  with the recorded partial (macos-15 leg is a named loud-skip, WINDOWS 5/6 carry the open entries),
  or option-c rewords DOCS-01/ROADMAP SC1 so the requirement claims only what is proven.
  Linux leg is machine-proven; the macOS broadcast probe died at the limited-broadcast row before
  the derived form was ever probed — option-c is a wording choice, not evidence-forced.
awaiting: user response

## Tests

### 1. DOCS-01 macOS-leg adjudication (decision)
expected: accept option-b (complete-with-recorded-partial) or option-c (reword DOCS-01/SC1 and close)
result: [pending]

### 2. OPERATIONS.md content-truth UAT (WINDOWS 7)
expected: the five mandated statements are TRUE of the binary — trust model (no auth/encryption in v1, HMAC future), value envelope, local-read semantics, not-a-lock warning, clock-drift runbook. Presence is machine-proven (41 anchors); constants (61440, 60s) already cross-checked against main.go. Human read-through sign-off.
result: [pending]

### 3. Stranger cold-read test (the phase goal itself)
expected: a stranger can deploy and operate a two-node sup-kv cluster from the documentation alone — follow README quickstart + docs/OPERATIONS.md + docs/EXAMPLES.md cold, without repo knowledge, and reach a working two-node cluster.
result: [pending]

### 4. Shipped-artifact README staleness (decision)
expected: |
  The v1.0.0 Release archives embed the pre-WR-01 quickstart (bare `sup-kv` invocations after
  `go build -o sup-kv` — a stranger extracting the archive hits command-not-found). The
  copy-paste-runnable `./sup-kv` fix is committed on gsd/v1.0-milestone (f3d5b86, green) but
  postdates the tag; origin/master is still at 0ead1e8. Repo README self-heals on branch merge;
  the frozen archive copy does not. Maintainer consciously accepts fixed-forward (v1.0.1 later)
  or delete-and-re-tag v1.0.0.
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps
