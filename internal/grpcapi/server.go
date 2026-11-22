// Package grpcapi is the sup-kv gRPC API adapter: the only gRPC- and
// protobuf-importing API code, mechanical — it maps the SupKV RPCs onto the
// persist.Engine seams and maps the engine's sentinel errors onto their
// pinned RPC codes. It owns no storage decision: versioning, tombstones,
// value limits and dump confirmation live in internal/app — the depguard
// gate (sup-kv-app-no-io) keeps internal/app free of this package's
// imports, and the generated wire code stays in apiproto. Replication
// notification of local writes crosses the LocalNotify seam (D-05: the
// write author notifies explicitly); a nil notify is the legal single-node
// configuration (D-11).
package grpcapi

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"sync/atomic"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
	"github.com/Djarvur/sup-kv/internal/persistfile"
)

// LocalNotify is the post-write replication notification seam: the gRPC
// write handler is "the author" of the phase-4 D-05 explicit-call contract.
// It receives the record the write just installed and whether the write
// requested a wait_for_replication report (track), so the adapter can
// create the waiter tracker synchronously before enqueueing the
// notification (XPORT-12). It reports nil when the notification was
// accepted for the replication loop, or a non-nil error when it was dropped
// (udprepl.ErrRunLoopExited once that loop has exited).
type LocalNotify func(key string, rec storage.Record, track bool) error

// Server implements the apiproto.SupKVServiceServer surface over one persist
// Engine. The UnimplementedSupKVServiceServer embed is mandatory with
// protoc-gen-go-grpc v1.6.2 (forward compatibility): every phase-5 RPC is
// implemented on it today, and the embed is what future wire additions
// would answer Unimplemented with until their plans land.
type Server struct {
	apiproto.UnimplementedSupKVServiceServer

	engine *persist.Engine
	notify LocalNotify
	hub    *WatchHub
	opts   ServerOptions

	// healthSrv is the standard gRPC health server whose empty-service
	// status is bound 1:1 to the gate (API-06, gate.go); it starts at
	// NOT_SERVING and only the gate resolver opens it.
	healthSrv *health.Server

	// gateOpen is the resolved startup-gate state (API-04): closed at
	// construction, opened exactly once by RunGate. Every data handler —
	// reads and writes alike — reads it through gated.
	gateOpen atomic.Bool
}

// NewServer returns the SupKV service over engine. notify may be nil: that
// is the single-node configuration with no replication (D-11), and the
// handlers then skip the post-write notification entirely. hub is the
// WatchHub backing the Watch stream; NewServer attaches the engine to it —
// the hub is constructed first because persist.Open threads the change tap
// through storage.Config — and a nil hub keeps Watch answering
// Unimplemented (the pre-Watch embed behavior). opts carries the
// startup-gate wiring (gate.go) and the wait_for_replication seams
// (wait.go): the gate starts closed with the health endpoint at
// NOT_SERVING for the empty service name, and the caller launches the
// resolver exactly once — go srv.RunGate(ctx) — to open both; a zero or
// negative opts.WaitTimeout normalizes to the 1s design default (D-09,
// A2), and nil wait seams mean zero peers (D-11).
func NewServer(engine *persist.Engine, notify LocalNotify, hub *WatchHub, opts ServerOptions) *Server {
	if hub != nil {
		hub.attach(engine)
	}
	if opts.WaitTimeout <= 0 {
		opts.WaitTimeout = defaultWaitTimeout
	}
	healthSrv := health.NewServer()
	// The health protocol defaults the whole-server status to SERVING; the
	// gate starts closed, so the binding begins at NOT_SERVING before any
	// client can observe the endpoint (API-06).
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	return &Server{
		engine:    engine,
		notify:    notify,
		hub:       hub,
		opts:      opts,
		healthSrv: healthSrv,
	}
}

// Get returns the live record for one key. The startup gate is checked
// first (API-04, D-01): while it is closed every read — like every write —
// answers Unavailable, because a cold node must not answer from a picture
// it does not yet have. Once the gate opens, an absent or tombstoned key is
// the NotFound code, never an empty success; a key a peer's complete
// listing named but whose value has not arrived is Unavailable (rule
// 14.5(6): Engine.Version distinguishes the local tombstone from the
// listed-but-unarrived key, because Get alone hides tombstones). An empty
// key is rejected with InvalidArgument before any engine call (A7), like
// the write handlers — but only past the gate, so a closed node answers
// Unavailable even for an empty key.
//
// Get deliberately does not enforce the key-length ceiling: the write side
// (Set, Delete) and the replication receive path do. Pre-upgrade records
// under keys longer than storage.MaxKeyBytes must stay readable, and Phase
// 5's load-time accounting decides what happens to them (RESEARCH Open
// Question 2: write-side only, Get permissive).
func (s *Server) Get(_ context.Context, req *apiproto.GetRequest) (*apiproto.GetResponse, error) {
	if s.gated() {
		return nil, gateUnavailable()
	}
	if req.GetKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "key must not be empty")
	}
	rec, err := s.engine.Get(req.GetKey())
	if err != nil {
		return nil, mapError(err)
	}
	return &apiproto.GetResponse{Record: recordToProto(req.GetKey(), rec)}, nil
}

// Set writes value under key — the create/update write, or its
// create-if-absent variant — and returns only after the engine confirmed
// the write on disk (PERS-04 inside the RPC): a nil error means a dump
// containing the record completed. The startup gate is checked first
// (API-04): while it is closed every write is Unavailable, full stop —
// before any validation, so even an empty key surfaces Unavailable, not
// InvalidArgument, until the gate opens. An empty key is then rejected
// with InvalidArgument before any engine call (A7: the store itself
// permits empty keys, the API does not), and a key longer than
// storage.MaxKeyBytes is rejected with InvalidArgument before any engine
// call too (XPORT-03: the one key-length ceiling, measured in bytes, also
// enforced on receive and at startup) — create_if_absent rides the same
// single validation path. wait_for_replication asks the
// server to wait after the confirmed write for the replication Acks and
// decorate the response with the D-08 aggregate report (wait.go): the wait
// is bounded by the positive per-request wait_timeout_ms, else the
// operator-tunable default (D-09), and never changes the write outcome —
// it runs strictly after the engine confirmed the write, so no path below
// can turn a confirmed write into an error. Without the flag the report
// fields keep their zero values. The replication notification is
// best-effort (D-05): a dropped notification never fails the RPC, because
// the durable write is already engine-confirmed.
func (s *Server) Set(ctx context.Context, req *apiproto.SetRequest) (*apiproto.SetResponse, error) {
	if s.gated() {
		return nil, gateUnavailable()
	}
	if req.GetKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "key must not be empty")
	}
	if len(req.GetKey()) > storage.MaxKeyBytes {
		return nil, keyTooLongError(len(req.GetKey()))
	}
	var (
		ver version.RecordVersion
		err error
	)
	if req.GetCreateIfAbsent() {
		ver, _, err = s.engine.SetIfAbsent(req.GetKey(), req.GetValue())
	} else {
		ver, _, err = s.engine.Set(req.GetKey(), req.GetValue())
	}
	if err != nil {
		return nil, mapError(err)
	}
	s.notifyLocalWrite(req.GetKey(), ver, req.GetValue(), false, req.GetWaitForReplication())
	resp := &apiproto.SetResponse{Version: ver[:]}
	if req.GetWaitForReplication() {
		timeout := s.waitTimeout(req.GetWaitTimeoutMs())
		resp.Acked, resp.AckedCount, resp.TotalPeers = s.waitForReplication(ctx, req.GetKey(), ver, timeout)
	}
	return resp, nil
}

// Delete writes a fresh-version tombstone for one key and returns only
// after the engine confirmed the tombstone on disk (PERS-04 inside the
// RPC). The startup gate is checked first, exactly like Set (API-04: a
// delete is a write). A missing or already-tombstoned key surfaces the
// engine's NotFound through the shared mapper — a delete of nothing
// happened. An empty key is rejected with InvalidArgument before any
// engine call, like Set, as is a key longer than storage.MaxKeyBytes
// (XPORT-03's single ceiling — a tombstone for an unwritable key would be
// its own unsendable record). The wait fields carry the same meaning as on Set
// (API-05: a delete is a write): wait_for_replication waits for the
// tombstone's replication Acks and decorates the response with the D-08
// aggregate report, bounded by the positive per-request wait_timeout_ms
// else the operator default (D-09), never changing the write outcome;
// without the flag the report fields keep their zero values.
func (s *Server) Delete(ctx context.Context, req *apiproto.DeleteRequest) (*apiproto.DeleteResponse, error) {
	if s.gated() {
		return nil, gateUnavailable()
	}
	if req.GetKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "key must not be empty")
	}
	if len(req.GetKey()) > storage.MaxKeyBytes {
		return nil, keyTooLongError(len(req.GetKey()))
	}
	ver, _, err := s.engine.Delete(req.GetKey())
	if err != nil {
		return nil, mapError(err)
	}
	s.notifyLocalWrite(req.GetKey(), ver, nil, true, req.GetWaitForReplication())
	resp := &apiproto.DeleteResponse{Version: ver[:]}
	if req.GetWaitForReplication() {
		timeout := s.waitTimeout(req.GetWaitTimeoutMs())
		resp.Acked, resp.AckedCount, resp.TotalPeers = s.waitForReplication(ctx, req.GetKey(), ver, timeout)
	}
	return resp, nil
}

// List returns the sorted live (non-tombstoned) keys under prefix in one
// response (D-12): the empty prefix matches every key (D-15), tombstones
// are never listed — List reflects live state, not history (D-14) — and
// the response carries keys only; values are fetched via Get (D-13). The
// startup gate is checked first like every other data RPC (API-04, D-01):
// a closed node answers Unavailable rather than an empty list, which would
// lie about a state it has not yet learned. The engine's Records() snapshot
// is tombstone-inclusive by contract, so the Deleted filter here is
// adapter-side. The scale envelope — tens of thousands of keys — fits the
// gRPC 4 MiB default send limit (research A6): grpc defaults are kept, and
// grpc.MaxSendMsgSize is the documented growth knob if key names ever grow
// the envelope — pagination is deliberately not hand-rolled (D-12).
func (s *Server) List(_ context.Context, req *apiproto.ListRequest) (*apiproto.ListResponse, error) {
	if s.gated() {
		return nil, gateUnavailable()
	}
	records := s.engine.Records()
	keys := make([]string, 0, len(records))
	for key, rec := range records {
		if rec.Deleted || !strings.HasPrefix(key, req.GetPrefix()) {
			continue
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return &apiproto.ListResponse{Keys: keys}, nil
}

// SnapshotCreate forces the current dump to complete — every confirmed
// write is on disk — then byte-copies that finished dump under name
// through the persistfile adapter (API-07, D-10). A snapshot is a write
// (A5): a create forces a confirmed dump, so the handler is gated exactly
// like Set and Delete — Unavailable while the startup gate is closed. The
// name crosses to persistfile untouched: its validateName is the single
// closed path-traversal control (T-05-07) and runs before any filesystem
// touch, so the adapter's whole job is mapping the sentinel through
// mapError (InvalidArgument) — never re-validating, never sanitizing.
func (s *Server) SnapshotCreate(_ context.Context, req *apiproto.SnapshotCreateRequest) (*apiproto.SnapshotCreateResponse, error) {
	if s.gated() {
		return nil, gateUnavailable()
	}
	err := s.engine.Snapshot(req.GetName())
	if err != nil {
		return nil, mapError(err)
	}
	return &apiproto.SnapshotCreateResponse{}, nil
}

// SnapshotLoad replays the named snapshot as ordinary fresh-version writes
// and deletes (API-07, D-09 — persist.Restore semantics): the replay
// replicates and wins by normal LWW, and the 05-03 change tap fires watch
// events for every replayed record with no extra wiring — a load is
// visible to Watch subscribers like any write burst. Gated exactly like
// SnapshotCreate (A5: a load replays writes through the store). The name
// crosses to persistfile untouched, for the same T-05-07 reason as the
// create path, and the call returns only after one final dump covering the
// whole replay — the PERS-04 confirmation contract, batched.
func (s *Server) SnapshotLoad(_ context.Context, req *apiproto.SnapshotLoadRequest) (*apiproto.SnapshotLoadResponse, error) {
	if s.gated() {
		return nil, gateUnavailable()
	}
	err := s.engine.Restore(req.GetName())
	if err != nil {
		return nil, mapError(err)
	}
	return &apiproto.SnapshotLoadResponse{}, nil
}

// notifyLocalWrite hands the just-confirmed write to the replication
// adapter through the LocalNotify seam (D-05). The record is rebuilt from
// the write's own inputs — the engine-stamped version, the request value
// (nil for a tombstone) and the checksum the store computed for exactly
// those inputs — so the notified record is the stored one; the defensive
// copy keeps the record's bytes owned by the notification. track carries
// the request's wait_for_replication flag through to the adapter, which
// creates the waiter tracker synchronously before enqueueing when set
// (XPORT-12). The notify error is deliberately ignored: a failed notify
// (ErrRunLoopExited during shutdown) never lies about the durable write,
// which the engine already confirmed.
func (s *Server) notifyLocalWrite(key string, ver version.RecordVersion, data []byte, deleted, track bool) {
	if s.notify == nil {
		return
	}
	rec := storage.Record{
		Version:  ver,
		Data:     bytes.Clone(data),
		Deleted:  deleted,
		Checksum: storage.RecordChecksum(key, ver, data, deleted),
	}
	_ = s.notify(key, rec, track)
}

// keyTooLongError builds the InvalidArgument for a write key over the
// single key-length ceiling (XPORT-03). The message names the offending
// length, the limit and the unit: the limit counts bytes, so a multi-byte
// UTF-8 key can exceed it with fewer runes. Reads stay permissive (Get's
// document explains why), so this error exists only on Set and Delete.
func keyTooLongError(length int) error {
	return status.Errorf(codes.InvalidArgument,
		"key length %d exceeds the maximum of %d bytes (the limit is measured in bytes)",
		length, storage.MaxKeyBytes)
}

// mapError translates one engine error into its pinned RPC code, matching
// with [errors.Is], never identity comparison — the table is pre-pinned in
// the sentinel files' own docs (internal/app/storage/errors.go,
// internal/app/persist/errors.go):
//
//	storage.ErrNotFound      -> codes.NotFound        ("key not found" — missing OR tombstoned)
//	storage.ErrAlreadyExists -> codes.AlreadyExists   ("key already exists" — any record incl. tombstone)
//	storage.ErrValueTooLarge -> codes.InvalidArgument ("value exceeds the maximum allowed size")
//	persist.ErrDumpFailed    -> codes.Unavailable     ("dump failed; write not confirmed")
//	persist.ErrClosed        -> codes.Unavailable     ("persist engine closed")
//	persistfile.ErrInvalidSnapshotName -> codes.InvalidArgument ("invalid snapshot name")
//
// A status error passes through unchanged; any other unmapped error becomes
// codes.Internal. Handlers stay total over any input — no panic path, the
// applyRemote precedent.
func mapError(err error) error {
	switch {
	case errors.Is(err, storage.ErrNotFound):
		return status.Error(codes.NotFound, storage.ErrNotFound.Error())
	case errors.Is(err, storage.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, storage.ErrAlreadyExists.Error())
	case errors.Is(err, storage.ErrValueTooLarge):
		return status.Error(codes.InvalidArgument, storage.ErrValueTooLarge.Error())
	case errors.Is(err, persist.ErrDumpFailed):
		return status.Error(codes.Unavailable, persist.ErrDumpFailed.Error())
	case errors.Is(err, persist.ErrClosed):
		return status.Error(codes.Unavailable, persist.ErrClosed.Error())
	case errors.Is(err, persistfile.ErrInvalidSnapshotName):
		return status.Error(codes.InvalidArgument, persistfile.ErrInvalidSnapshotName.Error())
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	return status.Errorf(codes.Internal, "internal error: %v", err)
}

// recordToProto maps one stored record onto the wire Record (D-01): the
// full 16-byte version travels as its complete slice, the value bytes as
// stored — the subscriber or writer never needs a second Get to act on the
// record.
func recordToProto(key string, rec storage.Record) *apiproto.Record {
	return &apiproto.Record{
		Key:     key,
		Version: rec.Version[:],
		Value:   rec.Data,
		Deleted: rec.Deleted,
	}
}
