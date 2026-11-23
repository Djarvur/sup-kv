---
phase: 06-api-and-semantics
updated: 2026-11-23T04:15:00Z
total: 9
---

# Code Review Disposition — Phase 06

One row per finding from `06-REVIEW.md`. Edit the **Disposition** column by hand when triaging
(`open` / `fixed` / `skipped` / `deferred`); put the reason in **Source**.

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| CR-01 | critical | fixed | Full-rollback snapshot load treats a named-but-undecodable snapshot record (bad checksum, bad version, malformed line, legacy truncation) as "absent from the snapshot", writes a rollback tombstone, and broadcasts it — data loss. `0c6dbf3`: `Restore` refuses `DecodeStats.Total() > 0` before touching the store; `ErrSnapshotIncomplete` → `codes.DataLoss`; flipped-checksum regression proves keys/versions survive with no notify/dump. Re-verified (independent reproducer now refused; `Open` discard-and-count resilience unchanged). |
| WR-01 | warning | fixed | A page token whose key exceeds `storage.MaxKeyBytes` is rejected, but the store/`Get` keep serving pre-upgrade longer keys; the paged walk emits such a token and the next page fails. `acb4620`: token domain is the exported `persist.MaxDumpLineBytes`, never `MaxKeyBytes`; ceilings re-derived (MaxRecv 187 472 / 190 440, MaxSend 6 416 464); 1 124-byte-key walk succeeds. Re-verified with arithmetic and the hostile-client bound assessment (≈18 MB worst-case in-flight, below v1.0 defaults). |
| WR-02 | warning | fixed | The Watch tap suppresses equal-version tie-break installs, contradicting `docs/API.md` and the store's tie-break law; a subscriber can miss a winner or remove-wins delete. `73d3549`: subscriber snapshot (version + store checksum) suppresses only an exact content duplicate; equal-version wins and remove-wins deletes are delivered (`TestWatch_equalVersionTieBreakDelivered`). Re-verified. |
| WR-03 | warning | fixed | `examples/client` sits inside the root module path prefix, so Go's internal rule does not stop it importing `internal/` — the "compile-time proof" claim is false. `73eb0b8`: `examples/client` is `example.com/sup-kv-client` (out of prefix) with a replace; an internal import is a compile error (re-probed on a fresh archive); depguard named as the enforced boundary. Re-verified. |
| IN-01 | info | fixed | Unreachable `ErrValueTooLarge` skip in `replayRecord` (same pattern as Phase-5 IN-05). `1635a03`: unreachable branch removed; direct-call test proves the store error surfaces. Re-verified. |
| IN-02 | info | fixed | Stale `wait.go:94-225` citation in the SPEC-DEVIATIONS row this phase edited. `d213044`: `S-ALG-CU-11` cites `wait.go:72-88`/`:93-98`. Re-verified. |
| IN-03 | info | fixed | Transport doc claims "every gRPC setting is explicit"; keepalive is explicit but some defaults remain. `a9090e7`: docs/comments claim only the pinned ceilings and keepalive cadence; other knobs keep gRPC defaults. Re-verified. |
| IN-04 | info | fixed | No largest-value transport round-trip test (research Pitfall 11) at `main.go:735`. `c55c2b8`: `TestGRPC_maxValueRoundTrip` at the 64 408-byte ceiling; the wiring-regression discrimination loss is disclosed (derivation pinned by `TestLimitsFor`). Re-verified. |

| N-1 | info | fixed | `f5dfc87`: the CR-01 insertion shifted the `S-PERS-11`/`S-PERS-12` citations; both repointed to the current `Restore`/`replaySnapshot`/`replayKeys`/`rollbackKey`/`restoreChanges`/`replayRecord` lines. Re-verified. |

**Legend:** `open` — seen, untriaged · `fixed` — resolved (record the commit) · `skipped` — will
not fix (say why) · `deferred` — postponed (name the phase).

**Re-verification verdict (same reviewer, read-only, independent reproducers):** CR-01, WR-01…WR-03, IN-01…IN-04 and N-1 FIXED; no new code defect. Two disclosed consequences accepted: a partially readable snapshot is refused for `SnapshotLoad` (`Open` keeps its count-and-skip resilience), and the transport ceilings rise to admit the maximum List token (bounded, below v1.0 library defaults).

**Post-fix gates (2026-11-23):** `make lint` clean; `go test -race -cover ./...` green; `make buf-lint && make buf-breaking && make client-example` green; `SUPKV_RED_TESTS=1 bash scripts/red-runner` green — 7/7 rows, 0 errors, both mutants killed; frozen tests, TSV, `.golangci.yml`, `go.mod`/`go.sum`, and the root module's dependency graph byte-identical.
