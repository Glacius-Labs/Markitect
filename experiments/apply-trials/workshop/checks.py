"""Workshop checks: black-box expectations of snapshot w1, the realistic integrated case.

Every expected amount was computed by hand from the model text; none is
derived from a realization. Error wording and the organization of the data
file are free; only the envelope, exit code, unchanged data and the
specified records are checked.
"""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from trial_kit import Checks, expect_equal, fail, no_crash, unusable_change, unusable_query  # noqa: E402

CHECKS = Checks("workshop", {"w1": {"snapshot": "w1"}})
W1 = ["w1"]
check = CHECKS.check


def customer(env, name="Ada"):
    return env.ok("add-customer", "--name", name)


def part(env, sku, price, qty=0, name="Part"):
    record = env.ok("add-part", "--sku", sku, "--name", name, "--price", str(price))
    if qty:
        record = env.ok("receive", "--sku", sku, "--qty", str(qty))
    return record


def job(env, customer_id="C1", minutes=(), uses=(), close=True):
    record = env.ok("open-job", "--customer", customer_id, "--bike", "City bike")
    for value in minutes:
        env.ok("add-labor", "--job", record["id"], "--minutes", str(value))
    for sku, qty in uses:
        env.ok("use-part", "--job", record["id"], "--sku", sku, "--qty", str(qty))
    if close:
        record = env.ok("close-job", "--job", record["id"])
    return record


def collection(env, command, field, *extra):
    result = env.ok(command, *extra)
    if set(result) != {field} or not isinstance(result[field], list):
        fail("`%s` must return exactly {\"%s\": [...]}, got %s" % (command, field, result))
    return result[field]


def invoice(env, job_id):
    return env.ok("invoice", "--job", job_id)


def amounts(record):
    return {key: record[key] for key in ("labor", "parts", "discount", "net", "vat", "total")}


@check("empty-state", W1, ["shop/storage", "shop/parts/list-parts", "shop/jobs/list-jobs",
                           "shop/billing/list-invoices", "shop/reports/report"], "A missing data file is empty data")
def _(env, p):
    expect_equal(collection(env, "parts", "parts"), [], "parts")
    expect_equal(collection(env, "jobs", "jobs"), [], "jobs")
    expect_equal(collection(env, "invoices", "invoices"), [], "invoices")
    expect_equal(env.ok("report"), {"openJobs": 0, "unpaidTotal": 0, "paidTotal": 0, "lowStock": []}, "report")


@check("customer-record", W1, ["shop/customers/customer", "shop/customers/add-customer",
                               "shop/customers/show-customer", "shop/inputs"],
       "Customers get ids C1, C2, ... and show their tier; names follow the text rule")
def _(env, p):
    expect_equal(customer(env, " Ada "), {"id": "C1", "name": "Ada", "paidInvoices": 0, "tier": "standard"}, "C1")
    expect_equal(customer(env, "é" * 60)["id"], "C2", "60 code points")
    expect_equal(env.ok("customer", "--id", "C2")["name"], "é" * 60, "show C2")
    for name in ("", "   ", "x" * 61):
        env.refuse("add-customer", "--name", name)
    for unknown in ("C3", "c1", " C1", "1"):
        env.refuse("customer", "--id", unknown)


@check("part-record", W1, ["shop/parts/part", "shop/parts/add-part", "shop/inputs", "shop/parts/stock"],
       "add-part normalizes the SKU, checks name and price and starts without stock")
def _(env, p):
    expect_equal(part(env, " brk-101 ", "0125", name=" Brake pad "),
                 {"sku": "BRK-101", "name": "Brake pad", "price": 125, "onHand": 0, "reserved": 0, "available": 0},
                 "added part")
    for sku in ("brk-101", "BR-101", "BRK101", "BRK-1011", ""):
        env.refuse("add-part", "--sku", sku, "--name", "X", "--price", "1")
    for price in ("0", "1000001", "-1", "1.5", " 5", "x"):
        env.refuse("add-part", "--sku", "CHN-010", "--name", "X", "--price", price)
    env.refuse("add-part", "--sku", "CHN-010", "--name", "", "--price", "1")
    expect_equal(part(env, "CHN-010", 1)["price"], 1, "price 1")
    expect_equal(part(env, "CHN-011", 1000000)["price"], 1000000, "price 1000000")
    expect_equal([entry["sku"] for entry in collection(env, "parts", "parts")], ["BRK-101", "CHN-010", "CHN-011"],
                 "parts by SKU")


@check("receive-cap", W1, ["shop/parts/receive", "shop/parts/stock"], "On hand reaches exactly 9999 and no further")
def _(env, p):
    part(env, "BRK-101", 100)
    for _ in range(19):
        env.ok("receive", "--sku", "brk-101", "--qty", "500")
    expect_equal(env.ok("receive", "--sku", "BRK-101", "--qty", "499")["onHand"], 9999, "9999 on hand")
    env.refuse("receive", "--sku", "BRK-101", "--qty", "1")
    for qty in ("0", "501", "-1"):
        env.refuse("receive", "--sku", "BRK-101", "--qty", qty)
    env.refuse("receive", "--sku", "XXX-999", "--qty", "1")


@check("job-lifecycle", W1, ["shop/jobs/job", "shop/jobs/open-job", "shop/jobs/add-labor", "shop/jobs/close-job",
                             "shop/inputs"], "Jobs open for existing customers, take labor in steps of 15 and close once")
def _(env, p):
    env.refuse("open-job", "--customer", "C1", "--bike", "Bike")
    customer(env)
    expect_equal(env.ok("open-job", "--customer", "C1", "--bike", " Gravel bike "),
                 {"id": "J1", "customer": "C1", "bike": "Gravel bike", "status": "open", "minutes": 0, "parts": []},
                 "opened job")
    for minutes in ("10", "0", "495", "15.0", ""):
        env.refuse("add-labor", "--job", "J1", "--minutes", minutes)
    for _ in range(5):
        env.ok("add-labor", "--job", "J1", "--minutes", "480")
    env.refuse("add-labor", "--job", "J1", "--minutes", "15")
    expect_equal(env.ok("close-job", "--job", "J1")["status"], "closed", "closed")
    env.refuse("close-job", "--job", "J1")
    env.refuse("add-labor", "--job", "J1", "--minutes", "15")
    env.refuse("close-job", "--job", "J9")


@check("reservation", W1, ["shop/jobs/use-part", "shop/parts/stock", "shop/jobs/job"],
       "use-part reserves available units and lists one entry per SKU in the order of first use")
def _(env, p):
    customer(env)
    part(env, "BRK-101", 1250, 3)
    part(env, "AAA-001", 200, 5)
    job(env, close=False)
    env.ok("use-part", "--job", "J1", "--sku", "brk-101", "--qty", "1")
    env.ok("use-part", "--job", "J1", "--sku", "AAA-001", "--qty", "2")
    record = env.ok("use-part", "--job", "J1", "--sku", "BRK-101", "--qty", "2")
    expect_equal(record["parts"], [{"sku": "BRK-101", "qty": 3}, {"sku": "AAA-001", "qty": 2}], "job parts")
    expect_equal(collection(env, "parts", "parts")[1], {"sku": "BRK-101", "name": "Part", "price": 1250, "onHand": 3,
                                                       "reserved": 3, "available": 0}, "reserved part")
    env.refuse("use-part", "--job", "J1", "--sku", "BRK-101", "--qty", "1")
    env.refuse("use-part", "--job", "J1", "--sku", "ZZZ-999", "--qty", "1")
    env.refuse("use-part", "--job", "J2", "--sku", "AAA-001", "--qty", "1")


@check("close-consumes", W1, ["shop/jobs/close-job", "shop/parts/stock"],
       "Closing a job takes its reserved units out of on hand and reserved")
def _(env, p):
    customer(env)
    part(env, "BRK-101", 1250, 4)
    job(env, uses=(("BRK-101", 3),))
    expect_equal(collection(env, "parts", "parts")[0]["onHand"], 1, "on hand after closing")
    expect_equal(collection(env, "parts", "parts")[0]["reserved"], 0, "reserved after closing")
    job(env, close=False)
    env.refuse("use-part", "--job", "J2", "--sku", "BRK-101", "--qty", "2")


@check("jobs-listing", W1, ["shop/jobs/list-jobs"], "jobs lists in creation order and filters by status")
def _(env, p):
    customer(env)
    job(env)
    job(env, close=False)
    job(env)
    expect_equal([entry["id"] for entry in collection(env, "jobs", "jobs")], ["J1", "J2", "J3"], "all jobs")
    expect_equal([entry["id"] for entry in collection(env, "jobs", "jobs", "--status", "closed")], ["J1", "J3"], "closed")
    expect_equal([entry["id"] for entry in collection(env, "jobs", "jobs", "--status", "open")], ["J2"], "open")
    for status in ("Open", "done", ""):
        env.refuse("jobs", "--status", status)


@check("invoice-standard", W1, ["shop/billing/pricing", "shop/billing/invoice", "shop/billing/write-invoice"],
       "A standard invoice: labor per quarter hour, parts at their price, tax rounded")
def _(env, p):
    customer(env)
    part(env, "BRK-101", 1250, 5)
    part(env, "CBL-002", 399, 5)
    job(env, minutes=(30, 45), uses=(("BRK-101", 2), ("CBL-002", 1)))
    record = invoice(env, "J1")
    # labor 75 min = 5 x 1500 = 7500; parts 2 x 1250 + 399 = 2899; net 10399; tax 1975.81 -> 1976
    expect_equal(amounts(record), {"labor": 7500, "parts": 2899, "discount": 0, "net": 10399, "vat": 1976,
                                   "total": 12375}, "amounts")
    expect_equal({key: record[key] for key in ("number", "job", "customer", "tier", "paid")},
                 {"number": "INV-0001", "job": "J1", "customer": "C1", "tier": "standard", "paid": False}, "invoice")
    if set(record) != {"number", "job", "customer", "tier", "labor", "parts", "discount", "net", "vat", "total", "paid"}:
        fail("the invoice record has other fields: %s" % sorted(record))


@check("vat-halves", W1, ["shop/billing/pricing"], "Tax rounds an exact half to the even cent")
def _(env, p):
    customer(env)
    part(env, "FRM-001", 1550, 1)
    part(env, "FRM-002", 250, 1)
    job(env, uses=(("FRM-001", 1),))
    job(env, uses=(("FRM-002", 1),))
    # net 1550: tax 294.5 -> 294; net 250: tax 47.5 -> 48
    expect_equal(amounts(invoice(env, "J1")), {"labor": 0, "parts": 1550, "discount": 0, "net": 1550, "vat": 294,
                                               "total": 1844}, "1550")
    expect_equal(amounts(invoice(env, "J2")), {"labor": 0, "parts": 250, "discount": 0, "net": 250, "vat": 48,
                                               "total": 298}, "250")


@check("loyalty", W1, ["shop/customers/loyalty", "shop/billing/pricing", "shop/customers/show-customer",
                       "shop/billing/pay"],
       "Three paid invoices make a customer loyal; the discount rounds half to even and invoices keep their tier")
def _(env, p):
    customer(env)
    customer(env, "Bob")
    part(env, "TUB-001", 1010, 50)
    for index in range(1, 4):
        job(env, minutes=(60,), uses=(("TUB-001", 1),))
        record = invoice(env, "J%d" % index)
        expect_equal(record["tier"], "standard", "tier of invoice %d" % index)
        if index < 3:
            env.ok("pay", "--invoice", record["number"])
    job(env, minutes=(60,), uses=(("TUB-001", 1),))
    expect_equal(invoice(env, "J4")["tier"], "standard", "two paid invoices are not enough")
    expect_equal(env.ok("customer", "--id", "C1"), {"id": "C1", "name": "Ada", "paidInvoices": 2, "tier": "standard"},
                 "C1 with two paid invoices")
    env.ok("pay", "--invoice", "INV-0003")
    expect_equal(env.ok("customer", "--id", "C1")["tier"], "loyal", "C1 after three paid invoices")
    job(env, minutes=(60,), uses=(("TUB-001", 1),))
    # labor 6000: 10 % = 600; parts 1010: 5 % = 50.5 -> 50; net 6360; tax 1208.4 -> 1208
    expect_equal(amounts(invoice(env, "J5")), {"labor": 6000, "parts": 1010, "discount": 650, "net": 6360, "vat": 1208,
                                               "total": 7568}, "loyal invoice")
    expect_equal(collection(env, "invoices", "invoices")[3]["tier"], "standard", "INV-0004 keeps its tier")
    job(env, customer_id="C2", minutes=(60,), uses=(("TUB-001", 1),))
    expect_equal(invoice(env, "J6")["discount"], 0, "Bob stays standard")


@check("invoice-refusals", W1, ["shop/billing/write-invoice", "shop/billing/pay", "shop/billing/invoice"],
       "Only closed jobs, once each; payments once; numbers in sequence")
def _(env, p):
    customer(env)
    job(env, close=False)
    env.refuse("invoice", "--job", "J1")
    env.refuse("invoice", "--job", "J9")
    env.ok("close-job", "--job", "J1")
    expect_equal(invoice(env, "J1")["number"], "INV-0001", "first number")
    env.refuse("invoice", "--job", "J1")
    job(env)
    expect_equal(invoice(env, "J2")["number"], "INV-0002", "second number")
    expect_equal(env.ok("pay", "--invoice", "INV-0002")["paid"], True, "paid")
    env.refuse("pay", "--invoice", "INV-0002")
    for number in ("INV-0003", "INV-2", "inv-0001", ""):
        env.refuse("pay", "--invoice", number)


@check("invoices-listing", W1, ["shop/billing/list-invoices"], "invoices lists by number and filters by customer")
def _(env, p):
    customer(env)
    customer(env, "Bob")
    job(env, customer_id="C2")
    job(env)
    job(env, customer_id="C2")
    for index in (1, 2, 3):
        invoice(env, "J%d" % index)
    expect_equal([entry["number"] for entry in collection(env, "invoices", "invoices")],
                 ["INV-0001", "INV-0002", "INV-0003"], "all invoices")
    expect_equal([entry["job"] for entry in collection(env, "invoices", "invoices", "--customer", "C2")], ["J1", "J3"],
                 "Bob's invoices")
    env.refuse("invoices", "--customer", "C9")


@check("report", W1, ["shop/reports/report", "shop/parts/stock", "shop/billing/invoice"],
       "The report counts open jobs, sums unpaid and paid totals and lists parts low in stock")
def _(env, p):
    customer(env)
    part(env, "BRK-101", 1000, 4)
    part(env, "AAA-001", 500, 10)
    part(env, "ZZZ-009", 100)
    job(env, uses=(("BRK-101", 2),))
    job(env, close=False)
    job(env, uses=(("AAA-001", 1),))
    invoice(env, "J1")  # 2000, tax 380, total 2380
    invoice(env, "J3")  # 500, tax 95, total 595
    env.ok("pay", "--invoice", "INV-0002")
    expect_equal(env.ok("report"), {"openJobs": 1, "unpaidTotal": 2380, "paidTotal": 595,
                                    "lowStock": ["BRK-101", "ZZZ-009"]}, "report")


@check("hyphen-values", W1, ["shop/interface", "shop/inputs"], "An option value may start with a hyphen")
def _(env, p):
    expect_equal(customer(env, "-Ada")["name"], "-Ada", "customer name")
    expect_equal(part(env, "BRK-101", 100, name="--brake")["name"], "--brake", "part name")
    expect_equal(env.ok("open-job", "--customer", "C1", "--bike", "-Gravel")["bike"], "-Gravel", "bike")


@check("reads-untouched", W1, ["shop/storage"], "Queries never write the data file")
def _(env, p):
    customer(env)
    part(env, "BRK-101", 100, 2)
    job(env, uses=(("BRK-101", 1),))
    invoice(env, "J1")
    env.reads_leave_data_untouched([["customer", "--id", "C1"], ["parts"], ["jobs"], ["invoices"], ["report"],
                                    ["invoices", "--customer", "C1"], ["jobs", "--status", "open"]])


@check("refusal-keeps-data", W1, ["shop/storage", "shop/interface"], "Refused changes leave the data file unchanged")
def _(env, p):
    customer(env)
    part(env, "BRK-101", 100, 1)
    job(env, close=False)
    env.refuse("use-part", "--job", "J1", "--sku", "BRK-101", "--qty", "2")
    env.refuse("invoice", "--job", "J1")
    env.refuse("add-part", "--sku", "brk-101", "--name", "Again", "--price", "5")


@check("malformed-kept", W1, ["shop/storage", "shop/interface"], "Malformed data fails with the error object and is kept")
def _(env, p):
    for content in ("{", "[{", "\xff"):  # not JSON at all: the organization of valid JSON is free
        env.write_db(content.encode("latin-1") if content == "\xff" else content)
        before = env.raw()
        env.refuse("add-customer", "--name", "Ada")
        env.refuse("report")
        expect_equal(env.raw() == before, True, "malformed data kept")


@check("missing-directory", W1, ["shop/storage", "shop/interface"], "Without its directory a change fails, a query sees empty data")
def _(env, p):
    env.db = env.root / "absent" / "db.json"
    env.refuse("add-customer", "--name", "Ada")
    expect_equal(collection(env, "parts", "parts"), [], "parts")
    env.refuse("customer", "--id", "C1")
    env.refuse("invoices", "--customer", "C1")
    if (env.root / "absent").exists():
        fail("the data directory was created")


@check("usage-errors", W1, ["shop/interface"], "Command-line errors use the error object and exit 2")
def _(env, p):
    for args in ((), ("unknown",), ("add-customer",), ("add-part", "--sku", "BRK-101", "--name", "X"),
                 ("parts", "--help"), ("--help",), ("jobs", "--status"), ("report", "extra"), ("customer", "--idd", "C1")):
        env.refuse(*args)


@check("edge-robustness", W1, ["shop/interface", "shop/inputs", "shop/storage"],
       "Extreme inputs fail through the error object, never with a crash")
def _(env, p):
    customer(env)
    part(env, "BRK-101", 100)
    env.refuse("receive", "--sku", "BRK-101", "--qty", "9" * 5000)
    expect_equal(env.ok("receive", "--sku", "BRK-101", "--qty", "0" * 5000 + "7")["onHand"], 7, "5000 leading zeros")
    no_crash(env, "add-customer", "--name", "Ann \udcff")
    env.write_db("[" * 100000)
    env.refuse("report")
    unusable_query(env, "parts")
    unusable_change(env, "add-customer", "--name", "Ada")


@check("no-leftovers", W1, ["shop/storage"], "Atomic replacement leaves no temporary files")
def _(env, p):
    customer(env)
    part(env, "BRK-101", 100, 1)
    env.refuse("receive", "--sku", "BRK-101", "--qty", "9999")
    leftovers = env.leftover_temp_files()
    if leftovers:
        fail("temporary files left beside the data file: %s" % leftovers)
