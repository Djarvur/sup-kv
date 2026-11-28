---
phase: 01-spec-reference-and-decisions
reviewed: 2026-11-19T16:14:35Z
depth: standard
files_reviewed: 61
files_reviewed_list:
  - docs/adr/0001-snapshot-load-full-rollback.md
  - docs/adr/0002-create-if-absent-blocked-by-tombstone.md
  - docs/adr/0003-dump-policy-unchanged.md
  - docs/adr/0004-no-conflict-journal.md
  - docs/adr/0005-interface-closed-until-full-key-list.md
  - docs/adr/0006-startup-gate-empty-wait.md
  - docs/adr/0007-public-proto-consumer-zone.md
  - docs/adr/0008-metrics-off-by-default.md
  - docs/adr/0009-multi-node-support.md
  - docs/adr/0010-darwin-development-only.md
  - docs/adr/0011-value-limit-60kib-kept.md
  - docs/IDEA.md
  - docs/reviews/2026-11-19-v1.0/evidence/docs-audit.md
  - docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/clock/review_repro_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/persist/review_envelope_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/persist/review_repro_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/repl/review_bench_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/repl/review_repro_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/storage/review_bench_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/storage/review_repro_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/metrics/review_repro_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/persistfile/review_repro_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/README.md
  - docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/app/repl/review_repro_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/persist_probe_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/probe_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/socket_probe_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/synctail2_probe_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/synctail3_probe_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/repl/internal/reviewprobe/synctail_probe_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/tests/out/bench_darwin.txt
  - docs/reviews/2026-11-19-v1.0/evidence/repro/tests/out/bench_linux.txt
  - docs/reviews/2026-11-19-v1.0/evidence/repro/tests/out/mutations.jsonl
  - docs/reviews/2026-11-19-v1.0/evidence/repro/tests/out/seed_sweep_1_600.txt
  - docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/app/persist/zz_fuzz_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/app/persist/zz_probe_contain_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/app/repl/zz_probe3_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/app/repl/zz_probe4_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/udprepl/zz_fuzz_internal_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/tests/probe/sup-kv/internal/udprepl/zz_probe_big_test.go
  - docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/flaketally.py
  - docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/mutate_b.py
  - docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/mutate.py
  - docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/mutlist2.py
  - docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/mutlist.py
  - docs/reviews/2026-11-19-v1.0/evidence/repro/tests/scripts/muttable.py
  - docs/reviews/2026-11-19-v1.0/evidence/review-engine.md
  - docs/reviews/2026-11-19-v1.0/evidence/review-repl-api.md
  - docs/reviews/2026-11-19-v1.0/evidence/stand/cpu-a-storm.pprof
  - docs/reviews/2026-11-19-v1.0/evidence/stand/docker-compose.yml
  - docs/reviews/2026-11-19-v1.0/evidence/stand/g.sh
  - docs/reviews/2026-11-19-v1.0/evidence/stand/probe.sh
  - docs/reviews/2026-11-19-v1.0/evidence/stand/RESULTS.md
  - docs/reviews/2026-11-19-v1.0/evidence/stand/seed.sh
  - docs/reviews/2026-11-19-v1.0/evidence/stand/src-cmd/durcheck/main.go
  - docs/reviews/2026-11-19-v1.0/evidence/stand/src-cmd/loadgen/main.go
  - docs/reviews/2026-11-19-v1.0/evidence/stand/storm-monitor.log
  - docs/reviews/2026-11-19-v1.0/evidence/test-audit.md
  - docs/reviews/2026-11-19-v1.0/REVIEW.md
  - docs/SPEC-DEVIATIONS.md
  - .gitignore
findings:
  critical: 0
  warning: 3
  info: 6
  total: 9
  necessity: 0
status: issues_found
---

# Phase 1: Code Review Report

**Reviewed:** 2026-11-19T16:14:35Z
**Depth:** standard
**Files Reviewed:** 61
**Status:** issues_found

## Summary

Documentation-only phase committing the 2026-11-19 v1.0 review payload, the spec-deviation
registry, 11 owner-decision ADRs, the IDEA.md historical banner and the .gitignore publication
guard. The payload's load-bearing numbers are internally consistent and reproduce from the
committed data: the registry has exactly 167 statement rows whose six statuses sum to 167, with
the 12 `superseded-by-amendment` rows matching REVIEW §13's amendment-removed list; every
`fixed-in-v1.1(...)`/`deferred-v2(...)` REQ ID resolves in `.planning/REQUIREMENTS.md`; the 11
ADRs map 1:1 to decisions 14.1–14.11 with semantically correct `informs:` targets, and all ADR
Out-of-Scope quotes resolve verbatim in REQUIREMENTS/PROJECT.md; the mutation tally recomputed
from the committed `mutations.jsonl` reproduces the claimed 57/54/16 exactly, as do the
benchmark outputs against REVIEW §4 and the storm-monitor total of 949 552 retransmissions;
spot-checked registry code links (store.go:83-89, clock.go:43-49, codec.go:25, core.go:751-759)
are accurate against the untouched Go source; the nested `go.mod` marker correctly stops
root-module descent into the intentionally-red repro tests; the `.gitignore` guard covers the
real non-public path and `git check-ignore` confirms it; no Cyrillic, no spec text and no
secrets leak anywhere outside REVIEW.md itself.

Three warnings: ADR 0005 states a registry linkage that the registry contradicts (WR-01);
`review-repl-api.md` quotes the non-public Spec in translated form under REVIEW §0's explicit
"no quotes in evidence/" claim, with a different (stricter) anonymization standard used by
`review-engine.md` (WR-02); and the committed repro tests carry ENG/RPL comment IDs from a stale
numbering that contradicts the published reports and REVIEW §16's correspondence table (WR-03).
Six info findings cover enum drift, stale mechanism claims, artefact-inventory mismatches, test
counts, an incomplete ID-correspondence table with a hidden crossing, and G-xx references that
resolve in no committed document.

Also examined (referenced by the phase note but outside the 61-file config list):
`docs/reviews/2026-11-19-v1.0/go.mod` (nested module marker) — found correct; no finding.

## Critical Issues

None.

## Warnings

### WR-01: ADR 0005 claims a registry linkage its own target rows do not contain

**File:** `docs/adr/0005-interface-closed-until-full-key-list.md:56-58`
**Issue:** The Links section states the S-08 family closes "as fixed-in-v1.1(PROTO-01) **with
this ADR in its link column**: S-HLC-04, S-ALG-RC-05, S-ALG-ST-06, S-PROTO-18". None of those
four rows in `docs/SPEC-DEVIATIONS.md` cites ADR 0005 — their Links columns read (S-HLC-04,
line 50) `internal/app/repl/core.go:631-644; REVIEW §7 S-08; G-03`; (S-ALG-RC-05, line 137)
the same; (S-ALG-ST-06, line 97) `internal/grpcapi/gate.go:122-145; internal/app/repl/core.go:632;
REVIEW §7 S-08; G-02, G-03`; (S-PROTO-18, line 84) `.../supkv.proto:66-74; REVIEW §7 S-08; G-03`.
The ADR's traceability claim is checkable and false for all four rows, which weakens exactly the
registry↔ADR↔REQ chain this phase exists to build. (Sibling ADRs 0001/0004/0007/0008 verify:
the rows they name do cite them.)
**Fix:** Either add `ADR 0005` to the Links column of S-HLC-04, S-ALG-RC-05, S-ALG-ST-06 and
S-PROTO-18 in `docs/SPEC-DEVIATIONS.md` (preferred — the decision behind PROTO-01 *is* ADR
0005, so the link is owed), or rewrite the ADR sentence to "closing as fixed-in-v1.1(PROTO-01);
see ADR 0005 for the accepted behavior" without claiming a link-column entry.

### WR-02: review-repl-api.md quotes the non-public Spec, contradicting REVIEW §0's publication-safety claim

**File:** `docs/reviews/2026-11-19-v1.0/evidence/review-repl-api.md:193-196` (and 127-128,
281-282, 328-330, 557-559, 611-612, 668-669, 717-719, 760-762, 852-853, 917)
**Issue:** REVIEW.md §0 (line 17) commits to: "В этом документе и в `evidence/` нет ссылок,
названий и цитат из неё" (no references, names or quotes from the non-public Spec). The repl/api
report breaks that contract: it quotes normative Spec sentences in English translation in
roughly a dozen places, e.g. §Startup "until the first full sync from the neighbour **has been
processed** or `startup_gate` elapsed" (193-194), §Replication "**`Ack` is not sent**"
(329-330), §Create and Update "if the key exists **and is not deleted**, `AlreadyExists` is
returned, otherwise an ordinary write is performed" (718-719), §Snapshots "for keys present in
the map and absent from the snapshot a delete is performed. Thanks to this the load is
replicated..." (760-762), §Dump "writes an error to the log and sets a metric" (852-853). The
sibling report `review-engine.md` follows a stricter discipline — it elides Spec text as
"[spec text]" placeholders — so the payload applies two different publication standards to the
same non-public source. The leakage is translation-level (no original Russian text anywhere in
`evidence/`, verified by a Cyrillic sweep), and the substance largely overlaps what ADRs 0001,
0002, 0004, 0005 already publish as own-words paraphrases — but the §0 claim as written is
false, and the quotes go beyond paraphrase.
**Fix:** Choose one standard and apply it. Cheapest: compress the review-repl-api.md quotes to
own-words glosses in the style of `docs/SPEC-DEVIATIONS.md` (keeping the SPEC §-section
citations), matching review-engine.md. Alternatively, if the translated quotes are judged
acceptable, rewrite REVIEW §0 to state the actual rule precisely: "own-words paraphrases and
bare line/section references only; no verbatim quotes and no original Spec text" — and note
that REVIEW.md itself names Spec sections (e.g. R-04 "третий исход раздела Replication Спеки
(L386–388)") which the current blanket sentence also fails to cover.

### WR-03: Repro test comments use stale ENG/RPL finding IDs that contradict the published reports

**File:** `docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/app/clock/review_repro_test.go:10`
**Issue:** The committed repro tests annotate each test with the finding ID it reproduces, but
the IDs follow an earlier draft numbering, not the published reports. Examples (comment →
actual report ID): clock drift test says "ENG-08", report ENG-08 is the dump-retry finding and
the drift finding is ENG-11; `TestReview_dumpCycleAtEnvelope` says "ENG-10", report ENG-10 is
Restore and the dump-cycle finding is ENG-12; `TestReview_dataSyncDoesNotAdvanceHLC` says
"ENG-06 (cross-boundary)", report ENG-06 is the `.prev` rotation and the DataSync/HLC finding is
ENG-03; `TestReview_equalVersionDivergenceAfterRestart` says "ENG-02" (report: ENG-07);
`TestReview_decodeGenerationFaults` and the bit-flip census say "ENG-04" (report: ENG-05);
`TestReview_failingDumpRetryCadence` says "ENG-07" (report: ENG-08);
`TestReview_sustainedLoadLatency` says "ENG-06" (report: ENG-09);
`TestReview_snapshotAfterFallbackCopiesCorruptMain` says "ENG-05" (report: ENG-06);
`TestReview_invalidUTF8KeyLostOnReload` says "ENG-15" (report: ENG-21);
`TestReview_restoreIgnoresDiscardedSnapshotRecords` says "ENG-09" (report: ENG-10);
`TestReview_orphanTempFilesNeverCollected` says "ENG-12" (report: ENG-17). The same drift
affects the repl side: in `evidence/repro/repl/internal/app/repl/review_repro_test.go` every
RPL comment uses the old numbers (its "RPL-01" is report RPL-04, its "RPL-02" is report RPL-03,
its "RPL-03"/"RPL-03b" are report RPL-02, its "RPL-04" is report RPL-09, its "RPL-06" is report
RPL-08, its "RPL-07" is report RPL-10, its "RPL-08" is report RPL-12, its "RPL-09" is report
RPL-11). REVIEW §16's correspondence table and the registry build on the *report* numbering, so
a Phase 3–6 fixer tracing a failing test by the ID in its comment lands on the wrong finding.
The storage file's two tests ("ENG-19") are correct — the inconsistency is internal to the
payload.
**Fix:** Update the comment IDs in the 10 affected engine files and the repl
`review_repro_test.go` to the published report numbering (a mechanical find/replace driven by
the mapping above), or strip the IDs and reference findings by test name from the reports.

## Info

### IN-01: S-HLC-07 status violates the registry's declared enum and overstates the decision level

**File:** `docs/SPEC-DEVIATIONS.md:53`
**Issue:** The preamble defines `accepted-deviation(<ADR|14.x>)` — the reference must name an
ADR or owner decision. S-HLC-07 instead carries `accepted-deviation(02-RESEARCH A2)`, pointing
at a v1.0-milestone research assumption (`.planning/milestones/v1.0-phases/02-engine-hlc-versioning-and-lww-store/02-RESEARCH.md:401`),
not an ADR or 14.x. The reference does resolve in-repo and the exception is consciously
documented (01-02-SUMMARY:124, STATE.md:74), so this is drift rather than a dangling link — but
REVIEW P-07 explicitly records "Решение не принималось" (no owner decision taken) for the
zero-filler question, so "accepted" overstates the decision level, and the row itself notes the
amendment left the random-filler requirement standing, i.e. the deviation is live and blessed
only by a research assumption.
**Fix:** Either record an actual owner decision (ADR or 14.x amendment) for the zero filler, or
reword the row to a status the enum covers and put the A2 provenance in the Links/summary
column with an explicit "not owner-decided" caveat.

### IN-02: REVIEW §0's guard-mechanism description is stale against the committed repo

**File:** `docs/reviews/2026-11-19-v1.0/REVIEW.md:17`
**Issue:** §0 states the internal Spec catalog "исключён из git через `.git/info/exclude`". The
committed repo's `.git/info/exclude` contains only `/.planning/milestone.lock`; the operative,
durable guard added by this phase is the tracked `.gitignore` entry
`docs/reviews/2026-11-19-v1.0/internal/` (verified via `git check-ignore`). Additionally §0/§16
locate the material only as "`internal/` рядом" / "Состав `internal/`", which read alone suggests
the repo-root `internal/` — the Go source tree — rather than the actual
`docs/reviews/2026-11-19-v1.0/internal/`. The protection itself is correct and now stronger
(travels with clones), but the committed review text describes a mechanism that does not exist
in this repo and a location that invites a root-level misread.
**Fix:** Amend §0 to name the `.gitignore` entry (and the full path
`docs/reviews/2026-11-19-v1.0/internal/`) as the guard, or add a phase addendum noting the
gitignore entry supersedes the local-only exclude.

### IN-03: test-audit.md appendix inventories evidence files that are not committed

**File:** `docs/reviews/2026-11-19-v1.0/evidence/test-audit.md:603-605`
**Issue:** The appendix says "All under `evidence/repro/tests/`" and lists ~15 artefacts —
`baseline_race.txt`, `verbose_race_p1.txt`, `linux_verbose_race_p1.txt`, `flaky_darwin.txt`,
`flaky_linux.txt`, `sweep_small.csv`, `sweep_nano.csv`, `cover_*.out`, `coverfunc_*.txt`,
`mutations_raw/`, `zz_probe_test.go.txt` — of which only 4 exist in the committed tree
(`bench_darwin.txt`, `bench_linux.txt`, `mutations.jsonl`, `seed_sweep_1_600.txt`). Part of the
gap is mechanical: the pre-existing `.gitignore` rule `*.out` (line 12) silently excluded the
coverage profiles at commit time. `scripts/` likewise commits 6 files while the appendix names
3 (`flaketally.py`, `mutate_b.py`, `muttable.py` are unnamed). The headline claims remain
verifiable from what did land — the 73-row `mutations.jsonl` reproduces the 57/54/16 scores and
the survivor list exactly — but a reader following the appendix hits missing files.
**Fix:** Rewrite the appendix to split "committed here" from "scratchpad-only", or commit the
remaining artefacts (renaming the `*.out` ones, e.g. to `.txt`, so the ignore rule does not
swallow them).

### IN-04: Engine repro test count drift, and one committed "test" has no assertion

**File:** `docs/reviews/2026-11-19-v1.0/evidence/repro/engine/internal/persistfile/review_repro_test.go:138`
**Issue:** REVIEW §2 says "(18 + 13 тестов и набор проб)" and §16 says "engine/ (18 тестов и
бенчмарки)"; the committed engine tree contains 21 `TestReview*` functions plus 7
`BenchmarkReview*` (the repl side's 13 is exact). Also
`TestReview_snapshotOfMissingMain` (persistfile, lines 138-144) only `t.Logf`s — it has no
assertion and can never fail, so the reports' "a repro fails while the defect is present"
framing does not hold for it (it is a behaviour-documentation probe, not a repro).
**Fix:** Correct the counts in REVIEW §2/§16 to 21 (+7 benchmarks), and either add an
assertion to `TestReview_snapshotOfMissingMain` (e.g. assert the `.snapshots` directory side
effect and the documented error/no-error outcome) or rename it to make its probe-only status
explicit.

### IN-05: REVIEW §16 correspondence table is partial and hides an ID crossing

**File:** `docs/reviews/2026-11-19-v1.0/REVIEW.md:450`
**Issue:** The ID-correspondence line maps R-01…R-09, P-01…P-07, S-01/S-02/S-06/S-07, A-01,
O-01/O-02/O-05 but omits R-10…R-16 and A-02…A-06 entirely. The omitted pairs are mostly
mechanical (R-10←RPL-09 … R-14←RPL-14), but two are crossed relative to numeric order —
R-15 (bind address/SO_REUSEADDR/jitter) ← RPL-17 and R-16 (receive-path garbage/CRC coverage)
← RPL-16 — so inferring by number misroutes. A-02…A-06 map to API-06, API-07, API-08, API-09
and docs-audit F10 respectively, none stated.
**Fix:** Extend the §16 correspondence line with the R-10…R-16 and A-02…A-06 pairs (explicitly
showing the R-15/R-16 crossing), so no reader has to guess.

### IN-06: G-xx references resolve in no committed document

**File:** `docs/SPEC-DEVIATIONS.md:26-27`
**Issue:** The registry preamble says "`REVIEW §7` and `G-xx` refer to the review's deviation
and gap findings", and G-01…G-13 are cited in ~20 registry Links columns (S-HLC-04, S-ALG-ST-06,
S-PERS-10, S-FAIL-09, S-FAIL-15, S-SEM-07, …) as well as in REVIEW §16. But REVIEW.md never
defines the G-xx IDs; per REVIEW §16 they live in `internal/spec-trace.md`, which is
deliberately not committed. A public-repo reader therefore cannot resolve any G-xx reference —
the preamble neither says where G-xx is defined nor that it is non-public. (SNAP-01, by
contrast, resolves: `.planning/REQUIREMENTS.md:192`.)
**Fix:** One sentence in the registry preamble: "G-xx IDs are defined in the non-public Spec
trace table (docs/reviews/2026-11-19-v1.0/internal/, not committed); they are kept for internal
cross-reference and cannot be resolved from the public repository."

---

_Necessity assessment: no runtime code changed in this phase, and the necessity rubric
(Impossible-state guard / already-defended-elsewhere / cargo-culted / phantom requirement /
generality-without-instantiation) does not apply to prose; `necessity: 0`._

_Verification notes: registry row/status arithmetic recomputed from the file (167 rows;
115 implemented + 33 fixed-in-v1.1 + 2 accepted-deviation + 4 consumer-zone + 12
superseded-by-amendment + 1 deferred-v2 = 167); mutation scores recomputed from
`mutations.jsonl` (54 killed-both, 3 Linux-only, 0 darwin-only, 16 survived = 57 Linux / 54
darwin); seed-sweep failure count (46 FAIL lines = 23 failures) and benchmark numbers matched
against REVIEW §4/§11 and `storm-monitor.log`; ADR frontmatter mapping, `informs:` targets,
Out-of-Scope quotes, IDEA.md banner links, and the exactly-five conflict-journal mentions
claimed by ADR 0004 all verified; `.gitignore` guard verified with `git check-ignore`; no
Cyrillic characters and no secret-shaped strings in the committed evidence._

_Reviewed: 2026-11-19T16:14:35Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
