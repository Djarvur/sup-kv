---
adr: 0003
review-decision: "14.3 (Dump write policy stays as-is)"
informs: [PERF-01]
date: 2026-11-19
status: accepted
---

# 0003 — Dump policy unchanged: protect the disk

## Context

The review questioned the disk load the dump policy causes (§13 open
questions row 2): a full JSON dump on every change window costs disk I/O
proportional to the state size, and no measurement on target hardware
exists. Owner decision 14.3 settled it on 2026-11-19
(`docs/reviews/2026-11-19-v1.0/REVIEW.md` §14 row 3): the disk-protection
bias is intended.

## Decision

The dump algorithm and its defaults stay as they are — a full dump, biased
toward sparing the disk. `dump_debounce` and `dump_max_wait` remain
configurable; the defaults do not change.

## Consequences

Only documentation and profiles change, not the algorithm:

- the write-latency model is documented: under a sustained write flow,
  confirmation latency reaches `dump_max_wait` plus the dump time by design;
- a fast-response profile (tighter `dump_debounce`/`dump_max_wait`) is
  documented together with the disk cost it pays;
- benchmarks and the stand cover both profiles, and the "~10 ms" claim in
  .planning/PROJECT.md is tied to its measurement conditions.

PERF-01 carries the documentation work (Phase 8); the measurement side lands
with PERF-02..PERF-06.

## Links

- Source: docs/reviews/2026-11-19-v1.0/REVIEW.md §14 row 3
- Registry row (docs/SPEC-DEVIATIONS.md): S-PERS-04 (superseded-by-amendment
  — the trailing-quiet-window + `dump_max_wait` policy was adopted by the
  amendment per decision 14.3)
- Informs: PERF-01 (the write-latency model documented for both profiles)
- Out of Scope: "Changing the dump policy (full dump, debounce/max-wait
  defaults)" — .planning/REQUIREMENTS.md
