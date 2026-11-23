// Command sup-kv is a leaderless replicated in-memory key-value store for
// small clusters on one trusted LAN broadcast segment.
//
// Per the layout contract, main() is wiring only: configuration parsing,
// validation and usage live in internal/config (OBSV-02, OBSV-08, D-03),
// signals and the process lifecycle live here, and every behavior (write
// confirmation, replication, watch, snapshots) lives in the packages being
// wired. Configuration resolves once — flag > env > default (D-16/D-17) —
// and is validated before any socket or file work. SIGTERM runs the pinned
// graceful-shutdown order (D-19) whose final step is the engine's
// write-confirmed dump.
package main

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	goruntime "runtime"
	"syscall"
	"time"

	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/config"
	"github.com/Djarvur/sup-kv/internal/grpcapi"
	"github.com/Djarvur/sup-kv/internal/metrics"
	"github.com/Djarvur/sup-kv/internal/persistfile"
	"github.com/Djarvur/sup-kv/internal/port"
	"github.com/Djarvur/sup-kv/internal/udprepl"
)

// version and commit are injected at link time via -ldflags "-X
// main.version=... -X main.commit=..." (D-18) — enough to identify a
// running build during incident triage; "dev" stands in for any field the
// build did not stamp.
var (
	version = "dev"
	commit  = "dev" //nolint:gochecknoglobals // the -X ldflags mechanism (D-18) can only write package-level vars.
)

// Process exit codes (D-06, O-05): a clean shutdown whose final
// write-confirmed dump completed exits 0; a usage or configuration error
// exits exitUsage; a startup or Serve failure exits exitStartup; and a failed
// final dump during shutdown exits exitDumpFailed. Precedence: the final-dump
// failure (3) wins over a Serve error (1) — the dump failure is the
// operator's data-loss signal — while a startup failure never runs a final
// dump (no client-confirmed write exists yet) and can only be 1.
const (
	exitStartup    = 1
	exitUsage      = 2
	exitDumpFailed = 3
)

// The wiring stages a startup failure names: each fatal step of wire reports
// its stage, so the one Startup-failed ERROR record identifies where the node
// gave up instead of a bare message (D-06).
const (
	stageReplicationBind = "replication_bind"
	stageGRPCListen      = "grpc_listen"
	stageMetricsListen   = "metrics_listen"
)

// startupError carries the wiring stage a startup failure came from. run logs
// its stage and detail as one ERROR record, so a fatal startup fault survives
// SUPKV_LOG_LEVEL=error instead of the pre-D-06 fatal-log exit, which rendered
// as an INFO record and vanished at that level (review O-01).
type startupError struct {
	stage string
	err   error
}

// Error renders the stage and the cause for any caller that sees only an
// error string.
func (e *startupError) Error() string {
	return e.stage + ": " + e.err.Error()
}

// Unwrap exposes the underlying cause to [errors.Is] and [errors.As].
func (e *startupError) Unwrap() error { return e.err }

// startupStage names the stage err carries, "startup" when it carries none.
func startupStage(err error) string {
	if staged, ok := errors.AsType[*startupError](err); ok {
		return staged.stage
	}
	return "startup"
}

// loadFailureAttrsPerCandidate is the number of attributes one candidate
// load failure contributes: its path, its cause and its preservation error.
const loadFailureAttrsPerCandidate = 3

// The startup/load catalog's stable outcome and generation tokens (D-01):
// every start's one load record carries them, so the values are constants
// rather than repeated literals.
const (
	loadTokenPrompt   = "prompt"
	loadTokenMain     = "main"
	loadTokenBackup   = "backup"
	loadTokenUnusable = "unusable"
	loadTokenPrev     = "prev"
	loadTokenNone     = "none"
)

// shutdownGrace bounds GracefulStop during the D-19 shutdown: past it,
// gs.Stop() force-closes whatever is still open (T-05-10 — a hang here
// would leave committed data only in memory when the supervisor SIGKILLs).
const shutdownGrace = 10 * time.Second

// readHeaderTimeout bounds one metrics/debug connection's header read — the
// explicit [http.Server] construction's timeout floor (never a bare
// [http.ListenAndServe]; the gosec-clean shape).
const readHeaderTimeout = 5 * time.Second

// signalBuffer is the armed signal channel's capacity: room for the first
// signal (consumed by serve) plus one hurry-up signal held for shutdown's
// force path while it is still selecting. Further signals are dropped by
// the signal package — the force path is already taken.
const signalBuffer = 2

// wired bundles the running services with their lifecycle handles — the
// plumbing main's shutdown path needs, no behavior of its own. metricsSrv is
// nil when --metrics-addr is unset: the endpoint is opt-in with no listener
// at all (D-01). api is the gRPC API server; its one-way Shutdown closes the
// health endpoint (D-05, APIS-05).
type wired struct {
	engine       *persist.Engine
	engineCancel context.CancelFunc
	engineDone   <-chan error
	replCancel   context.CancelFunc
	replDone     <-chan struct{}
	apiCancel    context.CancelFunc
	api          *grpcapi.Server
	gs           *grpc.Server
	metricsSrv   *http.Server
}

// runtime is the lifecycle seam run orchestrates (D-06): the four production
// functions and the signal release, declared with exactly the directions they
// have, so productionRuntime is a direct assignment and no adapter closure
// stands between the tested seam and the real path.
type runtime struct {
	installSignals func() chan os.Signal
	stopSignals    func(chan<- os.Signal)
	wire           func(config.Config) (*wired, net.Listener, error)
	serve          func(*wired, net.Listener, <-chan os.Signal) error
	shutdown       func(*wired, <-chan os.Signal) error
}

func main() {
	cfg, err := config.Parse(os.Args[1:], os.Getenv)
	if errors.Is(err, config.ErrHelp) {
		config.Usage(os.Stdout)
		return
	}
	if err != nil {
		usageError(err)
		os.Exit(exitUsage)
	}
	// --version is resolved by Parse before any environment or validation
	// error (Pitfall 20): the build-identity probe works on a broken
	// environment, and nothing below runs for it.
	if cfg.ShowVersion {
		printVersion()
		return
	}
	err = config.Validate(cfg)
	if err != nil {
		usageError(err)
		os.Exit(exitUsage)
	}

	// The one slog handler install (D-06/D-07) precedes every service
	// goroutine: the udprepl Log tap, the adapter's discard events and the
	// log-package calls redirected through slog.SetDefault all flow through
	// this handler for the process's whole life.
	setupLogging(cfg)

	// XPORT-09: a non-wildcard bind is loud before any service starts — the
	// node would send broadcasts but never receive them.
	warnOnNonWildcardBind(cfg.BindAddr)

	// The operator's first record (D-01, OBSV-01): what this binary is and
	// every effective knob, secrets-free. It precedes wire, so a bind failure
	// still has a preceding Starting record at info level.
	logStartup(cfg)

	os.Exit(run(cfg, productionRuntime()))
}

// run is the lifecycle seam (D-06): signals are armed before wire, so a
// SIGTERM during dump load or bind still takes the graceful path instead of
// killing the process with the default disposition (P1-IN-01). A wire failure
// is one ERROR record naming the failing stage and returns exitStartup
// without a shutdown — no client-confirmed write exists yet. A Serve error is
// one ERROR record but still runs the full D-19 shutdown, bounded final dump
// included, before the exit code is chosen; precedence makes a failed final
// dump (3) win over the Serve error (1) on the same path. Signal ownership
// spans serving and shutdown alike: the channel is released only after the
// final dump, so a second SIGTERM during the drain is observed as the
// operator's hurry-up and takes shutdown's force path.
func run(cfg config.Config, rt runtime) int {
	sig := rt.installSignals()
	node, listener, err := rt.wire(cfg)
	if err != nil {
		slog.Error("Startup failed", "stage", startupStage(err), "detail", err.Error())
		rt.stopSignals(sig)
		return exitStartup
	}
	serveErr := rt.serve(node, listener, sig)
	if serveErr != nil {
		slog.Error("Serve failed", "detail", serveErr.Error())
	}
	shutdownErr := rt.shutdown(node, sig) // D-19: the order is the contract
	if shutdownErr != nil {
		slog.Error("Final dump failed", "detail", shutdownErr.Error())
	}
	rt.stopSignals(sig)
	switch {
	case shutdownErr != nil:
		return exitDumpFailed
	case serveErr != nil:
		return exitStartup
	default:
		return 0
	}
}

// productionRuntime assembles run's seam from the real lifecycle functions.
func productionRuntime() runtime {
	return runtime{
		installSignals: armSignals,
		stopSignals:    signal.Stop,
		wire:           wire,
		serve:          serve,
		shutdown:       shutdown,
	}
}

// usageError reports a configuration error through the usage path: the error
// names the offending parameter, the full usage follows on stderr, and main
// exits with exitUsage. It never exits itself, so main stays the only exit
// site.
func usageError(err error) {
	fmt.Fprintf(os.Stderr, "sup-kv: %s\n", err)
	config.Usage(os.Stderr)
}

// setupLogging installs the process's one slog handler on stderr (D-06,
// D-07): JSON records by default, key=value text when SUPKV_LOG_FORMAT=text,
// at the level SUPKV_LOG_LEVEL resolved (debug through error). Called once
// from main before wire launches any goroutine. [slog.SetDefault] also
// redirects the log package's default logger, so the remaining direct
// [log.Printf] calls surface as level=INFO records in the same format —
// every stderr line the running node emits is one handler-formatted record.
func setupLogging(cfg config.Config) {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	var h slog.Handler = slog.NewJSONHandler(os.Stderr, opts)
	if cfg.IsTextFormat() {
		h = slog.NewTextHandler(os.Stderr, opts)
	}
	slog.SetDefault(slog.New(h))
}

// logStartup emits the lifecycle catalog's first record (D-01, OBSV-01): the
// build identity and every effective configuration value, so an operator can
// reconstruct what the node is and how it was launched from one line. Empty
// values are logged as empty — an unset metrics address is meaningful — and
// the certificate/key material is deliberately absent: only the metrics
// address and whether TLS is on cross this boundary, never the env-only
// cert/key/CA paths or their contents (T-07-21).
func logStartup(cfg config.Config) {
	slog.Info("Starting sup-kv",
		"version", version,
		"commit", commit,
		"go_version", goruntime.Version(),
		"node_id", cfg.NodeID,
		"bind_addr", cfg.BindAddr,
		"broadcast_addr", cfg.BroadcastAddr,
		"dump_path", cfg.DumpPath,
		"grpc_addr", cfg.GRPCAddr,
		"metrics_addr", cfg.MetricsAddr,
		"metrics_tls", cfg.MetricsCert != "",
		"value_max_bytes", cfg.ValueMaxBytes,
		"sync_chunk_bytes", cfg.SyncChunkBytes,
		"sync_response_rate", cfg.SyncResponseRate,
		"startup_gate", cfg.StartupGate.String(),
		"startup_gate_empty", cfg.StartupGateEmpty.String(),
		"wait_timeout", cfg.WaitTimeout.String(),
		"hello_interval", cfg.HelloInterval.String(),
		"peer_timeout", cfg.PeerTimeout.String(),
		"ack_timeout", cfg.AckTimeout.String(),
		"ack_timeout_max", cfg.AckTimeoutMax.String(),
		"retransmit_interval", cfg.RetransmitInterval.String(),
		"sync_interval", cfg.SyncInterval.String(),
		"max_clock_lead", cfg.MaxClockLead.String(),
		"dump_debounce", cfg.DumpDebounce.String(),
		"dump_max_wait", cfg.DumpMaxWait.String(),
		"drift_warn", cfg.DriftWarn.String(),
		"log_level", cfg.LogLevel.String(),
		"log_format", cfg.LogFormat,
	)
}

// warnOnNonWildcardBind emits XPORT-09's one startup WARN when --bind-addr
// names a specific interface host instead of a wildcard: the segment's
// broadcast datagrams are addressed to the broadcast address, and a socket
// bound to one unicast address never receives them — the node would
// replicate one way, sending itself but hearing no peer. Deliberately a WARN
// and never a rejection (RESEARCH Pattern 10): a unicast-only deployment
// stays usable, it just must not expect broadcast discovery. The wildcard
// forms are "" (all interfaces), 0.0.0.0 and ::; the flag validation already
// guaranteed host:port, so a SplitHostPort fault only reaches Debug.
func warnOnNonWildcardBind(bindAddr string) {
	host, _, err := net.SplitHostPort(bindAddr)
	if err != nil {
		slog.Debug("Bind address is not host:port", "bind_addr", bindAddr, "error", err)
		return
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		return
	}
	slog.Warn("Bind address is not a wildcard: broadcast datagrams are not received",
		"host", host,
		"detail", "broadcast replication is addressed to the segment broadcast address, which a non-wildcard bind never receives; unicast-only deployment assumed")
}

// gateBackstopFor resolves the startup-gate backstop once, from the loaded
// store's record count (D-02, ADR 0006): the engine finished loading before
// the wiring runs, so KeyCount is final and the resolver reads only the
// duration. A node with any record — live or tombstoned — never consults
// startup_gate_empty.
func gateBackstopFor(cfg config.Config, engine *persist.Engine) time.Duration {
	if engine.KeyCount() == 0 {
		return cfg.StartupGateEmpty
	}
	return cfg.StartupGate
}

// logGateOpen renders the gate resolver's one opening event (D-01): a static
// INFO message with the stable reason attribute — "sync" when a complete
// listing arrived, "backstop" when the startup gate elapsed. It runs on
// RunGate's goroutine and returns immediately.
func logGateOpen(reason string) {
	slog.Info("Startup gate opened", "reason", reason)
}

// logPeerEvent renders one peer-table lifecycle transition (D-01, OBSV-01).
// Appearance is the INFO lifecycle event; a restart and an eviction are WARN
// because both explain a change in replication reach. Every value is a
// structured attribute — the handle is a quoted value and a run id lowercase
// hex, never message text (T-07-20).
func logPeerEvent(ev repl.PeerEvent) {
	switch ev.Kind {
	case repl.PeerAppeared:
		slog.Info("Peer appeared",
			"node_id", ev.NodeID,
			"handle", ev.Handle,
			"run_id", hexRunID(ev.RunID))
	case repl.PeerRestarted:
		slog.Warn("Peer restarted",
			"node_id", ev.NodeID,
			"handle", ev.Handle,
			"previous_run_id", hexRunID(ev.PreviousRunID),
			"run_id", hexRunID(ev.RunID))
	case repl.PeerEvicted:
		slog.Warn("Peer evicted",
			"node_id", ev.NodeID,
			"handle", ev.Handle,
			"reason", "peer_timeout",
			"last_seen_age", ev.LastSeenAge.String())
	}
}

// logDriftEvent renders one clock-drift WARN (D-01, D-02): the received
// version's lead and the configured threshold as duration strings beside the
// peer identity. The handle is the same attribute the peer events carry —
// sloglint reserves the key "source" for the handler's own source attribute,
// so the event's Source field rides as handle. The core has already applied
// the per-peer rate limit.
func logDriftEvent(ev repl.DriftEvent, driftWarn time.Duration) {
	slog.Warn("Clock drift",
		"node_id", ev.NodeID,
		"handle", ev.Source,
		"lead", ev.Lead.String(),
		"drift_warn", driftWarn.String())
}

// hexRunID renders a run identifier as lowercase hex — the stable run_id log
// form.
func hexRunID(run [16]byte) string {
	return hex.EncodeToString(run[:])
}

// logAdapterWarn renders one adapter WARN detail string (D-08): the core's
// prebuilt metadata strings (REPL-08 configuration errors, apply failures,
// the CR-01 unbound-listing drop and the foreign proto_version discard) ride
// as the detail attribute under a static capitalized msg — the sloglint
// static-msg discipline; a dynamic string never becomes the msg. It is the
// named Log seam the catalog level matrix asserts.
func logAdapterWarn(detail string) {
	slog.Warn("Replication adapter", "detail", detail)
}

// logDumpFailure renders the engine's one-ERROR-per-failed-attempt callback
// (D-05, PERSIST-05): a static message carrying the failure class label the
// metric uses and the error text as detail. The engine's dumper goroutine
// only formats and logs — it never renders.
func logDumpFailure(err error, class port.FailureClass) {
	slog.Error("Dump failed", "class", class.String(), "detail", err.Error())
}

// startEngine opens the real dump store at the configured path, resolves
// the engine over it with the hub's change tap threaded through
// storage.Config and the hub's durability watermark through
// persist.Config.OnConfirmed, and launches the single dumper goroutine.
// Returns the engine with its lifecycle handles.
func startEngine(cfg config.Config, hub *grpcapi.WatchHub) (*persist.Engine, context.CancelFunc, <-chan error) {
	dumps := persistfile.New(cfg.DumpPath)
	engine, discarded := persist.Open(cfg.WireNodeID(), time.Now, cfg.DriftWarn, dumps, storage.Config{
		ValueMaxBytes: cfg.ValueMaxBytes,
		OnChange:      hub.Ingest,
	}, persist.Config{
		Debounce: cfg.DumpDebounce,
		MaxWait:  cfg.DumpMaxWait,
		// The durability seam (APIS-04, D-04): only a successful dump's
		// covered generation releases Watch events, so a change is announced
		// after it is durable, never at install time.
		OnConfirmed: hub.Confirm,
		// One ERROR per failed attempt with the class label the metric uses
		// (D-05, PERSIST-05): a static message plus attributes, never dynamic
		// text, and the engine's dumper goroutine only formats and logs.
		OnDumpFailure: logDumpFailure,
	})
	stats := engine.LoadStats()
	logLoadStats(stats, engine.KeyCount())
	if skipped := stats.Decoded.ValueLimit; skipped > 0 {
		slog.Warn("Dumped records skipped by the configured value limit",
			"count", skipped,
			"value_max_bytes", cfg.ValueMaxBytes)
	}
	if remaining := discarded - stats.Decoded.ValueLimit; remaining > 0 {
		logDiscardedRecords(remaining)
	}
	engineCtx, engineCancel := context.WithCancel(context.Background())
	engineDone := make(chan error, 1)
	go func() { engineDone <- engine.Run(engineCtx) }()
	return engine, engineCancel, engineDone
}

// logDiscardedRecords reports the undecodable-record count left after the
// value-limit skips were reported separately.
func logDiscardedRecords(count int) {
	//nolint:gosec // G706 is a false positive: the message is a constant and the only argument is a counted int, never operator text.
	log.Printf("sup-kv: discarded %d undecodable dumped records at startup", count)
}

// logLoadStats reports one startup load in the one-record-per-class policy
// (D-02, PERSIST-02/PERSIST-09, OBSV-01): a Prompt or Main load is one INFO
// "Dump load" record; a Backup keeps its single WARN naming the main
// generation's path, cause and preservation error; an Unusable keeps its
// single ERROR carrying one attribute group per candidate that actually
// failed — one or two, never a phantom. All three carry the catalog's four
// outcome attributes, so one record per start says what was loaded
// regardless of outcome. A non-zero sweep count is one INFO and a sweep
// error one WARN; a clean start therefore logs exactly the one load record.
func logLoadStats(stats persist.LoadStats, records int) {
	// The catalog's stable per-start attributes (D-01): the outcome token,
	// the accepted generation, the loaded key count and the decode census.
	loadAttrs := []any{
		"outcome", loadOutcomeName(stats.Outcome),
		"generation", loadGenerationName(stats),
		"records", records,
		"discarded", stats.Decoded.Total(),
	}
	switch stats.Outcome {
	case port.LoadBackup:
		logBackupFallback(stats, loadAttrs)
	case port.LoadUnusable:
		logUnusableGenerations(stats, loadAttrs)
	case port.LoadPrompt, port.LoadMain:
		slog.Info("Dump load", loadAttrs...)
	}
	if stats.Swept > 0 {
		slog.Info("Orphan temp sweep removed files", "count", stats.Swept)
	}
	if stats.SweepErr != nil {
		slog.Warn("Orphan temp sweep failed", "detail", stats.SweepErr.Error())
	}
}

// loadOutcomeName renders the port's load outcome as the catalog's stable
// token: prompt, main, backup or unusable.
func loadOutcomeName(outcome port.LoadOutcome) string {
	switch outcome {
	case port.LoadMain:
		return loadTokenMain
	case port.LoadBackup:
		return loadTokenBackup
	case port.LoadUnusable:
		return loadTokenUnusable
	case port.LoadPrompt:
		return loadTokenPrompt
	}
	return loadTokenPrompt
}

// loadGenerationName names the report's generation for the load record: the
// accepted candidate for a main/backup load, and "none" when nothing was
// accepted (prompt or unusable). The port's zero Generation is GenMain, so
// the outcome — never the enum value alone — decides whether a generation
// was accepted.
func loadGenerationName(stats persist.LoadStats) string {
	switch stats.Outcome {
	case port.LoadMain:
		return loadTokenMain
	case port.LoadBackup:
		return loadTokenPrev
	case port.LoadPrompt, port.LoadUnusable:
		return loadTokenNone
	}
	return loadTokenNone
}

// logBackupFallback emits the one WARN for a .prev fallback, carrying the
// catalog's load attributes beside the cause. The main candidate is found by
// scanning Failures by Candidate — never by slice index — so an absent main's
// recorded not-exist cause is what the line names, and a
// contract-violating empty list degrades to a path-less WARN instead of an
// index panic.
func logBackupFallback(stats persist.LoadStats, loadAttrs []any) {
	failure, ok := candidateFailure(stats.Failures, port.GenMain)
	if !ok {
		slog.Warn("Dump generation fallback", loadAttrs...)
		return
	}
	attrs := append([]any{
		"main_path", failure.Path,
		"cause", errorText(failure.Cause),
		"preserve_error", errorText(failure.PreserveErr),
	}, loadAttrs...)
	slog.Warn("Dump generation fallback", attrs...)
}

// logUnusableGenerations emits the one ERROR for an unusable chain: one
// <candidate>_path/<candidate>_cause/<candidate>_preserve_error group per
// failure actually present, in the adapter's chain order, followed by the
// catalog's load attributes. Iterating Failures handles the
// absent-candidate case by construction — a single corrupt generation yields
// one group and no phantom .prev entry.
func logUnusableGenerations(stats persist.LoadStats, loadAttrs []any) {
	attrs := make([]any, 0, loadFailureAttrsPerCandidate*len(stats.Failures)+len(loadAttrs))
	for _, failure := range stats.Failures {
		prefix := loadTokenPrev
		if failure.Candidate == port.GenMain {
			prefix = loadTokenMain
		}
		attrs = append(attrs,
			prefix+"_path", failure.Path,
			prefix+"_cause", errorText(failure.Cause),
			prefix+"_preserve_error", errorText(failure.PreserveErr))
	}
	attrs = append(attrs, loadAttrs...)
	slog.Error("No usable dump generation", attrs...)
}

// candidateFailure returns the failure recorded for candidate.
func candidateFailure(failures []port.GenerationFailure, candidate port.Generation) (port.GenerationFailure, bool) {
	for _, failure := range failures {
		if failure.Candidate == candidate {
			return failure, true
		}
	}
	return port.GenerationFailure{}, false
}

// errorText renders err for a log attribute, empty for nil.
func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// wire builds the whole service graph in the plan-05-06 order and launches
// every service goroutine up to (not including) Serve: the Watch hub first
// — persist.Open threads the change tap through storage.Config — then the
// dump store and the write-confirmation engine, the replication adapter
// over the engine as its StateStore, the API server consuming both through
// its seam options — its gate backstop chosen once from the loaded store's
// record count (D-02) — and the gate resolver plus hub goroutines on the API
// context. Returns the wired bundle and the listening gRPC endpoint.
func wire(cfg config.Config) (*wired, net.Listener, error) {
	hub := grpcapi.NewWatchHub()
	engine, engineCancel, engineDone := startEngine(cfg, hub)

	adapter, err := newReplicationAdapter(cfg, engine)
	if err != nil {
		engineCancel()
		return nil, nil, &startupError{
			stage: stageReplicationBind,
			err:   fmt.Errorf("bind replication socket: %w", err),
		}
	}
	// XPORT-05 startup evidence: the EFFECTIVE (kernel-doubled, clamped)
	// buffer sizes, never the requested constants — the adapter read them
	// back in New (RESEARCH Pattern 7).
	logSocketBuffers(adapter)

	replCtx, replCancel := context.WithCancel(context.Background())
	replDone := make(chan struct{})
	go func() {
		defer close(replDone)
		_ = adapter.Run(replCtx)
	}()

	// The empty-vs-data selection (D-02, ADR 0006): the engine finished
	// loading synchronously above, so KeyCount is final.
	srv := grpcapi.NewServer(engine, adapter.NotifyLocalWrite, hub, grpcapi.ServerOptions{
		SyncArrived:    adapter.SyncArrived,
		ListedKey:      adapter.ListedKey,
		WritePeers:     adapter.KnownPeerIDs,
		WaitReplicated: adapter.WaitReplicated,
		ReplayNotify:   adapter.NotifyReplayedWrite,
		StartupGate:    gateBackstopFor(cfg, engine),
		WaitTimeout:    cfg.WaitTimeout,
		OnGateOpen:     logGateOpen,
	})

	apiCtx, apiCancel := context.WithCancel(context.Background())
	go srv.RunGate(apiCtx)
	go func() { _ = hub.Run(apiCtx) }()

	listener, gs, err := listenGRPC(cfg.GRPCAddr, srv, grpcapi.LimitsFor(cfg.ValueMaxBytes))
	if err != nil {
		abortWire(engineCancel, replCancel, apiCancel, nil)
		return nil, nil, &startupError{stage: stageGRPCListen, err: err}
	}

	var metricsSrv *http.Server
	if cfg.MetricsAddr != "" { // D-01: unset means no listener, no goroutine.
		metricsSrv, err = listenMetrics(cfg, metricsSources(adapter, engine))
		if err != nil {
			abortWire(engineCancel, replCancel, apiCancel, listener)
			return nil, nil, &startupError{stage: stageMetricsListen, err: err}
		}
	}

	return &wired{
		engine:       engine,
		engineCancel: engineCancel,
		engineDone:   engineDone,
		replCancel:   replCancel,
		replDone:     replDone,
		apiCancel:    apiCancel,
		api:          srv,
		gs:           gs,
		metricsSrv:   metricsSrv,
	}, listener, nil
}

// logSocketBuffers reports the adapter's EFFECTIVE socket buffer sizes — the
// kernel-doubled, clamped values read back after bind, never the requested
// constants (XPORT-05 startup evidence, RESEARCH Pattern 7).
func logSocketBuffers(adapter *udprepl.Adapter) {
	recvBuf, sendBuf := adapter.SocketBuffers()
	slog.Info("Socket buffers", "recv_buf", recvBuf, "send_buf", sendBuf)
}

// abortWire cancels every context wire() opened — the compensation path when
// a later wiring step fails, with the already-open gRPC listener closed when
// the failure came after it.
func abortWire(engineCancel, replCancel, apiCancel context.CancelFunc, listener net.Listener) {
	engineCancel()
	replCancel()
	apiCancel()
	if listener != nil {
		_ = listener.Close()
	}
}

// newReplicationAdapter builds the UDP replication adapter over the engine
// with the resolved replication knobs.
func newReplicationAdapter(cfg config.Config, engine *persist.Engine) (*udprepl.Adapter, error) {
	return udprepl.New(cfg.BindAddr, cfg.BroadcastAddr, cfg.WireNodeID(), engine, repl.Config{
		HelloInterval:      cfg.HelloInterval,
		PeerTimeout:        cfg.PeerTimeout,
		AckTimeout:         cfg.AckTimeout,
		AckTimeoutMax:      cfg.AckTimeoutMax,
		RetransmitInterval: cfg.RetransmitInterval,
		SyncInterval:       cfg.SyncInterval,
		SyncChunkBytes:     cfg.SyncChunkBytes,
		SyncResponseRate:   cfg.SyncResponseRate,
		MaxClockLead:       cfg.MaxClockLead,
		// The core's prebuilt metadata strings (REPL-08 configuration
		// errors, apply failures) ride as the detail attr under a static
		// capitalized msg — the sloglint static-msg discipline (D-08); a
		// dynamic string never becomes the msg.
		Log: logAdapterWarn,
		// The typed lifecycle seams render once here (D-01): the peer table's
		// transitions and the drift warning the core's shared limiter already
		// admitted.
		OnPeerEvent: logPeerEvent,
		OnDrift:     func(ev repl.DriftEvent) { logDriftEvent(ev, cfg.DriftWarn) },
	})
}

// metricsSources collects the 06-03 seam accessors the exporter snapshots —
// method values over the adapter and engine wire() already holds.
func metricsSources(adapter *udprepl.Adapter, engine *persist.Engine) metrics.Sources {
	return metrics.Sources{
		Discards:      adapter.Discards,
		KnownPeers:    adapter.KnownPeers,
		Unacked:       adapter.UnackedStats,
		Sent:          adapter.SentByType,
		Retransmitted: adapter.Retransmissions,
		MergeStats:    adapter.MergeStats,
		DumpStats:     engine.DumpStats,
		LoadStats:     engine.LoadStats,
		Paced:         adapter.PacedSends,
		SendQueue:     adapter.SendQueue,
		EgressDropped: adapter.EgressDropped,
		ReadErrors:    adapter.ReadErrors,
	}
}

// listenMetrics opens the opt-in metrics/debug listener (D-01/D-02) — called
// only with a non-empty metrics address (wire guards the D-01 opt-in: unset
// means no listener at all). An address that cannot bind is fatal exactly
// like listenGRPC's listen half (the node never runs with its observability
// surface silently missing), and the certificate triple — already validated
// complete by checkMetricsCerts — turns the listener into the mTLS one when
// set, the plaintext loopback/debug listener when not. Serve runs on its own
// goroutine; the lifecycle handle shutdown() needs is the [http.Server].
func listenMetrics(cfg config.Config, sources metrics.Sources) (*http.Server, error) {
	inner, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", cfg.MetricsAddr)
	if err != nil {
		return nil, fmt.Errorf("listen tcp %s: %w", cfg.MetricsAddr, err)
	}
	listener := inner
	if cfg.MetricsCert != "" {
		tlsCfg, tlsErr := metrics.NewTLSConfig(cfg.MetricsCert, cfg.MetricsKey, cfg.MetricsClientCA)
		if tlsErr != nil {
			_ = inner.Close()
			return nil, fmt.Errorf("metrics TLS: %w", tlsErr)
		}
		listener = tls.NewListener(inner, tlsCfg)
	}
	mux := metrics.NewMux(sources)
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	go func() {
		serveErr := srv.Serve(listener)
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			slog.Error("Metrics endpoint", "detail", serveErr.Error())
		}
	}()
	return srv, nil
}

// listenGRPC opens the gRPC TCP listener — distinct from the UDP bind
// address (the Open Q1 pin: localhost is the safe default on the trusted
// segment) — and registers the SupKV service, the standard health server
// (bound 1:1 to the startup gate) and the gRPC server reflection service on
// one grpc.Server. The message-size and keepalive settings are explicit: the
// transport limits come from the effective value limit and the List page
// budget (D-05, APIS-05), so no gRPC default governs a message size or the
// keepalive cadence.
// Reflection lets grpcurl discover the whole API with zero proto files —
// docs/EXAMPLES.md walks the service exactly that way; the schema is public
// in the repo and the default bind is loopback on the trusted segment, so
// always-on reflection adds no exposure worth a flag.
func listenGRPC(grpcAddr string, srv *grpcapi.Server, limits grpcapi.TransportLimits) (net.Listener, *grpc.Server, error) {
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", grpcAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("listen tcp %s: %w", grpcAddr, err)
	}
	gs := grpc.NewServer(limits.ServerOptions()...)
	apiproto.RegisterSupKVServiceServer(gs, srv)
	healthpb.RegisterHealthServer(gs, srv.Health())
	reflection.Register(gs)
	return listener, gs, nil
}

// armSignals takes ownership of SIGTERM/SIGINT for the whole run — serving
// and the D-19 shutdown alike (P1-IN-01). The buffered channel holds the
// first signal for serve and a possible second for shutdown's force path,
// and nothing restores the default disposition until run stops the channel
// after the final dump, so no repeated signal can kill the process
// mid-shutdown.
func armSignals() chan os.Signal {
	sig := make(chan os.Signal, signalBuffer)
	signal.Notify(sig, syscall.SIGTERM, os.Interrupt)
	return sig
}

// serve launches gs.Serve on the listener and blocks until the first
// SIGTERM or SIGINT arrives on sig (armSignals owns the channel; ownership
// deliberately outlives serve — releasing it here would re-arm the default
// disposition and let a second signal kill the D-19 shutdown with no final
// dump, the P1-IN-01 regression). A Serve failure before any signal is an
// error return: the node cannot do its one job, and run decides the verdict.
func serve(node *wired, listener net.Listener, sig <-chan os.Signal) error {
	serveErr := make(chan error, 1)
	go func() { serveErr <- node.gs.Serve(listener) }()
	select {
	case <-sig:
		return nil
	case err := <-serveErr:
		return err
	}
}

// shutdown runs the pinned graceful-shutdown sequence (D-19 — the order is
// the contract, RESEARCH Pattern 6): close the health endpoint one-way
// through the API server's Shutdown, shut the opt-in metrics/debug listener
// down — positioned right after the health close so scrapers see it the
// moment the node stops serving, while the rest of the drain stays
// observable through logs (Pitfall 7) — cancel the API context to end Watch
// streams and the hub (GracefulStop blocks forever while any stream is open),
// bound GracefulStop with the grace period and fall back to Stop, stop
// replication and wait for the adapter goroutine, Flush — the final
// write-confirmed dump covering remote-merged dirtiness — and only then
// cancel the engine context, whose exit sweep now finds no waiters. A final
// dump failure is the returned error: a clean exit is reserved for a
// confirmed shutdown.
//
// The health close is the standard health server's Shutdown, never a direct
// SetServingStatus (D-05, APIS-05, T-06-23): it is one-way, so the gate
// resolver's backstop firing after this point cannot flip the node back to
// SERVING while it drains (RESEARCH Pitfall 9). apiCancel follows it, and
// neither path calls health.Resume.
//
// Both remaining waits are bounded by the same shutdownGrace (PERSIST-08,
// T-5-24): the final flush is FlushTimeout, so a stuck fsync answers
// ErrUnconfirmed instead of hanging the process, and the engineDone wait is
// a select against a grace timer — a dumper still stuck in the kernel at
// process exit is abandoned with one WARN, which is the deliberate choice
// over an unkillable shutdown. A flush error returns the failed-final-dump
// error and run maps it to exit code 3: a clean exit still means the final
// dump was confirmed.
//
// A second SIGTERM/SIGINT observed on sig while GracefulStop drains is an
// operator's hurry-up: it takes the same force path as the grace deadline
// (Stop) immediately. The final Flush sits after GracefulStop in the
// sequence and therefore survives the force stop — the confirmed dump still
// lands, and a clean shutdown still exits 0.
func shutdown(node *wired, sig <-chan os.Signal) error {
	node.api.Shutdown()
	if node.metricsSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
		err := node.metricsSrv.Shutdown(ctx)
		cancel()
		if err != nil {
			log.Printf("sup-kv: metrics listener shutdown: %v", err)
		}
	}
	node.apiCancel()

	stopped := make(chan struct{})
	go func() {
		node.gs.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-sig:
		log.Print("sup-kv: second shutdown signal — forcing stop")
		node.gs.Stop()
		<-stopped
	case <-time.After(shutdownGrace):
		log.Printf("sup-kv: graceful stop exceeded %v — forcing stop", shutdownGrace)
		node.gs.Stop()
		<-stopped
	}

	node.replCancel()
	<-node.replDone

	err := node.engine.FlushTimeout(shutdownGrace)
	if err != nil {
		return fmt.Errorf("final dump failed: %w", err)
	}

	node.engineCancel()
	awaitEngineStop(node.engineDone, shutdownGrace)
	return nil
}

// awaitEngineStop waits up to grace for the engine's Run to return. A dumper
// that outlives the grace — most likely a kernel fsync that never returned —
// is abandoned: the process is exiting, and waiting here would defeat the
// bounded shutdown (PERSIST-08). A Run error other than cancellation is
// reported; a clean cancellation is silent.
func awaitEngineStop(done <-chan error, grace time.Duration) {
	select {
	case runErr := <-done:
		if !errors.Is(runErr, context.Canceled) {
			log.Printf("sup-kv: engine Run returned %v", runErr)
		}
	case <-time.After(grace):
		slog.Warn("Engine did not stop within the shutdown grace", "grace", grace.String())
	}
}

// printVersion reports the ldflags-injected build identity (D-18): version,
// commit and the Go runtime version, "dev" standing in for unstamped
// fields.
func printVersion() {
	fmt.Printf("sup-kv %s %s %s\n", version, commit, goruntime.Version())
}
