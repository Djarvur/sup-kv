package config //nolint:testpackage // rules() and addrShape are unexported: the meta-test pins the rule table by iterating the same slice Validate walks, which the external package cannot reach.

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/app/storage"
)

// validConfig returns a fully valid configuration — every rule satisfied — so
// each subtest mutates exactly the field under test and any other rejection
// is a test bug, not noise. Every value is a shipped default, which also pins
// the "the defaults must validate" contract: the 2 m max-clock-lead default
// is strictly above the 60 s drift-warn default.
func validConfig() Config {
	return Config{
		NodeID:             1,
		BindAddr:           "0.0.0.0:7900",
		BroadcastAddr:      "127.255.255.255:7900",
		DumpPath:           "/var/lib/sup-kv/dump.json",
		GRPCAddr:           defaultGRPCAddr,
		StartupGate:        defaultStartupGate,
		StartupGateEmpty:   defaultStartupGateEmpty,
		WaitTimeout:        defaultWaitTimeout,
		ValueMaxBytes:      storage.ValueMaxBytes,
		SyncChunkBytes:     defaultSyncChunkBytes,
		HelloInterval:      defaultHelloInterval,
		PeerTimeout:        defaultPeerTimeout,
		AckTimeout:         defaultAckTimeout,
		AckTimeoutMax:      defaultAckTimeoutMax,
		RetransmitInterval: defaultRetransmit,
		SyncInterval:       defaultSyncInterval,
		SyncResponseRate:   defaultSyncRespRate,
		MaxClockLead:       defaultMaxClockLead,
		DumpDebounce:       defaultDumpDebounce,
		DumpMaxWait:        defaultDumpMaxWait,
		DriftWarn:          defaultDriftWarn,
		LogLevel:           defaultLogLevel,
		LogFormat:          defaultLogFormat,
	}
}

// assertValid requires a clean validation.
func assertValid(t *testing.T, c Config) {
	t.Helper()
	err := Validate(c)
	if err != nil {
		t.Fatalf("Validate() error = %v, want nil for a fully valid configuration", err)
	}
}

// assertRejected requires a rejection whose text names every want substring —
// the operator-facing contract is that the parameter is always named.
func assertRejected(t *testing.T, c Config, want ...string) {
	t.Helper()
	err := Validate(c)
	if err == nil {
		t.Fatalf("Validate() error = nil, want a rejection naming %v", want)
	}
	for _, w := range want {
		if !strings.Contains(err.Error(), w) {
			t.Fatalf("Validate() error %q does not name %q", err, w)
		}
	}
}

// TestValidate is the rule-table contract: one subtest per rule plus the
// boundary and relation rows, each starting from a fully valid configuration
// and mutating only the knob under test.
func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("the valid configuration is accepted", func(t *testing.T) {
		t.Parallel()
		assertValid(t, validConfig())
	})

	t.Run("required-knobs names every missing knob in one pass", func(t *testing.T) {
		t.Parallel()
		err := Validate(Config{})
		if err == nil {
			t.Fatal("Validate(Config{}) error = nil, want the required-knob rejection")
		}
		for _, want := range []string{"--node-id", "--bind-addr", "--broadcast-addr", "--dump-path"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("the empty-configuration error %q does not name %q", err, want)
			}
		}
	})

	t.Run("required-knobs rejects each missing knob alone", func(t *testing.T) {
		t.Parallel()
		rows := []struct {
			name   string
			mutate func(*Config)
			flag   string
		}{
			{"node-id", func(c *Config) { c.NodeID = 0 }, "--node-id"},
			{"bind-addr", func(c *Config) { c.BindAddr = "" }, "--bind-addr"},
			{"broadcast-addr", func(c *Config) { c.BroadcastAddr = "" }, "--broadcast-addr"},
			{"dump-path", func(c *Config) { c.DumpPath = "" }, "--dump-path"},
		}
		for _, row := range rows {
			t.Run(row.name, func(t *testing.T) {
				t.Parallel()
				c := validConfig()
				row.mutate(&c)
				assertRejected(t, c, row.flag)
			})
		}
	})

	t.Run("node-id-range accepts the top of the 16-bit space", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.NodeID = math.MaxUint16
		assertValid(t, c)
	})

	t.Run("node-id-range rejects one past the 16-bit space", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.NodeID = math.MaxUint16 + 1
		assertRejected(t, c, "--node-id", "65536")
	})

	t.Run("positive-values rejects every non-positive numeric knob", func(t *testing.T) {
		t.Parallel()
		rows := []struct {
			name   string
			mutate func(*Config)
			flag   string
		}{
			{"startup gate", func(c *Config) { c.StartupGate = 0 }, "--startup-gate"},
			{"startup gate empty", func(c *Config) { c.StartupGateEmpty = -time.Second }, "--startup-gate-empty"},
			{"wait timeout", func(c *Config) { c.WaitTimeout = 0 }, "--wait-timeout"},
			{"value max bytes", func(c *Config) { c.ValueMaxBytes = 0 }, "--value-max-bytes"},
			{"sync chunk bytes", func(c *Config) { c.SyncChunkBytes = 0 }, "--sync-chunk-bytes"},
			{"sync response rate", func(c *Config) { c.SyncResponseRate = -1 }, "--sync-response-rate"},
			{"hello interval", func(c *Config) { c.HelloInterval = 0 }, "--hello-interval"},
			{"peer timeout", func(c *Config) { c.PeerTimeout = 0 }, "--peer-timeout"},
			{"ack timeout", func(c *Config) { c.AckTimeout = 0 }, "--ack-timeout"},
			{"ack timeout max", func(c *Config) { c.AckTimeoutMax = 0 }, "--ack-timeout-max"},
			{"retransmit interval", func(c *Config) { c.RetransmitInterval = 0 }, "--retransmit-interval"},
			{"sync interval", func(c *Config) { c.SyncInterval = 0 }, "--sync-interval"},
			{"max clock lead", func(c *Config) { c.MaxClockLead = 0 }, "--max-clock-lead"},
			{"dump debounce", func(c *Config) { c.DumpDebounce = 0 }, "--dump-debounce"},
			{"dump max wait", func(c *Config) { c.DumpMaxWait = -time.Second }, "--dump-max-wait"},
			{"drift warn", func(c *Config) { c.DriftWarn = 0 }, "--drift-warn"},
		}
		for _, row := range rows {
			t.Run(row.name, func(t *testing.T) {
				t.Parallel()
				c := validConfig()
				row.mutate(&c)
				assertRejected(t, c, row.flag)
			})
		}
	})

	t.Run("retransmit-floor accepts exactly one millisecond", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.RetransmitInterval = time.Millisecond
		assertValid(t, c)
	})

	t.Run("retransmit-floor rejects 999 microseconds", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.RetransmitInterval = time.Millisecond - time.Microsecond
		assertRejected(t, c, "--retransmit-interval", "1ms")
	})

	t.Run("sync-response-cap accepts the cap", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.SyncResponseRate = fixtureSyncResponseCap
		assertValid(t, c)
	})

	t.Run("sync-response-cap rejects one above the cap", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.SyncResponseRate = fixtureSyncResponseCap + 1
		assertRejected(t, c, "--sync-response-rate", "1000000000")
	})

	t.Run("sync-chunk-ceiling accepts the datagram ceiling", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.SyncChunkBytes = repl.MaxSyncChunkBytes
		assertValid(t, c)
	})

	t.Run("sync-chunk-ceiling rejects one byte over", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.SyncChunkBytes = repl.MaxSyncChunkBytes + 1
		assertRejected(t, c, "--sync-chunk-bytes", "sync-chunk-bytes exceeds the datagram ceiling")
	})

	t.Run("datagram-ceiling accepts the largest fitting value", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.ValueMaxBytes = fixtureMaxValueBytes
		assertValid(t, c)
	})

	t.Run("datagram-ceiling rejects one byte more", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.ValueMaxBytes = fixtureMaxValueBytes + 1
		assertRejected(t, c, "datagram ceiling", "--value-max-bytes=64409", "65507")
	})

	t.Run("peer-vs-hello rejects equality", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.HelloInterval = time.Second
		c.PeerTimeout = time.Second
		assertRejected(t, c, "--peer-timeout", "--hello-interval")
	})

	t.Run("peer-vs-hello accepts a greater peer timeout", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.HelloInterval = time.Second
		c.PeerTimeout = time.Second + time.Millisecond
		assertValid(t, c)
	})

	t.Run("ack-max accepts equality", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.AckTimeout = 5 * time.Second
		c.AckTimeoutMax = 5 * time.Second
		assertValid(t, c)
	})

	t.Run("ack-max rejects a greater ack timeout", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.AckTimeout = 6 * time.Second
		c.AckTimeoutMax = 5 * time.Second
		assertRejected(t, c, "--ack-timeout", "--ack-timeout-max")
	})

	t.Run("drift-vs-lead rejects equality at millisecond resolution", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.DriftWarn = 2 * time.Minute
		c.MaxClockLead = 2 * time.Minute
		assertRejected(t, c, "--drift-warn", "--max-clock-lead")
	})

	t.Run("drift-vs-lead accepts one millisecond more", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.DriftWarn = 2 * time.Minute
		c.MaxClockLead = 2*time.Minute + time.Millisecond
		assertValid(t, c)
	})

	t.Run("drift-vs-lead rejects one millisecond less", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.DriftWarn = 2 * time.Minute
		c.MaxClockLead = 2*time.Minute - time.Millisecond
		assertRejected(t, c, "--drift-warn", "--max-clock-lead")
	})

	t.Run("drift-vs-lead rejects a drift warn near the duration maximum", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.DriftWarn = math.MaxInt64
		c.MaxClockLead = 2 * time.Minute
		// The relation's addition form overflows here (MaxInt64 + 1ms wraps
		// negative), so this row pins the overflow-free subtraction: the
		// documented `max_clock_lead >= drift_warn + 1ms` must still reject.
		assertRejected(t, c, "--drift-warn", "--max-clock-lead")
	})

	t.Run("metrics-cert-triple accepts none set", func(t *testing.T) {
		t.Parallel()
		assertValid(t, validConfig())
	})

	t.Run("metrics-cert-triple accepts all three set", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.MetricsCert = "/etc/sup-kv/server.crt"
		c.MetricsKey = "/etc/sup-kv/server.key"
		c.MetricsClientCA = "/etc/sup-kv/ca.crt"
		assertValid(t, c)
	})

	t.Run("metrics-cert-triple rejects a partial set naming all three envs", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.MetricsCert = "/etc/sup-kv/server.crt"
		assertRejected(t, c, "SUPKV_METRICS_CERT", "SUPKV_METRICS_KEY", "SUPKV_METRICS_CLIENT_CA")
	})

	t.Run("address-shape accepts host:port for every address", func(t *testing.T) {
		t.Parallel()
		assertValid(t, validConfig())
	})

	t.Run("address-shape rejects a missing port", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.BindAddr = "127.0.0.1"
		assertRejected(t, c, "--bind-addr")
	})

	t.Run("address-shape rejects a non-numeric port", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.GRPCAddr = "127.0.0.1:http"
		assertRejected(t, c, "--grpc-addr")
	})

	t.Run("address-shape rejects an explicitly empty gRPC address", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.GRPCAddr = ""
		// An empty --grpc-addr must never reach net.Listen: it binds the
		// wildcard address on an ephemeral port, exposing the unauthenticated
		// API to every interface. The rejection names the flag and the default
		// the operator meant to keep.
		assertRejected(t, c, "--grpc-addr", defaultGRPCAddr)
	})

	t.Run("address-shape rejects a port outside 0..65535", func(t *testing.T) {
		t.Parallel()
		rows := []struct {
			name   string
			mutate func(*Config)
			flag   string
		}{
			{"99999", func(c *Config) { c.BindAddr = "127.0.0.1:99999" }, "--bind-addr"},
			{"negative", func(c *Config) { c.BroadcastAddr = "127.255.255.255:-1" }, "--broadcast-addr"},
			{"65536", func(c *Config) { c.GRPCAddr = ":65536" }, "--grpc-addr"},
		}
		for _, row := range rows {
			t.Run(row.name, func(t *testing.T) {
				t.Parallel()
				c := validConfig()
				row.mutate(&c)
				assertRejected(t, c, row.flag, "outside 0..65535")
			})
		}
	})

	t.Run("address-shape rejects port 0 on the broadcast destination", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		// A broadcast address is a destination, not a bind: port 0 can never
		// be sent to, so the node would start mute.
		c.BroadcastAddr = "127.255.255.255:0"
		assertRejected(t, c, "--broadcast-addr", "can never be sent to")
	})

	t.Run("address-shape accepts port 0 on every listener", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		// Tests and operators use ephemeral binds: port 0 is legal wherever
		// the address is a listener, including the optional metrics endpoint.
		c.BindAddr = "127.0.0.1:0"
		c.GRPCAddr = "127.0.0.1:0"
		c.MetricsAddr = "127.0.0.1:0"
		assertValid(t, c)
	})

	t.Run("address-shape skips an unset metrics address", func(t *testing.T) {
		t.Parallel()
		c := validConfig()
		c.MetricsAddr = ""
		assertValid(t, c)
	})
}

// TestRules pins the startup-check catalog: the exact length, names and order,
// with a non-nil check for every entry. Adding, renaming or reordering a rule
// therefore fails here first, and the `subtest` column records the TestValidate
// subtest that owns each rule — a rule cannot be added without one.
func TestRules(t *testing.T) {
	t.Parallel()

	want := []struct {
		name    string
		subtest string
	}{
		{"required-knobs", "required-knobs names every missing knob in one pass"},
		{"node-id-range", "node-id-range rejects one past the 16-bit space"},
		{"positive-values", "positive-values rejects every non-positive numeric knob"},
		{"retransmit-floor", "retransmit-floor rejects 999 microseconds"},
		{"sync-response-cap", "sync-response-cap rejects one above the cap"},
		{"sync-chunk-ceiling", "sync-chunk-ceiling rejects one byte over"},
		{"datagram-ceiling", "datagram-ceiling rejects one byte more"},
		{"peer-vs-hello", "peer-vs-hello rejects equality"},
		{"ack-max", "ack-max rejects a greater ack timeout"},
		{"drift-vs-lead", "drift-vs-lead rejects equality at millisecond resolution"},
		{"metrics-cert-triple", "metrics-cert-triple rejects a partial set naming all three envs"},
		{"address-shape", "address-shape rejects a missing port"},
	}
	got := rules()
	if len(got) != len(want) {
		t.Fatalf("len(rules()) = %d, want %d — a rule needs its TestValidate subtest and this row",
			len(got), len(want))
	}
	for i, r := range got {
		if r.name != want[i].name {
			t.Fatalf("rules()[%d].name = %q, want %q", i, r.name, want[i].name)
		}
		if r.check == nil {
			t.Fatalf("rule %q has a nil check", r.name)
		}
		if want[i].subtest == "" {
			t.Fatalf("rule %q has no owning subtest recorded", r.name)
		}
	}
}

// TestMaxSyncResponseRateExport pins the exported core cap to the literal the
// boundary rows use, so the validation rule and the core's clamp cannot drift.
func TestMaxSyncResponseRateExport(t *testing.T) {
	t.Parallel()
	if repl.MaxSyncResponseRate != fixtureSyncResponseCap {
		t.Fatalf("repl.MaxSyncResponseRate = %d, want the documented %d/s cap",
			repl.MaxSyncResponseRate, fixtureSyncResponseCap)
	}
}

// fixtureSyncResponseCap is the top of the sync-response-rate domain: one
// second divided by the rate must stay a non-zero interval, so 1e9/s is the
// largest rate whose spacing does not truncate to zero (D-15, P4-WR-02).
const fixtureSyncResponseCap = 1_000_000_000

// fixtureMaxValueBytes is the largest value limit whose maximal-key record
// still encodes into one datagram: key 1 024 + value 64 408 is exactly the
// 65 507-byte UDP payload, so the datagram rule's boundary is `>`.
const fixtureMaxValueBytes = 64408
