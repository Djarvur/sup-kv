---
phase: "7"
slug: "v1-0-docs-examples-and-release"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-11-17"
---

# Phase 7 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | stdlib `testing` (repo mandate; no assertion library) |
| **Config file** | none needed — tests join `cmd/sup-kv` package `main_test` with the existing `TestMain` build-once harness |
| **Quick run command** | `go test -race ./cmd/sup-kv/ -run 'TestReadme|TestExamples'` |
| **Full suite command** | `make test` (`go test -race -cover ./...`) — the doc tests run inside the existing gate cells, no CI job changes |
| **Estimated runtime** | ~30–90 seconds (doc tests ride the existing binary-build-once harness) |

---

## Sampling Rate

- **After every task commit:** Run `go test -race ./cmd/sup-kv/ -run 'TestReadme|TestExamples'`
- **After every plan wave:** Run `make lint && make test` (full gate parity)
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

Seeded from `07-RESEARCH.md` §Validation Architecture. Task IDs finalize at planning time.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 07-01 T1 | 07-01 | 1 | DOCS-03 (D-04/D-05 slice) | T-7-01 | reflection disclosure accepted + documented | integration (real binary + pinned grpcurl) | `go test -race ./cmd/sup-kv/ -run TestExamplesDoc` | ❌ W0 (`cmd/sup-kv/examples_test.go`) | ⬜ pending |
| 07-01 T2 | 07-01 | 1 | DOCS-01 (macOS leg prerequisite) | — | N/A | capability probe (verdict via gate-cell pass/fail) | `go test -race ./cmd/sup-kv/ -run TestProbeQuickstartBroadcast` (macos-15 cell outcome = the verdict) | ❌ W0 (`cmd/sup-kv/broadcast_probe_test.go`) | ⬜ pending |
| 07-01 T3 | 07-01 | 1 | DOCS-01 (option selection) | — | N/A | checkpoint:decision (Option A/B/C) | recorded verdict in 07-01-SUMMARY.md `## Broadcast probe verdict` | n/a | ⬜ pending |
| 07-02 T1+T2 | 07-02 | 2 | DOCS-01 | T-7-03 | marked-block contract is opt-in | integration (real binary, real sockets, real processes) | `go test -race ./cmd/sup-kv/ -run TestReadmeQuickstart` | ❌ W0 (`cmd/sup-kv/readme_test.go`) | ⬜ pending |
| 07-03 T1 | 07-03 | 2 | DOCS-03 | — | N/A | integration (every-RPC blocks, output-asserted) | `go test -race ./cmd/sup-kv/ -run TestExamplesDoc` | ❌ W0 (extends 07-01 file) | ⬜ pending |
| 07-03 T2 | 07-03 | 2 | DOCS-02 | T-7-04 | docs must not overstate safety | docs presence-guard (content truth is UAT) | `go test ./cmd/sup-kv/ -run TestOperationsDoc` | ❌ W0 (`cmd/sup-kv/operations_test.go`) | ⬜ pending |
| 07-04 T2 | 07-04 | 2 | (release) | T-7-05/T-7-06 | supply-chain hardening of artifacts | manual gate in task | `make release-snapshot` (`go tool goreleaser release --snapshot --clean`) | ❌ W0 (Makefile target) | ⬜ pending |
| 07-05 T1 | 07-05 | 3 | (release) | T-7-07 | N/A | e2e, once | push tag → observe release.yml + Release assets (≥9) + --version identity | n/a (one-shot) | ⬜ pending |
| 07-05 T2 | 07-05 | 3 | D-11 | T-7-08 | N/A | ops, once | `gh api` POST + verify read per docs/setup/branch-protection.md | n/a (one-shot) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/sup-kv/readme_test.go` — extractor + verbatim runner + PATH shims (covers DOCS-01)
- [ ] `cmd/sup-kv/examples_test.go` — EXAMPLES.md execution (covers DOCS-03)
- [ ] `cmd/sup-kv/main.go` — `reflection.Register(gs)` (prerequisite for DOCS-03 / CONTEXT D-04)
- [ ] go.mod tool directives: goreleaser v2.20.2, grpcurl v1.9.4 (+ tidy gate)
- [ ] Makefile `release` / `release-snapshot`; `.gitignore` `/dist/`
- [ ] Optional: operations-doc presence guard

*Existing infra (TestMain build-once harness, exec harness, CI gate matrix) covers the rest.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| GitHub Release v1.0.0 assets present | DOCS/release | one-shot, external system | after tag push, verify 4 archives + checksums + SBOM on github.com/Djarvur/sup-kv/releases |
| Branch-protection ruleset re-applied | CONTEXT D-11 | one-shot org-settings op | follow docs/setup/branch-protection.md one-command re-apply; verify required checks enforced |
| OPERATIONS.md statements truthful | DOCS-02 | content truth vs prose claim | UAT review against CLAUDE.md constraints, IDEA.md trust model, main.go defaults |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
