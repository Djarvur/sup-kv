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
	"io/fs"
	"log/slog"
	"sync/atomic"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	apiproto "github.com/Djarvur/sup-kv/api/supkvapi/v1"
	"github.com/Djarvur/sup-kv/internal/app/persist"
	"github.com/Djarvur/sup-kv/internal/app/storage"
	"github.com/Djarvur/sup-kv/internal/app/version"
	"github.com/Djarvur/sup-kv/internal/persistfile"
	"github.com/Djarvur/sup-kv/internal/port"
)

// listedKeyNotArrivedMessage is the rule 14.5(6) status message for a read
// of a key a peer's complete listing named whose value has not arrived
// locally: the key provably exists on a peer, so claiming NotFound would
// lie about the cluster's state.
const listedKeyNotArrivedMessage = "startup gate: key is listed by a peer but its value has not arrived"

// LocalNotify is the post-write replication notification seam: the gRPC
// write handler is "the author" of the phase-4 D-05 explicit-call contract.
// It receives the record the write just installed, whether the write
// requested a wait_for_replication report (track), and the peer ids the
// handler captured immediately before the engine write (peers) — the
// write-time snapshot the core seeds its wait set from, so the adapter can
// create the waiter tracker synchronously before enqueueing the notification
// (XPORT-12) and the core can pin the promise to exactly those ids (A-01).
// It reports nil when the notification was accepted for the replication loop,
// or a non-nil error when it was dropped (udprepl.ErrRunLoopExited once that
// loop has exited).
type LocalNotify func(key string, rec storage.Record, track bool, peers []uint16) error

// LocalReplayNotify is the PACED replication notification seam for records a
// snapshot load replayed (PROTO-08, T-4-20, WR-04). The record is already
// installed; the seam queues it without a tracker and without an immediate
// broadcast, so a large load drains through the record-send pacer instead of
// emitting one datagram per key back-to-back. Production injects
// udprepl.Adapter.NotifyReplayedWrite; a nil seam means "no paced path" and
// the handler falls back to LocalNotify with track=false.
type LocalReplayNotify func(key string, rec storage.Record) error

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
		if errors.Is(err, storage.ErrNotFound) {
			// Engine.Get hides tombstones, so ErrNotFound covers both "no
			// local record" and "local tombstone". Version is
			// tombstone-visible — the only presence probe on this path — so
			// a present record is the local tombstone (keep NotFound), while
			// an absent one may be a key a peer's complete listing named
			// whose value has not arrived: that is Unavailable, never
			// NotFound (rule 14.5(6)).
			if _, present := s.engine.Version(req.GetKey()); !present &&
				s.opts.ListedKey != nil && s.opts.ListedKey(req.GetKey()) {
				return nil, status.Error(codes.Unavailable, listedKeyNotArrivedMessage)
			}
		}
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
// single validation path. A create_if_absent for a key a peer's complete
// listing named is then refused with AlreadyExists before the engine call
// (D-01, rule 14.5): the peer already owns the key, so a local create would
// overwrite it (F14). The check covers live and tombstoned listed keys
// alike, because the listed set carries keys, not versions. wait_for_replication asks the
// server to wait after the confirmed write for the replication Acks and
// decorate the response with the D-08 aggregate report (wait.go): the wait
// is bounded by the positive per-request wait_timeout_ms, else the
// operator-tunable default (D-09), and never changes the write outcome —
// it runs strictly after the engine confirmed the write, so no path below
// can turn a confirmed write into an error. Without the flag the report
// fields keep their zero values. The replication notification is
// best-effort (D-05): a dropped notification never fails the RPC, because
// the durable write is already engine-confirmed. A write the engine
// installed but whose confirming dump failed is still notified before the
// Unavailable answer (PROTO-08): the record is in memory and must replicate,
// and only a closed engine's abandoned work is not notified.
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
	if req.GetCreateIfAbsent() && s.opts.ListedKey != nil && s.opts.ListedKey(req.GetKey()) {
		return nil, status.Error(codes.AlreadyExists, storage.ErrAlreadyExists.Error())
	}
	// The write-time peer snapshot (D-01, A-01): captured exactly once,
	// immediately before the engine write — the acceptance point — and the
	// same slice feeds the replication notification and the wait report, so
	// no peer discovered after this line can extend the awaited set or move
	// the reported total. The helper returns a clone: the seam's slice is
	// never retained.
	peers := s.writePeers()
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
		// PROTO-08 totality (D-08, T-4-18, T-5-25): a write the engine installed
		// but whose confirming dump failed — or did not complete within the
		// confirmation timeout (PERSIST-08) — is in memory and must reach peers,
		// so it is notified before the error answers Unavailable. track is
		// false: the RPC errors, so no wait_for_replication report exists.
		// ErrClosed is shutdown work the process is abandoning — never notified.
		if errors.Is(err, persist.ErrDumpFailed) || errors.Is(err, persist.ErrUnconfirmed) {
			s.notifyLocalWrite(req.GetKey(), ver, req.GetValue(), false, false, peers)
		}
		return nil, mapError(err)
	}
	s.notifyLocalWrite(req.GetKey(), ver, req.GetValue(), false, req.GetWaitForReplication(), peers)
	resp := &apiproto.SetResponse{Version: ver[:]}
	if req.GetWaitForReplication() {
		timeout := s.waitTimeout(req.GetWaitTimeoutMs())
		resp.Acked, resp.AckedCount, resp.TotalPeers = s.waitForReplication(ctx, req.GetKey(), ver, peers, timeout)
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
// without the flag the report fields keep their zero values. Like Set, an
// unconfirmed tombstone whose dump failed is notified before the
// Unavailable answer (PROTO-08 totality), while a closed engine's
// abandoned work is not.
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
	// Same write-time peer capture as Set (D-01: a delete is a write): one
	// read immediately before the engine call, feeding both the notification
	// and the wait report.
	peers := s.writePeers()
	ver, _, err := s.engine.Delete(req.GetKey())
	if err != nil {
		// PROTO-08 totality (D-08, T-4-18, T-5-25): the unconfirmed tombstone
		// is already installed in memory and must reach peers before the RPC
		// answers Unavailable — whichever unconfirmed class it is, a failed
		// dump or an expired confirmation timeout (PERSIST-08); track is false
		// and ErrClosed never notifies.
		if errors.Is(err, persist.ErrDumpFailed) || errors.Is(err, persist.ErrUnconfirmed) {
			s.notifyLocalWrite(req.GetKey(), ver, nil, true, false, peers)
		}
		return nil, mapError(err)
	}
	s.notifyLocalWrite(req.GetKey(), ver, nil, true, req.GetWaitForReplication(), peers)
	resp := &apiproto.DeleteResponse{Version: ver[:]}
	if req.GetWaitForReplication() {
		timeout := s.waitTimeout(req.GetWaitTimeoutMs())
		resp.Acked, resp.AckedCount, resp.TotalPeers = s.waitForReplication(ctx, req.GetKey(), ver, peers, timeout)
	}
	return resp, nil
}

// List returns the live keys under prefix in one of two modes (D-02,
// APIS-02). With page_size unset (0) it keeps the legacy contract: every
// matching live key in one response, byte-wise ascending, keys only — the
// frozen R-01 single-call shape the stand poll reads — and it copies no
// value bytes because it never calls Lookup. With page_size > 0 it returns
// a bounded keyset page of records: at most the resolved page size entries
// with values, keys aligned to records, a page_token that resumes strictly
// after its decoded position, and a non-empty next_page_token unless the
// listing is exhausted. The empty prefix matches every live key (D-15);
// tombstones are never listed — List reflects live state, not history
// (D-14); the legacy response carries keys only, values are fetched via
// Get (D-13). The startup gate is checked first like every other data RPC
// (API-04, D-01): a closed node answers Unavailable rather than an empty
// list, which would lie about a state it has not yet learned. The page
// walk and the constants live in list.go; the live key→version view
// (Engine.LiveVersions) is tombstone-free by the stored Deleted flag, so
// no whole-store record snapshot is ever taken on this path.
func (s *Server) List(_ context.Context, req *apiproto.ListRequest) (*apiproto.ListResponse, error) {
	if s.gated() {
		return nil, gateUnavailable()
	}
	return listPage(s.engine, req)
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
// whole replay — the PERS-04 confirmation contract, batched. The handler
// notifies every record the replay installed with track=false (PROTO-08):
// a snapshot load replicates its result even when the confirming dump
// failed, because the replayed state is in memory; a no-op load notifies
// nothing, and a closed engine's abandoned work is not notified. A snapshot
// that does not decode cleanly is refused before any change (persist.
// ErrSnapshotIncomplete, mapped to DataLoss): the engine returns no records,
// so nothing is notified and the live state and on-disk generations stay
// untouched (CR-01).
func (s *Server) SnapshotLoad(_ context.Context, req *apiproto.SnapshotLoadRequest) (*apiproto.SnapshotLoadResponse, error) {
	if s.gated() {
		return nil, gateUnavailable()
	}
	records, err := s.engine.Restore(req.GetName())
	// PROTO-08 totality (D-08, T-4-19): every record the replay installed
	// reaches peers, track=false because a load has no waiter report. The
	// notify runs even when the confirming dump failed — the state is in
	// memory — but never on a closed engine, which is abandoning its work:
	// records installed before the close are exactly the abandoned ones.
	if !errors.Is(err, persist.ErrClosed) {
		for _, rec := range records {
			s.notifyReplayedWrite(rec)
		}
	}
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
// (XPORT-12), and peers carries the write-time peer snapshot the handler
// captured immediately before the engine call (D-01) — the same slice the
// wait report is pinned to. The notify error is deliberately ignored: a
// failed notify (ErrRunLoopExited during shutdown) never lies about the
// durable write, which the engine already confirmed.
func (s *Server) notifyLocalWrite(key string, ver version.RecordVersion, data []byte, deleted, track bool, peers []uint16) {
	if s.notify == nil {
		return
	}
	rec := storage.Record{
		Version:  ver,
		Data:     bytes.Clone(data),
		Deleted:  deleted,
		Checksum: storage.RecordChecksum(key, ver, data, deleted),
	}
	_ = s.notify(key, rec, track, peers)
}

// notifyReplayedWrite hands one snapshot-replayed record to the paced replay
// seam when the wiring provides one (WR-04, T-4-20), else through the live
// notification seam with track=false — a load has no waiter report. The
// peers slice is nil: a replay has no waiter, so no write-time peer set
// exists to carry (D-01's promise covers live writes only). The rebuilt
// record is notifyLocalWrite's: the replay's stored version, its data and the
// store's checksum. A notify error is ignored for the same reason as on the
// live path: the record is already installed and the RPC's own error (if any)
// reports the dump honestly.
func (s *Server) notifyReplayedWrite(rec persist.RestoredRecord) {
	if s.opts.ReplayNotify == nil {
		s.notifyLocalWrite(rec.Key, rec.Version, rec.Data, rec.Deleted, false, nil)
		return
	}
	_ = s.opts.ReplayNotify(rec.Key, storage.Record{
		Version:  rec.Version,
		Data:     bytes.Clone(rec.Data),
		Deleted:  rec.Deleted,
		Checksum: storage.RecordChecksum(rec.Key, rec.Version, rec.Data, rec.Deleted),
	})
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

// Status message constants of the error mapper (D-03). Every client-visible
// string is a constant or a package sentinel: no error text — and therefore
// no configured path, errno or wire-dialect detail — can cross the
// untrusted client channel (T-06-11, T-06-12).
const (
	// internalErrorMessage is the totality fallback's exact text: every
	// unmapped error answers this constant and the detail goes to the log
	// instead, so an unforeseen failure can never leak through the response.
	internalErrorMessage = "internal error"

	// snapshotNotFoundMessage pins the missing-snapshot row. ReadSnapshot
	// surfaces the adapter's [*fs.PathError], whose text names the configured
	// snapshot path, so the mapper substitutes this constant.
	snapshotNotFoundMessage = "snapshot not found"

	// permissionDeniedMessage is the permission row, shared by the adapter's
	// [fs.ErrPermission] and the port's FailurePermission class: both reduce
	// to the same client-visible fact.
	permissionDeniedMessage = "permission denied"

	// diskFullMessage is the FailureDiskFull class's pinned text.
	diskFullMessage = "disk full"
)

// mapError translates one engine or adapter error into its pinned RPC code,
// matching with [errors.Is], never identity comparison — the table is
// pre-pinned in the sentinel files' own docs (internal/app/storage/errors.go,
// internal/app/persist/errors.go, internal/persistfile/snapshot.go) and in
// the port's FailureClass vocabulary (internal/port/dump.go). The order is
// the contract:
//
//	storage.ErrNotFound      -> codes.NotFound        ("key not found" — missing OR tombstoned)
//	storage.ErrAlreadyExists -> codes.AlreadyExists   ("key already exists" — any record incl. tombstone)
//	storage.ErrValueTooLarge -> codes.InvalidArgument ("value exceeds the maximum allowed size")
//	persist.ErrDumpFailed    -> codes.Unavailable     ("dump failed; write not confirmed")
//	port.FailureDiskFull     -> codes.ResourceExhausted ("disk full")
//	port.FailurePermission   -> codes.PermissionDenied  ("permission denied")
//	persist.ErrUnconfirmed   -> codes.Unavailable     ("write not confirmed before the confirmation timeout")
//	persist.ErrClosed        -> codes.Unavailable     ("persist engine closed")
//	persist.ErrSnapshotIncomplete -> codes.DataLoss    ("snapshot does not decode cleanly")
//	persistfile.ErrInvalidSnapshotName -> codes.InvalidArgument ("invalid snapshot name")
//	persistfile.ErrNoDump    -> codes.FailedPrecondition ("no dump on disk to snapshot")
//	fs.ErrNotExist           -> codes.NotFound        ("snapshot not found")
//	fs.ErrPermission         -> codes.PermissionDenied ("permission denied")
//	context.Canceled         -> codes.Canceled
//	context.DeadlineExceeded -> codes.DeadlineExceeded
//
// The class rows sit after ErrDumpFailed because the engine wraps a failed
// save as ErrDumpFailed — the write answers Unavailable, never a class —
// while CreateSnapshot surfaces the adapter's class directly (T-06-13).
// ErrNoDump precedes the fs rows so the adapter's own sentinel text
// survives. A pre-built status error passes through unchanged. Every other
// error is logged exactly once — a static message with the detail as an
// attribute — and answered with internalErrorMessage, which is what makes
// the mapping total and the response path-free: totality is structural, not
// an enumeration of everything that can go wrong (D-03). Handlers stay total
// over any input — no panic path, the applyRemote precedent.
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
	case port.FailureClassOf(err) == port.FailureDiskFull:
		return status.Error(codes.ResourceExhausted, diskFullMessage)
	case port.FailureClassOf(err) == port.FailurePermission:
		return status.Error(codes.PermissionDenied, permissionDeniedMessage)
	case errors.Is(err, persist.ErrUnconfirmed):
		return status.Error(codes.Unavailable, persist.ErrUnconfirmed.Error())
	case errors.Is(err, persist.ErrClosed):
		return status.Error(codes.Unavailable, persist.ErrClosed.Error())
	case errors.Is(err, persist.ErrSnapshotIncomplete):
		// DataLoss, not Unavailable: the snapshot artifact exists and was read,
		// but part of its content is unrecoverable — a full rollback from the
		// partial key set could delete live keys, so the load was refused
		// before any change (CR-01). The snapshot's name is a client-supplied
		// label, but the row answers the sentinel constant like every other
		// mapped error.
		return status.Error(codes.DataLoss, persist.ErrSnapshotIncomplete.Error())
	case errors.Is(err, persistfile.ErrInvalidSnapshotName):
		return status.Error(codes.InvalidArgument, persistfile.ErrInvalidSnapshotName.Error())
	case errors.Is(err, persistfile.ErrNoDump):
		return status.Error(codes.FailedPrecondition, persistfile.ErrNoDump.Error())
	case errors.Is(err, fs.ErrNotExist):
		return status.Error(codes.NotFound, snapshotNotFoundMessage)
	case errors.Is(err, fs.ErrPermission):
		return status.Error(codes.PermissionDenied, permissionDeniedMessage)
	case errors.Is(err, context.Canceled):
		// The matched sentinel, not err: a wrapped cancellation would make
		// status.FromContextError answer Unknown.
		return status.FromContextError(context.Canceled).Err()
	case errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(context.DeadlineExceeded).Err()
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	// The fallback logs the detail (the operator's only view of an
	// unclassified failure) and answers the constant — never err.Error()
	// (T-06-11): static message, snake_case attribute, no path on the wire.
	slog.Error("Unmapped RPC error", "detail", err.Error())
	return status.Error(codes.Internal, internalErrorMessage)
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
