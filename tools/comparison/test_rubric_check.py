"""A critical finding counts only with a real citation and a verbatim requirement."""

from pathlib import Path
import tempfile
import unittest

import rubric_check


class RubricCheckTests(unittest.TestCase):
    def test_citations_requirements_and_absence_claims_are_verified(self):
        with tempfile.TemporaryDirectory() as tmp:
            token = Path(tmp) / "tok123"
            readme = token / "candidate" / "realization" / "README.md"
            readme.parent.mkdir(parents=True)
            readme.write_text("# Library\n\nA member may hold five open loans.\n", encoding="utf-8")
            (token / "candidate" / "realization" / "go.mod").write_text("module library\n", encoding="utf-8")
            (token / "candidate" / "realization" / "cmd" / "library").mkdir(parents=True)
            requirements = Path(tmp) / "requirements.md"
            requirements.write_text("| Maximum open loans | 3 |\n\nA loan due on that date is\nnot yet overdue.\n", encoding="utf-8")
            review = {"token": "tok123", "R1a": False, "R1b": False, "R1c": {"run": True, "test": False},
                      "R2": [{"cite": "realization/README.md:3", "quote": "may hold five open loans", "requirement": "| Maximum open loans | 3 |"},
                             {"cite": "realization/MISSING.md:3", "quote": "five open loans", "requirement": "| Maximum open loans | 3 |"},
                             {"cite": "realization/README.md:3", "quote": "five open loans", "requirement": "A loan due on that date is not yet overdue."},
                             {"cite": "realization/README.md:3", "quote": "five open loans", "requirement": "Loans may never be returned."}],
                      "R5": [{"path": "x.go", "class": "unjustified", "cite": None}, {"path": "y.go", "class": "optional"}]}
            result = rubric_check.check(token, requirements, review)
            verdicts = [(r["item"], r["valid"]) for r in result["criticals"]]
            self.assertEqual(verdicts, [("R1a", False), ("R1b", True), ("R1c", True), ("R2", True), ("R2", False),
                                        ("R2", True), ("R2", False), ("R5", False)])
            self.assertEqual(result["valid_criticals"], 4)


if __name__ == "__main__":
    unittest.main()
