"""The arm-B documentation is the canonical Library model, rendered verbatim."""

import difflib
from pathlib import Path
import unittest

import requirements

PACKET = Path(__file__).resolve().parent / "packet"


class RequirementsTests(unittest.TestCase):
    def test_committed_documents_are_the_current_rendering(self):
        # Every local Definition: M1's 22, then shortLoanPeriodDays, maxRenewals and renew, minus set-member-status.
        for profile, count in (("m1", 22), ("m2", 22), ("m3", 23), ("m4", 25), ("m5", 24)):
            with self.subTest(profile):
                result = requirements.check(requirements.DEFAULT_MODEL, profile, PACKET / f"requirements-{profile}.md")
                self.assertTrue(result["identical_to_render"])
                self.assertTrue(result["all_found"])
                self.assertEqual(len(result["bindings"]), count)

    def test_phase_b_overlays_change_what_the_addendum_states(self):
        docs = {p: (PACKET / f"requirements-{p}.md").read_text(encoding="utf-8") for p in ("m2", "m3", "m4", "m5")}
        self.assertNotIn("loanClass", docs["m2"])
        self.assertIn("| Short loan period in calendar days (`shortLoanPeriodDays`) | 7 |", docs["m3"])
        self.assertIn("| `renew` | `loanId`, `renewalDate` | `loan` |", docs["m4"])
        self.assertIn("| Maximum renewals per loan (`maxRenewals`) | 1 |", docs["m4"])
        self.assertIn("each [renewal](#renew) replaces the loan's due date", docs["m4"])
        self.assertIn("reservations, renewals", docs["m3"])
        self.assertNotIn("reservations, renewals", docs["m4"])
        self.assertIn("### Set member status", docs["m4"])
        self.assertNotIn("### Set member status", docs["m5"])
        self.assertNotIn("| `setMemberStatus` |", docs["m5"])
        self.assertIn("request names an unknown operation", docs["m5"])
        # The overdue Definition deliberately keeps its wording from M2 to M4.
        overdue = [d.split("### Assess overdue loans")[1] for d in (docs["m2"], docs["m3"], docs["m4"])]
        self.assertEqual(len(set(overdue)), 1)

    def test_materialize_writes_the_complete_profile(self):
        import tempfile
        with tempfile.TemporaryDirectory() as tmp:
            files = requirements.materialize(requirements.DEFAULT_MODEL, "m5", Path(tmp) / "m5")
            self.assertIn("model.yaml", files)
            self.assertIn("model/lending/renew.yaml", files)
            self.assertNotIn("model/members/set-member-status.yaml", files)
            self.assertEqual(len([f for f in files if f.startswith("model/")]), 24)

    def test_m2_differs_only_in_the_limit_value(self):
        m1 = (PACKET / "requirements-m1.md").read_text(encoding="utf-8").splitlines()
        m2 = (PACKET / "requirements-m2.md").read_text(encoding="utf-8").splitlines()
        changed = [line for line in difflib.unified_diff(m1, m2, lineterm="", n=0)
                   if line[:1] in "+-" and not line.startswith(("+++", "---"))]
        self.assertEqual(len(changed), 2)
        self.assertTrue(changed[0].startswith("-| Maximum open loans (`maxOpenLoans`) | 5 |"))
        self.assertEqual(changed[0][1:].replace("| 5 |", "| 3 |", 1), changed[1][1:])

    def test_every_body_link_resolves_and_no_model_vocabulary_remains(self):
        text = (PACKET / "requirements-m1.md").read_text(encoding="utf-8")
        self.assertNotIn("[[", text)
        anchors = {requirements.slug(title) for _, title, _ in requirements.SECTIONS}
        for target in requirements.re.findall(r"\]\(#([a-z0-9-]+)\)", text):
            self.assertIn(target, anchors)

    def test_parser_refuses_unsupported_yaml(self):
        for text in ("id: x\nkind: K\nbody: |\n  text\n", "id: x\nkind: K\nbody: >-\n  text\n    more\n",
                     "id: x\nkind: K\nbody: \"quoted\"\n", "id: x\nkind: K\n"):
            with self.subTest(text):
                with self.assertRaises(requirements.ModelError):
                    requirements.parse_definition(text)

    def test_substitutions_fail_closed_when_the_model_wording_changes(self):
        definition = requirements.parse_definition("id: realization\nkind: Software.Rule\nbody: >-\n  Agree with the code.\nspec: {}\n")
        definition.update(package="library", imports={})
        with self.assertRaises(requirements.ModelError):
            requirements.render_body(definition, "library/realization")


if __name__ == "__main__":
    unittest.main()
