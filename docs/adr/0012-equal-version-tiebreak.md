---
adr: 0012
review-decision: "P-07 (equal-version tie-break)"
informs: [PERSIST-07]
date: 2026-11-22
status: accepted
---

# 0012 — Equal versions resolve by the deterministic (version, content) law

## Context

The Spec's record version carries random filler beside the node ID (Spec L243:
`rand_b` holds the 16-bit `NodeID` in its high bits and random bits in the
rest), and it relies on that filler only for uniqueness, never for order. v1.0
replaced the filler with zeros (S-HLC-07, accepted deviation): uniqueness and
byte order survive, but the version becomes a pure function of the hybrid
logical clock reading and the writing node.

That makes an equal version with different content reachable: a node loses its
disk and its clock rolls back to a reading it already stamped, so a peer's
pre-loss record and the node's post-rollback write carry the same 16 bytes.
v1.0 handled that pathological state silently — the merge ignored the incoming
record and the replication core Acked every equal version before consulting the
store — so two replicas kept different values and both confirmed their own
write.

Deterministic simulator replay and byte-identical dumps are project invariants.
A random tail cannot settle the tie without breaking both, so P-07 asks for a
deliberate, recorded decision rather than the Spec's accidental one.

## Decision

The join order is `(version, content)`:

1. A strictly greater full 16-byte version wins (unchanged).
2. Equal versions with identical `Deleted` and identical `Data` are a full
   no-op: no conflict, no install, no dump.
3. Equal versions with different content resolve **remove-wins first**: a
   tombstone beats a live record whichever side carries it. When both are live
   or both are tombstones, the lexicographically greater `Data` wins by
   `bytes.Compare` (a shorter prefix sorts lower).

A tie-break win or loss reports exactly one `Conflict` whose `Winner` and
`Loser` are the same version bytes — the version cannot name the winner, so the
stored record is the only observable evidence.

The law applies at two sites that ship together. `storage.Store.join` decides
the merge, and `storage.Store.MergeChanged` reports whether the incoming
record became the stored record (`storage.RecordsEqual` is the one content
comparison both sites share). `repl.Core.applyRemote` stops short-circuiting
every equal version: identical content still Acks with no reply, while differing
content falls through to the store and the answer follows the stored record —
an incoming win Acks, a stored win is answered with exactly one unicast
`DataReplication` of the stored record and no Ack.

Explicit non-changes: the version layout and the zero filler stay as S-HLC-07
records them (no random tail is reintroduced), and per-prefix delete semantics
stay out of scope — PFXD-01 remains a v2 item.

## Consequences

- Two replicas holding the same version with different content converge on the
  same record, and the winner is a pure function of the two records: same-seed
  replay and byte-identical dumps are preserved.
- One of the two confirmed operations is lost. That is unavoidable — the
  version bytes cannot order the writes — and remove-wins deliberately loses
  the value rather than resurrecting a confirmed delete. This is the
  lost-disk-plus-clock-rollback state, not a normal conflict.
- The wire rule is deterministic and amplification-bounded: a stored win emits
  exactly one unicast reply and no Ack; an incoming win or identical content
  Acks and emits no reply.
- Conflict accounting counts the tie-break exactly once in both directions, so
  the merge counters keep reporting divergence truthfully.
- `S-ALG-REP-03` stays accurate only in its qualified form: an equal version is
  acknowledged without triggering a dump when it carries identical content.

## Links

- Source: docs/reviews/2026-11-19-v1.0/REVIEW.md P-07 (ENG-07); Spec L243
- Informs: PERSIST-07 (equal version with different content resolved
  deliberately — ADR recorded, implemented and tested)
- Related: S-HLC-07 (zero filler instead of the Spec's random tail);
  S-ALG-REP-03 (equal-version acknowledgement, now identical-content only)
