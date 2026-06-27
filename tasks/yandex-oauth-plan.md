# План: добавить OAuth через Яндекс, затем скрыть GitHub/Google

## Контекст и решения (из брейншторма)
- Конечная цель: авторизация через Яндекс вместо GitHub/Google.
- **Порядок (безопасный):** сначала полностью добавляем Яндекс → деплой → пользователь
  входит через GitHub и привязывает Яндекс в профиле → только потом скрываем GitHub/Google.
- «Убрать GitHub/Google» = **скрыть кнопки в UI**. Бэкенд-эндпоинты и поля `github_id`/`google_id`
  в БД остаются (807 мигрированных юзеров не должны пострадать, откат тривиален).
- Конфиг Яндекса: **одно глобальное приложение** на оба домена (как Google).
- Охват: **web-дашборд + десктоп GUI**.
- Яндекс-приложение ещё не создано — код пишем под конфиг, redirect URI выдадим для регистрации.

## Технические детали Яндекс OAuth
- Authorize: `https://oauth.yandex.ru/authorize?response_type=code&client_id=..&redirect_uri=..&state=..`
- Token: `POST https://oauth.yandex.ru/token` (form: grant_type=authorization_code, code, client_id, client_secret)
- User info: `GET https://login.yandex.ru/info?format=json`, заголовок `Authorization: OAuth <token>`
  - поля: `id` (строка), `login`, `default_email`, `display_name`/`real_name`, `default_avatar_id`
  - аватар: `https://avatars.yandex.net/get-yapic/{default_avatar_id}/islands-200`
- `yandex_id` — строка → колонка `VARCHAR(255)`, как `google_id`.
- Redirect URI для регистрации в Яндексе:
  - `https://fxtun.ru/api/auth/yandex/callback`
  - `https://fxtun.dev/api/auth/yandex/callback`

---

## Фаза 1 — Добавить Яндекс OAuth (этот батч)

### БД / sqlc
- [ ] Установить `sqlc` (`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`) — в PATH его нет
- [ ] Миграция `00009_add_yandex_id.sql`: `ADD COLUMN yandex_id VARCHAR(255)` + уникальный частичный индекс
- [ ] `queries/users.sql`: добавить `yandex_id` во все SELECT-списки колонок user; добавить
      `GetUserByYandexID`, `LinkYandex`; дополнить `CreateOAuthUser` колонкой `yandex_id`
- [ ] Перегенерировать sqlc (`cd internal/server/database && sqlc generate`)
- [ ] `models.go`: добавить `YandexID *string`; маппинг в `sqlcUserToDomain`
- [ ] `pg_user_repo.go`: `GetByYandexID`, `LinkYandex`, передать `YandexID` в `CreateOAuth`

### Конфиг
- [ ] `config/server.go`: `YandexOAuthSettings{ClientID, ClientSecret}` + поле `Yandex` в `OAuthSettings`
- [ ] `configs/server.yaml`: секция `oauth.yandex` (client_id/secret пустые-плейсхолдеры)

### auth-сервис
- [ ] `auth.go`: `YandexOAuthUserInfo`, `RegisterOrLoginYandexOAuth`, `LinkYandex` (зеркало Google)

### HTTP-хендлеры
- [ ] `handlers_oauth.go`: константы URL, типы, `handleYandexAuth`, `handleYandexLink`,
      `handleYandexCallback`, `handleYandexLinkCallback`, `exchangeYandexCode`, `getYandexUser`,
      `buildYandexRedirectURI` (зеркало Google)
- [ ] `api.go`: роуты `GET /api/auth/yandex`, `GET /api/auth/yandex/callback` (public),
      `POST /api/auth/yandex/link` (protected)

### Web frontend
- [ ] `LoginView.vue`: кнопка «Войти через Яндекс» (`href="/api/auth/yandex"`) + лого
- [ ] `ProfileView.vue`: строка Яндекса в Linked Accounts + обработка `?yandex_linked=true`
- [ ] типы профиля: `yandex_id?`
- [ ] i18n `en.json` + `ru.json`: ключи `auth.signInWithYandex`, `profile.yandexLinked` и т.д.
      (через скилл i18n-sync для паритета ru/en)

### GUI (десктоп)
- [ ] `AuthView.vue`: кнопка Яндекса (`handleOAuth('yandex')`) — бэкенд `StartOAuthFlow` уже generic
- [ ] проверить `gui/frontend/src/stores/auth.ts` (вероятно generic, без правок)

### Admin (опционально)
- [ ] `admin/src/api/types.ts`: `yandex_id?`; отображение в `UserDetailView.vue`

### Проверка
- [ ] `make build` (сервер собирается), `go test -race ./internal/server/...`, `make lint`
- [ ] code-review (обязателен перед коммитом по правилам проекта)
- [ ] коммит(ы) Conventional Commits; деплой согласуем отдельно

---

## Фаза 2 — Скрыть GitHub/Google (ПОЗЖЕ, после привязки Яндекса на проде)
> НЕ делать в одном батче с Фазой 1: пока ты не привязал Яндекс к своему аккаунту на проде,
> вход через GitHub должен оставаться, иначе потеряешь доступ.
- [ ] `LoginView.vue`: убрать/закомментировать кнопки GitHub и Google
- [ ] `ProfileView.vue`: убрать строки привязки GitHub/Google (оставить статус «привязано» по желанию)
- [ ] GUI `AuthView.vue`: убрать кнопки GitHub/Google
- [ ] Бэкенд-эндпоинты и поля БД НЕ трогаем

## Review (Фаза 1 — выполнено 2026-06-27)
- ✅ Реализовано: миграция 00009, sqlc-перегенерация, repo/auth/handlers/routes/config, web (Login/Profile/admin-view/i18n ru+en/типы), GUI AuthView.
- ✅ Верификация: `go build`, `go vet`, golangci-lint (мои пакеты чисто), web+GUI `vue-tsc`,
  полный `go test` api+database+auth против реального PG (одноразовый контейнер) — миграция 00009
  применяется чисто (version 9), все Яндекс-тесты link-callback зелёные.
- ✅ Code-review + security-review: реализация — корректное зеркало Google, новых уязвимостей нет.
- ➕ Доп. фикс из security-review: усилён `isLocalhostURI` (url.Parse + проверка host/userinfo) —
  закрывает известный HIGH open-redirect→ATO (audit-2026-06-26), переиспользуемый в desktop-флоу
  всех трёх провайдеров. Добавлен юнит-тест `TestIsLocalhostURI`.
- ⚠️ НЕ закоммичено (коммит не запрашивался). В рабочем дереве есть ПАРАЛЛЕЛЬНЫЕ чужие изменения
  (доменная консолидация/легал-ребрендинг: deploy/nginx-*, useSeo.ts, i18n/index.ts, легал-вью,
  rename в en/ru.json) — я их НЕ трогал. При коммите стейджить только Яндекс-файлы; en.json/ru.json
  содержат и мои ключи, и чужие легал-правки → нужен `git add -p`.
- 📌 Реальные client_id/secret прописаны в gitignored `configs/server.yaml`. Для прода нужно
  задать те же значения (oauth.yandex.* или FXTUNNEL_OAUTH_YANDEX_*) — согласовать при деплое.
- 📌 Primary redirect URI = https://fxtun.ru/api/auth/yandex/callback (fxtun.dev 301→fxtun.ru).
