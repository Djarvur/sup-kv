---
status: complete
phase: 07-v1.0 — Docs, Examples and Release
source: [07-VERIFICATION.md]
started: 2026-11-17T19:30:00Z
updated: 2026-11-17T19:45:00Z
---

## Current Test

[testing complete]

## Tests

### 1. DOCS-01 macOS-leg adjudication (decision)
expected: accept option-b (complete-with-recorded-partial) or option-c (reword DOCS-01/SC1 and close)
result: passed — maintainer accepted option-b (2026-11-17): DOCS-01 complete with the recorded partial; macos-15 leg stays a named loud-skip; WINDOWS 5/6 remain tracked open entries

### 2. OPERATIONS.md content-truth UAT (WINDOWS 7)
expected: the five mandated statements are TRUE of the binary — trust model (no auth/encryption in v1, HMAC future), value envelope, local-read semantics, not-a-lock warning, clock-drift runbook. Presence is machine-proven (41 anchors); constants (61440, 60s) already cross-checked against main.go. Human read-through sign-off.
result: pass — maintainer read-through 2026-11-18: all five statements truthful and clear

### 3. Stranger cold-read test (the phase goal itself)
expected: a stranger can deploy and operate a two-node sup-kv cluster from the documentation alone — follow README quickstart + docs/OPERATIONS.md + docs/EXAMPLES.md cold, without repo knowledge, and reach a working two-node cluster.
result: pass — maintainer confirmed 2026-11-18, then agent-assisted cold run executed 2026-11-18: fresh clone of gsd/v1.0-milestone to /tmp/supkv-cold, README quickstart block executed VERBATIM (grpcurl on PATH per the EXAMPLES-declared prerequisite, built from the repo's pinned tool directive) — build OK, Set on A visible on B, kill -9 + restart + Get on A recovers the value, human-readable dumps, block exit 0, no leftover processes

### 4. Shipped-artifact README staleness (decision)
expected: |
  The v1.0.0 Release archives embed the pre-WR-01 quickstart (bare `sup-kv` invocations after
  `go build -o sup-kv` — a stranger extracting the archive hits command-not-found). The
  copy-paste-runnable `./sup-kv` fix is committed on gsd/v1.0-milestone (f3d5b86, green) but
  postdates the tag; origin/master is still at 0ead1e8. Repo README self-heals on branch merge;
  the frozen archive copy does not. Maintainer consciously accepts fixed-forward (v1.0.1 later)
  or delete-and-re-tag v1.0.0.
result: passed — maintainer accepted fixed-forward (2026-11-17): v1.0.0 stands as shipped; repo README self-heals on branch merge; the archive copy's quickstart defect is consciously accepted and will close with v1.0.1

## Summary

total: 4
passed: 4
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps
