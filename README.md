# Разработка микросервисов на Go

Материалы курса: лабораторные работы и лекции.

| Что | Где |
|---|---|
| Лабораторные работы, документация, контракты | [`homework/`](homework/) |
| Слайды и записи занятий | [`lectures-and-practice/`](lectures-and-practice/) |

## С чего начать

1. [`homework/README.md`](homework/README.md) — какой сервис строим, дорожная
   карта по работам, стек и правила игры.
2. [`homework/docs/getting-started.md`](homework/docs/getting-started.md) — что
   сделать до первой работы: завести репозиторий, поднять окружение, поставить
   инструменты.
3. [`course-go-autumn-2026/template`](https://github.com/course-go-autumn-2026/template)
   — форкаете себе, там и работаете.

Все пять работ делаются в одном репозитории, каждая следующая продолжает
предыдущую.

## Что где искать дальше

| Вопрос | Файл |
|---|---|
| Как оценивают, как проходит защита, что со сроками | [`homework/docs/grading.md`](homework/docs/grading.md) |
| Что за домен, откуда взялись поездки и координаты | [`homework/docs/domain.md`](homework/docs/domain.md) |
| Какие требования действуют во всех работах | [`homework/docs/conventions.md`](homework/docs/conventions.md) |
| Как поднять окружение и откуда брать адреса | [`homework/docs/environment.md`](homework/docs/environment.md) |
| Какие ручки, события и таблицы зафиксированы | [`homework/contracts/`](homework/contracts/) |

## Вторая лабораторная

Ветка `lr2` содержит сервис поездок с разделением на слои, тестами и
OpenTelemetry. Окружение, генерация кода и все проверки запускаются из корня
репозитория. Подробная инструкция: [DEVELOPMENT.md](DEVELOPMENT.md).

## Архитектура сервиса

Сервис разделён на слои: `internal/domain` содержит правила поездки и ошибки,
`internal/usecase` выполняет сценарии создания, получения и завершения поездки,
`internal/http` принимает HTTP-запросы, а `internal/repository` работает с
PostgreSQL. `internal/infra` содержит технические части: транзакции и
телеметрию.

Зависимости направлены внутрь: HTTP и репозиторий используют usecase, usecase
использует домен, а домен не знает про транспорт, базу данных и OpenTelemetry.
Интерфейсы лежат у потребителя зависимости: так usecase сам описывает, какие
методы репозитория и транзакции ему нужны. Все конкретные реализации собираются
в `cmd/trip-service/main.go`.

Основные команды:

```sh
make generate
make environment
make migrate
make run
make lint
make test
make test-integration
```

HTTP-метрики, трейсы и структурированные логи экспортируются через
OpenTelemetry. В окружение входят PostgreSQL и observability-стек с Grafana и
Jaeger. Логи содержат `service`, `trace_id` и `span_id`, когда запрос связан с
активным span.

## Опрос устаревших поездок

Маршрут поездки сохраняется через `POST /api/v1/trips/{tripId}/positions` и
читается через `GET /api/v1/trips/{tripId}/positions`. Фоновый опрос устроен
так:

```text
PostgreSQL → сканер → ограниченная очередь → worker pool → Push Service
```

Сканер раз в `POSITION_CHECK_INTERVAL` выбирает активные поездки, по которым не
было координат дольше `POSITION_STALE_AFTER`. Очередь вмещает
`POSITION_QUEUE_SIZE` задач, одновременно Push Service вызывают
`POSITION_WORKERS` воркеров. Каждый вызов ограничен `PUSH_REQUEST_TIMEOUT`.
Адрес сервиса задаётся через `PUSH_HTTP_URL`.

Постановка неблокирующая. Если поездка уже ожидает или обрабатывается, повторная
задача не создаётся. Если очередь заполнена, задача отбрасывается, пишется
предупреждение и увеличивается метрика с результатом `dropped`; следующий тик
снова обнаружит поездку. При получении координаты ожидающая задача отменяется.

При остановке сервис сначала завершает входящие HTTP-запросы, затем останавливает
сканер и дорабатывает очередь в пределах `SHUTDOWN_TIMEOUT`. После этого
закрываются соединения Push Service, PostgreSQL и экспорт телеметрии.

Метрики пула имеют префиксы `tripgo.stale_checks`,
`tripgo.position_requests` и `tripgo.positions.saved`. В Jaeger вызов Push
Service отображается отдельным HTTP-спаном, а обработка задачи связана со спаном
сканирования через span link.
