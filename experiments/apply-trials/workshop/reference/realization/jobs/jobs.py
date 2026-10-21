"""Repair jobs: opened for a customer, labor and parts while open, closed once."""
from customers import customers
from parts import parts
from shop import inputs
from shop.errors import Refusal

MAX_MINUTES = 2400
STATUSES = ("open", "closed")


def existing(state, job_id):
    for job in state["jobs"]:
        if job["id"] == job_id:
            return job
    raise Refusal("there is no job %s" % job_id)


def open_job_of(state, job_id):
    job = existing(state, job_id)
    if job["status"] != "open":
        raise Refusal("the job %s is closed" % job_id)
    return job


def record(job):
    return {"id": job["id"], "customer": job["customer"], "bike": job["bike"], "status": job["status"],
            "minutes": job["minutes"], "parts": [dict(entry) for entry in job["parts"]]}


def open_job(state, customer_id, bike_text):
    customer, bike = customers.existing(state, customer_id), inputs.text(bike_text, "a bicycle's description")
    job = {"id": "J%d" % (len(state["jobs"]) + 1), "customer": customer["id"], "bike": bike, "status": "open",
           "minutes": 0, "parts": []}
    state["jobs"].append(job)
    return record(job)


def add_labor(state, job_id, minutes_text):
    job, minutes = open_job_of(state, job_id), inputs.minutes(minutes_text)
    if job["minutes"] + minutes > MAX_MINUTES:
        raise Refusal("a job has at most %d minutes of labor" % MAX_MINUTES)
    job["minutes"] += minutes
    return record(job)


def use_part(state, job_id, sku_text, qty_text):
    job = open_job_of(state, job_id)
    key, quantity = parts.existing(state, sku_text), inputs.quantity(qty_text)
    parts.reserve(state, key, quantity)
    for entry in job["parts"]:
        if entry["sku"] == key:
            entry["qty"] += quantity
            break
    else:
        job["parts"].append({"sku": key, "qty": quantity})
    return record(job)


def close_job(state, job_id):
    job = open_job_of(state, job_id)
    for entry in job["parts"]:
        parts.consume(state, entry["sku"], entry["qty"])
    job["status"] = "closed"
    return record(job)


def listing(state, status=None):
    if status is not None and status not in STATUSES:
        raise Refusal("a status is open or closed, got %r" % status)
    return [record(job) for job in state["jobs"] if status is None or job["status"] == status]


def validate(state):
    section = state["jobs"]
    if not isinstance(section, list):
        raise ValueError("jobs must be an array")
    known = {entry["id"] for entry in state["customers"]}
    reserved = {}
    for index, job in enumerate(section, start=1):
        if (not isinstance(job, dict) or set(job) != {"id", "customer", "bike", "status", "minutes", "parts"}
                or job["id"] != "J%d" % index or job["customer"] not in known or job["status"] not in STATUSES
                or not isinstance(job["bike"], str) or type(job["minutes"]) is not int
                or not 0 <= job["minutes"] <= MAX_MINUTES or job["minutes"] % 15 or not isinstance(job["parts"], list)):
            raise ValueError("invalid job %r" % (job,))
        seen = set()
        for entry in job["parts"]:
            if (not isinstance(entry, dict) or set(entry) != {"sku", "qty"} or entry["sku"] not in state["parts"]
                    or entry["sku"] in seen or type(entry["qty"]) is not int or entry["qty"] < 1):
                raise ValueError("invalid part entry in job %s" % job["id"])
            seen.add(entry["sku"])
            if job["status"] == "open":
                reserved[entry["sku"]] = reserved.get(entry["sku"], 0) + entry["qty"]
    for key, part in state["parts"].items():
        if part["reserved"] != reserved.get(key, 0):
            raise ValueError("the reserved quantity of %s does not match the open jobs" % key)
