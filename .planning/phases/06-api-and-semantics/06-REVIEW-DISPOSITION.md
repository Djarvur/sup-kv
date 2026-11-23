---
phase: 06-api-and-semantics
updated: 2026-11-23T03:10:00Z
total: 8
---

# Code Review Disposition — Phase 06

One row per finding from `06-REVIEW.md`. Edit the **Disposition** column by hand when triaging
(`open` / `fixed` / `skipped` / `deferred`); put the reason in **Source**.

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| CR-01 | critical | open | Full-rollback snapshot load treats a named-but-undecodable snapshot record (bad checksum, bad version, malformed line, legacy truncation) as "absent from the snapshot", writes a rollback tombstone, and broadcasts it — data loss. **Fix in phase:** a snapshot load with any discarded record must fail before applying any change (no rollback on a partially readable snapshot); regression test flips one checksum in an otherwise valid snapshot and asserts the key survives and the RPC fails. |
| WR-01 | warning | open | A page token whose key exceeds `storage.MaxKeyBytes` is rejected, but the store/`Get` keep serving pre-upgrade longer keys; the paged walk emits such a token and the next page fails. **Fix in phase:** validate only the token encoding (and the format's cursor bound), never `MaxKeyBytes`; adjust the send-ceiling proof. |
| WR-02 | warning | open | The Watch tap suppresses equal-version tie-break installs, contradicting `docs/API.md` and the store's tie-break law; a subscriber can miss a winner or remove-wins delete. **Fix in phase:** emit on a content change (RecordsEqual false), not on version inequality. |
| WR-03 | warning | open | `examples/client` sits inside the root module path prefix, so Go's internal rule does not stop it importing `internal/` — the "compile-time proof" claim is false. **Fix in phase:** give the example an out-of-prefix module path with a replace to the root so an internal import is rejected by the compiler; correct the README/comment; keep the depguard rule as enforcement. |
| IN-01 | info | open | Unreachable `ErrValueTooLarge` skip in `replayRecord` (same pattern as Phase-5 IN-05). **Fix in phase:** remove or justify. |
| IN-02 | info | open | Stale `wait.go:94-225` citation in the SPEC-DEVIATIONS row this phase edited. **Fix in phase:** repoint. |
| IN-03 | info | open | Transport doc claims "every gRPC setting is explicit"; keepalive is explicit but some defaults remain. **Fix in phase:** soften the claim to the settings actually pinned. |
| IN-04 | info | open | No largest-value transport round-trip test (research Pitfall 11) at `main.go:735`. **Fix in phase:** add the round-trip (or record a documented reason). |

**Legend:** `open` — seen, untriaged · `fixed` — resolved (record the commit) · `skipped` — will
not fix (say why) · `deferred` — postponed (name the phase).
