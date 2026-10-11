import contextlib
import io
import re
import unittest
from pathlib import Path
from unittest import mock

from playground import __main__ as entry
from playground import compare, evaluate, host, outcome, runner, study

README = Path(__file__).resolve().parent.parent / "README.md"
HOST_STATUSES = ("completed", "setup-failed", "start-failed", "wait-failed", "host-timeout", "host-interrupted",
                 "harness-error")
CLASSES = ("none", "harness", "environment", "product", None)  # None: no readable report.json


class MappingTests(unittest.TestCase):
    def test_host_run_maps_every_host_status_and_class(self):
        expected = {"setup-failed": 11, "start-failed": 11, "wait-failed": 11, "host-timeout": 124,
                    "host-interrupted": 130, "harness-error": 10}
        by_class = {"harness": 10, "environment": 11, "product": 12, None: 10}
        for status in HOST_STATUSES:
            for run_class in CLASSES:
                for container_exit in (0, 1, 2, None):
                    with self.subTest(status=status, run_class=run_class, container_exit=container_exit):
                        if status != "completed":
                            want = expected[status]
                        elif run_class == "none":
                            want = {0: 0, 1: 1}.get(container_exit, 10)  # 2: runner error
                        else:
                            want = by_class[run_class]
                        self.assertEqual(outcome.host_run(status, container_exit, run_class), want)

    def test_a_product_build_failure_and_a_killed_container(self):
        self.assertEqual(outcome.host_run("setup-failed", None, None, failure_class="product"), 12)
        self.assertEqual(outcome.host_run("setup-failed", None, None, failure_class="environment"), 11)
        self.assertEqual(outcome.host_run("start-failed", None, None, failure_class="product"), 11)
        for run_class in CLASSES:
            with self.subTest(run_class=run_class):
                self.assertEqual(outcome.host_run("completed", 137, run_class, oom_killed=True), 11)
                for oom in (False, None):  # Docker does not report OOMKilled: nothing explains the kill
                    self.assertEqual(outcome.host_run("completed", 137, run_class, oom_killed=oom), 10)
        self.assertEqual(outcome.host_run("completed", 1, "none", oom_killed=True), 1)  # only exit 137 counts
        self.assertEqual(outcome.host_run("host-timeout", 137, None, oom_killed=True), 124)
        self.assertEqual(outcome.assess("completed", 137, oom_killed=True), 11)
        self.assertEqual(outcome.assess("completed", 137, oom_killed=False), 10)
        self.assertEqual(outcome.assess("completed", 137), 10)

    def test_assess_fails_only_when_its_container_did(self):
        cases = {("completed", 0): 0, ("completed", 2): 10, ("completed", None): 10, ("start-failed", None): 11,
                 ("wait-failed", None): 11, ("host-timeout", None): 124, ("host-interrupted", None): 130}
        for (status, container_exit), want in cases.items():
            with self.subTest(status=status, container_exit=container_exit):
                self.assertEqual(outcome.assess(status, container_exit), want)

    def test_study_takes_the_worst_step_in_order(self):
        cases = [([], 0), ([0, 0], 0), ([0, 1], 1), ([1, 12], 12), ([12, 11, 1], 11), ([11, 10, 12], 10),
                 ([10, 124], 124), ([124, 130, 10], 130), ([0, 2], 10), ([1, 2, 12], 10), ([0, 137], 10)]
        for codes, want in cases:
            with self.subTest(codes=codes):
                self.assertEqual(outcome.study(codes), want)


class UnexpectedErrorTests(unittest.TestCase):
    def test_an_unhandled_exception_in_any_command_is_a_harness_failure(self):
        for argv, target in ((["run", "--manifest", "m.json", "--out", "o"], (runner, "load")),
                             (["host", "run", "--manifest", "m.json"], (host, "run")),
                             (["assess", "--run", "r"], (evaluate, "host_assess")),
                             (["compare", "a", "b"], (compare, "load")),
                             (["study", "s.json"], (study, "run"))):
            with self.subTest(command=argv[0]):
                err = io.StringIO()
                with mock.patch.object(*target, side_effect=RuntimeError("bug")), contextlib.redirect_stderr(err):
                    self.assertEqual(entry.main(argv), outcome.HARNESS)
                self.assertIn("RuntimeError: bug", err.getvalue())


class TableTests(unittest.TestCase):
    def help(self, *argv: str) -> str:
        out = io.StringIO()
        with contextlib.redirect_stdout(out), self.assertRaises(SystemExit) as caught:
            entry.main([*argv, "--help"])
        self.assertEqual(caught.exception.code, 0)
        return out.getvalue()

    def test_help_prints_every_code(self):
        for argv in (["host"], ["host", "run"], ["assess"], ["compare"], ["study"]):
            with self.subTest(argv=argv):
                text = self.help(*argv)
                self.assertIn("exit codes:", text)
                for code, meaning in outcome.TABLE.items():
                    self.assertIn(f"{code:>3}  {meaning}", text)

    def test_the_readme_lists_the_same_codes(self):
        section = README.read_text(encoding="utf-8").split("### Exit codes", 1)[1].split("\n#", 1)[0]
        rows = [line.split("|")[1] for line in section.splitlines() if re.match(r"\| *\d", line)]
        self.assertEqual(sorted(int(code) for row in rows for code in re.findall(r"\d+", row)),
                         sorted(outcome.TABLE))


if __name__ == "__main__":
    unittest.main()
