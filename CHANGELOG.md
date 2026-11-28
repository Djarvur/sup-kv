# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/2.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.0] - 2026-11-28

The spec-compliance and resilience milestone: every documented mismatch from the
2026-11-19 v1.0 review is closed by a code fix or by an honest reference, the
spec-deviation registry ends the milestone with zero unclosed rows, and the
documentation set is machine-anchored — a documented claim that stops matching the
binary fails the build.

### Added

- Closed-interface startup semantics: reads and writes stay unavailable until the
  neighbour's complete key listing has been processed — every chunk of one listing
  round, never a first chunk alone — or the applicable startup gate elapses, and an
  empty-disk node uses the separate `startup_gate_empty` gate so a fresh disk does
  not wait out the data-carrying default (`TestRed_R03_gateClosedUntilFullList` in
  `internal/grpcapi/red_gate_test.go` and `TestSync_partialListingKeepsGateClosed`
  in `internal/app/repl/sync_test.go`)
- The operator documentation set: the failure-behavior table and seven runbooks in
  `docs/OPERATIONS.md`, the configuration reference `docs/CONFIGURATION.md`, the
  deployment guide `docs/DEPLOYMENT.md`, the dump-format reference `docs/FORMATS.md`
  and the performance ledger `docs/PERFORMANCE.md` — every claim in them anchored to
  something that exists in the binary and machine-checked by the doc-anchor tests
- The systemd unit `deploy/sup-kv.service` for board-style deployment, shipped in
  the release archives beside the deployment guide and exercised by a CI smoke run
- Paginated `List` and an honest replication wait: a positive `page_size` returns a
  bounded page of records with an opaque `next_page_token` that resumes after the
  last examined key, and the optional `wait_for_replication` wait reports the
  write-time peer set's acknowledgements — a displaced pending write resolves with
  the honest false and a stranger's `Ack` is inert — without ever changing the
  write outcome
- The operator-facing performance gate `make perf` with the threshold constants in
  `internal/testutil/perf.go` and the measured ledger in `docs/PERFORMANCE.md`
  (`TestPerfBurstDistinctKeys` in `internal/stand/perf_test.go` holds the standing
  1 000 writes/s burst bound)

### Changed

- Replication sending is paced end to end: sync answers, listing rounds and
  retransmissions leave at a bounded rate instead of bursting, and the egress queue
  is identity-keyed and deduplicated so a stalled round cannot grow it without
  bound
- Socket send and receive buffers are configured at startup and the kernel's
  effective values are logged, so the documented throughput bound rests on real
  buffers, not hoped-for defaults
- The value limit has one enforced source (`storage.ValueMaxBytes`): the same limit
  bounds API writes, the dump decoder and the launch validation, and a configured
  limit that cannot fit a UDP datagram exits with code 2 at startup
- Exit codes are distinct and documented: 0 for a clean stop, 1 for a runtime
  failure, 2 for a startup validation error, 3 for a failed final dump
- Lifecycle logging is structured with distinct levels: every load outcome, gate
  opening, peer change, drift signal and dump failure logs through the catalog in
  `docs/OPERATIONS.md`
- darwin is demoted to a development-only platform (ADR 0010): release archives
  are Linux-only (x86_64 and arm64), while building and unit tests on macOS remain
  supported and CI keeps both darwin legs
- The gRPC schema is published as the public client package `api/supkvapi/v1`
  (importable from another module) with a standalone Go client example under
  `examples/client` that builds in CI

### Fixed

- The four Critical findings of the 2026-11-19 review: an empty node's recovery is
  paced and holds its documented bound at the documented scale
  (`TestRed_R01_pacedSync20k` in `internal/stand/stand_red_test.go`); a 1 000
  writes/s burst keeps the dump cadence and drains the unacked set inside its bound
  (`TestRed_R02_burst` and `TestRed_R02_unackedDrain` in
  `internal/stand/stand_red_test.go`, `TestPerfBurstDistinctKeys` in
  `internal/stand/perf_test.go`, `TestEngine_burstBoundedConfirmation` in
  `internal/app/persist/engine_test.go`); the interface stays closed until the
  complete listing has arrived, including the replaced-node case
  (`TestRed_R03_gateClosedUntilFullList` in `internal/grpcapi/red_gate_test.go` and
  `TestRed_R03_f14ReplacedNode` in `internal/stand/stand_red_test.go`); and an
  oversized value can no longer be written, dump-confirmed and then discarded at
  the next start — the limit is enforced at the store, the API and the decoder
  (`TestRed_P01_valueLimit` in `internal/stand/stand_red_test.go` and
  `TestRed_P01_valueLimitEngine` in `internal/persistfile/red_value_limit_test.go`)
- Metric honesty: the conflict counter measures a displaced pending local write
  (not every rejected version), the drift counter measures a received version's
  lead over local time, and the resident-memory gauge reports the current process
  RSS on Linux
- Dump-load outcomes: a fallback to the previous generation logs exactly one WARN
  naming the failed candidate, an unusable pair of generations logs exactly one
  ERROR and starts empty, and rejected generations are preserved on disk as
  `.corrupt-*` evidence instead of being deleted

## [1.0.0] - 2026-11-17

First stable release: a leaderless replicated in-memory key-value store for
small trusted-LAN clusters — every node stays writable at all times with no
coordination, and a write confirmed to a client is never lost.

### Added

- LWW-Element-Map engine with HLC/UUIDv7 record versions, proven by property
  tests (`TestMerge_laws`, `TestMerge_lawsTombstones` and `TestProp_mergeSmallSpace`
  in `internal/app/storage/merge_test.go` and
  `internal/app/storage/prop_small_space_test.go`; the byte-wise order by
  `TestVersion_byteOrder` in `internal/app/version/version_test.go`): merge is
  commutative, associative and idempotent (tombstones included), and byte-wise
  version comparison equals `(l, counter, NodeID)` order with no clock
  synchronization
- Atomic human-readable JSON dumps with a `.prev` fallback chain, the
  write-confirmed-after-dump guarantee (an acknowledged write is always on
  disk) and named snapshots that restore as ordinary replicating writes
- UDP broadcast replication — Hello discovery, DataReplication/Ack with
  exponential-backoff retransmission, chunked DataSync anti-entropy — with
  convergence bounds proven by a deterministic seeded simulator under a
  lossy/duplicating/reordering/partitioning virtual network
  (`TestBound_normalOneRTT`, `TestBound_lossWithinAckTimeoutMax` and
  `TestBound_partitionHeal` in `internal/app/repl/sim_bound_test.go`)
- gRPC API: Get, Set (including create-if-absent), Delete, List and Watch
  with an exact snapshot-then-live cut, startup gate, standard health
  checking, `wait_for_replication` and snapshot create/load
- Single static binary: flags with `SUPKV_` env twins, `--version` identity
  and ordered graceful shutdown
- Observability: opt-in metrics/debug endpoint (in-repo Prometheus text
  exporter, expvar, pprof, optional mTLS), structured slog JSON logging and
  per-class dropped-datagram counters
- Failure-scenario suite over the deterministic simulator, real sockets and
  real two-process runs, plus benchmarks for dump throughput, replication
  bursts and confirmation latency under load

[Unreleased]: https://github.com/Djarvur/sup-kv/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/Djarvur/sup-kv/releases/tag/v1.1.0
[1.0.0]: https://github.com/Djarvur/sup-kv/releases/tag/v1.0.0
