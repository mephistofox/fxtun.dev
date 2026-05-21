Generate a blog article (RU) for the fxTunnel dev diary based on $ARGUMENTS.

If $ARGUMENTS is empty — scan CHANGELOG and list uncovered topics, then stop.
If $ARGUMENTS is a version (e.g. "v3.0.0") — write about features from that version.
If $ARGUMENTS is a topic (e.g. "oauth") — write about that topic across versions.

---

## Step 1: Discovery

1. Read CHANGELOG: `/home/fxcode/Проекты/fxTunnel/CHANGELOG.md`
2. Read existing blog post titles to find covered topics:
   ```bash
   head -1 /home/fxcode/Проекты/fxTunnel/docs/blog/[0-9]*.md
   ```
3. Determine the next article number:
   ```bash
   ls /home/fxcode/Проекты/fxTunnel/docs/blog/[0-9]*.md | sort -V | tail -1
   ```
4. Identify CHANGELOG features NOT covered by existing posts
5. If $ARGUMENTS is empty — print uncovered topics with suggested titles and STOP

## Step 2: Research

1. Read the relevant CHANGELOG entries for the target topic
2. Read the actual source code. Use grep and file reads to find key files under `/home/fxcode/Проекты/fxTunnel/`
3. Read 2-3 reference posts for voice and style:
   - `/home/fxcode/Проекты/fxTunnel/docs/blog/01-idea-and-prototype.md` (original voice, tone)
   - The LATEST post for current style evolution
4. Extract real numbers from source code: buffer sizes, timeouts, limits, line counts, struct fields
5. Understand the feature deeply enough to explain it informally with real implementation details

## Step 3: Write the Article

Create file: `/home/fxcode/Проекты/fxTunnel/docs/blog/[NN]-[slug].md`

Where:
- `[NN]` = two-digit article number (19, 20, etc.)
- `[slug]` = short kebab-case topic name in English (e.g. `payments`, `oauth`, `ssg-seo`)

### Format rules

**Title line:**
```
# Пишу свой ngrok на Go: [тема]
```

**Subtitle line:**
```
*Часть N серии «Пишу свой ngrok на Go: дневник разработки»*
```

Then a horizontal rule `---`.

**Section headers** — use markdown `##` headers (this is a full blog article, NOT Telegram).

**Body rules:**
- First person: "я", "мне", "у меня". NEVER "мы", NEVER third person.
- Informal Russian. Use `...` (three dots) as pauses between thoughts.
- Short paragraphs: 2-4 sentences max.
- Self-irony and humor where natural: "Звучит как безумие? Наверное..."
- Real numbers from code: buffer sizes, timeouts, line counts.
- Accessible analogies for complex concepts.
- NO "данный", "осуществляется", "вышеуказанный", "следует отметить" or other formal Russian.

**Code blocks** — real code from the project, with Go syntax highlighting. Show key structs, interfaces, important functions. Each block should be meaningful, not decorative.

**Structure for each article:**
1. **Контекст** — date, what was the problem, why it matters (2-3 paragraphs)
2. **Main sections** — 3-5 sections covering the implementation. Each named after the feature/concept, NOT generic names like "Подход". Use real names: "State file", "Локальный HTTP API", etc.
3. **Грабли** — what went wrong, unexpected issues (2-3 paragraphs)
4. **Итог** — date, bullet list of what was achieved, relevant commit hashes
5. **Teaser** — "В следующей части" hint (1-2 sentences)

### Voice examples (DO copy this tone):

GOOD:
```
Все нормальные люди готовят оливье, а я сижу и злюсь на ngrok...
```
```
Звучит как безумие? Наверное... Но руки чешутся — значит надо делать...
```
```
Одна команда — весь продукт... Никаких npm install, никаких виртуальных окружений...
```

BAD (never write like this):
```
В данной статье мы рассмотрим реализацию...
```
```
Следует отметить, что данный подход обеспечивает...
```

### Length target
- 2000-4000 words per article
- Long enough for depth, short enough to stay engaging

## Step 4: Generate Cover Image

1. Read existing image prompts for style reference:
   `/home/fxcode/Проекты/fxTunnel/docs/blog/image-prompts.md`

2. Write an image prompt following the established style:
   - Cyber-industrial, dark background (#0a0a0f)
   - Neon green (#BFFF00) as accent, purple (#8B5CF6) as secondary
   - Minimal flat vector style with glow effects and grid background
   - No text on image (except technical labels like "404", IP addresses)
   - 16:9 ratio, horizontal orientation

3. Generate the cover image using this Python code (run via `python3 -c`):

```python
import base64, re, json, os, urllib.request

api_key = os.environ["FXTUN_LLM_KEY"]  # export FXTUN_LLM_KEY=sk_live_... before running

body = json.dumps({
    "model": "gemini-3-pro-image",
    "messages": [{"role": "user", "content": "YOUR_PROMPT_HERE"}],
    "size": "1024x1024"
}).encode()

req = urllib.request.Request(
    "https://llm.fxtun.dev/v1/chat/completions",
    data=body,
    headers={
        "Content-Type": "application/json",
        "Authorization": f"Bearer {api_key}"
    }
)

resp = urllib.request.urlopen(req, timeout=120)
data = json.loads(resp.read())
content = data["choices"][0]["message"]["content"]
match = re.search(r"data:image/([\w+]+);base64,([A-Za-z0-9+/=\s]+)", content)
if match:
    fmt = match.group(1).replace("jpeg", "jpg")
    img = base64.b64decode(match.group(2))
    path = "/home/fxcode/Проекты/fxTunnel/docs/blog/covers/NN-slug.FORMAT"
    open(path, "wb").write(img)
    print(f"Cover saved: {path} ({len(img)} bytes)")
else:
    print("No image in response")
    print(content[:300])
```

Replace:
- `YOUR_PROMPT_HERE` with the generated image prompt
- `NN-slug` with the article filename
- `FORMAT` with detected format (jpg/png)

4. Append the new prompt to `image-prompts.md`

## Step 5: Validate

1. Verify the article reads naturally and follows the voice guidelines
2. Verify all code blocks contain real code from the project (not invented)
3. Verify commit hashes referenced actually exist
4. Check that the cover image was generated successfully
5. Report to user:
   - Article number and file path
   - Word count
   - Topic covered
   - Cover image path (if generated)

---

## Article numbering reference

Articles 01-18 exist. Topics covered:
- 01: Idea and first prototype
- 02: From console to product in one day
- 03: First deploy and localhost
- 04: Auto-refresh, 404, first production feature
- 05: Full GUI redesign
- 06: Security in one day
- 07: Tests, races, panics
- 08: Squeezing performance
- 09: Traffic inspection
- 10: Retrospective
- 11: Custom domains and TLS
- 12: Refactoring
- 13: Product polish
- 14: CLI UX — init, browser login, pretty output
- 15: OAuth
- 16: QUIC and connection pooling
- 17: Plans and admin panel
- 18: Daemon mode

## Uncovered topics (candidates for articles 19+):

1. **Платёжная система** — Robokassa → YooKassa, subscriptions, USD/RUB exchange (v2.3-v3.0)
2. **SSG + SEO** — vite-ssg, JSON-LD, meta tags, sitemap, landing redesign (v3.0-v3.2)
3. **DDoS защита** — rate limiting, CORS, CSP, security headers (v3.3)
4. **Plans system** — plan limits, subscription lifecycle, payment integration (v2.1-v2.10)
5. **Email notifications** — SMTP, template redesign (v2.6-v2.12)
6. **Auto-update** — forced client updates, version checking (v2.0)
7. **i18n + locale detection** — domain-based locale, SSG routes (v3.3)
