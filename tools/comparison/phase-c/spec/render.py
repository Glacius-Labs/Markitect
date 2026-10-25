"""Render the single-agent arm's Makerspace documentation from the model snapshots.

Uses the comparison renderer (tools/comparison/requirements.py) unchanged in
method: every Definition body verbatim under plain headings, body links as
Markdown links to the linked section, and only the SUBSTITUTIONS below, which
replace the phrases that name the model itself. The snapshots are complete, so
each step is read directly from phase-c/model/<step>/.

Usage, from the repository root:
  python -I -B tools/comparison/phase-c/spec/render.py render --step w0 [--out FILE]
  python -I -B tools/comparison/phase-c/spec/render.py check --step w0 [--doc FILE]
  python -I -B tools/comparison/phase-c/spec/render.py write-all
  python -I -B tools/comparison/phase-c/spec/render.py check-all
"""
import argparse
import json
import pathlib
import sys

HERE = pathlib.Path(__file__).resolve().parent
PHASE_C = HERE.parent
sys.path.insert(0, str(PHASE_C.parent))
import requirements  # noqa: E402

STEPS = ("w0", "c1", "c2", "c3", "c4", "c5", "c6", "c7", "c8", "c9")
INTRO = "This document describes the Makerspace program to build. Links point to the related sections."

SUBSTITUTIONS = [
    ("workshop/requests", "operation of the model", "operation of this document", 1),
    ("workshop/invalid-requests", "operation of the model", "operation of this document", 1),
    ("workshop/argument-values", "Every range in the model", "Every range in this document", 1),
    ("workshop/initial-state", "in the model means", "in this document means", 1),
    ("workshop/delivery", "agree with the model.", "agree with this document.", 1),
    ("workshop/computed-values", "rules of the model", "rules of this document", 1),
    ("workshop/computed-values", "formula in the model", "formula in this document", 1),
]


def op(level, name, key):
    return (level, f"`{name}`", [key])


def sections(definitions):
    """Document order of W0, with the later Definitions in place; billing follows the C7 fold once payments are gone."""
    if "billing/payments/payment" in definitions:
        billing = [
            (3, "Charges and invoices", ["billing/invoices/charge", "billing/invoices/invoice"]),
            (4, "Blocked member", ["billing/invoices/blocked-member"]),
            op(4, "generateInvoices", "billing/invoices/generate-invoices"),
            op(4, "listInvoices", "billing/invoices/list-invoices"),
            (3, "Payments and credit", ["billing/payments/payment"]),
            op(4, "recordPayment", "billing/payments/record-payment"),
            op(4, "issueCredit", "billing/payments/issue-credit"),
            op(4, "getAccount", "billing/payments/get-account"),
        ]
    else:
        billing = [
            (3, "Invoices", ["billing/invoices/charge", "billing/invoices/invoice"]),
            (4, "Blocked member", ["billing/invoices/blocked-member"]),
            (4, "Payments and credit", ["billing/invoices/payment"]),
            op(4, "generateInvoices", "billing/invoices/generate-invoices"),
            op(4, "listInvoices", "billing/invoices/list-invoices"),
            op(4, "recordPayment", "billing/invoices/record-payment"),
            op(4, "issueCredit", "billing/invoices/issue-credit"),
            op(4, "getAccount", "billing/invoices/get-account"),
        ]
    return [
        (2, "Overview", ["workshop/makerspace"]),
        (2, "Program interface", []),
        (3, "Program, build and process", ["workshop/program"]),
        (3, "Requests", ["workshop/requests"]),
        (3, "Responses", ["workshop/responses"]),
        (3, "Invalid requests", ["workshop/invalid-requests"]),
        (3, "Argument values", ["workshop/argument-values"]),
        (3, "Initial state and generated identifiers", ["workshop/initial-state"]),
        (3, "Records in responses", ["workshop/results"]),
        (3, "Computed values and limits", ["workshop/computed-values"]),
        (2, "Processing a request", []),
        (3, "Order of checks", ["workshop/request-processing"]),
        (3, "Refusal codes", ["workshop/refusal-codes"]),
        (2, "Cross-cutting policies", []),
        (3, "Authorization", ["policies/authorization/authorization"]),
        (4, "Permission matrix", ["policies/authorization/permission-matrix"]),
        (3, "Audit", ["policies/audit/audit"]),
        (4, "Audit target list", ["policies/audit/audit-targets"]),
        op(4, "listAuditEntries", "policies/audit/list-audit-entries"),
        (3, "Money", ["policies/money/money"]),
        (4, "Credit balance", ["policies/money/credit-balance"]),
        (3, "Time", ["policies/time/time"]),
        (4, "Opening hours", ["policies/time/opening-hours"]),
        (4, "Booking granularity", ["policies/time/booking-granularity"]),
        (3, "Order of listed results", ["workshop/result-order"]),
        (2, "Records", ["workshop/records"]),
        (2, "Members", []),
        (3, "Registry", ["members/registry/registry"]),
        op(4, "registerUser", "members/registry/register-user"),
        op(4, "registerMember", "members/registry/register-member"),
        op(4, "getMember", "members/registry/get-member"),
        (3, "Memberships", ["members/memberships/membership"]),
        (4, "Good standing", ["members/memberships/good-standing"]),
        op(4, "changeTier", "members/memberships/change-tier"),
        op(4, "setMemberStanding", "members/memberships/set-member-standing"),
        (2, "Training", []),
        (3, "Courses and enrollments", ["training/course"]),
        op(4, "addCourse", "training/add-course"),
        op(4, "enrollInCourse", "training/enroll-in-course"),
        op(4, "completeCourse", "training/complete-course"),
        (3, "Certifications", ["training/certification"]),
        op(4, "revokeCertification", "training/revoke-certification"),
        op(4, "listCertifications", "training/list-certifications"),
        (2, "Equipment", []),
        (3, "Catalog", ["equipment/catalog/catalog"]),
        op(4, "addFamily", "equipment/catalog/add-family"),
        op(4, "addMachine", "equipment/catalog/add-machine"),
        op(4, "setMachineRate", "equipment/catalog/set-machine-rate"),
        op(4, "listMachines", "equipment/catalog/list-machines"),
        (3, "Maintenance", ["equipment/maintenance/maintenance-window"]),
        op(4, "scheduleMaintenance", "equipment/maintenance/schedule-maintenance"),
        op(4, "cancelMaintenance", "equipment/maintenance/cancel-maintenance"),
        (2, "Bookings", []),
        (3, "Reservations", ["bookings/reservation"]),
        (4, "Reservation rules", ["bookings/reservation-rules"]),
        op(4, "createReservation", "bookings/create-reservation"),
        op(4, "cancelReservation", "bookings/cancel-reservation"),
        op(4, "listMemberReservations", "bookings/list-member-reservations"),
        op(4, "getMachineSchedule", "bookings/get-machine-schedule"),
        (3, "Check-in, check-out and metering", ["bookings/metering"]),
        op(4, "checkIn", "bookings/check-in"),
        op(4, "checkOut", "bookings/check-out"),
        (2, "Materials", ["materials/shop"]),
        (3, "Bulk price tiers", ["materials/bulk-discount"]),
        op(3, "addMaterial", "materials/add-material"),
        op(3, "restockMaterial", "materials/restock-material"),
        op(3, "purchaseMaterial", "materials/purchase-material"),
        op(3, "listMaterials", "materials/list-materials"),
        (2, "Billing", []),
        *billing,
        (2, "Safety", ["safety/incident"]),
        op(3, "reportIncident", "safety/report-incident"),
        op(3, "resolveIncident", "safety/resolve-incident"),
        op(3, "listIncidents", "safety/list-incidents"),
        (3, "Lockouts", ["safety/lockout"]),
        op(4, "lockMachine", "safety/lock-machine"),
        (2, "Reporting", ["reporting/reports"]),
        op(3, "utilizationReport", "reporting/utilization-report"),
        op(3, "revenueReport", "reporting/revenue-report"),
        (2, "Delivery", ["workshop/delivery"]),
    ]


MAKERSPACE = requirements.Document("Makerspace requirements", INTRO, sections, substitutions=SUBSTITUTIONS)


def definitions(step, model_root=PHASE_C / "model"):
    if step not in STEPS:
        raise requirements.ModelError("unknown step " + step)
    base = pathlib.Path(model_root) / step
    files = {p.relative_to(base).as_posix(): p.read_bytes() for p in sorted((base / "model").rglob("*.yaml"))}
    return requirements.load_definitions(files, MAKERSPACE)


def render(step, model_root=PHASE_C / "model"):
    return requirements.render_document(definitions(step, model_root), MAKERSPACE)


def check(step, doc_path=None, model_root=PHASE_C / "model"):
    doc_path = doc_path or HERE / f"requirements-{step}.md"
    return {"step": step, **requirements.check_document(definitions(step, model_root), MAKERSPACE, doc_path)}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    p.add_argument("command", choices=["render", "check", "write-all", "check-all"])
    p.add_argument("--step", choices=STEPS)
    p.add_argument("--out")
    p.add_argument("--doc")
    a = p.parse_args(argv)
    if a.command == "write-all":
        for step in STEPS:
            (HERE / f"requirements-{step}.md").write_bytes(render(step).encode("utf-8"))
        return 0
    if a.command == "check-all":
        results = [check(step) for step in STEPS]
        print(json.dumps([{k: r[k] for k in ("step", "doc_sha256", "identical_to_render", "all_found")} | {"bodies": len(r["bindings"])} for r in results], indent=2))
        return 0 if all(r["identical_to_render"] and r["all_found"] for r in results) else 1
    if not a.step:
        p.error(a.command + " needs --step")
    if a.command == "render":
        text = render(a.step)
        if a.out:
            pathlib.Path(a.out).write_bytes(text.encode("utf-8"))
        else:
            sys.stdout.buffer.write(text.encode("utf-8"))
        return 0
    result = check(a.step, a.doc)
    print(json.dumps(result, indent=2))
    return 0 if result["identical_to_render"] and result["all_found"] else 1


if __name__ == "__main__":
    sys.exit(main())
