# Лабораторна робота 6 — Persistence Layer у Node.js та Go

Однакова система керування ліцензіями реалізована на Node.js/Express та Go/net/http. In-memory repository збережено для unit-тестів, а робоче сховище перемикається між PostgreSQL і MongoDB одним параметром `REPO_TYPE`. CRUD, validation, error handling, WebSocket і Streams працюють через однакові шари в обох runtime.

## Архітектура

```text
HTTP → Router → Controller → Service → LicenseRepository interface/contract
                                            ├── PostgreSQL repository
                                            ├── MongoDB repository
                                            └── InMemory repository (tests)

Service → Mongo EventLogger → events collection
        └→ WebSocket Hub → усі підключені клієнти

PostgreSQL/MongoDB → batch reader → Stream/Transform → CSV file
                                      └→ WebSocket export_progress
```

Controller і бізнес-правила не залежать від конкретної БД. Factory вибирає реалізацію repository за `REPO_TYPE`. Node використовує pool розміром 15, Go — 40 з'єднань, що відповідає Event Loop та goroutines.

## Відповідність пунктам лабораторної роботи

Нижче вказані конкретні файли та рядки, які закривають кожний пункт завдання. Предметну область `users/email` з методички адаптовано до проєкту: основна сутність — `license`, а унікальне бізнес-поле — `license key`.

### 1. Інфраструктура: Docker Compose та `.env`

- [`docker-compose.yml:4-20`](docker-compose.yml#L4-L20) — PostgreSQL 16, окремий volume, запуск `init.sql` та healthcheck.
- [`docker-compose.yml:22-38`](docker-compose.yml#L22-L38) — MongoDB 7, окремий volume, запуск `mongo-init.js` та healthcheck.
- [`docker-compose.yml:40-94`](docker-compose.yml#L40-L94) — Node і Go отримують усі параметри підключення тільки зі змінних середовища та стартують після готовності обох БД.
- [`.env.example:1-21`](.env.example#L1-L21) — повний перелік необхідних змінних без реальних значень.
- [`.gitignore:22-25`](.gitignore#L22-L25) — робочий `.env` і логи не потрапляють у Git.

Отже, у коді немає адрес, логінів або паролів БД. `docker compose` бере їх із локального `.env`.

### 2. Схема даних

- [`init.sql:1-11`](init.sql#L1-L11) — таблиця `licenses`; поле `license_key` має PostgreSQL-обмеження `UNIQUE`, обов'язкові поля мають `NOT NULL`, числові поля — `CHECK`.
- [`init.sql:13-25`](init.sql#L13-L25) — таблиця `accounts` і два початкові рахунки для демонстрації транзакції.
- [`mongo-init.js:1-3`](mongo-init.js#L1-L3) — MongoDB створює унікальний індекс для `licenses.key` та індекс за часом для write-heavy колекції `events`.
- [`go-service/internal/model/license.go:3-20`](go-service/internal/model/license.go#L3-L20) — одна структура Go має `json` і `bson` tags для HTTP та MongoDB.
- [`node-service/src/models/license.js:1-13`](node-service/src/models/license.js#L1-L13) — модель тієї самої ліцензії у Node.js.

### 3. Connection Pool

Node.js:

- [`node-service/src/db/postgres.js:3-18`](node-service/src/db/postgres.js#L3-L18) — створення `pg.Pool`; `max` читається з `PG_POOL_MAX`, після чого виконується тестовий `SELECT 1`.
- [`node-service/src/db/mongo.js:3-11`](node-service/src/db/mongo.js#L3-L11) — офіційний `MongoClient`, `maxPoolSize` і `ping`.

Go:

- [`go-service/internal/db/postgres.go:12-27`](go-service/internal/db/postgres.go#L12-L27) — `database/sql` із драйвером `pgx`, `SetMaxOpenConns` і `SetMaxIdleConns`.
- [`go-service/internal/db/mongo.go:14-28`](go-service/internal/db/mongo.go#L14-L28) — офіційний MongoDB driver, `SetMaxPoolSize` і `Ping`.
- [`.env.example:12-21`](.env.example#L12-L21) — окремі змінні для розмірів Node/Go pool; у локальному `.env` для демонстрації встановлено 15 для Node та 40 для Go, але значення не зашиті в код.

### 4. Дві реалізації Repository та повний CRUD

Абстракція Go:

- [`go-service/internal/repository/license.go:9-24`](go-service/internal/repository/license.go#L9-L24) — незмінний CRUD-interface `LicenseRepository` та окремий `LicenseBatchReader` для потокового export.
- [`go-service/internal/repository/factory.go:10-18`](go-service/internal/repository/factory.go#L10-L18) — factory повертає PostgreSQL або MongoDB implementation за `REPO_TYPE`.
- [`go-service/internal/repository/postgres.go:13-134`](go-service/internal/repository/postgres.go#L13-L134) — PostgreSQL CRUD через параметризовані `$1...$7` запити, без ORM.
- [`go-service/internal/repository/mongo.go:14-133`](go-service/internal/repository/mongo.go#L14-L133) — MongoDB CRUD через офіційний driver, BSON, `$set`, cursor та unique-index errors.
- [`go-service/internal/repository/memory.go:10-104`](go-service/internal/repository/memory.go#L10-L104) — InMemory implementation для unit-тестів.

Структурний contract Node.js:

- [`node-service/src/repositories/license-repository.factory.js:4-11`](node-service/src/repositories/license-repository.factory.js#L4-L11) — factory із тим самим перемиканням `postgres | mongo`.
- [`node-service/src/repositories/postgres-license.repository.js:16-89`](node-service/src/repositories/postgres-license.repository.js#L16-L89) — PostgreSQL CRUD через `pg` і параметризовані запити.
- [`node-service/src/repositories/mongo-license.repository.js:9-78`](node-service/src/repositories/mongo-license.repository.js#L9-L78) — MongoDB CRUD, `$set`, cursor та numeric sequence для сумісного identifier.
- [`node-service/src/repositories/license.repository.js:1-49`](node-service/src/repositories/license.repository.js#L1-L49) — InMemory implementation для тестів.

Ізоляція бізнес-рівня:

- [`node-service/src/services/license.service.js:4-83`](node-service/src/services/license.service.js#L4-L83) і [`node-service/src/controllers/license.controller.js:14-39`](node-service/src/controllers/license.controller.js#L14-L39) працюють лише з repository contract і не імпортують `pg` або `mongodb`.
- [`go-service/internal/service/license.go:20-150`](go-service/internal/service/license.go#L20-L150) і [`go-service/internal/controller/license.go:25-189`](go-service/internal/controller/license.go#L25-L189) залежать від Go-interface, а не від конкретної БД.
- [`node-service/src/server.js:33-41`](node-service/src/server.js#L33-L41) і [`go-service/cmd/server/main.go:28-54`](go-service/cmd/server/main.go#L28-L54) є composition root: саме тут створюються connections, repository, Service та Controller.

Конфлікт унікального ключа перехоплюється у repository та перетворюється Service/централізованим error handling у `409 Conflict`; відсутній identifier — у `404 Not Found`.

### 5. PostgreSQL-транзакція переказу

Node.js:

- [`node-service/src/services/account.service.js:13-46`](node-service/src/services/account.service.js#L13-L46) — `pool.connect()` → `BEGIN` → `SELECT ... FOR UPDATE` → перевірка балансу → два `UPDATE` → `COMMIT`.
- У `catch` виконується `ROLLBACK`, а у `finally` завжди виконується `client.release()`.
- [`node-service/src/controllers/account.controller.js:8-26`](node-service/src/controllers/account.controller.js#L8-L26) — validation і HTTP response; [`node-service/src/routes/account.routes.js:5-8`](node-service/src/routes/account.routes.js#L5-L8) — endpoint `POST /api/accounts/transfer`.

Go:

- [`go-service/internal/service/account.go:50-99`](go-service/internal/service/account.go#L50-L99) — `BeginTx`, `defer tx.Rollback()`, блокування рядків `FOR UPDATE`, перевірка, два оновлення й `tx.Commit()`.
- [`go-service/internal/controller/account.go:31-68`](go-service/internal/controller/account.go#L31-L68) — parsing, validation та коди `400/404/409/500`.
- [`go-service/internal/router/router.go:42-43`](go-service/internal/router/router.go#L42-L43) — маршрути перегляду рахунків і переказу.

Успіх повертає `status: committed`. Недостатній баланс дає `409`, після чого smoke-test повторно читає рахунки й доводить, що `ROLLBACK` не змінив баланс.

### 6. Polyglot persistence: CRUD у SQL, події в MongoDB

- [`node-service/src/services/license.service.js:22-75`](node-service/src/services/license.service.js#L22-L75) — після успішних create/update/delete Service створює `license_created`, `license_updated`, `license_deleted`.
- [`node-service/src/events/mongo-event-logger.js:1-18`](node-service/src/events/mongo-event-logger.js#L1-L18) — запис подій і читання останніх подій із MongoDB.
- [`node-service/src/server.js:18-20`](node-service/src/server.js#L18-L20) — одна подія спочатку записується через EventLogger, потім broadcast-иться у WebSocket.
- [`go-service/internal/service/license.go:25-32`](go-service/internal/service/license.go#L25-L32) і [`go-service/internal/service/license.go:54-122`](go-service/internal/service/license.go#L54-L122) — тип події та виклики publisher з CRUD Service.
- [`go-service/internal/events/mongo.go:23-61`](go-service/internal/events/mongo.go#L23-L61) — Go EventLogger для MongoDB.
- [`go-service/cmd/server/main.go:43-53`](go-service/cmd/server/main.go#L43-L53) — композиція Mongo EventLogger і WebSocket Hub.
- [`node-service/src/routes/event.routes.js:5-7`](node-service/src/routes/event.routes.js#L5-L7) та [`go-service/internal/router/router.go:38-40`](go-service/internal/router/router.go#L38-L40) — `GET /api/events`.

Це polyglot persistence: при `REPO_TYPE=postgres` ліцензії зберігаються у PostgreSQL, але журнал аудиту незалежно зберігається у MongoDB.

### 7. Інтеграція WebSocket та Streams із лабораторної №5

WebSocket broadcast:

- [`node-service/src/websocket/websocket-hub.js:12-106`](node-service/src/websocket/websocket-hub.js#L12-L106) — реєстрація clients, broadcast, heartbeat і захист від slow consumers.
- [`go-service/internal/realtime/hub.go:13-104`](go-service/internal/realtime/hub.go#L13-L104) — еквівалентний Go Hub із bounded client queues.
- [`node-service/scripts/ws-database-demo.js:3-38`](node-service/scripts/ws-database-demo.js#L3-L38) — підключає два клієнти, створює ліцензію та перевіряє отримання `license_created` обома клієнтами.

Stream export із Backpressure:

- [`node-service/src/services/database-export.service.js:19-47`](node-service/src/services/database-export.service.js#L19-L47) — async source читає repository batch-ами; `Readable → Transform → Writable` з'єднані через `pipeline()`, який автоматично керує Backpressure.
- [`node-service/src/repositories/postgres-license.repository.js:74-89`](node-service/src/repositories/postgres-license.repository.js#L74-L89) і [`node-service/src/repositories/mongo-license.repository.js:74-78`](node-service/src/repositories/mongo-license.repository.js#L74-L78) — порційне читання без завантаження всієї БД у RAM.
- [`go-service/internal/streaming/database_export.go:38-108`](go-service/internal/streaming/database_export.go#L38-L108) — `io.Pipe`; producer блокується, поки consumer через `io.CopyBuffer` не прочитає дані.
- [`go-service/internal/repository/postgres.go:109-127`](go-service/internal/repository/postgres.go#L109-L127) і [`go-service/internal/repository/mongo.go:109-123`](go-service/internal/repository/mongo.go#L109-L123) — keyset pagination/batch cursor.
- Node [`database-export.service.js:39-46`](node-service/src/services/database-export.service.js#L39-L46) і Go [`database_export.go:92-103`](go-service/internal/streaming/database_export.go#L92-L103) надсилають WebSocket-події `export_progress` та `export_completed`.
- [`node-service/src/routes/export.routes.js:5-7`](node-service/src/routes/export.routes.js#L5-L7) і [`go-service/internal/router/router.go:35-37`](go-service/internal/router/router.go#L35-L37) — endpoint `GET /api/licenses/export`.

### 8. Збереження багаторівневої архітектури

- [`node-service/src/app.js:20-44`](node-service/src/app.js#L20-L44) — dependency wiring та порядок middleware/routes; Router не містить бізнес-логіки.
- [`node-service/src/routes/license.routes.js:5-13`](node-service/src/routes/license.routes.js#L5-L13) — Router лише зв'язує method/path, validation middleware і Controller.
- [`node-service/src/controllers/license.controller.js:14-39`](node-service/src/controllers/license.controller.js#L14-L39) — Controller перетворює HTTP request/response, але не працює з БД.
- [`node-service/src/services/license.service.js:4-83`](node-service/src/services/license.service.js#L4-L83) — бізнес-правила та repository abstraction.
- [`go-service/internal/router/router.go:20-63`](go-service/internal/router/router.go#L20-L63) — усі Go endpoints і middleware composition.
- [`go-service/internal/controller/license.go:48-189`](go-service/internal/controller/license.go#L48-L189) — parsing, validation, HTTP status та централізоване перетворення Service errors.
- [`go-service/internal/service/license.go:20-150`](go-service/internal/service/license.go#L20-L150) — бізнес-логіка без залежності від `net/http`, PostgreSQL або MongoDB.

Фактичний ланцюг виклику лишився: `Router → Controller → Service → Repository interface → PostgreSQL/MongoDB`.

### 9. Тести Service, API та інтеграції

- [`node-service/test/service.test.js:12-62`](node-service/test/service.test.js#L12-L62) — Node Service CRUD, defaults, duplicate key, not found і CRUD events без HTTP/БД.
- [`node-service/test/api.test.js:26-86`](node-service/test/api.test.js#L26-L86) — Node API: POST/GET/PUT/DELETE, invalid POST/JSON, `404`, `409`, `500`.
- [`go-service/internal/service/license_test.go:25-92`](go-service/internal/service/license_test.go#L25-L92) — еквівалентні Go Service unit-тести через InMemoryRepository.
- [`go-service/internal/router/router_test.go:70-149`](go-service/internal/router/router_test.go#L70-L149) — Go API tests із `httptest`, без запуску production server.
- [`node-service/test/database-export.test.js:9-26`](node-service/test/database-export.test.js#L9-L26) і [`go-service/internal/streaming/database_export_test.go:14-43`](go-service/internal/streaming/database_export_test.go#L14-L43) — export і progress events.
- [`scripts/smoke-test.ps1:18-66`](scripts/smoke-test.ps1#L18-L66) — інтеграційна перевірка запущених контейнерів: Node створює, Go читає/оновлює/видаляє через спільну реальну БД; додатково перевіряються `409`, Mongo events, обидва exports, `COMMIT` і незмінність балансу після `ROLLBACK`.
- [`node-service/Dockerfile:1-15`](node-service/Dockerfile#L1-L15) і [`go-service/Dockerfile:1-18`](go-service/Dockerfile#L1-L18) запускають тести під час `docker compose build`, тому образ не збирається при падінні тестів.

## Підготовка і запуск

Файл `.env` не потрапляє в Git. Для першого запуску треба скопіювати `.env.example` у `.env` і заповнити всі значення. На ноутбуці для захисту `.env` уже підготовлено.

```powershell
cd C:\node_go_labs
docker compose up --build -d
docker compose ps
```

Після фінального перенесення проєкт запускається саме з кореня репозиторію. Node слухає `http://localhost:3006`, Go — `http://localhost:8086`; WebSocket endpoint — `/ws`.

Швидка повна перевірка поточного repository:

```powershell
.\scripts\smoke-test.ps1
```

Скрипт перевіряє health, CRUD між двома runtime через спільну реальну БД, `409 Conflict`, Mongo event log, `COMMIT`, `ROLLBACK` і обидва CSV exports.

## Допоміжні скрипти: що вони роблять і навіщо потрібні

Скрипти не містять основної бізнес-логіки застосунку і не замінюють Controller, Service або Repository. Вони потрібні для відтворюваної демонстрації: замість ручного виконання десятків команд запускається один сценарій, який сам перевіряє результат і завершується помилкою, якщо хоча б одна вимога не виконана.

### `scripts/smoke-test.ps1` — повна інтеграційна перевірка лабораторної №6

Запуск:

```powershell
.\scripts\smoke-test.ps1
```

Перед запуском контейнери мають бути підняті через `docker compose up --build -d`. За потреби адреси сервісів можна передати параметрами `-NodeBaseUrl` і `-GoBaseUrl`.

- [`scripts/smoke-test.ps1:1-18`](scripts/smoke-test.ps1#L1-L18) — приймає адреси Node/Go, викликає обидва `/health`, перевіряє статус `UP` і однаковий тип Repository.
- [`scripts/smoke-test.ps1:20-27`](scripts/smoke-test.ps1#L20-L27) — створює унікальну ліцензію через Node, читає її та оновлює через Go. Це доводить, що обидва runtime працюють зі спільним persistent storage, а не з окремою пам’яттю процесів.
- [`scripts/smoke-test.ps1:29-35`](scripts/smoke-test.ps1#L29-L35) — повторно надсилає той самий `key`, очікує `409 Conflict`, а потім перевіряє, що MongoDB зберегла події створення й оновлення.
- [`scripts/smoke-test.ps1:37-41`](scripts/smoke-test.ps1#L37-L41) — запускає CSV export у Node та Go і перевіряє, що обидві потокові реалізації реально обробили записи.
- [`scripts/smoke-test.ps1:43-54`](scripts/smoke-test.ps1#L43-L54) — перевіряє успішний `COMMIT`, потім навмисно викликає помилку недостатнього балансу та порівнює рахунки до/після, доводячи коректний `ROLLBACK`.
- [`scripts/smoke-test.ps1:56-67`](scripts/smoke-test.ps1#L56-L67) — видаляє тестову ліцензію та друкує підсумок `PASS`. Унікальний ключ із timestamp не конфліктує з попередніми запусками.

Навіщо потрібен: це не unit-тест, а короткий end-to-end доказ, що запущені контейнери, обидва HTTP-сервіси та реальні PostgreSQL/MongoDB інтегровані правильно.

### `scripts/switch-repository.ps1` — демонстрація заміни Repository

Запуск:

```powershell
.\scripts\switch-repository.ps1 -Type mongo
.\scripts\switch-repository.ps1 -Type postgres
```

- [`scripts/switch-repository.ps1:1-5`](scripts/switch-repository.ps1#L1-L5) — дозволяє лише два коректні значення: `postgres` або `mongo`.
- [`scripts/switch-repository.ps1:8-20`](scripts/switch-repository.ps1#L8-L20) — знаходить кореневий `.env`, перевіряє наявність `REPO_TYPE` і змінює тільки цей параметр.
- [`scripts/switch-repository.ps1:22-28`](scripts/switch-repository.ps1#L22-L28) — перестворює лише контейнери `node` і `go`, після чого показує їх стан. Контейнери та volumes баз даних не видаляються.

Навіщо потрібен: наочно показує Dependency Inversion — Controller і Service не змінюються, а конкретна реалізація Repository обирається конфігурацією під час старту.

### `node-service/scripts/ws-database-demo.js` — WebSocket після операції з БД

Запуск усередині Node-контейнера:

```powershell
docker compose exec node node scripts/ws-database-demo.js
```

- [`node-service/scripts/ws-database-demo.js:3-13`](node-service/scripts/ws-database-demo.js#L3-L13) — створює WebSocket-клієнта та накопичує отримані JSON-події.
- [`node-service/scripts/ws-database-demo.js:16-24`](node-service/scripts/ws-database-demo.js#L16-L24) — очікує конкретну подію максимум 5 секунд; відсутність події вважається помилкою.
- [`node-service/scripts/ws-database-demo.js:26-37`](node-service/scripts/ws-database-demo.js#L26-L37) — одночасно підключає двох клієнтів, створює ліцензію через HTTP/Repository і завершується успішно лише після отримання `license_created` обома клієнтами.

Навіщо потрібен: доводить повний ланцюжок `HTTP POST → Service → Database Repository → event → WebSocket broadcast`, а також broadcast для кількох клієнтів.

### `node-service/scripts/ws-demo.js` — базова демонстрація WebSocket із лабораторної №5

Запуск:

```powershell
docker compose exec node node scripts/ws-demo.js
```

- [`node-service/scripts/ws-demo.js:3-17`](node-service/scripts/ws-demo.js#L3-L17) — відкриває два клієнти, після їх підключення надсилає валідне повідомлення, потім навмисно надсилає `invalid-json` і закриває з’єднання.
- [`node-service/scripts/ws-demo.js:19-25`](node-service/scripts/ws-demo.js#L19-L25) — друкує відповіді для обох клієнтів і повертає ненульовий exit code при socket error.

Навіщо потрібен: ізольовано демонструє broadcast і обробку некоректного повідомлення. На відміну від `ws-database-demo.js`, цей сценарій не створює запис у БД і залишений для демонстрації вимог попередньої лабораторної.

### `bench/run-backpressure.ps1` — порівняння safe/unsafe stream processing

Запуск, наприклад для файлу 100 MB і повільного consumer:

```powershell
.\bench\run-backpressure.ps1 -SizeMB 100 -SlowDelayMS 2
```

- [`bench/run-backpressure.ps1:1-6`](bench/run-backpressure.ps1#L1-L6) — приймає розмір тестових даних і штучну затримку consumer.
- [`bench/run-backpressure.ps1:10-19`](bench/run-backpressure.ps1#L10-L19) — опитує endpoint стану, доки background job не завершиться або не поверне помилку.
- [`bench/run-backpressure.ps1:21-38`](bench/run-backpressure.ps1#L21-L38) — запускає один сценарій і збирає runtime, режим, час, приріст пам’яті та кількість рядків.
- [`bench/run-backpressure.ps1:40-49`](bench/run-backpressure.ps1#L40-L49) — генерує однакові source-файли та послідовно запускає `safe` і `unsafe` режими для Node та Go.
- [`bench/run-backpressure.ps1:51-65`](bench/run-backpressure.ps1#L51-L65) — друкує таблицю та записує відтворюваний звіт у `bench/results.md`.

Навіщо потрібен: чисельно демонструє backpressure із лабораторної №5. `safe` обмежує дані в пам’яті швидкістю consumer, тоді як `unsafe` накопичує їх перед записом. Скрипт не потрібен для звичайного запуску CRUD лабораторної №6.

### Автоматичні init-скрипти баз даних

Ці файли вручну запускати не потрібно: Docker виконує їх під час першого створення відповідного volume.

- [`init.sql:1-25`](init.sql#L1-L25) — створює таблиці PostgreSQL, constraint-и та два початкові рахунки для транзакційної демонстрації.
- [`mongo-init.js:1-3`](mongo-init.js#L1-L3) — створює у MongoDB унікальний індекс ключа ліцензії та індекс часу подій; collections створюються MongoDB автоматично під час `createIndex`.
- [`docker-compose.yml:4-38`](docker-compose.yml#L4-L38) — монтує ці файли у стандартні init-каталоги офіційних образів PostgreSQL і MongoDB.

Після звичайного `docker compose down` дані та volumes зберігаються, тому init-скрипти повторно не виконуються. Для повної чистої ініціалізації використовується `docker compose down -v`, але ця команда видаляє локальні дані лабораторних БД.

## Перемикання PostgreSQL ↔ MongoDB

Змінюється лише один рядок у `.env`:

```dotenv
REPO_TYPE=postgres
```

або:

```dotenv
REPO_TYPE=mongo
```

Готова команда змінює цей рядок і перестворює лише застосунки, не торкаючись БД:

```powershell
.\scripts\switch-repository.ps1 -Type mongo
.\scripts\smoke-test.ps1

.\scripts\switch-repository.ps1 -Type postgres
.\scripts\smoke-test.ps1
```

Service та Controller при перемиканні не змінюються.

## HTTP API

Основні endpoint однакові на портах `3006` і `8086`:

- `GET /health`
- `GET /api/licenses`
- `GET /api/licenses/:id`
- `POST /api/licenses`
- `PUT /api/licenses/:id`
- `DELETE /api/licenses/:id`
- `GET /api/events?limit=50`
- `GET /api/accounts`
- `POST /api/accounts/transfer`
- `GET /api/licenses/export?batch_size=100`
- `GET /ws` через WebSocket Upgrade
- endpoint лабораторної №5 у `/streams/*` також збережені

Для сумісності CRUD доступний також через `/licenses`.

Приклад ліцензії:

```json
{
  "key": "MTGM-LAB6-DEMO-0001",
  "product": "MTG MODS VIP",
  "owner": "student"
}
```

Дубль `key` перетворюється у `409 Conflict` і для PostgreSQL UNIQUE constraint, і для MongoDB unique index.

## PostgreSQL transaction

`POST /api/accounts/transfer` вручну бере connection із pool, виконує `BEGIN`, блокує обидва рахунки через `FOR UPDATE`, перевіряє баланс, робить два `UPDATE` і завершує `COMMIT`. Будь-яка помилка проходить через `ROLLBACK`; Node завжди викликає `release()`, Go — `defer tx.Rollback()`.

Успішний сценарій:

```powershell
curl.exe -X POST http://localhost:3006/api/accounts/transfer -H "Content-Type: application/json" -d "{\"from_account_id\":1,\"to_account_id\":2,\"amount\":10}"
```

Rollback через недостатній баланс:

```powershell
curl.exe -X POST http://localhost:8086/api/accounts/transfer -H "Content-Type: application/json" -d "{\"from_account_id\":2,\"to_account_id\":1,\"amount\":999999}"
```

## Polyglot persistence та WebSocket

Навіть коли основний CRUD працює через PostgreSQL, події `license_created`, `license_updated`, `license_deleted` незалежно записуються в MongoDB `events`. Та сама подія broadcast-иться всім WebSocket-клієнтам.

Перевірка двох одночасних WebSocket-клієнтів:

```powershell
docker compose exec node node scripts/ws-database-demo.js
```

Скрипт підключає два клієнти, створює ліцензію через реальну БД і завершується лише тоді, коли обидва отримали `license_created`.

## Stream export із Backpressure

`GET /api/licenses/export` не виконує `FindAll` для всього набору. Repository читає ліцензії порціями через keyset pagination.

- Node: `Readable.from()` → `Transform` (object → CSV) → `Writable`, об'єднані `stream.pipeline()`.
- Go: producer читає batch-и та пише CSV в `io.Pipe`; запис блокується, поки consumer не прочитає дані.
- Після кожного запису WebSocket отримує `export_progress`; наприкінці — `export_completed`.
- Файли зберігаються у `data/node` і `data/go`, але не комітяться.

```powershell
curl.exe "http://localhost:3006/api/licenses/export?batch_size=100"
curl.exe "http://localhost:8086/api/licenses/export?batch_size=100"
```

## Тести

Docker build автоматично запускає unit, Service, API, Streams і WebSocket тести. InMemory repository дозволяє тестувати Service без HTTP і без БД.

```powershell
docker compose build --no-cache node go
.\scripts\smoke-test.ps1
```

Для повного чистого повтору, включно з повторним виконанням `init.sql` і `mongo-init.js`:

```powershell
docker compose down -v
docker compose up --build -d
```
