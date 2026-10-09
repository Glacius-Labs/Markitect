"""Offline response/launcher fixtures; no application or provider is executed."""
import importlib.util
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch


SOURCE = Path(__file__).resolve().parents[1] / "public/common/checks/acceptance.py"
SPEC = importlib.util.spec_from_file_location("public_acceptance", SOURCE)
CHECKER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(CHECKER)


class ReportingTests(unittest.TestCase):
    def test_missing_entrypoint_is_one_failure_with_unexecuted_checks(self):
        with tempfile.TemporaryDirectory() as folder, patch.object(CHECKER.subprocess, "run") as run:
            result = CHECKER.assess(Path(folder), "roombook", 1)
        self.assertEqual([f["status"] for f in result["findings"]], ["FAIL", "NOT RUN"])
        self.assertIn("app.py", result["findings"][0]["detail"])
        run.assert_not_called()

    def test_exit_and_stderr_are_reported_before_json_parse(self):
        with tempfile.TemporaryDirectory() as folder:
            repo = Path(folder)
            proc = subprocess.CompletedProcess([], 1, b"", b"application launch error")
            with patch.object(CHECKER.subprocess, "run", return_value=proc):
                with self.assertRaisesRegex(CHECKER.CandidateFailure, "exit 1, expected 0.*application launch error"):
                    CHECKER.invoke(repo, repo / "db.json", ("list",))

    def test_success_exit_with_invalid_json_is_output_failure(self):
        with tempfile.TemporaryDirectory() as folder:
            repo = Path(folder)
            proc = subprocess.CompletedProcess([], 0, b"not json", b"")
            with patch.object(CHECKER.subprocess, "run", return_value=proc):
                with self.assertRaisesRegex(CHECKER.CandidateFailure, "invalid UTF-8 JSON"):
                    CHECKER.invoke(repo, repo / "db.json", ("list",))

    def test_error_response_cannot_mutate_database(self):
        with tempfile.TemporaryDirectory() as folder:
            repo = Path(folder)
            db = repo / "db.json"
            db.write_bytes(b"[]")

            def fabricated_response(*args, **kwargs):
                db.write_bytes(b"[1]")
                return subprocess.CompletedProcess([], 2, b'{"error":"invalid"}', b"")

            with patch.object(CHECKER.subprocess, "run", side_effect=fabricated_response):
                with self.assertRaisesRegex(CHECKER.CandidateFailure, "mutated stored state"):
                    CHECKER.invoke(repo, db, ("bad",), error=True)

    def test_unavailable_python_process_is_evaluation_error(self):
        with tempfile.TemporaryDirectory() as folder:
            repo = Path(folder)
            (repo / "app.py").write_bytes(b"# fixture marker; never executed\n")
            with patch.object(CHECKER.subprocess, "run", side_effect=FileNotFoundError("fixture Python unavailable")):
                result = CHECKER.assess(repo, "roombook", 1)
        self.assertTrue(result["findings"])
        self.assertTrue(all(f["status"] == "EVALUATION_ERROR" for f in result["findings"]))


if __name__ == "__main__":
    unittest.main()
