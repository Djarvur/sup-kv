// Package metrics is the sup-kv metrics/debug endpoint adapter: the repo's
// only HTTP-serving code, rendering the replication and persistence counter
// seams (the 06-03 accessors) as the classic Prometheus text format (version
// 0.0.4) and assembling the one opt-in listener's whole debug surface —
// /metrics together with /debug/vars and /debug/pprof/* (D-03), optionally
// behind client-certificate mTLS (D-02). The adapter owns rendering and
// surface assembly only: values come from the snapshot accessors a caller
// supplies (a closed Sources set, no registration lifecycle), and every
// decision — what counts, when state changes — stays in the packages those
// accessors read. It owns no listener lifecycle either: main binds, serves
// and shuts the listener down inside the D-19 order.
package metrics

import (
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

// FamilyType is a metric family's exposition type — the two kinds the sup-kv
// family set needs (D-04 deliberately scopes the exporter to counters and
// gauges; histograms would be the revisit trigger).
type FamilyType string

// The exposition types Render emits.
const (
	Counter FamilyType = "counter"
	Gauge   FamilyType = "gauge"
)

// Label is one sample label. Label values are the fixed class/type enums
// only (bounded cardinality) — key names, record values and other user data
// never become labels (the OBS-02 prohibitions).
type Label struct {
	Name  string
	Value string
}

// Sample is one time series value under its family's label set.
type Sample struct {
	Labels []Label
	Value  float64
}

// Family is one metric family: name, type and help with every sample the
// exposition renders under it. Construction order carries no meaning —
// Render sorts families and labels in the writer.
type Family struct {
	Name    string
	Type    FamilyType
	Help    string
	Samples []Sample
}

// contentType is the exposition's Content-Type: the classic text format
// version 0.0.4 with the conventional utf-8 charset parameter.
const contentType = "text/plain; version=0.0.4; charset=utf-8"

// Render writes families in the classic Prometheus text format (version
// 0.0.4): family names sorted, exactly one # HELP and one # TYPE line per
// name emitted before that name's first sample, each sample's labels sorted
// by label name, values as Go ParseFloat floats, the final line
// newline-terminated. HELP escapes backslash and line feed; label values
// escape backslash, double quote and line feed (the exposition rules —
// double quotes pass through in HELP unescaped). Sorting happens here, in
// the writer: call sites construct families in whatever order they like,
// which is what keeps scrape diffs stable (D-04).
func Render(w io.Writer, families []Family) error {
	sorted := slices.Clone(families)
	slices.SortFunc(sorted, func(a, b Family) int { return strings.Compare(a.Name, b.Name) })
	var b strings.Builder
	lastName := ""
	for _, f := range sorted {
		if f.Name != lastName {
			b.WriteString("# HELP ")
			b.WriteString(f.Name)
			b.WriteString(" ")
			b.WriteString(escapeHelp(f.Help))
			b.WriteString("\n# TYPE ")
			b.WriteString(f.Name)
			b.WriteString(" ")
			b.WriteString(string(f.Type))
			b.WriteString("\n")
			lastName = f.Name
		}
		for _, s := range f.Samples {
			writeSample(&b, f.Name, s)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// writeSample renders one sample line: the family name, the sorted label
// set (a leading comma between labels, none before the brace when the set
// is empty) and the value.
func writeSample(b *strings.Builder, name string, s Sample) {
	b.WriteString(name)
	labels := slices.Clone(s.Labels)
	slices.SortFunc(labels, func(x, y Label) int { return strings.Compare(x.Name, y.Name) })
	if len(labels) > 0 {
		b.WriteString("{")
		for i, l := range labels {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(l.Name)
			b.WriteString("=\"")
			b.WriteString(escapeLabelValue(l.Value))
			b.WriteString("\"")
		}
		b.WriteString("}")
	}
	b.WriteString(" ")
	b.WriteString(strconv.FormatFloat(s.Value, 'g', -1, 64))
	b.WriteString("\n")
}

// escapeHelp applies the HELP docstring escapes: backslash doubles and line
// feed becomes a literal \n — the only two the exposition defines for HELP,
// so a double quote passes through unescaped.
func escapeHelp(s string) string {
	return strings.NewReplacer(`\`, `\\`, "\n", `\n`).Replace(s)
}

// escapeLabelValue applies the label-value escapes: backslash doubles,
// double quote gains its backslash and line feed becomes a literal \n.
func escapeLabelValue(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(s)
}

// NewHandler returns the /metrics handler over snapshot: every GET renders
// the snapshot's families fresh — the exporter is a pure formatter over the
// supplied accessors, with no registry and no caching of its own. Any other
// method is a 405; the Content-Type header names the exact exposition
// format so scrapers never guess.
func NewHandler(snapshot func() []Family) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", contentType)
		_ = Render(w, snapshot()) // a vanished scraper mid-write is not an error worth acting on.
	})
}
