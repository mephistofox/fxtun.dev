#!/usr/bin/env python3
"""Сверка адресов блога с sitemap.

    urls.py check <public> <sitemap.xml>...   все <loc> отдаются из public/, иначе код 1
    urls.py new <old.xml> <new.xml>           пути, которых не было в старом sitemap

Файл ищется так же, как nginx: try_files $uri $uri/ $uri.html
(location /blog/ — alias на корень сборки).
"""
import sys
import xml.etree.ElementTree as ET
from pathlib import Path

PREFIX = "https://fxtun.ru/blog/"
NS = "{http://www.sitemaps.org/schemas/sitemap/0.9}"


def locs(path):
    return [el.text.strip() for el in ET.parse(path).getroot().iter(NS + "loc")]


def served(public, loc):
    if not loc.startswith(PREFIX):
        return False
    rel = loc[len(PREFIX):]
    base = public / rel
    candidates = [base / "index.html", Path(str(base).rstrip("/") + ".html")]
    if rel and not rel.endswith("/"):
        candidates.insert(0, base)
    return any(c.is_file() for c in candidates)


def new_paths(old, new):
    before = set(locs(old))
    return [l[len("https://fxtun.ru"):] for l in locs(new) if l not in before]


def main(argv):
    cmd, *args = argv
    if cmd == "check":
        public = Path(args[0])
        missing = [l for s in args[1:] for l in locs(s) if not served(public, l)]
        for l in missing:
            print("нет в сборке:", l)
        print(f"пропавших адресов: {len(missing)}")
        return 1 if missing else 0
    if cmd == "new":
        print("\n".join(new_paths(*args)))
        return 0
    raise SystemExit(__doc__)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
