import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from playground import assess, codex_agent

PLAYGROUND = Path(__file__).resolve().parents[1]
GIT_IDENTITY = ["-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "core.autocrlf=false",
                "-c", "maintenance.auto=false", "-c", "gc.auto=0"]


def make_candidate(root: Path) -> Path:
    """The seeded Readinglog baseline as a committed Git checkout."""
    repo = root / "candidate"
    shutil.copytree(PLAYGROUND / "cases" / "common", repo)
    shutil.copytree(PLAYGROUND / "cases" / "readinglog", repo, dirs_exist_ok=True)
    for args in (["init", "-q", "--initial-branch=main"], ["add", "-A"], ["commit", "-q", "-m", "seed"]):
        subprocess.run(["git", *GIT_IDENTITY, *args], cwd=repo, check=True, capture_output=True)
    return repo


def tree_digest(root: Path) -> dict:
    return {path.relative_to(root).as_posix(): hashlib.sha256(path.read_bytes()).hexdigest()
            for path in sorted(root.rglob("*")) if path.is_file()}


class AssessTest(unittest.TestCase):
    def setUp(self):
        self.temp = Path(tempfile.mkdtemp())
        self.addCleanup(assess._rmtree, self.temp)
        if codex_agent.container_mode():  # checks run as the agent user, which writes here too
            os.chown(self.temp, *codex_agent._agent_ids())
        self.candidate = make_candidate(self.temp)

    def test_candidate_fixture_does_not_start_automatic_maintenance(self):
        config = self.temp / "global.gitconfig"
        config.write_text("[maintenance]\n\tauto = true\n[gc]\n\tauto = 1\n", encoding="utf-8")
        trace = self.temp / "git-trace.jsonl"
        env = {key: value for key, value in os.environ.items() if not key.upper().startswith("GIT_")}
        env.update(GIT_CONFIG_GLOBAL=str(config), GIT_CONFIG_NOSYSTEM="1", GIT_TRACE2_EVENT=trace.as_posix())
        with mock.patch.dict(os.environ, env, clear=True):
            candidate = make_candidate(self.temp / "isolated")
        events = [json.loads(line) for line in trace.read_text(encoding="utf-8").splitlines()]
        self.assertTrue((candidate / ".git" / "HEAD").is_file())
        self.assertTrue(any(event["event"] == "start" and "commit" in event["argv"] for event in events))
        maintenance = [event["argv"] for event in events if event["event"] == "child_start"
                       and any(command in event["argv"] for command in ("maintenance", "gc"))]
        self.assertEqual(maintenance, [])

    def test_station_one_fails_new_items_but_baseline_passes(self):
        result = assess.station(self.candidate, "readinglog", 1, PLAYGROUND, self.temp / "S1")
        by_check = {item["check"]: item["status"] for item in result["findings"]}
        self.assertEqual(by_check["finish-idempotence"], "FAIL")
        self.assertEqual(by_check["status-filter-regression"], "FAIL")
        for legacy in ("legacy-add-restart", "duplicate-no-mutation", "malformed-db-preserved"):
            self.assertEqual(by_check[legacy], "PASS", legacy)
        self.assertEqual((result["passed"], result["total"], result["status"]), (3, 5, "fail"))
        raw = json.loads((self.temp / "S1" / result["stdout"]).read_text(encoding="utf-8"))
        self.assertEqual(len(raw["findings"]), 5)

    def test_the_candidates_own_copy_of_the_checks_never_judges_it(self):
        (self.candidate / "checks" / "readinglog.py").write_text("def checks(ctx):\n    pass\n", encoding="utf-8")
        (self.candidate / "STATIONS.json").write_text('{"schema": 1, "case": "readinglog", "stations": []}',
                                                      encoding="utf-8")
        result = assess.station(self.candidate, "readinglog", 1, PLAYGROUND, self.temp / "S1")
        self.assertEqual((result["passed"], result["total"], result["status"]), (3, 5, "fail"))

    def test_scratch_copy_has_no_git(self):
        copy = assess.scratch_copy(self.candidate, self.temp / "scratch")
        self.assertTrue((copy / "app.py").is_file())
        self.assertEqual([p for p in copy.rglob(".git")], [])

    def test_final_records_own_tests_and_never_changes_candidate(self):
        before = tree_digest(self.candidate)
        result = assess.final(self.candidate, "readinglog", "conventional", PLAYGROUND, self.temp / "final")
        self.assertEqual(tree_digest(self.candidate), before)
        self.assertEqual(result["total"], 13)
        self.assertEqual(result["checks"]["station"], 4)  # the last station of readinglog's STATIONS.json
        self.assertEqual(result["status"], "fail")
        self.assertEqual((result["ownTests"]["status"], result["ownTests"]["ran"]), ("pass", 1))
        self.assertIsNone(result["conformance"])

    def test_final_takes_an_explicit_station_and_names_who_failed_the_product_check(self):
        with mock.patch.object(assess, "_markitect_command", return_value=None):
            result = assess.final(self.candidate, "readinglog", "markitect", PLAYGROUND, self.temp / "final", station=1)
        self.assertEqual((result["checks"]["station"], result["total"]), (1, 5))
        self.assertEqual((result["conformance"]["status"], result["conformance"]["errorSource"]), ("error", "harness"))

    def test_final_markitect_runs_product_check_on_scratch_git_copy(self):
        log = self.temp / "markitect-argv.json"
        fake = self.temp / "fake_markitect.py"
        fake.write_text(
            "import json, pathlib, sys\n"
            "args = sys.argv[1:]\n"
            "repo = pathlib.Path(args[args.index('--repo') + 1])\n"
            f"pathlib.Path({str(log)!r}).write_text(json.dumps({{'args': args, 'git': (repo / '.git').is_dir(),"
            " 'app': (repo / 'app.py').is_file()}), encoding='utf-8')\n"
            "print(json.dumps({'status': 'succeeded', 'findings': None, 'coverage': {'conforming': False}}))\n"
            "sys.exit(1)\n", encoding="utf-8")
        before = tree_digest(self.candidate)
        result = assess.final(self.candidate, "readinglog", "markitect", PLAYGROUND, self.temp / "final",
                              markitect_cmd=[sys.executable, str(fake)])
        self.assertEqual(tree_digest(self.candidate), before)
        seen = json.loads(log.read_text(encoding="utf-8"))
        self.assertEqual(seen["args"][:2], ["check", "--repo"])
        self.assertNotEqual(Path(seen["args"][2]).resolve(), self.candidate.resolve())
        self.assertTrue(seen["git"] and seen["app"])
        conformance = result["conformance"]
        self.assertEqual((conformance["status"], conformance["exitCode"]), ("fail", 1))
        self.assertEqual((conformance["reportStatus"], conformance["coverageConforming"]), ("succeeded", False))


if __name__ == "__main__":
    unittest.main()
