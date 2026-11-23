package config_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Djarvur/sup-kv/internal/config"
)

// envOf returns a getenv seam answering only the one variable — every other
// name is unset, which is exactly the production "an unset or empty env value
// counts as absent" contract.
func envOf(name, value string) func(string) string {
	return func(key string) string {
		if key == name {
			return value
		}
		return ""
	}
}

// TestParse pins the Parse contract: --help yields ErrHelp, --version reports
// ShowVersion before any env error (a broken environment must not block the
// build-identity probe), flag > env > default holds on the observable gRPC
// address, and a malformed env value, an unknown flag and positional tokens
// each return an error naming the offending text.
func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("help returns ErrHelp", func(t *testing.T) {
		t.Parallel()
		_, err := config.Parse([]string{"--help"}, nil)
		if !errors.Is(err, config.ErrHelp) {
			t.Fatalf("Parse(--help) error = %v, want ErrHelp", err)
		}
	})

	t.Run("version wins over a malformed environment", func(t *testing.T) {
		t.Parallel()
		cfg, err := config.Parse([]string{"--version"}, envOf("SUPKV_LOG_LEVEL", "bogus"))
		if err != nil {
			t.Fatalf("Parse(--version) with a malformed SUPKV_LOG_LEVEL error = %v, want nil", err)
		}
		if !cfg.ShowVersion {
			t.Fatal("Parse(--version).ShowVersion = false, want true")
		}
	})

	t.Run("precedence flag over env over default", func(t *testing.T) {
		t.Parallel()
		flagCfg, err := config.Parse([]string{"--grpc-addr=127.0.0.1:7001"}, envOf("SUPKV_GRPC_ADDR", "127.0.0.1:7002"))
		if err != nil {
			t.Fatalf("Parse with the flag and env twins error = %v, want nil", err)
		}
		if flagCfg.GRPCAddr != "127.0.0.1:7001" {
			t.Fatalf("GRPCAddr = %q, want the flag value 127.0.0.1:7001", flagCfg.GRPCAddr)
		}

		envCfg, err := config.Parse(nil, envOf("SUPKV_GRPC_ADDR", "127.0.0.1:7002"))
		if err != nil {
			t.Fatalf("Parse with only the env twin error = %v, want nil", err)
		}
		if envCfg.GRPCAddr != "127.0.0.1:7002" {
			t.Fatalf("GRPCAddr = %q, want the env value 127.0.0.1:7002", envCfg.GRPCAddr)
		}

		defCfg, err := config.Parse(nil, nil)
		if err != nil {
			t.Fatalf("Parse with neither source error = %v, want nil", err)
		}
		if defCfg.GRPCAddr != "127.0.0.1:7901" {
			t.Fatalf("GRPCAddr = %q, want the design default 127.0.0.1:7901", defCfg.GRPCAddr)
		}
	})

	t.Run("an empty env value counts as unset", func(t *testing.T) {
		t.Parallel()
		cfg, err := config.Parse(nil, envOf("SUPKV_GRPC_ADDR", ""))
		if err != nil {
			t.Fatalf("Parse with an empty env twin error = %v, want nil", err)
		}
		if cfg.GRPCAddr != "127.0.0.1:7901" {
			t.Fatalf("GRPCAddr = %q, want the design default 127.0.0.1:7901", cfg.GRPCAddr)
		}
	})

	t.Run("a malformed env value names the variable", func(t *testing.T) {
		t.Parallel()
		_, err := config.Parse(nil, envOf("SUPKV_MAX_CLOCK_LEAD", "not-a-duration"))
		if err == nil {
			t.Fatal("Parse with a malformed SUPKV_MAX_CLOCK_LEAD error = nil, want a named error")
		}
		if !strings.Contains(err.Error(), "SUPKV_MAX_CLOCK_LEAD") {
			t.Fatalf("the malformed env error %q does not name SUPKV_MAX_CLOCK_LEAD", err)
		}
	})

	t.Run("an unknown flag names the token", func(t *testing.T) {
		t.Parallel()
		_, err := config.Parse([]string{"--bogus-flag=x"}, nil)
		if err == nil {
			t.Fatal("Parse(--bogus-flag=x) error = nil, want an unknown-flag error")
		}
		if !strings.Contains(err.Error(), "bogus-flag") {
			t.Fatalf("the unknown-flag error %q does not name bogus-flag", err)
		}
	})

	t.Run("positional arguments are rejected by name", func(t *testing.T) {
		t.Parallel()
		_, err := config.Parse([]string{"stray", "another"}, nil)
		if err == nil {
			t.Fatal("Parse(stray another) error = nil, want a positional-argument error")
		}
		for _, want := range []string{"stray", "another"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("the positional-argument error %q does not name %q", err, want)
			}
		}
	})
}
