---
phase: 03-transport
plan: 03
subsystem: version-validation
tags: [rfc9562, uuidv7, validation, clock-poisoning, xport-04, d-06, sentinel-errors, property-test]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: "the RFC 9562 layout constants, PackVersion/Unpack/Compare and the version package"
  - phase: 02-stand-and-simulator-as-tests
    provides: "the seeded testutil RNG generators (RandRawVersion) and the replay-seed convention"
provides:
  - "version.RecordVersion.Validate(now, maxLead) — the pure layout + future-bound predicate XPORT-04 requires"
  - "ErrVersionLayout and ErrVersionFuture sentinels matched with errors.Is"
  - "A pinned inclusive boundary (exactly now+maxLead accepted, +1 ms rejected) and the RFC 9562 filler-bit interoperability rule"
  - "Totality proof over arbitrary 16-byte wire input and consistency with PackVersion"
affects: [03-08-repl-core-validation, 03-09-udprepl-adapter, 03-11-scenarios, 05-rpc-error-mapping]

actuals:
  tokens: 3167
  tasks: 2
  commits: 2
plan_head_before: c65041415b65b54157202772a0f4d469f171f638
plan_head_after: 39f0b07c28fff872275e874380d74a7833b29e2f

tech-stack:
  added: []
  patterns:
    - "Pure predicate seam: Validate takes now as a parameter, reads no clock and allocates nothing (D-02)"
    - "Overflow-safe future bound: l >= nowMs && l-nowMs > lead, with a negative lead clamped to zero"
    - "Sentinel errors in the owning package, matched with errors.Is so Phase 5 can map them to RPC codes"
    - "Property model recomputed independently in the test from the documented masks and Unpack"

key-files:
  created:
    - internal/app/version/validate_test.go
  modified:
    - internal/app/version/version.go

key-decisions:
  - "The layout verdict runs before the future-bound arithmetic: a record with garbage version/variant bits never reaches the subtraction, and both sentinels are distinguishable for the pipeline's bad_version class (XPORT-04, D-06)."
  - "The 46 rand_b filler bits RFC 9562 leaves to the implementation are accepted, not rejected: rejecting them would break conforming UUIDv7s from other vendors (research Open Question 9)."
  - "The future bound is inclusive at now+maxLead and exclusive one millisecond beyond; milliseconds are the only precision the layout carries, so a fractional now is truncated to unix_ts_ms."
  - "A negative maxLead normalizes to zero via max(maxLead.Milliseconds(), 0), so only readings at or before now are accepted."
  - "The packed property arm stratifies half its draws within one second of the edge (still inside the plan's [nowMs-1h, nowMs+61s] domain) so the future branch is non-vacuous, and asserts both branches occurred."
  - "Seed logging delegates to testutil.NewRNG, whose printed 'property seed:' line plus replay command is the repo's seeded-RNG convention named in the task's read_first."

patterns-established:
  - "Sender-supplied time is untrusted input: validate it against the local clock before any clock or store call."
  - "Version-validation tests separate a table of exact boundary rows from a property arm with an independent model."

requirements-completed: [XPORT-04]

coverage:
  - id: D1
    description: "Validate enforces the RFC 9562 UUIDv7 layout and rejects readings beyond now+max_lead, returning ErrVersionLayout / ErrVersionFuture matched with errors.Is"
    requirement: XPORT-04
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/version/ -run '^TestValidate$'"
        status: pass
    human_judgment: false
  - id: D2
    description: "Validate is total over arbitrary 16-byte wire input and consistent with PackVersion, with a replay seed logged"
    requirement: XPORT-04
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/version/ -run '^TestValidate_property$'"
        status: pass
    human_judgment: false
  - id: D3
    description: "RFC 9562 filler bits in rand_b are accepted for third-party UUIDv7 interoperability"
    requirement: XPORT-04
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/app/version/ -run '^TestValidate$/^random_filler_bits_in_rand_b_are_accepted$'"
        status: pass
    human_judgment: false

# Metrics
duration: 2min
completed: 2026-11-21
status: complete
---

# Phase 3 Plan 03: Version Validation Summary

**Pure UUIDv7 `Validate` rejects non-RFC 9562 layout bits and readings beyond the inclusive `now + max_clock_lead` bound before any clock call, proven total over 2 000 raw and 1 000 packed versions**

## Performance

- **Duration:** 2 min (148 s)
- **Started:** 2026-11-21T18:41:54Z
- **Completed:** 2026-11-21T18:44:22Z
- **Tasks:** 2
- **Files modified:** 2 (1 created, 1 modified)

## Accomplishments

- `version.RecordVersion.Validate(now time.Time, maxLead time.Duration) error` implements XPORT-04's validation half of D-06: the version 7 nibble and variant `0b10` masks (`uuidVerBits` / `uuidVarBits`) reject `ErrVersionLayout`, and the subtraction form `l >= nowMs && l-nowMs > lead` rejects `ErrVersionFuture` without overflow for a 48-bit reading or an absurd lead.
- The exact boundary is pinned: a reading of exactly `now + maxLead` is accepted, one millisecond beyond is rejected; a negative lead normalizes to zero; a fractional `now` truncates to its unix_ts_ms.
- The RFC 9562 interoperability decision is recorded and tested: the 46 `rand_b` filler bits are accepted, so a conforming UUIDv7 from another vendor passes.
- Totality is proven: 2 000 seeded arbitrary 16-byte versions run through a recover guard and agree with an independently computed layout/bound model, plus 1 000 `PackVersion` outputs around the future edge obey the inclusive boundary. The replay seed is printed by `testutil.NewRNG`.
- Threat mitigations delivered: T-3-07 (far-future version rejected before `ApplyRemote`), T-3-08 (layout masks pinned by property plus filler-bit/wrong-nibble rows), T-3-09 (subtraction bound, zero-clamped lead, `math.MaxInt64/2` lead rows). T-3-SC: zero installs, `go.mod`/`go.sum` untouched.

## Task Commits

Each task was committed atomically:

1. **Task 1: Validate — layout and future-bound predicate with sentinel errors** — `cc0cdba` (feat)
2. **Task 2: Property — Validate is total over arbitrary versions and consistent with PackVersion** — `39f0b07` (test)

**Plan metadata:** see the `docs(03-03)` commit that follows this summary.

## Files Created/Modified

- `internal/app/version/version.go` — adds `ErrVersionLayout`, `ErrVersionFuture` and `RecordVersion.Validate` after `Compare`; imports `errors` and `time`; no existing behavior changed.
- `internal/app/version/validate_test.go` — `TestValidate` (15-row boundary table) and `TestValidate_property` (2 000 raw + 1 000 packed draws with helpers `layoutOK`, `modelVerdict`, `verdictMatches`, `validateSafe`).

## Decisions Made

- Layout is checked before the future bound, matching the inbound pipeline order and keeping the sentinels distinguishable for the `bad_version` discard class.
- Sentinels live in the `version` package (there is no `errors.go` there today) and are matched with `errors.Is`, per the `storage/errors.go` convention; Phase 5 maps them to RPC codes.
- The `max` builtin normalizes a negative lead to zero (see Deviations).
- The property test recomputes the expected verdict from the documented masks and `Unpack` rather than calling back into `Validate`, so the model is independent.
- Half the packed draws are stratified within one second of the edge — still inside the plan's `[nowMs-1h, nowMs+61s]` interval — and the test asserts both the accepted and future branches occurred, so the property cannot pass vacuously.
- Seed logging uses `testutil.NewRNG`'s printed `property seed:`/replay line, the seeded-RNG convention the task's `read_first` names, rather than a bespoke `t.Logf`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Replaced the negative-lead if-statement with the `max` builtin**
- **Found during:** Task 1 (lint gate)
- **Issue:** The plan's code example clamps a negative lead with `if lead < 0 { lead = 0 }`; the vendored `modernize` linter rejects that form and the task's lint gate requires a clean run.
- **Fix:** `lead := max(maxLead.Milliseconds(), 0)` — semantics identical (clamp at zero); the doc comment already states the normalization.
- **Files modified:** `internal/app/version/version.go`
- **Verification:** `go tool golangci-lint run ./internal/app/version/` clean; the two negative-lead rows in `TestValidate` pass.
- **Committed in:** `cc0cdba` (part of Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Lint-gate compliance only; the predicate's contract, boundary semantics and error surface are exactly as planned. No scope creep.

## Issues Encountered

- `gsd_run` was not on PATH in the executor shell; the init/config queries were run through the absolute launcher `/home/nil/.zcode/gsd-core/bin/gsd_run`. No effect on the code changes.
- The plan ledger was created fresh for this plan (base `c650414`) with no stale artifact, so the measured `commits: 2` matches the two task commits.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None - no stubs, placeholders or hardcoded empty values were introduced.

## Threat Flags

None - this plan adds a pure predicate over already-received bytes; no new network endpoint, auth path, file access or schema surface.

## Next Phase Readiness

- 03-08's inbound validation pipeline can call `Validate` before `ApplyRemote`; `repl.Config.MaxClockLead` and the `bad_version` counter belong to that plan, not this one.
- The change is additive: no existing function signature changed, so all dependent packages and the simulator compile untouched.
- Verification is green: `go test -race ./internal/app/version/`, `go tool golangci-lint run ./internal/app/version/`, no `time.Now` in `version.go`, no `go.mod`/`go.sum` change.

## Self-Check: PASSED

- Files: `internal/app/version/version.go` and `internal/app/version/validate_test.go` exist.
- Commits: `cc0cdba` and `39f0b07` are ancestors of HEAD.
- Gates re-run at close-out: `TestValidate`, `TestValidate_property`, full package race test and package lint all pass.

---
*Phase: 03-transport*
*Completed: 2026-11-21*
