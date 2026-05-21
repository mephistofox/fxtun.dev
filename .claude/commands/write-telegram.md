Generate a Telegram dev diary post (RU) for the fxTunnel project based on $ARGUMENTS.

If $ARGUMENTS is empty — scan CHANGELOG and list uncovered topics, then stop.
If $ARGUMENTS is a version (e.g. "v3.0.0") — write about features from that version.
If $ARGUMENTS is a topic (e.g. "oauth") — write about that topic across versions.

---

## Step 1: Discovery

1. Read CHANGELOG: `/home/fxcode/Проекты/fxTunnel/CHANGELOG.md`
2. Read existing Telegram post titles to find covered topics:
   ```bash
   head -1 /home/fxcode/Проекты/fxTunnel/docs/blog/telegram/*.md
   ```
3. Determine the next part number:
   ```bash
   ls /home/fxcode/Проекты/fxTunnel/docs/blog/telegram/*.md | sort -t/ -k9 -V | tail -1
   ```
   Current latest is part 18. Next is 19.
4. Identify CHANGELOG features NOT covered by existing posts (parts 01-18 cover approximately v1.0 through v2.2 daemon mode)
5. If $ARGUMENTS is empty — print uncovered topics with suggested titles and STOP

## Step 2: Research

1. Read the relevant CHANGELOG entries for the target topic
2. Read the actual source code. Use `grep -r` and file reads to find key files under `/home/fxcode/Проекты/fxTunnel/`
3. Read 2-3 reference posts for voice and style:
   - `/home/fxcode/Проекты/fxTunnel/docs/blog/telegram/01-1.md` (original voice, tone, format)
   - The LATEST post (currently `18-2.md`) for current style evolution
4. Extract real numbers from source code: buffer sizes, timeouts, limits, line counts, struct fields
5. Understand the feature deeply enough to explain it informally with real implementation details

## Step 3: Write

Create file: `/home/fxcode/Проекты/fxTunnel/docs/blog/telegram/[NN]-[part].md`

Where:
- `[NN]` = two-digit part number (19, 20, etc.)
- `[part]` = 1 for first part, 2 for second part (if split needed)

### Format rules

**Title line:**
```
*Пишу свой ngrok на Go: [тема]*
```

**Subtitle line:**
```
_Часть N серии «Пишу свой ngrok на Go: дневник разработки»_
```

**Section headers** — use `*bold*` text, NOT markdown headers (# ## ###). Telegram does NOT render markdown headers.

**Body rules:**
- First person: "я", "мне", "у меня". NEVER "мы", NEVER third person.
- Informal Russian. Use `...` (three dots) as pauses between thoughts.
- Short paragraphs: 2-4 sentences max.
- Self-irony and humor where natural: "Звучит как безумие? Наверное..."
- Real numbers from code: buffer sizes, timeouts, line counts.
- Accessible analogies for complex concepts.
- NO emoji. None. Zero.
- NO "данный", "осуществляется", "вышеуказанный", "следует отметить" or other formal Russian.
- NO markdown headers (# ## ###) — Telegram does not render them.

**Markers** — use `▸` for bullet lists, `•` for simple lists:
```
▸ *Причина 1...* Пояснение...
▸ *Причина 2...* Пояснение...
```

**Code blocks** — up to 20 lines. Practical, real code from the project. Use triple backticks with language hint:
```go
// Real code from the project
```

**Spoilers** — use `||spoiler||` for punchlines or surprising results.

**Structure for each post:**
1. **Контекст** — what was the problem, why it matters (2-3 paragraphs)
2. **Подход** — how I solved it, key decisions (3-5 paragraphs with code)
3. **Грабли** — what went wrong, unexpected issues (2-3 paragraphs)
4. **Итог** — what was achieved, bullet list of results (1-2 paragraphs + list)
5. **Тизер** — hint at next topic (1 sentence)

### Voice examples (DO copy this tone):

GOOD:
```
Всё нормальные люди готовят оливье, а я сижу и злюсь на ngrok...
```
```
Звучит как безумие? Наверное... Но руки чешутся — значит надо делать...
```
```
Одна команда — весь продукт... Никаких npm install, никаких виртуальных окружений...
```
```
Линтер прав — хорошие привычки не зависят от контекста...
```

BAD (never write like this):
```
В данной статье мы рассмотрим реализацию...
```
```
Следует отметить, что данный подход обеспечивает...
```
```
Нами было принято решение использовать...
```

## Step 4: Validate

1. Count characters:
   ```bash
   wc -c /home/fxcode/Проекты/fxTunnel/docs/blog/telegram/[NN]-[part].md
   ```
2. If > 4000 characters — split into parts:
   - Part 1 (`[NN]-1.md`): context + approach + start of implementation
   - Part 2 (`[NN]-2.md`): grables (issues) + results + teaser
   - Each part must be self-contained and readable independently
   - Part 2 starts with a brief recap: `*[Тема]: грабли и результаты*`
3. Verify NO markdown headers (# ## ###) are present in the output
4. Report to user:
   - Part number assigned
   - Character count per file
   - Topic covered
   - Whether split was needed

---

## Post numbering reference

Parts 01-18 are taken. Topics covered approximately:
- 01: Idea and first prototype (basic tunnel concept)
- 02-03: TCP connection, yamux multiplexing
- 04-05: HTTP tunneling, subdomain routing
- 06: TLS, authentication
- 07-08: UDP tunneling
- 09-10: GUI (Wails desktop app)
- 11-12: Admin panel, web management
- 13-14: Request inspector
- 15-16: Custom domains, WebSocket
- 17: Security (rate limiting, CORS, headers)
- 18: Daemon mode

## Uncovered topics (candidates for parts 19+):

1. **OAuth + Device Flow** — GitHub/Google login, browser-based CLI auth (v1.17-v1.19)
2. **Платёжная система** — Robokassa → YooKassa, subscriptions, USD/RUB exchange (v2.3-v3.0)
3. **SSG + SEO** — vite-ssg, JSON-LD, meta tags, sitemap, landing redesign (v3.0-v3.2)
4. **DDoS защита** — rate limiting, CORS, CSP, security headers (v3.3)
5. **QUIC эксперимент** — attempt and rollback to yamux (v1.18-v1.19)
6. **Plans system** — plan limits, subscription lifecycle, payment integration (v2.1-v2.10)
7. **Email notifications** — SMTP, template redesign (v2.6-v2.12)
8. **Auto-update** — forced client updates, version checking (v2.0)
9. **i18n + locale detection** — domain-based locale, SSG routes (v3.3)
10. **Connection pooling** — multi-session pools, binary stream headers (v1.19)
