#!/usr/bin/env python3
"""Проверяет, что у каждого поискового лендинга есть полная копия.

Запуск из корня репозитория:
    python3 web/scripts/check-landings.py

Код возврата 1, если у лендинга нет текстов, тексты неполные, SEO-поля вышли
за длину или собранная страница не содержит своего заголовка.
"""
import json
import pathlib
import sys

WEB = pathlib.Path(__file__).resolve().parents[1]

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

BANNED = ("ИИ", "нейросет", "LLM", "автогенерац", "ИИ-ассистент")

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
                for field in ITEM_KEYS.get(name, ()):
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
    page = WEB / "dist" / f"{slug}.html"
    if not page.exists():
        return  # сборки нет — проверяем только копию
    h1 = ru.get("useCase", {}).get(ns, {}).get("h1", "")
    if h1 and h1 not in page.read_text(encoding="utf-8"):
        problems.append(f"{slug}.html: не содержит заголовок «{h1}»")


def main():
    ru = json.loads((WEB / "src" / "i18n" / "ru.json").read_text(encoding="utf-8"))
    problems = []
    for ns, seo_key, slug in LANDINGS:
        check_copy(ru, ns, seo_key, problems)
        check_prerender(ru, ns, slug, problems)

    if problems:
        print(f"Проблем: {len(problems)}")
        for p in problems:
            print("  -", p)
        return 1
    print(f"Все {len(LANDINGS)} лендингов на месте")
    return 0


if __name__ == "__main__":
    sys.exit(main())
