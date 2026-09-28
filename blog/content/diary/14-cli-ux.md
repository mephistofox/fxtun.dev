---
title: "Пишу свой ngrok на Go: CLI как продукт — init, логин через браузер и красивый вывод"
date: "2026-02-01"
description: "CLI превращается в продукт: интерактивный визард конфига, логин через браузер по образцу GitHub CLI и цветной вывод HTTP-запросов вместо голых логов."
series: "Пишу свой ngrok на Go"
part: 14
---

## Контекст
1 февраля. CLI работает, но UX хромает: чтобы запустить туннель, надо вручную скопировать API-токен из веб-панели, написать YAML-конфиг руками, вывод — голые логи без форматирования. Для утилиты, которую запускаешь раз в месяц — сойдёт. Для инструмента, который запускаешь каждый день — нет. Сегодня превращаю CLI из утилиты в продукт.

## fxtunnel init

Первая проблема — конфиг. Чтобы запустить туннель, нужен `client.yaml` с правильной структурой. Новичок открывает документацию, копирует пример, правит руками — и ошибается в отступах. YAML не прощает.

Решение — интерактивный визард. Запускаешь `fxtunnel init`, отвечаешь на вопросы, получаешь готовый конфиг:

```go
func runInit(cmd *cobra.Command, args []string) error {
    scanner := bufio.NewScanner(os.Stdin)
    _, _, ok := checkAuth()
    if !ok {
        fmt.Println("You are not logged in.")
        return fmt.Errorf("authentication required")
    }
    var tunnels []config.TunnelConfig
    for {
        fmt.Print("Type [http/tcp/udp] (default: http): ")
        tunnelType := readLine(scanner)
        if tunnelType == "" { tunnelType = "http" }
        fmt.Print("Local port: ")
        port, _ := strconv.Atoi(readLine(scanner))
        tunnel := config.TunnelConfig{Name: name, Type: tunnelType, LocalPort: port}
        tunnels = append(tunnels, tunnel)
        fmt.Print("\nAdd another tunnel? [y/N]: ")
        if !strings.HasPrefix(strings.ToLower(readLine(scanner)), "y") { break }
    }
    data, _ := yaml.Marshal(&projectConfig{Tunnels: tunnels})
    os.WriteFile(projectConfigFile, data, 0644)
    fmt.Printf("\n✓ Saved %s with %d tunnel(s)\n", projectConfigFile, len(tunnels))
    return nil
}
```

Обратите внимание на `checkAuth()` в начале — визард не запустится, пока не залогинишься. Нет смысла генерировать конфиг без токена.

Визард создаёт файл `fxtunnel.yaml` — отдельно от `client.yaml`. Приоритет конфигов: если в текущей директории есть `fxtunnel.yaml` — используется он. Нет — ищем `client.yaml`. Идея в том, что `fxtunnel.yaml` — это проектный конфиг, который лежит рядом с кодом и коммитится в репозиторий (без секретов — токен в keyring). А `client.yaml` — глобальный, для настроек по умолчанию.

## Логин через браузер

Вторая проблема — аутентификация. Старый флоу: открой веб-панель → залогинься → создай API-токен → скопируй → вставь в конфиг. Пять шагов, где на каждом можно споткнуться.

Как делает GitHub CLI: вводишь `gh auth login`, открывается браузер, логинишься там, CLI автоматически получает токен. Это называется device flow.

Паттерн такой:
1. CLI отправляет запрос на сервер: «хочу залогиниться»
2. Сервер возвращает `session_id` и URL для авторизации
3. CLI открывает браузер с этим URL
4. Пользователь логинится в браузере, подтверждает
5. CLI поллит сервер каждые 2 секунды: «уже подтвердили?»
6. Когда подтвердили — сервер возвращает токен

```go
func loginWithBrowser() error {
    resp, _ := http.Post(webURL+"/api/auth/device/code", "application/json", nil)
    var deviceResp struct {
        SessionID string `json:"session_id"`
        AuthURL   string `json:"auth_url"`
        ExpiresIn int    `json:"expires_in"`
    }
    json.NewDecoder(resp.Body).Decode(&deviceResp)
    fmt.Printf("\nOpen this URL in your browser:\n\n  %s\n\n", deviceResp.AuthURL)
    _ = openBrowser(deviceResp.AuthURL)

    deadline := time.Now().Add(time.Duration(deviceResp.ExpiresIn) * time.Second)
    for time.Now().Before(deadline) {
        time.Sleep(2 * time.Second)
        pollResp, _ := http.Get(webURL + "/api/auth/device/token?session=" + deviceResp.SessionID)
        var result struct { Status, Token string }
        json.NewDecoder(pollResp.Body).Decode(&result)
        pollResp.Body.Close()
        if result.Status == "authorized" {
            kr := keyring.New()
            kr.SaveCredentials(keyring.Credentials{Token: result.Token, ServerAddress: serverAddr})
            return nil
        }
    }
    return fmt.Errorf("authorization timed out")
}
```

Токен сохраняется в keyring — системное хранилище секретов (Keychain на macOS, Secret Service на Linux, Credential Manager на Windows). Не в файле, не в переменной окружения. Один раз залогинился — и забыл.

На стороне сервера пришлось добавить два эндпоинта (`/api/auth/device/code` и `/api/auth/device/token`) и страницу подтверждения в веб-панели. Пользователь видит: «CLI запрашивает доступ к вашему аккаунту. Подтвердить?» Нажимает кнопку — CLI получает токен.

## Управление доменами из CLI

Поддомены раньше можно было резервировать только через веб-панель. Но если я уже в терминале — зачем переключаться в браузер?

```
fxtunnel domains list           — список моих доменов
fxtunnel domains reserve myapp  — зарезервировать поддомен
fxtunnel domains delete myapp   — удалить резервацию
```

Реализация простая — REST-вызовы к API. Но для пользователя это меняет флоу: всё можно сделать, не покидая терминал.

## Красивый вывод

Последний штрих — вывод. Раньше при работе туннеля в консоль сыпались строки вида `[INFO] new connection from 1.2.3.4`. Полезно для дебага, бесполезно для повседневной работы.

Теперь для HTTP-туннелей — цветной вывод с методом, путём и временем ответа:

```go
if httpMethod != "" {
    elapsed := time.Since(reqStart).Milliseconds()
    var methodColor string
    switch httpMethod {
    case "GET":    methodColor = "\033[32m"
    case "POST":   methodColor = "\033[33m"
    case "PUT":    methodColor = "\033[34m"
    case "DELETE": methodColor = "\033[31m"
    default:       methodColor = "\033[90m"
    }
    fmt.Printf("  %s%s\033[0m %s \033[90m%dms\033[0m\n", methodColor, httpMethod, httpPath, elapsed)
}
```

GET — зелёный, POST — жёлтый, PUT — синий, DELETE — красный. ANSI escape-коды — никаких библиотек. Для такой задачи тянуть зависимость не хочется.

Выглядит примерно так:
```
  GET /api/users 12ms
  POST /api/login 45ms
  DELETE /api/sessions/5 8ms
```

Мелочь, но теперь видно, что происходит в туннеле, не копаясь в логах.

## Compile-time DefaultServerURL

Ещё одна мелочь, которая экономит время. Когда собираешь клиент для своего сервера — хочется, чтобы адрес сервера был «зашит» в бинарник. Чтобы пользователю не нужно было указывать `--server` каждый раз.

```bash
go build -ldflags "-X main.DefaultServerURL=tunnel.example.com:4443" ./cmd/client
```

Стандартный паттерн Go — `ldflags` для инъекции значений при компиляции. В коде — обычная переменная:
```go
var DefaultServerURL = "localhost:4443"
```

Если пользователь не указал сервер явно — используется значение из бинарника. Указал — его значение приоритетнее.

## Итог
1 февраля:
- **fxtunnel init** — интерактивный визард для создания конфига
- **Device flow login** — логин через браузер, как в GitHub CLI
- **fxtunnel domains** — управление поддоменами из терминала
- **Цветной вывод** — HTTP-запросы с цветовой кодировкой методов
- **Compile-time server URL** — адрес сервера зашит в бинарник
```
8945bb2 feat(cli): add checkAuth helper for keyring and home config
1b49eb8 feat(config): prioritize fxtunnel.yaml over client.yaml in CWD
bd610fd feat(cli): add interactive 'fxtunnel init' command
361e8f6 feat(cli): add compile-time DefaultServerURL variable
9b05c2b feat(api): add device flow endpoints for CLI browser auth
07585a1 feat(web): add CLI auth confirmation page for device flow
6a51afb feat(cli): add browser-based device flow to login command
e006978 feat(cli): add 'domains' command for subdomain management
6cda1bf feat(cli): add custom domains management to 'domains' command
4a84c12 feat(client): pretty CLI output with HTTP request logging
```
Ни одна из этих фич не меняет функциональность туннелей. Они меняют ощущение от использования. Разница между «работает» и «приятно пользоваться» — именно в таких вещах: не нужно копировать токены, не нужно писать YAML руками, вывод не требует парсинга глазами.

---

**В следующей части:** OAuth — GitHub и Google авторизация за один день.
