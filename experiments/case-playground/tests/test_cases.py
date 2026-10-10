"""Case discovery and the public checks' driver in the source tree and in a case repository."""
from __future__ import annotations

import json
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

from playground import cases, lifecycle

PLAYGROUND = Path(__file__).resolve().parents[1]
ACCEPTANCE = PLAYGROUND / "cases" / "common" / "checks" / "acceptance.py"
READINGLOG_S1 = ["legacy-add-restart", "duplicate-no-mutation", "finish-idempotence", "status-filter-regression",
                 "malformed-db-preserved"]


def make_case(playground: Path, name: str, stations: int = 2) -> Path:
    folder = playground / "cases" / name
    (folder / "checks").mkdir(parents=True)
    (folder / "README.md").write_text("# case\n", encoding="utf-8")
    (folder / "BACKLOG.md").write_text("- X01\n", encoding="utf-8")
    plan = {"schema": 1, "case": name, "stations": [{"id": f"S{n}", "items": [f"X{n:02d}"]}
                                                     for n in range(1, stations + 1)]}
    (folder / "STATIONS.json").write_text(json.dumps(plan), encoding="utf-8")
    (folder / "checks" / f"{name}.py").write_text("def checks(ctx):\n    pass\n", encoding="utf-8")
    return folder


def acceptance(script: Path, repo: Path, case: str, station: int, *, isolated: bool = True):
    flags = ["-I", "-B"] if isolated else []
    return subprocess.run([sys.executable, *flags, str(script), "--repo", str(repo), "--case", case,
                           "--station", str(station)], capture_output=True, text=True, encoding="utf-8",
                          cwd=repo, timeout=300)


def statuses(proc) -> list[tuple[str, str]]:
    return [(item["check"], item["status"]) for item in json.loads(proc.stdout)["findings"]]


class DiscoveryTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        (self.root / "cases" / "common" / "checks").mkdir(parents=True)
        (self.root / "cases" / "task-prompt.txt").write_text("prompt\n", encoding="utf-8")

    def assertInvalid(self, fragment: str) -> None:
        with self.assertRaises(cases.CaseError) as caught:
            cases.discover(self.root)
        self.assertIn(fragment, str(caught.exception))

    def test_the_playground_cases(self):
        found = cases.discover()
        self.assertEqual({name: case.stations for name, case in found.items()},
                         {"readinglog": 4, "readinglog2": 6, "roombook": 4})
        self.assertEqual(found["readinglog2"].evaluation, PLAYGROUND / "evaluation" / "readinglog2")
        self.assertIsNone(found["roombook"].evaluation)
        self.assertEqual(found["roombook"].checks, PLAYGROUND / "cases" / "roombook" / "checks" / "roombook.py")

    def test_a_valid_folder_is_a_case_and_files_and_common_are_not(self):
        make_case(self.root, "todo-list", stations=3)
        (self.root / "evaluation" / "todo-list").mkdir(parents=True)
        found = cases.discover(self.root)
        self.assertEqual(list(found), ["todo-list"])
        self.assertEqual((found["todo-list"].stations, found["todo-list"].evaluation),
                         (3, self.root / "evaluation" / "todo-list"))
        self.assertEqual(cases.get("todo-list", self.root).name, "todo-list")

    def test_an_unknown_case_names_the_known_ones(self):
        make_case(self.root, "todo")
        with self.assertRaisesRegex(cases.CaseError, "unknown case 'roombook'.*: todo"):
            cases.get("roombook", self.root)

    def test_every_rule_names_the_folder(self):
        rules = {
            "Bad_Name": (lambda folder: None, "cases/Bad_Name: a case folder name must match"),
            "acceptance": (lambda folder: None, "cases/acceptance: the name 'acceptance' is reserved"),
            "random": (lambda folder: None, "cases/random: the name 'random' is a Python standard-library module"),
            "no-readme": (lambda folder: (folder / "README.md").unlink(), "cases/no-readme: README.md is missing"),
            "no-backlog": (lambda folder: (folder / "BACKLOG.md").unlink(), "cases/no-backlog: BACKLOG.md is missing"),
            "no-plan": (lambda folder: (folder / "STATIONS.json").unlink(), "cases/no-plan: STATIONS.json is missing"),
            "other-plan": (lambda folder: (folder / "STATIONS.json").write_text(
                json.dumps({"case": "elsewhere", "stations": [{"id": "S1", "items": ["X"]}]}), encoding="utf-8"),
                "cases/other-plan: STATIONS.json does not describe case other-plan"),
            "bad-plan": (lambda folder: (folder / "STATIONS.json").write_text("{", encoding="utf-8"),
                         "cases/bad-plan: cannot read JSON record"),
            "no-checks": (lambda folder: (folder / "checks" / "no-checks.py").unlink(),
                          "cases/no-checks: its public checks checks/no-checks.py are missing"),
        }
        for name, (breaks, message) in rules.items():
            with self.subTest(rule=name):
                if name in ("Bad_Name", "acceptance"):
                    (self.root / "cases" / name).mkdir()
                    folder = self.root / "cases" / name
                else:
                    folder = make_case(self.root, name)
                    breaks(folder)
                self.assertInvalid(message)
                shutil.rmtree(folder)
        self.assertEqual(cases.discover(self.root), {})

    def test_no_cases_folder(self):
        shutil.rmtree(self.root / "cases")
        self.assertInvalid("no cases folder")


class AcceptanceLayoutTests(unittest.TestCase):
    """acceptance.py finds the case's checks beside itself (case repository) or in the source tree."""

    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name).resolve()
        self.repo = self.root / "work" / "readinglog"
        lifecycle.prepare(PLAYGROUND / "cases", self.repo, self.root / "audit", case="readinglog", method="m")

    def test_source_tree_and_case_repository_give_the_same_checks(self):
        source = acceptance(ACCEPTANCE, self.repo, "readinglog", 1)
        self.assertEqual(source.returncode, 1, source.stderr)  # the seed fails the S1 items
        seeded = acceptance(self.repo / "checks" / "acceptance.py", self.repo, "readinglog", 1)
        self.assertEqual(seeded.returncode, 1, seeded.stderr)
        self.assertEqual([name for name, _status in statuses(source)], READINGLOG_S1)
        self.assertEqual(statuses(seeded), statuses(source))

    def test_unknown_case_and_station_fail_clearly(self):
        for script in (ACCEPTANCE, self.repo / "checks" / "acceptance.py"):
            with self.subTest(script=script):
                for case in ("roombook-x", "common", "acceptance", "../readinglog"):
                    proc = acceptance(script, self.repo, case, 1)
                    self.assertEqual(proc.returncode, 2)
                    self.assertIn("unknown case", proc.stderr)
                proc = acceptance(script, self.repo, "readinglog", 5)
                self.assertEqual(proc.returncode, 2)
                self.assertIn("case readinglog has stations 1 to 4", proc.stderr)
        # the case repository holds only its own checks: no fallback to another case
        proc = acceptance(self.repo / "checks" / "acceptance.py", self.repo, "roombook", 1)
        self.assertEqual(proc.returncode, 2)
        self.assertIn("unknown case 'roombook'", proc.stderr)

    def test_station_count_comes_from_the_case_plan_and_no_bytecode_is_written(self):
        plan = json.loads((self.repo / "STATIONS.json").read_text(encoding="utf-8"))
        plan["stations"] = plan["stations"][:2]
        (self.repo / "STATIONS.json").write_text(json.dumps(plan), encoding="utf-8")
        proc = acceptance(self.repo / "checks" / "acceptance.py", self.repo, "readinglog", 3, isolated=False)
        self.assertIn("case readinglog has stations 1 to 2", proc.stderr)
        proc = acceptance(self.repo / "checks" / "acceptance.py", self.repo, "readinglog", 2, isolated=False)
        self.assertEqual(proc.returncode, 1, proc.stderr)
        self.assertFalse((self.repo / "checks" / "__pycache__").exists())


if __name__ == "__main__":
    unittest.main()
