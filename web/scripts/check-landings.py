#!/usr/bin/env python3
"""Проверяет, что у каждого поискового лендинга есть полная копия.

Запуск из корня репозитория:
    python3 web/scripts/check-landings.py

Код возврата 1, если у лендинга нет текстов, тексты неполные, SEO-поля вышли
за длину, собранная страница не содержит своего заголовка или её hreflang
ведёт на несуществующую английскую копию. Для документации — SEO-поля
frontmatter и пререндер каждой страницы из manifest.json.
"""
import json
import os
from html.parser import HTMLParser
import pathlib
import re
import sys

WEB = pathlib.Path(__file__).resolve().parents[1]

# Сборка для локальной проверки может лежать не в web/dist (см. FXTUN_WEB_OUT в vite.config.ts).
DIST = pathlib.Path(os.environ.get("FXTUN_WEB_OUT") or WEB / "dist")

# (пространство имён перевода, ключ SEO, слаг страницы)
LANDINGS = [
    ("ngrokAlt", "ngrokAlternative", "ngrok-alternative"),
    ("minecraft", "minecraftServer", "minecraft-server"),
    ("noWhiteIp", "noWhiteIp", "bez-belogo-ip"),
    ("portForward", "portForwarding", "probros-portov"),
    ("remoteAccess", "remoteAccess", "udalennyy-dostup"),
    ("hamachiAlt", "hamachiAlternative", "analog-hamachi"),
]

# ключ -> (тип, минимум элементов для списка)
SHAPE = {
    "eyebrow": (str, 0),
    "h1": (str, 0),
    "intro": (str, 0),
    "whyTitle": (str, 0),
    "whyParagraphs": (list, 2),
    "cardsTitle": (str, 0),
    "cards": (list, 3),
    "stepsTitle": (str, 0),
    "steps": (list, 3),
    "faq": (list, 4),
    "relatedTitle": (str, 0),
    "related": (list, 2),
    "ctaTitle": (str, 0),
    "ctaButton": (str, 0),
}

ITEM_KEYS = {
    "cards": ("title", "desc"),
    "steps": ("title", "desc"),
    "faq": ("q", "a"),
    "related": ("to", "label"),
}

TITLE_MAX = 60
DESC_MIN, DESC_MAX = 120, 160

BANNED = ("ИИ", "нейросет", "LLM", "автогенерац")

DOCS = WEB / "src" / "docs"

# vue-i18n требует экранировать вертикальную черту как {'|'}; в выдаче это
# один символ, поэтому меряем то, что увидит поисковик, а не исходник.
PIPE_LITERAL = "{'|'}"


def rendered(text):
    return text.replace(PIPE_LITERAL, "|")


def check_copy(ru, ns, seo_key, problems):
    block = ru.get("useCase", {}).get(ns)
    if not block:
        problems.append(f"{ns}: нет блока useCase.{ns}")
        return

    for name, (kind, minimum) in SHAPE.items():
        value = block.get(name)
        if value is None:
            problems.append(f"{ns}: нет ключа {name}")
            continue
        if not isinstance(value, kind):
            problems.append(f"{ns}.{name}: ожидался {kind.__name__}")
            continue
        if kind is str and not value.strip():
            problems.append(f"{ns}.{name}: пустая строка")
        if kind is list:
            if len(value) < minimum:
                problems.append(
                    f"{ns}.{name}: элементов {len(value)}, нужно минимум {minimum}"
                )
            for i, item in enumerate(value):
                fields = ITEM_KEYS.get(name, ())
                if fields and not isinstance(item, dict):
                    problems.append(f"{ns}.{name}[{i}]: элемент не объект")
                    continue
                for field in fields:
                    if not str(item.get(field, "")).strip():
                        problems.append(f"{ns}.{name}[{i}]: нет поля {field}")

    seo = ru.get("seo", {}).get(seo_key)
    if not seo:
        problems.append(f"{ns}: нет блока seo.{seo_key}")
        return
    title, desc = rendered(seo.get("title", "")), seo.get("description", "")
    if len(title) > TITLE_MAX:
        problems.append(f"seo.{seo_key}.title: {len(title)} символов, максимум {TITLE_MAX}")
    if not DESC_MIN <= len(desc) <= DESC_MAX:
        problems.append(
            f"seo.{seo_key}.description: {len(desc)} символов, нужно {DESC_MIN}-{DESC_MAX}"
        )

    haystack = json.dumps(block, ensure_ascii=False) + json.dumps(seo, ensure_ascii=False)
    for word in BANNED:
        if word in haystack:
            problems.append(f"{ns}: в текстах встречается «{word}»")


def check_prerender(ru, ns, slug, problems):
    page = DIST / f"{slug}.html"
    if not page.exists():
        return  # сборки нет — проверяем только копию
    html = page.read_text(encoding="utf-8")
    h1 = ru.get("useCase", {}).get(ns, {}).get("h1", "")
    if h1 and h1 not in html:
        problems.append(f"{slug}.html: не содержит заголовок «{h1}»")
    # hreflang="en" обещает, что есть английская версия. У лендингов, написанных
    # только под русский поиск, её нет, и поисковик уходит на 404.
    dist = DIST
    if not (dist / "en" / f"{slug}.html").exists():
        for copy in (page, dist / "ru" / f"{slug}.html"):
            if copy.exists() and 'hreflang="en"' in copy.read_text(encoding="utf-8"):
                name = copy.relative_to(dist)
                problems.append(f'{name}: hreflang="en" ведёт на несуществующий /en/{slug}')


def frontmatter(text):
    match = re.match(r"---\r?\n(.*?)\r?\n---\r?\n", text, re.S)
    data = {}
    for line in (match.group(1).splitlines() if match else []):
        key, _, value = line.partition(":")
        if value:
            data[key.strip()] = value.strip().strip('"')
    return data


def check_docs(problems):
    """Страницы документации: SEO-поля, запретные слова и пререндер."""
    manifest = json.loads((DOCS / "manifest.json").read_text(encoding="utf-8"))
    for entry in manifest:
        slug = entry["slug"]
        name = f"docs/{slug}" if slug else "docs"
        source = (DOCS / f"{slug or 'index'}.md").read_text(encoding="utf-8")
        meta = frontmatter(source)
        title, desc = meta.get("title", ""), meta.get("description", "")
        if len(title) > TITLE_MAX:
            problems.append(f"{name}: title {len(title)} символов, максимум {TITLE_MAX}")
        if not DESC_MIN <= len(desc) <= DESC_MAX:
            problems.append(f"{name}: description {len(desc)} символов, нужно {DESC_MIN}-{DESC_MAX}")
        for word in BANNED:
            if word in source:
                problems.append(f"{name}: в тексте встречается «{word}»")

        if not DIST.exists():
            continue  # сборки нет — проверяем только исходники
        page = DIST / f"{name}.html"
        if not page.exists():
            problems.append(f"{name}.html: страница не пререндерена")
            continue
        html = page.read_text(encoding="utf-8")
        if "<h1" not in html:
            problems.append(f"{name}.html: нет <h1>")
        if f'rel="canonical" href="https://fxtun.ru/{name}"' not in html:
            problems.append(f"{name}.html: нет canonical на https://fxtun.ru/{name}")
        if "hreflang=" in html:
            problems.append(f"{name}.html: hreflang у страницы, которой нет на английском")
        # Считаем слова только внутри <article> — иначе в счёт попадает инлайновый
        # critical CSS/JS, и проверка никогда не падает.
        article = re.search(r"<article[^>]*>(.*?)</article>", html, re.S)
        if not article:
            problems.append(f"{name}.html: нет <article> с текстом")
        elif len(re.sub(r"<[^>]+>", " ", article.group(1)).split()) < 60:
            problems.append(f"{name}.html: в пререндере почти нет текста")


# Демо-терминалы и команды установки на главной. По запросу «fxtun» Google
# собирал из них сниппет («Tunnel established! https://myapp.fxtun.ru…»),
# потому что в description бренда не было. Такие блоки прячем data-nosnippet.
SNIPPET_JUNK = ("Tunnel established", "install.sh", "myapp.fxtun.ru")
SITE_NAME = "fxTunnel"


class SnippetText(HTMLParser):
    """Видимый текст страницы вне элементов с data-nosnippet."""

    VOID = {"area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "source", "track", "wbr"}

    def __init__(self):
        super().__init__()
        self.stack, self.hidden, self.skip, self.text = [], 0, 0, []

    def handle_starttag(self, tag, attrs):
        if tag in self.VOID:
            return
        hide = any(k == "data-nosnippet" for k, _ in attrs)
        code = tag in ("script", "style", "template")
        self.stack.append((tag, hide, code))
        self.hidden += hide
        self.skip += code

    def handle_endtag(self, tag):
        # Закрывающий тег без открытого (<br/>, лишний </p>) не должен
        # сбрасывать весь стек — иначе текст внутри nosnippet «вылезает».
        if not any(t == tag for t, _, _ in self.stack):
            return
        while self.stack:
            t, hide, code = self.stack.pop()
            self.hidden -= hide
            self.skip -= code
            if t == tag:
                break

    def handle_data(self, data):
        if not self.hidden and not self.skip:
            self.text.append(data)


def check_home_snippet(problems):
    page = DIST / "index.html"
    if not page.exists():
        return
    html = page.read_text(encoding="utf-8")
    site = re.search(r'<meta[^>]+property="?og:site_name"?[^>]*content="([^"]*)"', html)
    if not site or site.group(1) != SITE_NAME:
        problems.append(f"index.html: og:site_name «{site and site.group(1)}», ждём «{SITE_NAME}»")
    desc = re.search(r'<meta[^>]+name="?description"?[^>]*content="([^"]*)"', html)
    if not desc or SITE_NAME not in desc.group(1):
        problems.append(f"index.html: в description нет «{SITE_NAME}» — Google соберёт сниппет из текста страницы")
    parser = SnippetText()
    parser.feed(html)
    visible = " ".join(parser.text)
    for junk in SNIPPET_JUNK:
        if junk in visible:
            problems.append(f"index.html: «{junk}» доступен для сниппета — нужен data-nosnippet")


def main():
    ru = json.loads((WEB / "src" / "i18n" / "ru.json").read_text(encoding="utf-8"))
    problems = []
    for ns, seo_key, slug in LANDINGS:
        check_copy(ru, ns, seo_key, problems)
        check_prerender(ru, ns, slug, problems)
    check_docs(problems)
    check_home_snippet(problems)

    if problems:
        print(f"Проблем: {len(problems)}")
        for p in problems:
            print("  -", p)
        return 1
    print(f"Все {len(LANDINGS)} лендингов и страницы документации на месте")
    return 0


if __name__ == "__main__":
    sys.exit(main())
