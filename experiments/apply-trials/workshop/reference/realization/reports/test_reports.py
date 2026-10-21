import unittest

from customers import customers
from jobs import jobs
from parts import parts
from reports import reports
from shop.store import empty


class ReportTests(unittest.TestCase):
    def test_empty_and_low_stock(self):
        state = empty()
        self.assertEqual(reports.report(state), {"openJobs": 0, "unpaidTotal": 0, "paidTotal": 0, "lowStock": []})
        customers.add_customer(state, "Ada")
        parts.add_part(state, "BRK-101", "Brake pad", "100")
        parts.add_part(state, "AAA-001", "Bell", "100")
        parts.receive(state, "BRK-101", "4")
        jobs.open_job(state, "C1", "Bike")
        jobs.use_part(state, "J1", "BRK-101", "2")
        self.assertEqual(reports.report(state), {"openJobs": 1, "unpaidTotal": 0, "paidTotal": 0,
                                                 "lowStock": ["AAA-001", "BRK-101"]})


if __name__ == "__main__":
    unittest.main()
