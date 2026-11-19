# sup-kv — test-suite evidential-strength audit

Audit date: 2026-11-19. Repository state: `master` @ `d6b2a75` (working tree clean), audited from a scratch copy without `.git`.
Question asked: do the tests *prove* the properties the project claims, or do they merely execute code?

## Summary

**Verdict.** The suite proves the algorithmic core and little beyond it. The LWW merge, the HLC, the retransmission ladder, peer eviction, sync chunking and the response limiter are pinned by exact, mutation-resistant tests, and the simulator that exercises them is genuinely deterministic and drives the production decision core. The properties the product is sold on — a confirmed write is on disk, deletes replicate, a restarted node cannot lose a write to an unseen version, the documented bounds hold on a real network — rest on tests that pass when the behaviour is removed, or on no test.

| | |
|---|---|
| Suite, single run | passes on darwin/arm64 (160 pass, 13 skip) and linux/arm64 (173 pass, 0 skip); `go vet` clean |
| Flakiness | `TestChaos_convergence` fails for 3.74 % of seeds (374 / 10 000) and turned the `master` CI run for `6458adb` (2026-11-18) red; port-allocation and signal-ordering flakes in `cmd/sup-kv` / `internal/udprepl` hit at least 5 of 73 Linux suite runs |
| Mutation score | 57 / 73 killed on Linux, 54 / 73 on darwin; 16 survive everywhere |
| Survivors | M18 restart ignores dump maximum · M21 snapshot before waiter capture · M26 no data fsync · M27 no directory fsync · M31 non-atomic dump write · M38 `DataSync` omits tombstones · M53 `Deleted` flag dropped on the wire · M54, M55 decoder domain checks removed · M61 delete announced as a live value · M64 Watch cut filter removed · M66, M68, M69, M70 configuration knobs unwired · M71 stale `DataReplication` not Acked · darwin only: M60, M65, M67 |
| Scenario matrix (18 failure rows) | Proven without qualification: 12, 16. Proven within a stated limit: 1 (≤ 4 consecutive losses), 9 (toy scale). Mixed — some element Proven, another Weak or Missing: 2, 3, 5, 7, 11, 14, 17. Weak: 4, 6, 10, 13, 15, 18. Row 8: Weak, tombstone element Missing. The Missing elements that matter: stale-version reply (3), tombstones after downtime (8), crash of both nodes and kill-after-`OK` (10), the gate's purpose (7), socket-buffer overflow (15) |
| Benchmarks | dump at 10 k keys ≈ 17–21 ms (5.1 MB); "~975 w/s" not reproduced (420–443 — the benchmark reports its pacing ticker) and the burst benchmark fails at ≥ 10 000 keys; "~10 ms confirmation latency" reproduces numerically but measures a fake disk, a 1 ms debounce and a poll interval |
| Not done | 13 prepared mutants (M74–M86), fuzz harness runs, a `-shuffle=on` pass, linux/amd64, GitHub runners |

## 0. Method, and what was run versus only read

| What | How | Status |
|---|---|---|
| Scratch copy | `rsync -a --exclude .git` to `evidence/repro/tests/sup-kv/`; every run and every edit happened there; the tree was verified byte-identical to the repository (`diff -rq`) before and after the mutation batches | ran |
| Host platform | darwin/arm64, Apple M3 Max (16 cores), `go1.27.4` | ran |
| Linux platform | `golang:1.27-bookworm` container (linux/arm64, `go1.27.4`, OrbStack VM, kernel 7.2), **non-root** user, `GOPROXY=off` against the host module cache, scratch copy bind-mounted. Used because 13 top-level tests skip on darwin (section 2). The image was pulled for this audit. This is *not* GitHub's `ubuntu-latest`/amd64 runner: same OS family and Go version, different CPU arch, timer granularity and filesystem (section 10) | ran |
| Baseline, flakiness, coverage, benchmarks | commands quoted in each section | ran |
| Mutation probes | 73 single-edit mutants of production code, one at a time (13 more were prepared and **not run** — listed in section 8.5), whole suite on **both** platforms (`go test -count=1 -p 1 -timeout 150s ./...`, `-race` off, `SUPKV_SIM_SEED=1` pinned so the known chaos flake cannot fake a kill), every failure re-run once and counted as a kill only if it reproduced | ran |
| Extra probes | six throw-away test files written in a separate probe copy (never in the repository): chaos seed sweeps, empty-disk recovery at scale, startup-gate vs HLC, chaos with deletes and skew, max-size datagrams, a containment stress test | ran |
| Fuzz harnesses, `-shuffle=on` | two fuzz harnesses (datagram decoder + core classifier; dump decoder) were written and vetted but **not run**; a `-shuffle=on` pass was planned and **not run** | not done |
| CI history | `gh run list/view` on the public repository (read-only) to check whether the flake found locally has already fired in CI | ran |
| Production code and tests | `internal/app/{repl,storage,clock,version,persist}`, `internal/persistfile`, `internal/udprepl`, `internal/grpcapi`, `internal/metrics`, `cmd/sup-kv`, simulator and all `_test.go` files | read (core, store, clock, version, engine, codec, dump file adapter, UDP adapter, simulator and the simulator test files line by line; the remaining test files by doc comment, assertions and targeted reads) |
| Upstream spec | `SPEC.md` sections [spec section], [spec section], [spec section], [spec section], [spec section], [spec section], [spec section] | read |

The repository gained a `.planning/reviews/` directory while the audit ran (another review track's output); it is not part of the audited snapshot and no source file differs.

Not done: no run on real GitHub runners, no run on linux/amd64, no power-loss or real-NIC fault injection, no multi-host broadcast segment. Nothing needed `.git` metadata (the whole suite passes in a copy without it). Network: the doc-contract tests shell out to `go tool grpcurl`; that resolved offline here only because the module was already in the host module cache.

## 1. Baseline

`go vet ./...` — exit 0, no diagnostics, on both platforms.

`go test -race -cover ./...` on darwin/arm64 (the `make test` / CI command plus `-cover`) — **pass**, 15.3 s wall:

| Package | Result | Time | Statement coverage (own package) |
|---|---|---|---|
| `cmd/sup-kv` | ok | 3.48 s | 0.0 % (see note) |
| `internal/app/clock` | ok | 1.46 s | 100.0 % |
| `internal/app/internal/simnet` | ok | 1.36 s | 74.0 % |
| `internal/app/persist` | ok | 2.21 s | 97.7 % |
| `internal/app/repl` | ok | 2.65 s | 97.8 % |
| `internal/app/storage` | ok | 1.98 s | 100.0 % |
| `internal/app/version` | ok | 1.30 s | 100.0 % |
| `internal/grpcapi` | ok | 2.54 s | 90.2 % |
| `internal/metrics` | ok | 1.47 s | 67.0 % |
| `internal/persistfile` | ok | 2.34 s | 87.6 % |
| `internal/udprepl` | ok | 1.64 s | 83.9 % |
| `internal/port`, `testutil`, generated `*.pb.go` | no test files | — | — |

Note on `cmd/sup-kv` 0.0 %: every test there drives a separately built binary (`TestMain` runs `go build -o … .`), so `main.go` (797 lines) has no in-process coverage and — because that inner build has no `-race` — **is never exercised under the race detector**, not even in CI. With `GOFLAGS=-cover -coverpkg=…` the child binary can be measured (done on Linux): `main` 85 %, `wire` 78 %, `shutdown` 79 %, `abortWire` 0 %.

Test counts (`go test -race -count=1 -p 1 -v ./...`): 173 top-level tests, 361 including subtests.

| Platform | Top-level pass | Top-level skip | Subtest skip | Fail |
|---|---|---|---|---|
| darwin/arm64 (this machine; same OS/arch as the `macos-15` CI cell) | 160 | **13** | 3 | 0 |
| linux/arm64 container, non-root | 173 | 0 | 0 | 0 |
| linux/arm64 container, root | 172 | 1 (`TestFileStore_saveFailureHygiene`, root ignores directory modes) | 0 | 0 |

One green run on each platform is not the whole story: section 3 shows the suite is flaky.

## 2. Skip inventory — what is not proven on which platform

There are no build tags on test files and no `testing.Short` gates. Skips are decided at run time:

| Gate | Tests | darwin/arm64 (and the `macos-15` CI cell) | Linux CI |
|---|---|---|---|
| Loopback-broadcast capability probe: bind, send one datagram to `127.255.255.255:<port>`, wait **300 ms**; no datagram ⇒ `t.Skipf` | `cmd/sup-kv`: `TestExec_killNineRestart`, `TestExec_bothNodesDown`, `TestExec_sigtermFinalDump`, `TestExec_emptyDiskRecovery`, `TestLog_jsonDefault`, `TestLog_formatText`; `internal/udprepl`: `TestScenario_convergence`, `TestScenario_discardClasses/{bad_crc,bad_proto_version,duplicate_node_id}`, `TestMirror_unackedSent`, `BenchmarkReplicationBurst`; `internal/grpcapi`: `TestWaitIntegration_ackedThroughRealAdapterMirror`, `BenchmarkConfirmationLatency` | **always skip** (lo0 has no BROADCAST flag) | run; but a probe that merely *times out* on a loaded runner is indistinguishable from "platform lacks the capability" and also skips — green without running |
| Second wildcard bind on one port (`SO_REUSEADDR` only) | `internal/udprepl` `TestSmoke_twoNodesUDP` | skip (darwin needs `SO_REUSEPORT`) | runs |
| Hard-coded `runtime.GOOS == "darwin"` skip ("option-b decision") | `TestReadmeQuickstart`, `TestExamplesDoc`, `TestProbeQuickstartBroadcast` | skip | run |
| `os.Geteuid() == 0` | `internal/persistfile` `TestFileStore_saveFailureHygiene` | runs | runs (runners are non-root); skips in a root container |
| `exec.LookPath("go")` / `go build` failure in `TestMain` | every `cmd/sup-kv` test that needs the binary | runs | runs |
| `SUPKV_SIM_SEED` (env), `-seed` (flag) | not skips — seed replay hooks for the simulator and the property tests | honoured | honoured |

Consequences:

* **On darwin nothing above the in-process core is proven.** No test on darwin ever moves a datagram between two nodes over a socket, starts two processes, kills a process, checks the final dump after SIGTERM against a replicated write, or executes a documented command. The `macos-15` gate cell therefore certifies unit and simulator behaviour only; the `darwin-amd64-cross` cell compiles and vets, nothing more. darwin is a release target (`.goreleaser.yaml`, 4 targets) and `README.md` carries a "Quickstart on macOS" block, which CI executes **on Linux**, through its `|| echo 127.255.255.255` fallback arm — it has never been executed on macOS.
* **On Linux the same tests can silently stop running.** The only canary that fails instead of skipping is `TestProbeQuickstartBroadcast` (it `t.Fatalf`s when loopback broadcast does not deliver). The eleven tests behind the per-test 300 ms probe do not fail when their own probe misses.
* The real-socket tests that do run on darwin (`TestWire_twoNodeHandoff`, `TestDiscardLog_classes`, `TestMirror_*` except `unackedSent`, `TestNotifyLocalWrite_*`) are single-adapter or hand-driven and never exercise broadcast discovery or a unicast Ack between two sockets.

## 3. Flakiness

Commands: simulator/core packages `go test -race -count=5`; socket/exec packages `go test -race -count=3 -p 1`, plus a `GOMAXPROCS=2` pass; fixed seeds `SUPKV_SIM_SEED ∈ {1, 2, 3, 42, 4242, 20261119, -1, max int64, min int64}`; all on both platforms where applicable. `SUPKV_SIM_SEED` is honoured by the nine `TestScenario_*` tests and `TestChaos_convergence` (not by the `TestBound_*` tests or `TestChaos_determinismReplay`, which hard-code their seeds).

### 3.1 `TestChaos_convergence` fails for about 3.7 % of seeds — and CI draws a fresh wall-clock seed every run

* All nine fixed seeds above pass. darwin `-count=5` passed. Linux `-count=5` **failed once**: `sim_chaos_test.go:250: node 1 holds 21 records, the pointwise-supremum model holds 21 — divergence after quiescence replay: SUPKV_SIM_SEED=1795037024500600012`.
* Seed sweep with the compiled test binary, seeds 1…600, tests `TestChaos_convergence|TestScenario_`: **23 failures, all in `TestChaos_convergence`**, all "divergence after quiescence" (seeds 27, 69, 85, 104, 136, 160, 169, 186, 264, 276, 279, 340, 355, 356, 376, 383, 448, 457, 464, 492, 573, 578, 579). The nine `TestScenario_*` tests passed for all 600 seeds.
* Probe over 10 000 seeds (1…5000 and 5000 nanosecond-shaped seeds): the assertion fails for **374 seeds (3.74 %)**; never with 2 nodes, 8.2 % of 3-node scenarios, 4.4 % of 4-node, 2.5 % of 5-node.
* **It has already turned `master` red.** CI run 44351136063 (push of `6458adb`, the v1.0 milestone-close commit, 2026-11-18), cell `gate (ubuntu-latest)`: `--- FAIL: TestChaos_convergence … node 1 holds 23 records, the pointwise-supremum model holds 24 — divergence after quiescence replay: SUPKV_SIM_SEED=1795020540436451818`. That seed reproduces locally with the identical message, so the replay hook works across machines and architectures.
* Root cause (probe, same scenario driven past the first quiescent instant): the divergence is **transient in every one of the 374 cases**; all 10 000 seeds converge to the pointwise-supremum model. The test asserts convergence at the *first* instant its "three-seam quiescence" holds (no pending deliveries, no unacked entries, no queued responses). With three or more nodes that predicate does not imply convergence: a peer evicted by `peer_timeout` during a partition leaves the writer's wait-set, the remaining peers' Acks drain the entry, and the returning peer (not a restart, so no targeted listing; the table is not empty, so no discovery flush) learns the record only from the next periodic `DataSync`. The cluster is "quiescent" and unconverged until that round. In the CI-failing seed convergence came 10.6 s after the last write, at the next 30 s sync tick.
* Expected CI impact: one `go test` run fails with p ≈ 0.037; a pipeline has three full-gate cells, so about 1 − 0.963³ ≈ **11 % of pipelines are red for this reason alone**, on pull requests and on pushes to `master`.

So the flagship "convergence under loss, duplication, reordering and partition" test is simultaneously the least reliable test in the suite and (section 5) weaker than its name.

### 3.2 `cmd/sup-kv`: three more flake classes

| Symptom | Where seen | Cause |
|---|---|---|
| `main_test.go:714/762: probe rx bind: bind udp4 0.0.0.0:<port>: … address already in use` in `TestLog_jsonDefault` / `TestLog_formatText`; the same bind error from `udprepl.New` in `TestMirror_unackedSent`; `TestShutdown_secondSignal` hanging to the package timeout (its node lost its port, and the test's `Watch(…, WaitForReady(true))` at `main_test.go:516` has no deadline) | darwin `-count=3` (1 of 3 iterations); Linux `-count=3` second pass; in the mutation runs (non-race, `-p 1`, 16 cores): 5 of 73 Linux suite runs and 1 of 73 darwin runs, none related to the mutant under test | `freeUDPPort` binds `:0`, closes, returns the number; a parallel test or child process takes the port before `probeLoopbackBroadcast` re-binds it, and the bind error is a `t.Fatalf`. The exec scenarios were moved to a private port window for exactly this reason (`scenario_test.go:93-108`); the two log tests and the whole `internal/udprepl` / `internal/grpcapi` real-socket family still use the racy picker |
| `main_test.go:769: process exit code = -1 (signal: terminated), want 0` in `TestLog_formatText` | Linux `-count=3` first pass | The test sends SIGTERM as soon as node A logs the driven event. That log line is produced by the adapter goroutine started inside `wire()`, while `main()` installs its signal handler only afterwards (`main.go:229` then `:238`). A SIGTERM in that window kills the process with the default disposition. It is a test race and also a small production window in which SIGTERM skips the D-19 shutdown |
| `TestReadmeQuickstart` hung until the 150 s package timeout | Linux, once, while this audit was running a CPU-heavy probe in parallel (so: under heavy contention only) | The README block's bounded-retry loops multiply to minutes when each `go tool grpcurl` call is slow; the test has no deadline of its own |

No flake was observed in `internal/udprepl`, `internal/grpcapi`, `internal/app/persist`, `internal/persistfile` or `internal/metrics` in these runs, including the `GOMAXPROCS=2` pass.

## 4. Coverage — the uncovered branches that matter

Profiles: darwin in-process, Linux in-process, and Linux in-process merged with the exec'd binary (`GOFLAGS="-cover -coverpkg=…"`). Totals: 73.6 % / 74.1 % in-process over all packages (generated code and `main.go` drag the figure down; it is not the interesting number). 133 blocks are covered on Linux but not on darwin — 106 of them in `main.go`, the rest in `udprepl`, `metrics`, `grpcapi/watch.go`.

Uncovered on **both** platforms even with the exec'd binary merged in (file:line in the repository):

| Area | Uncovered | Why it matters |
|---|---|---|
| Dump write error paths, `internal/persistfile/dumpfile.go` | 101-102 (`MkdirAll` error), 109-111 (rename failed → temp removed), 129-132 (write/fsync/close failed → temp removed), 141-142 (`Write` error — the ENOSPC path), 153-154 (rotation rename error), 172-173 (directory open/fsync error) | The spec's "dump failed → `Unavailable`" starts here. Only the first failure of the chain (`CreateTemp` in a read-only directory) is ever produced. A failing fsync, a full disk and a failing rename are untested |
| Exit code on a failed final dump, `cmd/sup-kv/main.go` | 246-248 and 678-679 (`final dump failed` → exit 1) | `scenario_test.go:569` states this negative "is already pinned"; it is not executed by any test |
| Shutdown and startup failure paths, `main.go` | 668-670 (graceful stop exceeds `shutdownGrace` → forced stop), 442-444 / 467-469 / 475-477 / 497-502 (`abortWire`: UDP bind, gRPC listen or metrics listen failure), 241-242 (serve error) | Port-in-use at startup and a hung drain are the operator-visible failure modes |
| Config parsing, `main.go` | 733-737, 747-751, 761-765 — the bodies of `envUint16`, `envInt`, `envDuration` for a **non-empty** value; 386-391, 694-704 — node-id range and missing-required errors | No test sets a numeric, duration or node-id environment variable, and none starts the binary with a required knob missing |
| Datagram decoder, `internal/udprepl/repl.go` | 735-739 (empty / unparsable envelope, node_id beyond 16 bits, run_id not 16 bytes), 752-757 (body fails to unmarshal, unknown body type), 805-806 / 819-820 / 830-831 / 846-847 (record version not 16 bytes, in DataReplication, Ack and SyncEntry) | **Every `Undecodable` branch of the real decoder.** The class is tested only by handing the core a synthetic `port.Undecodable` status. The trust-boundary parser has never been fed a malformed datagram |
| Send path, `internal/udprepl/repl.go` | 519-520 (encode error), 529-530 (unresolvable unicast handle), 558-559 (`WriteToUDP` error) | A send failure (ENOBUFS, `message too long`, interface down) is the "socket buffer overflow / network fault" class of the spec. See probe 5 in section 9: an oversize datagram hits line 558 forever |
| Adapter metric accessors, `internal/udprepl/repl.go` | 417 `MergeStats`, 427 `Retransmissions` (0 % in-process; reached only through `/metrics` scrapes of the child binary, whose values are not asserted) | The conflict, drift and retransmission counters are never observed beyond the core |
| Wait-for-replication, `internal/grpcapi/wait.go` | 126, 191 (context cancelled during the wait), 166 (entry vanished from the mirror) | Client cancellation and the "superseded between polls" report |
| Watch, `internal/grpcapi/watch.go` | 298 (per-subscriber channel full → disconnect), 375-377 / 394-395 / 406-408 (`stream.Send` errors), 230 / 239 / 370-371 (hub shutting down) | `TestWatch_slowConsumer` reaches the shed status only through the hub-ingress overflow path; the per-subscriber shed branch is never executed |
| Engine, `internal/app/persist/engine.go` | 671-681 (flush racing Run exit — partly), 585-586 (restore replay error) | Shutdown races |
| Error mapping, `internal/grpcapi/server.go` | 306-309 (`mapError` fall-through to `Internal`) | Unknown errors |

There is no "rate-limit overflow" branch to cover: the sync egress queue (`core.go` `egress`) is unbounded and has no dedup, and no test examines its growth (it reached 19 997 entries for 10 000 keys in probe 2).

## 5. Simulator fidelity (`internal/app/internal/simnet` and how `repl` tests drive it)

### 5.1 Determinism — sound

* `Scheduler` is a discrete-event queue ordered by `(virtual time, insertion sequence)` (`simnet.go:56-61`); no goroutines, no locks, no wall clock, no sleeps. All time values are built from `time.UnixMilli`.
* Every random decision is drawn from one injected `math/rand/v2` PCG at **send** time in a fixed order (drop, duplicate, per-delivery delay); fan-out iterates a slice, never a map.
* The production core never lets map order reach an emission path (sorted keys in `scanRetransmit`, `listingChunks`, the discovery flush).
* Evidence, ran: `TestSimnet_determinism` and `TestChaos_determinismReplay` (same seed, two fresh clusters, byte-identical trace digest — one seed each); my 600-seed sweep and my separately compiled probe binary failed on exactly the same 23 seeds; the seed from the red CI run on linux/amd64 reproduced on darwin/arm64 with the identical message. Determinism holds across processes, architectures and operating systems.
* Seed handling: `SUPKV_SIM_SEED` or a wall-clock draw; the replay command is logged on every run and repeated in every failure message. Replayable — verified.

### 5.2 What it drives

| Component | In the simulator | In production |
|---|---|---|
| Replication decisions | **production** `repl.Core` (`HandleDatagram`, `Advance`, `NotifyLocalWrite`) — no simulation mode | same |
| Store, LWW merge, HLC | **production** `storage.Store` and `clock.Clock`; the clock reads virtual time plus a per-node offset | same |
| `StateStore` seam | test adapter `nodeStore` → `store.Merge` (`simnet.go:330-338`) | `persist.Engine.ApplyRemote` (dirty tracking, dump scheduling) |
| Local write path | `Cluster.Write` = `store.Set` then **immediately** `NotifyLocalWrite` (`simnet.go:569-579`) | gRPC handler: `engine.Set` blocks for the dump (≥ `dump_debounce`), **then** `NotifyLocalWrite`; on a failed dump the write stays in memory and is **not** announced at all (`server.go:142-150`) |
| Wire | Go structs passed by value; every delivery is `Status: Decoded`, `ProtoVersion: Supported` | protobuf envelope, CRC32, 65 507-byte limit, real sockets |
| Driver | `Cluster.drive` loop, one timer per node | `udprepl.Adapter.Run`: read goroutine, 64-slot inbound queue, 1024-slot notify queue, atomics mirror, drained-row retention |

### 5.3 What it models, and what it does not

| Fault / condition | Modelled? | Used by |
|---|---|---|
| Datagram loss | yes (`DropProb`, and deterministic loss windows through the partition predicate) | chaos, `TestScenario_loss`, `TestBound_lossWithinAckTimeoutMax` |
| Duplication | yes (two deliveries, independent delays) | chaos, `TestScenario_loss` phase 2 |
| Reordering | yes, as a by-product of random delay (1–10 ms in chaos, 1–40 ms in the loss scenario) | chaos |
| Delay | yes; never longer than 40 ms, so "a datagram from before the partition arrives after it" exists only as one hand-injected datagram in `TestScenario_restart` | — |
| Symmetric partition | yes (predicate on `(from, to, now)`) | chaos, bound and scenario tests |
| One-way link | yes (the predicate is directed) | `TestScenario_oneWay` only; chaos partitions are always symmetric |
| Per-node clock offset, live jumps | yes (`SetClockOffset`) | `TestScenario_clockJumps`, `TestScenario_farAhead` only; **chaos never skews a clock** |
| Node replaced with an **empty** store, new run id | yes (`ReplaceNode`) | `TestScenario_restart`, second half of `TestScenario_clockJumps` |
| Brand-new node joining | yes (`AddNode`) | `TestScenario_emptyDisk` |
| **Restart that keeps its state** (the normal restart: dump on disk, HLC seeded from the dump maximum) | **no** | — |
| **Deletes / tombstones** | **no** — `Cluster.Write` only calls `Set`; no simulator test, chaos included, ever deletes a key. The chaos comment "tombstones included in the comparison" is vacuous | — |
| Persistence (dump, debounce, confirmation, dump failure, `.prev`) | **no** | — |
| Codec, CRC, proto version, malformed datagrams | **no** (always `Decoded`) | — |
| Datagram size limits, MTU, fragmentation, value sizes (payloads are ≤ 50 bytes) | **no** | — |
| Socket buffer overflow, inbound-queue back-pressure, per-datagram CPU cost | **no** (unbounded instantaneous queues) | — |
| Duplicate `node_id` | **no** (ids are always 1…n) | — |
| gRPC surface: startup gate, `wait_for_replication`, Watch | **no** | — |

Probes run in the probe copy to see whether the unmodelled-but-cheap dimensions hide defects:

* **Deletes and skew (probe 4):** the chaos scenario with every third operation a `Delete` and per-node offsets of ±30 s (10 % of nodes 1–5 h ahead), 1 500 seeds, 7 722 deletes issued, checked 20 × `sync_interval` after the last write: **0 non-converged**. The property appears to hold; it is simply not in the suite.
* **Scale (probe 2):** empty-disk join at 10 000 keys converges at 39.9 s of virtual time, inside the scenario's own bound formula (70 s). Wall cost grows quadratically (6.5 s for 16-byte values, 27.4 s for 1 KiB values) because `handleDataRequest` and `handleDataSync` copy the entire store per datagram (`core.go:606`, `:633`) — invisible to a simulator that does not model CPU time.

### 5.4 Are the bounds asserted, with the documented numbers, on the virtual clock?

Yes for three single-fault settings, on a **loss-free network with a fixed 1 ms one-way delay**:

| Documented bound | Test | Assertion | Strength |
|---|---|---|---|
| Normal operation: one RTT | `TestBound_normalOneRTT` (`sim_bound_test.go:128`) | absent on B at t0; byte-equal on B at exactly t0 + 2 ms | tight, has teeth |
| Datagram loss ≤ `ack_timeout_max` | `TestBound_lossWithinAckTimeoutMax` (`:161`) | three consecutive drops; absent at +600 ms and +1400 ms; present at +5 s (it actually arrives at +1401 ms) | proven for episodes of at most four consecutive losses; with five the next attempt is at +6.2 s, outside the bound, and the test's doc comment says so |
| Partition heal ≤ 2 × `sync_interval` + response time | `TestBound_partitionHeal` (`:207`), `TestScenario_partitionSeconds` (`sim_scenario_test.go:266`) | stores differ at the heal instant; equal at heal + 60 s + (K−1) × 2 ms | **not discriminating**: the partition (2–3 s) is shorter than `peer_timeout`, so the retransmission ladder heals it about 2 s after the heal — 25 × inside the bound. Both tests still pass with periodic `DataSync` disabled (mutant M73, section 8). The path the bound was written for — peer evicted, three or more nodes, heal by the periodic listing — is never timed |
| Empty-disk recovery ∝ keys / `sync_response_rate` | `TestScenario_emptyDisk` (`:797`) | N = 50, bound `sync_interval + (2N+2) × 2 ms` | tight, but at toy scale |

Under random faults the suite asserts no bound at all: `TestChaos_convergence` only requires quiescence within 10 × `sync_interval` (300 s) of the last write, and `TestScenario_partitionDays`, `_oneWay`, `_clockJumps`, `_farAhead`, `_restart` use the same 300 s cap. The simulator itself contradicts the README's unqualified wording once faults combine: in the 10 000-seed probe the time from `max(last write, last partition end)` to convergence exceeded 2 × `sync_interval` in 8 seeds (0.08 %), worst 88.8 s (seed 1763500000223000669, 3 nodes, 22.7 % loss), because sync listings, `DataRequest`s and their replies are not retransmitted and a lost one waits for the next 30 s round.

Verdict on "convergence bounds proven by a deterministic simulator … under loss/duplication/reordering/partition": the simulator is genuinely deterministic and drives the production decision core. The bounds are proven for one fault at a time on an otherwise perfect network, and one of the three is proven by a test that does not exercise the mechanism the bound describes. Under combined faults only eventual convergence is checked — by a test that fails for 3.7 % of seeds for reasons unrelated to the protocol.

### 5.5 How much chaos runs in CI

One scenario per `go test` invocation: one wall-clock seed, 2–5 nodes, 40 writes over 200 virtual seconds, 1–3 symmetric partitions of 10–40 s, drop 5–25 %, duplicate 2–10 %, about 10 ms of wall time. Three native CI cells ⇒ three scenarios per pipeline. There is no fixed regression corpus of seeds, no `-count`, no nightly sweep. The four fault classes are asserted to have occurred (never violated in 10 000 seeds).

### 5.6 What a green simulator run does not tell you — concretely

The simulator answers one question well: *given datagrams that arrive intact, in any order, any number of times or not at all, does the decision core converge?* Everything below is outside it, and each item has already produced behaviour on real sockets that the simulator reports as fine.

| Not modelled | What escapes | Evidence |
|---|---|---|
| **CPU cost of handling a datagram** — events take zero virtual time | `handleDataRequest` and `handleDataSync` copy the whole store per datagram (`core.go:606`, `:633`), and the listing is rebuilt per sync tick; recovery cost is quadratic in key count while the simulator's clock sees only the limiter spacing | Simulator, 10 000 keys: recovered in 39.9 s of virtual time, inside the scenario's 70 s bound — but 27.4 s of real CPU for 1 KiB values (probe 2). Coordinator's real two-node Linux stand: ≈ 17 min for 11.7 k keys against an ≈ 83 s bound (reported to me; not reproduced in this audit) |
| **Socket receive buffer and the adapter's 64-slot inbound queue** — the virtual network has unbounded, instantaneous queues and loss that is independent of load | Load-correlated loss: the more a node sends (retransmission scans emit one datagram per due pending key, unpaced; listings every `sync_interval`), the more is dropped, the more is retransmitted | `BenchmarkReplicationBurst` fails to converge within 10 s at 10 000–20 000 keys on Linux, backlog growing 1 → 13 → 114 before that (section 10). Coordinator's stand: a 10 k pending set produced a 14.5 min retransmission storm (reported to me) |
| **Datagram size** — messages are Go structs | A record whose encoded size exceeds 65 507 bytes can never be sent; 60 KiB values fragment at IP level on a real link | Probe 5: value 61 440 bytes + key 5 000 bytes → `sendto: message too long` forever, write acknowledged to its author, never replicated |
| **Codec, CRC, protocol version** — every delivery is `Decoded` at the supported version | Decoder domain checks, the `Deleted` flag on the wire, malformed input | Mutants M53, M54, M55 survive the whole suite |
| **Persistence in the write path** — `Cluster.Write` announces a write the instant it is stored | In production a write is announced only after its dump (≥ 50 ms later) and not at all if the dump fails; remote applies dirty the engine; restart reloads a dump | Row 13 and mutants M18, M21; no simulator scenario contains a dump |
| **The gRPC layer: startup gate, `create_if_absent`, `wait_for_replication`** | Whether the gate actually protects anything; conditional writes racing recovery | Probe 3 (section 6.3). Coordinator's stand: `create_if_absent` on a replaced empty node overwrote a user value on both nodes because the gate opened early (reported to me) — the system-level form of the same defect |
| **Restart that keeps state; downtime longer than `peer_timeout` with state kept** — `ReplaceNode` always boots an empty store | HLC seeding from the dump, stale-but-non-empty reconciliation, tombstones arriving after downtime | Rows 7, 8, 13 |
| **Deletes** | Tombstone replication, listing, healing | M38, M53, M61 survive; probe 4 passes when deletes are added |
| **Clock skew inside chaos** | Skew combined with loss and partition | Probe 4 passes; not in the suite |
| **Long delays** (seconds; datagrams outliving a partition or a restart) | Late stale traffic | One hand-injected datagram in `TestScenario_restart` |

The practical reading: the simulator is a sound proof tool for the merge-and-retransmit logic and should be kept. It is not evidence for anything that depends on bytes, buffers, disks, CPU time or the API layer — which is where the real-stand failures are. Section 11 (TST-06) proposes the smallest extensions that would have caught them.

## 6. Scenario → test matrix

Source: upstream spec, table [spec section] (18 scenario rows under the header line) and the four sequences of [spec section]. Levels: **U** pure unit (hand-built datagrams into one core, or store/clock alone); **S** deterministic simulator (production core + store + clock, virtual network); **R** real UDP sockets, two adapters in one process; **P** two real processes of the built binary. **R and P run on Linux only** (section 2) unless stated. Paths are relative to the repository root; `repl/` = `internal/app/repl/`, `persist/` = `internal/app/persist/`, `storage/` = `internal/app/storage/`, `clock/` = `internal/app/clock/`. Verdicts: **Proven** — the assertion fails when the behaviour is broken; **Weak** — the scenario runs but the assertion would pass with the behaviour broken, or the documented bound is not checked; **Missing** — no test. Mutant ids refer to section 8.

### 6.1 Failure table

| # | Scenario (spec row) | Tests | Level | What is actually asserted | Verdict |
|---|---|---|---|---|---|
| 1 | Loss of one `DataReplication`; retried with growing pause; ≤ `ack_timeout_max` | `TestRetransmit_backoffLadder` `repl/retransmit_test.go:232`; `TestBound_lossWithinAckTimeoutMax` `repl/sim_bound_test.go:161`; `TestScenario_loss` `repl/sim_scenario_test.go:548` (phase 1) | U, S | Exact ladder instants 200/600/1400/3000/6200 ms then every 5000 ms, never early; in the simulator three consecutive drops, key absent at +600 ms and +1400 ms, present at +5 s. Mutants M35, M36, M37 killed | **Proven** (unit + simulator) for up to four consecutive losses. As worded the bound is false for five (next attempt at +6.2 s); the test's own comment restricts it. No real-socket loss injection |
| 2 | Loss of `Ack`; retransmit, neighbour re-Acks the equal version; ≤ `ack_timeout_max` | `repl/retransmit_test.go:585` ("duplicate redelivery … still Acks"); `TestWire_twoNodeHandoff` `internal/udprepl/repl_test.go:76` (through the real codec); implicitly `TestScenario_loss`, `_oneWay`, `_clockJumps`, `TestChaos_convergence` (their quiescence requires every unacked entry to drain) | U, S | A second delivery of the same record emits a second Ack and leaves tracking unchanged. M32 (no Ack on an equal version) is killed by two unit tests and four simulator tests | Mechanism **Proven**. Bound **Weak**: no test drops an Ack and times the sender's drain; only "eventually within 300 s" |
| 3 | Duplicate, reordering, own broadcast: equal versions Acked and not forwarded; a stale version draws **one unicast reply**; own datagrams dropped by `NodeID`; immediate, no avalanche | dup: `TestMerge_duplicateDelivery` `storage/merge_test.go:194`, `repl/retransmit_test.go:585`; reorder: chaos (`FaultStats.Reordered > 0` asserted), `TestScenario_loss` phase 2, merge laws; own echo: `TestDiscard_classes` `repl/discard_test.go:45`, `TestSimnet_faultsObserved` `internal/app/internal/simnet/simnet_test.go:177`, `TestSmoke_twoNodesUDP` `internal/udprepl/smoke_test.go:289` (R) | U, S, R | Duplicates are no-ops that Ack; the own-echo class counts and never reaches protocol processing (M40 killed by 16 tests) | dup / reorder / own echo **Proven**. **Stale version: Missing** — no test feeds the core a strictly older `DataReplication`. The implementation Acks the stale version and sends **no** reply (`repl/core.go:523-543`), the opposite of the spec row and of sequence 3; nothing pins either behaviour — M71, which stops Acking stale versions, survives. "No avalanche": **Weak** — implied by chaos quiescence, no datagram count anywhere. Whether replies really leave as unicast on real sockets was not probed |
| 4 | Partition for seconds and minutes; both sides serve; unacked set bounded; heal by retransmit and sync within 2 × `sync_interval` | `TestBound_partitionHeal` `repl/sim_bound_test.go:207` (3 nodes, 3 s); `TestScenario_partitionSeconds` `repl/sim_scenario_test.go:266` (2–3 s); chaos (10–40 s windows) | S | Stores differ at the heal instant and are equal at heal + 60 s + (K−1) × 2 ms | **Weak.** The partitions are shorter than `peer_timeout`, so the ladder heals them ≈ 2 s after the heal; the 60 s bound cannot fail and the sync path is not exercised (both tests pass under M73, periodic sync disabled). The minute-scale case — peer evicted, ≥ 3 nodes, heal only by the periodic listing — is timed nowhere; chaos covers it with the flaky quiescence assertion. "Bounded unacked set" is not asserted |
| 5 | Partition for hours and days; same key written on both sides; greater version wins; losers go to the conflict journal | `TestScenario_partitionDays` `repl/sim_scenario_test.go:343` | S | 2 nodes, 2–3 virtual days, four shared keys written on both sides; every node serves the later-stamped record byte-equal; `MergeStats.Conflicts ≥ 8`; quiescence within 300 s | Convergence and deterministic winner **Proven** (2 nodes, no clock skew, timers altered to hello 30 s / peer_timeout 90 s). Bound **Weak** (300 s cap, not 60 s). **Conflict journal: Missing** — no journal exists; a counter replaces it (documented decision D-10) and the counter is asserted only inside the core; the adapter accessor that feeds `/metrics` (`udprepl.MergeStats`) has no in-process coverage and no test checks the exported value |
| 6 | One-way connectivity; changes flow one way, the other side keeps retransmitting, no avalanche; heal within 2 × `sync_interval` | `TestScenario_oneWay` `repl/sim_scenario_test.go:449` | S | n1→n2 blocked for 2.5–3 s: the receiving side holds both writes, the far side lacks the blocked one; after the block lifts, quiescence and model equality | **Weak.** Short block only (< `peer_timeout`, so the deaf side never evicts); "retransmissions continue" and "no avalanche" are not asserted (no counter read); the bound is the 300 s cap |
| 7 | Restart of one node: dump read, neighbour detects the new `RunID` and sends a full sync, writes open after the sync or `startup_gate` | dump: `TestEngine_endToEndRestart` `persist/engine_test.go:117`; run id: `TestRestart_runIDChangeTriggersFullSync` `repl/restart_test.go:97`, `TestScenario_restart` `repl/sim_scenario_test.go:876`; gate: `TestGate_writesUnavailableReadsOK` / `_opensOnSync` / `_opensOnTimeout` `internal/grpcapi/gate_test.go:64,111,141`, `TestSync_syncArrived` `repl/sync_test.go:483`, `TestMirror_syncArrivedFlip` `internal/udprepl/repl_test.go:437`; whole: `TestExec_killNineRestart` `cmd/sup-kv/scenario_test.go:394` (P) | U, S, R, P | Each link separately: records reload byte-equal; a changed run id yields exactly one unicast full listing; the gate rejects writes until an injected `SyncArrived` closure flips or the backstop fires. In the two-process test the restarted node serves both keys again | Components **Proven**. End to end **Weak**: the P test cannot tell "restored from the dump" from "resynced from the live peer", and it awaits `SERVING` with a 1 s gate, so sync-opening is not isolated. The simulator's restart always uses an **empty** store. **The gate's purpose is unproven and does not hold** — probe 3, below |
| 8 | One node down for hours or days; on return tombstones and records arrive by sync; ≤ 2 × `sync_interval` plus replies | `TestSync_compare` `repl/sync_test.go:556`; `TestDataRequest_replyAndSilence` `repl/sync_test.go:75` (a tombstoned key is answered with the tombstone) | U | Per-key compare against one snapshot; a newer local tombstone travels as the reply | **Weak → Missing for tombstones.** No simulator or process test returns a node with stale state, and none ever deletes a key. Three independent mutants that break delete replication survive the whole suite on both platforms: M38 (listing omits tombstones), M53 (`Deleted` flag dropped on the wire), M61 (API announces a delete as a live empty value) |
| 9 | Disk lost / machine replaced: starts empty, pulls everything, rate-limited; time ∝ keys / `sync_response_rate` | `TestScenario_emptyDisk` `repl/sim_scenario_test.go:797`; `TestSync_rateLimit` `repl/sync_test.go:649`; `TestExec_emptyDiskRecovery` `cmd/sup-kv/scenario_test.go:626` (P) | U, S, P | Simulator: 50 keys recovered within `sync_interval + (2N+2) × 2 ms` of virtual time. Processes: 20 keys at 50/s within N/rate + 3.4 s. Unit: spacing exact to the nanosecond (M39 killed) | **Proven at toy scale**; upper bounds only, so the P test passes with the limiter knob unwired (M69). Proportionality at two sizes and the envelope scale (10³–10⁴ keys) untested (probe 2: fine in the simulator up to 10 000) |
| 10 | Both nodes down: each reads its dump; confirmed writes are on disk; unconfirmed may be lost; sync after start | `TestExec_bothNodesDown` `cmd/sup-kv/scenario_test.go:503` (P); `TestExec_sigtermFinalDump` `:571` (P); `TestMain_lifecycle` `cmd/sup-kv/main_test.go:429` | P | Both nodes stopped **by SIGTERM**, restarted, both keys served; a replicated write is in the final dump and is served by a lone restart | **Weak.** Graceful stop only. No test kills both nodes, and none kills a lone node right after an acknowledged write and restarts it alone — the product's core promise. In-process, "confirmed ⇒ on disk" is asserted only for a quiescent single writer; the ordering mutant M21 survives and M20 is killed only by dump-*failure* tests (section 8) |
| 11 | Dump corruption: bad-checksum records dropped; unreadable file replaced by `.prev`; neither usable → empty start, error logged, metric set; recovered from the neighbour | checksum: `TestCodec_discardClasses` `persist/codec_test.go:176`, `TestEngine_restartDiscardsTornRecord` `persist/engine_test.go:865`, `TestExec_bothNodesDown` (P: record gone alone, healed when the peer returns); `.prev`: `TestFileStore_fallbackChain` `internal/persistfile/dumpfile_test.go:97`, `TestCodec_truncatedGenerationUnreadable` `persist/codec_test.go:367`; empty: `TestEngine_openEmptyStart` `persist/engine_test.go:216` | U, P | Per-record discard with a count and a startup log line; chain main → `.prev` with a stand-in oracle; both files **absent** → empty start (M15, M17, M28, M29, M30 killed) | Bad checksum **Proven**, including two processes. `.prev` fallback **Proven at unit level only** — never through `persist.Open` with the real decoder, never with the binary. Both generations **corrupt**: **Weak** (only "both removed"). "Error logged and metric set" on an empty start: **Missing** — the binary emits neither |
| 12 | System clock changed on a running node; HLC monotonic | `TestClock_monotonic` `clock/hlc_prop_test.go:35`; `TestClock_tick` `clock/clock_test.go:88`; `TestScenario_clockJumps` `repl/sim_scenario_test.go:115` | U, S | 2 000-step adversarial walk strictly increasing; in the simulator a forward jump of hours and a backward jump of minutes still produce dominating versions, and the cluster converges | **Proven** |
| 13 | Clock went backwards between restarts: `l` continues from the dump maximum | `TestClock_restartFromDumpMax` `clock/clock_test.go:355`; `TestClock_restartProperty` `clock/hlc_prop_test.go:238`; `TestStore_restartDominance` `storage/store_test.go:586`; `TestWatermark_fullProperty`, `_equalCounterDomain` `persist/codec_test.go:291,346`; `TestEngine_endToEndRestart` `persist/engine_test.go:117` | U | A clock **constructed by hand** with the watermark dominates the dump; the watermark function is exact | **Weak.** The composition that runs at startup (`persist.Open`: `Watermark` → `NewClock`) is exercised only with `time.Now` on both sides of the restart, where the wall clock has moved forward by itself. **M18 — `Open` ignores the dump maximum — survives on both platforms.** The simulator half ("restart-time jump") restarts with an empty store |
| 14 | One node's clock far ahead: `l` of both runs ahead; log warning and metric; convergence kept | `TestScenario_farAhead` `repl/sim_scenario_test.go:197`; `TestClock_driftSignal` `clock/clock_test.go:500`; `TestMergeStats_counts` `repl/merge_stats_test.go:18` | U, S | 24 h skew: convergence, far-ahead write wins a concurrent pair, drift warnings ≥ 1 on the peers and 0 on the fast node (M07, M14 killed) | Convergence and the core counter **Proven**. **Log warning: Missing** (nothing logs drift). Metric end to end **Weak**: the `--drift-warn` wiring can be removed (M66 survives) and the exported metric value is asserted nowhere (`TestMetrics_families` renders an all-zero snapshot) |
| 15 | Burst ≈ 1000 writes/s: dumps coalesce to at most one per `dump_debounce`; one datagram per write; socket-buffer losses repaired by retransmit; client reply delayed by `dump_debounce` + dump time | `TestEngine_burstBoundedConfirmation` `persist/engine_test.go:432`; `TestScenario_burst` `repl/sim_scenario_test.go:718`; `BenchmarkReplicationBurst` (not in CI) | U (virtual time), S | 1000 engine writes at 1 ms: ≤ 3 dumps, **every write confirmed by oldest + 1 s (`dump_max_wait`)**; simulator: 3000 writes on 3 nodes, loss-free, model equality 60.6 s after the burst | **Weak.** Coalescing is proven against a bound 20 × looser than the spec row (the debounce is a quiet window that every write resets). The simulator half asserts only convergence long after the burst, on a perfect network. Socket-buffer overflow: **Missing** (not modelled, never provoked) |
| 16 | Same `NodeID` on both nodes: configuration error logged at the first `Hello`; neighbour's datagrams dropped as own | `TestDiscard_classes` `repl/discard_test.go:45` (both variants, log text); `TestScenario_discardClasses/duplicate_node_id` `internal/udprepl/scenario_test.go:428` (R); `TestLog_jsonDefault`, `TestLog_formatText` `cmd/sup-kv/main_test.go:709,757` (P) | U, R, P | Counter isolated to its class, nothing reaches protocol processing, log names the node id and handles; two real processes with `--node-id=1` each log the error | **Proven** (core on every platform; R and P on Linux, and the two P tests are flaky — section 3.2) |
| 17 | Incompatible protocol version after upgrading one node: datagrams dropped and logged; nodes work independently; sync after the second upgrade | `TestDiscard_classes` (`bad_proto_version`); `TestDiscardLog_classes` `internal/udprepl/repl_test.go:623` (real socket, runs on darwin too); `TestScenario_discardClasses/bad_proto_version` `internal/udprepl/scenario_test.go:402` (R) | U, R | Class counted, datagram ignored (M41 killed); one log record **at Debug level** | Classification **Proven**. "Logged": only at Debug — invisible at the default level, and the test pins that. The operational half (two versions side by side, convergence after the second upgrade): **Missing** — the version is a constant; no test builds a second version |
| 18 | Foreign datagram on the segment: not detectable (CRC is not authentication) | `TestDataRequest_replyAndSilence` (a stranger's request is answered); `TestDiscard_totality` `repl/discard_test.go:367` (500 random *decoded* datagrams never panic; counter conservation) | U | Totality over port structs | Accepted limitation, so nothing to prove about detection. Robustness **Weak**: no raw-byte input ever reaches the real decoder (M54, M55 survive; no fuzz) |

### 6.2 The four exchange sequences

| Sequence | Tests | Level | Asserted | Verdict |
|---|---|---|---|---|
| Write and replication: client → A; `Tick`, map, dump; broadcast, mark unacked; `OK` without waiting for the neighbour; B `Update`s, applies, notifies subscribers, Acks; A stops retransmitting | `TestWire_twoNodeHandoff` `internal/udprepl/repl_test.go:76`; `TestBound_normalOneRTT` `repl/sim_bound_test.go:128`; `TestServer_setGetRoundTrip` `internal/grpcapi/server_test.go:296`; `TestWatch_replicatedChange` `internal/grpcapi/watch_test.go:335`; `TestWait_timeout` `internal/grpcapi/wait_test.go:175`; `TestWaitIntegration_ackedThroughRealAdapterMirror` `internal/grpcapi/wait_integration_test.go:173` (R); `TestScenario_convergence` `internal/udprepl/scenario_test.go:270` (R); README / EXAMPLES (P) | U, S, R, P | Each hop, and on Linux the whole chain through gRPC and real sockets | **Proven on Linux.** On darwin the chain stops at "core + codec": removing the API's replication hand-off (M60) fails nothing there. Even on Linux the four `TestExec_*` scenarios do not notice M60 — their 6 s convergence deadline is satisfied by the 200 ms anti-entropy round, so they do not distinguish one-RTT replication from eventual sync |
| Loss and retransmission; pause doubles to `ack_timeout_max`; suspended when no `Hello` for `peer_timeout` | rows 1–2; `TestRetransmit_suspendAndFlush` `repl/retransmit_test.go:332`; `TestPeerTable_solePeer` `repl/peertable_test.go:147`; `TestPeerTable_lifecycle` `:28` | U, S | Exact ladder, exact eviction boundary, suspension retains the entry, first `Hello` flushes (M37, M47, M49 killed) | **Proven** |
| Concurrent write of one key: B applies `vA`, drops `vB` from retransmit, Acks; A sees the lower `vB`, **replies with its record, no Ack**; B Acks the equal version; no loop; loser journalled | `repl/retransmit_test.go:508` (remote supersession drops the entry); `TestScenario_farAhead` (same-instant pair, winner on a third node); `TestScenario_partitionDays` (conflict counter) | U, S | Final state and the supersession of tracking (M34 killed) | Outcome **Proven**. Steps 3–4 **Missing**: the implementation Acks the lower version instead of replying, and no test defines the behaviour. Journal **Missing** (feature absent) |
| Restart and full sync: A reads the dump, seeds the HLC, announces a new `RunID`; reads served, writes gated; B sends a chunked `DataSync`; A requests missing/stale keys and pushes newer ones; replies rate-limited; B's missing keys found through A's own sync | row 7; `TestSync_triggers` `repl/sync_test.go:438`; `TestSync_chunking` `:338`; `TestSync_compare` `:556`; `TestSync_rateLimit` `:649`; `TestSync_chunkWireBudget` `internal/udprepl/syncsize_internal_test.go:76` | U, S, P | Startup order `Hello` then listing; chunks within the byte budget against the real codec; compare, request and push per key; exact pacing (M42, M43, M45, M46, M51 killed) | **Proven piecewise.** Never as one scenario with a node that restarts **with** state: the "A holds newer records than B" direction exists only in a hand-built unit case |

### 6.3 Probe 3 — the startup gate opens before the clock has absorbed the neighbour's versions

The spec gates writes "until the first full sync from the neighbour has been processed", and its Recovery algorithm says a node receiving a `DataSync` part "calls `Update` with the part's maximum version". The implementation raises the gate signal on the first `DataSync` datagram received (`repl/core.go:632`) and never feeds listing versions to the clock; the HLC only advances when the requested records arrive, behind the 500/s limiter.

Probe (simulator, production core/store/clock; 2 nodes; B's wall clock 1 h ahead; A is replaced by an empty node with a correct clock; `SyncResponseRate: 1` to make the window visible):

```
gate opened 2ms after restart; A holds k1 at that instant: false;
A's new version 0192cc09146b7000 vs B's listed version 0192cc4002857001; A's confirmed write dominates: false
final value on A: "written by B, clock +1h" (A's post-gate write survived: false)
```

A write accepted and (in production) acknowledged immediately after the gate opens is stamped below a version the neighbour had already advertised, and silently loses. This needs clock skew (or an `l` that ran ahead earlier) plus a restart — exactly spec rows 13 and 14 combined. No test covers it: the gate is tested as a boolean, never for the property it exists to provide.

## 7. Property tests

* **Framework: hand-rolled.** Seeded loops over `math/rand/v2` PCG through `testutil.NewRNG`. `pgregory.net/rapid` — presented in the project's stack notes as "the one justified test-only dependency", chosen for shrinking — is **not in `go.mod` and not imported anywhere**. `testing/quick` is not used either. Consequences: no shrinking (a failure reports a step index inside a 500-step run), no stateful/model-based generation.
* **Seed replay:** every property test logs `property seed: hi,lo (replay: go test ./... -run '<name>' -args -seed=N)`; shown on failure. The printed command is wrong as written: with `./...` the `-seed` flag reaches test binaries that do not link `testutil` and they abort with a usage error (ran: `internal/metrics` → `FAIL`). It works when the package is named. `-seed=0` cannot be replayed (0 means "use the wall clock").
* **Iteration counts:** merge laws 500 store triples (twice: mixed and tombstone-heavy), duplicate delivery 200, loser accounting 500; HLC monotonic walk 2 000 steps plus 8 × 500 concurrent, post-receive 1 000 + 100, counter reset 2 000, drift 5 000; version order 1 000; codec round trip 500; discard totality 500; uniqueness 10 000 ticks. Each CI run uses fresh seeds.

| Stated law | Test | Asserted in general form? | Generator bias |
|---|---|---|---|
| Merge commutative | `TestMerge_laws`, `TestMerge_lawsTombstones` (`storage/merge_test.go:130,142`) | yes — `merge(a,b)` and `merge(b,a)` reach equal states **and** equal conflict sets, over random stores | 1–16 records over 32 keys (collisions frequent); one third tombstones, or seven eighths in the heavy variant. **Versions are uniform over the full 48-bit `l`**, so two records never tie on `l`: the counter and node-id tie-breaks are never exercised by the law tests |
| Merge associative | same | yes, as `(a⊔b)⊔c = (b⊔c)⊔a` | same |
| Merge idempotent | same, plus `TestMerge_duplicateDelivery` | yes — self-merge changes nothing and reports no conflicts | same |
| No resurrection | `TestMerge_lawsTombstones` second subtest | one random pair per run | — |
| HLC monotonic under adversarial clocks | `TestClock_monotonic` (`clock/hlc_prop_test.go:35`) | yes — strictly increasing `(l, c)` over Tick/Update; drift sign checked against a model at every step | good: 10 % backward jumps of up to 4 h, 10 % forward, 20 % stuck, remote `l` within ±2 ms of local, remote counter over the full 12 bits |
| Versions unique | `TestVersion_uniquePerWrite` | 10 000 ticks on one stuck clock, one node id; cross-node uniqueness rests on the packing tests | covers counter overflow twice |
| Post-receive ordering | `TestClock_postReceive` | yes — after `Update` (and after a full `Merge` into a populated store) the next local version beats the received one | half the remote readings are within ±1 ms of the local state — the interesting region |
| Restart dominance | `TestClock_restartProperty`, `TestClock_restartFromDumpMax`, `TestStore_restartDominance`, `TestWatermark_*` | yes at clock level (wall clock up to 4 h behind the dump maximum, 100 ticks) — but **one** random `(dumpMax, counter)` per run, and only for a clock constructed by hand | the composition actually used at startup, `persist.Open` → `Watermark` → `NewClock`, is never run with a wall clock behind the dump (mutant M18 survives) |
| Version byte order = `(l, c, node)` | `TestVersion_byteOrder` | yes | good: half the triples come from an 8 × 4 × 4 space, so ties occur |

The evidence from mutation: the tie-break mutant M02 (join compares the timestamp only) is **not** killed by any merge-law property — only by the deterministic `TestStore_restartDominance`. All four HLC mutants (M10–M13) and the drift mutant (M14) are killed several times over. The HLC property tests are strong; the merge-law tests are sound but blind to equal-`l` conflicts — the case that decides concurrent writes inside one millisecond and every write after a counter-driven `l`.

Laws are asserted at store level. Order-independence at system level rests on the chaos test (section 3.1, 5.4).

## 8. Mutation probes

### 8.1 Method

One exact-string edit to one production file per mutant; `go build ./...` must pass; then the **whole** suite on darwin/arm64 and in the linux/arm64 container: `go test -count=1 -p 1 -timeout 150s ./...`, `-race` off, `SUPKV_SIM_SEED=1` (a seed on which the unmutated suite is green, so the 3.7 % chaos flake cannot be mistaken for a kill). Failing packages were re-run once; a test counts as a killer only if it failed both times. The file was restored after every mutant and the tree re-checked against the repository at the end. 73 mutants were evaluated; 13 more were prepared and not run (8.5).

**Score: 57 of 73 killed where the whole suite runs (Linux, 78 %); 54 of 73 on darwin (74 %). 16 survive on both platforms; 3 more survive on darwin only.**

### 8.2 Table

| # | Mutation (file:line — one-line diff) | Verdict | Killed by (top-level tests; darwin = Linux unless noted) |
|---|---|---|---|
| M01 | `internal/app/storage/store.go:242` — LWW join: equal version re-installs (> becomes >=). *case cmp > 0: -> case cmp >= 0:* | killed | `TestEngine_redundantDirty`, `TestMergeStats_counts`, `TestMerge_duplicateDelivery` +4 |
| M02 | `internal/app/storage/store.go:240` — LWW join compares the 48-bit timestamp only (counter and node_id tie-break ignored). *cmp := rec.Version.Compare(stored.Version) -> bytes.Compare(rec.Version[:6], stored.Version[:6])* | killed | `TestStore_restartDominance` |
| M03 | `internal/app/storage/store.go:235` — Merge treats a stored tombstone as absent (any older live record resurrects the key). *if !ok { install } -> if !ok \|\| stored.Deleted { install }* | killed | `TestMerge_laws`, `TestMerge_lawsTombstones`, `TestMerge_losersProperty` |
| M04 | `internal/app/storage/store.go:193` — create-if-absent ignores tombstones (SetIfAbsent succeeds over a deleted key). *if _, ok := s.records[key]; ok -> if rec, ok := s.records[key]; ok && !rec.Deleted* | killed | `TestEngine_writeSurface`, `TestServer_errorMapping`, `TestStore_SetIfAbsent` |
| M05 | `internal/app/storage/store.go:160` — value-size check off by one in Set (a value of exactly value_max_bytes is rejected). *Set: len(data) > ValueMaxBytes -> >=* | killed | `TestStore_valueLimit` |
| M06 | `internal/app/storage/store.go:220` — value-size check off by one in Merge (remote record of exactly value_max_bytes refused). *Merge: len(rec.Data) > ValueMaxBytes -> >=* | killed | `TestStore_valueLimit` |
| M07 | `internal/app/storage/store.go:227` — HLC Update not called when a remote record is merged (DataReplication / DataRequest reply path). *Merge: drop s.clock.Update(remoteL, remoteC)* | killed | `TestClock_postReceive`, `TestMergeStats_counts`, `TestScenario_farAhead` |
| M08 | `internal/app/storage/store.go:235` — Watch tap not fired for replicated changes (OnChange silenced in the remote-merge install). *join: cmp>0 and !ok installs run with OnChange swapped to a no-op* | killed | `TestStore_onChange`, `TestWatch_replicatedChange` ; Linux adds `TestExamplesDoc` |
| M09 | `internal/app/storage/store.go:206` — Delete of an already-deleted key writes a new tombstone instead of NotFound. *Delete: if !ok \|\| rec.Deleted -> if !ok* | killed | `TestEngine_snapshotCreateLoadRestore`, `TestStore_Delete` |
| M10 | `internal/app/clock/clock.go:104` — Tick: counter overflow wraps to 0 without bumping l (versions repeat / go backwards). *Tick overflow: drop c.l++* | killed | `TestClock_counterOverflowBoundary`, `TestClock_restartFromDumpMax`, `TestStore_restartDominance` +1 ; Linux adds `TestClock_monotonic` |
| M11 | `internal/app/clock/clock.go:129` — Update: remote counter ignored when l is equal on both sides. *c.c = max(c.c, remoteC) + 1 -> c.c++* | killed | `TestClock_ceilingSaturation`, `TestClock_postReceive`, `TestClock_update` |
| M12 | `internal/app/clock/clock.go:133` — Update: counter reset to 0 when the remote l wins (next Tick may not beat the received version). *case newL == remoteL: c.c = remoteC + 1 -> c.c = 0* | killed | `TestClock_postReceive`, `TestClock_update` |
| M13 | `internal/app/clock/clock.go:80` — restart seeding ignores the dump watermark counter (NewClock drops initialC). *c: initialC & counterMask -> c: 0* | killed | `TestClock_ceilingSaturation`, `TestClock_restartFromDumpMax`, `TestClock_restartProperty` +1 |
| M14 | `internal/app/clock/clock.go:172` — drift warning threshold ignored (any non-zero divergence is reported). *driftBetween: drop the > warn gate* | killed | `TestClock_driftSignal`, `TestClock_driftUnderChaos`, `TestClock_monotonic` +2 |
| M15 | `internal/app/persist/codec.go:142` — record checksum not verified when loading a dump. *decodeRecord: drop the RecordChecksum != line.Checksum discard* | killed | `TestCodec_discardClasses`, `TestEngine_restartDiscardsTornRecord` ; Linux adds `TestExec_bothNodesDown` |
| M16 | `internal/app/persist/codec.go:174` — dump watermark ignores the counter at the max l. *Watermark: drop case l == initialL && c > initialC* | killed | `TestWatermark_equalCounterDomain` |
| M17 | `internal/app/persist/codec.go:104` — dump format_version not checked (any version accepted). *Decode: drop the FormatVersion != dumpFormatVersion rejection* | killed | `TestCodec_discardClasses` |
| M18 | `internal/app/persist/engine.go:146` — restart seeding ignores the dump max entirely (clock starts from the wall clock). *Open: clock.NewClock(now, initialL, initialC, ..) -> NewClock(now, 0, 0, ..)* | **SURVIVED** | — |
| M19 | `internal/app/persist/engine.go:306` — dump error still confirms the waiting writes (client gets OK for a failed dump). *runDump: on error confirmWaiters instead of failWaiters* | killed | `TestConfirmWaiters_aboveCoverInvariant`, `TestEngine_dumpFailureFansOut`, `TestEngine_snapshotFlushErrorAborts` +2 |
| M20 | `internal/app/persist/engine.go:302` — write confirmed BEFORE the dump is saved (waiters released between Encode and Save). *runDump: confirmWaiters moved in front of dumps.Save* | killed | `TestEngine_dumpFailureFansOut`, `TestEngine_snapshotFlushErrorAborts`, `TestEngine_snapshotFlushForcesDump` +1 |
| M21 | `internal/app/persist/engine.go:293` — dump snapshot taken BEFORE the waiter/cover capture (a write landing in between is confirmed although absent from the saved dump). *runDump: Encode(e.store.Records()) hoisted above the mutex capture* | **SURVIVED** | — |
| M22 | `internal/app/persist/engine.go:513` — remote apply does not mark the engine dirty (replicated records never dumped until a local write). *ApplyRemote: drop e.markDirty()* | killed | `TestDumpStats_records`, `TestEngine_applyRemoteTriggersDump`, `TestEngine_redundantDirty` +1 ; Linux adds `TestExec_bothNodesDown`, `TestExec_sigtermFinalDump` |
| M23 | `internal/app/persist/engine.go:379` — failed dump does not re-arm (no retry of the dump after an error). *failWaiters: drop e.dirty = true* | killed | `TestEngine_snapshotFlushErrorAborts`, `TestEngine_snapshotFlushForcesDump` |
| M24 | `internal/app/persist/engine.go:241` — dump_max_wait deadline never armed (a steady write stream can postpone the dump forever). *rearm: drop the deadline timer* | killed | `TestEngine_burstBoundedConfirmation` |
| M25 | `internal/app/persist/engine.go:618` — snapshot restore re-installs the snapshot versions via Merge instead of fresh versions (restore loses to current values). *replayRecord: store.Merge(key, rec) instead of Set/Delete* | killed | `TestEngine_configShrunkValueLimits`, `TestEngine_snapshotCreateLoadRestore`, `TestSnapshot_createLoadRoundTrip` |
| M26 | `internal/persistfile/dumpfile.go:143` — fsync of the temp file skipped before rename. *writeSynced: return syncDurable(tmp) -> return nil* | **SURVIVED** | — |
| M27 | `internal/persistfile/dumpfile.go:112` — directory fsync skipped after rename. *saveAtomically: return syncDir(dir) -> return nil* | **SURVIVED** | — |
| M28 | `internal/persistfile/dumpfile.go:150` — .prev rotation skipped (previous generation overwritten). *placeFile: rotation branch disabled* | killed | `TestFileStore_atomicRotation`, `TestFileStore_nameValidation` |
| M29 | `internal/persistfile/dumpfile.go:75` — Load never falls back to .prev. *Load: candidates {main, main.prev} -> {main}* | killed | `TestFileStore_fallbackChain` |
| M30 | `internal/persistfile/dumpfile.go:80` — Load accepts a main file the parse oracle rejects. *Load: parse error no longer advances the chain* | killed | `TestFileStore_fallbackChain` |
| M31 | `internal/persistfile/dumpfile.go:103` — non-atomic dump: main file written in place (no temp file, no data fsync, no rename). *saveAtomically: rotate, then os.WriteFile(path) directly* | **SURVIVED** | — |
| M32 | `internal/app/repl/core.go:524` — no Ack on an equal-version redelivery (lost-Ack recovery broken). *applyRemote: return without Ack when the stored version already equals the received one* | killed | `TestChaos_convergence`, `TestMergeStats_counts`, `TestRetransmit_ackDrainSemantics` +4 |
| M33 | `internal/app/repl/core.go:527` — Ack sent for a record the store refused (error path falls through to the Ack). *applyRemote: drop the return in the ApplyRemote error branch* | killed | `TestRetransmit_ackDrainSemantics` |
| M34 | `internal/app/repl/core.go:539` — unacked entry not dropped when a greater remote version supersedes it. *applyRemote: drop the supersession delete* | killed | `TestRestart_staleRunIdAndInFlight`, `TestRetransmit_ackDrainSemantics` |
| M35 | `internal/app/repl/core.go:883` — retransmit backoff does not double. *backoff = min(2*backoff, max) -> min(backoff, max)* | killed | `TestBound_lossWithinAckTimeoutMax`, `TestRetransmit_backoffLadder` |
| M36 | `internal/app/repl/core.go:883` — retransmit backoff not capped at ack_timeout_max. *backoff = min(2*backoff, max) -> 2*backoff* | killed | `TestRetransmit_backoffLadder` |
| M37 | `internal/app/repl/core.go:873` — retransmission not suspended while no peer is known. *scanRetransmit: drop the len(c.peers) == 0 guard* | killed | `TestPeerTable_solePeer`, `TestRetransmit_suspendAndFlush` |
| M38 | `internal/app/repl/core.go:775` — DataSync listing omits tombstones. *listingChunks: skip records with Deleted* | **SURVIVED** | — |
| M39 | `internal/app/repl/core.go:676` — sync_response_rate limiter removed (egress queue drains in one burst). *drainEgress: nextSendAt no longer advances by the spacing* | killed | `TestDataRequest_replyAndSilence`, `TestScenario_restart`, `TestSync_compare` +2 |
| M40 | `internal/app/repl/core.go:391` — own-broadcast echo filter removed (own datagrams processed as a peer). *HandleDatagram: if in.NodeID == c.nodeID {...} disabled* | killed | `TestBound_lossWithinAckTimeoutMax`, `TestBound_normalOneRTT`, `TestBound_partitionHeal` +13 ; Linux adds `TestLog_formatText`, `TestLog_jsonDefault`, `TestMirror_drainedAckedDedup` +3 |
| M41 | `internal/app/repl/core.go:386` — unsupported proto_version accepted. *HandleDatagram: proto version check disabled* | killed | `TestDiscardLog_classes`, `TestDiscard_classes`, `TestDiscard_onDiscardSeam` +1 ; Linux adds `TestScenario_discardClasses` |
| M42 | `internal/app/repl/core.go:638` — DataSync: locally absent keys are not requested (empty-node recovery broken). *handleDataSync: case !present \|\| cmp < 0 -> case present && cmp < 0* | killed | `TestScenario_clockJumps`, `TestScenario_emptyDisk`, `TestScenario_restart` +3 ; Linux adds `TestExec_bothNodesDown`, `TestExec_killNineRestart` |
| M43 | `internal/app/repl/core.go:640` — DataSync: a locally newer record is not pushed back to the lister. *handleDataSync: drop case cmp > 0* | killed | `TestSync_compare` |
| M44 | `internal/app/repl/core.go:569` — a stale Ack (other version) drains the newer pending entry. *handleAck: drop entry.version != ack.Version* | killed | `TestRetransmit_ackDrainSemantics` |
| M45 | `internal/app/repl/core.go:803` — DataSync chunk size limit ignored (whole listing in one datagram). *chunkSyncEntries: split condition disabled* | killed | `TestRestart_runIDChangeTriggersFullSync`, `TestRestart_staleRunIdAndInFlight`, `TestSync_chunkWireBudget` +1 |
| M46 | `internal/app/repl/core.go:486` — neighbour restart (new run_id) no longer triggers the targeted full sync. *handleHello: restarted never set* | killed | `TestDiscard_classes`, `TestRestart_runIDChangeTriggersFullSync`, `TestRestart_staleRunIdAndInFlight` +2 |
| M47 | `internal/app/repl/core.go:496` — pending set not flushed when the first peer is discovered. *handleHello: discovery flush disabled* | killed | `TestMirror_drainedAckedDedup`, `TestMirror_pendingAckVisible`, `TestPeerTable_solePeer` +1 |
| M48 | `internal/app/repl/core.go:484` — startup gate sync signal raised by a Hello instead of a DataSync. *handleHello: c.syncArrived = true* | killed | `TestMirror_syncArrivedFlip`, `TestSync_syncArrived` |
| M49 | `internal/app/repl/core.go:834` — peers never evicted after peer_timeout. *evictPeers: expiry check disabled* | killed | `TestChaos_convergence`, `TestChaos_determinismReplay`, `TestPeerTable_lifecycle` +3 |
| M50 | `internal/app/repl/core.go:606` — DataRequest for a tombstoned key is not answered (deletes do not heal through sync). *handleDataRequest: if !ok -> if !ok \|\| rec.Deleted* | killed | `TestDataRequest_replyAndSilence` |
| M51 | `internal/app/repl/core.go:638` — DataSync: stale local keys are not re-requested (only absent ones are). *handleDataSync: case !present \|\| cmp < 0 -> case !present* | killed | `TestSync_compare` |
| M52 | `internal/udprepl/repl.go:745` — bad CRC accepted by the datagram decoder. *decode: drop the crc32 mismatch -> BadCRC branch* | killed | `TestDiscardLog_classes` ; Linux adds `TestScenario_discardClasses` |
| M53 | `internal/udprepl/repl.go:776` — Deleted flag dropped on the wire (tombstones replicate as live empty values). *protoBody: drop Deleted: m.Record.Deleted* | **SURVIVED** | — |
| M54 | `internal/udprepl/repl.go:737` — decoder accepts an envelope with out-of-range node_id / wrong-length run_id. *decode: drop the node_id/run_id domain check* | **SURVIVED** | — |
| M55 | `internal/udprepl/repl.go:845` — decoder accepts a record version that is not 16 bytes. *recordVersion: drop the length check* | **SURVIVED** | — |
| M56 | `internal/udprepl/repl.go:775` — record data dropped on the wire (replicated values arrive empty). *protoBody: drop Data: m.Record.Data* | killed | `TestWire_twoNodeHandoff` ; Linux adds `TestExamplesDoc`, `TestExec_bothNodesDown`, `TestExec_emptyDiskRecovery` +7 |
| M57 | `internal/grpcapi/gate.go:109` — startup gate open immediately (writes accepted before the first sync / startup_gate). *gated(): return false* | killed | `TestGate_opensOnSync`, `TestGate_opensOnTimeout`, `TestGate_writesUnavailableReadsOK` +1 |
| M58 | `internal/grpcapi/wait.go:177` — wait_for_replication reports acked=true when the wait-set emptied without full Ack coverage (eviction counted as Ack). *waitForDrain: resolved branch returns true* | killed | `TestWait_evictionNotAck` |
| M59 | `internal/grpcapi/wait.go:185` — wait_for_replication reports acked=true on timeout. *waitForDrain: deadline branch returns true* | killed | `TestWait_overflowingTimeoutClamped` |
| M60 | `internal/grpcapi/server.go:150` — Set does not hand the write to replication (no immediate DataReplication; only anti-entropy heals). *Set: drop s.notifyLocalWrite(...)* | **killed on Linux only** | Linux: `TestExamplesDoc`, `TestReadmeQuickstart`, `TestWaitIntegration_ackedThroughRealAdapterMirror` ; darwin: none |
| M61 | `internal/grpcapi/server.go:182` — Delete is replicated as a live empty value (deleted=false in the notified record). *Delete: notifyLocalWrite(key, ver, nil, true) -> false* | **SURVIVED** | — |
| M62 | `internal/grpcapi/server.go:147` — dump failure answered with OK instead of Unavailable. *Set: ErrDumpFailed swallowed* | killed | `TestServer_errorMapping` |
| M63 | `internal/grpcapi/server.go:142` — create_if_absent flag ignored by the API (plain Set). *Set: if req.GetCreateIfAbsent() disabled* | killed | `TestServer_errorMapping` |
| M64 | `internal/grpcapi/watch.go:313` — Watch: live events not filtered against the subscription snapshot (duplicate / stale events delivered). *wants(): return true after the prefix check* | **SURVIVED** | — |
| M65 | `cmd/sup-kv/main.go:676` — shutdown skips the final dump (Flush) on SIGTERM. *shutdown: err := node.engine.Flush() -> var err error* | **killed on Linux only** | Linux: `TestExec_bothNodesDown`, `TestExec_sigtermFinalDump` ; darwin: none |
| M66 | `cmd/sup-kv/main.go:416` — drift_warn not wired into the node (drift warnings/metric dead in production). *persist.Open(.., cfg.driftWarn, ..) -> 0* | **SURVIVED** | — |
| M67 | `cmd/sup-kv/main.go:418` — Watch hub not wired to the store (no live change events in the real binary). *startEngine: drop OnChange: hub.Ingest* | **killed on Linux only** | Linux: `TestExamplesDoc` ; darwin: none |
| M68 | `cmd/sup-kv/main.go:417` — value_max_bytes knob not wired (default always used). *startEngine: drop ValueMaxBytes: cfg.valueMaxBytes* | **SURVIVED** | — |
| M69 | `cmd/sup-kv/main.go:515` — sync_response_rate / sync_chunk_bytes knobs not wired. *newReplicationAdapter: drop SyncChunkBytes and SyncResponseRate* | **SURVIVED** | — |
| M70 | `cmd/sup-kv/main.go:510` — ack_timeout_max / peer_timeout knobs not wired. *newReplicationAdapter: drop PeerTimeout and AckTimeoutMax* | **SURVIVED** | — |
| M71 | `internal/app/repl/core.go:542` — no Ack for a stale (lower-version) DataReplication (the upstream spec behaviour: "Ack is not sent"). *applyRemote: return before the Ack when the store rejected the record as older* | **SURVIVED** | — |
| M72 | `internal/app/repl/core.go:708` — local write not broadcast immediately (first DataReplication only at ack_timeout). *NotifyLocalWrite: drop out.Broadcast(DataReplication)* | killed | `TestBound_normalOneRTT`, `TestScenario_loss`, `TestWire_twoNodeHandoff` |
| M73 | `internal/app/repl/core.go:755` — periodic DataSync never re-fires after the startup listing. *syncListing: nextSync pushed 1e6 hours ahead* | killed | `TestScenario_emptyDisk`, `TestScenario_restart`, `TestSync_chunking` +1 ; Linux adds `TestLog_formatText` |

Note on M73: `TestLog_formatText` in its Linux list is the SIGTERM race of section 3.2 failing twice in a row, not an effect of the mutant.

### 8.3 Reading the result

By area (killed / total): LWW store 9/9; HLC 5/5; dump codec 3/3; persistence engine 6/8; dump file adapter 3/6; replication core 21/23; UDP codec 2/5; gRPC API 6/8 on Linux, 5/8 on darwin; `main` wiring 2/6 on Linux, **0/6 on darwin**.

The pure algorithmic core is well defended: every LWW, HLC, backoff, eviction, chunking and limiter mutant died, usually to several independent tests with exact expectations. The survivors cluster in four places — exactly the places the simulator does not reach.

1. **Deletes (M38, M53, M61).** A node can stop listing tombstones in `DataSync`, drop the `Deleted` flag in the wire encoding, or announce an API delete as a live empty value, and all 173 tests pass on both platforms. M53 and M61 produce two replicas holding *the same version with different content* — a divergence anti-entropy can never repair, because it compares versions. No simulator, real-socket, two-process or documentation test deletes a key and looks at the other node.
2. **Durability of the dump (M21, M26, M27, M31, and M20 by the back door).** The file adapter can skip both fsyncs and replace temp-file-plus-rename with an in-place write; the engine can take its snapshot before it captures the waiters it will confirm. Nothing fails. M20 (confirm before `Save`) is killed, but only by four dump-*failure* tests that see `nil` where they expect `ErrDumpFailed`; no success-path test notices a confirmation that precedes the save (`TestEngine_confirmDebounceTiming` and `TestEngine_endToEndRestart` both pass under M20). "A write confirmed to a client is never lost" is the product's core value and the least mutation-resistant property in the suite.
3. **Restart seeding (M18).** `persist.Open` can ignore the dump maximum altogether.
4. **The trust boundary and the wiring (M54, M55, M66, M68, M69, M70; M60, M65, M67 on darwin).** Decoder domain checks are unexercised; four configuration knobs can be disconnected from the node; on darwin the API can stop announcing writes, shutdown can skip the final dump and the Watch hub can be unplugged.

Weak kills worth knowing about:

* M02 (LWW compares the timestamp only) dies to one deterministic test, `TestStore_restartDominance`; none of the merge-law properties sees it (their generator never ties on `l`).
* M16 dies to a single fixture (`TestWatermark_equalCounterDomain`).
* M39 (limiter removed) is not noticed by `TestScenario_emptyDisk` or `TestExec_emptyDiskRecovery` — both assert upper bounds only.
* M60 (API never announces a write) is not noticed by any of the four `TestExec_*` two-process scenarios: their 6 s deadline is met by the 200 ms anti-entropy round.
* M73 (periodic sync disabled) is not noticed by `TestBound_partitionHeal`, `TestScenario_partitionSeconds`, `TestScenario_partitionDays`, `TestScenario_oneWay` or (at seed 1) `TestChaos_convergence` — the tests that carry the "2 × `sync_interval`" bound do not depend on sync.
* M71 (no Ack for a stale `DataReplication` — the upstream spec's wording) survives: the stale-version path is undefined by tests in either direction.
* M64 (Watch delivers events already contained in the subscription snapshot) survives although `TestWatch_exactCut` exists to forbid duplicates across the snapshot/live cut; the race window it relies on did not open in any run.

### 8.4 A proposed test, validated

For M21 the missing test was written and run in the probe copy (probe 6): 16 writers × 150 confirmed `Set`s against an engine with a recording dump store; after each `Set` returns, the most recently **completed** save must contain that key at that version or newer. Unmutated engine: 0 violations in 3 runs of 2 400 writes. With M21 applied: 9, 19 and 5 violations — 3 of 3 runs fail. (As written it takes ≈ 30–55 s under `-race` because of 3 000 filler keys; it needs tuning before adoption.)

### 8.5 Prepared, not run

M74–M76 (`/metrics` families exporting the wrong counter or a frozen value), M77–M78 (core never counts drift warnings / conflicts), M79–M80 (adapter does not mirror conflict, drift and retransmission counters), M81 (unicast replies sent to the broadcast address), M82 (slow Watch subscriber silently loses events), M83 (startup log of discarded dump records removed), M84 (dump debounce ignored), M85 (failed final dump exits 0), M86 (`Hello` does not refresh liveness). Coverage data (section 4) predicts M79, M80, M82 and M85 survive; that prediction is untested.

## 9. Test-quality smells

### 9.1 Real time

* **Capability probes with a 300 ms verdict** (`logProbeWait`, `scenarioProbeWait`, `integrationProbeWait`): a timeout is reported as "platform lacks the capability" and the test skips. Eleven tests and both socket benchmarks sit behind it (section 2).
* **Wall-clock upper bounds on short operations** in `internal/grpcapi/wait_test.go`: a 100 ms wait must return in < 400 ms (`:196`), a 120 ms per-request timeout in < 400 ms (`:511`), the zero-peer short-circuit in < 400 ms (`:131`); `TestGate_opensOnTimeout` uses a 50 ms backstop. Under `-race` on a two-core shared runner a 300 ms stall fails them. None failed here, including with `GOMAXPROCS=2`.
* **Sleeps as synchronisation**: `wait_test.go:102,153,258,336,379` (2–60 ms) script a fake mirror on a timeline; `watch_test.go:274` (10 ms).
* **Absence proven by waiting**: `watchQuietWindow` 200 ms, `scenarioEffectWait` 2 s — "nothing happened" assertions that pass vacuously when the system is slow.
* The persistence engine is the good example: `testing/synctest` bubbles, exact virtual instants, no real I/O.

### 9.2 Ports

* `freeUDPPort` / `freeTCPPort` / `integrationUDPPort` are bind-`:0`-close-reuse pickers. Observed failing (section 3.2).
* README and EXAMPLES blocks bind fixed ports (27901/27902/27911/27912 and the 280xx band). Two concurrent runs of `./cmd/sup-kv` tests on one host, or one leftover node, fail them.

### 9.3 Missing negative and boundary tests

* **No `Fuzz*` function exists** — not for the datagram decoder, the dump decoder, or the version parser. (Two harnesses were written for this audit and not run — not done.)
* Malformed datagrams: none (section 4). Maximum-size datagrams: none — the largest value any real-socket test replicates is a few dozen bytes.
* **Probe 5 (Linux, real sockets):** 1 KiB, 32 KiB and 61 440-byte values replicate in 6–9 ms. A record with a 61 440-byte value and a 5 000-byte key — legal, because nothing limits key length — makes every send fail with `sendto: message too long`, broadcast and unicast alike; the write was confirmed to its author, stays unacked forever, is retried forever, and the key never reaches the peer. No test covers key size, record size versus the 65 507-byte UDP payload, or the send-error path.
* Disk faults: only "temp file cannot be created" (section 4, section 8).
* No test restarts a node with a wall clock behind its dump, kills a lone node immediately after an acknowledged write, or kills both nodes.

### 9.4 Coupling and shape

* Tests are assertion-dense (no assertion-free tests found; about 1 100 lines with a `Fatal`/`Error` call) and table tests use named subtests. The largest single function is `TestRetransmit_ackDrainSemantics` (326 lines, ten named cases).
* Coupling to implementation detail is deliberate and mostly useful (exact ladder instants, exact limiter spacing to the nanosecond). Two places pin behaviour that **contradicts the upstream spec**, which makes the tests an obstacle rather than evidence there: `SetIfAbsent` over a tombstone returns `AlreadyExists` (`TestStore_SetIfAbsent`, `TestEngine_writeSurface`, `TestServer_errorMapping`; the spec says a deleted key is absent), and confirmation latency under a burst is asserted against `dump_max_wait` = 1 s (`TestEngine_burstBoundedConfirmation`; the spec row says `dump_debounce` plus dump time).
* No goroutine-leak check outside the `synctest` bubbles (which do detect leaks for the engine). The adapter's read goroutine, the watch hub and the gate resolver are not checked. A stdlib-only check (compare `runtime.NumGoroutine`/stack dump after cleanup with a retry loop) would respect the no-assertion-library rule.

### 9.5 Doc-contract tests — real coverage

| Document | Fenced blocks | Executed verbatim | Assertion |
|---|---|---|---|
| `README.md` | 3 | 2 (the two marked quickstarts; the build line is not) — **Linux only** | `bash -e` exit status of every command, plus the base64 value appearing somewhere in the combined output. Because the final `Get` after `kill -9`/restart runs under `-e`, the restart claim is checked — but with the peer alive, so it cannot tell "restored from the dump" from "resynced" |
| `docs/EXAMPLES.md` | 11 `bash` blocks | 10 marked — **Linux only** | per-block substring table; several are weak (`"{}"` for the snapshot round trip, `"demo/wait"` for delete-and-list, `"SERVING"`) |
| `docs/OPERATIONS.md` | 0 | — | 41 anchor substrings must be present and the trust-model heading must come first. Nothing in the document is executed; its numbers (bounds, defaults, metric names) are checked for presence, not truth |

### 9.6 CI (`.github/workflows/ci.yml`)

Runs, per native cell (ubuntu-latest, ubuntu-24.04-arm, macos-15): `go vet ./...`, `go test -race ./...`, `go mod tidy` + diff. The fourth cell cross-compiles darwin/amd64 and vets. Lint, buf and vulncheck are separate.

Never run in CI:

* benchmarks — compiled by `go vet` only; not even a `-benchtime=1x` smoke, so the two socket benchmarks can rot unnoticed (and `BenchmarkReplicationBurst` fails here at ≥ 10 000 iterations, section 10);
* more than one chaos seed; any fixed seed; `-count` > 1; `-shuffle=on`;
* any real-socket or two-process test on macOS (they skip);
* any test at all for darwin/amd64;
* the child binary under the race detector;
* coverage measurement; a scheduled soak.

Not done in this audit: a `-shuffle=on` pass and the two fuzz harnesses (written, not run).

## 10. Benchmarks

Command (the `make bench` equivalent): `go test -run '^$' -bench . -benchmem ./internal/app/persist/ ./internal/udprepl/ ./internal/grpcapi/`, once, then `-count=3`.

### 10.1 Raw numbers

darwin/arm64, Apple M3 Max, APFS (the dump path uses `F_FULLFSYNC`):

```
BenchmarkDump10kKeys-16    58   19995466 ns/op   5147550 bytes/op   5147550 dump-bytes   18938008 B/op   40104 allocs/op
(-count=3: 20552192, 20658563, 20525493 ns/op)
internal/udprepl   PASS   (BenchmarkReplicationBurst: SKIP, loopback broadcast unavailable)
internal/grpcapi   PASS   (BenchmarkConfirmationLatency: SKIP, same)
```

Without `-v` the two skipped benchmarks print nothing at all — `make bench` on a Mac silently reports one benchmark out of three.

linux/arm64 container (OrbStack VM, overlay filesystem, plain `fsync`):

```
BenchmarkDump10kKeys-16            73   17553504 ns/op   5147550 bytes/op   5147550 dump-bytes   18937049 B/op   40104 allocs/op
BenchmarkReplicationBurst-16      442    2380774 ns/op   4.000 max-unacked   420.0 writes/s   51880 B/op   420 allocs/op
BenchmarkConfirmationLatency-16   120   10459524 ns/op   10454268 latency-ns/op   21738374 max-latency-ns   692907 B/op   3576 allocs/op
-count=3:
  Dump:    17240965 / 18585397 / 18931888 ns/op
  Burst:   436.4 / 418.7 / 425.8 writes/s, max-unacked 6 / 4 / 2
  Latency: 9474543 / 9099244 / 9417623 latency-ns/op, max 14.2 / 13.1 / 14.3 ms
Fixed iteration counts for the burst benchmark:
  -benchtime=100x    401.4 writes/s   max-unacked 1
  -benchtime=1000x   410.0 writes/s   max-unacked 13
  -benchtime=5000x   443.3 writes/s   max-unacked 114
  -benchtime=10000x  FAIL: bench_test.go:77: timed out after 10s waiting for node B to serve every burst key under its written version
  -benchtime=15000x  FAIL (same)
  -benchtime=20000x  FAIL (same)
Latency at -benchtime=2000x: 16772347 ns/op mean, max-latency 333811014 ns (334 ms)
A bare `time.NewTicker(time.Millisecond)` in the same container: 1000 ticks took 2.53 s.
```

### 10.2 What each benchmark measures — and does not

| Benchmark | Measures | Does not measure |
|---|---|---|
| `BenchmarkDump10kKeys` (`internal/app/persist/bench_test.go:69`) | One real dump cycle: `store.Records()` copy + JSONL `Encode` + `persistfile.Save` (temp file, fsync, rotate, rename, directory fsync) on the real filesystem of the temp dir. In-process, single goroutine | The engine loop (debounce, waiters, concurrent writers holding the store lock); the documented disk-load figure. The dump is **5.1 MB** — values cycle 16…1024 bytes, mean ≈ 290 — not the spec's "10 000 keys × 1 KB ≈ 10 MB", and nothing approaches 60 KiB values. Memory: 18.9 MB allocated per dump for a 5.1 MB dump |
| `BenchmarkReplicationBurst` (`internal/udprepl/bench_test.go:51`) | Two adapters **in one process** over real UDP sockets on **loopback** broadcast; `b.N` distinct keys written straight into node A's store at a **1 ms ticker**, then a wait until B serves all of them; reports `b.N / elapsed` | Throughput. The writer is paced, so the figure cannot exceed 1000 and equals the ticker rate minus a convergence tail: on this container a 1 ms ticker fires every ≈ 2.5 ms and the benchmark reports ≈ 420. It involves no persistence, no gRPC, no dump confirmation, no second process, no real NIC, one hot path only (new key per write — never the "hot key" of the requirement), 42-byte values. It would report the same number for any implementation faster than the ticker |
| `BenchmarkConfirmationLatency` (`internal/grpcapi/bench_test.go:64`) | Round-trip time of a gRPC `Set` with `wait_for_replication` over **bufconn** (in-memory, no TCP) against an engine with a **fake in-memory dump store**, `Debounce: 1 ms`, `MaxWait: 50 ms`; two in-process adapters on loopback; a background writer at a 1 ms ticker | Dump-confirmation latency in any real configuration: no disk, no fsync (≈ 17–20 ms per dump at 10 k keys by the first benchmark), not the default 50 ms debounce. The background burst bypasses the engine (`store.Set` + `NotifyLocalWrite`), so it never resets the debounce window the measured writes wait in. The reported ≈ 10 ms is essentially the 1 ms test debounce plus one or two 5 ms `waitPoll` quanta of `waitForReplication` — a polling constant, insensitive to replication speed below 5 ms |

### 10.3 Are the `.planning/PROJECT.md` claims reproduced here?

* **"replication burst ~975 w/s" — not reproduced: 401–443 writes/s** on linux/arm64 in the container; not measurable on darwin (skip). The difference is the container's timer granularity, which is exactly the point: the benchmark reports its own pacing ticker. 975 on a machine with 1 ms timers is "1000 offered, minus the tail". It is evidence that the node keeps up with 1000 offered writes/s over loopback without persistence — not a throughput figure, and it cannot detect a regression until capacity falls below the offered load.
* **"confirmation latency ~10 ms under 1000 w/s" — the number is reproduced (9.1–10.5 ms mean, 13–22 ms max), the claim is not.** The measured path has a fake disk and a 1 ms debounce; the "1000 w/s" load does not pass through the component whose latency is being claimed and ran at ≈ 400/s here. With production defaults a confirmed write waits at least `dump_debounce` (50 ms) plus a real dump; under a sustained burst through the API the project's own `TestEngine_burstBoundedConfirmation` pins confirmation at up to `dump_max_wait` (1 s), because the debounce is a quiet-window timer that every write resets. At 2000 iterations the benchmark's own max latency reached 334 ms.
* **"dump at 10k keys"** — exists and is honest about what it does; about 17–21 ms per 5.1 MB dump on both platforms here.

Two further observations (ran):

* `BenchmarkReplicationBurst` **fails** at 10 000, 15 000 and 20 000 iterations on Linux — node B does not hold every key 10 s after the burst — while the backlog (`max-unacked`) grows 1 → 13 → 114 between 100 and 5 000 iterations. The default `go test -bench` picks ≈ 450–525 iterations here and never reaches that regime. Not root-caused in this audit; the harness runs both nodes with a 200 ms sync interval, so each node broadcasts its full listing five times a second and the receiver copies its whole store per listing chunk (`core.go:633`). Whatever the cause, the one benchmark that should exercise "bursts of ~1000 writes/s" stops converging in time at the same key count the dump benchmark uses.
* No benchmark runs in CI and there are no thresholds (a documented decision, D-13), so none of the above is guarded against regression.

## 11. Ranked findings

Severity reflects how much of a stated product promise currently rests on no evidence. "Acceptance" is what must be true for the finding to close.

### High

**TST-01 — `TestChaos_convergence` is flaky by construction and has already broken `master`.**
Evidence: 374 of 10 000 seeds fail (3.74 %); 23 of seeds 1–600; Linux `-count=5` failed; CI run 44351136063 (`6458adb`, 2026-11-18) red with seed 1795020540436451818, reproduced locally; every failing seed converges later (section 3.1). The test asserts convergence at the first "quiescent" instant, which with ≥ 3 nodes precedes the periodic sync that completes convergence.
Test to change/add: assert model equality at `max(last write, last partition end) + bound`, where the bound is one the protocol can honour under the drawn loss (see TST-09), not at first quiescence; run a fixed regression corpus (at least seeds 27, 160, 573, 578, 1795020540436451818, 1763500000223000669) plus N fresh seeds per run; add deletes and per-node clock offsets to the draw.
Acceptance: 0 failures over a 10 000-seed sweep; the test fails under mutants M38 and M73.

**TST-02 — Delete replication is unproven at every level.**
Evidence: M38 (`DataSync` omits tombstones), M53 (`Deleted` dropped on the wire) and M61 (API announces a delete as a live value) survive all 173 tests on both platforms; no simulator, real-socket, two-process or doc test deletes a key and inspects the peer (`simnet.Cluster` has no delete). M53/M61 yield equal versions with different content — unrepairable by sync.
Tests to add: (a) codec round trip for every message kind including `Deleted: true`, compared field by field; (b) `TestSync_chunking`/`_triggers` with tombstones in the store — the listing must carry them; (c) `Cluster.Delete` in simnet and deletes in chaos and in the partition scenarios (resurrection check after heal); (d) two processes: `Delete` on A ⇒ `Get` on B is `NotFound`, B's dump line has `"deleted":true`, and a node that was down during the delete learns it after restart.
Acceptance: M38, M53, M61 killed; (d) green on Linux.

**TST-03 — "A confirmed write is on disk" is not pinned: ordering, atomicity and fsync can all be removed.**
Evidence: M21 (snapshot before waiter capture), M26 (no data fsync), M27 (no directory fsync), M31 (in-place non-atomic write) survive on both platforms; M20 (confirm before save) dies only to dump-failure tests. Uncovered: every `Save` error path except `CreateTemp` (`dumpfile.go:101-173`), exit code 1 on a failed final dump (`main.go:246-248`). No test kills a lone node after an acknowledged write; `TestExec_bothNodesDown` stops both nodes gracefully. The spec assigns disk faults to fault injection; none exists below the `DumpStore` port.
Tests to add: (a) the containment stress test of section 8.4; (b) a gated fake `Save`: while it blocks, `Set` must not have returned; (c) a small filesystem seam under `persistfile` (create/write/sync/rename/sync-dir) with a recording-and-failing fake: assert the operation order temp-write → fsync → rotate → rename → dir-fsync, and inject an error at each step (ENOSPC on write, EIO on fsync, EXDEV/EACCES on rename) asserting `Save` fails, no temp litter, previous generations intact, API answers `Unavailable`, and a failed final dump exits 1; (d) processes: lone node, `Set` OK, SIGKILL, restart alone, `Get`; and SIGKILL of both nodes followed by lone restarts, each serving every acknowledged key.
Acceptance: M20 (by a success-path test), M21, M26, M27, M31 killed; `dumpfile.go` error branches covered.

**TST-04 — The startup gate is tested as a boolean; the property it exists for is false.**
Evidence: probe 3 (section 6.3) — the gate signal fires on the first `DataSync` datagram, the HLC does not absorb listing versions (the spec requires `Update` on each part), and a write accepted right after the gate opens is stamped below a version the neighbour already advertised and loses. The coordinator's real-stand observation (`create_if_absent` on a replaced empty node overwriting a user value on both nodes) is the same hole at API level. Gate tests use an injected closure; two-process tests only await `SERVING`.
Tests to add: (a) simulator: neighbour clock ahead, node replaced or restarted, first post-gate write on a key the neighbour holds must dominate the neighbour's version; (b) API level with a real adapter pair: on a restarted empty node `create_if_absent` for a key the peer holds must return `AlreadyExists` or `Unavailable`, never overwrite; (c) the gate stays closed until the first listing has been *reconciled*, whatever definition the fix adopts.
Acceptance: (a) and (b) fail on today's code and pass after the fix. This is a product defect surfaced by a missing test; route to the replication/API track.

**TST-05 — On darwin nothing above the in-process core is tested, and on Linux the same tests can skip silently.**
Evidence: 13 tests + 3 subtests + 2 benchmarks skip on darwin (section 2); M60, M65, M67 are killed on Linux only and all six `main`-wiring mutants survive on darwin; eleven tests skip when a 300 ms probe times out; the README's macOS block is executed only on Linux through its fallback arm; the child binary is never built with `-race`.
Changes: in CI on Linux turn the capability skip into a failure (an environment switch such as `SUPKV_REQUIRE_BROADCAST=1`, set in `ci.yml`); print a skip summary and fail the job when the count is non-zero on Linux; give darwin a working segment for the tests (the interface's directed broadcast, as the macOS quickstart already does, or `SO_REUSEPORT`) — or state in README and OPERATIONS that replication on darwin is untested; build the smoke binary with `-race` when the test binary has it.
Acceptance: Linux CI cannot go green with a skipped real-socket test; the macOS cell either runs ≥ 1 two-node real-socket test or the docs say it does not.

**TST-06 — The simulator's blind spots are where the product fails on real sockets.**
Evidence: section 5.6. Real stand (coordinator): ≈ 17 min to recover 11.7 k keys against ≈ 83 s; 14.5 min retransmission storm for a 10 k pending set. This audit: `BenchmarkReplicationBurst` fails at ≥ 10 000 keys; probe 2 shows quadratic CPU; probe 5 shows a record that can never be sent. Largest test scales: 50 keys (simulator), 20 keys (processes), values ≤ 50 bytes on the wire.
Tests to add: (a) simnet: an optional per-node inbound budget (datagrams per virtual millisecond, drop beyond it) and a per-datagram size check using the real encoder, so load-correlated loss and oversize records exist in simulation; announce local writes after a virtual dump delay; a restart that keeps the store; (b) a nightly real-socket job at envelope scale: two processes, 10 000 keys × 1 KiB, empty-node recovery within the documented bound; a 10 000-entry pending set drains within a stated time after the peer appears; `BenchmarkReplicationBurst -benchtime=20000x` converges; (c) one test replicating a 61 440-byte value, and one asserting that a record too large for one datagram is rejected at the API (requires a key/record size rule).
Acceptance: (b) green on Linux CI hardware; the bound in the docs is either met at 10 000 keys or restated.

### Medium

**TST-07 — Restart dominance is proven for a hand-built clock, not for `persist.Open`.**
Evidence: M18 survives; `TestEngine_endToEndRestart` uses `time.Now` on both sides of the restart.
Test to add: `Open` with an injected `now` returning a time *before* the dump's maximum `l`; the first post-restart `Set` must dominate every loaded version, including one sharing the watermark millisecond with a non-zero counter. Simulator: restart-with-state under a negative clock offset.
Acceptance: M18 killed.

**TST-08 — The datagram decoder has no negative tests and no fuzzing; size limits are untested.**
Evidence: M54, M55 survive; all `Undecodable` branches of `udprepl.decode`/`portMessage` are uncovered (section 4); no `Fuzz*` anywhere; probe 5.
Tests to add: a table of malformed datagrams through `decode` (empty, truncated envelope, node_id > 65535, run_id of 0/15/17 bytes, version of 0/15/17 bytes in each message kind, unknown body, body that fails to unmarshal, flipped CRC), each asserting class and that nothing reaches the store; `FuzzDecode` (decode → `HandleDatagram` → re-encode what the core emits) and `FuzzDumpDecode` (decode → verify checksums → re-encode round trip) with the seed corpus checked in and run for a fixed budget in CI.
Acceptance: M54, M55 killed; decoder branch coverage 100 %; fuzzers run ≥ 60 s in a scheduled job without findings.

**TST-09 — The convergence bounds are proven only one fault at a time, and the partition-heal bound not by its own mechanism.**
Evidence: `TestBound_partitionHeal` and `TestScenario_partitionSeconds` pass under M73; partitions never exceed `peer_timeout` in a timed test; under combined faults 8 of 10 000 seeds exceed 2 × `sync_interval` (max 88.8 s); the loss bound excludes ≥ 5 consecutive losses by the test's own comment.
Tests to add: a bound test with 3 nodes and a partition longer than `peer_timeout` in which the write is Acked by the remaining peer: convergence must not occur before the first post-heal sync tick (teeth) and must occur within the documented bound; a loss-bound test at the boundary (four losses pass, five must be covered by a restated bound). Docs: state that the bounds assume no further loss, or give the bound that holds with loss.
Acceptance: the new heal test fails under M73.

**TST-10 — The benchmarks do not measure what the project claims from them.**
Evidence: section 10. "~975 w/s" is the pacing ticker (420–443 here); "~10 ms confirmation latency" is a fake disk, a 1 ms debounce and a 5 ms poll quantum, with the burst bypassing the engine; the burst benchmark fails at ≥ 10 000 iterations; two of three benchmarks silently skip on darwin; none runs in CI.
Changes: an unpaced throughput benchmark (writes as fast as the API confirms them, real `persistfile` store, default debounce) reporting sustained writes/s and p50/p99 confirmation latency; a hot-key variant; `-benchtime=1x` smoke in CI on Linux; restate the PROJECT.md numbers with their conditions.
Acceptance: the claims in `.planning/PROJECT.md` cite a benchmark that would change if replication or dump speed changed by 2×.

**TST-11 — Three flake classes in `cmd/sup-kv` and `internal/udprepl`.**
Evidence: section 3.2; 5 of 73 Linux suite runs and 1 of 73 darwin runs failed for these reasons during the mutation batch.
Changes: one port allocator for every real-socket test (the private-window registry already in `scenario_test.go`), with bind errors retried rather than fatal; wait for health `SERVING` before sending SIGTERM in `TestLog_*` (and consider arming the signal handler before `wire()` in `main`); a deadline on the `WaitForReady` call in `TestShutdown_secondSignal`; a per-test deadline around the doc blocks.
Acceptance: `go test -race -count=20 ./cmd/sup-kv/ ./internal/udprepl/ ./internal/grpcapi/` green on Linux, three times in a row.

**TST-12 — Wiring and observability are asserted only for presence.**
Evidence: M66, M68, M69, M70 survive (four knobs disconnected); `TestMetrics_families` renders an all-zero snapshot, so a family exporting the wrong counter cannot fail; `udprepl.MergeStats`/`Retransmissions` have no in-process coverage; no test sets a numeric, duration or node-id environment variable or starts the binary without a required knob (section 4); nothing logs drift although the spec asks for a warning.
Tests to add: a table-driven binary test per knob that observes the knob's effect (a 1 KiB `--value-max-bytes` rejecting 2 KiB; a tiny `--sync-chunk-bytes` visible in `sent_total{type="sync"}`; a short `--peer-timeout` visible in the peers gauge; a tiny `--drift-warn` plus a skewed peer visible in `supkv_clock_drift_warnings_total`); the same through `SUPKV_*` variables; `/metrics` scraped after a scripted conflict, retransmission and discard, with exact values; missing-required and out-of-range exits.
Acceptance: M66, M68, M69, M70 killed; `metrics.Snapshot` asserted with non-zero, pairwise-distinct inputs.

**TST-13 — Behaviour that differs from the accepted spec is either pinned by tests or defined by none.**
Evidence: stale `DataReplication` is Acked and not answered (spec: answered, not Acked) — no test, M71 survives; `create_if_absent` over a tombstone returns `AlreadyExists` in three tests (spec: a deleted key is absent); confirmation under a burst is asserted at ≤ `dump_max_wait` 1 s (spec row: `dump_debounce` + dump time); a failed dump returns `Unavailable` and the write is *not* announced to replication (`server.go:142-150`; spec: "stays in memory and replicates") — untested either way; no HLC `Update` on `DataSync`; protocol-version mismatch logged at Debug only; conflict journal replaced by a counter.
Action: for each, record the decision, then make one test state it. Hand-off to the spec-trace track.
Acceptance: each item has a test whose name cites the decision.

### Low

**TST-14 — Watch edge paths.** M64 survives (`wants` filter removable; `TestWatch_exactCut` cannot open the race); the per-subscriber shed branch (`watch.go:298`) and all `stream.Send` error branches are uncovered. Add a deterministic cut test (event enqueued before, dispatched after the subscription) and a shed test with a subscriber buffer of 1 and a blocked stream. Acceptance: M64 killed; `watch.go:298` covered.

**TST-15 — Property-test infrastructure.** No shrinking (`rapid` is described as adopted, it is not in `go.mod`); merge-law generators never tie on `l` (M02 escapes the properties); the printed replay command fails with `./...`; one random scenario per run in `TestClock_restartProperty`. Either adopt `rapid` or update the stack notes; draw half the versions from a tiny `(l, c, node)` space as `TestVersion_byteOrder` does; print the package path in the replay hint. Acceptance: M02 killed by `TestMerge_laws`.

**TST-16 — CI and timing hygiene.** No fixed or multiple chaos seeds, no `-shuffle`, no benchmark smoke, no nightly soak, no tests on darwin/amd64; wall-clock upper bounds of 400 ms on 100–120 ms operations in `wait_test.go`; "quiet window" absence checks; fixed ports in the documented blocks. Acceptance: a scheduled workflow running a 1 000-seed sweep, `-shuffle=on -count=3`, and `-bench . -benchtime=1x`.

## Appendix — artefacts

All under `evidence/repro/tests/`: `out/baseline_race.txt`, `out/verbose_race_p1.txt`, `out/linux_verbose_race_p1.txt`, `out/flaky_darwin.txt`, `out/flaky_linux.txt`, `out/seed_sweep_1_600.txt`, `out/sweep_small.csv`, `out/sweep_nano.csv` (10 000-seed probe), `out/cover_*.out`, `out/coverfunc_*.txt`, `out/bench_*.txt`, `out/mutations.jsonl` and `out/mutations_raw/` (two raw logs per mutant and platform), `scripts/mutlist.py`, `scripts/mutlist2.py`, `scripts/mutate.py`; probe sources in `probe/sup-kv/**/zz_*_test.go` and `out/zz_probe_test.go.txt`. The scratchpad is temporary storage.
