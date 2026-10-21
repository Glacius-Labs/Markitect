import unittest

from customers import customers
from jobs import jobs
from parts import parts
from shop.errors import Refusal
from shop.store import empty


class JobTests(unittest.TestCase):
    def setUp(self):
        self.state = empty()
        customers.add_customer(self.state, "Ada")
        parts.add_part(self.state, "CHN-010", "Chain", "2500")
        parts.receive(self.state, "CHN-010", "3")

    def test_parts_are_reserved_then_consumed(self):
        jobs.open_job(self.state, "C1", "Bike")
        jobs.use_part(self.state, "J1", "chn-010", "1")
        self.assertEqual(jobs.use_part(self.state, "J1", "CHN-010", "1")["parts"], [{"sku": "CHN-010", "qty": 2}])
        self.assertEqual(parts.record(self.state, "CHN-010")["available"], 1)
        with self.assertRaises(Refusal):
            jobs.use_part(self.state, "J1", "CHN-010", "2")
        jobs.close_job(self.state, "J1")
        self.assertEqual(parts.record(self.state, "CHN-010")["onHand"], 1)
        self.assertEqual(parts.record(self.state, "CHN-010")["reserved"], 0)

    def test_labor_limits_and_closed_jobs(self):
        jobs.open_job(self.state, "C1", "Bike")
        for _ in range(5):
            jobs.add_labor(self.state, "J1", "480")
        with self.assertRaises(Refusal):
            jobs.add_labor(self.state, "J1", "15")
        jobs.open_job(self.state, "C1", "Other")
        for minutes in ("10", "0", "495"):
            with self.subTest(minutes=minutes), self.assertRaises(Refusal):
                jobs.add_labor(self.state, "J2", minutes)
        jobs.close_job(self.state, "J1")
        with self.assertRaises(Refusal):
            jobs.close_job(self.state, "J1")

    def test_unknown_customer(self):
        with self.assertRaises(Refusal):
            jobs.open_job(self.state, "C9", "Bike")


if __name__ == "__main__":
    unittest.main()
