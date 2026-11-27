# Phase 9 Patterns: Documentation and Release

**Mapped:** 2026-11-27
**Sources:** 09-CONTEXT.md (D-01…D-10), 09-RESEARCH.md (§1–14), working tree at `4cf390a`.
Every excerpt below was read from the cited file at this commit.

The phase's one governing pattern is **doc-is-test-input** (CONTEXT `<code_context>`):
a document is authored from a machine source (the binary's `--help`, `metrics.Snapshot`,
`persist.EncodeTo`, the Makefile targets), then a Go test reads the repo doc by relative
path and fails loudly on drift — in either direction. Every new artifact below either
*becomes* that pattern's input (docs), *extends* it (new doc-anchor tests), or is *consumed
by* it (release tooling, code surfaces).

---

## 1. File inventory and classification

| # | File | New/Mod | Role | Data flow | Closest existing analog |
|---|---|---|---|---|---|
| 1 | `docs/CONFIGURATION.md` | NEW | Doc under test (flag table) | binary `--help` (stderr, exit 0, clean env) → doc table → iterator test both directions | `docs/OPERATIONS.md` `## Configuration validation` + `TestMain_helpListsEnvOnly` |
| 2 | `docs/DEPLOYMENT.md` | NEW | Doc under test (executable blocks) | topology claims → marked blocks run verbatim by harness | `docs/EXAMPLES.md` + `TestExamplesDoc` |
| 3 | systemd unit (`deploy/sup-kv.service` or `docs/`, planner's choice) | NEW | Deploy artifact, CI-verified | unit ExecStart args ↔ documented flags; `systemd-analyze verify` + ExecStart-args smoke in CI | no in-repo unit; shape from `main.go` flag defaults + goreleaser `files:` consumers |
| 4 | `docs/FORMATS.md` | NEW | Doc under test (byte-identical) | fixed `storage.Record` set → `persist.EncodeTo` bytes must equal doc's example; checksum recipe recomputed | `internal/app/persist/codec.go` determinism contract + `codec_test.go` |
| 5 | `docs/ARCHITECTURE.md` | NEW | Prose contract (anchor-guarded) | as-built package graph (`go list ./...`) → doc → anchor test | `AGENTS.md` Layout section + `api_docs_test.go` idiom |
| 6 | ADR index (`docs/adr/README.md` or `index.md`) | NEW | Prose contract | 12 ADR files' frontmatter → index rows | `docs/adr/0010-…md` frontmatter convention |
| 7 | Simulator guide (new file or ARCHITECTURE section) | NEW | Prose contract | `internal/app/internal/simnet` + Makefile `red-run`/`race-soak`/`perf` targets → guide | `Makefile` `## target:` doc-comments + `docs/PERFORMANCE.md` "Running the measurement" |
| 8 | Release-procedure doc (new file or CONTRIBUTING section) | NEW | Executed claim (documented, then performed) | tag → `make release-snapshot` → verify archive → tag → goreleaser → asset check | `docs/setup/branch-protection.md` "every command below is re-runnable verbatim" |
| 9 | `CONTRIBUTING.md` | NEW | Prose contract | `make lint/test/bench/…` reality → doc | `AGENTS.md` Conventions + Makefile target docs |
| 10 | `SECURITY.md` | NEW | Prose contract | OPERATIONS trust model + API transport envelope → disclosure doc | `docs/OPERATIONS.md` `## Trust model` |
| 11 | `README.md` | MOD | Doc under test (quickstart blocks + claims) | blocks run verbatim; proven-claim checker; darwin demotion statement (ADR 0010) | `cmd/sup-kv/readme_test.go` marker contract |
| 12 | `docs/OPERATIONS.md` | MOD | Doc under test (the phase's largest edit) | failure table + 7 runbooks + metrics reference + log catalog; anchored + cross-checked | `cmd/sup-kv/operations_test.go` anchor/forbidden/family idiom |
| 13 | `docs/API.md` | MOD | Doc under test | replaces `## What Phase 9 adds` with per-RPC table, latency model, key-prefix convention | `cmd/sup-kv/api_docs_test.go` one-literal-per-statement |
| 14 | `docs/EXAMPLES.md` | MOD | Doc under test (executable blocks) | new create_if_absent block + assertion-table row; M-17/18/19/21 fixes | `cmd/sup-kv/examples_test.go` `examplesAssertions` (bidirectional) |
| 15 | `docs/SPEC-DEVIATIONS.md` | MOD | Structured registry (grep-able closure) | 22 `fixed-in-v1.1(` rows → `implemented` with per-row evidence; all Links cells re-derived | `performance_test.go#TestSpecDeviationsBurstRow` row-prefix pattern |
| 16 | `docs/setup/branch-protection.md` | MOD | Executed claim | fixed by-id `gh api` command run once, output recorded | the doc's own "Verify (read side)" section |
| 17 | `CHANGELOG.md` | MOD | Closure document (hand-written) | 1.1.0 entry in 1.0.0's prose style; scrub seven `(Phase N)` suffixes | existing `## [1.0.0] - 2026-11-17` entry |
| 18 | `AGENTS.md` | MOD | Stack audit target (D-10) | corrected against `go.mod` + `go list ./...`; deviations 1–3 and origin header untouched | current Layout/Conventions prose (M-32…M-37 list) |
| 19 | `.claude/CLAUDE.md` | MOD | Stack audit target (D-10) | edit inside `<!-- GSD:stack-start … -->` … `stack-end` markers (lines 25/156) or rewrite markers+section | same markers (M-38…M-44 list) |
| 20 | `docs/IDEA.md` (+ README banner wording) | MOD (may be no-op per row) | Banner-routed historical rows | M-24…M-31 dispositions: banner covers or content fix; DOC-12 bar is README-vs-banner match | IDEA.md's existing HISTORICAL banner |
| 21 | `docs/PERFORMANCE.md` | MOD (likely small) | Cross-check surface | DOCS-10 wording vs existing hardware-block ledger | already the pattern (perf_test.go guards it) |
| 22 | new machine-check tests in `cmd/sup-kv` (package `main_test`) | NEW | Machine checks (D-04) | read docs by `../../docs/…`, verify against binary/source | `readme_test.go`, `operations_test.go`, `api_docs_test.go`, `examples_test.go`, `performance_test.go` |
| 23 | FORMATS encode test in `internal/app/persist` | NEW | Machine check beside codec (D-04 exception) | reads `../../../docs/FORMATS.md`, byte-compares `EncodeTo` output | `internal/app/persist/codec_test.go` (external `persist_test` package) |
| 24 | `internal/app/storage/store.go` + `doc.go` | MOD | Comment scrub (no behavior) | M-48: drop "per D-03, internal/port holds no interfaces" | the two cited comment sites (store.go:~100, doc.go:6) |
| 25 | `internal/udprepl/replproto/supkv/v1/supkv.proto` | MOD | Comment scrub (D-07) | lines 16/26 `(D-02)`/`(D-01)` removed → `make proto` regenerates committed pb.go | Phase-6 06-10 scrub of `api/supkvapi/v1/supkvapi.proto` (now clean — verify only) |
| 26 | `.goreleaser.yaml` | MOD | Release tooling | `goos: [linux]` (darwin out, D-06); `files:` gains DEPLOYMENT.md + unit (D-05); `header:` slicing (DOC-24) | current file's own comments prescribe the second-release change |
| 27 | `.github/workflows/release.yml` | MOD (likely unchanged) | Release tooling | documents what exists (D-08); release procedure references it | current tag-triggered workflow |
| 28 | `Makefile` | MOD | Release tooling + M-52 | `help` target from `## target:` comments; M-49 buf-breaking comment fix; possible doc-check target | existing `## target:` comment convention |
| 29 | `.github/workflows/ci.yml` | MOD (comment + possible unit leg) | CI tooling | M-49 buf-gate comment fix; `systemd-analyze verify` / ExecStart smoke (planner discretion) | gate matrix cell shape; keep 5 check-run names byte-identical |
| 30 | `.github/dependabot.yml` | MOD | Comment scrub | M-50: GHAS comment stale since 2026-11-18 public flip | the file's own comment |
| 31 | Q-09 dead code (`lastLocal`, `cover`, `persist.Parse`, `ownHandle`) | VERIFY (expected no-op) | Code deletion | D-09 verify-then-delete; research §8: all four already gone — record the greps | — |

---

## 2. The doc-anchor test pattern (D-04's home: `cmd/sup-kv`, package `main_test`)

Five precedents establish the idiom. All read repo docs by relative path — never fixtures.

### 2.1 Presence-anchor guard (`operations_test.go` — model for the OPERATIONS extension, CONFIGURATION anchors, ARCHITECTURE/CONTRIBUTING/SECURITY anchors)

Structure: a doc-path const, a literal table with a `pins` explanation per entry, a
forbidden-phrase table matched against whitespace-normalized text, and a first-section rule.

```go
// cmd/sup-kv/operations_test.go:22-54 (abridged)
const operationsDocPath = "../../docs/OPERATIONS.md"

var operationsAnchors = []struct {
	literal string
	pins    string
}{
	{"## Trust model", "the trust-model section heading — the mandated FIRST section of the file"},
	{"ack_timeout_max", "REPL-10 datagram-loss convergence bound"},
	{"Restart=on-failure", "the systemd reaction to exit code 1"},
	// ... ~60 literals, each chosen from the authored doc's own wording
}

var operationsForbidden = []struct {
	phrase string
	why    string
}{
	{"failed-final-dump exit code 1", "a failed final dump exits with code 3; ..."},
}
```

The normalized-match trick (line wrapping is not a way to keep a stale sentence alive):

```go
// operations_test.go:161-170
normalized := strings.Join(strings.Fields(doc), " ")
for _, f := range operationsForbidden {
	phrase := strings.Join(strings.Fields(f.phrase), " ")
	if strings.Contains(normalized, phrase) {
		t.Fatalf("%s still contains the stale claim %q — %s", operationsDocPath, f.phrase, f.why)
	}
}
```

The first-section rule (reuse for "the failure table must follow the runbooks" or
"Configuration validation" ordering if mandated):

```go
// operations_test.go:147-155
for line := range strings.SplitSeq(doc, "\n") {
	if !strings.HasPrefix(line, "## ") { continue }
	if line != "## Trust model" {
		t.Fatalf("%s: the first ## section is %q — the trust model must be the FIRST section, ...")
	}
	break
}
```

The existing doc→exporter family grep is the seed of the new two-directional metrics check:

```go
// operations_test.go:181-207 (abridged)
var supkvFamilyPattern = regexp.MustCompile(`supkv_[a-z][a-z0-9_]*`)
func TestOperationsDocFamilies(t *testing.T) {
	// every supkv_ family the runbook names must appear in internal/metrics/metrics.go
	names := supkvFamilyPattern.FindAllString(string(doc), -1)
	if len(names) == 0 { t.Fatalf("... the cross-check would pass vacuously") }
	for _, name := range names { ... strings.Contains(string(exporter), name) ... }
}
```

**Upgrade for DOCS-07/D-09 (research §5, §11.3):** replace/extend this one-directional
grep with structured iteration over `metrics.Snapshot(metrics.Sources{})`, which renders
the *complete* family/label vocabulary at zero state — both directions fail
(exported-not-documented and documented-not-exported). The seam:

```go
// internal/metrics/metrics.go:219-230
func Snapshot(s Sources) []Family {
	families := replicationFamilies(s.Unacked(), s.KnownPeers(), s.Retransmitted(), s.MergeStats())
	...
	return append(families, runtimeFamilies()...)
}
// "A zero-state snapshot renders every family with zero-valued samples — an idle
//  node still publishes its full surface."
```

### 2.2 One-literal-per-statement guard (`api_docs_test.go` — model for the API.md extension and ARCHITECTURE.md)

```go
// api_docs_test.go:26-35
var apiDocAnchors = []struct {
	literal string
	pins    string
}{
	{"# sup-kv API", "the document title"},
	{"## Errors", "the error-table section heading"},
	{"key not found", "storage.ErrNotFound — an absent or tombstoned key"},
	// one literal per mandated fact; the `pins` column names what a failure loses
}
```

API.md's own tail names the phase's pending deliverables — the extension replaces them:

```markdown
<!-- docs/API.md tail today -->
## What Phase 9 adds
...the full per-RPC failure table..., the write-latency model, and the key-prefix
convention. Treat those sections as pending until Phase 9 lands them;
```

and `apiDocAnchors` already pins that section, so the plan must flip these three anchors
(`{"DOCS-05", ...}`, `{"per-RPC failure table", ...}`, `{"write-latency model", ...}`,
`{"key-prefix convention", ...}`) when the sections land — deleting the pending section
without updating the test fails the gate.

### 2.3 Executable-block harness (`examples_test.go` — model for DEPLOYMENT.md marked blocks and README changes)

The marker extraction contract (a marker selects the NEXT fenced block; `###` title keys
it; zero blocks or an unclosed fence is fatal):

```go
// examples_test.go:53-104 (abridged)
const examplesMarker = "<!-- examples-verify -->"
type docBlock struct { Title string; Body string }
func extractMarkedBlocks(t *testing.T, path, marker string) []docBlock { ... }
```

The bidirectional assertion table — every new block needs a row, every row a block:

```go
// examples_test.go:307-324
var examplesAssertions = []struct {
	title  string
	expect []string
}{
	{title: "Start the demo pair", expect: []string{"SERVING"}},
	{title: "Wait for replication", expect: []string{`"acked": true`, "ackedCount", "totalPeers"}},
	{title: "Stop the pair", expect: []string{"demo pair stopped"}},
}
// both directions fatal: "has no marked block titled ..." / "marked block ... is not
// in the assertion table — every executed block must carry an asserted output"
```

Execution machinery the DEPLOYMENT blocks inherit unchanged: `runBashVerbatim` (bash -e,
stdin, output to file, process group + SIGKILL reap, `docBlockBudget` clamped inside
`t.Deadline()`), `supkvBinDir` (binary on PATH as `sup-kv` — the release-archive
contract), `grpcurlShim` (`go tool grpcurl` — never a PATH stranger), `docRunEnv`
(PATH replaced, not appended), `keepSurvivors`/`reapSurvivors`, PID-file cleanup, and the
reserved port band: marked blocks use `[27900, 28100)` (research Pitfall 15).

M-fixes riding EXAMPLES.md: M-17's `grep -q SERVING` matches `NOT_SERVING` — the quoted
`grep -q '"SERVING"'` form (already in README.md:61) is the fix, and the assertion table's
`{title: "Start the demo pair", expect: []string{"SERVING"}}` is equally
NOT_SERVING-satisfiable, so the entry must become a quoted shape (`"status": "SERVING"`,
research Pitfall 7). M-18: "The listing names the three services" — four are listed
(Health, reflection v1, reflection v1alpha, SupKVService). State carries across blocks
(shared dir, doc order) — place the new create_if_absent block deliberately (Pitfall 5).

### 2.4 Threshold-from-constant formatting + registry row guard (`performance_test.go` — model for CONFIGURATION defaults and the registry gate)

Formatting from constants, never hand-typed (the same rule keeps CONFIGURATION.md
defaults honest — but sourced from the binary's `--help`, not Go constants):

```go
// performance_test.go:175-186
func thresholdAnchors() []string {
	return []string{
		testutil.PerfDefaultP50.String(),
		fmt.Sprintf("%.0f", testutil.PerfBurstMinRate),
		fmt.Sprintf("≤ %.1f×", testutil.PerfBytesPerDumpRatio),
	}
}
```

Per-section hardware block + producing command (the shape OPERATIONS' metrics reference
and DEPLOYMENT's measured claims inherit):

```go
// performance_test.go:270-281
sections := docSections(doc)   // splits on "## ", body ends at next heading
for _, heading := range measuredSections {
	body, found := sections[heading]
	if count := strings.Count(body, "hardware: "); count != 1 { t.Fatalf("... want exactly 1 ...") }
	if !strings.Contains(body, "make perf") && !strings.Contains(body, "go test") { t.Fatalf("... unreproducible") }
}
```

The doc-side hardware-block format every new measured claim copies:

```markdown
<!-- docs/PERFORMANCE.md:73-76 -->
hardware: CPU=Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz; logical=16; RAM=62.7 GiB; load=4.00 5.75 6.89; root-disk=/dev/mapper/ubuntu--vg-ubuntu--lv (ext4,rotational=unknown); go=linux/amd64 go1.27.4; module=github.com/Djarvur/sup-kv (vcs unknown, unknown); race=off; GOMAXPROCS=16

Commit: `9cdc0f1`. Command:
`SUPKV_PERF=1 ... go test -count=1 -p 1 -timeout 30m -v -run '^TestPerfDefaultProfileLatency$' ./internal/stand/`
```

The registry row guard — **the pattern for the Phase-9 registry gate**:

```go
// performance_test.go:327-372 (abridged)
func TestSpecDeviationsBurstRow(t *testing.T) {
	const rowPrefix = "| S-FAIL-15 |"
	required := []struct{ literal, pins string }{
		{"fixed-in-v1.1(PERF-02)", "the status that records this phase's closure"},
		{"TestRed_R02_burst", "the frozen red contract's burst scenario"},
	}
	var row string
	for line := range strings.SplitSeq(string(raw), "\n") {
		normalized := strings.Join(strings.Fields(line), " ")
		if !strings.HasPrefix(normalized, rowPrefix) { continue }
		if row != "" { t.Fatalf("... the registry has one row per ID") }
		row = normalized
	}
	if row == "" { t.Fatalf("... the registry closure is the row, not only the test") }
	for _, want := range required { ... strings.Contains(row, want.literal) ... }
}
```

**Registry-gate shape for Phase 9 (D-01, research §1/§12):** one test asserting zero
table-cell `fixed-in-v1.1(` rows — the match must be the **table-status form**
(`strings.HasPrefix(normalizedLine, "| S-") && strings.Contains(line, "| fixed-in-v1.1(")`)
or must exclude the preamble, because the status-vocabulary line itself contains the
token:

```markdown
<!-- docs/SPEC-DEVIATIONS.md:12-13 — the preamble carries the same token as the rows -->
Status vocabulary (fixed enum, machine-checkable; Phase 9 greps it for unclosed rows):
- `fixed-in-v1.1(<REQ-ID>)` — live gap; the named v1.1 requirement closes it;
```

`deferred-v2(SNAP-01)` is a legitimate permanent row — the exit criterion is zero
`fixed-in-v1.1(` cells, **not** zero non-`implemented` rows (Pitfall 2). Current row
shape being flipped (22 rows):

```markdown
| S-PROTO-10 | A neighbour whose run id changed or who is seen for the first time must get an immediate full sync; ... | fixed-in-v1.1(PROTO-04) | `internal/app/repl/core.go:486-506`; REVIEW §7 S-08; G-02 |
```

Flip target: `implemented` + re-derived Links cell (symbol-verified line ranges — Pitfall
3, the 06-10 neighbouring-declaration lesson). The 22-row flip list and per-row fix
evidence are table 1 of 09-RESEARCH.md §1.

### 2.5 Binary-boundary iterator seed (`main_test.go` — model for the CONFIGURATION.md flag-table iterator)

`TestMain_helpListsEnvOnly` (main_test.go:539-561) is the existing `--help`-as-source
precedent:

```go
func TestMain_helpListsEnvOnly(t *testing.T) {
	bin := smokeBinary(t)
	out, err := runToCompletion(t, bin, nil, "--help")
	if err != nil { t.Fatalf("--help exited with %v, want the flag package's clean help exit 0", err) }
	if !strings.Contains(out, "Environment-only (no flag):") { t.Fatalf(...) }
	for _, name := range []string{"SUPKV_LOG_LEVEL", "SUPKV_LOG_FORMAT", "SUPKV_METRICS_CERT",
		"SUPKV_METRICS_KEY", "SUPKV_METRICS_CLIENT_CA"} {
		if !strings.Contains(out, name) { t.Fatalf("--help output does not name the env-only variable %s:...") }
	}
}
```

Research grounding for the iterator: `--help` output is **stderr**, exit 0 (Pitfall 4);
`config.Usage` (usage.go:27) renders it, resolving defaults from the process environment
— run the binary with a clean env so env twins do not skew printed defaults. The
registered surface is the 22 flags of `internal/config/config.go:213 register` /
`:254 registerDurationFlags` plus the five env-only knobs.

### 2.6 Log-catalog level precedent (`catalog_internal_test.go` — model for the log-message catalog cross-check)

```go
// catalog_internal_test.go:26-50 (abridged)
// TestCatalogLevels calls each catalog renderer directly and asserts exactly
// one record at the documented level with the documented message.
func TestCatalogLevels(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(old) })
	// capture renderers; assert Level + Msg per catalog entry
}
```

The catalog cross-check greps the catalog's quoted message literals in
`cmd/sup-kv/main.go` (`TestOperationsDocFamilies` precedent) and pins levels through
`TestCatalogLevels`' renderers. The ~22 message literals to cover are listed in
research §3 ("Starting sup-kv", "Dump load", "Startup gate opened", "Peer appeared",
"Socket buffers", "Orphan temp sweep removed files", WARN/ERROR/DEBUG sets).

---

## 3. Docs-under-test patterns (authoring shapes)

### 3.1 `docs/CONFIGURATION.md` (new)

Author from the binary, not the source prose: a flag table (name / default / env twin /
meaning) mirroring the `--help` surface, plus the five env-only knobs as their own
section (the `Environment-only (no flag):` help block is the anchor for that section
title), plus the validation-relation catalog from `internal/config/validate.go:71
Validate` (12 rules; `peer_timeout > hello_interval`, `ack_timeout ≤ ack_timeout_max`,
`drift_warn < max_clock_lead`, ...). The "must match on all nodes" list already exists in
OPERATIONS.md `## Scale envelope` wording — cross-link, do not duplicate divergently.
Existing prose precedent for the relation phrasing:

```markdown
<!-- docs/OPERATIONS.md:98+ ## Configuration validation (exists; CONFIGURATION.md extends the surface) -->
```

Test shape: iterator parses the built binary's `--help` (name + default + env twin per
flag) and requires each flag as a table row with the matching default — a new flag or
changed default without a doc edit fails (research §11.2).

### 3.2 `docs/DEPLOYMENT.md` (new) + systemd unit

Two-board demo topology claims get marked executable blocks (`<!-- examples-verify -->`
markers are the existing contract; the DEPLOYMENT test reuses `extractMarkedBlocks`/
`runBashVerbatim`/`supkvBinDir`/`grpcurlShim`/`docRunEnv` unchanged — the harness lives
in package `main_test`, so a DEPLOYMENT doc test belongs in the same package). The
systemd unit documents the flag surface in `ExecStart=` — author its arguments from
documented defaults (`--node-id`, `--bind-addr`, `--broadcast-addr`, `--dump-path`,
`--grpc-addr`, `--metrics-addr`), matching OPERATIONS' `Restart=on-failure` exit-code
contract (exit 1 restarts, exit 2 usage does not). CI smoke: `systemd-analyze verify`
+ ExecStart-args run (research §7). The unit ships in goreleaser archives via `files:`
(D-05; DOC-05 acceptance: `tar tzf` lists docs/ and the unit).

### 3.3 `docs/FORMATS.md` (new) + encode test beside the codec

Author the format description from the codec's own contracts — JSONL, header
`format_version`/`record_count`, one sorted record line per key, base64 data, 32-char
hex `record_version`, CRC32 checksum domain, byte stability. The codec doc-comments are
the authoritative prose source:

```go
// internal/app/persist/codec.go:119-132 (abridged)
// EncodeTo writes records as the JSONL dump to w, streaming one format line
// at a time: the header object first, then one record object per line sorted
// by key — the sort is mandatory because Go map iteration order is random and
// byte stability across generations of the same state is the contract.
// SetEscapeHTML(false) keeps keys containing ampersand, less-than and
// greater-than literal and greppable. ...
func EncodeTo(w io.Writer, records map[string]storage.Record, valueMaxBytes int) error {
```

and the byte-stability guarantee:

```go
// codec.go:93-100
// The header carries no timestamp or other volatile field, so
// D-06's byte stability across generations of the same state holds.
```

The encode test lives in `internal/app/persist` (D-04's stated exception), reading
`../../../docs/FORMATS.md` (three levels up — Pitfall 13), in the external
`persist_test` package idiom of `codec_test.go` (imports
`internal/app/persist`, `internal/app/storage`, `internal/app/version`; fixed records
need no clock injection — `version.RecordVersion` is a plain struct). Test asserts:
`EncodeTo` over a fixed record set reproduces the doc's example dump byte-for-byte, and
the doc's checksum recipe recomputes the example's checksums
(`storage.RecordChecksum(key, ver, data, deleted)` is the one checksum domain, codec.go:370).

### 3.4 README.md edits

Anchor tests exist (`readme_test.go`) — edits must keep them green: markers
`<!-- quickstart-verify -->` on both blocks, the `c3VwLWt2` round-trip assertion, the
docs/OPERATIONS.md + docs/EXAMPLES.md link requirement. The edits themselves (M-01:
`README.md:22-23` "release artifacts ... in preparation" → install-from-release section
after v1.1.0; M-02: "The block cleans up after itself" is false — add the `rm
supkv-a.json supkv-b.json` or soften the claim; M-03: the macOS derive-line description;
darwin demotion per ADR 0010's canonical consequences wording):

```markdown
<!-- docs/adr/0010-darwin-development-only.md ## Consequences (canonical for D-06) -->
- darwin leaves the release archives, or its archives are marked unsuitable for production (DOCS-10);
- the README's "Quickstart on macOS" section is aligned with the development-only status; the v1.0
  macOS quickstart remains the Linux-CI-verified + darwin loud-skip pair until revisited;
- building and unit tests on macOS remain supported.
```

The proven-claim checker scans README (plus OPERATIONS + CHANGELOG) — current bare
"proven" spots: `README.md:30`, `docs/OPERATIONS.md:40,257`, CHANGELOG 1.0.0 body
(M-05). Each occurrence must carry a named `Test…` reference in the same
sentence/paragraph or a claimed-in-doc table (mechanism is planner discretion).

### 3.5 `docs/OPERATIONS.md` extension (D-03 — the largest doc edit)

Existing section map (headings, in order): Trust model → What you are operating → Scale
envelope → Record and datagram limits → Configuration validation → Load outcomes →
Memory model → Read semantics → Not a lock service → Clock drift runbook → Monitoring →
Lifecycle logs → Lifecycle and exit codes. D-03 adds: the Spec's 18-row failure-behavior
table (rows S-FAIL-01…18; per row symptom/behavior + recovery + bound + metric + log
line), seven new runbooks (dump failure/full disk, missing peer, growing unconfirmed
set, write conflict, incompatible versions, corrupted dump, duplicate `node_id` — clock
drift is the eighth, existing), the metrics reference (Monitoring prose becomes the
typed/labels/alert table), and the full log-message catalog (Lifecycle logs carries ~13
entries today; main.go has ~22 slog sites).

Authoring templates in place: the runbook shape is the Clock drift runbook's
Symptom → Cause → Action prose (OPERATIONS.md:233+); the metrics vocabulary is
research §5's family list; the failure table's metric/log columns must reference
existing families and message literals (the cross-checks of §2.1/§2.6 fail otherwise).
The trust model must remain the FIRST `##` section (the anchor test enforces it — put
new sections after the existing ones).

### 3.6 `docs/ARCHITECTURE.md`, ADR index, simulator guide, CONTRIBUTING.md, SECURITY.md

- ARCHITECTURE.md: as-built package graph from `go list ./...` (19 packages, research
  §9); the IDEA.md banner already promises it ("The as-built architecture description
  and docs/ARCHITECTURE.md land by Phase 9"). Prose style follows AGENTS.md's Layout
  section (module-relative paths, one paragraph per package, boundary rules named with
  their depguard rule IDs).
- ADR index: rows from the 12 files' frontmatter convention:

```markdown
<!-- docs/adr/0010 frontmatter — the index's row source -->
---
adr: 0010
review-decision: "14.10 (darwin is development-only)"
informs: [DOCS-10, XPORT-05]
date: 2026-11-19
status: accepted
---
```

- Simulator guide: name `internal/app/internal/simnet`, the deterministic seeded
  network, the scenario suite, and the Makefile entry points (`red-run` via
  `scripts/red-runner` + `scripts/red-expectations.tsv`, `race-soak`, `perf`).
- CONTRIBUTING.md: dev loop = `make lint` / `make test` (`go tool golangci-lint`,
  `go test -race`), branch workflow, `make proto` for proto changes, `make
  client-example` for the nested module; keep AGENTS.md deviations 1–3 accurate
  (AGENTS.md remains the canonical conventions file — CONTRIBUTING points at it).
- SECURITY.md: distills OPERATIONS `## Trust model` (no auth/encryption, CRC32
  integrity-only, reflection always on) + API.md `## Transport and health` envelope +
  a private-reporting path. Do not contradict the trust-model wording; link it.

### 3.7 `CHANGELOG.md`

The 1.0.0 entry is the prose precedent — narrative opening paragraph, then `### Added`
bullets:

```markdown
<!-- CHANGELOG.md:10-16 (style precedent) -->
## [1.0.0] - 2026-11-17

First stable release: a leaderless replicated in-memory key-value store for
small trusted-LAN clusters — every node stays writable at all times with no
coordination, and a write confirmed to a client is never lost.

### Added
- LWW-Element-Map engine with HLC/UUIDv7 record versions, proven by property
  tests: ... (Phase 2)
```

D-07: add a hand-written `1.1.0` entry in this style **without** `(Phase N)` suffixes,
and scrub the seven existing `(Phase 2)`…`(Phase 6)` suffixes from the 1.0.0 prose
(published-text edit is explicitly mandated). Keep the Keep-a-Changelog skeleton
(`[Unreleased]` empty-first or sliced per the goreleaser decision, §4.3; compare-link
footers `[1.0.0]: https://github.com/Djarvur/sup-kv/releases/tag/v1.0.0` gain a 1.1.0 row).
Proto comment scrub: `api/supkvapi/v1/supkvapi.proto` is already clean (verify);
`internal/udprepl/replproto/supkv/v1/supkv.proto` lines 16/26 carry `(D-02)`/`(D-01)` —
remove, then `make proto` and keep the generated diff comment-only (research §10).

### 3.8 Stack audits (`AGENTS.md`, `.claude/CLAUDE.md`)

AGENTS.md targets (M-32…M-37): `go 1.27` → `go 1.27.4` (go.mod directive); "CI runs the
same targets" → CI runs the tools directly (ci.yml does not call make for lint/test);
"three packages" → the real `internal/app` split (clock, persist, repl, storage, version
+ `internal/testutil`, `internal/app/internal/{simnet,testutil}`); adapters "appear with
the phases" → name the four existing (grpcapi, udprepl, persistfile, metrics) plus
config/stand; "Phase 1 carries zero direct runtime requires" → grpc/protobuf are direct;
"two tool directives" → seven (buf, grpcurl, golangci-lint, goreleaser, govulncheck,
protoc-gen-go-grpc, protoc-gen-go). **Do not touch** the `# Origin:` first line of
`.golangci.yml`-provenance prose or the deviations 1–3 substance (project instruction:
deviations are the recorded contract; AGENTS.md prose around them is free).

CLAUDE.md targets (M-38…M-44): the stack section sits between
`<!-- GSD:stack-start source:research/STACK.md -->` (line 25) and
`<!-- GSD:stack-end -->` (line 156); planner decides edit-inside vs rewrite-section-plus-
markers. `go list ./...` (19 packages), `go.mod` go directive `1.27.4`, and the tool
block are the audit sources (research §9); a machine-check comparing doc package/tool
lists against reality is the DOC-17 acceptance shape.

---

## 4. Release tooling patterns

### 4.1 `.goreleaser.yaml` (D-05, D-06)

Current state and the two edits:

```yaml
# .goreleaser.yaml:15-22 (today: 4 archives)
builds:
  - id: sup-kv
    main: ./cmd/sup-kv
    env: [CGO_ENABLED=0]
    goos: [linux, darwin]
    goarch: [amd64, arm64]

# :33-46 — no files: config (M-04); the header comment prescribes this phase's change:
#   "from the second release on, slice the current-version section instead of shipping
#    every prior entry plus the [Unreleased] skeleton."
release:
  header: '{{ mustReadFile "CHANGELOG.md" }}'
```

Edits: `goos: [linux]` (darwin leaves the asset set; CI keeps both darwin legs —
ADR 0010: "building and unit tests on macOS remain supported"); add `files:` so
archives list DEPLOYMENT.md + the unit (DOC-05: `tar tzf dist/*.tar.gz` lists them);
resolve the header slicing per Pitfall 8 (verify the template shape against the pinned
v2.20.2 func map, or ship the whole file and discharge DOC-24's "slice" promise
explicitly in the plan). `ci.yml` gate matrix and the five check-run contexts stay
byte-identical — no ruleset change is needed (research §7).

### 4.2 `Makefile` (M-52, M-49) and `.github/workflows/release.yml`

The `help` target writes itself from the existing convention — every target already
carries a `## target:` doc-comment:

```make
# Makefile:14-22 (the comment convention `help` will parse)
## lint: strict golangci-lint gate (same pinned binary as CI via the go.mod tool directive).
lint:
	go tool golangci-lint config verify
	go tool golangci-lint run ./...
```

M-49: the `buf-breaking` recipe and its `## buf-breaking:` comment still describe the
self-destructing guard ("it self-destructs the moment the proto lands there and every
later run does the real comparison") — the proto IS on origin/master since Phase 6, so
simplify to the real comparison and fix the ci.yml buf-gate comment too. The release
procedure documents the existing targets verbatim:

```make
## release-snapshot: ... --snapshot builds all 4 targets plus checksums into
## dist/. The mandatory pre-tag gate (D-09/Pitfall 8): run this and see it
## green before any v* tag is pushed (07-05).
release-snapshot:
	go tool goreleaser release --snapshot --clean
```

`release.yml` needs no demotion change (research §7); it is the procedure doc's subject:

```yaml
# .github/workflows/release.yml:11-15
on:
  push:
    tags: ["v*"]
  workflow_dispatch: # the dispatch path runs the --snapshot form
```

### 4.3 Executed-claim pattern (`docs/setup/branch-protection.md` — model for the release procedure and M-51)

The doc's own style: "every command below is re-runnable verbatim". M-51 is proven:
the current verify command fails (`cannot iterate over: null` — the list endpoint's
objects carry no `rules` key); the by-id form works and prints exactly the documented
shape:

```bash
# docs/setup/branch-protection.md:44-48 — the broken form to replace
gh api repos/Djarvur/sup-kv/rulesets \
  --jq '.[] | select(.name == "protect master") | {name, enforcement, rules: [.rules[].type]}'

# the proven-working replacement (research §2 M-51, executed 2026-11-27):
# gh api repos/Djarvur/sup-kv/rulesets/27576625 \
#   --jq '{name, enforcement, rules: [.rules[].type]}'
# → {"enforcement":"active","name":"protect master","rules":["pull_request",
#    "required_status_checks","non_fast_forward","deletion"]}
```

The release-procedure doc copies this audit-trail discipline: each step (tag on merged
master → `make release-snapshot` → verify `dist/` contents: 2 Linux archives, no darwin,
DEPLOYMENT.md + unit present, checksums + SBOMs → push tag → verify the GitHub Release
body carries no `[Unreleased]` and no phase suffixes) is a command someone re-runs, with
expected output recorded.

---

## 5. Comment-scrub patterns (small, lint-visible)

- `internal/app/storage/store.go` Clock comment and `doc.go:6` both say
  "per D-03, internal/port holds no interfaces" while `port/dump.go`/`port/repl.go`
  define them (M-48). Fix is prose-only; keep the consumer-side Clock rationale, drop
  the false claim.
- `.github/dependabot.yml`: the "GHAS-gated on this private repo… stay off until the
  deferred public flip" comment is stale (repo public 2026-11-18) — M-50.
- replproto lines 16/26 `(D-02)`/`(D-01)` → scrub + `make proto` (§3.7).
- AGENTS.md edits must keep `make lint` green: the file documents the depguard rules and
  vendored-config provenance; refresh prose, keep rule names/justifications intact
  (Pitfall 12).

---

## 6. Cross-cutting conventions (do not break)

1. **Paths:** doc tests in `cmd/sup-kv` use `../../docs/…`; the FORMATS test in
   `internal/app/persist` uses `../../../docs/FORMATS.md` (Pitfall 13).
2. **Package:** new machine checks join `package main_test` in `cmd/sup-kv` and ride
   `go test -race ./...` — no new CI job (research §11).
3. **Markers are opt-in:** a prose fence without `<!-- …-verify -->` is never executed;
   every marked block needs an assertion-table row and vice versa (examples_test.go
   fatals both directions).
4. **Port bands:** marked DEPLOYMENT blocks use `[27900, 28100)`; allocator bands are
   28100/29100/30100/31100 (examples_test.go header, Makefile race-soak comment).
5. **grpcurl through the shim:** any executable block resolves `grpcurl` via the
   `go tool grpcurl` shim; never document a bare `grpcurl`/`sup-kv` PATH dependency the
   harness does not provide (Pitfall 16; `supkvBinDir` provides `sup-kv` for
   release-archive-style blocks).
6. **Serialization:** doc walkthrough tests are serialized with
   `//nolint:paralleltest // serialized on purpose: ...` (readme_test.go:83,
   examples_test.go:344) — keep the reason-bearing nolint form (deviation 3).
7. **Honest-scope doc comments:** every doc-anchor test opens with a comment stating
   what it proves and what it does not ("HONEST SCOPE: this is a presence/anchor guard
   only…" — operations_test.go:7-12, performance_test.go:19-24); new machine checks
   follow the same disclosure style.
8. **Vacuity guards:** any "must contain ≥1" cross-check fatals when the doc names
   nothing (`TestOperationsDocFamilies`' "the cross-check would pass vacuously") —
   apply to the metrics/log-catalog/registry gates too.
9. **`metrics.Snapshot(metrics.Sources{})` is the complete vocabulary** — the
   structured metrics cross-check needs no live node; label enums come from the family
   builders (12 discard classes, 3 dump failure classes, 4 load outcomes, 4 record
   discard classes, 5 datagram types — research §5).
10. **M-closure tracking:** per-plan disposition tables (D-02);
    `docs/reviews/2026-11-19-v1.0/docs-audit.md` is the source list and gains no sibling
    registry — dispositions name doc-fix or code-fix per M row.
