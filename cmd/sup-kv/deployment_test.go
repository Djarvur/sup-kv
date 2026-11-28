package main_test

// The DOCS-03 deployment contract of plan 09-07: docs/DEPLOYMENT.md and
// deploy/sup-kv.service are the operator's entry point (and ship in the
// release archives), so both are machine-checked in the doc-anchor idiom.
// TestDeploymentDocBlocks reuses the executable-doc harness unchanged
// (extractMarkedBlocks, runBashVerbatim, supkvBinDir, grpcurlShim, docRunEnv —
// the TestExamplesDoc machinery of this same package): every fenced block
// marked with the deployment-verify marker runs verbatim through bash -e
// against the built binary on the reserved doc port band [27900, 28100), in
// one shared directory in doc order, and each block's captured output must
// carry its assertion-table substrings — the bidirectional contract, exactly
// like the EXAMPLES walkthrough. The demo pair's start block keeps its
// survivors (the pair itself) alive for the later block, and a cleanup
// backstop kills the pair by its PID files if a block fatals first.
// TestDeploymentUnitConsistency holds the unit to the configuration
// reference: every ExecStart flag must be a docs/CONFIGURATION.md flag-table
// row (the parsed table rows, never a --help capture), every SUPKV_
// assignment must be a documented env knob, Restart=on-failure with
// RestartPreventExitStatus=2 must be present (exit code 2 is a usage error
// and must never loop-restart — M-16's structural answer), the four required
// knobs must be carried, and every non-default ExecStart value must be
// stated in DEPLOYMENT.md's unit section.
//
// HONEST SCOPE: these tests prove the doc/unit pair's internal consistency
// and that the marked blocks run on this machine. They do not prove that the
// reader's boards behave like CI's loopback demo — the hardware, the L2
// segment and the segment broadcast address remain the operator's
// environment, and the doc says so wherever it generalizes from the demo.

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

const (
	// deploymentVerifyMarker selects the fenced blocks of docs/DEPLOYMENT.md
	// that CI executes verbatim — an explicit opt-in contract, so prose
	// fences (shell transcripts, dev notes) are never silently enrolled.
	deploymentVerifyMarker = "<!-- deployment-verify -->"

	// deploymentDocPath and unitPath are the artifacts under test, relative
	// to this package's directory — the doc and the unit are the inputs,
	// never copied fixtures.
	deploymentDocPath = "../../docs/DEPLOYMENT.md"
	unitPath          = "../../deploy/sup-kv.service"

	// minUnitFlags is the vacuity guard over the ExecStart parse: the unit
	// carries six flags today, so a parse that sees fewer is a broken parse,
	// not a small argument list.
	minUnitFlags = 6

	// minDocumentedEnvKnobs is the vacuity guard over the CONFIGURATION.md
	// env parse: the reference documents the flags' SUPKV_ twins plus the
	// five env-only knobs, so a parse that sees fewer is broken and the
	// documented-env check must not run.
	minDocumentedEnvKnobs = 20
)

// supkvEnvPattern matches one SUPKV_ knob name anywhere in a file — the
// unit's (commented or active) Environment= assignments and the
// configuration reference's documented knobs alike.
var supkvEnvPattern = regexp.MustCompile(`SUPKV_[A-Z0-9_]+`)

// requiredKnobs are the four knobs the binary refuses to start without
// (docs/CONFIGURATION.md's required-knobs rule): a unit that omits one is
// not a working unit.
var requiredKnobs = []string{"node-id", "bind-addr", "broadcast-addr", "dump-path"}

// deploymentSectionOrder is the reading order a deployer needs — the plan's
// mandated section sequence, asserted in order so a reader never meets the
// systemd unit before the topology, the addresses, the identity, the paths,
// the sizes and the time-sync contract that decide the unit's arguments.
var deploymentSectionOrder = []string{
	"## Topology",
	"## Addresses and ports",
	"## Node identity",
	"## Paths and permissions",
	"## Sizes",
	"## Time sync",
	"## Dump setup",
	"## Metrics setup",
	"## Upgrading and compatibility",
	"## Node replacement",
	"## systemd unit",
}

// demoPairTitle is the ### heading above the walkthrough's pair-start block:
// the assertion-table key and the keepSurvivors selector in one constant.
const demoPairTitle = "The two-board demo on one machine"

// deploymentAssertions is the title-keyed output contract of the deployment
// walkthrough: each entry names one marked block of docs/DEPLOYMENT.md (the
// ### heading above the marker is the key) and the substrings that block's
// captured combined output MUST carry when it runs verbatim. The quoted
// `"SERVING"` forms cannot mistake NOT_SERVING for SERVING (the M-17
// lesson), `c3VwLWt2` is the base64 of "sup-kv" read back on the second
// node, `supkv_replication_peers` is the metrics line the scrape must carry,
// and the stop line proves the pair exited. Both directions drift loudly: a
// table entry with no matching doc block fatals naming the title, and a
// marked block the table does not know fatals naming the block.
var deploymentAssertions = []struct {
	title  string
	expect []string
}{
	{title: demoPairTitle, expect: []string{`"SERVING"`, "c3VwLWt2"}},
	{title: "Check health and scrape metrics", expect: []string{`"SERVING"`, "supkv_replication_peers", "demo pair stopped"}},
}

// deploymentSkipDarwin is the option-b darwin leg, mirroring
// examplesSkipDarwin: the demo pair converges over loopback broadcast, and
// the recorded macos-15 verdict (limited broadcast dies at the sendto) makes
// this test's darwin leg a deterministic named skip, never a real run. What
// still proves the behavior: every marked block runs verbatim on the Linux
// gate cells, and the unit-consistency check runs everywhere.
func deploymentSkipDarwin(t *testing.T) {
	t.Helper()
	t.Skipf(
		"option-b decision (maintainer 2026-11-17, plan 07-01 Task 3, recorded in 07-01-SUMMARY.md " +
			"'## Broadcast probe verdict): the demo pair converges over loopback broadcast, and the macos-15 " +
			"gate cell recorded that limited broadcast dies at the sendto (no route to host), so " +
			"docs/DEPLOYMENT.md ships as a CI-verified Linux walkthrough and this test's darwin leg is a " +
			"named skip, never a real run. What still proves the behavior: every marked block runs verbatim " +
			"on the Linux gate cells, TestDeploymentUnitConsistency runs everywhere, and the phase-6 exec " +
			"scenarios prove two-node replication on Linux for real",
	)
}

// assertDeploymentSectionOrder walks the doc's `## ` headings and requires the
// mandated sequence to appear in order: a missing heading, a renamed heading
// or a reordering fails naming the sequence. The check is order-preserving
// (the mandated headings must occur as a subsequence of the document's
// headings), so extra sections are free but the deployer's reading order is
// not.
func assertDeploymentSectionOrder(t *testing.T, doc string) {
	t.Helper()
	found := 0
	for line := range strings.SplitSeq(doc, "\n") {
		if found < len(deploymentSectionOrder) && strings.TrimSpace(line) == deploymentSectionOrder[found] {
			found++
		}
	}
	if found != len(deploymentSectionOrder) {
		t.Fatalf("%s: mandated sections missing or out of order — matched %d of %d in sequence; want %s",
			deploymentDocPath, found, len(deploymentSectionOrder), strings.Join(deploymentSectionOrder, " → "))
	}
}

// unitFlag is one flag argument parsed from the unit's ExecStart line: the
// name without its dashes and the value ("" when the flag carries none).
type unitFlag struct {
	name  string
	value string
}

// unitExecStart returns the unit's ExecStart value with systemd's backslash
// continuations joined: a trailing backslash splices the next line's trimmed
// content onto the value, exactly how systemd reads the file. Commented lines
// never match (they start with `#` after trimming).
func unitExecStart(t *testing.T, unit string) string {
	t.Helper()
	lines := strings.Split(unit, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "ExecStart=") {
			continue
		}
		full := strings.TrimPrefix(line, "ExecStart=")
		for strings.HasSuffix(full, `\`) && i+1 < len(lines) {
			full = strings.TrimSuffix(full, `\`)
			i++
			full += " " + strings.TrimSpace(lines[i])
		}
		return full
	}
	t.Fatalf("%s carries no ExecStart= line — the unit does not start anything", unitPath)
	return ""
}

// parseExecStartFlags splits the joined ExecStart line into flag/value pairs.
// fields[0] is the binary path; both the space form (--flag value) and the =
// form (--flag=value) are accepted, and a flag whose next token is another
// flag (or the end of the line) carries no value.
func parseExecStartFlags(execStart string) []unitFlag {
	fields := strings.Fields(execStart)
	flags := make([]unitFlag, 0, minUnitFlags)
	for i := 1; i < len(fields); i++ {
		token := fields[i]
		if !strings.HasPrefix(token, "--") {
			continue
		}
		bare := strings.TrimPrefix(token, "--")
		if name, value, found := strings.Cut(bare, "="); found {
			flags = append(flags, unitFlag{name: name, value: value})
			continue
		}
		value := ""
		if i+1 < len(fields) && !strings.HasPrefix(fields[i+1], "--") {
			value = fields[i+1]
			i++
		}
		flags = append(flags, unitFlag{name: bare, value: value})
	}
	return flags
}

// execStartValueStated reports whether the doc states the unit's `--flag
// value` as a standalone argument: the exact literal with a boundary after
// it, so `--node-id 11` can never satisfy a check for `--node-id 1`.
func execStartValueStated(doc, name, value string) bool {
	pattern := regexp.QuoteMeta("--"+name+" "+value) + `(?:\s|$)`
	return regexp.MustCompile(pattern).MatchString(doc)
}

// TestDeploymentDocBlocks is the deployment walkthrough's machine contract:
// every deployment-verify block of docs/DEPLOYMENT.md runs verbatim through
// bash -e against the built binary with the pinned grpcurl on PATH, the
// blocks share one working directory in doc order, and each block's captured
// output must carry its assertion-table substrings — the pair's two boards
// both report SERVING, the write lands on the second node, the metrics
// scrape carries the families, and the pair stops. The mandated section
// order is asserted before any block runs, so the reading order is part of
// the contract.
//
// The walkthrough is serialized on purpose (the TestExamplesDoc precedent):
// it drives bash -e plus the `go tool grpcurl` shim, its two blocks share one
// working directory and the reserved doc port band [27900, 28100), and the
// pair-start block's survivors are the later block's state.
//
//nolint:paralleltest // serialized on purpose: the two blocks share one working directory and the reserved doc port band, and the pair-start block's survivors are the later block's state (doc comment).
func TestDeploymentDocBlocks(t *testing.T) {
	raw, err := os.ReadFile(deploymentDocPath)
	if err != nil {
		t.Fatalf("read %s: %v — the deployment guide is the DOCS-03 contract this package executes", deploymentDocPath, err)
	}
	doc := string(raw)
	assertDeploymentSectionOrder(t, doc)
	if runtime.GOOS == "darwin" && examplesBroadcastOption != "option-a" {
		deploymentSkipDarwin(t)
	}
	blocks := extractMarkedBlocks(t, deploymentDocPath, deploymentVerifyMarker)
	byTitle := make(map[string]docBlock, len(blocks))
	for _, block := range blocks {
		if block.Title == "" {
			t.Fatalf("%s: a marked block has no ### heading above it — titled blocks are the assertion-table keys", deploymentDocPath)
		}
		byTitle[block.Title] = block
	}
	for _, a := range deploymentAssertions {
		if _, ok := byTitle[a.title]; !ok {
			t.Fatalf("%s has no marked block titled %q — the walkthrough is missing the block (the assertion table keys the doc's ### titles exactly)",
				deploymentDocPath, a.title)
		}
	}
	expectByTitle := make(map[string][]string, len(deploymentAssertions))
	for _, a := range deploymentAssertions {
		expectByTitle[a.title] = a.expect
	}
	binDir := supkvBinDir(t)
	env := docRunEnv(t, grpcurlShim(t), binDir)
	// One shared directory for every block: the pair's dump paths and the
	// PID files are relative, so the doc-order state carries.
	dir := t.TempDir()
	// CI-hygiene backstop: the pair-start block deliberately leaves its two
	// nodes alive for the later block, and the stop block reaps them by the
	// PID files in the happy path. If any block fatals before the stop block
	// runs, this cleanup kills the pair by those same PID files — best-effort
	// (before the start block the files do not exist yet), never fatal.
	t.Cleanup(func() {
		for _, pidFile := range []string{"deploy-a.pid", "deploy-b.pid"} {
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
				deploymentDocPath, block.Title)
		}
		// The pair-start block's survivors (the demo pair itself) are the
		// walkthrough's shared state — every other block reaps its own
		// background jobs, so only this one keeps them.
		survivors := reapSurvivors
		if block.Title == demoPairTitle {
			survivors = keepSurvivors
		}
		out := runBashVerbatim(t, dir, env, block.Body, "DEPLOYMENT.md: "+block.Title, survivors)
		for _, substr := range expect {
			if !strings.Contains(out, substr) {
				t.Fatalf("DEPLOYMENT.md block %q: combined output does not carry %q:\n%s", block.Title, substr, out)
			}
		}
	}
}

// TestDeploymentUnitConsistency is the M-16 unit's machine contract:
// deploy/sup-kv.service exists, restarts failures but never a usage error,
// and names only knobs the configuration reference documents. Every ExecStart
// flag must be a docs/CONFIGURATION.md flag-table row; every SUPKV_
// assignment (the commented optional knobs included) must be a documented env
// knob; the four required knobs must be carried; and every non-default
// ExecStart value must be stated in docs/DEPLOYMENT.md's unit section — the
// unit may deviate from a documented default, but only out loud.
func TestDeploymentUnitConsistency(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("read %s: %v — the M-16 systemd unit is the deployment story's working artifact", unitPath, err)
	}
	unit := string(raw)
	if !strings.Contains(unit, "Restart=on-failure") {
		t.Fatalf("%s does not set Restart=on-failure — exit codes 1 and 3 must restart (docs/OPERATIONS.md ## Lifecycle and exit codes)", unitPath)
	}
	if !strings.Contains(unit, "RestartPreventExitStatus=2") {
		t.Fatalf("%s does not set RestartPreventExitStatus=2 — a usage error (exit code 2) must never loop-restart (M-16)", unitPath)
	}
	execStart := unitExecStart(t, unit)
	fields := strings.Fields(execStart)
	if !strings.HasPrefix(fields[0], "/") {
		t.Fatalf("%s ExecStart must name an absolute binary path, got %q — systemd requires it", unitPath, fields[0])
	}
	flags := parseExecStartFlags(execStart)
	if len(flags) < minUnitFlags {
		t.Fatalf("the %s ExecStart parse saw %d flags, want at least %d — the consistency check would pass vacuously below that",
			unitPath, len(flags), minUnitFlags)
	}

	conf, err := os.ReadFile(configurationDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", configurationDocPath, err)
	}
	rows := parseDocFlagRows(string(conf))
	rowByName := make(map[string]string, len(rows))
	for _, r := range rows {
		rowByName[r.name] = r.row
	}
	documentedEnv := make(map[string]bool)
	for _, name := range supkvEnvPattern.FindAllString(string(conf), -1) {
		documentedEnv[name] = true
	}
	if len(documentedEnv) < minDocumentedEnvKnobs {
		t.Fatalf("the %s env parse saw %d SUPKV_ knobs, want at least %d — the documented-env check would pass vacuously below that",
			configurationDocPath, len(documentedEnv), minDocumentedEnvKnobs)
	}
	deploy, err := os.ReadFile(deploymentDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", deploymentDocPath, err)
	}
	deployDoc := string(deploy)

	seen := make(map[string]bool, len(flags))
	for _, f := range flags {
		if seen[f.name] {
			t.Fatalf("%s ExecStart repeats --%s — one argument per knob", unitPath, f.name)
		}
		seen[f.name] = true
		if _, ok := rowByName[f.name]; !ok {
			t.Fatalf("%s ExecStart names --%s, which docs/CONFIGURATION.md has no flag-table row for — the unit may name only documented flags", unitPath, f.name)
		}
	}
	for _, knob := range requiredKnobs {
		if !seen[knob] {
			t.Fatalf("%s ExecStart omits --%s — one of the required knobs the binary refuses to start without (the required-knobs rule)", unitPath, knob)
		}
	}
	unitEnv := supkvEnvPattern.FindAllString(unit, -1)
	if len(unitEnv) == 0 {
		t.Fatalf("%s names no SUPKV_ knob — the optional-knob contract (SUPKV_LOG_LEVEL, the metrics TLS triple) is gone", unitPath)
	}
	for _, name := range unitEnv {
		if !documentedEnv[name] {
			t.Fatalf("%s assigns %s, which docs/CONFIGURATION.md does not document — the unit may set only documented env knobs", unitPath, name)
		}
	}
	for _, f := range flags {
		if f.value == "" {
			continue
		}
		defval := docRowCell(t, rowByName[f.name], 3)
		if defval == f.value && defval != "none" && !strings.Contains(defval, "no default") {
			continue // the unit runs the documented default — nothing to state
		}
		if !execStartValueStated(deployDoc, f.name, f.value) {
			t.Fatalf("%s runs --%s %s (documented default %q) and docs/DEPLOYMENT.md does not state the value — every non-default ExecStart value must be stated in the unit section",
				unitPath, f.name, f.value, defval)
		}
	}
}
