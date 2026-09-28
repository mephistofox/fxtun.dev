---
title: "Тестирование вебхуков через туннель — Stripe, GitHub"
date: 2026-02-03T14:00:00+03:00
draft: false
description: "Stripe и GitHub не могут достучаться до localhost. Показываем, как принимать вебхуки через fxTunnel, закреплять URL и отлаживать запросы в Inspector."
tags: [webhooks, testing, stripe, github, telegram, fxTunnel, localhost, tunnel, developer-tools]
image: ""
rubrics: [guides]
need: inspector
cover: ["Stripe", "fxtunnel http", "localhost:8080"]
updated: 2026-09-27
---

## Проблема: вебхуки требуют публичный URL

Знакомая ситуация: вы пишете интеграцию со Stripe, endpoint готов, и тут понимаете -- Stripe не может достучаться до вашего `localhost:8080`. [Локальный туннель](/blog/what-is-tunneling/) решает эту проблему за 30 секунд.

Вебхук -- это HTTP-запрос, который внешний сервис отправляет на ваш сервер при наступлении события. Stripe уведомляет об оплате, GitHub — о push, Telegram — о новом сообщении боту. Проблема в том, что все эти сервисы требуют публичный URL-адрес. Ваш `localhost:8080` недоступен из интернета, и настроить вебхук на `http://127.0.0.1` попросту невозможно.

Классические обходные пути — деплой на staging-сервер после каждого изменения или использование специальных сервисов для перехвата запросов — медленные и неудобные. Вы тратите минуты на деплой, чтобы проверить одну строку кода. А сервисы-перехватчики показывают только тело запроса, но не позволяют отлаживать логику обработки.

## Решение: локальный туннель для тестирования вебхуков

Локальный туннель создаёт публичный URL, который ведёт прямо на ваш localhost. Вебхук от Stripe летит на `https://abc.fxtun.ru`, [сервер туннеля](/blog/fxtunnel-architecture/) перенаправляет запрос на ваш `localhost:8080`, и ваше приложение получает данные в реальном времени. Цикл разработки сокращается до секунд.

Что это даёт разработчику:

- **Мгновенная обратная связь** — вы видите вебхуки сразу после их отправки, без задержки на деплой.
- **Inspector + Replay** — встроенный веб-интерфейс показывает все входящие запросы с заголовками и телом. Replay позволяет повторить любой вебхук одним кликом — не нужно заново тригерить событие в Stripe или GitHub. Доступно с тарифа Base.
- **Реальные данные** — тестируете с настоящими событиями от Stripe, GitHub или Telegram, а не с самодельными моками.
- **Отладка в IDE** — ставите брейкпоинты и изучаете payload прямо в отладчике. Это невозможно при тестировании на удалённом сервере.
- **Нет деплоя** — всё работает на вашей локальной машине. Поменяли код — сразу видите результат.
- **Безопасность** — тестовые данные не покидают вашу машину. Туннель шифрует трафик через TLS.

## Пошаговый гайд: тестирование вебхуков через fxTunnel

Настройка занимает 30 секунд: установите CLI, запустите туннель одной командой и укажите полученный публичный URL в настройках вебхука. Не нужно поднимать сервер, настраивать DNS или возиться с сертификатами.

### Шаг 1. Установка fxTunnel

```bash
# Быстрая установка (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Проверяем, что всё работает
fxtunnel version

# Входим в аккаунт fxtun.ru (один раз)
fxtunnel login
```

### Шаг 2. Запуск локального сервера

Убедитесь, что ваше приложение запущено и слушает нужный порт. Для примера возьмём простой Express-сервер:

```javascript
// webhook-server.js
const express = require('express');
const app = express();

app.use(express.json());

app.post('/webhook', (req, res) => {
  console.log('Получен вебхук:', JSON.stringify(req.body, null, 2));
  console.log('Заголовки:', JSON.stringify(req.headers, null, 2));
  res.status(200).json({ received: true });
});

app.listen(8080, () => {
  console.log('Сервер слушает порт 8080');
});
```

```bash
node webhook-server.js
# → Сервер слушает порт 8080
```

### Шаг 3. Создание туннеля

Без флага `--domain` каждый запуск `fxtunnel` выдаёт новый случайный поддомен. Для вебхуков это неудобно — сервис запоминает URL, который вы ему зарегистрировали, и при новом случайном адресе придётся перенастраивать endpoint заново. Зарезервируйте поддомен один раз (с тарифа Base) и всегда запускайте туннель с ним:

```bash
fxtunnel domains add wh-demo
fxtunnel http 8080 --domain wh-demo
```

Вы получите публичный URL:

```console
Connecting to fxtunnel server...
Tunnel established!
HTTP:  http://wh-demo.fxtun.ru
HTTPS: https://wh-demo.fxtun.ru
Forwarding to localhost:8080
Inspector: http://127.0.0.1:4040
```

Теперь `https://wh-demo.fxtun.ru/webhook` — это ваш endpoint для вебхуков, и адрес не меняется между перезапусками. Строка `Inspector:` в выводе появляется с тарифа Base — на бесплатном тарифе Inspector недоступен.

### Шаг 4. Настройка вебхука в сервисе

Укажите полный URL `https://wh-demo.fxtun.ru/webhook` в панели настроек нужного сервиса. Подробности для каждого сервиса — ниже.

## Примеры тестирования вебхуков для популярных сервисов

Вот готовые примеры для Stripe, GitHub и Telegram. Каждый включает настройку endpoint, проверку подписи и обработку событий. Во всех примерах используется один и тот же URL туннеля -- достаточно запустить `fxtunnel http 8080`.

### Stripe: тестирование платёжных вебхуков

Stripe активно использует вебхуки для уведомления о платежах, подписках и спорах. Без вебхуков вы не узнаете о статусе асинхронных операций — например, успешном платеже через 3D Secure или споре (dispute) по транзакции. При разработке платёжной интеграции тестирование вебхуков — обязательный этап.

#### Настройка через Dashboard

1. Откройте [Stripe Dashboard](https://dashboard.stripe.com/webhooks) в тестовом режиме.
2. Нажмите **Add endpoint**.
3. Укажите URL: `https://wh-demo.fxtun.ru/webhook`.
4. Выберите события: `payment_intent.succeeded`, `checkout.session.completed`.
5. Сохраните и скопируйте **Webhook Signing Secret** (`whsec_...`).

#### Проверка подписи на сервере

```javascript
// Проверка подписи Stripe webhook
const stripe = require('stripe')('sk_test_...');
const endpointSecret = 'whsec_...';

app.post('/webhook', express.raw({ type: 'application/json' }), (req, res) => {
  const sig = req.headers['stripe-signature'];

  try {
    const event = stripe.webhooks.constructEvent(req.body, sig, endpointSecret);
    console.log('Stripe событие:', event.type);

    // Обработка конкретных событий
    switch (event.type) {
      case 'payment_intent.succeeded':
        console.log('Платёж успешен:', event.data.object.id);
        break;
      case 'checkout.session.completed':
        console.log('Оформление завершено:', event.data.object.id);
        break;
    }

    res.status(200).json({ received: true });
  } catch (err) {
    console.error('Ошибка проверки подписи:', err.message);
    res.status(400).send(`Webhook Error: ${err.message}`);
  }
});
```

#### Отправка тестового события

```bash
# Через Stripe CLI (альтернативный способ тестирования)
stripe trigger payment_intent.succeeded \
  --override payment_intent:metadata.order_id=test_123
```

### GitHub: вебхук для push events

#### Настройка

1. Откройте репозиторий на GitHub → **Settings** → **Webhooks**.
2. Нажмите **Add webhook**.
3. Payload URL: `https://wh-demo.fxtun.ru/webhook`.
4. Content type: `application/json`.
5. Secret: придумайте секрет для проверки подписи.
6. Выберите событие: **Just the push event**.

#### Обработка на сервере

```javascript
const crypto = require('crypto');

const GITHUB_SECRET = 'your_webhook_secret';

app.post('/webhook', (req, res) => {
  // Проверка подписи GitHub
  const signature = req.headers['x-hub-signature-256'];
  const hmac = crypto.createHmac('sha256', GITHUB_SECRET);
  const digest = 'sha256=' + hmac.update(JSON.stringify(req.body)).digest('hex');

  if (signature !== digest) {
    return res.status(401).send('Invalid signature');
  }

  const event = req.headers['x-github-event'];
  console.log(`GitHub событие: ${event}`);

  if (event === 'push') {
    const { ref, commits, pusher } = req.body;
    console.log(`Push в ${ref} от ${pusher.name}`);
    console.log(`Коммитов: ${commits.length}`);
    commits.forEach(c => console.log(`  - ${c.message}`));
  }

  res.status(200).json({ ok: true });
});
```

### Telegram Bot: setWebhook

Telegram-боты могут работать через polling (бот периодически спрашивает сервер «есть новые сообщения?») или через вебхуки (Telegram сам присылает сообщение на ваш URL). Вебхуки быстрее, эффективнее и потребляют меньше ресурсов. Для разработки бота с вебхуками вам нужен публичный HTTPS-адрес — именно то, что даёт туннель.

#### Регистрация вебхука

```bash
# Устанавливаем вебхук через Telegram Bot API
curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://wh-demo.fxtun.ru/webhook"}'

# Проверяем статус
curl "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getWebhookInfo"
```

#### Обработка сообщений

```javascript
app.post('/webhook', (req, res) => {
  const { message } = req.body;

  if (message && message.text) {
    console.log(`Сообщение от ${message.from.first_name}: ${message.text}`);

    // Ответ пользователю
    const chatId = message.chat.id;
    const replyText = `Вы написали: ${message.text}`;

    fetch(`https://api.telegram.org/bot${BOT_TOKEN}/sendMessage`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ chat_id: chatId, text: replyText }),
    });
  }

  res.status(200).json({ ok: true });
});
```

#### Не забудьте удалить вебхук после тестирования

```bash
# Удаляем вебхук, когда закончили тестирование
curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/deleteWebhook"
```

## Troubleshooting: проблемы при тестировании вебхуков через туннель

Чаще всего проблемы с вебхуками сводятся к трём вещам: туннель не запущен, локальный сервер не слушает порт или неверно указан URL. Inspector в fxTunnel (с тарифа Base) покажет, дошёл ли запрос до туннеля, и сразу сузит область поиска.

| Проблема | Возможная причина | Решение |
|---|---|---|
| Вебхук не приходит | Туннель не запущен | Убедитесь, что `fxtunnel` работает и URL доступен в браузере |
| HTTP 502 Bad Gateway | Локальный сервер не запущен | Запустите приложение на нужном порту до создания туннеля |
| HTTP 401 Unauthorized | Неверная подпись вебхука | Проверьте, что секрет совпадает в настройках сервиса и в коде |
| Тайм-аут запроса | Обработчик слишком долгий | Отвечайте `200 OK` сразу, обрабатывайте асинхронно |
| Данные приходят пустые | Неверный Content-Type | Убедитесь, что `express.json()` подключён и путь совпадает |
| URL изменился после перезапуска | Туннель запущен без `--domain` — поддомен каждый раз случайный | Зарезервируйте поддомен (`fxtunnel domains add wh-demo`) и запускайте с `--domain wh-demo` |
| Не видно, какие данные пришли | Нет инструмента инспекции | Откройте Inspector в fxTunnel — он показывает все запросы с заголовками и телом |
| Нужно повторить запрос | Событие сложно воспроизвести | Используйте Replay в fxTunnel — повторная отправка любого запроса одним кликом |

## Лучшие практики тестирования вебхуков на localhost

Правильный подход к тестированию вебхуков экономит часы отладки. Вот пять принципов, которые стоит соблюдать с первого дня.

### 1. Всегда проверяйте подпись

Не пропускайте верификацию подписи даже при разработке. Это поможет выявить ошибки конфигурации до деплоя.

### 2. Логируйте всё (или используйте Inspector)

Вместо ручного логирования можно использовать Inspector в fxTunnel — он показывает все входящие запросы в веб-интерфейсе с заголовками, телом и статусом ответа. Но для серверного лога полезно и явное логирование:

```javascript
// Логируйте полный запрос для отладки
app.post('/webhook', (req, res) => {
  console.log('--- Входящий вебхук ---');
  console.log('Метод:', req.method);
  console.log('Путь:', req.path);
  console.log('Заголовки:', JSON.stringify(req.headers, null, 2));
  console.log('Тело:', JSON.stringify(req.body, null, 2));
  console.log('--- Конец ---');
  res.status(200).json({ received: true });
});
```

### 3. Отвечайте быстро

Большинство сервисов ожидают ответ в течение 5–30 секунд. Если обработка занимает больше времени — отвечайте `200 OK` сразу, а задачу ставьте в очередь.

### 4. Обрабатывайте повторные отправки

Сервисы повторно отправляют вебхуки, если не получили ответ `2xx`. Реализуйте идемпотентность: проверяйте уникальный ID события перед обработкой.

### 5. Используйте тестовый режим

Stripe, GitHub и другие сервисы имеют тестовый режим или возможность отправить тестовое событие. Используйте это перед настройкой реальных данных.

## Почему fxTunnel -- лучший инструмент для тестирования вебхуков

Как fxTunnel выглядит на фоне других инструментов именно для тестирования вебхуков? Подробное сравнение -- в статье [ngrok vs Cloudflare vs fxTunnel](/blog/ngrok-vs-cloudflare-vs-fxtunnel/).

| Возможность | fxTunnel | Другие инструменты |
|---|---|---|
| Бесплатный тариф | 1 туннель, без резерва поддомена | Часто ограничен по времени или запросам |
| Inspector + Replay | С тарифа Base — просмотр всех запросов и повтор одним кликом | Отсутствует или требует отдельного инструмента |
| Настройка | Одна команда, 30 секунд | Часто требует конфигурации, токенов, регистрации |
| HTTP / TCP / UDP | Все три протокола (UDP — с платных тарифов) | Обычно только HTTP |
| Кастомный домен (свой DNS) | С тарифа Base | Дороже или недоступно |
| Больше туннелей сразу | Pro — 15, Business — 50 | Значительно дороже |
| Открытость | Клиент открыт (GitHub), сервер можно развернуть у себя | Часто полностью проприетарный |

Полный обзор инструментов -- в статье [«Инструменты туннелирования 2026»](/blog/best-tunneling-tools-2026/).

