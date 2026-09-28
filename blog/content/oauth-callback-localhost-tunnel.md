---
title: "OAuth Callback на localhost через туннель"
date: 2026-03-09T14:00:00+03:00
draft: false
description: "Redirect URI требует публичный HTTPS-адрес, а localhost для OAuth не годится. Разбираем настройку fxTunnel для Google, GitHub и Facebook на живом примере кода."
tags: [oauth, authentication, testing, fxTunnel, localhost, tunnel, developer-tools]
image: ""
rubrics: [guides]
need: domain
cover: ["localhost:3000", "fxtunnel http", "redirect_uri"]
updated: 2026-09-27
---

## Проблема: OAuth callback требует публичный HTTPS URL

Если вы хоть раз пытались протестировать «Войти через Google» на localhost -- вы знаете эту боль. OAuth-провайдеры требуют redirect URI -- публичный HTTPS-адрес, на который провайдер перенаправляет пользователя после аутентификации. Localhost под это не подходит. [Локальный туннель](/blog/what-is-tunneling/) решает проблему за полминуты: fxTunnel создаёт публичный HTTPS URL для вашего localhost, и OAuth callback просто работает.

Вот как устроен поток: ваше приложение перенаправляет пользователя к провайдеру (Google, GitHub, Facebook), пользователь входит и даёт разрешения, а провайдер перенаправляет обратно в ваше приложение с кодом авторизации. Загвоздка -- в шаге «перенаправление обратно»: провайдер отправляет пользователя на URL из настроек OAuth. Если это `http://localhost:3000/auth/callback`, редирект происходит в браузере, так что технически работает. Но многие провайдеры отклоняют `http://` URL или требуют валидацию домена, а тестирование с коллегами или на мобильных устройствах не работает вовсе.

Классические обходные пути -- деплой на staging после каждого изменения или исключения для `localhost`, которые есть не у всех провайдеров -- медленные, ненадёжные и непоследовательные. Туннель даёт каждому разработчику в команде стабильный HTTPS URL, который работает с любым OAuth-провайдером.

## Как работает OAuth — и где ломается localhost

OAuth 2.0 Authorization Code flow включает три стороны: браузер пользователя, OAuth-провайдер и ваше приложение. Понимание роли redirect URI объясняет, почему [туннель](/blog/expose-localhost-to-internet/) — самое чистое решение для локальной разработки.

{{< flow caption="Провайдер отправляет браузер на redirect_uri; для localhost этим адресом становится URL туннеля, который сервер пересылает на ваше приложение." >}}
- {label: браузер пользователя, title: редирект от провайдера, sub: "?code=xxx"}
- {link: HTTPS, kind: tls}
- {label: fxTunnel Server, title: oauth-dev.fxtun.ru, sub: redirect_uri}
- {link: TLS, kind: tls}
- {label: ваше приложение, title: клиент fxtunnel, sub: "localhost:3000/auth/callback", me: true}
{{< /flow >}}

Шаг 4 — это место, где возникает проблема. Провайдер перенаправляет браузер пользователя на `redirect_uri`. Если этот URI — `https://abc123.fxtun.ru/auth/callback`, браузер делает запрос на публичный URL туннеля, [сервер туннеля](/blog/fxtunnel-architecture/) перенаправляет его на ваш `localhost:3000`, и приложение получает код авторизации. Весь поток работает бесшовно.

### Почему одного localhost недостаточно

Разные OAuth-провайдеры имеют разные правила для redirect URI:

| Провайдер | `http://localhost` разрешён? | Требуется HTTPS? | Валидация домена? |
|---|---|---|---|
| **Google** | Да (только для разработки) | Продакшен: да | Да (authorized domains) |
| **GitHub** | Да | Нет (но рекомендуется) | Нет |
| **Facebook** | Нет (требуется HTTPS) | Да | Да (app domains) |
| **Apple** | Нет | Да | Да (associated domains) |
| **Microsoft** | Да (только для разработки) | Продакшен: да | Да (redirect URIs) |
| **Twitter/X** | Нет | Да | Да (callback URL) |

Даже когда провайдер разрешает `http://localhost`, есть практические проблемы:

- **Коллеги не могут протестировать вашу ветку** — `localhost` работает только на вашей машине.
- **Мобильное тестирование невозможно** — телефон не может достучаться до localhost вашего ноутбука.
- **Некоторые библиотеки требуют HTTPS** — OAuth SDK могут отклонять не-HTTPS redirect URI.
- **Непоследовательное поведение** — то, что работает для GitHub, может не работать для Google в продакшен-режиме.

Туннель устраняет все эти проблемы разом. Одна команда, один HTTPS URL, любой провайдер работает.

## Настройка тестирования OAuth через fxTunnel

Чтобы OAuth callback заработал локально, нужно три шага: установить CLI, запустить туннель и зарегистрировать публичный HTTPS URL как redirect URI. fxTunnel работает как SaaS -- не нужно настраивать сервер, DNS или сертификаты.

### Шаг 1. Установка fxTunnel

```bash
# Быстрая установка (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Проверяем установку
fxtunnel version
```

### Шаг 2. Запуск приложения

Убедитесь, что ваше веб-приложение запущено локально. В примерах используется порт 3000:

```bash
# Пример: запуск Next.js-приложения
npm run dev
# → http://localhost:3000

# Пример: запуск Django-приложения
python manage.py runserver 3000
# → http://localhost:3000
```

### Шаг 3. Создание туннеля

Без флага `--domain` каждый новый запуск `fxtunnel` выдаёт случайный поддомен, и redirect URI пришлось бы каждый раз перерегистрировать у провайдера. Чтобы адрес не менялся, сначала зарезервируйте поддомен (доступно с тарифа Base):

```bash
fxtunnel domains add oauth-dev

# Создаём HTTPS-туннель на этом поддомене
fxtunnel http 3000 --domain oauth-dev
```

В выводе появится публичный URL:

```console
Connecting to fxtunnel server...
Tunnel established!
HTTP:  http://oauth-dev.fxtun.ru
HTTPS: https://oauth-dev.fxtun.ru
Forwarding to localhost:3000
```

Теперь `https://oauth-dev.fxtun.ru` — ваш базовый URL, и он не меняется от запуска к запуску. Redirect URI будет вида `https://oauth-dev.fxtun.ru/auth/callback`.

### Шаг 4. Регистрация redirect URI

Перейдите в консоль разработчика OAuth-провайдера и добавьте URL туннеля как redirect URI. Инструкции для конкретных провайдеров — ниже.

## Настройка OAuth для популярных провайдеров

Ниже -- готовые конфигурации для Google, GitHub и Facebook OAuth с fxTunnel. Каждый раздел описывает настройку в консоли разработчика, формат redirect URI и рабочий пример кода. Во всех примерах предполагается, что у вас запущен `fxtunnel http 3000`.

### Google OAuth: вход через Google

Google OAuth — самый распространённый метод аутентификации в вебе. Google разрешает `http://localhost` для тестирования, но требует HTTPS для продакшен redirect URI. Туннель позволяет разработчикам тестировать с тем же HTTPS-потоком, который будет работать в продакшене.

#### Настройка в консоли разработчика

1. Откройте [Google Cloud Console](https://console.cloud.google.com/apis/credentials).
2. Создайте или выберите проект.
3. Перейдите в **Credentials** и нажмите **Create Credentials > OAuth client ID**.
4. Тип приложения: **Web application**.
5. В **Authorized redirect URIs** добавьте: `https://oauth-dev.fxtun.ru/auth/google/callback`.
6. Скопируйте **Client ID** и **Client Secret**.

#### Пример на Express.js с Passport

```javascript
// app.js — Google OAuth с Passport.js
const express = require('express');
const passport = require('passport');
const GoogleStrategy = require('passport-google-oauth20').Strategy;
const session = require('express-session');

const app = express();

app.use(session({
  secret: 'dev-secret',
  resave: false,
  saveUninitialized: false,
}));
app.use(passport.initialize());
app.use(passport.session());

passport.use(new GoogleStrategy({
    clientID: process.env.GOOGLE_CLIENT_ID,
    clientSecret: process.env.GOOGLE_CLIENT_SECRET,
    // Используем URL туннеля как callback
    callbackURL: 'https://oauth-dev.fxtun.ru/auth/google/callback',
  },
  (accessToken, refreshToken, profile, done) => {
    console.log('Google профиль:', profile.displayName, profile.emails[0].value);
    return done(null, profile);
  }
));

passport.serializeUser((user, done) => done(null, user));
passport.deserializeUser((user, done) => done(null, user));

// Редирект на Google
app.get('/auth/google',
  passport.authenticate('google', { scope: ['profile', 'email'] })
);

// Google перенаправляет сюда
app.get('/auth/google/callback',
  passport.authenticate('google', { failureRedirect: '/login' }),
  (req, res) => {
    console.log('OAuth успешен! Пользователь:', req.user.displayName);
    res.redirect('/dashboard');
  }
);

app.get('/dashboard', (req, res) => {
  if (!req.user) return res.redirect('/auth/google');
  res.send(`Добро пожаловать, ${req.user.displayName}!`);
});

app.listen(3000, () => console.log('Сервер запущен на порту 3000'));
```

```bash
# Запуск приложения с OAuth-ключами
GOOGLE_CLIENT_ID=xxx GOOGLE_CLIENT_SECRET=yyy node app.js

# В другом терминале — запуск туннеля
fxtunnel http 3000
```

Откройте `https://oauth-dev.fxtun.ru/auth/google` в браузере. Google покажет экран согласия, и после подтверждения браузер перенаправит на `https://oauth-dev.fxtun.ru/auth/google/callback` — туннель перешлёт запрос на локальный сервер.

### GitHub OAuth: вход через GitHub

GitHub OAuth широко используется для инструментов разработчиков, CLI-приложений и платформ, связанных с кодом. GitHub — один из самых лояльных провайдеров, он разрешает `http://localhost` для разработки. Однако туннель обеспечивает единообразный HTTPS-опыт и позволяет тестировать с других устройств.

#### Настройка в консоли разработчика

1. Откройте [GitHub Developer Settings](https://github.com/settings/developers).
2. Нажмите **New OAuth App**.
3. Homepage URL: `https://oauth-dev.fxtun.ru`.
4. Authorization callback URL: `https://oauth-dev.fxtun.ru/auth/github/callback`.
5. Скопируйте **Client ID** и сгенерируйте **Client Secret**.

#### Пример на Express.js

```javascript
// github-oauth.js
const express = require('express');
const axios = require('axios');
const app = express();

const CLIENT_ID = process.env.GITHUB_CLIENT_ID;
const CLIENT_SECRET = process.env.GITHUB_CLIENT_SECRET;
const REDIRECT_URI = 'https://oauth-dev.fxtun.ru/auth/github/callback';

// Шаг 1: редирект на GitHub
app.get('/auth/github', (req, res) => {
  const params = new URLSearchParams({
    client_id: CLIENT_ID,
    redirect_uri: REDIRECT_URI,
    scope: 'user:email',
  });
  res.redirect(`https://github.com/login/oauth/authorize?${params}`);
});

// Шаг 2: обработка callback
app.get('/auth/github/callback', async (req, res) => {
  const { code } = req.query;

  if (!code) {
    return res.status(400).send('Отсутствует код авторизации');
  }

  try {
    // Обмен кода на access token
    const tokenResponse = await axios.post(
      'https://github.com/login/oauth/access_token',
      {
        client_id: CLIENT_ID,
        client_secret: CLIENT_SECRET,
        code: code,
        redirect_uri: REDIRECT_URI,
      },
      { headers: { Accept: 'application/json' } }
    );

    const accessToken = tokenResponse.data.access_token;

    // Получение профиля пользователя
    const userResponse = await axios.get('https://api.github.com/user', {
      headers: { Authorization: `Bearer ${accessToken}` },
    });

    console.log('GitHub пользователь:', userResponse.data.login);
    res.json({
      login: userResponse.data.login,
      name: userResponse.data.name,
      email: userResponse.data.email,
    });
  } catch (error) {
    console.error('Ошибка OAuth:', error.message);
    res.status(500).send('Аутентификация не удалась');
  }
});

app.listen(3000, () => console.log('Сервер запущен на порту 3000'));
```

### Facebook OAuth: вход через Facebook

Facebook предъявляет строгие требования к OAuth: HTTPS обязателен для redirect URI, а домен должен быть добавлен в настройки приложения. Это делает туннель необходимым для локальной разработки — без него Facebook OAuth невозможно протестировать на localhost.

#### Настройка в консоли разработчика

1. Откройте [Facebook for Developers](https://developers.facebook.com/apps/).
2. Создайте или выберите приложение.
3. Добавьте продукт **Facebook Login**.
4. В **Settings > Basic** добавьте `oauth-dev.fxtun.ru` в **App Domains**.
5. В **Facebook Login > Settings** добавьте `https://oauth-dev.fxtun.ru/auth/facebook/callback` в **Valid OAuth Redirect URIs**.
6. Скопируйте **App ID** и **App Secret**.

#### Пример на Express.js

```javascript
// facebook-oauth.js
const express = require('express');
const axios = require('axios');
const app = express();

const APP_ID = process.env.FACEBOOK_APP_ID;
const APP_SECRET = process.env.FACEBOOK_APP_SECRET;
const REDIRECT_URI = 'https://oauth-dev.fxtun.ru/auth/facebook/callback';

app.get('/auth/facebook', (req, res) => {
  const params = new URLSearchParams({
    client_id: APP_ID,
    redirect_uri: REDIRECT_URI,
    scope: 'email,public_profile',
    response_type: 'code',
  });
  res.redirect(`https://www.facebook.com/v18.0/dialog/oauth?${params}`);
});

app.get('/auth/facebook/callback', async (req, res) => {
  const { code } = req.query;

  if (!code) {
    return res.status(400).send('Отсутствует код авторизации');
  }

  try {
    // Обмен кода на access token
    const tokenResponse = await axios.get(
      'https://graph.facebook.com/v18.0/oauth/access_token', {
        params: {
          client_id: APP_ID,
          client_secret: APP_SECRET,
          redirect_uri: REDIRECT_URI,
          code: code,
        },
      }
    );

    const accessToken = tokenResponse.data.access_token;

    // Получение профиля пользователя
    const userResponse = await axios.get('https://graph.facebook.com/me', {
      params: {
        fields: 'id,name,email',
        access_token: accessToken,
      },
    });

    console.log('Facebook пользователь:', userResponse.data.name);
    res.json(userResponse.data);
  } catch (error) {
    console.error('Ошибка OAuth:', error.message);
    res.status(500).send('Аутентификация не удалась');
  }
});

app.listen(3000, () => console.log('Сервер запущен на порту 3000'));
```

## Использование туннеля с OAuth в разных фреймворках

Подход с fxTunnel работает с любым веб-фреймворком. Ниже — конфигурационные сниппеты для популярных стеков. Во всех случаях единственное изменение — указание URL туннеля как redirect URI.

### Next.js с NextAuth.js

```javascript
// pages/api/auth/[...nextauth].js
import NextAuth from 'next-auth';
import GoogleProvider from 'next-auth/providers/google';
import GitHubProvider from 'next-auth/providers/github';

export default NextAuth({
  providers: [
    GoogleProvider({
      clientId: process.env.GOOGLE_CLIENT_ID,
      clientSecret: process.env.GOOGLE_CLIENT_SECRET,
    }),
    GitHubProvider({
      clientId: process.env.GITHUB_CLIENT_ID,
      clientSecret: process.env.GITHUB_CLIENT_SECRET,
    }),
  ],
});
```

```bash
# .env.local
NEXTAUTH_URL=https://oauth-dev.fxtun.ru
GOOGLE_CLIENT_ID=xxx
GOOGLE_CLIENT_SECRET=yyy
GITHUB_CLIENT_ID=xxx
GITHUB_CLIENT_SECRET=yyy
```

NextAuth.js использует `NEXTAUTH_URL` для автоматического формирования redirect URI. Укажите URL туннеля — и всё работает.

### Django с django-allauth

```python
# settings.py
SITE_ID = 1

# Устанавливаем URL туннеля как базовый
ACCOUNT_DEFAULT_HTTP_PROTOCOL = 'https'

SOCIALACCOUNT_PROVIDERS = {
    'google': {
        'APP': {
            'client_id': os.environ['GOOGLE_CLIENT_ID'],
            'secret': os.environ['GOOGLE_CLIENT_SECRET'],
        },
        'SCOPE': ['profile', 'email'],
    },
    'github': {
        'APP': {
            'client_id': os.environ['GITHUB_CLIENT_ID'],
            'secret': os.environ['GITHUB_CLIENT_SECRET'],
        },
        'SCOPE': ['user:email'],
    },
}
```

```bash
# Обновляем объект Django Site, указывая URL туннеля
python manage.py shell -c "
from django.contrib.sites.models import Site
site = Site.objects.get(id=1)
site.domain = 'oauth-dev.fxtun.ru'
site.name = 'Dev (tunnel)'
site.save()
"
```

### Ruby on Rails с OmniAuth

```ruby
# config/initializers/omniauth.rb
Rails.application.config.middleware.use OmniAuth::Builder do
  provider :google_oauth2,
    ENV['GOOGLE_CLIENT_ID'],
    ENV['GOOGLE_CLIENT_SECRET'],
    { callback_path: '/auth/google/callback' }

  provider :github,
    ENV['GITHUB_CLIENT_ID'],
    ENV['GITHUB_CLIENT_SECRET'],
    { callback_path: '/auth/github/callback' }
end
```

```bash
# Устанавливаем URL туннеля как хост для разработки
RAILS_HOST=oauth-dev.fxtun.ru rails server -p 3000
```

### Go с golang.org/x/oauth2

```go
package main

import (
    "fmt"
    "net/http"
    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
)

var googleOauthConfig = &oauth2.Config{
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
    // URL туннеля как redirect
    RedirectURL:  "https://oauth-dev.fxtun.ru/auth/google/callback",
    Scopes:       []string{"profile", "email"},
    Endpoint:     google.Endpoint,
}

func handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
    url := googleOauthConfig.AuthCodeURL("state-token")
    http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
    code := r.URL.Query().Get("code")
    token, err := googleOauthConfig.Exchange(r.Context(), code)
    if err != nil {
        http.Error(w, "Ошибка обмена токена", http.StatusBadRequest)
        return
    }
    fmt.Fprintf(w, "Access token: %s", token.AccessToken[:20]+"...")
}

func main() {
    http.HandleFunc("/auth/google", handleGoogleLogin)
    http.HandleFunc("/auth/google/callback", handleGoogleCallback)
    fmt.Println("Сервер запущен на порту 3000")
    http.ListenAndServe(":3000", nil)
}
```

## Динамический redirect URI через переменные окружения

Хардкодить URL туннеля в приложении ненадёжно — URL может меняться между сессиями или разработчиками. Лучшая практика — использовать переменную окружения для базового URL и формировать redirect URI динамически.

```javascript
// config.js — динамический redirect URI
const BASE_URL = process.env.BASE_URL || 'http://localhost:3000';

module.exports = {
  google: {
    callbackURL: `${BASE_URL}/auth/google/callback`,
  },
  github: {
    callbackURL: `${BASE_URL}/auth/github/callback`,
  },
  facebook: {
    callbackURL: `${BASE_URL}/auth/facebook/callback`,
  },
};
```

```bash
# Запуск с URL туннеля
BASE_URL=https://oauth-dev.fxtun.ru node app.js

# Запуск без туннеля (только локально)
node app.js
# → используется http://localhost:3000 по умолчанию
```

Такой подход позволяет одному и тому же коду работать как с туннелем, так и без него. Разработчики в команде устанавливают `BASE_URL` на свой URL туннеля, и приложение адаптируется автоматически.

## Troubleshooting: типичные проблемы OAuth + туннель

Большинство проблем OAuth с туннелем сводятся к трём вещам: redirect URI не совпадает, туннель не запущен или провайдер отклоняет домен. [fxTunnel Inspector](/blog/traffic-inspector-debug-requests/) поможет увидеть, дошёл ли callback-запрос до туннеля.

| Проблема | Возможная причина | Решение |
|---|---|---|
| Ошибка `redirect_uri_mismatch` | URI в коде не совпадает с URI в настройках провайдера | Скопируйте точный URL туннеля из вывода fxTunnel и вставьте его в консоль разработчика провайдера. Убедитесь, что путь совпадает точно (включая завершающие слэши). |
| Страница OAuth загружается, но callback не срабатывает | Туннель не запущен | Убедитесь, что `fxtunnel` работает. Проверьте доступность URL, открыв его в браузере. |
| Ошибка `invalid_client` | Неверный client ID или secret | Перепроверьте переменные окружения. Используйте учётные данные для среды разработки, а не продакшена. |
| Бесконечный цикл редиректов | Сессионная cookie не сохраняется | Установите домен cookie, совпадающий с URL туннеля. Для Express: `cookie: { domain: '.fxtun.ru', secure: true }`. |
| Ошибки CORS после callback | Фронтенд и бэкенд на разных origin | Убедитесь, что фронтенд использует тот же URL туннеля, что и бэкенд, или настройте CORS для домена туннеля. |
| Провайдер отклоняет домен | Домен не добавлен в настройки провайдера | Добавьте `oauth-dev.fxtun.ru` в список разрешённых доменов в консоли разработчика провайдера (Google, Facebook это требуют). |
| Обмен токена не срабатывает | Бэкенд не может достучаться до API провайдера | Это не проблема туннеля — бэкенд вызывает API провайдера напрямую, не через туннель. Проверьте интернет-соединение и правила файрвола. |
| URL изменился после перезапуска | Туннель запущен без `--domain` — поддомен каждый раз случайный | Зарезервируйте поддомен (`fxtunnel domains add oauth-dev`, с тарифа Base) и всегда запускайте с `--domain oauth-dev` |

## Лучшие практики тестирования OAuth

Если наладить тестирование OAuth при локальной разработке, это сэкономит часы отладки и не пустит проблемы безопасности в продакшен. Эти пять принципов работают с любым [инструментом туннелирования](/blog/what-is-tunneling/).

### 1. Используйте отдельные OAuth-приложения для разработки

Создайте отдельное OAuth-приложение в консоли каждого провайдера для локальной разработки. Не используйте продакшен-учётные данные при тестировании. Это предотвращает случайные утечки данных и поддерживает чистоту списка redirect URI.

### 2. Храните все секреты в переменных окружения

Никогда не хардкодите client ID, client secret или redirect URI в исходном коде. Используйте переменные окружения или файл `.env` (исключённый из системы контроля версий через `.gitignore`).

```bash
# .env — никогда не коммитьте этот файл
GOOGLE_CLIENT_ID=123456789.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-...
GITHUB_CLIENT_ID=Iv1.abc123
GITHUB_CLIENT_SECRET=abc123def456
BASE_URL=https://oauth-dev.fxtun.ru
```

### 3. Тестируйте полный поток

Не ограничивайтесь тестированием только happy path. Проверьте, что приложение обрабатывает:

- **Отклонение разрешений** — пользователь нажимает «Отмена» на экране согласия.
- **Истёкшие коды авторизации** — код имеет короткий срок жизни (обычно 10 минут).
- **Недействительный параметр state** — CSRF-защита должна отклонять несовпадающие state-токены.
- **Недостающие scopes** — пользователь даёт меньше разрешений, чем запрошено.

Функция [Inspector + Replay](/blog/traffic-inspector-debug-requests/) в fxTunnel позволяет легко повторить callback-запросы без прохождения всего OAuth-потока заново.

### 4. Валидируйте параметр state

Всегда используйте параметр `state` для защиты от CSRF-атак. Сгенерируйте случайный токен, сохраните его в сессии, включите в URL авторизации и проверьте при получении callback.

```javascript
const crypto = require('crypto');

// Генерация state перед редиректом
app.get('/auth/google', (req, res) => {
  const state = crypto.randomBytes(16).toString('hex');
  req.session.oauthState = state;
  const url = googleOauthConfig.authorizationUrl({ state });
  res.redirect(url);
});

// Проверка state на callback
app.get('/auth/google/callback', (req, res) => {
  if (req.query.state !== req.session.oauthState) {
    return res.status(403).send('Недействительный параметр state');
  }
  // ... обмен кода на токен
});
```

### 5. Закрывайте туннель по завершении

Запущенный туннель открывает доступ к вашему локальному приложению из интернета. Когда тестирование OAuth завершено, остановите туннель комбинацией `Ctrl+C`. Для командных окружений [кастомные домены](/blog/custom-domain-tunnel-setup/) дают каждому разработчику стабильный, предсказуемый URL.

## Почему fxTunnel удобен для тестирования OAuth callback

fxTunnel -- SaaS-туннель с бесплатным тарифом и автоматическим HTTPS для каждого туннеля. Для тестирования OAuth это означает, что одна команда `fxtunnel http 3000 --domain oauth-dev` заменяет весь пайплайн деплоя. Вот как он выглядит на фоне [других инструментов](/blog/ngrok-vs-cloudflare-vs-fxtunnel/).

| Возможность | fxTunnel | Другие инструменты |
|---|---|---|
| Бесплатный тариф | 1 туннель, без резерва поддомена | Часто ограничен по времени или запросам |
| HTTPS | Автоматически для каждого туннеля | Обычно автоматически, но может требовать настройки |
| Стабильный URL | При закреплённом поддомене (`--domain`), с тарифа Base | Часто меняется при каждом перезапуске (бесплатные тарифы) |
| Кастомный домен (свой DNS) | С тарифа Base | Дороже или недоступно |
| Inspector + Replay | С тарифа Base — просмотр всех OAuth callback | Отсутствует или требует отдельного инструмента |
| Настройка | Одна команда, 30 секунд | Часто требует конфигурации и регистрации |
| Открытость | Клиент открыт (GitHub), сервер можно развернуть у себя | Часто полностью проприетарный |

Более широкое сравнение -- в [рейтинге инструментов туннелирования 2026](/blog/best-tunneling-tools-2026/).

