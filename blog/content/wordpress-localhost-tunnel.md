---
title: "WordPress на localhost через туннель — гайд"
date: 2026-03-15T14:00:00+03:00
draft: false
description: "WordPress хранит URL в базе и ломается за туннелем без правки wp-config.php. Показываем настройку WP_HOME/WP_SITEURL, демо клиенту и вебхуки WooCommerce."
tags: [wordpress, localhost, tunnel, fxTunnel, woocommerce, webhooks, development, developer-tools]
image: ""
rubrics: [guides]
need: domain
cover: ["localhost:8080", "WP_HOME", "wp-demo.fxtun.ru"]
updated: 2026-09-27
---

## Проблема: WordPress заперт на localhost

Разработка WordPress на localhost удобна — до тех пор, пока не нужно показать результат кому-то ещё. Клиент не может открыть ваш `http://localhost:8080`. Платёжные шлюзы не могут отправить вебхуки на `127.0.0.1`. Плагин, использующий внешние OAuth-коллбэки, не имеет публичного URL для редиректа. А деплоить на staging-сервер после каждого изменения CSS — пустая трата времени.

Помимо этого, у WordPress есть уникальная проблема: он хардкодит свой URL в базе данных. Опции `siteurl` и `home` определяют, как WordPress генерирует ссылки, подключает ассеты и обрабатывает редиректы. Если вы просто направите [туннель](/blog/what-is-tunneling/) на локальный WordPress без обновления этих значений, сайт сломается — стили не загрузятся, ссылки будут вести на `localhost`, а админ-панель станет недоступна.

В этой статье мы покажем, как правильно настроить связку localhost + туннель для WordPress: сконфигурировать `WP_HOME` и `WP_SITEURL`, протестировать плагины и темы с реальным трафиком, показать сайт клиенту и отладить вебхуки WooCommerce — всё без деплоя.

## Решение: туннель с правильной конфигурацией WordPress

fxTunnel создаёт публичный HTTPS-адрес для вашего локального WordPress одной командой. В сочетании с двумя строками в `wp-config.php` вы получаете полностью рабочий WordPress-сайт, доступный из любой точки интернета.

Рабочий процесс:

1. Запускаете WordPress локально (XAMPP, Docker, wp-env, Local — что угодно).
2. Создаёте туннель командой `fxtunnel http 8080`.
3. Устанавливаете `WP_HOME` и `WP_SITEURL` на URL туннеля.
4. Открываете URL туннеля в браузере — WordPress работает с полноценным HTTPS.

Если вы впервые работаете с туннелями, в статье [«Что такое туннелирование»](/blog/what-is-tunneling/) разобраны основы.

## Пошаговый гайд: WordPress + fxTunnel

### Шаг 1. Установка fxTunnel

```bash
# Быстрая установка (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Проверяем, что всё работает
fxtunnel version

# Входим в аккаунт fxtun.ru (один раз)
fxtunnel login
```

### Шаг 2. Запуск локального WordPress

Используйте любую удобную среду разработки. Вот самые популярные варианты:

**Docker Compose (рекомендуется):**
```yaml
# docker-compose.yml
version: '3.8'
services:
  wordpress:
    image: wordpress:latest
    ports:
      - "8080:80"
    environment:
      WORDPRESS_DB_HOST: db
      WORDPRESS_DB_USER: wordpress
      WORDPRESS_DB_PASSWORD: wordpress
      WORDPRESS_DB_NAME: wordpress
    volumes:
      - ./wp-content:/var/www/html/wp-content

  db:
    image: mysql:8.0
    environment:
      MYSQL_DATABASE: wordpress
      MYSQL_USER: wordpress
      MYSQL_PASSWORD: wordpress
      MYSQL_ROOT_PASSWORD: rootpassword
    volumes:
      - db_data:/var/lib/mysql

volumes:
  db_data:
```

```bash
docker compose up -d
# -> WordPress запущен на http://localhost:8080
```

**wp-env (для разработки плагинов/тем):**
```bash
# Устанавливаем wp-env глобально
npm install -g @wordpress/env

# Запускаем WordPress в директории плагина/темы
wp-env start
# -> WordPress запущен на http://localhost:8888
```

**XAMPP / MAMP / Local by Flywheel:** запускайте среду как обычно и запомните порт.

### Шаг 3. Создание туннеля

Без флага `--domain` каждый запуск `fxtunnel` выдаёт новый случайный поддомен, и `wp-config.php` придётся переписывать заново при каждом перезапуске. Зарезервируйте поддомен один раз (с тарифа Base) и запускайте туннель с ним:

```bash
fxtunnel domains add wp-demo
fxtunnel http 8080 --domain wp-demo
```

Результат:

```
Connecting to fxtunnel server...
Tunnel established!
HTTP:  http://wp-demo.fxtun.ru
HTTPS: https://wp-demo.fxtun.ru
Forwarding to localhost:8080
```

Теперь `https://wp-demo.fxtun.ru` указывает на ваш локальный WordPress и не меняется между перезапусками. Но если вы откроете этот URL прямо сейчас, WordPress, скорее всего, перенаправит вас на `localhost:8080` или загрузится без стилей. Причина — проблема с `siteurl` и `home`. Давайте её решим.

### Шаг 4. Настройка WP_HOME и WP_SITEURL

Откройте `wp-config.php` и добавьте две строки **перед** строкой `/* That's all, stop editing! */`:

```php
// wp-config.php — добавьте эти строки для доступа через туннель
define('WP_HOME', 'https://wp-demo.fxtun.ru');
define('WP_SITEURL', 'https://wp-demo.fxtun.ru');
```

Эти константы переопределяют значения из базы данных. WordPress теперь будет генерировать все ссылки, загружать все ассеты и обрабатывать все редиректы, используя URL туннеля.

#### Динамическая конфигурация (лучше для разработки)

Если вы не хотите каждый раз редактировать `wp-config.php` при перезапуске туннеля (или если вы часто переключаетесь между локальным и туннельным доступом), используйте динамический подход:

```php
// wp-config.php — динамический URL на основе запроса
if (isset($_SERVER['HTTP_HOST'])) {
    $scheme = (isset($_SERVER['HTTP_X_FORWARDED_PROTO']) && $_SERVER['HTTP_X_FORWARDED_PROTO'] === 'https')
        ? 'https'
        : (isset($_SERVER['HTTPS']) && $_SERVER['HTTPS'] === 'on' ? 'https' : 'http');
    $host = $_SERVER['HTTP_HOST'];
    define('WP_HOME', "{$scheme}://{$host}");
    define('WP_SITEURL', "{$scheme}://{$host}");
}
```

Этот сниппет считывает имя хоста и протокол из входящего запроса. Когда вы обращаетесь к сайту через туннель, WordPress использует URL туннеля. Когда заходите напрямую через `localhost:8080` — использует `localhost:8080`. Никаких ручных изменений не нужно.

#### Почему важен X-Forwarded-Proto

fxTunnel терминирует TLS на сервере туннеля и перенаправляет запросы на ваш локальный WordPress по обычному HTTP. WordPress видит `HTTP`, а не `HTTPS`. Без проверки `HTTP_X_FORWARDED_PROTO` WordPress может генерировать ссылки с `http://`, хотя клиент подключён через `https://`. Динамический сниппет выше решает эту проблему автоматически.

Также добавьте это в `wp-config.php`, если WordPress работает за реверс-прокси (а туннель — это фактически реверс-прокси):

```php
// Доверяем заголовку X-Forwarded-Proto от туннеля
if (isset($_SERVER['HTTP_X_FORWARDED_PROTO']) && $_SERVER['HTTP_X_FORWARDED_PROTO'] === 'https') {
    $_SERVER['HTTPS'] = 'on';
}
```

### Шаг 5. Проверяем, что всё работает

Откройте `https://wp-demo.fxtun.ru` в браузере. Вы должны увидеть:

- Зелёный замочек (валидный HTTPS-сертификат).
- Вашу WordPress-тему со всеми загруженными стилями.
- Рабочую навигацию — все ссылки ведут на `https://wp-demo.fxtun.ru/...`.
- Админ-панель по адресу `https://wp-demo.fxtun.ru/wp-admin/` корректно загружается.

Если что-то не так — смотрите раздел «Troubleshooting» ниже.

## Сценарий 1: демонстрация клиенту без деплоя

Показать WordPress-сайт клиенту обычно означает деплой на staging-сервер — загрузка файлов, миграция базы данных, исправление URL, настройка SSL. С туннелем всё это можно пропустить.

Рабочий процесс прост:

1. Доводите сайт до нужного состояния на локальной машине.
2. Запускаете `fxtunnel http 8080`.
3. Устанавливаете `WP_HOME` и `WP_SITEURL` на URL туннеля (или используйте динамический сниппет).
4. Отправляете ссылку клиенту.

Клиент видит именно то, что у вас на экране — ту же тему, тот же контент, те же плагины. Вы вносите изменения, клиент обновляет страницу. Никакого цикла деплоя, никаких расходов на staging-сервер.

Больше сценариев демонстрации — в статье [«Как показать локальный проект клиенту без деплоя»](/blog/demo-localhost-without-deploy/).

### Собственный домен для профессиональных демонстраций

Ссылка `https://wp-demo.fxtun.ru` работает, но `https://preview.clientsite.com` выглядит профессиональнее. С тарифа Base можно привязать собственный домен к зарезервированному поддомену:

```bash
fxtunnel domains custom add preview.clientsite.com --target wp-demo
# в DNS: TXT _fxtunnel-challenge.preview.clientsite.com = fxtunnel-verify=<токен из личного кабинета>
#        CNAME preview.clientsite.com → wp-demo.fxtun.ru
fxtunnel domains custom verify preview.clientsite.com

fxtunnel http 8080 --domain wp-demo
```

Обновите `wp-config.php` соответственно:

```php
define('WP_HOME', 'https://preview.clientsite.com');
define('WP_SITEURL', 'https://preview.clientsite.com');
```

## Сценарий 2: тестирование плагинов и тем с реальным трафиком

Пробовали тестировать OAuth-плагин авторизации на localhost? Не получится — публичного URL для редиректа просто нет. Та же история с платёжными шлюзами, кнопками соцсетей, встраиванием контента и интеграциями с CDN. Туннель даёт публичный URL без деплоя.

### Тестирование OAuth-плагинов

Плагины вроде **Nextend Social Login**, **WP Social Login** или **Login with Google** требуют OAuth redirect URI. На localhost зарегистрировать валидный редирект невозможно. С туннелем:

```bash
fxtunnel http 8080
# -> https://wp-demo.fxtun.ru
```

Зарегистрируйте `https://wp-demo.fxtun.ru/wp-admin/admin-ajax.php` (или другой URL, который использует плагин) как redirect URI в настройках OAuth-провайдера (Google, Facebook, GitHub). Поток авторизации работает целиком на вашей локальной машине. Этот паттерн подробно разобран в статье [«OAuth Callback на localhost»](/blog/oauth-callback-localhost-tunnel/).

### Тестирование SEO и соцсетей

Когда вы делитесь страницей WordPress в соцсетях, платформы (Facebook, Twitter, LinkedIn) запрашивают Open Graph мета-теги по URL. На localhost они не могут добраться до вашего сайта. Через туннель публичный URL доступен — можно проверить, как выглядят ваши страницы при расшаривании.

```bash
# Тестируем Open Graph теги
curl -A "facebookexternalhit/1.1" https://wp-demo.fxtun.ru/sample-page/
```

### Тестирование Cron и запланированных задач

WordPress использует `wp-cron.php`, который срабатывает при посещении страницы. Некоторые разработчики используют внешние cron-сервисы, которые вызывают `wp-cron.php` через HTTP. Через туннель эти сервисы могут добраться до вашего локального WordPress:

```bash
# Внешний cron-сервис теперь может обращаться по этому URL
# https://wp-demo.fxtun.ru/wp-cron.php?doing_wp_cron
```

## Сценарий 3: тестирование вебхуков WooCommerce

WooCommerce использует вебхуки для уведомления внешних систем о заказах, платежах, изменениях остатков и событиях клиентов. При разработке кастомной интеграции — например, синхронизации заказов с ERP или отправки уведомлений в Slack — необходимо тестировать вебхуки с реальными данными. На localhost у WooCommerce нет публичного URL для доставки вебхуков.

### Настройка вебхуков WooCommerce через туннель

```bash
# Запускаем туннель
fxtunnel http 8080
# -> https://wp-demo.fxtun.ru
```

1. Откройте **WooCommerce -> Настройки -> Дополнительно -> Вебхуки** в админ-панели WordPress.
2. Нажмите **Добавить вебхук**.
3. Название: `Тестовый вебхук заказов`.
4. Статус: **Активен**.
5. Тема: **Заказ создан**.
6. URL доставки: `https://wp-demo.fxtun.ru/wp-json/custom/v1/order-hook` (или ваш внешний эндпоинт через туннель).
7. Секрет: сгенерируйте секрет для проверки подписи.
8. Сохраните.

Теперь создайте тестовый заказ в WooCommerce. Вебхук сработает и отправит запрос на URL доставки — туннель перенаправит его на ваш локальный обработчик.

### Пример обработчика вебхуков

Если ваш эндпоинт для вебхуков находится на том же локальном WordPress (например, кастомный эндпоинт плагина), можно протестировать весь цикл:

```php
// В вашем плагине — регистрация REST API эндпоинта для вебхука
add_action('rest_api_init', function () {
    register_rest_route('custom/v1', '/order-hook', [
        'methods'  => 'POST',
        'callback' => 'handle_woo_webhook',
        'permission_callback' => '__return_true',
    ]);
});

function handle_woo_webhook(WP_REST_Request $request) {
    $payload = $request->get_json_params();
    $signature = $request->get_header('X-WC-Webhook-Signature');

    // Проверка подписи
    $secret = 'your_webhook_secret';
    $expected = base64_encode(hash_hmac('sha256', $request->get_body(), $secret, true));

    if ($signature !== $expected) {
        return new WP_REST_Response(['error' => 'Invalid signature'], 401);
    }

    // Логируем данные вебхука
    error_log('WooCommerce вебхук получен: ' . wp_json_encode($payload));

    // Обрабатываем заказ
    $order_id = $payload['id'] ?? null;
    $status = $payload['status'] ?? 'unknown';
    error_log("Заказ #{$order_id} — статус: {$status}");

    return new WP_REST_Response(['received' => true], 200);
}
```

### Внешний приёмник вебхуков

Если ваш приёмник вебхуков — отдельное приложение (Node.js, Python и т.д.), работающее на другом порту, откройте второй туннель (два туннеля одновременно требуют тариф Base и выше — на бесплатном тарифе доступен только один):

```bash
# Терминал 1 — WordPress
fxtunnel http 8080
# -> https://wp-demo.fxtun.ru

# Терминал 2 — приёмник вебхуков (Node.js на порту 3000)
fxtunnel http 3000
# -> https://hook-receiver.fxtun.ru
```

Используйте `https://hook-receiver.fxtun.ru/webhook` как URL доставки в WooCommerce. Или, если оба сервиса работают локально, можно указать `http://localhost:3000/webhook` как URL доставки и обойтись без второго туннеля — но первый туннель всё равно нужен для доступа к админ-панели WooCommerce.

### Отладка вебхуков через Inspector

Inspector в fxTunnel (с тарифа Base) показывает каждый HTTP-запрос, проходящий через туннель — заголовки, тело, код ответа. Это незаменимо при отладке вебхуков: вы видите, что именно отправил WooCommerce и как ответил ваш обработчик. Replay позволяет повторить любой запрос одним кликом, так что не нужно создавать новый тестовый заказ каждый раз.

В статье [«Тестирование вебхуков через туннель»](/blog/webhook-testing-with-tunnel/) разобран весь процесс подробнее.

## Сценарий 4: Multisite и командная разработка

### WordPress Multisite

WordPress Multisite использует доменные или поддиректорийные URL для каждого подсайта. Через туннель можно тестировать основной сайт, но подсайты требуют дополнительной настройки. Самый простой подход — Multisite на основе поддиректорий:

```php
// wp-config.php для Multisite через туннель
define('WP_HOME', 'https://wp-demo.fxtun.ru');
define('WP_SITEURL', 'https://wp-demo.fxtun.ru');
define('DOMAIN_CURRENT_SITE', 'wp-demo.fxtun.ru');
```

Поддиректорийные подсайты (`https://wp-demo.fxtun.ru/site2/`) будут работать через туннель без дополнительной настройки.

### Работа с другими разработчиками

Когда двум разработчикам нужно протестировать интеграцию (например, фронтенд-разработчику нужен WordPress REST API), бэкенд-разработчик открывает туннель и делится URL:

```bash
# Бэкенд-разработчик
fxtunnel http 8080
# -> https://wp-api.fxtun.ru

# Фронтенд-разработчик использует API
curl https://wp-api.fxtun.ru/wp-json/wp/v2/posts
```

Фронтенд-разработчик может указать `https://wp-api.fxtun.ru` как базовый URL API в своём React или Vue приложении и протестировать полную интеграцию без деплоя.

## Локальные среды разработки: сравнение с поддержкой туннелей

Разные локальные WordPress-среды используют разные порты и конфигурации. Вот как они работают с fxTunnel:

| Среда | Порт по умолчанию | Команда туннеля | Примечания |
|---|---|---|---|
| **Docker Compose** | 8080 (настраивается) | `fxtunnel http 8080` | Максимальная гибкость, полный контроль над версиями PHP/MySQL |
| **wp-env** | 8888 | `fxtunnel http 8888` | Официальный инструмент WordPress, идеален для разработки плагинов |
| **XAMPP** | 80 | `fxtunnel http 80` | Классический стек, работает на Windows/macOS/Linux |
| **MAMP** | 8888 | `fxtunnel http 8888` | Популярен на macOS, есть GUI |
| **Local (Flywheel)** | варьируется | `fxtunnel http <порт>` | GUI-приложение, проверьте порт в настройках сайта |
| **Lando** | варьируется | `fxtunnel http <порт>` | На основе Docker, проверьте `lando info` для порта |
| **Laravel Valet** | 80 (через домен .test) | `fxtunnel http 80` | Лёгкий вариант, только macOS |

## Troubleshooting: частые проблемы WordPress + туннель

Большинство проблем связаны с тремя вещами: несоответствие `WP_HOME`/`WP_SITEURL`, путаница HTTPS/HTTP или локальный сервер не запущен. Используйте таблицу ниже для быстрой диагностики.

| Проблема | Причина | Решение |
|---|---|---|
| Зацикленный редирект на localhost | `WP_HOME`/`WP_SITEURL` всё ещё указывают на `localhost` | Обновите `wp-config.php`, указав URL туннеля |
| Стили и скрипты не загружаются | Mixed content (HTTP-ассеты на HTTPS-странице) | Добавьте фикс `$_SERVER['HTTPS'] = 'on'` для X-Forwarded-Proto |
| Админ-панель недоступна | `WP_SITEURL` не совпадает с URL туннеля | Установите `WP_SITEURL` точно на URL туннеля, включая `https://` |
| Ошибка «Too many redirects» | WordPress принудительно перенаправляет на HTTPS, туннель повторно перенаправляет | Добавьте `$_SERVER['HTTPS'] = 'on'` при `X-Forwarded-Proto` = `https` |
| Изображения с битыми URL | `wp_get_attachment_url` возвращает localhost URL | `WP_HOME` должен совпадать с URL туннеля |
| REST API возвращает 401 | Несовпадение nonce из-за смены домена | Выйдите из системы и войдите заново через URL туннеля |
| Вебхук WooCommerce не работает | URL доставки всё ещё указывает на localhost | Обновите URL доставки вебхука на URL туннеля |
| Медленная загрузка страниц | Большой сайт с множеством ассетов | Нормально — ассеты проходят через туннель. Используйте кэширование браузера |

### Фикс для определения HTTPS

Если WordPress-плагины или темы проверяют `is_ssl()` и получают `false` из-за того, что туннель пересылает HTTP, добавьте это в начало `wp-config.php`:

```php
// Фикс определения SSL за туннелем/реверс-прокси
if (
    (isset($_SERVER['HTTP_X_FORWARDED_PROTO']) && $_SERVER['HTTP_X_FORWARDED_PROTO'] === 'https') ||
    (isset($_SERVER['HTTP_X_FORWARDED_SSL']) && $_SERVER['HTTP_X_FORWARDED_SSL'] === 'on')
) {
    $_SERVER['HTTPS'] = 'on';
}
```

## WordPress-туннель vs staging-сервер: когда что использовать

Туннель не заменяет staging-сервер — он дополняет его. Используйте туннель во время активной разработки и для быстрых демонстраций. Staging — для длительного QA, приёмочного тестирования клиентом и предпродакшен-проверки.

| Критерий | fxTunnel | Staging-сервер |
|---|---|---|
| **Время настройки** | 30 секунд | От 30 минут до нескольких часов |
| **Стоимость** | Бесплатно (1 туннель), платные тарифы — больше туннелей и собственный домен | $5-50/мес за хостинг |
| **HTTPS** | Автоматический, валидный сертификат | Нужен Let's Encrypt или ручная настройка |
| **Актуальность данных** | Всегда актуальные (live localhost) | Снимок с последнего деплоя |
| **Совместная работа** | Один разработчик за раз (его машина) | Общий, доступен всей команде |
| **Доступность** | Пока туннель запущен | Всегда доступен |
| **Производительность** | Зависит от машины + сети | Зависит от характеристик сервера |
| **Лучше всего для** | Разработки, быстрых демо, тестирования вебхуков | QA, приёмочного тестирования, предпродакшена |

Сравнение других методов демонстрации — в статье [«Как показать проект без деплоя»](/blog/demo-localhost-without-deploy/).

## Лучшие практики для разработки WordPress + туннель

### 1. Используйте динамический сниппет WP_HOME

Не хардкодьте URL туннеля — используйте динамический сниппет из Шага 4. Так вы сможете переключаться между `localhost` и URL туннеля без редактирования конфигурации.

### 2. Защитите wp-admin

Когда ваш WordPress-сайт доступен публично через туннель, `wp-admin` тоже доступен. Используйте надёжные пароли и рассмотрите добавление HTTP Basic-авторизации:

```php
// .htaccess — базовая авторизация для wp-admin (Apache)
<Files wp-login.php>
  AuthType Basic
  AuthName "Restricted"
  AuthUserFile /path/to/.htpasswd
  Require valid-user
</Files>
```

### 3. Используйте тестовые данные

Никогда не демонстрируйте WordPress-сайт с реальными данными клиентов. Используйте [FakerPress](https://wordpress.org/plugins/fakerpress/) или WP CLI для генерации тестового контента:

```bash
# Генерируем тестовые записи через WP CLI
wp post generate --count=20 --post_type=post --post_status=publish
wp user generate --count=5 --role=subscriber
```

### 4. Закрывайте туннель после использования

Нажмите `Ctrl+C`, когда демонстрация или тестирование завершены. Не оставляйте туннель запущенным на ночь с WordPress-сайтом — `wp-admin` будет доступен любому, кто знает URL.

### 5. Используйте Inspector для отладки

Inspector в fxTunnel (с тарифа Base) особенно полезен с WordPress, потому что WordPress отправляет множество внутренних HTTP-запросов (вызовы REST API, cron, проверка обновлений). Inspector позволяет увидеть их все и быстро обнаружить неправильно настроенные URL или неудавшиеся запросы.

## Почему fxTunnel для разработки WordPress

WordPress-сайты грузят много ассетов на каждую страницу. Встроенный [HTTPS](/blog/https-localhost-development/) с валидным сертификатом означает, что HTTPS-функции WordPress (Secure Cookies, `is_ssl()`, принудительный SSL в админке) корректно работают через туннель. При закреплённом поддомене (`--domain`) URL остаётся постоянным между перезапусками, поэтому не нужно обновлять `wp-config.php` каждый раз.

| Возможность | fxTunnel | Другие инструменты |
|---|---|---|
| Бесплатный тариф | 1 туннель, без резерва поддомена | Часто ограничен по времени или запросам |
| HTTPS-сертификат | Валидный, автоматический — без предупреждений браузера | Аналогично у большинства инструментов |
| Стабильный URL | При закреплённом поддомене, с тарифа Base | Часто случайный на бесплатных тарифах |
| Inspector + Replay | С тарифа Base — просмотр всех запросов, повтор одним кликом | Отсутствует или требует отдельной настройки |
| Кастомный домен (свой DNS) | С тарифа Base | Дороже или недоступно |
| Больше туннелей сразу | Pro — 15, Business — 50 | Значительно дороже |
| Открытость | Клиент открыт (GitHub), сервер можно развернуть у себя | Часто полностью проприетарный |

Все три инструмента мы подробно сравниваем в статье [«ngrok vs Cloudflare vs fxTunnel»](/blog/ngrok-vs-cloudflare-vs-fxtunnel/).

