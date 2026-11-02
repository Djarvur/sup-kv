package app_test

import (
	"bytes"
	"encoding/hex"
	"math/rand/v2"
	"testing"

	"github.com/Djarvur/sup-kv/internal/app"
)

// Version-domain bounds from RFC 9562: the 48-bit unix_ts_ms ceiling and the
// 16-bit nodeID ceiling.
const (
	maxL      = 1<<48 - 1
	maxNodeID = 0xFFFF
)

// TestVersion_layout covers the RFC 9562 octet layout with fixed vectors:
// the packed bytes of (1, 0x123, 0xABCD), the 48-bit big-endian l in octets
// 0-5, the zero filler of rand_b, and the version and variant marker bits on
// every packed value (VERS-01).
func TestVersion_layout(t *testing.T) {
	t.Parallel()
	t.Run("PackVersion(1, 0x123, 0xABCD) matches the RFC 9562 octet vector", func(t *testing.T) {
		t.Parallel()
		got := app.PackVersion(1, 0x123, 0xABCD)
		const want = "0000000000017123aaf3400000000000"
		if gotHex := hex.EncodeToString(got[:]); gotHex != want {
			t.Errorf("PackVersion(1, 0x123, 0xABCD) = %s, want %s", gotHex, want)
		}
	})
	t.Run("the 48-bit l occupies octets 0-5 big-endian", func(t *testing.T) {
		t.Parallel()
		got := app.PackVersion(0x112233445566, 0, 0)
		want := [6]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66}
		if !bytes.Equal(got[0:6], want[:]) {
			t.Errorf("octets 0-5 = %x, want %x", got[0:6], want[:])
		}
	})
	t.Run("the rand_b filler bits are zero-filled", func(t *testing.T) {
		t.Parallel()
		got := app.PackVersion(testNowMs, 0x123, 0xABCD)
		for i := 11; i < len(got); i++ {
			if got[i] != 0 {
				t.Errorf("octet %d = %#x, want 0 (zero filler)", i, got[i])
			}
		}
		if got := got[10] & 0x3F; got != 0 {
			t.Errorf("octet 10 low bits = %#x, want 0 (zero filler)", got)
		}
	})
	t.Run("every packed value carries the version 7 and variant bits", func(t *testing.T) {
		t.Parallel()
		rng := newRNG(t)
		for range 100 {
			v := randVersion(rng)
			if got := v[6] & uuidVerMask; got != uuidVerBits {
				t.Fatalf("octet 6 version bits = %#x, want %#x", got, uuidVerBits)
			}
			if got := v[8] & uuidVarMask; got != uuidVarBits {
				t.Fatalf("octet 8 variant bits = %#x, want %#x", got, uuidVarBits)
			}
		}
	})
}

// TestVersion_roundTrip covers Unpack(PackVersion(x)) == x on the RFC 9562
// domain boundaries: l 0 and 0xFFFFFFFFFFFF, counter 0 and 4095, nodeID 0
// and 0xFFFF.
func TestVersion_roundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		l       int64
		counter uint16
		nodeID  uint16
	}{
		{name: "all zero", l: 0, counter: 0, nodeID: 0},
		{name: "maximum l, counter and nodeID", l: maxL, counter: testCounterMax, nodeID: maxNodeID},
		{name: "maximum l only", l: maxL, counter: 0, nodeID: 0},
		{name: "maximum counter only", l: 0, counter: testCounterMax, nodeID: 0},
		{name: "maximum nodeID only", l: 0, counter: 0, nodeID: maxNodeID},
		{name: "l with the top bit set", l: 0x800000000000, counter: 0x123, nodeID: 0xABCD},
		{name: "the layout vector", l: 1, counter: 0x123, nodeID: 0xABCD},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			packed := app.PackVersion(tt.l, tt.counter, tt.nodeID)
			l, counter, nodeID := packed.Unpack()
			if l != tt.l || counter != tt.counter || nodeID != tt.nodeID {
				t.Errorf("Unpack(PackVersion(%d, %d, %d)) = (%d, %d, %d), want the input triple",
					tt.l, tt.counter, tt.nodeID, l, counter, nodeID)
			}
		})
	}
}

// versionTriple is the reference-model domain: the unpacked fields of a
// RecordVersion in comparison order.
type versionTriple struct {
	l       int64
	counter uint16
	nodeID  uint16
}

// compareTriple is the field-by-field reference comparison of (l, counter,
// nodeID): -1, 0 or 1 as a sorts before, equal to or after b. It exists only
// inside the test — production comparison stays byte-wise.
func compareTriple(a, b versionTriple) int {
	switch {
	case a.l < b.l:
		return -1
	case a.l > b.l:
		return 1
	case a.counter < b.counter:
		return -1
	case a.counter > b.counter:
		return 1
	case a.nodeID < b.nodeID:
		return -1
	case a.nodeID > b.nodeID:
		return 1
	default:
		return 0
	}
}

// randTriple draws a comparison triple: half from a tiny space so ties on l
// and on (l, counter) occur and the nodeID tie-break decides, half from the
// full RFC 9562 domains.
func randTriple(rng *rand.Rand) versionTriple {
	if rng.IntN(2) == 0 {
		return versionTriple{
			l:       int64(rng.IntN(8)),
			counter: uint16(rng.IntN(4)),
			nodeID:  uint16(rng.IntN(4)),
		}
	}
	return versionTriple{
		l:       rng.Int64N(1 << 48),
		counter: uint16(rng.IntN(testCounterMax + 1)),
		nodeID:  uint16(rng.IntN(maxNodeID + 1)),
	}
}

// TestVersion_byteOrder proves the ordering equivalence: over 1000 seeded
// iterations the sign of the byte-wise Compare equals the sign of the
// field-by-field reference comparison of (l, counter, nodeID) (VERS-01). A
// failure reports the offending pair; the "property seed:" line printed by
// newRNG carries the replay seed (Pitfall 5).
func TestVersion_byteOrder(t *testing.T) {
	t.Parallel()
	rng := newRNG(t)
	for range 1000 {
		a, b := randTriple(rng), randTriple(rng)
		got := app.PackVersion(a.l, a.counter, a.nodeID).
			Compare(app.PackVersion(b.l, b.counter, b.nodeID))
		want := compareTriple(a, b)
		if got != want {
			t.Fatalf("Compare of (%d, %d, %d) vs (%d, %d, %d) = %d, want %d (replay with the property seed above)",
				a.l, a.counter, a.nodeID, b.l, b.counter, b.nodeID, got, want)
		}
	}
}

// canonicalRaw returns v with the RFC 9562 marker and filler bits forced to
// their packed values: the version 7 nibble in octet 6's high nibble, the
// variant 0b10 in octet 8's top two bits — bit 0x40 cleared — and the zero
// filler in octet 10's low six bits and octets 11-15. Every bit Unpack
// keeps survives verbatim; these are exactly the bits PackVersion rewrites.
func canonicalRaw(v app.RecordVersion) app.RecordVersion {
	v[6] = v[6]&0x0F | uuidVerBits
	v[8] = v[8]&0x3F | uuidVarBits
	v[10] &= 0xC0
	clear(v[11:])
	return v
}

// TestVersion_unpackTotal proves Unpack is total and lossless over the bits
// it keeps: for 1000 seeded random raw 16-byte values — arbitrary bit
// patterns off the wire — Unpack never panics and PackVersion of the
// unpacked triple reproduces v exactly modulo the RFC 9562 marker and filler
// bits (canonicalRaw), and Compare never panics on raw pairs (T-02-03).
func TestVersion_unpackTotal(t *testing.T) {
	t.Parallel()
	rng := newRNG(t)
	for range 1000 {
		v := randRawVersion(rng)
		l, counter, nodeID := v.Unpack()
		want := canonicalRaw(v)
		got := app.PackVersion(l, counter, nodeID)
		if got != want {
			t.Fatalf("PackVersion(Unpack(%x)) = %x, want %x — the input with the marker and filler bits normalized",
				v[:], got[:], want[:])
		}
		other := randRawVersion(rng)
		v.Compare(other) // total over raw wire patterns: must not panic.
	}
}
