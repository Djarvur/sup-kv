---
phase: "4"
slug: "replication-and-deterministic-simulator"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-11-07"
---

# Phase 4 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| UDP broadcast segment → node | Lossy single-subnet broadcast domain; any host on the segment can send datagrams to the replication port | Protobuf Envelope (Hello/DataReplication/Ack/DataSync/DataRequest), CRC32-prefixed, ≤65507 bytes |
| Node → UDP broadcast segment | Outbound replication traffic (broadcast writes, unicast replies/sync) | Same Envelope schema; unicast replies only, rate-limited |
| Wire → store (apply seam) | Decoded records crossing from the adapter into the pure core's StateStore.ApplyRemote | storage.Record (key, value ≤ value_max_bytes, version) |
| Local store → simulator (test-only) | simnet drives the production repl.Core handlers over a virtual network | Same structs; no production-code reachability from simnet (imports guarded) |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-04-01 | DoS | adapter decode + core classification | high | mitigate | 65507-byte capped single read buffer; CRC gate before body unmarshal; undecodable/garbage → Undecodable discard, never panic (repl.go:48,402-431; core.go:302-336; TestDiscard_totality 500 seeded rounds) | closed |
| T-04-02 | DoS | own-broadcast feedback | medium | mitigate | OwnEcho drop before protocol processing by run_id equality (core.go:315-319); real-socket evidence TestSmoke_twoNodesUDP OwnEcho≥1 | closed |
| T-04-03 | Tampering | duplicate node_id | medium | mitigate | Both duplicate variants classified DuplicateNodeID with config-error log naming node_id and both handles; restart branch never pre-empted (core.go:320-334; TestDiscard_classes rows) | closed |
| T-04-04 | DoS | oversize record via wire | low | mitigate | ApplyRemote surfaces ErrValueTooLarge unchanged; apply error → log, no Ack, no panic (engine.go:424-435; core.go:441-448; store Merge rejects > value_max_bytes; TestEngine_applyRemoteTriggersDump) | closed |
| T-04-05 | DoS | retransmit post-heal burst | medium | mitigate | One broadcast per entry per ladder step, sorted-key emission; heal-drain bound asserted with exact limiter response term (core.go:748-762; TestBound_partitionHeal) | closed |
| T-04-06 | Spoofing/Tampering | forged Acks drain tracking | low | accept | Trusted-segment trust model (docs/IDEA.md:150-154): CRC32 integrity-only, no auth/encryption in v1, HMAC documented future work | closed |
| T-04-07 | DoS | sync-response amplification | medium | mitigate | Egress queue with exactly 3 enqueue sites, unicast-only responses, rate limiter with integer-division floor spacing (core.go:497,520,522,533-559; TestSync_rateLimit) | closed |
| T-04-08 | DoS | oversized keys vs chunk budget | low | accept | Product-envelope limitation documented (04-03-PLAN, DOCS-02; IDEA.md:145); graceful degradation: oversized entry ships alone, never dropped (core.go:673-696) | closed |
| T-04-09 | Spoofing | forged restart Hello | low | accept | Trusted segment (IDEA.md:150-154,198); reaction emits only key+version listings, values flow solely through Merge-gated replies; once-per-change idempotence (core.go:407-416; restart tests) | closed |
| T-04-10 | DoS/Repudiation | simulator non-determinism | high | mitigate | (Time,Seq) total order, monotonic insertion seq, fault draws only via injected rng, deterministic simRunID; zero goroutines/testing-imports/sleeps in simnet (simnet.go:55-60,98-101,225-252,324-333; TestSimnet_determinism, TestChaos_determinismReplay + SUPKV_SIM_SEED) | closed |
| T-04-11 | Tampering (evidence) | harness divergence | medium | mitigate | Cluster.Write = store Set then NotifyLocalWrite (production author sequence); drive's Advance loop mirrors the adapter event loop over the same repl.Core handlers; real driver covered by smoke test (simnet.go:421-439,457-467) | closed |
| T-04-SC | Tampering | supply chain: first direct runtime require (protobuf v1.36.12) | high | mitigate | Single direct require pinned; tool directive; go.sum untouched by phase 4; Package Legitimacy Audit verdict OK with [SUS]: none (04-RESEARCH.md:124-137); depguard + lint gates enforced | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-04-01 | T-04-06 (low) | Trusted single-subnet segment; forged Acks require an on-segment attacker, out of the v1 trust model; HMAC documented as future work (IDEA.md) | Maintainer (documented trust model, IDEA.md:150-154) | 2026-11-07 |
| AR-04-02 | T-04-08 (low) | Keys near the datagram budget ship as single-entry chunks — documented product-envelope degradation, never data loss | Maintainer (documented, 04-03-PLAN / DOCS-02) | 2026-11-07 |
| AR-04-03 | T-04-09 (low) | Forged restart Hello triggers only key+version listings through Merge-gated path; no value exposure; trusted segment assumption | Maintainer (documented trust model, IDEA.md:150-154,198) | 2026-11-07 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-11-07 | 17 | 17 | 0 | gsd-security-auditor (register authored at plan time; evidence tests executed green under -race) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-11-07

## Security Audit 2026-11-06

| Metric | Count |
|---|---|
| Threats found | 17 |
| Closed | 17 |
| Open | 0 |
