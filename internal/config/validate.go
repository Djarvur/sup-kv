package config

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

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
)

// Validate runs every startup check over a parsed Config, before any bind:
// the node-id range and the required knobs, the metrics certificate triple,
// the startup gates, the value limit and its datagram-ceiling relation, and
// the sync chunk budget. A failure names the offending parameter and the
// caller exits through the usage path (exit 2), so an operator typo is never
// silently normalized into a running node.
func Validate(c Config) error {
	if c.NodeID > math.MaxUint16 {
		return fmt.Errorf("--node-id %d: %w", c.NodeID, errNodeIDRange)
	}
	if missing := missingRequired(c); len(missing) > 0 {
		return fmt.Errorf("%w: %s", errMissingRequired, strings.Join(missing, ", "))
	}
	err := checkMetricsCerts(c)
	if err != nil {
		return err
	}
	err = checkStartupGates(c)
	if err != nil {
		return err
	}
	err = checkValueMaxBytes(c)
	if err != nil {
		return err
	}
	err = checkDatagramCeiling(c)
	if err != nil {
		return err
	}
	return checkSyncChunkBytes(c)
}

// missingRequired names every required knob left at its zero default — its
// SUPKV_ twin unset as well; node-id's zero is not a usable identity.
func missingRequired(c Config) []string {
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
	return missing
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

// checkStartupGates enforces T-4-12's positivity contract for both gate
// backstops before any bind: a zero or negative value can never mean "stay
// closed forever" — the empty gate's default and this check both guard it —
// and the error names the offending parameter. Only positivity is checked
// here; cross-parameter relations are Phase 7's one-point validation (Open
// Question 7), and ADR 0006 defines the two parameters separately, so neither
// implies the other.
func checkStartupGates(c Config) error {
	if c.StartupGate <= 0 {
		return fmt.Errorf("%w: --startup-gate=%s", errStartupGateRange, c.StartupGate)
	}
	if c.StartupGateEmpty <= 0 {
		return fmt.Errorf("%w: --startup-gate-empty=%s", errStartupGateRange, c.StartupGateEmpty)
	}
	return nil
}

// checkValueMaxBytes rejects a non-positive --value-max-bytes before any bind
// (PERSIST-01, T-5-02): the store normalizes such a value to the default, so
// silently accepting it would hide an operator typo behind what looks like a
// deliberate setting. The error names the flag, and the caller exits with the
// usage code. A positive value is passed through unchanged — the
// datagram-ceiling relation is checkDatagramCeiling's.
func checkValueMaxBytes(c Config) error {
	if c.ValueMaxBytes <= 0 {
		return fmt.Errorf("%w: --value-max-bytes=%d", errValueMaxBytesRange, c.ValueMaxBytes)
	}
	return nil
}

// checkDatagramCeiling enforces XPORT-03's startup half: the largest record
// the write API accepts — a key of storage.MaxKeyBytes with a value of
// value_max_bytes — must encode into one UDP datagram. The size comes from
// udprepl.MaxRecordDatagramBytes, which marshals the real schema at
// worst-case varint widths, so this check cannot undercount a maximal
// record. A non-positive value-max-bytes is the store's own default, so the
// check evaluates that same effective value; the probe width is bounded by
// the ceiling itself, because materializing a misconfigured multi-gigabyte
// value just to measure it would turn a config typo into an out-of-memory
// startup.
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

// checkSyncChunkBytes rejects an explicit --sync-chunk-bytes above the UDP
// datagram ceiling before any bind (WR-06): a chunk budget above 65 507
// describes a chunk no datagram can carry, and the core's defensive clamp
// would silently ship smaller chunks than the operator configured. The
// ceiling is the core's own constant (repl.MaxSyncChunkBytes, pinned to the
// adapter's datagram ceiling by a cross-seam test). A zero or negative value
// stays legal here — the core normalizes it to the design default.
func checkSyncChunkBytes(c Config) error {
	if c.SyncChunkBytes > repl.MaxSyncChunkBytes {
		return fmt.Errorf("%w: --sync-chunk-bytes=%d (at most %d, the largest IPv4 UDP payload)",
			errSyncChunkBytes, c.SyncChunkBytes, repl.MaxSyncChunkBytes)
	}
	return nil
}
