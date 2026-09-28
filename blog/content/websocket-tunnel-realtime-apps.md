---
title: "WebSocket через туннель — real-time приложения на localhost"
date: 2026-03-21T14:00:00+03:00
draft: false
description: "Как пробросить WebSocket-сервер через туннель: HTTP Upgrade, чат на ws и Socket.IO, отладка соединений и на что смотреть при задержке."
tags: [websocket, tunnel, realtime, socket.io, fxTunnel, localhost, developer-tools, http-upgrade]
image: ""
rubrics: [protocols]
need: free
cover: ["ws://localhost", "Upgrade", "wss://tunnel"]
updated: 2026-09-27
---

WebSocket -- основа real-time веб-приложений: чатов, live-дашбордов, совместных редакторов, уведомлений и многопользовательских игр. Но при локальной разработке знакомая проблема: ваш `localhost:8080` недоступен из интернета, и протестировать с внешними клиентами, мобильными устройствами или сторонними сервисами не получится. [Туннель](/blog/what-is-tunneling/) даёт вашему локальному серверу публичный URL с полной поддержкой WebSocket-соединений.

Ниже разбираем, как WebSocket работает через туннель, как fxTunnel обрабатывает механизм HTTP Upgrade, и пошагово строим real-time чат с `ws` и Socket.IO -- всё на localhost, доступно откуда угодно.

## Как работает WebSocket: краткое напоминание

WebSocket — это протокол коммуникации, обеспечивающий полнодуплексную двунаправленную связь через одно TCP-соединение. В отличие от HTTP, где клиент отправляет запрос и ждёт ответ, WebSocket позволяет и клиенту, и серверу отправлять сообщения в любой момент без ожидания.

Протокол начинается как обычный HTTP-запрос. Клиент отправляет специальный заголовок `Upgrade`, а сервер отвечает `101 Switching Protocols`. После этого соединение перестаёт быть HTTP — оно становится постоянным WebSocket-каналом.

### HTTP Upgrade Handshake

**Клиент → сервер:**

```http
GET /chat HTTP/1.1
Host: abc123.fxtun.ru
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
Sec-WebSocket-Version: 13
```

**Сервер → клиенту:**

```http
HTTP/1.1 101 Switching Protocols
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
```

После upgrade обе стороны обмениваются бинарными или текстовыми фреймами через то же TCP-соединение. Больше нет HTTP-запросов и ответов — только чистые фреймы.

### WebSocket vs HTTP Polling

Многие разработчики начинают с HTTP polling, прежде чем открывают для себя WebSocket. Вот как сравниваются два подхода:

| Характеристика | HTTP Polling | WebSocket |
|---|---|---|
| **Соединение** | Новый запрос каждые N секунд | Одно постоянное соединение |
| **Направление** | Клиент запрашивает у сервера | Двунаправленное (push + pull) |
| **Задержка** | 1–10 секунд (интервал опроса) | < 50 мс (мгновенный push) |
| **Трафик** | Высокий (повторяющиеся заголовки) | Низкий (маленькие фреймы) |
| **Нагрузка на сервер** | Высокая (много коротких соединений) | Низкая (одно длинное соединение) |
| **Сложность** | Просто реализовать | Немного сложнее |
| **Применение** | Простые дашборды, проверка статуса | Чат, игры, совместная работа |

Для всего, что требует взаимодействия в реальном времени — сообщения чата, позиции курсоров, live-потоки цен — WebSocket является правильным выбором. HTTP polling тратит трафик и добавляет секунды задержки на каждое обновление.

## Проблема: WebSocket на localhost

При разработке real-time приложения вы естественно начинаете на localhost. Ваш WebSocket-сервер слушает `ws://localhost:8080`, и клиент подключается к нему напрямую. Это работает для одиночной разработки, но ломается в момент, когда вам нужно:

- **Протестировать с мобильного устройства** — ваш телефон не может достучаться до `localhost` на рабочей машине.
- **Поделиться превью** — коллега или заказчик не может открыть ваш локальный URL.
- **Интегрироваться с внешними сервисами** — сервисы вроде Slack, Discord-ботов или IoT-устройств требуют публичный endpoint.
- **Протестировать между сетями** — поведение WebSocket может отличаться между локальным и удалённым подключением из-за прокси, файрволов и NAT.
- **Показать демо без деплоя** — вы хотите показать рабочий прототип, не загружая код на сервер.

Туннель решает все эти задачи. Общий подход описан в гайде [Как открыть localhost в интернет](/blog/expose-localhost-to-internet/).

## Как WebSocket работает через туннель fxTunnel

Нужен ли какой-то специальный флаг для включения WebSocket? Нет. fxTunnel поддерживает WebSocket нативно через HTTP-туннель. Когда внешний клиент подключается к публичному URL туннеля и отправляет HTTP Upgrade запрос, fxTunnel пересылает handshake вашему локальному серверу, а после upgrade проксирует WebSocket-фреймы в обоих направлениях.

### Схема соединения

{{< flow caption="Сервер пропускает Upgrade-запрос насквозь; после хэндшейка туннель просто копирует фреймы в обе стороны." >}}
- {label: клиент, title: браузер / мобильное приложение, sub: "GET с Upgrade: websocket"}
- {link: HTTPS, kind: tls, note: "https://abc123.fxtun.ru"}
- {label: сервер fxTunnel, sub: пересылает Upgrade насквозь}
- {link: TLS, kind: tls, note: "поток мультиплексора, по умолчанию"}
- {label: клиент fxtunnel, sub: "fxtunnel http 8080", me: true}
- {link: HTTP, kind: http, note: "101 Switching Protocols"}
- {label: localhost:8080, title: ваш WebSocket-сервер}
{{< /flow >}}

Ключевой момент: после завершения HTTP Upgrade handshake соединение превращается в сырой TCP-поток с WebSocket-фреймами. Мультиплексор fxTunnel рассматривает его как долгоживущий HTTP-поток -- ничем не отличающийся от chunked-ответа или server-sent events. Как именно работает мультиплексирование, описано в статье про [архитектуру fxTunnel](/blog/fxtunnel-architecture/).

### Что происходит под капотом

1. **Внешний клиент** отправляет HTTP GET запрос с `Upgrade: websocket` на `https://abc123.fxtun.ru/chat`.
2. **Сервер fxTunnel** получает запрос, видит заголовок Upgrade и пересылает весь запрос через мультиплексор клиенту fxTunnel. Критически важно: сервер не завершает Upgrade — он пропускает его насквозь.
3. **Клиент fxTunnel** открывает соединение с `localhost:8080` и пересылает Upgrade-запрос.
4. **Ваш локальный сервер** отвечает `101 Switching Protocols`.
5. **Ответ идёт обратно** по цепочке: локальный сервер -> клиент fxTunnel -> сервер fxTunnel -> внешний клиент.
6. **Обе стороны** теперь обмениваются WebSocket-фреймами. Туннель проксирует фреймы в обоих направлениях без инспекции и модификации.
7. **Когда одна из сторон закрывает** WebSocket, close-фрейм распространяется по всей цепочке, и поток мультиплексора освобождается.

Этот процесс полностью прозрачен. Ваш локальный сервер видит обычное WebSocket-соединение со стандартными заголовками. Внешний клиент видит обычный WebSocket-endpoint на публичном HTTPS URL с валидным TLS — именно то, что требуют браузеры и мобильные приложения.

## Создание real-time чата: пошагово

Построим простое, но полноценное real-time чат-приложение и откроем его через fxTunnel. Это демонстрирует полный рабочий процесс: локальная разработка, настройка туннеля и тестирование с нескольких устройств.

### Шаг 1. Установка fxTunnel

```bash
# Быстрая установка (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Проверяем, что всё работает
fxtunnel version
```

### Шаг 2. Создание WebSocket-сервера

Используем библиотеку `ws` — самую популярную реализацию WebSocket для Node.js.

```bash
mkdir ws-chat && cd ws-chat
npm init -y
npm install ws
```

```javascript
// server.js
const http = require('http');
const fs = require('fs');
const WebSocket = require('ws');

// Создаём HTTP-сервер для отдачи HTML-страницы
const server = http.createServer((req, res) => {
  if (req.url === '/' || req.url === '/index.html') {
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end(fs.readFileSync('index.html'));
  } else {
    res.writeHead(404);
    res.end('Not found');
  }
});

// Создаём WebSocket-сервер, привязанный к HTTP-серверу
const wss = new WebSocket.Server({ server });

// Отслеживаем подключённых клиентов
const clients = new Set();

wss.on('connection', (ws, req) => {
  const clientId = req.headers['x-forwarded-for'] || req.socket.remoteAddress;
  console.log(`Клиент подключился: ${clientId}`);
  clients.add(ws);

  // Отправляем приветственное сообщение
  ws.send(JSON.stringify({
    type: 'system',
    message: `Добро пожаловать! ${clients.size} пользователь(ей) онлайн.`,
    timestamp: new Date().toISOString(),
  }));

  // Уведомляем остальных клиентов
  broadcast(ws, {
    type: 'system',
    message: 'Новый пользователь присоединился к чату.',
    timestamp: new Date().toISOString(),
  });

  // Обработка входящих сообщений
  ws.on('message', (data) => {
    const parsed = JSON.parse(data);
    console.log(`Сообщение от ${clientId}: ${parsed.message}`);

    // Рассылаем сообщение всем клиентам (включая отправителя)
    const outgoing = {
      type: 'message',
      user: parsed.user || 'Anonymous',
      message: parsed.message,
      timestamp: new Date().toISOString(),
    };

    for (const client of clients) {
      if (client.readyState === WebSocket.OPEN) {
        client.send(JSON.stringify(outgoing));
      }
    }
  });

  // Обработка отключения
  ws.on('close', () => {
    clients.delete(ws);
    console.log(`Клиент отключился: ${clientId}`);
    broadcast(null, {
      type: 'system',
      message: 'Пользователь покинул чат.',
      timestamp: new Date().toISOString(),
    });
  });
});

function broadcast(excludeWs, data) {
  for (const client of clients) {
    if (client !== excludeWs && client.readyState === WebSocket.OPEN) {
      client.send(JSON.stringify(data));
    }
  }
}

server.listen(8080, () => {
  console.log('Чат-сервер запущен на http://localhost:8080');
});
```

### Шаг 3. Создание клиентской HTML-страницы

```html
<!-- index.html -->
<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8">
  <title>WebSocket Чат</title>
  <style>
    body { font-family: sans-serif; max-width: 600px; margin: 40px auto; }
    #messages { border: 1px solid #ccc; height: 400px; overflow-y: auto; padding: 10px; }
    .system { color: #888; font-style: italic; }
    .message { margin: 4px 0; }
    #form { display: flex; gap: 8px; margin-top: 8px; }
    #form input { flex: 1; padding: 8px; }
    #form button { padding: 8px 16px; }
  </style>
</head>
<body>
  <h1>WebSocket Чат</h1>
  <div id="messages"></div>
  <form id="form">
    <input id="user" placeholder="Ваше имя" value="" />
    <input id="input" placeholder="Введите сообщение..." autocomplete="off" />
    <button type="submit">Отправить</button>
  </form>
  <script>
    // Подключаемся к WebSocket — работает и с localhost, и с URL туннеля
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${protocol}//${location.host}`);

    const messages = document.getElementById('messages');

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      const div = document.createElement('div');

      if (data.type === 'system') {
        div.className = 'system';
        div.textContent = data.message;
      } else {
        div.className = 'message';
        div.textContent = `[${data.user}]: ${data.message}`;
      }

      messages.appendChild(div);
      messages.scrollTop = messages.scrollHeight;
    };

    ws.onclose = () => {
      const div = document.createElement('div');
      div.className = 'system';
      div.textContent = 'Соединение закрыто.';
      messages.appendChild(div);
    };

    document.getElementById('form').onsubmit = (e) => {
      e.preventDefault();
      const input = document.getElementById('input');
      const user = document.getElementById('user').value || 'Anonymous';

      if (input.value.trim()) {
        ws.send(JSON.stringify({ user, message: input.value }));
        input.value = '';
      }
    };
  </script>
</body>
</html>
```

Обратите внимание: клиентский код использует `location.host` для определения WebSocket URL. Это значит, что он работает прозрачно — как при доступе через `localhost:8080`, так и через туннель на `abc123.fxtun.ru` — без хардкода URL.

### Шаг 4. Запуск сервера и создание туннеля

```bash
# Терминал 1: Запускаем чат-сервер
node server.js
# -> Чат-сервер запущен на http://localhost:8080

# Терминал 2: Создаём туннель
fxtunnel http 8080
```

fxTunnel выводит публичный URL:

```
Connecting to fxtunnel server...
Tunnel established!
HTTPS: https://ws-chat.fxtun.ru
Forwarding to localhost:8080
Inspector: http://127.0.0.1:4040
```

### Шаг 5. Тестирование

1. Откройте `https://ws-chat.fxtun.ru` в браузере — чат загружается, WebSocket подключается.
2. Откройте тот же URL на телефоне — вы увидите, что второй пользователь присоединился.
3. Отправьте сообщения — они мгновенно появляются на обоих устройствах.
4. Отправьте URL коллеге — он сможет подключиться к чату из своей сети.

WebSocket-соединение полностью зашифровано (WSS) на публичной стороне благодаря TLS-сертификату туннеля. Ваш локальный сервер работает на обычном `ws://` — туннель берёт на себя терминацию TLS.

## Socket.IO через туннель

Socket.IO — самый популярный real-time фреймворк для Node.js. Он добавляет автоматическое переподключение, комнаты, пространства имён и fallback на HTTP long-polling, когда WebSocket недоступен. Socket.IO работает через fxTunnel без какой-либо специальной настройки.

### Как Socket.IO использует WebSocket

Socket.IO использует двухфазную стратегию подключения:

1. **Фаза 1: HTTP long-polling** — Socket.IO начинает с обычных HTTP-запросов для установки соединения и обмена метаданными.
2. **Фаза 2: WebSocket upgrade** — после установки соединения Socket.IO переключается на WebSocket для лучшей производительности.

Оба этапа идут через один и тот же путь — клиент, сервер fxTunnel, ваш сервер — туннель просто пересылает то, что видит: сперва poll-запросы, потом Upgrade, потом фреймы.

fxTunnel обрабатывает обе фазы прозрачно — HTTP polling-запросы и WebSocket upgrade. Это важно, потому что некоторые инструменты туннелирования ломают Socket.IO, не пересылая заголовок Upgrade корректно.

### Пример сервера Socket.IO

```bash
npm install express socket.io
```

```javascript
// socketio-server.js
const express = require('express');
const http = require('http');
const { Server } = require('socket.io');

const app = express();
const server = http.createServer(app);
const io = new Server(server, {
  cors: { origin: '*' },
});

app.get('/', (req, res) => {
  res.sendFile(__dirname + '/socketio-client.html');
});

// Отслеживаем онлайн-пользователей
let onlineCount = 0;

io.on('connection', (socket) => {
  onlineCount++;
  console.log(`Пользователь подключился (${onlineCount} онлайн)`);

  // Уведомляем всех о новом количестве
  io.emit('online-count', onlineCount);

  // Обработка сообщений чата
  socket.on('chat-message', (data) => {
    console.log(`${data.user}: ${data.message}`);
    // Рассылаем всем (включая отправителя)
    io.emit('chat-message', {
      user: data.user,
      message: data.message,
      timestamp: new Date().toISOString(),
    });
  });

  // Индикатор набора текста
  socket.on('typing', (user) => {
    socket.broadcast.emit('typing', user);
  });

  // Обработка отключения
  socket.on('disconnect', () => {
    onlineCount--;
    console.log(`Пользователь отключился (${onlineCount} онлайн)`);
    io.emit('online-count', onlineCount);
  });
});

server.listen(8080, () => {
  console.log('Socket.IO сервер запущен на http://localhost:8080');
});
```

### Пример клиента Socket.IO

```html
<!-- socketio-client.html -->
<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8">
  <title>Socket.IO Чат</title>
  <script src="/socket.io/socket.io.js"></script>
</head>
<body>
  <h1>Socket.IO Чат <span id="count"></span></h1>
  <div id="messages" style="height:300px; overflow-y:auto; border:1px solid #ccc; padding:10px;"></div>
  <p id="typing" style="color:#888; height:20px;"></p>
  <form id="form">
    <input id="user" placeholder="Ваше имя" />
    <input id="input" placeholder="Сообщение..." autocomplete="off" />
    <button type="submit">Отправить</button>
  </form>
  <script>
    // Socket.IO автоматически определяет URL сервера по origin страницы
    const socket = io();

    socket.on('online-count', (count) => {
      document.getElementById('count').textContent = `(${count} онлайн)`;
    });

    socket.on('chat-message', (data) => {
      const div = document.createElement('div');
      div.textContent = `[${data.user}]: ${data.message}`;
      document.getElementById('messages').appendChild(div);
    });

    socket.on('typing', (user) => {
      document.getElementById('typing').textContent = `${user} печатает...`;
      setTimeout(() => {
        document.getElementById('typing').textContent = '';
      }, 2000);
    });

    document.getElementById('input').addEventListener('input', () => {
      const user = document.getElementById('user').value || 'Anonymous';
      socket.emit('typing', user);
    });

    document.getElementById('form').onsubmit = (e) => {
      e.preventDefault();
      const input = document.getElementById('input');
      const user = document.getElementById('user').value || 'Anonymous';
      if (input.value.trim()) {
        socket.emit('chat-message', { user, message: input.value });
        input.value = '';
      }
    };
  </script>
</body>
</html>
```

### Запуск Socket.IO через fxTunnel

```bash
# Терминал 1: Запускаем сервер
node socketio-server.js

# Терминал 2: Создаём туннель
fxtunnel http 8080
```

Socket.IO автоматически определяет URL сервера по `window.location`, поэтому клиентский код работает без изменений — как через `localhost:8080`, так и через `https://abc123.fxtun.ru`.

## Задержки: на что обратить внимание

Задержка — важнейший фактор для real-time приложений. Туннель добавляет дополнительный сетевой хоп между клиентом и вашим сервером, что увеличивает round-trip time. Понимание этого overhead помогает решить, когда туннель уместен, а когда нет.

### Откуда берётся задержка

- **Без туннеля** — клиент и сервер соединены напрямую, один сетевой переход.
- **С туннелем** — клиент, сервер fxTunnel, TLS-соединение с мультиплексором, клиент fxTunnel, ваш сервер: добавляется ретранслятор и один дополнительный переход. Точный overhead зависит от того, где стоит сервер туннеля относительно клиента и вашей машины.

Туннель добавляет три источника задержки:

1. **Дополнительный сетевой хоп** — данные идут клиент -> сервер туннеля -> клиент туннеля -> локальный сервер (и обратно). Если сервер туннеля близко к обоим конечным точкам, задержка минимальна.
2. **TLS шифрование/дешифрование** — каждый фрейм шифруется на одном конце и расшифровывается на другом. Современное железо обрабатывает TLS 1.2 (версия, которую по умолчанию использует канал клиент-сервер fxTunnel) с незначительным overhead.
3. **Фрейминг мультиплексора** — данные оборачиваются в фреймы мультиплексора для передачи. Это добавляет несколько байт на фрейм.

### Overhead в реальных условиях

Точной универсальной цифры в миллисекундах здесь нет — overhead зависит от того, где физически стоит сервер туннеля относительно вас и вашего сервиса, от загрузки сети и от TLS-издержек на конкретном железе. Но порядок понятен: relay добавляет один сетевой хоп поверх базовой задержки, и это справедливо для WebSocket так же, как для HTTP или любого другого протокола через туннель. Для большинства real-time приложений — чатов, уведомлений, дашбордов, совместного редактирования — этот overhead теряется на фоне обычного джиттера сети.

### Когда задержка туннеля имеет значение

Для подавляющего большинства real-time веб-приложений overhead туннеля незначителен и практически неразличим для пользователей. Однако в некоторых специфических сценариях каждая миллисекунда на счету:

- **Соревновательные многопользовательские игры** — покадровый ввод требует задержки менее 10 мс. Используйте [UDP-туннель](/blog/tcp-udp-tunneling-explained/).
- **Высокочастотная торговля** — не типичный сценарий localhost-разработки, но крайне чувствительный к задержке.
- **Real-time обработка аудио/видео** — для продакшена рассмотрите WebRTC с прямыми peer-соединениями.

Для разработки и тестирования overhead туннеля всегда приемлем. Вы тестируете функциональность, а не измеряете продакшен-задержку.

## WebSocket-туннель vs polling: сравнение трафика

Одно из главных преимуществ WebSocket перед polling становится ещё заметнее через туннель. Polling генерирует значительно больше трафика, что важно и для расхода полосы, и для производительности туннеля.

### Сравнение трафика: чат со 100 сообщениями

```
HTTP Polling (каждые 2 секунды, 30-секундная сессия):
  15 poll-запросов x ~500 байт заголовков = 7 500 байт overhead
  100 сообщений x ~200 байт каждое       = 20 000 байт payload
  Итого: ~27 500 байт
  Открыто соединений: 15

WebSocket:
  1 Upgrade-запрос                        = ~500 байт
  100 сообщений x ~210 байт (фрейм)      = 21 000 байт payload
  Итого: ~21 500 байт
  Открыто соединений: 1
```

WebSocket использует на 22% меньше трафика в этом простом примере. В реальных сценариях с частыми мелкими обновлениями (индикаторы набора, позиции курсоров, presence) разница может быть намного больше, потому что polling каждый раз отправляет полные HTTP-заголовки, а WebSocket — только полезную нагрузку.

Через туннель эта разница усиливается. Каждый HTTP poll-запрос проходит полную цепочку туннеля (TLS-шифрование, фрейминг мультиплексора, пересылка). Одно WebSocket-соединение проходит цепочку один раз, а потом эффективно стримит данные.

## Отладка WebSocket-соединений через туннель

Отладка real-time приложений требует специализированных инструментов. Inspector в fxTunnel записывает весь HTTP-трафик, включая WebSocket Upgrade handshake. Для более глубокой инспекции WebSocket-фреймов комбинируйте его с браузерными DevTools.

### Использование Browser DevTools

1. Откройте DevTools (F12) в браузере.
2. Перейдите на вкладку **Network**.
3. Отфильтруйте по **WS**, чтобы видеть WebSocket-соединения.
4. Кликните на WebSocket-соединение для просмотра отдельных фреймов.
5. Вкладка **Messages** показывает отправленные и полученные фреймы в реальном времени.

### Использование Inspector в fxTunnel

Inspector в fxTunnel захватывает начальный HTTP Upgrade запрос, что полезно для отладки сбоев соединения:

- **401/403 ошибки** — проблемы аутентификации до upgrade.
- **400 Bad Request** — некорректные Upgrade-заголовки.
- **502 Bad Gateway** — локальный сервер не запущен или не принимает WebSocket-соединения.
- **Тайм-аут соединения** — локальный сервер не отвечает на Upgrade-запрос.

Функция Replay позволяет повторно отправить Upgrade-запрос для тестирования разных сценариев без переподключения клиента. Весь процесс описан в статье про [Traffic Inspector](/blog/traffic-inspector-debug-requests/).

### Типичные проблемы и решения

| Проблема | Причина | Решение |
|---|---|---|
| WebSocket-соединение не устанавливается | Локальный сервер не запущен | Запустите сервер до создания туннеля |
| `ws://` соединение отклонено | Mixed content (HTTPS-страница, WS-протокол) | Используйте `wss://` — туннель предоставляет TLS автоматически |
| Соединение обрывается через 60 секунд | Idle timeout | Реализуйте ping/pong фреймы (большинство библиотек делают это по умолчанию) |
| Socket.IO откатывается на polling | Заголовок Upgrade не пересылается | Убедитесь, что ваш инструмент туннелирования поддерживает WebSocket — fxTunnel поддерживает |
| CORS-ошибки при подключении | Отсутствуют CORS-заголовки | Настройте `cors: { origin: '*' }` в Socket.IO или добавьте CORS-заголовки |
| Несколько клиентов получают одни и те же данные | Ошибка в логике рассылки | Проверьте серверную логику broadcast |

## Практические сценарии использования

WebSocket-туннели — это не только для демо-чатов. Вот практические сценарии, в которых разработчики используют WebSocket через fxTunnel каждый день.

### Live-дашборды и мониторинг

Отправляйте метрики в реальном времени на дашборд, запущенный на localhost. Поделитесь URL туннеля с командой, чтобы все могли наблюдать за прогрессом деплоя, метриками сервера или статусом CI/CD пайплайна в реальном времени.

```javascript
// Отправляем метрики сервера каждую секунду
setInterval(() => {
  const metrics = {
    cpu: os.loadavg()[0],
    memory: process.memoryUsage().heapUsed / 1024 / 1024,
    uptime: process.uptime(),
    timestamp: Date.now(),
  };
  io.emit('metrics', metrics);
}, 1000);
```

### Совместное редактирование

Создайте совместный текстовый редактор или доску. Позиции курсоров и правки каждого пользователя рассылаются всем подключённым клиентам. Туннель позволяет тестировать с несколькими пользователями на разных устройствах и в разных сетях — это критически важно для обнаружения ошибок синхронизации, которые проявляются только при реальной сетевой задержке.

### Тестирование IoT-устройств

IoT-устройство отправляет данные сенсоров по WebSocket на ваш локальный сервер. Туннелируя WebSocket endpoint, устройство может подключаться из любой сети без деплоя сервера. Этот паттерн подробнее разобран в статье про [IoT-туннелирование](/blog/iot-tunnel-devices-cloud/).

### Разработка мобильных приложений

Ваше мобильное приложение подключается к WebSocket API на рабочей машине. Туннель даёт стабильный HTTPS URL, который работает и из iOS-симулятора, и с физического Android-устройства в другой WiFi-сети. Подробнее — в статье [Тестирование мобильных приложений с туннелем](/blog/mobile-app-testing-tunnel/).

## Тарифы и планы

Поддержка WebSocket включена во все тарифы fxTunnel, включая бесплатный.

| Тариф | Возможности WebSocket |
|---|---|
| **Free** | 1 туннель, полная поддержка WebSocket, без Inspector |
| **Base** | 5 туннелей, Inspector (просмотр Upgrade-запросов) и Replay, кастомный домен |
| **Pro** | 15 туннелей, до 5 кастомных доменов |
| **Business** | 50 туннелей, до 50 кастомных доменов |

Для одного WebSocket-сервиса хватает бесплатного тарифа. Если нужно разобраться в сложных проблемах с подключением, Inspector от тарифа Base захватывает Upgrade handshake и показывает, где именно что-то пошло не так. Командам с несколькими WebSocket-микросервисами подойдёт тариф Pro и выше.
