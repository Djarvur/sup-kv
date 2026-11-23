# sup-kv examples

A complete walk through the sup-kv API with plain `grpcurl` — no proto files,
no Go client code. The binary serves gRPC server reflection, so `grpcurl`
discovers the whole API (`supkvapi.v1.SupKVService`) by itself, and every
command below is copy-paste runnable against the shipped binary.

The marked blocks are executed verbatim by CI (`cmd/sup-kv/examples_test.go`),
so what you copy from this page is exactly what CI proves works. The blocks
run in order and share one running pair of nodes and one working directory —
work top to bottom the first time; the pair starts in the first block and
stays up until the last one cleans it up.

## Prerequisites

Two tools on your `PATH`:

- `sup-kv` — build it with `make build` (output in `bin/sup-kv`) or unpack a
  release archive;
- `grpcurl` — any recent release; `go install
  github.com/fullstorydev/grpcurl/cmd/grpcurl@latest` or a binary from
  <https://github.com/fullstorydev/grpcurl/releases>. In this repository a
  pinned version is recorded as a go.mod tool directive, so `go tool
  grpcurl` works here without installing anything.

Two wire facts that shape every command:

- values are proto3 `bytes`, which JSON carries as **base64 in both
  directions**. This page uses one literal throughout — `c3VwLWt2`, which is
  the string `sup-kv`:

  ```bash
  printf sup-kv | base64   # -> c3VwLWt2
  ```

- grpcurl prints pretty JSON with **camelCase field names**
  (`ackedCount`, not `acked_count`) and **omits zero-valued fields**, so a
  response only shows the fields that carry a value in that call.

The demo pair uses fixed ports: node A on UDP 28001 with gRPC at
`127.0.0.1:28011`, node B on UDP 28002 with gRPC at `127.0.0.1:28012`. The
nodes discover each other over loopback broadcast (`127.255.255.255`) — the
two-nodes-on-one-host demo topology; on a real cluster pass your LAN's
broadcast address instead.

### Start the demo pair

<!-- examples-verify -->
```bash
# Node A: node-id 1, UDP 28001 broadcasting to node B's 28002, gRPC on 28011,
# dump file in the current directory.
sup-kv --node-id 1 --bind-addr 0.0.0.0:28001 --broadcast-addr 127.255.255.255:28002 \
  --grpc-addr 127.0.0.1:28011 --dump-path supkv-ex-a.json --startup-gate-empty 2s & echo $! > ex-a.pid
# Node B: the mirror — UDP 28002 broadcasting to 28001, gRPC on 28012.
sup-kv --node-id 2 --bind-addr 0.0.0.0:28002 --broadcast-addr 127.255.255.255:28001 \
  --grpc-addr 127.0.0.1:28012 --dump-path supkv-ex-b.json --startup-gate-empty 2s & echo $! > ex-b.pid
# Bounded wait: a node answers health once its startup gate has opened (the
# first sync round, or the demo's pinned 2s empty-disk gate — a node with data
# on disk opens on the 1s startup gate).
for i in $(seq 1 50); do
  if grpcurl -plaintext -d '{}' 127.0.0.1:28011 grpc.health.v1.Health/Check 2>/dev/null | grep -q SERVING \
    && grpcurl -plaintext -d '{}' 127.0.0.1:28012 grpc.health.v1.Health/Check 2>/dev/null | grep -q SERVING; then
    break
  fi
  sleep 0.2
done
# The unconditional proof: both nodes report SERVING.
grpcurl -plaintext -d '{}' 127.0.0.1:28011 grpc.health.v1.Health/Check
grpcurl -plaintext -d '{}' 127.0.0.1:28012 grpc.health.v1.Health/Check
```

Each health check answers (output from a real run):

```
{
  "status": "SERVING"
}
```

The PID files (`ex-a.pid`, `ex-b.pid`) are how the final block stops the
pair — shell variables do not survive between the blocks on this page.

### Discover the API

<!-- examples-verify -->
```bash
# Reflection: list every service on the node — zero proto files needed.
grpcurl -plaintext 127.0.0.1:28011 list
# Describe the sup-kv service: every RPC with its request/response shapes.
grpcurl -plaintext 127.0.0.1:28011 describe supkvapi.v1.SupKVService
```

The listing names the three services a node serves (output from a real run):

```
grpc.health.v1.Health
grpc.reflection.v1.ServerReflection
grpc.reflection.v1alpha.ServerReflection
supkvapi.v1.SupKVService
```

And `describe` prints the whole surface — all seven RPCs, with `Watch`
marked as a stream:

```
supkvapi.v1.SupKVService is a service:
service SupKVService {
  rpc Delete ( .supkvapi.v1.DeleteRequest ) returns ( .supkvapi.v1.DeleteResponse );
  rpc Get ( .supkvapi.v1.GetRequest ) returns ( .supkvapi.v1.GetResponse );
  rpc List ( .supkvapi.v1.ListRequest ) returns ( .supkvapi.v1.ListResponse );
  rpc Set ( .supkvapi.v1.SetRequest ) returns ( .supkvapi.v1.SetResponse );
  rpc SnapshotCreate ( .supkvapi.v1.SnapshotCreateRequest ) returns ( .supkvapi.v1.SnapshotCreateResponse );
  rpc SnapshotLoad ( .supkvapi.v1.SnapshotLoadRequest ) returns ( .supkvapi.v1.SnapshotLoadResponse );
  rpc Watch ( .supkvapi.v1.WatchRequest ) returns ( stream .supkvapi.v1.WatchResponse );
}
```

### Read a missing key

<!-- examples-verify -->
```bash
# A Get of a key nobody has written fails with NotFound — absent and deleted
# keys are the NotFound error, never an empty response. grpcurl exits
# non-zero on RPC errors, so capture the output and assert the code.
grpcurl -plaintext -d '{"key":"demo/hello"}' 127.0.0.1:28011 \
  supkvapi.v1.SupKVService/Get >missing.out 2>&1 || true
grep -q NotFound missing.out && cat missing.out
```

The captured error (output from a real run):

```
ERROR:
  Code: NotFound
  Message: key not found
```

### Write and read

<!-- examples-verify -->
```bash
# Write on node A: value is proto3 bytes, so JSON carries base64
# (c3VwLWt2 is "sup-kv"). The response carries the new record's version —
# a 16-byte RFC 9562 record version, base64 on the wire.
grpcurl -plaintext -d '{"key":"demo/hello","value":"c3VwLWt2"}' \
  127.0.0.1:28011 supkvapi.v1.SupKVService/Set
# Read on node B — the write replicated across the pair. The first read may
# race the replication datagram, so retry bounded, then prove it with one
# final Get.
for i in $(seq 1 50); do
  if grpcurl -plaintext -d '{"key":"demo/hello"}' 127.0.0.1:28012 \
    supkvapi.v1.SupKVService/Get 2>/dev/null | grep -q c3VwLWt2; then break; fi
  sleep 0.2
done
grpcurl -plaintext -d '{"key":"demo/hello"}' 127.0.0.1:28012 \
  supkvapi.v1.SupKVService/Get
```

The plain `Set` prints only the version (output from a real run) — the
replication-report fields are zero-valued without a wait, and grpcurl omits
zero-valued fields:

```
{
  "version": "AaHppeNZcACAAEAAAAAAAA=="
}
```

The `Get` on node B returns the full record, value still base64:

```
{
  "record": {
    "key": "demo/hello",
    "version": "AaHppeNZcACAAEAAAAAAAA==",
    "value": "c3VwLWt2"
  }
}
```

### Wait for replication

<!-- examples-verify -->
```bash
# Set with wait_for_replication: the response waits for peer Acks (bounded
# by wait_timeout_ms) and reports the aggregate — acked, ackedCount,
# totalPeers. The wait never changes the write outcome; it only reports.
for i in $(seq 1 25); do
  grpcurl -plaintext \
    -d '{"key":"demo/wait","value":"c3VwLWt2","wait_for_replication":true,"wait_timeout_ms":5000}' \
    127.0.0.1:28011 supkvapi.v1.SupKVService/Set >wait.out 2>&1 || true
  if grep -q '"acked": true' wait.out && grep -q ackedCount wait.out && grep -q totalPeers wait.out; then
    break
  fi
  sleep 0.2
done
grep -q '"acked": true' wait.out && grep -q ackedCount wait.out && cat wait.out
```

With one peer up and healthy, all three report fields carry values (output
from a real run):

```
{
  "version": "AaHppi5DcACAAEAAAAAAAA==",
  "acked": true,
  "ackedCount": 1,
  "totalPeers": 1
}
```

`ackedCount` counts the peers that Acked within the timeout; `totalPeers`
is the peer set captured once at write acceptance — a peer discovered
during the wait never joins it. A `false` acked means a peer did not confirm
in time — the write still stands on the writer and keeps retransmitting.
With zero known peers at write time the answer is immediate: `acked=false`
with `ackedCount` and `totalPeers` at zero (both omitted by grpcurl),
because nothing was awaited and nothing was acknowledged.

### Delete and list

<!-- examples-verify -->
```bash
# List is one method with two modes. page_size > 0 asks for a bounded page
# of records: the page's keys plus the same entries with their values, and
# nextPageToken naming the last examined key while the listing continues.
# Page 1 (page_size 1) over the two live demo/ keys stops at demo/hello.
grpcurl -plaintext -d '{"prefix":"demo/","page_size":1}' 127.0.0.1:28011 \
  supkvapi.v1.SupKVService/List >page1.out
grep -q demo/hello page1.out && grep -q nextPageToken page1.out
# Resume strictly after that position (the token is the raw-URL base64 of
# demo/hello): page 2 carries the last key and an empty nextPageToken — the
# end signal, which grpcurl omits.
grpcurl -plaintext -d '{"prefix":"demo/","page_size":1,"page_token":"ZGVtby9oZWxsbw"}' \
  127.0.0.1:28011 supkvapi.v1.SupKVService/List >page2.out
grep -q demo/wait page2.out && ! grep -q nextPageToken page2.out
# Delete demo/hello — a delete is a write: fresh version, replicates like
# one, and leaves a tombstone.
grpcurl -plaintext -d '{"key":"demo/hello"}' 127.0.0.1:28011 \
  supkvapi.v1.SupKVService/Delete
# An unset page_size keeps the legacy all-keys listing: every live key in
# one response, keys only — no records, no token. demo/hello is gone
# (tombstoned), demo/wait remains.
grpcurl -plaintext -d '{"prefix":"demo/"}' 127.0.0.1:28011 \
  supkvapi.v1.SupKVService/List >list.out
grep -q demo/wait list.out && ! grep -q '"demo/hello"' list.out && \
  cat page1.out page2.out list.out
```

Page 1 carries `demo/hello` as a record with its value and the position
token naming the key after it; the resumed page carries `demo/wait` with an
empty token, the end of the listing. The delete answers with the tombstone's
version, and the legacy listing carries keys only (output from a real run):

```
{
  "version": "AaHppjFfcACAAEAAAAAAAA=="
}
{
  "keys": [
    "demo/hello"
  ],
  "records": [
    {
      "key": "demo/hello",
      "version": "AaHppeNZcACAAEAAAAAAAA==",
      "value": "c3VwLWt2"
    }
  ],
  "nextPageToken": "ZGVtby9oZWxsbw"
}
{
  "keys": [
    "demo/wait"
  ],
  "records": [
    {
      "key": "demo/wait",
      "version": "AaHppi5DcACAAEAAAAAAAA==",
      "value": "c3VwLWt2"
    }
  ]
}
{
  "keys": [
    "demo/wait"
  ]
}
```

### Watch a stream

<!-- examples-verify -->
```bash
# Subscribe on node A to every change under demo/. Watch is a server
# stream: it first sends the matching snapshot, then live events for local
# and replicated changes. Run it in the background writing to a file.
grpcurl -plaintext -d '{"prefix":"demo/"}' 127.0.0.1:28011 \
  supkvapi.v1.SupKVService/Watch >watch.out & WATCH=$!
sleep 1
# Write demo/live on node B; the change replicates to A and appears on the
# stream.
grpcurl -plaintext -d '{"key":"demo/live","value":"c3VwLWt2"}' 127.0.0.1:28012 \
  supkvapi.v1.SupKVService/Set >set-live.out
for i in $(seq 1 50); do
  if grep -q demo/live watch.out 2>/dev/null; then break; fi
  sleep 0.2
done
# An endless stream has no natural exit — stop the watcher, then prove the
# event arrived (the assert is the block's last command, so exhaustion is
# fatal, not silent).
kill $WATCH 2>/dev/null || true; wait $WATCH 2>/dev/null || true
grep -q demo/live watch.out && cat watch.out
```

The stream file shows the shape (output from a real run): the snapshot
first — including the `demo/hello` tombstone with `deleted: true` — then
the live `demo/live` event triggered from the other node:

```
{
  "record": {
    "key": "demo/hello",
    "version": "AaHppjFfcACAAEAAAAAAAA==",
    "deleted": true
  }
}
{
  "record": {
    "key": "demo/wait",
    "version": "AaHppi5DcACAAEAAAAAAAA==",
    "value": "c3VwLWt2"
  }
}
{
  "record": {
    "key": "demo/live",
    "version": "AaHppjljcACAAIAAAAAAAA==",
    "value": "c3VwLWt2"
  }
}
```

### Check health

<!-- examples-verify -->
```bash
# The standard health endpoint — SERVING once the node's startup gate has
# opened, NOT_SERVING while it shuts down.
grpcurl -plaintext -d '{}' 127.0.0.1:28011 grpc.health.v1.Health/Check
```

```
{
  "status": "SERVING"
}
```

### Snapshot round-trip

Snapshots are named copies of a node's confirmed dump — a recovery point
you can restore by name. `SnapshotCreate` copies the current confirmed dump
under a name (it lands next to the dump file, in `supkv-ex-a.json.snapshots/`);
`SnapshotLoad` is a **full rollback**, not an additive replay: it replays the
named snapshot with fresh versions, deletes the live keys absent from the snapshot
with fresh tombstones, and those rollback tombstones replicate to
peers like any write, so the restored state wins by normal last-write-wins.
Both are gated like writes.

<!-- examples-verify -->
```bash
# Copy the confirmed dump under a name...
grpcurl -plaintext -d '{"name":"demo-snap"}' 127.0.0.1:28011 \
  supkvapi.v1.SupKVService/SnapshotCreate
# ...and replay it back as ordinary writes (empty responses: {} both times).
grpcurl -plaintext -d '{"name":"demo-snap"}' 127.0.0.1:28011 \
  supkvapi.v1.SupKVService/SnapshotLoad
```

```
{}
{}
```

### Stop the pair

<!-- examples-verify -->
```bash
# Graceful stop: SIGTERM runs the ordered shutdown — the node stops serving,
# finishes a final confirmed dump, then exits (exit code 0 on a clean stop).
kill "$(cat ex-a.pid)" "$(cat ex-b.pid)" 2>/dev/null || true
for i in $(seq 1 50); do
  if ! kill -0 "$(cat ex-a.pid)" 2>/dev/null && ! kill -0 "$(cat ex-b.pid)" 2>/dev/null; then break; fi
  sleep 0.2
done
# The unconditional proof: fail the block if either node is still alive.
ps -p "$(cat ex-a.pid)" >/dev/null 2>&1 && exit 1
ps -p "$(cat ex-b.pid)" >/dev/null 2>&1 && exit 1
echo "demo pair stopped: both nodes exited"
```

The dump files (`supkv-ex-a.json`, `supkv-ex-b.json`) remain in the working
directory — human-readable JSON you can inspect directly, and what each node
reloads on its next start.
