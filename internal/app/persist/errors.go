package persist

import "errors"

// Sentinel errors returned by the write-confirmation engine. Callers match
// them with [errors.Is], never identity comparison; Phase 5 maps them to RPC
// codes.
var (
	// ErrDumpFailed is handed to every waiter of a failed dump (D-05): the
	// write stays in memory and replicates later. Phase 5 maps it to the
	// Unavailable RPC code.
	ErrDumpFailed = errors.New("dump failed; write not confirmed")

	// ErrClosed is handed to every waiter stranded by Run's exit: waiters
	// resolve exactly once, so no writer can block past the engine's
	// shutdown.
	ErrClosed = errors.New("persist engine closed")
)
