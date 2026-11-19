# sup-kv v1.0 — stand results (raw)

Date: 2026-11-19. Binary: `sup-kv` built from `master` d6b2a75 (`CGO_ENABLED=0 GOOS=linux GOARCH=arm64`, go1.27.4).
Stand: Docker (OrbStack 29.8.2, linux/arm64 VM on an Apple-silicon host, 16 vCPU, 16 GiB), `alpine:3.20`
containers, one user-defined bridge network as the broadcast segment, dump on a named Docker volume
(virtual disk — fsync cost is NOT representative of target hardware). All parameters at defaults unless
stated. Two stands were used: `supkv-lab` (172.31.77.0/24) and `supkv-lab2` (172.31.78.0/24);
see `docker-compose.yml`, `g.sh`, `src-cmd/loadgen`, `src-cmd/durcheck`.

Latencies are milliseconds. "open loop" = fixed arrival rate, latency measured from the scheduled send
time (queueing included). "closed loop" = N workers, each issuing the next request when the previous returns.

## 1. Functional scenarios

| # | Scenario | Observed |
|---|----------|----------|
| F1 | Set on A, Get on B | OK, value visible on B |
| F2 | Status codes | Get/Delete missing → `NotFound`; `create_if_absent` existing → `AlreadyExists`; empty key → `InvalidArgument`; 61441-byte value → `InvalidArgument`; 61440-byte value OK and replicated (fragmented datagram) |
| F2a | `create_if_absent` on a deleted key | `AlreadyExists` (spec: ordinary write) |
| F2c | Key of 70 000 chars, 1-byte value | `Set` → OK; `sendto: message too long` on every send; never on B (checked after 2 × `sync_interval`); after A restarts `unacked_records` = 0 and the DataSync chunk itself fails to send every 30 s |
| F3 | Startup gate, no peer | writes `Unavailable` ("startup gate: waiting for first sync round or startup…") for ~1 s, reads served, health `NOT_SERVING` → `SERVING` |
| F5 | Watch on B, prefix `w/` | local write, replicated write and replicated tombstone (`deleted:true`) all delivered; other prefixes filtered |
| F6 | `wait_for_replication`, peer alive | `acked:true, ackedCount:1, totalPeers:1` |
| F6a | `wait_for_replication`, peer partitioned > `peer_timeout` | `{"version":…, "acked": true}` (ackedCount 0, totalPeers 0) |
| F7 | Two-way partition 7 s (iptables), both sides write `p/k`, heal | converged ~1 s after heal; later write wins on both; `supkv_merges_conflicts_total` = 1 on BOTH nodes; no log line on either node |
| F8a | `kill -9` under 32-worker write load, restart alone (peer stopped) | 1536 confirmed writes, 1536 present after restart, 0 missing |
| F8b | Peer B returns after A was restarted in between | B needed 33 s to receive 1536 keys (1536 `data_request`); expected "seconds" |
| F8c | B returns after 9 s down (A not restarted), gate set to 30 s | 200 keys written meanwhile arrive in 0.2 s (pending flush); B's gate opens only at 14.5 s |
| F9 | Empty-disk B, A holds 10k × 1 KiB written while B was down (all pending on A) | B has 10 000 keys after 14.2 s; A then stays in a retransmission storm: `unacked_records` 8909 → 0 took ~14.5 min; `retransmitted_total` 949 552; A CPU ≈ 120 %; A UDP socket drops (`ss -uanm` `d…`) 880 300 at ~4 min and 1 044 161 at ~5.5 min; 7 keys that had to come through DataSync were still missing on B at 5.5 min, all present at 9.5 min |
| F9b | Empty-disk B, A has nothing pending (pure anti-entropy), 11 740 keys | B: 5932 keys at 43 s, 9192 at 68 s, 9892 at 99 s, 9902 at 119 s, 9908 from 160 s to 271 s (no progress over 4 rounds; at 271 s all 1832 missing keys are in deciles 6–10 of the sorted keyspace), 10 669 at 496 s, 11 532 at 888 s and 974 s, all 11 740 at 1015 s; B socket drops grow by ~250 per round; documented bound ≈ 2 × 30 s + 11 740 / 500 s ≈ 83 s |
| F10a | One record corrupted in main dump, peer alive | record discarded (`INFO … discarded 1 undecodable dumped records at startup`), healed from peer within 0.4 s |
| F10b | Main dump truncated mid-file, peer down | starts from `.prev`; the last confirmed write (only in main) is gone; nothing logged about the fallback |
| F10c | Both generations garbage | empty start; no log line; no metric; healed after peer returns |
| F11 | Disk full (120 k tmpfs) | `Unavailable: dump failed; write not confirmed`; `supkv_dumps_errors_total` grows; zero log lines; failed writes readable locally, NOT on peers until the next periodic sync (75 keys local / 4 on peers, equal ~50 s later); health stays `SERVING`; dump retried ~18 times/s |
| F12 | Third process with a duplicate `node_id` | WARN on every datagram on both duplicates, `duplicate_node_id` counter grows |
| F13 | Snapshot create → change/add/re-create keys → load | changed key restored, snapshot tombstone re-applied; key created after the snapshot is KEPT (spec: deleted); peer unchanged 0.7 s and 60 s after the load, converged only after its next periodic sync (A `data` counter 5 → 8); missing snapshot → `Internal: internal error: open /data/dump.json.snapshots/nope: no such file or directory` |
| F14 | Replaced node B (empty disk) seeds a default with `create_if_absent` as soon as its gate opens; A holds the user's value, fully acked | B's write succeeds; 3 s later BOTH nodes hold the default — the user's value is gone |
| F15 | `--value-max-bytes 64000`, Set 63 000 bytes, restart | Set OK; after restart the key is gone (`discarded 1 undecodable dumped records at startup`) |
| F16 | SIGTERM / SIGKILL | exit 0 with data on disk / exit 137 |
| F17 | `--bind-addr <node unicast IP>:7900` | node sees 0 peers and receives nothing written elsewhere; its own writes reach the others; no log. A second process on the same bind address starts without error |
| F18 | drift counter on a stand with ONE kernel clock | `supkv_clock_drift_warnings_total` = 1 on node A |

Not exercised on the stand: per-node clock skew/jumps (containers share the kernel clock), protocol-version
mismatch (one binary), power loss (process kill only), multi-day partitions.

## 2. Write latency and throughput (1 KiB values unless stated)

### Closed loop
| State | Workers | Throughput w/s | p50 | p90 | p99 | max | Notes |
|---|---|---|---|---|---|---|---|
| 1000 keys (dump 1.5 MB, ~14 ms) | 1 | 14 | 70.8 | 81.0 | 87.3 | 107 | floor 58 ms |
| same | 8 | 107 | 73.9 | 85.4 | 92.8 | 94 | |
| same | 64 | 896 | 70.8 | 79.6 | 99.2 | 120 | |
| same | 256 | 827 | 360 | 469 | 629 | 719 | |
| 10k keys (dump 15 MB, 40–70 ms) | 1 | 10 | 86.9 | 98.7 | 182 | 308 | 109 dumps/10 s, ~155 MiB/s written |
| same | 8 | 86 | 87.9 | 99.3 | 215 | 271 | ~154 MiB/s |
| same | 64 | 699 | 89.7 | 96.4 | 156 | 196 | ~157 MiB/s |
| 1000 keys × 60 KiB (dump 82 MB, 85–113 ms) | 1 | 5 | 175 | 223 | 283 | 335 | RSS 344 → 539 MiB per node |

### Open loop
| State | Rate w/s | Key spread | Achieved | p50 | p90 | p99 | max | Notes |
|---|---|---|---|---|---|---|---|---|
| 1000 keys | 20 | 100 hot | 19 | 119 | 219 | 430 | 580 | 100 dumps / 15 s |
| 1000 keys | 100 | 100 hot | 99 | 528 | 928 | 1019 | 1036 | 15 dumps / 15 s |
| 1000 keys | 500 | 100 hot | 497 | 532 | 931 | 1022 | 1042 | 15 dumps / 15 s |
| 1000 keys | 1000 | 100 hot | 994 | 540 | 936 | 1026 | 1062 | 15 dumps / 15 s |
| 1000 keys | 2000 | 100 hot | 1991 | 337 | 540 | 590 | 631 | 30 dumps / 15 s |
| 10k keys | 100 | 100 hot | 99 | 544 | 944 | 1038 | 1064 | ~14 MiB/s written |
| 10k keys | 1000 | 100 hot | 992 | 560 | 959 | 1054 | 1081 | |
| 10k keys | 1000 | all 10k existing keys | 992 | 557 | 956 | 1051 | 1081 | 12 617 retransmissions; 1397 unacked at the end, 675 still unacked 120 s later (both nodes healthy and idle) |
| growing | 1000 | new key per write | 465 | 778 | 6669 | 10 942 | 11 482 | A CPU 141 %; 42 s later 4327 unacked, 102 835 retransmissions, 277 835 socket drops |

Preload of 10 000 new keys: 942 w/s for the first 1000 (c=64); 209 w/s at c=16 with both nodes up;
278 w/s at c=128 with the peer down (mean latency ≈ 460 ms).

## 3. Reads
| Workers | Throughput ops/s | p50 | p90 | p99 | p99.9 |
|---|---|---|---|---|---|
| 1 | 14 161 | 0.036 | 0.068 | 0.87 | 4.1 |
| 64 | 46 484 | 0.575 | 4.2 | 7.6 | 10.8 |
Mixed 90 % Get / 10 % Set, 64 workers: 7550 ops/s, p50 0.48, p99 86.9.

## 4. Replication lag (Set start on A → Watch event on B)
| Rate w/s | Value | Loss on A egress | p50 | p90 | p99 | Not seen within run + 3 s |
|---|---|---|---|---|---|---|
| 20 | 1 KiB | 0 | 112 | 212 | 321 | 0 of 201 |
| 200 | 1 KiB | 0 | 545 | 941 | 1028 | 0 of 2001 |
| 2 | 60 KiB | 0 | 165 | 202 | 484 | 0 of 41 |
| 2 | 60 KiB | 0.5 % | 160 | 668 | 1774 | 0 of 41 (9 retransmissions) |
| 2 | 60 KiB | 2 % | 350 | 1558 | 2840 | 22 of 41 (127 retransmissions) |
| 2 | 60 KiB | 5 % | – | – | – | 41 of 41 (346 retransmissions) |
The lag tracks the confirmation latency: the DataReplication is broadcast after the dump is confirmed.

## 5. CPU profile during the retransmission storm (node A, 10 s)
`udprepl.(*Adapter).mirrorState` 52 % cumulative (map iteration, `slices.Sorted`, string sort),
`repl.(*Core).scanRetransmit` 20 %. File: `cpu-a-storm.pprof`.
