package app

import "errors"

// Sentinel errors returned by the store operations. Callers match them with
// [errors.Is], never identity comparison; Phase 5 maps them 1:1 to RPC codes.
var (
	// ErrNotFound is returned by Get and Delete when the key is absent or
	// tombstoned. Phase 5 maps it to the NotFound RPC code.
	ErrNotFound = errors.New("key not found")

	// ErrAlreadyExists is returned by SetIfAbsent when the key holds any
	// record, including a tombstone. Phase 5 maps it to AlreadyExists.
	ErrAlreadyExists = errors.New("key already exists")

	// ErrValueTooLarge is returned by Set, SetIfAbsent and Merge when the
	// value exceeds Config.ValueMaxBytes. Phase 5 maps it to InvalidArgument.
	ErrValueTooLarge = errors.New("value exceeds the maximum allowed size")
)
