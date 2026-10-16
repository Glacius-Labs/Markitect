"""Oracle self-tests use controlled transcripts, never a substitute library."""

from __future__ import annotations

import copy
import unittest

import oracle


def materialize(value, captures):
    """Make a deterministic protocol fixture from the independent expectation."""
    if isinstance(value, dict):
        if set(value) == {"$ref"}:
            return captures[value["$ref"]]
        if set(value) == {"$capture"}:
            return f"fixed-opaque-{value['$capture']}"
        if value == {"$nonempty": True}:
            return "Controlled refusal reason"
        return {key: materialize(item, captures) for key, item in value.items()}
    if isinstance(value, list):
        return [materialize(item, captures) for item in value]
    return value


def bind_captures(step, response, captures):
    for symbol, dotted_path in step.get("capture", {}).items():
        value = response
        for part in dotted_path.split("."):
            value = value[part]
        captures[symbol] = value


def expected_transcript(case):
    captures = {}
    rows = []
    for step in case["steps"]:
        response = materialize(step["expect"], captures)
        rows.append(response)
        bind_captures(step, response, captures)
    return rows


def fixture_at(case, index):
    captures = {}
    for position, (step, response) in enumerate(zip(case["steps"], expected_transcript(case))):
        if position == index:
            return response, captures
        bind_captures(step, response, captures)
    raise IndexError(index)


class OracleTests(unittest.TestCase):
    def test_repeated_correlation_id_does_not_replay_registration_success(self):
        case = next(case for case in oracle.build_cases("m1") if case["id"] == "E12-callable")
        exact = '{"id":"same-id","op":"registerMember","args":{"memberId":"transport-a","name":"Transport A"}}'
        duplicates = [i for i, step in enumerate(case["steps"]) if step.get("raw") == exact]
        self.assertEqual(len(duplicates), 2)
        first, repeated = (case["steps"][index] for index in duplicates)
        self.assertEqual(first["expect"]["status"], "ok")
        self.assertEqual(repeated["expect"]["status"], "refused")
        cached_success, captures = fixture_at(case, duplicates[0])
        self.assertTrue(oracle.evaluate_step(repeated, cached_success, captures)["errors"])
        post_snapshot = next(step for step in case["steps"][duplicates[1] + 1:]
                             if step.get("snapshot") and step.get("request", {}).get("op") == "inspectMember"
                             and step["request"]["args"].get("memberId") == "transport-a")
        response, captures = fixture_at(case, case["steps"].index(post_snapshot))
        self.assertEqual(response["result"]["member"]["name"], "Transport A")
        self.assertFalse(oracle.evaluate_step(post_snapshot, response, captures)["errors"])

    def test_unknown_loan_probes_do_not_assume_generated_id_format(self):
        step = {"request": {"id": "probe", "op": "inspectLoan", "args": {"loanId": {"$unusedLoan": "unknown"}}}}
        captures = {"a": "unknown", "b": "unknown_", "c": "opaque space / ü"}
        self.assertEqual(oracle.resolve_request(step, captures)["args"]["loanId"], "unknown__")

    def test_success_shaped_refusal_is_rejected(self):
        step = {"expect": {"id": "q", "status": "refused", "reason": {"$nonempty": True}}}
        self.assertTrue(oracle.evaluate_step(step, {"id": "q", "status": "refused", "reason": "Unknown", "result": None}, {})["errors"])

    def test_paired_refusal_preserves_member_and_unrelated_copy(self):
        case = oracle.witness_case("m2")
        fourth = next(i for i, step in enumerate(case["steps"]) if step.get("witness") == "fourth-admission")
        for op, field, value in (("inspectMember", "status", "suspended"), ("inspectCopy", "available", False)):
            index = next(i for i in range(fourth + 1, len(case["steps"]))
                         if case["steps"][i].get("snapshot") and case["steps"][i]["request"]["op"] == op
                         and (op != "inspectCopy" or case["steps"][i]["request"]["args"]["copyId"] == "copy-5"))
            response, captures = fixture_at(case, index)
            record = "member" if op == "inspectMember" else "copy"
            response["result"][record][field] = value
            self.assertTrue(oracle.evaluate_step(case["steps"][index], response, captures)["errors"])

    def test_trimmed_identifiers_names_and_titles_fail(self):
        for profile in ("m1", "m2"):
            case = next(case for case in oracle.build_cases(profile) if case["id"] == "E12-callable")
            for op, record, field, identifier in (("registerMember", "member", "memberId", " padded-member "),
                                                 ("registerMember", "member", "name", " padded-member "),
                                                 ("registerCopy", "copy", "copyId", " padded-copy "),
                                                 ("registerCopy", "copy", "title", " padded-copy ")):
                with self.subTest(profile=profile, field=field):
                    key = "memberId" if record == "member" else "copyId"
                    index = next(i for i, step in enumerate(case["steps"]) if step["request"]["op"] == op
                                 and step["request"]["args"].get(key) == identifier)
                    response, captures = fixture_at(case, index)
                    response["result"][record][field] = response["result"][record][field].strip()
                    self.assertTrue(oracle.evaluate_step(case["steps"][index], response, captures)["errors"])

    def test_profiles_and_required_families_are_explicit(self):
        m1 = oracle.expectation_profile("m1")
        m2 = oracle.expectation_profile("m2")
        self.assertEqual((m1["maxOpenLoans"], m1["loanPeriodDays"]), (5, 14))
        self.assertEqual((m2["maxOpenLoans"], m2["loanPeriodDays"]), (3, 14))
        for profile in ("m1", "m2"):
            cases = oracle.build_cases(profile)
            families = {family for case in cases for family in case["families"]}
            self.assertTrue({f"E{i:02}" for i in range(1, 13)} <= families)
            self.assertEqual(sum(case["id"] == "paired-fourth-loan" for case in cases), 1)
        m1_steps = oracle.build_cases("m1")[-1]["steps"]
        m2_steps = oracle.build_cases("m2")[-1]["steps"]
        self.assertEqual(len(m1_steps), len(m2_steps))
        self.assertEqual([step["request"] for step in m1_steps], [step["request"] for step in m2_steps])
        self.assertEqual([step.get("witness") for step in m1_steps], [step.get("witness") for step in m2_steps])
        self.assertEqual([step["witness"] for step in m1_steps if "witness" in step], ["fourth-admission", "open-loan-count", "copy-4-availability"])
        fourth_m1 = next(step for step in m1_steps if step.get("witness") == "fourth-admission")
        fourth_m2 = next(step for step in m2_steps if step.get("witness") == "fourth-admission")
        self.assertNotEqual(fourth_m1["expect"]["status"], fourth_m2["expect"]["status"])

    def test_controlled_expected_transcripts_pass_for_every_case(self):
        for profile in ("m1", "m2"):
            for case in oracle.build_cases(profile):
                captures = {}
                for step, response in zip(case["steps"], expected_transcript(case)):
                    outcome = oracle.evaluate_step(step, response, captures)
                    self.assertFalse(outcome["errors"], (profile, case["id"], step, outcome))
                    self.assertGreater(outcome["assertions"], 0)
                    bind_captures(step, response, captures)

    def test_wrong_date_and_wrong_policy_result_fail(self):
        case = next(c for c in oracle.build_cases("m1") if c["id"] == "E06-calendar")
        index = next(i for i, row in enumerate(case["steps"]) if row["request"].get("args", {}).get("borrowDate") == "2028-02-20")
        step = case["steps"][index]
        response, captures = fixture_at(case, index)
        response["result"]["loan"]["dueDate"] = "2028-03-06"
        self.assertTrue(oracle.evaluate_step(step, response, captures)["errors"])

        witness_m2 = oracle.build_cases("m2")[-1]
        index = next(i for i, row in enumerate(witness_m2["steps"]) if row.get("witness") == "fourth-admission")
        response, captures = fixture_at(witness_m2, index)
        response["status"] = "ok"
        response.pop("reason", None)
        response["result"] = {"loan": {"loanId": "unexpected", "memberId": "member-limit", "copyId": "copy-4",
                                       "borrowDate": "2026-10-01", "dueDate": "2026-10-15", "status": "open", "returnDate": None}}
        self.assertTrue(oracle.evaluate_step(witness_m2["steps"][index], response, captures)["errors"])

    def test_duplicate_notices_and_mutating_refusal_fail(self):
        case = next(c for c in oracle.build_cases("m1") if c["id"] == "E10-overdue")
        index = next(i for i, row in enumerate(case["steps"]) if not row.get("snapshot") and row.get("request", {}).get("op") == "assessOverdue" and row["request"].get("args", {}).get("assessmentDate") == "2026-10-16")
        step = case["steps"][index]
        response, captures = fixture_at(case, index)
        response["result"]["notices"].append(copy.deepcopy(response["result"]["notices"][0]))
        result = oracle.evaluate_step(step, response, captures)
        self.assertTrue(result["errors"])

        admission = next(c for c in oracle.build_cases("m2") if c["id"] == "E04-admission")
        refused_index = next(i for i, row in enumerate(admission["steps"]) if not row.get("snapshot") and row.get("request", {}).get("op") == "borrow" and row["expect"].get("status") == "refused")
        response, captures = fixture_at(admission, refused_index)
        response["status"] = "ok"
        response.pop("reason", None)
        response["result"] = {"loan": {"loanId": "mutated", "memberId": "member-a", "copyId": "copy-4",
                                       "borrowDate": "2026-10-01", "dueDate": "2026-10-15", "status": "open", "returnDate": None}}
        self.assertTrue(oracle.evaluate_step(admission["steps"][refused_index], response, captures)["errors"])

        ordinary = next(c for c in oracle.build_cases("m1") if c["id"] == "E03-successful-loan")
        refusal_index = next(i for i, row in enumerate(ordinary["steps"])
                             if not row.get("snapshot") and row.get("request", {}).get("op") == "borrow"
                             and row.get("expect", {}).get("status") == "refused")
        after_snapshot = next(i for i in range(refusal_index + 1, len(ordinary["steps"]))
                              if ordinary["steps"][i].get("snapshot")
                              and ordinary["steps"][i].get("request", {}).get("op") == "inspectCopy"
                              and ordinary["steps"][i]["request"]["args"].get("copyId") == "copy-1")
        state_check = ordinary["steps"][after_snapshot]
        response, captures = fixture_at(ordinary, after_snapshot)
        if state_check["request"]["op"] == "inspectCopy":
            response["result"]["copy"]["available"] = True
        else:
            self.fail("refusal snapshot did not inspect observable state")
        self.assertTrue(oracle.evaluate_step(state_check, response, captures)["errors"])

    def test_list_order_is_ignored_but_duplicate_ids_are_rejected(self):
        case = next(c for c in oracle.build_cases("m1") if c["id"] == "E04-admission")
        index = next(i for i, row in enumerate(case["steps"]) if row["request"]["op"] == "inspectOpenLoans")
        step = case["steps"][index]
        response, captures = fixture_at(case, index)
        response["result"]["loans"].reverse()
        self.assertFalse(oracle.evaluate_step(step, response, captures)["errors"])
        response["result"]["loans"].append(copy.deepcopy(response["result"]["loans"][0]))
        self.assertTrue(oracle.evaluate_step(step, response, captures)["errors"])

    def test_snapshot_catches_old_return_closing_new_loan_and_assessment_mutation(self):
        old_return = next(c for c in oracle.build_cases("m1") if c["id"] == "E08-old-return")
        repeat_index = next(i for i, row in enumerate(old_return["steps"])
                            if not row.get("snapshot") and row.get("request", {}).get("op") == "return"
                            and row.get("expect", {}).get("result", {}).get("outcome") == "alreadyReturned")
        later_loan_index = next(i for i in range(repeat_index + 1, len(old_return["steps"]))
                                if old_return["steps"][i].get("snapshot")
                                and old_return["steps"][i].get("request", {}).get("op") == "inspectLoan"
                                and old_return["steps"][i]["request"]["args"].get("loanId") == {"$ref": "new"})
        step = old_return["steps"][later_loan_index]
        response, captures = fixture_at(old_return, later_loan_index)
        response["result"]["loan"]["status"] = "closed"
        response["result"]["loan"]["returnDate"] = "2026-10-04"
        self.assertTrue(oracle.evaluate_step(step, response, captures)["errors"])

        overdue = next(c for c in oracle.build_cases("m1") if c["id"] == "E10-overdue")
        assessment_index = next(i for i, row in enumerate(overdue["steps"])
                                if not row.get("snapshot") and row.get("request", {}).get("op") == "assessOverdue"
                                and row["request"].get("args", {}).get("assessmentDate") == "2026-10-16")
        later_read = next(i for i in range(assessment_index + 1, len(overdue["steps"]))
                          if overdue["steps"][i].get("snapshot")
                          and overdue["steps"][i].get("request", {}).get("op") == "inspectLoan"
                          and overdue["steps"][i]["request"]["args"].get("loanId") == {"$ref": "loan-1"})
        step = overdue["steps"][later_read]
        response, captures = fixture_at(overdue, later_read)
        response["result"]["loan"]["status"] = "closed"
        response["result"]["loan"]["returnDate"] = "2026-10-16"
        self.assertTrue(oracle.evaluate_step(step, response, captures)["errors"])


if __name__ == "__main__":
    unittest.main()
