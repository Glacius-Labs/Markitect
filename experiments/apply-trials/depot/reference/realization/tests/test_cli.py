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
        self.db = Path(self.folder.name) / "depot.json"

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

    def stored(self):
        try:
            return self.db.read_bytes()
        except OSError:  # missing, or a path the system cannot use
            return None

    def refused(self, *args):
        before = self.stored()
        code, result = self.run_app(*args)
        self.assertEqual(code, 2)
        self.assertEqual(list(result), ["error"])
        self.assertEqual(self.stored(), before)

    def test_empty_depot(self):
        self.assertEqual(self.ok("items"), {"items": []})
        self.assertEqual(self.ok("orders"), {"orders": []})
        self.assertEqual(self.ok("reorder"), {"items": []})
        self.assertFalse(self.db.exists())

    def test_order_flow_across_processes(self):
        self.assertEqual(self.ok("add-item", "--sku", "bol-006", "--name", "Bolt", "--reorder", "5"),
                         {"sku": "BOL-006", "name": "Bolt", "reorderPoint": 5})
        self.assertEqual(self.ok("receive", "--sku", "BOL-006", "--qty", "12"), {"sku": "BOL-006", "onHand": 12})
        order = self.ok("place-order", "--line", "BOL-006:8")
        self.assertEqual(order, {"id": 1, "status": "open", "lines": [{"sku": "BOL-006", "qty": 8}]})
        self.assertEqual(self.ok("reorder")["items"][0]["shortfall"], 1)
        self.assertEqual(self.ok("fulfil-order", "--id", "1")["status"], "fulfilled")
        self.assertEqual(self.ok("items")["items"][0],
                         {"sku": "BOL-006", "name": "Bolt", "reorderPoint": 5, "onHand": 4, "reserved": 0, "available": 4})

    def test_refusals_leave_the_file_unchanged(self):
        self.ok("add-item", "--sku", "BOL-006", "--name", "Bolt")
        self.ok("receive", "--sku", "BOL-006", "--qty", "3")
        self.refused("add-item", "--sku", "BOL-006", "--name", "Again")
        self.refused("receive", "--sku", "BOL-006", "--qty", "501")
        self.refused("place-order", "--line", "BOL-006:4")
        self.refused("fulfil-order", "--id", "1")
        self.refused("orders", "--status", "done")

    def test_usage_errors_use_the_error_object(self):
        self.refused("receive", "--sku", "BOL-006")
        self.refused("ship", "--id", "1")
        self.refused("place-order")

    def test_malformed_file_is_reported_and_kept(self):
        self.db.write_text("[]", encoding="utf-8")
        self.refused("items")
        self.refused("add-item", "--sku", "BOL-006", "--name", "Bolt")
        self.assertEqual(self.db.read_text(encoding="utf-8"), "[]")

    def test_extreme_inputs_fail_without_a_crash(self):
        self.ok("add-item", "--sku", "BOL-006", "--name", "Bolt")
        self.refused("receive", "--sku", "BOL-006", "--qty", "9" * 5000)
        self.refused("fulfil-order", "--id", "9" * 5000)
        self.refused("add-item", "--sku", "NUT-010", "--name", "Nut \udcff")
        self.db.write_text("[" * 100000, encoding="utf-8")
        self.refused("items")
        self.db = Path(self.folder.name) / ("x" * 5000 + ".json")
        code, result = self.run_app("items")  # an unusable path may read as missing data or fail cleanly
        self.assertIn((code, list(result)), [(0, ["items"]), (2, ["error"])])
        self.refused("add-item", "--sku", "BOL-006", "--name", "Bolt")


if __name__ == "__main__":
    unittest.main()
