---
title: "Стайлгайд блога"
date: 2026-09-26T12:00:00+03:00
draft: true
description: "Проверочная страница со всеми компонентами текста: код, таблицы, плашки, цитаты, рисунки."
rubrics: [protocols]
---

Эта страница собирает по одному экземпляру каждого компонента текста — так их удобно смотреть рядом. Например, `fxtunnel tcp 5432` — это встроенный код прямо в абзаце.

## Заголовок второго уровня

Обычный абзац под H2, а ниже — заголовок третьего уровня.

### Заголовок третьего уровня

Список того, что проверяет эта страница:

- терминальные блоки кода с кнопкой копирования
- таблицы
- плашки note и цитаты pull
- рисунки со счётчиком и легаси-диаграмма

{{< note type="info" >}}
Это информационная плашка. Она поясняет что-то важное, но не критичное.
{{< /note >}}

{{< note type="warn" >}}
Это предупреждающая плашка. Она обращает внимание на риск.
{{< /note >}}

{{< pull >}}
Одна выделенная мысль на всю ширину колонки — цитата из текста статьи.
{{< /pull >}}

```bash
$ fxtunnel tcp 5432
```

```go
func main() {
	srv := server.New()
	srv.Listen(":4443")
	log.Println("fxtunnel server started")
}
```

| Протокол | Доставка | Пример |
|---|---|---|
| TCP | гарантирована | `fxtunnel tcp 5432` |
| UDP | без гарантий | `fxtunnel udp 27015` |
| HTTP | поверх TCP | `fxtunnel http 3000` |

{{< figure caption="Путь запроса от клиента до локального сервиса." >}}
psql → fxtun.ru:41234 → fxtunnel client → localhost:5432
{{< /figure >}}

{{< flow caption="Внешнее подключение приходит на публичный порт…" >}}
- {label: интернет, title: psql, sub: любой компьютер}
- {link: TCP, kind: tcp}
- {label: сервер fxTunnel, title: публичный порт, sub: "fxtun.ru:41234"}
- {link: TLS, note: "поток #7", kind: tls}
- {label: ваш ноутбук, title: клиент fxtunnel, sub: fxtunnel tcp 5432, me: true}
- {link: TCP, kind: tcp}
- {label: локально, title: PostgreSQL, sub: "localhost:5432"}
{{< /flow >}}

{{< arch caption="…" bus="TLS-соединения клиента · yamux" >}}
- group: сервер fxTunnel
  items: [{title: HTTP-роутер, sub: по поддомену}, {title: TCP/UDP-роутер, sub: по порту}]
- group: клиент на ноутбуке
  me: true
  items: [{title: Вход, sub: API-токен}, {title: Управление, sub: первый поток}]
{{< /arch >}}

{{< frame caption="…" payload="данные потока — Length байт" >}}
- {name: Version, bytes: 1}
- {name: Type, bytes: 1}
- {name: Flags, bytes: 2}
- {name: Stream ID, bytes: 4, hl: true}
- {name: Length, bytes: 4}
{{< /frame >}}

{{< lanes caption="…" >}}
- title: Поверх TCP
  tag: HTTP/2, yamux
  rows: {поток 1: "##..~~##", поток 2: "#x..~###", поток 3: "##..~###"}
  result: {bad: "Потерялся пакет потока 2 — ждут все три."}
- title: Поверх QUIC
  tag: HTTP/3
  rows: {поток 1: "########", поток 2: "#x.~####", поток 3: "########"}
  result: {good: "Ждёт только поток 2, остальные идут дальше."}
{{< /lanes >}}

## Ссылка на будущую статью

Опубликованная: {{< later "expose-localhost-to-internet" >}}как открыть localhost{{< /later >}}.
Ещё не вышедшая: {{< later "no-such-article-yet" >}}статья из будущего{{< /later >}}.
