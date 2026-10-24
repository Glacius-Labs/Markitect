"""The arm-B documentation is the canonical Library model, rendered verbatim."""

import difflib
from pathlib import Path
import unittest

import requirements

PACKET = Path(__file__).resolve().parent / "packet"


class RequirementsTests(unittest.TestCase):
    def test_committed_documents_are_the_current_rendering(self):
        for profile in ("m1", "m2"):
            with self.subTest(profile):
                result = requirements.check(requirements.DEFAULT_MODEL, profile, PACKET / f"requirements-{profile}.md")
                self.assertTrue(result["identical_to_render"])
                self.assertTrue(result["all_found"])
                self.assertEqual(len(result["bindings"]), 22)  # every local Definition of M1

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
