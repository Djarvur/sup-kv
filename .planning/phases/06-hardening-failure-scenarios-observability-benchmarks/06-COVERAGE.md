---
phase: 06-hardening-failure-scenarios-observability-benchmarks
status: sealed
created: 2026-11-16
---

# Phase 6 — API Coverage Matrix

> Required by the `api-coverage` verify:pre gate. Every capability INTEGRATE or OPT-OUT, with a reason.

## Matrix

| Capability | Decision | Reason |
|------------|----------|--------|
| External/third-party APIs (any) | OPT-OUT | None integrated. Phase 6 is hardening of sup-kv's own code: review-fix backlogs, slog observability seams, an opt-in metrics/debug listener served by sup-kv itself (in-repo exporter, no client library), simulator/real-socket/exec scenario suites, benchmarks, and a darwin F_FULLFSYNC flush. Zero outbound calls to any external service; zero new client SDKs; the shipped dependency graph remains grpc + protobuf only. |
| sup-kv's own gRPC surface | N/A (own product surface) | The gate's detection signals — ("surface", "api") and ("wire", "endpoint") — match planning prose referring to sup-kv's OWN gRPC API (phase 5 deliverable, unchanged in phase 6: `internal/grpcapi` byte-identical per the phase-5 regeneration audit) and its own UDP wire protocol. An internal product surface is not an external API integration. |

## Verdict

**No external API coverage obligations for this phase.** The detection that produced this matrix was a keyword false positive on self-referential surface prose. Re-evaluate if a future phase integrates an outbound/third-party API (the gate remains enabled project-wide).

## Audit Trail

| Date | Event | By |
|------|-------|-----|
| 2026-11-16 | Matrix sealed after verify:pre gate halt (detected: surface/api, wire/endpoint); maintainer chose truthful opt-out over gate disable | main session (verify-work 6) |
