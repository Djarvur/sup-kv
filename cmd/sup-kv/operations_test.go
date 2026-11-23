package main_test

// The DOCS-02 presence guard of Phase 7 (07-03 Task 2): docs/OPERATIONS.md is
// the single operations file (07-CONTEXT D-01 — no split runbook), and this
// test pins it against deletion and topic drift by asserting one anchor
// literal per mandated statement plus the operational extras (reflection
// disclosure, monitoring knobs, lifecycle/exit codes). HONEST SCOPE: this is
// a presence/anchor guard only — it proves the file exists and carries its
// mandated topics; the TRUTH of each statement (that the trust model, the
// bounds, the envelope and the runbook match what the shipped binary actually
// does) is the phase's manual UAT verification (07-VALIDATION.md's manual
// table), not something a substring check can establish.

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// operationsDocPath is the doc under guard, relative to this package's
// directory — the doc itself is the input, never a copied fixture.
const operationsDocPath = "../../docs/OPERATIONS.md"

// operationsAnchors pins every mandated topic of docs/OPERATIONS.md: the
// five DOCS-02 statements (trust model first and prominent, value-size
// envelope, local-read semantics, the not-a-lock warning, the clock-drift
// runbook) plus the operational extras the plan mandates (the always-on
// reflection disclosure beside the trust model, monitoring knobs, lifecycle
// and exit codes). Each literal is chosen from the authored doc's own
// wording — renaming a heading or dropping a knob breaks the guard loudly.
//
// The Phase 5 persistence rows pin the behavior this phase shipped: the
// load-outcomes section (## Load outcomes), the memory-model section
// (## Memory model), the preserved evidence naming (.corrupt-), the
// preservation-failure signal (preserve_error), the orphan-temp marker
// (.tmp~, the sweep's only selection key), the retry ladder's cap
// (5 s cap) and the health statement (SERVING under a persistent dump
// failure), plus the four new metric families pinned row by row.
// TestOperationsDocFamilies cross-checks those family names against the
// exporter, so the doc cannot document a family that does not exist.
//
// The Phase 7 rows pin the as-built operator contract (07-10): the lifecycle
// catalog (## Lifecycle logs) with one message literal per entry, the 0/1/2/3
// exit table, the reachable drift direction (received version leads), the
// startup drift_warn/max_clock_lead relation, the four routed core counters,
// the unbound_listing label and the current-RSS definition.
var operationsAnchors = []struct {
	literal string
	pins    string
}{
	{"## Trust model", "the trust-model section heading — the mandated FIRST section of the file"},
	{"no authentication", "the v1 wire ships no authentication (never overstated)"},
	{"no encryption", "the v1 wire ships no encryption"},
	{"CRC32", "integrity-only checksum — guards corruption, never tampering"},
	{"HMAC", "authenticated datagrams are documented future work, not a v1 feature"},
	{"reflection is always on", "the always-on gRPC reflection disclosure sits beside the trust model"},
	{"## What you are operating", "the AP/LWW framing section"},
	{"one round trip", "REPL-10 normal-operation convergence bound"},
	{"ack_timeout_max", "REPL-10 datagram-loss convergence bound"},
	{"sync_interval", "REPL-10 partition-heal bound component (2 × sync_interval + response)"},
	{"## Scale envelope", "the scale-envelope section"},
	{"60 KiB", "the value envelope in human units"},
	{"61440", "the envelope constant exactly as the code carries it (storage.ValueMaxBytes)"},
	{"--value-max-bytes", "the flag that tunes the envelope"},
	{"SUPKV_VALUE_MAX_BYTES", "the envelope flag's env twin"},
	{"InvalidArgument", "larger values are rejected with InvalidArgument"},
	{"in RAM", "the whole state lives in RAM"},
	{"## Read semantics", "the local-read semantics section"},
	{"read-your-writes", "the mandated read-consistency keyword (same node only)"},
	{"## Not a lock service", "the not-a-lock/not-leader-election warning section"},
	{"linearizability", "locks need linearizability — impossible on an AP LWW store"},
	{"fencing", "locks need fencing — the Out-of-Scope wording"},
	{"last-write-wins", "concurrent-write resolution is deterministic LWW"},
	{"supkv_merges_conflicts_total", "the merge-conflict counter is the observable (no conflict journal exists)"},
	{"## Clock drift runbook", "the clock-drift runbook section"},
	{"supkv_clock_drift_warnings_total", "the drift symptom metric"},
	{"--drift-warn", "the drift warning threshold knob (default 60s)"},
	{"NTP", "the runbook action: fix the clock source (NTP/PTP)"},
	{"## Monitoring", "the monitoring section"},
	{"--metrics-addr", "the opt-in metrics flag (unset means no listener)"},
	{"/metrics", "the Prometheus text endpoint"},
	{"/debug/vars", "the expvar endpoint"},
	{"SUPKV_METRICS_CERT", "the env-only mTLS certificate triple"},
	{"## Lifecycle logs", "the lifecycle-catalog section heading — one record per catalog entry at its documented level"},
	{"Starting sup-kv", "the startup INFO record's message"},
	{"Dump load", "the load-outcome INFO record's message (prompt/main)"},
	{"Startup gate opened", "the gate INFO record's message — reason sync or backstop"},
	{"Peer restarted", "the peer-restart WARN record's message"},
	{"Clock drift", "the drift WARN record's message"},
	{"## Lifecycle and exit codes", "the lifecycle/exit-codes section"},
	{"SIGTERM", "the graceful-shutdown trigger"},
	{"exit code 0", "clean shutdown: final dump confirmed on disk"},
	{"exit code 1", "startup failure or Serve error (Restart=on-failure reacts; a Serve error still runs the bounded final dump)"},
	{"exit code 2", "malformed configuration"},
	{"exit code 3", "a failed final dump, including on the Serve-error path"},
	{"Restart=on-failure", "the systemd reaction to exit code 1"},
	{"received version leads", "the drift condition's only direction — a received version ahead of local wall time (OBSV-05)"},
	{"drift_warn < max_clock_lead", "the startup relation that keeps the drift warning reachable"},
	{"unbound_listing", "the unbound-listing discard class in the complete catalog"},
	{"supkv_replication_listings_dropped_total", "the routed listing-drop counter (Phase 4 deferral)"},
	{"supkv_replication_sync_streams_dropped_total", "the routed partial-DataSync-stream counter (Phase 4 deferral)"},
	{"supkv_replication_listed_keys_evicted_total", "the routed listed-key eviction counter (Phase 4 deferral)"},
	{"supkv_replication_notifications_regressed_total", "the routed regressed-notification counter (Phase 4 deferral)"},
	{"current resident set size", "process_resident_memory_bytes reports the current RSS on Linux, not the peak"},
	{"SUPKV_LOG_LEVEL", "the structured-logging level knob"},
	{"SUPKV_LOG_FORMAT", "the structured-logging format knob (JSON default)"},
	{"## Load outcomes", "the load-outcomes section heading — Prompt/Main/Backup/Unusable and the one-WARN/one-ERROR rule"},
	{"## Memory model", "the memory-model section heading — one copy plus one record line plus the 64 KiB writer"},
	{".corrupt-", "the preserved evidence naming — never rotated, never auto-deleted, removed only by an operator after inspection"},
	{"preserve_error", "the preservation-failure hazard signal — non-empty means the corrupt generation may still be at main"},
	{".tmp~", "the hidden orphan-temp marker — the startup sweep's only selection key, age-bounded at one hour"},
	{"5 s cap", "the dump retry ladder's cap after doubling from 100 ms"},
	{"SERVING", "health stays SERVING under a persistent dump failure"},
	{"supkv_dump_load_total", "the startup load-outcome family — all four outcome labels render, at zero when unseen"},
	{"supkv_dump_records_discarded_total", "the load-time discard family — checksum/version/value_limit/malformed labels"},
	{"supkv_dumps_failures_total", "the dump failure-class family — disk_full/permission/other labels"},
	{"supkv_dumps_unconfirmed_total", "the unconfirmed-write counter — a confirmation timeout is not a dump error"},
}

// operationsForbidden pins the claims this phase's corrections removed: a
// failed final dump does not exit 1, and a Main load is not silent — the one
// INFO "Dump load" record carries the outcome. The entries are matched
// against a whitespace-normalized document (line wrapping in the source is
// not a way to keep a stale sentence alive), which is why the doc is
// normalized the same way in the test.
var operationsForbidden = []struct {
	phrase string
	why    string
}{
	{"Nothing is logged about the outcome itself", "the Main load now logs one INFO `Dump load` record carrying outcome=main"},
	{"failed-final-dump exit code 1", "a failed final dump exits with code 3; exit code 1 means startup or a Serve error"},
}

// TestOperationsDoc is the DOCS-02 drift guard: docs/OPERATIONS.md must
// exist, must open with the trust model before any other section (the
// prohibition: no usage instruction may precede the trust-model warning),
// and must carry every anchor literal.
func TestOperationsDoc(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(operationsDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", operationsDocPath, err)
	}
	doc := string(raw)
	for line := range strings.SplitSeq(doc, "\n") {
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		if line != "## Trust model" {
			t.Fatalf("%s: the first ## section is %q — the trust model must be the FIRST section, before any usage instruction (D-01)", operationsDocPath, strings.TrimSpace(line))
		}
		break
	}
	for _, a := range operationsAnchors {
		if !strings.Contains(doc, a.literal) {
			t.Fatalf("%s does not contain %q — %s", operationsDocPath, a.literal, a.pins)
		}
	}
	// The stale-sentence guard compares normalized text on both sides: the
	// doc wraps its paragraphs, so a raw substring search over the authored
	// wording would pass vacuously for a phrase split across a line break.
	normalized := strings.Join(strings.Fields(doc), " ")
	for _, f := range operationsForbidden {
		phrase := strings.Join(strings.Fields(f.phrase), " ")
		if strings.Contains(normalized, phrase) {
			t.Fatalf("%s still contains the stale claim %q — %s", operationsDocPath, f.phrase, f.why)
		}
	}
}

// metricsSourcePath is the exporter source whose declared family names the
// doc's family list must match — the API the runbook promises an operator,
// not a rendered sample.
const metricsSourcePath = "../../internal/metrics/metrics.go"

// supkvFamilyPattern matches one supkv_ family name in the runbook. The
// doc's own spelling drives the cross-check: whatever family it names must
// be grep-present in the exporter.
var supkvFamilyPattern = regexp.MustCompile(`supkv_[a-z][a-z0-9_]*`)

// TestOperationsDocFamilies is the exporter cross-check (T-5-31's
// documented-name half): every supkv_ family the runbook names must appear
// in internal/metrics/metrics.go, so a documented family that does not
// exist cannot send an operator to an empty graph. It is a presence check,
// not a truth claim: the values, labels and semantics stay
// TestOperationsDoc's and the phase's manual UAT's.
func TestOperationsDocFamilies(t *testing.T) {
	t.Parallel()
	doc, err := os.ReadFile(operationsDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", operationsDocPath, err)
	}
	exporter, err := os.ReadFile(metricsSourcePath)
	if err != nil {
		t.Fatalf("read %s: %v", metricsSourcePath, err)
	}
	names := supkvFamilyPattern.FindAllString(string(doc), -1)
	if len(names) == 0 {
		t.Fatalf("%s names no supkv_ family — the cross-check would pass vacuously", operationsDocPath)
	}
	for _, name := range names {
		if !strings.Contains(string(exporter), name) {
			t.Errorf("%s documents %s, which internal/metrics/metrics.go does not declare", operationsDocPath, name)
		}
	}
}
