"""Focused tests for the Go test-event summary used by CI."""

from __future__ import annotations

import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


SCRIPT = Path(__file__).with_name("go-test-report.py")
SPEC = importlib.util.spec_from_file_location("go_test_report", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
go_test_report = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(go_test_report)


def events(*items: dict) -> list[str]:
    return [json.dumps(item) + "\n" for item in items]


class GoTestReportTests(unittest.TestCase):
    def test_non_ascii_failure_output_survives_a_legacy_code_page(self) -> None:
        failure = "plan → run, got � and ü\n"
        with tempfile.TemporaryDirectory() as temp:
            stream = Path(temp) / "events.json"
            stream.write_text("".join(events(
                {"Action": "output", "Package": "example/u", "Test": "TestArrow", "Output": failure},
                {"Action": "fail", "Package": "example/u", "Test": "TestArrow", "Elapsed": 0.1},
                {"Action": "fail", "Package": "example/u", "Elapsed": 0.2},
            )), encoding="utf-8")
            # A Windows runner redirects stdout with its ANSI code page.
            env = dict(os.environ, PYTHONIOENCODING="cp1252", PYTHONUTF8="0")
            result = subprocess.run([sys.executable, "-B", str(SCRIPT), str(stream)], capture_output=True, env=env)
        self.assertEqual(result.returncode, 1, result.stderr.decode("utf-8", "replace"))
        self.assertNotIn(b"Traceback", result.stderr)
        self.assertIn(failure.rstrip("\n").encode("utf-8"), result.stdout)

    def test_passing_stream_reports_timings_without_test_output(self) -> None:
        report = go_test_report.parse(events(
            {"Action": "run", "Package": "example/a", "Test": "TestFast"},
            {"Action": "output", "Package": "example/a", "Test": "TestFast", "Output": "noise\n"},
            {"Action": "pass", "Package": "example/a", "Test": "TestFast", "Elapsed": 0.5},
            {"Action": "skip", "Package": "example/a", "Test": "TestSkipped", "Elapsed": 0},
            {"Action": "pass", "Package": "example/a", "Elapsed": 1.25},
            {"Action": "skip", "Package": "example/empty", "Elapsed": 0},
        ))
        self.assertTrue(report.passed())
        out = io.StringIO()
        go_test_report.write_log(report, out, 5)
        self.assertIn("1.25s  example/a", out.getvalue())
        self.assertIn("1 tests passed, 0 failed, 1 skipped", out.getvalue())
        self.assertNotIn("noise", out.getvalue())

    def test_failed_test_and_package_output_are_printed(self) -> None:
        report = go_test_report.parse(events(
            {"Action": "output", "Package": "example/b", "Test": "TestBroken", "Output": "want 1, got 2\n"},
            {"Action": "fail", "Package": "example/b", "Test": "TestBroken", "Elapsed": 0.1},
            {"Action": "output", "Package": "example/b", "Output": "FAIL\n"},
            {"Action": "fail", "Package": "example/b", "Elapsed": 0.2},
        ))
        self.assertFalse(report.passed())
        out = io.StringIO()
        go_test_report.write_log(report, out, 5)
        self.assertIn("=== FAIL example/b TestBroken\nwant 1, got 2", out.getvalue())
        self.assertIn("=== FAIL package example/b\nFAIL", out.getvalue())

    def test_build_failure_and_empty_stream_fail(self) -> None:
        report = go_test_report.parse(events(
            {"Action": "build-output", "ImportPath": "example/c [example/c.test]", "Output": "c_test.go:3: undefined: x\n"},
            {"Action": "build-fail", "ImportPath": "example/c [example/c.test]"},
            {"Action": "output", "Package": "example/c", "Output": "FAIL\texample/c [build failed]\n"},
            {"Action": "fail", "Package": "example/c", "Elapsed": 0, "FailedBuild": "example/c [example/c.test]"},
        ))
        self.assertEqual(report.failed_packages(), ["example/c"])
        self.assertFalse(go_test_report.parse([]).passed())

    def test_interrupted_package_without_result_fails(self) -> None:
        report = go_test_report.parse(events(
            {"Action": "run", "Package": "example/d", "Test": "TestHangs"},
            {"Action": "output", "Package": "example/d", "Test": "TestHangs", "Output": "panic: test timed out after 1s\n"},
            {"Action": "output", "Package": "example/d", "Output": "FAIL\texample/d\t1.0s\n"},
            {"Action": "fail", "Package": "example/d", "Elapsed": 1.0},
        ))
        self.assertEqual(report.failed_tests(), [("example/d", "TestHangs")])
        out = io.StringIO()
        go_test_report.write_log(report, out, 5)
        self.assertIn("=== INTERRUPTED example/d TestHangs\npanic: test timed out", out.getvalue())

    def test_package_without_result_fails(self) -> None:
        report = go_test_report.parse(events(
            {"Action": "start", "Package": "example/e"},
            {"Action": "output", "Package": "example/e", "Output": "signal: killed\n"},
        ))
        self.assertEqual(report.failed_packages(), ["example/e"])

    def test_summary_is_appended_as_markdown(self) -> None:
        report = go_test_report.parse(events(
            {"Action": "pass", "Package": "example/a", "Test": "TestFast", "Elapsed": 0.5},
            {"Action": "pass", "Package": "example/a", "Elapsed": 1.0},
        ))
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / "summary.md"
            path.write_text("existing\n", encoding="utf-8")
            go_test_report.write_summary(report, path, "Go tests (linux)", 5)
            text = path.read_text(encoding="utf-8")
        self.assertTrue(text.startswith("existing\n### Go tests (linux)"))
        self.assertIn("| `example/a` | pass | 1.0 |", text)
        self.assertIn("| `TestFast` | `example/a` | 0.5 |", text)


if __name__ == "__main__":
    unittest.main()
