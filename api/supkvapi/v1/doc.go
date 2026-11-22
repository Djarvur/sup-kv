// Package supkvapi is the client-facing wire contract of a sup-kv node.
//
// The seven RPCs of SupKVService are Get, Set, Delete, List, Watch (the only
// server-streaming method), SnapshotCreate and SnapshotLoad. Every
// state-carrying message travels as a full Record — key, 16-byte RFC 9562
// RecordVersion, value bytes and the deleted tombstone flag — so a client never
// needs a second round trip to act on a change.
//
// The proto schema (supkvapi.proto) is the source of truth; the committed
// *.pb.go files beside it are produced by `make proto`, which runs the
// go.mod-pinned `go tool buf generate api --template buf.gen.api.yaml`. Edit
// the .proto and regenerate — never hand-edit generated code.
//
// This package is generated code plus this file and its tests. It must never
// import github.com/Djarvur/sup-kv/internal: that tree is not importable from
// other modules, and this package exists precisely to be importable from them.
// The boundary is machine-enforced by the sup-kv-public-api-no-internal
// depguard rule.
package supkvapi
