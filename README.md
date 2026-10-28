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

### Бонусные задания

`POST /api/v1/trips` поддерживает необязательный UUID-заголовок
`Idempotency-Key`. Повтор с тем же телом в течение `IDEMPOTENCY_KEY_TTL`
(по умолчанию 24 часа) возвращает созданную поездку с кодом `200`; другое тело
с тем же ключом возвращает `409 idempotency_conflict`.

Собрать и запустить контейнер рядом с локальным PostgreSQL:

```sh
docker build -f deploy/Dockerfile -t trip-service .
docker run --rm --env-file .env \
  -e DATABASE_URL=postgres://tripgo:tripgo@host.docker.internal:5432/tripgo?sslmode=disable \
  -p 8080:8080 trip-service
```

Финальный образ построен на `scratch` и запускается от непривилегированного
пользователя. На macOS ARM64 он занимает около 11 MB.
