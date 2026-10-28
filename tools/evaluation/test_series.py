"""Offline tests of change series: series judges, chain judging, planted tasks and the Library and Makerspace series judges."""
import datetime
import json
import os
import pathlib
import subprocess
import sys
import tempfile
import textwrap
import unittest

HERE = pathlib.Path(__file__).resolve().parent
JUDGES = HERE.parent / "evaluation-judges"
sys.path.insert(0, str(HERE))
sys.path.insert(0, str(JUDGES / "common"))
import judgekit  # noqa: E402
import judges  # noqa: E402
import judging  # noqa: E402
import series  # noqa: E402
import suite  # noqa: E402
from test_judging import IMAGE, Engine, go_version  # noqa: E402

LIBRARY = JUDGES / "library"
MAKERSPACE = JUDGES / "makerspace"
MODEL = "claude:claude-sonnet-5-5"

# A fake series judge over the steps A1, A2, A3: case c-<step> is introduced at each step, and every case passes
# except as FAKE_SERIES_MODE says: "regress" fails c-A1 at A3, "regress-early" fails c-A1 from A2 on, "flat" leaves
# earlier steps out of byStep, "other-step" answers for another step.
FAKE_SERIES_JUDGE = textwrap.dedent(r'''
    import json, os, pathlib, subprocess, sys
    task, start, result, out = sys.argv[-4:]
    mode, step = os.environ.get("FAKE_SERIES_MODE", "ok"), os.environ["EVALUATION_STEP"]
    steps = ["A1", "A2", "A3"]
    upto = steps[:steps.index(step) + 1]
    commit = subprocess.run(["git", "bundle", "list-heads", result], capture_output=True, text=True).stdout.split()[0]
    failing = (mode == "regress" and step == "A3") or (mode == "regress-early" and step != "A1")
    cases = [{"id": "c-" + s, "passed": not (failing and s == "A1")} for s in upto]
    failed = [c["id"] for c in cases if not c["passed"]]
    by = {s: {"planned": 1, "passed": int(not ("c-" + s) in failed), "failed": int(("c-" + s) in failed)} for s in upto}
    if mode == "flat":
        by = {step: {"planned": len(cases), "passed": len(cases) - len(failed), "failed": len(failed)}}
    record = {"format": 1, "commit": commit, "step": "A1" if mode == "other-step" else step,
              "correctness": {"planned": len(cases), "passed": len(cases) - len(failed), "failed": len(failed), "failedCases": failed,
                              "byStep": by, "cases": cases},
              "buildAndTests": {"build": {"passed": True}, "tests": {"passed": True}},
              "scope": {"changedPaths": [], "unrequested": [], "protectedTouched": []},
              "drift": {"forgottenRules": [c for c in failed if c != "c-" + step], "staleFiles": [], "unaccounted": []}}
    (pathlib.Path(out) / "judge.json").write_text(json.dumps(record))
''')

# A variant that commits one file named after the first line of task.md.
VARIANT = textwrap.dedent('''
    import json, os, pathlib, subprocess, sys
    task, out = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
    repo = out / "repo"
    env = dict(os.environ, GIT_AUTHOR_NAME="v", GIT_AUTHOR_EMAIL="v@x", GIT_COMMITTER_NAME="v", GIT_COMMITTER_EMAIL="v@x")
    g = lambda *a: subprocess.run(["git", "-C", str(repo), *a], check=True, capture_output=True, env=env)
    subprocess.run(["git", "clone", "-q", str(task / "start.bundle"), str(repo)], check=True, capture_output=True)
    g("checkout", "-q", "-B", "work", json.loads((task / "task.json").read_text())["start"]["commit"])
    mode = (task / "task.md").read_text().splitlines()[0]
    if mode != "nothing":
        (repo / (mode + ".txt")).write_text(mode + "\\n")
    g("add", "-A"); g("-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "work"); g("branch", "-f", "result", "HEAD")
    g("bundle", "create", str((out / "result.bundle").resolve()), "refs/heads/result")
''')


class SeriesJudgeTest(Engine):
    def setUp(self):
        super().setUp()
        self.judge = self.tmp / "judge"
        self.judge.mkdir()
        (self.judge / "judge.py").write_text(FAKE_SERIES_JUDGE)
        (self.judge / "judge.json").write_text(json.dumps({
            "format": 1, "name": "fake-series", "image": IMAGE, "command": ["python3", "-B", "/judge/judge.py"], "timeoutSeconds": 60,
            "limits": {"cpus": "1", "memoryBytes": 1 << 30, "pids": 64, "tmpBytes": 1 << 28}, "steps": ["A1", "A2", "A3"]}))
        (self.tmp / "variant.py").write_text(VARIANT)
        self.variant = [sys.executable, "-B", str(self.tmp / "variant.py")]
        steps = []
        for name in ("A1", "A2", "A3"):
            (self.tmp / "steps" / name).mkdir(parents=True)
            (self.tmp / "steps" / name / "task.md").write_text(name.lower() + "\n")
            steps.append(self.tmp / "steps" / name)
        self.chain = self.tmp / "chain"
        suite.chain(steps, None, 600, self.variant, self.chain, MODEL)

    def run_dir(self, i):
        return self.chain / f"step-{i:02d}" / "run"

    def test_a_series_judge_is_told_its_step_and_must_answer_cumulatively(self):
        judgment = suite.judge(self.run_dir(2), self.judge, self.tmp / "j2", self.engine, "A2")
        self.assertTrue(judgment["valid"], judgment["problems"])
        self.assertEqual(judgment["step"], "A2")
        self.assertEqual(list(judgment["areas"]["correctness"]["byStep"]), ["A1", "A2"])
        run = next(c for c in self.calls() if c[0] == "run")
        self.assertIn("EVALUATION_STEP=A2", run)
        for mode, problem in (("flat", "cumulative"), ("other-step", "not A2")):
            with self.subTest(mode):
                os.environ["FAKE_SERIES_MODE"] = mode
                judgment = suite.judge(self.run_dir(2), self.judge, self.tmp / ("j-" + mode), self.engine, "A2")
                self.assertFalse(judgment["valid"])
                self.assertTrue(any(problem in p for p in judgment["problems"]), judgment["problems"])

    def test_the_step_is_checked_before_anything_runs(self):
        with self.assertRaisesRegex(judging.JudgeError, "--step"):
            suite.judge(self.run_dir(1), self.judge, self.tmp / "none", self.engine)
        with self.assertRaisesRegex(judging.JudgeError, "--step"):
            suite.judge(self.run_dir(1), self.judge, self.tmp / "bad", self.engine, "B1")
        self.assertFalse((self.tmp / "calls.jsonl").exists())

    def test_a_chain_is_judged_step_by_step_with_regressions_and_its_final_drift(self):
        os.environ["FAKE_SERIES_MODE"] = "regress"
        record = suite.judge_chain(self.chain, self.judge, ["A1", "A2", "A3"], self.tmp / "judged", self.engine)
        self.assertEqual([r["step"] for r in record["steps"]], ["A1", "A2", "A3"])
        self.assertTrue(all(r["valid"] for r in record["steps"]))
        # passedAt is the last step at which the case passed.
        self.assertEqual(record["regressions"], [{"case": "c-A1", "passedAt": "A2", "failsAt": "A3"}])
        self.assertEqual(record["finalDrift"]["step"], "A3")
        self.assertEqual(record["finalDrift"]["forgottenRules"], ["c-A1"])
        self.assertTrue(record["lastStepJudged"])
        self.assertTrue((self.tmp / "judged" / "step-03" / "judgment.json").is_file())
        self.assertEqual(json.loads((self.chain / "chain.json").read_text())["steps"][0]["name"], "A1")

    def test_a_case_that_keeps_failing_is_one_regression(self):
        os.environ["FAKE_SERIES_MODE"] = "regress-early"
        record = suite.judge_chain(self.chain, self.judge, ["A1", "A2", "A3"], self.tmp / "early", self.engine)
        self.assertEqual(record["regressions"], [{"case": "c-A1", "passedAt": "A1", "failsAt": "A2"}])

    def test_a_chain_needs_a_judge_step_per_chain_step(self):
        with self.assertRaisesRegex(suite.SuiteError, "name a judge step"):
            suite.judge_chain(self.chain, self.judge, ["A1", "A2"], self.tmp / "short", self.engine)


class PlantTest(Engine):
    def setUp(self):
        super().setUp()
        (self.tmp / "variant.py").write_text(VARIANT)
        self.variant = [sys.executable, "-B", str(self.tmp / "variant.py")]

    def planted(self, kind, mode, recorded=None, name="plant"):
        folder = self.tmp / ("judge-" + name)
        folder.mkdir()
        recorded = recorded or (datetime.datetime.now(datetime.timezone.utc) - datetime.timedelta(hours=1)).isoformat()
        (folder / "plant.json").write_text(json.dumps({"format": 1, "kind": kind, "expected": {"defect": "no-landing", "gap": "landing"}[kind],
                                                       "description": "planted", "recordedUtc": recorded}))
        (self.tmp / (name + ".md")).write_text(mode + "\n")
        suite.make_task(self.tmp / (name + ".md"), None, None, 600, self.tmp / ("task-" + name))
        suite.run(self.tmp / ("task-" + name), self.variant, self.tmp / ("run-" + name), MODEL)
        return suite.judge(self.tmp / ("run-" + name), folder, self.tmp / ("out-" + name), self.engine)

    def test_a_defect_must_not_land_and_a_gap_must(self):
        for kind, mode, correct in (("defect", "nothing", True), ("defect", "work", False), ("gap", "work", True), ("gap", "nothing", False)):
            with self.subTest(kind=kind, mode=mode):
                judgment = self.planted(kind, mode, name=f"{kind}-{mode}")
                self.assertTrue(judgment["valid"], judgment["problems"])
                self.assertEqual(judgment["stops"]["landed"], mode != "nothing")
                self.assertEqual(judgment["stops"]["correct"], correct)
                self.assertIsNone(judgment["areas"])

    def test_a_plant_recorded_after_the_run_does_not_count(self):
        later = (datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(hours=1)).isoformat()
        judgment = self.planted("defect", "nothing", recorded=later)
        self.assertFalse(judgment["valid"])
        self.assertIn("recorded after the run", judgment["problems"][0])

    def test_a_plant_must_match_its_kind(self):
        folder = self.tmp / "bad"
        folder.mkdir()
        for plant, problem in (({"format": 1, "kind": "defect", "expected": "landing", "description": "x", "recordedUtc": "2026-10-28T00:00:00+00:00"},
                                "no-landing"),
                               ({"format": 1, "kind": "gap", "expected": "landing", "description": "x", "recordedUtc": "2026-10-28T00:00:00"},
                                "time zone")):
            (folder / "plant.json").write_text(json.dumps(plant))
            with self.assertRaisesRegex(judging.JudgeError, problem):
                judging.load_plant(folder)


class JudgeKitTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = pathlib.Path(self._tmp.name)

    def tearDown(self):
        self._tmp.cleanup()

    def test_correctness_is_split_by_the_step_that_introduced_each_case(self):
        first = judgekit.introduced({"S1": ["a", "b"], "S2": ["a", "b", "c"], "S3": ["a", "c"]}, ["S1", "S2", "S3"])
        self.assertEqual(first, {"a": "S1", "b": "S1", "c": "S2"})
        cases = [{"id": "a", "passed": False}, {"id": "c", "passed": True}, {"id": "d", "passed": False}]
        c = judgekit.correctness(cases, "S3", ["S1", "S2", "S3"], first)
        self.assertEqual(c["byStep"], {"S1": {"planned": 1, "passed": 0, "failed": 1}, "S2": {"planned": 1, "passed": 1, "failed": 0},
                                       "S3": {"planned": 1, "passed": 0, "failed": 1}})
        self.assertEqual(judgekit.drift(cases, "S3", self.tmp, "realization/", [], ["S1", "S2", "S3"], set())["forgottenRules"], ["a"])

    def test_unaccounted_files_and_stale_terms_are_found_in_the_whole_tree(self):
        files = {"realization/go.mod": "", "realization/main.go": "package main\n", "realization/ignored.go": "package main\n",
                 "realization/README.md": "", "realization/x/testdata/in.txt": "", "realization/notes.txt": "",
                 "realization/old_test.go": "const status = \"suspended\"\n", "realization/lib.go": "const s = \"suspended\"\n",
                 ".markitect/registry/root.md": "", ".gitignore": "", ".claude/settings.json": "", "TASK.md": ""}
        for rel, text in files.items():
            (self.tmp / rel).parent.mkdir(parents=True, exist_ok=True)
            (self.tmp / rel).write_text(text)
        used = {(self.tmp / p).resolve() for p in ("realization/main.go", "realization/lib.go", "realization/old_test.go")}
        self.assertEqual(judgekit.unaccounted(self.tmp, "realization/", used),
                         [".claude/settings.json", "realization/ignored.go", "realization/notes.txt"])
        self.assertIsNone(judgekit.unaccounted(self.tmp, "realization/", None))
        retired = [{"step": "S2", "pattern": "\"suspended\"", "cite": "x"}]
        self.assertEqual(judgekit.stale(self.tmp, "realization/", retired, "S1", ["S1", "S2"]), [])
        hits = judgekit.stale(self.tmp, "realization/", retired, "S2", ["S1", "S2"])
        self.assertEqual([(h["path"], h["line"]) for h in hits], [("realization/lib.go", 1)])

    def test_scope_sorts_bookkeeping_protected_inputs_and_residue(self):
        repo = self.tmp / "repo"
        env = dict(os.environ, GIT_AUTHOR_NAME="t", GIT_AUTHOR_EMAIL="t@x", GIT_COMMITTER_NAME="t", GIT_COMMITTER_EMAIL="t@x")
        run = lambda *a: subprocess.run(["git", "-C", str(repo), *a], check=True, capture_output=True, env=env)  # noqa: E731
        repo.mkdir()
        run("init", "-q")
        run("-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "start")
        start = suite.git(repo, "rev-parse", "HEAD")
        task = self.tmp / "task"
        (task / "model" / "root").mkdir(parents=True)
        (task / "model" / "root" / "project.md").write_bytes(b"model\n")
        for rel, data in {"realization/a.go": b"package a\n", ".markitect/registry/a.md": b"r\n", ".markitect/results/r1.md": b"r\n",
                          ".markitect/model/root/project.md": b"model\n", ".markitect/model/root/rule.md": b"edited\n",
                          "TASK.md": b"t\n", "tool": b"\x7fELF....", "ci.yaml": b"x\n"}.items():
            (repo / rel).parent.mkdir(parents=True, exist_ok=True)
            (repo / rel).write_bytes(data)
        run("add", "-A")
        run("-c", "commit.gpgsign=false", "commit", "-q", "-m", "result")
        result = judgekit.scope(task, repo, start, suite.git(repo, "rev-parse", "HEAD"), "realization/")
        self.assertEqual(result["unrequested"], ["ci.yaml", "tool"])
        self.assertEqual(result["protectedTouched"], [".markitect/model/root/rule.md", "TASK.md"])
        categories = {c["path"]: c["category"] for c in result["changedPaths"]}
        self.assertEqual(categories[".markitect/registry/a.md"], "bookkeeping")
        self.assertEqual(categories["realization/a.go"], "requested")


class SeriesSourceTest(unittest.TestCase):
    """Every series judge source: one reference per step, mutants that apply once, and fixed reference starts."""

    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = pathlib.Path(self._tmp.name)

    def tearDown(self):
        self._tmp.cleanup()

    def test_every_mutant_applies_to_its_steps_reference(self):
        for source in (LIBRARY, MAKERSPACE):
            config = judges.source_config(source)
            tasks = judges.series_tasks(source, self.tmp / source.name)
            found = judges.mutants(source)
            names = {n for n, _ in found}
            self.assertTrue({"failing-build", "unrequested-change", "protected-input", "forgotten-rule"} <= names, names)
            self.assertTrue(any(m["expect"].get("stale") for _, m in found))
            self.assertTrue(any(m["expect"].get("unaccounted") for _, m in found))
            for name, mutant in found:
                with self.subTest(source=source.name, mutant=name):
                    self.assertIn(mutant["step"], config["steps"])
                    judges.materialize(source, mutant, tasks[mutant["step"]], self.tmp / f"{source.name}-{name}.bundle", mutant["step"])
        self.assertEqual(sum(n.startswith("family-") for n, _ in judges.mutants(MAKERSPACE)), 31)

    def test_a_reference_start_replaces_the_tree_of_the_step_before(self):
        start = judges.start(MAKERSPACE, "C5", self.tmp / "c5.bundle")
        repo = self.tmp / "repo"
        suite.git(self.tmp, "clone", "-q", "-b", "start", str(self.tmp / "c5.bundle"), str(repo))
        self.assertEqual(suite.git(repo, "rev-parse", "HEAD"), start["commit"])
        self.assertTrue((repo / "realization" / "internal" / "ms" / "blocked_test.go").is_file())
        (self.tmp / "t.md").write_text("C6\n")
        suite.make_task(self.tmp / "t.md", None, self.tmp / "c5.bundle", 600, self.tmp / "task")
        judges.materialize(MAKERSPACE, None, self.tmp / "task", self.tmp / "c6.bundle", "C6")
        reference = {p.relative_to(JUDGES.parent / "makerspace-reference" / "c6").as_posix()
                     for p in (JUDGES.parent / "makerspace-reference" / "c6").rglob("*") if p.is_file()}
        suite.git(self.tmp, "clone", "-q", "-b", "result", str(self.tmp / "c6.bundle"), str(self.tmp / "c6"))
        tree = {p.relative_to(self.tmp / "c6" / "realization").as_posix() for p in (self.tmp / "c6" / "realization").rglob("*") if p.is_file()}
        self.assertEqual(tree, reference)
        with self.assertRaisesRegex(judging.JudgeError, "not a step"):
            judges.start(MAKERSPACE, "C10", self.tmp / "x.bundle")

    def test_export_keeps_the_steps_and_drops_the_test_material(self):
        for source in (LIBRARY, MAKERSPACE):
            config = judges.export(source, self.tmp / ("export-" + source.name))
            for key in judges.SOURCE_ONLY:
                self.assertNotIn(key, config)
            self.assertEqual(judging.load(self.tmp / ("export-" + source.name))["steps"], judges.source_config(source)["steps"])
            self.assertIn("judgekit.py", config["frozen"]["files"])

    def test_case_ids_are_cumulative_and_introduced_once(self):
        sys.path[:0] = [str(HERE.parent / "makerspace-oracle")]
        import catalog  # noqa: E402
        steps = judges.source_config(MAKERSPACE)["steps"]
        first = judgekit.introduced({s: [c["id"] for c in catalog.build_cases(s)] for s in steps}, steps)
        self.assertEqual(sum(v == "W0" for v in first.values()), len(catalog.build_cases("W0")))
        self.assertEqual({v for v in first.values()} - set(steps), set())


class PortfolioSeriesTest(unittest.TestCase):
    """The change series' inputs (series.py) and the planted tasks (tools/evaluation-judges/plants)."""

    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = pathlib.Path(self._tmp.name)

    def tearDown(self):
        self._tmp.cleanup()

    def test_each_series_has_the_judges_steps_and_both_representations(self):
        for name, source in (("library", LIBRARY), ("makerspace", MAKERSPACE)):
            self.assertEqual(series.steps(name), judges.source_config(source)["steps"])
            written = series.write_steps(name, self.tmp / name)
            self.assertEqual(len(written), len(series.steps(name)))
            for folder in map(pathlib.Path, written):
                with self.subTest(step=folder.name):
                    self.assertTrue((folder / "task.md").read_text(encoding="utf-8").startswith("# "))
                    for part in ("manifest.yaml", "lock.yaml", "execution.yaml", "model/project.md"):
                        self.assertTrue((folder / "model" / part).is_file(), part)
        with self.assertRaisesRegex(series.SeriesError, "not steps"):
            series.write_steps("library", self.tmp / "bad", ["M9"])

    def test_every_plant_edits_both_representations_once(self):
        plants = sorted((JUDGES / "plants").glob("*.json"))
        self.assertEqual({json.loads(p.read_text(encoding="utf-8"))["kind"] for p in plants}, {"defect", "gap"})
        for path in plants:
            with self.subTest(plant=path.stem):
                record = json.loads(path.read_text(encoding="utf-8"))
                self.assertLess(datetime.datetime.fromisoformat(record["recordedUtc"]), datetime.datetime.now(datetime.timezone.utc))
                result = judges.plant(path, self.tmp / path.stem / "step", None, self.tmp / path.stem / "judge")
                self.assertEqual(result["expected"], {"defect": "no-landing", "gap": "landing"}[record["kind"]])
                original = self.tmp / path.stem / "original"
                series.write_step(record["series"], record["step"], original)
                planted = self.tmp / path.stem / "step"
                self.assertNotEqual((planted / "task.md").read_bytes(), (original / "task.md").read_bytes())
                for edit in record["model"]:
                    self.assertNotEqual((planted / "model" / edit["path"]).read_bytes(), (original / "model" / edit["path"]).read_bytes())
                self.assertEqual(judging.load_plant(self.tmp / path.stem / "judge")["kind"], record["kind"])

    def test_a_plant_that_leaves_its_removed_statement_elsewhere_is_refused(self):
        record = json.loads((JUDGES / "plants" / "makerspace-c5-threshold-gap.json").read_text(encoding="utf-8"))
        record["model"], record["task"] = record["model"][:-1], record["task"][:-1]  # keep the "Times after 2099" sentence
        (self.tmp / "partial.json").write_text(json.dumps(record), encoding="utf-8")
        with self.assertRaisesRegex(judging.JudgeError, "still states.*43200"):
            judges.plant(self.tmp / "partial.json", self.tmp / "partial")

    def test_a_planted_judge_folder_carries_the_series_judge(self):
        judge = self.tmp / "judge"
        judges.export(LIBRARY, judge)
        plant = JUDGES / "plants" / "library-m2-transition-gap.json"
        judges.plant(plant, self.tmp / "step", judge, self.tmp / "planted-judge")
        self.assertEqual(judging.load(self.tmp / "planted-judge")["steps"], judges.source_config(LIBRARY)["steps"])
        self.assertTrue((self.tmp / "planted-judge" / "plant.json").is_file())
        with self.assertRaisesRegex(judging.JudgeError, "does not judge"):
            judges.plant(JUDGES / "plants" / "makerspace-c3-discount-contradiction.json", self.tmp / "s2", judge, self.tmp / "j2")


@unittest.skipUnless(sys.platform == "linux" and (go_version() or 0) >= 27,
                     "the judges run in a Linux container; needs Linux and Go 1.27 or later on the path")
class SeriesSelfTest(Engine):
    """The Library and Makerspace series judges pass their references and fail their mutants, through the fake engine."""

    def selftest(self, source, only=None):
        os.environ["FAKE_JUDGE_IMAGE"] = judges.source_config(source)["image"]
        judge = self.tmp / ("judge-" + source.name)
        judges.export(source, judge)
        summary = judges.selftest(source, judge, None, self.tmp / ("selftest-" + source.name), self.engine, only)
        self.assertTrue(summary["passed"], json.dumps([c for c in summary["cases"] if not c["ok"]], indent=1))
        return summary

    def test_library_series(self):
        summary = self.selftest(LIBRARY)
        self.assertEqual(len(summary["cases"]), 5 + 7)
        m5 = json.loads((self.tmp / "selftest-library" / "reference-M5" / "judgment.json").read_text())["areas"]["correctness"]
        self.assertEqual(list(m5["byStep"]), ["M1", "M2", "M3", "M4", "M5"])

    def test_makerspace_series_sample(self):
        # The full self-test (every step and all 37 mutants) runs in the judge container; here a sample that covers a
        # fresh build, a census step, the retired terms, an unchanged step and each kind of mutant.
        only = ["reference-W0", "reference-C3", "reference-C6", "reference-C9", "failing-build", "unrequested-change",
                "protected-input", "unaccounted-files", "stale-expiry", "forgotten-rule", "family-P-TIME-HOURS", "family-P-MONEY-LOCKOUT"]
        summary = self.selftest(MAKERSPACE, only)
        self.assertEqual(len(summary["cases"]), len(only))


if __name__ == "__main__":
    unittest.main()
