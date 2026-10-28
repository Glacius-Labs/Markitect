"""Offline tests of the judge runner and the Library judge: a fake engine runs the judge's command on the host."""
import json
import os
import pathlib
import re
import shutil
import subprocess
import sys
import tempfile
import textwrap
import unittest

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import judges  # noqa: E402
import judging  # noqa: E402
import suite  # noqa: E402

LIBRARY = HERE.parent / "evaluation-judges" / "library-m1"

# A fake engine for judges: `image inspect` knows FAKE_JUDGE_IMAGE only, `run` runs the command on the host with each
# mount target replaced by its source (python3 by this Python, --contained dropped: on a host it must never stop the
# user's processes), and FAKE_RESIDUE leaves a container behind.
FAKE_ENGINE = textwrap.dedent(r'''
    import json, os, subprocess, sys
    args = sys.argv[1:]
    with open(os.path.join(os.environ["FAKE_DIR"], "calls.jsonl"), "a") as f:
        f.write(json.dumps(args) + "\n")
    if args[:2] == ["image", "inspect"]:
        if args[-1] != os.environ["FAKE_JUDGE_IMAGE"]:
            sys.exit("no such image")
        print("sha256:" + "b" * 64)
    elif args[0] == "ps":
        print(os.environ.get("FAKE_RESIDUE", ""))
    elif args[:2] == ["network", "ls"] or args[0] in ("kill", "rm"):
        pass
    elif args[0] == "run":
        assert args[args.index("--network") + 1] == "none", "the judge must have no network"
        mounts, i = [], 1
        while args[i] != os.environ["FAKE_JUDGE_IMAGE"]:
            if args[i] == "--mount":
                fields = dict(f.partition("=")[::2] for f in args[i + 1].split(","))
                mounts.append((fields["dst"], fields["src"], "readonly" in args[i + 1]))
            i += 1 if args[i] in ("--init", "--read-only") else 2
        assert [m[2] for m in mounts] == [True, True, True, True, False], mounts
        mounts.sort(key=lambda m: -len(m[0]))
        def host(a):
            for dst, src, _ in mounts:
                if a == dst or a.startswith(dst + "/"):
                    return src + a[len(dst):]
            return sys.executable if a == "python3" else a
        sys.exit(subprocess.run([host(a) for a in args[i + 1:] if a != "--contained"]).returncode)
    else:
        sys.exit("fake engine: unknown call " + " ".join(args))
''')

IMAGE = "example/judge@sha256:" + "c" * 64

# A fake judge: reads the result's one commit and answers as FAKE_JUDGE_MODE says.
FAKE_JUDGE = textwrap.dedent(r'''
    import json, os, pathlib, subprocess, sys, time
    task, start, result, out = sys.argv[-4:]
    mode = os.environ.get("FAKE_JUDGE_MODE", "ok")
    heads = subprocess.run(["git", "bundle", "list-heads", result], capture_output=True, text=True).stdout.split()
    commit = heads[0] if mode != "other-commit" else "0" * 40
    if mode == "sleep":
        time.sleep(30)
    if mode == "silent":
        sys.exit(0)
    planned, failed = 3, (1 if mode != "bad-counts" else 2)
    record = {"format": 1, "commit": commit,
              "correctness": {"planned": planned, "passed": planned - 1, "failed": failed, "failedCases": ["c2"]},
              "buildAndTests": {"build": {"passed": True}, "tests": {"passed": False}},
              "scope": {"changedPaths": [{"path": "work.txt"}], "unrequested": [], "protectedTouched": []}}
    if mode == "error":
        record = {"format": 1, "error": "RuntimeError: broken"}
    (pathlib.Path(out) / "judge.json").write_text(json.dumps(record))
''')


def go_version():
    """The local Go's minor version, or None when no Go is on the path."""
    go = shutil.which("go")
    if not go:
        return None
    match = re.search(r"go1\.(\d+)", subprocess.run([go, "version"], capture_output=True, text=True).stdout)
    return int(match.group(1)) if match else None


class Engine(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = pathlib.Path(self._tmp.name)
        (self.tmp / "engine.py").write_text(FAKE_ENGINE)
        self.engine = [sys.executable, "-I", str(self.tmp / "engine.py")]
        self._env = dict(os.environ)
        os.environ.update(FAKE_DIR=str(self.tmp), FAKE_JUDGE_IMAGE=IMAGE)

    def tearDown(self):
        os.environ.clear()
        os.environ.update(self._env)
        self._tmp.cleanup()

    def calls(self):
        return [json.loads(line) for line in (self.tmp / "calls.jsonl").read_text().splitlines()]


class JudgeRunnerTest(Engine):
    def setUp(self):
        super().setUp()
        self.judge = self.tmp / "judge"
        self.judge.mkdir()
        (self.judge / "judge.py").write_text(FAKE_JUDGE)
        self.config = {"format": 1, "name": "fake", "image": IMAGE, "command": ["python3", "-B", "/judge/judge.py"], "timeoutSeconds": 60,
                       "limits": {"cpus": "1", "memoryBytes": 1 << 30, "pids": 64, "tmpBytes": 1 << 28}}
        self.write_config()
        (self.tmp / "task.md").write_text("ok\n")
        self.task = self.tmp / "task"
        suite.make_task(self.tmp / "task.md", None, None, 600, self.task)
        fake = self.tmp / "variant.py"
        fake.write_text(textwrap.dedent('''
            import json, os, pathlib, subprocess, sys
            task, out = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
            repo = out / "repo"
            env = dict(os.environ, GIT_AUTHOR_NAME="v", GIT_AUTHOR_EMAIL="v@x", GIT_COMMITTER_NAME="v", GIT_COMMITTER_EMAIL="v@x")
            g = lambda *a: subprocess.run(["git", "-C", str(repo), *a], check=True, capture_output=True, env=env)
            subprocess.run(["git", "clone", "-q", str(task / "start.bundle"), str(repo)], check=True, capture_output=True)
            g("checkout", "-q", "-B", "work", json.loads((task / "task.json").read_text())["start"]["commit"])
            (repo / "work.txt").write_text("work\\n")
            g("add", "-A"); g("-c", "commit.gpgsign=false", "commit", "-q", "-m", "work"); g("branch", "-f", "result", "HEAD")
            g("bundle", "create", str((out / "result.bundle").resolve()), "refs/heads/result")
        '''))
        self.run_dir = self.tmp / "run"
        self.record = suite.run(self.task, [sys.executable, "-B", str(fake)], self.run_dir, "claude:claude-sonnet-5-5")

    def write_config(self, **change):
        (self.judge / "judge.json").write_text(json.dumps({**self.config, **change}))

    def judged(self, mode="ok", out="judgment"):
        os.environ["FAKE_JUDGE_MODE"] = mode
        return suite.judge(self.run_dir, self.judge, self.tmp / out, self.engine)

    def test_a_judge_runs_without_network_on_read_only_inputs_and_is_recorded(self):
        judgment = self.judged()
        self.assertTrue(judgment["valid"], judgment["problems"])
        self.assertEqual(judgment["commit"], self.record["result"]["commit"])
        self.assertEqual(judgment["areas"]["correctness"]["failedCases"], ["c2"])
        self.assertEqual(judgment["judge"]["imageId"], "sha256:" + "b" * 64)
        self.assertEqual(judgment["judge"]["folderSha256"], judging.folder_digest(self.judge))
        run = next(c for c in self.calls() if c[0] == "run")
        self.assertEqual(run[-4:], list(judging.MOUNTS))
        for flag in ("--read-only", "--init"):
            self.assertIn(flag, run)
        self.assertEqual(run[run.index("--cap-drop") + 1], "ALL")
        # Only the suite's container lets the judge stop every other process.
        self.assertIn("EVALUATION_JUDGE_CONTAINER=" + run[run.index("--name") + 1], run)
        written = json.loads((self.tmp / "judgment" / "judge.json").read_text())
        self.assertEqual(written["commit"], judgment["commit"])
        self.assertIn(["rm", "-f", run[run.index("--name") + 1]], self.calls())

    def test_a_wrong_or_missing_answer_makes_the_judgment_invalid(self):
        for mode, problem in (("other-commit", "not the result"), ("bad-counts", "add up"), ("silent", "no judge.json"),
                              ("error", "reports an error")):
            with self.subTest(mode):
                judgment = self.judged(mode, "out-" + mode)
                self.assertFalse(judgment["valid"])
                self.assertTrue(any(problem in p for p in judgment["problems"]), judgment["problems"])

    def test_a_judge_that_overruns_is_stopped_and_invalid(self):
        self.write_config(timeoutSeconds=2)
        judgment = self.judged("sleep")
        self.assertFalse(judgment["valid"])
        self.assertIn("the judge timed out", judgment["problems"])
        self.assertTrue(any(c[0] == "kill" for c in self.calls()))

    def test_residue_makes_the_judgment_invalid(self):
        os.environ["FAKE_RESIDUE"] = "judge-left"
        judgment = self.judged()
        self.assertFalse(judgment["valid"])
        self.assertIn("residue: judge-left", judgment["problems"])

    def test_configuration_and_inputs_are_checked_before_anything_runs(self):
        self.write_config(image="golang:1.27")
        with self.assertRaisesRegex(judging.JudgeError, "digest"):
            self.judged()
        self.write_config(include={"judge.py": "x"})
        with self.assertRaisesRegex(judging.JudgeError, "export"):
            self.judged(out="o2")
        self.write_config(image="example/other@sha256:" + "d" * 64)
        self.assertIn("pull it first", self.judged(out="o3")["problems"][0])
        (self.run_dir / "variant" / "result.bundle").write_bytes(b"changed")
        with self.assertRaisesRegex(suite.SuiteError, "not the result"):
            self.judged(out="o4")

    def test_a_run_without_result_is_not_judged(self):
        record = json.loads((self.run_dir / "suite.json").read_text())
        record["delivered"] = False
        (self.run_dir / "suite.json").write_text(json.dumps(record))
        with self.assertRaisesRegex(suite.SuiteError, "nothing to judge"):
            self.judged()


class HiddenMaterialTest(unittest.TestCase):
    def test_no_judge_material_is_in_the_folder_variants_get_as_their_code(self):
        # The documented commands mount tools/evaluation at /variant in every variant run and reviewer session.
        found = [p.relative_to(HERE).as_posix() for p in HERE.rglob("*")
                 if p.name in ("judge.json", "go.mod") or p.parent.name in ("reference", "mutants")]
        self.assertEqual(found, [])


class LibraryJudgeTest(Engine):
    """The Library judge must pass its reference and fail each seeded mutant where it is seeded."""

    def test_export_takes_every_file_from_one_revision_with_its_hash(self):
        out = self.tmp / "exported"
        config = judges.export(LIBRARY, out)
        self.assertEqual(config["frozen"]["rev"], "working tree")
        self.assertNotIn("include", json.loads((out / "judge.json").read_text()))
        for rel, digest in config["frozen"]["files"].items():
            self.assertEqual(judging.hashlib.sha256((out / rel).read_bytes()).hexdigest(), digest, rel)
        self.assertTrue((out / "rubric.md").is_file())
        self.assertEqual(judging.load(out)["image"], json.loads((LIBRARY / "judge.json").read_text())["image"])

    def test_export_at_a_revision_ignores_the_working_tree(self):
        repo, source = self.tmp / "repo", self.tmp / "repo" / "judges" / "x"
        source.mkdir(parents=True)
        (repo / "oracle.py").write_text("committed\n")
        config = {**json.loads((LIBRARY / "judge.json").read_text()), "include": {"tools/oracle.py": "oracle.py"}}
        (source / "judge.json").write_text(json.dumps(config))
        for args in (["init", "-q"], ["add", "-A"], ["-c", "user.name=t", "-c", "user.email=t@x", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "c"]):
            subprocess.run(["git", "-C", str(repo), *args], check=True, capture_output=True)
        (repo / "oracle.py").write_text("changed later\n")
        head = suite.git(repo, "rev-parse", "HEAD")
        frozen = judges.export(source, self.tmp / "frozen", rev="HEAD", repo=repo)
        self.assertEqual(frozen["frozen"]["rev"], head)
        self.assertEqual((self.tmp / "frozen" / "tools" / "oracle.py").read_text(), "committed\n")
        with self.assertRaisesRegex(judging.JudgeError, "exists"):
            judges.export(source, self.tmp / "frozen", rev="HEAD", repo=repo)

    def test_every_mutant_changes_the_reference_once(self):
        (self.tmp / "task.md").write_text("Library M1\n")
        suite.make_task(self.tmp / "task.md", None, None, 600, self.tmp / "task")
        for path in sorted((LIBRARY / "mutants").glob("*.json")):
            with self.subTest(path.stem):
                mutant = json.loads(path.read_text())
                self.assertTrue(mutant.get("expect"))
                judges.materialize(LIBRARY, mutant, self.tmp / "task", self.tmp / (path.stem + ".bundle"))
        stems = {p.stem for p in (LIBRARY / "mutants").glob("*.json")}
        self.assertTrue({"broken-rule", "unrequested-change", "failing-build"} <= stems, stems)

    @unittest.skipUnless(sys.platform == "linux" and (go_version() or 0) >= 27,
                         "the judge runs in a Linux container; needs Linux and Go 1.27 or later on the path")
    def test_reference_passes_and_mutants_fail(self):
        os.environ["FAKE_JUDGE_IMAGE"] = json.loads((LIBRARY / "judge.json").read_text())["image"]
        judge = self.tmp / "judge"
        judges.export(LIBRARY, judge)
        (self.tmp / "task.md").write_text("Library M1\n")
        suite.make_task(self.tmp / "task.md", None, None, 600, self.tmp / "task")
        summary = judges.selftest(LIBRARY, judge, self.tmp / "task", self.tmp / "selftest", self.engine)
        self.assertTrue(summary["passed"], json.dumps(summary["cases"], indent=1))
        self.assertEqual([c["case"] for c in summary["cases"]][0], "reference")
        self.assertGreaterEqual(len(summary["cases"]), 4)
        reference = json.loads((self.tmp / "selftest" / "reference" / "judgment.json").read_text())["areas"]
        self.assertEqual(reference["correctness"]["planned"], 19)
        self.assertEqual(reference["correctness"]["failed"], 0)

    def test_a_judgment_that_misses_the_seeded_defect_fails_the_selftest(self):
        good = {"valid": True, "problems": [], "areas": {
            "correctness": {"planned": 19, "passed": 19, "failed": 0, "failedCases": []},
            "buildAndTests": {"build": {"passed": True}, "tests": {"passed": True}},
            "scope": {"changedPaths": [], "unrequested": [], "protectedTouched": []}}}
        self.assertEqual(judges.verdict(good, None), [])
        self.assertTrue(judges.verdict(good, {"expect": {"correctnessFails": True}}))
        self.assertTrue(judges.verdict(good, {"expect": {"buildFails": True}}))
        self.assertTrue(judges.verdict(good, {"expect": {"unrequested": [".github/workflows/ci.yaml"]}}))
        self.assertTrue(judges.verdict(good, {"expect": {"protected": ["TASK.md"]}}))
        self.assertTrue(judges.verdict({**good, "valid": False, "problems": ["x"]}, None))


if __name__ == "__main__":
    unittest.main()
