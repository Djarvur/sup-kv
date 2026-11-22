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

	// ErrUnconfirmed is returned to a write whose covering dump did not
	// complete within ConfirmTimeout (PERSIST-08): the record is installed in
	// memory — and will be covered by a later successful dump — but it is not
	// yet durable. It is NOT a dump failure: the dump's outcome is unknown.
	// It must replicate exactly like a dump-failed write (PROTO-08 totality),
	// and Phase 5 maps it to the Unavailable RPC code.
	ErrUnconfirmed = errors.New("write not confirmed before the confirmation timeout")

	// ErrClosed is handed to every waiter stranded by Run's exit: waiters
	// resolve exactly once, so no writer can block past the engine's
	// shutdown.
	ErrClosed = errors.New("persist engine closed")
)
