---
phase: 01-ci-bootstrap-and-repo-skeleton
verified: 2026-11-11T11:13:23Z
status: passed
score: 25/25 must-haves verified
covered_files:
  - .claude/CLAUDE.md
  - .github/workflows/ci.yml
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-01-PLAN.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-01-SUMMARY.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-02-PLAN.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-02-SUMMARY.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-03-PLAN.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-03-SUMMARY.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-04-PLAN.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-04-SUMMARY.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-05-PLAN.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-05-SUMMARY.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-06-PLAN.md
  - .planning/phases/01-ci-bootstrap-and-repo-skeleton/01-06-SUMMARY.md
  - Makefile
  - cmd/sup-kv/main.go
  - go.mod
  - go.sum
  - internal/app/storage/store.go
  - internal/app/storage/store_test.go
covered_digest: "v4:sha256:ecd441ea46bf652739861cc7e581020ebac64da22eeb8074c32210d47f00a2a8"
behavior_unverified: 0
overrides_applied: 4
re_verification:
  previous_status: passed
  previous_score: 25/25
  gaps_closed:
    - "Триггер регенерации: покрытые входы изменились после прогона 2026-11-08T18:01:26Z — buf-миграция (ed880c7/9727c80: .claude/CLAUDE.md, ci.yml, Makefile) и работа фазы 5 (46633f6: cmd/sup-kv/main.go; a09b8b0: go.mod/grpc + go.sum; c2746cb: internal/app/storage/{store,store_test}.go; правки planning-доков). Этот прогон заново вывел каждый предикат по текущему дереву и пересчитал covered-input fingerprint: новый digest по 20 входам (8 изменившихся файлов реализации + автоматически добавленные PLAN/SUMMARY; planning-root доки инертны per #4623; все прочие покрытые файлы git-подтверждённо байт-неизменны с прошлых прогонов и наследуют их вердикты)."
  gaps_remaining: []
  regressions: []
overrides:
  - must_have: "Direct pushes to master are rejected by branch protection; a red gate blocks the merge"
    reason: "Rulesets API 403 on private org repo (plan-gated); payload + one-command re-apply recorded in docs/setup/branch-protection.md; defer decision taken 2026-10-31 at the plan 01-04 blocking checkpoint ; ratified by the maintainer as an accepted override 2026-10-31 (no org plan upgrade, no public flip)"
    accepted_by: "onokonem"
    accepted_at: "2026-10-31T18:41:25Z"
  - must_have: "Dependabot is active (updates and alerts)"
    reason: "Alerts GHAS-gated on private repo tier; version updates + govulncheck delivered and green; alerts unlock at the deferred public flip ; ratified by the maintainer as an accepted override 2026-10-31."
    accepted_by: "onokonem"
    accepted_at: "2026-10-31T18:41:25Z"
  - must_have: "Dependabot opens its first version-update PR within the weekly schedule window"
    reason: "Window elapsed (Mon 2026-11-02 06:00 UTC cron) and no version-update PR exists because nothing is updatable: golangci-lint v2.15.0 is the latest upstream release (verifier-confirmed live via the GitHub releases API this pass) and is deliberately ignored in dependabot.yml (WR-01 — must move in lockstep with the vendored 2.15.0-exact strict config); the weekly machinery demonstrably fires — the vulncheck cron ran on schedule 2026-11-02 with conclusion success (verifier-observed run 41466437895). Maintainer decision 2026-11-02 (UAT Test 1, option A): the backstop's substance — Dependabot wired and healthy — is satisfied; a version-update PR will appear when an upstream release actually exists."
    accepted_by: "onokonem"
    accepted_at: "2026-11-02T15:55:45Z"
advisory:
  - finding: "WR-01 (ревью 2026-11-11): CI-задачи lint/gate в ci.yml не задают timeout-minutes — зависшая задача держит раннер до 6-часового дефолта GitHub (до пяти раннеров на коммит при fail-fast:false)"
    category: other
    reason: "Новая находка инкрементального ревью на файле, изменённом с прошлой верификации; ни один must_have фазы 1 не нарушает (таймауты задач в контракт фазы не входят); disposition: open в 01-REVIEW-DISPOSITION.md. Разрешается фикс timeout-minutes: 15/20 в ci.yml"
    evidence_status: "none provided — это совет по устойчивости/стоимости, детерминированного red-теста не существует (текущее поведение — задокументированный дефолт GitHub 360 мин)"
  - finding: "IN-01 (ревью 2026-11-11): serve() в cmd/sup-kv/main.go отпускает обработку сигналов (defer stop()) до завершения shutdown — второй SIGTERM/SIGINT в окне graceful-shutdown убивает процесс без финального write-confirmed дампа"
    category: other
    reason: "Находка инкрементального ревью на коде фазы 5 (main.go — покрытый файл, изменённый с прошлой верификации); контракт фазы 1 не нарушает; disposition: open в 01-REVIEW-DISPOSITION.md; экспозиция ограничена (write-confirmed записи уже на диске, remote-merged dirt восстанавливается с пиров)"
    evidence_status: "none provided — поведенческое окно гонки сигнал/ширкдаун, воспроизводимого red-теста в этом прогоне не запускалось"
---

# Phase 1: CI Bootstrap and Repo Skeleton — отчёт верификации

**Phase Goal:** Quality gates and the enforced repo skeleton exist before any feature code, so every subsequent change lands inside the architecture and behind a red-light-blocking gate
**Verified:** 2026-11-11T11:13:23Z
**Status:** passed
**Re-verification:** Да — санкционированная регенерация устаревшего отчёта (прошлый прогон passed 25/25, verified 2026-11-08T18:01:26Z; инвалидирован изменениями покрытых входов от работы фаз 4-5: buf-миграция — .claude/CLAUDE.md, ci.yml, Makefile; фаза 5 — cmd/sup-kv/main.go, go.mod (grpc v1.85.0), go.sum, internal/app/storage/{store,store_test}.go; правки planning-доков)

## Regeneration Scope

Дифф покрытого множества с момента прошлого верифицированного состояния (рабочее дерево — ветка
`gsd/phase-4-replication-and-deterministic-simulator`, впереди origin-ветки на 38 коммитов; origin/master
неизменен на 617951e):

- **buf-миграция (ed880c7, 9727c80, 2026-11-09)**: Makefile +43 строки (цели proto/buf-lint/buf-format/buf-breaking; гейт-цели lint/test/generate/build не тронуты), ci.yml +20 строк (шаг Buf proto schema gates в lint-задаче + fetch-depth: 0; матрица, permissions, concurrency, standing config-verify не тронуты), .claude/CLAUDE.md (10 строк, правка тулинга; deferral-parenthetical цела — см. G6).
- **Фаза 5 (a09b8b0, 46633f6, c2746cb)**: go.mod — второй санкционированный прямой require `google.golang.org/grpc v1.85.0` + рост tool-блока до 5 директив (buf, protoc-gen-go, protoc-gen-go-grpc); go.sum соответственно; cmd/sup-kv/main.go — 507 строк полного wiring (doc-комментарий цел, main() держит только flags/env/signals/wiring по layout-контракту AGENTS.md); internal/app/storage/store.go/store_test.go — OnChange-tap (пакет остаётся чистым — красная проба net подтверждает).
- **Planning-доки**: PROJECT.md/REQUIREMENTS.md/STATE.md/WINDOWS.md — обновления строк фаз 4-5; каждый маркер фазы 1 перепроверен почиткой (истины G3-G8).

Все прочие покрытые файлы git-подтверждённо байт-неизменны с прошлых прогонов (dependabot.yml, vulncheck.yml,
.gitignore, .golangci.yml, AGENTS.md, branch-protection.md, ruleset.json, clock/version/testutil-файлы,
storage/doc.go+errors.go+merge_test.go, internal/port/doc.go). Изменившиеся входы получили полную
реверификацию: локальные гейты перезапущены живьём, обе depguard-красные пробы перезапущены, локальный
govulncheck по новому графу, live-запросы GitHub, свежий upstream-дифф вендоренного конфига, fingerprint
пересчитан. Ничего из SUMMARY и из claims прошлого отчёта не принято на веру.

Рабочее дерево: два untracked-файла (`.planning/state.json`, `04-REVIEW-FIX.md` фазы 4) и незакоммиченная
правка `.planning/STATE.md` (phase-pointer верификационного workflow) — planning-книжчество, не покрытые
входы; `git status internal/` чист, go.mod/go.sum без churn.

## Goal Achievement

### Observable Truths

Тот же контракт из 25 истин, что в прошлом отчёте (4 критерия успеха ROADMAP, разбитые на независимо
проверяемые половины + дедуплицированные истины PLAN + 8 record-keeping истин G1-G8). Вся доказательная
база собрана этим верификатором в этом прогоне против текущего дерева.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC-1a: PR запускает полную матрицу гейтов (strict lint, go vet, race-тесты) на Linux/macOS × amd64/arm64 | ✓ VERIFIED | ci.yml перечитан: push(master)+pull_request, permissions contents:read, go-version-file: go.mod (обе задачи), pin v2.15.0 full-patch, fail-fast:false, 4 ячейки; buf-гейты добавлены шагом в lint-задачу (усиление, не ослабление). Live: `gh api commits/617951e/check-runs` — lint + 4 gate-ячейки success; последний CI-прогон PR #31 — success (run 42514571899, 2026-11-08T19:20Z) |
| 2 | SC-1b: красный гейт блокирует merge | ✓ PASSED (override) | Красный путь перепроован живьём: обе depguard-пробы дали exit 1 с сообщениями своих списков (истина 7); на PR #31 гейт дважды честно красный до фиксов (runs 42509010340, 42513202587 — failure), merge не состоялся (master 617951e не двигался). Серверная половина — ратифицированный override №1 (accepted_by onokonem, 2026-10-31T18:41:25Z; предикат 403 переподтверждён live) |
| 3 | SC-2a: прямые push в master отклоняются branch protection | ✓ PASSED (override) | `gh api repos/Djarvur/sup-kv/rulesets` → 403 «Upgrade to GitHub Pro or make this repository public» — предикат override жив (repo private=true, default=master переподтверждены). Все маркеры целы: REQUIREMENTS CI-03 suffix, branch-protection.md DEFERRED, WINDOWS waiver, STATE decision log |
| 4 | SC-2b: каждое изменение (код и доки) идёт через PR | ✓ VERIFIED | Первая-родительская практика: origin/master не двигался с 617951e; вся работа фаз 4-5 идёт через открытый PR #31 (branch gsd/phase-4-...); 8 незапушенных локальных master-коммитов перепросканированы per-commit — все касаются только .planning-доков, ноль файлов реализации |
| 5 | SC-3a: Dependabot активен (updates и alerts) | ✓ PASSED (override) | Updates: dependabot.yml git-неизменен с прошлого прогона; ignore golangci-lint/v2 с lockstep-обоснованием цел (старый WR-01, fixed). Alerts: GHAS-гейт на private tier — ратифицированный override №2; REQUIREMENTS CI-02 честно Partial (перечитано после правок фаз 4-5) |
| 6 | SC-3b: запланированный govulncheck зелёный против БД уязвимостей Go | ✓ VERIFIED | Live: `gh run list --workflow=vulncheck.yml` — свежий недельный cron run 42644047095, event schedule, success, 2026-11-09T15:22:42Z (машина стреляет и после buf-миграции); плюс два PR-прогона vulncheck на ветке фазы 4 — success. Локально: `go tool govulncheck ./...` exit 0 на текущем графе (включая grpc v1.85.0). Заметное отличие от прошлого прогона: граф теперь содержит required-but-not-called уязвимости из grpc-зависимостей — семантика гейта («called vulns fail», exit 0/3) не нарушена, скан зелёный |
| 7 | SC-4: граница зависимостей machine-enforced (net/os/grpc в internal/app, третьи стороны в internal/port — красный гейт) | ✓ VERIFIED | Обе красные пробы перезапущены на текущем дереве не-тестовыми файлами с правильными именами пакетов: `net` в internal/app/clock → `import 'net' is not allowed from list 'sup-kv-app-no-io'`; `golang.org/x/mod/semver` в internal/port → `not allowed from list 'sup-kv-port-stdlib-only'`. После удаления проб lint зелёный, `git status internal/` чист. Граница держит на графе с grpc/protobuf: deny `google.golang.org/` в internal/app байт-идентичен проверенному состоянию (конфиг git-неизменен, pristine-дифф — истина 8) |
| 8 | .golangci.yml — вендоренный powerman 2.15.0, байт-нетронутый, ровно две аддитивные sup-kv правила | ✓ VERIFIED | Свежий fetch upstream-конфига 2.15.0 + diff: единственный hunk `64a65,88` — чистая вставка (два правила + баннер), ноль upstream-строк удалено/изменено; Origin-заголовок строка 1 цел; файл git-неизменен с e456764 |
| 9 | go.mod пинует toolchain (golangci-lint v2.15.0, govulncheck v1.9.0) с минимальным policy-санкционированным runtime-графом | ✓ VERIFIED | Tool-блок: 5 директив (buf, golangci-lint v2.15.0 — строка 137, govulncheck x/vuln v1.9.0 — строка 292, protoc-gen-go, protoc-gen-go-grpc). Прямые require ровно пара санкционированных AGENTS.md («gRPC and protobuf enter go.mod in the phases that first use them»): grpc v1.85.0 + protobuf v1.36.13-0.20261016172727; google/uuid и x/sync остаются indirect. `go mod tidy` идемпотентен (`git diff --exit-code go.mod go.sum` чист) |
| 10 | Strict lint-гейт жив и зелён; config verify exit 0 (QUAL-01) — постоянный гейт в Makefile и CI | ✓ VERIFIED | Вербификатор-прогон: `make lint` → exit 0 (config verify, затем run ./...) на полном текущем дереве. Постоянный verify-шаг перечитан в Makefile (строка 12) и ci.yml (строка 30); локальный/CI пин одинаков — v2.15.0 (go.mod 137 / ci.yml 51) |
| 11 | Race-гейт не пустой (CI отчитывается ok по internal/app) | ✓ VERIFIED | Вербификатор-прогон с обходом кэша: `go build ./... && go vet ./... && go test -race -count=1 ./...` → exit 0; ok для 10 тестовых пакетов: cmd/sup-kv 3.093s, clock 1.070s, simnet 1.041s, persist 2.770s, repl 1.251s, storage 1.582s, version 1.035s, grpcapi 2.175s, persistfile 5.864s, udprepl 1.068s — фазовые race-цели clock/storage/version зеленны на текущем коде (включая OnChange-tap фазы 5 в storage) |
| 12 | Ячейки матрицы независимы (fail-fast выключен) | ✓ VERIFIED | ci.yml перечитан: `fail-fast: false` с комментарием-обоснованием (строка 57); дифф buf-миграции матрицу и стратегию не трогал |
| 13 | Push и PR одного ref не отменяют друг друга; superseded-прогоны отменяются | ✓ VERIFIED | ci.yml перечитан: concurrency group `ci-${{ github.event_name }}-${{ github.ref }}`, `cancel-in-progress: true` (строки 14-16) |
| 14 | ruleset.json кодирует полный payload защиты master; контексты совпадают с именами check-run'ов дословно | ✓ VERIFIED | Python-сравнение: 5 контекстов == 5 именам ci.yml-check-run'ов на 617951e (lint + 4 gate-ячейки; совпадение 5/5; в live-списке дополнительно есть Dependabot/vuln из других workflows — они не контексты ruleset). Типы правил [pull_request, required_status_checks, non_fast_forward, deletion]; strict_required_status_checks_policy true; conditions ref_name refs/heads/master |
| 15 | delete_branch_on_merge включён на репозитории | ✓ VERIFIED | `gh api repos/Djarvur/sup-kv` → delete_branch_on_merge=true, default_branch=master, private=true |
| 16 | branch-protection.md хранит apply/verify/re-apply команды, zero-approval rationale, context-coupling и текущее ратифицированное состояние | ✓ VERIFIED | Перегреплено: `Status: DEFERRED (accepted override)` (строка 14), `Deferral decision 2026-10-31` (строка 24), ноль PENDING-заголовков; файл git-неизменен с 0a4ca44 |
| 17 | Backstop: Dependabot открывает первый version-update PR в недельном окне | ✓ PASSED (override) | `gh pr list --author app/dependabot --state all` → 0; `gh api repos/golangci/golangci-lint/releases/latest` → v2.15.0 == пин (переподтверждено live 2026-11-11 — по-прежнему нечего обновлять); недельная машина стреляет — свежий scheduled vulncheck 42644047095 (schedule, success, 2026-11-09). Все фактические предикаты override №3 живы |
| G1 | (01-05 T1) overrides: frontmatter несёт ратифицированные записи дословно, accepted_by «onokonem», реальные UTC accepted_at | ✓ VERIFIED | Все три записи перенесены в frontmatter этого отчёта дословно (№№ 1-2 — 2026-10-31T18:41:25Z; № 3 — санкционированное post-UAT дополнение 2026-11-02T15:55:45Z) |
| G2 | (01-05 T2) Правки gap-closure не меняли тело отчёта вне overrides-блока | ✓ VERIFIED | Исторически: PR #17 — только overrides-блок. Тело с тех пор легитимно переписывалось только верификатором (санкционированные регенерации, включая эту); record-keeping план 01-05 это предвидит; fingerprint пересчитывается каждый раз |
| G3 | (01-05 T3) REQUIREMENTS.md CI-02 аннотирован Partial: чекбокс снят, строка Partial с ратифицированной оговоркой | ✓ VERIFIED | Перечитано после правок фаз 4-5: `- [ ] **CI-02**` (строка 84); трейсабилити-строка «Partial — version updates + scheduled govulncheck delivered and green; alerts half deferred (GHAS-gated on private tier), ratified accepted override 2026-10-31; unlocks at GHAS license or the deferred public flip» (строка 140) цела |
| G4 | (01-05 T4) REQUIREMENTS.md CI-03 фиксирует ратификацию деферрала | ✓ VERIFIED | Строка 141: «ratified 2026-10-31 as accepted override (no org plan upgrade, no public flip)» после решения `defer`, упоминание re-apply path цело |
| G5 | (01-06 T1) branch-protection.md: статус DEFERRED (accepted override), параграф ратификации, строка решения, все команды дословно | ✓ VERIFIED | См. истину 16: заголовок + строка решения присутствуют ровно по разу, PENDING нет |
| G6 | (01-06 T2) PROJECT.md и .claude/CLAUDE.md Process constraints несут идентичную deferral-parenthetical | ✓ VERIFIED | `server-side ruleset deferred 2026-10-31` ровно по разу в каждом (CLAUDE.md:17, PROJECT.md:77); оба файла пережили buf-миграцию и правки фазы 5 — parenthetical цел и идентичен в зеркалах |
| G7 | (01-06 T3) WINDOWS.md: пункт 2 waived с reason override; ledger консистентен; пункт 1 resolved | ✓ VERIFIED | `gsd-tools windows status`: item 1 — fixed (resolved_at 2026-11-02), item 2 — waived, reason цитирует ратифицированный override, 403 и re-apply path. Замечание: в ledger появился третий пункт id=3 с phase=05 (Watch/Snapshot RPC) — вне контракта фазы 1, см. Anti-Patterns |
| G8 | (01-06 T4) STATE.md decision log: по одной записи на каждый ратифицированный override с provenance плана | ✓ VERIFIED | Ровно по одной «Verification gap 1/2 ratified 2026-10-31» (строки 114-115) + provenance-записи плана 01-05 (строки 111-113) — decision log пережил 16 коммитов правок STATE и незакоммиченный phase-pointer |

**Score:** 25/25 истин verified (4 через ратифицированные maintainer'ом override; 0 present-but-behavior-unverified)

### Override Ratification Audit (Step 3b)

Три ратифицированных override переносятся во frontmatter дословно. Фактические предикаты перепроверены
live в этом прогоне: rulesets endpoint по-прежнему 403 на по-прежнему private репо (№ 1); Dependabot PR
по-прежнему ноль, потому что golangci-lint v2.15.0 по-прежнему upstream-latest (переспрошено 2026-11-11),
а недельная машина демонстративно стреляет зелёным — свежий scheduled vulncheck 2026-11-09 (№ 3).
Сопоставление по token-overlap: № 1 → истины 2+3, № 2 → истина 5, № 3 → истина 17 — итого 4 PASSED
(override). Все record-keeping маркеры рассказывают одну историю (G3-G8 перепроверены по текущим
planning-файлам).

### Deferred Items

Нет — ни один gap фазы 1 не запланирован в более позднюю фазу milestone; исходные два gap закрыты
ратифицированными override (№№ 1-2), backstop — override № 3. Деффернутые половины (применение ruleset,
GHAS alerts) остаются в одном командном/биллинг-решении от закрытия, трекируются в WINDOWS.md (waived) и
docs/setup/, а не в поздней фазе.

### Advisory (New Scope, Unevidenced)

Повторная верификация выполнена; Step-7 скан по всем изменившимся покрытым файлам (buf-миграция + файлы
фазы 5 + planning-доки) не нашёл ни блокеров, ни unreferenced debt-маркеров, ни стаб-паттернов. Две
открытые находки инкрементального ревью 2026-11-11 (disposition open в 01-REVIEW-DISPOSITION.md) —
new-scope, детерминированного доказательства не имеют, фиксируются как advisory, не блокируют:

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| 1 | WR-01: ни одна CI-задача не задаёт timeout-minutes (6-часовой дефолт GitHub × fail-fast:false) | other | new-scope, нет детерминированного red-теста; не нарушает ни один must_have фазы; recorded open в disposition ledger |
| 2 | IN-01: defer stop() отпускает сигналы до конца shutdown в main.go (код фазы 5) | other | new-scope, окно гонки сигнал/ширкдаун не воспроизведено тестом в этом прогоне; контракт фазы 1 не затрагивает; recorded open |

### Required Artifacts

gsd-tools `verify.artifacts`: 18/20 по всем 6 планам (01-02…01-06 all_passed true; 01-01 — 6/8). Два
«промаха» — известные superseded walking-skeleton пути `internal/app/doc.go` и `internal/app/app_test.go`
(декларации плана 01-01), удалённые PR #28 в пользу реального движка. Их декларированное назначение
выполнено заменителями и проверено live в этом прогоне: strict lint зелён (package-comments гейт по
полному дереву; storage несёт собственный doc.go), `go test -race -count=1` ok по
internal/app/{clock,persist,repl,storage,version} + simnet + grpcapi + persistfile + udprepl. Supersession,
не gap.

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| go.mod | module + tool directives + пины инструментов | ✓ VERIFIED | CHANGED (buf-миграция + фаза 5, санкционировано): 5 tool-директив; ровно два прямых require (grpc + protobuf) per AGENTS.md policy; tidy идемпотентен; lint/test/govulncheck зелёные на новом графе |
| cmd/sup-kv/main.go | Doc-комментарий команды, main() без тестируемой логики | ✓ VERIFIED | CHANGED (фаза 5): 507 строк — config resolution, version, wiring, ordered graceful shutdown; doc-комментарий цел; main() держит flags/env/signals/wiring по layout-контракту AGENTS.md; config-логика покрыта main_test.go (ok 3.093s под -race); cross-package wiring перепроверен ревью 2026-11-11 |
| internal/port/doc.go | Package comment, stdlib-only target | ✓ VERIFIED | git-неизменен с прошлого прогона; package comment цел; port-красная проба fired в этом прогоне |
| internal/app/doc.go | Package comment, pure-core target | ✓ SUPERSEDED (назначение выполнено) | удалён PR #28; пакеты internal/app документированы per-package; package-comments гейт lint зелён в этом прогоне |
| internal/app/app_test.go | Реальная race-цель (D-01) | ✓ SUPERSEDED (назначение выполнено) | удалён PR #28; заменён реальными race-протестированными сьютами clock/storage/version/persist/repl (+ grpcapi, persistfile, udprepl, simnet) — все ok под -race в этом прогоне |
| .golangci.yml | Вендоренный strict config + boundary | ✓ VERIFIED | git-неизменен (e456764); LIVE pristine-дифф против свежего upstream fetch: единственная чистая вставка, ноль upstream-строк тронато; обе boundary-пробы fired red |
| Makefile | lint/test/generate/build через go tool | ✓ VERIFIED | CHANGED (buf-миграция): buf-цели ДОБАВЛЕНЫ; четыре гейт-цели не тронуты; `make lint` verifier-run exit 0 (verify + run) |
| AGENTS.md | Конвенции + 3 deviations + reading D-03 | ✓ VERIFIED | git-неизменен; все три deviation, reading D-03, правило go tool перечитаны целыми — dependency policy санкционирует go.mod-изменения фаз 5 (grpc clause) |
| .github/workflows/ci.yml | lint + 4-ячеечная матрица + гигиена | ✓ VERIFIED | CHANGED (buf-миграция, +20 строк): триггеры/permissions/go-version-file/пин/fail-fast/concurrency целы; buf-гейты добавлены шагом; 5/5 gate/lint check-runs зелёные live на 617951e |
| .github/workflows/vulncheck.yml | cron + paths trigger + dispatch | ✓ VERIFIED | git-неизменен; scheduled прогон зелёный 2026-11-09 (свежий, run 42644047095) |
| .github/dependabot.yml | gomod + github-actions updates | ✓ VERIFIED | git-неизменен; WR-01-lockstep ignore (старый WR-01, fixed) цел |
| docs/setup/ruleset.json | Payload ruleset, 4 типа правил, 5 контекстов | ✓ VERIFIED | git-неизменен; контексты == имена live check-run'ов ci.yml переподтверждены в этом прогоне (5/5) |
| docs/setup/branch-protection.md | Apply/verify/re-apply + ратифицированный DEFERRED-маркер | ✓ VERIFIED | git-неизменен; маркеры перегреплены |
| .planning/REQUIREMENTS.md | CI-02 Partial + CI-03 ratification suffix | ✓ VERIFIED | перечитано после правок фаз 4-5; строки фазы 1 целы |
| .planning/WINDOWS.md | ledger пункт 2 waived; пункт 1 fixed | ✓ VERIFIED | open 0/waived 1/fixed 1 по пунктам фазы 1 (tooling-прогон); третий open-пункт — phase=05, вне контракта |
| .planning/STATE.md | decision-log записи по обоим override | ✓ VERIFIED | ровно две gap-ратификации + provenance 01-05 пережили rewrite'ы STATE фаз 4-5 и незакоммиченный phase-pointer |
| .planning/PROJECT.md | Process constraint, честно фиксирующий деферрал | ✓ VERIFIED | правки только строк фаз 4-5; parenthetical ровно один раз |
| .claude/CLAUDE.md | сгенерированное зеркало в синхроне (IN-02 закрыт) | ✓ VERIFIED | CHANGED (buf-миграция — правки тулинга): parenthetical пережил, посимвольно идентичен источнику |

### Key Link Verification

gsd-tools `verify.key-links`: 9/15 automated-verified; все 6 «промахов» — известное ограничение тула
(описательные `from:`, не являющиеся относительными путями: «from: must be a relative file path»); каждый
перепроверен вручную прямым доказательством в этом прогоне.

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| Makefile | go.mod | `go tool golangci-lint` | ✓ WIRED | Automated (01-01: 3/3); плюс новые buf-цели через `go tool buf` |
| .golangci.yml | internal/app | `**/internal/app/**` deny rule | ✓ WIRED | Automated; красная проба re-fired в этом прогоне — срабатывает на текущем дереве с grpc/protobuf-графом |
| .golangci.yml | internal/port | `**/internal/port/**` strict list-mode | ✓ WIRED | Automated; красная проба re-fired |
| ci.yml | go.mod | `version: v2.15.0` + `go-version-file` | ✓ WIRED | Automated; absorbs go.mod-изменения (buf + grpc) без правки workflow |
| ci.yml | .golangci.yml | lint job гоняет вендоренный config + standing verify | ✓ WIRED | Automated; строка 30 (verify) + строка 49-51 (action с пином) |
| ci.yml (gate display names) | docs/setup/ruleset.json | контексты дословно | ✓ WIRED (manual) | Python-сравнение в этом прогоне: 5/5 на 617951e |
| vulncheck.yml | go.mod | `go tool govulncheck` | ✓ WIRED | Automated (01-03: 2/2); scheduled прогон зелёный на новом графе |
| dependabot.yml | repo root | `directory: /` обе экосистемы | ✓ WIRED | Automated |
| ruleset.json | ci.yml check-run names | контексты == display names | ✓ WIRED | Automated (01-04: 2/2) |
| branch-protection.md | GitHub rulesets API | перезапускаемые gh api команды | ✓ WIRED | Automated; GET перезапущен live (403 как записано) |
| overrides-записи 01-VERIFICATION.md | ратифицированное override-состояние | дословно + клауза ратификации | ✓ WIRED (manual) | Все три записи перенесены дословно; предикаты перепроверены live |
| REQUIREMENTS.md CI-02 строка | CI-03 строка | одинаковый паттерн Partial-with-explanation | ✓ WIRED (manual) | Обе строки перечитаны — форма одна, строки фазы 1 целы |
| .claude/CLAUDE.md Process line | PROJECT.md Process constraint | идентичная parenthetical | ✓ WIRED (manual) | `server-side ruleset deferred 2026-10-31` ровно по разу в каждом, текст идентичен |
| WINDOWS.md пункт 2 reason | ратифицированный override 2026-10-31 | reason называет ратификацию + re-apply path | ✓ WIRED (manual) | reason перечитан через windows status — цитирует оба |
| branch-protection.md status | override entry 1 (план 01-05) | DEFERRED-маркер ссылается на ратификацию | ✓ WIRED (manual) | заголовок строки 14 + решение строки 24 перегреплены |

### Data-Flow Trace (Level 4)

UI/рендеримых данных в фазе нет. Инфраструктурные эквиваленты переподтверждены: lint job ← .golangci.yml
(схема верифицируется постоянным `config verify`, зелёный в make lint); setup-go ← go.mod
(go-version-file — поглотил buf/grpc-изменения go.mod без правки workflow); govulncheck ← tool-граф
go.mod (пин v1.9.0; скан нового графа exit 0); buf-гейты ← go.mod-пиновый `go tool buf` (локально
lint/format exit 0); check-runs ← реальное исполнение матрицы (5/5 success наблюдено live на 617951e);
ledger-счётчики ← gsd-tools windows status читает реальный файл. Ни статических, ни hollow-значений.

### Behavioral Spot-Checks

Все команды исполнены верификатором в свежем шелле в этом прогоне; probe-файлы созданы и удалены
верификатором; дерево чисто после.

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Build + vet + race tests (кэш обходом) | `go build ./... && go vet ./... && go test -race -count=1 ./...` | exit 0; ok 10 тестовых пакетов (см. истину 11) | ✓ PASS |
| Полный локальный гейт incl. standing verify | `make lint` | exit 0 (config verify + run ./... зелёные, полное дерево) | ✓ PASS |
| Идемпотентность module-графа | `go mod tidy && git diff --exit-code go.mod go.sum` | exit 0 — без churn | ✓ PASS |
| Скан уязвимостей на текущем графе | `go tool govulncheck ./...` | exit 0 (required-but-not-called vulns от grpc-графа; called — ноль) | ✓ PASS |
| Depguard RED — net в internal/app (не-тест, `package clock`) | probe + lint run | `import 'net' is not allowed from list 'sup-kv-app-no-io'` (exit 1) | ✓ PASS (red как требуется) |
| Depguard RED — третья сторона в internal/port (не-тест) | probe + lint run | `not allowed from list 'sup-kv-port-stdlib-only'` (exit 1) | ✓ PASS (red как требуется) |
| GREEN после удаления проб | scoped `golangci-lint run` + `git status internal/` | exit 0; дерево чистое | ✓ PASS |
| Вендоренный config pristine против upstream | curl upstream 2.15.0 + diff | единственный hunk 64a65,88 (чистая вставка), ноль `<` строк | ✓ PASS |
| Матрица зелёная на последнем merged head | `gh api commits/617951e/check-runs` | lint + 4 gate-ячейки, все success | ✓ PASS |
| CI на PR фазы 4/5 | `gh run list --workflow=ci.yml` | последний прогон PR #31 success (42514571899); два ранних — failure (гейт честно красный до фиксов; master не тронут) | ✓ PASS |
| Scheduled vulncheck стреляет зелёным | `gh run list --workflow=vulncheck.yml` | run 42644047095: schedule, success, 2026-11-09T15:22:42Z (свежий недельный cron) | ✓ PASS |
| Инвентарь Dependabot PR | `gh pr list --author app/dependabot --state all` | 0 — нечего обновлять (см. override 3) | ✓ PASS (состояние как записано) |
| golangci-lint upstream latest | `gh api repos/golangci/golangci-lint/releases/latest` | v2.15.0 == пин (переподтверждено 2026-11-11) | ✓ PASS |
| Состояние rulesets endpoint | `gh api repos/Djarvur/sup-kv/rulesets` | 403 plan-gated — предикат override жив | ✓ PASS (состояние как записано) |
| delete_branch_on_merge | `gh api repos/Djarvur/sup-kv` | true (master, private true) | ✓ PASS |
| Контексты ruleset == имена check-run | python3-сравнение против 617951e | 5/5 на именах ci.yml; strict policy true; 4 типа правил | ✓ PASS |
| Состояние windows ledger | `gsd-tools windows status` | фаза 1: item 1 fixed, item 2 waived; плюс новый open id=3 phase=05 (вне контракта) | ✓ PASS (фазовая часть) |
| Незапушенные master-коммиты docs-only | per-commit diff-tree scan origin/master..master | 8/8 — только .planning, ноль файлов реализации | ✓ PASS |
| Запрет nolint (D-12) | grep `//nolint` по cmd/ + internal/ | 16/16 называют линтер и несут письменное обоснование (было 13 + 3 добавки фазы 5: main.go:48, udprepl/repl.go:611, grpcapi/wait.go:100,158) | ✓ PASS |
| Новый buf-гейт (sanity) | `go tool buf lint` + `go tool buf format --diff --exit-code` | оба exit 0 локально | ✓ PASS |

### Probe Execution

`scripts/*/tests/probe-*.sh` не существует (нет каталога scripts). Пробы фазы — инлайн lint-прогоны,
исполненные верификатором напрямую (Behavioral Spot-Checks, строки 5-7), перезапущенные на текущем
дереве, потому что internal/ и go.mod изменились с прошлого прогона (OnChange-tap и grpc/protobuf-граф).
Первый дубль пробы internal/app (probe в `_test.go`) не сработал — граница documented-освобождает
тестовые файлы (`'!$test'`, находка IN-04 disposition); проба перезапущена не-тестовым файлом с правильным
именем пакета и сработала как требуется. Claims SUMMARY о прогонах воспроизведены first-hand через
чтение файлов, live gh-запросы и tooling-запросы, не приняты на веру.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| QUAL-01 | 01-01 | Strict golangci-lint config + go-ultimate conventions над каждым пакетом; красный гейт блокирует merge | ✓ SATISFIED | `make lint` зелён на полном текущем дереве (standing config verify в Makefile и CI); вендоренный config байт-неизменен (LIVE pristine-дифф); boundary red/green перепроованы по обоим спискам; все 16 `//nolint` по проекту называют линтер и обоснованы (D-12, включая 4 добавки фазы 5); merge-blocking половина — под ратифицированным override (скоуп CI-03) |
| CI-01 | 01-01, 01-02 | Actions гоняет гейты на каждый push и PR, Linux/macOS, amd64/arm64 | ✓ SATISFIED | push+PR триггеры, 4 ячейки перечитаны; 5/5 зелёных live на 617951e; последний CI-прогон PR #31 success; standing config-verify в lint job; go-version-file поглотил go.mod-изменения buf/grpc |
| CI-02 | 01-03, 01-05, 01-06 | Dependabot актуален (updates и alerts) + scheduled govulncheck зелён | ✓ SATISFIED-BY-RATIFICATION (Partial) | Updates настроены и живы (lockstep-ignore цел, config git-неизменен); scheduled govulncheck зелён на свежем cron-возникновении (run 42644047095, 2026-11-09) + локальный скан нового графа exit 0; alerts-половина деферрована под override № 2; REQUIREMENTS честно Partial/unchecked; first-PR backstop решён override № 3 (предикаты перепроверены live) |
| CI-03 | 01-04, 01-05, 01-06 | PR-only development — branch protection на master, прямые push запрещены | ✓ SATISFIED-BY-RATIFICATION (Partial) | PR-only практикуется и проверено на всей истории (origin/master не двигался; ноль не-PR коммитов касаются файлов реализации); работа фаз 4-5 корректно идёт через открытый PR #31 (дважды красный гейт до фиксов); 8 незапушенных master-коммитов — docs-only; серверная защита деферрована под override № 1 (403-предикат перепроверен live); payload + one-command re-apply сохранены дословно; DEFERRED-маркер + ledger waiver + decision log на месте |

Осиротевших requirements нет — REQUIREMENTS.md мапит ровно CI-01, CI-02, CI-03, QUAL-01 на Phase 1
(строки 83-85, 139-142); все четыре фигурируют во frontmatter plans requirements (01-01: QUAL-01, CI-01;
01-02: CI-01; 01-03: CI-02; 01-04: CI-03; 01-05/01-06: CI-02, CI-03).

### Anti-Patterns Found

Debt-marker гейт по всем изменившимся покрытым файлам (.claude/CLAUDE.md, ci.yml, Makefile,
cmd/sup-kv/main.go, go.mod, go.sum, storage/store.go, storage/store_test.go): ноль
TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER. Стаб-паттерны (placeholder/not yet implemented/empty returns) в
изменённых файлах: ноль. Вендоренный FIXME .golangci.yml:155 остаётся upstream-контентом powerman
(запрет байт-неизменности сильнее).

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| .github/workflows/ci.yml | 19-20, 53-55 | WR-01 (ревью 2026-11-11, disposition open): нет timeout-minutes — 6-часовой дефолт GitHub × fail-fast:false | ℹ️ Info | Advisory № 1; не нарушает must_have фазы; фикс — timeout-minutes: 15/20 |
| cmd/sup-kv/main.go | 377-378, 397-428 | IN-01 (ревью 2026-11-11, disposition open): defer stop() отпускает сигналы до конца shutdown (код фазы 5) | ℹ️ Info | Advisory № 2; экспозиция ограничена; относится к фазе 5 |
| .planning/state.json | — | untracked planning-tooling state | ℹ️ Info | Не покрытый вход; коммитить или игнорить сознательно |
| .planning/phases/04-.../04-REVIEW-FIX.md | — | untracked рабочий файл фазы 4 | ℹ️ Info | Книжчество фазы 4, вне покрытого множества |
| .planning/STATE.md | — | незакоммиченная правка phase-pointer (01 executing) верификационного workflow | ℹ️ Info | Planning-книжчество; decision log (G8) цел |
| WINDOWS.md ledger | id=3 | open-пункт phase=05 «Watch/SnapshotCreate/SnapshotLoad Unimplemented until plans 05-04/05-05» — 05-04/05-05 уже завершены по git-истории | ℹ️ Info | Книжчество фазы 5 вне контракта фазы 1; возможно stale — закрывается верификацией фазы 5 |
| ci.yml (buf-шаги) | 40-48 | новые buf-гейты ещё не исполнялись в CI (коммит 9727c80 не запушен — ветка впереди origin на 38 коммитов) | ℹ️ Info | Локально зелёные (buf lint + format exit 0); исполнятся при следующем push в PR #31 |
| govulncheck | — | граф теперь содержит required-but-not-called уязвимости из grpc-зависимостей фазы 5 (прошлый прогон: «No vulnerabilities found») | ℹ️ Info | Семантика гейта сохранена (exit 0, called = fail); tracked Govulncheck-механикой, не suppressions |
| .golangci.yml | 155 | upstream FIXME (вендор, запрет правки) | ℹ️ Info | Переносится; трогать нельзя |
| .planning/ROADMAP.md | 63+ | `**Plans**: TBD` незапланированных фаз | ℹ️ Info | Планировочные placeholder'ы, называющие свою фазу |
| AGENTS.md | 5 | IN-01 (старое): цитирует `go 1.27`, go.mod пинует 1.27.1 | ℹ️ Info | Открыто; безобидный само-ссылочный указатель |
| ci.yml / vulncheck.yml | — | IN-03: actions запинены mutable major-тегами | ℹ️ Info | Открытый hardening-совет (ограничено contents:read, нет секретов) |
| .golangci.yml | '!$test' | IN-04: граница освобождает тестовые файлы, x/sys не покрыт | ℹ️ Info | Открыто; задокументировано в самом конфиге; продемонстрировано первым дублем пробы в этом прогоне |
| internal/app/internal/testutil | — | IN-01 (review): doc-комментарий NewRNG преувеличивает printed-seed контракт | ℹ️ Info | Открытый review Info на коде фазы 2; tracked в disposition ledger |
| docs/setup/branch-protection.md | — | IN-05: «Apply command above» направляет не туда | ℹ️ Info | Открытый review Info; Apply-команда в доке ровно одна |
| ci.yml lint job | — | IN-06: schema-гейт и lint run — два отдельно запиненных бинарника | ℹ️ Info | Паритет версий держится (оба пина v2.15.0 перепроверены); WR-01-lockstep ignore сторожит дрейф |

### Test Quality Audit

Requirement-linked тест-файлы фазы (internal/app/clock/{clock_test,hlc_prop_test}.go,
internal/app/storage/{store_test,merge_test}.go, internal/app/version/version_test.go) активны, зелёные
под -race в этом прогоне; в покрытом множестве ноль skip'ов. storage/store_test.go прирос OnChange-тестами
фазы 5 (+262 строки) — сьют storage ok 1.582s под -race. PR #29 регрессионные тесты
(TestClock_watermarkClampedIntoDomain, TestClock_driftSignalSaturates) присутствуют и зелёные. Циклической
генерации нет (testutil — независимые PCG-конструкторы), ассерты на уровне значений/свойств.
Gate-teeth-доказательства — lint-probe red/green, исполнены напрямую в этом прогоне.

### Decision Coverage

Тул `check.decision-coverage-verify` в этом прогоне: CONTEXT.md отсутствует — нечего трекать (skipped,
blocking false). Существенная сторона перепроверена напрямую: D-03 (dependency policy AGENTS.md + ровно
санкционированная пара прямых require), D-08 (branch-protection.md audit trail + ruleset.json payload +
D-08-гард buf-breaking в ci.yml/Makefile), D-10/D-11 (.golangci.yml две аддитивные правила, live pristine
дифф), D-12 (16/16 nolint названы и обоснованы), D-16/D-17/D-18/D-19 — решения фаз 5, их артефакты
(main.go config/wiring, Makefile ldflags) вне контракта фазы 1 и покрыты ревью 2026-11-11.

## Human Verification Required

Не осталось. Инфраструктурная/фундационная фаза — пользовательских UAT-шагов нет, behavior-unverified
истин нет (все behavior-dependent истины — 6, 7, 10, 11 — несут behavioral evidence этого прогона:
красные пробы исполнены, гейты перезапущены, live check-runs наблюдены). Prior open human item (первый
Dependabot PR) закрыт recorded maintainer-решением 2026-11-02 (01-UAT.md Test 1, UAT complete 3/3;
WINDOWS item 1 fixed). Judgment-tier prohibitions несут recorded maintainer concurrence (01-UAT.md Test
2), и независимый скан этого верификатора не нашёл нарушенных: вендоренный config байт-неизменен (live
pristine дифф), каждый nolint назван и обоснован (16/16), gate-theater-механизма нет (standing config
verify зелён в make lint), локальная/CI версии линтера идентичны v2.15.0, единственные runtime
зависимости графа — санкционированная пара, suppressions в vulncheck.yml нет (файл git-неизменен),
path-based exemptions в ruleset.json нет (файл git-неизменен).

## Gaps Summary

Гэпов нет. Триггер регенерации — вновь цель фазы, работающая как задумано: более поздние фазы изменили
покрытые build-входы (buf-миграция тулинга; grpc-зависимость и полный main.go фазы 5; OnChange-tap в
storage) — и гейты поглотили изменения без ослабления: lint зелён на полном дереве со стоящим config
verify, обе depguard-красные пробы перезапущены на новом графе (protobuf/grpc демонстративно не могут
пройти в internal/app), `go mod tidy` идемпотентен без churn, race-тесты зелёны по всем десяти
тест-пакетам, локальный govulncheck нового графа exit 0, свежий недельный vulncheck cron зелён. Каждый
ранее верифицированный артефакт, который ещё существует, байт-неизменен с прошлого прогона, кроме
восьми перепроверенных изменившихся входов (маркеры фазы 1 целы под правками фаз 4-5). Три
ратифицированных override переносятся дословно с перепроверенными live предикатами (403 rulesets,
private repo, ноль Dependabot PR, upstream-latest v2.15.0, scheduled vulncheck зелён). Фаза закрывается
на 25/25 без открытых пунктов ledger по фазе 1 (открытый id=3 — phase 05), UAT complete 3/3;
реализация фаз 4-5 корректно ждёт своего merge через тот же гейт, который сама и бутстрапила.

---

_Verified: 2026-11-11T11:13:23Z_
_Verifier: Claude (gsd-verifier)_
