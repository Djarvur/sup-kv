# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/2.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
  synchronization (Phase 2)
- Atomic human-readable JSON dumps with a `.prev` fallback chain, the
  write-confirmed-after-dump guarantee (an acknowledged write is always on
  disk) and named snapshots that restore as ordinary replicating writes
  (Phase 3)
- UDP broadcast replication — Hello discovery, DataReplication/Ack with
  exponential-backoff retransmission, chunked DataSync anti-entropy — with
  convergence bounds proven by a deterministic seeded simulator under a
  lossy/duplicating/reordering/partitioning virtual network
  (`TestBound_normalOneRTT`, `TestBound_lossWithinAckTimeoutMax` and
  `TestBound_partitionHeal` in `internal/app/repl/sim_bound_test.go`) (Phase 4)
- gRPC API: Get, Set (including create-if-absent), Delete, List and Watch
  with an exact snapshot-then-live cut, startup gate, standard health
  checking, `wait_for_replication` and snapshot create/load (Phase 5)
- Single static binary: flags with `SUPKV_` env twins, `--version` identity
  and ordered graceful shutdown (Phase 5)
- Observability: opt-in metrics/debug endpoint (in-repo Prometheus text
  exporter, expvar, pprof, optional mTLS), structured slog JSON logging and
  per-class dropped-datagram counters (Phase 6)
- Failure-scenario suite over the deterministic simulator, real sockets and
  real two-process runs, plus benchmarks for dump throughput, replication
  bursts and confirmation latency under load (Phase 6)

[Unreleased]: https://github.com/Djarvur/sup-kv/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/Djarvur/sup-kv/releases/tag/v1.0.0
