package config

import (
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/udprepl"
)

// Startup-check sentinels the reported errors wrap; callers and process tests
// match them with [errors.Is] or by the stable substring, never identity
// comparison.
var (
	errNodeIDRange        = errors.New("node-id exceeds the 16-bit node ID space")
	errMissingRequired    = errors.New("missing required knobs")
	errIncompleteCertSet  = errors.New("the metrics certificate envs must be set together")
	errDatagramCeiling    = errors.New("value-max-bytes exceeds the datagram ceiling")
	errValueMaxBytesRange = errors.New("value-max-bytes must be positive")
	errSyncChunkBytes     = errors.New("sync-chunk-bytes exceeds the datagram ceiling")
	errStartupGateRange   = errors.New("startup gate must be positive")
	errPositiveValue      = errors.New("must be positive")
	errRetransmitFloor    = errors.New("must be at least 1ms")
	errSyncResponseCap    = errors.New("exceeds the sync response rate cap")
	errPeerVsHello        = errors.New("peer timeout must be greater than the hello interval")
	errAckMax             = errors.New("ack timeout must not exceed ack timeout max")
	errDriftVsLead        = errors.New("max clock lead must be at least drift warn plus 1ms")
	errAddressShape       = errors.New("must be a host:port address")
)

// rule is one named startup check. Validate walks the slice in order and joins
// every failure under the rule's name, so the operator sees each bad knob in
// one pass instead of fixing them one restart at a time.
type rule struct {
	name  string
	check func(Config) error
}

// rules returns the startup-check catalog. It is a function rather than a
// package global so no mutable package state exists, and Validate and the
// meta-test iterate the same list — a rule cannot be added untested or
// silently skipped.
func rules() []rule {
	return []rule{
		{"required-knobs", checkRequiredKnobs},
		{"node-id-range", checkNodeIDRange},
		{"positive-values", checkPositiveValues},
		{"retransmit-floor", checkRetransmitFloor},
		{"sync-response-cap", checkSyncResponseCap},
		{"sync-chunk-ceiling", checkSyncChunkCeiling},
		{"datagram-ceiling", checkDatagramCeiling},
		{"peer-vs-hello", checkPeerVsHello},
		{"ack-max", checkAckMax},
		{"drift-vs-lead", checkDriftVsLead},
		{"metrics-cert-triple", checkMetricsCerts},
		{"address-shape", checkAddressShape},
	}
}

// Validate runs every startup rule over a parsed Config, before any bind, and
// returns the failures joined: each names its rule and the offending
// parameter, and the caller exits through the usage path (exit 2). A
// configuration error is therefore reported once, in full, and never
// normalized into a running node.
func Validate(c Config) error {
	var errs []error
	for _, r := range rules() {
		err := r.check(c)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", r.name, err))
		}
	}
	return errors.Join(errs...)
}

// checkRequiredKnobs names every required knob left at its zero default — its
// SUPKV_ twin unset as well; node-id's zero is not a usable identity.
func checkRequiredKnobs(c Config) error {
	var missing []string
	if c.NodeID == 0 {
		missing = append(missing, "--node-id")
	}
	if c.BindAddr == "" {
		missing = append(missing, "--bind-addr")
	}
	if c.BroadcastAddr == "" {
		missing = append(missing, "--broadcast-addr")
	}
	if c.DumpPath == "" {
		missing = append(missing, "--dump-path")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", errMissingRequired, strings.Join(missing, ", "))
}

// checkNodeIDRange keeps the node identity inside the wire's 16-bit field.
func checkNodeIDRange(c Config) error {
	if c.NodeID > math.MaxUint16 {
		return fmt.Errorf("--node-id %d: %w", c.NodeID, errNodeIDRange)
	}
	return nil
}

// positiveValues are the duration knobs with their positivity sentinel: both
// gate backstops keep the original "startup gate must be positive" text (the
// process tests assert it), while the rest share the generic message.
func positiveValues(c Config) []struct {
	flag     string
	value    time.Duration
	sentinel error
} {
	return []struct {
		flag     string
		value    time.Duration
		sentinel error
	}{
		{"--startup-gate", c.StartupGate, errStartupGateRange},
		{"--startup-gate-empty", c.StartupGateEmpty, errStartupGateRange},
		{"--wait-timeout", c.WaitTimeout, errPositiveValue},
		{"--hello-interval", c.HelloInterval, errPositiveValue},
		{"--peer-timeout", c.PeerTimeout, errPositiveValue},
		{"--ack-timeout", c.AckTimeout, errPositiveValue},
		{"--ack-timeout-max", c.AckTimeoutMax, errPositiveValue},
		{"--retransmit-interval", c.RetransmitInterval, errPositiveValue},
		{"--sync-interval", c.SyncInterval, errPositiveValue},
		{"--max-clock-lead", c.MaxClockLead, errPositiveValue},
		{"--dump-debounce", c.DumpDebounce, errPositiveValue},
		{"--dump-max-wait", c.DumpMaxWait, errPositiveValue},
		{"--drift-warn", c.DriftWarn, errPositiveValue},
	}
}

// checkPositiveValues rejects every non-positive numeric knob (T-07-07): a
// zero or negative value would otherwise be silently normalized to a design
// default, hiding an operator typo behind what looks like a deliberate
// setting. All offenders are reported in one pass.
func checkPositiveValues(c Config) error {
	var errs []error
	for _, d := range positiveValues(c) {
		if d.value <= 0 {
			errs = append(errs, fmt.Errorf("%w: %s=%s", d.sentinel, d.flag, d.value))
		}
	}
	if c.ValueMaxBytes <= 0 {
		errs = append(errs, fmt.Errorf("%w: --value-max-bytes=%d", errValueMaxBytesRange, c.ValueMaxBytes))
	}
	if c.SyncChunkBytes <= 0 {
		errs = append(errs, fmt.Errorf("%w: --sync-chunk-bytes=%d", errPositiveValue, c.SyncChunkBytes))
	}
	if c.SyncResponseRate <= 0 {
		errs = append(errs, fmt.Errorf("%w: --sync-response-rate=%d", errPositiveValue, c.SyncResponseRate))
	}
	return errors.Join(errs...)
}

// checkRetransmitFloor rejects a sub-millisecond scan period (D-03 refined):
// the driver may sleep RetransmitInterval while pending entries exist, so a
// nanosecond period is a busy loop. Non-positive values are positivity's
// finding, not a second report here.
func checkRetransmitFloor(c Config) error {
	if c.RetransmitInterval > 0 && c.RetransmitInterval < time.Millisecond {
		return fmt.Errorf("--retransmit-interval=%s: %w", c.RetransmitInterval, errRetransmitFloor)
	}
	return nil
}

// checkSyncResponseCap keeps the rate inside the domain where one second
// divided by the rate stays a non-zero spacing (the core's own cap, never a
// duplicated literal). Non-positive values are positivity's finding.
func checkSyncResponseCap(c Config) error {
	if c.SyncResponseRate > repl.MaxSyncResponseRate {
		return fmt.Errorf("--sync-response-rate=%d: %w (%d/s)",
			c.SyncResponseRate, errSyncResponseCap, repl.MaxSyncResponseRate)
	}
	return nil
}

// checkSyncChunkCeiling rejects an explicit --sync-chunk-bytes above the UDP
// datagram ceiling before any bind (WR-06): a chunk budget above 65 507
// describes a chunk no datagram can carry, and the core's defensive clamp
// would silently ship smaller chunks than the operator configured. The
// ceiling is the core's own constant.
func checkSyncChunkCeiling(c Config) error {
	if c.SyncChunkBytes > repl.MaxSyncChunkBytes {
		return fmt.Errorf("%w: --sync-chunk-bytes=%d (at most %d, the largest IPv4 UDP payload)",
			errSyncChunkBytes, c.SyncChunkBytes, repl.MaxSyncChunkBytes)
	}
	return nil
}

// checkDatagramCeiling enforces XPORT-03's startup half: the largest record
// the write API accepts — a key of storage.MaxKeyBytes with a value of
// value_max_bytes — must encode into one UDP datagram. The size comes from
// udprepl.MaxRecordDatagramBytes, which marshals the real schema at
// worst-case varint widths, so this check cannot undercount a maximal record.
// A non-positive value-max-bytes is the store's own default, so the check
// evaluates that same effective value; the probe width is bounded by the
// ceiling itself, because materializing a misconfigured multi-gigabyte value
// just to measure it would turn a config typo into an out-of-memory startup.
func checkDatagramCeiling(c Config) error {
	valueBytes := c.ValueMaxBytes
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
		errDatagramCeiling, c.ValueMaxBytes, storage.MaxKeyBytes, measured, udprepl.MaxDatagramBytes)
}

// checkPeerVsHello enforces the OBSV-02 adjacency rule in the safe direction:
// a peer evicted at the same instant it greets would never stay discovered,
// so peer_timeout must be strictly greater than hello_interval. A
// non-positive side is positivity's finding, so the relation is evaluated
// only when both are in domain.
func checkPeerVsHello(c Config) error {
	if c.PeerTimeout <= 0 || c.HelloInterval <= 0 {
		return nil
	}
	if c.PeerTimeout <= c.HelloInterval {
		return fmt.Errorf("%w: --peer-timeout=%s must be greater than --hello-interval=%s",
			errPeerVsHello, c.PeerTimeout, c.HelloInterval)
	}
	return nil
}

// checkAckMax keeps the retransmit ladder's ceiling at or above its base:
// equality is legal (a single rung), a base above the ceiling is not.
func checkAckMax(c Config) error {
	if c.AckTimeout <= 0 || c.AckTimeoutMax <= 0 {
		return nil
	}
	if c.AckTimeout > c.AckTimeoutMax {
		return fmt.Errorf("%w: --ack-timeout=%s must not exceed --ack-timeout-max=%s",
			errAckMax, c.AckTimeout, c.AckTimeoutMax)
	}
	return nil
}

// checkDriftVsLead keeps the drift WARN window reachable (D-02 refined):
// version.Validate and driftBetween both truncate to the layout's millisecond
// resolution, so the comparison is `max_clock_lead >= drift_warn + 1ms`.
// Equality at millisecond resolution leaves no reachable warning window — the
// leading version is discarded before the warning can fire.
func checkDriftVsLead(c Config) error {
	if c.DriftWarn <= 0 || c.MaxClockLead <= 0 {
		return nil
	}
	if c.MaxClockLead < c.DriftWarn+time.Millisecond {
		return fmt.Errorf("%w: --max-clock-lead=%s must be at least --drift-warn=%s plus 1ms",
			errDriftVsLead, c.MaxClockLead, c.DriftWarn)
	}
	return nil
}

// checkMetricsCerts enforces D-02's all-or-nothing contract before any bind
// (Pitfall 8): any of the three certificate envs set without all three is a
// startup error naming every env of the triple — no code path can reach the
// TLS builder with a half-loaded set, and with none set the listener serves
// plaintext (the loopback/debug mode).
func checkMetricsCerts(c Config) error {
	noneSet := c.MetricsCert == "" && c.MetricsKey == "" && c.MetricsClientCA == ""
	allSet := c.MetricsCert != "" && c.MetricsKey != "" && c.MetricsClientCA != ""
	if noneSet || allSet {
		return nil
	}
	return fmt.Errorf("%w: %s, %s and %s",
		errIncompleteCertSet, envMetricsCert, envMetricsKey, envMetricsClientCA)
}

// checkAddressShape rejects a non-empty address that is not host:port and an
// explicitly empty --grpc-addr: [net.Listen] with an empty address binds the
// wildcard address on an ephemeral port, so an empty value would expose the
// unauthenticated API to every reachable host instead of meaning "no
// listener". Only the metrics endpoint has that documented opt-in; an empty
// bind/broadcast value is required-knobs' finding.
func checkAddressShape(c Config) error {
	addresses := []struct {
		flag  string
		value string
	}{
		{"--bind-addr", c.BindAddr},
		{"--broadcast-addr", c.BroadcastAddr},
		{"--grpc-addr", c.GRPCAddr},
		{"--metrics-addr", c.MetricsAddr},
	}
	var errs []error
	if c.GRPCAddr == "" {
		errs = append(errs, fmt.Errorf("%w: --grpc-addr must not be empty (the default is %s)",
			errAddressShape, defaultGRPCAddr))
	}
	for _, a := range addresses {
		if a.value == "" {
			continue
		}
		if !validHostPort(a.value) {
			errs = append(errs, fmt.Errorf("%w: %s=%q", errAddressShape, a.flag, a.value))
		}
	}
	return errors.Join(errs...)
}

// validHostPort reports whether addr splits into a host and a numeric port.
func validHostPort(addr string) bool {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	_, err = strconv.Atoi(port)
	return err == nil
}
