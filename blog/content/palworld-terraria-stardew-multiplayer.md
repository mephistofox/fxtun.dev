---
title: "Palworld, Terraria и Stardew Valley по сети с друзьями без белого IP"
date: 2026-11-12T10:00:00+03:00
description: "Как открыть друзьям сервер Palworld, Terraria или Stardew Valley без белого IP: какой порт и протокол нужен каждой игре, команда туннеля, подключение и тариф."
rubrics: [games]
need: udp
tags: [game-server, palworld, terraria, stardew-valley, udp, fxTunnel]
cover: ["3 игры", "fxtunnel tcp/udp", "fxtun.ru"]
coverHL: 1
---

Всем трём играм для игры через интернет нужен один открытый порт, но протоколы разные. Выделенный сервер Palworld слушает UDP-порт 8211, сервер Terraria — TCP-порт 7777, ферма Stardew Valley — UDP-порт 24642. Без белого IP такой порт открывает туннель fxTunnel: `fxtunnel tcp` для Terraria работает на бесплатном тарифе, `fxtunnel udp` для Palworld и Stardew Valley — с тарифа Base. Друзья подключаются к адресу `fxtun.ru` и порту, который выдал туннель.

| Игра | Порт по умолчанию | Протокол | Туннель | Тариф |
|---|---|---|---|---|
| Palworld, выделенный сервер | 8211 | UDP | `fxtunnel udp` | Base |
| Terraria | 7777 | TCP | `fxtunnel tcp` | бесплатный |
| Stardew Valley | 24642 | UDP | `fxtunnel udp` | Base |

Источники — в разделе каждой игры. Ниже для каждой: как запустить сервер, какую команду дать и что ввести друзьям.

## Сначала: клиент fxTunnel

Клиент ставится на компьютер, где работает сервер игры. На Windows в PowerShell:

```powershell
irm https://fxtun.ru/install.ps1 | iex
```

На Linux и macOS:

```bash
curl -fsSL https://fxtun.ru/install.sh | sh
```

Затем в новом окне терминала войдите в аккаунт, это делается один раз:

```bash
fxtunnel login
```

Два правила туннелей, которые пригодятся для всех трёх игр:

- **Порт снаружи выдаёт сервер fxTunnel.** Для TCP — из диапазона 10000–20000, для UDP — из 20001–30000. Номер снаружи может отличаться от порта игры на вашем компьютере.
- **После перезапуска `fxtunnel` порт новый**, если не задан флаг `--remote-port`. С флагом вы сами выбираете номер из диапазона, и друзьям не придётся каждый раз узнавать новый адрес. Выбранный номер может оказаться занят кем-то другим, тогда клиент ответит `Failed to connect: server rejected tunnel: port … is already in use`, и нужно взять другой.

{{< flow caption="Друг подключается к fxtun.ru и порту туннеля, клиент fxtunnel передаёт пакеты серверу игры на вашем компьютере." >}}
- {label: интернет, title: игра друга, sub: "fxtun.ru:порт"}
- {link: TCP или UDP, kind: udp}
- {label: сервер fxTunnel, title: порт туннеля, sub: "TCP 10000–20000, UDP 20001–30000"}
- {link: TLS, kind: tls}
- {label: ваш ПК, title: клиент fxtunnel, sub: fxtunnel udp / tcp, me: true}
- {link: TCP или UDP, kind: udp}
- {label: локально, title: сервер игры, sub: "Palworld, Terraria, Stardew"}
{{< /flow >}}

## Palworld: выделенный сервер

### Порт

[Официальная документация Palworld](https://docs.palworldgame.com/getting-started/requirements) в требованиях к серверу пишет: `UDP Port 8211 (Default, Changeable)`. Порт меняется аргументом запуска `-port=8211` ([аргументы сервера](https://docs.palworldgame.com/settings-and-operation/arguments)). Один порт, один UDP-туннель.

Там же требования к машине: рекомендуются 4 ядра и 16 ГБ памяти, сервер запустится и на 8 ГБ, но чаще будет падать от нехватки памяти.

### Установка

Palworld Dedicated Server бесплатен ([инструкция](https://docs.palworldgame.com/getting-started/deploy-dedicated-server)). В Steam найдите его в библиотеке, включив фильтр инструментов, и установите. Через SteamCMD:

```bash
steamcmd +login anonymous +app_update 2394010 validate +quit
```

Запуск на Windows — `PalServer.exe` из папки `steamapps\common\PalServer`, на Linux — `./PalServer.sh`. Пароль сервера задаётся параметром `ServerPassword` в `PalWorldSettings.ini` ([настройки](https://docs.palworldgame.com/settings-and-operation/configuration)).

### Туннель

Сервер Palworld знает свой порт из аргумента запуска. Как клиент игры поведёт себя, если номер снаружи отличается от номера на сервере, мы не проверяли, поэтому убираем сам вопрос: запускаем сервер на порту из диапазона UDP-туннелей и просим у fxTunnel ровно такой же номер. Например, 28211:

```bash
PalServer.exe -port=28211
```

На Linux — `./PalServer.sh -port=28211`. Если запускаете через Steam, допишите `-port=28211` в параметры запуска сервера. Затем туннель:

```bash
fxtunnel udp 28211 --remote-port 28211
```

```console
Connecting to fxtunnel server...
Tunnel established!
UDP: fxtun.ru:28211
Forwarding to localhost:28211
Inspector: http://127.0.0.1:4040
Ready to receive connections
```

Строку `Inspector:` для игры можно не замечать, она про HTTP-туннели.

### Подключение друзей

По [документации](https://docs.palworldgame.com/getting-started/connect-server) к выделенному серверу подключаются так: на экране списка серверов ввести IP-адрес и порт в поле под списком. Друг вводит `fxtun.ru:28211`. Если игра не примет имя, впишите вместо него IP-адрес: его покажет команда `nslookup -type=A fxtun.ru`.

Подключение по IP работает с ПК. Игрокам на Xbox и PS5, по [документации](https://docs.palworldgame.com/getting-started/about-server), нужен сервер, развёрнутый как community server, то есть видимый в списке серверов игры. Community server за туннелем мы не проверяли и здесь не описываем.

### Тариф

UDP-туннели работают с тарифа Base. На бесплатном тарифе сервер ответит:

```console
Failed to connect: server rejected tunnel: UDP tunnels are not available on your plan — upgrade to enable UDP
```

Это ответ сервера, а не сбой сети.

## Terraria

### Порт

[Вики Terraria](https://terraria.wiki.gg/wiki/Server) пишет прямо: сервер работает на TCP-порту 7777 по умолчанию. Порт меняется ключом запуска `-port <номер>`, пароль — ключом `-pass <пароль>` или строкой `password=` в файле настроек. TCP-туннель работает на бесплатном тарифе.

### Сервер

На Windows сервер входит в комплект игры: `TerrariaServer.exe` в папке Terraria (у Steam — `steamapps\common\terraria`). Для других систем файлы выделенного сервера скачиваются с terraria.org. Мир, порт и пароль можно передать ключами запуска, все они описаны на той же странице вики:

```bash
TerrariaServer.exe -world <путь к миру> -port 7777 -pass <пароль>
```

С ключом `-world` сервер загружает мир и сразу стартует.

### Туннель

Для TCP номер снаружи и внутри может различаться: друг просто вводит адрес и порт, которые выдал туннель. Чтобы адрес не менялся от запуска к запуску, попросим конкретный порт:

```bash
fxtunnel tcp 7777 --remote-port 17777
```

```console
Connecting to fxtunnel server...
Tunnel established!
TCP: fxtun.ru:17777
Forwarding to localhost:7777
Ready to receive connections
```

### Подключение друзей

Вики описывает вход на сервер через **Join via IP** в меню **Multiplayer** ([вики, Multiplayer](https://terraria.wiki.gg/wiki/Multiplayer)). Друг вводит там адрес `fxtun.ru` и порт `17777`. Если имя не подойдёт, впишите IP-адрес из `nslookup -type=A fxtun.ru`.

Если все друзья в Steam, туннель может не понадобиться вовсе: вики описывает вход через **Join via Steam** для друзей в Steam. Туннель нужен, когда кто-то играет не из Steam или хочется сервер, работающий без вашего участия.

### Тариф

Один TCP-туннель — бесплатный тариф. Второй туннель одновременно, например для второго мира или сервера другой игры, — уже тариф Base.

## Stardew Valley

### Порт

По [вики Stardew Valley](https://stardewvalleywiki.com/Multiplayer) у ПК-версии несколько способов подключения:

- **Steam и GOG Galaxy.** Фермы друзей в Steam видны на экране подключения. Есть и код приглашения: он появляется в настройках игры, и его вводит друг, у которого тоже Steam или GOG Galaxy.
- **По IP-адресу.** Вики оговаривает, что для этого может понадобиться открытый порт или виртуальная локальная сеть вроде Hamachi, и что по умолчанию используется UDP-порт 24642.

Если друзья заходят по приглашению Steam или GOG и всё работает, туннель не нужен. Туннель нужен для подключения по IP.

### Туннель

Здесь повезло: 24642 входит в диапазон UDP-туннелей 20001–30000. Значит, можно попросить у fxTunnel ровно этот порт, и номер снаружи совпадёт со стандартным:

```bash
fxtunnel udp 24642 --remote-port 24642
```

```console
Connecting to fxtunnel server...
Tunnel established!
UDP: fxtun.ru:24642
Forwarding to localhost:24642
Inspector: http://127.0.0.1:4040
Ready to receive connections
```

Совпадение важно: как сменить порт фермы или указать другой порт при подключении по IP, вики не говорит. Поэтому если 24642 окажется занят другим пользователем fxTunnel (`port 24642 is already in use`), подключение по IP через туннель не получится, остаются приглашения Steam и GOG.

### Хост и подключение

Хозяин фермы запускает игру: **Co-op → Host → Host New Farm**, не забыв построить по домику на каждого игрока, или открывает существующую ферму через Co-op. Пока хозяин не в игре или ферма не открыта для других, друзья к ней не подключатся. Туннель должен работать всё это время.

Друг на экране подключения выбирает вход по IP-адресу и вписывает `fxtun.ru`. Если игра не примет имя, нужен IP-адрес сервера fxTunnel, его покажет команда:

```bash
nslookup -type=A fxtun.ru
```

Порт указывать не нужно: он стандартный, 24642.

### Тариф

UDP — тариф Base, как и у Palworld.

## Кто может зайти

Адрес туннеля публичный: кто знает `fxtun.ru` и порт, тот может постучаться. Защита у всех трёх игр своя:

- Palworld — `ServerPassword` в `PalWorldSettings.ini`;
- Terraria — `-pass` при запуске сервера;
- Stardew Valley — домики: по вики на каждый построенный домик на ферму может зайти один игрок.

У туннелей есть и флаг `--allow-ip`: сервер fxTunnel пропустит только перечисленные адреса. Он работает у TCP и UDP:

```bash
fxtunnel udp 28211 --remote-port 28211 --allow-ip 203.0.113.10 --allow-ip 198.51.100.0/24
```

Список придётся обновлять, если у друга меняется домашний IP. Пароля на стороне туннеля (`--auth`) у TCP и UDP нет, он только для HTTP.

Особенность туннеля: сервер игры видит всех игроков с адреса `127.0.0.1`, ведь пакеты ему передаёт клиент `fxtunnel` на той же машине. В Stardew Valley команда `ban` принимает имя, ID или IP-адрес. Бан по IP здесь заблокирует общий адрес, то есть всех сразу. Вики пишет, что бан по имени, ID или IP действует одинаково, так что и бан по имени может задеть всех игроков за туннелем. Как это работает через туннель, мы не проверяли; прежде чем банить, имейте это в виду.

## Если друг не может подключиться

**Открыт не тот протокол.** Palworld и Stardew Valley — только `fxtunnel udp`, Terraria — только `fxtunnel tcp`.

**Номера не совпадают.** Для Palworld порт в `-port=` и в `--remote-port` должен быть одним числом. Для Terraria друг вводит порт из строки `TCP:`, а не 7777.

**Туннель перезапустили без `--remote-port`.** Порт стал другим, а друг стучится на старый.

**UDP на бесплатном тарифе.** Смотрите строку `Failed to connect: server rejected tunnel: UDP tunnels are not available on your plan`.

## Что дальше

- Как узнать порт любой игры и что делать, если их два, — [«Как играть с другом по сети в любую игру»](/blog/play-with-friends-over-internet/).
- Игра с двумя UDP-портами и приём с совпадающими номерами — [«Сервер Valheim для друзей»](/blog/valheim-server-friends/).
- Когда игре нужна общая локальная сеть, а не адрес, — страница [«Аналог Хамачи»](https://fxtun.ru/analog-hamachi).
