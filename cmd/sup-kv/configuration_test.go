package main_test

// The DOCS-02 registered-flags iterator (09-01, the phase's tracer slice):
// the binary's own --help is the machine source for docs/CONFIGURATION.md.
// TestConfigurationDoc parses a clean-environment --help capture of the
// built smoke binary and checks the doc against it in BOTH directions:
// every registered flag must have a flag-table row whose cells carry the
// printed default and the printed env twin (a flag added, renamed or
// re-defaulted in internal/config without a doc edit fails), and every
// --flag token the doc names must actually be registered — a
// documented-but-nonexistent flag is fatal, with the offending token named.
// The rows must preserve the --help registration order row-by-row, and the
// five env-only knobs must have entries in the doc's Environment-only
// section.
//
// HONEST SCOPE: this is a presence/drift guard over the registered surface.
// It proves the doc matches the binary's --help — never that the help is
// right, and never that a Meaning cell's prose is true. The vacuity guards
// (a minimum parsed-flag count, a minimum relation-row count and a minimum
// validate.go catalog count) exist so a broken parse can never turn the
// comparison into a silent pass.
//
// TestConfigurationDocRelations pins the doc's validation-relations section
// against internal/config/validate.go's rule catalog — the catalog file is
// the anchor source, so a validation rule added without a doc row fails the
// anchors — plus the must-match-on-all-nodes list and the operator-facing
// relation literals.

import (
	"bytes"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

const (
	// configurationDocPath is the doc under guard, relative to this
	// package's directory — the doc itself is the input, never a copied
	// fixture.
	configurationDocPath = "../../docs/CONFIGURATION.md"

	// validateSourcePath is the one validation point whose rule catalog the
	// relations section must mirror (internal/config/validate.go, read as
	// source so an added rule fails the anchors without a test edit).
	validateSourcePath = "../../internal/config/validate.go"

	// minRegisteredFlags is the vacuity guard over the --help parse: the
	// iterator would pass vacuously below this many registered flags, so a
	// parse that sees fewer is a broken parse, not a small surface.
	minRegisteredFlags = 22

	// minCatalogRules is the vacuity guard over the validate.go rule parse:
	// the catalog carries twelve rules today, so a parse that sees fewer is
	// a broken parse and the relations check must not run.
	minCatalogRules = 12

	// minRelationRows is the vacuity guard over the relations section: the
	// catalog alone has twelve rows, so fewer means the section lost its
	// table.
	minRelationRows = 8

	// helpEnvOnlyBlockTitle is the --help footer's own title for the five
	// flagless knobs — the anchor the doc's env-only section mirrors.
	helpEnvOnlyBlockTitle = "Environment-only (no flag):"

	// docEnvOnlyHeading is the doc section title the iterator's env-only
	// lookup requires (the help block's title, as a heading).
	docEnvOnlyHeading = "## Environment-only (no flag)"
)

// envOnlyKnobs are the five SUPKV_ variables with no flag twin, in the help
// footer's own order: the structured-logging pair and the metrics
// certificate triple. Both the --help capture and the doc's env-only
// section must carry all five.
var envOnlyKnobs = []string{
	"SUPKV_LOG_LEVEL",
	"SUPKV_LOG_FORMAT",
	"SUPKV_METRICS_CERT",
	"SUPKV_METRICS_KEY",
	"SUPKV_METRICS_CLIENT_CA",
}

// configurationDocAnchors pins the doc-level statements that are not per-flag
// rows: the opening scope statement naming this iterator, the precedence
// rule, the metrics listener's no-default behavior and the --version row's
// special statement. Each literal is chosen from the authored doc's own
// wording — dropping the statement breaks the guard loudly.
var configurationDocAnchors = []struct {
	literal string
	pins    string
}{
	{"cmd/sup-kv/configuration_test.go", "the opening scope statement names this iterator as the drift guard"},
	{"flag > env > default", "the precedence rule, stated with its OPERATIONS pointer"},
	{"stays off unless", "the --metrics-addr row states the metrics listener stays off unless the flag (or its env twin) is set"},
	{"not a runtime knob", "the --version row states it prints the build identity, not a runtime knob"},
}

// configurationRelationsAnchors pin the operator-facing relation literals the
// validation-relations section must state, with the exit-2 behavior. The
// validate.go catalog names come from the live parse; these literals come
// from the relations' as-built wording.
var configurationRelationsAnchors = []struct {
	literal string
	pins    string
}{
	{"exit code 2", "every catalog violation is a configuration error the binary reports by exiting 2, never a silent fallback"},
	{"must be positive", "the positivity sentinel's own wording (the process tests match it)"},
	{"peer_timeout > hello_interval", "the peer-vs-hello adjacency relation in operator terms"},
	{"ack_timeout ≤ ack_timeout_max", "the retransmit ladder's base at or under its ceiling"},
	{"dump_debounce ≤ dump_max_wait", "the dump cycle's quiet-window relation"},
	{"drift_warn < max_clock_lead", "the drift WARN window's reachability relation"},
	{"65 507", "the datagram ceiling the value and chunk relations cite"},
	{"1 024", "the key limit in the value/datagram fit relation"},
	{"zero or negative", "the max_clock_lead normalization the replication core applies"},
	{"positional arguments", "Parse's reject-by-name behavior for non-flag arguments"},
}

// configurationMustMatchAnchors pin the must-match list: the knobs whose
// values must agree on every node, the node-id uniqueness statement and the
// OPERATIONS cross-link (the as-built reasoning lives there — the reference
// links it, never restates it divergently).
var configurationMustMatchAnchors = []struct {
	literal string
	pins    string
}{
	{"--value-max-bytes", "the value-size envelope must match on all nodes"},
	{"--sync-chunk-bytes", "the DataSync chunk budget must match on all nodes"},
	{"--max-clock-lead", "the inbound version bound must match on all nodes"},
	{"--node-id", "the node identity knob whose uniqueness the list states"},
	{"unique", "node-id uniqueness across the cluster"},
	{"OPERATIONS.md", "the cross-link to the as-built reasoning in docs/OPERATIONS.md"},
}

// helpFlag is one flag parsed from the binary's --help capture: the
// registered name, the printed default token ("" when the help prints none —
// required knobs, the no-listener metrics address and the bool --version),
// and the printed SUPKV_ env twin ("" for --version).
type helpFlag struct {
	name    string
	defval  string
	envTwin string
}

// helpFlagLine matches one PrintDefaults entry line: two spaces, a dash, the
// flag name, and the optional type token (`duration`, `string`, `uint`, … —
// absent for bools).
var helpFlagLine = regexp.MustCompile(`^ {2}-([a-zA-Z0-9][a-zA-Z0-9_.-]*)(?:\s+(\S+))?\s*$`)

// helpDefaultToken matches the printed default at a usage line's end; string
// defaults arrive quoted, so the token is unquoted on capture.
var helpDefaultToken = regexp.MustCompile(`\(default (.+)\)\s*$`)

// helpEnvToken matches the printed env twin inside a usage line.
var helpEnvToken = regexp.MustCompile(`\(env (SUPKV_[A-Z0-9_]+)\)`)

// docFlagTokenPattern matches every --flag token the doc names; each must be
// a registered flag (the --help capture's own --help mention is allowed).
var docFlagTokenPattern = regexp.MustCompile(`--[a-z][a-z0-9-]*`)

// docFlagRowPattern matches a flag-table row line and captures the flag name.
var docFlagRowPattern = regexp.MustCompile(`^\| --([a-z][a-z0-9-]*)`)

// docRuleRowPattern matches a validation-relations table row whose first cell
// is a backticked catalog rule name.
var docRuleRowPattern = regexp.MustCompile(`^\| \x60[a-z]`)

// catalogRulePattern matches one validate.go rules() entry (the slice
// literal's tab-indented lines):
// {"required-knobs", checkRequiredKnobs}, — the catalog file is the anchor
// source, so the doc check follows the file, not a copied list.
var catalogRulePattern = regexp.MustCompile(`(?m)^\t+\{"([a-z][a-z0-9-]*)", check[A-Za-z]+\},`)

// runHelpCleanEnv runs the built binary's --help with a minimal environment
// (PATH/HOME-class minimums, never a SUPKV_ twin) and returns the merged
// stdout+stderr capture. config.Usage resolves the printed defaults from the
// process environment, so an inherited SUPKV_MAX_CLOCK_LEAD would skew them;
// the capture is merged like runToCompletion because --help's stream is an
// implementation detail of the flag package.
func runHelpCleanEnv(t *testing.T, bin string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), bin, "--help")
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME")}
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	if err != nil {
		t.Fatalf("--help exited with %v, want the flag package's clean help exit 0", err)
	}
	return out.String()
}

// parseHelpFlags walks the --help capture in registration order: an entry
// line names the flag, the indented usage line(s) that follow carry the env
// twin and the printed default.
func parseHelpFlags(help string) []helpFlag {
	lines := strings.Split(help, "\n")
	flags := make([]helpFlag, 0, minRegisteredFlags)
	for i, line := range lines {
		if !strings.HasPrefix(line, "  -") {
			continue
		}
		m := helpFlagLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		f := helpFlag{name: m[1]}
		for j := i + 1; j < len(lines); j++ {
			usage := lines[j]
			if !strings.HasPrefix(usage, "    ") {
				break
			}
			text := strings.TrimSpace(usage)
			if d := helpDefaultToken.FindStringSubmatch(text); d != nil {
				f.defval = strings.Trim(d[1], `"`)
			}
			if e := helpEnvToken.FindStringSubmatch(text); e != nil {
				f.envTwin = e[1]
			}
		}
		flags = append(flags, f)
	}
	return flags
}

// docFlagRow is one flag-table row of the doc: the flag name the row opens
// with and the full row line (its cells are read with docRowCell).
type docFlagRow struct {
	name string
	row  string
}

// parseDocFlagRows returns the doc's flag-table rows (lines starting
// `| --name`) in document order.
func parseDocFlagRows(doc string) []docFlagRow {
	rows := make([]docFlagRow, 0, minRegisteredFlags)
	for line := range strings.SplitSeq(doc, "\n") {
		m := docFlagRowPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		rows = append(rows, docFlagRow{name: m[1], row: line})
	}
	return rows
}

// docRowCell returns the row's nth table cell (the text between the nth and
// n+1th pipe), trimmed: cell 1 is the flag, cell 2 the env twin, cell 3 the
// default — the five columns are Flag | Env twin | Default | Units | Meaning.
func docRowCell(t *testing.T, row string, n int) string {
	t.Helper()
	fields := strings.Split(row, "|")
	if len(fields) < n+2 {
		t.Fatalf("malformed flag-table row %q — want the five columns Flag | Env twin | Default | Units | Meaning", row)
	}
	return strings.TrimSpace(fields[n])
}

// docSection returns the body of the doc's `## <heading>` section: the lines
// after the heading line up to, not including, the next `## ` heading. ok is
// false when no heading matches.
func docSection(doc, heading string) (string, bool) {
	var body strings.Builder
	inSection := false
	for line := range strings.SplitSeq(doc, "\n") {
		switch {
		case inSection && strings.HasPrefix(line, "## "):
			return body.String(), true
		case strings.HasPrefix(line, "## ") && strings.TrimSpace(line) == heading:
			inSection = true
		case inSection:
			body.WriteString(line)
			body.WriteString("\n")
		}
	}
	if inSection {
		return body.String(), true
	}
	return "", false
}

// TestConfigurationDoc is the DOCS-02 drift guard: the binary's --help is
// parsed first (with its own vacuity and env-only assertions, so the check
// can never pass vacuously), then docs/CONFIGURATION.md must match it in
// both directions — every registered flag as a row with the printed default,
// every documented flag actually registered — plus the row order, the
// env-only section and the doc-level anchors.
func TestConfigurationDoc(t *testing.T) {
	t.Parallel()
	bin := smokeBinary(t)

	help := runHelpCleanEnv(t, bin)
	flags := parseHelpFlags(help)
	if len(flags) < minRegisteredFlags {
		t.Fatalf("the --help parse saw %d registered flags, want at least %d — the iterator would pass vacuously below that", len(flags), minRegisteredFlags)
	}
	_, envTail, found := strings.Cut(help, helpEnvOnlyBlockTitle)
	if !found {
		t.Fatalf("--help output lacks the %q block:\n%s", helpEnvOnlyBlockTitle, help)
	}
	for _, name := range envOnlyKnobs {
		if !strings.Contains(envTail, name) {
			t.Fatalf("--help output does not name the env-only variable %s in its footer block:\n%s", name, help)
		}
	}

	doc, err := os.ReadFile(configurationDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", configurationDocPath, err)
	}
	text := string(doc)
	for _, a := range configurationDocAnchors {
		if !strings.Contains(text, a.literal) {
			t.Fatalf("%s does not contain %q — %s", configurationDocPath, a.literal, a.pins)
		}
	}

	rows := parseDocFlagRows(text)
	rowByName := make(map[string]string, len(rows))
	rowOrder := make([]string, 0, len(rows))
	for _, r := range rows {
		if _, dup := rowByName[r.name]; dup {
			t.Fatalf("%s has more than one flag-table row for --%s — one row per registered flag", configurationDocPath, r.name)
		}
		rowByName[r.name] = r.row
		rowOrder = append(rowOrder, r.name)
	}

	// Forward direction: every registered flag has a row whose env-twin and
	// default cells carry exactly what the help prints. A default the help
	// does not print may not be invented (a hand-typed default cell may not
	// carry a digit).
	for _, f := range flags {
		row, ok := rowByName[f.name]
		if !ok {
			t.Fatalf(
				"%s has no flag-table row for --%s (the help prints default %q) — every registered flag needs a row in --help order; a flag added, renamed or re-defaulted in internal/config without a doc edit fails this iterator",
				configurationDocPath,
				f.name,
				f.defval,
			)
		}
		envCell := docRowCell(t, row, 2)
		defCell := docRowCell(t, row, 3)
		if f.envTwin != "" {
			if !strings.Contains(envCell, f.envTwin) {
				t.Fatalf("the --%s row's env-twin cell %q does not name the help-printed twin %s", f.name, envCell, f.envTwin)
			}
		} else if strings.Contains(envCell, "SUPKV") {
			t.Fatalf("the --%s row's env-twin cell %q names a twin although --help prints none", f.name, envCell)
		}
		if f.defval != "" {
			if !strings.Contains(defCell, f.defval) {
				t.Fatalf(
					"the --%s row's default cell %q does not contain the help-printed default %q — defaults are byte-copied from the --help capture, never hand-typed",
					f.name,
					defCell,
					f.defval,
				)
			}
		} else if strings.ContainsAny(defCell, "0123456789") {
			t.Fatalf("the --%s row's default cell %q carries a value although --help prints no default for it — a hand-typed default is prohibited", f.name, defCell)
		}
	}

	// Reverse direction: every --flag token the doc names is registered — a
	// documented-but-nonexistent flag is fatal, with the token named.
	tokens := docFlagTokenPattern.FindAllString(text, -1)
	if len(tokens) == 0 {
		t.Fatalf("%s names no --flag token — the reverse check would pass vacuously", configurationDocPath)
	}
	for _, token := range tokens {
		if token == "--help" {
			continue // the scope paragraph names the --help capture itself
		}
		if !flagRegistered(flags, strings.TrimPrefix(token, "--")) {
			t.Fatalf("%s documents %s, which the binary's --help does not register — a documented-but-unregistered flag is fatal", configurationDocPath, token)
		}
	}

	// Order: the doc's rows preserve the --help registration order, checked
	// row-by-row against the parsed help order.
	if len(rowOrder) != len(flags) {
		t.Fatalf("%s's flag table carries %d rows for %d registered flags — the sets must be equal", configurationDocPath, len(rowOrder), len(flags))
	}
	for i, f := range flags {
		if rowOrder[i] != f.name {
			t.Fatalf("%s's flag-table row %d is --%s, want --%s — the table preserves the --help registration order", configurationDocPath, i+1, rowOrder[i], f.name)
		}
	}

	body, ok := docSection(text, docEnvOnlyHeading)
	if !ok {
		t.Fatalf("%s has no %q section — the five env-only knobs need their own entries", configurationDocPath, docEnvOnlyHeading)
	}
	for _, name := range envOnlyKnobs {
		if !strings.Contains(body, name) {
			t.Fatalf("%s's Environment-only section does not name %s", configurationDocPath, name)
		}
	}
}

// flagRegistered reports whether name is among the parsed --help flags — the
// reverse-direction membership check.
func flagRegistered(flags []helpFlag, name string) bool {
	for _, f := range flags {
		if f.name == name {
			return true
		}
	}
	return false
}

// TestConfigurationDocRelations pins the doc's validation-relations section
// against internal/config/validate.go's rule catalog — read as source, so a
// validation rule added without a doc row fails the anchors — plus the
// operator-facing relation literals with the exit-2 behavior, and the
// must-match-on-all-nodes list with its OPERATIONS cross-link.
func TestConfigurationDocRelations(t *testing.T) {
	t.Parallel()

	source, err := os.ReadFile(validateSourcePath)
	if err != nil {
		t.Fatalf("read %s: %v", validateSourcePath, err)
	}
	catalog := catalogRulePattern.FindAllStringSubmatch(string(source), -1)
	if len(catalog) < minCatalogRules {
		t.Fatalf("the validate.go rule parse found %d rules, want at least %d — the relations check would pass vacuously below that", len(catalog), minCatalogRules)
	}

	doc, err := os.ReadFile(configurationDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", configurationDocPath, err)
	}
	text := string(doc)

	relations, ok := docSection(text, "## Validation relations")
	if !ok {
		t.Fatalf("%s has no %q section — every validate.go catalog rule must have a doc row there", configurationDocPath, "## Validation relations")
	}
	relationRows := 0
	for line := range strings.SplitSeq(relations, "\n") {
		if docRuleRowPattern.MatchString(line) {
			relationRows++
		}
	}
	if relationRows < minRelationRows {
		t.Fatalf(
			"%s's validation-relations section carries %d rule rows, want at least %d — the catalog check would pass vacuously below that",
			configurationDocPath,
			relationRows,
			minRelationRows,
		)
	}
	for _, m := range catalog {
		if !strings.Contains(relations, m[1]) {
			t.Fatalf(
				"%s's validation-relations section does not name the validate.go catalog rule %q — an added validation rule without a doc row fails the anchors",
				configurationDocPath,
				m[1],
			)
		}
	}
	for _, a := range configurationRelationsAnchors {
		if !strings.Contains(relations, a.literal) {
			t.Fatalf("%s's validation-relations section does not contain %q — %s", configurationDocPath, a.literal, a.pins)
		}
	}

	mustMatch, ok := docSection(text, "## Must match on all nodes")
	if !ok {
		t.Fatalf("%s has no %q section — the knobs that must agree on every node need their own list", configurationDocPath, "## Must match on all nodes")
	}
	for _, a := range configurationMustMatchAnchors {
		if !strings.Contains(mustMatch, a.literal) {
			t.Fatalf("%s's must-match section does not contain %q — %s", configurationDocPath, a.literal, a.pins)
		}
	}
}
