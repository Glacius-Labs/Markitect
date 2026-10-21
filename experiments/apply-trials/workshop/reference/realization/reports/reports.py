"""The workshop report: open jobs, unpaid and paid totals, parts low in stock."""
from parts import parts

LOW_STOCK = 3


def report(state):
    return {"openJobs": sum(1 for job in state["jobs"] if job["status"] == "open"),
            "unpaidTotal": sum(invoice["total"] for invoice in state["invoices"] if not invoice["paid"]),
            "paidTotal": sum(invoice["total"] for invoice in state["invoices"] if invoice["paid"]),
            "lowStock": [key for key in sorted(state["parts"]) if parts.available(state, key) < LOW_STOCK]}
