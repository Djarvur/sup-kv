// Package metrics is the sup-kv metrics/debug endpoint adapter: the repo's
// only HTTP-serving code, rendering the replication and persistence counter
// seams (the 06-03 accessors) as the classic Prometheus text format (version
// 0.0.4) and assembling the one opt-in listener's whole debug surface —
// /metrics together with /debug/vars and /debug/pprof/* (D-03), optionally
// behind client-certificate mTLS (D-02). The adapter owns rendering and
// surface assembly only: values come from the snapshot accessors a caller
// supplies (a closed Sources set, no registration lifecycle), and every
// decision — what counts, when state changes — stays in the packages those
// accessors read. It owns no listener lifecycle either: main binds, serves
// and shuts the listener down inside the D-19 order.
package metrics

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"expvar"
	"fmt"
	"io"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/repl"
	"github.com/Djarvur/sup-kv/internal/port"
	"github.com/Djarvur/sup-kv/internal/udprepl"
)

// FamilyType is a metric family's exposition type — the two kinds the sup-kv
// family set needs (D-04 deliberately scopes the exporter to counters and
// gauges; histograms would be the revisit trigger).
type FamilyType string

// The exposition types Render emits.
const (
	Counter FamilyType = "counter"
	Gauge   FamilyType = "gauge"
)

// The label names of the two labeled families — the only labels the sup-kv
// exposition defines, and their values are fixed enums (bounded cardinality
// by construction).
const (
	labelType    = "type"
	labelClass   = "class"
	labelOutcome = "outcome"
)

// errNoClientCertPEM reports a client-CA file whose bytes hold no PEM
// certificate — wrapped with the file path at the failure site.
var errNoClientCertPEM = errors.New("client CA file contains no PEM certificate")

// Label is one sample label. Label values are the fixed class/type enums
// only (bounded cardinality) — key names, record values and other user data
// never become labels (the OBS-02 prohibitions).
type Label struct {
	Name  string
	Value string
}

// Sample is one time series value under its family's label set.
type Sample struct {
	Labels []Label
	Value  float64
}

// Family is one metric family: name, type and help with every sample the
// exposition renders under it. Construction order carries no meaning —
// Render sorts families and labels in the writer.
type Family struct {
	Name    string
	Type    FamilyType
	Help    string
	Samples []Sample
}

// contentType is the exposition's Content-Type: the classic text format
// version 0.0.4 with the conventional utf-8 charset parameter.
const contentType = "text/plain; version=0.0.4; charset=utf-8"

// Render writes families in the classic Prometheus text format (version
// 0.0.4): family names sorted, exactly one # HELP and one # TYPE line per
// name emitted before that name's first sample, each sample's labels sorted
// by label name, values as Go ParseFloat floats, the final line
// newline-terminated. HELP escapes backslash and line feed; label values
// escape backslash, double quote and line feed (the exposition rules —
// double quotes pass through in HELP unescaped). Sorting happens here, in
// the writer: call sites construct families in whatever order they like,
// which is what keeps scrape diffs stable (D-04).
func Render(w io.Writer, families []Family) error {
	sorted := slices.Clone(families)
	slices.SortFunc(sorted, func(a, b Family) int { return strings.Compare(a.Name, b.Name) })
	var b strings.Builder
	lastName := ""
	for _, f := range sorted {
		if f.Name != lastName {
			b.WriteString("# HELP ")
			b.WriteString(f.Name)
			b.WriteString(" ")
			b.WriteString(escapeHelp(f.Help))
			b.WriteString("\n# TYPE ")
			b.WriteString(f.Name)
			b.WriteString(" ")
			b.WriteString(string(f.Type))
			b.WriteString("\n")
			lastName = f.Name
		}
		for _, s := range f.Samples {
			writeSample(&b, f.Name, s)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// writeSample renders one sample line: the family name, the sorted label
// set (a leading comma between labels, none before the brace when the set
// is empty) and the value.
func writeSample(b *strings.Builder, name string, s Sample) {
	b.WriteString(name)
	labels := slices.Clone(s.Labels)
	slices.SortFunc(labels, func(x, y Label) int { return strings.Compare(x.Name, y.Name) })
	if len(labels) > 0 {
		b.WriteString("{")
		for i, l := range labels {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(l.Name)
			b.WriteString("=\"")
			b.WriteString(escapeLabelValue(l.Value))
			b.WriteString("\"")
		}
		b.WriteString("}")
	}
	b.WriteString(" ")
	b.WriteString(strconv.FormatFloat(s.Value, 'g', -1, 64))
	b.WriteString("\n")
}

// escapeHelp applies the HELP docstring escapes: backslash doubles and line
// feed becomes a literal \n — the only two the exposition defines for HELP,
// so a double quote passes through unescaped.
func escapeHelp(s string) string {
	return strings.NewReplacer(`\`, `\\`, "\n", `\n`).Replace(s)
}

// escapeLabelValue applies the label-value escapes: backslash doubles,
// double quote gains its backslash and line feed becomes a literal \n.
func escapeLabelValue(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(s)
}

// NewHandler returns the /metrics handler over snapshot: every GET renders
// the snapshot's families fresh — the exporter is a pure formatter over the
// supplied accessors, with no registry and no caching of its own. Any other
// method is a 405; the Content-Type header names the exact exposition
// format so scrapers never guess.
func NewHandler(snapshot func() []Family) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", contentType)
		_ = Render(w, snapshot()) // a vanished scraper mid-write is not an error worth acting on.
	})
}

// Sources is the closed set of value accessors the exporter snapshots (the
// 06-03 counter seams; D-04's "no registry" reading made structural): main
// supplies one function per seam and Snapshot reads each once per scrape.
// There is no registration lifecycle — the family set is fixed, and adding
// a family means editing Snapshot, not calling a registry at init time.
type Sources struct {
	Discards      func() repl.DiscardStats
	KnownPeers    func() uint64
	Unacked       func() repl.UnackedStats
	Sent          func() udprepl.SentByType
	Retransmitted func() uint64
	MergeStats    func() repl.MergeStats
	DumpStats     func() persist.DumpStats
	LoadStats     func() persist.LoadStats

	// The XPORT-07 emission set (03-12): the paced-datagram counter, the
	// total send-queue gauge, the egress overflow drops and the receive-path
	// error count — adapter accessors over the core's post-event mirrors.
	Paced         func() uint64
	SendQueue     func() int
	EgressDropped func() uint64
	ReadErrors    func() uint64
}

// Snapshot reads every source once and returns the whole family table: the
// supkv_ families over the supplied seams (D-04/D-05's final names —
// counters carry _total, units are base seconds/bytes, the only labels are
// the fixed class and type enums) plus the go_*/process_* runtime families
// read here, at scrape time, from the runtime and the kernel. A zero-state
// snapshot renders every family with zero-valued samples — an idle node
// still publishes its full surface.
func Snapshot(s Sources) []Family {
	families := replicationFamilies(s.Unacked(), s.KnownPeers(), s.Retransmitted(), s.MergeStats())
	families = append(families, pacingFamilies(s.Paced(), s.SendQueue(), s.EgressDropped(), s.ReadErrors())...)
	families = append(families, trafficFamilies(s.Discards(), s.Sent())...)
	families = append(families, dumpFamilies(s.DumpStats())...)
	load := s.LoadStats()
	families = append(families, loadFamilies(load)...)
	families = append(families, discardClassFamilies(load.Decoded)...)
	return append(families, runtimeFamilies()...)
}

// loadFamilies builds the startup load-outcome family (PERSIST-02): the four
// outcome labels always render, with the loaded outcome at 1 and the others
// at 0; an unloaded engine reports all four at 0.
func loadFamilies(stats persist.LoadStats) []Family {
	samples := []Sample{
		{Labels: []Label{{Name: labelOutcome, Value: "prompt"}}},
		{Labels: []Label{{Name: labelOutcome, Value: "main"}}},
		{Labels: []Label{{Name: labelOutcome, Value: "backup"}}},
		{Labels: []Label{{Name: labelOutcome, Value: "unusable"}}},
	}
	if stats.Loaded {
		switch stats.Outcome {
		case port.LoadPrompt:
			samples[0].Value = 1
		case port.LoadMain:
			samples[1].Value = 1
		case port.LoadBackup:
			samples[2].Value = 1
		case port.LoadUnusable:
			samples[3].Value = 1
		}
	}
	return []Family{{
		Name: "supkv_dump_load_total", Type: Counter,
		Help:    "Startup dump loads by outcome.",
		Samples: samples,
	}}
}

// discardClassFamilies builds the load-time discard family (PERSIST-10): the
// four decode classes always render from the accepted generation's census, so
// an idle node publishes zero-valued samples rather than omitting the family.
func discardClassFamilies(decoded persist.DecodeStats) []Family {
	return []Family{{
		Name: "supkv_dump_records_discarded_total", Type: Counter,
		Help: "Dumped records discarded at load, by classification.",
		Samples: []Sample{
			{Labels: []Label{{Name: labelClass, Value: "checksum"}}, Value: float64(decoded.Checksum)},
			{Labels: []Label{{Name: labelClass, Value: "version"}}, Value: float64(decoded.Version)},
			{Labels: []Label{{Name: labelClass, Value: "value_limit"}}, Value: float64(decoded.ValueLimit)},
			{Labels: []Label{{Name: labelClass, Value: "malformed"}}, Value: float64(decoded.Malformed)},
		},
	}}
}

// replicationFamilies builds the peer/unacked/retransmit/merge families (the
// unlabeled half of the OBS-01 replication surface) over one reading of the
// replication seams.
func replicationFamilies(
	un repl.UnackedStats,
	peers uint64,
	retr uint64,
	merged repl.MergeStats,
) []Family {
	return []Family{
		{
			Name: "supkv_replication_unacked_records", Type: Gauge,
			Help: "Replication records awaiting acknowledgement.",
			Samples: []Sample{
				{Value: float64(un.Count)},
			},
		},
		{
			Name: "supkv_replication_unacked_oldest_age_seconds", Type: Gauge,
			Help: "Age of the oldest unacknowledged replication record in seconds.",
			Samples: []Sample{
				{Value: un.OldestAge.Seconds()},
			},
		},
		{
			Name: "supkv_replication_peers", Type: Gauge,
			Help: "Currently known replication peers.",
			Samples: []Sample{
				{Value: float64(peers)},
			},
		},
		{
			Name: "supkv_replication_datagrams_retransmitted_total", Type: Counter,
			Help: "Replication datagram retransmissions.",
			Samples: []Sample{
				{Value: float64(retr)},
			},
		},
		{
			Name: "supkv_merges_conflicts_total", Type: Counter,
			Help: "Remote merges that displaced a stored record.",
			Samples: []Sample{
				{Value: float64(merged.Conflicts)},
			},
		},
		{
			Name: "supkv_clock_drift_warnings_total", Type: Counter,
			Help: "Remote timestamps absorbed beyond the drift-warn threshold.",
			Samples: []Sample{
				{Value: float64(merged.DriftWarnings)},
			},
		},
	}
}

// pacingFamilies builds the XPORT-07 emission families: the paced-datagram
// counter (counted at the Outbox handoff, never at enqueue), the total
// send-queue gauge (all three core queues), the egress overflow drops and the
// receive-path errors — one reading of each seam per scrape.
func pacingFamilies(paced uint64, sendQueue int, egressDropped, readErrors uint64) []Family {
	return []Family{
		{
			Name: "supkv_replication_datagrams_paced_total", Type: Counter,
			Help: "Replication datagrams emitted by the listing, record and egress pacers.",
			Samples: []Sample{
				{Value: float64(paced)},
			},
		},
		{
			Name: "supkv_replication_send_queue", Type: Gauge,
			Help: "Replication datagrams waiting to emit across every core queue.",
			Samples: []Sample{
				{Value: float64(sendQueue)},
			},
		},
		{
			Name: "supkv_replication_egress_dropped_total", Type: Counter,
			Help: "Egress responses dropped by the queue cap.",
			Samples: []Sample{
				{Value: float64(egressDropped)},
			},
		},
		{
			Name: "supkv_replication_read_errors_total", Type: Counter,
			Help: "Receive-path errors the read loop survived.",
			Samples: []Sample{
				{Value: float64(readErrors)},
			},
		},
	}
}

// trafficFamilies builds the two labeled families — outbound datagrams by
// message type and inbound discards by classification (the OBS-02 surface in
// full) — and always publishes the whole label vocabulary: a zero count is
// information, not absence.
func trafficFamilies(disc repl.DiscardStats, sentBy udprepl.SentByType) []Family {
	return []Family{
		{
			Name: "supkv_replication_datagrams_sent_total", Type: Counter,
			Help: "Replication datagrams sent, by message type.",
			Samples: []Sample{
				{Labels: []Label{{Name: labelType, Value: "hello"}}, Value: float64(sentBy.Hello)},
				{Labels: []Label{{Name: labelType, Value: "data"}}, Value: float64(sentBy.Data)},
				{Labels: []Label{{Name: labelType, Value: "ack"}}, Value: float64(sentBy.Ack)},
				{Labels: []Label{{Name: labelType, Value: "data_request"}}, Value: float64(sentBy.DataRequest)},
				{Labels: []Label{{Name: labelType, Value: "sync"}}, Value: float64(sentBy.Sync)},
			},
		},
		{
			Name: "supkv_datagrams_discarded_total", Type: Counter,
			Help: "Inbound replication datagrams discarded, by classification.",
			Samples: []Sample{
				{Labels: []Label{{Name: labelClass, Value: "bad_crc"}}, Value: float64(disc.BadCRC)},
				{Labels: []Label{{Name: labelClass, Value: "undecodable"}}, Value: float64(disc.Undecodable)},
				{Labels: []Label{{Name: labelClass, Value: "bad_proto_version"}}, Value: float64(disc.BadProtoVersion)},
				{Labels: []Label{{Name: labelClass, Value: "own_broadcast"}}, Value: float64(disc.OwnEcho)},
				{Labels: []Label{{Name: labelClass, Value: "duplicate_node_id"}}, Value: float64(disc.DuplicateNodeID)},
				{Labels: []Label{{Name: labelClass, Value: "empty_key"}}, Value: float64(disc.EmptyKey)},
				{Labels: []Label{{Name: labelClass, Value: "key_too_long"}}, Value: float64(disc.KeyTooLong)},
				{Labels: []Label{{Name: labelClass, Value: "tombstone_with_data"}}, Value: float64(disc.TombstoneWithData)},
				{Labels: []Label{{Name: labelClass, Value: "bad_checksum"}}, Value: float64(disc.BadChecksum)},
				{Labels: []Label{{Name: labelClass, Value: "bad_version"}}, Value: float64(disc.BadVersion)},
				{Labels: []Label{{Name: labelClass, Value: "bad_sync_header"}}, Value: float64(disc.BadSyncHeader)},
			},
		},
	}
}

// dumpFamilies builds the persistence-cycle families over one reading of the
// engine's DumpStats seam: the last duration, the attempt and error totals,
// and the failure-class split (PERSIST-05). The class family always renders
// all three values; "other" is the residual between the error total and the
// two classed counters (Errors is their superset), clamped so a scrape
// racing a dump's counter updates never renders a negative sample.
func dumpFamilies(stats persist.DumpStats) []Family {
	other := uint64(0)
	if classified := stats.DiskFull + stats.Permission; stats.Errors > classified {
		other = stats.Errors - classified
	}
	return []Family{
		{
			Name: "supkv_dumps_duration_seconds", Type: Gauge,
			Help: "Duration of the last dump's encode and save in seconds.",
			Samples: []Sample{
				{Value: stats.DurationSeconds},
			},
		},
		{
			Name: "supkv_dumps_total", Type: Counter,
			Help: "Dump attempts.",
			Samples: []Sample{
				{Value: float64(stats.Total)},
			},
		},
		{
			Name: "supkv_dumps_errors_total", Type: Counter,
			Help: "Failed dumps.",
			Samples: []Sample{
				{Value: float64(stats.Errors)},
			},
		},
		{
			Name: "supkv_dumps_failures_total", Type: Counter,
			Help: "Failed dump attempts, by failure class.",
			Samples: []Sample{
				{Labels: []Label{{Name: labelClass, Value: "disk_full"}}, Value: float64(stats.DiskFull)},
				{Labels: []Label{{Name: labelClass, Value: "permission"}}, Value: float64(stats.Permission)},
				{Labels: []Label{{Name: labelClass, Value: "other"}}, Value: float64(other)},
			},
		},
	}
}

// runtimeFamilies reads the go_*/process_* runtime families (06-RESEARCH A2's
// minimal D-04 enumeration) at scrape time — fresh gauges of the process
// itself, never cached.
func runtimeFamilies() []Family {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	cpu, rss := processUsage()
	return []Family{
		{
			Name: "go_goroutines", Type: Gauge,
			Help: "Number of goroutines that currently exist.",
			Samples: []Sample{
				{Value: float64(runtime.NumGoroutine())},
			},
		},
		{
			Name: "go_memstats_alloc_bytes", Type: Gauge,
			Help: "Number of bytes allocated and still in use.",
			Samples: []Sample{
				{Value: float64(ms.Alloc)},
			},
		},
		{
			Name: "go_memstats_sys_bytes", Type: Gauge,
			Help: "Number of bytes obtained from system.",
			Samples: []Sample{
				{Value: float64(ms.Sys)},
			},
		},
		{
			Name: "process_cpu_seconds_total", Type: Counter,
			Help: "Total user and system CPU time spent in seconds.",
			Samples: []Sample{
				{Value: cpu},
			},
		},
		{
			Name: "process_resident_memory_bytes", Type: Gauge,
			Help: "Resident memory size in bytes.",
			Samples: []Sample{
				{Value: rss},
			},
		},
	}
}

// processUsage reads this process's CPU time and resident set size through
// [syscall.Getrusage] — Linux and darwin both provide it (06-RESEARCH A2's
// minimal process_* enumeration under D-04). CPU is user+system seconds;
// ru_maxrss is KiB on Linux and bytes on darwin, normalized here to bytes.
func processUsage() (cpuSeconds, rssBytes float64) {
	var ru syscall.Rusage
	err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	if err != nil {
		return 0, 0
	}
	sec := func(t syscall.Timeval) float64 { return float64(t.Sec) + float64(t.Usec)/1e6 }
	rss := float64(ru.Maxrss)
	if rss > 0 && runtime.GOOS != "darwin" {
		rss *= 1024
	}
	return sec(ru.Utime) + sec(ru.Stime), rss
}

// NewMux assembles the one listener's whole debug surface (D-03): /metrics
// — the exporter rendering a fresh Snapshot per request — together with
// /debug/vars through [expvar.Handler] and the /debug/pprof/* routes: the
// standard index/cmdline/profile/symbol/trace endpoints plus every named
// profile endpoint through [pprof.Handler]. Every package is imported by
// name and every handler is mounted explicitly on this fresh mux — neither
// expvar nor net/http/pprof is ever blank-imported, so nothing registers on
// [http.DefaultServeMux] (Pitfall 4). The whole surface rides the one
// listener, so the optional mTLS of D-02 covers all of it with one story.
func NewMux(s Sources) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/metrics", NewHandler(func() []Family { return Snapshot(s) }))
	mux.Handle("/debug/vars", expvar.Handler())
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	return mux
}

// NewTLSConfig builds the metrics listener's server-side TLS configuration
// (D-02): the certificate/key pair from disk, the client CA pool from the CA
// file, client certificates required and verified, and an explicit TLS 1.2
// floor. No custom cipher configuration — the crypto/tls defaults (the
// T-6-08 mitigation).
func NewTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load key pair: %w", err)
	}
	caPEM, err := os.ReadFile(caFile) //nolint:gosec // G304: the path derives from the operator's SUPKV_METRICS_CLIENT_CA env, never caller input.
	if err != nil {
		return nil, fmt.Errorf("read client CA: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("%s: %w", caFile, errNoClientCertPEM)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
