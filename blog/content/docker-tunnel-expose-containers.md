---
title: "Docker + туннель — контейнеры в интернет без портов"
date: 2026-02-19T14:00:00+03:00
draft: false
description: "Как открыть Docker-контейнер в интернет через fxTunnel: два подхода (на хосте и внутри контейнера), docker-compose примеры, HTTP и TCP туннели."
tags: [docker, tunneling, containers, fxTunnel, devops, developer-tools]
image: ""
rubrics: [guides]
need: tunnels
cover: ["container:8080", "fxtunnel http", "myapp.fxtun.ru"]
updated: 2026-09-27
---

## Проблема: Docker-контейнеры изолированы от внешнего мира

Контейнер запущен, на `localhost:8080` всё работает, и тут кто-то просит ссылку. Docker-контейнеры живут в изолированной сети — даже с `-p 8080:80` сервис доступен только на вашей машине. Чтобы показать проект коллеге, принять [вебхук](/blog/webhook-testing-with-tunnel/) от Stripe или протестировать мобильное приложение с реального устройства, нужен публичный URL. [Туннель](/blog/what-is-tunneling/) решает эту задачу за 30 секунд — без проброса портов на роутере, без статического IP, без деплоя.

Стандартный docker-маппинг портов выглядит так:

```bash
docker run -p 8080:80 my-web-app
```

Контейнер слушает порт 80, Docker пробрасывает его на порт 8080 хоста. Вы открываете `http://localhost:8080` — всё работает. Но этот адрес доступен только вам. Ни коллега, ни внешний сервис не смогут до него достучаться.

## Почему обычный проброс портов не помогает

Проброс портов на роутере звучит как решение, но на практике это сложно, небезопасно и зачастую невозможно. Провайдер может выдать серый IP (CGNAT), в корпоративной сети к роутеру вас не пустят, а открытый порт — это постоянная дыра. Подробнее о разнице подходов — в статье [«Как открыть localhost из интернета»](/blog/expose-localhost-to-internet/).

Типичные препятствия:

- **CGNAT / серый IP** — провайдер разделяет один публичный IP между десятками клиентов. Пробросить порт невозможно.
- **Корпоративная сеть** — настройками роутера управляет администратор, а не вы.
- **Динамический IP** — адрес меняется при каждом переподключении. Нужен DDNS, а это ещё одна точка отказа.
- **Безопасность** — открытый порт на роутере виден всему интернету. Без TLS и аутентификации это приглашение для атак.

Туннелю проброс портов не нужен: клиент сам устанавливает исходящее соединение с публичным сервером, и тот выдаёт [HTTPS-адрес](/blog/https-localhost-development/), ведущий на ваш контейнер. Входящие порты открывать не требуется, поэтому серый и динамический IP не мешают. В корпоративной сети запуск туннеля стоит согласовать с администратором.

## Решение 1: fxTunnel на хосте (самый простой способ)

Самый быстрый путь: fxTunnel прямо на хосте. Контейнер пробрасывает порт через `-p`, fxTunnel делает к нему туннель. Две команды — готово.

### Шаг 1. Установка fxTunnel

```bash
# Быстрая установка (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Проверяем
fxtunnel version
```

### Шаг 2. Запуск контейнера с пробросом порта

```bash
# Запускаем контейнер с маппингом порта на хост
docker run -d -p 8080:80 --name my-app nginx
```

Теперь nginx доступен на `localhost:8080`.

### Шаг 3. Создание туннеля

```bash
# Открываем туннель к порту 8080 на хосте
fxtunnel http 8080
```

Вы получите публичный URL:

```console
Connecting to fxtunnel server...
Tunnel established!
HTTP:  http://docker-demo.fxtun.ru
HTTPS: https://docker-demo.fxtun.ru
Forwarding to localhost:8080
Inspector: http://127.0.0.1:4040
```

Готово. `https://docker-demo.fxtun.ru` теперь ведёт на nginx внутри контейнера. Любой человек с этим URL может открыть ваше приложение в браузере.

### Когда использовать этот подход

- Быстрое тестирование и демонстрация — одна команда, минимум конфигурации.
- Локальная разработка — fxTunnel уже установлен на машине, не нужно менять `Dockerfile` или `docker-compose.yml`.
- Разовые задачи — показать проект коллеге, принять вебхук, протестировать интеграцию.

## Решение 2: fxTunnel внутри Docker-контейнера

Для воспроизводимых окружений, CI/CD пайплайнов и командной работы лучше добавить fxTunnel как отдельный сервис в docker-compose. Готового образа клиента для этого нет — `ghcr.io/mephistofox/fxtunnel` собирает **сервер** (`fxtunnel-server`), не клиент. Поэтому образ для клиента нужно собрать самим, а адрес соседнего контейнера передать через `local_addr` в конфиг-файле: позиционный аргумент `fxtunnel http <port>` принимает только номер локального порта и имя сервиса не резолвит.

### Свой образ клиента

```dockerfile
# tunnel/Dockerfile
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates && \
    curl -fsSL https://fxtun.ru/install.sh | bash && \
    rm -rf /var/lib/apt/lists/*
COPY client.yaml /etc/fxtunnel/client.yaml
ENTRYPOINT ["fxtunnel", "--config", "/etc/fxtunnel/client.yaml"]
```

```yaml
# tunnel/client.yaml
server:
  address: tunnel.fxtun.ru:443
  token: sk_fxtunnel_...   # из личного кабинета на fxtun.ru

tunnels:
  - type: http
    local_addr: web
    local_port: 80
```

### docker-compose.yml с fxTunnel

```yaml
version: "3.8"

services:
  web:
    image: nginx
    ports:
      - "8080:80"

  tunnel:
    build: ./tunnel
    depends_on:
      - web
```

Запуск:

```bash
docker compose up
```

`tunnel` и `web` — сервисы одного docker-compose файла, по умолчанию Compose кладёт их в общую сеть, и `local_addr: web` резолвится обычным Docker DNS. Туннель поднимается и останавливается вместе с остальными контейнерами, публичный URL появится в логах.

### Когда использовать этот подход

- Воспроизводимые окружения, CI/CD — конфигурация туннеля хранится в коде вместе с остальным стеком.
- Готового образа клиента нет, поэтому потребуется свой `Dockerfile` — для разовых задач быстрее Решение 1 (на хосте).

## Практические сценарии

### Сценарий 1: HTTP-туннель для веб-приложения

Допустим, вы собираете React/Vue/Next.js приложение в Docker и хотите показать результат коллеге или заказчику. В `tunnel/client.yaml` — тот же образ, только `local_addr` смотрит на нужный сервис:

```yaml
tunnels:
  - type: http
    local_addr: frontend
    local_port: 3000
```

```yaml
version: "3.8"

services:
  frontend:
    build: .
    ports:
      - "3000:3000"
    volumes:
      - .:/app
    environment:
      - NODE_ENV=development

  tunnel:
    build: ./tunnel
    depends_on:
      - frontend
```

```bash
docker compose up
# tunnel-1  | Tunnel established!
# tunnel-1  | HTTP:  http://xyz.fxtun.ru
# tunnel-1  | HTTPS: https://xyz.fxtun.ru
# tunnel-1  | Forwarding to localhost:3000
```

Отправьте ссылку `https://xyz.fxtun.ru` заказчику — он увидит ваше приложение без деплоя.

### Сценарий 2: TCP-туннель для базы данных

Хотите дать коллеге доступ к dev-базе PostgreSQL из Docker? [TCP-туннель](/blog/tcp-udp-tunneling-explained/) откроет порт базы через публичный адрес.

На хосте (если порт проброшен):

```bash
# PostgreSQL проброшен на хост: docker run -p 5432:5432 postgres
fxtunnel tcp 5432
```

В docker-compose — тот же образ клиента, только конфиг смотрит на `db`:

```yaml
# tunnel/client.yaml — tunnels:
tunnels:
  - type: tcp
    local_addr: db
    local_port: 5432
```

```yaml
version: "3.8"

services:
  db:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: devpass
      POSTGRES_DB: myapp
    ports:
      - "5432:5432"

  db-tunnel:
    build: ./tunnel
    depends_on:
      - db
```

```bash
docker compose up
# db-tunnel-1 | Tunnel established!
# db-tunnel-1 | TCP: fxtun.ru:18432
# db-tunnel-1 | Forwarding to localhost:5432
```

Коллега подключается к базе данных:

```bash
psql -h fxtun.ru -p 18432 -U postgres -d myapp
```

### Сценарий 3: несколько туннелей для микросервисов

Работаете с микросервисным стеком — фронтенд, API, база данных? Каждому сервису нужен свой туннель — а значит, свой образ с `client.yaml`, где `local_addr` смотрит на конкретный сервис.

```yaml
version: "3.8"

services:
  frontend:
    build: ./frontend
    ports:
      - "3000:3000"

  api:
    build: ./api
    ports:
      - "4000:4000"
    environment:
      - DATABASE_URL=postgres://postgres:devpass@db:5432/myapp

  db:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: devpass
      POSTGRES_DB: myapp

  tunnel-frontend:
    build: ./tunnel-frontend   # client.yaml: local_addr frontend, local_port 3000
    depends_on:
      - frontend

  tunnel-api:
    build: ./tunnel-api        # client.yaml: local_addr api, local_port 4000
    depends_on:
      - api

  tunnel-db:
    build: ./tunnel-db         # client.yaml: type tcp, local_addr db, local_port 5432
    depends_on:
      - db
```

```bash
docker compose up
# tunnel-frontend-1 | HTTP: http://front-xyz.fxtun.ru
# tunnel-frontend-1 | Forwarding to localhost:3000
# tunnel-api-1      | HTTP: http://api-xyz.fxtun.ru
# tunnel-api-1      | Forwarding to localhost:4000
# tunnel-db-1       | TCP: fxtun.ru:19432
# tunnel-db-1       | Forwarding to localhost:5432
```

Три публичных адреса, три сервиса — всё поднимается одним `docker compose up`.

## Docker Compose + fxTunnel: готовый рецепт

Ниже — шаблон docker-compose для типичного веб-приложения с API, базой данных и туннелями. У каждого туннеля — свой образ с `client.yaml`, где `local_addr` смотрит на нужный сервис. Скопируйте, измените имена сервисов и порты под свой проект.

```yaml
version: "3.8"

services:
  # === Ваше приложение ===
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://postgres:devpass@db:5432/myapp
    depends_on:
      - db

  db:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: devpass
      POSTGRES_DB: myapp
    volumes:
      - pgdata:/var/lib/postgresql/data

  # === Туннели ===
  tunnel-app:
    build: ./tunnel-app        # client.yaml: local_addr app, local_port 8080
    depends_on:
      - app
    restart: unless-stopped

  tunnel-db:
    build: ./tunnel-db         # client.yaml: type tcp, local_addr db, local_port 5432
    depends_on:
      - db
    restart: unless-stopped

volumes:
  pgdata:
```

Команды для работы:

```bash
# Запуск всего стека
docker compose up -d

# Просмотр логов туннелей (там будут публичные URL)
docker compose logs tunnel-app tunnel-db

# Остановка
docker compose down
```

## Советы по работе с Docker-сетями и туннелями

### Сервис на хост-машине, а не в другом контейнере

Если fxTunnel запущен внутри контейнера, а нужный сервис работает на хост-машине (не в Docker), используйте в `client.yaml` специальный адрес `host.docker.internal`:

```yaml
tunnels:
  - type: http
    local_addr: host.docker.internal
    local_port: 3000
```

```yaml
  tunnel:
    build: ./tunnel
    extra_hosts:
      - "host.docker.internal:host-gateway"
```

Директива `extra_hosts` нужна на Linux — на macOS и Windows `host.docker.internal` работает из коробки.

### Bridge-сеть по умолчанию

Docker Compose автоматически создаёт bridge-сеть для всех сервисов в файле. Контейнеры видят друг друга по имени сервиса — этим и пользуется `local_addr` в конфиге клиента. Если вы используете `docker run` без Compose, контейнеры находятся в сети `bridge` по умолчанию и обращаются друг к другу по IP-адресу, а не по имени. В этом случае проще запустить fxTunnel на хосте (решение 1).

### Несколько сетей

Если ваши контейнеры находятся в разных Docker-сетях, убедитесь, что контейнер с fxTunnel подключён к той же сети, что и целевой сервис:

```yaml
services:
  web:
    image: nginx
    networks:
      - frontend

  tunnel:
    build: ./tunnel   # client.yaml: local_addr web
    networks:
      - frontend

networks:
  frontend:
```

### Проверка доступности изнутри контейнера

Если туннель не подключается к целевому сервису, проверьте сетевую связность:

```bash
# Заходим в контейнер туннеля
docker compose exec tunnel sh

# Проверяем, доступен ли целевой сервис
wget -qO- http://web:80
```
