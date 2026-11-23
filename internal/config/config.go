// Package config is the single parse, validate and usage point for the
// sup-kv launch configuration (OBSV-02, D-03): it resolves every SUPKV_ env
// twin into a flag default (flag > env > default), rejects unknown flags and
// positional arguments by name, and owns [Validate], the one function every
// startup range and relation check lives in — so misconfiguration fails once,
// naming the parameter, before any socket or file work.
//
// The package reads the environment and imports the adapters whose datagram
// ceiling it validates, so it deliberately lives outside internal/app: the
// no-I/O depguard rule must not cover it, and the configuration package uses
// the standard library and own-module imports only.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/storage"
)

// ErrHelp reports that --help (or -h) was requested: the caller renders
// [Usage] on standard output and exits 0, never through the error path.
var ErrHelp = errors.New("help requested")

// errPositionalArguments reports non-flag arguments: the usage contract is
// reject-by-name, never a silently ignored token (T-07-05).
var errPositionalArguments = errors.New("unexpected positional arguments")

// errNodeIDEnv reports a SUPKV_NODE_ID value outside the wire's 16-bit node
// ID space; the env name is attached by the resolver.
var errNodeIDEnv = errors.New("must be an integer within the 16-bit node ID space")

// errLogFormatEnv reports a SUPKV_LOG_FORMAT value other than json or text.
var errLogFormatEnv = errors.New("must be json or text")

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

// Config is the resolved launch configuration, built once at startup. NodeID
// stays wide (uint64) so the 16-bit range check is a [Validate] rule rather
// than a parse side effect; every other field carries the flag/env value
// verbatim.
type Config struct {
	NodeID             uint64
	BindAddr           string
	BroadcastAddr      string
	DumpPath           string
	GRPCAddr           string
	StartupGate        time.Duration
	StartupGateEmpty   time.Duration
	WaitTimeout        time.Duration
	ValueMaxBytes      int
	SyncChunkBytes     int
	HelloInterval      time.Duration
	PeerTimeout        time.Duration
	AckTimeout         time.Duration
	AckTimeoutMax      time.Duration
	RetransmitInterval time.Duration
	SyncInterval       time.Duration
	SyncResponseRate   int
	MaxClockLead       time.Duration
	DumpDebounce       time.Duration
	DumpMaxWait        time.Duration
	DriftWarn          time.Duration
	LogLevel           slog.Level
	LogFormat          string
	MetricsAddr        string
	MetricsCert        string
	MetricsKey         string
	MetricsClientCA    string
	ShowVersion        bool
}

// WireNodeID narrows NodeID to the 16-bit wire identity. [Validate] proved
// the range, so the conversion is total on a validated Config.
func (c Config) WireNodeID() uint16 {
	return uint16(c.NodeID) //nolint:gosec // Validate rejects NodeID > math.MaxUint16 before any wire use, so the narrowing cannot truncate.
}

// IsTextFormat reports whether the resolved log format is the key=value text
// form rather than the JSON default (D-06).
func (c Config) IsTextFormat() bool {
	return c.LogFormat == logFormatText
}

// Parse resolves args against the environment behind getenv. It is pure over
// its inputs — one local flag set, no package globals, no slog install — so
// parallel callers cannot observe each other's flags.
//
// The order is the contract: --help yields [ErrHelp]; --version returns
// ShowVersion with a nil error BEFORE any malformed-env error is joined (the
// build-identity probe must work on a broken environment); positional
// arguments are rejected by name; only then does a collected malformed-env
// error surface, naming every offending variable.
func Parse(args []string, getenv func(string) string) (Config, error) {
	var cfg Config
	env := &resolver{getenv: getenv}
	fs := newFlagSet(&cfg, env)
	err := fs.Parse(args)
	if errors.Is(err, flag.ErrHelp) {
		return Config{}, ErrHelp
	}
	if err != nil {
		return Config{}, err
	}
	if cfg.ShowVersion {
		return cfg, nil
	}
	if rest := fs.Args(); len(rest) > 0 {
		return Config{}, fmt.Errorf("%w: %s", errPositionalArguments, strings.Join(rest, " "))
	}
	return cfg, errors.Join(env.errs...)
}

// newFlagSet builds the one local flag set: a fresh set per call, so no
// package-level parser state exists and every test can parse its own args.
func newFlagSet(cfg *Config, env *resolver) *flag.FlagSet {
	fs := flag.NewFlagSet("sup-kv", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	register(fs, cfg, env)
	return fs
}

// register declares every knob as a flag whose default is its SUPKV_ env twin
// (or the design default) — an explicit flag overrides the environment and
// the environment overrides the default (D-17). A malformed env value is
// collected under its variable name; Parse joins the collection after the
// --version short-circuit.
func register(fs *flag.FlagSet, cfg *Config, env *resolver) {
	fs.BoolVar(&cfg.ShowVersion, "version", false,
		"print version, commit and Go runtime version, then exit")
	fs.Uint64Var(&cfg.NodeID, "node-id", env.nodeID(envNodeID),
		"required unique 16-bit node ID (env SUPKV_NODE_ID)")
	fs.StringVar(&cfg.BindAddr, "bind-addr", env.text(envBindAddr, ""),
		"required local UDP bind address, e.g. 0.0.0.0:7900 (env SUPKV_BIND_ADDR)")
	fs.StringVar(&cfg.BroadcastAddr, "broadcast-addr", env.text(envBroadcastAddr, ""),
		"required segment broadcast address, e.g. 127.255.255.255:7900 (env SUPKV_BROADCAST_ADDR)")
	fs.StringVar(&cfg.DumpPath, "dump-path", env.text(envDumpPath, ""),
		"required dump file path (env SUPKV_DUMP_PATH)")
	fs.StringVar(&cfg.GRPCAddr, "grpc-addr", env.text(envGRPCAddr, defaultGRPCAddr),
		"gRPC listen address, distinct from the UDP bind (env SUPKV_GRPC_ADDR)")
	fs.StringVar(&cfg.MetricsAddr, "metrics-addr", env.text(envMetricsAddr, ""),
		"optional metrics/debug listen address — unset means no listener (env SUPKV_METRICS_ADDR)")
	fs.DurationVar(&cfg.StartupGate, "startup-gate", env.duration(envStartupGate, defaultStartupGate),
		"gate backstop: writable this long after start even without a sync round (env SUPKV_STARTUP_GATE)")
	fs.DurationVar(&cfg.StartupGateEmpty, "startup-gate-empty",
		env.duration(envStartupGateEmpty, defaultStartupGateEmpty),
		"empty-disk gate backstop: a node with no loaded records waits this long for a peer listing (env SUPKV_STARTUP_GATE_EMPTY)")
	fs.DurationVar(&cfg.WaitTimeout, "wait-timeout", env.duration(envWaitTimeout, defaultWaitTimeout),
		"default wait_for_replication bound (env SUPKV_WAIT_TIMEOUT)")
	fs.IntVar(&cfg.ValueMaxBytes, "value-max-bytes", env.integer(envValueMaxBytes, storage.ValueMaxBytes),
		"maximum value size in bytes (env SUPKV_VALUE_MAX_BYTES)")
	fs.IntVar(&cfg.SyncChunkBytes, "sync-chunk-bytes", env.integer(envSyncChunkBytes, defaultSyncChunkBytes),
		"DataSync chunk byte budget (env SUPKV_SYNC_CHUNK_BYTES)")
	fs.IntVar(&cfg.SyncResponseRate, "sync-response-rate",
		env.integer(envSyncResponseRate, defaultSyncRespRate),
		"sync responses per second cap (env SUPKV_SYNC_RESPONSE_RATE)")
	registerDurationFlags(fs, cfg, env)

	cfg.LogFormat = env.logValue(envLogFormat, defaultLogFormat)
	cfg.LogLevel = env.slogLevel(envLogLevel, defaultLogLevel)
	cfg.MetricsCert = env.text(envMetricsCert, "")
	cfg.MetricsKey = env.text(envMetricsKey, "")
	cfg.MetricsClientCA = env.text(envMetricsClientCA, "")
}

// registerDurationFlags declares the replication-timing and dump-cycle knobs
// — every duration flag except startup-gate and wait-timeout, which sit with
// the API-facing registrations.
func registerDurationFlags(fs *flag.FlagSet, cfg *Config, env *resolver) {
	fs.DurationVar(&cfg.HelloInterval, "hello-interval", env.duration(envHelloInterval, defaultHelloInterval),
		"discovery broadcast period (env SUPKV_HELLO_INTERVAL)")
	fs.DurationVar(&cfg.PeerTimeout, "peer-timeout", env.duration(envPeerTimeout, defaultPeerTimeout),
		"peer eviction silence bound (env SUPKV_PEER_TIMEOUT)")
	fs.DurationVar(&cfg.AckTimeout, "ack-timeout", env.duration(envAckTimeout, defaultAckTimeout),
		"retransmit ladder base (env SUPKV_ACK_TIMEOUT)")
	fs.DurationVar(&cfg.AckTimeoutMax, "ack-timeout-max", env.duration(envAckTimeoutMax, defaultAckTimeoutMax),
		"retransmit ladder ceiling (env SUPKV_ACK_TIMEOUT_MAX)")
	fs.DurationVar(&cfg.RetransmitInterval, "retransmit-interval",
		env.duration(envRetransmitInterval, defaultRetransmit),
		"retransmit scan period (env SUPKV_RETRANSMIT_INTERVAL)")
	fs.DurationVar(&cfg.SyncInterval, "sync-interval", env.duration(envSyncInterval, defaultSyncInterval),
		"proactive sync listing period (env SUPKV_SYNC_INTERVAL)")
	fs.DurationVar(&cfg.MaxClockLead, "max-clock-lead", env.duration(envMaxClockLead, defaultMaxClockLead),
		"reject received versions beyond now + this bound (env SUPKV_MAX_CLOCK_LEAD)")
	fs.DurationVar(&cfg.DumpDebounce, "dump-debounce", env.duration(envDumpDebounce, defaultDumpDebounce),
		"dump quiet window (env SUPKV_DUMP_DEBOUNCE)")
	fs.DurationVar(&cfg.DumpMaxWait, "dump-max-wait", env.duration(envDumpMaxWait, defaultDumpMaxWait),
		"dump start bound from the oldest unconfirmed write (env SUPKV_DUMP_MAX_WAIT)")
	fs.DurationVar(&cfg.DriftWarn, "drift-warn", env.duration(envDriftWarn, defaultDriftWarn),
		"clock drift warning threshold (env SUPKV_DRIFT_WARN)")
}

// resolver reads the SUPKV_ environment through one seam and collects every
// malformed value under its variable name — the contract that a bad env value
// aborts startup naming the variable, never a silently swallowed default.
type resolver struct {
	getenv func(string) string
	errs   []error
}

// lookup returns the env value, treating an unset or empty value as absent
// (the D-17 reading the shipped binary has always applied).
func (r *resolver) lookup(env string) string {
	if r.getenv == nil {
		return ""
	}
	return r.getenv(env)
}

// fail records a malformed value under its env name.
func (r *resolver) fail(env string, err error) {
	r.errs = append(r.errs, fmt.Errorf("%s: %w", env, err))
}

// text resolves a string knob.
func (r *resolver) text(env, def string) string {
	if v := r.lookup(env); v != "" {
		return v
	}
	return def
}

// duration resolves a knob that must parse as a Go duration.
func (r *resolver) duration(env string, def time.Duration) time.Duration {
	raw := r.lookup(env)
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		r.fail(env, fmt.Errorf("must be a duration: %w", err))
		return def
	}
	return d
}

// integer resolves a knob that must parse as an int; the positivity rules are
// [Validate]'s, so a negative value is parsed, never silently normalized.
func (r *resolver) integer(env string, def int) int {
	raw := r.lookup(env)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		r.fail(env, fmt.Errorf("must be an integer: %w", err))
		return def
	}
	return v
}

// nodeID resolves the node-id env twin: an integer inside the wire's 16-bit
// node ID space.
func (r *resolver) nodeID(env string) uint64 {
	raw := r.lookup(env)
	if raw == "" {
		return 0
	}
	v, err := strconv.ParseUint(raw, 10, 16)
	if err != nil {
		r.fail(env, errNodeIDEnv)
		return 0
	}
	return v
}

// slogLevel resolves the log-level env knob (D-07): debug, info, warn or
// error, case-insensitively, per [slog.Level.UnmarshalText].
func (r *resolver) slogLevel(env string, def slog.Level) slog.Level {
	raw := r.lookup(env)
	if raw == "" {
		return def
	}
	var lvl slog.Level
	err := lvl.UnmarshalText([]byte(raw))
	if err != nil {
		r.fail(env, fmt.Errorf("must be debug, info, warn or error: %w", err))
		return def
	}
	return lvl
}

// logValue resolves the log-format env knob (D-06): json (the default) or
// text, case-insensitively.
func (r *resolver) logValue(env, def string) string {
	raw := r.lookup(env)
	switch strings.ToLower(raw) {
	case "":
		return def
	case logFormatJSON, logFormatText:
		return strings.ToLower(raw)
	default:
		r.fail(env, fmt.Errorf("%w: %q", errLogFormatEnv, raw))
		return def
	}
}
