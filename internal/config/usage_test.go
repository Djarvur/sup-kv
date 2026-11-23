package config_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Djarvur/sup-kv/internal/config"
)

// flagEnvTwins is the OBSV-02 usage contract: every flag knob names its SUPKV_
// env twin in the rendered help, so an operator can discover the environment
// form of each flag without reading the source.
var flagEnvTwins = []struct {
	flag string
	env  string
}{
	{"node-id", "SUPKV_NODE_ID"},
	{"bind-addr", "SUPKV_BIND_ADDR"},
	{"broadcast-addr", "SUPKV_BROADCAST_ADDR"},
	{"dump-path", "SUPKV_DUMP_PATH"},
	{"grpc-addr", "SUPKV_GRPC_ADDR"},
	{"metrics-addr", "SUPKV_METRICS_ADDR"},
	{"startup-gate", "SUPKV_STARTUP_GATE"},
	{"startup-gate-empty", "SUPKV_STARTUP_GATE_EMPTY"},
	{"wait-timeout", "SUPKV_WAIT_TIMEOUT"},
	{"value-max-bytes", "SUPKV_VALUE_MAX_BYTES"},
	{"sync-chunk-bytes", "SUPKV_SYNC_CHUNK_BYTES"},
	{"sync-response-rate", "SUPKV_SYNC_RESPONSE_RATE"},
	{"hello-interval", "SUPKV_HELLO_INTERVAL"},
	{"peer-timeout", "SUPKV_PEER_TIMEOUT"},
	{"ack-timeout", "SUPKV_ACK_TIMEOUT"},
	{"ack-timeout-max", "SUPKV_ACK_TIMEOUT_MAX"},
	{"retransmit-interval", "SUPKV_RETRANSMIT_INTERVAL"},
	{"sync-interval", "SUPKV_SYNC_INTERVAL"},
	{"max-clock-lead", "SUPKV_MAX_CLOCK_LEAD"},
	{"dump-debounce", "SUPKV_DUMP_DEBOUNCE"},
	{"dump-max-wait", "SUPKV_DUMP_MAX_WAIT"},
	{"drift-warn", "SUPKV_DRIFT_WARN"},
}

// envOnlyNames is the five-knob list the plan pins as the usage contract: the
// logging and metrics-certificate knobs configure the process's own output and
// identity, so they have no flag twin and must still be discoverable.
var envOnlyNames = []string{
	"SUPKV_LOG_LEVEL",
	"SUPKV_LOG_FORMAT",
	"SUPKV_METRICS_CERT",
	"SUPKV_METRICS_KEY",
	"SUPKV_METRICS_CLIENT_CA",
}

// TestUsage pins the rendered help: every flag appears with its env twin, and
// the env-only block names all five flagless knobs under its own header.
func TestUsage(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	config.Usage(&buf)
	out := buf.String()

	for _, twin := range flagEnvTwins {
		if !strings.Contains(out, "\n  -"+twin.flag) {
			t.Fatalf("Usage output does not list the -%s flag:\n%s", twin.flag, out)
		}
		if !strings.Contains(out, "env "+twin.env) {
			t.Fatalf("Usage output does not name the %s env twin of -%s:\n%s", twin.env, twin.flag, out)
		}
	}
	if !strings.Contains(out, "\n  -version") {
		t.Fatalf("Usage output does not list the -version flag:\n%s", out)
	}
	if !strings.Contains(out, "Environment-only (no flag):") {
		t.Fatalf("Usage output does not carry the %q header:\n%s", "Environment-only (no flag):", out)
	}
	for _, name := range envOnlyNames {
		if !strings.Contains(out, name) {
			t.Fatalf("Usage output does not name the env-only variable %s:\n%s", name, out)
		}
	}
}
