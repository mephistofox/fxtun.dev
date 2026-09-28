---
title: "Telegram Mini App: разработка на локальной машине с HTTPS"
date: 2026-10-09T10:00:00+03:00
description: "Telegram Mini App на локальной машине: HTTPS-адрес для dev-сервера Vite через туннель, кнопка в BotFather, HMR и проверка initData на сервере."
rubrics: [guides]
need: domain
tags: [telegram, localhost, https, developer-tools, fxTunnel]
cover: ["Vite :5173", "fxtunnel http", "Telegram"]
coverHL: 1
---

Telegram Mini App открывается только по HTTPS-адресу, поэтому `http://localhost:5173` в настройки бота не впишешь. Чтобы разрабатывать мини-приложение на своём компьютере, нужен публичный HTTPS-адрес, который ведёт на dev-сервер. Его даёт туннель: `fxtunnel http 5173 --domain myapp` открывает Vite по адресу `https://myapp.fxtun.ru`. Вы указываете этот адрес в @BotFather, открываете бота в Telegram на телефоне и видите свой код, а правки подтягиваются без передеплоя.

Дальше по шагам: проект на Vite, настройка `allowedHosts`, туннель с постоянным адресом, кнопка в BotFather, HMR и проверка `initData` на своём бэкенде.

## Почему Mini App не открывается с localhost

Причин две.

**Telegram требует HTTPS.** В Bot API адрес мини-приложения описан как «An HTTPS URL of a Web App» ([Bot API, WebAppInfo](https://core.telegram.org/bots/api#webappinfo)). Обычный HTTP разрешён только в тестовом окружении Telegram, а для него нужен отдельный аккаунт и отдельный бот ([документация Mini Apps](https://core.telegram.org/bots/webapps#using-bots-in-the-test-environment)).

**Telegram открывает адрес на устройстве пользователя.** Даже если вы тестируете на своём телефоне, `localhost` для телефона — это сам телефон, а не ваш ноутбук.

Туннель закрывает обе проблемы. Клиент fxtunnel подключается к серверу fxTunnel исходящим соединением, а сервер принимает запросы на `https://<поддомен>.fxtun.ru` с настоящим сертификатом и передаёт их вашему dev-серверу.

{{< flow caption="Telegram открывает HTTPS-адрес туннеля, клиент fxtunnel передаёт запросы dev-серверу Vite." >}}
- {label: телефон, title: Telegram, sub: "кнопка меню бота"}
- {link: HTTPS, kind: tls}
- {label: сервер fxTunnel, title: поддомен, sub: "myapp.fxtun.ru"}
- {link: TLS, kind: tls}
- {label: ваш компьютер, title: клиент fxtunnel, sub: fxtunnel http 5173, me: true}
- {link: HTTP, kind: tcp}
- {label: локально, title: Vite, sub: "localhost:5173"}
{{< /flow >}}

## Шаг 1. Проект на Vite

Создайте пустой проект:

```bash
npm create vite@latest miniapp -- --template vanilla
cd miniapp
npm install
```

В `index.html` подключите скрипт Telegram. Документация просит ставить его в `<head>` раньше остальных скриптов ([core.telegram.org](https://core.telegram.org/bots/webapps#initializing-mini-apps)):

```html
<script src="https://telegram.org/js/telegram-web-app.js?63"></script>
```

Замените содержимое `src/main.js`:

```js
const tg = window.Telegram.WebApp
tg.ready()

fetch('/api/me', { headers: { 'X-Telegram-Init-Data': tg.initData } })
  .then((res) => (res.ok ? res.json() : null))
  .then((user) => {
    document.querySelector('#app').textContent = user
      ? `Привет, ${user.first_name}`
      : 'initData не прошла проверку'
  })
```

`tg.ready()` убирает заглушку загрузки. Строка `tg.initData` — данные о пользователе и запуске, подписанные Telegram. Их мы отправим на свой сервер и проверим (шаг 6).

## Шаг 2. Разрешите Vite отвечать на адрес туннеля

Vite по умолчанию отвечает только на `localhost` и IP-адреса. Запрос с чужим заголовком `Host` он отклоняет текстом:

```console
Blocked request. This host ("myapp.fxtun.ru") is not allowed.
```

Туннель заголовок `Host` не переписывает, поэтому адрес туннеля нужно добавить в `server.allowedHosts` ([документация Vite](https://vite.dev/config/server-options#server-allowedhosts)). Создайте `vite.config.js`:

```js
import { defineConfig } from 'vite'

export default defineConfig({
  server: {
    allowedHosts: ['myapp.fxtun.ru'],
    proxy: {
      '/api': 'http://127.0.0.1:8000',
    },
  },
})
```

Вписывайте конкретное имя. Значение `true` разрешает любые хосты, и документация Vite прямо предупреждает: так любой сайт сможет через DNS rebinding скачать исходники с вашего dev-сервера.

Блок `proxy` отправляет запросы `/api/...` на бэкенд на порту 8000 ([server.proxy](https://vite.dev/config/server-options#server-proxy)). Так фронтенд и API живут за одним адресом, и хватает одного туннеля.

Запустите dev-сервер:

```bash
npm run dev
```

Он слушает порт 5173, это значение Vite по умолчанию. Если порт занят, Vite возьмёт следующий свободный. Сверьте номер в выводе `npm run dev` с командой туннеля.

## Шаг 3. Туннель с постоянным поддоменом

Установите клиент и войдите (один раз):

```bash
curl -fsSL https://fxtun.ru/install.sh | sh
fxtunnel login
```

На Windows вместо первой строки выполните в PowerShell `irm https://fxtun.ru/install.ps1 | iex`.

Откройте туннель к Vite:

```bash
fxtunnel http 5173 --domain myapp
```

```console
Connecting to fxtunnel server...
Tunnel established!
HTTP:  http://myapp.fxtun.ru
HTTPS: https://myapp.fxtun.ru
Forwarding to localhost:5173
Inspector: http://127.0.0.1:4040
Ready to receive connections
```

Строка `Inspector:` появляется на тарифе Base и выше, на бесплатном её не будет.

Почему сразу с `--domain`: адрес здесь прописан в двух местах, в `allowedHosts` и в @BotFather. Без флага каждый запуск `fxtunnel http 5173` даёт новый случайный поддомен, и после каждого перезапуска пришлось бы править конфиг и снова идти к BotFather.

`--domain myapp` работает и на бесплатном тарифе, но без гарантии: пока ваш туннель выключен, имя может занять кто-то другой. Чтобы поддомен был только вашим, его резервируют:

```bash
fxtunnel domains add myapp
```

```console
Reserved: myapp → https://myapp.fxtun.ru
```

Резерв поддоменов доступен с тарифа Base. Все варианты, от своего поддомена до своего домена, разобраны в статье [«Постоянный адрес туннеля»](/blog/permanent-tunnel-address/).

Имя `myapp` в примерах условное: выберите своё и проверьте, свободно ли оно, командой `fxtunnel domains check <имя>`.

## Шаг 4. Кнопка Mini App в @BotFather

Если бота ещё нет, создайте его в @BotFather командой `/newbot` ([руководство Telegram](https://core.telegram.org/bots/tutorial)). Дальше привяжите к боту адрес мини-приложения. Проще всего через кнопку меню: команда `/setmenubutton` или пункт Bot Settings > Menu Button ([документация Mini Apps](https://core.telegram.org/bots/webapps#launching-mini-apps-from-the-menu-button)). BotFather спросит текст кнопки и адрес: укажите `https://myapp.fxtun.ru`.

Откройте чат с ботом в Telegram и нажмите кнопку меню. Мини-приложение загрузится с вашего компьютера.

Кнопка меню — не единственный способ запуска. Всего их семь: кнопка в профиле бота (Main Mini App), кнопки клавиатуры и инлайн-кнопки, инлайн-режим, прямая ссылка, меню вложений ([список способов](https://core.telegram.org/bots/webapps#implementing-mini-apps)). Для разработки кнопки меню хватает, остальные можно подключить позже тем же адресом.

### Первый экран — страница fxTunnel

Приготовьтесь: в клиенте Telegram на телефоне и на компьютере первым экраном Mini App откроется не ваше приложение, а предупредительная страница fxTunnel. Она сообщает, что сайт работает через туннель разработчика, и защищает посетителей от фишинга. После кнопки «Продолжить» Mini App загрузится.

Когда страница появляется ([документация HTTP-туннелей](https://fxtun.ru/docs/http)):

- адрес на поддомене `*.fxtun.ru`;
- запрос `GET`, и ваш сервер ответил HTML;
- в запросе нет заголовка `X-FxTunnel-Skip-Warning`;
- нет cookie согласия `_fxt_consent_<поддомен>`.

Кнопка «Продолжить» ставит это cookie на 12 часов. Значит, страница повторится при первом открытии и потом раз в 12 часов, причём на каждом устройстве и у каждого тестировщика отдельно. Язык страницы берётся из настроек языка браузера. Запросы к `/api` она не трогает: они отдают JSON.

В веб-версии Telegram (web.telegram.org) Mini App открывается внутри iframe. Cookie ставится с `SameSite=Lax`, а в таком встроенном контексте браузер его, скорее всего, не отправит, и страница будет появляться снова и снова. Поэтому тестируйте в приложении Telegram.

Чтобы показывать Mini App коллегам или заказчику без этого экрана, подключите свой домен: на своих доменах предупредительной страницы нет. Как это сделать — в конце статьи.

## Шаг 5. HMR: правки без перезагрузки

Горячая замена модулей работает через тот же адрес. Клиент HMR в браузере по умолчанию подключается по WebSocket к хосту, с которого загружена страница, а для HTTPS-страницы берёт `wss`. Документация Vite оговаривает условие: прокси перед Vite должен пропускать WebSocket ([server.ws](https://vite.dev/config/server-options#server-ws)). HTTP-туннель fxTunnel WebSocket пропускает ([документация](https://fxtun.ru/docs/http)), так что отдельной настройки не требуется.

Поменяйте текст в `src/main.js`, сохраните, и мини-приложение на телефоне обновится.

Если страница открылась, а правки не приходят, откройте инструменты разработчика для WebView. Telegram описывает, как их включить на iOS, Android, Windows, Linux и macOS ([Debug Mode for Mini Apps](https://core.telegram.org/bots/webapps#debug-mode-for-mini-apps)). В консоли будет видно, куда клиент HMR пытается подключиться.

## Шаг 6. Проверка initData на сервере

`initData` приходит в мини-приложение от Telegram, но на сервер её отправляет уже ваш JavaScript. Подделать запрос к `/api/me` может кто угодно, особенно когда адрес публичный. Поэтому Telegram прямо пишет: данным из `initDataUnsafe` доверять нельзя, использовать `initData` можно только на сервере бота и только после проверки ([core.telegram.org](https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app)).

Алгоритм из документации:

1. Разобрать `initData` как строку запроса, вынуть поле `hash`.
2. Остальные поля отсортировать по имени и склеить в строку `key=value` через перевод строки (`\n`).
3. Секретный ключ — HMAC-SHA256 от токена бота с ключом `WebAppData`.
4. Посчитать HMAC-SHA256 строки из пункта 2 этим ключом и сравнить в hex с `hash`.
5. Дополнительно проверить `auth_date`, чтобы не принимать старые данные.

Бэкенд на стандартной библиотеке Python:

```python
# api.py
import hashlib
import hmac
import json
import os
import time
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import parse_qsl

BOT_TOKEN = os.environ["BOT_TOKEN"]


def check_init_data(init_data: str, max_age: int = 3600):
    fields = dict(parse_qsl(init_data, keep_blank_values=True))
    received = fields.pop("hash", "")
    check_string = "\n".join(f"{k}={v}" for k, v in sorted(fields.items()))
    secret = hmac.new(b"WebAppData", BOT_TOKEN.encode(), hashlib.sha256).digest()
    expected = hmac.new(secret, check_string.encode(), hashlib.sha256).hexdigest()
    if not hmac.compare_digest(expected.encode(), received.encode()):
        return None
    if time.time() - int(fields.get("auth_date", "0")) > max_age:
        return None
    return json.loads(fields.get("user", "{}"))


class Api(BaseHTTPRequestHandler):
    def do_GET(self):
        user = check_init_data(self.headers.get("X-Telegram-Init-Data", ""))
        if self.path != "/api/me" or user is None:
            self.send_response(401)
            self.end_headers()
            return
        body = json.dumps(user).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(body)


HTTPServer(("127.0.0.1", 8000), Api).serve_forever()
```

Запустите его в отдельном окне, подставив токен бота от @BotFather:

```bash
BOT_TOKEN=123456:ABC... python3 api.py
```

На Windows в PowerShell: `$env:BOT_TOKEN="123456:ABC..."; python api.py`.

Теперь `/api/me` отвечает `200` с данными пользователя только на настоящую `initData` и `401` на всё остальное. Срок `max_age` в час взят для примера, выберите свой. Токен бота держите только на сервере: в код фронтенда он попасть не должен.

## Если что-то не открывается

**`Blocked request. This host ... is not allowed`.** Имя в `allowedHosts` не совпадает с поддоменом туннеля. Сверьте `vite.config.js` и флаг `--domain` и перезапустите `npm run dev`.

**Страница открылась, но пишет «initData не прошла проверку».** Скорее всего, вы открыли адрес в обычном браузере: там `initData` пустая, её передаёт только Telegram. Откройте мини-приложение кнопкой в боте. Если и в Telegram ответ `401`, проверьте, что `BOT_TOKEN` — токен именно этого бота.

**Ничего не открывается, а туннель работает.** Vite мог занять не 5173, а следующий порт, если 5173 был занят. Номер порта есть в выводе `npm run dev`, он должен совпадать с числом в `fxtunnel http`.

**Адрес вчера работал, сегодня нет.** Туннель запущен без `--domain` или с другим именем. Вернитесь к имени из BotFather.

## Что получилось и что взять в работу

Схема разработки такая: Vite и бэкенд работают на вашем компьютере, туннель даёт им один HTTPS-адрес, Telegram открывает этот адрес по кнопке бота. Вы правите код, и изменения видны на телефоне сразу, без сборки и выкладки.

Для команды или долгого проекта полезны две вещи:

- **Зарезервированный поддомен**, чтобы адрес в BotFather оставался вашим, даже когда ноутбук выключен.
- **Свой домен** вида `miniapp.example.com`. Он выглядит как продакшен, и предупредительной страницы на нём нет. Как его подключить, описано в [документации](https://fxtun.ru/docs/custom-domains) и в статье о [постоянном адресе](/blog/permanent-tunnel-address/).

## Что дальше

- Как работают вебхуки и почему они не доходят до localhost — [«Вебхук: как принять вебхук на своём компьютере»](/blog/webhooks-on-localhost/).
- Бот Telegram на вебхуках, а не только мини-приложение, — [разработка ботов Telegram и Discord через туннель](/blog/telegram-discord-bot-tunnel/).
- Чтобы адрес не менялся — [постоянный адрес туннеля](/blog/permanent-tunnel-address/).
