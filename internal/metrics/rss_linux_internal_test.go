//go:build linux

package metrics

// Linux-only RSS suite: /proc/self/statm is a Linux interface, so both the
// fixture table and the live probe assert only on the platform that has it —
// the non-Linux fallback is a different document (rss_other.go). White-box on
// purpose (the *_internal_test.go testpackage exemption): parseStatm and
// currentRSS are unexported, and only a same-package test can drive the
// parser with fixtures.

import (
	"os"
	"testing"
)

// TestParseStatm pins the statm parser's four shapes (OBSV-03, D-05): the
// kernel's own multi-field reading parses to the resident page count times
// the page size, while a truncated, non-numeric or negative resident field
// reports failure rather than a silently wrong byte count.
func TestParseStatm(t *testing.T) {
	t.Parallel()
	page := float64(os.Getpagesize())
	for _, tc := range []struct {
		name   string
		raw    string
		want   float64
		wantOK bool
	}{
		{name: "the kernel's own shape", raw: "2113 495 466 5 0 124 0\n", want: 495 * page, wantOK: true},
		{name: "surrounding whitespace", raw: "  2113\t495 466\n", want: 495 * page, wantOK: true},
		{name: "too few fields", raw: "2113\n", want: 0, wantOK: false},
		{name: "empty reading", raw: "", want: 0, wantOK: false},
		{name: "non-numeric resident field", raw: "2113 abc 466 5 0 124 0\n", want: 0, wantOK: false},
		{name: "negative resident field", raw: "2113 -5 466 5 0 124 0\n", want: 0, wantOK: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseStatm([]byte(tc.raw))
			if ok != tc.wantOK || got != tc.want {
				t.Fatalf("parseStatm(%q) = (%v, %v), want (%v, %v)", tc.raw, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

// TestCurrentRSS is the Linux integration half: the live probe reads this
// test process's own statm and reports a positive resident size. A zero would
// mean the file or the parser failed silently, which the parser table alone
// cannot catch.
func TestCurrentRSS(t *testing.T) {
	t.Parallel()
	if got := currentRSS(); got <= 0 {
		t.Fatalf("currentRSS() = %v, want a positive resident size", got)
	}
}
