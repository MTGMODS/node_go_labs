# node_go_labs

Лабораторні роботи з порівняння серверних моделей **Node.js** і **Go**:

- лабораторна 1: однаковий CRUD + `/health`;
- лабораторна 2: I/O та CPU навантаження, блокування event loop, goroutines і benchmark;
- лабораторна 3: автономний конкурентний pipeline на Go у папці `lab3/`;
- лабораторна 4: багаторівневий `license-service` на Node.js та Go у папці `lab4/`.

Тема — спрощений `license-service` з [license-management-platform](https://github.com/MTGMODS/license-management-platform): ліцензійні ключі в **спільній PostgreSQL**. Без JWT і ботів.

## Що де лежить

| Папка / сервіс | Стек | URL |
| --- | --- | --- |
| `node/` | Express | http://localhost:3000 |
| `go/` | `net/http` | http://localhost:8080 |
| `postgres/` | PostgreSQL 16 | localhost:5433 (`labs` / `labs` / `licenses`) |
| `web/` | HTML + nginx | http://localhost:8081 |
| `bench/` | k6 + PowerShell | сценарії лабораторної 2 |
| `lab3/` | Go + окремий Docker Compose | конкурентний pipeline лабораторної 3 |
| `lab4/` | Express + `net/http` | layered license-service, порти 3004 та 8084 |

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

## Аналогія з Python

| Python | Node | Go |
| --- | --- | --- |
| `psycopg` + SQL | пакет `pg` у `store.js` | `database/sql` + `lib/pq` у `store.go` |
| `@app.get("/licenses")` | `router.get("/")` | `mux.HandleFunc("GET /licenses", ...)` |
| Pydantic | `readString` / `readInt` | `LicensePayload` + `applyPayload` |
| `uvicorn` | `app.listen(...)` | `http.ListenAndServe(...)` |

## Лабораторна 2: I/O та CPU

Нові endpoint-и мають однакові параметри, щоб сервіси виконували однаковий обсяг роботи.

| Runtime | Метод і шлях | Реалізація |
| --- | --- | --- |
| Node | `GET /io?delay_ms=1000` | асинхронний `setTimeout`, event loop не блокується |
| Go | `GET /io?delay_ms=1000` | `time.Timer`, блокується лише goroutine запиту |
| Node | `GET /cpu?iterations=75000000&tasks=4` | синхронні цикли в main thread |
| Go | `GET /cpu` або `/cpu/sequential` | ті самі цикли послідовно |
| Go | `GET /cpu/parallel` | ті самі цикли в окремих goroutines |

У CPU-відповіді `result` завжди дорівнює `tasks * iterations`. Це підтверджує, що sequential та parallel варіанти виконали однаковий обсяг роботи.

Швидка ручна перевірка:

```powershell
curl.exe "http://localhost:3000/io?delay_ms=1000"
curl.exe "http://localhost:8080/io?delay_ms=1000"
curl.exe "http://localhost:3000/cpu?iterations=75000000&tasks=4"
curl.exe "http://localhost:8080/cpu/sequential?iterations=75000000&tasks=4"
curl.exe "http://localhost:8080/cpu/parallel?iterations=75000000&tasks=4"
```

Поведінка `/health` під час CPU-запиту:

```powershell
./bench/check-health-during-cpu.ps1
```

Скрипт спочатку запускає важкий CPU-запит, потім надсилає п'ять `/health` запитів. Для Node перший health чекає завершення CPU-циклу. Go scheduler продовжує планувати health handler, тому він відповідає під час CPU-роботи.

Повний benchmark:

```powershell
./bench/run-benchmarks.ps1
```

Він послідовно запускає п'ять k6-сценаріїв, паралельно збирає `docker stats` і записує:

- підсумкову таблицю в `bench/results.md`;
- health-експеримент у `bench/health-results.md`;
- raw JSON/CSV у `bench/results/`.
