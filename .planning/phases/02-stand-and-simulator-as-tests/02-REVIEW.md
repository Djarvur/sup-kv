---
phase: 02-stand-and-simulator-as-tests
reviewed: 2026-11-21T14:37:46Z
depth: standard
files_reviewed: 63
files_reviewed_list:
  - cmd/sup-kv/broadcast_probe_test.go
  - cmd/sup-kv/examples_test.go
  - cmd/sup-kv/main_test.go
  - .github/workflows/ci.yml
  - .github/workflows/scheduled.yml
  - internal/app/internal/simnet/simnet_ext_test.go
  - internal/app/internal/simnet/simnet.go
  - internal/app/internal/testutil/bench.go
  - internal/app/internal/testutil/rng.go
  - internal/app/persist/bench_test.go
  - internal/app/persist/codec_test.go
  - internal/app/persist/engine_blocking_test.go
  - internal/app/persist/engine_test.go
  - internal/app/persist/fuzz_dump_test.go
  - internal/app/persist/testdata/fuzz/FuzzDumpDecode/seed_00_accepted_with_tombstone
  - internal/app/persist/testdata/fuzz/FuzzDumpDecode/seed_01_accepted
  - internal/app/persist/testdata/fuzz/FuzzDumpDecode/seed_02_generation_fault
  - internal/app/persist/testdata/fuzz/FuzzDumpDecode/seed_03_generation_fault
  - internal/app/persist/testdata/fuzz/FuzzDumpDecode/seed_04_per_record_discard
  - internal/app/persist/testdata/fuzz/FuzzDumpDecode/seed_05_generation_fault
  - internal/app/persist/testdata/fuzz/FuzzDumpDecode/seed_06_generation_fault
  - internal/app/persist/testdata/fuzz/FuzzDumpDecode/seed_07_per_record_discard
  - internal/app/repl/decision_coverage_test.go
  - internal/app/repl/sim_chaos_test.go
  - internal/app/repl/sim_scenario_test.go
  - internal/app/repl/testdata/chaos-seeds.txt
  - internal/app/storage/prop_small_space_test.go
  - internal/grpcapi/bench_test.go
  - internal/grpcapi/gate_test.go
  - internal/grpcapi/red_gate_test.go
  - internal/grpcapi/server_test.go
  - internal/grpcapi/watch_senderr_test.go
  - internal/persistfile/dumpfile.go
  - internal/persistfile/fsops.go
  - internal/persistfile/fsops_test.go
  - internal/persistfile/open_seed_test.go
  - internal/persistfile/red_value_limit_test.go
  - internal/persistfile/snapshot.go
  - internal/stand/stand_delete_test.go
  - internal/stand/stand.go
  - internal/stand/stand_metrics_test.go
  - internal/stand/stand_nondarwin_test.go
  - internal/stand/stand_red_test.go
  - internal/stand/stand_test.go
  - internal/testutil/bench.go
  - internal/udprepl/bench_test.go
  - internal/udprepl/decode_negative_test.go
  - internal/udprepl/fuzz_decode_test.go
  - internal/udprepl/testdata/fuzz/FuzzDecode/seed_00_valid_hello
  - internal/udprepl/testdata/fuzz/FuzzDecode/seed_01_valid_data_replication
  - internal/udprepl/testdata/fuzz/FuzzDecode/seed_02_valid_tombstone
  - internal/udprepl/testdata/fuzz/FuzzDecode/seed_03_valid_ack
  - internal/udprepl/testdata/fuzz/FuzzDecode/seed_04_valid_data_request
  - internal/udprepl/testdata/fuzz/FuzzDecode/seed_05_valid_data_sync
  - internal/udprepl/testdata/fuzz/FuzzDecode/seed_06_corruption_undecodable
  - internal/udprepl/testdata/fuzz/FuzzDecode/seed_07_corruption_bad_crc
  - internal/udprepl/testdata/fuzz/FuzzDecode/seed_08_corruption_undecodable
  - internal/udprepl/testdata/fuzz/FuzzDecode/seed_09_corruption_undecodable
  - internal/udprepl/testdata/fuzz/FuzzDecode/seed_10_corruption_wrong_proto_version
  - internal/udprepl/testdata/fuzz/FuzzDecode/seed_11_corruption_undecodable
  - Makefile
  - scripts/red-expectations.tsv
  - scripts/red-runner
findings:
  critical: 0
  warning: 6
  info: 8
  total: 14
  necessity: 0
status: issues_found
---

# Phase 02: Code Review Report

**Reviewed:** 2026-11-21T14:37:46Z
**Depth:** standard
**Files Reviewed:** 63
**Status:** issues_found

## Summary

This is the test-infrastructure phase (stand harness move, simulator extensions, red-test gate, mutation probes, fuzz corpora, benchmarks, CI/scheduled machinery). I reviewed all 63 listed files at standard depth, read the call sites outside the list where correctness depended on them (`internal/app/repl/core.go`, `storage.Store`, `persist.Decode`, `udprepl` test helpers), and ran the gates rather than trusting them:

- `go vet ./...`, `go tool golangci-lint config verify`, `go tool golangci-lint run ./...`, and `go test -race -cover ./...` are green on the committed tree.
- The red gate works in both directions: `TestRed_R03_gateClosedUntilFullList` skips without `SUPKV_RED_TESTS=1` and fails on its documented assertion with it; the committed evidence shows all seven expected-red rows with a `fail` event, and the mutation stage's control/ mutant verdicts reproduce (I rebuilt the `sync-without-tombstones` overlay mutant independently: the sed fires, the mutant compiles, and it is killed on corpus seed 848612892191557850 with a genuine convergence failure).
- The committed fuzz corpora load and pass in seed mode (`FuzzDecode` 12 seeds, `FuzzDumpDecode` 8 seeds), and the three benchmark smokes emit `p50-ms`/`p99-ms`.
- Behavior preservation of the three production files holds: `simnet.go`'s extensions are inert in the zero `Policy` and add no goroutines/wall-clock reads; the `persistfile` `fsops` extraction reproduces the old write→fsync→rotate→rename→dir-fsync order and the old temp-hygiene on every failure path (the `fileFSOps` per-save state is clean after each failure branch).

No critical findings. The six warnings are gate-soundness and flake-coverage issues plus one silent-error harness API; they are listed below with concrete fixes. The info items are documentation/enforcement gaps, one misleading filename, and duplicated test constants. I read the necessity rubric; no necessity finding met the evidence bar (the obvious candidates are test-seam guards or guards against states that can actually occur), so `necessity: 0`.

## Warnings

### WR-01: `red-runner` cannot see a package-level failure, so a crash after the expected reds still exits green

**File:** `scripts/red-runner:289-296,347-362`
**Issue:** The JSON parser discards every event without a `Test` field (`if not name or action not in ("run","pass","fail","skip"): continue`). Go reports process-level outcomes only as package events: a panic in a non-test goroutine, a `go test -timeout` expiry, and a `TestMain` failure all emit `{"Action":"fail","Package":...,"Elapsed":...}` with no `Test` key (verified locally with `go test -json`: a package whose non-test goroutine panics prints `run TestRed`, `fail TestRed`, then a package-only `fail`, and later tests never appear). Because the expected-red rows must fail, their packages always end with a package-level fail, so the parser cannot distinguish that normal case from a crash, and the "unexpected failing test" loop only inspects names. A crash after the red rows have recorded their failures therefore yields `0 error(s)` and exit 0 while the rest of the package never ran — exactly the "a broken normal test can never hide behind the expected reds" guarantee the script's header claims.
**Fix:** Attribute package failures and scan for crash markers, mirroring the mutation stage's timeout check:

```python
package_fail = set()
failing = set()
...
if action == "fail":
    if name:
        failing.add(name)
    else:
        package_fail.add(event.get("Package", "?"))
# after the table loop:
for pkg in package_fail:
    if not any(packages.get(n) and pkg in packages[n] for n in failing):
        errors.append(f"{pkg}: package failed without any failing test event (crash or timeout)")
```
and treat `Output` lines starting with `panic:` / `fatal error:` / `panic: test timed out` as errors (the mutation stage already greps the last one).

### WR-02: `Cluster.Write` swallows the store error and then broadcasts a fabricated record

**File:** `internal/app/internal/simnet/simnet.go:714-724`
**Issue:** `_, _, _ = c.nodes[i].store.Set(key, data)` discards the error, then the harness reads `Records()[key]` back and hands whatever that is to `NotifyLocalWrite`, appending it to the write log that feeds the pointwise-supremum model. If `Set` rejects the value — over the 61440-byte `storage.Config.ValueMaxBytes`, the exact class the phase's P-01/value-limit work exercises — the node broadcasts a zero/old record (zero version, zero checksum), every receiver rejects it, and the scenario fails later as an opaque convergence divergence with a phantom entry in the model. `Cluster.Delete` (line 741) handles the same error correctly and returns it, so the asymmetry is unintended. Neither `Write` nor `Delete` calls `checkIndex`, so an out-of-range index panics with a raw slice-bounds message instead of the harness's named panic.
**Fix:** Fail loudly, as `ReplaceNodeWithState`/`SetRecvBudget` already do, and route both entry points through `checkIndex`:

```go
func (c *Cluster) Write(i int, key string, data []byte) storage.Record {
	c.checkIndex(i, "Write")
	now := c.sched.Now()
	if _, _, err := c.nodes[i].store.Set(key, data); err != nil {
		panic(fmt.Sprintf("simnet: Cluster.Write: Set %q on node %d: %v", key, i, err))
	}
	...
}
```

### WR-03: The TEST-13 meta-test walks the whole module tree, including the in-tree Go module cache, so it can pass vacuously

**File:** `internal/app/repl/decision_coverage_test.go:77-105`
**Issue:** `repoTestSources` walks from the module root and skips only `docs/`, any directory literally named `.git`, and any directory named `testdata`. The repo root also holds `.gomodcache/` and `.gocache/` (gitignored, but this checkout has `GOMODCACHE` inside the tree; a developer can configure it that way anywhere). `.gomodcache` contains thousands of dependency `_test.go` files (e.g. `google.golang.org/protobuf@.../integration_test.go`, `mvdan.cc/gofumpt@.../main_test.go`) which the test reads. Consequences: (a) a row can be satisfied by a citation living in a dependency's test file, not in any repo test — the vacuous pass TEST-13 exists to prevent; (b) the walk cost is unbounded and environment-dependent (3.1 s here); (c) a damaged cache makes the test fail for an unrelated reason. The same walk also reads `dist/`, `bin/`, `.tmp/` when present.
**Fix:** Skip hidden and build-output directories, and keep the search inside the module's own trees:

```go
if entry.IsDir() {
    name := entry.Name()
    if path == docsRoot || strings.HasPrefix(name, ".") ||
        name == "testdata" || name == "dist" || name == "bin" {
        return filepath.SkipDir
    }
}
```
(Equivalently, enumerate package dirs via `go list ./...` and only read their `_test.go` files.)

### WR-04: The workflow_dispatch seed validation is not anchored and echoes the untrusted value into a workflow command

**File:** `.github/workflows/scheduled.yml:53-55`
**Issue:** The T-2-21 check is line-oriented: `printf '%s' "$requested" | grep -Eq '^[1-9][0-9]*$'` succeeds when *any* line of a multi-line input is digits, so `"1000\n<junk>"` passes validation and the multi-line value is propagated. The untrusted value is then echoed unescaped into a workflow command (`echo "::warning::seeds input '$requested' ..."`): an embedded newline lets a dispatch input emit additional `::` runner commands (annotation/log spoofing such as `::error::` or `::add-mask::`). The value never reaches a shell unquoted and `workflow_dispatch` requires write access, so this is not an RCE; but the workflow's explicit "untrusted input, digits only" claim is not met, and the malformed value later fails the sweep with an `Atoi` error instead of the documented fallback.
**Fix:** Anchor the whole string, do not echo the raw value, and bound the sweep:

```yaml
if [[ ! "${requested}" =~ ^[1-9][0-9]{0,4}$ ]]; then
  echo "::warning::seeds input is not a positive integer <= 99999; falling back to 1000"
  echo "SUPKV_SIM_FRESH_SEEDS=1000" >> "$GITHUB_ENV"
fi
```

### WR-05: The private-port-window mitigation is per-process only; two packages and `freeTCPPort` reintroduce the pick-to-bind race

**File:** `internal/stand/stand.go:76-92,192-231`; `cmd/sup-kv/broadcast_probe_test.go:49-95`
**Issue:** The "structurally removes the pick-to-bind race" argument (T-11 flake class 1) holds only inside one process. Both `cmd/sup-kv` and `internal/stand` now carry an independent copy of the same `[28100,32678)` window plus a claim registry, and `go test ./...` (and `make race-soak`) run packages concurrently (`-p` defaults to GOMAXPROCS). Each pickers claims a port by binding it and then **closes it before the node binds**: `internal/stand` reserves its ports in `newStandPair` and launches the node later (after `standBinary`, two `freeTCPPort` calls and the loopback probe), so the port is free for the other package's registry to hand out, and one node's bind fails. Separately, `freeTCPPort` (stand.go:223-231) takes the gRPC ports from the shared ephemeral pool with the same close-then-rebind window. The Makefile's class-1 comment claims the class is mitigated, and the race-soak (which runs both packages together) is where it would surface.
**Fix:** One shared picker with a cross-process claim (e.g. `O_EXCL` lock files under a temp dir keyed by port) or disjoint per-package sub-windows (stand `[28100,30000)`, cmd/sup-kv `[30000,32678)`); allocate the gRPC ports from the same private window instead of `freeTCPPort`.

### WR-06: TEST-11's soak excludes `internal/stand`, the package the phase created and whose flake class the adjacent comment describes

**File:** `Makefile:43-58`
**Issue:** `race-soak` runs `-race -count=$(COUNT)` over `./cmd/sup-kv/ ./internal/udprepl/ ./internal/app/repl/ ./internal/app/persist/ ./internal/persistfile/ ./internal/grpcapi/` — but not `./internal/stand/`. The comment directly above it disposes of flake class 1 as "structurally mitigated: internal/stand's private window … plus the claim registry moved intact", yet the package owning that mitigation and the two-process socket scenarios is the one package the ×3-green nightly bar never repeats; the stand suite runs once per CI cell. The list matches the plan's literal wording, but it undercuts the plan's own class-1 disposition.
**Fix:** Add `./internal/stand/` to the soak set (≈7 s × 20 ≈ 2-3 min at default scale, well inside the 120-minute `race-flake` job budget), or record the deliberate exclusion and its rationale in the same comment block.

## Info

### IN-01: The bench smoke prints the percentile lines but never asserts them

**File:** `Makefile:35-41`; `.github/workflows/scheduled.yml:76-77`
**Issue:** The comment says the smoke "asserts the p50-ms/p99-ms lines appear", but `go test -bench` only prints them; nothing greps the output or fails if the metrics vanish (e.g. a renamed unit in `ReportLatencyPercentiles`, or a benchmark that returns before reporting). TEST-10's evidence is therefore observed by eye, not enforced.
**Fix:** Gate the workflow step on the output (`make bench-smoke | tee /tmp/bench.txt; test "$(grep -c 'p50-ms' /tmp/bench.txt)" -eq 3 && test "$(grep -c 'p99-ms' /tmp/bench.txt)" -eq 3`), or add a tiny benchmark whose test asserts the rendered shape.

### IN-02: `scnRunID` collides with `simRunID` for `d == 9` in a 9+-node cluster

**File:** `internal/app/repl/sim_scenario_test.go:39-44`
**Issue:** `scnRunID(nodeID, d)` overwrites the last decimal digit of the 16-digit node id. For `d == nodeID <= 9` it equals `simRunID(nodeID)` (TestScenario_restart *relies* on this for `scnRunID(2,2)`), and for `d == 9` it equals `simRunID(9)` — the run id of node 9. The doc comment says "distinct from every simRunID a cluster derives", which is false; a future 9-node scenario replacing node `i` with `d = 9` would give the replacement another live node's identity.
**Fix:** Derive replacement ids from a namespace that cannot collide (e.g. force a non-digit marker byte), keep an explicit `scnRunIDEqualsSimRunID` helper for the restart case, and correct the comment.

### IN-03: `stand_nondarwin_test.go` is constrained to `!linux`, not to darwin

**File:** `internal/stand/stand_nondarwin_test.go:1`
**Issue:** The filename is not a Go GOOS suffix (Go recognizes `_darwin.go`, not `_nondarwin.go`), so the name is inert and the `//go:build !linux` tag makes the file compile on every non-Linux platform (Windows, FreeBSD), not just darwin. The file's own comment correctly says "non-Linux", so the name is the misleading half and invites a future maintainer to assume darwin-only coverage.
**Fix:** Rename to `stand_nonlinux_test.go` (no behavior change), or keep the name and state explicitly that the constraint is `!linux`.

### IN-04: The TEST-15 small-space constants are duplicated across two test packages

**File:** `internal/app/persist/codec_test.go:32-47`; `internal/app/storage/prop_small_space_test.go:26-38`
**Issue:** The same 3×3×3 box is restated as literals in `persist_test` and `storage_test`, including `smallSpaceLBase = int64(1730000000000)` which also duplicates `testNowMs`. The comment claims they are "the same 3x3x3 box", but nothing keeps them equal; if one copy drifts, the two suites stop covering the same counterexample shape — the cross-suite shrinking property TEST-15 is built on.
**Fix:** Put the canonical values in one test-only package both importers can reach (the module-wide `internal/testutil` already exists for exactly this reason) and reference them from both, or add a compile-time equality assertion.

### IN-05: The scheduled workflow's "latest evidence" step can print a previous run's committed summary

**File:** `.github/workflows/scheduled.yml:101-109`
**Issue:** `ls -1dt .../red-evidence/*/ | head -n 1` also considers the evidence directories committed in the repo; git sets their mtimes to checkout time, i.e. "now". If `bash scripts/red-runner` fails before it creates its own evidence directory (e.g. missing table, no `go`/`python3`, unwritable tree), the step prints the newest *committed* summary — which can be green — as if it were this run's evidence. The `else` branch only covers "no summary at all".
**Fix:** Have the runner announce the directory it wrote (stdout line or `$GITHUB_ENV` entry) and have the step read that path; or filter with `find ... -newermt "$(date -u -d '-1 hour')"`.

### IN-06: The nightly chaos job never verifies how many fresh seeds actually ran

**File:** `.github/workflows/scheduled.yml:61-63`
**Issue:** TEST-16's acceptance is "1000 seeds, `-shuffle=on`, `-count=3`", but `chaosFreshSeedCount` silently falls back to 1 when `SUPKV_SIM_FRESH_SEEDS` is unset (the PR default). The job asserts nothing about the count, so a plumbing regression (renamed variable, env not propagated) would reduce the nightly sweep to a single fresh seed while the job stays green and its log still says "Chaos sweep (TEST-16)".
**Fix:** Log the count from the test and assert it in the job: `t.Logf("fresh chaos seeds: %d", n)` under `-v`, then `go test -v ... | grep -q "fresh chaos seeds: 1000"` (or `-count=3` aware), so the acceptance is machine-checked.

### IN-07: The fuzz corpora have no drift guard against their generators

**File:** `internal/udprepl/fuzz_decode_test.go:18-24,95-101`; `internal/app/persist/fuzz_dump_test.go:20-27`
**Issue:** Both file docs claim the committed `testdata/fuzz/...` corpus mirrors the generated seeds, but nothing compares them: the fuzz targets re-add `fuzzDecodeSeeds()`/`fuzzDumpSeeds()` at run time, so a stale, edited or mislabeled corpus file is undetected — including a file whose name claims a class ("corruption_bad_crc") the bytes no longer represent. Contrast the chaos corpus, which has `TestChaos_seedCorpusGenerator`.
**Fix:** Add a plain test that renders each generated seed in `go test fuzz v1` form and compares it byte-for-byte with the named file (regenerating only under an explicit env gate), or drop the committed corpus and rely on `f.Add`, which is what the runtime actually consumes.

### IN-08: The size-gate test measures a hand-copied encoder, not the production one

**File:** `internal/app/internal/simnet/simnet_ext_test.go:57-115`
**Issue:** `simWireBody` hand-mirrors `udprepl`'s private `protoBody` mapping, so the test's premise — "measures exactly what a real socket would send" — is an unverified copy. A production mapping change (new field, envelope change) leaves the test measuring an outdated shape while still passing, and the size-gate assertions then prove nothing about the real wire. The import-boundary rationale is documented, but nothing pins the mirror.
**Fix:** Add one parity test comparing `simWireSize(msg)` with the length produced by `udprepl.EncodeDatagramForTest(msg, ...)` for every `port.Message` kind (test-layer only, so the boundary rationale still holds), or expose a test-only length seam from `udprepl` and use it directly.

---

_Reviewed: 2026-11-21T14:37:46Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
