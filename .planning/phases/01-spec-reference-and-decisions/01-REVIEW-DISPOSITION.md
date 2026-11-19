---
phase: 01-spec-reference-and-decisions
updated: 2026-11-19T16:30:00Z
total: 9
---

# Code Review Disposition — Phase 01

One row per finding from `01-REVIEW.md`. Edit the **Disposition** column by hand when triaging
(`open` / `fixed` / `skipped` / `deferred`); put the reason in **Source**. This ledger is rebuilt
from REVIEW.md on each review run — decisions recorded here are preserved.

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | open | ADR 0005 claims S-08-family registry rows cite it; the rows do not. Fix: add ADR 0005 to the Links cells of S-HLC-04, S-ALG-RC-05, S-ALG-ST-06, S-PROTO-18 — or reword the ADR claim. |
| WR-02 | warning | open | review-repl-api.md carries translated Spec quotes in ~12 places, contradicting REVIEW §0's evidence-clean claim. Owner call: tighten §0's wording (evidence is paraphrase-free except this file) or paraphrase the file. |
| WR-03 | warning | open | Repro test comments carry stale ENG/RPL IDs that disagree with the published reports. Fix target is Phase 9 (D-01, M-list) or a targeted comment sweep; tests are verbatim evidence, so annotate rather than rewrite. |
| IN-01 | info | open | S-HLC-07 status `accepted-deviation(02-RESEARCH A2)` stretches the registry enum's `<ADR|14.x>` grammar. Registry-internal wording fix (Phase 1 artifact). |
| IN-02 | info | open | REVIEW §0 names `.git/info/exclude` as the guard; the committed `.gitignore` line is the real guard. Review-payload annotation, Phase 9 scope. |
| IN-03 | info | open | test-audit appendix lists artifacts partly uncommitted (`*.out` ignored). Review-payload annotation, Phase 9 scope. |
| IN-04 | info | open | Engine repro test count drift (18 vs 21) and one assertion-less "test". Review-payload annotation, Phase 9 scope. |
| IN-05 | info | open | REVIEW §16 correspondence table omits R-10…R-16/A-02…A-06 with an R-15↔RPL-17 / R-16↔RPL-16 crossing. Phase 9 scope. |
| IN-06 | info | open | ~20 G-xx citations resolve in no committed document. Phase 9 scope (registry/anotation decision). |

**Legend:** `open` — seen, untriaged · `fixed` — resolved (record the commit) · `skipped` — will
not fix (say why) · `deferred` — postponed (name the phase; Phase 9 doc-fix sweep D-01 is the
natural home for the review-payload items above).
