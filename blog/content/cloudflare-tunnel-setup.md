---
title: "Cloudflare Tunnel: настройка, ошибка 1033 и когда проще fxTunnel"
date: 2026-10-21T10:00:00+03:00
description: "Cloudflare Tunnel по шагам: быстрый туннель cloudflared, именованный туннель со своим доменом, ошибка 1033 и случаи, где проще взять fxTunnel."
rubrics: [compare]
need: free
tags: [cloudflare, comparison, tunneling, developer-tools, fxTunnel]
cover: ["cloudflared", "vs", "fxtunnel"]
coverHL: 1
---

Cloudflare Tunnel открывает локальный сервис в интернет через сеть Cloudflare. Программа `cloudflared` на вашей машине сама подключается к Cloudflare исходящим соединением, поэтому белый IP и проброс портов не нужны. Попробовать можно одной командой: `cloudflared tunnel --url http://localhost:8080` выдаёт случайный адрес на `trycloudflare.com` без регистрации. Для постоянного адреса нужен именованный туннель и домен, делегированный в Cloudflare. Ошибка 1033 значит одно: Cloudflare не видит ни одного работающего `cloudflared` для этого туннеля. Ниже — настройка обоих вариантов по [документации Cloudflare](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/), разбор 1033 и честное сравнение: где Cloudflare Tunnel удобнее, а где быстрее взять fxTunnel.

## Установка cloudflared

По [странице загрузок](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/downloads/): для Windows там лежат установщик MSI и исполняемый файл `.exe`, для macOS есть Homebrew:

```bash
brew install cloudflared
```

Для Linux Cloudflare даёт репозиторий пакетов и готовые `.deb`, `.rpm` и бинарники на [GitHub](https://github.com/cloudflare/cloudflared/releases). Есть и Docker-образ.

## Быстрый туннель: одна команда без аккаунта

Quick Tunnel, он же TryCloudflare, нужен, чтобы показать кому-то локальный сайт прямо сейчас:

```bash
cloudflared tunnel --url http://localhost:8080
```

`cloudflared` создаст случайный поддомен на `trycloudflare.com` и напечатает его в выводе. Аккаунт Cloudflare для этого не нужен.

Ограничения из [документации Quick Tunnels](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/trycloudflare/):

- одновременно обрабатывается не больше 200 запросов;
- Server-Sent Events (SSE) не поддерживаются;
- быстрые туннели предназначены только для тестирования и разработки;
- если в папке `.cloudflared` лежит `config.yaml`, быстрый туннель не запустится. На этом спотыкаются те, кто уже настраивал именованный туннель на этой машине.

Адрес случайный и новый при каждом запуске. Для вебхука, который нужно один раз прописать в настройках сервиса, это неудобно.

## Именованный туннель со своим доменом

Постоянный адрес вида `app.example.com` даёт именованный туннель. Главное условие: домен должен быть подключён к Cloudflare, то есть добавлен в аккаунт, а NS-записи у регистратора переключены на серверы Cloudflare ([документация](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/local-management/create-local-tunnel/)). Если домена нет или его DNS должен оставаться у текущего провайдера, этот путь закрыт.

Ниже вариант с настройкой из командной строки (locally-managed tunnel). Туннель можно создать и в панели Cloudflare, шаги там другие.

**1. Войдите в аккаунт.** Команда откроет браузер: войдите в Cloudflare и выберите домен. В папке `cloudflared` появится сертификат аккаунта `cert.pem`:

```bash
cloudflared tunnel login
```

**2. Создайте туннель:**

```bash
cloudflared tunnel create myapp
```

Команда напечатает UUID туннеля и создаст файл с учётными данными `<UUID>.json` в папке `.cloudflared`.

**3. Опишите, что куда вести.** Файл `config.yml` в папке `.cloudflared`:

```yaml
tunnel: <UUID>
credentials-file: /home/user/.cloudflared/<UUID>.json

ingress:
  - hostname: app.example.com
    service: http://localhost:8080
  - service: http_status:404
```

Последнее правило без `hostname` обязательно: файл с правилами `ingress` должен заканчиваться правилом «для всего остального» ([документация по файлу конфигурации](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/local-management/configuration-file/)). Проверить правила можно так:

```bash
cloudflared tunnel ingress validate
```

**4. Привяжите имя к туннелю.** Команда создаст DNS-запись в Cloudflare:

```bash
cloudflared tunnel route dns myapp app.example.com
```

**5. Запустите:**

```bash
cloudflared tunnel run myapp
```

Пока процесс работает, `https://app.example.com` ведёт на `localhost:8080`.

## Ошибка 1033: Cloudflare Tunnel error

Страница с кодом 1033 вместо сайта — самая частая жалоба. По [описанию ошибки](https://developers.cloudflare.com/support/troubleshooting/http-status-codes/cloudflare-1xxx-errors/error-1033/), туннель не подключён к сети Cloudflare: Cloudflare не находит ни одного исправного `cloudflared`, которому можно отдать запрос.

Начните с проверки статуса. В панели Cloudflare он виден в разделе **Networking → Tunnels**, из командной строки:

```bash
cloudflared tunnel list
```

Дальше по статусу:

| Статус | Что значит и что делать |
|---|---|
| Healthy | Туннель подключён, с ним всё в порядке |
| Inactive | `cloudflared` ни разу не запускали: установите и запустите его на машине с сервисом |
| Down | Процесс `cloudflared` не работает: запустите его снова, проверьте, что машина включена и не упала |
| Degraded | Туннель работает с проблемами: смотрите логи `cloudflared` и правила брандмауэра, которые могут мешать его соединениям |

На практике 1033 чаще всего значит, что `cloudflared tunnel run` запускали в терминале, а терминал закрыли, или машина ушла в сон. Для постоянной работы `cloudflared` умеет установить себя службой в Linux и Windows и агентом запуска в macOS ([документация](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/local-management/create-local-tunnel/)).

## Где у Cloudflare Tunnel ограничения

Cloudflare Tunnel сделан в первую очередь для HTTP. С остальным есть оговорки, и они записаны в документации.

**TCP, SSH, RDP требуют `cloudflared` у того, кто подключается.** В [списке протоколов](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/routing-to-tunnel/protocols/) рядом с HTTP и HTTPS есть TCP, SSH, RDP и SMB, но для не-HTTP сервисов конечный пользователь должен поставить `cloudflared` у себя. Для произвольного TCP он запускает на своей машине ([инструкция](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/non-http/cloudflared-authentication/arbitrary-tcp/)):

```bash
cloudflared access tcp --hostname tcp.example.com --url localhost:9210
```

и подключается уже к `localhost:9210`. Для коллеги-разработчика это нормально. Для друга, которого вы зовёте на игровой сервер, — лишний шаг.

**UDP в списке протоколов для публичных адресов нет.** TCP- и UDP-приложения Cloudflare открывает в сценарии частной сети, и подключаются к ней, как правило, через клиент Cloudflare One на устройстве пользователя ([пример в документации](https://developers.cloudflare.com/cloudflare-one/tutorials/mysql-network-policy/)). Открыть UDP-порт игрового сервера для всех по адресу и порту так не получится.

**Постоянный адрес — только со своим доменом в Cloudflare.** Без домена остаётся быстрый туннель со случайным адресом и лимитами.

## Когда проще fxTunnel

fxTunnel устроен похоже: клиент сам подключается к серверу и держит соединение, сервер принимает внешние подключения и передаёт их клиенту. Разница в том, что снаружи туннеля.

| | Cloudflare Tunnel | fxTunnel |
|---|---|---|
| HTTP без домена | Быстрый туннель, случайный адрес на `trycloudflare.com` | Случайный поддомен на `fxtun.ru` с HTTPS |
| Постоянный HTTP-адрес | Свой домен, делегированный в Cloudflare | Зарезервированный поддомен `myapp.fxtun.ru` или свой домен через CNAME, с тарифа Base |
| TCP для внешних клиентов | Нужен `cloudflared` на стороне клиента | Адрес `fxtun.ru:порт`, клиенту ничего ставить не нужно |
| UDP | Через частную сеть и клиент Cloudflare One | UDP-туннель на порт, с тарифа Base |

Отсюда случаи, где fxTunnel быстрее.

**Нужно показать localhost, а домена нет.** Поставить клиент, войти и открыть порт:

```bash
curl -fsSL https://fxtun.ru/install.sh | sh
fxtunnel login
fxtunnel http 8080
```

На Windows установка — `irm https://fxtun.ru/install.ps1 | iex` в PowerShell.

```console
Connecting to fxtunnel server...
Tunnel established!
  HTTP:  http://lemon.fxtun.ru
  HTTPS: https://lemon.fxtun.ru
Forwarding to localhost:8080
Ready to receive connections
```

Адрес случайный и меняется при каждом запуске. Постоянное имя закрепляется с тарифа Base, подробности — в статье [«Постоянный адрес туннеля»](/blog/permanent-tunnel-address/).

**Друзьям или коллегам нужен TCP-порт.** База данных для коллеги, SSH, сервер Minecraft Java:

```bash
fxtunnel tcp 25565
```

Клиент выдаст адрес вида `fxtun.ru:14215`, и к нему подключаются обычной программой, без дополнительных утилит. Пример с Майнкрафтом — в статье [«Майнкрафт по сети без Хамачи»](/blog/minecraft-multiplayer-without-hamachi/).

**Нужен UDP.** Игровые серверы, голосовые сервисы: `fxtunnel udp 19132` открывает UDP-порт, к которому подключаются напрямую. UDP-туннели работают с тарифа Base.

**Нет белого IP и доступа к роутеру.** Это работает и у Cloudflare, и у fxTunnel: оба клиента подключаются к серверу сами. Про то, как понять, что у вас серый адрес, — в статье [«Серый IP: как узнать и что делать»](/blog/grey-ip-what-to-do/).

На бесплатном тарифе fxTunnel работает один туннель одновременно, HTTP или TCP. Для нескольких сервисов сразу, UDP и инспектора запросов нужен тариф Base.

## Когда удобнее Cloudflare Tunnel

Честно о другой стороне.

- **Домен уже в Cloudflare**, и нужен сайт или API на `app.example.com`. Именованный туннель встаёт в существующую инфраструктуру, DNS-записи создаются одной командой.
- **Нужна авторизация через корпоративный вход** перед сервисом. У Cloudflare для этого есть свои средства контроля доступа, а у fxTunnel проще: пароль `--auth` для HTTP и список адресов `--allow-ip`.
- **Всё, что открываете, — HTTP**, а пользователи сервиса — ваши коллеги, которым не сложно поставить `cloudflared` для SSH.

Если же нужно быстро открыть порт игры, дать другу TCP-адрес или обойтись без своего домена, fxTunnel короче на несколько шагов.

## Что дальше

- Сравнение трёх инструментов по функциям — [«ngrok vs Cloudflare Tunnel vs fxTunnel»](/blog/ngrok-vs-cloudflare-vs-fxtunnel/).
- Что делать без белого IP — [«Серый IP: как узнать и что делать»](/blog/grey-ip-what-to-do/).
- Постоянный адрес на `fxtun.ru` и свой домен — [«Постоянный адрес туннеля»](/blog/permanent-tunnel-address/).
