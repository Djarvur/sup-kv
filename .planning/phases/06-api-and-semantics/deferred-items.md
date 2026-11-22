# Phase 6 Deferred Items

Out-of-scope findings recorded during plan execution. None blocks 06-01; each is
for the owner named in the row.

| Found during | Item | Evidence | Owner |
|--------------|------|----------|-------|
| 06-01 metadata close-out | `roadmap.update-plan-progress 06` wrote its result into the **v1.0** phase-6 row of `ROADMAP.md` (`6. Hardening … | v1.0`) instead of the v1.1 row, because it matches by phase number and the table carries both milestones. Repaired by hand: the v1.0 row restored to `9/9 | Complete | 2026-11-16`, the v1.1 row set to `1/13 | In Progress`. | `git show 50436ae -- .planning/ROADMAP.md` | GSD tooling (roadmap handler) — every later phase-6 plan will hit it again |
| 06-01 metadata close-out | The v1.1 `4. Protocol` and `5. Persistence` rows in the same table still read `0/? | Not started` although both phases are complete (13/13 and 12/12, per their SUMMARYs). Pre-existing drift, not caused by this plan. | `.planning/ROADMAP.md` Progress table vs `.planning/phases/04-protocol/` and `05-persistence/` SUMMARY counts | Phase-gate/verification owner |
| 06-01 gate run | `go test -race ./api/supkvapi/v1/ ./internal/grpcapi/ ./internal/stand/ ./cmd/...` failed once in `cmd/sup-kv` `TestLog_formatText` with `bind: address already in use` (UDP probe port) while `internal/stand` bound its fixed test ports in a parallel package. Package-alone and `make test` are green. | `ok github.com/Djarvur/sup-kv/cmd/sup-kv 5.868s` standalone; `make test` exit 0 | Phase 8 flake owner (pre-existing cross-package port contention) |
