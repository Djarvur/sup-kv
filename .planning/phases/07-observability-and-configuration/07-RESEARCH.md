# Phase 7: Observability and Configuration - Research

**Researched:** 2026-11-23
**Domain:** Go 1.27 operator observability, one-point configuration validation, honest metric definitions, exit-code contract, depguard architecture boundary, `udprepl`/`main` decomposition
**Confidence:** HIGH for the current-state map (every claim read from the named file this session); MEDIUM for design defaults marked `[ASSUMED]` in the Assumptions Log

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** One lifecycle log catalog, emitted through `slog` with stable levels and fields: a startup entry carrying the binary version and the effective configuration (secrets-free); one entry each for the loaded dump generation, a dump failure, gate opening, peer appear/disappear/restart and drift warnings. A fatal bind error at `SUPKV_LOG_LEVEL=error` produces an ERROR entry and a non-zero exit, never a silent exit 1. The Spec-mandated entries (dump failure, start without usable dumps, foreign proto_version, duplicate node_id) sit at the documented levels. — [auto] [Lifecycle logs] Q: what does the catalog cover? → Selected: startup+config, load generation, dump failure, gate, peers, drift at stable levels (O-01/S-07). — **Reversibility:** reversible.
- **D-02:** Clock-drift warnings match the Spec: a received version leading system time by more than `drift_warn` yields one WARN plus the drift metric; the runbook text describes exactly this condition in the shipped binary (no fictional conditions). — [auto] [Drift] Q: does the runbook match the binary? → Selected: WARN + metric at `drift_warn`, runbook tied to the code path (S-06). — **Reversibility:** reversible.
- **D-03:** One config package and one validation function own ranges and relations: non-positive values, the `sync-chunk-bytes` overflow (the Phase-5 IN-04 library-level guard lands here too), the `retransmit-interval=1ns` busy loop, `peer_timeout ≤ hello_interval`, `ack_timeout > ack_timeout_max`, and ignored positional arguments. Every failure exits 2 naming the parameter through the usage path; the table test covers every rule. Flag parsing moves out of `main.go` (which stays wiring). — [auto] [Validation] Q: where does validation live? → Selected: one config package + one function, exit 2, table-tested (O-02/Q-07). — **Reversibility:** costly — public flags/env.
- **D-04:** Fold the deferrals into this validation catalog: `sync_chunk_bytes` gains bounds (Phase-3 WR-02) and `value_max_bytes` keeps its datagram-ceiling relation with the library-level codec guard added (Phase-5 IN-04); the apply-failure WARN becomes rate-limited per peer/class (Phase-3 IN-01); the unbound-listing case gets its own discard class (Phase-4 New-2) and joins the metrics family. — [auto] [Deferrals] Q: do the routed deferrals land here? → Selected: yes, in the validation/metrics catalog (prior ledgers). — **Reversibility:** reversible.
- **D-05:** Metric definitions are honest and each is test-confirmed: the conflict counter counts only the displacement of a local unconfirmed record by another value; a failed send is not counted as sent; `process_resident_memory_bytes` reports current RSS (not peak); the dump-duration and drift-counter semantics are stated; the four Phase-4 counters (`ListingDropped`, `SyncStreamDropped`, `ListedKeysEvicted`, `NotificationsRegressed`) get their production surface. — [auto] [Metrics] Q: how are definitions confirmed? → Selected: one documented definition per metric + a test per definition (O-03/14.4). — **Reversibility:** reversible.
- **D-06:** Exit codes are distinct and ordered: usage errors 2, startup failure 1, final-dump failure a distinct code (3); a `Serve` error still runs the bounded final dump; the signal handler is installed before dump load and bind; `--help` lists the five env-only variables. — [auto] [Exit codes] Q: which codes? → Selected: 2 usage / 1 startup / 3 final dump, ordered shutdown (O-05). — **Reversibility:** costly — operator contract.
- **D-07:** The dependency direction is machine-enforced with additive depguard rules: adapters may not be imported by `app`/`port`, and the observed cross-imports (`metrics`→`udrepl`, `grpcapi`→`persistfile`/`*persist.Engine`) are removed or explicitly allowed with a written justification; any boundary change is a visible `.golangci.yml` diff (the vendored body stays byte-unmodified; project rules are additive keys only). — [auto] [Boundaries] Q: which imports are ruled? → Selected: direction + the observed cross-imports (Q-05). — **Reversibility:** reversible.
- **D-08:** `internal/udprepl` splits socket, event loop, mirror and codec into separate units; `cmd/sup-kv/main.go` stays wiring only (config parsing extracted per D-03, shutdown testable). Public behavior and tests stay unchanged; the depguard direction holds after the split. — [auto] [Decomposition] Q: how far to split? → Selected: four units in udprepl + wiring-only main (Q-07). — **Reversibility:** reversible.

### Claude's Discretion

- Exact catalog field names/levels, the final-dump exit code choice (3 unless research grounds another), rule names in `.golangci.yml`, and the split file names in `udprepl`.

### Deferred Ideas (OUT OF SCOPE)

- Dead-code deletion (`lastLocal`/`cover`, exported `Parse`, `ownHandle`) is Phase 9 per Q-09 routing; do not absorb it.
- Performance thresholds and the flake work are Phase 8.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description (from REQUIREMENTS.md) | Research Support |
|----|-----------------------------------|------------------|
| OBSV-01 | The node logs its lifecycle (O-01) | Current log sites mapped in "Current State Map"; catalog and typed event seams in D-01 design; tests in Validation Architecture |
| OBSV-02 | Configuration is validated in one place (O-02, Q-07) | Existing checks enumerated; `internal/config` + one `Validate` rule table designed; every rule and exit-2 path mapped |
| OBSV-03 | Metric definitions are honest and test-confirmed (O-03, 14.4) | Defect evidence for conflict/sent/RSS/duration/drift; definitions table + one test per definition; four Phase-4 counters wired |
| OBSV-04 | Exit codes and shutdown are distinct and ordered (O-05) | Exit table 0/1/2/3, signal-before-load/bind order, Serve-error dump path, bounded final dump, `--help` env-only list |
| OBSV-05 | Clock-drift warnings match the Spec (S-06) | Current both-directions/pre-absorption behavior quoted; fix in `clock.Update`; `drift_warn < max_clock_lead` relation; runbook alignment |
| OBSV-06 | Spec-mandated log entries exist (S-07) | Four mandated entries located and level-checked; missing level/coverage gaps (proto mismatch second branch) identified |
| OBSV-07 | The dependency direction is machine-enforced (Q-05) | Current depguard rules and every real import mapped; additive rules + probe method; allowed-exception justification |
| OBSV-08 | `repl.go` and `main.go` are decomposed (Q-07) | `repl.go` responsibility map with line ranges; four-file split that preserves package/tests; testable `main` orchestration |
</phase_requirements>

## Summary

Phase 7 is a composition-and-semantics phase, not a new-technology phase: every building block (slog, `flag`, Prometheus-text rendering, the metrics adapter, the `Sources` seam, the per-class WARN limiter, the dump retry ladder) already exists. The work is to make the observable surface honest and to put one boundary around configuration. The code currently proves this: `main.go` is 1 104 lines (contract says wiring only), `udprepl/repl.go` is 1 417 lines mixing socket, loop, mirror and codec, and validation is scattered across six `check*` functions plus `registerConfig`.

The highest-value finding is a latent contradiction between two locked behaviors: the default `max-clock-lead` equals the default `drift_warn` (both 60 s), and `version.Validate` rejects anything strictly beyond `now + max_clock_lead`, while drift requires a received version leading by *more than* `drift_warn`. With both defaults, a version that would trigger a drift warning is rejected first as `bad_version`; the Spec-mandated drift WARN is unreachable out of the box. Phase 3's research already recorded this as assumption A1 and explicitly handed it to Phase 7. The recommendation is a validation relation `drift_warn < max_clock_lead` plus a 2 m default for `max-clock-lead`.

The second cluster is metric honesty. `supkv_merges_conflicts_total` currently counts every displaced **or rejected** version (storage `join` returns a Conflict for ordinary overwrites and tie-break losses); the review's engine evidence defines the correct denominator as displacement of a **pending (unacked) local version**, which the replication core can compute from its `pending` set. `countSent` runs before `WriteToUDP`, so failed sends are counted; `process_resident_memory_bytes` uses `ru_maxrss` (peak); and the drift counter reads the pre-update local-vs-wall divergence in both directions, not the received version's lead. Each needs a small code change plus a behavior-level test.

The remaining work is mechanical but adversarial to tests: extract `internal/config`, split `internal/udprepl` into four files, add typed event seams for peer/drift/gate lifecycle, add four metric families, and add depguard rules — while the frozen red tests, the red-runner's mutation seds, the stand's fixed-port window and the strict lint gate stay untouched. The plan should sequence the conflict/drift semantics **before** touching the stand metrics test and should run the red gate as a phase gate, because the red-runner's package list includes `internal/grpcapi`, `internal/stand` and `internal/persistfile` and treats any unexpected failure in those packages as an error.

**Primary recommendation:** Do the phase as five ordered workstreams — (1) config package + validation rule table, (2) lifecycle events + exit/shutdown orchestration, (3) metric semantics + the five counter surfaces, (4) depguard rules + `udprepl` split, (5) docs/registry ripple — with the conflict/drift semantic changes landing in the same commit as their unit tests and the adapted `TestStand_metricsByEffect`.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Configuration parsing, precedence and validation | Process wiring (`cmd` + `internal/config`) | — | One parse/validate point before any bind; `main` owns exit codes and output |
| Lifecycle log rendering | Process wiring (`cmd/main`) | Adapters (`grpcapi`, `udprepl`) | Core emits typed events; only adapters/wiring touch `slog` and stderr |
| Lifecycle event detection (peer/gate/drift) | Application core (`internal/app/repl`, `internal/grpcapi`) | Adapter mirror | The state transitions exist only in the core/resolver; rendering is a wiring concern |
| Metric value production | Application core (`repl`, `persist`, `storage`) | Adapter mirrors (`udprepl`, `persistfile`) | Counters are mutated under the single-driver lock; the adapter publishes atomic snapshots |
| Metric exposition | Observability adapter (`internal/metrics`) | — | One closed `Sources` set, one text renderer, no registry |
| Exit-code decision | Process wiring (`cmd/main`) | — | Only `main` may call `os.Exit`; packages return typed outcomes |
| Dependency direction | Lint gate (`.golangci.yml`) | — | Machine-enforced, visible diff; not a convention |
| RSS platform probe | Observability adapter (`internal/metrics`) | OS (`/proc`, `getrusage`) | OS knowledge belongs at the adapter edge; build-tagged files |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go standard library `log/slog` | Go 1.27.4 | Structured lifecycle logging, JSON/text handlers, level filtering | Already installed once in `setupLogging`; `slog.SetDefault` also redirects the `log` package (verified in Go source) |
| Go standard library `flag` | Go 1.27.4 | Flag registration, `--help`, precedence via env-derived defaults | Existing contract: flag > env > default; no third-party CLI dependency |
| Go standard library `log/slog` + `json` | Go 1.27.4 | Log-record assertions in tests | Parse JSON records instead of matching formatted text |
| Go standard library `runtime`, `syscall`, `os` | Go 1.27.4 | RSS/CPU probes and build-tagged platform code | No new dependency; `/proc/self/statm` is the Linux current-RSS source |
| `internal/metrics` (repo) | current tree | Prometheus text exposition and `Sources` seam | Already renders sorted families, fixed labels and runtime families; add families, not a client library |
| `internal/port` (repo) | current tree | Cross-layer value types and error classes | `LoadOutcome`, `Generation`, `FailureClass` already carry the dump observability vocabulary |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `go tool golangci-lint` | 2.15.0 (pinned in `go.mod`) | Depguard boundary + strict lint gate | Every task; `make lint` also runs `config verify` |
| `scripts/red-runner` + `scripts/red-expectations.tsv` | repo | Frozen red-gate verification and mutation probes | Phase gate only; never edit |
| `internal/stand` harness | repo | Real-socket end-to-end assertions (metrics/log/exit) | When a behavior crosses process/socket boundaries |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `/proc/self/statm` for Linux current RSS | `syscall.Getrusage` `ru_maxrss` | `ru_maxrss` is peak, not current (`man 2 getrusage`: "maximum resident set size"); fails OBSV-03 |
| Typed event callbacks for peer/drift/gate | Match `slog` strings in tests | Brittle to formatting/field order and to text-vs-JSON mode; violates D-05's testability intent |
| A separate `internal/config` package | Keep `config` in `cmd/sup-kv` | `main.go` stays >1 000 lines and the validation table can only be tested as package `main` |
| Moving `udprepl` units into subpackages | Four files in the same package | Subpackages would change unexported visibility and force test edits; D-08 requires tests unchanged |
| Prometheus `client_golang` | Existing `internal/metrics` renderer | Adds a runtime dependency and bypasses the closed `Sources` seam; forbidden by the minimal-graph policy |

**Installation:** none. This phase adds no package to `go.mod`. The direct runtime require set stays empty; the `tool` directives and their indirect graph are unchanged.

**Version verification:** `go version` reports `go1.27.4 linux/amd64`; `go tool golangci-lint --version` reports `2.15.0` built with `go1.27.4`; `python3 --version` reports `3.15.2` (red-runner); `git --version` reports `2.43.0`.

## Package Legitimacy Audit

No external packages are installed or recommended by this phase. `go.mod`'s shipped runtime graph remains empty (only the `tool` directives and indirect requires already present). The Package Legitimacy Gate is therefore not applicable; any plan task that proposes adding a module dependency is out of scope and must be rejected.

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.

## Current State Map (OBSV-01…08)

Every quote below was read from the named file in this session. Line ranges are inclusive.

### OBSV-01 / OBSV-05 / OBSV-06 — Lifecycle Logging

#### Startup entry with version and effective config — absent

`main` installs the handler, warns about a non-wildcard bind, wires, arms signals and serves. There is no log call carrying `version`, `commit` or the resolved configuration before `wire`:

> `cmd/sup-kv/main.go:245-253`
> ```go
> 	// The one slog handler install (D-06/D-07) precedes every service
> 	// goroutine: the udprepl Log tap, the adapter's discard events and the
> 	// log-package calls redirected through slog.SetDefault all flow through
> 	// this handler for the process's whole life.
> 	setupLogging(cfg)
>
> 	// XPORT-09: a non-wildcard bind is loud before any service starts — the
> 	// node would send broadcasts but never receive them.
> 	warnOnNonWildcardBind(cfg.bindAddr)
> ```
> `[VERIFIED: cmd/sup-kv/main.go:245-253]`

The build identity exists but only `--version` prints it:

> `cmd/sup-kv/main.go:47-54`
> ```go
> // version and commit are injected at link time via -ldflags "-X
> // main.version=... -X main.commit=..." (D-18) — enough to identify a
> // running build during incident triage; "dev" stands in for any field the
> // build did not stamp.
> var (
> 	version = "dev"
> 	commit  = "dev" //nolint:gochecknoglobals // the -X ldflags mechanism (D-18) can only write package-level vars.
> )
> ```
> `[VERIFIED: cmd/sup-kv/main.go:47-54]`

> `cmd/sup-kv/main.go:1015-1020`
> ```go
> // printVersion reports the ldflags-injected build identity (D-18): version,
> // commit and the Go runtime version, "dev" standing in for unstamped
> // fields.
> func printVersion() {
> 	fmt.Printf("sup-kv %s %s %s\n", version, commit, runtime.Version())
> }
> ```
> `[VERIFIED: cmd/sup-kv/main.go:1015-1020]`

#### Loaded dump generation — only failures are logged

`logLoadStats` is silent for `LoadPrompt` and `LoadMain` by design; a successful load produces no lifecycle entry:

> `cmd/sup-kv/main.go:611-632`
> ```go
> // logLoadStats reports one startup load in the one-line-per-class policy
> // (D-02, PERSIST-02/PERSIST-09): a Backup emits exactly one WARN naming the
> // main generation's path, cause and preservation error; an Unusable exactly
> // one ERROR carrying one attribute group per candidate that actually failed
> // — one or two, never a phantom — and both are silent for Prompt and Main.
> // A non-zero sweep count is one INFO and a sweep error one WARN; a clean
> // start therefore logs nothing here.
> func logLoadStats(stats persist.LoadStats) {
> 	switch stats.Outcome {
> 	case port.LoadBackup:
> 		logBackupFallback(stats)
> 	case port.LoadUnusable:
> 		logUnusableGenerations(stats)
> 	case port.LoadPrompt, port.LoadMain:
> 	}
> 	if stats.Swept > 0 {
> 		slog.Info("Orphan temp sweep removed files", "count", stats.Swept)
> 	}
> 	if stats.SweepErr != nil {
> 		slog.Warn("Orphan temp sweep failed", "detail", stats.SweepErr.Error())
> 	}
> }
> ```
> `[VERIFIED: cmd/sup-kv/main.go:611-632]`

The load report already carries everything the missing entry needs:

> `internal/app/persist/engine.go:157-165`
> ```go
> type LoadStats struct {
> 	Loaded     bool
> 	Outcome    port.LoadOutcome
> 	Generation port.Generation
> 	Failures   []port.GenerationFailure
> 	Decoded    DecodeStats
> 	Swept      int
> 	SweepErr   error
> }
> ```
> `[VERIFIED: internal/app/persist/engine.go:157-165]`

#### Dump failure — present at ERROR, one per failed attempt

> `cmd/sup-kv/main.go:586-594`
> ```go
> 			// One ERROR per failed attempt with the class label the metric uses
> 			// (D-05, PERSIST-05): a static message plus attributes, never dynamic
> 			// text, and the engine's dumper goroutine only formats and logs.
> 			OnDumpFailure: func(err error, class port.FailureClass) {
> 				slog.Error("Dump failed", "class", class.String(), "detail", err.Error())
> 			},
> ```
> `[VERIFIED: cmd/sup-kv/main.go:586-594]`

The cadence is already bounded by the retry ladder (100 ms doubling to 5 s), so a persistent failure cannot flood the log:

> `internal/app/persist/scheduler.go:251-277`
> ```go
> // handleDumpFailure applies one failed attempt's decay and report (D-05,
> // PERSIST-05): the class counters, the next ladder rung fixed at this
> // failure's instant — so a write arriving during the backoff registers a
> // waiter the armed retry covers instead of postponing it — the classified
> // failure the next flush answers with instead of forcing an extra attempt
> // (WR-02), and exactly one OnDumpFailure callback carrying the underlying
> // error and the adapter's class. It runs on the dumper goroutine once per
> // failed attempt; the callback must not block.
> func (s *scheduler) handleDumpFailure(err error) {
> 	class := port.FailureClassOf(err)
> 	s.mu.Lock()
> 	s.retryDelay = nextRetry(s.retryDelay)
> 	s.retryAt = s.cfg.Now().Add(s.retryDelay)
> 	s.lastErr = err
> 	s.mu.Unlock()
> 	switch class {
> 	case port.FailureDiskFull:
> 		s.stats.diskFull.Add(1)
> 	case port.FailurePermission:
> 		s.stats.permission.Add(1)
> 	case port.FailureOther:
> 		s.stats.other.Add(1)
> 	}
> 	if s.cfg.OnDumpFailure != nil {
> 		s.cfg.OnDumpFailure(err, class)
> 	}
> }
> ```
> `[VERIFIED: internal/app/persist/scheduler.go:251-277]`

#### Fatal bind error — exits, but silently at `SUPKV_LOG_LEVEL=error`

`wire` errors (replication bind, gRPC listen, metrics listen/TLS) go through `log.Fatalf`:

> `cmd/sup-kv/main.go:255-258`
> ```go
> 	node, listener, err := wire(cfg)
> 	if err != nil {
> 		log.Fatalf("sup-kv: %v", err)
> 	}
> ```
> `[VERIFIED: cmd/sup-kv/main.go:255-258]`

`setupLogging` installs `slog.SetDefault`. Go's `slog.SetDefault` redirects the `log` package through a handler writer at `LevelInfo`:

> `$GOROOT/src/log/slog/logger.go:62-74` (Go 1.27.4)
> ```go
> func SetDefault(l *Logger) {
> 	defaultLogger.Store(l)
> 	// If the default's handler is a defaultHandler, then don't use a handleWriter,
> 	// or we'll deadlock as they both try to acquire the log default mutex.
> 	// The defaultHandler will use whatever the log default writer is currently
> 	// set to, which is correct.
> 	// This can occur with SetDefault(Default()).
> 	// See TestSetDefault.
> 	if _, ok := l.Handler().(*defaultHandler); !ok {
> 		capturePC := log.Flags()&(log.Lshortfile|log.Llongfile) != 0
> 		log.SetOutput(&handlerWriter{l.Handler(), &logLoggerLevel, capturePC})
> 		log.SetFlags(0) // we want just the log message, no time or location
> 	}
> }
> ```
> `[VERIFIED: $GOROOT/src/log/slog/logger.go:62-74]`

`log.Fatalf` calls `log.Output` at `LevelInfo`; with `SUPKV_LOG_LEVEL=error` the handler drops it and `os.Exit(1)` runs with no stderr record — exactly review O-01's "silent exit 1".

#### Gate opening — absent

`RunGate` opens on the earlier of the sync flip or the backstop and changes health, with no log call:

> `internal/grpcapi/gate.go:163-198`
> ```go
> func (s *Server) RunGate(ctx context.Context) {
> 	deadline := s.opts.StartupGate
> 	if deadline <= 0 {
> 		deadline = defaultStartupGate
> 	}
> 	backstop := time.NewTimer(deadline)
> 	defer backstop.Stop()
> 	poll := time.NewTicker(gatePollInterval)
> 	defer poll.Stop()
> 	for {
> 		select {
> 		case <-ctx.Done():
> 			return
> 		case <-backstop.C:
> 			s.openGate()
> 			return
> 		case <-poll.C:
> 			if s.opts.SyncArrived != nil && s.opts.SyncArrived() {
> 				s.openGate()
> 				return
> 			}
> 		}
> 	}
> }
>
> // openGate opens the gate exactly once and mirrors the transition into the
> // health endpoint under the same branch. The resolver is the only opener,
> // but its two opening paths (sync flip, backstop) can both fire in one
> // select iteration — the CAS keeps the health flip single-shot with the
> // flag store.
> func (s *Server) openGate() {
> 	if s.gateOpen.CompareAndSwap(false, true) {
> 		s.healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
> 	}
> }
> ```
> `[VERIFIED: internal/grpcapi/gate.go:163-198]`

#### Peer appear / disappear / restart — absent

`handleHello` distinguishes first-seen, repeat and new-run; `evictPeers` removes expired rows. Neither calls a log seam:

> `internal/app/repl/core.go:1357-1409`
> ```go
> 	discovery := len(c.peers) == 0
> 	known, remembered := c.peers[in.NodeID]
> 	if remembered {
> 		switch {
> 		case known.runID == in.RunID:
> 			// A repeat Hello is idempotence: refresh the liveness stamp, emit
> 			// nothing, never a second listing.
> 			known.handle = in.Source
> 			known.lastHello = now
> 			c.peers[in.NodeID] = known
> 			return
> 		case known.seen(in.RunID):
> 			// A run_id this row carried before and no longer has is an
> 			// alternating identity — a named configuration error (PROTO-06):
> 			// counted, WARNed through the shared per-(node_id, class) limiter,
> 			// dropped with no listing and no row change, so a flapping
> 			// process cannot force a full listing per datagram.
> 			c.discards.DuplicateNodeID++
> 			c.cfg.OnDiscard(DiscardDuplicateNodeID, fmt.Sprintf(
> 				"node_id %d announced a previously seen run_id from handle %q", in.NodeID, in.Source))
> 			c.warnRateLimited(now, in.NodeID, DiscardDuplicateNodeID, fmt.Sprintf(
> 				"sup-kv: configuration error: node_id %d alternated to a previously seen run_id from handle %q, datagram discarded (PROTO-06)",
> 				in.NodeID, in.Source,
> 			))
> 			return
> 		}
> 		// A new run under this node_id no longer stands behind the old run's
> 		// listing: its partial rounds and queued FIFO are dropped, so neither
> 		// the bounded listing queue nor the accumulator inherits dead state
> 		// (CR-01). Its RETAINED listed keys stay, though: retainListedKeys
> 		// replaces the whole per-peer set on the new run's first complete
> 		// listing, so dropping them here would reopen the listed-but-unarrived
> 		// hole (F14) for the restarted peer's keys until its replacement
> 		// listing arrives (WR-02) — the two-interval expiry in
> 		// expireListedKeys bounds the wait when no replacement ever comes.
> 		// The outgoing current run
> 		// joins the row's bounded history before the row moves on, which is
> 		// what makes a later return to it an alternation.
> 		c.dropPeerSyncStreams(in.NodeID)
> 		c.dropListing(listingDst{nodeID: in.NodeID, runID: known.runID})
> 		c.dropOwedPeer(in.NodeID)
> 		known.pushSeenRun(known.runID)
> 	}
> 	known.handle = in.Source
> 	known.runID = in.RunID
> 	known.lastHello = now
> 	c.peers[in.NodeID] = known
> 	if !remembered {
> 		// A first-seen id is the one Hello shape that changes the peer id
> 		// set; the peer-id mirror publishes on this generation (A-01).
> 		c.peerGen++
> 	}
> 	c.greetAndList(now, in, known, remembered, out)
> ```
> `[VERIFIED: internal/app/repl/core.go:1357-1409]`

> `internal/app/repl/core.go:2749-2762`
> ```go
> func (c *Core) evictPeers(now time.Time, out *port.Outbox) {
> 	expired := make([]uint16, 0, len(c.peers))
> 	for id := range c.peers {
> 		if !now.Before(c.peers[id].lastHello.Add(c.cfg.PeerTimeout)) {
> 			expired = append(expired, id)
> 		}
> 	}
> 	for _, id := range expired {
> 		c.dropListing(listingDst{nodeID: id, runID: c.peers[id].runID})
> 		c.dropOwedPeer(id)
> 		delete(c.peers, id)
> 		// An evicted id leaves the set: the peer-id mirror republishes so the
> 		// write-time capture stops counting a peer that can never Ack (A-01).
> 		c.peerGen++
> ```
> `[VERIFIED: internal/app/repl/core.go:2749-2762]`

There is no `OnPeerEvent`-style seam in `repl.Config`; the only lifecycle-adjacent output is the one-shot no-Hello WARN:

> `internal/app/repl/core.go:2334-2343`
> ```go
> // warnIfNoPeer surfaces a node that has heard no peer since it started
> // (XPORT-09, D-07): when no Hello has arrived by firstAdvance +
> // cfg.PeerTimeout, one WARN names the condition — the likely causes are a
> // bind address that receives no broadcasts, a wrong broadcast address, or a
> // silent segment. The warning fires at most once per process lifetime
> // (peerWarned latches it) because the deaf state persists and a per-interval
> // repeat would be noise, not signal; it never changes serving behavior.
> // runs. Counting from the first Advance rather than from construction keeps the
> // deadline on the driver's own clock, so a core built long before its loop
> // runs does not warn before it ever had a chance to read a datagram.
> ```
> `[VERIFIED: internal/app/repl/core.go:2334-2343]`

#### Drift — metric only, no WARN, both directions, pre-absorption

The apply path counts `drift != 0` and logs nothing:

> `internal/app/repl/core.go:1525-1535`
> ```go
> 	conflicts, drift, err := c.store.ApplyRemote(msg.Key, msg.Record)
> 	if err != nil {
> 		c.cfg.Log(fmt.Sprintf("sup-kv: applying replicated record for key %q from handle %q: %v", msg.Key, source, err))
> 		// Not applied: no Ack, no supersession — never confirm a record the
> 		// store refused.
> 		return
> 	}
> 	c.mergeStats.Conflicts += uint64(len(conflicts))
> 	if drift != 0 {
> 		c.mergeStats.DriftWarnings++
> 	}
> ```
> `[VERIFIED: internal/app/repl/core.go:1525-1535]`

The counter's documented definition admits both signs and the pre-update comparison:

> `internal/app/repl/core.go:436-446`
> ```go
> // MergeStats counts the remote-merge observability signals (OBS-01, D-10):
> // Conflicts is every displaced-or-rejected version the store reported — an
> // LWW loser is an ordinary overwrite, counted and never journaled — and
> // DriftWarnings is every apply whose drift signal was nonzero: the store
> // clock gates the signal at its drift-warn threshold (zero drift is within
> // tolerance, any nonzero return is beyond it), so the counter mirrors
> // exactly the warnings storage.Merge already decided.
> type MergeStats struct {
> 	Conflicts     uint64
> 	DriftWarnings uint64
> }
> ```
> `[VERIFIED: internal/app/repl/core.go:436-446]`

`Clock.Update` evaluates `driftBetween(c.l, ptMs, …)` **before** absorbing the remote reading, and `driftBetween` returns both signs:

> `internal/app/clock/clock.go:119-124`
> ```go
> func (c *Clock) Update(remoteL int64, remoteC uint16) (l int64, counter uint16, drift time.Duration) {
> 	c.mu.Lock()
> 	defer c.mu.Unlock()
> 	remoteC &= counterMask
> 	ptMs := c.now().UnixMilli()
> 	drift = driftBetween(c.l, ptMs, c.driftWarn)
> ```
> `[VERIFIED: internal/app/clock/clock.go:119-124]`

> `internal/app/clock/clock.go:159-175`
> ```go
> func driftBetween(l, ptMs int64, warn time.Duration) time.Duration {
> 	if warn <= 0 {
> 		return 0
> 	}
> 	// Clamp both operands into the representable band first: the raw
> 	// difference of arbitrary int64 operands can itself wrap, and even a
> 	// difference that fits overflows below when scaled to nanoseconds.
> 	// Clamping never distorts a representable divergence — both operands
> 	// saturate to the same bound, so only already-unrepresentable readings
> 	// change.
> 	l = clampDriftRange(l)
> 	ptMs = clampDriftRange(ptMs)
> 	drift := time.Duration(clampDriftRange(l-ptMs)) * time.Millisecond
> 	if drift > warn || drift < -warn {
> 		return drift
> 	}
> 	return 0
> }
> ```
> `[VERIFIED: internal/app/clock/clock.go:159-175]`

#### Spec-mandated entries — status matrix

| Spec entry | Current site | Level | Gap |
|-----------|--------------|-------|-----|
| Dump failure | `main.go:591-593` `slog.Error("Dump failed", …)` | ERROR | none |
| Start without usable dumps | `main.go:656-668` `slog.Error("No usable dump generation", …)` | ERROR | none |
| Foreign `proto_version` | `core.go:1026-1034` (`ProtoVersionMismatch` branch) and `core.go:1036-1040` | WARN in the first branch; **silent** in the second | second branch needs the same counted+WARN route or must be proven unreachable |
| Duplicate `node_id` | `core.go:1041-1063` (two paths) and `core.go:1374-1380` (alternation) and `core.go:1713-1719` (unbound listing) | WARN | the unbound-listing path currently reuses `duplicate_node_id`; New-2 splits it |

> `internal/app/repl/core.go:1026-1063`
> ```go
> 	case port.ProtoVersionMismatch:
> 		// The adapter judged the envelope's version before decoding the body
> 		// (D-07: a foreign version's body need not follow this version's CRC
> 		// contract), so there is no message — only the counted, attributed
> 		// discard, WARNed at most once per (node_id, class) per interval.
> 		c.discards.BadProtoVersion++
> 		c.cfg.OnDiscard(DiscardBadProtoVersion, in.Source)
> 		c.warnDiscard(now, in.NodeID, DiscardBadProtoVersion)
> 		return
> 	}
> 	if in.ProtoVersion != SupportedProtoVersion {
> 		c.discards.BadProtoVersion++
> 		c.cfg.OnDiscard(DiscardBadProtoVersion, in.Source)
> 		return
> 	}
> 	if in.NodeID == c.nodeID {
> 		if in.RunID == c.runID {
> 			c.discards.OwnEcho++
> 			c.cfg.OnDiscard(DiscardOwnBroadcast, in.Source)
> 			return
> 		}
> 		c.discards.DuplicateNodeID++
> 		c.cfg.OnDiscard(DiscardDuplicateNodeID, fmt.Sprintf("node_id %d at handle %q", in.NodeID, in.Source))
> 		c.warnRateLimited(now, in.NodeID, DiscardDuplicateNodeID, fmt.Sprintf(
> 			"sup-kv: configuration error: node_id %d announced by another run from handle %q, datagram discarded (REPL-08)",
> 			in.NodeID, in.Source,
> 		))
> 		return
> 	}
> 	if known, ok := c.peers[in.NodeID]; ok && in.RunID == known.runID && in.Source != known.handle {
> 		c.discards.DuplicateNodeID++
> 		c.cfg.OnDiscard(DiscardDuplicateNodeID, fmt.Sprintf("node_id %d handles %q/%q", in.NodeID, known.handle, in.Source))
> 		c.warnRateLimited(now, in.NodeID, DiscardDuplicateNodeID, fmt.Sprintf(
> 			"sup-kv: configuration error: node_id %d known at handle %q announced again from handle %q, datagram discarded (REPL-08)",
> 			in.NodeID, known.handle, in.Source,
> 		))
> 		return
> 	}
> ```
> `[VERIFIED: internal/app/repl/core.go:1026-1063]`

### OBSV-02 — Configuration Validation

#### What exists

All knobs are declared in one place, but parsing and validation live in `main`:

> `cmd/sup-kv/main.go:99-137`
> ```go
> // The SUPKV_ env twins of the flags (D-16, the published operational
> // contract): one per knob, resolved as the flag's default value.
> const (
> 	envNodeID             = "SUPKV_NODE_ID"
> 	envBindAddr           = "SUPKV_BIND_ADDR"
> 	envBroadcastAddr      = "SUPKV_BROADCAST_ADDR"
> 	envDumpPath           = "SUPKV_DUMP_PATH"
> 	envGRPCAddr           = "SUPKV_GRPC_ADDR"
> 	envStartupGate        = "SUPKV_STARTUP_GATE"
> 	envStartupGateEmpty   = "SUPKV_STARTUP_GATE_EMPTY"
> 	envWaitTimeout        = "SUPKV_WAIT_TIMEOUT"
> 	envValueMaxBytes      = "SUPKV_VALUE_MAX_BYTES"
> 	envSyncChunkBytes     = "SUPKV_SYNC_CHUNK_BYTES"
> 	envHelloInterval      = "SUPKV_HELLO_INTERVAL"
> 	envPeerTimeout        = "SUPKV_PEER_TIMEOUT"
> 	envAckTimeout         = "SUPKV_ACK_TIMEOUT"
> 	envAckTimeoutMax      = "SUPKV_ACK_TIMEOUT_MAX"
> 	envRetransmitInterval = "SUPKV_RETRANSMIT_INTERVAL"
> 	envSyncInterval       = "SUPKV_SYNC_INTERVAL"
> 	envSyncResponseRate   = "SUPKV_SYNC_RESPONSE_RATE"
> 	envMaxClockLead       = "SUPKV_MAX_CLOCK_LEAD"
> 	envDumpDebounce       = "SUPKV_DUMP_DEBOUNCE"
> 	envDumpMaxWait        = "SUPKV_DUMP_MAX_WAIT"
> 	envDriftWarn          = "SUPKV_DRIFT_WARN"
>
> 	// The structured-logging env knobs (D-06/D-07) are env-only — no flag
> 	// twins: they configure the process's own output format, not its
> 	// service behavior.
> 	envLogLevel  = "SUPKV_LOG_LEVEL"
> 	envLogFormat = "SUPKV_LOG_FORMAT"
>
> 	// The metrics/debug endpoint knobs (D-01/D-02): the listen address has
> 	// a flag twin with NO default — unset means no listener at all; the
> 	// certificate triple is env-only and all-or-nothing.
> 	envMetricsAddr     = "SUPKV_METRICS_ADDR"
> 	envMetricsCert     = "SUPKV_METRICS_CERT"
> 	envMetricsKey      = "SUPKV_METRICS_KEY"
> 	envMetricsClientCA = "SUPKV_METRICS_CLIENT_CA"
> )
> ```
> `[VERIFIED: cmd/sup-kv/main.go:99-137]`

The five env-only variables are exactly `SUPKV_LOG_LEVEL`, `SUPKV_LOG_FORMAT`, `SUPKV_METRICS_CERT`, `SUPKV_METRICS_KEY`, `SUPKV_METRICS_CLIENT_CA` (the metrics **address** has a flag twin, so it is not env-only). `[VERIFIED: cmd/sup-kv/main.go:124-137]`

`finalizeConfig` is the current validation entry point, but it is a sequence of separate checks:

> `cmd/sup-kv/main.go:432-462`
> ```go
> // finalizeConfig validates what only the parsed flags can decide — the
> // node-id range and the required knobs — and applies the resolved node id.
> func finalizeConfig(cfg *config, nodeID uint64, envErr error) error {
> 	if envErr != nil {
> 		return envErr
> 	}
> 	if nodeID > math.MaxUint16 {
> 		return fmt.Errorf("--node-id %d: %w", nodeID, errNodeIDRange)
> 	}
> 	cfg.nodeID = uint16(nodeID)
> 	if missing := missingRequired(*cfg); len(missing) > 0 {
> 		return fmt.Errorf("%w: %s", errMissingRequired, strings.Join(missing, ", "))
> 	}
> 	err := checkMetricsCerts(*cfg)
> 	if err != nil {
> 		return err
> 	}
> 	err = checkStartupGates(*cfg)
> 	if err != nil {
> 		return err
> 	}
> 	err = checkValueMaxBytes(*cfg)
> 	if err != nil {
> 		return err
> 	}
> 	err = checkDatagramCeiling(*cfg)
> 	if err != nil {
> 		return err
> 	}
> 	return checkSyncChunkBytes(*cfg)
> }
> ```
> `[VERIFIED: cmd/sup-kv/main.go:432-462]`

The checks that exist today:

| Check | Site | What it enforces |
|-------|------|------------------|
| required knobs | `main.go:996-1013` | `--node-id` non-zero, `--bind-addr`, `--broadcast-addr`, `--dump-path` non-empty |
| node-id range | `main.go:438-441` | `nodeID <= math.MaxUint16` |
| metrics cert triple | `main.go:479-492` | all three envs set or none |
| startup gates | `main.go:494-509` | both gates `> 0` |
| value max positivity | `main.go:511-522` | `--value-max-bytes > 0` |
| datagram ceiling | `main.go:524-557` | largest record encodes `<= 65 507` |
| sync chunk ceiling | `main.go:464-477` | `--sync-chunk-bytes <= repl.MaxSyncChunkBytes` |

`checkSyncChunkBytes` explicitly leaves non-positive values legal because the core normalizes them:

> `cmd/sup-kv/main.go:464-477`
> ```go
> // checkSyncChunkBytes rejects an explicit --sync-chunk-bytes above the UDP
> // datagram ceiling before any bind (WR-06): a chunk budget above 65 507
> // describes a chunk no datagram can carry, and the core's defensive clamp
> // would silently ship smaller chunks than the operator configured. The
> // ceiling is the core's own constant (repl.MaxSyncChunkBytes, pinned to the
> // adapter's datagram ceiling by a cross-seam test). A zero or negative value
> // stays legal — the core normalizes it to the design default.
> func checkSyncChunkBytes(cfg config) error {
> 	if cfg.syncChunkBytes > repl.MaxSyncChunkBytes {
> 		return fmt.Errorf("%w: --sync-chunk-bytes=%d (at most %d, the largest IPv4 UDP payload)",
> 			errSyncChunkBytes, cfg.syncChunkBytes, repl.MaxSyncChunkBytes)
> 	}
> 	return nil
> }
> ```
> `[VERIFIED: cmd/sup-kv/main.go:464-477]`

#### What is missing

- No relation checks: `peer_timeout > hello_interval`, `ack_timeout <= ack_timeout_max`, `drift_warn < max_clock_lead`.
- No positivity checks for the other durations/rates: `hello_interval`, `peer_timeout`, `ack_timeout`, `ack_timeout_max`, `retransmit_interval`, `sync_interval`, `sync_response_rate`, `max_clock_lead`, `dump_debounce`, `dump_max_wait`, `drift_warn`, `wait_timeout`.
- `retransmit_interval=1ns` is accepted; the event loop arms timers at every nanosecond.
- `sync-response-rate` above 1e9 is clamped by `repl.Config.normalized`, not rejected.
- Positional arguments are ignored: `flag.Parse` stops at the first non-flag and the remainder is never examined. `[VERIFIED: cmd/sup-kv/main.go:227-232]` (`flag.Parse()` then only `*showVersion` is read).
- `--help` is the stdlib default usage; it prints flag descriptions (which name env twins) but does not list the five env-only knobs. No custom `flag.Usage` is assigned; the only reference is the call at `main.go:241`. `[VERIFIED: cmd/sup-kv/main.go:227-276]`

The core's normalization is the defense-in-depth layer that hides these omissions today:

> `internal/app/repl/core.go:299-335`
> ```go
> // normalized returns cfg with out-of-domain fields replaced by the design
> // defaults — zero or negative everywhere, and SyncResponseRate additionally
> // above maxSyncResponseRate, where the spacing division would truncate to
> // zero (D-15 P4-WR-02) — and a nil Log or OnDiscard replaced by a no-op.
> func (c Config) normalized() Config {
> 	if c.HelloInterval <= 0 {
> 		c.HelloInterval = defaultHelloInterval
> 	}
> 	if c.PeerTimeout <= 0 {
> 		c.PeerTimeout = defaultPeerTimeout
> 	}
> 	if c.AckTimeout <= 0 {
> 		c.AckTimeout = defaultAckTimeout
> 	}
> 	if c.AckTimeoutMax <= 0 {
> 		c.AckTimeoutMax = defaultAckTimeoutMax
> 	}
> 	if c.RetransmitInterval <= 0 {
> 		c.RetransmitInterval = defaultRetransmitInterval
> 	}
> 	if c.SyncInterval <= 0 {
> 		c.SyncInterval = defaultSyncInterval
> 	}
> 	if c.SyncChunkBytes <= 0 {
> 		c.SyncChunkBytes = defaultSyncChunkBytes
> 	}
> 	if c.SyncChunkBytes > MaxSyncChunkBytes {
> 		// A budget above the datagram ceiling describes a chunk no datagram can
> 		// carry, and the cached listing build would pin the oversized
> 		// allocation for its whole age (WR-06).
> 		c.SyncChunkBytes = MaxSyncChunkBytes
> 	}
> 	if c.SyncResponseRate <= 0 || c.SyncResponseRate > maxSyncResponseRate {
> 		c.SyncResponseRate = defaultSyncResponseRate
> 	}
> 	if c.MaxClockLead <= 0 {
> 		c.MaxClockLead = defaultMaxClockLead
> 	}
> ```
> `[VERIFIED: internal/app/repl/core.go:299-335]`

> `internal/app/repl/core.go:322-330`
> ```go
> 	if c.SyncChunkBytes <= 0 {
> 		c.SyncChunkBytes = defaultSyncChunkBytes
> 	}
> 	if c.SyncChunkBytes > MaxSyncChunkBytes {
> 		// A budget above the datagram ceiling describes a chunk no datagram can
> 		// carry, and the cached listing build would pin the oversized
> 		// allocation for its whole age (WR-06).
> 		c.SyncChunkBytes = MaxSyncChunkBytes
> 	}
> ```
> `[VERIFIED: internal/app/repl/core.go:322-330]`

**Trap found while mapping:** `drift_warn` and `max_clock_lead` both default to 60 s. `version.Validate` accepts `l == now+maxLead` but rejects anything greater:

> `internal/app/version/version.go:108-118`
> ```go
> func (v RecordVersion) Validate(now time.Time, maxLead time.Duration) error {
> 	if v[6]&0xF0 != uuidVerBits || v[8]&0xC0 != uuidVarBits {
> 		return ErrVersionLayout
> 	}
> 	l, _, _ := v.Unpack()
> 	nowMs := now.UnixMilli()
> 	lead := max(maxLead.Milliseconds(), 0)
> 	if l >= nowMs && l-nowMs > lead {
> 		return ErrVersionFuture
> 	}
> 	return nil
> }
> ```
> `[VERIFIED: internal/app/version/version.go:108-118]`

With `max_clock_lead == drift_warn == 60 s`, a version leading by more than 60 s is rejected as `bad_version` before `ApplyRemote`, so it can never reach the drift counter; a version leading by exactly 60 s is accepted but is not "more than `drift_warn`". The drift WARN is unreachable at the defaults. Phase 3 research recorded this as assumption A1 and handed it to Phase 7:
> `.planning/phases/03-transport/03-RESEARCH.md:602` — "`max_clock_lead = 60 s` is the right default (equals `drift_warn`, so peers inside the documented drift tolerance are accepted) … Phase 7 can revisit validation." `[CITED: .planning/phases/03-transport/03-RESEARCH.md:602]`

#### The library-level value guard (IN-04) is still a precondition

> `internal/app/persist/doc.go:31-34`
> ```go
> // EncodeTo trusts its valueMaxBytes argument to stay at or below the dump
> // format's datagram ceiling (maxDumpValueBytes), because the format's line
> // bound is sized from that ceiling: a larger configured limit could emit a
> // line this package's own Decode rejects. The shipped binary enforces the
> // ceiling at startup; the library-level guard belongs to Phase 7's config
> // validation (IN-04, deferred there).
> ```
> `[VERIFIED: internal/app/persist/doc.go:31-34]`

`EncodeTo` checks record sizes against `valueMaxBytes` but does not validate `valueMaxBytes` itself:

> `internal/app/persist/codec.go:133-146`
> ```go
> func EncodeTo(w io.Writer, records map[string]storage.Record, valueMaxBytes int) error {
> 	enc := json.NewEncoder(w)
> 	enc.SetEscapeHTML(false)
> 	count := len(records)
> 	err := enc.Encode(header{FormatVersion: dumpFormatVersion, RecordCount: &count})
> 	if err != nil {
> 		return err
> 	}
> 	for _, key := range slices.Sorted(maps.Keys(records)) {
> 		rec := records[key]
> 		if len(rec.Data) > valueMaxBytes {
> 			return fmt.Errorf("persist: record %q is %d bytes, over the configured value limit %d: %w",
> 				key, len(rec.Data), valueMaxBytes, storage.ErrValueTooLarge)
> 		}
> ```
> `[VERIFIED: internal/app/persist/codec.go:133-146]`

Any guard must run **before** the header encode; otherwise an invalid limit writes a partial generation and then returns. The format ceiling is `maxDumpValueBytes = 65507`:

> `internal/app/persist/codec.go:40-41`
> ```go
> const (
> 	maxDumpValueBytes      = 65507
> ```
> `[VERIFIED: internal/app/persist/codec.go:40-41]`

### OBSV-03 — Metric Definitions

#### Conflict counter counts every displaced **or rejected** version

`storage.join` returns a `Conflict` for an ordinary overwrite, a stale rejection and a tie-break loss:

> `internal/app/storage/store.go:402-441`
> ```go
> // join compares rec against the stored record by the full 16-byte version and,
> // at an equal version, by the deterministic content tie-break (PERSIST-07,
> // ADR 0012): identical content is a full no-op, remove-wins settles a
> // differing tombstone flag, and otherwise the greater Data bytes win by
> // [bytes.Compare] over explicit fields (no map order, no randomness). It
> // installs only when the incoming record wins and reports whether it did; the
> // caller must hold the write lock. Every displaced or rejected version is
> // returned exactly once as a Conflict — the tie-break included, where Winner
> // and Loser are the same version bytes and the stored record is the only
> // observable evidence of the outcome.
> func (s *Store) join(key string, rec Record) ([]Conflict, bool) {
> 	stored, ok := s.records[key]
> 	if !ok {
> 		s.install(key, rec)
> 		return nil, true
> 	}
> 	switch cmp := rec.Version.Compare(stored.Version); {
> 	case cmp > 0:
> 		s.install(key, rec)
> 		return []Conflict{{Key: key, Winner: rec.Version, Loser: stored.Version}}, true
> 	case cmp < 0:
> 		return []Conflict{{Key: key, Winner: stored.Version, Loser: rec.Version}}, false
> 	default:
> 		switch {
> 		case RecordsEqual(stored, rec):
> 			return nil, false
> 		case rec.Deleted && !stored.Deleted:
> 			// Remove-wins: a delete at the same version beats a live value.
> 			s.install(key, rec)
> 			return []Conflict{{Key: key, Winner: rec.Version, Loser: stored.Version}}, true
> 		case stored.Deleted && !rec.Deleted:
> 			return []Conflict{{Key: key, Winner: stored.Version, Loser: rec.Version}}, false
> 		case bytes.Compare(rec.Data, stored.Data) > 0:
> 			s.install(key, rec)
> 			return []Conflict{{Key: key, Winner: rec.Version, Loser: stored.Version}}, true
> 		default:
> 			return []Conflict{{Key: key, Winner: stored.Version, Loser: rec.Version}}, false
> 		}
> 	}
> }
> ```
> `[VERIFIED: internal/app/storage/store.go:402-441]`

The review's engine evidence defines the intended denominator exactly:

> `docs/reviews/2026-11-19-v1.0/evidence/review-engine.md:736`
> ```text
> Fix: `Merge` returns `(changed bool, displaced *Conflict, ...)`; `ApplyRemote` uses `changed`; the conflict counter is fed only for displacement of a pending local version (the replication core knows the unacked set); correct HELP and doc text.
> ```
> `[CITED: docs/reviews/2026-11-19-v1.0/evidence/review-engine.md:736]`

"Pending local version" is the replication core's `pending` set (created by `trackLocalWrite` on every accepted local write), not the dump-confirmation set. `MergeStats`'s current doc calls an LWW loser "an ordinary overwrite, counted" — the opposite of the required rule. `[VERIFIED: internal/app/repl/core.go:436-446]`

#### A failed send is counted as sent

`flush` increments the `SentByType` counter before attempting the socket write; `write` only logs an error and cannot prevent the count:

> `internal/udprepl/repl.go:1103-1156`
> ```go
> // flush encodes and sends every outbound message the core emitted for the
> // just-handled event: broadcast to the configured broadcast address,
> // unicast to the handle resolved through [net.ResolveUDPAddr] — handles are
> // opaque to the core; only here do they become addresses (D-03). Each
> // encode-successful message counts into its SentByType kind (OBS-01) — the
> // datagram is on its way to the socket; an encode failure counts nowhere.
> // Failures are logged, never panics: a transient send error must not take
> // the node down.
> func (a *Adapter) flush(out *port.Outbox) {
> 	for _, ob := range out.Drain() {
> 		datagram, err := encodeDatagram(a.nodeID, a.runID, ob.Msg)
> 		if err != nil {
> 			a.log(err.Error())
> 			continue
> 		}
> 		a.countSent(ob.Msg)
> 		if ob.Broadcast {
> 			a.write(datagram, a.broadcastAddr)
> 			continue
> 		}
> 		addr, err := net.ResolveUDPAddr("udp4", ob.To)
> 		if err != nil {
> 			a.log(fmt.Sprintf("udprepl: resolve unicast handle %q: %v", ob.To, err))
> 			continue
> 		}
> 		a.write(datagram, addr)
> 	}
> }
>
> // countSent adds one to the message kind's send counter (OBS-01) — the
> // five-kind vocabulary of the wire Body oneof, counted in flush where every
> // outbound message crosses the encode step on the event-loop goroutine.
> func (a *Adapter) countSent(msg port.Message) {
> 	switch msg.(type) {
> 	case port.Hello:
> 		a.sentHello.Add(1)
> 	case port.DataReplication:
> 		a.sentData.Add(1)
> 	case port.Ack:
> 		a.sentAck.Add(1)
> 	case port.DataRequest:
> 		a.sentDataRequest.Add(1)
> 	case port.DataSync:
> 		a.sentSync.Add(1)
> 	}
> }
>
> // write sends one datagram to addr, logging (never panicking on) a failure.
> func (a *Adapter) write(datagram []byte, addr *net.UDPAddr) {
> 	_, err := a.conn.WriteToUDP(datagram, addr)
> 	if err != nil {
> 		a.log(fmt.Sprintf("udprepl: send to %s: %v", addr, err))
> 	}
> }
> ```
> `[VERIFIED: internal/udprepl/repl.go:1103-1156]`

Note that a unicast `ResolveUDPAddr` failure also happens **after** `countSent`, so an unresolved handle is counted as sent too.

#### `process_resident_memory_bytes` reports peak RSS

> `internal/metrics/metrics.go:483-509`
> ```go
> 		{
> 			Name: "process_resident_memory_bytes", Type: Gauge,
> 			Help: "Resident memory size in bytes.",
> 			Samples: []Sample{
> 				{Value: rss},
> 			},
> 		},
> 	}
> }
>
> // processUsage reads this process's CPU time and resident set size through
> // [syscall.Getrusage] — Linux and darwin both provide it (06-RESEARCH A2's
> // minimal process_* enumeration under D-04). CPU is user+system seconds;
> // ru_maxrss is KiB on Linux and bytes on darwin, normalized here to bytes.
> func processUsage() (cpuSeconds, rssBytes float64) {
> 	var ru syscall.Rusage
> 	err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
> 	if err != nil {
> 		return 0, 0
> 	}
> 	sec := func(t syscall.Timeval) float64 { return float64(t.Sec) + float64(t.Usec)/1e6 }
> 	rss := float64(ru.Maxrss)
> 	if rss > 0 && runtime.GOOS != "darwin" {
> 		rss *= 1024
> 	}
> 	return sec(ru.Utime) + sec(ru.Stime), rss
> }
> ```
> `[VERIFIED: internal/metrics/metrics.go:483-509]`

`ru_maxrss` is the **maximum** resident set size, not current:

> `man 2 getrusage` (Linux manual page)
> ```text
> ru_maxrss (since Linux 2.6.32)
>        This is the maximum resident set size used (in kilobytes).
> ```
> `[VERIFIED: man 2 getrusage, ru_maxrss]`

#### Dump duration is the last attempt's span, stated only in Help

> `internal/metrics/metrics.go:405-413`
> ```go
> func dumpFamilies(stats persist.DumpStats) []Family {
> 	return []Family{
> 		{
> 			Name: "supkv_dumps_duration_seconds", Type: Gauge,
> 			Help: "Duration of the last dump's encode and save in seconds.",
> 			Samples: []Sample{
> 				{Value: stats.DurationSeconds},
> 			},
> 		},
> ```
> `[VERIFIED: internal/metrics/metrics.go:405-413]`

The scheduler records the span for both success and failure (`recordDump` runs before the error check), so "last dump" means last **attempt**, including failures. That distinction is not stated in Help or OPERATIONS. `[VERIFIED: internal/app/persist/scheduler.go:228-249]`

#### Drift counter definition is not the Spec's

Already quoted above: `MergeStats.DriftWarnings` is "every apply whose drift signal was nonzero" and the clock signal is local-`l`-versus-wall, both signs, before absorption. `[VERIFIED: internal/app/repl/core.go:436-446; internal/app/clock/clock.go:119-124,159-175]`

#### Four Phase-4 counters exist but have no production reader

| Counter | Core field + increment | Accessor | Adapter mirror/accessor | Metrics family |
|---------|------------------------|----------|--------------------------|----------------|
| `ListingDropped` | `core.go:797`, increment `core.go:2410` | `core.go:3169-3171` | none | none |
| `SyncStreamDropped` | `core.go:885`, increment `core.go:1892` | `core.go:3066-3068` | none | none |
| `ListedKeysEvicted` | `core.go:903`, increments `core.go:1959`, `core.go:2033` | `core.go:3074-3076` | none | none |
| `NotificationsRegressed` | `core.go:876`, increments `core.go:2252`, `core.go:2256` | `core.go:3014-3016` | none | none |

> `internal/app/repl/core.go:3008-3016`
> ```go
> // NotificationsRegressed returns the number of local-write notifications the
> // not-greater-than-tracked guard ignored (PROTO-05, D-05) — both regressive
> // notifications (a strictly older version, whose waiter also resolves as
> // Superseded, WR-03) and equal duplicate notifications of the tracked
> // version. A value read like the discard counters; Phase 7's OBSV surface owns
> // publishing it (no adapter mirror this phase).
> func (c *Core) NotificationsRegressed() uint64 {
> 	return c.notificationsRegressed
> }
> ```
> `[VERIFIED: internal/app/repl/core.go:3008-3016]`

> `internal/app/repl/core.go:3063-3076`
> ```go
> // SyncStreamDropped returns the number of partial-listing streams the bounds
> // evicted — the T-4-03 observability signal: a growing value means a peer is
> // flooding distinct sync_ids with missing chunks.
> func (c *Core) SyncStreamDropped() uint64 {
> 	return c.syncStreamDropped
> }
>
> // ListedKeysEvicted returns the number of retained listed keys released by
> // whole-peer-set eviction or by expiry (T-4-41, CR-01) — each key removed
> // from the union by the maxListedKeysTotal bound or by the two-sync-interval
> // age bound increments it once.
> func (c *Core) ListedKeysEvicted() uint64 {
> 	return c.listedKeysEvicted
> }
> ```
> `[VERIFIED: internal/app/repl/core.go:3063-3076]`

> `internal/app/repl/core.go:3163-3171`
> ```go
> // ListingDropped returns the number of targeted listing triggers a cap
> // refused (CR-01, T-4-23, WR-01) — the observability seam for the listing
> // drop policy, mirroring EgressDropped: every refused first-seen or restart
> // listing is counted once, and the refused trigger is re-armed for a later
> // Advance, so the count measures back-pressure rather than permanent loss (a
> // trigger refused past the retry set's bound is lost and stays counted).
> func (c *Core) ListingDropped() uint64 {
> 	return c.listingDropped
> }
> ```
> `[VERIFIED: internal/app/repl/core.go:3163-3171]`

The exporter's closed `Sources` set has no fields for them:

> `internal/metrics/metrics.go:183-200`
> ```go
> type Sources struct {
> 	Discards      func() repl.DiscardStats
> 	KnownPeers    func() uint64
> 	Unacked       func() repl.UnackedStats
> 	Sent          func() udprepl.SentByType
> 	Retransmitted func() uint64
> 	MergeStats    func() repl.MergeStats
> 	DumpStats     func() persist.DumpStats
> 	LoadStats     func() persist.LoadStats
>
> 	// The XPORT-07 emission set (03-12): the paced-datagram counter, the
> 	// total send-queue gauge, the egress overflow drops and the receive-path
> 	// error count — adapter accessors over the core's post-event mirrors.
> 	Paced         func() uint64
> 	SendQueue     func() int
> 	EgressDropped func() uint64
> 	ReadErrors    func() uint64
> }
> ```
> `[VERIFIED: internal/metrics/metrics.go:183-200]`

### OBSV-04 — Exit Codes and Shutdown Ordering

Current codes:

> `cmd/sup-kv/main.go:70-77`
> ```go
> // Process exit codes: a configuration error (missing or malformed knob)
> // exits with exitUsage; a failed final dump during shutdown exits with
> // exitDumpFailed — a clean exit 0 is reserved for a shutdown whose final
> // write-confirmed dump completed (D-19).
> const (
> 	exitDumpFailed = 1
> 	exitUsage      = 2
> )
> ```
> `[VERIFIED: cmd/sup-kv/main.go:70-77]`

Startup failure and final-dump failure therefore share code 1. There is no code 3.

Signal ownership is installed **after** `wire` — which loads the dump and binds the UDP socket — and `Serve` failure skips `shutdown` entirely:

> `cmd/sup-kv/main.go:255-276`
> ```go
> 	node, listener, err := wire(cfg)
> 	if err != nil {
> 		log.Fatalf("sup-kv: %v", err)
> 	}
> 	// Signal ownership spans serving AND the D-19 shutdown (P1-IN-01): the
> 	// armed channel is released only after shutdown completes, so a second
> 	// SIGTERM during the graceful drain is observed as an operator's
> 	// hurry-up — it takes the force-stop path — instead of killing the
> 	// process with the default disposition and forfeiting the final dump.
> 	sig := armSignals()
> 	err = serve(node, listener, sig)
> 	if err != nil {
> 		log.Fatalf("sup-kv: grpc serve exited before the shutdown signal: %v", err)
> 	}
> 	err = shutdown(node, sig) // D-19: the order is the contract
> 	signal.Stop(sig)
> 	if err != nil {
> 		log.Printf("sup-kv: %v", err)
> 		os.Exit(exitDumpFailed)
> 	}
> 	os.Exit(0)
> ```
> `[VERIFIED: cmd/sup-kv/main.go:255-276]`

`wire` performs the synchronous dump load and the UDP bind before returning:

> `cmd/sup-kv/main.go:697-705`
> ```go
> func wire(cfg config) (*wired, net.Listener, error) {
> 	hub := grpcapi.NewWatchHub()
> 	engine, engineCancel, engineDone := startEngine(cfg, hub)
>
> 	adapter, err := newReplicationAdapter(cfg, engine)
> 	if err != nil {
> 		engineCancel()
> 		return nil, nil, fmt.Errorf("bind replication socket: %w", err)
> 	}
> ```
> `[VERIFIED: cmd/sup-kv/main.go:697-705]`

The final dump is already time-bounded at 10 s:

> `cmd/sup-kv/main.go:970-977`
> ```go
> 	err := node.engine.FlushTimeout(shutdownGrace)
> 	if err != nil {
> 		return fmt.Errorf("final dump failed: %w", err)
> 	}
>
> 	node.engineCancel()
> 	awaitEngineStop(node.engineDone, shutdownGrace)
> 	return nil
> }
> ```
> `[VERIFIED: cmd/sup-kv/main.go:970-977]`

> `cmd/sup-kv/main.go:83-86`
> ```go
> // shutdownGrace bounds GracefulStop during the D-19 shutdown: past it,
> // gs.Stop() force-closes whatever is still open (T-05-10 — a hang here
> // would leave committed data only in memory when the supervisor SIGKILLs).
> const shutdownGrace = 10 * time.Second
> ```
> `[VERIFIED: cmd/sup-kv/main.go:83-86]`

The OPERATIONS doc and its anchor test currently encode the wrong final-dump code:

> `docs/OPERATIONS.md:280-293`
> ```text
> ## Lifecycle and exit codes
>
> `SIGTERM` (or `SIGINT`) triggers the graceful path: the node stops serving
> (health flips to NOT_SERVING), finishes a final confirmed dump, then exits.
> Exit codes:
>
> - exit code 0 — clean shutdown; the final dump is confirmed on disk;
> - exit code 1 — the final dump failed (a systemd unit with
>   `Restart=on-failure` reacts to this);
> - exit code 2 — malformed configuration: a bad flag value or a malformed
>   `SUPKV_*` environment value.
> ```
> `[VERIFIED: docs/OPERATIONS.md:280-293]`

`cmd/sup-kv/operations_test.go:81-84` pins those literals:

> `cmd/sup-kv/operations_test.go:79-84`
> ```go
> 	{"## Lifecycle and exit codes", "the lifecycle/exit-codes section"},
> 	{"SIGTERM", "the graceful-shutdown trigger"},
> 	{"exit code 0", "clean shutdown: final dump confirmed on disk"},
> 	{"exit code 1", "failed final dump (Restart=on-failure reacts)"},
> 	{"exit code 2", "malformed configuration"},
> 	{"Restart=on-failure", "the systemd reaction to exit code 1"},
> ```
> `[VERIFIED: cmd/sup-kv/operations_test.go:79-84]`

`--help` currently lists each flag with its env twin in the flag description, but there is no env-only block; the five env-only names never appear in usage output. `[VERIFIED: cmd/sup-kv/main.go:99-137,227-276]`

### OBSV-07 — The Dependency Boundary

The vendored strict config's project rules are additive keys after the byte-frozen upstream body:

> `.golangci.yml:65-95`
> ```yaml
>         # ---- sup-kv architecture boundary (project additions) ------------
>         # ---- upstream powerman/golangci-lint-strict 2.15.0 content above -
>         # ---- this banner stays byte-unmodified; additive keys only ------
>         sup-kv-app-no-io:
>           files:
>             - '**/internal/app/**' # leading **/ mandatory (depguard globs match absolute paths); quotes mandatory (leading * breaks YAML)
>             - '!$test' # exempts *_test.go from the boundary
>           deny:
>             - pkg: net # plain string = prefix match (catches net, net/http, net/url)
>               desc: internal/app is the pure decision core - no networking
>             - pkg: os
>               desc: internal/app is the pure decision core - no filesystem
>             - pkg: syscall
>               desc: internal/app is the pure decision core - no syscalls
>             - pkg: google.golang.org/
>               desc: internal/app is the pure decision core - no gRPC/protobuf
>         sup-kv-port-stdlib-only:
>           files:
>             - '**/internal/port/**'
>             - '!$test'
>           list-mode: strict
>           allow:
>             - $gostd
>             - github.com/Djarvur/sup-kv # own module path allowed per D-11's deny-third-party wording
>         sup-kv-public-api-no-internal:
>           files:
>             - '**/api/**' # the published consumer package; leading **/ + quotes for the same depguard glob reason as above
>             - '!$test'
>           deny:
>             - pkg: github.com/Djarvur/sup-kv/internal
>               desc: the public API package must remain importable by other modules
> ```
> `[VERIFIED: .golangci.yml:65-95]`

The file's first line is the provenance header:

> `.golangci.yml:1`
> ```yaml
> # Origin: https://github.com/powerman/golangci-lint-strict version 2.15.0
> ```
> `[VERIFIED: .golangci.yml:1]`

No rule currently forbids adapters in `internal/app` or `internal/port`, and the two observed cross-imports are real:

> `go list -f '{{.ImportPath}} -> {{join .Imports " "}}' ./internal/... ./cmd/sup-kv` — module-local imports only (standard library and third-party imports omitted for width; the full command was run)
> ```text
> internal/app/clock -> (none)
> internal/app/internal/simnet -> internal/app/clock internal/app/repl internal/app/storage internal/app/version internal/port
> internal/app/internal/testutil -> internal/testutil
> internal/app/persist -> internal/app/clock internal/app/storage internal/app/version internal/port
> internal/app/repl -> internal/app/storage internal/app/version internal/port
> internal/app/storage -> internal/app/version
> internal/port -> internal/app/storage internal/app/version
> internal/udprepl -> internal/app/repl internal/app/storage internal/app/version internal/port internal/udprepl/replproto/supkv/v1
> internal/persistfile -> internal/port
> internal/grpcapi -> api/supkvapi/v1 internal/app/persist internal/app/storage internal/app/version internal/persistfile internal/port
> internal/metrics -> internal/app/persist internal/app/repl internal/port internal/udprepl
> cmd/sup-kv -> api/supkvapi/v1 internal/app/persist internal/app/repl internal/app/storage internal/grpcapi internal/metrics internal/persistfile internal/port internal/udprepl
> ```
> `[VERIFIED: go list -f '{{.ImportPath}} -> {{join .Imports " "}}' ./internal/... ./cmd/sup-kv (module-local filtered)]`

Two concrete cross-imports:

1. `internal/metrics` imports `internal/udprepl` only for the `SentByType` value type. Exact lines:
   > `internal/metrics/metrics.go:187`
   > ```go
   > 	Sent          func() udprepl.SentByType
   > ```
   > `[VERIFIED: internal/metrics/metrics.go:187]`

   > `internal/metrics/metrics.go:361`
   > ```go
   > func trafficFamilies(disc repl.DiscardStats, sentBy udprepl.SentByType) []Family {
   > ```
   > `[VERIFIED: internal/metrics/metrics.go:361]`
2. `internal/grpcapi` imports `internal/persistfile` for two sentinels and uses `*persist.Engine` as its store facade. Exact lines:
   > `internal/grpcapi/server.go:30`
   > ```go
   > 	"github.com/Djarvur/sup-kv/internal/persistfile"
   > ```
   > `[VERIFIED: internal/grpcapi/server.go:30]`

   > `internal/grpcapi/server.go:70`
   > ```go
   > 	engine *persist.Engine
   > ```
   > `[VERIFIED: internal/grpcapi/server.go:70]`

   > `internal/grpcapi/server.go:99`
   > ```go
   > func NewServer(engine *persist.Engine, notify LocalNotify, hub *WatchHub, opts ServerOptions) *Server {
   > ```
   > `[VERIFIED: internal/grpcapi/server.go:99]`

   > `internal/grpcapi/server.go:519-522`
   > ```go
   > 	case errors.Is(err, persistfile.ErrInvalidSnapshotName):
   > 		return status.Error(codes.InvalidArgument, persistfile.ErrInvalidSnapshotName.Error())
   > 	case errors.Is(err, persistfile.ErrNoDump):
   > 		return status.Error(codes.FailedPrecondition, persistfile.ErrNoDump.Error())
   > ```
   > `[VERIFIED: internal/grpcapi/server.go:519-522]`

`grpcapi` calls 12 `*persist.Engine` methods (`ApplyRemote`, `Delete`, `DumpStats`, `Get`, `LiveVersions`, `Lookup`, `Records`, `Restore`, `Set`, `SetIfAbsent`, `Snapshot`, `Version`), plus the Watch hub stores the engine for `Records`/`Set`. Replacing the concrete type with a consumer interface is possible but is a larger refactor than D-07 requires. `[VERIFIED: go list + grep of engine method call sites]`

### OBSV-08 — Decomposition

`main.go` and `repl.go` sizes:

> `wc -l cmd/sup-kv/main.go internal/udprepl/repl.go`
> ```text
>  1104 cmd/sup-kv/main.go
>  1417 internal/udprepl/repl.go
> ```
> `[VERIFIED: wc -l]`

`repl.go` declares its own mixed role in the package doc, then keeps socket, loop, mirror and codec in one file:

> `internal/udprepl/repl.go:1-15`
> ```go
> // Package udprepl is the sup-kv UDP replication adapter: the only net- and
> // protobuf-importing replication code, mechanical per D-01/D-02 — it maps
> // the port message structs onto the wire schema, verifies the CRC before
> // any body decode, owns the socket, and feeds the pure core
> // (internal/app/repl) from one event-loop goroutine (D-04) — mirroring
> // persistfile's adapter role. It owns no protocol decision: discards,
> // the peer table and every timer live in the core. Production binds never
> // set SO_REUSEADDR, so a second process on the node's address fails with
> // EADDRINUSE instead of silently splitting unicast traffic (XPORT-09); only
> // the explicit test-only [WithAddressReuse] option restores same-host port
> // sharing. Same-host caveat under that option: two nodes sharing one UDP
> // port receive each other's broadcasts, but unicast to the shared port is
> // delivered to exactly one socket, the kernel's choice — the real-socket
> // smoke test therefore asserts only broadcast-delivered behavior.
> ```
> `[VERIFIED: internal/udprepl/repl.go:1-15]`

Responsibility map with the exact declarations:

| Unit | Declarations / line ranges |
|------|----------------------------|
| Package doc, wire constants, queue sizes, rate limits | `1-15`, `41-95` |
| Encode/decode codec | `encodeDatagram` `1251-1272`; `decode` `1285-1322`; `protoBody` `1324-1357`; `portMessage` `1359-1408`; `recordVersion` `1410-1417` |
| Socket construction/options | `packetConn` `224-232`; `options`/`Option`/`WithAddressReuse` `234-255`; `listenDatagram` `464-501`; `configureSocketBuffers` `503-534`; `validateBufferRequest` `536-543`; `socketOption` `545-567`; `checkSendBuffer` `569-584`; bind/buffer half of `New` `393-440` |
| Event loop | `Run` `586-625`; `readLoop` `1066-1086`; `logReadError` `1092-1101`; `flush` `1111-1130`; `countSent` `1135-1148`; `write` `1151-1156`; timers `1220-1249`; inbound/notify types `125-150`; tracker/wait `141-223`, `661-872` |
| Mirror | `Adapter` mirror fields `310-358`; `listingMirror` `1023-1030`; `mirrorStats` `1158-1215`; accessors `874-1038` |

`Adapter` states the single-driver rule the split must preserve:

> `internal/udprepl/repl.go:257-264`
> ```go
> // Adapter is the UDP replication adapter for one node: it owns the bound
> // socket and drives one repl.Core. Exactly one goroutine — the caller's
> // Run — mutates core state (D-04); the read goroutine only queues inbound
> // datagrams, write authors only queue notifications and create waiter
> // trackers, and the discard counters, known-peer count and the replication
> // read states (the first-sync signal and each write's tracker) are
> // mutex- or atomic-guarded so cross-goroutine readers never touch core
> // state.
> ```
> `[VERIFIED: internal/udprepl/repl.go:257-264]`

### Routed Deferrals — Exact Sites

| Deferral | Exact site | Current state | Phase-7 obligation |
|----------|-----------|---------------|--------------------|
| Phase-3 WR-02 `sync_chunk_bytes` unbounded | `cmd/sup-kv/main.go:464-477`; `internal/app/repl/core.go:322-330` | Upper bound exists (WR-06); non-positive still normalizes silently | Add bounds and positivity to the one validation table; keep the core clamp as defense |
| Phase-3 IN-01 apply-failure WARN | `internal/app/repl/core.go:1525-1527` | Names the key and is not rate-limited | Rate-limit per peer/class and drop the key from the message (review fix text) |
| Phase-4 IN-03 four counters | `internal/app/repl/core.go:3014-3016`, `3066-3068`, `3074-3076`, `3169-3171` | Accessors exist; no adapter mirror, no `Sources` field, no family | Publish all four |
| Phase-4 New-2 unbound listing | `internal/app/repl/core.go:1709-1721` | Increments `DuplicateNodeID` and warns as `duplicate_node_id` | Dedicated `unbound_listing` class/label, same rate limiter |
| Phase-5 IN-04 library guard | `internal/app/persist/doc.go:31-34`; `internal/app/persist/codec.go:133-146` | Documented precondition only | Guard at the top of `EncodeTo` before any write; test |
| Phase-6 routing | `.planning/phases/06-api-and-semantics/06-REVIEW-DISPOSITION.md` | No row names Phase 7 | Confirmed: nothing routed here |

> `internal/app/repl/core.go:1709-1721`
> ```go
> func (c *Core) listingBound(now time.Time, in port.InboundDatagram) bool {
> 	if known, ok := c.peers[in.NodeID]; ok && known.runID == in.RunID {
> 		return true
> 	}
> 	c.discards.DuplicateNodeID++
> 	c.cfg.OnDiscard(DiscardDuplicateNodeID, fmt.Sprintf(
> 		"node_id %d listing from unbound run at handle %q", in.NodeID, in.Source))
> 	c.warnRateLimited(now, in.NodeID, DiscardDuplicateNodeID, fmt.Sprintf(
> 		"sup-kv: configuration error: node_id %d sent a listing from a run the peer table does not hold, datagram discarded (CR-01)",
> 		in.NodeID,
> 	))
> 	return false
> }
> ```
> `[VERIFIED: internal/app/repl/core.go:1709-1721]`

## Design Options and Recommendations (D-01…D-08)

### D-01 — One Lifecycle Catalog

**Recommendation:** define the catalog as data first, then implement it through typed event seams and a single rendering layer. Do **not** have `internal/app` call `slog` directly: the core is the pure decision layer, and a typed callback is both testable and boundary-clean. `grpcapi` and `cmd` are adapters/wiring and may render.

#### Catalog

| Event | Level | Emitter | Stable fields | Why this level |
|-------|-------|---------|---------------|----------------|
| Startup | INFO | `cmd/main` after `setupLogging`, before `wire` | `version`, `commit`, `go_version`, `node_id`, `bind_addr`, `broadcast_addr`, `dump_path`, `grpc_addr`, `metrics_addr`, `metrics_tls`, `value_max_bytes`, `sync_chunk_bytes`, `sync_response_rate`, `startup_gate`, `startup_gate_empty`, `wait_timeout`, `hello_interval`, `peer_timeout`, `ack_timeout`, `ack_timeout_max`, `retransmit_interval`, `sync_interval`, `max_clock_lead`, `dump_debounce`, `dump_max_wait`, `drift_warn`, `log_level`, `log_format` | The operator's first record; secrets-free (no certificate/key bytes, no file contents) |
| Dump load outcome | INFO for `prompt`/`main`; the existing WARN/ERROR for `backup`/`unusable`, extended with the same fields | `cmd/main` in `startEngine` after `persist.Open` | `outcome` (`prompt`/`main`/`backup`/`unusable`), `generation` (`main`/`prev`/`none`), `records` (loaded key count), `discarded` (`DecodeStats.Total()`) | Exactly one load record per start: a clean load is INFO; a fallback/unusable load already has its mandated WARN/ERROR and gains the fields rather than a second line |
| Dump failure | ERROR | `cmd/main` via `persist.Config.OnDumpFailure` | `class`, `detail` | Existing; one per failed attempt, ladder-bounded |
| Startup gate opened | INFO | `cmd/main` via `grpcapi.ServerOptions.OnGateOpen` | `reason` (`sync`/`backstop`) | The transition that makes the node writable; reason distinguishes a healthy join from a timed-out one |
| Peer appeared | INFO | `cmd/main` via `repl.Config.OnPeerEvent` | `node_id`, `handle`, `run_id` | First-seen identity is a lifecycle event |
| Peer restarted | WARN | same | `node_id`, `handle`, `previous_run_id`, `run_id` | A restart is operationally notable; rows are replaced and listings re-requested |
| Peer evicted | WARN | same | `node_id`, `reason` (`peer_timeout`), `last_seen_age` | Peer loss explains gaps in replication; rate is bounded by `peer_timeout`, not per datagram |
| Clock drift | WARN | `cmd/main` via `repl.Config.OnDrift` | `node_id`, `source`, `lead`, `drift_warn` | One per offending peer per limiter interval; metric keeps the per-record truth |
| Fatal startup error | ERROR | `cmd/main` | `stage` (`replication_bind`, `grpc_listen`, `metrics_listen`, `dump_open`), `detail` | Must survive `SUPKV_LOG_LEVEL=error`; replaces `log.Fatalf` |
| Serve error | ERROR | `cmd/main` | `detail` | Then runs the shutdown/final-dump path |

Notes:
- Use `slog` snake_case keys (`sloglint.key-naming-case: snake`); avoid the forbidden keys `time`, `level`, `msg`, `source`.
- Use capitalized static messages (`sloglint.msg-style: capitalized`) and dynamic values only in attributes.
- `run_id`/`previous_run_id` render as hex strings; `handle` is already the `src.String()` opaque handle.
- The startup record is emitted **before** `wire`, so a bind failure still has a preceding "Starting" record; at `SUPKV_LOG_LEVEL=error` the INFO record is filtered and only the ERROR survives.
- Do not log the five env-only metrics certificate paths' *contents* (they are only paths, but the catalog should log `metrics_tls` as a bool and `metrics_addr`; adding `metrics_cert`/`metrics_key` paths is optional and should be skipped to keep the record secrets-free).

#### Typed seams (recommended)

```go
// internal/app/repl — new, nil-safe.
type PeerEventKind uint8

const (
	PeerAppeared PeerEventKind = iota + 1
	PeerRestarted
	PeerEvicted
)

type PeerEvent struct {
	Kind          PeerEventKind
	NodeID        uint16
	Handle        string
	RunID         [16]byte
	PreviousRunID [16]byte
	LastSeenAge   time.Duration // PeerEvicted only
}

type DriftEvent struct {
	NodeID uint16
	Source string
	Lead   time.Duration
}

// Config additions:
OnPeerEvent func(PeerEvent) // nil = no-op
OnDrift     func(DriftEvent) // nil = fall back to Log for compatibility
```

```go
// internal/grpcapi — ServerOptions addition.
OnGateOpen func(reason string) // nil = no-op; reasons: "sync", "backstop"
```

Emission rules:
- `PeerAppeared`: only when `!remembered` in `handleHello` (the first-seen id path that bumps `peerGen`).
- `PeerRestarted`: only when a remembered row's `runID` differs from the incoming run and the run is **not** in `seenRuns` (the alternation path returns before this). Emit after `known.pushSeenRun(known.runID)` so the event carries both ids.
- `PeerEvicted`: inside the `expired` loop in `evictPeers`, before/after the delete; include `last_seen_age = now.Sub(lastHello)`.
- No event for a repeat Hello, a stale/alternating run, or the node's own echo.
- `OnDrift`: called from `applyRemote` and `handleDataSync` through the existing per-(node_id, class) limiter; the counter increments per offending record/part regardless of the limiter. Fallback when `OnDrift == nil`: `cfg.Log` with a prebuilt message (keeps existing test harnesses working).
- `OnPeerEvent`/`OnDrift` are invoked on the single driver goroutine; callbacks must not block (the `OnDumpFailure` contract).

#### Why typed events instead of string assertions

The existing `Config.Log func(string)` is a WARN-only, prebuilt-text seam. It is convenient but forces tests to match substrings. The tests that need to prove the catalog should assert **state transitions** (event kind, node id, run id) and only then assert that the rendering layer turns each event into one `slog` record at the right level. That is the non-brittle split.

#### Testing

- Core: `TestPeerEvents_appearRestartEvict` with a captured event slice; `HandleDatagram` drives the callback synchronously, so the test can read the slice after the call; assert exact sequence and fields; assert no event on repeat Hello and on alternation rejection.
- Gate: `TestGate_openReason` with a fake `SyncArrived` and a callback; table rows for `sync` and `backstop`; the callback fires on `RunGate`'s goroutine, so deliver it over a buffered channel (or guard with a mutex) before reading; assert one callback and `SERVING`.
- Drift: `TestDrift_receivedLeadCounts` (below).
- Main rendering: process tests parse JSON records (`json.Unmarshal` into a struct with `Msg`, `Level`, and pointer fields, as `parseSocketBuffersLine` already does), never line-order or `strings.Contains` on formatted output.
- Fatal bind: process test with an occupied UDP port and `SUPKV_LOG_LEVEL=error`; assert exactly one parsed record with `level=ERROR`, `msg="Startup failed"` and exit 1.
- Load outcome: `TestMain_startupLogLoadOutcome` starts over a seeded dump and over an empty directory; parse the `Dump load` record and assert `outcome=main`/`prompt`.

### D-02 — Drift Warnings Match the Spec

**Recommendation:** fix the signal at the clock, keep the core as the counter/event emitter, and enforce `drift_warn < max_clock_lead`.

#### Code change

`Clock.Update` should evaluate the **received** version's lead against the wall clock, not the local `l`:

```go
// internal/app/clock/clock.go — replacement sketch.
func (c *Clock) Update(remoteL int64, remoteC uint16) (l int64, counter uint16, drift time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	remoteC &= counterMask
	ptMs := c.now().UnixMilli()
	drift = driftBetween(remoteL, ptMs, c.driftWarn) // received lead, not local l
	...
}
```

`Tick` keeps its existing local-vs-wall signal (no production caller counts it), or can be documented as diagnostic-only. `driftBetween` stays signed and saturation-safe; the core counts only `drift > 0`.

`applyRemote` changes from `if drift != 0` to `if drift > 0`, increments `MergeStats.DriftWarnings`, and emits `OnDrift` through the limiter:

```go
if drift > 0 {
	c.mergeStats.DriftWarnings++
	c.emitDrift(now, in.NodeID, in.Source, drift)
}
```

`handleDataSync` currently discards `Observe`'s return:

> `internal/app/repl/core.go:1682-1685`
> ```go
> 	if len(msg.Entries) > 0 {
> 		c.store.Observe(maxEntryVersion(msg.Entries))
> 	}
> ```
> `[VERIFIED: internal/app/repl/core.go:1682-1685]`

Recommendation: capture the returned drift and run the same `> 0` count/event for the part's max entry version. This matches "received version" and XPORT-04's decision that DataSync entry versions are validated too. Document the definition as "received records and listing-part maxima".

#### The unreachable-default trap

The equal defaults are explicit in the flag table:

> `cmd/sup-kv/main.go:158-165`
> ```go
> 	// defaultMaxClockLead is the inbound version bound's launch default
> 	// (XPORT-04): 60 s, the documented drift tolerance. The core normalizes
> 	// a zero or negative value to the same bound; the flag default only
> 	// makes the documented value visible in --help.
> 	defaultMaxClockLead = time.Minute
> 	defaultDumpDebounce = 50 * time.Millisecond
> 	defaultDumpMaxWait  = time.Second
> 	defaultDriftWarn    = 60 * time.Second
> ```
> `[VERIFIED: cmd/sup-kv/main.go:158-165]`

> `internal/app/repl/core.go:99-105`
> ```go
> 	// defaultMaxClockLead is the inbound version bound's design default
> 	// (XPORT-04, D-06): the documented drift tolerance, equal to
> 	// drift_warn's default, so a peer inside the drift window stays
> 	// accepted while a crafted far-future version is rejected before any
> 	// store or clock call. A Config.MaxClockLead of zero or less is
> 	// normalized to this value.
> 	defaultMaxClockLead = 60 * time.Second
> ```
> `[VERIFIED: internal/app/repl/core.go:99-105]`

The current runbook states the intended condition, but its "Cause" line still admits the wrong direction:

> `docs/OPERATIONS.md:215-236`
> ```text
> ## Clock drift runbook
>
> **Symptom.** `supkv_clock_drift_warnings_total` rises on one node while its
> peers stay flat. A warning is counted when an absorbed remote timestamp sits
> beyond the `--drift-warn` threshold ahead of the node's own clock (default
> 60s; environment twin `SUPKV_DRIFT_WARN`).
>
> **Cause.** The node's wall clock runs behind (or ahead of) its peers'.
>
> **Action.** Fix the clock source — NTP or PTP on the host. No sup-kv-side
> repair exists, and none is needed: correctness never depends on wall-clock
> synchronization. Clock skew only affects which simultaneous write wins, and
> it does so deterministically and identically on all replicas. A node whose
> clock runs far ahead will see its writes dominate (win every concurrent
> conflict) until the peers' clocks catch up; once the clock is corrected, new
> writes resume normal ordering. This behavior is proven by the
> failure-scenario suite (simulated far-ahead nodes converge
> deterministically).
>
> Do not restart the cluster to "fix" drift: it changes nothing. Record
> versions are hybrid logical clocks, not raw wall-clock readings, so the same
> version order applies after any restart.
> ```
> `[VERIFIED: docs/OPERATIONS.md:215-236]`

With defaults `drift_warn = 60 s` and `max_clock_lead = 60 s`, `version.Validate` rejects every version that would satisfy `lead > drift_warn`. Required fix:
- Validation relation: `drift_warn < max_clock_lead` (strict; reject equal). Compare at the layout's millisecond resolution — both `version.Validate` and `driftBetween` truncate to milliseconds, so require `max_clock_lead >= drift_warn + time.Millisecond` (a sub-millisecond margin would pass a naive duration comparison but leave no reachable window).
- Default `max-clock-lead` becomes **2 m** (2 × the Spec's 60 s `drift_warn`), keeping the Spec threshold intact and giving a 60 s window where drift warnings fire before the future bound rejects. This is an `[ASSUMED]` default choice (see Assumptions Log).
- Align `repl.defaultMaxClockLead` and the `cmd` default; update `docs/OPERATIONS.md:89-92`, `cmd/sup-kv/main_test.go:438-459`, and `internal/app/repl/discard_test.go:21-25` (the test model's duplicated 60 s constant).

If the project prefers to keep `max-clock-lead=60s`, the only honest alternative is to change the Spec's `drift_warn` default (e.g., 30 s) or document that the drift WARN is unreachable — neither satisfies OBSV-05. Recommend the 2 m change.

#### Runbook alignment

`docs/OPERATIONS.md:215-236` already describes "a version ... beyond the `--drift-warn` threshold ahead of the node's own clock". After the fix that sentence is true for the first time. Edits:
- Cause: remove "or ahead of" — only a *received version ahead of this node's clock* warns. A node whose clock is ahead makes its peers warn.
- Symptom: state that the counter counts received records/listing-part maxima whose version leads local wall time by more than `--drift-warn`; the WARN is rate-limited per peer.
- Action: keep NTP/PTP; add "a node whose clock is ahead will not warn itself; check the peer that WARNs".
- Keep the existing anchor literals (`supkv_clock_drift_warnings_total`, `--drift-warn`, `NTP`); add one anchor literal such as "received version leads" to tie the runbook to the code path (update `cmd/sup-kv/operations_test.go` anchors).

#### Tests

- Clock/unit table: remote lead exactly `drift_warn` → 0; `drift_warn+1ms` → positive; wall ahead of remote → 0 (negative is not counted); zero/negative warn disables.
- `TestClock_driftSignalSaturates`: the Update subtest currently calls `clk.Update(testNowMs, 0)` with a saturated local `l`; under received-lead semantics that becomes 0. Rebuild the Update case so `remoteL` is the far-ahead operand (e.g., `clk.Update(maxL, 0)` with a normal wall) and keep the Tick saturation case as-is.
- Core: a far-ahead DataReplication within `max_clock_lead` → one WARN event + `DriftWarnings=1`; a second within the limiter interval → metric 2, one WARN; a record with wall ahead → metric unchanged; a record beyond `max_clock_lead` → `bad_version`, metric unchanged.
- DataSync: one part with a far-ahead max version → metric + WARN; listing-only drift is no longer "clock-only" (supersedes the Phase-4 OQ6 comment in `merge_stats_test.go:87-94`; update that comment/test).
- Process: `TestMain_driftWarn` drives a crafted/real far-ahead record and parses the WARN record; stand is optional if the unit + process coverage is already exact.
- Registry: flip `S-HLC-13` and `S-PAR-16` after the fix.

### D-03 — One Config Package and One Validation Function

**Recommendation:** create `internal/config` with exported `Config`, `Parse`, `Validate`, `Usage`, and the default constants. `cmd/sup-kv/main.go` keeps `main()` and wiring only.

#### API sketch

```go
package config // internal/config

type Config struct {
	NodeID uint16
	BindAddr, BroadcastAddr, DumpPath, GRPCAddr, MetricsAddr string
	StartupGate, StartupGateEmpty, WaitTimeout time.Duration
	ValueMaxBytes, SyncChunkBytes, SyncResponseRate int
	HelloInterval, PeerTimeout, AckTimeout, AckTimeoutMax time.Duration
	RetransmitInterval, SyncInterval, MaxClockLead time.Duration
	DumpDebounce, DumpMaxWait, DriftWarn time.Duration
	LogLevel slog.Level
	LogFormat string
	MetricsCert, MetricsKey, MetricsClientCA string
	ShowVersion bool
}

var ErrHelp = errors.New("help requested")

func Parse(args []string, getenv func(string) string) (Config, error)
func Validate(c Config) error
func Usage(w io.Writer)
```

- `Parse` uses `flag.NewFlagSet("sup-kv", flag.ContinueOnError)` with `fs.SetOutput(io.Discard)`; main owns all output.
- Env values become flag defaults; a malformed env value is collected with the env name, exactly like today's `fail(env, err)`.
- `--help`/`-h` returns `ErrHelp`; main calls `config.Usage(os.Stdout)` and exits 0.
- `--version` sets `ShowVersion`; Parse returns it before validation, preserving "version works even on a broken environment".
- Positional arguments: after flags, if `len(fs.Args()) > 0`, return an error naming every positional arg (unless `ShowVersion`/`ErrHelp`, which short-circuit).
- On any parse/validation error, main prints `sup-kv: <err>` and `config.Usage(os.Stderr)` then exits 2. Error text keeps the existing substrings the process tests assert (`datagram ceiling`, `value-max-bytes must be positive`, `startup gate must be positive`, `sync-chunk-bytes exceeds the datagram ceiling`, `node-id`).

#### Rule table (one function, table-driven)

| # | Rule | Invalid input | Error names |
|---|------|---------------|-------------|
| 1 | required knobs | `--node-id` 0, empty bind/broadcast/dump | the flag(s) |
| 2 | node id range | `> 65535` | `--node-id` |
| 3 | every duration/rate positive | `<= 0` for all 13 durations + `sync-response-rate` | the flag |
| 4 | retransmit floor | `retransmit_interval < 1ms` | `--retransmit-interval` |
| 5 | sync response cap | `sync_response_rate > 1e9` | `--sync-response-rate` |
| 6 | sync chunk bounds | `sync_chunk_bytes <= 0` or `> repl.MaxSyncChunkBytes` | `--sync-chunk-bytes` |
| 7 | value max positivity | `value_max_bytes <= 0` | `--value-max-bytes` |
| 8 | datagram ceiling | largest record > 65 507 | `--value-max-bytes` |
| 9 | peer vs hello | `peer_timeout <= hello_interval` | both flags |
| 10 | ack max | `ack_timeout > ack_timeout_max` | both flags |
| 11 | drift vs lead | `max_clock_lead < drift_warn + 1ms` (millisecond-resolution strict) | both flags |
| 12 | metrics cert triple | exactly one/two of the three envs | all three env names |
| 13 | log level/format parse | malformed `SUPKV_LOG_*` | env name |
| 14 | positional args | any non-flag args | the args |
| 15 | address shape (optional) | non-empty addr without `host:port`/valid port | the flag |

Implementation notes:
- A `rule` slice of `{name, check func(Config) error}` makes "one function" literal and lets the table test iterate every rule. `Validate` returns `errors.Join` of all failures (operator sees every bad knob in one pass) or the first; recommend join, with one row per rule still asserting its own message.
- Export `repl.MaxSyncResponseRate` (currently `maxSyncResponseRate` at `internal/app/repl/core.go:112-119`) so the config package does not duplicate `1e9`. The existing `MaxSyncChunkBytes` is already exported.
- The datagram check keeps `udprepl.MaxRecordDatagramBytes(storage.MaxKeyBytes, probe)` and the `>` boundary semantics; move `checkDatagramCeiling` body into the config rule.
- `sync-chunk-bytes` lower bound: `> 0`; the core's clamp stays. Do not invent a minimum above `chunkOverheadBytes` (the core intentionally allows a single-entry oversized chunk).
- `retransmit-interval` floor `1ms` is a design choice; it must be documented and covered by a rule test. `[ASSUMED]`

#### `--help` env-only block

`Usage` prints the flag table (`fs.PrintDefaults()` preserves the existing `-flag ... (env SUPKV_X)` and `(default ...)` lines), then:

```text
Environment-only (no flag):
  SUPKV_LOG_LEVEL       debug, info, warn or error (default info)
  SUPKV_LOG_FORMAT      json or text (default json)
  SUPKV_METRICS_CERT    server certificate path (all three or none)
  SUPKV_METRICS_KEY     server key path (all three or none)
  SUPKV_METRICS_CLIENT_CA  client CA path (all three or none)
```

The exact five names are the test contract.

#### Tests

- `internal/config` table test: one subtest per rule, each with a fully valid Config except the one field(s) under test; assert the parameter name and sentinel. Use named constants for expected numbers (mnd).
- Meta-test: the rule table's `len` is pinned; every rule name appears in at least one subtest (prevents adding a rule without a test).
- `Parse` tests: flag > env > default; malformed env; `--version` with a malformed env; `--help` returns `ErrHelp`; positional args rejected; unknown flag rejected.
- `Usage` test: all five env-only names and every flag name appear.
- Process tests: keep the existing exit-2 tests (they now hit the config package); add rows for a relation violation (`peer-timeout=1s --hello-interval=1s`), `retransmit-interval=1ns`, `sync-response-rate=1000000001`, positional arg, and `drift-warn=2m --max-clock-lead=2m`.
- `persist` test: `EncodeTo(&buf, nil, maxDumpValueBytes+1)` returns an error and `buf.Len()==0`; same for 0/-1; valid value writes.

### D-04 — Folding the Routed Deferrals

1. **`sync_chunk_bytes` bounds (WR-02).** The specific `--sync-chunk-bytes=1125899906842624` overflow no longer reaches the core: `checkSyncChunkBytes` (WR-06) currently rejects anything above `repl.MaxSyncChunkBytes` (65 507) before bind. WR-02's remaining obligation is therefore the one-point location and the non-positive bound, not a new panic fix. Add the lower bound and the `sync-response-rate` cap/positivity in the same table; keep the core clamp for library callers. Close the Phase-3 ledger row with a pointer to the config rule and its test.
2. **Library value-limit guard (IN-04).** Add to the top of `persist.EncodeTo`:
   ```go
   if valueMaxBytes <= 0 || valueMaxBytes > maxDumpValueBytes {
       return fmt.Errorf("persist: value limit %d is outside (0, %d]: %w",
           valueMaxBytes, maxDumpValueBytes, storage.ErrValueTooLarge)
   }
   ```
   Do this before `enc.Encode(header{...})`. Test the empty buffer on rejection; update `doc.go` to say the guard now exists.
3. **Apply-failure WARN (IN-01).** Change the message to omit the key and route it through the shared limiter with class `apply_failure`:
   ```go
   c.warnRateLimited(now, in.NodeID, warnClassApplyFailure, fmt.Sprintf(
       "sup-kv: applying replicated record from handle %q: %v", source, err))
   ```
   The limiter suppresses only the log; counters stay per datagram. Add a test with a fake store returning `storage.ErrValueTooLarge` for repeated datagrams: one WARN per peer/interval, no key name, no Ack, no supersession.
4. **Unbound-listing class (New-2).** Add `DiscardUnboundListing = "unbound_listing"`, `DiscardStats.UnboundListing`, an adapter atomic/mirror/accessor, a `supkv_datagrams_discarded_total{class="unbound_listing"}` sample, and route `listingBound` through it (same per-(node, class) limiter). Drop "configuration error" from this message — an unbound listing can legitimately overtake its Hello; reserve that phrase for the two identity paths and the alternation path. Update the totality model and every test that currently expects `DuplicateNodeID` for an unbound listing:
   - `internal/app/repl/listedkeys_internal_test.go:144-145,210-211`
   - `internal/app/repl/discard_test.go:631` (the sum), plus the class-string switches at `:328-349`, `:452-453`, `:738-740`
   - `internal/udprepl/decode_negative_test.go:650-651,680-681`
   - `internal/udprepl/scenario_test.go:333-334`
   - `internal/metrics/metrics_test.go` zero-label list
   - `docs/OPERATIONS.md` label list
   The `duplicate_node_id` label stays for the two identity paths and the alternation path; only the unbound-listing site moves.

#### Registry ripple (docs/SPEC-DEVIATIONS.md)

Phase 7 closes three rows and should update their evidence after the code lands; Phase 9 owns the full registry audit, but leaving a fixed-in-v1.1 row stale after its fix is exactly the drift DOCS-09 guards against.

| Row | Current status | Phase-7 action |
|-----|----------------|----------------|
| `S-HLC-13` | `fixed-in-v1.1(OBSV-05)` with old `core.go:532-534`/`clock.go:159-176` links | Flip to `implemented` after the drift fix; cite the new `Clock.Update` lead check, `MergeStats.DriftWarnings`, the WARN seam and the drift tests |
| `S-PAR-16` | `fixed-in-v1.1(OBSV-05)` with stale `main.go:146,347` links | Flip to `implemented`; cite the 60 s `drift_warn` default, the new `max-clock-lead` default, and the `drift_warn < max_clock_lead` rule |
| `S-FAIL-17` | `fixed-in-v1.1(OBSV-06)` ("required log entry at debug") | Flip to `implemented`; cite the WARN path at `core.go:1031-1033` and the level test |
| `S-SEM-07` | `superseded-by-amendment` (journal removed) | Update the note: the counter now counts pending-local displacement, not every displaced/rejected version (ADR 0004 / OBSV-03) |
| `S-IMP-08` | `accepted-deviation(ADR 0008)` (endpoint off by default) | Keep the decision; update the evidence sentence to say the counters are now honest (OBSV-03/05), while the endpoint remains opt-in |

Do not touch `S-HLC-07` (ADR 0012) or the unimplemented v2 rows.

### D-05 — Honest, Test-Confirmed Metric Definitions

#### Definitions to change or add

| Family | Type | Definition to state in Help + OPERATIONS | Test |
|--------|------|------------------------------------------|------|
| `supkv_merges_conflicts_total` | counter | Pending (unacked) local writes displaced by a replicated value. Ordinary overwrites of confirmed/remote records and rejected incoming records do not count. | Unit: pending local displaced → 1; confirmed/local acked overwrite → 0; stale incoming → 0; equal tie-break win displacing pending → 1; equal tie-break loss → 0. Stand adapted. |
| `supkv_replication_datagrams_sent_total` | counter | Outbound datagrams successfully handed to the socket, by message type. Encode failures, resolve failures and `WriteToUDP` failures do not count. UDP "sent" is not delivery. | `flush` with a fake `packetConn` returning `WriteToUDP` error: counter unchanged; success: +1 per message; resolve failure unchanged. |
| `process_resident_memory_bytes` | gauge | Current resident set size in bytes (Linux `/proc/self/statm`; development-only darwin fallback documented). | Parser table + Linux integration `currentRSS() > 0`; source is not `ru_maxrss`. |
| `supkv_dumps_duration_seconds` | gauge | Duration of the last dump **attempt** (encode + save) in seconds; failed attempts included. | `DumpStats{DurationSeconds:1.25}` renders 1.25; scheduler test already covers both outcomes; Help asserts "last dump attempt". |
| `supkv_clock_drift_warnings_total` | counter | Received records/listing-part maxima whose version leads local wall time by more than `drift_warn`. | Drift tests above. |
| `supkv_replication_listings_dropped_total` | counter | Targeted listing triggers refused by the destination/chunk cap; the trigger is retried later. | Metrics zero/value + core accessor test. |
| `supkv_replication_sync_streams_dropped_total` | counter | Partial `DataSync` streams evicted by the accumulator bounds. | same |
| `supkv_replication_listed_keys_evicted_total` | counter | Retained listed keys released by whole-peer eviction or expiry; a key stays counted once per release. | same |
| `supkv_replication_notifications_regressed_total` | counter | Local-write notifications ignored because their version was not greater than the tracked version (regressive and equal duplicates). | same |
| `supkv_datagrams_discarded_total{class="unbound_listing"}` | counter | Listings from a sender with no matching peer-table row/run; dropped before clock/accumulation. | New-2 tests |
| `supkv_dump_load_total` | counter | Startup load outcome (unchanged). | existing |
| other existing families | — | Keep current definitions unless the audit finds an overstatement. | existing `TestMetrics_values` + new Help table |

The four Phase-4 counter names are new choices and therefore `[ASSUMED]`; they follow the `supkv_replication_*` prefix and `_total` suffix convention already used. The frozen red tests only pin `supkv_replication_unacked_records` and `supkv_replication_datagrams_retransmitted_total`, so these new names are unconstrained.

#### Conflict counter algorithm

Count displaced **pending** entries, not `len(conflicts)`:

1. Call `ApplyRemote`; on error, no count (already the behavior).
2. Determine whether the incoming record won: after a successful apply, `stored, ok := c.store.Lookup(msg.Key)` and `storage.RecordsEqual(stored, msg.Record)`. This is already the reply-verdict probe later in the function; reuse one read.
3. Walk `c.pending` for entries under `msg.Key`:
   - `incoming.Version > pending.version` → the pending write is displaced; emit `Superseded` (existing) and count 1.
   - `incoming.Version == pending.version` and the incoming record won the tie-break → the pending write is displaced; emit `Superseded` and count 1. (This extends the current `>`-only loop; an equal-version tie-break that displaces a pending local write is a real loss.)
   - `incoming.Version < pending.version` or equal-and-incoming-lost → leave the pending entry and count nothing.
4. Do not count a rejected incoming record, a no-op identical redelivery, or an ordinary overwrite whose stored loser is not in `pending`.

This preserves the `Outbox` contract (`Superseded` is still the waiters' honest false) and makes `MergeStats.Conflicts` equal to the number of pending versions displaced by the apply. `storage.join`'s `Conflict` slice remains available for other callers but is no longer the metric's source.

#### Four-counter wiring

- Add to `metrics.Sources`:
  ```go
  ListingDropped        func() uint64
  SyncStreamDropped     func() uint64
  ListedKeysEvicted     func() uint64
  NotificationsRegressed func() uint64
  ```
- `Adapter`: four `atomic.Uint64` fields, populated in `mirrorStats` from `a.core.*()`, four accessors. No new core methods needed (accessors exist).
- `metrics.Snapshot`: one new `listingFamilies(...)` returning the four families with zero-valued samples; add them to the `families` test table and the zero-state sample list.
- `cmd/main.metricsSources`: wire the four accessors.
- Tests: `metrics` fake-Sources value test (all four render exact values); adapter mirror test for at least `NotificationsRegressed` (trigger with two equal `NotifyLocalWrite` calls) and one of the listing counters; core accessor tests already exist.

#### RSS implementation

- `internal/metrics/rss_linux.go` (`//go:build linux`): read `/proc/self/statm`, parse field 2 (`resident` pages), multiply by `os.Getpagesize()`.
- `internal/metrics/rss_other.go` (`//go:build !linux`): `syscall.Getrusage` `ru_maxrss`, documented as the development-only darwin fallback (ADR 0010). If the project prefers strict honesty over availability, omit the family on non-Linux and document; recommend the fallback with a Help that says "Linux current; darwin dev-build peak".
- Keep `processUsage`'s CPU half where it is; split so `runtimeFamilies` calls `currentRSS()`.
- Test `/proc` parsing with fixture strings and a Linux-only integration assertion; do not compare against `ru_maxrss` (a peak can equal current, making the test vacuous).

#### One test per definition

Add `TestMetrics_definitions` with one subtest per `supkv_*` family: parse the `# HELP` line from `Snapshot` and assert a short semantic phrase (e.g., conflict → "pending", sent → "successfully", duration → "last dump attempt", drift → "leads", RSS → "current"). This is a documentation-drift guard, not a truth proof; the behavior tests above are the truth. Keep the existing `TestMetrics_families` and `TestMetrics_values`.

### D-06 — Exit Codes and Ordered Shutdown

#### Exit table

| Situation | Code | Emitted record | Notes |
|-----------|------|----------------|-------|
| `--help`, `--version` | 0 | none / stdout version | `ErrHelp` and `ShowVersion` short-circuit |
| Clean SIGTERM/SIGINT shutdown, final dump confirmed | 0 | none (or INFO shutdown if added) | Existing |
| Usage/config error | 2 | stderr error + usage (not slog) | `config.Parse/Validate`; names the parameter |
| Startup failure (bind, listen, metrics TLS/config, dump open) | 1 | ERROR `Startup failed` | Replaces `log.Fatalf`; must survive `SUPKV_LOG_LEVEL=error` |
| Serve error, final dump confirmed | 1 | ERROR `Serve failed` | Still runs `shutdown` |
| Final dump failed (including a Serve-error path) | 3 | ERROR `Dump failed`/final-dump error | Distinct operator contract; systemd `Restart=on-failure` still reacts |
| SIGKILL/host crash | n/a | none | Confirmed writes already durable |

Precedence: final-dump failure (3) wins over a Serve error (1); a startup failure never runs a final dump (no client-confirmed writes yet). Document in OPERATIONS.

#### Shutdown order (unchanged D-19 order, with signal earlier)

1. `armSignals()` **before** `wire` (before dump load and bind).
2. Parse/validate config, install logging, emit startup record.
3. `wire`: `persist.Open` (load), UDP bind + socket buffers, gRPC listener, metrics listener; `wire` error → compensate, ERROR, exit 1.
4. `serve`: select on signal or `gs.Serve` error.
5. On either: `api.Shutdown` (one-way health), metrics server shutdown, `apiCancel`, bounded `GracefulStop`/`Stop`, `replCancel` + await, `FlushTimeout(shutdownGrace)`, `engineCancel` + bounded await, `signal.Stop`.
6. Exit 0/1/3 per the table.

`Serve` error currently returns from `main` before `shutdown`; the plan must call `shutdown` and only then decide the code. If `shutdown` also fails, return 3.

#### Testable `main`

Recommended shape (small and injectable):

```go
type runtime struct {
	installSignals func() <-chan os.Signal
	wire           func(config.Config) (*wired, net.Listener, error)
	serve          func(*wired, net.Listener, <-chan os.Signal) error
	shutdown       func(*wired, <-chan os.Signal) error
}

func run(cfg config.Config, rt runtime) int
```

Tests:
- `TestRun_signalInstalledBeforeWire`: record call order.
- `TestRun_serveErrorStillShutsDown`: `serve` returns an error; `shutdown` records a call; exit 1.
- `TestRun_dumpFailureCode`: `shutdown` returns an error; exit 3.
- `TestRun_serveAndDumpFailure`: exit 3.
- `main()` becomes parse → if help/version handle → `setupLogging` → `os.Exit(run(cfg, productionRuntime))`.

`shutdown` can stay a method on `*wired`; the injectable `shutdown` function in `runtime` makes the orchestration testable without changing the production sequence.

#### Docs/tests ripple

- `docs/OPERATIONS.md:280-310`: exit 0/1/2/3 table; update "final dump failed" from 1 to 3 and the "keeps the failed-final-dump exit code 1" sentence to 3.
- `cmd/sup-kv/operations_test.go` anchors: change `exit code 1` pin to `exit code 3` and add `exit code 1` as startup failure; keep `exit code 2`.
- `cmd/sup-kv/main_test.go:879-908` `TestMain_duplicateBind` keeps asserting 1 (startup failure).
- Add final-dump process test: start a node with a dump directory, await serving, `chmod 0500` the directory (restore in cleanup), SIGTERM, assert exit 3 and one ERROR record. `ensureDir` `Stat`s the existing dir and does not recreate it, so the temp-file write fails with EACCES.
- `--help` process test: assert all five env-only names, plus the existing flag/env/default substrings.

### D-07 — Depguard Rules Additive and Probe-Verified

**Recommendation:** add three rules after the existing project rules (never edit lines 1-64 upstream):

```yaml
        sup-kv-adapters-not-in-core:
          files:
            - '**/internal/app/**'
            - '**/internal/port/**'
            - '!$test'
          deny:
            - pkg: github.com/Djarvur/sup-kv/internal/udprepl
              desc: adapters are wired from cmd; the pure core/ports may not import them
            - pkg: github.com/Djarvur/sup-kv/internal/persistfile
              desc: adapters are wired from cmd; the pure core/ports may not import them
            - pkg: github.com/Djarvur/sup-kv/internal/grpcapi
              desc: adapters are wired from cmd; the pure core/ports may not import them
            - pkg: github.com/Djarvur/sup-kv/internal/metrics
              desc: adapters are wired from cmd; the pure core/ports may not import them
            - pkg: github.com/Djarvur/sup-kv/internal/stand
              desc: test infrastructure is not a production dependency
        sup-kv-metrics-no-adapters:
          files:
            - '**/internal/metrics/**'
            - '!$test'
          deny:
            - pkg: github.com/Djarvur/sup-kv/internal/udprepl
              desc: the exporter reads value types; it must not import the socket adapter (move SentByType to app/repl)
            - pkg: github.com/Djarvur/sup-kv/internal/persistfile
              desc: the exporter reads the engine's stats, not the dump adapter
            - pkg: github.com/Djarvur/sup-kv/internal/grpcapi
              desc: the exporter must not import the API adapter
        sup-kv-grpcapi-no-repl-metrics:
          files:
            - '**/internal/grpcapi/**'
            - '!$test'
          deny:
            - pkg: github.com/Djarvur/sup-kv/internal/udprepl
              desc: the API adapter consumes the port seams, never the replication adapter
            - pkg: github.com/Djarvur/sup-kv/internal/metrics
              desc: the API adapter must not import the exporter
        # Allowed exception (Q-05), written justification:
        # sup-kv-grpcapi -> internal/persistfile is allowed for the two snapshot
        # sentinels (ErrInvalidSnapshotName, ErrNoDump) and sup-kv-grpcapi ->
        # internal/app/persist for the *persist.Engine store facade (12 methods,
        # plus the Watch hub's Records/Set). Replacing the engine with a local
        # interface is a larger refactor with no Phase-7 requirement; the
        # exception is deliberate and visible here rather than hidden in code.
```

Important:
- Do **not** write a rule denying `github.com/Djarvur/sup-kv/internal` in `app`/`port`: `port` legitimately imports `internal/app/storage` and `internal/app/version`, and `app` packages import each other. Name adapters individually.
- `internal/app/internal/testutil` imports `internal/testutil`; do not deny `internal/testutil` in the app rule (it is a shared benchmark helper, not an adapter).
- `$test` exemption keeps fakes and scenario harnesses free to import adapters.
- `internal/config` is composition-level and may import `udprepl`, `persist`, `storage`, `repl`; no rule covers it.
- Any future boundary change is an additive `.golangci.yml` diff; the vendored body (lines 1-64) stays byte-identical and `make lint` runs `golangci-lint config verify`.

#### Import audit (why a naive rule breaks)

| Current import | Would fail a naive rule | Correct treatment |
|----------------|--------------------------|-------------------|
| `internal/app/persist` → `internal/port`, `internal/app/{clock,storage,version}` | deny-own-module | allow (own module, same layer) |
| `internal/app/repl` → `internal/port`, `internal/app/{storage,version}` | deny-own-module | allow |
| `internal/app/storage` → `internal/app/version` | deny-own-module | allow |
| `internal/port` → `internal/app/{storage,version}` | strict stdlib-only without own-module | already allows `github.com/Djarvur/sup-kv` |
| `internal/app/internal/testutil` → `internal/testutil` | deny any `internal/*` | do not include `internal/testutil` in the deny list |
| `internal/grpcapi` → `internal/app/persist`, `internal/persistfile`, `api/supkvapi/v1` | deny all adapters / deny internal | only deny `udprepl`/`metrics`; allow `persistfile`/`app/persist` with justification |
| `internal/metrics` → `internal/app/{persist,repl}`, `internal/port`, `internal/udprepl` | deny internal | remove only `udprepl` (move `SentByType`), keep core imports |
| `internal/udprepl` → `internal/app/{repl,storage,version}`, `internal/port`, `replproto` | deny subpackages | allow (adapter→core is the normal direction) |
| `cmd/sup-kv` → everything | any rule with `$all` | no rule; wiring imports adapters |
| `internal/stand` test files → adapters/engine | `!$test` missing | always include `!$test` on production rules |

#### Depguard probe

The Phase-6 precedent is the method: create a temporary non-test file that imports the forbidden package, run the focused lint, assert the rule name appears in the output, then delete the file. Do it for each new rule:
- `internal/app/probe_boundary.go` importing `internal/udprepl` → expect `sup-kv-adapters-not-in-core`.
- `internal/metrics/probe_boundary.go` importing `internal/udprepl` → expect `sup-kv-metrics-no-adapters`.
- `internal/grpcapi/probe_boundary.go` importing `internal/udprepl` → expect `sup-kv-grpcapi-no-repl-metrics`.
- Verify `make lint` is green after deletion and the tree is clean.

### D-08 — `udprepl` Split and Wiring-Only `main`

**Recommendation:** split `internal/udprepl/repl.go` into four files in the **same package**; do not create subpackages, do not export new names, do not edit tests.

| New file | Moves | Notes |
|----------|-------|-------|
| `socket.go` | `packetConn`, `options`, `Option`, `WithAddressReuse`, `listenDatagram`, `configureSocketBuffers`, `validateBufferRequest`, `socketOption`, `checkSendBuffer`, bind/buffer portion of `New`, `SocketBuffers` | The `New` constructor stays where it is or is split into `New` (loop/core) + `newSocket` (socket); either preserves unexported symbols and tests |
| `codec.go` | `maxNodeIDWire`, `wireIDSize`, `errNoStandaloneBody`, `encodeDatagram`, `decode`, `protoBody`, `portMessage`, `recordVersion` | Pure functions; no dependency on `Adapter` |
| `loop.go` | `inboundEvent`, `localWrite`, `Run`, `readLoop`, `logReadError`, `flush`, `countSent`, `write`, `armTimer`, `timerChan`, `stopTimer`, `NotifyLocalWrite`, `NotifyReplayedWrite`, `enqueueLocalWrite`, tracker/wait methods, `applyResults` | The single-driver event loop and its queues |
| `mirror.go` | mirror fields of `Adapter`, `listingMirror`, `mirrorStats`, the accessors that read atomics (`Discards`, `KnownPeers`, `KnownPeerIDs`, `UnackedStats`, `MergeStats`, `Retransmissions`, `PacedSends`, `SendQueue`, `EgressDropped`, `SentByType`, `SyncArrived`, `ListedKey`, `ReadErrors`, new four accessors) | The publication layer |
| `adapter.go` (optional) | `Adapter` struct, `New`, package doc | Keeps the composition root legible; `New` may remain in `socket.go` if the plan prefers fewer files |

Constraints:
- Same package `udprepl`; unexported identifiers stay unexported, so `repl_internal_test.go`, `socket_internal_test.go`, `readloop_internal_test.go`, `decode_negative_test.go`, `scenario_test.go`, `repl_test.go` compile unchanged. D-08's "public behavior and tests stay unchanged" is satisfied by a file-only move.
- Do not change the `EncodeDatagramForTest`/`DecodeDatagramForTest` internal helpers or the `packetConn` interface shape; those are test contracts.
- Keep the single-driver invariant: only `Run` mutates `core`; `mirrorStats` remains called after every loop event.
- After the split, `internal/metrics` must no longer import `udprepl` (D-07); move `SentByType` to `internal/app/repl` and keep `type SentByType = repl.SentByType` in `udprepl` if any test references the old name. If no test references it, update the two `metrics` references and remove the alias.
- New tests "proving the new units": a codec round-trip table already exists; add only focused tests if a unit boundary is not covered (e.g., `TestCodec_encodeDecodeRoundTrip` is already there). Do not rewrite existing suites.

**`main` decomposition:** move `config`, `registerConfig`, `registerDurationFlags`, `resolveMetricsKnobs`, `finalizeConfig`, `check*`, `missingRequired`, env helpers and defaults into `internal/config`; `main.go` keeps `version`/`commit`, exit codes, `run` orchestration, `wire`, `startEngine`, logging functions, `serve`, `shutdown`, `awaitEngineStop`, and `wiring`. Target: no flag registration, no validation, no default table in `main.go`.

**Sequencing:** do the file split **after** the metrics/event changes that touch `repl.go`, or do it first and then apply the metric/event edits to the new files. Recommend split first (mechanical, zero behavior) so later diffs are small; run `go test ./internal/udprepl/` immediately after to prove no behavior change. `main` extraction and `udprepl` split are independent; plan them in separate waves to keep commits reviewable.

## Architecture Patterns

### System Architecture Diagram

```text
                         args + SUPKV_* env
                                 |
                                 v
                    +----------------------------+
                    | internal/config            |
                    |  Parse -> Validate         |  exit 2 + usage on any error
                    |  defaults / relations      |
                    +-------------+--------------+
                                  | Config
                                  v
                    +----------------------------+
                    | cmd/sup-kv main/run        |
                    |  install signals FIRST     |
                    |  setupLogging (slog)       |
                    |  startup INFO (version,cfg)|
                    +------+---------------+-----+
                           |               |
             persist.Open  |               |  udprepl.New (bind, buffers)
       (load stats/generation)              |
                           v               v
                 +------------------+   +------------------------+
                 | app/persist      |   | udprepl.Adapter        |
                 | store + dumper   |<--| event loop (one driver)|
                 | confirmer        |   | codec/socket/mirror    |
                 +--------+---------+   +-----+------------+-----+
                          |                   |            |
             OnDumpFailure|        core events|            | accessors
             (ERROR)      v                   v            v
                 +------------------+   +---------------------+
                 | cmd slog render  |<--| repl.Config         |
                 | lifecycle catalog|   | OnPeerEvent/OnDrift |
                 +------------------+   +---------------------+
                          ^
                          | OnGateOpen(reason)
                 +--------+---------+
                 | grpcapi.RunGate  |
                 | health one-way   |
                 +------------------+

 runtime metrics path:
 core counters/accessors -> Adapter atomics (mirrorStats)
   -> metrics.Sources (closed set) -> metrics.Snapshot -> /metrics text
```

The diagram is conceptual: file ownership is in the Component Responsibilities table (D-07/D-08 above), not in the boxes.

### Recommended Project Structure

```text
internal/
├── config/                 # NEW: Config, Parse, Validate, Usage, defaults, rule table
│   ├── config.go
│   ├── validate.go
│   ├── usage.go
│   └── *_test.go
├── app/
│   ├── repl/               # typed PeerEvent/DriftEvent, conflict/drift semantics, four accessors
│   ├── persist/            # EncodeTo guard, dump scheduler (unchanged otherwise)
│   └── storage/            # join unchanged (metric no longer consumes its Conflict slice)
├── metrics/                # new families, RSS files, definitions test
│   ├── metrics.go
│   ├── rss_linux.go
│   ├── rss_other.go
│   └── *_test.go
├── udprepl/                # same package, four files (socket/codec/loop/mirror/adapter)
│   ├── socket.go
│   ├── codec.go
│   ├── loop.go
│   ├── mirror.go
│   └── adapter.go          # optional
├── grpcapi/                # OnGateOpen option
└── port/                   # unchanged (no new interface required)
cmd/sup-kv/                 # main.go wiring + run orchestration + log rendering
```

### Pattern 1: Typed lifecycle event seam

**What:** the core detects a state transition and calls a typed callback; `cmd/main` renders it to `slog`.
**When to use:** peer appear/restart/evict, drift, gate opening.
**Example:**

```go
// internal/app/repl (design sketch)
type PeerEvent struct {
	Kind          PeerEventKind
	NodeID        uint16
	Handle        string
	RunID         [16]byte
	PreviousRunID [16]byte
	LastSeenAge   time.Duration
}

// in handleHello, after the row is updated:
if !remembered {
	c.notifyPeer(PeerEvent{Kind: PeerAppeared, NodeID: in.NodeID, Handle: in.Source, RunID: in.RunID})
}
```

**Why:** tests assert the transition once; the rendering test asserts level/fields once; formatting changes cannot break behavioral tests.

### Pattern 2: One pure-ish validation rule table

**What:** `Validate(Config) error` iterates a `[]rule` where each rule returns a parameter-named error; `Parse` only builds the Config.
**When to use:** every startup knob and relation.
**Example:**

```go
type rule struct {
	name  string
	check func(Config) error
}

var rules = []rule{
	{"--node-id", func(c Config) error {
		if c.NodeID == 0 {
			return ErrMissingRequired
		}
		return nil
	}},
	{"--peer-timeout/--hello-interval", func(c Config) error {
		if c.PeerTimeout <= c.HelloInterval {
			return ErrPeerTimeoutRelation
		}
		return nil
	}},
	// ...
}
```

**Why:** one place for ranges/relations, one table test, one usage path. The core's normalization remains defense in depth, not the contract.

### Pattern 3: Mirror-then-export

**What:** core counters are mutated only on the driver goroutine; the adapter copies them into atomics after every loop event; `metrics.Sources` reads the atomics once per scrape; the renderer sorts and formats.
**When to use:** every new counter.
**Example:** the existing `mirrorStats` pattern (`repl.go:1158-1215`), extended with the four new counters and `unbound_listing`.
**Why:** no cross-goroutine core reads, no registry, stable scrape output.

### Pattern 4: Build-tagged platform probe

**What:** `rss_linux.go` implements current RSS from `/proc/self/statm`; `rss_other.go` implements the documented darwin fallback; a common `runtimeFamilies` calls the one function.
**When to use:** any OS-specific metric.
**Why:** `syscall.Getrusage` is portable but peak-only; the build tags keep the Linux honesty without breaking the darwin build/vet cell.

### Pattern 5: Injectable orchestration for exit/shutdown tests

**What:** `run(cfg, runtime)` with `installSignals`, `wire`, `serve`, `shutdown` function fields; production wires the real functions.
**When to use:** order and exit-code tests that cannot be forced from outside the process (Serve error).
**Why:** process tests cover the real binary; unit tests cover combinations that would otherwise be untestable without test hooks in production.

### Pattern 6: Additive depguard rules with a probe

**What:** append project rules after the vendored body; name adapter packages explicitly; verify each rule with a temporary forbidden import and the rule name in lint output.
**When to use:** any boundary change.
**Why:** `config verify` proves schema validity, the probe proves enforcement; the upstream body stays byte-identical.

### Anti-Patterns to Avoid

- **Logging from `internal/app` directly.** It couples the pure core to a rendering choice and makes text-vs-JSON/level tests brittle. Emit typed events; render in `cmd`.
- **Asserting `slog` output by full-line string equality or substring on formatted text.** Parse JSON records and assert fields; use text mode only for a format-specific test.
- **Counting sends at enqueue or before the socket call.** A queued or failed datagram is not sent; count at the successful `WriteToUDP` return.
- **Using `mergeStats.Conflicts += len(conflicts)`.** That is the ordinary-overwrite counter the review rejected.
- **Using `ru_maxrss` for `process_resident_memory_bytes`.** It is peak; the Linux current source is `/proc/self/statm`.
- **Denying `github.com/Djarvur/sup-kv/internal` in app/port.** It breaks legitimate same-layer imports; name adapters.
- **Making `internal/config` import `cmd` or `net` for address validation.** Let adapters resolve/bind; config only checks shape if needed.
- **Putting `os.Exit` in a library/package.** Only `cmd/main` decides the code.
- **Editing `.golangci.yml` lines 1-64 or the frozen red tests/runner.** Both are byte-frozen contracts.
- **Adding labels with keys, node ids or values.** The label vocabulary stays the fixed enum sets.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Flag parsing / precedence / help | A custom argv parser | stdlib `flag.FlagSet` with env-derived defaults | Existing flag > env > default contract, built-in help and error handling |
| JSON/text log formatting and level filtering | A custom logger/formatter | `log/slog` handlers | Already installed once; `slog.SetDefault` bridges the `log` package |
| Prometheus text exposition | A new metrics library or custom escaping | Existing `internal/metrics.Render` | Sorted families, HELP/TYPE, label escaping already tested |
| Per-class WARN rate limiting | A new limiter | Existing `warnRateLimited`/`discardWarnAt` | Shared `(node_id, class)` semantics and event-time discipline |
| Dump retry cadence | A new backoff loop | Existing scheduler ladder (100 ms → 5 s) | PERSIST-05-tested; logging hooks already exist |
| Cross-goroutine counter reads | Locks around core fields | Existing adapter atomics + `mirrorStats` | Single-driver invariant |
| Build-tag OS split | `runtime.GOOS` branches in one function | `//go:build linux` / `!linux` files | Compile-time platform separation; `govet`/lint clean |
| Dependency enforcement | Code-review convention or `go list` script | depguard rules + `config verify` + probe | Machine-enforced, visible diff |
| Test log capture | Re-parsing formatted text | `json.Unmarshal` into a struct (existing `parseSocketBuffersLine` pattern) | Stable fields, no format coupling |
| Exit-status testing | Shell scripts | `exec.ExitError.ExitCode()` (existing `assertExitCode`) | Already in `main_test.go` |

**Key insight:** every piece of machinery this phase needs already exists in the tree. The risk is not missing libraries; it is duplicating one of these mechanisms in a second place (a second validation path, a second limiter, a second metrics renderer, a second logger) and letting the copies drift.

## Runtime State Inventory

Phase 7 is a refactor/decomposition phase plus small behavior changes; it does not rename stored data or formats. Still, the categories must be answered explicitly.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — no dump format, key, collection, or record-version layout changes. `persist.EncodeTo` gains a guard; valid dumps are byte-identical. | None |
| Live service config | Operator-visible defaults change: `max-clock-lead` 60 s → 2 m (recommended) and previously-silent non-positive values now exit 2. Existing deployments relying on a zero/negative value to mean "default" will fail fast; document in OPERATIONS and the release notes. | Docs; no data migration |
| OS-registered state | None — no service names, task descriptions, plist/unit names, or process names change. | None |
| Secrets/env vars | No env var is renamed. The five env-only names are documented in `--help`; certificate/key **contents** never enter logs. The metrics TLS triple behavior is unchanged. | Docs only |
| Build artifacts | None — module path, binary name, generated protobuf, and `go.mod` runtime graph are unchanged. `internal/config` is a new package, not a new module. | None |

**Canonical question:** after every file is updated, what runtime system still holds the old string or value? Answer: only operator command lines/environments that used a now-rejected non-positive value or depended on the old `max-clock-lead` default; no persisted state carries a Phase-7 value.

## Common Pitfalls

### Pitfall 1: The drift WARN is unreachable at the shipped defaults

**What goes wrong:** with `drift_warn == max_clock_lead == 60 s`, every version that would satisfy `lead > drift_warn` is discarded as `bad_version` by `version.Validate`; the metric and WARN never fire.
**Why it happens:** the two bounds were introduced independently (XPORT-04's future bound copied the documented drift tolerance), and Phase 3 explicitly deferred the relation to Phase 7.
**How to avoid:** validate `drift_warn < max_clock_lead` and raise the `max-clock-lead` default to 2 m (or choose another documented pair); test both the boundary and the acceptance.
**Warning signs:** a stand with a crafted +61 s record produces `bad_version`, not a drift WARN; a unit test that sets `MaxClockLead` but not `DriftWarn` sees no warning.

### Pitfall 2: Conflict counter still counts ordinary overwrites

**What goes wrong:** `len(conflicts)` from `storage.join` includes an ordinary newer-version overwrite, a stale rejection and a tie-break loss. The review's definition is only the displacement of a **pending local** version.
**Why it happens:** the store's `Conflict` type models LWW outcomes, not authorship/confirmation; the metric was fed the slice wholesale.
**How to avoid:** compute from `c.pending` (the unacked set) and the post-apply winner; test confirmed-overwrite = 0, pending-displacement = 1, tie-break pending = 1, rejected = 0.
**Warning signs:** `TestMergeStats_counts` currently writes directly to `st.store.Set` (no pending) yet expects 1; under the new rule it must expect 0 unless it goes through `NotifyLocalWrite`.

### Pitfall 3: A failed send is counted as sent

**What goes wrong:** `flush` increments `countSent` before `WriteToUDP` (and before unicast resolve); an error still leaves the counter increased.
**Why it happens:** the "sent" vocabulary was implemented at the encode boundary.
**How to avoid:** make `write` return an error; count only on a nil `WriteToUDP`; treat resolve/encode failures as not sent. Add a fake-conn error test.
**Warning signs:** `supkv_replication_datagrams_sent_total` rises while `read_errors_total` or logs show send failures; a test with a failing `WriteToUDP` still increments.

### Pitfall 4: Peak RSS masquerades as current RSS

**What goes wrong:** `ru_maxrss` is the historical maximum; it never decreases and overstates current memory after a transient.
**Why it happens:** `Getrusage` is portable and was the easiest source.
**How to avoid:** Linux `/proc/self/statm` field 2 × page size; document the darwin dev fallback; test the parser and the source selection.
**Warning signs:** RSS stays flat after a large allocation is freed; the value equals the process's lifetime peak.

### Pitfall 5: `log.Fatalf` is filtered at `SUPKV_LOG_LEVEL=error`

**What goes wrong:** the bind error exits 1 with an empty stderr at error level; the operator sees a silent death.
**Why it happens:** `slog.SetDefault` routes the `log` package through the handler at Info; `log.Fatalf` is an Info record.
**How to avoid:** replace every fatal path with `slog.Error("Startup failed"/"Serve failed", ...)` and an explicit `os.Exit` code.
**Warning signs:** a process test with `SUPKV_LOG_LEVEL=error` and an occupied port exits 1 with no ERROR record.

### Pitfall 6: Signals are armed after load/bind; a Serve error skips the final dump

**What goes wrong:** SIGTERM during a slow dump load kills the process by default disposition; a `Serve` error calls `log.Fatalf` and never runs `shutdown`/`FlushTimeout`.
**Why it happens:** `armSignals` sits after `wire`, and `serve`'s error branch returns to `main`'s fatal path.
**How to avoid:** arm before `wire`; on a Serve error, log ERROR and run `shutdown`, then choose 3 if the dump fails else 1. Test with an injectable `run`.
**Warning signs:** a SIGTERM sent during startup exits 143; a closed listener leaves an un-dumped write.

### Pitfall 7: Frozen red tests pin two metric names

**What goes wrong:** renaming `supkv_replication_unacked_records` or `supkv_replication_datagrams_retransmitted_total` (or removing the families) breaks `TestRed_R02_unackedDrain`/`TestRed_R02_burst` in the frozen red tests.
**Why it happens:** the red tests scrape `/metrics` by exact name; the red-runner treats an unexpected failure in `internal/stand` as an error.
**How to avoid:** keep those two names and families; put new counters under new names. Do not edit `internal/stand/stand_red_test.go` or the TSV.
**Warning signs:** the red-runner reports a failure for a non-red test name; the `TestRed_*` files' hashes change.

### Pitfall 8: The red-runner mutation seds target exact core/storage lines

**What goes wrong:** the periodic-sync mutant seds `if !c.nextSync.IsZero() && now.Before(c.nextSync) {` in `core.go`; the tombstone mutant seds `out[key] = rec.Version` in `storage/store.go`. If Phase 7 edits either shape, the sed no-ops and the runner errors ("sed no-op").
**Why it happens:** the mutants are source-pattern based, not AST-based.
**How to avoid:** do not touch `Core.syncListing`'s gate/return shape or the `Versions` write line; keep core edits in `handleHello`, `applyRemote`, `handleDataSync`, `evictPeers`, `listingBound` and the accessors. Run the red-runner after each core change.
**Warning signs:** `red-runner: mutant ... sed produced no change`.

### Pitfall 9: `.golangci.yml` upstream body is byte-frozen

**What goes wrong:** editing a line above the project banner invalidates the provenance requirement and the `# Origin` contract.
**Why it happens:** the file is a vendored strict config, not a project-owned config.
**How to avoid:** append only new keys under `linters.settings.depguard.rules` after the existing `sup-kv-*` rules; run `golangci-lint config verify`; diff against the fetched 2.15.0 config.
**Warning signs:** the first line or lines 1-64 change; `config verify` fails on a new key.

### Pitfall 10: A naive depguard rule breaks legitimate imports

**What goes wrong:** denying `github.com/Djarvur/sup-kv/internal` in `app`/`port` breaks `port → app/storage/version` and app-internal imports; denying all adapters in `grpcapi` breaks its `persistfile`/`app/persist` exception.
**Why it happens:** depguard matches package prefixes, and this repo's layers share the `internal/` prefix.
**How to avoid:** name adapter packages individually; use `!$test`; document the allowed exception; run a probe.
**Warning signs:** `make lint` fails on existing files immediately after adding a rule.

### Pitfall 11: `go test ./...` does not cover the nested client module

**What goes wrong:** `examples/client` is a separate module (`module example.com/sup-kv-client`), so the root `./...` pattern skips it; a change to the public API/config could go untested there.
**Why it happens:** Go module boundaries stop `./...`.
**How to avoid:** run `make client-example` (build + vet + tidy diff) as part of the phase gate; Phase 7 does not change the API, so this is a regression check.
**Warning signs:** `go test ./...` green but `make client-example` fails.

### Pitfall 12: Strict lint rejects new code shapes

**What goes wrong:** `mnd` flags magic numbers beyond 0/1; `paralleltest` requires `t.Parallel()` in every test/subtest; `noinlineerr` forbids `if err := f(); err != nil`; `sloglint` requires capitalized static messages, snake_case keys, no `msg`/`level`/`time`/`source` keys; `godoclint` caps comments at 97 columns; `gochecknoglobals` flags package vars.
**Why it happens:** `default: all` with the powerman strict config.
**How to avoid:** name constants for thresholds/expected counts; add `t.Parallel()` or a specific reasoned `//nolint:paralleltest`; assign errors on their own line; use static messages; run `make lint` per task.
**Warning signs:** lint failures only in the new files; `nolintlint` rejects an unexplained suppression.

### Pitfall 13: Log assertions race or depend on formatting

**What goes wrong:** reading a plain `bytes.Buffer` while the process writes it races under `-race`; matching formatted text breaks when the handler switches JSON/text or field order changes.
**Why it happens:** `startNode` uses a plain buffer and is only safe after exit; `startLoggedNode` uses the lock-guarded `logBuffer`.
**How to avoid:** poll with `startLoggedNode` and the lock-guarded buffer; parse JSON records into structs; assert fields/levels, not full lines; keep text-mode assertions to the existing `TestLog_formatText` format test.
**Warning signs:** `-race` reports a write/read on `bytes.Buffer`; a test passes in JSON mode but fails in text mode.

### Pitfall 14: The stand's fixed ports and known flakes

**What goes wrong:** new process tests that pick `:0` ports or use the stand window can collide with parallel packages; the known `TestScenario_dumpNotify` hang, `TestCore_keyCountCostFlat` ratio and `TestStand_killNineRestart` baseline flake can fire under load.
**Why it happens:** the stand uses the private window `[28100, 32678)` below the ephemeral range, and the full suite runs packages in parallel; the flake classes are documented and deferred to Phase 8.
**How to avoid:** use `freeUDPPort`/`freeTCPPort`/`scenarioUDPPort` helpers; hold an occupied socket instead of pick-then-bind where possible; run the affected package standalone before declaring a regression; record any flake with evidence, never weaken the bound.
**Warning signs:** `bind: address already in use` in an untouched test; a timing bound fails only in the full suite.

### Pitfall 15: The metrics `Sources` set is closed and nil-unsafe

**What goes wrong:** adding four fields to `metrics.Sources` silently panics at scrape time in any test/harness that constructs `Sources` without them (`s.Sent()` etc. on nil).
**Why it happens:** `Snapshot` calls every accessor unconditionally.
**How to avoid:** update `zeroSources` and every `Sources` literal in the same commit; grep for `metrics.Sources{`; consider a `Snapshot` nil guard only if it does not hide wiring mistakes.
**Warning signs:** `TestMetrics_families` panics with a nil function call.

### Pitfall 16: The unbound-listing class change ripples through the totality model

**What goes wrong:** the discard-conservation test sums every class; adding `UnboundListing` without updating the sum/switch makes conservation fail. Tests that expect `DuplicateNodeID` for unbound listings fail by design.
**Why it happens:** the core's discard classes are mirrored by `DiscardStats`, the adapter atomics, the metrics label list, and the negative-table model.
**How to avoid:** update all six places listed in D-04 in one commit; keep `DuplicateNodeID` for identity paths; add the label to the zero-state metrics test and OPERATIONS.
**Warning signs:** `TestDiscard_totality` sum mismatch; a metrics scrape missing the new label.

### Pitfall 17: The library value guard must precede the header write

**What goes wrong:** adding the guard inside the record loop after `enc.Encode(header{...})` writes a partial generation for an invalid config.
**Why it happens:** the existing size check is per-record.
**How to avoid:** validate `valueMaxBytes` at the top of `EncodeTo`, before any write; test `buf.Len()==0` on rejection.
**Warning signs:** a rejected dump leaves a header-only temp/generation.

### Pitfall 18: Changing `max-clock-lead`'s default breaks assertions and docs

**What goes wrong:** `TestMain_maxClockLead` asserts `(default 1m0s)`, OPERATIONS documents 60 s, and `internal/app/repl/discard_test.go` duplicates a 60 s model constant.
**Why it happens:** the default is asserted in three places.
**How to avoid:** change all three in the same commit and add the `drift_warn < max_clock_lead` relation test; do not leave a stale help/doc claim.
**Warning signs:** process help test fails; a boundary test uses the old lead.

### Pitfall 19: `flag` globals and test parallelism

**What goes wrong:** `flag.CommandLine` is process-global; tests that call `flag.Parse` or swap `slog.SetDefault` in parallel corrupt each other.
**Why it happens:** the current main uses the global flag set; tests that capture logs swap the global logger.
**How to avoid:** use a local `*flag.FlagSet` in `internal/config`; keep logger-swapping tests sequential with a reasoned `//nolint:paralleltest` (the `TestAwaitEngineStop` precedent); production uses the config package once.
**Warning signs:** a test sees another test's flags; `-race` reports `slog.SetDefault`/`log.SetOutput` races.

### Pitfall 20: `--version` must survive a broken environment

**What goes wrong:** moving env resolution into `Parse` and validating before checking `--version` makes a broken `SUPKV_*` value fatal for `--version`, regressing the existing contract.
**Why it happens:** the current order parses flags, checks `*showVersion`, then returns the collected env error.
**How to avoid:** Parse flags first, return `ShowVersion` before joining env/validation errors; test `SUPKV_LOG_LEVEL=bogus sup-kv --version` exits 0.
**Warning signs:** `TestMain_version`/precedence run with a poisoned env fails.

### Pitfall 21: `slog.SetDefault` writes to the global default logger

**What goes wrong:** unit tests that call `setupLogging` or `slog.SetDefault` and run in parallel change each other's output; a leaked handler swallows later test logs.
**Why it happens:** the handler is process-global by design.
**How to avoid:** never call `setupLogging` in parallel tests; inject a `*slog.Logger` into the rendering function where possible; restore the old default in `t.Cleanup`.
**Warning signs:** missing log lines in unrelated tests; `-race` on the default handler.

### Pitfall 22: The red-runner's evidence directories are intentionally untracked

**What goes wrong:** a plan that "cleans up" `red-evidence/` or edits the runner's expected rows invalidates the frozen contract.
**Why it happens:** the evidence directories are generated artifacts; the TSV rows are the frozen data.
**How to avoid:** run the runner, leave the evidence dirs untracked/untouched, and change only the code until the existing 7/7 rows stay green. A red-to-green flip is a TSV data change, not a runner change; Phase 7 has no expected flips.
**Warning signs:** `git status` shows changes under `.planning/phases/02-.../red-evidence` or `scripts/`.

## Code Examples

The snippets below are design sketches. Values quoted from the tree are tagged in the surrounding text; new constants/defaults (2 m, 1 ms, new metric names) are listed in the Assumptions Log.

### Config rule table + one Validate function

```go
// internal/config/validate.go — design sketch.
package config

import (
	"errors"
	"fmt"

	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/udprepl"
)

var (
	ErrPositive        = errors.New("must be positive")
	ErrRelation        = errors.New("relation violated")
	ErrDatagramCeiling = errors.New("exceeds the datagram ceiling")
	ErrPositional      = errors.New("unexpected positional arguments")
)

type rule struct {
	name  string
	check func(Config) error
}

func Validate(c Config) error {
	var errs []error
	for _, r := range rules {
		err := r.check(c)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", r.name, err))
		}
	}
	return errors.Join(errs...)
}

var rules = []rule{
	{"--node-id", func(c Config) error {
		if c.NodeID == 0 {
			return ErrMissingRequired
		}
		return nil
	}},
	{"--hello-interval", positiveDuration(func(c Config) time.Duration { return c.HelloInterval })},
	{"--peer-timeout", positiveDuration(func(c Config) time.Duration { return c.PeerTimeout })},
	{"--ack-timeout", positiveDuration(func(c Config) time.Duration { return c.AckTimeout })},
	{"--ack-timeout-max", positiveDuration(func(c Config) time.Duration { return c.AckTimeoutMax })},
	{"--retransmit-interval", func(c Config) error {
		if c.RetransmitInterval < minRetransmitInterval {
			return fmt.Errorf("%w: at least %s", ErrRetransmitFloor, minRetransmitInterval)
		}
		return nil
	}},
	{"--sync-response-rate", func(c Config) error {
		if c.SyncResponseRate <= 0 || c.SyncResponseRate > repl.MaxSyncResponseRate {
			return fmt.Errorf("%w: (0, %d]", ErrPositive, repl.MaxSyncResponseRate)
		}
		return nil
	}},
	{"--sync-chunk-bytes", func(c Config) error {
		if c.SyncChunkBytes <= 0 || c.SyncChunkBytes > repl.MaxSyncChunkBytes {
			return fmt.Errorf("%w: (0, %d]", ErrSyncChunkBounds, repl.MaxSyncChunkBytes)
		}
		return nil
	}},
	{"--peer-timeout/--hello-interval", func(c Config) error {
		if c.PeerTimeout <= c.HelloInterval {
			return fmt.Errorf("%w: peer-timeout must be greater than hello-interval", ErrRelation)
		}
		return nil
	}},
	{"--ack-timeout/--ack-timeout-max", func(c Config) error {
		if c.AckTimeout > c.AckTimeoutMax {
			return fmt.Errorf("%w: ack-timeout must not exceed ack-timeout-max", ErrRelation)
		}
		return nil
	}},
	{"--drift-warn/--max-clock-lead", func(c Config) error {
		if c.DriftWarn >= c.MaxClockLead {
			return fmt.Errorf("%w: drift-warn must be less than max-clock-lead so drift warnings are reachable", ErrRelation)
		}
		return nil
	}},
	{"--value-max-bytes", func(c Config) error {
		if c.ValueMaxBytes <= 0 {
			return fmt.Errorf("%w", ErrValueMaxBytesRange)
		}
		probe := min(c.ValueMaxBytes, udprepl.MaxDatagramBytes)
		size := udprepl.MaxRecordDatagramBytes(storage.MaxKeyBytes, probe)
		if probe == c.ValueMaxBytes && size <= udprepl.MaxDatagramBytes {
			return nil
		}
		return fmt.Errorf("%w: needs %d-byte datagrams, over %d",
			ErrDatagramCeiling, size, udprepl.MaxDatagramBytes)
	}},
}
```

The existing ceiling probe is preserved verbatim in spirit:

> `cmd/sup-kv/main.go:541-557`
> ```go
> func checkDatagramCeiling(cfg config) error {
> 	valueBytes := cfg.valueMaxBytes
> 	if valueBytes <= 0 {
> 		valueBytes = storage.ValueMaxBytes
> 	}
> 	probe := min(valueBytes, udprepl.MaxDatagramBytes)
> 	size := udprepl.MaxRecordDatagramBytes(storage.MaxKeyBytes, probe)
> 	if probe == valueBytes && size <= udprepl.MaxDatagramBytes {
> 		return nil
> 	}
> 	measured := strconv.Itoa(size)
> 	if probe < valueBytes {
> 		measured = "at least " + measured
> 	}
> 	return fmt.Errorf("%w: --value-max-bytes=%d with the %d-byte key limit needs %s-byte datagrams, over the %d-byte UDP ceiling",
> 		errDatagramCeiling, cfg.valueMaxBytes, storage.MaxKeyBytes, measured, udprepl.MaxDatagramBytes)
> }
> ```
> `[VERIFIED: cmd/sup-kv/main.go:541-557]`

### Count a datagram as sent only after the socket accepts it

```go
// internal/udprepl/repl.go — change sketch. The original is quoted in D-05.
func (a *Adapter) flush(out *port.Outbox) {
	for _, ob := range out.Drain() {
		datagram, err := encodeDatagram(a.nodeID, a.runID, ob.Msg)
		if err != nil {
			a.log(err.Error())
			continue
		}
		var addr *net.UDPAddr
		if ob.Broadcast {
			addr = a.broadcastAddr
		} else {
			addr, err = net.ResolveUDPAddr("udp4", ob.To)
			if err != nil {
				a.log(fmt.Sprintf("udprepl: resolve unicast handle %q: %v", ob.To, err))
				continue
			}
		}
		err = a.write(datagram, addr)
		if err != nil {
			continue // logged by write; never counted as sent
		}
		a.countSent(ob.Msg)
	}
}

func (a *Adapter) write(datagram []byte, addr *net.UDPAddr) error {
	_, err := a.conn.WriteToUDP(datagram, addr)
	if err != nil {
		a.log(fmt.Sprintf("udprepl: send to %s: %v", addr, err))
	}
	return err
}
```

`noinlineerr` forbids the `if err := a.write(...); err != nil` form; the sketch above already uses the required assigned form. Apply the same style to `if err := r.check(c)` in the config rule loop.

### Current RSS from `/proc/self/statm` (Linux)

```go
//go:build linux

package metrics

import (
	"os"
	"strconv"
	"strings"
)

// currentRSS returns the process's current resident set size in bytes.
func currentRSS() float64 {
	raw, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(raw))
	if len(fields) < 2 {
		return 0
	}
	pages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return float64(pages) * float64(os.Getpagesize())
}
```

The field is the second whitespace-separated value (resident pages); `/proc/self/statm` on this host reports `2113 495 466 5 0 124 0` and `getconf PAGESIZE` reports `4096`. `[VERIFIED: /proc/self/statm; getconf PAGESIZE]`

### Peer lifecycle event emission

```go
// internal/app/repl/core.go — sketch around the handleHello branches.
if !remembered {
	c.peerGen++
	c.notifyPeer(PeerEvent{
		Kind: PeerAppeared, NodeID: in.NodeID, Handle: in.Source, RunID: in.RunID,
	})
} else if known.runID != in.RunID {
	// after pushSeenRun and the row update
	c.notifyPeer(PeerEvent{
		Kind: PeerRestarted, NodeID: in.NodeID, Handle: in.Source,
		RunID: in.RunID, PreviousRunID: previous,
	})
}
```

`notifyPeer` is a nil-safe wrapper:

```go
func (c *Core) notifyPeer(ev PeerEvent) {
	if c.cfg.OnPeerEvent != nil {
		c.cfg.OnPeerEvent(ev)
	}
}
```

### Exit and shutdown orchestration

```go
// cmd/sup-kv/main.go — design sketch.
func run(cfg config.Config, rt runtime) int {
	sig := rt.installSignals()
	node, listener, err := rt.wire(cfg)
	if err != nil {
		slog.Error("Startup failed", "detail", err.Error())
		return exitStartup // 1
	}
	serveErr := rt.serve(node, listener, sig)
	if serveErr != nil {
		slog.Error("Serve failed", "detail", serveErr.Error())
	}
	shutdownErr := rt.shutdown(node, sig)
	signal.Stop(sig)
	switch {
	case shutdownErr != nil:
		return exitDumpFailed // 3
	case serveErr != nil:
		return exitStartup // 1
	default:
		return 0
	}
}
```

### Additive depguard rule

```yaml
        sup-kv-adapters-not-in-core:
          files:
            - '**/internal/app/**'
            - '**/internal/port/**'
            - '!$test'
          deny:
            - pkg: github.com/Djarvur/sup-kv/internal/udprepl
              desc: adapters are wired from cmd; the pure core/ports may not import them
            - pkg: github.com/Djarvur/sup-kv/internal/persistfile
              desc: adapters are wired from cmd; the pure core/ports may not import them
            - pkg: github.com/Djarvur/sup-kv/internal/grpcapi
              desc: adapters are wired from cmd; the pure core/ports may not import them
            - pkg: github.com/Djarvur/sup-kv/internal/metrics
              desc: adapters are wired from cmd; the pure core/ports may not import them
            - pkg: github.com/Djarvur/sup-kv/internal/stand
              desc: test infrastructure is not a production dependency
```

Append after the existing `sup-kv-public-api-no-internal` rule; the upstream body ends at the additive-key banner (`golangci.yml:65-67`). `[VERIFIED: .golangci.yml:65-95]`

### Test-side JSON log record helper

```go
// cmd/sup-kv/main_test.go — design sketch, following parseSocketBuffersLine.
func jsonRecord(output, msg string) (map[string]json.RawMessage, bool) {
	for line := range strings.SplitSeq(output, "\n") {
		var rec map[string]json.RawMessage
		if json.Unmarshal([]byte(line), &rec) != nil {
			continue
		}
		var got string
		if raw, ok := rec["msg"]; ok && json.Unmarshal(raw, &got) == nil && got == msg {
			return rec, true
		}
	}
	return nil, false
}
```

The existing helper proves the pattern:

> `cmd/sup-kv/main_test.go:1002-1024`
> ```go
> func parseSocketBuffersLine(output string) (recv, send int, ok bool) {
> 	for line := range strings.SplitSeq(output, "\n") {
> 		if !strings.Contains(line, "Socket buffers") {
> 			continue
> 		}
> 		var entry struct {
> 			Msg     string `json:"msg"`
> 			RecvBuf *int   `json:"recv_buf"`
> 			SendBuf *int   `json:"send_buf"`
> 		}
> 		if json.Unmarshal([]byte(line), &entry) != nil || entry.Msg != "Socket buffers" {
> 			continue
> 		}
> 		if entry.RecvBuf == nil || entry.SendBuf == nil {
> 			continue
> 		}
> 		return *entry.RecvBuf, *entry.SendBuf, true
> 	}
> 	return 0, 0, false
> }
> ```
> `[VERIFIED: cmd/sup-kv/main_test.go:1002-1024]`

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `log.Printf`/`log.Fatalf` for everything | `log/slog` with one handler and typed attributes | Go 1.21+ | Levels and fields filterable; `log` remains bridged, so `log.Fatalf` silently becomes Info unless replaced |
| Peak RSS from `getrusage` | Current RSS from `/proc/self/statm` on Linux | long-standing Linux interface | Honest memory gauge; darwin needs a documented fallback |
| Matching formatted log strings in tests | Parsing structured records / typed events | project trajectory (Phase 3-6) | Tests survive format changes and text-vs-JSON mode |
| Counting "conflicts" as LWW outcomes | Counting pending-local displacement | this phase (14.4) | The counter matches the Spec/review definition |
| `GOOS` branches in one function | `//go:build` files | Go convention | Compile-time separation; no unreachable code on either platform |
| Adapter imports enforced by review | depguard with probe verification | Phase 1/6 precedent | Machine-enforced boundary; visible config diff |

**Deprecated/outdated:**
- `log.Fatalf` on startup/serve paths: replaced by `slog.Error` + explicit exit code.
- `syscall.Getrusage` `ru_maxrss` as a current-RSS source: peak-only.
- `MergeStats.Conflicts += len(conflicts)` and the "ordinary overwrite is counted" comment: replaced by pending-displacement counting.
- `DuplicateNodeID` for unbound listings: replaced by the dedicated `unbound_listing` class.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `max-clock-lead` default should become 2 m, with a strict `drift_warn < max_clock_lead` validation relation | D-02, D-03 | If the owner prefers keeping 60 s, OBSV-05's WARN is unreachable; the alternative is changing the Spec's `drift_warn` default (30 s) or documenting the unreachability (fails acceptance) |
| A2 | `retransmit-interval` minimum is 1 ms | D-03 | A different floor (e.g., 10 ms) is a policy choice; 1 ms rejects the 1 ns busy loop while preserving fast test configs |
| A3 | "Local unconfirmed record" means the replication core's pending (unacked) set, not the dump-confirmation set | D-05 | If the owner means dump-unconfirmed, the counter must consult the confirmer per key and the stand scenario changes; the review-engine evidence says "the replication core knows the unacked set" |
| A4 | Peer appear = INFO, restart/evict = WARN | D-01 | Levels are a discretion; a different split is a documentation-only change |
| A5 | Darwin reports the peak as a documented dev-only fallback for `process_resident_memory_bytes` | D-05 | If a strict current value is required on darwin, the family must be omitted or a darwin libproc/sysctl probe added |
| A6 | A Serve error with a successful final dump exits 1; a final-dump failure exits 3 with precedence | D-06 | The requirement only names distinct codes; precedence is our operator contract |
| A7 | Drift warnings also apply to the max version of a `DataSync` part/listing | D-02 | Spec says "received version"; if listings are excluded, the DataSync branch stays metric-free and the doc must say so |
| A8 | New package name/location is `internal/config` with exported `Config`/`Parse`/`Validate`/`Usage` | D-03 | A different name/location is mechanical; the contract is one parse/validate point outside `main` |
| A9 | Address shape validation is limited to non-empty `host:port` parsing, no DNS resolution | D-03 | A stricter check could reject hostnames that currently work; a looser one leaves a late bind failure |
| A10 | `drift-warn=0` is rejected (no "disable" semantics) | D-03 | An operator who used 0 to disable warnings now gets exit 2; the requirement says non-positive values fail |
| A11 | The four new counter names use `supkv_replication_*_total` | D-05 | Names are a discretion; changing them is a docs/metrics test update, no frozen test pins them |
| A12 | Final-dump exit code is 3 (D-06's default) | D-06 | Research found no reason for another code; changing it is a docs/test update |
| A13 | `SentByType` moves to `internal/app/repl` with an alias in `udprepl` if tests need it | D-07 | If aliasing is dropped, two `metrics` test references must be updated |
| A14 | `grpcapi`'s `persistfile`/`*persist.Engine` imports are allowed with a written justification rather than refactored | D-07 | D-07 allows "removed or explicitly allowed"; a refactor to a consumer interface is larger and can be a Phase 9+ change |

## Open Questions (RESOLVED)

Every question below is RESOLVED; the resolution is recorded in `.planning/phases/07-observability-and-configuration/07-CONTEXT.md` ("Research resolutions") and implemented by the named plan. The `[ASSUMED]` rows above are therefore locked before execution.

1. **Should `max-clock-lead` default to 2 m?**
   - What we know: `drift_warn=60 s` is the Spec default; the future bound is inclusive; with equal defaults the drift WARN is unreachable.
   - What's unclear: the owner may prefer a different bound (e.g., 5 m) or may accept changing `drift_warn`.
   - Recommendation: 2 m (2 × `drift_warn`) and a strict relation; update OPERATIONS/tests in the same commit.
   - **Resolved:** 2 m with the strict `max_clock_lead >= drift_warn + 1 ms` rule (07-CONTEXT research resolution; plan 07-02).

2. **Does "unconfirmed" mean unacked or undumped?**
   - What we know: the review-engine fix text says "the replication core knows the unacked set"; the core has `pending`; the confirmer is in `persist`.
   - What's unclear: no owner decision explicitly defines the term.
   - Recommendation: pending/unacked; document the definition in the metric Help and OPERATIONS.
   - **Resolved:** pending/unacked — D-05's refinement (07-CONTEXT; plans 07-05/07-06).

3. **Should the drift counter count listing maxima?**
   - What we know: `Observe` already absorbs listing versions and returns a drift signal; the current code discards it.
   - What's unclear: whether a listing part is a "received version" for the Spec WARN.
   - Recommendation: yes, once per part's max version; it is cheap and makes the metric total over received versions.
   - **Resolved:** yes, once per part's maximum version — D-02's refinement (07-CONTEXT; plan 07-05).

4. **Darwin current RSS?**
   - What we know: `getrusage` is peak-only; darwin is development-only per ADR 0010.
   - What's unclear: whether a darwin-current source is worth a sysctl/libproc dependency.
   - Recommendation: Linux exact via `/proc/self/statm`; darwin documented peak fallback; no new dependency.
   - **Resolved:** Linux exact current RSS; darwin documented peak fallback — D-05's refinement (07-CONTEXT; plan 07-09).

5. **Refactor `grpcapi` off `*persist.Engine` and `persistfile`?**
   - What we know: 12 engine methods plus two sentinels; D-07 permits an explicit justified exception.
   - What's unclear: whether the owner wants the stronger boundary now.
   - Recommendation: allow with a written `.golangci.yml` justification; defer a consumer-interface refactor unless the owner asks.
   - **Resolved:** explicitly allowed with the written `.golangci.yml` justification — D-07's refinement (07-CONTEXT; plan 07-03).

6. **Where do the four counter names sit?**
   - What we know: no frozen test pins them; existing names use `supkv_replication_*` and `_total`.
   - What's unclear: whether `notifications_regressed` belongs under `supkv_replication_` or a new `supkv_local_` prefix.
   - Recommendation: `supkv_replication_*` for all four (they are replication-core counters), documented in OPERATIONS.
   - **Resolved:** `supkv_replication_*` for all four (assumption A11; plans 07-09/07-10).

7. **Should the startup log include metrics certificate/key paths?**
   - What we know: they are file paths, not secrets, and the metrics listener is opt-in.
   - What's unclear: the owner's tolerance for path disclosure in logs.
   - Recommendation: log `metrics_addr` and `metrics_tls` only; omit certificate/key paths.
   - **Resolved:** `metrics_addr` and `metrics_tls` only — D-01 (07-CONTEXT; plans 07-07/07-10).

8. **Should `serve` errors run the final dump on a listener that never served?**
   - What we know: D-06 requires it; a Serve error can occur after clients have written.
   - What's unclear: whether a startup-only Serve failure should skip the dump.
   - Recommendation: always run the bounded shutdown/final dump; a startup failure before `wire` still exits 1 without a dump.
   - **Resolved:** always run the bounded final dump; a pre-`wire` startup failure exits 1 without one — D-06 (07-CONTEXT; plan 07-04).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| Go toolchain | Build, test, race, vet | ✓ | go1.27.4 linux/amd64 | — |
| golangci-lint (via `go tool`) | `make lint`, depguard probe | ✓ | 2.15.0 | — |
| Python 3 | `scripts/red-runner` JSON parser | ✓ | 3.15.2 | — |
| `git` | Red-runner evidence, phase commits | ✓ | 2.43.0 | — |
| `bash`/`sed`/`awk` coreutils | Red-runner mutation stage | ✓ (Linux) | system | — |
| `/proc/self/statm` | Current RSS on Linux | ✓ | present; `PAGESIZE=4096` | Darwin dev fallback (peak) |
| `man`/`getconf` | Evidence only | ✓ | — | — |
| Network/external services | none | — | — | Phase is code-only |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** darwin current-RSS source (peak fallback documented; darwin is development-only per ADR 0010).

## Validation Architecture

`workflow.nyquist_validation` is `true` in `.planning/config.json`, so this section is required.

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go 1.27 `testing` + `go test -race` |
| Config file | `Makefile` (`lint`, `test`, `red-run`, `bench-smoke`, `buf-lint`, `buf-breaking`, `client-example`) |
| Quick run command | `go test -race -count=1 ./internal/config/ ./internal/metrics/ ./internal/udprepl/ ./internal/grpcapi/ ./cmd/sup-kv/` |
| Full suite command | `make lint && make test && make buf-lint && make buf-breaking && make client-example` plus `SUPKV_RED_TESTS=1 bash scripts/red-runner` |
| Process helper | `cmd/sup-kv` builds the binary once (`smokeBinary`), runs it with `exec.CommandContext`, asserts `exec.ExitError.ExitCode()`; stand uses `internal/stand` and its private port window |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command / File | File Exists? |
|--------|----------|-----------|--------------------------|--------------|
| OBSV-01 | Startup entry has version + effective config | process | `go test ./cmd/sup-kv -run TestMain_startupLog -x` (new) | ❌ Wave 0 |
| OBSV-01 | Load outcome entry for prompt/main | process/unit | `TestMain_startupLogLoadOutcome` (new) | ❌ Wave 0 |
| OBSV-01 | Gate-open entry with reason | unit | `go test ./internal/grpcapi -run TestGate_openReason -x` (new) | ❌ Wave 0 |
| OBSV-01 | Peer appear/restart/evict entries | unit | `go test ./internal/app/repl -run TestPeerEvents -x` (new) | ❌ Wave 0 |
| OBSV-01 | Bind failure at `SUPKV_LOG_LEVEL=error` is ERROR + non-zero | process | `TestMain_fatalBindLoggedAtError` (new) | ❌ Wave 0 |
| OBSV-02 | One validation function/table | unit | `go test ./internal/config -run TestValidate -x` (new) | ❌ Wave 0 |
| OBSV-02 | Exit 2 names the parameter | process | existing `TestMain_*` rows + new relation/positional rows | ⚠️ update |
| OBSV-02 | Flag parsing out of `main` | unit | `go test ./internal/config -run TestParse -x` (new) | ❌ Wave 0 |
| OBSV-02 | `--help` lists five env-only names | process | `TestMain_helpListsEnvOnly` (new) | ❌ Wave 0 |
| OBSV-03 | Conflict definition | unit + stand | `TestMergeStats_conflictPendingOnly`; adapted `TestStand_metricsByEffect` | ⚠️ rewrite |
| OBSV-03 | Failed send not counted | unit | `TestAdapter_flushCountsOnlySuccessfulSends` (new) | ❌ Wave 0 |
| OBSV-03 | Current RSS | unit (Linux) | `TestCurrentRSS_*` (new) | ❌ Wave 0 |
| OBSV-03 | Dump duration semantics | unit | `TestMetrics_definitions` (new) + existing dump stats test | ⚠️ add |
| OBSV-03 | Drift counter definition | unit | `TestDrift_receivedLeadCounts` (new) | ❌ Wave 0 |
| OBSV-03 | Four Phase-4 counters surface | unit | `TestMetrics_families`/`TestMetrics_values` + adapter mirror test | ⚠️ add |
| OBSV-04 | Usage 2 / startup 1 / final dump 3 | process | existing duplicate bind + new final-dump test | ⚠️ add |
| OBSV-04 | Serve error still dumps | unit | `TestRun_serveErrorStillShutsDown` (new) | ❌ Wave 0 |
| OBSV-04 | Signals before load/bind | unit | `TestRun_signalInstalledBeforeWire` (new) | ❌ Wave 0 |
| OBSV-04 | Final dump bounded | unit/process | existing `TestAwaitEngineStop` + final-dump test | ✅ |
| OBSV-05 | Drift WARN + metric at threshold | unit + process | `TestDrift_*`, `TestMain_driftWarn` | ❌ Wave 0 |
| OBSV-06 | Four mandated entries at levels | unit/stand/process | existing stand persist tests + `TestProtoMismatch_warns` + `TestDuplicateNodeID_warns` | ⚠️ add matrix |
| OBSV-07 | Adapter imports forbidden in app/port | lint + probe | `make lint`; temporary probe file | ❌ Wave 0 |
| OBSV-07 | `metrics`→`udprepl` removed; `grpcapi` exceptions justified | lint + probe | `make lint` + probe | ❌ Wave 0 |
| OBSV-08 | `udprepl` units split, tests unchanged | regression | `go test ./internal/udprepl/` before/after | ✅ existing |
| OBSV-08 | `main` wiring only; shutdown testable | unit + process | `TestRun_*` + existing lifecycle tests | ⚠️ add |

### Sampling Rate

- **Per task commit:** the task's own automated command, ≤ 60 s (config/metrics/unit packages).
- **Per wave merge:** `go build ./... && go test -race ./...` (or the affected package set), plus `make lint` when `.golangci.yml` changes.
- **Phase gate:** `make lint`, `make test`, `make buf-lint`, `make buf-breaking`, `make client-example`, and `SUPKV_RED_TESTS=1 bash scripts/red-runner` on the committed tree. The red-runner must be 7/7 with both mutants killed; no TSV row changes are expected.
- **Max feedback latency:** ~300 s full suite; per-task ≤ 60 s.

### Wave 0 Gaps

- [ ] `internal/config/config.go`, `validate.go`, `usage.go` — new package and table tests.
- [ ] `cmd/sup-kv/main_test.go` — JSON record helper, startup/load/bind/help/final-dump/drift/Serve-error tests.
- [ ] `internal/grpcapi/gate_test.go` (or existing `gate` tests) — `OnGateOpen` reason test.
- [ ] `internal/app/repl/events_test.go` — peer event sequence; drift WARN/counter; conflict pending-only; apply-failure limiter.
- [ ] `internal/udprepl/mirror_internal_test.go` — four counter mirrors + send-failure counting.
- [ ] `internal/metrics/rss_linux_test.go`, `metrics_definitions_test.go` — RSS parser/source and Help definitions.
- [ ] `.golangci.yml` probe runs (temporary files, deleted) — no committed test file.
- [ ] `docs/OPERATIONS.md` + `cmd/sup-kv/operations_test.go` anchors — exit code 3, metrics list, drift runbook, `max-clock-lead` 2 m.

### Red-Runner Impact

- The runner's packages are `./internal/grpcapi/`, `./internal/stand/`, `./internal/persistfile/`; it runs **all** tests in them and treats any unexpected failure as an error.
- The frozen tests assert only two metric names (`supkv_replication_unacked_records`, `supkv_replication_datagrams_retransmitted_total`); keep both. They do **not** assert log text or non-zero exit codes: `stand_red_test.go` scrapes `/metrics` and checks values/absence, and `persistfile/red_value_limit_test.go` is engine-level and asserts store/dump behavior only. They do, however, call `signalAndWaitExit`, which requires a clean SIGTERM shutdown to exit **0** — so the zero path must stay zero. New lifecycle lines and the 1/2/3 code changes therefore cannot break them by text, but a metric rename/removal or a non-zero clean shutdown can.
- `TestStand_metricsByEffect` is in `internal/stand` and currently asserts `conflicts == 1` after a scenario where A's local write was already Acked. Under the new pending-only definition it must be rebuilt (e.g., A writes while B is stopped, then a crafted valid DataReplication from an impostor with a higher version displaces the pending record). This is the one non-frozen stand test that must change in the same commit as the conflict semantics, or the red gate fails.
- The periodic-sync mutation sed targets `core.go:2383` and the tombstone sed targets `storage/store.go:248,269`; do not edit those line shapes. If a code move changes them, the runner reports a sed no-op — treat that as a blocker, not a reason to edit the runner.
- The runner's evidence directories are untracked; do not clean or commit them.

## Security Domain

`workflow.security_enforcement` is `true` and ASVS level 1.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | The wire and gRPC API ship no authentication (documented trust model); Phase 7 does not change it |
| V3 Session Management | no | No sessions; Watch streams are bounded by the existing gRPC shutdown contract |
| V4 Access Control | partial | The metrics/debug endpoint remains opt-in and mTLS-capable; Phase 7 adds no new endpoint and must not log certificate/key contents |
| V5 Input Validation | yes | The one `config.Validate` table: all ranges/relations, datagram ceiling, value-limit guard, positional args, log env parsing. Reject before bind; exit 2 naming the parameter |
| V6 Cryptography | no new crypto | `metrics.NewTLSConfig` is unchanged; do not hand-roll TLS or checksums. The value guard uses the existing dump format ceiling |
| V7 Logging (ASVS 5.0) | yes (operational) | Lifecycle entries are structured, secrets-free, and rate-limited per peer/class; no key names in discard/apply-failure details |

### Known Threat Patterns for This Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Log flood via oversized/apply-failing datagrams | Denial of Service | Per-(node_id, class) limiter; drop the key from the apply-failure message; counters remain per datagram |
| Log injection via key/handle/run-id strings | Tampering / Repudiation | Structured `slog` values (`%q` for handles), never string concatenation into the message; no key names in discard details |
| Secret leakage in startup config | Information Disclosure | Log values/paths, never certificate/key bytes; omit certificate/key paths or log only the TLS boolean |
| Config-driven memory/CPU exhaustion (`sync-chunk-bytes` overflow, huge `value-max-bytes`, `retransmit-interval=1ns`) | Denial of Service | Validate before bind; library-level `EncodeTo` guard; datagram ceiling probe; retransmit floor |
| Metrics cardinality blow-up | Denial of Service / Information Disclosure | Fixed label enums only; no keys, node ids or values as labels |
| Silent bind failure at error level | Repudiation / operational blindness | ERROR record + non-zero exit; no `log.Fatalf` on lifecycle paths |
| `--bind-addr` non-wildcard | Spoofing / availability | Existing WARN unchanged; do not turn it into a startup failure in this phase |
| Unbound listing poisons the gate/retained keys | Tampering | Existing CR-01 binding; New-2 makes the class honest and keeps the same rate limiter |
| Dependency-direction erosion (`metrics`→`udprepl`, `grpcapi`→adapter types) | Tampering (architecture) | Additive depguard rules + probe; visible `.golangci.yml` diff |
| Test-only boundary imports | Tampering | `!$test` exemptions are deliberate; production files remain covered |

## Sources

### Primary (HIGH confidence)

- `cmd/sup-kv/main.go` (whole file, 1 104 lines) — flags/env, defaults, validation, wiring, logging, shutdown, exit codes.
- `internal/app/repl/core.go` (relevant ranges) — `Config`, `MergeStats`, `applyRemote`, `handleHello`, `evictPeers`, `listingBound`, four accessors, discard classes.
- `internal/app/clock/clock.go:112-180` — `Update`, `driftBetween`, saturation.
- `internal/app/storage/store.go:344-441` — `MergeChanged`, `Observe`, `join`.
- `internal/app/persist/{doc.go,codec.go,scheduler.go,engine.go}` — value guard precondition, enqueue/encode, dump failure cadence, `LoadStats`.
- `internal/udprepl/repl.go` (whole file, 1 417 lines) — adapter, socket, loop, mirror, codec.
- `internal/metrics/metrics.go` (whole file, 562 lines) — `Sources`, families, runtime probes, RSS.
- `internal/grpcapi/{gate.go,limits.go,server.go}` — gate resolver, transport limits, API dependencies.
- `internal/port/dump.go`, `internal/app/version/version.go:108-118` — load outcomes/generations, future bound.
- `.golangci.yml` — vendored provenance header, project depguard rules.
- `docs/OPERATIONS.md:40-310` — monitoring list, scale envelope, load outcomes, drift runbook, lifecycle/exit codes.
- `docs/reviews/2026-11-19-v1.0/REVIEW.md` §9 O-01…O-05 and §10 Q-05/Q-07; §7 S-06/S-07; §14 rows 4/5/6/8.
- `docs/reviews/2026-11-19-v1.0/evidence/review-engine.md:736` — conflict pending-local definition.
- `docs/adr/0004-no-conflict-journal.md`, `0008-metrics-off-by-default.md`.
- Phase routing ledgers: `03-REVIEW-DISPOSITION.md` WR-02/IN-01; `04-REVIEW-DISPOSITION.md` IN-03/New-2; `05-REVIEW-DISPOSITION.md` IN-04; `06-REVIEW-DISPOSITION.md` (no Phase-7 row).
- `go list ./internal/...` — current import graph.
- `scripts/red-expectations.tsv`, `scripts/red-runner`, `internal/stand/stand_red_test.go` — frozen red contract and mutation targets.
- Go 1.27.4 `$GOROOT/src/log/slog/logger.go:62-74` — `SetDefault` log bridge.
- `man 2 getrusage` — `ru_maxrss` is maximum RSS.
- `/proc/self/statm`, `getconf PAGESIZE` — current resident pages and page size.

### Secondary (MEDIUM confidence)

- `.planning/phases/03-transport/03-RESEARCH.md:329,602,733` — the `max_clock_lead`/`drift_warn` assumption and the deferred relation.
- `.planning/phases/05-persistence/05-12-SUMMARY.md` — known flakes and the frozen test/runner contract.
- `.planning/phases/06-api-and-semantics/06-13-SUMMARY.md` — stand flake and port-contention observations.
- `.planning/phases/06-api-and-semantics/06-VERIFICATION.md` — depguard probe precedent.

### Tertiary (LOW confidence)

- None. No claim in this research rests on an unverified web source or training memory.

## Metadata

**Confidence breakdown:**
- Current-state map: HIGH — every quote read from the named file this session.
- Standard stack / dependencies: HIGH — no new packages; toolchain versions probed.
- Design recommendations: MEDIUM — grounded in the code/review, but the defaults and levels in the Assumptions Log need owner confirmation.
- Pitfalls: HIGH for the code-evidenced traps (drift unreachable, sent-before-write, RSS peak, signal order, frozen tests/mutants); MEDIUM for the lint/test-process traps (behavior depends on the exact golangci-lint run).

**Research date:** 2026-11-23
**Valid until:** 2027-01-22 (60 days; the codebase moves per phase, but the frozen contracts and Go 1.27 toolchain are stable)
