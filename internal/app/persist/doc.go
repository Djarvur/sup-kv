// Package persist is the sup-kv persistence decision core: the pure JSONL
// dump codec plus the write-confirmation engine, consuming the sibling
// storage, clock and version packages and saving whole dump bytes through
// the port.DumpStore seam (D-01/D-02). The lint gate forbids filesystem,
// networking, syscall and gRPC/protobuf imports here — see the
// sup-kv-app-no-io depguard rule in .golangci.yml.
//
// The engine's dumper goroutine lifecycle is part of the API contract: the
// caller launches Run in its own goroutine, Run exits when the context is
// cancelled resolving every stranded waiter with ErrClosed, and the engine
// itself spawns no goroutine. A write blocks inside Set, SetIfAbsent or
// Delete until a dump containing it completed or failed — nil confirms the
// write is on disk (PERS-04), ErrDumpFailed reports the failed dump while
// the record stays readable in memory (D-05). Reads are never gated on
// dumps. No callbacks, no events — results only (Phase 2 D-01 tradition).
//
// The package is split into five internal roles (D-10, PERSIST-11): the
// loader/composition root (Open, mergeRecords and the load report), the
// store facade (the thin read and write forwards plus ApplyRemote), the
// dump scheduler (Run, the dump cycle, the retry ladder and the dump
// counters), confirmation accounting (the waiters, the dirty epoch, the
// flush request channel and the single-claim resolution) and snapshot
// orchestration (Snapshot, Restore and the replay helpers). Engine is the
// composition of those values; the split adds no exported name and no
// interface.
//
// Timing guarantees (D-04, PERS-03): under any write cadence a write is
// confirmed within MaxWait plus one dump duration — the dump starts no
// later than the oldest unconfirmed write plus MaxWait, because the
// deadline timer is armed from the clean-to-dirty transition and only the
// debounce timer resets on later changes, so bursts can coalesce but never
// starve confirmation. After a failed dump the next attempt starts one
// debounce interval later — the re-armed timer, never a tight retry loop —
// and it also carries the failed era's surviving records.
//
// Snapshots (PERS-05, D-08..D-10): Snapshot first forces the current dump
// to complete — a clean engine skips the redundant dump, a dirty one is
// dumped before anything is copied — then byte-copies the finished main
// dump under the adapter-validated name, so the snapshot is byte-identical
// to the confirmed on-disk state at creation; a failed flush aborts with
// nothing copied. Restore decodes the snapshot with the same codec and
// replays ONLY the snapshot's keys through the store's ordinary write path
// with fresh versions: a snapshot-live value beats a current tombstone, a
// snapshot tombstone beats a current live value, and keys absent from the
// snapshot keep their state untouched; the snapshot's own versions are
// never re-installed, so a restored state replicates and wins by normal LWW
// like any write. Replaying the same snapshot twice converges to the
// snapshot's values — the second restore's fresh versions beat the first's.
// Tombstone no-ops (nothing to beat, no version burned) and values a
// config-shrunk limit rejects are skipped, never errors. Both operations
// require a running Run goroutine like every blocking engine call, and
// both fail with the underlying error rather than degrading silently.
package persist
