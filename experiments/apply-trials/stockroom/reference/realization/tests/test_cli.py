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
        self.db = Path(self.folder.name) / "stock.json"

    def tearDown(self):
        self.folder.cleanup()

    def run_app(self, *args):
        done = subprocess.run([sys.executable, "-B", "app.py", "--db", str(self.db), *args],
                              cwd=REALIZATION, capture_output=True, text=True, encoding="utf-8")
        return done.returncode, json.loads(done.stdout)

    def ok(self, *args):
        code, result = self.run_app(*args)
        self.assertEqual(code, 0, result)
        return result

    def refused(self, *args):
        before = self.db.read_bytes() if self.db.exists() else None
        code, result = self.run_app(*args)
        self.assertEqual(code, 2)
        self.assertEqual(list(result), ["error"])
        self.assertTrue(result["error"].strip())
        self.assertEqual(self.db.read_bytes() if self.db.exists() else None, before)

    def test_missing_file_lists_nothing(self):
        self.assertEqual(self.ok("list"), {"items": []})
        self.assertFalse(self.db.exists())

    def test_round_trip_across_processes(self):
        self.assertEqual(self.ok("add-item", "--sku", " bol-006 ", "--name", " Bolt M6 "),
                         {"sku": "BOL-006", "name": "Bolt M6", "onHand": 0})
        self.assertEqual(self.ok("receive", "--sku", "BOL-006", "--qty", "1000")["onHand"], 1000)
        self.assertEqual(self.ok("issue", "--sku", "BOL-006", "--qty", "250")["onHand"], 750)
        self.assertEqual(self.ok("list"), {"items": [{"sku": "BOL-006", "name": "Bolt M6", "onHand": 750}]})

    def test_list_order(self):
        self.ok("add-item", "--sku", "NUT-010", "--name", "Nut")
        self.ok("add-item", "--sku", "BOL-006", "--name", "Bolt")
        self.assertEqual([item["sku"] for item in self.ok("list")["items"]], ["BOL-006", "NUT-010"])

    def test_refusals_leave_the_file_unchanged(self):
        self.ok("add-item", "--sku", "BOL-006", "--name", "Bolt")
        self.ok("receive", "--sku", "BOL-006", "--qty", "5")
        self.refused("add-item", "--sku", "bol-006", "--name", "Again")
        self.refused("receive", "--sku", "BOL-006", "--qty", "0")
        self.refused("receive", "--sku", "BOL-006", "--qty", "1001")
        self.refused("issue", "--sku", "BOL-006", "--qty", "6")
        self.refused("issue", "--sku", "NUT-010", "--qty", "1")
        self.refused("add-item", "--sku", "BOLT-6", "--name", "Bolt")

    def test_usage_errors_use_the_error_object(self):
        self.refused("receive", "--sku", "BOL-006")
        self.refused("move", "--sku", "BOL-006")

    def test_malformed_file_is_reported_and_kept(self):
        self.db.write_text("{not json", encoding="utf-8")
        self.refused("list")
        self.refused("add-item", "--sku", "BOL-006", "--name", "Bolt")
        self.assertEqual(self.db.read_text(encoding="utf-8"), "{not json")


if __name__ == "__main__":
    unittest.main()
