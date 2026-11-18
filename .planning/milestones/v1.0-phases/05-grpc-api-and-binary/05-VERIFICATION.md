---
phase: 05-grpc-api-and-binary
verified: 2026-11-18T11:03:39Z
status: passed
score: 28/28 must-haves verified
covered_files: [".planning/phases/05-grpc-api-and-binary/05-01-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-01-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-02-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-02-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-03-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-03-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-04-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-04-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-05-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-05-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-06-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-06-SUMMARY.md", "Makefile", "buf.gen.api.yaml", "buf.gen.repl.yaml", "buf.yaml", "cmd/sup-kv/main.go", "cmd/sup-kv/main_test.go", "go.mod", "go.sum", "internal/app/persist/engine.go", "internal/app/persist/engine_test.go", "internal/app/repl/core.go", "internal/app/repl/retransmit_test.go", "internal/app/repl/sync_test.go", "internal/app/storage/store.go", "internal/app/storage/store_test.go", "internal/grpcapi/apiproto/supkvapi/v1/supkvapi.pb.go", "internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto", "internal/grpcapi/apiproto/supkvapi/v1/supkvapi_grpc.pb.go", "internal/grpcapi/bench_test.go", "internal/grpcapi/gate.go", "internal/grpcapi/gate_test.go", "internal/grpcapi/server.go", "internal/grpcapi/server_test.go", "internal/grpcapi/wait.go", "internal/grpcapi/wait_integration_test.go", "internal/grpcapi/wait_test.go", "internal/grpcapi/watch.go", "internal/grpcapi/watch_test.go", "internal/udprepl/repl.go", "internal/udprepl/repl_test.go"]
covered_digest: "v4:sha256:9e1cb10195a82727dfa38b4ec54c1eafa3a44dd9cc66505600caef45574c35dd"
behavior_unverified: 0 # Count of ⚠️ PRESENT_BEHAVIOR_UNVERIFIED truths (present + wired, behavior not exercised); each is detailed in behavior_unverified_items below (and in human_verification when status is human_needed)
overrides_applied: 0 # Count of PASSED (override) items included in score
re_verification:
  previous_status: passed
  previous_score: 28/28
  previous_verified: 2026-11-16T10:56:44Z
  reason: "stale fingerprint — phase-7 commits (через выпуск v1.0.0) тронули covered inputs после прохода 2026-11-16 (HEAD e47733c): Makefile — аддитивные release/release-snapshot targets (ac14f92, фаза-7 D-09); cmd/sup-kv/main.go — reflection.Register(gs) в listenGRPC + doc (фаза-7 D-09, grpcurl-интерактивность); go.mod/go.sum — директива go 1.27.4 (поглощена toolchain) и два новых tool-директивы grpcurl/goreleaser с их косвенными графами. Весь internal/ + main_test.go байт-идентичны e47733c (git diff пуст)."
  gaps_closed: []
  gaps_remaining: []
  regressions: []
deferred:
  - truth: "systemd unit behavior end-to-end (TOOL-01: Type=simple unit + SIGTERM lifecycle observed on a real host)"
    addressed_in: "Phase 7"
    evidence: "Phase 7 goal: 'A stranger can deploy and operate a two-node sup-kv cluster from the documentation alone'; 05-06-PLAN defers systemd documentation to Phase 7; Phase 7 исполнена и отшипована (v1.0.0 на HEAD): docs/OPERATIONS.md документирует systemd-контракт бинарня (exit code 1 → Restart=on-failure, exit code 2 → конфигурация), остаточные human-items фазы 7 управляются её собственным VERIFICATION/UAT (07-UAT.md stranger cold-read). Бинарная сторона контракта (SIGTERM → exit 0 после финального dump, health readiness, коды 0/1/2, второй SIGTERM → force-stop с сохранением dump) поведенчески переподтверждена свежими прогонами TestMain_lifecycle и TestShutdown_secondSignal этой верификации на HEAD e9eeecc."
advisory:
  - finding: "WR-01 (05-REVIEW, записан 2026-11-18 — позже прошлого прохода): gate-резолвер может вернуть health в SERVING посреди drain — apiCancel стоит после metrics-shutdown и расширяет окно гонки; резидуальный вариант переживает apiCancel через случайный выбор select"
    category: architectural
    reason: "Открытая warning-находка рабочего процесса code-review со своим triage-циклом (disposition: open в 05-REVIEW-DISPOSITION.md). Не фальсифицирует ни одну из 28 истин: истина 13 (gate↔health 1:1 до/после открытия) доказана свежим прогоном TestHealth_reflectsGate (PASS), истина 20 (exit 0 + финальный dump) — свежими TestMain_lifecycle/TestShutdown_secondSignal; транзиент mid-drain вне формулировок истин. Сайты находки (gate.go целиком; shutdown-порядок main.go:645-654) байт-идентичны дважды верифицированному коду — drift main.go с e47733c не касается shutdown (только listenGRPC/import/doc). Детерминистичного evidence нет (ни одного красного теста, ни воспроизводимого отказа). Hardening: gateDone-канал + сериализация RunGate-exit до NOT_SERVING-flip — по triage WR-01 в review-цикле."
    evidence_status: "none provided"
  - finding: "IN-02 (05-REVIEW, записан 2026-11-18): serve-error exit path пропускает финальный Flush и graceful drain"
    category: other
    reason: "Открытая info-находка review-цикла, disposition: open. Не must-have фазы 5 (провал listen — деградированный, но чистый exit; подтверждённых данных на этом пути ещё нет). Детерминистичного evidence нет."
    evidence_status: "none provided"
behavior_unverified_items: []
coincidental_reliance_items: []
decision_coverage:
  honored: 19
  total: 19
  not_honored: []
---

# Phase 5: gRPC API and Binary — Verification Report

**Phase Goal:** sup-kv becomes a usable product — a single static binary exposing the typed gRPC surface with Watch, startup gate, `wait_for_replication` and standard health checking
**Verified:** 2026-11-18T11:03:39Z
**Status:** passed
**Re-verification:** Yes — stale-fingerprint regeneration (прошлый проход 2026-11-16, HEAD e47733c, passed 28/28 без гэпов; фаза-7 релизные коммиты тронули 4 covered-файла и инвалидировали digest). Контракт — переименованный baseline (SupKVService / WatchResponse / apiproto/supkvapi/v1, решение мейнтейнера 2026-11-14) на текущем HEAD e9eeecc (v1.0.0).

## Drift Contract (что изменилось с прошлого прохода)

Контракт этой проверки — текущее дерево на HEAD e9eeecc. Отдифферовано против e47733c (HEAD прошлого прохода) по всем covered inputs:

- **Весь внутренний слой фазы байт-идентичен**: `git diff e47733c..HEAD -- internal/ cmd/sup-kv/main_test.go` — пусто. Вся поверхность хендлеров (server.go/gate.go/watch.go/wait.go), wire-контракт (apiproto/), движок, store, repl-ядро, адаптер и все тестовые файлы не менялись с дважды верифицированного состояния; истины 1-19, 23, 25-28 опираются на неизменённый код, перепроверенный свежим `-race` прогоном.
- **Makefile (+17 строк, фаза-7 D-09)**: аддитивные `release` / `release-snapshot` targets (go.mod-pinned goreleaser) + расширение .PHONY. Build/proto/bench targets неизменны — перечитано; `make build` исполнен в этой верификации.
- **cmd/sup-kv/main.go (+10/-4 строк, фаза-7 D-09)**: `reflection.Register(gs)` добавлен в listenGRPC (main.go:591) ПОСЛЕ фаз-5 регистраций RegisterSupKVServiceServer (589) + RegisterHealthServer (590); плюс doc-комментарий и import. Фаз-5 проводка не тронута и переконфирмлена grep'ом на HEAD: `go srv.RunGate(apiCtx)` (462), `go hub.Run(apiCtx)` (463), пять ServerOptions-швов (454-458), D-19 shutdown (NOT_SERVING на 645 → bounded GracefulStop на 658 → Flush → exit-коды 0/1/2). Дрейф поведенчески перепокрыт свежими TestMain_lifecycle / TestMain_precedence / TestShutdown_secondSignal (реальные процессы/TCP через изменённый listenGRPC — PASS).
- **go.mod/go.sum (фаза-7)**: директива `go 1.27.4` (toolchain-строка поглощена), два новых tool-директива — grpcurl и goreleaser (последний тянет большой косвенный граф, весь `// indirect`, в runtime-граф бинарня не входит). Прямые runtime-require неизменны: grpc v1.85.0 + protobuf v1.36.13-pseudo (grep этой верификации); protoc-gen-go-grpc v1.6.2 в tool-блоке; tidy-гейт exit 0 (исполнено).
- **Новые фаз-5 документы после прошлого прохода**: 05-REVIEW.md / 05-REVIEW-DISPOSITION.md перегенерированы 2026-11-18 — CR-01/WR-02 остаются fixed, WR-01/IN-02 записаны как open (см. Advisory), IN-01 переносится без изменений. Планировочные документы inert (#4623), в covered-набор не входят.

## Goal Achievement

### Observable Truths

Те же 28 истин, что в прошлом проходе (SC1-SC5 ROADMAP + truths планов 05-01..05-06), перепроверенные на HEAD e9eeecc. Каждая поведенческая истина имеет свежий прогон (`-count=1 -race`) или исполненную команду из ЭТОЙ верификации.

| # | Truth | Source | Status | Evidence |
|---|-------|--------|--------|----------|
| 1 | gRPC Set затем Get возвращает идентичное значение и 16-байтовую версию (D-01) | SC1, 05-01 | ✓ VERIFIED | Свежий -race прогон пакета grpcapi: ok 2.324s, вкл. TestServer_setGetRoundTrip; server.go байт-идентичен верифицированному состоянию; Record{key,version,value,deleted} в apiproto/supkvapi/v1 (proto перечитан: все 7 RPC на месте) |
| 2 | Каждый сентинел маппится на закреплённый RPC-код через errors.Is (NotFound/AlreadyExists/InvalidArgument/Unavailable×2/InvalidArgument для имён снапшотов) | SC1, 05-01 | ✓ VERIFIED | Свежий прогон вкл. TestServer_errorMapping; единственный mapError в неизменённом server.go |
| 3 | create_if_absent=true возвращает AlreadyExists для любой существующей записи, включая tombstoned | SC1, 05-01 | ✓ VERIFIED | TestServer_errorMapping (live + tombstoned строки) в свежем прогоне |
| 4 | List возвращает сортированные live-ключи под префиксом; пустой префикс = все ключи; без пагинации (D-12..D-15) | SC1, 05-01 | ✓ VERIFIED | TestServer_list в свежем прогоне; в proto нет пагинационных полей (перечитано) |
| 5 | go.mod/go.sum tidy-exact; grpc v1.85.0 direct require; protoc-gen-go-grpc v1.6.2 tool-директива | 05-01 | ✓ VERIFIED | `go mod tidy && git diff --exit-code go.mod go.sum` exit 0 (исполнено в этой верификации); grpc v1.85.0 direct (go.mod:16), protoc-gen-go-grpc v1.6.2 (go.mod:11/544); фаза-7 tool-директивы grpcurl/goreleaser аддитивны, косвенный граф в runtime не входит |
| 6 | Watch-подписчик сначала получает текущее состояние как обычные события (tombstones как deleted=true), затем живые изменения — без дублей и пропусков (D-02/D-03) | SC2, 05-03 | ✓ VERIFIED | TestWatch_snapshotThenLive + TestWatch_exactCut в свежем -race прогоне пакета grpcapi (вкл. всё семейство TestWatch_*) |
| 7 | События только при реальных install; проигравшая/равная версия не даёт события (D-06) | SC2, 05-03 | ✓ VERIFIED | TestStore_onChange в свежем прогоне storage (ok 1.690s); TestWatch_replicatedChange в свежем прогоне grpcapi |
| 8 | Локальные и реплицированные изменения в одном потоке в порядке прибытия, неразличимы по содержимому (D-05/D-07) | SC2, 05-03 | ✓ VERIFIED | TestWatch_replicatedChange в свежем прогоне; toProto без origin-поля в неизменённом watch.go |
| 9 | Медленный потребитель отключается (ResourceExhausted), store продолжает принимать записи — tap никогда не блокирует write lock | SC2, 05-03 | ✓ VERIFIED | TestWatch_slowConsumer в свежем -race прогоне; select-with-default bounded ingress в watch.go |
| 10 | Prefix-фильтр применяется к snapshot и live-событиям; пустой префикс = все ключи (D-04) | SC2, 05-03 | ✓ VERIFIED | TestWatch_prefix в свежем прогоне |
| 11 | До открытия gate Set/Delete возвращают Unavailable, Get/List/Watch работают (API-04) | SC3, 05-04 | ✓ VERIFIED | TestGate_writesUnavailableReadsOK в свежем прогоне; gate-проверка — первый statement Set/Delete в неизменённом server.go |
| 12 | Gate открывается по более раннему из sync-flip или startup_gate (A8) | SC3, 05-04 | ✓ VERIFIED | TestGate_opensOnSync + TestGate_opensOnTimeout в свежем прогоне; RunGate 20ms poll + timer-backstop в неизменённом gate.go |
| 13 | Стандартный health-endpoint: NOT_SERVING для пустого имени сервиса до gate, SERVING после — связь 1:1 (API-06) | SC3, 05-04 | ✓ VERIFIED | TestHealth_reflectsGate в свежем прогоне; регистрация health рядом с SupKVService перечитана на HEAD (main.go:589-590; reflection.Register:591 — аддитивен, фаз-7 D-09, см. Drift Contract) |
| 14 | SnapshotCreate/SnapshotLoad гейтятся как записи; ErrInvalidSnapshotName -> InvalidArgument (A5, API-07) | SC3/SC4, 05-04 | ✓ VERIFIED | TestSnapshot_gated/_invalidName/_createLoadRoundTrip в свежем прогоне |
| 15 | wait_for_replication отчитывает D-08 агрегат и никогда не меняет исход записи (API-05) | SC4, 05-05 | ✓ VERIFIED | TestWait_timeout (Set RPC OK, acked=false 0/1) + TestWait_acksArrive (1/1) в свежем прогоне |
| 16 | Ноль известных пиров — acked=true немедленно, total_peers 0 (D-11), включая nil-швы | SC4, 05-05 | ✓ VERIFIED | TestWait_zeroPeers в свежем прогоне |
| 17 | Эвиктированный пир никогда не считается Ack; superseded-запись — acked=false с замороженными счётчиками (Q2) | SC4, 05-05 | ✓ VERIFIED | TestWait_evictionNotAck + TestWait_superseded в свежем прогоне |
| 18 | Wait ждёт появления (key, version) перед трактовкой отсутствия как drained — без ложного acked из notify-queue race (Pitfall 4) | SC4, 05-05 | ✓ VERIFIED | TestWait_appearanceRace + TestWaitIntegration_ackedThroughRealAdapterMirror в свежем -race прогоне |
| 19 | Timeout — per-request wait_timeout_ms если положителен, иначе операторский дефолт (D-09), overflow-safe (WR-02) | SC4, 05-05 | ✓ VERIFIED | TestWait_perRequestTimeout + TestWait_overflowingTimeoutClamped в свежем прогоне |
| 20 | SIGTERM даёт чистый exit 0 после финального write-confirmed dump; подтверждённые данные никогда не остаются только в памяти (D-19) | SC5, 05-06 | ✓ VERIFIED | Свежий прогон `go test -race -count=1 -run 'TestMain_\|TestShutdown_' ./cmd/sup-kv/ -v`: 4/4 PASS (TestMain_lifecycle 0.14s — реальные процесс/TCP: SERVING → Set/Get → SIGTERM → exit 0 → dump содержит ключ; TestShutdown_secondSignal 1.14s); shutdown-последовательность D-19 перечитана на HEAD — reflection-дрейф её не тронул (diff подтверждает: изменения только в listenGRPC/import/doc) |
| 21 | Каждая ручка — flag и SUPKV_ env с приоритетом flag > env > default (D-16/D-17) | SC5, 05-06 | ✓ VERIFIED | TestMain_precedence PASS (свежий); registerConfig байт-идентичен e47733c (дрейф main.go — только listenGRPC); `./bin/sup-kv` без ручек — exit 2 с usage, SUPKV_WAIT_TIMEOUT=abc — exit 2 (оба исполнены в этой верификации) |
| 22 | --version печатает version + commit + Go runtime, ldflags-инъекция, dev-фолбэк (D-18) | SC5, 05-06 | ✓ VERIFIED | `./bin/sup-kv --version` → `sup-kv e9eeecc e9eeecc go1.27.4` (исполнено; make build на HEAD); TestMain_version PASS (свежий) |
| 23 | Engine.Flush() превращает остаточную dirty (remote merges без waiter) в подтверждённый dump до отмены ctx движка | SC5, 05-06 | ✓ VERIFIED | Свежий прогон persist: ok 2.374s вкл. TestFlush_remoteDirtiness/_closed/_clean; engine.go байт-идентичен |
| 24 | Поставляемый артефакт — один статический CGO_ENABLED=0 бинарй | SC5, 05-06 | ✓ VERIFIED | `make build` + `file bin/sup-kv` → "statically linked" (исполнено); build target Makefile неизменен (релизные targets аддитивны; goreleaser-артефакты — фаза-7 scope, тот же статический binary из того же target) |
| 25 | PendingSnapshot различает пира, который Acked (в Acked), от эвиктированного по peer_timeout (вне обоих множеств) | 05-02 | ✓ VERIFIED | Свежий прогон repl: ok 2.751s вкл. TestRetransmit_ackDrainSemantics; core.go байт-идентичен |
| 26 | Core.SyncArrived() false до первого DataSync, true навсегда после (A1) | 05-02 | ✓ VERIFIED | TestSync_syncArrived в свежем прогоне repl |
| 27 | gRPC-читатели получают оба состояния репликации из любой горутины через атомики адаптера, не через single-driver core | 05-02 | ✓ VERIFIED | Свежий прогон udprepl: ok 1.354s вкл. TestMirror_pendingAckVisible + TestMirror_syncArrivedFlip; repl.go байт-идентичен |
| 28 | В core не добавлено коллбеков и каналов — результаты, не коллбеки (дисциплина port.Outbox) | 05-02 | ✓ VERIFIED | core.go байт-идентичен дважды верифицированному состоянию (git diff e47733c..HEAD пуст — включая санкционированную фаза-6 OnDiscard-запись, задокументированную в прошлом проходе) |

**Score:** 28/28 truths verified (0 present, behavior-unverified)

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | systemd unit end-to-end (реальный хост, Type=simple unit) | Phase 7 | Фаза 7 исполнена и отшипована (v1.0.0 на HEAD): docs/OPERATIONS.md документирует systemd-контракт (exit 1 → Restart=on-failure реагирует, exit 2 → конфигурация); остаточные human-items фазы 7 — в её собственном VERIFICATION (human_needed) / 07-UAT.md (stranger cold-read). Контракт бинарня на стороне юнита поведенчески доказан свежими прогонами этой верификации. |

### Advisory (New Scope, Unevidenced)

Находки Step 7 без детерминистичного evidence — доложены, не блокируют (#3304). Обе — открытые записи перегенерированного 2026-11-18 фаз-5 code-review (05-REVIEW-DISPOSITION.md), их triage принадлежит review-циклу.

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| 1 | WR-01: gate-резолвер может вернуть health в SERVING посреди drain (apiCancel после metrics-shutdown; резидуальный select-race) | architectural | Записана после прошлого прохода; не фальсифицирует истины 13/20 (свежие TestHealth_reflectsGate / TestMain_lifecycle / TestShutdown_secondSignal PASS); сайты (gate.go, shutdown-порядок) байт-идентичны дважды верифицированному коду, main.go-дрейф их не касается; ни одного красного теста / воспроизводимого отказа |
| 2 | IN-02: serve-error exit path пропускает финальный Flush и graceful drain | other | Открытая info-находка review-цикла; деградированный, но чистый exit без подтверждённых данных; детерминистичного evidence нет |

Перенос из прошлого прохода: IN-01 (недостижимая mapError status-пасsthru ветка, server.go:306-308) — info, disposition open, файл байт-идентичен, статус не меняет.

### Required Artifacts

`verify.artifacts` гейт-запрос возвращает total 0 для строковой формы must_haves.artifacts планов (формат-варианс, как во всех прошлых проходах) — все артефакты проверены вручную. Все internal/-артефакты и main_test.go байт-идентичны верифицированному состоянию; Makefile/main.go перечитаны в зоне дрейфа:

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto` | wire-контракт фазы, 7 RPC, service SupKVService, WatchResponse, живые wait-поля | ✓ VERIFIED | Перечитан: все 7 RPC (Get/Set/Delete/List/Watch-streaming/SnapshotCreate/SnapshotLoad), create_if_absent/wait_for_replication/wait_timeout_ms/D-08 поля/prefix; buf lint + format exit 0 (исполнено) |
| `internal/grpcapi/apiproto/supkvapi/v1/supkvapi.pb.go`, `supkvapi_grpc.pb.go` | закоммиченный сгенерированный код | ✓ VERIFIED | Байт-идентичны (diff пуст); RegisterSupKVServiceServer × 80 упоминаний SupKVService в grpc-файле |
| `internal/grpcapi/server.go`, `watch.go`, `gate.go`, `wait.go` | хендлеры + хаб + gate + wait | ✓ VERIFIED | Байт-идентичны верифицированному состоянию; вся поверхность перепокрыта свежим -race прогоном пакета |
| `internal/app/storage/store.go` | Config.OnChange tap из install | ✓ VERIFIED | Байт-идентичен; TestStore_onChange в свежем прогоне |
| `internal/app/repl/core.go` | SyncArrived + PendingInfo.Acked | ✓ VERIFIED | Байт-идентичен; TestRetransmit_ackDrainSemantics + TestSync_syncArrived в свежем прогоне |
| `internal/udprepl/repl.go` | зеркала адаптера | ✓ VERIFIED | Байт-идентичен; TestMirror_* в свежем прогоне |
| `internal/app/persist/engine.go` | экспортированный Flush | ✓ VERIFIED | Байт-идентичен; TestFlush_* в свежем прогоне |
| `cmd/sup-kv/main.go` | только flags/env/signals/wiring (+ фаза-6 observability, + фаза-7 reflection) | ✓ VERIFIED | Дрейф с e47733c = reflection.Register + doc + import (10 строк, diff прочитан построчно); вся фаз-5 проводка переконфирмлена grep'ом на HEAD: 19 ручек в байт-идентичном registerConfig, D-19 shutdown (645-679), ServerOptions-швы (454-458), регистрации (589-591), RunGate/hub.Run (462-463) |
| `cmd/sup-kv/main_test.go` | семейство smoke бинарня | ✓ VERIFIED | Байт-идентичен; свежий прогон 4/4 PASS |
| `internal/grpcapi/bench_test.go` | фаза-6 D-13 бенчмарки | ✓ VERIFIED | Байт-идентичен; участвует в свежем прогоне пакета |
| `Makefile`, `go.mod`, `go.sum`, `buf.gen.api.yaml`, `buf.gen.repl.yaml`, `buf.yaml` | build/codegen/deps | ✓ VERIFIED | Makefile: аддитивные release/release-snapshot (фаза-7), build/proto targets неизменны — перечитано; go.mod: grpc v1.85.0 direct, protoc-gen-go-grpc v1.6.2, go 1.27.4, аддитивные grpcurl/goreleaser tool-директивы; tidy-exact (исполнено); buf.*.yaml байт-идентичны, buf-гейты зелёные |

Проводка (import + использование) подтверждена: все артефакты задействованы main.go, харнессом или собственными потребителями пакетов; orphan-файлов нет.

### Key Link Verification

Все key_links шести планов. Ссылки на байт-идентичных файлах унаследованы от дважды верифицированного состояния; ссылки на дрейфнувших файлах (main.go, Makefile, go.mod) перечитаны на текущем HEAD:

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| Хендлеры (Get/Set/Delete/List/Snapshot) | persist.Engine | прямые вызовы движка | ✓ WIRED | server.go (байт-идентичен) |
| Server notify-шов | udprepl.Adapter.NotifyLocalWrite | LocalNotify, nil-safe | ✓ WIRED | main.go:452 инъектирует адаптер (перечитано на HEAD) |
| storage.Store.install | Config.OnChange → WatchHub.Ingest | tap → hub ingress | ✓ WIRED | store.go байт-идентичен; OnChange: hub.Ingest в startEngine |
| Exact-cut подавление | RecordVersion.Compare | строгое доминирование | ✓ WIRED | watch.go (байт-идентичен) |
| ServerOptions.SyncArrived → gate resolver → gated() | зеркало адаптера | poll + флаг | ✓ WIRED | main.go:454 (перечитано); RunGate (462); Set/Delete читают gated() |
| Разрешение gate → health.SetServingStatus | связь 1:1 | одиночный CAS | ✓ WIRED | gate.go (байт-идентичен); health зарегистрирован main.go:590 рядом с SupKVService (589); reflection.Register (591) — аддитивная третья регистрация, порядок фаз-5 регистраций не изменён |
| waitForReplication | KnownPeers/PendingCopy швы | только инъектированные замыкания | ✓ WIRED | wait.go; main.go:455-456 + WaitTimeout (458) (перечитано) |
| Snapshot-хендлеры | валидация имён persistfile | единственный закрытый путь | ✓ WIRED | server.go → engine.Snapshot/Restore |
| handleAck/handleDataSync → core → зеркала адаптера | post-iteration mirrorState | refresh после каждого события | ✓ WIRED | repl.go + core.go (байт-идентичны) |
| Граф проводки main | hub → engine → adapter → server → gate/health → serve | полный цикл | ✓ WIRED | main.go: RunGate (462), hub.Run (463), listenGRPC регистрации (589-591) — перечитано на HEAD; TestMain_lifecycle проходит весь граф end-to-end (свежий PASS) |
| SIGTERM → порядок shutdown | D-19 шесть шагов | последовательность | ✓ WIRED | main.go:645-679 (перечитано на HEAD, дрейф не коснулся): NOT_SERVING → metrics-listener down (фаза-6) → apiCancel → bounded GracefulStop/Stop (второй сигнал — force, TestShutdown_secondSignal PASS) → replCancel+join → Flush → engineCancel + exit 0/1 |
| Makefile build | статический штампованный bin/sup-kv | CGO_ENABLED=0 + ldflags | ✓ WIRED | Исполнено: statically linked, `sup-kv e9eeecc e9eeecc go1.27.4`; build target неизменен |
| tool-блок go.mod | protoc-gen-go-grpc через go tool | buf-плагины | ✓ WIRED | go.mod tool-блок (protoc-gen-go-grpc v1.6.2 на месте; grpcurl/goreleaser — аддитивные фаза-7 записи); buf.gen.*.yaml local go-tool плагины байт-идентичны |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| Get-хендлер | record | engine.Get (persist поверх реального store) | Yes | ✓ FLOWING |
| List-хендлер | keys | engine.Records() snapshot, фильтр/сортировка | Yes | ✓ FLOWING |
| Watch-поток | events | storage install tap → hub ingress → подписчики | Yes | ✓ FLOWING |
| Wait-отчёт | acked/counts | инъектированные зеркала адаптера (core-acknowledged состояние) | Yes | ✓ FLOWING |
| Health-статус | gate atomic | resolver openGate / shutdown NOT_SERVING | Yes | ✓ FLOWING |
| Set-ответ | version | engine-штампованный RecordVersion | Yes | ✓ FLOWING |
| --version вывод | version/commit | ldflags -X (git describe) | Yes | ✓ FLOWING (исполнено: e9eeecc / e9eeecc / go1.27.4) |

Статических return, hardcoded-литералов и моков в продакшн-путях ответов нет. (Тестовые fakes — только в _test.go.)

### Behavioral Spot-Checks

Все команды исполнены в ЭТОЙ верификации на HEAD e9eeecc (рабочее дерево чистое):

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Полный фаза-5 срез пакетов (grpcapi вкл. TestServer_/TestWatch_/TestGate_/TestHealth_/TestSnapshot_/TestWait_/TestWaitIntegration_; storage; persist вкл. TestFlush_; repl; udprepl вкл. TestMirror_*) | `go test -race -count=1 ./internal/grpcapi/... ./internal/app/storage/ ./internal/app/persist/ ./internal/app/repl/ ./internal/udprepl/` | все ok (grpcapi 2.324s, storage 1.690s, persist 2.374s, repl 2.751s, udprepl 1.354s) | ✓ PASS |
| Бинарй: lifecycle/precedence/version + shutdown force-path | `go test -race -count=1 -run 'TestMain_\|TestShutdown_' ./cmd/sup-kv/ -v` | 4/4 PASS (TestMain_version 0.01s, TestMain_precedence 0.10s, TestMain_lifecycle 0.14s, TestShutdown_secondSignal 1.14s; реальные процессы/TCP через изменённый reflection-listenGRPC) | ✓ PASS |
| Статический артефакт + версия | `make build && file bin/sup-kv && ./bin/sup-kv --version` | statically linked; `sup-kv e9eeecc e9eeecc go1.27.4` | ✓ PASS |
| Коды выхода | `./bin/sup-kv` (без ручек); `SUPKV_WAIT_TIMEOUT=abc ./bin/sup-kv ...` | exit 2 с usage; exit 2 (переменная названа) | ✓ PASS |
| Tidy-гейт | `go mod tidy && git diff --exit-code go.mod go.sum` | exit 0 | ✓ PASS |
| buf-гейты (zero-waiver) | `go tool buf lint && go tool buf format --diff --exit-code` | оба exit 0 | ✓ PASS |
| Rename-контракт | grep WatchEvent/RegisterSupKVServer/SupKVServer/NewSupKVClient в не-тестовых не-генерированных Go/proto | ноль совпадений; SupKVService × 80 / WatchResponse × 15 на месте | ✓ PASS |
| Идентичность внутреннего слоя | `git diff e47733c..HEAD -- internal/ cmd/sup-kv/main_test.go` | пусто (байт-идентично дважды верифицированному состоянию) | ✓ PASS |

### Probe Execution

Фаза не объявляет probe-скриптов (`scripts/*/tests/probe-*` не существуют); runnable-check контракт фазы — сам Go-набор, покрытый выше свежими прогонами.

### Requirements Coverage

ID из планов: 05-01 [API-01, API-02], 05-02 [API-04, API-05], 05-03 [API-03], 05-04 [API-04, API-06, API-07], 05-05 [API-05], 05-06 [TOOL-01]. Объединение = ровно 8 ID фазы. REQUIREMENTS.md (Traceability, строки 173-180) маппит ровно API-01..API-07 + TOOL-01 на Phase 5, статус Complete — сиротских требований нет, незаявленных ID нет.

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| API-01 | 05-01 | gRPC-сервис с protobuf; Get/Set (incl. create-if-absent)/Delete с полным набором ошибок | ✓ SATISFIED | Истины 1-3; свежий прогон grpcapi вкл. TestServer_errorMapping |
| API-02 | 05-01 | List ключей под префиксом в сортированном порядке | ✓ SATISFIED | Истина 4; TestServer_list (свежий PASS) |
| API-03 | 05-03 | Watch-поток, покрывающий локальные и реплицированные изменения | ✓ SATISFIED | Истины 6-10; семейство TestWatch_* + TestStore_onChange (свежие PASS) |
| API-04 | 05-02, 05-04 | Записи Unavailable во время startup gate; чтения немедленно | ✓ SATISFIED | Истины 11-12, 25-26; TestGate_* + TestSync_syncArrived (свежие PASS) |
| API-05 | 05-02, 05-05 | wait_for_replication: ожидание Ack, отчёт, исход не меняется | ✓ SATISFIED | Истины 15-19, 27; TestWait_* + TestWaitIntegration_* (свежие PASS) |
| API-06 | 05-04 | Стандартный health-endpoint отражает gate | ✓ SATISFIED | Истина 13; TestHealth_reflectsGate (свежий PASS) |
| API-07 | 05-04 | Snapshot create/load через API | ✓ SATISFIED | Истина 14; TestSnapshot_* (свежие PASS) |
| TOOL-01 | 05-06 | Один статический бинарй, flags/env, --version, systemd-friendly | ✓ SATISFIED | Истины 20-24; TestMain_* + TestShutdown_secondSignal (свежие PASS) + исполненные пробы бинарня (statically linked, exit 0/1/2) |

### Test Quality Audit

Активные тесты с поведенческими assertion-level по всем 8 требованиям; skipped — только громкие capability-probe (перечитаны на HEAD: main_test.go:116 go-tool build, main_test.go:663 и udprepl/repl_test.go:699 loopback-broadcast — каждый называет пробуемое и что остаётся доказанным package-сьютами); disabled-on-requirement: 0; circular: 0; insufficient assertions: 0.

### Decision Coverage

Гейт `check.decision-coverage-verify` исполнен в этой верификации: 19/19 honored, 0 not_honored — D-01..D-07 (watch-контракт под wire-именами), D-08..D-11 (wait-отчёт), D-12..D-15 (list), D-16..D-19 (бинарй/конфигурация/shutdown).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | — | TBD/FIXME/XXX маркеры долга во всех covered files (вкл. дрейфнувшие Makefile/main.go/go.mod) — ноль (grep исполнен) | — | — |
| (none) | — | Заглушек нет: единственные Unimplemented-return — обязательный embed protoc-gen-go-grpc v1.6.2 и nil-hub защитный путь watch.go (продакшн always wires хаб) | — | — |
| internal/grpcapi/server.go | 306-308 | IN-01: недостижимая mapError status-пасsthru ветка — info-уровень, disposition open в 05-REVIEW-DISPOSITION.md (перенос из прошлого прохода; файл байт-идентичен) | ℹ️ Info | Не блокирует |

### Review Findings Disposition (контекст)

CR-01 / WR-02 исправлены и перепокрыты свежими прогонами этой верификации (TestWaitIntegration_ackedThroughRealAdapterMirror, TestWait_overflowingTimeoutClamped PASS). Перегенерированный 2026-11-18 review записал WR-01 и IN-02 как open — доложены в Advisory (#3304: без детерминистичного evidence не блокируют, triage в review-цикле). IN-01 — info, открыт, внеблокирующая. 05-SECURITY.md: 12/12 угроз закрыты, threats_open: 0.

### Human Verification Required

N/A — инфраструктурная фаза (внутренний gRPC API + бинарй; нет UI). Все критерии проверены программно в этой верификации; каждая поведенческая истина покрыта именованным тестом со свежим прогоном. Единственный deployment-environment item (systemd unit e2e) отложен в Phase 7 (Deferred Items); фаза 7 исполнена, её остаточные human-items управляются её собственным VERIFICATION/UAT.

### Gaps Summary

Отсутствуют. Все 28 must-have истин проверены на HEAD e9eeecc; внутренний слой фазы байт-идентичен дважды верифицированному состоянию, весь фаза-7 дрейф (reflection-регистрация, релизные targets, tool-директивы grpcurl/goreleaser, go 1.27.4) проверен построчным diff + свежими прогонами без единого отказа; все 8 ID требований удовлетворены без сирот; гейты (race-срез 5 пакетов + бинарные smoke 4/4, статический артефакт с версией, exit-коды 0/1/2, tidy, buf zero-waiver, decision coverage 19/19) зелёные в собственных прогонах этой верификации; цель фазы наблюдаемо достигнута. Fingerprint пересчитан: `v4:sha256:9e1cb10195a82727dfa38b4ec54c1eafa3a44dd9cc66505600caef45574c35dd` (42 covered files).

---

_Verified: 2026-11-18T11:03:39Z_
_Verifier: Claude (gsd-verifier)_
