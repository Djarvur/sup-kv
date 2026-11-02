package app

import (
	"sync"
	"time"
)

// counterMax is the highest value the 12-bit rand_a counter holds; past it
// the clock bumps l by one millisecond instead of wrapping.
const counterMax = counterMask

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
// constructor is total over any uint16 input. driftWarn bounds the
// tolerated divergence between l and the wall clock before Tick and Update
// report a drift signal; driftWarn less than or equal to zero disables the
// signal.
func NewClock(now func() time.Time, initialL int64, initialC uint16, driftWarn time.Duration) *Clock {
	return &Clock{l: initialL, c: initialC & counterMask, now: now, driftWarn: driftWarn}
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
		c.l = ptMs
		c.c = 0
		return c.l, c.c, drift
	}
	if c.c == counterMax {
		c.l++
		c.c = 0
		return c.l, c.c, drift
	}
	c.c++
	return c.l, c.c, drift
}

// Update advances the clock for a received event carrying (remoteL, remoteC)
// per the Kulkarni receive rule: l becomes the maximum of the local l, the
// remote l and the wall clock, and the counter is picked so the next local
// Tick beats the remote reading. The drift signal follows the Tick
// convention. A remote counter above the 12-bit range is masked, so Update
// is total over every input.
func (c *Clock) Update(remoteL int64, remoteC uint16) (l int64, counter uint16, drift time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	remoteC &= counterMask
	ptMs := c.now().UnixMilli()
	drift = driftBetween(c.l, ptMs, c.driftWarn)
	newL := max(c.l, remoteL, ptMs)
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
// The divergence is evaluated before any clamping. warn less than or equal
// to zero disables the signal.
func driftBetween(l, ptMs int64, warn time.Duration) time.Duration {
	if warn <= 0 {
		return 0
	}
	d := (l - ptMs) * int64(time.Millisecond)
	if d > int64(warn) || d < -int64(warn) {
		return time.Duration(d)
	}
	return 0
}
