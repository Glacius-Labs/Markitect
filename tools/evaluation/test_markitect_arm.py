"""Offline tests of the concept 2.0 arm: real Git, a scripted `markitect` stand-in, no engine and no provider."""
import json
import os
import pathlib
import subprocess
import sys
import tempfile
import textwrap
import time
import unittest

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import classify  # noqa: E402
import results  # noqa: E402
import suite  # noqa: E402

MODEL = "claude:claude-sonnet-5-5"

# A scripted `markitect`. `check --json` reports every file outside .markitect/ that no registry table lists, on the
# page of the Package whose `DIR/**` pattern is the longest match (Root's registry.md otherwise), as the compiler words
# it. `apply` records its arguments and environment in FAKE_DIR/apply-N.json and acts as FAKE_APPLY says:
#   land        commits realization/<run>.txt, lists it on the first Package's page, adds the Result page, pushes the
#               commit to the target's main and prints a landed Result
#   defect      a Run that ends at Root on a model defect
#   incomplete  a Run that stops without approved work; its makers' last starts finished
#   provider    as incomplete, but the lending maker's last start failed on a rate limit
#   recovered   as incomplete, with a rate limit followed by a finished start of the same maker
#   machinery   as incomplete, whose only start failed before any agent turn
#   gate        a Run whose initial gate failed: no agent started
#   none        prints nothing and exits 2, as when the host cannot run
# FAKE_CHECK=fail makes every `check` report a compile error.
STANDIN = textwrap.dedent(r'''
    import json, os, pathlib, re, subprocess, sys
    args = sys.argv[1:]
    opt = lambda name: args[args.index(name) + 1] if name in args else None
    repo = pathlib.Path(opt("--repo"))
    git = lambda *a, cwd=repo: subprocess.run(["git", "-c", "core.autocrlf=false", "-C", str(cwd), *a], check=True, capture_output=True,
                                              text=True).stdout.strip()

    def pages():
        found = {}
        for page in sorted((repo / ".markitect" / "registry").rglob("*.md")):
            text = page.read_text(encoding="utf-8")
            found[page.relative_to(repo).as_posix()] = (re.findall(r"^  - (\S+)$", text, re.M), re.findall(r"^\| `([^`]+)` \|", text, re.M))
        return found

    if args[0] == "check":
        if os.environ.get("FAKE_CHECK") == "fail":
            print(json.dumps({"errors": [{"code": "link.unresolved", "file": ".markitect/model/project.md", "message": "no such Definition"}]}))
            sys.exit(1)
        listed = {f for _, rows in pages().values() for f in rows}
        root = repo / ".markitect" / "registry.md"
        if root.exists():
            listed |= set(re.findall(r"^\| `([^`]+)` \|", root.read_text(encoding="utf-8"), re.M))
        errors = []
        for f in git("ls-files").splitlines():
            if f.startswith(".markitect/") or f in listed:
                continue
            owner, best = ".markitect/registry.md", ""
            for page, (paths, _) in pages().items():
                for p in paths:
                    # A trailing /** owns what lies below the folder; a pattern without wildcards names one file.
                    hit = f.startswith(p[:-3] + "/") if p.endswith("/**") else f == p
                    if hit and (len(p) > len(best) or not p.endswith("/**")):
                        owner, best = page, p
            name = "Root" if owner == ".markitect/registry.md" else owner[len(".markitect/registry/"):-3]
            errors.append({"code": "file.unregistered", "file": f,
                           "message": f"neither registered nor ignored: list it on {owner}, the page of its owner {name}, or add it to the ignore page"})
        print(json.dumps({"errors": errors}))
        sys.exit(1 if errors else 0)

    d = pathlib.Path(os.environ["FAKE_DIR"])
    n = len(list(d.glob("apply-*.json"))) + 1
    env = {k: v for k, v in os.environ.items() if k.startswith(("MARKITECT_", "ANTHROPIC_", "CLAUDE_", "OPENAI_", "CODEX_", "GH_", "GITHUB_", "GIT_"))}
    policy = repo / ".markitect" / "model" / "lending" / "loan-policy.md"
    (d / f"apply-{n}.json").write_text(json.dumps({"args": args, "env": env, "head": git("rev-parse", "HEAD"),
                                                    "policy": git("show", "HEAD:.markitect/model/lending/loan-policy.md") if policy.exists() else None,
                                                    "lendingPage": (repo / ".markitect" / "registry" / "lending.md").read_text(encoding="utf-8")
                                                    if (repo / ".markitect" / "registry" / "lending.md").exists() else None,
                                                    "model": sorted(p.relative_to(repo).as_posix() for p in (repo / ".markitect").rglob("*") if p.is_file())}))
    mode, run = os.environ.get("FAKE_APPLY", "land"), opt("--run")
    if mode == "none":
        print("markitect: no Manager image", file=sys.stderr)
        sys.exit(2)
    agent = "claude:claude-sonnet-5-5"
    starts = [{"manager": ".", "role": "maker", "round": 1, "reason": "finished", "agent": agent},
              {"manager": "lending", "role": "maker", "round": 1, "reason": "finished", "agent": agent}]
    item = {"manager": "lending", "item": "lending/loan-policy", "part": "own", "verdict": "changed", "evidence": "loans.go enforces it",
            "checker": "lending/checker", "approved": True}
    result = {"format": 1, "run": run, "command": "apply", "items": [item], "requiredChecks": {"ran": True, "wholeRepository": True},
              "notParticipating": [], "findings": [], "starts": starts, "usage": {"input": 1000, "output": 100}, "cleanup": []}
    if mode == "land":
        work = d / f"land-{n}"
        git("clone", "-q", git("remote", "get-url", "origin"), str(work), cwd=d)
        (work / "realization").mkdir(exist_ok=True)
        (work / "realization" / (run + ".txt")).write_text(run + "\n")
        first = sorted((work / ".markitect" / "registry").rglob("*.md"))[0]
        first.write_text(first.read_text(encoding="utf-8") + f"| `realization/{run}.txt` | | made by the stand-in |\n", encoding="utf-8")
        (work / ".markitect" / "results").mkdir(exist_ok=True)
        (work / ".markitect" / "results" / (run + ".md")).write_text("# Result\n")
        ident = ["-c", "user.name=Markitect", "-c", "user.email=markitect@markitect.invalid"]
        git(*ident, "add", "-A", cwd=work)
        git(*ident, "commit", "-q", "-m", "Run " + run, cwd=work)
        git("push", "-q", "origin", "HEAD:main", cwd=work)
        result.update(outcome="landed", state="landed", why="landed", landed=git("rev-parse", "HEAD", cwd=work))
    elif mode == "defect":
        result.update(outcome="not-landed", state="model-defect", why="a model defect ended the Run at Root", items=[])
    elif mode == "gate":
        result.update(outcome="not-landed", state="check-failed", why="the project does not compile", items=[], starts=[])
    else:
        result.update(outcome="not-landed", state="incomplete", why="Manager lending's maker made no progress in 3 starts in a row")
        if mode == "provider":
            starts.append({"manager": "lending", "role": "maker", "round": 2, "reason": "failed", "class": "rate-limit", "detail": "429"})
        elif mode == "machinery":
            result["starts"] = [{"manager": ".", "role": "maker", "round": 1, "reason": "failed", "class": "other", "detail": "container start"}]
        elif mode == "recovered":
            starts.insert(1, {"manager": "lending", "role": "maker", "round": 0, "reason": "failed", "class": "rate-limit", "detail": "429"})
    print(json.dumps(result, indent=2))
    sys.exit(0 if result["outcome"] == "landed" else 1)
''')


class ArmTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = pathlib.Path(self._tmp.name)
        (self.tmp / "fake").mkdir()
        script = self.tmp / "standin.py"
        script.write_text(STANDIN)
        if os.name == "nt":
            self.markitect = self.tmp / "markitect.cmd"
            self.markitect.write_text(f'@"{sys.executable}" -B "{script}" %*\n')
        else:
            self.markitect = self.tmp / "markitect"
            self.markitect.write_text(f'#!/bin/sh\nexec "{sys.executable}" -B "{script}" "$@"\n')
            self.markitect.chmod(0o755)
        login = {"claudeAiOauth": {"accessToken": "sk-ant-oat-arm-secret", "expiresAt": int((time.time() + 7200) * 1000)}}
        (self.tmp / "login.json").write_text(json.dumps(login))
        self._env = dict(os.environ)
        os.environ.update(FAKE_DIR=str(self.tmp / "fake"), EVALUATION_LOGIN=str(self.tmp / "login.json"), EVALUATION_NETWORK="eval-x-internal",
                          EVALUATION_HOSTS="api.anthropic.com:10.9.0.2", EVALUATION_DNS="192.0.2.1", EVALUATION_ENGINE=json.dumps(["docker"]),
                          EVALUATION_LABEL="markitect.evaluation.arm=eval-0123456789ab", ANTHROPIC_BASE_URL="http://127.0.0.1:9/leak",
                          CLAUDE_CODE_ENTRYPOINT="desktop", MARKITECT_MODEL="claude-opus-leak", OPENAI_API_KEY="sk-leak", CODEX_HOME="leak",
                          GH_TOKEN="leak", GITHUB_TOKEN="leak", GIT_LEAK_PROBE="leak")
        self.variant = [sys.executable, "-B", str(HERE / "markitect_arm.py"), "--markitect", str(self.markitect),
                        "--remote-binary", "markitect-remote", "--manager-image", "sha256:" + "a" * 64]

    def tearDown(self):
        os.environ.clear()
        os.environ.update(self._env)
        self._tmp.cleanup()

    def model(self, name, text="# Policy\n\nThe limit is 5.\n"):
        model = self.tmp / name
        (model / "model" / "lending").mkdir(parents=True)
        (model / "manifest.yaml").write_text("libraries:\n  std: 1.0.0\n")
        (model / "lock.yaml").write_text("lock\n")
        (model / "execution.yaml").write_text("checks: []\n")
        (model / "model" / "project.md").write_text("# Project\n")
        (model / "model" / "lending" / "loan-policy.md").write_text(text)
        (model / "registry").mkdir()
        (model / "registry" / "lending.md").write_text("---\npaths:\n  - realization/internal/lending/**\n---\n# Registry: lending\n\n"
                                                       "| File | Realizes | Why |\n|---|---|---|\n")
        return model

    def task(self, name, model, start=None, base_model=None):
        (self.tmp / (name + ".md")).write_text("Build the library.\n")
        suite.make_task(self.tmp / (name + ".md"), model, start, 600, self.tmp / name, base_model)
        return self.tmp / name

    def applies(self):
        return [json.loads((self.tmp / "fake" / f"apply-{i}.json").read_text()) for i in range(1, len(list((self.tmp / "fake").glob("apply-*.json"))) + 1)]

    def run_arm(self, task, mode="land", state=None):
        os.environ["FAKE_APPLY"] = mode
        out = self.tmp / ("run-" + task.name + "-" + mode)
        return suite.run(task, self.variant, out, MODEL, state), out

    def tree(self, bundle, commit):
        repo = self.tmp / ("look-" + commit[:8])
        if not repo.exists():
            subprocess.run(["git", "init", "-q", "--bare", str(repo)], check=True)
            subprocess.run(["git", "-C", str(repo), "fetch", "-q", str(bundle), "+refs/heads/*:refs/heads/*"], check=True, capture_output=True)
        return suite.git(repo, "ls-tree", "-r", "--name-only", commit).splitlines()

    def test_a_fresh_build_lands_and_delivers_the_landing(self):
        state = self.tmp / "state"
        state.mkdir()
        record, out = self.run_arm(self.task("fresh", self.model("m1")), state=state)
        self.assertTrue(record["delivered"], record)
        self.assertEqual(record["exitCode"], 0, (out / "variant.stderr.log").read_text())
        changed = [line.split("\t")[1] for line in record["result"]["changedPaths"]]
        for path in (".markitect/model/lending/loan-policy.md", ".markitect/registry/lending.md", "realization/eval-0123456789ab.txt",
                     ".markitect/results/eval-0123456789ab.md"):
            self.assertIn(path, changed)
        apply = self.applies()[0]
        self.assertIn("--full", apply["args"])
        self.assertEqual(apply["args"][apply["args"].index("--run") + 1], "eval-0123456789ab")
        env = apply["env"]
        self.assertEqual((env["MARKITECT_NETWORK"], env["MARKITECT_HOSTS"], env["MARKITECT_DNS"], env["MARKITECT_ENGINE"]),
                         ("eval-x-internal", "api.anthropic.com:10.9.0.2", "192.0.2.1", "docker"))
        self.assertEqual((env["MARKITECT_PROVIDER"], env["MARKITECT_MODEL"], env["MARKITECT_LOGIN"]),
                         ("claude", "claude-sonnet-5-5", str(self.tmp / "login.json")))
        self.assertEqual(env["MARKITECT_TRANSCRIPTS"], str(out / "variant" / "transcripts"))
        self.assertFalse([k for k in env if k.startswith(("ANTHROPIC_", "CLAUDE_", "OPENAI_", "CODEX_", "GH_", "GITHUB_")) or k == "GIT_LEAK_PROBE"], env)
        # The owner's commit is the task's model, before the Run.
        self.assertIn(".markitect/registry/lending.md", apply["model"])
        self.assertTrue(record["modelMatches"])
        self.assertEqual((record["reported"]["status"], record["reported"]["mode"]), ("landed", "fresh"))
        self.assertTrue(results.check_run(out)["complete"], results.check_run(out))
        self.assertEqual(json.loads((state / "markitect-arm.json").read_text())["landed"], record["result"]["commit"])
        self.assertEqual(classify.classify(out)["class"], "invalid")  # host mode: for tests only

    def test_a_chain_step_changes_the_model_from_the_last_landing(self):
        steps = []
        for name, text in (("one", "# Policy\n\nThe limit is 5.\n"), ("two", "# Policy\n\nThe limit is 3.\n")):
            (self.tmp / "steps" / name).mkdir(parents=True)
            (self.tmp / "steps" / name / "task.md").write_text("Build.\n")
            self.model("steps/" + name + "/model", text)
            steps.append(self.tmp / "steps" / name)
        os.environ["FAKE_APPLY"] = "land"
        summary = suite.chain(steps, None, 600, self.variant, self.tmp / "chain", MODEL)
        self.assertEqual([s["delivered"] for s in summary["steps"]], [True, True], summary)
        first, second = self.applies()
        self.assertIn("--full", first["args"])
        self.assertNotIn("--full", second["args"])
        self.assertNotIn("--base", second["args"])
        record = json.loads((self.tmp / "chain" / "step-02" / "run" / "suite.json").read_text())
        self.assertEqual(record["reported"]["mode"], "change")
        changed = [line.split("\t")[1] for line in record["result"]["changedPaths"]]
        self.assertIn(".markitect/model/lending/loan-policy.md", changed)
        # The registry page keeps the row the first Run wrote: its table is its Manager's.
        repo = self.tmp / "look"
        subprocess.run(["git", "init", "-q", "--bare", str(repo)], check=True)
        bundle = self.tmp / "chain" / "step-02" / "run" / "variant" / "result.bundle"
        subprocess.run(["git", "-C", str(repo), "fetch", "-q", str(bundle), "+refs/heads/*:refs/heads/*"], check=True, capture_output=True)
        page = suite.git(repo, "show", record["result"]["commit"] + ":.markitect/registry/lending.md")
        self.assertEqual(page.count("made by the stand-in"), 2)

    def test_a_run_that_does_not_land_delivers_its_start(self):
        for mode, code in (("defect", 0), ("incomplete", 1)):
            with self.subTest(mode=mode):
                record, out = self.run_arm(self.task("stop-" + mode, self.model("model-" + mode)), mode)
                self.assertTrue(record["delivered"])
                self.assertEqual(record["result"]["changedPaths"], [])
                self.assertEqual(record["exitCode"], code)
                self.assertTrue((out / "variant" / "results" / "eval-0123456789ab.json").is_file())
                self.assertEqual(results.check_run(out)["problems"], [])

    def test_a_provider_failure_counts_only_when_it_ended_a_managers_work(self):
        record, out = self.run_arm(self.task("provider", self.model("model-p")), "provider")
        record["conditions"], record["validForCounting"] = "native", True
        (out / "suite.json").write_text(json.dumps(record))
        verdict = classify.classify(out)
        self.assertEqual((verdict["class"], verdict["kind"]), ("void", "provider"))
        self.assertEqual(verdict["evidence"][0]["manager"], "lending")
        record, out = self.run_arm(self.task("recovered", self.model("model-r")), "recovered")
        record["conditions"], record["validForCounting"] = "native", True
        (out / "suite.json").write_text(json.dumps(record))
        self.assertEqual(classify.classify(out)["class"], "result")

    def test_a_reference_start_is_adopted_then_the_model_changes(self):
        start = self.tmp / "reference"
        subprocess.run(["git", "init", "-q", "-b", "start", str(start)], check=True)
        for rel in ("realization/go.mod", "realization/internal/lending/loans.go", "realization/internal/library/library.go"):
            (start / rel).parent.mkdir(parents=True, exist_ok=True)
            (start / rel).write_text(rel + "\n")
        env = dict(os.environ, GIT_AUTHOR_NAME="r", GIT_AUTHOR_EMAIL="r@x", GIT_COMMITTER_NAME="r", GIT_COMMITTER_EMAIL="r@x")
        subprocess.run(["git", "-C", str(start), "add", "-A"], check=True, env=env)
        subprocess.run(["git", "-C", str(start), "-c", "commit.gpgsign=false", "commit", "-q", "-m", "reference"], check=True, env=env)
        subprocess.run(["git", "-C", str(start), "bundle", "create", "-q", str(self.tmp / "reference.bundle"), "start"], check=True)
        # Both models' files have the same times and sizes, as a checkout leaves them: the change must still be committed.
        models = [self.model("m2", "# Policy\n\nThe limit is 3.\n"), self.model("m1")]
        for f in (f for m in models for f in m.rglob("*") if f.is_file()):
            os.utime(f, (1_700_000_000, 1_700_000_000))
        task = self.task("adopt", *models[:1], self.tmp / "reference.bundle", models[1])
        record, out = self.run_arm(task, "defect")
        self.assertEqual(record["exitCode"], 0, (out / "variant" / "run.json").read_text())
        apply = self.applies()[0]
        self.assertIn("The limit is 3.", apply["policy"])
        adoption = record["reported"]["adoption"]
        self.assertEqual(apply["args"][apply["args"].index("--base") + 1], adoption["commit"])
        self.assertEqual((adoption["unlinkedRows"], adoption["pageRows"], adoption["registry"]), (3, 0, "unlinked"))
        self.assertNotEqual(record["reported"]["ownerCommit"], adoption["commit"])
        # A stop delivers the reference start itself, without the owner's commits.
        self.assertEqual(record["result"]["commit"], json.loads((task / "task.json").read_text())["start"]["commit"])
        # Without a base model, the adopted start runs in full.
        task = self.task("adopt-full", self.model("m2b"), self.tmp / "reference.bundle")
        self.run_arm(task, "defect")
        self.assertIn("--full", self.applies()[1]["args"])

    def test_a_host_that_cannot_run_delivers_nothing(self):
        record, out = self.run_arm(self.task("none", self.model("model-n")), "none")
        self.assertFalse(record["delivered"])
        self.assertEqual(record["exitCode"], 2)
        self.assertEqual(record["reported"]["status"], "could-not-run")
        self.assertFalse((out / "variant" / "results" / "eval-0123456789ab.json").exists())

    def test_what_is_not_the_arms_work_never_scores_as_a_stop(self):
        # A gate that fails, with no agent started: nothing is delivered, so no unchanged start can count as a stop.
        record, out = self.run_arm(self.task("gate", self.model("model-g")), "gate")
        self.assertEqual((record["delivered"], record["exitCode"], record["reported"]["status"]), (False, 2, "check-failed"))
        # The owner's input does not compile: apply never runs.
        os.environ["FAKE_CHECK"] = "fail"
        record, _ = self.run_arm(self.task("broken", self.model("model-b")))
        self.assertEqual((record["delivered"], record["exitCode"]), (False, 2))
        self.assertIn("the owner's input does not compile", record["reported"]["cause"])
        self.assertEqual(len(self.applies()), 1)
        # A Run that stopped before any agent completed a turn.
        del os.environ["FAKE_CHECK"]
        record, _ = self.run_arm(self.task("machinery", self.model("model-m")), "machinery")
        self.assertEqual((record["delivered"], record["exitCode"], record["reported"]["noAgentTurn"]), (False, 2, True))

    def test_an_owners_pages_adopt_the_reference_start_with_links(self):
        start = self.tmp / "reference"
        subprocess.run(["git", "init", "-q", "-b", "start", str(start)], check=True)
        for rel in ("realization/go.mod", "realization/internal/library/loans.go", "realization/internal/library/library.go"):
            (start / rel).parent.mkdir(parents=True, exist_ok=True)
            (start / rel).write_text(rel + "\n")
        env = dict(os.environ, GIT_AUTHOR_NAME="r", GIT_AUTHOR_EMAIL="r@x", GIT_COMMITTER_NAME="r", GIT_COMMITTER_EMAIL="r@x")
        subprocess.run(["git", "-C", str(start), "add", "-A"], check=True, env=env)
        subprocess.run(["git", "-C", str(start), "-c", "commit.gpgsign=false", "commit", "-q", "-m", "reference"], check=True, env=env)
        subprocess.run(["git", "-C", str(start), "bundle", "create", "-q", str(self.tmp / "reference.bundle"), "start"], check=True)
        # The owner's pages for this reference: one file on lending's page by a single-file pattern, two on Root's.
        base = self.model("paged-base")
        (base / "registry" / "lending.md").write_text(
            "---\npaths:\n  - realization/internal/library/loans.go\n---\n# Registry: lending\n\n| File | Realizes | Why |\n|---|---|---|\n"
            "| `realization/internal/library/loans.go` | [[lending/loan-policy]] | Lending operations |\n")
        (base / "registry.md").write_text("# Registry\n\n| File | Realizes | Why |\n|---|---|---|\n"
                                          "| `realization/go.mod` | | The Go module |\n"
                                          "| `realization/internal/library/library.go` | [[project]] | Records |\n")
        record, out = self.run_arm(self.task("paged", self.model("m2"), self.tmp / "reference.bundle", base), "defect")
        self.assertEqual(record["exitCode"], 0, (out / "variant" / "run.json").read_text())
        adoption = record["reported"]["adoption"]
        self.assertEqual((adoption["unlinkedRows"], adoption["pageRows"], adoption["registry"]), (0, 3, "pages"))
        apply = self.applies()[0]
        self.assertIn(".markitect/registry.md", apply["model"])
        self.assertIn("[[lending/loan-policy]]", apply["lendingPage"])
        self.assertNotIn("adopted by the owner", apply["lendingPage"])

    def test_a_base_model_without_registry_pages_still_places_each_file_on_its_owners_page(self):
        start = self.tmp / "reference"
        subprocess.run(["git", "init", "-q", "-b", "start", str(start)], check=True)
        (start / "realization" / "internal" / "lending").mkdir(parents=True)
        (start / "realization" / "internal" / "lending" / "loans.go").write_text("package lending\n")
        env = dict(os.environ, GIT_AUTHOR_NAME="r", GIT_AUTHOR_EMAIL="r@x", GIT_COMMITTER_NAME="r", GIT_COMMITTER_EMAIL="r@x")
        subprocess.run(["git", "-C", str(start), "add", "-A"], check=True, env=env)
        subprocess.run(["git", "-C", str(start), "-c", "commit.gpgsign=false", "commit", "-q", "-m", "reference"], check=True, env=env)
        subprocess.run(["git", "-C", str(start), "bundle", "create", "-q", str(self.tmp / "reference.bundle"), "start"], check=True)
        base = self.model("bare-base")
        for page in (base / "registry").rglob("*.md"):
            page.unlink()
        record, out = self.run_arm(self.task("bare", self.model("m2"), self.tmp / "reference.bundle", base), "defect")
        self.assertEqual(record["exitCode"], 0, (out / "variant" / "run.json").read_text())
        self.assertEqual(self.applies()[0]["model"].count(".markitect/registry/lending.md"), 1)
        self.assertNotIn(".markitect/registry.md", self.applies()[0]["model"])

    def test_only_agents_that_ran_name_the_model(self):
        import markitect_arm
        starts = [{"agent": "claude:claude-sonnet-5-5", "reason": "failed"}, {"agent": "claude:claude-sonnet-5-5", "reason": "deadline"},
                  {"agent": "claude:claude-sonnet-5-5-20261001", "reason": "finished"}]
        self.assertEqual(markitect_arm.reported_model({"starts": starts}, "claude", "claude-sonnet-5-5"), ("claude", "claude-sonnet-5-5-20261001"))
        starts.append({"agent": "claude:claude-opus-5-5", "reason": "finished"})
        self.assertTrue(markitect_arm.reported_model({"starts": starts}, "claude", "x")[1].startswith("mixed"))
        self.assertEqual(markitect_arm.reported_model({"starts": []}, "claude", "x"), ("claude", "x"))

    def test_the_arm_refuses_without_a_model_or_the_suites_login(self):
        os.environ.pop("EVALUATION_LOGIN")
        record, _ = self.run_arm(self.task("nologin", self.model("model-l")))
        self.assertFalse(record["delivered"])
        self.assertIn("EVALUATION_LOGIN", record["reported"]["cause"])
        os.environ["EVALUATION_LOGIN"] = str(self.tmp / "login.json")
        record, _ = self.run_arm(self.task("nomodel", None))
        self.assertIn("no model/", record["reported"]["cause"])


if __name__ == "__main__":
    unittest.main()
