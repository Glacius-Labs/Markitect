import unittest

from billing import billing
from customers import customers
from jobs import jobs
from shop.errors import Refusal
from shop.store import empty


class CustomerTests(unittest.TestCase):
    def setUp(self):
        self.state = empty()

    def test_ids_and_record(self):
        self.assertEqual(customers.add_customer(self.state, " Ada "),
                         {"id": "C1", "name": "Ada", "paidInvoices": 0, "tier": "standard"})
        self.assertEqual(customers.add_customer(self.state, "Ada")["id"], "C2")
        with self.assertRaises(Refusal):
            customers.show(self.state, "C3")

    def test_three_paid_invoices_make_a_customer_loyal(self):
        customers.add_customer(self.state, "Ada")
        for index in range(1, 4):
            job = jobs.open_job(self.state, "C1", "Bike")["id"]
            jobs.close_job(self.state, job)
            billing.write_invoice(self.state, job)
            self.assertEqual(customers.show(self.state, "C1")["tier"], "standard")
            billing.pay(self.state, "INV-%04d" % index)
        self.assertEqual(customers.show(self.state, "C1"), {"id": "C1", "name": "Ada", "paidInvoices": 3, "tier": "loyal"})


if __name__ == "__main__":
    unittest.main()
