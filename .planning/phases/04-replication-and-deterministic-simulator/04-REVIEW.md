---
phase: 04-replication-and-deterministic-simulator
reviewed: 2026-11-14T22:03:58Z
depth: standard
files_reviewed: 3
files_reviewed_list:
  - internal/udprepl/repl.go
  - internal/udprepl/replproto/supkv/v1/supkv.proto
  - internal/udprepl/replproto/supkv/v1/supkv.pb.go
findings:
  critical: 0
  warning: 1
  info: 1
  total: 2
  necessity: 0
status: issues_found
---

# Phase 4: Code Review Report (incremental — delta 7b55ad1)

**Reviewed:** 2026-11-14T22:03:58Z
**Depth:** standard
**Files Reviewed:** 3
**Status:** issues_found

## Summary

Incremental review of commit 7b55ad1's phase-4 slice (diff base 22dc340):
the replication schema's move into its package-derived tree
(`internal/udprepl/replproto/supkv/v1/`) and the adapter's import update.
The delta-specific checks all pass:

- **repl.go is import-path-only** — `diff` of the whole file against
  22dc340 shows exactly ONE changed line (the import
  `internal/udprepl/replproto` → `internal/udprepl/replproto/supkv/v1`,
  with an explicit `replproto` alias). No handler, mirror, codec, timer or
  lifecycle logic touched; no behavior change in the replication adapter.
- **The moved proto is byte-identical apart from go_package** — `diff`
  against 22dc340's `internal/udprepl/replproto/supkv.proto` shows exactly
  one changed line: the go_package import path gains `/supkv/v1`. Crucially
  the `;replproto` Go package alias is preserved, so every
  `replproto.Xxx` symbol reference in the adapter keeps compiling
  unchanged — zero churn beyond the import line. Field names and numbers
  untouched, so the UDP wire format is unchanged (the CRC/oneof schema is
  identical bytes).
- **The regenerated pb.go is a clean, complete mechanical rename** —
  DO-NOT-EDIT header intact, same generator stamp (protoc-gen-go
  v1.36.12-devel), `package replproto` preserved, `source:` now the
  module-relative `supkv/v1/supkv.proto`, and the raw descriptor embeds
  the moved path (`"\x14supkv/v1/supkv.proto\x12\bsupkv.v1"`). All 43
  internal symbol renames (`file_internal_udprepl_replproto_supkv_proto*`
  → `file_supkv_v1_supkv_proto*`, incl. the `File_...` descriptor var and
  the init/registration block) were checked at head and tail of the diff —
  consistent, no hand edits. `make proto` re-run during this review
  reproduces the file byte-identically.
- Live gates: `go build ./...` + `go vet ./...` clean; `go test -race` on
  `internal/udprepl` (and the other touched packages) green; `buf lint` 0
  with zero waivers for this tree.

No new findings in the delta itself. Two still-open findings from the
previous review live in the in-scope file `repl.go` and are re-reported
unchanged below (same ids, same findings — disposition rows preserved).
WR-02 (SyncResponseRate, `internal/app/repl/core.go:122-124`) and IN-02
(restore dirty marking, engine) remain open in the ledger but their files
are outside this delta's scope. No structural-findings pre-pass and no
external reviewer evidence were provided.

## Narrative Findings (AI reviewer)

All findings below are carry-forwards from the 2026-11-14 review,
re-verified present in the current `repl.go` at the cited lines; the
7b55ad1 delta did not touch the implicated logic (import-line-only
change).

## Warnings

### WR-01: Drained-row retention can publish a duplicate node id in Acked

**File:** `internal/udprepl/repl.go:513-523`
**Issue:** (Carry-forward, unchanged by 7b55ad1 — the `retainDrained`
union is still `append(slices.Clone(prev.Acked), prev.WaitingOn...)` +
`slices.Sort` with no dedup, re-verified at line 513.) Because
`handleHello`'s discovery flush re-adds a peer to `entry.waitSet` without
touching `entry.acked`, a peer can sit in BOTH sets when the drain fires,
and the retained row publishes `Acked: [B, B]`, violating the documented
"union is exactly the core's acked set at deletion" invariant — a
`wait_for_replication` consumer counting `len(Acked)` over-reports by the
duplicate. Full four-step production sequence in the original finding
(previous 04-REVIEW.md, WR-01).
**Fix:** Dedup the union — both inputs are sorted, so `slices.Compact`
after the sort:

```go
acked := append(slices.Clone(prev.Acked), prev.WaitingOn...)
slices.Sort(acked)
acked = slices.Compact(acked)
```

## Info

### IN-01: NotifyLocalWrite pre-check leaves a narrow nil-return window during Run's exit

**File:** `internal/udprepl/repl.go:314-339`
**Issue:** (Carry-forward, unchanged by 7b55ad1 — the closed-loop
`select`/`default` pre-check at lines 328-332 followed by the racing
buffered send is exactly the code the original finding described.)
Between the pre-check's `default:` (loop looks alive) and the second
select's `a.notify <- lw` send, Run can return and close `a.done`; the
buffered send then wins and NotifyLocalWrite reports nil for a
notification nobody will drain. Bounded, informational window — see the
original finding for the accepted-risk reasoning.
**Fix:** As originally suggested — a single select over both channels, or
an atomic/once-guarded loop-liveness flag checked after a successful send
(to drain-and-report ErrRunLoopExited), if the window is ever judged worth
closing.

---

_Reviewed: 2026-11-14T22:03:58Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
_Delta: 7b55ad1 (base 22dc340)_
