"""Provider-free controller tests using dynamically pinned fixture adapters."""
from datetime import datetime, timedelta, timezone
import json
from pathlib import Path
import shutil
import sys
import tempfile
import unittest
from unittest.mock import patch
import contextlib
import io

PLAYGROUND = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(PLAYGROUND))

import run_variant_study as study


class VariantStudyTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.playground = self.root / "playground"
        self.playground.mkdir()
        shutil.copytree(PLAYGROUND / "public", self.playground / "public")
        shutil.copy2(PLAYGROUND / "lifecycle.py", self.playground / "lifecycle.py")
        self.package_name = "fixture_adapter_" + self.root.name.replace("-", "_")
        package = self.playground / self.package_name
        package.mkdir()
        (package / "__init__.py").write_text('"""Fixture method package."""\n', encoding="utf-8")
        (package / "adapter.py").write_text(self._adapter_source(), encoding="utf-8")
        self.choice = {"case": "readinglog", "method": "Fixture", "factory": f"{self.package_name}.adapter:FixtureAdapter",
                       "manifestSourceRoots": [self.package_name],
                       "config": {"model": "fixture-model", "effort": "high", "backend": "fixture"}}
        self.plan = {"schema": 1, "id": "offline-controller-fixture", "execution_authorized": True,
                     "profileId": "offline", "jobWallSeconds": 14400, "turnWallSeconds": 5400,
                     "overallWallSeconds": 28800, "startAllowance": 256, "outerTurnsPerTrajectory": 4,
                     "executableVersion": "fixture-native-unavailable", "sourceRoots": [self.package_name],
                     "sourceFiles": [], "trajectories": [self.choice]}

    def _adapter_source(self):
        return '''
import hashlib
import json
from pathlib import Path

def record(config, event):
    path = Path(config["audit"]) / "fixture-calls.json"
    prior = json.loads(path.read_text()) if path.exists() else []
    prior.append(event)
    path.write_text(json.dumps(prior))

class FixtureAdapter:
    def __init__(self, config_path):
        self.config = json.loads(Path(config_path).read_text())
        self.pins = None
        self.counter = 0

    def bind_manifest_source_pins(self, pins):
        self.pins = pins

    def describe(self):
        return {"schema": 1, "id": "fixture", "method": "Fixture", "version": "test",
                "sourcePins": self.pins,
                "capabilities": {"ensure_runtime": True, "start": True, "resume": True,
                                  "status": True, "cancel": True, "close": True},
                "runtime": {"kind": "fake", "owner": "none", "scope": "fixture process-free"}}

    def setup(self, context):
        mode = self.config.get("fixtureMode", "complete")
        state = "blocked" if mode == "setup-blocked" else "ready"
        record(self.config, {"event": "setup", "state": state})
        return {"schema": 1, "state": state, "contextCase": context["case"]}

    def ensure_runtime(self):
        mode = self.config.get("fixtureMode", "complete")
        state = "blocked" if mode == "runtime-blocked" else "ready"
        record(self.config, {"event": "ensure", "state": state})
        return {"schema": 1, "state": state, "runtime": {"fixtureOnly": True}}

    def start(self, prompt):
        mode = self.config.get("fixtureMode", "complete")
        record(self.config, {"event": "start", "prompt": prompt})
        if mode == "start-blocked":
            return {"schema": 1, "state": "blocked", "reason": "offline pre-dispatch fixture"}
        self.counter += 1
        return {"schema": 1, "state": "accepted", "runId": f"fixture-{self.counter}"}

    def resume(self, run_id, prompt):
        record(self.config, {"event": "resume", "runId": run_id, "prompt": prompt})
        self.counter += 1
        return {"schema": 1, "state": "accepted", "runId": f"fixture-{self.counter}"}

    def status(self, run_id):
        mode = self.config.get("fixtureMode", "complete")
        state = "uncertain" if mode == "status-uncertain" else "completed"
        record(self.config, {"event": "status", "runId": run_id, "state": state})
        return {"schema": 1, "state": state, "runId": run_id}

    def cancel(self, run_id):
        record(self.config, {"event": "cancel", "runId": run_id})
        return {"schema": 1, "state": "cancelled", "runId": run_id}

    def close(self):
        record(self.config, {"event": "close"})
        return {"schema": 1, "state": "closed"}
'''

    def run_choice(self, mode="complete"):
        self.choice["config"]["fixtureMode"] = mode
        destination = self.root / ("external-" + mode)
        destination.mkdir()
        original_assess = study.CHECKER.assess
        with patch.object(study, "ROOT", self.playground), \
             patch.object(study.CHECKER, "assess", return_value={"findings": []}), \
             patch("final_assessor.assess", return_value={"state": "completed", "fixtureOnly": True}):
            result = study.execute(self.plan, self.choice, destination, Path(sys.executable),
                                   "fixture-source-sha", datetime.now(timezone.utc) + timedelta(hours=8))
        self.assertIs(study.CHECKER.assess, original_assess)
        audit = destination / "readinglog" / "audit"
        calls_path = audit / "fixture-calls.json"
        calls = json.loads(calls_path.read_text(encoding="utf-8")) if calls_path.exists() else []
        return result, audit, calls

    def test_four_stations_run_through_real_manifest_loader_and_capture(self):
        result, audit, calls = self.run_choice()
        self.assertEqual(result["status"], "trajectory_completed")
        self.assertEqual(result["independentFinalAssessment"]["state"], "completed")
        self.assertEqual([row["station"] for row in result["stations"]], ["S1", "S2", "S3", "S4"])
        self.assertEqual(len(list(audit.glob("snapshot-S*/binding.json"))), 4)
        self.assertTrue((audit / "final-freeze" / "binding.json").is_file())
        self.assertEqual([call["event"] for call in calls].count("start"), 1)
        self.assertEqual([call["event"] for call in calls].count("resume"), 3)
        self.assertEqual([call["event"] for call in calls].count("close"), 1)
        first_resume = next(call for call in calls if call["event"] == "resume")
        self.assertEqual(first_resume["runId"], "fixture-1")

    def test_known_setup_and_runtime_blocks_are_captured_and_closed(self):
        for mode, expected in (("setup-blocked", "setup_blocked"), ("runtime-blocked", "runtime_blocked_before_agent")):
            with self.subTest(mode=mode):
                self.choice["config"].pop("fixtureMode", None)
                result, audit, calls = self.run_choice(mode)
                self.assertEqual(result["status"], expected)
                self.assertTrue((audit / "final-freeze" / "binding.json").is_file())
                self.assertEqual([call["event"] for call in calls].count("close"), 1)
                self.assertEqual([call["event"] for call in calls].count("start"), 0)
                self.assertTrue((audit / "adapter-close-result.json").is_file())

    def test_known_start_block_is_captured_without_status_or_replay(self):
        result, audit, calls = self.run_choice("start-blocked")
        self.assertEqual(result["status"], "runtime_blocked_before_agent")
        self.assertEqual([call["event"] for call in calls].count("start"), 1)
        self.assertEqual([call["event"] for call in calls].count("status"), 0)
        self.assertEqual([call["event"] for call in calls].count("close"), 1)
        self.assertTrue((audit / "final-freeze" / "binding.json").is_file())

    def test_uncertain_status_preserves_worktree_and_skips_close_and_resume(self):
        result, audit, calls = self.run_choice("status-uncertain")
        self.assertEqual(result["status"], "unresolved_runtime")
        self.assertTrue(result["stopStudy"])
        self.assertFalse(list(audit.glob("snapshot-S*")))
        self.assertTrue((audit / "unresolved-runtime.json").is_file())
        self.assertFalse((audit / "adapter-close-result.json").exists())
        self.assertEqual([call["event"] for call in calls].count("start"), 1)
        self.assertEqual([call["event"] for call in calls].count("resume"), 0)
        self.assertEqual([call["event"] for call in calls].count("close"), 0)

    def test_plan_rejects_a_different_overall_wall_limit(self):
        invalid = dict(self.plan, overallWallSeconds=28801)
        with self.assertRaisesRegex(ValueError, "four-hour/eight-hour/256-start"):
            study._validate_plan(invalid)

    def test_main_does_not_start_second_choice_after_unresolved_disposition(self):
        checkout = self.root / "source-checkout"
        checkout.mkdir()
        plan = dict(self.plan)
        plan["trajectories"] = [self.choice, {**self.choice, "case": "roombook"}]
        plan_path = self.root / "plan.json"
        plan_path.write_text(json.dumps(plan), encoding="utf-8")
        destination = self.root / "external-study"
        calls = []

        def stop_after_first(*args):
            calls.append(args[1]["case"])
            return {"case": args[1]["case"], "status": "unresolved_runtime", "stopStudy": True}

        def fake_git(_repo, *args):
            if args == ("status", "--porcelain"):
                return ""
            if args == ("rev-parse", "--show-toplevel"):
                return str(checkout)
            if args == ("rev-parse", "HEAD"):
                return "fixture-clean-source"
            raise AssertionError(args)

        with patch.object(study, "git", side_effect=fake_git), patch.object(study, "execute", side_effect=stop_after_first), patch("delivery_binding.verify", return_value=b"offline-authority-fixture"):
            with contextlib.redirect_stdout(io.StringIO()):
                code = study.main(["--plan", str(plan_path), "--destination", str(destination),
                                   "--executable", sys.executable])
        self.assertEqual(code, 1)
        self.assertEqual(calls, ["readinglog"])
        stored = json.loads((destination / "study-results.json").read_text(encoding="utf-8"))
        self.assertEqual(len(stored["trajectories"]), 1)


if __name__ == "__main__":
    unittest.main()
