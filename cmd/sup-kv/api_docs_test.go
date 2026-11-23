package main_test

// The phase-6 API-contract drift guard (06-09): docs/API.md is the
// phase-owned half of the client-facing contract (APIS-01…07, D-04), and this
// test pins each mandated statement by one anchor literal read from the repo
// file — never a copied fixture. Task 1 pins the public proto, the honest
// wait_for_replication semantics and the two List pagination modes; Task 2
// extends the table with the error table, the Watch durability contract, the
// snapshot full rollback, the transport/health envelope and the
// consumer-library boundary. HONEST SCOPE: this is a presence/anchor guard —
// deleting a section, renaming a heading or dropping a mandated statement
// fails the normal test gate; the truth of each statement against the code is
// the phase verification's manual read-through, not something a substring
// check can establish.

import (
	"os"
	"strings"
	"testing"
)

// apiDocPath is the doc under guard, relative to this package's directory —
// the repo file is the input, never a fixture copy.
const apiDocPath = "../../docs/API.md"

// apiDocAnchors pins the phase-owned statements of docs/API.md: one literal
// per mandated fact, chosen from the authored doc's own wording. The `pins`
// column names the contract statement so a failure says what was lost.
var apiDocAnchors = []struct {
	literal string
	pins    string
}{
	// The document opening and the public proto (APIS-06, D-06).
	{"# sup-kv API", "the document title"},
	{"## Public proto", "the public proto section heading"},
	{"github.com/Djarvur/sup-kv/api/supkvapi/v1", "the public Go import path"},
	{"supkvapi.v1", "the proto package name"},
	{"SupKVService", "the service name"},
	{"make proto", "the regeneration command for the committed generated code"},

	// The honest wait_for_replication contract (APIS-01, D-01).
	{"## wait_for_replication", "the wait section heading"},
	{"write-time peer set", "the awaited set is captured before the engine write"},
	{"zero known peers", "the zero-peer answer is reported, never overstated"},
	{"acked=false", "the zero-peer acked flag is explicitly false"},
	{"total_peers=0", "the zero-peer total is explicitly zero"},
	{"Superseded", "a displaced version resolves through the Superseded result, not the timeout"},
	{"wait_timeout_ms", "the per-request timeout field"},
	{"caller-bounded", "the wait is bounded by the caller's request and operator default"},

	// The List pagination contract (APIS-02, D-02).
	{"## List pagination", "the List section heading"},
	{"page_size", "the page-size request field"},
	{"page_token", "the opaque position request field"},
	{"next_page_token", "the continuation field"},
	{"byte-wise", "the documented key ordering"},
	{"position", "the token is a position, not a snapshot"},
	{"unset", "page_size unset selects the legacy mode"},
	{"legacy", "the frozen single-call listing is named"},
	{"keys only", "the legacy mode returns keys without values"},
	{"256", "the maximum page size"},
	{"6 MiB", "the per-page serialized-byte budget"},
	{"empty token", "the end-of-list signal"},
}

// TestAPIDoc fails when docs/API.md is absent or loses any mandated anchor:
// the contract document is a published artifact, so its sections are pinned
// against drift by the same gate that guards the code.
func TestAPIDoc(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(apiDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", apiDocPath, err)
	}
	doc := string(raw)
	for _, a := range apiDocAnchors {
		if !strings.Contains(doc, a.literal) {
			t.Fatalf("%s does not contain %q — %s", apiDocPath, a.literal, a.pins)
		}
	}
}
