# Лабораторна робота 3

Консольний застосунок на Go демонструє конкурентну обробку числових задач за допомогою worker pool, channels, `sync.WaitGroup` і `context.Context`.

## Архітектура

```text
Producer -> jobs channel -> Worker Pool -> results channel -> Aggregator
```

- `Producer` створює задану кількість `Job`, передає їх у `jobs` і після завершення закриває channel.
- `Worker Pool` складається з конфігурованої кількості goroutines. Кожен worker читає `Job`, виконує CPU-обчислення і передає `Result` у `results`.
- `sync.WaitGroup` очікує завершення всіх workers. Після цього `results` коректно закривається.
- `Aggregator` читає результати, рахує завершені задачі, checksum, час виконання та throughput.
- `context.Context` і `select` забезпечують скасування producer та workers через timeout, `Ctrl+C` або завершення контейнера.

Channels `jobs` і `results` використовують однаковий конфігурований розмір buffer. Значення `0` створює unbuffered channels.

## Параметри

- `-tasks` — кількість задач, будь-яке додатне число;
- `-workers` — `1`, `2`, `4`, `8` або `16`;
- `-buffer` — `0`, `10`, `100` або `1000`;
- `-iterations` — кількість ітерацій CPU-обчислення для кожної задачі;
- `-timeout` — необов'язкове обмеження часу, наприклад `500ms`;
- `-format` — формат результату: `text` або `json`.

## Запуск у Docker

```powershell
cd C:\node_go_labs\lab3
docker compose up --build
```

Застосунок виконує один pipeline, друкує статистику і завершується з кодом `0`.

Запуск з іншою конфігурацією:

```powershell
docker compose run --rm pipeline -tasks=100000 -workers=8 -buffer=1000 -iterations=10000
```

Перевірка скасування через `context`:

```powershell
docker compose run --rm pipeline -tasks=100000 -workers=8 -buffer=100 -iterations=10000 -timeout=1ms
```

У такому запуску `Canceled` дорівнює `true`, а кількість `Completed` може бути меншою за `Tasks`.

## Race detector

```powershell
docker compose --profile race run --rm race
```

Відсутність повідомлення `WARNING: DATA RACE` означає, що race detector не виявив конкурентного доступу до пам'яті.

## Benchmark

```powershell
.\benchmark.ps1
```

Benchmark виконує однакові `10000` задачі для таких конфігурацій:

- `1`, `2`, `4`, `8`, `16` workers з unbuffered channels;
- `4` workers з buffers `10`, `100`, `1000`.

Час вимірюється всередині Go-програми, тому створення Docker-контейнера не входить у результат. Підсумкова таблиця записується в `results.md`.

`Checksum` має бути однаковим у всіх рядках з однаковими `tasks` та `iterations`. Це підтверджує, що всі конфігурації виконали той самий обсяг роботи. Значення checksum у race-запуску відрізняється, оскільки там використовується менша кількість `iterations`.

Збільшення кількості workers може скоротити час CPU-обробки, доки системі вистачає доступних ядер. Buffered channels зменшують кількість моментів, коли producer, workers і aggregator очікують один одного. Надмірне збільшення buffer не гарантує подальшого прискорення та використовує більше пам'яті.
