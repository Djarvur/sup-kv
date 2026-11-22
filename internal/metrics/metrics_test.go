package metrics_test

// The exporter unit suite: byte-exact rendering of a family table with
// adversarial HELP and label text (Pitfall 3 — promtool-free CI means this
// test is the only thing that catches escaping bugs), writer-side sorting of
// families and labels regardless of construction order, and the HTTP
// handler's contract over httptest (status, the exact Content-Type, HELP and
// TYPE lines preceding samples).

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/metrics"
	"github.com/Djarvur/sup-kv/internal/udprepl"
)

// renderTo renders families into a string, failing the test on a write
// error — the shared harness of the two Render tests.
func renderTo(t *testing.T, families []metrics.Family) string {
	t.Helper()
	var buf strings.Builder
	err := metrics.Render(&buf, families)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return buf.String()
}

// TestRender_exact pins the byte-exact 0.0.4 rendering of a small family
// table, adversarial text included: HELP carrying a double quote, a
// backslash and an embedded newline, and a label value doing the same. The
// exposition's asymmetry is the point — a double quote passes through a
// HELP docstring unescaped but must be backslash-escaped inside a label
// value, and only the writer-side escapes make this output valid.
func TestRender_exact(t *testing.T) {
	t.Parallel()
	got := renderTo(t, []metrics.Family{
		{
			Name: "zeta_family",
			Type: metrics.Gauge,
			Help: `Zeta "quoted" help with a \ backslash`,
			Samples: []metrics.Sample{
				{Value: 0},
			},
		},
		{
			Name: "alpha_family",
			Type: metrics.Counter,
			Help: "Alpha help with a\nnewline",
			Samples: []metrics.Sample{
				{Labels: []metrics.Label{{Name: "b", Value: "two"}, {Name: "a", Value: "one"}}, Value: 1.5},
				{Labels: []metrics.Label{{Name: "esc", Value: "q\"uote\\back\nnewline"}}, Value: 42},
			},
		},
	})
	want := strings.Join([]string{
		`# HELP alpha_family Alpha help with a\nnewline`,
		`# TYPE alpha_family counter`,
		`alpha_family{a="one",b="two"} 1.5`,
		`alpha_family{esc="q\"uote\\back\nnewline"} 42`,
		`# HELP zeta_family Zeta "quoted" help with a \\ backslash`,
		`# TYPE zeta_family gauge`,
		`zeta_family 0`,
		``,
	}, "\n")
	if got != want {
		t.Fatalf("Render output:\n%q\nwant:\n%q", got, want)
	}
}

// TestRender_sorted asserts the writer does the sorting: the same family
// table built in descending construction order (families and labels alike)
// renders byte-identically to the ascending construction, and both match the
// sorted expectation — scrape diffs stay stable no matter how call sites
// assemble the table (D-04).
func TestRender_sorted(t *testing.T) {
	t.Parallel()
	reversed := renderTo(t, []metrics.Family{
		{
			Name: "zz_last",
			Type: metrics.Gauge,
			Help: "Last family.",
			Samples: []metrics.Sample{
				{Labels: []metrics.Label{{Name: "zeta", Value: "z"}, {Name: "alpha", Value: "a"}}, Value: 2},
			},
		},
		{
			Name: "aa_first",
			Type: metrics.Counter,
			Help: "First family.",
			Samples: []metrics.Sample{
				{Value: 1},
			},
		},
	})
	ordered := renderTo(t, []metrics.Family{
		{
			Name: "aa_first",
			Type: metrics.Counter,
			Help: "First family.",
			Samples: []metrics.Sample{
				{Value: 1},
			},
		},
		{
			Name: "zz_last",
			Type: metrics.Gauge,
			Help: "Last family.",
			Samples: []metrics.Sample{
				{Labels: []metrics.Label{{Name: "alpha", Value: "a"}, {Name: "zeta", Value: "z"}}, Value: 2},
			},
		},
	})
	if reversed != ordered {
		t.Fatalf("construction order leaked into the rendering:\nreversed: %q\nordered:  %q", reversed, ordered)
	}
	want := strings.Join([]string{
		`# HELP aa_first First family.`,
		`# TYPE aa_first counter`,
		`aa_first 1`,
		`# HELP zz_last Last family.`,
		`# TYPE zz_last gauge`,
		`zz_last{alpha="a",zeta="z"} 2`,
		``,
	}, "\n")
	if ordered != want {
		t.Fatalf("Render output:\n%q\nwant:\n%q", ordered, want)
	}
}

// TestMetricsHandler_serves drives the handler through a real HTTP round
// trip: status 200, the exact exposition Content-Type, and the HELP and TYPE
// lines present and strictly before the family's first sample.
func TestMetricsHandler_serves(t *testing.T) {
	t.Parallel()
	handler := metrics.NewHandler(func() []metrics.Family {
		return []metrics.Family{{
			Name:    "h_family",
			Type:    metrics.Gauge,
			Help:    "H family.",
			Samples: []metrics.Sample{{Value: 7}},
		}}
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, http.NoBody)
	if err != nil {
		t.Fatalf("NewRequestWithContext(%s): %v", srv.URL, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", srv.URL, err)
	}
	body, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if readErr != nil {
		t.Fatalf("read body: %v", readErr)
	}
	if closeErr != nil {
		t.Fatalf("close body: %v", closeErr)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/plain; version=0.0.4; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want %q", ct, "text/plain; version=0.0.4; charset=utf-8")
	}
	out := string(body)
	helpAt := strings.Index(out, "# HELP h_family ")
	typeAt := strings.Index(out, "# TYPE h_family ")
	sampleAt := strings.Index(out, "\nh_family 7\n")
	if helpAt != 0 || typeAt < 0 || sampleAt < 0 || typeAt <= helpAt || sampleAt <= typeAt {
		t.Fatalf("output lacks HELP/TYPE before the sample (help %d, type %d, sample %d):\n%s", helpAt, typeAt, sampleAt, out)
	}
}

// families pins the complete D-04/D-05 family set — names, types and the
// fixed label vocabularies — as the table TestMetrics_families asserts over.
var families = []struct {
	name string
	typ  string
}{
	{"supkv_replication_unacked_records", "gauge"},
	{"supkv_replication_unacked_oldest_age_seconds", "gauge"},
	{"supkv_replication_peers", "gauge"},
	{"supkv_replication_datagrams_sent_total", "counter"},
	{"supkv_replication_datagrams_retransmitted_total", "counter"},
	{"supkv_replication_datagrams_paced_total", "counter"},
	{"supkv_replication_send_queue", "gauge"},
	{"supkv_replication_egress_dropped_total", "counter"},
	{"supkv_replication_read_errors_total", "counter"},
	{"supkv_datagrams_discarded_total", "counter"},
	{"supkv_merges_conflicts_total", "counter"},
	{"supkv_clock_drift_warnings_total", "counter"},
	{"supkv_dumps_duration_seconds", "gauge"},
	{"supkv_dumps_total", "counter"},
	{"supkv_dumps_errors_total", "counter"},
	{"go_goroutines", "gauge"},
	{"go_memstats_alloc_bytes", "gauge"},
	{"go_memstats_sys_bytes", "gauge"},
	{"process_cpu_seconds_total", "counter"},
	{"process_resident_memory_bytes", "gauge"},
}

// zeroSources is the fake seam snapshot: every accessor returns its type's
// zero value — a node that has never replicated, merged or dumped.
func zeroSources() metrics.Sources {
	return metrics.Sources{
		Discards:      func() repl.DiscardStats { return repl.DiscardStats{} },
		KnownPeers:    func() uint64 { return 0 },
		Unacked:       func() repl.UnackedStats { return repl.UnackedStats{} },
		Sent:          func() udprepl.SentByType { return udprepl.SentByType{} },
		Retransmitted: func() uint64 { return 0 },
		MergeStats:    func() repl.MergeStats { return repl.MergeStats{} },
		DumpStats:     func() persist.DumpStats { return persist.DumpStats{} },
		Paced:         func() uint64 { return 0 },
		SendQueue:     func() int { return 0 },
		EgressDropped: func() uint64 { return 0 },
		ReadErrors:    func() uint64 { return 0 },
	}
}

// TestMetrics_families asserts the closed family contract over a fake
// snapshot: every D-04/D-05 family name appears exactly once with its TYPE
// line, every class and type label of the fixed enums renders, and the
// zero-state snapshot still publishes zero-valued samples — an idle node's
// scrape is complete, not empty (the empty-state edge).
func TestMetrics_families(t *testing.T) {
	t.Parallel()
	out := renderTo(t, metrics.Snapshot(zeroSources()))
	for _, tc := range families {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			typeLine := "# TYPE " + tc.name + " " + tc.typ
			if n := strings.Count(out, typeLine); n != 1 {
				t.Errorf("family %s: %q appears %d times, want exactly 1:\n%s", tc.name, typeLine, n, out)
			}
		})
	}
	for _, line := range []string{
		"supkv_replication_unacked_records 0\n",
		"supkv_replication_unacked_oldest_age_seconds 0\n",
		"supkv_replication_peers 0\n",
		"supkv_replication_datagrams_retransmitted_total 0\n",
		"supkv_replication_datagrams_paced_total 0\n",
		"supkv_replication_send_queue 0\n",
		"supkv_replication_egress_dropped_total 0\n",
		"supkv_replication_read_errors_total 0\n",
		"supkv_merges_conflicts_total 0\n",
		"supkv_clock_drift_warnings_total 0\n",
		"supkv_dumps_duration_seconds 0\n",
		"supkv_dumps_total 0\n",
		"supkv_dumps_errors_total 0\n",
		"supkv_datagrams_discarded_total{class=\"bad_crc\"} 0\n",
		"supkv_datagrams_discarded_total{class=\"undecodable\"} 0\n",
		"supkv_datagrams_discarded_total{class=\"bad_proto_version\"} 0\n",
		"supkv_datagrams_discarded_total{class=\"own_broadcast\"} 0\n",
		"supkv_datagrams_discarded_total{class=\"duplicate_node_id\"} 0\n",
		"supkv_datagrams_discarded_total{class=\"empty_key\"} 0\n",
		"supkv_datagrams_discarded_total{class=\"key_too_long\"} 0\n",
		"supkv_datagrams_discarded_total{class=\"tombstone_with_data\"} 0\n",
		"supkv_datagrams_discarded_total{class=\"bad_checksum\"} 0\n",
		"supkv_datagrams_discarded_total{class=\"bad_version\"} 0\n",
		"supkv_datagrams_discarded_total{class=\"bad_sync_header\"} 0\n",
		"supkv_replication_datagrams_sent_total{type=\"hello\"} 0\n",
		"supkv_replication_datagrams_sent_total{type=\"data\"} 0\n",
		"supkv_replication_datagrams_sent_total{type=\"ack\"} 0\n",
		"supkv_replication_datagrams_sent_total{type=\"data_request\"} 0\n",
		"supkv_replication_datagrams_sent_total{type=\"sync\"} 0\n",
	} {
		if !strings.Contains(out, line) {
			t.Errorf("zero-state output lacks the sample line %q:\n%s", line, out)
		}
	}
}

// TestMetrics_values pins the by-effect half of the metrics surface: Sources
// carrying nonzero readings render those exact values — the four XPORT-07
// emission families and the five wire-validation discard classes. A wiring
// mistake that swapped two accessors, or a family still rendering a constant
// zero, fails here even though TestMetrics_families (the zero-state contract)
// would pass.
func TestMetrics_values(t *testing.T) {
	t.Parallel()
	src := zeroSources()
	src.Discards = func() repl.DiscardStats {
		return repl.DiscardStats{
			EmptyKey:          11,
			KeyTooLong:        12,
			TombstoneWithData: 13,
			BadChecksum:       14,
			BadVersion:        15,
			BadSyncHeader:     16,
		}
	}
	src.Paced = func() uint64 { return 7001 }
	src.SendQueue = func() int { return 23 }
	src.EgressDropped = func() uint64 { return 3 }
	src.ReadErrors = func() uint64 { return 5 }
	out := renderTo(t, metrics.Snapshot(src))
	for _, line := range []string{
		"supkv_replication_datagrams_paced_total 7001\n",
		"supkv_replication_send_queue 23\n",
		"supkv_replication_egress_dropped_total 3\n",
		"supkv_replication_read_errors_total 5\n",
		"supkv_datagrams_discarded_total{class=\"empty_key\"} 11\n",
		"supkv_datagrams_discarded_total{class=\"key_too_long\"} 12\n",
		"supkv_datagrams_discarded_total{class=\"tombstone_with_data\"} 13\n",
		"supkv_datagrams_discarded_total{class=\"bad_checksum\"} 14\n",
		"supkv_datagrams_discarded_total{class=\"bad_version\"} 15\n",
		"supkv_datagrams_discarded_total{class=\"bad_sync_header\"} 16\n",
	} {
		if !strings.Contains(out, line) {
			t.Errorf("output lacks the by-effect sample line %q:\n%s", line, out)
		}
	}
}
