---
title: Конфигурационный файл — документация fxTunnel
description: Как описать несколько туннелей в fxtunnel.yaml и запустить их одной командой: создание файла, все параметры, порядок поиска, приоритет и переменные окружения.
---

Конфигурационный файл нужен, когда туннелей несколько или их параметры не хочется набирать каждый раз. Все туннели из файла запускаются одной командой.

## Создание файла

```bash
fxtunnel init
```

Мастер спросит тип туннеля, имя и локальный порт, для HTTP — поддомен, для TCP и UDP — удалённый порт, и сохранит `fxtunnel.yaml` в текущей папке. Если файл уже есть, туннели можно добавить к нему или перезаписать его. Перед этим войдите по токену — [«Аутентификация»](/docs/auth).

Файл можно написать и вручную:

```yaml
# fxtunnel.yaml
tunnels:
  - name: web
    type: http
    local_port: 3000
    subdomain: myapp
    basic_auth: "team:SharedPass123"

  - name: ssh
    type: tcp
    local_port: 22
    remote_port: 12222
    allow_ips:
      - 203.0.113.0/24
```

## Запуск

```bash
# fxtunnel.yaml из текущей папки
fxtunnel

# другой файл
fxtunnel --config path/to/config.yaml
```

Туннели работают, пока запущен клиент. Чтобы они работали в фоне, запустите их через [daemon-режим](/docs/daemon): `fxtunnel up`.

> [!NOTE]
> Команды `fxtunnel http`, `fxtunnel tcp` и `fxtunnel udp` конфигурационный файл не читают — они открывают один туннель с параметрами из флагов.

## Где клиент ищет файл

Без `--config` клиент берёт первый найденный:

1. `fxtunnel.yaml` в текущей папке;
2. `client.yaml` в текущей папке;
3. `configs/client.yaml`;
4. `~/.fxtunnel/client.yaml`.

## Параметры туннеля

Каждый элемент списка `tunnels`:

| Параметр | Описание | По умолчанию |
|---|---|---|
| `name` | Имя туннеля, видно в логах и инспекторе | — |
| `type` | `http`, `tcp` или `udp`; обязателен | — |
| `local_port` | Локальный порт, 1–65535; обязателен | — |
| `local_addr` | Адрес локального сервиса, если он не на этом компьютере, например IP контейнера | этот компьютер |
| `subdomain` | Поддомен, только для `http` | случайный |
| `remote_port` | Порт на сервере для `tcp` и `udp`; `0` — выбрать автоматически | `0` |
| `basic_auth` | Basic Auth в формате `user:password`, пароль от 8 символов, только для `http` | — |
| `allow_ips` | Разрешённые IP и подсети | все адреса |
| `auto_close` | Закрытие при простое: `30m`, `2h` | — |
| `max_lifetime` | Максимальное время жизни: `8h`, `7d` | — |

Параметры повторяют флаги команд — подробнее в разделах [«HTTP-туннели»](/docs/http) и [«TCP-туннели»](/docs/tcp). Пароль из `basic_auth` клиент отправляет на сервер в виде bcrypt-хеша.

## Остальные секции

```yaml
server:
  token: "sk_fxtunnel_ваш_токен"
  compression: true

reconnect:
  enabled: true
  interval: 5s
  max_attempts: 0

inspect:
  enabled: true
  addr: "127.0.0.1:4040"
```

| Параметр | Описание | По умолчанию |
|---|---|---|
| `server.address` | Адрес сервера `host:port`; без порта подставляется `4443` | `tunnel.fxtun.ru:443` |
| `server.token` | API-токен | — |
| `server.insecure` | Подключаться без TLS | `false` |
| `server.tls_verify` | Проверять сертификат сервера | `true` |
| `server.compression` | Сжимать трафик между клиентом и сервером | `true` |
| `reconnect.enabled` | Переподключаться после обрыва | `true` |
| `reconnect.interval` | Пауза перед повторной попыткой | `5s` |
| `reconnect.max_attempts` | Сколько попыток сделать; `0` — без ограничений | `0` |
| `inspect.enabled` | Включить инспектор | `true` |
| `inspect.addr` | Адрес инспектора | `127.0.0.1:4040` |
| `inspect.max_entries` | Сколько запросов хранить на каждый туннель | `1000` |
| `inspect.max_body_size` | Сколько байт тела сохранять | `262144` (256 КБ) |

Если сервер `tunnel.fxtun.ru:443` недоступен, клиент сам пробует запасной адрес `fxtun.ru:4443`. Как устроено переподключение — в [справочнике CLI](/docs/cli), об инспекторе — в разделе [«Инспектор трафика»](/docs/inspector).

## Приоритет настроек

Если параметр задан в нескольких местах, действует первый из списка:

1. флаги командной строки: `--server`, `--token`, `--no-inspect`, `--inspect-addr`, `--insecure`;
2. переменные окружения `FXTUNNEL_*`;
3. конфигурационный файл;
4. значения по умолчанию.

Токен и адрес сервера ищутся иначе: флаги `-t` и `-s`, затем `FXTUNNEL_TOKEN` и `FXTUNNEL_SERVER_ADDRESS`, затем токен, сохранённый командой `fxtunnel login`, и только потом `server.token` и `server.address` из файла.

## Переменные окружения

Имя переменной — путь к параметру с префиксом `FXTUNNEL_`, заглавными буквами, точки заменены на `_`:

```bash
export FXTUNNEL_TOKEN="sk_fxtunnel_ваш_токен"
export FXTUNNEL_SERVER_ADDRESS="tunnel.fxtun.ru:443"
export FXTUNNEL_SERVER_COMPRESSION="false"
export FXTUNNEL_RECONNECT_INTERVAL="10s"
export FXTUNNEL_RECONNECT_MAX_ATTEMPTS="5"
export FXTUNNEL_INSPECT_ENABLED="true"
export FXTUNNEL_INSPECT_ADDR="127.0.0.1:4041"
```

Токен задаётся переменной `FXTUNNEL_TOKEN`. Переменные для туннелей не поддерживаются — туннели описываются только в файле.

> [!WARNING]
> Команды `fxtunnel http`, `tcp` и `udp` понимают только `FXTUNNEL_TOKEN` и `FXTUNNEL_SERVER_ADDRESS`. Остальные переменные действуют на `fxtunnel` с конфигурационным файлом и на `fxtunnel up`.
