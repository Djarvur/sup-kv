# Walking Skeleton — sup-kv

**Phase:** 1
**Generated:** 2026-10-31

## Capability Proven End-to-End

A compileable skeleton commit travels the full pipeline: feature branch cut from up-to-date master → pull request → strict golangci-lint + go vet + race-enabled tests across the 4 OS/arch gate cells → merge to master.

**Adaptation note (explicit):** Phase 1 delivers CI + repo skeleton, not an application. The generic template's DB/UI/deploy slices do not apply to an infrastructure phase; the recorded slice is the delivery pipeline itself — the path every later feature commit takes. This adaptation is deliberate and was set by the planning orchestrator.

## Architectural Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Module | `github.com/Djarvur/sup-kv`, go 1.27, zero direct runtime requires (tool graph indirect-only) | Matches the existing remote (D-02, costly to reverse); dependency bar holds from day one (D-03) |
| Layout | `cmd/sup-kv` + `internal/port` + `internal/app` (go-ultimate service layout) | Adapter dirs appear with the phases that need them (D-01); the decision core is I/O-free by construction |
| Dev tool pinning | `go tool` directives: golangci-lint v2.15.0, govulncheck v1.9.0 | Go 1.24+ pattern; local and CI run identical builds (D-04) |
| Lint gate | Vendored powerman/golangci-lint-strict 2.15.0 (byte-unmodified, Origin header intact) + depguard rules `sup-kv-app-no-io` / `sup-kv-port-stdlib-only` | No inheritance mechanism exists in v2.15.0 — single-file vendoring with additive keys is the verified pattern (D-10 amended); the architecture boundary is machine-enforced (D-11) |
| Go version source | `go.mod` only; CI reads it via `setup-go` `go-version-file` | Single source of truth (D-06) |
| CI matrix | Native full gates: ubuntu-latest, ubuntu-24.04-arm, macos-15; cross cell: darwin-amd64-cross | All four Linux/macOS × amd64/arm64 cells gated at minimal cost (D-05 amended on verified runner facts) |
| Merge gating | Repository ruleset on master (scripted via gh api): PR required, CI checks required, no force pushes/deletions | Server-side enforcement independent of CI config (D-08); plan-gated by the org plan 403 — decision checkpoint in plan 01-04 |
| Dependency currency | Dependabot (gomod + github-actions, weekly, grouped minor+patch) + scheduled govulncheck | Updates as reviewable PRs; continuous CVE scanning; alerts GHAS-gated on private (documented) |

## Stack Touched in Phase 1

- [x] Project scaffold (module init, pinned tools, Makefile, AGENTS.md, .gitignore merge)
- [x] Gate pipeline — local gates (lint/vet/race) and CI matrix proven on a real PR-to-merge cycle
- [x] Architecture boundary — depguard rules proven red/green by probe
- [x] Dependency currency — Dependabot config + dispatched vulncheck run green
- [x] Delivery path — every change (code and docs) lands through a pull request; branch protection scripted (applied or explicitly deferred per the plan-blocker decision)
- [ ] Database / UI / deployment slices — **not applicable** to this infrastructure phase (recorded adaptation; feature phases 2-5 build those slices on this skeleton)

## Out of Scope (Deferred to Later Slices)

- All feature code: engine (Phase 2), persistence (Phase 3), replication + simulator (Phase 4), gRPC API + binary (Phase 5), hardening (Phase 6), docs/release (Phase 7)
- Adapter directories (`internal/in`, `internal/dal`, `internal/out`) — created by the phases that need them (D-01)
- Buf CLI + protoc plugin pinning — first codegen phase (Phase 4/5)
- goreleaser release archives — Phase 7
- Public repo flip (unlocks Dependabot alerts + free arm64 public runners) — user decision at/after v1.0
- gRPC/protobuf entering go.mod — Phase 4/5 respectively (D-03)

## Subsequent Slice Plan

Each later phase adds one vertical slice on top of this skeleton without altering its architectural decisions:

- Phase 2: engine slice inside the depguard-proven boundary (`internal/app` pure decision core + `internal/port` contracts, property tests running under the race gate)
- Phase 3: persistence slice (atomic dumps, `.prev` fallback, write-confirmed-after-dump)
- Phase 4: replication slice (UDP transport adapter, deterministic simulator)
- Phase 5: gRPC API slice (adapters, Watch, startup gate, single static binary)
- Phase 6: hardening slice (failure-scenario suite, metrics, structured logs, benchmarks)
- Phase 7: docs + examples + v1.0 release
