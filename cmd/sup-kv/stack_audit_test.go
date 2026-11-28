package main_test

// The DOCS-08 stack-audit guard (09-03): AGENTS.md's and .claude/CLAUDE.md's
// stack claims are machine-audited against go.mod and `go list ./...` — the
// Go directive string, the backticked package paths, the tool-directive
// inventory and the module package inventory are read from the module at
// test time, never hand-typed, so a dependency or package change without a
// doc update fails `make test` (D-10's stack-audit half). HONEST SCOPE: this
// is an inventory and stale-claim audit only — it proves every repo path the
// docs cite exists, the tools and packages the docs name are the module's
// (and the module's are the docs', both directions), and the superseded
// v1.0/pre-implementation sentences are gone; it does not prove the prose is
// well-written or that the docs state every property of what they name.

import (
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const (
	agentsDocPath = "../../AGENTS.md"
	claudeDocPath = "../../.claude/CLAUDE.md"
	goModPath     = "../../go.mod"
	goListDir     = "../.." // the test binary's cwd is cmd/sup-kv; the module root is two levels up

	stackStartMarker = "GSD:stack-start"
	stackEndMarker   = "GSD:stack-end"

	// Vacuity floors: a scan that finds nothing would pass vacuously, so
	// each doc must cite at least this many paths before the audit believes
	// it, and `go list ./...` must return a real inventory.
	minAgentsPaths    = 15
	minStackPaths     = 5
	minModulePackages = 10
)

// moduleInventory is what the docs are compared against, read from go.mod at
// test time — the expected values are derived, never typed into the test.
type moduleInventory struct {
	goDirective string   // the `go` directive's version string, e.g. 1.27.4
	toolNames   []string // the tool block's short command names, e.g. buf, govulncheck
}

var (
	goDirectivePattern = regexp.MustCompile(`(?m)^go (\S+)$`)
	modulePathPattern  = regexp.MustCompile(`(?m)^module (\S+)$`)
	toolNamePattern    = regexp.MustCompile(`^[a-z0-9-]+$`)
	versionSegPattern  = regexp.MustCompile(`^v\d+$`)
	backtickPattern    = regexp.MustCompile("`([^`]+)`")
)

func readModuleInventory(t *testing.T) moduleInventory {
	t.Helper()
	raw, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("read %s: %v", goModPath, err)
	}
	text := string(raw)
	dir := goDirectivePattern.FindStringSubmatch(text)
	if dir == nil {
		t.Fatalf("%s declares no `go` directive — the audit has no expected version to compare against", goModPath)
	}
	return moduleInventory{
		goDirective: dir[1],
		toolNames:   toolShortNames(t, text),
	}
}

// toolShortNames derives the tool block's short command names — the last
// path segment that is not a major-version suffix (…/goreleaser/v2 →
// goreleaser) — so the docs' tool inventory is compared against the module's
// without hand-copying the list.
func toolShortNames(t *testing.T, goMod string) []string {
	t.Helper()
	var names []string
	inBlock := false
	for line := range strings.SplitSeq(goMod, "\n") {
		switch {
		case strings.HasPrefix(line, "tool ("):
			inBlock = true
		case inBlock && strings.HasPrefix(strings.TrimSpace(line), ")"):
			inBlock = false
		case inBlock:
			path := strings.TrimSpace(line)
			if path == "" {
				continue
			}
			names = append(names, toolShortName(path))
		}
	}
	if len(names) == 0 {
		t.Fatalf("%s declares no `tool` block entries — the audit has no expected tool inventory", goModPath)
	}
	return names
}

// toolShortName strips a tool module path down to its command name: the last
// segment that is not a v<major> suffix (…/cmd/golangci-lint →
// golangci-lint; …/goreleaser/v2 → goreleaser).
func toolShortName(path string) string {
	parts := strings.Split(path, "/")
	for _, part := range slices.Backward(parts) {
		if versionSegPattern.MatchString(part) {
			continue
		}
		return part
	}
	return path
}

// goListPackages runs `go list ./...` at the module root — the audit's
// package inventory source, executed at test time so the expectation moves
// with the module instead of freezing the planning-time snapshot.
func goListPackages(t *testing.T) []string {
	t.Helper()
	mod := modulePathPattern.FindStringSubmatch(readGoMod(t))
	if mod == nil {
		t.Fatalf("%s declares no `module` directive", goModPath)
	}
	cmd := exec.CommandContext(t.Context(), "go", "list", "./...")
	cmd.Dir = goListDir // the test binary's cwd is cmd/sup-kv; the inventory source is the module root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("`go list ./...` at %s failed: %v", goListDir, err)
	}
	var pkgs []string
	for line := range strings.SplitSeq(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pkgs = append(pkgs, strings.TrimPrefix(line, mod[1]+"/"))
	}
	if len(pkgs) < minModulePackages {
		t.Fatalf("`go list ./...` returned %d packages — below the %d-package floor, the inventory comparison would pass vacuously", len(pkgs), minModulePackages)
	}
	return pkgs
}

func readGoMod(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("read %s: %v", goModPath, err)
	}
	return string(raw)
}

// citedToolNames extracts the tool short names a doc cites on its
// tool-directive lines: backticked lowercase tokens on every line whose
// text names the tool directives, after dropping the structural tokens
// (`tool` itself, the go files, module paths). The phrase scopes the
// inventory sentence, so ordinary prose backticks elsewhere never count.
func citedToolNames(lines []string) []string {
	var names []string
	for _, line := range lines {
		if !strings.Contains(strings.ReplaceAll(line, "`", ""), "tool directive") {
			continue
		}
		for _, match := range backtickPattern.FindAllStringSubmatch(line, -1) {
			token := match[1]
			if strings.ContainsAny(token, " /*") || strings.Contains(token, ".") {
				continue
			}
			switch token {
			case "tool", "go":
				continue
			}
			if !toolNamePattern.MatchString(token) {
				continue
			}
			names = append(names, token)
		}
	}
	return names
}

// citedPaths extracts the doc's backticked repo paths (internal/…, cmd/…,
// api/…) whose existence the audit checks; glob patterns such as `api/**`
// are not concrete path claims and are skipped.
func citedPaths(text string) []string {
	var paths []string
	for _, match := range backtickPattern.FindAllStringSubmatch(text, -1) {
		token := match[1]
		for _, prefix := range []string{"internal/", "cmd/", "api/"} {
			if strings.HasPrefix(token, prefix) && !strings.ContainsAny(token, " *") {
				paths = append(paths, token)
				break
			}
		}
	}
	return paths
}

// requirePathsExist asserts every cited repo path exists in the tree, after
// a vacuity floor: a doc that names no paths would make the existence check
// pass for nothing.
func requirePathsExist(t *testing.T, docName string, paths []string, floor int) {
	t.Helper()
	if len(paths) < floor {
		t.Fatalf("%s cites only %d repo paths — below the %d-path floor, the existence check would pass vacuously", docName, len(paths), floor)
	}
	for _, path := range paths {
		_, statErr := os.Stat("../../" + strings.TrimSuffix(path, "/"))
		if statErr != nil {
			t.Errorf("%s cites repo path %q, which does not exist in the tree — an imagined package (T-09-07)", docName, path)
		}
	}
}

// compareToolInventories asserts the doc's cited tool set equals go.mod's
// derived tool set — both directions: a cited tool the module lacks is an
// imagined dependency, a module tool the doc omits is a stale inventory
// (D-10's count comparison).
func compareToolInventories(t *testing.T, docName string, cited, derived []string) {
	t.Helper()
	if len(cited) == 0 {
		t.Errorf("%s cites no tool names on its tool-directive lines — the tool audit would pass vacuously", docName)
		return
	}
	citedSet := make(map[string]bool, len(cited))
	for _, name := range cited {
		citedSet[name] = true
	}
	derivedSet := make(map[string]bool, len(derived))
	for _, name := range derived {
		derivedSet[name] = true
		if !citedSet[name] {
			t.Errorf("%s does not name the tool directive %s — the tool inventory must match %s's tool block (M-37)", docName, name, goModPath)
		}
	}
	for _, name := range cited {
		if !derivedSet[name] {
			t.Errorf("%s cites tool %q, which %s's tool block does not declare — an imagined dependency (T-09-07)", docName, name, goModPath)
		}
	}
}

// normalizeDoc joins whitespace-separated fields the way the sibling
// doc-anchor guards do, so source line wrapping cannot keep a stale
// sentence alive.
func normalizeDoc(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// agentsForbidden pins the v1.0 sentences the as-built rewrite replaced,
// matched whitespace-normalized. The zero-requires phrase is pinned to the
// FULL v1.0 sentence on purpose — the corrected dependency policy
// legitimately names the historical Phase-1 count, so the bare fragment
// must stay legal.
var agentsForbidden = []struct {
	phrase string
	why    string
}{
	{
		"Phase 1 carries **zero direct runtime requires**",
		"grpc and protobuf are direct requires today (M-36); only the corrected prose may name the historical Phase-1 count",
	},
	{
		"three packages",
		"internal/app is five packages (clock, persist, repl, storage, version) plus the harness packages (M-34)",
	},
	{
		"Adapter packages under `internal/` appear with the phases",
		"four adapters exist and are named: grpcapi, udprepl, persistfile, metrics (M-35)",
	},
}

// claudeStackForbidden pins the pre-implementation research claims the stack
// section must not carry (M-38…M-43), matched inside the GSD:stack slice.
var claudeStackForbidden = []struct {
	phrase string
	why    string
}{
	{"pgregory.net/rapid", "property tests are hand-rolled seeded math/rand/v2 generators in internal/app/internal/testutil (M-38)"},
	{"synctest", "the simulator is internal/app/internal/simnet, a discrete-event scheduler with no goroutines and no wall clock (M-39)"},
	{"json.MarshalIndent", "dumps are json.Encoder one-line JSONL (M-41)"},
	{"ReadBuildInfo", "version identity is ldflags-stamped (M-42)"},
	{"v0.23.0", "the x/sync version is stale (M-43)"},
}

// claudeWholeFileForbidden pins the two not-yet lines (M-44): they sit
// OUTSIDE the GSD:stack markers (the GSD:conventions and GSD:architecture
// sections), so the assertion runs against the whole file — a slice-scoped
// check could never match them.
var claudeWholeFileForbidden = []struct {
	phrase string
	why    string
}{
	{"Conventions not yet established", "the conventions are established and AGENTS.md is their home (M-44)"},
	{"Architecture not yet mapped", "the architecture pointers name the as-built sources (M-44)"},
}

// TestAgentsStackAudit audits AGENTS.md against the module: the cited Go
// directive equals go.mod's, every cited repo path exists, the
// tool-directive inventory equals go.mod's tool block in both directions,
// and the v1.0 sentences are gone.
func TestAgentsStackAudit(t *testing.T) {
	t.Parallel()
	inv := readModuleInventory(t)
	raw, err := os.ReadFile(agentsDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", agentsDocPath, err)
	}
	doc := string(raw)

	// M-32: the Go pin is the go.mod directive's own value, derived at test
	// time — never typed.
	needle := "go " + inv.goDirective
	if !strings.Contains(doc, needle) {
		t.Errorf("%s does not cite the module's go directive %q (M-32)", agentsDocPath, needle)
	}

	requirePathsExist(t, agentsDocPath, citedPaths(doc), minAgentsPaths)
	compareToolInventories(t, agentsDocPath, citedToolNames(strings.Split(doc, "\n")), inv.toolNames)

	normalized := normalizeDoc(doc)
	for _, f := range agentsForbidden {
		if strings.Contains(normalized, normalizeDoc(f.phrase)) {
			t.Errorf("%s still contains the stale claim %q — %s", agentsDocPath, f.phrase, f.why)
		}
	}
}

// TestClaudeStackAudit audits .claude/CLAUDE.md's GSD:stack section: the
// markers are the section's contract (missing markers are fatal), the slice
// cites the module's Go directive, existing package paths and the module's
// tool inventory in both directions, and the pre-implementation research
// claims are gone; the two not-yet lines sit outside the markers, so they
// are asserted against the whole file.
func TestClaudeStackAudit(t *testing.T) {
	t.Parallel()
	inv := readModuleInventory(t)
	raw, err := os.ReadFile(claudeDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", claudeDocPath, err)
	}
	doc := string(raw)

	start := strings.Index(doc, stackStartMarker)
	if start < 0 {
		t.Fatalf("%s is missing the %s marker — the stack section's contract is gone", claudeDocPath, stackStartMarker)
	}
	end := strings.Index(doc, stackEndMarker)
	if end < 0 || end < start {
		t.Fatalf("%s is missing a %s marker after %s — the stack section's contract is gone", claudeDocPath, stackEndMarker, stackStartMarker)
	}
	slice := doc[start:end]

	needle := "go " + inv.goDirective
	if !strings.Contains(slice, needle) {
		t.Errorf("the %s stack section does not cite the module's go directive %q (M-32)", claudeDocPath, needle)
	}

	requirePathsExist(t, claudeDocPath+" stack section", citedPaths(slice), minStackPaths)
	compareToolInventories(t, claudeDocPath+" stack section", citedToolNames(strings.Split(slice, "\n")), inv.toolNames)

	normalizedSlice := normalizeDoc(slice)
	for _, f := range claudeStackForbidden {
		if strings.Contains(normalizedSlice, f.phrase) {
			t.Errorf("the %s stack section still contains the stale claim %q — %s", claudeDocPath, f.phrase, f.why)
		}
	}

	normalizedDoc := normalizeDoc(doc)
	for _, f := range claudeWholeFileForbidden {
		if strings.Contains(normalizedDoc, f.phrase) {
			t.Errorf("%s still contains the stale claim %q — %s", claudeDocPath, f.phrase, f.why)
		}
	}
}

// TestAgentsStackAuditPackageInventory is the bidirectional package count
// (M-34/M-35 as-built): every package `go list ./...` returns must appear in
// AGENTS.md's Layout and adapter prose, so a package added or removed
// without a doc edit fails the audit.
func TestAgentsStackAuditPackageInventory(t *testing.T) {
	t.Parallel()
	pkgs := goListPackages(t)
	raw, err := os.ReadFile(agentsDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", agentsDocPath, err)
	}
	doc := string(raw)
	for _, pkg := range pkgs {
		if !strings.Contains(doc, pkg) {
			t.Errorf("%s does not name the module package %q — the Layout section must name every `go list ./...` package (M-34/M-35)", agentsDocPath, pkg)
		}
	}
}
