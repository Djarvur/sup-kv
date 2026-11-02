---
phase: "02"
slug: "engine-hlc-versioning-and-lww-store"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-11-02"
---

# Phase 02 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Caller → Store API | Untrusted input crosses here: keys, value bytes (size validated before any copy) | key strings, value bytes ≤ 60 KiB |
| Future replication → Merge (Phase 4) | Remote records will arrive over an unauthenticated UDP segment; this phase makes Merge total and non-panicking on any record content | arbitrary Record versions and data |
| Future dump load → Record (Phase 3) | Record fields stay exported and self-validating (Checksum) for on-disk round-trip | JSON record fields |
| Arbitrary 16-byte versions → Unpack/Compare/Merge | Every version operation must be total over all bit patterns | raw RecordVersion bytes |
| Adversarial wall clock → Clock | The injected `now func() time.Time` is the single seam; correctness survives any jump pattern (D-02) | wall-clock readings |
| Concurrent callers → Store/Clock | Multiple goroutines share one Store and one Clock; lock discipline is the only guard | concurrent mixed operations |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-02-01 | DoS | Store.Set/SetIfAbsent/Merge size check | high | mitigate | `len(data) > ValueMaxBytes` rejected with `ErrValueTooLarge` before any allocation or copy — store.go:129/160/192; boundary tested at 61440/61441 in `TestStore_valueLimit` | closed |
| T-02-02 | Tampering | Record.Data aliasing between store and callers | medium | mitigate | `bytes.Clone` defensive copies on write (store.go:119/144) and read (store.go:227/246); ownership documented in doc.go | closed |
| T-02-03 | DoS | RecordVersion.Unpack/Compare on arbitrary bit patterns | medium | mitigate | Unpack/Compare are pure masking arithmetic with no panic path; `TestVersion_unpackTotal` feeds seeded-random raw 16-byte values through them | closed |
| T-02-04 | Tampering / Elevation of Privilege | Forged future timestamps winning every merge | medium | accept | Trusted-segment trust model is a documented accepted limitation (docs/IDEA.md "Trust model"); no auth in v1 by design; the drift signal (VERS-05) makes the condition observable; HMAC is documented future work | closed |
| T-02-05 | Tampering | Record.Checksum (CRC32) misconstrued as integrity/auth control | low | mitigate | doc.go:32 states Checksum is corruption detection for the Phase 3 dump only; `TestMerge_losersProperty` confirms Checksum never participates in version comparison | closed |
| T-02-SC | Tampering | Package installs (new modules) | high | mitigate | Zero installs this phase per D-04; `go.mod`/`go.sum` byte-identical to master across the whole phase branch (verified `git diff --quiet origin/master..HEAD -- go.mod go.sum`); no install commands in any task | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-02-01 | T-02-04 | No authentication of replication traffic in v1 — trusted-segment trust model decided in docs/IDEA.md; drift signal makes abuse observable; HMAC documented as future work | Project design (docs/IDEA.md Trust model) | 2026-11-02 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-11-02 | 6 | 6 | 0 | auto chain (execute-phase verify:post, ASVS L1 grep-depth, register authored at plan time) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-11-02
