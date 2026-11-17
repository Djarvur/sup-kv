// Package port holds the sup-kv port layer: the interface definitions that
// decouple the pure decision core in internal/app from the outside world.
//
// DumpStore is the Phase 3 persistence seam — whole dump bytes exchanged
// between the persistence engine (internal/app/persist) and the file adapter
// (internal/persistfile). Repl's message types (repl.go) are the Phase 4
// replication seam — unlike the whole-bytes dump seam, they carry decoded
// message structs: the UDP adapter (internal/udprepl) decodes the protobuf
// datagram and hands the core (internal/app/repl) plain message types,
// because protobuf cannot cross into internal/app (D-01). The lint gate
// restricts this package to standard-library imports (plus the own module
// path) — see the sup-kv-port-stdlib-only depguard rule in .golangci.yml.
package port
