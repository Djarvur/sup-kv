# Phase 1: Spec Reference and Decisions - Research

**Researched:** 2026-11-19
**Domain:** Repository documentation engineering — mechanical derivation of a 167-row spec-deviation registry from the v1.0 review artifacts, slim-MADR ADR authoring, review-artifact relocation with a publication guard
**Confidence:** HIGH (every claim verified against repo files this session; the domain is entirely local — no external libraries, packages, or services are involved)

## Summary

Phase 1 is a docs-only phase whose raw materials all exist locally and are internally consistent: the review at `docs/2026-11-19-v1.0-review/2026-11-19-v1.0-review/REVIEW.md` (uncommitted, untracked) and its `internal/spec-trace.md` (167 rows, locally verified: counts 116/22/18/5/6 match REVIEW.md §13 exactly; IDs contiguous per prefix with no gaps). The registry is therefore not an open design problem but a **mechanical derivation** with a closed ingredient list: seed statuses from the trace's five-value status vocabulary, apply three locked corrections (the 12+1 amendment-affected rows from §13, the five Missing rows, the six Out-of-scope rows), then close the six section-7 deviations using the §7→v1.1-requirement mapping that CONTEXT.md D-08 already fixes verbatim. The only genuine judgment left to the executor is per-row status assignment inside the fixed enum for ~10 rows where two closures are defensible; research provides a decision-priority rule (§14 decision > §7 proposal > trace note, amend-check against `SPEC-amended.md` first) and flags the specific rows.

The second risk area is **publication safety**, and it verified clean: REVIEW.md and `evidence/` contain no spec-identifying strings (grepped for `confstor|0031|cyp-config|cyp-perf` — clean), and the repo's CI doc tests read three fixed files only (`docs/OPERATIONS.md`, `docs/EXAMPLES.md`, README links) with no directory globbing, so new files (`docs/SPEC-DEVIATIONS.md`, `docs/adr/0001..0011-*.md`) cannot break CI or be executed as doc tests. The current `internal/` guard is **ineffective** — `.git/info/exclude` contains only `/.planning/milestone.lock`, and the pattern that `internal/README.md` claims (`/.planning/reviews/*/internal/`) does not match the actual path; D-04's committed `.gitignore` line is the real fix and must land in the same commit as the review artifacts. The old review path is referenced in exactly three repo files (`.planning/PROJECT.md` twice, `.planning/ROADMAP.md` once, `.planning/REQUIREMENTS.md` once) — AGENTS.md and `.claude/CLAUDE.md` have **zero** references (resolves the planner-verify item from CONTEXT.md discretion). Neither the review directory nor the zip is git-tracked, so the move is plain `mv`, the zip deletion is plain `rm`, and `git mv`/`git rm` would fail.

**Primary recommendation:** Build the registry by scripted extraction of the 167 trace rows (never hand-transcription), statuses transformed by the recipe in Architecture Patterns; commit REVIEW.md + evidence/ + `.gitignore` guard + registry + ADRs in one reviewed changeset with selective `git add` (never `git add docs/`), and verify with the five scripted checks in Validation Architecture before pushing.

<user_constraints>

## User Constraints (from CONTEXT.md)

Copied verbatim from `.planning/phases/01-spec-reference-and-decisions/01-CONTEXT.md` (gathered 2026-11-19, status Ready for planning). Research did not re-open any locked decision.

### Locked Decisions

**Review artifact handling**
- **D-01:** Commit REVIEW.md + evidence/ (docs-audit, review-engine, review-repl-api, test-audit, repro/, stand/) to the public repo; `internal/` (SPEC.md, SPEC-amended.md, SPEC-amendments.diff, spec-trace.md) **never** enters git — the upstream Spec is non-public and the repo is public. The registry may carry short own-words paraphrases of statements, no spec text quotes. — **Reversibility:** one-way — published material cannot be reliably retracted from a public repo (clones, forks, caches retain it); the owner accepted publishing the review's paraphrases, not the spec itself.
- **D-02:** Layout: flatten the nested duplicate dir to `docs/reviews/2026-11-19-v1.0/` (REVIEW.md, evidence/). Update every reference to the old path `docs/2026-11-19-v1.0-review/` in PROJECT.md, ROADMAP.md, REQUIREMENTS.md, AGENTS.md, `.claude/CLAUDE.md` wherever it appears. Reversible.
- **D-03:** Delete `docs/2026-11-19-v1.0-review.zip` — the extracted copy already exists locally. Reversible (re-extractable).
- **D-04:** Committed `.gitignore` entry for the `internal/` path under `docs/reviews/2026-11-19-v1.0/` so the non-public material can never be committed from any clone. Reversible.

**Deviation registry**
- **D-05:** Single file `docs/SPEC-DEVIATIONS.md`: one table, rows grouped under spec-section headings (matching the review's trace sections). Reversible.
- **D-06:** Row = `ID | one-line own-words summary | status | links` (code path, ADR file, v1.1 requirement ID). Never quote spec text verbatim. Reversible.
- **D-07:** All statement IDs seeded in Phase 1, not progressively: statuses taken from the review trace's verdicts + REVIEW.md §7 + decisions 14.x. Registry is accurate on day one; later phases only flip statuses as fixes land. Reversible.
- **D-08:** Fixed status enum (machine-checkable; Phase 9 test greps for unclosed rows — DOCS-08): `implemented` / `fixed-in-v1.1(<REQ-ID>)` / `accepted-deviation(<ADR|14.x>)` / `consumer-zone` / `superseded-by-amendment` / `deferred-v2(<REQ-ID>)`. Section-7 rows close per §7's own proposals: S-01→fixed by APIS-07 (14.1), S-02→fixed by PROTO-08, S-06→fixed by OBSV-05, S-07→fixed by OBSV-06, S-08→fixed by PROTO-01/03/04 (+14.5 scope), S-11→consumer-zone (14.7), S-12→closed by this phase itself (registry exists, IDEA.md historical). Reversible (enum change = visible diff).

**ADR conventions**
- **D-09:** `docs/adr/0001-<slug>.md` … `0011-<slug>.md`, sequential numbering, slim MADR shape: Context / Decision / Consequences / Links — no "considered options" ceremony, because review §14 already argued each choice. English (repo docs convention; the review stays the Russian source). Reversible.
- **D-10:** Each ADR carries an explicit mapping: Review decision `14.x` ↔ ADR number ↔ v1.1 requirement IDs it informs (frontmatter or Links section). An ADR index page is Phase 9 scope (D-10 in REQUIREMENTS); Phase 1 links ADRs from the registry rows instead. Reversible.
- **D-11:** ADRs for 14.5(6) (read of a listed-but-not-yet-arrived key → `Unavailable`) and 14.7 (last-read-value file lives outside sup-kv) carry a loud status line: "accepted (by default; reopenable at Phase 4/6 discussion before implementation)" — mirrors SPEC-02's acceptance wording; Phases 4/6 agents see it in the ADR itself. Reversible.

**IDEA.md historical marking**
- **D-12:** Loud banner at the top of `docs/IDEA.md`, body untouched: states it is the historical intent-at-start document, not current behavior; points to the as-built trio README + docs/OPERATIONS.md + docs/EXAMPLES.md; notes that docs/ARCHITECTURE.md and the as-built description land by Phase 9. Reversible.
- **D-13:** The conflict-journal promises inside IDEA.md (five places) stay under the banner — their scrub belongs to Phase 9 (D-01, M-01…M-52); the `store.go` comment fix is Phase 7 territory (OBSV-03/14.4). No double-editing of the same lines in Phase 1. Reversible.

### Claude's Discretion
- Exact banner and ADR wording; the parenthetical in PROJECT.md calling IDEA.md "the authoritative design document" may be minimally adjusted to point at the historical banner (one-line edit, no re-litigation of the design itself).
- Registry heading grouping and column order within D-05/D-06; ADR slugs.
- Which of AGENTS.md / `.claude/CLAUDE.md` actually reference the old review path (planner verifies before touching).

### Deferred Ideas (OUT OF SCOPE)
- ADR index page (`docs/adr/README.md`) — Phase 9 (DOCS-10 lists "ADR index")
- Scrub of conflict-journal promises: IDEA.md five places + `.claude/CLAUDE.md` — Phase 9 (D-01, M-01…M-52); `store.go` comment — Phase 7 (OBSV-03)
- `docs/ARCHITECTURE.md` as-built description — Phase 9 (DOCS-08)
- Flipping registry statuses to `fixed-in-v1.1(<REQ>)` as fixes land — Phases 3–7; final "no unclosed rows" check — Phase 9 (milestone exit criterion)

</user_constraints>

<phase_requirements>

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SPEC-01 | Repository carries a spec deviation registry mapping every Spec statement ID to a decision and a code/ADR link; every section-7 deviation closed by code or registry entry; `docs/IDEA.md` marked historical | Complete ingredient lists verified: 167 trace rows (counts + contiguity proven), trace status vocabulary, the 12+1 amendment rows, 5 Missing rows, 6 Out-of-scope rows, §7→closure map (D-08-locked), derivation recipe with scripted completeness checks (Architecture Patterns); IDEA.md banner target lines + PROJECT.md:66 parenthetical located; CI doc-test isolation verified |
| SPEC-02 | Each owner decision 14.1–14.11 recorded as an ADR, including 14.5(6) and 14.7 as accepted-by-default confirmations | REVIEW.md §14 table read in full (11 rows at REVIEW.md:392–407); ADR file scheme (D-09), mapping requirement (D-10), loud status line for ADRs 0005/0007 (D-11); full 14.x ↔ ADR ↔ v1.1-REQ mapping table with quoted REQUIREMENTS.md parentheticals (Architecture Patterns, Pattern 3) |

</phase_requirements>

## Architectural Responsibility Map

Docs-only phase; "tiers" are documentation artifacts instead of runtime layers.

| Capability | Primary Tier (artifact) | Secondary Tier | Rationale |
|------------|------------------------|----------------|-----------|
| Statement-level spec compliance status | `docs/SPEC-DEVIATIONS.md` (new, committed) | `spec-trace.md` (local-only seed) | The registry is the single public lookup; the trace stays local (D-01) and is only the derivation source |
| Owner decision record | `docs/adr/0001..0011-*.md` (new, committed) | REVIEW.md §14 (Russian source) | ADRs are the English permanent record; review §14 is the argumentation source (D-09) |
| Deviation ↔ decision ↔ code linkage | `docs/SPEC-DEVIATIONS.md` links column | ADR Links sections | D-06/D-10: registry rows carry code path + ADR + REQ ID; ADRs carry 14.x ↔ REQ mapping back-references |
| Review evidence preservation | `docs/reviews/2026-11-19-v1.0/{REVIEW.md, evidence/}` (committed) | zip (deleted) | D-02 flatten + commit; D-03 removes the redundant archive |
| Non-public material guard | `.gitignore` one line (committed) | — | D-04; only machine-enforceable guard that works in any clone |
| Reader orientation (historical vs as-built) | `docs/IDEA.md` banner | PROJECT.md:66 parenthetical | D-12/D-13: banner only, body untouched; one-line PROJECT.md adjustment (discretion) |
| Planning-doc path hygiene | `.planning/{PROJECT,ROADMAP,REQUIREMENTS}.md` | — | D-02 reference updates; exact lines verified (see Runtime State Inventory) |

## Standard Stack

No runtime packages, libraries, or frameworks are introduced. This phase installs nothing. The only "stack" is tooling already in use in this environment:

| Tool | Role | Note |
|------|------|------|
| git (repo) | move, selective staging, guard verification | Review dir + zip are untracked — plain `mv`/`rm`, not `git mv`/`git rm` [VERIFIED: `git status --porcelain docs/` shows exactly `?? docs/2026-11-19-v1.0-review.zip` and `?? docs/2026-11-19-v1.0-review/`] |
| grep/awk (stdlib shell) | mechanical registry extraction + completeness checks | 167-row seed is machine-extractable; no scripting language dependency needed |
| `gsd_run query commit` | planning-doc commits (`docs(01): …` conventional commits) | Established pattern [VERIFIED: CONTEXT.md code_context + git log `66aa0a0`, `4527d87`] |
| `make lint` / `make test` (`go tool golangci-lint run ./...`, `go test -race ./...`) | quality gates before push | Unaffected by docs changes but must stay green per AGENTS.md [VERIFIED: AGENTS.md project file; Makefile:11,16] |

**Package Legitimacy Audit:** Not applicable — this phase installs zero external packages (docs-only). No registry lookups required.

## Architecture Patterns

### System Architecture Diagram

Data flow from local-only sources to committed public artifacts:

```
LOCAL-ONLY (never committed)                PUBLIC (committed)
─────────────────────────────               ──────────────────────────────────────────
internal/spec-trace.md ─┐                   docs/SPEC-DEVIATIONS.md
  (167 rows: ID,        │                    (167 rows: ID, own-words summary,
   statement, status,   ├── derivation ─>    status from fixed enum, links)
   code/test evidence)  │   recipe              │ rows link to ──────────┐
internal/SPEC-amended.md┘   (below)           docs/adr/0001..0011-*.md <──────┘
  (amend-check source)                         (11 slim-MADR ADRs from §14)
                                                      ^
REVIEW.md (§7, §13, §14, §15) ── moved ──────> docs/reviews/2026-11-19-v1.0/REVIEW.md
evidence/ (5 files + repro/ + stand/) ─ moved > docs/reviews/2026-11-19-v1.0/evidence/
internal/ ────────────────────────── moved ──> docs/reviews/2026-11-19-v1.0/internal/
                                                (stays local; guarded by .gitignore line)

docs/2026-11-19-v1.0-review.zip ── deleted (D-03)
docs/IDEA.md ── banner prepended (D-12); body untouched incl. 5 conflict-journal lines (D-13)
.planning/{PROJECT,ROADMAP,REQUIREMENTS}.md ── old-path refs updated (D-02)
```

Primary use case trace: a reader opens `docs/SPEC-DEVIATIONS.md`, finds a statement ID row, reads its own-words summary and status, and follows the links column to the code path, the ADR, or the v1.1 requirement ID.

### Recommended Project Structure

```
docs/
├── SPEC-DEVIATIONS.md           # new — the 167-row registry (D-05/D-06)
├── adr/
│   ├── 0001-<slug>.md … 0011-<slug>.md   # new — 11 slim-MADR ADRs (D-09)
├── reviews/
│   └── 2026-11-19-v1.0/
│       ├── REVIEW.md            # moved (D-02)
│       ├── evidence/            # moved (D-01: docs-audit, review-engine, review-repl-api, test-audit, repro/, stand/)
│       └── internal/            # moved but NEVER committed (D-01/D-04; gitignored)
├── IDEA.md                      # banner prepended (D-12)
├── OPERATIONS.md                # untouched
└── EXAMPLES.md                  # untouched
.gitignore                       # +1 guard line (D-04)
```

### Pattern 1: Mechanical registry derivation (the core recipe)

**What:** Derive all 167 registry rows from `internal/spec-trace.md` + REVIEW.md §7/§13/§14 by script, not by hand.

**Step 1 — Extract.** Every line in spec-trace.md matching `^\| S-` is one registry row. Verified this session [VERIFIED: grep -c `^| S-` = 167; per-prefix contiguity check all CONTIGUOUS]:

| Prefix(es) | Rows | Prefix(es) | Rows |
|---|---|---|---|
| S-DM | 9 | S-PERS | 13 |
| S-HLC | 15 | S-PROTO | 20 |
| S-ALG-ST/CU/DEL/RC/RD/REP/RT | 6/11/3/10/1/5/5 (=41) | S-SCN | 4 |
| S-FAIL | 18 | S-SEM | 14 |
| S-AVL | 4 | S-PAR | 16 |
| S-IMP | 13 | **Total** | **167** |

Completeness invariant for the executor's extraction script: count per prefix == max number per prefix (no gaps) and total == 167.

**Step 2 — Group.** Rows grouped under the 13 spec-section headings matching the trace's subsections (D-05): Data model; Record versions; Exchange protocol; Algorithms — Startup; Algorithms — Read/Create/Update/Delete; Algorithms — Replication/Retransmission/Recovery; Exchange scenarios; Persistence; Storage semantics and client responses; Failure behavior (18 rows); Availability; Parameters; Impact. (English headings; repo docs are English.)

**Step 3 — Summarize.** One-line own-words summary per row, derived from the trace's statement column. **Never quote spec text verbatim (D-06).** The trace's statement column is a quotation of the non-public Spec — it is transcription source, not registry content. Translating to English forces paraphrase, which is the safest workflow. Code-path links may be reused verbatim from the trace's Code evidence column (file:line references are the review's own, already public in REVIEW.md §7's style). L-line references (e.g. `L511–512`) are safe: REVIEW.md, which is committed under D-01, uses the same convention [VERIFIED: REVIEW.md:244 `S-01 … (L511–512)`].

**Step 4 — Amend-check.** The trace was made against the pre-amendment Spec edition [VERIFIED: internal/README.md + REVIEW.md §13:377]. For every row whose trace status is **not** `Implemented`, consult `internal/SPEC-amended.md` / `SPEC-amendments.diff` (local, never committed): if the amendment removed or rewrote the requirement, the row's status is `superseded-by-amendment` (optionally linking ADR 0002/0003/0004 where a 14.x decision drove the amendment); if the amended Spec still requires the statement, the deviation is live and closes via `fixed-in-v1.1(<REQ>)` or `accepted-deviation(...)`.

**Step 5 — Close.** Apply the transform rules (table below) using REVIEW.md §7 proposals and §14 decisions, with the §7→REQ mapping D-08 locks.

**Step 6 — Assert.** Scripted completeness checks (see Validation Architecture): row count == 167; per-prefix contiguity; every status cell non-empty and drawn from the fixed enum; no `TBD` anywhere; each of S-01, S-02, S-06, S-07, S-08, S-11, S-12 appears in at least one row's closure.

**Trace status vocabulary** (the seed statuses; quoted verbatim from the trace's summary section) [VERIFIED: internal/spec-trace.md:38-53, counts confirmed by awk]:

`Implemented` (116) · `Implemented-with-deviation` (22) · `Partial` (18) · `Missing` (5) · `Out-of-scope-for-sup-kv` (6)

**Transform rules (trace status → registry enum):**

| Trace status | Registry status | Conditions |
|---|---|---|
| Implemented (116) | `implemented` | Default. Exceptions: the stand-downgraded trio (Open Question 1) and any row the amendment changed in substance (S-ALG-ST-06 → `fixed-in-v1.1(PROTO-01)`) |
| Implemented-with-deviation (22) | `superseded-by-amendment` | The 12 amendment-removed rows (list below); amendment legalized the as-built behavior |
| | `implemented` | Amendment made code compliant without removing the row (amend-check decides; candidates: S-DM-02, S-PERS-01 — JSON Lines now matches the amended dump description) |
| | `fixed-in-v1.1(<REQ>)` | Live deviation with a v1.1 fix (e.g. S-PROTO-14 → PROTO-08) |
| | `accepted-deviation(<ADR|14.x>)` | Owner accepted the deviation (e.g. S-HLC-07 zero-filler → Open Question 3) |
| Partial (18) | `fixed-in-v1.1(<REQ>)` | Via the §7 mapping table below, after amend-check |
| Missing (5) | `fixed-in-v1.1(<REQ>)` | S-HLC-04, S-ALG-RC-05 → PROTO-01 (14.5(4)); S-ALG-REP-04 → PROTO-03; S-PERS-12 → APIS-07 (14.1). S-SEM-07 → Open Question 2 |
| Out-of-scope-for-sup-kv (6) | `consumer-zone` | S-AVL-03, S-IMP-06, S-IMP-13, S-IMP-10 (S-11's scope per §7: "consumer zone; sup-kv side needs only public proto + gRPC code (14.7, A-06)") |
| | `deferred-v2(SNAP-01)` | S-IMP-04 (snapshot-proto mapping — REQUIREMENTS v2 list: "SNAP-01: Mapping snapshot operations onto the consumer's proto — open question #7, consumer zone") [VERIFIED: .planning/REQUIREMENTS.md:192] |
| | `superseded-by-amendment` | S-IMP-05 (in the 12-list) |

**Complete special-row lists (all verified by reading the trace this session):**

- **12 amendment-removed rows** [VERIFIED: REVIEW.md:377, verbatim ID list]: S-DM-02, S-PERS-01, S-PROTO-05, S-PROTO-11, S-PROTO-12, S-ALG-CU-10, S-PERS-04, S-PAR-15, S-SEM-07, S-FAIL-05, S-IMP-05, S-IMP-12. Plus: the latency requirement inside **S-FAIL-15** was removed (row itself remains — burst statement still live), and **S-ALG-ST-06** "changed in substance" — the amended Spec now requires the interface closed until the full key list arrives, which the code does not do → live deviation → `fixed-in-v1.1(PROTO-01)` [VERIFIED: REVIEW.md:377; trace S-ALG-ST-06 row].
- **5 Missing rows** [VERIFIED: spec-trace.md summary section]: S-HLC-04, S-ALG-RC-05 (same omission from two spec sections), S-ALG-REP-04, S-PERS-12, S-SEM-07.
- **6 Out-of-scope rows** [VERIFIED: awk over trace status column]: S-AVL-03 (client last-read-value file), S-IMP-04 (snapshot.proto mapping), S-IMP-05 (storage inside cyp-config-manager), S-IMP-06 (client on-disk cache), S-IMP-10 (fault injection), S-IMP-13 (config manager stage 3).

**§7 deviation → trace rows → registry closure** (closures locked by D-08; row membership verified from trace G-xx cross-references):

| §7 ID (REVIEW.md:238–251) | Trace rows affected | Registry closure (D-08 verbatim) | ADR link |
|---|---|---|---|
| S-01 (High, snapshot rollback) | S-PERS-12 | `fixed-in-v1.1(APIS-07)` | 0001 (14.1) |
| S-02 (High, every change replicates) | S-PROTO-14, S-ALG-CU-09, S-PERS-13, S-SEM-08 (G-05 rows) | `fixed-in-v1.1(PROTO-08)` | 0001 links |
| S-06 (Medium, drift warn log+metric) | S-HLC-13, S-FAIL-14 (G-07 rows) | `fixed-in-v1.1(OBSV-05)` | — |
| S-07 (Medium, four required log entries) | S-ALG-CU-08 (dump failure log), S-PERS-09 (start w/o usable dumps), S-PROTO-04 (foreign proto_version + own node_id logged — currently `slog.Debug`) | `fixed-in-v1.1(OBSV-06)` | — |
| S-08 (DataSync header, Update per part, third Replication outcome, first-seen-neighbor sync) | G-03 rows: S-HLC-04, S-ALG-RC-05, S-PROTO-18, S-ALG-ST-06 → `fixed-in-v1.1(PROTO-01)`; G-04 rows: S-PROTO-15, S-PROTO-16, S-ALG-CU-11, S-ALG-REP-04, S-SCN-03 → `fixed-in-v1.1(PROTO-03)`; G-02 rows: S-PROTO-10, S-SCN-04, S-FAIL-07 (S-ALG-ST-06 already covered) → `fixed-in-v1.1(PROTO-04)` | `fixed-in-v1.1(PROTO-01/03/04)` per row | 0005 (14.5) |
| S-11 (consumer zone) | S-AVL-03, S-IMP-04, S-IMP-06, S-IMP-10, S-IMP-13 | `consumer-zone` / `deferred-v2(SNAP-01)` per transform table | 0007 (14.7) |
| S-12 (High, process) | whole registry + IDEA.md banner | closed by this phase itself; note it in the registry preamble, not as a row (it has no Spec statement ID) | — |

### Pattern 2: Review artifact relocation with publication guard

**What:** Move untracked material to the D-02 layout and make `internal/` uncommitable.

**Verified mechanics** [VERIFIED this session]:
- Old layout is a **nested duplicate**: `docs/2026-11-19-v1.0-review/2026-11-19-v1.0-review/{REVIEW.md, evidence/, internal/}`.
- Git state: both `docs/2026-11-19-v1.0-review/` and `docs/2026-11-19-v1.0-review.zip` are untracked (`??` in porcelain). So: plain `mv` for the dir, plain `rm` for the zip. `git mv`/`git rm` **fail** on untracked paths.
- Current `.gitignore` has no `docs/` entries at all (last block verbatim: `# Local environment` / `.env`); the new line is purely additive.
- The existing guard is ineffective: `.git/info/exclude` contains only `/.planning/milestone.lock`. The pattern `internal/README.md` claims (`.git/info/exclude`, `/.planning/reviews/*/internal/`) is **not present** and would not match the path anyway. Do not rely on it; D-04's committed line is the only real guard.

**Recommended sequence:**
1. `mkdir -p docs/reviews/2026-11-19-v1.0` and `mv docs/2026-11-19-v1.0-review/2026-11-19-v1.0-review/* docs/reviews/2026-11-19-v1.0/` (flatten — do not reproduce the nesting), then remove the empty old dirs; `rm docs/2026-11-19-v1.0-review.zip`.
2. Add the guard line `docs/reviews/2026-11-19-v1.0/internal/` to `.gitignore`.
3. Stage selectively: `git add .gitignore docs/reviews/2026-11-19-v1.0/REVIEW.md docs/reviews/2026-11-19-v1.0/evidence docs/SPEC-DEVIATIONS.md docs/adr docs/IDEA.md .planning/…` — **never `git add docs/` or `git add -A`** (would stage internal/ if the guard line were mistyped).
4. Verify: `git status --porcelain docs/` lists no `internal/` path; `git check-ignore -v docs/reviews/2026-11-19-v1.0/internal/spec-trace.md` reports a match.

Payload note: committable material is small — REVIEW.md 104K + evidence/ 760K [VERIFIED: du]. No size concerns.

### Pattern 3: ADR authoring from REVIEW.md §14

**What:** 11 slim-MADR files, one per §14 row, English, no options-analysis ceremony.

**Verified inputs:** §14 table spans REVIEW.md:392–407, 11 rows, each with (question / decision / meaning-for-work); §14 closes with "Спека приведена в соответствие с этими решениями 2026-11-19" [VERIFIED: REVIEW.md:408]. Each ADR maps: review `14.x` ↔ ADR number ↔ v1.1 REQ IDs (D-10). Slim-MADR sections exactly: Context / Decision / Consequences / Links (D-09). ADRs 0005 and 0007 carry the loud line "accepted (by default; reopenable at Phase 4/6 discussion before implementation)" (D-11 verbatim wording).

**Proposed mapping and slugs** (slugs are CONTEXT discretion — proposals, planner may rename; the number↔14.x mapping is the substance):

| ADR | 14.x | Proposed slug | v1.1 REQ IDs it informs (evidence: REQUIREMENTS.md parentheticals, quoted) |
|---|---|---|---|
| 0001 | 14.1 | `snapshot-load-full-rollback` | APIS-07 ("S-01, 14.1", :138) |
| 0002 | 14.2 | `create-if-absent-blocked-by-tombstone` | none (Out of Scope: "Owner decision 14.2: deleted key blocks with `AlreadyExists`; no code change", :201) |
| 0003 | 14.3 | `dump-policy-unchanged` | PERF-01 ("PERF-01, PERF-02, 14.3", :162); Out of Scope :202 |
| 0004 | 14.4 | `no-conflict-journal` | OBSV-03 ("O-03, 14.4", :147); Out of Scope :199 |
| 0005 | 14.5 | `interface-closed-until-full-key-list` | PROTO-01 ("R-03, 14.5", :80) — **carries D-11 status line** |
| 0006 | 14.6 | `startup-gate-empty-wait` | PROTO-02 ("14.6", :82) |
| 0007 | 14.7 | `public-proto-consumer-zone` | APIS-06 ("A-06, 14.7", :136); Out of Scope :203 — **carries D-11 status line** |
| 0008 | 14.8 | `metrics-off-by-default` | DOCS-10 ("(14.3, 14.8)", :182 — deployment-guide metrics setup); O-04 resolved by 14.8 [VERIFIED: REVIEW.md:271, :403] |
| 0009 | 14.9 | `multi-node-support` | PROTO-09 ("14.9", :96); TEST-01 stabilization context |
| 0010 | 14.10 | `darwin-development-only` | DOCS-10 ("14.10", :182); XPORT-05 ("14.10", :62) |
| 0011 | 14.11 | `value-limit-60kib-kept` | PERSIST-01 (single-source limit, :101); note: the 14.11 consequence "measure loss share on a real link" has **no dedicated v1.1 REQ** — record it honestly in Consequences rather than inventing a mapping |

### Pattern 4: IDEA.md historical banner

Banner prepended above the existing title; body byte-untouched. The five conflict-journal promise lines stay (D-13) — exact locations per the docs audit: `docs/IDEA.md:139`, `:163-164`, `:168-169`, `:188-190`, `:213` [VERIFIED: evidence/docs-audit.md:85, row M-23]. Banner points to README + docs/OPERATIONS.md + docs/EXAMPLES.md and notes docs/ARCHITECTURE.md lands by Phase 9 (D-12). CI safety verified: no test reads IDEA.md [VERIFIED: docs-audit.md:344 "Edit IDEA.md … No test reads them"; doc tests target fixed paths — cmd/sup-kv/operations_test.go:22 `operationsDocPath = "../../docs/OPERATIONS.md"`, examples_test.go:47 `examplesDocPath = "../../docs/EXAMPLES.md"`, readme_test.go:109 link check for `docs/OPERATIONS.md`+`docs/EXAMPLES.md`; no Glob/ReadDir/WalkDir over docs/ in any test]. The PROJECT.md:66 parenthetical ("the authoritative, self-contained design document") gets the minimal one-line adjustment (discretion).

### Anti-Patterns to Avoid

- **Hand-transcribing 167 rows.** Guarantees missed rows and invented statuses. Extract by script; eyeball only the summaries.
- **`git add docs/` (or `-A`) for the review commit.** One mistyped `.gitignore` line away from publishing the Spec. Stage explicit paths; check porcelain output before committing.
- **Reproducing the nested directory at the new location.** D-02 says flatten to `docs/reviews/2026-11-19-v1.0/` — one level, no duplicate dirname.
- **Quoting spec statements (or near-verbatim translations) as registry summaries.** D-06 violation with one-way consequences (D-01). Summarize in own words; reuse only IDs, L-refs, code paths, statuses.
- **Editing the five IDEA.md conflict-journal lines or the `store.go` comment.** D-13 reserves them for Phases 9/7 — double-editing creates merge conflicts for later phases.
- **Trusting `internal/README.md`'s exclusion claim.** Stale — the claimed `.git/info/exclude` pattern is absent [VERIFIED: exclude file read this session].

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 167-row extraction + status census | Manual transcription in an editor | grep/awk over `internal/spec-trace.md` (`grep -c '^\| S-'` = 167; per-prefix count==max contiguity check) | The seed is already a machine-parseable table; manual work is where silent row loss happens |
| Completeness verification | "I checked it visually" | Scripted asserts: total count, prefix contiguity, enum membership per row, no empty/TBD cells, §7-ID coverage | D-08 exists precisely so Phase 9 can grep; Phase 1 should pass the same class of check from day one |
| Publication guard | Local-only exclusion (.git/info/exclude, pre-commit hooks) | The committed `.gitignore` line (D-04) + selective `git add` | Local excludes don't survive clones — D-04's whole point; verified the current exclude doesn't even match |
| Row status decisions | Re-deriving compliance per row from scratch | The trace verdict + amend-check + §7/§14 mapping (recipe steps 4–5) | The review already did the code-level judgment; re-deriving risks contradicting settled decisions |

**Key insight:** Everything in this phase is derivation from artifacts that already exist and are mutually consistent (trace counts == §13 counts, verified). The phase's difficulty is not knowledge — it is completeness discipline and one-way publication safety.

## Runtime State Inventory

Path-move phase (old review path → new layout), so the canonical question applies: *after every file moves, what still references the old path?*

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — no databases or datastores in this phase. Verified: docs-only. | none |
| Live service config | None — no external services involved. | none |
| OS-registered state | None — no OS registrations. | none |
| Secrets/env vars | None — no secrets touched. | none |
| Untracked filesystem artifacts | `docs/2026-11-19-v1.0-review/` (nested dir, incl. `internal/` 300K) and `docs/2026-11-19-v1.0-review.zip` (371,985 bytes) — both `??` untracked [VERIFIED: git status --porcelain] | Plain `mv` + plain `rm` (not `git mv`/`git rm`) |
| Stale text references to old path | Exactly: `.planning/PROJECT.md:15` and `:137`; `.planning/ROADMAP.md:5`; `.planning/REQUIREMENTS.md:6` [VERIFIED: repo-wide grep excluding .git/caches/dist]. **AGENTS.md and `.claude/CLAUDE.md: zero references** (resolves the CONTEXT discretion item). `.planning/phases/01-…/01-CONTEXT.md` lines 23–24, 55–58 also mention the old path — see note below | Update the three planning files (D-02); leave phase CONTEXT.md untouched (see Open Question 4) |
| Ineffective guard state | `.git/info/exclude` contains only `/.planning/milestone.lock`; the pattern `internal/README.md` claims is absent and non-matching [VERIFIED: file read] | Replaced by the committed `.gitignore` line (D-04) |
| Build artifacts | None affected — no compiled artifacts reference docs paths | none |

**Note on 01-CONTEXT.md:** its own decision text (D-02/D-03) and canonical-refs section cite the pre-move path. That file records decisions as gathered — do not rewrite it; PLAN.md and this RESEARCH.md carry the post-move paths, and downstream agents read the moved location.

## Common Pitfalls

### Pitfall 1: Publishing non-public Spec material (D-01 breach — one-way)
**What goes wrong:** `internal/` (SPEC.md, SPEC-amended.md, SPEC-amendments.diff, spec-trace.md — 300K of spec-derived text including verbatim quotations) lands in a public commit.
**Why it happens:** `git add docs/` / `git add -A`; a mistyped or missing `.gitignore` line; assuming the old `.git/info/exclude` still guards (it does not).
**How to avoid:** Guard line first, selective `git add` of explicit paths, `git status --porcelain` reviewed before every commit, `git check-ignore -v` asserted. Published material cannot be retracted (clones/forks/caches) — this is the phase's only irreversible action.
**Warning signs:** porcelain output containing any `internal/` path; `git check-ignore` non-match.

### Pitfall 2: Spec text leaking into registry summaries (D-06 breach)
**What goes wrong:** Near-verbatim (or close-translation) copies of statement text appear in `docs/SPEC-DEVIATIONS.md`.
**Why it happens:** The trace's statement column is right there; copy-paste is faster than paraphrase; translation can be deceptively close to the original normative sentence.
**How to avoid:** Summaries written as one-line own-words glosses; L-refs, IDs, statuses, code paths are fine to carry (REVIEW.md precedent). A cheap audit: no registry summary cell should be a substring-match of its trace statement cell.
**Warning signs:** summary cells longer than ~1 line; exact punctuation/patterns mirroring the trace.

### Pitfall 3: Inventing statuses outside the enum
**What goes wrong:** Rows get statuses like "partially implemented", "won't fix", "N/A" — breaking D-08's machine-checkability and the Phase 9 grep.
**Why it happens:** The enum's six values feel incomplete for edge rows (they are — that's what the links column + own-words summary are for).
**How to avoid:** Enum membership asserted per row by script; nuance lives in the summary/links, never in a new status value.
**Warning signs:** any status cell not matching one of the six enum patterns.

### Pitfall 4: Missing a stale old-path reference
**What goes wrong:** Reader follows `.planning/PROJECT.md` footnote to a nonexistent path.
**Why it happens:** Header references are obvious; the PROJECT.md **footer** (`:137`) and ROADMAP milestone-paragraph (`:5`) are easy to miss.
**How to avoid:** Post-edit grep for `2026-11-19-v1.0-review` across the repo (expect zero hits outside phase-archival docs).
**Warning signs:** grep hits after the "reference update" task claims completion.

### Pitfall 5: Trusting the trace's pre-amendment verdicts blindly
**What goes wrong:** A row is marked `implemented` (or left `Implemented-with-deviation`) when the 2026-11-19 amendment changed the requirement in substance — S-ALG-ST-06 is the confirmed example (amended Spec now demands the closed interface the code lacks → must be `fixed-in-v1.1(PROTO-01)`).
**Why it happens:** Trace statuses were awarded against the pre-amendment edition [VERIFIED: REVIEW.md:377].
**How to avoid:** Recipe step 4 (amend-check every non-`Implemented` row against SPEC-amended.md) — plus the 13 named rows above.
**Warning signs:** a §7-listed deviation row showing `implemented` in the draft registry.

### Pitfall 6: Double-editing deferred lines
**What goes wrong:** Scrubbing IDEA.md conflict-journal lines (or `.claude/CLAUDE.md:21`, or `store.go:63-64` comment) while banner-ing.
**Why it happens:** M-23 makes them look like phase-1 work.
**How to avoid:** D-13 is explicit: banner only; scrub is Phase 9 (IDEA/CLAUDE.md) and Phase 7 (store.go).
**Warning signs:** diffs touching IDEA.md lines 139/163-164/168-169/188-190/213.

## Code Examples

### Registry row shape and enum (D-05/D-06/D-08, verbatim constraints)

```markdown
<!-- docs/SPEC-DEVIATIONS.md — row = ID | one-line own-words summary | status | links -->
<!-- Status enum (D-08, fixed, machine-checkable):
     implemented
     fixed-in-v1.1(<REQ-ID>)
     accepted-deviation(<ADR|14.x>)
     consumer-zone
     superseded-by-amendment
     deferred-v2(<REQ-ID>) -->

| ID | Summary (own words) | Status | Links |
|---|---|---|---|
| S-PERS-12 | Snapshot load leaves map keys that the snapshot lacks untouched, instead of deleting them | fixed-in-v1.1(APIS-07) | `internal/app/persist/engine.go:552-587`; ADR 0001; REVIEW §7 S-01 |
| S-DM-02 | Persistence writes the map as JSON Lines rather than one JSON document; Spec amended 2026-11-19 to match | superseded-by-amendment | `internal/app/persist/codec.go`; REVIEW §13 |
| S-AVL-03 | Consumer's client library keeps the last read value in a local file for outage fallback | consumer-zone | ADR 0007; REVIEW §7 S-11; 14.7 |
```

### ADR skeleton (slim MADR per D-09/D-10/D-11)

```markdown
---
adr: 0001
review-decision: "14.1 (Snapshot load — full rollback as per Spec)"
informs: [APIS-07]
date: 2026-11-19
status: accepted
---

# 0001 — Snapshot load is a full rollback

## Context
Review decision 14.1 (REVIEW.md §14): the v1.0 snapshot load merges instead of
restoring — keys created after the snapshot survive it (stand finding F13, S-01).

## Decision
Snapshot load deletes keys absent from the snapshot (full rollback); the load
result broadcasts to peers (with PROTO-08).

## Consequences
`TestEngine_snapshotCreateLoadRestore` (internal/app/persist/engine_test.go) is
rewritten for the new behavior; APIS-07 implements this in Phase 6.

## Links
REVIEW.md §14 row 1; registry rows S-PERS-12, S-PERS-13, S-SEM-08, S-ALG-CU-09.
```

ADRs 0005 and 0007 add the loud line directly under the title (D-11 verbatim): **`accepted (by default; reopenable at Phase 4/6 discussion before implementation)`**.

### Relocation + guard + verification (verified against this repo's git state)

```bash
# 1. Move (both paths untracked — plain mv/rm, NOT git mv/git rm)
mkdir -p docs/reviews/2026-11-19-v1.0
mv docs/2026-11-19-v1.0-review/2026-11-19-v1.0-review/REVIEW.md docs/reviews/2026-11-19-v1.0/
mv docs/2026-11-19-v1.0-review/2026-11-19-v1.0-review/evidence  docs/reviews/2026-11-19-v1.0/
mv docs/2026-11-19-v1.0-review/2026-11-19-v1.0-review/internal  docs/reviews/2026-11-19-v1.0/
rmdir docs/2026-11-19-v1.0-review/2026-11-19-v1.0-review docs/2026-11-19-v1.0-review
rm docs/2026-11-19-v1.0-review.zip

# 2. Guard (D-04) — append to .gitignore:
#    docs/reviews/2026-11-19-v1.0/internal/

# 3. Prove the guard works
git check-ignore -v docs/reviews/2026-11-19-v1.0/internal/spec-trace.md   # must match

# 4. Stage explicitly, never 'git add docs/'
git add .gitignore docs/reviews/2026-11-19-v1.0/REVIEW.md docs/reviews/2026-11-19-v1.0/evidence

# 5. Prove nothing internal/ is staged
git status --porcelain | grep internal/   # expect empty
```

### Registry completeness asserts (run before commit)

```bash
R=docs/SPEC-DEVIATIONS.md
grep -c '^| S-' "$R"                                   # expect 167
grep -nE '\| (TBD|)\s*\|' "$R"                         # expect no hits (no TBD/empty status cells)
grep -c 'fixed-in-v1.1(APIS-07)' "$R"                  # >= 1  (S-01 closed)
grep -c 'fixed-in-v1.1(PROTO-08)' "$R"                 # >= 1  (S-02 closed)
grep -c 'fixed-in-v1.1(OBSV-05)' "$R"                  # >= 1  (S-06 closed)
grep -c 'fixed-in-v1.1(OBSV-06)' "$R"                  # >= 1  (S-07 closed)
grep -cE 'fixed-in-v1.1\(PROTO-0[134]\)' "$R"          # >= 1  (S-08 closed)
grep -c 'consumer-zone' "$R"                           # >= 1  (S-11 closed)
# stale-reference sweep after the move:
grep -rn '2026-11-19-v1.0-review' .planning/PROJECT.md .planning/ROADMAP.md .planning/REQUIREMENTS.md  # expect empty
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Full MADR ADRs with options analysis | Slim MADR (Context/Decision/Consequences/Links) | Locked by D-09 for this repo | No "considered options" ceremony; §14 already argued each choice |
| Local git exclusion for sensitive dirs (` .git/info/exclude`) | Committed `.gitignore` guard | D-04 (this phase) | Guard survives clones — local excludes do not |
| Spec compliance asserted ad hoc at phase gates | Statement-ID registry with fixed status enum | D-05–D-08 (this phase) | Later phases flip statuses; Phase 9 greps for unclosed rows |

Nothing in this phase depends on fast-moving external state — no deprecations affect it.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Registry summaries and ADRs are written in English (D-09 states English explicitly for ADRs; the registry follows the same repo docs convention — not stated verbatim for the registry) | Pattern 1, Pattern 3 | Cosmetic; a Russian registry would still satisfy D-05/D-06 mechanics |
| A2 | Proposed ADR slugs (Pattern 3 table) — CONTEXT lists slugs as discretion; only the 0001..0011 numbering and 14.x mapping are locked | Pattern 3 | None — planner/executor may rename freely |
| A3 | Status choices for the ~10 rows flagged as decision points resolve by the priority rule (§14 > §7 > trace note, amend-check first) | Transform rules, Open Questions 1–3 | Low — worst case a row's status differs but stays inside the enum and honest |
| A4 | `deferred-v2(SNAP-01)` (not `consumer-zone`) chosen for S-IMP-04 since REQUIREMENTS tracks SNAP-01 as a v2 requirement | Transform rules | Cosmetic — both statuses are true of it; enum membership holds either way |

## Open Questions

1. **Status for the three stand-downgraded rows (S-FAIL-09, S-ALG-RC-10, S-FAIL-15).**
   - What we know: trace marks them `Implemented`/`Implemented-with-deviation` per code; the stand proved the stated bounds fail at 10 000 keys (§13:373 names exactly these three: S-FAIL-09 and S-ALG-RC-10 fail at 10k keys per R-01/R-02; S-FAIL-15 holds only on hot keys per R-02/PERF-02).
   - What's unclear: `implemented` (statement as written, caveats in summary) vs `fixed-in-v1.1(XPORT-01/02, PERF-02)`.
   - Recommendation: `fixed-in-v1.1(<corresponding REQ>)` with an own-words summary noting it holds for small N — honest and consistent with how every other live gap is closed.
2. **S-SEM-07 closure flavor.** In both the amendment-removed list (§13) and the Missing five. Recommendation: `superseded-by-amendment` with ADR 0004 in links (the conflict-journal requirement was removed from the Spec by decision 14.4); `accepted-deviation(ADR 0004)` is the defensible alternative — pick one primary status for enum cleanliness.
3. **S-HLC-07 (version filler zero vs Spec's "random").** Not in the 12 removed rows; no v1.1 requirement randomizes it; M-25 (IDEA.md wording) is a Phase 9 doc fix. Recommendation: `accepted-deviation` with summary "documented zero filler; uniqueness/order unaffected" and links to M-25/Phase 9 — but amend-check SPEC-amended.md first in case the amendment touched the wording.
4. **Whether to touch `.planning/phases/01-…/01-CONTEXT.md`'s stale canonical-ref paths.** Recommendation: leave untouched (decision record); PLAN.md restates post-move paths; Runtime State Inventory documents the mapping.

## Environment Availability

SKIPPED — docs-only phase with no external dependencies (no tools, services, or runtimes beyond git and the repo's existing quality gates, all already in use).

## Validation Architecture

Docs-only phase: no new automated test framework; validation is scripted derivation asserts plus the existing suite (which must stay green untouched).

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Shell asserts (grep/awk) + existing Go suite (unchanged, `go test -race ./...`) |
| Config file | none needed (`.golangci.yml`/Makefile untouched) |
| Quick run command | Registry completeness asserts (Code Examples block — each < 1 s) |
| Full suite command | `make lint && make test` (AGENTS.md pre-push gate) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SPEC-01 | 167 rows, enum-conformant, no TBD, §7 closures present | scripted grep/awk | asserts in Code Examples | ❌ run inline (Phase 9 adds the permanent grep test per D-08/DOCS-08) |
| SPEC-01 | internal/ uncommitable | git guard check | `git check-ignore -v docs/reviews/2026-11-19-v1.0/internal/spec-trace.md` | ❌ inline |
| SPEC-01 | no stale old-path refs | grep | `grep -rn '2026-11-19-v1.0-review' .planning/{PROJECT,ROADMAP,REQUIREMENTS}.md` → empty | ❌ inline |
| SPEC-02 | 11 ADRs exist with 14.x mapping; D-11 lines in 0005/0007 | scripted grep | `ls docs/adr/0[01][01]-*.md | wc -l` = 11; `grep -l 'reopenable at Phase 4/6' docs/adr/000[57]-*.md` | ❌ inline |
| SPEC-01/02 | repo quality gates unaffected | existing suite | `make lint && make test` | ✅ |

### Sampling Rate
- **Per task commit:** the relevant inline asserts (registry count/enum after registry tasks; check-ignore after the move task).
- **Per wave merge:** full assert block + `git status --porcelain | grep internal/` (empty).
- **Phase gate:** `make lint && make test` green (no code changes expected — green confirms zero accidental code impact) before `/gsd:verify-work`.

### Wave 0 Gaps
None — existing test infrastructure is untouched by design; permanent automated coverage of the registry arrives in Phase 9 (D-08). Inline scripted asserts above are the phase's validation.

## Security Domain

`security_enforcement: true`, ASVS level 1 (config). This phase touches no runtime surface — no authentication, session, access-control, input-validation, or cryptography code exists in the changeset. The applicable security property is **information disclosure control** over non-public material in a public repository.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — (no auth surface) |
| V3 Session Management | no | — |
| V4 Access Control | no | — (repo-level branch protection already enforced; PROJECT.md constraint) |
| V5 Input Validation | no | — (no code; the registry's own "input" is the fixed status enum, machine-checked) |
| V6 Cryptography | no | — |
| (Publication/content control — not an ASVS V2–V6 category but the phase's real control) | yes | D-01/D-04: committed `.gitignore` guard + selective `git add` + porcelain/check-ignore asserts |

### Known Threat Patterns for this changeset

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Non-public Spec text published via accidental staging of `internal/` | Information Disclosure | `.gitignore` guard line + explicit-path staging + pre-commit porcelain check (Pitfall 1; one-way consequence per D-01) |
| Spec content reconstructed from near-verbatim registry paraphrases | Information Disclosure | D-06 own-words summaries; substring-audit against trace statement cells |
| Spec-identifying strings inside committed review artifacts | Information Disclosure | Verified clean this session: grep for `confstor\|0031\|cyp-config\|cyp-perf` over REVIEW.md + evidence/ returns nothing [VERIFIED]; assert stays in the phase checklist |

No high-severity security findings block this phase provided the guard asserts pass.

## Sources

### Primary (HIGH confidence — all read this session)
- `docs/2026-11-19-v1.0-review/2026-11-19-v1.0-review/REVIEW.md` — §0 method (L-numbers point into pre-amendment edition; REVIEW/evidence contain no spec names/quotes), §7 deviations (S-01…S-08, S-11 table + S-12 paragraph), §13 trace summary (counts, amendment-removed rows, open-question positions), §14 decisions 14.1–14.11, §15 milestone shape, §16 evidence inventory
- `docs/2026-11-19-v1.0-review/2026-11-19-v1.0-review/internal/spec-trace.md` — 167-row trace: status vocabulary and counts, G-01…G-14 cross-references, per-row code/test evidence; contiguity + count census run mechanically
- `docs/2026-11-19-v1.0-review/2026-11-19-v1.0-review/internal/README.md` — internal/ contents and (stale) exclusion claim
- `docs/2026-11-19-v1.0-review/2026-11-19-v1.0-review/internal/SPEC-amendments.diff` — hunk inventory (amendment surface)
- `docs/2026-11-19-v1.0-review/2026-11-19-v1.0-review/evidence/docs-audit.md` — M-01…M-52; M-23 (IDEA.md conflict-journal lines); corpus/test-coverage notes (line 344: no test reads IDEA.md)
- `.planning/phases/01-spec-reference-and-decisions/01-CONTEXT.md` — all locked decisions D-01…D-13 (verbatim)
- `.planning/REQUIREMENTS.md` — SPEC-01/SPEC-02 wording; v1.1 REQ ID ↔ review-ID parentheticals; v2 list (SNAP-01); Out of Scope table
- `.planning/PROJECT.md` — milestone goal, constraints (public repo, PR-only, doc-is-test-input), line 66 parenthetical, old-path refs at :15/:137
- `.planning/ROADMAP.md` — old-path ref at :5; Phase 1 entry
- `.planning/config.json` — nyquist_validation true, security_enforcement true/ASVS L1, branching templates
- `.gitignore`, `.git/info/exclude` — current guard state
- `cmd/sup-kv/{operations,examples,readme}_test.go` — doc-test file targets (fixed paths, no globbing)
- `internal/app/repl/restart_test.go`, `internal/app/persist/engine_test.go` — locations of `TestRestart_newPeerNoFullSync` / `TestEngine_snapshotCreateLoadRestore` (registry link targets)
- `docs/IDEA.md` — banner target; conflict-journal lines confirmed at :139, :163, :169, :189, :213
- `docs/OPERATIONS.md:87-106` — drift runbook (S-06's "operator chases nonexistent NTP problem" context)

### Secondary (MEDIUM confidence)
- None needed — no external sources consulted; the domain is fully repo-local.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Registry derivation: HIGH — every ingredient list (167 counts, contiguity, 12+1 amendment rows, 5 Missing, 6 Out-of-scope, §7→G-xx row membership) verified mechanically against the trace this session; enum and closures locked verbatim by D-08
- Relocation/guard mechanics: HIGH — git state (untracked dir + zip, exclude contents, .gitignore shape) read directly; commands validated against that state
- ADR mapping: HIGH — §14 table read in full; REQ parentheticals quoted from REQUIREMENTS.md; one honest gap flagged (14.11 measurement has no dedicated REQ)
- Pitfalls: HIGH — each grounded in a verified repo state (stale exclude claim, nested dir, doc-test file targets, M-23 line numbers)

**Research date:** 2026-11-19
**Valid until:** 2026-12-19 (stable — all sources are versioned repo artifacts; the only mutable input is the working tree's untracked review directory, which this phase itself relocates)
