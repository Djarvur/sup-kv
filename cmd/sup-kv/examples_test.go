package main_test

// The executable-doc tracer of Phase 7 (D-04/D-05): docs/EXAMPLES.md is the
// test input — every fenced block marked with the examples-verify marker is
// extracted and executed VERBATIM through bash -e against the real binary
// (the TestMain build-once harness), with a PATH under which `sup-kv`
// resolves to a copy of the built binary and `grpcurl` resolves to a shim
// exec'ing the pinned go.mod tool directive (v1.9.4 — never a PATH-resolved
// stranger binary). The seed block proves the whole seam end to end: the
// gRPC reflection registration (grpcurl list enumerates
// supkvapi.v1.SupKVService with zero proto files) plus a Set/Get round-trip
// whose captured output carries the base64 value. The doc cannot drift from
// reality after release: a broken example fails CI. Marked blocks use the
// fixed doc ports of the reserved [27900, 28100) band (scenarioPortLo sits
// above it, so scenario port picks can never collide), start their node as a
// background job, gate on a bounded startup retry and clean up after
// themselves; the runner additionally reaps the block's whole process group
// after the run, so a failing block never leaks a node into the next one.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
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

// runBashVerbatim executes one doc block with bash -e in dir: the block
// arrives on stdin, combined stdout/stderr is captured and returned, and any
// failure is fatal with the block and its output in the message — the doc's
// commands must succeed exactly as a stranger would run them. Output is
// captured to a file, not a pipe: a block's background node INHERITS the
// bash-side descriptors, and with a pipe the exec package waits for every
// writer to close before Run returns — a node the block left running (bash -e
// aborts on the first failing command, before the block's own cleanup) would
// hang the run until the test timeout. The bash runs as its own process
// group and the group is SIGKILLed once bash has exited, so a failed block
// cannot leak its node into the next run either way; the file is read after
// the kill, when no writer is left, so its content is stable.
func runBashVerbatim(t *testing.T, dir string, env []string, block, what string) string {
	t.Helper()
	outFile, err := os.CreateTemp(t.TempDir(), "doc-block-output-*")
	if err != nil {
		t.Fatalf("create the output capture file: %v", err)
	}
	cmd := exec.CommandContext(t.Context(), "bash", "-e")
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin = strings.NewReader(block + "\n")
	cmd.Stdout, cmd.Stderr = outFile, outFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	runErr := cmd.Run()
	if cmd.Process != nil {
		// CI hygiene: reap the whole group (bash is gone; this is its
		// left-behind background jobs).
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) //nolint:errcheck // best-effort reaping of an already-finished group.
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
	if runErr != nil {
		t.Fatalf("%s failed: %v\n--- block ---\n%s\n--- output ---\n%s", what, runErr, block, out)
	}
	return out
}

// supkvBinDir copies the built binary into a fresh temp dir named `sup-kv`
// and returns the dir: a doc block run with this dir on PATH resolves
// `sup-kv` exactly the way a stranger with the release archive would.
func supkvBinDir(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(smokeBinary(t))
	if err != nil {
		t.Fatalf("read the built binary: %v", err)
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "sup-kv")
	err = os.WriteFile(bin, data, 0o755)
	if err != nil {
		t.Fatalf("write %s: %v", bin, err)
	}
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
// inherited environment with PATH replaced by shimDir, binDir and the
// existing PATH in that order. PATH is replaced (not appended) — a duplicate
// PATH entry would leave the FIRST (unshimmed) value winning in the child.
func docRunEnv(t *testing.T, shimDir, binDir string) []string {
	t.Helper()
	augmented := strings.Join(
		[]string{shimDir, binDir, os.Getenv("PATH")},
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

// TestExamplesDoc is the Phase 7 tracer for executable documentation: every
// examples-verify block of docs/EXAMPLES.md runs verbatim through bash -e
// against the built binary with the pinned grpcurl on PATH, and the combined
// output must prove the seam the doc claims — the reflection listing names
// supkvapi.v1.SupKVService, and the Set/Get round-trip returns the base64
// value the doc writes. 07-03 grows the doc; this test grows with it.
func TestExamplesDoc(t *testing.T) {
	t.Parallel()
	blocks := extractMarkedBlocks(t, examplesDocPath, examplesMarker)
	binDir := supkvBinDir(t)
	env := docRunEnv(t, grpcurlShim(t), binDir)

	var combined strings.Builder
	for i, block := range blocks {
		name := block.Title
		if name == "" {
			name = fmt.Sprintf("block %d", i+1)
		}
		combined.WriteString(runBashVerbatim(t, binDir, env, block.Body, "EXAMPLES.md: "+name))
	}
	out := combined.String()
	if !strings.Contains(out, "supkvapi.v1.SupKVService") {
		t.Fatalf("the combined output of the marked blocks does not name supkvapi.v1.SupKVService (reflection list):\n%s", out)
	}
	if !strings.Contains(out, "c3VwLWt2") {
		t.Fatalf("the combined output of the marked blocks does not carry the base64 value c3VwLWt2 (the Set/Get round-trip):\n%s", out)
	}
}
