---
status: testing
phase: 04-Replication and Deterministic Simulator
source: [04-VERIFICATION.md]
started: 2026-11-06T18:30:00Z
updated: 2026-11-06T18:30:00Z
---

## Current Test

number: 1
name: Judgment-tier CRC-wording prohibition ratification (plans 04-01/04-06)
expected: |
  All 6 mention sites of CRC in phase-4 code and docs say "corruption detection
  only, never tamper protection" (ADR-550: an LLM-judge PASS verdict is
  non-authoritative in autonomous mode, so a human ratifies the wording once).
awaiting: user response

## Tests

### 1. Judgment-tier CRC-wording prohibition ratification (plans 04-01/04-06)
expected: All 6 CRC mention sites say "corruption detection only, never tamper protection"; no site implies tamper protection.
result: [pending]

### 2. D-08 macOS smoke outcome
expected: Empirical until the first macOS CI run (Phase 5+ contexts). On this Linux host TestSmoke_twoNodesUDP passes; on macOS the smoke test either passes or loud-skips after its probe, naming both addresses tried and the documented rationale (macOS lo0 has no BROADCAST flag).
result: [pending]

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps
