---
title: "Пишу свой ngrok на Go: платёжная система — от «бесплатно для всех» к подпискам"
date: "2026-02-04"
description: "Проект работает, пользователи есть, но денег — ноль. Все сидят на бесплатном плане без ограничений. Хостинг, домены, TLS-сертификаты — всё за мой счёт."
series: "Пишу свой ngrok на Go"
part: 19
image: "https://fxtun.ru/blog/diary/covers/19-payments.jpg"
---

## Контекст

4 февраля. Проект работает, пользователи есть, но денег — ноль. Все сидят на бесплатном плане без ограничений. Хостинг, домены, TLS-сертификаты — всё за мой счёт. Пора это менять.

Задача звучит просто: добавить платные планы. На практике — это база данных, платёжный шлюз, подписки, автопродление, email-уведомления, курсы валют и чеки по 54-ФЗ. И всё это за один день.

## Планы и лимиты

Первый вопрос: что именно ограничивать? Посмотрел, как делают ngrok, Cloudflare Tunnel, Tailscale. Выбрал четыре оси ограничений: количество туннелей, доменов, кастомных доменов и токенов. Плюс пропускная способность и доступ к инспектору трафика.

```go
type Plan struct {
    ID                  int64   `json:"id"`
    Slug                string  `json:"slug"`
    Name                string  `json:"name"`
    Price               float64 `json:"price"`
    MaxTunnels          int     `json:"max_tunnels"`
    MaxDomains          int     `json:"max_domains"`
    MaxCustomDomains    int     `json:"max_custom_domains"`
    MaxTokens           int     `json:"max_tokens"`
    MaxTunnelsPerToken  int     `json:"max_tunnels_per_token"`
    BandwidthMbps       int     `json:"bandwidth_mbps"`
    InspectorEnabled    bool    `json:"inspector_enabled"`
}
```

Пять планов: Free (3 туннеля, 1 домен, $0), Base ($5, 5 туннелей), Pro ($10, 15 туннелей), Business ($20, 50 туннелей) и скрытый Admin с безлимитом. Цены в долларах — но платить-то будут в рублях...

## Курс USD → RUB

Держать цены в долларах удобно — один прайс-лист на весь мир. Но Robokassa (а потом YooKassa) принимает только рубли. Значит, нужен конвертер.

Написал сервис обмена курсов с двумя API-источниками и фоллбэком:

```go
var apiURLs = []string{
    "https://api.exchangerate-api.com/v4/latest/USD",
    "https://open.er-api.com/v6/latest/USD",
}

const fallbackRate = 75.0
```

Курс кэшируется на час. Если оба API недоступны — используется захардкоженный фоллбэк 75 рублей за доллар. Грубо? Да. Но лучше продать подписку по примерному курсу, чем показать ошибку.

Интересная деталь — округление. Никто не хочет видеть цену «746.25₽». Округляю до ближайшей пятёрки:

```go
func roundToNearest5(n float64) float64 {
    return float64(int((n+2.5)/5) * 5)
}
```

$5 × 92.3 = 461.5 → 460₽. Аккуратно и психологически приятно.

## Robokassa — первый платёжный шлюз

Выбрал Robokassa — подключается быстро, документация понятная, для ИП без проблем. Схема классическая: формируешь ссылку на оплату с подписью, пользователь платит, Robokassa дёргает webhook.

Создал модели подписок и платежей в SQLite:

```sql
CREATE TABLE subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id),
    plan_id INTEGER NOT NULL REFERENCES plans(id),
    status TEXT NOT NULL DEFAULT 'pending',
    recurring INTEGER NOT NULL DEFAULT 0,
    current_period_start DATETIME,
    current_period_end DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE payments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id),
    subscription_id INTEGER REFERENCES subscriptions(id),
    invoice_id INTEGER UNIQUE,
    amount REAL NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

Invoice ID стартует с 100001 — чтобы номера выглядели солидно, а не как «Счёт №3».

Процесс оплаты: пользователь выбирает план → бэкенд создаёт pending-подписку и pending-платёж → генерирует ссылку на Robokassa → пользователь платит → webhook приходит на `/api/webhooks/payment` → проверяем подпись → активируем подписку → меняем план пользователя.

## Email-уведомления

Подписки без уведомлений — как будильник без звука. Бесполезно. Написал email-сервис с шестью шаблонами:

1. **Подписка скоро истекает** — за 7 дней до конца
2. **Подписка истекла** — понижен до Free
3. **Подписка продлена** — автоплатёж прошёл
4. **Ошибка продления** — карта не прошла
5. **Тариф изменён** — апгрейд или даунгрейд
6. **Оплата прошла** — подтверждение платежа

Шаблоны в стиле проекта — тёмный фон (#0a0b0f), зелёный акцент (#80ff00). Никаких безликих белых писем.

С SMTP был отдельный квест. Стандартный Go-пакет `net/smtp` поддерживает только PLAIN-аутентификацию. А мой хостинг (Beget) требует LOGIN. Пришлось написать свой `loginAuth`:

```go
type loginAuth struct {
    username, password string
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
    return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
    command := strings.TrimSpace(string(fromServer))
    if strings.Contains(command, "Username") {
        return []byte(a.username), nil
    }
    if strings.Contains(command, "Password") {
        return []byte(a.password), nil
    }
    return nil, nil
}
```

LOGIN вместо PLAIN — разница в три строки, а без этого письма просто не уходили.

## Планировщик подписок

Подписки живут своей жизнью. Истекают, продлеваются, меняются. Нужен фоновый процесс, который за этим следит. Написал планировщик с пятью задачами, которые крутятся каждый час:

```go
func (s *Scheduler) runChecks() {
    s.processExpiredSubscriptions()    // деактивировать просроченные
    s.processRecurringRenewals()       // автопродление за час до конца
    s.applyPlanChanges()               // отложенные смены тарифа
    s.sendExpirationReminders()        // напомнить за 7 дней
    s.cleanupStalePending()            // почистить зависшие платежи
}
```

Самое интересное — автопродление. За час до истечения подписки планировщик проверяет: есть ли сохранённый платёжный метод? Нет ли уже pending-платежа? Не бесплатный ли план? Если всё ок — инициирует автоплатёж через API платёжной системы. Если платёж прошёл — продлевает подписку ровно на месяц: `now.AddDate(0, 1, 0)`.

Архитектура событийная. Планировщик не знает про email — он просто эмитит события:

```go
type EventType string

const (
    EventSubscriptionExpiring   EventType = "subscription_expiring"
    EventSubscriptionExpired    EventType = "subscription_expired"
    EventSubscriptionRenewed    EventType = "subscription_renewed"
    EventSubscriptionRenewFailed EventType = "subscription_renew_failed"
    EventPlanChanged            EventType = "plan_changed"
)
```

Email-сервис подписывается на эти события и отправляет нужные письма. Хочешь добавить Telegram-уведомления? Подпишись на те же события. Ноль связанности.

## Checkout UI

На фронтенде — страница выбора плана с карточками. Рекомендуемый план подсвечивается. Цена отображается в рублях для русской локали и в долларах для остальных. Кнопка «Оплатить» редиректит на платёжный шлюз.

Важный UX-момент: если незалогиненный пользователь кликает «Купить» на лендинге — он попадает на логин, а после логина автоматически возвращается на чекаут. Реализовал через `localStorage.setItem('authRedirect', '/checkout?plan=123')`.

Для русских чисел написал плюрализацию: «1 туннель», «2 туннеля», «5 туннелей». Три формы вместо двух, как в английском. Мелочь, но без неё выглядит по-любительски.

## Грабли

**Robokassa и тестовый режим.** В тестовом режиме Robokassa присылает в callback `IsTest=1`. Я поначалу не проверял этот флаг в продакшене. Результат: можно было «оплатить» подписку тестовым платежом. Добавил проверку — в продакшене тестовые платежи отклоняются.

**Сумма в рублях.** Robokassa присылает в webhook сумму в рублях, а я хранил в базе сумму в долларах. При конвертации обратно получалась другая сумма из-за округления, и подпись не совпадала. Пришлось хранить рублёвую сумму — ту самую, которую отправил в Robokassa при создании платежа.

**Удаление зависших платежей.** Первая версия `cleanupStalePending()` удаляла платежи напрямую. Но у платежей есть внешний ключ на подписки. Если подписка тоже в статусе pending — foreign key constraint ломал удаление. Пришлось делать двухфазную очистку: сначала pending-подписки, потом pending-платежи.

**Публичная оферта.** Robokassa требует ссылку на публичную оферту на сайте. Без неё не активируют магазин. Добавил страницу `/offer` — но показываю её только на домене fxtun.ru, потому что оферта — штука юрисдикционная.

## Итог

4 февраля (v2.3–v2.12):
- **5 тарифных планов** с гибкими лимитами
- **Robokassa интеграция** с подписью, webhook и фискальными чеками
- **Автопродление** через планировщик с часовым интервалом
- **6 email-шаблонов** в стиле проекта
- **Курс USD/RUB** с двумя API, кэшем на час и фоллбэком 75₽
- **Событийная архитектура** для уведомлений
- **Двухфазная очистка** зависших платежей
```
d5b3ccf feat(web): add pricing section to landing page
f9b9d1a feat(web): add descriptive hints to pricing plan features
fb27fab feat(db): add subscription and payment models
8944821 feat(payment): add Robokassa integration module
f3b46cc feat(email): add email notification service
848aad2 feat(scheduler): add subscription lifecycle scheduler
aad3d0c feat(api): add payment and subscription endpoints
3a3b9e8 feat(web): add checkout and payment result pages
cec0050 feat(payment): add dynamic USD to RUB exchange rate
84f4a3e feat(payment): centralize price calculation on backend
b5a1698 improve subscriptions and payments management
86855c6 feat(payment): send email notification after successful payment
4d7920e feat(email): redesign email templates to match project style
```

Платёжная система — самая «бизнесовая» часть проекта. Чисто технически тут нет ничего космического. Но количество нюансов... Фискализация, часовые пояса, идемпотентность webhook, конвертация валют, двухфазное удаление. Каждый пункт — потенциальный баг, который стоит денег. Буквально.

В следующей части — как я мигрировал с Robokassa на YooKassa, превратил SPA в статический сайт и начал воевать с поисковиками за место в выдаче.
