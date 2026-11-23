package main

// The D-01 lifecycle-catalog level matrix (OBSV-01, OBSV-06). Every entry
// the catalog mandates — the four Spec-mandated ones included — is exercised
// through the named renderer this package actually wires, against a captured
// slog handler, and asserted at its documented level and message. The level
// is the operator contract: a refactor that demoted an entry to INFO would
// otherwise pass every behavioral test, because levels are never observable
// from the RPC or metrics surface.

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/config"
	"github.com/Djarvur/sup-kv/internal/port"
)

// catalogRecord is one parsed slog JSON record from the captured handler.
type catalogRecord struct {
	Level  string `json:"level"`
	Msg    string `json:"msg"`
	Detail string `json:"detail"`
	Class  string `json:"class"`
}

// TestCatalogLevels calls each catalog renderer directly and asserts exactly
// one record at the documented level with the documented message. The shared
// `Replication adapter` WARN renderer is covered in both shapes the core's
// branches build: the foreign proto_version discard detail and the
// duplicate-node-id configuration-error detail.
//
//nolint:paralleltest // it installs a captured process-global logger and restores it in t.Cleanup
func TestCatalogLevels(t *testing.T) {
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(old) })

	capture := func(t *testing.T, emit func()) []catalogRecord {
		t.Helper()
		buf.Reset()
		emit()
		var records []catalogRecord
		for line := range strings.SplitSeq(buf.String(), "\n") {
			if line == "" {
				continue
			}
			var rec catalogRecord
			err := json.Unmarshal([]byte(line), &rec)
			if err != nil {
				t.Fatalf("captured line %q is not one JSON record: %v", line, err)
			}
			records = append(records, rec)
		}
		return records
	}

	const (
		levelInfo  = "INFO"
		levelWarn  = "WARN"
		levelError = "ERROR"
	)
	rows := []struct {
		name   string
		level  string
		msg    string
		detail string
		class  string
		emit   func()
	}{
		{"startup", levelInfo, "Starting sup-kv", "", "", func() { logStartup(config.Config{}) }},
		{"load prompt", levelInfo, "Dump load", "", "", func() {
			logLoadStats(persist.LoadStats{Outcome: port.LoadPrompt}, 0)
		}},
		{"load backup", levelWarn, "Dump generation fallback", "", "", func() {
			logLoadStats(persist.LoadStats{Outcome: port.LoadBackup}, 0)
		}},
		{"load unusable", levelError, "No usable dump generation", "", "", func() {
			logLoadStats(persist.LoadStats{Outcome: port.LoadUnusable}, 0)
		}},
		{"dump failure", levelError, "Dump failed", "file does not exist", "disk_full", func() {
			logDumpFailure(fs.ErrNotExist, port.FailureDiskFull)
		}},
		{"gate open", levelInfo, "Startup gate opened", "", "", func() { logGateOpen("backstop") }},
		{"peer appeared", levelInfo, "Peer appeared", "", "", func() {
			logPeerEvent(repl.PeerEvent{Kind: repl.PeerAppeared, NodeID: 2})
		}},
		{"peer restarted", levelWarn, "Peer restarted", "", "", func() {
			logPeerEvent(repl.PeerEvent{Kind: repl.PeerRestarted, NodeID: 2})
		}},
		{"peer evicted", levelWarn, "Peer evicted", "", "", func() {
			logPeerEvent(repl.PeerEvent{Kind: repl.PeerEvicted, NodeID: 2})
		}},
		{"clock drift", levelWarn, "Clock drift", "", "", func() {
			logDriftEvent(repl.DriftEvent{NodeID: 2, Lead: time.Minute}, time.Minute)
		}},
		{"adapter foreign proto_version", levelWarn, "Replication adapter", "class bad_proto_version", "", func() {
			logAdapterWarn("sup-kv: discarded inbound datagram: class bad_proto_version from node 2")
		}},
		{"adapter duplicate node_id", levelWarn, "Replication adapter", "configuration error", "", func() {
			logAdapterWarn(`sup-kv: configuration error: node_id 2 announced by another run from handle "n2"`)
		}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			records := capture(t, row.emit)
			if len(records) != 1 {
				t.Fatalf("the %s renderer emitted %d record(s), want exactly 1: %+v", row.name, len(records), records)
			}
			if records[0].Level != row.level {
				t.Fatalf("the %s record level = %q, want %q: %+v", row.name, records[0].Level, row.level, records[0])
			}
			if records[0].Msg != row.msg {
				t.Fatalf("the %s record msg = %q, want %q: %+v", row.name, records[0].Msg, row.msg, records[0])
			}
			if row.detail != "" && !strings.Contains(records[0].Detail, row.detail) {
				t.Fatalf("the %s record detail = %q, want it to carry %q", row.name, records[0].Detail, row.detail)
			}
			if row.class != "" && records[0].Class != row.class {
				t.Fatalf("the %s record class = %q, want %q: %+v", row.name, records[0].Class, row.class, records[0])
			}
		})
	}
}
