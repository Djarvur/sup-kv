---
phase: 261108-vxx-migratsiya-proto-codegen-na-buf-lint-bre
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - go.mod
  - go.sum
  - buf.yaml
  - buf.gen.yaml
  - Makefile
  - internal/udprepl/replproto/supkv.pb.go
  - .github/workflows/ci.yml
  - .claude/CLAUDE.md
autonomous: true
requirements: [TOOLING-BUF-01]
estimate:
  tokens: 45000
  raw_tokens: 30000
  tasks: 2
  confidence: low

must_haves:
  truths:
    - "go tool buf lint exits 0 at repo root: STANDARD (buf v1.75's name for the DEFAULT rule set, rule lists probe-verified identical) plus one file-scoped PACKAGE_DIRECTORY_MATCH exception with an inline rationale, and the local .gomodcache excluded via modules.excludes (buf walks dot-dirs; the cache holds 138 stray .proto files)"
    - "go tool buf format --diff --exit-code exits 0 on the committed supkv.proto (probe-verified already format-clean)"
    - "make buf-breaking and the CI buf step pass today via the loud defer guard (supkv.proto absent on origin/master; buf v1.75 errors on a proto-less against-ref instead of passing) and run the real comparison the moment the proto lands on master; a deleted Envelope field is caught under FILE policy (probe-verified exit 100)"
    - "make proto regenerates internal/udprepl/replproto/supkv.pb.go through go tool buf generate invoking the go-tool protoc-gen-go plugin with zero system compiler dependency; the git diff vs the old protoc output is exactly the two header version lines (D-07 accepted-difference case); go build ./... and go test -race ./internal/udprepl/... stay green"
    - "go.mod pins github.com/bufbuild/buf/cmd/buf v1.75.0 as a tool directive; google.golang.org/protobuf MVS-bumps to v1.36.13-0.20261016172727-e9ee2a76ab8e (plugin and runtime remain one identical module; go tool protoc-gen-go --version reports v1.36.12-devel); go mod tidy is idempotent so the CI tidy gate stays green"
    - "The CI lint job runs buf lint + buf format --diff --exit-code + guarded buf breaking with fetch-depth: 0 on its checkout; the job set, check-run names and the 4-cell gate matrix are unchanged; make lint and make test are unchanged and green"
  artifacts:
    - buf.yaml
    - buf.gen.yaml
    - Makefile
    - go.mod
    - go.sum
    - internal/udprepl/replproto/supkv.pb.go
    - .github/workflows/ci.yml
    - .claude/CLAUDE.md
  key_links:
    - "buf.yaml single root module (path .) covers internal/udprepl/replproto now and internal/grpcapi/apiproto in phase 5 with zero config change (D-06)"
    - "buf.gen.yaml local plugin [go, tool, protoc-gen-go] resolves through the go.mod tool directive — no global PATH install, no BSR remote plugins (D-05)"
    - "Makefile buf-breaking against-ref is byte-identical to the ci.yml against-ref: .git#ref=refs/remotes/origin/master (D-08)"
    - "CI buf step invokes the same go tool buf binary as the local Makefile targets (local/CI parity, same pinning discipline as golangci-lint)"
---

<objective>
Migrate proto codegen from system protoc to buf (buf generate), and turn on buf lint (STANDARD/DEFAULT rules), buf breaking (FILE policy against origin/master) and buf format — locally via Makefile targets and in the CI lint job (D-01, D-02, D-03, D-04, D-05, D-06, D-07, D-08, D-09).

Decision namespace: D-01..D-09 below are the maintainer's locked quick-task decisions of 2026-11-08 (numbered 1..9 in the brief). They are distinct from any phase-4 CONTEXT.md D-NN cited inside proto comments.

Purpose: kill the system protoc dependency, pin the whole protobuf toolchain through go.mod tool directives (buf v1.75.0 + protoc-gen-go), and put wire-schema lint/breaking/format gates around the replication protocol before phase 5 adds the gRPC schema.

Output: buf.yaml + buf.gen.yaml at repo root, buf as a pinned go.mod tool, a buf-based Makefile proto target plus buf-lint/buf-format/buf-breaking targets, a regenerated supkv.pb.go canonical-from-buf, buf gates wired into .github/workflows/ci.yml, and corrected tool rows in .claude/CLAUDE.md.

Planner probe evidence (2026-11-08, buf v1.75.0 in scratch dirs — repo untouched):
- latest stable buf = v1.75.0 via go list -m -versions; module path github.com/bufbuild/buf/cmd/buf works as a tool directive.
- STANDARD and DEFAULT lint categories produce identical rule lists; DEFAULT additionally prints a deprecation WARN on every invocation.
- Exactly one DEFAULT/STANDARD finding on supkv.proto: PACKAGE_DIRECTORY_MATCH (package supkv.v1 vs dir internal/udprepl/replproto).
- buf format: the committed proto is already clean (exit 0, empty diff).
- buf walks dot-directories: a root module picks up the repo-local .gomodcache (138 stray .proto files, gitignored). v2 has NO ignore field — the working shape is modules.excludes; verified lint/format/breaking all green with it.
- buf breaking vs .git#ref=refs/remotes/origin/master: ref resolves; against a proto-less ref buf ERRORS (exit 1, "had no .proto files") rather than passing — D-08's "trivially clean pass" assumption is false, hence the guard. Against a proto-bearing ref: exit 0; deleting Envelope.crc32 = exit 100 (FILE policy catches it).
- go get -tool buf@v1.75.0 + tidy MVS-bumps google.golang.org/protobuf v1.36.12 -> v1.36.13-0.20261016172727-e9ee2a76ab8e; go tool protoc-gen-go --version reports v1.36.12-devel; tidy re-run is a no-op.
- Regeneration via buf with the local go-tool plugin is byte-identical to the committed supkv.pb.go except exactly two header lines: protoc-gen-go v1.36.12-devel and protoc (unknown) replacing v1.36.12 and v3.21.12.
</objective>

<execution_context>
@/home/nil/.zcode/gsd-core/workflows/execute-plan.md
@/home/nil/.zcode/gsd-core/templates/summary.md
</execution_context>

<context>
@AGENTS.md
@Makefile
@internal/udprepl/replproto/supkv.proto
@internal/udprepl/replproto/supkv.pb.go
@.github/workflows/ci.yml
@go.mod
@.claude/CLAUDE.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: Pin buf v1.75.0 as a go.mod tool, land buf.yaml + buf.gen.yaml, port the Makefile to buf, regenerate supkv.pb.go (D-01, D-02, D-04, D-05, D-06, D-07, D-09)</name>
  <precondition>Working tree clean at HEAD on branch gsd/phase-4-replication-and-deterministic-simulator, with network access to proxy.golang.org (first buf tool build downloads ~100 modules).</precondition>
  <reversibility rating="reversible">The whole migration is one git revert; dropping the buf tool directive returns google.golang.org/protobuf to v1.36.12 on the next tidy.</reversibility>
  <files>go.mod, go.sum, buf.yaml, buf.gen.yaml, Makefile, internal/udprepl/replproto/supkv.pb.go, .claude/CLAUDE.md</files>
  <action>
Run from repo root. Do NOT create or switch branches; commits extend the current phase-4 branch (PR #31).

1. Pin buf (D-05): run go get -tool github.com/bufbuild/buf/cmd/buf@v1.75.0 — exact version, never a latest tag. Then go mod tidy twice; the second run must not change go.mod/go.sum (the CI gate job runs tidy then git diff --exit-code, per AGENTS.md dependency policy — indirect requires dragged by tools are the accepted D-03 reading). Expected, probe-verified consequences you must NOT fight: the tool block gains the buf line; the direct require google.golang.org/protobuf bumps from v1.36.12 to v1.36.13-0.20261016172727-e9ee2a76ab8e (MVS via buf's graph — do not add a replace or downgrade; gomoddirectives lint forbids replaces and the downgrade is impossible without downgrading buf); go.sum grows by buf's graph. Assert go tool buf --version prints 1.75.0 and go tool protoc-gen-go --version prints protoc-gen-go v1.36.12-devel (the plugin and the runtime are the same module, so they stay version-identical — this replaces .claude/CLAUDE.md's "keep plugin and runtime versions identical" note rather than violating it).

2. Create buf.yaml (v2) at repo root — exact shape (D-06): version v2; a modules list with one entry path "." carrying excludes [".gomodcache"] — buf walks dot-dirs by design and the repo-local module cache holds 138 stray .proto files (gitignored, CI unaffected; the excludes entry exists only for local runs); lint section with use ["STANDARD"] — STANDARD is buf v1.75's rename of the DEFAULT category, rule lists verified identical via buf config ls-lint-rules, and it avoids the deprecation WARN the DEFAULT spelling now prints on every run (implements D-02's DEFAULT rule set); an ignore_only map under lint with key PACKAGE_DIRECTORY_MATCH mapping to the single path internal/udprepl/replproto/supkv.proto (paths relative to the buf.yaml location); breaking section with use ["FILE"] (D-03). Every non-default choice carries an inline comment: the one-module-covers-both-proto-dirs rationale (D-06: phase 5 adds internal/grpcapi/apiproto with zero config change), the STANDARD-is-DEFAULT note, the moving-the-proto churns-phase-4-imports-for-no-gain rationale for the PACKAGE_DIRECTORY_MATCH exception, and the local-module-cache rationale for excludes.

3. Create buf.gen.yaml (v2) at repo root (D-05): version v2; plugins list with one entry whose local is the three-element list ["go", "tool", "protoc-gen-go"] (buf execs the go.mod-pinned plugin — no global PATH install, no BSR remote plugins per the .claude/CLAUDE.md tool policy), out "." and opt "paths=source_relative". No managed mode.

4. Makefile (D-01, D-04, D-09): replace the proto target recipe with the single line go tool buf generate and rewrite its comment block — regeneration now runs buf (go.mod-pinned) with the go-tool protoc-gen-go plugin at source-relative paths; no system compiler needed; dev machines only, CI never regenerates (supkv.pb.go stays committed). Add three new targets: buf-lint running go tool buf lint; buf-format running go tool buf format -w (the write form is local-only — CI uses --diff --exit-code per D-04); buf-breaking running git fetch origin master and then the guarded comparison — an if/else testing git cat-file -e refs/remotes/origin/master:internal/udprepl/replproto/supkv.proto, running go tool buf breaking --against '.git#ref=refs/remotes/origin/master' when the object exists, else echoing a one-line deferral: buf breaking deferred, supkv.proto not on origin/master yet, activates when the proto lands (the guard exists because buf v1.75 errors exit 1 on a proto-less against-ref instead of passing — probe-verified; the explicit remote-tracking refspec pins the fetched ref, D-08). Extend .PHONY with the three new target names. Leave lint, test, generate and build targets byte-unchanged (D-09).

5. Regenerate: run make proto, then inspect git diff -U0 internal/udprepl/replproto/supkv.pb.go. The expected and acceptable diff (D-07's anticipated-difference case, probe-verified) is exactly four diff lines: the protoc-gen-go version header line and the compiler version header line replaced by protoc-gen-go v1.36.12-devel and protoc (unknown). If ANY other hunk appears, stop and investigate before committing (check go tool protoc-gen-go --version and the protobuf require line) — do not commit unexplained generated-code changes. Run make proto a second time; the file must not change further.

6. Correct the two stale tool rows in .claude/CLAUDE.md (scoped Edit, nothing else in the file): the Buf CLI row's release-tag-binary clause becomes pinned-as-go.mod-tool-directive v1.75.0 wording (invoked as go tool buf locally and in CI), and the google.golang.org/protobuf row's version becomes v1.36.13-0.20261016172727-e9ee2a76ab8e with the graph-bumped-by-buf note (plugin and runtime remain one identical module; protoc-gen-go reports v1.36.12-devel).

Commit message: feat(tools): migrate proto codegen to buf — pinned tool, lint/format/breaking config, Makefile targets.
  </action>
  <verify>
    <automated>go tool buf lint && go tool buf format --diff --exit-code && test "$(go tool protoc-gen-go --version)" = "protoc-gen-go v1.36.12-devel" && grep -q 'github.com/bufbuild/buf/cmd/buf' go.mod && grep -q 'google.golang.org/protobuf v1.36.13-0.20261016172727-e9ee2a76ab8e' go.mod && go build ./... && go test -race ./internal/udprepl/... && test "$(git diff -U0 internal/udprepl/replproto/supkv.pb.go | grep -cE '^[+-]//')" -eq 4 && h=$(sha256sum go.mod go.sum) && go mod tidy && printf '%s\n' "$h" | sha256sum -c - && ! grep -q 'pinned by release tag' .claude/CLAUDE.md</automated>
    <fails_when>buf lint exits 100 listing STANDARD findings — a typo in the ignore_only path re-flags PACKAGE_DIRECTORY_MATCH, and a missing .gomodcache excludes entry floods lint with the 138 cache protos; buf format exits 100 printing a reformat diff; the version assert fails if the tool graph resolved differently than the pinned probe state; build/test failures; the diff-count gate fails on any regenerated hunk beyond the two header version lines (0 = regeneration skipped, more than 4 lines = plugin drift); sha256sum -c prints FAILED when tidy rewrites go.mod/go.sum, which would break the CI tidy gate; the .claude/CLAUDE.md negative grep fails while the stale release-tag clause is still present (baseline at HEAD: 1 occurrence).</fails_when>
  </verify>
  <done>buf is a pinned go.mod tool at v1.75.0; buf.yaml (one root module with excludes, STANDARD lint + one documented file-scoped exception, FILE breaking) and buf.gen.yaml (local go-tool plugin, source-relative) exist; make proto regenerates through buf with the only diff being the two header version lines; make buf-lint, make buf-format and make buf-breaking exist and behave as specified; go build ./... and go test -race ./internal/udprepl/... green; tidy idempotent; .claude/CLAUDE.md tool rows truthful; lint/test/generate/build Makefile targets untouched.</done>
</task>

<task type="auto">
  <name>Task 2: Wire buf lint + format + breaking into the CI lint job with the origin/master guard and run the full parity gate (D-03, D-08)</name>
  <precondition>Task 1 committed; origin remote reachable for git fetch origin master.</precondition>
  <files>.github/workflows/ci.yml</files>
  <action>
Edit .github/workflows/ci.yml lint job only (Task 1 artifacts are inputs, do not re-touch them):

1. The lint job's actions/checkout@v7 step gains with: fetch-depth: 0 — the breaking against-input resolves refs/remotes/origin/master from the local git dir, which a shallow checkout lacks. The gate job's checkout stays shallow; it never runs buf.

2. Insert one new step in the lint job after the golangci-lint config-verify run step and before the golangci/golangci-lint-action step (cheap checks fail fast), named to state it runs the buf schema gates. Its run block executes, in order: go tool buf lint; go tool buf format --diff --exit-code; then the same if/else guard as the Makefile buf-breaking target — test git cat-file -e refs/remotes/origin/master:internal/udprepl/replproto/supkv.proto, run go tool buf breaking --against '.git#ref=refs/remotes/origin/master' when present, else echo the one-line deferral. The step carries a short comment: the guard is required because buf v1.75 exits 1 on a proto-less against-ref (probe-verified) while origin/master does not yet carry the proto — it self-destructs the moment PR #31 merges and every later PR gets the full FILE-policy comparison; the .git-dir against-input avoids any private-repo token plumbing the remote-https form would need (D-08, which permits exactly this .git#branch/form fallback — the explicit remote-tracking refspec is used for determinism).

3. Do NOT add, rename or remove jobs, matrix cells or check-run names — the workflow header records that the check-run contexts are recorded in the branch-protection ruleset (STATE.md decision). No new permissions; contents: read already covers reading the local checkout.

4. Prove the exact CI step block locally (D-08's prove-locally requirement): git fetch origin master, then run the three commands verbatim from repo root — expect lint exit 0, format exit 0, and the breaking branch printing the deferral line today (proto absent on origin/master). This is the "clean pass now" D-08 asked for, delivered via the guard because buf's real behavior on an empty against-ref is an error, not a pass.

5. Full parity gate per AGENTS.md: make lint && make test green before pushing; CI runs the same targets.

Commit message: ci(tools): gate proto schema with buf lint + format + breaking vs origin/master.
  </action>
  <verify>
    <automated>git fetch origin master && go tool buf lint && go tool buf format --diff --exit-code && if git cat-file -e refs/remotes/origin/master:internal/udprepl/replproto/supkv.proto 2>/dev/null; then go tool buf breaking --against '.git#ref=refs/remotes/origin/master'; else echo 'buf breaking deferred: supkv.proto not on origin/master yet (D-08 guard)'; fi && test "$(grep -v '^\s*#' .github/workflows/ci.yml | grep -c 'go tool buf')" -eq 3 && test "$(grep -v '^\s*#' .github/workflows/ci.yml | grep -c 'fetch-depth: 0')" -eq 1 && test "$(grep -v '^\s*#' Makefile | grep -c '^[[:space:]]*protoc ')" -eq 0 && make lint && make test</automated>
    <fails_when>buf breaking exits 100 naming the wire break (for example a deleted or renumbered Envelope field) once the proto exists on origin/master; buf format exits 100 printing the reformat diff; a wrong grep count flags a missing fetch-depth: 0 (the CI breaking step would then fail ref resolution with a shallow checkout), a missing or duplicated buf command in the lint job (baseline at HEAD: zero go tool buf and zero fetch-depth occurrences; after this task exactly 3 and 1), or a leftover system-protoc recipe line in the Makefile (baseline at HEAD: 1; after Task 1: 0); make lint or make test surfaces any Go-level regression the regeneration or module bump caused.</fails_when>
  </verify>
  <done>The CI lint job checks out full history and runs buf lint, buf format --diff --exit-code and the guarded buf breaking against .git#ref=refs/remotes/origin/master via the same pinned go tool buf binary as local; the identical command sequence passes locally against the freshly fetched origin/master (breaking defers loudly today); job set, check-run names and the gate matrix are untouched; make lint && make test green.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| builder → proxy.golang.org | go get downloads buf v1.75.0 plus its ~100-module graph into go.mod/go.sum; the tool code executes on dev machines and CI |
| CI lint job → local git history | buf breaking consumes refs/remotes/origin/master from the checked-out .git as its comparison baseline |
| developer → committed generated artifact | make proto rewrites internal/udprepl/replproto/supkv.pb.go, a reviewed committed file |

## STRIDE Threat Register

ASVS level 1; block_on high. Threat IDs are quick-task scoped (QT-NN); QT-SC is the reserved supply-chain row.

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| QT-SC | Tampering | go.mod/go.sum via the buf tool graph (proxy.golang.org) | high | mitigate | Exact version pin github.com/bufbuild/buf/cmd/buf@v1.75.0 in every command — never a latest tag; go.sum checksums cover the whole graph; the existing weekly govulncheck cron already triggers on go.mod/go.sum path changes and scans the tool graph; no replace directives (gomoddirectives lint forbids them), so no silent source substitution is possible. |
| QT-01 | Tampering | ci.yml buf breaking guard | medium | mitigate | The guard is object-path-precise (refs/remotes/origin/master:internal/udprepl/replproto/supkv.proto), echoes its deferral loudly in the job log, and self-destructs the moment the proto lands on master; detection was proven by the planner's negative probe (deleting Envelope.crc32 field 4 → exit 100 under FILE policy), so the activated gate has demonstrated teeth. |
| QT-02 | Tampering | make buf-format write target | low | accept | Local convenience only; CI enforces the canonical format with --diff --exit-code, and any local rewrite is git-reviewed before it lands; buf format is deterministic and protobuf-official-styled. |
| QT-03 | Information Disclosure | buf breaking against-input | low | mitigate | Chose the .git-dir input form: no remote fetch, no credential plumbing in the workflow — the remote-https form D-08 mentioned as primary would have required routing the private repo's token into buf's git client on CI; permissions stay contents: read. |
</threat_model>

<verification>
Overall task checks, in order:

1. Task 1 automated chain: buf lint and format clean at repo root; plugin version assert; tool-directive and bumped-protobuf asserts in go.mod; build + race tests of the consuming package; the regenerated-file diff gate (exactly the two header version lines); tidy idempotence via sha256 re-check; the stale-doc negative grep.
2. Task 2 automated chain: the exact CI step block replayed locally against freshly fetched origin/master (breaking takes the loud defer branch today); grep-count gates on ci.yml (3 buf invocations, 1 fetch-depth: 0) and Makefile (0 system-protoc recipe lines) against baselines recorded at HEAD; full make lint && make test parity gate.
3. Decision coverage: D-01 (buf generate replaces the protoc invocation), D-02 (DEFAULT rule set as STANDARD + one documented exception), D-03 (FILE breaking vs origin/master, CI), D-04 (format --diff --exit-code in CI, write target locally), D-05 (buf pinned tool directive, local go-tool plugin), D-06 (root buf.yaml, one module, exception over file moves), D-07 (byte-identical-except-header regeneration, build/tests green), D-08 (one against-ref form, proven locally, guard documented), D-09 (Makefile targets, lint/test unchanged) — every decision maps to a task action above.
4. Two deliberate, documented implementations of locked-decision wording, both evidence-backed, neither scope-reducing: STANDARD implements D-02/D-06's "use: [DEFAULT]" (rule lists probe-verified identical; the DEFAULT spelling now deprecation-warns on every run), and the CI guard implements D-08's "passes trivially until PR #31 merges" (buf v1.75 errors exit 1 on a proto-less against-ref — the guard delivers the clean pass now and the protection after merge).
</verification>

<success_criteria>
- All must_haves.truths hold verbatim on the final tree; all must_haves.artifacts exist.
- go tool buf lint, go tool buf format --diff --exit-code exit 0 at repo root; make buf-breaking and the CI step exit 0 today and protect after the proto lands on master.
- supkv.pb.go regenerated by buf with exactly the two header version lines changed; go build ./... and go test -race ./internal/udprepl/... green; make lint && make test green.
- go.mod carries the buf tool pin; tidy idempotent; no replaces; job set and check-run names in ci.yml unchanged.
- D-01 through D-09 each implemented and cited; no deferred ideas introduced.
</success_criteria>

<output>
Create .planning/quick/261108-vxx-migratsiya-proto-codegen-na-buf-lint-bre/261108-vxx-SUMMARY.md when done. Commits land on the current branch gsd/phase-4-replication-and-deterministic-simulator (extending PR #31); never create or switch branches.
</output>
