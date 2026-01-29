<p align="center">
  <img src="assets/logo.png" alt="fxTunnel Logo" width="120" height="120">
</p>

<h1 align="center">fxTunnel</h1>

<p align="center">
  <strong>Самостоятельно размещаемый сервер обратных туннелей для доступа к localhost из интернета</strong>
</p>

<p align="center">
  <a href="https://github.com/mephistofox/fxtunnel/releases/latest"><img src="https://img.shields.io/github/v/release/mephistofox/fxtunnel?style=flat-square&color=brightgreen" alt="Релиз"></a>
  <a href="https://github.com/mephistofox/fxtunnel/actions"><img src="https://img.shields.io/github/actions/workflow/status/mephistofox/fxtunnel/release.yml?style=flat-square" alt="Сборка"></a>
  <a href="https://goreportcard.com/report/github.com/mephistofox/fxtunnel"><img src="https://goreportcard.com/badge/github.com/mephistofox/fxtunnel?style=flat-square" alt="Go Report Card"></a>
  <a href="https://github.com/mephistofox/fxtunnel/releases"><img src="https://img.shields.io/github/downloads/mephistofox/fxtunnel/total?style=flat-square&logo=github" alt="Загрузки"></a>
  <a href="https://ghcr.io/mephistofox/fxtunnel"><img src="https://img.shields.io/badge/docker-ghcr.io-blue?style=flat-square&logo=docker" alt="Docker"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT%20with%20Attribution-yellow?style=flat-square" alt="Лицензия"></a>
</p>

<p align="center">
  <a href="README.md">English</a>
</p>

---

## Что такое fxTunnel?

**fxTunnel** — это быстрое, самостоятельно размещаемое решение для обратного туннелирования, написанное на Go. Оно позволяет открыть доступ к локальным HTTP, TCP и UDP сервисам через интернет с использованием сервера под вашим полным контролем — без сторонних зависимостей, без ограничений по использованию, без привязки к вендору.

Разверните сервер на любом VPS, настройте wildcard DNS-запись, и ваша команда мгновенно получит защищённые публичные URL для локальных серверов разработки, тестирования вебхуков, IoT-устройств, SSH-доступа и многого другого.

### Почему fxTunnel?

| | fxTunnel | Облачные туннельные сервисы |
|---|---|---|
| **Владение данными** | Ваш сервер, ваш трафик | Трафик проходит через стороннюю инфраструктуру |
| **Стоимость** | Бесплатно и с открытым кодом | Бесплатные тарифы ограничены; платные быстро растут в цене |
| **Свои домены** | Полный контроль wildcard-поддоменов | Часто ограничено или платно |
| **Поддержка протоколов** | HTTP, TCP, UDP | Обычно только HTTP |
| **Лимиты подключений** | Нет (вы задаёте свои) | Ограничены или лимитированы |
| **Управление пользователями** | Встроенная веб-панель, инвайт-коды, 2FA | Зависит от сервиса |
| **GUI-клиент** | Кросс-платформенное десктопное приложение | Почти не встречается |

## Основные возможности

- **HTTP-туннели** — Открывайте локальные веб-сервисы по адресу `yourapp.tunnel.example.com` с автоматической маршрутизацией по поддоменам
- **TCP-туннели** — Проброс любого TCP-порта: SSH, базы данных, игровые серверы, RDP
- **UDP-туннели** — Проброс UDP-трафика для DNS, VoIP, игровых протоколов
- **Wildcard-домены** — Полная поддержка `*.yourdomain.com` через nginx + Let's Encrypt
- **Веб-панель администратора** — Управление пользователями, токенами, доменами и активными туннелями из встроенного интерфейса на Vue 3
- **Управление пользователями** — Регистрация по инвайт-кодам, двухфакторная аутентификация TOTP, API-токены с ограничениями
- **Десктопный GUI-клиент** — Кросс-платформенное приложение на Wails (Linux, macOS, Windows) с поддержкой системного трея
- **CLI-клиент** — Легковесный консольный клиент с YAML-конфигурацией и автоматическим переподключением
- **Мультиплексирование потоков** — Эффективные мультиплексированные соединения на базе [yamux](https://github.com/hashicorp/yamux) через одно TCP-подключение
- **Docker** — Официальный образ в GitHub Container Registry
- **Безопасность** — Предупредительные страницы для недоверенного трафика, TLS-терминация через nginx, токены с ограниченными правами

## Быстрый старт

### Установка

Скачайте последнюю версию из [Releases](https://github.com/mephistofox/fxtunnel/releases) или используйте Docker:

```bash
docker pull ghcr.io/mephistofox/fxtunnel:latest
```

Или соберите из исходников:

```bash
git clone https://github.com/mephistofox/fxtunnel.git
cd fxtunnel
make build
```

### Настройка сервера

1. Создайте конфигурационный файл:
```bash
cp configs/server.example.yaml configs/server.yaml
# Отредактируйте configs/server.yaml — укажите ваш домен и секреты
```

2. Запустите сервер:
```bash
./bin/fxtunnel-server --config configs/server.yaml
```

3. Настройте wildcard DNS-запись:
```
*.tunnel.example.com  →  A  →  IP_ВАШЕГО_СЕРВЕРА
```

### Использование клиента

Открыть локальный HTTP-сервер:
```bash
fxtunnel http 3000 --server tunnel.example.com:4443 --token sk_your_token
# → https://random-subdomain.tunnel.example.com
```

С указанием поддомена:
```bash
fxtunnel http 3000 --subdomain myapp --server tunnel.example.com:4443 --token sk_your_token
# → https://myapp.tunnel.example.com
```

Проброс TCP-порта (SSH, БД и т.д.):
```bash
fxtunnel tcp 22 --server tunnel.example.com:4443 --token sk_your_token
```

Проброс UDP-трафика:
```bash
fxtunnel udp 53 --server tunnel.example.com:4443 --token sk_your_token
```

Использование конфиг-файла:
```bash
fxtunnel --config configs/client.yaml
```

## Архитектура

```
                                    ИНТЕРНЕТ
                                        │
                    ┌───────────────────┼───────────────────┐
                    │                   │                   │
                    ▼                   ▼                   ▼
              *.domain.com         TCP-порты           UDP-порты
              (через nginx)       (динамические)      (динамические)
                    │                   │                   │
                    └───────────────────┼───────────────────┘
                                        │
                                        ▼
                            ┌───────────────────┐
                            │   fxtunnel-server  │
                            │                    │
                            │  • HTTP-маршрутизатор │
                            │  • TCP-менеджер    │
                            │  • UDP-менеджер    │
                            │  • Веб-панель      │
                            │  • REST API        │
                            └─────────┬──────────┘
                                      │
                         yamux-мультиплексирование (TCP)
                                      │
              ┌───────────────────────┼───────────────────────┐
              │                       │                       │
              ▼                       ▼                       ▼
      ┌──────────────┐       ┌──────────────┐       ┌──────────────┐
      │   Клиент 1   │       │   Клиент 2   │       │   Клиент N   │
      │ webapp:3000  │       │   ssh:22     │       │ dns:53/udp   │
      └──────────────┘       └──────────────┘       └──────────────┘
```

## Конфигурация

### Сервер (`server.yaml`)

```yaml
server:
  control_port: 4443      # Подключения клиентов
  http_port: 8080         # HTTP-трафик туннелей
  tcp_port_range:
    min: 10000
    max: 20000
  udp_port_range:
    min: 20001
    max: 30000

domain:
  base: "tunnel.example.com"
  wildcard: true

web:
  port: 3000              # Панель администратора и API

auth:
  jwt_secret: "change-me"
  totp_key: "change-me"

database:
  path: "./data/fxtunnel.db"
```

### Клиент (`client.yaml`)

```yaml
server:
  address: "tunnel.example.com:4443"
  token: "sk_your_token"

tunnels:
  - name: "webapp"
    type: "http"
    local_port: 3000
    subdomain: "myapp"

  - name: "ssh"
    type: "tcp"
    local_port: 22

reconnect:
  enabled: true
  interval: 5s
```

### Переменные окружения

Все параметры конфигурации можно задать через переменные окружения с префиксом `FXTUNNEL_`:

```bash
export FXTUNNEL_AUTH_JWT_SECRET="your-secret"
export FXTUNNEL_SERVER_CONTROL_PORT=4443
export FXTUNNEL_DATABASE_PATH="./data/fxtunnel.db"
```

## Nginx + SSL

Для продакшена с HTTPS настройте nginx как TLS-терминирующий обратный прокси:

```nginx
server {
    listen 443 ssl http2;
    server_name *.tunnel.example.com;

    ssl_certificate /etc/letsencrypt/live/tunnel.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/tunnel.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 86400;
    }
}
```

Получение wildcard-сертификата:
```bash
certbot certonly --dns-cloudflare \
  --dns-cloudflare-credentials /etc/letsencrypt/cloudflare.ini \
  -d tunnel.example.com \
  -d *.tunnel.example.com
```

## Docker

```bash
docker run -d \
  --name fxtunnel \
  -p 4443:4443 \
  -p 8080:8080 \
  -p 3000:3000 \
  -p 10000-20000:10000-20000 \
  -v ./data:/app/data \
  -v ./configs/server.yaml:/app/configs/server.yaml \
  ghcr.io/mephistofox/fxtunnel:latest
```

## Сборка из исходников

```bash
make build          # Собрать CLI-клиент + сервер
make server         # Только сервер
make client         # Только CLI-клиент
make gui            # Десктопный GUI-клиент (требуется Wails)
make web            # Веб-панель на Vue 3
make test           # Запуск тестов
make build-all      # Полная сборка: web + сервер + клиенты для всех платформ
```

**Требования:** Go 1.24+, Node.js 20+ (для веб-интерфейса и GUI-клиента)

## Протокол

fxTunnel использует собственный протокол с JSON-сообщениями и префиксом длины поверх TCP с мультиплексированием потоков [yamux](https://github.com/hashicorp/yamux):

```
┌──────────────┬──────────────────────────────┐
│ Длина (4Б)   │ JSON-полезная нагрузка        │
│ big-endian    │                              │
└──────────────┴──────────────────────────────┘
```

## Участие в разработке

Мы приветствуем вклад в проект! Пожалуйста, сначала создайте issue для обсуждения предлагаемых изменений.

## Лицензия

MIT с требованием атрибуции — см. [LICENSE](LICENSE).

При любом использовании, развёртывании или распространении необходимо указать:
- **GitHub:** [github.com/mephistofox/fxtunnel](https://github.com/mephistofox/fxtunnel)
- **Сайт:** [mfdev.ru](https://mfdev.ru)
