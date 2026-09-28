#!/usr/bin/env python3
"""Проверка front matter статей блога.

    check-content.py <content_dir> [--strict] [--html <public_dir>]

Статья (не _index.md, не draft) должна иметь description 120–160 символов и
ровно одну рубрику из data/rubrics.toml. Без --strict нарушения печатаются как
предупреждения и код возврата 0: рубрики раздаются в части 4 переделки блога.
Раздел diary и changelog не проверяются на рубрику.

С --html <public_dir> дополнительно сканируются собранные *.html: псевдографика
внутри <pre> (остатки текстовых диаграмм) — всегда ошибка, код возврата 1 даже
без --strict.

Поле need (ключ из data/needs.toml) обязательно для статей с датой от 2026-09-28.

Записи раздела changelog дополнительно проверяются на непустой version и на то,
что kinds и components берутся только из допустимых значений.
"""
import re
import sys
import tomllib
from pathlib import Path

SKIP_RUBRIC = ("diary", "changelog")
BOX_DRAWING = set("┌└│├┤┬┴┼─►▶◄▼▲")
KINDS = {"new", "improved", "fixed"}
COMPONENTS = {"client", "server", "gui", "web"}


def front(path):
    text = path.read_text(encoding="utf-8")
    m = re.match(r"^---\n(.*?)\n---\n", text, re.S)
    return m.group(1) if m else ""


def field(fm, name):
    m = re.search(rf"^{name}:\s*(.+)$", fm, re.M)
    return m.group(1).strip() if m else None


NEED_FROM = "2026-09-28"  # статьи с этой даты обязаны указывать need


def problems(path, rubrics, needs=None):
    if path.name == "_index.md":
        return []
    fm = front(path)
    if field(fm, "draft") == "true":
        return []
    out = []
    desc = (field(fm, "description") or "").strip('"')
    if not 120 <= len(desc) <= 160:
        out.append(f"description {len(desc)} символов (нужно 120–160)")
    if path.parent.name not in SKIP_RUBRIC:
        raw = field(fm, "rubrics")
        items = [s.strip().strip("\"'") for s in raw.strip("[]").split(",")] if raw else []
        items = [s for s in items if s]
        if not items:
            out.append("нет рубрики")
        elif len(items) > 1:
            out.append("рубрика должна быть одна")
        elif items[0] not in rubrics:
            out.append(f"неизвестная рубрика {items[0]}")
    if needs is not None and path.parent.name not in SKIP_RUBRIC:
        need = (field(fm, "need") or "").strip("\"'")
        date = (field(fm, "date") or "").strip("\"'")
        if need and need not in needs:
            out.append(f"неизвестный need {need}")
        elif not need and date[:10] >= NEED_FROM:
            out.append("нет need")
    if path.parent.name == "changelog":
        if not (field(fm, "version") or "").strip("\"'"):
            out.append("нет version")
        for name, allowed in (("kinds", KINDS), ("components", COMPONENTS)):
            raw = field(fm, name) or ""
            for item in [s.strip().strip("\"'") for s in raw.strip("[]").split(",") if s.strip()]:
                if item not in allowed:
                    out.append(f"неизвестный {name}: {item}")
    return out


def html_problems(public_dir):
    """Built html files whose <pre> contains box-drawing / arrow characters."""
    out = []
    for path in sorted(Path(public_dir).rglob("*.html")):
        text = path.read_text(encoding="utf-8")
        for pre in re.findall(r"<pre[^>]*>(.*?)</pre>", text, re.S):
            if BOX_DRAWING & set(pre):
                out.append(path)
                break
    return out


def main(argv):
    content = Path(argv[0])
    strict = "--strict" in argv
    html_dir = Path(argv[argv.index("--html") + 1]) if "--html" in argv else None
    rubrics = set(tomllib.loads((content.parent / "data" / "rubrics.toml").read_text(encoding="utf-8")))
    needs = set(tomllib.loads((content.parent / "data" / "needs.toml").read_text(encoding="utf-8")))
    total = 0
    for path in sorted(content.rglob("*.md")):
        for p in problems(path, rubrics, needs):
            total += 1
            print(f"{path.relative_to(content)}: {p}")
    print(f"замечаний: {total}")
    bad_html = html_problems(html_dir) if html_dir else []
    for path in bad_html:
        print(f"ERROR: {path.relative_to(html_dir)}: псевдографика внутри <pre>")
    return 1 if (strict and total) or bad_html else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
