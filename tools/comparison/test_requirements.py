"""The arm-B documentation is the canonical Library model, rendered verbatim."""

import difflib
import hashlib
import importlib.util
from pathlib import Path
import unittest

import requirements

PACKET = Path(__file__).resolve().parent / "packet"
PHASE_C = Path(__file__).resolve().parent / "phase-c"


def load_phase_c_renderer():
    spec = importlib.util.spec_from_file_location("phase_c_render", PHASE_C / "spec" / "render.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


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
        self.assertNotIn("setMemberStatus", docs["m5"])
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
        for profile in requirements.PROFILES:
            with self.subTest(profile):
                text = (PACKET / f"requirements-{profile}.md").read_text(encoding="utf-8")
                self.assertNotIn("[[", text)
                anchors = {requirements.slug(line.lstrip("#").strip()) for line in text.splitlines() if line.startswith("#")}
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

    def test_document_drops_empty_groups_and_refuses_duplicate_anchors(self):
        files = {"model/area/sub/x.yaml": b"id: x\nkind: Software.Rule\nbody: |-\n  Text of x.\nspec: {}\n"}
        sections = [(2, "Area", []), (3, "Sub", ["area/sub/x"]), (2, "Empty group", []), (3, "Gone", ["area/sub/y"])]
        doc = requirements.Document("T", "Intro.", sections)
        definitions = requirements.load_definitions(files, doc)
        self.assertEqual(requirements.present_sections(definitions, doc), [(2, "Area", []), (3, "Sub", ["area/sub/x"])])
        self.assertIn("## Area\n\n### Sub\n\nText of x.\n", requirements.render_document(definitions, doc))
        nested = requirements.Document("T", "Intro.", [(2, "Top", []), (3, "Middle", []), (4, "Leaf", ["area/sub/x"]),
                                                       (2, "Hollow", []), (3, "Empty middle", []), (4, "Gone", ["area/sub/y"])])
        self.assertEqual(requirements.present_sections(definitions, nested),
                         [(2, "Top", []), (3, "Middle", []), (4, "Leaf", ["area/sub/x"])])
        twice = requirements.Document("T", "Intro.", [(2, "Sub", []), (3, "Sub", ["area/sub/x"])])
        with self.assertRaises(requirements.ModelError):
            requirements.render_document(definitions, twice)


class PhaseCTests(unittest.TestCase):
    """The single-agent Makerspace documents are the model snapshots, rendered verbatim."""

    COUNTS = {"w0": 75, "c1": 75, "c2": 77, "c3": 78, "c4": 78, "c5": 79, "c6": 79, "c7": 79, "c8": 79, "c9": 79}

    @classmethod
    def setUpClass(cls):
        cls.render = load_phase_c_renderer()

    def test_committed_documents_are_the_current_rendering(self):
        self.assertEqual(tuple(self.COUNTS), self.render.STEPS)
        for step, count in self.COUNTS.items():
            with self.subTest(step):
                result = self.render.check(step)
                self.assertTrue(result["identical_to_render"])
                self.assertTrue(result["all_found"])
                self.assertEqual(len(result["bindings"]), count)

    def test_snapshot_manifests_cover_every_file(self):
        for step in self.COUNTS:
            with self.subTest(step):
                root = PHASE_C / "model" / step
                listed = {}
                for line in (root / "files.sha256").read_text(encoding="utf-8").splitlines():
                    digest, _, name = line.partition("  ")
                    listed[name] = digest
                files = sorted(p.relative_to(root).as_posix() for p in root.rglob("*") if p.is_file() and p.name != "files.sha256")
                self.assertEqual(sorted(listed), files)
                for name in files:
                    self.assertEqual(listed[name], hashlib.sha256((root / name).read_bytes()).hexdigest(), name)

    def test_steps_change_what_their_texts_state(self):
        docs = {s: (PHASE_C / "spec" / f"requirements-{s}.md").read_text(encoding="utf-8") for s in self.COUNTS}
        self.assertEqual(docs["c8"], docs["c9"])
        self.assertEqual((PHASE_C / "model" / "c8" / "files.sha256").read_bytes(), (PHASE_C / "model" / "c9" / "files.sha256").read_bytes())
        self.assertNotIn("actorRole", docs["w0"])
        self.assertIn("actorRole", docs["c1"])
        self.assertNotIn("lockMachine", docs["c1"])
        self.assertIn("#### `lockMachine`", docs["c2"])
        self.assertIn("| 25 or more | 12 |", docs["c3"])
        self.assertIn("| Operation | Area | Member | Staff | Admin | Steward |", docs["c4"])
        self.assertIn("account_overdue", docs["c5"])
        self.assertIn("expiresOn", docs["c5"])
        self.assertNotIn("expiresOn", docs["c6"])
        self.assertIn("\n### Payments and credit\n", docs["c6"])
        self.assertIn("\n### Invoices\n", docs["c7"])
        self.assertNotIn("\n### Payments and credit\n", docs["c7"])
        self.assertIn("\n#### Payments and credit\n", docs["c7"])
        self.assertIn("| Sunday | 12:00 | 16:00 | 240 |", docs["c8"])
        self.assertFalse(any(p.parts[-2] == "payments" for p in (PHASE_C / "model" / "c7" / "model").rglob("*.yaml")))

    def test_every_link_resolves_and_no_model_vocabulary_remains(self):
        for step in self.COUNTS:
            with self.subTest(step):
                text = (PHASE_C / "spec" / f"requirements-{step}.md").read_text(encoding="utf-8")
                self.assertNotIn("[[", text)
                anchors = {requirements.slug(line.lstrip("#").strip()) for line in text.splitlines() if line.startswith("#")}
                for target in requirements.re.findall(r"\]\(#([a-z0-9-]+)\)", text):
                    self.assertIn(target, anchors)


if __name__ == "__main__":
    unittest.main()
