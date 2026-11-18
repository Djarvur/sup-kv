---
phase: 04-replication-and-deterministic-simulator
reviewed: 2026-11-18T10:40:44Z
depth: deep
files_reviewed: 3
files_reviewed_list:
  - Makefile
  - go.mod
  - go.sum
findings:
  critical: 0
  warning: 2
  info: 0
  total: 2
  necessity: 0
status: issues_found
---

# Phase 4: Code Review Report (incremental — delta since d201a5b)

**Reviewed:** 2026-11-18T10:40:44Z
**Depth:** deep
**Files Reviewed:** 3 (Makefile, go.mod, go.sum)
**Status:** issues_found

## Summary

Incremental re-review of the three phase-04-covered files that changed since
the last full review (base `d201a5b`): the Makefile gains `bench` (06-08) and
`release`/`release-snapshot` (07-04), go.mod bumps the go directive
1.27.1 → 1.27.4 (GO-2026-6859) and adds `grpcurl` + `goreleaser` tool
directives, and go.sum carries the corresponding lock entries (go.mod 304 →
559 lines, 285 → 538 indirect requires; go.sum 1154 → 1582 lines — almost
entirely goreleaser's tree).

**Standing findings from the prior review: both are FIXED — not carried
forward.** The 2026-11-14 review carried WR-01 (drained-row duplicate node
id) and IN-01 (NotifyLocalWrite nil-return window), both in
`internal/udprepl/repl.go`. At HEAD, `retainDrained` now does
sort-then-`slices.Compact` (`repl.go:644-645`, comment cites WR-01) and
`NotifyLocalWrite` is a single select over `notify`+`done` with a liveness
re-check after a won send (`repl.go:358-383`) — the disposition ledger
records all five findings `fixed` with `open: 0`, and the named regression
tests all exist (`TestMirror_drainedAckedDedup`,
`TestNotifyLocalWrite_afterDone/afterRunReturns`,
`TestSync_responseRateClamp`, `TestEngine_redundantDirty`). New findings
below are numbered WR-03/WR-04 so the recorded WR-01/WR-02/IN-01/IN-02
disposition rows survive.

**Verification performed (all green unless noted):**

- `make test` (`go test -race -cover ./...`): full race suite passes,
  including the deterministic-simulator packages (`internal/app/internal/
  simnet`, `internal/app/repl`, `internal/udprepl`) — no regression to the
  simulator build path from the toolchain or tool-directive changes.
- `make bench`: all three TEST-05 benchmarks run
  (`BenchmarkDump10kKeys` / `BenchmarkReplicationBurst` /
  `BenchmarkConfirmationLatency`), `-run '^$$'` correctly excludes unit
  tests, and each named benchmark exists in exactly the listed package.
- `go build ./...`, `go tool golangci-lint config verify` (T-01-01 gate),
  `go mod verify` (all modules verified), `go mod tidy -diff` (exact —
  go.mod/go.sum are precisely tidy), `go tool govulncheck ./...` (no
  vulnerabilities — the 1.27.4 bump is effective on the reachable graph).
- go.sum (lock file, machine-generated): no per-line findings; correctness
  established by verify + tidy + presence of the `h1:`/`go.mod` pairs for
  the new tools (grpcurl v1.9.4, goreleaser v2.20.2).
- `go tool grpcurl` resolves and is load-bearing: the README/examples tests
  exec it through a pinned shim (`cmd/sup-kv/readme_test.go:10`).
- `go tool goreleaser` executes the pipeline correctly up to the SBOM
  stage — where the one empirical failure of this review occurs (WR-03).
  Running the gate for this review cleaned and partially repopulated the
  gitignored `dist/` with artifacts for the current commit; the previous
  snapshot's artifacts are regenerable.

Cross-file checks against the Makefile's claims: `release.yml` uses the
same `go tool goreleaser` pinned binary (no downloaded action binary for
goreleaser; syft is installed by the pinned `download-syft@v0.24.3` step),
`.goreleaser.yaml` matches the "4 targets + checksums + per-archive SBOMs +
CHANGELOG.md header" description, and `mustReadFile` exists in the pinned
goreleaser v2.20.2 template func map (`internal/tmpl/tmpl.go:321`). All
three workflows read the toolchain via `go-version-file: go.mod`.

## Warnings

### WR-03: `release-snapshot` — the "mandatory pre-tag gate" — fails on any machine without the external `syft` binary

**File:** `Makefile:82-87`
**Issue:** The target's help text bills it as "prove the whole artifact
pipeline locally … The mandatory pre-tag gate (D-09/Pitfall 8): run this
and see it green before any v* tag is pushed (07-05)". Empirically
reproduced during this review on this repository's own committing machine:

```
• software bill of materials
    • cataloging  cmd=syft sbom=sup-kv_0.0.0-SNAPSHOT-ab0c321_Darwin_arm64.tar.gz.sbom.json
⨯ release failed after 5s  error=exec: "syft": executable file not found in $PATH
make: *** [Makefile:87: release-snapshot] Error 1
```

Snapshot mode does NOT skip the SBOM stage (proven both by the failure and
by the `.sbom.json` artifacts in `dist/`), and goreleaser v2.20.2's sboms
pipe shells out to `syft` with no in-process fallback
(`internal/pipe/sbom/sbom.go:64` sets `cfg.Cmd = "syft"`, line 204
`exec.CommandContext`). The prerequisite is documented nowhere a Makefile
user would look: no `syft` mention in `docs/`, README, or AGENTS.md — only
a `.goreleaser.yaml:28-29` comment that attributes syft installation to the
CI workflow's action step, i.e. exactly not the local run this target
exists for. The help text is also materially incomplete: it says
"--snapshot builds all 4 targets plus checksums into dist/" — the pipeline
also builds SBOMs (the stage that actually fails), and on a syft-less
machine it never reaches checksums at all. Net effect: the gate cannot go
green as documented, and a developer who "sees it red" gets a puzzling
tool-missing error instead of a pipeline verdict. (The tag-triggered CI
release is unaffected — `release.yml:32` installs syft — so this is a
robustness defect of the local gate, not of the shipped release path.)
**Fix:** Make the prerequisite explicit and fail with a pointed message
(or install it pinned). Minimal:

```make
## release-snapshot: prove the whole artifact pipeline locally with NO tag
## and NO publishing — --snapshot builds all 4 targets, per-archive SBOMs
## (needs `syft` on PATH — CI pins it via anchore/sbom-action v0.24.3;
## locally install the same version) plus checksums into dist/.
release-snapshot:
	@command -v syft >/dev/null || { echo "release-snapshot: 'syft' not on PATH — the SBOM stage shells out to it (see .goreleaser.yaml)"; exit 1; }
	go tool goreleaser release --snapshot --clean
```

Better (keeps local/CI version parity inside the repo): point the sbom
pipe's `cmd` at a checked-in wrapper that `go run`s a pinned
`github.com/anchore/syft/cmd/syft@<version>` and drop the CI download step.

### WR-04: go.mod tool-directive additions sit permanently outside the vulnerability gate's reach

**File:** `go.mod:7` (grpcurl), `go.mod:9` (goreleaser) — cross-ref
`.github/workflows/vulncheck.yml:36`
**Issue:** This delta pins two new executable tools and, via goreleaser's
graph, grows the module set by 253 indirect requires (285 → 538) — the
largest supply-chain surface in the repo, including sigstore/cosign,
AWS/Azure/GCP SDKs and their crypto stacks. None of it is visible to the
project's vulnerability gate: `vulncheck.yml` runs
`go tool govulncheck ./...` (symbol/source mode), which analyzes only the
import graph of repository packages — empirically, `go list -deps ./...`
contains zero goreleaser/grpcurl packages. The workflow even triggers on
`go.mod`/`go.sum` PR paths, i.e. it is invoked precisely for Dependabot
tool-dep bumps it cannot evaluate: a CVE in a goreleaser dependency
bypasses the gate while the weekly scan and the dependency-PR scan both
stay green. The shipped runtime graph (grpc/protobuf) remains fully
covered — the gap is the tool code that executes during releases, proto
codegen and linting with repo contents and, for goreleaser, GitHub
credentials. Provable from the repo (workflow command + import-graph
isolation), hence Warning.
**Fix:** Add a tool-scan pass to the scheduled vulncheck workflow
(binary mode needs no symbols and works on built tool binaries):

```yaml
- name: Vulnerability scan pinned dev tools (invisible to ./... source scan)
  run: |
    GOBIN=/tmp/toolbin go install github.com/goreleaser/goreleaser/v2@v2.20.2
    GOBIN=/tmp/toolbin go install github.com/fullstorydev/grpcurl/cmd/grpcurl@v1.9.4
    go tool govulncheck -mode=binary /tmp/toolbin/goreleaser /tmp/toolbin/grpcurl
```

(or record the accepted trade-off in `.planning` with the rationale that
tools are hash-pinned and run only in trusted contexts).

---

_Reviewed: 2026-11-18T10:40:44Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
_Delta: base d201a5b (incremental re-review; files: Makefile, go.mod, go.sum)_
