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
        self.db = Path(self.folder.name) / "workshop.json"

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

    def test_a_job_from_opening_to_payment(self):
        self.assertEqual(self.ok("add-customer", "--name", " Ada ")["id"], "C1")
        self.ok("add-part", "--sku", "brk-101", "--name", "Brake pad", "--price", "1250")
        self.ok("receive", "--sku", "BRK-101", "--qty", "4")
        self.ok("open-job", "--customer", "C1", "--bike", "-Gravel bike")
        self.ok("add-labor", "--job", "J1", "--minutes", "45")
        self.ok("use-part", "--job", "J1", "--sku", "BRK-101", "--qty", "2")
        self.assertEqual(self.ok("close-job", "--job", "J1")["status"], "closed")
        invoice = self.ok("invoice", "--job", "J1")
        self.assertEqual((invoice["number"], invoice["labor"], invoice["parts"], invoice["net"], invoice["vat"]),
                         ("INV-0001", 4500, 2500, 7000, 1330))
        self.assertTrue(self.ok("pay", "--invoice", "INV-0001")["paid"])
        self.assertEqual(self.ok("report"), {"openJobs": 0, "unpaidTotal": 0, "paidTotal": 8330, "lowStock": ["BRK-101"]})
        self.refused("pay", "--invoice", "INV-0001")

    def test_usage_errors_use_the_error_object(self):
        for args in ((), ("unknown",), ("add-customer",), ("parts", "--help"), ("jobs", "--status")):
            with self.subTest(args=args):
                self.refused(*args)


if __name__ == "__main__":
    unittest.main()
