---
title: "Доступ к API для партнёра без деплоя — туннель"
date: 2026-03-13T14:00:00+03:00
draft: false
description: "Как поделиться локальным API с партнёрами через fxTunnel: безопасный доступ, аутентификация, лимиты запросов и реальные рабочие сценарии для интеграций."
tags: [api, sharing, partners, integration, fxTunnel, tunnel, developer-tools, collaboration]
image: ""
rubrics: [guides]
need: domain
cover: ["localhost:8080", "fxtunnel http", "api.fxtun.ru"]
updated: 2026-09-27
---

## Проблема: партнёру нужен доступ к API, но вы ещё не задеплоили

Вы разрабатываете API. Партнёрская команда, внешний подрядчик или фронтенд-разработчик клиента должны начать интеграцию с вашими эндпоинтами сегодня. Но ваш API живёт на `localhost:8080`. Staging-сервера нет, CI/CD-пайплайна нет, а деплоить ради интеграционного теста — явный перебор.

Традиционный подход выглядит так: поднять облачный инстанс, настроить reverse proxy, выпустить SSL-сертификаты, задеплоить код, настроить переменные окружения и молиться, чтобы ничего не сломалось. На это уходят часы — иногда дни, если инфраструктура не готова. И каждый раз, когда вы меняете строчку кода, нужно передеплоить.

Тем временем партнёр заблокирован. Он не может писать код интеграции без живого API для тестирования. Сроки проекта сдвигаются. Все теряют время.

В этом суть проблемы: поделиться локальным API с внешней стороной должно быть просто, но накладные расходы на деплой делают это болезненным.

## Решение: туннель открывает доступ к локальному API за 30 секунд

[Локальный туннель](/blog/what-is-tunneling/) создаёт публичный HTTPS URL, который указывает прямо на ваш localhost. Одна команда — и партнёр может вызывать ваш API из любой точки мира. Никакого деплоя, серверов и настройки DNS.

fxTunnel — SaaS-туннель, заточенный именно под такой сценарий. Одна команда -- `fxtunnel http 8080` -- и у вас публичный URL с TLS-шифрованием по умолчанию. Встроенный [Inspector](/blog/traffic-inspector-debug-requests/) (от тарифа Base) позволяет и вам, и партнёру видеть каждый запрос в реальном времени, что сильно ускоряет отладку интеграций.

Как это работает:

1. Вы запускаете API-сервер локально на порту 8080.
2. Вы запускаете `fxtunnel http 8080` в соседнем терминале.
3. fxTunnel устанавливает зашифрованное соединение с публичным relay-сервером.
4. Relay-сервер выдаёт публичный URL вида `https://partner-api.fxtun.ru`.
5. Партнёр отправляет запросы на этот URL — они перенаправляются на ваш `localhost:8080`.

Для партнёра это выглядит как обычный HTTPS API. Ему не нужно знать, что сервер работает на вашем ноутбуке. Для него всё ведёт себя как любой другой удалённый API.

## Пошаговый гайд: шаринг API с партнёром

### Шаг 1. Установите fxTunnel

```bash
# Быстрая установка (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Проверяем
fxtunnel version
```

Клиент открыт на [GitHub](https://github.com/mephistofox/fxtun.dev), сервер можно развернуть у себя.

### Шаг 2. Запустите локальный API

Убедитесь, что ваш API-сервер работает и слушает на нужном порту. Вот минимальный пример на Express:

```javascript
// api-server.js
const express = require('express');
const app = express();

app.use(express.json());

// Health check
app.get('/health', (req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// Каталог продуктов API
app.get('/api/v1/products', (req, res) => {
  res.json([
    { id: 1, name: 'Widget A', price: 29.99, stock: 150 },
    { id: 2, name: 'Widget B', price: 49.99, stock: 75 },
  ]);
});

app.get('/api/v1/products/:id', (req, res) => {
  res.json({ id: req.params.id, name: 'Widget A', price: 29.99, stock: 150 });
});

app.post('/api/v1/orders', (req, res) => {
  console.log('New order:', req.body);
  res.status(201).json({
    orderId: 'ord_' + Date.now(),
    status: 'created',
    items: req.body.items,
  });
});

app.listen(8080, () => {
  console.log('API server listening on port 8080');
});
```

```bash
node api-server.js
# -> API server listening on port 8080
```

### Шаг 3. Откройте туннель

```bash
fxtunnel http 8080
```

Результат:

```
Connecting to fxtunnel server...
Tunnel established!
HTTP:  http://partner-api.fxtun.ru
HTTPS: https://partner-api.fxtun.ru
Forwarding to localhost:8080
Inspector: http://127.0.0.1:4040
```

### Шаг 4. Поделитесь URL с партнёром

Отправьте `https://partner-api.fxtun.ru` партнёру. Он сразу может начать вызывать ваши эндпоинты:

```bash
# Партнёр тестирует ваш API
curl https://partner-api.fxtun.ru/health
# -> {"status":"ok","timestamp":"2026-03-13T11:00:00.000Z"}

curl https://partner-api.fxtun.ru/api/v1/products
# -> [{"id":1,"name":"Widget A","price":29.99,"stock":150}, ...]

curl -X POST https://partner-api.fxtun.ru/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{"items": [{"productId": 1, "quantity": 2}]}'
# -> {"orderId":"ord_1710320400000","status":"created","items":[...]}
```

Вот и всё. Партнёр получил полный доступ к вашему API. Вы видите его запросы в терминале и можете дебажить в реальном времени.

## Реальные сценарии шаринга API через туннель

Шаринг локального API через туннель — это не нишевый трюк, а рабочий паттерн, который встречается постоянно. Вот пять сценариев, где туннель убирает накладные расходы деплоя и снимает блокировку с партнёрских команд.

### Сценарий 1: Разработка B2B-интеграции

Вы разрабатываете API обработки платежей. Партнёрская компания должна интегрировать свой checkout-процесс с вашими эндпоинтами. Их фронтенд-команда должна протестировать полный цикл запрос-ответ: создание заказов, обработка платежей, обработка ошибок.

Без туннеля вы деплоите на staging и даёте им URL. Каждое изменение кода требует передеплоя. С туннелем:

```bash
# Ваш локальный API с платёжными эндпоинтами
fxtunnel http 8080
# -> https://payments-dev.fxtun.ru

# Фронтенд-разработчик партнёра обращается к вашим эндпоинтам напрямую
# Вы исправляете баг — изменение доступно мгновенно, без передеплоя
```

### Сценарий 2: Внешний подрядчик разрабатывает фронтенд

Вы наняли фрилансера-React-разработчика для создания дашборда на основе вашего API. Ему нужно подключиться к вашим эндпоинтам, чтобы строить UI, но ваш API ещё нигде не задеплоен.

```bash
# Ваш API на порту 8080
fxtunnel http 8080
# -> https://dashboard-api.fxtun.ru

# Подрядчик прописывает это в .env:
# REACT_APP_API_URL=https://dashboard-api.fxtun.ru
```

Подрядчик работает с вашим живым API. Вы итерируете над эндпоинтами вместе в реальном времени. Когда вы меняете формат ответа, подрядчик видит изменение мгновенно.

### Сценарий 3: Мобильная команда в другом офисе

В вашей компании мобильная команда находится в другом городе. Они разрабатывают iOS/Android-приложение, которое потребляет ваш API. Им нужно тестировать на реальных эндпоинтах, а не на моках. Этот сценарий мы разбираем в статье [тестирование мобильных приложений через туннель](/blog/mobile-app-testing-tunnel/).

```bash
# API-сервер работает локально
fxtunnel http 3000
# -> https://mobile-backend.fxtun.ru

# Мобильные разработчики настраивают приложение:
# API_BASE_URL = "https://mobile-backend.fxtun.ru"
```

### Сценарий 4: Клиент оценивает API перед подписанием контракта

Потенциальный клиент хочет оценить ваш API перед подписанием контракта. Он хочет отправить реальные запросы и увидеть ответы. Вы не хотите разворачивать staging-инстанс ради одной sales-демонстрации.

```bash
fxtunnel domains custom add api-demo.yourcompany.com --target partner-demo
# DNS-запись у регистратора, затем:
fxtunnel domains custom verify api-demo.yourcompany.com
fxtunnel http 8080 --domain partner-demo
# -> https://api-demo.yourcompany.com
```

С кастомным доменом (от тарифа Base) клиент видит профессиональный URL вместо случайного поддомена. Этот подход мы разбираем в статье [как показать локальный проект без деплоя](/blog/demo-localhost-without-deploy/).

### Сценарий 5: Тестирование webhook-коллбэков между двумя API

Система партнёра должна отправлять коллбэки (вебхуки) на ваш API при наступлении событий на их стороне. Это обратная сторона исходящих API-вызовов — вам нужно принимать запросы. Туннель справляется с этим естественным образом, потому что делает ваш localhost публично доступным.

```bash
fxtunnel http 8080
# -> https://callback-receiver.fxtun.ru

# Передайте этот URL партнёру для доставки вебхуков
# Они настраивают: callback_url = "https://callback-receiver.fxtun.ru/api/v1/callbacks"
```

В статье [тестирование вебхуков через туннель](/blog/webhook-testing-with-tunnel/) этот сценарий разобран подробнее.

## Безопасность шаринга API

Если вы делитесь API с партнёром, он становится доступен из интернета, и безопасность важна даже на этапе разработки. fxTunnel шифрует весь трафик через TLS, но защита на уровне приложения — ваша зона ответственности. Подробнее — в статье [как безопасно открыть localhost](/blog/expose-localhost-to-internet/).

### Аутентификация по API-ключу

Самый простой подход: требовать API-ключ в каждом запросе. Сгенерируйте ключ, поделитесь им с партнёром и отклоняйте запросы без него.

```javascript
// Мидлвара: аутентификация по API-ключу
const API_KEY = process.env.API_KEY || 'partner-test-key-2026';

function authenticateApiKey(req, res, next) {
  const key = req.headers['x-api-key'];
  if (!key || key !== API_KEY) {
    return res.status(401).json({ error: 'Invalid or missing API key' });
  }
  next();
}

// Применяем ко всем маршрутам /api
app.use('/api', authenticateApiKey);
```

Партнёр включает ключ в каждый запрос:

```bash
curl -H "X-API-Key: partner-test-key-2026" \
  https://partner-api.fxtun.ru/api/v1/products
```

### JWT Bearer Token аутентификация

Для более сложных сценариев используйте JWT-токены. Это позволяет встроить идентификацию партнёра, разрешения и срок действия прямо в токен.

```javascript
const jwt = require('jsonwebtoken');
const SECRET = process.env.JWT_SECRET || 'dev-secret-change-me';

// Генерация токена для партнёра
app.post('/auth/token', (req, res) => {
  const { partnerId, secret } = req.body;
  if (secret !== 'shared-secret') {
    return res.status(401).json({ error: 'Invalid credentials' });
  }
  const token = jwt.sign(
    { partnerId, role: 'partner', exp: Math.floor(Date.now() / 1000) + 86400 },
    SECRET
  );
  res.json({ token });
});

// Мидлвара проверки токена
function authenticateJWT(req, res, next) {
  const authHeader = req.headers.authorization;
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Missing Bearer token' });
  }
  try {
    const decoded = jwt.verify(authHeader.split(' ')[1], SECRET);
    req.partner = decoded;
    next();
  } catch (err) {
    return res.status(403).json({ error: 'Invalid or expired token' });
  }
}

app.use('/api', authenticateJWT);
```

### Ограничение частоты запросов (Rate Limiting)

Защитите локальную машину от случайной перегрузки. Партнёр, запустивший нагрузочный тест на ваш туннель, может уронить вашу dev-среду.

```javascript
const rateLimit = require('express-rate-limit');

const apiLimiter = rateLimit({
  windowMs: 60 * 1000, // 1 минута
  max: 100,            // 100 запросов в минуту на IP
  message: { error: 'Too many requests. Limit: 100/min.' },
});

app.use('/api', apiLimiter);
```

### Чеклист безопасности для шаринга API

| Мера | Зачем нужна | Реализация |
|---|---|---|
| Аутентификация по ключу или JWT | Предотвращает несанкционированный доступ | Мидлвара на всех маршрутах `/api` |
| Ограничение частоты запросов | Защищает от перегрузки | `express-rate-limit` или аналог |
| Только тестовые данные | Предотвращает утечку реальных данных | Seed-данные в БД |
| Закрывать туннель после сессии | Ограничивает окно экспозиции | `Ctrl+C` по завершении |
| HTTPS (автоматически) | Шифрует данные при передаче | Встроено в fxTunnel |
| Не хардкодить секреты | Предотвращает утечку учётных данных | `.env` + `.gitignore` |
| Защитить админские маршруты | Предотвращает повышение привилегий | Отдельная авторизация для admin-эндпоинтов |

## Inspector для отладки партнёрских запросов

Когда партнёр сообщает «ваш API возвращает ошибку», вам нужно увидеть, что именно он отправил. Встроенный [Traffic Inspector](/blog/traffic-inspector-debug-requests/) в fxTunnel (от тарифа Base) показывает каждый запрос и ответ в реальном времени.

```bash
fxtunnel http 8080
```

```
Connecting to fxtunnel server...
Tunnel established!
HTTP:  http://partner-api.fxtun.ru
HTTPS: https://partner-api.fxtun.ru
Forwarding to localhost:8080
Inspector: http://127.0.0.1:4040
```

Откройте адрес Inspector в браузере — он работает локально, только с вашей машины. Когда партнёр отправляет запрос, вы видите его мгновенно:

```
POST /api/v1/orders HTTP/1.1
X-API-Key: partner-test-key-2026
Content-Type: application/json

{"items": [{"product_id": "abc", "quantity": -1}]}

--- Response ---
HTTP/1.1 400 Bad Request
{"error": "Quantity must be a positive integer"}
```

Теперь вы точно знаете, что пошло не так. Партнёр отправил `product_id` как строку вместо числа, а `quantity` был отрицательным. Вы можете дать точную обратную связь вместо гадания.

Функция Replay позволяет переотправить точный запрос партнёра после исправления бага. Не нужно просить партнёра повторить — нажмите Replay, убедитесь в исправлении, а потом сообщите ему.

## Шаринг нескольких API одновременно

А что если партнёру нужен доступ к REST API, WebSocket-серверу и админ-панели фонового воркера — и каждый сервис на своём порту? В реальных интеграциях это обычное дело, и туннели справляются с этим естественным образом.

### Два туннеля, два сервиса

```bash
# Терминал 1 — REST API на порту 8080
fxtunnel http 8080
# -> https://rest-api.fxtun.ru

# Терминал 2 — WebSocket-сервер на порту 8081
fxtunnel http 8081
# -> https://ws-server.fxtun.ru
```

Дайте партнёру оба URL. REST-запросы идут на `https://rest-api.fxtun.ru`, WebSocket-соединения — на `https://ws-server.fxtun.ru`.

### Docker Compose + несколько туннелей

Если ваши сервисы работают в [Docker-контейнерах](/blog/docker-tunnel-expose-containers/), откройте порт каждого контейнера через отдельный туннель:

```yaml
# docker-compose.yml
services:
  api:
    build: ./api
    ports:
      - "8080:8080"
  auth:
    build: ./auth
    ports:
      - "8081:8081"
  gateway:
    build: ./gateway
    ports:
      - "3000:3000"
```

```bash
# Открываем туннели к каждому сервису
fxtunnel http 8080   # -> https://api.fxtun.ru
fxtunnel http 8081   # -> https://auth.fxtun.ru
fxtunnel http 3000   # -> https://gateway.fxtun.ru
```

На тарифе Pro доступно 15 туннелей одновременно — достаточно для сложных микросервисных архитектур.

## Сравнение: туннель vs другие способы шаринга API

Есть несколько способов дать партнёру доступ к вашему API. У каждого метода свои компромиссы по времени настройки, стоимости и актуальности тестируемого кода.

| Способ | Время настройки | Стоимость | Всегда актуально | HTTPS | Поддержка авторизации |
|---|---|---|---|---|---|
| **fxTunnel** | 30 секунд | Бесплатно (кастомный домен — платный тариф) | Да (live-код) | Из коробки | На стороне приложения |
| Деплой на staging | 30-120 минут | Платно (сервер + инфраструктура) | После каждого деплоя | Нужна настройка | Полный контроль |
| Деплой на Vercel/Railway | 5-15 минут | Бесплатно с лимитами | После каждого push | Да | Зависит от платформы |
| API-мок (Postman, Stoplight) | 1-4 часа | Бесплатно / платные тарифы | Нет (статический мок) | Да | Встроенная |
| VPN в вашу сеть | 30-60 минут | Стоимость VPN | Да | Зависит от настройки | На уровне сети |
| SSH-туннель | 10-30 минут | Нужен SSH-сервер | Да | Ручная настройка TLS | Нет встроенной |

Туннель выигрывает, когда нужен мгновенный доступ к живому, активно меняющемуся коду без накладных расходов на деплой. Подробное сравнение инструментов мы разбираем в [обзоре за 2026 год](/blog/best-tunneling-tools-2026/).

### Когда туннель — правильный выбор

- **Ранняя разработка** — API меняется несколько раз в день, деплоить после каждого изменения расточительно.
- **Короткие интеграционные спринты** — партнёру нужен доступ на дни или недели, а не на месяцы.
- **Proof of concept** — вы хотите, чтобы партнёр валидировал дизайн API до вложений в инфраструктуру.
- **Фриланс-проекты** — нет бюджета на выделенный staging-сервер.

### Когда туннеля недостаточно

- **Production-трафик** — туннель для разработки, а не для production. Для реальных пользователей деплойте API как положено.
- **Требования SLA** — если партнёр ожидает 99,9% аптайма, используйте полноценный деплой.
- **Большие команды с CI/CD** — если у вас уже есть staging-инфраструктура, используйте её.

## Примеры для разных фреймворков

### Python FastAPI

```python
# main.py
from fastapi import FastAPI, Header, HTTPException
from pydantic import BaseModel
from typing import Optional

app = FastAPI(title="Partner API", version="0.1.0")

API_KEY = "partner-test-key-2026"

async def verify_api_key(x_api_key: str = Header(...)):
    if x_api_key != API_KEY:
        raise HTTPException(status_code=401, detail="Invalid API key")

@app.get("/api/v1/products")
async def list_products(x_api_key: str = Header(...)):
    await verify_api_key(x_api_key)
    return [
        {"id": 1, "name": "Widget A", "price": 29.99},
        {"id": 2, "name": "Widget B", "price": 49.99},
    ]

class OrderRequest(BaseModel):
    product_id: int
    quantity: int

@app.post("/api/v1/orders", status_code=201)
async def create_order(order: OrderRequest, x_api_key: str = Header(...)):
    await verify_api_key(x_api_key)
    return {"order_id": f"ord_{order.product_id}_{order.quantity}", "status": "created"}
```

```bash
uvicorn main:app --reload --port 8080
fxtunnel http 8080
# -> https://fastapi-partner.fxtun.ru
# Партнёр открывает https://fastapi-partner.fxtun.ru/docs для интерактивного Swagger
```

Автоматический Swagger UI от FastAPI — убийственная фича для партнёрских интеграций. Партнёр получает интерактивную документацию API по адресу `https://your-tunnel.fxtun.ru/docs` без какой-либо дополнительной настройки.

### Go (net/http)

```go
// main.go
package main

import (
    "encoding/json"
    "log"
    "net/http"
)

const apiKey = "partner-test-key-2026"

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        key := r.Header.Get("X-API-Key")
        if key != apiKey {
            http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
            return
        }
        next(w, r)
    }
}

func productsHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode([]map[string]interface{}{
        {"id": 1, "name": "Widget A", "price": 29.99},
        {"id": 2, "name": "Widget B", "price": 49.99},
    })
}

func main() {
    http.HandleFunc("/api/v1/products", authMiddleware(productsHandler))
    log.Println("API server listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

```bash
go run main.go
fxtunnel http 8080
# -> https://go-api.fxtun.ru
```

### Ruby on Rails

```bash
rails server -p 8080
fxtunnel http 8080
# -> https://rails-api.fxtun.ru
```

Rails в API-режиме работает через туннель без проблем. Если партнёру нужно видеть структуру API, рассмотрите добавление `rswag` для Swagger-документации.

## Советы для плавных партнёрских интеграций

### Предоставляйте документацию API вместе с URL туннеля

Туннель даёт партнёру доступ. Документация рассказывает, что с ним делать. Поделитесь OpenAPI-спецификацией, Postman-коллекцией или хотя бы списком эндпоинтов с примерами запросов.

```bash
# Экспорт OpenAPI-спецификации, если ваш фреймворк поддерживает
# FastAPI: GET /openapi.json
# Express + swagger-jsdoc: GET /api-docs.json

curl https://partner-api.fxtun.ru/openapi.json > api-spec.json
# Поделитесь этим файлом с партнёром
```

### Используйте кастомный домен для профессионального вида

Случайный поддомен вида `https://a3f7b.fxtun.ru` работает отлично, но `https://api.yourcompany.com` вызывает больше доверия в B2B-контексте. Кастомные домены доступны от тарифа Base.

```bash
fxtunnel domains custom add api.yourcompany.com --target partner-api
fxtunnel domains custom verify api.yourcompany.com
fxtunnel http 8080 --domain partner-api
```

### Держите туннель запущенным в рабочие часы

Партнёр заблокирован, если туннель не работает. В период активной интеграции держите туннель запущенным в течение рабочего дня. Без `--domain` каждый перезапуск даёт новый случайный поддомен — если не хотите пересылать партнёру новый URL после каждого перерыва, зарезервируйте поддомен (`fxtunnel domains add`) и запускайте туннель с `--domain`.

### Сообщайте об ограничениях заранее

Расскажите партнёру заранее:
- Какая аутентификация требуется (API-ключ, JWT и т.д.)
- Какие лимиты на частоту запросов установлены (например, 100 запросов/минуту)
- Какие данные тестовые и на них нельзя полагаться
- В какие часы туннель будет доступен

### Используйте версионные префиксы в URL

Всегда используйте версионные пути API вида `/api/v1/...`. Так при внесении ломающих изменений вы сможете ввести `/api/v2/...` без нарушения существующей интеграции партнёра.

## Стоимость

Актуальные цены — на странице [тарифы](/pricing). Вот краткий обзор лимитов.

| Тариф | Что вы получаете |
|---|---|
| **Free** | 1 туннель, без UDP, без резерва поддомена, без кастомных доменов и Inspector |
| **Base** | 5 туннелей, UDP, 1 кастомный домен, Inspector + Replay |
| **Pro** | 15 туннелей, UDP, до 5 кастомных доменов |
| **Business** | 50 туннелей, UDP, до 50 кастомных доменов |

Для разового шаринга одного API бесплатного тарифа хватает. Переходите на Base, если нужно несколько туннелей одновременно, кастомный домен (профессиональный вид для B2B-интеграций) или Inspector (визуальная отладка партнёрских запросов). Тарифы Pro и Business актуальны, когда у вас несколько микросервисов, которым нужны туннели одновременно.

Если выбираете между решениями, мы разбираем [ngrok vs Cloudflare vs fxTunnel](/blog/ngrok-vs-cloudflare-vs-fxtunnel/) и [бесплатные альтернативы ngrok](/blog/free-ngrok-alternatives-2026/) в отдельных статьях.
