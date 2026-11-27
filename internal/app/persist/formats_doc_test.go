package persist_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
)

// Formats-doc guard: this file pins docs/FORMATS.md to the codec's own
// output. The doc's example fences must be byte-identical to persist.EncodeTo
// over the fixed record set below, the documented checksum recipe must
// recompute the example's checksums, and the documented rejection rules must
// name exactly the discard classes persist.DecodeStats defines. It is a
// byte-identity and recipe guard, not a truth claim about the format's
// wisdom: prose can be wrong in ways the bytes do not show.
//
// The test lives beside the codec it pins (D-04's stated exception to the
// cmd/sup-kv doc-anchor pattern) and reads the doc three levels up, at
// ../../../docs/FORMATS.md.
const (
	// formatsDocPath is the doc under test, relative to this package's
	// directory — the doc itself is the input, never a copied fixture.
	formatsDocPath = "../../../docs/FORMATS.md"

	// formatsExampleMarker selects the fenced block carrying the
	// three-record example dump; formatsEmptyExampleMarker selects the
	// zero-record (header-only) example. A marker takes the NEXT fenced
	// block — the cmd/sup-kv examples_test.go marker contract, in spirit:
	// prose fences are never silently enrolled.
	formatsExampleMarker      = "<!-- formats-example -->"
	formatsEmptyExampleMarker = "<!-- formats-example-empty -->"

	// formatsRejectionHeading is the section the rejection-rules coverage
	// check reads.
	formatsRejectionHeading = "## Rejection rules"

	// The fixed example record set's clock readings and node identity.
	// These values are load-bearing: the doc's example bytes and this test
	// must share them (regenerate the doc after any codec change).
	exampleNodeID   = uint16(0x5060)
	exampleCounter  = uint16(1)
	exampleAlphaL   = int64(1736937600000)
	exampleBravoL   = int64(1736937600001)
	exampleCharlieL = int64(1736937600002)
)

// formatsClassPattern extracts the discard-class literals the doc names in
// the metric-label form class=<name> (the vocabulary
// supkv_dump_records_discarded_total uses).
var formatsClassPattern = regexp.MustCompile(`class=([a-z][a-z_0-9]*)`)

// formatsFieldCase splits a Go field name at the lower-to-upper boundary so
// the class names render in the snake_case form the doc and metrics use.
var formatsFieldCase = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// formatsDocRecords returns the fixed example record set: a live record
// ("alpha"), a tombstone ("bravo", Deleted true with no data) and a live
// empty value ("charlie"). The map literal deliberately lists the keys
// non-sorted — the writer's mandatory sort is what puts the example's lines
// in key order. Versions are plain structs packed from explicit fields, so
// the encode is deterministic with no clock injection. The node id is the
// plan's 48-bit literal's low 16 bits: the RFC 9562 layout carries a 16-bit
// NodeID, and the upper bytes have no place to go.
func formatsDocRecords() map[string]storage.Record {
	return map[string]storage.Record{
		"charlie": sealed("charlie", storage.Record{
			Version: version.PackVersion(exampleCharlieL, exampleCounter, exampleNodeID),
			Data:    []byte{},
		}),
		"alpha": sealed("alpha", storage.Record{
			Version: version.PackVersion(exampleAlphaL, exampleCounter, exampleNodeID),
			Data:    []byte("hello-alpha"),
		}),
		"bravo": sealed("bravo", storage.Record{
			Version: version.PackVersion(exampleBravoL, exampleCounter, exampleNodeID),
			Deleted: true,
		}),
	}
}

// formatsDocFence returns the body of the fenced block marker selects: the
// block opened by the first fence line after the marker line, closed by the
// next fence line. A missing marker, a missing fence or an unclosed fence is
// fatal — the doc is the contract, never a silent empty result.
func formatsDocFence(t *testing.T, marker string) string {
	t.Helper()
	raw, err := os.ReadFile(formatsDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", formatsDocPath, err)
	}
	lines := strings.Split(string(raw), "\n")
	isFence := func(line string) bool { return strings.HasPrefix(strings.TrimSpace(line), "```") }
	for i, line := range lines {
		if !strings.Contains(line, marker) {
			continue
		}
		j := i + 1
		for j < len(lines) && !isFence(lines[j]) {
			j++
		}
		if j >= len(lines) {
			t.Fatalf("%s: marker %q at line %d is not followed by a fenced block", formatsDocPath, marker, i+1)
		}
		var body []string
		for j++; j < len(lines) && !isFence(lines[j]); j++ {
			body = append(body, lines[j])
		}
		if j >= len(lines) {
			t.Fatalf("%s: the fence opened after marker %q (line %d) is never closed", formatsDocPath, marker, i+1)
		}
		return strings.Join(body, "\n")
	}
	t.Fatalf("%s carries no %q marker — the byte-identity contract is gone", formatsDocPath, marker)
	return ""
}

// formatsDocSection returns the doc text from the named "## " heading to the
// next "## " heading (or end of file). A missing heading is fatal.
func formatsDocSection(t *testing.T, heading string) string {
	t.Helper()
	raw, err := os.ReadFile(formatsDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", formatsDocPath, err)
	}
	lines := strings.Split(string(raw), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != heading {
			continue
		}
		var body []string
		for j := i + 1; j < len(lines); j++ {
			if strings.HasPrefix(lines[j], "## ") {
				return strings.Join(body, "\n")
			}
			body = append(body, lines[j])
		}
		return strings.Join(body, "\n")
	}
	t.Fatalf("%s carries no %q section — the coverage contract is gone", formatsDocPath, heading)
	return ""
}

// assertDocFenceEquals compares want with the bytes of the fenced block
// marker selects, naming the first differing line on a mismatch.
func assertDocFenceEquals(t *testing.T, marker, want string) {
	t.Helper()
	got := formatsDocFence(t, marker)
	if bytes.Equal([]byte(want), []byte(got)) {
		return
	}
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	for i := range max(len(wantLines), len(gotLines)) {
		w, g := "⟨no line⟩", "⟨no line⟩"
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			t.Fatalf("the doc fence after %q differs from EncodeTo output at line %d:\n  doc:    %s\n  encode: %s",
				marker, i+1, g, w)
		}
	}
	t.Fatalf("the doc fence after %q differs from EncodeTo output in a way the line scan missed", marker)
}

// TestFormatsDocEncodeMatchesExample is DOCS-06's byte-identity guard: the
// doc's example dumps ARE the encoder's output, so a codec format change
// without a doc regeneration fails. The three-record example encodes the
// fixed record set (live value, tombstone, live empty value) and the
// zero-record example encodes an empty map — both compared byte-for-byte.
func TestFormatsDocEncodeMatchesExample(t *testing.T) {
	t.Parallel()
	t.Run("the three-record example is the encoder's own output", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		err := persist.EncodeTo(&buf, formatsDocRecords(), storage.ValueMaxBytes)
		if err != nil {
			t.Fatalf("EncodeTo(example records) error = %v, want nil", err)
		}
		assertDocFenceEquals(t, formatsExampleMarker, buf.String())
	})
	t.Run("the zero-record example is the encoder's own output", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		err := persist.EncodeTo(&buf, make(map[string]storage.Record), storage.ValueMaxBytes)
		if err != nil {
			t.Fatalf("EncodeTo(empty map) error = %v, want nil", err)
		}
		assertDocFenceEquals(t, formatsEmptyExampleMarker, buf.String())
	})
}

// TestFormatsDocChecksumRecipe proves the documented recipe executable: for
// every record line in the doc's example, storage.RecordChecksum over the
// line's own key, packed version bytes, data and deleted flag reproduces the
// decimal checksum the line carries — the one checksum domain, restated.
func TestFormatsDocChecksumRecipe(t *testing.T) {
	t.Parallel()
	dump := formatsDocFence(t, formatsExampleMarker)
	checked := 0
	for line := range strings.SplitSeq(dump, "\n") {
		if !strings.Contains(line, `"record_version"`) {
			continue
		}
		var parsed dumpLine
		err := json.Unmarshal([]byte(line), &parsed)
		if err != nil {
			t.Fatalf("unmarshal example line %q: %v", line, err)
		}
		raw, err := hex.DecodeString(parsed.Version)
		if err != nil {
			t.Fatalf("example line for %s carries a non-hex record_version %q: %v", parsed.Key, parsed.Version, err)
		}
		var ver version.RecordVersion
		if len(raw) != len(ver) {
			t.Fatalf("example line for %s carries a %d-byte record_version, want %d bytes",
				parsed.Key, len(raw), len(ver))
		}
		copy(ver[:], raw)
		want := storage.RecordChecksum(parsed.Key, ver, parsed.Data, parsed.Deleted)
		if want != parsed.Checksum {
			t.Errorf("example line for %s carries checksum %d, want %d — the documented recipe must recompute it",
				parsed.Key, parsed.Checksum, want)
		}
		checked++
	}
	if checked == 0 {
		t.Fatalf("the doc fence after %q carries no record lines — the recipe scan would pass vacuously",
			formatsExampleMarker)
	}
}

// TestFormatsDocRejectionRules proves the rejection-rules section covers
// exactly the codec's own class set, both directions: every class= name the
// section carries must be a real DecodeStats field, and every real field
// must be named. The real set is enumerated from persist.DecodeStats via
// reflection — never a hand list, so a class the codec adds or removes
// without a doc edit fails. The section must also carry the generation-level
// record_count integrity rules.
func TestFormatsDocRejectionRules(t *testing.T) {
	t.Parallel()
	statsType := reflect.TypeFor[persist.DecodeStats]()
	realClasses := make(map[string]bool)
	for field := range statsType.Fields() {
		realClasses[strings.ToLower(formatsFieldCase.ReplaceAllString(field.Name, "${1}_${2}"))] = true
	}
	if len(realClasses) == 0 {
		t.Fatal("persist.DecodeStats exposes no fields — the class enumeration is broken")
	}
	section := formatsDocSection(t, formatsRejectionHeading)
	tokens := formatsClassPattern.FindAllStringSubmatch(section, -1)
	if len(tokens) == 0 {
		t.Fatalf("%s: the %s section names no class=... discard class — the coverage scan would pass vacuously",
			formatsDocPath, formatsRejectionHeading)
	}
	named := make(map[string]bool, len(tokens))
	for _, tok := range tokens {
		name := tok[1]
		if !realClasses[name] {
			t.Errorf("%s: the %s section names class=%q, which the codec's DecodeStats does not define (real classes: %v)",
				formatsDocPath, formatsRejectionHeading, name, keysOf(realClasses))
		}
		named[name] = true
	}
	for name := range realClasses {
		if !named[name] {
			t.Errorf("%s: the %s section never names the %s discard class the codec emits",
				formatsDocPath, formatsRejectionHeading, name)
		}
	}
	for _, anchor := range []struct {
		literal string
		pins    string
	}{
		{"record_count", "the completeness census field the generation-level rules turn on"},
		{"incomplete", "fewer record lines than record_count is an incomplete generation"},
		{"tolerated", "more record lines than record_count is tolerated"},
		{"legacy", "an absent record_count field is the documented v1.0 legacy path"},
	} {
		if !strings.Contains(section, anchor.literal) {
			t.Errorf("%s: the %s section lacks %q — %s",
				formatsDocPath, formatsRejectionHeading, anchor.literal, anchor.pins)
		}
	}
}

// keysOf renders a set's members sorted for a failure message.
func keysOf(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}
