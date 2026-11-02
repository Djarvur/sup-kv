---
phase: 01-ci-bootstrap-and-repo-skeleton
reviewed: 2026-11-02T16:03:17Z
depth: standard
files_reviewed: 13
files_reviewed_list:
  - AGENTS.md
  - internal/app/clock/clock.go
  - internal/app/clock/clock_test.go
  - internal/app/clock/hlc_prop_test.go
  - internal/app/internal/testutil/gen.go
  - internal/app/internal/testutil/rng.go
  - internal/app/storage/doc.go
  - internal/app/storage/errors.go
  - internal/app/storage/merge_test.go
  - internal/app/storage/store.go
  - internal/app/storage/store_test.go
  - internal/app/version/version.go
  - internal/app/version/version_test.go
findings:
  critical: 0
  warning: 2
  info: 1
  total: 3
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-11-02T16:03:17Z
**Depth:** standard
**Files Reviewed:** 13
**Status:** issues_found

## Summary

Reviewed the current on-disk state of the AGENTS.md conventions update and the
split of `internal/app` into the `version`, `clock` and `storage` packages plus
the shared `internal/app/internal/testutil` generators (the delta since
diff_base 0ca46c9 that touched phase-01's covered files).

The core is well-built: the HLC Tick/Update state machines were traced through
every Kulkarni branch including counter overflow, ceiling saturation and
adversarial inputs (negative/over-domain remote readings, out-of-range
counters) and are correct — every returned reading is strictly greater than
the pre-state, and saturation pins at `(maxL, counterMax)` instead of wrapping.
The RFC 9562 pack/unpack round-trip was verified against the layout vector,
the 12/14-bit masking, and the truncation trick that drops the variant bits;
byte-wise `Compare` order is sound. The LWW join laws, tombstone asymmetry and
defensive copying in `storage` match the documented contract. Machine gates
verified on this tree: `go test -race ./internal/...` passes,
`go tool golangci-lint run ./internal/...` is clean, the depguard rules match
AGENTS.md, the `# Origin:` header is in place, and all `//nolint` directives
in scope name their linter and carry justifications per deviation #3.

Two warnings survive the adversarial pass, both in `clock.go` and both
involving the 48-bit domain ceiling — the code carefully clamps every input
into the domain (initialC, remoteC, ptMs, newL) except one, and the drift
signal arithmetic overflows precisely in the saturated state it exists to
report. Both were confirmed by execution, not just reading.

## Warnings

### WR-01: driftBetween overflows int64 at the domain ceiling, corrupting the drift signal exactly when the clock is pathological

**File:** `internal/app/clock/clock.go:144`
**Issue:** `d := (l - ptMs) * int64(time.Millisecond)` overflows when
`|l - ptMs|` exceeds ~9.22e12 ms (~292 years). That state is reachable
through the public API without any contract violation: `remoteL = maxL`
(2^48-1) is a valid in-domain RFC 9562 timestamp, so a single
`Store.Merge` of a peer record carrying `l = maxL` (or a clock seam past
year ~8921) ratchets `l` to `maxL` permanently via `Update`'s documented
saturation. With `l = maxL` and a realistic wall clock (2024-11), the true
divergence is 279,744,976,710,655,000,000 ns (~+8871 years), but the int64
product wraps to 3,043,815,605,011,725,760 ns (~+96 years) — wrong
magnitude, and for other (l, ptMs) pairs the wrapped value flips sign or
lands inside tolerance. The drift signal (VERS-05) is the operator's
mechanism for detecting a stuck/lying clock; after the wrap it reports
garbage in precisely the state that needs flagging. Verified by running
the arithmetic: true delta ~8871 years, wrapped int64 delta ~96 years.
Monotonicity and stored versions are unaffected (drift never feeds back
into clock state), which is why this is a Warning, not a Critical.
**Fix:** Compare in milliseconds before scaling, and saturate instead of
wrapping:

```go
func driftBetween(l, ptMs int64, warn time.Duration) time.Duration {
	if warn <= 0 {
		return 0
	}
	const maxDeltaMs = int64(math.MaxInt64 / int64(time.Millisecond)) // ±~292 years.
	d := l - ptMs
	if d > maxDeltaMs {
		return time.Duration(maxDeltaMs) * time.Millisecond // saturate ahead, never wrap
	}
	if d < -maxDeltaMs {
		return time.Duration(-maxDeltaMs) * time.Millisecond // saturate behind
	}
	scaled := d * int64(time.Millisecond)
	if scaled > int64(warn) || scaled < -int64(warn) {
		return time.Duration(scaled)
	}
	return 0
}
```

### WR-02: NewClock never clamps initialL to the 48-bit domain — the one unclamped input silently defeats the documented wrap protection

**File:** `internal/app/clock/clock.go:63-65`
**Issue:** `NewClock` masks `initialC` (`& counterMask`), `Update` masks
`remoteC`, clamps `newL` with `min(newL, maxL)`, and `Tick` clamps the wall
reading with `min(ptMs, maxL)` — but `initialL` is stored raw with no clamp
and, unlike `PackVersion`, its doc comment states no range constraint on the
caller. For `initialL > maxL`: `Tick`'s saturation guard `c.l == maxL`
(clock.go:85) never fires because `c.l > maxL`, `l` only grows, and every
stamped version reaches `PackVersion` with `l` beyond the domain, where
`l&tsMask` silently wraps it to the bottom — verified by execution:
`initialL = maxL+5` ticks to `l = maxL+6`, and `PackVersion` packs `l = 6`.
That is exactly the "silently wrap the timestamp" failure the package doc
(clock.go:20-23) promises cannot happen ("readings saturate at the 48-bit
timestamp ceiling instead of wrapping out of the version domain ... never a
wrap to zero"): a node restarted with a corrupted out-of-range watermark
would issue low versions that lose every future LWW merge against its own
prior writes. Not reachable through the intended dump pipeline (`Unpack` is
masking, so dump-derived watermarks are always ≤ maxL), hence Warning
rather than Critical — but the package's stated totality philosophy makes
this input the inconsistent one.
**Fix:** Clamp in the constructor, matching the treatment of every other
input:

```go
func NewClock(now func() time.Time, initialL int64, initialC uint16, driftWarn time.Duration) *Clock {
	return &Clock{l: min(initialL, maxL), c: initialC & counterMask, now: now, driftWarn: driftWarn}
}
```

## Info

### IN-01: NewRNG doc comment overstates the printed-seed contract ("on every run")

**File:** `internal/app/internal/testutil/rng.go:38-40`
**Issue:** The comment says the seed "is printed with the exact
`property seed:` prefix on every run", but it is emitted via `t.Logf`,
which `go test` suppresses for passing tests unless `-v` is given. The
D-04 replay contract still holds where it matters — the output appears on
failure, and every failure message says "replay with the property seed
above" — so this is a doc-accuracy nit, not a broken contract.
**Fix:** Reword to "printed with the exact `property seed:` prefix on every
failing run (or under `-v`)".

---

_Reviewed: 2026-11-02T16:03:17Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
