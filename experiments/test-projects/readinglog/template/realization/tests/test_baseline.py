import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


APP = Path(__file__).resolve().parents[1] / "app.py"


class ExistingContract(unittest.TestCase):
    def test_add_list_restart_duplicate_and_invalid_input(self):
        with tempfile.TemporaryDirectory() as folder:
            db = Path(folder) / "books.json"
            def call(*args):
                result = subprocess.run([sys.executable, "-B", str(APP), "--db", str(db), *args], capture_output=True, text=True)
                return result.returncode, json.loads(result.stdout)
            self.assertEqual(call("list"), (0, {"books": []}))
            code, book = call("add", "--id", " a ", "--title", " Example ", "--author", " Writer ", "--pages", "120")
            self.assertEqual(code, 0)
            self.assertEqual(book, {"id": "a", "title": "Example", "author": "Writer", "pages": 120, "status": "unread"})
            self.assertEqual(call("list"), (0, {"books": [book]}))
            before = db.read_bytes()
            self.assertEqual(call("add", "--id", "a", "--title", "Again", "--author", "Writer", "--pages", "1")[0], 2)
            self.assertEqual(call("add", "--id", "b", "--title", "Bad", "--author", "Writer", "--pages", "0")[0], 2)
            self.assertEqual(db.read_bytes(), before)


if __name__ == "__main__":
    unittest.main()
