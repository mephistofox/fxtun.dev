#!/usr/bin/env python3
"""Черновик записи «Что нового» из git log.

    changelog-draft.py <от-тега> <до-тега>

Печатает front matter и коммиты feat/perf/fix, сгруппированные по типу и
scope. Внутренние области (деплой, платежи, CI…) отброшены. Текст записи
пишется по черновику вручную, по-русски и для пользователя.
"""
import re
import subprocess
import sys

INTERNAL = {"deploy", "ci", "payments", "payment", "billing", "db", "admin", "seo", "blog", "docs", "scheduler", "web-admin"}
GROUPS = (("feat", "Новое"), ("perf", "Улучшено"), ("fix", "Исправлено"))
SUBJECT = re.compile(r"^(feat|fix|perf)(?:\(([^)]*)\))?!?:\s*(.+)$")


def draft(lines, version, date):
    seen = set()
    buckets = {t: {} for t, _ in GROUPS}
    for line in lines:
        m = SUBJECT.match(line.strip())
        if not m:
            continue
        typ, scope, text = m.group(1), (m.group(2) or "").strip(), m.group(3).strip()
        if scope in INTERNAL or text.lower() in seen:
            continue
        seen.add(text.lower())
        buckets[typ].setdefault(scope or "общее", []).append(text)
    out = ["---", 'title: ""', f"date: {date}T10:00:00+03:00", f'version: "{version}"',
           'description: ""', "kinds: []", "components: []", "---", ""]
    for typ, name in GROUPS:
        if not buckets[typ]:
            continue
        out += [f"## {name}", ""]
        for scope in sorted(buckets[typ]):
            out += [f"- **{scope}:** {t}" for t in buckets[typ][scope]]
        out.append("")
    return "\n".join(out)


def git(*args):
    return subprocess.run(["git", *args], capture_output=True, text=True, check=True).stdout


def main(argv):
    start, end = argv
    lines = git("log", "--no-merges", "--format=%s", f"{start}..{end}").splitlines()
    date = git("log", "-1", "--format=%cs", end).strip()
    version = re.sub(r"^v(\d+\.\d+).*$", r"\1", end)
    print(draft(lines, version, date))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
