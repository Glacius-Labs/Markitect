"""Shared test helpers: run the CLI as a separate process on temporary files."""
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

APP = Path(__file__).resolve().parents[1] / "app.py"


class CliCase(unittest.TestCase):
    def setUp(self):
        folder = tempfile.TemporaryDirectory()
        self.addCleanup(folder.cleanup)
        self.folder = Path(folder.name)
        self.db = self.folder / "entries.json"
        self.log = self.folder / "entries.json.log"

    def run_cli(self, *args, db=None):
        result = subprocess.run([sys.executable, "-B", str(APP), "--db", str(db or self.db), *args],
                                capture_output=True, encoding="utf-8")
        return result.returncode, json.loads(result.stdout)

    def ok(self, *args, db=None):
        code, output = self.run_cli(*args, db=db)
        self.assertEqual(code, 0, output)
        return output

    def state(self, db=None):
        db = db or self.db
        return tuple(path.read_bytes() if path.exists() else None
                     for path in (db, db.with_name(db.name + ".log")))

    def fails(self, code, *args, db=None):
        before = self.state(db)
        status, output = self.run_cli(*args, db=db)
        self.assertEqual((status, output["error"]["code"]), (2, code), output)
        self.assertIsInstance(output["error"]["message"], str)
        self.assertEqual(self.state(db), before)

    def logged(self):
        if not self.log.exists():
            return []
        entries = [json.loads(line) for line in self.log.read_text(encoding="utf-8").splitlines()]
        return [(entry["op"], entry["ids"]) for entry in entries]

    def add(self, entry_id, author="Writer", pages="100", title="Title", db=None):
        return self.ok("add", "--id", entry_id, "--title", title, "--author", author, "--pages", pages, db=db)

    def ids(self, *args, db=None):
        return [entry["id"] for entry in self.ok("list", *args, db=db)["entries"]]
