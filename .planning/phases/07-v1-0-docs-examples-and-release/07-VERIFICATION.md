---
phase: 07-v1-0-docs-examples-and-release
verified: 2026-11-18T19:12:00Z
status: passed
score: 10/10 must-haves verified
covered_files: [".github/workflows/release.yml", ".gitignore", ".goreleaser.yaml", ".planning/phases/07-v1-0-docs-examples-and-release/07-01-PLAN.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-01-SUMMARY.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-02-PLAN.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-02-SUMMARY.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-03-PLAN.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-03-SUMMARY.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-04-PLAN.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-04-SUMMARY.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-05-PLAN.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-05-SUMMARY.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-REVIEW-DISPOSITION.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-REVIEW.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-UAT.md", "CHANGELOG.md", "Makefile", "README.md", "cmd/sup-kv/broadcast_probe_test.go", "cmd/sup-kv/examples_test.go", "cmd/sup-kv/main.go", "cmd/sup-kv/operations_test.go", "cmd/sup-kv/readme_test.go", "cmd/sup-kv/scenario_test.go", "docs/EXAMPLES.md", "docs/OPERATIONS.md", "docs/setup/branch-protection.md"]
covered_digest: "v4:sha256:dcf5fba5886d214ca318db41f4a9ecf4cb6b537297cdbf5ed80dafede24429b2"
behavior_unverified: 0
overrides_applied: 1
overrides:
  - must_have: "SC1 macOS clause: the README quickstart works exactly as written on macOS"
    reason: "Maintainer accepted option-b (2026-11-17, 07-UAT test 1, commit cbe4974): the quickstart ships a CI-verified Linux block plus a marked darwin variant block; the macos-15 leg is a deterministic named skip carrying the recorded probe verdict (limited broadcast sendto 'no route to host' on macos-15, PR #32 run 44083351667; lo0 carries no BROADCAST flag). DOCS-01 recorded complete-with-recorded-partial in REQUIREMENTS.md; WINDOWS ids 5/6 remain tracked open entries by the same decision."
    accepted_by: "maintainer (Daniel Podolsky)"
    accepted_at: "2026-11-17T20:01:35Z"
re_verification:
  previous_status: human_needed
  previous_score: 9/10
  gaps_closed:
    - "SC1 macOS clause adjudicated: maintainer accepted option-b (07-UAT test 1, passed) — recorded as override above"
    - "OPERATIONS.md content-truth UAT passed: maintainer read-through 2026-11-18, all five statements truthful (07-UAT test 2)"
    - "Stranger cold-read passed: maintainer confirmation plus agent-assisted verbatim cold run, fresh clone to /tmp/supkv-cold, README quickstart executed literally, exit 0 (07-UAT test 3)"
    - "Shipped-archive README staleness dispositioned: maintainer accepted fixed-forward, v1.0.0 stands as shipped (07-UAT test 4)"
  gaps_remaining: []
  regressions: []
---

# Phase 7: v1.0 — Docs, Examples and Release Verification Report

**Phase Goal:** A stranger can deploy and operate a two-node sup-kv cluster from the documentation alone, and v1.0 ships
**Verified:** 2026-11-18T19:12:00Z
**Status:** passed
**Re-verification:** Yes — final pass after human UAT completion (prior report: human_needed, 9/10)

This pass re-ran first-hand evidence at HEAD (343d4c1), read the completed UAT record, and corroborated the cold-run outcome against the filesystem rather than accepting the UAT prose. No implementation file changed between the prior verification (2026-11-17T19:35:00Z) and HEAD — `git log --since` shows only `.planning/` commits — so the prior pass's first-hand green evidence (full-package runs, tidy gate, binary download) applies unchanged to HEAD's tree, on top of this pass's fresh runs below.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1 (Linux leg): README quickstart fenced blocks execute VERBATIM against the built binary — two nodes, write visible on both, kill -9, restart, convergence, cleanup | ✓ VERIFIED | Fresh run at HEAD: `go test -race ./cmd/sup-kv/ -run '^TestReadmeQuickstart$' -v` → PASS (4.54s); README carries 2 `quickstart-verify` markers; `kill -9 $NODE_A` present; README.md is the literal test input |
| 2 | SC1 (macOS clause): quickstart works exactly as written on macOS | ✓ PASSED (override) | Maintainer accepted option-b (07-UAT test 1, passed 2026-11-17; commit cbe4974 flipped DOCS-01 complete-with-recorded-partial): darwin variant block present + deterministic named skip (`readmeSkipDarwin` readme_test.go:55) carrying the recorded macos-15 probe verdict; WINDOWS ids 5/6 remain tracked open. Override recorded in frontmatter |
| 3 | SC2 (DOCS-02): OPERATIONS.md states the trust model prominently FIRST, the value-size envelope, local-read semantics, not-a-lock warning, clock-drift runbook (+ monitoring/lifecycle) | ✓ VERIFIED | Fresh run at HEAD: `TestOperationsDoc` → PASS (0.00s); first `##` is `## Trust model` (line 8); constants parity re-confirmed: 61440 = main.go:135 `defaultValueMaxBytes` ↔ OPERATIONS.md:54, drift-warn 60s = main.go:146 ↔ OPERATIONS.md:92. Content TRUTH now human-verified: maintainer read-through 2026-11-18, all five statements truthful (07-UAT test 2, passed) |
| 4 | SC3 (DOCS-03): every RPC has a copy-paste grpcurl example that works against the shipped binary | ✓ VERIFIED | Fresh run at HEAD: `TestExamplesDoc` → PASS (8.17s); 10 `examples-verify` markers, 10 titled blocks (pair start, discovery, NotFound, cross-node write/read, acked replication wait, delete+list, Watch, health, snapshot round-trip, graceful stop) |
| 5 | Goal clause "v1.0 ships": annotated tag on a green master commit; GitHub Release with 4 archives + 4 SBOMs + checksums.txt, CHANGELOG header | ✓ VERIFIED | Re-checked live this pass: `git ls-remote` tag ea57bfd → 0ead1e8 (local tag identical); `gh release view v1.0.0`: published 2026-11-17T17:41:59Z, exactly 9 assets (checksums.txt + 4 archives + 4 SBOMs); all gate checks green on 0ead1e8 per prior first-hand check-run query — tag immutable since |
| 6 | Shipped binary identity: `sup-kv v1.0.0 0ead1e8 go1.27.4` (v-prefixed {{ .Tag }} parity) | ✓ VERIFIED | Prior pass downloaded the Release archive, `sha256sum -c` → OK, ran `--version` first-hand; tag and Release assets unchanged since (re-verified this pass), so the artifact verified then is the artifact shipped now |
| 7 | 07-01: reflection live (grpcurl zero-proto list) + pinned tools + clean tidy gate, zero runtime require changes | ✓ VERIFIED | `reflection.Register(gs)` in listenGRPC (main.go); TestExamplesDoc output assertion covers `supkvapi.v1.SupKVService` (passed this pass); go.mod tool directives grpcurl v1.9.4 + goreleaser v2.20.2; go.mod/go.sum unchanged since the prior pass's clean `go mod tidy && git diff --exit-code` |
| 8 | 07-01: macos-15 probe verdict recorded BEFORE 07-02 froze the README; option encoded in the gate | ✓ VERIFIED | 07-01-SUMMARY `## Broadcast probe verdict` (verbatim sendto evidence, option-b decision); `quickstartBroadcastOption = "option-b"` at readme_test.go:49; skip text names PR run 44083351667 job 134039317468 — re-read this pass |
| 9 | 07-04: KaC 2.0.0 CHANGELOG + goreleaser 4-target/SBOM/checksums pipeline + ci.yml byte-untouched + Makefile targets | ✓ VERIFIED | CHANGELOG.md 43 lines (Unreleased/1.0.0/Added/tag-link/keepachangelog 2.0.0); .goreleaser.yaml 46 lines incl. the IN-01 fix comment (4c69fa0, in HEAD ancestry); release.yml 40 lines (tags v* + dispatch, syft@v0.24.3); `git log --since=2026-11-16 -- .github/workflows/ci.yml` → EMPTY (re-checked this pass); Makefile 87 lines with release/release-snapshot; .gitignore `/dist/` |
| 10 | 07-05: D-11 ruleset re-apply attempted verbatim, read-side verified, marker matches live state either way | ✓ VERIFIED | docs/setup/branch-protection.md marker "still plan-gated, re-verified 2026-11-17"; prior pass reproduced the read-side 403; file unchanged since; REQUIREMENTS.md CI-03 remains honestly unchecked `[ ]` (prohibition honored) |

**Score:** 10/10 truths verified (0 present-but-unverified; 1 passed via maintainer-accepted override)

Truths 1 and 4 are behavior-dependent (kill/restart/convergence; cross-node replication) and were re-proven by fresh single-test runs this pass — not symbol presence. Truth 2 is the maintainer-adjudicated option-b partial, carried as a documented override.

### Deferred Items

None — all milestone phases complete; nothing deferred to later phases.

### Advisory (New Scope, Unevidenced)

Re-verification ran; no unevidenced new-scope findings arose (all Step 7 observations below carry directly observable evidence).

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| — | None | — | — |

### Required Artifacts

Quick regression per re-verification policy (existence + substance + marker sanity; full three-level verification was done in the prior pass; no implementation file changed since):

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| cmd/sup-kv/examples_test.go | harness + TestExamplesDoc | ✓ VERIFIED | 383 lines; test passed this pass (8.17s) |
| cmd/sup-kv/broadcast_probe_test.go | TestProbeQuickstartBroadcast + probeSendBroadcast | ✓ VERIFIED | 188 lines; test passed this pass ("127.255.255.255 delivers to a local socket") |
| cmd/sup-kv/readme_test.go | TestReadmeQuickstart consuming 07-01 helpers | ✓ VERIFIED | 114 lines; test passed this pass (4.54s) |
| cmd/sup-kv/operations_test.go | TestOperationsDoc presence guard | ✓ VERIFIED | 103 lines; test passed this pass |
| cmd/sup-kv/scenario_test.go | shared scenario harness (07-01 files_modified) | ✓ VERIFIED | 684 lines; exercised by the suite |
| docs/EXAMPLES.md | every-RPC walkthrough, marked blocks | ✓ VERIFIED | 358 lines; 10 markers; zero install commands inside any marked block (grep of extracted fenced blocks) |
| docs/OPERATIONS.md | five mandated statements, trust first | ✓ VERIFIED | 156 lines; `## Trust model` is the first `##` (line 8); content-truth human-verified (UAT test 2) |
| README.md | quickstart-verify marker, kill -9, doc links, no "Design phase" | ✓ VERIFIED | 149 lines; 2 markers; WR-01 fix (f3d5b86) in HEAD ancestry |
| CHANGELOG.md | KaC 2.0.0, curated 1.0.0 | ✓ VERIFIED | 43 lines; structure intact |
| .goreleaser.yaml | v2, 4 targets, checksums, SBOMs, CHANGELOG header | ✓ VERIFIED | 46 lines; IN-01 fix comment present (4c69fa0) |
| .github/workflows/release.yml | tag-triggered, pinned, contents:write | ✓ VERIFIED | 40 lines; run 44138907991 success; 9 assets live on the Release |
| git tag v1.0.0 (annotated) | on green master squash | ✓ VERIFIED | ea57bfd → 0ead1e8, local + remote identical (re-checked live) |
| GitHub Release v1.0.0 | 9+ assets | ✓ VERIFIED | Exactly 9 assets, published, not draft (re-checked live) |
| docs/setup/branch-protection.md | honest ruleset marker | ✓ VERIFIED | 96 lines; marker matches live state |

### Key Link Verification

All links re-checked at wiring level (markers/constants present, tests green this pass; deep verification in prior pass, files unchanged since):

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| examples_test.go | cmd/sup-kv/main.go | reflection.Register → grpcurl list reaches the server | WIRED | TestExamplesDoc asserts `supkvapi.v1.SupKVService` — passed this pass |
| examples_test.go | docs/EXAMPLES.md | extractMarkedBlocks — doc is the test input | WIRED | 10 blocks extracted and executed — passed this pass |
| go.mod | examples_test.go | grpcurl shim exec'ing `go -C <modroot> tool grpcurl` | WIRED | Shim replaces PATH entirely (examples_test.go:171 comment + prior pass's line-level check); marked blocks contain no install commands |
| readme_test.go | README.md | extractMarkedBlocks('../../README.md') | WIRED | Test passed this pass; zero-blocks pre-flight fatal |
| readme_test.go | broadcast_probe_test.go | darwin leg gated by named skip carrying the probe verdict | WIRED (mechanism evolved) | Deliberate 64a3cbb determinism decision; skip names probeSendBroadcast + PR run + verdict — the maintainer-accepted option-b contract |
| release.yml | .goreleaser.yaml | `go tool goreleaser release --clean` on tag push | WIRED | 9 live assets on the Release prove the pipe end-to-end |
| .goreleaser.yaml | Makefile | ldflags {{ .Tag }}/{{ .ShortCommit }} parity | WIRED | Shipped binary identity verified first-hand (prior pass); tag unchanged |
| git tag v1.0.0 | release.yml | push origin v1.0.0 triggers the job | WIRED | Run event=push headBranch=v1.0.0 success (prior first-hand); tag unchanged |
| docs/setup/ruleset.json | rulesets API | POST + read-side verify | EXECUTED (403) | Org-plan gate; marker matches live state; CI-03 left honestly unchecked |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| TestReadmeQuickstart | captured block output | README.md blocks → bash -e → real binary | Yes — `c3VwLWt2` asserted from a real Get (passed this pass) | FLOWING |
| TestExamplesDoc | per-block captured output | EXAMPLES.md blocks → real two-node pair | Yes — acked/wait/NotFound/SERVING assertions from real runs (passed this pass) | FLOWING |
| Cold-run node dumps | supkv-a.json / supkv-b.json | Fresh clone /tmp/supkv-cold → verbatim README quickstart | Yes — both dumps hold the identical replicated record (`demo/hello` → base64 `c3VwLWt2`, same record_version `01a1ef5a…` and checksum) — first-hand convergence proof in human-readable JSON | FLOWING |
| Release v1.0.0 | release header | CHANGELOG.md via mustReadFile | Yes — workflow succeeded; CHANGELOG in the tagged tree | FLOWING |
| --version identity | version/commit vars | goreleaser ldflags -X at build time | Yes — downloaded binary printed the stamped identity (prior first-hand) | FLOWING |

### Behavioral Spot-Checks

All four doc-contract tests re-run first-hand at HEAD (343d4c1), single named tests:

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| DOCS-02 presence guard | `go test -race -run '^TestOperationsDoc$' -v` | PASS (0.00s) | ✓ PASS |
| Quickstart derive line + broadcast forms | `go test -race -run '^TestProbeQuickstartBroadcast$' -v` | PASS — "127.255.255.255 delivers to a local socket on this platform" | ✓ PASS |
| README quickstart verbatim incl. kill -9/restart/convergence | `go test -race -run '^TestReadmeQuickstart$' -v` | PASS (4.54s) | ✓ PASS |
| Every-RPC walkthrough against a real pair | `go test -race -run '^TestExamplesDoc$' -v` | PASS (8.17s) | ✓ PASS |
| Cold-run residue (goal evidence) | `ls /tmp/supkv-cold` + dump diff + `pgrep sup-kv` | Clone of gsd/v1.0-milestone @ edfb52b (implementation-identical to HEAD), built binary present, both dumps carry the same replicated record, zero leftover processes | ✓ PASS |
| Regression posture | `git log --since=2026-11-17T19:35:00Z --name-only` | Only `.planning/` files changed since the prior green verification; full-package green runs (prior pass + 07-REVIEW-DISPOSITION) therefore apply to HEAD's unchanged implementation tree | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
|-------|---------|--------|--------|
| scripts/*/tests/probe-*.sh | n/a | No probe-script convention in this repo; the phase's probe is TestProbeQuickstartBroadcast (executed above, PASS) | N/A |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|------------------|----------|--------|
| DOCS-01 | 07-01, 07-02 | README quickstart works as written (two nodes, kill -9, restart, convergence) | ✓ SATISFIED | Linux leg machine-proven (truth 1, fresh run); macOS leg = maintainer-accepted option-b override (UAT test 1, commit cbe4974); REQUIREMENTS.md checkbox now `[x]`. Note: the traceability table row (REQUIREMENTS.md:186) still reads "Pending" — stale vs. the checkbox, see Anti-Patterns |
| DOCS-02 | 07-03 | Ops docs: trust model prominent, envelope, local-read semantics, not-a-lock warning, drift runbook | ✓ SATISFIED | Presence machine-proven (truth 3, fresh run) + content-truth human-verified (UAT test 2: maintainer read-through 2026-11-18, all five statements truthful) |
| DOCS-03 | 07-01, 07-03, 07-04, 07-05 | API usage examples provided, work against shipped binary | ✓ SATISFIED | Truths 4-6: TestExamplesDoc green (fresh run); same-source release binary proven (prior first-hand download + identity) |

Orphaned requirements: none — all three Phase-7 IDs in REQUIREMENTS.md traceability appear in plan frontmatter.

### Decision Coverage

All trackable 07-CONTEXT.md decisions honored by shipped artifacts: 11/11 (gate non-blocking; `check.decision-coverage-verify` exit 0).

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| readme_test.go | DOCS-01 | Linux leg (PASS this pass) | darwin named skip (option-b, adjudicated) | none | Value/behavioral (kill -9/restart/convergence asserted) | OK |
| examples_test.go | DOCS-03 | PASS this pass | darwin named skip (option-b) | none — outputs produced by real runs, asserted from captured output | Value/behavioral | OK |
| operations_test.go | DOCS-02 | PASS this pass | none | none | Presence/anchor (by design; content truth human-verified via UAT) | OK |
| broadcast_probe_test.go | DOCS-01 | PASS this pass | darwin loud skip (recorded verdict) | none | Behavioral (socket delivery) | OK |

**Disabled tests on requirements:** 0 — the darwin skips are the maintainer-adjudicated option-b contract (named, deterministic, carrying the probe verdict), not silent disables; the Linux legs of the same tests run and pass. main_test.go/scenario_test.go skips are environment/capability guards (build availability, loopback-broadcast capability), not requirement proofs being avoided. **Circular patterns:** 0 — the doc tests execute the real binary and assert real captured output (07-03 prohibition: no hand-written expected outputs). **Insufficient assertions:** 0.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| .planning/REQUIREMENTS.md | 186 | Traceability table row `DOCS-01 \| Phase 7 \| Pending` is stale — the checkbox at line 89 was flipped to `[x]` in cbe4974 but the table row was not updated (DOCS-02/03 rows WERE updated in c7c6312, so the table is maintained) | ⚠️ Warning | Internal inconsistency in the tracking doc only; the canonical checkbox reflects the maintainer decision. Fix: one-line row edit to `Complete` (maintainer/orchestrator action — not made by the verifier per workflow rules) |
| .planning/WINDOWS.md | ids 5-7 | Ledger hygiene at ship time: id 5/6 (option-b deviation) intentionally kept open by maintainer decision; id 7 (DOCS-02 content-truth unmet-truth) is now satisfied by UAT test 2 and is a candidate for `gsd-tools windows fixed 7`; id 8 remains the dispositioned post-v1.0 Info | ℹ️ Info | `/gsd-ship` (windows_enforce) will surface open entries; dispositions already recorded — no action lost |
| v1.0.0 archive README (frozen in tag/Release) | quickstart blocks | Pre-WR-01 quickstart embedded in the frozen archives — maintainer consciously accepted fixed-forward (UAT test 4, passed 2026-11-17); repo README self-heals on branch merge; closes with v1.0.1 | ℹ️ Info | Accepted disposition; no action outstanding for this phase |
| docs/EXAMPLES.md | 20 | Prerequisites prose offers strangers an unpinned grpcurl (`go install ...@latest`) alongside the named in-repo pinned directive; zero install commands exist inside any CI-executed marked block (deterministic evidence: extracted-block grep empty; shim replaces PATH) | ℹ️ Info | The 07-01 prohibition's enforcement clause (CI resolves only via pinned directives/shims) holds with deterministic evidence; the prose mention is an outside-repo convenience the maintainer's cold run bypassed by using the pinned directive. Surfaced for transparency |
| repo (current tree) | — | TBD/FIXME/XXX/TODO/HACK/placeholder scan across all phase files | none | Clean |
| IN-02..IN-05, WINDOWS 8 | — | Review findings dispositioned post-v1.0 (07-REVIEW-DISPOSITION.md) | ℹ️ Info | Tracked in the ledger; none blocks the goal |

### Human Verification — Discharged

All four items from the prior report's `human_verification` frontmatter are discharged by 07-UAT.md (status: complete, 4/4 passed, 0 issues, commit 343d4c1). Each was checked against artifacts, not prose:

| Prior item | UAT outcome | Corroborating evidence verified this pass |
|------------|-------------|--------------------------------------------|
| 1. DOCS-01 macOS-leg adjudication | passed — maintainer accepted option-b (2026-11-17); DOCS-01 complete with recorded partial; WINDOWS 5/6 tracked open | Commit cbe4974 flipped REQUIREMENTS.md DOCS-01 to `[x]`; named skips in readme_test.go:55 / examples_test.go:262 carry the full verdict text; override recorded in this report's frontmatter |
| 2. OPERATIONS.md content-truth | passed — maintainer read-through 2026-11-18: all five statements truthful and clear | Constants parity re-confirmed against main.go (61440, 60s); TestOperationsDoc green; `## Trust model` first |
| 3. Stranger cold-read (the phase goal itself) | passed — maintainer confirmation plus agent-assisted verbatim cold run 2026-11-18: fresh clone to /tmp/supkv-cold, README quickstart executed literally, exit 0, no leftover processes | /tmp/supkv-cold exists (cloned Nov 18 18:48, branch gsd/v1.0-milestone @ edfb52b — implementation-identical to HEAD); built binary present; **both node dumps carry the identical replicated record** (`demo/hello`, base64 `c3VwLWt2`, same record_version + checksum) — first-hand proof the write on A converged to B and the dumps are human-readable JSON; `pgrep` finds no leftover processes |
| 4. Shipped-archive README staleness | passed — maintainer accepted fixed-forward (2026-11-17); v1.0.0 stands as shipped; repo README self-heals on merge; closes with v1.0.1 | Release re-checked live (published, 9 assets, tag unchanged); corrected `./sup-kv` form confirmed in HEAD's README via WR-01 commit f3d5b86 (in ancestry, tests green) |

No new human-verification items arose: nothing in this phase remains visual, real-time, or external-service dependent beyond the completed UAT, and the one behavior-dependent truth without an executed macOS run is closed by the maintainer's recorded option-b acceptance (override), not left open.

### Gaps Summary

None. All artifacts exist, are substantive, and are wired; the four doc-contract tests were re-run first-hand at HEAD and pass under `-race`; the shipped release, its nine assets, and the tag were re-verified against live GitHub state; the implementation tree is byte-identical to the tree the prior pass verified green (only `.planning/` commits since); debt markers are absent; decision coverage is 11/11; and the four prior human items are discharged by a completed, committed UAT whose load-bearing claim — the stranger cold run — was corroborated first-hand from the clone's own dump files.

The single deviation from the plan's literal wording — the macOS quickstart leg shipping as a named skip rather than a CI-verified run — is the maintainer's recorded option-b decision, carried transparently as an override (WINDOWS ids 5/6 stay tracked). Two bookkeeping notes are recorded above (stale DOCS-01 traceability row; WINDOWS id 7 now closable) — neither affects goal achievement.

**The phase goal is achieved:** a stranger followed the documentation alone from a fresh clone to a working, converging, kill-9-surviving two-node cluster (UAT test 3, corroborated by the clone's dumps), and v1.0.0 is shipped and verified.

---

_Verified: 2026-11-18T19:12:00Z_
_Verifier: Claude (gsd-verifier)_
