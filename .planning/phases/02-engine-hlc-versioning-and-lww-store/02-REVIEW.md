---
phase: 02-engine-hlc-versioning-and-lww-store
reviewed: 2026-11-17T20:26:00Z
depth: deep
files_reviewed: 2
files_reviewed_list:
  - go.mod
  - go.sum
findings:
  critical: 0
  warning: 0
  info: 1
  total: 1
  necessity: 0
status: issues_found
---

# Phase 02: Code Review Report (incremental re-review — go.mod / go.sum only)

**Reviewed:** 2026-11-17T20:26:00Z
**Depth:** deep
**Files Reviewed:** 2 (`go.mod`, `go.sum`)
**Status:** issues_found (1 Info finding; 0 Critical, 0 Warning)

## Summary

Incremental re-review of phase 02. Since the phase's last review (2588852),
only `go.mod` and `go.sum` changed among the phase's covered files; the delta
is two commits:

- `c194c6f` — `go` directive bump 1.27.1 → 1.27.4, the GO-2026-6859
  (CVE-2026-97054) stdlib remediation;
- `9a536b7` — two new tool directives (`github.com/fullstorydev/grpcurl/cmd/grpcurl`,
  `github.com/goreleaser/goreleaser/v2`), phase-7 additions.

Both files were verified mechanically and against actual source usage — every
claim below was checked against the working tree, not taken from prior review
artifacts.

**Shipped graph: unchanged and exactly at the bar.**

- Direct requires are exactly the two permitted runtime deps:
  `google.golang.org/grpc v1.85.0`, `google.golang.org/protobuf
  v1.36.13-0.20261016172727-e9ee2a76ab8e` (`go.mod:15-18`) — byte-identical to
  the baseline at 2588852.
- `go list` of `./cmd/sup-kv`'s package dependencies yields only:
  `golang.org/x/net`, `golang.org/x/sys`, `golang.org/x/text`,
  `google.golang.org/genproto/googleapis/rpc`, `grpc`, `protobuf` — grpc +
  protobuf's own transitive closure. **Nothing new entered the shipped
  graph**; the ~500-module indirect require list is entirely the tool
  directives' graph (buf, golangci-lint, govulncheck, goreleaser, grpcurl,
  protoc plugins), which is the documented go-ultimate pinning pattern, not
  shipped code.
- The "zero-marginal optional" deps were never promoted and are correctly
  absent from the linked binary: no source file imports
  `github.com/google/uuid` or `golang.org/x/sync` (both remain `// indirect`
  via grpc's graph). RunID generation uses `crypto/rand` in-repo
  (`internal/app/repl/core.go:197`), the alternative CLAUDE.md explicitly
  permits; no errgroup fan-out exists to justify x/sync. No violation.

**Test-only bar item (`pgregory.net/rapid`): absent by ratified decision, not
by omission.** `git log -S 'pgregory.net/rapid' -- go.mod` is empty — rapid
was never in go.mod at any commit. The phase-02 discussion
(`02-DISCUSSION-LOG.md:47-52`) records the user's ratified choice of Option A
(stdlib generators, printed seed, manual minimization, zero new dependencies)
over Option B (rapid), and the property tests implement exactly that:
`math/rand/v2` seeded loops with "replay with the property seed above"
failure messages (`internal/app/storage/merge_test.go`,
`internal/app/clock/hlc_prop_test.go`). The `pgregory.net/rapid v1.2.0`
entries in `go.sum:1573-1574` are graph-required — `go mod graph` shows the
sole requirer is `github.com/moby/moby/api@v1.56.1` (a goreleaser
transitive) — kept by tidy, never compiled into anything. This is normal
go.sum behavior, not drift.

**Tool-directive consistency: verified.** All seven `tool` directives resolve
to versions in the require block: buf v1.75.0 (`go.mod:150`), grpcurl v1.9.4
(`go.mod:216`), golangci-lint v2.15.0 (`go.mod:274`), goreleaser v2.20.2
(`go.mod:297`), govulncheck v1.9.0 (`go.mod:538`), protoc-gen-go-grpc v1.6.2
(`go.mod:544`), protoc-gen-go via the direct protobuf require (`go.mod:17`,
  plugin and runtime one identical module). All match the CLAUDE.md pins where
CLAUDE.md lists them. Both new tools are genuinely invoked: goreleaser in
`Makefile:80,87`, `.github/workflows/release.yml:35,40`, `.goreleaser.yaml`;
grpcurl documented for use as `go tool grpcurl` in `docs/EXAMPLES.md:20-24`.

**`go 1.27.4` bump: valid remediation.** The `go` directive floors the
toolchain at the patched release (GOTOOLCHAIN switching for local devs;
CI's `setup-go` reads `go-version-file: go.mod`), which is the standard fix
for a stdlib CVE and satisfies go-ultimate's apps-pin-latest policy.

**Mechanical integrity:** `go mod tidy -diff` exits 0 (go.mod and go.sum are
exactly tidy — this is also the mechanical necessity proof that no require
entry is unneeded), `go mod verify` reports "all modules verified",
`go build ./...` succeeds. No `replace`/`exclude`/`retract` directives.

**Prior standing findings: all three resolved, none preserved.** Re-checked
against the current tree: IN-01's OnChange doc misstatement is fixed (the
comment now names `Records()` as the always-visible surface and states Get
returns the record "only when it is live", tombstone → `ErrNotFound`,
`internal/app/storage/store.go:30-36`); IN-02's re-entrancy contract is added
("must not call back into the same store … even a read-only Get
self-deadlocks", `internal/app/storage/store.go:41-46`); IN-03's duplicated
subtest now appears exactly once in `store_test.go`.

**Necessity scan:** none found. For a lockfile pair the tidy check is the
necessity check — every require is provably reachable from the main module's
imports or the tool graph, and go.sum's entry set is exactly what tidy keeps.

No Critical or Warning findings. The single Info finding is a
project-documentation drift surfaced by checking go.mod against the stated
bar; it has no runtime effect.

## Narrative Findings (AI reviewer)

## Info

### IN-01: CLAUDE.md stack tables drifted from the ratified dependency posture (rapid/uuid/x/sync promotion never happened)

**File:** `go.mod:15-18` (against `.claude/CLAUDE.md` "Technology Stack" tables)
**Issue:** CLAUDE.md still documents `pgregory.net/rapid v1.3.0` as "the one
justified test-only dependency … mandatory quality", `github.com/google/uuid`
and `golang.org/x/sync` as promoted zero-marginal direct deps, and the Go
directive as "1.27 (current patch 1.27.1)". The implemented and ratified
reality diverges on every point: phase 02's discussion chose stdlib-only
property testing (`02-DISCUSSION-LOG.md:47-52`), RunID uses `crypto/rand`
(`internal/app/repl/core.go:197`, a CLAUDE.md-permitted alternative), no
errgroup fan-out exists, and the directive is now `go 1.27.4`
(GO-2026-6859). go.mod is correct per the ratified decisions; the stale doc
is the cost: this review's own scope statement carried the stale bar
("test-only pgregory.net/rapid"), and future phases or reviewers reading
CLAUDE.md may re-add rapid believing it mandated, or flag its absence as a
violation — the mirror-image mistake of accepting the recommendation
unverified.
**Fix:** Regenerate the CLAUDE.md stack tables to match the ratified posture
(rapid: not adopted, Option A recorded in 02-DISCUSSION-LOG.md:47-52; uuid:
superseded by in-repo crypto/rand RunID; x/sync: unused; Go: 1.27.4 per
GO-2026-6859), or add a short ADR noting the Option-A decision so the stack
table's "recommended" status is traceable to a disposition.

---

_Reviewed: 2026-11-17T20:26:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
