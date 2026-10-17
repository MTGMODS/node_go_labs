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
