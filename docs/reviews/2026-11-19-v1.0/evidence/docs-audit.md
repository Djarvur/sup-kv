# sup-kv documentation audit

Audit date: 2026-11-19. Repository at `d6b2a75` (master, clean; tag `v1.0.0` = `0ead1e8`).
Reference spec: `the upstream spec` (Russian, ACCEPTED), read in full.

## 0. Method and counts

- Corpus read in full: `README.md`, `docs/OPERATIONS.md`, `docs/EXAMPLES.md`, `docs/IDEA.md`, `docs/setup/branch-protection.md`, `docs/setup/ruleset.json`, `CHANGELOG.md`, `AGENTS.md`, `.claude/CLAUDE.md`, `LICENSE`, both `.proto` files, every non-test Go file under `cmd/` and `internal/` (package docs and comments), `Makefile`, `.goreleaser.yaml`, `.github/*`, the three doc tests.
- Binary built outside the repo (`go build -o <scratch>/sup-kv ./cmd/sup-kv`, Go 1.27.4, darwin/arm64); `--help` and `--version` captured (Appendix A).
- Claims were checked by reading code and, where reading was not conclusive, by running the scratch binary as a single node (and once as a loopback pair) on private ports 391xx, and by two throw-away Go tests in a scratch **copy** of the repo. Nothing in the repo was modified; the three doc tests were not run. Experiments are listed in Appendix B.
- The published release was inspected read-only (`gh release view v1.0.0`, one archive downloaded to scratch and listed).

| Measure | Count |
|---|---|
| Accuracy mismatches (doc statement contradicted by code, binary or repository state) | **52** (M-01 … M-52): 11 High, 19 Medium, 22 Low |
| Completeness checklist items | 47 |
| — present | **1** |
| — partial | **21** |
| — absent | **25** |
| Ranked findings | 26 (DOC-01 … DOC-26): 11 High, 9 Medium, 6 Low |

Seven of the mismatches are behaviour defects rather than wording problems (the documented behaviour is the reasonable one and the binary does something else). They are collected in section 7 so they can be routed to the code reviewers; each needs a decision "fix code" or "fix doc" before the documentation deliverable can be written.

## 1. What is solid

- README quickstart and all ten EXAMPLES blocks are executed verbatim by `go test` on both Linux CI cells; ports, RPC names and JSON field names in them match the binary.
- All 11 `supkv_*` metric names and the five discard class label values in `docs/OPERATIONS.md:118-126` match `internal/metrics/metrics.go` and the live `/metrics` output exactly.
- Every flag default printed by `--help` matches the constants in `cmd/sup-kv/main.go:131-146` and the defaults quoted in docs (61440, 5s, 30s, 60s, 1s).
- Trust-model disclosure (no auth, no encryption, CRC is not tamper protection, reflection always on, loopback default) is accurate and placed first.
- All relative links in all Markdown files resolve; no anchors are used; all external links return 200.
- Every hand-written package has a package comment and every exported declaration has a doc comment (0 undocumented exported top-level declarations; `godoclint` is enabled).
- EXAMPLES sample outputs are real: the versions shown decode to 2026-11-17 13:15 UTC with node ids 1 and 2.
- `docs/setup/ruleset.json` matches the live ruleset (id 27576625, enforcement active, four rules, five check contexts equal to the `ci.yml` job names).

## 2. Accuracy mismatches

Each row quotes the document and the contradicting evidence. "Measured" means observed with the scratch binary or scratch test (Appendix B).

### 2.1 README.md and release artifacts

| ID | Document says | Code / reality | Sev |
|---|---|---|---|
| M-01 | `README.md:22-23` "The v1.0.0 release artifacts are in preparation; until the tag lands, build from source." | Annotated tag `v1.0.0` exists at `0ead1e8` (2026-11-17). GitHub Release `v1.0.0` published `2026-11-17T17:41:59Z` with 9 assets (4 archives, 4 SBOMs, `checksums.txt`). `CHANGELOG.md:10` "## [1.0.0] - 2026-11-17". The stale sentence also ships inside every release archive's `README.md`. README has no install-from-release instructions. | High |
| M-02 | `README.md:90-91` "The block cleans up after itself." (run, per `README.md:84`, "at the repository root") | Block `README.md:53-81` has no `rm`. It leaves `supkv-a.json` and `supkv-b.json` in the repository root; neither is in `.gitignore`. A second run starts both nodes with `demo/hello` already in their dumps, so "read it on node B" succeeds without any replication. | Medium |
| M-03 | `README.md:129-131` "this variant derives the segment broadcast address from the primary interface instead" | `README.md:97` `BCAST=$(ipconfig getoption en0 broadcast 2>/dev/null \|\| echo 127.255.255.255)` hard-codes `en0` and needs a DHCP-supplied broadcast option. Measured on macOS 27.2 arm64: `ipconfig getoption en0 broadcast` prints nothing, exit 1, so the block silently falls back to `127.255.255.255` — the address the same paragraph says "does not fan out on macOS". Measured with that address: `supkv_replication_peers 0` on both nodes, `Get` on B is `NotFound`, no log line at debug level. | Medium |
| M-04 | `README.md:139-145` links `docs/OPERATIONS.md`, `docs/EXAMPLES.md`, `docs/IDEA.md`; `docs/EXAMPLES.md:18-19` "or unpack a release archive" | Archive `sup-kv_1.0.0_Linux_x86_64.tar.gz` contains exactly `CHANGELOG.md`, `LICENSE`, `README.md`, `sup-kv`. No `docs/`, no proto, no unit file. The three links are dead for an archive user. | Medium |

### 2.2 "Proven" claims

| ID | Document says | Code / reality | Sev |
|---|---|---|---|
| M-05 | `docs/OPERATIONS.md:40-41` "Convergence bounds (proven in simulation under datagram loss, duplication, reordering and partitions)"; `README.md:29-30` "convergence bounds proven by a deterministic simulator"; `CHANGELOG.md:27-29` same | The three bounds are each asserted by one scripted scenario on a fault-free network: `internal/app/repl/sim_bound_test.go:37-39` "every datagram delivered after exactly simOneWay, no drops, no duplicates". The randomized test (`sim_chaos_test.go`, drop/dup/reorder/partition) asserts only eventual equality at quiescence with a cap of "10 × sync_interval past the last write", not the documented bounds, with one wall-clock seed per run. | Medium |
| M-06 | `docs/OPERATIONS.md:44-45` "datagram loss: retransmission with exponential backoff bounds recovery at `ack_timeout_max` (default 5s)" (unconditional) | `sim_bound_test.go:102-108` restricts it: "the bound covers loss episodes whose delivery opportunity falls inside the window … sustained loss past the window is the partition-heal bound's territory". The ladder gives five attempts in 5 s (0, 200, 600, 1400, 3000 ms; `core.go:854-861`); if all are lost the next is at 6200 ms. | Low |

### 2.3 docs/OPERATIONS.md

| ID | Document says | Code / reality | Sev |
|---|---|---|---|
| M-07 | `docs/OPERATIONS.md:3-5` "The operational contract … in one file: what the system promises, what it does not promise, which knobs exist" | The file names 4 of the 20 configuration flags (`--grpc-addr`, `--value-max-bytes`, `--drift-warn`, `--metrics-addr`). Twelve flags appear in none of README / OPERATIONS / EXAMPLES: `startup-gate`, `wait-timeout`, `sync-chunk-bytes`, `hello-interval`, `peer-timeout`, `ack-timeout`, `ack-timeout-max`, `retransmit-interval`, `sync-interval`, `sync-response-rate`, `dump-debounce`, `dump-max-wait`. Lines 45-47 name two of them as `ack_timeout_max` and `sync_interval`, spellings the binary does not accept. | High |
| M-08 | `docs/OPERATIONS.md:54-56` "values are at most 60 KiB — 61440 bytes, the default of `--value-max-bytes`" (presented as a tunable default) | 61440 is a hard ceiling of two other components. `internal/app/persist/codec.go:25` `const maxRecordBytes = 61440`, `:139-141` discards larger records at load. The flag accepts any integer (`main.go:311`). Measured with `--value-max-bytes 200000`: a 100 kB `Set` returns OK; replication logs `sendto: message too long`; after restart the key is `NotFound` and the only trace is `"sup-kv: discarded 1 undecodable dumped records at startup"`. A confirmed write is lost. | High |
| M-09 | `docs/OPERATIONS.md:83-85` "A merge loser is an ordinary overwrite, not an event … The observable is the `supkv_merges_conflicts_total` counter"; HELP text `metrics.go:245` "Remote merges that displaced a stored record." | `core.go:531` `c.mergeStats.Conflicts += uint64(len(conflicts))`; `core.go:214` "Conflicts is every displaced-or-rejected version". Measured through `persist.Open` + `repl.New`: an ordinary sequential replicated update of an existing key (no concurrency) adds 1; a rejected stale redelivery adds 1. The counter is "replicated overwrites plus stale deliveries", not concurrent-write conflicts, and the HELP text omits the rejected half. | High |
| M-10 | `docs/OPERATIONS.md:89-92` "`supkv_clock_drift_warnings_total` rises on one node while its peers stay flat. A warning is counted when an absorbed remote timestamp sits beyond the `--drift-warn` threshold ahead of the node's own clock"; `:94` "Cause. The node's wall clock runs behind (or ahead of) its peers'." | `clock.go:124` computes drift from the node's own pre-update `l` against its own wall clock; `core.go:532-534` counts any non-zero value. Measured with perfectly synchronized clocks: +1 on the first replicated write a fresh node receives; +1 on every replicated write that arrives after the node was quiet for longer than `--drift-warn` (60 s); +1 on the first replicated write after a restart from an old dump. A far-ahead stamp is not counted when absorbed, only on later merges (`merge_stats_test.go:55-62`). In the target workload ("units of writes per hour") the counter rises on nearly every replicated write. | High |
| M-11 | `docs/OPERATIONS.md:126-127` "plus the standard `go_*`/`process_*` runtime families" | `metrics.go:322-363` exports exactly five: `go_goroutines`, `go_memstats_alloc_bytes`, `go_memstats_sys_bytes`, `process_cpu_seconds_total`, `process_resident_memory_bytes`. The last is `ru_maxrss` (`metrics.go:368-380`), i.e. peak RSS, under HELP "Resident memory size in bytes." | Low |
| M-12 | `docs/OPERATIONS.md:146-150` "exit code 1 — the final dump failed …; exit code 2 — malformed configuration: a bad flag value or a malformed `SUPKV_*` environment value" | `main.go:231` and `:241` call `log.Fatalf` (exit 1) for every wiring failure. Measured: `--bind-addr not-an-address` → exit 1; `--broadcast-addr nonsense` → exit 1; `--grpc-addr nonsense` → exit 1; port in use and unreadable TLS files take the same path. Exit 1 therefore also means "could not start". | High |
| M-13 | `docs/OPERATIONS.md:137-138` "Structured logs go to stderr as JSON by default. `SUPKV_LOG_LEVEL` (debug through error) … tune them." | `main.go:255-258`: every `log.Printf`/`log.Fatalf` is emitted at level INFO. Measured: `{"level":"INFO","msg":"sup-kv: final dump failed: …"}` and `{"level":"INFO","msg":"sup-kv: bind replication socket: …"}`. With `SUPKV_LOG_LEVEL=warn` or `error` both cases exit 1 with **0 bytes** on stderr. Usage errors (exit 2) are plain text, not JSON. | High |
| M-14 | `docs/OPERATIONS.md:154-156` "On restart the node loads its dump file (falling back to the previous generation, `.prev`, on a corrupt one), and any gap is healed by replication from its peers." | Third step omitted: `dumpfile.go:71` "Both generations exhausted is the empty start"; `engine.go:116` "No decodable generation at all is the empty start, never an error". Measured with both files unreadable: the node starts empty and serves writes, with no log line (debug level) and no metric. Dump failures at runtime are not logged either (`engine.go:202`). | High |
| M-15 | `docs/OPERATIONS.md:17-21` "the gRPC API has no authentication and no authorization" | Also no transport security and no option for it: `main.go:588` `gs := grpc.NewServer()`. The paragraph discloses two of three properties; the replication paragraph above it does say "no encryption". | Low |
| M-16 | `docs/OPERATIONS.md:147-148` "(a systemd unit with `Restart=on-failure` reacts to this)" | No unit file exists in the repo or the release archive. `.planning/PROJECT.md:26` records "systemd unit lands in Phase 7". With `Restart=on-failure` and no `RestartPreventExitStatus=2`, the exit-2 configuration error restart-loops as well. | Medium |

### 2.4 docs/EXAMPLES.md

| ID | Document says | Code / reality | Sev |
|---|---|---|---|
| M-17 | `docs/EXAMPLES.md:57-61` "a node answers health once its startup gate has opened" with `grep -q SERVING` | `gate.go:93-95`: the health endpoint answers from construction, with `NOT_SERVING` until the gate opens. `grep -q SERVING` matches `NOT_SERVING`, so the wait loop ends as soon as the node answers at all. `README.md:59` uses the correct `grep -q '"SERVING"'`. The test assertion `examples_test.go:285,295` `expect: []string{"SERVING"}` is satisfied by `NOT_SERVING` too. | Medium |
| M-18 | `docs/EXAMPLES.md:92` "The listing names the three services a node serves" | The listing that follows (`:95-98`) has four entries. | Low |
| M-19 | `docs/EXAMPLES.md:3` "A complete walk through the sup-kv API"; `README.md:142-143` "every RPC as a copy-paste grpcurl walkthrough" | `create_if_absent` (`supkvapi.proto:67`, a CHANGELOG headline) is never used or named. The strings `create_if_absent`, `AlreadyExists`, `Unavailable`, `ResourceExhausted` occur 0 times in README, OPERATIONS and EXAMPLES. | Medium |
| M-20 | `docs/EXAMPLES.md:320-322` "`SnapshotLoad` replays the named snapshot as ordinary writes with fresh versions, so the restore replicates to peers and wins"; `:329` "...and replay it back as ordinary writes" | `engine.go:555-562`: "keys absent from the snapshot keep their state untouched"; an identical snapshot "replays as a whole-restore no-op". The example loads immediately after creating, so it replays nothing. Measured: a key created after the snapshot survives `SnapshotLoad`; a key deleted after it is resurrected. The spec says keys absent from the snapshot are deleted. | High |
| M-21 | `docs/EXAMPLES.md:40-44` loopback-broadcast pair with no platform caveat; `:8-9` "what you copy from this page is exactly what CI proves works" | `examples_test.go:319-321` skips on darwin; its message calls the page "a CI-verified Linux walkthrough". Measured on macOS: peers stay 0 and "Write and read" fails at the final `Get` on node B. The page never says it is Linux-only. | Medium |
| M-22 | `docs/EXAMPLES.md:211-214` "A `false` acked means a peer did not confirm in time" | `wait.go:101-105`: zero known peers returns `acked=true` immediately. Measured with the peer unreachable: `{"version": "…", "acked": true}` (no `ackedCount`, no `totalPeers`). `acked: true` does not mean a peer has the write. Neither the page nor `supkvapi.proto:72-75` says so. | High |

### 2.5 docs/IDEA.md (linked from README as "the decided architecture, parameters and failure-scenario rationale")

| ID | Document says | Code / reality | Sev |
|---|---|---|---|
| M-23 | `docs/IDEA.md:139` "Conflict journal: lost versions logged on merge conflicts." Also `:163-164`, `:168-169`, `:188-190`, `:213`. `.claude/CLAUDE.md:21` "(dump, logs, conflict journal)". `store.go:63-64` "contract behind the Phase 4/6 conflict journal" | `docs/OPERATIONS.md:83-84` "there is no conflict journal and none is needed"; `core.go:214-215` "counted and never journaled". The cancellation is recorded only in `.planning` ("conflict journal cancelled per phase-6 D-10"). | High |
| M-24 | `docs/IDEA.md:104-106` "Dump = JSON file with the full map contents: format version + records (`Key`, `RecordVersion`, `Data` base64, `Deleted`, `Checksum`)"; `README.md:31` "Human-readable JSON dumps"; `docs/EXAMPLES.md:356-358` "human-readable JSON" | `codec.go:39-54`: JSON Lines. Line 1 `{"format_version":1}`, then one object per record with `key`, `record_version` (32 hex characters), `data` (base64 or `null`), `deleted`, `checksum` (decimal). Measured file content in Appendix B. The file is not one JSON document. | Medium |
| M-25 | `docs/IDEA.md:67-69` "62-bit `rand_b` = 16-bit `NodeID` in the high bits + random filler" | `version.go:21-22` "the remaining 46 rand_b bits are zero filler". | Low |
| M-26 | `docs/IDEA.md:78-79` "`run_id` (fresh UUID per process start)" | `core.go:189-199` "16 random bytes from crypto/rand". | Low |
| M-27 | `docs/IDEA.md:83-84` "`DataReplication` — … unicast as a reply to `DataRequest` and when the receiver has a newer version" | `core.go:523-543`: a stale `DataReplication` is merged (rejected), then Acked at `:542`; nothing is sent back. The newer-record unicast exists only in `handleDataSync` (`core.go:640-641`). The spec says: reply with the local record and send no Ack. | Medium |
| M-28 | `docs/IDEA.md:64-66` "`Update` on receiving any foreign version" | `core.go:631-644` `handleDataSync` never touches the clock; only a `DataReplication` merge does. The spec requires `Update` once per DataSync chunk. | Low |
| M-29 | `docs/IDEA.md:125-127` API list: Get, Set, Delete, Watch, snapshot create/load. `:144-148` parameter list. | `List` exists (`supkvapi.proto:38`). The parameter list omits `grpc-addr`, `wait-timeout`, `metrics-addr` and all env-only knobs, and uses underscore names. | Low |
| M-30 | `docs/IDEA.md:204` "Client caching libraries (documented pattern; shipped later)"; `:51-52` key-prefix convention "enforced by docs, not by the store" | Neither the cache pattern nor the key-prefix convention is documented anywhere outside IDEA.md itself. | Medium |
| M-31 | `docs/IDEA.md:222-223` "two nodes on one localhost broadcast socket"; `:224` "green in CI on Linux and macOS, on amd64 and arm64" | `README.md:53-56` uses two different UDP ports, each node broadcasting to the other's port. `ci.yml:65` darwin/amd64 is a "cross-compile cell" (build and vet only, no tests). | Low |

### 2.6 AGENTS.md

| ID | Document says | Code / reality | Sev |
|---|---|---|---|
| M-32 | `AGENTS.md:5` "as pinned in `go.mod` (`go 1.27`)"; `.claude/CLAUDE.md:35` "`go 1.27` directive; current patch 1.27.1" | `go.mod:3` `go 1.27.4`. | Low |
| M-33 | `AGENTS.md:6-7` "Run `make lint` and `make test` (`go tool golangci-lint run ./...` and `go test -race ./...`) …; CI runs the same targets." | `Makefile:11-17`: lint also runs `config verify`; test is `go test -race -cover ./...`. `ci.yml` calls no make target and additionally runs `buf lint`, `buf format --diff`, `buf breaking`, `go vet` and the `go mod tidy` diff. | Low |
| M-34 | `AGENTS.md:36-39` "`internal/app/` — pure decision core, split into three packages: `version` …, `clock` … and `storage`" | Five packages: `clock`, `persist`, `repl`, `storage`, `version`, plus `internal/simnet` and `internal/testutil`. | Medium |
| M-35 | `AGENTS.md:44-45` "Adapter packages under `internal/` appear with the phases that need them; do not create empty adapter directories." | Four adapters exist and are not described: `internal/grpcapi`, `internal/udprepl`, `internal/persistfile`, `internal/metrics`. | Medium |
| M-36 | `AGENTS.md:49-51` "Phase 1 carries **zero direct runtime requires**. gRPC and protobuf enter `go.mod` in the phases that first use them." | `go.mod:16-17` requires `google.golang.org/grpc v1.85.0` and `google.golang.org/protobuf v1.36.13-0.20261016172727-e9ee2a76ab8e` directly. | Medium |
| M-37 | `AGENTS.md:52-55` "The `tool` directives (golangci-lint v2.15.0, govulncheck v1.9.0) drag ~200 `// indirect` requires … the shipped build graph stays empty" | `go.mod:5-13` has seven tool directives (buf, grpcurl, golangci-lint, goreleaser, govulncheck, protoc-gen-go-grpc, protoc-gen-go) and 538 `// indirect` lines. | Low |

### 2.7 .claude/CLAUDE.md (stack section)

| ID | Document says | Code / reality | Sev |
|---|---|---|---|
| M-38 | `.claude/CLAUDE.md:59` "`pgregory.net/rapid` v1.3.0 … The one justified test-only dependency" | Not imported anywhere and not required in `go.mod` (only `v1.2.0` hash lines in `go.sum`). Property tests use a hand-rolled seeded `math/rand/v2` (`internal/app/internal/testutil/rng.go`). | Medium |
| M-39 | `.claude/CLAUDE.md:49,61` simulator: "nodes as goroutines inside a `synctest` bubble" | `simnet.go:11-12` "The package starts no goroutines, takes no locks and never reads a wall clock" (discrete-event scheduler). `testing/synctest` is imported only by `internal/app/persist/engine_test.go`. | Medium |
| M-40 | `.claude/CLAUDE.md:45` "Call `SetReadBuffer` early — kernel default rcvbuf is the main datagram-loss source" | No `SetReadBuffer` or `SO_RCVBUF` anywhere in the code. | Low |
| M-41 | `.claude/CLAUDE.md:46` "Use `json.MarshalIndent` for human readability" | `codec.go:62-85` uses `json.Encoder`, one object per line. | Low |
| M-42 | `.claude/CLAUDE.md:52` "`runtime/debug.ReadBuildInfo` … Prefer over `-ldflags -X`" | `main.go:51-54`: ldflags only. The README build command yields `sup-kv dev dev go1.27.4`. | Low |
| M-43 | `.claude/CLAUDE.md:39` "golang.org/x/sync v0.23.0"; `:71,136` "Pin `version: v2.15`"; `:134` "protoc-gen-go v1.36.12 … protobuf runtime v1.36.12 … keep plugin and runtime versions identical" | `go.mod:531` `golang.org/x/sync v0.25.0`; `ci.yml:52` `version: v2.15.0`; `go.mod:17` runtime `v1.36.13-0.2026…` (line 37 of the same file says so; the file contradicts itself). | Low |
| M-44 | `.claude/CLAUDE.md:162` "Conventions not yet established."; `:169` "Architecture not yet mapped." | v1.0.0 is shipped. The Development Tools table also omits the goreleaser v2.20.2 and grpcurl v1.9.4 tool directives (`go.mod:7,9`). | Low |

### 2.8 Proto files and package docs

| ID | Document says | Code / reality | Sev |
|---|---|---|---|
| M-45 | `supkvapi.proto:2-5` "the later phase-5 plans (Watch, snapshots, wait_for_replication) consume the already-generated messages; the unimplemented RPCs inherit Unimplemented from the server embed" | `server.go:42-43` "every phase-5 RPC is implemented on it today". All seven RPCs are implemented. | Medium |
| M-46 | `supkv.proto:4-6` "the Phase 4 wave-1 core consumes only Hello, DataReplication and Ack — DataRequest and DataSync land with the later plans"; `core.go:3-5` package doc "the Hello/DataReplication/Ack processing of the tracer path" | `handleDataRequest` and `handleDataSync` are implemented (`core.go:605`, `:631`), as are retransmission, eviction and the egress limiter. | Low |
| M-47 | `supkvapi.proto:60-62` "wait_timeout_ms (D-09) bounds that wait per request (0 = the operator-tunable default)" | `wait.go:210-224`: negative also means default, and the value is capped at 10 × `--wait-timeout` (10 s with defaults). A request for 60000 ms waits at most 10 s. | Medium |
| M-48 | `storage/doc.go:5-6`, `store.go:73-74`, `core.go:175-176` "internal/port holds no interfaces" | `port/dump.go:16` `type DumpStore interface`, `port/repl.go:35` `type Message interface`; `port/doc.go:1-2` "the interface definitions". | Low |

### 2.9 Build, CI and setup documentation

| ID | Document says | Code / reality | Sev |
|---|---|---|---|
| M-49 | `Makefile:61-65` "supkv.proto is not on origin/master yet (probe-verified); it self-destructs the moment the proto lands there"; `ci.yml:34-36` "self-destructs the moment PR #31 merges" | `git cat-file -e refs/remotes/origin/master:internal/udprepl/replproto/supkv/v1/supkv.proto` succeeds. The guard's else-branch is dead and the comments describe a past state. | Low |
| M-50 | `.github/dependabot.yml:3-4` "Dependabot ALERTS are GHAS-gated on this private repo and stay off until the deferred public flip" | `gh api repos/Djarvur/sup-kv` returns `"visibility":"public"`; `branch-protection.md:17` "The repository flipped PUBLIC on 2026-11-18". | Low |
| M-51 | `docs/setup/branch-protection.md:5` "every command below is re-runnable verbatim"; verify command at `:44-49` | Run verbatim it prints `cannot iterate over: null` and exits 1: the list endpoint returns no `rules` field (keys: `_links, created_at, enforcement, id, name, node_id, source, source_type, target, updated_at`). `gh api repos/Djarvur/sup-kv/rulesets/27576625` returns the documented state. | Medium |
| M-52 | `Makefile` documents every target with `## target:` help comments | There is no `help` target: `make help` → "No rule to make target `help'". The comments are readable only by opening the file. | Low |

## 3. Completeness checklist

Verdict key: **present** = a reader can act on it; **partial** = mentioned but not sufficient to act; **absent** = not in the user-facing corpus.

### A. Configuration reference

| # | Item | Verdict | Where / what is missing |
|---|---|---|---|
| A1 | One table of all flags, env twins, defaults, valid ranges, meaning, tuning guidance, inter-parameter constraints | partial | Only `--help` (names, env twins, defaults, one-line meanings for 20 flags). No table in any document. Missing everywhere: the five env-only knobs in one place with the flags; valid ranges (`node-id` 1..65535; `value-max-bytes` ≤ 61440); the rule that zero or negative values silently become the default (`core.go:136-168`, `engine.go:37-45`, `gate.go:124-126`) except `drift-warn ≤ 0`, which disables the signal; constraints (`peer-timeout` > `hello-interval`, `ack-timeout` ≤ `ack-timeout-max`, `dump-debounce` ≤ `dump-max-wait`, same UDP port and same `value-max-bytes` on all nodes). The binary validates none of these (measured: `--ack-timeout -1s --sync-response-rate 0 --value-max-bytes -5` starts). |

### B. Deployment guide

| # | Item | Verdict | Where / what is missing |
|---|---|---|---|
| B1 | 2-node appliance topology | absent | No description of the interconnect deployment the product is for. |
| B2 | Choosing bind/broadcast addresses and ports | partial | `EXAMPLES.md:43-44` "on a real cluster pass your LAN's broadcast address instead". Missing: both nodes use the same UDP port on a real segment (the demo's crossed ports are a same-host trick); bind address must receive broadcasts (binding a unicast interface address on Linux is expected not to — untested here, must be verified and stated); conventional ports 7900/7901 appear only in `--help` examples; N > 2 nodes. |
| B3 | Firewall | absent | No list of ports and protocols (UDP replication, TCP gRPC, TCP metrics). |
| B4 | NodeID assignment | partial | `--help`: "required unique 16-bit node ID". Missing: 0 is rejected; uniqueness consequences; that the id is embedded in every record version and breaks ties. |
| B5 | File locations and permissions | partial | EXAMPLES names the dump files and `<dump>.snapshots/`. Missing: `<dump>.prev`, temp files `dump-*` in the dump directory, modes (files 0600, created directories 0700; `dumpfile.go:26-31`), the service user needs write access to the dump's parent directory. |
| B6 | systemd unit | absent | None ships (M-16). |
| B7 | Container usage | absent | No Dockerfile, no notes on host networking for broadcast. |
| B8 | Sizing: RAM, disk write rate, socket buffers | absent | Only "the whole state lives in RAM". Missing: the full-dump write per debounce window (spec: ~10 MB per dump at 10k × 1 KiB, up to ~100 MB/s under a constant write stream), `net.core.rmem_default/rmem_max` (the code never calls `SetReadBuffer`), measured benchmark numbers. |
| B9 | Time-sync recommendation | partial | Only as the runbook action "Fix the clock source — NTP or PTP" (`OPERATIONS.md:96`). Not stated as a deployment recommendation (spec: chrony over the direct link). |
| B10 | Upgrade procedure, rolling-upgrade and compatibility policy | absent | `proto_version` (currently 1, `core.go:32`) and dump `format_version` (1, `codec.go:20`) are not mentioned in any user document. What happens when versions differ (datagrams discarded as `bad_proto_version`; an unknown dump format is treated as unreadable and falls through to `.prev`, then to an empty start) is not documented. |
| B11 | Backup/restore with snapshots | partial | EXAMPLES shows the two RPCs. Missing: name rules (`[A-Za-z0-9._-]`, 1..64, not `.` or `..`; `snapshot.go:98-108`), silent overwrite of an existing name, real restore semantics (M-20), off-box copy, restoring onto a stopped node. |
| B12 | Node replacement / empty-disk recovery | absent | One clause "any gap is healed by replication from its peers". No procedure, no expected duration (keys ÷ `sync-response-rate`), no way to tell when it is done. |
| B13 | Duplicate NodeID symptom | partial | The label value `duplicate_node_id` is listed. No symptom description, no log line quoted, no fix. |

### C. Failure-behaviour reference

| # | Item | Verdict | Where / what is missing |
|---|---|---|---|
| C1 | Table equivalent to the spec's failure table (symptom → behaviour → recovery → metric/log) | partial | No table. Scattered prose covers about 6 of 18 spec rows (single loss, partition heal, restart, crash of a node, far-ahead clock, foreign datagram). Not covered at all: one-way connectivity, long single-node downtime, disk loss, burst load, duplicate NodeID, protocol mismatch. No row names the metric or log line that shows it. |

### D. Troubleshooting runbooks

| # | Item | Verdict | Where / what is missing |
|---|---|---|---|
| D1 | Dump failures / disk full | absent | Measured behaviour that needs a runbook: writes return `Unavailable: dump failed; write not confirmed`, yet the value is readable and replicates; a `create_if_absent` retry then returns `AlreadyExists`; the dump is retried every debounce interval; nothing is logged; health stays `SERVING`; only `supkv_dumps_errors_total` moves. |
| D2 | Peer not discovered | partial | Only the README macOS note. No runbook for `supkv_replication_peers 0` (wrong broadcast address, port mismatch, bind address, firewall). No log line exists for discovery or eviction. |
| D3 | Unacked backlog growing | absent | Metric named only. Not stated: with no peer the backlog grows to the number of written keys and its age grows without bound (measured: 5 records, age 25 s, peers 0), so an alert must be conditioned on peers > 0. |
| D4 | Split-brain conflicts, which write lost, conflict-journal consistency | partial | OPERATIONS says there is no journal. IDEA.md, CLAUDE.md and a Go comment still promise one (M-23). There is no way to see which write lost, and the suggested counter does not count conflicts (M-09). |
| D5 | Protocol version mismatch | absent | Label value only. |
| D6 | Corrupted dump | absent | One clause. Not documented: per-record discard with a single INFO line that does not name the keys; silent empty start when both generations are unreadable (M-14); how to inspect or hand-repair a dump. |

### E. Metrics and logging reference

| # | Item | Verdict | Where / what is missing |
|---|---|---|---|
| E1 | Every metric with type, labels, meaning, alert suggestions | partial | All names and the discard class values are listed. Missing: counter/gauge type; label names `type` and `class`; the five `type` values (`hello`, `data`, `ack`, `data_request`, `sync`); real semantics (M-09, M-10, M-11; `supkv_dumps_duration_seconds` is the last dump only); any alert guidance. |
| E2 | Log event catalogue | absent | The whole runtime catalogue is small and undocumented: WARN `Replication adapter` (`detail`), DEBUG `Discarded datagram` (`class`, `detail`), ERROR `Metrics endpoint`, and eight INFO messages from `log.Print*`/`log.Fatalf` call sites in `main.go` (lines 231, 241, 246, 421, 651, 664, 668, 684). A healthy node logs nothing at all, including at startup and shutdown (measured). |

### F. API reference and integrator guide

| # | Item | Verdict | Where / what is missing |
|---|---|---|---|
| F1 | Semantics of every RPC including error codes | partial | EXAMPLES shows each RPC's happy path and `NotFound`. No error table. Undocumented, measured: empty key → `InvalidArgument`; gated write → `Unavailable` "startup gate: waiting for first sync round or startup_gate"; dump failure → `Unavailable`; `create_if_absent` on a tombstone → `AlreadyExists`; invalid snapshot name → `InvalidArgument`; missing snapshot → `Internal` with the server file path in the message. |
| F2 | Watch guarantees | partial | EXAMPLES shows snapshot-then-live and the tombstone in the snapshot. Missing: snapshot in key order, live events in commit order; slow consumer is disconnected with `ResourceExhausted` "watch subscriber overwhelmed; resubscribe" after 256 queued events (`watch.go:26,42`); no resume token, a reconnect re-sends the full snapshot; the stream ends with OK status on server shutdown (measured). |
| F3 | `wait_for_replication` semantics | partial | EXAMPLES explains the three fields. Missing: zero-peer result (M-22), default and cap of the timeout (M-47), meaning under supersession, that it never changes the write result is stated. |
| F4 | List pagination and limits | partial | Proto comment: no pagination. Missing in user docs: keys only, tombstones excluded, single response bounded by the 4 MiB gRPC default (`server.go:196-200`). |
| F5 | Key naming convention | absent | Only in IDEA.md (M-30). |
| F6 | Value limits | present | `OPERATIONS.md:50-56`. (Upper bound of the flag is wrong, M-08.) |
| F7 | Consistency caveats | partial | Read-your-writes on the same node is documented. Missing: no multi-key atomicity; a delete can be undone by a concurrent update with a greater version; a write after a read always wins; a write that returned `Unavailable` may still be applied. |
| F8 | Client-side last-good-value cache pattern | absent | Promised as "documented pattern" (M-30). |
| F9 | Retry guidance for `Unavailable` | absent | Two different causes (gate, dump failure) need different client behaviour; the second one is not idempotent for `create_if_absent`. |
| F10 | How to generate a client; importability; published proto | absent | The stubs live under `internal/` and cannot be imported by another module; `go_package` points into `internal/`; the proto is not in the release archive and not published to a registry (`buf.yaml` modules have no name). README's "no proto files are needed" holds for grpcurl only. |
| F11 | Example client code in Go | absent | The only Go client code is in tests and uses the internal package. |

Also undocumented and relevant to integrators: a confirmed write takes at least `dump-debounce` plus one dump (measured ≈ 70 ms per `Set` against ≈ 10 ms per `Get`), and up to `dump-max-wait` plus one dump under sustained load.

### G. Wire-protocol and on-disk format specifications

| # | Item | Verdict | Where / what is missing |
|---|---|---|---|
| G1 | Wire protocol | partial | `supkv.proto` plus the IDEA.md narrative. Missing for a second implementation: the `proto_version` value; CRC coverage (IEEE over the marshaled `body` only, not the envelope header fields); record-version byte layout (only in `version.go`); the record checksum definition and the fact that the transported `checksum` is ignored on receive; chunking rule; rate limiter scope; as-built deviations (M-25 … M-28). |
| G2 | Dump format | absent | The only description (IDEA.md) is wrong (M-24). Needed for forensic inspection and repair: line structure, field encodings, `checksum` = CRC32-IEEE over key bytes ‖ 16 version bytes ‖ data bytes ‖ one flag byte (`store.go:131-137`), per-record discard rules, duplicate key is last-wins, size bound. |

### H. Maintainer documentation

| # | Item | Verdict | Where / what is missing |
|---|---|---|---|
| H1 | Architecture overview: package map, data flow, concurrency model | partial | `AGENTS.md` layout is stale (M-34, M-35). Package comments are detailed but there is no single overview of the write path, the three event loops, or the cross-goroutine mirrors. |
| H2 | Design decisions / ADRs in a user-visible place | absent | Decisions live in `.planning/`. Go comments cite them by id on 237 lines (`D-nn`), with no resolvable target. `AGENTS.md:5` says "Do not raise it without an ADR" but there is no ADR directory. |
| H3 | How to run the simulator with a seed | absent | Exists only as a string in test code: `SUPKV_SIM_SEED=<n> go test ./internal/app/repl/ -run '^TestChaos_convergence$'`; property tests: `-args -seed=<n>`. |
| H4 | How to add a failure scenario | absent | |
| H5 | CONTRIBUTING | absent | No file. `AGENTS.md` covers lint/test commands and three conventions. |
| H6 | SECURITY policy | absent | No file; private vulnerability reporting not mentioned. |
| H7 | Release process | partial | Makefile and `.goreleaser.yaml` comments. No written procedure (pre-tag snapshot gate, tag, verify assets, update README). |
| H8 | Versioning / compatibility policy | partial | `CHANGELOG.md:6` claims SemVer. Not stated what the version covers: gRPC schema, flags and env names, metric names, wire protocol, dump format. |
| H9 | Benchmark results and how to reproduce | partial | `make bench` exists and its comment names the three benchmarks. No results anywhere, no reference hardware. |

### I. Relationship to the upstream proposal

| # | Item | Verdict | Where / what is missing |
|---|---|---|---|
| I1 | Which spec this implements | absent | `IDEA.md:5` "it comes from a reviewed architecture proposal" (unnamed). The upstream spec is not named anywhere in the repo. |
| I2 | Where the implementation deliberately deviates | absent | No list. Deviations observed during this audit are in Appendix C. |

## 4. Quality notes

**Structure and navigability**

- `README.md` has `## Quickstart` (line 37) containing `### Quickstart` (line 49) and `### Quickstart on macOS` (line 93): two 30-line blocks that differ in one line and two substitutions. On Linux the second block executes identically to the first.
- `docs/EXAMPLES.md`: every walkthrough section is a `###` under `## Prerequisites`; there is no walkthrough parent heading.
- There is no documentation index by audience. An operator has one file of 156 lines; an integrator has a grpcurl walkthrough; a maintainer has `AGENTS.md` (stale) and code comments.
- Three spellings for every parameter: `ack_timeout_max` (OPERATIONS, IDEA), `--ack-timeout-max` (flag), `SUPKV_ACK_TIMEOUT_MAX` (env). Only the last two work.
- No glossary. The same mechanism is called "anti-entropy sync", "full reconciliation", "sync round", "full sync" and "listing"; the peer is "peer", "neighbor", "neighbour" and "replica".

**Internal jargon in reader-facing text**

- `CHANGELOG.md`: "(Phase 2)" … "(Phase 6)" on all seven entries. The GitHub Release body is the whole file, including the empty `## [Unreleased]` heading, followed by goreleaser's automatic commit list with entries such as "docs(01): add code review report".
- `docs/IDEA.md:3-6` opens with "Seed for `/gsd:new-project --auto @docs/IDEA.md`".
- `docs/setup/branch-protection.md`: "decision D-08", "CI-03", "Pitfall 6".
- `supkvapi.proto`: decision ids on 16 lines ("D-01" … "D-15"), "phase-5 plan 05-04", "plan 05-05". These comments are copied into the generated Go code and are what `grpcurl describe` users and client authors read.
- Operator-visible log text: `core.go:400,409` "…datagram discarded (REPL-08)".
- Go comments in non-test, non-generated code: `D-nn` on 237 lines, requirement ids on 122, review ids (`WR-`, `IN-`, `CR-`) on 23, threat ids on 24, "Pitfall N" on 22, "probe #N" on 13, plan/research file references on 23, "Phase N" on 26. None of the targets is reachable from the published documentation.
- Makefile help comments: "TEST-05", "D-13", "D-14", "D-18", "(07-05)".

**Stale tense**

- `storage/errors.go:6-17` and `persist/errors.go:6-10` "Phase 5 maps them to RPC codes"; `engine.go:448-449` "Phase 4's remote merges will reuse the dirty path"; M-45, M-46.

**Tone**

- "proven" / "proving" appears in `README.md:21,30`, `OPERATIONS.md:40,102`, `CHANGELOG.md:18,28`. The evidence is tests (M-05). `README.md:89` "so it cannot drift from reality" holds only for the marked block, not for the surrounding prose.

**Copy-paste safety**

- README block: leftover dump files and false positive on rerun (M-02); macOS variant degrades silently (M-03).
- `docs/EXAMPLES.md:351-352` `ps -p … && exit 1`: pasted into an interactive shell, this closes the user's terminal if a node is still alive.
- EXAMPLES needs `sup-kv` on `PATH`, but the suggested build puts it in `bin/` and no line says to add it.
- EXAMPLES leaves `ex-a.pid`, `ex-b.pid`, dumps and a `.snapshots` directory in the working directory; only the dumps are mentioned.
- README and EXAMPLES build the binary two different ways (`go build -o sup-kv ./cmd/sup-kv`, unstamped; `make build`, stamped and static).

**Proto comments as API documentation**

- They describe shapes well. They do not list status codes per RPC (only `NotFound` on `GetResponse` and `AlreadyExists` on `SetRequest`), carry stale statements (M-45, M-46) and omit behaviour a client needs (M-22, M-47, Watch termination).

**Go doc comments**

- Coverage is complete. Content is accurate where checked, with the contradictions in M-48 and the stale items above. Comments are long and reference-heavy; a reader without `.planning/` cannot follow the cited ids.

**`--help`**

- Lists 20 configuration flags with env twins and defaults. Does not list the five env-only knobs, gives no ranges, prints flags alphabetically rather than grouped, and does not say node id 0 is invalid.

**CHANGELOG conformance**

- Structure follows Keep a Changelog (Unreleased, dated version, `Added`, link references). Deviations: phase suffixes; no "Known limitations" although several exist.

**Language**

- All repository documentation is English; the upstream spec is Russian. No term mapping between the two exists.

## 5. Docs-as-tests coverage

All three tests are in package `main_test` under `cmd/sup-kv/` and run in CI through `go test -race ./...` (`ci.yml:83`) on `ubuntu-latest`, `ubuntu-24.04-arm` and `macos-15`. `TestReadmeQuickstart` and `TestExamplesDoc` skip on darwin (`readme_test.go:84-86`, `examples_test.go:319-321`), so they execute on the two Linux cells only. All three also skip if the test binary build fails (`main_test.go:113-123`).

### 5.1 `TestReadmeQuickstart` (`readme_test.go`)

Machine-verified:

- Both fenced blocks preceded by `<!-- quickstart-verify -->` are extracted from `README.md` and run verbatim with `bash -e` in a temp directory holding `./sup-kv`, with `grpcurl` resolved to the pinned v1.9.4 tool. Each block must exit 0.
- Inside a block, the retry loops are not assertions; the unconditional `grpcurl … Get` after each loop is (a missing key makes grpcurl exit non-zero).
- The combined output of both blocks contains `c3VwLWt2` at least once (`readme_test.go:105`).
- `README.md` contains the strings `docs/OPERATIONS.md` and `docs/EXAMPLES.md`.

Not verified:

- The build command at `README.md:41-43` (unmarked; the test builds with `go build -o <bin> .`).
- The macOS variant as a macOS variant. On Linux `ipconfig` is absent, the fallback arm runs, and the block is the first block again. The derive arm is executed nowhere.
- The second block's replication: both blocks share the directory and dump file names, so node B starts the second block with `demo/hello` already on disk (M-02).
- Any prose: Status, Highlights, "within one round trip", "proven", the cleanup claim.
- The output substring is checked once over the union of both blocks, not per block.

### 5.2 `TestExamplesDoc` (`examples_test.go`)

Machine-verified:

- All ten blocks preceded by `<!-- examples-verify -->` run verbatim, in document order, in one shared directory, with `sup-kv` and `grpcurl` on `PATH`. Each must exit 0 under `bash -e`.
- Each block's combined output must contain its table substrings (`examples_test.go:281-298`): `SERVING`; `supkvapi.v1.SupKVService`; `NotFound`; `demo/hello` and `c3VwLWt2`; `"acked": true`, `ackedCount`, `totalPeers`; `demo/wait`; `demo/live`; `SERVING`; `{}`; `demo pair stopped`.
- Title set and table agree in both directions.

Not verified:

- The ten unmarked output fences ("output from a real run") and the one unmarked command (`printf sup-kv | base64`). The outputs are never compared with real output; only the substrings above are.
- `SERVING` is satisfied by `NOT_SERVING` (M-17).
- "Snapshot round-trip" asserts `{}` only, and the load is a no-op (M-20). Restore behaviour is not exercised.
- "Watch a stream" asserts that `demo/live` arrives. Ordering, snapshot-first and the tombstone in the snapshot are not asserted.
- "Wait for replication" retries the write up to 25 times until `acked` is true, so a first-attempt failure is invisible.
- "Stop the pair" asserts the processes are gone, not the "exit code 0" stated in its comment.
- `create_if_absent`, every error code except `NotFound`, and all prose.

### 5.3 `TestOperationsDoc` (`operations_test.go`)

- Verifies that the first `## ` heading is `## Trust model` and that 41 literal strings occur somewhere in the file.
- Its own header says so (`operations_test.go:7-12`): "HONEST SCOPE: this is a presence/anchor guard only … the TRUTH of each statement … is the phase's manual UAT verification".
- No statement in `docs/OPERATIONS.md` is checked against the binary. Anchored literals include only 2 of the 11 metric names, 4 of the 8 env names used in the file, and none of the numeric defaults except `61440` (as a string, not compared with `defaultValueMaxBytes`).

### 5.4 Where a document can rot unnoticed

| Change | Detected today? |
|---|---|
| Rename or remove a metric, label or label value | No. `metrics_test.go` pins the code; nothing compares OPERATIONS with `/metrics`. |
| Change a flag default, add or remove a flag | No. Nothing compares any document with `--help`. |
| Change an exit code or a log message | No. |
| Change the semantics of a counter (M-09, M-10 already happened this way) | No. |
| Change an EXAMPLES sample output shape | No, unless one of the listed substrings disappears. |
| Edit IDEA.md, CHANGELOG, AGENTS.md, CLAUDE.md, branch-protection.md, proto comments, package docs | No test reads them. The `ruleset.json` contexts are not compared with `ci.yml` job names either. |
| Break the macOS quickstart or the EXAMPLES walkthrough on macOS | No (skipped). |

## 6. Ranked findings

Each finding lists evidence (mismatch ids and checklist items), the deliverable, and an acceptance criterion that can be checked mechanically or by one command.

### High

**DOC-01 — Clock-drift runbook describes behaviour the binary does not have.**
Evidence: M-10. The runbook's symptom and cause are wrong for the shipped counter; an operator following it will chase NTP on healthy nodes.
Deliverable: a decision recorded in the review (change the counter to count absorbed stamps beyond the threshold, or keep it), then a rewritten "Clock drift runbook" stating the exact trigger, what a healthy idle cluster shows, and an alert expression.
Acceptance: a test with two in-sync nodes, one replicated write after 2 × `drift-warn` of quiet, asserts the counter value the runbook states; the runbook text is anchored to that test's name.

**DOC-02 — `--value-max-bytes` is documented as a tunable default; raising it loses confirmed writes.**
Evidence: M-08.
Deliverable: configuration-reference entry: valid range 1..61440, must be equal on all nodes, what happens beyond it. (Code side: reject out-of-range values with exit 2.)
Acceptance: the reference states the range; starting with `--value-max-bytes 61441` behaves as the reference says, covered by a test.

**DOC-03 — `wait_for_replication` reports `acked: true` when no peer is known.**
Evidence: M-22, F3. Integrators will read `acked` as "the other board has it".
Deliverable: integrator-guide section with a truth table over (`acked`, `acked_count`, `total_peers`) including zero peers, peer evicted during the wait, supersession, timeout; the same text in the `SetResponse`/`DeleteResponse` proto comments; the rule "treat the write as replicated only if `acked` and `total_peers ≥ 1`".
Acceptance: an EXAMPLES block run with the peer stopped shows and asserts the zero-peer response.

**DOC-04 — No configuration reference.**
Evidence: A1, M-07.
Deliverable: `docs/CONFIGURATION.md` with one table: flag, env twin, default, unit, valid range, behaviour on zero/negative, meaning, tuning note, constraints; plus the five env-only knobs and a "must match across nodes" column.
Acceptance: a test iterates the binary's registered flags and the env-only constants and fails if any is missing from the table or its default differs; OPERATIONS uses only spellings the binary accepts.

**DOC-05 — No deployment guide; no systemd unit.**
Evidence: B1–B13, M-16.
Deliverable: `docs/DEPLOYMENT.md` covering the two-board topology, bind and broadcast address selection on the interconnect, ports and firewall, NodeID assignment, file layout and permissions, a shipped `sup-kv.service` (with `Restart=on-failure`, `RestartPreventExitStatus=2`, `TimeoutStopSec` above the 10 s grace period), container notes, sizing (RAM, dump write rate, `rmem`), time-sync recommendation, upgrade and compatibility policy, backup/restore, node replacement, duplicate-NodeID symptom. Ship the unit and the docs in the release archive.
Acceptance: every checklist item B1–B13 has a section; the unit file is in `.goreleaser.yaml` archives and starts the binary in a CI smoke test; `tar tzf` of a snapshot archive lists `docs/` and the unit.

**DOC-06 — No failure-behaviour table and no runbooks; several failures are silent.**
Evidence: C1, D1–D6, M-13, M-14.
Deliverable: a table in OPERATIONS with one row per spec failure scenario (symptom, behaviour, recovery, bound, metric, log line), and runbooks for dump failure/disk full, peer not discovered, growing unacked backlog, protocol mismatch, corrupted dump, duplicate NodeID. Where no signal exists today (empty start, dump failure log, discovery/eviction log), the row says so until the code adds one.
Acceptance: 18 rows, each with a metric name or log message that exists in the binary (checked by a test that greps `/metrics` output and the log-message constants).

**DOC-07 — No API reference or integrator guide.**
Evidence: F1–F5, F7–F11, M-19, M-47.
Deliverable: `docs/API.md`: per-RPC semantics and status-code table; Watch contract (ordering, snapshot, slow-consumer disconnect, reconnect, stream end); write latency model; consistency caveats; key-prefix convention; retry guidance per `Unavailable` cause; the client-side last-good-value cache pattern; how to obtain the proto and generate a client; a compilable Go client example. Decide where the proto and stubs live so another module can import them.
Acceptance: every status code produced by `server.go`, `gate.go` and `watch.go` appears in the table (test over the code constants); the Go example builds in CI as a separate module against the published proto path.

**DOC-08 — Snapshot restore is described incorrectly and is not a point-in-time restore.**
Evidence: M-20, B11.
Deliverable: correct text in EXAMPLES and the backup/restore procedure: what load changes, what it leaves, the no-op case, name rules, overwrite behaviour, error codes; an explicit note that this differs from the spec.
Acceptance: an EXAMPLES block that changes a key, adds a key and deletes a key after the snapshot, loads, and asserts all three outcomes.

**DOC-09 — Conflict observability is misdocumented and the conflict journal is promised in three places.**
Evidence: M-09, M-23, D4.
Deliverable: one consistent statement everywhere (README-linked design text, CLAUDE.md, Go comment): no journal; what `supkv_merges_conflicts_total` counts; that the losing write cannot be identified. Correct the HELP string.
Acceptance: `grep -ri "conflict journal"` over the repo (outside `.planning`) returns only statements that it does not exist; the metric's documented definition matches a unit test.

**DOC-10 — README status is stale and there is no install path for the release.**
Evidence: M-01, M-04.
Deliverable: README "Status" and "Install" sections for v1.0.0 (download, `checksums.txt` verification, SBOM), and docs included in the archive or linked by absolute tagged URL.
Acceptance: no "in preparation" text; a release-snapshot check fails if a relative link in the archived README does not resolve inside the archive.

**DOC-11 — Exit codes and logging are documented incompletely and partly wrongly.**
Evidence: M-12, M-13.
Deliverable: exit-code table including startup failures; statement of which messages are emitted at which level and the effect of `SUPKV_LOG_LEVEL` above `info`. (Code side: log fatal conditions at ERROR.)
Acceptance: a test starts the binary with an unusable `--grpc-addr` and with a failing final dump under `SUPKV_LOG_LEVEL=error` and asserts the documented exit code and a non-empty stderr.

### Medium

**DOC-12 — IDEA.md is presented as the design but is not as-built, and the upstream proposal is never named.**
Evidence: M-23 … M-31, I1, I2.
Deliverable: either rename it clearly as a historical seed with a banner, or replace it with `docs/DESIGN.md` (as-built). Add "Relationship to the upstream spec" with a deviation table (start from Appendix C).
Acceptance: each deviation row cites a code location; README's description of the file matches its banner.

**DOC-13 — No on-disk format or wire-protocol specification.**
Evidence: G1, G2, M-24.
Deliverable: `docs/FORMATS.md`: dump line grammar with a worked example, checksum definition, discard rules, format version policy; envelope and message layout, `proto_version`, CRC coverage, record-version byte layout, how to decode a version by hand.
Acceptance: a test encodes a fixed record set and compares it byte-for-byte with the example in the document; the documented checksum recipe reproduces the example's checksum.

**DOC-14 — Metrics reference lacks types, labels, semantics and alerts; no log catalogue.**
Evidence: E1, E2, M-11.
Deliverable: metrics table (name, type, labels and values, exact meaning, suggested alert); log catalogue (level, message, keys, when).
Acceptance: a test scrapes the binary and fails if any exported family or label value is missing from the table or has a different type.

**DOC-15 — EXAMPLES defects.**
Evidence: M-17, M-18, M-19, M-21, section 4 copy-paste items.
Deliverable: quoted `"SERVING"` match; a `create_if_absent` block showing `AlreadyExists`; a platform note at the top; corrected service count; `exit 1` replaced by a form safe in an interactive shell; a `## Walkthrough` parent heading.
Acceptance: assertion table extended; the health assertion fails against `NOT_SERVING`.

**DOC-16 — "Proven" claims exceed the evidence.**
Evidence: M-05, M-06.
Deliverable: reword to what is tested (three scripted bound scenarios; randomized convergence with a printed seed), state the loss-bound precondition.
Acceptance: no occurrence of "proven"/"proving" without the name of the test that backs it.

**DOC-17 — AGENTS.md is stale.**
Evidence: M-32 … M-37.
Deliverable: rewrite Layout (all packages, adapters, data flow) and Dependency policy to the shipped state; fix the Go version and CI statements.
Acceptance: a test or lint step compares the package list in AGENTS.md with `go list ./...`.

**DOC-18 — CLAUDE.md stack section is pre-implementation research, labelled as the stack.**
Evidence: M-38 … M-44, M-23.
Deliverable: regenerate from the repository (actual dependencies, tool directives, simulator design, dump format) or mark the section as historical; fill Conventions and Architecture.
Acceptance: every version in the file equals `go.mod`; no library is listed that the module does not use.

**DOC-19 — OPERATIONS.md has no machine verification.**
Evidence: section 5.3, 5.4.
Deliverable: executable or cross-checked operations doc: metric names and flags compared with the binary, exit codes exercised, defaults compared with constants.
Acceptance: renaming a metric or changing a default without editing the document turns CI red.

**DOC-20 — README quickstart duplication and rerun behaviour.**
Evidence: M-02, M-03, section 4.
Deliverable: one block parameterized by a broadcast-address variable, per-run temp directory or explicit cleanup, a macOS note that states how to find the interface and what failure looks like.
Acceptance: running the block twice in the same directory still proves replication; the test runs blocks in separate directories.

### Low

**DOC-21 — Proto comments.** Evidence: M-45 … M-47, section 4. Deliverable: remove phase/decision ids and stale sentences; add per-RPC status codes and the client-relevant behaviour. Acceptance: no match for `D-[0-9]`, `phase`, `plan 0` in either `.proto`; `buf lint` comment rules enabled.

**DOC-22 — Go comments: unreachable ids, stale tense, contradictions.** Evidence: M-48, section 4 counts. Deliverable: publish the decision records the comments cite (see DOC-23) or strip the ids; fix the stale and contradictory statements; remove "(REPL-08)" from log text. Acceptance: every id pattern in comments resolves to an anchor in a published document, checked by a script.

**DOC-23 — Maintainer documentation.** Evidence: H1–H9. Deliverable: `CONTRIBUTING.md`, `SECURITY.md`, `docs/ARCHITECTURE.md` (package map, write path, concurrency), decision log in `docs/adr/`, simulator how-to (seed replay, adding a scenario), release procedure, compatibility policy, benchmark results with hardware. Acceptance: files exist and are linked from README; the replay commands in the how-to are executed by a test.

**DOC-24 — CHANGELOG and release notes.** Evidence: section 4. Deliverable: remove phase suffixes; add "Known limitations"; release header limited to the current version's section; exclude process commits from the generated list. Acceptance: a release snapshot body contains no "Phase", no "[Unreleased]", no `docs(0N)` commit lines.

**DOC-25 — Stale or broken build and setup documentation.** Evidence: M-49 … M-52. Deliverable: remove the dead buf guard and its comments; update the dependabot comment; fix the ruleset verify command (query by id); add a `help` target. Acceptance: the documented verify command exits 0 and prints the documented shape; `make help` lists every target.

**DOC-26 — Terminology.** Evidence: section 4. Deliverable: glossary with the spec's Russian terms mapped to one English term each; one parameter spelling in prose. Acceptance: glossary linked from README; OPERATIONS uses flag spellings only.

## 7. Behaviour defects surfaced while verifying documentation

These are not documentation wording problems. Each was observed with the scratch binary or a scratch test and needs an owner on the code side; the related doc finding cannot be closed before the decision.

| # | Observed behaviour | Evidence | Doc finding |
|---|---|---|---|
| B-1 | With `--value-max-bytes` above 61440, a confirmed write is discarded from the node's own dump at the next start and cannot replicate. | Appendix B.7; `codec.go:25,139-141` | DOC-02 |
| B-2 | Fatal and shutdown errors are logged at INFO; with `SUPKV_LOG_LEVEL=warn` or `error` the process exits 1 silently. | Appendix B.1, B.5; `main.go:231,241,246,255-258` | DOC-11 |
| B-3 | `supkv_clock_drift_warnings_total` increments with synchronized clocks after any quiet period longer than `drift-warn`, and on a fresh node's first replicated write. | Appendix B.8; `clock.go:124`, `core.go:532-534` | DOC-01 |
| B-4 | Both dump generations unreadable: silent empty start (no log, no metric). Runtime dump failures are not logged and health stays `SERVING` while every write returns `Unavailable`. The spec requires a log entry and a metric for both. | Appendix B.4, B.5; `engine.go:116,202` | DOC-06 |
| B-5 | `supkv_merges_conflicts_total` counts ordinary replicated overwrites and stale redeliveries. | Appendix B.8; `core.go:531` | DOC-09 |
| B-6 | `wait_for_replication` returns `acked: true` with zero known peers. | Appendix B.6; `wait.go:101-105` | DOC-03 |
| B-7 | `SnapshotLoad` of a missing name returns `Internal` and puts the server-side path in the message; restore keeps keys that are absent from the snapshot (spec: delete them). | Appendix B.3, B.9; `server.go:309`, `engine.go:555-557` | DOC-07, DOC-08 |

## Appendix A — captured `--help` and `--version`

Binary: `go build -o <scratch>/docs-audit-work/sup-kv ./cmd/sup-kv` (Go 1.27.4, darwin/arm64). `--help` exit status 0, output on stderr.

```
Usage of <scratch>/docs-audit-work/sup-kv:
  -ack-timeout duration
    	retransmit ladder base (env SUPKV_ACK_TIMEOUT) (default 200ms)
  -ack-timeout-max duration
    	retransmit ladder ceiling (env SUPKV_ACK_TIMEOUT_MAX) (default 5s)
  -bind-addr string
    	required local UDP bind address, e.g. 0.0.0.0:7900 (env SUPKV_BIND_ADDR)
  -broadcast-addr string
    	required segment broadcast address, e.g. 127.255.255.255:7900 (env SUPKV_BROADCAST_ADDR)
  -drift-warn duration
    	clock drift warning threshold (env SUPKV_DRIFT_WARN) (default 1m0s)
  -dump-debounce duration
    	dump quiet window (env SUPKV_DUMP_DEBOUNCE) (default 50ms)
  -dump-max-wait duration
    	dump start bound from the oldest unconfirmed write (env SUPKV_DUMP_MAX_WAIT) (default 1s)
  -dump-path string
    	required dump file path (env SUPKV_DUMP_PATH)
  -grpc-addr string
    	gRPC listen address, distinct from the UDP bind (env SUPKV_GRPC_ADDR) (default "127.0.0.1:7901")
  -hello-interval duration
    	discovery broadcast period (env SUPKV_HELLO_INTERVAL) (default 1s)
  -metrics-addr string
    	optional metrics/debug listen address — unset means no listener (env SUPKV_METRICS_ADDR)
  -node-id uint
    	required unique 16-bit node ID (env SUPKV_NODE_ID)
  -peer-timeout duration
    	peer eviction silence bound (env SUPKV_PEER_TIMEOUT) (default 5s)
  -retransmit-interval duration
    	retransmit scan period (env SUPKV_RETRANSMIT_INTERVAL) (default 100ms)
  -startup-gate duration
    	gate backstop: writable this long after start even without a sync round (env SUPKV_STARTUP_GATE) (default 1s)
  -sync-chunk-bytes int
    	DataSync chunk byte budget (env SUPKV_SYNC_CHUNK_BYTES) (default 1400)
  -sync-interval duration
    	proactive sync listing period (env SUPKV_SYNC_INTERVAL) (default 30s)
  -sync-response-rate int
    	sync responses per second cap (env SUPKV_SYNC_RESPONSE_RATE) (default 500)
  -value-max-bytes int
    	maximum value size in bytes (env SUPKV_VALUE_MAX_BYTES) (default 61440)
  -version
    	print version, commit and Go runtime version, then exit
  -wait-timeout duration
    	default wait_for_replication bound (env SUPKV_WAIT_TIMEOUT) (default 1s)
```

`--version` (exit status 0, stdout):

```
sup-kv dev dev go1.27.4
```

Env-only knobs absent from `--help` (`main.go:114-126`): `SUPKV_LOG_LEVEL`, `SUPKV_LOG_FORMAT`, `SUPKV_METRICS_CERT`, `SUPKV_METRICS_KEY`, `SUPKV_METRICS_CLIENT_CA`.

## Appendix B — experiments

All runs used the scratch binary, ports UDP 39101/39102, gRPC 127.0.0.1:39111/39112, metrics 127.0.0.1:39121/39122, working directory under the scratchpad, grpcurl v1.9.4 built from the repo's tool directive into scratch. Host: macOS 27.2, arm64.

**B.1 Startup errors**

| Invocation | Exit | Output |
|---|---|---|
| no arguments | 2 | `sup-kv: missing required knobs: --node-id, --bind-addr, --broadcast-addr, --dump-path` + usage (plain text) |
| `--nope` | 2 | `flag provided but not defined: -nope` |
| `--ack-timeout banana` | 2 | `invalid value "banana" for flag -ack-timeout: parse error` |
| `SUPKV_ACK_TIMEOUT=banana` | 2 | `sup-kv: SUPKV_ACK_TIMEOUT: must be a duration: …` |
| `--node-id 0` | 2 | `sup-kv: missing required knobs: --node-id` |
| `--node-id 70000` | 2 | `sup-kv: --node-id 70000: node-id exceeds the 16-bit node ID space` |
| `--bind-addr not-an-address` | 1 | `{"level":"INFO","msg":"sup-kv: bind replication socket: udprepl: bind udp4 not-an-address: …"}` |
| `--broadcast-addr nonsense` | 1 | `{"level":"INFO","msg":"sup-kv: bind replication socket: udprepl: resolve broadcast nonsense: …"}` |
| `--grpc-addr nonsense` | 1 | `{"level":"INFO","msg":"sup-kv: listen tcp nonsense: …"}` |
| same, `SUPKV_LOG_LEVEL=error` | 1 | 0 bytes |
| `--value-max-bytes -5 --ack-timeout -1s --sync-response-rate 0 --ack-timeout-max 1ms` | runs | no message |

**B.2 Healthy single node, `SUPKV_LOG_LEVEL=debug`:** stderr stayed empty for the whole run (start, writes, SIGTERM). SIGTERM → exit 0. One `Set` ≈ 0.07 s wall, one `Get` ≈ 0.01 s (three samples each, native grpcurl).

**B.3 API status codes**

| Call | Result |
|---|---|
| `Get` with empty key | `InvalidArgument: key must not be empty` |
| `Set` 61440 bytes / 61441 bytes | OK / `InvalidArgument: value exceeds the maximum allowed size` |
| `Set create_if_absent` on a live key | `AlreadyExists: key already exists` |
| `Set create_if_absent` on a deleted key | `AlreadyExists: key already exists` |
| `Delete` twice | version, then `NotFound: key not found` |
| `Set` without value, then `Get` | OK; record returned with no `value` field |
| `SnapshotCreate {"name":"../x"}` | `InvalidArgument: invalid snapshot name` |
| `SnapshotLoad {"name":"nope"}` | `Internal: internal error: open n1/state.json.snapshots/nope: no such file or directory` |
| `Watch`, then SIGTERM to the server | stream ends, grpcurl exit 0 |

Dump file after these calls (mode 0600; `state.json.prev` 0600; `state.json.snapshots/` 0700):

```
{"format_version":1}
{"key":"a","record_version":"01a1f08421e870008001c00000000000","data":"YQ==","deleted":false,"checksum":2857936995}
{"key":"svc/v1/a","record_version":"01a1f0842e2870008001c00000000000","data":null,"deleted":true,"checksum":2581624298}
{"key":"svc/v1/b","record_version":"01a1f0842c6f70008001c00000000000","data":"Yg==","deleted":false,"checksum":1582491917}
{"key":"svc/v1/empty","record_version":"01a1f084317b70008001c00000000000","data":null,"deleted":false,"checksum":618995932}
```

`/metrics` on that node (peer absent): `supkv_replication_peers 0`, `supkv_replication_unacked_records 5`, `supkv_replication_unacked_oldest_age_seconds 24.99…`, `supkv_replication_datagrams_sent_total{type="hello"|"data"|"ack"|"data_request"|"sync"}`, `supkv_datagrams_discarded_total{class=…}` with the five documented values; runtime families exactly the five in M-11.

**B.4 Corrupted dumps**

- `state.json` = `garbage not json`, `state.json.prev` = `{"format_version":99}`: node starts, `List` → `{}`, stderr empty at debug level, `supkv_dumps_*` all 0, both files left untouched, SIGTERM → exit 0.
- Valid dump with one record's checksum changed: node starts with the other record; one line `{"level":"INFO","msg":"sup-kv: discarded 1 undecodable dumped records at startup"}`.

**B.5 Dump failure** (dump directory made read-only after the first write)

- `Set` → `Unavailable: dump failed; write not confirmed` (≈ 0.06 s).
- `Get` of the same key → value returned.
- `Set create_if_absent` of the same key → `AlreadyExists`.
- After 0.5 s: `supkv_dumps_total 12`, `supkv_dumps_errors_total 11`. Health `SERVING`. stderr empty.
- SIGTERM → exit 1, `{"level":"INFO","msg":"sup-kv: final dump failed: open …/dump-4242509444: permission denied"}`. With `SUPKV_LOG_LEVEL=warn`: exit 1, 0 bytes.

**B.6 Loopback pair on macOS** (the documented demo topology, `127.255.255.255`): after 3.5 s `supkv_replication_peers 0` on both; `Set … wait_for_replication:true, wait_timeout_ms:2000` on A → `{"version":"…","acked":true}`; `Get` on B → `NotFound`; stderr empty on both at debug level. `ipconfig getoption en0 broadcast` → no output, exit 1.

**B.7 `--value-max-bytes 200000`:** 100000-byte `Set` → OK; `Get` returns it; stderr `{"level":"WARN","msg":"Replication adapter","detail":"udprepl: send to 127.255.255.255:39102: … sendto: message too long"}`; SIGTERM → exit 0, dump 133476 bytes. Restart with the same flags: `Get` → `NotFound`; stderr `discarded 1 undecodable dumped records at startup`.

**B.8 Counter semantics** (scratch copy of the repo; `persist.Open` + `repl.New` with an injected clock, datagrams fed through `Core.HandleDatagram`; peer stamps equal to the local wall clock):

| Step | `MergeStats` after |
|---|---|
| fresh node, first replicated write | `{Conflicts:0 DriftWarnings:1}` |
| 1 s later, peer updates the same key | `{Conflicts:1 DriftWarnings:1}` |
| 1 h later, peer updates the key | `{Conflicts:2 DriftWarnings:2}` |
| 1 h later, peer creates a new key | `{Conflicts:2 DriftWarnings:3}` |

Direct `storage.Merge` returns: fresh node `drift=-500000h`; after 2 min idle `drift=-2m0s`; stale redelivery `conflicts=1`; record stamped 10 min ahead `drift=0s` on absorption and `drift=10m0s` on the next merge; restart from a 3-day-old watermark `drift=-72h`.

**B.9 Snapshot semantics:** `cfg/a=v1`, `cfg/gone` set; `SnapshotCreate before`; then `cfg/a=v2`, `cfg/new` created, `cfg/gone` deleted; `SnapshotLoad before` → `List` = `cfg/a`, `cfg/gone`, `cfg/new`; `cfg/a` = v1. A second `SnapshotCreate before` overwrote the file without error.

**B.10 Repository and release state:** `gh release view v1.0.0` → published 2026-11-17T17:41:59Z, 9 assets; archive listing as in M-04; Linux binary "statically linked … with debug_info, not stripped"; documented ruleset verify command → `cannot iterate over: null`, exit 1; `make help` → no rule.

## Appendix C — as-built deviations from the upstream spec observed during the audit

Input for the deviation table of DOC-12. Items marked (measured) were observed with the binary; the others are from reading the cited code. A full spec trace is out of scope for this audit.

| # | Spec | As built | Evidence |
|---|---|---|---|
| 1 | Conflict journal records the losing version | No journal; one counter | `core.go:214-215`; M-23 |
| 2 | `create if absent` proceeds when the key is deleted | `AlreadyExists` on a tombstone (measured) | `store.go` `SetIfAbsent`; B.3 |
| 3 | Snapshot load deletes keys absent from the snapshot | Such keys are kept (measured) | `engine.go:555-557`; B.9 |
| 4 | `DataSync` carries `SyncID`, `ChunkIndex`, `ChunkCount` | `entries` only | `supkv.proto:66-68` |
| 5 | `RunID` is a UUID v7 | 16 random bytes | `core.go:189-199` |
| 6 | `rand_b` remainder is random | zero | `version.go:21-22` |
| 7 | Dump is a JSON file with `Key`, `RecordVersion`, … | JSON Lines, snake_case fields, hex version (measured) | `codec.go:39-54`; B.3 |
| 8 | `dump_debounce` = minimum pause between dumps | quiet window that restarts on every change, bounded by the added `dump-max-wait` | `engine.go:165-170` |
| 9 | Receiver with a newer version replies with its record and sends no Ack | Receiver Acks and sends nothing | `core.go:523-543` |
| 10 | HLC `Update` on every `DataSync` chunk | not called | `core.go:631-644` |
| 11 | Unreadable dumps, dump failure and clock drift are logged | none of the three is logged (measured for the first two) | B.4, B.5; no log call sites |
| 12 | Gate opens after the first full sync is processed | opens on the first `DataSync` datagram received | `core.go:617-621` |
| 13 | Peer table has one entry | N peers; aggregate `wait_for_replication` report | `core.go:288` |
| 14 | Restart-triggered sync is broadcast | unicast to the restarted peer | `core.go:490-494` |
| 15 | — | Additions: `List`, health, reflection, `wait_timeout_ms`, `--wait-timeout`, `--grpc-addr`, metrics endpoint, empty key rejected | `supkvapi.proto`, `main.go` |
