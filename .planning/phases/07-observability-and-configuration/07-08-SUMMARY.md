---
phase: 07-observability-and-configuration
plan: 08
subsystem: observability
tags: [logging, metrics, discard-classes, slog, prometheus, catalog]
requires:
  - phase: 07-observability-and-configuration
    provides: the adapter/mirror/exporter surface (07-01) and the peer/drift event seams plus startup/load renderers (07-07)
provides:
  - the unbound_listing discard class across core, adapter and exporter, with its own counter, rate-limited WARN and metric label
  - the decoded proto-version mismatch WARN, so a foreign proto_version is never silent on either branch
  - the catalog level matrix (TestCatalogLevels) over every named renderer
affects: [07-09, 07-10, 07-11, docs/OPERATIONS.md]
actuals:
  tokens: 8991
  tasks: 2
  commits: 3
plan_head_before: 29e893a4fbc0f0c48de1b4c61e121edf8c289fdf
plan_head_after: 43f11c18e92d94f083906a1fdbe5bae7bbc265c0
tech-stack:
  added: []
  patterns:
    - "A discard class is one literal shared by the core token, the adapter mirror and the metric label"
    - "Catalog levels are proven by calling the named renderer against a captured slog handler"
key-files:
  created:
    - cmd/sup-kv/catalog_internal_test.go
  modified:
    - internal/app/repl/core.go
    - internal/app/repl/discard_test.go
    - internal/app/repl/listedkeys_internal_test.go
    - internal/udprepl/adapter.go
    - internal/udprepl/mirror.go
    - internal/udprepl/decode_negative_test.go
    - internal/udprepl/scenario_test.go
    - internal/metrics/metrics.go
    - internal/metrics/metrics_test.go
    - cmd/sup-kv/main.go
    - cmd/sup-kv/main_test.go
    - docs/04-MULTI-NODE.md
key-decisions:
  - "unbound_listing is its own class; the three identity paths (own node_id, second handle, alternation) keep duplicate_node_id"
  - "The unbound-listing WARN drops the 'configuration error' wording — such a listing may legitimately overtake its Hello — and names CR-01 instead"
  - "OPERATIONS.md's discard label catalog stays with 07-10, which explicitly owns that edit; docs/04-MULTI-NODE.md had no owner and was corrected here"
  - "TestProtoVersionWarns lives in discard_test.go rather than a new file, keeping the realized footprint inside the plan's declared file set"
patterns-established:
  - "Pattern: named renderer + captured-handler matrix — every catalog entry's level and message is asserted where the binary wires it"
  - "Pattern: a class split updates the counter, adapter mirror, metric label, totality sums, class switches and doc prose in one commit"
requirements-completed: []
coverage:
  - id: D1
    description: "unbound_listing class end-to-end: DiscardUnboundListing token, DiscardStats.UnboundListing, rate-limited WARN without 'configuration error', adapter mirror and zero-state metric label"
    requirement: "OBSV-06"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run 'TestDiscard|TestSync_unbound|TestSync_forgedRun'"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/udprepl/ ./internal/metrics/ -run 'TestDecode|TestScenario|TestMetrics'"
        status: pass
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestDiscard_unboundListingClass"
        status: pass
    human_judgment: false
  - id: D2
    description: "The decoded ProtoVersion-mismatch branch counts, fires OnDiscard and WARNs exactly like the adapter verdict branch"
    requirement: "OBSV-06"
    verification:
      - kind: unit
        ref: "internal/app/repl/discard_test.go#TestProtoVersionWarns"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/repl/ -run 'TestProtoVersionWarns|TestDiscard|TestPeerEvents'"
        status: pass
    human_judgment: false
  - id: D3
    description: "Catalog level matrix: every named renderer called directly and asserted at its documented level and message, the four Spec-mandated entries included"
    requirement: "OBSV-06"
    verification:
      - kind: unit
        ref: "cmd/sup-kv/catalog_internal_test.go#TestCatalogLevels"
        status: pass
    human_judgment: false
  - id: D4
    description: "Process-level parsed adapter record pinned at WARN beside msg=Replication adapter for the duplicate-node_id path"
    requirement: "OBSV-06"
    verification:
      - kind: process
        ref: "cmd/sup-kv/main_test.go#TestLog_jsonDefault"
        status: pass
    human_judgment: false
  - id: D5
    description: "Metrics label is a fixed enum value and renders at zero when unseen, contributing no attacker-controlled cardinality"
    requirement: "OBSV-03"
    verification:
      - kind: unit
        ref: "internal/metrics/metrics_test.go#TestMetrics_families / #TestMetrics_values"
        status: pass
    human_judgment: false
duration: 20min
completed: 2026-11-23
status: complete
---

# Phase 07 Plan 08: Unbound-listing class and catalog level matrix Summary

An unbound listing now reports as its own `unbound_listing` class end-to-end — counter, OnDiscard, rate-limited WARN and zero-state metric label — the decoded proto-version branch warns like the adapter verdict, and `TestCatalogLevels` pins every catalog entry at its documented level.

## Performance

- **Duration:** 20min (started 2026-11-23T10:33:32Z, completed ~10:53:46Z)
- **Tasks:** 2 of 2
- **Commits:** 3 (2 task commits + 1 doc-alignment commit)
- **Files modified:** 13 (1 created, 12 modified)
- **Realized diff:** 398 insertions, 44 deletions

## Accomplishments

- The CR-01 listing-binding drop stopped masquerading as `duplicate_node_id`. `DiscardUnboundListing = "unbound_listing"` is a first-class token with its own `DiscardStats.UnboundListing` counter, an OnDiscard detail naming node and handle, and a WARN that says what happened (`CR-01`) instead of claiming a configuration error — a listing may legitimately overtake its Hello.
- The class is mirrored through `udprepl` (`unboundListing` atomic, `Discards()` literal, `mirrorStats`) and published as `supkv_datagrams_discarded_total{class="unbound_listing"}`, rendering at zero when unseen.
- Only the `listingBound` site moved: the own-node-id, second-handle and alternating-run paths still increment `duplicate_node_id`, each proven by the existing rows.
- Both proto-version branches now count, fire OnDiscard and WARN through the shared per-(node_id, class) limiter; `TestProtoVersionWarns` proves one WARN and one counter per branch plus a silent repeat inside the interval.
- `logAdapterWarn` and `logDumpFailure` are named renderers wired as the adapter `Log` seam and the engine `OnDumpFailure` callback; `TestCatalogLevels` calls twelve entries and asserts one record at the documented level/message each — the four Spec-mandated entries (ERROR `Dump failed`, ERROR `No usable dump generation`, WARN `Replication adapter` for foreign proto_version and for duplicate node_id) included.
- `TestLog_jsonDefault` now fails unless the driven adapter record's parsed level is exactly `WARN`, pinning the process-level half of the duplicate-node_id entry.

## Task Commits

1. **Task 1: The unbound_listing class from the core counter to the metrics label** — `e17ecaa` (feat)
2. **Task 2: The proto-version WARN and the catalog level matrix** — `a453cc6` (feat)
3. **Doc alignment (deviation)** — `43f11c1` (docs)

## Files Created/Modified

- `internal/app/repl/core.go` — `DiscardUnboundListing` token, `UnboundListing` counter, `listingBound` routing and WARN text; decoded proto-version WARN; doc comments for `handleDataSync`, `listingBound`, `OnDiscard` and `warnRateLimited`
- `internal/app/repl/discard_test.go` — model `classify`/`bump`/`onlyClass`/`sumDiscards`; `TestDiscard_unboundListingClass`; `TestProtoVersionWarns`; decoded-proto row now expects its WARN
- `internal/app/repl/listedkeys_internal_test.go` — both unbound/forged-run tests assert `UnboundListing == 1` and `DuplicateNodeID == 0`
- `internal/udprepl/adapter.go`, `internal/udprepl/mirror.go` — `unboundListing` atomic field, `Discards()` mirror, `mirrorStats` store
- `internal/udprepl/decode_negative_test.go`, `internal/udprepl/scenario_test.go` — class switches carry the new token
- `internal/metrics/metrics.go`, `internal/metrics/metrics_test.go` — `unbound_listing` sample, zero-state line and by-effect value 17
- `cmd/sup-kv/main.go` — named `logAdapterWarn` and `logDumpFailure` renderers, wired into `newReplicationAdapter` and `startEngine`
- `cmd/sup-kv/catalog_internal_test.go` — the new level matrix (`TestCatalogLevels`)
- `cmd/sup-kv/main_test.go` — `TestLog_jsonDefault` requires `level == WARN`
- `docs/04-MULTI-NODE.md` — CR-01 paragraph names `unbound_listing`

## Decisions Made

- **Class boundary:** `unbound_listing` covers the whole CR-01 binding (unknown node_id or run-mismatched row); the three identity paths keep `duplicate_node_id`. This follows D-04's New-2 exactly and the plan's prohibition.
- **WARN wording:** the unbound listing line omits "configuration error" and names `CR-01` and the node_id; the class token remains the stable machine key (OnDiscard + limiter slot), not message text.
- **Test placement:** `TestProtoVersionWarns` went into `discard_test.go` (already in the plan's file set) instead of a new repl test file.
- **Doc ownership:** `docs/OPERATIONS.md`'s label catalog is 07-10's explicit action item, so it was left untouched; `docs/04-MULTI-NODE.md` had no owning plan and was corrected here.
- **Requirements:** `OBSV-06` and `OBSV-03` stay unchecked — `requirements.ready-ids` reports both blocked because sibling plans in this phase also declare them (shared-ID gate).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Lint] Reworded a pre-existing doc-comment token flagged by the lint gate**
- **Found during:** Task 2, running `golangci-lint` over `cmd/sup-kv/...`
- **Issue:** `godoclint` flagged `fs.ErrNotExist` in `logBackupFallback`'s doc comment (present since `881d681`, plan 05-06) as needing a doc link. The line is not touched by this plan, but AGENTS.md makes the lint gate mandatory.
- **Fix:** reworded to "recorded not-exist cause" — doc-only, no behavior change.
- **Files modified:** `cmd/sup-kv/main.go`
- **Verification:** `make lint` exits 0.
- **Commit:** `a453cc6`

**2. [Rule 1 - Test] Decoded-proto row gained its expected WARN**
- **Found during:** Task 2, first run of the Task 1 verify filter
- **Issue:** `TestDiscard_classes` asserts "healthy datagram logged none" for rows without `wantLogged`/`wantWarn`; the newly warning decoded-proto row tripped it.
- **Fix:** the row now sets `wantWarn: repl.DiscardBadProtoVersion` and is named "...classifies BadProtoVersion and warns".
- **Files modified:** `internal/app/repl/discard_test.go`
- **Commit:** `a453cc6`

**3. [Rule 2 - Totality] The inline conservation sum now counts every class**
- **Found during:** Task 1, updating the totality model
- **Issue:** `TestDiscard_totality`'s inline sum omitted `BadSyncHeader` (latent: the generator's `syncPart` always emits a valid header), so "the sum equals the discarded count" was not actually total.
- **Fix:** the sum includes `UnboundListing` and `BadSyncHeader` (twelve counters); `sumDiscards`/`onlyClass` gained the new class too.
- **Files modified:** `internal/app/repl/discard_test.go`
- **Verification:** `TestDiscard_totality` and the `TestDiscard_beforeClock` totals pass.
- **Commit:** `e17ecaa`

**4. [Rule 2 - Docs] `docs/04-MULTI-NODE.md` named the old class**
- **Found during:** close-out verification against the plan's success criterion ("across core, adapter, exporter and docs")
- **Issue:** the CR-01 paragraph still said an unbound listing is discarded as `duplicate_node_id`; the split made that sentence false, and no phase-7 plan owns the file.
- **Fix:** the sentence names `unbound_listing`.
- **Files modified:** `docs/04-MULTI-NODE.md`
- **Commit:** `43f11c1`

## Issues Encountered

- **Known wall-clock ratio flake, not a regression.** `TestCore_keyCountCostFlat` failed once in a combined run (DataRequest ratio 2.20, 4809 vs 10562 ns/op) and once standalone (DataSync ratio 2.84, 44 120 vs 125 209 ns/op). The DataRequest path is untouched by this plan, and three consecutive standalone re-runs passed at ratios 0.37, 0.46 and 1.01 with a full-package pass afterwards (158 s). This is the flake STATE.md already deferred to the Phase-8 flake owner; no bound was relaxed and no test was changed.
- **Frozen red anchors intact.** Both red-runner mutation seds were dry-run after the `core.go` edits (the `storage` `out[key] = rec.Version` target and the `core.go` `if !c.nextSync.IsZero() && now.Before(c.nextSync) {` target): each still matches and produces the intended mutation. The frozen tests and `scripts/red-expectations.tsv` are byte-identical to the plan base.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Ready for 07-09. Three plans remain in Phase 7 (07-09, 07-10, 07-11).
- 07-10 must add `unbound_listing` to the OPERATIONS monitoring label catalog and the lifecycle bullets; the level names `TestCatalogLevels` asserts are the ones it should document (`Starting sup-kv`, `Dump load`, `Dump generation fallback`, `No usable dump generation`, `Dump failed`, `Startup gate opened`, `Peer appeared`, `Peer restarted`, `Peer evicted`, `Clock drift`, `Replication adapter`).
- `OBSV-06` and `OBSV-03` remain pending until the last sibling plan that declares them completes.

## Self-Check

- Created file exists: `cmd/sup-kv/catalog_internal_test.go` — FOUND
- Commits exist on HEAD's ancestry: `e17ecaa`, `a453cc6`, `43f11c1` — FOUND
- Working tree clean at SUMMARY time — confirmed
