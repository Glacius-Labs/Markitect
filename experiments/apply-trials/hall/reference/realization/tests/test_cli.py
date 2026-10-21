import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

REALIZATION = Path(__file__).resolve().parents[1]


class CommandLineTests(unittest.TestCase):
    def setUp(self):
        self.folder = tempfile.TemporaryDirectory()
        self.db = Path(self.folder.name) / "hall.json"

    def tearDown(self):
        self.folder.cleanup()

    def run_app(self, *args):
        done = subprocess.run([sys.executable, "-B", "app.py", "--db", str(self.db), *args], cwd=REALIZATION,
                              capture_output=True, text=True, encoding="utf-8")
        return done.returncode, json.loads(done.stdout)

    def ok(self, *args):
        code, result = self.run_app(*args)
        self.assertEqual(code, 0, result)
        return result

    def refused(self, *args):
        before = self.db.read_bytes() if self.db.exists() else None
        code, result = self.run_app(*args)
        self.assertEqual(code, 2, result)
        self.assertEqual(set(result), {"error"})
        self.assertEqual(self.db.read_bytes() if self.db.exists() else None, before)
        return result

    def test_rooms_and_bookings_end_to_end(self):
        self.assertEqual(self.ok("rooms"), {"rooms": []})
        room = self.ok("add-room", "--code", " a101 ", "--seats", "10")
        self.assertEqual((room["code"], room["seats"]), ("A101", 10))
        self.assertEqual(self.ok("rooms"), {"rooms": [room]})
        self.assertEqual(self.ok("book", "--code", "a101", "--seats", str(room["bookable"])),
                         {"id": 1, "code": "A101", "seats": room["bookable"]})
        self.refused("book", "--code", "A101", "--seats", "1")
        self.assertEqual(self.ok("bookings", "--code", "A101")["bookings"][0]["id"], 1)

    def test_usage_errors_use_the_error_object(self):
        for args in ((), ("unknown",), ("add-room", "--code", "A101"), ("rooms", "--help"), ("bookings", "--code")):
            with self.subTest(args=args):
                self.refused(*args)

    def test_malformed_data_is_kept(self):
        self.db.write_text("[]", encoding="utf-8")
        self.refused("add-room", "--code", "A101", "--seats", "10")
        self.refused("rooms")
        self.assertEqual(self.db.read_text(encoding="utf-8"), "[]")


if __name__ == "__main__":
    unittest.main()
