# Phase 1: Spec Reference and Decisions - Pattern Map

**Mapped:** 2026-11-19
**Files analyzed:** 13 (4 new artifacts, 1 moved payload, 1 moved local-only dir, 5 modified, 1 deleted, 1 created-by-move directory)
**Analogs found:** 9 / 13 (docs-only phase: pure relocations and reference edits need no analog; the registry and ADRs have style analogs but no same-role predecessor)

**Tracked-source gate:** every analog named below was verified present in `git ls-files` this session. Two exceptions are named as **inputs**, not analogs: `docs/2026-11-19-v1.0-review/**` (REVIEW.md, internal/spec-trace.md) is **untracked** (`??` in `git status --porcelain`) and REVIEW.md only becomes tracked after this phase's D-02 move + explicit staging. PATTERNS.md cites them as derivation sources with their post-move paths, never as tracked code analogs.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `docs/SPEC-DEVIATIONS.md` (new) | reference-registry doc | transform (scripted derivation from local trace) | schema input: `docs/reviews/2026-11-19-v1.0/internal/spec-trace.md` (LOCAL-ONLY, never committed); heading/table style: `docs/OPERATIONS.md` | schema-exact (input), style exact |
| `docs/adr/0001-<slug>.md` … `0011-<slug>.md` (11 new) | decision-record doc (slim MADR) | transform (from REVIEW §14) | source: `docs/reviews/2026-11-19-v1.0/REVIEW.md` §14 (rows 396–406); frontmatter precedent: `.planning/milestones/v1.0-phases/01-ci-bootstrap-and-repo-skeleton/01-01-PLAN.md`; skeleton: RESEARCH.md Pattern 3 | source-exact, format partial (no ADR exists in repo yet) |
| `docs/reviews/2026-11-19-v1.0/{REVIEW.md, evidence/}` (moved, then committed) | evidence archive | file relocation (plain `mv`, untracked → explicit `git add`) | none needed — mechanical per RESEARCH.md Pattern 2 | n/a (mechanical) |
| `docs/reviews/2026-11-19-v1.0/internal/` (moved, NEVER committed) | local-only derivation input | file relocation + gitignore guard | none needed — guarded by the new `.gitignore` line | n/a (mechanical) |
| `docs/IDEA.md` (modified — banner prepended) | historical design doc | prepend-only edit (body byte-untouched, D-13) | self: existing preamble lines 1–6; loud-marker wording source: `.planning/REQUIREMENTS.md:15` | partial (no public-doc banner precedent) |
| `.gitignore` (modified — +1 guard line) | repo config | append-one-line | self: final commented-block convention (lines 31–32) | exact (self-analog) |
| `.planning/PROJECT.md` (modified) | planning doc — old-path refs + parenthetical | in-place reference update | self: lines 15, 66, 137 (all old-path hits verified) | exact (self-analog) |
| `.planning/ROADMAP.md` (modified) | planning doc — old-path ref | in-place reference update | self: line 5 | exact (self-analog) |
| `.planning/REQUIREMENTS.md` (modified) | planning doc — old-path ref | in-place reference update | self: line 6 | exact (self-analog) |
| `AGENTS.md`, `.claude/CLAUDE.md` | — | none | **Verified zero old-path references** (grep this session) — resolves the CONTEXT discretion item; do not touch | n/a |
| `docs/2026-11-19-v1.0-review.zip` (deleted) | archive | deletion | none — plain `rm` (untracked; `git rm` would fail) | n/a (mechanical) |

## Pattern Assignments

### `docs/SPEC-DEVIATIONS.md` (reference-registry doc, transform)

**Schema basis (INPUT — local-only, never committed):** `docs/reviews/2026-11-19-v1.0/internal/spec-trace.md` (pre-move path: `docs/2026-11-19-v1.0-review/2026-11-19-v1.0-review/internal/spec-trace.md`). Copy the table SHAPE only — never the statement column (D-06).

**Trace columns declaration** (line 56) — the registry keeps ID / Status / evidence-links and replaces the statement column with an own-words summary:
```markdown
Columns: **ID** · **Spec statement** · **Status** · **Code evidence** · **Test evidence (judgement)** · **Deviation / notes (recorded or silent)**.
```

**Trace row shape** (lines 60–63) — pipe table, `|---|` separator, code evidence as `shortname:lines`, `—` for empty cells:
```markdown
| ID | Spec statement | Status | Code evidence | Test evidence | Deviation / notes |
|---|---|---|---|---|---|
| S-DM-01 | ... (L190) | Implemented | `store:83-89` (`mu sync.RWMutex`, `records map[string]Record`) | `TestStore_concurrentHammer` store_test.go:716 ‡ ... | — |
| S-DM-02 | ... (L191–192) | Implemented-with-deviation | `codec:62-85` | see S-PERS-01 | JSON Lines, see S-PERS-01 |
```

**Section-grouping shape** (lines 58, 54): registry rows group under `###` section headings matching the trace's subsections (D-05); the trace uses `## (b) Traceability table` then `### Модель данных (L188–L211)` — the registry uses the same hierarchy with English headings.

**Extraction is scripted, never hand-transcribed:** every line matching `^\| S-` is one row (167 total; per-prefix count == max contiguity). Status vocabulary seeds (trace lines 38–47): `Implemented` (116) · `Implemented-with-deviation` (22) · `Partial` (18) · `Missing` (5) · `Out-of-scope-for-sup-kv` (6) — transformed per RESEARCH.md's recipe table into the fixed D-08 enum. L-line refs (`L511–512`) are safe to carry — REVIEW.md (committed under D-01) uses the same convention at line 244.

**Heading/table style (tracked analog):** `docs/OPERATIONS.md` lines 1–8 — `#` title, bold-link intro paragraph pointing at sibling docs, sentence-case `##` sections:
```markdown
# sup-kv operations

The operational contract of a sup-kv cluster in one file: what the system
promises, what it does not promise, ... For trying the API see [docs/EXAMPLES.md](EXAMPLES.md);
for the full design see [docs/IDEA.md](IDEA.md).

## Trust model
```
The registry's preamble should follow this shape: `# sup-kv spec deviations` + an intro paragraph stating derivation source (review of 2026-11-19), the status enum, and a link to `docs/reviews/2026-11-19-v1.0/REVIEW.md`. It should also note S-12 closure (registry exists + IDEA.md historical) per RESEARCH.md §7-table last row.

**Row schema (D-06, locked):** `| ID | Summary (own words) | Status | Links |` — Links cell carries code path `file:lines`, ADR file, v1.1 REQ ID, REVIEW §7 ID. Worked examples are in RESEARCH.md Code Examples ("Registry row shape and enum").

---

### `docs/adr/0001-…-0011-<slug>.md` (decision-record doc, transform — 11 files)

**Source table (moves to tracked path `docs/reviews/2026-11-19-v1.0/REVIEW.md` in this phase):** REVIEW.md §14, lines 392–408. Each row's four columns map one-to-one onto slim-MADR sections (D-09):

```markdown
## 14. Решения владельца (приняты 2026-11-19)

| № | Вопрос | Решение | Что это значит для работ |
|---|---|---|---|
| 1 | Загрузка снимка (S-01) | полный откат, как в Спеке | код: ключи, которых нет в снимке, удаляются; ... тест `TestEngine_snapshotCreateLoadRestore` переписывается под новое поведение |
```
Mapping: Вопрос → **Context**; Решение → **Decision**; Что-это-значит-для-работ → **Consequences**; № + related REQ IDs + registry rows → **Links**. The Russian source stays in the review; ADRs are English (D-09). §14 closes with the dating note (line 408) — cite it in Context, not re-argue it.

**Number↔decision mapping and slugs:** locked table in RESEARCH.md Pattern 3 (0001=14.1 snapshot-load-full-rollback … 0011=14.11 value-limit-60kib-kept; slugs are discretion). ADR 0011 records the honest gap: the "measure loss share" consequence has no dedicated v1.1 REQ — say so in Consequences, invent no mapping.

**Frontmatter precedent (tracked analog):** `.planning/milestones/v1.0-phases/01-ci-bootstrap-and-repo-skeleton/01-01-PLAN.md` lines 1–6 — lowercase-key YAML frontmatter is the repo's established machine-readable-metadata convention (D-10 allows "frontmatter or Links section"; frontmatter matches repo practice):
```yaml
---
phase: 01-ci-bootstrap-and-repo-skeleton
plan: 01
type: execute
wave: 1
depends_on: []
```
Note: no file under public `docs/` uses frontmatter today — the ADR frontmatter is new-to-docs but consistent with planning-artifact practice. Keep keys lowercase, one fact per key (`adr:`, `review-decision:`, `informs:`, `date:`, `status:`), per RESEARCH.md's skeleton.

**Loud status line for ADRs 0005 and 0007 (D-11):** wording source is the SPEC-02 acceptance text in `.planning/REQUIREMENTS.md` lines 14–15 (tracked):
```markdown
- [ ] **SPEC-02**: Each owner decision 14.1–14.11 is recorded as an ADR, including the two confirmations the review left with the owner
  - Acceptance: ... 14.5(6) read-before-value-arrives → `Unavailable` and 14.7 last-read-value-file outside sup-kv are recorded as accepted (owner may reopen at Phase 4/6 discussion before implementation)
```
Place `**accepted (by default; reopenable at Phase 4/6 discussion before implementation)**` directly under the ADR title, before `## Context` — so Phase 4/6 agents see it on open.

---

### `docs/reviews/2026-11-19-v1.0/**` (relocation, mechanical)

No analog needed. Mechanics verified in RESEARCH.md Pattern 2: both old paths are untracked (`??`), so plain `mv` for `REVIEW.md`, `evidence/`, `internal/`, plain `rm` for the zip; flatten the nested duplicate (`docs/2026-11-19-v1.0-review/2026-11-19-v1.0-review/*`) one level — do not reproduce the dirname. Guard line must land in `.gitignore` in the same changeset; stage explicit paths only; assert `git check-ignore -v` matches and porcelain shows no `internal/` path.

---

### `docs/IDEA.md` (banner prepended, body untouched)

**Insertion point:** line 1 is `# sup-kv — Idea Document`, lines 3–6 are the existing italic preamble. The banner goes above line 1 (nothing precedes it today), body byte-untouched.

**Banner-content anchors (all tracked/verified):**
- Points to the as-built trio — the same links OPERATIONS.md lines 5–6 use: `[docs/EXAMPLES.md](EXAMPLES.md)` relative-link style; trio is README + `docs/OPERATIONS.md` + `docs/EXAMPLES.md` (D-12).
- Notes `docs/ARCHITECTURE.md` + as-built description land by Phase 9 (D-12).
- Loud-marker wording precedent: REQUIREMENTS.md:15's parenthetical style — see ADR excerpt above; keep the banner a blockquote or bold warning, visually louder than the existing italic preamble it sits above.
- **D-13 hard constraint:** lines 139, 163–164, 168–169, 188–190, 213 (conflict-journal promises) must not appear in the diff. Verified present in the read; scrub belongs to Phase 9.
- Companion one-line edit (discretion): `.planning/PROJECT.md:66` — `It comes from a reviewed architecture proposal (`docs/IDEA.md` — the authoritative, self-contained design document).` → minimally adjust the parenthetical to point at the historical banner. No other line of the paragraph.

---

### `.gitignore` (one guard line, self-analog)

**Existing convention** (tracked; lines 31–32, final block):
```gitignore
# Local environment
.env
```
Every entry sits under a `# Comment` block header. Append (D-04):
```gitignore

# Non-public review material (upstream Spec; never committed)
docs/reviews/2026-11-19-v1.0/internal/
```
Comment header matches repo style and explains the one-way D-01 reason. Verify with `git check-ignore -v docs/reviews/2026-11-19-v1.0/internal/spec-trace.md` — the current `.git/info/exclude` (only `/.planning/milestone.lock`) does NOT match this path and must not be relied on.

---

### `.planning/PROJECT.md` / `.planning/ROADMAP.md` / `.planning/REQUIREMENTS.md` (reference updates, self-analog)

**Verified complete old-path hit list** (grep this session; AGENTS.md and `.claude/CLAUDE.md` are clean):
- `.planning/PROJECT.md:15` — Milestone goal: `...review (`docs/2026-11-19-v1.0-review/`) — ...`
- `.planning/PROJECT.md:137` — footer: `*Last updated: ... (input: v1.0 review at `docs/2026-11-19-v1.0-review/`, ...)` (easy to miss — footer, not header)
- `.planning/ROADMAP.md:5` — Overview paragraph: `...review (`docs/2026-11-19-v1.0-review/`): ...`
- `.planning/REQUIREMENTS.md:6` — Source line: `**Source:** v1.0 review of 2026-11-19 (`docs/2026-11-19-v1.0-review/`). ...`

All four replace `docs/2026-11-19-v1.0-review/` → `docs/reviews/2026-11-19-v1.0/`; change nothing else on the line. Post-edit assert: `grep -rn '2026-11-19-v1.0-review' .planning/{PROJECT,ROADMAP,REQUIREMENTS}.md` → empty. Leave `.planning/phases/01-…/01-CONTEXT.md` untouched (decision record; RESEARCH.md Open Question 4).

## Shared Patterns

### Language and register
**Source:** `docs/OPERATIONS.md` (whole file), `docs/EXAMPLES.md` lines 1–12
**Apply to:** SPEC-DEVIATIONS.md, all 11 ADRs, IDEA.md banner
English, direct declarative sentences, sentence-case headings (`## Trust model`, `## Clock drift runbook`). Repo docs are English (D-09 makes it explicit for ADRs); the review stays the Russian source and is cited, not translated verbatim (also the D-06 safety mechanism — translation forces paraphrase).

### Table formatting
**Source:** `.planning/milestones/v1.0-ROADMAP.md` lines 218–226; `.planning/PROJECT.md` lines 102–118
**Apply to:** SPEC-DEVIATIONS.md; any tabular preamble in ADRs
GitHub-flavored pipe tables, left-aligned `|---|` separators, `—` for empty cells (spec-trace convention), inline code spans for paths/IDs (`file:lines`), no alignment colons. Keep one table per concern; group rows under `###` headings rather than adding a grouping column.

### YAML frontmatter (machine-readable metadata)
**Source:** `.planning/milestones/v1.0-phases/01-ci-bootstrap-and-repo-skeleton/01-01-PLAN.md` lines 1–6
**Apply to:** the 11 ADRs (D-10 mapping)
Lowercase keys, scalar/list values, no comments inside the block. Pair it with a `## Links` section for prose-level back-references (registry rows, REVIEW §14 row, related REQ IDs) — frontmatter for machines, Links for readers.

### Loud status markers
**Source:** `.planning/REQUIREMENTS.md` lines 14–15; `.planning/PROJECT.md` line 117 (`⚠️ Revisit at v1.0.1+/ruleset window`)
**Apply to:** IDEA.md banner; ADRs 0005 and 0007
Repo convention for "read this before proceeding": bold/blockquote marker line placed at the top of the artifact, wording carried from the acceptance text rather than paraphrased loosely (D-11 mandates verbatim: `accepted (by default; reopenable at Phase 4/6 discussion before implementation)`).

### Git guard + selective staging
**Source:** `.gitignore` lines 31–32 (block convention); RESEARCH.md Pattern 2 command sequence
**Apply to:** the relocation changeset
Guard line first, explicit-path `git add` only (never `git add docs/` or `-A`), `git status --porcelain | grep internal/` empty and `git check-ignore -v` matching before every commit in the phase. Commits follow the established conventional form `docs(01): …` via `gsd_run query commit` (git log `66aa0a0`, `4527d87` precedent).

### Doc-is-test-input fence caution
**Source:** `docs/EXAMPLES.md` lines 8–12 (`The marked blocks are executed verbatim by CI ...`)
**Apply to:** SPEC-DEVIATIONS.md and ADRs
Only README/EXAMPLES/OPERATIONS fenced blocks are CI-executed, and CI reads three fixed paths only (verified in RESEARCH.md — no docs/ globbing). Even so: new files must not introduce fenced blocks that look executable (no unlabelled ```bash blocks containing runnable commands). Prefer inline code spans and tables; if a fence is unavoidable, keep it illustrative (e.g. the status enum as a comment, as in RESEARCH.md's registry example).

## No Analog Found

| File | Role | Data Flow | Reason | Planner fallback |
|------|------|-----------|--------|------------------|
| `docs/adr/0001..0011-*.md` | decision-record | transform | No ADR directory or ADR file exists anywhere in the repo (`docs/adr/` absent; no `## ADR`/MADR hits in tracked docs) | RESEARCH.md Pattern 3 skeleton + §14 source table (excerpt above) + planning-artifact frontmatter convention |
| `docs/IDEA.md` banner | doc banner | prepend-only | No public `docs/` file carries a historical/banner block | REQUIREMENTS.md:15 acceptance wording + PROJECT.md:117 marker style (excerpts above) |
| `docs/reviews/2026-11-19-v1.0/**` relocation | archive move | file relocation | Nothing in repo history moves untracked dirs | RESEARCH.md Pattern 2 verified command sequence (mechanical, no style decision) |

## Metadata

**Analog search scope:** `docs/`, `.planning/` (root files, `milestones/`, `milestones/v1.0-phases/`), `.claude/CLAUDE.md`, `AGENTS.md`, `.gitignore`
**Files scanned:** 20+ (git ls-files census of docs/ and .planning/, frontmatter scan over all tracked .md, old-path grep over the five named files)
**Tracked-source gate:** passed — all analog paths verified in `git ls-files`; untracked review inputs explicitly labeled as local-only derivation sources
**Pattern extraction date:** 2026-11-19
