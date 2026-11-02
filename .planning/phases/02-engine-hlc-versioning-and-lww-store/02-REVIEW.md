---
phase: 02-engine-hlc-versioning-and-lww-store
reviewed: 2026-11-02T02:46:49Z
depth: standard
files_reviewed: 11
files_reviewed_list:
  - internal/app/errors.go
  - internal/app/version.go
  - internal/app/clock.go
  - internal/app/store.go
  - internal/app/doc.go
  - internal/app/store_test.go
  - internal/app/clock_test.go
  - internal/app/props_test.go
  - internal/app/version_test.go
  - internal/app/hlc_prop_test.go
  - internal/app/merge_test.go
findings:
  critical: 0
  warning: 0
  info: 3
  total: 3
status: issues_found
---

# Phase 02: Code Review Report

**Reviewed:** 2026-11-02T02:46:49Z (incremental pass over the 02-04 gap-closure delta, commits a5a78a5..ba54587; first pass 2026-11-02T01:51:30Z)
**Depth:** standard
**Files Reviewed:** 11 (plus the removal of `internal/app/app_test.go`, commit 148fba4). Delta pass scope: `clock.go`, `store.go`, `clock_test.go`, `store_test.go`, `hlc_prop_test.go`.
**Status:** issues_found (3 open Info findings; the one Warning is fixed)

## Summary

The pure decision core is, on the whole, sound. Verified against the source (not
just the tests): the `PackVersion`/`Unpack` bit layout matches RFC 9562 (the
fixed vector `0000000000017123aaf3400000000000` reproduces by hand; the variant
bit shifts out of the `uint16` on unpack, and byte-wise `Compare` is provably
monotone in `(l, counter, nodeID)`); all four Kulkarni receive branches in
`Clock.Update` plus both overflow paths are strictly monotone; the
post-receive invariant holds — after `Update(remoteL, remoteC)` the clock state
strictly exceeds `(remoteL, remoteC)` in `(l, c)` order, so every later local
`Set`/`Delete` beats every previously merged record; `Store.join` is a per-key
max join over the full 16-byte version, so commutativity, associativity,
idempotence and no-resurrection hold; `s.mu` is held across every clock advance
(version order == commit order) and `clock.mu` is a leaf lock, so there is no
lock inversion; stored slices are never mutated in place, making the clone
after `RUnlock` in `Get` safe; all imports satisfy `sup-kv-app-no-io`.
`go test -race ./internal/app/` passes. The deleted `app_test.go` leaves no
dangling references (its helpers live on in `store_test.go`).

**Incremental pass (02-04 delta, +237/−33 over `internal/app/`).** The delta
seeds the restart seam with the dump's full watermark: `NewClock` gained
`initialC uint16` masked with `counterMask` (`clock.go:39-41`), `Config.InitialC`
propagates at `NewStore` (`store.go:31-32`, `store.go:75`); `Tick`/`Update`
bodies are byte-identical to the first-pass review. Adversarial trace of the
delta found no defects:

- The documented dominance claim holds branch-by-branch: `pt > initialL` →
  `(pt, 0)`; `pt <= initialL` with seed `< counterMax` → `(initialL,
  initialC+1)`; seed `== counterMax` → `(initialL+1, 0)` through the existing
  overflow rule — each strictly after `(initialL, initialC)`, and since
  byte-wise `Compare` orders `(l, counter, nodeID)`, the `(l, c)` pair alone
  is informationally complete for dominance (no nodeID seed is needed or
  possible: packed versions physically cannot carry a counter above 4095).
- The masking convention (`initialC & counterMask`) mirrors `Update`'s existing
  `remoteC &= counterMask` totality rule; a dump-derived seed can never
  legitimately exceed 12 bits, and over-seeding could only strengthen
  dominance, never weaken it.
- `Update` before the first post-restart `Tick` (a Merge arriving right after
  restart) stays monotone and dominant over the seed, including the
  `max(c0, remoteC)+1` overflow path.
- `Config.normalized()` passes `InitialL`/`InitialC` through untouched —
  correct, since `uint16` cannot be negative and a zero counter seed
  degenerates safely (`(T,1)` still beats a `(T,0)` watermark).
- Regression strength verified against the pre-fix code: each new row fails
  under the old constructor — `TestStore_restartDominance` variants A/B would
  observe zero conflicts and stale/reverted data through the equal-version
  no-op, so both WR-01 consequences are pinned dead. The watermark property
  draws `watermarkC` from 1..4095 with a wall-clock seed per run (excluding 0
  is correct targeting: counter 0 was never hazardous) and initializes `prev`
  at the full watermark, so the very first tick must strictly dominate it.
- `go test -race -count=1 ./internal/app/` green, plus 8 randomized runs of
  the three restart suites — all green.

No new findings in the delta. **WR-01 is fixed** (see below); counts in the
frontmatter reflect open findings. IN-01/IN-02/IN-03 remain open — the delta
did not touch them (note: the clock.go constructor region grew by nine lines,
so IN-02's cited overflow paths now sit at `clock.go:57-60` and
`clock.go:89-93`; the finding text is unchanged).

## Warnings

### WR-01 [FIXED by 02-04]: Restart seam seeds only `l` — post-restart writes can collide with or lose to pre-restart versions

**Status:** FIXED in gap-closure 02-04 (commits a5a78a5, 68e4b62, b4b5397).
No longer counted in the open findings above.

**Fix evidence (verified this pass against the source):**
`NewClock(now, initialL, initialC, driftWarn)` now seeds `c = initialC &
counterMask` (`internal/app/clock.go:39-41`) and `Config.InitialC` propagates
through `NewStore` (`internal/app/store.go:31-32`, `internal/app/store.go:75`),
with both doc contracts rewritten to demand the dump's FULL watermark and to
state that seeding `initialL` alone is insufficient. The original hazard — the
same-ms restart stamping `(T, 1)` identical to a dumped `(T, 1, N)` (variant A:
silent revert on peers) and post-restart writes landing below a `(T, 5, N)`
watermark (variant B: acknowledged writes reverting on merge) — is now
unreachable through the seam: the first `Tick` is strictly after `(initialL,
initialC)` on every branch. Regressions: `TestClock_restartFromDumpMax`
counter-seeded rows (`internal/app/clock_test.go:300-352` — seed+1, counterMax
overflow bump, wall-ahead adoption, out-of-range masking),
`TestClock_restartProperty` now models the full watermark
`(dumpMax, watermarkC ∈ 1..4095, ownNodeID)` (`internal/app/hlc_prop_test.go:236-272`),
and `TestStore_restartDominance` (`internal/app/store_test.go:577-698`) encodes
both WR-01 consequences as dead at the store level with exact Conflict
assertions. Residual exposure is IN-02's open 48-bit ceiling (a dump watermark
at `l = 2^48-1` with counter 4095 still wraps on the overflow bump), which
subsumes this seam's last theoretical corner.

Original finding, retained for the record:

**File:** `internal/app/clock.go:24-31` (and the mirrored contract at `internal/app/store.go:20-28`, `Config.InitialL`)
**Issue:** `NewClock(now, initialL, driftWarn)` documented VERS-04 restart as
"pass the dump's maximum l". That was not enough information to preserve the
package's own stated laws ("versions unique per write"; a local write after
seeing a record beats it):

1. **Collision trace.** A node writes twice in one millisecond: versions
   `(T, 0, N)` and `(T, 1, N)` land in the dump, `dumpMax = T`. Restart with
   the wall clock at or below `T` (same-ms restart or clock regression —
   exactly the adversarial class this package targets). `NewClock(now, T, …)`,
   first `Tick` hits the `pt <= l` path and returns `(T, 1)`
   (`clock.go:48-54`), identical to the dumped version. Phase 3 will reload
   the dump with the same node ID, so a fresh `Set` on that key stamps
   `(T, 1, N)` with *different data*. `write` overwrites the local map
   unconditionally (`store.go:141-152`) while `join` treats the equal version
   as a full no-op (`store.go:219-221`) — so the local node keeps the new data,
   every peer keeps the dumped data, and no later merge of either copy ever
   reconciles them: a permanent per-key split on that key. This also silently
   breaks the `merge(a,a)`-idempotence premise that equal versions carry equal
   payloads.
2. **Subordinate-write trace.** Dump max version `(T, 5, N)`; restart seeds
   `l = T`; the first post-restart writes produce `(T, 1)`, `(T, 2)`, … all
   below `(T, 5)`. Each such write is acknowledged locally but is strictly
   older than the reloaded record, so on merge it is rejected and the store
   converges back to the pre-restart value — acknowledged writes silently
   revert.

Not reachable through any code path shipped in this phase (nothing loads a
dump yet), hence Warning rather than Critical — but the seam and its doc
contract are this phase's deliverable, and `TestClock_restartProperty`
(`hlc_prop_test.go:230-265`) actively cemented the weak contract by modelling
the watermark only as `(dumpMax, 0, maxNodeID)`, so the hazardous cases were
untested and the insufficiency was invisible to the suite.

## Info

### IN-01: Unused constant `varPayloadMask`

**File:** `internal/app/version.go:21`
**Issue:** `varPayloadMask = 0x3F` is defined ("the nodeID bits kept from
octet 8") but referenced nowhere — `PackVersion` sets those bits via
`PutUint16`/`|= uuidVarBits` and `Unpack` drops the variant bit by uint16
shift truncation. Dead code; the linter does not flag unused constants.
**Fix:** Delete the constant, or use it where the bits are actually kept to
make the masking explicit (e.g. document that `Unpack` relies on shift
truncation instead of masking with it).

### IN-02: No guard at the 48-bit `l` ceiling — counter overflow wraps the version stream to zero

**File:** `internal/app/clock.go:49` and `internal/app/clock.go:81` (masked away at `internal/app/version.go:35`; after the 02-04 delta these overflow paths sit at `clock.go:57-60` and `clock.go:89-93`)
**Issue:** When `l` reaches `2^48-1` with the counter at `counterMax`, both
`Tick` and `Update` execute `c.l++`, producing `l = 2^48`. `PackVersion` then
applies `l & tsMask`, silently wrapping the timestamp to `0` — the version
stream restarts from the bottom of the domain, breaking monotonicity and
uniqueness (equal/predecessor versions on merge → the same divergence class as
WR-01). Unreachable with a truthful wall clock before ~year 8921, but the D-02
seam explicitly admits an injected `now`, and `randVersion`
(`props_test.go:53-59`) already generates readings at the ceiling. The 02-04
restart seed adds one more route into this corner (a dump watermark at the
ceiling with counter 4095), but the finding is unchanged in kind.
**Fix:** Either clamp (skip the bump, reuse the reading — acceptable since
`Tick` may return a repeated `(l, c)` only if callers tolerate it) or document
the invariant "l < 2^48-1 is a caller precondition on `now`" in `NewClock`,
and assert it in tests via the injected seam.

### IN-03: `TestVersion_unpackTotal` range assertions are tautologies

**File:** `internal/app/version_test.go:173-197`
**Issue:** The assertions `l < 0 || l > maxL`, `counter > testCounterMax`,
`nodeID > maxNodeID`, `cmp < -1 || cmp > 1` and the antisymmetry check are
structurally impossible to fail: `l` is an unsigned 48-bit shift of a `uint64`
(`version.go:48`), `counter` is masked with `0xFFF` at `version.go:49`,
`nodeID` is a `uint16` compared against `0xFFFF`, and `bytes.Compare` only
returns `-1/0/1` and is antisymmetric by construction. The test's only real
power is detecting panics over raw bit patterns — which is its stated purpose,
but the four dead checks present it as verifying more than it can.
**Fix:** Replace the tautologies with a substantive canonicalization property:
for every raw 16-byte `v`, `PackVersion` of `v.Unpack()` must equal `v` with
the version/variant/filler bits normalized (octet 6 high nibble `0x7`, octet 8
top bits `0b10`, bit `0x40` of octet 8 cleared, octets 11-15 and low 6 bits of
octet 10 zero). That property can actually fail and is the meaningful
"Unpack is total and lossless over its kept bits" claim.

---

_Reviewed: 2026-11-02T02:46:49Z (incremental 02-04 delta pass; first pass 2026-11-02T01:51:30Z)_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
