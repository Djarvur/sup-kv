# sup-kv

**sup-kv** ("supremum storage") is a leaderless replicated in-memory key-value store
for small clusters — two to a handful of nodes on one trusted LAN broadcast segment.

Every node accepts reads and writes locally and independently (AP); replicas exchange
changes over UDP and converge to the identical state — the pointwise supremum of all
writes under the record-version order (strong eventual consistency via a
Last-Write-Wins-Element-Map CRDT). It ships as one static Go binary with minimal
dependencies and human-readable JSON persistence.

For service builders on small HA clusters (e.g. two management boards in one appliance)
who need a shared, always-writable store for configuration and durable service state
where quorum-based stores (etcd, Consul) cannot survive the loss of a single node.

## Status

v1.0 feature-complete and continuously verified in CI: the LWW-Element-Map engine with
HLC/UUIDv7 record versions, atomic human-readable JSON persistence, UDP broadcast
replication with ack/retransmission and periodic anti-entropy sync, the gRPC API with
Watch streams and `wait_for_replication`, and the failure-scenario suite proving
convergence under loss, duplication, reordering, partitions and restarts
(`internal/stand/stand_red_test.go` drives the real-socket red scenarios; the
simulator bound scenarios are in `internal/app/repl/sim_bound_test.go`).

Highlights:

- AP with strong eventual consistency: all nodes writable, no quorum, no leader
- Record versions are Hybrid Logical Clocks packed into UUID v7
- Replication over UDP: broadcast discovery, ack with retransmission, periodic full
  reconciliation — convergence bounds proven by a deterministic simulator
  (`TestBound_normalOneRTT`, `TestBound_lossWithinAckTimeoutMax` and
  `TestBound_partitionHeal` in `internal/app/repl/sim_bound_test.go`, with the paced
  transport under load by `internal/stand/stand_red_test.go`)
- Human-readable JSON dumps with atomic writes and `.prev` fallback; named snapshots
- gRPC API with Watch streams, startup gate, `wait_for_replication`, snapshots and
  a health endpoint
- Opt-in mTLS metrics/debug endpoint and structured slog JSON logs
- Minimal dependencies, one static Go binary

## Install from a release

Download the archive for your platform from the
[GitHub releases page](https://github.com/Djarvur/sup-kv/releases) and unpack
it next to a working directory of your choice. From v1.1 on, the release
archives are Linux-only and contain the `sup-kv` binary,
[docs/DEPLOYMENT.md](docs/DEPLOYMENT.md), the `sup-kv.service` systemd unit
and the docs — the published v1.0.0 archives are the current example of the
older shape (the binary plus `CHANGELOG.md`, `LICENSE` and this README, with
darwin archives included, predating the macOS development-only demotion per
[ADR 0010](docs/adr/0010-darwin-development-only.md)).

Every release also ships a `checksums.txt` manifest and one SPDX SBOM per
archive. Verify the download before running the binary:

```bash
shasum -c checksums.txt
```

The checksum line must read `OK` for every archive you keep. The SBOM (one
`*.sbom` beside each archive, produced by syft at release time) lists exactly
what the build assembled, so you can audit an artifact's contents against
this repository without rebuilding it.

## Quickstart

Build the binary from a checkout of this repository:

```bash
go build -o sup-kv ./cmd/sup-kv
```

The client throughout is [grpcurl](https://github.com/fullstorydev/grpcurl) — sup-kv
serves gRPC reflection, so no proto files are needed, and
[docs/EXAMPLES.md](docs/EXAMPLES.md) walks every RPC the same way.

### Quickstart

<!-- quickstart-verify -->
```bash
./sup-kv --node-id 1 --bind-addr 0.0.0.0:27901 --broadcast-addr 127.255.255.255:27902 \
  --grpc-addr 127.0.0.1:27911 --dump-path supkv-a.json --startup-gate-empty 2s & NODE_A=$!
./sup-kv --node-id 2 --bind-addr 0.0.0.0:27902 --broadcast-addr 127.255.255.255:27901 \
  --grpc-addr 127.0.0.1:27912 --dump-path supkv-b.json --startup-gate-empty 2s & NODE_B=$!
# Node A accepts writes once health reports SERVING: the demo pins the
# empty-disk gate to 2s (an empty node otherwise waits 60s for a peer listing);
# a node with data on disk opens on the 1s startup gate.
for i in $(seq 1 50); do
  if grpcurl -plaintext -d '{}' 127.0.0.1:27911 grpc.health.v1.Health/Check 2>/dev/null | grep -q '"SERVING"'; then break; fi
  sleep 0.2
done
# Write on node A (value is proto3 bytes => base64 in JSON; c3VwLWt2 is "sup-kv").
grpcurl -plaintext -d '{"key":"demo/hello","value":"c3VwLWt2"}' \
  127.0.0.1:27911 supkvapi.v1.SupKVService/Set
# The write replicates over UDP broadcast — read it on node B (bounded retry).
for i in $(seq 1 50); do
  if grpcurl -plaintext -d '{"key":"demo/hello"}' 127.0.0.1:27912 supkvapi.v1.SupKVService/Get 2>/dev/null | grep -q c3VwLWt2; then break; fi
  sleep 0.2
done
grpcurl -plaintext -d '{"key":"demo/hello"}' 127.0.0.1:27912 supkvapi.v1.SupKVService/Get
# Hard-kill node A and restart it with the identical arguments.
kill -9 $NODE_A; wait $NODE_A 2>/dev/null || true
./sup-kv --node-id 1 --bind-addr 0.0.0.0:27901 --broadcast-addr 127.255.255.255:27902 \
  --grpc-addr 127.0.0.1:27911 --dump-path supkv-a.json --startup-gate-empty 2s & NODE_A=$!
# The restarted node serves the state again with no operator action (bounded retry).
for i in $(seq 1 50); do
  if grpcurl -plaintext -d '{"key":"demo/hello"}' 127.0.0.1:27911 supkvapi.v1.SupKVService/Get 2>/dev/null | grep -q c3VwLWt2; then break; fi
  sleep 0.2
done
grpcurl -plaintext -d '{"key":"demo/hello"}' 127.0.0.1:27911 supkvapi.v1.SupKVService/Get
kill $NODE_A $NODE_B 2>/dev/null; wait 2>/dev/null || true
rm -f supkv-a.json supkv-b.json
```

Copy-paste the block into a bash shell at the repository root — the directory the
build above placed the binary in; the block invokes it as `./sup-kv`. It starts two
nodes on one broadcast segment, writes on node A, reads the value on node B,
hard-kills node A with `kill -9`, restarts it with the identical arguments and reads
the state back — no operator action. CI runs this exact block verbatim
(`cmd/sup-kv/readme_test.go`), so it cannot drift from reality. Values are proto3
`bytes`, which JSON carries as base64: `c3VwLWt2` is `"sup-kv"`. The demo crosses
the UDP ports — node A binds 27901 and broadcasts to 27902, node B the mirror —
because two processes on one host cannot both bind the same replication port; on a
real segment every board binds the same port and broadcast address
([docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) ## Topology). The block cleans up
after itself.

### Quickstart on macOS

<!-- quickstart-verify -->
```bash
BCAST=$(ipconfig getoption en0 broadcast 2>/dev/null || echo 127.255.255.255)
./sup-kv --node-id 1 --bind-addr 0.0.0.0:27901 --broadcast-addr $BCAST:27902 \
  --grpc-addr 127.0.0.1:27911 --dump-path supkv-a.json --startup-gate-empty 2s & NODE_A=$!
./sup-kv --node-id 2 --bind-addr 0.0.0.0:27902 --broadcast-addr $BCAST:27901 \
  --grpc-addr 127.0.0.1:27912 --dump-path supkv-b.json --startup-gate-empty 2s & NODE_B=$!
# Node A accepts writes once health reports SERVING: the demo pins the
# empty-disk gate to 2s; a node with data on disk opens on the 1s startup gate.
for i in $(seq 1 50); do
  if grpcurl -plaintext -d '{}' 127.0.0.1:27911 grpc.health.v1.Health/Check 2>/dev/null | grep -q '"SERVING"'; then break; fi
  sleep 0.2
done
# Write on node A (value is proto3 bytes => base64 in JSON; c3VwLWt2 is "sup-kv").
grpcurl -plaintext -d '{"key":"demo/hello","value":"c3VwLWt2"}' \
  127.0.0.1:27911 supkvapi.v1.SupKVService/Set
# The write replicates over UDP broadcast — read it on node B (bounded retry).
for i in $(seq 1 50); do
  if grpcurl -plaintext -d '{"key":"demo/hello"}' 127.0.0.1:27912 supkvapi.v1.SupKVService/Get 2>/dev/null | grep -q c3VwLWt2; then break; fi
  sleep 0.2
done
grpcurl -plaintext -d '{"key":"demo/hello"}' 127.0.0.1:27912 supkvapi.v1.SupKVService/Get
# Hard-kill node A and restart it with the identical arguments.
kill -9 $NODE_A; wait $NODE_A 2>/dev/null || true
./sup-kv --node-id 1 --bind-addr 0.0.0.0:27901 --broadcast-addr $BCAST:27902 \
  --grpc-addr 127.0.0.1:27911 --dump-path supkv-a.json --startup-gate-empty 2s & NODE_A=$!
# The restarted node serves the state again with no operator action (bounded retry).
for i in $(seq 1 50); do
  if grpcurl -plaintext -d '{"key":"demo/hello"}' 127.0.0.1:27911 supkvapi.v1.SupKVService/Get 2>/dev/null | grep -q c3VwLWt2; then break; fi
  sleep 0.2
done
grpcurl -plaintext -d '{"key":"demo/hello"}' 127.0.0.1:27911 supkvapi.v1.SupKVService/Get
kill $NODE_A $NODE_B 2>/dev/null; wait 2>/dev/null || true
rm -f supkv-a.json supkv-b.json
```

**What the derive line actually does:** `ipconfig getoption en0 broadcast` queries
the DHCP broadcast option on the hard-coded interface `en0` and needs the network
to have supplied one — when it prints nothing (measured: exit 1), the line falls
back to `127.255.255.255`, the loopback broadcast address the first block uses.
That fallback does not fan out on macOS, so with it the two nodes never hear each
other: the peers stay at zero and the cross-node read fails.

**darwin is development-only** ([ADR 0010](docs/adr/0010-darwin-development-only.md)):
building and unit tests remain supported on macOS, replication is untested on
darwin, and the release archives ship Linux only — this section documents the
platform's status, it does not make macOS a deployment target. CI keeps the macos
cells as cross-compile legs (build and vet, no tests) and does not prove the pair
there either — the macos-15 runner cannot deliver broadcast datagrams
(`sendto: no route to host`, recorded when this topology was probed). CI executes
this exact block verbatim on Linux (`cmd/sup-kv/readme_test.go`), where the derive
line falls back to loopback broadcast, so the block itself stays verified.

## Documentation

In a repository clone the docs live under `docs/`; from v1.1 on they ship inside
the release archive, so an archive reader finds the same pages beside the unpacked
binary — the links below work from either place.

- [docs/OPERATIONS.md](docs/OPERATIONS.md) — operations: the trust model, value-size
  envelope, local-read semantics, the "not a lock/election service" warning and the
  clock-drift runbook
- [docs/EXAMPLES.md](docs/EXAMPLES.md) — every RPC as a copy-paste grpcurl
  walkthrough, executed by CI against the shipped binary
- [docs/IDEA.md](docs/IDEA.md) — the original design document: the decided
  architecture, parameters and failure-scenario rationale

## License

MIT — see [LICENSE](LICENSE).
