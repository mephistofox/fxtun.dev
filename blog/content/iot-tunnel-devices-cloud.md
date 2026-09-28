---
title: "Туннели для IoT — устройства к облаку без IP"
date: 2026-03-07T14:00:00+03:00
draft: false
description: "Подключаем Raspberry Pi, ESP32 и IoT к облаку через туннель: SSH-доступ, MQTT-брокер, веб-интерфейс и мониторинг без статического IP."
tags: [iot, raspberry-pi, mqtt, tunneling, fxTunnel, esp32, developer-tools]
image: ""
rubrics: [guides]
need: udp
cover: ["Raspberry Pi", "fxtunnel tcp", "fxtun.ru:14322"]
updated: 2026-09-27
---

## Проблема IoT: устройства за NAT без публичного доступа

Raspberry Pi стоит на полке дома. ESP32-датчики разбросаны по складу. Контроллер умного дома сидит за CGNAT провайдера. Ни у одного из них нет статического IP и прямого доступа из интернета. Когда вам нужно зайти на Pi по SSH из офиса, отправить данные с датчика в облако или показать веб-интерфейс ESP32 коллеге -- сеть не пускает.

Это фундаментальная проблема IoT: устройства должны быть доступны извне, но сетевая инфраструктура активно этому мешает. [Туннелирование](/blog/what-is-tunneling/) решает задачу -- устройство само устанавливает исходящее соединение с публичным сервером и получает адрес, доступный откуда угодно.

{{< flow caption="Устройство само открывает исходящее соединение к серверу — по умолчанию по TLS; проброс портов на роутере не нужен, NAT и CGNAT не мешают." >}}
- {label: домашняя / офисная сеть, title: Raspberry Pi / ESP32, sub: "SSH, MQTT, веб — без публичного IP", me: true}
- {link: TLS, kind: tls}
- {label: fxTunnel Server, title: tunnel.fxtun.ru, sub: выдаёт публичные адреса}
{{< /flow >}}

## Традиционные решения и их проблемы

Без туннеля остаются три классических варианта: проброс портов на роутере, VPN или аренда статического IP. Все три сложны в настройке и создают дополнительные проблемы. Подробнее о компромиссах -- в статье [«Как открыть localhost из интернета»](/blog/expose-localhost-to-internet/).

**Проброс портов на роутере** требует доступа к настройкам роутера, публичного IP (не CGNAT) и ручной конфигурации для каждого устройства. Порт открыт всему интернету без шифрования — это прямая угроза безопасности для IoT-устройств, которые часто имеют слабую защиту.

**VPN (WireGuard, OpenVPN)** создаёт зашифрованный канал, но требует сервера с публичным IP, настройки на каждом устройстве и поддержки. Для одного Raspberry Pi — это избыточное решение. Для ESP32 — часто невозможное из-за ограничений памяти и процессора.

**Статический IP** стоит денег, привязывает к провайдеру и всё равно требует проброса портов и настройки файрвола. Многие провайдеры не предоставляют статический IP для домашних тарифов.

**Туннель** — это одна команда на устройстве. Никаких настроек роутера, никакого VPN-сервера, никакого статического IP. Устройство устанавливает исходящее TLS-соединение, а [fxTunnel](/blog/ssh-tunnel-vs-modern-tools/) выдаёт публичный адрес.

## Как туннели решают проблему IoT-связности

Клиент на IoT-устройстве открывает исходящее TLS-соединение с сервером fxTunnel. Сервер выделяет публичный адрес (URL для HTTP или хост:порт для TCP/UDP) и маршрутизирует входящий трафик через это соединение на локальный порт устройства. NAT и файрвол не мешают, потому что соединение идёт изнутри сети.

Что делает fxTunnel особенно полезным для IoT -- поддержка [TCP и UDP](/blog/tcp-udp-tunneling-explained/). Большинство IoT-протоколов (MQTT, CoAP, Modbus TCP) работают поверх TCP, а некоторые (CoAP, mDNS) -- поверх UDP. fxTunnel покрывает оба случая, в отличие от ngrok (только TCP/HTTP) или Cloudflare Tunnel (только HTTP на бесплатном тарифе).

## Сценарий 1: Удалённый доступ к Raspberry Pi

Нужно зайти по SSH на Raspberry Pi, который стоит дома, а вы -- в офисе? Это, пожалуй, самый распространённый IoT-сценарий для туннеля.

### SSH через TCP-туннель

```bash
# На Raspberry Pi: пробрасываем SSH-порт
fxtunnel tcp 22
# → fxtun.ru:14322
```

```bash
# С любого компьютера: подключаемся к Raspberry Pi
ssh -p 14322 pi@fxtun.ru
```

Готово. Вы получили SSH-доступ к Raspberry Pi без статического IP, без проброса портов и без VPN.

### Веб-интерфейс через HTTP-туннель

Если на Raspberry Pi работает веб-приложение (Home Assistant, OctoPrint, Pi-hole), откройте HTTP-туннель:

```bash
# Веб-интерфейс Home Assistant на порте 8123
fxtunnel http 8123
# → https://rpi-home.fxtun.ru
```

Публичный HTTPS-URL можно открыть в браузере с любого устройства — телефона, рабочего компьютера, планшета.

## Сценарий 2: MQTT-брокер через TCP-туннель

MQTT — основной протокол для IoT-сенсоров и актуаторов. Брокер (Mosquitto, EMQX) обычно работает на Raspberry Pi или сервере в локальной сети. Чтобы IoT-устройства из других сетей могли публиковать и получать сообщения, нужен публичный доступ к брокеру.

```bash
# На машине с MQTT-брокером
# Mosquitto слушает порт 1883 (TCP)
fxtunnel tcp 1883
# → fxtun.ru:11883
```

{{< flow caption="Датчики публикуют в брокер через TCP-туннель: каждое внешнее подключение открывает свой поток yamux внутри соединения клиента с сервером — по умолчанию оно защищено TLS." >}}
- {label: IoT-датчики, title: ESP32, sub: "публикуют по TCP"}
- {link: TCP, kind: tcp}
- {label: fxTunnel Server, title: tunnel.fxtun.ru}
- {link: TLS, kind: tls}
- {label: домашняя сеть, title: Mosquitto, sub: "localhost:1883", me: true}
{{< /flow >}}

Теперь ESP32 из любой сети может подключиться к брокеру:

```cpp
// Arduino / ESP32 — подключение к MQTT через туннель
#include <WiFi.h>
#include <PubSubClient.h>

const char* mqtt_server = "fxtun.ru";
const int mqtt_port = 11883;

WiFiClient espClient;
PubSubClient client(espClient);

void setup() {
  Serial.begin(115200);
  WiFi.begin("SSID", "password");
  while (WiFi.status() != WL_CONNECTED) delay(500);

  client.setServer(mqtt_server, mqtt_port);
  while (!client.connected()) {
    client.connect("esp32-sensor-01");
    delay(1000);
  }
}

void loop() {
  float temp = analogRead(34) * 0.1; // пример чтения датчика
  char payload[16];
  snprintf(payload, sizeof(payload), "%.1f", temp);
  client.publish("sensors/temperature", payload);
  client.loop();
  delay(5000);
}
```

## Сценарий 3: Веб-сервер ESP32

ESP32 часто используется с встроенным веб-сервером для конфигурации и мониторинга. Проблема — веб-интерфейс доступен только в локальной сети. Через туннель вы можете открыть его для удалённого доступа.

Поскольку ESP32 не может запустить fxTunnel напрямую (ограничения ресурсов), используйте промежуточный хост — Raspberry Pi или любой компьютер в той же сети. Позиционный аргумент `fxtunnel http` принимает только номер локального порта, поэтому IP ESP32 указывается через `local_addr` в конфиг-файле:

```yaml
# client.yaml на Raspberry Pi — ESP32 имеет IP 192.168.1.50, веб-сервер на порте 80
tunnels:
  - type: http
    local_addr: 192.168.1.50
    local_port: 80
```

```bash
fxtunnel --config client.yaml
# → https://esp32-web.fxtun.ru
```

Теперь веб-интерфейс ESP32 доступен по публичному HTTPS-адресу. Это полезно для удалённой отладки, конфигурации устройства и демонстрации прототипов.

## Сценарий 4: Дашборд мониторинга (Grafana / Node-RED)

На Raspberry Pi часто работают дашборды для мониторинга IoT-данных: Grafana (визуализация метрик), Node-RED (автоматизация потоков) или собственные веб-приложения. Чтобы дать доступ команде, откройте HTTP-туннель:

```bash
# Grafana на порте 3000
fxtunnel http 3000
# → https://iot-dashboard.fxtun.ru

# Node-RED на порте 1880
fxtunnel http 1880
# → https://nodered.fxtun.ru
```

Для сценария с несколькими сервисами на одном Raspberry Pi запустите отдельный туннель для каждого:

```bash
# Запускаем все туннели одновременно
fxtunnel tcp 22 &       # SSH-доступ
fxtunnel http 3000 &    # Grafana
fxtunnel http 1880 &    # Node-RED
fxtunnel tcp 1883 &     # MQTT-брокер
```

## Установка fxTunnel на Raspberry Pi

fxTunnel написан на Go и поставляется с arm64-бинарником из коробки. Установка одной командой работает на 64-битной Raspberry Pi OS (Pi 3/4/5, Zero 2 W); старые 32-битные модели на armv6/armv7 не поддерживаются.

### Установка

```bash
# Установка fxTunnel (скрипт сам определяет amd64/arm64)
curl -fsSL https://fxtun.ru/install.sh | bash

# Проверяем установку
fxtunnel version
```

### Автозапуск через systemd

Для IoT-устройств критично, чтобы туннель запускался автоматически при загрузке и перезапускался при сбоях. Создайте systemd-сервис:

```ini
# /etc/systemd/system/fxtunnel.service
[Unit]
Description=fxTunnel - IoT tunnel service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/fxtunnel tcp 22
Restart=always
RestartSec=5
User=pi
Environment=FXTUNNEL_TOKEN=your-auth-token-here

[Install]
WantedBy=multi-user.target
```

Активируйте сервис:

```bash
# Перечитываем конфигурацию systemd
sudo systemctl daemon-reload

# Включаем автозапуск при загрузке
sudo systemctl enable fxtunnel

# Запускаем сейчас
sudo systemctl start fxtunnel

# Проверяем статус
sudo systemctl status fxtunnel
```

### Несколько туннелей через systemd

Если нужно пробросить несколько портов, создайте отдельный сервис для каждого:

```ini
# /etc/systemd/system/fxtunnel-ssh.service
[Unit]
Description=fxTunnel SSH tunnel
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/fxtunnel tcp 22
Restart=always
RestartSec=5
User=pi
Environment=FXTUNNEL_TOKEN=your-auth-token-here

[Install]
WantedBy=multi-user.target
```

```ini
# /etc/systemd/system/fxtunnel-mqtt.service
[Unit]
Description=fxTunnel MQTT tunnel
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/fxtunnel tcp 1883
Restart=always
RestartSec=5
User=pi
Environment=FXTUNNEL_TOKEN=your-auth-token-here

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable fxtunnel-ssh fxtunnel-mqtt
sudo systemctl start fxtunnel-ssh fxtunnel-mqtt
```

## Безопасность IoT-туннелей

IoT-устройства — частая цель атак из-за слабых паролей, устаревшего ПО и отсутствия мониторинга. Туннель шифрует трафик через TLS, но безопасность устройства — ваша ответственность.

### Аутентификация и токены

Используйте токены аутентификации fxTunnel, чтобы только авторизованные клиенты могли создавать туннели:

```bash
# Указываем токен при запуске
fxtunnel tcp 22 --token=your-secure-token

# Или через переменную окружения (рекомендуется для systemd)
export FXTUNNEL_TOKEN=your-secure-token
fxtunnel tcp 22
```

### SSH-ключи вместо паролей

Для SSH-доступа к Raspberry Pi всегда используйте ключи вместо паролей:

```bash
# На Raspberry Pi: отключаем вход по паролю
sudo sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config
sudo systemctl restart sshd
```

### Файрвол на устройстве

Настройте ufw, чтобы ограничить доступ к портам:

```bash
# Устанавливаем ufw
sudo apt install ufw

# Разрешаем только SSH и необходимые порты
sudo ufw default deny incoming
sudo ufw allow ssh
sudo ufw allow 1883/tcp  # MQTT
sudo ufw enable
```

### Принцип минимального доступа

Открывайте через туннель только те порты, которые действительно нужны. Не пробрасывайте все порты устройства «на всякий случай». Закрывайте туннели к сервисам, которые не используются в данный момент.
