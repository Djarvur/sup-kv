# sup-kv multi-node behavior

How a sup-kv cluster behaves with three or more nodes on one UDP broadcast
segment: the traffic shape, the timing bounds, what
`wait_for_replication` means when more than one peer exists, the protocol
bounds actually shipped, the topology the tests exercise, and the windows
that remain. [ADR 0009](adr/0009-multi-node-support.md) decides that
three-or-more-node support is completed; this document records what the code
and tests establish, with every residual window labelled. For the per-node
operational contract see [OPERATIONS.md](OPERATIONS.md); for trying the API
see [EXAMPLES.md](EXAMPLES.md).

## Traffic shape

Every node broadcasts one full listing per `sync_interval` — and one at
startup, right after its first Hello (`internal/app/repl/core.go`,
`Core.syncListing`) — cut into chunks by the `sync_chunk_bytes` budget
(default 1400; an entry that alone reaches the budget ships as its own
chunk) and drained one datagram per chunk at the shared listing pacer's
`sync_response_rate` cadence (`Core.drainListings`).
The listing is a broadcast, so one emission reaches every peer; the sender's
listing work is one shared chunk build, not one store copy per peer
(`Core.listingBuild` reuses a build while it is younger than `sync_interval`
and the store generation is unchanged — one `Versions()` copy per build age,
not per trigger or per peer).

Discovery adds targeted traffic on top. The first `Hello` from an unknown
`node_id` — a join, or a peer returning after `peer_timeout` — receives
exactly one unicast targeted listing; a restart (a new `run_id`) receives
one more (`Core.handleHello`; pinned by
`internal/app/repl/restart_test.go#TestRestart_firstHelloExactlyOneListing`,
`#TestRestart_runIDChangeTriggersFullSync` and
`#TestRestart_returningAfterEvictionGetsListing`). A node joining an N-node
segment therefore receives up to N−1 targeted listings from its peers plus
the N−1 periodic listings, while each sender still emits only its own chunk
stream.

Reactive answers — the answer to a `DataRequest`, a stale-record answer, a
`DataSync` reply — are unicast and share the single egress queue paced at
`sync_response_rate` (`Core.enqueueResponse`, `Core.drainEgress`). The
receiver's completeness state is per `(node_id, run_id, sync_id)`
(`Core.accumulateSync`), bounded as the next section records.

## Timing

These are the bounds the tests establish; none is a new rule.

- **Normal write visibility:** one write broadcast, one round trip. Every
  peer applies the `DataReplication` locally; the N=3 real-socket test polls
  every origin's write into both other stores byte-equal
  (`internal/udprepl/three_node_test.go#TestSmoke_threeNodeBroadcastConvergence`).
- **Partition heal without eviction:** within the existing
  `2 × sync_interval + response` bound (default `sync_interval` 30 s). Both
  sides resume periodic listings and answer the missed keys; the N=3
  simulator applies a real block and asserts convergence inside
  `boundPartitionHeal`
  (`internal/app/repl/sim_scenario_test.go#TestScenario_threeNode`;
  `boundPartitionHeal` in `internal/app/repl/sim_bound_test.go`).
- **Heal after eviction:** at the next `Hello` plus `N /
  sync_response_rate`, with no `sync_interval` term — the returning node is
  unknown again, so the first-seen path sends one targeted listing
  immediately (`#TestRestart_returningAfterEvictionGetsListing`; D-04's
  N / `sync_response_rate` bound). The default hello cadence is 1 s, so a
  restarted node rides this path instead of waiting for the periodic round.
- **A joining node:** converges at its first `Hello` listing. The joiner's
  `Hello` triggers every peer's first-seen listing, and the empty-disk
  scenario's bound carries no `sync_interval` term
  (`#TestScenario_emptyDisk`; `#TestRestart_firstHelloExactlyOneListing`).

## wait_for_replication at N > 2

`wait_for_replication` is the optional write-time wait for peer
acknowledgements (`internal/grpcapi/wait.go`; the
`ServerOptions.WaitReplicated`/`KnownPeers` seams in
`internal/grpcapi/gate.go`). At N > 2 its report is:

- `total` is the peer count captured exactly once at write time
  (`KnownPeers`, before any waiting), not a live count: a peer discovered
  during the wait never extends the awaited set.
- `acked=true` requires every one of those write-time peers to Ack the
  version. One unreachable peer therefore yields `acked=false` at the
  timeout even when the other N−1 peers Acked.
- An Ack from a peer unknown at write time still counts in the core's
  acknowledged set, but the reported count is clamped to `total`
  (`wait.go`, the WR-01 clamp), so the report can never claim more coverage
  than exists.

Known limitation IN-05: a peer discovered after the write-time snapshot but
before the core processes the notification joins the core wait set without
joining `total`, so a write fully covered by the write-time peers can still
report `false` at the timeout. This is the safe direction — the report
understates, never overstates, coverage. Phase 6 (APIS-01) owns any
tightening of this rule and the final `docs/API.md` wording.

## Protocol bounds

The bounds are code constants in `internal/app/repl/core.go`. Each refused
or evicted item is counted by an exported accessor; nothing is dropped
silently.

| Bound | Value | What it bounds | Why this value |
|---|---|---|---|
| `maxSyncStreams` | 16 | concurrent partial `(node_id, run_id, sync_id)` rounds | a healthy segment has one in-flight round per peer plus transient stragglers; overflow evicts the least-progressed stream deterministically |
| `maxSyncEntriesPerStream` | 65 536 | one round's accumulated key→version entries | a single hostile round is dropped whole before it can fill memory |
| `maxSyncEntriesTotal` | 131 072 | all partial rounds' retained entries | sized for the documented 20 000-key scale with headroom |
| `maxSyncChunks` | 65 536 | an accepted `chunk_count` | a worst-case 20 000-key round at one entry per chunk fits |
| `maxListedKeysTotal` | 131 072 | the retained per-peer listed-key union | same 20 000-key scale with headroom; overflow evicts whole peer sets in LRU order so the union, refcounts and retained total shrink together |
| `maxListingDestinations` | 32 | in-flight listing FIFOs | one per known peer plus the broadcast round; room for a 32-node segment |
| `maxListingChunks` | 65 536 | queued listing chunk work across destinations | one worst-case 20 000-key round fits whole; a trigger crossing the bound is refused whole |
| `seenRunsCap` | 8 | per-peer history of previous `run_id`s | bounds hostile identity flapping; the oldest entry is forgotten when the ring wraps |

The counters are `Core.SyncStreamDropped` (streams the entry bounds
dropped), `Core.ListedKeysEvicted` (keys released from the retained union,
including whole-peer-set LRU eviction), `Core.ListingDropped` (refused
listing triggers — each destination- or chunk-bound refusal is one count)
and `Core.NotificationsRegressed` (local-write notifications ignored as not
greater than the tracked version). The receive path also counts every
discarded datagram class, including `bad_sync_header` for a listing part
whose header is outside its domain.

## Three-node topology in tests

The real-socket N=3 proof is
`internal/udprepl/three_node_test.go#TestSmoke_threeNodeBroadcastConvergence`:
three production stores, cores and adapters — the real codec, read loop and
pacing, never a fake socket — constructed in-process and sharing ONE
wildcard loopback port through `udprepl.WithAddressReuse()`.

That topology is deliberately broadcast-only. Unicast to a port shared by
`SO_REUSEADDR` sockets is delivered to exactly one socket, the kernel's
choice (kernel-arbitrary), so the test asserts broadcast-delivered
convergence: every origin's write, and same-key LWW, in both other stores.
Unicast at N=3 is covered by the deterministic simulator and by the
asymmetric-port two-node real-adapter harness, where each node binds a
different port and broadcasts to the other's — every broadcast and every
unicast crosses to exactly the peer
(`internal/udprepl/scenario_test.go`). The production binary intentionally
has no address-reuse flag; `WithAddressReuse` is a test-only option.

## Residual windows (accepted)

These windows are accepted, not fixed in this phase. Each is labelled with
its owner.

- **Mixed-cluster gate window (accepted, D-01 literal reading).** A
  complete listing from ANY peer opens the gate. In a three-node cluster
  where one peer is empty and another holds data, a joiner can open on the
  empty peer's complete one-chunk listing before the data peer's listing
  arrives, then write with a clock that has not absorbed the data peer's
  versions. The first-Hello listings from every peer shrink the window to
  the hello cadence; the two-empty-nodes acceptance is pinned by
  `internal/stand/stand_test.go#TestStand_startupGateEmptyTwoEmptyNodes`.
  ADR 0005 and the two-empty-nodes decision (research Open Question 8) keep
  the literal reading.
- **Listed-but-unarrived Delete (accepted, 14.5(6) scope).** A read of a
  key named by a peer's complete listing whose value has not arrived answers
  `Unavailable` (`internal/grpcapi/server.go`, `(*Server).Get`;
  `internal/grpcapi/server_test.go#TestServer_getListedNotArrived`), but a
  `Delete` of that key keeps the local `NotFound`. ADR 0005 rule (6)
  specifies reads only (research A7); Phase 6 owns any API-wording change.
- **Unbounded peer table (residual trusted-LAN risk, Phase 7).** The peer
  table itself is unbounded in peer rows — a pre-existing risk on a trusted
  segment, recorded as research Open Question 5 and owned by Phase 7
  hardening. Each row is O(1), and the retained listed-key payload is capped
  at `maxListedKeysTotal` with whole-peer-set eviction
  (`Core.retainListedKeys`), so spoofed peers cannot pin peers × keys.
- **Listing drift is not in MergeStats (residual, Phase 7 OBSV-05).** A
  listing part's maximum version reaches the clock (`StateStore.Observe`),
  but listing drift does not increment `MergeStats`
  (`internal/app/repl/merge_stats_test.go#TestMergeStats_listingAndStaleExcluded`).
  Phase 7 owns drift semantics and the catalog that will expose it.

## Handoffs

- **Phase 6 (APIS-01/DOCS-05)** owns the final `docs/API.md` wording for
  `wait_for_replication`, including the IN-05 limitation described above.
- **Phase 9** owns the node-replacement runbook text, including what happens
  when the neighbour never appears: an empty-disk node waits
  `--startup-gate-empty` (default 60 s) before opening.
- **Phase 7 (OBSV)** owns the drift semantics (the listing-drift decision
  above) and the one-point configuration-relation validation for
  `startup_gate_empty` against `startup_gate`.
