---
phase: 07-v1-0-docs-examples-and-release
verified: 2026-11-17T19:35:00Z
status: human_needed
score: 9/10 must-haves verified
covered_files: [".github/workflows/release.yml", ".gitignore", ".goreleaser.yaml", ".planning/phases/07-v1-0-docs-examples-and-release/07-01-PLAN.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-01-SUMMARY.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-02-PLAN.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-02-SUMMARY.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-03-PLAN.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-03-SUMMARY.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-04-PLAN.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-04-SUMMARY.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-05-PLAN.md", ".planning/phases/07-v1-0-docs-examples-and-release/07-05-SUMMARY.md", "CHANGELOG.md", "Makefile", "README.md", "cmd/sup-kv/broadcast_probe_test.go", "cmd/sup-kv/examples_test.go", "cmd/sup-kv/main.go", "cmd/sup-kv/operations_test.go", "cmd/sup-kv/readme_test.go", "cmd/sup-kv/scenario_test.go", "docs/EXAMPLES.md", "docs/OPERATIONS.md", "docs/setup/branch-protection.md"]
covered_digest: "v4:sha256:adfa93e27470203d288fcdbc89f8f57907728678e686598060c8f47d28a0dbd0"
behavior_unverified: 1 # Count of PRESENT_BEHAVIOR_UNVERIFIED truths (present + wired, behavior not exercised); each is detailed in behavior_unverified_items below (and in human_verification when status is human_needed)
overrides_applied: 0
behavior_unverified_items:
  - truth: "Roadmap SC1 macOS clause: the README quickstart works exactly as written on macOS"
    test: "On a real macOS host/segment, run the README's marked darwin variant block verbatim (BCAST=$(ipconfig getoption en0 broadcast ...) derive line)"
    expected: "Two nodes exchange the write, kill -9/restart/converge — or the block fails exactly as the recorded macos-15 verdict predicts (limited broadcast: sendto no route to host; loopback: lo0 has no BROADCAST flag)"
    why_human: "Darwin paths cannot execute on this linux verifier. The green macos-15 gate cell on tagged commit 0ead1e8 proves only that the deterministic loud skip fires (readmeSkipDarwin), not that the quickstart runs; the derived subnet-broadcast form was never probed — the probe failed at the limited-broadcast row before reaching the derived-row verdict (07-01-SUMMARY verbatim evidence)"
human_verification:
  - test: "Adjudicate DOCS-01 (WINDOWS ids 5/6): accept the option-b partial — CI-verified Linux quickstart + marked darwin variant block + deterministic named skip — or take option-c (reword REQUIREMENTS.md DOCS-01 and ROADMAP Phase 7 SC1 to 'Linux CI-verified; macOS documented with the probe-named limitation') and close DOCS-01 accordingly"
    expected: "DOCS-01 reaches an honest terminal state (complete-with-recorded-partial, or reworded and complete); REQUIREMENTS.md 'Pending' resolved; WINDOWS ids 5/6 closed. Note: option-a's premise was never empirically closed — the derived subnet-broadcast form remains unprobed on macos-15 — so option-c is not forced by evidence, only by wording preference"
    why_human: "Requirement-level scope decision explicitly reserved to the maintainer at the 07-01 Task 3 checkpoint and re-reserved to phase verification by the recorded option-b decision text"
  - test: "OPERATIONS.md content-truth UAT (WINDOWS id 7, 07-VALIDATION.md manual table): work the five mandated statements against a running two-node cluster — trust-model wording matches reality, 61440-byte envelope rejects with InvalidArgument, read-your-writes holds on the same node only, not-a-lock framing, clock-drift runbook symptom/action incl. supkv_clock_drift_warnings_total"
    expected: "Each statement TRUE of the shipped binary; no overstatement of safety (T-7-04)"
    why_human: "TestOperationsDoc is a presence/anchor guard only (41 anchors + firstness); no automated check ties prose claims to runtime behavior. Constants spot-checked by the verifier (61440 = main.go defaultValueMaxBytes; drift-warn 60s = defaultDriftWarn) strengthen but do not replace the UAT"
  - test: "Stranger cold-read test (the phase goal itself): from a clean checkout, follow README -> docs/EXAMPLES.md -> docs/OPERATIONS.md only (no source reading) to deploy and operate a two-node cluster: start pair, write on A read on B, kill -9/restart/converge, list/watch/health/snapshot"
    expected: "Deployable and operable from the documentation alone"
    why_human: "The goal's 'a stranger' is a human-judgment construct; TestReadmeQuickstart/TestExamplesDoc approximate it under CI-shaped conditions (PATH shim, temp cwd)"
  - test: "Disposition the shipped-artifact README staleness: v1.0.0 Release archives (and origin/master's rendered README) embed the pre-WR-01 quickstart (bare 'sup-kv' invocations after 'go build -o sup-kv' — a stranger hits command-not-found); the copy-paste-runnable './sup-kv' form is committed on gsd/v1.0-milestone (f3d5b86) but postdates the tag"
    expected: "Branch merges to master (repo README self-heals — normal /gsd-ship flow); explicit maintainer accept of the frozen archive copy, or a delete-and-re-tag decision per the documented recovery path"
    why_human: "Release-contract tradeoff (the tag is the download contract; re-tag is costly); the disposition ledger treats WR-01 as fixed-forward, but awareness that every v1.0.0 archive carries the pre-fix quickstart belongs to the maintainer"
---

# Phase 7: v1.0 — Docs, Examples and Release Verification Report

**Phase Goal:** A stranger can deploy and operate a two-node sup-kv cluster from the documentation alone, and v1.0 ships
**Verified:** 2026-11-17T19:35:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

Every claim below was verified first-hand against the working tree, the git history, and live GitHub state — not taken from SUMMARY.md.

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1 (Linux leg): README quickstart fenced blocks execute VERBATIM against the built binary — two nodes, write visible on both, kill -9, restart, convergence, cleanup | ✓ VERIFIED | `go test -race ./cmd/sup-kv/ -run '^TestReadmeQuickstart$' -v` → PASS (5.80s) on current tree; README carries 2 `<!-- quickstart-verify -->` markers (Linux block with 127.255.255.255 literals + darwin variant with BCAST derive line); `kill -9 $NODE_A` at README lines 72/117; README.md is the literal test input (readmeDocPath) |
| 2 | SC1 (macOS clause): quickstart works exactly as written on macOS | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Darwin variant block present + deterministic named skip wired (readmeSkipDarwin names the probe, PR run 44083351667 job 134039317468, the verbatim sendto error, AND what still proves the behavior); macos-15 gate cell green on tagged commit 0ead1e8 proves the skip fires — but the macOS quickstart itself never ran; derived form never probed. Maintainer option-b decision recorded (07-01-SUMMARY, STATE.md, WINDOWS 5/6); adjudication reserved to phase verification — see Human Verification |
| 3 | SC2 (DOCS-02): OPERATIONS.md is ONE file stating the trust model prominently FIRST, the value-size envelope, local-read semantics, not-a-lock warning, clock-drift runbook (+ monitoring/lifecycle) | ✓ VERIFIED | `go test -race -run '^TestOperationsDoc$'` → PASS; 41 anchors + machine-asserted firstness (first `##` must be `## Trust model`, operations_test.go:93); constants cross-checked against code: 61440 = main.go:135 defaultValueMaxBytes, drift-warn 60s = main.go:146. Content TRUTH remains manual UAT (human item 2, WINDOWS 7) |
| 4 | SC3 (DOCS-03): every RPC has a copy-paste grpcurl example that works against the shipped binary | ✓ VERIFIED | `go test -race -run '^TestExamplesDoc$' -v` → PASS (7.29s); 10 `examples-verify` markers, 10 titled blocks covering pair start, discovery (list/describe), missing-key Get NotFound, cross-node write/read, wait_for_replication (acked), delete+list, Watch, health, SnapshotCreate/SnapshotLoad round-trip, graceful stop |
| 5 | Goal clause "v1.0 ships": annotated tag on a green master commit; GitHub Release with 4 archives + 4 SBOMs + checksums.txt, CHANGELOG header | ✓ VERIFIED | `git ls-remote`: tag ea57bfd → 0ead1e8; check-runs on 0ead1e8: lint + 4 gate cells + release ALL success; `gh release view v1.0.0`: published (not draft), exactly 9 assets matching the claimed list; release workflow run 44138907991 (event push, headBranch v1.0.0): success |
| 6 | Shipped binary identity: `sup-kv v1.0.0 0ead1e8 go1.27.4` (v-prefixed {{ .Tag }} parity) | ✓ VERIFIED | Verifier downloaded sup-kv_1.0.0_Linux_x86_64.tar.gz + checksums.txt from the Release, `sha256sum -c` → OK, extracted, ran `--version` → `sup-kv v1.0.0 0ead1e8 go1.27.4` |
| 7 | 07-01: reflection live (grpcurl zero-proto list) + pinned tools + clean tidy gate, zero runtime require changes | ✓ VERIFIED | `reflection.Register(gs)` at main.go:591 inside listenGRPC; TestExamplesDoc output assertion covers `supkvapi.v1.SupKVService`; go.mod tool directives grpcurl v1.9.4 + goreleaser v2.20.2 (lines 7/9); verifier re-ran `go mod tidy && git diff --exit-code go.mod go.sum` → TIDY-GATE-CLEAN |
| 8 | 07-01: macos-15 probe verdict recorded BEFORE 07-02 froze the README; option encoded in the gate | ✓ VERIFIED | 07-01-SUMMARY `## Broadcast probe verdict` (verbatim sendto evidence, option-b maintainer decision); `quickstartBroadcastOption = "option-b"` at readme_test.go:49 with lockstep comment |
| 9 | 07-04: KaC 2.0.0 CHANGELOG + goreleaser 4-target/SBOM/checksums pipeline + ci.yml byte-untouched + Makefile targets | ✓ VERIFIED | CHANGELOG.md: Unreleased/1.0.0/Added/tag-link/keepachangelog.com/en/2.0.0 all present; .goreleaser.yaml v2 with all required elements (header via mustReadFile — the documented v2.20.2 schema deviation from the plan's from_file literal); release.yml: tags v* + dispatch, contents:write, syft@v0.24.3, timeout 20; `git log --since=2026-11-16 -- .github/workflows/ci.yml` EMPTY; Makefile release/release-snapshot + .PHONY; .gitignore `/dist/` |
| 10 | 07-05: D-11 ruleset re-apply attempted verbatim, read-side verified, marker matches live state either way | ✓ VERIFIED | Verifier re-ran read-side `gh api .../rulesets` → 403 "Upgrade to GitHub Pro or make this repository public..."; docs/setup/branch-protection.md:16 "still plan-gated, re-verified 2026-11-17" — marker and live state agree; STATE.md carries both decision entries (07-05 ruleset + release facts); CI-03 sub-item honestly left plan-gated |

**Score:** 9/10 truths verified (1 present, behavior-unverified)

Truths 1 and 4 are behavior-dependent (state transitions: kill/restart/convergence; cross-node replication) and carry passing behavioral tests — VERIFIED on behavioral evidence, not symbol presence.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| cmd/sup-kv/examples_test.go | harness + TestExamplesDoc + title-keyed table | ✓ VERIFIED | 17.0 KB; extractMarkedBlocks:62, runBashVerbatim:133, supkvBinDir:189, grpcurlShim:203, blockSurvivors (reap/keep), examplesSkipDarwin:260, TestExamplesDoc:310 |
| cmd/sup-kv/broadcast_probe_test.go | TestProbeQuickstartBroadcast + probeSendBroadcast | ✓ VERIFIED | 8.2 KB; probeSendBroadcast:45, probeBroadcastForm:83, Test:147, darwin loud-skip:150 |
| cmd/sup-kv/readme_test.go | TestReadmeQuickstart consuming 07-01 helpers | ✓ VERIFIED | 5.8 KB; marker pre-flight, option-b constant, readmeSkipDarwin:55, Test:75; reuses extractMarkedBlocks/runBashVerbatim (no second implementation) |
| cmd/sup-kv/operations_test.go | TestOperationsDoc presence guard | ✓ VERIFIED | 41 anchors + firstness assertion; honest-scope header |
| docs/EXAMPLES.md | every-RPC walkthrough, ≥7 marked blocks | ✓ VERIFIED | 10 markers, 10 titled sections |
| docs/OPERATIONS.md | five mandated statements, trust first | ✓ VERIFIED | All sections present; `## Trust model` is first `##` |
| README.md | quickstart-verify marker, kill -9, doc links, no "Design phase" | ✓ VERIFIED | 2 markers; links at lines 47/139/142; "Design phase" absent |
| CHANGELOG.md | KaC 2.0.0, curated 1.0.0 | ✓ VERIFIED | Structural grep suite all present; date 2026-11-17 = tag date |
| .goreleaser.yaml | v2, 4 targets, checksums, SBOMs, header from CHANGELOG | ✓ VERIFIED | All elements; header via `mustReadFile "CHANGELOG.md"` (documented schema deviation, IN-01 comment) |
| .github/workflows/release.yml | tag-triggered, pinned, contents:write | ✓ VERIFIED | Full element suite present; run 44138907991 succeeded |
| git tag v1.0.0 (annotated) | on green master squash | ✓ VERIFIED | ea57bfd → 0ead1e8, local + remote; all 5 gate checks + release green on that SHA |
| GitHub Release v1.0.0 | ≥9 assets | ✓ VERIFIED | Exactly 9: 4 archives + 4 SBOMs + checksums.txt; published 2026-11-17T17:41:59Z |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| examples_test.go | cmd/sup-kv/main.go | reflection.Register in listenGRPC → grpcurl list reaches the server | WIRED | main.go:591; TestExamplesDoc asserts the service name in captured output (passed) |
| examples_test.go | docs/EXAMPLES.md | extractMarkedBlocks('../../docs/EXAMPLES.md', marker) | WIRED | Marker const :43; 10 blocks extracted and executed (test green) |
| go.mod | examples_test.go | grpcurl shim exec'ing `go -C <modroot> tool grpcurl` | WIRED | grpcurlShim:203 writes exactly that /bin/sh script, chmod 0755; PATH replaced (not appended) — no unpinned binary can win |
| readme_test.go | README.md | extractMarkedBlocks('../../README.md', quickstart-verify) | WIRED | Test green; zero-blocks pre-flight fatal names the DOCS-01 contract |
| readme_test.go | broadcast_probe_test.go | probeSendBroadcast gates the darwin leg | WIRED (mechanism evolved) | The skip NAMES probeSendBroadcast + the recorded verdict rather than invoking it live — the deliberate 64a3cbb determinism decision recorded in 07-01/07-02 summaries; intent (never a silent skip) fully preserved |
| release.yml | .goreleaser.yaml | `go tool goreleaser release --clean` on tag push | WIRED | Run 44138907991 success on v1.0.0 push; 9 assets prove the whole pipe incl. SBOMs |
| .goreleaser.yaml | Makefile | ldflags {{ .Tag }}/{{ .ShortCommit }} identity parity | WIRED | Shipped binary prints v1.0.0 + 0ead1e8; snapshot binary printed v0.0.0 + 1db9265 — same shape |
| git tag v1.0.0 | release.yml | push origin v1.0.0 triggers the job | WIRED | Verified: run event=push, headBranch=v1.0.0, conclusion success |
| docs/setup/ruleset.json | rulesets API | gh api POST + read-side verify | EXECUTED (403) | Both attempts returned the org-plan gate; marker updated honestly; verifier reproduced the read-side 403 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| TestReadmeQuickstart | captured block output | README.md fenced blocks → bash -e → real binary + pinned grpcurl | Yes — asserted `c3VwLWt2` from a real Get | FLOWING |
| TestExamplesDoc | per-block captured output | EXAMPLES.md blocks → real two-node pair | Yes — acked/wait/watch/NotFound assertions from real runs | FLOWING |
| Release v1.0.0 | release header | CHANGELOG.md via mustReadFile | Yes — workflow succeeded; CHANGELOG present in tagged tree and in archives | FLOWING |
| --version identity | version/commit vars | goreleaser ldflags -X at build time | Yes — downloaded binary printed the stamped identity | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| DOCS-02 presence guard | `go test -race -run '^TestOperationsDoc$' -v` | PASS (0.00s) | ✓ PASS |
| Linux quickstart derive line + 3 broadcast forms | `go test -race -run '^TestProbeQuickstartBroadcast$' -v` | PASS — all rows "reached" | ✓ PASS |
| README quickstart verbatim incl. kill -9/restart/convergence | `go test -race -run '^TestReadmeQuickstart$' -v` | PASS (5.80s) | ✓ PASS |
| Every-RPC walkthrough against a real pair | `go test -race -run '^TestExamplesDoc$' -v` | PASS (7.29s) | ✓ PASS |
| Tidy gate idempotency | `go mod tidy && git diff --exit-code go.mod go.sum` | clean | ✓ PASS |
| Tagged-commit gate matrix | `gh api commits/0ead1e8.../check-runs` | lint + 4 gates + release: all success | ✓ PASS |
| Shipped artifact integrity + identity | download, `sha256sum -c`, `./sup-kv --version` | OK; `sup-kv v1.0.0 0ead1e8 go1.27.4` | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
|-------|---------|--------|--------|
| scripts/*/tests/probe-*.sh | n/a | no probe-script convention in this repo; the phase's probe is TestProbeQuickstartBroadcast (executed above) | N/A |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|---------------------|----------|
| DOCS-01 | 07-01, 07-02 | README quickstart works as written (two nodes, kill -9, restart, convergence) | ? NEEDS HUMAN | Linux leg machine-proven (truth 1); macOS leg is the recorded option-b named skip — REQUIREMENTS.md honestly "Pending"; adjudication = human item 1. NOT blocked: all deliverable evidence exists; only the requirement-level decision remains |
| DOCS-02 | 07-03 | Ops docs: trust model prominent, envelope, local-read semantics, not-a-lock warning, drift runbook | ✓ SATISFIED | Truth 3 (presence machine-proven + constants cross-checked); content-truth UAT listed as human item 2 per WINDOWS 7 |
| DOCS-03 | 07-01, 07-03, 07-04, 07-05 | API usage examples provided, work against shipped binary | ✓ SATISFIED | Truths 4-6 (TestExamplesDoc green; same-source release binary proven) |

Orphaned requirements: none — all three Phase-7 IDs in REQUIREMENTS.md traceability appear in plan frontmatter.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| v1.0.0 archive README (frozen in tag/Release) | quickstart blocks | Bare `sup-kv` invocations after `go build -o sup-kv` — not copy-paste runnable for a stranger (pre-WR-01) | ⚠️ Warning | The fix (`./sup-kv` form + stranger-mirroring harness) is committed on gsd/v1.0-milestone (f3d5b86, green, in this verification's passing run) but postdates the tag; origin/master still at 0ead1e8. Repo README self-heals on branch merge; the frozen archive copy does not — human item 4 |
| repo (current tree) | — | TBD/FIXME/XXX/TODO/HACK/placeholder scan across all 10 phase files | none | Clean |
| walkthrough scratch artifacts (.gitignore gap) | IN-02 | Review finding, open | ℹ️ Info | Maintainer-dispositioned post-v1.0 (07-REVIEW-DISPOSITION.md) |
| EXAMPLES stop-block poll window = shutdownGrace | IN-03 | Review finding, open | ℹ️ Info | Dispositioned post-v1.0 |
| extractMarkedBlocks title reset on #/## | IN-04 | Review finding, latent, open | ℹ️ Info | Dispositioned post-v1.0 |
| README duplicate Quickstart heading | IN-05 | Review finding, open | ℹ️ Info | Dispositioned post-v1.0 |
| cmd/sup-kv/examples_test.go | WINDOWS 8 | Hard runner cancel between blocks can leak the pair | ℹ️ Info | PID-file t.Cleanup backstops normal aborts; surfaced, not blocking |

### Human Verification Required

Four items — see frontmatter `human_verification` for the full test/expected/why of each:

1. **DOCS-01 macOS-leg adjudication** — accept option-b (complete-with-recorded-partial) or take option-c (reword DOCS-01/SC1 and close). This is the single blocking decision for closing the phase and DOCS-01's checkbox (WINDOWS ids 5/6). Option-a's premise was never empirically closed: the derived subnet-broadcast form remains unprobed on macos-15, so option-c is a wording choice, not an evidence-forced one.
2. **OPERATIONS.md content-truth UAT** — the five statements TRUE of the shipped binary (WINDOWS id 7; constants already cross-checked: 61440, 60s).
3. **Stranger cold-read test** — the phase goal itself, performed by a human from the docs alone.
4. **Shipped-artifact README staleness disposition** — v1.0.0 archives embed the pre-WR-01 quickstart; fix is branch-only until merge. Confirm merge (normal flow) and explicitly accept or act on the frozen archive copy.

### Gaps Summary

No blocking gaps. All artifacts exist, are substantive, and are wired; all machine-provable behaviors passed under `-race` on first-hand runs; the shipped release, its assets, its workflow run, and the binary identity were all verified against live GitHub state rather than SUMMARY claims; the tidy gate is clean; ci.yml is untouched; debt markers are absent; the ruleset marker matches live state exactly.

Two things honestly remain for a human:

1. **The macOS quickstart leg (SC1/DOCS-01).** Empirically closed on the macos-15 runner for the forms probed; the quickstart's macOS form was never run. The maintainer pre-decided option-b at the 07-01 checkpoint and explicitly reserved final adjudication to this verification (WINDOWS 5/6; REQUIREMENTS.md DOCS-01 still "Pending" — consistent). This is the reason the status is human_needed.
2. **The shipped v1.0.0 archives embed the pre-WR-01 README quickstart** (bare-name invocations) because the code review and its fix landed after the tag; the branch (verified green here) carries the corrected, copy-paste-runnable form. The repo README self-heals when the branch merges; the frozen archive copy does not. The maintainer should consciously accept this (fixed-forward) or re-tag.

The phase's engineering substance — doc-as-test contracts that make README and EXAMPLES.md unable to drift from reality, the drift-guarded ops doc, the pinned release pipeline, and a verified-shipped v1.0.0 — is real, present, and green.

---

_Verified: 2026-11-17T19:35:00Z_
_Verifier: Claude (gsd-verifier)_
