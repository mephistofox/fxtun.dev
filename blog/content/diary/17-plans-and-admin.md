---
title: "Пишу свой ngrok на Go: планы, лимиты и админка 2.0"
date: "2026-02-02"
description: "Проект растёт, появляются реальные пользователи. До сих пор всё было бесплатно и без ограничений — любой мог создать сколько угодно туннелей, токенов, доменов."
series: "Пишу свой ngrok на Go"
part: 17
---

## Контекст

2 февраля. Проект растёт, появляются реальные пользователи. До сих пор всё было бесплатно и без ограничений — любой мог создать сколько угодно туннелей, токенов, доменов. Щедро? Да. Жизнеспособно? Нет. Один пользователь может занять все порты, а у сервера ресурсы не бесконечные.

Нужна система тарифных планов. И заодно — нормальная админка, потому что текущая выглядит как прототип из первой недели.

## Система планов

Начал с модели. План — это набор лимитов: сколько туннелей, доменов, токенов может создать пользователь.

```go
type Plan struct {
    ID                 int64   `json:"id"`
    Slug               string  `json:"slug"`
    Name               string  `json:"name"`
    Price              float64 `json:"price"`
    MaxTunnels         int     `json:"max_tunnels"`
    MaxDomains         int     `json:"max_domains"`
    MaxCustomDomains   int     `json:"max_custom_domains"`
    MaxTokens          int     `json:"max_tokens"`
    MaxTunnelsPerToken int     `json:"max_tunnels_per_token"`
    InspectorEnabled   bool    `json:"inspector_enabled"`
}
```

Пять уровней: free, base, pro, business, admin. Миграция с дефолтными значениями:

```sql
INSERT INTO plans (slug, name, price, max_tunnels, max_domains, max_custom_domains,
    max_tokens, max_tunnels_per_token, inspector_enabled) VALUES
    ('free',     'Free',     0,  3,  1,  0,  1,  3, 0),
    ('base',     'Base',     5,  5,  5,  1,  5,  5, 1),
    ('pro',      'Pro',     10, 15, 15,  5, 10, 10, 1),
    ('business', 'Business', 20, 50, 50, 50, 50, 50, 1),
    ('admin',    'Admin',    0, -1, -1, -1, -1, -1, 1);
```

Минус один — магическое значение «без ограничений». Админ может всё. Free — 3 туннеля, 1 домен, без кастомных доменов, без инспектора. Достаточно, чтобы попробовать продукт, но недостаточно для серьёзной работы.

## Enforce лимитов

Модель без enforcement — это просто таблица в базе. Лимиты нужно проверять в двух местах: на сервере (при создании туннелей) и в API (при создании токенов и доменов).

Утилита для проверки «безлимитности»:

```go
func IsUnlimited(v int) bool { return v < 0 }
```

Пример — проверка лимита токенов:

```go
maxTokens := 10
if user.Plan != nil && user.Plan.MaxTokens >= 0 {
    maxTokens = user.Plan.MaxTokens
}
tokenCount, _ := s.db.Tokens.Count(user.ID)
if tokenCount >= maxTokens {
    s.respondErrorWithCode(w, http.StatusForbidden, "MAX_TOKENS", "token limit reached")
    return
}
```

Тот же паттерн повторяется для доменов и туннелей. Если у пользователя нет плана — fallback на дефолтные значения. Если план есть и лимит не безлимитный — проверяем текущее количество.

Отдельная история — инспектор трафика. Это ресурсоёмкая фича (сохраняет тела запросов/ответов), поэтому на free-плане она отключена:

```go
func (s *Server) checkInspectorAccess(w http.ResponseWriter, user *auth.AuthenticatedUser) bool {
    if !user.IsAdmin && (user.Plan == nil || !user.Plan.InspectorEnabled) {
        s.respondErrorWithCode(w, http.StatusForbidden, "INSPECTOR_DISABLED",
            "inspector not available on your plan")
        return false
    }
    return true
}
```

Админы всегда имеют доступ — `IsAdmin` проверяется первым. Для остальных — смотрим план.

## Админка 2.0

Старая админка была функциональной, но некрасивой. `confirm('Вы уверены?')` и `alert('Готово')` — уровень лабораторной работы. Пора привести в порядок.

Что сделал:
- **Фильтры и поиск** во всех таблицах. Пользователи, токены, домены, инвайт-коды — везде можно искать и фильтровать по статусу.
- **i18n** — полный русский и английский для всех admin views. Раньше часть текстов была захардкожена.
- **Inline-подтверждения** вместо `confirm()`. Нажал «Удалить» — кнопка меняется на «Точно удалить?» с таймаутом. Никаких системных диалогов.
- **Карточки планов** с цветовой кодировкой. Free — серый, Pro — синий, Business — золотой. Сразу видно, кто на каком плане.
- **TLS-предупреждения** для кастомных доменов. Если домен без сертификата — жёлтая плашка.
- **Expandable timeline** для аудит-лога. Раньше лог был плоской таблицей. Теперь — компактный timeline, который раскрывается по клику.

## UX-улучшения

Помимо админки, обновил пользовательскую часть.

Дашборд теперь показывает hero-карточку с текущим планом и гридом лимитов. Видно, сколько из лимита использовано: «Туннели: 2 из 5». Прогресс-бар заполняется — интуитивно понятно.

Профиль получил отдельную страницу загрузок с инструкциями для каждой платформы. Раньше ссылки на скачивание были разбросаны по разным местам.

В GUI-клиенте добавил use-case карточки в пустом состоянии дашборда. Когда у пользователя нет туннелей — вместо пустой страницы показываем: «Откройте локальный веб-сервер», «Поделитесь SSH-доступом», «Тестируйте вебхуки». Каждая карточка — один клик для создания соответствующего туннеля.

Ещё мелочь: `--subdomain` как алиас для `--domain` в CLI. Пользователи путали эти флаги — теперь оба работают.

## Итог

2 февраля:
- **Система планов** — 5 уровней с лимитами на всё
- **Enforce лимитов** — проверки на сервере и в API
- **Админка 2.0** — фильтры, поиск, i18n, inline-подтверждения
- **UX** — дашборд, профиль, загрузки, use-case карточки
```
7a1ea10 feat(gui): add use-case template cards to dashboard empty state
672e709 feat(web): improve dashboard, tokens, domains, and downloads UX
bf34366 feat(server): add plans system with per-user limits
ef96a0d feat(api): add plan management endpoints and enforce plan limits
06c1a59 feat(web): add plan support, redesign profile, fix limits display
3754bfc feat(web): redesign all admin views with filters and full i18n
2c76c2b fix(web): show blocked account message on login and fix TOTP code check
7f44996 feat(web): show email instead of phone in admin users table
f3c229e feat(cli): add --subdomain alias for --domain on http command
```

Планы — это фундамент для монетизации. Пока без платёжной системы, но структура готова: добавить Stripe или что-то ещё — дело одного интеграционного слоя. А админка наконец-то выглядит как админка, а не как TODO-лист разработчика.
