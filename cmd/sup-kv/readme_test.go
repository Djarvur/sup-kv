package main_test

// The README quickstart machine contract of Phase 7 (DOCS-01, D-06/D-07/D-08):
// README.md itself is the test input — every fenced block marked with the
// quickstart-verify marker is extracted and executed VERBATIM through bash -e
// against the real binary (the 07-01 executable-doc harness:
// extractMarkedBlocks, runBashVerbatim, supkvBinDir, grpcurlShim), with a PATH
// under which `sup-kv` resolves to a copy of the built binary and `grpcurl`
// resolves to a shim exec'ing the pinned go.mod tool directive. The marked
// block proves the full DOCS-01 story in the README's own words: two nodes on
// one broadcast segment, a write on node A visible on node B, kill -9 of one
// node, restart with identical arguments, and convergence with no operator
// action (D-07) — grpcurl is the first client (D-08). Zero marked blocks is a
// failure, not an empty run: the README cannot drift from reality without this
// test going red (D-06). The README must also link docs/OPERATIONS.md and
// docs/EXAMPLES.md (D-02 — ops detail never lives inline). The macos-15 leg
// follows the recorded option-b verdict of 07-01-SUMMARY.md: the darwin leg
// loud-skips naming the probe and its recorded evidence, exactly like the
// probe's own darwin leg (the phase-6 D-12/507228a precedent).

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
)

// readmeMarker selects the fenced blocks of README.md that CI executes
// verbatim — the same explicit opt-in contract as examplesMarker, so future
// prose fences (build instructions, dev setup) are never silently enrolled.
const readmeMarker = "<!-- quickstart-verify -->"

// readmeDocPath is the doc under test, relative to this package's directory —
// README.md itself is the input, never a copied fixture (D-06).
const readmeDocPath = "../../README.md"

// quickstartBroadcastOption encodes the verdict 07-01-SUMMARY.md records under
// `## Broadcast probe verdict` (maintainer decision 2026-11-17, plan 07-01
// Task 3): option-b — the README ships a CI-verified Linux block plus a marked
// darwin variant block, and the darwin leg of this test is a named skip
// carrying the recorded macos-15 evidence. Only a future probe proving the
// derive line on darwin (recorded as option-a) would remove the skip; keep
// this constant in lockstep with that verdict.
const quickstartBroadcastOption = "option-b"

// readmeSkipDarwin is the option-b darwin leg: a deterministic loud skip (it
// fires before any block runs — determinism of the macos-15 gate cell beats
// re-gathering evidence the probe verdict already recorded) naming the probe,
// the recorded verdict, and what still proves the behavior.
func readmeSkipDarwin(t *testing.T) {
	t.Helper()
	t.Skipf(
		"option-b decision (maintainer 2026-11-17, plan 07-01 Task 3, recorded in 07-01-SUMMARY.md " +
			"'## Broadcast probe verdict'): TestProbeQuickstartBroadcast/probeSendBroadcast on the macos-15 " +
			"gate cell of PR #32 (run 44083351667, job 134039317468, 2026-11-17) recorded that limited " +
			"broadcast dies at the sendto (no route to host) and phase 6 proved lo0 carries no BROADCAST " +
			"flag, so the README quickstart ships a CI-verified Linux block plus a marked darwin variant " +
			"block and this test's darwin leg is a named skip, never a real run. What still proves the " +
			"behavior: this test runs both marked blocks verbatim on the Linux gate cells (the Linux " +
			"block, and the darwin variant through its || echo fallback arm), and the phase-6 exec " +
			"scenarios (plan 06-07) prove kill -9/restart/convergence on Linux for real",
	)
}

// TestReadmeQuickstart is the DOCS-01 machine contract: every quickstart-verify
// block of README.md runs verbatim through bash -e against the built binary
// with the pinned grpcurl on PATH, the combined output must carry the base64
// value the quickstart writes (the write/replication round-trip), and the
// README must link the two docs (D-02).
func TestReadmeQuickstart(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(readmeDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", readmeDocPath, err)
	}
	if !strings.Contains(string(raw), readmeMarker) {
		t.Fatalf("README.md carries no %s block — the DOCS-01 contract is gone", readmeMarker)
	}
	if runtime.GOOS == "darwin" && quickstartBroadcastOption != "option-a" {
		readmeSkipDarwin(t)
	}
	blocks := extractMarkedBlocks(t, readmeDocPath, readmeMarker)
	binDir := supkvBinDir(t)
	env := docRunEnv(t, grpcurlShim(t), binDir)
	dir := t.TempDir()

	var combined strings.Builder
	for i, block := range blocks {
		name := block.Title
		if name == "" {
			name = fmt.Sprintf("block %d", i+1)
		}
		combined.WriteString(runBashVerbatim(t, dir, env, block.Body, "README.md: "+name, reapSurvivors))
	}
	out := combined.String()
	if !strings.Contains(out, "c3VwLWt2") {
		t.Fatalf("the combined output of the marked blocks does not carry the base64 value c3VwLWt2 "+
			"(the quickstart write/replication round-trip):\n%s", out)
	}
	for _, link := range []string{"docs/OPERATIONS.md", "docs/EXAMPLES.md"} {
		if !strings.Contains(string(raw), link) {
			t.Fatalf("README.md does not link %s (D-02: the README points at the docs instead of carrying ops detail)", link)
		}
	}
}
