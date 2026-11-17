# Phase 7: Code Review Disposition

**Source review:** [07-REVIEW.md](07-REVIEW.md) (reviewed 2026-11-17, depth: deep)
**Dispositioned:** 2026-11-17
**Scope applied:** Critical+Warning (WR-01) plus one trivial Info taken along (IN-01).
IN-02 through IN-05 remain **open for post-v1.0** by maintainer decision — none blocks
the v1.0.0 release; all four are recorded here so they survive into the post-release
backlog.

| Finding | Severity | Disposition | Note |
|---------|----------|-------------|------|
| WR-01 | Warning | fixed | README blocks invoke `./sup-kv` and the prose names the repository root; harness mirrors the post-build checkout (binary in the block cwd via `writeSupkvBinary`, no sup-kv PATH entry in `docRunEnv` for the README test) so a bare-name regression now fails CI as command-not-found. Commit `f3d5b86`. |
| IN-01 | Info | fixed | `.goreleaser.yaml` comment now states the actual behavior: the whole CHANGELOG.md becomes the release header, acceptable while releases are few, with the slice-the-current-version-section revisit named for the second release. Comment-only; `go tool goreleaser check` passes. Commit `4c69fa0`. |
| IN-02 | Info | open | `.gitignore` does not cover walkthrough scratch artifacts (`supkv-*.json`, `supkv-*.json.snapshots/`, `ex-*.pid`) — a user running the docs from a checkout gets a polluted `git status`. Post-v1.0. |
| IN-03 | Info | open | EXAMPLES stop-block poll window (10s) equals the binary's whole `shutdownGrace` bound (10s) — a legitimately held stream can outlast the documented wait. Post-v1.0. |
| IN-04 | Info | open | `extractMarkedBlocks` never resets the tracked `###` title on `#`/`##` headings, so a marker directly under a `##` section inherits the previous title. Latent (every current marker sits under its own `###`). Post-v1.0. |
| IN-05 | Info | open | Duplicate Quickstart heading in README (`## Quickstart` followed by `### Quickstart`) renders the same title twice in the outline. Post-v1.0. |

**Counts:** 6 findings — 2 fixed, 4 open (0 critical, 0 warnings outstanding).

**Verification of the fixes** (main checkout, branch `gsd/v1.0-milestone`):

- `go test -race ./cmd/sup-kv/ -run 'TestReadme|TestExamples' -v` — PASS (TestReadmeQuickstart 3.60s, TestExamplesDoc 6.72s)
- `go test -race -count=1 ./cmd/sup-kv/` — PASS (one unrelated load flake in `TestExec_emptyDiskRecovery`'s 10s health-gate wait on a first run; two consecutive green full runs after)
- `go tool golangci-lint run ./cmd/sup-kv/...` — clean
- `go tool goreleaser check` — 1 configuration validated
- Negative probe of the WR-01 property: under the new README harness environment (binary in cwd, no sup-kv PATH entry), a bare `sup-kv` exits 127 while `./sup-kv` resolves

---

_Dispositioned: 2026-11-17_
_Fixer: Claude (gsd-code-fixer)_
