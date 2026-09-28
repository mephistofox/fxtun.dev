import importlib.util
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("cc", HERE.parent / "scripts" / "check-content.py")
cc = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cc)

RUBRICS = {"guides", "protocols"}
DESC = "Д" * 130


def post(tmp, name, front):
    path = Path(tmp) / name
    path.write_text(f"---\n{front}\n---\nТекст.\n", encoding="utf-8")
    return path


class Problems(unittest.TestCase):
    def check(self, front):
        with tempfile.TemporaryDirectory() as tmp:
            return cc.problems(post(tmp, "a.md", front), RUBRICS)

    def test_clean_post(self):
        self.assertEqual(self.check(f'title: "A"\ndescription: "{DESC}"\nrubrics: [guides]'), [])

    def test_missing_rubric(self):
        self.assertIn("нет рубрики", " ".join(self.check(f'title: "A"\ndescription: "{DESC}"')))

    def test_unknown_rubric(self):
        self.assertIn("неизвестная рубрика news", " ".join(self.check(f'title: "A"\ndescription: "{DESC}"\nrubrics: [news]')))

    def test_two_rubrics(self):
        self.assertIn("рубрика должна быть одна", " ".join(self.check(f'title: "A"\ndescription: "{DESC}"\nrubrics: [guides, protocols]')))

    def test_description_length(self):
        self.assertIn("description 20 символов", " ".join(self.check('title: "A"\ndescription: "' + "д" * 20 + '"\nrubrics: [guides]')))

    def test_section_pages_are_skipped(self):
        with tempfile.TemporaryDirectory() as tmp:
            self.assertEqual(cc.problems(post(tmp, "_index.md", 'title: "Раздел"'), RUBRICS), [])

    def test_drafts_are_skipped(self):
        self.assertEqual(self.check('title: "A"\ndraft: true'), [])

    NEEDS = {"free", "udp"}

    def check_need(self, front):
        with tempfile.TemporaryDirectory() as tmp:
            return cc.problems(post(tmp, "a.md", front), RUBRICS, self.NEEDS)

    def test_unknown_need(self):
        self.assertIn("неизвестный need gpu", " ".join(self.check_need(f'title: "A"\ndate: 2026-01-01\ndescription: "{DESC}"\nrubrics: [guides]\nneed: gpu')))

    def test_new_post_requires_need(self):
        self.assertIn("нет need", " ".join(self.check_need(f'title: "A"\ndate: 2026-09-29T10:00:00+03:00\ndescription: "{DESC}"\nrubrics: [guides]')))

    def test_old_post_may_skip_need(self):
        self.assertEqual(self.check_need(f'title: "A"\ndate: 2026-02-01\ndescription: "{DESC}"\nrubrics: [guides]'), [])

    def test_known_need(self):
        self.assertEqual(self.check_need(f'title: "A"\ndate: 2026-09-29\ndescription: "{DESC}"\nrubrics: [guides]\nneed: udp'), [])


class Changelog(unittest.TestCase):
    def check(self, front):
        with tempfile.TemporaryDirectory() as tmp:
            d = Path(tmp) / "changelog"
            d.mkdir()
            return cc.problems(post(d, "v3-16.md", front), RUBRICS, {"free"})

    def test_valid_entry(self):
        self.assertEqual(self.check(f'title: "A"\ndate: 2026-09-27\ndescription: "{DESC}"\nversion: "3.16"\nkinds: [new, fixed]\ncomponents: [client]'), [])

    def test_unknown_kind(self):
        self.assertIn("неизвестный kinds: added", " ".join(self.check(f'title: "A"\ndate: 2026-09-27\ndescription: "{DESC}"\nversion: "3.16"\nkinds: [added]\ncomponents: [client]')))

    def test_unknown_component(self):
        self.assertIn("неизвестный components: api", " ".join(self.check(f'title: "A"\ndate: 2026-09-27\ndescription: "{DESC}"\nversion: "3.16"\nkinds: [new]\ncomponents: [api]')))

    def test_missing_version(self):
        self.assertIn("нет version", " ".join(self.check(f'title: "A"\ndate: 2026-09-27\ndescription: "{DESC}"\nkinds: [new]\ncomponents: [client]')))


class HtmlProblems(unittest.TestCase):
    def test_box_drawing_in_pre_only(self):
        with tempfile.TemporaryDirectory() as tmp:
            good = Path(tmp) / "good.html"
            good.write_text(
                "<body><p>стрелка вне pre ► не считается</p><pre>plain text</pre></body>",
                encoding="utf-8",
            )
            bad = Path(tmp) / "bad.html"
            bad.write_text(
                "<body><pre>┌──┐\n│  │\n└──┘</pre></body>",
                encoding="utf-8",
            )
            problems = cc.html_problems(Path(tmp))
            self.assertEqual([p.name for p in problems], ["bad.html"])


if __name__ == "__main__":
    unittest.main()
