package main_test

// The PERF-01…PERF-06 publication guard (08-10 Task 1): docs/PERFORMANCE.md
// is the phase's one measurement ledger, and this test pins it against
// deletion, section reordering and drift. It asserts the mandated headings in
// their mandated order, the model's profile conditions, the burst's
// guarantee/non-guarantee split, the disk-cost trade, the amplification
// method and target-media gap, the memory formula's terms, the absent-peer
// statement, the producing command, and one hardware block plus a command
// inside EACH measured section (a document-wide count would let one section
// borrow a neighbour's block).
//
// The threshold literals are FORMATTED FROM internal/testutil's constants —
// never hand-typed — so a threshold change without a doc change fails here.
// TestSpecDeviationsBurstRow closes the D-02 registry half: the S-FAIL-15 row
// must cite the shipped standing tests and the documented bounds instead of
// the old hot-keys-only wording.
//
// HONEST SCOPE: this is a presence/drift guard only. It proves the ledger
// exists and carries its mandated statements and the current threshold
// strings; the TRUTH of any measurement (that the published p50/p99, burst
// rate, amplification or memory point matches what the binary does on the
// named hardware) is what `make perf` itself establishes, not something a
// substring check can verify.

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/Djarvur/sup-kv/internal/testutil"
)

// The two documents under guard, relative to this package's directory — the
// authored docs themselves are the input, never copied fixtures.
const (
	performanceDocPath    = "../../docs/PERFORMANCE.md"
	specDeviationsDocPath = "../../docs/SPEC-DEVIATIONS.md"
)

// performanceAnchors pins every mandated statement of docs/PERFORMANCE.md.
// Each literal is chosen from the authored ledger's own wording: renaming a
// heading or dropping a condition breaks the guard loudly. The entries are
// matched against a whitespace-normalized document, so line wrapping is not a
// way to keep a stale sentence alive.
var performanceAnchors = []struct {
	literal string
	pins    string
}{
	{"## Write-latency model", "the model section heading — the mandated FIRST section of the ledger (D-01)"},
	{"## Burst at 1000 writes/s", "the burst section heading (PERF-02)"},
	{"## Disk-write amplification", "the amplification section heading (PERF-03); the model section must also link here"},
	{"## Memory model", "the memory section heading (PERF-04)"},
	{"## Absent peer", "the absent-peer section heading (PERF-05)"},
	{"## Running the measurement", "the run-instructions section heading (PERF-06)"},

	{"dump_debounce", "the debounce knob the model derives its cadence from"},
	{"dump_max_wait", "the max-wait knob the model derives its latency from"},
	{"50 ms", "the default profile's debounce value"},
	{"1 ms", "the fast profile's debounce value"},
	{"50 ms max-wait", "the fast profile's max-wait value with its knob name"},
	{"in-memory dump", "the fast profile's dump condition — it is never stated standalone"},
	{"real-disk dump", "the default profile's dump condition"},
	{"two loopback", "the fast profile's adapter condition"},
	{"race off", "the measurement's race-detector condition"},
	{"U(0, MaxWait) + dump_time", "the sustained-flow latency formula"},
	{"p50 ≈ MaxWait/2 + dump", "the sustained-flow p50 model"},
	{"p99 ≈ MaxWait + dump", "the sustained-flow p99 model"},
	{"≈ 14 writes/s", "the sequential single-client ceiling from Debounce + dump time"},
	{"by design", "the default profile reaches dump_max_wait + dump time by design, not by accident"},

	{"disk cost", "the profiles are a disk/latency trade — the cost is stated, not implied"},
	{"about 1 dump/s", "the default profile's dump cadence at the 1 s max-wait"},
	{"about 20 dumps/s", "the fast profile's dump cadence at the 50 ms max-wait"},
	{"rewrites the whole state", "each dump's full-generation cost (ADR 0003's policy stands)"},
	{"(#disk-write-amplification)", "the disk-cost paragraph links to the amplification section where the bytes are measured"},

	{"Guaranteed at the measured rate", "the burst's guarantee list heading"},
	{"Not guaranteed", "the burst's non-guarantee list heading"},
	{"at least 990 accepted writes/s", "the burst's documented rate floor (formatted from PerfBurstMinRate elsewhere)"},
	{"15 s", "the burst's send-window length"},
	{"10 000 distinct keys", "the burst's distinct-key shape"},
	{"one first datagram per write", "the burst's datagram guarantee"},
	{"empty within 5s", "the unacked set's drain bound (the derived threshold anchor is 5s)"},
	{"60 KiB values", "the value size the burst does NOT guarantee"},
	{"rates above 1 000/s", "the rate the burst does NOT guarantee"},
	{"an absent or unresponsive peer", "the peer state the burst does NOT guarantee"},

	{"bytes-per-dump", "the amplification method's primary quantity"},
	{"Δ dumps × dump size", "the amplification numerator's method"},
	{"write_bytes", "the /proc cross-check's field name"},
	{"Spec open question #2", "the target-media gap's reference"},
	{"eMMC", "the target-grade media the gap names"},
	{"target-grade", "the row is never presented as target-grade media"},

	{"logical state", "the memory formula's first term"},
	{"1.37", "the dump-bytes ratio over value bytes"},
	{"live heap", "the memory model's peak-live term"},
	{"alloc churn", "the memory model's allocation term, distinct from peak live"},
	{"RSS", "the memory model's resident-set term"},
	{"peak-live-B", "the precise dump-transient source (the persist benchmark's metric)"},
	{"64 KiB", "the streamed dump's buffer size (PERSIST-06)"},
	{"key-dominated", "the caveat that key overhead can dominate a small-value state"},
	{"value-dominated", "the caveat's other half"},

	{"O(1)", "the absent-peer notification cost claim"},
	{"per-key index", "the mechanism that makes the notification path key-local"},
	{"unacked set", "the absent-peer condition (a growing unconfirmed set)"},

	{"make perf", "the one operator-facing command that produced the numbers"},
	{"SUPKV_PERF=1", "the gate the operator target opens"},
	{"-p 1", "the package serialization that keeps one measurement out of the next"},
	{"internal/testutil/perf.go", "the one threshold place (D-06)"},
	{"make bench-smoke", "CI keeps the threshold-free smoke target (D-13/D-14)"},
	{"Linux", "the perf target is Linux-only (the stand harness is Linux-only)"},

	{"CPU=", "the hardware block's CPU field"},
	{"logical=", "the hardware block's logical-CPU field"},
	{"RAM=", "the hardware block's RAM field"},
	{"load=", "the hardware block's load field"},
	{"root-disk=", "the hardware block's disk/device/filesystem field"},
	{"go=", "the hardware block's Go version field"},
	{"module=", "the hardware block's module/VCS field"},
	{"race=", "the hardware block's race-state field"},
	{"GOMAXPROCS=", "the hardware block's GOMAXPROCS field"},
	{"commit", "the measured commit is recorded beside the block (test binaries report vcs unknown)"},
}

// performanceForbidden pins the claims this plan removed: the standalone
// fast-profile number, and the old registry wording that admitted the burst
// bound held only on hot keys. Both are matched against the
// whitespace-normalized document.
var performanceForbidden = []struct {
	phrase string
	why    string
}{
	{"~10 ms under 1000 writes/s.", "the fast-profile number must never appear standalone — it is only valid with its in-memory-dump, 1 ms/50 ms, two-loopback, race-off conditions"},
	{"holds only on hot keys today", "the S-FAIL-15 bound now holds at the documented distinct-key scale and is proven by the shipped standing tests"},
}

// performanceSources is the test/benchmark each measured section must name,
// so a reader can find the executable half of every claim.
var performanceSources = []struct {
	name string
	pins string
}{
	{"TestPerfDefaultProfileLatency", "the default-profile measured row (internal/stand)"},
	{"TestPerfFastProfileLatency", "the fast-profile measured row (internal/grpcapi)"},
	{"TestPerfBurstDistinctKeys", "the distinct-key burst row (internal/stand)"},
	{"TestPerfAmplification", "the disk-amplification row (internal/stand)"},
	{"TestPerfMemoryPoints", "the memory points row (internal/stand)"},
	{"TestPerfAbsentPeerWriteCost", "the paired absent/live write-cost gate (internal/app/repl)"},
	{"TestRed_R02_burst", "the frozen red contract the burst row is a twin of"},
	{"TestEngine_burstBoundedConfirmation", "the deterministic dump-cadence proof"},
}

// measuredSections are the five sections that publish numbers: each must
// carry exactly ONE hardware block and at least one producing command INSIDE
// its own body, so no section can borrow a neighbour's block.
var measuredSections = []string{
	"## Write-latency model",
	"## Burst at 1000 writes/s",
	"## Disk-write amplification",
	"## Memory model",
	"## Absent peer",
}

// thresholdAnchors formats every published threshold from
// internal/testutil/perf.go by ONE fixed rule: durations through
// time.Duration.String(), the burst rate through %.0f, and the two ratios as
// the sentence fragment "≤ %.1f×". A threshold changed in Go without a doc
// change fails TestPerformanceDoc here.
func thresholdAnchors() []string {
	return []string{
		testutil.PerfDefaultP50.String(),
		testutil.PerfDefaultP99.String(),
		testutil.PerfFastMean.String(),
		testutil.PerfFastP99.String(),
		testutil.PerfBurstDrain.String(),
		fmt.Sprintf("%.0f", testutil.PerfBurstMinRate),
		fmt.Sprintf("≤ %.1f×", testutil.PerfBytesPerDumpRatio),
		fmt.Sprintf("≤ %.1f×", testutil.PerfAbsentPeerRatio),
	}
}

// docSections splits a markdown document on its "## " headings and returns
// each heading's body (the heading itself excluded). A section's body ends at
// the next "## " line, so a per-section assertion cannot see a neighbour's
// text.
func docSections(doc string) map[string]string {
	sections := make(map[string]string)
	var (
		heading string
		body    strings.Builder
	)
	flush := func() {
		if heading != "" {
			sections[heading] = body.String()
		}
		body.Reset()
	}
	for line := range strings.SplitSeq(doc, "\n") {
		if strings.HasPrefix(line, "## ") {
			flush()
			heading = strings.TrimSpace(line)
			continue
		}
		body.WriteString(line)
		body.WriteByte('\n')
	}
	flush()
	return sections
}

// TestPerformanceDoc is the PERF-01…PERF-06 drift guard: the ledger must
// exist, must open with the write-latency model before any other section, and
// must carry every anchor, every threshold formatted from the Go constants,
// every named source, no forbidden claim, and one hardware block plus a
// producing command per measured section.
func TestPerformanceDoc(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(performanceDocPath)
	if err != nil {
		t.Fatalf("read %s: %v — the measurement ledger must exist; it is written from a real `make perf` run (PERF-06)", performanceDocPath, err)
	}
	doc := string(raw)

	// The model is the ledger's FIRST section: a reader meets the conditions
	// before any number.
	for line := range strings.SplitSeq(doc, "\n") {
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		if line != "## Write-latency model" {
			t.Fatalf("%s: the first ## section is %q — the write-latency model must be the FIRST section, before any measured row (D-01)", performanceDocPath, strings.TrimSpace(line))
		}
		break
	}

	normalized := strings.Join(strings.Fields(doc), " ")
	for _, a := range performanceAnchors {
		literal := strings.Join(strings.Fields(a.literal), " ")
		if !strings.Contains(normalized, literal) {
			t.Fatalf("%s does not contain %q — %s", performanceDocPath, a.literal, a.pins)
		}
	}
	for _, f := range performanceForbidden {
		phrase := strings.Join(strings.Fields(f.phrase), " ")
		if strings.Contains(normalized, phrase) {
			t.Fatalf("%s still contains the stale claim %q — %s", performanceDocPath, f.phrase, f.why)
		}
	}
	for _, literal := range thresholdAnchors() {
		if !strings.Contains(normalized, literal) {
			t.Fatalf(
				"%s does not contain the threshold %q formatted from internal/testutil/perf.go — a threshold change without a doc change leaves the ledger stale (D-06)",
				performanceDocPath,
				literal,
			)
		}
	}
	for _, source := range performanceSources {
		if !strings.Contains(normalized, source.name) {
			t.Fatalf("%s does not name %q — %s", performanceDocPath, source.name, source.pins)
		}
	}

	sections := docSections(doc)
	for _, heading := range measuredSections {
		body, found := sections[heading]
		if !found {
			t.Fatalf("%s has no %q section — every measured row lives in its own section", performanceDocPath, heading)
		}
		if count := strings.Count(body, "hardware: "); count != 1 {
			t.Fatalf("%s section %q carries %d hardware blocks, want exactly 1 — a per-section block is what lets a reader rerun that row (T-08-25)", performanceDocPath, heading, count)
		}
		if !strings.Contains(body, "make perf") && !strings.Contains(body, "go test") {
			t.Fatalf("%s section %q names no producing command (`make perf` or `go test`) — a number without its command is unreproducible", performanceDocPath, heading)
		}
	}

	// The model section must reference the amplification section by link, not
	// merely share a document with it: that is where the fast profile's disk
	// cost becomes a measured number.
	model := sections["## Write-latency model"]
	if !strings.Contains(model, "(#disk-write-amplification)") {
		t.Fatalf("%s: the write-latency model does not link to (#disk-write-amplification) — the disk-cost paragraph must point at the measured bytes (BL-2)", performanceDocPath)
	}
}

// performanceFamilyPattern matches one supkv_ family name in the ledger. The
// doc's own spelling drives the cross-check: whatever family it names must be
// declared in the exporter.
var performanceFamilyPattern = regexp.MustCompile(`supkv_[a-z][a-z0-9_]*`)

// TestPerformanceDocFamilies is the exporter cross-check (T-08-27's
// documented-name half): every supkv_ family the ledger names must appear in
// internal/metrics/metrics.go, so the doc cannot send an operator to a metric
// the product does not export.
func TestPerformanceDocFamilies(t *testing.T) {
	t.Parallel()
	doc, err := os.ReadFile(performanceDocPath)
	if err != nil {
		t.Fatalf("read %s: %v — the ledger must exist before its families can be checked", performanceDocPath, err)
	}
	exporter, err := os.ReadFile(metricsSourcePath)
	if err != nil {
		t.Fatalf("read %s: %v", metricsSourcePath, err)
	}
	names := performanceFamilyPattern.FindAllString(string(doc), -1)
	if len(names) == 0 {
		t.Fatalf("%s names no supkv_ family — the cross-check would pass vacuously", performanceDocPath)
	}
	for _, name := range names {
		if !strings.Contains(string(exporter), name) {
			t.Errorf("%s documents %s, which internal/metrics/metrics.go does not declare", performanceDocPath, name)
		}
	}
}

// TestSpecDeviationsBurstRow is D-02's registry half: the S-FAIL-15 row must
// be CLOSED against the shipped tests, not left saying the burst bound holds
// only on hot keys. It asserts the row's status, its three test citations and
// its two documented bounds, and rejects the old wording.
func TestSpecDeviationsBurstRow(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(specDeviationsDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", specDeviationsDocPath, err)
	}
	const rowPrefix = "| S-FAIL-15 |"
	required := []struct {
		literal string
		pins    string
	}{
		{"fixed-in-v1.1(PERF-02)", "the status that records this phase's closure"},
		{"TestRed_R02_burst", "the frozen red contract's burst scenario"},
		{"TestPerfBurstDistinctKeys", "the operator-runnable distinct-key burst row"},
		{"TestEngine_burstBoundedConfirmation", "the deterministic dump-cadence proof"},
		{"990", "the documented achieved-rate floor (PerfBurstMinRate)"},
		{"5 s", "the documented drain bound (PerfBurstDrain)"},
	}
	var row string
	for line := range strings.SplitSeq(string(raw), "\n") {
		normalized := strings.Join(strings.Fields(line), " ")
		if !strings.HasPrefix(normalized, rowPrefix) {
			continue
		}
		if row != "" {
			t.Fatalf("%s carries more than one %q row — the registry has one row per ID", specDeviationsDocPath, strings.TrimSpace(rowPrefix))
		}
		row = normalized
	}
	if row == "" {
		t.Fatalf("%s has no %q row — D-02's registry closure is the row, not only the test", specDeviationsDocPath, strings.TrimSpace(rowPrefix))
	}
	for _, want := range required {
		if !strings.Contains(row, want.literal) {
			t.Fatalf("%s %q row does not contain %q — %s; row: %s", specDeviationsDocPath, strings.TrimSpace(rowPrefix), want.literal, want.pins, row)
		}
	}
	if strings.Contains(row, "holds only on hot keys today") {
		t.Fatalf(
			"%s %q row still says the burst bound holds only on hot keys today — the documented distinct-key bound is proven by the shipped standing tests (D-02); row: %s",
			specDeviationsDocPath,
			strings.TrimSpace(rowPrefix),
			row,
		)
	}
}
