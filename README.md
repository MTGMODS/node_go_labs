# node_go_labs

Лабораторна 1: два однакових HTTP-сервіси (CRUD + `/health`) на **Node.js** і **Go**.

Тема — спрощений `license-service` з прод-проєкту [license-management-platform](https://github.com/MTGMODS/license-management-platform): ліцензійні ключі, статуси, ліміт пристроїв. Без PostgreSQL, JWT і ботів — лише in-memory сховище, як вимагає методичка.

## Що де лежить

| Папка | Стек | Порт |
| --- | --- | --- |
| `node/` | Express (рантайм Node.js) | http://localhost:3000 |
| `go/` | стандартний `net/http` | http://localhost:8080 |

Node і Go на комп ставити не потрібно. Потрібен лише Docker.

```bash
docker compose up --build
```

Зупинити: `Ctrl+C`, потім за бажанням `docker compose down`.

## API (однакове в обох сервісах)

Сутність `license`:

```json
{
  "id": 1,
  "key": "MTGM-VIP1-AAAA-0001",
  "product": "MTG MODS VIP",
  "owner": "bogdan",
  "status": "ACTIVE",
  "duration_days": 30,
  "max_devices": 2
}
```

`status`: `NOT_ACTIVATED` | `ACTIVE` | `EXPIRED` | `BANNED`.

Після старту вже є одна демо-ліцензія з `id=1`. Дані живуть у пам'яті процесу: рестарт контейнера = порожня база знову з демо-записом.

| Метод | Шлях | Код успіху | Що робить |
| --- | --- | --- | --- |
| `GET` | `/health` | 200 | статус сервера |
| `GET` | `/licenses` | 200 | список |
| `GET` | `/licenses/:id` | 200 | одна ліцензія |
| `POST` | `/licenses` | 201 | створити |
| `PUT` | `/licenses/:id` | 200 | оновити (можна не всі поля) |
| `DELETE` | `/licenses/:id` | 204 | видалити (тіла відповіді немає) |

Інші коди: **400** невалідне тіло/id, **404** немає такого id, **409** ключ уже існує, **405** метод не дозволений.

Обов'язкові поля на `POST`: `key`, `product`, `owner`. Якщо не передати інше: `status=NOT_ACTIVATED`, `duration_days=30`, `max_devices=1`.

## Як перевірити (Windows)

У PowerShell краще `curl.exe`, бо `curl` там часто є аліасом на `Invoke-WebRequest`.

Підстав `3000` для Node або `8080` для Go.

```powershell
curl.exe http://localhost:3000/health
curl.exe http://localhost:3000/licenses
curl.exe http://localhost:3000/licenses/1

curl.exe -X POST http://localhost:3000/licenses -H "Content-Type: application/json" -d "{\"key\":\"MTGM-VIP1-AAAA-0002\",\"product\":\"MTG MODS VIP\",\"owner\":\"student\"}"

curl.exe -X PUT http://localhost:3000/licenses/2 -H "Content-Type: application/json" -d "{\"status\":\"ACTIVE\"}"

curl.exe -X DELETE http://localhost:3000/licenses/2 -i
```

Те саме через PowerShell без curl:

```powershell
Invoke-RestMethod http://localhost:8080/health
Invoke-RestMethod http://localhost:8080/licenses

$body = @{
  key = "MTGM-VIP1-AAAA-0003"
  product = "MTG MODS VIP"
  owner = "student"
} | ConvertTo-Json

Invoke-RestMethod -Method Post -Uri http://localhost:8080/licenses -ContentType "application/json" -Body $body
```

## Якщо ти з Python (карта файлів)

Це майже FastAPI-сервіс, тільки без БД.

| Python | Node | Go |
| --- | --- | --- |
| `dict` / список об'єктів | `Map` у `store.js` | `map` + `sync.RWMutex` у `store.go` |
| `@app.get("/licenses")` | `router.get("/")` | `mux.HandleFunc("GET /licenses", ...)` |
| `pydantic` перевірка полів | функції `readString` / `readInt` | `LicensePayload` + `applyPayload` |
| `uvicorn` | `app.listen(...)` | `http.ListenAndServe(...)` |

Навіщо mutex лише в Go: Node обробляє запити по одному в event loop (лекція 1–2). У Go кожен запит — окрема goroutine, тому map треба захищати.

## Захист лаби: що сказати

1. Два сервіси з **однаковим** REST API, різниця лише в рантаймі.
2. Сховище **in-memory**, бо так у завданні; після рестарту дані зникають.
3. `/health` — стандартна перевірка, що процес живий (як у проді license-service).
4. Коди: 201 після створення, 204 після видалення, 404 якщо id немає, 409 якщо `key` дубль.
5. Node: Express поверх runtime (V8 + libuv, event loop). Go: native binary + `net/http` і goroutines.
