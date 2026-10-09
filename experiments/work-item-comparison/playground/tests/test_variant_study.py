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
        self.choice_runs = 0
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
import subprocess

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

    def integrate_station(self):
        repo = Path(self.config["cwd"])
        path = repo / "fixture-work.md"
        with path.open("ab") as stream:
            stream.write(f"station {self.counter + 1}\\n".encode("utf-8"))
        def git(*args, check=True):
            result = subprocess.run(["git", "-C", str(repo), *args], check=False,
                                    capture_output=True, timeout=30)
            if check and result.returncode:
                raise subprocess.CalledProcessError(result.returncode, result.args,
                                                    result.stdout, result.stderr)
            return result
        previous_main = git("rev-parse", "main").stdout.decode().strip()
        git("add", "fixture-work.md")
        checked = git("diff", "--cached", "--check", check=False)
        if checked.returncode:
            raise ValueError(f"fixture cached diff check failed ({checked.returncode}): " +
                             checked.stdout.decode("utf-8", "replace") + checked.stderr.decode("utf-8", "replace") +
                             git("diff", "--cached").stdout.decode("utf-8", "replace"))
        git("commit", "-m", f"Fixture station {self.counter + 1}")
        next_main = git("rev-parse", "HEAD").stdout.decode().strip()
        git("update-ref", "refs/heads/main", next_main, previous_main)

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
        if mode == "workspace-snapshot":
            repo = Path(self.config["cwd"])
            (repo / "README.md").write_text("actor changed the requirements file\\n", encoding="utf-8")
            subprocess.run(["git", "-C", str(repo), "add", "README.md"], check=True,
                           capture_output=True, timeout=30)
            subprocess.run(["git", "-C", str(repo), "commit", "-m", "Actor changed README"], check=True,
                           capture_output=True, timeout=30)
            (Path(self.config["cwd"]) / "fixture-work.md").write_text("station 1\\n", encoding="utf-8")
        elif mode != "no-integration":
            self.integrate_station()
        return {"schema": 1, "state": "accepted", "runId": f"fixture-{self.counter}"}

    def resume(self, run_id, prompt):
        record(self.config, {"event": "resume", "runId": run_id, "prompt": prompt})
        self.counter += 1
        if self.config.get("fixtureMode") == "workspace-snapshot":
            with (Path(self.config["cwd"]) / "fixture-work.md").open("a", encoding="utf-8") as stream:
                stream.write(f"station {self.counter}\\n")
        else:
            self.integrate_station()
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

    def run_choice(self, mode="complete", completion_target="main_merge", findings=None):
        self.choice_runs += 1
        self.choice["config"]["fixtureMode"] = mode
        self.plan["completionTarget"] = completion_target
        destination = self.root / ("external-" + mode + f"-{self.choice_runs}")
        destination.mkdir()
        original_assess = study.CHECKER.assess
        if findings is None:
            findings = []
        final_assessor_call = {}
        def fake_final(*args, **kwargs):
            final_assessor_call["args"] = args
            final_assessor_call["kwargs"] = kwargs
            return {"state": "completed", "fixtureOnly": True,
                    "candidateFilesUnchanged": True,
                    "frozenInitialPublicRequirementsUnchanged": True,
                    "initialPublicWorkspaceBindingUnchanged": True}
        with patch.object(study, "ROOT", self.playground), \
             patch.object(study.CHECKER, "assess", side_effect=lambda *args: {"findings": findings}), \
             patch("final_assessor.assess", side_effect=fake_final):
            result = study.execute(self.plan, self.choice, destination, Path(sys.executable),
                                   "fixture-source-sha", datetime.now(timezone.utc) + timedelta(hours=8))
        self.assertIs(study.CHECKER.assess, original_assess)
        audit = destination / "readinglog" / "audit"
        calls_path = audit / "fixture-calls.json"
        calls = json.loads(calls_path.read_text(encoding="utf-8")) if calls_path.exists() else []
        return result, audit, calls, final_assessor_call

    def test_four_stations_run_through_real_manifest_loader_and_capture(self):
        result, audit, calls, _ = self.run_choice()
        self.assertEqual(result["status"], "trajectory_completed")
        self.assertEqual(result["independentFinalAssessment"]["state"], "completed")
        self.assertEqual([row["station"] for row in result["stations"]], ["S1", "S2", "S3", "S4"])
        self.assertEqual(len(list(audit.glob("snapshot-S*/binding.json"))), 4)
        self.assertEqual(len({row["mainCommit"] for row in result["stations"]}), 4)
        self.assertTrue(all(row["integration"]["mainAdvanced"] and
                            row["integration"]["worktreeAndIndexCleanExceptControllerStationDelta"]
                            for row in result["stations"]))
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
                result, audit, calls, _ = self.run_choice(mode)
                self.assertEqual(result["status"], expected)
                self.assertTrue((audit / "final-freeze" / "binding.json").is_file())
                self.assertEqual([call["event"] for call in calls].count("close"), 1)
                self.assertEqual([call["event"] for call in calls].count("start"), 0)
                self.assertTrue((audit / "adapter-close-result.json").is_file())

    def test_known_start_block_is_captured_without_status_or_replay(self):
        result, audit, calls, _ = self.run_choice("start-blocked")
        self.assertEqual(result["status"], "runtime_blocked_before_agent")
        self.assertEqual([call["event"] for call in calls].count("start"), 1)
        self.assertEqual([call["event"] for call in calls].count("status"), 0)
        self.assertEqual([call["event"] for call in calls].count("close"), 1)
        self.assertTrue((audit / "final-freeze" / "binding.json").is_file())

    def test_completed_turn_without_integrated_main_is_not_advanced(self):
        result, audit, calls, _ = self.run_choice("no-integration")
        self.assertEqual(result["status"], "station_incomplete")
        self.assertTrue(result["stopStudy"])
        self.assertEqual(len(result["stations"]), 1)
        self.assertFalse(result["stations"][0]["integration"]["mainAdvanced"])
        self.assertFalse((audit / "transitions" / "S1-to-S2.json").exists())
        self.assertEqual([call["event"] for call in calls].count("resume"), 0)

    def test_uncertain_status_preserves_worktree_and_skips_close_and_resume(self):
        result, audit, calls, _ = self.run_choice("status-uncertain")
        self.assertEqual(result["status"], "unresolved_runtime")
        self.assertTrue(result["stopStudy"])
        self.assertFalse(list(audit.glob("snapshot-S*")))
        self.assertTrue((audit / "unresolved-runtime.json").is_file())
        self.assertFalse((audit / "adapter-close-result.json").exists())
        self.assertEqual([call["event"] for call in calls].count("start"), 1)
        self.assertEqual([call["event"] for call in calls].count("resume"), 0)
        self.assertEqual([call["event"] for call in calls].count("close"), 0)

    def test_workspace_snapshot_assesses_dirty_workspace_and_advances_only_on_all_pass(self):
        pass_findings = [{"check": "fixture-public-check", "status": "PASS"}]
        result, audit, calls, assessor = self.run_choice(
            "workspace-snapshot", "workspace_snapshot", pass_findings)
        self.assertEqual(result["status"], "trajectory_completed")
        self.assertEqual(result["completionTarget"], "workspace_snapshot")
        self.assertEqual([row["station"] for row in result["stations"]], ["S1", "S2", "S3", "S4"])
        self.assertEqual([call["event"] for call in calls].count("resume"), 3)
        first = json.loads((audit / "snapshot-S1" / "snapshot.json").read_text(encoding="utf-8"))
        candidate = first["assessmentCandidate"]
        self.assertEqual(candidate["kind"], "workspace_snapshot")
        self.assertEqual(candidate["path"], "worktree")
        self.assertTrue(candidate["manifest"])
        self.assertEqual(candidate["manifest"], first["rawWorktreeManifest"])
        candidate_path = audit / "snapshot-S1" / candidate["path"]
        self.assertEqual((candidate_path / "fixture-work.md").read_text(encoding="utf-8"), "station 1\n")
        self.assertNotEqual(candidate["manifest"], first["immutableMain"]["rawGitManifest"])
        initial_binding = json.loads((audit / "initial-public-binding.json").read_text(encoding="utf-8"))
        self.assertEqual((audit / "initial-public" / "README.md").read_bytes(),
                         (self.playground / "public/cases/readinglog/README.md").read_bytes())
        self.assertNotEqual((candidate_path / "README.md").read_text(encoding="utf-8"),
                            (audit / "initial-public" / "README.md").read_text(encoding="utf-8"))
        self.assertEqual(initial_binding["fileManifest"]["README.md"],
                         study.digest(audit / "initial-public" / "README.md"))
        self.assertEqual(assessor["kwargs"]["completion_target"], "workspace_snapshot")
        self.assertEqual(Path(assessor["args"][3]).resolve(), (audit / "final-freeze" / "worktree").resolve())

        result, audit, calls, _ = self.run_choice(
            "workspace-snapshot", "workspace_snapshot", [{"check": "missing", "status": "NOT RUN"}])
        self.assertEqual(result["status"], "station_incomplete")
        self.assertTrue(result["stopStudy"])
        self.assertEqual([call["event"] for call in calls].count("resume"), 0)
        self.assertFalse((audit / "transitions" / "S1-to-S2.json").exists())

    def test_workspace_snapshot_does_not_advance_if_public_check_changes_captured_candidate(self):
        self.choice["config"]["fixtureMode"] = "workspace-snapshot"
        destination = self.root / "external-mutated-check"
        destination.mkdir()
        with patch.object(study, "ROOT", self.playground), \
             patch.object(study.CHECKER, "assess", side_effect=self._mutating_pass):
            self.plan["completionTarget"] = "workspace_snapshot"
            self.choice["config"]["fixtureMode"] = "workspace-snapshot"
            result = study.execute(self.plan, self.choice, destination, Path(sys.executable),
                                   "fixture-source-sha", datetime.now(timezone.utc) + timedelta(hours=8))
        audit = destination / "readinglog" / "audit"
        self.assertEqual(result["status"], "station_incomplete")
        self.assertTrue(result["stopStudy"])
        self.assertFalse(result["stations"][0]["assessmentCandidateUnchangedAfterPublicChecks"])
        self.assertFalse((audit / "transitions" / "S1-to-S2.json").exists())

    @staticmethod
    def _mutating_pass(candidate, case, station):
        (Path(candidate) / "fixture-work.md").write_text("changed during public check\n", encoding="utf-8")
        return {"findings": [{"check": "fixture", "status": "PASS"}]}

    def test_plan_rejects_a_different_overall_wall_limit(self):
        invalid = dict(self.plan, overallWallSeconds=28801)
        with self.assertRaisesRegex(ValueError, "four-hour/eight-hour/256-start"):
            study._validate_plan(invalid)

    def test_workspace_snapshot_plan_binds_single_case_clock_and_assessment_reserves(self):
        snapshot_plan = dict(self.plan, completionTarget="workspace_snapshot", overallWallSeconds=14400,
                             implementationSeconds=10800, freshFinalSeconds=2700,
                             captureAssessmentReserveSeconds=900, absoluteEndUtc="2026-10-10T04:31:08Z",
                             lastStartUtc="2026-10-10T00:31:08Z")
        self.assertIs(study._validate_plan(snapshot_plan), snapshot_plan)
        with self.assertRaisesRegex(ValueError, "3h/45m/15m"):
            study._validate_plan(dict(snapshot_plan, freshFinalSeconds=2701))

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

    def test_main_stops_after_known_runtime_failure_or_cancellation(self):
        for terminal_status, choice_count in (("runtime_failed", 1), ("runtime_failed", 2), ("cancelled", 2)):
            terminal_key = terminal_status + str(choice_count)
            with self.subTest(status=terminal_status, choices=choice_count):
                checkout = self.root / ("checkout-" + terminal_key)
                checkout.mkdir()
                plan = dict(self.plan)
                plan["trajectories"] = [self.choice, {**self.choice, "case": "roombook"}][:choice_count]
                plan_path = self.root / (terminal_key + "-plan.json")
                plan_path.write_text(json.dumps(plan), encoding="utf-8")
                destination = self.root / ("destination-" + terminal_key)
                calls = []

                def stop_after_known(*args):
                    calls.append(args[1]["case"])
                    return {"case": args[1]["case"], "status": terminal_status}

                def fake_git(_repo, *args):
                    if args == ("status", "--porcelain"):
                        return ""
                    if args == ("rev-parse", "--show-toplevel"):
                        return str(checkout)
                    if args == ("rev-parse", "HEAD"):
                        return "fixture-clean-source"
                    raise AssertionError(args)

                with patch.object(study, "git", side_effect=fake_git), \
                     patch.object(study, "execute", side_effect=stop_after_known), \
                     patch("delivery_binding.verify", return_value=b"offline-authority-fixture"):
                    with contextlib.redirect_stdout(io.StringIO()):
                        code = study.main(["--plan", str(plan_path), "--destination", str(destination),
                                           "--executable", sys.executable])
                self.assertEqual(code, 1)
                self.assertEqual(calls, ["readinglog"])
                stored = json.loads((destination / "study-results.json").read_bytes())
                self.assertEqual(stored["status"], "stopped")


if __name__ == "__main__":
    unittest.main()
