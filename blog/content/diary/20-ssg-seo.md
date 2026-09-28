---
title: "Пишу свой ngrok на Go: SSG, SEO и война за поисковики"
date: "2026-02-08"
description: "Платёжная система работает, пользователи платят. Но откуда им вообще приходить? Google не знает о существовании fxTunnel."
series: "Пишу свой ngrok на Go"
part: 20
image: "https://fxtun.ru/blog/diary/covers/20-ssg-seo.jpg"
---

## Контекст

7–8 февраля. Платёжная система работает, пользователи платят. Но откуда им вообще приходить? Google не знает о существовании fxTunnel. Telegram-превью выглядит убого — белый квадрат и дефолтный title. А лендинг — это SPA на Vue, который поисковые боты видят как пустую страницу с `<div id="app"></div>`.

Задача на два дня: научить поисковики видеть сайт, научить мессенджеры показывать красивые превью, мигрировать с Robokassa на YooKassa и заодно переделать лендинг под SaaS-позиционирование.

## YooKassa вместо Robokassa

Начну с платежей — потому что это breaking change, и лучше сделать его первым.

Robokassa работала, но интеграция была примитивной: формируешь URL с подписью, редиректишь пользователя, ждёшь webhook. Без рекуррентных платежей, без сохранения карты. YooKassa — современнее: полноценный REST API, сохранение платёжных методов, автоплатежи.

Переписал весь платёжный модуль. Было ~279 строк robokassa.go, стало 381 строка yookassa.go. Ключевое отличие — идемпотентность. Каждый запрос к YooKassa идёт с `Idempotence-Key` (UUID), чтобы повторный запрос не создал дублирующий платёж:

```go
req.Header.Set("Idempotence-Key", uuid.New().String())
req.Header.Set("Content-Type", "application/json")
req.SetBasicAuth(c.shopID, c.secretKey)
```

Для фискализации по 54-ФЗ добавил receipt в каждый платёж — email покупателя, описание услуги, код НДС. Без этого YooKassa просто не принимает платежи:

```go
Receipt: &Receipt{
    Customer: ReceiptCustomer{Email: email},
    Items: []ReceiptItem{{
        Description:    fmt.Sprintf("fxTunnel %s subscription", planName),
        Quantity:       "1",
        Amount:         Amount{Value: formatAmount(amountRUB), Currency: "RUB"},
        VatCode:        1,  // без НДС (для самозанятых)
        PaymentSubject: "service",
        PaymentMode:    "full_payment",
    }},
},
```

Для webhook'ов — валидация IP. YooKassa шлёт уведомления только с определённых адресов. Захардкодил их список:

```go
var yookassaCIDRs = []string{
    "185.71.76.0/27",
    "185.71.77.0/27",
    "77.75.153.0/25",
    "77.75.154.128/25",
}
```

В тестовом режиме проверку IP пропускаю — иначе локальная разработка невозможна.

## Vite-SSG — SPA → статический сайт

Самая важная техническая задача. Vue SPA — это JavaScript, который рендерит HTML в браузере. Googlebot вроде умеет исполнять JS, но делает это медленно и не всегда. Telegram, Facebook, Slack — вообще не исполняют JS, берут только то, что есть в HTML.

Решение — Static Site Generation. При сборке рендерим каждую страницу в готовый HTML. Поисковик получает полную страницу, а Vue подхватывает её на клиенте (hydration).

Подключил `vite-ssg` — обёртка над Vite, которая заменяет `createApp` на `ViteSSG`:

```typescript
export const createApp = ViteSSG(
  App,
  {
    routes,
    base: import.meta.env.BASE_URL,
  },
  ({ router, isClient }) => {
    if (isClient) {
      // Auth guards — только на клиенте
      setTimeout(() => {
        router.beforeEach((to, from, next) => {
          // проверяем авторизацию
        })
      })
    }
  }
)
```

Важный момент — `setTimeout` для auth guards. При SSG роутер бежит по страницам на сервере (Node.js), где нет ни `localStorage`, ни `window`. Auth guards ломают пре-рендеринг. Оборачиваю в `setTimeout` — они выполнятся только на клиенте, после гидрации.

Генерируемые страницы: `/`, `/login`, `/register`, `/offer`. Плюс мультиязычные версии: `/ru`, `/ru/login`, `/en/login` — итого 12 пререндеренных HTML-файлов.

## JSON-LD — структурированные данные

Поисковики любят, когда им явно говоришь: «Это — организация, а это — программа, а вот тут — FAQ». Четыре схемы JSON-LD:

```typescript
function useOrganizationSchema() {
  useHead({
    script: [{
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'Organization',
        name: 'fxTunnel',
        url: 'https://fxtun.dev',
        logo: 'https://fxtun.dev/logo.png',
      })
    }]
  })
}
```

Плюс `SoftwareApplication` с тремя ценовыми предложениями (Free $0, Base $5, Pro $10), `WebSite` и `FAQPage` с 15 вопросами, сгенерированными из i18n-файлов.

Попался на дубликатах. Сначала рендерил JSON-LD на всех роутах: `/`, `/ru`, `/en`. Google Search Console начал жаловаться: «Duplicate FAQPage detected». Добавил проверку — структурированные данные только на каноническом роуте `/`:

```typescript
const isCanonicalRoute = route.path === '/'
if (isCanonicalRoute) {
  useOrganizationSchema()
  useSoftwareApplicationSchema()
  useFaqSchema(faqItems)
}
```

## OG-теги и мета

Для Telegram, Facebook, Slack нужны Open Graph теги прямо в HTML. Создал composable `useSeo()`:

```typescript
export function useSeo(options: SeoOptions) {
  const canonical = getDomainLocale() === 'ru'
    ? `https://fxtun.ru${cleanPath}`
    : `https://fxtun.dev${cleanPath}`

  useHead({
    title: options.title,
    meta: [
      { property: 'og:title', content: options.title },
      { property: 'og:description', content: options.description },
      { property: 'og:image', content: 'https://fxtun.dev/og-image.png' },
      { name: 'twitter:card', content: 'summary_large_image' },
    ],
    link: [
      { rel: 'canonical', href: canonical },
      { rel: 'alternate', hreflang: 'en', href: `https://fxtun.dev${cleanPath}` },
      { rel: 'alternate', hreflang: 'ru', href: `https://fxtun.ru${cleanPath}` },
      { rel: 'alternate', hreflang: 'x-default', href: `https://fxtun.dev${cleanPath}` },
    ],
  })
}
```

Два домена — `fxtun.dev` (английский) и `fxtun.ru` (русский). Canonical URL зависит от текущего домена. hreflang связывает языковые версии друг с другом. `x-default` — фоллбэк на английский.

Статические OG-теги в `index.html` — для ботов, которые не ждут гидрации:

```html
<meta property="og:title" content="fxTunnel — Secure Localhost Tunneling" />
<meta property="og:image" content="https://fxtun.dev/og-image.png" />
<meta name="twitter:card" content="summary_large_image" />
```

## Favicon suite и шрифты

Для PWA-подобного опыта — полный набор: `favicon.ico` (48×48), png-иконки (16, 32, 192, 512), `apple-touch-icon` (180×180) и `site.webmanifest`:

```json
{
  "name": "fxTunnel",
  "short_name": "fxTunnel",
  "theme_color": "#0d1117",
  "background_color": "#0d1117",
  "display": "standalone",
  "icons": [
    { "src": "/android-chrome-192x192.png", "sizes": "192x192" },
    { "src": "/android-chrome-512x512.png", "sizes": "512x512" }
  ]
}
```

А шрифты — отключил Google Fonts CDN и захостил всё локально. Три шрифта, 12 файлов WOFF2 с разбивкой по unicode ranges (латиница, кириллица, расширенная кириллица):

- **Unbounded** — заголовки, variable weight 200–900
- **Onest** — текст, variable weight 100–900
- **JetBrains Mono** — код, variable weight 100–800

Все с `font-display: swap` и preload для критических файлов. Результат: минус три DNS-запроса к `fonts.googleapis.com` и `fonts.gstatic.com`, плюс контроль над версиями.

## Sitemap и robots.txt

`vite-plugin-sitemap` генерирует XML при сборке. Настроил:

```typescript
Sitemap({
  hostname: 'https://fxtun.dev',
  dynamicRoutes: ['/login', '/register', '/offer'],
  exclude: ['/docs/offer', '/ru/*', '/en/*'],
  changefreq: 'weekly',
  priority: 0.7,
})
```

Исключил языковые версии `/ru/*` и `/en/*` из sitemap — канонические URL на корневом пути, а переводы связаны через hreflang.

robots.txt — вручную, не автогенерация. 12 защищённых путей (`/dashboard`, `/admin/*`, `/api/*`, `/payment*`), отдельные правила для AI-краулеров. Три sitemap — основной и два для блога (EN/RU).

## Лендинг: топографические контуры

Переделал фоновую анимацию. Был CSS-грид с точками — скучно. Заменил на анимированные топографические контуры. Звучит сложно, но по сути это шум Перлина + Marching Squares:

```typescript
// Simplex 3D noise → marching squares → canvas contours
const CELL = 14        // пикселей на ячейку
const FREQ = 0.012     // масштаб шума
const OCTAVES = 2      // два слоя детализации
const THRESHOLDS = 9   // уровней контуров
const TIME_STEP = 0.003 // скорость анимации
```

Шум генерирует двумерное поле значений. Marching Squares преобразует его в контурные линии по 9 пороговым уровениям. Canvas рисует их каждые 33мс (~30 FPS). IntersectionObserver останавливает анимацию, когда элемент не видим — экономия батареи.

Результат — медленно дрейфующие контуры, как на топографической карте. Плюс радиальная маска, чтобы края плавно исчезали.

## Грабли

**SSG и `window`.** Первая сборка упала с `ReferenceError: window is not defined`. Логично — SSG рендерит в Node.js, где нет `window`. Пришлось обернуть все обращения к `window`, `document`, `localStorage` в проверки `import.meta.env.SSR` или `typeof window !== 'undefined'`.

**Дублирование FAQPage.** Google нашёл одинаковые FAQ на трёх URL (`/`, `/ru`, `/en`) и начал слать предупреждения. Решение — JSON-LD только на канонической странице `/`.

**hreflang на lang-prefixed роутах.** Если `/ru/login` указывает hreflang на `/en/login`, а `/en/login` — обратно на `/ru/login`, Google не знает, какая версия каноническая. Убрал hreflang с lang-prefixed роутов — они для прямых ссылок, канонические страницы без префикса.

**Google Fonts и CSP.** После перехода на self-hosted шрифты Content Security Policy перестала ругаться на внешние ресурсы. Побочный бонус — CSP стала строже.

## Итог

7–8 февраля (v3.0–v3.2):
- **YooKassa** вместо Robokassa — идемпотентность, рекуррентные платежи, фискализация
- **12 пререндеренных страниц** через vite-ssg (3 локали × 4 страницы)
- **4 JSON-LD схемы** — Organization, SoftwareApplication, WebSite, FAQPage
- **OG/Twitter теги** — красивые превью в Telegram, Slack, Facebook
- **hreflang/canonical** — правильная связка EN/RU версий
- **6 favicon форматов** + webmanifest
- **12 self-hosted шрифтов** — минус Google Fonts CDN
- **Топографические контуры** — canvas + simplex noise + marching squares
- **Sitemap** — автогенерация с исключением дублей
```
5b9c66d feat(payment): migrate from Robokassa to YooKassa
9e7d52a feat(web): integrate vite-ssg for static site generation
42abbaf feat(seo): add JSON-LD structured data (Organization, SoftwareApplication)
9cfaacf feat(seo): add FAQ section with 15 questions and FAQPage JSON-LD
acd5de4 feat(seo): add @unhead/vue with dynamic meta tags and OG/Twitter cards
1075527 feat(seo): add sitemap.xml generation via vite-plugin-sitemap
8327760 feat(seo): add robots.txt and llms.txt
d80f35f feat(seo): add favicon suite, OG image, and webmanifest
22e1cfd perf(web): self-host fonts, remove Google Fonts CDN
89c3e14 feat(web): replace grid overlay with animated topographic contours
5094661 feat(web): add OG meta tags, canonical/hreflang, 404 page, WebSite schema
0cd5498 feat(web): redesign landing page for SaaS positioning
```

SPA → SSG — не просто техническая миграция. Это переход от «сайт для людей с JavaScript» к «сайт для всех, включая ботов». А боты — это поисковики, мессенджеры, AI-краулеры. Без них проект невидим.

В следующей части — интернационализация по доменам, попытка DDoS-защиты и уроки того, как чрезмерная безопасность может убить продакшен.
