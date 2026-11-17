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
	{"61440", "the envelope constant exactly as the code carries it (defaultValueMaxBytes)"},
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
	{"## Lifecycle and exit codes", "the lifecycle/exit-codes section"},
	{"SIGTERM", "the graceful-shutdown trigger"},
	{"exit code 0", "clean shutdown: final dump confirmed on disk"},
	{"exit code 1", "failed final dump (Restart=on-failure reacts)"},
	{"exit code 2", "malformed configuration"},
	{"Restart=on-failure", "the systemd reaction to exit code 1"},
	{"SUPKV_LOG_LEVEL", "the structured-logging level knob"},
	{"SUPKV_LOG_FORMAT", "the structured-logging format knob (JSON default)"},
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
	for _, line := range strings.Split(doc, "\n") {
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
}
