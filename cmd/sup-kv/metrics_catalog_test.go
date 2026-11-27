package main_test

// The DOCS-04/DOCS-07/DOCS-09 machine checks for the operator reference: the
// typed metrics reference, the log-message catalog and the Spec's failure
// table are checked against the binary's observable surface in BOTH
// directions. HONEST SCOPE: these checks prove docs/OPERATIONS.md cannot cite
// a metric family, a label value or a log message the binary does not render
// or emit, and that nothing rendered or emitted is undocumented — they do not
// prove the surface's design is good, and the truth of a runbook's advice
// stays the anchors' and the manual verification's business.
//
// The vocabulary source is metrics.Snapshot over zero-valued sources: the
// family builders render the complete label catalogs at zero state (the
// complete-catalog rule), so no live node is needed. Snapshot dereferences
// its accessor functions, so a bare metrics.Sources{} would panic — the
// closures below mirror internal/metrics' own package-internal zeroSources()
// test helper and return the zero stats.
//
// The log checks are source-greps over cmd/sup-kv/main.go: the sloglint
// static-msg discipline keeps every slog message a literal first argument, so
// a level-prefixed source grep ("slog.Warn(\"<literal>") is the exact
// call-site check, at the exact level; the redirected log-package records and
// the usage path carry their own grep forms. This is also why the failure
// table needs no separate adapter-renderer lookup: every literal a row can
// cite resolves through the same source (TestOperationsLogCatalog enforces
// the same set for the catalog).

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/metrics"
)

// The three section headings the checks parse, and the honest no-metric form
// the failure table's metric column may carry instead of a family name. The
// em dash in honestNoneMetric is part of the literal.
const (
	metricsReferenceHeading = "## Metrics reference"
	logCatalogHeading       = "## Log-message catalog"
	failureTableHeading     = "## Failure behaviour"
	operationsMainPath      = "main.go"
	honestNoneMetric        = "none — log only"
)

// vacuity floors: the exporter's complete supkv_ surface is 23 families today
// and the reference must carry them plus the five runtime families (the
// plan's "≥24 supkv" estimate was a research miscount — 23 is the real count,
// and direction A still fails on any missing family). The reference/catalog
// floors keep a parse regression from passing vacuously.
const (
	minSupkxFamilies      = 23
	minReferenceRows      = 15
	minCatalogLiterals    = 15
	minLabelValueChecks   = 20
	failureTableRowExpect = 18
)

var (
	// backtickTokenPattern extracts one markdown code span's contents.
	backtickTokenPattern = regexp.MustCompile("`([^`]+)`")
	// labelAssignmentPattern extracts one `class=xxx`-form label citation.
	labelAssignmentPattern = regexp.MustCompile(`(class|type|outcome)=([a-z0-9_]+)`)
)

// catalogLevelPrefixes maps a documented level to the call-site prefixes that
// prove a literal is emitted at that level: slog.Info/slog.Warn/slog.Error/
// slog.Debug take the message as their first argument (the static-msg
// discipline), and the log package's default logger is redirected through the
// one slog handler at INFO, so log.Print/log.Printf render as INFO records.
var catalogLevelPrefixes = map[string][]string{
	"INFO":  {"slog.Info(\"", "log.Print(\"", "log.Printf(\""},
	"WARN":  {"slog.Warn(\""},
	"ERROR": {"slog.Error(\""},
	"DEBUG": {"slog.Debug(\""},
}

// familyVocabulary is one exported family's observable surface: the
// exposition type and, per label name, the fixed value catalog.
type familyVocabulary struct {
	typ    string
	labels map[string][]string
}

// zeroMetricsSources mirrors internal/metrics' package-internal zeroSources()
// test helper: every accessor returns the zero value, and the family builders
// still render the complete label catalogs (the complete-catalog rule).
func zeroMetricsSources() metrics.Sources {
	return metrics.Sources{
		Discards:      func() repl.DiscardStats { return repl.DiscardStats{} },
		KnownPeers:    func() uint64 { return 0 },
		Unacked:       func() repl.UnackedStats { return repl.UnackedStats{} },
		Sent:          func() repl.SentByType { return repl.SentByType{} },
		Retransmitted: func() uint64 { return 0 },
		MergeStats:    func() repl.MergeStats { return repl.MergeStats{} },
		DumpStats:     func() persist.DumpStats { return persist.DumpStats{} },
		LoadStats:     func() persist.LoadStats { return persist.LoadStats{} },
		Paced:         func() uint64 { return 0 },
		SendQueue:     func() int { return 0 },
		EgressDropped: func() uint64 { return 0 },
		ReadErrors:    func() uint64 { return 0 },

		ListingDropped:         func() uint64 { return 0 },
		SyncStreamDropped:      func() uint64 { return 0 },
		ListedKeysEvicted:      func() uint64 { return 0 },
		NotificationsRegressed: func() uint64 { return 0 },
	}
}

// snapshotVocabulary builds the family→{type, labels} map from the zero-state
// snapshot — the exported truth both directions are checked against.
func snapshotVocabulary(t *testing.T) map[string]familyVocabulary {
	t.Helper()
	families := metrics.Snapshot(zeroMetricsSources())
	if len(families) == 0 {
		t.Fatal("metrics.Snapshot over zero sources renders no families — the vocabulary source is broken")
	}
	vocab := make(map[string]familyVocabulary, len(families))
	for _, f := range families {
		if _, dup := vocab[f.Name]; dup {
			t.Fatalf("the snapshot renders family %s twice", f.Name)
		}
		labels := make(map[string][]string)
		for _, s := range f.Samples {
			for _, l := range s.Labels {
				if !slices.Contains(labels[l.Name], l.Value) {
					labels[l.Name] = append(labels[l.Name], l.Value)
				}
			}
		}
		vocab[f.Name] = familyVocabulary{typ: string(f.Type), labels: labels}
	}
	return vocab
}

// readOperationsDoc reads the doc under guard (the doc itself is the input).
func readOperationsDoc(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(operationsDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", operationsDocPath, err)
	}
	return string(raw)
}

// tableRows returns the trimmed lines of one section that start with
// rowPrefix — one markdown table's rows of a known shape.
func tableRows(section, rowPrefix string) []string {
	var rows []string
	for line := range strings.SplitSeq(section, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, rowPrefix) {
			rows = append(rows, trimmed)
		}
	}
	return rows
}

// referenceRow is one parsed row of the metrics reference table.
type referenceRow struct {
	typ    string
	labels map[string][]string
}

// parseReferenceRows parses the reference table's rows: `| \`family\` | type |
// labels | meaning | alert |`, where the labels cell backticks the label name
// first and then its values (an em dash means unlabeled). One row per family.
func parseReferenceRows(t *testing.T, section string) map[string]referenceRow {
	t.Helper()
	rows := tableRows(section, "| `")
	out := make(map[string]referenceRow, len(rows))
	for _, row := range rows {
		cells := strings.Split(row, "|")
		if len(cells) < 6 {
			t.Fatalf("the metrics reference row %q does not have the Family/Type/Labels/Meaning/Alert columns", row)
		}
		nameTokens := backtickTokenPattern.FindAllStringSubmatch(cells[1], -1)
		if len(nameTokens) == 0 {
			t.Fatalf("the metrics reference row %q names no family in backticks", row)
		}
		name := nameTokens[0][1]
		if _, dup := out[name]; dup {
			t.Fatalf("the metrics reference has two rows for %s — one row per family", name)
		}
		typ := strings.TrimSpace(cells[2])
		if typ != "counter" && typ != "gauge" {
			t.Fatalf("the metrics reference row for %s states type %q, want counter or gauge", name, typ)
		}
		labels := make(map[string][]string)
		labelTokens := backtickTokenPattern.FindAllStringSubmatch(cells[3], -1)
		if len(labelTokens) > 0 {
			lname := labelTokens[0][1]
			for _, tok := range labelTokens[1:] {
				labels[lname] = append(labels[lname], tok[1])
			}
		}
		out[name] = referenceRow{typ: typ, labels: labels}
	}
	if len(out) == 0 {
		t.Fatal("the metrics reference section parses to zero rows — the cross-check would pass vacuously")
	}
	return out
}

// checkReferenceFamilies runs direction A (every exported family documented,
// with matching type and complete label values) and the doc-side label
// direction (every documented label name and value exported), returning the
// complaints and the number of label-value comparisons that had teeth.
func checkReferenceFamilies(vocab map[string]familyVocabulary, rows map[string]referenceRow) ([]string, int) {
	var complaints []string
	checked := 0
	for name, want := range vocab {
		row, ok := rows[name]
		if !ok {
			complaints = append(complaints, fmt.Sprintf("exported family %s has no row in the reference (direction A)", name))
			continue
		}
		if row.typ != want.typ {
			complaints = append(complaints, fmt.Sprintf("family %s is documented as type %q, the exporter renders %q", name, row.typ, want.typ))
		}
		for lname, values := range want.labels {
			documented, found := row.labels[lname]
			if !found {
				complaints = append(complaints, fmt.Sprintf("family %s does not document its %q label (direction A)", name, lname))
				continue
			}
			for _, v := range values {
				checked++
				if !slices.Contains(documented, v) {
					complaints = append(complaints, fmt.Sprintf("family %s renders label %s=%s but the reference omits it (direction A)", name, lname, v))
				}
			}
		}
		for lname, documented := range row.labels {
			values, exported := want.labels[lname]
			if !exported {
				complaints = append(complaints, fmt.Sprintf("family %s documents label %q which the exporter does not define (direction B)", name, lname))
				continue
			}
			for _, v := range documented {
				if !slices.Contains(values, v) {
					complaints = append(complaints, fmt.Sprintf("family %s documents label value %s=%s which the exporter does not render (direction B)", name, lname, v))
				}
			}
		}
	}
	return complaints, checked
}

// TestMetricsReferenceCatalog is the DOCS-07/DOCS-09 two-directional check:
// the `## Metrics reference` section and metrics.Snapshot over zero sources
// must agree in both directions — an exported family without a row fails, a
// documented family that is not exported fails, and a label value documented
// or omitted wrongly fails.
func TestMetricsReferenceCatalog(t *testing.T) {
	t.Parallel()
	vocab := snapshotVocabulary(t)
	supkvExported := 0
	for name := range vocab {
		if strings.HasPrefix(name, "supkv_") {
			supkvExported++
		}
	}
	if supkvExported < minSupkxFamilies {
		t.Fatalf("the zero-state snapshot renders %d supkv_ families, below the %d-family floor — the exported surface shrank", supkvExported, minSupkxFamilies)
	}
	section, hasSection := docSections(readOperationsDoc(t))[metricsReferenceHeading]
	if !hasSection {
		t.Fatalf("%s carries no %q section — the typed metrics reference the two-directional check needs (DOCS-07)", operationsDocPath, metricsReferenceHeading)
	}
	rows := parseReferenceRows(t, section)
	if len(rows) < minReferenceRows {
		t.Fatalf("the %s section carries %d rows, below the %d-row floor — the cross-check would pass vacuously", metricsReferenceHeading, len(rows), minReferenceRows)
	}
	complaints, labelChecks := checkReferenceFamilies(vocab, rows)
	for _, token := range supkvFamilyPattern.FindAllString(section, -1) {
		if _, exported := vocab[token]; !exported {
			complaints = append(complaints, fmt.Sprintf("the reference names %s, which the exporter does not render (direction B)", token))
		}
	}
	if labelChecks < minLabelValueChecks {
		complaints = append(complaints, fmt.Sprintf("only %d label-value comparisons ran — the label check would pass vacuously", labelChecks))
	}
	if len(complaints) > 0 {
		t.Fatalf("%s: the metrics reference and the exported surface disagree in %d place(s):\n  %s",
			operationsDocPath, len(complaints), strings.Join(complaints, "\n  "))
	}
}

// catalogRow is one parsed row of the log-message catalog.
type catalogRow struct {
	level   string
	literal string
}

// parseCatalogRows parses the catalog table's rows: `| Level | \`message\` |
// where |`. The header and separator rows carry no backticked message and are
// skipped; a malformed row is fatal.
func parseCatalogRows(t *testing.T, section string) []catalogRow {
	t.Helper()
	var rows []catalogRow
	for line := range strings.SplitSeq(section, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "| ") || strings.Contains(trimmed, "---") {
			continue
		}
		cells := strings.Split(trimmed, "|")
		if len(cells) < 4 {
			t.Fatalf("the log catalog row %q does not have the Level/Message/Where columns", trimmed)
		}
		tokens := backtickTokenPattern.FindAllStringSubmatch(cells[2], -1)
		if len(tokens) == 0 {
			continue // the header row carries no backticked message literal.
		}
		rows = append(rows, catalogRow{level: strings.TrimSpace(cells[1]), literal: tokens[0][1]})
	}
	return rows
}

// levelCallPresent proves the literal is a call site's message at the
// documented level: the static-msg discipline makes the first argument the
// message literal, so the level-prefixed source form is the exact check.
func levelCallPresent(src, literal, level string) bool {
	for _, prefix := range catalogLevelPrefixes[level] {
		if strings.Contains(src, prefix+literal) {
			return true
		}
	}
	return false
}

// TestOperationsLogCatalog is the catalog's literal check: every
// `## Log-message catalog` row's message literal must be a main.go call site
// at the row's documented level (INFO/WARN/ERROR/DEBUG; the usage path's
// plain text carries the `os.Stderr, "` form), so the catalog cannot promise
// a record the binary does not emit, at a level it does not use.
func TestOperationsLogCatalog(t *testing.T) {
	t.Parallel()
	mainSrc, err := os.ReadFile(operationsMainPath)
	if err != nil {
		t.Fatalf("read %s: %v", operationsMainPath, err)
	}
	src := string(mainSrc)
	section, hasSection := docSections(readOperationsDoc(t))[logCatalogHeading]
	if !hasSection {
		t.Fatalf("%s carries no %q section — the log-message catalog the level check needs (DOCS-07)", operationsDocPath, logCatalogHeading)
	}
	rows := parseCatalogRows(t, section)
	if len(rows) < minCatalogLiterals {
		t.Fatalf("the %s section carries %d literal rows, below the %d-literal floor — the check would pass vacuously", logCatalogHeading, len(rows), minCatalogLiterals)
	}
	var complaints []string
	for _, row := range rows {
		switch row.level {
		case "INFO", "WARN", "ERROR", "DEBUG":
			if !levelCallPresent(src, row.literal, row.level) {
				complaints = append(complaints, fmt.Sprintf("catalog literal %q is documented at %s, but main.go carries no %s call site for it", row.literal, row.level, row.level))
			}
		case "stderr":
			if !strings.Contains(src, `os.Stderr, "`+row.literal) {
				complaints = append(complaints, fmt.Sprintf("catalog literal %q is documented as the usage path's plain text, but the usage Fprintf does not carry it", row.literal))
			}
		default:
			t.Fatalf("the catalog row for %q carries unknown level %q — the level matrix is INFO/WARN/ERROR/DEBUG plus the stderr usage path", row.literal, row.level)
		}
	}
	if len(complaints) > 0 {
		t.Fatalf("%s: the log-message catalog disagrees with main.go in %d place(s):\n  %s",
			operationsDocPath, len(complaints), strings.Join(complaints, "\n  "))
	}
}

// labelRendered reports whether one of the metric cell's cited families
// renders the label=value citation (a family missing from the vocabulary
// already drew its own complaint).
func labelRendered(families []string, vocab map[string]familyVocabulary, name, value string) bool {
	for _, fam := range families {
		fv, ok := vocab[fam]
		if !ok {
			continue
		}
		if slices.Contains(fv.labels[name], value) {
			return true
		}
	}
	return false
}

// TestOperationsFailureTable is the failure table's two-column check: the
// `## Failure behaviour` section carries exactly the Spec's 18 rows keyed
// S-FAIL-01…S-FAIL-18 in registry order; per row, the metric column cites
// only exported families and label values (or carries the honest
// `none — log only` form), and the log column's backticked literals are
// main.go call-site messages — no row can send an operator hunting for a
// signal the binary does not produce.
func TestOperationsFailureTable(t *testing.T) {
	t.Parallel()
	vocab := snapshotVocabulary(t)
	mainSrc, err := os.ReadFile(operationsMainPath)
	if err != nil {
		t.Fatalf("read %s: %v", operationsMainPath, err)
	}
	src := string(mainSrc)
	section, hasSection := docSections(readOperationsDoc(t))[failureTableHeading]
	if !hasSection {
		t.Fatalf("%s carries no %q section — the Spec's 18-row failure-behavior table (DOCS-04)", operationsDocPath, failureTableHeading)
	}
	rows := tableRows(section, "| S-FAIL-")
	if len(rows) != failureTableRowExpect {
		t.Fatalf("the %s section carries %d S-FAIL rows, want exactly the Spec's %d", failureTableHeading, len(rows), failureTableRowExpect)
	}
	var complaints []string
	for i, row := range rows {
		id := fmt.Sprintf("S-FAIL-%02d", i+1)
		cells := strings.Split(row, "|")
		if len(cells) < 9 {
			complaints = append(complaints, fmt.Sprintf("row %d does not have the ID/Symptom/Behavior/Recovery/Bound/Metric/Log-line columns", i+1))
			continue
		}
		if got := strings.TrimSpace(cells[1]); got != id {
			complaints = append(complaints, fmt.Sprintf("row %d is keyed %q, want %s — the table must follow the registry order", i+1, got, id))
			continue
		}
		metricCell := strings.TrimSpace(cells[6])
		logCell := strings.TrimSpace(cells[7])
		families := supkvFamilyPattern.FindAllString(metricCell, -1)
		for _, fam := range families {
			if _, exported := vocab[fam]; !exported {
				complaints = append(complaints, fmt.Sprintf("row %s's metric column cites %s, which the exporter does not render", id, fam))
			}
		}
		if len(families) == 0 && !strings.Contains(metricCell, honestNoneMetric) {
			complaints = append(complaints, fmt.Sprintf("row %s's metric column cites no metric and is not the honest %q form", id, honestNoneMetric))
		}
		for _, m := range labelAssignmentPattern.FindAllStringSubmatch(metricCell, -1) {
			if !labelRendered(families, vocab, m[1], m[2]) {
				complaints = append(complaints, fmt.Sprintf("row %s's metric column cites label %s=%s, which no cited family renders", id, m[1], m[2]))
			}
		}
		for _, tok := range backtickTokenPattern.FindAllStringSubmatch(logCell, -1) {
			if !strings.Contains(src, tok[1]) {
				complaints = append(complaints, fmt.Sprintf("row %s's log column cites %q, which main.go does not emit", id, tok[1]))
			}
		}
	}
	if len(complaints) > 0 {
		t.Fatalf("%s: the failure table disagrees with the binary in %d place(s):\n  %s",
			operationsDocPath, len(complaints), strings.Join(complaints, "\n  "))
	}
}
