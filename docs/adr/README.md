# sup-kv ADR index

Every recorded architecture decision as a numbered ADR, 0001–0012. Each row
is derived from the ADR file's own frontmatter (`adr`, `review-decision`,
`informs`, `date`, `status`) — `TestADRIndex` in `cmd/sup-kv` keeps the
index and the files in one-row-per-file correspondence in both directions,
in numeric order, so an unindexed decision or a row without a file fails
`make test`. The decision bodies live in the linked files; this index never
paraphrases them.

| ADR | Decision | Informs | Date | Status |
|-----|----------|---------|------|--------|
| [0001](0001-snapshot-load-full-rollback.md) | 14.1 — Snapshot load is a full rollback as per Spec | APIS-07 | 2026-11-19 | accepted |
| [0002](0002-create-if-absent-blocked-by-tombstone.md) | 14.2 — Create-if-absent on a deleted key stays blocked by the tombstone | none | 2026-11-19 | accepted |
| [0003](0003-dump-policy-unchanged.md) | 14.3 — Dump write policy stays as-is | PERF-01 | 2026-11-19 | accepted |
| [0004](0004-no-conflict-journal.md) | 14.4 — No conflict journal | OBSV-03 | 2026-11-19 | accepted |
| [0005](0005-interface-closed-until-full-key-list.md) | 14.5 — Interface stays closed until the full key list arrives | PROTO-01 | 2026-11-19 | accepted |
| [0006](0006-startup-gate-empty-wait.md) | 14.6 — An empty-disk node waits longer for a peer | PROTO-02 | 2026-11-19 | accepted |
| [0007](0007-public-proto-consumer-zone.md) | 14.7 — Public proto package; the last-read-value file stays outside sup-kv | APIS-06 | 2026-11-19 | accepted |
| [0008](0008-metrics-off-by-default.md) | 14.8 — Metrics stay off by default | DOCS-10 | 2026-11-19 | accepted |
| [0009](0009-multi-node-support.md) | 14.9 — Three-or-more-node support is completed | PROTO-09 | 2026-11-19 | accepted |
| [0010](0010-darwin-development-only.md) | 14.10 — darwin is development-only | DOCS-10, XPORT-05 | 2026-11-19 | accepted |
| [0011](0011-value-limit-60kib-kept.md) | 14.11 — The ~60 KiB single-value limit stays | PERSIST-01 | 2026-11-19 | accepted |
| [0012](0012-equal-version-tiebreak.md) | P-07 — Equal-version tie-break | PERSIST-07 | 2026-11-22 | accepted |
