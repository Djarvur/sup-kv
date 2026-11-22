package persist

// flushRequest is one flush handed to Run through the confirmer's flushCh:
// reply receives the dump's resolution, and snapshot marks a
// snapshot-originated request (Snapshot, Restore). A snapshot request may
// defer to a pending retry rung instead of forcing an attempt, so a client
// snapshot loop cannot drive the dump rate (WR-02); an explicit
// Flush/FlushTimeout request always attempts one dump even while the ladder
// is armed, because a rung must never make shutdown skip the mandated final
// dump (N-1). The type is shared by the confirmer's channel, the scheduler and
// the engine's flush seam.
type flushRequest struct {
	reply    chan error
	snapshot bool
}
