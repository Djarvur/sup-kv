package grpcapi

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"
	"github.com/Djarvur/sup-kv/internal/app/clock"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/port"
)

// errDiscardNoSnapshots is what the internal tests' dump seam reports for a
// snapshot: these tests never take one.
var errDiscardNoSnapshots = errors.New("discardDumps: no snapshots")

// discardDumps is the internal tests' dump seam: it runs the fill into
// io.Discard so a generated dump costs no memory, and serves no load or
// snapshot. The cost tests only need a store whose values are already in
// memory.
type discardDumps struct{}

// SaveStream runs the fill into io.Discard and reports success.
func (discardDumps) SaveStream(fill func(io.Writer) error) error {
	return fill(io.Discard)
}

// Load reports a fresh start: the internal tests build engines with New, so
// no load ever runs.
func (discardDumps) Load(time.Time, func([]byte) error) port.LoadResult {
	return port.LoadResult{}
}

// CreateSnapshot refuses: the internal tests store no snapshot.
func (discardDumps) CreateSnapshot(string) error {
	return errDiscardNoSnapshots
}

// ReadSnapshot refuses: the internal tests store no snapshot.
func (discardDumps) ReadSnapshot(string) ([]byte, error) {
	return nil, errDiscardNoSnapshots
}

// This file holds the internal List tests: the page walk is the only seam
// that can deterministically hand a page the view→lookup window a concurrent
// write opens, so the tombstone-race contract is pinned here rather than
// through the RPC surface (which cannot schedule a write between the view
// and the lookup). The file name keeps the testpackage exemption's
// *_internal_test.go convention.

// TestListPage_maxKeySerializedBound proves the worst-case serialized page
// bound plan 06-07 turns into the transport send limit: with maximum-length
// keys at the production value limit, the page stops on the byte budget
// below the page size, still carries a token, and its marshaled size stays
// within listPageByteBudget plus listEnvelopeOverheadBytes.
func TestListPage_maxKeySerializedBound(t *testing.T) {
	t.Parallel()

	store := storage.NewStore(
		7,
		clock.NewClock(time.Now, 0, 0, time.Minute),
		storage.Config{ValueMaxBytes: storage.ValueMaxBytes},
	)
	engine := persist.New(store, discardDumps{}, persist.Config{})

	value := bytes.Repeat([]byte("v"), storage.ValueMaxBytes)
	keyPrefix := strings.Repeat("k", storage.MaxKeyBytes-4)
	for i := range maxListPageSize + 1 {
		key := fmt.Sprintf("%s%04d", keyPrefix, i)
		if got := len(key); got != storage.MaxKeyBytes {
			t.Fatalf("seed key length = %d, want exactly storage.MaxKeyBytes (%d)", got, storage.MaxKeyBytes)
		}
		rec := storage.Record{Version: version.PackVersion(int64(i)+1, 0, 7), Data: value}
		if _, _, err := engine.ApplyRemote(key, rec); err != nil {
			t.Fatalf("seed ApplyRemote(%q) error = %v, want nil", key, err)
		}
	}

	resp, err := listPage(engine, &apiproto.ListRequest{PageSize: maxListPageSize})
	if err != nil {
		t.Fatalf("listPage page_size %d: want nil error, got %v", maxListPageSize, err)
	}
	if got := len(resp.GetRecords()); got == 0 {
		t.Fatal("listPage: want a non-empty page, got 0 records")
	}
	if got := len(resp.GetRecords()); got >= maxListPageSize {
		t.Fatalf("listPage: want the byte budget to stop the page below %d, got %d records",
			maxListPageSize, got)
	}
	if got := resp.GetNextPageToken(); got == "" {
		t.Fatal("listPage: want a non-empty token while keys remain, got empty")
	}
	for _, key := range resp.GetKeys() {
		if got := len(key); got != storage.MaxKeyBytes {
			t.Fatalf("page key length = %d, want the maximum %d", got, storage.MaxKeyBytes)
		}
	}
	raw, err := proto.Marshal(resp)
	if err != nil {
		t.Fatalf("proto.Marshal(ListResponse): want nil error, got %v", err)
	}
	if got, max := len(raw), listPageByteBudget+listEnvelopeOverheadBytes; got > max {
		t.Fatalf("marshaled page = %d bytes, want at most listPageByteBudget+listEnvelopeOverheadBytes (%d)", got, max)
	}
}

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
