# Phase 1: Spec Reference and Decisions - Context

**Gathered:** 2026-11-19
**Status:** Ready for planning

<domain>
## Phase Boundary

Docs-only phase — no production code changes. Delivers:

1. **Deviation registry** `docs/SPEC-DEVIATIONS.md` — one row per Spec statement ID (the 167 IDs assigned by the review's trace) mapping it to a decision and a code/ADR link; every section-7 deviation of the review (S-01, S-02, S-06, S-07, S-08, S-12) closed one of two ways: by code (forward reference to the fixing v1.1 requirement) or by an accepted-deviation entry — no row empty or TBD (SPEC-01, S-12).
2. **Eleven ADRs** — one per owner decision 14.1–14.11, including the two accepted-by-default confirmations 14.5(6) and 14.7 (SPEC-02).
3. **`docs/IDEA.md` marked historical** — loud banner, body kept.
4. **Review artifacts committed** — REVIEW.md + evidence/ land under `docs/reviews/2026-11-19-v1.0/`; the non-public spec material (`internal/`) never enters git.

</domain>

<decisions>
## Implementation Decisions

### Review artifact handling
- **D-01:** Commit REVIEW.md + evidence/ (docs-audit, review-engine, review-repl-api, test-audit, repro/, stand/) to the public repo; `internal/` (SPEC.md, SPEC-amended.md, SPEC-amendments.diff, spec-trace.md) **never** enters git — the upstream Spec is non-public and the repo is public. The registry may carry short own-words paraphrases of statements, no spec text quotes. — **Reversibility:** one-way — published material cannot be reliably retracted from a public repo (clones, forks, caches retain it); the owner accepted publishing the review's paraphrases, not the spec itself.
- **D-02:** Layout: flatten the nested duplicate dir to `docs/reviews/2026-11-19-v1.0/` (REVIEW.md, evidence/). Update every reference to the old path `docs/2026-11-19-v1.0-review/` in PROJECT.md, ROADMAP.md, REQUIREMENTS.md, AGENTS.md, `.claude/CLAUDE.md` wherever it appears. Reversible.
- **D-03:** Delete `docs/2026-11-19-v1.0-review.zip` — the extracted copy already exists locally. Reversible (re-extractable).
- **D-04:** Committed `.gitignore` entry for the `internal/` path under `docs/reviews/2026-11-19-v1.0/` so the non-public material can never be committed from any clone. Reversible.

### Deviation registry
- **D-05:** Single file `docs/SPEC-DEVIATIONS.md`: one table, rows grouped under spec-section headings (matching the review's trace sections). Reversible.
- **D-06:** Row = `ID | one-line own-words summary | status | links` (code path, ADR file, v1.1 requirement ID). Never quote spec text verbatim. Reversible.
- **D-07:** All statement IDs seeded in Phase 1, not progressively: statuses taken from the review trace's verdicts + REVIEW.md §7 + decisions 14.x. Registry is accurate on day one; later phases only flip statuses as fixes land. Reversible.
- **D-08:** Fixed status enum (machine-checkable; Phase 9 test greps for unclosed rows — DOCS-08): `implemented` / `fixed-in-v1.1(<REQ-ID>)` / `accepted-deviation(<ADR|14.x>)` / `consumer-zone` / `superseded-by-amendment` / `deferred-v2(<REQ-ID>)`. Section-7 rows close per §7's own proposals: S-01→fixed by APIS-07 (14.1), S-02→fixed by PROTO-08, S-06→fixed by OBSV-05, S-07→fixed by OBSV-06, S-08→fixed by PROTO-01/03/04 (+14.5 scope), S-11→consumer-zone (14.7), S-12→closed by this phase itself (registry exists, IDEA.md historical). Reversible (enum change = visible diff).

### ADR conventions
- **D-09:** `docs/adr/0001-<slug>.md` … `0011-<slug>.md`, sequential numbering, slim MADR shape: Context / Decision / Consequences / Links — no "considered options" ceremony, because review §14 already argued each choice. English (repo docs convention; the review stays the Russian source). Reversible.
- **D-10:** Each ADR carries an explicit mapping: Review decision `14.x` ↔ ADR number ↔ v1.1 requirement IDs it informs (frontmatter or Links section). An ADR index page is Phase 9 scope (D-10 in REQUIREMENTS); Phase 1 links ADRs from the registry rows instead. Reversible.
- **D-11:** ADRs for 14.5(6) (read of a listed-but-not-yet-arrived key → `Unavailable`) and 14.7 (last-read-value file lives outside sup-kv) carry a loud status line: "accepted (by default; reopenable at Phase 4/6 discussion before implementation)" — mirrors SPEC-02's acceptance wording; Phases 4/6 agents see it in the ADR itself. Reversible.

### IDEA.md historical marking
- **D-12:** Loud banner at the top of `docs/IDEA.md`, body untouched: states it is the historical intent-at-start document, not current behavior; points to the as-built trio README + docs/OPERATIONS.md + docs/EXAMPLES.md; notes that docs/ARCHITECTURE.md and the as-built description land by Phase 9. Reversible.
- **D-13:** The conflict-journal promises inside IDEA.md (five places) stay under the banner — their scrub belongs to Phase 9 (D-01, M-01…M-52); the `store.go` comment fix is Phase 7 territory (OBSV-03/14.4). No double-editing of the same lines in Phase 1. Reversible.

### Claude's Discretion
- Exact banner and ADR wording; the parenthetical in PROJECT.md calling IDEA.md "the authoritative design document" may be minimally adjusted to point at the historical banner (one-line edit, no re-litigation of the design itself).
- Registry heading grouping and column order within D-05/D-06; ADR slugs.
- Which of AGENTS.md / `.claude/CLAUDE.md` actually reference the old review path (planner verifies before touching).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Review and trace (source inputs for registry + ADRs)
- `docs/2026-11-19-v1.0-review/2026-11-19-v1.0-review/REVIEW.md` §7 (deviations S-01…S-12), §13 (Spec trace summary), §14 (owner decisions 14.1–14.11 — the ADR source text), §15 (milestone composition) — **this file moves to `docs/reviews/2026-11-19-v1.0/REVIEW.md` in this phase**
- `docs/2026-11-19-v1.0-review/2026-11-19-v1.0-review/internal/spec-trace.md` — the 167-row trace (ID, status, code/test evidence, deviation notes) — seed basis for the registry; **read locally, NEVER committed**
- `docs/2026-11-19-v1.0-review/2026-11-19-v1.0-review/internal/SPEC-amended.md` — the Spec as amended 2026-11-19; **read locally, NEVER committed**
- `docs/2026-11-19-v1.0-review/2026-11-19-v1.0-review/evidence/docs-audit.md` — M-01…M-52 list; cross-check so Phase 1 does not duplicate Phase 9 doc fixes

### Planning anchors
- `.planning/ROADMAP.md` — Phase 1 goal, success criteria (4 items), requirement IDs SPEC-01/SPEC-02
- `.planning/REQUIREMENTS.md` — SPEC-01, SPEC-02 acceptance wording (incl. the reopenable-at-Phase-4/6 clause)
- `.planning/PROJECT.md` — constraints (public repo, doc-is-test-input, PR-only), Key Decisions table
- `docs/IDEA.md` — target of the historical banner
- `AGENTS.md`, `.claude/CLAUDE.md` — check for old review-path references when moving the dir

</canonical_refs>

<code_context>
## Existing Code Insights

Docs-only phase: no production code is created or modified (`.gitignore` gains one guard line).

### Reusable Assets
- `spec-trace.md` table shape (ID / statement / status / code evidence / test evidence / notes) — direct template for the registry schema, minus the verbatim statement column (D-06)
- REVIEW.md §14 table (decision / meaning-for-work / acceptance) — maps one-to-one onto slim-MADR ADR sections: meaning→Context+Decision, acceptance→Consequences, IDs→Links

### Established Patterns
- Repo docs are English Markdown; README/EXAMPLES/OPERATIONS fenced blocks are CI-executed (doc-is-test-input) — registry/ADR files should not introduce fences that look executable unless intended
- Conventional commits (`docs(01): …`), PR-only flow, quality gates from the first commit — planning docs commit via `gsd_run query commit`

### Integration Points
- `.gitignore` — one added line guarding `docs/reviews/2026-11-19-v1.0/internal/`
- Reference updates: PROJECT.md, ROADMAP.md, REQUIREMENTS.md (+AGENTS.md/.claude/CLAUDE.md if they cite the old path)

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches.

</specifics>

<deferred>
## Deferred Ideas

- ADR index page (`docs/adr/README.md`) — Phase 9 (DOCS-10 lists "ADR index")
- Scrub of conflict-journal promises: IDEA.md five places + `.claude/CLAUDE.md` — Phase 9 (D-01, M-01…M-52); `store.go` comment — Phase 7 (OBSV-03)
- `docs/ARCHITECTURE.md` as-built description — Phase 9 (DOCS-08)
- Flipping registry statuses to `fixed-in-v1.1(<REQ>)` as fixes land — Phases 3–7; final "no unclosed rows" check — Phase 9 (milestone exit criterion)

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 1-Spec Reference and Decisions*
*Context gathered: 2026-11-19*
