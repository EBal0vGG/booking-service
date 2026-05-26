# Booking service

Сервис бронирования переговорных комнат: расписание по дням недели, **материализованные слоты** фиксированной длины (30 минут), бронирование с защитой от гонок на уровне БД.

## Быстрый старт

```bash
# Поднять Postgres и Redis
# Postgres доступен на localhost:5432, Redis используется внутри docker-compose как redis:6379
docker compose up -d postgres redis

# Применить миграции
make migrate-up

# (Опционально) тестовые пользователи, комнаты и расписания — см. раздел «Сид данных»
make seed

# Запуск API
make run
# или
go run ./cmd/app
```

Переменные окружения: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASS`, `DB_NAME`, `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB`, `REDIS_CHANNEL`, `APP_ENV`, `LOG_LEVEL`, `PORT`, `JWT_SECRET`, `RUN_MIGRATIONS`, `MIGRATIONS_PATH`, `SLOT_GENERATE_INTERVAL_SECONDS`, `CORS_ALLOWED_ORIGINS`.

## Frontend MVP

Frontend расположен в `frontend` (Next.js + TypeScript, App Router, Tailwind, dark UI) и предназначен для ручного прогона полного сценария.

### Запуск frontend локально

```bash
cd frontend
cp .env.example .env.local
npm install
npm run dev
```

UI доступен на `http://localhost:3000`.

Переменные:

- `NEXT_PUBLIC_API_URL` (по умолчанию `http://localhost:8080`)
- `NEXT_PUBLIC_WS_URL` (по умолчанию `ws://localhost:8080/ws`)

### Запуск frontend через docker compose

```bash
docker compose up --build frontend
```

Сервис `frontend` проксируется на `localhost:3000` и зависит от `app`.

### Страницы frontend

- `/login` — логин по email/password + быстрый вход через `dummyLogin`.
- `/register` — регистрация пользователя.
- `/rooms` — список комнат и (для admin) форма создания комнаты.
- `/rooms/{roomId}` — создание schedule (admin), timeline всех slots со статусами (UTC), booking/waitlist (user).
- `/bookings` — мои брони (user), отмена брони.
- `/reservations` — активные резервы пользователя, countdown, confirm/cancel.
- `/admin/bookings` — список всех броней с пагинацией (admin).

### Минимальный тестовый flow через UI

1. Войти как `admin` через `dummyLogin`.
2. Создать room на `/rooms`.
3. Создать schedule на `/rooms/{roomId}`.
4. Войти как `user`.
5. Открыть room и создать booking.
6. Вторым user сделать `Join waitlist` на booked-слоте в timeline комнаты.
7. Первым user отменить booking.
8. Второй user получает WS-уведомление `waitlist_slot_reserved`.
9. Подтвердить резерв через modal (`confirm`) или отменить (`cancel`).
10. Убедиться, что список slots обновляется по room events (`slot_booked`, `slot_reserved`, `slot_available`, `slot_released`, `slot_reservation_expired`).

## Аутентификация

- `POST /register` — создаёт пользователя (`email`, `password`, `role`) и возвращает `user` без пароля.
- `POST /login` — проверяет `email/password` и возвращает JWT access token.
- `POST /dummyLogin` — тестовый endpoint, оставлен для обратной совместимости.

Пароли хранятся как `bcrypt`-хеш в `users.password_hash` (добавлено в базовой схеме миграции `0001_init`).
JWT содержит `user_id`, `role`, `exp`; middleware валидирует одинаково токены из `/login` и `/dummyLogin`.

## Realtime (WebSocket)

- `GET /ws` — WebSocket endpoint.
- Аутентификация: `Authorization: Bearer <jwt>`; для браузерного WS-клиента поддержан fallback `?token=<jwt>`.
- После подключения клиент отправляет:

```json
{"type":"subscribe","roomId":"<uuid>"}
```

- Сервер подтверждает подписку:

```json
{"type":"subscribed","roomId":"<uuid>"}
```

- После успешного `POST /bookings/create` рассылается:

```json
{"type":"slot_booked","roomId":"...","slotId":"...","bookingId":"...","timestamp":"..."}
```

- После успешного `POST /bookings/{bookingId}/cancel` рассылается:

```json
{"type":"slot_released","roomId":"...","slotId":"...","bookingId":"...","timestamp":"..."}
```

- Если активный reservation отменён/истёк и replacement reservation не создан, рассылается:

```json
{"type":"slot_available","roomId":"...","slotId":"...","timestamp":"..."}
```

- Если при cancel создан временный резерв для первого waitlist-пользователя, дополнительно рассылаются:

```json
{"type":"slot_reserved","roomId":"...","slotId":"...","reservationId":"...","timestamp":"..."}
```

и персонально пользователю:

```json
{"type":"waitlist_slot_reserved","roomId":"...","userId":"...","slotId":"...","reservationId":"...","waitlistEntryId":"...","expiresAt":"...","timestamp":"..."}
```

Доставка realtime-событий выполняется через Redis Pub/Sub:

- `booking usecase -> Redis publisher -> channel -> Redis subscriber -> local hub -> ws clients`;
- Redis отвечает за межпроцессную доставку;
- локальный hub отвечает за fan-out по подпискам, backpressure и отключение медленных клиентов.

Семантика доставки: Redis Pub/Sub работает в режиме **at-most-once** (offline subscriber или offline клиент может пропустить событие).  
Публикация событий — **best-effort**: если Redis временно недоступен, HTTP `booking/cancel` не откатываются и не падают только из-за realtime-транспорта.

## Waitlist

- `POST /waitlist/join` — поставить пользователя в очередь ожидания для занятого слота.
- Нельзя встать в waitlist на прошлый слот или на слот, который уже забронирован этим же пользователем.
- `POST /waitlist/{waitlistId}/leave` — выйти из очереди ожидания (идемпотентно для уже отменённой записи).
- Для безопасности `leave` возвращает `WAITLIST_NOT_FOUND` и для несуществующей записи, и для чужой записи (без раскрытия существования чужих ID).
- Очередь хранится в `waitlist_entries` и упорядочивается детерминированно по `position` (`bigserial`), а не по `created_at`.
- При `POST /bookings/{bookingId}/cancel` сервис atomically:
  1) переводит booking `active -> cancelled`,
  2) выбирает первый waitlist entry (`FOR UPDATE SKIP LOCKED`) и переводит `active -> notified`,
  3) создаёт `slot_reservations` со статусом `active` и `expires_at = now + TTL`.
- После V2 жизненный цикл приглашения хранится в `slot_reservations`.
  `waitlist_entries.status = notified` означает, что запись выбрана из очереди, а terminal-состояние (`confirmed`/`expired`/`cancelled`) определяется по reservation.
- Если резерв создан, `slot_released` не публикуется (слот не считается свободным для всех).
- `POST /reservations/{reservationId}/confirm` подтверждает резерв и создаёт бронь.
- `POST /reservations/{reservationId}/cancel` отменяет резерв владельцем.
- `GET /reservations/my/active` возвращает активные резервы пользователя (для восстановления состояния после reload/reconnect).
- Background worker периодически истекает просроченные резервы и двигает очередь дальше.
- `GET /rooms/{roomId}/slots/list` исключает и `active booking`, и `active reservation`.
- `GET /rooms/{roomId}/slots/all` возвращает все слоты комнаты за дату со статусами `available|booked|reserved|past`.
- Конфигурация V2 через env:
  - `RESERVATION_TTL_SECONDS` (по умолчанию `300`);
  - `RESERVATION_EXPIRE_INTERVAL_SECONDS` (по умолчанию `15`);
  - `RESERVATION_EXPIRE_BATCH` (по умолчанию `100`).
- Генерация слотов запускается в фоне с первым immediate-run на старте приложения и далее по интервалу `SLOT_GENERATE_INTERVAL_SECONDS` (по умолчанию `60`).
- CORS для browser-frontend конфигурируется через `CORS_ALLOWED_ORIGINS` (CSV, по умолчанию `http://localhost:3000`).

## Observability

- Структурные логи через `slog` (`APP_ENV=dev|prod`, `LOG_LEVEL=debug|info|warn|error`).
- Request ID middleware: входящий `X-Request-ID` переиспользуется, иначе генерируется UUID; значение возвращается в response header.
- HTTP access logging: `method`, `path` (route pattern), `status`, `duration_ms`, `request_id`, `remote_addr`, `user_id` (когда доступен).
- Prometheus endpoint: `GET /metrics` (без auth).

Ключевые метрики:

- HTTP: `http_requests_total`, `http_request_duration_seconds`
- Booking: `booking_created_total`, `booking_cancelled_total`, `booking_conflicts_total`, `booking_create_errors_total`
- Realtime WS: `ws_connections_current`, `ws_messages_sent_total`, `ws_subscriptions_current`
- Realtime Redis: `redis_realtime_publish_total`, `redis_realtime_events_received_total`, `redis_realtime_subscriber_reconnects_total`
- Waitlist: `waitlist_joined_total`, `waitlist_notifications_total`, `waitlist_cancelled_total`
- Reservations: `reservations_created_total`, `reservations_confirmed_total`, `reservations_expired_total`, `reservations_cancelled_total`

Запуск Prometheus:

```bash
docker compose up -d prometheus
```

Prometheus UI: `http://localhost:9090`  
Target приложения в `deploy/prometheus/prometheus.yml`: `app:8080` с `metrics_path: /metrics`.

### Makefile

- **`make up`** — `docker compose up --build -d` (контейнеры в фоне, терминал не блокируется).
- **`DATABASE_URL`** в `Makefile` задаёт строку подключения для **`make migrate-up` / `migrate-down`** и **`make test-integration`**. Само приложение (`cmd/app`) собирает DSN из **`DB_*`** через `config.DatabaseURL()` и переменную **`DATABASE_URL` не читает**.
- **Миграции** вызывают `docker run ... --network host`, чтобы контейнер `migrate` видел Postgres на `localhost` хоста. Это рассчитано на **Linux**; на **macOS** / **Windows** с Docker Desktop иногда нужно убрать `--network host` и использовать адрес хоста, который видит Docker (или запускать migrate иначе).
- **`make test-cover`** считает покрытие только по **`./internal/...`**, а не по всему репозиторию как `make test` / `go test ./...`.
- **`make seed`** — накатывает `scripts/seed.sql` через `psql` на `DATABASE_URL` (пользователи с UUID как у dummy login, 3 комнаты, расписания). Нужен установленный `psql`.
- **`make lint`** — `golangci-lint run ./...` по `.golangci.yaml`. Бинарник должен быть собран с **Go не ниже версии из `go.mod`**, иначе линтер может отказаться анализировать модуль.

## Соответствие `api.yaml`

Ответ **`POST /rooms/{roomId}/schedule/create`** (201): тело `schedule` в формате спецификации — `roomId`, `daysOfWeek[]`, `startTime`, `endTime` (строки `HH:MM` UTC), без вложенного массива `rules`.

Ошибки в JSON: `{ "error": { "code": "<ENUM>", "message": "..." } }` — как в `components.schemas.ErrorResponse`.

### HTTP-статусы и коды доменных ошибок

| HTTP | Код в `error.code` | Пример ситуации |
|------|-------------------|-----------------|
| 400 | `INVALID_REQUEST` | невалидный JSON, пагинация, прошлый слот |
| 401 | `UNAUTHORIZED` | нет/битый JWT |
| 403 | `FORBIDDEN` | не та роль (не admin / не user) |
| 404 | `ROOM_NOT_FOUND`, `SLOT_NOT_FOUND`, `BOOKING_NOT_FOUND`, `WAITLIST_NOT_FOUND`, `RESERVATION_NOT_FOUND` | сущность не найдена |
| 409 | `SLOT_ALREADY_BOOKED`, `SLOT_RESERVED`, `SLOT_NOT_BOOKED`, `SCHEDULE_EXISTS`, `WAITLIST_ALREADY_JOINED` | конфликт брони/очереди/резерва/расписания |
| 500 | `INTERNAL_ERROR` | прочие ошибки |

Проверка вручную: получить токен `POST /dummyLogin` с телом `{"role":"admin"|"user"}`, затем вызывать эндпоинты с заголовком `Authorization: Bearer <token>`.

## Тесты

```bash
# Unit-тесты (usecase, slot generator)
go test ./...

# Интеграционные тесты (нужен Postgres; при недоступности БД тесты пропускаются)
export TEST_DATABASE_URL=postgres://booking:booking@localhost:5432/booking?sslmode=disable
go test -tags=integration ./internal/integrationtest/...
```

## Нагрузочный прогон

Скрипт `scripts/load_slots_list.sh` дергает `POST /dummyLogin` (роль `user`), затем запускает [**hey**](https://github.com/rakyll/hey) по **`GET /rooms/{roomId}/slots/list?date=YYYY-MM-DD`** с заголовком `Authorization`. Нужны `curl`, `jq`, `hey`.

```bash
export BASE_URL=http://localhost:8080
# Комната по умолчанию из make seed — Seed Room A
export ROOM_ID=aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa
./scripts/load_slots_list.sh
```

Переменные: `REQUESTS` (число запросов, по умолчанию 300), `CONCURRENCY` (параллелизм, по умолчанию 15), `DATE` (UTC-дата для `?date=`). Отчёт — вывод `hey` (QPS, латентность, коды ответов).

## Realtime smoke test

Скрипт `scripts/ws_smoke.go` проверяет e2e-поток realtime:

1. логин через `POST /dummyLogin`;
2. подключение к `ws://localhost:8080/ws` и `subscribe` на комнату;
3. `POST /bookings/create` -> ожидается `slot_booked`;
4. `POST /bookings/{id}/cancel` -> ожидается `slot_released`.

Перед запуском нужен существующий будущий слот (можно создать вручную в БД). Затем:

```bash
go run ./scripts/ws_smoke.go
```

## Почему rolling window в генераторе слотов

Генератор не создаёт слоты «навсегда» вперёд: он поддерживает **скользящее окно** (по умолчанию 14 суток от текущего момента). Это ограничивает рост таблицы `slots`, снижает нагрузку на вставки и соответствует смыслу продукта: пользователю нужны ближайшие даты, а не бесконечная сетка.

Окно считается по **времени начала слота** (`slot.start`): в окно попадают только старты, которые ещё не достигли правой границы `now + rollingWindow`.

## Почему слоты материализованы

Слоты хранятся в таблице `slots` (UTC, `TIMESTAMPTZ`), а не вычисляются на лету из расписания и календаря. Это даёт:

- предсказуемые идентификаторы слотов (стабильный UUID от `room_id` + `start_time`);
- простые JOIN-ы с `bookings` и список свободных слотов одним SQL-запросом;
- индексы по `room_id`, `start_time` и ограничение пересечений (`EXCLUDE` на gist) на уровне БД.

## Почему расписание нельзя изменить

Расписание создаётся один раз и далее считается неизменяемым.  
Это упрощает генерацию слотов и исключает необходимость сложной переработки уже созданных слотов при изменении расписания.

## Почему slot ID детерминированный

UUID слота генерируется из `(room_id, start_time)`.  
Это гарантирует идемпотентность генерации и исключает дублирование слотов.

## Как решена конкуренция при бронировании

1. **Частичный уникальный индекс** `uq_bookings_active_slot` на `(slot_id)` при `status = 'active'`: у комнаты в один момент времени не может быть двух активных броней на один слот.
2. Вставка брони: `INSERT ... ON CONFLICT (slot_id) WHERE (status = 'active') DO NOTHING` (соответствует частичному уникальному индексу `uq_bookings_active_slot`); если строка не вставилась (`RowsAffected == 0`), репозиторий возвращает доменную ошибку `SLOT_ALREADY_BOOKED`.
3. Бизнес-операция бронирования выполняется в **транзакции** (`TxManager.WithinTransaction`): проверка слота и вставка согласованы.

Так устраняется гонка «прочитали свободно → двое вставили»: побеждает одна вставка, вторая получает конфликт.

## Почему partial index на бронированиях

Индекс `idx_bookings_user_active` на `(user_id) WHERE status = 'active'` ускоряет запросы вида «мои активные брони в будущем» (`/bookings/my`): при росте таблицы полный скан по `user_id` без фильтра по статусу хуже масштабируется. Активных броней обычно мало относительно всей истории.

Дополнительно в схеме есть:

- `uq_bookings_active_slot` — частичный уникальный индекс для конкуренции по слоту;
- обычные индексы по `user_id` и `slot_id` для админских списков и FK.

## Как работает генератор слотов

1. Загружаются комнаты, у которых есть хотя бы одна строка в `schedules` (`ListWithSchedule`).
2. Для каждой комнаты читается последний `start_time` в `slots` (если есть); нижняя граница генерации — `max(now, last_start + 30m)`, чтобы не дублировать уже созданные слоты.
3. По каждому дню в диапазоне `[сегодня .. конец rolling window]` и каждому правилу расписания (`day_of_week`, интервал `start_time`–`end_time`) генерируются интервалы длины 30 минут, полностью лежащие внутри окна правила.
4. Слоты вставляются батчами с `ON CONFLICT (room_id, start_time) DO NOTHING` для идемпотентности и параллельных запусков.

День недели в БД: **1 = понедельник … 7 = воскресенье** (ISO 8601).

## Структура проекта

- `cmd/app` — точка входа, конфиг, миграции, HTTP.
- `internal/domain` — сущности и коды ошибок API.
- `internal/usecase` — сценарии (room, schedule, slot, booking, waitlist, reservation, auth).
- `internal/repository/postgres` — реализация репозиториев и `TxManager`.
- `internal/app/slotgen` — генератор слотов.
- `internal/transport/http` — Chi, handlers, middleware JWT, JSON-ответы.
