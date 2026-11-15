---
phase: "05"
slug: "grpc-api-and-binary"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-11-15"
---

# Phase 05 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

> Retroactive run (2026-11-15): phase 05 executed 2026-11-08/09 without a SECURITY.md — the secure-phase pass was skipped in that era (same gap as phase 03, closed the same way). Register rebuilt from the plans' own `<threat_model>` blocks (authored at plan time, one per plan) and verified at L1 grep depth against the current tree (ASVS L1 short-circuit; register_authored_at_plan_time: true, threats_open: 0).

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| gRPC client → SupKVService server | Untrusted-length (bounded by the 4 MiB default recv cap) client input over the trusted LAN | requests (keys, values, names) |
| gRPC client → snapshot RPCs | Client-supplied names cross into filesystem-adjacent persistfile operations | snapshot names → filesystem paths |
| handler → persist.Engine / repl mirrors | Validated requests cross into the decision core; replication state crosses goroutines only through adapter atomics | requests / mirrored snapshots |
| store write lock → Watch hub → subscriber streams | Events cross via bounded non-blocking ingress; per-subscriber bounded channels | change events |
| process environment → config; SIGTERM → shutdown | Operator-controlled local input; the signal triggers the only path that must not skip the final dump | env/flags; signals |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-05-01 | Tampering | SupKVService server (unauthenticated RPC surface) | medium | accept | v1 accepted limitation per docs/IDEA.md §Trust model (trusted LAN, no auth/encryption); Phase 7 docs restate prominently | closed — accepted risk (see log) |
| T-05-02 | DoS | Set/List handlers (oversized requests) | medium | mitigate | grpc default 4 MiB recv cap kept (MaxSendMsgSize documented as the growth knob); value_max_bytes rejects oversized values → storage.ErrValueTooLarge → InvalidArgument | closed |
| T-05-03 | DoS | handlers (panic on hostile input) | medium | mitigate | Handlers total over any input: empty-key pre-check (server.go:100-121, InvalidArgument before any engine call; gate check even earlier — Unavailable during the gate), size enforcement in the engine, errors mapped to codes; no panic path | closed |
| T-05-04 | Information Disclosure | PendingSnapshot wait-set (eviction misread as Ack) | medium | mitigate | The Acked seam makes Ack and eviction distinguishable by construction (repl.go drained-row retention, WaitingOn nil + Acked exactly the core's set); eviction-without-Ack regression tests lock it (wait_test.go, repl_test.go) | closed |
| T-05-05 | DoS | OnChange tap under the store write lock | high | mitigate | Fan-out is a select-with-default into a bounded ingress (watch.go:191,214) — the store never waits on any subscriber; slow consumers shed with ResourceExhausted (watch.go:42), never silently served holes (re-snapshot on resubscribe); TestWatch_slowConsumer locks store liveness (watch_test.go:463) | closed |
| T-05-06 | DoS | WatchHub subscriber table (subscription flood) | medium | accept | v1 trusted-LAN: no per-node subscriber cap; each subscriber costs one bounded channel — memory bounded per subscriber; documented operational envelope, revisit with Phase 6 OBS work | closed — accepted risk (see log) |
| T-05-07 | Tampering | Snapshot name → persistfile (path traversal / elevation) | high | mitigate | persistfile validateName is the control (charset [A-Za-z0-9._-], length 1..64, "."/".." rejected, before any filesystem touch); the adapter only maps ErrInvalidSnapshotName → InvalidArgument (server.go:303-304), never re-validates or bypasses; hostile-name tests lock the mapping | closed |
| T-05-08 | Information Disclosure | Health endpoint (SERVING while gated) | low | mitigate | Health bound 1:1 to the gate atomic, set only by the resolver and shutdown shedding (gate.go:154); TestHealth_reflectsGate asserts both transitions; empty service name = whole-server status | closed |
| T-05-09 | Information Disclosure | wait report (false acked = integrity lie about durability) | medium | mitigate | Report derives only from the Acked seam, the appearance phase (notify-queue race), and frozen supersession counts; deadline/cancel always acked=false; the wait never changes the write outcome; adversarial tests lock the honest readings | closed |
| T-05-10 | DoS | Graceful shutdown (GracefulStop hang on open Watch streams) | high | mitigate | Pinned D-19 order: end Watch streams (apiCtx) before GracefulStop, bound with the 10s shutdownGrace and gs.Stop() fallback, final Engine.Flush before engine cancel (main.go:68-71,390-408); TestMain_lifecycle asserts exit 0 + dump contains the committed key after SIGTERM | closed |
| T-05-11 | Tampering | SUPKV_ env precedence (env silently beating an explicit flag) | low | mitigate | Each knob is a flag whose default is its SUPKV_ env twin — the stdlib flag parser delivers flag > env > default structurally (main.go:4-8, 215-216); TestMain_precedence observes listener placement end-to-end | closed |
| T-05-SC | Tampering | Package installs (grpc v1.85.0, protoc-gen-go(-grpc)) | high | mitigate | Modules proxy.golang.org/sumdb-verified, versions pinned in go.mod, probe-built during phase research; the graph later gained only buf (tool) and the 2026-11-15 toolchain go1.27.4 bump closing GO-2026-6859 (CVE-2026-97054) — govulncheck exit 0 | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on (high) count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-5-01 | T-05-01 | Unauthenticated gRPC surface: v1 trust model is the trusted LAN (docs/IDEA.md §Trust model) — no auth/encryption in v1; Phase 7 operational docs must restate the LAN-only exposure prominently | plan 05-01 (disposition: accept) | 2026-11-08 |
| AR-5-02 | T-05-06 | No per-node Watch subscriber cap: each subscriber costs one bounded channel (memory bounded per subscriber); trusted-LAN operational envelope; revisit with Phase 6 observability work | plan 05-03 (disposition: accept) | 2026-11-08 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-11-15 | 12 | 12 (2 via accepted risks) | 0 | gsd-secure-phase retroactive (orchestrator, L1 short-circuit) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed

## Security Audit 2026-11-15

| Metric | Count |
|---|---|
| Threats found | 12 |
| Closed | 12 |
| Open | 0 |
