---
title: Быстрый старт — документация fxTunnel
description: Быстрый старт с fxTunnel: одной командой откройте доступ из интернета к локальному веб-серверу, SSH или DNS через HTTP-, TCP- или UDP-туннель.
---

Клиент установлен и токен сохранён ([установка](/docs/install), [аутентификация](/docs/auth))? Тогда туннель открывается одной командой. Он работает, пока запущен клиент; остановить — `Ctrl+C`.

## HTTP-туннель

Откройте локальный веб-сервер на порту 3000:

```bash
fxtunnel http 3000
→ Tunnel established!
→ HTTP:  http://lemon.fxtun.ru
→ HTTPS: https://lemon.fxtun.ru
→ Forwarding to localhost:3000
```

Адрес `https://lemon.fxtun.ru` доступен из интернета, запросы на него приходят на ваш `localhost:3000`. Поддомен выбирается случайно и при следующем запуске будет другим. Выбрать поддомен самому можно флагом `--domain`, если его не занял другой пользователь:

```bash
fxtunnel http 3000 --domain myapp
```

Подробнее — в разделе [«HTTP-туннели»](/docs/http).

## TCP-туннель

Откройте SSH-сервер:

```bash
fxtunnel tcp 22
→ Tunnel established!
→ TCP: fxtun.ru:12345
→ Forwarding to localhost:22
```

Порт на сервере назначается автоматически. Подключение к вашему SSH:

```bash
ssh user@fxtun.ru -p 12345
```

Подробнее — в разделе [«TCP-туннели»](/docs/tcp).

## UDP-туннель

Откройте локальный DNS-сервер:

```bash
fxtunnel udp 53
→ Tunnel established!
→ UDP: fxtun.ru:21234
→ Forwarding to localhost:53
```

Проверка:

```bash
dig @fxtun.ru -p 21234 example.com
```

> [!NOTE]
> UDP-туннели недоступны на бесплатном тарифе. На нём клиент напишет в лог ошибку `UDP tunnels are not available on your plan`.

Подробнее — в разделе [«UDP-туннели»](/docs/udp).

> [!TIP]
> Если токен не сохранён, передайте его прямо в команде: `fxtunnel http 3000 -t sk_fxtunnel_ваш_токен`.
