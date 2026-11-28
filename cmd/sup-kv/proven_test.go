package main_test

// The DOCS-09 proven-claim gate (09-08 Task 1): every word-boundary
// prove/proves/proven/proving occurrence in the three prose docs a newcomer
// or operator actually reads — README.md, docs/OPERATIONS.md and
// CHANGELOG.md — must name its evidence inside the same blank-line-delimited
// paragraph: a backticked `Test…` name or a cited _test.go file. A bare
// "proven" is unverifiable marketing, so the checker makes one fail
// `make test`.
//
// HONEST SCOPE: this checker proves only that every performance/convergence
// claim names its evidence — it does NOT judge whether the named test
// actually proves the claim. That judgment is the named test's own job (the
// bound scenarios, property tests and red scenarios carry their own
// assertions); a paragraph that cites an irrelevant test would pass here and
// must be caught by review, not by this scan.
//
// Matching mechanics: the claim pattern is word-bounded and case-sensitive
// (`\b(prove|proves|proven|proving)\b`), so "improved" never matches; the
// paragraph scope is the blank-line-delimited text block, so a test named in
// a neighboring paragraph does not count. Evidence extraction is a backticked
// `Test[A-Z]…` identifier or the literal `_test.go` inside the same
// paragraph. The vacuity guard keeps the rule from passing vacuously: below
// three claim occurrences across the three files the corpus would prove
// nothing about the rule, and the scan fails instead of passing.

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// provenDocPaths are the three scanned prose docs, relative to this
// package's directory — the docs themselves are the input, never copied
// fixtures (the operationsDocPath and readmeDocPath consts are reused where
// they exist).
var provenDocPaths = []string{
	readmeDocPath,
	operationsDocPath,
	"../../CHANGELOG.md",
}

// provenClaimRe matches the claim words at word boundaries, case-sensitive:
// "improved" contains "prove" mid-word and must never match.
var provenClaimRe = regexp.MustCompile(`\b(?:prove|proves|proven|proving)\b`)

// provenTestRefRe matches a backticked Go test identifier (`Test…` with an
// uppercase letter after the prefix) — the executable evidence a claim names.
var provenTestRefRe = regexp.MustCompile("`Test[A-Z][A-Za-z0-9_]*`")

// provenClaimsFloor is the vacuity floor: the smallest total claim count
// across the three files at which the scan's passing verdict means anything.
// Today's corpus carries seven occurrences (three in README.md, two in
// docs/OPERATIONS.md, two in CHANGELOG.md); a corpus below three would make
// an all-green scan indistinguishable from a broken scan.
const provenClaimsFloor = 3

// TestProvenClaims scans the three prose docs and fails on every claim word
// whose paragraph carries no named test, then asserts the corpus is large
// enough for the verdict to be meaningful. Failures name the file, the line
// and the bare claim word, so the fix is a one-paragraph edit.
func TestProvenClaims(t *testing.T) {
	t.Parallel()
	total := 0
	for _, path := range provenDocPaths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		lines := strings.Split(string(raw), "\n")
		var para []string
		start := 0
		flush := func() {
			if len(para) == 0 {
				return
			}
			text := strings.Join(para, "\n")
			named := provenTestRefRe.MatchString(text) || strings.Contains(text, "_test.go")
			for _, loc := range provenClaimRe.FindAllStringIndex(text, -1) {
				total++
				if named {
					continue
				}
				lineNo := start + strings.Count(text[:loc[0]], "\n")
				t.Errorf("%s:%d: bare proven claim %q — the paragraph names no test "+
					"(DOCS-09: every prove/proves/proven/proving paragraph must cite a "+
					"backticked `Test…` name or a _test.go file)",
					path, lineNo, text[loc[0]:loc[1]])
			}
			para = para[:0]
		}
		for i, line := range lines {
			if strings.TrimSpace(line) == "" {
				flush()
				continue
			}
			if len(para) == 0 {
				start = i + 1
			}
			para = append(para, line)
		}
		flush()
	}
	if total < provenClaimsFloor {
		t.Fatalf("the proven-claim scan counted only %d claim words across the three "+
			"docs (floor %d) — below the floor a passing scan is vacuous, so it fails: "+
			"either the claims were scrubbed (keep the rule honest with a floor review) "+
			"or the scan is broken", total, provenClaimsFloor)
	}
}
