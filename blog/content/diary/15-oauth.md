---
title: "Пишу свой ngrok на Go: OAuth — GitHub, Google и один день на всё"
date: "2026-02-01"
description: "OAuth через GitHub и Google за один день: от миграции базы до линковки и мержа задвоившихся аккаунтов, когда один человек залогинился дважды."
series: "Пишу свой ngrok на Go"
part: 15
---

## Контекст
До сих пор единственный способ входа — телефон + пароль. Для dev-тула это архаично. Разработчик хочет нажать «Login with GitHub» и через три секунды быть внутри. Не вводить телефон, не придумывать пароль, не подтверждать SMS. OAuth — стандарт индустрии, и если его нет — продукт выглядит как поделка.

GitHub — основной провайдер, потому что целевая аудитория — разработчики. Google — второй, потому что у всех есть Google-аккаунт.

## Как работает OAuth (для начинающих)

Если вы никогда не реализовывали OAuth — вот суть. Проблема: пользователь хочет залогиниться в ваш сервис, но не хочет придумывать ещё один пароль. Решение: пусть логинится через сервис, которому уже доверяет (GitHub, Google).

Флоу:
1. Пользователь нажимает «Login with GitHub» на вашем сайте
2. Вы перенаправляете его на `github.com/login/oauth/authorize?client_id=xxx&redirect_uri=yyy`
3. GitHub показывает: «Приложение fxTunnel хочет получить доступ к вашему профилю. Разрешить?»
4. Пользователь нажимает «Authorize»
5. GitHub перенаправляет обратно на ваш `redirect_uri` с параметром `code`
6. Ваш сервер обменивает этот `code` на `access_token`, обращаясь к GitHub API
7. С `access_token` запрашивает профиль пользователя (имя, email, аватарку)
8. Создаёт или находит пользователя в базе, выдаёт JWT

Два ключевых момента: `redirect_uri` должен совпадать с тем, что указан в настройках OAuth-приложения на GitHub (иначе GitHub отклонит запрос). И обмен `code` на токен — серверный, не клиентский. `code` живёт 10 минут и одноразовый.

## Миграция базы данных

Начинаем с фундамента. В таблице `users` не было полей для OAuth. Добавляем:

```sql
ALTER TABLE users ADD COLUMN github_id TEXT;
ALTER TABLE users ADD COLUMN google_id TEXT;
ALTER TABLE users ADD COLUMN email TEXT;
ALTER TABLE users ADD COLUMN avatar_url TEXT;
ALTER TABLE users ADD COLUMN display_name TEXT;
```

`github_id` и `google_id` — уникальные идентификаторы пользователя в соответствующих сервисах. По ним будем искать: «этот GitHub-аккаунт уже привязан к кому-то?» Плюс новые методы в репозитории: `GetByGitHubID`, `GetByGoogleID`, `CreateOAuth`.

## Ядро: RegisterOrLoginOAuth

Центральная функция — одна точка входа для любого OAuth-провайдера:

```go
func (s *Service) RegisterOrLoginOAuth(info *OAuthUserInfo, userAgent, ipAddress string) (*database.User, *TokenPair, error) {
    user, err := s.db.Users.GetByGitHubID(info.GitHubID)
    if err != nil && !errors.Is(err, database.ErrUserNotFound) {
        return nil, nil, fmt.Errorf("get user by github id: %w", err)
    }
    if user == nil {
        user = &database.User{
            DisplayName: info.DisplayName, IsActive: true,
            GitHubID: &info.GitHubID, Email: info.Email, AvatarURL: info.AvatarURL,
        }
        s.db.Users.CreateOAuth(user)
    }
    tokenPair, refreshTokenHash, _ := s.jwt.GenerateTokenPair(user.ID, user.Phone, user.IsAdmin)
    session := &database.Session{UserID: user.ID, RefreshTokenHash: refreshTokenHash, UserAgent: userAgent, IPAddress: ipAddress}
    s.db.Sessions.Create(session)
    return user, tokenPair, nil
}
```

Логика простая: ищем пользователя по `github_id`. Нашли — логиним. Не нашли — создаём. В обоих случаях — генерируем JWT-пару и создаём сессию. Тот же паттерн, что для обычного логина (статья 6), только вместо пароля — OAuth.

## Проблема дубликатов

Вот сценарий, который я не сразу увидел. Пользователь зарегистрировался по телефону месяц назад. Создал туннели, зарезервировал поддомены. Потом нажимает «Login with GitHub» — и получает новый, пустой аккаунт. Два аккаунта, один человек. Его данные — в старом аккаунте, а логинится он в новый.

Два решения:
1. **Линковка** — пользователь может привязать GitHub к существующему аккаунту из профиля. Залогинился по телефону → зашёл в настройки → нажал «Link GitHub» → авторизовал → `github_id` записан в его аккаунт.
2. **Админский мерж** — если дубликат уже создан, админ может объединить два аккаунта.

Мерж — операция непростая. Нужно перенести все связанные записи:

```go
func (r *UserRepository) MergeUsers(primaryID, secondaryID int64) error {
    tx, _ := r.db.Begin()
    defer tx.Rollback()
    tables := []string{"sessions", "api_tokens", "reserved_domains", "totp_secrets",
        "custom_domains", "audit_logs", "user_history"}
    for _, table := range tables {
        tx.Exec(fmt.Sprintf(`UPDATE %s SET user_id = ? WHERE user_id = ?`, table), primaryID, secondaryID)
    }
    tx.Exec(`UPDATE OR IGNORE user_bundles SET user_id = ? WHERE user_id = ?`, primaryID, secondaryID)
    tx.Exec(`UPDATE users SET
        github_id = COALESCE(github_id, (SELECT github_id FROM users WHERE id = ?)),
        google_id = COALESCE(google_id, (SELECT google_id FROM users WHERE id = ?))
        WHERE id = ?`, secondaryID, secondaryID, primaryID)
    tx.Exec(`DELETE FROM users WHERE id = ?`, secondaryID)
    return tx.Commit()
}
```

Всё в одной транзакции. `UPDATE OR IGNORE` для `user_bundles` — потому что там уникальный ключ `(user_id, key)`, и если у обоих аккаунтов есть бандл с одинаковым ключом, конфликт игнорируется (остаётся бандл основного аккаунта). `COALESCE` для OAuth-полей — берём из вторичного аккаунта только то, чего нет в основном.

## Google OAuth

Второй провайдер добавился по тому же шаблону. Разница — в URL'ах и формате ответа. GitHub возвращает `login` как имя пользователя, Google — `name`. GitHub возвращает `avatar_url`, Google — `picture`. Но абстракция `OAuthUserInfo` скрывает эти различия — для ядра системы все провайдеры одинаковы.

Добавить третий провайдер (GitLab, Bitbucket) — это написать один адаптер и зарегистрировать маршруты.

## GUI: OAuth через системный браузер

В десктопном GUI-клиенте OAuth работает через системный браузер. Wails не встраивает WebView с полноценным браузером — нельзя просто открыть `github.com/login/oauth/authorize` внутри приложения. Да и не нужно: пользователь не доверяет вводить пароль GitHub в стороннее приложение.

Флоу: GUI открывает системный браузер → пользователь логинится → callback возвращает на сервер → сервер генерирует одноразовый код → GUI получает код и обменивает на JWT. По сути тот же device flow, что для CLI (статья 14), но с OAuth вместо пароля.

## Грабли

**Обязательное поле phone.** OAuth-регистрация ломалась, потому что в таблице `users` поле `phone` было `NOT NULL UNIQUE`. GitHub не даёт телефон — только email. Фикс: сделал `phone` nullable, добавил поддержку логина по email. Один коммит — `101e944` — а сломано было всё.

**Редизайн авторизации.** Когда добавил кнопку «Login with GitHub», стало очевидно: форма с телефоном и паролем выглядит устаревшей. Перерисовал обе страницы (логин и регистрация). GitHub — главная кнопка наверху. Телефон + пароль — ниже, мелким шрифтом. Это не просто косметика — это сигнал: «мы современный инструмент».

## Итог
1 февраля:
- **GitHub OAuth** — регистрация и логин в один клик
- **Google OAuth** — второй провайдер
- **Линковка аккаунтов** — привязка OAuth к существующему аккаунту
- **Мерж пользователей** — админское объединение дубликатов
- **GUI OAuth** — через системный браузер
- **Редизайн авторизации** — GitHub как основной способ входа
```
6fe35ea feat(config): add OAuth settings for GitHub
d7e88b8 feat(db): add OAuth fields to users table
54e83f6 feat(db): add OAuth user repository methods
5a1b099 feat(auth): add OAuth register/login and GitHub linking
c4c356d feat(api): add GitHub OAuth login, register, and account linking
fcdbc3e feat(web): add OAuth callback page
781484a feat(web): add GitHub OAuth button to login and register pages
d945d0e feat(web): add GitHub account linking to profile page
07836fb feat(web): redesign auth pages with GitHub as primary login method
35fc1bd feat(oauth): add Google OAuth as second provider alongside GitHub
430138a feat(admin): add user merge and password reset functionality
ca80658 feat(gui): add GitHub OAuth authentication flow
101e944 fix(auth): fix OAuth registration and support email-based login
```
13 коммитов за один день — от миграции БД до редизайна фронтенда. OAuth кажется простым, пока не начнёшь думать о дубликатах, линковке и nullable-полях. Но результат того стоит: разработчик нажимает одну кнопку и через три секунды — внутри.

---

**В следующей части:** QUIC — эксперимент, провал и план Б с connection pooling.
