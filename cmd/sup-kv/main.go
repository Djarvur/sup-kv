// Command sup-kv is a leaderless replicated in-memory key-value store for
// small clusters on one trusted LAN broadcast segment.
//
// Per the layout contract, main() holds flags, environment, signals and
// wiring only — every behavior (write confirmation, replication, watch,
// snapshots) lives in the packages being wired. Configuration resolves
// once: each knob is a flag whose default is its SUPKV_ env twin, so the
// stdlib flag parser delivers flag > env > default (D-16/D-17). SIGTERM
// runs the pinned graceful-shutdown order (D-19) whose final step is the
// engine's write-confirmed dump.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/grpcapi"
	apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
	"github.com/Djarvur/sup-kv/internal/persistfile"
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

// Configuration-error sentinels the startup path wraps with the offending
// detail; callers match them with [errors.Is], never identity comparison.
var (
	errNodeIDRange     = errors.New("node-id exceeds the 16-bit node ID space")
	errMissingRequired = errors.New("missing required knobs")
	errNodeIDEnv       = errors.New("must be an integer within the 16-bit node ID space")
)

// Process exit codes: a configuration error (missing or malformed knob)
// exits with exitUsage; a failed final dump during shutdown exits with
// exitDumpFailed — a clean exit 0 is reserved for a shutdown whose final
// write-confirmed dump completed (D-19).
const (
	exitDumpFailed = 1
	exitUsage      = 2
)

// shutdownGrace bounds GracefulStop during the D-19 shutdown: past it,
// gs.Stop() force-closes whatever is still open (T-05-10 — a hang here
// would leave committed data only in memory when the supervisor SIGKILLs).
const shutdownGrace = 10 * time.Second

// The SUPKV_ env twins of the flags (D-16, the published operational
// contract): one per knob, resolved as the flag's default value.
const (
	envNodeID             = "SUPKV_NODE_ID"
	envBindAddr           = "SUPKV_BIND_ADDR"
	envBroadcastAddr      = "SUPKV_BROADCAST_ADDR"
	envDumpPath           = "SUPKV_DUMP_PATH"
	envGRPCAddr           = "SUPKV_GRPC_ADDR"
	envStartupGate        = "SUPKV_STARTUP_GATE"
	envWaitTimeout        = "SUPKV_WAIT_TIMEOUT"
	envValueMaxBytes      = "SUPKV_VALUE_MAX_BYTES"
	envSyncChunkBytes     = "SUPKV_SYNC_CHUNK_BYTES"
	envHelloInterval      = "SUPKV_HELLO_INTERVAL"
	envPeerTimeout        = "SUPKV_PEER_TIMEOUT"
	envAckTimeout         = "SUPKV_ACK_TIMEOUT"
	envAckTimeoutMax      = "SUPKV_ACK_TIMEOUT_MAX"
	envRetransmitInterval = "SUPKV_RETRANSMIT_INTERVAL"
	envSyncInterval       = "SUPKV_SYNC_INTERVAL"
	envSyncResponseRate   = "SUPKV_SYNC_RESPONSE_RATE"
	envDumpDebounce       = "SUPKV_DUMP_DEBOUNCE"
	envDumpMaxWait        = "SUPKV_DUMP_MAX_WAIT"
	envDriftWarn          = "SUPKV_DRIFT_WARN"
)

// The design defaults (docs/IDEA.md §Parameters, plus the two API-layer
// knobs: the Open Q1 pin's distinct gRPC listener and the A2 wait default).
const (
	defaultGRPCAddr       = "127.0.0.1:7901"
	defaultStartupGate    = time.Second
	defaultWaitTimeout    = time.Second
	defaultValueMaxBytes  = 61440
	defaultSyncChunkBytes = 1400
	defaultHelloInterval  = time.Second
	defaultPeerTimeout    = 5 * time.Second
	defaultAckTimeout     = 200 * time.Millisecond
	defaultAckTimeoutMax  = 5 * time.Second
	defaultRetransmit     = 100 * time.Millisecond
	defaultSyncInterval   = 30 * time.Second
	defaultSyncRespRate   = 500
	defaultDumpDebounce   = 50 * time.Millisecond
	defaultDumpMaxWait    = time.Second
	defaultDriftWarn      = 60 * time.Second
)

// config is the resolved launch configuration, built once at startup.
type config struct {
	nodeID             uint16
	bindAddr           string
	broadcastAddr      string
	dumpPath           string
	grpcAddr           string
	startupGate        time.Duration
	waitTimeout        time.Duration
	valueMaxBytes      int
	syncChunkBytes     int
	helloInterval      time.Duration
	peerTimeout        time.Duration
	ackTimeout         time.Duration
	ackTimeoutMax      time.Duration
	retransmitInterval time.Duration
	syncInterval       time.Duration
	syncResponseRate   int
	dumpDebounce       time.Duration
	dumpMaxWait        time.Duration
	driftWarn          time.Duration
}

// wired bundles the running services with their lifecycle handles — the
// plumbing main's shutdown path needs, no behavior of its own.
type wired struct {
	engine       *persist.Engine
	engineCancel context.CancelFunc
	engineDone   <-chan error
	replCancel   context.CancelFunc
	replDone     <-chan struct{}
	apiCancel    context.CancelFunc
	health       *health.Server
	gs           *grpc.Server
}

func main() {
	var cfg config
	nodeID := new(uint64)
	showVersion := new(bool)
	envErr := registerConfig(&cfg, nodeID, showVersion)
	flag.Parse()

	if *showVersion {
		printVersion()
		return
	}
	err := finalizeConfig(&cfg, *nodeID, envErr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sup-kv: %s\n", err)
		flag.Usage()
		os.Exit(exitUsage)
	}

	node, listener, err := wire(cfg)
	if err != nil {
		log.Fatalf("sup-kv: %v", err)
	}
	err = serve(node, listener) // blocks until SIGTERM/SIGINT
	if err != nil {
		log.Fatalf("sup-kv: grpc serve exited before the shutdown signal: %v", err)
	}
	err = shutdown(node) // D-19: the order is the contract
	if err != nil {
		log.Printf("sup-kv: %v", err)
		os.Exit(exitDumpFailed)
	}
	os.Exit(0)
}

// registerConfig declares every knob as a flag whose default is its SUPKV_
// env twin (or the design default) — an explicit flag overrides the
// environment and the environment overrides the default (D-17). Parsing
// stays with the caller (main owns every exit per the deep-exit rule): a
// malformed env value is collected as the returned error, main aborts with
// usage after Parse, and --version keeps working even on a broken
// environment.
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

	id, err := envUint16(envNodeID, 0)
	fail(envNodeID, err)
	flag.Uint64Var(nodeID, "node-id", uint64(id), "required unique 16-bit node ID (env SUPKV_NODE_ID)")
	flag.StringVar(&cfg.bindAddr, "bind-addr", envOrDefault(envBindAddr, ""),
		"required local UDP bind address, e.g. 0.0.0.0:7900 (env SUPKV_BIND_ADDR)")
	flag.StringVar(&cfg.broadcastAddr, "broadcast-addr", envOrDefault(envBroadcastAddr, ""),
		"required segment broadcast address, e.g. 127.255.255.255:7900 (env SUPKV_BROADCAST_ADDR)")
	flag.StringVar(&cfg.dumpPath, "dump-path", envOrDefault(envDumpPath, ""),
		"required dump file path (env SUPKV_DUMP_PATH)")
	flag.StringVar(&cfg.grpcAddr, "grpc-addr", envOrDefault(envGRPCAddr, defaultGRPCAddr),
		"gRPC listen address, distinct from the UDP bind (env SUPKV_GRPC_ADDR)")
	flag.DurationVar(&cfg.startupGate, "startup-gate", envDur(envStartupGate, defaultStartupGate),
		"gate backstop: writable this long after start even without a sync round (env SUPKV_STARTUP_GATE)")
	flag.DurationVar(&cfg.waitTimeout, "wait-timeout", envDur(envWaitTimeout, defaultWaitTimeout),
		"default wait_for_replication bound (env SUPKV_WAIT_TIMEOUT)")
	flag.IntVar(&cfg.valueMaxBytes, "value-max-bytes", envNum(envValueMaxBytes, defaultValueMaxBytes),
		"maximum value size in bytes (env SUPKV_VALUE_MAX_BYTES)")
	flag.IntVar(&cfg.syncChunkBytes, "sync-chunk-bytes", envNum(envSyncChunkBytes, defaultSyncChunkBytes),
		"DataSync chunk byte budget (env SUPKV_SYNC_CHUNK_BYTES)")
	flag.DurationVar(&cfg.helloInterval, "hello-interval", envDur(envHelloInterval, defaultHelloInterval),
		"discovery broadcast period (env SUPKV_HELLO_INTERVAL)")
	flag.DurationVar(&cfg.peerTimeout, "peer-timeout", envDur(envPeerTimeout, defaultPeerTimeout),
		"peer eviction silence bound (env SUPKV_PEER_TIMEOUT)")
	flag.DurationVar(&cfg.ackTimeout, "ack-timeout", envDur(envAckTimeout, defaultAckTimeout),
		"retransmit ladder base (env SUPKV_ACK_TIMEOUT)")
	flag.DurationVar(&cfg.ackTimeoutMax, "ack-timeout-max", envDur(envAckTimeoutMax, defaultAckTimeoutMax),
		"retransmit ladder ceiling (env SUPKV_ACK_TIMEOUT_MAX)")
	flag.DurationVar(&cfg.retransmitInterval, "retransmit-interval", envDur(envRetransmitInterval, defaultRetransmit),
		"retransmit scan period (env SUPKV_RETRANSMIT_INTERVAL)")
	flag.DurationVar(&cfg.syncInterval, "sync-interval", envDur(envSyncInterval, defaultSyncInterval),
		"proactive sync listing period (env SUPKV_SYNC_INTERVAL)")
	flag.IntVar(&cfg.syncResponseRate, "sync-response-rate", envNum(envSyncResponseRate, defaultSyncRespRate),
		"sync responses per second cap (env SUPKV_SYNC_RESPONSE_RATE)")
	flag.DurationVar(&cfg.dumpDebounce, "dump-debounce", envDur(envDumpDebounce, defaultDumpDebounce),
		"dump quiet window (env SUPKV_DUMP_DEBOUNCE)")
	flag.DurationVar(&cfg.dumpMaxWait, "dump-max-wait", envDur(envDumpMaxWait, defaultDumpMaxWait),
		"dump start bound from the oldest unconfirmed write (env SUPKV_DUMP_MAX_WAIT)")
	flag.DurationVar(&cfg.driftWarn, "drift-warn", envDur(envDriftWarn, defaultDriftWarn),
		"clock drift warning threshold (env SUPKV_DRIFT_WARN)")
	return errors.Join(errs...)
}

// finalizeConfig validates what only the parsed flags can decide — the
// node-id range and the required knobs — and applies the resolved node id.
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
	return nil
}

// startEngine opens the real dump store at the configured path, resolves
// the engine over it with the hub's change tap threaded through
// storage.Config, and launches the single dumper goroutine. Returns the
// engine with its lifecycle handles.
func startEngine(cfg config, hub *grpcapi.WatchHub) (*persist.Engine, context.CancelFunc, <-chan error) {
	dumps := persistfile.New(cfg.dumpPath)
	engine, discarded := persist.Open(cfg.nodeID, time.Now, cfg.driftWarn, dumps, storage.Config{
		ValueMaxBytes: cfg.valueMaxBytes,
		OnChange:      hub.Ingest,
	}, persist.Config{Debounce: cfg.dumpDebounce, MaxWait: cfg.dumpMaxWait})
	if discarded > 0 {
		log.Printf("sup-kv: discarded %d undecodable dumped records at startup", discarded)
	}
	engineCtx, engineCancel := context.WithCancel(context.Background())
	engineDone := make(chan error, 1)
	go func() { engineDone <- engine.Run(engineCtx) }()
	return engine, engineCancel, engineDone
}

// wire builds the whole service graph in the plan-05-06 order and launches
// every service goroutine up to (not including) Serve: the Watch hub first
// — persist.Open threads the change tap through storage.Config — then the
// dump store and the write-confirmation engine, the replication adapter
// over the engine as its StateStore, the API server consuming both through
// its seam options, and the gate resolver plus hub goroutines on the API
// context. Returns the wired bundle and the listening gRPC endpoint.
func wire(cfg config) (*wired, net.Listener, error) {
	hub := grpcapi.NewWatchHub()
	engine, engineCancel, engineDone := startEngine(cfg, hub)

	adapter, err := udprepl.New(cfg.bindAddr, cfg.broadcastAddr, cfg.nodeID, engine, repl.Config{
		HelloInterval:      cfg.helloInterval,
		PeerTimeout:        cfg.peerTimeout,
		AckTimeout:         cfg.ackTimeout,
		AckTimeoutMax:      cfg.ackTimeoutMax,
		RetransmitInterval: cfg.retransmitInterval,
		SyncInterval:       cfg.syncInterval,
		SyncChunkBytes:     cfg.syncChunkBytes,
		SyncResponseRate:   cfg.syncResponseRate,
		Log:                func(msg string) { log.Printf("%s", msg) },
	})
	if err != nil {
		engineCancel()
		return nil, nil, fmt.Errorf("bind replication socket: %w", err)
	}

	replCtx, replCancel := context.WithCancel(context.Background())
	replDone := make(chan struct{})
	go func() {
		defer close(replDone)
		_ = adapter.Run(replCtx)
	}()

	srv := grpcapi.NewServer(engine, adapter.NotifyLocalWrite, hub, grpcapi.ServerOptions{
		SyncArrived: adapter.SyncArrived,
		KnownPeers:  adapter.KnownPeers,
		PendingCopy: adapter.PendingCopy,
		StartupGate: cfg.startupGate,
		WaitTimeout: cfg.waitTimeout,
	})

	apiCtx, apiCancel := context.WithCancel(context.Background())
	go srv.RunGate(apiCtx)
	go func() { _ = hub.Run(apiCtx) }()

	listener, gs, err := listenGRPC(cfg.grpcAddr, srv)
	if err != nil {
		engineCancel()
		replCancel()
		apiCancel()
		return nil, nil, err
	}

	return &wired{
		engine:       engine,
		engineCancel: engineCancel,
		engineDone:   engineDone,
		replCancel:   replCancel,
		replDone:     replDone,
		apiCancel:    apiCancel,
		health:       srv.Health(),
		gs:           gs,
	}, listener, nil
}

// listenGRPC opens the gRPC TCP listener — distinct from the UDP bind
// address (the Open Q1 pin: localhost is the safe default on the trusted
// segment) — and registers the SupKV service plus the standard health
// server (bound 1:1 to the startup gate) on one grpc.Server.
func listenGRPC(grpcAddr string, srv *grpcapi.Server) (net.Listener, *grpc.Server, error) {
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", grpcAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("listen tcp %s: %w", grpcAddr, err)
	}
	gs := grpc.NewServer()
	apiproto.RegisterSupKVServiceServer(gs, srv)
	healthpb.RegisterHealthServer(gs, srv.Health())
	return listener, gs, nil
}

// serve launches gs.Serve on the listener and blocks until the first
// SIGTERM or SIGINT. A Serve failure before any signal is an error return:
// the node cannot do its one job, and main decides the verdict.
func serve(node *wired, listener net.Listener) error {
	serveErr := make(chan error, 1)
	go func() { serveErr <- node.gs.Serve(listener) }()
	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()
	select {
	case <-signalCtx.Done():
		return nil
	case err := <-serveErr:
		return err
	}
}

// shutdown runs the pinned graceful-shutdown sequence (D-19 — the order is
// the contract, RESEARCH Pattern 6): shed client traffic through the health
// endpoint, cancel the API context to end Watch streams and the hub
// (GracefulStop blocks forever while any stream is open), bound
// GracefulStop with the grace period and fall back to Stop, stop
// replication and wait for the adapter goroutine, Flush — the final
// write-confirmed dump covering remote-merged dirtiness — and only then
// cancel the engine context, whose exit sweep now finds no waiters. A final
// dump failure is the returned error: a clean exit is reserved for a
// confirmed shutdown.
func shutdown(node *wired) error {
	node.health.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	node.apiCancel()

	stopped := make(chan struct{})
	go func() {
		node.gs.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(shutdownGrace):
		log.Printf("sup-kv: graceful stop exceeded %v — forcing stop", shutdownGrace)
		node.gs.Stop()
		<-stopped
	}

	node.replCancel()
	<-node.replDone

	err := node.engine.Flush()
	if err != nil {
		return fmt.Errorf("final dump failed: %w", err)
	}

	node.engineCancel()
	runErr := <-node.engineDone
	if !errors.Is(runErr, context.Canceled) {
		log.Printf("sup-kv: engine Run returned %v", runErr)
	}
	return nil
}

// missingRequired names every required knob left at its zero default — its
// SUPKV_ twin unset as well; node-id's zero is not a usable identity.
func missingRequired(cfg config) []string {
	var missing []string
	if cfg.nodeID == 0 {
		missing = append(missing, "--node-id")
	}
	if cfg.bindAddr == "" {
		missing = append(missing, "--bind-addr")
	}
	if cfg.broadcastAddr == "" {
		missing = append(missing, "--broadcast-addr")
	}
	if cfg.dumpPath == "" {
		missing = append(missing, "--dump-path")
	}
	return missing
}

// printVersion reports the ldflags-injected build identity (D-18): version,
// commit and the Go runtime version, "dev" standing in for unstamped
// fields.
func printVersion() {
	fmt.Printf("sup-kv %s %s %s\n", version, commit, runtime.Version())
}

// envOrDefault returns the SUPKV_ env value when set (empty counts as
// unset) or def — the flag default, so the stdlib flag parser delivers
// flag > env > default with zero extra machinery (D-16/D-17).
func envOrDefault(env, def string) string {
	if v, ok := os.LookupEnv(env); ok && v != "" {
		return v
	}
	return def
}

// envUint16 resolves an env knob that must parse as a 16-bit unsigned
// integer — the wire's node_id space. A malformed value is an error, never
// a silently-swallowed default.
func envUint16(env string, def uint16) (uint16, error) {
	raw := envOrDefault(env, "")
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 || v > math.MaxUint16 {
		return def, errNodeIDEnv
	}
	return uint16(v), nil
}

// envInt resolves an env knob that must parse as an int. A malformed value
// is an error, never a silently-swallowed default.
func envInt(env string, def int) (int, error) {
	raw := envOrDefault(env, "")
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def, fmt.Errorf("must be an integer: %w", err)
	}
	return v, nil
}

// envDuration resolves an env knob that must parse as a Go duration. A
// malformed value is an error, never a silently-swallowed default.
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
