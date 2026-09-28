import importlib.util
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("urls", HERE.parent / "scripts" / "urls.py")
urls = importlib.util.module_from_spec(spec)
spec.loader.exec_module(urls)

SITEMAP = """<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
{}
</urlset>"""


def sitemap(tmp, name, *locs):
    path = Path(tmp) / name
    path.write_text(SITEMAP.format("".join(f"<url><loc>{l}</loc></url>" for l in locs)))
    return path


class Served(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.public = Path(self.tmp.name)
        (self.public / "post").mkdir()
        (self.public / "post" / "index.html").write_text("x")
        (self.public / "diary").mkdir()
        (self.public / "diary" / "21-i18n-ddos.html").write_text("x")
        (self.public / "index.html").write_text("x")

    def tearDown(self):
        self.tmp.cleanup()

    def test_directory_page(self):
        self.assertTrue(urls.served(self.public, "https://fxtun.ru/blog/post/"))

    def test_blog_root(self):
        self.assertTrue(urls.served(self.public, "https://fxtun.ru/blog/"))

    def test_flat_diary_page_without_extension(self):
        # nginx: try_files $uri $uri/ $uri.html
        self.assertTrue(urls.served(self.public, "https://fxtun.ru/blog/diary/21-i18n-ddos"))

    def test_missing_page(self):
        self.assertFalse(urls.served(self.public, "https://fxtun.ru/blog/gone/"))

    def test_foreign_host_is_not_ours(self):
        self.assertFalse(urls.served(self.public, "https://fxtun.dev/blog/post/"))


class Locs(unittest.TestCase):
    def test_reads_namespaced_sitemap(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = sitemap(tmp, "s.xml", "https://fxtun.ru/blog/a/", "https://fxtun.ru/blog/b/")
            self.assertEqual(urls.locs(path), ["https://fxtun.ru/blog/a/", "https://fxtun.ru/blog/b/"])


class NewPaths(unittest.TestCase):
    def test_only_addresses_absent_before(self):
        with tempfile.TemporaryDirectory() as tmp:
            old = sitemap(tmp, "old.xml", "https://fxtun.ru/blog/a/")
            new = sitemap(tmp, "new.xml", "https://fxtun.ru/blog/a/", "https://fxtun.ru/blog/b/")
            self.assertEqual(urls.new_paths(old, new), ["/blog/b/"])


if __name__ == "__main__":
    unittest.main()
