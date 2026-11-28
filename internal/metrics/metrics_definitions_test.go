package metrics_test

// The definition guard: every published supkv_*/process_* family states WHAT
// it measures in its # HELP line, and this suite asserts one semantic phrase
// per definition. It is a documentation-drift guard, not a truth proof — the
// behavior tests (TestFlush_countsOnlySuccessfulWrites, TestMetrics_values,
// TestCurrentRSS, the merge and drift suites) are the truth; this test makes
// a Help edit that quietly drops a definition fail the build.

import (
	"strings"
	"testing"

	"github.com/Djarvur/sup-kv/internal/metrics"
)

// helpTexts renders a zero-state snapshot and indexes each family's HELP text
// by family name. None of the sup-kv Helps carry a backslash or newline, so
// the rendered (escaped) text is the authored text.
func helpTexts(t *testing.T) map[string]string {
	t.Helper()
	var buf strings.Builder
	err := metrics.Render(&buf, metrics.Snapshot(zeroSources()))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	helps := make(map[string]string)
	for line := range strings.SplitSeq(buf.String(), "\n") {
		rest, ok := strings.CutPrefix(line, "# HELP ")
		if !ok {
			continue
		}
		name, help, found := strings.Cut(rest, " ")
		if !found {
			continue
		}
		helps[name] = help
	}
	return helps
}

// TestMetrics_definitions is the one-test-per-definition guard (OBSV-03,
// D-05): each family's Help must carry the phrase that states what it
// measures — pending-only conflicts, socket-accepted sends, the last dump
// ATTEMPT, wall-time-leading drift, current RSS, and the four routed Phase-4
// counter definitions. Both sides are lowercased, so the Help's sentence case
// can neither manufacture nor mask a match.
func TestMetrics_definitions(t *testing.T) {
	t.Parallel()
	helps := helpTexts(t)
	for _, tc := range []struct {
		family string
		phrase string
	}{
		{"supkv_merges_conflicts_total", "pending"},
		{"supkv_replication_datagrams_sent_total", "successfully"},
		{"supkv_dumps_duration_seconds", "last dump attempt"},
		{"supkv_clock_drift_warnings_total", "lead local wall time"},
		{"process_resident_memory_bytes", "current"},
		{"supkv_replication_listings_dropped_total", "refused by the destination"},
		{"supkv_replication_sync_streams_dropped_total", "accumulator bounds"},
		{"supkv_replication_listed_keys_evicted_total", "whole-peer eviction"},
		{"supkv_replication_notifications_regressed_total", "not greater than the tracked version"},
	} {
		t.Run(tc.family, func(t *testing.T) {
			t.Parallel()
			help, ok := helps[tc.family]
			if !ok {
				t.Fatalf("family %s has no # HELP line", tc.family)
			}
			if !strings.Contains(strings.ToLower(help), strings.ToLower(tc.phrase)) {
				t.Errorf("%s Help = %q, want the definition phrase %q", tc.family, help, tc.phrase)
			}
		})
	}
}
