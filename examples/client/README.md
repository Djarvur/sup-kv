# sup-kv client example

A standalone Go module that consumes the public sup-kv gRPC API. It is the
consumer-side proof of the public package: it imports only
`github.com/Djarvur/sup-kv/api/supkvapi/v1` plus gRPC, so it fails to build if
the public package ever grows an `internal/` dependency.

The module is separate from the server module and requires it through
`replace github.com/Djarvur/sup-kv => ../..`, which is exactly what any
out-of-tree consumer would do with a published version.

## Run it

Start a sup-kv node (see the repository README and `docs/EXAMPLES.md`), then:

```bash
cd examples/client
go run . -addr 127.0.0.1:28011 -prefix example/
```

Flags:

- `-addr` — the node's gRPC address (default `127.0.0.1:28011`).
- `-prefix` — the key prefix the example writes, lists and watches
  (default `example/`).

The process exits non-zero on the first failed RPC.

## What it demonstrates

| RPC | What the example shows |
|-----|------------------------|
| `Set` | A write with `wait_for_replication=true` and `wait_timeout_ms=1000`. The response reports the write's version and the replication outcome: `acked`, `acked_count` and `total_peers`. The wait never changes the write outcome; it only reports. |
| `Get` | The point read of the same key, carrying the full record (key, version, value). |
| `List` | A page loop with `page_size=2`: each page prints its `records` and its `next_page_token`. The loop follows `next_page_token` until it is empty, which is the end-of-list signal. A token is an opaque position, not a snapshot: keys added or deleted between pages may appear or disappear. |
| `Watch` | One bounded subscription to the prefix. The server sends the matching snapshot first, so the record written above arrives immediately; the example prints exactly one record and ends. The bounded window closing is a clean end. |

## The last-read-value cache is consumer-library territory

A consumer that must keep serving during a sup-kv outage commonly wants a
local copy of the last value it read. That last-read-value file pattern
belongs in the **consumer's own client library**, not in sup-kv: the server
stores values and streams changes, while the fallback policy (which value to
keep, when to serve it, how stale it may be) is application-specific.

ADR 0007 (owner decision 14.7) records that boundary. This example therefore
ships no cache and no fallback path — it is a direct, stateless demonstration
of the public API. The server contract itself is documented in `docs/API.md`.
