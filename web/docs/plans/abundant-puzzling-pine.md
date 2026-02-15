# Блог: мультиязычная архитектура по доменам

## Контекст

Сейчас блог Hugo: RU — default (root), EN — `/blog/en/`. Языковой префикс **внутри** `/blog/`. Нужно: язык определяется **доменом** (как fxTunnel), а `/en/blog/` и `/ru/blog/` — явные переключатели с редиректом на каноничный домен.

**Результат:**
- `fxtun.dev/blog/*` = EN
- `fxtun.ru/blog/*` = RU
- `fxtun.dev/ru/blog/*` → 302 → `fxtun.ru/blog/*`
- `fxtun.ru/en/blog/*` → 302 → `fxtun.dev/blog/*`

## Шаги

### 1. hugo.toml — EN как default

```toml
defaultContentLanguage = "en"
```

Hugo output: EN at root (`public/`), RU at `public/ru/`.

### 2. Переименовать контент-файлы

`*.md` (без суффикса, сейчас RU) → `*.ru.md`
`*.en.md` → `*.md` (новый EN default)

Файлы: `_index`, `hello-world`, `expose-localhost-to-internet`, `fxtunnel-architecture`, и все остальные ~17 пар.

### 3. head.html — cross-domain canonical + hreflang

Текущий canonical: `{{ .Permalink }}` (всегда `fxtun.dev/blog/...`).
Нужно: RU canonical → `fxtun.ru/blog/...`, EN → `fxtun.dev/blog/...`.

Логика:
```
cleanPath = RelPermalink с удалённым /blog/ru/ → /blog/
enURL = https://fxtun.dev + cleanPath
ruURL = https://fxtun.ru + cleanPath

canonical = lang=="ru" ? ruURL : enURL
hreflang en → enURL, ru → ruURL, x-default → enURL
og:url = canonical
```

### 4. header.html — language switcher по доменам

Текущий: ссылка на `.Translations` (ведёт к `/blog/en/article`).
Нужно: ссылка на другой домен (`fxtun.ru/blog/article` ↔ `fxtun.dev/blog/article`).

### 5. schema.html — обновить URL-ы

`"item": "{{ .Permalink }}"` → использовать каноничный URL (domain-based).

### 6. Nginx — все 3 конфига

**nginx-fxtun-ru.conf** — блог:
```nginx
# RU blog by default on .ru domain
location /blog/ {
    alias /var/www/fxtunnel-blog/ru/;
    ...
}
# Prefix redirects
location /en/blog/ { return 302 https://fxtun.dev$request_uri; }  # strip /en
location /ru/blog/ { rewrite ^/ru(/blog/.*)$ $1 redirect; }  # /ru/blog/* → /blog/*
```

**nginx.example.conf (fxtun.dev)** — блог:
```nginx
# EN blog by default (Hugo root = EN)
location /blog/ { alias /var/www/fxtunnel-blog/; ... }
# Prefix redirects
location /ru/blog/ { return 302 https://fxtun.ru$request_uri; }  # strip /ru → fxtun.ru
location /en/blog/ { rewrite ^/en(/blog/.*)$ $1 redirect; }  # /en/blog/* → /blog/*
```

**nginx-mfdev.conf** — уже noindex, добавить аналогичные redirects.

### 7. deploy.sh — без изменений

`hugo --minify` продолжает работать, просто output structure меняется.

## Файлы

| Файл | Действие |
|------|----------|
| `hugo.toml` | `defaultContentLanguage = "en"` |
| `content/*.md` | Переименовать: `.md`→`.ru.md`, `.en.md`→`.md` |
| `layouts/partials/head.html` | Cross-domain canonical/hreflang/og:url |
| `layouts/partials/header.html` | Language switcher → другой домен |
| `layouts/partials/schema.html` | Каноничные URL |
| `configs/nginx-fxtun-ru.conf` | Blog alias → `/ru/`, prefix redirects |
| `configs/nginx.example.conf` | Prefix redirects для блога |
| `configs/nginx-mfdev.conf` | Prefix redirects для блога |

## Верификация

1. `cd blog && hugo --minify` — проверить что EN at root, RU at /ru/
2. `public/index.html` — EN content, canonical=fxtun.dev, hreflang en/ru
3. `public/ru/index.html` — RU content, canonical=fxtun.ru
4. `public/hello-world/index.html` — EN article
5. `public/ru/hello-world/index.html` — RU article
6. Language switcher links → cross-domain
