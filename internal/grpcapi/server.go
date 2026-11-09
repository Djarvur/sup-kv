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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/grpcapi/apiproto"
	"github.com/Djarvur/sup-kv/internal/persistfile"
)

// LocalNotify is the post-write replication notification seam: the gRPC
// write handler is "the author" of the phase-4 D-05 explicit-call contract.
// It receives the record the write just installed and reports nil when the
// notification was accepted for the replication loop, or a non-nil error
// when it was dropped (udprepl.ErrRunLoopExited once that loop has exited).
type LocalNotify func(key string, rec storage.Record) error

// Server implements the apiproto.SupKVServer surface over one persist
// Engine. The UnimplementedSupKVServer embed is mandatory with
// protoc-gen-go-grpc v1.6.2 (forward-compat) and serves the phase's later
// plans: RPCs not yet implemented — SnapshotCreate, SnapshotLoad — inherit
// its Unimplemented responses until their plans land, so the whole phase's
// wire surface could be generated in one codegen pass.
type Server struct {
	apiproto.UnimplementedSupKVServer

	engine *persist.Engine
	notify LocalNotify
	hub    *WatchHub
}

// NewServer returns the SupKV service over engine. notify may be nil: that
// is the single-node configuration with no replication (D-11), and the
// handlers then skip the post-write notification entirely. hub is the
// WatchHub backing the Watch stream; NewServer attaches the engine to it —
// the hub is constructed first because persist.Open threads the change tap
// through storage.Config — and a nil hub keeps Watch answering
// Unimplemented (the pre-Watch embed behavior).
func NewServer(engine *persist.Engine, notify LocalNotify, hub *WatchHub) *Server {
	if hub != nil {
		hub.attach(engine)
	}
	return &Server{engine: engine, notify: notify, hub: hub}
}

// Get returns the live record for one key. Reads are never gated on dumps
// or replication state — an absent or tombstoned key is the NotFound code,
// never an empty success. An empty key is rejected with InvalidArgument
// before any engine call (A7), like the write handlers.
func (s *Server) Get(_ context.Context, req *apiproto.GetRequest) (*apiproto.GetResponse, error) {
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
// containing the record completed. An empty key is rejected with
// InvalidArgument before any engine call (A7: the store itself permits
// empty keys, the API does not). wait_for_replication is accepted but not
// acted on until plan 05-05 — the response's D-08 report fields carry zero
// values. The replication notification is best-effort (D-05): a dropped
// notification never fails the RPC, because the durable write is already
// engine-confirmed.
func (s *Server) Set(_ context.Context, req *apiproto.SetRequest) (*apiproto.SetResponse, error) {
	if req.GetKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "key must not be empty")
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
	s.notifyLocalWrite(req.GetKey(), ver, req.GetValue(), false)
	return &apiproto.SetResponse{Version: ver[:]}, nil
}

// Delete writes a fresh-version tombstone for one key and returns only
// after the engine confirmed the tombstone on disk (PERS-04 inside the
// RPC). A missing or already-tombstoned key surfaces the engine's NotFound
// through the shared mapper — a delete of nothing happened. An empty key
// is rejected with InvalidArgument before any engine call, like Set. The
// wait fields carry the same accepted-but-deferred meaning as on Set
// (API-05: a delete is a write), and the response's D-08 report fields
// carry zero values until plan 05-05.
func (s *Server) Delete(_ context.Context, req *apiproto.DeleteRequest) (*apiproto.DeleteResponse, error) {
	if req.GetKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "key must not be empty")
	}
	ver, _, err := s.engine.Delete(req.GetKey())
	if err != nil {
		return nil, mapError(err)
	}
	s.notifyLocalWrite(req.GetKey(), ver, nil, true)
	return &apiproto.DeleteResponse{Version: ver[:]}, nil
}

// List returns the sorted live (non-tombstoned) keys under prefix in one
// response (D-12): the empty prefix matches every key (D-15), tombstones
// are never listed — List reflects live state, not history (D-14) — and
// the response carries keys only; values are fetched via Get (D-13). The
// engine's Records() snapshot is tombstone-inclusive by contract, so the
// Deleted filter here is adapter-side. The scale envelope — tens of
// thousands of keys — fits the gRPC 4 MiB default send limit (research
// A6): grpc defaults are kept, and grpc.MaxSendMsgSize is the documented
// growth knob if key names ever grow the envelope — pagination is
// deliberately not hand-rolled (D-12).
func (s *Server) List(_ context.Context, req *apiproto.ListRequest) (*apiproto.ListResponse, error) {
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

// notifyLocalWrite hands the just-confirmed write to the replication
// adapter through the LocalNotify seam (D-05). The record is rebuilt from
// the write's own inputs — the engine-stamped version, the request value
// (nil for a tombstone) and the checksum the store computed for exactly
// those inputs — so the notified record is the stored one; the defensive
// copy keeps the record's bytes owned by the notification. The notify
// error is deliberately ignored: a failed notify (ErrRunLoopExited during
// shutdown) never lies about the durable write, which the engine already
// confirmed.
func (s *Server) notifyLocalWrite(key string, ver version.RecordVersion, data []byte, deleted bool) {
	if s.notify == nil {
		return
	}
	rec := storage.Record{
		Version:  ver,
		Data:     bytes.Clone(data),
		Deleted:  deleted,
		Checksum: storage.RecordChecksum(key, ver, data, deleted),
	}
	_ = s.notify(key, rec)
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
