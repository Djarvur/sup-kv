package storage_test

import (
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// ENG-19: Merge accepts any 16 bytes as a version. Bits outside (l, c, node)
// take part in the byte order but not in the HLC, so one malformed version
// makes every later LOCAL write to that key sort below the stored record.
func TestReview_malformedVersionBeatsLaterLocalWrites(t *testing.T) {
	now := func() time.Time { return time.UnixMilli(1_700_000_000_000) }
	store := storage.NewStore(1, clock.NewClock(now, 0, 0, 0), storage.Config{})
	bad := version.PackVersion(now().UnixMilli(), 0, 2)
	bad[6] |= 0xF0 // version nibble 0xF instead of 0x7 (hand-edited dump, foreign implementation, bit flip that kept the CRC valid)
	if _, _, err := store.Merge("k", storage.Record{Version: bad, Data: []byte("remote")}); err != nil {
		t.Fatal(err)
	}
	local, _, _ := store.Set("k", []byte("local, written afterwards"))
	t.Logf("stored remote %x\nlater local   %x  compare=%d", bad[:], local[:], local.Compare(bad))
	if local.Compare(bad) <= 0 {
		t.Errorf("a local write issued after the merge sorts below the merged version: peers will reject it and send the old record back")
	}
}

// ENG-19: one version carrying l = 2^48-1 pins the clock at the domain ceiling;
// 4095 writes later every write of this node reuses the same version.
func TestReview_ceilingVersionExhaustsDomain(t *testing.T) {
	now := func() time.Time { return time.UnixMilli(1_700_000_000_000) }
	store := storage.NewStore(1, clock.NewClock(now, 0, 0, 0), storage.Config{})
	if _, _, err := store.Merge("poison", storage.Record{Version: version.PackVersion(1<<48-1, 0, 2)}); err != nil {
		t.Fatal(err)
	}
	var last, prev version.RecordVersion
	for i := range 4200 {
		prev = last
		last, _, _ = store.Set("k", []byte{byte(i)})
	}
	t.Logf("write #4199 %x\nwrite #4200 %x", prev[:], last[:])
	if last == prev {
		t.Errorf("version domain exhausted after one poisoned merge: consecutive writes carry the identical version")
	}
}
