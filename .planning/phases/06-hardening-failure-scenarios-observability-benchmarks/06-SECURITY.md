---
phase: "6"
slug: "hardening-failure-scenarios-observability-benchmarks"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-11-16"
---

# Phase 6 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| network -> metrics/debug listener | Unauthenticated HTTP/TLS clients; pprof heap dumps and expvar are information-disclosure and CPU-DoS surfaces (T-6-07/08/09) | heap/profile data (memory contents), runtime vars |
| operator env -> listener config | Cert/key/CA file paths and bind address cross from the environment; incomplete or wrong config must fail closed (T-6-10) | filesystem paths, TLS key material |
| scrape -> core state | Scraper-triggered reads of adapter/engine state must be lock-free or copy-on-write (no scrape-induced stalls) | counters/gauges only |
| OS signals -> main | SIGTERM/SIGINT arrive from outside the process; the shutdown path must stay total under repeated signals (T-6-03) | process lifecycle |
| core -> stderr (slog) | Structured log events leave the process; they must never carry record values or user data (T-6-05/06) | metadata strings (class, handles, node ids) |
| CI runner -> jobs | Untrusted load spikes on shared runners; job bounds prevent indefinite hangs (T-6-04); benchmarks stay manual-only (T-6-14) | none |
| tests -> shared hosts | Scenario/benchmark sockets on CI hosts bind ephemeral ports for the test lifetime only (T-6-12) | fixed small datagrams |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-6-01 | Tampering | persist.Engine ApplyRemote/Restore conditional markDirty | high | mitigate | Condition coded to storage.ApplyRemote's documented return contract; both directions locked by TestEngine_redundantDirty cases (a)/(b) — landed 0f5e03d, re-verified PASS by phase-3 verifier 2026-11-16 | closed |
| T-6-02 | DoS | repl normalized() rate clamp | medium | mitigate | maxSyncResponseRate clamp pins the 1e9 boundary; TestSync_responseRateClamp — landed d7c5f4e, re-verified PASS by phase-4 verifier 2026-11-16 | closed |
| T-6-03 | Tampering | cmd/sup-kv serve() signal lifecycle | high | mitigate | armSignals() holds one buffered channel across serving+shutdown; second SIGTERM takes force-stop with final dump surviving; TestShutdown_secondSignal (real process, NOT_SERVING observed) — landed 69d1e47, re-verified PASS by phase-5 verifier 2026-11-16 | closed |
| T-6-04 | DoS | ci.yml jobs without bounds | low | mitigate | timeout-minutes 15/20 additive-only, zero check-run context diffs — landed ba0d984, audited by phase-1 verifier 2026-11-16 | closed |
| T-6-05 | Information Disclosure | slog pipeline (tap bridge, discard events) | medium | mitigate | Tap forwards prebuilt metadata strings only (class, handles, node ids — never keys or values); TestLog_jsonDefault/TestDiscardLog_classes assert the attr surface — landed d9a0a43/34b089f | closed |
| T-6-06 | DoS | discard-class log events under datagram flood | medium | mitigate | Discard lines emit at Debug level (default info suppresses); O(1) counters carry the signal; capturing handler's Debug filter enforced in test — landed 34b089f | closed |
| T-6-07 | Information Disclosure | /debug/pprof/* and /debug/vars | high | mitigate | Opt-in with no default listener (D-01), mTLS with verified client certs for non-loopback (D-02), one listener for the surface (D-03); TestMetrics_noDefaultListener/TestMetricsTLS_required — landed 3714924/c59db3c/b296006 | closed |
| T-6-08 | Spoofing | metrics TLS listener | medium | mitigate | crypto/tls defaults + explicit MinVersion tls.VersionTLS12, no custom ciphers; TestMetricsTLS_required proves unauthenticated handshakes fail — landed b296006 | closed |
| T-6-09 | DoS | pprof profile endpoints (CPU burn per request) | medium | accept | Same opt-in + mTLS gate as T-6-07 bounds who can request profiles; residual plaintext-loopback exposure is the operator's deliberate debug choice per D-02 — accepted with documentation (Phase 7 ops docs restate) | closed |
| T-6-10 | Tampering | cert-triple validation order | medium | mitigate | Validation runs before any bind; incomplete set can never reach LoadX509KeyPair; TestMetrics_incompleteCertSet pins exit 2 naming all three envs — landed c59db3c | closed |
| T-6-11 | Tampering | sim scenario assertions (false-evidence risk) | low | mitigate | Assertions reuse the shared pointwise-supremum model and documented bound constants; seeded replay keeps failures reproducible — landed 641b92d/816b45d | closed |
| T-6-12 | DoS | scenario sockets on shared CI hosts | low | accept | Ephemeral wildcard/loopback UDP ports for the test lifetime only, fixed small datagrams, t.Cleanup teardown; exposure bounded to the test process — landed 3457109/cb06fb5 | closed |
| T-6-13 | Tampering | exec scenario exit-code/dump assertions | low | mitigate | Assertions reuse assertExitCode and dump-grep precedents; TestExec_sigtermFinalDump pins exit 0 AND on-disk replicated-record containment together — landed 05958b0 | closed |
| T-6-14 | DoS | benchmarks on CI runners | low | mitigate | D-13/D-14 keep benchmarks out of CI entirely (make bench is manual; .github/ byte-untouched, audited by phase-1 verifier 2026-11-16); scoped three-package command bounds local runs to seconds — landed 9aa7bc0 | closed |
| T-6-15 | Tampering | fsync_darwin fcntl wrapper | low | mitigate | ~15-line helper over one syscall call site; both darwin arches cross-compile-checked (GOOS=darwin build+vet green, phase-3 verifier 2026-11-16); non-darwin byte-identical pass-through; power-loss documented as untestable in CI — landed 4f75a3b | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-6-01 | T-6-09 | pprof CPU burn bounded by the opt-in + mTLS gate; plaintext loopback is the operator's deliberate debug choice per D-02 — documented, Phase 7 ops docs restate the caveat | plan 06-04 (user-approved CONTEXT) | 2026-11-16 |
| AR-6-02 | T-6-12 | Test sockets bind ephemeral ports for the process lifetime only; no privileged ports, no user data, no residual listeners | plan 06-06 (user-approved CONTEXT) | 2026-11-16 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-11-16 | 15 | 15 | 0 | execute-phase verify:post gate (L1, register-authored-at-plan-time short-circuit; mitigation evidence from plan SUMMARYs + phase 1/3/4/5 verifier re-confirmations) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-11-16
