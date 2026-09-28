import importlib.util
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("cd", HERE.parent / "scripts" / "changelog-draft.py")
cd = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cd)

LINES = [
    "feat(client): add --auto-close flag",
    "fix(client): keep added tunnels on reconnect",
    "fix(client): keep added tunnels on reconnect",
    "perf(core): stream request bodies",
    "feat(payments): recurring renewals",
    "fix(deploy): atomic rename",
    "docs: update readme",
    "chore(release): 3.16.0",
    "feat: top-level feature",
]


class Draft(unittest.TestCase):
    def setUp(self):
        self.out = cd.draft(LINES, "3.16", "2026-09-27")

    def test_front_matter(self):
        self.assertIn('version: "3.16"', self.out)
        self.assertIn("date: 2026-09-27T10:00:00+03:00", self.out)

    def test_groups_in_order(self):
        self.assertLess(self.out.index("## Новое"), self.out.index("## Улучшено"))
        self.assertLess(self.out.index("## Улучшено"), self.out.index("## Исправлено"))

    def test_internal_scopes_dropped(self):
        self.assertNotIn("recurring", self.out)
        self.assertNotIn("atomic rename", self.out)
        self.assertNotIn("readme", self.out)
        self.assertNotIn("3.16.0", self.out.split("---")[-1])

    def test_dedup_and_scope_label(self):
        self.assertEqual(self.out.count("keep added tunnels on reconnect"), 1)
        self.assertIn("- **client:** add --auto-close flag", self.out)
        self.assertIn("- **общее:** top-level feature", self.out)

    def test_empty_group_omitted(self):
        out = cd.draft(["fix(client): x"], "3.1", "2026-01-01")
        self.assertNotIn("## Новое", out)
        self.assertIn("## Исправлено", out)


if __name__ == "__main__":
    unittest.main()
