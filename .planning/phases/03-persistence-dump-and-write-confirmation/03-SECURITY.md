---
phase: "03"
slug: "persistence-dump-and-write-confirmation"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-11-14"
---

# Phase 03 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

> Retroactive run (2026-11-14): phase 03 executed 2026-11-03 without a SECURITY.md — the secure-phase pass was skipped in that era. Register rebuilt from the plans' own `<threat_model>` blocks (authored at plan time) and verified at L1 grep depth against the current tree (ASVS L1 short-circuit; register_authored_at_plan_time: true, threats_open: 0).

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Disk → startup load | Dump file may be torn, truncated, hand-edited or hostile (operator error, crash residue, disk corruption) — crosses into Decode/Parse/Open | dump bytes → in-memory records |
| Caller → snapshot name | Untrusted string reaching filesystem path construction in the adapter | snapshot name → filesystem path |
| Engine waiters → dumper goroutine | Internal concurrency boundary: waiter resolution must be exactly-once and deadlock-free | confirmation tickets (done channel) |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-3-01 | Tampering | codec.Decode / Parse on a hostile dump file | high | mitigate | Parse-never-panic: json.Decoder streaming, hex decode length-checked to 16, version.Unpack total over any bit pattern, unknown format_version = unreadable generation, per-record checksum discard via storage.RecordChecksum, Data bounded by maxRecordBytes (61440, codec.go:25); every fault is a returned error or a counted discard — verified in internal/app/persist/codec.go + codec_test.go | closed |
| T-3-02 | Tampering | persistfile saveAtomically (torn/crashed generations) | medium | mitigate | temp-in-target-dir → write → fsync → rotate → rename → dir-fsync; rotate-first crash window absorbed by fallback chain main → .prev (dumpfile.go:28,54,68,119,142); crash-residue table tests cover each window | closed |
| T-3-03 | DoS | Engine waiter lifecycle | high | mitigate | Every waiter resolves exactly once (success / ErrDumpFailed / ErrClosed sweep on Run exit, engine.go:151,225-245,262,303); single dumper goroutine; mutex never held across I/O; synctest quiescence asserts no leaked goroutine | closed |
| T-3-04 | Tampering / Elevation of Privilege | persistfile CreateSnapshot / ReadSnapshot path construction | high | mitigate | validateName BEFORE any filesystem operation: charset [A-Za-z0-9._-], length 1..64, explicit "." / ".." rejection (snapshot.go:39,60,92-102) — no separator or traversal reaches filepath.Join; ErrInvalidSnapshotName with zero filesystem side effects | closed |
| T-3-05 | Tampering | local symlink swap on the dump or snapshot path | low | accept | Trusted-host trust model (docs/IDEA.md §Trust model): CreateTemp plus rename replaces paths atomically rather than following symlinks; defending against a hostile local user is out of scope | closed — accepted risk (see log) |
| T-3-06 | Tampering | torn or unconfirmed snapshot content | medium | mitigate | Snapshot copies the completed on-disk main dump (ErrNoDump guard, snapshot.go:20-34); Restore decodes through the checksum-discard codec (T-3-01 carry-over) | closed |
| T-3-SC | Tampering | Package installs (new modules) | high | mitigate | Zero installs at phase time (stdlib + in-repo only); the shipped graph later gained exactly the two sanctioned direct requires (grpc, protobuf) per the documented phase-5 dependency policy (AGENTS.md D-03) — no property-test/UUID/serialization library entered the build | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on (high) count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-3-01 | T-3-05 | Local symlink swap by a hostile user on the same host: atomic CreateTemp+rename already replaces paths rather than following symlinks; full local-attacker defense is outside the trusted-host trust model (docs/IDEA.md §Trust model) | plan 03-02 (disposition: accept) | 2026-11-03 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-11-14 | 7 | 7 (1 via accepted risk) | 0 | gsd-secure-phase retroactive (orchestrator, L1 short-circuit) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed

## Security Audit 2026-11-14

| Metric | Count |
|---|---|
| Threats found | 7 |
| Closed | 7 |
| Open | 0 |
