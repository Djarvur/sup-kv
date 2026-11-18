---
phase: 02-engine-hlc-versioning-and-lww-store
fixed_at: 2026-11-02T06:42:20Z
review_path: .planning/phases/02-engine-hlc-versioning-and-lww-store/02-REVIEW.md
iteration: 1
findings_in_scope: 3
fixed: 3
skipped: 0
status: all_fixed
---

# Phase 02: Code Review Fix Report

**Fixed at:** 2026-11-02T06:42:20Z
**Source review:** .planning/phases/02-engine-hlc-versioning-and-lww-store/02-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 3 (IN-01, IN-02, IN-03 — all open findings, all Info)
- Fixed: 3
- Skipped: 0

## Fixed Issues

### IN-01: Unused constant `varPayloadMask`

**Files modified:** `internal/app/version.go`
**Commit:** 329263e
**Applied fix:** Deleted the dead `varPayloadMask = 0x3F` constant (const-block
alignment re-flowed) and added one precise sentence to `Unpack`'s doc contract
stating the reliance the constant used to hint at: the variant bits of octet 8
carry no explicit mask — the nodeID's left shift truncates them out of the
`uint16`.

### IN-02: No guard at the 48-bit `l` ceiling — counter overflow wraps the version stream to zero

**Files modified:** `internal/app/clock.go`, `internal/app/clock_test.go`
**Commit:** 03e3f9c
**Applied fix:** Saturating ceiling, VERS-02 no-wrap spirit preserved:
- `clock.go` gains `maxL = tsMask` beside `counterMax`. `Tick` clamps a wall
  reading beyond the domain to `maxL` (adopting it only when still strictly
  ahead, so drift stays computed on the raw reading and the counter keeps
  climbing) and, when the counter-overflow bump would push `l` past the
  ceiling, repeats the top reading `(maxL, counterMax)` instead of wrapping.
  `Update` clamps `newL` to `maxL` and pins the state at `(maxL, counterMax)`
  on the saturated overflow path. `NewClock`'s doc contract documents the
  `(maxL, counterMax)` domain end: a `now` seam beyond it — possible only
  with a clock lying past year ~8921 — saturates, and writes at the
  exhausted domain may repeat versions, an honest documented degradation,
  never a wrap to zero; `Update`'s doc notes the ceiling. Below the ceiling
  every branch is unchanged: the clamp and saturation conditions are
  unreachable there, and the untouched 02-02 Kulkarni proof suites still
  pass, so their validity carries over byte-for-byte.
- `clock_test.go` gains `TestClock_ceilingSaturation`: deterministic rows for
  the saturated Tick bump and the saturated Update overflow, plus a seeded
  adversarial walk through the injected `now` seam (D-02) with the wall
  beyond the ceiling and the counter run past `counterMax`, asserting every
  emitted reading holds `l = maxL` and no packed version ever reads below
  its predecessor (monotone, no wrap-to-zero).

### IN-03: `TestVersion_unpackTotal` range assertions are tautologies

**Files modified:** `internal/app/version_test.go`
**Commit:** c00c618
**Applied fix:** Replaced the four impossible-to-fail checks (range/counter/
nodeID/antisymmetry) with the substantive canonicalization property: a new
`canonicalRaw` helper normalizes exactly the RFC 9562 marker and filler bits
(octet 6 high nibble `0x7`, octet 8 top bits `0b10` with bit `0x40` cleared,
octet 10 low six bits and octets 11-15 zeroed) and the test asserts
`PackVersion(v.Unpack()) == canonicalRaw(v)` over 1000 seeded raw 16-byte
values — Unpack is total and lossless over every bit it keeps. The
panic-freedom purpose is kept: raw bit patterns still flow through Unpack
and Compare.

## Prior-Fixed Reference

### WR-01: Restart seam seeds only `l` — post-restart writes can collide with or lose to pre-restart versions

**Status:** prior-fixed by gap-closure 02-04 (commits a5a78a5, 68e4b62,
b4b5397); verified in the 2026-11-02 incremental review pass. Not re-fixed
this iteration. Its residual theoretical corner — a dump watermark at
`l = 2^48-1` with counter 4095 still wrapping on the overflow bump — is
closed by IN-02 above.

## Verification

| Finding | Gate result | Extra evidence |
|---------|-------------|----------------|
| IN-01 | lint + `go test -race ./...` green, `go.mod`/`go.sum` untouched | dead constant gone; shift-truncation reliance now documented in `Unpack` |
| IN-02 | lint + `go test -race ./...` green, `go.mod`/`go.sum` untouched | all three new subtests fail against the pre-fix `clock.go` (observed `l = 2^48 = 281474976710656` and beyond — the exact wrap hazard) and pass with the fix |
| IN-03 | lint + `go test -race ./...` green, `go.mod`/`go.sum` untouched | mutation check: removing `Unpack`'s `counterMask` fails the property immediately (octet 6 `f2` vs `72`) |

Every commit ends with the full gate green: `go tool golangci-lint config
verify && go tool golangci-lint run ./...` plus `go test -race ./...` plus
`git diff --quiet go.mod go.sum`. The vendored strict lint config was not
weakened, no `//nolint` was introduced, and the tests stay in the external
`app_test` package with `t.Parallel()` and seeded generators (D-04).

Verification ran in the isolated gsd-code-fixer worktree (`rf-02-366146`) at
commit chain 329263e..c00c618 — identical to the `feat/02-engine-hlc-lww-store`
history after the fast-forward, so the numbers are reproducible from the
branch tip.

---

_Fixed: 2026-11-02T06:42:20Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
