# Phase 2: Stand and Simulator as Tests - Pattern Map

**Mapped:** 2026-11-19
**Files analyzed:** 22 (new/modified/migrated)
**Analogs found:** 19 / 22 with in-repo analogs (3 fixtures/scripts are genuinely new; their mechanisms are pinned in 02-RESEARCH.md)

All analog paths verified git-tracked (`git ls-files`, this session). Excerpt line numbers verified by direct read.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/stand/stand.go` (new, `//go:build linux`) | test-harness package (non-test) | process/sockets (exec I/O) | `cmd/sup-kv/main_test.go` + `cmd/sup-kv/scenario_test.go` helpers | exact (wholesale move source) |
| `internal/stand/stand_test.go` (new, linux) | test (scenario) | process+socket request-response | `cmd/sup-kv/scenario_test.go` `TestExec_*` | exact |
| `internal/stand/stand_red_test.go` (new, linux) | test (red scenario, env-gated) | process+socket request-response | `TestExec_emptyDiskRecovery` (bound shape) + `internal/grpcapi/gate.go` (R-03 attach point) | role-match (gating mechanism is new — RESEARCH Pattern 1) |
| `internal/stand/stand_nondarwin_test.go` (new, `//go:build !linux`) | test (loud platform report) | none (single named result) | `internal/persistfile/fsync_darwin.go` / `fsync_other.go` (build-tag pair precedent) | partial (no loud-report analog exists) |
| `internal/app/internal/simnet/simnet.go` (extend) | harness lib (non-test) | event-driven simulation | itself (`Cluster.Write`, `ReplaceNode`, `AddNode`, `Network.Send`) | exact |
| `internal/app/repl/sim_chaos_test.go` (extend) | test (simulation) | event-driven | itself | exact |
| `internal/app/repl/sim_scenario_test.go` (extend) | test (simulation) | event-driven | itself (`TestScenario_partitionDays` cadence-scaling) | exact |
| `internal/app/repl/testdata/chaos-seeds.txt` (new) | test fixture | static file | — (first in-repo testdata) | none |
| `internal/app/repl/sim_prop_test.go` or similar (new, TEST-15) | test (property + replay executor) | transform | `internal/app/clock/hlc_prop_test.go` + `internal/app/internal/testutil/rng.go` | exact (generator contract) |
| `internal/persistfile/fsops.go` (new) | adapter seam (non-test) | file-I/O | `internal/persistfile/dumpfile.go` `saveAtomically` chain | exact |
| `internal/persistfile/fsops_test.go` etc. (new) | test (fakes, order, blocking) | file-I/O | `internal/persistfile/dumpfile_test.go` + `internal/app/persist/engine_test.go` `fakeDumpPort` | exact |
| `internal/udprepl/` negative-table test (new file) | test (table, codec) | request-response (datagram in, verdict out) | `internal/udprepl/repl_test.go` `TestDiscardLog_classes` + `internal/udprepl/repl.go` `decode` | exact |
| `internal/udprepl/testdata/fuzz/FuzzDecode/` (new) | fuzz seed corpus | static file | — (format verified in RESEARCH) | none |
| `internal/udprepl/` `FuzzDecode` (new) | fuzz test | transform | reviewer's `FuzzZZDecode` shape (RESEARCH §Code Examples, consult-only foreign module) | role-match |
| `internal/app/persist/` clock-seeding test (TEST-07) | test (composition path) | file-I/O + transform | `internal/app/persist/engine.go` `Open` | exact |
| `internal/app/persist/` `FuzzDumpDecode` + `testdata/fuzz/FuzzDumpDecode/` (new) | fuzz test + corpus | transform | reviewer's `FuzzZZDumpDecode` shape (consult-only) | role-match |
| `internal/grpcapi/` `Send`-error-branch test (TEST-14) | test (stream backpressure) | streaming | `internal/grpcapi/watch_test.go` `TestWatch_slowConsumer` | exact |
| metrics-by-effect stand test (TEST-12) | test (observed effect) | request-response | `cmd/sup-kv/main_test.go` `TestMetrics_optIn` + `internal/metrics/metrics.go` `Snapshot`/`Render` | exact |
| `scripts/red-runner` (new, D-01/D-02) | script (CI runner) | batch | — (no in-repo script dir); workflow `run:` steps in `ci.yml` are the bash-step precedent | none (mechanism RESEARCH-verified) |
| `.github/workflows/scheduled.yml` (new) | config (workflow) | batch | `.github/workflows/vulncheck.yml` (trigger triad) + `ci.yml` (job shape) | exact |
| `.github/workflows/ci.yml` (tighten) | config (workflow) | batch | itself | exact |
| `cmd/sup-kv/scenario_test.go` + harness half of `main_test.go` (migrate) | test (migration source) | process/sockets | themselves — the move source, not a pattern target | exact |

## Pattern Assignments

### `internal/stand/stand.go` + `internal/stand/stand_test.go` (harness + scenarios, linux-tagged)

**Analog:** `cmd/sup-kv/scenario_test.go` (whole file) and `cmd/sup-kv/main_test.go` lines 53-352. D-03 locks "moved/extended rather than duplicated" — `package main_test` external test code cannot be imported, so the helpers move wholesale.

**Build-once binary pattern** (`cmd/sup-kv/main_test.go:69-123`):
```go
var (
	smokeDir       string
	smokeBin       string
	errSmokeBinary error
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "sup-kv-main-smoke-*")
	// ... build once into dir, run tests, remove dir ...
	smokeBin, errSmokeBinary = buildSmokeBinary(context.Background(), filepath.Join(dir, "sup-kv"))
}

func buildSmokeBinary(ctx context.Context, bin string) (string, error) {
	goTool, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("probe exec.LookPath(go): %w", err)
	}
	cmd := exec.CommandContext(ctx, goTool, "build", "-o", bin, ".")
	// capture combined output; wrap failure naming the probe
}

// loud skip naming the failed probe — never a silent pass
func smokeBinary(t *testing.T) string { ... }
```
The stand package should keep this build-once shape (a package-level `TestMain` or a `sync.Once`-guarded helper); the loud-skip becomes `t.Fatalf` on Linux per TEST-05.

**Node spawn/reap pattern** (`cmd/sup-kv/main_test.go:227-252, 255-263, 267-307`):
```go
func launchNode(t *testing.T, bin string, stderr outputCapture, env []string, args ...string) *smokeNode {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), bin, args...)
	stdout := &bytes.Buffer{}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	if err := cmd.Start(); err != nil { t.Fatalf("start %s %v: %v", bin, args, err) }
	node := &smokeNode{cmd: cmd, stdout: stdout, stderr: stderr, exit: make(chan error, 1), exited: make(chan struct{})}
	go func() { node.exit <- cmd.Wait(); close(node.exited) }()
	t.Cleanup(node.reap)
	return node
}

// signalAndWaitExit SIGTERMs and waits for a clean exit 0 (D-19 contract)
// assertExitCode checks nil==0 / signal kill==-1 loudly
```
**kill -9 pattern** (`cmd/sup-kv/scenario_test.go:369-385`):
```go
func killAndReap(t *testing.T, node *smokeNode) {
	t.Helper()
	err := node.cmd.Process.Kill() // SIGKILL semantics, never the SIGTERM contract
	if err != nil { t.Fatalf("SIGKILL %d: %v", node.cmd.Process.Pid, err) }
	var exitErr error
	waitFor(t, exitDeadline, "process exit after SIGKILL", func() bool {
		select { case exitErr = <-node.exit: return true; default: return false }
	})
	assertExitCode(t, exitErr, -1)
}
```

**Polling with hard-deadline clamp** (`cmd/sup-kv/main_test.go:150-168`) — every stand wait uses this, never a bare sleep loop:
```go
func waitFor(t *testing.T, timeout time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	if d, ok := t.Deadline(); ok {
		if hard := d.Add(-hardDeadlineMargin); hard.Before(deadline) { deadline = hard }
	}
	for time.Now().Before(deadline) {
		if cond() { return }
		time.Sleep(pollEvery) // 10ms
	}
	t.Fatalf("timed out after %v waiting for %s", timeout, what)
}
```

**Reserved port window + registry (flake-class 1 mitigation — keep intact during the move)** (`cmd/sup-kv/scenario_test.go:105-151`):
```go
const (
	scenarioPortLo    = 28100
	scenarioPortHi    = 32678
	scenarioPortTries = 32
)
var (
	scenarioPortsMu sync.Mutex
	scenarioPorts   = make(map[int]bool)
)
// scenarioUDPPort: bind candidate → claim in registry while STILL holding
// the bind → close → t.Cleanup deletes the claim
```

**Three-outcome capability probe** (`cmd/sup-kv/scenario_test.go:167-197`): `probeLoopbackSend` returns reached / not-reached (platform lacks capability) / probe-error (port race, retry). On the stand, the not-reached outcome becomes `t.Fatalf` on Linux (TEST-05) — keep the three-outcome shape, change only the severity (RESEARCH Pitfall 2). The current loud skip (`scenario_test.go:232-236`) is the T-05 defect being replaced.

**Asymmetric-port topology** (`cmd/sup-kv/scenario_test.go:202-283`): `execPair` struct (bin, udpA, udpB, grpcA, grpcB, dumpA, dumpB), `newExecPair` (probe + reserve, nothing started), `nodeArgs` (flags: `--node-id --bind-addr --broadcast-addr=127.255.255.255:P --grpc-addr --dump-path --hello-interval --sync-interval --startup-gate`), `slots(nodeID)` mirror. Node knobs are flags with `SUPKV_` env twins (`cmd/sup-kv/main.go:94-126`) — the stand passes shrunken timers as flags exactly as today.

**Scenario shape with timer-derived bounds (never magic constants)** (`cmd/sup-kv/scenario_test.go:626-671`, `TestExec_emptyDiskRecovery` — the F9 seed):
```go
// the recovery bound, derived from the flags the nodes run with
paced := time.Duration(emptyDiskKeys) * (time.Second / time.Duration(emptyDiskRate))
margin := execStartupGate + 2*execHelloInterval + 2*execSyncInterval + 2*time.Second
deadline := paced + margin
start := time.Now()
nodeB := startExecNode(t, p, execNodeBID, "--sync-response-rate="+strconv.Itoa(emptyDiskRate))
awaitServing(t, p.grpcB, servingDeadline)
waitFor(t, deadline, "the empty-disk node B serving every key within the paced bound", func() bool { ... })
t.Logf("empty-disk recovery: %d keys in %v (paced bound %v, margin %v)", ...)
```
Every D-04 bound (F9b `≤ 2×sync_interval + N/sync_response_rate + margin`, burst `≥990/s` + set-drain `≤5 s`) follows this derivation-from-config pattern. Constants used: `servingDeadline=10s`, `exitDeadline=15s`, `checkTimeout=500ms`, `rpcTimeout=5s`, `hardDeadlineMargin=250ms` (`main_test.go:53-61`).

**Convergence poll + dump-grep assertion shapes**: `execServing` (`scenario_test.go:353-363`, transient RPC errors read as not-yet), dump grep (`scenario_test.go:591-597` reads the final dump and asserts key presence — the F2c "never on B" and `"deleted":true` assertions extend this), and the discard-line poll on live stderr (`scenario_test.go:534-536` via `startExecLoggedNode` + `logBuffer`).

---

### `internal/stand/stand_red_test.go` (red tests, D-01)

**Analog:** scenario shape above + `internal/grpcapi/gate.go` (the R-03 attach point). The env-gating mechanism itself has no in-repo precedent — RESEARCH Pattern 1 is the source.

**Gate attach point** (`internal/grpcapi/gate.go:122-150`, quoted in RESEARCH; `SyncArrived` semantics at `gate.go:44-50`):
```go
func (s *Server) RunGate(ctx context.Context) {
	deadline := s.opts.StartupGate
	if deadline <= 0 { deadline = defaultStartupGate }
	backstop := time.NewTimer(deadline)
	defer backstop.Stop()
	poll := time.NewTicker(gatePollInterval)
	defer poll.Stop()
	for {
		select {
		case <-ctx.Done(): return
		case <-backstop.C: s.openGate(); return
		case <-poll.C:
			if s.opts.SyncArrived != nil && s.opts.SyncArrived() { s.openGate(); return }
		}
	}
}
```
`ServerOptions` carries exported fields on purpose: "Production injects the udprepl adapter's SyncArrived mirror (A1 semantics: any received DataSync datagram counts); tests inject fakes" (`gate.go:44-50`).

**Red-gate idiom (no in-repo precedent — RESEARCH Pattern 1, empirically verified `go test -json` event shape):**
```go
func TestRed_R01_pacedSync20k(t *testing.T) {
	if os.Getenv("SUPKV_RED_TESTS") != "1" {
		t.Skip("red test: runs only with SUPKV_RED_TESTS=1 (D-01)")
	}
	// ... assert the DOCUMENTED bound (which v1.0 code breaches) ...
}
```
Write red tests as the FULL acceptance assertions (R-03's five-point PROTO-01 acceptance, R-01's `≤ 2×sync_interval + N/sync_response_rate` bound, R-02's `≤ N/sync_response_rate + 10 s` drain + `≤3N` retransmits, P-01's `maxRecordBytes` restart survival) so the Phase 3-5 flip needs zero test edits — only the runner expectation table changes. P-01's constant: `internal/app/persist/codec.go:25` `const maxRecordBytes = 61440`; the observed failure line is `execDiscardLine = "discarded 1 undecodable dumped records at startup"` (`scenario_test.go:78`).

---

### `internal/stand/stand_nondarwin_test.go` (loud darwin report, TEST-05)

**Analog:** `internal/persistfile/fsync_darwin.go` / `fsync_other.go` — the repo's build-tag pair precedent (small platform files, one concern each). No loud-report analog exists; TEST-05's acceptance wording is the spec. The skip sites being replaced for the error-on-Linux policy: `internal/udprepl/smoke_test.go:226-270` (`probeBroadcastAddr` — three-outcome probe whose not-reached outcome is `t.Skipf`), `internal/udprepl/repl_test.go:674-709` (`probeLoopbackBroadcast`), `internal/udprepl/scenario_test.go:103-141`, `cmd/sup-kv/scenario_test.go:222-241`.

---

### `internal/app/internal/simnet/simnet.go` (D-05 extensions)

**Analog:** itself. Every extension must copy the existing seams' contracts; the package doc (`simnet.go:1-18`) is the determinism law: "starts no goroutines, takes no locks and never reads a wall clock or sleeps... every random draw flows through the one rng injected at construction... the non-test files here must not import Go's testing package."

**`Cluster.Delete` mirrors `Cluster.Write`** (`simnet.go:569-579`):
```go
func (c *Cluster) Write(i int, key string, data []byte) storage.Record {
	now := c.sched.Now()
	_, _, _ = c.nodes[i].store.Set(key, data)
	rec := c.nodes[i].store.Records()[key]
	var out port.Outbox
	c.nodes[i].core.NotifyLocalWrite(now, key, rec, &out)
	c.route(i, out.Drain())
	c.drive(i, now)
	c.writes = append(c.writes, LoggedWrite{Key: key, Rec: rec})
	return rec
}
```
`Delete(i, key)` copies this with `store.Delete` (`internal/app/storage/store.go:199-209` — returns `ErrNotFound` for missing/already-tombstoned, writes a fresh-version tombstone otherwise) and the tombstone record through the same `NotifyLocalWrite` path; extend `LoggedWrite` (or add a sibling log entry) so the pointwise-supremum model covers tombstones.

**Restart-with-preserved-state mirrors `ReplaceNode`** (`simnet.go:457-470`):
```go
func (c *Cluster) ReplaceNode(i int, runID [16]byte, offset time.Duration) {
	c.checkIndex(i, "ReplaceNode")
	gen := c.nodes[i].timerGen + 1
	c.offsets[i] = offset
	replacement := c.newClusterNode(i, c.nodes[i].id, runID)
	replacement.timerGen = gen
	c.nodes[i] = replacement
	if !c.net.Reregister(replacement.id, runID, replacement.handle, func(in port.InboundDatagram) {
		c.deliver(i, in)
	}) { panic(...) }
	c.sched.Schedule(c.sched.Now(), func() { c.drive(i, c.sched.Now()) })
}
```
A `ReplaceNodeWithState` sibling seeds the replacement's store from `records` before wiring (keep the in-place `Reregister` + timer-generation supersession exactly). Shared node construction lives in `newClusterNode` (`simnet.go:415-426`) — one wiring, never three.

**Budget/size/dump-delay join `Send`/`scheduleDelivery`/`Policy`** (`simnet.go:131-136, 241-268, 285-305`):
```go
type Policy struct {
	DropProb  float64
	DupProb   float64
	Delay     func(rng *rand.Rand) time.Duration
	Partition func(from, to string, now time.Time) bool
}

func (n *Network) Send(fromHandle, toHandle string, msg port.Message) {
	// partition check → fate draw at send time → scheduleDelivery at now+delay
}
func (n *Network) scheduleDelivery(from, to netNode, msg port.Message, seq uint64, delay time.Duration) {
	n.pending++
	n.sched.Schedule(n.sched.Now().Add(delay), func() {
		n.pending--
		// reorder observation, trace append, deliver Decoded InboundDatagram
	})
}
```
Per-node budget = a `netNode`/Cluster counter consulted in `scheduleDelivery` (refused past budget, counted in a new stats field); dump-delay = an optional per-cluster latency applied to record announcements (mirroring production's dump-then-broadcast order); size check = a `SizeOf func(port.Message) int` injected via `Policy`/`Cluster` config (RESEARCH Pitfall 7's preferred shape — keeps the non-test file's imports at stdlib+app+port, matching today's import block `simnet.go:20-32`; the test file injects the real `replproto` encoder).

**Determinism-contract doc comment to copy** (`simnet.go:428-443`, `SetClockOffset`): every new API must document "plain harness state, consumes no rng draws and schedules no events, so the event stream stays a pure function of the seed."

**Drive loop / route** (`simnet.go:533-563`) — unchanged by the extensions; delivery refusal happens inside `scheduleDelivery`, not in `route`.

---

### `internal/app/repl/sim_chaos_test.go` (D-06 corpus + stabilization)

**Analog:** itself. **Seed hook + replay contract** (`sim_chaos_test.go:29-31, 46-56`):
```go
const (
	simSeedEnv       = "SUPKV_SIM_SEED"
	chaosReplayFmt   = "replay: SUPKV_SIM_SEED=%d go test ./internal/app/repl/ -run '^TestChaos_convergence$'"
	// chaosMaxNodes=5, chaosLenMs=200_000, chaosWriteCount=40, chaosQuiesceStep=100ms ...
)
func chaosSeed(t *testing.T) int64 {
	t.Helper()
	if raw := os.Getenv(simSeedEnv); raw != "" {
		seed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil { t.Fatalf("%s=%q is not an int64 seed: %v", simSeedEnv, raw, err) }
		return seed
	}
	return time.Now().UnixNano()
}
```
The corpus extension: when no env seed, iterate `testdata/chaos-seeds.txt` (one seed per line) in subtests plus one fresh-seed run; a sibling generator test regenerates the file. Keep the printed replay contract verbatim.

**Quiescence drive (the thing TEST-01 strengthens)** (`sim_chaos_test.go:159-201, 206-216`):
```go
func chaosQuiescent(cluster *simnet.Cluster, now time.Time) bool {
	if cluster.Network().PendingDeliveries() != 0 { return false }
	for i := range cluster.Nodes() {
		if cluster.Core(i).UnackedStats(now).Count != 0 || cluster.Core(i).QueuedResponses() != 0 { return false }
	}
	return true
}
```
`runChaosScenario` drives `sched.RunUntil` in `chaosQuiesceStep` increments up to `10*syncInterval` past the last write, then asserts stores against `chaosModelStore` (`sim_chaos_test.go:222-228`, the pointwise-supremum merge over `cluster.WriteLog()`) via `recordMapsEqual` (`sim_bound_test.go:77-97`, full-record equality incl. tombstones). Stabilization per RESEARCH Pitfall 1: after quiescence, CONTINUE polling until every store equals the model (quiescence is necessary-not-sufficient under R-05 delayed heal) — test-side semantics only.

**Model assertion + fault teeth** (`sim_chaos_test.go:238-267`, `TestChaos_convergence`): assert `run.quiesced`, per-node `recordMapsEqual(store, model.Records())`, and all four fault classes non-zero — the teeth pattern the deletions-in-the-mix extension must preserve.

**RNG seed-split** (`sim_bound_test.go:31-36`): `simRNG(seed)` = PCG halves from the int64 — reuse for corpus seeds so replay is byte-identical.

---

### `internal/app/repl/sim_scenario_test.go` (TEST-09 combined-failure + long-partition)

**Analog:** itself. **Replay helper** (`sim_scenario_test.go:46-48`): `scnReplay(seed, test)` prints the per-test replay command. **Quiesce helper** (`57-72`): `scnQuiesce` leaves the clock AT the quiescence instant so the caller measures elapsed virtual time off `sched.Now()` — the long-partition bound assertion (`not before first post-heal sync, not later than the bound`) measures this way.

**Cadence-scaling trick for long virtual horizons** (`sim_scenario_test.go:343-414`, `TestScenario_partitionDays` — the direct template for the long-partition scenario):
```go
// Virtual-time economics: scale ONLY the periodic cadence — hello_interval
// and the scan granularity to one sync_interval, peer_timeout to three
// hello rounds (the documented 5s/1s ratio kept) — while every semantic
// asserted is cadence-independent.
daysCfg := repl.Config{
	HelloInterval:      syncInterval,
	PeerTimeout:        3 * syncInterval,
	RetransmitInterval: syncInterval,
}
policy := simFixedDelay(func(from, to string, now time.Time) bool {
	pair := (from == "n1" && to == "n2") || (from == "n2" && to == "n1")
	return pair && !now.Before(partStart) && now.Before(partEnd)
})
```
with divergence teeth before heal (`404-406`: stores must NOT already agree at the heal instant), `scnQuiesce` + `scnAssertConverged` after, and per-key winner assertions. The long-partition scenario derives its bound from `boundPartitionHeal` (`sim_bound_test.go:118-125`): `2*syncInterval + (keys-1)*floor(1s/rate)` — the review's bound arithmetic in named-function form ("the assertion and the documented expression stay one artifact"). Shared constants: `syncInterval = 30 * time.Second` (`sync_test.go:29`), `at()` virtual-time base (`retransmit_test.go:59-62`).

---

### `internal/app/repl` property shrinking + replay executor (TEST-15)

**Analog:** `internal/app/clock/hlc_prop_test.go` (the property-suite precedent — subtests calling `testutil.NewRNG(t)` per run) and `internal/app/internal/testutil/rng.go:44-57`:
```go
func NewRNG(t *testing.T) *rand.Rand {
	t.Helper()
	seedNano := *seedFlag // -seed flag; 0 = wall-clock draw
	if seedNano == 0 { seedNano = time.Now().UnixNano() }
	hi, lo := uint64(seedNano>>seedHalfBits), uint64(seedNano)&(1<<seedHalfBits-1)
	t.Logf("property seed: %d,%d (replay: go test ./... -run '%s' -args -seed=%d)",
		hi, lo, t.Name(), seedNano)
	return rand.New(rand.NewPCG(hi, lo))
}
```
The printed replay hint `go test ./... -run '%s' -args -seed=%d` is the KNOWN-BROKEN command TEST-15 fixes (wrong for packages without the flag) — the new property test prints a working, package-scoped replay command (SUPKV_SIM_SEED env style or per-package `-args`) AND executes it from a test via `os/exec`. Shrinking is dependency-free (no `rapid`): generators draw half the versions from a small `(l, c, node)` space so counterexamples shrink naturally. No new imports beyond stdlib + in-repo packages.

---

### `internal/persistfile/fsops.go` + fakes/order tests (D-07, TEST-03)

**Analog:** `internal/persistfile/dumpfile.go` — the seam target whose real order IS the TEST-03 order. **The exact sequence to pin** (`dumpfile.go:97-113, 118-134, 149-157, 169-176`):
```go
func saveAtomically(path, prevPath string, data []byte) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, dirMode)          // 1. MkdirAll
	if err != nil { return err }
	tmpName, err := syncedTemp(dir, data)     // 2. CreateTemp → Write → fsync → Close
	if err != nil { return err }
	err = placeFile(path, prevPath, tmpName)  // 3. rotate main→.prev (Rename) → rename tmp→main
	if err != nil { _ = os.Remove(tmpName); return err }
	return syncDir(dir)                       // 4. open dir → fsync
}
```
The `fsops` interface extracts write/fsync/rotate/rename/dir-fsync behind a small seam so `FileStore.Save` calls it; recording fakes assert the ORDER, failing fakes inject at each step. `Save`'s mutex (`dumpfile.go:35-43, 62-66`) stays — the blocking-`Save` fake proves `Set` does not return while the dump stands via the engine's synchronous dump path (`internal/app/persist/engine.go` `Run` serializes every Save; `engine_test.go:52-71` `runEngine`/`startDumper` drive it).

**Fake/injection style to copy** — recording fake from `internal/app/persist/engine_test.go:247-297`:
```go
var errFakeSave = errors.New("fake dump save failure")
type fakeSave struct { at time.Time; dump []byte }
type fakeDumpPort struct {
	saves     []fakeSave
	saveErr   error   // injected failure
	// ...
}
func (f *fakeDumpPort) Save(dump []byte) error {
	if f.saveErr != nil { return f.saveErr }  // fail-first-injection shape
	f.saves = append(f.saves, fakeSave{at: time.Now(), dump: bytes.Clone(dump)})
	return nil
}
```
and table-driven failure style from `internal/persistfile/dumpfile_test.go:97-154` (`TestFileStore_fallbackChain`: table of corruption shapes → subtest → assert chain outcome) and `:160-192` (failure injection via `os.Chmod` read-only dir, `t.Cleanup` restore). `port.DumpStore` interface (`internal/port/dump.go`) is what engine-level fakes implement; the fsops fakes are persistfile-local.

**`persist.Open` clock seeding (TEST-07)** — the real composition path, `internal/app/persist/engine.go:122-152`:
```go
func Open(nodeID uint16, now func() time.Time, driftWarn time.Duration,
	dumps port.DumpStore, storeCfg storage.Config, engineCfg Config,
) (eng *Engine, discarded int) {
	// decode := first generation whose Decode succeeds (per-record faults discard)
	dumps.Load(decode)
	if records == nil { records = make(map[string]storage.Record) }
	initialL, initialC := Watermark(records)
	clk := clock.NewClock(now, initialL, initialC, driftWarn)
	store := storage.NewStore(nodeID, clk, storeCfg)
	mergeRecords(store, records)
	return New(store, dumps, engineCfg), discarded
}
```
`now` is already injectable — the test seeds a dump whose max version `l` exceeds the injected now, writes locally via the returned engine/store, asserts the first local write dominates all loaded versions. No seam changes needed. `mergeRecords` skips config-shrunk oversized records (`engine.go:157-166`) — relevant to P-01's red test too.

---

### `internal/udprepl` negative table + `FuzzDecode` (D-08, TEST-08)

**Analog:** `internal/udprepl/repl.go:725-761` (`decode`) — the function under test with the fixed verdict vocabulary:
```go
func decode(datagram []byte) port.InboundDatagram {
	var env replproto.Envelope
	if len(datagram) == 0 || proto.Unmarshal(datagram, &env) != nil {
		return port.InboundDatagram{Status: port.Undecodable}
	}
	if env.GetNodeId() > maxNodeIDWire || len(env.GetRunId()) != wireIDSize {
		return port.InboundDatagram{Status: port.Undecodable}
	}
	in := port.InboundDatagram{ ProtoVersion: env.GetProtoVersion(), NodeID: uint16(env.GetNodeId()), /*nolint G115: check above proved fit*/ }
	copy(in.RunID[:], env.GetRunId())
	if crc32.ChecksumIEEE(env.GetBody()) != env.GetCrc32() { in.Status = port.BadCRC; return in }
	var body replproto.Body
	if err := proto.Unmarshal(env.GetBody(), &body); err != nil { return port.InboundDatagram{Status: port.Undecodable} }
	msg, ok := portMessage(&body)
	if !ok { return port.InboundDatagram{Status: port.Undecodable} }
	in.Status = port.Decoded; in.Msg = msg; return in
}
```
Verdict vocabulary: `Decoded`/`BadCRC`/`Undecodable` (`internal/port/repl.go:16-26`); counter struct `repl.DiscardStats{BadCRC, Undecodable, BadProtoVersion, OwnEcho, DuplicateNodeID}` (`internal/app/repl/core.go:205-211`); adapter surface `Adapter.Discards()` (`internal/udprepl/repl.go:388`).

**Corrupted-datagram construction to copy** (`internal/udprepl/repl_test.go:623-665`, `TestDiscardLog_classes`):
```go
// bad CRC: an encoded Hello with one body byte corrupted
datagram, err := udprepl.EncodeDatagramForTest(port.Hello{}, nodeIDB, runIDB)
datagram[len(datagram)-1] ^= corruptByte

// wrong proto_version: hand-built envelope, CRC correct — the core's version
// check only runs past a CRC-passing envelope
body, _ := proto.Marshal(&replproto.Body{Msg: &replproto.Body_Hello{Hello: &replproto.Hello{}}})
mismatch, _ := proto.Marshal(&replproto.Envelope{
	ProtoVersion: repl.SupportedProtoVersion + 1,
	NodeId: uint32(nodeIDB), RunId: runIDB[:],
	Crc32: crc32.ChecksumIEEE(body), Body: body,
})
```
The negative table extends this construction per row (bad CRC / truncated / wrong proto_version / bad run_id / oversize / empty key / tombstone-with-data), each asserting its verdict class + the counter. Dump-decode per-record discard shape (`internal/app/persist/codec.go:97-119`): `Decode` returns `(records, discarded, err)` — generation-level faults error, per-record faults `discarded++` and continue; `decodeRecord` (`codec.go:128+`) is total on any bit pattern. The tombstone-with-data row and the four-level deletion pin ride `DataReplication.Deleted` (`repl.go:772-778` wire mapping carries `Deleted` per message).

**Fuzz shape (RESEARCH-verified; no in-repo fuzz test yet):** `FuzzDecode(f)` seeds with each valid message type encoded (`encodeDatagram(3, run, msg)` shape from the reviewer's probe) + empty bytes; the fuzz body asserts verdict invariants (Decoded never carries nil Msg, etc.). Corpus files: `testdata/fuzz/FuzzDecode/<entry>`, two-line `go test fuzz v1` + `[]byte("...")` format. PR CI runs plain `go test` (seed corpus only); scheduled runs `-fuzz=FuzzDecode -fuzztime=...` (one fuzz target per package — `FuzzDumpDecode` lives in `internal/app/persist` to avoid the single-match regexp collision, RESEARCH Pitfall 4).

---

### `internal/grpcapi` Send-error-branch test (TEST-14)

**Analog:** `internal/grpcapi/watch_test.go:483-546` (`TestWatch_slowConsumer`) — the existing backpressure proof the `Send`-error branch extends:
```go
// The subscriber stops reading here. Real writes keep completing on the
// store — every one must succeed (store liveness under a wedged subscriber)
// — while the hub's dispatch overflows the buffer-2 channel and sheds.
for i := range watchSlowWrites {
	_, err := ts.client.Set(ctx, &apiproto.SetRequest{...})
	if err != nil { t.Fatalf("Set ... with a wedged subscriber: want OK (store liveness)") }
}
// resumed reader collects the verdict
select {
case err := <-got:
	assertCode(t, "Recv on the shed watch stream", err, codes.ResourceExhausted)
case <-time.After(watchShedDeadline):
	t.Fatal("shed verdict did not arrive within the deadline")
}
```
Existing siblings to extend, not duplicate: `TestWatch_snapshotThenLive` (92), `TestWatch_exactCut` (198), `TestWatch_replicatedChange` (335). The missing piece is the handler-side `Send` error branch (`internal/grpcapi/watch.go`, 410 lines) — deterministic snapshot-then-live cut + slow-subscriber disconnect exist; drive `Send` into an error and assert its handling.

---

### Metrics-by-effect test (TEST-12)

**Analog:** `cmd/sup-kv/main_test.go:828-853` (`httpGetBody` + `TestMetrics_optIn`) for the live-node HTTP assertion, and `internal/metrics/metrics.go:95-97, 164, 181-198` for the exact-value surface:
```go
func Render(w io.Writer, families []Family) error   // exact Prometheus text
func NewHandler(snapshot func() []Family) http.Handler
type Sources struct{...}; func Snapshot(s Sources) []Family  // pulls repl.DiscardStats, udprepl.SentByType, persist.DumpStats
```
The stand scenario (`/metrics` after a conflict+retransmit+discard run) asserts exact values rendered from `metrics.Snapshot` families — opt-in via `--metrics-addr`, fetched like `httpGetBody` does today.

---

### `scripts/red-runner` (D-01/D-02)

**Analog:** none in-repo (no `scripts/` dir exists). Bash-step precedent: `ci.yml`'s multi-step `run:` blocks (e.g. the Buf gates, ci.yml lines 34-44). Mechanism (RESEARCH-verified on go 1.27.4):
```bash
# D-01: machine-readable expected-red assertion — parse events, never text
SUPKV_RED_TESTS=1 go test -race -count=1 -json ./internal/stand/ ./internal/persistfile/ ... > out.json
# expected-red = event with Action=="fail" AND Test=="<name>"; Action=="skip" for an expected-red name is an ERROR

# D-02: mutants via temp overlay — production tree never mutated
cp internal/app/repl/core.go "$TMP/core.go.mut"
sed 's/<original>/<mutant>/' "$TMP/core.go.mut" > "$TMP/core.go"
printf '{"Replace": {"%s/internal/app/repl/core.go": "%s/core.go"}}' "$PWD" "$TMP" > "$TMP/ov.json"
SUPKV_SIM_SEED=$seed go test -overlay="$TMP/ov.json" ./internal/app/repl/ -run '^TestChaos_convergence$' -count=1   # must FAIL
```
Expectation table maps test name → `red` (flip to `green` in Phases 3-5, one line). Mutant targets: "sync without tombstones" → `core.go:751 syncListing`/`:771 listingChunks`; "periodic sync disabled" → the periodic sync emission in `Advance` (`core.go:724-750`). Evidence goes to `.planning/phases/02-…/red-evidence/`.

---

### `.github/workflows/scheduled.yml` (D-09, TEST-16)

**Analog:** `vulncheck.yml` (whole trigger triad, lines 6-13) + `ci.yml` (job shape). **Trigger triad to copy** (`vulncheck.yml:6-13`):
```yaml
on:
  schedule:
    - cron: "0 6 * * 1" # weekly, Monday 06:00 UTC
  pull_request:
    paths:
      - go.mod
      - go.sum
  workflow_dispatch:
```
scheduled.yml uses `schedule: - cron: <nightly>` + `workflow_dispatch:` (minus the paths filter). **Job shape to copy** (`ci.yml:19-21, 54-60, 83`):
```yaml
jobs:
  lint:
    runs-on: ubuntu-latest
    timeout-minutes: 15 # P1-WR-01: a wedged job must not hold a runner for the 6h default
  gate:
    runs-on: ${{ matrix.runs-on || matrix.name }}
    timeout-minutes: 20 # P1-WR-01: race suite + tidy bound
    strategy:
      fail-fast: false
```
Steps use `actions/checkout@v7` + `actions/setup-go@v7` with `go-version-file: go.mod`, `permissions: contents: read`, `concurrency:` group (ci.yml:10-16). Jobs per D-09: chaos-1000seeds (`-shuffle=on -count=3`), bench-smoke (the `make bench` package set, `Makefile:26-27`: `go test -run '^$' -bench . ./internal/app/persist/ ./internal/udprepl/ ./internal/grpcapi/`, with `-benchtime=1x`), red-run (the runner script), race-flake (`-race -count=20` over cmd/sup-kv + udprepl + affected; generous `timeout-minutes` per RESEARCH A1), nightly-stand (10 000 × 1 KiB section-15 scenarios). Dispatchable seed-count parameter for the one-off 10k proof (RESEARCH Open Question 1).

**`ci.yml` tightening (D-09):** keep the 4-cell matrix (ubuntu-latest, ubuntu-24.04-arm, macos-15, darwin-amd64-cross); make the Linux socket tests mandatory/non-skippable (the stand package's Linux cell fails on skip — TEST-05); note the check-run names ("lint", "gate (<cell>)") are branch-protection contexts — any matrix change is a ruleset change (ci.yml header comment).

## Shared Patterns

### Determinism contract (simnet extensions)
**Source:** `internal/app/internal/simnet/simnet.go:1-18` (package doc), `:428-443` (`SetClockOffset` contract)
**Apply to:** every simnet API addition (D-05)
```go
// The package starts no goroutines, takes no locks and never reads a wall
// clock or sleeps: virtual time is the only time ... every random draw flows
// through the one rng injected at construction ... Determinism contract:
// plain harness state, consumes no rng draws and schedules no events, so
// for any fixed sequence of calls the event stream stays a pure function
// of the seed.
```

### Red-test gating (D-01)
**Source:** RESEARCH Pattern 1 (no in-repo precedent); `go test -json` event shape verified
**Apply to:** all four Critical red tests (R-01, R-02, R-03, P-01)
```go
if os.Getenv("SUPKV_RED_TESTS") != "1" { t.Skip("red test: runs only with SUPKV_RED_TESTS=1 (D-01)") }
```
Runner asserts `Action=="fail" && Test=="<name>"`; `Action=="skip"` for an expected-red name is an error.

### Timer-derived bounds, never magic constants
**Source:** `cmd/sup-kv/scenario_test.go:626-653` (paced + margin derivation); `internal/app/repl/sim_bound_test.go:100-125` (named bound functions); `internal/app/repl/sync_test.go:29` (`syncInterval = 30 * time.Second`)
**Apply to:** every stand timing assertion (F9/F9b/burst/F2c) and TEST-09 bound. Wall-clock precision is never asserted; every deadline is configured-timers + explicit margin.

### Polling with test-deadline clamp
**Source:** `cmd/sup-kv/main_test.go:150-168` (`waitFor`), `:312-325` (`awaitServing` — fresh client per poll)
**Apply to:** all stand waits (convergence polling, dump-grep polling, drain polling).

### Loud-skip probe → error-on-Linux (TEST-05)
**Source:** `internal/udprepl/smoke_test.go:226-270`, `internal/udprepl/repl_test.go:674-709`, `cmd/sup-kv/scenario_test.go:167-197`
**Apply to:** `internal/stand` probes. Keep the three-outcome shape (reached / not-reached / probe-error-with-retry); only the not-reached-on-Linux outcome becomes `t.Fatalf`. Non-Linux never runs scenarios (build tag) and reports via the named darwin test.

### Table-driven tests with named subtests + `t.Parallel`
**Source:** `internal/persistfile/dumpfile_test.go:97-154`; `internal/app/repl/sim_chaos_test.go:254-266` (fault-teeth table)
**Apply to:** negative decoder table, fsops failure-injection matrix, per-parameter TEST-12 tables, chaos corpus iteration.

### Recording/failing fake style
**Source:** `internal/app/persist/engine_test.go:247-365` (`fakeDumpPort`: recorded saves, injected `saveErr`, `errFakeSave` marker), `internal/persistfile/dumpfile_test.go:28-51` (marker oracle)
**Apply to:** fsops recording/failing fakes, any `port.DumpStore` engine-level fake, blocking-`Save` fake.

### Seeded replay contract
**Source:** `internal/app/repl/sim_chaos_test.go:30-31, 46-56` (`SUPKV_SIM_SEED` env + printed replay command); `internal/app/internal/testutil/rng.go:44-57` (`-seed` flag + printed seed)
**Apply to:** chaos corpus runs, new property tests (TEST-15 fixes the broken `go test ./... -args -seed=` hint to a package-scoped command and executes it from a test).

### CI workflow conventions
**Source:** `vulncheck.yml:6-13` (schedule + dispatch triad), `ci.yml:19-21, 54-60` (`timeout-minutes` with P1-WR-01 justification comments, `fail-fast: false`), `ci.yml:10-16` (permissions/concurrency), checkout@v7/setup-go@v7 with `go-version-file: go.mod`
**Apply to:** scheduled.yml (all five jobs) and the ci.yml tightening.

### Depguard boundary awareness
**Source:** `internal/app/internal/simnet/simnet.go:20-32` (imports: stdlib + internal/app/* + internal/port only); RESEARCH Pitfall 7
**Apply to:** the simnet size-check extension — inject `SizeOf func(port.Message) int` from the test layer (which may import `replproto`) rather than importing the adapter into the harness's non-test file. The fsops seam lives under `persistfile` (adapter), never under `internal/app` — the lint gate would reject it.

## No Analog Found

| File | Role | Data Flow | Reason | Substitutes |
|------|------|-----------|--------|-------------|
| `internal/app/repl/testdata/chaos-seeds.txt` | fixture | static | first in-repo testdata file; format is Claude's discretion (D-06) | generator test is itself the format spec |
| `testdata/fuzz/FuzzDecode/`, `testdata/fuzz/FuzzDumpDecode/` corpora | fixture | static | no fuzz corpus exists yet; `go test fuzz v1` format RESEARCH-verified | RESEARCH §Fuzz harness seed-corpus seeding; reviewer probes (foreign module, consult-only) |
| `scripts/red-runner` | script | batch | no scripts/ dir exists; the overlay + `-json` mechanisms have no in-repo precedent | RESEARCH Patterns 1-2 (empirically verified go 1.27.4); ci.yml `run:` blocks as bash style |
| `internal/stand/stand_nondarwin_test.go` | test | — | no loud-platform-report test exists | TEST-05 acceptance wording; `fsync_darwin.go`/`fsync_other.go` build-tag pair |

## Metadata

**Analog search scope:** `cmd/sup-kv/`, `internal/app/repl/`, `internal/app/internal/simnet/`, `internal/app/internal/testutil/`, `internal/app/persist/`, `internal/persistfile/`, `internal/udprepl/`, `internal/grpcapi/`, `internal/metrics/`, `internal/app/storage/`, `internal/port/`, `.github/workflows/`, `Makefile`
**Files scanned:** ~40 project files (15 read in full or targeted depth, all excerpted above; `.gomodcache` excluded throughout)
**Tracked-source gate:** all named analogs verified via `git ls-files` (this session) — no gitignored mirrors referenced
**Pattern extraction date:** 2026-11-19
