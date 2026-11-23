package grpcapi

import (
	"encoding/base64"
	"slices"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
)

// The List pagination constants (D-02, APIS-02). This file owns all four
// size constants; plan 06-07 consumes listPageByteBudget and
// listEnvelopeOverheadBytes for the transport send limit and must not
// redefine them.
const (
	// maxListPageSize bounds one page_size > 0 response: at most this many
	// records, whatever the client asked for (a larger page_size clamps).
	// The bound keeps one page's record count, and therefore the caller's
	// working set, independent of the store's key count.
	maxListPageSize = 256

	// listPageByteBudget is the serialized-byte budget of one page: the
	// summed listEntrySerializedBytes of the page's entries stays at or
	// below it, save the one-entry progress floor. The paged path's cost is
	// therefore independent of the store's total value bytes (A-02); plan
	// 06-07 sizes MaxSendBytes from this budget plus
	// listEnvelopeOverheadBytes.
	listPageByteBudget = 6 << 20

	// listRecordOverheadBytes is an upper bound on one entry's fixed
	// serialized bytes besides its key and value: the repeated-keys field
	// framing, the records-field and Record-message framing, the 16-byte
	// version field, the deleted bool and every varint length prefix.
	listRecordOverheadBytes = 128

	// listEntryKeyCopies is how many times one entry's key rides in the
	// serialized response: once in keys and once in records[].key.
	listEntryKeyCopies = 2

	// base64CharsPerGroup and base64GroupBytes are base64's expansion
	// terms: every three input bytes encode to four characters.
	base64CharsPerGroup = 4
	base64GroupBytes    = 3

	// tokenFieldTagBytes is the next_page_token field's fixed framing: the
	// one-byte field tag plus the two-byte varint length of a token at the
	// encoded ceiling.
	tokenFieldTagBytes = 3

	// grpcMessagePrefixBytes is the length prefix every gRPC message
	// carries ahead of its serialized bytes.
	grpcMessagePrefixBytes = 5

	// repeatedFieldFramingBytes is the response's two repeated fields'
	// framing outside the entries: the keys and records field tags plus one
	// length varint each (three bytes at this budget), rounded up.
	repeatedFieldFramingBytes = 8

	// listTokenEncodedBytes is the longest possible next_page_token: the
	// base64.RawURLEncoding expansion (ceil of 4n/3) of a key at the
	// storage.MaxKeyBytes ceiling — 1 366 characters for 1 024 bytes.
	listTokenEncodedBytes = (storage.MaxKeyBytes*base64CharsPerGroup + base64GroupBytes - 1) /
		base64GroupBytes

	// listEnvelopeOverheadBytes bounds everything a ListResponse carries
	// besides its entries: the next_page_token (its tag, length varint and
	// encoded ceiling), the response's two repeated-field tags and the
	// gRPC message prefix. plan 06-07 adds it to listPageByteBudget for
	// MaxSendBytes.
	listEnvelopeOverheadBytes = listTokenEncodedBytes + tokenFieldTagBytes +
		grpcMessagePrefixBytes + repeatedFieldFramingBytes
)

// Client-visible List status messages (D-03): constants, like every other
// message on the untrusted boundary, so no internal detail can leak.
const (
	// invalidPageSizeMessage answers a negative page_size and a non-empty
	// page_token without a positive page_size: the legacy listing has no
	// page to continue.
	invalidPageSizeMessage = "invalid page size"

	// invalidPageTokenMessage answers an undecodable token and one whose
	// decoded key exceeds storage.MaxKeyBytes: a token names a position,
	// and an out-of-domain position is client error, never a seek.
	invalidPageTokenMessage = "invalid page token"
)

// listPage is the List handler's body past the startup gate (D-02,
// APIS-02): one live key→version view and two truthful modes.
//
// page_size unset (0) is the legacy listing the frozen R-01 call shape
// requires: every live key under the prefix in byte-wise ascending order,
// keys only — no records, no token and no value bytes, because the mode
// never calls Lookup. A page_token is rejected on this path by the size
// resolver: the legacy listing has no page to continue.
//
// page_size > 0 is a bounded keyset page: at most the resolved page size
// records with values, keys aligned to records, a page_token that resumes
// strictly after its decoded position (a position, never a snapshot: keys
// added or deleted between pages may appear or disappear), and a non-empty
// next_page_token unless no matching key follows the one it encodes. A key
// that races into a tombstone or disappears between the view and its
// Lookup is skipped while the token advances past it, so the walk always
// progresses.
func listPage(engine *persist.Engine, req *apiproto.ListRequest) (*apiproto.ListResponse, error) {
	pageSize := resolveListPageSize(req.GetPageSize())
	start := ""
	if token := req.GetPageToken(); token != "" {
		decoded, err := decodeListToken(token)
		if err != nil {
			return nil, err
		}
		start = decoded
	}

	prefix := req.GetPrefix()
	live := engine.LiveVersions()
	keys := make([]string, 0, len(live))
	for key := range live {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if start != "" && key <= start {
			continue
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)

	if pageSize == 0 {
		return &apiproto.ListResponse{Keys: keys}, nil
	}
	return pagedResponse(engine, keys, pageSize), nil
}

// resolveListPageSize maps a request's page_size onto the resolved page
// size: zero and negative values keep the legacy all-keys listing, and a
// value above the maximum clamps to it. The returned size is what the page
// walk bounds itself by.
func resolveListPageSize(pageSize int32) int {
	if pageSize <= 0 {
		return 0
	}
	if pageSize > maxListPageSize {
		return maxListPageSize
	}
	return int(pageSize)
}

// pagedResponse walks the sorted matching keys and fills one page: at most
// pageSize entries, each record read through Lookup so only the page's
// values are copied, and the encoded last examined key as the token when
// any matching key follows it. A key that raced away between the view and
// the Lookup is skipped while the token still advances past it, and the
// page always includes at least one entry, so the walk cannot stall.
func pagedResponse(engine *persist.Engine, keys []string, pageSize int) *apiproto.ListResponse {
	capacity := min(pageSize, len(keys))
	resp := &apiproto.ListResponse{
		Keys:    make([]string, 0, capacity),
		Records: make([]*apiproto.Record, 0, capacity),
	}
	lastExamined := -1
	for i, key := range keys {
		if len(resp.Keys) >= pageSize {
			break
		}
		lastExamined = i
		rec, ok := engine.Lookup(key)
		if !ok || rec.Deleted {
			continue
		}
		resp.Keys = append(resp.Keys, key)
		resp.Records = append(resp.Records, recordToProto(key, rec))
	}
	if lastExamined >= 0 && lastExamined+1 < len(keys) {
		resp.NextPageToken = encodeListToken(keys[lastExamined])
	}
	return resp
}

// listEntrySerializedBytes is one entry's serialized size bound within a
// page: the key twice (it rides in keys and in records[].key), the value
// once, and listRecordOverheadBytes for the fixed framing. The page walk
// uses it to keep the response within listPageByteBudget.
func listEntrySerializedBytes(keyLen, valueLen int) int {
	return listEntryKeyCopies*keyLen + valueLen + listRecordOverheadBytes
}

// encodeListToken renders one page position as the opaque next_page_token:
// the raw URL-safe base64 of the key. The empty key set encodes to the
// empty string, which is exactly the end-of-list signal, so the walk never
// emits a token that cannot be followed.
func encodeListToken(key string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(key))
}

// decodeListToken reads one opaque page_token back into its position. A
// RawURLEncoding decode failure is client error with the constant message:
// the token crosses from an untrusted client into the page seek, so a bad
// token is rejected, never interpreted (T-06-15).
func decodeListToken(token string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", status.Error(codes.InvalidArgument, invalidPageTokenMessage)
	}
	return string(raw), nil
}
