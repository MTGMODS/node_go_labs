# Лабораторна робота 4 — багаторівнева архітектура

Еквівалентні Node.js та Go реалізації `license-service` з першої лабораторної. Предметна область не змінена: система керує лише ліцензіями, без окремих сутностей пристроїв або платежів.

## Архітектура

```text
HTTP Request
    ↓
Router
    ↓
Controller
    ↓
Service
    ↓
LicenseRepository
    ↓
InMemoryLicenseRepository
```

- **Router** визначає endpoint, HTTP method, Controller та middleware.
- **Controller** читає HTTP-запит, перевіряє identifier і тіло та формує HTTP-відповідь.
- **Service** реалізує бізнес-правила, дефолти й унікальність ліцензійного ключа.
- **Repository** приховує реалізацію сховища.
- **DataStore** — захищені in-memory колекції, окремі для Node.js та Go.

Service отримує Repository через конструктор. У Go залежність формалізована через `LicenseRepository` interface; у Node.js використано окремий repository class зі структурним контрактом. Для переходу на БД достатньо реалізувати той самий контракт і замінити dependency у composition root.

## Модель ліцензії

```json
{
  "id": 1,
  "key": "MTGM-VIP1-AAAA-0001",
  "product": "MTG MODS VIP",
  "owner": "student",
  "status": "NOT_ACTIVATED",
  "duration_days": 30,
  "max_devices": 1
}
```

На `POST` обов'язкові `key`, `product` і `owner`. Значення за замовчуванням:

- `status=NOT_ACTIVATED`;
- `duration_days=30`;
- `max_devices=1`.

Допустимі статуси: `NOT_ACTIVATED`, `ACTIVE`, `EXPIRED`, `BANNED`. `PUT` підтримує часткове оновлення. Поле `key` унікальне.

## API

- `GET /health` — `200`;
- `GET /licenses` — `200`;
- `GET /licenses/:id` — `200` або `404`;
- `POST /licenses` — `201`, `400` або `409`;
- `PUT /licenses/:id` — `200`, `400`, `404` або `409`;
- `DELETE /licenses/:id` — `204`, `400` або `404`.

Помилки мають спільний формат:

```json
{
  "error": "license_not_found",
  "message": "License with id 42 was not found"
}
```

## Middleware

Обидва сервіси мають:

1. Request logging: method, path, status code, duration.
2. Request validation: тип контенту/JSON, допустимі поля та значення, ліміт тіла запиту.

## Запуск

```powershell
cd C:\node_go_labs\lab4
docker compose up --build -d
docker compose ps
```

- Node.js: `http://localhost:3004`
- Go: `http://localhost:8084`

```powershell
curl.exe -X POST http://localhost:3004/licenses -H "Content-Type: application/json" -d "{\"key\":\"MTGM-VIP1-AAAA-0001\",\"product\":\"MTG MODS VIP\",\"owner\":\"student\"}"
curl.exe http://localhost:3004/licenses
curl.exe -X PUT http://localhost:3004/licenses/1 -H "Content-Type: application/json" -d "{\"status\":\"ACTIVE\"}"
```

Для Go використовуються ті самі запити з портом `8084`.

## Тести

Dockerfile кожного сервісу запускає тести під час build. Є окремі Service tests без HTTP-сервера та API tests для успішних операцій, validation, `404`, `409`, `500` і правильних HTTP status codes.
