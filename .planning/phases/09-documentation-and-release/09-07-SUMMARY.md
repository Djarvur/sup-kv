---
phase: 09-documentation-and-release
plan: 07
subsystem: documentation-and-deploy
tags: [deployment-guide, systemd-unit, ci-smoke, doc-anchor-tests, executable-docs, m16, docs-03, tdd]

requires:
  - phase: 09-documentation-and-release (09-01…09-06)
    provides: the as-built docs/CONFIGURATION.md the unit consistency check anchors to, the OPERATIONS.md reference the guide links (runbooks, metrics reference, exit codes), the FORMATS.md facts the upgrade section cites, and the green cmd/sup-kv doc-anchor harness this plan extends
  - phase: 07-tooling (Phase 7)
    provides: the exit-code contract (0/1/2/3) the restart policy encodes and the executable-doc harness (extractMarkedBlocks, runBashVerbatim, supkvBinDir, grpcurlShim)
provides:
  - docs/DEPLOYMENT.md — the two-board topology end to end (bind/broadcast relation, wildcard-bind WARN, per-node unicast cross-addressing alternative, the three listeners, node identity and the duplicate_node_id configuration error, paths/permissions/generations, sizes with the must-match pointer, time sync, the 14.3 dump setup and 14.8 metrics setup sections, upgrading/compatibility, node replacement, the systemd unit section with the exit-code/restart table) in the mandated reading order, with exactly two deployment-verify marked blocks executed verbatim in make test
  - deploy/sup-kv.service — the working M-16 unit: Restart=on-failure + RestartPreventExitStatus=2 (usage errors never loop), StateDirectory, dedicated-user shape, documented-flag-only ExecStart with every non-default value stated in the guide
  - cmd/sup-kv/deployment_test.go — TestDeploymentDocBlocks (bidirectional title-keyed block contract + mandated section order + option-b darwin skip) and TestDeploymentUnitConsistency (flag table / env knob / restart policy / required-knobs / non-default-value-stated checks)
  - .github/workflows/ci.yml — systemd-analyze verify + the ExecStart-args smoke inside the existing gate job's Linux cells, check-run contexts and matrix byte-identical
affects: [09-08 (its proven-claims sweep must keep DEPLOYMENT.md text green), 09-10 (the goreleaser files: half ships this unit and this guide — the doc states the contract), 09-11 (milestone verification)]

actuals:
  tokens: 31000
  tasks: 4
  commits: 4

tech-stack:
  added: []
  patterns:
    - "unit↔docs consistency by parsed-table anchoring: ExecStart flags are compared against docs/CONFIGURATION.md table rows (the configuration_test.go parse, never a --help capture) and non-default values must be stated in the guide via a boundary-exact `--flag value` regex, so --node-id 11 can never satisfy a check for --node-id 1"
    - "CI joins an existing job under a runner-guarded step condition (if: !matrix.cross && runner.os == 'Linux') — the five check-run contexts and the matrix stay byte-identical, verified by the job-key-count-vs-HEAD and additive-diff guards instead of a YAML parse"

key-files:
  created:
    - docs/DEPLOYMENT.md
    - deploy/sup-kv.service
    - cmd/sup-kv/deployment_test.go
    - .planning/phases/09-documentation-and-release/red-evidence/09-07-task1-red.json
  modified:
    - .github/workflows/ci.yml

key-decisions:
  - "The binary wins over the plan's 16-hex node-id gloss: --node-id is a 16-bit integer (0..65535) parsed as a base-0 uint64 — the plan's example literal 0123456789abcdef is an invalid octal and exits 2 — so the unit carries --node-id 1 (the README-demo idiom) and the CI smoke runs the unit's own value; the guide documents the integer domain and the uniqueness contract"
  - "The unit's metrics listener stays on loopback (127.0.0.1:7902), not the plan's 0.0.0.0 gloss: OPERATIONS' monitoring guidance and CONFIGURATION's own metrics-addr row say plaintext stays on loopback, and the SUPKV_METRICS_* mTLS triple is the documented beyond-loopback path (the guide's Metrics setup section explains it)"
  - "ExecStart is one line, not backslash continuations: the plan's verify greps (ExecStart=.*--node-id / .*--dump-path) are single-line, and a one-line ExecStart keeps them honest; the consistency test's unitExecStart joins continuations either way, so the parser is not weakened"
  - "The CI smoke's override delta is documented in the step comment, not hidden: the four addresses go to loopback doc-band ports and the dump path to /tmp (CI must never emit LAN broadcast), and --startup-gate-empty 2s is pinned because the smoke node's disk is empty — every other flag and value is the unit's, verbatim"
  - "TestDeploymentDocBlocks is serialized with the keepSurvivors walkthrough shape (the pair-start block's survivors are the metrics block's state) and carries the option-b darwin loud skip mirroring examplesSkipDarwin — without it the macos-15 native gate cell would fail on loopback broadcast's sendto"
  - "The mandated section order is asserted as a subsequence walk over the doc's ## headings, so extra sections are free but the deployer's reading order (topology before the unit) is part of the contract"

patterns-established:
  - "runner-guarded CI steps: OS-specific steps gate on `!matrix.cross && runner.os == 'Linux'` so the macos-15 native cell keeps running the full Go gate while systemd-specific work stays Linux-only — no matrix edit, no ruleset change"

requirements-completed: []
# DOCS-01/DOCS-03/DOCS-10 stay open on the shared-ID gate: DOCS-03's archive-shipping
# half is 09-10's (the doc states the contract), DOCS-10's 14.3/14.8 halves landed here
# but the CONTRIBUTING/SECURITY/ARCHITECTURE halves belong to siblings, and DOCS-01's
# M-closure row M-16 closes with this plan while the other M rows have their own owners.
# M-16 is closed by this plan.

coverage:
  - id: deployment-doc-blocks
    description: "The two marked blocks run verbatim (two-board demo pair with a replicated write; health SERVING + metrics scrape + SIGTERM stop) with bidirectional assertion-table coverage and the mandated section order"
    requirement: DOCS-03
    verification:
      - kind: tests
        ref: cmd/sup-kv/deployment_test.go#TestDeploymentDocBlocks
        status: pass
      - kind: tests
        ref: ".planning/phases/09-documentation-and-release/red-evidence/09-07-task1-red.json (RED evidence, classifier RED_EVIDENCE_OK)"
        status: pass
    human_judgment: false
  - id: unit-consistency
    description: "deploy/sup-kv.service restarts failures (1/3) but never a usage error (2), names only CONFIGURATION.md flag-table rows and documented env knobs, carries the four required knobs, and states every non-default value in the guide"
    requirement: DOCS-03
    verification:
      - kind: tests
        ref: cmd/sup-kv/deployment_test.go#TestDeploymentUnitConsistency
        status: pass
    human_judgment: false
  - id: unit-in-ci
    description: "systemd-analyze verify passes with the binary installed at the ExecStart path; the ExecStart-args smoke (the unit's argument list with the documented loopback overrides) reaches SERVING and exits 0 on SIGTERM, locally in the step's exact shape; the CI guards prove no job/matrix/check-run change"
    requirement: DOCS-03
    verification:
      - kind: command
        ref: "systemd-analyze verify deploy/sup-kv.service (exit 0, systemd 255); the local smoke: SERVING then wait-exit 0; the job-key-count-vs-HEAD and additive-diff guards"
        status: pass
      - kind: tests
        ref: ".github/workflows/ci.yml gate job Linux cells (first run happens on GitHub — the local smoke replicates the step verbatim)"
        status: pass
    human_judgment: true
    rationale: "The CI steps themselves execute on GitHub runners only; the honest evidence is the local replication of the step's exact command shape plus systemd-analyze verify in the same systemd version family (255). The first pushed PR is the definitive CI proof."
  - id: m16-closure
    description: "M-16 (no systemd unit anywhere) is closed: the working unit exists, is machine-checked, CI-verified and documented; the archive-shipping statement is in the guide (the goreleaser half is 09-10's)"
    requirement: DOCS-01
    verification:
      - kind: tests
        ref: cmd/sup-kv/deployment_test.go#TestDeploymentUnitConsistency
        status: pass
      - kind: command
        ref: "grep guards: Restart=on-failure, RestartPreventExitStatus=2, ExecStart flags present"
        status: pass
    human_judgment: false
  - id: setup-sections-143-148
    description: "The 14.3 dump setup and 14.8 metrics setup sections are first-class sections that link (never duplicate) OPERATIONS/PERFORMANCE/CONFIGURATION"
    requirement: DOCS-10
    verification:
      - kind: command
        ref: "grep gates: '## Dump setup', '## Metrics setup' present; the metrics section names the three SUPKV_METRICS_* knobs and points at the OPERATIONS metrics reference without a family table"
        status: pass
    human_judgment: true
    rationale: "That the sections answer owner decisions 14.3/14.8's documentation half (defaults, cost profile, knob semantics, scrape guidance) is a manual read against the review's decision list; the greps pin only presence."

# Metrics
duration: 35min
completed: 2026-11-28
status: complete
---

# Phase 9 Plan 7: The Deployment Story Summary

The deployment story landed: docs/DEPLOYMENT.md documents the two-board topology end to end (with the 14.3/14.8 dump and metrics setup sections and the explicit single-node shape), deploy/sup-kv.service is the working M-16 unit with the exit-code-aware restart policy, both are machine-checked by two new tests in the cmd/sup-kv doc-anchor package, and the CI gate job's Linux cells now verify the unit and smoke its ExecStart argument list — check-run contexts and matrix byte-identical.

## Performance

- **Duration:** 35 min
- **Started:** 2026-11-28T02:52Z
- **Completed:** 2026-11-28T03:27Z
- **Tasks:** 4
- **Files modified:** 5 (4 created, 1 modified)

## Accomplishments

- **Task 1 (RED, TDD).** `cmd/sup-kv/deployment_test.go` in package main_test, reusing the executable-doc harness unchanged: `TestDeploymentDocBlocks` extracts the `deployment-verify` marked blocks of ../../docs/DEPLOYMENT.md (zero blocks or an unclosed fence fatal), asserts the mandated section order as a subsequence walk, runs the blocks verbatim in one shared directory on the reserved doc port band [27900, 28100) against a bidirectional title-keyed assertion table, keeps the pair-start block's survivors for the later block with a PID-file cleanup backstop, and carries the option-b darwin loud skip; `TestDeploymentUnitConsistency` parses the unit's ExecStart (continuation-aware), requires Restart=on-failure + RestartPreventExitStatus=2, the four required knobs, an absolute ExecStart path, ≥6 flags (vacuity floor), every flag a docs/CONFIGURATION.md table row (the configuration_test.go parse, never --help), every SUPKV_ assignment a documented env knob, and every non-default ExecStart value stated in DEPLOYMENT.md via a boundary-exact `--flag value` regex. RED recorded (red-evidence/09-07-task1-red.json, classifier RED_EVIDENCE_OK): both tests failed naming the missing artifacts (DEPLOYMENT.md, sup-kv.service); vet and lint clean.
- **Task 2.** `deploy/sup-kv.service`: [Unit] Description/Documentation + After/Wants=network-online.target; [Service] Type=simple, User=sup-kv, StateDirectory=sup-kv, one-line ExecStart=/usr/local/bin/sup-kv with the documented argument set (--node-id 1 as the must-be-unique per-board example, wildcard replication bind 0.0.0.0:7900, /24 segment broadcast example 192.168.1.255:7900, StateDirectory dump path, LAN gRPC listener 0.0.0.0:7901, loopback plaintext metrics 127.0.0.1:7902), Restart=on-failure + RestartPreventExitStatus=2 + RestartSec=2, and commented optional Environment= lines for SUPKV_LOG_LEVEL and the all-or-nothing SUPKV_METRICS_* triple; [Install] WantedBy=multi-user.target. Verified: built binary installed at /usr/local/bin/sup-kv via sudo, `systemd-analyze verify deploy/sup-kv.service` exit 0 (systemd 255), plus the plan's grep guards.
- **Task 3 (GREEN).** `docs/DEPLOYMENT.md` (447 lines) in the mandated reading order — Topology (two Linux boards on one L2 segment; the two-board demo; N-board generalization; the explicit minimal single-node shape; the marked demo block), Addresses and ports (bind-listens / broadcast-sends relation, the non-wildcard WARN, the per-node unicast cross-addressing alternative for broadcast-hostile networks, the three-listener table), Node identity (16-bit integer, uniqueness, the duplicate_node_id configuration error and runbook pointer), Paths and permissions (generations main/.prev/.corrupt-*/.tmp~, the dedicated sup-kv user, the StateDirectory mapping, dump-directory sizing), Sizes (61440 / 65 507 / sync chunk / must-match pointer), Time sync (NTP, drift-warn/max-clock-lead, runbook pointer), Dump setup (14.3: debounce/max-wait cadence, the measured 14.36× disk-amplification cost profile, the .prev chain, the fast-profile tuning pointer), Metrics setup (14.8: --metrics-addr off-by-default, the PEM mTLS triple table, scrape guidance pointing at the OPERATIONS metrics reference), Upgrading and compatibility (proto_version gate, dump format v1 + record_count tolerance, rolling order), Node replacement (same node-id + dump copy, what neighbours see), systemd unit (install steps, the ExecStart explained flag by flag, the exit-code/restart table, the journal as the log source, the archive-shipping statement) — with exactly two marked blocks (demo pair; health/metrics check + stop). Both deployment tests green under -race; the whole cmd/sup-kv package green.
- **Task 4.** `.github/workflows/ci.yml`: two steps join the existing gate job's Linux cells only (`if: ${{ !matrix.cross && runner.os == 'Linux' }}` — no new job, no matrix entry, no job-level name/runs-on): `systemd unit verify` (build, sudo install to the ExecStart path, systemd-analyze verify) and `ExecStart-args smoke` (the unit's argument list with the documented loopback overrides, --startup-gate-empty 2s pinned, SERVING awaited via the pinned `go tool grpcurl`, SIGTERM, `wait` + exit-0 assert). The plan's job-key-count-vs-HEAD and additive-diff guards pass byte-for-byte.

## Task Commits

Each task was committed atomically:

1. **Task 1: RED — the deployment doc-block test and the unit↔docs consistency check** - `78304bb` (test)
2. **Task 2: the systemd unit with the exit-code-aware restart policy** - `f8c02b9` (feat)
3. **Task 3: the deployment guide with executable proof (GREEN)** - `ee9a80f` (docs)
4. **Task 4: the CI verify + smoke steps in the existing gate job** - `44b7ad1` (ci)

_Note: Task 1 is the plan's RED commit; Task 3 is its GREEN (the tests turn green on the doc+unit pair, which Tasks 2 and 3 carry together)._

## Files Created/Modified

- `cmd/sup-kv/deployment_test.go` - the two deployment machine checks (honest-scope header; harness reuse unchanged)
- `deploy/sup-kv.service` - the M-16 unit
- `docs/DEPLOYMENT.md` - the deployment guide with two marked executable blocks
- `.github/workflows/ci.yml` - the Linux-cells-only verify + smoke steps inside the gate job
- `.planning/phases/09-documentation-and-release/red-evidence/09-07-task1-red.json` - the classifier-validated RED record

## Decisions Made

- **The binary wins over the plan's node-id gloss (Rule 1).** The plan's Task 2 asked for "a 16-hex example value" and Task 4's verify used `--node-id 0123456789abcdef`; the as-built parser is a base-0 uint64 over the 16-bit domain (CONFIGURATION.md's own row), so that literal is an invalid octal and exits 2. The unit carries `--node-id 1` (the README-demo idiom) and the smoke runs the unit's value; the guide documents the integer domain and uniqueness. The consistency check anchors to CONFIGURATION.md, so the doc-anchored property is unaffected.
- **Loopback metrics in the unit (Rule 1, ops guidance).** The plan glossed `--metrics-addr 0.0.0.0:<metrics port>`; OPERATIONS' monitoring guidance and CONFIGURATION's metrics-addr row state plaintext stays on loopback, so the unit ships `--metrics-addr 127.0.0.1:7902` and the guide's Metrics setup section explains the mTLS triple as the beyond-loopback path.
- **One-line ExecStart (Rule 3, mechanical).** The plan's verify greps match `ExecStart=.*--node-id` on one line; backslash continuations fail them. The unit uses the single-line form; the consistency test still joins continuations (unitExecStart), so nothing else depends on the shape.
- **Smoke deltas documented, not hidden.** The CI step comment states the override delta (four addresses to loopback doc-band ports, dump path to /tmp — CI never emits LAN broadcast) and the `--startup-gate-empty 2s` pin (fresh empty-disk node; the unit's production node loads a dump and opens on the 1s startup gate). Every other flag and value is the unit's, verbatim.
- **Serialized walkthrough with survivors.** The demo-pair block keeps its two nodes alive for the health/metrics block (keepSurvivors + PID files), mirroring the EXAMPLES walkthrough; the darwin leg loud-skips per the recorded option-b verdict, keeping the macos-15 native cell green.
- **Honest scope stated in the test header.** The tests prove doc/unit internal consistency and that the marked blocks run on this machine — not that the reader's boards behave like CI's loopback demo.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Plan fact] --node-id is a 16-bit integer, not a 16-hex value**
- **Found during:** Task 2 (unit authoring) and Task 4 (smoke)
- **Issue:** the plan's Task 2 action ("a 16-hex example value") and Task 4's verify literal (`0123456789abcdef`) contradict the as-built parser (base-0 uint64, 16-bit domain; the literal is an invalid octal → exit 2) and docs/CONFIGURATION.md's own row
- **Fix:** unit `--node-id 1`; guide documents the integer domain; smoke uses the unit's value
- **Files modified:** deploy/sup-kv.service, docs/DEPLOYMENT.md, .github/workflows/ci.yml
- **Verification:** the local smoke reached SERVING and exited 0 with the corrected value; the literal form is exercised nowhere
- **Committed in:** f8c02b9, ee9a80f, 44b7ad1

**2. [Rule 1 - Plan fact] The unit's metrics listener is loopback, not 0.0.0.0**
- **Found during:** Task 2 (unit authoring)
- **Issue:** the plan's ExecStart gloss binds metrics to all interfaces plaintext, contradicting OPERATIONS' "do not expose the metrics listener beyond loopback as plaintext" and CONFIGURATION's "keep it on loopback"
- **Fix:** `--metrics-addr 127.0.0.1:7902` in the unit; the non-default value stated in the guide's unit section (the consistency check enforces the statement) and the mTLS triple documented for remote scraping
- **Files modified:** deploy/sup-kv.service, docs/DEPLOYMENT.md
- **Verification:** TestDeploymentUnitConsistency green; the guide carries the exact literal
- **Committed in:** f8c02b9, ee9a80f

**3. [Rule 3 - Mechanical] Single-line ExecStart instead of continuations**
- **Found during:** Task 2 verify
- **Issue:** the plan's grep guards (`ExecStart=.*--node-id`, `ExecStart=.*--dump-path`) are single-line; the continuation form failed them while being otherwise equivalent
- **Fix:** the ExecStart is one line; the consistency parser joins continuations either way
- **Files modified:** deploy/sup-kv.service
- **Verification:** both grep guards pass; systemd-analyze verify unchanged (exit 0)
- **Committed in:** f8c02b9

---

**Total deviations:** 3 auto-fixed (2 × Rule 1 plan-fact corrections, 1 × Rule 3 mechanical). **Impact on plan:** none on scope — all three make the plan's own guards executable against the as-built binary and the project's own operator guidance.

## Issues Encountered

- The guide's intro originally spelled the marker as the literal `<!-- deployment-verify -->`, which the reused extractor treats as a marker occurrence (it scans every line containing the marker) and left a titleless block; the intro was reworded to `deployment-verify` without the comment syntax. Resolved inside Task 3 before the verify ran.
- The first local smoke run backgrounded the whole `systemd-analyze && install && sup-kv && …` chain, so the kill/wait observed the subshell (exit 143), not the node; the CI step itself backgrounds the single sup-kv command (verified correct), and the clean re-run reached SERVING and exited 0. No CI change was needed.

## Verification

- Task 1 verify: `go test -race -count=1 ./cmd/sup-kv/ -run 'TestDeploymentDocBlocks|TestDeploymentUnitConsistency' -v` — both target tests FAIL naming the missing artifacts; the log grep gate passes; `go vet ./cmd/sup-kv/` clean — PASS.
- Task 2 verify: `go build -o /tmp/sup-kv-09-07-bin ./cmd/sup-kv/ && sudo install -m 0755 /tmp/sup-kv-09-07-bin /usr/local/bin/sup-kv && systemd-analyze verify deploy/sup-kv.service` — exit 0 — PASS; the Restart/ExecStart grep guards — PASS.
- Task 3 verify: the same test filter — PASS (both green); the section greps (Topology/Metrics setup/Dump setup/Node replacement/RestartPreventExitStatus=2/single-node) — PASS; the whole `go test -race -count=1 ./cmd/sup-kv/` package — PASS.
- Task 4 verify: the ci.yml grep + job-key-count-vs-HEAD + additive-diff guards — PASS; `systemd-analyze verify` + the local ExecStart-args smoke in the step's exact shape (SERVING, SIGTERM, exit 0) — PASS.
- Full gates: `make lint` (config verify + golangci-lint run ./..., one modernize fix on the new test landed before its commit) and `make test` (the full -race suite, exit 0) — PASS.

## TDD Gate Compliance

- RED: `test(09-07)` commit 78304bb precedes GREEN; both target tests failed on their planned missing-artifact assertions; classifier verdict RED_EVIDENCE_OK.
- GREEN: `docs(09-07)` commit ee9a80f makes both pass under -race (with Task 2's `feat(09-07)` unit as the pair's other half).
- REFACTOR: none needed (one lint modernize fix landed inside the RED commit before its evidence was captured).

## Self-Check: PASSED

- All tasks executed and committed individually (78304bb, f8c02b9, ee9a80f, 44b7ad1).
- All acceptance criteria re-run after the final edit: both deployment tests under -race, the section greps, systemd-analyze verify, the local ExecStart-args smoke, the CI guards, the whole cmd/sup-kv package, make lint and make test — all green.
- Success criteria: the two-board topology is documented end to end with executable proof; the unit is working, verified and restart-aware (M-16 closed); the 14.3/14.8 setup sections land; the archive-shipping contract is stated (the goreleaser half belongs to 09-10); no check-run name, matrix entry or job-level key changed.

Next: Ready for 09-08.
