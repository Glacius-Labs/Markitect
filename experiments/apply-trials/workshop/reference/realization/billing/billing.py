"""Invoices of closed jobs: pricing with the loyalty discount, tax, numbering and payment."""
from decimal import ROUND_HALF_EVEN, Decimal

from customers import customers
from jobs import jobs
from parts import parts
from shop.errors import Refusal

LABOR_CENTS_PER_QUARTER = 1500
LABOR_DISCOUNT = Decimal("0.10")
PARTS_DISCOUNT = Decimal("0.05")
TAX = Decimal("0.19")
AMOUNTS = ("labor", "parts", "discount", "net", "vat", "total")


def cents(amount, rate):
    """RATE of AMOUNT cents in whole cents, an exact half rounded to the even cent."""
    return int((Decimal(amount) * rate).quantize(Decimal(1), rounding=ROUND_HALF_EVEN))


def number(sequence):
    return "INV-%04d" % sequence


def paid_count(state, customer_id):
    return sum(1 for invoice in state["invoices"] if invoice["customer"] == customer_id and invoice["paid"])


def existing(state, number_text):
    for invoice in state["invoices"]:
        if invoice["number"] == number_text:
            return invoice
    raise Refusal("there is no invoice %s" % number_text)


def amounts(state, job, tier):
    labor = job["minutes"] // 15 * LABOR_CENTS_PER_QUARTER
    parts_amount = sum(entry["qty"] * parts.price(state, entry["sku"]) for entry in job["parts"])
    discount = cents(labor, LABOR_DISCOUNT) + cents(parts_amount, PARTS_DISCOUNT) if tier == "loyal" else 0
    net = labor + parts_amount - discount
    vat = cents(net, TAX)
    return {"labor": labor, "parts": parts_amount, "discount": discount, "net": net, "vat": vat, "total": net + vat}


def write_invoice(state, job_id):
    job = jobs.existing(state, job_id)
    if job["status"] != "closed":
        raise Refusal("the job %s is still open" % job_id)
    if any(invoice["job"] == job_id for invoice in state["invoices"]):
        raise Refusal("the job %s already has an invoice" % job_id)
    tier = customers.tier(state, job["customer"])
    invoice = {"number": number(len(state["invoices"]) + 1), "job": job_id, "customer": job["customer"],
               "tier": tier, **amounts(state, job, tier), "paid": False}
    state["invoices"].append(invoice)
    return dict(invoice)


def pay(state, number_text):
    invoice = existing(state, number_text)
    if invoice["paid"]:
        raise Refusal("the invoice %s is already paid" % number_text)
    invoice["paid"] = True
    return dict(invoice)


def listing(state, customer_id=None):
    if customer_id is not None:
        customers.existing(state, customer_id)
    return [dict(invoice) for invoice in state["invoices"] if customer_id is None or invoice["customer"] == customer_id]


def validate(state):
    section = state["invoices"]
    if not isinstance(section, list):
        raise ValueError("invoices must be an array")
    closed = {job["id"]: job["customer"] for job in state["jobs"] if job["status"] == "closed"}
    seen = set()
    for index, invoice in enumerate(section, start=1):
        keys = {"number", "job", "customer", "tier", "paid", *AMOUNTS}
        if (not isinstance(invoice, dict) or set(invoice) != keys or invoice["number"] != number(index)
                or closed.get(invoice["job"]) != invoice["customer"] or invoice["job"] in seen
                or invoice["tier"] not in ("loyal", "standard") or type(invoice["paid"]) is not bool
                or any(type(invoice[key]) is not int or invoice[key] < 0 for key in AMOUNTS)):
            raise ValueError("invalid invoice %r" % (invoice,))
        seen.add(invoice["job"])
