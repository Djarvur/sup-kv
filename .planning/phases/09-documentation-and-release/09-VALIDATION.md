---
phase: "9"
slug: "documentation-and-release"
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-11-27"
---

# Phase 9 — Validation Strategy

> Seed contract for the documentation-and-release phase, grounded in
> `09-RESEARCH.md` §11 (Validation Architecture). The planner refines the
> per-task map below to one row per task once `09-*-PLAN.md` files exist;
> statuses stay `pending` until the plan's command has run on the committed
> tree.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go 1.27 `testing` + `go tool golangci-lint` v2.15.0; doc-anchor tests in `cmd/sup-kv` (five precedents: readme, operations, api, examples, plus the build-once/shim/port-band harness) |
| **Config file** | `go.mod`, `.golangci.yml`, `Makefile`, `.github/workflows/{ci,release}.yml`, `.goreleaser.yaml` |
| **Quick run command** | `go test -race -count=1 ./cmd/sup-kv/` |
| **Full suite command** | `make lint && make test && make buf-lint && make buf-breaking && make client-example` |
| **Release checks** | goreleaser snapshot dry-run + archive inspection (`tar -tzf`) for shipped docs/unit; `systemd-analyze verify` on the unit |
| **Estimated runtime** | per-task commands ≤ 120 s; `make test` ~6 min under `-race`; snapshot release dry-run ~2–4 min |

---

## Sampling Rate

- **After every task commit:** the task's own `<automated>` command (the map below).
- **After every plan wave:** `go build ./... && make test`.
- **Before `/gsd:verify-work`:** Full suite green; the registry gate (zero table-cell
  `fixed-in-v1.1(` rows) green on the final tree.
- **Max feedback latency:** ~360 s (full suite); doc-anchor commands are seconds-scale.

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (planner refines to one row per task) | TBD | TBD | DOCS-01 | — | N/A | doc-anchor | registry gate in `cmd/sup-kv` | ❌ W0 | ⬜ pending |
| (planner refines) | TBD | TBD | DOCS-02 | — | N/A | doc-anchor | `--help`-driven flags iterator vs `docs/CONFIGURATION.md` | ❌ W0 | ⬜ pending |
| (planner refines) | TBD | TBD | DOCS-04/07 | — | N/A | doc-anchor | two-directional metrics cross-check + log-literal grep | ❌ W0 | ⬜ pending |
| (planner refines) | TBD | TBD | DOCS-06 | — | N/A | codec | byte-identical FORMATS encode beside the codec | ❌ W0 | ⬜ pending |
| (planner refines) | TBD | TBD | DOCS-09 | — | N/A | doc-anchor | proven-claim checker + metric-rename-breaks-CI coupling | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/sup-kv` doc-anchor package grows the new machine-check tests (registry gate,
      flags iterator, metrics/log cross-check, proven-claim checker) — the package and its
      harness already exist, so Wave 0 is new test files, not new infrastructure
- [ ] `docs/FORMATS.md` byte-identical encode test sits beside the codec it pins
- [ ] No new framework install needed — Go testing covers everything

*Existing infrastructure covers all phase requirements beyond the new test files themselves.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| The actual `v1.1.0` tag publishes release assets | DOCS-10 | Tag-push triggers GitHub-hosted release.yml; CI cannot pre-verify its own publication | After the tag lands: inspect the GitHub Release assets (archives, SBOMs, checksums; darwin absent; systemd unit + docs present) |
| Ruleset check command against the live repo | DOCS-10 | Requires `gh api` against the real repository | Execute the fixed by-id command (`gh api repos/Djarvur/sup-kv/rulesets/27576625`) as written; record the output in the plan summary |

*All other phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 360 s
- [ ] `nyquist_compliant: true` set in frontmatter (after the planner fills the map)

**Approval:** pending
