# Pitfalls Research

**Domain:** Leaderless replicated in-memory KV store (LWW-Element-Map CRDT, HLC/UUIDv7 versions, UDP broadcast replication with ack/retransmit/anti-entropy, JSON dump persistence, gRPC API) in Go for small trusted-LAN clusters
**Researched:** 2026-10-31
**Confidence:** MEDIUM (web-sourced findings cross-checked against multiple independent sources; items marked "design-derived" come from analysis of `docs/IDEA.md` rather than external sources — confidence LOW until validated by phase tests)

Phase references use the suggested milestone shape from `docs/IDEA.md`: 0 CI bootstrap, 1 Engine, 2 Persistence, 3 Replication, 4 API, 5 Hardening, 6 v1.0.

---

## Critical Pitfalls

### Pitfall 1: Debounce starvation — "write confirmed after dump" hangs under sustained writes

**What goes wrong:**
The design confirms a client write with `OK` only after a dump containing it completed, with dumps debounced (`dump_debounce` 50 ms). If the debounce is implemented as reset-on-every-write (timer pushed forward each time a write arrives), a sustained write stream (the design targets bursts of ~1000 writes/s) re-arms the timer forever, the dump never fires, and every pending write blocks until its RPC deadline — the store reports `Unavailable` or hangs while perfectly healthy.

**Why it happens:**
"Debounce" is commonly understood as trailing-edge reset-on-event (the UI-programming mental model). Persistence debounce needs the opposite: bounded coalescing (leading-edge + trailing-edge, or trailing-edge with a max-wait cap). The bug is invisible in unit tests, which write once and wait.

**How to avoid:**
- Implement the debounce as "fire at most `dump_debounce` after the first dirty mark, coalescing everything that arrives meanwhile" — a max-wait cap, not a sliding delay.
- Define "a dump contains the write" by a **version watermark**: capture the max RecordVersion at dump-snapshot time; a write is confirmed when a completed dump's watermark ≥ the write's version. This removes any ambiguity about which writes "landed in" a given dump when writes race the snapshot.
- Test it in Phase 2: a burst writer at ≥1000 writes/s for several seconds must observe bounded write latency (debounce + one dump duration), not starvation.

**Warning signs:**
- Write latency grows with sustained write rate (p99 climbing to RPC deadline).
- Dump-duration metric shows no dumps during continuous writes; dump count ≪ write batches.
- Long "oldest unconfirmed write" age in metrics/logs.

**Phase to address:**
Phase 2 (Persistence) — the confirmation contract and watermark land with the dump code; burst test belongs in Phase 2's test list (benchmarks re-checked in Phase 5).

---

### Pitfall 2: Dump serialization while holding the map lock stalls the whole store

**What goes wrong:**
Serializing the full map to JSON (with base64 values) under the store's `RWMutex` blocks every read and write for the entire dump. At 10k keys × KiB values this is tens to hundreds of milliseconds; at the pessimistic envelope (tens of thousands of keys × up to 60 KiB values, base64 +33%) it is seconds — repeated on every debounced dump during bursts, the store becomes effectively unavailable even though the design promises "disk flush latency within the dump window is acceptable".

**Why it happens:**
The naive implementation iterates `map[string]Record` under `RLock` and writes to the file in the same critical section. It works fine at hundreds of keys (service-configuration scale) and only falls over at the top of the documented scale envelope — a classic works-in-demo trap.

**How to avoid:**
- Snapshot under a short `RLock`: copy the map (or a slice of key→record pointers plus the version watermark) into a snapshot structure, release the lock, then marshal and write outside it.
- Alternatives if copy cost matters: iterate keys under `RLock` into a slice of shallow record copies (records are immutable once versioned — copies share byte slices safely).
- Benchmark at the documented envelope (Phase 5's "dump at 10k keys" benchmark should use large values, not tiny ones) and assert dump holds the lock only for the snapshot step.

**Warning signs:**
- Read/write p99 latency spikes in phase with dump duration.
- Dump duration metric ≫ snapshot time; lock-held-time metric (if added) high.

**Phase to address:**
Phase 2 (structure the dump path correctly from the start); Phase 5 (benchmark proves it at envelope scale).

---

### Pitfall 3: Large DataReplication datagrams silently depend on IP fragmentation

**What goes wrong:**
`DataReplication` carries one full record in one datagram; values may be up to 60 KiB. On a 1500-byte-MTU segment that datagram is IP-fragmented into ~40 fragments. Losing **any one fragment loses the whole datagram** — effective datagram loss is amplified roughly to `1 − (1 − p)^N` (1% per-frame loss → ~33% datagram loss at 40 fragments). Linux reassembly has global/per-host limits (`ipfrag_high_thresh`, `ipfrag_time`) that can drop fragmented datagrams under bursts, and RFC 4963 documents reassembly failures at high data rates. Alertmanager hit exactly this: gossip payloads exceeding MTU were problematic enough that large state moved to TCP.

**Why it happens:**
UDP "supports" 65,507-byte payloads, so sending works and small-value tests pass. Fragmentation is invisible to the application — it only shows up as elevated datagram loss on real segments (and is invisible in a virtual network that models loss per-datagram, not per-fragment).

**How to avoid:**
- The ack/retransmit + periodic `DataSync` design already makes loss recoverable — that is the primary defense; keep it correct and tested with high loss rates for large values specifically.
- Know and document the envelope: fragmentation is *accepted* for values > ~1400 bytes on 1500-MTU segments; convergence bound for a large-value write is governed by `ack_timeout`/retransmit, not one RTT. Consider documenting a recommended value size < `sync_chunk_bytes` for lossy (especially Wi-Fi) segments.
- In the simulator, model loss per-fragment for oversized datagrams (or at least add a failure-scenario test where large-value datagrams have multiplied loss probability).
- Do **not** try to split datagrams in v1 mid-design; that is Alertmanager's "problematic" path. If measurements demand it, it is a deliberate future change.

**Warning signs:**
- Repeated retransmissions of the same (key, large-version) records while small-value records flow through.
- Unacked count spiking specifically after large-value writes; `netstat -su` reassembly failures (`ReasmFails` / `IpReasmTimeOut` counters).
- Convergence time for large values ≫ one RTT in real-network smoke tests while simulation shows RTT.

**Phase to address:**
Phase 3 (replication tests must include large-value loss scenarios); Phase 5 (failure-scenario suite + real-socket smoke validation); Phase 6 (document the envelope).

---

### Pitfall 4: Kernel UDP receive-buffer overflow — silent burst drops

**What goes wrong:**
Linux defaults `SO_RCVBUF` to ~208 KiB (`net.core.rmem_default` = 212992) and silently caps requests above `rmem_max`. At 1000 writes/s broadcast to a peer, the receiving kernel queue fills in a couple of milliseconds whenever the reader goroutine is briefly delayed (GC pause, dump snapshot, CPU contention) — datagrams are dropped with no error to the application. Symptoms look exactly like "mysterious datagram loss on fast networks" and the retransmit path hides them at the cost of convergence latency.

**Why it happens:**
UDP has no flow control; the kernel simply drops when the queue is full (`RcvbufErrors` in `netstat -su`). Nobody sets a read buffer in a greenfield Go service until they've been burned; the effect is rare at config-store rates (writes/hour) and appears under the burst target.

**How to avoid:**
- Call `conn.SetReadBuffer(4<<20)` (or make it a launch parameter); log the *effective* buffer size at startup (the kernel may cap it — surface that loudly).
- One dedicated read goroutine in a tight loop with a pre-allocated, reused buffer; do nothing expensive on that goroutine — hand parsed envelopes to a channel.
- Export a discarded-datagram metric; document checking `netstat -su` in ops docs.
- Add a failure-scenario/simulation proxy: a "slow reader" test at burst rate verifying recovery via retransmit, plus (on Linux) an integration assertion on effective socket buffer size.

**Warning signs:**
- Nonzero `RcvbufErrors` on hosts running the node during bursts.
- Convergence latency degrading with write rate even on a healthy LAN; retransmit counters climbing although the segment has no real loss.

**Phase to address:**
Phase 3 (socket setup + read loop structure land with replication); Phase 5 (burst scenario in failure suite; ops docs in Phase 6).

---

### Pitfall 5: HLC future-clock poisoning — a fast node deterministically wins everything

**What goes wrong:**
A node whose wall clock jumps far ahead (bad RTC, NTP step, VM restore from snapshot) emits versions with far-future `l`. Because HLC is monotonic and merge takes the max, the poisoned `l` propagates to all peers via `Update` and persists until **real time catches up with the bogus timestamp** (a 3-day jump ≈ 3 days of poisoning). During that window every write from every node loses LWW conflicts against the poisoned node's writes. Worse for sup-kv: HLC is initialized from the dump's max `l` at startup, so restarting the poisoned node does **not** clear it — the dump re-seeds the future `l`. CockroachDB treats excessive offset as fatal (self-termination) precisely to avoid this class; sup-kv's design has only `drift_warn` (60 s).

**Why it happens:**
HLC guarantees monotonicity, not sanity. "Correctness never depends on wall clocks" (true — convergence holds) is easily misread as "wall-clock health doesn't matter" (false — *which write wins* is skewed for the entire poisoning window, deterministically and identically everywhere).

**How to avoid:**
- Keep `drift_warn` and make it loudly actionable: metric + structured log with the current `l − pt` divergence and the offending node, documented as "fix the clock / isolate the node".
- Property-test adversarial clocks (already planned): forward jumps on live node, between restarts, far-ahead node — assert convergence *and* that divergence is visible in the drift metric.
- Document the recovery lever: the snapshot-restore path replays with **fresh** versions, which is the operator's tool to outrun a poisoned version set (see Recovery Strategies).
- Decide deliberately (and write it down) that sup-kv warns rather than refuses future timestamps — refusing would violate the AP promise; but the docs must say "under skew the LWW winner is arbitrary-but-deterministic".

**Warning signs:**
- `drift_warn` firing; `l − pt` divergence metric growing and never shrinking.
- Conflict journal showing one node winning a suspiciously high fraction of merges.
- After a known clock incident, writes on other nodes still losing conflicts.

**Phase to address:**
Phase 1 (HLC implementation + adversarial property tests + divergence metric surface); Phase 5 (far-ahead-node failure scenario); Phase 6 (operational documentation of the behavior and remedy).

---

### Pitfall 6: HLC implementation slips — counter reset and restart initialization

**What goes wrong:**
Two subtle bugs dominate HLC implementations:
1. **Counter never resets**: when physical time advances past `l`, `c` must reset to 0. If it doesn't, `c` ratchets ever upward; with only 12 bits (4096 values/ms in `rand_a`), the "counter overflow bumps `l` by 1 ms" path runs constantly, `l` drifts ahead of physical time *without any clock fault*, and versions slowly march into the future.
2. **Restart from wall clock instead of dump max**: initializing `l` from `time.Now()` on a node whose dump contains newer versions (e.g., clock stepped back while down, or versions learned from a once-fast peer) generates versions that *lose* to records already in the node's own dump — the node's local writes then mysteriously fail to apply to its own store after restart. The design already says "init from dump max `l`"; the pitfall is silently regressing to wall clock during refactoring, or forgetting to also consider versions learned only via replication that were not yet dumped (Ack-after-memory means the dump is the only durable source — write-confirmed-after-dump makes this safe *only if* the confirmation contract of Pitfall 1 holds).

**Why it happens:**
HLC is "tens of lines", and the edge cases live in exactly two conditionals (`l' = pt` ⇒ `c = 0`; startup `l₀ = max(dump_max, pt)`… with care: taking max is right, taking only dump max also freezes `c` handling). The happy path passes all casual tests; only adversarial-clock property tests catch the rest.

**How to avoid:**
- Implement HLC as its own small type with the property-test suite the design mandates: monotonicity under adversarial clock input (jumps backward/forward, stuck clock, post-receive ordering, uniqueness per write, counter-overflow behavior).
- Byte-comparison property: generated UUIDv7 byte order must equal `(l, c, NodeID)` order — test it directly on random sequences, not just by construction.
- RFC 9562 note: within one millisecond, batch-generated UUIDs must stay monotonic — the `c`-in-`rand_a` scheme must not re-randomize per identifier within the same `l`.

**Warning signs:**
- Property tests are the warning system; operationally — versions drifting ahead of wall clock with no clock fault, or freshly restarted nodes whose writes lose to their own dump contents.

**Phase to address:**
Phase 1 (this is the core of the Engine phase's test deliverable).

---

### Pitfall 7: Tombstone mistakes — resurrection via dump, delete-without-version, create-if-absent

**What goes wrong:**
Three concrete resurrection bugs in LWW-map implementations:
1. **Dump drops tombstones**: persisting only live values "to save space" means a restart forgets the delete; the next sync from a peer that still holds the pre-delete value (e.g., long partition) resurrects it — and the node itself answers `Get` with the deleted value.
2. **Delete without a fresh version**: implementing `Delete` as clearing `Data` while keeping the old `RecordVersion` makes the tombstone lose to any concurrent update (and to a *duplicate delivery* of the pre-delete update — which idempotent merge would otherwise discard). Delete must allocate a new HLC version exactly like `Set`.
3. **`CreateIfAbsent` checks key presence, not tombstone state**: after a delete the key *exists* as a tombstone; a presence-check implementation returns `AlreadyExists` for the build-time-defaults seeding pattern, or worse, treats the tombstoned key as absent but merges wrong.
Also: `Watch` must emit notifications for replicated deletes and lost merges, or operators watching for deletions see nothing.

**Why it happens:**
Tombstones are record-state, not absence — every natural instinct ("deleted means gone", "don't persist garbage") points the wrong way. The design's keep-tombstones-forever decision removes the GC variant of this bug (premature tombstone GC is the classic CRDT resurrection), but the three implementation slips above remain fully live.

**How to avoid:**
- Dump format includes `Deleted` from day one; Phase 2's round-trip test must cover delete → dump → load → `Get` = `NotFound` → merge with stale pre-delete record → still `NotFound`.
- Treat `Delete` internally as `Set(tombstone with fresh version)`; property tests assert a delete concurrent with a set resolves to exactly one deterministic winner on all replicas.
- `CreateIfAbsent` succeeds iff no record or record is a tombstone.
- CRDT merge property tests (commutative/associative/idempotent) must include tombstoned entries explicitly.

**Warning signs:**
- Deleted keys reappearing after restarts or partition heals; `AlreadyExists` from `CreateIfAbsent` on a deleted key; Watch consumers missing deletes.

**Phase to address:**
Phase 1 (map semantics + property tests), Phase 2 (dump round-trip with tombstones), Phase 4 (Watch delete notifications).

---

### Pitfall 8: Atomic-dump durability slips — rename without directory fsync, temp file on the wrong filesystem, macOS fsync

**What goes wrong:**
Three separate traps:
1. **`rename` without `fsync` of the parent directory**: rename is atomic but not durable; on ext4 a crash can lose the directory entry — the dump file reverts to the previous generation or vanishes, despite fsyncing the temp file first. This is the documented ext4 silent-data-loss class (PostgreSQL thread; Ted Ts'o's delayed-allocation write-up).
2. **Temp file created in `/tmp`** (or any directory on a different mount than `dump_path`): `os.Rename` fails `EXDEV` (cross-device) — every dump fails at runtime though unit tests on the dev machine pass because `/tmp` and the project dir share a filesystem.
3. **macOS**: `fsync` does **not** flush the drive cache; without `F_FULLFSYNC`, "confirmed writes always on disk" is weaker than on Linux. The CI matrix includes macOS — tests pass there while the durability guarantee differs from the Linux one.

**Why it happens:**
The POSIX recipe (write temp → fsync file → rename → fsync dir) has two steps most people never see fail; failure requires power loss (1) or a specific filesystem layout (2). Go's `os` package makes the wrong thing easy (`os.CreateTemp("", ...)` defaults to `/tmp`).

**How to avoid:**
- Create the temp file with `os.CreateTemp(filepath.Dir(dumpPath), ...)`; after rename, `fsync` the directory (open `O_RDONLY`, `Sync`, close).
- Wrap the whole dump in error handling: dump failure returns `Unavailable` per design — make sure `EXDEV`/`ENOSPC`/`EIO` reach that path and the metrics, never a panic or an ignored error.
- Pin the `.prev` rotation order (main → `.prev`) in the same transaction-ish sequence, and fsync the directory after the rotation too.
- Document macOS as a dev/CI platform where the durability contract is best-effort (or call `F_FULLFSYNC` via `golang.org/x/sys/unix` on Darwin — decide explicitly; it is one fcntl away and matches the "correctness from proven components" principle, at a latency cost).

**Warning signs:**
- Dump failures only in production layouts (`EXDEV`); after forced power-loss testing, missing or previous-generation dumps; `ls` showing temp files accumulating next to the dump (cleanup path broken after failed renames).

**Phase to address:**
Phase 2 (dump path implementation); Phase 5 (crash/kill −9 scenario in the failure suite exercising main → `.prev` → empty fallback).

---

### Pitfall 9: gRPC Watch fan-out — one slow client freezes the store

**What goes wrong:**
gRPC server streams are subject to HTTP/2 flow control: a slow (or stalled) Watch client's window fills and `SendMsg` blocks. If the fan-out loop sends to subscribers while holding the store mutex — or from a single goroutine shared across subscribers — one dead client head-of-line blocks **all** watchers and, via the mutex, every read and write. Separately, clients that die silently (no TCP RST — laptop sleep, network change) leave the stream open forever without server-side keepalive, leaking goroutines and subscriber slots.

**Why it happens:**
Watch is "just a loop over subscribers"; flow control is invisible in local tests where consumers are fast. gRPC's default server keepalive is not aggressive enough to reap silently-dead clients promptly, and client keepalive misconfiguration (pinging more often than the server's `PermitKeepaliveTime`, default 5 min without streams) gets connections killed with `GOAWAY ENHANCE_YOUR_CALM / too_many_pings` — the most common gRPC misconfiguration incident class.

**How to avoid:**
- Per-subscriber buffered channel; the fan-out under lock only does a **non-blocking send** (drop-oldest or drop-newest with a dropped counter). Watch semantics for a KV store tolerate gaps if clients can re-`Get`; alternatively close the subscriber on overflow (etcd/Kubernetes-watch style: disconnect stuck consumers).
- One dedicated sender goroutine per stream that drains the channel into `SendMsg`; blocked sender affects only that subscriber.
- Enable server keepalive (`keepalive.ServerParameters{Time, Timeout}`) so dead clients are reaped; if shipping a client example, set its keepalive within the server's enforcement policy.
- Define the notification contract: Watch is best-effort notification + authoritative `Get` (avoids promising ordered, gapless streams under chaos).

**Warning signs:**
- Store p99 latency correlated with a stuck watch client; goroutine count climbing over days; subscribers list growing without churn; `too_many_pings` in client logs.

**Phase to address:**
Phase 4 (Watch implementation lands with the queue/drop architecture, not after); Phase 5 (slow-consumer scenario in tests).

---

### Pitfall 10: Own-broadcast echo and multi-homed duplicate delivery mishandled by address, not identity

**What goes wrong:**
A node broadcasting on a segment receives **its own broadcasts back** (the stack loops broadcast frames to local listeners); on multi-homed hosts bound to the wildcard address, a broadcast can be delivered **once per eligible interface** (duplicates from distinct source addresses). Filtering "self" by source IP address breaks both ways: own echoes from an unexpected interface are processed (re-applying own records — harmless for merge but corrupts ack tracking: the node acks to itself, or worse, treats itself as a peer), and legitimate peers behind NAT/rebinding are dropped. Conversely, filtering *only* by `node_id` but ignoring `run_id` means a restarted node's stale Hello (delayed, duplicated) can flip the peer table back to the old run and re-trigger or miss the full-sync-toward-restarted-node behavior.

**Why it happens:**
In localhost demos every address is 127.0.0.1 and the wrong filter accidentally works. The differences only appear on real multi-interface hosts (Docker veth + physical NIC, VPN + LAN) — precisely the appliance form factor sup-kv targets.

**How to avoid:**
- Identity filter: drop envelopes where `node_id` == own (own-broadcast echo) — design already specifies this; never compare addresses for self-detection.
- Peer table keyed by `NodeID` with `{addr, run_id, last hello}` (as designed); on `run_id` change for a known node, trigger resync and **replace** the entry atomically; ignore Hellos older than the last seen run (run IDs are UUIDs — track latest, don't timestamp-compare them; "latest" = the one currently in the table, switching only on change, with a short dedupe window for in-flight stale datagrams).
- Duplicates from multiple interfaces are neutralized by merge idempotence, but ack tracking must key by `(NodeID, key, version)`, not by address — the same peer's two source addresses must share one unacked set entry.
- For the localhost quickstart: bind/broadcast guidance should use a subnet-directed broadcast address on loopback (e.g. `127.255.255.255`) — `255.255.255.255` behavior on loopback differs across OSes; the design's failure-scenario list already includes own-broadcast echo — keep that test on both a real two-process setup and in-simulation.

**Warning signs:**
- Node appears as its own peer; acks credited from unexpected addresses; peer table flapping run IDs; quickstart works on Linux but not macOS (or vice versa).

**Phase to address:**
Phase 3 (peer table + echo filtering); Phase 5/6 (multi-homed host noted in docs; own-echo scenario already mandated).

---

### Pitfall 11: Simulation passes while reality fails — virtual-network blind spots

**What goes wrong:**
The deterministic simulator's fault model (drop/duplicate/reorder/partition per *datagram*) quietly omits the failure modes that real stacks add: per-**fragment** loss amplification for oversized datagrams (Pitfall 3), kernel receive-buffer overflow (Pitfall 4), per-interface duplicate delivery and own-echo (Pitfall 10), silent truncation, checksum-drop classes, and GC/CPU scheduling stalls on the read loop. Everything converges in simulation, then the quickstart on two real processes on one host hits convergence stalls. The inverse trap also exists: a single deterministic run explores a tiny fraction of possible event orderings — passing seeds prove nothing about the unexplored traces.

**Why it happens:**
A virtual network is a *model*; whatever it doesn't model can't fail. Deterministic simulation (FoundationDB, Tigerbeetle, Antithesis practice) is the right backbone — but its practitioners pair it with real-socket smoke tests and randomized fault schedules precisely because of this gap.

**How to avoid:**
- Make the failure-scenario suite (Phase 5) run against the **real** UDP stack as well as the simulator: the quickstart scenario (two processes, localhost broadcast socket, kill −9, restart, converge) belongs in CI, not just docs.
- Run simulator scenarios across many seeds with randomized fault schedules (loss *rate*, not just fixed drops; duplicates; reordering windows), asserting the documented convergence bounds.
- Add simulator fault classes that mirror reality's quirks: corrupt-datagram (bad crc), oversized-datagram multiplied loss, burst-to-slow-reader.
- Treat any behavior only reproducible in simulation (or only on real sockets) as a model bug to fix, not a flake to skip.

**Warning signs:**
- CI green on simulator scenarios while the manual quickstart intermittently stalls; convergence bounds documented from simulation timings that real hosts exceed; tests that only pass for specific seeds.

**Phase to address:**
Phase 3 (simulator + virtual network design includes the fault-class inventory); Phase 5 (failure-scenario suite incl. real-socket runs); Phase 6 (quickstart in CI/docs).

---

### Pitfall 12: Confusing "no peers known" with "safe to suspend/drop" — retransmission-set and unacked-accounting traps

**What goes wrong:**
The design suspends retransmission while no peer is known and drops records from the unacked set on `Ack` or supersession. Two slips commonly occur:
1. **Dropping instead of suspending**: when the peer table empties (partition), pending unacked records are discarded "because there's nobody to send to" — those writes then exist only locally and on whatever peer got them before the partition; after heal, convergence relies solely on the next `DataSync` — the documented "loss ≤ `ack_timeout_max`" bound is silently violated for that window.
2. **Forever-unacked cold keys under one-way connectivity**: if Acks from a peer are persistently lost (asymmetric connectivity), cold keys (written once, never superseded) never leave the unacked set — retransmission runs at `ack_timeout_max` forever, and the "oldest unacked age" metric grows without bound, generating permanent operator alarm for a condition the anti-entropy sync actually heals.

**Why it happens:**
Retransmission logic is usually written and tested for the happy bipartite case; the degenerate cases (zero peers, half connectivity) are afterthoughts. `wait_for_replication` semantics ("never changes the write outcome") also tempt implementations to treat ack accounting as advisory and let it drift.

**How to avoid:**
- Suspend (stop the clock), never discard, unacked records when the peer table is empty; resume on the next Hello.
- Define the operational meaning of unacked-age under sustained Ack loss (expected during partitions): document it, distinguish "partitioned" from "degraded" in metrics (peer-present-but-not-acking vs no-peer), and let `DataSync` reconciliation be the acknowledged backstop.
- Failure-scenario tests already mandated: Ack loss and one-way connectivity — extend them to assert the unacked set drains *after heal* and that no confirmed write is lost during a zero-peer window.

**Warning signs:**
- Unacked count dropping during partitions (suspension bug); unacked-oldest-age monotonically growing weeks after a network incident; writes present on one node only after a partition+heal cycle.

**Phase to address:**
Phase 3 (retransmission + peer-table lifecycle); Phase 5 (Ack-loss/one-way scenarios asserting set behavior, not just convergence).

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Serialize dump under the map lock | Simpler code, no snapshot copy | Store-wide stalls at scale envelope (Pitfall 2) | Never — the snapshot pattern is ~20 lines; do it in Phase 2 |
| Drop tombstones from dump ("they're not data") | Smaller dumps | Resurrection bugs (Pitfall 7) | Never |
| String-match on error text (`"use of closed network connection"`) | Works today | Breaks across Go versions/platforms; `errors.Is(err, net.ErrClosed)` is the stable API | Never |
| Filter own broadcasts by IP address | Passes localhost demo | Breaks on multi-homed hosts and real NICs (Pitfall 10) | Never |
| Reset-style debounce (sliding delay) | Trivial implementation | Write starvation under bursts (Pitfall 1) | Never for the confirm path |
| Fixed-seed simulator scenarios | Deterministic, fast CI | Tiny trace coverage; false confidence (Pitfall 11) | Only as *additions* to randomized-seed runs |
| Skip directory fsync after rename | One syscall saved | Dump can vanish/revert on power loss (Pitfall 8) | Never |
| Ack tracking keyed by address | One less map lookup | Double-counted peers on multi-homed/rebinding hosts | Never |
| `time.After` inside retransmit loops | Concise code | Pre-1.23 timer leaks at 5 timers/s/key scale | With Go ≥1.23 semantics, acceptable; else reusable Timer |
| Test assertions on map-iteration output | Quick to write | Flaky tests from randomized iteration order | Never — sort keys or compare maps |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| LAN broadcast segment | Assuming broadcast is reliable enough to skip retransmit tests; assuming Wi-Fi behaves like Ethernet (broadcast at lowest rate, unacknowledged, AP filtering/conversion can drop large fractions) | Treat broadcast as a lossy medium by default; keep ack+retransmit+DataSync; document wired-segment recommendation |
| Loopback (quickstart/CI) | Using `255.255.255.255` on loopback; expecting identical echo behavior across OSes | Subnet-directed loopback broadcast (e.g. `127.255.255.255`); test on both Linux and macOS CI runners |
| systemd | Service starts before network is up: bind/broadcast fails, node exits or runs deaf | `After=network-online.target`; retry bind with backoff; socket activation unnecessary — but socket close semantics must handle stop cleanly (`net.ErrClosed`) |
| Docker/containers | Assuming broadcast works across the bridge; tests that pass on host network fail in bridge networks | Run replication tests with host networking or a shared network namespace; document supported topologies |
| NTP/chrony | Assuming clocks slew, never step; test clocks never step | Property-test stepped clocks both directions (design mandates this); document recommended NTP config (slew-preferred) in ops docs |
| macOS CI runners | Assuming `fsync` durability (Pitfall 8) and identical broadcast/UDP behavior | Explicitly decide F_FULLFSYNC policy; keep macOS as compile/test platform, treat Linux as the durability reference |
| etcd/Consul-adjacent tooling | Operators expecting quorum semantics/labels from those systems | Docs that state AP semantics plainly: both nodes writable, deterministic LWW winner, no fencing |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|-----------------|
| Broadcast amplification (N nodes × every local change) | Datagrams/s grows quadratically with cluster size × write rate | Envelope per change is by design; keep N small (documented niche 2–a handful); retransmit backoff; measure replication throughput benchmark | N ≥ 5–6 at 1000 writes/s |
| Dump cost at envelope (10k keys × 60 KiB, base64 +33% → ~800 MB JSON) | Multi-second dumps; disk fill; confirm-path latency | Snapshot-outside-lock (Pitfall 2); watermark confirmation; benchmark with large values | ~tens of thousands of large keys |
| fsync latency spikes gate write confirms (1–4 ms+ SSD, journal commit dominated; worse on eMMC/SD appliance boards) | Write latency p99 = debounce + fsync | Coalescing is the design's own answer (bursts → one dump); make the debounce bounded so coalescing actually engages (Pitfall 1) | Sustained bursts on slow flash |
| Per-datagram allocations in the read loop | GC pauses → RcvbufErrors (Pitfall 4 compounds) | Reused read buffer; pre-allocated unmarshal targets; allocation benchmarks in Phase 5 | ≥1000 datagrams/s |
| Watch fan-out copying payloads under the store lock | Store latency tracks slowest watcher | Non-blocking send of small notifications; subscribers re-`Get` for data (Pitfall 9) | Any stalled Watch client |
| Hello broadcast every 1 s forever | Negligible — listed to prevent "optimization" of it into event-driven-only discovery (breaks partition detection) | Keep interval Hellos; they are the heartbeat | If removed: `peer_timeout` never fires |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Treating CRC32 as integrity against attackers | Anyone on the segment can craft valid-crc envelopes (e.g., future-timestamp versions that poison LWW — a denial-of-service via Pitfall 5, or state injection) | Document the trusted-segment trust model prominently; HMAC is already planned future work — keep the envelope format HMAC-ready (field reserved) |
| Dump/snapshot files world-readable | Keys/values may hold service secrets | Create dumps with 0600 perms; document directory perms for `dump_path` |
| Metrics/logs leaking values | Human-readable artifacts invite grepping secrets into logs | Conflict journal logs versions/keys by default, not Data; document |
| Duplicate `node_id` treated as normal operation | Silent split-brain of identity: two nodes answering as one, peers flapping | Design already: log as config error, drop datagrams — also alert (counter + distinct log class), never auto-resolve |
| gRPC endpoint exposed without any auth on a "trusted" LAN | Same trust model as UDP, but HTTP-ish surface invites scanning | Bind gRPC to the management interface or localhost by default; document |

## UX Pitfalls (Operator Experience)

| Pitfall | User Impact | Better Approach |
|---------|-------------|------------------|
| `Unavailable` used for both startup gate and dump failure with no distinguishing signal | Operator cannot tell "wait 1 s" from "disk is broken" | Same error code (per design) but distinct structured-log reason + metric counter per class |
| Conflict journal that exists but is off/hard to read | Silent LWW losses erode trust in the store | On by default, one line per lost version, greppable format documented in Phase 6 |
| Metrics that alarm during normal partitions | Unacked-age grows during every partition = page fatigue | Distinguish partitioned (no peers) from degraded (peers present, no acks) states (Pitfall 12) |
| Human-readable dumps invite hand-editing | Operator edits the JSON, checksum mismatch, records silently discarded on load | Document the CRC constraint loudly in the dump format docs; discard-with-log per record (designed) |
| Drift warning without a remedy | Operator sees clock-drift alert, no runbook action | Runbook: fix clock / isolate node / snapshot-restore replay to outrun poisoned versions |

## "Looks Done But Isn't" Checklist

- [ ] **Own-broadcast echo:** verify datagrams from own `node_id` are dropped *by identity* and counted — test on a real two-process localhost setup, not only in simulation
- [ ] **Kill −9 during dump:** verify main dump is never partial (atomic rename) and `.prev` fallback actually loads — exercise the full chain main → `.prev` → empty, not just "corrupt record discarded"
- [ ] **Delete round-trip:** delete → dump → restart → `Get` = `NotFound`; then merge a stale pre-delete replica record → still `NotFound` (tombstone persisted and versioned)
- [ ] **Burst durability:** 1000 writes/s for ≥5 s with bounded latency — proves the debounce is capped, not sliding (Pitfall 1)
- [ ] **Watch under chaos:** replicated changes and deletes notify watchers; a stalled consumer does not stall the store
- [ ] **One-way connectivity:** unacked set survives (not dropped) during no-peer windows and drains after heal
- [ ] **Restart with regressed clock:** HLC inits from dump max, node's new writes still apply locally (Pitfall 6)
- [ ] **Large values on a real network:** 60 KiB value converges within retransmit bounds on a lossy simulated segment (fragment-amplified loss modeled)
- [ ] **Effective socket buffer:** startup log shows the actual `SO_RCVBUF` the kernel granted, not the requested value
- [ ] **Cross-platform CI:** quickstart passes on macOS as well as Linux (broadcast + fsync differences surface here)
- [ ] **Stale Hello:** duplicated/delayed Hello with an old `run_id` does not flip the peer table or re-trigger spurious full syncs
- [ ] **Proto mismatch:** mismatched `proto_version` dropped + counted, never parsed; mixed-version upgrade (additive field) documented

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| HLC future poisoning (Pitfall 5) | MEDIUM | Fix the offending node's clock; since HLC is monotonic the poisoned `l` persists — either wait for real time to catch up, or use the snapshot-restore path: restore replays records with fresh versions that outrun the poisoned ones (design's own mechanism); verify with conflict journal |
| Debounce starvation deployed (Pitfall 1) | LOW | Code fix is small; while waiting, quiesce writes (the failure is latency, not data loss — write stays in memory and replicates) |
| Tombstone resurrection incident (Pitfall 7) | MEDIUM | Re-issue the delete (fresh version wins everywhere); audit conflict journal for other resurrected keys; fix the underlying dump/delete bug |
| Dump lost/corrupted on disk (Pitfall 8) | LOW | `.prev` fallback, else empty start + full sync from peers (designed); operator action: none if peers hold state |
| Stuck Watch clients (Pitfall 9) | LOW | Server keepalive reaps them; restart the client; fix fan-out to non-blocking sends |
| Own-echo/peer-table corruption (Pitfall 10) | LOW | Restart the node (new RunID triggers clean resync); fix identity filtering |
| Fragmentation-driven loss storms (Pitfall 3) | LOW | Reduce value sizes or tolerate retransmit latency; convergence holds by design — it is a latency issue, not a correctness one |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| 1. Debounce starvation / watermark confirm | Phase 2 | Burst-write test: bounded latency at 1000 writes/s ≥5 s |
| 2. Dump under lock | Phase 2 | Phase 5 benchmark at 10k keys with large values; lock-held-time does not scale with dump size |
| 3. Fragmentation loss amplification | Phase 3 / 5 / 6 | Large-value loss scenario in simulator (fragment-scaled loss); real-segment smoke; documented value-size guidance |
| 4. SO_RCVBUF overflow | Phase 3 / 5 | SetReadBuffer at socket open + effective-size log; slow-reader burst scenario recovers via retransmit |
| 5. HLC future poisoning | Phase 1 / 5 / 6 | Adversarial-clock property tests; far-ahead-node failure scenario; drift runbook |
| 6. HLC counter/init slips | Phase 1 | Property suite: monotonicity, uniqueness, post-receive ordering, restart-from-dump-max, overflow bump |
| 7. Tombstone mistakes | Phase 1 / 2 / 4 | Merge property tests with tombstones; delete-dump-restart round-trip; Watch delete notification test |
| 8. Atomicity/durability slips (dir fsync, EXDEV, macOS) | Phase 2 / 5 | Temp-file-in-target-dir; kill −9 scenario exercises main→`.prev`→empty; documented platform durability matrix |
| 9. Watch fan-out blocking | Phase 4 / 5 | Slow-consumer test: store latency unaffected, stuck subscriber reaped/dropped |
| 10. Broadcast echo/multi-homed handling | Phase 3 / 5 | Own-echo scenario (mandated) on real sockets; identity+run_id keyed peer table; macOS+Linux quickstart |
| 11. Simulation-vs-reality gap | Phase 3 / 5 / 6 | Real-socket quickstart in CI; randomized-seed fault schedules; fault-class inventory matches documented discard counters |
| 12. Suspend-vs-drop and forever-unacked | Phase 3 / 5 | One-way connectivity and Ack-loss scenarios assert unacked-set behavior, not only final convergence |

## Sources

- UDP sizing/fragmentation: [Stack Overflow — largest safe UDP packet size](https://stackoverflow.com/questions/1098897/what-is-the-largest-safe-udp-packet-size-on-the-internet), [shadow #3199](https://github.com/shadow/shadow/issues/3199), [RFC 4963 — IPv4 Reassembly Errors at High Data Rates](https://datatracker.ietf.org/doc/rfc4963/), [Cisco — Fragmentation, MTU, MSS, PMTUD](https://www.cisco.com/c/en/us/support/docs/ip/generic-routing-encapsulation-gre/25885-pmtud-ipfrag.html), [Packet Pushers — IP Fragmentation in Detail](https://packetpushers.net/blog/ip-fragmentation-in-detail), [quiche #2649](https://github.com/cloudflare/quiche/issues/2649) (MEDIUM)
- UDP buffer tuning: [Red Hat — Tuning UDP connections](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/10/html/network_troubleshooting_and_performance_tuning/tuning-udp-connections), [Baeldung — UDP socket buffer in Linux](https://www.baeldung.com/linux/udp-socket-buffer), [Stack Overflow — max UDP receive buffer](https://stackoverflow.com/questions/16460261/linux-udp-max-size-of-receive-buffer) (MEDIUM)
- Broadcast quirks: [Loopback UDP broadcasts — Lauri Koskela](https://www.laakkonen.io/), [Stack Overflow — broadcast in multiple processes on one NIC](https://stackoverflow.com/questions/33450330/udp-broadcast-packets-in-multiple-processes-bound-to-one-nic), [ITNEXT — Socket options in Go](https://itnext.io/socket-options-go-multiple-listeners-one-port-7e5257044bb1), [SO — UDP multicast wire→wireless](https://stackoverflow.com/questions/72500245/udp-multicast-does-not-work-when-sending-from-wire-to-wireless), TI E2E / Cisco community threads on AP broadcast loss (MEDIUM)
- Go UDP semantics: [Golangbridge — buffer smaller than datagram discards data](https://forum.golangbridge.org/t/reading-from-udp-socket-blocks-even-if-there-is-data/3290), [SO — gracefully stop ReadFromUDP](https://stackoverflow.com/questions/37277281/golang-gracefully-stop-readfromudp-when-udp-connection-closes), grpc-go #7388 (`net.ErrClosed` best practice) (MEDIUM)
- HLC/clocks: Kulkarni et al. 2014 (HLC paper, cited in design), [CockroachDB — Living Without Atomic Clocks](https://www.cockroachlabs.com/blog/living-without-atomic-clocks/) (fetched, HIGH relevance), Daniel Imfeld — Hybrid Logical Clock (future-timestamp poisoning), Sergei Turukin — HLC overview (partial order caveat) (MEDIUM)
- CRDT/LWW: Shapiro et al. CRDT survey / crdt.tech (cited in design); resurrection and skew anomalies cross-checked across CRDT literature summaries (MEDIUM)
- Durability: [PostgreSQL — silent data loss with ext4](https://www.postgresql.org/message-id/56583BDD.9060302@2ndquadrant.com), [Ted Ts'o — delayed allocation and the zero-length file problem](https://thunk.org/tytso/blog/2009/03/12/delayed-allocation-and-the-zero-length-file-problem/), [LWN — fast commits for ext4](https://lwn.net/Articles/842385), misfra.me "Darwin's Deceptive Durability" + SQLite-on-macOS F_FULLFSYNC discussions (macOS item: LOW-MEDIUM — single-main-source cluster) 
- gRPC: [grpc.io — Flow Control](https://grpc.io/docs/guides/flow-control/), [grpc.io — Keepalive](https://grpc.io/docs/guides/keepalive), [Ably — gRPC stream performance gotcha](https://ably.com/blog/grpc-stream-performance), [SO — fast sender slow receiver](https://stackoverflow.com/questions/77003723/grpc-client-streaming-with-fast-sender-and-slow-receiver), Evan Jones — "gRPC is easy to misconfigure" (MEDIUM)
- Alertmanager precedent: [secure-cluster-traffic design doc](https://github.com/prometheus/alertmanager/blob/master/doc/design/secure-cluster-traffic.md) (UDP MTU → TCP for large state), [Prometheus HA docs](https://prometheus.io/docs/alerting/latest/high_availability/) (settle time), Robust Perception + "Alertmanager Has Amnesia" (gossip loss pain) (MEDIUM)
- Simulation testing: [Antithesis — Deterministic Simulation Testing](https://antithesis.com/docs/resources/deterministic_simulation_testing/), Will Wilson — Testing Distributed Systems with Deterministic Simulation (trace-coverage limitation) (MEDIUM)
- Protobuf evolution: [protobuf.dev — Proto Best Practices](https://protobuf.dev/best-practices/dos-donts), unknown-fields preservation since 3.5 (MEDIUM)
- Go tooling: [Go issue #37196 — timer/ticker channel semantics](https://github.com/golang/go/issues/37196), Go 1.23 unbuffered timer channels write-ups, [SO — race detector goroutine limit](https://stackoverflow.com/questions/34936571/), [SO — sorting map keys](https://stackoverflow.com/questions/25350710/if-maps-cant-be-sorted-in-go-lang-what-s-the-right-way-to-do-it) (MEDIUM)
- UUIDv7: RFC 9562 §6.2 monotonicity/counter recommendations (batch monotonicity warning) (MEDIUM)
- Design-derived items (Pitfall 1 watermark, Pitfall 6 restart-init, Pitfall 7 create-if-absent, Pitfall 12 suspend-vs-drop, create-if-absent seeding race): analysis of `docs/IDEA.md` — LOW until covered by phase tests (the design's own failure-scenario list already mandates most of them)

---
*Pitfalls research for: leaderless CRDT replicated KV store (sup-kv)*
*Researched: 2026-10-31*
