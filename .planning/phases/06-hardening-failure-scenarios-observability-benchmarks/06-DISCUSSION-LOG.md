# Phase 6: Hardening — Failure Scenarios, Observability, Benchmarks - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-11-14 — 2026-11-15
**Phase:** 6-Hardening — Failure Scenarios, Observability, Benchmarks
**Areas discussed:** Метрики, Логи+журнал конфликтов, Реал-сокетные тесты в CI, Бенчмарки+находки ревью

---

## Метрики

### Как включается metrics-эндпоинт?

| Option | Description | Selected |
|--------|-------------|----------|
| Вкл по умолчанию | Флаг --metrics-addr с дефолтом 127.0.0.1:9090 | |
| Opt-in без дефолта | Флаг без дефолта: не задан — эндпоинта нет | ✓ |
| Без HTTP-порта | Счётчики через gRPC-RPC на существующем порту | |

**User's choice:** Opt-in без дефолта + предусмотреть конфигурацию mTLS для этого endpoint
**Notes:** free-text дополнение про mTLS задало второй вопрос области.

### Как устроен mTLS?

| Option | Description | Selected |
|--------|-------------|----------|
| mTLS обязателен | Вкл ⇒ всегда mTLS, иначе отказ запуска | |
| mTLS опционален | Заданы CERT/KEY/CLIENT_CA ⇒ mTLS; только addr ⇒ plaintext | ✓ |
| Только серверный TLS | Сертификат сервера без проверки клиента | |

**User's choice:** mTLS опционален

### Состав семейств метрик?

| Option | Description | Selected |
|--------|-------------|----------|
| Минимум + Go-runtime | OBS-список + go_*/process_* | |
| Только минимум | Строго OBS-01/OBS-02 | |
| Минимум + runtime + топология | + peer count, датаграммы по типам сообщений | ✓ |

**User's choice:** Минимум + runtime + топология

### Держать ли /debug/vars (expvar) рядом с /metrics?

| Option | Description | Selected |
|--------|-------------|----------|
| Оба | Мониторинг + живой дебаг, один слушатель и один mTLS | ✓ |
| Только /metrics | Без expvar | |

**User's choice:** «оба и профайлер» — оба + /debug/pprof/* на том же слушателе

---

## Логи+журнал конфликтов

### Формат структурных логов?

| Option | Description | Selected |
|--------|-------------|----------|
| slog JSON | JSON по умолчанию, текст флагом — stdlib | ✓ |
| slog Text | key=value по умолчанию, JSON флагом | |
| std log | Плоские строки — риск не закрыть OBS-03 | |

**User's choice:** slog JSON

### Куда пишет лог?

| Option | Description | Selected |
|--------|-------------|----------|
| stdout | Канонично для journald | |
| stderr | Сохранить нынешнее поведение | ✓ |
| Флаг выбора | SUPKV_LOG_OUTPUT | |

**User's choice:** stderr

### Куда пишется журнал конфликтов?

| Option | Description | Selected |
|--------|-------------|----------|
| Лог | slog-события с key/winner/loser | |
| Файл-журнал | Append-only рядом с дампом | |
| Оба | Событие + строка в файл | |

**User's choice:** сначала встречный вопрос «зачем нужен журнал конфликтов?» — после объяснения: «это чушь. если запись была перезаписана — она была перезаписана, это нормальная работа, а не событие, требующее разбора»
**Notes:** журнал конфликтов ОТМЕНЁН как концепт; merge-loser молчалив (последовательно с фазой-5 D-06); остаётся счётчик merge conflicts (OBS-01); формулировка TEST-04 «conflict-journal entries» меняется на «детерминированный победитель + счётчик» при планировании.

### Управление уровнем логирования?

| Option | Description | Selected |
|--------|-------------|----------|
| Флаг уровня | SUPKV_LOG_LEVEL: debug|info|warn|error, дефолт info | ✓ |
| Фиксированный info | Без конфигурации | |

**User's choice:** Флаг уровня

---

## Реал-сокетные тесты в CI

### Что считать two-process прогоном?

| Option | Description | Selected |
|--------|-------------|----------|
| exec бинаря | Все real-socket сценарии через os/exec | |
| Смешанно | Масса in-process; exec для рестарт-сценариев | ✓ |
| In-process | Не закрывает букву TEST-04 | |

**User's choice:** Смешанно
**Notes:** вопрос задавался дважды: первый AskUserQuestion был прерван («не вижу сессию», затем проверка ввода «asd user»), повторён успешно.

### Состав exec-сценариев?

| Option | Description | Selected |
|--------|-------------|----------|
| 3 базовых | kill -9 рестарт, both-nodes-down, SIGTERM shutdown | |
| 4: + пустой диск | Те же 3 + запуск с пустым dump-каталогом против живого | ✓ |
| Расширенный | 4 + несовпадение версий протокола (не рекомендовано) | |

**User's choice:** 4: + пустой диск

### Поведение на macOS?

| Option | Description | Selected |
|--------|-------------|----------|
| Skip на darwin | Громкий skip с причиной (прецедент smoke) | ✓ |
| Только Linux | Build-тег/env-гейт | |

**User's choice:** Skip на darwin

---

## Бенчмарки+находки ревью

### Форма бенчмарков и место в CI?

| Option | Description | Selected |
|--------|-------------|----------|
| Bench без CI | make bench руками, без порогов | ✓ |
| CI с порогами | Отдельная джоба, риск ложных срабатываний | |
| Bench + профили | cpuprofile/memprofile артефактами | |

**User's choice:** Bench без CI

### Состав бенчмарков?

| Option | Description | Selected |
|--------|-------------|----------|
| Три по букве | Дамп 10k, burst-репликация, задержка подтверждения | ✓ |
| + мердж | BenchmarkMerge на 10k/100k | |
| Минимальные два | Без отдельного bench на задержку | |

**User's choice:** Три по букве

### Влить ли открытые находки ревью в фазу 6?

| Option | Description | Selected |
|--------|-------------|----------|
| Влить Warning | 4 Warning-фикса; info в триаже | |
| Влить всё | И Warning (4), и все info (8) | ✓ |
| Не влить | Строго OBS/TEST | |

**User's choice:** Влить всё
**Notes:** 12 находок из 01..04-REVIEW-DISPOSITION.md; связка P3-IN-02/P4-IN-02 (лишний дамп при no-op) объединяется планировщиком.

---

## Вне обсуждения (по ходу сессии, отдельные решения maintainer'а)

- «у тебя линтеры в buf отключены! возвращайся и исправляй» → проверка показала: 36 правил STANDARD активны, но 4 находки на 3 правилах погашены ignore_only. Выбран «Полный фикс»: перенос прото в package-каталоги, SupKV→SupKVService, WatchEvent→WatchResponse, снятие всех ваиверов (7b55ad1). Следствие: отпечатки верификации фаз 1/4/5 снова устарели — перегенерация влита в фазу 6 (D-16).

## Claude's Discretion

- Точные имена метрик/лейблов и help-строки
- Устройство slog-хендлера и маппинг крана Log на атрибуты
- Внутренняя структура пакета экспортера и exec-обвязки (прецеденты адаптеров фаз 4/5)
- Какие in-process socket-сценарии получают выделенные тесты против параметризованных
- macOS F_FULLFSYNC — исследовательский пункт (открытый блокер из STATE)

## Deferred Ideas

- CI-джоба с порогами бенчмарков / хранение профилей — отклонено (шум shared runner'ов)
- HMAC аутентификация репликации — вне v1 (docs/IDEA.md)
- Журнал конфликтов в любой форме — ОТМЕНЁН (не просто отложен)
