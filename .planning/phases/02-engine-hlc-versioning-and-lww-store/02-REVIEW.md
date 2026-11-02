---
phase: 02-engine-hlc-versioning-and-lww-store
reviewed: 2026-11-02T01:51:30Z
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
  warning: 1
  info: 3
  total: 4
status: issues_found
---

# Phase 02: Code Review Report

**Reviewed:** 2026-11-02T01:51:30Z
**Depth:** standard
**Files Reviewed:** 11 (plus the removal of `internal/app/app_test.go`, commit 148fba4)
**Status:** issues_found

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

What lint cannot see turned up one real contract deficiency and three smaller
defects, listed below. No Critical issues: nothing in this phase's shipped
code path loses data or breaks convergence — the Warning is a seam that Phase 3
(VERS-04 dump/restart) will build on and that is insufficient as documented.

## Warnings

### WR-01: Restart seam seeds only `l` — post-restart writes can collide with or lose to pre-restart versions

**File:** `internal/app/clock.go:24-31` (and the mirrored contract at `internal/app/store.go:20-28`, `Config.InitialL`)
**Issue:** `NewClock(now, initialL, driftWarn)` documents VERS-04 restart as
"pass the dump's maximum l". That is not enough information to preserve the
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
(`hlc_prop_test.go:230-265`) actively cements the weak contract by modelling
the watermark only as `(dumpMax, 0, maxNodeID)`, so the hazardous cases are
untested and the insufficiency is invisible to the suite.
**Fix:** Seed the clock from the dump's maximum *full* reading, e.g. add an
`initialC uint16` parameter (`NewClock(now, initialL, initialC, driftWarn)`
with first Tick then returning at least `(initialL, initialC+1)` and the
existing overflow-to-`l+1` rule), or keep the signature and document + enforce
that restarts start at `dumpMax + 1` so no post-restart version can equal or
precede a dumped one. Extend `TestClock_restartProperty` with a dump whose
watermark is `(dumpMax, c>0, ownNodeID)`.

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

**File:** `internal/app/clock.go:49` and `internal/app/clock.go:81` (masked away at `internal/app/version.go:35`)
**Issue:** When `l` reaches `2^48-1` with the counter at `counterMax`, both
`Tick` and `Update` execute `c.l++`, producing `l = 2^48`. `PackVersion` then
applies `l & tsMask`, silently wrapping the timestamp to `0` — the version
stream restarts from the bottom of the domain, breaking monotonicity and
uniqueness (equal/predecessor versions on merge → the same divergence class as
WR-01). Unreachable with a truthful wall clock before ~year 8921, but the D-02
seam explicitly admits an injected `now`, and `randVersion`
(`props_test.go:53-59`) already generates readings at the ceiling.
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

_Reviewed: 2026-11-02T01:51:30Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
