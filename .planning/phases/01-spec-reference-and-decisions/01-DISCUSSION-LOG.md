# Phase 1: Spec Reference and Decisions - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-11-19
**Phase:** 1-Spec Reference and Decisions
**Areas discussed:** Review commits, Registry design, ADR conventions, IDEA.md historical

---

## Review commits

| Option | Description | Selected |
|--------|-------------|----------|
| Review, no internal | Commit REVIEW.md + evidence/ (audit, repro, stand); internal/ never; registry may carry short own-words paraphrases | ✓ |
| Registry+ADRs only | Review stays local-only; git gets only the registry + ADRs (own words, no review quotes) | |

| Option | Description | Selected |
|--------|-------------|----------|
| docs/reviews/ flatten | Flatten to docs/reviews/2026-11-19-v1.0/ (REVIEW.md, evidence/); update refs in PROJECT/ROADMAP/REQUIREMENTS | ✓ |
| Keep nested | Commit as-is with the awkward nested duplicate dir | |
| One-level flatten | docs/2026-11-19-v1.0-review/ one level (drop inner duplicate) | |

| Option | Description | Selected |
|--------|-------------|----------|
| Delete | Delete the .zip — extracted copy already on disk | ✓ |
| Keep untracked | Keep locally, never commit | |

| Option | Description | Selected |
|--------|-------------|----------|
| .gitignore entry | Committed .gitignore entry for the internal/ dir — protects future clones too | ✓ |
| Local exclude | .git/info/exclude only — guard stays invisible in the public repo | |
| No guard | No rule; rely on explicit adds only | |

**User's choice:** Review, no internal / docs/reviews/ flatten / Delete zip / .gitignore entry
**Notes:** Driver: the upstream Spec is non-public ("Internal — do not commit" marker in the review's internal/README.md) while the repo is public. Owner judged the review's own paraphrases publishable, the spec material not.

---

## Registry design

| Option | Description | Selected |
|--------|-------------|----------|
| ID+summary+status | ID + one-line own-words summary + status + links. Self-contained for a public reader; small per-row disclosure | ✓ |
| ID+status only | ID + status + links only, no statement text. Zero extra disclosure; rows opaque without REVIEW.md + private spec | |
| Full statement text | Reuse spec-trace's full statement-text column — publishes a statement-by-statement mirror of the spec | |

| Option | Description | Selected |
|--------|-------------|----------|
| Seed from trace | Seed every row from spec-trace verdicts + REVIEW.md section 7 + decisions 14.x — registry lands accurate on day one | ✓ |
| Skeleton first | Skeleton now (IDs + empty statuses), fill progressively in later phases | |

| Option | Description | Selected |
|--------|-------------|----------|
| Single file | docs/SPEC-DEVIATIONS.md — one file, one table, grouped by spec-section headings; easy for the Phase 9 test | ✓ |
| Per-section files | docs/spec/ directory, one file per spec section | |

| Option | Description | Selected |
|--------|-------------|----------|
| Fixed enum | implemented / fixed-in-v1.1(REQ) / accepted-deviation(ADR|14.x) / consumer-zone / superseded-by-amendment / deferred-v2(ID) — Phase 9 test greps for unclosed rows | ✓ |
| Free text | Free-text status column | |

**User's choice:** ID+summary+status / Seed from trace / Single file / Fixed enum
**Notes:** External-practice search (per research_before_questions) returned only heavyweight requirements-tracing tools (StrictDoc etc.) — rejected against the repo's minimal-dependency principle; the public-repo/private-spec leakage question had no external coverage and was decided by the owner.

---

## ADR conventions

| Option | Description | Selected |
|--------|-------------|----------|
| Slim MADR | MADR 4.0 slimmed: Context / Decision / Consequences / Links — no option-comparison ceremony since review section 14 already argued the choice | ✓ |
| Nygard | Nygard one-pager: Context / Decision / Status / Consequences — most concise | |
| Full MADR | Full MADR 4.0 with considered-options sections filled from the review's argumentation | |

| Option | Description | Selected |
|--------|-------------|----------|
| Sequential 0001-0011 | docs/adr/0001-snapshot-load-full-rollback.md … 0011; frontmatter maps each to Review decision 14.x and v1.1 requirement IDs | ✓ |
| Keyed to 14.x | File names keyed to review numbers: docs/adr/adr-14-01-*.md … adr-14-11-*.md | |

| Option | Description | Selected |
|--------|-------------|----------|
| English | English — matches every other doc in the repo; review stays the Russian source | ✓ |
| Russian | Russian — matches the review text the ADRs carry over | |
| Bilingual | English body + short Russian summary | |

| Option | Description | Selected |
|--------|-------------|----------|
| Loud in ADR | Status line: accepted (by default; reopenable at Phase 4/6 discussion before implementation) — mirrors SPEC-02 exactly | ✓ |
| Plain accepted | Plain accepted status; the reopen window lives only in REQUIREMENTS.md | |

**User's choice:** Slim MADR / Sequential 0001-0011 / English / Loud in ADR
**Notes:** Research informed the format choice: MADR 4.0 = structural standard, Nygard = concise (empirical comparison arXiv 2604.27333). Since §14 already carries the argumentation, the slim variant keeps traceability without ceremony.

---

## IDEA.md historical

| Option | Description | Selected |
|--------|-------------|----------|
| Banner + keep body | Loud banner at top (HISTORICAL — intent at start, not current behavior; as-built pointers), body untouched — history stays readable in place | ✓ |
| Replace body | Replace body with a short pointer; full text remains only in git history | |
| Delete file | Delete the file (git history preserves it) | |

| Option | Description | Selected |
|--------|-------------|----------|
| Leave under banner | Banner neutralizes them; actual scrub lands with Phase 9 (D-01, M-list); store.go comment is Phase 7 territory | ✓ |
| Scrub now | Scrub the conflict-journal mentions from IDEA.md now, in Phase 1 | |

| Option | Description | Selected |
|--------|-------------|----------|
| As-built trio | README + docs/OPERATIONS.md + docs/EXAMPLES.md as current entry points; note docs/ARCHITECTURE.md and the registry land by Phase 9 | ✓ |
| Registry only | Point to docs/SPEC-DEVIATIONS.md and the ADR index only | |

**User's choice:** Banner + keep body / Leave under banner / As-built trio
**Notes:** Conflict-journal promise scrub overlaps D-01/M-01…M-52 (Phase 9) and OBSV-03 (Phase 7) — no double-editing of the same lines.

---

## Claude's Discretion

- Exact banner/ADR wording; PROJECT.md "authoritative design document" parenthetical may get a one-line pointer adjustment
- Registry heading grouping, column order, ADR slugs
- Whether AGENTS.md / .claude/CLAUDE.md actually cite the old review path (planner verifies)

## Deferred Ideas

- ADR index page — Phase 9 (DOCS-10)
- Conflict-journal promise scrub — Phase 9 (D-01) / Phase 7 (OBSV-03)
- docs/ARCHITECTURE.md as-built description — Phase 9 (DOCS-08)
- Registry status flips as fixes land — Phases 3–7; final no-unclosed-rows check — Phase 9
