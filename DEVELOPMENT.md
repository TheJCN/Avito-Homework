# Разработка второй лабораторной

Рабочая ветка: `lr2`. Все команды выполняются из корня репозитория
`Avito-Homework`.

## Архитектура

Код разделён на простые слои:

- `internal/domain/trip` — модель поездки, статусы и бизнес-правила;
- `internal/usecase/trip` — сценарии создания, получения и завершения поездки;
- `internal/http` — HTTP-обработчики, маршрутизация и middleware;
- `internal/repository/trip` — работа с PostgreSQL;
- `internal/infra` — транзакции и OpenTelemetry;
- `cmd/trip-service` — сборка зависимостей и запуск приложения.

Интерфейсы объявлены рядом с кодом, который их использует. Конкретные
реализации соединяются только в `cmd/trip-service/main.go`.

## Генерация и проверки

Нужна версия Go из `go.mod` или новее. Дополнительная установка генераторов не
требуется: они закреплены в зависимостях проекта.

```sh
make generate
make build
make lint
make test
```

`make generate` обновляет `api/api.gen.go` из OpenAPI-контракта и запускает все
директивы `go:generate`, включая генерацию моков `gomock`. Сгенерированные
файлы вручную не редактируются.

`make test` запускает unit-тесты с race detector и покрытием. Интеграционные
тесты PostgreSQL запускаются отдельно:

```sh
make test-integration
```

## Окружение и запуск

Нужны Docker Desktop и `tripgoctl`, доступный через `PATH`.

```sh
make environment
make migrate
make run
```

`make environment` запускает PostgreSQL и observability-стек и создаёт `.env`.
`make migrate` применяет миграции goose. `make run` собирает и запускает сервис.

Проверки доступности:

- `GET /health` — процесс сервиса работает;
- `GET /ready` — сервис может обратиться к PostgreSQL.

Для остановки окружения с сохранением данных используется команда:

```sh
tripgoctl environment stop
```

## Конфигурация

Основные переменные находятся в `.env.example`:

- `HTTP_ADDR`, `LOG_LEVEL` — HTTP-адрес и уровень логирования;
- `HTTP_*_TIMEOUT`, `SHUTDOWN_TIMEOUT` — таймауты сервера;
- `DATABASE_URL` — адрес PostgreSQL;
- `DATABASE_MAX_CONNS`, `DATABASE_MIN_CONNS` — размер пула;
- `DATABASE_CONNECT_TIMEOUT`, `DATABASE_QUERY_TIMEOUT` — таймауты БД;
- `DATABASE_MAX_CONN_LIFETIME` — время жизни соединения;
- `OTEL_SERVICE_NAME` — имя сервиса в телеметрии;
- `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_OTLP_PROTOCOL` — адрес и
  протокол OTLP;
- `OTEL_TRACES_SAMPLER` — режим семплирования трейсов.

Секреты и локальный `.env` в репозиторий не добавляются.

## Observability

Сервис экспортирует:

- HTTP duration и active requests;
- бизнес-счётчики созданных и завершённых поездок;
- HTTP- и PostgreSQL-трейсы;
- JSON-логи с полем `service` и корреляцией через `trace_id` и `span_id`.

Метрики смотрятся в Grafana, трейсы — в Jaeger. Для проверки нужно выполнить
несколько HTTP-запросов, открыть соответствующий trace в Jaeger и убедиться,
что его идентификатор совпадает с `trace_id` в логах.

## Миграции

```sh
make migrate
make migrate-down
```

Создание поездки и запись истории статуса выполняются в одной транзакции.
Ошибка или panic откатывают транзакцию. Идемпотентность создания хранится в
PostgreSQL и сохраняет поведение повторного запроса с тем же ключом и телом.
