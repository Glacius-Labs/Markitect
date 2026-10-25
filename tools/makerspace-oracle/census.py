"""Expected change census per step, where the requirements state what must not change.

C3: only material purchases of 10 or more units may answer differently; every
other request answers as at C2. C7: nothing answers differently than at C6.
C9: nothing changes at all, so even the free `reason` wording stays the same.
`run` replays the selected hidden cases against the start step's program and
the candidate and reports every differing response. Paths per arm are the
comparison assessor's census; this is the behavioral one.
"""

from __future__ import annotations

import base64
import json
from typing import Any

import catalog
from facts import C
import harness

CENSUS: dict[str, dict[str, Any]] = {
    "C3": {"start": "C2", "reason": False,
           "unchanged": ["bookings (reservations, rules, check-in, check-out, metering, usage charges and the tier usage discount)",
                         "memberships (tiers, fees, usage discount table, standing, membership line)",
                         "training (courses, fees, enrollments, course charges, certifications)",
                         "everything else: unit prices, stock, refusal order of purchaseMaterial, invoices, payments, credit, report definitions, audit, authorization, time, records, codes"],
           "allowed": "purchaseMaterial with quantity 10 or more: discountPercent and amountCents of the purchase and its material charge, and what sums them",
           "cite": [C(3, "What must not change", "The bulk discount is a materials rule only. Nothing outside material purchases changes:"),
                    C(3, "What must not change", "No operation, argument, field or code is added or removed.")]},
    "C7": {"start": "C6", "reason": False, "unchanged": ["every behavior"], "allowed": "none",
           "cite": [C(7, "What changes", "A program that meets the requirements after C6 meets the requirements after C7 without any change, and any difference in observable behavior is a defect.")]},
    "C9": {"start": "C8", "reason": True, "unchanged": ["every behavior, and the code, tests and documentation"], "allowed": "none",
           "cite": [C(9, "What must not change", "no change to code, tests or documentation is needed or wanted.")]},
}


def _bulk(case: dict[str, Any]) -> bool:
    """Whether the case expects an accepted purchase of 10 or more units (refused ones cannot differ)."""
    for step in case["steps"]:
        request = step.get("request") or {}
        if (request.get("op") == "purchaseMaterial" and step["expect"].get("status") == "ok"
                and isinstance(request["args"].get("quantity"), int) and request["args"]["quantity"] >= 10):
            return True
    return False


def census_cases(step: str) -> list[dict[str, Any]]:
    """The hidden cases of `step` whose every response must equal the start step's."""
    if step not in CENSUS:
        return []
    cases = catalog.build_cases(step)
    return [case for case in cases if not (step == "C3" and _bulk(case))]


def _responses(case, command, cwd, limits) -> list[Any]:
    result = harness.run_case(case, command, cwd, limits, keep_going=True)
    out = []
    for pair in result.get("trace", {}).get("pairs", []):
        raw = pair.get("response_base64")
        try:
            out.append(json.loads(base64.b64decode(raw)) if raw else None)
        except ValueError:
            out.append(base64.b64decode(raw).decode("utf-8", "replace"))
    return out


def _normalize(response: Any, keep_reason: bool) -> Any:
    if isinstance(response, dict) and not keep_reason:
        return {k: v for k, v in response.items() if k != "reason"}
    return response


def compare(step: str, before: list[Any], after: list[Any]) -> list[dict[str, Any]]:
    keep = CENSUS[step]["reason"]
    diffs = []
    for index in range(max(len(before), len(after))):
        b = _normalize(before[index], keep) if index < len(before) else "<missing>"
        a = _normalize(after[index], keep) if index < len(after) else "<missing>"
        if b != a:
            diffs.append({"index": index, "start": b, "candidate": a})
    return diffs


def run(step: str, start_command: list[str], candidate_command: list[str], cwd: str, limits) -> dict[str, Any]:
    if step not in CENSUS:
        return {"step": step, "applies": False}
    report: dict[str, Any] = {"step": step, "applies": True, "start_step": CENSUS[step]["start"], "cases": {}, "changed_cases": []}
    for case in census_cases(step):
        diffs = compare(step, _responses(case, start_command, cwd, limits), _responses(case, candidate_command, cwd, limits))
        report["cases"][case["id"]] = len(diffs)
        if diffs:
            report["changed_cases"].append({"id": case["id"], "differences": diffs[:5], "count": len(diffs)})
    report["empty"] = not report["changed_cases"]
    return report
