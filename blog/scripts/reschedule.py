#!/usr/bin/env python3
"""Сдвиг дат ещё не вышедших статей плана.

    reschedule.py <content_dir> <YYYY-MM-DD> [--today YYYY-MM-DD]

Статьи с полем need и датой позже today получают даты first, first+2, …
в прежнем порядке, в 10:00 МСК. Меняется только строка date: в front matter.
"""
import datetime as dt
import re
import sys
from pathlib import Path

DATE_RE = re.compile(r"^date:\s*(\S+)\s*$", re.M)


def front(text):
    m = re.match(r"^---\n(.*?)\n---\n", text, re.S)
    return m.group(1) if m else ""


def plan(posts, first, today):
    todo = []
    for path in posts:
        fm = front(path.read_text(encoding="utf-8"))
        m = DATE_RE.search(fm)
        if m and re.search(r"^need:", fm, re.M) and m.group(1)[:10] > today:
            todo.append((m.group(1), path))
    todo.sort()
    start = dt.date.fromisoformat(first)
    return [(p, f"{start + dt.timedelta(days=2 * i)}T10:00:00+03:00") for i, (_, p) in enumerate(todo)]


def main(argv):
    content, first = Path(argv[0]), argv[1]
    today = argv[argv.index("--today") + 1] if "--today" in argv else dt.date.today().isoformat()
    for path, new in plan(sorted(content.glob("*.md")), first, today):
        text = path.read_text(encoding="utf-8")
        head, sep, body = text.partition("\n---\n")
        path.write_text(DATE_RE.sub(f"date: {new}", head, count=1) + sep + body, encoding="utf-8")
        print(f"{path.name}: {new}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
