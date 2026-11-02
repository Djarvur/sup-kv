// Package port holds the sup-kv port layer: the interface definitions that
// decouple the pure decision core in internal/app from the outside world.
//
// DumpStore is the Phase 3 persistence seam — whole dump bytes exchanged
// between the persistence engine (internal/app/persist) and the file adapter
// (internal/persistfile). The lint gate restricts this package to
// standard-library imports (plus the own module path) — see the
// sup-kv-port-stdlib-only depguard rule in .golangci.yml.
package port
