---
phase: 01-spec-reference-and-decisions
verified: 2026-11-19T16:27:38Z
status: passed
score: 8/8 must-haves verified
covered_files: [".gitignore", ".planning/phases/01-spec-reference-and-decisions/01-01-PLAN.md", ".planning/phases/01-spec-reference-and-decisions/01-01-SUMMARY.md", ".planning/phases/01-spec-reference-and-decisions/01-02-PLAN.md", ".planning/phases/01-spec-reference-and-decisions/01-02-SUMMARY.md", ".planning/phases/01-spec-reference-and-decisions/01-03-PLAN.md", ".planning/phases/01-spec-reference-and-decisions/01-03-SUMMARY.md", ".planning/phases/01-spec-reference-and-decisions/01-04-PLAN.md", ".planning/phases/01-spec-reference-and-decisions/01-04-SUMMARY.md", "docs/IDEA.md", "docs/SPEC-DEVIATIONS.md", "docs/adr/0001-snapshot-load-full-rollback.md", "docs/adr/0002-create-if-absent-blocked-by-tombstone.md", "docs/adr/0003-dump-policy-unchanged.md", "docs/adr/0004-no-conflict-journal.md", "docs/adr/0005-interface-closed-until-full-key-list.md", "docs/adr/0006-startup-gate-empty-wait.md", "docs/adr/0007-public-proto-consumer-zone.md", "docs/adr/0008-metrics-off-by-default.md", "docs/adr/0009-multi-node-support.md", "docs/adr/0010-darwin-development-only.md", "docs/adr/0011-value-limit-60kib-kept.md", "docs/reviews/2026-11-19-v1.0/REVIEW.md", "docs/reviews/2026-11-19-v1.0/evidence/docs-audit.md", "docs/reviews/2026-11-19-v1.0/evidence/repro/README.md", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/clock/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/persist/review_envelope_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/persist/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/repl/review_bench_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/repl/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/storage/review_bench_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/storage/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/metrics/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/persistfile/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/app/repl/review_repro_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/persist_probe_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/probe_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/socket_probe_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/synctail2_probe_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/synctail3_probe_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/synctail_probe_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/out/bench_darwin.txt", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/out/bench_linux.txt", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/out/mutations.jsonl", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/out/seed_sweep_1_600.txt", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/app/persist/zz_fuzz_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/app/persist/zz_probe_contain_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/app/repl/zz_probe3_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/app/repl/zz_probe4_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/udprepl/zz_fuzz_internal_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/udprepl/zz_probe_big_test.go", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/flaketally.py", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/mutate.py", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/mutate_b.py", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/mutlist.py", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/mutlist2.py", "docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/muttable.py", "docs/reviews/2026-11-19-v1.0/evidence/review-engine.md", "docs/reviews/2026-11-19-v1.0/evidence/review-repl-api.md", "docs/reviews/2026-11-19-v1.0/evidence/stand/RESULTS.md", "docs/reviews/2026-11-19-v1.0/evidence/stand/cpu-a-storm.pprof", "docs/reviews/2026-11-19-v1.0/evidence/stand/docker-compose.yml", "docs/reviews/2026-11-19-v1.0/evidence/stand/g.sh", "docs/reviews/2026-11-19-v1.0/evidence/stand/probe.sh", "docs/reviews/2026-11-19-v1.0/evidence/stand/seed.sh", "docs/reviews/2026-11-19-v1.0/evidence/stand/src-cmd/durcheck/main.go", "docs/reviews/2026-11-19-v1.0/evidence/stand/src-cmd/loadgen/main.go", "docs/reviews/2026-11-19-v1.0/evidence/stand/storm-monitor.log", "docs/reviews/2026-11-19-v1.0/evidence/test-audit.md"]
covered_digest: "v4:sha256:a6e1a1191aad6dfddd92d4bfee69099b35db9256f6d5ab0af36d46dc62f765c9"
behavior_unverified: 0
overrides_applied: 0
---

# Phase 1: Spec Reference and Decisions — Verification Report

**Phase Goal:** Every deviation between sup-kv and its Spec is visible and every settled owner decision is recorded — a reader can always answer "does sup-kv implement Spec statement X, and if not, who decided that and where is the code?"
**Verified:** 2026-11-19T16:27:38Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

Must-haves merged from ROADMAP.md success criteria (the contract, T1–T4) and non-duplicating PLAN frontmatter truths (T5–T8). No PLAN truth subtracts from the roadmap set.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A deviation registry exists mapping every Spec statement ID to a decision and a code/ADR link — any statement findable in one lookup (SC-1) | ✓ VERIFIED | `docs/SPEC-DEVIATIONS.md`: 167 rows, all with non-empty Summary/Status; Links cells cite code paths, ADR numbers or REVIEW refs on 165/167 rows (the 2 exceptions carry the D-06 em-dash convention and mirror the trace's own `n/a` code evidence). One-lookup spot-checks: `S-ALG-DEL-02`, `S-PERS-07`, `S-ALG-RT-02`, `S-PAR-09` each found in a single grep. Cited code paths sampled and confirmed live (`internal/app/storage/store.go:209` tombstone write; `internal/app/persist/engine.go:552` snapshot restore; `internal/grpcapi/server.go:171` Delete RPC) |
| 2 | Each review §7 deviation (S-01, S-02, S-06, S-07, S-08, S-12) closed by code-forward-reference or accepted-deviation — no row empty or TBD (SC-2) | ✓ VERIFIED | S-01 → S-PERS-12 `fixed-in-v1.1(APIS-07)` + ADR 0001 + REVIEW §7 ref; S-02 → 4 rows `fixed-in-v1.1(PROTO-08)`; S-06 → `OBSV-05`; S-07 → `OBSV-06`; S-08 → 12 rows across `PROTO-01/03/04`; S-11 → 4 × `consumer-zone` + ADR 0007 + S-IMP-04 `deferred-v2(SNAP-01)`; S-12 → registry preamble closure note (no statement ID, so preamble not row, per D-08). `grep TBD` → 0; empty-status gate → 0 |
| 3 | Eleven ADRs record decisions 14.1–14.11 incl. the two accepted-by-default confirmations of 2026-11-19 — 14.5(6) Unavailable rule and 14.7 last-read-value outside sup-kv (SC-3) | ✓ VERIFIED | 11 files at `docs/adr/0001..0011-*.md`; 11 unique `review-decision:` values covering 14.1–14.11 exactly once; D-11 line `accepted (by default; reopenable at Phase 4/6 discussion before implementation)` in exactly ADRs 0005 and 0007; ADR 0005 carries rule (6) "reading a key that is listed but whose value has not arrived yet answers `Unavailable`"; ADR 0007 records "the last-read-value file belongs to the consumer's client library ... not in sup-kv" |
| 4 | docs/IDEA.md marked historical — no reader mistakes the Spec for current behavior (SC-4) | ✓ VERIFIED | 6-line loud banner (blockquote+bold) above the H1: "HISTORICAL DOCUMENT ... NOT a description of current behavior", as-built trio links (README, docs/OPERATIONS.md, docs/EXAMPLES.md) in the repo's relative-link style, Phase 9 note for docs/ARCHITECTURE.md. Banner commit 3b9569d numstat: 6 added / 0 deleted (233 → 239 lines); the five D-13 conflict-journal lines byte-untouched |
| 5 | Review committed at docs/reviews/2026-11-19-v1.0/ (REVIEW.md + 46 evidence files); local-only internal/ uncommitable from any clone; stale pre-move refs swept (plan 01) | ✓ VERIFIED | 47 tracked files under `docs/reviews/2026-11-19-v1.0/` (1 REVIEW.md + 46 evidence); `git check-ignore -v` matches at `.gitignore:35`; 0 tracked files under the guarded prefix; 0 of the four non-public filenames tracked anywhere; zip gone; old nested dir gone; `git grep '2026-11-19-v1.0-review'` outside phase archives → 0; planning docs carry the new path (PROJECT.md ×2, ROADMAP.md ×2, REQUIREMENTS.md ×1) |
| 6 | Registry structure per D-06/D-07/D-08: 167 rows in D-06 schema, 13 sections, six-value enum only, no empty cells, own-words summaries (plan 02) | ✓ VERIFIED | Row count 167; per-prefix contiguity gate exit 0; exactly 13 `###` headings matching the trace's 13 subsections 1:1; ID sequence diff between the local-only trace and the registry is **empty** (order and membership 1:1 — the strongest derivation proof); enum-membership gate exit 0; status census 115 implemented + 12 superseded-by-amendment + 33 fixed-in-v1.1 + 4 consumer-zone + 2 accepted-deviation + 1 deferred-v2 = 167; substring own-words audit exit 0; all 12 amendment-removed rows match REVIEW §13's list exactly |
| 7 | Each ADR carries the explicit 14.x ↔ ADR ↔ informed-REQ mapping via lowercase-key frontmatter, slim MADR shape (plan 03) | ✓ VERIFIED | All 11 files have exactly 5 frontmatter keys (`adr`, `review-decision`, `informs`, `date`, `status`); zero options-analysis/alternatives headings; honest gaps recorded (ADR 0002 `informs: []` with Out-of-Scope rationale; ADR 0011 states the 14.11 loss-share measurement has no dedicated v1.1 requirement); ADR 0008 Context resolves O-04 via 14.8 |
| 8 | Cross-artifact consistency and repo quality gates green on the committed tree (plan 04) | ✓ VERIFIED | Every ADR number cited in registry Links cells (0001, 0004 ×2, 0007 ×4, 0008) resolves to an existing file — 0 unresolved; spec-identifying-string sweep (`confstor\|0031\|cyp-config\|cyp-perf`) over committed REVIEW.md + evidence/ → 0 hits; `make lint` exit 0; `make test` exit 0 (11+ packages ok under -race, zero FAIL lines); GUARD-OK re-asserted |

**Score:** 8/8 truths verified (0 present, behavior-unverified)

All eight truths are documentation-structure truths with deterministic machine evidence; none asserts a runtime state transition or cancellation/cleanup/ordering invariant, so none is behavior-dependent. The strongest check beyond presence: the trace→registry ID sequence diff is empty and cited code paths were opened and match their row summaries.

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts. (gate: `check.decision-coverage-verify` — total 13, honored 13, not_honored none)

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `docs/SPEC-DEVIATIONS.md` | Deviation registry: preamble + 13 sections + 167 D-06 rows | ✓ VERIFIED | 259 lines; all gates green; commit ffbe162 tracked |
| `docs/adr/` | 11 slim-MADR ADRs 0001..0011 | ✓ VERIFIED | 11 files, correct slugs, 5-key frontmatter each. (gsd-tools verify.artifacts reports "Not a file: path is a directory" for directory-path artifacts — checker limitation, content verified manually) |
| `docs/reviews/2026-11-19-v1.0/REVIEW.md` | Committed review record (§7/§13/§14 source) | ✓ VERIFIED | Tracked; §7 at :238-251, §13 at :354, §14 at :392-408 |
| `docs/reviews/2026-11-19-v1.0/evidence/` | 46-file evidence archive committed | ✓ VERIFIED | 46 tracked files (same directory-artifact tool note) |
| `.gitignore` | Guard block matching `docs/reviews/2026-11-19-v1.0/internal/` | ✓ VERIFIED | `.gitignore:35`; check-ignore -v match proven; local-only internal/ present on disk with 5 files, untracked |
| `docs/IDEA.md` | Historical banner, body byte-untouched incl. five D-13 lines | ✓ VERIFIED | 6 added / 0 deleted at 3b9569d; banner anchors all present |
| `.planning/PROJECT.md` | :66 parenthetical names IDEA.md historical intent-at-start | ✓ VERIFIED | Wording present; "authoritative, self-contained design document" claim gone |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| `.gitignore` | `docs/reviews/2026-11-19-v1.0/internal/spec-trace.md` | check-ignore guard | ✓ WIRED | `.gitignore:35` match; 0 tracked paths under the prefix |
| `docs/IDEA.md` | `docs/OPERATIONS.md` (+README, EXAMPLES) | banner trio links | ✓ WIRED | Relative-link style present in banner |
| `.planning/PROJECT.md` | `docs/IDEA.md` | :66 historical parenthetical | ✓ WIRED | "historical intent-at-start" + banner pointer |
| `docs/SPEC-DEVIATIONS.md` | `docs/reviews/.../internal/spec-trace.md` | scripted row/status derivation | ✓ WIRED (manual) | Tool false-negative on the anchored `^\| S-` regex (literal-substring search); manually proven: ID sequence diff trace↔registry is empty (167 = 167, order identical), contiguity gate exit 0 |
| `docs/SPEC-DEVIATIONS.md` | `docs/adr/0001..0011` | Links cells cite ADR numbers | ✓ WIRED | Cited numbers {0001, 0004, 0007, 0008} all resolve; ADRs 0001/0004/0007/0008 back-cite the exact rows |
| `docs/SPEC-DEVIATIONS.md` | `docs/reviews/.../REVIEW.md` | preamble + Links REVIEW refs | ✓ WIRED | Preamble links REVIEW.md; 57 REVIEW references in Links cells; relative link resolves from docs/ |
| `docs/adr/0001` | REVIEW.md §14 row 1 | frontmatter + Links | ✓ WIRED | review-decision cites 14.1; Links cite the four named registry rows |
| `docs/adr/0005` | `.planning/REQUIREMENTS.md` | informs PROTO-01; D-11 wording | ✓ WIRED | D-11 line verbatim; wording mirrors SPEC-02 acceptance parenthetical |
| `docs/adr/` | `docs/SPEC-DEVIATIONS.md` | bidirectional registry↔ADR citation | ✓ WIRED (manual) | Tool EISDIR on directory source; manually proven via the resolution loop (0 unresolved). Caveat: ADR 0005's Links bullet over-claims 4 back-references — see WR-01 below |

### Data-Flow Trace (Level 4)

For a docs phase, "data" is the cited evidence resolving to something real.

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| Registry rows | Code path links | Live repo files | Sampled `store.go:209`, `engine.go:552`, `server.go:171` — all exist and their content matches the row summaries | ✓ FLOWING |
| Registry statuses | Trace verdicts + REVIEW §7/§13/§14 | Local-only spec-trace.md (167 rows on disk) + committed REVIEW.md | 1:1 ID join; §13 amendment list matches the 12 superseded rows exactly | ✓ FLOWING |
| ADR Context/Decision | REVIEW §14 rows 1–11 | Committed REVIEW.md:396-406 | Each ADR's content matches its §14 row (0005 rule 6, 0007 last-read-value verified by read) | ✓ FLOWING |
| S-PERS-06, S-SEM-03 Links | Code evidence | Trace's own `n/a` code column | Em-dash mirrors source; status still visible | ℹ️ INFO (per-convention, not hollow — the trace itself has no code evidence for these two) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| SC-1 one-lookup | `grep -E '^\| S-(ALG-DEL-02\|PERS-07) '` | both rows found with status | ✓ PASS |
| SC-2 §7 closures | closure-token grep + per-row extraction | all six families closed per D-08 | ✓ PASS |
| Registry completeness | count/contiguity/empty/enum awk gates | 167; 0 GAP; 0 empty; 0 off-enum | ✓ PASS |
| Own-words audit | awk substring join trace↔registry | exit 0, no VERBATIM lines | ✓ PASS |
| ADR mapping | grep review-decision census | 11 unique values 14.1–14.11 | ✓ PASS |
| D-11 placement | `grep -l 'reopenable at Phase 4/6...'` | exactly 0005, 0007 | ✓ PASS |
| Publication guard | `git check-ignore -v` | `.gitignore:35` match | ✓ PASS |
| Repo quality gates | `make lint` / `make test` | exit 0 / exit 0, zero FAIL | ✓ PASS |

### Probe Execution

No probes declared or conventional probes present (`find scripts -path '*/tests/probe-*.sh'` → 0). Plan 04 introduces no scripts by design — SKIPPED (no runnable probes for a docs-only phase).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| SPEC-01 | 01-01, 01-02, 01-04 | Registry maps every Spec statement ID to decision + code/ADR link; every §7 deviation closed; IDEA.md historical | ✓ SATISFIED | Truths 1, 2, 4, 5, 6 above; REQUIREMENTS.md Traceability marks SPEC-01 Phase 1 Complete |
| SPEC-02 | 01-03, 01-04 | Decisions 14.1–14.11 recorded as ADRs incl. the two accepted confirmations (reopenable at Phase 4/6) | ✓ SATISFIED | Truths 3, 7 above; D-11 lines verbatim in ADRs 0005/0007 |

Orphaned requirements: none — REQUIREMENTS.md maps exactly SPEC-01 and SPEC-02 to Phase 1, and both are claimed by plans.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| docs/adr/0005-interface-closed-until-full-key-list.md | 56-58 | Links bullet claims registry rows S-HLC-04, S-ALG-RC-05, S-ALG-ST-06, S-PROTO-18 cite "this ADR in its link column" — none of the 4 rows does | ⚠️ Warning | Factual over-claim in a back-reference; the goal's lookup chain (registry row → PROTO-01 forward ref → REVIEW) is intact. Review finding WR-01, open in `01-REVIEW-DISPOSITION.md` with a precise fix |
| docs/reviews/2026-11-19-v1.0/evidence/review-repl-api.md | 713, 741, 1020 (+~9 more) | Translated Spec references in committed evidence, vs REVIEW §0's evidence-clean claim | ⚠️ Warning | Publication-quality owner call; spec-identifying-string sweep still 0. Review finding WR-02, open in the disposition ledger |
| docs/SPEC-DEVIATIONS.md (row S-HLC-07) | — | `accepted-deviation(02-RESEARCH A2)` stretches the D-08 `<ADR\|14.x>` grammar (references a research assumption, not an ADR/14.x) | ℹ️ Info | Honest reference, deliberately documented in plan 02 SUMMARY; enum shape and machine gate unaffected. Review finding IN-01, open |
| docs/SPEC-DEVIATIONS.md (rows S-PERS-06, S-SEM-03) | — | Em-dash Links cells | ℹ️ Info | Mirrors the trace's own `n/a` code evidence; per D-06 convention; status remains visible |
| docs/reviews/.../REVIEW.md, evidence/ (5 findings IN-02..IN-06) | — | Review-payload annotations (stale `.git/info/exclude` claim, uncommitted `*.out` in test-audit, repro-ID drift, §16 table gaps, ~20 unresolvable G-xx citations) | ℹ️ Info | Ledgered open, Phase 9 scope per disposition ledger |

Debt-marker gate: `grep -E 'TBD\|FIXME\|XXX\|TODO\|PLACEHOLDER'` over every phase-modified file → 0. No blockers.

### Review Findings Disposition (advisory, per orchestrator note)

The `01-REVIEW-DISPOSITION.md` ledger carries 9 open findings (0 critical): WR-01, WR-02, WR-03, IN-01..IN-06. Weighed during this verification: the two warning-level findings were independently confirmed accurate (WR-01: 0 of 4 claimed rows cite ADR 0005; WR-02: translated spec references present in review-repl-api.md). None fails a must-have truth: WR-01 is an over-claimed convenience back-reference (the registry→decision→code chain the goal promises works for every row), and WR-02 concerns annotation wording inside committed evidence whose identifying-string sweep is clean. They remain recorded and open for owner triage in the ledger — advisory, not blocking.

### Human Verification Required

N/A — documentation-only phase with no user-facing runtime elements. All four success criteria are machine-verifiable and were verified programmatically (lookups, gates, guard, lint/test). No truth is behavior-dependent (no state transitions or ordering invariants are asserted), so no PRESENT_BEHAVIOR_UNVERIFIED items exist. The three editorial-judgment items flagged during execution (banner loudness, ADR prose fidelity to §14, near-verbatim avoidance beyond the exact-substring audit) were dispositioned with rationale in the plan summaries and independently examined by the phase's code review (9 findings, 0 critical) — the registry summaries sampled in this verification read as genuine own-words glosses, structurally distinct from the normative sentences. Residual judgment risk is carried by the open disposition ledger for owner triage.

### Gaps Summary

None. The phase goal is achieved: for any of the 167 Spec statement IDs, a reader finds status, decision and code/ADR link in one lookup (`docs/SPEC-DEVIATIONS.md`, proven 1:1 against the derivation trace); all six §7 deviation families close per the D-08 mapping (S-12 via the preamble, exactly as D-08 specifies); eleven ADRs record 14.1–14.11 with the two loud accepted-by-default lines carrying the 14.5(6) Unavailable rule and the 14.7 last-read-value placement; IDEA.md is loudly historical over a byte-untouched body; and the publication guard makes the non-public Spec material uncommitable from any clone while nothing identifying has entered git. The open review findings are quality nits on phase artifacts, ledgered with fix suggestions, none touching the goal.

Plan 04's seven flagged probe assumptions (SPEC-01/SPEC-02 adjacency/empty/ordering/concurrency probes) are honestly marked N/A-for-docs in the plan frontmatter and carry forward to the code phases that build data structures — recorded, not dropped.

---

_Verified: 2026-11-19T16:27:38Z_
_Verifier: Claude (gsd-verifier)_
