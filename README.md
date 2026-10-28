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

## Первая лабораторная

Окружение и генерация API запускаются из корня репозитория.
Инструкция: [DEVELOPMENT.md](DEVELOPMENT.md).

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

Для запуска второй лабораторной работы используйте `make environment`,
`make migrate` и `make run`. Проверки: `make lint`, `make test` и
`make test-integration`.
