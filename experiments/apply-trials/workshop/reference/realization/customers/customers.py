"""Customers: registration, lookup and the loyalty tier from the paid invoices of the billing part."""
from shop import inputs
from shop.errors import Refusal

LOYAL_AFTER = 3


def existing(state, customer_id):
    for entry in state["customers"]:
        if entry["id"] == customer_id:
            return entry
    raise Refusal("there is no customer %s" % customer_id)


def tier(state, customer_id):
    """The loyalty tier now: loyal from three paid invoices."""
    from billing import billing  # the billing part owns invoices; it also asks for the tier
    return "loyal" if billing.paid_count(state, customer_id) >= LOYAL_AFTER else "standard"


def record(state, entry):
    from billing import billing
    return {"id": entry["id"], "name": entry["name"], "paidInvoices": billing.paid_count(state, entry["id"]),
            "tier": tier(state, entry["id"])}


def add_customer(state, name_text):
    name = inputs.text(name_text, "a customer's name")
    entry = {"id": "C%d" % (len(state["customers"]) + 1), "name": name}
    state["customers"].append(entry)
    return record(state, entry)


def show(state, customer_id):
    return record(state, existing(state, customer_id))


def validate(state):
    section = state["customers"]
    if not isinstance(section, list):
        raise ValueError("customers must be an array")
    for index, entry in enumerate(section, start=1):
        if (not isinstance(entry, dict) or set(entry) != {"id", "name"} or entry["id"] != "C%d" % index
                or not isinstance(entry["name"], str) or inputs.text(entry["name"], "a name") != entry["name"]):
            raise ValueError("invalid customer %r" % (entry,))
