// Package apiproto is the legacy home of the sup-kv client API, kept only so
// importers that predate the public move keep compiling.
//
// It re-exports the generated identifiers as thin aliases; new code must use
// the public package github.com/Djarvur/sup-kv/api/supkvapi/v1 (package
// supkvapi) instead. This package deliberately does not carry the Go
// "Deprecated:" marker: the frozen red tests must keep importing it
// byte-unchanged, and the marker would make staticcheck fail their build.
//
//nolint:revive // the package keeps its historical name at the legacy import path.
package apiproto
