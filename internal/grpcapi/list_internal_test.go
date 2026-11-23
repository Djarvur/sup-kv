package grpcapi

import (
	"slices"
	"testing"

	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// This file holds the internal List tests: the page walk is the only seam
// that can deterministically hand a page the view→lookup window a concurrent
// write opens, so the tombstone-race contract is pinned here rather than
// through the RPC surface (which cannot schedule a write between the view
// and the lookup). The file name keeps the testpackage exemption's
// *_internal_test.go convention.

// TestServer_listRace pins the view→lookup window: a key the live view
// named can race into a tombstone or disappear entirely before the page
// walk reads it. The walk must skip it, keep keys and records aligned, and
// advance the token past it, so the next page makes progress and no key is
// lost or repeated.
func TestServer_listRace(t *testing.T) {
	t.Parallel()

	seen := func(key string) (storage.Record, bool) {
		return storage.Record{
			Version: version.RecordVersion{1},
			Data:    []byte("v-" + key),
		}, true
	}
	cases := []struct {
		name   string
		lookup func(key string) (storage.Record, bool)
	}{
		{
			name: "tombstoned after the view",
			lookup: func(key string) (storage.Record, bool) {
				if key == "c" {
					return storage.Record{Version: version.RecordVersion{2}, Deleted: true}, true
				}
				return seen(key)
			},
		},
		{
			name: "vanished after the view",
			lookup: func(key string) (storage.Record, bool) {
				if key == "c" {
					return storage.Record{}, false
				}
				return seen(key)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// The view named a, b, c, d, e; c raced away before its turn.
			page := pagedResponse(tc.lookup, []string{"a", "b", "c", "d", "e"}, 3)
			if got, want := page.GetKeys(), []string{"a", "b", "d"}; !slices.Equal(got, want) {
				t.Fatalf("page keys = %v, want the raced-away key skipped: %v", got, want)
			}
			if got := len(page.GetRecords()); got != len(page.GetKeys()) {
				t.Fatalf("page records = %d, want the %d aligned keys", got, len(page.GetKeys()))
			}
			for i, key := range page.GetKeys() {
				if got := page.GetRecords()[i].GetKey(); got != key {
					t.Fatalf("records[%d] key = %q, want the aligned key %q", i, got, key)
				}
			}

			// The token is the last examined key — d, past the skipped c —
			// so the next page resumes with e and the skipped key leaves no
			// gap to stall on.
			if got, want := page.GetNextPageToken(), encodeListToken("d"); got != want {
				t.Fatalf("next_page_token = %q, want the last examined position %q", got, want)
			}
			next := pagedResponse(seen, []string{"e"}, 3)
			if got, want := next.GetKeys(), []string{"e"}; !slices.Equal(got, want) {
				t.Fatalf("next page keys = %v, want %v", got, want)
			}
			if got := next.GetNextPageToken(); got != "" {
				t.Fatalf("next page next_page_token = %q, want the empty end-of-list token", got)
			}
		})
	}
}
