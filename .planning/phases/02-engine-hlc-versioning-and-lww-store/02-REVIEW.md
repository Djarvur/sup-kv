---
phase: 02-engine-hlc-versioning-and-lww-store
reviewed: 2026-11-14T18:55:27Z
depth: standard
files_reviewed: 2
files_reviewed_list:
  - internal/app/storage/store.go
  - internal/app/storage/store_test.go
findings:
  critical: 0
  warning: 0
  info: 3
  total: 3
  necessity: 0
status: issues_found
---

# Phase 02: Code Review Report (stale-verification re-run)

**Reviewed:** 2026-11-14T18:55:27Z
**Depth:** standard
**Files Reviewed:** 2
**Status:** issues_found (3 Info findings; 0 Critical, 0 Warning)

## Summary

Incremental re-review of the current homes of the phase-02 store after the
package-per-concern relocation. Delta since the last review (307f252,
2026-11-02) is commit c2746cb (phase 5-03): the `Config.OnChange` change tap
fired from `install()`, plus its tests. Both files were read in full and
traced against the phase-02 contract; siblings were cross-referenced
(`internal/app/clock/clock.go`, `internal/app/version/version.go`,
`internal/app/storage/errors.go`, `internal/app/storage/merge_test.go`,
`internal/app/storage/doc.go`) without being re-reviewed as scope.

The phase-02 contract holds in the current state, verified against the
source rather than the test suite:

- **Every change source converges on `install`** (`store.go:251-261`): local
  writes through `write` (`store.go:167-172`), remote merges through the
  strictly-greater branch and the missing-key branch of `join`
  (`store.go:226-242`). No path mutates `s.records` outside `install`, so
  the OnChange tap fires exactly when a record becomes the stored record,
  and stays silent for a losing merge and an equal-version redelivery
  (D-06) — confirmed by tracing the `join` switch, not just by
  `TestStore_onChange`.
- **Version order equals commit order**: `Set`, `SetIfAbsent` and `Delete`
  hold the write lock across `clock.Tick` and `Merge` across `clock.Update`
  (`store.go:155-158`, `store.go:183-189`, `store.go:196-201`,
  `store.go:215-220`). `clock.mu` is a leaf lock (no store calls inside the
  clock), so there is no lock inversion.
- **Exact loser pairing**: one `Conflict` per differing merge (winner =
  greater full 16-byte version), none on equal versions, nothing accumulated
  inside the store (`join`, `store.go:232-241`); the property laws remain
  intact in `merge_test.go` and pass.
- **60 KiB limit** (`defaultValueMaxBytes = 61440`, `store.go:15`) enforced
  on all three write paths before any state changes or clock advances;
  boundary tested at exactly-limit and limit+1 on each path
  (`TestStore_valueLimit`).
- **Recomputed checksums**: `install` ignores the incoming `rec.Checksum`
  and recomputes via `RecordChecksum` (`store.go:257`); the tap and `Get`
  agree on it (asserted in `TestStore_onChange`).
- **Defensive copying** holds on every path in and out (`install` clones on
  the way in, `Get`/`Records` clone on the way out), and records are
  replaced wholesale — never mutated in place — so the tap's documented
  "retaining the handed record is safe" contract is true.
- **Cross-file consistency**: the consumer-side `Clock` interface
  (`store.go:67-70`) signature-matches `clock.Clock.Tick`/`Update`;
  `PackVersion`/`Unpack`/`Compare` are total over the inputs the store
  feeds them; imports satisfy the `sup-kv-app-no-io` depguard boundary.
- **Necessity scan**: none found. The nil-`OnChange` normalization is
  load-bearing (`install` calls the callback unconditionally), the
  zero-`ValueMaxBytes` default serves the zero-value `Config` used by every
  test and by phase-5 construction sites, and no guard in either file
  defends a state the repository proves unreachable.
- **Tooling evidence**: `go test -race -count=1 ./internal/app/storage/`
  passes; `go vet`, `gofmt -l` and `go tool golangci-lint run` are clean on
  the package.

No Critical or Warning findings. The three Info findings are documentation
precision and test-duplication nits on the delta surface; none affects
behavior.

## Narrative Findings (AI reviewer)

## Info

### IN-01: OnChange doc misstates the record a "subsequent Get returns" for tombstone installs

**File:** `internal/app/storage/store.go:31-33`
**Issue:** The `Config.OnChange` contract says install fires "with the record
that just became the stored record — the exact record a subsequent Get
returns". For a tombstone install (local `Delete` or a Merge-delivered
winning tombstone), a subsequent `Get` returns `ErrNotFound` — the
tombstone is visible only through `Records()`. The tests themselves verify
the tombstone case against `Records()` (`store_test.go:1031-1040`), so the
comment misdescribes the API a phase-5 Watch-stream implementer will code
against.
**Fix:** Reword to name both surfaces, e.g.: "the exact record a subsequent
Records() snapshot holds — and that a subsequent Get returns when the
record is live; a tombstone install hands the tombstone Get cannot see."

### IN-02: OnChange contract does not state the re-entrancy deadlock boundary

**File:** `internal/app/storage/store.go:37-39`
**Issue:** The comment constrains the callback to "not block — fan-out
belongs behind a non-blocking channel send" but never says the callback
must not call back into the same store. `install` invokes `OnChange` while
the caller holds `s.mu` for writing (`store.go:215-220`, `store.go:251-261`);
Go's `sync.RWMutex` is not reentrant, so a tap that calls even the read-only
`Get` on the same store self-deadlocks, stalling every store operation. The
hazard is real but undocumented; a one-line contract addition closes it.
**Fix:** Extend the comment: "The callback must not call any method of the
same store — even Get deadlocks against the held write lock."

### IN-03: Duplicated subtest body between TestStore_endToEnd and TestStore_Set

**File:** `internal/app/storage/store_test.go:106-130`
**Issue:** The subtest "a second set replaces the value under a strictly
greater version" appears verbatim (same name, same scenario) in
`TestStore_endToEnd` (`store_test.go:106-130`) and `TestStore_Set`
(`store_test.go:228-247`), and both also overlap "two sets inside one
millisecond get distinct strictly increasing versions"
(`store_test.go:248-270`). A future contract change must be edited in two
places; the copies can drift.
**Fix:** Keep the endToEnd copy as the layer-wiring probe (it additionally
asserts the unpacked `(l, counter, nodeID)`) and rename or fold the
`TestStore_Set` copy into the same-millisecond subtest, or vice versa —
one canonical home for the scenario.

---

_Reviewed: 2026-11-14T18:55:27Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
