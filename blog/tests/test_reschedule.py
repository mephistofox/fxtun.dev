import importlib.util
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("rs", HERE.parent / "scripts" / "reschedule.py")
rs = importlib.util.module_from_spec(spec)
spec.loader.exec_module(rs)


def post(tmp, name, date, need=True):
    p = Path(tmp) / name
    p.write_text(f"---\ntitle: \"{name}\"\ndate: {date}\n" + ("need: free\n" if need else "") + "---\nтекст date: не трогать\n", encoding="utf-8")
    return p


class Reschedule(unittest.TestCase):
    def test_shifts_future_posts_keeping_order_and_step(self):
        with tempfile.TemporaryDirectory() as tmp:
            b = post(tmp, "b.md", "2026-10-01T10:00:00+03:00")
            a = post(tmp, "a.md", "2026-09-29T10:00:00+03:00")
            old = post(tmp, "old.md", "2026-02-01T10:00:00+03:00")
            plain = post(tmp, "plain.md", "2026-10-05T10:00:00+03:00", need=False)
            rs.main([tmp, "2026-10-20", "--today", "2026-09-28"])
            self.assertIn("date: 2026-10-20T10:00:00+03:00\n", a.read_text(encoding="utf-8"))
            self.assertIn("date: 2026-10-22T10:00:00+03:00\n", b.read_text(encoding="utf-8"))
            self.assertIn("date: 2026-02-01T10:00:00+03:00\n", old.read_text(encoding="utf-8"))
            self.assertIn("date: 2026-10-05T10:00:00+03:00\n", plain.read_text(encoding="utf-8"))
            self.assertIn("текст date: не трогать", a.read_text(encoding="utf-8"))

    def test_published_posts_stay(self):
        with tempfile.TemporaryDirectory() as tmp:
            done = post(tmp, "done.md", "2026-09-29T10:00:00+03:00")
            rs.main([tmp, "2026-10-20", "--today", "2026-09-30"])
            self.assertIn("date: 2026-09-29T10:00:00+03:00\n", done.read_text(encoding="utf-8"))


if __name__ == "__main__":
    unittest.main()
