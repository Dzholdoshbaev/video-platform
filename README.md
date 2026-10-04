# Streaming Platform Backend

Учебный production-like backend видеостриминговой платформы на идиоматичном Go. Проект рассчитан на backend-разработчика с опытом Java/Spring: здесь нет web framework, ORM, DI-контейнера или лишних инфраструктурных сервисов. Основной фокус — явные зависимости, маленькие интерфейсы, errors as values, `context.Context`, конкурентность и lifecycle приложения.

## Что умеет проект

- создавать и читать пользователей;
- активировать подписку с authorize/capture платежа и уведомлениями;
- добавлять favorites одной транзакцией вместе с analytics event;
- сохранять и читать историю просмотра;
- проверять moderation status;
- принимать analytics events в worker pool;
- выполнять batch-транскодирование через worker pool;
- загружать рекомендации с request timeout;
- восстанавливаться после panic в HTTP handler;
- корректно завершать HTTP server, workers и PostgreSQL pool.

## Архитектура

```mermaid
flowchart LR
    H[HTTP Handler] --> S[Service]
    S --> R[Repository interface]
    R --> P[PostgreSQL / pgxpool]
```

Handlers отвечают только за transport: JSON, DTO, HTTP status. Services реализуют use cases и зависят от небольших интерфейсов. PostgreSQL implementations находятся за repository interfaces. `cmd/api` — composition root, где конкретные реализации связываются вручную.

Для batch-транскодирования:

```mermaid
flowchart LR
    P[Producer] --> J[Jobs Channel]
    J --> W[Worker Pool]
    W --> R[Results Channel]
    R --> C[Collector]
```

Feeder — единственный sender/owner `jobs`, поэтому именно он закрывает канал. Coordinator ждёт все workers через `WaitGroup` и только после этого закрывает `results`. Collector никогда не закрывает канал, из которого читает. Это исключает `send on closed channel`, deadlock и goroutine leak при cancellation.

## Структура

```text
.
├── cmd/api/main.go                    # composition root, http.Server, shutdown
├── internal/domain/                   # entities и domain errors
├── internal/service/                  # use cases и внешние ports
├── internal/repository/               # repository interfaces
├── internal/repository/postgres/      # pgxpool implementations и transaction
├── internal/http/                     # net/http handlers и error mapping
├── internal/http/middleware/          # request ID, logging, recovery
├── internal/worker/                   # analytics, transcoding, player queue
├── internal/platform/database/        # создание PostgreSQL pool
├── migrations/                        # up/down SQL migration
├── docker-compose.yml                 # только PostgreSQL
└── README.md
```

## Запуск PostgreSQL

Требуются Go 1.24+ и Docker с Compose plugin.

```bash
docker compose up -d
docker compose ps
```

Значение `DATABASE_URL` по умолчанию:

```text
postgres://streaming:streaming@localhost:5432/streaming?sslmode=disable
```

## Миграции

В проект намеренно не добавлен migration framework. Для учебного запуска migration применяется через `psql` внутри контейнера:

```bash
docker compose exec -T postgres psql -U streaming -d streaming < migrations/000001_init.up.sql
```

Откат:

```bash
docker compose exec -T postgres psql -U streaming -d streaming < migrations/000001_init.down.sql
```

## Запуск приложения

```bash
go mod download
go run ./cmd/api
```

Настройки задаются environment variables:

```bash
HTTP_ADDR=:8080
DATABASE_URL=postgres://streaming:streaming@localhost:5432/streaming?sslmode=disable
```

Проверка:

```bash
curl http://localhost:8080/health
```

## API

| Method | Path | Назначение |
|---|---|---|
| `POST` | `/users` | создать профиль |
| `GET` | `/users/{id}` | получить профиль |
| `POST` | `/subscriptions/activate` | активировать подписку |
| `GET` | `/subscriptions/{id}` | получить подписку |
| `POST` | `/favorites` | добавить favorite и analytics event транзакционно |
| `GET` | `/users/{id}/favorites` | favorites пользователя |
| `POST` | `/watch-history` | upsert позиции просмотра |
| `GET` | `/users/{id}/watch-history` | история просмотра |
| `POST` | `/moderation/check` | проверить moderation status |
| `POST` | `/analytics/events` | поставить event в очередь |
| `POST` | `/transcoding/batch` | транскодировать batch |
| `GET` | `/recommendations/{userID}` | рекомендации с timeout 2 секунды |
| `GET` | `/health` | liveness endpoint |

Примеры:

```bash
curl -X POST http://localhost:8080/users \
  -H 'Content-Type: application/json' \
  -d '{"username":"ada","email":"ada@example.com","age":30,"country":"UK"}'
```

```bash
curl -X POST http://localhost:8080/subscriptions/activate \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"USER_ID","plan":"premium","price":12.50,"expiration_date":"2030-01-31"}'
```

```bash
curl -X POST http://localhost:8080/transcoding/batch \
  -H 'Content-Type: application/json' \
  -d '{"jobs":[{"id":"job-1","video_id":"video-1","filename":"video.mp4"}]}'
```

## Error handling

Ошибки проходят цепочку `repository -> service -> handler` и оборачиваются через `%w`, поэтому причина не теряется. HTTP слой использует `errors.Is` для sentinel errors и `errors.As` для typed errors.

| Error | HTTP status |
|---|---:|
| `ErrUserNotFound`, `ErrEpisodeNotFound`, `ErrSubscriptionNotFound` | 404 |
| `*ValidationError` | 400 |
| `ErrConflict` | 409 |
| `*ModerationError` | 422 |
| `context.DeadlineExceeded` | 504 |
| неизвестная ошибка | 500 |

Business errors не вызывают panic. `RecoveryMiddleware` — последняя защита от programmer/runtime failures: он логирует request ID и возвращает JSON `500`, не завершая процесс.

## Transaction design

`FavoriteRepository.AddWithAnalytics` выполняет `Begin`, проверку пользователя, `INSERT favorites`, `INSERT analytics_events` и `Commit`. Сразу после `Begin` регистрируется `defer Rollback`. После успешного `Commit` pgx возвращает `pgx.ErrTxClosed` из `Rollback`; это ожидаемое безопасное состояние и оно игнорируется. На любом раннем `return` rollback остаётся гарантированным.

## Concurrency design

- `VideoTranscodingService` создаёт worker pool на один batch и общий `context.WithTimeout`.
- Все потенциально блокирующие send/receive учитывают `ctx.Done()`.
- `WaitGroup` определяет момент, когда разрешено закрыть `results`.
- `AnalyticsProcessor` — долгоживущий buffered queue с несколькими workers.
- `processedEvents` защищён `Mutex`: без него конкурентные increment/read образуют data race и могут терять обновления.
- Analytics queue останавливается cancellation, а не `close`, чтобы HTTP `Submit` не мог соревноваться с закрытием канала.
- `PlayerEventQueue` отдельно демонстрирует buffered channel, `close` и `range`; закрывает канал sender/owner, не consumer.
- `UserTasteService` выбирает через `select` между результатом источника и `ctx.Done()`. HTTP deadline распространяется вниз по call chain.

## Graceful shutdown

`signal.NotifyContext` обрабатывает `SIGINT/SIGTERM`. Порядок завершения:

1. `http.Server.Shutdown` прекращает приём новых соединений и ждёт текущие handlers.
2. После отсутствия новых producers отменяется context background workers и ожидается их `WaitGroup`.
3. PostgreSQL pool закрывается последним, потому что активные handlers/workers потенциально могли использовать DB.

Shutdown имеет отдельный timeout context: signal context уже отменён и поэтому не подходит для ожидания cleanup.

## Тесты и проверки

```bash
gofmt -w cmd internal
go vet ./...
go test ./...
go test -race ./...
```

Unit tests покрывают `ValidationError`, `errors.Is/As`, moderation typed error и typed-nil pitfall, subscription orchestration, notifications и конкурентный analytics counter. HTTP tests через `httptest` покрывают success, validation, not found и panic recovery.

## Что изучать в каждом package

- `internal/domain`: structs, JSON tags, constructors, pointer/value receivers, zero values, custom/sentinel errors, `Unwrap`.
- `internal/service`: маленькие implicit interfaces, interface composition, runtime polymorphism, error wrapping, `context`, `select`, constructor injection без DI container.
- `internal/repository`: consumer-oriented contracts вместо Java-style base repositories.
- `internal/repository/postgres`: `pgxpool`, передача context, scan, constraint mapping, named return, `defer`, transaction lifecycle.
- `internal/worker`: goroutines, directional/buffered channels, ownership, `WaitGroup`, `Mutex`, cancellation и отсутствие leaks.
- `internal/http`: DTO boundary, JSON decoder, Go 1.22+ `ServeMux`, centralized error mapping, request-scoped context.
- `internal/http/middleware`: function composition, custom `ResponseWriter`, panic/recover и context value.
- `cmd/api`: manual dependency wiring, `http.Server`, signals и graceful shutdown.

Особенно полезно после чтения переписать сначала `domain + service` с in-memory repositories, затем отдельно восстановить worker pools и лишь после этого подключить HTTP/PostgreSQL.
