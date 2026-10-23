import unittest

from billing import billing
from customers import customers
from jobs import jobs
from parts import parts
from shop.errors import Refusal
from shop.store import empty


class BillingTests(unittest.TestCase):
    def setUp(self):
        self.state = empty()
        customers.add_customer(self.state, "Ada")
        parts.add_part(self.state, "TUB-001", "Tube", "1010")
        parts.receive(self.state, "TUB-001", "100")

    def closed_job(self, minutes="60", tubes="1"):
        job = jobs.open_job(self.state, "C1", "City bike")["id"]
        jobs.add_labor(self.state, job, minutes)
        jobs.use_part(self.state, job, "TUB-001", tubes)
        jobs.close_job(self.state, job)
        return job

    def test_standard_invoice(self):
        invoice = billing.write_invoice(self.state, self.closed_job())
        self.assertEqual({key: invoice[key] for key in billing.AMOUNTS},
                         {"labor": 6000, "parts": 1010, "discount": 0, "net": 7010, "vat": 1332, "total": 8342})
        self.assertEqual((invoice["number"], invoice["tier"], invoice["paid"]), ("INV-0001", "standard", False))

    def test_loyal_discount_rounds_half_to_even(self):
        for _ in range(3):
            billing.pay(self.state, billing.write_invoice(self.state, self.closed_job())["number"])
        invoice = billing.write_invoice(self.state, self.closed_job())
        # 10 % of 6000 = 600; 5 % of 1010 = 50.5 -> 50; net 6360; tax 1208.4 -> 1208
        self.assertEqual((invoice["tier"], invoice["discount"], invoice["net"], invoice["vat"]), ("loyal", 650, 6360, 1208))

    def test_refusals(self):
        open_job = jobs.open_job(self.state, "C1", "Road bike")["id"]
        with self.assertRaises(Refusal):
            billing.write_invoice(self.state, open_job)
        job = self.closed_job()
        billing.write_invoice(self.state, job)
        with self.assertRaises(Refusal):
            billing.write_invoice(self.state, job)
        billing.pay(self.state, "INV-0001")
        with self.assertRaises(Refusal):
            billing.pay(self.state, "INV-0001")


if __name__ == "__main__":
    unittest.main()
