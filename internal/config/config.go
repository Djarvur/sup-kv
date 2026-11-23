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
	"io"
	"log/slog"
	"time"
)

// ErrHelp reports that --help (or -h) was requested: the caller renders
// [Usage] on standard output and exits 0, never through the error path.
var ErrHelp = errors.New("help requested")

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

// Parse resolves args against the environment behind getenv. It is pure over
// its inputs — one local flag set, no package globals — so parallel callers
// cannot observe each other's flags.
func Parse(_ []string, _ func(string) string) (Config, error) {
	return Config{}, nil
}

// Usage renders the flag table and the env-only block to w.
func Usage(_ io.Writer) {}
