package version_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/internal/testutil"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// validateLead is the shipped max_clock_lead default (07-02): 2 m, strictly
// greater than the 60 s drift-warn tolerance so an accepted version can be
// old enough to warn.
const validateLead = 2 * time.Minute

// hourMs is one hour in the layout's unix_ts_ms units, used for the accepted
// past-reading rows.
const hourMs = 60 * 60 * 1000

// TestValidate pins the XPORT-04 verdict at its exact boundaries: the layout
// masks, the inclusive now+maxLead edge, the overflow-safe future bound and
// the RFC 9562 filler-bit interoperability rule.
func TestValidate(t *testing.T) {
	t.Parallel()
	now := time.UnixMilli(testNowMs)
	leadMs := validateLead.Milliseconds()

	allOnes := version.RecordVersion{}
	for i := range allOnes {
		allOnes[i] = 0xFF
	}

	badVersion := version.PackVersion(testNowMs, 0x123, 0xABCD)
	badVersion[6] = badVersion[6]&0x0F | 0x60
	badVariantZero := version.PackVersion(testNowMs, 0x123, 0xABCD)
	badVariantZero[8] &^= uuidVarMask
	badVariantOne := version.PackVersion(testNowMs, 0x123, 0xABCD)
	badVariantOne[8] |= uuidVarMask

	// A conforming UUIDv7 from another vendor: the filler bits RFC 9562
	// permits set in octet 10's low six bits and octets 11-15.
	filler := version.PackVersion(testNowMs, 0x123, 0xABCD)
	filler[10] |= 0x3F
	copy(filler[11:], []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x01})

	// A wall clock 900 microseconds into the millisecond: the layout carries
	// milliseconds only, so both verdicts must match the truncated reading.
	fractional := time.UnixMilli(testNowMs).Add(900 * time.Microsecond)

	tests := []struct {
		name    string
		v       version.RecordVersion
		now     time.Time
		maxLead time.Duration
		wantErr error
	}{
		{
			name:    "a version packed at now is accepted",
			v:       version.PackVersion(testNowMs, 0x123, 0xABCD),
			now:     now,
			maxLead: validateLead,
		},
		{
			name:    "a version packed 24 h in the past is accepted",
			v:       version.PackVersion(testNowMs-24*hourMs, 0x123, 0xABCD),
			now:     now,
			maxLead: validateLead,
		},
		{
			name:    "a reading exactly at now plus the maximum lead is accepted",
			v:       version.PackVersion(testNowMs+leadMs, 0x123, 0xABCD),
			now:     now,
			maxLead: validateLead,
		},
		{
			name:    "a reading one millisecond beyond now plus the maximum lead is rejected",
			v:       version.PackVersion(testNowMs+leadMs+1, 0x123, 0xABCD),
			now:     now,
			maxLead: validateLead,
			wantErr: version.ErrVersionFuture,
		},
		{
			name:    "all bits set fails the layout at the default lead",
			v:       allOnes,
			now:     now,
			maxLead: validateLead,
			wantErr: version.ErrVersionLayout,
		},
		{
			name:    "all bits set fails the layout at an absurd lead",
			v:       allOnes,
			now:     now,
			maxLead: math.MaxInt64 / 2,
			wantErr: version.ErrVersionLayout,
		},
		{
			name:    "a version nibble of 0x60 fails the layout",
			v:       badVersion,
			now:     now,
			maxLead: validateLead,
			wantErr: version.ErrVersionLayout,
		},
		{
			name:    "variant 0b00 fails the layout",
			v:       badVariantZero,
			now:     now,
			maxLead: validateLead,
			wantErr: version.ErrVersionLayout,
		},
		{
			name:    "variant 0b11 fails the layout",
			v:       badVariantOne,
			now:     now,
			maxLead: validateLead,
			wantErr: version.ErrVersionLayout,
		},
		{
			name:    "a negative lead normalizes to zero: a reading at now is accepted",
			v:       version.PackVersion(testNowMs, 0x123, 0xABCD),
			now:     now,
			maxLead: -time.Second,
		},
		{
			name:    "a negative lead normalizes to zero: a reading 1 ms ahead is rejected",
			v:       version.PackVersion(testNowMs+1, 0x123, 0xABCD),
			now:     now,
			maxLead: -time.Second,
			wantErr: version.ErrVersionFuture,
		},
		{
			name:    "random filler bits in rand_b are accepted",
			v:       filler,
			now:     now,
			maxLead: validateLead,
		},
		{
			name:    "a fractional now accepts exactly nowMs plus the lead",
			v:       version.PackVersion(testNowMs+leadMs, 0x123, 0xABCD),
			now:     fractional,
			maxLead: validateLead,
		},
		{
			name:    "a fractional now rejects one millisecond beyond the lead",
			v:       version.PackVersion(testNowMs+leadMs+1, 0x123, 0xABCD),
			now:     fractional,
			maxLead: validateLead,
			wantErr: version.ErrVersionFuture,
		},
		{
			name:    "an absurd lead does not overflow: a past reading is accepted",
			v:       version.PackVersion(testNowMs-hourMs, 0x123, 0xABCD),
			now:     now,
			maxLead: math.MaxInt64 / 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.v.Validate(tt.now, tt.maxLead)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("Validate(%x) = %v, want nil", tt.v[:], err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate(%x) = %v, want %v", tt.v[:], err, tt.wantErr)
			}
		})
	}
}

// layoutOK is the test's independent reading of the RFC 9562 layout rule: the
// version 7 nibble in octet 6's high bits and the variant 0b10 in octet 8's
// top two bits, exactly the masks the package documents.
func layoutOK(v version.RecordVersion) bool {
	return v[6]&uuidVerMask == uuidVerBits && v[8]&uuidVarMask == uuidVarBits
}

// modelVerdict is the test's independent model of Validate at (nowMs, leadMs):
// a layout failure first, then the overflow-safe future bound, which rejects
// only readings strictly beyond the inclusive now+lead edge.
func modelVerdict(v version.RecordVersion, nowMs, leadMs int64) error {
	if !layoutOK(v) {
		return version.ErrVersionLayout
	}
	l, _, _ := v.Unpack()
	if l >= nowMs && l-nowMs > leadMs {
		return version.ErrVersionFuture
	}
	return nil
}

// verdictMatches reports whether got carries the want sentinel, or both are nil.
func verdictMatches(got, want error) bool {
	if want == nil {
		return got == nil
	}
	return errors.Is(got, want)
}

// validateSafe calls Validate and converts a panic into a value the property
// loop can assert on, so a bad draw is reported with its bytes instead of
// aborting the test binary.
func validateSafe(v version.RecordVersion, now time.Time, maxLead time.Duration) (panicked any, err error) {
	defer func() { panicked = recover() }()
	return nil, v.Validate(now, maxLead)
}

// TestValidate_property proves Validate is total over arbitrary wire bytes and
// consistent with the package's own packing (XPORT-04, T-3-07/T-3-08/T-3-09):
// 2000 seeded raw versions — attacker-chosen bit patterns — never panic and
// always agree with an independently computed layout/bound model; 1000
// PackVersion outputs around the future edge obey the inclusive boundary. The
// "property seed:" line printed by testutil.NewRNG carries the replay seed.
func TestValidate_property(t *testing.T) {
	t.Parallel()
	rng := testutil.NewRNG(t)
	now := time.UnixMilli(testNowMs)
	nowMs := now.UnixMilli()
	leadMs := validateLead.Milliseconds()

	const (
		rawDraws     = 2000
		packedDraws  = 1000
		windowMs     = int64(121 * 1000) // one second beyond the 2 m lead.
		nearBoundary = int64(1000)       // readings within a second of the edge.
	)

	for range rawDraws {
		v := testutil.RandRawVersion(rng)
		panicked, err := validateSafe(v, now, validateLead)
		if panicked != nil {
			t.Fatalf("Validate panicked on raw version %x: %v", v[:], panicked)
		}
		want := modelVerdict(v, nowMs, leadMs)
		if !verdictMatches(err, want) {
			t.Fatalf("Validate(%x) = %v, want %v (replay with the property seed above)",
				v[:], err, want)
		}
	}

	var accepted, future int
	for range packedDraws {
		var l int64
		if rng.IntN(2) == 0 {
			// Half the draws hug the inclusive edge so both branches of the
			// future bound are exercised, not just the vast past.
			l = nowMs + leadMs - nearBoundary/2 + rng.Int64N(nearBoundary+1)
		} else {
			l = nowMs - hourMs + rng.Int64N(hourMs+windowMs)
		}
		v := version.PackVersion(l,
			uint16(rng.IntN(testCounterMax+1)),
			uint16(rng.IntN(maxNodeID+1)))
		panicked, err := validateSafe(v, now, validateLead)
		if panicked != nil {
			t.Fatalf("Validate panicked on packed version %x: %v", v[:], panicked)
		}
		if l <= nowMs+leadMs {
			accepted++
			if err != nil {
				t.Fatalf("Validate(%x) with l=%d = %v, want nil", v[:], l, err)
			}
			continue
		}
		future++
		if !errors.Is(err, version.ErrVersionFuture) {
			t.Fatalf("Validate(%x) with l=%d = %v, want ErrVersionFuture", v[:], l, err)
		}
	}
	if accepted == 0 || future == 0 {
		t.Fatalf("the packed arm exercised accepted=%d future=%d, want both non-zero",
			accepted, future)
	}
}
