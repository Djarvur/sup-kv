# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v1.0 — sup-kv v1.0

**Shipped:** 2026-11-18
**Phases:** 7 | **Plans:** 38 | **Timeline:** 2026-10-30 → 2026-11-18 (19 days)

### What Was Built
- Leaderless replicated in-memory KV store: HLC/UUIDv7 versioning + LWW-Element-Map CRDT core (Phase 2, property-proven)
- Human-readable JSON persistence with the write-confirmed-after-dump contract (Phase 3)
- UDP broadcast replication proven under chaos by an in-repo deterministic simulator (Phase 4)
- gRPC API with Watch, startup gate, `wait_for_replication` — one static binary (Phase 5)
- Failure-scenario suite (simulator + real sockets + 4 two-process exec scenarios), opt-in mTLS metrics, slog (Phase 6)
- Machine-verified documentation (CI executes the README/EXAMPLES/OPERATIONS blocks verbatim), goreleaser pipeline, **v1.0.0 released** (Phase 7: 4 archives + SBOMs + checksums on GitHub)

### What Worked
- Dependency-forced phase order (engine ← persistence ← replication ← API) meant zero rework across 38 plans — no plan ever invalidated a predecessor
- Doc-is-test-input: documentation correctness enforced like code correctness — the quickstart cannot drift from reality; the WR-01 finding (bare `sup-kv` not on PATH) proved the review layer catches what the harness initially masked
- Ratified-override discipline: CI-02/CI-03 external-plan limitations were recorded, dated, re-verified at close, and never silently dropped — milestone audit passed with them as documented partials
- Checkpoint design in plans: the two human decisions (macOS option-b, merge-before-tag) arrived with evidence tables and were resolved in minutes

### What Was Inefficient
- Verification fingerprints went stale after every later phase touched covered files — five phases needed re-verification at close (predicted as D-16, but the regeneration pass is ~1h of agent time that a per-phase refresh cadence could have avoided)
- Two interruptions (API 429 rate limit, host restart) hit the re-verification run; resume-from-transcript worked but cost a cycle
- macOS loopback-broadcast limitation was discovered empirically at execution time (phase-6 D-12) rather than probed at planning time — a one-run macos-15 probe during research would have moved the option-b decision a week earlier

### Patterns Established
- Machine-anchored docs: `<!-- quickstart-verify -->` marker blocks + PATH-shimmed doc tests riding the build-once harness
- Release tooling as go.mod tool directives (`go tool goreleaser`), tag-triggered workflow, ci.yml as an untouchable ruleset contract
- Loud-skip on platform-impossible tests (darwin broadcast) with verbatim verdict evidence in the skip message
- Executive evidence in SUMMARYs: verbatim CI verdicts, run IDs, asset counts — verifiers re-checked claims first-hand instead of trusting prose

### Key Lessons
1. A linear milestone converges verification debt — schedule fingerprint regeneration per phase, not at close
2. Empirical platform facts (broadcast behavior) belong in research probes on the real CI runner, not in plan assumptions
3. When a premise in CONTEXT.md is falsified by code (reflection not registered), fix the plan sequence immediately — the one-line tracer task made every later grpcurl example work

### Cost Observations
- ~328 commits, 19 days, 7 phases; heavy subagent delegation throughout (executor, verifier, reviewer, checker, integration, mapper roles)
- Two 429 interruptions; one Orca host restart mid-run — resume mechanics (SendMessage + transcript) recovered both without work loss

## Cross-Milestone Trends

| Milestone | Phases | Plans | Days | Notes |
|-----------|--------|-------|------|-------|
| v1.0 | 7 | 38 | 19 | First release; all phases verified; 2 ratified overrides |
