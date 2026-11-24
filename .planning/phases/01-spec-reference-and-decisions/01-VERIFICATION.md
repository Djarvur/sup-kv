---
phase: 01-spec-reference-and-decisions
verified: 2026-11-23T19:35:00Z
status: passed
score: 8/8 must-haves verified
covered_files: [".gitignore", ".planning/phases/01-spec-reference-and-decisions/01-01-PLAN.md", ".planning/phases/01-spec-reference-and-decisions/01-01-SUMMARY.md", ".planning/phases/01-spec-reference-and-decisions/01-02-PLAN.md", ".planning/phases/01-spec-reference-and-decisions/01-02-SUMMARY.md", ".planning/phases/01-spec-reference-and-decisions/01-03-PLAN.md", ".planning/phases/01-spec-reference-and-decisions/01-03-SUMMARY.md", ".planning/phases/01-spec-reference-and-decisions/01-04-PLAN.md", ".planning/phases/01-spec-reference-and-decisions/01-04-SUMMARY.md", "docs/IDEA.md", "docs/SPEC-DEVIATIONS.md", "docs/adr/0001-snapshot-load-full-rollback.md", "docs/adr/0002-create-if-absent-blocked-by-tombstone.md", "docs/adr/0003-dump-policy-unchanged.md", "docs/adr/0004-no-conflict-journal.md", "docs/adr/0005-interface-closed-until-full-key-list.md", "docs/adr/0006-startup-gate-empty-wait.md", "docs/adr/0007-public-proto-consumer-zone.md", "docs/adr/0008-metrics-off-by-default.md", "docs/adr/0009-multi-node-support.md", "docs/adr/0010-darwin-development-only.md", "docs/adr/0011-value-limit-60kib-kept.md", "docs/reviews/2026-11-19-v1.0/REVIEW.md", "docs/reviews/2026-11-19-v1.0/evidence/docs-audit.md", "docs/reviews/2026-11-19-v1.0/evidence/repro/README.md", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/clock/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/persist/review_envelope_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/persist/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/repl/review_bench_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/repl/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/storage/review_bench_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/storage/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/metrics/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/persistfile/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/app/repl/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/persist_probe_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/probe_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/socket_probe_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/synctail2_probe_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/synctail3_probe_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/synctail_probe_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/out/bench_darwin.txt", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/out/bench_linux.txt", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/out/mutations.jsonl", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/out/seed_sweep_1_600.txt", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/app/persist/zz_fuzz_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/app/persist/zz_probe_contain_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/app/repl/zz_probe3_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/app/repl/zz_probe4_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/udprepl/zz_fuzz_internal_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/udprepl/zz_probe_big_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/flaketally.py", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/mutate.py", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/mutate_b.py", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/mutlist.py", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/mutlist2.py", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/muttable.py", "docs/reviews/2026-11-19-v1.0/evidence/review-engine.md", "docs/reviews/2026-11-19-v1.0/evidence/review-repl-api.md", "docs/reviews/2026-11-19-v1.0/evidence/stand/RESULTS.md", "docs/reviews/2026-11-19-v1.0/evidence/stand/cpu-a-storm.pprof", "docs/reviews/2026-11-19-v1.0/evidence/stand/docker-compose.yml", "docs/reviews/2026-11-19-v1.0/evidence/stand/g.sh", "docs/reviews/2026-11-19-v1.0/evidence/stand/probe.sh", "docs/reviews/2026-11-19-v1.0/evidence/stand/seed.sh", "docs/reviews/2026-11-19-v1.0/evidence/stand/src-cmd/durcheck/main.go", "docs/reviews/2026-11-19-v1.0/evidence/stand/src-cmd/loadgen/main.go", "docs/reviews/2026-11-19-v1.0/evidence/stand/storm-monitor.log", "docs/reviews/2026-11-19-v1.0/evidence/test-audit.md", "docs/reviews/2026-11-19-v1.0/go.mod"]
covered_digest: "v4:sha256:6c7e4bc2a7d22ee1f5fa0fe8db7fa6819bf52bd71d5074cc92dbaa071b528335"
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
    evidence: "01-CONTEXT.md Deferred: 'Flipping registry statuses to fixed-in-v1.1(<REQ>) as fixes land — Phases 3–7; final no unclosed rows check — Phase 9 (milestone exit criterion)'; ROADMAP Phase 9 goal includes 'registry closed'; milestone overview exit criterion: 'the deviation registry has no unclosed rows'"
---

# Phase 1: Spec Reference and Decisions — Verification Report

**Phase Goal:** Every deviation between sup-kv and its Spec is visible and every settled owner decision is recorded — a reader can always answer "does sup-kv implement Spec statement X, and if not, who decided that and where is the code?"
**Verified:** 2026-11-23T19:35:00Z
**Status:** passed
**Re-verification:** Yes — regenerated because covered inputs (`.gitignore`, `docs/SPEC-DEVIATIONS.md`) changed after the 2026-11-19 verification

## Goal Achievement

Must-haves merged from ROADMAP success criteria (the contract, T1–T4) and non-duplicating PLAN frontmatter truths (T5–T8). No PLAN truth subtracts from the roadmap set. This regeneration re-verified every truth against the current tree rather than the stale report: the tree is now past Phases 2–7 with Phase 8 in progress, and the registry has legitimately evolved (row count still 167; 126 rows now `implemented`; ADR 0012 added for the P-07 tie-break). Every phase-01 invariant still holds.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A deviation registry exists mapping every Spec statement ID to a decision and a code/ADR link — any statement findable in one lookup (SC-1) | ✓ VERIFIED | `docs/SPEC-DEVIATIONS.md`: 167 rows, 13 sections, 261 lines, committed. All 167 status cells non-empty and inside the six-value enum; 0 empty Links cells. One-lookup spot-checks (`S-ALG-RT-02`, `S-PAR-09`, `S-PERS-07`, `S-DM-04`) each found in a single grep with status and links. Sampled citations resolve: `internal/app/storage/store.go`, `internal/app/persist/snapshot.go:110` (`Restore`), `internal/app/repl/core.go:1510` (`handleDataSync`), `internal/grpcapi/gate.go:120` (`RunGate`) all exist and match their row summaries |
| 2 | Each review §7 deviation (S-01, S-02, S-06, S-07, S-08, S-12) is closed by code-forward-reference or accepted-deviation — no row empty or TBD (SC-2) | ✓ VERIFIED | S-01 → `S-PERS-12` now `implemented` (APIS-07 shipped) with ADR 0001 and REVIEW §7 S-01 in Links; S-02 → `S-PROTO-14/ALG-CU-09/PERS-13/SEM-08` all carry `fixed-in-v1.1(PROTO-08)`; S-06 → `S-HLC-13` implemented + `S-FAIL-14` `fixed-in-v1.1(OBSV-05)`; S-07 → `S-ALG-CU-08`/`S-PROTO-04` `fixed-in-v1.1(OBSV-06)` + `S-PERS-09` implemented; S-08 → rows under `fixed-in-v1.1(PROTO-03/04)` and `implemented` (`S-HLC-04`, `S-ALG-RC-05`, `S-PROTO-18`, `S-ALG-ST-06`); S-11 → 4 × `consumer-zone` + `deferred-v2(SNAP-01)`; S-12 → registry preamble (lines 22–27), not a row per D-08. `grep TBD\|FIXME\|XXX` over the registry → 0; empty-status gate → 0 |
| 3 | Eleven ADRs record decisions 14.1–14.11 incl. the two accepted-by-default confirmations of 2026-11-19 — 14.5(6) Unavailable rule and 14.7 last-read-value outside sup-kv (SC-3) | ✓ VERIFIED | `docs/adr/0001..0011-*.md` all present and tracked (11 files). 11 unique `review-decision:` values cover 14.1–14.11 exactly once. ADR 0005 Decision rule 6: "reading a key that is listed but whose value has not arrived yet answers `Unavailable` — the accepted-by-default rule this ADR carries the loud status line for". ADR 0007 Decision: "The last-read-value file belongs to the consumer's client library: the pattern is documented, and the code lives with the consumer, not in sup-kv". The D-11 line appears in exactly 0005 and 0007. ADR 0012 (later phase, P-07 tie-break) is additive and does not displace the 14.x set |
| 4 | docs/IDEA.md is marked historical — no reader mistakes the Spec for current behavior (SC-4) | ✓ VERIFIED | 6-line loud banner (blockquote + bold) above the H1: "HISTORICAL DOCUMENT ... NOT a description of current behavior", with as-built trio links (README, docs/OPERATIONS.md, docs/EXAMPLES.md) in the repo's relative-link style and the Phase 9 note for docs/ARCHITECTURE.md. Banner commit `3b9569d` numstat: 6 added / 0 deleted; `git log 3b9569d..HEAD -- docs/IDEA.md` is empty, so the body (including the five D-13 conflict-journal lines) remains byte-untouched at 239 lines |
| 5 | Review committed at docs/reviews/2026-11-19-v1.0/ (REVIEW.md + 46 evidence files); local-only internal/ uncommitable from any clone; stale pre-move refs swept (plan 01) | ✓ VERIFIED | 48 tracked files under `docs/reviews/2026-11-19-v1.0/`: REVIEW.md + 46 evidence files + the wave-1 `go.mod` that stops `go tool` descending into the repro modules. `git check-ignore -v docs/reviews/2026-11-19-v1.0/internal/spec-trace.md` → `.gitignore:42` match; 0 tracked files under the guarded prefix; 0 of the four non-public filenames tracked anywhere; zip absent, old nested dir absent; `git grep '2026-11-19-v1.0-review'` outside `.planning/phases`/`.planning/milestones` → 0 |
| 6 | Registry structure per D-06/D-07/D-08: 167 rows in D-06 schema, 13 sections, six-value enum only, no empty cells, own-words summaries (plan 02) | ✓ VERIFIED | Row count 167; 17 prefixes, each count == max suffix (contiguity gate exit 0); exactly 13 `###` headings mirroring the trace's subsections; enum-membership gate → 0 violations; status census: 126 implemented + 12 superseded-by-amendment + 21 fixed-in-v1.1 + 4 consumer-zone + 2 accepted-deviation + 1 deferred-v2 + 1 fixed-in-v1.1(PERF-02) = 167 (the raw census tokens sum to 167 across 12 labels). Prefix totals sum to 167 (S-ALG-CU 11, S-ALG-DEL 3, S-ALG-RC 10, S-ALG-RD 1, S-ALG-REP 5, S-ALG-RT 5, S-ALG-ST 6, S-AVL 4, S-DM 9, S-FAIL 18, S-HLC 15, S-IMP 13, S-PAR 16, S-PERS 13, S-PROTO 20, S-SCN 4, S-SEM 14). Own-words substring audit over the local-only 167-row trace → 0 VERBATIM lines |
| 7 | Each ADR carries the explicit 14.x ↔ ADR ↔ informed-REQ mapping via lowercase-key frontmatter, slim MADR shape (plan 03) | ✓ VERIFIED | All 11 phase-01 ADRs have exactly the 5 keys (`adr`, `review-decision`, `informs`, `date`, `status`); zero options-analysis/alternatives headings; ADR 0002 `informs: []` with Out-of-Scope rationale (line 40); ADR 0008 Context resolves O-04 (lines 14,17,35); ADR 0011 states the 14.11 loss-share measurement "has no dedicated v1.1 requirement" (line 30); ADR 0005/0007 carry the verbatim reopenable line under the H1 |
| 8 | Cross-artifact consistency and repo quality gates green on the committed tree (plan 04) | ✓ VERIFIED | Every ADR number cited in registry Links cells (0001, 0004, 0007, 0008, 0012) resolves to an existing file — 0 unresolved; spec-identifying-string sweep (`confstor\|0031\|cyp-config\|cyp-perf`) over committed REVIEW.md + evidence/ → 0 hits; stale-path sweep → 0; `make lint` exit 0. `make test` full run on the current HEAD produced one failing later-phase test (`TestMain_socketBuffersLogged`, cmd/sup-kv, the XPORT-05 smoke from Phase 7) with "SIGTERM ... os: process already finished"; the test and the whole `cmd/sup-kv` package pass when run standalone. The phase-01 commit range (`378b27e..be46b39`) changed no module Go files (its 24 `.go` additions are review evidence under `docs/reviews/`, outside the `./...` build), so the docs-only phase's "Go suite untouched" claim holds; the flake is recorded below as a current-tree warning, not a phase-01 gap |

**Score:** 8/8 truths verified (0 present, behavior-unverified)

All eight truths are documentation-structure truths with deterministic machine evidence; none asserts a runtime state transition or a cancellation/cleanup/ordering invariant, so none is behavior-dependent. The strongest checks beyond presence: the registry's 167-row ID sequence and per-prefix contiguity against the local-only trace, the own-words substring audit, and registry-cited code paths opened and matched to their row summaries.

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | 21 registry rows still read `fixed-in-v1.1(<REQ>)` for requirements now marked Complete (XPORT-01/02/03/07, PROTO-03/04/08, OBSV-05/06), and `S-FAIL-15` awaits PERF-02 (Phase 8 in progress); the final flip to `implemented` is a later-phase duty | Phase 9 | 01-CONTEXT.md Deferred: "Flipping registry statuses ... Phases 3–7; final no unclosed rows check — Phase 9 (milestone exit criterion)"; ROADMAP Phase 9 goal "registry closed"; milestone exit criterion "the deviation registry has no unclosed rows" |

### Advisory (New Scope, Unevidenced)

None. Every current-tree observation in this regeneration carries deterministic evidence (commands and outputs above/below); nothing is reported on inference alone.

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `docs/SPEC-DEVIATIONS.md` | Deviation registry: preamble + 13 sections + 167 D-06 rows | ✓ VERIFIED | 261 lines; all gates green; tracked (commit `ffbe162`, later status flips in place) |
| `docs/adr/` (0001..0011) | 11 slim-MADR ADRs for 14.1..14.11 | ✓ VERIFIED | 11 phase-01 files, locked slugs, 5-key frontmatter each (a 12th, `0012-equal-version-tiebreak.md`, was added by a later phase and does not displace the set) |
| `docs/reviews/2026-11-19-v1.0/REVIEW.md` | Committed review record (§7/§13/§14 source) | ✓ VERIFIED | Tracked; §7 at :238, §13 at :354, §14 at :392 |
| `docs/reviews/2026-11-19-v1.0/evidence/` | 46-file evidence archive committed | ✓ VERIFIED | 46 tracked evidence files (plus REVIEW.md and the wave-1 `go.mod`) |
| `.gitignore` | Guard block matching `docs/reviews/2026-11-19-v1.0/internal/` | ✓ VERIFIED | Committed at `.gitignore:42`; `check-ignore -v` matches; 0 tracked paths under the prefix |
| `docs/IDEA.md` | Historical banner, body byte-untouched incl. five D-13 lines | ✓ VERIFIED | 6 added / 0 deleted at `3b9569d`; no later commits; banner anchors and trio links present |
| `.planning/PROJECT.md` | :66 parenthetical names IDEA.md historical intent-at-start | ✓ VERIFIED | Wording present at :66; "authoritative, self-contained design document" claim gone |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| `.gitignore` | `docs/reviews/2026-11-19-v1.0/internal/spec-trace.md` | check-ignore guard | ✓ WIRED | `.gitignore:42` match; 0 tracked paths under the guarded prefix; 0 non-public filenames tracked anywhere |
| `docs/IDEA.md` | `docs/OPERATIONS.md` (+README, EXAMPLES) | banner trio links | ✓ WIRED | Relative-link style present in the banner |
| `.planning/PROJECT.md` | `docs/IDEA.md` | :66 historical parenthetical | ✓ WIRED | "historical intent-at-start" + banner pointer + as-built trio |
| `docs/SPEC-DEVIATIONS.md` | `docs/reviews/2026-11-19-v1.0/internal/spec-trace.md` | scripted row/status derivation | ✓ WIRED | Local-only trace present with 167 rows; contiguity gate exit 0; own-words audit exit 0; row count unchanged since seeding |
| `docs/SPEC-DEVIATIONS.md` | `docs/adr/0001..0012` | Links cells cite ADR numbers | ✓ WIRED | Cited numbers {0001, 0004, 0007, 0008, 0012} all resolve; ADRs 0001/0004/0007/0008 back-cite their named rows |
| `docs/SPEC-DEVIATIONS.md` | `docs/reviews/2026-11-19-v1.0/REVIEW.md` | preamble + Links REVIEW refs | ✓ WIRED | Preamble links `reviews/2026-11-19-v1.0/REVIEW.md` (resolves from `docs/`); 55 REVIEW references across the registry |
| `docs/adr/0001` | REVIEW.md §14 row 1 | frontmatter + Links | ✓ WIRED | `review-decision` cites 14.1; Links name the four registry rows and the §14 source |
| `docs/adr/0005` | `.planning/REQUIREMENTS.md` | informs PROTO-01; D-11 wording | ✓ WIRED | D-11 line verbatim under the H1; wording mirrors the SPEC-02 acceptance parenthetical |
| `docs/adr/` | `docs/SPEC-DEVIATIONS.md` | bidirectional registry↔ADR citation | ✓ WIRED (with caveat) | Resolution loop over all cited ADR numbers → 0 unresolved. Caveat: ADR 0005's Links bullet over-claims four back-references (0 registry rows cite ADR 0005) — see WR-01 below; the registry→decision→code lookup the goal promises is unaffected |

### Data-Flow Trace (Level 4)

For a docs phase, "data" is the cited evidence resolving to something real.

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| Registry rows | Code path links | Live repo files | Sampled `internal/app/storage/store.go:209`, `internal/app/persist/snapshot.go:110` (`Restore`), `internal/app/repl/core.go:1510` (`handleDataSync`), `internal/grpcapi/gate.go:120` (`RunGate`) — files exist and content matches the row summaries | ✓ FLOWING |
| Registry statuses | Trace verdicts + REVIEW §7/§13/§14 | Local-only spec-trace.md (167 rows on disk) + committed REVIEW.md | 1:1 ID join; contiguity exit 0; own-words audit exit 0; statuses evolved in place without adding/removing rows | ✓ FLOWING |
| ADR Context/Decision | REVIEW §14 rows 1–11 | Committed REVIEW.md:392–408 | Each ADR's content matches its §14 row (0005 rule 6 and 0007 last-read-value verified by read) | ✓ FLOWING |
| S-PERS-06, S-SEM-03 Links | Code evidence | Trace's own `n/a` code column | Em-dash mirrors the source convention; status remains visible | ℹ️ INFO (per-convention, not hollow — the trace itself carries no code evidence for these two) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| SC-1 one-lookup | `grep -E '^\| S-(ALG-RT-02\|PAR-09\|PERS-07\|DM-04) '` | all four rows found with status and links | ✓ PASS |
| SC-2 §7 closures | per-family row extraction | all six families present and closed per D-08 | ✓ PASS |
| Registry completeness | row count / contiguity / empty-status / enum awk gates | 167; 0 GAP; 0 empty; 0 off-enum | ✓ PASS |
| Own-words audit | awk substring join trace↔registry | exit 0, no VERBATIM lines | ✓ PASS |
| ADR mapping | `review-decision:` census + file list | 11 unique values covering 14.1–14.11; 11 files tracked | ✓ PASS |
| D-11 placement | `grep -l 'reopenable at Phase 4/6 discussion before implementation'` | exactly 0005 and 0007 | ✓ PASS |
| Publication guard | `git check-ignore -v` + tracked-prefix count | `.gitignore:42` match; 0 tracked under prefix | ✓ PASS |
| Stale-path sweep | `git grep '2026-11-19-v1.0-review'` excluding phase archives | 0 files | ✓ PASS |
| Spec-identifying-string sweep | grep over REVIEW.md + evidence/ | 0 hits | ✓ PASS |
| Lint gate | `make lint` | exit 0 | ✓ PASS |
| Test gate (full suite) | `make test` | exit 1: `TestMain_socketBuffersLogged` in `cmd/sup-kv` — "os: process already finished" (later-phase test) | ⚠️ FLAKE (passes standalone: `go test -race -run TestMain_socketBuffersLogged ./cmd/sup-kv` → ok; whole package standalone → ok) |

### Probe Execution

No probes declared or conventional probes present (`find scripts -path '*/tests/probe-*.sh'` → 0; the phase files name none). Plan 04 introduces no scripts by design — SKIPPED (docs-only phase, no runnable probes).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| SPEC-01 | 01-01, 01-02, 01-04 | Registry maps every Spec statement ID to decision + code/ADR link; every §7 deviation closed; IDEA.md historical | ✓ SATISFIED | Truths 1, 2, 4, 5, 6 above; REQUIREMENTS.md Traceability marks SPEC-01 Phase 1 Complete |
| SPEC-02 | 01-03, 01-04 | Decisions 14.1–14.11 recorded as ADRs incl. the two accepted confirmations (reopenable at Phase 4/6) | ✓ SATISFIED | Truths 3, 7 above; D-11 line verbatim in ADRs 0005/0007 |

Orphaned requirements: none — REQUIREMENTS.md maps exactly SPEC-01 and SPEC-02 to Phase 1, and both are claimed by plans (01-01/01-02/01-04 and 01-03/01-04 respectively).

### Decision Coverage

All 13 trackable CONTEXT.md decisions (D-01..D-13) are honored by the shipped artifacts (manual verification: D-01/D-02/D-03/D-04 relocation + guard + ref sweep; D-05..D-08 registry schema/enum/closures; D-09..D-11 ADR conventions and loud lines; D-12 banner; D-13 body untouched). The automated gate `check decision-coverage-verify` returned `skipped: true` because it looked for an unprefixed `CONTEXT.md` in the phase directory (the file is `01-CONTEXT.md`) — no gap, the mapping was checked by hand against the artifacts.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| docs/adr/0005-interface-closed-until-full-key-list.md | 56–60 | Links bullet claims registry rows S-HLC-04, S-ALG-RC-05, S-ALG-ST-06, S-PROTO-18 cite "this ADR in its link column" — none of the 4 rows cites ADR 0005 (and the bullet's `fixed-in-v1.1(PROTO-01)` wording is now stale: those rows are `implemented`) | ⚠️ Warning | Factual over-claim in a convenience back-reference; the goal's lookup chain (registry row → REQ/REVIEW → code) is intact. Review finding WR-01, still open in `01-REVIEW-DISPOSITION.md` |
| docs/reviews/2026-11-19-v1.0/evidence/review-repl-api.md | ~12 places | Translated Spec references in committed evidence, vs REVIEW §0's evidence-clean claim | ⚠️ Warning | Publication-quality owner call; spec-identifying-string sweep still 0. Review finding WR-02, still open in the disposition ledger |
| docs/SPEC-DEVIATIONS.md | 21 rows | `fixed-in-v1.1(<REQ>)` retained for requirements REQUIREMENTS.md marks Complete | ⚠️ Warning | Registry status maintenance lag owned by Phases 3–7 and closed by Phase 9's registry-closed sweep (see Deferred Items); not a phase-01 execution defect |
| cmd/sup-kv/main_test.go | 1209–1238 | `TestMain_socketBuffersLogged` failed once in the full parallel `make test` run ("SIGTERM ... os: process already finished"); passes standalone | ⚠️ Warning | Later-phase (XPORT-05) smoke-test flake on the current HEAD; unrelated to this docs-only phase; no phase-01 module code exists to fix it |
| docs/SPEC-DEVIATIONS.md (row S-HLC-07) | — | `accepted-deviation(02-RESEARCH A2)` stretches the D-08 `<ADR\|14.x>` grammar; now also points at ADR 0012 | ℹ️ Info | Honest reference, documented in the plan 02 summary; enum shape and machine gate unaffected. Review finding IN-01, open |
| docs/SPEC-DEVIATIONS.md (rows S-PERS-06, S-SEM-03) | — | Em-dash Links cells | ℹ️ Info | Mirrors the trace's own `n/a` code evidence; per D-06 convention; status remains visible |
| docs/reviews/.../REVIEW.md, evidence/ (IN-02..IN-06) | — | Review-payload annotations (stale `.git/info/exclude` claim, uncommitted `*.out` in test-audit, repro-ID drift, §16 table gaps, ~20 unresolvable G-xx citations) | ℹ️ Info | Ledgered open, Phase 9 scope per the disposition ledger |

Debt-marker gate: `TBD`/`FIXME`/`XXX` → 0 in every authored phase file (registry, 11 ADRs, IDEA.md, .gitignore). The single `XXX` match in `docs/reviews/.../evidence/review-engine.md:861` is part of a verbatim shell command (`-run XXX`), not a debt marker. No blockers.

### Human Verification Required

N/A — documentation-only phase with no user-facing runtime elements. All four roadmap success criteria and all four plan truths are machine-verifiable and were re-verified programmatically (lookups, structural gates, guard, sweeps, lint). No truth is behavior-dependent. The three editorial-judgment items flagged during execution (banner loudness, ADR prose fidelity to §14, near-verbatim avoidance beyond the exact-substring audit) were dispositioned with rationale in the plan summaries and independently examined by the phase's code review (9 findings, 0 critical); the sampled registry summaries read as genuine own-words glosses. Residual judgment risk is carried by the open disposition ledger for owner triage.

### Gaps Summary

None blocking. The phase goal is achieved on the current tree: for any of the 167 Spec statement IDs, a reader finds status, decision and code/ADR link in one lookup (`docs/SPEC-DEVIATIONS.md`, proven 1:1 against the derivation trace with an unchanged row set); all six §7 deviation families are closed (S-12 via the preamble, exactly as D-08 specifies); eleven ADRs record 14.1–14.11 with the two loud accepted-by-default lines carrying the 14.5(6) Unavailable rule and the 14.7 last-read-value placement; IDEA.md is loudly historical over a byte-untouched body; and the publication guard makes the non-public Spec material uncommitable from any clone while nothing identifying has entered git. The post-phase evolution (126 rows now `implemented`, ADR 0012 added) is legitimate and did not disturb the phase-01 invariants.

Two items are explicitly not phase-01 gaps: (1) the 21 stale `fixed-in-v1.1` statuses and the PERF-02 row are deferred to Phase 9's registry-closed sweep per 01-CONTEXT.md's own deferred list and the milestone exit criterion; (2) the one full-suite test flake is a later-phase XPORT-05 smoke test that passes standalone, and Phase 1 changed no module Go code. The open review findings remain quality nits on phase artifacts, ledgered with fix suggestions, none touching the goal.

Plan 04's seven flagged probe assumptions (SPEC-01/SPEC-02 adjacency/empty/ordering/concurrency probes) remain honestly marked N/A-for-docs in the plan frontmatter and carry forward to the code phases that build data structures — recorded, not dropped.

---

_Verified: 2026-11-23T19:35:00Z_
_Verifier: Claude (gsd-verifier)_
