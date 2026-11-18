# Phase 2: Engine — HLC Versioning and LWW Store - Pattern Map

**Mapped:** 2026-11-02
**Files analyzed:** 12 (5 production + 7 test)
**Analogs found:** 8 / 12 (all test files + package doc have exact convention analogs; the 4 new production source files are greenfield — no engine code exists yet)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/app/doc.go` (rewrite) | doc | n/a | itself + `internal/port/doc.go` | exact (self) |
| `internal/app/clock.go` (new) | service (stateful core type) | transform (state machine) | none — greenfield; RESEARCH.md Kulkarni Tick/Update | none |
| `internal/app/version.go` (new) | model (value type) / utility (packing) | transform | none — greenfield; RESEARCH.md RFC 9562 layout | none |
| `internal/app/store.go` (new) | service (in-memory store) + config (Config struct) | CRUD (in-memory) | none — greenfield; RESEARCH.md Pattern 2 | none |
| `internal/app/errors.go` (new) | utility (error taxonomy) | n/a | none — greenfield; RESEARCH.md Pattern 4 | none |
| `internal/app/app_test.go` (modify: remove skeleton) | test | n/a | itself | exact (self) |
| `internal/app/version_test.go` (new) | test (unit + property) | n/a | `internal/app/app_test.go` | exact (conventions) |
| `internal/app/clock_test.go` (new) | test (unit, table-driven) | n/a | `internal/app/app_test.go` | exact |
| `internal/app/store_test.go` (new) | test (unit, table-driven) | n/a | `internal/app/app_test.go` | exact |
| `internal/app/merge_test.go` (new) | test (property, TEST-01) | n/a | `internal/app/app_test.go` + RESEARCH.md generator skeleton | exact (conventions) |
| `internal/app/hlc_prop_test.go` (new) | test (property, TEST-02) | n/a | `internal/app/app_test.go` + RESEARCH.md generator skeleton | exact (conventions) |
| `internal/app/props_test.go` (new, optional — planner decides: one shared helpers file vs per-file helpers) | test utility | n/a | `internal/app/app_test.go` | exact (conventions) |

Not touched this phase (confirmed by D-03 and layout rules): `internal/port/doc.go` (no interfaces declared in Phase 2), `cmd/sup-kv/main.go` (binary logic is Phase 5), `go.mod` (zero new dependencies, D-04), `.golangci.yml` (boundary unchanged).

## Pattern Assignments

### Test files: `version_test.go`, `clock_test.go`, `store_test.go`, `merge_test.go`, `hlc_prop_test.go`, `props_test.go` (all test roles)

**Analog:** `internal/app/app_test.go` (29 lines — the only existing test file; it IS the established pattern)

**File header / package convention** (`internal/app/app_test.go` lines 1-3):
```go
package app_test

import "testing"
```
- External test package `app_test` — enforced by the `testpackage` linter (`.golangci.yml` line 426-427; only `*_internal_test.go` may be internal, and only if non-exported access is genuinely needed — it is not expected to be).
- Vanilla `testing` only — no assertion library (go-ultimate mandate; `testifylint` config exists but testify itself is not in the module graph).

**Core test pattern — `t.Parallel()` + table-driven + `t.Run` subtests** (`internal/app/app_test.go` lines 8-29):
```go
func TestAppSkeleton(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "package app builds and race-tests",
			got:  "app",
			want: "app",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}
```
Copy exactly: `t.Parallel()` on the outer test AND every subtest (`paralleltest.check-cleanup: true`, `.golangci.yml` lines 203-204), anonymous struct table with a `name` field, `got`/`want` comparison, failure message formatted `"got %q, want %q"` style, tab indentation (gofumpt/gofmt/golines formatters).

**Comment convention** (lines 5-7): every exported test function carries a sentence-case doc comment ending in a period (`godot` linter; comments are checked even in tests — `godoclint` is excluded in tests but `godot` is not).

**`app_test.go` itself is MODIFIED:** delete `TestAppSkeleton` (superseded by real engine tests — CONTEXT.md "Existing Code Insights"). Either leave the file as a home for package-level test scaffolding (e.g. the seed flag) or remove the file entirely if helpers move to `props_test.go` — planner decides.

**Property-test additions (no codebase analog — from 02-RESEARCH.md "Code Examples"):** the seeded-generator pattern (`newRNG(t)` with `rand.New(rand.NewPCG(hi, lo))`, seed printed via `t.Logf`, replay hook), the adversarial `nextWallTime` fake-clock walk, and the clock-free merge-law skeleton (`TestMerge_laws` shape) are fully specified in 02-RESEARCH.md lines 328-381. Lint note: `gochecknoglobals`/`gochecknoinits`/`funlen`/`gocognit` are EXCLUDED for `_test.go` files (`.golangci.yml` lines 452-467), so a package-level `var seed = flag.Int64(...)` replay flag and table-heavy test funcs are lint-safe; helpers must still use `t.Helper()` (go-ultimate, `thelper` linter).

---

### `internal/app/doc.go` (doc, rewrite)

**Analog:** itself (current content, lines 1-9) and `internal/port/doc.go` (lines 1-9)

**Current pattern** (`internal/app/doc.go` lines 1-9):
```go
// Package app will hold the sup-kv application core: the pure decision
// logic — the LWW-Element-Map CRDT store, hybrid logical clock and record
// versions — with no I/O of its own.
//
// This is the Phase 1 skeleton placeholder. The package carries no code yet;
// the engine lands in Phase 2. The lint gate forbids networking, filesystem,
// syscall and gRPC/protobuf imports here — see the sup-kv-app-no-io depguard
// rule in .golangci.yml.
package app
```
Rewrite keeps: `// Package app ...` first line starting the sentence with "Package app" (`revive:package-comments`), the I/O-boundary statement, the pointer to the depguard rule. Drop: the "Phase 1 skeleton placeholder / carries no code yet" paragraph. Add (per go-ultimate public-API discipline cited in RESEARCH.md): the contracts callers rely on — LWW semantics summary, tombstone-visibility rules, ownership/aliasing (defensive copies of `Data`), concurrency safety of Store and Clock, and the sentinel errors to match. Doc-comment lines must stay ≤ 97 chars (`godoclint max-len`, `.golangci.yml` lines 143-147).

---

### `internal/app/clock.go` (service, transform) — greenfield

**No code analog.** Algorithm source: Kulkarni Tick/Update pseudocode in 02-RESEARCH.md lines 280-298, plus sup-kv additions (overflow rule, init-from-dump-max, drift signal). Design anchors in `docs/IDEA.md`:
- lines 64-66: HLC state `(l, c)`; `Tick` on local write/delete; `Update` on receiving any foreign version; initialized at startup from the dump's max `l`; counter overflow bumps `l` by 1 ms.
- line 148: `drift_warn` 60s default.
- D-02: constructor takes `now func() time.Time`; exactly one seam; method signatures `Tick()` / `Update(remote)`.

File conventions to follow (the only in-repo "source" style references): sentence-case doc comment on the package-visible types (style of `doc.go`), single import declaration (`grouper.import-require-single-import: true`), stdlib imports only — `time`, `sync` are legal here (depguard `sup-kv-app-no-io` denies only `net`/`os`/`syscall`/`google.golang.org/`). Store `l` as `int64` milliseconds (`t.UnixMilli()`), never a `time.Time` (Pitfall 8 in RESEARCH.md).

### `internal/app/version.go` (model/utility, transform) — greenfield

**No code analog.** Byte-layout spec: RFC 9562 packing table in 02-RESEARCH.md lines 301-326 (octet-by-octet: `l` big-endian in 0-5, `0x70 | c>>8` / `c` in 6-7, `0x80 | nodeID>>10` etc.). Design anchor `docs/IDEA.md` lines 67-74: 48-bit `unix_ts_ms` = `l`; 12-bit `rand_a` = `c`; 62-bit `rand_b` = 16-bit NodeID high bits + filler; byte-wise comparison equals `(l, c, NodeID)` order. Use `bytes.Compare`/`slices.Compare` on `[16]byte` for production comparison (RESEARCH.md "Don't Hand-Roll"); field-by-field compare exists only inside the property test as reference model.

### `internal/app/store.go` (service, CRUD) + Config — greenfield

**No code analog.** Structural pattern: RESEARCH.md Pattern 2 (merge applies externally-versioned records; strictly-greater displaces, equal is a no-op) and the semantics table from Pitfall 1 (`Get: !ok || rec.Deleted → ErrNotFound`; `SetIfAbsent: ok → ErrAlreadyExists` regardless of `Deleted`). Design anchors `docs/IDEA.md`: lines 53-57 (per-key Record fields, map under RWMutex, LWW-Element-Map merge by greater RecordVersion, tombstones kept indefinitely), line 145 (`value_max_bytes` 61440), lines 104-106 (Phase 3 JSON dump — keep `Record` fields exported: `Key`-adjacent shape `RecordVersion`, `Data`, `Deleted`, `Checksum`).

Lint-driven shape constraints (from `.golangci.yml`):
- `embeddedstructfieldcheck.forbid-mutex: true` (lines 91-92) — do NOT embed `sync.RWMutex`; use a named field `mu sync.RWMutex`.
- `revive:enforce-map-style: make` (lines 251-253) — build the internal map with `make(map[string]Record)`.
- Exported `Record` fields satisfy `errchkjson.report-no-exported` (line 99) for Phase 3.

### `internal/app/errors.go` (utility) — greenfield

**No code analog.** Pattern 4 in 02-RESEARCH.md lines 199-200: `var ErrNotFound, ErrAlreadyExists, ErrValueTooLarge = errors.New(...)`; empirically verified lint-clean under the vendored strict config (gochecknoglobals has a built-in `errors.New` exception — probe documented in RESEARCH.md). Naming `ErrXxx` enforced by `errname`/`revive:error-naming`; error strings lowercase, no trailing punctuation (`revive:error-strings`, `string-format` rule, `.golangci.yml` lines 313-317). Callers branch with `errors.Is`, never `==`.

## Shared Patterns

### Test conventions
**Source:** `internal/app/app_test.go` (whole file)
**Apply to:** all 7 test files (classification rows 6-12)
External `app_test` package, vanilla `testing`, `t.Parallel()` everywhere including subtests, table-driven with `name` field, `got`/`want` `t.Errorf` formatting, `t.Helper()` in helpers, no assertion library.

### Architecture boundary (machine-enforced import gate)
**Source:** `.golangci.yml` lines 68-80 (rule `sup-kv-app-no-io`)
**Apply to:** `clock.go`, `version.go`, `store.go`, `errors.go`, `doc.go`
```yaml
sup-kv-app-no-io:
  files:
    - '**/internal/app/**'
    - '!$test' # exempts *_test.go from the boundary
  deny:
    - pkg: net # plain string = prefix match (catches net, net/http, net/url)
    - pkg: os
    - pkg: syscall
    - pkg: google.golang.org/
```
Legal imports for the engine: `time`, `sync`, `bytes`, `errors`, `fmt`, `hash/crc32`, `slices` — all `$gostd`. `math/rand` v1 is denied repo-wide (`use math/rand/v2`, lines 34-35); use `math/rand/v2` with a local `rand.New(rand.NewPCG(...))` in tests only. Test files are exempt from the boundary (`'!$test'`) but nothing in this phase needs I/O even in tests. No logging imports in `internal/app` — D-01: drift and losers are return values (RESEARCH.md Pitfall 9).

### Doc-comment discipline
**Source:** `internal/app/doc.go` lines 1-9, `internal/port/doc.go` lines 1-9
**Apply to:** every new file; every exported identifier (go-ultimate public-API discipline)
Package comment starts "Package app ..."; each exported type/func gets a sentence-case comment ending in a period (`godot`); doc lines ≤ 97 chars (`godoclint`); document concurrency safety, ownership/aliasing, and matching sentinel errors.

### Quality gates (unchanged from Phase 1)
**Source:** `Makefile` lines 6-12
**Apply to:** every task commit and the phase gate
```
lint: go tool golangci-lint config verify && go tool golangci-lint run ./...
test:  go test -race -cover ./...
```
Fast per-task loop: `go test -race ./internal/app/` + `go tool golangci-lint run ./internal/app/...`. No `//nolint` without a linter-specific written reason (`nolintlint`, `.golangci.yml` lines 200-202; AGENTS.md deviation 3). Go version stays `go 1.27.1` (go.mod line 3).

### Error taxonomy
**Source:** 02-RESEARCH.md Pattern 4 (lint-probe verified)
**Apply to:** `errors.go`; every `Get`/`SetIfAbsent`/`Set`/`Merge` signature and the CORE-01/03/05 test expectations
Sentinel errors via `errors.New`, matched with `errors.Is`; Phase 5 maps them 1:1 to NotFound/AlreadyExists/InvalidArgument RPC codes.

## No Analog Found

Files with no close code analog (planner builds them from RESEARCH.md code examples + `docs/IDEA.md` design anchors, following the shared patterns above):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/app/clock.go` | service | transform (state machine) | No stateful engine types exist yet; repo has zero non-main production Go code. Algorithm: Kulkarni Tick/Update (RESEARCH.md lines 280-298). |
| `internal/app/version.go` | model/utility | transform | No bit-packing/value-type code exists. Spec: RFC 9562 layout table (RESEARCH.md lines 301-326). |
| `internal/app/store.go` | service | CRUD (in-memory) | No store/CRUD code exists. Structure: RESEARCH.md Pattern 2 + Pitfall 1 semantics table. |
| `internal/app/errors.go` | utility | n/a | No error taxonomy exists. Pattern 4, empirically lint-verified. |

## Metadata

**Analog search scope:** whole tracked tree (`git ls-files`: 4 Go files total — `cmd/sup-kv/main.go`, `internal/app/doc.go`, `internal/app/app_test.go`, `internal/port/doc.go`); constraint files `.golangci.yml`, `go.mod`, `Makefile`, `docs/IDEA.md`; `.claude/skills/` and `.agents/skills/` absent (no project skills).
**Files scanned:** 8 tracked files read in full (all < 300 lines except `.golangci.yml` at 535 lines, read in full once)
**Tracked-source check:** all named analogs verified via `git ls-files -- <path>` (non-empty output); no gitignored/mirror paths referenced.
**Pattern extraction date:** 2026-11-02
