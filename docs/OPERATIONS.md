# sup-kv operations

The operational contract of a sup-kv cluster in one file: what the system
promises, what it does not promise, which knobs exist, and what to do when
the clocks drift. For trying the API see [docs/EXAMPLES.md](EXAMPLES.md);
for the full design see [docs/IDEA.md](IDEA.md).

## Trust model

**v1 ships with no authentication and no encryption on the replication
wire.** The replication segment is trusted: every node on the UDP broadcast
segment you configure is assumed friendly. CRC32 in the datagram envelope
guards integrity only — it detects accidental corruption in transit; it is
not tamper protection. Authenticating datagrams (HMAC with a shared key) is
documented future work, not a v1 feature.

The same honesty applies to the client surface: the gRPC API has no
authentication and no authorization — any client that reaches a node's gRPC
address can read and write everything. The default `--grpc-addr` binds
loopback only; bind it to a wider address only on a segment where every
reachable client is trusted.

One disclosure belongs beside the trust model rather than in a tuning
guide: **gRPC server reflection is always on.** It exposes the full service
schema (every method and message shape) to any client that reaches the gRPC
port. This is accepted deliberately: the default bind is loopback, the
segment is trusted by design, and the schema is public in this repository
anyway — `grpcurl` and other reflection clients rely on it (that is how
[docs/EXAMPLES.md](EXAMPLES.md) works with zero proto files).

The three absences, stated together: **no authentication** — no client or
peer identity is verified anywhere in the binary; **no authorization** — any
client that reaches a node's gRPC address can read and write everything;
**no transport security** — the replication wire and the gRPC listener are
plaintext, and the optional mTLS certificate triple on the metrics endpoint
(`SUPKV_METRICS_CERT`, `SUPKV_METRICS_KEY`, `SUPKV_METRICS_CLIENT_CA`) is
the only TLS surface the binary has.

## What you are operating

sup-kv is a leaderless, always-writable (AP) replicated in-memory store.
Every node accepts reads and writes locally and independently; replicas
exchange changes over UDP broadcast and converge to identical state — the
pointwise supremum of all writes under the record-version order (a
Last-Write-Wins-Element-Map CRDT). There is no leader, no quorum and no
coordination: losing any single node never blocks reads or writes anywhere.

Convergence bounds (proven in simulation under datagram loss, duplication,
reordering and partitions — `TestBound_normalOneRTT`,
`TestBound_lossWithinAckTimeoutMax` and `TestBound_partitionHeal` in
`internal/app/repl/sim_bound_test.go`, with the paced transport under load by
`TestRed_R01_pacedSync20k` and `TestRed_R02_burst` in
`internal/stand/stand_red_test.go`):

- normal operation: a write is visible on every peer within one round trip;
- datagram loss: retransmission recovers a loss episode whose delivery
  opportunity falls inside the ladder's window — the ladder guarantees
  attempts at 200/600/1400/3000 ms from the write, all inside the 5 s default
  `ack_timeout_max` ceiling; sustained loss past the ladder's window is the
  partition-heal bound's territory below;
- after a partition heals: at most `2 × sync_interval` + response time
  (default `sync_interval` is 30s, so a healed partition converges within
  about a minute without operator action).

## Scale envelope

sup-kv is sized for small clusters and small values:

- values are at most 60 KiB by default — 61440 bytes, the
  `storage.ValueMaxBytes` default of `--value-max-bytes` (environment twin
  `SUPKV_VALUE_MAX_BYTES`). A configured limit may raise it, never past the
  64 408-byte datagram bound below. A write with a larger value is rejected
  with `InvalidArgument`;
- the whole state lives in RAM — thousands to tens of thousands of keys is
  the design envelope;
- replication occupies one UDP broadcast segment — no WAN, no explicit peer
  lists.

## Record and datagram limits

Every accepted record must fit one UDP datagram, and the write API enforces
the two bounds that make that true:

- **Keys are at most 1 024 bytes**, measured in bytes of the key string
  (not runes). `Set`, `SetIfAbsent` and `Delete` reject a longer key with
  `InvalidArgument`, and the replication receive path discards an inbound
  datagram that carries one (`key_too_long`). `Get` and `List` stay
  permissive so records written by an older version under a longer key
  remain readable.
- **Values are at most the one value limit.** `storage.ValueMaxBytes` is
  the single default — 61 440 bytes — and a configured
  `--value-max-bytes` (environment twin `SUPKV_VALUE_MAX_BYTES`) is the
  effective limit. A zero or negative `--value-max-bytes` is malformed
  configuration: the node exits with code 2 and names the flag
  (`value-max-bytes must be positive`) instead of falling back to the
  default.
- **The combined record must fit 65 507 bytes** — the maximum IPv4 UDP
  payload. With a maximal 1 024-byte key the largest fitting value is
  64 408 bytes: 1 024 + 64 408 plus the envelope overhead is exactly the
  ceiling. At startup the node measures the worst case through the real
  codec; a `--value-max-bytes` that cannot fit exits with code 2 and an
  error naming the flag, the key limit and the computed size. The default
  61 440 leaves roughly 3 KiB of headroom.
- **`--max-clock-lead`** — environment twin `SUPKV_MAX_CLOCK_LEAD`,
  default 2m — bounds how far ahead of local time a received record version
  may be. Keep the same value on every node: a node with a smaller bound
  discards versions its peers still consider valid. Startup requires
  `drift_warn < max_clock_lead` at the record layout's millisecond
  resolution — `max_clock_lead` must be at least `drift_warn` plus one
  millisecond — so the drift warning below is reachable: a leading version
  is discarded by this bound before the warning could fire.

## Configuration validation

One validation point owns every range and relation. A violation is a
configuration error: the node exits with code 2 and names the offending
parameter, and never falls back to a default silently. The complete flag
table — every registered flag with its default and its environment twin — is
[docs/CONFIGURATION.md](CONFIGURATION.md); this section states the validation
behavior only. Zero or negative
durations and rates are rejected (all offenders in one pass), alongside the
bounds above. The relations:

- `--retransmit-interval` is at least 1ms — a nanosecond scan period is a
  busy loop;
- `--peer-timeout` must exceed `--hello-interval` — a peer evicted at the
  same instant it greets would never stay discovered;
- `--ack-timeout` must not exceed `--ack-timeout-max` — the retransmit
  ladder's ceiling sits at or above its base, and equality is a single rung;
- `--sync-chunk-bytes` must be within 1..65507 — a chunk budget above the
  largest IPv4 UDP payload describes a chunk no datagram can carry;
- `--max-clock-lead` must be at least `--drift-warn` plus 1ms, as stated
  under the record limits above.

The four addresses (`--bind-addr`, `--broadcast-addr`, `--grpc-addr` and
`--metrics-addr`) must be `host:port` with a port in 0..65535. Port 0 is
legal on a listener (`--bind-addr`, `--grpc-addr`, `--metrics-addr`) for an
ephemeral bind; `--broadcast-addr` is a destination, so port 0 there is
rejected — the kernel can never send to it. `--grpc-addr` must not be empty:
an empty value would bind the unauthenticated API to every interface on an
ephemeral port (only `--metrics-addr` has the empty "no listener" opt-in).

## Load outcomes

At startup the node walks the fallback chain — the main dump file, then its
`.prev` generation — and reports exactly one outcome:

- **Prompt** — neither file exists. A fresh start: no WARN and no ERROR,
  just the normal empty-node startup. The node recovers its state from
  peers;
- **Main** — the main generation decoded. One INFO `Dump load` record
  carries `outcome=main`, the accepted `generation`, the loaded `records`
  count and the `discarded` census;
- **Backup** — the main generation was absent or unusable and `.prev`
  decoded. Exactly one WARN names the main path and the cause, so an absent
  main is reported as `fs.ErrNotExist` and an unreadable or undecodable one
  as its read or decode error, together with the candidate's
  `preserve_error`. The loaded `.prev` survives: the corrupt main left the
  rotation path before any save, so the next rotate finds nothing to
  displace it;
- **Unusable** — no candidate decoded. Exactly one ERROR, carrying one
  `<candidate>_path`/`<candidate>_cause`/`<candidate>_preserve_error` group
  per candidate that actually failed (one or two, never a phantom). The
  node starts empty and recovers from peers.

A generation written before this format's `record_count` field carries no
completeness census, so a truncation that lands exactly on a line boundary of
such a legacy dump is undetectable and loads as authoritative. A snapshot
taken right after a clean `Main` load copies that legacy generation
byte-for-byte and therefore carries no census either; only a confirmed write
(or a flush of a dirty engine, which rewrites the main generation) produces a
censused main. After upgrading, let one confirmed write rotate the main
generation before relying on the census.

Discarded records are counted by class in
`supkv_dump_records_discarded_total`: `checksum`, `version`, `malformed`
and `value_limit`. A record skipped because it exceeds the configured value
limit is not evidence — it stays in the dump file until normal rotation
replaces that generation — and the skip is counted and logged once with the
count and the limit. A non-zero total of the other three classes is
reported in one startup line.

A candidate the load cannot accept is preserved, never deleted: it is
renamed out of the rotation path before any save, to `<path>.corrupt-<UTC>`
where the stamp is the UTC instant of the rejecting load, with `-1`, `-2`,
… appended when two loads fall in the same second, so earlier evidence is
never overwritten. These files are never rotated and never auto-deleted;
an operator inspects them and removes them by hand when the evidence is no
longer needed.

**The preservation-failure hazard.** When the load line's `preserve_error`
attribute is non-empty, the rename itself failed and the corrupt generation
may still be sitting at `main`. A later successful save would then rotate
it over `.prev`, destroying the last good generation — exactly what
preservation exists to prevent. `preserve_error` is the signal to act:
move the file it names out of the dump path before that save.

**The startup orphan-temp sweep.** Every in-flight dump writes a temp file
carrying the hidden `.tmp~` marker; `~` is outside the snapshot name
charset, so no legal dump, `.prev` or `.corrupt-*` name and no operator
snapshot name can contain it. At startup the node removes only regular
files carrying that marker whose modification time is older than one hour,
in the dump directory and the snapshot directory beside it. It never
removes the main file (whatever its name), `.prev`, a `.corrupt-*`
generation or an operator snapshot. The removal is one INFO line with the
count; a sweep error is one WARN and never aborts the load.

## Memory model

A dump cycle takes exactly one `Records()` deep copy and streams one encoded
record line at a time through the adapter's 64 KiB buffered writer, so the
encoded generation is never materialized as a second buffer. One residual
pause remains, and it is stated rather than claimed away: `Records()` clones
every value while holding the store's read lock, so writers are paused for
one state-sized copy at snapshot time; eliminating it needs a copy-on-write
view of the store and is residual work, not a property this model claims.
The formula, the measured state→RSS points and the dump-transient gate live
in [docs/PERFORMANCE.md](PERFORMANCE.md#memory-model).

## Read semantics

Reads are local: a `Get` is served from the node it lands on, with no
cross-node coordination. You get read-your-writes on the same node only — a
write confirmed by node A is immediately visible on A, and visible on every
peer within the convergence bounds above. Clients that must read what they
just wrote keep talking to the same node.

A node stays closed — reads and writes both answer `Unavailable` — until it
has received a complete key listing from a peer or its startup gate elapses
(health stays `NOT_SERVING` until then). A node that loaded records from
disk uses `--startup-gate` (environment twin `SUPKV_STARTUP_GATE`, default
1 s); a node whose disk was empty uses `--startup-gate-empty` (environment
twin `SUPKV_STARTUP_GATE_EMPTY`, default 60 s, empty-disk nodes only), so a
cold node waits longer for a peer. A restarted node may briefly miss peer
writes that happened while it was down, until replication catches up —
bounded by the same convergence bounds.

## Not a lock service

sup-kv is not a distributed lock service and not a leader-election service.
Locks need linearizability and fencing — impossible on an AP
last-write-wins store. Concurrent writes to the same key are resolved
deterministically by last-write-wins: every replica converges to the same
winner, picked by the record version (a hybrid logical clock), never by
which replica you ask.

A merge loser is an ordinary overwrite, not an event: there is no conflict
journal and none is needed. The observable is the
`supkv_merges_conflicts_total` counter on the metrics endpoint.

## Clock drift runbook

**Symptom.** `supkv_clock_drift_warnings_total` rises on one node while its
peers stay flat, and one WARN `Clock drift` record appears per peer per
rate-limit interval. A warning is counted when a received version leads
local wall time by more than `--drift-warn` (default 60s; environment twin
`SUPKV_DRIFT_WARN`): the received version's own `unix_ts_ms` reading is
compared with the local clock, for every received record and for each
DataSync listing part's maximum entry version. The counter counts every
offending record; the WARN is rate-limited per (peer, `clock_drift`), so a
peer flooding leading versions cannot flood the log.

**Cause.** The warning node's wall clock runs behind the sender's — the
received version leads local time. Note the direction: a node whose clock
runs ahead does not warn itself; its peers do, because they receive its
leading versions.

**Action.** Fix the clock source — NTP or PTP — on the host that warns. No
sup-kv-side repair exists, and none is needed: correctness never depends on
wall-clock synchronization. Clock skew only affects which simultaneous
write wins, and it does so deterministically and identically on all
replicas. A node whose clock runs far ahead will see its writes dominate
(win every concurrent conflict) until the peers' clocks catch up; once the
clock is corrected, new writes resume normal ordering. This behavior is
proven by `TestScenario_farAhead` in `internal/app/repl/sim_scenario_test.go`
(simulated far-ahead nodes converge deterministically).

Do not restart the cluster to "fix" drift: it changes nothing. Record
versions are hybrid logical clocks, not raw wall-clock readings, so the same
version order applies after any restart.

## Monitoring

The metrics/debug endpoint is opt-in. Pass `--metrics-addr` (environment
twin `SUPKV_METRICS_ADDR`); when it is unset, no listener exists at all.
The endpoint serves:

- `/metrics` — Prometheus text format from an in-repo exporter: the
  `supkv_*` families below plus the standard `go_*`/`process_*` runtime
  families. The complete-catalog rule holds for every labeled family, not
  only the discard counter: each class and outcome is a fixed label value
  that renders as a zero-valued sample rather than disappearing, because a
  zero count is information, not absence.
  - replication reach and backlog — `supkv_replication_peers`,
    `supkv_replication_unacked_records` and
    `supkv_replication_unacked_oldest_age_seconds` (the retransmission
    backlog), plus `supkv_replication_datagrams_retransmitted_total`;
  - replication traffic — `supkv_replication_datagrams_sent_total` (by
    message type). A datagram counts as **sent** only after the socket
    accepted it: a refused write, an unresolvable unicast handle and an
    encode failure count nothing, so the counter measures hand-off, not
    delivery;
  - the four routed core counters — `supkv_replication_listings_dropped_total`
    (targeted listing triggers refused by the destination or the chunk cap,
    retried later), `supkv_replication_sync_streams_dropped_total` (partial
    DataSync streams evicted by the accumulator bounds),
    `supkv_replication_listed_keys_evicted_total` (retained listed keys
    released by whole-peer eviction or expiry) and
    `supkv_replication_notifications_regressed_total` (local-write
    notifications ignored because their version was not greater than the
    tracked one);
  - discards — `supkv_datagrams_discarded_total` (by classification; the
    label set is the complete catalog and every class is published even at
    zero: `bad_crc`, `undecodable`, `bad_proto_version`, `own_broadcast`,
    `duplicate_node_id`, `unbound_listing`, `empty_key`, `key_too_long`,
    `tombstone_with_data`, `bad_checksum`, `bad_version`,
    `bad_sync_header`);
  - conflicts and drift — `supkv_merges_conflicts_total` counts only a
    pending (unconfirmed) local write displaced by a replicated value; an
    overwrite of an already-confirmed record is not a conflict and counts
    nothing. `supkv_clock_drift_warnings_total` counts received records and
    DataSync listing-part maxima whose version leads local wall time by
    more than `--drift-warn` (see the clock-drift runbook);
  - dumps — `supkv_dumps_duration_seconds` is the last dump attempt's
    encode and save, `supkv_dumps_total` the attempts,
    `supkv_dumps_errors_total` the failures, `supkv_dumps_failures_total`
    the failures by class (`disk_full`, `permission`, `other` — `other` is
    the clamped residual, so every scrape carries all three),
    `supkv_dumps_unconfirmed_total` the writes answered before their
    covering dump completed, `supkv_dump_load_total` the startup outcome
    (`prompt`, `main`, `backup`, `unusable`) and
    `supkv_dump_records_discarded_total` the load-time skip class
    (`checksum`, `version`, `value_limit`, `malformed`);
  - process — `process_resident_memory_bytes` is the current resident set size
    in bytes on Linux; development-only darwin builds report the process
    peak instead;
- `/debug/vars` (expvar) and `/debug/pprof/*` — runtime introspection.

Because pprof is an arbitrary-cost endpoint, do not expose the metrics
listener beyond loopback as plaintext. TLS with mutual client
authentication is available through an environment-only certificate triple:
`SUPKV_METRICS_CERT`, `SUPKV_METRICS_KEY` and `SUPKV_METRICS_CLIENT_CA`
(file paths; all three must be set together or none — a partial set aborts
startup with a usage error).

Structured logs go to stderr as JSON by default. `SUPKV_LOG_LEVEL` (debug
through error) and `SUPKV_LOG_FORMAT` (`json` or `text`) tune them.

## Lifecycle logs

Every record below is one structured `slog` record on stderr, JSON by
default; the message is static and every value is a structured attribute.
The levels are the operator contract, pinned by `TestCatalogLevels`.

- **INFO `Starting sup-kv`** — once, before any bind or dump load: the
  build identity (`version`, `commit`, `go_version`), the node identity and
  every effective configuration value. Certificate and key material never
  crosses this boundary: only `metrics_addr` and a `metrics_tls` boolean.
- **INFO `Dump load`** — once per start for a `prompt` or `main` outcome:
  `outcome`, `generation`, `records` and `discarded`.
- **WARN `Dump generation fallback`** — a `backup` load: the same four
  attributes plus `main_path`, `cause` and `preserve_error`.
- **ERROR `No usable dump generation`** — the `unusable` outcome: the same
  four attributes plus one `main_*`/`prev_*` attribute group per candidate
  that actually failed.
- **INFO `Startup gate opened`** — the gate's one opening event: `reason`
  is `sync` (a complete listing arrived) or `backstop` (the startup gate
  elapsed).
- **INFO `Peer appeared`**, **WARN `Peer restarted`**, **WARN `Peer evicted`**
  — the peer table's lifecycle transitions, each with `node_id` and
  `handle`: an appearance and a restart carry the `run_id` (a restart also
  `previous_run_id`), an eviction its `reason` and `last_seen_age`.
- **WARN `Clock drift`** — one per peer per rate-limit interval: `node_id`,
  `handle`, the measured `lead` and the configured `drift_warn` (see the
  clock-drift runbook).
- **ERROR `Dump failed`** — one per failed dump attempt: the failure
  `class` (`disk_full`, `permission`, `other`) and the underlying error as
  `detail`.
- **WARN `Replication adapter`** — one record per adapter WARN, with the
  rendered text in `detail`. Three sources share the message: the core's
  rate-limited adaptation and discard details — a foreign `proto_version`
  (`class bad_proto_version`), a duplicate `node_id`, an apply failure and
  the `unbound_listing` drop; adapter-side send failures (`udprepl: send to
  ...`), one record per refused datagram and **not** rate-limited, so a
  broken broadcast destination produces a steady WARN stream; and
  adapter-side receive errors (`udprepl: receive error ...`), rate-limited
  to one record per second while the read loop keeps running.

## Lifecycle and exit codes

The signal handler is armed **before** dump load and bind, so a `SIGTERM`
(or `SIGINT`) that arrives during startup still takes the graceful path
instead of the default disposition. The graceful path stops serving (health
flips to NOT_SERVING), finishes a final confirmed dump, then exits. A
second signal during the drain is the operator's hurry-up: it forces the
remaining stop calls without skipping the final flush. Exit codes:

- exit code 0 — clean shutdown; the final dump is confirmed on disk;
- exit code 1 — a startup failure (UDP bind, gRPC listen or metrics listen)
  or a `Serve` error. A startup failure never runs a final dump (no
  client-confirmed write exists yet); a `Serve` error still runs the
  bounded final dump before the code is chosen. A systemd unit with
  `Restart=on-failure` reacts to this;
- exit code 2 — malformed configuration: a bad flag value or a malformed
  `SUPKV_*` environment value, naming the parameter;
- exit code 3 — the final dump failed, including on the `Serve`-error path.
  The failed final dump is the operator's data-loss signal and takes
  precedence over the `Serve` error's 1.

A dump path the node cannot read or write is **not** a startup failure: the
load reports its outcome (`prompt`, `main`, `backup` or `unusable`) through
the `Dump load`, `Dump generation fallback` or `No usable dump generation`
record, and the node starts. A persistently failing dump surfaces on writes
as `Unavailable` (see below), and a final dump that cannot be written exits
with code 3 — never 1.

`--help` lists the five env-only knobs that have no flag: `SUPKV_LOG_LEVEL`,
`SUPKV_LOG_FORMAT`, `SUPKV_METRICS_CERT`, `SUPKV_METRICS_KEY` and
`SUPKV_METRICS_CLIENT_CA`.

A failed dump is retried, not fatal. Each failed attempt logs exactly one
ERROR carrying the failure class and the underlying error, and publishes
its class counter; the node retries on a ladder that starts at 100 ms and
doubles after every further failure up to a 5 s cap, and the next success
resets it. A snapshot request's forced flush does not bypass that ladder:
while a rung is pending it answers the last dump failure immediately instead
of driving an extra dump attempt and an extra ERROR line at the client's
request rate. Health stays `SERVING` under a persistent dump failure: reads
keep serving and the node keeps retrying, because the failure is a
durability risk, not a serving failure. It surfaces on writes instead — a
write whose covering dump does not complete within the confirmation timeout
is answered `Unavailable` (`write not confirmed before the confirmation
timeout`) and counted in `supkv_dumps_unconfirmed_total`. Shutdown is
bounded: the final flush waits at most the shutdown grace (a stuck fsync
answers `Unconfirmed`, and a failed final dump exits with code 3), and
a dumper goroutine still not stopped at process exit is abandoned with one
WARN rather than hanging the shutdown.

A `SIGKILL` (or host crash) skips the final dump, but confirmed writes are
still safe: every client-confirmed write was already contained in a
completed dump on disk before the client saw success. On restart the node
loads its dump file (falling back to the previous generation, `.prev`, on a
corrupt one), and any gap is healed by replication from its peers.

## Failure behaviour

The Spec's failure-behavior table, as built: one row per S-FAIL-01 through
S-FAIL-18 statement, in the registry's order (the statements themselves live
in [docs/SPEC-DEVIATIONS.md](SPEC-DEVIATIONS.md); this table is the operator
view — what you see, what the node does, what recovers it, what bounds it,
and the metric and log evidence that exists in the binary). A row without a
dedicated metric says so honestly — `none — log only` — rather than naming a
family the exporter does not render.

| ID | Symptom | Behavior | Recovery | Bound | Metric | Log line |
|---|---|---|---|---|---|---|
| S-FAIL-01 | A record a peer should hold is missing or late after one lost datagram | The sender's retransmit ladder resends the record with a growing pause from `ack_timeout` (200 ms) up to `ack_timeout_max` (5 s); the receiving node applies it and its Ack removes the pending entry | Automatic — no operator action | A loss episode whose delivery opportunity falls inside the ladder's window (attempts at 200/600/1400/3000 ms, all inside the 5 s default ceiling); sustained loss past the window is the partition-heal bound's territory (S-FAIL-04) | `supkv_replication_unacked_records`, `supkv_replication_unacked_oldest_age_seconds`, `supkv_replication_datagrams_retransmitted_total` | — |
| S-FAIL-02 | An Ack was lost while the record itself arrived | The sender retransmits; the receiver's equal-version branch Acks again without re-applying — no extra dump, no extra broadcast, no conflict | Automatic — no operator action | The same ladder window as S-FAIL-01 | `supkv_replication_datagrams_retransmitted_total` | — |
| S-FAIL-03 | Duplicates, reordering or the node's own broadcast echo arrive on the receive path | The own echo is dropped before decoding; an equal-version duplicate is a no-op; reordering resolves by the record-version order; a stale payload draws exactly one unicast reply carrying the held record and no Ack | Automatic — absorption is by construction, convergence is unaffected | None needed — absorption does not depend on timing, and the stale reply is one datagram | `supkv_datagrams_discarded_total{class=own_broadcast}` covers the echo; duplicates and reordering are absorbed silently — none — log only | — |
| S-FAIL-04 | A peer goes silent: `supkv_replication_peers` drops, both sides keep serving writes | The silent peer is evicted at `peer_timeout`; after the heal its Hellos re-discover it, one targeted listing is enqueued for it, and retransmission plus the sync exchange reconcile both sides | Automatic — no operator action | At most `2 × sync_interval` + response time after the heal (about a minute at the 30 s default) | `supkv_replication_peers` | WARN `Peer evicted`, INFO `Peer appeared` |
| S-FAIL-05 | A long partition — writes accumulate on both sides for hours or days | Both sides serve; on the heal the listings reconcile and every replica converges to the same deterministic winner per key, picked by the record version | Automatic — no operator action | The same partition-heal bound; the winner is identical on every replica regardless of which one you ask | `supkv_merges_conflicts_total` (a displaced pending local write is the counted conflict) | — |
| S-FAIL-06 | One-way connectivity — a node sends but hears nothing | The deaf node's retransmissions climb to the ladder cap and stay there; the send pacers and the egress queue cap bound the send rate, so no message avalanche builds | Automatic on connectivity — the heal is the partition bound | Bounded send rate (the pacers, `sync_response_rate`) and a bounded retransmit ladder; heal within the partition-heal bound | `supkv_replication_unacked_oldest_age_seconds` (climbs on the deaf side), `supkv_replication_send_queue` | — |
| S-FAIL-07 | A node restarts (a new run_id on a known node) | Its dump loads and the first Hello reaches the peers; each peer immediately targets one full listing at it; past `peer_timeout` the peer row's expiry hands the wait to the periodic rounds | Automatic — no operator action | Convergence starts from the restarted node's first Hello; the periodic sync is the backstop | `supkv_dump_load_total{outcome=main}`, `supkv_replication_peers` | WARN `Peer restarted`, INFO `Peer appeared`, INFO `Dump load` |
| S-FAIL-08 | A node was down longer than `peer_timeout` | The peer table evicts it; the survivor's periodic DataSync rounds carry records and tombstones to the returning node; its recovery traffic is rate-limited at `sync_response_rate` | Automatic — no operator action | The partition-heal bound from the heal instant | `supkv_replication_peers` | WARN `Peer evicted`, INFO `Peer appeared` |
| S-FAIL-09 | An empty node joins — fresh disk, no dump to load | It loads nothing (outcome prompt), stays gated (reads and writes answer `Unavailable`) until a complete listing arrives or `startup_gate_empty` elapses, then pulls the full state through rate-limited DataRequest and DataReplication answers | Automatic — no operator action | The documented paced-recovery bound holds at the documented scale (the registry row's evidence names the stand contract and the nightly job) | `supkv_dump_load_total{outcome=prompt}` | INFO `Dump load`, INFO `Startup gate opened` |
| S-FAIL-10 | Both nodes down at once — host crash or `SIGKILL` | Each restarts from its own dump; every client-confirmed write was already contained in a completed dump before the client saw success | Automatic; the window each node missed while down heals by replication | Confirmed writes are never lost; only unconfirmed last-window writes may be absent locally until replication heals them | `supkv_dump_load_total{outcome=main}` | INFO `Dump load` |
| S-FAIL-11 | A dump generation is corrupt — truncated, torn or undecodable | The load classifies and skips bad records by class, falls back to `.prev`, and starts empty when no generation decodes; the rejected generation is preserved as `.corrupt-<UTC>` and never rotated | Automatic — the node starts and heals from peers; the evidence stays on disk for the operator (docs/FORMATS.md documents the census the load reports) | The chain is main, then `.prev`, then an empty start — the node always starts | `supkv_dump_load_total{outcome=backup}`, `supkv_dump_load_total{outcome=unusable}`, `supkv_dump_records_discarded_total` | WARN `Dump generation fallback`, ERROR `No usable dump generation` |
| S-FAIL-12 | The wall clock jumps on a running node — an NTP step or a VM pause | Record versions stay monotonic: the clock jumps forward with the wall and bumps its logical counter when it does not; a received version that now leads local time raises the drift signal | None needed — correctness never depends on wall-clock synchronization | No bound applies; the version order is unchanged by the jump | none — log only; `supkv_clock_drift_warnings_total` carries the received-lead symptom (see the clock-drift runbook) | WARN `Clock drift` |
| S-FAIL-13 | The wall clock stepped back between restarts | The clock is seeded from the loaded dump's highest logical time and counter, so the restarted node's versions still dominate its own pre-restart ones | None needed | No bound applies — the seed dominates any earlier version the node issued | none — log only | INFO `Dump load` |
| S-FAIL-14 | One node's clock runs far ahead — for example a 24 h skew | The peers absorb the far-ahead stamps through the normal clock update rule; the receivers count and warn; the far-ahead node's writes win every concurrent conflict until the peers catch up | Fix the clock source (NTP or PTP) on the offending host — see the clock-drift runbook | Correctness never depends on wall-clock synchronization; the skew decides which simultaneous write wins, deterministically and identically on all replicas | `supkv_clock_drift_warnings_total` | WARN `Clock drift` |
| S-FAIL-15 | A burst of about 1000 writes/s on distinct keys | Writes confirm at the dump cadence (the debounce window under its `dump_max_wait` cap) and one datagram per write is emitted through the pacers; the unacked set drains as the peers Ack | Automatic — no operator action | At least 990 writes/s achieved at the documented scale (1000 writes/s, 1 KiB values, two healthy nodes, default buffers) and the unacked set empty within 5 s of the send window | `supkv_dumps_total`, `supkv_dumps_duration_seconds`, `supkv_replication_unacked_records`, `supkv_replication_datagrams_paced_total` | — |
| S-FAIL-16 | Two processes share one `--node-id` | The peers classify the impostor's datagrams `duplicate_node_id` and drop them; the first Hello logs the configuration error; no record data is exchanged with the impostor | Operator — fix the duplicated `--node-id` and restart one of the two processes | Neither node converges with the other until the configuration is fixed | `supkv_datagrams_discarded_total{class=duplicate_node_id}` | WARN `Replication adapter` |
| S-FAIL-17 | Nodes on the segment run different protocol versions | Datagrams with a foreign `proto_version` are discarded at the adapter's pre-CRC verdict and again on the decoded twin; each node keeps serving, persisting and retransmitting independently | Operator — upgrade the lagging node; convergence resumes automatically once the versions match | Each node serves independently for the whole mismatch window | `supkv_datagrams_discarded_total{class=bad_proto_version}` | WARN `Replication adapter` |
| S-FAIL-18 | A foreign datagram on the segment | CRC32 detects accidental corruption only — a forged datagram with a valid CRC and a valid record cannot be detected, under the accepted trust model | None in v1 — the trust model accepts this (see the trust model above) | The trust boundary is the segment itself, not the datagram | none — log only; `supkv_datagrams_discarded_total{class=bad_crc}` counts accidental corruption, never forgery | — |

## Runbook: dump failure and full disk

**Symptom.** One ERROR `Dump failed` record per failed dump attempt, carrying
the failure class — `disk_full` (the disk is out of space), `permission` (the
dump path rejects the write) or `other` (the clamped residual, for example a
missing directory or an I/O error) — and
`supkv_dumps_failures_total{class=disk_full}` rises in step. Writes whose
covering dump does not complete within the confirmation timeout are answered
`Unavailable` and counted in `supkv_dumps_unconfirmed_total`. Health stays
`SERVING`: reads keep serving while the node retries.

**Cause.** The dump path is not writable. The failure classes map to the
operating system's answers: out of space (`ENOSPC`), permission denied
(`EACCES`), and everything else clamped to `other`.

**Action.** Free space or fix the permissions. The node is already retrying
on the decaying ladder — the first failure retries after 100 ms, every
further failure doubles the wait up to the 5 s cap, and the next success
resets it — so no restart is needed, and none would help. A final dump that
still fails at shutdown exits with code 3: that is the data-loss signal, and
the failed attempt's ERROR records name the class.

## Runbook: missing peer

**Symptom.** `supkv_replication_peers` drops — one fewer, or to zero on a
lone node — with one WARN `Peer evicted` naming the `node_id`, the reason
(`peer_timeout`) and the `last_seen_age`; the unacked backlog grows
(`supkv_replication_unacked_records`,
`supkv_replication_unacked_oldest_age_seconds`).

**Cause.** The peer has not been heard from for `peer_timeout` (default 5s):
it is down, partitioned, or its datagrams stopped arriving.

**Action.** Check the peer's process and host first; nothing is needed on
this node. When the peer returns, its first Hello re-discovers it (INFO
`Peer appeared`) and one targeted listing reconciles the pair within the
partition-heal bound.

## Runbook: growing unconfirmed set

**Symptom.** `supkv_replication_unacked_records` and
`supkv_replication_unacked_oldest_age_seconds` climb; writes with
`wait_for_replication` answer `acked: false`; retransmissions accumulate in
`supkv_replication_datagrams_retransmitted_total`.

**Cause.** Peers are not acknowledging: lost datagrams, an evicted peer, a
segment problem, or a peer discarding records (check
`supkv_datagrams_discarded_total`).

**Action.** Check `supkv_replication_peers` and the discard classes first.
The pending index is one entry per key, so the set is bounded by the key
count and cannot grow without bound; entries leave on Ack or supersession.
Sustained loss past the retransmit ladder's window is the partition-heal
bound's territory.

## Runbook: write conflict

**Symptom.** `supkv_merges_conflicts_total` rises.

**Cause.** A replicated value displaced a pending (unconfirmed) local write
under the same key — concurrent writers, normally during or right after a
partition or a burst. An overwrite of an already-confirmed record counts
nothing.

**Action.** None. The merge is deterministic last-write-wins: every replica
converges to the same winner. The displaced write's `wait_for_replication`
waiter resolves with the honest false (the Superseded outcome), so the
client may retry; the counter is a monitor, not an error.

## Runbook: incompatible versions

**Symptom.** `supkv_datagrams_discarded_total{class=bad_proto_version}` rises
on one or both nodes, with one rate-limited WARN `Replication adapter` per
peer per interval. Peers on the other version never discover each other —
the discarded datagrams include every Hello — so `supkv_replication_peers`
shows the split too.

**Cause.** Mixed sup-kv binary versions on the segment, typically a rolling
upgrade that half-landed.

**Action.** Upgrade the lagging node. Until the versions match, each node
serves, persists and retransmits independently; convergence resumes
automatically once the versions agree.

## Runbook: corrupted dump

**Symptom.** At startup: one WARN `Dump generation fallback` (a `.prev`
load) or one ERROR `No usable dump generation` (nothing decoded);
`supkv_dump_load_total{outcome=backup}` or
`supkv_dump_load_total{outcome=unusable}` carries the outcome sample, and
`supkv_dump_records_discarded_total` counts the skipped records by class.

**Cause.** A truncated or damaged generation — the `record_count` census
catches a truncation that lands inside a record line.

**Action.** Inspect the preserved evidence: the rejected generation sits at
`<path>.corrupt-<UTC>` beside the dump, never rotated and never
auto-deleted, and a non-empty `preserve_error` means the rename itself
failed — move that file out of the dump path before the next save. The node
starts from `.prev` or empty and heals from its peers;
[docs/FORMATS.md](FORMATS.md) documents the format and the census.

## Runbook: duplicate node_id

**Symptom.** `supkv_datagrams_discarded_total{class=duplicate_node_id}` rises
on both nodes, with one rate-limited WARN `Replication adapter` per interval
naming the configuration error; neither node converges with the other.

**Cause.** Two processes share one `--node-id` — a deployment
misconfiguration, not a data hazard: the peers treat the impostor's
datagrams as a configuration error and drop them.

**Action.** Fix the duplicated `--node-id` and restart one of the two
processes. The WARN carries the `node_id` and the handle only — no record
data is ever sent to the impostor.

## Metrics reference

The complete sup-kv metric surface as one typed table — every family the
exporter renders at zero state, the `supkv_` families and the standard
runtime families alike — with the meanings mirroring the exporter's HELP
text and recommended alerts only where the binary can observe the condition;
informational families state `no alert recommended`. The label values below
are the complete catalogs: every class and outcome renders as a zero-valued
sample even when unseen, because a zero count is information, not absence.

| Family | Type | Labels | Meaning | Recommended alert |
|---|---|---|---|---|
| `supkv_replication_peers` | gauge | — | Currently known replication peers. | Alert when it stays 0 for more than 2 × peer_timeout — the node hears no peer (the missing-peer runbook). |
| `supkv_replication_unacked_records` | gauge | — | Replication records awaiting acknowledgement. | no alert recommended — monitor it with the age gauge below. |
| `supkv_replication_unacked_oldest_age_seconds` | gauge | — | Age of the oldest unacknowledged replication record in seconds. | Alert above the ack_timeout_max ceiling (5 s default) — a delivery opportunity inside the ladder window should have recovered by then. |
| `supkv_replication_datagrams_retransmitted_total` | counter | — | Replication datagram retransmissions. | no alert recommended. |
| `supkv_merges_conflicts_total` | counter | — | Pending (unacked) local writes displaced by a replicated value. | no alert recommended — expected under concurrent writers; a rising rate is the write-conflict runbook's context. |
| `supkv_clock_drift_warnings_total` | counter | — | Received records and listing-part versions that lead local wall time by more than drift-warn. | Alert on any increase — a clock source is drifting (the clock-drift runbook). |
| `supkv_replication_datagrams_paced_total` | counter | — | Replication datagrams emitted by the listing, record and egress pacers. | no alert recommended. |
| `supkv_replication_send_queue` | gauge | — | Replication datagrams waiting to emit across every core queue. | no alert recommended — the queue is capped; the egress drop counter below is the cap's evidence. |
| `supkv_replication_egress_dropped_total` | counter | — | Egress responses dropped by the queue cap. | no alert recommended — sustained growth means a peer cannot drain its answers. |
| `supkv_replication_read_errors_total` | counter | — | Receive-path errors the read loop survived. | Alert on sustained growth — the node is losing inbound datagrams at the socket. |
| `supkv_replication_listings_dropped_total` | counter | — | Targeted listing triggers refused by the destination or chunk cap; retried later. | no alert recommended. |
| `supkv_replication_sync_streams_dropped_total` | counter | — | Partial DataSync streams evicted by the accumulator bounds. | no alert recommended. |
| `supkv_replication_listed_keys_evicted_total` | counter | — | Retained listed keys released by whole-peer eviction or expiry. | no alert recommended. |
| `supkv_replication_notifications_regressed_total` | counter | — | Local-write notifications ignored because their version was not greater than the tracked version. | no alert recommended. |
| `supkv_replication_datagrams_sent_total` | counter | `type`: `hello`, `data`, `ack`, `data_request`, `sync` | Outbound datagrams successfully handed to the socket, by message type. | no alert recommended — sent measures hand-off, not delivery. |
| `supkv_datagrams_discarded_total` | counter | `class`: `bad_crc`, `undecodable`, `bad_proto_version`, `own_broadcast`, `duplicate_node_id`, `unbound_listing`, `empty_key`, `key_too_long`, `tombstone_with_data`, `bad_checksum`, `bad_version`, `bad_sync_header` | Inbound replication datagrams discarded, by classification. | Alert on duplicate_node_id (any increase — a configuration error) and bad_proto_version (sustained — a version mismatch); the remaining classes rate alerts by the runbooks' evidence. |
| `supkv_dumps_duration_seconds` | gauge | — | Duration of the last dump attempt's encode and save in seconds. | no alert recommended. |
| `supkv_dumps_total` | counter | — | Dump attempts. | no alert recommended. |
| `supkv_dumps_errors_total` | counter | — | Failed dumps. | Alert on any increase — the class split is the next row. |
| `supkv_dumps_failures_total` | counter | `class`: `disk_full`, `permission`, `other` | Failed dump attempts, by failure class. | Alert on any increase — the dump-failure runbook names the per-class actions. |
| `supkv_dumps_unconfirmed_total` | counter | — | Writes answered before their covering dump completed. | Alert on any increase — writes are being answered Unavailable. |
| `supkv_dump_load_total` | counter | `outcome`: `prompt`, `main`, `backup`, `unusable` | Startup dump loads by outcome. | Alert on the unusable sample — a data-losing start that healed from peers (the corrupted-dump runbook). |
| `supkv_dump_records_discarded_total` | counter | `class`: `checksum`, `version`, `value_limit`, `malformed` | Dumped records discarded at load, by classification. | Alert on checksum, version or malformed increases — discarded evidence names a damaged generation; value_limit is the configured limit's own skip. |
| `go_goroutines` | gauge | — | Number of goroutines that currently exist. | no alert recommended. |
| `go_memstats_alloc_bytes` | gauge | — | Number of bytes allocated and still in use. | no alert recommended. |
| `go_memstats_sys_bytes` | gauge | — | Number of bytes obtained from system. | no alert recommended. |
| `process_cpu_seconds_total` | counter | — | Total user and system CPU time spent in seconds. | no alert recommended. |
| `process_resident_memory_bytes` | gauge | — | Current resident set size in bytes (Linux); development-only darwin builds report the process peak. | no alert recommended — the memory model and its measured points live in docs/PERFORMANCE.md. |

## Log-message catalog

Every record below is one message the running binary emits, at the
documented level — the full level matrix of the operator contract. The
`## Lifecycle logs` section above documents the attribute contract of the
startup and load records; this catalog is the complete literal surface, and
the renderers it shares with the lifecycle entries are pinned by
`TestCatalogLevels`. The redirected log-package records render through the
same slog handler at INFO; the usage path is plain stderr text, not a slog
record.

| Level | Message | Where it appears |
|---|---|---|
| INFO | `Starting sup-kv` | once at startup, before any bind or load — the build identity and every effective knob |
| INFO | `Dump load` | one record per start for the prompt or main load outcome |
| WARN | `Dump generation fallback` | the backup load outcome — the main generation's path, cause and preserve_error |
| ERROR | `No usable dump generation` | the unusable load outcome — one attribute group per failed candidate |
| INFO | `Orphan temp sweep removed files` | the startup sweep removed one or more orphan temp files |
| WARN | `Orphan temp sweep failed` | the startup sweep could not run |
| WARN | `Dumped records skipped by the configured value limit` | the load skipped records over the configured value limit |
| INFO | `sup-kv: discarded %d undecodable dumped records at startup` | the redirected log-package record for the remaining undecodable count |
| INFO | `Socket buffers` | the effective kernel socket buffer sizes read back after bind |
| INFO | `Startup gate opened` | the gate opened — reason sync or backstop |
| INFO | `Peer appeared` | a peer entered the table |
| WARN | `Peer restarted` | a peer came back with a new run_id |
| WARN | `Peer evicted` | a peer exceeded peer_timeout |
| WARN | `Clock drift` | a received version leads local wall time by more than drift-warn — rate-limited per peer |
| WARN | `Replication adapter` | the core's rate-limited discard and configuration details, adapter send failures (not rate-limited) and receive errors (one per second) |
| WARN | `Bind address is not a wildcard: broadcast datagrams are not received` | a non-wildcard bind address at startup |
| DEBUG | `Bind address is not host:port` | a bind address the host:port split rejected |
| ERROR | `Dump failed` | one record per failed dump attempt — the failure class and the error detail |
| ERROR | `Startup failed` | a wiring stage failed — the stage and the detail |
| ERROR | `Serve failed` | the gRPC server returned before any signal |
| ERROR | `Final dump failed` | the bounded shutdown flush failed — exit code 3 |
| ERROR | `Metrics endpoint` | the metrics listener's server returned an error |
| WARN | `Engine did not stop within the shutdown grace` | the dumper goroutine was abandoned at process exit |
| INFO | `sup-kv: metrics listener shutdown: %v` | the redirected log-package record for a metrics shutdown error |
| INFO | `sup-kv: second shutdown signal — forcing stop` | the hurry-up signal during the drain |
| INFO | `sup-kv: graceful stop exceeded %v — forcing stop` | the grace deadline forced the stop |
| INFO | `sup-kv: engine Run returned %v` | the redirected log-package record for a non-cancellation engine exit |
| stderr | `sup-kv: %s` | the usage path — the configuration error followed by the full usage, exit code 2 |
