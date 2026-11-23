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

## What you are operating

sup-kv is a leaderless, always-writable (AP) replicated in-memory store.
Every node accepts reads and writes locally and independently; replicas
exchange changes over UDP broadcast and converge to identical state — the
pointwise supremum of all writes under the record-version order (a
Last-Write-Wins-Element-Map CRDT). There is no leader, no quorum and no
coordination: losing any single node never blocks reads or writes anywhere.

Convergence bounds (proven in simulation under datagram loss, duplication,
reordering and partitions):

- normal operation: a write is visible on every peer within one round trip;
- datagram loss: retransmission with exponential backoff bounds recovery at
  `ack_timeout_max` (default 5s);
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
parameter, and never falls back to a default silently. Zero or negative
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

A dump cycle's peak live heap is one state copy plus one record line. The
engine takes exactly one `Records()` deep copy — the map plus every
record's cloned value — and streams it through the adapter's 64 KiB
buffered writer one encoded record line at a time. The only other live
buffer is the record line being encoded, whose base64 expansion is at most
about 1.34 times the configured value limit, so the encoded dump is never
materialized as a second buffer. The kernel page cache that holds the
written file is not process heap and is excluded from this model. Peak
live therefore stays near one state size, not the two-plus of an
encode-into-bytes design.

One residual pause remains, and it is stated rather than claimed away:
`Records()` clones every value while holding the store's read lock, so
writers are paused for one state-sized copy at snapshot time. Eliminating
that pause needs a copy-on-write view of the store and is residual work,
not a property this model claims.

Peak live and allocation churn are different numbers. `B/op` from the
benchmark measures everything allocated over a cycle — the deep copy, then
one line's worth per record, each line garbage as soon as it is written —
and so reads higher (roughly 1.4 times the state size in total
allocations) than the peak-live figure above. The bound this runbook states
is peak live: one copy, plus one record line, plus the 64 KiB writer.

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
proven by the failure-scenario suite (simulated far-ahead nodes converge
deterministically).

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
- **WARN `Replication adapter`** — the core's rate-limited adaptation and
  discard details as `detail`: a foreign `proto_version`
  (`class bad_proto_version`), a duplicate `node_id`, an apply failure and
  the `unbound_listing` drop.

## Lifecycle and exit codes

The signal handler is armed **before** dump load and bind, so a `SIGTERM`
(or `SIGINT`) that arrives during startup still takes the graceful path
instead of the default disposition. The graceful path stops serving (health
flips to NOT_SERVING), finishes a final confirmed dump, then exits. A
second signal during the drain is the operator's hurry-up: it forces the
remaining stop calls without skipping the final flush. Exit codes:

- exit code 0 — clean shutdown; the final dump is confirmed on disk;
- exit code 1 — a startup failure (bind, listen, metrics, dump open) or a
  `Serve` error. A startup failure never runs a final dump (no
  client-confirmed write exists yet); a `Serve` error still runs the
  bounded final dump before the code is chosen. A systemd unit with
  `Restart=on-failure` reacts to this;
- exit code 2 — malformed configuration: a bad flag value or a malformed
  `SUPKV_*` environment value, naming the parameter;
- exit code 3 — the final dump failed, including on the `Serve`-error path.
  The failed final dump is the operator's data-loss signal and takes
  precedence over the `Serve` error's 1.

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
