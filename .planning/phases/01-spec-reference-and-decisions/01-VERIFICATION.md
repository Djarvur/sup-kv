---
phase: 01-spec-reference-and-decisions
verified: 2026-11-27T16:04:16Z
status: passed
score: 8/8 must-haves verified
covered_files: [".gitignore", ".planning/phases/01-spec-reference-and-decisions/01-01-PLAN.md", ".planning/phases/01-spec-reference-and-decisions/01-01-SUMMARY.md", ".planning/phases/01-spec-reference-and-decisions/01-02-PLAN.md", ".planning/phases/01-spec-reference-and-decisions/01-02-SUMMARY.md", ".planning/phases/01-spec-reference-and-decisions/01-03-PLAN.md", ".planning/phases/01-spec-reference-and-decisions/01-03-SUMMARY.md", ".planning/phases/01-spec-reference-and-decisions/01-04-PLAN.md", ".planning/phases/01-spec-reference-and-decisions/01-04-SUMMARY.md", "docs/IDEA.md", "docs/SPEC-DEVIATIONS.md", "docs/adr/0001-snapshot-load-full-rollback.md", "docs/adr/0002-create-if-absent-blocked-by-tombstone.md", "docs/adr/0003-dump-policy-unchanged.md", "docs/adr/0004-no-conflict-journal.md", "docs/adr/0005-interface-closed-until-full-key-list.md", "docs/adr/0006-startup-gate-empty-wait.md", "docs/adr/0007-public-proto-consumer-zone.md", "docs/adr/0008-metrics-off-by-default.md", "docs/adr/0009-multi-node-support.md", "docs/adr/0010-darwin-development-only.md", "docs/adr/0011-value-limit-60kib-kept.md", "docs/adr/0012-equal-version-tiebreak.md", "docs/reviews/2026-11-19-v1.0/REVIEW.md", "docs/reviews/2026-11-19-v1.0/evidence/docs-audit.md", "docs/reviews/2026-11-19-v1.0/evidence/repro/README.md", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/clock/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/persist/review_envelope_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/persist/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/repl/review_bench_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/repl/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/storage/review_bench_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/storage/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/metrics/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/persistfile/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/app/repl/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/persist_probe_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/probe_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/socket_probe_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/synctail2_probe_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/synctail3_probe_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/synctail_probe_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/out/bench_darwin.txt", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/out/bench_linux.txt", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/out/mutations.jsonl", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/out/seed_sweep_1_600.txt", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/app/persist/zz_fuzz_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/app/persist/zz_probe_contain_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/app/repl/zz_probe3_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/app/repl/zz_probe4_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/udprepl/zz_fuzz_internal_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/udprepl/zz_probe_big_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/flaketally.py", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/mutate.py", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/mutate_b.py", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/mutlist.py", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/mutlist2.py", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/muttable.py", "docs/reviews/2026-11-19-v1.0/evidence/review-engine.md", "docs/reviews/2026-11-19-v1.0/evidence/review-repl-api.md", "docs/reviews/2026-11-19-v1.0/evidence/stand/RESULTS.md", "docs/reviews/2026-11-19-v1.0/evidence/stand/cpu-a-storm.pprof", "docs/reviews/2026-11-19-v1.0/evidence/stand/docker-compose.yml", "docs/reviews/2026-11-19-v1.0/evidence/stand/g.sh", "docs/reviews/2026-11-19-v1.0/evidence/stand/probe.sh", "docs/reviews/2026-11-19-v1.0/evidence/stand/seed.sh", "docs/reviews/2026-11-19-v1.0/evidence/stand/src-cmd/durcheck/main.go", "docs/reviews/2026-11-19-v1.0/evidence/stand/src-cmd/loadgen/main.go", "docs/reviews/2026-11-19-v1.0/evidence/stand/storm-monitor.log", "docs/reviews/2026-11-19-v1.0/evidence/test-audit.md", "docs/reviews/2026-11-19-v1.0/go.mod"]
covered_digest: "v4:sha256:20d6f71ed75e3ec3ffd091d094b3b935cd36fbe363edb3c3de58dc853f6b3292"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 8/8
  gaps_closed: []
  gaps_remaining: []
  regressions: []
deferred:
  - truth: "Registry statuses are flipped to implemented as the named v1.1 requirements ship, so no row still calls a shipped fix a live gap"
    addressed_in: "Phase 9"
    evidence: "01-CONTEXT.md Deferred: 'Flipping registry statuses to fixed-in-v1.1(<REQ>) as fixes land — Phases 3–7; final no unclosed rows check — Phase 9 (milestone exit criterion)'; ROADMAP Phase 9 goal: 'the deviation registry fully closed'; milestone exit criterion: 'the deviation registry has no unclosed rows'"
---

# Phase 1: Spec Reference and Decisions — Verification Report

**Phase Goal:** Every deviation between sup-kv and its Spec is visible and every settled owner decision is recorded — a reader can always answer "does sup-kv implement Spec statement X, and if not, who decided that and where is the code?"
**Verified:** 2026-11-27T16:04:16Z
**Status:** passed
**Re-verification:** Yes — stale-verification regeneration. The tree moved 37 commits past the 2026-11-23 verification (Phases 2–7 complete, Phase 8 complete); exactly one covered implementation file changed since (`docs/SPEC-DEVIATIONS.md`, commit `d89b1b4`: Phase 8's legitimate S-FAIL-15 summary enrichment — row count and status vocabulary unchanged). Every truth was re-verified against the current tree, not carried forward from the prior report.

## Goal Achievement

Must-haves merged from ROADMAP success criteria (the contract, T1–T4) and non-duplicating PLAN frontmatter truths (T5–T8). No PLAN truth subtracts from the roadmap set. The registry has legitimately evolved with the milestone (row set still 167; 126 rows now `implemented`; ADR 0012 present for the P-07 tie-break); every phase-01 invariant still holds on the current tree.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A deviation registry exists mapping every Spec statement ID to a decision and a code/ADR link — any statement findable in one lookup (SC-1) | ✓ VERIFIED | `docs/SPEC-DEVIATIONS.md`: 167 rows, 13 sections, 261 lines, committed (1 tracked file). All 167 status cells non-empty and inside the six-value enum (awk gates exit 0). One-lookup spot-checks (`S-ALG-RT-02`, `S-PAR-09`, `S-PERS-07`, `S-DM-04`) each found in a single grep with status and links. Sampled citations resolve on the current tree: `internal/config/config.go:96` reads `defaultAckTimeout = 200 * time.Millisecond` (matches S-PAR-09), `internal/app/persist/snapshot.go:110` is `func (e *Engine) Restore(...)` (matches S-PERS-12), `internal/grpcapi/gate.go:120` (RunGate), `internal/app/clock/clock.go:87` (wall-clock drift), `internal/app/persist/engine.go:227` all exist and match their row summaries |
| 2 | Each review §7 deviation (S-01, S-02, S-06, S-07, S-08, S-12) is closed by code-forward-reference or accepted-deviation — no row empty or TBD (SC-2) | ✓ VERIFIED | S-01 → `S-PERS-12` now `implemented` (closed by code; full citations `internal/app/persist/snapshot.go:110-157` Restore + engine/grpcapi tests + ADR 0001 + REVIEW §7 S-01 in Links); S-02 → `S-PROTO-14/ALG-CU-09/PERS-13/SEM-08` all `fixed-in-v1.1(PROTO-08)`; S-06 → `S-HLC-13` implemented + `S-FAIL-14` `fixed-in-v1.1(OBSV-05)`; S-07 → `S-ALG-CU-08`/`S-PROTO-04` `fixed-in-v1.1(OBSV-06)` + `S-PERS-09` implemented; S-08 → all four rows (`S-HLC-04`, `S-ALG-RC-05`, `S-PROTO-18`, `S-ALG-ST-06`) now `implemented`; S-11 → 4 × `consumer-zone` + `deferred-v2(SNAP-01)`; S-12 → registry preamble (lines 22–24), not a row, per D-08. `grep TBD\|FIXME\|XXX` over the registry → 0; empty-status gate → 0. Later phases flipping closure tokens (APIS-07, PROTO-01) out of status cells is closure-by-code, exactly the first closure path SC-2 names |
| 3 | Eleven ADRs record decisions 14.1–14.11 incl. the two accepted-by-default confirmations of 2026-11-19 — 14.5(6) Unavailable rule and 14.7 last-read-value outside sup-kv (SC-3) | ✓ VERIFIED | `docs/adr/0001..0011-*.md` all present and tracked (11 of 12 tracked files; the 12th, `0012-equal-version-tiebreak.md` with `review-decision: "P-07 ..."`, is a later-phase addition that does not displace the 14.x set — the census over `review-decision:` values covers 14.1–14.11 each exactly once). D-11 line `accepted (by default; reopenable at Phase 4/6 discussion before implementation)` appears in exactly 0005 and 0007 (1 hit each, 0 elsewhere); wording mirrors the SPEC-02 acceptance parenthetical at `.planning/REQUIREMENTS.md:15` |
| 4 | docs/IDEA.md is marked historical — no reader mistakes the Spec for current behavior (SC-4) | ✓ VERIFIED | 6-line loud banner (blockquote + bold) above the H1: "HISTORICAL DOCUMENT ... NOT a description of current behavior", with as-built trio links (README, docs/OPERATIONS.md, docs/EXAMPLES.md) in the repo's relative-link style and the Phase 9 note for docs/ARCHITECTURE.md. Banner commit `3b9569d` numstat: 6 added / 0 deleted; `git log 3b9569d..HEAD -- docs/IDEA.md` is empty — zero later commits, body byte-untouched at 239 lines incl. the five D-13 conflict-journal lines |
| 5 | Review committed at docs/reviews/2026-11-19-v1.0/ (REVIEW.md + 46 evidence files); local-only internal/ uncommitable from any clone; stale pre-move refs swept (plan 01) | ✓ VERIFIED | REVIEW.md tracked (1) + 46 evidence files tracked (+ the wave-1 `go.mod`). Guard matches at `.gitignore:42` (guard block present at the file tail; line drifted from :35 as later phases appended entries — match, not position, is the invariant). `git ls-files -- docs/reviews/2026-11-19-v1.0/internal/` → 0; none of the four non-public filenames tracked anywhere (0); zip absent; old nested dir absent; stale-path sweep (`git grep '2026-11-19-v1.0-review'` outside `.planning/phases`/`.planning/milestones`) → 0 |
| 6 | Registry structure per D-06/D-07/D-08: 167 rows in D-06 schema, 13 sections, six-value enum only, no empty cells, own-words summaries (plan 02) | ✓ VERIFIED | Row count 167; 17 prefixes, each count == max suffix (contiguity gate exit 0); exactly 13 `###` headings; enum-membership gate → 0 violations; empty-status gate → 0; census: 126 implemented + 22 fixed-in-v1.1 + 12 superseded-by-amendment + 4 consumer-zone + 2 accepted-deviation + 1 deferred-v2 = 167. Own-words substring audit over the local-only 167-row trace (`docs/reviews/2026-11-19-v1.0/internal/spec-trace.md`, intact, 167 rows) → exit 0, no VERBATIM lines |
| 7 | Each ADR carries the explicit 14.x ↔ ADR ↔ informed-REQ mapping via lowercase-key frontmatter, slim MADR shape (plan 03) | ✓ VERIFIED | All 11 phase-01 ADRs have exactly the 5 keys (`adr`, `review-decision`, `informs`, `date`, `status`); zero options-analysis/alternatives headings; ADR 0002 `informs: []` with Out-of-Scope rationale; ADR 0008 Context resolves O-04 (3 hits); ADR 0011 states the 14.11 loss-share measurement "has no dedicated v1.1 requirement"; ADR 0005/0007 carry the verbatim reopenable line under the H1 |
| 8 | Cross-artifact consistency and repo quality gates green on the committed tree (plan 04) | ✓ VERIFIED | Every ADR number cited in registry Links cells ({0001, 0004, 0007, 0008, 0012}) resolves to an existing file — 0 unresolved; ADR 0001 back-cites its four rows (S-PERS-12/S-PERS-13/S-SEM-08/S-ALG-CU-09), ADR 0007 its five rows; registry preamble cites the committed REVIEW.md (55 REVIEW references); spec-identifying-string sweep (`confstor\|0031\|cyp-config\|cyp-perf`) over committed REVIEW.md + evidence/ → 0 hits; stale-path sweep → 0. `make lint` exit 0. `make test` exit 0 — full suite (19 packages, `-race -cover`), all ok, zero FAIL; the one flake the previous verification observed (`TestMain_socketBuffersLogged`) is gone on the current HEAD |

**Score:** 8/8 truths verified (0 present, behavior-unverified)

All eight truths are documentation-structure truths with deterministic machine evidence; none asserts a runtime state transition or a cancellation/cleanup/ordering invariant, so none is behavior-dependent. The strongest checks beyond presence: the registry's 167-row ID sequence and per-prefix contiguity against the local-only trace, the own-words substring audit, and registry-cited code paths opened and matched to their row summaries on the current tree.

### Deferred Items

Items not yet met but explicitly addressed in later milestone phases.

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | 22 registry rows still read `fixed-in-v1.1(<REQ>)` (XPORT-01/02/03/07, PROTO-03/04/08, OBSV-05/06, PERF-02 — all marked Complete in REQUIREMENTS.md traceability); the final flip to `implemented` is a later-phase duty | Phase 9 | 01-CONTEXT.md Deferred: "Flipping registry statuses ... Phases 3–7; final no unclosed rows check — Phase 9 (milestone exit criterion)"; ROADMAP Phase 9 goal: "the deviation registry fully closed"; milestone exit criterion: "the deviation registry has no unclosed rows" |

### Advisory (New Scope, Unevidenced)

None. Every current-tree observation in this regeneration carries deterministic evidence (commands and outputs recorded above/below); nothing is reported on inference alone. The WR/IN findings below are pre-existing, ledgered in `01-REVIEW-DISPOSITION.md` (recorded 2026-11-24, 9 open), not new findings of this pass.

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `docs/SPEC-DEVIATIONS.md` | Deviation registry: preamble + 13 sections + 167 D-06 rows | ✓ VERIFIED | 261 lines; all gates green; tracked; last touched by `d89b1b4` (Phase 8 S-FAIL-15 enrichment, in-place per the flip-don't-restructure rule) |
| `docs/adr/` (0001..0011) | 11 slim-MADR ADRs for 14.1..14.11 | ✓ VERIFIED | 11 phase-01 files tracked, 5-key frontmatter each (a 12th, `0012-equal-version-tiebreak.md`, was added by a later phase and does not displace the set) |
| `docs/reviews/2026-11-19-v1.0/REVIEW.md` | Committed review record (§7/§13/§14 source) | ✓ VERIFIED | Tracked; §7 at :238, §13 at :354, §14 at :392 |
| `docs/reviews/2026-11-19-v1.0/evidence/` | 46-file evidence archive committed | ✓ VERIFIED | 46 tracked evidence files (plus REVIEW.md and the wave-1 `go.mod`) |
| `.gitignore` | Guard block matching `docs/reviews/2026-11-19-v1.0/internal/` | ✓ VERIFIED | Committed guard block at file tail; `check-ignore -v` matches at `.gitignore:42`; 0 tracked paths under the prefix |
| `docs/IDEA.md` | Historical banner, body byte-untouched incl. five D-13 lines | ✓ VERIFIED | 6 added / 0 deleted at `3b9569d`; no later commits; banner anchors and trio links present |
| `.planning/PROJECT.md` | :66 parenthetical names IDEA.md historical intent-at-start | ✓ VERIFIED | Wording present; post-move review-path references in place (2 hits) |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| `.gitignore` | `docs/reviews/2026-11-19-v1.0/internal/spec-trace.md` | check-ignore guard | ✓ WIRED | Match at `.gitignore:42`; 0 tracked paths under the guarded prefix; 0 non-public filenames tracked anywhere; local-only internal/ intact on disk (5 files) and still untracked |
| `docs/IDEA.md` | `docs/OPERATIONS.md` (+README, EXAMPLES) | banner trio links | ✓ WIRED | Relative-link style present in the banner |
| `.planning/PROJECT.md` | `docs/IDEA.md` | :66 historical parenthetical | ✓ WIRED | "historical intent-at-start" + banner pointer + as-built trio |
| `docs/SPEC-DEVIATIONS.md` | `docs/reviews/2026-11-19-v1.0/internal/spec-trace.md` | scripted row/status derivation | ✓ WIRED | Local-only trace present with 167 rows; contiguity gate exit 0; own-words audit exit 0; row set unchanged since seeding |
| `docs/SPEC-DEVIATIONS.md` | `docs/adr/0001..0012` | Links cells cite ADR numbers | ✓ WIRED | Cited numbers {0001, 0004, 0007, 0008, 0012} all resolve; ADRs 0001/0007 back-cite their named rows |
| `docs/SPEC-DEVIATIONS.md` | `docs/reviews/2026-11-19-v1.0/REVIEW.md` | preamble + Links REVIEW refs | ✓ WIRED | Preamble cites the committed review; 55 REVIEW references across the registry |
| `docs/adr/0001` | REVIEW.md §14 row 1 | frontmatter + Links | ✓ WIRED | `review-decision: "14.1 (Snapshot load is a full rollback as per Spec)"`; Links name the four registry rows and the §14 source |
| `docs/adr/0005` | `.planning/REQUIREMENTS.md` | informs PROTO-01; D-11 wording | ✓ WIRED | D-11 line verbatim under the H1; wording mirrors the SPEC-02 acceptance parenthetical (REQUIREMENTS.md:15) |
| `docs/adr/` | `docs/SPEC-DEVIATIONS.md` | bidirectional registry↔ADR citation | ✓ WIRED (with caveat) | Resolution loop over all cited ADR numbers → 0 unresolved. Caveat: ADR 0005's Links bullet still over-claims four back-references (0 registry rows cite ADR 0005) — see WR-01 below; the registry→decision→code lookup the goal promises is unaffected |

### Data-Flow Trace (Level 4)

For a docs phase, "data" is the cited evidence resolving to something real — re-checked on the current tree, since later phases moved the cited code.

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| Registry rows | Code path links | Live repo files | `internal/config/config.go:96` = `defaultAckTimeout = 200 * time.Millisecond` (S-PAR-09); `internal/app/persist/snapshot.go:110` = `func (e *Engine) Restore(...)` (S-PERS-12); `internal/grpcapi/gate.go:120` (RunGate); `internal/app/clock/clock.go:87` (wall-clock drift, S-FAIL-12/13/14); `internal/app/persist/engine.go:227` (S-FAIL-13/15) — files exist and content matches the row summaries | ✓ FLOWING |
| Registry statuses | Trace verdicts + REVIEW §7/§13/§14 | Local-only spec-trace.md (167 rows on disk) + committed REVIEW.md (§7 :238, §13 :354, §14 :392) | 1:1 ID join; contiguity exit 0; own-words audit exit 0; statuses evolved in place without adding/removing rows | ✓ FLOWING |
| ADR Context/Decision | REVIEW §14 rows 1–11 | Committed REVIEW.md:392–408 | Census of `review-decision:` values covers 14.1–14.11 exactly once each | ✓ FLOWING |
| S-PERS-06, S-SEM-03 Links | Code evidence | Trace's own `n/a` code column | Em-dash mirrors the source convention; status remains visible | ℹ️ INFO (per-convention, not hollow — the trace itself carries no code evidence for these two) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| SC-1 one-lookup | `grep -E '^\| S-(ALG-RT-02\|PAR-09\|PERS-07\|DM-04) '` | all four rows found with status and links | ✓ PASS |
| SC-2 §7 closures | per-family row extraction | all six families present and closed per D-08 | ✓ PASS |
| Registry completeness | row count / contiguity / empty-status / enum awk gates | 167; 17 prefixes contiguous; 0 empty; 0 off-enum | ✓ PASS |
| Own-words audit | awk substring join trace↔registry | exit 0, no VERBATIM lines | ✓ PASS |
| ADR mapping | `review-decision:` census + file list | 11 unique values covering 14.1–14.11; 11 phase-01 files tracked | ✓ PASS |
| D-11 placement | `grep -l 'reopenable at Phase 4/6 discussion before implementation'` | exactly 0005 and 0007 | ✓ PASS |
| Publication guard | `git check-ignore -v` + tracked-prefix count | `.gitignore:42` match; 0 tracked under prefix; 0 non-public filenames tracked anywhere | ✓ PASS |
| Stale-path sweep | `git grep '2026-11-19-v1.0-review'` excluding phase archives | 0 files | ✓ PASS |
| Spec-identifying-string sweep | grep over REVIEW.md + evidence/ | 0 hits | ✓ PASS |
| Lint gate | `make lint` | exit 0 | ✓ PASS |
| Test gate (full suite, run once) | `make test` | exit 0 — 19 packages ok under `-race -cover`, zero FAIL | ✓ PASS |

### Probe Execution

No probes declared or conventional probes present (`find scripts -path '*/tests/probe-*.sh'` → 0; the phase files name none). Plan 04 introduces no scripts by design — SKIPPED (docs-only phase, no runnable probes).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| SPEC-01 | 01-01, 01-02, 01-04 | Registry maps every Spec statement ID to decision + code/ADR link; every §7 deviation closed; IDEA.md historical | ✓ SATISFIED | Truths 1, 2, 4, 5, 6 above; REQUIREMENTS.md Traceability marks SPEC-01 Phase 1 Complete |
| SPEC-02 | 01-03, 01-04 | Decisions 14.1–14.11 recorded as ADRs incl. the two accepted confirmations (reopenable at Phase 4/6) | ✓ SATISFIED | Truths 3, 7 above; D-11 line verbatim in ADRs 0005/0007 |

Orphaned requirements: none — REQUIREMENTS.md traceability maps exactly SPEC-01 and SPEC-02 to Phase 1, and both are claimed by plans (01-01/01-02/01-04 and 01-03/01-04 respectively).

### Decision Coverage

All 13 trackable CONTEXT.md decisions (D-01..D-13) are honored by the shipped artifacts. The automated gate `check decision-coverage-verify` ran clean this time (`01-CONTEXT.md` resolved by glob): `total: 13, honored: 13, not_honored: []` — "All trackable CONTEXT.md decisions are honored by shipped artifacts."

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| docs/adr/0005-interface-closed-until-full-key-list.md | 57 | Links bullet claims registry rows S-HLC-04, S-ALG-RC-05, S-ALG-ST-06, S-PROTO-18 cite "this ADR in its link column" — 0 registry rows cite ADR 0005 (grep confirmed), and the bullet's `fixed-in-v1.1(PROTO-01)` wording is now doubly stale: those rows are `implemented` | ⚠️ Warning | Factual over-claim in a convenience back-reference; the goal's lookup chain (registry row → REQ/REVIEW → code) is intact. Review finding WR-01, open in `01-REVIEW-DISPOSITION.md` |
| docs/reviews/2026-11-19-v1.0/evidence/review-repl-api.md | ~12 places | Translated Spec references in committed evidence, vs REVIEW §0's evidence-clean claim | ⚠️ Warning | Publication-quality owner call; spec-identifying-string sweep still 0. Review finding WR-02, open in the disposition ledger |
| docs/SPEC-DEVIATIONS.md | 22 rows | `fixed-in-v1.1(<REQ>)` retained for requirements REQUIREMENTS.md marks Complete | ⚠️ Warning | Registry status maintenance lag owned by Phases 3–7 and closed by Phase 9's registry-closed sweep (see Deferred Items); not a phase-01 execution defect |
| docs/SPEC-DEVIATIONS.md (row S-HLC-07) | — | `accepted-deviation(02-RESEARCH A2)` stretches the D-08 `<ADR\|14.x>` grammar | ℹ️ Info | Honest reference, documented in the plan 02 summary; enum shape and machine gate unaffected. Review finding IN-01, open |
| docs/reviews/.../REVIEW.md, evidence/ (WR-03, IN-02..IN-06) | — | Ledgered open review findings (stale guard-mechanism description, uncommitted-evidence inventory drift, repro-ID drift, §16 table gaps, ~unresolvable G-xx citations) | ℹ️ Info | Recorded 2026-11-24 in `01-REVIEW-DISPOSITION.md` (9 open of 9); owner triage / Phase 9 scope per the ledger. None touches the goal's lookup chain |

Debt-marker gate: `TBD`/`FIXME`/`XXX` → 0 in every authored phase file (registry, 11 ADRs, IDEA.md, .gitignore); placeholder/not-yet-implemented language → 0. No blockers.

### Human Verification Required

N/A — documentation-only phase with no user-facing runtime elements. All four roadmap success criteria and all four plan truths are machine-verifiable and were re-verified programmatically on the current tree (lookups, structural gates, guard, sweeps, lint, full test suite). No truth is behavior-dependent, so no ⚠️ PRESENT_BEHAVIOR_UNVERIFIED items exist. The editorial-judgment items flagged during execution (banner loudness, ADR prose fidelity to §14, near-verbatim avoidance beyond the exact-substring audit) were dispositioned with rationale in the plan summaries and independently examined by the phase's code review (9 findings, 0 critical); the sampled registry summaries read as genuine own-words glosses. Residual judgment risk is carried by the open disposition ledger for owner triage.

### Gaps Summary

None blocking. The phase goal is achieved on the current tree: for any of the 167 Spec statement IDs, a reader finds status, decision and code/ADR link in one lookup (`docs/SPEC-DEVIATIONS.md`, proven 1:1 against the derivation trace with an unchanged row set); all six §7 deviation families are closed (S-12 via the preamble, exactly as D-08 specifies); eleven ADRs record 14.1–14.11 with the two loud accepted-by-default lines carrying the 14.5(6) Unavailable rule and the 14.7 last-read-value placement; IDEA.md is loudly historical over a byte-untouched body; and the publication guard makes the non-public Spec material uncommitable from any clone while nothing identifying has entered git. The post-phase evolution (126 rows now `implemented`, ADR 0012 added, S-FAIL-15 summary enriched with burst bounds and test names) is legitimate later-phase registry maintenance and did not disturb any phase-01 invariant.

Two items are explicitly not phase-01 gaps: (1) the 22 stale `fixed-in-v1.1` statuses are deferred to Phase 9's registry-closed sweep per 01-CONTEXT.md's own deferred list and the milestone exit criterion; (2) the open review findings are quality nits on phase artifacts, ledgered with dispositions in `01-REVIEW-DISPOSITION.md`, none touching the goal.

Plan 04's seven flagged probe assumptions (SPEC-01/SPEC-02 adjacency/empty/ordering/concurrency probes) remain honestly marked N/A-for-docs in the plan frontmatter and carry forward to the code phases that build data structures — recorded, not dropped.

---

_Verified: 2026-11-27T16:04:16Z_
_Verifier: Claude (gsd-verifier)_
