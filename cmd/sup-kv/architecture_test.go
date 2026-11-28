package main_test

// The DOCS-08/DOCS-10 architecture-doc guard (09-09): docs/ARCHITECTURE.md is
// the as-built description IDEA.md's banner promises — the package graph from
// `go list ./...` with one paragraph per package and the layering rules named
// with their depguard rule IDs — and docs/adr/README.md is the ADR index whose
// rows derive from the 0001…0012 files' frontmatter. HONEST SCOPE: these are
// coverage and correspondence guards only — they prove every `go list ./...`
// package is named in the doc, the six enforced boundary rule IDs appear
// there, the ADR index pairs every file with exactly one row (both
// directions, in numeric order, each row linking its file) and the
// ARCHITECTURE/AGENTS division of labor keeps its pointer while AGENTS.md
// keeps the conventions contract; they do not judge the prose's quality or
// verify each description's accuracy — that claim rests on every paragraph
// being written from the package's own doc comment at authoring time.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	architectureDocPath = "../../docs/ARCHITECTURE.md"
	adrIndexPath        = "../../docs/adr/README.md"
	adrFileGlob         = "../../docs/adr/00*.md"

	// Vacuity floors: a package scan or an ADR corpus that finds nothing
	// would pass the guard vacuously. 19 is the planning-time inventory
	// (09-RESEARCH §9) — the package list itself is derived from
	// `go list ./...` at test time, so a graph that shrinks below the floor
	// demands a conscious floor edit, and an ADR corpus below 12 is a
	// deleted decision.
	minArchPackages = 19
	minADRFiles     = 12
	minADRRows      = 12
)

// depguardRuleIDs are the six machine-enforced layering rules the doc's
// enforced-boundaries section must name — the IDs exactly as .golangci.yml
// declares them (AGENTS.md records their written justifications and is the
// contract they live in; the doc names and links, never restates).
var depguardRuleIDs = []string{
	"sup-kv-app-no-io",
	"sup-kv-port-stdlib-only",
	"sup-kv-public-api-no-internal",
	"sup-kv-adapters-not-in-core",
	"sup-kv-metrics-no-adapters",
	"sup-kv-grpcapi-no-repl-metrics",
}

// architectureForbidden pins the duplication the division of labor forbids:
// ARCHITECTURE.md names the rule IDs and links AGENTS.md — it never
// re-derives the vendored lint-config provenance (AGENTS.md deviation 2's
// contract) or restates the boundary justifications a second time, because a
// second copy of the contract can drift from it.
var architectureForbidden = []struct {
	phrase string
	why    string
}{
	{
		"golangci-lint-strict",
		"the vendored-config provenance is AGENTS.md deviation 2's contract — ARCHITECTURE.md names the rule IDs and links, never copies the provenance",
	},
	{
		"powerman/",
		"the vendored-config source is recorded once, in AGENTS.md — a restatement here is a drift surface",
	},
}

// TestArchitectureDocPackages is the package-graph guard (T-09-26): it runs
// `go list ./...` at the module root at test time (the stack audit's exec —
// the expectation moves with the module instead of freezing a snapshot) and
// requires docs/ARCHITECTURE.md to name every listed package, so a package
// added or removed without a doc edit fails `make test`. The doc's enforced-
// boundaries section must name all six depguard rule IDs.
func TestArchitectureDocPackages(t *testing.T) {
	t.Parallel()
	pkgs := goListPackages(t)
	if len(pkgs) < minArchPackages {
		t.Fatalf("`go list ./...` returned %d packages — below the %d-package floor, the coverage check would pass vacuously", len(pkgs), minArchPackages)
	}
	raw, err := os.ReadFile(architectureDocPath)
	if err != nil {
		t.Fatalf("read %s: %v — the as-built architecture description the guard binds does not exist", architectureDocPath, err)
	}
	doc := string(raw)
	for _, pkg := range pkgs {
		if !strings.Contains(doc, pkg) {
			t.Errorf("%s does not name the module package %q — one paragraph per `go list ./...` package is the as-built claim (T-09-26)", architectureDocPath, pkg)
		}
	}
	for _, rule := range depguardRuleIDs {
		if !strings.Contains(doc, rule) {
			t.Errorf("%s does not name the depguard rule %q — the layering section must name every enforced boundary by its rule ID", architectureDocPath, rule)
		}
	}
}

// adrFrontmatterPattern extracts one ADR file's own number from its
// frontmatter — the index's rows derive from the files, never a hand-typed
// list.
var adrFrontmatterPattern = regexp.MustCompile(`(?m)^adr:\s*(\d{4})\s*$`)

// adrRowNumberPattern extracts a table row's ADR number: the row's first cell
// must start with the number (bare or markdown-linked) — dates or years
// elsewhere in the cell never count.
var adrRowNumberPattern = regexp.MustCompile(`^\[?(\d{4})\]?`)

// adrFiles enumerates the ADR corpus and derives each file's number from its
// own `adr:` frontmatter line, returning number -> basename.
func adrFiles(t *testing.T) map[string]string {
	t.Helper()
	paths, err := filepath.Glob(adrFileGlob)
	if err != nil {
		t.Fatalf("glob %s: %v", adrFileGlob, err)
	}
	if len(paths) < minADRFiles {
		t.Fatalf("the ADR corpus holds %d files under %s — below the %d-file floor, the correspondence check would pass vacuously", len(paths), adrFileGlob, minADRFiles)
	}
	numbers := make(map[string]string, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		match := adrFrontmatterPattern.FindStringSubmatch(string(raw))
		if match == nil {
			t.Fatalf("%s declares no `adr: NNNN` frontmatter — the index derives its rows from the files' frontmatter, so an unparseable file is a corpus error", path)
		}
		numbers[match[1]] = filepath.Base(path)
	}
	return numbers
}

// adrIndexRows extracts the index's table rows — lines starting `|` whose
// first cell starts with a 4-digit ADR number (the header and separator rows
// carry none and never count) — returning number -> row line, and asserting
// the rows are in strictly ascending numeric order.
func adrIndexRows(t *testing.T, index string) map[string]string {
	t.Helper()
	rows := make(map[string]string)
	var order []string
	for line := range strings.SplitSeq(index, "\n") {
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 3 {
			continue
		}
		first := strings.TrimSpace(cells[1])
		match := adrRowNumberPattern.FindStringSubmatch(first)
		if match == nil {
			continue
		}
		num := match[1]
		if _, dup := rows[num]; dup {
			t.Errorf("%s carries more than one row for ADR %s — the index is one row per ADR", adrIndexPath, num)
			continue
		}
		rows[num] = line
		order = append(order, num)
	}
	if len(rows) < minADRRows {
		t.Fatalf("%s carries %d ADR rows — below the %d-row floor, the correspondence check would pass vacuously", adrIndexPath, len(rows), minADRRows)
	}
	for i := 1; i < len(order); i++ {
		if order[i] <= order[i-1] {
			t.Errorf("%s carries ADR rows out of numeric order (%s after %s) — the index is ordered 0001…NNNN", adrIndexPath, order[i], order[i-1])
		}
	}
	return rows
}

// TestADRIndex is the ADR-index correspondence check (T-09-27): the index
// must have one row per ADR file, derived from the file's own `adr:`
// frontmatter value, and no row without its file — the correspondence is
// bidirectional, each row must link the file it names, and the rows stay in
// numeric order. An unindexed ADR hides a recorded decision; an indexed one
// without a file points at nothing.
func TestADRIndex(t *testing.T) {
	t.Parallel()
	files := adrFiles(t)
	raw, err := os.ReadFile(adrIndexPath)
	if err != nil {
		t.Fatalf("read %s: %v — the ADR index the correspondence check binds does not exist", adrIndexPath, err)
	}
	rows := adrIndexRows(t, string(raw))
	for num, basename := range files {
		row, ok := rows[num]
		switch {
		case !ok:
			t.Errorf("%s has no row for ADR %s (%s) — an unindexed ADR hides a recorded decision (T-09-27)", adrIndexPath, num, basename)
		case !strings.Contains(row, basename):
			t.Errorf("%s's row for ADR %s does not link its file %s — the row must point at the decision it indexes", adrIndexPath, num, basename)
		}
	}
	for num := range rows {
		if _, ok := files[num]; !ok {
			t.Errorf("%s carries a row for ADR %s, which has no file under docs/adr/ — an indexed decision without a file points at nothing (T-09-27)", adrIndexPath, num)
		}
	}
}

// TestArchitectureDocDivisionOfLabor is the division-of-labor anchor
// (DOCS-08's adjacency probe): ARCHITECTURE.md carries the as-built deep
// description and points at AGENTS.md for the conventions contract; AGENTS.md
// keeps its role — the conventions section and the recorded depguard
// deviations stay there, and ARCHITECTURE.md never restates the
// vendored-config provenance or the boundary justifications a second time
// (a second copy of the contract can drift from it).
func TestArchitectureDocDivisionOfLabor(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(architectureDocPath)
	if err != nil {
		t.Fatalf("read %s: %v — the as-built architecture description the division-of-labor anchor binds does not exist", architectureDocPath, err)
	}
	doc := string(raw)

	pointer := false
	for line := range strings.SplitSeq(doc, "\n") {
		if strings.Contains(line, "AGENTS.md") && strings.Contains(strings.ToLower(line), "convention") {
			pointer = true
			break
		}
	}
	if !pointer {
		t.Errorf(
			"%s carries no line pointing at AGENTS.md for the conventions — the as-built description and the conventions contract name each other instead of duplicating",
			architectureDocPath,
		)
	}

	agents, err := os.ReadFile(agentsDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", agentsDocPath, err)
	}
	if !strings.Contains(string(agents), "## Conventions") {
		t.Errorf("%s lost its Conventions section — AGENTS.md keeps the conventions contract; the anchor assumes its role intact", agentsDocPath)
	}
	if !strings.Contains(string(agents), "Depguard architecture boundary") {
		t.Errorf("%s lost deviation 1's recorded boundary — the justifications stay in AGENTS.md; ARCHITECTURE.md names rule IDs and links, never restates", agentsDocPath)
	}

	normalized := normalizeDoc(doc)
	for _, f := range architectureForbidden {
		if strings.Contains(normalized, normalizeDoc(f.phrase)) {
			t.Errorf("%s still restates the boundary contract detail %q — %s", architectureDocPath, f.phrase, f.why)
		}
	}
}
