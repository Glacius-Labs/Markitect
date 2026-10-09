"""Provider-free admission and startup-control tests for the supplement."""
from copy import deepcopy
from datetime import datetime, timezone
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import run_scratch_assessment as runner


class ScratchRunnerTests(unittest.TestCase):
    def setUp(self):
        self.plan = json.loads((runner.ROOT / "pilots" / runner.ORDER_ID / "plan.json").read_bytes())
        self.plan["sourcePins"] = {name: runner.digest(runner.ROOT / name) for name in self.plan["sourcePins"]}
        self.exe = Path("C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/9691020b546a15b2/codex.exe")

    def verify_fresh_fixture(self, plan):
        class Clock(datetime):
            @classmethod
            def now(cls, tz=None):
                return cls(2026, 10, 9, 21, 10, tzinfo=timezone.utc)
        # Admission fixtures do not depend on whether the actual single-use
        # execution has already closed. No live ledger is edited or removed.
        with patch.object(runner, "datetime", Clock), patch.object(Path, "exists", return_value=False):
            return runner.verify(plan, self.exe)

    def test_exact_frozen_inputs_admit_without_native_dispatch(self):
        with patch.object(runner.subprocess, "run") as native:
            raw, candidate, requirements, binding = self.verify_fresh_fixture(self.plan)
        native.assert_not_called()
        self.assertEqual(runner.manifest(candidate), self.plan["candidateManifest"])
        self.assertEqual(runner.manifest(requirements), self.plan["initialPublicManifest"])
        self.assertEqual(runner.digest(binding), self.plan["initialPublicBindingSha256"])
        self.assertEqual(json.loads(raw)["id"], runner.ORDER_ID)

    def test_changed_target_inputs_limits_or_sources_decline(self):
        for field in ("candidateManifest", "initialPublicManifest", "limits", "sourcePins"):
            with self.subTest(field=field):
                plan = deepcopy(self.plan)
                plan[field] = {}
                with self.assertRaises(ValueError):
                    self.verify_fresh_fixture(plan)

    def test_reservation_is_single_use_and_original_bytes_retained(self):
        with tempfile.TemporaryDirectory() as folder:
            ledger = Path(folder) / "ledger.json"
            with patch.object(runner, "LEDGER", ledger):
                runner.reserve(self.plan)
                original = ledger.read_bytes()
                with self.assertRaises(FileExistsError):
                    runner.reserve(self.plan)
                self.assertEqual(ledger.read_bytes(), original)

    def test_startup_control_accepts_scratch_and_outside_write_denial(self):
        with tempfile.TemporaryDirectory() as folder:
            audit = Path(folder)
            cwd = audit / "scratch" / "candidate"
            (cwd / ".scratch").mkdir(parents=True)
            def fixture(args, **kwargs):
                self.assertEqual(kwargs["cwd"], cwd)
                self.assertIn(":workspace", args)
                self.assertIn("-NoProfile", args)
                self.assertNotIn("exec", args)
                self.assertIn("readonly-control", args[-1])
                return subprocess.CompletedProcess(args, 0,
                    b"SCRATCH_WRITE_READ_OK\nOUTSIDE_ROOT_WRITE_DENIED\nSCRATCH_PROBE_OK", b"")
            with patch.object(runner.subprocess, "run", side_effect=fixture) as native:
                result = runner.sandbox_check(self.exe, cwd, audit,
                    runner.datetime.now(runner.timezone.utc) + runner.timedelta(minutes=10))
            self.assertEqual(result["state"], "ready")
            self.assertEqual(native.call_count, 1)
            self.assertFalse(result["modelStarted"])

    def test_changed_outside_control_is_blocked_even_with_success_output(self):
        with tempfile.TemporaryDirectory() as folder:
            audit = Path(folder)
            cwd = audit / "scratch" / "candidate"
            (cwd / ".scratch").mkdir(parents=True)
            def fixture(args, **kwargs):
                (audit / "readonly-control" / "sentinel.txt").write_bytes(b"unexpected-write")
                return subprocess.CompletedProcess(args, 0,
                    b"OUTSIDE_ROOT_WRITE_DENIED\nSCRATCH_PROBE_OK", b"")
            with patch.object(runner.subprocess, "run", side_effect=fixture):
                result = runner.sandbox_check(self.exe, cwd, audit,
                    runner.datetime.now(runner.timezone.utc) + runner.timedelta(minutes=10))
            self.assertEqual(result["state"], "blocked")


if __name__ == "__main__":
    unittest.main()
