---
title: "Несколько туннелей для микросервисов — гайд разработчика"
date: 2026-03-23T14:00:00+03:00
draft: false
description: "Каждому микросервису — свой туннель. Разбираем запуск нескольких туннелей из CLI и Docker Compose, тарифные лимиты и то, как сервисы находят адреса друг друга."
tags: [microservices, tunneling, docker, fxTunnel, developer-tools, multiple-tunnels, service-discovery]
image: ""
rubrics: [guides]
need: tunnels
cover: ["frontend :3000", "api :4000", "N туннелей"]
updated: 2026-09-27
---

## Проблема: микросервисам нужно несколько точек входа

Современные приложения редко укладываются в один процесс. Типичный стек -- это фронтенд, API-шлюз, несколько бэкенд-сервисов, база данных и, может быть, очередь сообщений. При локальной разработке каждый сервис живёт на своём порту: фронтенд на `localhost:3000`, API на `localhost:4000`, платёжный сервис на `localhost:4100` и так далее.

На вашей машине всё работает отлично. Но как только нужен внешний доступ -- коллега тестирует вашу ветку, [вебхук от Stripe](/blog/webhook-testing-with-tunnel/) приходит на платёжный сервис, или мобильное приложение подключается к API -- нужны публичные URL. Не один, а несколько -- по одному для каждого сервиса, который принимает внешний трафик.

Одним [туннелем](/blog/what-is-tunneling/) тут не обойтись. Нужно несколько туннелей одновременно, каждый направленный на свой сервис. В этом гайде разбираем, как настроить такую конфигурацию с fxTunnel -- и на хосте, и с Docker Compose -- и как связать микросервисы через URL туннелей.

## Архитектура: один туннель на сервис

Идея проста: каждый микросервис, которому нужен внешний доступ, получает свой туннель. Внутренние сервисы, которые общаются только с другими локальными сервисами, не нуждаются в туннеле — они используют Docker-сеть или localhost напрямую.

{{< arch caption="Каждый сервис — через свой туннель на общем сервере fxTunnel." bus="интернет → свой туннель на каждый сервис" >}}
- group: сервер fxTunnel
  items: [{title: HTTP-туннель, sub: "→ фронтенд :3000"}, {title: HTTP-туннель, sub: "→ API :4000"}, {title: TCP-туннель, sub: "→ PostgreSQL :5432"}]
- group: машина разработчика
  me: true
  items: [{title: Фронтенд, sub: "React, порт 3000"}, {title: API, sub: "Node, порт 4000"}, {title: PostgreSQL, sub: "порт 5432"}]
{{< /arch >}}

Каждый туннель создаёт свой публичный URL. Фронтенд доступен по адресу `https://front-abc.fxtun.ru`, API по адресу `https://api-xyz.fxtun.ru`, а база данных через [TCP-туннель](/blog/tcp-udp-tunneling-explained/) по адресу `tcp://fxtun.ru:18432` (у TCP/UDP-туннелей всегда базовый домен, без поддомена).

Держать несколько туннелей открытыми одновременно можно с тарифа Base (5 туннелей) и выше — Pro (15), Business (50). Бесплатный тариф позволяет держать открытым только один туннель, подробнее — в разделе «Тарифы» ниже.

### Что нуждается в туннеле, а что нет

Не каждый сервис в вашем стеке нуждается в туннеле. Простое правило:

| Сервис | Нужен туннель? | Почему |
|---|---|---|
| Фронтенд (React, Vue) | Да | Доступ из браузеров, мобильных приложений, от коллег |
| API-шлюз | Да | Вызывается фронтендом, вебхуками, внешними клиентами |
| Платёжный сервис | Да | Принимает вебхуки от Stripe, PayPal |
| Сервис авторизации | Возможно | Только если [OAuth-коллбэки](/blog/oauth-callback-localhost-tunnel/) приходят от внешних провайдеров |
| База данных | Редко | Только если коллеге нужен прямой доступ |
| Redis / Очередь сообщений | Нет | Только внутренняя коммуникация |
| Воркер / Фоновые задачи | Нет | Обрабатывает сообщения из очереди, входящего трафика нет |

## Быстрый старт: несколько туннелей из CLI

Самый быстрый способ запустить несколько туннелей — открыть несколько терминалов и запустить `fxtunnel` в каждом.

### Шаг 1. Установка fxTunnel

```bash
# Быстрая установка (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Проверяем
fxtunnel version

# Входим в аккаунт fxtun.ru (один раз)
fxtunnel login
```

### Шаг 2. Запуск сервисов

```bash
# Терминал 1: Фронтенд
cd frontend && npm run dev
# → listening on localhost:3000

# Терминал 2: API
cd api && npm run dev
# → listening on localhost:4000

# Терминал 3: Платёжный сервис
cd payment-service && go run main.go
# → listening on localhost:4100
```

### Шаг 3. Открытие туннеля для каждого сервиса

```bash
# Терминал 4: Туннель для фронтенда
fxtunnel http 3000

# Терминал 5: Туннель для API
fxtunnel http 4000

# Терминал 6: Туннель для платёжного сервиса
fxtunnel http 4100
```

Каждая команда выводит публичный URL:

```console
Connecting to fxtunnel server...
Tunnel established!
HTTP:  http://front-abc.fxtun.ru
HTTPS: https://front-abc.fxtun.ru
Forwarding to localhost:3000
```

Три сервиса, три туннеля, три публичных URL. Одновременно открытыми на бесплатном тарифе получится держать только один из них — для трёх параллельных туннелей нужен тариф Base.

## Автоматизация: bash-скрипт для нескольких туннелей

Открывать шесть окон терминала неудобно для ежедневной работы. Простой bash-скрипт запускает все сервисы и туннели в фоне и собирает URL. Скрипт ниже держит четыре туннеля открытыми одновременно — это укладывается в лимит тарифа Base (5 туннелей), на бесплатном тарифе (1 туннель) так не получится.

```bash
#!/bin/bash
# start-tunnels.sh — Запуск нескольких туннелей для микросервисов

set -e

SERVICES=(
  "frontend:3000:http"
  "api:4000:http"
  "payment:4100:http"
  "db:5432:tcp"
)

LOG_DIR="/tmp/tunnels"
mkdir -p "$LOG_DIR"

echo "Запуск туннелей..."

for entry in "${SERVICES[@]}"; do
  IFS=':' read -r name port proto <<< "$entry"
  log_file="$LOG_DIR/${name}.log"

  fxtunnel "$proto" "$port" > "$log_file" 2>&1 &
  echo "  $name ($proto:$port) — PID $!"
done

# Ждём появления URL
sleep 3

echo ""
echo "=== URL туннелей ==="
for entry in "${SERVICES[@]}"; do
  IFS=':' read -r name port proto <<< "$entry"
  log_file="$LOG_DIR/${name}.log"
  url=$(grep -oP 'https://[a-z0-9-]+\.fxtun\.ru|TCP: \Kfxtun\.ru:[0-9]+' "$log_file" 2>/dev/null | head -1)
  url=${url:-ожидание...}
  echo "  $name → $url"
done

echo ""
echo "Нажмите Ctrl+C для остановки всех туннелей"
wait
```

Использование:

```bash
chmod +x start-tunnels.sh
./start-tunnels.sh
```

Вывод:

```console
Запуск туннелей...
  frontend (http:3000) — PID 12345
  api (http:4000) — PID 12346
  payment (http:4100) — PID 12347
  db (tcp:5432) — PID 12348

=== URL туннелей ===
  frontend → https://front-abc.fxtun.ru
  api → https://api-xyz.fxtun.ru
  payment → https://pay-def.fxtun.ru
  db → fxtun.ru:18432

Нажмите Ctrl+C для остановки всех туннелей
```

При нажатии Ctrl+C все фоновые процессы завершаются, и туннели закрываются.

## Docker Compose: подход для команды

Для командной работы и воспроизводимых окружений Docker Compose — правильный инструмент. У fxTunnel нет отдельного Docker-образа клиента (образ `ghcr.io/mephistofox/fxtunnel` — это образ сервера), поэтому туннели поднимает не Compose, а клиент `fxtunnel` на хосте: сервисы публикуют порты на хост-машину, а один процесс `fxtunnel` с конфиг-файлом открывает туннель к каждому из них.

### Полный микросервисный стек

```yaml
# docker-compose.yml — Микросервисы, порты опубликованы на хост
version: "3.8"

services:
  frontend:
    build: ./frontend
    ports:
      - "3000:3000"
    environment:
      - REACT_APP_API_URL=${API_TUNNEL_URL:-http://localhost:4000}
    depends_on:
      - api

  api:
    build: ./api
    ports:
      - "4000:4000"
    environment:
      - DATABASE_URL=postgres://postgres:devpass@db:5432/myapp
      - REDIS_URL=redis://redis:6379
      - PAYMENT_SERVICE_URL=http://payment:4100
    depends_on:
      - db
      - redis

  payment:
    build: ./payment-service
    ports:
      - "4100:4100"
    environment:
      - DATABASE_URL=postgres://postgres:devpass@db:5432/myapp
      - STRIPE_WEBHOOK_SECRET=${STRIPE_WEBHOOK_SECRET}
    depends_on:
      - db

  db:
    image: postgres:16
    ports:
      - "5432:5432"
    environment:
      POSTGRES_PASSWORD: devpass
      POSTGRES_DB: myapp
    volumes:
      - pgdata:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine

volumes:
  pgdata:
```

### Туннели: один клиент, конфиг-файл с несколькими `tunnels[]`

Вместо четырёх отдельных команд опишите все туннели в одном YAML-файле и запустите один процесс `fxtunnel`:

```yaml
# client.yaml
server:
  address: tunnel.fxtun.ru:443
  token: sk_fxtunnel_...

tunnels:
  - type: http
    local_port: 3000
    subdomain: myapp-front
  - type: http
    local_port: 4000
    subdomain: myapp-api
  - type: http
    local_port: 4100
    subdomain: myapp-pay
  - type: tcp
    local_port: 5432
    remote_port: 15432
```

```bash
docker compose up -d
fxtunnel --config client.yaml
```

Вывод (строка `Tunnel established!` одна, дальше адрес и локальный порт каждого туннеля из `tunnels[]`):

```console
Connecting to fxtunnel server...
Tunnel established!
HTTP:  http://myapp-front.fxtun.ru
HTTPS: https://myapp-front.fxtun.ru
Forwarding to localhost:3000
HTTP:  http://myapp-api.fxtun.ru
HTTPS: https://myapp-api.fxtun.ru
Forwarding to localhost:4000
HTTP:  http://myapp-pay.fxtun.ru
HTTPS: https://myapp-pay.fxtun.ru
Forwarding to localhost:4100
TCP: fxtun.ru:15432
Forwarding to localhost:5432
Inspector: http://127.0.0.1:4040
Ready to receive connections
```

Четыре туннеля одновременно укладываются в лимит тарифа Base (5 туннелей и 5 поддоменов); на бесплатном тарифе (1 туннель, 0 поддоменов) так держать весь стек не получится — придётся выбрать один сервис для туннеля за раз.

### Как работает внутренняя коммуникация

Сервисы внутри Docker Compose общаются через внутреннюю DNS Docker — туннели для межсервисных вызовов не нужны. API обращается к базе данных по адресу `db:5432`, API обращается к Redis по адресу `redis:6379`, а API вызывает платёжный сервис по адресу `payment:4100`. Всё это происходит внутри Docker-сети без дополнительных затрат.

Туннели нужны только для трафика, приходящего извне Docker-сети: из браузеров, мобильных приложений, от вебхуков и коллег.

| Внешний трафик (через туннели) | Внутренний трафик (Docker DNS) |
|---|---|
| Браузер → туннель → frontend | frontend → api (`http://api:4000`) |
| Stripe → туннель → payment | api → db (`postgres://db:5432`) |
| Моб. приложение → туннель → api | api → redis (`redis://redis:6379`) |
| Коллега → туннель → api | api → payment (`http://payment:4100`) |

## Обнаружение сервисов: связь микросервисов через URL туннелей

Самая непростая часть работы с несколькими туннелями — связать сервисы друг с другом. Фронтенд должен знать публичный URL API для запросов из браузера. Платёжный сервис должен сообщить Stripe, куда отправлять вебхуки. Есть несколько подходов.

### Подход 1: фиксированные поддомены в конфиге (рекомендуется)

Если туннели заданы через `tunnels[]` в `client.yaml` с явным полем `subdomain` (как в примере выше), URL известен заранее — его не нужно вычитывать из логов. Просто пропишите его в переменные окружения сервисов напрямую:

```bash
# .env
REACT_APP_API_URL=https://myapp-api.fxtun.ru
PAYMENT_SERVICE_URL=https://myapp-pay.fxtun.ru
```

Такой подход требует зарезервированных поддоменов — они доступны с тарифа Base (5 поддоменов); на бесплатном тарифе поддомены закрепить нельзя.

### Подход 2: без резервирования — считать URL из вывода клиента

Если вы не резервируете поддомены и запускаете `fxtunnel http <port>` без `--domain`, адрес каждый раз новый — его нужно считать из вывода. Клиент пишет адрес и логи в stdout, поэтому вывод можно перенаправить в файл (`> file 2>&1`) и разобрать:

```bash
#!/bin/bash
# start-with-discovery.sh

fxtunnel http 4000 > /tmp/tunnel-api.log 2>&1 &
fxtunnel http 3000 > /tmp/tunnel-frontend.log 2>&1 &
fxtunnel http 4100 > /tmp/tunnel-payment.log 2>&1 &

sleep 3

API_URL=$(grep -oP 'HTTPS: \Khttps://[a-z0-9-]+\.fxtun\.ru' /tmp/tunnel-api.log | head -1)
FRONTEND_URL=$(grep -oP 'HTTPS: \Khttps://[a-z0-9-]+\.fxtun\.ru' /tmp/tunnel-frontend.log | head -1)
PAYMENT_URL=$(grep -oP 'HTTPS: \Khttps://[a-z0-9-]+\.fxtun\.ru' /tmp/tunnel-payment.log | head -1)

echo "API:      $API_URL"
echo "Frontend: $FRONTEND_URL"
echo "Payment:  $PAYMENT_URL"

echo "Укажите URL вебхука Stripe: $PAYMENT_URL/webhooks/stripe"
```

Три параллельных туннеля из этого скрипта требуют тариф Base и выше.

### Подход 3: собственный домен

Зарезервированный поддомен вида `myapp-api.fxtun.ru` — не то же самое, что домен вашей компании. Чтобы адрес выглядел как `api.myteam.com`, добавьте свой домен через `domains custom` (доступно с тарифа Base):

```bash
fxtunnel domains add myapp-api   # если поддомен ещё не зарезервирован
fxtunnel domains custom add api.myteam.com --target myapp-api
# в DNS: TXT _fxtunnel-challenge.api.myteam.com = fxtunnel-verify=<токен из личного кабинета>
#        CNAME api.myteam.com → myapp-api.fxtun.ru
fxtunnel domains custom verify api.myteam.com
```

Сам туннель по-прежнему запускается с поддоменом, который вы привязали к домену: `fxtunnel http 4000 --domain myapp-api`.

## Командная работа: общий доступ к микросервисному стеку

Когда несколько разработчиков работают над одной микросервисной архитектурой, каждому нужен свой набор туннелей. Вот практические паттерны для командной работы.

### Паттерн 1: каждый разработчик запускает свой стек

Самый простой подход: каждый разработчик запускает `docker compose up` и свой `fxtunnel --config client.yaml` на своей машине (с разными поддоменами в конфиге). Каждый получает уникальные URL туннелей. URL передаются через командный чат, когда нужен фидбэк. Для трёх туннелей на человека нужен тариф Base или выше на каждой учётной записи.

| Сервис | Разработчик A | Разработчик B |
|---|---|---|
| frontend | `https://a-front.fxtun.ru` | `https://b-front.fxtun.ru` |
| api | `https://a-api.fxtun.ru` | `https://b-api.fxtun.ru` |
| payment | `https://a-pay.fxtun.ru` | `https://b-pay.fxtun.ru` |

Это хорошо работает для небольших команд (2-5 разработчиков). Стек каждого разработчика полностью изолирован. Конфликтов нет.

### Паттерн 2: общий бэкенд, локальный фронтенд

Во многих проектах фронтенд меняется чаще бэкенда. Один разработчик запускает бэкенд с туннелем, а остальные направляют свои локальные фронтенды на общий API.

```bash
# Разработчик A (запускает бэкенд и туннель к нему)
docker compose up -d
fxtunnel http 4000 --domain shared-api
# API: https://shared-api.fxtun.ru

# Разработчик B (запускает только фронтенд)
REACT_APP_API_URL=https://shared-api.fxtun.ru npm run dev
```

Это снижает потребление ресурсов и позволяет не запускать весь стек на каждой машине.

### Паттерн 3: preview-окружения в CI/CD

Для ревью пул-реквестов комбинируйте туннели с CI/CD. Каждый PR получает свой набор туннелей через [GitHub Actions](/blog/cicd-tunnel-preview-environments/), и ревьюеры переходят по ссылке вместо того, чтобы выкачивать ветку локально.

## Отладка нескольких туннелей

Когда туннелей несколько, разбираться, какой сервис получил какой запрос, становится важнее. [Инспектор трафика](/blog/traffic-inspector-debug-requests/) fxTunnel (доступен с тарифа Base) записывает каждый HTTP-запрос через каждый туннель -- с заголовками, телом и таймингами. Replay позволяет повторить любой запрос одним кликом.

### Проверка статуса туннелей

`fxtunnel` умеет работать в фоновом режиме и отдавать статус отдельной командой — не нужно искать процесс вручную:

```bash
fxtunnel up --config client.yaml
fxtunnel status
fxtunnel down
```

### Проверка связности

Если конкретный сервис недоступен через туннель, сначала проверьте его напрямую на хосте — порт уже опубликован через `ports:` в docker-compose.yml:

```bash
curl -I http://localhost:4000/health
```

Если ответ есть на localhost, но не через туннель, — проблема в конфиге `fxtunnel` (неверный `local_port` или `subdomain`). Если ответа нет и на localhost, проблема в самом сервисе или в докер-сети.

## Продвинутый уровень: выборочные туннели

Вам не всегда нужны туннели ко всем сервисам сразу. Раз туннели не привязаны к Docker Compose, а запускаются отдельным процессом `fxtunnel`, выбор сводится к тому, какой конфиг-файл (или команду) запустить:

```bash
# Локальная разработка без туннелей — сервисы доступны только на localhost
docker compose up -d

# Туннель только к API (для тестирования вебхука)
fxtunnel http 4000 --domain myapp-api

# Все туннели сразу — по конфигу из раздела выше
fxtunnel --config client.yaml
```

Держите под рукой несколько конфиг-файлов (`client-api-only.yaml`, `client-all.yaml`) — так каждый разработчик поднимает ровно те туннели, которые нужны в моменте, не трогая docker-compose.yml.

## Тарифы: сколько туннелей можно запустить?

| Тариф | Одновременных туннелей | Поддоменов | Инспектор | UDP |
|---|---|---|---|---|
| **Free** | 1 | 0 | нет | нет |
| **Base** | 5 | 5 | есть | есть |
| **Pro** | 15 | 15 | есть | есть |
| **Business** | 50 | 50 | есть | есть |

Бесплатный тариф даёт один туннель одновременно — этого достаточно для единственного сервиса, но не для целого микросервисного стека. Чтобы держать открытыми несколько туннелей параллельно (как в примерах этой статьи), нужен тариф Base или выше; чем больше сервисов и разработчиков в команде, тем ближе к Pro или Business. Актуальные тарифы — на [странице тарифов](/pricing).

Бесплатные тарифы других сервисов туннелирования тоже ограничены — перед выбором сверяйтесь с их актуальными условиями. Полное сравнение инструментов -- в статье [ngrok vs Cloudflare vs fxTunnel](/blog/ngrok-vs-cloudflare-vs-fxtunnel/).

## Лучшие практики

### 1. Создавайте туннели только для того, что нуждается во внешнем доступе

Не создавайте туннели для баз данных, Redis или очередей сообщений, если только коллеге не нужен прямой доступ к ним. Внутренние сервисы общаются через Docker-сеть без дополнительных затрат.

### 2. Используйте скрипт или Makefile

Оберните запуск туннелей в скрипт или Makefile, чтобы каждый разработчик в команде использовал одинаковую настройку:

```makefile
# Makefile
.PHONY: dev dev-tunnels

dev:
	docker compose up -d

dev-tunnels: dev
	fxtunnel up --config client.yaml
	fxtunnel status
```

### 3. Закрывайте туннели после работы

Открытые туннели — это публичные URL, доступные любому, кто знает адрес. Для [безопасности](/blog/https-localhost-development/) закрывайте туннели после завершения сессии:

```bash
# Остановить туннели (фоновый режим)
fxtunnel down

# Остановить сервисы
docker compose down
```

### 4. Используйте .env-файлы для командной конфигурации

Храните конфигурацию, связанную с туннелями, в `.env`-файлах, которые исключены из системы контроля версий:

```bash
# .env (не коммитится в git)
STRIPE_WEBHOOK_SECRET=whsec_test_xxx
API_TUNNEL_URL=https://api-xyz.fxtun.ru
```

### 5. Документируйте настройку

Добавьте раздел в README проекта с описанием запуска туннелей. Будущие участники команды скажут вам спасибо.

