"""Focused source-only tests for change_smoke safety boundaries."""
from __future__ import annotations

import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time
import unittest

sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parent))
import change_smoke as change


def fixture_repo(root: Path) -> Path:
    repo = root / "repo"
    repo.mkdir()
    use_case = repo / change.USE_CASE_PATH
    program = repo / change.PROGRAM_PATH
    use_case.parent.mkdir(parents=True)
    program.parent.mkdir(parents=True)
    use_case.write_bytes(b"purpose: >\n" + change.BASELINE_PURPOSE.encode() + b"spec: {}\n")
    program.write_bytes(change.BASELINE_PROBE.encode() + b"// unchanged source\n")
    subprocess.run(["git", "init", "-q", str(repo)], check=True, capture_output=True)
    subprocess.run(["git", "-C", str(repo), "config", "user.name", "Smoke Test"], check=True, capture_output=True)
    subprocess.run(["git", "-C", str(repo), "config", "user.email", "smoke@example.invalid"], check=True, capture_output=True)
    subprocess.run(["git", "-C", str(repo), "add", "--all"], check=True, capture_output=True)
    subprocess.run(["git", "-C", str(repo), "commit", "-qm", "baseline"], check=True, capture_output=True)
    return repo


class ChangeSmokeTests(unittest.TestCase):
    def test_unknown_source_marker_is_refused_without_source_writes(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            repo = fixture_repo(root)
            path = repo / change.USE_CASE_PATH
            original = path.read_bytes().replace(change.BASELINE_PURPOSE.encode(), b"purpose: unknown\n")
            path.write_bytes(original)
            program_path = repo / change.PROGRAM_PATH
            original_program = program_path.read_bytes()
            receipt = {"outputRoot": str(root / "receipt")}
            Path(receipt["outputRoot"]).mkdir()
            with self.assertRaises(change.ChangeSmokeFailure):
                change.accept_exact_intent_change(repo, True, receipt)
            self.assertEqual(original, path.read_bytes())
            self.assertEqual(original_program, program_path.read_bytes())

    def test_real_example_source_has_exact_edit_markers(self) -> None:
        source = Path(__file__).resolve().parents[3]
        purpose, program = change.validate_exact_source(source)
        self.assertEqual(1, purpose.count(change.BASELINE_PURPOSE.encode()))
        self.assertEqual(1, program.count(change.BASELINE_PROBE.encode()))
        self.assertEqual(1, purpose.replace(change.BASELINE_PURPOSE.encode(), change.MINIMUM_TWO_PURPOSE.encode(), 1).count(change.MINIMUM_TWO_PURPOSE.encode()))
        self.assertEqual(1, program.replace(change.BASELINE_PROBE.encode(), change.MINIMUM_TWO_PROBE.encode(), 1).count(change.MINIMUM_TWO_PROBE.encode()))

    def test_exact_owner_selected_change_is_two_paths_and_changes_oracle(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            repo = fixture_repo(root)
            output = root / "output"
            output.mkdir()
            receipt = {"outputRoot": str(output)}
            revision = change.accept_exact_intent_change(repo, True, receipt)
            self.assertEqual(40, len(revision))
            purpose = (repo / change.USE_CASE_PATH).read_bytes()
            program = (repo / change.PROGRAM_PATH).read_bytes()
            self.assertIn(change.MINIMUM_TWO_PURPOSE.encode(), purpose)
            self.assertNotIn(change.BASELINE_PURPOSE.encode(), purpose)
            self.assertIn(change.MINIMUM_TWO_PROBE.encode(), program)
            self.assertNotIn(change.BASELINE_PROBE.encode(), program)
            changed_paths = subprocess.run(
                ["git", "-C", str(repo), "diff", "--name-only", "HEAD^", "HEAD"],
                check=True, capture_output=True, text=True,
            ).stdout.splitlines()
            self.assertEqual(sorted([change.PROGRAM_PATH, change.USE_CASE_PATH]), sorted(changed_paths))
            self.assertEqual(set(changed_paths), set(receipt["intentChange"]["paths"]))
            self.assertTrue((output / "accepted-intent.diff").is_file())

    def test_unselected_intent_change_writes_preview_only(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            repo = fixture_repo(root)
            before = {path: (repo / path).read_bytes() for path in (change.USE_CASE_PATH, change.PROGRAM_PATH)}
            output = root / "output"
            output.mkdir()
            receipt = {"outputRoot": str(output)}
            with self.assertRaises(change.ChangeSmokeFailure):
                change.accept_exact_intent_change(repo, False, receipt)
            self.assertEqual(before, {path: (repo / path).read_bytes() for path in before})
            self.assertTrue((output / "accepted-intent.diff").read_bytes())

    def test_native_start_quota_refuses_before_launch(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            output = Path(temp)
            budget = change.ProcessBudget(output, time.monotonic() + 10)
            budget.native_mutating_starts = change.MAX_NATIVE_MUTATING_STARTS
            with self.assertRaises(change.ChangeSmokeFailure):
                budget([sys.executable, "canonical", "--action", "controller-apply"], output)
            self.assertEqual([], budget.commands)
            self.assertEqual(0, budget.max_active)

    def test_timeout_preserves_partial_raw_output_and_marks_timeout(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            output = Path(temp)
            budget = change.ProcessBudget(output, time.monotonic() + 5.3)
            with self.assertRaises(change.ChangeSmokeFailure):
                budget([sys.executable, "-c", "import time; print('started', flush=True); time.sleep(10)"], output)
            command = budget.commands[-1]
            self.assertTrue(command["timedOut"])
            self.assertEqual(124, command["exitCode"])
            self.assertIn(b"started", Path(command["stdoutPath"]).read_bytes())
            self.assertTrue(Path(command["stderrPath"]).is_file())


if __name__ == "__main__":
    unittest.main()
