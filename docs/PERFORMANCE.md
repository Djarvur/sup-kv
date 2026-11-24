# Performance

sup-kv's measurement ledger: the write-latency model for both dump profiles, the standing
burst bound over distinct keys, disk-write amplification with its target-media gap, the memory
model with its measured points, the absent-peer write cost, and the one command that produced
every number here.

**Publication rule.** Every measured row below was produced by `make perf` on this host and is
copied beside the hardware block that run logged and the command that produced it. Numbers from
other hosts (including `docs/reviews/`) are not republished here as measurements. These rows are
this tree's: two consecutive `make perf` runs at commit `9cdc0f1` both exited 0, and every table
below is the **second** run (research A4: publish the calmer rerun, and record the load instead
of gating on it). Go's test binaries report their VCS revision as `vcs unknown`, so the measured
commit is recorded beside each block rather than inside it. On different hardware the block must
be replaced with that host's block — a number alone is not a measurement.

**The thresholds.** Every bound this ledger states lives in one const block,
`internal/testutil/perf.go` (D-06): no stored profiles, no self-pacing targets, no environment
parsing. `cmd/sup-kv/performance_test.go` formats each threshold literal from those constants and
asserts it appears here, so changing a threshold in Go without changing this ledger fails
`make test`. Where this document quotes a threshold, the constant is named.

## Write-latency model

Write latency is a property of the dump policy, not of the RPC path alone: a client write
returns only after a dump containing it completed (PERS-04), so the quiet window and the
full-dump cost both sit inside the measured span.

| Knob | Default profile | Fast-response profile |
|------|-----------------|-----------------------|
| `dump_debounce` | 50 ms (`defaultDumpDebounce`) | 1 ms |
| `dump_max_wait` | 1 s (`defaultDumpMaxWait`) | 50 ms |
| Dump | real-disk dump: the production `persistfile` store, so encode + fsync + rotate are inside the span | in-memory dump: no file, no fsync, no rotate |
| Client path | gRPC client over the real stack | bufconn gRPC client over two loopback UDP adapters |
| Race detector | off | off |

Under sustained write flow the debounce timer never fires alone — the quiet window never
arrives — so the dump starts at `oldest_unconfirmed + dump_max_wait` and the write waits for
that dump. With `dump_time` naming the dump cycle's duration:

```text
latency = U(0, MaxWait) + dump_time
p50     ≈ MaxWait/2 + dump
p99     ≈ MaxWait + dump
```

That is the design, not a regression: the default profile reaches `dump_max_wait` + dump time by
design, and its confirmation bound is therefore the knob plus the dump, never a sub-debounce
figure. A single sequential client is a different shape: every write waits for the quiet window,
so the rate is `1/(Debounce + dump_time)` — at the default 50 ms debounce plus a measured
real-disk dump of tens of milliseconds, the ceiling is ≈ 14 writes/s. The default profile is the
disk-efficient profile for concurrent flow, not a low-latency profile for one writer.

**The fast-response profile, with its conditions.** The fast profile is `dump_debounce` 1 ms and
a 50 ms max-wait, an in-memory dump, two loopback UDP adapters, a bufconn gRPC client, a ~1 ms
background burst, one `Set(WaitForReplication: true)` per measured iteration, and race off.
Under exactly those conditions the second run measured mean 2.82 ms (bound `PerfFastMean` 25ms),
p50 2.60 ms, p99 5.55 ms (bound `PerfFastP99` 100ms) and max 14.72 ms over 817 samples
(`TestPerfFastProfileLatency`, wrapping `BenchmarkConfirmationLatency`'s fast arm). The measured
span includes the write's wait for the in-memory dump and the loopback Ack; it excludes real
disk time by construction. On a real-disk dump of 10 000 × 1 KiB the dump alone is tens of
milliseconds, so the fast profile's 50 ms max-wait is a floor, not a ceiling: the number above
is an in-memory-dump benchmark result and is never quoted without these conditions. The fast arm
ran later in the same `make perf` invocation at the same host; its logged load was
`3.84 5.26 6.59`, and it differs from the block below only in that field.

**Measured default profile.** `TestPerfDefaultProfileLatency` preloaded 10 000 × 1 KiB on the
real filesystem, waited for the covering dump and an empty unacked set, then ran an open-loop
1 000 writes/s × 15 s schedule over 100 hot keys with `--sync-interval=10m0s`, so no periodic
listing round entered the window. Each sample is completion minus that op's **scheduled**
instant, so a backlog cannot hide behind a closed loop (no coordinated omission).

hardware: CPU=Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz; logical=16; RAM=62.7 GiB; load=4.00 5.75 6.89; root-disk=/dev/mapper/ubuntu--vg-ubuntu--lv (ext4,rotational=unknown); go=linux/amd64 go1.27.4; module=github.com/Djarvur/sup-kv (vcs unknown, unknown); race=off; GOMAXPROCS=16

Commit: `9cdc0f1`. Command:
`SUPKV_PERF=1 SUPKV_PERF_DUMP_DIR=$(pwd)/.tmp/perf go test -count=1 -p 1 -timeout 30m -v -run '^TestPerfDefaultProfileLatency$' ./internal/stand/`
(one arm of `make perf`).

| Profile | State | Open-loop rate | p50 | p99 | Drain after window | Media | Race |
|---------|-------|----------------|-----|-----|--------------------|-------|------|
| default | 10 000 × 1 KiB, 100 hot keys | 1 000.1 writes/s | 598.87 ms | 1.095 s | 638.96 ms | ext4 on the local NVMe SSD | off |

Bounds: p50 ≤ `PerfDefaultP50` 750ms, p99 ≤ `PerfDefaultP99` 1.5s. Both measured values sit
inside the model's `MaxWait + dump` envelope; the p99 is just over `dump_max_wait` because it
carries one full dump.

**The disk cost.** The profiles are a disk/latency trade, not a free lunch. The default profile
dumps at most once per `dump_max_wait` — about 1 dump/s at the 1 s default, and one per write for
a sequential single client — while the fast profile dumps about 20 dumps/s at its 50 ms
max-wait. Each dump rewrites the whole state (the full-dump policy stands, ADR 0003), so the fast
profile trades roughly twenty times the dump bytes for its latency. Those bytes are measured in
[Disk-write amplification](#disk-write-amplification): the fast profile is never presented as
free.

## Burst at 1000 writes/s

The burst is a standing tested bound, not folklore. `TestRed_R02_burst` is the frozen red
contract (nightly, `SUPKV_RED_TESTS=1`); `TestPerfBurstDistinctKeys` is its operator-runnable
twin under `make perf`, exercising the same scenario through the same helper with the same
bounds. `TestEngine_burstBoundedConfirmation` is the deterministic cadence half: 1 000 records at
a 1 ms virtual cadence, first dump exactly at 1 s, at most three dumps, every write confirmed by
then.

**Guaranteed at the measured rate.** At 1 KiB values, two healthy nodes, default buffers and an
open-loop schedule, the standing row guarantees: at least 990 accepted writes/s over a 15 s send
window over 10 000 distinct keys; one first datagram per write plus paced retransmits; the unacked
set is empty within 5s of the send window's end — and because an Ack is emitted only after the
receiver applied the record, an empty unacked set also proves peer receipt, so "no loss of
confirmed writes" is observable from the same gauge; and the dump cadence stays at most one dump
per `dump_max_wait` under sustained flow (one datagram per write, not one dump per write).

**Not guaranteed.** 60 KiB values (IP fragmentation), rates above 1 000/s, an absent or
unresponsive peer (the pending set grows; the write stays dump-confirmed but the wait report
degrades), lossy or WAN segments, and client latency below `dump_max_wait` under sustained flow.
The `S-FAIL-15` registry row is closed against these tests and these two bounds.

Measured (second run): achieved 1000.1 writes/s against the `PerfBurstMinRate` floor of 990,
drain 305.13 ms against the `PerfBurstDrain` bound of 5s, 10 000 distinct keys, 1 KiB values,
fs ext4, zero failed writes.

hardware: CPU=Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz; logical=16; RAM=62.7 GiB; load=3.64 5.79 6.92; root-disk=/dev/mapper/ubuntu--vg-ubuntu--lv (ext4,rotational=unknown); go=linux/amd64 go1.27.4; module=github.com/Djarvur/sup-kv (vcs unknown, unknown); race=off; GOMAXPROCS=16

Commit: `9cdc0f1`. Command:
`SUPKV_PERF=1 SUPKV_PERF_DUMP_DIR=$(pwd)/.tmp/perf go test -count=1 -p 1 -timeout 30m -v -run '^TestPerfBurstDistinctKeys$' ./internal/stand/`
(one arm of `make perf`).

## Disk-write amplification

The full-dump policy is unchanged (ADR 0003, decision 14.3); this section measures what it
costs, it does not redesign it. The method, in the order the row was produced:

1. Run the two-node pair with default dump knobs, metrics on, and the dump on a **real
   filesystem**. `TestPerfAmplification` refuses a tmpfs/ramfs base by name: a memory-backed
   dump measures memory, not disk.
2. Preload a fixed 10 000 × 1 KiB state, then wait for its covering dump and an empty
   replication unacked set. Take the baseline: `supkv_dumps_total`, the main dump file's size,
   `/proc/<pid>/io` `write_bytes`, the summed `supkv_dumps_failures_total` classes, and the
   logical bytes the clients wrote.
3. Drive an open-loop 1 000 writes/s × 15 s hot-key window and wait for the unacked set to
   empty, so the final snapshot sees every dump the window caused.
4. Compute `amplification = (Δ dumps × dump size) / logical bytes`, and check
   `bytes-per-dump = write_bytes Δ / Δ dumps` against the dump size. A failed attempt writes no
   generation, so the row also requires the failure counter not to move inside the window.
5. Record the media facts: filesystem type, device and the rotational flag where the platform
   reports one.

Measured (second run), node A of the pair, healthy window (failures 0 on both sides):

| Quantity | Measured |
|----------|----------|
| Δ dumps | 15 |
| Main dump file | 14 977 443 B (14.28 MiB) |
| `write_bytes` Δ | 224 686 080 B (214.27 MiB) |
| bytes-per-dump | 14 979 072 B (14.29 MiB) |
| bytes-per-dump bound | ≤ 1.1× the dump size = `PerfBytesPerDumpRatio` × 16 475 187 B |
| Logical bytes written by clients | 15 645 000 B (14.92 MiB): 15 000 writes × (19-byte key + 1 KiB value) |
| Amplification | 14.36× |
| Media | ext4 on an LVM logical volume over dm-crypt on the workstation's NVMe SSD (`nvme0n1`, `lsblk` ROTA 0); the block's mountinfo probe reports `rotational=unknown` |

The 14.36× is the policy, not a leak: fifteen dumps each rewrote the whole 14.3 MiB state while
the clients wrote 14.9 MiB of logical value, so the window submitted 15 × the state size. The
per-dump check confirms the other half — `write_bytes` per dump equals the dump file's size
(0.9999×), so one generation is written once, not twice.

**The target-media gap.** The appliance's target media is eMMC/SD on the two-board system; that
hardware is not attached to this workstation. This row is therefore a workstation NVMe/SSD
result and is **not** target-grade: no number above is quoted as the appliance's amplification,
and the fast profile's dump cadence is exactly what would make target-media write endurance the
next question. `docs/SPEC-DEVIATIONS.md` records the gap against the Spec open question #2. If
a target-grade eMMC/SD device is ever attached, rerun this same row on it and replace the table —
do not annotate it.

hardware: CPU=Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz; logical=16; RAM=62.7 GiB; load=4.67 5.77 6.85; root-disk=/dev/mapper/ubuntu--vg-ubuntu--lv (ext4,rotational=unknown); go=linux/amd64 go1.27.4; module=github.com/Djarvur/sup-kv (vcs unknown, unknown); race=off; GOMAXPROCS=16

Commit: `9cdc0f1`. Command:
`SUPKV_PERF=1 SUPKV_PERF_DUMP_DIR=$(pwd)/.tmp/perf go test -count=1 -p 1 -timeout 30m -v -run '^TestPerfAmplification$' ./internal/stand/`
(one arm of `make perf`).

## Memory model

The model, from the as-built streaming path:

```text
logical state   = Σ (len(key) + len(value)) over live records
dump bytes      ≈ 1.37 × Σ len(value) + JSON/key framing          (base64 4/3 plus framing)
live heap peak  ≈ logical state + N × record overhead + 64 KiB + max encoded line
alloc churn     ≈ one Records copy + one garbage line per record
RSS             ≈ baseline + live heap + GC headroom (GOGC=100 by default)
```

A dump cycle takes exactly one `Records()` deep copy and streams one encoded record line at a
time through the adapter's 64 KiB buffered writer, so the encoded generation is never
materialized as a second buffer (PERSIST-06). The only other live buffer is the record line being
encoded, whose base64 expansion is at most about 1.34 × the configured value limit. The precise
transient is the persist benchmark's `peak-live-B` metric; this row's sampled VmRSS peak is the
loose tripwire. One residual pause stays stated rather than claimed away: `Records()` clones
every value while holding the store's read lock, so writers are paused for one state-sized copy
at snapshot time; eliminating it needs a copy-on-write view of the store and is residual work.

Measured points (second run), each on a fresh pair, real-disk dump, pinned sync interval:

| State | logical state | dump bytes | RSS | RSS / logical | VmHWM | sampled peak growth | peak / logical |
|-------|---------------|------------|-----|---------------|-------|---------------------|----------------|
| 1 000 × 1 KiB | 1 038 000 B | 1 492 779 B | 28 639 232 B (27.31 MiB) | 27.59× | 28 725 248 B | 4 096 B | 0.00× |
| 10 000 × 1 KiB | 10 380 000 B | 14 927 449 B | 67 739 648 B (64.60 MiB) | 6.53× | 70 320 128 B | 24 576 B | 0.00× |
| 1 000 × 60 KiB | 61 454 000 B | 82 044 792 B (78.24 MiB) | 229 527 552 B (218.89 MiB) | 3.73× | 255 778 816 B | 33 480 704 B | 0.54× |

Read the ratios by row shape. The 1 KiB rows are **key-dominated**: per-key map, version and
record overhead plus Go's runtime baseline dominate a small state, so `RSS / logical` is large
even though the state itself is small. The 60 KiB row is **value-dominated**: its dump is
82.0 MB for 61.5 MB of value (1.34×, inside the model's ≈ 1.37 × value-bytes plus framing), and
the one-dump sampled peak growth is 0.54 × the logical state. Only the value-dominated row gates:
RSS ≤ 12 × logical + a 256 MiB allowance (`PerfMemoryRSSRatio`, `PerfMemoryRSSBase`) and one dump
cycle's sampled peak growth ≤ 2 × logical (`PerfDumpPeakLiveRatio`, PERSIST-06 accepts ≤ 1.3×
live). The measured 3.73× and 0.54× sit inside both; a return to encode-into-bytes would show
roughly 4-5× on the peak.

hardware: CPU=Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz; logical=16; RAM=62.7 GiB; load=4.12 5.52 6.73; root-disk=/dev/mapper/ubuntu--vg-ubuntu--lv (ext4,rotational=unknown); go=linux/amd64 go1.27.4; module=github.com/Djarvur/sup-kv (vcs unknown, unknown); race=off; GOMAXPROCS=16

Commit: `9cdc0f1`. Command:
`SUPKV_PERF=1 SUPKV_PERF_DUMP_DIR=$(pwd)/.tmp/perf go test -count=1 -p 1 -timeout 30m -v -run '^TestPerfMemoryPoints$' ./internal/stand/`
(one arm of `make perf`).

## Absent peer

The claim: with no peer present, and with the unacked set growing, a local write's notification
cost stays O(1) in the pending-set size. The mechanism is the per-key index:
`trackLocalWrite`'s tracked-version guard, the supersession check and `applyRemote`'s
displacement read only the written key's index entry. The one O(n) path left is the first-seen
discovery flush, which must emit every pending pair once to a newly appeared peer.

The deterministic gates are the always-on proof, in `make test`, with no clock and no gate:
`TestCore_notifyWriteCostFlat` and `TestCore_applyRemoteCostFlat` assert `PendingVisits` per
operation is exactly 0 at 10 and at 10 000 unacked records with fresh keys, and
`TestCore_pendingVisitsRepeatedKey` proves the counter is wired to the real loops (a repeated-key
write inspects exactly 2 entries). A regression to a whole-pending scan fails these by name.

The paired wall-clock measurement runs only under `make perf`: `TestPerfAbsentPeerWriteCost`
measures both arms back to back at 10 000 pending 1 KiB records — the absent arm knows no peer
and its unacked set grows one entry per iteration, the live arm Acks each write and stays at
10 000. Second run: absent 2 249 ns/op vs live 8 598 ns/op, ratio 0.26 against the
`PerfAbsentPeerRatio` bound of ≤ 2.0×; both arms reported 0.00 pending-visits/op.

hardware: CPU=Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz; logical=16; RAM=62.7 GiB; load=3.80 5.31 6.62; root-disk=/dev/mapper/ubuntu--vg-ubuntu--lv (ext4,rotational=unknown); go=linux/amd64 go1.27.4; module=github.com/Djarvur/sup-kv (vcs unknown, unknown); race=off; GOMAXPROCS=16

Commit: `9cdc0f1`. Command:
`SUPKV_PERF=1 SUPKV_PERF_DUMP_DIR=$(pwd)/.tmp/perf go test -count=1 -p 1 -timeout 30m -v -run '^TestPerfAbsentPeerWriteCost$' ./internal/app/repl/`
(one arm of `make perf`).

## Running the measurement

```text
make perf
```

The target is Linux-only: the stand harness spawns real binaries and is Linux-only, so on any
other platform the target fails fast with a named message. It invokes the perf rows with
`SUPKV_PERF=1`, `-p 1` (one measurement at a time, so no sibling package contaminates the next),
no `-race` (the detector measures itself rather than the product) and a 30-minute timeout, then
runs the ordinary stand suite once more with no thresholds. Dumps land in `.tmp/perf`
(gitignored) rather than a memory-backed `/tmp`, and every row logs its hardware block in `-v`
output.

Thresholds live only in `internal/testutil/perf.go`; a perf row's first statement is
`testutil.RequirePerf`, so plain `go test` (and `make test`) skips every row with a message
naming `make perf`. CI deliberately keeps the threshold-free `make bench-smoke` (D-13/D-14) and
never invokes `make perf`: shared runners cannot supply the calm host a wall-clock bound needs.
To rerun one row, use the per-section command above; to reproduce the whole ledger, run
`make perf` twice and publish the second run's block and figures, as this document does.
