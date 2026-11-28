---
phase: 03-transport
plan: 10
subsystem: transport-socket-policy
tags: [xport-08, xport-09, read-loop, net-errclosed, so-reuseaddr, eaddrinuse, bind-address, peer-timeout, startup-warn]

# Dependency graph
requires:
  - phase: 03-transport
    provides: "03-01's socket construction and buffer read-back (configureSocketBuffers, checkSendBuffer) whose socket path this plan re-homes"
  - phase: 03-transport
    provides: "03-07's post-event mirror/atomic discipline the new ReadErrors accessor follows"
  - phase: 03-transport
    provides: "03-09's bounded startup envelope, so the bind/read-loop work starts from a node that refuses an unsendable configuration"
provides:
  - "The receive-path contract: only net.ErrClosed exits readLoop; every other error is counted in Adapter.ReadErrors, logged at most once per second, paused 100 ms and survived"
  - "The unexported packetConn seam plus newAdapterWithConn, letting a scripted fake drive the read path deterministically"
  - "Socket policy: production binds set no SO_REUSEADDR (a second instance fails EADDRINUSE at adapter and binary level); WithAddressReuse() is the explicit test-only opt-in"
  - "listenDatagram(bindAddr, options) replacing reuseAddrBroadcastListenPacket; SO_BROADCAST left to the Go runtime"
  - "startup bind-address WARN for a non-wildcard --bind-addr (never a rejection) and the one-shot no-Hello WARN after peer_timeout in Core.Advance"
affects:
  - "03-12 wires Adapter.ReadErrors into supkv_replication_read_errors_total (its plan already names the accessor this plan ships)"
  - "03-11 and later phases inherit a socket layer where a second process cannot silently share the node's port"
  - "Phase 4+ deployments: a unicast-only bind remains usable but is loudly explained at startup"

actuals:
  tokens: 10686
  tasks: 3
  commits: 4
plan_head_before: 83f31e0ae2c93585c8668bf1dc26371ea1afc93b
plan_head_after: 6d7d703a832a347955521799ef1121f06490543c

tech-stack:
  added: []
  patterns:
    - "A one-option interface makes the receive path testable without a live fault: the fake packetConn serves a scripted error then a datagram, and a persistent-error mode proves the 100 ms backoff is not a spin."
    - "The read-error log is rate-limited on the adapter's wall clock (one line per second) while the atomic counter carries the exact signal — a flood cannot flood the log."
    - "Socket policy is expressed as a functional option whose zero value is the production policy: `New(..., opts ...Option)` with WithAddressReuse test-only, so production callers cannot opt in by accident."
    - "The duplicate-instance proof is layered: an adapter-level test pins EADDRINUSE and the opt-in, and a binary-level exec test pins the operator-visible exit code and stderr text."
    - "Diagnostic WARNs are one-shot latched state on the driver goroutine: firstAdvance/helloSeen/peerWarned in the core, a single startup call in main — never a per-interval repeat."

key-files:
  created:
    - internal/udprepl/readloop_internal_test.go
    - internal/app/repl/warn_test.go
  modified:
    - internal/udprepl/repl.go
    - internal/udprepl/smoke_test.go
    - internal/udprepl/repl_test.go
    - internal/app/repl/core.go
    - cmd/sup-kv/main.go
    - cmd/sup-kv/main_test.go

key-decisions:
  - "The packetConn seam carries LocalAddr in addition to the plan's six methods. Construction derives the core's own opaque handle from the bound socket (D-03) and the existing white-box test reads it; adding the method keeps both on the seam instead of reintroducing a concrete *net.UDPConn field."
  - "The no-Hello WARN counts from the first Advance, not from Core construction: a core built long before its driver loop runs must not warn before it ever had a chance to read a datagram."
  - "helloSeen ignores the node's own broadcast echo (OwnEcho is discarded before handleHello): a lone node that hears only itself has discovered no peer and must still warn."
  - "The bind-address WARN is emitted in main between setupLogging and wire, so it is the first stderr line an operator sees and is never lost to a later construction failure."
  - "SO_BROADCAST is no longer set by the adapter: Go sets it on every datagram socket (RESEARCH Pattern 7), and the removal makes the option-gated SO_REUSEADDR the only socket option the production path can touch."
  - "The opt-in test binds both sockets with WithAddressReuse: Linux grants a UDP double-bind only when every socket on the address set the option, so a production first socket plus an opted-in second one must still fail — the asymmetry the policy relies on."

patterns-established:
  - "A resource-owning adapter with no Close method is released in tests by running its event loop and cancelling the context (Run's deferred socket close), via a runAdapter helper."
  - "Kernel failure classes are asserted both semantically and textually (errors.Is(syscall.EADDRINUSE) plus the 'address already in use' text), so a refactor cannot silently change the operator-visible message."

requirements-completed: [XPORT-08, XPORT-09]

coverage:
  - id: C1
    description: "Only net.ErrClosed exits the read loop: a scripted fake returns a non-closed error then a real datagram (processed, byte- and source-exact, after at least the 100 ms backoff) and a later Close returns the loop."
    tests: [TestReadLoop_survivesNonClosedError]
  - id: C2
    description: "A persistent read error neither exits nor spins: two errors are survived at the backoff cadence and Close still exits through net.ErrClosed, with a bounded read count."
    tests: [TestReadLoop_persistentErrorExitsOnClose]
  - id: C3
    description: "Production binds set no SO_REUSEADDR: a second adapter on the same address fails with syscall.EADDRINUSE and the kernel's text."
    tests: [TestListen_noAddressReuse]
  - id: C4
    description: "The test-only option restores same-port binding (both sockets opted in) and the real-socket two-node smoke test still passes with it."
    tests: [TestListen_noAddressReuse, TestSmoke_twoNodesUDP]
  - id: C5
    description: "A non-wildcard --bind-addr emits exactly one explanatory WARN and keeps serving; a wildcard bind emits none."
    tests: [TestMain_bindAddrWarn]
  - id: C6
    description: "An occupied UDP address makes the real binary exit 1 with 'address already in use' in its output."
    tests: [TestMain_duplicateBind]
  - id: C7
    description: "No Hello within peer_timeout of the first Advance WARNs exactly once, naming peer_timeout and its duration, never per interval; a Hello before the deadline suppresses it."
    tests: [TestCore_noPeerWarn]

duration: 925s
completed: 2026-11-22
status: complete
---

# Phase 03 Plan 10: Undying Read Loop and Loud Socket Misuse Summary

The UDP adapter can no longer go silently deaf: only `net.ErrClosed` ends the read loop, every other receive error is counted, rate-limit logged and backed off; production binds refuse a second instance with EADDRINUSE; and both a non-wildcard bind and never hearing a peer now WARN once, by design.

## Performance

- **Duration:** 15m 25s (925 s)
- **Tasks:** 3/3
- **Commits:** 4 (3 task commits + 1 lint-cleanup commit)
- **Files:** 8 (2 created, 6 modified)
- **Estimate vs actual:** 85 000 estimated tokens vs ~10 686 measured (diff chars/4). The estimate's low-confidence token figure was roughly 8× the realized diff; the task-count estimate (3) was exact.

## Accomplishments

**Task 1 — read loop survives every non-closed error.**
`Adapter.conn` is now the unexported `packetConn` interface (`ReadFromUDP`, `WriteToUDP`, `SetReadBuffer`, `SetWriteBuffer`, `SyscallConn`, `LocalAddr`, `Close`), `New` assigns the real `*net.UDPConn`, and `newAdapterWithConn` gives the white-box test a bare adapter over a scripted fake. `readLoop` exits only on `errors.Is(err, net.ErrClosed)`; every other error increments `readErrors`, logs at most once per second, sleeps `readErrorBackoff` (100 ms) and continues. `ReadErrors()` is the cross-goroutine counter 03-12 will publish. `readloop_internal_test.go` proves continuation (error then datagram, with the backoff observed) and bounded persistence (two errors survived, no spin, Close exits via ErrClosed).

**Task 2 — production SO_REUSEADDR removed, test-only opt-in.**
`reuseAddrBroadcastListenPacket` is replaced by `listenDatagram(bindAddr, options)`: SO_REUSEADDR is set through a pre-bind `Control` hook only when the resolved options ask for it, and SO_BROADCAST is deliberately left to the Go runtime. `New` takes functional options; `WithAddressReuse()` is the sole option and is documented test-only. `TestListen_noAddressReuse` pins both directions: a second production bind fails with `syscall.EADDRINUSE` and the kernel text, while two opted-in binds share the port. `TestMain_duplicateBind` drives the real binary against an occupied address and asserts exit 1 with "address already in use". The smoke test opts in and still passes on this Linux host (no skip).

**Task 3 — bind-address and no-Hello WARNs.**
`warnOnNonWildcardBind` runs in `main` after `setupLogging` and before `wire`: `""`, `0.0.0.0` and `::` are wildcards, any other host emits one WARN naming the broadcast-reception loss and the assumed unicast-only deployment — never a rejection. `Core` gained `firstAdvance`, `helloSeen` and `peerWarned`; `warnIfNoPeer` runs at the top of `Advance` and calls `cfg.Log` exactly once at `firstAdvance + PeerTimeout` when no Hello ever arrived, naming `peer_timeout` and its duration. `TestMain_bindAddrWarn` proves the WARN appears once for `127.0.0.1` and not at all for `0.0.0.0` while the node keeps serving; `TestCore_noPeerWarn` pins the one-shot cadence and the Hello-suppression case.

## Task Commits

| Task | Commit | Subject |
|------|--------|---------|
| 1 | `f91cae2` | feat(03-10): make the read loop survive every non-closed error |
| 2 | `e87f681` | feat(03-10): drop production SO_REUSEADDR for a test-only opt-in |
| — | `ca3913f` | refactor(03-10): keep the socket seam within the strict lint gate |
| 3 | `6d7d703` | feat(03-10): warn on non-wildcard binds and on hearing no peer |

## Files Created/Modified

**Created**
- `internal/udprepl/readloop_internal_test.go` — scripted `packetConn`, continuation and persistence cases.
- `internal/app/repl/warn_test.go` — one-shot no-Hello WARN and Hello suppression.

**Modified**
- `internal/udprepl/repl.go` — packetConn seam, ReadErrors/backoff/rate-limited log, options, listenDatagram, package doc.
- `internal/udprepl/smoke_test.go` — both adapters opt in with `WithAddressReuse()`.
- `internal/udprepl/repl_test.go` — `TestListen_noAddressReuse` and the `runAdapter` helper.
- `internal/app/repl/core.go` — firstAdvance/helloSeen/peerWarned and `warnIfNoPeer`.
- `cmd/sup-kv/main.go` — `warnOnNonWildcardBind` and its call site.
- `cmd/sup-kv/main_test.go` — `TestMain_bindAddrWarn`, `TestMain_duplicateBind`.

## Decisions Made

- The seam includes `LocalAddr` (see frontmatter): construction and an existing white-box test both need the bound address, and the alternative — a second concrete socket field — would have leaked the implementation the interface exists to hide.
- The opt-in double-bind test runs in two subtests on two ports: the plan's prose could be read as one sequence on one port, but Linux refuses an opted-in second bind while a non-opted-in first socket holds the address. That asymmetry is the policy, so both servers of a shared port must opt in.
- `SO_BROADCAST` is no longer set explicitly anywhere in production: Go already sets it for datagram sockets, and the removal leaves the option-gated reuse branch as the only socket-option touch point.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical functionality] `packetConn` needed `LocalAddr`**
- **Found during:** Task 1 implementation
- **Issue:** The plan's interface listed six methods, but `New` derives the core's own opaque handle from `conn.LocalAddr()` and `repl_internal_test.go` reads it after construction. Without `LocalAddr`, the only options were a concrete second field or a type assertion at both sites — both reintroduce the coupling the seam removes.
- **Fix:** Added `LocalAddr() net.Addr` to `packetConn`; `*net.UDPConn` and the fake both implement it.
- **Files modified:** `internal/udprepl/repl.go`, `internal/udprepl/readloop_internal_test.go`
- **Commit:** `f91cae2`

**2. [Rule 3 - Blocking] The listen result needs a checked assertion and a static sentinel**
- **Found during:** Task 2 implementation
- **Issue:** `net.ListenConfig.ListenPacket` returns `net.PacketConn`; the socket-buffer read-back needs the concrete `*net.UDPConn`. The required assertion had no error class, and the strict lint config (err113) forbids constructing errors at the site.
- **Fix:** `listenDatagram` returns `(*net.UDPConn, error)`, checks the assertion and wraps a new package-level `errUnexpectedSocketType`; the branch is unreachable with the standard library but returns instead of panicking.
- **Files modified:** `internal/udprepl/repl.go`
- **Verification:** full suite and lint green.
- **Commit:** `e87f681` (sentinel/lint shape finalized in `ca3913f`)

**3. [Rule 3 - Blocking] Lint-gate refactors on the socket seam**
- **Found during:** the first `go tool golangci-lint run ./...` after all tasks
- **Issue:** revive rejected the `reuse bool` control flag (flag-parameter) and the `bytes` parameter names shadowing the import; err113 rejected `errors.New` at test use sites; paralleltest required the new subtests to run parallel; noctx rejected `net.ListenPacket` in the binary test; sloglint required the Debug message capitalized.
- **Fix:** `listenDatagram` takes the resolved `options` struct; static test sentinels replace inline errors; the new test and subtests call `t.Parallel()`; the occupier uses `(&net.ListenConfig{}).ListenPacket(t.Context(), ...)`; the Debug msg is capitalized. Committed separately so the task commits keep their own subject.
- **Files modified:** `internal/udprepl/repl.go`, `internal/udprepl/readloop_internal_test.go`, `internal/udprepl/repl_test.go`, `cmd/sup-kv/main_test.go`
- **Verification:** `go tool golangci-lint run ./...` green.
- **Commit:** `ca3913f` (main_test.go portion in `6d7d703`)

**4. [Rule 1 - Test correctness] The opt-in case cannot share a port with a non-opted-in socket**
- **Found during:** Task 2's first test run
- **Issue:** A literal single-sequence reading of the plan (plain bind first, then opted-in bind on the same port) fails with EADDRINUSE: Linux grants a UDP double-bind only when every socket on the address set SO_REUSEADDR. The first run failed exactly here.
- **Fix:** `TestListen_noAddressReuse` splits into two subtests on separate ports: a production duplicate must fail, and two opted-in binds must succeed. The comment records the kernel rule.
- **Files modified:** `internal/udprepl/repl_test.go`
- **Commit:** `e87f681`

**5. [Note - plan option not taken] `TestStand_duplicateBind` not added**
- The plan explicitly prefers the binary-level main test and allows the stand test only if the main test cannot be deterministic. `TestMain_duplicateBind` passed repeatedly, so `internal/stand/stand_test.go` stayed untouched (it was listed as an optional file, not a requirement).

---

**Total deviations:** 4 auto-fixed (1 test-correctness, 1 missing seam method, 2 lint/construction blockers) + 1 explicitly untaken plan option.
**Impact on plan:** No scope change. Every must-have truth and acceptance criterion is met; the kernel double-bind rule and the seam extension are recorded so later plans do not re-litigate them.

## Issues Encountered

- **Intermediate commit red, restored immediately:** Task 1's commit (`f91cae2`) carries the whole `repl.go` rewrite, including the listener change that task 2 owns (same file), while `smoke_test.go` did not yet opt in. `TestSmoke_twoNodesUDP` was therefore red at that one commit and green from `e87f681` on. Final HEAD is fully green; the plan itself ordered the removal and the opt-in across tasks 1 and 2 and predicted this breakage.
- **Pre-existing pick-then-bind flake recurred once:** a combined `go test -race ./internal/udprepl/ ./internal/app/repl/ ./cmd/sup-kv/` run failed `TestLog_formatText` at its `probe rx bind` with EADDRINUSE. This is the harness race already recorded in `deferred-items.md` for 03-08 (freeUDPPort closes its throwaway socket before the test binds the chosen port). Three immediate `cmd/sup-kv` reruns and the final full `go test -race ./...` were green. The recurrence is appended to `deferred-items.md`; the harness was not fixed (out of scope).
- **No auth or setup gates:** none encountered; no `user_setup` entries.

## Threat Flags

None — the plan removes an attack surface (no address reuse in production) and adds only rate-limited WARN/log lines; no new endpoints, auth paths or trust boundaries.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- XPORT-08 and XPORT-09 are complete: the read loop has a single, tested exit; a second instance cannot bind; bind misuse and deafness are visible.
- 03-11 (dead-code removal) is unblocked: it owns `ownHandle`, which this plan deliberately kept because the adapter still passes the bound address into the core through `New`.
- 03-12 can wire `Adapter.ReadErrors()` directly into `supkv_replication_read_errors_total` as its plan already specifies; the accessor reads an atomic incremented on the read goroutine, so no further mirror work is needed for this counter.
- Operators on a unicast-only deployment get a single explanatory WARN at startup; an operator who sees the no-Hello WARN after `peer_timeout` should check `--broadcast-addr` and whether the bind address receives broadcasts.

## Self-Check: PASSED

- All 8 key files exist on disk (2 created, 6 modified).
- Measured commits from the persisted plan ledger: `git rev-list --count 83f31e0..6d7d703` = 4; `plan_head_before` = `83f31e0ae2c93585c8668bf1dc26371ea1afc93b`, `plan_head_after` = `6d7d703a832a347955521799ef1121f06490543c`.
- `gsd_run` is unavailable in this executor runtime; the close-out uses the persisted plan ledger, `git rev-list --count` and direct STATE/ROADMAP/REQUIREMENTS edits as the documented fallback.
- Task verifies re-run at close-out: `TestReadLoop*`, `TestListen_noAddressReuse`, `TestSmoke_twoNodesUDP`, `TestMain_bindAddrWarn`, `TestMain_duplicateBind`, `TestCore_noPeerWarn` all green.
- Plan verification re-run: `go test -race ./internal/udprepl/ ./internal/app/repl/ ./cmd/sup-kv/` green; the final full `go test -race -count=1 ./...` green; `grep -n "SO_REUSEADDR" internal/udprepl/*.go` shows one production `SetsockoptInt` inside the option-gated branch only; `go tool golangci-lint run ./...` green.
- Acceptance re-check: only `net.ErrClosed` exits; other errors are counted, logged (rate-limited) and backed off; production duplicate bind fails with EADDRINUSE at adapter and binary level; the test option restores sharing; a non-wildcard bind WARNs once and still serves; the no-Hello WARN fires exactly once and never after a Hello; no task test skips.

---
*Phase: 03-transport*
*Completed: 2026-11-22*
