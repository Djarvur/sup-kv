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
convergence under loss, duplication, reordering, partitions and restarts. The v1.0.0
release artifacts are in preparation; until the tag lands, build from source.

Highlights:

- AP with strong eventual consistency: all nodes writable, no quorum, no leader
- Record versions are Hybrid Logical Clocks packed into UUID v7
- Replication over UDP: broadcast discovery, ack with retransmission, periodic full
  reconciliation — convergence bounds proven by a deterministic simulator
- Human-readable JSON dumps with atomic writes and `.prev` fallback; named snapshots
- gRPC API with Watch streams, startup gate, `wait_for_replication`, snapshots and
  a health endpoint
- Opt-in mTLS metrics/debug endpoint and structured slog JSON logs
- Minimal dependencies, one static Go binary

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
sup-kv --node-id 1 --bind-addr 0.0.0.0:27901 --broadcast-addr 127.255.255.255:27902 \
  --grpc-addr 127.0.0.1:27911 --dump-path supkv-a.json & NODE_A=$!
sup-kv --node-id 2 --bind-addr 0.0.0.0:27902 --broadcast-addr 127.255.255.255:27901 \
  --grpc-addr 127.0.0.1:27912 --dump-path supkv-b.json & NODE_B=$!
# Node A accepts writes once health reports SERVING (the startup gate, 1s default).
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
sup-kv --node-id 1 --bind-addr 0.0.0.0:27901 --broadcast-addr 127.255.255.255:27902 \
  --grpc-addr 127.0.0.1:27911 --dump-path supkv-a.json & NODE_A=$!
# The restarted node serves the state again with no operator action (bounded retry).
for i in $(seq 1 50); do
  if grpcurl -plaintext -d '{"key":"demo/hello"}' 127.0.0.1:27911 supkvapi.v1.SupKVService/Get 2>/dev/null | grep -q c3VwLWt2; then break; fi
  sleep 0.2
done
grpcurl -plaintext -d '{"key":"demo/hello"}' 127.0.0.1:27911 supkvapi.v1.SupKVService/Get
kill $NODE_A $NODE_B 2>/dev/null; wait 2>/dev/null || true
```

Copy-paste the block into a bash shell. It starts two nodes on one broadcast segment,
writes on node A, reads the value on node B, hard-kills node A with `kill -9`, restarts
it with the identical arguments and reads the state back — no operator action. CI runs
this exact block verbatim (`cmd/sup-kv/readme_test.go`), so it cannot drift from
reality. Values are proto3 `bytes`, which JSON carries as base64: `c3VwLWt2` is
`"sup-kv"`. The block cleans up after itself.

### Quickstart on macOS

<!-- quickstart-verify -->
```bash
BCAST=$(ipconfig getoption en0 broadcast 2>/dev/null || echo 127.255.255.255)
sup-kv --node-id 1 --bind-addr 0.0.0.0:27901 --broadcast-addr $BCAST:27902 \
  --grpc-addr 127.0.0.1:27911 --dump-path supkv-a.json & NODE_A=$!
sup-kv --node-id 2 --bind-addr 0.0.0.0:27902 --broadcast-addr $BCAST:27901 \
  --grpc-addr 127.0.0.1:27912 --dump-path supkv-b.json & NODE_B=$!
# Node A accepts writes once health reports SERVING (the startup gate, 1s default).
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
sup-kv --node-id 1 --bind-addr 0.0.0.0:27901 --broadcast-addr $BCAST:27902 \
  --grpc-addr 127.0.0.1:27911 --dump-path supkv-a.json & NODE_A=$!
# The restarted node serves the state again with no operator action (bounded retry).
for i in $(seq 1 50); do
  if grpcurl -plaintext -d '{"key":"demo/hello"}' 127.0.0.1:27911 supkvapi.v1.SupKVService/Get 2>/dev/null | grep -q c3VwLWt2; then break; fi
  sleep 0.2
done
grpcurl -plaintext -d '{"key":"demo/hello"}' 127.0.0.1:27911 supkvapi.v1.SupKVService/Get
kill $NODE_A $NODE_B 2>/dev/null; wait 2>/dev/null || true
```

The first block uses `127.255.255.255` (loopback broadcast), which does not fan out on
macOS; this variant derives the segment broadcast address from the primary interface
instead. It is documentation for macOS — CI does not prove it there (the macos-15
runner cannot deliver broadcast datagrams: `sendto: no route to host`, recorded when
this topology was probed) — but CI does execute this exact block verbatim on Linux,
where the derive line falls back to loopback broadcast, so the block itself stays
verified.

## Documentation

- [docs/OPERATIONS.md](docs/OPERATIONS.md) — operations: the trust model, value-size
  envelope, local-read semantics, the "not a lock/election service" warning and the
  clock-drift runbook
- [docs/EXAMPLES.md](docs/EXAMPLES.md) — every RPC as a copy-paste grpcurl
  walkthrough, executed by CI against the shipped binary
- [docs/IDEA.md](docs/IDEA.md) — the original design document: the decided
  architecture, parameters and failure-scenario rationale

## License

MIT — see [LICENSE](LICENSE).
