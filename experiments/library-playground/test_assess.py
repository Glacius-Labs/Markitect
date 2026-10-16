"""Assessment plumbing against tiny transcript fixtures, never a library."""

import copy
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

import assess
from driver import Limits
from oracle import build_cases, witness_case
from test_oracle import expected_transcript


CASE = {"id": "controlled-missing-member", "families": ["E09"], "steps": [
    {"request": {"id": "q", "op": "inspectMember", "args": {"memberId": "absent"}},
     "expect": {"id": "q", "status": "refused", "reason": {"$nonempty": True}}}]}


class AssessmentTests(unittest.TestCase):
    def fixture(self, directory, response):
        path = Path(directory) / "protocol-fixture.py"
        path.write_text('import sys\nfor line in sys.stdin:\n print(' + repr(json.dumps(response)) + ',flush=True)\n', encoding="utf-8")
        return [sys.executable, "-I", str(path)]

    def manifest(self, directory, command):
        return {"experiment_id": "synthetic-test", "artifact_id": "controlled-protocol-fixture",
                "model_digest": "not-a-compiled-model-test-only", "source_tree": "not-an-Apply-test-only",
                "profile": "m1", "command": command, "cwd": directory,
                "limits": {"request_seconds": 1, "process_seconds": 4, "output_bytes": 4096, "request_bytes": 4096},
                "program_sha256": {command[0]: assess.sha256(command[0]), command[2]: assess.sha256(command[2])},
                "protected_sha256": assess.protected_hashes()}

    def test_bound_report_trace_and_counts(self):
        with tempfile.TemporaryDirectory() as directory:
            command = self.fixture(directory, {"id": "q", "status": "refused", "reason": "Unknown"})
            manifest = self.manifest(directory, command)
            with patch.object(assess, "build_cases", return_value=[CASE]), patch.object(
                    assess, "discrimination", return_value={"expected_mismatch_confirmed": True}):
                report = assess.assess(manifest)
            self.assertTrue(report["passed"], report)
            self.assertEqual(report["executed_cases"], 1)
            self.assertEqual(report["missing_steps"], 0)
            self.assertGreater(report["assertions"], 0)
            self.assertEqual(report["cases"][0]["trace"]["exit_code"], 0)
            self.assertEqual(report["manifest_sha256"], assess.data_hash(manifest))
            self.assertEqual(report["apply_evidence"], "not-run")

    def test_wrong_response_and_skipped_steps_fail(self):
        with tempfile.TemporaryDirectory() as directory:
            command = self.fixture(directory, {"id": "wrong-correlation", "status": "ok", "result": {}})
            case = copy.deepcopy(CASE)
            case["steps"] *= 2
            report = assess.run_case(case, command, directory, Limits())
            self.assertFalse(report["passed"])
            self.assertEqual(report["executed_steps"], 1)
            self.assertEqual(report["planned_steps"], 2)
            self.assertTrue(report["errors"])

    def test_integrity_rejects_before_execution(self):
        with tempfile.TemporaryDirectory() as directory:
            command = self.fixture(directory, {})
            manifest = self.manifest(directory, command)
            for field in ("protected_sha256", "program_sha256"):
                wrong = copy.deepcopy(manifest)
                wrong[field] = {}
                report = assess.assess(wrong)
                self.assertFalse(report["passed"])
                self.assertEqual(report["executed_cases"], 0)
            Path(command[2]).write_text("changed", encoding="utf-8")
            self.assertFalse(assess.assess(manifest)["passed"])

    def test_integrity_rejects_after_execution(self):
        with tempfile.TemporaryDirectory() as directory:
            command = self.fixture(directory, {"id": "q", "status": "refused", "reason": "Unknown"})
            manifest = self.manifest(directory, command)
            actual = assess.protected_hashes()
            with patch.object(assess, "build_cases", return_value=[CASE]), patch.object(
                    assess, "discrimination", return_value={"expected_mismatch_confirmed": True}), patch.object(
                    assess, "protected_hashes", side_effect=[actual, {"changed": "bytes"}]):
                report = assess.assess(manifest)
            self.assertFalse(report["passed"])
            self.assertEqual(report["executed_cases"], 1)
            self.assertIn("binding mismatch", report["errors"][0])

    def transcript_fixture(self, directory, rows):
        path = Path(directory) / "replay.py"
        path.write_text('import sys\nrows = ' + repr([json.dumps(row) for row in rows]) +
                        '\nfor row in rows:\n if not sys.stdin.readline(): break\n print(row,flush=True)\n', encoding="utf-8")
        return [sys.executable, "-I", str(path)]

    def test_all_expected_transcripts_through_real_transport(self):
        with tempfile.TemporaryDirectory() as directory:
            for profile in ("m1", "m2"):
                for case in build_cases(profile):
                    command = self.transcript_fixture(directory, expected_transcript(case))
                    result = assess.run_case(case, command, directory, Limits())
                    self.assertTrue(result["passed"], (profile, case["id"], result["errors"]))

    def test_paired_discrimination_in_both_directions(self):
        with tempfile.TemporaryDirectory() as directory:
            for profile in ("m1", "m2"):
                case = witness_case(profile)
                case["steps"] = [step for step in case["steps"] if not step.get("snapshot")]
                rows = expected_transcript(case)
                command = self.transcript_fixture(directory, rows)
                result = assess.discrimination(profile, command, directory, Limits())
                self.assertTrue(result["expected_mismatch_confirmed"], result["errors"])
                # A bad setup cannot count as a policy distinction.
                rows[0]["id"] = "wrong"
                command = self.transcript_fixture(directory, rows)
                self.assertFalse(assess.discrimination(profile, command, directory, Limits())["expected_mismatch_confirmed"])

    def test_cli_records_invalid_manifest_and_refuses_overwrite(self):
        with tempfile.TemporaryDirectory() as directory:
            manifest = Path(directory) / "manifest.json"
            for index, payload in enumerate(("{broken", "null", "[]")):
                manifest.write_text(payload, encoding="utf-8")
                output = Path(directory) / (str(index) + ".json")
                argv = ["assess.py", "--manifest", str(manifest), "--output", str(output)]
                with patch.object(sys, "argv", argv):
                    self.assertEqual(assess.main(), 1)
                report = json.loads(output.read_text(encoding="utf-8"))
                self.assertFalse(report["passed"])
                original = output.read_bytes()
                with patch.object(sys, "argv", argv), self.assertRaises(FileExistsError):
                    assess.main()
                self.assertEqual(output.read_bytes(), original)

    def test_cli_keeps_output_out_of_target(self):
        with tempfile.TemporaryDirectory() as directory:
            manifest = Path(directory) / "manifest.json"
            manifest.write_text(json.dumps({"cwd": directory}), encoding="utf-8")
            output = Path(directory) / "forbidden.json"
            with patch.object(sys, "argv", ["assess.py", "--manifest", str(manifest), "--output", str(output)]):
                with self.assertRaises(SystemExit) as stopped:
                    assess.main()
                self.assertEqual(stopped.exception.code, 2)
            self.assertFalse(output.exists())


if __name__ == "__main__":
    unittest.main()
