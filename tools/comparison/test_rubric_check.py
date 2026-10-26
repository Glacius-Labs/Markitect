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

    def test_the_entry_point_is_the_programs_own(self):
        with tempfile.TemporaryDirectory() as tmp:
            token = Path(tmp) / "tok789"
            (token / "candidate" / "realization" / "cmd" / "makerspace").mkdir(parents=True)
            (token / "candidate" / "realization" / "go.mod").write_text("module makerspace\n", encoding="utf-8")
            requirements = Path(tmp) / "requirements.md"
            requirements.write_text("The program's main package lives at `realization/cmd/makerspace`.\n", encoding="utf-8")
            review = {"token": "tok789", "R1a": False}
            # A claim that the entry point is missing is true for the Library's path and false for Makerspace's.
            self.assertTrue(rubric_check.check(token, requirements, review)["criticals"][0]["valid"])
            self.assertFalse(rubric_check.check(token, requirements, review, entry="realization/cmd/makerspace")["criticals"][0]["valid"])

    def test_removals_are_cited_in_the_start_tree_or_the_diff(self):
        with tempfile.TemporaryDirectory() as tmp:
            token = Path(tmp) / "tok456"
            (token / "candidate" / "realization").mkdir(parents=True)
            old = token / "start" / "realization" / "loan_test.go"
            old.parent.mkdir(parents=True)
            old.write_text("package lending\n\nfunc TestReturnedLoanExcluded(t *testing.T) {}\n", encoding="utf-8")
            (token / "diff.patch").write_text(
                "--- a/realization/loan_test.go\n+++ /dev/null\n@@ -1,3 +0,0 @@\n-package lending\n-\n-func TestReturnedLoanExcluded(t *testing.T) {}\n"
                "--- a/realization/README.md\n+++ b/realization/README.md\n@@ -1 +1 @@\n-Old wording.\n+New wording.\n", encoding="utf-8")
            requirements = Path(tmp) / "requirements.md"
            requirements.write_text("Returned loans are excluded.\n", encoding="utf-8")
            review = {"token": "tok456", "R5": [
                {"path": "realization/loan_test.go", "class": "unjustified", "cite": "realization/loan_test.go:3", "quote": "func TestReturnedLoanExcluded"},
                {"path": "realization/loan_test.go", "class": "unjustified", "cite": "realization/gone.go:1", "quote": "func TestReturnedLoanExcluded"},
                {"path": "realization/loan_test.go", "class": "unjustified", "cite": "realization/loan_test.go:3", "quote": "never written anywhere"},
                {"path": "realization/README.md", "class": "optional", "why": "rewording"},
                {"path": "realization/other.go", "class": "optional", "why": "not in the diff"}]}
            plain = rubric_check.check(token, requirements, review)
            self.assertEqual([r["valid"] for r in plain["criticals"]], [True, True, False])  # start tree, diff, nowhere
            s3 = rubric_check.check(token, requirements, review, s3=True)
            self.assertEqual([r["valid"] for r in s3["criticals"]], [True, True, False, True, False])


if __name__ == "__main__":
    unittest.main()
