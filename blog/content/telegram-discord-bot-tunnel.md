---
title: "Telegram и Discord боты через туннель — гайд"
date: 2026-03-19T14:00:00+03:00
draft: false
description: "Telegram и Discord доставляют события только на публичный HTTPS-адрес. Показываем, как дать боту такой адрес через fxTunnel и отлаживать вебхуки в Inspector."
tags: [telegram, discord, bots, webhooks, fxTunnel, localhost, tunnel, python, nodejs, developer-tools]
image: ""
rubrics: [guides]
need: domain
cover: ["localhost:8080", "fxtunnel http", "setWebhook"]
updated: 2026-09-27
---

## Проблема: вебхуки бота требуют публичный URL

Допустим, вы пишете Telegram- или Discord-бота и хотите использовать вебхуки вместо polling. Загвоздка в том, что ваш локальный сервер должен быть доступен из интернета. [Туннель](/blog/what-is-tunneling/) решает это за считанные секунды: вы получаете публичный HTTPS URL, и Telegram с Discord отправляют обновления прямо на вашу машину — без деплоя, без VPS, без cloud functions.

И Telegram, и Discord предлагают модель на основе вебхуков для получения событий. В Telegram это `setWebhook` — вместо того чтобы опрашивать API каждые несколько секунд, Telegram сам отправляет каждое новое сообщение на ваш HTTPS-адрес. В Discord используется Interactions Endpoint — когда пользователь вызывает slash-команду, Discord отправляет POST-запрос на ваш URL. В обоих случаях платформе нужен публичный HTTPS-адрес.

Проблема очевидна: ваш `localhost:8080` недоступен из интернета. Указать `http://127.0.0.1` в `setWebhook` невозможно. Классические обходные пути — деплой на облачный сервер после каждого изменения кода или использование long-polling — медленные и ограничивают рабочий процесс разработки. Вы теряете возможность ставить брейкпоинты, изучать payload в реальном времени и быстро итерировать.

## Решение: локальный туннель для разработки ботов

Одна команда — `fxtunnel http 8080` — и у вас есть публичный HTTPS URL, ведущий на ваш localhost. Этот URL сразу работает как endpoint для Telegram `setWebhook` или Discord Interactions.

Вот как выглядит рабочий процесс с туннелем:

1. Ваш сервер бота работает локально на `localhost:8080`.
2. fxTunnel создаёт публичный URL, например `https://bot-dev.fxtun.ru`.
3. Вы регистрируете `https://bot-dev.fxtun.ru/webhook` как URL вебхука в Telegram или Discord.
4. Когда пользователь отправляет сообщение или вызывает команду, платформа отправляет POST-запрос на URL туннеля.
5. [Сервер туннеля](/blog/fxtunnel-architecture/) перенаправляет запрос на ваш `localhost:8080`.
6. Ваш бот обрабатывает событие и отвечает.

Что это даёт разработчику:

- **Мгновенная обратная связь** — сообщения и команды бота приходят сразу, без задержки на деплой.
- **Отладка в IDE** — ставите брейкпоинты и изучаете payload прямо в отладчике. Это невозможно при работе на удалённом сервере.
- **Inspector + Replay** — встроенный веб-интерфейс показывает все входящие запросы с заголовками и телом. Replay позволяет повторить любой вебхук одним кликом -- не нужно заново писать сообщение боту. Подробно об этом в статье про [Traffic Inspector](/blog/traffic-inspector-debug-requests/).
- **Нет деплоя** — всё работает на вашей локальной машине. Поменяли код — сразу видите результат.
- **Стабильный URL** — если поддомен зарезервирован (`fxtunnel domains add`, с тарифа Base) и указан флагом `--domain`, адрес не меняется между перезапусками. Не нужно вызывать `setWebhook` заново.

## Быстрый старт: установка fxTunnel и создание туннеля

Установите CLI, запустите туннель — и у вас есть публичный URL для бота.

### Шаг 1. Установка fxTunnel

```bash
# Быстрая установка (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Проверяем, что всё работает
fxtunnel version
```

### Шаг 2. Запуск туннеля

Без `--domain` каждый запуск `fxtunnel` выдаёт новый случайный поддомен, а Telegram и Discord запоминают ровно тот URL, что вы им зарегистрировали. Чтобы не перерегистрировать вебхук при каждом рестарте, зарезервируйте поддомен один раз (с тарифа Base) и запускайте туннель с ним:

```bash
fxtunnel domains add bot-dev
fxtunnel http 8080 --domain bot-dev
```

Вы получите публичный URL:

```
Connecting to fxtunnel server...
Tunnel established!
HTTP:  http://bot-dev.fxtun.ru
HTTPS: https://bot-dev.fxtun.ru
Forwarding to localhost:8080
```

Теперь `https://bot-dev.fxtun.ru` — это ваш endpoint для вебхуков, и он не меняется от запуска к запуску. Не закрывайте этот терминал — туннель работает, пока процесс запущен.

## Telegram-бот: разработка на вебхуках с Python

Telegram-боты могут работать в двух режимах: polling (бот периодически запрашивает обновления у API) и webhooks (Telegram сам отправляет обновления на ваш URL). Вебхуки быстрее, эффективнее и рекомендуются для продакшен-ботов. Во время разработки туннель позволяет использовать вебхуки с первого дня — код работает одинаково в разработке и в продакшене.

### Установка вебхука через curl

```bash
# Регистрируем URL вебхука в Telegram
curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://bot-dev.fxtun.ru/webhook"}'

# Проверяем, что вебхук установлен
curl "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getWebhookInfo"
```

Ответ `getWebhookInfo` должен показать `"url": "https://bot-dev.fxtun.ru/webhook"` и `"has_custom_certificate": false` — fxTunnel обеспечивает TLS автоматически.

### Python-бот с python-telegram-bot

Вот полноценный Telegram-бот на библиотеке `python-telegram-bot` в режиме webhook:

```python
# bot_telegram.py
import os
from telegram import Update
from telegram.ext import Application, CommandHandler, MessageHandler, filters, ContextTypes

BOT_TOKEN = os.environ["TELEGRAM_BOT_TOKEN"]
WEBHOOK_URL = "https://bot-dev.fxtun.ru/webhook"

async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text(
        "Привет! Я работаю на localhost через вебхук fxTunnel."
    )

async def echo(update: Update, context: ContextTypes.DEFAULT_TYPE):
    text = update.message.text
    await update.message.reply_text(f"Вы написали: {text}")

async def handle_error(update: Update, context: ContextTypes.DEFAULT_TYPE):
    print(f"Ошибка: {context.error}")

def main():
    app = Application.builder().token(BOT_TOKEN).build()

    app.add_handler(CommandHandler("start", start))
    app.add_handler(MessageHandler(filters.TEXT & ~filters.COMMAND, echo))
    app.add_error_handler(handle_error)

    # Запуск в режиме webhook на порту 8080
    app.run_webhook(
        listen="0.0.0.0",
        port=8080,
        url_path="/webhook",
        webhook_url=WEBHOOK_URL,
    )

if __name__ == "__main__":
    main()
```

Запуск бота:

```bash
# Установка зависимостей
pip install python-telegram-bot

# Задаём токен бота
export TELEGRAM_BOT_TOKEN="123456:ABC-DEF..."

# Запускаем бота
python bot_telegram.py
```

Метод `run_webhook` делает две вещи: запускает HTTP-сервер на порту 8080 и автоматически вызывает `setWebhook`. В связке с туннелем fxTunnel Telegram отправляет обновления прямо на вашу машину.

### Node.js Telegram-бот с Express

Если вы предпочитаете Node.js, вот тот же бот с использованием пакета `node-telegram-bot-api`:

```javascript
// bot_telegram.js
const TelegramBot = require('node-telegram-bot-api');
const express = require('express');

const BOT_TOKEN = process.env.TELEGRAM_BOT_TOKEN;
const WEBHOOK_URL = 'https://bot-dev.fxtun.ru/webhook';

const bot = new TelegramBot(BOT_TOKEN);
const app = express();

app.use(express.json());

// Обработка входящих обновлений от Telegram
app.post('/webhook', (req, res) => {
  bot.processUpdate(req.body);
  res.status(200).json({ ok: true });
});

// Обработчики команд
bot.onText(/\/start/, (msg) => {
  bot.sendMessage(msg.chat.id, 'Привет! Я работаю на localhost через вебхук fxTunnel.');
});

bot.on('message', (msg) => {
  if (msg.text && !msg.text.startsWith('/')) {
    bot.sendMessage(msg.chat.id, `Вы написали: ${msg.text}`);
  }
});

// Запуск сервера и установка вебхука
app.listen(8080, async () => {
  console.log('Сервер бота слушает порт 8080');
  await bot.setWebHook(WEBHOOK_URL);
  console.log(`Вебхук установлен: ${WEBHOOK_URL}`);
});
```

```bash
npm install node-telegram-bot-api express

export TELEGRAM_BOT_TOKEN="123456:ABC-DEF..."
node bot_telegram.js
```

### Очистка после разработки

Когда вы заканчиваете сессию разработки, удалите вебхук, чтобы Telegram не продолжал отправлять запросы на неактивный URL:

```bash
# Удаляем вебхук
curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/deleteWebhook"

# Проверяем, что вебхук удалён
curl "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getWebhookInfo"
```

## Discord-бот: Interactions Endpoint на localhost

Discord поддерживает две модели получения команд: Gateway (постоянное WebSocket-соединение) и Interactions Endpoint (HTTP-вебхуки). Interactions Endpoint идеально подходит для slash-команд и компонентов сообщений — Discord отправляет POST-запрос на ваш URL, а сервер отвечает синхронно. Как и Telegram, Discord требует публичный HTTPS-адрес с корректной проверкой подписи.

### Как работают Discord Interactions

Когда пользователь вызывает slash-команду в Discord:

1. Discord отправляет POST-запрос на ваш Interactions Endpoint URL.
2. Тело запроса содержит данные взаимодействия (имя команды, параметры, информация о пользователе).
3. Ваш сервер проверяет Ed25519-подпись с помощью публичного ключа из настроек Discord-приложения.
4. Ваш сервер отвечает соответствующим ответом (сообщение, отложенный ответ, модальное окно и т.д.).

Discord также отправляет `PING`-взаимодействие (type 1), когда вы сохраняете Interactions Endpoint URL в Developer Portal. Ваш сервер должен ответить `{"type": 1}`, чтобы пройти проверку.

### Python Discord-бот с discord.py и Flask

```python
# bot_discord.py
import os
from flask import Flask, request, jsonify
from nacl.signing import VerifyKey
from nacl.exceptions import BadSignatureError

DISCORD_PUBLIC_KEY = os.environ["DISCORD_PUBLIC_KEY"]
DISCORD_BOT_TOKEN = os.environ["DISCORD_BOT_TOKEN"]

app = Flask(__name__)
verify_key = VerifyKey(bytes.fromhex(DISCORD_PUBLIC_KEY))

def verify_signature(req):
    """Проверка Ed25519-подписи Discord."""
    signature = req.headers.get("X-Signature-Ed25519")
    timestamp = req.headers.get("X-Signature-Timestamp")
    body = req.data.decode("utf-8")

    try:
        verify_key.verify(f"{timestamp}{body}".encode(), bytes.fromhex(signature))
        return True
    except (BadSignatureError, Exception):
        return False

@app.route("/discord", methods=["POST"])
def interactions():
    if not verify_signature(request):
        return "Invalid signature", 401

    data = request.json

    # Обработка PING (type 1) — обязательна для верификации endpoint
    if data["type"] == 1:
        return jsonify({"type": 1})

    # Обработка slash-команд (type 2)
    if data["type"] == 2:
        command_name = data["data"]["name"]

        if command_name == "hello":
            return jsonify({
                "type": 4,  # CHANNEL_MESSAGE_WITH_SOURCE
                "data": {
                    "content": f"Привет, {data['member']['user']['username']}! "
                               f"Этот бот работает на localhost через fxTunnel."
                }
            })

        if command_name == "ping":
            return jsonify({
                "type": 4,
                "data": {"content": "Pong! Ответ с localhost."}
            })

    return jsonify({"type": 1})

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
```

```bash
pip install flask pynacl

export DISCORD_PUBLIC_KEY="your_public_key_hex"
export DISCORD_BOT_TOKEN="your_bot_token"
python bot_discord.py
```

### Node.js Discord-бот с Express

```javascript
// bot_discord.js
const express = require('express');
const { verifyKey } = require('discord-interactions');

const DISCORD_PUBLIC_KEY = process.env.DISCORD_PUBLIC_KEY;

const app = express();

// Discord требует raw body для проверки подписи
app.use(express.raw({ type: 'application/json' }));

app.post('/discord', (req, res) => {
  const signature = req.headers['x-signature-ed25519'];
  const timestamp = req.headers['x-signature-timestamp'];

  const isValid = verifyKey(req.body, signature, timestamp, DISCORD_PUBLIC_KEY);
  if (!isValid) {
    return res.status(401).send('Invalid signature');
  }

  const data = JSON.parse(req.body.toString());

  // Обработка PING
  if (data.type === 1) {
    return res.json({ type: 1 });
  }

  // Обработка slash-команд
  if (data.type === 2) {
    const commandName = data.data.name;

    if (commandName === 'hello') {
      return res.json({
        type: 4,
        data: {
          content: `Привет, ${data.member.user.username}! Этот бот работает на localhost через fxTunnel.`,
        },
      });
    }

    if (commandName === 'ping') {
      return res.json({
        type: 4,
        data: { content: 'Pong! Ответ с localhost.' },
      });
    }
  }

  res.json({ type: 1 });
});

app.listen(8080, () => {
  console.log('Сервер Discord-бота слушает порт 8080');
});
```

```bash
npm install express discord-interactions

export DISCORD_PUBLIC_KEY="your_public_key_hex"
node bot_discord.js
```

### Настройка Interactions Endpoint в Discord Developer Portal

1. Перейдите в [Discord Developer Portal](https://discord.com/developers/applications).
2. Выберите ваше приложение.
3. Откройте раздел **General Information**.
4. В поле **Interactions Endpoint URL** введите: `https://bot-dev.fxtun.ru/discord`.
5. Нажмите **Save Changes**.

Discord отправит PING-запрос для проверки endpoint. Если ваш сервер работает за туннелем fxTunnel и правильно отвечает, URL будет сохранён. Если проверка не прошла — убедитесь, что сервер запущен, туннель активен, а код проверки подписи корректен.

### Регистрация slash-команд

Прежде чем пользователи смогут вызывать ваши команды, их нужно зарегистрировать в Discord:

```bash
# Регистрация глобальной slash-команды
curl -X POST "https://discord.com/api/v10/applications/<APP_ID>/commands" \
  -H "Authorization: Bot <BOT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "hello",
    "description": "Say hello from the bot",
    "type": 1
  }'

curl -X POST "https://discord.com/api/v10/applications/<APP_ID>/commands" \
  -H "Authorization: Bot <BOT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ping",
    "description": "Check bot responsiveness",
    "type": 1
  }'
```

Глобальные команды могут появляться до часа. Для быстрого тестирования регистрируйте команды для конкретного сервера, добавив `/guilds/<GUILD_ID>` в URL — они появятся мгновенно.

## Отладка вебхуков бота через Inspector

Бот молчит, а почему -- непонятно? Inspector в fxTunnel (с тарифа Base) показывает каждый входящий запрос в реальном времени -- заголовки, тело, статус-код и время обработки. Вы видите, что именно отправила платформа и что вернул ваш сервер.

### Типичный сценарий отладки

1. Пользователь отправляет `/start` вашему Telegram-боту.
2. Бот не отвечает.
3. Откройте Inspector на своей машине: `http://127.0.0.1:4040`.
4. Вы видите входящий POST-запрос с полным Telegram Update payload.
5. В ответе отображается `500 Internal Server Error` с текстом ошибки.
6. Вы исправляете баг, нажимаете **Replay**, чтобы повторно отправить тот же самый запрос.
7. Ответ теперь показывает `200 OK`. Проблема решена.

Без Inspector вам пришлось бы отправить ещё одно сообщение боту, дождаться доставки от Telegram и надеяться воспроизвести те же условия. С Replay вы тестируете точно тот же payload столько раз, сколько нужно.

### Что Inspector показывает для вебхуков бота

Для Telegram-обновления:

```
POST /webhook HTTP/1.1
Content-Type: application/json

{
  "update_id": 123456789,
  "message": {
    "message_id": 42,
    "from": {"id": 100, "first_name": "Алексей"},
    "chat": {"id": 100, "type": "private"},
    "text": "/start"
  }
}

--- Response ---
HTTP/1.1 200 OK
{"ok": true}
```

Для Discord-взаимодействия:

```
POST /discord HTTP/1.1
Content-Type: application/json
X-Signature-Ed25519: abc123...
X-Signature-Timestamp: 1711022400

{
  "type": 2,
  "data": {"name": "hello"},
  "member": {"user": {"username": "Алексей"}}
}

--- Response ---
HTTP/1.1 200 OK
{"type": 4, "data": {"content": "Привет, Алексей! ..."}}
```

Вы видите ровно то, что получает ваш сервер и что он возвращает -- без догадок, без ручного логирования. Все возможности Inspector разобраны в статье про [Traffic Inspector](/blog/traffic-inspector-debug-requests/).

## Сравнение: polling vs webhooks для разработки ботов

И Telegram, и Discord предлагают альтернативу вебхукам. Понимание компромиссов поможет выбрать правильный подход для вашего проекта.

| Аспект | Polling / Gateway | Webhooks + туннель |
|---|---|---|
| **Сложность настройки** | Минимальная — публичный URL не нужен | Требуется туннель или публичный сервер |
| **Задержка** | Выше — периодические запросы к API или задержки переподключения | Ниже — мгновенная доставка от платформы |
| **Потребление ресурсов** | Выше — постоянное соединение или повторяющиеся запросы | Ниже — сервер обрабатывает только реальные события |
| **Отладка** | Ограничена — нужно добавлять логирование вручную | Inspector показывает каждый запрос в реальном времени |
| **Готовность к продакшену** | Polling не рекомендуется для продакшен Telegram-ботов | Код на вебхуках работает одинаково в разработке и продакшене |
| **Брейкпоинты в IDE** | Работают локально | Работают локально с туннелем |
| **Replay** | Нужно заново тригерить событие вручную | Повтор одним кликом в Inspector |

Для разработки вебхуки с туннелем дают максимально близкое к продакшену поведение плюс лучшие инструменты отладки. Код, написанный во время разработки, деплоится в продакшен без изменений — нужно лишь заменить URL туннеля на ваш продакшен-домен.

## Одновременный запуск обоих ботов

Если вы разрабатываете одновременно Telegram и Discord бота (или любую комбинацию сервисов), можно запустить несколько туннелей одновременно — но на бесплатном тарифе (1 туннель) два туннеля сразу не поднять, нужен тариф Base и выше. Вариант 2 ниже работает и на бесплатном тарифе — там всего один туннель.

### Вариант 1: разные порты, отдельные туннели (тариф Base и выше)

```bash
# Терминал 1: Telegram-бот на порту 8080
python bot_telegram.py  # слушает 8080
fxtunnel http 8080

# Терминал 2: Discord-бот на порту 8090
python bot_discord.py   # слушает 8090
fxtunnel http 8090
```

### Вариант 2: один сервер, разные маршруты

Запустите оба бота на одном сервере с разными URL-путями:

```python
# bot_combined.py
from flask import Flask, request, jsonify

app = Flask(__name__)

@app.route("/telegram", methods=["POST"])
def telegram_webhook():
    update = request.json
    # Обработка Telegram-обновления...
    return jsonify({"ok": True})

@app.route("/discord", methods=["POST"])
def discord_webhook():
    data = request.json
    # Обработка Discord-взаимодействия...
    return jsonify({"type": 1})

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
```

```bash
# Один туннель, два endpoint
fxtunnel http 8080

# Telegram webhook -> https://bot-dev.fxtun.ru/telegram
# Discord endpoint -> https://bot-dev.fxtun.ru/discord
```

Этот подход использует один туннель и один порт — проще управлять и мониторить.

## Troubleshooting: типичные проблемы с вебхуками ботов

Что-то не работает? Чаще всего причина одна из трёх: туннель не запущен, проверка подписи не проходит или формат ответа неверный. [Inspector](/blog/traffic-inspector-debug-requests/) покажет, дошёл ли запрос до сервера вообще -- это сразу сужает область поиска.

| Проблема | Возможная причина | Решение |
|---|---|---|
| Telegram: `setWebhook` возвращает ошибку | URL недоступен | Убедитесь, что туннель запущен и URL открывается в браузере |
| Telegram: бот не отвечает | Неверный порт или путь | Проверьте, что `url_path` в коде совпадает с путём в URL вебхука |
| Telegram: `getWebhookInfo` показывает ошибки | Сервер возвращает не 200 | Проверьте Inspector — статус-код ответа и сообщение об ошибке |
| Discord: верификация endpoint не проходит | Ошибка в проверке подписи | Убедитесь, что используете raw body (не распарсенный JSON) для Ed25519-верификации |
| Discord: PING не подтверждён | Нет обработчика type 1 | Сервер должен отвечать `{"type": 1}` на PING-взаимодействия |
| Discord: slash-команда не появляется | Команда не зарегистрирована | Зарегистрируйте команды через Discord API; глобальные команды могут появляться до 1 часа |
| HTTP 502 Bad Gateway | Сервер бота не запущен | Запустите приложение до создания туннеля |
| URL изменился после перезапуска | Туннель запущен без `--domain` — поддомен каждый раз случайный | Зарезервируйте поддомен (`fxtunnel domains add bot-dev`) и запускайте с `--domain bot-dev` |

## Лучшие практики разработки ботов через туннель

Структурированный подход к разработке ботов экономит часы отладки. Эти практики применимы и к Telegram, и к Discord ботам.

### 1. Проверяйте подписи с первого дня

Telegram предоставляет параметр `secret_token` в `setWebhook` для проверки входящих запросов. Discord требует Ed25519-верификацию подписи. Реализуйте обе проверки с самого начала — пропуск верификации во время разработки приводит к тому, что вы забудете о ней перед продакшеном.

### 2. Используйте Inspector вместо print-отладки

Вместо того чтобы раскидывать `print(request.json)` по коду, откройте Inspector. Он показывает каждый запрос и ответ с правильным форматированием, а Replay позволяет перепроверить без повторного тригера события.

### 3. Отвечайте быстро

Telegram ожидает ответ в течение 60 секунд. Discord строже — сервер должен ответить в течение 3 секунд. Для долгих задач используйте паттерн отложенного ответа Discord (type 5) с последующим редактированием через webhook.

### 4. Обрабатывайте ошибки корректно

Если бот падает, платформа повторит вебхук. Реализуйте правильную обработку ошибок, чтобы сервер всегда возвращал валидный ответ — даже когда логика обработки завершается с ошибкой.

### 5. Отделяйте логику бота от HTTP-обработки

Держите обработчики маршрутов тонкими. Парсите входящие данные, передавайте в соответствующую функцию-обработчик и возвращайте ответ. Это делает код тестируемым и переносимым между режимами webhook и polling.

### 6. Используйте переменные окружения

Никогда не хардкодьте токены ботов, публичные ключи и URL вебхуков. Используйте переменные окружения, чтобы переключаться между разработкой (URL туннеля) и продакшеном (ваш домен) без изменения кода.

## Почему fxTunnel для разработки ботов

Что важно именно для ботов? Закреплённый URL (чтобы не перерегистрировать вебхуки), инспектор трафика (чтобы видеть, что пришло от платформы) и бесплатный тариф, с которого можно начать. Вот как fxTunnel выглядит на фоне [других инструментов](/blog/ngrok-vs-cloudflare-vs-fxtunnel/).

| Возможность | fxTunnel | Другие инструменты |
|---|---|---|
| Бесплатный тариф | 1 туннель, без резерва поддомена | Часто ограничен по времени или запросам |
| Стабильный URL | При закреплённом поддомене (`--domain`), с тарифа Base | Часто меняется при перезапуске |
| Inspector + Replay | С тарифа Base — просмотр всех запросов и повтор одним кликом | Отсутствует или требует отдельного инструмента |
| Кастомный домен (свой DNS) | С тарифа Base | Дороже или недоступно |
| Больше туннелей сразу | Pro — 15, Business — 50 | Значительно дороже |
| HTTPS | Автоматический TLS — требуется Telegram и Discord | Обычно включён |
| Настройка | Одна команда, 30 секунд | Часто требует токенов, регистрации, конфигурации |

Подробный обзор инструментов -- в статье [«Лучшие инструменты туннелирования 2026»](/blog/best-tunneling-tools-2026/). Также стоит заглянуть в гайд по [открытию localhost в интернет](/blog/expose-localhost-to-internet/) с рекомендациями по безопасности.

