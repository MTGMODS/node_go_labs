# Лабораторна робота 5 — Streams Backpressure та WebSockets

Розвиток багаторівневого `license-service` з лабораторної №4. До Node.js і Go сервісів додано потокову обробку великих NDJSON-файлів, WebSocket-комунікацію та передавання прогресу обробки в реальному часі.

## Архітектура

```text
HTTP API ──→ Controller ──→ Stream Service
                               │
Large NDJSON ──→ Read ──→ Transform ──→ Slow Write
                               │
                               └──→ progress events ──→ WebSocket Hub ──→ clients
```

CRUD ліцензій і розділення Router → Controller → Service → Repository з лабораторної №4 збережені.

## Stream Processing Module

`POST /streams/source` створює тестовий NDJSON-файл заданого розміру. За замовчуванням створюється 100 MB. Файл генерується та записується частинами.

`POST /streams/process` запускає фонову обробку:

- `safe` — bounded pipeline із Backpressure;
- `unsafe` — лише контрольний експеримент: producer не враховує швидкість consumer і накопичує chunks у пам'яті;
- Transform переводить текст у верхній регістр і підраховує рядки;
- повільний consumer моделюється затримкою запису;
- результат записується в новий файл;
- job зберігає час, кількість байтів, рядків і приріст пам'яті.

Node.js використовує `fs.createReadStream`, `Transform`, `Writable` та `stream.pipeline()`. `pipeline()` з'єднує Readable → Transform → Writable, автоматично передає Backpressure і завершує весь ланцюг при помилці. Go використовує читання chunks і bounded channels; заповнений канал блокує producer.

Типовий chunk — 64 KiB. Це достатньо великий блок, щоб не витрачати забагато часу на системні виклики, але достатньо малий, щоб пам'ять safe-режиму залишалася обмеженою незалежно від розміру файла. Розмір можна змінити параметром `high_water_mark_kb` у Node.js або `chunk_size_kb` у Go.

## WebSocket Module

Endpoint: `ws://localhost:3005/ws` для Node.js і `ws://localhost:8085/ws` для Go.

Підтримуються:

- реєстрація клієнта з `clientId` і `connectedAt`;
- валідація JSON-повідомлень;
- broadcast повідомлень іншим клієнтам;
- connect, message, close, error і heartbeat/ping-pong;
- progress/completed/failed events від Stream Service;
- bounded queue або `bufferedAmount` limit для повільних клієнтів;
- message dropping і розрив стабільно повільного з'єднання.

Формат повідомлення клієнта:

```json
{ "type": "message", "text": "Hello" }
```

## HTTP endpoints

- `GET /health`
- `GET|POST /licenses`, `GET|PUT|DELETE /licenses/:id`
- `GET|POST /streams/source`
- `POST /streams/process`
- `GET /streams/jobs`
- `GET /streams/jobs/:id`
- `GET /realtime/clients`
- `GET /ws` з WebSocket Upgrade

## Запуск

```powershell
cd C:\node_go_labs\lab5
docker compose up --build -d
docker compose ps
```

- Node.js HTTP і demo UI: http://localhost:3005
- Go HTTP: http://localhost:8085

## Демонстрація Streams

```powershell
curl.exe -X POST http://localhost:3005/streams/source -H "Content-Type: application/json" -d "{\"size_mb\":100}"
curl.exe -X POST http://localhost:3005/streams/process -H "Content-Type: application/json" -d "{\"mode\":\"safe\",\"slow_delay_ms\":2,\"high_water_mark_kb\":64}"
curl.exe http://localhost:3005/streams/jobs
```

Для Go використовуються порт `8085` і параметр `chunk_size_kb`.

Автоматичне порівняння safe/unsafe обох runtime:

```powershell
.\bench\run-backpressure.ps1 -SizeMB 100 -SlowDelayMS 2
```

Результат записується в `bench/results.md`.

Контрольний замір для файла 100 MB і затримки consumer 2 ms показав однакові 1 059 168 рядків у всіх сценаріях. Приріст пам'яті: Node.js safe — 9.72 MB, unsafe — 103.54 MB; Go safe — 3.26 MB, unsafe — 167.50 MB. Отже, Backpressure майже не змінює час у цьому тесті, бо його задає повільний consumer, але не дозволяє producer накопичити файл у RAM.

## Демонстрація WebSocket

Відкрийте http://localhost:3005 у двох вкладках або запустіть два автоматичні клієнти:

```powershell
docker compose exec node node scripts/ws-demo.js ws://127.0.0.1:3005/ws
docker compose exec node node scripts/ws-demo.js ws://go:8085/ws
```

Під час stream job у клієнтів з'являються повідомлення `progress`, а після завершення — `completed`.

## Тести

Dockerfile обох сервісів запускає Service, API, Stream і WebSocket тести під час build. Go-додатково перевіряється race detector. Великі згенеровані файли не зберігаються в Git.
