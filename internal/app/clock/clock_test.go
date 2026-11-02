package clock_test

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/internal/testutil"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// Clock-test constants: the drift tolerance under test, the 12-bit rand_a
// counter ceiling from RFC 9562, the fixed wall-clock millisecond and node
// ID of the shared fixtures, and the version-domain ceilings.
const (
	testDriftWarn  = time.Minute
	testCounterMax = 4095
	testNowMs      = int64(1730000000000)
	testNodeID     = uint16(0xABCD)
	maxL           = 1<<48 - 1
	maxNodeID      = 0xFFFF
)

// reading is one (l, counter) pair returned by Tick or Update.
type reading struct {
	l       int64
	counter uint16
}

// after reports whether r sorts strictly after other in (l, counter) order.
func (r reading) after(other reading) bool {
	return r.l > other.l || (r.l == other.l && r.counter > other.counter)
}

// fixedNow returns a now func parked at the given Unix millisecond, built
// with time.UnixMilli so no monotonic reading is involved (D-02).
func fixedNow(ms int64) func() time.Time {
	return func() time.Time { return time.UnixMilli(ms) }
}

// seqNow returns a now func yielding the given Unix milliseconds in order,
// sticking at the last value once the sequence is exhausted; every value is
// built with time.UnixMilli so no monotonic reading is involved (D-02,
// Pitfall 8).
func seqNow(ms ...int64) func() time.Time {
	var i int
	return func() time.Time {
		t := time.UnixMilli(ms[min(i, len(ms)-1)])
		i++
		return t
	}
}

// ticks returns n repetitions of ms, the now sequence that builds a counter
// of n at l = ms.
func ticks(ms int64, n int) []int64 {
	out := make([]int64, n)
	for i := range out {
		out[i] = ms
	}
	return out
}

// nextWallTime advances the adversarial wall-clock walk: one in ten a
// backward jump of up to four hours in milliseconds, one in ten a forward
// jump of the same scale, two in ten stuck at the same millisecond, else a
// 0..2 ms step (research "Code Examples", the TEST-02 chaos input).
func nextWallTime(rng *rand.Rand, prev int64) int64 {
	switch rng.IntN(10) {
	case 0:
		return prev - rng.Int64N(4*60*60*1000)
	case 1:
		return prev + rng.Int64N(4*60*60*1000)
	case 2, 3:
		return prev
	default:
		return prev + rng.Int64N(3)
	}
}

// TestClock_tick covers the Kulkarni local-event rule per table: the wall
// clock ahead of l adopts the wall and resets the counter, the wall equal to
// or behind l increments the counter, a first Tick on a clock whose InitialL
// is ahead of now starts at (InitialL, 1), and a first Tick with InitialL
// zero starts from the wall clock (VERS-03).
func TestClock_tick(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		initialL int64
		nows     []int64
		want     []reading
	}{
		{
			name:     "wall ahead of l adopts the wall and resets the counter",
			initialL: testNowMs,
			nows:     []int64{testNowMs + 5},
			want:     []reading{{l: testNowMs + 5, counter: 0}},
		},
		{
			name:     "wall equal to l increments the counter",
			initialL: testNowMs,
			nows:     []int64{testNowMs},
			want:     []reading{{l: testNowMs, counter: 1}},
		},
		{
			name:     "wall behind l increments the counter",
			initialL: testNowMs,
			nows:     []int64{testNowMs - 3600_000},
			want:     []reading{{l: testNowMs, counter: 1}},
		},
		{
			name:     "first tick with InitialL ahead of now returns (InitialL, 1)",
			initialL: testNowMs + 3600_000,
			nows:     []int64{testNowMs, testNowMs},
			want: []reading{
				{l: testNowMs + 3600_000, counter: 1},
				{l: testNowMs + 3600_000, counter: 2},
			},
		},
		{
			name:     "first tick with InitialL zero adopts the wall clock",
			initialL: 0,
			nows:     []int64{testNowMs},
			want:     []reading{{l: testNowMs, counter: 0}},
		},
		{
			name:     "second tick inside one millisecond increments the counter",
			initialL: 0,
			nows:     []int64{testNowMs, testNowMs},
			want:     []reading{{l: testNowMs, counter: 0}, {l: testNowMs, counter: 1}},
		},
		{
			name:     "wall catching up after being behind adopts the wall and resets",
			initialL: testNowMs,
			nows:     []int64{testNowMs - 5, testNowMs + 1},
			want:     []reading{{l: testNowMs, counter: 1}, {l: testNowMs + 1, counter: 0}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			clk := clock.NewClock(seqNow(tt.nows...), tt.initialL, 0, testDriftWarn)
			for i, want := range tt.want {
				l, counter, _ := clk.Tick()
				got := reading{l: l, counter: counter}
				if got != want {
					t.Errorf("tick %d = (%d, %d), want (%d, %d)",
						i+1, got.l, got.counter, want.l, want.counter)
				}
			}
		})
	}
}

// TestClock_update covers all four Kulkarni receive branches as named
// subtests, each under a wall clock below or equal to the candidates (and,
// for the default branch, above both): l == old == msg gives c = max(c, cm)+1,
// l == old gives c+1, l == msg gives cm+1, otherwise c resets to 0. Every
// branch result is strictly greater than the pre-state in (l, counter) order.
func TestClock_update(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		initialL int64
		preTicks int
		updNow   int64
		remoteL  int64
		remoteC  uint16
		want     reading
	}{
		{
			name:     "l==old==msg wall below gives c = max(c, cm)+1",
			initialL: testNowMs, preTicks: 5, updNow: testNowMs - 10,
			remoteL: testNowMs, remoteC: 3,
			want: reading{l: testNowMs, counter: 6},
		},
		{
			name:     "l==old==msg wall equal gives c = max(c, cm)+1",
			initialL: testNowMs, preTicks: 5, updNow: testNowMs,
			remoteL: testNowMs, remoteC: 3,
			want: reading{l: testNowMs, counter: 6},
		},
		{
			name:     "l==old==msg remote counter above local max picks the remote",
			initialL: testNowMs, preTicks: 2, updNow: testNowMs,
			remoteL: testNowMs, remoteC: 7,
			want: reading{l: testNowMs, counter: 8},
		},
		{
			name:     "l==old wall below gives c+1",
			initialL: testNowMs, preTicks: 5, updNow: testNowMs - 10,
			remoteL: testNowMs - 50, remoteC: 9,
			want: reading{l: testNowMs, counter: 6},
		},
		{
			name:     "l==old wall equal gives c+1",
			initialL: testNowMs, preTicks: 5, updNow: testNowMs,
			remoteL: testNowMs - 50, remoteC: 9,
			want: reading{l: testNowMs, counter: 6},
		},
		{
			name:     "l==msg wall below gives c = cm+1",
			initialL: testNowMs - 100, preTicks: 5, updNow: testNowMs - 10,
			remoteL: testNowMs, remoteC: 7,
			want: reading{l: testNowMs, counter: 8},
		},
		{
			name:     "l==msg wall equal gives c = cm+1",
			initialL: testNowMs - 100, preTicks: 5, updNow: testNowMs,
			remoteL: testNowMs, remoteC: 7,
			want: reading{l: testNowMs, counter: 8},
		},
		{
			name:     "wall above both jumps l to the wall and resets the counter",
			initialL: testNowMs, preTicks: 5, updNow: testNowMs + 50,
			remoteL: testNowMs + 10, remoteC: 3,
			want: reading{l: testNowMs + 50, counter: 0},
		},
		{
			name:     "remote counter above 12 bits is masked and overflows to l+1",
			initialL: testNowMs, preTicks: 1, updNow: testNowMs,
			remoteL: testNowMs, remoteC: 0xFFFF,
			want: reading{l: testNowMs + 1, counter: 0},
		},
		{
			name:     "l==msg at the counter ceiling overflows to l+1",
			initialL: testNowMs - 10, preTicks: 0, updNow: testNowMs - 5,
			remoteL: testNowMs, remoteC: testCounterMax,
			want: reading{l: testNowMs + 1, counter: 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			nows := ticks(tt.initialL, tt.preTicks)
			nows = append(nows, tt.updNow)
			clk := clock.NewClock(seqNow(nows...), tt.initialL, 0, testDriftWarn)
			for range tt.preTicks {
				clk.Tick()
			}
			pre := reading{l: tt.initialL, counter: uint16(tt.preTicks)}
			l, counter, _ := clk.Update(tt.remoteL, tt.remoteC)
			got := reading{l: l, counter: counter}
			if got != tt.want {
				t.Errorf("Update(%d, %d) = (%d, %d), want (%d, %d)",
					tt.remoteL, tt.remoteC, got.l, got.counter, tt.want.l, tt.want.counter)
			}
			if !got.after(pre) {
				t.Errorf("Update(%d, %d) = (%d, %d), want strictly after the pre-state (%d, %d)",
					tt.remoteL, tt.remoteC, got.l, got.counter, pre.l, pre.counter)
			}
		})
	}
}

// TestClock_counterOverflowBoundary covers the counter-overflow rule under a
// stuck clock: Ticks 1 through 4096 fill the counters 0 through 4095 at an
// unchanged l with no repetition, and Tick 4097 bumps l by one and resets the
// counter to 0 (VERS-03, TEST-02 boundary).
func TestClock_counterOverflowBoundary(t *testing.T) {
	t.Parallel()
	clk := clock.NewClock(fixedNow(testNowMs), 0, 0, testDriftWarn)
	seen := make(map[reading]bool, testCounterMax+2)
	for i := 1; i <= testCounterMax+1; i++ {
		l, counter, _ := clk.Tick()
		got := reading{l: l, counter: counter}
		want := reading{l: testNowMs, counter: uint16(i - 1)}
		if got != want {
			t.Fatalf("tick %d = (%d, %d), want (%d, %d)", i, got.l, got.counter, want.l, want.counter)
		}
		if seen[got] {
			t.Fatalf("tick %d repeated reading (%d, %d)", i, got.l, got.counter)
		}
		seen[got] = true
	}
	l, counter, _ := clk.Tick()
	if l != testNowMs+1 || counter != 0 {
		t.Errorf("tick %d = (%d, %d), want (%d, 0)", testCounterMax+2, l, counter, testNowMs+1)
	}
}

// TestClock_ceilingSaturation covers the 48-bit domain ceiling (VERS-02):
// seeded at maxL with the adversarial now seam beyond the ceiling, the
// clock saturates — every Tick and Update reading holds l at maxL, no
// packed version ever reads below its predecessor (the old code bumped l to
// 2^48 and PackVersion's tsMask silently wrapped the timestamp to zero),
// and once the counter passes counterMax the top reading (maxL, counterMax)
// repeats: a documented degradation, never a restart from the bottom of
// the domain.
func TestClock_ceilingSaturation(t *testing.T) {
	t.Parallel()
	t.Run("the saturated Tick bump repeats the top reading", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewClock(fixedNow(maxL+5), maxL, testCounterMax, testDriftWarn)
		got := reading{}
		got.l, got.counter, _ = clk.Tick()
		want := reading{l: maxL, counter: testCounterMax}
		if got != want {
			t.Errorf("tick at the ceiling = (%d, %d), want the repeated top reading (%d, %d)",
				got.l, got.counter, want.l, want.counter)
		}
	})
	t.Run("the saturated Update overflow pins the top reading", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewClock(fixedNow(maxL), maxL, 0, testDriftWarn)
		got := reading{}
		got.l, got.counter, _ = clk.Update(maxL, testCounterMax)
		want := reading{l: maxL, counter: testCounterMax}
		if got != want {
			t.Errorf("Update at the ceiling = (%d, %d), want the pinned top reading (%d, %d)",
				got.l, got.counter, want.l, want.counter)
		}
	})
	t.Run("an adversarial walk beyond the ceiling never wraps", func(t *testing.T) {
		t.Parallel()
		rng := testutil.NewRNG(t)
		nowMs := maxL + 1 + rng.Int64N(1_000_000)
		now := func() time.Time { return time.UnixMilli(nowMs) }
		clk := clock.NewClock(now, maxL, 0, testDriftWarn)
		prev := version.PackVersion(maxL, 0, testNodeID)
		for i := range testCounterMax + 10 {
			nowMs = maxL + 1 + rng.Int64N(1_000_000)
			got := reading{}
			if rng.IntN(2) == 0 {
				got.l, got.counter, _ = clk.Tick()
			} else {
				got.l, got.counter, _ = clk.Update(maxL, uint16(rng.IntN(testCounterMax+1)))
			}
			if got.l != maxL {
				t.Fatalf("step %d l = %d, want the saturated ceiling %d (replay with the property seed above)",
					i+1, got.l, maxL)
			}
			ver := version.PackVersion(got.l, got.counter, testNodeID)
			if ver.Compare(prev) < 0 {
				t.Fatalf("step %d version %x reads below its predecessor %x — the stream wrapped to zero (replay with the property seed above)",
					i+1, ver[:], prev[:])
			}
			prev = ver
		}
	})
}

// TestClock_restartFromDumpMax covers the restart seam (VERS-04): a clock
// whose InitialL is far ahead of a wall clock that then jumps backward hours
// further never drops below InitialL and strictly increases, a clock with
// InitialL zero starts from the wall clock, and — the counter-seeded rows —
// a clock seeded with the dump's full watermark (initialL plus initialC)
// emits a first Tick strictly after (initialL, initialC): (initialL,
// initialC+1) inside the millisecond, the counterMax overflow bump to
// (initialL+1, 0), adoption of the wall clock when it already runs ahead,
// and out-of-range initialC masked to the 12-bit domain.
func TestClock_restartFromDumpMax(t *testing.T) {
	t.Parallel()
	t.Run("InitialL ahead of a backward-jumped wall stays at or above InitialL", func(t *testing.T) {
		t.Parallel()
		initialL := testNowMs + int64(time.Hour/time.Millisecond)
		nowMs := testNowMs
		now := func() time.Time { return time.UnixMilli(nowMs) }
		clk := clock.NewClock(now, initialL, 0, testDriftWarn)
		nowMs = testNowMs - 3*int64(time.Hour/time.Millisecond)
		prev := reading{l: initialL, counter: 0}
		for i := 1; i <= 100; i++ {
			nowMs -= 1000
			l, counter, _ := clk.Tick()
			got := reading{l: l, counter: counter}
			if got.l < initialL {
				t.Fatalf("tick %d l = %d, want at or above InitialL %d", i, got.l, initialL)
			}
			if !got.after(prev) {
				t.Fatalf("tick %d = (%d, %d), want strictly after (%d, %d)",
					i, got.l, got.counter, prev.l, prev.counter)
			}
			prev = got
		}
	})
	t.Run("InitialL zero starts from the wall clock", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewClock(fixedNow(testNowMs), 0, 0, testDriftWarn)
		first := reading{}
		first.l, first.counter, _ = clk.Tick()
		if first != (reading{l: testNowMs, counter: 0}) {
			t.Errorf("first tick = (%d, %d), want (%d, 0)", first.l, first.counter, testNowMs)
		}
		second := reading{}
		second.l, second.counter, _ = clk.Tick()
		if second != (reading{l: testNowMs, counter: 1}) {
			t.Errorf("second tick = (%d, %d), want (%d, 1)", second.l, second.counter, testNowMs)
		}
	})
	t.Run("InitialC seeds the counter: first tick strictly after the watermark", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewClock(fixedNow(testNowMs), testNowMs, 5, testDriftWarn)
		first := reading{}
		first.l, first.counter, _ = clk.Tick()
		if first != (reading{l: testNowMs, counter: 6}) {
			t.Errorf("first tick = (%d, %d), want (%d, 6)", first.l, first.counter, testNowMs)
		}
		second := reading{}
		second.l, second.counter, _ = clk.Tick()
		if second != (reading{l: testNowMs, counter: 7}) {
			t.Errorf("second tick = (%d, %d), want (%d, 7)", second.l, second.counter, testNowMs)
		}
	})
	t.Run("InitialC at counterMax bumps l on the first tick", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewClock(fixedNow(testNowMs), testNowMs, testCounterMax, testDriftWarn)
		got := reading{}
		got.l, got.counter, _ = clk.Tick()
		want := reading{l: testNowMs + 1, counter: 0}
		if got != want {
			t.Errorf("first tick = (%d, %d), want (%d, %d) through the overflow rule",
				got.l, got.counter, want.l, want.counter)
		}
		watermark := reading{l: testNowMs, counter: testCounterMax}
		if !got.after(watermark) {
			t.Errorf("first tick = (%d, %d), want strictly after the watermark (%d, %d)",
				got.l, got.counter, watermark.l, watermark.counter)
		}
	})
	t.Run("wall clock ahead of the watermark starts from the wall clock", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewClock(fixedNow(testNowMs+10), testNowMs, 5, testDriftWarn)
		got := reading{}
		got.l, got.counter, _ = clk.Tick()
		if got != (reading{l: testNowMs + 10, counter: 0}) {
			t.Errorf("first tick = (%d, %d), want (%d, 0)", got.l, got.counter, testNowMs+10)
		}
		watermark := reading{l: testNowMs, counter: 5}
		if !got.after(watermark) {
			t.Errorf("first tick = (%d, %d), want strictly after the watermark (%d, %d) because l leads",
				got.l, got.counter, watermark.l, watermark.counter)
		}
	})
	t.Run("out-of-range InitialC is masked to the 12-bit domain", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewClock(fixedNow(testNowMs), testNowMs, testCounterMax+6, testDriftWarn)
		got := reading{}
		got.l, got.counter, _ = clk.Tick()
		if got != (reading{l: testNowMs, counter: 6}) {
			t.Errorf("first tick = (%d, %d), want (%d, 6) with the seed 4101 masked to 5",
				got.l, got.counter, testNowMs)
		}
	})
}

// TestClock_watermarkClampedIntoDomain covers the constructor's watermark
// clamp (WR-02): an InitialL beyond the 48-bit domain ceiling — a corrupted
// dump watermark, the one input the intended Unpack pipeline can never
// produce — saturates at maxL instead of persisting outside the domain,
// where PackVersion's tsMask would silently wrap every subsequent write to
// the bottom of the version stream, and a negative watermark reads as zero,
// starting from the wall clock like an unseeded restart.
func TestClock_watermarkClampedIntoDomain(t *testing.T) {
	t.Parallel()
	t.Run("watermark above the ceiling saturates at maxL", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewClock(fixedNow(testNowMs), maxL+5, 0, testDriftWarn)
		got := reading{}
		got.l, got.counter, _ = clk.Tick()
		want := reading{l: maxL, counter: 1}
		if got != want {
			t.Errorf("first tick = (%d, %d), want the clamped (%d, %d)",
				got.l, got.counter, want.l, want.counter)
		}
		ver := version.PackVersion(got.l, got.counter, testNodeID)
		top := version.PackVersion(maxL, 0, testNodeID)
		if ver.Compare(top) < 0 {
			t.Errorf("packed version %x reads below the domain top %x — the stream wrapped to the bottom",
				ver[:], top[:])
		}
	})
	t.Run("an int64-extreme watermark still lands inside the domain", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewClock(fixedNow(testNowMs), math.MaxInt64, 0, testDriftWarn)
		l, _, _ := clk.Tick()
		if l != maxL {
			t.Errorf("first tick l = %d, want the saturated ceiling %d", l, maxL)
		}
	})
	t.Run("a negative watermark reads as zero and adopts the wall clock", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewClock(fixedNow(testNowMs), -3600_000, 0, testDriftWarn)
		got := reading{}
		got.l, got.counter, _ = clk.Tick()
		if got != (reading{l: testNowMs, counter: 0}) {
			t.Errorf("first tick = (%d, %d), want the wall reading (%d, 0)",
				got.l, got.counter, testNowMs)
		}
	})
}

// TestClock_driftSignal covers the signed drift contract: zero within
// tolerance including exactly DriftWarn out, positive when l runs ahead of
// the wall clock beyond DriftWarn, negative on a forward wall-clock jump
// beyond DriftWarn past l, and always zero when DriftWarn is disabled
// (VERS-05, D-01).
func TestClock_driftSignal(t *testing.T) {
	t.Parallel()
	warnMs := int64(testDriftWarn / time.Millisecond)
	beyond := testDriftWarn + time.Millisecond
	tests := []struct {
		name     string
		op       string
		initialL int64
		nowMs    int64
		warn     time.Duration
		want     time.Duration
	}{
		{
			name: "tick: l exactly DriftWarn ahead of the wall is within tolerance",
			op:   "tick", initialL: testNowMs + warnMs, nowMs: testNowMs,
			warn: testDriftWarn, want: 0,
		},
		{
			name: "tick: wall exactly DriftWarn ahead of l is within tolerance",
			op:   "tick", initialL: testNowMs, nowMs: testNowMs + warnMs,
			warn: testDriftWarn, want: 0,
		},
		{
			name: "tick: l one ms beyond DriftWarn ahead of a backward wall is positive",
			op:   "tick", initialL: testNowMs + warnMs + 1, nowMs: testNowMs,
			warn: testDriftWarn, want: beyond,
		},
		{
			name: "tick: a forward wall jump beyond DriftWarn past l is negative",
			op:   "tick", initialL: testNowMs, nowMs: testNowMs + warnMs + 1,
			warn: testDriftWarn, want: -beyond,
		},
		{
			name: "tick: DriftWarn of zero disables the signal even an hour out",
			op:   "tick", initialL: testNowMs + 3600_000, nowMs: testNowMs,
			warn: 0, want: 0,
		},
		{
			name: "tick: negative DriftWarn disables the signal even an hour out",
			op:   "tick", initialL: testNowMs, nowMs: testNowMs + 3600_000,
			warn: -time.Minute, want: 0,
		},
		{
			name: "update: l one ms beyond DriftWarn ahead of the wall is positive",
			op:   "update", initialL: testNowMs + warnMs + 1, nowMs: testNowMs,
			warn: testDriftWarn, want: beyond,
		},
		{
			name: "update: a forward wall jump beyond DriftWarn past l is negative",
			op:   "update", initialL: testNowMs, nowMs: testNowMs + warnMs + 1,
			warn: testDriftWarn, want: -beyond,
		},
		{
			name: "update: DriftWarn of zero disables the signal",
			op:   "update", initialL: testNowMs + 3600_000, nowMs: testNowMs,
			warn: 0, want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			clk := clock.NewClock(fixedNow(tt.nowMs), tt.initialL, 0, tt.warn)
			var drift time.Duration
			switch tt.op {
			case "tick":
				_, _, drift = clk.Tick()
			case "update":
				_, _, drift = clk.Update(tt.nowMs, 0)
			default:
				t.Fatalf("unknown op %q", tt.op)
			}
			if drift != tt.want {
				t.Errorf("%s drift = %v, want %v", tt.op, drift, tt.want)
			}
		})
	}
}

// TestClock_driftSignalSaturates covers the drift signal at divergences a
// time.Duration cannot represent (WR-01): the raw (l - ptMs) * Millisecond
// product wraps int64 past ~292 years, flipping the magnitude — and on some
// operand pairs the sign — of exactly the worst drift the signal exists to
// report: the VERS-05 diagnostic reads a bogus ~96-year drift on a clock one
// maxL peer record saturated, and a pre-epoch wall reading reads negative.
// The saturated signal keeps the correct sign at a magnitude no smaller than
// the band left after the wall offset, never a wrapped one.
func TestClock_driftSignalSaturates(t *testing.T) {
	t.Parallel()
	// The floor the saturated readings must clear: both operands clamp into
	// the ±driftMsMax band before the nanosecond conversion, so a wall
	// offset of testNowMs still leaves ~237 years of the ~292-year band.
	floor := 200 * 365 * 24 * time.Hour
	t.Run("tick after a maxL Merge reports a saturated positive drift", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewClock(fixedNow(testNowMs), 0, 0, testDriftWarn)
		clk.Update(maxL, 0)
		_, _, drift := clk.Tick()
		if drift <= 0 || drift < floor {
			t.Errorf("drift after a maxL Merge = %v, want a saturated reading >= %v", drift, floor)
		}
	})
	t.Run("update from a saturated clock reports a saturated positive drift", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewClock(fixedNow(testNowMs), maxL, 0, testDriftWarn)
		_, _, drift := clk.Update(testNowMs, 0)
		if drift <= 0 || drift < floor {
			t.Errorf("drift at maxL = %v, want a saturated reading >= %v", drift, floor)
		}
	})
	t.Run("a pre-epoch wall reading reads positive, not sign-flipped", func(t *testing.T) {
		t.Parallel()
		clk := clock.NewClock(fixedNow(-maxL), maxL, 0, testDriftWarn)
		_, _, drift := clk.Tick()
		if drift <= 0 || drift < floor {
			t.Errorf("drift over a pre-epoch wall = %v, want a saturated reading >= %v", drift, floor)
		}
	})
}
