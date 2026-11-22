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

- values are at most 60 KiB — 61440 bytes, the default of
  `--value-max-bytes` (environment twin `SUPKV_VALUE_MAX_BYTES`). A write
  with a larger value is rejected with `InvalidArgument`;
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
- **Values are at most `--value-max-bytes`** (environment twin
  `SUPKV_VALUE_MAX_BYTES`, default 61 440 bytes).
- **The combined record must fit 65 507 bytes** — the maximum IPv4 UDP
  payload. With a maximal 1 024-byte key the largest fitting value is
  64 408 bytes: 1 024 + 64 408 plus the envelope overhead is exactly the
  ceiling. At startup the node measures the worst case through the real
  codec; a `--value-max-bytes` that cannot fit exits with code 2 and an
  error naming the flag, the key limit and the computed size. The default
  61 440 leaves roughly 3 KiB of headroom.
- **`--max-clock-lead`** (environment twin `SUPKV_MAX_CLOCK_LEAD`, default
  60s) bounds how far ahead of local time a received record version may
  be. Keep the same value on every node: a node with a smaller bound
  discards versions its peers still consider valid.

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
peers stay flat. A warning is counted when an absorbed remote timestamp sits
beyond the `--drift-warn` threshold ahead of the node's own clock (default
60s; environment twin `SUPKV_DRIFT_WARN`).

**Cause.** The node's wall clock runs behind (or ahead of) its peers'.

**Action.** Fix the clock source — NTP or PTP on the host. No sup-kv-side
repair exists, and none is needed: correctness never depends on wall-clock
synchronization. Clock skew only affects which simultaneous write wins, and
it does so deterministically and identically on all replicas. A node whose
clock runs far ahead will see its writes dominate (win every concurrent
conflict) until the peers' clocks catch up; once the clock is corrected, new
writes resume normal ordering. This behavior is proven by the
failure-scenario suite (simulated far-ahead nodes converge
deterministically).

Do not restart the cluster to "fix" drift: it changes nothing. Record
versions are hybrid logical clocks, not raw wall-clock readings, so the same
version order applies after any restart.

## Monitoring

The metrics/debug endpoint is opt-in. Pass `--metrics-addr` (environment
twin `SUPKV_METRICS_ADDR`); when it is unset, no listener exists at all.
The endpoint serves:

- `/metrics` — Prometheus text format from an in-repo exporter, the
  `supkv_*` families:
  `supkv_replication_unacked_records` and
  `supkv_replication_unacked_oldest_age_seconds` (retransmission backlog),
  `supkv_replication_peers`, `supkv_replication_datagrams_sent_total` (by
  message type), `supkv_replication_datagrams_retransmitted_total`,
  `supkv_datagrams_discarded_total` (by classification: `bad_crc`,
  `undecodable`, `bad_proto_version`, `own_broadcast`,
  `duplicate_node_id`, `bad_sync_header`), `supkv_merges_conflicts_total`,
  `supkv_clock_drift_warnings_total`, `supkv_dumps_duration_seconds`,
  `supkv_dumps_total`, `supkv_dumps_errors_total` — plus the standard
  `go_*`/`process_*` runtime families;
- `/debug/vars` (expvar) and `/debug/pprof/*` — runtime introspection.

Because pprof is an arbitrary-cost endpoint, do not expose the metrics
listener beyond loopback as plaintext. TLS with mutual client
authentication is available through an environment-only certificate triple:
`SUPKV_METRICS_CERT`, `SUPKV_METRICS_KEY` and `SUPKV_METRICS_CLIENT_CA`
(file paths; all three must be set together or none — a partial set aborts
startup with a usage error).

Structured logs go to stderr as JSON by default. `SUPKV_LOG_LEVEL` (debug
through error) and `SUPKV_LOG_FORMAT` (`json` or `text`) tune them.

## Lifecycle and exit codes

`SIGTERM` (or `SIGINT`) triggers the graceful path: the node stops serving
(health flips to NOT_SERVING), finishes a final confirmed dump, then exits.
Exit codes:

- exit code 0 — clean shutdown; the final dump is confirmed on disk;
- exit code 1 — the final dump failed (a systemd unit with
  `Restart=on-failure` reacts to this);
- exit code 2 — malformed configuration: a bad flag value or a malformed
  `SUPKV_*` environment value.

A `SIGKILL` (or host crash) skips the final dump, but confirmed writes are
still safe: every client-confirmed write was already contained in a
completed dump on disk before the client saw success. On restart the node
loads its dump file (falling back to the previous generation, `.prev`, on a
corrupt one), and any gap is healed by replication from its peers.
