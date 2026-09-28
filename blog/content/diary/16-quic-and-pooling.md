---
title: "Пишу свой ngrok на Go: QUIC — эксперимент, провал и план Б"
date: "2026-02-02"
description: "Эксперимент с QUIC вместо yamux закончился тройной деградацией скорости. Честный откат и план Б: пул из 15 соединений и бинарный формат заголовка."
series: "Пишу свой ngrok на Go"
part: 16
---

## Контекст
yamux работает. Работает хорошо — стримы, мультиплексирование, реконнект. Но я прочитал про QUIC: 0-RTT подключение, мультиплексирование на уровне протокола, нет head-of-line blocking. Звучит как идеальный транспорт для туннелей. Логичный следующий шаг... казалось бы.

## Правильная абстракция

Прежде чем втыкать QUIC, нужно подготовить код. Сервер и клиент работают напрямую с yamux — вызывают `session.OpenStream()`, `session.AcceptStream()`. Если я просто заменю yamux на QUIC — получу ту же связанность, только с другой библиотекой. Если завтра захочу попробовать третий транспорт — опять переписывать.

Решение — интерфейс:

```go
type Session interface {
    OpenStream(ctx context.Context) (Stream, error)
    AcceptStream(ctx context.Context) (Stream, error)
    Close() error
    IsClosed() bool
}
```

`Stream` — обёртка над `io.ReadWriteCloser` с парой дополнительных методов. `Listener` — принимает входящие сессии. Три интерфейса, и ни сервер, ни клиент не знают, что под капотом: yamux, QUIC или голубиная почта.

Адаптеры — тонкие обёртки:

```go
// yamux adapter
func (s *yamuxSession) OpenStream(ctx context.Context) (Stream, error) {
    stream, err := s.session.Open()
    if err != nil { return nil, err }
    return &yamuxStream{stream}, nil
}

// QUIC adapter
func (s *quicSession) OpenStream(ctx context.Context) (Stream, error) {
    stream, err := s.conn.OpenStreamSync(ctx)
    if err != nil { return nil, err }
    return &quicStream{stream}, nil
}
```

Четыре коммита: интерфейсы, yamux-адаптер, QUIC-адаптер, рефакторинг сервера и клиента на использование интерфейсов. Красиво, чисто. Код готов к эксперименту.

## QUIC: подключаем

`quic-go` — зрелая Go-реализация QUIC. Подключил как второй listener на сервере и как альтернативный транспорт на клиенте. Клиент пробует QUIC, если не получается — fallback на TCP+yamux.

Всё заработало с первой попытки. Туннели открываются, трафик идёт. Победа?

## Результат: 55 секунд против 19

Нет. Открываю браузер, загружаю страницу через QUIC-туннель. 55 секунд. Та же страница через yamux — 19 секунд. Тройная деградация.

Перепроверил. Несколько раз. Разные страницы, разные размеры. QUIC стабильно медленнее. Не на проценты — в разы.

## Почему QUIC проиграл

QUIC спроектирован для другой задачи. Он хорош, когда браузер открывает 50 параллельных запросов к серверу — каждый в своём стриме, и потеря пакета в одном стриме не блокирует остальные (нет head-of-line blocking). Это его главное преимущество над HTTP/2 поверх TCP.

Но туннельный трафик — это другая история. Каждое HTTP-соединение клиента уже изолировано в отдельном yamux-стриме. Head-of-line blocking на уровне TCP? Есть, но на практике при хорошем соединении — незаметен. А вот overhead QUIC — заметен:

- **TLS на каждом пакете.** QUIC шифрует каждый UDP-пакет отдельно. yamux работает поверх одного TLS-соединения — handshake один раз, дальше потоковое шифрование.
- **Congestion control.** QUIC реализует свой congestion control поверх UDP. Но внутри туннеля идёт TCP-трафик со своим congestion control. Два уровня управления потоком — конфликтуют.
- **UDP overhead.** UDP-пакеты меньше TCP-сегментов (нет потокового протокола), больше пакетов на тот же объём данных, больше заголовков.

Для CDN, где миллионы клиентов делают короткие запросы — QUIC выигрывает. Для туннеля, где одно долгоживущее соединение гоняет большие объёмы данных — TCP+yamux эффективнее.

## Честный реверт

```
02b2246 revert: remove QUIC transport, restore yamux-only operation
```

Один коммит — откат всего QUIC-кода. Абстракция `transport.Session` осталась (она полезна сама по себе), но QUIC-адаптер и listener — удалены.

Не стыдно. Эксперимент дал чёткий ответ: для этого use case QUIC — не то. Лучше потратить день и узнать, чем тащить медленный транспорт ради модного слова в README.

## План Б: connection pooling

Раз менять протокол не вариант — сделаем больше соединений. Раньше клиент подключался к серверу одной TCP-сессией. Все стримы мультиплексировались через неё. Одно соединение — одна точка bottleneck.

Теперь: помимо control-сессии, клиент открывает 15 дополнительных data-сессий. Каждая — отдельное TCP-соединение с yamux-мультиплексированием. Входящие запросы распределяются по сессиям round-robin:

```go
const dataConnectionCount = 15

func (c *Client) openDataConnections() {
    for i := 0; i < dataConnectionCount; i++ {
        go c.openDataConnection(i)
    }
}
```

15 — эмпирическое число. Тестировал с 5, 10, 15, 20. После 15 прирост производительности прекращался, а overhead от лишних соединений начинал расти.

## Бинарный StreamHeader

Раз уж оптимизируем — давайте по-настоящему. Раньше при открытии нового стрима сервер отправлял `NewConnectionMessage` — JSON с tunnel ID и адресом клиента. JSON-парсинг на каждое соединение — расточительно.

Заменил на бинарный формат — два length-prefixed строки:

```go
// Wire: [1B: tid_len][tid][1B: addr_len][addr]
func WriteStreamHeader(w io.Writer, tunnelID, remoteAddr string) error {
    buf := make([]byte, 1+len(tunnelID)+1+len(remoteAddr))
    buf[0] = byte(len(tunnelID))
    copy(buf[1:], tunnelID)
    buf[1+len(tunnelID)] = byte(len(remoteAddr))
    copy(buf[2+len(tunnelID):], remoteAddr)
    _, err := w.Write(buf)
    return err
}
```

Один байт — длина tunnel ID, потом сам ID, один байт — длина адреса, потом адрес. Никакого JSON, никакого парсинга. Один `Write`, один `Read`. На каждом соединении — экономия микросекунд. На тысячах соединений — заметно.

## v2.0 и принудительное обновление

Connection pooling и бинарный header — это breaking change. Старые клиенты не понимают новый формат, новый сервер не понимает старых клиентов. Выход один — мажорная версия.

```
8f86e75 feat!: bump to v2.0 — no backward compatibility with pre-1.17 clients
```

Восклицательный знак в conventional commit — `feat!` — означает breaking change. Сервер теперь сообщает клиенту `min_version` при подключении. Если клиент старше — автоматическое обновление (из статьи 13). Скачивает новый бинарник, заменяет себя, перезапускается. Пользователь видит: «Updating to v2.0...» — и через пару секунд всё работает.

Жёсткое решение, но необходимое. Тянуть обратную совместимость с бинарными протоколами — путь к бесконечным `if version < X` в каждом обработчике.

## Итог
2 февраля:
- **Transport abstraction** — интерфейсы Session/Stream/Listener
- **QUIC эксперимент** — 3x деградация, честный откат
- **Connection pooling** — 15 data-сессий с round-robin
- **Binary StreamHeader** — бинарный формат вместо JSON
- **v2.0** — breaking change с принудительным обновлением
```
0da2cf1 feat(transport): add multiplexed transport abstraction interfaces
90c8178 feat(transport): add yamux adapter implementing transport.Session
2d33c0d feat(transport): add QUIC adapter implementing transport.Session
13c68d3 refactor(server): use transport.Session interface instead of direct yamux
7310209 refactor(client): use transport.Session interface instead of direct yamux
50b4d99 feat(server): add QUIC listener alongside TCP/yamux
88f8e9f feat(client): add QUIC transport with automatic fallback to yamux
02b2246 revert: remove QUIC transport, restore yamux-only operation
8c51490 feat: add multi-session connection pooling with binary stream headers
07b6c15 feat(client): add forced auto-update when client version is below server min_version
8f86e75 feat!: bump to v2.0 — no backward compatibility with pre-1.17 clients
```
Главный урок: не каждая модная технология — правильный выбор. QUIC великолепен для HTTP/3 в браузерах. Для long-lived туннельных соединений — нет. Зато правильная абстракция позволила проверить гипотезу за день и откатить без боли. А connection pooling дал реальный прирост производительности без смены протокола. Иногда скучное решение — лучшее решение.

---

**В следующей части:** daemon mode — запуск туннелей в фоне и управление через CLI.
