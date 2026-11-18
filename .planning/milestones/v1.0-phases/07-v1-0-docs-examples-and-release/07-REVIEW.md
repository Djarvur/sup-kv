---
phase: 07-v1-0-docs-examples-and-release
reviewed: 2026-11-17T12:00:00Z
depth: deep
files_reviewed: 17
files_reviewed_list:
  - .github/workflows/release.yml
  - .gitignore
  - .goreleaser.yaml
  - CHANGELOG.md
  - Makefile
  - README.md
  - cmd/sup-kv/broadcast_probe_test.go
  - cmd/sup-kv/examples_test.go
  - cmd/sup-kv/main.go
  - cmd/sup-kv/operations_test.go
  - cmd/sup-kv/readme_test.go
  - cmd/sup-kv/scenario_test.go
  - docs/EXAMPLES.md
  - docs/OPERATIONS.md
  - docs/setup/branch-protection.md
  - go.mod
  - go.sum
findings:
  critical: 0
  warning: 1
  info: 5
  total: 6
  necessity: 0
status: issues_found
---

# Phase 7: Code Review Report

**Reviewed:** 2026-11-17T12:00:00Z
**Depth:** deep
**Files Reviewed:** 17
**Status:** issues_found

## Summary

Docs-and-release phase reviewed at deep depth with cross-file tracing: the doc-test harness (`extractMarkedBlocks`/`runBashVerbatim` in `cmd/sup-kv/examples_test.go`, reused by `readme_test.go`), the release workflow ↔ goreleaser ↔ Makefile wiring, and every truthfulness anchor the docs assert about the binary. The core verdict is positive — this is unusually well-verified work — but the verification harness itself masks one real defect in the phase's headline deliverable (WR-01): the README quickstart is machine-verified under a PATH the harness injects, so its central "copy-paste this block" claim is false for the stranger it is written for.

Cross-file verification performed (all confirmed, none flagged):

- `go tool goreleaser check` validates `.goreleaser.yaml`; `mustReadFile` exists in the pinned goreleaser v2.20.2 (`internal/tmpl/tmpl.go:321`); snapshot mode with no tag resolves `CurrentTag` to `"v0.0.0"` (`internal/pipe/git/git.go:143`), so the `{{ .Tag }}` ldflags stamp stays well-formed on the `workflow_dispatch` snapshot path — no empty-version-token hazard.
- The five required-check contexts in `docs/setup/ruleset.json` match `.github/workflows/ci.yml`'s check-run names (`lint`, `gate (<cell>)` × 4) byte-for-byte, as `docs/setup/branch-protection.md` claims; `release.yml` is tag/dispatch-only and makes zero ci.yml edits.
- All 40 anchor literals in `operations_test.go` are present in `docs/OPERATIONS.md`, and the doc's factual claims verify against the code: reads never consult the startup gate (`internal/grpcapi/gate.go:102-107`), all 11 `supkv_*` metric families and the five discard labels exist (`internal/metrics/metrics.go`, `internal/app/repl/core.go:40-44`), the snapshot dir suffix `.snapshots` matches `internal/persistfile/snapshot.go:14`, defaults (5s ack max, 30s sync, 60s drift, loopback gRPC) match `cmd/sup-kv/main.go`.
- `cmd/sup-kv/main.go`'s unvalidated int knobs (`--sync-response-rate`, `--value-max-bytes`) are normalized downstream (`internal/app/repl/core.go:158`, `internal/app/storage/store.go:107`) — no divide-by-zero or negative-envelope path.
- The doc-test port bands are disjoint (README 279xx, EXAMPLES 280xx fixed; scenario window `[28100, 32678)`; `freeTCPPort`/`freeUDPPort` use the ephemeral range), so parallel package tests cannot collide.
- `corruptDumpRecord`'s field arithmetic is sound: checksum is the last JSON field in the dump record (`internal/app/persist/codec.go:53`).
- `reflection.Register` registers both v1 and v1alpha reflection services (grpc v1.85.0 `reflection/serverreflection.go:64-65`), matching EXAMPLES's documented `grpcurl list` output.
- Harness symbols referenced by `scenario_test.go`/`broadcast_probe_test.go` all resolve in `main_test.go`; `go vet ./cmd/...` is clean; benchmarks referenced by `make bench` exist.
- CHANGELOG feature claims spot-checked against the API surface (create-if-absent: `internal/grpcapi/server.go:142`, seven RPCs, wait fields, snapshots) — truthful.

Necessity (#4100) was assessed per the rubric: the `option-b` skip-latch constants, the `cmd.Process != nil` guard (reachable on Start failure), and `signalBuffer` all defend real, reachable states — zero necessity findings.

CI/CD pipeline checks per the reference rubric: `release.yml` has a job timeout (20m), pinned action refs, no non-idempotent installs, no static credentials, build-once-per-tag. No findings. `go.sum` is a generated lockfile — reviewed for consistency only (CI's tidy-diff gate covers it); no findings possible in it.

## Warnings

### WR-01: README quickstart is not copy-paste runnable — the build command and the verified blocks disagree about how `sup-kv` resolves

**Status:** fixed (2026-11-17, commit `f3d5b86`) — every block invocation is now `./sup-kv` (matching the build step's output location), and the harness runs the README blocks in a directory carrying the built binary at `./sup-kv` with no sup-kv PATH entry, so a bare-name regression fails the test as command-not-found instead of being masked. See 07-REVIEW-DISPOSITION.md.

**File:** `README.md:42-53` (also `README.md:84-85`)
**Issue:** The README builds the binary with `go build -o sup-kv ./cmd/sup-kv` (line 42), placing it at `./sup-kv` in the checkout — then every machine-verified block invokes it as a bare `sup-kv` (lines 53, 55, 73), and line 84 explicitly invites: "Copy-paste the block into a bash shell." On any normal system `.` is not on `PATH`, so a stranger following the README verbatim gets `sup-kv: command not found` on the first line of the quickstart. The defect is invisible to CI because `TestReadmeQuickstart` runs the block under a synthetic `PATH` (`supkvBinDir` in `cmd/sup-kv/examples_test.go:169-182` puts a copy named `sup-kv` first on `PATH`) — the verification harness masks exactly the gap it exists to close. Note `docs/EXAMPLES.md` avoids the trap by declaring "Two tools on your `PATH`" as a prerequisite (EXAMPLES.md:16), though its own `make build` pointer (output `bin/sup-kv`, not on `PATH`) leaves the same friction unaddressed.
**Fix:** Make the build step and the invocation agree. Cheapest: change line 42 to `go install ./cmd/sup-kv` (installs into `GOBIN`/`~/go/bin`, which a Go developer already has on `PATH`), or add one setup line before the block (`mv sup-kv "$(go env GOBIN 2>/dev/null || echo ~/go/bin)/"` or `PATH=$PWD:$PATH`) — or switch the blocks to `./sup-kv` and teach the harness to also copy the binary into the shared block directory so the verbatim run still resolves it.

## Info

### IN-01: goreleaser release header pulls the whole CHANGELOG.md, not "the curated KaC entry" the comment promises

**Status:** fixed (2026-11-17, commit `4c69fa0`) — comment-only fix per the review's simplest-robust-form recommendation: it now documents that the whole CHANGELOG.md becomes the release header and names the slice-the-section revisit trigger for the second release. See 07-REVIEW-DISPOSITION.md.

**File:** `.goreleaser.yaml:40-43`
**Issue:** `header: '{{ mustReadFile "CHANGELOG.md" }}'` reads the entire file. For v1.0.0 (single entry) that is indistinguishable from the curated entry, but from the second release on the GitHub Release header will carry every prior version plus the empty `## [Unreleased]` skeleton and the link footer — the in-file comment ("pulls the curated KaC entry as the Release header") overstates what the mechanism does, and nothing will flag the drift.
**Fix:** Either reword the comment to "the full CHANGELOG.md becomes the release header (acceptable while releases are few)", or slice the current-version section, e.g. `header: '{{ split (mustReadFile "CHANGELOG.md") "\n## [" }}...'` — simplest robust form is documenting the current behavior.

### IN-02: .gitignore does not cover the documented walkthroughs' scratch artifacts

**File:** `.gitignore:1-13`
**Issue:** The quickstart and EXAMPLES walkthroughs are designed to be run from a checkout (README builds there; dump paths are relative). They leave `supkv-a.json`, `supkv-b.json`, `supkv-ex-a.json`, `supkv-ex-b.json`, `ex-a.pid`, `ex-b.pid` and `supkv-ex-a.json.snapshots/` in the working directory — none ignored (`*.out` at line 12 coincidentally covers only the EXAMPLES `.out` capture files). A user who follows the docs in the repo root gets a polluted `git status` and risks committing demo state.
**Fix:** Add the walkthrough scratch names (or `supkv-*.json`, `supkv-*.json.snapshots/`, `ex-*.pid`) to `.gitignore`.

### IN-03: EXAMPLES stop-block poll window equals the binary's whole graceful-shutdown bound

**File:** `docs/EXAMPLES.md:346-353` (vs `cmd/sup-kv/main.go:78`)
**Issue:** The "Stop the pair" block polls for both exits for 50 × 0.2s = 10s before its unconditional `ps` proof, while the binary's own `shutdownGrace` is also 10s (`main.go:78`) — a legitimate worst-case drain (e.g., an operator still holding a Watch stream from the earlier section when the stop runs, which keeps `GracefulStop` busy for the full grace) makes a correctly-behaving node exceed the block's poll and fail it. Unreachable in the walkthrough itself (the Watch client is killed in its own block), but the documented bound is tighter than the contract it claims to demonstrate.
**Fix:** Extend the bounded wait to comfortably exceed one node's worst-case shutdown (e.g., `seq 1 100` → 20s), or note in the block comment that a held stream legitimately lengthens the drain.

### IN-04: extractMarkedBlocks lets a ### title leak across ## section boundaries

**File:** `cmd/sup-kv/examples_test.go:71-77`
**Issue:** Title tracking only updates on `### ` lines and never resets on `#`/`##` headings, so a marker placed directly under a new `## ` section (no intervening `###`) inherits the previous section's title. The block then runs and asserts against the wrong (or an unexpectedly-matching) assertion-table key instead of failing with the intended "marked block has no ### heading above it" error. Latent only — every marker in the current README and EXAMPLES sits directly under its own `### ` heading — but the harness is reused by three tests and will outlive these two docs.
**Fix:** Reset `title` on any ATX heading: `if strings.HasPrefix(line, "#") { title = "" }; if strings.HasPrefix(line, "### ") { title = ... }`.

### IN-05: Duplicate Quickstart heading in README

**File:** `README.md:37,49`
**Issue:** `## Quickstart` (line 37) is immediately followed by `### Quickstart` (line 49), rendering the same title twice in the outline. The inner heading exists to key `extractMarkedBlocks`' block naming, but its text need not repeat the parent's.
**Fix:** Rename the inner heading (e.g., `### Two nodes on one host`) — any `### ` text works as the harness key for the README test (titles are only used in failure messages there; the assertion table keys only apply to EXAMPLES).

---

_Reviewed: 2026-11-17T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
