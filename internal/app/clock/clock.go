// Package clock implements the sup-kv Hybrid Logical Clock (Kulkarni et
// al., 2014): node state (l, c) where l is Unix-millisecond wall-clock time
// and c counts events inside one millisecond. The wall clock enters through
// exactly one seam — the now func() [time.Time] constructor parameter of
// NewClock (D-02): production passes [time.Now], tests pass a controlled
// function driving the adversarial scenarios. A restart seeds the clock with
// the dump's full watermark so every post-restart reading dominates the dump
// (VERS-04), and readings saturate at the 48-bit timestamp ceiling instead
// of wrapping out of the version domain. The package depends on the
// standard library only.
package clock

import (
	"math"
	"sync"
	"time"
)

// Version-domain ceilings: counterMax is the highest value the 12-bit rand_a
// counter holds — past it the clock bumps l by one millisecond instead of
// wrapping — and maxL is the highest value the 48-bit unix_ts_ms holds — at
// it the clock saturates, repeating the top reading instead of bumping l
// past the domain, where the version package's PackVersion would silently
// wrap the timestamp to zero.
const (
	counterMask = 0xFFF // the 12-bit counter domain of the version layout.
	counterMax  = counterMask
	maxL        = 1<<48 - 1

	// driftMsMax is the widest millisecond divergence that still converts
	// to nanoseconds inside a [time.Duration]: at or beyond it the drift
	// signal saturates instead of wrapping. The unsaturated (l - ptMs) *
	// Millisecond product wraps int64 past ~292 years of divergence,
	// flipping the sign of exactly the worst drift the signal exists to
	// report (WR-01).
	driftMsMax = math.MaxInt64 / int64(time.Millisecond)
)

// Clock is a Hybrid Logical Clock (Kulkarni et al., 2014): node state (l, c)
// where l is Unix-millisecond wall-clock time and c counts events inside one
// millisecond. Clock is safe for concurrent use by multiple goroutines: every
// method takes the clock's own mutex, independent of any Store lock.
type Clock struct {
	mu        sync.Mutex
	l         int64
	c         uint16
	now       func() time.Time
	driftWarn time.Duration
}

// NewClock returns a clock driven by now; now must be non-nil. A restart
// from a dump passes the dump's FULL maximum reading (VERS-04): initialL
// the dump's maximum l, initialC the dump's maximum counter at that l. The
// first Tick then returns a reading strictly greater than (initialL,
// initialC) — (initialL, initialC+1), or (initialL+1, 0) through the
// existing counterMax overflow rule, or (pt, 0) when the wall clock
// already runs ahead — so every emitted version strictly dominates every
// version in that dump. Seeding initialL alone is insufficient whenever
// any dump version carries counter >= 1 at the watermark millisecond.
// initialL zero and initialC zero start from the wall clock on the first
// Tick. initialC values above the 12-bit counter range are masked, so the
// constructor is total over any uint16 input; initialL is clamped into the
// [0, maxL] version domain the same way — a watermark beyond the ceiling
// (a corrupted restart dump, an input the intended Unpack pipeline can
// never produce) saturates at maxL rather than persisting outside the
// domain, where the version package's PackVersion would silently wrap
// every subsequent write to the bottom of the stream, and a negative
// watermark reads as zero, starting from the wall clock. The version
// domain ends at
// (maxL, counterMax): a now seam beyond maxL — possible only with a clock
// lying past year ~8921 — saturates l at maxL, and once the counter also
// exhausts at the ceiling Tick and Update repeat the top reading (maxL,
// counterMax) instead of bumping l out of the domain, so writes at the
// exhausted domain may repeat versions: an honest, documented degradation,
// never a wrap to zero. driftWarn bounds the
// tolerated divergence before Tick and Update report a drift signal — Tick
// measures its own l against the wall clock, Update the received reading
// against it (OBSV-05); driftWarn less than or equal to zero disables the
// signal.
func NewClock(now func() time.Time, initialL int64, initialC uint16, driftWarn time.Duration) *Clock {
	return &Clock{l: min(max(initialL, 0), maxL), c: initialC & counterMask, now: now, driftWarn: driftWarn}
}

// Tick advances the clock for a local event and returns the new reading plus
// the drift signal observed at the event: zero within tolerance, positive
// when l already ran ahead of the wall clock beyond driftWarn, negative when
// the wall clock jumped forward past l beyond driftWarn.
func (c *Clock) Tick() (l int64, counter uint16, drift time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ptMs := c.now().UnixMilli()
	drift = driftBetween(c.l, ptMs, c.driftWarn)
	if ptMs > c.l {
		ptMs = min(ptMs, maxL) // a wall reading beyond the domain saturates.
		if ptMs > c.l {
			c.l = ptMs
			c.c = 0
			return c.l, c.c, drift
		}
	}
	if c.c == counterMax {
		if c.l == maxL {
			return c.l, c.c, drift // the domain is exhausted: repeat the top reading.
		}
		c.l++
		c.c = 0
		return c.l, c.c, drift
	}
	c.c++
	return c.l, c.c, drift
}

// Update advances the clock for a received event carrying (remoteL, remoteC)
// per the Kulkarni receive rule: l becomes the maximum of the local l, the
// remote l and the wall clock — saturated at the maxL domain ceiling (see
// NewClock) — and the counter is picked so the next local
// Tick beats the remote reading. The drift signal is the RECEIVED reading's
// lead against the wall clock (OBSV-05): positive when remoteL leads the
// wall beyond driftWarn, zero within tolerance including exactly driftWarn,
// and negative when the wall clock jumped past remoteL beyond driftWarn — a
// reading the core never counts. Tick keeps the separate local-versus-wall
// diagnostic. A remote counter above the 12-bit range is masked, so Update
// is total over every input.
func (c *Clock) Update(remoteL int64, remoteC uint16) (l int64, counter uint16, drift time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	remoteC &= counterMask
	ptMs := c.now().UnixMilli()
	drift = driftBetween(remoteL, ptMs, c.driftWarn)
	newL := max(c.l, remoteL, ptMs)
	newL = min(newL, maxL) // a wall or remote reading beyond the domain saturates.
	switch {
	case newL == c.l && c.l == remoteL:
		c.c = max(c.c, remoteC) + 1
	case newL == c.l:
		c.c++
	case newL == remoteL:
		c.c = remoteC + 1
	default:
		c.c = 0
	}
	if c.c > counterMax {
		if newL == maxL {
			c.l = newL
			c.c = counterMax
			return c.l, c.c, drift // the domain is exhausted: repeat the top reading.
		}
		c.l = newL + 1
		c.c = 0
		return c.l, c.c, drift
	}
	c.l = newL
	return c.l, c.c, drift
}

// driftBetween returns the signed divergence between the logical time l and
// the physical time ptMs: zero when within warn, positive milliseconds when
// l runs ahead beyond warn, negative when ptMs jumped past l beyond warn.
// The divergence is evaluated before any clamping, saturating at
// ±driftMsMax milliseconds — the widest value a Duration holds — so a
// divergence past ~292 years reads as a saturated reading of the correct
// sign, never a wrapped one. warn less than or equal to zero disables the
// signal.
func driftBetween(l, ptMs int64, warn time.Duration) time.Duration {
	if warn <= 0 {
		return 0
	}
	// Clamp both operands into the representable band first: the raw
	// difference of arbitrary int64 operands can itself wrap, and even a
	// difference that fits overflows below when scaled to nanoseconds.
	// Clamping never distorts a representable divergence — both operands
	// saturate to the same bound, so only already-unrepresentable readings
	// change.
	l = clampDriftRange(l)
	ptMs = clampDriftRange(ptMs)
	drift := time.Duration(clampDriftRange(l-ptMs)) * time.Millisecond
	if drift > warn || drift < -warn {
		return drift
	}
	return 0
}

// clampDriftRange saturates v into the ±driftMsMax band.
func clampDriftRange(v int64) int64 {
	return max(-driftMsMax, min(driftMsMax, v))
}
