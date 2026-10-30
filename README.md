# sup-kv

**sup-kv** ("supremum storage") is a leaderless replicated in-memory key-value store
for small clusters — two to a handful of nodes on one trusted LAN broadcast segment.

Every node accepts reads and writes locally and independently (AP); replicas exchange
changes over UDP and converge to the identical state — the pointwise supremum of all
writes under the record-version order (strong eventual consistency via a
Last-Write-Wins-Element-Map CRDT).

## Status

Design phase. The full product idea and the decided architecture live in
[docs/IDEA.md](docs/IDEA.md).

Planned highlights:

- AP with strong eventual consistency: all nodes writable, no quorum, no leader
- Record versions are Hybrid Logical Clocks packed into UUID v7
- Replication over UDP: broadcast discovery, ack with retransmission, periodic
  full reconciliation
- Human-readable JSON dumps with atomic writes and snapshots
- gRPC API with change subscriptions
- Minimal dependencies, one static Go binary

## License

MIT — see [LICENSE](LICENSE).
