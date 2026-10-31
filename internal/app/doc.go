// Package app will hold the sup-kv application core: the pure decision
// logic — the LWW-Element-Map CRDT store, hybrid logical clock and record
// versions — with no I/O of its own.
//
// This is the Phase 1 skeleton placeholder. The package carries no code yet;
// the engine lands in Phase 2. The lint gate forbids networking, filesystem,
// syscall and gRPC/protobuf imports here — see the sup-kv-app-no-io depguard
// rule in .golangci.yml.
package app
