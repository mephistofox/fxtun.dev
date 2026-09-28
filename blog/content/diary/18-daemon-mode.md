---
title: "Пишу свой ngrok на Go: daemon mode — туннели как сервис"
date: "2026-02-02"
description: "CLI-клиент работает, но каждый туннель — отдельный процесс. Хочу открыть три туннеля — запускаю три терминала. Закрыл терминал — потерял туннель."
series: "Пишу свой ngrok на Go"
part: 18
---

## Контекст

2 февраля. CLI-клиент работает, но каждый туннель — отдельный процесс. Хочу открыть три туннеля — запускаю три терминала. Закрыл терминал — потерял туннель. Это неудобно.

Хочу как у ngrok: один фоновый процесс управляет всеми туннелями. Примерно так:

```bash
fxtunnel up                  # daemon стартует фоном
fxtunnel http 3000           # туннель добавляется к работающему daemon
fxtunnel tcp 22              # ещё один
fxtunnel status              # что сейчас запущено
fxtunnel down                # всё останавливается
```

Один процесс, один yamux-сеанс, много туннелей. Закрыл терминал — daemon продолжает работать.

## State file

Первый вопрос: как CLI узнает, что daemon уже запущен? Нужен файл состояния. Daemon при старте записывает свои координаты:

```go
type State struct {
    PID       int       `json:"pid"`
    APIAddr   string    `json:"api_addr"`
    Server    string    `json:"server"`
    StartedAt time.Time `json:"started_at"`
}

func DefaultStatePath() string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".fxtunnel", "daemon.json")
}
```

PID — чтобы проверить, жив ли процесс. APIAddr — адрес локального HTTP API, через который CLI общается с daemon. Server — к какому серверу подключён. StartedAt — для информации.

## Проверка liveness

Наличие файла ещё ничего не значит. Daemon мог упасть, а файл остался. Нужно проверить, что процесс реально жив.

На Unix это просто: `os.FindProcess(pid)` всегда возвращает процесс (даже несуществующий), но `process.Signal(syscall.Signal(0))` вернёт ошибку, если процесса нет. Сигнал 0 — специальный: ничего не делает, только проверяет существование.

Если процесс мёртв — удаляем stale state file и считаем, что daemon не запущен.

## Локальный HTTP API

Daemon слушает на localhost на случайном порту. Четыре эндпоинта — больше не нужно:

```go
type TunnelManager interface {
    GetTunnels() []TunnelInfo
    RequestTunnel(cfg config.TunnelConfig) (TunnelInfo, error)
    CloseTunnel(id string) error
    Shutdown()
}

func NewAPI(mgr TunnelManager, server string) *API {
    a := &API{mgr: mgr, server: server, started: time.Now()}
    a.mux.HandleFunc("GET /status", a.handleStatus)
    a.mux.HandleFunc("POST /tunnels", a.handleAddTunnel)
    a.mux.HandleFunc("DELETE /tunnels/{id}", a.handleRemoveTunnel)
    a.mux.HandleFunc("POST /shutdown", a.handleShutdown)
    return a
}
```

`GET /status` — список активных туннелей, аптайм, сервер. `POST /tunnels` — добавить туннель. `DELETE /tunnels/{id}` — убрать. `POST /shutdown` — остановить daemon.

Обратите внимание на `TunnelManager` — это интерфейс. Реальная реализация оборачивает клиент из `internal/client`, но для тестов можно подставить мок.

## Команды up / status / down

`fxtunnel up` — самая интересная часть. Нужно запустить процесс, который переживёт закрытие терминала. `os.StartProcess` с отсоединением от родителя:

```go
// Запускаем daemon как отдельный процесс
cmd := exec.Command(os.Args[0], "daemon", "--server", serverAddr, "--token", token)
cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
cmd.Start()
```

После запуска — поллим state file. Daemon записывает его, когда готов принимать соединения. Как только файл появился и процесс жив — выводим «Daemon started».

`fxtunnel status` читает state file, дёргает `GET /status` и красиво выводит таблицу туннелей.

`fxtunnel down` — `POST /shutdown`, daemon завершается, state file удаляется.

## Daemon-aware tunnel creation

Ключевая идея: существующие команды `fxtunnel http`, `fxtunnel tcp`, `fxtunnel udp` должны работать в двух режимах. Если daemon запущен — делегируем ему. Если нет — работаем как раньше, в foreground.

```go
func addTunnelToDaemon(tunnelCfg config.TunnelConfig) bool {
    st, running := daemon.IsDaemonRunning(statePath)
    if !running { return false }
    body, _ := json.Marshal(daemon.AddTunnelRequest{
        Type: tunnelCfg.Type, LocalPort: tunnelCfg.LocalPort,
        Subdomain: tunnelCfg.Subdomain,
    })
    resp, _ := http.Post(fmt.Sprintf("http://%s/tunnels", st.APIAddr),
        "application/json", bytes.NewReader(body))
    var info daemon.TunnelInfo
    json.NewDecoder(resp.Body).Decode(&info)
    fmt.Printf("  Tunnel added: %s -> localhost:%d\n", info.URL, info.LocalPort)
    return true
}
```

Возвращает `true` — туннель добавлен через daemon, CLI может завершиться. Возвращает `false` — daemon не запущен, работаем в обычном режиме. Пользователь не замечает разницы — команда та же.

## Грабли

Права на state file. Первая версия — `0644`. Логично? В файле нет секретов, только PID и адрес. Но потом подумал: адрес локального API без аутентификации. Любой пользователь системы может прочитать его и управлять чужими туннелями. Поменял на `0600`.

А потом вернул `0644`, потому что на некоторых системах daemon запускается от одного пользователя, а CLI-команды — от другого (например, через sudo). И `0600` ломает этот сценарий. В итоге оставил `0644` — localhost API и так доступен только локально.

Ещё линтер заставил добавить `ReadHeaderTimeout` на HTTP-сервер. Без него возможна атака slowloris: открыл соединение, отправляешь заголовки по байту в секунду — сервер ждёт вечно. На localhost это некритично, но линтер прав — хорошие привычки не зависят от контекста.

## Интеграционные тесты

Daemon — штука stateful, и тестировать его нужно как целое. Написал lifecycle-тест:

1. Создаём API с мок-менеджером
2. Добавляем туннель через `POST /tunnels`
3. Проверяем `GET /status` — туннель на месте
4. Удаляем через `DELETE /tunnels/{id}`
5. Проверяем статус — пусто
6. `POST /shutdown` — daemon останавливается

Тест проходит за миллисекунды, потому что реальных сетевых подключений нет — только HTTP к localhost и мок вместо клиента.

## Итог

2 февраля:
- **State file** — daemon записывает PID и адрес API
- **Liveness check** — проверка, что процесс жив, а не только файл есть
- **Локальный HTTP API** — 4 эндпоинта для управления туннелями
- **up/status/down** — полный lifecycle из CLI
- **Daemon-aware режим** — существующие команды автоматически делегируют daemon
```
10cbae5 feat(daemon): add state file save/load helpers
d0008c5 feat(daemon): add process liveness and daemon running check
ba893d4 feat(daemon): add local HTTP API for daemon IPC
475a6fd feat(daemon): add ClientManager adapter for tunnel management
65d9d08 feat(cli): add up/status/down commands and daemon-aware tunnel creation
d26ed55 test(daemon): add integration test for daemon API lifecycle
fe17938 fix(daemon): fix lint issues — file perms, ReadHeaderTimeout, errcheck
```

Daemon mode меняет UX кардинально. Раньше: один терминал = один туннель. Теперь: `fxtunnel up` один раз, дальше добавляешь и убираешь туннели как хочешь. Закрыл ноутбук — daemon работает. Открыл — `fxtunnel status` показывает, что всё на месте.

В следующей части — посмотрим, что ещё можно улучшить. Проект не стоит на месте.
