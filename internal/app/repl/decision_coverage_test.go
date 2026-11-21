package repl_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// decisionRow is one row of the TEST-13 index: the owner decision, its ADR
// file, and the anchor test that references the decision by name. The rows
// are documentation first and the enforcement table second — the meta-test
// fails naming any row whose citation is absent from the repo's test files.
type decisionRow struct {
	decision string // "14.N", the owner-decision number as REVIEW.md §14 writes it
	adrFile  string // the ADR filename under docs/adr/, one per decision
	anchor   string // the test expected to carry the citation
}

// decisionIndex enumerates all 11 section-14 owner decisions (ADR 0001-0011):
// the TEST-13 deliverable's index. Citations accepted by the meta-test are the
// ADR token ("ADR 000N") and the decision token ("decision 14.N"); the anchor
// column names where each one lives so a missing row points a reader at the
// right test.
//
// Two rows cite deliberately adjacent evidence rather than a behaviour test of
// their own: 14.7's last-read-value file is consumer territory (sup-kv ships
// none — the public proto surface is the boundary the anchor pins), and 14.9's
// real-socket 3+-node suite is PROTO-09's Phase 4 deliverable, so the current
// 2-5-node chaos runs carry the citation.
var decisionIndex = []decisionRow{
	{"14.1", "0001-snapshot-load-full-rollback.md", "TestEngine_snapshotCreateLoadRestore"},
	{"14.2", "0002-create-if-absent-blocked-by-tombstone.md", "TestServer_errorMapping"},
	{"14.3", "0003-dump-policy-unchanged.md", "TestEngine_confirmDebounceTiming"},
	{"14.4", "0004-no-conflict-journal.md", "TestStand_metricsByEffect"},
	{"14.5", "0005-interface-closed-until-full-key-list.md", "TestRed_R03_gateClosedUntilFullList"},
	{"14.6", "0006-startup-gate-empty-wait.md", "TestGate_opensOnTimeout"},
	{"14.7", "0007-public-proto-consumer-zone.md", "TestServer_setGetRoundTrip"},
	{"14.8", "0008-metrics-off-by-default.md", "TestMetrics_noDefaultListener"},
	{"14.9", "0009-multi-node-support.md", "TestChaos_convergence"},
	{"14.10", "0010-darwin-development-only.md", "TestPlatform_darwinReplicationUntested"},
	{"14.11", "0011-value-limit-60kib-kept.md", "TestEngine_configShrunkValueLimits"},
}

// coverageTestFile is this file's base name. The walk skips it: the index
// above spells every ADR token, so counting the meta-test itself would satisfy
// all 11 rows vacuously.
const coverageTestFile = "decision_coverage_test.go"

// moduleRoot walks up from the test binary's working directory — the package
// directory under `go test` — to the directory holding go.mod.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	for {
		_, statErr := os.Stat(filepath.Join(dir, "go.mod"))
		if statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod found above the working directory %q", dir)
		}
		dir = parent
	}
}

// repoTestSources returns the content of every *_test.go file of the module,
// keyed by path. The reviewer's repro module under docs/ is a separate Go
// module (evidence, not this module's suite) and testdata corpora are data, so
// both subtrees are skipped; everything else under the module root is read.
func repoTestSources(t *testing.T, root string) map[string]string {
	t.Helper()
	docsRoot := filepath.Join(root, "docs")
	sources := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path == docsRoot || entry.Name() == ".git" || entry.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), "_test.go") || entry.Name() == coverageTestFile {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		sources[path] = string(content)
		return nil
	})
	if err != nil {
		t.Fatalf("walking %q: %v", root, err)
	}
	return sources
}

// citationPattern builds the accepted citation form for one decision: the ADR
// token or the decision token. Both are word-bounded so decision 14.1 is not
// satisfied by "14.10" (and ADR 0001 not by "ADR 0010") — the coincidental
// prefix match a version string could otherwise supply.
func citationPattern(row decisionRow) *regexp.Regexp {
	adr := row.adrFile[:len("0001")]
	return regexp.MustCompile(`\bADR ` + adr + `\b|\bdecision ` + regexp.QuoteMeta(row.decision) + `\b`)
}

// TestDecisions_referencedByTests enforces TEST-13: every owner decision
// 14.1-14.11 (ADR 0001-0011) is referenced by name in at least one of the
// module's test files, and a decision without a citation fails this test
// naming the missing decision, its ADR file and the anchor test expected to
// carry it. The test reads source text only and executes none of it (T-2-17);
// the index above makes a removed citation visible in review.
func TestDecisions_referencedByTests(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t)
	adrDir := filepath.Join(root, "docs", "adr")
	sources := repoTestSources(t, root)

	var missing []string
	for _, row := range decisionIndex {
		_, statErr := os.Stat(filepath.Join(adrDir, row.adrFile))
		if statErr != nil {
			t.Errorf("decision %s: ADR file docs/adr/%s: %v",
				row.decision, row.adrFile, statErr)
		}
		pattern := citationPattern(row)
		found := false
		for _, content := range sources {
			if pattern.MatchString(content) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, "decision "+row.decision+
				" (docs/adr/"+row.adrFile+") — expected in "+row.anchor)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("TEST-13: %d of %d owner decisions lack a named test reference:\n  %s\n"+
			"add a `decision 14.N (ADR 000N)` citation to the anchor test of each row in this file's index",
			len(missing), len(decisionIndex), strings.Join(missing, "\n  "))
	}
}
