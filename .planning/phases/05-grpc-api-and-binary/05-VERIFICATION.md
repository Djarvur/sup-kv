---
phase: 05-grpc-api-and-binary
verified: 2026-11-16T10:56:44Z
status: passed
score: 28/28 must-haves verified
covered_files: [".planning/phases/05-grpc-api-and-binary/05-01-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-01-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-02-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-02-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-03-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-03-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-04-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-04-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-05-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-05-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-06-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-06-SUMMARY.md", "Makefile", "buf.gen.api.yaml", "buf.gen.repl.yaml", "buf.yaml", "cmd/sup-kv/main.go", "cmd/sup-kv/main_test.go", "go.mod", "go.sum", "internal/app/persist/engine.go", "internal/app/persist/engine_test.go", "internal/app/repl/core.go", "internal/app/repl/retransmit_test.go", "internal/app/repl/sync_test.go", "internal/app/storage/store.go", "internal/app/storage/store_test.go", "internal/grpcapi/apiproto/supkvapi/v1/supkvapi.pb.go", "internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto", "internal/grpcapi/apiproto/supkvapi/v1/supkvapi_grpc.pb.go", "internal/grpcapi/bench_test.go", "internal/grpcapi/gate.go", "internal/grpcapi/gate_test.go", "internal/grpcapi/server.go", "internal/grpcapi/server_test.go", "internal/grpcapi/wait.go", "internal/grpcapi/wait_integration_test.go", "internal/grpcapi/wait_test.go", "internal/grpcapi/watch.go", "internal/grpcapi/watch_test.go", "internal/udprepl/repl.go", "internal/udprepl/repl_test.go"]
covered_digest: "v4:sha256:1c1c587f96f0a8ec7d8935cc744d9fcdcfe0e4b410fc02c2f12b4a911e66e32c"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 28/28
  previous_verified: 2026-11-15T18:27:10Z
  reason: "stale fingerprint — phase-6 commits since a631fa5 touched covered inputs after the 2026-11-15 pass: signal ownership fix 69d1e47 and observability setup d9a0a43 in cmd/sup-kv/main.go (+348 lines: slog pipeline, opt-in metrics/debug listener), seams 34b089f/3ff81db (repl.Config.OnDiscard, MergeStats, engine DumpStats, adapter mirrors in core.go/engine.go/repl.go/store.go), scenario suites in main_test.go (+819), benchmarks 9aa7bc0/9997abc/8aea7fc (new internal/grpcapi/bench_test.go, server_test.go testing.TB generalization), darwin flush 4f75a3b"
  gaps_closed: []
  gaps_remaining: []
  regressions: []
deferred:
  - truth: "systemd unit behavior end-to-end (TOOL-01: Type=simple unit + SIGTERM lifecycle observed on a real host)"
    addressed_in: "Phase 7"
    evidence: "Phase 7 goal: 'A stranger can deploy and operate a two-node sup-kv cluster from the documentation alone'; 05-06-PLAN defers systemd documentation to Phase 7 ('Type=simple with the health endpoint as readiness — documentation lands in Phase 7'); 05-VALIDATION.md records the unit e2e as the single manual-only item, deployment-environment state. The binary-side observable contract it depends on (SIGTERM -> exit 0 after final dump, health readiness, exit codes 0/1/2) is behaviorally proven by fresh TestMain_lifecycle and TestShutdown_secondSignal runs in this verification on HEAD e47733c."
decision_coverage:
  honored: 19
  total: 19
  not_honored: []
---

# Phase 5: gRPC API and Binary — Verification Report

**Phase Goal:** sup-kv becomes a usable product — a single static binary exposing the typed gRPC surface with Watch, startup gate, `wait_for_replication` and standard health checking
**Verified:** 2026-11-16T10:56:44Z
**Status:** passed
**Re-verification:** Yes — stale-fingerprint regeneration (no gaps in the prior 2026-11-15 pass; phase-6 hardening commits touched covered inputs and invalidated the covered-input digest). The contract is the renamed baseline (SupKVService / WatchResponse / apiproto/supkvapi/v1 — maintainer decision 2026-11-14, 7b55ad1) plus the current HEAD e47733c.

## Drift Contract (what changed since the prior pass)

Контракт этой проверки — текущее дерево на HEAD e47733c. Отдиффовано против a631fa5 (HEAD прошлого прохода) по всем covered inputs:

- **Фаза-5 API-слой байт-идентичен прошлому проходу**: `git diff a631fa5..HEAD -- internal/grpcapi/server.go gate.go watch.go wait.go apiproto/` — пусто. Вся поверхность хендлеров, gate-семантика, watch-хаб, wait-алгоритм и wire-контракт не менялись; истины 1–19 опираются на неизменённый код, перепроверенный свежими прогонами поверх дрейфнувших нижних слоёв.
- **cmd/sup-kv/main.go (+348 строк, фаза 6)**: прочитан полностью в этой верификации. Контракт фазы 5 цел: все 19 ручек с SUPKV_-двойниками (main.go:93-126, 274-348), precedence flag > env > default через envOrDefault-в-качестве-flag-default, --version (211-214, 705-707), полная проводка NewServer со всеми пятью швами ServerOptions (452-458), RegisterSupKVServiceServer + RegisterHealthServer на одном grpc.Server (584-585), `go srv.RunGate(apiCtx)` + `go hub.Run(apiCtx)` (461-462), шесть шагов D-19 в shutdown (639-679): health NOT_SERVING → [новое: shutdown metrics-listener, фаза-6, позиционирован после health-flip] → apiCancel → bounded GracefulStop/Stop → replCancel+join → engine.Flush() → engineCancel → exit 0/1. Fix 69d1e47 (signal ownership: armSignals/serve/second-signal force path) усиливает D-19 — второй SIGTERM во время drain берёт force-path вместо дефолтной disposition, так что финальный dump выживает; заперт новым TestShutdown_secondSignal (PASS в этой верификации).
- **Нижние слои (фаза 6, аддитивные seams)**: repl.Config.OnDiscard callback (34b089f, 06-03 D-08) и MergeStats-счётчики (3ff81db), engine.DumpStats, adapter-зеркала UnackedStats/SentByType/Retransmissions — всё observability-поверхность фазы 6 со своими decision-ID; семантика install/tap/Acked/syncArrived/Flush не тронута (store.go-дрейф — только уточнение doc-комментария OnChange; Flush() на месте, engine.go:643).
- **Тесты**: main_test.go +819 строк новых фаз-6 семейств (TestShutdown_secondSignal, TestLog_*, TestMetrics_*) поверх нетронутых TestMain_version/precedence/lifecycle; server_test.go — механическая генерализация waitFor на testing.TB для переиспользования бенчмарком; новый bench_test.go (фаза-6 D-13).
- **Makefile**: добавлен только ручной `bench` target; build/proto targets неизменны.
- **Rename-контракт перепроверен**: ноль символов WatchEvent/RegisterSupKVServer/SupKVServer/NewSupKVClient в не-тестовых Go/proto файлах (grep этой верификации); SupKVService × 80 в supkvapi_grpc.pb.go, WatchResponse × 15 в supkvapi.pb.go; proto объявляет все 7 RPC (Get/Set/Delete/List/Watch-streaming/SnapshotCreate/SnapshotLoad) с живыми wait-полями (wait_for_replication/wait_timeout_ms, D-09); buf.yaml — zero waivers, STANDARD, с записанным решением мейнтейнера от 2026-11-14.

## Goal Achievement

### Observable Truths

Те же 28 истин, что в прошлом проходе (SC1–SC5 ROADMAP + truths планов 05-01..05-06), перепроверенные на HEAD e47733c. Каждая поведенческая истина имеет свежий прогон (`-count=1 -race`) или исполненную команду из ЭТОЙ верификации.

| # | Truth | Source | Status | Evidence |
|---|-------|--------|--------|----------|
| 1 | gRPC Set затем Get возвращает идентичное значение и 16-байтовую версию (D-01) | SC1, 05-01 | ✓ VERIFIED | Пакет grpcapi свежий -race прогон: ok 2.313s, вкл. TestServer_setGetRoundTrip; server.go байт-идентичен прошлому верифицированному проходу; proto Record{key,version,value,deleted} на новом пути apiproto/supkvapi/v1 |
| 2 | Каждый сентинел маппится на закреплённый RPC-код через errors.Is (NotFound/AlreadyExists/InvalidArgument/Unavailable×2/InvalidArgument для имён снапшотов) | SC1, 05-01 | ✓ VERIFIED | Свежий прогон вкл. TestServer_errorMapping; единственный mapError в неизменённом server.go |
| 3 | create_if_absent=true возвращает AlreadyExists для любой существующей записи, включая tombstoned | SC1, 05-01 | ✓ VERIFIED | TestServer_errorMapping (live + tombstoned строки) в свежем прогоне; маршрутизация SetIfAbsent в server.go:142-146 |
| 4 | List возвращает сортированные live-ключи под префиксом; пустой префикс = все ключи; без пагинации (D-12..D-15) | SC1, 05-01 | ✓ VERIFIED | TestServer_list в свежем прогоне; фильтр Deleted + HasPrefix + slices.Sort в неизменённом server.go |
| 5 | go.mod/go.sum tidy-exact; grpc v1.85.0 direct require; protoc-gen-go-grpc v1.6.2 tool-директива | 05-01 | ✓ VERIFIED | `go mod tidy && git diff --exit-code go.mod go.sum` exit 0 (исполнено в этой верификации); toolchain go1.27.4 |
| 6 | Watch-подписчик сначала получает текущее состояние как обычные события (tombstones как deleted=true), затем живые изменения — без дублей и пропусков (D-02/D-03) | SC2, 05-03 | ✓ VERIFIED | TestWatch_snapshotThenLive + TestWatch_exactCut в свежем -race прогоне пакета grpcapi (26 тестов, вкл. всё семейство TestWatch_*) |
| 7 | События только при реальных install; проигравшая/равная версия не даёт события (D-06) | SC2, 05-03 | ✓ VERIFIED | TestStore_onChange (8 субтестов, вкл. loser/equal silence) — свежий прогон storage: ok 1.676s; TestWatch_replicatedChange в свежем прогоне grpcapi |
| 8 | Локальные и реплицированные изменения в одном потоке в порядке прибытия, неразличимы по содержимому (D-05/D-07) | SC2, 05-03 | ✓ VERIFIED | TestWatch_replicatedChange в свежем прогоне; toProto без origin-поля в неизменённом watch.go |
| 9 | Медленный потребитель отключается (ResourceExhausted), store продолжает принимать записи — tap никогда не блокирует write lock | SC2, 05-03 | ✓ VERIFIED | TestWatch_slowConsumer в свежем -race прогоне; select-with-default bounded 4096 ingress в watch.go |
| 10 | Prefix-фильтр применяется к snapshot и live-событиям; пустой префикс = все ключи (D-04) | SC2, 05-03 | ✓ VERIFIED | TestWatch_prefix в свежем прогоне; wants() в watch.go |
| 11 | До открытия gate Set/Delete возвращают Unavailable, Get/List/Watch работают (API-04) | SC3, 05-04 | ✓ VERIFIED | TestGate_writesUnavailableReadsOK в свежем прогоне; gate-проверка — первый statement Set/Delete (server.go:132-134, 172-174), чтения gate не консультируют |
| 12 | Gate открывается по более раннему из sync-flip или startup_gate (A8) | SC3, 05-04 | ✓ VERIFIED | TestGate_opensOnSync + TestGate_opensOnTimeout в свежем прогоне; RunGate 20ms poll + timer-backstop в gate.go |
| 13 | Стандартный health-endpoint: NOT_SERVING для пустого имени сервиса до gate, SERVING после — связь 1:1 (API-06) | SC3, 05-04 | ✓ VERIFIED | TestHealth_reflectsGate в свежем прогоне; NOT_SERVING в конструкторе, одиночный CAS в openGate; регистрация рядом с SupKVService в main.go:584-585 (перечитано в этой верификации) |
| 14 | SnapshotCreate/SnapshotLoad гейтятся как записи; ErrInvalidSnapshotName -> InvalidArgument (A5, API-07) | SC3/SC4, 05-04 | ✓ VERIFIED | TestSnapshot_gated/_invalidName/_createLoadRoundTrip в свежем прогоне; адаптер только маппит |
| 15 | wait_for_replication отчитывает D-08 агрегат и никогда не меняет исход записи (API-05) | SC4, 05-05 | ✓ VERIFIED | TestWait_timeout (Set RPC OK, acked=false 0/1) + TestWait_acksArrive (1/1) в свежем прогоне; wait строго после подтверждения движком |
| 16 | Ноль известных пиров — acked=true немедленно, total_peers 0 (D-11), включая nil-швы | SC4, 05-05 | ✓ VERIFIED | TestWait_zeroPeers в свежем прогоне; nil-шов и нулевой count короткое замыкание в wait.go |
| 17 | Эвиктированный пир никогда не считается Ack; superseded-запись — acked=false с замороженными счётчиками (Q2) | SC4, 05-05 | ✓ VERIFIED | TestWait_evictionNotAck + TestWait_superseded в свежем прогоне |
| 18 | Wait ждёт появления (key, version) перед трактовкой отсутствия как drained — без ложного acked из notify-queue race (Pitfall 4) | SC4, 05-05 | ✓ VERIFIED | TestWait_appearanceRace + TestWaitIntegration_ackedThroughRealAdapterMirror (реальные адаптеры через реальный mirror) в свежем -race прогоне |
| 19 | Timeout — per-request wait_timeout_ms если положителен, иначе операторский дефолт (D-09), overflow-safe (WR-02) | SC4, 05-05 | ✓ VERIFIED | TestWait_perRequestTimeout + TestWait_overflowingTimeoutClamped в свежем прогоне |
| 20 | SIGTERM даёт чистый exit 0 после финального write-confirmed dump; подтверждённые данные никогда не остаются только в памяти (D-19) | SC5, 05-06 | ✓ VERIFIED | Свежий прогон `go test -race -count=1 -run 'TestMain_\|TestShutdown_' ./cmd/sup-kv/ -v`: TestMain_lifecycle PASS (реальный процесс/TCP: SERVING → Set/Get → SIGTERM → exit 0 → dump содержит ключ); shutdown() перечитан в этой верификации — шесть шагов D-19 целы под fix 69d1e47, второй SIGTERM берёт force-path (TestShutdown_secondSignal PASS); провал финального dump — exit 1 |
| 21 | Каждая ручка — flag и SUPKV_ env с приоритетом flag > env > default (D-16/D-17) | SC5, 05-06 | ✓ VERIFIED | TestMain_precedence PASS (свежий); 19 ручек в registerConfig/main.go:274-348 с env-двойниками (93-126) — перечитано полностью; `./bin/sup-kv` без ручек — exit 2 с usage (исполнено) |
| 22 | --version печатает version + commit + Go runtime, ldflags-инъекция, dev-фолбэк (D-18) | SC5, 05-06 | ✓ VERIFIED | `./bin/sup-kv --version` → `sup-kv e47733c-dirty e47733c go1.27.4` (исполнено в этой верификации); TestMain_version PASS (свежий) |
| 23 | Engine.Flush() превращает остаточную dirty (remote merges без waiter) в подтверждённый dump до отмены ctx движка | SC5, 05-06 | ✓ VERIFIED | Свежий прогон persist: ok 2.362s вкл. TestFlush_remoteDirtiness/_closed/_clean; Flush() экспортирован (engine.go:643, перечитано — фаза-6 DumpStats-дрейф его не тронул) |
| 24 | Поставляемый артефакт — один статический CGO_ENABLED=0 бинарй | SC5, 05-06 | ✓ VERIFIED | `make build` + `file bin/sup-kv` → "statically linked" (исполнено); Makefile build target CGO_ENABLED=0 + ldflags (bench-дрейф его не тронул) |
| 25 | PendingSnapshot различает пира, который Acked (в Acked), от эвиктированного по peer_timeout (вне обоих множеств) | 05-02 | ✓ VERIFIED | Свежий прогон repl: ok 3.080s вкл. TestRetransmit_ackDrainSemantics; evictPeers никогда не трогает acked — переконфирмано чтением текущего core.go (фаза-6 OnDiscard-дрейф семантику не менял) |
| 26 | Core.SyncArrived() false до первого DataSync, true навсегда после (A1) | 05-02 | ✓ VERIFIED | TestSync_syncArrived в свежем прогоне repl; флаг в handleDataSync, акцессор на месте |
| 27 | gRPC-читатели получают оба состояния репликации из любой горутины через атомики адаптера, не через single-driver core | 05-02 | ✓ VERIFIED | Свежий прогон udprepl: ok 1.361s вкл. TestMirror_pendingAckVisible + TestMirror_syncArrivedFlip; copy-on-write pendingCopy atomic + mirrorState — переконфирмано (фаза-6 добавила sibling-зеркала статистики в том же стиле, семантика не тронута) |
| 28 | В core не добавлено коллбеков и каналов — результаты, не коллбеки (дисциплина port.Outbox) | 05-02 | ✓ VERIFIED | Дифф фаз-5 коммитов (b06b54d, d1026dc) по core.go: добавлены только bool-флаг, acked-срез, accessors — ноль func-полей Config, ноль каналов. Примечание текущего дерева: repl.Config.OnDiscard func(class, detail string) добавлен ФАЗОЙ 6 (34b089f, решение 06-03 D-08, собственный план/decision-ID) — вне скоупа истины фазы 5, санкционированный drift с документацией; Log func существовал до фазы 5 |

**Score:** 28/28 truths verified (0 present, behavior-unverified)

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | systemd unit end-to-end (реальный хост, Type=simple unit) | Phase 7 | Цель Phase 7 покрывает deploy/operate по документации; 05-06-PLAN прямо относит systemd-документацию к Phase 7; 05-VALIDATION.md — единственный manual-only item. Контракт бинарня, который юнит супервизирует (SIGTERM → exit 0 + финальный dump, health readiness, коды 0/1/2, второй SIGTERM → force-stop с сохранением dump), поведенчески доказан свежими прогонами TestMain_lifecycle и TestShutdown_secondSignal этой верификации. |

### Advisory (New Scope, Unevidenced)

None — новые находки вне контракта отсутствуют. Весь дрейф с прошлого прохода (фаза-6 seams, slog, metrics listener, signal ownership, бенчмарки, darwin flush) проверен прямыми прогонами (тесты -race по всем затронутым пакетам, lint-tidy-buf-гейты, бинарные пробы) без единого отказа; все additions аддитивны и несут собственные фаз-6 decision-ID.

### Required Artifacts

gsd-tools `verify artifacts` не применим к строковой форме must_haves.artifacts планов (формат-варианс, как в прошлых проходах); все артефакты проверены вручную. Прото-артефакты байт-идентичны прошлому верифицированному проходу (`git diff a631fa5..HEAD` пуст):

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto` | wire-контракт фазы, 7 RPC, service SupKVService, WatchResponse, живые wait-поля | ✓ VERIFIED | Перечитан: все 7 RPC, create_if_absent/wait_for_replication/wait_timeout_ms/D-08 поля/prefix; buf lint + format exit 0 (исполнено) |
| `internal/grpcapi/apiproto/supkvapi/v1/supkvapi.pb.go`, `supkvapi_grpc.pb.go` | закоммиченный сгенерированный код | ✓ VERIFIED | Байт-идентичны прошлому проходу (diff пуст); RegisterSupKVServiceServer/SupKVServiceServer interface на месте |
| `internal/grpcapi/server.go`, `watch.go`, `gate.go`, `wait.go` | хендлеры + хаб + gate + wait | ✓ VERIFIED | Байт-идентичны прошлому верифицированному проходу (git diff a631fa5..HEAD пуст); вся поверхность перепокрыта свежим -race прогоном пакета |
| `internal/app/storage/store.go` | Config.OnChange tap из install | ✓ VERIFIED | Дрейф — только doc-комментарий OnChange (уточнение о tombstone/реентрабельности); семантика tap неизменна; TestStore_onChange в свежем прогоне |
| `internal/app/repl/core.go` | SyncArrived + PendingInfo.Acked | ✓ VERIFIED | Оба шва на месте под фаза-6 OnDiscard/MergeStats-дрейфом; TestRetransmit_ackDrainSemantics + TestSync_syncArrived в свежем прогоне |
| `internal/udprepl/repl.go` | зеркала адаптера | ✓ VERIFIED | syncArrived/pendingCopy целы; фаза-6 добавила observability-зеркала в том же post-iteration стиле; TestMirror_* в свежем прогоне |
| `internal/app/persist/engine.go` | экспортированный Flush | ✓ VERIFIED | Flush() → flush() (engine.go:643), док D-19; TestFlush_* в свежем прогоне; фаза-6 DumpStats-дрейф аддитивен |
| `cmd/sup-kv/main.go` | только flags/env/signals/wiring | ✓ VERIFIED | Прочитан полностью в этой верификации: 19 ручек + двойники, D-19 shutdown (шесть шагов, сохранённых под signal-ownership fix), полная проводка ServerOptions, регистрация SupKVService + health, ldflags-переменные, slog/metrics — фаза-6 аддитивы с собственными decision-ID |
| `cmd/sup-kv/main_test.go` | семейство smoke бинарня | ✓ VERIFIED | TestMain_version/precedence/lifecycle на месте (+819 строк фаза-6 семейств поверх); свежий прогон 4/4 PASS (вкл. TestShutdown_secondSignal) |
| `internal/grpcapi/bench_test.go` (новый covered input) | фаза-6 D-13 бенчмарки | ✓ VERIFIED | Существует, компилируется, участвует в свежем прогоне пакета; не изменяет продакшн-поверхность (тест-файл); серверная waitFor-генерализация — единственное касание server_test.go |
| `Makefile`, `go.mod`, `go.sum`, `buf.gen.api.yaml`, `buf.gen.repl.yaml`, `buf.yaml` | build/codegen/deps | ✓ VERIFIED | bench target добавлен (ручной, без CI-порогов); build/proto targets неизменны; grpc v1.85.0 direct; tidy-exact (исполнено); buf.yaml zero waivers STANDARD |

Проводка (import + использование) подтверждена для каждого артефакта: все задействованы main.go, харнессом или собственными потребителями пакетов; orphan-файлов нет.

### Key Link Verification

Все key_links шести планов прослежены; ссылки на неизменённых файлах унаследованы от байт-идентичного состояния, ссылки на дрейфнувших файлах перечитаны на текущем HEAD:

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| Хендлеры (Get/Set/Delete/List/Snapshot) | persist.Engine | прямые вызовы движка | ✓ WIRED | server.go (байт-идентичен); движок перечитан — интерфейсные вызовы целы под DumpStats-дрейфом |
| Server notify-шов | udprepl.Adapter.NotifyLocalWrite | LocalNotify, nil-safe | ✓ WIRED | main.go:452 инъектирует адаптер (перечитано) |
| storage.Store.install | Config.OnChange → WatchHub.Ingest | tap → hub ingress | ✓ WIRED | store.go tap цел (дрейф — doc только); main.go:417 `OnChange: hub.Ingest` через startEngine |
| Exact-cut подавление | RecordVersion.Compare | строгое доминирование | ✓ WIRED | watch.go (байт-идентичен) |
| ServerOptions.SyncArrived → gate resolver → gated() | зеркало адаптера | poll + флаг | ✓ WIRED | main.go:453 `SyncArrived: adapter.SyncArrived` (перечитано); RunGate поллит; Set/Delete читают gated() |
| Разрешение gate → health.SetServingStatus | связь 1:1 | одиночный CAS | ✓ WIRED | gate.go; health зарегистрирован main.go:585 рядом с SupKVService (584) |
| waitForReplication | KnownPeers/PendingCopy швы | только инъектированные замыкания | ✓ WIRED | wait.go; main.go:454-455 инъектирует зеркала адаптера + WaitTimeout (457) |
| Snapshot-хендлеры | валидация имён persistfile | единственный закрытый путь | ✓ WIRED | server.go → engine.Snapshot/Restore; адаптер маппит ErrInvalidSnapshotName |
| handleAck/handleDataSync → core → зеркала адаптера | post-iteration mirrorState | refresh после каждого события | ✓ WIRED | repl.go (перечитан — фаза-6 sibling-зеркала в том же сайте); core.go handleAck/handleDataSync целы |
| Граф проводки main | hub → engine → adapter → server → gate/health → serve | полный цикл | ✓ WIRED | main.go:435-489, вкл. `go srv.RunGate(apiCtx)` (461), `go hub.Run(apiCtx)` (462); abortWire компенсирует провал поздних шагов |
| SIGTERM → порядок shutdown | D-19 шесть шагов | последовательность | ✓ WIRED | main.go:638-681: NOT_SERVING → [metrics listener down] → apiCancel → bounded GracefulStop/Stop (второй сигнал — force) → replCancel+join → Flush → engineCancel + join engineDone |
| Makefile build | статический штампованный bin/sup-kv | CGO_ENABLED=0 + ldflags | ✓ WIRED | Исполнено: statically linked, версия `sup-kv e47733c-dirty e47733c go1.27.4` |
| tool-блок go.mod | protoc-gen-go-grpc через go tool | buf-плагины | ✓ WIRED | go.mod tool-блок + buf.gen.api.yaml local go-tool плагины; make proto per-tree |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| Get-хендлер | record | engine.Get (persist поверх реального store) | Yes | ✓ FLOWING |
| List-хендлер | keys | engine.Records() snapshot, фильтр/сортировка | Yes | ✓ FLOWING |
| Watch-поток | events | storage install tap → hub ingress → подписчики | Yes | ✓ FLOWING |
| Wait-отчёт | acked/counts | инъектированные зеркала адаптера (core-acknowledged состояние) | Yes | ✓ FLOWING |
| Health-статус | gate atomic | resolver openGate / shutdown NOT_SERVING | Yes | ✓ FLOWING |
| Set-ответ | version | engine-штампованный RecordVersion | Yes | ✓ FLOWING |
| --version вывод | version/commit | ldflags -X (git describe) | Yes | ✓ FLOWING (исполнено: e47733c-dirty / e47733c / go1.27.4) |

Статических return, hardcoded-литералов и моков в продакшн-путях ответов нет. (Тестовые fakes — только в _test.go.)

### Behavioral Spot-Checks

Все команды исполнены в ЭТОЙ верификации на HEAD e47733c (рабочее дерево dirty только по .planning/):

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Полный фаза-5 срез пакетов (26 grpcapi-тестов вкл. TestServer_/TestWatch_/TestGate_/TestHealth_/TestSnapshot_/TestWait_/TestWaitIntegration_; storage вкл. TestStore_onChange; persist вкл. TestFlush_; repl вкл. TestSync_syncArrived/TestRetransmit_ackDrainSemantics; udprepl вкл. TestMirror_*) | `go test -race -count=1 ./internal/grpcapi/... ./internal/app/storage/ ./internal/app/persist/ ./internal/app/repl/ ./internal/udprepl/` | все ok (grpcapi 2.313s, storage 1.676s, persist 2.362s, repl 3.080s, udprepl 1.361s) | ✓ PASS |
| Бинарй: lifecycle/precedence/version + shutdown force-path | `go test -race -count=1 -run 'TestMain_\|TestShutdown_' ./cmd/sup-kv/ -v` | 4/4 PASS (TestMain_version, TestMain_precedence, TestMain_lifecycle, TestShutdown_secondSignal; 3.707s, реальные процессы/TCP) | ✓ PASS |
| Статический артефакт + версия + коды выхода | `make build && file bin/sup-kv && ./bin/sup-kv --version && ./bin/sup-kv` | statically linked; `sup-kv e47733c-dirty e47733c go1.27.4`; exit 2 с usage | ✓ PASS |
| Tidy-гейт | `go mod tidy && git diff --exit-code go.mod go.sum` | exit 0 | ✓ PASS |
| buf-гейты (zero-waiver) | `go tool buf lint && go tool buf format --diff --exit-code` | оба exit 0 | ✓ PASS |
| Rename-контракт | grep старых символов (WatchEvent/RegisterSupKVServer/SupKVServer/NewSupKVClient) в не-тестовых Go/proto | ноль совпадений; SupKVService/WatchResponse на месте | ✓ PASS |
| Идентичность API-слоя прошлому проходу | `git diff a631fa5..HEAD -- internal/grpcapi/server.go gate.go watch.go wait.go apiproto/` | пусто (байт-идентично) | ✓ PASS |

### Probe Execution

Фаза не объявляет probe-скриптов (`scripts/*/tests/probe-*` не существуют); runnable-check контракт фазы — сам Go-набор, покрытый выше свежими прогонами.

### Requirements Coverage

ID из планов: 05-01 [API-01, API-02], 05-02 [API-04, API-05], 05-03 [API-03], 05-04 [API-04, API-06, API-07], 05-05 [API-05], 05-06 [TOOL-01]. Объединение = ровно 8 ID фазы. REQUIREMENTS.md (Traceability) маппит ровно API-01..API-07 + TOOL-01 на Phase 5, статус Complete — сиротских требований нет, незаявленных ID нет.

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| API-01 | 05-01 | gRPC-сервис с protobuf; Get/Set (incl. create-if-absent)/Delete с полным набором ошибок | ✓ SATISFIED | Истины 1-3; свежий прогон grpcapi вкл. TestServer_errorMapping |
| API-02 | 05-01 | List ключей под префиксом в сортированном порядке | ✓ SATISFIED | Истина 4; TestServer_list (свежий PASS) |
| API-03 | 05-03 | Watch-поток, покрывающий локальные и реплицированные изменения | ✓ SATISFIED | Истины 6-10; семейство TestWatch_* + TestStore_onChange (свежие PASS) |
| API-04 | 05-02, 05-04 | Записи Unavailable во время startup gate; чтения немедленно | ✓ SATISFIED | Истины 11-12, 25-26; TestGate_* + TestSync_syncArrived (свежие PASS) |
| API-05 | 05-02, 05-05 | wait_for_replication: ожидание Ack, отчёт, исход не меняется | ✓ SATISFIED | Истины 15-19, 27; TestWait_* + TestWaitIntegration_* (свежие PASS) |
| API-06 | 05-04 | Стандартный health-endpoint отражает gate | ✓ SATISFIED | Истина 13; TestHealth_reflectsGate (свежий PASS) |
| API-07 | 05-04 | Snapshot create/load через API | ✓ SATISFIED | Истина 14; TestSnapshot_* (свежие PASS) |
| TOOL-01 | 05-06 | Один статический бинарй, flags/env, --version, systemd-friendly | ✓ SATISFIED | Истины 20-24; TestMain_* + TestShutdown_secondSignal (свежие PASS) + исполненные пробы бинарня |

### Test Quality Audit

Активные тесты с поведенческими assertion-level по всем 8 требованиям; skipped — только два документированных громких capability-probe (loopback broadcast в wait_integration_test.go, go-tool в main_test.go:104, именуют пробуемое); disabled-on-requirement: 0; circular: 0; insufficient assertions: 0. Фаза-6 добавила семейства (TestShutdown_secondSignal, TestLog_*, TestMetrics_*, bench), только усиливающие покрытие фаза-5 контрактов (shutdown force-path, env validation, opt-in listener).

### Decision Coverage

Гейт `check.decision-coverage-verify` исполнен в этой верификации: 19/19 honored, 0 not_honored — D-01..D-07 (watch-контракт под новыми wire-именами), D-08..D-11 (wait-отчёт), D-12..D-15 (list), D-16..D-19 (бинарй/конфигурация/shutdown).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | — | TBD/FIXME/XXX маркеры долга во всех covered files (вкл. дрейфнувшие) — ноль (grep исполнен) | — | — |
| (none) | — | Заглушек нет: единственные Unimplemented-return — обязательный embed protoc-gen-go-grpc v1.6.2 и nil-hub защитный путь watch.go (продакшн always wires хаб) | — | — |
| internal/grpcapi/server.go | 306-308 | IN-01: недостижимая mapError status-пасsthru ветка — info-уровень, disposition open в 05-REVIEW-DISPOSITION.md (перенесено из прошлого прохода без изменений; файл байт-идентичен) | ℹ️ Info | Не блокирует |

### Review Findings Disposition (контекст)

CR-01 / WR-01 / WR-02 исправлены и перепокрыты свежими прогонами этой верификации (TestWaitIntegration_ackedThroughRealAdapterMirror, TestWait_overflowingTimeoutClamped PASS; WR-01 кламп в wait.go). IN-01 — info, открыта, внеблокирующая. 05-SECURITY.md: 12/12 угроз закрыты, threats_open: 0; T-05-01 (no-auth LAN) и T-05-06 (нет капа подписчиков) — принятые риски с записью.

### Human Verification Required

N/A — инфраструктурная фаза (внутренний gRPC API + бинарй; нет UI). Все критерии проверены программно в этой верификации; каждая поведенческая истина покрыта именованным тестом со свежим прогоном. Единственный deployment-environment item (systemd unit e2e) отложен в Phase 7 (Deferred Items); контракт бинарня на его стороне поведенчески доказан, включая новый second-SIGTERM force-path.

### Gaps Summary

Отсутствуют. Все 28 must-have истин проверены на HEAD e47733c с контрактом под новыми именами; API-слой фазы байт-идентичен прошлому верифицированному проходу, весь фаза-6 дрейф (seams, slog, metrics listener, signal ownership, бенчмарки) проверен чтением и свежими прогонами без единого отказа; все 8 ID требований удовлетворены без сирот; гейты (race-срез, бинарные пробы, tidy, buf zero-waiver, decision coverage 19/19) зелёные в собственных прогонах этой верификации; цель фазы наблюдаемо достигнута. Fingerprint пересчитан: `v4:sha256:1c1c587f96f0a8ec7d8935cc744d9fcdcfe0e4b410fc02c2f12b4a911e66e32c` (41 covered file, вкл. новый bench_test.go).

---

_Verified: 2026-11-16T10:56:44Z_
_Verifier: Claude (gsd-verifier)_
