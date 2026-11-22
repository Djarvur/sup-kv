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
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"math"
	"net"
	"net/http"
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
	"google.golang.org/grpc/reflection"

	apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
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

// Configuration-error sentinels the startup path wraps with the offending
// detail; callers match them with [errors.Is], never identity comparison.
var (
	errNodeIDRange        = errors.New("node-id exceeds the 16-bit node ID space")
	errMissingRequired    = errors.New("missing required knobs")
	errNodeIDEnv          = errors.New("must be an integer within the 16-bit node ID space")
	errLogFormatEnv       = errors.New("must be json or text")
	errIncompleteCertSet  = errors.New("the metrics certificate envs must be set together")
	errDatagramCeiling    = errors.New("value-max-bytes exceeds the datagram ceiling")
	errValueMaxBytesRange = errors.New("value-max-bytes must be positive")
	errSyncChunkBytes     = errors.New("sync-chunk-bytes exceeds the datagram ceiling")
	errStartupGateRange   = errors.New("startup gate must be positive")
)

// Process exit codes: a configuration error (missing or malformed knob)
// exits with exitUsage; a failed final dump during shutdown exits with
// exitDumpFailed — a clean exit 0 is reserved for a shutdown whose final
// write-confirmed dump completed (D-19).
const (
	exitDumpFailed = 1
	exitUsage      = 2
)

// loadFailureAttrsPerCandidate is the number of attributes one candidate
// load failure contributes: its path, its cause and its preservation error.
const loadFailureAttrsPerCandidate = 3

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

// The SUPKV_ env twins of the flags (D-16, the published operational
// contract): one per knob, resolved as the flag's default value.
const (
	envNodeID             = "SUPKV_NODE_ID"
	envBindAddr           = "SUPKV_BIND_ADDR"
	envBroadcastAddr      = "SUPKV_BROADCAST_ADDR"
	envDumpPath           = "SUPKV_DUMP_PATH"
	envGRPCAddr           = "SUPKV_GRPC_ADDR"
	envStartupGate        = "SUPKV_STARTUP_GATE"
	envStartupGateEmpty   = "SUPKV_STARTUP_GATE_EMPTY"
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
	envMaxClockLead       = "SUPKV_MAX_CLOCK_LEAD"
	envDumpDebounce       = "SUPKV_DUMP_DEBOUNCE"
	envDumpMaxWait        = "SUPKV_DUMP_MAX_WAIT"
	envDriftWarn          = "SUPKV_DRIFT_WARN"

	// The structured-logging env knobs (D-06/D-07) are env-only — no flag
	// twins: they configure the process's own output format, not its
	// service behavior.
	envLogLevel  = "SUPKV_LOG_LEVEL"
	envLogFormat = "SUPKV_LOG_FORMAT"

	// The metrics/debug endpoint knobs (D-01/D-02): the listen address has
	// a flag twin with NO default — unset means no listener at all; the
	// certificate triple is env-only and all-or-nothing.
	envMetricsAddr     = "SUPKV_METRICS_ADDR"
	envMetricsCert     = "SUPKV_METRICS_CERT"
	envMetricsKey      = "SUPKV_METRICS_KEY"
	envMetricsClientCA = "SUPKV_METRICS_CLIENT_CA"
)

// The design defaults (docs/IDEA.md §Parameters, plus the two API-layer
// knobs: the Open Q1 pin's distinct gRPC listener and the A2 wait default).
const (
	defaultGRPCAddr    = "127.0.0.1:7901"
	defaultStartupGate = time.Second
	defaultWaitTimeout = time.Second

	// defaultStartupGateEmpty is the empty-disk node's backstop (D-02,
	// ADR 0006): a node that loaded no records has nothing local to answer
	// with, so it waits longer for a peer listing than a node with data.
	defaultStartupGateEmpty = 60 * time.Second
	defaultSyncChunkBytes   = 1400
	defaultHelloInterval    = time.Second
	defaultPeerTimeout      = 5 * time.Second
	defaultAckTimeout       = 200 * time.Millisecond
	defaultAckTimeoutMax    = 5 * time.Second
	defaultRetransmit       = 100 * time.Millisecond
	defaultSyncInterval     = 30 * time.Second
	defaultSyncRespRate     = 500
	// defaultMaxClockLead is the inbound version bound's launch default
	// (XPORT-04): 60 s, the documented drift tolerance. The core normalizes
	// a zero or negative value to the same bound; the flag default only
	// makes the documented value visible in --help.
	defaultMaxClockLead = time.Minute
	defaultDumpDebounce = 50 * time.Millisecond
	defaultDumpMaxWait  = time.Second
	defaultDriftWarn    = 60 * time.Second

	// The structured-logging defaults (D-06/D-07): JSON records on stderr,
	// info level.
	defaultLogLevel  = slog.LevelInfo
	defaultLogFormat = logFormatJSON
)

// The accepted SUPKV_LOG_FORMAT values (D-06).
const (
	logFormatJSON = "json"
	logFormatText = "text"
)

// config is the resolved launch configuration, built once at startup.
type config struct {
	nodeID             uint16
	bindAddr           string
	broadcastAddr      string
	dumpPath           string
	grpcAddr           string
	startupGate        time.Duration
	startupGateEmpty   time.Duration
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
	maxClockLead       time.Duration
	dumpDebounce       time.Duration
	dumpMaxWait        time.Duration
	driftWarn          time.Duration
	logLevel           slog.Level
	logFormat          string
	metricsAddr        string
	metricsCert        string
	metricsKey         string
	metricsClientCA    string
}

// wired bundles the running services with their lifecycle handles — the
// plumbing main's shutdown path needs, no behavior of its own. metricsSrv is
// nil when --metrics-addr is unset: the endpoint is opt-in with no listener
// at all (D-01).
type wired struct {
	engine       *persist.Engine
	engineCancel context.CancelFunc
	engineDone   <-chan error
	replCancel   context.CancelFunc
	replDone     <-chan struct{}
	apiCancel    context.CancelFunc
	health       *health.Server
	gs           *grpc.Server
	metricsSrv   *http.Server
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

	// The one slog handler install (D-06/D-07) precedes every service
	// goroutine: the udprepl Log tap, the adapter's discard events and the
	// log-package calls redirected through slog.SetDefault all flow through
	// this handler for the process's whole life.
	setupLogging(cfg)

	// XPORT-09: a non-wildcard bind is loud before any service starts — the
	// node would send broadcasts but never receive them.
	warnOnNonWildcardBind(cfg.bindAddr)

	node, listener, err := wire(cfg)
	if err != nil {
		log.Fatalf("sup-kv: %v", err)
	}
	// Signal ownership spans serving AND the D-19 shutdown (P1-IN-01): the
	// armed channel is released only after shutdown completes, so a second
	// SIGTERM during the graceful drain is observed as an operator's
	// hurry-up — it takes the force-stop path — instead of killing the
	// process with the default disposition and forfeiting the final dump.
	sig := armSignals()
	err = serve(node, listener, sig)
	if err != nil {
		log.Fatalf("sup-kv: grpc serve exited before the shutdown signal: %v", err)
	}
	err = shutdown(node, sig) // D-19: the order is the contract
	signal.Stop(sig)
	if err != nil {
		log.Printf("sup-kv: %v", err)
		os.Exit(exitDumpFailed)
	}
	os.Exit(0)
}

// setupLogging installs the process's one slog handler on stderr (D-06,
// D-07): JSON records by default, key=value text when SUPKV_LOG_FORMAT=text,
// at the level SUPKV_LOG_LEVEL resolved (debug through error). Called once
// from main before wire launches any goroutine. [slog.SetDefault] also
// redirects the log package's default logger, so the remaining direct
// [log.Printf] calls surface as level=INFO records in the same format —
// every stderr line the running node emits is one handler-formatted record.
func setupLogging(cfg config) {
	opts := &slog.HandlerOptions{Level: cfg.logLevel}
	var h slog.Handler = slog.NewJSONHandler(os.Stderr, opts)
	if cfg.logFormat == logFormatText {
		h = slog.NewTextHandler(os.Stderr, opts)
	}
	slog.SetDefault(slog.New(h))
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
	flag.StringVar(&cfg.metricsAddr, "metrics-addr", envOrDefault(envMetricsAddr, ""),
		"optional metrics/debug listen address — unset means no listener (env SUPKV_METRICS_ADDR)")
	flag.DurationVar(&cfg.startupGate, "startup-gate", envDur(envStartupGate, defaultStartupGate),
		"gate backstop: writable this long after start even without a sync round (env SUPKV_STARTUP_GATE)")
	flag.DurationVar(&cfg.startupGateEmpty, "startup-gate-empty", envDur(envStartupGateEmpty, defaultStartupGateEmpty),
		"empty-disk gate backstop: a node with no loaded records waits this long for a peer listing (env SUPKV_STARTUP_GATE_EMPTY)")
	flag.DurationVar(&cfg.waitTimeout, "wait-timeout", envDur(envWaitTimeout, defaultWaitTimeout),
		"default wait_for_replication bound (env SUPKV_WAIT_TIMEOUT)")
	flag.IntVar(&cfg.valueMaxBytes, "value-max-bytes", envNum(envValueMaxBytes, storage.ValueMaxBytes),
		"maximum value size in bytes (env SUPKV_VALUE_MAX_BYTES)")
	flag.IntVar(&cfg.syncChunkBytes, "sync-chunk-bytes", envNum(envSyncChunkBytes, defaultSyncChunkBytes),
		"DataSync chunk byte budget (env SUPKV_SYNC_CHUNK_BYTES)")
	flag.IntVar(&cfg.syncResponseRate, "sync-response-rate", envNum(envSyncResponseRate, defaultSyncRespRate),
		"sync responses per second cap (env SUPKV_SYNC_RESPONSE_RATE)")
	registerDurationFlags(cfg, envDur)

	resolveLogKnobs(cfg, &errs)
	resolveMetricsCerts(cfg)
	return errors.Join(errs...)
}

// registerDurationFlags declares the replication-timing and dump-cycle knobs
// — every duration flag except startup-gate and wait-timeout, which sit with
// the API-facing registrations. The envDur closure is registerConfig's
// collected-error resolution, shared so a malformed SUPKV_ duration twin
// aborts startup through the same path regardless of which helper declared
// the flag.
func registerDurationFlags(cfg *config, envDur func(env string, def time.Duration) time.Duration) {
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
	flag.DurationVar(&cfg.maxClockLead, "max-clock-lead", envDur(envMaxClockLead, defaultMaxClockLead),
		"reject received versions beyond now + this bound (env SUPKV_MAX_CLOCK_LEAD)")
	flag.DurationVar(&cfg.dumpDebounce, "dump-debounce", envDur(envDumpDebounce, defaultDumpDebounce),
		"dump quiet window (env SUPKV_DUMP_DEBOUNCE)")
	flag.DurationVar(&cfg.dumpMaxWait, "dump-max-wait", envDur(envDumpMaxWait, defaultDumpMaxWait),
		"dump start bound from the oldest unconfirmed write (env SUPKV_DUMP_MAX_WAIT)")
	flag.DurationVar(&cfg.driftWarn, "drift-warn", envDur(envDriftWarn, defaultDriftWarn),
		"clock drift warning threshold (env SUPKV_DRIFT_WARN)")
}

// resolveMetricsCerts reads the env-only metrics certificate triple (D-02,
// the 06-RESEARCH A4 env-only reading) into cfg — file paths, so no
// malformed-value error exists here; the all-or-nothing check runs in
// finalizeConfig, before any bind (Pitfall 8).
func resolveMetricsCerts(cfg *config) {
	cfg.metricsCert = envOrDefault(envMetricsCert, "")
	cfg.metricsKey = envOrDefault(envMetricsKey, "")
	cfg.metricsClientCA = envOrDefault(envMetricsClientCA, "")
}

// resolveLogKnobs resolves the env-only structured-logging knobs (D-06/D-07)
// into cfg, appending each malformed value to errs under its env name — the
// same collected env-error contract registerConfig's fail() applies to the
// flag twins: a malformed SUPKV_LOG_* value aborts startup with exitUsage,
// never a silently-swallowed default.
func resolveLogKnobs(cfg *config, errs *[]error) {
	level, levelErr := envSlogLevel(envLogLevel, defaultLogLevel)
	if levelErr != nil {
		*errs = append(*errs, fmt.Errorf("%s: %w", envLogLevel, levelErr))
	}
	cfg.logLevel = level
	format, formatErr := envLogValue(envLogFormat, defaultLogFormat)
	if formatErr != nil {
		*errs = append(*errs, fmt.Errorf("%s: %w", envLogFormat, formatErr))
	}
	cfg.logFormat = format
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
	err := checkMetricsCerts(*cfg)
	if err != nil {
		return err
	}
	err = checkStartupGates(*cfg)
	if err != nil {
		return err
	}
	err = checkValueMaxBytes(*cfg)
	if err != nil {
		return err
	}
	err = checkDatagramCeiling(*cfg)
	if err != nil {
		return err
	}
	return checkSyncChunkBytes(*cfg)
}

// checkSyncChunkBytes rejects an explicit --sync-chunk-bytes above the UDP
// datagram ceiling before any bind (WR-06): a chunk budget above 65 507
// describes a chunk no datagram can carry, and the core's defensive clamp
// would silently ship smaller chunks than the operator configured. The
// ceiling is the core's own constant (repl.MaxSyncChunkBytes, pinned to the
// adapter's datagram ceiling by a cross-seam test). A zero or negative value
// stays legal — the core normalizes it to the design default.
func checkSyncChunkBytes(cfg config) error {
	if cfg.syncChunkBytes > repl.MaxSyncChunkBytes {
		return fmt.Errorf("%w: --sync-chunk-bytes=%d (at most %d, the largest IPv4 UDP payload)",
			errSyncChunkBytes, cfg.syncChunkBytes, repl.MaxSyncChunkBytes)
	}
	return nil
}

// checkMetricsCerts enforces D-02's all-or-nothing contract before any bind
// (Pitfall 8): any of the three certificate envs set without all three is a
// startup error naming every env of the triple — no code path can reach the
// TLS builder with a half-loaded set, and with none set the listener serves
// plaintext (the loopback/debug mode).
func checkMetricsCerts(cfg config) error {
	noneSet := cfg.metricsCert == "" && cfg.metricsKey == "" && cfg.metricsClientCA == ""
	allSet := cfg.metricsCert != "" && cfg.metricsKey != "" && cfg.metricsClientCA != ""
	if noneSet || allSet {
		return nil
	}
	return fmt.Errorf("%w: %s, %s and %s",
		errIncompleteCertSet, envMetricsCert, envMetricsKey, envMetricsClientCA)
}

// checkStartupGates enforces T-4-12's positivity contract for both gate
// backstops before any bind: a zero or negative value can never mean "stay
// closed forever" — the empty gate's default and this check both guard it —
// and the error names the offending parameter. Only positivity is checked
// here; cross-parameter relations are Phase 7's one-point validation (Open
// Question 7), and ADR 0006 defines the two parameters separately, so neither
// implies the other.
func checkStartupGates(cfg config) error {
	if cfg.startupGate <= 0 {
		return fmt.Errorf("%w: --startup-gate=%s", errStartupGateRange, cfg.startupGate)
	}
	if cfg.startupGateEmpty <= 0 {
		return fmt.Errorf("%w: --startup-gate-empty=%s", errStartupGateRange, cfg.startupGateEmpty)
	}
	return nil
}

// checkValueMaxBytes rejects a non-positive --value-max-bytes before any bind
// (PERSIST-01, T-5-02): the store normalizes such a value to the default, so
// silently accepting it would hide an operator typo behind what looks like a
// deliberate setting. The error names the flag, and finalizeConfig's caller
// exits with the usage code. A positive value is passed through unchanged —
// the datagram-ceiling relation is checkDatagramCeiling's.
func checkValueMaxBytes(cfg config) error {
	if cfg.valueMaxBytes <= 0 {
		return fmt.Errorf("%w: --value-max-bytes=%d", errValueMaxBytesRange, cfg.valueMaxBytes)
	}
	return nil
}

// checkDatagramCeiling enforces XPORT-03's startup half: the largest record
// the write API accepts — a key of storage.MaxKeyBytes with a value of
// value_max_bytes — must encode into one UDP datagram. The size comes from
// udprepl.MaxRecordDatagramBytes, which marshals the real schema at
// worst-case varint widths, so this check cannot undercount a maximal
// record (and is why --value-max-bytes=64000 still starts: its worst case
// is well below the ceiling). A non-positive value-max-bytes is the store's
// own default (storage.NewStore normalizes it), so the check evaluates that
// same effective value. The probe width is bounded by the ceiling itself: a
// misconfigured multi-gigabyte value is over the ceiling before any key or
// envelope byte, and materializing it just to measure would turn a config
// typo into an out-of-memory startup. A non-positive setting never reaches
// this check — checkValueMaxBytes rejects it first — so the
// storage.ValueMaxBytes fallback keeps the probe total over its input.
// Called by finalizeConfig, before any bind or socket work, so a bad
// combination exits with the usage code (2) rather than starting a node
// whose largest confirmed write could never replicate.
func checkDatagramCeiling(cfg config) error {
	valueBytes := cfg.valueMaxBytes
	if valueBytes <= 0 {
		valueBytes = storage.ValueMaxBytes
	}
	probe := min(valueBytes, udprepl.MaxDatagramBytes)
	size := udprepl.MaxRecordDatagramBytes(storage.MaxKeyBytes, probe)
	if probe == valueBytes && size <= udprepl.MaxDatagramBytes {
		return nil
	}
	measured := strconv.Itoa(size)
	if probe < valueBytes {
		measured = "at least " + measured
	}
	return fmt.Errorf("%w: --value-max-bytes=%d with the %d-byte key limit needs %s-byte datagrams, over the %d-byte UDP ceiling",
		errDatagramCeiling, cfg.valueMaxBytes, storage.MaxKeyBytes, measured, udprepl.MaxDatagramBytes)
}

// gateBackstopFor resolves the startup-gate backstop once, from the loaded
// store's record count (D-02, ADR 0006): the engine finished loading before
// the wiring runs, so KeyCount is final and the resolver reads only the
// duration. A node with any record — live or tombstoned — never consults
// startup_gate_empty.
func gateBackstopFor(cfg config, engine *persist.Engine) time.Duration {
	if engine.KeyCount() == 0 {
		return cfg.startupGateEmpty
	}
	return cfg.startupGate
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
	}, persist.Config{
		Debounce: cfg.dumpDebounce,
		MaxWait:  cfg.dumpMaxWait,
		// One ERROR per failed attempt with the class label the metric uses
		// (D-05, PERSIST-05): a static message plus attributes, never dynamic
		// text, and the engine's dumper goroutine only formats and logs.
		OnDumpFailure: func(err error, class port.FailureClass) {
			slog.Error("Dump failed", "class", class.String(), "detail", err.Error())
		},
	})
	stats := engine.LoadStats()
	logLoadStats(stats)
	if skipped := stats.Decoded.ValueLimit; skipped > 0 {
		slog.Warn("Dumped records skipped by the configured value limit",
			"count", skipped,
			"value_max_bytes", cfg.valueMaxBytes)
	}
	if remaining := discarded - stats.Decoded.ValueLimit; remaining > 0 {
		log.Printf("sup-kv: discarded %d undecodable dumped records at startup", remaining)
	}
	engineCtx, engineCancel := context.WithCancel(context.Background())
	engineDone := make(chan error, 1)
	go func() { engineDone <- engine.Run(engineCtx) }()
	return engine, engineCancel, engineDone
}

// logLoadStats reports one startup load in the one-line-per-class policy
// (D-02, PERSIST-02/PERSIST-09): a Backup emits exactly one WARN naming the
// main generation's path, cause and preservation error; an Unusable exactly
// one ERROR carrying one attribute group per candidate that actually failed
// — one or two, never a phantom — and both are silent for Prompt and Main.
// A non-zero sweep count is one INFO and a sweep error one WARN; a clean
// start therefore logs nothing here.
func logLoadStats(stats persist.LoadStats) {
	switch stats.Outcome {
	case port.LoadBackup:
		logBackupFallback(stats)
	case port.LoadUnusable:
		logUnusableGenerations(stats)
	case port.LoadPrompt, port.LoadMain:
	}
	if stats.Swept > 0 {
		slog.Info("Orphan temp sweep removed files", "count", stats.Swept)
	}
	if stats.SweepErr != nil {
		slog.Warn("Orphan temp sweep failed", "detail", stats.SweepErr.Error())
	}
}

// logBackupFallback emits the one WARN for a .prev fallback. The main
// candidate is found by scanning Failures by Candidate — never by slice
// index — so an absent main's recorded fs.ErrNotExist cause is what the line
// names, and a contract-violating empty list degrades to a path-less WARN
// instead of an index panic.
func logBackupFallback(stats persist.LoadStats) {
	failure, ok := candidateFailure(stats.Failures, port.GenMain)
	if !ok {
		slog.Warn("Dump generation fallback")
		return
	}
	slog.Warn("Dump generation fallback",
		"main_path", failure.Path,
		"cause", errorText(failure.Cause),
		"preserve_error", errorText(failure.PreserveErr))
}

// logUnusableGenerations emits the one ERROR for an unusable chain: one
// <candidate>_path/<candidate>_cause/<candidate>_preserve_error group per
// failure actually present, in the adapter's chain order. Iterating Failures
// handles the absent-candidate case by construction — a single corrupt
// generation yields one group and no phantom .prev entry.
func logUnusableGenerations(stats persist.LoadStats) {
	attrs := make([]any, 0, loadFailureAttrsPerCandidate*len(stats.Failures))
	for _, failure := range stats.Failures {
		prefix := "prev"
		if failure.Candidate == port.GenMain {
			prefix = "main"
		}
		attrs = append(attrs,
			prefix+"_path", failure.Path,
			prefix+"_cause", errorText(failure.Cause),
			prefix+"_preserve_error", errorText(failure.PreserveErr))
	}
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
func wire(cfg config) (*wired, net.Listener, error) {
	hub := grpcapi.NewWatchHub()
	engine, engineCancel, engineDone := startEngine(cfg, hub)

	adapter, err := newReplicationAdapter(cfg, engine)
	if err != nil {
		engineCancel()
		return nil, nil, fmt.Errorf("bind replication socket: %w", err)
	}
	// XPORT-05 startup evidence: the EFFECTIVE (kernel-doubled, clamped)
	// buffer sizes, never the requested constants — the adapter read them
	// back in New (RESEARCH Pattern 7).
	recvBuf, sendBuf := adapter.SocketBuffers()
	slog.Info("Socket buffers", "recv_buf", recvBuf, "send_buf", sendBuf)

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
		WaitTimeout:    cfg.waitTimeout,
	})

	apiCtx, apiCancel := context.WithCancel(context.Background())
	go srv.RunGate(apiCtx)
	go func() { _ = hub.Run(apiCtx) }()

	listener, gs, err := listenGRPC(cfg.grpcAddr, srv)
	if err != nil {
		abortWire(engineCancel, replCancel, apiCancel, nil)
		return nil, nil, err
	}

	var metricsSrv *http.Server
	if cfg.metricsAddr != "" { // D-01: unset means no listener, no goroutine.
		metricsSrv, err = listenMetrics(cfg, metricsSources(adapter, engine))
		if err != nil {
			abortWire(engineCancel, replCancel, apiCancel, listener)
			return nil, nil, err
		}
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
		metricsSrv:   metricsSrv,
	}, listener, nil
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
func newReplicationAdapter(cfg config, engine *persist.Engine) (*udprepl.Adapter, error) {
	return udprepl.New(cfg.bindAddr, cfg.broadcastAddr, cfg.nodeID, engine, repl.Config{
		HelloInterval:      cfg.helloInterval,
		PeerTimeout:        cfg.peerTimeout,
		AckTimeout:         cfg.ackTimeout,
		AckTimeoutMax:      cfg.ackTimeoutMax,
		RetransmitInterval: cfg.retransmitInterval,
		SyncInterval:       cfg.syncInterval,
		SyncChunkBytes:     cfg.syncChunkBytes,
		SyncResponseRate:   cfg.syncResponseRate,
		MaxClockLead:       cfg.maxClockLead,
		// The core's prebuilt metadata strings (REPL-08 configuration
		// errors, apply failures) ride as the detail attr under a static
		// capitalized msg — the sloglint static-msg discipline (D-08); a
		// dynamic string never becomes the msg.
		Log: func(msg string) { slog.Warn("Replication adapter", "detail", msg) },
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
func listenMetrics(cfg config, sources metrics.Sources) (*http.Server, error) {
	inner, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", cfg.metricsAddr)
	if err != nil {
		return nil, fmt.Errorf("listen tcp %s: %w", cfg.metricsAddr, err)
	}
	listener := inner
	if cfg.metricsCert != "" {
		tlsCfg, tlsErr := metrics.NewTLSConfig(cfg.metricsCert, cfg.metricsKey, cfg.metricsClientCA)
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
// one grpc.Server. Reflection lets grpcurl discover the whole API with zero
// proto files — docs/EXAMPLES.md walks the service exactly that way; the
// schema is public in the repo and the default bind is loopback on the
// trusted segment, so always-on reflection adds no exposure worth a flag.
func listenGRPC(grpcAddr string, srv *grpcapi.Server) (net.Listener, *grpc.Server, error) {
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", grpcAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("listen tcp %s: %w", grpcAddr, err)
	}
	gs := grpc.NewServer()
	apiproto.RegisterSupKVServiceServer(gs, srv)
	healthpb.RegisterHealthServer(gs, srv.Health())
	reflection.Register(gs)
	return listener, gs, nil
}

// armSignals takes ownership of SIGTERM/SIGINT for the whole run — serving
// and the D-19 shutdown alike (P1-IN-01). The buffered channel holds the
// first signal for serve and a possible second for shutdown's force path,
// and nothing restores the default disposition until [signal.Stop] runs in
// main after the final dump, so no repeated signal can kill the process
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
// error return: the node cannot do its one job, and main decides the
// verdict.
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
// the contract, RESEARCH Pattern 6): shed client traffic through the health
// endpoint, shut the opt-in metrics/debug listener down — positioned right
// after the health flip so scrapers see the close the moment the node stops
// serving, while the rest of the drain stays observable through logs
// (Pitfall 7) — cancel the API context to end Watch streams and the hub
// (GracefulStop blocks forever while any stream is open), bound
// GracefulStop with the grace period and fall back to Stop, stop
// replication and wait for the adapter goroutine, Flush — the final
// write-confirmed dump covering remote-merged dirtiness — and only then
// cancel the engine context, whose exit sweep now finds no waiters. A final
// dump failure is the returned error: a clean exit is reserved for a
// confirmed shutdown.
//
// Both remaining waits are bounded by the same shutdownGrace (PERSIST-08,
// T-5-24): the final flush is FlushTimeout, so a stuck fsync answers
// ErrUnconfirmed instead of hanging the process, and the engineDone wait is
// a select against a grace timer — a dumper still stuck in the kernel at
// process exit is abandoned with one WARN, which is the deliberate choice
// over an unkillable shutdown. A flush error keeps the exit-code-1 path: a
// clean exit still means the final dump was confirmed.
//
// A second SIGTERM/SIGINT observed on sig while GracefulStop drains is an
// operator's hurry-up: it takes the same force path as the grace deadline
// (Stop) immediately. The final Flush sits after GracefulStop in the
// sequence and therefore survives the force stop — the confirmed dump still
// lands, and a clean shutdown still exits 0.
func shutdown(node *wired, sig <-chan os.Signal) error {
	node.health.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
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

// envSlogLevel resolves the log-level env knob (D-07): debug, info, warn or
// error, case-insensitively, per [slog.Level.UnmarshalText]. A malformed
// value is an error, never a silently-swallowed default.
func envSlogLevel(env string, def slog.Level) (slog.Level, error) {
	raw := envOrDefault(env, "")
	if raw == "" {
		return def, nil
	}
	var lvl slog.Level
	err := lvl.UnmarshalText([]byte(raw))
	if err != nil {
		return def, fmt.Errorf("must be debug, info, warn or error: %w", err)
	}
	return lvl, nil
}

// envLogValue resolves the log-format env knob (D-06): json (the default)
// or text, case-insensitively. A malformed value is an error, never a
// silently-swallowed default.
func envLogValue(env, def string) (string, error) {
	raw := envOrDefault(env, "")
	switch strings.ToLower(raw) {
	case "":
		return def, nil
	case logFormatJSON, logFormatText:
		return strings.ToLower(raw), nil
	default:
		return def, fmt.Errorf("%w: %q", errLogFormatEnv, raw)
	}
}
