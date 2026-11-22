# sup-kv spec deviations

Statement-level registry of how sup-kv stands against the upstream Spec: one row per
statement ID from the 2026-11-19 v1.0 review, with an own-words summary, a resolution
status, and links to code, decisions and review findings. The registry derives from the
review of 2026-11-19 ([docs/reviews/2026-11-19-v1.0/REVIEW.md](reviews/2026-11-19-v1.0/REVIEW.md),
section 13 trace). Later phases only flip statuses as fixes land — they never add or
remove rows. Spec text is never quoted here; summaries are glosses in our own words.

Status vocabulary (fixed enum, machine-checkable; Phase 9 greps it for unclosed rows):

- `implemented` — the shipped v1.0 code satisfies the statement as amended;
- `fixed-in-v1.1(<REQ-ID>)` — live gap; the named v1.1 requirement closes it;
- `accepted-deviation(<ADR|14.x>)` — the departure is accepted; the reference names the
  ADR or owner decision recording it;
- `consumer-zone` — the statement describes the consumer's side of the boundary, not sup-kv;
- `superseded-by-amendment` — the 2026-11-19 Spec amendment removed or rewrote the
  requirement, so the code no longer deviates from it;
- `deferred-v2(<REQ-ID>)` — deliberately pushed past v1.1; tracked under the named v2
  requirement.

Review finding S-12 (the Spec was invisible to the repository and to acceptance) closes
with this registry itself plus the loud historical banner on [docs/IDEA.md](IDEA.md).
S-12 has no Spec statement ID, so it is recorded here in the preamble rather than as a
table row. ADR numbers 0001–0011 under `docs/adr/` record owner decisions 14.1–14.11, and
ADR 0012 ([docs/adr/0012-equal-version-tiebreak.md](adr/0012-equal-version-tiebreak.md))
records the P-07 equal-version tie-break; `REVIEW §7` and `G-xx` refer to the review's
deviation and gap findings; `L<n>` refers to line numbers in the Spec edition the review
audited.

### Data model

| ID | Summary (own words) | Status | Links |
|---|---|---|---|
| S-DM-01 | All records live in one in-process Go map guarded by a single reader-writer mutex | implemented | `internal/app/storage/store.go:83-89` |
| S-DM-02 | Persistence writes the map to disk; the one-JSON-document requirement was rewritten by the 2026-11-19 amendment into the shipped JSON Lines dump format | superseded-by-amendment | `internal/app/persist/codec.go:62-85`; REVIEW §13 |
| S-DM-03 | Keys are plain strings and a per-service prefix is only advice; the API additionally rejects the empty key, and key length is capped at 1 024 bytes (XPORT-03) | implemented | `internal/app/storage/store.go:20-27` (`MaxKeyBytes`); `internal/grpcapi/server.go:151,196,304` (API check); `internal/app/repl/core.go:815` (receive check); `cmd/sup-kv/main.go:439` (startup datagram ceiling); `internal/udprepl/maxdatagram.go` (ceiling oracle); REVIEW G-11 |
| S-DM-04 | Every write stamps a UUIDv7 record version, and on merge the greater version takes the key | implemented | `internal/app/storage/store.go:252-273` (`Set`/`write` stamping); `internal/app/storage/store.go:306-328` (`MergeChanged`); `internal/app/storage/store.go:376-415` (`join`); `internal/app/storage/merge_test.go#TestMerge_equalVersionContentLaw`; ADR 0012 |
| S-DM-05 | Record payload is raw bytes, emptied by deletion and capped at 60 KiB | implemented | `internal/app/storage/store.go:12-24` (`ValueMaxBytes`, the one limit); `internal/app/storage/store.go:125-135` (`Config.ValueLimit`); `internal/app/storage/store.go:148-152` (`Store.ValueMaxBytes`); `internal/app/storage/store.go:252-260,280-283` (Set/SetIfAbsent reject over it) |
| S-DM-06 | Each record carries a Deleted flag serving as a permanent tombstone | implemented | `internal/app/storage/store.go:55-60` |
| S-DM-07 | A CRC32 over key, version, payload and flag protects each record and is re-checked when a dump is read back | implemented | `internal/app/storage/store.go:131-137`; `internal/app/persist/codec.go:142` (the wire copy of the checksum is carried but re-derived at install, matching the Spec's file-verification scope) |
| S-DM-08 | The store is an in-repo LWW-Element-Map: per key the higher version wins, and merging is order-insensitive (commutative, associative, idempotent) | implemented | `internal/app/storage/store.go:306-328` (`MergeChanged`), `internal/app/storage/store.go:376-415` (`join`), `internal/app/storage/store.go:178-186` (`RecordsEqual`); `internal/app/storage/merge_test.go#TestMerge_laws`; `internal/app/storage/prop_small_space_test.go#TestProp_mergeSmallSpace`; ADR 0012 |
| S-DM-09 | The CRDT is written in-project, no library | implemented | `internal/app/storage` |

### Record versions

| ID | Summary (own words) | Status | Links |
|---|---|---|---|
| S-HLC-01 | Clock state is a pair: logical milliseconds and a counter | implemented | `internal/app/clock/clock.go:43-49` |
| S-HLC-02 | Every local write or delete ticks the clock: jump to wall time when it moved forward, otherwise bump the counter | implemented | `internal/app/clock/clock.go:87-110`; `internal/app/storage/store.go:176` |
| S-HLC-03 | Replication payloads drive the clock through the four-branch update rule on install | implemented | `internal/app/clock/clock.go:119-149`; `internal/app/storage/store.go:226-227` (also applied once per dumped record at startup; skipped when the store rejects an oversize record) |
| S-HLC-04 | Each non-empty DataSync part feeds the clock one update taken from the part's highest version, after header validation and before any entry comparison | implemented | `internal/app/repl/core.go:1510-1543` (`handleDataSync`, `maxEntryVersion`); `internal/app/storage/store.go:295-299` (`Observe`); `internal/app/repl/listedkeys_internal_test.go#TestSync_observePerPart`; REVIEW §7 S-08; G-03 |
| S-HLC-05 | At startup the clock is seeded from the dump's highest logical time and counter, so a restarted node keeps issuing dominant versions | implemented | `internal/app/persist/codec.go:168-179`; `internal/app/persist/engine.go:145-146`; `internal/app/clock/clock.go:79-81` (computed from the accepted generation's surviving records) |
| S-HLC-06 | Counter overflow advances the logical millisecond instead of wrapping, keeping versions monotonic | implemented | `internal/app/clock/clock.go:100-106,137-146` (extra: saturates at the 48-bit ceiling, repeating the top reading) |
| S-HLC-07 | Version packing leaves the low bits zero where the Spec asks for random filler; the documented zero filler keeps uniqueness and byte order intact | accepted-deviation(02-RESEARCH A2) | `internal/app/version/version.go:19-22,38-47`; REVIEW P-07 (recorded v1.0 research assumption A2); `docs/adr/0012-equal-version-tiebreak.md` (the recorded equal-version tie-break, which keeps the zero filler) |
| S-HLC-08 | Comparing packed versions byte-wise reproduces the logical-time, counter, node-id order | implemented | `internal/app/version/version.go:63-65` |
| S-HLC-09 | Two writes on the same node can never collide on one version | implemented | `internal/app/clock/clock.go:87-110`; `internal/app/version/version.go:38-47` (documented exception: versions repeat at the 48-bit ceiling) |
| S-HLC-10 | A version issued after seeing a neighbour's record dominates it, whether the version arrived in a replicated record or only in a DataSync listing part (each part's maximum reaches the clock) | implemented | `internal/app/storage/store.go:295-299` (`Observe`); `internal/app/repl/core.go:1510-1543` (per-part absorption); `internal/app/repl/listedkeys_internal_test.go#TestSync_observePerPart`; REVIEW G-03 (the same omission as S-HLC-04) |
| S-HLC-11 | The generator stays monotonic on one node under arbitrary wall-clock movement | implemented | `internal/app/clock/clock.go:87-149`; simulator clock-jump scenario |
| S-HLC-12 | Logical time never trails wall time when a version is issued, and lags the wall only by the inter-node skew | implemented | `internal/app/clock/clock.go:92-98,125` |
| S-HLC-13 | A received version far ahead of local time should warn in the log and raise a metric; today a differently-scoped counter fires and no log line exists | fixed-in-v1.1(OBSV-05) | `internal/app/repl/core.go:532-534`; `internal/app/clock/clock.go:159-176`; REVIEW §7 S-06; G-07 |
| S-HLC-14 | Concurrent writes to one key settle identically on every replica, ordered by logical time | implemented | `internal/app/storage/store.go:240-249` |
| S-HLC-15 | Time synchronisation is recommended practice, never a correctness requirement | implemented | `docs/OPERATIONS.md:87-108` |

### Exchange protocol — common rules, Hello, DataReplication, Ack, DataRequest, DataSync

| ID | Summary (own words) | Status | Links |
|---|---|---|---|
| S-PROTO-01 | One UDP datagram carries one protobuf envelope holding protocol version, node id, run id, a CRC and exactly one message body | implemented | `internal/udprepl/replproto/supkv/v1/supkv.proto:17-35`; `internal/udprepl/repl.go:702-761` (the CRC covers the body bytes only) |
| S-PROTO-02 | A replication datagram must fit value plus service fields under the UDP size ceiling; key length is unbounded and the value flag is never checked against the ceiling, so an oversized record can become permanently unreplicable | fixed-in-v1.1(XPORT-03) | `internal/app/storage/store.go:160,220`; `internal/udprepl/repl.go:50,555-560`; REVIEW G-11 |
| S-PROTO-03 | Everything other than DataReplication is capped at the sync chunk size in one frame; Ack and DataRequest keys go unchecked, and one oversized entry ships alone as its own chunk (recorded intent) | fixed-in-v1.1(XPORT-03) | `internal/app/repl/core.go:87,95,792-815`; REVIEW G-11 |
| S-PROTO-04 | Datagrams failing the CRC, carrying our own node id or an unsupported protocol version are dropped per class; the latter two drops are counted but logged only at debug, invisible at the default level | fixed-in-v1.1(OBSV-06) | `internal/app/repl/core.go:377-404`; `internal/udprepl/repl.go:262-264`; REVIEW §7 S-07; G-10 |
| S-PROTO-05 | Which messages broadcast and which unicast; the restart-triggered listing went out unicast, and the amendment aligned the address rules with the implementation | superseded-by-amendment | `internal/app/repl/core.go:491-493,672`; REVIEW §13 |
| S-PROTO-06 | Node id, UDP bind address, broadcast address and dump path are required launch parameters | implemented | `cmd/sup-kv/main.go:296-302,691-706` |
| S-PROTO-07 | Hello goes out on the hello interval carrying no extra fields | implemented | `internal/app/repl/core.go:725-728`; `internal/udprepl/replproto/supkv/v1/supkv.proto:40` |
| S-PROTO-08 | A Hello echoing our own node id is a configuration error: warned and ignored; the warning repeats for every datagram from the duplicate id, while a same-run-id echo is dropped silently | implemented | `internal/app/repl/core.go:391-404`; recorded 04-CONTEXT D-02 |
| S-PROTO-09 | The datagram's source address is adopted as the neighbour's address | implemented | `internal/app/repl/core.go:489` |
| S-PROTO-10 | A neighbour whose run id changed or who is seen for the first time must get an immediate full sync; only the changed-run-id half exists — a first-seen peer gets nothing until the next periodic round | fixed-in-v1.1(PROTO-04) | `internal/app/repl/core.go:486-506`; REVIEW §7 S-08; G-02 |
| S-PROTO-11 | One peer-table entry for a two-machine cluster; the code generalised to N peers with per-peer acknowledgement, which the amendment adopted | superseded-by-amendment | `internal/app/repl/core.go:268,288`; REVIEW §13 |
| S-PROTO-12 | RunID regenerates at every start; the code draws 16 random bytes rather than a UUIDv7, and the amendment adopted the random-bytes form | superseded-by-amendment | `internal/app/repl/core.go:195-199`; REVIEW §13 |
| S-PROTO-13 | DataReplication carries the whole record: key, version, payload, deleted flag, checksum | implemented | `internal/udprepl/replproto/supkv/v1/supkv.proto:44-50`; `internal/udprepl/repl.go:771-778` |
| S-PROTO-14 | Every local change should broadcast and unacked records retransmit; dump-failed writes and snapshot-load records are never broadcast and never tracked as unacked | fixed-in-v1.1(PROTO-08) | `internal/grpcapi/server.go:147-149,243-252`; REVIEW §7 S-02; G-05 |
| S-PROTO-15 | Fresher-local unicast replies fire when answering DataRequest and comparing DataSync, but there is no such branch while processing a DataReplication | fixed-in-v1.1(PROTO-03) | `internal/app/repl/core.go:523-543,605-641`; REVIEW §7 S-08; G-04 |
| S-PROTO-16 | An Ack goes only for an applied or already-equal version; the code also acks strictly older versions, telling the sender its stale record landed | fixed-in-v1.1(PROTO-03) | `internal/app/repl/core.go:542`; REVIEW §7 S-08; G-04 |
| S-PROTO-17 | A DataRequest names one key and is answered unicast with the local record, tombstones included; absent keys draw no reply | implemented | `internal/app/repl/core.go:605-611`; `internal/udprepl/replproto/supkv/v1/supkv.proto:60-62` |
| S-PROTO-18 | DataSync carries SyncID, ChunkIndex and ChunkCount next to the entries; the receiver accumulates the parts of one (node, run, sync_id) round and marks the listing complete only when every index 1..ChunkCount has arrived | implemented | `internal/udprepl/replproto/supkv/v1/supkv.proto:66-74` (the three header fields); `internal/app/repl/core.go:40` (`SupportedProtoVersion`); `internal/app/repl/core.go:1510-1615` (header validation and completeness accumulation); `internal/app/repl/sync_test.go#TestSync_partialListingKeepsGateClosed`; `internal/udprepl/syncsize_internal_test.go#TestSync_chunkWireBudget`; `internal/udprepl/repl_test.go#TestMirror_listingComplete`; REVIEW §7 S-08; G-03 |
| S-PROTO-19 | The full key-to-version listing, tombstones included, is sorted by key and cut into chunks under the byte budget | implemented | `internal/app/repl/core.go:771-778,792-815` (extra: an empty store announces itself with one empty chunk) |
| S-PROTO-20 | Each chunk stands alone; a lost part only postpones its keys to the next round | implemented | `internal/app/repl/core.go:631-644` |

### Algorithms — Startup

| ID | Summary (own words) | Status | Links |
|---|---|---|---|
| S-ALG-ST-01 | Startup reads the dump, discards checksum-failed records with a log line, and heals them from the neighbour | implemented | `internal/app/persist/engine.go:132-141`; `internal/app/persist/codec.go:117-121,142`; `cmd/sup-kv/main.go:420-422` (one aggregate INFO line with a count) |
| S-ALG-ST-02 | The clock is initialised from the loaded state at startup | implemented | `internal/app/persist/engine.go:145-146` |
| S-ALG-ST-03 | A fresh RunID is generated at startup | implemented | `internal/udprepl/repl.go:236` |
| S-ALG-ST-04 | The UDP socket opens, the first Hello goes out, and the timers start | implemented | `internal/udprepl/repl.go:223,270-291,308-309`; `cmd/sup-kv/main.go:436-451` |
| S-ALG-ST-05 | Read traffic is answered at once, even before the startup gate opens | implemented | `internal/grpcapi/server.go:102-111`; `internal/grpcapi/gate.go:102-110` |
| S-ALG-ST-06 | The whole interface — reads and writes — stays unavailable until the neighbour's complete key listing is processed or the applicable startup gate elapses; the gate opens only on a complete listing (all parts of a round), never on a first chunk | implemented | `internal/grpcapi/gate.go:120-157` (`gated`, `RunGate`); `internal/grpcapi/server.go:124-126,180-182,239-241,280-282,305-307,329-331` (Get/Set/Delete/List and the snapshot operations closed); `internal/grpcapi/watch.go:369-371` (Watch closed); `internal/app/repl/core.go:1560-1615` (completeness accumulation); `internal/app/repl/sync_test.go#TestSync_partialListingKeepsGateClosed`; `internal/grpcapi/gate_test.go#TestGate_readsAndWritesUnavailable`; REVIEW §7 S-08; G-02, G-03 |

### Algorithms — Read, Create and Update, Delete

| ID | Summary (own words) | Status | Links |
|---|---|---|---|
| S-ALG-RD-01 | Reads return the local value, or NotFound for absent and deleted keys | implemented | `internal/grpcapi/server.go:102-111`; `internal/app/storage/store.go:141-153` |
| S-ALG-CU-01 | A write first checks the size limit; over it the client gets InvalidArgument | implemented | `internal/app/storage/store.go:160-162`; `internal/grpcapi/server.go:297-298` |
| S-ALG-CU-02 | The write ticks the clock and uses the result as the record version | implemented | `internal/app/storage/store.go:176-177` |
| S-ALG-CU-03 | The stamped record installs directly; it dominates by construction because the clock update and install share the store lock | implemented | `internal/app/storage/store.go:178,223-227,259-269` |
| S-ALG-CU-04 | A dump is scheduled and the client waits for a dump containing the record to complete | implemented | `internal/app/persist/scheduler.go:109-147,221-238` (`Run`'s debounce/deadline loop and `runDump`); `internal/app/persist/engine.go:190-220` (the write forwards block on `await`); `internal/app/persist/confirm.go:98-130` (`await`) (scheduling policy follows S-PERS-04, adopted by the amendment) |
| S-ALG-CU-05 | After confirmation the record broadcasts as DataReplication and joins the unacked set | implemented | `internal/grpcapi/server.go:150`; `internal/app/repl/core.go:689-709` (notification rides an async queue whose send errors are ignored during shutdown) |
| S-ALG-CU-06 | The client gets OK | implemented | `internal/grpcapi/server.go:151-156` (the response also carries the version) |
| S-ALG-CU-07 | A dump failure surfaces to the client as Unavailable | implemented | `internal/app/persist/confirm.go:229-250` (`failWaiters` resolves every waiter with `ErrDumpFailed`); `internal/app/persist/engine.go:190-220` (the write forwards return that error); `internal/grpcapi/server.go:299-300` |
| S-ALG-CU-08 | A dump failure should raise a metric and log an error; the counter exists but the engine has no logger, so the log line never fires | fixed-in-v1.1(OBSV-06) | `internal/app/persist/scheduler.go:248-266` (`handleDumpFailure`: class counters and `OnDumpFailure`); `internal/app/persist/scheduler.go:296-302` (`recordDump`); `internal/metrics/metrics.go:405-445` (`dumpFamilies`); `cmd/sup-kv/main.go:586-588` (the wired ERROR log); REVIEW §7 S-07; G-10 |
| S-ALG-CU-09 | A dump-failed write stays in memory and replicates to the neighbour as unacked; today it does neither — peers only learn of it via a later periodic sync | fixed-in-v1.1(PROTO-08) | `internal/grpcapi/server.go:147-149`; `internal/app/persist/engine.go:288-291`; REVIEW §7 S-02; G-05 |
| S-ALG-CU-10 | Create-if-absent on a live key answers AlreadyExists; a deleted key blocks re-creation, which the amendment adopted as Spec behaviour (decision 14.2) | superseded-by-amendment | `internal/app/storage/store.go:187-198`; REVIEW §13 |
| S-ALG-CU-11 | Optional wait-for-replication waits for peer acknowledgements within a timeout and reports without changing the write result; the report can say acked for a version a peer discarded as stale | fixed-in-v1.1(PROTO-03) | `internal/grpcapi/wait.go:94-225`; `internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto:64-81`; REVIEW §7 S-08; G-04 |
| S-ALG-DEL-01 | Deleting an absent or already deleted key answers NotFound | implemented | `internal/app/storage/store.go:203-208`; `internal/grpcapi/server.go:293-294` |
| S-ALG-DEL-02 | A real delete runs the write path with an empty payload and the tombstone flag set | implemented | `internal/app/storage/store.go:209`; `internal/grpcapi/server.go:171-189` |
| S-ALG-DEL-03 | Tombstones are never garbage-collected | implemented | `internal/app/storage/store.go` (no removal path exists) |

### Algorithms — Replication, Retransmission, Recovery

| ID | Summary (own words) | Status | Links |
|---|---|---|---|
| S-ALG-REP-01 | A replication payload updates the clock and is compared against the local record under one lock | implemented | `internal/app/repl/core.go:524`; `internal/app/storage/store.go:225-227` |
| S-ALG-REP-02 | An absent-or-greater payload installs, schedules a dump, notifies watchers, acks and clears the pending entry | implemented | `internal/app/repl/core.go:539-542`; `internal/app/storage/store.go:237,243,268`; `internal/app/persist/facade.go:40-50` (`ApplyRemote` dirties only on a changed merge) |
| S-ALG-REP-03 | An equal version is acknowledged without triggering a dump when it carries identical content; with differing content the store's deterministic (version, content) tie-break decides — an incoming win Acks, a stored win is answered with the stored record and no Ack | implemented | `internal/app/repl/core.go:1487-1530` (`applyRemote`'s equal-version branch); `internal/app/persist/facade.go:15-49` (`ApplyRemote` dirties only on a changed merge); `internal/app/repl/retransmit_test.go#TestReplication_equalVersionContentLaw`; `docs/adr/0012-equal-version-tiebreak.md` |
| S-ALG-REP-04 | A strictly older payload should draw a unicast reply with the local record and no Ack; the branch does not exist and stale payloads are acked unconditionally | fixed-in-v1.1(PROTO-03) | `internal/app/repl/core.go:523-543`; REVIEW §7 S-08; G-04 |
| S-ALG-REP-05 | The Ack leaves right after the memory write, with no wait for the dump | implemented | `internal/app/repl/core.go:524-542` (extra: a record the store refuses as oversize is logged and not acked) |
| S-ALG-RT-01 | The unacked set tracks key-version pairs with their next send time | implemented | `internal/app/repl/core.go:265-273,289` |
| S-ALG-RT-02 | Due records resend on the retransmit interval with exponential backoff from ack_timeout up to the ceiling | implemented | `internal/app/repl/core.go:872-887,935-944` (retries fire at their exact due time; the interval only caps the driver's sleep) |
| S-ALG-RT-03 | Entries leave the set when acked at the same version or when superseded by anything fresher, local or received | implemented | `internal/app/repl/core.go:539-541,567-579,700` (N-peer rule: every peer known at write time must ack) |
| S-ALG-RT-04 | With no known neighbour, retransmission suspends and resumes once one appears | implemented | `internal/app/repl/core.go:499-506,831-845,873-875` (resume flushes the whole pending set with the ladder reset) |
| S-ALG-RT-05 | The set never exceeds the key count — one entry per key | implemented | `internal/app/repl/core.go:289,700` |
| S-ALG-RC-01 | A full sync runs on the sync-interval timer | implemented | `internal/app/repl/core.go:751-759` |
| S-ALG-RC-02 | A Hello with a changed run id triggers an immediate full sync toward that peer | implemented | `internal/app/repl/core.go:486-495` (unicast, only while the peer is still in the table) |
| S-ALG-RC-03 | A full sync goes out at startup, right after the first Hello | implemented | `internal/app/repl/core.go:724-729` |
| S-ALG-RC-04 | The listing collects every pair, tombstones included, and broadcasts as DataSync parts | implemented | `internal/app/repl/core.go:756-758,771-778` |
| S-ALG-RC-05 | Each received non-empty sync part advances the clock by the part's maximum version, after header validation and before any entry comparison | implemented | `internal/app/repl/core.go:1510-1543` (`handleDataSync`, `maxEntryVersion`); `internal/app/storage/store.go:295-299` (`Observe`); `internal/app/repl/listedkeys_internal_test.go#TestSync_observePerPart`; REVIEW §7 S-08; G-03 (the same omission as S-HLC-04) |
| S-ALG-RC-06 | Per listed pair: request when missing or behind, unicast the record when ahead, stay silent on equality | implemented | `internal/app/repl/core.go:634-643` |
| S-ALG-RC-07 | Both sides run their sync independently of each other | implemented | `internal/app/repl/core.go:751-759` |
| S-ALG-RC-08 | Sync answers queue up and drain no faster than the response rate | implemented | `internal/app/repl/core.go:342,652-678` (one global FIFO; out-of-range rates clamp to 500) |
| S-ALG-RC-09 | Whatever a sync round cannot send is caught by the next; the reply queue is never trimmed, so each round re-appends the keys still outstanding | fixed-in-v1.1(XPORT-07) | `internal/app/repl/core.go:634-643,652-657`; REVIEW G-12 |
| S-ALG-RC-10 | State converges within two sync intervals plus reply time after reconnection; proven in the simulator at small key counts, the stand shows the bound failing at 10k keys | fixed-in-v1.1(XPORT-02) | `internal/app/internal/simnet/simnet.go:418`; REVIEW §13 (stand findings R-01, R-02) |

### Exchange scenarios

| ID | Summary (own words) | Status | Links |
|---|---|---|---|
| S-SCN-01 | The write-and-replicate scenario runs end to end as specified | implemented | `internal/grpcapi/server.go:131-157`; `internal/app/repl/core.go:523-579,689-709` |
| S-SCN-02 | A lost datagram is recovered by retransmission inside the backoff ceiling | implemented | `internal/app/repl/core.go:872-887` |
| S-SCN-03 | Concurrent writes of one key converge via the winner's retransmission, but the Spec's unicast-reply step is absent (and its conflict-journal step was removed by decision 14.4) | fixed-in-v1.1(PROTO-03) | `internal/app/repl/core.go:523-543`; REVIEW §7 S-08; G-04, G-08 |
| S-SCN-04 | The restart scenario's neighbour sync only fires while the restarted node is still remembered; past the peer timeout the immediate sync never happens | fixed-in-v1.1(PROTO-04) | `internal/app/repl/core.go:486-495,831-845`; REVIEW §7 S-08; G-02 |

### Persistence

| ID | Summary (own words) | Status | Links |
|---|---|---|---|
| S-PERS-01 | The dump is a JSON Lines file — one format-version line, then one record per line with hex versions; the amendment rewrote the dump-format requirement to match | superseded-by-amendment | `internal/app/persist/codec.go:39-85`; REVIEW §13 |
| S-PERS-02 | Dump files are readable by a human in a standard editor | implemented | `internal/app/persist/codec.go:62-85` (files 0600, directories 0700) |
| S-PERS-03 | Dumps write atomically — temp file, fsync, rename — keeping the previous generation as .prev | implemented | `internal/persistfile/fsops.go:85-163` (also fsyncs the parent directory; the darwin F_FULLFSYNC durability path, with EINTR retry and ENOTSUP fallback, is the standard library's `os.File.Sync`) |
| S-PERS-04 | Dump scheduling is a trailing quiet window plus a hard cap (dump_max_wait), not a minimal pause between dumps; the amendment adopted this policy | superseded-by-amendment | `internal/app/persist/engine.go:180-245`; REVIEW §13 (decision 14.3) |
| S-PERS-05 | A client response only lands after a dump containing the record completed, so confirmed writes are always on disk | implemented | `internal/app/persist/engine.go:190-220` (the write forwards block on `await`); `internal/app/persist/confirm.go:98-130,207-222` (`await`, `confirmWaiters`); `internal/app/persist/scheduler.go:221-238` (`runDump` captures before `Records` and confirms after the save); `internal/persistfile/dumpfile.go:136-176` (exception with a raised value limit: see S-PAR-05) |
| S-PERS-06 | Crash durability is promised only for dump-confirmed records; anything not yet dumped may vanish locally | implemented | — |
| S-PERS-07 | Disk load stays bounded by dump size over the debounce window, since dumps are serialised and rarer than the Spec allows | implemented | `internal/app/persist/engine.go:180-206` |
| S-PERS-08 | If the main dump is unreadable the previous generation loads; the load reports that as `port.LoadBackup` and the binary logs exactly one WARN naming the main generation's path, cause and preservation error | implemented | `internal/persistfile/dumpfile.go:127-175` (`Load`'s main-then-.prev chain); `internal/persistfile/dumpfile.go:206-233` (`preserve`); `cmd/sup-kv/main.go:613-644` (the Backup WARN); `internal/stand/persist_test.go#TestStand_persistLoadBackup`; REVIEW O-01 |
| S-PERS-09 | With both generations unusable the node starts empty, logs exactly one ERROR with one attribute group per failed candidate and publishes the `Unusable` sample on `supkv_dump_load_total`, so a data-losing start is distinguishable from a first start | implemented | `internal/persistfile/dumpfile.go:127-175` (`Load`'s Unusable outcome); `internal/persistfile/dumpfile.go:206-233` (`preserve` keeps the evidence); `cmd/sup-kv/main.go:651-664` (`logUnusableGenerations`); `internal/metrics/metrics.go:223-247` (`loadFamilies`); REVIEW §7 S-07; G-10 |
| S-PERS-10 | Snapshots are named on-disk copies of the current dump, created on API request after a forced flush if dirty; the .prev-fallback edge cases close with PERSIST-04 | implemented | `internal/persistfile/snapshot.go:38-55`; `internal/app/persist/snapshot.go:36-42` (`Snapshot`); `internal/app/persist/engine.go:268-294` (the bounded flush that forces a covering dump); REVIEW G-13 |
| S-PERS-11 | Loading a snapshot replays every snapshot key as a fresh write or delete | implemented | `internal/app/persist/snapshot.go:89-115` (`Restore`); `internal/app/persist/snapshot.go:125-150,181-200` (`replaySnapshot`, `replayRecord`) |
| S-PERS-12 | Snapshot load must delete map keys the snapshot lacks — a full rollback; the code leaves them untouched and a test pins that behaviour | fixed-in-v1.1(APIS-07) | `internal/app/persist/snapshot.go:125-150,160-168` (`replaySnapshot` iterates only the snapshot's keys; `restoreChanges` compares only those); ADR 0001; REVIEW §7 S-01 |
| S-PERS-13 | A snapshot load reaches the peer through regular replication and wins by LWW; today nothing is broadcast and the peer learns via periodic sync | fixed-in-v1.1(PROTO-08) | `internal/grpcapi/server.go:243-252`; `internal/app/persist/engine.go:617-630`; REVIEW §7 S-02; G-05 |

### Storage semantics and client responses

| ID | Summary (own words) | Status | Links |
|---|---|---|---|
| S-SEM-01 | Per-key last-writer-wins: displaced versions vanish — never stored, never served | implemented | `internal/app/storage/store.go:234-250` |
| S-SEM-02 | A write issued after a read on any node dominates the value that was read | implemented | `internal/app/storage/store.go:225-227` (caveat for listed-but-unreceived data: S-HLC-10) |
| S-SEM-03 | Atomicity stops at the key boundary; no operation spans several keys | implemented | — |
| S-SEM-04 | The only conditional write is create-if-absent; nothing else compares-then-writes | implemented | `internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto:30-45` |
| S-SEM-05 | Deletes are not sticky: a concurrent write holding a higher version resurrects the key | implemented | `internal/app/storage/store.go:240-249` (no delete-wins policy) |
| S-SEM-06 | Tombstones are kept forever | implemented | `internal/app/storage/store.go` |
| S-SEM-07 | The conflict-journal requirement (recording the loser when a pending record is displaced) was dropped from the Spec by decision 14.4; only a counter remains, to be made honest under OBSV-03 | superseded-by-amendment | `internal/app/repl/core.go:531`; ADR 0004; REVIEW §13; G-08 |
| S-SEM-08 | Visibility bounds — one RTT normally, the ack ceiling under loss, two sync intervals after a partition; the bound stretches for dump-failed writes and snapshot loads until they replicate | fixed-in-v1.1(PROTO-08) | REVIEW §7 S-02; G-05 (bound shown by the simulator bound tests in `internal/app/repl/sim_bound_test.go`) |
| S-SEM-09 | Watch subscribers receive the same notification for a change whether it arrived locally or by replication | implemented | `internal/grpcapi/watch.go:213-219,290-301`; `internal/app/storage/store.go:268` (local changes are announced at install, before the dump confirms them) |
| S-SEM-10 | A write or delete that landed and is on disk answers OK | implemented | `internal/grpcapi/server.go:131-189` |
| S-SEM-11 | Reads and deletes of absent or deleted keys answer NotFound | implemented | `internal/grpcapi/server.go:293-294` |
| S-SEM-12 | Create-if-absent over a live key answers AlreadyExists | implemented | `internal/grpcapi/server.go:295-296` (also over a tombstoned key: S-ALG-CU-10) |
| S-SEM-13 | Values over the size limit answer InvalidArgument | implemented | `internal/grpcapi/server.go:297-298` |
| S-SEM-14 | During the startup gate or after a dump failure, writes answer Unavailable | implemented | `internal/grpcapi/gate.go:85-90`; `internal/grpcapi/server.go:299-302` |

### Failure behavior

| ID | Summary (own words) | Status | Links |
|---|---|---|---|
| S-FAIL-01 | One lost replication datagram returns via retransmission with a growing pause, inside the ack ceiling | implemented | `internal/app/repl/core.go:872-887` |
| S-FAIL-02 | A lost Ack is recovered by retransmission; the neighbour acks the equal version | implemented | `internal/app/repl/core.go:542` |
| S-FAIL-03 | Duplicates, reordering and own broadcast echo are absorbed; stale payloads should draw one unicast reply but are acked silently instead | fixed-in-v1.1(PROTO-03) | `internal/app/repl/core.go:391-396,542`; REVIEW G-04 |
| S-FAIL-04 | Short partitions heal with both sides converged | implemented | `internal/app/repl/core.go:831-845,872-887` |
| S-FAIL-05 | Long partitions settle on the greater version; the conflict-journal clause was removed from the Spec by decision 14.4 | superseded-by-amendment | `internal/app/storage/store.go:240-249`; ADR 0004; REVIEW §13 |
| S-FAIL-06 | One-way connectivity causes no message avalanche and heals within two sync intervals | implemented | `internal/app/repl/core.go:872-887` |
| S-FAIL-07 | A restart converges in seconds when the neighbour syncs the new run id; past the peer timeout the neighbour forgets the node and the wait stretches to the next periodic round | fixed-in-v1.1(PROTO-04) | `internal/app/repl/core.go:483-507`; `internal/app/persist/engine.go:124-150`; REVIEW §7 S-08; G-02 |
| S-FAIL-08 | A long-down node receives records and tombstones through periodic sync | implemented | `internal/app/repl/core.go:631-644,751-759` |
| S-FAIL-09 | An empty node pulls everything from the neighbour rate-limited; the documented recovery bound holds only for small key counts on the v1.0 stand | fixed-in-v1.1(XPORT-01) | `internal/app/repl/core.go:631-644,668-678`; REVIEW §13 (stand finding R-01) |
| S-FAIL-10 | After both nodes go down, each restarts from its own dump with confirmed records intact | implemented | `internal/app/persist/engine.go:124-150` |
| S-FAIL-11 | Dump corruption drops bad-checksum records, falls back to .prev, and starts empty when both generations are gone | implemented | `internal/app/persist/codec.go:132-151`; `internal/persistfile/dumpfile.go:74-86` |
| S-FAIL-12 | Wall-clock changes on a running node never break version monotonicity | implemented | `internal/app/clock/clock.go:87-110` |
| S-FAIL-13 | A wall-clock step back between restarts is absorbed by seeding the clock from the dump maximum | implemented | `internal/app/persist/engine.go:145-146`; `internal/app/clock/clock.go:79-81` |
| S-FAIL-14 | One node far ahead drags both clocks forward; the required log warning is missing and the counter measures a different quantity | fixed-in-v1.1(OBSV-05) | `internal/app/clock/clock.go:125-136`; `internal/app/repl/core.go:532-534`; REVIEW §7 S-06; G-07 |
| S-FAIL-15 | A 1000 writes/s burst keeps the dump cadence and one datagram per write; the Spec's client-latency clause was removed by the amendment, and the burst bound holds only on hot keys today | fixed-in-v1.1(PERF-02) | `internal/app/persist/engine.go:227-245`; REVIEW §13 (stand findings R-02, PERF-02); G-06 |
| S-FAIL-16 | A duplicated node id is logged as a configuration error at the first Hello, and the stranger's datagrams are dropped | implemented | `internal/app/repl/core.go:397-403` |
| S-FAIL-17 | Incompatible protocol versions drop datagrams and the nodes keep working independently; the required log entry sits at debug level, invisible by default | fixed-in-v1.1(OBSV-06) | `internal/app/repl/core.go:386-390`; `internal/udprepl/repl.go:262-264`; REVIEW §7 S-07; G-10 |
| S-FAIL-18 | A foreign datagram on the segment cannot be detected — CRC32 is integrity, not authentication, under the accepted trust model | implemented | `internal/udprepl/repl.go:732-761`; `docs/OPERATIONS.md:8-29` |

### Availability and single point of failure

| ID | Summary (own words) | Status | Links |
|---|---|---|---|
| S-AVL-01 | Applications talk only to their local node; the peer is never on the request path | implemented | `cmd/sup-kv/main.go:132,303` (gRPC binds loopback by default) |
| S-AVL-02 | The node's own startup configuration comes only from static sources — flags and environment — never from the store | implemented | `cmd/sup-kv/main.go:275-322` |
| S-AVL-03 | The consumer's client library keeps the last read value on disk for outage fallback — client-side work, nothing needed on the sup-kv side beyond the public API | consumer-zone | ADR 0007; REVIEW §7 S-11; 14.7 |
| S-AVL-04 | Alone for a long time the node keeps serving, with unacked records bounded by the key count | implemented | `internal/app/repl/core.go:289,873-875` |

### Parameters

| ID | Summary (own words) | Status | Links |
|---|---|---|---|
| S-PAR-01 | node_id is a required launch parameter (16-bit; zero is rejected) | implemented | `cmd/sup-kv/main.go:294-296,693-695` |
| S-PAR-02 | bind_addr is a required launch parameter | implemented | `cmd/sup-kv/main.go:297-298,696-698` |
| S-PAR-03 | broadcast_addr is a required launch parameter | implemented | `cmd/sup-kv/main.go:299-300,699-701` |
| S-PAR-04 | dump_path is a required launch parameter | implemented | `cmd/sup-kv/main.go:301-302,702-704` |
| S-PAR-05 | The value limit is configurable; v1.0's dump decoder enforced a hard-wired 61440, so a raised limit yielded writes confirmed on disk and discarded at the next start. The limit now has one enforced source: `storage.ValueMaxBytes` is the default and a configured `--value-max-bytes` the effective limit at the decoder, the store and the binary's ceiling check | implemented | `internal/app/storage/store.go:12-24` (`ValueMaxBytes`); `internal/app/storage/store.go:125-135` (`Config.ValueLimit`); `internal/app/storage/store.go:148-152` (`Store.ValueMaxBytes`); `internal/app/persist/codec.go:236-240` (`Decode` at the configured limit); `internal/app/persist/loader.go:43-53` (`Open` decodes through the same helper); `internal/persistfile/red_value_limit_test.go#TestRed_P01_valueLimitEngine`; REVIEW G-11 (datagram-ceiling half: XPORT-03) |
| S-PAR-06 | sync_chunk_bytes defaults to 1400 and governs DataSync chunking | implemented | `cmd/sup-kv/main.go:136,313`; `internal/app/repl/core.go:56,155` |
| S-PAR-07 | hello_interval defaults to 1 s | implemented | `cmd/sup-kv/main.go:137,331`; `internal/app/repl/core.go:50` |
| S-PAR-08 | peer_timeout defaults to 5 s | implemented | `cmd/sup-kv/main.go:138,333`; `internal/app/repl/core.go:51,834` |
| S-PAR-09 | ack_timeout defaults to 200 ms | implemented | `cmd/sup-kv/main.go:139,335`; `internal/app/repl/core.go:52` |
| S-PAR-10 | ack_timeout_max defaults to 5 s | implemented | `cmd/sup-kv/main.go:140,337`; `internal/app/repl/core.go:53,883` |
| S-PAR-11 | retransmit_interval defaults to 100 ms | implemented | `cmd/sup-kv/main.go:141,339`; `internal/app/repl/core.go:54,936` |
| S-PAR-12 | sync_interval defaults to 30 s | implemented | `cmd/sup-kv/main.go:142,341`; `internal/app/repl/core.go:55,755` |
| S-PAR-13 | sync_response_rate defaults to 500 per second | implemented | `cmd/sup-kv/main.go:143,315`; `internal/app/repl/core.go:57,158-160,342` (out-of-range values silently clamp to 500) |
| S-PAR-14 | startup_gate defaults to 1 s | implemented | `cmd/sup-kv/main.go:133,307`; `internal/grpcapi/gate.go:25,123-127` |
| S-PAR-15 | dump_debounce was reinterpreted as the quiet window after the last change, with dump_max_wait added; the amendment adopted the policy and its wording | superseded-by-amendment | `cmd/sup-kv/main.go:144,343`; `internal/app/persist/engine.go:21,236-240`; REVIEW §13 |
| S-PAR-16 | drift_warn defaults to 60 s; today the threshold watches local-versus-wall in both directions rather than received-version drift — semantics change with OBSV-05 | fixed-in-v1.1(OBSV-05) | `cmd/sup-kv/main.go:146,347`; `internal/app/clock/clock.go:159-176`; REVIEW G-07 |

### Impact

| ID | Summary (own words) | Status | Links |
|---|---|---|---|
| S-IMP-01 | The API gains create-if-absent | implemented | `internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto:67`; `internal/grpcapi/server.go:142-143` |
| S-IMP-02 | The API gains optional wait-for-replication | implemented | `internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto:68-69,87-88`; `internal/grpcapi/wait.go` |
| S-IMP-03 | Subscribers learn about replicated changes; sup-kv exposes the Watch stream and the consumer bridges it | implemented | `internal/grpcapi/watch.go:364-398`; `internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto:40` |
| S-IMP-04 | Mapping sup-kv snapshot operations onto the consumer's existing snapshot proto is consumer-side work, tracked as a v2 requirement | deferred-v2(SNAP-01) | REVIEW §7 S-11; v2 backlog (SNAP-01) |
| S-IMP-05 | Embedding the storage inside the consumer's config-manager; sup-kv ships as a separate binary and the amendment adopted the standalone program | superseded-by-amendment | `cmd/sup-kv`; REVIEW §13 |
| S-IMP-06 | The client library's on-disk cache of the last read value is consumer-side work | consumer-zone | ADR 0007; REVIEW §7 S-11; 14.7 |
| S-IMP-07 | Deployment needs the UDP port, broadcast address, node id and paths, with time sync recommended | implemented | `cmd/sup-kv/main.go:292-348`; `docs/OPERATIONS.md:87-108` |
| S-IMP-08 | All seven required metric families exist; the endpoint stays off unless enabled (decision 14.8), and the conflict and drift counters still measure something other than the Spec's definitions | accepted-deviation(ADR 0008) | `internal/metrics/metrics.go:214-317`; `cmd/sup-kv/main.go:472,527-537`; REVIEW O-04 (counter honesty: OBSV-03, OBSV-05) |
| S-IMP-09 | Deterministic two-node simulation over a lossy channel with failure scenarios as tests; the simulator covers the replication core, not persistence or the API layer | implemented | `internal/app/internal/simnet/simnet.go`; chaos and scenario suites in `internal/app/repl/sim_*_test.go` |
| S-IMP-10 | Fault-injection coverage of disk and network failures belongs to the consumer's tooling | consumer-zone | ADR 0007; REVIEW §7 S-11 |
| S-IMP-11 | The channel between nodes is trusted; no authentication on the replication segment or the gRPC API — accepted limitation, documented future work | implemented | `docs/OPERATIONS.md:8-29` |
| S-IMP-12 | Designed for one pair; generalised to N nodes on one broadcast segment, which the amendment adopted | superseded-by-amendment | `internal/app/repl/core.go:268,288`; REVIEW §13 |
| S-IMP-13 | The consumer's stage-3 config service is built on top of this storage — integration work outside sup-kv | consumer-zone | ADR 0007; REVIEW §7 S-11; 14.7 |
