---
phase: 01-ci-bootstrap-and-repo-skeleton
reviewed: 2026-11-14T22:03:58Z
depth: standard
files_reviewed: 6
files_reviewed_list:
  - .claude/CLAUDE.md
  - .github/workflows/ci.yml
  - Makefile
  - buf.yaml
  - buf.gen.repl.yaml
  - buf.gen.api.yaml
findings:
  critical: 0
  warning: 1
  info: 2
  total: 3
  necessity: 0
status: issues_found
---

# Phase 01: Code Review Report (incremental — delta 7b55ad1)

**Reviewed:** 2026-11-14T22:03:58Z
**Depth:** standard
**Files Reviewed:** 6
**Status:** issues_found

## Summary

Incremental review of the phase-1 owned files as changed by commit 7b55ad1
("refactor(proto): satisfy buf STANDARD with zero waivers", diff base
12e300f): the buf workspace restructured to one module per proto tree, the
single `buf.gen.yaml` split into `buf.gen.repl.yaml` / `buf.gen.api.yaml`,
and the D-08 breaking-guard path updated in Makefile and ci.yml.

**Phase 01 contract verification (all hold):**

- `ci.yml` — the delta is exactly ONE line (the guard path
  `internal/udprepl/replproto/supkv.proto` → `.../supkv/v1/supkv.proto`);
  check-run names ("lint", "gate (<cell>)") and the 4-cell matrix are
  byte-unchanged, so the required-check contexts in the branch-protection
  ruleset are untouched. `permissions: contents: read` and the concurrency
  group intact. No new steps, no `continue-on-error`, no credentials.
- Guard correctness re-probed live against this tree: `origin/master` sits
  at 617951e (phase 3) — NEITHER the old nor the new proto path exists
  there (`git cat-file -e` exits 128 for both), so the deferred branch is
  the correct behavior today, and the guard now watches the path that will
  actually land, self-activating on the first PR after the move merges.
  Had the guard kept the old path it would defer forever — the update is
  the load-bearing half of this edit and it is right.
- `Makefile` — all eight targets intact (lint/test/generate/build/proto/
  buf-lint/buf-format/buf-breaking), tab-indented, all `.PHONY`. The new
  `proto` target's two per-tree passes were executed live: `make proto`
  regenerates both trees byte-identically (working tree shows no source
  diff afterwards) — the `out:` pins inside each template write back
  inside the tree as documented.
- Pinned-tool policy (D-05, `.claude/CLAUDE.md`): `go tool buf` plus
  `local: ["go", "tool", "protoc-gen-go"]` / `...protoc-gen-go-grpc`
  plugins in both new templates — everything resolves through go.mod tool
  directives, no global PATH installs, no BSR remote plugins. Consistent.
- `buf.yaml` — zero `ignore_only` waivers confirmed structurally, and the
  dropped `.gomodcache` exclude is sound: the repo-root `.gomodcache`
  (which really does contain stray `.proto` files — verified) now sits
  outside both module roots (`internal/udprepl/replproto`,
  `internal/grpcapi/apiproto`), and `go tool buf lint` passes with it
  present. Gates re-run live during this review: `buf lint` exit 0, `buf
  format --diff --exit-code` exit 0.
- `.claude/CLAUDE.md` — the Buf CLI tool-table row accurately describes
  the per-tree templates and the one-module-per-tree workspace.

Two new Info findings (stale doc references to the deleted `buf.gen.yaml`
and a now-false "one pass, both trees" comment in `buf.yaml`), plus the
carried-open WR-01 (CI job timeouts) which this delta did not address and
which is re-reported unchanged below. No structural-findings pre-pass and
no external reviewer evidence were provided for this run; findings are all
narrative (direct review). No Critical issues.

## Narrative Findings (AI reviewer)

All findings below originate from the direct adversarial review above.
WR-01 is the still-open carry from the previous review of this file (same
id, same finding — the disposition row is kept); IN-07/IN-08 are new in
this delta.

## Warnings

### WR-01: Neither CI job sets a timeout-minutes bound — a wedged job holds its runner for GitHub's 6-hour default

**File:** `.github/workflows/ci.yml:19-20` (job `lint`), `.github/workflows/ci.yml:53-55` (job `gate`)
**Issue:** (Carry-forward, unchanged by 7b55ad1 — re-verified against the
current file.) Both jobs declare `runs-on:` but neither sets a job-level
`timeout-minutes:`. A hung `golangci-lint` download, a wedged `-race` test
suite, or a stuck `go tool buf` invocation holds the runner until GitHub's
6-hour default. `fail-fast: false` keeps all four gate cells alive, so one
wedged cell means up to five held runners per pushed commit — paid runner
minutes under the branch-protection ruleset.
**Fix:** Bound each job:

```yaml
jobs:
  lint:
    runs-on: ubuntu-latest
    timeout-minutes: 15
  gate:
    name: gate (${{ matrix.name }})
    runs-on: ${{ matrix.runs-on || matrix.name }}
    timeout-minutes: 20
```

## Info

### IN-07: Three stale `buf.gen.yaml` references remain in .claude/CLAUDE.md after the file was deleted

**File:** `.claude/CLAUDE.md:117`, `.claude/CLAUDE.md:138`, `.claude/CLAUDE.md:147`
**Issue:** 7b55ad1 updated the Buf CLI tool-table row to name the new
per-tree templates, but three other mentions of the generate config still
name `buf.gen.yaml`, a file the same commit deleted: the rejected-row "BSR
remote plugins in `buf.gen.yaml`" (line 117), the version-compatibility
row "buf.yaml/buf.gen.yaml `version: v2` configs" (line 138), and the
sources entry "buf.gen.yaml v2 local-plugin syntax" (line 147). A reader
following the doc to the config file finds nothing on disk.
**Fix:** Reword to the plural/template names, e.g. "BSR remote plugins in
the buf.gen.*.yaml templates", "buf.yaml / buf.gen.repl.yaml /
buf.gen.api.yaml `version: v2` configs", "buf.gen v2 local-plugin
syntax".

### IN-08: buf.yaml comment claims a single workspace-wide generate pass that no longer exists

**File:** `buf.yaml:14-15`
**Issue:** The module comment ends "buf generate covers every module in
the workspace — one pass, both trees." Since 7b55ad1 deleted the default
`buf.gen.yaml`, a bare `go tool buf generate` now fails outright
(probe-verified: "Failure: read buf.gen.yaml: file does not exist"), and
the configured flow is the Makefile's TWO per-tree passes with explicit
`--template` flags. The sentence describes the pre-delta world and
contradicts both the Makefile comment and the actual targets one screen
over in the same repo. The failure mode is self-revealing (a loud buf
error, not silent wrong output), hence Info.
**Fix:** Replace the trailing sentence with the current reality, e.g.
"generate is driven per tree by the Makefile proto target (buf.gen.repl.yaml /
buf.gen.api.yaml) — a bare workspace generate has no default template
anymore."

---

_Reviewed: 2026-11-14T22:03:58Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
_Delta: 7b55ad1 (base 12e300f)_
