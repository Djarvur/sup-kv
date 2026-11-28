# Phase 3: Transport - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-11-21
**Phase:** 3-Transport
**Mode:** --auto (all areas auto-selected; recommended option chosen for every question)

---

## Auto-selected areas and choices

| Area | Question | Selected |
|------|----------|----------|
| Sync pacing | How is the listing paced and made loss-robust? | Token-bucket at `sync_response_rate` + rotating start chunk (recommended default) |
| Socket buffers | Configure+report buffers or document limits only? | Configure + log + fail-fast (R-08 acceptance) |
| Unacked cost | How does event-loop cost stop growing with the unacked set? | Per-peer unacked ledger keyed (key,version) (Q-02 remedy) |
| Ack seam | How do waiters learn about acks? | Outbox result + waiter channel, removing 5 ms polling (XPORT-12) |
| Retransmit pacing | How are retransmits bounded? | Per-peer jittered pacing on the existing backoff ladder (R-02 acceptance) |
| Validation | Validate before clock or trust the sender? | Validate before clock, own discard classes, `max_clock_lead` bound (R-07/R-16) |
| Read loop | What may terminate the read loop? | Only `net.ErrClosed`; everything else counted/logged (R-14/R-15) |
| Seams | key→version view + dedupe queue + dead code? | Shallow view + dedupe queue + delete-with-justification (Q-01/R-11/Q-09) |

**Notes:** All acceptance numbers come verbatim from REQUIREMENTS.md XPORT-01…13; the Phase-2 red tests already assert them in full, so the phase's job is code fixes until the runner rows flip green.

## Claude's Discretion

- Pacer implementation details (token bucket vs timer window), jitter distribution, `max_clock_lead` default (60 s unless research grounds better), queue capacities, rotation persistence.

## Deferred Ideas

None — discussion stayed within phase scope.
