---
title: "Туннелирование на Windows — PowerShell, WSL, fxTunnel"
date: 2026-03-27T14:00:00+03:00
draft: false
description: "Настройка localhost-туннелей на Windows: PowerShell, WSL2, Docker Desktop, файрвол и решение типичных Windows-проблем с сетью и портами."
tags: [windows, wsl, tunneling, fxTunnel, powershell, docker-desktop, developer-tools]
image: ""
rubrics: [guides]
need: free
cover: ["WSL2", "fxtunnel http", "win-demo.fxtun.ru"]
updated: 2026-09-27
---

## Зачем Windows-разработчикам нужны туннели

Если вы пишете .NET API в Visual Studio, гоняете Node.js в WSL2 или поднимаете контейнеры в Docker Desktop, рано или поздно локальный сервис нужно показать миру: протестировать вебхуки от Stripe или GitHub, продемонстрировать проект удалённому клиенту, отладить OAuth-коллбэк, подключить мобильное устройство к локальному бэкенду.

Сложность на Windows в том, что туннелирование затрагивает несколько слоёв. Ваш сервис может работать нативно в Windows, внутри дистрибутива WSL2 или в Docker-контейнере. У каждого окружения своя модель сетевого взаимодействия, а Windows Firewall добавляет ещё один уровень. Это руководство охватывает все сценарии: нативная установка через PowerShell, интеграция с WSL2, Docker Desktop, настройка файрвола и решение типичных Windows-проблем.

Общее введение в туннелирование — в статье [«Что такое туннелирование»](/blog/what-is-tunneling/).

## Установка fxTunnel на Windows (нативная)

fxTunnel работает нативно на Windows как самостоятельный `.exe`-файл. WSL и Docker не требуются. Установка через PowerShell занимает меньше минуты.

### Способ 1: однострочная команда в PowerShell

Откройте PowerShell (обычный или от администратора) и выполните:

```powershell
# Скачать и установить fxTunnel для Windows
Invoke-WebRequest -Uri "https://fxtun.ru/install.ps1" -UseBasicParsing | Invoke-Expression
```

Проверьте установку:

```powershell
fxtunnel version
```

### Способ 2: ручная загрузка

Скачайте бинарник для Windows со страницы релизов на GitHub и поместите его в директорию из `PATH`:

```powershell
# Скачать бинарник
Invoke-WebRequest -Uri "https://github.com/mephistofox/fxtun.dev/releases/latest/download/fxtunnel-windows-amd64.exe" -OutFile "$env:USERPROFILE\bin\fxtunnel.exe"

# Добавить в PATH для текущей сессии (или постоянно через свойства системы)
$env:PATH += ";$env:USERPROFILE\bin"

# Проверить
fxtunnel version
```

### Первый туннель на Windows

Запустите локальный веб-сервер (любой фреймворк, любой порт) и откройте туннель:

```powershell
# Пример: Node.js-сервер на порту 3000
fxtunnel http 3000
```

Вывод:

```
Connecting to fxtunnel server...
Tunnel established!
HTTP:  http://win-demo.fxtun.ru
HTTPS: https://win-demo.fxtun.ru
Forwarding to localhost:3000
Inspector: http://127.0.0.1:4040
```

Готово. Публичный URL активен, имеет валидный TLS-сертификат и перенаправляет запросы на ваш локальный сервер. Как fxTunnel автоматически обеспечивает TLS, описано в статье [«HTTPS на localhost»](/blog/https-localhost-development/).

## Туннелирование из WSL2

WSL2 (Windows Subsystem for Linux 2) запускает настоящее ядро Linux внутри легковесной виртуальной машины. Многие Windows-разработчики используют WSL2 как основное окружение для Node.js, Python, Ruby, Go и других Linux-стеков. Сетевая модель WSL2 имеет особенности, которые нужно учитывать.

### Как устроена сеть WSL2

WSL2 создаёт виртуальный сетевой адаптер с собственным IP-адресом. Сервисы внутри WSL2 привязаны к сети виртуальной машины, а не к хосту Windows напрямую. В Windows 10 и ранних сборках Windows 11 использовалась NAT-модель, при которой проброс localhost из Windows в WSL2 работал ненадёжно. В Windows 11 (22H2+) появился режим mirrored networking, который разделяет сетевой стек хоста с WSL2 и обеспечивает корректную работу localhost.

Что это значит для туннелирования:

- **Режим mirrored (Windows 11 22H2+):** сервисы в WSL2 доступны на `localhost` из Windows. Можно запускать fxTunnel как на Windows, так и внутри WSL2 — оба варианта работают.
- **Режим NAT (по умолчанию в Windows 10):** сервисы в WSL2 привязаны к отдельному IP. Проще запустить fxTunnel внутри WSL2. Если запускать fxTunnel на Windows, нужен IP-адрес WSL2.

### Вариант A: установка fxTunnel внутри WSL2 (рекомендуется)

Самый простой подход. Установите fxTunnel в дистрибутив WSL2 и запустите его там — никаких проблем с кросс-сетевым взаимодействием.

```bash
# Внутри терминала WSL2 (Ubuntu, Debian и т.д.)
curl -fsSL https://fxtun.ru/install.sh | bash
fxtunnel login   # один раз: вход в аккаунт fxtun.ru

# Запуск туннеля к серверу внутри WSL2
fxtunnel http 8080
```

Туннель подключается к сервису напрямую в рамках одного Linux-окружения. Никаких поисков IP, правил файрвола и неожиданностей.

### Вариант B: fxTunnel на Windows, туннель к WSL2

Если вы предпочитаете запускать fxTunnel нативно на Windows, а сервис WSL2 недоступен на `localhost`. Позиционный аргумент `fxtunnel http` принимает только номер порта, поэтому IP WSL2 указывается через `local_addr` в конфиг-файле:

```powershell
# Шаг 1: узнать IP-адрес WSL2
wsl hostname -I
# Пример вывода: 172.23.48.1
```

```yaml
# client.yaml
tunnels:
  - type: http
    local_addr: 172.23.48.1
    local_port: 8080
```

```powershell
# Шаг 2: запустить fxTunnel на Windows с этим конфигом
fxtunnel --config client.yaml
```

### Включение mirrored networking (Windows 11)

Если вы на Windows 11 и хотите максимально простой опыт, включите mirrored networking. Создайте или отредактируйте файл `%USERPROFILE%\.wslconfig`:

```powershell
# Создать .wslconfig с mirrored networking
@"
[wsl2]
networkingMode=mirrored
"@ | Out-File -FilePath "$env:USERPROFILE\.wslconfig" -Encoding UTF8
```

Затем перезапустите WSL:

```powershell
wsl --shutdown
wsl
```

В режиме mirrored сервисы внутри WSL2 напрямую доступны на `localhost` из Windows, и fxTunnel можно запускать с любой стороны.

## Docker Desktop на Windows

Docker Desktop для Windows запускает контейнеры внутри бэкенда WSL2 (или Hyper-V в старых конфигурациях). Порты контейнеров пробрасываются на `localhost` хоста Windows, поэтому туннелирование работает точно так же, как для нативного Windows-сервиса.

### Туннелирование Docker-контейнера

```powershell
# Запуск контейнера с пробросом порта
docker run -d -p 8080:80 --name my-app nginx

# Открытие туннеля к проброшенному порту
fxtunnel http 8080
```

Контейнер теперь доступен из интернета. Другие Docker-сценарии, включая docker-compose, разобраны в статье [«Docker + туннель: доступ к контейнерам из интернета»](/blog/docker-tunnel-expose-containers/).

### Docker Compose с fxTunnel

Готового образа клиента нет — `ghcr.io/mephistofox/fxtunnel` собирает **сервер**, не клиент, а позиционный аргумент `fxtunnel http` принимает только номер порта. Чтобы запустить туннель как отдельный сервис, соберите свой образ и передайте имя соседнего контейнера через `local_addr` в конфиге (подробный разбор — в статье [«Docker + туннель»](/blog/docker-tunnel-expose-containers/)):

```yaml
# tunnel/client.yaml
tunnels:
  - type: http
    local_addr: web
    local_port: 80
```

```yaml
version: "3.8"

services:
  web:
    image: nginx
    ports:
      - "8080:80"

  tunnel:
    build: ./tunnel
    depends_on:
      - web
```

```powershell
docker compose up
```

`tunnel` и `web` — сервисы одного docker-compose файла и по умолчанию оказываются в общей сети, поэтому `local_addr: web` резолвится обычным Docker DNS. Публичный URL появится в логах.

## Настройка Windows Firewall

Windows Firewall — самый частый источник проблем с туннелированием на Windows. fxTunnel создаёт исходящее TLS-соединение на порту 443, которое разрешено по умолчанию. Однако корпоративные окружения, сторонние антивирусы и пользовательские групповые политики могут его блокировать.

### Проверка соединения

```powershell
# Проверить исходящее подключение к серверу fxTunnel
Test-NetConnection -ComputerName tunnel.fxtun.ru -Port 443
```

Если `TcpTestSucceeded` показывает `True`, соединение не блокируется. Если `False` — нужно добавить исключение в файрвол.

### Добавление исключения

```powershell
# Запустите PowerShell от администратора
New-NetFirewallRule -DisplayName "fxTunnel" `
  -Direction Outbound `
  -Program "$env:USERPROFILE\bin\fxtunnel.exe" `
  -Action Allow `
  -Protocol TCP `
  -RemotePort 443
```

Укажите правильный путь к `fxtunnel.exe` в параметре `-Program`.

### Windows Defender и сторонние антивирусы

Некоторые антивирусы помечают неизвестные бинарники, совершающие исходящие подключения. Если fxTunnel блокируется или помещается в карантин:

1. Добавьте `fxtunnel.exe` в список исключений антивируса.
2. Для Windows Defender добавьте исключение через PowerShell:

```powershell
# Добавить исключение для бинарника fxTunnel
Add-MpPreference -ExclusionPath "$env:USERPROFILE\bin\fxtunnel.exe"
```

## Туннелирование разных протоколов на Windows

fxTunnel поддерживает HTTP, TCP и UDP на Windows с теми же командами, что и на Linux и macOS. Вот типичные сценарии Windows-разработки.

### HTTP: ASP.NET / .NET API

```powershell
# Ваше ASP.NET Core приложение работает на порту 5000
dotnet run --urls "http://localhost:5000"

# В другом терминале
fxtunnel http 5000
```

### TCP: SQL Server, PostgreSQL

```powershell
# Туннель к локальному экземпляру SQL Server (порт по умолчанию 1433)
fxtunnel tcp 1433

# Туннель к PostgreSQL
fxtunnel tcp 5432
```

### UDP: игровые серверы, VoIP

```powershell
# Туннель к игровому серверу
fxtunnel udp 27015
```

fxTunnel поддерживает все три протокола, включая UDP, которого нет у большинства аналогов; на бесплатном тарифе доступны HTTP и TCP, UDP открывается начиная с тарифа Base. Разница между протоколами разобрана в статье [«TCP и UDP туннелирование»](/blog/tcp-udp-tunneling-explained/).

## Решение типичных Windows-проблем

Что-то не работает? Большинство проблем с туннелированием на Windows делятся на три категории: блокировка файрволом и антивирусом, особенности сети WSL2 и конфликты портов. В таблице ниже -- самые распространённые ситуации.

| Проблема | Симптом | Причина | Решение |
|---|---|---|---|
| **Файрвол блокирует исходящее** | `fxtunnel` зависает или показывает таймаут | Windows Firewall или групповая политика блокирует неизвестные исходящие подключения | Добавьте правило файрвола для `fxtunnel.exe`, разрешающее исходящий TCP на порту 443 |
| **Антивирус помещает в карантин** | `fxtunnel.exe` исчезает после скачивания или блокируется при запуске | Windows Defender или сторонний антивирус помечает бинарник | Добавьте `fxtunnel.exe` в список исключений |
| **Сервис WSL2 недоступен** | `connection refused` при туннелировании к порту WSL2 из Windows | WSL2 использует NAT-сеть; сервис привязан к IP виртуальной машины, а не к localhost Windows | Запустите fxTunnel внутри WSL2, включите mirrored networking или используйте `wsl hostname -I` для получения правильного IP |
| **Порт уже занят** | `bind: address already in use` при запуске сервера | Другой процесс занимает порт | Выполните `netstat -ano \| findstr :8080`, чтобы найти PID, затем `Stop-Process -Id <PID>` |
| **Конфликт портов Docker Desktop** | Контейнер запускается, но проброс порта не работает | Hyper-V или другой сервис резервирует диапазон портов | Проверьте `netsh int ipv4 show excludedportrange protocol=tcp` и выберите другой порт |
| **Политика выполнения PowerShell** | Скрипт установки отказывается запускаться | PowerShell блокирует неподписанные скрипты по умолчанию | Выполните `Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy RemoteSigned` |
| **Ошибка DNS в WSL2** | fxTunnel внутри WSL2 не может разрешить `tunnel.fxtun.ru` | Сломана конфигурация DNS в WSL2 | Добавьте `nameserver 8.8.8.8` в `/etc/resolv.conf` внутри WSL2 или установите `[wsl2] dnsTunneling=true` в `.wslconfig` |
| **Медленный туннель в WSL2** | Высокая задержка или таймауты | Низкая производительность I/O при обслуживании файлов с `/mnt/c/` | Храните файлы проекта в Linux-файловой системе (`~/projects/`), а не на примонтированном диске Windows |

### Диагностика конфликтов портов

```powershell
# Найти, что использует порт 8080
netstat -ano | findstr :8080

# Пример вывода:
#   TCP    0.0.0.0:8080    0.0.0.0:0    LISTENING    12345

# Завершить процесс
Stop-Process -Id 12345
```

### Проверка связности с WSL2

```bash
# Внутри WSL2: убедитесь, что сервис работает
curl http://localhost:8080

# Из PowerShell Windows: проверьте доступность порта WSL2
wsl curl http://localhost:8080
```

### Проверка зарезервированных портов Hyper-V

Docker Desktop и Hyper-V иногда резервируют диапазоны портов, конфликтующие с вашим приложением:

```powershell
# Вывести зарезервированные диапазоны портов
netsh int ipv4 show excludedportrange protocol=tcp
```

Если ваш порт попадает в зарезервированный диапазон, выберите другой порт для приложения.

## Дополнительно: постоянные туннели и пользовательские домены

### Запуск fxTunnel как фонового процесса

На Windows можно запустить fxTunnel в фоновом режиме через `Start-Process`:

```powershell
# Запуск fxTunnel в фоне
Start-Process -NoNewWindow -FilePath "fxtunnel" -ArgumentList "http", "8080"
```

Для более надёжной настройки рассмотрите запуск fxTunnel как Windows-сервиса через `NSSM` (Non-Sucking Service Manager) или как задачу планировщика, запускаемую при входе в систему.

### Пользовательские домены

Бесплатный тариф назначает случайный поддомен при каждом запуске. Для стабильных URL -- эндпоинтов вебхуков, URI перенаправления OAuth, общих ссылок для разработки -- используйте пользовательский домен:

```powershell
fxtunnel domains custom add dev.yoursite.com --target win-dev
# DNS-запись у регистратора, затем:
fxtunnel domains custom verify dev.yoursite.com
fxtunnel http 8080 --domain win-dev
```

Пользовательские домены, инспектор запросов и повтор (replay) доступны от тарифа Base. Инспектор показывает каждый входящий запрос в реальном времени -- незаменимо при отладке вебхуков. От тарифа Pro доступно 15 одновременных туннелей для микросервисных архитектур.

### Инспектор трафика

Встроенный инспектор трафика позволяет просматривать, фильтровать и повторять HTTP-запросы, проходящие через туннель. На Windows это особенно полезно, поскольку инструменты вроде `tcpdump` недоступны из коробки. Подробный обзор -- в статье [«Инспектор трафика: отладка запросов в реальном времени»](/blog/traffic-inspector-debug-requests/).

## Лучшие практики туннелирования на Windows

Следуйте этим рекомендациям, чтобы избежать типичных проблем при туннелировании на Windows:

- **Используйте WSL2 для Linux-стеков.** Если ваше приложение предназначено для Linux в продакшене, разрабатывайте и туннелируйте из WSL2. Это устраняет кросс-платформенные сюрпризы и соответствует вашему продакшен-окружению.
- **Храните проекты в Linux-файловой системе WSL2.** Файлы в `/mnt/c/` (примонтированный диск Windows) имеют значительно худшую производительность I/O. Храните код в `~/projects/` внутри WSL2 для быстрой сборки и быстрых ответов туннеля.
- **Закрывайте туннели, когда они не нужны.** Нажмите `Ctrl+C`, когда закончите работу. Открытый туннель — это открытая дверь к вашей машине. Это касается всех ОС, но на Windows процесс туннеля может пережить закрытие терминала, если запущен в фоне.
- **Используйте тестовые данные.** Никогда не подключайте туннель к продакшен-базе данных и не используйте боевые API-ключи. Это универсальное правило, но его стоит повторить, учитывая, как легко fxTunnel позволяет открыть любой порт.
- **Выбирайте fxTunnel вместо SSH-туннелей на Windows.** SSH-туннелирование на Windows требует SSH-клиента, удалённого сервера, ручного проброса портов и не создаёт HTTPS-URL. fxTunnel делает всё одной командой. Почему -- разобрано в статье [«SSH-туннели vs современные инструменты»](/blog/ssh-tunnel-vs-modern-tools/).
