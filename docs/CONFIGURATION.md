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
