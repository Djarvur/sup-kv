// Package storage is the sup-kv in-memory LWW-Element-Map store: a pure,
// I/O-free library over records versioned by the sibling version package
// and stamped by a clock supplied by the caller through the consumer-side
// Clock interface declared here — the concrete clock (internal/app/clock)
// satisfies it and tests may substitute fakes; declaring the interface on
// the consuming side keeps the store free of the concrete clock's import,
// while the port interfaces (dump, replication) live in internal/port. The
// lint gate forbids networking, filesystem, syscall and gRPC/protobuf
// imports here — see the sup-kv-app-no-io depguard rule in .golangci.yml.
//
// The store is a Last-Write-Wins-Element-Map CRDT: per key, the record with
// the greater full version.RecordVersion wins on merge, so merge is
// commutative, associative and idempotent — delivery order, duplicates and
// retransmissions cannot affect the outcome.
//
// Tombstone visibility is deliberately asymmetric. Get treats a missing key
// and a tombstoned key alike as ErrNotFound, while SetIfAbsent treats any
// existing record, including a tombstone, as ErrAlreadyExists, so a delete
// followed by a create-if-absent cannot resurrect the key. Tombstones stay
// in the map forever and merge like any other write.
//
// The package has no side effects: Merge reports every displaced or rejected
// version by returning it as a Conflict value, and every clock-advancing
// operation returns a drift signal as a signed [time.Duration] — zero within
// the injected clock's driftWarn tolerance, positive when the logical clock
// runs ahead of the wall clock beyond it, negative when the wall clock
// jumped forward past it. Callers decide where losers and drift warnings
// are recorded; nothing is logged or emitted inside this package. The
// errors ErrNotFound, ErrAlreadyExists and ErrValueTooLarge are sentinels
// matched with [errors.Is].
//
// Store is safe for concurrent use by multiple goroutines and the clock is
// independently safe; the store holds the write lock across each clock
// advance, so versions match commit order. Record Data is defensively copied
// on the way in and out — callers keep ownership of their slices.
// RecordChecksum is corruption detection for the Phase 3 dump, never tamper
// protection.
package storage
