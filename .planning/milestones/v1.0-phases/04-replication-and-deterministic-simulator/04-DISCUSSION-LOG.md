# Phase 4: Replication and Deterministic Simulator - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-11-03
**Phase:** 4-Replication and Deterministic Simulator
**Areas discussed:** Разделение кода (core/adapter seam), Устройство ядра (core concurrency), Симулятор (simulator form and scenarios), Реальная сеть (real-network coverage)

---

## Разделение кода (core/adapter seam)

### What crosses the core/adapter boundary?

| Option | Description | Selected |
|--------|-------------|----------|
| Структуры (совет) | Adapter fully decodes protobuf; core sees its own struct types; peer address an opaque handle | ✓ |
| Сырые байты | Bytes cross like Phase 3's dump seam; core would hand-parse protobuf | |

**User's choice:** Структуры
**Notes:** Protobuf cannot live in `internal/app` (depguard forbids `google.golang.org/*`), so the Phase 3 bytes-across precedent does not transfer; forced mechanically, the choice confirmed structs with the opaque-address peer table.

### Who decides "this datagram is garbage" and keeps the discard-class counters?

| Option | Description | Selected |
|--------|-------------|----------|
| Ядро (совет) | Decisions and per-class counters in the core; simulator covers the same logic | ✓ |
| Адаптер | Adapter drops bad datagrams itself; core sees only clean messages | |

**User's choice:** Ядро
**Notes:** Adapter stays mechanical (CRC check, envelope decode, no body decode on CRC failure); core owns all four discard classes (own echo, duplicate node_id, bad CRC, bad proto version).

---

## Устройство ядра (core concurrency)

### How is the replication core's internal loop structured?

| Option | Description | Selected |
|--------|-------------|----------|
| Один поток (совет) | One event-loop goroutine, one queue; only it mutates replication state | ✓ |
| Блокировки | Multiple goroutines with mutexes, storage-style | |

**User's choice:** Один поток
**Notes:** Determinism for the simulator falls out naturally; throughput a non-issue at project scale.

### How does a local write reach the broadcast path?

| Option | Description | Selected |
|--------|-------------|----------|
| Явный вызов (совет) | Write author explicitly calls replication; no subscriptions | ✓ |
| Подписка | Store notifies observers; order becomes implicit | |

**User's choice:** Явный вызов
**Notes:** Follows Phase 2 D-01 "results, not callbacks"; Phase 5's gRPC handler will call store + broadcast explicitly.

---

## Симулятор (simulator form and scenarios)

### What form does the simulator take?

| Option | Description | Selected |
|--------|-------------|----------|
| Тесты (совет) | Test-only package driven by go test; seed printed on failure; re-run via env var | ✓ |
| Программа cmd/sim | Standalone binary for long soaks; extra maintained surface | |

**User's choice:** Тесты
**Notes:** Same discipline as Phase 2 property tests; nothing simulator-specific ships in the binary.

### Which scenarios does the simulator run?

| Option | Description | Selected |
|--------|-------------|----------|
| Границы+хаос (совет) | Named test per documented bound + one seeded randomized chaos test | ✓ |
| Только границы | Only the named per-bound tests | |
| Только хаос | Only the seeded randomized test | |

**User's choice:** Границы+хаос
**Notes:** A failure in a named test names the broken promise; the chaos test catches unscripted fault combinations.

---

## Реальная сеть (real-network coverage)

### Does this phase include a real-socket network test?

| Option | Description | Selected |
|--------|-------------|----------|
| Дымовой тест (совет) | One localhost two-node UDP smoke test: Hello discovery + one replicated write | ✓ |
| Только симулятор | All verification virtual; real network first in Phases 5–6 | |

**User's choice:** Дымовой тест
**Notes:** Catches real protobuf serialization and real socket behavior the virtual network cannot; hostile real-network scenarios stay in Phase 6.

---

## Claude's Discretion

Package names/locations, port interface names and signatures, .proto file location and codegen wiring, message struct field spellings, seed environment-variable name, virtual network internal API, timer-tick delivery mechanism into the event loop.

## Deferred Ideas

- REPL-11 (DataSync unicast fast path) — conditional future work, already recorded in REQUIREMENTS.md
- Standalone `cmd/sim` long-run binary — rejected this phase; reconsider if randomized chaos outgrows CI budgets
