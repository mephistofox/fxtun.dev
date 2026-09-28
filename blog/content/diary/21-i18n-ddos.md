---
title: "Пишу свой ngrok на Go: i18n, DDoS-защита и откат на продакшене"
date: "2026-02-12"
description: "Домены вместо URL-префиксов делят сайт на языки, а самодельная DDoS-защита блокирует своих же клиентов и откатывается за час — урок на будущее."
series: "Пишу свой ngrok на Go"
part: 21
image: "https://fxtun.ru/blog/diary/covers/21-i18n-ddos.jpg"
---

## Контекст

11–12 февраля. Сайт живёт на двух доменах — fxtun.dev (английский) и fxtun.ru (русский). Но язык пока определяется только по настройкам браузера, а URL одинаковые на обоих доменах. Поисковики видят один и тот же контент на разных доменах — это плохо для SEO.

Плюс нарастающая тревога: сервер открыт в интернет, DDoS-защиты нет. Одна мысль не даёт покоя: что будет, если кто-то решит залить трафиком?

## Домен = язык

Идея простая: `fxtun.ru` → русский, `fxtun.dev` → английский. Без URL-префиксов, без кук. Домен определяет всё.

```typescript
export function getDomainLocale(): string | null {
  const host = window.location.hostname
  if (host === 'fxtun.ru' || host.endsWith('.fxtun.ru')) return 'ru'
  if (host === 'fxtun.dev' || host.endsWith('.fxtun.dev')) return 'en'
  return null
}
```

Фоллбэк для локальной разработки (localhost) — детекция по языку браузера. Русскоязычные страны (ru, uk, be) → русский, остальные → английский. Результат сохраняется в `localStorage`, чтобы не определять заново при каждом заходе.

Но SSG ничего не знает про домены — он рендерит страницы в Node.js при сборке. Поэтому нужны явные языковые маршруты:

```typescript
function langPrefixedRoutes(): RouteRecordRaw[] {
  const publicRoutes = ['/', '/login', '/register', '/offer']
  const result: RouteRecordRaw[] = []

  for (const locale of ['ru', 'en']) {
    for (const route of publicRoutes) {
      result.push({
        path: `/${locale}${route === '/' ? '' : route}`,
        component: routeComponent(route),
        meta: { forcedLocale: locale },
      })
    }
  }
  return result
}
```

Итого 9 пререндеренных роутов: корневые `/`, `/login`, `/register`, `/offer` + `/ru`, `/ru/login`, `/en/login`... Каждый — готовый HTML с правильным языком.

## llms.txt — сайт для ИИ

Новый стандарт: `llms.txt` — как `robots.txt`, но для AI-краулеров. Файл объясняет, что такое сайт, на человеческом языке.

Сделал два файла: `llms.txt` (3.5 КБ, краткая версия) и `llms-full.txt` (6.4 КБ, с архитектурой и сравнениями). В robots.txt разрешил 16 AI-ботов:

- OpenAI: GPTBot, ChatGPT-User, OAI-SearchBot
- Anthropic: ClaudeBot, anthropic-ai
- Google: Google-Extended, Google-CloudVertexBot
- Meta: Meta-ExternalAgent
- И ещё PerplexityBot, Applebot-Extended, Amazonbot, CCBot, Bytespider, cohere-ai, diffbot

Зачем? Если ChatGPT или Claude могут рекомендовать fxTunnel — это бесплатный маркетинг. В `llms.txt` явно прописал конкурентные преимущества, ценовые сравнения и команду быстрого старта.

## IndexNow

Чтобы поисковики узнавали об изменениях быстрее — подключил IndexNow. Два ключа (по одному на домен), два файла-верификатора:

- `fxtun.dev` → `1c290e9b7ace4a0ca39d173a08028ae4`
- `fxtun.ru` → `c7679285937613cc3f1f188a761e464d`

Теперь при обновлении страницы можно дёрнуть API IndexNow — и Bing, Yandex, Seznam узнают об изменении за минуты, а не за дни.

## Accessibility и производительность

Пока возился с SEO — заодно подчистил доступность. Добавил `aria-label` на кнопки без текста (копирование кода, закрытие меню), `aria-expanded` на мобильное меню.

Но интереснее — оптимизация анимаций. HowItWorksSection следил за курсором через `mousemove` для эффекта свечения карточек. На каждое движение — `getBoundingClientRect()` для каждой карточки. Тормоза на мобилках гарантированы.

Переписал: кэширую `getBoundingClientRect()` на `mouseenter`, а `mousemove` оборачиваю в `requestAnimationFrame`:

```typescript
let rafId: number | null = null
let cachedCards: { el: HTMLElement; rect: DOMRect }[] = []

function handleMouseMove(e: MouseEvent) {
  if (rafId) return
  rafId = requestAnimationFrame(() => {
    for (const { el, rect } of cachedCards) {
      el.style.setProperty('--mouse-x', `${e.clientX - rect.left}px`)
      el.style.setProperty('--mouse-y', `${e.clientY - rect.top}px`)
    }
    rafId = null
  })
}
```

Плюс анимация border-glow теперь запускается только при наведении, а не крутится бесконечно. ProtocolsSection похудел с 365 до ~80 строк CSS.

## DDoS-защита — великий провал

А теперь — самая драматичная часть. Решил добавить серьёзную DDoS-защиту. Написал три новых файла:

- `accept_rate.go` — ограничение новых подключений: 5 в секунду на IP
- `bandwidth.go` — ограничение пропускной способности по тарифу (token bucket)
- `ipban.go` — бан IP с экспоненциальным backoff при нарушениях

Плюс per-tunnel HTTP concurrency limit (семафор на 100 одновременных запросов), таймауты на чтение/запись, лимит тела запроса 100 МБ. Всего 873 строки нового кода.

Задеплоил. Через пять минут начались проблемы.

**Проблема 1: yamux и accept rate.** Клиент открывает одно TCP-соединение, но внутри него — десятки yamux-стримов. Каждый стрим — это «новое соединение» с точки зрения сервера. Accept rate limiter начал блокировать легитимные data-стримы от авторизованных клиентов.

**Проблема 2: bandwidth + concurrency = 429.** Token bucket для пропускной способности и семафор для конкурентности начали взаимодействовать непредсказуемо. Под нормальной нагрузкой клиенты получали перманентные 429 — семафор не отпускал слот, пока bandwidth limiter не разрешит передачу, а тот ждал, пока семафор отпустит...

**Проблема 3: connection pooling.** У клиента есть пул из нескольких yamux-сессий для балансировки нагрузки. Каждая сессия — отдельное TCP-соединение. Per-IP rate limiter видит это как атаку: «Один IP открывает пять соединений? Бан!»

Попытался исправить: добавил whitelist для авторизованных клиентов, ref-counting для сессий. Не помогло — слишком много edge cases.

Через час откатил всё. Один коммит:

```
3de9e1a revert(server): remove DDoS protection that blocked legitimate traffic
```

873 строки удалены. Сервер вернулся к состоянию v3.2.

## Уроки

DDoS-защита на уровне приложения — ловушка. Проблема в том, что приложение не видит «чистую» картину трафика. yamux-стримы, connection pools, TLS-рукопожатия — всё это создаёт паттерны, неотличимые от атаки для наивного rate limiter.

Что осталось работать — API rate limiting. Простой, IP-based, двухуровневый:

```go
if s.cfg.Web.RateLimit.Enabled {
    globalRL := newIPRateLimiter(s.cfg.Web.RateLimit.GlobalPerMin)
    authRL := newIPRateLimiter(s.cfg.Web.RateLimit.AuthPerMin)
}
```

Token bucket алгоритм (`golang.org/x/time/rate`), TTL 10 минут на неактивные лимитеры, очистка каждые 5 минут. Работает — потому что HTTP API имеет понятные паттерны: один запрос = один запрос.

DDoS-защиту на уровне сети нужно делать на уровне сети — nginx, Cloudflare, iptables. Не на уровне Go-приложения, которое уже приняло TCP-соединение.

## Security hardening

Параллельно с DDoS-фиаско сделал полезные вещи — серию из 12 security-фиксов:

1. Валидация DTO через `go-playground/validator`
2. Лимит массивов в sync API: 500 элементов
3. Host header injection — проверка `Request.Host` в install.sh endpoint
4. Secure-флаг на interstitial cookie
5. Per-IP UDP rate limiting с FNV-64a хэшем
6. TOTP key derivation через SHA-256 (если ключ не 32 байта)
7. Валидация URL загрузок: только HTTPS + whitelist хостов
8. Лимит импорта GUI: 10 МБ / 100 bundles
9. Truncation тела в инспекторе: 1 МБ в базе
10. Audit log retention: минимум 90 дней
11. CSP-заголовок с `unsafe-eval` для Vue 3 runtime compilation
12. HSTS: `max-age=31536000; includeSubDomains`

Security headers — в отдельном middleware:

```go
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
```

Каждый из этих 12 пунктов — отдельный вектор атаки, который теперь закрыт. Без фанфар, без 873 строк удалённого кода. Просто работает.

## Грабли

**`unsafe-eval` в CSP.** Vue 3 компилирует шаблоны в рантайме — это требует `eval()`. Без `unsafe-eval` в Content-Security-Policy сайт падает с ошибками в консоли. Неприятно, но альтернатива — пре-компиляция шаблонов, а это другая архитектура.

**Абсолютные URL в nginx.** Blog проксируется через nginx rewrite. Относительные URL в rewrites наследовали порт приложения (`:3000`), а не порт nginx (`:443`). Пришлось перейти на абсолютные URL: `https://fxtun.dev/blog/...`.

**Email+password логин.** Добавил форму email/пароль для входа. Через день удалил — мёртвый код. Аутентификация через OAuth (GitHub, Google) и так покрывает все сценарии. 153 строки добавлены и удалены за 12 часов. Иногда лучшее решение — откатить.

## Итог

11–12 февраля (v3.3):
- **Domain-based i18n** — `fxtun.ru` = русский, `fxtun.dev` = английский
- **9 пререндеренных роутов** с языковыми префиксами
- **llms.txt** — 3.5 КБ инструкций для AI-краулеров
- **IndexNow** — мгновенная индексация обновлений
- **Accessibility** — ARIA labels, оптимизация анимаций через RAF
- **DDoS-защита** — написана (873 строки), задеплоена, откачена за час
- **12 security fixes** — валидация, лимиты, headers, HSTS
- **API rate limiting** — двухуровневый, с TTL и очисткой
```
572c86f feat(web): add domain-based locale detection and lang-prefixed SSG routes
4c6788f feat(seo): update robots.txt and llms.txt for AI crawlers
ead92ba feat(web): i18n sync, SEO/GEO improvements, IndexNow key
1fd0148 fix(web): improve accessibility and landing page performance
5e94b67 feat(security): add comprehensive DDoS protection
d8d30cd fix(server): prevent rate limiter from blocking authenticated client sessions
3de9e1a revert(server): remove DDoS protection that blocked legitimate traffic
580cba8 fix(security): apply medium-priority security hardening
7f5e857 fix(security): comprehensive security hardening across server, API and frontend
907b1a5 fix(nginx,csp): use absolute URLs in blog rewrites, add unsafe-eval to CSP
d844738 fix(seo): emit structured data only on canonical route
60689dd refactor(web): remove unused email login and password change forms
```

Главный урок этой серии: не каждая защита защищает. DDoS-защита, которая блокирует легитимных пользователей — это не защита, а DoS на самого себя. Иногда 873 строк удалённого кода — лучший коммит за день.

Проект продолжает развиваться. 3 мажорных версии, 59 pull requests, от консольной утилиты до SaaS с платежами, SEO и международной аудиторией. А всё началось 24 декабря с двух строчек `io.Copy`.
