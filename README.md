# node_go_labs

Лабораторна 1: два однакових HTTP-сервіси (CRUD + `/health`) на **Node.js** і **Go**.

Тема — спрощений `license-service` з [license-management-platform](https://github.com/MTGMODS/license-management-platform): ліцензійні ключі в **спільній PostgreSQL**. Без JWT і ботів.

## Що де лежить

| Папка / сервіс | Стек | URL |
| --- | --- | --- |
| `node/` | Express | http://localhost:3000 |
| `go/` | `net/http` | http://localhost:8080 |
| `postgres/` | PostgreSQL 16 | localhost:5433 (`labs` / `labs` / `licenses`) |
| `web/` | HTML + nginx | http://localhost:8081 |

Node і Go на комп ставити не потрібно. Потрібен лише Docker.

```bash
docker compose up --build
```

Пісочниця замість curl: відкрий http://localhost:8081 — підпис, кнопка, відповідь сервера. Можна перемикати Node/Go; дані одні, бо БД спільна.

Зупинити: `Ctrl+C`, потім `docker compose down`. Стерти дані БД: `docker compose down -v`.

## API (однакове в обох сервісах)

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

Після першого старту є демо-запис `id=1` (скрипт `postgres/init.sql`). Далі дані живуть у Postgres, поки не зробиш `down -v`.

| Метод | Шлях | Код успіху | Що робить |
| --- | --- | --- | --- |
| `GET` | `/health` | 200 | статус сервера і БД |
| `GET` | `/licenses` | 200 | список |
| `GET` | `/licenses/:id` | 200 | одна ліцензія |
| `POST` | `/licenses` | 201 | створити |
| `PUT` | `/licenses/:id` | 200 | оновити (можна не всі поля) |
| `DELETE` | `/licenses/:id` | 204 | видалити (тіла відповіді немає) |

Інші коди: **400** невалідне тіло/id, **404** немає такого id, **409** ключ уже існує, **405** метод не дозволений, **503** БД недоступна (`/health`).

Обов'язкові поля на `POST`: `key`, `product`, `owner`. Якщо не передати інше: `status=NOT_ACTIVATED`, `duration_days=30`, `max_devices=1`.

## curl (Windows)

У PowerShell краще `curl.exe`.

```powershell
curl.exe http://localhost:3000/health
curl.exe http://localhost:3000/licenses
curl.exe -X POST http://localhost:3000/licenses -H "Content-Type: application/json" -d "{\"key\":\"MTGM-VIP1-AAAA-0002\",\"product\":\"MTG MODS VIP\",\"owner\":\"student\"}"
```

Те саме на `8080` для Go.

## Якщо ти з Python

| Python | Node | Go |
| --- | --- | --- |
| `psycopg` + SQL | пакет `pg` у `store.js` | `database/sql` + `lib/pq` у `store.go` |
| `@app.get("/licenses")` | `router.get("/")` | `mux.HandleFunc("GET /licenses", ...)` |
| Pydantic | `readString` / `readInt` | `LicensePayload` + `applyPayload` |
| `uvicorn` | `app.listen(...)` | `http.ListenAndServe(...)` |

## Захист лаби: що сказати

1. Два сервіси з **однаковим** REST API, різниця лише в рантаймі.
2. Сховище — **спільна PostgreSQL** у Docker; Node і Go не тримають свої копії даних.
3. `/health` перевіряє процес і ping до БД.
4. Коди: 201 створення, 204 видалення, 404 немає id, 409 дубль `key`.
5. Node: Express + event loop. Go: `net/http` + goroutines. HTML у `web/` лише клієнт, логіки CRUD там немає.
