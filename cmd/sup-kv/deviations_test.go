package main_test

// The DOCS-01/DOCS-08 registry gate of Phase 9 (09-04 D-01): the deviation
// registry's closure is machine-checked, per the preamble's own "Phase 9
// greps it for unclosed rows" prescription. TestSpecDeviationsNoUnclosedRows
// fails on any table row whose status cell carries the unclosed
// fixed-in-v1.1(...) marker and enforces the status-vocabulary enum, so a
// typo'd status cannot slip through either. TestSpecDeviationsLinksResolve
// fails on any cited repo path, path#TestName pair or path:lines (`Symbol`)
// pair that does not resolve against the working tree — the citation
// resweep's permanent guard (the 06-10 neighbouring-declaration lesson).
//
// HONEST SCOPE: this is a closure-form and existence guard only. It proves
// the registry's rows are closed and that every citation names a path, a
// test or a symbol that exists; it does NOT pin line numbers — a citation
// can name the right symbol at a drifted line range and still pass here,
// because line-number currency is the audit pass's (human-verified) job, not
// the gate's. The deferred-v2(SNAP-01) and accepted-deviation(...) rows are
// legitimate permanent statuses: the exit criterion is zero unclosed rows,
// not zero non-implemented rows.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	// repoRoot resolves the registry's repo-relative citations against the
	// working tree (the package lives at cmd/sup-kv, two levels below root).
	// The registry path itself is performance_test.go's existing
	// specDeviationsDocPath const — the same doc, one declaration.
	repoRoot = "../.."

	// registryRowFloor is the row-parser vacuity floor: the registry carries
	// 167 rows, so a parse below this floor means the row matcher is broken
	// (for example it counted the preamble or dropped rows), not that the
	// table shrank.
	registryRowFloor = 150

	// linkCheckFloor is the link-resolver vacuity floor: the current table
	// yields well over four hundred path/symbol/test checks, so a walk below
	// this floor would pass vacuously.
	linkCheckFloor = 250

	// unclosedMarker is the one unclosed status marker the milestone exit
	// criterion forbids in table cells. The preamble's vocabulary line spells
	// the same token, which is exactly why the row matcher matches the
	// table-row form ("| S-"), never the preamble.
	unclosedMarker = "fixed-in-v1.1("
)

// statusEnum is the fixed status vocabulary a table cell may carry. The
// unclosed fixed-in-v1.1( marker is deliberately absent: a row carrying it is
// the gate's named failure, and any other unknown status is a typo the gate
// must reject — the gate cannot pass on an unparseable table.
var statusEnum = []string{
	"implemented",
	"accepted-deviation(",
	"consumer-zone",
	"superseded-by-amendment",
	"deferred-v2(",
}

// registryRow is one parsed table row of the deviation registry.
type registryRow struct {
	id     string
	status string
	links  string
}

var (
	backtickToken = regexp.MustCompile("`([^`]+)`")
	// lineSpecTail splits a path:lines token such as
	// internal/app/repl/core.go:87,95,792-815 into path and line spec; the
	// line spec itself is never verified (honest scope: no line pinning).
	lineSpecTail = regexp.MustCompile(`^(.+):(\d[0-9,-]*)$`)
	// lineSpecOnly matches a continuation line token such as `:447`, which
	// inherits the last path seen in the same Links cell.
	lineSpecOnly = regexp.MustCompile(`^:(\d[0-9,-]*)$`)
	// testFileRef matches the file#TestName citation form.
	testFileRef = regexp.MustCompile(`^(.+_test\.go)#([A-Za-z_][A-Za-z0-9_]*)$`)
)

// parseRegistryRows extracts the table rows: a row is a line whose
// whitespace-normalized form starts "| S-". The preamble's vocabulary line
// carries the same status tokens but never this row-prefix form, so it cannot
// be mistaken for a row, and a malformed row (fewer than four columns) yields
// no status and is caught by the enum check, not silently dropped.
func parseRegistryRows(raw string) []registryRow {
	var rows []registryRow
	for line := range strings.SplitSeq(raw, "\n") {
		normalized := strings.Join(strings.Fields(line), " ")
		if !strings.HasPrefix(normalized, "| S-") {
			continue
		}
		cells := strings.Split(normalized, "|")
		row := registryRow{id: strings.TrimSpace(normalized)}
		if len(cells) >= 5 {
			row.id = strings.TrimSpace(cells[1])
			row.status = strings.TrimSpace(cells[3])
			row.links = strings.TrimSpace(cells[4])
		}
		rows = append(rows, row)
	}
	return rows
}

// hasStatusVocabulary reports whether the status cell is inside the enum.
func hasStatusVocabulary(status string) bool {
	for _, allowed := range statusEnum {
		if strings.HasPrefix(status, allowed) {
			return true
		}
	}
	return false
}

// TestSpecDeviationsNoUnclosedRows is the milestone exit criterion: zero
// table-cell fixed-in-v1.1( rows, with every parsed status inside the
// vocabulary so a typo'd status is fatal. A pass with fewer rows than the
// floor would be vacuous, not evidence of closure.
func TestSpecDeviationsNoUnclosedRows(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(specDeviationsDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", specDeviationsDocPath, err)
	}
	rows := parseRegistryRows(string(raw))
	if len(rows) < registryRowFloor {
		t.Fatalf(
			"%s: parsed only %d table rows, below the %d-row floor — the row matcher is broken and a pass here would be vacuous, not closure evidence",
			specDeviationsDocPath, len(rows), registryRowFloor,
		)
	}
	var problems []string
	for _, row := range rows {
		switch {
		case strings.Contains(row.status, unclosedMarker):
			problems = append(problems, fmt.Sprintf(
				"row %s still carries the unclosed status %q — a live fixed-in-v1.1(...) table cell forges the milestone exit criterion (ROADMAP success criterion 3: no unclosed rows); the flip needs the fix evidence in its Links cell",
				row.id,
				row.status,
			))
		case !hasStatusVocabulary(row.status):
			problems = append(problems, fmt.Sprintf(
				"row %s status %q is outside the status vocabulary — a typo'd status would let an unclosed or malformed row slip past this gate",
				row.id, row.status))
		}
	}
	if len(problems) > 0 {
		t.Fatalf(
			"%s carries %d status problems:\n  %s",
			specDeviationsDocPath, len(problems), strings.Join(problems, "\n  "),
		)
	}
}

// registryAuditor resolves registry citations against the working tree,
// reading each cited file once.
type registryAuditor struct {
	contents map[string]string
	checks   int
}

func newRegistryAuditor() *registryAuditor {
	return &registryAuditor{contents: make(map[string]string)}
}

// content returns the cited file's text and whether the file was readable.
func (a *registryAuditor) content(rel string) (string, bool) {
	if text, ok := a.contents[rel]; ok {
		return text, ok
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
	text, ok := "", false
	if err == nil {
		text, ok = string(data), true
	}
	a.contents[rel] = text
	return text, ok
}

// exists reports whether the cited path (file or directory) is on disk.
func (a *registryAuditor) exists(rel string) bool {
	a.checks++
	_, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(rel)))
	return err == nil
}

// globHits reports whether the cited glob pattern matches at least one path.
func (a *registryAuditor) globHits(pattern string) bool {
	a.checks++
	matches, err := filepath.Glob(filepath.Join(repoRoot, filepath.FromSlash(pattern)))
	return err == nil && len(matches) > 0
}

// isRepoPath reports whether a backticked token names a repo path: it must
// carry a slash plus a known suffix or a known top-level directory. Bare test
// names, REVIEW §, G-xx and L<n> tokens never match and are skipped.
func isRepoPath(path string) bool {
	if !strings.Contains(path, "/") {
		return false
	}
	for _, suffix := range []string{".go", ".md", ".proto", ".tsv", ".yml"} {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	for _, prefix := range []string{"internal/", "cmd/", "api/", "docs/", "scripts/", ".github/", "deploy/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// trailingSymbols extracts the backticked symbols of the parenthetical that
// immediately follows a citation token, if any — the `path:lines` (`Symbol`)
// association.
func trailingSymbols(links string, from int) []string {
	rest := links[from:]
	trimmed := strings.TrimLeft(rest, " ")
	if !strings.HasPrefix(trimmed, "(") {
		return nil
	}
	head, _, found := strings.Cut(trimmed, ")")
	if !found {
		return nil
	}
	var symbols []string
	for _, match := range backtickToken.FindAllStringSubmatch(head, -1) {
		symbols = append(symbols, match[1])
	}
	return symbols
}

// checkSymbols verifies every associated symbol appears in the cited file.
func (a *registryAuditor) checkSymbols(rowID, path string, symbols []string, report func(string)) {
	if len(symbols) == 0 {
		return
	}
	text, ok := a.content(path)
	if !ok {
		return // the missing path is already reported; no symbol verdict here
	}
	for _, symbol := range symbols {
		a.checks++
		if !strings.Contains(text, symbol) {
			report(fmt.Sprintf(
				"row %s cites symbol %q in %q, but the file does not contain it — a pass would leave a citation naming a declaration that is not there (the 06-10 stale-citation lesson)",
				rowID,
				symbol,
				path,
			))
		}
	}
}

// TestSpecDeviationsLinksResolve is the citation resweep's permanent guard:
// every cited repo path, path#TestName pair and path:lines (`Symbol`) pair in
// every Links cell must resolve against the working tree. Non-path tokens
// (REVIEW §, G-xx, L<n>, bare test names) are skipped, and a walk below the
// check floor would pass vacuously.
func TestSpecDeviationsLinksResolve(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(specDeviationsDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", specDeviationsDocPath, err)
	}
	rows := parseRegistryRows(string(raw))
	if len(rows) < registryRowFloor {
		t.Fatalf(
			"%s: parsed only %d table rows, below the %d-row floor — the row matcher is broken and the link walk would be vacuous",
			specDeviationsDocPath, len(rows), registryRowFloor,
		)
	}
	auditor := newRegistryAuditor()
	var problems []string
	report := func(problem string) {
		problems = append(problems, problem)
	}
	for _, row := range rows {
		auditor.checkRow(row, report)
	}
	if auditor.checks < linkCheckFloor {
		problems = append(problems, fmt.Sprintf(
			"the link walk performed only %d path/symbol checks, below the %d-check floor — the resolver would pass vacuously",
			auditor.checks, linkCheckFloor))
	}
	if len(problems) > 0 {
		t.Fatalf(
			"%s carries %d unresolvable citations (%d checks performed):\n  %s",
			specDeviationsDocPath, len(problems), auditor.checks, strings.Join(problems, "\n  "),
		)
	}
	t.Logf("link walk resolved %d path/symbol/test citations across %d rows", auditor.checks, len(rows))
}

// checkRow walks one row's Links cell and reports every citation that does
// not resolve. Three token classes are checked: repo paths must exist
// (glob tokens must match at least one path), `file_test.go#TestName` pairs
// must name an existing func in that file, and `path:lines` (`Symbol`) pairs
// must name symbols the cited file carries. A `:lines` or `#TestName`
// continuation token inherits the path of the last citation in the cell.
func (a *registryAuditor) checkRow(row registryRow, report func(string)) {
	matches := backtickToken.FindAllStringSubmatchIndex(row.links, -1)
	lastPath := ""
	lastTestFile := ""
	for _, match := range matches {
		token := row.links[match[2]:match[3]]
		switch {
		case strings.HasPrefix(token, "#"):
			if lastTestFile == "" {
				report(fmt.Sprintf(
					"row %s cites the test continuation %q with no preceding file#Test citation — a pass would leave a citation that names no test file",
					row.id, token))
				continue
			}
			a.checkTestRef(row.id, lastTestFile, token[1:], report)
		case lineSpecOnly.MatchString(token):
			if lastPath == "" {
				report(fmt.Sprintf(
					"row %s cites the line continuation %q with no preceding path citation — a pass would leave a citation that names no file",
					row.id, token))
				continue
			}
			a.checkSymbols(row.id, lastPath, trailingSymbols(row.links, match[1]), report)
		case testFileRef.MatchString(token):
			parts := testFileRef.FindStringSubmatch(token)
			lastTestFile = parts[1]
			a.checkTestRef(row.id, parts[1], parts[2], report)
		default:
			path := token
			hasLines := false
			if parts := lineSpecTail.FindStringSubmatch(token); len(parts) > 1 && isRepoPath(parts[1]) {
				path, hasLines = parts[1], true
			}
			if !isRepoPath(path) {
				continue // bare test names and review shorthand are not citations
			}
			lastPath = path
			switch {
			case strings.ContainsAny(path, "*?["):
				if !a.globHits(path) {
					report(fmt.Sprintf(
						"row %s cites %q, which matches no path in the tree — a pass would leave a dead citation pointing readers nowhere",
						row.id, path))
				}
			case !a.exists(path):
				report(fmt.Sprintf(
					"row %s cites %q, which does not exist in the tree — a pass would leave a dead citation pointing readers nowhere",
					row.id, path))
			}
			if hasLines {
				a.checkSymbols(row.id, path, trailingSymbols(row.links, match[1]), report)
			}
		}
	}
}

// checkTestRef verifies the cited test exists in the cited file.
func (a *registryAuditor) checkTestRef(rowID, file, test string, report func(string)) {
	a.checks++
	text, ok := a.content(file)
	if !ok {
		report(fmt.Sprintf(
			"row %s cites %s#%s, but the file does not exist — a pass would leave a citation naming a test that cannot be read",
			rowID, file, test))
		return
	}
	if !strings.Contains(text, "func "+test+"(") {
		report(fmt.Sprintf(
			"row %s cites %s#%s, but the file declares no func %s — a pass would leave a citation naming a test that does not exist",
			rowID, file, test, test))
	}
}
