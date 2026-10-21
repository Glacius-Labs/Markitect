"""Provider-free checks of the one realistic R1 input and its independent controls.

    python -B -m unittest discover -s experiments/apply-trials -p test_workshop.py
"""
import sys
import tempfile
import unittest
from pathlib import Path

import assess
import freeze
import oracle
import plan_packet
import selfcheck
from cases import CASES, W1_DISCOUNT, W2_TIER, W3_LOW_STOCK
from materialize import materialize, packet_files

PYTHON = [sys.executable]


class Workshop(unittest.TestCase):
    def test_reference_and_real_cross_area_workflow(self):
        with tempfile.TemporaryDirectory() as folder:
            realization = selfcheck.build("workshop", folder)
            result = oracle.assess("workshop", "w1", realization, PYTHON)
            self.assertTrue(result["passed"], result)
            self.assertEqual(result["summary"]["total"], 22)
            passed, output = assess.run_tests(realization, "workshop", PYTHON)
            self.assertTrue(passed, output)

    def test_fault_discrimination_and_own_test_signals(self):
        for fault in (W1_DISCOUNT, W2_TIER, W3_LOW_STOCK):
            with self.subTest(fault=fault.id), tempfile.TemporaryDirectory() as folder:
                realization = selfcheck.build("workshop", folder, faults=[fault])
                failed, _ = selfcheck.failing("workshop", "w1", realization, PYTHON)
                self.assertEqual(failed, set(fault.detected_by))
                passed, _ = assess.run_tests(realization, "workshop", PYTHON)
                self.assertEqual(not passed, fault.test_signal)

    def test_citations_counts_and_delivery_ownership(self):
        root = Path(__file__).parent / "workshop/snapshots/w1/model"
        definitions = {p.relative_to(root).with_suffix("").as_posix() for p in root.rglob("*.yaml")}
        self.assertEqual(len(definitions), 26)
        self.assertEqual(len({p.rsplit("/", 1)[0] for p in definitions}), 6)
        for check in oracle.load_checks("workshop").checks:
            self.assertTrue(set(check["definitions"]) <= definitions, check)
        self.assertEqual(assess.owner_of("workshop", "realization/billing/billing.py"), "shop/billing")
        self.assertEqual(assess.owner_of("workshop", "realization/app.py"), "shop")
        self.assertEqual(assess.owner_of("workshop", ".markitect/registry/shop/billing/registry.yaml"), "shop/billing")
        self.assertEqual(assess.owner_of("hall", "realization/rooms/rooms.py"), "hall/rooms")

    def test_model_only_packet_is_neutral_and_has_no_answers(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            direct = Path(freeze.scenario_packet("R1", root / "direct", "direct")["repo"])
            staged = Path(freeze.scenario_packet("R1", root / "staged", "staged")["repo"])
            self.assertFalse((direct / "realization").exists())
            self.assertFalse((direct / ".markitect/registry").exists())
            for path in packet_files(direct):
                a, b = (direct / path).read_bytes(), (staged / path).read_bytes()
                if path == ".markitect/execution.yaml":
                    self.assertEqual(a.replace(b"strategy: direct", b"strategy: staged"), b)
                else:
                    self.assertEqual(a, b, path)
            plan = plan_packet.build_from_packet(direct, root / "plan")
            self.assertFalse((root / "plan/.markitect").exists())
            text = (root / "plan/plan.md").read_text(encoding="utf-8")
            self.assertEqual(text.count("- Source: `.markitect/model/"), 26)
            self.assertTrue(plan["planSha256"])
            for case in ("W-N", "W-M"):
                materialize(case, root / case)


if __name__ == "__main__":
    unittest.main()
