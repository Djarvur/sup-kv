---
phase: 05-grpc-api-and-binary
verified: 2026-11-15T18:27:10Z
status: passed
score: 28/28 must-haves verified
covered_files: [".planning/phases/05-grpc-api-and-binary/05-01-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-01-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-02-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-02-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-03-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-03-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-04-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-04-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-05-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-05-SUMMARY.md", ".planning/phases/05-grpc-api-and-binary/05-06-PLAN.md", ".planning/phases/05-grpc-api-and-binary/05-06-SUMMARY.md", "Makefile", "buf.gen.api.yaml", "buf.gen.repl.yaml", "buf.yaml", "cmd/sup-kv/main.go", "cmd/sup-kv/main_test.go", "go.mod", "go.sum", "internal/app/persist/engine.go", "internal/app/persist/engine_test.go", "internal/app/repl/core.go", "internal/app/repl/retransmit_test.go", "internal/app/repl/sync_test.go", "internal/app/storage/store.go", "internal/app/storage/store_test.go", "internal/grpcapi/apiproto/supkvapi/v1/supkvapi.pb.go", "internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto", "internal/grpcapi/apiproto/supkvapi/v1/supkvapi_grpc.pb.go", "internal/grpcapi/gate.go", "internal/grpcapi/gate_test.go", "internal/grpcapi/server.go", "internal/grpcapi/server_test.go", "internal/grpcapi/wait.go", "internal/grpcapi/wait_integration_test.go", "internal/grpcapi/wait_test.go", "internal/grpcapi/watch.go", "internal/grpcapi/watch_test.go", "internal/udprepl/repl.go", "internal/udprepl/repl_test.go"]
covered_digest: "v4:sha256:d96ec90ed45c79d76a9f9cdc982cb4f7c6e1ba486c008d6eaf1af1b09aa0d42f"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 28/28
  previous_verified: 2026-11-09T15:02:03Z
  reason: "stale fingerprint — 7b55ad1 (buf zero-waiver rename), 4f76e26 (doc fixes), c194c6f (toolchain go1.27.4, CVE-2026-6859 closure), a631fa5 (retroactive 05-SECURITY.md) touched covered inputs after the 2026-11-09 pass"
  gaps_closed: []
  gaps_remaining: []
  regressions: []
deferred:
  - truth: "systemd unit behavior end-to-end (TOOL-01: Type=simple unit + SIGTERM lifecycle observed on a real host)"
    addressed_in: "Phase 7"
    evidence: "Phase 7 goal: 'A stranger can deploy and operate a two-node sup-kv cluster from the documentation alone'; 05-06-PLAN defers systemd documentation to Phase 7 ('Type=simple with the health endpoint as readiness — documentation lands in Phase 7'); 05-VALIDATION.md records the unit e2e as the single manual-only item, deployment-environment state. The binary-side observable contract it depends on (SIGTERM -> exit 0 after final dump, health readiness, exit codes 0/1/2) is behaviorally proven by TestMain_lifecycle and the ./bin/sup-kv probes re-executed in this verification on HEAD a631fa5."
decision_coverage:
  honored: 19
  total: 19
  not_honored: []
---

# Phase 5: gRPC API and Binary — Verification Report

**Phase Goal:** sup-kv becomes a usable product — a single static binary exposing the typed gRPC surface with Watch, startup gate, `wait_for_replication` and standard health checking
**Verified:** 2026-11-15T18:27:10Z
**Status:** passed
**Re-verification:** Yes — stale-fingerprint regeneration (no gaps in the prior pass; the 2026-11-14 buf zero-waiver rename and 2026-11-15 toolchain/docs commits invalidated the 2026-11-09 covered-input digest)

## Rename Contract (what changed since the prior pass)

Контрактом этой проверки является текущее дерево с НОВЫМИ именами — решение мейнтейнера от 2026-11-14 (7b55ad1, pre-release окно, записано в `.planning/STATE.md:204`, признано в `06-CONTEXT.md` D-16): сервис `SupKV` → `SupKVService`, сообщение `WatchEvent` → `WatchResponse`, прото перенесены `internal/grpcapi/apiproto/` → `internal/grpcapi/apiproto/supkvapi/v1/`, сгенерированный код перегенерирован, все потребители обновлены. Пины фазы 5 в 05-CONTEXT.md на старые имена превзойдены этим решением; пути артефактов в `must_haves` планов (строковая форма, старые пути) читаются по новому расположению — роль артефакта неизменна, перемещение документировано. Проверено собственными grep-прогонами этой верификации: ноль символьных ссылок на `WatchEvent`, `RegisterSupKVServer`, `SupKVServer`, `NewSupKVClient` и прочие старые символы; остаточные вхождения «SupKV» — только проза в комментариях (gate.go:95, server.go:2/64, main.go:358, тесты), что подтверждает и инкрементальный 05-REVIEW.md от 2026-11-14. `make proto` воспроизводит оба .pb.go байт-в-байт (`git diff --exit-code` после регенерации — exit 0, выполнено в этой верификации). buf.yaml: zero waivers, rule set STANDARD (D-06 waivers удалены в направлении кода).

## Goal Achievement

### Observable Truths

Must-haves: объединены из success criteria 1–5 ROADMAP (контракт) и frontmatter truths планов 05-01..05-06 — те же 28 истин, что в предыдущем проходе, перепроверенные на текущем HEAD (a631fa5) с новыми именами. Каждая поведенческая истина имеет именованный тест, запущенный свежим прогоном (`-count=1`, `-race`) в ЭТОЙ верификации, либо непосредственно исполненную команду — не заявление SUMMARY.

| # | Truth | Source | Status | Evidence |
|---|-------|--------|--------|----------|
| 1 | gRPC Set затем Get возвращает идентичное значение и 16-байтовую версию (D-01) | SC1, 05-01 | ✓ VERIFIED | TestServer_setGetRoundTrip PASS (свежий -race прогон); wire-контракт supkvapi.proto (новый путь apiproto/supkvapi/v1) объявляет Record{key,version,value,deleted} |
| 2 | Каждый сентинел маппится на закреплённый RPC-код через errors.Is (NotFound/AlreadyExists/InvalidArgument/Unavailable×2/InvalidArgument для имён снапшотов) | SC1, 05-01 | ✓ VERIFIED | TestServer_errorMapping PASS (свежий); единственный mapError в server.go:291-310, таблица шести сентинелов |
| 3 | create_if_absent=true возвращает AlreadyExists для любой существующей записи, включая tombstoned | SC1, 05-01 | ✓ VERIFIED | строки errorMapping (live + tombstoned) PASS; Set-хендлер маршрутизирует engine.SetIfAbsent (server.go:142-146) |
| 4 | List возвращает сортированные live-ключи под префиксом; пустой префикс = все ключи; без пагинации (D-12..D-15) | SC1, 05-01 | ✓ VERIFIED | TestServer_list PASS (свежий); server.go:201-211 фильтрует Deleted, HasPrefix, slices.Sort, один ответ |
| 5 | go.mod/go.sum tidy-exact; grpc v1.85.0 direct require; protoc-gen-go-grpc v1.6.2 tool-директива | 05-01 | ✓ VERIFIED | `go mod tidy && git diff --exit-code go.mod go.sum` exit 0 (исполнено); go.mod:16 grpc v1.85.0, go.mod:297 protoc-gen-go-grpc v1.6.2; toolchain go1.27.4 (c194c6f, закрытие GO-2026-6859 — решение мейнтейнера в STATE.md:206) |
| 6 | Watch-подписчик сначала получает текущее состояние как обычные события (tombstones как deleted=true), затем живые изменения — без дублей и пропусков (D-02/D-03) | SC2, 05-03 | ✓ VERIFIED | TestWatch_snapshotThenLive PASS + TestWatch_exactCut PASS (свежие -race прогоны; konkurentные писатели, строго возрастающие версии) |
| 7 | События только при реальных install; проигравшая/равная версия не даёт события (D-06) | SC2, 05-03 | ✓ VERIFIED | TestStore_onChange PASS (свежий, 8 субтестов incl. loser/equal silence); TestWatch_replicatedChange PASS |
| 8 | Локальные и реплицированные изменения в одном потоке в порядке прибытия, неразличимы по содержимому (D-05/D-07) | SC2, 05-03 | ✓ VERIFIED | TestWatch_replicatedChange PASS (свежий); toProto возвращает apiproto.WatchResponse без origin-поля (watch.go:58-60) |
| 9 | Медленный потребитель отключается (ResourceExhausted), store продолжает принимать записи — tap никогда не блокирует write lock | SC2, 05-03 | ✓ VERIFIED | TestWatch_slowConsumer PASS (свежий, 0.71s: ResourceExhausted + store liveness); Ingest — select-with-default в bounded 4096 ingress (watch.go:213-219) |
| 10 | Prefix-фильтр применяется к snapshot и live-событиям; пустой префикс = все ключи (D-04) | SC2, 05-03 | ✓ VERIFIED | TestWatch_prefix PASS (свежий); wants() watch.go:309-315 |
| 11 | До открытия gate Set/Delete возвращают Unavailable, Get/List/Watch работают (API-04) | SC3, 05-04 | ✓ VERIFIED | TestGate_writesUnavailableReadsOK PASS (свежий); gate-проверка — первый statement Set/Delete (server.go:132-134, 172-174) до валидации; чтения gate не консультируют |
| 12 | Gate открывается по более раннему из sync-flip или startup_gate (A8) | SC3, 05-04 | ✓ VERIFIED | TestGate_opensOnSync + TestGate_opensOnTimeout PASS (свежие); RunGate 20ms poll + timer-backstop (gate.go:122-145) |
| 13 | Стандартный health-endpoint: NOT_SERVING для пустого имени сервиса до gate, SERVING после — связь 1:1 (API-06) | SC3, 05-04 | ✓ VERIFIED | TestHealth_reflectsGate PASS (свежий, обе стороны); NOT_SERVING выставлен в конструкторе (server.go:88), одиночный CAS в openGate хранит флаг и переключает health (gate.go:152-156); регистрация рядом с SupKVService в main.go:366-367 |
| 14 | SnapshotCreate/SnapshotLoad гейтятся как записи; ErrInvalidSnapshotName -> InvalidArgument (A5, API-07) | SC3/SC4, 05-04 | ✓ VERIFIED | TestSnapshot_gated + TestSnapshot_invalidName (пустое, "..", "a/b", 65 символов) + TestSnapshot_createLoadRoundTrip PASS (свежие); адаптер только маппит, валидация — в persistfile |
| 15 | wait_for_replication отчитывает D-08 агрегат и никогда не меняет исход записи (API-05) | SC4, 05-05 | ✓ VERIFIED | TestWait_timeout PASS (Set RPC при этом OK, acked=false 0/1); TestWait_acksArrive PASS (1/1); wait строго после подтверждения движком (server.go:150-156, 184-188) |
| 16 | Ноль известных пиров — acked=true немедленно, total_peers 0 (D-11), включая nil-швы | SC4, 05-05 | ✓ VERIFIED | TestWait_zeroPeers PASS (свежий); nil-шов и нулевой count короткое замыкание wait.go:95-105 |
| 17 | Эвиктированный пир никогда не считается Ack; superseded-запись — acked=false с замороженными счётчиками (Q2) | SC4, 05-05 | ✓ VERIFIED | TestWait_evictionNotAck PASS; TestWait_superseded PASS (свежие) |
| 18 | Wait ждёт появления (key, version) перед трактовкой отсутствия как drained — без ложного acked из notify-queue race (Pitfall 4) | SC4, 05-05 | ✓ VERIFIED | TestWait_appearanceRace PASS (свежий); CR-01 фикс закрыт интеграционным TestWaitIntegration_ackedThroughRealAdapterMirror PASS (свежий -race: два реальных адаптера, Set с wait_for_replication через реальный mirror, acked=true 1/1) |
| 19 | Timeout — per-request wait_timeout_ms если положителен, иначе операторский дефолт (D-09), overflow-safe (WR-02) | SC4, 05-05 | ✓ VERIFIED | TestWait_perRequestTimeout + TestWait_overflowingTimeoutClamped PASS (свежие); waitTimeout клампит MaxInt64/ms затем 10× операторский потолок (wait.go:210-225) |
| 20 | SIGTERM даёт чистый exit 0 после финального write-confirmed dump; подтверждённые данные никогда не остаются только в памяти (D-19) | SC5, 05-06 | ✓ VERIFIED | TestMain_lifecycle PASS (свежий -race: реальный процесс, реальный TCP — health SERVING → Set/Get → SIGTERM → exit 0 → dump содержит ключ → без паники); shutdown() main.go:397-428 — шесть закреплённых шагов; провал финального dump — exit 1 |
| 21 | Каждая ручка — flag и SUPKV_ env с приоритетом flag > env > default (D-16/D-17) | SC5, 05-06 | ✓ VERIFIED | TestMain_precedence PASS (свежий); 19 ручек в registerConfig (main.go:211-251) с env-двойниками (main.go:75-95); `./bin/sup-kv` без ручек — exit 2 с usage (исполнено в этой верификации) |
| 22 | --version печатает version + commit + Go runtime, ldflags-инъекция, dev-фолбэк (D-18) | SC5, 05-06 | ✓ VERIFIED | `./bin/sup-kv --version` → `sup-kv a631fa5-dirty a631fa5 go1.27.4` (исполнено); TestMain_version PASS (свежий) |
| 23 | Engine.Flush() превращает остаточную dirty (remote merges без waiter) в подтверждённый dump до отмены ctx движка | SC5, 05-06 | ✓ VERIFIED | TestFlush_ (remoteDirtiness / closed / clean) PASS (свежие); Flush экспортирован engine.go:533 с D-19 доком |
| 24 | Поставляемый артефакт — один статический CGO_ENABLED=0 бинарй | SC5, 05-06 | ✓ VERIFIED | `make build` + `file bin/sup-kv` → "statically linked" (исполнено); Makefile build target CGO_ENABLED=0 + ldflags |
| 25 | PendingSnapshot различает пира, который Acked (в Acked), от эвиктированного по peer_timeout (вне обоих множеств) | 05-02 | ✓ VERIFIED | TestRetransmit_ackDrainSemantics PASS (свежий: сортированная вставка, eviction-регрессионный лок, идемпотентность дублей); evictPeers никогда не трогает acked (core.go:768-751 документировано; акцессор core.go:901) |
| 26 | Core.SyncArrived() false до первого DataSync, true навсегда после (A1) | 05-02 | ✓ VERIFIED | TestSync_syncArrived PASS (свежий); флаг в handleDataSync (core.go:559), акцессор core.go:901-902 |
| 27 | gRPC-читатели получают оба состояния репликации из любой горутины через атомики адаптера, не через single-driver core | 05-02 | ✓ VERIFIED | TestMirror_pendingAckVisible + TestMirror_syncArrivedFlip PASS (свежие, реальный event loop); copy-on-write pendingCopy atomic + syncArrived atomic, mirrorState на post-iteration сайте (repl.go:161-162, 301, 469-487) |
| 28 | В core не добавлено коллбеков и каналов — результаты, не коллбеки | 05-02 | ✓ VERIFIED | Структурная проверка Config (core.go, sed по type Config struct): только duration/int поля + давно существующий Log func; фазовый дифф добавил bool-флаг, acked-срез и акцессоры |

**Score:** 28/28 truths verified (0 present, behavior-unverified)

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | systemd unit end-to-end (реальный хост, Type=simple unit) | Phase 7 | Цель Phase 7 покрывает deploy/operate по документации; 05-06-PLAN прямо относит systemd-документацию к Phase 7; 05-VALIDATION.md — единственный manual-only item. Контракт бинарня, который юнит супервизирует (SIGTERM → exit 0 + финальный dump, health readiness, коды 0/1/2), поведенчески доказан на уровне бинарня свежими прогонами этой верификации. |

### Advisory (New Scope, Unevidenced)

None — новых находок вне контракта в этом повторном проходе не выявлено; изменения со времени прошлого прохода (7b55ad1/4f76e26/c194c6f/a631fa5) проверены прямыми прогонами (тесты, lint, buf, регенерация, бинарй) без единого отказа.

### Required Artifacts

gsd-tools `verify artifacts` не применим к строковой форме must_haves.artifacts планов (тот же формат-варианс, что в прошлом проходе); все артефакты проверены вручную на трёх уровнях + data-flow. Прото-артефакты — по НОВЫМ путям (переезд 7b55ad1, роль неизменна):

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/grpcapi/apiproto/supkvapi/v1/supkvapi.proto` | wire-контракт фазы, 7 RPC, формы D-01..D-15, service SupKVService, WatchResponse | ✓ VERIFIED | Прочитан полностью: все 7 RPC (Get/Set/Delete/List/Watch-streaming/SnapshotCreate/SnapshotLoad); create_if_absent/wait_for_replication/wait_timeout_ms/D-08 поля/prefix; buf lint + format exit 0 (исполнено) |
| `internal/grpcapi/apiproto/supkvapi/v1/supkvapi.pb.go`, `supkvapi_grpc.pb.go` | закоммиченный сгенерированный код | ✓ VERIFIED | package apiproto; DO-NOT-EDIT маркеры; `make proto` воспроизводит байт-в-байт (git diff --exit-code, исполнено); RegisterSupKVServiceServer/SupKVServiceServer interface на месте |
| `internal/grpcapi/server.go` | unary-хендлеры + mapError + швы | ✓ VERIFIED | Прочитан полностью: все 7 хендлеров, gate-first записи, таблица шести сентинелов |
| `internal/grpcapi/watch.go` | WatchHub + Watch-хендлер | ✓ VERIFIED | Прочитан полностью: bounded ingress 4096, exact cut (wants, Version.Compare > 0), disconnect-never-drop, nil-hub защитный путь (продакшн always wires, main.go:298/334) |
| `internal/grpcapi/gate.go` | ServerOptions + resolver + health | ✓ VERIFIED | Прочитан полностью: RunGate poll+backstop, CAS openGate, все пять seam-полей |
| `internal/grpcapi/wait.go` | waitForReplication | ✓ VERIFIED | Прочитан полностью: appearance/drain две фазы, Acked-only подсчёт, WR-01 кламп, WR-02 границы |
| `internal/app/storage/store.go` | Config.OnChange tap из install | ✓ VERIFIED | OnChange + nil-нормализация (102-103), write через install, tap с stored record (260) |
| `internal/app/repl/core.go` | SyncArrived + PendingInfo.Acked | ✓ VERIFIED | syncArrived флаг (268), Acked поле PendingInfo (926), ackedInsert sorted/idempotent (512) |
| `internal/udprepl/repl.go` | зеркала адаптера | ✓ VERIFIED | syncArrived/pendingCopy атомики (161-162), mirrorState post-iteration (301, 469-487), drainedRetention (76); импорты обновлены на replproto/supkv/v1 (7b55ad1) |
| `internal/app/persist/engine.go` | экспортированный Flush | ✓ VERIFIED | Flush() → flush() (533), док называет D-19 и exit-sweep |
| `cmd/sup-kv/main.go` | только flags/env/signals/wiring | ✓ VERIFIED | Прочитан полностью: 19 ручек + SUPKV_ двойники, D-19 shutdown, полная проводка ServerOptions, регистрация SupKVService + health, ldflags-переменные |
| `cmd/sup-kv/main_test.go` | семейство smoke бинарня | ✓ VERIFIED | Реальные процессы/TCP; два громких capability-skip с именованием проб (main_test.go:104, wait_integration_test.go:135) |
| `Makefile`, `go.mod`, `go.sum`, `buf.gen.api.yaml`, `buf.gen.repl.yaml`, `buf.yaml` | build/codegen/deps | ✓ VERIFIED | proto = `go tool buf generate` per-tree шаблонами (buf.gen.api.yaml для apiproto, buf.gen.repl.yaml для replproto); buf.yaml zero waivers STANDARD; grpc v1.85.0 direct; tidy-exact; buf.gen.yaml удалён 7b55ad1 (разделён на два шаблона) |

Проводка (import + использование) подтверждена для каждого артефакта: все задействованы main.go, харнессом или собственными потребителями пакетов;孤儿-файлов нет.

### Key Link Verification

Все key_links шести планов прослежены в коде (вручную — верба не применим к прозаической форме):

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| Хендлеры (Get/Set/SetIfAbsent/Delete/List/Snapshot/Restore) | persist.Engine | прямые вызовы движка | ✓ WIRED | server.go:106/143-145/178/202/227/247 |
| Server notify-шов | udprepl.Adapter.NotifyLocalWrite | LocalNotify, nil-safe | ✓ WIRED | main.go:324 инъектирует адаптер; Set/Delete зовут notifyLocalWrite |
| storage.Store.install | Config.OnChange → WatchHub.Ingest | tap → hub ingress | ✓ WIRED | store.go:260 fires; main.go:279 `OnChange: hub.Ingest` через persist.Open |
| Exact-cut подавление | RecordVersion.Compare | строгое доминирование | ✓ WIRED | watch.go:314 `ev.rec.Version.Compare(snapVer) > 0` |
| ServerOptions.SyncArrived → gate resolver → gated() | зеркало адаптера | poll + флаг | ✓ WIRED | main.go:325 `SyncArrived: adapter.SyncArrived`; RunGate поллит; Set/Delete читают gated() |
| Разрешение gate → health.SetServingStatus | связь 1:1 | одиночный CAS | ✓ WIRED | gate.go:152-156; health зарегистрирован main.go:367 рядом с SupKVService (366) |
| waitForReplication | KnownPeers/PendingCopy швы | только инъектированные замыкания | ✓ WIRED | wait.go:95-105/114/188; main.go:326-327 инъектирует зеркала адаптера |
| Snapshot-хендлеры | валидация имён persistfile | единственный закрытый путь | ✓ WIRED | server.go:227/247 → engine.Snapshot/Restore; адаптер маппит ErrInvalidSnapshotName (303-304) |
| handleAck/handleDataSync → core → зеркала адаптера | post-iteration mirrorState | refresh после каждого события | ✓ WIRED | repl.go:301; core.go:500 acked insert, 559 syncArrived |
| Граф проводки main | hub → engine → adapter → server → gate/health → serve | полный цикл | ✓ WIRED | main.go:297-354, вкл. `go hub.Run(apiCtx)` (334), `go srv.RunGate(apiCtx)` (333) |
| SIGTERM → порядок shutdown | D-19 шесть шагов | последовательность | ✓ WIRED | main.go:397-428: NOT_SERVING → apiCancel → bounded GracefulStop/Stop → replCancel+join → Flush → engineCancel |
| Makefile build | статический штампованный bin/sup-kv | CGO_ENABLED=0 + ldflags | ✓ WIRED | Makefile build target; исполнено — statically linked |
| tool-блок go.mod | protoc-gen-go-grpc через go tool, никогда go install | buf-плагины | ✓ WIRED | go.mod tool-блок + buf.gen.api.yaml local go-tool плагины; make proto per-tree |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| Get-хендлер | record | engine.Get (persist поверх реального store) | Yes | ✓ FLOWING |
| List-хендлер | keys | engine.Records() snapshot, фильтр/сортировка | Yes | ✓ FLOWING |
| Watch-поток | events | storage install tap → hub ingress → подписчики | Yes | ✓ FLOWING |
| Wait-отчёт | acked/counts | инъектированные зеркала адаптера (core-acknowledged состояние) | Yes | ✓ FLOWING |
| Health-статус | gate atomic | resolver openGate / shutdown | Yes | ✓ FLOWING |
| Set-ответ | version | engine-штампованный RecordVersion | Yes | ✓ FLOWING |

Статических return, hardcoded-литералов и моков в продакшн-путях ответов нет. (Тестовые fakes — только в _test.go, санкционированная in-memory fake дисциплина.)

### Behavioral Spot-Checks

Все команды исполнены в ЭТОЙ верификации на HEAD a631fa5; именованные тесты — свежие прогоны `-count=1 -race`:

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Unary-поверхность (round trip, error mapping, list) | `go test -race -count=1 ./internal/grpcapi/ -run 'TestServer_setGetRoundTrip\|TestServer_errorMapping\|TestServer_list' -v` | 3/3 PASS | ✓ PASS |
| Watch (snapshot-then-live, exact cut, replicated, prefix, slow consumer) | та же серия `-run 'TestWatch_...' -v` | 5/5 PASS | ✓ PASS |
| Gate/health/snapshot (7 тестов) | `-run 'TestGate_\|TestHealth_\|TestSnapshot_' -v` | 7/7 PASS | ✓ PASS |
| Wait (8 тестов + CR-01 интеграционный) | `-run 'TestWait_\|TestWaitIntegration_...' -v` | 9/9 PASS | ✓ PASS |
| Бинарй: lifecycle/precedence/version | `go test -race -count=1 ./cmd/sup-kv/ -run 'TestMain_...' -v` | 3/3 PASS (реальный процесс/TCP, 3.19s) | ✓ PASS |
| OnChange tap | `go test -race -count=1 ./internal/app/storage/ -run 'TestStore_onChange'` | ok | ✓ PASS |
| Flush-семантика | `go test -race -count=1 ./internal/app/persist/ -run 'TestFlush_' -v` | 3/3 PASS | ✓ PASS |
| Швы репликации (Acked/syncArrived) | `go test -race -count=1 ./internal/app/repl/ -run 'TestRetransmit_ackDrainSemantics\|TestSync_syncArrived' -v` | 2/2 PASS | ✓ PASS |
| Зеркала адаптера | `go test -race -count=1 ./internal/udprepl/ -run 'TestMirror_' -v` | 2/2 PASS | ✓ PASS |
| Полный набор (единственный запуск) | `make test` (go test -race -cover ./...) | exit 0, все пакеты ok (grpcapi 91.1%, storage 100%, persist 97.1%, repl 97.7%) | ✓ PASS |
| Полный lint-гейт | `make lint` (config verify + run ./...) | exit 0, ноль находок | ✓ PASS |
| buf-гейты (zero-waiver) | `go tool buf lint && go tool buf format --diff --exit-code` | оба exit 0 | ✓ PASS |
| Регенерация прото | `make proto && git diff --exit-code -- internal/grpcapi/apiproto/ internal/udprepl/replproto/` | exit 0 — байт-идентично | ✓ PASS |
| Статический артефакт + версия + коды выхода | `make build && file bin/sup-kv && ./bin/sup-kv --version && ./bin/sup-kv` | statically linked; `sup-kv a631fa5-dirty a631fa5 go1.27.4`; exit 2 с usage | ✓ PASS |
| Tidy-гейт | `go mod tidy && git diff --exit-code go.mod go.sum` | exit 0 | ✓ PASS |

### Probe Execution

Фаза не объявляет probe-скриптов (`scripts/*/tests/probe-*` не существуют); runnable-check контракт фазы — сам Go-набор, покрытый выше.

### Requirements Coverage

ID требований из планов: 05-01 [API-01, API-02], 05-02 [API-04, API-05], 05-03 [API-03], 05-04 [API-04, API-06, API-07], 05-05 [API-05], 05-06 [TOOL-01]. Объединение = ровно 8 ID фазы. REQUIREMENTS.md (Traceability) маппит ровно API-01..API-07 + TOOL-01 на Phase 5 со статусом Complete — сиротских требований нет, незаявленных ID нет.

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| API-01 | 05-01 | gRPC-сервис с protobuf; Get/Set (incl. create-if-absent)/Delete с полным набором ошибок | ✓ SATISFIED | Истины 1-3; proto + хендлеры + TestServer_errorMapping (свежий PASS) |
| API-02 | 05-01 | List ключей под префиксом в сортированном порядке | ✓ SATISFIED | Истина 4; TestServer_list (свежий PASS) |
| API-03 | 05-03 | Watch-поток, покрывающий локальные и реплицированные изменения | ✓ SATISFIED | Истины 6-10; TestWatch_* семейство (свежие PASS) |
| API-04 | 05-02, 05-04 | Записи Unavailable во время startup gate; чтения немедленно | ✓ SATISFIED | Истины 11-12, 25-26; TestGate_* + TestSync_syncArrived (свежие PASS) |
| API-05 | 05-02, 05-05 | wait_for_replication: ожидание Ack в пределах timeout, отчёт, исход записи не меняется | ✓ SATISFIED | Истины 15-19, 27; TestWait_* + CR-01 интеграционный (свежие PASS) |
| API-06 | 05-04 | Стандартный health-endpoint отражает gate (NOT_SERVING до открытия) | ✓ SATISFIED | Истина 13; TestHealth_reflectsGate (свежий PASS) |
| API-07 | 05-04 | Snapshot create/load через API | ✓ SATISFIED | Истина 14; TestSnapshot_* (свежие PASS) |
| TOOL-01 | 05-06 | Один статический бинарй, flags/env конфигурация, --version, systemd-friendly | ✓ SATISFIED | Истины 20-24; TestMain_* + make build + исполненные пробы бинарня |

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| internal/grpcapi/server_test.go | API-01/02 | 3 | 0 | none | value/behavioral (коды, байты, порядок) | adequate |
| internal/grpcapi/watch_test.go | API-03 | 5 | 0 | none | behavioral (exact cut, ResourceExhausted, store liveness) | adequate |
| internal/grpcapi/gate_test.go | API-04/06/07 | 7 | 0 | none | behavioral (обе стороны gate, враждебные имена, round trip поверх реального persistfile) | adequate |
| internal/grpcapi/wait_test.go | API-05 | 8+ | 0 | none | behavioral (adversarial mirrors, тайминговые границы, клампы) | adequate |
| internal/grpcapi/wait_integration_test.go | API-05 (CR-01) | 1 | громкий capability-skip (loopback broadcast) | none | end-to-end behavioral (два реальных адаптера, acked=true 1/1) | adequate |
| internal/app/repl/retransmit_test.go, sync_test.go | API-04/05 швы | расширенные семейства | 0 | none | value/behavioral | adequate |
| internal/udprepl/repl_test.go | API-05 зеркала | TestMirror_* 2 | 0 | none | behavioral поверх реального event loop | adequate |
| internal/app/storage/store_test.go | API-03 tap | TestStore_onChange (8 субтестов) | 0 | none | value/behavioral (loser silence, stored-record equality) | adequate |
| internal/app/persist/engine_test.go | TOOL-01 Flush | TestFlush_* 3 | 0 | none | value (байты dump содержат запись) | adequate |
| cmd/sup-kv/main_test.go | TOOL-01 | 3 | громкий skip-путь (go-tool probe) | none | end-to-end behavioral (реальный процесс/TCP, коды выхода, содержимое dump) | adequate |

**Disabled tests on requirements:** 0 (два документированных громких capability-probe skip-а именуют пробуемое: loopback broadcast — wait_integration_test.go:135; go-tool — main_test.go:104)
**Circular patterns detected:** 0
**Insufficient assertions:** 0

### Decision Coverage

Все отслеживаемые решения CONTEXT.md соблюдены (gate `check.decision-coverage-verify`, исполнен в этой верификации): 19/19 honored, 0 not honored — включая D-01..D-07 (watch-контракт под новыми wire-именами), D-08..D-11 (wait-отчёт), D-12..D-15 (list), D-16..D-19 (бинарй/конфигурация/shutdown).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | — | TBD/FIXME/XXX маркеры долга во всех файлах фазы (вкл. переименованные) — ноль | — | — |
| (none) | — | Заглушек нет: единственные Unimplemented-return — обязательный embed protoc-gen-go-grpc v1.6.2 и nil-hub защитный путь watch.go:366 (продакшн always wires хаб; документирован как guard от programming error) | — | — |
| internal/grpcapi/server.go | 306-308 | IN-01: недостижимая mapError status-пасsthru ветка (necessity) — открыта на info-уровне | ℹ️ Info | Не блокирует; записана в 05-REVIEW-DISPOSITION.md (disposition: open); защитная ветка безвредна, найдена повторно 05-REVIEW.md от 2026-11-14 |

### Review Findings Disposition (контекст)

CR-01 (wait_for_replication не мог дать acked=true против реального зеркала), WR-01 (Ack вне wait-set завышали acked_count), WR-02 (overflow wait_timeout_ms) — исправлены и повторно доказаны свежими прогонами этой верификации: TestWaitIntegration_ackedThroughRealAdapterMirror PASS (истина 18), TestWait_overflowingTimeoutClamped PASS (19), WR-01 кламп в waitForDrain wait.go:158 (17). IN-01 — info, открыта, внеблокирующая. 05-SECURITY.md (a631fa5, ретроспективно): 12/12 угроз закрыты, threats_open: 0; T-05-01 (no-auth LAN) и T-05-06 (нет капа подписчиков) — принятые риски с записью в Accepted Risks Log.

### Human Verification Required

N/A — инфраструктурная/фундаментная фаза (внутренний gRPC API + бинарй; нет UI или пользовательского визуального интерфейса). Все критерии приёмки проверяемы программно и проверены выше; каждая поведенческая истина покрыта именованным тестом, запущенным свежим прогоном в этой верификации. Единственный deployment-environment item (systemd unit end-to-end) отложен в Phase 7 с обоснованием в Deferred Items — контракт бинарня на его стороне поведенчески доказан полностью.

### Gaps Summary

Отсутствуют. Все 28 must-have истин проверены с поведенческим доказательством на текущем HEAD (a631fa5) с контрактом под новыми именами (SupKVService / WatchResponse / apiproto/supkvapi/v1); все артефакты субстантивны и подключены; все key links прослежены; все 8 ID требований удовлетворены без сирот; гейты (make test -race, make lint, buf lint/format zero-waiver, регенерация прото, tidy, статический бинарь с --version и кодами выхода) зелёные в собственных прогонах этой верификации; цель фазы — единственный статический бинарй с типизированной gRPC-поверхностью Watch, startup gate, wait_for_replication и стандартным health-checking — наблюдаемо достигнута.

---

_Verified: 2026-11-15T18:27:10Z_
_Verifier: Claude (gsd-verifier)_
