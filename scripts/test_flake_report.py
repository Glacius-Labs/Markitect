"""Focused tests for the flake report over downloaded CI test results."""

import importlib.util
import json
from pathlib import Path
import tempfile
import unittest


SCRIPT = Path(__file__).with_name("flake-report.py")
SPEC = importlib.util.spec_from_file_location("flake_report", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
flake_report = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(flake_report)

SHA_A = "a" * 40
SHA_B = "b" * 40


def stream(*events: dict) -> str:
    return "".join(json.dumps(event) + "\n" for event in events)


def result(package: str, test: str, status: str) -> str:
    return stream(
        {"Action": status, "Package": package, "Test": test, "Elapsed": 0.1},
        {"Action": "pass" if status == "pass" else "fail", "Package": package, "Elapsed": 0.2},
    )


class FlakeReportTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.results = self.root / "results"
        self.runs = []

    def tearDown(self) -> None:
        self.temp.cleanup()

    def artifact(self, run: int, name: str, content: str) -> None:
        if not any(entry["databaseId"] == run for entry in self.runs):
            self.runs.append({"databaseId": run, "createdAt": f"2026-10-{10 + len(self.runs):02d}T00:00:00Z"})
        folder = self.results / str(run) / name
        folder.mkdir(parents=True)
        (folder / "go-test.json").write_text(content, encoding="utf-8")

    def report(self) -> str:
        return flake_report.render(flake_report.collect(self.runs, self.results), self.runs, 30)

    def test_a_re_run_that_passes_keeps_the_failed_attempt_visible(self) -> None:
        self.artifact(1, f"go-test-ubuntu-24.04-{SHA_A}-1", result("example/a", "TestWobbly", "fail"))
        self.artifact(1, f"go-test-ubuntu-24.04-{SHA_A}-2", result("example/a", "TestWobbly", "pass"))
        text = self.report()
        self.assertIn("## Flaky tests (2)", text)  # the test and its package
        self.assertIn("| `TestWobbly` | `example/a` | ubuntu-24.04 | 1 | 1 / 2 | 1#1 |", text)

    def test_a_consistent_failure_is_failing_but_not_flaky(self) -> None:
        self.artifact(1, f"go-test-ubuntu-24.04-{SHA_A}-1", result("example/b", "TestBroken", "fail"))
        self.artifact(2, f"go-test-ubuntu-24.04-{SHA_B}-1", result("example/b", "TestBroken", "fail"))
        self.artifact(3, f"go-test-windows-latest-{SHA_A}-1", result("example/b", "TestBroken", "pass"))
        text = self.report()
        self.assertIn("## Flaky tests (0)", text)
        self.assertIn("| `TestBroken` | `example/b` | ubuntu-24.04 | 2 / 2 | 2#1 |", text)

    def test_build_failures_count_as_package_failures(self) -> None:
        self.artifact(1, f"go-test-ubuntu-24.04-{SHA_A}", stream(
            {"Action": "build-fail", "ImportPath": "example/c [example/c.test]"},
            {"Action": "fail", "Package": "example/c", "Elapsed": 0, "FailedBuild": "example/c [example/c.test]"},
        ))
        text = self.report()
        self.assertIn("| `(package)` | `example/c` | ubuntu-24.04 | 1 / 1 | 1#1 |", text)

    def test_unknown_runs_and_unreadable_artifacts_are_reported_as_skipped(self) -> None:
        self.artifact(1, f"go-test-ubuntu-24.04-{SHA_A}-1", result("example/d", "TestFine", "pass"))
        (self.results / "999" / f"go-test-ubuntu-24.04-{SHA_A}-1").mkdir(parents=True)
        (self.results / "1" / "go-test-unrelated").mkdir()
        text = self.report()
        self.assertIn("1 CI runs with test results", text)
        self.assertIn("Skipped 2 downloads", text)
        self.assertIn("## All failing tests (0)", text)


if __name__ == "__main__":
    unittest.main()
