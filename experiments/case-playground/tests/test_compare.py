"""Fairness check and side-by-side comparison of two assessed runs."""

import contextlib
import copy
import io
import json
import tempfile
import unittest
from pathlib import Path

from playground import __main__ as entry
from playground import compare

CATEGORIES = ("missed_obligation", "unnecessary_change", "rule_violation", "contradiction", "regression",
              "false_claim", "escalation_needed", "escalation_unneeded")


def summary(findings: int, high: int, covered: int, total: int) -> dict:
    return {"status": "ok", "findings": findings, "bySeverity": {"high": high, "medium": 0, "low": findings - high},
            "byCategory": {c: (findings if c == "missed_obligation" else 0) for c in CATEGORIES},
            "obligations": {"covered": covered, "total": total}}


def report(run_id: str, method: str, *, stations: int = 2, image: str = "sha256:img") -> dict:
    entries = []
    for number in range(1, stations + 1):
        entries.append({
            "station": f"S{number}", "items": [f"B0{number}"],
            "publicChecks": {"passed": 4, "total": 5, "status": "fail"},
            "holdouts": {"passed": number, "total": 3, "status": "fail"},
            "diff": {"status": "ok", "files": 3, "added": 30 * number, "deleted": 2,
                     "categories": {"model": {"files": 1, "added": 5, "deleted": 0},
                                    "code": {"files": 1, "added": 20, "deleted": 2},
                                    "tests": {"files": 1, "added": 5, "deleted": 0},
                                    "docs": {"files": 0, "added": 0, "deleted": 0},
                                    "config/other": {"files": 0, "added": 0, "deleted": 0}}},
            "reviewers": {"codex": summary(2, 1, 3, 4), "claude": {"status": "invalid", "error": "bad"}},
            "agreement": None, "escalations": {"codex": {"needed": 1, "unneeded": 0}, "claude": None},
            "classification": {"class": "none", "reason": "station completed"}})
    return {
        "schema": 1, "kind": "assessment",
        "run": {"id": run_id, "case": "readinglog2", "method": method, "outerProvider": "codex",
                "fairness": {"case": "readinglog2", "codex": "0.162.0", "imageId": image, "model": "m",
                             "effort": "high", "limits": {"stationSeconds": 60}, "container": {"cpus": 4}},
                "totals": {"agentSeconds": 100.5, "tokens": {"input": 10, "cachedInput": 5, "output": 1}}},
        "evaluation": {"files": {"groundTruth": {"sha256": "aa"}, "reviewerPrompt": {"sha256": "bb"}}},
        "reviewers": {"codex": {"model": "gpt-6.1-sol", "effort": "high"},
                      "claude": {"model": "claude-opus-5-5", "effort": "high"}},
        "stations": entries, "classification": {"class": "none", "reason": "run completed"},
        "totals": {"publicChecks": {"passed": 8, "total": 10}, "holdouts": {"passed": 3, "total": 6},
                   "failedMcpCalls": 0,
                   "reviewers": {"codex": {"byCategory": {c: (4 if c == "missed_obligation" else 0) for c in CATEGORIES}},
                                 "claude": {"byCategory": {c: 0 for c in CATEGORIES}}}}}


class CompareTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)

    def save(self, name: str, value: dict) -> Path:
        run = self.root / name
        (run / "assessment").mkdir(parents=True)
        (run / "assessment" / "report.json").write_text(json.dumps(value), encoding="utf-8")
        return run

    def main(self, *argv: str) -> tuple[int, str]:
        err = io.StringIO()
        with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(err):
            code = entry.main(["compare", *argv])
        return code, err.getvalue()

    def test_fair_runs_side_by_side(self):
        a = self.save("conv", report("conv-1", "conventional"))
        b_report = report("mkt-1", "markitect", stations=3)
        b_report["evaluation"]["files"]["groundTruth"] = {"sha256": "aa"}
        b = self.save("mkt", b_report)
        code, err = self.main(str(a), str(b))
        self.assertEqual(code, 0, err)
        text = (self.root / "compare-conv-1-vs-mkt-1.md").read_text(encoding="utf-8")
        self.assertIn("Fairness fields match", text)
        self.assertIn("| S1 | 4/5 │ 4/5 | 1/3 │ 1/3 | +30/-2 │ +30/-2 | +5/-0 │ +5/-0 |", text)
        self.assertIn("| S3 | n/a │ 4/5 |", text)  # the shorter run has no S3
        self.assertIn("2 (1 high) │ 2 (1 high)", text)
        self.assertIn("invalid │ invalid", text)
        self.assertIn("codex 1/0", text)
        self.assertIn("| missed_obligation | 4 │ 4 | 0 │ 0 |", text)
        self.assertIn("100.5 │ 100.5", text)

    def test_mismatch_is_refused_unless_allowed(self):
        a = self.save("a", report("a-1", "conventional"))
        other = report("b-1", "markitect", image="sha256:other")
        other["run"]["outerProvider"] = "claude"
        other["evaluation"]["files"]["reviewerPrompt"] = {"sha256": "changed"}
        b = self.save("b", other)
        code, err = self.main(str(a), str(b))
        self.assertEqual(code, 2)
        for field in ("fairness.imageId", "outerProvider", "evaluation.reviewerPrompt"):
            self.assertIn(field, err)
        self.assertFalse(list(self.root.glob("compare-*.md")))
        out = self.root / "out" / "cmp.md"
        code, err = self.main(str(a), str(b), "--allow-mismatch", "--out", str(out))
        self.assertEqual(code, 0, err)
        text = out.read_text(encoding="utf-8")
        self.assertTrue(text.split("\n", 2)[2].startswith("**Fairness mismatch"))
        self.assertIn("`fairness.imageId`: sha256:img vs sha256:other", text)

    def test_fairness_fields_cover_the_required_set(self):
        fields = compare.fairness_fields(report("x", "conventional"))
        for key in ("case", "outerProvider", "fairness.imageId", "fairness.model", "fairness.effort",
                    "fairness.limits", "fairness.container", "evaluation.groundTruth", "reviewer.codex"):
            self.assertIn(key, fields)
        same = copy.deepcopy(report("y", "markitect"))
        self.assertEqual(compare.mismatches(report("x", "conventional"), same), [])
        same["reviewers"]["claude"]["version"] = "fake-reviewer 1.0"  # a smoke assessment never pairs with a real one
        self.assertEqual([key for key, _a, _b in compare.mismatches(report("x", "conventional"), same)],
                         ["reviewer.claude"])

    def test_missing_assessment(self):
        code, err = self.main(str(self.root / "nope"), str(self.root / "nope2"))
        self.assertEqual(code, 2)
        self.assertIn("no readable assessment report", err)


if __name__ == "__main__":
    unittest.main()
