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
package persist
