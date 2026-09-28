---
title: Daemon-режим — документация fxTunnel
description: Как запустить туннели fxTunnel в фоне командой fxtunnel up, проверить их статус, добавить туннель на ходу, остановить демон и включить автозапуск через systemd.
---

В daemon-режиме клиент работает в фоне: туннели из [конфигурационного файла](/docs/config) остаются открытыми после закрытия терминала, а управлять ими можно командами `status` и `down`.

## Запуск

```bash
fxtunnel up
→ Daemon started (PID 12345)
→   HTTP: http://myapp.fxtun.ru
→   TCP: fxtun.ru:12222
→   Uptime: 1s
```

Демон читает туннели из того же файла, что и команда `fxtunnel`: сначала `fxtunnel.yaml` в текущей папке, затем `client.yaml` — порядок описан в разделе [«Конфигурационный файл»](/docs/config). Другой файл указывается флагом `--config`:

```bash
fxtunnel up --config path/to/config.yaml
```

Если демон уже запущен, команда ответит `Daemon is already running.` и второй не запустит.

> [!TIP]
> В фоне демон никуда не пишет ошибки. Если вместо адресов туннелей вы видите `Daemon started but status not available yet.`, запустите `fxtunnel up --foreground` — клиент останется в терминале и покажет, что пошло не так.

## Статус

```bash
fxtunnel status
→ Daemon running (PID 12345)
→ Server: tunnel.fxtun.ru:443
→   HTTP: http://myapp.fxtun.ru
→   TCP: fxtun.ru:12222
→   Uptime: 2h15m0s
```

Если демон не запущен, команда ответит `Daemon is not running.`

## Туннель на ходу

Пока демон работает, команды `fxtunnel http`, `tcp` и `udp` не открывают отдельное подключение, а добавляют туннель в демон и сразу возвращают терминал:

```bash
fxtunnel http 8080
→   Tunnel added: http://k3j9x2.fxtun.ru -> localhost:8080
```

Такой туннель не записывается в файл и пропадает, когда демон переподключается к серверу или перезапускается. Туннели, которые нужны постоянно, добавьте в `fxtunnel.yaml`.

## Остановка

```bash
fxtunnel down
→ Daemon stopped.
```

Демон закрывает все туннели и завершается.

## Особенности

- Демон всегда переподключается после обрыва связи, даже если в файле `reconnect.enabled: false`.
- Флаги `--no-inspect`, `--inspect-addr` и `--insecure` демон не учитывает: инспектор настраивается в секции `inspect` конфигурационного файла.

## Автозапуск через systemd

В Linux демон можно запускать вместе с системой. Токен удобно передать через файл окружения, доступный только вашему пользователю:

```bash
mkdir -p ~/.fxtunnel
echo 'FXTUNNEL_TOKEN=sk_fxtunnel_ваш_токен' > ~/.fxtunnel/env
chmod 600 ~/.fxtunnel/env
```

Файл службы `/etc/systemd/system/fxtunnel.service` — замените `user` на своё имя пользователя, а `/home/user/project` — на папку с `fxtunnel.yaml`:

```ini
[Unit]
Description=fxTunnel client
After=network-online.target
Wants=network-online.target

[Service]
User=user
WorkingDirectory=/home/user/project
EnvironmentFile=/home/user/.fxtunnel/env
ExecStart=/home/user/.local/bin/fxtunnel up --foreground
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now fxtunnel
```

Служба запускает клиент с `--foreground`: так systemd сам следит за процессом и перезапускает его после сбоя.
