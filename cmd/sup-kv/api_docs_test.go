package main_test

// The phase-6 API-contract drift guard (06-09): docs/API.md is the
// phase-owned half of the client-facing contract (APIS-01…07, D-04), and this
// test pins each mandated statement by one anchor literal read from the repo
// file — never a copied fixture. Task 1 pins the public proto, the honest
// wait_for_replication semantics and the two List pagination modes; Task 2
// extends the table with the error table, the Watch durability contract, the
// snapshot full rollback, the transport/health envelope and the
// consumer-library boundary. The phase-9 DOCS-05 extension (09-06) replaces
// the pending tail: TestAPIDocPerRPCSections mirrors the public proto's own
// declaration order, the anchors pin the new sections, and the literals that
// pinned the pending section are now forbidden phrases. HONEST SCOPE: this is
// a presence/anchor guard — deleting a section, renaming a heading or
// dropping a mandated statement fails the normal test gate; the truth of each
// statement (and of every per-RPC error table's rows) against the code is the
// phase verification's manual read-through, not something a substring check
// can establish.

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

	// The total, path-free error table (APIS-03, D-03): one literal per
	// mapper constant plus every List constant message.
	{"## Errors", "the error-table section heading"},
	{"key not found", "storage.ErrNotFound — an absent or tombstoned key"},
	{"snapshot not found", "fs.ErrNotExist on a snapshot read"},
	{"key already exists", "storage.ErrAlreadyExists"},
	{"value exceeds the maximum allowed size", "storage.ErrValueTooLarge"},
	{"invalid snapshot name", "persistfile.ErrInvalidSnapshotName"},
	{"invalid page size", "the List page_size rejection"},
	{"invalid page token", "the List page_token rejection"},
	{"no dump on disk to snapshot", "persistfile.ErrNoDump — FailedPrecondition"},
	{"dump failed; write not confirmed", "persist.ErrDumpFailed"},
	{"write not confirmed before the confirmation timeout", "persist.ErrUnconfirmed"},
	{"persist engine closed", "persist.ErrClosed"},
	{"watch hub is shutting down", "the Watch shutdown status"},
	{"disk full", "port.FailureDiskFull"},
	{"permission denied", "port.FailurePermission and fs.ErrPermission"},
	{"Canceled", "context.Canceled through status.FromContextError"},
	{"DeadlineExceeded", "context.DeadlineExceeded through status.FromContextError"},
	{"internal error", "the totality fallback's constant"},
	{"detail is logged", "internal detail goes to the log, never the response"},
	{"no server path", "no message carries a server filesystem path"},

	// The Watch durability contract (APIS-04, D-04).
	{"## Watch", "the Watch section heading"},
	{"after the dump confirmation", "live events are released only once durable"},
	{"exactly once", "one confirmed write yields one event"},
	{"Deletions are events", "a tombstone is a change event, not a silent removal"},
	{"version order", "per-key events arrive in increasing version order"},
	{"resubscribe", "the slow-subscriber recovery path"},
	{"current in-memory state", "the snapshot half is current memory, not a durable view"},

	// The snapshot full rollback (APIS-07, D-07).
	{"## Snapshots", "the snapshots section heading"},
	{"full rollback", "a load makes the snapshot the store's state, not a merge"},
	{"absent from the snapshot", "live keys the snapshot does not name are deleted"},
	{"fresh version", "replayed records carry new versions, never the snapshot's"},
	{"no version is burned", "an already-tombstoned absent key is untouched"},
	{"broadcast", "the rollback result replicates"},
	{"not linearizable", "the honest limit on concurrent-write ordering"},

	// The explicit transport/health envelope (APIS-05, D-05).
	{"## Transport and health", "the transport section heading"},
	{"2 m", "the server keepalive idle ping period"},
	{"20 s", "the keepalive answer timeout"},
	{"30 s", "the minimum client ping interval"},
	{"MaxRecvBytes", "the explicit receive ceiling"},
	{"MaxSendBytes", "the explicit send ceiling"},
	{"187 472", "the default-value-limit receive ceiling"},
	{"ResourceExhausted", "an over-limit request is rejected at the transport"},
	{"NOT_SERVING", "the health state while the gate is closed and after shutdown begins"},
	{"never returns to SERVING", "the one-way health rule"},

	// The consumer-zone boundary (APIS-06, ADR 0007 / 14.7).
	{"## Consumer library territory", "the consumer-zone section heading"},
	{"consumer-library territory", "the last-read-value pattern belongs to the consumer's library"},
	{"ADR 0007", "the accepted placement decision"},
	{"14.7", "the owner decision that settled it"},
	{"not a server feature", "the binary ships no last-read-value cache"},

	// The completed integrator reference (DOCS-05): the pending section's
	// replacement. The rows below pin the phase-9 sections exactly like the
	// rows above pin the phase-6 contract; the literals that pinned the
	// pending section itself moved to apiDocForbidden.
	{"## RPC reference", "the per-RPC reference section heading — the pending section's replacement"},
	{"in the schema's own declaration order", "the reference mirrors the proto's RPC declaration order"},
	{"## Watch contract", "the API-level Watch guarantee section heading"},
	{"snapshot-then-live", "the exact cut is stated as the API-level guarantee"},
	{"bounded pending", "the pending queue's bound and its shed policy are part of the guarantee"},
	{"## Write-latency model", "the latency-model section heading"},
	{"docs/PERFORMANCE.md", "the one-ledger link: the model's figures live in PERFORMANCE.md, never here"},
	{"## Key-prefix convention", "the key-prefix convention section heading"},
	{"enforced by documentation", "the prefix convention binds by documentation, not by the store"},
	{"## Consistency caveats", "the consistency-caveats section heading"},
	{"read-your-writes", "the writing node serves read-your-writes"},
	{"Unavailable, not NotFound", "the listed-but-unarrived window answers Unavailable (rule 14.5(6))"},
	{"## Client patterns", "the client-patterns section heading"},
	{"last-read-value cache", "the cache pattern named where the integrator guidance lives"},
	{"obtaining the proto", "the proto-obtainment guidance in the client patterns"},
	{"api/supkvapi/v1/supkvapi.proto", "the proto source a client obtains the schema from"},
	{"examples/client", "the runnable client example module"},
	{"example.com/sup-kv-client", "the example module's own module path — the importability proof"},
}

// apiDocForbidden carries the pending-section literals the phase-9 extension
// flipped (09-06): they pinned `## What Phase 9 adds` until the real sections
// it promised replaced it, so their return would regress the contract.
// Matching is whitespace-normalized on both sides, per the operations-test
// idiom — line wrapping is not a way to keep a stale claim alive.
var apiDocForbidden = []struct {
	phrase string
	why    string
}{
	{"What Phase 9 adds", "the pending section is replaced by the RPC reference; its return would regress the contract"},
	{"Treat those sections as pending", "the pending disclaimer is gone — the document claims the sections it states"},
	{"per-RPC failure table", "the pending deliverable's old name; the shipped section is the RPC reference with per-RPC error tables"},
	{"DOCS-05", "planning vocabulary stays out of the published contract"},
}

// readAPIDoc reads the doc under guard from the repo — the file is the input,
// never a fixture copy.
func readAPIDoc(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(apiDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", apiDocPath, err)
	}
	return string(raw)
}

// TestAPIDoc fails when docs/API.md is absent, carries a forbidden
// pending-section claim, or loses any mandated anchor: the contract document
// is a published artifact, so its sections are pinned against drift by the
// same gate that guards the code. The forbidden check runs first, so a
// returned pending section is named even while sections are also missing.
func TestAPIDoc(t *testing.T) {
	t.Parallel()
	doc := readAPIDoc(t)
	normalized := strings.Join(strings.Fields(doc), " ")
	for _, f := range apiDocForbidden {
		phrase := strings.Join(strings.Fields(f.phrase), " ")
		if strings.Contains(normalized, phrase) {
			t.Fatalf("%s still contains the stale claim %q — %s", apiDocPath, f.phrase, f.why)
		}
	}
	for _, a := range apiDocAnchors {
		if !strings.Contains(doc, a.literal) {
			t.Fatalf("%s does not contain %q — %s", apiDocPath, a.literal, a.pins)
		}
	}
}

// apiProtoPath is the public schema the RPC reference must mirror, relative to
// this package's directory — the repo file is the input, never a fixture copy.
const apiProtoPath = "../../api/supkvapi/v1/supkvapi.proto"

// apiDocMinRPCs is the vacuity floor for the schema parse: the proto declares
// seven RPCs today (Get, Set, Delete, List, Watch, SnapshotCreate,
// SnapshotLoad — SetIfAbsent is Set's create_if_absent variant and
// WaitForReplication is Set/Delete's wait fields, never separate RPCs). A
// parse below the floor means the parser or the schema drifted, and the order
// check below would pass vacuously.
const apiDocMinRPCs = 7

// protoRPCs parses the service SupKVService block's rpc declarations in
// source order. The parse is structural, not a full protobuf parse: a line
// opens the service block, a lone closing brace ends it, and `rpc Name(`
// lines inside declare the RPCs.
func protoRPCs(t *testing.T, proto string) []string {
	t.Helper()
	var (
		rpcs      []string
		inService bool
	)
	for line := range strings.SplitSeq(proto, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "service SupKVService {"):
			inService = true
		case inService && trimmed == "}":
			return rpcs
		case inService && strings.HasPrefix(trimmed, "rpc "):
			name, _, _ := strings.Cut(strings.TrimPrefix(trimmed, "rpc "), "(")
			name = strings.TrimSpace(name)
			if name == "" {
				t.Fatalf("%s: an rpc declaration without a name — %q", apiProtoPath, trimmed)
			}
			rpcs = append(rpcs, name)
		}
	}
	t.Fatalf("%s: the SupKVService block never closes — the schema parse found no end", apiProtoPath)
	return nil // unreachable
}

// TestAPIDocPerRPCSections fails when the RPC reference does not mirror the
// public schema: every rpc the proto declares must have a `### <Name>` section
// inside `## RPC reference`, the section headings must be exactly the schema's
// declarations in the schema's own order (a missing RPC, an extra section or a
// reordered one all fail — the reference and the schema cannot silently
// diverge), and each section must carry its error table. HONEST SCOPE: the
// check pins section presence, order and the `#### Errors` marker; the truth
// of each table's rows against the mapper is the phase verification's manual
// read-through, the same disclosure the anchor test makes.
func TestAPIDocPerRPCSections(t *testing.T) {
	t.Parallel()
	protoRaw, err := os.ReadFile(apiProtoPath)
	if err != nil {
		t.Fatalf("read %s: %v", apiProtoPath, err)
	}
	rpcs := protoRPCs(t, string(protoRaw))
	if len(rpcs) < apiDocMinRPCs {
		t.Fatalf("%s: the schema parse found %d rpcs, below the floor of %d — the parse or the schema drifted and the order check would pass vacuously",
			apiProtoPath, len(rpcs), apiDocMinRPCs)
	}
	sections := docSections(readAPIDoc(t))
	reference, ok := sections["## RPC reference"]
	if !ok {
		t.Fatalf("%s carries no %q section — the per-RPC reference in the schema's own declaration order (%s)",
			apiDocPath, "## RPC reference", strings.Join(rpcs, ", "))
	}

	// Split the section into its `### ` subsections, keeping declaration
	// order; `#### ` sub-headings stay inside their section's body.
	var (
		order   []string
		bodies  = make(map[string]string)
		current string
		text    strings.Builder
	)
	flush := func() {
		if current != "" {
			order = append(order, current)
			bodies[current] = text.String()
		}
		current = ""
		text.Reset()
	}
	for line := range strings.SplitSeq(reference, "\n") {
		if strings.HasPrefix(line, "### ") {
			flush()
			current = strings.TrimSpace(strings.TrimPrefix(line, "### "))
			continue
		}
		if current != "" {
			text.WriteString(line)
			text.WriteByte('\n')
		}
	}
	flush()

	if len(order) != len(rpcs) {
		t.Fatalf("%s: the RPC reference carries %d sections, the schema declares %d rpcs — the reference and the schema have diverged (schema order: %s; doc order: %s)",
			apiDocPath, len(order), len(rpcs), strings.Join(rpcs, ", "), strings.Join(order, ", "))
	}
	for i := range rpcs {
		if order[i] != rpcs[i] {
			t.Fatalf("%s: the RPC reference's section %d is %q, the schema's declaration %d is %q — the reference must follow the proto's declaration order",
				apiDocPath, i+1, order[i], i+1, rpcs[i])
		}
	}
	for _, rpc := range rpcs {
		if !strings.Contains(bodies[rpc], "#### Errors") {
			t.Fatalf("the %s section carries no %q error-table marker — every RPC's section must state the codes its inputs can produce", rpc, "#### Errors")
		}
	}
}
