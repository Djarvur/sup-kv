# Phase 7: Observability and Configuration - Pattern Map

**Mapped:** 2026-11-23
**Files classified:** 43 (production, test, config, docs)
**Analogs found:** 40 / 43 close matches; 3 have no close analog (all named in the final table). Every path named below is git-tracked source.

Every analog path in this file was verified with `git ls-files -- <path>` and is tracked source.
No gitignored mirror path is cited.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/config/config.go` (NEW) | config | transform (parse/resolve) | `cmd/sup-kv/main.go:99-430` | extraction-target |
| `internal/config/validate.go` (NEW) | config/validation | transform (rule table) | `cmd/sup-kv/main.go:432-557`; `internal/grpcapi/limits.go:87-106` | extraction-target |
| `internal/config/usage.go` (NEW) | config/usage | transform (render) | `cmd/sup-kv/main.go:341-402` | role-match |
| `internal/config/config_test.go` (NEW) | test | table | `internal/app/repl/config_internal_test.go:32-55` | role-match |
| `internal/config/validate_test.go` (NEW) | test | table | `cmd/sup-kv/main_test.go:584-684` | role-match |
| `internal/config/usage_test.go` (NEW) | test | contract | `cmd/sup-kv/main_test.go:438-449` | role-match |
| `internal/app/repl/events.go` (NEW) | model/event seam | event-driven | `internal/app/persist/engine.go:44-58`; `internal/grpcapi/gate.go:47-111` | role-match |
| `internal/app/repl/events_test.go` (NEW) | test | event-driven | `internal/app/repl/warn_test.go:35-100`; `internal/app/repl/merge_stats_test.go:28-85` | role-match |
| `internal/app/repl/sentbytype.go` (NEW, name at discretion) | model/value | transform | `internal/udprepl/repl.go:974-995` | exact-move |
| `internal/app/repl/core.go` | service/core | event-driven + request-response | itself + `internal/app/storage/store.go:402-441` | self |
| `internal/app/clock/clock.go` | utility/clock | transform | itself (`Update`/`driftBetween`) | self |
| `internal/app/persist/codec.go` | utility/codec | file-I/O + transform | itself (`EncodeTo`) | self |
| `internal/app/persist/doc.go` | docs | — | itself (IN-04 precondition note) | self |
| `internal/app/persist/scheduler.go` | service/scheduler | batch + event-driven | itself (`runDump`/`handleDumpFailure`) | self |
| `internal/metrics/metrics.go` | provider/adapter | request-response (scrape) | itself (`Sources`/families) | self |
| `internal/metrics/rss_linux.go` (NEW) | utility/platform | file-I/O | none (build-tag precedent: `internal/stand/stand_nondarwin_test.go:1`) | none |
| `internal/metrics/rss_other.go` (NEW) | utility/platform | syscall | `internal/metrics/metrics.go:493-509` (`processUsage`) | partial |
| `internal/metrics/rss_linux_test.go` (NEW) | test | file-I/O | `internal/metrics/metrics_test.go:27-35` | role-match |
| `internal/metrics/metrics_definitions_test.go` (NEW) | test | contract | `internal/metrics/metrics_test.go:186-301` | role-match |
| `internal/metrics/metrics_test.go` | test | table | itself (`families`, `zeroSources`) | self |
| `internal/udprepl/socket.go` (NEW) | adapter/socket | socket I/O | `internal/udprepl/repl.go:217-255,464-584` | extraction-target |
| `internal/udprepl/codec.go` (NEW) | adapter/codec | transform | `internal/udprepl/repl.go:1251-1417` | extraction-target |
| `internal/udprepl/loop.go` (NEW) | adapter/event loop | event-driven | `internal/udprepl/repl.go:586-625,1066-1156,1217-1245` | extraction-target |
| `internal/udprepl/mirror.go` (NEW) | adapter/mirror | pub-sub (atomic snapshot) | `internal/udprepl/repl.go:874-1038,1158-1215` | extraction-target |
| `internal/udprepl/adapter.go` (NEW, optional) | adapter/composition | — | `internal/udprepl/repl.go:257-439` | extraction-target |
| `internal/udprepl/repl.go` | adapter | — | being split; no behavior change | self |
| `internal/udprepl/mirror_internal_test.go` (NEW) | test | snapshot | `internal/udprepl/repl_internal_test.go:448-507` | role-match |
| `internal/udprepl/decode_negative_test.go` | test | table/mutation model | itself (class-string switch) | self |
| `internal/udprepl/scenario_test.go` | test | scenario | itself | self |
| `cmd/sup-kv/main.go` | wiring/orchestrator | request-response + lifecycle | itself (`main`/`wire`/`shutdown`) | self |
| `cmd/sup-kv/main_test.go` | test | process | itself (helpers, exit assertions) | self |
| `cmd/sup-kv/operations_test.go` | test | doc anchor | itself + `cmd/sup-kv/api_docs_test.go:133-148` | role-match |
| `internal/grpcapi/gate.go` | adapter/gate | event-driven | itself (`ServerOptions`, `RunGate`) | self |
| `internal/grpcapi/gate_test.go` | test | event-driven | itself (`TestGate_opensOnSync`/`opensOnTimeout`) | self |
| `.golangci.yml` | config/lint | — | itself (project depguard rules, lines 65-95) | self |
| `docs/OPERATIONS.md` | docs | — | itself + anchor contract | self |
| `docs/SPEC-DEVIATIONS.md` | docs/registry | — | itself (table rows + status vocabulary) | self |
| `AGENTS.md` | docs/config | — | itself (`## Layout`, `### Project deviations`) | self |
| `internal/app/repl/merge_stats_test.go` | test | unit | itself (rewrite conflict expectation) | self |
| `internal/app/repl/warn_test.go` | test | unit | itself (limiter table) | self |
| `internal/app/repl/discard_test.go` | test | model/totality | itself (class-string switches) | self |
| `internal/app/repl/listedkeys_internal_test.go` | test | unit | itself (unbound-listing expectations) | self |
| `internal/stand/stand_metrics_test.go` | test | end-to-end | itself (`TestStand_metricsByEffect`) | self |

## Pattern Assignments

### `internal/config/config.go` (config, transform/parse)

**Analog:** `cmd/sup-kv/main.go:99-430` — the parsing code is being *moved*, so copy this shape rather than inventing a new one.

**Env constant and default tables** (`main.go:99-137,141-171`):

```go
const (
	envNodeID             = "SUPKV_NODE_ID"
	envBindAddr           = "SUPKV_BIND_ADDR"
	// ... one const per knob ...
	envLogLevel  = "SUPKV_LOG_LEVEL"
	envLogFormat = "SUPKV_LOG_FORMAT"
	// The metrics/debug endpoint knobs: the listen address has a flag twin
	// with NO default — unset means no listener at all; the certificate
	// triple is env-only and all-or-nothing.
	envMetricsAddr     = "SUPKV_METRICS_ADDR"
	envMetricsCert     = "SUPKV_METRICS_CERT"
	envMetricsKey      = "SUPKV_METRICS_KEY"
	envMetricsClientCA = "SUPKV_METRICS_CLIENT_CA"
)
```

**Config struct** (`main.go:179-208`) becomes the exported `config.Config`; keep field order and names so
the main-side mapping is mechanical. Research's API sketch adds `ShowVersion` and `LogLevel slog.Level`.

**Env-as-flag-default closures** (`main.go:324-340`) — preserve the collected-error contract:

```go
func registerConfig(cfg *config, nodeID *uint64, showVersion *bool) error {
	var errs []error
	fail := func(env string, err error) {
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", env, err))
		}
	}
	envDur := func(env string, def time.Duration) time.Duration {
		v, err := envDuration(env, def)
		fail(env, err)
		return v
	}
	envNum := func(env string, def int) int {
		v, err := envInt(env, def)
		fail(env, err)
		return v
	}
	flag.BoolVar(showVersion, "version", false, "print version, commit and Go runtime version, then exit")
	// ... flag registrations with envDur/envNum as defaults ...
	return errors.Join(errs...)
}
```

**Env value helpers** (`main.go:1022-1104`) — copy verbatim but accept a `getenv func(string) string`
instead of reading `os.LookupEnv` directly (research's `Parse(args, getenv)` sketch):

```go
func envOrDefault(env, def string) string {
	if v, ok := os.LookupEnv(env); ok && v != "" {
		return v
	}
	return def
}
func envDuration(env string, def time.Duration) (time.Duration, error) {
	raw := envOrDefault(env, "")
	if raw == "" {
		return def, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return def, fmt.Errorf("must be a duration: %w", err)
	}
	return d, nil
}
func envSlogLevel(env string, def slog.Level) (slog.Level, error) {
	// lvl.UnmarshalText([]byte(raw)) with the "debug, info, warn or error" wrap
}
```

**Parse mechanics (from research, no exact in-tree analog):**
- Use `flag.NewFlagSet("sup-kv", flag.ContinueOnError)` + `fs.SetOutput(io.Discard)`; `main` owns output.
- `--help`/`-h` returns `ErrHelp`; `main` calls `config.Usage(os.Stdout)` and exits 0.
- Parse flags first, return `ShowVersion` **before** joining env/validation errors (Pitfall 20).
- Reject `len(fs.Args()) > 0` naming every positional argument.
- Keep these existing error substrings (process tests pin them):
  `datagram ceiling`, `value-max-bytes must be positive`, `startup gate must be positive`,
  `sync-chunk-bytes exceeds the datagram ceiling`, `node-id`.
- Do **not** place this package under `internal/app` — it imports `flag`, `io`, `os`/`os`-reading
  helpers and `udprepl`; the `sup-kv-app-no-io` rule would reject it.

---

### `internal/config/validate.go` (config/validation, transform)

**Analog:** `cmd/sup-kv/main.go:432-557` (the six-check sequence) plus `internal/grpcapi/limits.go:87-106`
(the project's pure-derivation style).

**Current sequence to fold into one table** (`main.go:434-461`):

```go
func finalizeConfig(cfg *config, nodeID uint64, envErr error) error {
	if envErr != nil {
		return envErr
	}
	if nodeID > math.MaxUint16 {
		return fmt.Errorf("--node-id %d: %w", nodeID, errNodeIDRange)
	}
	cfg.nodeID = uint16(nodeID)
	if missing := missingRequired(*cfg); len(missing) > 0 {
		return fmt.Errorf("%w: %s", errMissingRequired, strings.Join(missing, ", "))
	}
	err := checkMetricsCerts(*cfg)
	// ... checkStartupGates, checkValueMaxBytes, checkDatagramCeiling ...
	return checkSyncChunkBytes(*cfg)
}
```

**Error-wrapping style** (`main.go:464-477,541-557`) — every failure names the parameter and wraps a
sentinel with `%w`:

```go
func checkSyncChunkBytes(cfg config) error {
	if cfg.syncChunkBytes > repl.MaxSyncChunkBytes {
		return fmt.Errorf("%w: --sync-chunk-bytes=%d (at most %d, the largest IPv4 UDP payload)",
			errSyncChunkBytes, cfg.syncChunkBytes, repl.MaxSyncChunkBytes)
	}
	return nil
}
// checkDatagramCeiling keeps the probe shape:
// probe := min(valueBytes, udprepl.MaxDatagramBytes)
// size := udprepl.MaxRecordDatagramBytes(storage.MaxKeyBytes, probe)
```

**Pure-derivation style to copy for the rule table** (`limits.go:95-106`):

```go
func LimitsFor(valueMaxBytes int) TransportLimits {
	if valueMaxBytes <= 0 {
		valueMaxBytes = storage.ValueMaxBytes
	}
	return TransportLimits{ /* every value derived from the input */ }
}
```

**Rule table (research design; one function + one slice):**

```go
type rule struct {
	name  string
	check func(Config) error
}

func Validate(c Config) error {
	var errs []error
	for _, r := range rules {
		err := r.check(c)        // noinlineerr: assign, don't inline
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", r.name, err))
		}
	}
	return errors.Join(errs...)  // operator sees every bad knob in one pass
}
```

Rules to include (research D-03 table): required knobs, node-id range, positivity of all durations +
`sync-response-rate`, retransmit floor (`>= 1ms`), sync-response cap (`<= repl.MaxSyncResponseRate`,
which must be **exported** from `internal/app/repl/core.go:119`), sync-chunk bounds `(0, repl.MaxSyncChunkBytes]`,
value-max positivity + datagram ceiling, `peer_timeout > hello_interval`, `ack_timeout <= ack_timeout_max`,
`max_clock_lead >= drift_warn + 1ms` (millisecond-resolution strict), metrics cert triple, log env parse,
positional args, optional address shape.

Keep the core's `Config.normalized()` (`core.go:303-344`) untouched as defense in depth — it is not the contract.

---

### `internal/config/usage.go` (config/usage, transform)

**Analog:** `cmd/sup-kv/main.go:341-402` — the flag descriptions already carry `(env SUPKV_X)` and default
text. `fs.PrintDefaults()` reproduces them.

Pattern: render flag table first (via `fs.PrintDefaults()`), then append the env-only block. The five names
are the test contract:

```text
Environment-only (no flag):
  SUPKV_LOG_LEVEL       debug, info, warn or error (default info)
  SUPKV_LOG_FORMAT      json or text (default json)
  SUPKV_METRICS_CERT    server certificate path (all three or none)
  SUPKV_METRICS_KEY     server key path (all three or none)
  SUPKV_METRICS_CLIENT_CA  client CA path (all three or none)
```

The metrics **address** has a flag twin and is therefore not env-only (`main.go:124-137`).

---

### `internal/config/*_test.go` (tests)

**Analog A — table-test shape:** `internal/app/repl/config_internal_test.go:32-55`

```go
rows := []struct {
	name string
	in   int
	want int
}{
	{"above the ceiling clamps", MaxSyncChunkBytes + 1, MaxSyncChunkBytes},
	{"exactly the ceiling passes", MaxSyncChunkBytes, MaxSyncChunkBytes},
	{"zero keeps the design default", 0, defaultSyncChunkBytes},
}
for _, row := range rows {
	t.Run(row.name, func(t *testing.T) {
		t.Parallel()
		// ... assert exactly one normalized value ...
	})
}
```

**Analog B — process-level exit-2 rows:** `cmd/sup-kv/main_test.go:584-616` and `659-684`

```go
out, err := cmd.CombinedOutput()
assertExitCode(t, err, exitUsageCode)
for _, want := range []string{"value-max-bytes must be positive", tc.flag} {
	if !strings.Contains(string(out), want) {
		t.Fatalf("... does not contain %q:\n%s", want, out)
	}
}
```

Planner notes:
- One subtest per rule, each with an otherwise-valid Config; pin the rule slice length and assert every
  rule name appears in at least one subtest (meta-test) so a rule cannot be added untested.
- Use named constants for expected numbers (`mnd` rejects magic numbers beyond 0/1).
- `paralleltest` is enabled; every test and subtest needs `t.Parallel()`.
- Keep the existing process rows and add: relation violations (`peer-timeout=1s --hello-interval=1s`),
  `retransmit-interval=1ns`, `sync-response-rate=1000000001`, positional arg,
  `drift-warn=2m --max-clock-lead=2m`.

---

### `cmd/sup-kv/main.go` (wiring, lifecycle orchestration)

**Analog:** itself — the edits are reorganizations of the existing exit/order functions.

**Exit-code table** (`main.go:70-77`) currently collapses startup and final-dump failure into 1:

```go
const (
	exitDumpFailed = 1
	exitUsage      = 2
)
```

Target (D-06): `exitStartup = 1`, `exitUsage = 2`, `exitDumpFailed = 3`; precedence dump-failure(3) >
Serve-error(1). `--help`/`--version` exit 0.

**Current ordering defect** (`main.go:255-276`): `wire` (dump load + UDP bind) runs before `armSignals`;
a Serve error takes `log.Fatalf` and never runs `shutdown`:

```go
node, listener, err := wire(cfg)
if err != nil {
	log.Fatalf("sup-kv: %v", err)
}
sig := armSignals()
err = serve(node, listener, sig)
if err != nil {
	log.Fatalf("sup-kv: grpc serve exited before the shutdown signal: %v", err)
}
err = shutdown(node, sig)
```

Target shape (research D-06 sketch; production functions stay as they are):

```go
func run(cfg config.Config, rt runtime) int {
	sig := rt.installSignals()          // BEFORE wire: load/bind are covered
	node, listener, err := rt.wire(cfg)
	if err != nil {
		slog.Error("Startup failed", "detail", err.Error())
		return exitStartup
	}
	serveErr := rt.serve(node, listener, sig)
	if serveErr != nil {
		slog.Error("Serve failed", "detail", serveErr.Error())
	}
	shutdownErr := rt.shutdown(node, sig)
	signal.Stop(sig)
	switch {
	case shutdownErr != nil:
		return exitDumpFailed
	case serveErr != nil:
		return exitStartup
	default:
		return 0
	}
}
```

`runtime{installSignals, wire, serve, shutdown}` is the injectable seam; production wires the existing
`armSignals`, `wire`, `serve`, `shutdown` (`main.go:881-978`). `main()` becomes parse → help/version →
`setupLogging` → `os.Exit(run(...))`.

**Startup lifecycle record:** emit after `setupLogging` (`main.go:285-292`) and before `wire`. Use
capitalized static `msg`, snake_case attrs, secrets-free values (no cert/key contents).
The existing `warnOnNonWildcardBind` (`main.go:303-315`) shows the static-msg + attr style:

```go
slog.Warn("Bind address is not a wildcard: broadcast datagrams are not received",
	"host", host,
	"detail", "broadcast replication is addressed to the segment broadcast address, ...")
```

**Load-outcome record:** extend the existing `logLoadStats` (`main.go:618-632`) and its call in
`startEngine` (`main.go:595-596`) with `outcome`, `generation`, `records`, `discarded`
(`persist.LoadStats` fields: `engine.go:157-165`). Keep the one-WARN/one-ERROR rule; a clean load becomes
one INFO record, never a second line.

**Dump failure:** keep `OnDumpFailure` (`main.go:586-594`) as-is — the ladder already bounds it
(`persist/scheduler.go:251-277`).

**Fatal/wire/metrics binds:** replace `log.Fatalf` with `slog.Error("Startup failed", "stage", ..., "detail", ...)`
so the record survives `SUPKV_LOG_LEVEL=error` (the `slog.SetDefault` bridge makes `log.Fatalf` an INFO
record, research O-01).

**Metrics sources:** extend the literal at `main.go:798-813` with the four new accessors; `Snapshot` calls
every accessor unconditionally, so nil fields panic at scrape time.

**Gate/peer/drift rendering:** wire `grpcapi.ServerOptions.OnGateOpen`, `repl.Config.OnPeerEvent`,
`repl.Config.OnDrift` to defined functions in this file (same file-local style as `logLoadStats`).

**Process-test helpers to reuse** (`cmd/sup-kv/main_test.go`): `smokeBinary` 109-123, `freeUDPPort`
137-148, `startLoggedNode` 219-224, `signalAndWaitExit` 267-288, `assertExitCode` 290-307,
`parseSocketBuffersLine` 1002-1024. For a new final-dump exit-3 test, `chmod 0500` the dump directory and
restore it in `t.Cleanup` (research D-06).

---

### `internal/app/repl/events.go` (typed lifecycle events)

**Analog A — typed, nil-valid callback Config:** `internal/app/persist/engine.go:44-58`

```go
// OnDumpFailure receives every failed dump attempt exactly once with the
// underlying error and the adapter's failure class (D-05, PERSIST-05):
// the binary wires it to one ERROR log line and must not block. Nil is
// valid — the counters and the retry ladder still apply.
OnDumpFailure func(err error, class port.FailureClass)
```

**Analog B — exported options struct with per-field docs:** `internal/grpcapi/gate.go:47-111`
(`ServerOptions`; every seam is nil-valid and explains its production injector).

**Analog C — the existing string callback being extended:** `core.go:288-297`

```go
// OnDiscard is invoked at every HandleDatagram discard site with the
// class token (DiscardBadCRC, ...) and a short metadata detail ...
// the class is a first-class value, never a message prefix to parse.
OnDiscard func(class, detail string)
```

**Target types (research D-01):** `PeerEventKind` (`PeerAppeared`/`PeerRestarted`/`PeerEvicted`),
`PeerEvent{Kind, NodeID, Handle, RunID, PreviousRunID, LastSeenAge}`, `DriftEvent{NodeID, Source, Lead}`,
and Config fields `OnPeerEvent func(PeerEvent)`, `OnDrift func(DriftEvent)`.

Nil-safe dispatch wrapper — copy the `handleDumpFailure` guard (`scheduler.go:274-277`):

```go
if s.cfg.OnDumpFailure != nil {
	s.cfg.OnDumpFailure(err, class)
}
```

Emission rules (research D-01): appear only on `!remembered`; restart only on an unseen run (after
`pushSeenRun`); evicted with `last_seen_age`; no event for repeat Hello/alternation/own echo.
`OnDrift == nil` falls back to the existing `cfg.Log` so old harnesses keep compiling.

---

### `internal/app/repl/core.go` (core semantics)

**Analog:** itself. The exact sites to change:

**1. Conflict counter — stop counting `len(conflicts)`** (`core.go:1525-1535`):

```go
conflicts, drift, err := c.store.ApplyRemote(msg.Key, msg.Record)
if err != nil {
	c.cfg.Log(fmt.Sprintf("sup-kv: applying replicated record for key %q from handle %q: %v", msg.Key, source, err))
	return
}
c.mergeStats.Conflicts += uint64(len(conflicts))
if drift != 0 {
	c.mergeStats.DriftWarnings++
}
```

`storage.join` returns a `Conflict` for *every* LWW outcome (`internal/app/storage/store.go:402-441`),
so this is the defect. Reuse the existing post-apply verdict probe at `core.go:1552-1558` and walk
`c.pending` (the unacked set) instead; the supersession loop immediately below is the walk shape:

```go
// core.go:1543-1548 — the existing pending walk to model the count on.
for pk := range c.pending {
	if pk.key == msg.Key && msg.Record.Version.Compare(pk.version) > 0 {
		out.Superseded(pk.key, pk.version)
		c.removePending(pk)
	}
}
```

Count 1 per displaced pending entry; extend the equality case so an equal-version tie-break that displaces
a pending local write counts, and an equal-and-incoming-lost does not. Do not count a rejected incoming
record, an identical redelivery, or a displaced record that is not in `pending`.
`pendingKey`/`pendingEntry` are at `core.go:641-677`; the install path is `trackLocalWrite`
(`core.go:2243-2296`).

**2. Drift WARN + counter** (`core.go:1533-1535`): change `drift != 0` to `drift > 0` (received lead only),
then emit `OnDrift` through the shared limiter:

```go
if drift > 0 {
	c.mergeStats.DriftWarnings++
	c.emitDrift(now, in.NodeID, in.Source, drift)
}
```

`handleDataSync` currently discards `Observe`'s return (`core.go:1682-1685`); capture it and apply the
same `> 0` count/event for the part's max entry version.

**3. Apply-failure WARN limiter** (`core.go:1526-1528`): replace the unrated, key-naming `c.cfg.Log(...)`
with the shared limiter and drop the key from the message:

```go
c.warnRateLimited(now, in.NodeID, warnClassApplyFailure, fmt.Sprintf(
	"sup-kv: applying replicated record from handle %q: %v", source, err))
```

Limiter implementation to reuse verbatim (`core.go:1219-1246`):

```go
func (c *Core) warnRateLimited(now time.Time, nodeID uint16, class, msg string) {
	key := scanDiscardKey{nodeID: nodeID, class: class}
	last, warned := c.discardWarnAt[key]
	if warned && now.Sub(last) < discardWarnInterval {
		return
	}
	c.discardWarnAt[key] = now
	c.cfg.Log(msg)
}
```

`discardWarnInterval = 10 * time.Second` (`core.go:107-110`); `scanDiscardKey` at `core.go:599-606`.

**4. Peer events** — `handleHello` (`core.go:1352-1427`) and `evictPeers` (`core.go:2749-2790`):
emit after the row update in the `!remembered` branch, after `pushSeenRun(known.runID)` in the restart
branch (both ids available), and inside the `expired` loop with `now.Sub(lastHello)`.

**5. Unbound-listing discard class (New-2)** — `listingBound` (`core.go:1709-1721`) currently increments
`DuplicateNodeID` and warns as `duplicate_node_id`:

```go
c.discards.DuplicateNodeID++
c.cfg.OnDiscard(DiscardDuplicateNodeID, fmt.Sprintf(
	"node_id %d listing from unbound run at handle %q", in.NodeID, in.Source))
```

Add `DiscardUnboundListing = "unbound_listing"`, a `DiscardStats.UnboundListing` field
(`core.go:410-434`), route this site through it with the same limiter, and drop "configuration error"
from this message (an unbound listing can legitimately overtake its Hello). Keep
`DuplicateNodeID` for the two identity paths and the alternation path.

**6. Foreign proto_version second branch (S-FAIL-17)** — `core.go:1036-1040` counts and fires `OnDiscard`
but never warns. Make it identical to the first branch (`core.go:1026-1034`), which calls
`c.warnDiscard(now, in.NodeID, DiscardBadProtoVersion)`.

**7. Four accessors already exist** — no core changes needed for the metric surface:
`NotificationsRegressed` 3008-3016, `SyncStreamDropped` 3063-3068, `ListedKeysEvicted` 3070-3076,
`ListingDropped` 3163-3171.

**8. MergeStats doc rewrite** (`core.go:436-446`): the current text says an LWW loser is "counted";
replace it with the pending-displacement definition and the received-lead drift definition.

**Avoid the frozen mutation shapes:** do not edit `Core.syncListing`'s `nextSync` gate/return shape
(`core.go:2383`) or `storage/store.go:248,269`; the red-runner seds target those exact lines.

---

### `internal/app/clock/clock.go` (clock drift signal)

**Analog:** itself (`clock.go:112-149,151-176`).

Current `Update` compares the *local* logical time against wall time before absorbing the remote reading:

```go
func (c *Clock) Update(remoteL int64, remoteC uint16) (l int64, counter uint16, drift time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	remoteC &= counterMask
	ptMs := c.now().UnixMilli()
	drift = driftBetween(c.l, ptMs, c.driftWarn) // <-- target: remoteL
	newL := max(c.l, remoteL, ptMs)
	...
}
```

Change only the signal operand to `remoteL` (received lead); keep `driftBetween` signed and
saturation-safe (`clock.go:151-176`) and keep `Tick`'s diagnostic local-vs-wall signal.

**Test analog:** `internal/app/clock/clock_test.go:500-586` (`TestClock_driftSignal`) is a table over
`{op: "tick"|"update", initialL, nowMs, warn, want}` — add rows for received lead exactly `drift_warn`
→ 0, `drift_warn+1ms` → positive, wall ahead of remote → 0, zero/negative warn disabled. Rebuild the
`Update` subtest of `TestClock_driftSignalSaturates` (`clock_test.go:586`) so `remoteL` is the far-ahead
operand.

---

### `internal/app/persist/codec.go` + `doc.go` (library value-limit guard, IN-04)

**Analog:** itself (`codec.go:119-160`; `doc.go:31-34`).

The guard must run **before** the header encode; today the first write is the header:

```go
func EncodeTo(w io.Writer, records map[string]storage.Record, valueMaxBytes int) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	count := len(records)
	err := enc.Encode(header{FormatVersion: dumpFormatVersion, RecordCount: &count})
	...
```

Target (research D-04):

```go
if valueMaxBytes <= 0 || valueMaxBytes > maxDumpValueBytes {
	return fmt.Errorf("persist: value limit %d is outside (0, %d]: %w",
		valueMaxBytes, maxDumpValueBytes, storage.ErrValueTooLarge)
}
```

`maxDumpValueBytes = 65507` is `codec.go:40-41`. Update `doc.go:31-34` to say the guard now exists.

**Test shape:** `internal/app/persist/codec_test.go:332-388` (`TestCodec_discardClasses`) is the table
pattern; the new test asserts `EncodeTo(&buf, nil, maxDumpValueBytes+1)` errors and `buf.Len() == 0`
(same for 0/-1), and a valid limit still writes.

---

### `internal/app/persist/scheduler.go` (WARN cadence, dump-duration semantics)

**Analog:** itself.

- `runDump` measures and records the Encode+Save span for success **and** failure (`scheduler.go:228-249`):
  `s.recordDump(s.cfg.Now().Sub(start), err)` runs before the error check. This is why the metric Help must
  say "last dump **attempt**".
- `handleDumpFailure` is the existing bounded-cadence WARN source (`scheduler.go:251-277`): one
  `OnDumpFailure` per failed attempt, cadence bounded by the retry ladder. Do not add a second failure log
  in `main`; reuse this callback.
- `nextRetry`/`resetRetry` (`scheduler.go:279-302`) implement 100 ms doubling to a 5 s cap.
- `recordDump` (`scheduler.go:304-313`) is the atomics shape to cite in the metric definition test.

No behavior change is required here beyond documenting the semantics; if the plan adds a cadence guard, it
must sit on this same callback path.

---

### `internal/metrics/metrics.go` (families, Sources, RSS)

**Analog:** itself.

**Sources seam** (`metrics.go:183-200`) — add four fields + keep the closed-set discipline:

```go
type Sources struct {
	Discards      func() repl.DiscardStats
	KnownPeers    func() uint64
	Unacked       func() repl.UnackedStats
	Sent          func() udprepl.SentByType   // <-- must become repl.SentByType
	Retransmitted func() uint64
	MergeStats    func() repl.MergeStats
	DumpStats     func() persist.DumpStats
	LoadStats     func() persist.LoadStats
	Paced         func() uint64
	SendQueue     func() int
	EgressDropped func() uint64
	ReadErrors    func() uint64
	// NEW: ListingDropped, SyncStreamDropped, ListedKeysEvicted, NotificationsRegressed
}
```

`Snapshot` (`metrics.go:202-218`) calls every accessor unconditionally — update `zeroSources` and every
`Sources` literal in the same commit (Pitfall 15).

**Definition/Help style** (`replicationFamilies` 268-318, `trafficFamilies` 357-392,
`dumpFamilies` 405-445):

```go
{
	Name: "supkv_merges_conflicts_total", Type: Counter,
	Help: "Pending (unacked) local writes displaced by a replicated value.",
	Samples: []Sample{{Value: float64(merged.Conflicts)}},
},
{
	Name: "supkv_dumps_duration_seconds", Type: Gauge,
	Help: "Duration of the last dump attempt's encode and save in seconds.",
	Samples: []Sample{{Value: stats.DurationSeconds}},
},
```

Rewrite the conflict/drift Helps, add the four new families (one `listingFamilies` helper is enough),
and add `{class="unbound_listing"}` to `trafficFamilies` (complete label catalog, zero-valued when unseen).
Keep names `supkv_replication_unacked_records` and `supkv_replication_datagrams_retransmitted_total`
unchanged — the frozen red tests scrape them by exact name.

**RSS split:** `runtimeFamilies` (`metrics.go:450-491`) currently calls `processUsage`; change it to call
`currentRSS()` and keep the CPU half. `processUsage` (`metrics.go:493-509`) is the fallback source for
`rss_other.go`.

---

### `internal/metrics/rss_linux.go` / `rss_other.go` (build-tagged platform probe)

**No direct analog** — the repo has no build-tagged production file. The only build-tag precedent is
`internal/stand/stand_nondarwin_test.go:1`:

```go
//go:build !linux
```

Use research's D-05 implementation:

```go
//go:build linux

package metrics

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

`rss_other.go` (`//go:build !linux`) keeps `syscall.Getrusage`/`ru_maxrss` as the documented
development-only darwin fallback (ADR 0010). Help must say "current resident set size (Linux);
darwin dev-build peak".

---

### `internal/udprepl` split (socket / codec / loop / mirror / adapter)

**Analog:** `internal/udprepl/repl.go` — this is a **file-only move**, same package `udprepl`, no exported
name changes, no test edits (D-08).

**Package doc and invariants** (`repl.go:1-15,257-264`): copy the package doc verbatim to the new
`adapter.go` (or keep at the top of `socket.go`); preserve the single-driver rule:

```go
// Adapter is the UDP replication adapter for one node: it owns the bound
// socket and drives one repl.Core. Exactly one goroutine — the caller's
// Run — mutates core state (D-04); ...
```

**Split map (verified declarations):**

| New file | Move |
|----------|------|
| `socket.go` | `packetConn` 217-232, `options`/`Option`/`WithAddressReuse` 234-255, `listenDatagram` 464-491, `configureSocketBuffers` 493-529, `validateBufferRequest` 531-541, `socketOption` 543-567, `checkSendBuffer` 569-584 |
| `codec.go` | `encodeDatagram` 1251-1272, `decode` 1285-1322, `protoBody` 1324-1357, `portMessage` 1359-1408, `recordVersion` 1410-1417 (boundaries from the research responsibility map, checked against the tree) |
| `loop.go` | `inboundEvent`/`localWrite` 125-150, `Run` 586-625, `readLoop` 1066-1086, `logReadError` 1092-1101, `flush` 1111-1130, `countSent` 1132-1148, `write` 1150-1156, `armTimer`/`timerChan`/`stopTimer` 1217-1245, tracker/notify methods |
| `mirror.go` | `Adapter` mirror fields 284-358, `listingMirror` 1023-1030, `mirrorStats` 1158-1215, accessors 870-1038 |
| `adapter.go` (optional) | `Adapter` struct 257-370, `New` 393-439, package doc |

**Send accounting change (D-05)** — `flush`/`write` in `loop.go` (`repl.go:1103-1156`): count only after a
successful socket write; `write` returns an error; resolve failures and encode failures count nowhere:

```go
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
				a.log(...)
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
```

`noinlineerr` forbids the inline `if err := a.write(...); err != nil` form.

**Mirror extension** — `mirrorStats` (`repl.go:1167-1215`) is the template: copy every core accessor into
an `atomic.Uint64` field and expose a value-copy accessor (e.g. `Discards` 884-898, `MergeStats` 937-942).
Add four atomic fields/accessors for the Phase-4 counters; add `UnboundListing` to the `Discards()` literal.

**SentByType relocation (D-07):** move `SentByType` (`repl.go:974-995`) to `internal/app/repl`; if tests
still reference `udprepl.SentByType`, keep `type SentByType = repl.SentByType` in `udprepl`. After the move
`internal/metrics` must no longer import `udprepl`.

**Mirror test analog:** `internal/udprepl/repl_internal_test.go:448-507`
(`TestMirror_identicalListingFloodDoesNotRepublish`) drives `adapter.core.HandleDatagram` directly,
calls `adapter.mirrorStats()`, then reads the atomic accessors — copy this shape for the four counters and
the send-failure case.

---

### `internal/grpcapi/gate.go` (OnGateOpen seam)

**Analog:** itself.

Add one nil-valid field to `ServerOptions` (`gate.go:47-111`), documented like `SyncArrived`:

```go
// OnGateOpen receives the gate-opening reason exactly once: "sync" when a
// complete listing arrived, "backstop" when StartupGate elapsed. Nil is a
// no-op; the callback runs on RunGate's goroutine and must not block.
OnGateOpen func(reason string)
```

Emit inside `openGate` (`gate.go:188-197`) after the CAS succeeds, so the transition stays single-shot:

```go
func (s *Server) openGate(reason string) {
	if s.gateOpen.CompareAndSwap(false, true) {
		s.healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
		if s.opts.OnGateOpen != nil {
			s.opts.OnGateOpen(reason)
		}
	}
}
```

Call sites: `RunGate`'s backstop branch and poll branch (`gate.go:163-186`).
Test analog: `internal/grpcapi/gate_test.go:231-293` (`TestGate_opensOnSync`, `TestGate_opensOnTimeout`) —
use a buffered channel for the callback because it fires on the resolver goroutine.

`internal/grpcapi/server.go` needs **no** Phase-7 edit: its `persistfile`/`*persist.Engine` imports are the
explicitly allowed exception recorded in `.golangci.yml` (D-07 refinement).

---

### `.golangci.yml` (additive depguard rules)

**Analog:** itself. Lines 65-95 are the project-additions block; lines 1-64 are byte-frozen upstream and
must not change.

Existing rule shape to copy (`lines 68-95`):

```yaml
        sup-kv-app-no-io:
          files:
            - '**/internal/app/**'
            - '!$test'
          deny:
            - pkg: net
              desc: internal/app is the pure decision core - no networking
        sup-kv-port-stdlib-only:
          files:
            - '**/internal/port/**'
            - '!$test'
          list-mode: strict
          allow:
            - $gostd
            - github.com/Djarvur/sup-kv
```

Append after line 95: `sup-kv-adapters-not-in-core`, `sup-kv-metrics-no-adapters`,
`sup-kv-grpcapi-no-repl-metrics` exactly as in research D-07, plus the written justification comment for
the allowed `grpcapi -> persistfile` / `grpcapi -> app/persist` composition edge. Rules:
- name adapter packages individually; never deny `github.com/Djarvur/sup-kv/internal` wholesale;
- always include `!$test`;
- run `go tool golangci-lint config verify` (`make lint` already does) and probe each rule with a temporary
  forbidden-import file, then delete it.

---

### `docs/OPERATIONS.md` + `cmd/sup-kv/operations_test.go` (doc anchors)

**Analog:** the existing anchor test is the contract.

`operations_test.go:42-98` pins one anchor literal per statement:

```go
var operationsAnchors = []struct {
	literal string
	pins    string
}{
	{"## Clock drift runbook", "the clock-drift runbook section"},
	{"supkv_clock_drift_warnings_total", "the drift symptom metric"},
	{"--drift-warn", "the drift warning threshold knob (default 60s)"},
	{"exit code 1", "failed final dump (Restart=on-failure reacts)"},
	// ...
}
```

`TestOperationsDoc` (`operations_test.go:104-125`) also enforces "## Trust model" is the **first** `##`
section. `TestOperationsDocFamilies` (`:143-162`) greps every `supkv_*` name in the doc and requires it in
`internal/metrics/metrics.go`.

Required ripple in one commit:
- `exit code 1` anchor moves to startup failure; add `exit code 3` for final-dump failure
  (`docs/OPERATIONS.md:280-293`);
- add the four new families and `class="unbound_listing"` to the monitoring list (`:238-273`);
- drift runbook (`:215-236`): symptom says "received version leads local wall time"; remove "or ahead of"
  from Cause; add the rate-limit and "a node ahead does not warn itself" statements; add an anchor literal
  such as "received version leads";
- `--max-clock-lead` default 60s → 2m (`:89-92`);
- exit code 3 and `Restart=on-failure` reaction text.

`cmd/sup-kv/api_docs_test.go:133-148` is the minimal anchor-test shape if a second anchor list is wanted.

---

### `docs/SPEC-DEVIATIONS.md` (registry ripple)

**Analog:** itself. Rows are `| ID | Summary | Status | Links |` with status vocabulary
`fixed-in-v1.1(...)`, `implemented`, `superseded-by-amendment`, `accepted-deviation(...)`.

Rows to update (verified line numbers):
- `S-HLC-13` line 61 → `implemented`; cite the new `Clock.Update` lead check, `MergeStats.DriftWarnings`,
  WARN seam and drift tests.
- `S-PAR-16` line 243 → `implemented`; cite the 60 s `drift_warn` default, the new `max-clock-lead` default
  and the `drift_warn < max_clock_lead` rule.
- `S-FAIL-17` line 212 → `implemented`; cite the WARN path at the proto-version branch and its level test.
- `S-SEM-07` line 183 → update the note to pending-local displacement.
- `S-IMP-08` line 256 → keep `accepted-deviation(ADR 0008)`; update evidence to say the counters are honest.
Do not touch `S-HLC-07` (ADR 0012) or the unimplemented v2 rows.

---

### `AGENTS.md` (layout/registry ripple)

**Analog:** itself.
- `## Layout` (line 35) must gain `internal/config/` as a composition-level package (stdlib + own module,
  imports `repl`/`storage`/`udprepl` constants only; not under the `internal/app` no-I/O rule).
- `### Project deviations` (line 10) item 1 must mention the new additive depguard rules
  (`sup-kv-adapters-not-in-core`, `sup-kv-metrics-no-adapters`, `sup-kv-grpcapi-no-repl-metrics`) and the
  written `grpcapi -> persistfile` exception.
- Keep the go-ultimate deviation register accurate; no new runtime dependency is allowed.

---

### Test-ripple files (grouped)

**`internal/stand/stand_metrics_test.go`** — `TestStand_metricsByEffect` (`:151-230`) currently builds a
conflict by having B overwrite A's *acked* value (`:175-189`) and asserts `conflicts == 1` (`:210-217`).
Under the pending-only definition this scenario must be rebuilt: A writes while B is stopped (record stays
pending), then a crafted valid `DataReplication` with a higher version displaces it; assert 1. Keep the
bad_crc and retransmit effects unchanged. This is the one non-frozen stand test that must change in the same
commit as the conflict semantics, or the red gate fails.

**`internal/app/repl/merge_stats_test.go`** — `TestMergeStats_counts` (`:28-85`) currently seeds the store
directly (no pending) and expects 1; rewrite to seed through `NotifyLocalWrite`/`trackLocalWrite` and expect
1 only for pending displacement, 0 for confirmed overwrite/stale. `TestMergeStats_listingAndStaleExcluded`
(`:95-149`) has the comment "listing drift is clock-only this phase (OQ6)" — update it for received-lead
DataSync counting.

**`internal/app/repl/warn_test.go`** — `TestDiscard_warnLimiterKeyedByNodeAndClass` (`:35-125`) is the
limiter table to copy for the apply-failure limiter and drift WARN.

**`internal/app/repl/discard_test.go`** — the model's class-string switch (`:315-350`) and the totality sum
must add `unbound_listing` and move the unbound-listing expectation off `DuplicateNodeID`.

**`internal/app/repl/listedkeys_internal_test.go`** — `TestSync_unboundListingDiscarded` (`:134-170`) and
`TestSync_forgedRunIDCannotReplaceListing` (`:177-215`) both assert `Discards().DuplicateNodeID`, and BOTH
route through `listingBound` (an unknown node_id; a known node_id with a mismatched run_id), so both
expectations move to `UnboundListing == 1` with `DuplicateNodeID == 0`. `DuplicateNodeID` stays only for the
own-node-id-foreign-run, the second-handle and the alternating-run paths inside `HandleDatagram`/`handleHello`.

**`internal/udprepl/decode_negative_test.go`** and **`internal/udprepl/scenario_test.go`** — add
`unbound_listing` to the class-string switches/totality models (research lists exact lines; re-grep before
editing).

**`internal/metrics/metrics_test.go`** — update `families` (`:186-216`), `zeroSources` (`:220-235`), the
zero-state sample list (`:242-301`) and `TestMetrics_values` (`:362-418`) for the four new families and the
new discard label; add `TestMetrics_definitions` parsing each `# HELP` and asserting a semantic phrase
("pending", "successfully", "last dump attempt", "leads", "current").

## Shared Patterns

### Nil-safe callback seams
**Source:** `internal/app/persist/engine.go:44-58`, `internal/grpcapi/gate.go:47-111`
**Apply to:** `repl.Config.OnPeerEvent`/`OnDrift`, `grpcapi.ServerOptions.OnGateOpen`, any new callback.

```go
if s.cfg.OnDumpFailure != nil {
	s.cfg.OnDumpFailure(err, class)
}
```

Callbacks run on the single driver/resolver goroutine and must not block; nil is always a no-op.

### Per-(node_id, class) WARN rate limiter
**Source:** `internal/app/repl/core.go:1219-1246`
**Apply to:** `apply_failure`, `unbound_listing`, drift WARN, and every new WARN that can be driven by a peer.

```go
key := scanDiscardKey{nodeID: nodeID, class: class}
last, warned := c.discardWarnAt[key]
if warned && now.Sub(last) < discardWarnInterval {
	return
}
c.discardWarnAt[key] = now
c.cfg.Log(msg)
```

Counters stay per-event; the limiter suppresses only the log line. Time is the handler's `now`, never wall
clock.

### Mirror-then-export
**Source:** `internal/udprepl/repl.go:1158-1215`, `internal/metrics/metrics.go:183-218`
**Apply to:** the four Phase-4 counters, send accounting, and any new metric.

Core counters mutate on the driver goroutine only; `mirrorStats` copies them into atomics after every loop
event; `Sources` reads the atomics once per scrape; `Snapshot` renders the closed family set with zero-valued
samples for unseen labels.

### Browser/process log assertions parse JSON records, never formatted text
**Source:** `cmd/sup-kv/main_test.go:1002-1024` (`parseSocketBuffersLine`) and `:1284-1327`
(`TestLog_jsonDefault`)
**Apply to:** startup/load/bind/Serve/final-dump/gate/peer/drift log tests.

```go
var entry struct {
	Msg     string `json:"msg"`
	RecvBuf *int   `json:"recv_buf"`
	SendBuf *int   `json:"send_buf"`
}
if json.Unmarshal([]byte(line), &entry) != nil || entry.Msg != "Socket buffers" { continue }
```

Use `startLoggedNode` (lock-guarded `logBuffer`) when reading a live process; a plain `bytes.Buffer` is
safe only after exit (Pitfall 13).

### Process exit assertions
**Source:** `cmd/sup-kv/main_test.go:290-307`
**Apply to:** exit-code table tests (0/1/2/3), final-dump test, bind failure at `SUPKV_LOG_LEVEL=error`.

```go
func assertExitCode(t *testing.T, err error, want int) {
	t.Helper()
	if err == nil {
		if want != 0 { t.Fatalf("process exited 0, want %d", want) }
		return
	}
	var ee *exec.ExitError
	if !errors.As(err, &ee) { t.Fatalf("process wait error = %v, want an ExitError", err) }
	if ee.ExitCode() != want { t.Fatalf("process exit code = %d (%s), want %d", ee.ExitCode(), ee.String(), want) }
}
```

### Additive lint config over a vendored body
**Source:** `.golangci.yml:1,65-95`
**Apply to:** every boundary rule.

Only append under `linters.settings.depguard.rules` after the line-65 banner. Never edit lines 1-64; run
`make lint` (which starts with `config verify`) and a temporary forbidden-import probe per new rule.

### Doc anchors + exporter cross-check
**Source:** `cmd/sup-kv/operations_test.go:42-162`, `cmd/sup-kv/api_docs_test.go:133-148`
**Apply to:** `docs/OPERATIONS.md`, `docs/SPEC-DEVIATIONS.md`, `AGENTS.md`.

Pin one literal per mandated statement; cross-check family names against `internal/metrics/metrics.go`.
Update the anchors in the same commit as the doc.

### Frozen contracts (do not "fix" these)
**Apply to:** every file above.
- `scripts/red-expectations.tsv`, `scripts/red-runner`, `internal/stand/stand_red_test.go` and
  `internal/grpcapi/red_gate_test.go`: untouched.
- Mutation sed targets: keep `Core.syncListing`'s `nextSync` gate/return shape (`core.go:2383`) and
  `internal/app/storage/store.go:248,269` byte-identical in shape.
- Keep metric names `supkv_replication_unacked_records` and
  `supkv_replication_datagrams_retransmitted_total`.
- Clean SIGTERM shutdown must remain exit 0 (the red tests' `signalAndWaitExit` depends on it).
- No `os.Exit` outside `cmd/sup-kv`; no `slog` import inside `internal/app`.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/config/usage.go` | config/usage | transform | No custom `flag.Usage` exists today — only the stdlib default. Use research D-03's env-only block; the flag descriptions it prints already exist in `main.go:341-402`. |
| `internal/metrics/rss_linux.go` | utility/platform | file-I/O | No production build-tagged file exists; use research's `/proc/self/statm` sketch and the `//go:build` form from `internal/stand/stand_nondarwin_test.go:1`. |
| `internal/metrics/rss_other.go` | utility/platform | syscall | Only the peak-RSS `processUsage` half exists (`metrics.go:493-509`); the current-vs-peak split is new. |

`internal/config/config.go` and `internal/config/validate.go` are **extraction targets**, not greenfield
design: their code exists in `cmd/sup-kv/main.go`, so copy the excerpts above rather than inventing a shape.

## Metadata

**Analog search scope:** `cmd/sup-kv/`, `internal/app/{repl,clock,persist,storage,version}/`,
`internal/{metrics,udprepl,grpcapi,port,stand}/`, `api/`, `docs/`, repo root `.golangci.yml`, `Makefile`.
**Files classified:** 43; source/test/doc files read in full or in targeted ranges: 38.
**Pattern extraction date:** 2026-11-23.
**Tracked-source verification:** every analog path passed `git ls-files -- <path>`; no mirror paths cited.
**Frozen contracts:** `scripts/red-expectations.tsv`, `scripts/red-runner`, `internal/stand/stand_red_test.go`,
`.golangci.yml` lines 1-64 — read-only for this phase.
