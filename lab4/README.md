# Лабораторна робота 4

Два еквівалентні HTTP-сервіси на Node.js та Go реалізують CRUD користувачів за багаторівневою архітектурою.

```text
HTTP Request
    -> Router
    -> Controller
    -> Service
    -> UserRepository abstraction
    -> InMemoryUserRepository
```

## Відповідальність рівнів

- Router визначає HTTP method і path, підключає controller та middleware.
- Controller читає HTTP request, перевіряє identifier і JSON, викликає Service та формує HTTP response.
- Service містить нормалізацію даних, перевірку унікальності email та правила CRUD. Він не залежить від HTTP або конкретного storage.
- Repository реалізує операції з даними та приховує in-memory storage.
- DataStore зберігає користувачів у пам'яті окремого процесу.

У Go Service приймає `UserRepository` interface. У Node.js Service отримує repository через constructor injection і використовує лише його методи. Тому in-memory реалізацію можна замінити на database repository без зміни бізнес-логіки.

## Запуск

```powershell
cd C:\node_go_labs\lab4
docker compose up --build -d
docker compose ps
```

- Node.js: `http://localhost:3004`
- Go: `http://localhost:8084`

Обидва сервіси мають однакове API:

```text
GET    /health
GET    /api/users
GET    /api/users/:id
POST   /api/users
PUT    /api/users/:id
DELETE /api/users/:id
```

Приклад створення:

```powershell
curl.exe -X POST http://localhost:3004/api/users `
  -H "Content-Type: application/json" `
  -d '{"name":"John Doe","email":"john@example.com"}'
```

Для Go використовується той самий запит із портом `8084`.

## Validation та помилки

Для `POST` і `PUT` обов'язкові непорожні string-поля `name` та `email`. Email повинен мати коректний формат і бути унікальним. Identifier повинен бути додатним цілим числом.

Централізований error response:

```json
{
  "error": "user_not_found",
  "message": "User with id 42 was not found"
}
```

Сервіси розрізняють `400 Bad Request`, `404 Not Found`, `409 Conflict` і `500 Internal Server Error`.

## Middleware

- Request logging записує HTTP method, path, status code та duration.
- Request validation перевіряє спільні вимоги до JSON-запитів. Controller додатково перевіряє body та identifier.

## Тести

Тести Service перевіряють CRUD, missing entity, нормалізацію та duplicate email. API-тести перевіряють успішні GET/POST/DELETE, invalid POST, missing resource, conflict, invalid JSON і HTTP status codes.

Тести автоматично виконуються під час Docker build. Окремий запуск:

```powershell
docker compose build --no-cache node go
```

Зупинка сервісів:

```powershell
docker compose down
```
