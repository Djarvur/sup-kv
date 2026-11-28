# sup-kv configuration

The as-built configuration reference, generated from the binary itself: the
flag table below mirrors the built binary's `--help` output byte-for-byte —
the registered surface, in registration order, with the printed defaults —
and the iterator in cmd/sup-kv/configuration_test.go fails when this table
drifts in either direction. A flag added, renamed, re-defaulted or removed
in internal/config without a doc edit fails `make test`, and so does a
documented flag the binary does not register. Every startup range and
relation the binary enforces is catalogued under
[Validation relations](#validation-relations); what the envelopes mean for
operators and why they exist is
[docs/OPERATIONS.md](OPERATIONS.md), which this reference links rather than
restates.

Every knob resolves in the order **flag > env > default**: an explicit flag
beats its `SUPKV_` environment twin, and the twin beats the printed default.
The precedence is proven on the real binary in
[docs/OPERATIONS.md](OPERATIONS.md#configuration-validation); an unset or
empty env value counts as absent.

## Flags

One row per registered flag, in the order `--help` lists it (defaults
byte-copied from the `--help` capture, durations in their printed unit):

| Flag | Env twin | Default | Units | Meaning |
|---|---|---|---|---|
| --ack-timeout | SUPKV_ACK_TIMEOUT | 200ms | duration | retransmit ladder base — the first wait before a retransmission, doubling with every further failure up to the ceiling |
| --ack-timeout-max | SUPKV_ACK_TIMEOUT_MAX | 5s | duration | retransmit ladder ceiling — the convergence bound under datagram loss |
| --bind-addr | SUPKV_BIND_ADDR | required — no default | host:port | local UDP bind address, e.g. 0.0.0.0:7900 |
| --broadcast-addr | SUPKV_BROADCAST_ADDR | required — no default | host:port | segment broadcast address, e.g. 127.255.255.255:7900 |
| --drift-warn | SUPKV_DRIFT_WARN | 1m0s | duration | clock drift warning threshold — a received version leading local wall time by more than this warns (see the clock drift runbook in docs/OPERATIONS.md) |
| --dump-debounce | SUPKV_DUMP_DEBOUNCE | 50ms | duration | dump quiet window — how long a burst of writes may keep postponing the dump |
| --dump-max-wait | SUPKV_DUMP_MAX_WAIT | 1s | duration | dump start bound from the oldest unconfirmed write — a dump starts no later than oldest + this, whatever the write cadence |
| --dump-path | SUPKV_DUMP_PATH | required — no default | path | dump file path — the human-readable JSON generation the node loads and confirms writes against |
| --grpc-addr | SUPKV_GRPC_ADDR | 127.0.0.1:7901 | host:port | gRPC listen address, distinct from the UDP bind — the default binds loopback only |
| --hello-interval | SUPKV_HELLO_INTERVAL | 1s | duration | discovery broadcast period |
| --max-clock-lead | SUPKV_MAX_CLOCK_LEAD | 2m0s | duration | reject received versions beyond now + this bound — keep it equal on every node (see Must match on all nodes) |
| --metrics-addr | SUPKV_METRICS_ADDR | none | host:port | optional metrics/debug listen address — no default: the metrics listener stays off unless the flag (or its env twin) is set |
| --node-id | SUPKV_NODE_ID | required — no default | integer | unique 16-bit node ID (0..65535), required |
| --peer-timeout | SUPKV_PEER_TIMEOUT | 5s | duration | peer eviction silence bound |
| --retransmit-interval | SUPKV_RETRANSMIT_INTERVAL | 100ms | duration | retransmit scan period — at least 1ms, a nanosecond period is a busy loop |
| --startup-gate | SUPKV_STARTUP_GATE | 1s | duration | gate backstop: writable this long after start even without a sync round |
| --startup-gate-empty | SUPKV_STARTUP_GATE_EMPTY | 1m0s | duration | empty-disk gate backstop: a node with no loaded records waits this long for a peer listing |
| --sync-chunk-bytes | SUPKV_SYNC_CHUNK_BYTES | 1400 | bytes | DataSync chunk byte budget |
| --sync-interval | SUPKV_SYNC_INTERVAL | 30s | duration | proactive sync listing period — half of the partition-heal bound |
| --sync-response-rate | SUPKV_SYNC_RESPONSE_RATE | 500 | records/s | sync responses per second cap |
| --value-max-bytes | SUPKV_VALUE_MAX_BYTES | 61440 | bytes | maximum value size in bytes — the single value limit, storage.ValueMaxBytes |
| --version | none | none | — | prints version, commit and Go runtime version, then exit — the build identity, not a runtime knob |
| --wait-timeout | SUPKV_WAIT_TIMEOUT | 1s | duration | default wait_for_replication bound |

The three address knobs carry the required `host:port` shape and the two
path-like required knobs (`--dump-path`) name files, not directories. The
defaults the help does not print are printed by nothing: the four required
knobs (`--node-id`, `--bind-addr`, `--broadcast-addr`, `--dump-path`) must
be set per node, and `--metrics-addr` is the one deliberate opt-out — an
empty value means no listener, which only the metrics endpoint admits.

## Environment-only (no flag)

Five knobs have no flag twin — they configure the process's own output and
the metrics endpoint's TLS, not its service behavior. `--help` lists them
in its footer block; the structured knobs resolve like every other twin
(env over default):

- `SUPKV_LOG_LEVEL` — debug, info, warn or error, case-insensitively
  (default info);
- `SUPKV_LOG_FORMAT` — json or text (default json; JSON records on stderr);
- `SUPKV_METRICS_CERT` — server certificate path for the metrics/debug
  endpoint;
- `SUPKV_METRICS_KEY` — server key path for the metrics/debug endpoint;
- `SUPKV_METRICS_CLIENT_CA` — client CA path for the metrics/debug
  endpoint.

The certificate triple is all-or-nothing: any of the three set without the
other two is a startup error, and with none set the metrics listener (when
`--metrics-addr` is set) serves plaintext on the address given — keep it on
loopback.

## Validation relations

One validation point owns every startup range and relation:
internal/config/validate.go's `Validate` catalog. A violation is a
configuration error — the node exits with code 2, names the offending
parameter, and never falls back to a default silently; the failures of one
pass are joined, so a single restart shows every bad knob. The catalog,
rule by rule:

| Catalog rule | Relation in operator terms | Violation |
|---|---|---|
| `required-knobs` | `--node-id`, `--bind-addr`, `--broadcast-addr` and `--dump-path` must be set — node-id zero and empty addresses are "missing", not defaults | exit code 2 naming the missing knobs |
| `node-id-range` | `--node-id` stays inside the wire's 16-bit identity space (0..65535) | exit code 2 |
| `positive-values` | every duration knob plus `--value-max-bytes`, `--sync-chunk-bytes` and `--sync-response-rate` must be positive — "value-max-bytes must be positive" is the sentinel the process tests match; a zero or negative value is malformed configuration, never a silent fallback to the default, and all offenders report in one pass | exit code 2 naming the parameter |
| `retransmit-floor` | `--retransmit-interval` is at least 1ms — a nanosecond scan period is a busy loop | exit code 2 |
| `sync-response-cap` | `--sync-response-rate` is at most 1e9/s — above the core's own cap the pacing division would truncate to zero spacing | exit code 2 |
| `sync-chunk-ceiling` | `--sync-chunk-bytes` stays within 1..65507 — the datagram ceiling; a larger budget describes a chunk no datagram can carry | exit code 2 |
| `datagram-ceiling` | the largest record the write API accepts — a 1 024-byte key with a value of `--value-max-bytes` — must fit 65 507 bytes, one UDP datagram; the node measures the worst case through the real codec before any bind | exit code 2 naming the flag, the key limit and the computed size |
| `peer-vs-hello` | `peer_timeout > hello_interval` — a peer evicted at the same instant it greets would never stay discovered | exit code 2 |
| `ack-max` | `ack_timeout ≤ ack_timeout_max` — the retransmit ladder's ceiling sits at or above its base; equality is a single rung | exit code 2 |
| `drift-vs-lead` | `drift_warn < max_clock_lead` at the record layout's millisecond resolution — `max_clock_lead` must be at least `drift_warn` plus 1ms, so the drift WARN window stays reachable | exit code 2 |
| `metrics-cert-triple` | `SUPKV_METRICS_CERT`, `SUPKV_METRICS_KEY` and `SUPKV_METRICS_CLIENT_CA` are all-or-nothing | exit code 2 naming every env of the triple |
| `address-shape` | the four addresses must be host:port with a port in 0..65535 — port 0 is legal on a listener but rejected on `--broadcast-addr`, a destination the kernel can never send to, and `--grpc-addr` must not be empty | exit code 2 |

Two facts complete the catalog's operator picture. Parse (not Validate)
rejects positional arguments by name — a non-flag argument is a
configuration error with exit code 2, never a silently ignored token. And
inside the replication core a zero or negative `max_clock_lead` normalizes
to the 2 m design default (the library caller's behavior; the binary path
never exercises it because the positivity rule above exits first).

One relation the validator does not enforce, stated so nobody assumes
otherwise: the dump cycle coalesces bursts only while
`dump_debounce ≤ dump_max_wait`. Both knobs carry positivity rules only; a
`--dump-max-wait` below `--dump-debounce` starts the node, and the max-wait
deadline then truncates every quiet window — bursts dump at the deadline
instead of coalescing. docs/PERFORMANCE.md's fast profile keeps the debounce
well under the max-wait.

## Must match on all nodes

Four knobs are cluster-wide settings, not per-node preferences — configure
them identically on every node. Three are correctness constraints and one
keeps every node's reconciliation cost the same; the as-built reasoning
lives in docs/OPERATIONS.md (## Scale envelope, ## Record and datagram
limits, ## Configuration validation) and is cross-linked here, not
restated divergently:

- `--node-id` must be unique on the segment: a duplicated node ID is a
  configuration error — the peers classify the impostor's datagrams
  `duplicate_node_id` and drop them, so neither node converges with the
  other;
- `--value-max-bytes` — a node with a smaller limit refuses records over its
  limit (logged, never acked), so the sender retransmits until supersession
  and those records never converge on the strict node;
- `--max-clock-lead` — a node with a smaller bound discards versions its
  peers still consider valid (`bad_version`), so writes visible elsewhere
  stay invisible on the strict node;
- `--sync-chunk-bytes` — the protocol tolerates a disagreement (each
  listing stream carries its own chunk count), but the budget decides how
  many datagrams one full-listing round needs, so equal budgets keep every
  node's reconciliation cost the same; on every node it must stay within
  1..65507 regardless, which the startup validation enforces per node.
