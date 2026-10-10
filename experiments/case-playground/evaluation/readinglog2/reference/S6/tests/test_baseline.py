import json
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import unittest


APP = Path(__file__).resolve().parents[1] / "app.py"


class Readinglog(unittest.TestCase):
    def setUp(self):
        folder = tempfile.TemporaryDirectory()
        self.addCleanup(folder.cleanup)
        self.db = Path(folder.name) / "entries.json"
        self.log = Path(folder.name) / "entries.json.log"

    def call(self, *args):
        result = subprocess.run([sys.executable, "-B", str(APP), "--db", str(self.db), *args],
                                capture_output=True, encoding="utf-8")
        return result.returncode, json.loads(result.stdout)

    def log_lines(self):
        if not self.log.exists():
            return []
        return [json.loads(line) for line in self.log.read_text(encoding="utf-8").splitlines()]

    def assert_error(self, args, code):
        before = (self.db.read_bytes() if self.db.exists() else None,
                  self.log.read_bytes() if self.log.exists() else None)
        status, output = self.call(*args)
        self.assertEqual(status, 2)
        self.assertEqual(output["error"]["code"], code)
        self.assertIsInstance(output["error"]["message"], str)
        self.assertEqual((self.db.read_bytes() if self.db.exists() else None,
                          self.log.read_bytes() if self.log.exists() else None), before)

    def test_add_list_and_restart(self):
        self.assertEqual(self.call("list"), (0, {"entries": []}))
        self.assertFalse(self.db.exists())
        code, entry = self.call("add", "--id", " a ", "--title", " An \t Example ", "--author", "Writer",
                               "--pages", " 120 ")
        self.assertEqual(code, 0)
        self.assertEqual(entry, {"id": "a", "title": "An Example", "author": "Writer", "pages": 120,
                                "status": "unread", "tags": []})
        self.call("add", "--id", "A", "--title", "Second", "--author", "Writer", "--pages", "1")
        self.assertEqual([b["id"] for b in self.call("list")[1]["entries"]], ["a", "A"])

    def test_audit_log_has_one_line_per_successful_change(self):
        self.call("add", "--id", "a", "--title", "T", "--author", "W", "--pages", "1")
        self.call("list")
        self.call("add", "--id", "a", "--title", "T", "--author", "W", "--pages", "1")
        lines = self.log_lines()
        self.assertEqual([(line["op"], line["ids"]) for line in lines], [("add", ["a"])])
        self.assertRegex(lines[0]["at"], re.compile(r"^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ$"))

    def test_errors_use_the_contract_and_change_nothing(self):
        self.assert_error(["list", "--bogus"], "invalid_input")
        self.assertFalse(self.db.exists())
        self.call("add", "--id", "a", "--title", "T", "--author", "W", "--pages", "1")
        self.assert_error(["add", "--id", "a", "--title", "Again", "--author", "W", "--pages", "1"], "duplicate")
        self.assert_error(["add", "--id", "\u00e9", "--title", " ", "--author", "W", "--pages", "1"], "invalid_input")
        for pages in ("0", "-1", "1.5", "x", "\uff11"):
            self.assert_error(["add", "--id", "b", "--title", "T", "--author", "W", "--pages", pages],
                              "invalid_input")
        self.assert_error(["add", "--id", "b", "--title", "T", "--pages", "1"], "invalid_input")
        self.assert_error(["remove"], "invalid_input")

    def test_ids_are_nfc_normalized_and_case_sensitive(self):
        self.call("add", "--id", "e\u0301", "--title", "T", "--author", "W", "--pages", "1")
        self.assert_error(["add", "--id", "\u00e9", "--title", "T", "--author", "W", "--pages", "1"], "duplicate")
        self.assertEqual(self.call("add", "--id", "\u00c9", "--title", "T", "--author", "W", "--pages", "1")[0], 0)

    def test_malformed_database_is_never_overwritten(self):
        for content in ("{broken", '{"entries": []}', '[{"id": "a"}]'):
            self.db.write_text(content, encoding="utf-8")
            self.assert_error(["list"], "storage")
            self.assert_error(["add", "--id", "b", "--title", "T", "--author", "W", "--pages", "1"], "storage")
            self.assertEqual(self.db.read_text(encoding="utf-8"), content)


if __name__ == "__main__":
    unittest.main()
