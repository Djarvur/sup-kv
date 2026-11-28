# sup-kv deployment guide

How to put sup-kv on real hardware: two or a few Linux boards on one trusted
LAN broadcast segment, one static binary per board, one systemd unit per
board. The companion references carry the details this guide links rather
than restates: every flag, default and validation relation is
[docs/CONFIGURATION.md](CONFIGURATION.md); the operator semantics (bounds,
runbooks, metrics, exit codes) are [docs/OPERATIONS.md](OPERATIONS.md); the
on-disk and on-wire formats are [docs/FORMATS.md](FORMATS.md).

**Platform.** Linux is the production platform. darwin is development-only
([ADR 0010](adr/0010-darwin-development-only.md)): on macOS, datagrams
larger than ~9 KiB do not leave the machine, which silently breaks
replication for realistic record sizes. macOS stays buildable for
development; it is not a deployment target, and the release archives ship
Linux only.

The two fenced blocks marked `deployment-verify` in this guide are
executed verbatim in CI (`cmd/sup-kv/deployment_test.go`): they run the
two-board demo and the health/metrics check on one loopback host against the
real binary, so the shapes shown here cannot drift from what the binary
does. On your segment the same argument shapes apply with your addresses.

## Topology

A sup-kv cluster is two to a handful of boards on **one L2 broadcast
segment** — the product niche is a single trusted LAN; there is no WAN
mode, no explicit peer list and no seeding step. Every node accepts reads
and writes locally and independently; replicas exchange changes over UDP
broadcast and converge to identical state. Discovery is the `Hello`
broadcast: a node only needs `--bind-addr` and `--broadcast-addr` to find
its peers, and a new board joining the segment is discovered on its first
`Hello` (within one `--hello-interval`, default 1s).

The canonical two-board shape:

- each board runs one `sup-kv` process with its own **distinct
  `--node-id`**;
- **both nodes bind the same replication port** (`--bind-addr
  0.0.0.0:7900` in the unit example below) — the wildcard bind, so the
  board receives the segment's broadcast datagrams;
- `--broadcast-addr` is the **segment broadcast address at that same port**
  (`192.168.1.255:7900` in the unit example — adapt to your segment), so
  **one datagram reaches every board**: a write is one broadcast, and every
  peer's acknowledgement comes back over the same rule.

The shape generalizes to N boards unchanged: still one bind port, still one
broadcast destination, still distinct node-ids — the peer table fills
itself, sync is chunked and paced, and no per-node configuration names any
other node.

**The minimal single-node shape**, stated explicitly so a reader with one
board is not left extrapolating: one node, no peers, health SERVING after
the startup gate — a node that loaded records from disk opens on the 1s
`--startup-gate` backstop; a node with an empty disk waits
`--startup-gate-empty` (default 60s) for a peer listing that never comes,
so a lone first board should either have a dump on disk or pin a shorter
empty-disk gate. Everything else in this guide (identity, dump path, sizes,
time sync, the unit) applies unchanged; replication simply has nothing to
do.

### The two-board demo on one machine

<!-- deployment-verify -->
```bash
# Two nodes on one machine stand in for the two boards. One loopback host
# cannot double-bind the LAN shape's single replication port, so the demo
# points each node's --broadcast-addr at the peer's port through the loopback
# broadcast address 127.255.255.255 (node A sends to node B's port and vice
# versa) — the same bind-listens / broadcast-sends relation the segment
# broadcast gives two boards, with per-node unicast cross-addressing instead.
sup-kv --node-id 11 --bind-addr 0.0.0.0:27921 --broadcast-addr 127.255.255.255:27922 \
  --grpc-addr 127.0.0.1:27923 --metrics-addr 127.0.0.1:27925 \
  --dump-path supkv-deploy-a.json --startup-gate-empty 2s & echo $! > deploy-a.pid
sup-kv --node-id 12 --bind-addr 0.0.0.0:27922 --broadcast-addr 127.255.255.255:27921 \
  --grpc-addr 127.0.0.1:27924 --metrics-addr 127.0.0.1:27926 \
  --dump-path supkv-deploy-b.json --startup-gate-empty 2s & echo $! > deploy-b.pid
# Bounded wait: a node answers health once its startup gate has opened. The
# demo pins the empty-disk gate to 2s (an empty node otherwise waits 60s for a
# peer listing; a node with data on disk opens on the 1s startup gate).
for i in $(seq 1 50); do
  if grpcurl -plaintext -d '{}' 127.0.0.1:27923 grpc.health.v1.Health/Check 2>/dev/null | grep -q '"SERVING"' \
    && grpcurl -plaintext -d '{}' 127.0.0.1:27924 grpc.health.v1.Health/Check 2>/dev/null | grep -q '"SERVING"'; then
    break
  fi
  sleep 0.2
done
# The unconditional proof: both boards report SERVING.
grpcurl -plaintext -d '{}' 127.0.0.1:27923 grpc.health.v1.Health/Check
grpcurl -plaintext -d '{}' 127.0.0.1:27924 grpc.health.v1.Health/Check
# Write on board A (value is proto3 bytes => base64 in JSON; c3VwLWt2 is "sup-kv").
grpcurl -plaintext -d '{"key":"deploy/hello","value":"c3VwLWt2"}' \
  127.0.0.1:27923 supkvapi.v1.SupKVService/Set
# One UDP datagram to the broadcast address carries the write to board B (bounded retry).
for i in $(seq 1 50); do
  if grpcurl -plaintext -d '{"key":"deploy/hello"}' 127.0.0.1:27924 supkvapi.v1.SupKVService/Get 2>/dev/null | grep -q c3VwLWt2; then break; fi
  sleep 0.2
done
grpcurl -plaintext -d '{"key":"deploy/hello"}' 127.0.0.1:27924 supkvapi.v1.SupKVService/Get
```

The block needs the `sup-kv` binary and [grpcurl](https://github.com/fullstorydev/grpcurl)
on your PATH. Both nodes report SERVING, the write on node A is read back on
node B within a round trip (normal-operation convergence is one RTT; the
documented bounds under loss and partitions are
[docs/OPERATIONS.md](OPERATIONS.md) `## What you are operating`), and the
pair keeps running — the next marked block stops it. On one machine the
demo leaves the two dump files (`supkv-deploy-a.json`, `supkv-deploy-b.json`)
and the two PID files (`deploy-a.pid`, `deploy-b.pid`) in the working
directory; they are the same artifacts a real deployment keeps under its
state directory.

## Addresses and ports

`--bind-addr` and `--broadcast-addr` are the two halves of one relation —
**bind is where this node listens; broadcast is where it sends**:

- `--bind-addr 0.0.0.0:7900` — the UDP socket the node receives on. The
  wildcard bind is the deployment default: a broadcast datagram is accepted
  only by a wildcard or broadcast-address-bound socket, so a **non-wildcard
  bind warns at startup** (`Bind address is not a wildcard: broadcast
  datagrams are not received`) and the node then hears only unicast —
  replicas will not converge with it. Keep the wildcard on every board.
- `--broadcast-addr <segment-broadcast>:7900` — the UDP destination every
  write, `Hello` and sync chunk is sent to. The segment broadcast address
  is what makes one datagram reach all boards; the port is the same
  replication port every board binds.

**Broadcast-hostile networks** (some Wi-Fi APs, bridge/container setups
that drop subnet broadcast) have a two-board alternative: per-node unicast
cross-addressing — point each node's `--broadcast-addr` at the *peer's*
unicast address and port (`192.168.1.10:7900` on board A,
`192.168.1.11:7900` on board B). Every datagram is then sent once per peer,
which is exactly what the demo block above does over loopback. It scales to
exactly two boards — one `--broadcast-addr` cannot name three peers — so
beyond two boards, fix the network or use the segment broadcast.

The replication socket is the first of **three listeners** per node:

| Listener | Flag | Default | Notes |
|---|---|---|---|
| replication (UDP) | `--bind-addr` / `--broadcast-addr` | required — no default | the trusted segment's channel; CRC32 integrity only, no authentication ([docs/OPERATIONS.md](OPERATIONS.md) `## Trust model`) |
| gRPC API (TCP) | `--grpc-addr` | `127.0.0.1:7901` | loopback-only by default; a board's clients connect over the LAN, so the unit example binds `0.0.0.0:7901` |
| metrics/debug (TCP) | `--metrics-addr` | none — the listener stays off unless the flag (or `SUPKV_METRICS_ADDR`) is set | keep plaintext on loopback; see Metrics setup |

Port 0 is legal on a listener for an ephemeral bind and rejected on
`--broadcast-addr` — the kernel can never send to it
([docs/CONFIGURATION.md](CONFIGURATION.md) `## Validation relations`).

## Node identity

`--node-id` is a **16-bit integer (0..65535), required, and must be unique
across the cluster**. Assign it per board and keep it stable — it is part of
every record version (the writing node's identity in the packed UUIDv7), so
a board's identity survives restarts and dump copies. Small integers are
fine (`1`, `2`, …); nothing allocates them but you.

A **duplicated node-id is a configuration error**, not a data hazard: the
peers classify the impostor's datagrams `duplicate_node_id` and drop them —
`supkv_datagrams_discarded_total{class=duplicate_node_id}` rises on both
nodes with one rate-limited WARN per interval, and neither node converges
with the other. The symptom-cause-action runbook is
[docs/OPERATIONS.md](OPERATIONS.md) `## Runbook: duplicate node_id`; the fix
is to give one of the two processes its own id and restart it.

## Paths and permissions

The node's durable state is one human-readable JSON dump file at
`--dump-path` (required — no default). The node keeps **generations** beside
it:

- the **main** generation at the configured path — what the node loads and
  what every confirmed write is contained in;
- **`.prev`** — the previous generation, kept by every atomic rotation and
  the fallback the load walks when the main is absent or unusable
  (main → `.prev` → empty start, healing from peers);
- **`.corrupt-<UTC>[-N]`** — a generation the load rejected, preserved out
  of the rotation path as evidence, never rotated and never auto-deleted;
  and hidden **`.tmp~`** in-flight temp files, swept at startup when older
  than one hour.

Load outcomes, preservation and the `preserve_error` hazard are
[docs/OPERATIONS.md](OPERATIONS.md) `## Load outcomes`; the format is
[docs/FORMATS.md](FORMATS.md).

**Run it as a dedicated user.** The unit below runs as `sup-kv` with
`StateDirectory=sup-kv`, which systemd creates at **`/var/lib/sup-kv`**,
owned by that user — the dump path in the unit is
`/var/lib/sup-kv/supkv.json`. Create the user once per board:

```bash
sudo useradd --system --home-dir /var/lib/sup-kv --shell /usr/sbin/nologin sup-kv
```

**Size the dump directory for the full state.** A dump is a full-state
rewrite: main + `.prev` live simultaneously (2× the state), plus one in
flight temp file and any snapshots you take. The state is the whole store —
thousands to tens of thousands of keys at values up to the value limit — so
the comfortably sized floor is a few times the peak store size; the memory
model and the state→RSS points are [docs/PERFORMANCE.md](PERFORMANCE.md)
`## Memory model`.

## Sizes

The cluster's sizing knobs are small and few — the full table with defaults
is [docs/CONFIGURATION.md](CONFIGURATION.md):

- **values are at most `--value-max-bytes`** — 61440 bytes (60 KiB) by
  default, the single value limit (`storage.ValueMaxBytes`);
- **a record must fit one UDP datagram: 65 507 bytes**, the maximum IPv4
  UDP payload. With a maximal 1 024-byte key the largest fitting value is
  64 408 bytes; the node measures the worst case through the real codec at
  startup and exits 2 on a configuration that cannot fit;
- **`--sync-chunk-bytes`** (default 1400, within 1..65507) is the DataSync
  chunk byte budget — how many datagrams one full-listing round needs.

Four knobs are **cluster-wide settings, not per-node preferences** —
node-id uniqueness, `--value-max-bytes`, `--max-clock-lead` and
`--sync-chunk-bytes` — and they must match on every node. The list, the
reasons and the failure modes are
[docs/CONFIGURATION.md](CONFIGURATION.md) `## Must match on all nodes`;
configure them identically and deploy-time drift cannot silently split the
cluster.

## Time sync

Run **NTP (or PTP) on every board**. Correctness never depends on wall-clock
synchronization — record versions are hybrid logical clocks — but the wall
clock decides which of two simultaneous writes wins, and two thresholds read
it:

- `--drift-warn` (default 60s): a received version leading local wall time
  by more than this warns (`Clock drift` WARN, counted in
  `supkv_clock_drift_warnings_total`);
- `--max-clock-lead` (default 2m): a received version beyond now + this is
  discarded (`bad_version`) — a node whose clock runs far ahead finds its
  writes rejected by every peer instead of dominating the cluster. Startup
  requires `drift_warn < max_clock_lead`.

Exceeding the warn threshold is a clock-source problem, not a store
problem: fix NTP on the host that warns, never restart the cluster "to fix
drift" — it changes nothing. The runbook with the as-built directions and
the proven far-ahead behavior is [docs/OPERATIONS.md](OPERATIONS.md)
`## Clock drift runbook`.

## Dump setup

Dumps are how a confirmed write survives a crash: **a write returns OK only
after a dump containing it completed** — confirmed writes are always on
disk, and a `SIGKILL` still loses nothing confirmed.

- **Cadence.** Dumps are debounced: a burst of writes keeps postponing the
  dump for `--dump-debounce` (default 50ms), and the dump starts no later
  than the oldest unconfirmed write plus `--dump-max-wait` (default 1s) —
  at steady state that is **about one full-state rewrite per second at
  most**, and one per burst otherwise. Keep `dump_debounce ≤ dump_max_wait`
  (the intended relation the validator does not enforce —
  [docs/CONFIGURATION.md](CONFIGURATION.md) `## Validation relations`).
- **Disk cost profile.** Every dump rewrites the whole state: the write
  amplification is the state size per second at the cadence above, measured
  at 14.36× the logical bytes on an NVMe/ext4 workstation in
  [docs/PERFORMANCE.md](PERFORMANCE.md) `## Disk-write amplification` —
  size the media for the cadence, not for the logical store size.
- **The fallback chain.** Each rotation keeps the previous generation as
  `.prev`; a load that cannot decode the main falls back to it, and a
  generation it rejects is preserved as `.corrupt-<UTC>`. The chain, the
  outcomes and the retry ladder (a failed dump is retried, never fatal —
  writes answer `Unavailable` until the next success) are
  [docs/OPERATIONS.md](OPERATIONS.md) `## Load outcomes` and
  `## Runbook: dump failure and full disk`.
- **Sizing the dump directory** is under Paths and permissions above: main
  + `.prev` + one temp + snapshots.
- **When to tune.** The defaults are the documented profile. The
  fast-response profile (`dump_debounce` 1 ms, max-wait 50 ms, in-memory
  dump) and the latency it buys are measured in
  [docs/PERFORMANCE.md](PERFORMANCE.md) `## Write-latency model`; tune the
  pair only with that data.

## Metrics setup

The metrics/debug endpoint is **off unless `--metrics-addr` is set** (or its
`SUPKV_METRICS_ADDR` twin). When set, the node serves:

- `/metrics` — Prometheus text format, the `supkv_*` families plus the
  standard `go_*`/`process_*` runtime families;
- `/debug/vars` (expvar) and `/debug/pprof/*` — runtime introspection;
  because pprof is an arbitrary-cost endpoint, **do not expose the listener
  beyond loopback as plaintext** — the unit example keeps
  `--metrics-addr 127.0.0.1:7902` for a local scraper.

For scraping from **another host**, turn on the mutual-TLS triple — the
three env-only knobs (no flags), all-or-nothing, any partial set is a
startup error:

| Knob | File format |
|---|---|
| `SUPKV_METRICS_CERT` | PEM-encoded X.509 server certificate chain |
| `SUPKV_METRICS_KEY` | PEM-encoded private key matching the certificate |
| `SUPKV_METRICS_CLIENT_CA` | PEM-encoded CA certificate clients are verified against |

With all three set the endpoint requires a client certificate issued under
that CA; the certificate and key material never appear in logs.

**What to scrape and alert on**: the typed family table — every
`supkv_*` family with its type, labels, meaning and recommended alert — is
[docs/OPERATIONS.md](OPERATIONS.md) `## Metrics reference`. This guide does
not duplicate it; the one deployment-level hint is that
`supkv_replication_peers` is the fastest "am I actually clustered" check
(the demo block below scrapes it).

### Check health and scrape metrics

<!-- deployment-verify -->
```bash
# The demo pair from Topology is still running (the PID files carry it).
# Health on node A: the quoted grep cannot mistake NOT_SERVING for SERVING.
grpcurl -plaintext -d '{}' 127.0.0.1:27923 grpc.health.v1.Health/Check | grep '"SERVING"'
# The metrics endpoint node A's --metrics-addr opened serves the supkv_*
# families; their meanings are the metrics reference in docs/OPERATIONS.md.
curl -sf http://127.0.0.1:27925/metrics | grep '^supkv_replication_peers'
# Stop the pair the way systemd would: SIGTERM runs the ordered shutdown
# (final confirmed dump, exit code 0 on a clean stop).
kill "$(cat deploy-a.pid)" "$(cat deploy-b.pid)" 2>/dev/null || true
for i in $(seq 1 50); do
  if ! kill -0 "$(cat deploy-a.pid)" 2>/dev/null && ! kill -0 "$(cat deploy-b.pid)" 2>/dev/null; then break; fi
  sleep 0.2
done
# The unconditional proof: fail the block if either node is still alive.
ps -p "$(cat deploy-a.pid)" >/dev/null 2>&1 && exit 1
ps -p "$(cat deploy-b.pid)" >/dev/null 2>&1 && exit 1
echo "demo pair stopped: both nodes exited"
```

The scrape carries `supkv_replication_peers` with each node's live peer
count (1 — each other), and the SIGTERM stops both nodes cleanly.

## Upgrading and compatibility

Upgrades are rolling and order-free; two gates make mixed versions safe
instead of corrupting:

- **The wire gate: `proto_version`.** Every datagram announces its protocol
  version (this build speaks 2), and it is checked before the CRC and the
  body: a foreign version's datagrams are classified `bad_proto_version`
  and discarded — including every `Hello`, so peers on different versions
  simply never discover each other. Each node keeps serving, persisting
  and retransmitting independently until the versions agree, and
  convergence resumes automatically once they do. The runbook is
  [docs/OPERATIONS.md](OPERATIONS.md) `## Runbook: incompatible versions`;
  upgrade one node at a time in any order.
- **The dump gate: format v1.** The dump format is stable at
  `format_version` 1; an unknown version makes a generation unreadable and
  the fallback chain advances. `record_count` was added as an optional
  field inside v1: fewer record lines than announced is an incomplete
  generation, more is tolerated, and a legacy v1.0 dump without the field
  loads unchanged through the documented path — no migration step exists or
  is needed. The full policy is
  [docs/FORMATS.md](FORMATS.md) `## Format-version policy`. After an
  upgrade, let one confirmed write rotate the main generation before
  relying on the census (the legacy-generation caveat in
  [docs/OPERATIONS.md](OPERATIONS.md) `## Load outcomes`).

## Node replacement

Replace a dead or dying board without stopping the cluster (the survivors
keep accepting writes the whole time — AP):

1. **Stop the old node** if it still runs: SIGTERM takes the graceful path
   (final confirmed dump, exit code 0).
2. **Copy the dump** to the replacement board's state directory — the main
   generation is enough; copying `.prev` as well keeps the fallback chain.
   Keep the **same `--node-id`**: the identity is part of every record
   version the board wrote, and a copy under a new id would present old
   writes as new ones.
3. **Start the new node** with the same argument set and the dump at the
   same `--dump-path`.

What the neighbours see: the board went silent (evicted from their peer
tables after `--peer-timeout`, default 5s — the missing-peer runbook in
[docs/OPERATIONS.md](OPERATIONS.md)), then returned with a new `run_id` —
one WARN `Peer restarted` per peer, and a **full sync toward the restarted
node** (its key listing is fetched fresh, so nothing written while it was
down is lost). The node itself opens on the 1s `--startup-gate` once its
dump has loaded, and health flips to SERVING. A replacement with an **empty
disk** is also safe: it starts `prompt`, waits out `--startup-gate-empty`
(default 60s) for a peer's full listing, and heals from the cluster — the
documented recovery path for a lost disk
([docs/OPERATIONS.md](OPERATIONS.md) `## Runbook: corrupted dump`'s
load-outcome behavior).

## systemd unit

The repo ships a working unit at [`deploy/sup-kv.service`](../deploy/sup-kv.service);
**the release archives carry the unit file and this guide** (see
[docs/CONFIGURATION.md](CONFIGURATION.md) for the flag reference the unit's
arguments come from). Install per board:

```bash
# 1. the dedicated user (see Paths and permissions)
sudo useradd --system --home-dir /var/lib/sup-kv --shell /usr/sbin/nologin sup-kv
# 2. the binary at the ExecStart path
sudo install -m 0755 sup-kv /usr/local/bin/sup-kv
# 3. the unit (edit --node-id and --broadcast-addr per board first)
sudo install -m 0644 deploy/sup-kv.service /etc/systemd/system/sup-kv.service
# 4. start and enable
sudo systemctl daemon-reload
sudo systemctl enable --now sup-kv
```

The unit's ExecStart, explained (every flag is a
[docs/CONFIGURATION.md](CONFIGURATION.md) row; each non-default value is a
working LAN example to adapt per board):

```text
ExecStart=/usr/local/bin/sup-kv --node-id 1 --bind-addr 0.0.0.0:7900 --broadcast-addr 192.168.1.255:7900 --dump-path /var/lib/sup-kv/supkv.json --grpc-addr 0.0.0.0:7901 --metrics-addr 127.0.0.1:7902
```

- `--node-id 1` — this board's identity; **unique on the segment**, change
  it on every board (Node identity above);
- `--bind-addr 0.0.0.0:7900` — the wildcard replication bind that receives
  the segment's broadcast (Addresses and ports above);
- `--broadcast-addr 192.168.1.255:7900` — a /24 segment's broadcast
  address at the same port; adapt to your segment;
- `--dump-path /var/lib/sup-kv/supkv.json` — inside the unit's
  `StateDirectory=sup-kv`, which systemd creates owned by `User=sup-kv`;
- `--grpc-addr 0.0.0.0:7901` — the API on the LAN (the flag's own default
  is loopback-only); the segment is the trust boundary
  ([docs/OPERATIONS.md](OPERATIONS.md) `## Trust model`);
- `--metrics-addr 127.0.0.1:7902` — plaintext metrics kept on loopback;
  another-host scraping turns on the `SUPKV_METRICS_*` triple (Metrics
  setup above).

**Exit codes and the restart policy** — the unit encodes
[docs/OPERATIONS.md](OPERATIONS.md) `## Lifecycle and exit codes`:

| Exit code | Meaning | systemd reaction (`Restart=on-failure`, `RestartPreventExitStatus=2`) |
|---|---|---|
| 0 | clean shutdown — the final dump is confirmed on disk | no restart |
| 1 | startup failure (a bind/listen stage) or a `Serve` error | restarts after `RestartSec=2` |
| 2 | malformed configuration, naming the parameter | **no restart** — a usage error must never loop on the same bad arguments |
| 3 | a failed final dump (the operator's data-loss signal) | restarts after `RestartSec=2` |

**The journal is the log source.** The node logs structured JSON on stderr;
under the unit that lands in the journal — `journalctl -u sup-kv` (add
`SUPKV_LOG_FORMAT=text` via the unit's commented `Environment=` lines for
human-oriented output, `SUPKV_LOG_LEVEL` for verbosity). The message
catalog is [docs/OPERATIONS.md](OPERATIONS.md) `## Log-message catalog`.
