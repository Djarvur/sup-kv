# sup-kv simulator guide

How the deterministic simulator works, how to run and replay its scenarios,
and which Makefile entry points exercise them. The simulator is the proof
surface for replication correctness under chaos: unit tests cannot show
convergence bounds under loss, duplication, reordering and partitions — the
simulator proves them on the virtual clock, and the real-socket suites
re-prove the critical ones on real binaries.

## What the simulator is

`internal/app/internal/simnet` is a deterministic, seeded virtual network:

- **Discrete-event scheduler.** Events execute in the strict (Time, Seq)
  total order — every event with an earlier virtual time first, events at
  the same virtual time in Schedule-call (insertion) order. `Scheduler.Now`
  advances only as `RunUntil` executes events.
- **No goroutines, no locks, no wall clock.** The package starts no
  goroutines and never reads a wall clock or sleeps; virtual time is the
  only time. (The one scheduler lock exists so the dump engine's own
  goroutine can arm timers through the scheduling surface — `RunUntil` stays
  the single driver.)
- **The same production handlers.** The cluster harness drives the same
  `internal/app/repl` handlers the production UDP adapter drives — no
  simulation mode anywhere. A sim-side reimplementation of any protocol
  decision would prove nothing about production; the determinism argument IS
  that the same handlers run.
- **One seeded rng.** Every random draw (faults, delays) flows through the
  one rng injected at construction, so the same seed reproduces the whole
  run byte for byte. `Cluster.TraceDigest` fingerprints the event stream for
  same-seed equality assertions.

## Running scenarios

The simulator suites live in `internal/app/repl` (the simulator-facing
tests; `go list ./...` keeps simnet itself internal to the app tree):

- **Failure-scenario suite** — `go test ./internal/app/repl/ -run 'TestScenario_'`.
  The suite covers clock jumps on a live node, a far-ahead node, partitions
  of seconds and of days, a multi-day partition at the eviction boundary,
  one-way connectivity, datagram loss, the 1000 writes/s burst, empty-disk
  recovery, restart resync, the three-node join, the replaced node with
  stale state, combined failures, and the dump-notify and equal-version
  tiebreak paths.
- **Chaos convergence** — `go test ./internal/app/repl/ -run 'TestChaos_convergence'`:
  randomized topologies (2–5 nodes, random write/delete mixes, random
  partitions and delays over ~200 virtual seconds) driven by the committed
  seed corpus below.
- **Real-socket twins** — `internal/udprepl`'s scenario tests and the
  `internal/stand` harness re-prove the critical scenarios against real
  sockets and the real binary (Linux only; see `internal/stand`'s package
  doc). The simulator and the real-socket suites are complements, not
  substitutes.

## Seed replay

Every scenario and chaos run seeds itself and prints its replay command on
start and on failure — a failing run reproduces byte for byte:

```text
replay: SUPKV_SIM_SEED=<seed> go test ./internal/app/repl/ -run '^TestChaos_convergence$'
```

- **`SUPKV_SIM_SEED=<seed>`** (decimal int64) forces the seed; without it a
  run draws a fresh wall-clock seed. Set the variable and run the printed
  command to replay exactly.
- **Fixed corpus + fresh seeds.** `TestChaos_convergence` always replays the
  committed fixed corpus `internal/app/repl/testdata/chaos-seeds.txt` — 16
  seeds, one decimal line each and nothing else, so even `head -1` of the
  file is a valid `SUPKV_SIM_SEED` replay argument — then appends a small
  batch of fresh wall-clock seeds (default 1; the scheduled jobs raise it
  via `SUPKV_SIM_FRESH_SEEDS`). Fixed seeds catch known-sensitive regressions
  on every run; fresh seeds keep finding new ones. The corpus is regenerated
  byte-exactly on any machine from its fixed generator constant
  (`SUPKV_SIM_CORPUS_REGEN`); the generator test owns the format.
- **Determinism gate.** The extension tests and the three-node scenario
  assert same-seed `TraceDigest` equality in addition to convergence, so a
  change that alters the event stream under a fixed seed fails loudly even
  when the final state still converges.

## Adding a scenario

A scenario composes the simnet harness pieces and asserts a bound; nothing
else is needed:

1. **Build the cluster.** `simnet.NewCluster(sched, rng, n, repl.Config,
   policy)` wires n nodes (node_id 1..n) over one `Scheduler`, one seeded
   rng and one fault `Policy` (the loss/duplication/reordering/partition
   profile). Construction schedules each node's first timer pump, so the
   first `RunUntil` fires the startup Hello and DataSync.
2. **Drive virtual time.** `sched.RunUntil(t)` executes events; between
   calls the scenario mutates plain harness state — `Cluster.Write`,
   `Cluster.Delete`, `Cluster.SetClockOffset` (the wall-clock seam for jump
   and far-ahead scenarios), `Cluster.ReplaceNode` /
   `Cluster.ReplaceNodeWithState` (the restart seam), `Cluster.AddNode`.
   None of these consume rng draws or schedule events, so the event stream
   stays a pure function of the seed.
3. **Assert the bound.** Every scenario asserts its documented acceptance
   expression — the convergence bounds and behaviors the operator reference
   states — against the final stores (`Cluster.Store(i)`) and the core's
   exported counters; it does not settle for "eventually something
   happened". Deletion-heavy scenarios close with a deterministic drain
   sweep so the final state (empty or tombstone-only) is guaranteed without
   weakening model equality.
4. **Keep it deterministic.** No wall clock, no goroutines, no time-based
   waiting: a scenario that passes only sometimes is a bug in the scenario,
   and the same-seed digest assertions will find it.

## Makefile entry points

Three targets exercise the simulator and its real-socket siblings. Each
target's own Makefile comment is the authoritative statement of its flags —
the summaries below are orientation, not the contract:

- **`make red-run`** — the expected-red gate. `scripts/red-runner` opens the
  red env gate itself (`SUPKV_RED_TESTS`), runs the affected packages under
  `go test -race -count=1 -json` and asserts every row of
  `scripts/red-expectations.tsv` from the parsed events: an expected-red row
  must FAIL, a `green` row must not, and an unexpected failing test is an
  error — a broken normal test can never hide behind the expected reds. The
  mutation stage then builds two sed-edited copies of production sources
  through `go test -overlay` ("sync without tombstones", "periodic sync
  disabled") and requires the chaos convergence test to kill each mutant on
  fixed corpus seeds while the control passes. `--skip-mutants` is for local
  iteration only; the scheduled job runs the full script.
- **`make race-soak`** — the flake detector: the affected package set under
  the race detector for `COUNT` repetitions (default 20;
  `COUNT=1` is the cheap local sanity pass). The red tests skip without the
  env gate, so the soak repeats only the ordinary suites.
- **`make perf`** — the operator-facing threshold run (Linux only, no race
  detector, `-p 1`): the perf rows over the real-binary stand harness with
  the thresholds applied from `internal/testutil/perf.go`, then the ordinary
  stand suite once more with no thresholds. Thresholds live only in that
  const block; a threshold change without the matching
  [docs/PERFORMANCE.md](PERFORMANCE.md) edit fails `make test`. Plain
  `go test` (and `make test`) skips every perf row by design; CI keeps the
  threshold-free `make bench-smoke` and never runs `make perf`.
