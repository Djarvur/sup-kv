package version_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/version"
)

// validateLead is the design-default max_clock_lead locked by D-06: 60 s,
// equal to the documented drift tolerance, so peers inside it stay accepted.
const validateLead = 60 * time.Second

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
