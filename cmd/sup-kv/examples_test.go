package main_test

// The executable-doc contract of Phase 7 (D-04/D-05): docs/EXAMPLES.md is
// the test input — every fenced block marked with the examples-verify marker
// is extracted and executed VERBATIM through bash -e against the real binary
// (the TestMain build-once harness), with a PATH under which `sup-kv`
// resolves to a copy of the built binary and `grpcurl` resolves to a shim
// exec'ing the pinned go.mod tool directive (v1.9.4 — never a PATH-resolved
// stranger binary). The blocks run in doc order and share one working
// directory, so the walkthrough's state carries across blocks exactly as a
// stranger following the page top to bottom would see it: the pair starts in
// the first block and stays up until the final cleanup block, the
// Get-before-Set NotFound is documented behavior, and the Watch block's
// stream file is written and read inside its own block. Every block carries
// a ### title, the title keys the assertion table, and each block's captured
// combined output MUST carry the table's substrings — the walkthrough is not
// "full" until every RPC appears with an asserted output. Both directions
// drift loudly: a table entry with no matching doc block fatals naming the
// title, and a marked block the table does not know fatals naming the block.
// Marked blocks use the fixed doc ports of the reserved [27900, 28100) band
// (scenarioPortLo sits above it, so scenario port picks can never collide),
// start their nodes as background jobs, gate on bounded retries whose final
// command makes exhaustion fatal, and clean up after themselves; the runner
// additionally reaps the block's whole process group after the run, so a
// failing block never leaks a node into the next one.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// examplesMarker selects the fenced blocks of docs/EXAMPLES.md that CI
// executes verbatim — an explicit opt-in contract, so future prose fences
// (shell output, dev notes) are never silently enrolled.
const examplesMarker = "<!-- examples-verify -->"

// examplesDocPath is the doc under test, relative to this package's
// directory — the doc itself is the input, never a copied fixture (D-05).
const examplesDocPath = "../../docs/EXAMPLES.md"

// docBlock is one marked fenced block of a doc: the body between the fences
// and the title of the nearest preceding ### heading (empty when none).
type docBlock struct {
	Title string
	Body  string
}

// extractMarkedBlocks returns the fenced blocks that a marker line selects,
// scanning the doc as plain lines: a marker takes the NEXT fenced block,
// the body is the lines between the fences, and the title is the nearest
// ### heading above the marker (empty when there is none). Zero blocks is a
// failure, not an empty result — the doc IS the contract (D-06's extraction
// discipline); an unclosed fence after a marker is likewise fatal.
func extractMarkedBlocks(t *testing.T, path, marker string) []docBlock {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	lines := strings.Split(string(raw), "\n")
	isFence := func(line string) bool { return strings.HasPrefix(strings.TrimSpace(line), "```") }
	var blocks []docBlock
	title := ""
	for i, line := range lines {
		if !strings.Contains(line, marker) {
			if strings.HasPrefix(line, "### ") {
				title = strings.TrimSpace(strings.TrimPrefix(line, "###"))
			}
			continue
		}
		j := i + 1
		for j < len(lines) && !isFence(lines[j]) {
			j++
		}
		if j >= len(lines) {
			t.Fatalf("%s: marker at line %d is not followed by a fenced block", path, i+1)
		}
		open := j
		j++
		var body []string
		for j < len(lines) && !isFence(lines[j]) {
			body = append(body, lines[j])
			j++
		}
		if j >= len(lines) {
			t.Fatalf("%s: fenced block opened at line %d (after the marker at line %d) is never closed", path, open+1, i+1)
		}
		blocks = append(blocks, docBlock{Title: title, Body: strings.Join(body, "\n")})
	}
	if len(blocks) == 0 {
		t.Fatalf("%s carries no %q block — the executable-doc contract is gone", path, marker)
	}
	return blocks
}

// blockSurvivors says what happens to a block's background jobs once its
// bash has exited.
type blockSurvivors int

const (
	// reapSurvivors SIGKILLs the block's whole process group after bash
	// exits — the default contract: a block cleans up after itself, and
	// anything left behind is a leak.
	reapSurvivors blockSurvivors = iota
	// keepSurvivors leaves a SUCCESSFUL block's background jobs alive: the
	// 07-03 walkthrough's pair-start block launches the two demo nodes that
	// every later block (and the final stop block, by PID file) shares, so
	// its survivors are state, not leaks. A failed block still reaps — its
	// half-started leftovers are leaks nothing can use.
	keepSurvivors
)

// docBlockBudget bounds one verbatim doc block: the README/EXAMPLES
// walkthroughs take single-digit seconds locally, so a minute per block is
// generous headroom while a wedged node/grpcurl wait still fails with this
// test's own deadline instead of the package-wide `go test` timeout — the
// T-11 README-deadline flake class, where -count=20 multiplies every run
// and a hang must not consume the whole soak. The effective deadline is
// clamped 250ms inside t.Deadline() when the test has one (the
// main_test.go waitFor pattern), so an explicit -timeout also wins.
const docBlockBudget = time.Minute

// runBashVerbatim executes one doc block with bash -e in dir: the block
// arrives on stdin, combined stdout/stderr is captured and returned, and any
// failure is fatal with the block and its output in the message — the doc's
// commands must succeed exactly as a stranger would run them. Each block is
// bounded by docBlockBudget (clamped inside t.Deadline()): a wedged block
// fails here, in its own run, with the block and captured output attached.
// Output is captured to a file, not a pipe: a block's background node
// INHERITS the bash-side descriptors, and with a pipe the exec package waits
// for every writer to close before Run returns — a node the block left
// running (bash -e aborts on the first failing command, before the block's
// own cleanup) would hang the run until the test timeout. The bash runs as
// its own process group and the group is SIGKILLed once bash has exited, so
// a failed block cannot leak its node into the next run either way; the file
// is read after the kill, when no writer is left, so its content is stable.
func runBashVerbatim(t *testing.T, dir string, env []string, block, what string, survivors blockSurvivors) string {
	t.Helper()
	blockDeadline := time.Now().Add(docBlockBudget)
	if hard, ok := t.Deadline(); ok {
		if clamp := hard.Add(-hardDeadlineMargin); clamp.Before(blockDeadline) {
			blockDeadline = clamp
		}
	}
	ctx, cancel := context.WithDeadline(t.Context(), blockDeadline)
	defer cancel()
	outFile, err := os.CreateTemp(t.TempDir(), "doc-block-output-*")
	if err != nil {
		t.Fatalf("create the output capture file: %v", err)
	}
	cmd := exec.CommandContext(ctx, "bash", "-e")
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin = strings.NewReader(block + "\n")
	cmd.Stdout, cmd.Stderr = outFile, outFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	runErr := cmd.Run()
	if cmd.Process != nil && (survivors == reapSurvivors || runErr != nil) {
		// CI hygiene: reap the whole group (bash is gone; this is its
		// left-behind background jobs). The kill is best-effort — an
		// already-empty group reports ESRCH, which is the success case.
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	closeErr := outFile.Close()
	if closeErr != nil {
		t.Fatalf("close the output capture file: %v", closeErr)
	}
	raw, err := os.ReadFile(outFile.Name())
	if err != nil {
		t.Fatalf("read the captured output: %v", err)
	}
	out := string(raw)
	ctxErr := ctx.Err()
	if ctxErr != nil {
		t.Fatalf("%s exceeded its doc-block deadline (%v): %v\n--- block ---\n%s\n--- output ---\n%s", what, docBlockBudget, ctxErr, block, out)
	}
	if runErr != nil {
		t.Fatalf("%s failed: %v\n--- block ---\n%s\n--- output ---\n%s", what, runErr, block, out)
	}
	return out
}

// writeSupkvBinary copies the built binary to dir/sup-kv, ready to exec: a
// doc block whose cwd is dir then resolves `./sup-kv` exactly the way a
// stranger who ran the README's `go build -o sup-kv ./cmd/sup-kv` in their
// checkout would — the binary the doc's own build step produces, at the path
// the doc's blocks invoke, with no PATH entry doing the resolving.
func writeSupkvBinary(t *testing.T, dir string) {
	t.Helper()
	data, err := os.ReadFile(smokeBinary(t))
	if err != nil {
		t.Fatalf("read the built binary: %v", err)
	}
	bin := filepath.Join(dir, "sup-kv")
	err = os.WriteFile(bin, data, 0o755)
	if err != nil {
		t.Fatalf("write %s: %v", bin, err)
	}
}

// supkvBinDir returns a fresh temp dir carrying the built binary named
// `sup-kv`: a doc block run with this dir on PATH resolves `sup-kv` exactly
// the way a stranger with the release archive would — the contract
// docs/EXAMPLES.md declares ("two tools on your PATH").
func supkvBinDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeSupkvBinary(t, dir)
	return dir
}

// grpcurlShim writes a `grpcurl` shim into a fresh temp dir and returns the
// dir: put on PATH ahead of everything else, the doc's plain `grpcurl`
// invocations exec the pinned go.mod tool directive (v1.9.4) instead of
// whatever grpcurl may or may not be installed on the machine. The shim
// shells out through `go -C <modroot> tool grpcurl` because it runs with a
// cwd outside the module — the -C flag is what makes the tool directive
// resolve.
func grpcurlShim(t *testing.T) string {
	t.Helper()
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("probe exec.LookPath(go) for the grpcurl shim: %v", err)
	}
	modroot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolve the module root for the grpcurl shim: %v", err)
	}
	dir := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nexec %q -C %q tool grpcurl \"$@\"\n", goTool, modroot)
	shim := filepath.Join(dir, "grpcurl")
	err = os.WriteFile(shim, []byte(script), 0o755)
	if err != nil {
		t.Fatalf("write the grpcurl shim: %v", err)
	}
	return dir
}

// docRunEnv builds the child environment for a verbatim doc block: the
// inherited environment with PATH replaced by the given dirs followed by the
// existing PATH, in that order. PATH is replaced (not appended) — a duplicate
// PATH entry would leave the FIRST (unshimmed) value winning in the child.
// The README quickstart passes only the grpcurl shim dir: its blocks invoke
// the binary as `./sup-kv` from the block cwd (writeSupkvBinary mirrors the
// doc's build step), and a sup-kv dir on PATH would resolve a bare-name
// regression the doc's stranger would hit as command-not-found — WR-01.
func docRunEnv(t *testing.T, dirs ...string) []string {
	t.Helper()
	augmented := strings.Join(
		slices.Concat(dirs, []string{os.Getenv("PATH")}),
		string(os.PathListSeparator),
	)
	env := make([]string, 0, len(os.Environ())+1)
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "PATH=") {
			continue
		}
		env = append(env, kv)
	}
	return append(env, "PATH="+augmented)
}

// examplesBroadcastOption encodes the recorded 07-01 broadcast verdict for
// the examples pair (same contract as quickstartBroadcastOption in
// readme_test.go): option-b — the walkthrough's two-node loopback-broadcast
// topology is CI-verified on the Linux gate cells, and this test's darwin
// leg is a deterministic named skip carrying the recorded macos-15 evidence.
// Only a future probe proving loopback-broadcast fan-out on darwin (recorded
// as option-a) would remove the skip; keep this constant in lockstep.
const examplesBroadcastOption = "option-b"

// examplesSkipDarwin is the option-b darwin leg: a deterministic loud skip
// (it fires before any block runs — determinism of the macos-15 gate cell
// beats re-gathering evidence the probe verdict already recorded) naming the
// probe, the recorded verdict, and what still proves the behavior.
func examplesSkipDarwin(t *testing.T) {
	t.Helper()
	t.Skipf(
		"option-b decision (maintainer 2026-11-17, plan 07-01 Task 3, recorded in 07-01-SUMMARY.md " +
			"'## Broadcast probe verdict'): the demo pair converges over loopback broadcast, and the macos-15 " +
			"gate cell of PR #32 (run 44083351667, job 134039317468, 2026-11-17) recorded that limited " +
			"broadcast dies at the sendto (no route to host) while phase 6 proved lo0 carries no BROADCAST " +
			"flag, so docs/EXAMPLES.md ships as a CI-verified Linux walkthrough and this test's darwin leg " +
			"is a named skip, never a real run. What still proves the behavior: every marked block runs " +
			"verbatim on the Linux gate cells, and the phase-6 exec scenarios (plan 06-07) prove two-node " +
			"replication on Linux for real",
	)
}

// examplesAssertions is the title-keyed output contract of the walkthrough:
// each entry names one marked block of docs/EXAMPLES.md (the ### heading
// above the marker is the key) and the substrings that block's captured
// combined output MUST carry when it runs verbatim. Together the titles are
// the every-RPC set of supkvapi.v1.SupKVService plus discovery and health —
// the walkthrough is not full while a title is missing, and an executed
// block without a table entry has no asserted output and is refused.
var examplesAssertions = []struct {
	title  string
	expect []string
}{
	{title: "Start the demo pair", expect: []string{"SERVING"}},
	{title: "Discover the API", expect: []string{"supkvapi.v1.SupKVService"}},
	{title: "Read a missing key", expect: []string{"NotFound"}},
	{title: "Write and read", expect: []string{"demo/hello", "c3VwLWt2"}},
	// The wait entry's substrings are the REAL grpcurl bytes: pretty-printed
	// JSON with camelCase protojson names ("acked": true, ackedCount,
	// totalPeers) — captured from a run, not hand-written (Pitfall 3/5).
	{title: "Wait for replication", expect: []string{`"acked": true`, "ackedCount", "totalPeers"}},
	{title: "Delete and list", expect: []string{"demo/wait"}},
	{title: "Watch a stream", expect: []string{"demo/live"}},
	{title: "Check health", expect: []string{"SERVING"}},
	{title: "Snapshot round-trip", expect: []string{"{}"}},
	{title: "Stop the pair", expect: []string{"demo pair stopped"}},
}

// TestExamplesDoc is the DOCS-03 machine contract: every examples-verify
// block of docs/EXAMPLES.md runs verbatim through bash -e against the built
// binary with the pinned grpcurl on PATH, the blocks share one working
// directory in doc order, and each block's captured output must carry its
// assertion-table substrings — the wait_for_replication response acks true
// with non-zero acked_count/total_peers, the cross-node read returns the
// base64 value the doc writes, the missing-key Get surfaces the NotFound
// error text, and the Watch stream carries the live event key. grpcurl
// omits zero-valued fields, so every asserted field is non-zero in the
// example flow (the zero-valued shapes are documented in the doc's prose).
//
// The walkthrough is serialized on purpose: it drives bash -e plus the
// `go tool grpcurl` shim and competes with this package's process-spawning
// tests, so running it beside those siblings produced empty tool captures
// under compile load (04-protocol/deferred-items.md). The per-block budget
// (docBlockBudget) and the process-group reap still bound a genuinely
// wedged block; only the explicit parallel marker is gone.
//
//nolint:paralleltest // serialized on purpose: the grpcurl walkthrough competes with the package's process-spawning tests and stalls under compile load (doc comment).
func TestExamplesDoc(t *testing.T) {
	raw, err := os.ReadFile(examplesDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", examplesDocPath, err)
	}
	if !strings.Contains(string(raw), examplesMarker) {
		t.Fatalf("docs/EXAMPLES.md carries no %s block — the DOCS-03 contract is gone", examplesMarker)
	}
	if runtime.GOOS == "darwin" && examplesBroadcastOption != "option-a" {
		examplesSkipDarwin(t)
	}
	blocks := extractMarkedBlocks(t, examplesDocPath, examplesMarker)
	byTitle := make(map[string]docBlock, len(blocks))
	for _, block := range blocks {
		if block.Title == "" {
			t.Fatalf("%s: a marked block has no ### heading above it — titled blocks are the assertion-table keys", examplesDocPath)
		}
		byTitle[block.Title] = block
	}
	for _, a := range examplesAssertions {
		if _, ok := byTitle[a.title]; !ok {
			t.Fatalf("%s has no marked block titled %q — the walkthrough is missing the block (the assertion table keys the doc's ### titles exactly)",
				examplesDocPath, a.title)
		}
	}
	expectByTitle := make(map[string][]string, len(examplesAssertions))
	for _, a := range examplesAssertions {
		expectByTitle[a.title] = a.expect
	}
	binDir := supkvBinDir(t)
	env := docRunEnv(t, grpcurlShim(t), binDir)
	// One shared directory for every block: the pair's dump paths and the
	// blocks' scratch files are relative, so the doc-order state carries.
	dir := t.TempDir()
	// CI-hygiene backstop: the pair-start block deliberately leaves its two
	// nodes alive for the later blocks, and the stop block reaps them by the
	// PID files in the happy path. If any block fatals before the stop block
	// runs, this cleanup kills the pair by those same PID files — best-effort
	// (before the start block the files do not exist yet), never fatal.
	t.Cleanup(func() {
		for _, pidFile := range []string{"ex-a.pid", "ex-b.pid"} {
			pidBytes, readErr := os.ReadFile(filepath.Join(dir, pidFile))
			if readErr != nil {
				continue
			}
			pid, parseErr := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
			if parseErr != nil || pid <= 0 {
				continue
			}
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	})
	for _, block := range blocks {
		expect, ok := expectByTitle[block.Title]
		if !ok {
			t.Fatalf("%s: marked block %q is not in the assertion table — every executed block must carry an asserted output",
				examplesDocPath, block.Title)
		}
		// The pair-start block's survivors (the demo pair itself) are the
		// walkthrough's shared state — every other block reaps its own
		// background jobs, so only this one keeps them.
		survivors := reapSurvivors
		if block.Title == "Start the demo pair" {
			survivors = keepSurvivors
		}
		out := runBashVerbatim(t, dir, env, block.Body, "EXAMPLES.md: "+block.Title, survivors)
		for _, substr := range expect {
			if !strings.Contains(out, substr) {
				t.Fatalf("EXAMPLES.md block %q: combined output does not carry %q:\n%s", block.Title, substr, out)
			}
		}
	}
}
