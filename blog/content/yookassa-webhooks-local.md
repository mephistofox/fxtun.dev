---
title: "Вебхуки ЮKassa на локальной машине: уведомления об оплате"
date: 2026-10-25T10:00:00+03:00
description: "Вебхук ЮKassa на своём компьютере: где указать URL уведомлений, как принять payment.succeeded на localhost через туннель и проверить, что его прислала ЮKassa."
rubrics: [guides]
need: inspector
tags: [webhooks, yookassa, payments, localhost, fxTunnel]
cover: ["ЮKassa", "fxtunnel http", "localhost:8000"]
coverHL: 1
---

ЮKassa сообщает о смене статуса платежа HTTP-уведомлением: POST-запросом с JSON на адрес, который вы указали в личном кабинете. Адрес должен быть публичным, с HTTPS на порту 443 или 8443, поэтому `http://localhost:8000` не подойдёт. Чтобы принять вебхук ЮKassa на своём компьютере, откройте туннель `fxtunnel http 8000`, впишите выданный адрес `https://…fxtun.ru` в разделе «Интеграция — HTTP-уведомления» тестового магазина и проведите тестовый платёж. Ниже всё по шагам: обработчик на Python, проверка подлинности уведомления и повтор уведомления без новой оплаты.

## Как устроены уведомления ЮKassa

Всё в этом разделе взято из [документации ЮKassa о входящих уведомлениях](https://yookassa.ru/developers/using-api/webhooks).

Событие называется по шаблону `<объект>.<статус>`. Для приёма платежей в интернете доступны:

| Событие | Когда приходит |
|---|---|
| `payment.waiting_for_capture` | Платёж оплачен и ждёт подтверждения (если вы создали его с `capture: false`) |
| `payment.succeeded` | Платёж прошёл, деньги ваши |
| `payment.canceled` | Платёж отменён: покупатель передумал или что-то пошло не так |
| `refund.succeeded` | Возврат выполнен |
| `payment_method.active` | Способ оплаты сохранён для автоплатежей |

В теле три поля: `type` всегда равен `notification`, `event` содержит событие, `object` — сам платёж или возврат в том виде, в каком он был в момент события.

```json
{
  "type": "notification",
  "event": "payment.succeeded",
  "object": {
    "id": "22d6d597-000f-5000-9000-145f6df21d6f",
    "status": "succeeded",
    "amount": { "value": "2.00", "currency": "RUB" },
    "test": true
  }
}
```

Пример сокращён, в настоящем уведомлении полей больше. Получение нужно подтвердить ответом `200`, тело ответа ЮKassa игнорирует. На любой другой код она продолжит доставлять уведомление в течение 24 часов с момента события.

Требования к адресу: протокол HTTPS, TCP-порт 443 или 8443, TLS 1.2 или выше, сертификат подойдёт любой. Туннель fxTunnel этим требованиям отвечает: адрес `https://<поддомен>.fxtun.ru` работает на порту 443 с сертификатом Let's Encrypt.

{{< flow caption="ЮKassa отправляет POST на HTTPS-адрес туннеля, клиент fxtunnel доставляет его обработчику на вашем компьютере." >}}
- {label: облако, title: ЮKassa, sub: "payment.succeeded"}
- {link: HTTPS, kind: tls}
- {label: сервер fxTunnel, title: поддомен, sub: "shop-yk.fxtun.ru"}
- {link: TLS, kind: tls}
- {label: ваш компьютер, title: клиент fxtunnel, sub: fxtunnel http 8000, me: true}
- {link: HTTP, kind: http}
- {label: локально, title: обработчик, sub: "localhost:8000"}
{{< /flow >}}

## Шаг 1. Тестовый магазин

Настоящие деньги для отладки не нужны. По [документации о тестировании](https://yookassa.ru/developers/payment-acceptance/testing-and-going-live/testing) тестовый магазин создаётся в личном кабинете ЮKassa, даже до заключения договора. В нём работают платежи, возвраты и уведомления, а деньги никуда не переводятся.

Из настроек тестового магазина понадобятся две вещи: идентификатор магазина (shopId) и секретный ключ. Ключ тестового магазина подходит только к нему, с боевым их не перепутать: у тестовых платежей в объекте `"test": true`.

Документация советует отдельный URL для тестовых уведомлений, чтобы случайно не выдать товар за тестовую оплату. Туннель на свой компьютер как раз такой отдельный адрес.

## Шаг 2. Обработчик

Обработчик на стандартной библиотеке Python. На каждое уведомление он запрашивает у API ЮKassa актуальное состояние объекта и печатает его. Почему именно так, объясним в разделе о проверке подлинности.

```python
# yk_hook.py
import base64
import json
import os
import urllib.parse
import urllib.request
from http.server import BaseHTTPRequestHandler, HTTPServer

SHOP_ID = os.environ["YOOKASSA_SHOP_ID"]
SECRET_KEY = os.environ["YOOKASSA_SECRET_KEY"]
AUTH = "Basic " + base64.b64encode(f"{SHOP_ID}:{SECRET_KEY}".encode()).decode()
PATHS = {"payment": "payments", "refund": "refunds"}


def fetch(kind, object_id):
    req = urllib.request.Request(
        f"https://api.yookassa.ru/v3/{PATHS[kind]}/{urllib.parse.quote(object_id, safe='')}",
        headers={"Authorization": AUTH},
    )
    with urllib.request.urlopen(req, timeout=10) as resp:
        return json.load(resp)


class Hook(BaseHTTPRequestHandler):
    def reply(self, code):
        self.send_response(code)
        self.end_headers()

    def do_POST(self):
        raw = self.rfile.read(int(self.headers.get("Content-Length", 0)))
        try:
            note = json.loads(raw)
            kind = note["event"].split(".")[0]
            object_id = note["object"]["id"]
        except (ValueError, KeyError, AttributeError, TypeError):
            return self.reply(400)
        if kind not in PATHS:
            print("skip", note["event"], flush=True)
            return self.reply(200)
        try:
            actual = fetch(kind, object_id)
        except Exception as err:
            print("check failed:", note["event"], object_id, err, flush=True)
            return self.reply(500)
        print(note["event"], object_id, "->", actual.get("status"),
              "test" if actual.get("test") else "live", flush=True)
        self.reply(200)


HTTPServer(("127.0.0.1", 8000), Hook).serve_forever()
```

Что здесь важно:

- **Секреты из переменных окружения.** shopId и ключ в коде не лежат и в репозиторий не попадут. API ЮKassa принимает их как HTTP Basic Auth, так же, как в примерах `curl -u` из [быстрого старта](https://yookassa.ru/developers/payment-acceptance/getting-started/quick-start).
- **Решение принимается по ответу API, а не по телу уведомления.** Если бы обработчик выдавал заказ, он смотрел бы на `actual.get("status")`.
- **Не удалось проверить — ответ `500`.** ЮKassa повторит уведомление позже, в пределах 24 часов. Так временная ошибка сети не съест оплату.
- **Одно событие может прийти дважды**, например если ответ `200` потерялся по дороге. Выдачу заказа делайте так, чтобы повтор ничего не ломал: по `id` платежа проверяйте, не выдан ли уже заказ.

Запуск (подставьте данные тестового магазина):

```bash
export YOOKASSA_SHOP_ID=123456
export YOOKASSA_SECRET_KEY=test_ваш_ключ
python3 yk_hook.py
```

На Windows в PowerShell: `$env:YOOKASSA_SHOP_ID="123456"; $env:YOOKASSA_SECRET_KEY="test_ваш_ключ"; python yk_hook.py`.

## Шаг 3. Туннель

Установите клиент и войдите, это делается один раз:

```bash
curl -fsSL https://fxtun.ru/install.sh | sh
fxtunnel login
```

На Windows вместо первой строки в PowerShell: `irm https://fxtun.ru/install.ps1 | iex`.

Во втором окне терминала откройте туннель на порт обработчика. Имя поддомена задайте сразу: адрес будет записан в настройках магазина, и менять его после каждого перезапуска неудобно.

```bash
fxtunnel http 8000 --domain shop-yk
```

```console
Connecting to fxtunnel server...
Tunnel established!
HTTP:  http://shop-yk.fxtun.ru
HTTPS: https://shop-yk.fxtun.ru
Forwarding to localhost:8000
Inspector: http://127.0.0.1:4040
Ready to receive connections
```

Имя `shop-yk` условное, выберите своё. Строка `Inspector:` появляется на тарифах с инспектором, на бесплатном её нет.

Страница-предупреждение, которую fxTunnel показывает в браузере на `*.fxtun.ru`, уведомлениям не мешает: она бывает только у GET-запросов с HTML в ответе, а ЮKassa шлёт POST.

## Шаг 4. URL в личном кабинете

Для магазинов с аутентификацией по HTTP Basic Auth (а приём платежей в интернете работает именно так) уведомления настраиваются в личном кабинете, раздел **Интеграция — HTTP-уведомления** ([документация](https://yookassa.ru/developers/using-api/webhooks)):

1. Откройте тестовый магазин.
2. Перейдите в **Интеграция — HTTP-уведомления**.
3. В URL для уведомлений укажите `https://shop-yk.fxtun.ru/yookassa`.
4. Отметьте события: `payment.succeeded`, `payment.canceled`, а если нужны, то `payment.waiting_for_capture` и `refund.succeeded`.
5. Сохраните.

Путь `/yookassa` доходит до обработчика как есть. Обработчику из примера путь не важен, в своём приложении направьте его на нужный маршрут.

Если вы работаете через OAuth в партнёрской программе, личный кабинет не подойдёт: подписка оформляется запросом `POST /v3/webhooks`, это описано в той же документации.

## Шаг 5. Тестовый платёж

Создайте платёж запросом из [быстрого старта](https://yookassa.ru/developers/payment-acceptance/getting-started/quick-start). Ключ идемпотентности должен быть уникальным для каждого нового платежа:

```bash
curl https://api.yookassa.ru/v3/payments \
  -X POST \
  -u "$YOOKASSA_SHOP_ID:$YOOKASSA_SECRET_KEY" \
  -H "Idempotence-Key: $(python3 -c 'import uuid; print(uuid.uuid4())')" \
  -H 'Content-Type: application/json' \
  -d '{
        "amount": {"value": "100.00", "currency": "RUB"},
        "capture": true,
        "confirmation": {"type": "redirect", "return_url": "https://shop-yk.fxtun.ru/"},
        "description": "Тестовый заказ"
      }'
```

В ответе будет платёж в статусе `pending` и ссылка `confirmation_url`. Откройте её в браузере и заплатите тестовой картой `5555 5555 5555 4477`: срок действия любой в будущем, CVC и код 3-D Secure — любые цифры ([список тестовых карт](https://yookassa.ru/developers/payment-acceptance/testing-and-going-live/testing)).

Через несколько секунд в окне обработчика появится строка:

```console
payment.succeeded 2e7c3f1a-000f-5000-8000-1b2c3d4e5f60 -> succeeded test
```

Идентификатор у вас будет свой. Уведомление пришло через туннель, обработчик сверил статус с API и ответил `200`.

## Проверка подлинности уведомления

Адрес туннеля публичный. Кто его узнал, тот может отправить на `/yookassa` запрос, похожий на уведомление об оплате. Документация ЮKassa предлагает два способа отличить настоящее уведомление.

**Проверить статус объекта.** Запросить платёж по `id` из уведомления (`GET https://api.yookassa.ru/v3/payments/{id}`) и смотреть на статус из ответа API. Поддельное уведомление с чужим или выдуманным `id` ничего не даст: API либо не найдёт такой платёж в вашем магазине, либо вернёт его настоящий статус. Обработчик выше делает именно это. Способ не зависит от сети и работает одинаково через туннель и без него.

**Проверить IP-адрес отправителя.** ЮKassa присылает уведомления только с адресов из списка в документации. На момент написания он такой: `185.71.76.0/27`, `185.71.77.0/27`, `77.75.153.0/25`, `77.75.156.11`, `77.75.156.35`, `77.75.154.128/25`, `2a02:5180::/32`.

Через туннель эту проверку удобнее делать не в обработчике, а на самом туннеле. До обработчика запрос доходит от клиента fxtunnel на той же машине, и адрес соединения у него `127.0.0.1`. А флаг `--allow-ip` проверяет настоящий адрес отправителя на сервере fxTunnel и отвечает остальным `403 Forbidden`, не пропуская их к вам:

```bash
fxtunnel http 8000 --domain shop-yk \
  --allow-ip 185.71.76.0/27 --allow-ip 185.71.77.0/27 \
  --allow-ip 77.75.153.0/25 --allow-ip 77.75.156.11 \
  --allow-ip 77.75.156.35 --allow-ip 77.75.154.128/25 \
  --allow-ip 2a02:5180::/32
```

```console
Connecting to fxtunnel server...
Tunnel established!
HTTP:  http://shop-yk.fxtun.ru
HTTPS: https://shop-yk.fxtun.ru
Forwarding to localhost:8000
IP Allowlist: 7 entries
Inspector: http://127.0.0.1:4040
Ready to receive connections
```

Два нюанса. Со своего компьютера вы теперь тоже получите `403`: для ручных проверок запускайте туннель без списка или добавьте свой адрес ещё одним `--allow-ip`. И список ЮKassa может поменяться, поэтому сверяйте его с документацией, а не копируйте отсюда навсегда. Проверку статуса через API оставьте в любом случае: она главная.

## Повтор уведомления инспектором

Самое долгое в отладке — вызвать уведомление заново. Поправили обработчик, и снова создаёте платёж и вводите тестовую карту.

Инспектор fxTunnel снимает эту рутину. Откройте `http://127.0.0.1:4040` в браузере: там каждое уведомление, пришедшее через туннель, с заголовками, JSON-телом и ответом обработчика. Кнопка Replay («Повторить» в русском интерфейсе) отправляет сохранённый запрос в локальный обработчик ещё раз, минуя сервер fxTunnel и ЮKassa ([документация инспектора](https://fxtun.ru/docs/inspector)). Список `--allow-ip` повтору не мешает: запрос идёт прямо на `localhost:8000`.

Цикл отладки становится таким:

1. Один раз оплачиваете тестовый платёж, уведомление сохраняется в инспекторе.
2. Правите код и перезапускаете обработчик.
3. Нажимаете Replay. Обработчик получает то же уведомление и снова сверяет платёж с API. Тестовый платёж никуда не делся, так что проверка проходит.

Инспектор доступен с тарифа Base и слушает только `127.0.0.1`. На бесплатном тарифе выручает ручной повтор. Возьмите `id` платежа из вывода обработчика, подставьте его в файл `note.json` по образцу уведомления из начала статьи и отправьте: `curl -X POST -H 'Content-Type: application/json' --data @note.json http://127.0.0.1:8000/yookassa`. Обработчик всё равно сверяет платёж с API, поэтому с настоящим тестовым `id` такой запрос обработается так же, как уведомление от ЮKassa. Подробнее о возможностях — в разборе [инспектора трафика](/blog/traffic-inspector-debug-requests/).

## Постоянный адрес

Без `--domain` каждый запуск туннеля даёт новый случайный поддомен, а ЮKassa продолжает слать уведомления на старый. Флаг `--domain shop-yk` решает это на любом тарифе, но пока туннель выключен, имя может занять кто-то другой. Закрепить его за собой можно резервом с тарифа Base:

```bash
fxtunnel domains add shop-yk
```

Для долгой работы подойдёт и свой домен вида `pay-dev.example.com`. Варианты описаны в статье [«Постоянный адрес туннеля»](/blog/permanent-tunnel-address/).

## Если уведомление не приходит

**Не отмечены события.** В разделе «Интеграция — HTTP-уведомления» сохранён URL, но не выбраны нужные события. Уведомления приходят только о тех, на которые вы подписались.

**Адрес устарел.** Туннель перезапустили без `--domain`, и поддомен сменился. Запускайте туннель с тем же именем или поправьте URL в кабинете.

**Обработчик отвечает не 200.** В выводе видно `check failed:` и код `500`. Частая причина — ключ не от того магазина: API отвечает `401`. Проверьте, что shopId и ключ взяты из тестового магазина, на который вы платите. Пока ответ не `200`, ЮKassa будет повторять уведомление в течение суток.

**Обработчик не запущен или слушает другой порт.** Туннель открыт на 8000, а обработчик не работает. Сверьте число в `fxtunnel http 8000` с портом в `yk_hook.py`.

**Запрос отбит списком IP.** Если туннель запущен с `--allow-ip`, а ЮKassa поменяла адреса, уведомления получат `403`. Сверьте список с документацией.

## Что дальше

- Как работают вебхуки вообще и как проверять их у других сервисов — [«Вебхук: как принять вебхук на своём компьютере»](/blog/webhooks-on-localhost/).
- Вебхуки Stripe и GitHub через туннель — [«Тестирование вебхуков через туннель»](/blog/webhook-testing-with-tunnel/).
- Инспектор, повтор и фильтры запросов — [«Traffic Inspector: отладка HTTP-запросов в реальном времени»](/blog/traffic-inspector-debug-requests/).
- Чтобы адрес в ЮKassa не менялся — [постоянный адрес туннеля](/blog/permanent-tunnel-address/).
