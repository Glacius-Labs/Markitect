"""Offline regressions for the R3 positive-prerequisite launch boundary."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location("r3_driver", ROOT / "run-native-integration-r3.py")
driver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(driver)


class R3CheckpointTests(unittest.TestCase):
    def test_zero_exit_alone_does_not_authorize_continuation(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for action, status in driver.classic_integration.EXPECTED_STATUSES.items():
                for problem in (None, "wrong_status", "timeout", "stop", "changed_receipt", "nonzero"):
                    with self.subTest(action=action, problem=problem):
                        report = root / "stdout.json"
                        report.write_text(json.dumps({"status": status if problem != "wrong_status" else "blocked"}))
                        receipt = {"returnCode": 2 if problem == "nonzero" else 0,
                                   "stopReason": "wall_deadline" if problem == "stop" else None,
                                   "timedOut": problem == "timeout"}
                        process = root / "process.json"
                        process.write_text(json.dumps(receipt))
                        capture = {"action": action, "returnCode": 0, "stdoutPath": str(report),
                                   "processPath": str(process),
                                   "processSha256": driver.dispatch.digest(process.read_bytes())}
                        if problem == "changed_receipt":
                            process.write_text(process.read_text() + " ")
                        if problem is None:
                            driver.require_positive_classic_checkpoint(capture)
                        else:
                            with self.assertRaises(ValueError):
                                driver.require_positive_classic_checkpoint(capture)

    def test_audit_findings_or_next_steps_stop_before_replay(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            process = root / "process.json"
            process.write_text(json.dumps({"returnCode": 0, "stopReason": None, "timedOut": False}))
            report = root / "stdout.json"
            capture = {"action": "audit", "returnCode": 0, "stdoutPath": str(report),
                       "processPath": str(process), "processSha256": driver.dispatch.digest(process.read_bytes())}
            for field in ("findings", "nextSteps"):
                report.write_text(json.dumps({"status": "complete", field: ["unresolved"]}))
                with self.assertRaisesRegex(ValueError, "Audit has findings"):
                    driver.require_positive_classic_checkpoint(capture)
