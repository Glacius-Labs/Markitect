"""Offline tests of the concept 3.0 arm: real Git, a scripted `markitect` stand-in, no engine and no provider."""
import json
import os
import pathlib
import subprocess
import sys
import tempfile
import textwrap
import unittest

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import classify  # noqa: E402
import suite  # noqa: E402

MODEL = "claude:claude-sonnet-5-5"
# A token in the shape `claude setup-token` makes. It must reach the host and nothing the arm writes.
TOKEN = "sk-ant-oat01-" + "t" * 48

# A scripted `markitect`. `check --json` reports every file outside .markitect/ that no registry table lists, on the
# page of the Package whose `DIR/**` pattern is the longest match (Root's registry.md otherwise), as the compiler words
# it. `apply` records its arguments and environment in FAKE_DIR/apply-N.json and acts as FAKE_APPLY says, with a
# Result of format 2 (internal/run/result.go):
#   land        commits realization/<run>.txt, lists it on the first Package's page, adds the Result page, pushes the
#               commit to the target's main and prints a landed Result
#   spec        a Command that ends as `spec-problem`: the implementer stopped, the spec needs the owner
#   nothing     a Command that finds nothing affected: `nothing-to-do`, no session
#   incomplete  a Command that stops without approved work; its sessions finished
#   provider    as incomplete, but the reviewer's session failed on a rate limit
#   other       as incomplete, but the reviewer's session failed with the class `other`, after a finished implementer
#   machinery   as incomplete, whose only session failed before any agent turn
#   refused     the repair rounds ran out with the required checks still failing: `refused`
#   kept        the work passed but cannot land: `kept`
#   gate        a Command whose start does not compile for the host: `check-failed`, no session
#   gate-late   `check-failed` after a session finished (the engine does not do this; the arm decides by the sessions)
#   models      a landing whose reviewer's session answered with a second model
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
    env = {k: v for k, v in os.environ.items() if k.startswith(("MARKITECT_", "EVALUATION_", "ANTHROPIC_", "CLAUDE_", "OPENAI_", "CODEX_", "GH_", "GITHUB_", "GIT_"))}
    policy = repo / ".markitect" / "model" / "lending" / "loan-policy.md"
    (d / f"apply-{n}.json").write_text(json.dumps({"args": args, "env": env, "head": git("rev-parse", "HEAD"),
                                                    "policy": git("show", "HEAD:.markitect/model/lending/loan-policy.md") if policy.exists() else None,
                                                    "sinceBase": git("diff", "--name-only", opt("--base"), "HEAD").splitlines() if opt("--base") else None,
                                                    "lendingPage": (repo / ".markitect" / "registry" / "lending.md").read_text(encoding="utf-8")
                                                    if (repo / ".markitect" / "registry" / "lending.md").exists() else None,
                                                    "model": sorted(p.relative_to(repo).as_posix() for p in (repo / ".markitect").rglob("*") if p.is_file())}))
    mode, run = os.environ.get("FAKE_APPLY", "land"), opt("--run")
    if mode == "none":
        print("markitect: no Manager image", file=sys.stderr)
        sys.exit(2)
    agent = "claude:claude-sonnet-5-5"
    done = lambda role, **more: {"role": role, "round": 1, "reason": "finished", "agent": agent, "models": ["claude-sonnet-5-5"],
                                 "usage": {"input": 500, "output": 50}, "seconds": 1.5, **more}
    sessions = [done("implementer"), done("reviewer")]
    item = {"item": "lending/loan-policy", "kind": "reached", "definition": "lending/loan-policy", "answer": "ok",
            "pointers": [{"file": "realization/loans.go", "line": 42}]}
    result = {"format": 2, "run": run, "command": "apply", "outcome": "not-landed", "target": "main", "full": "--full" in args,
              "items": [item], "requiredChecks": {"ran": True, "checks": []}, "ownerDecisions": [], "findings": [], "repairs": 0,
              "sessions": sessions, "usage": {"input": 1000, "output": 100}, "cleanup": []}
    if args[0] == "verify":
        # verify: FAKE_APPLY "verified" or "findings" checks every item; the other modes end as for apply.
        result.update(command="verify", outcome="not-landed")
        if mode in ("verified", "findings"):
            result.update(state=mode, why=mode, findings=[] if mode == "verified" else [
                {"definition": "lending/loan-policy", "observed": "no limit", "expected": "the limit is 5"}])
            print(json.dumps(result, indent=2))
            sys.exit(0 if mode == "verified" else 1)
    if mode in ("land", "models"):
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
        # FAKE_AFTER: "left" reports a branch the cleanup after landing could not delete; "missing" is a host that
        # predates afterLanding; otherwise a clean landing.
        after = os.environ.get("FAKE_AFTER", "")
        if after == "malformed":
            result.update(afterLanding="not a list", cleanup="not a list either")
        elif after != "missing":
            result["afterLanding"] = [f"the landed Run's branch markitect/runs/{run} could not be deleted: refused"] if after == "left" else []
        if after == "left":
            result["cleanup"] = ["the Run's remote may remain if its start failed half-way; Sweep removes it"]
        if mode == "models":
            sessions[1]["models"] = ["claude-sonnet-5-5", "claude-haiku-5"]
    elif mode == "spec":
        result.update(state="spec-problem", why="the implementer stopped: the spec needs the owner", sessions=sessions[:1],
                      ownerDecisions=[{"matter": "the limit is 5 and 3", "recommendation": "Settle it in the spec."}])
    elif mode == "nothing":
        result.update(state="nothing-to-do", why="nothing changed that reaches the spec; no agent starts", items=[], sessions=[])
    elif mode == "refused":
        result.update(state="refused", why="the repair rounds ran out, and these stay open: the required check go test fails", repairs=2,
                      requiredChecks={"ran": False, "checks": [{"name": "go test", "passed": False}]})
    elif mode == "kept":
        result.update(state="kept", why="main moved and the merge conflicts in realization/loans.go", branch="markitect/runs/" + run)
    elif mode in ("gate", "gate-late"):
        result.update(state="check-failed", why="check fails with 1 compile errors, so no agent starts", items=[],
                      sessions=[] if mode == "gate" else sessions[:1])
    else:
        result.update(state="incomplete", why="the reviewer's session ended as failed")
        if mode == "provider":
            sessions[1].update(reason="failed", **{"class": "rate-limit", "detail": "429"})
        elif mode == "other":
            sessions[1].update(reason="failed", **{"class": "other", "detail": "the launch broke its contract"})
        elif mode == "machinery":
            result["sessions"] = [{"role": "implementer", "round": 1, "reason": "failed", "class": "other", "detail": "container start",
                                   "usage": {}, "seconds": 0.1}]
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
        self._env = dict(os.environ)
        # The suite still sets EVALUATION_LOGIN for its native runs; the arm does not use it.
        os.environ.update(FAKE_DIR=str(self.tmp / "fake"), MARKITECT_CLAUDE_TOKEN=TOKEN, EVALUATION_LOGIN=str(self.tmp / "login.json"),
                          EVALUATION_NETWORK="eval-x-internal",
                          EVALUATION_HOSTS="api.anthropic.com:10.9.0.2", EVALUATION_DNS="192.0.2.1", EVALUATION_ENGINE=json.dumps(["docker"]),
                          EVALUATION_LABEL="markitect.evaluation.arm=eval-0123456789ab", ANTHROPIC_BASE_URL="http://127.0.0.1:9/leak",
                          CLAUDE_CODE_ENTRYPOINT="desktop", MARKITECT_MODEL="claude-opus-leak", OPENAI_API_KEY="sk-leak", CODEX_HOME="leak",
                          MARKITECT_SESSION_BUDGET="1", GH_TOKEN="leak", GITHUB_TOKEN="leak", GIT_LEAK_PROBE="leak")
        self.variant = [sys.executable, "-B", str(HERE / "markitect_arm.py"), "--markitect", str(self.markitect),
                        "--image", "sha256:" + "a" * 64]

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

    def test_what_the_host_left_after_landing_is_recorded_and_its_absence_is_not_clean(self):
        # Only afterLanding counts as leftovers: `cleanup` says what may remain, and the engine's census finds what does.
        for after, recorded, reported, leftovers in (("", [], True, 0), ("left", 1, True, 1), ("missing", None, False, 0),
                                                     ("malformed", None, False, 0)):
            with self.subTest(after=after):
                os.environ["FAKE_AFTER"] = after
                record, _ = self.run_arm(self.task("after-" + (after or "clean"), self.model("model-a-" + (after or "clean"))))
                rep = record["reported"]
                self.assertTrue(record["delivered"])
                self.assertEqual(len(rep["afterLanding"]) if isinstance(recorded, int) else rep["afterLanding"], recorded)
                self.assertEqual((rep["afterLandingReported"], len(rep["leftovers"])), (reported, leftovers))
        # A Run that does not land has nothing after landing to report.
        record, _ = self.run_arm(self.task("after-stop", self.model("model-a-stop")), "spec")
        self.assertEqual((record["reported"]["afterLanding"], record["reported"]["afterLandingReported"]), (None, True))

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
        self.assertEqual((env["MARKITECT_PROVIDER"], env["MARKITECT_MODEL"], env["MARKITECT_MANAGER_IMAGE"]),
                         ("claude", "claude-sonnet-5-5", "sha256:" + "a" * 64))
        self.assertEqual(env["MARKITECT_TRANSCRIPTS"], str(out / "variant" / "transcripts"))
        # The owner's token is the login. The suite's login copy, a remote binary and the caller's other variables do not reach the host.
        self.assertEqual(env["MARKITECT_CLAUDE_TOKEN"], TOKEN)
        self.assertFalse([k for k in env if k.startswith(("EVALUATION_", "ANTHROPIC_", "CLAUDE_", "OPENAI_", "CODEX_", "GH_", "GITHUB_"))
                          or k in ("GIT_LEAK_PROBE", "MARKITECT_LOGIN", "MARKITECT_REMOTE_BINARY", "MARKITECT_SESSION_BUDGET")], env)
        # The owner's commit is the task's model, before the Run.
        self.assertIn(".markitect/registry/lending.md", apply["model"])
        self.assertTrue(record["modelMatches"])
        self.assertEqual((record["reported"]["status"], record["reported"]["mode"]), ("landed", "fresh"))
        self.assertEqual((record["reported"]["starts"], record["reported"]["failedStarts"], record["reported"]["tokens"]),
                         (2, [], {"input": 1000, "output": 100}))
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
        # The states a Command ends on purpose exit 0; the others are the agents' work and exit 1, both with the start delivered.
        for mode, code in (("spec", 0), ("nothing", 0), ("incomplete", 1), ("refused", 1), ("kept", 1)):
            with self.subTest(mode=mode):
                record, out = self.run_arm(self.task("stop-" + mode, self.model("model-" + mode)), mode)
                self.assertTrue(record["delivered"], record)
                self.assertEqual(record["result"]["changedPaths"], [])
                self.assertEqual(record["exitCode"], code)
                self.assertTrue((out / "variant" / "results" / "eval-0123456789ab.json").is_file())
                self.assertNotIn("noAgentTurn", record["reported"])

    def test_a_provider_failure_voids_the_run_and_any_other_failure_is_the_agents_work(self):
        record, out = self.run_arm(self.task("provider", self.model("model-p")), "provider")
        record["conditions"], record["validForCounting"] = "native", True
        (out / "suite.json").write_text(json.dumps(record))
        verdict = classify.classify(out)
        self.assertEqual((verdict["class"], verdict["kind"]), ("void", "provider"))
        self.assertEqual((verdict["evidence"][0]["role"], verdict["evidence"][0]["error"]), ("reviewer", "rate-limit"))
        self.assertEqual(record["reported"]["failedStarts"], [{"role": "reviewer", "round": 1, "class": "rate-limit", "detail": "429"}])
        record, out = self.run_arm(self.task("other", self.model("model-o")), "other")
        record["conditions"], record["validForCounting"] = "native", True
        (out / "suite.json").write_text(json.dumps(record))
        self.assertEqual((record["delivered"], record["exitCode"]), (True, 1))
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
        record, out = self.run_arm(task, "spec")
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
        self.run_arm(task, "spec")
        self.assertIn("--full", self.applies()[1]["args"])

    def verify(self, task, mode):
        os.environ["FAKE_APPLY"] = mode
        out = self.tmp / ("verify-" + task.name + "-" + mode)
        return suite.run(task, [*self.variant, "--verify"], out, MODEL, None), out

    def test_verify_checks_an_end_state_with_its_model_and_delivers_it_unchanged(self):
        record, out = self.run_arm(self.task("m-end", self.model("m-end-model")))
        landed = out / "variant" / "result.bundle"
        task = self.task("verify-m", self.model("m-end-model-2"), landed)
        record, out = self.verify(task, "verified")
        self.assertEqual(record["exitCode"], 0, (out / "variant" / "run.json").read_text())
        self.assertTrue(record["delivered"])
        self.assertEqual(record["result"]["changedPaths"], [])
        call = self.applies()[-1]
        self.assertEqual((call["args"][0], "--full" in call["args"], "--base" in call["args"]), ("verify", True, False))
        rep = record["reported"]
        self.assertEqual((rep["command"], rep["mode"], rep["adoption"], rep["findings"]), ("verify", "verify", None, 0))

    def test_verify_adopts_an_end_state_without_a_model_first(self):
        start = self.tmp / "k-end"
        subprocess.run(["git", "init", "-q", "-b", "start", str(start)], check=True)
        for rel in ("realization/go.mod", "realization/internal/lending/loans.go", "README.md"):
            (start / rel).parent.mkdir(parents=True, exist_ok=True)
            (start / rel).write_text(rel + "\n")
        env = dict(os.environ, GIT_AUTHOR_NAME="k", GIT_AUTHOR_EMAIL="k@x", GIT_COMMITTER_NAME="k", GIT_COMMITTER_EMAIL="k@x")
        subprocess.run(["git", "-C", str(start), "add", "-A"], check=True, env=env)
        subprocess.run(["git", "-C", str(start), "-c", "commit.gpgsign=false", "commit", "-q", "-m", "K's end state"], check=True, env=env)
        subprocess.run(["git", "-C", str(start), "bundle", "create", "-q", str(self.tmp / "k-end.bundle"), "start"], check=True)
        task = self.task("verify-k", self.model("k-model"), self.tmp / "k-end.bundle")
        record, out = self.verify(task, "findings")
        self.assertEqual(record["exitCode"], 0, (out / "variant" / "run.json").read_text())
        rep = record["reported"]
        self.assertEqual((rep["mode"], rep["findings"], rep["status"]), ("verify-adopted", 1, "findings"))
        self.assertEqual((rep["adoption"]["unlinkedRows"], rep["adoption"]["pageRows"]), (3, 0))
        self.assertEqual(record["result"]["commit"], json.loads((task / "task.json").read_text())["start"]["commit"])
        self.assertIn("realization/internal/lending/loans.go", self.applies()[-1]["lendingPage"])

    def test_verify_that_does_not_check_every_item_is_no_verdict(self):
        record, _ = self.verify(self.task("verify-empty", self.model("e-model")), "incomplete")
        self.assertEqual((record["exitCode"], record["reported"]["mode"], record["reported"]["status"]), (1, "verify-empty", "incomplete"))
        record, _ = self.verify(self.task("verify-machinery", self.model("v-model")), "machinery")
        self.assertEqual((record["delivered"], record["exitCode"], record["reported"]["noAgentTurn"]), (False, 2, True))

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
        record, out = self.run_arm(self.task("paged", self.model("m2"), self.tmp / "reference.bundle", base), "spec")
        self.assertEqual(record["exitCode"], 0, (out / "variant" / "run.json").read_text())
        adoption = record["reported"]["adoption"]
        self.assertEqual((adoption["unlinkedRows"], adoption["pageRows"], adoption["registry"]), (0, 3, "pages"))
        apply = self.applies()[0]
        self.assertIn(".markitect/registry.md", apply["model"])
        self.assertIn("[[lending/loan-policy]]", apply["lendingPage"])
        self.assertNotIn("adopted by the owner", apply["lendingPage"])

    def test_an_adopted_start_gets_the_models_new_paths_and_its_files_keep_their_owners(self):
        start = self.tmp / "reference"
        subprocess.run(["git", "init", "-q", "-b", "start", str(start)], check=True)
        for rel in ("realization/cmd/main.go", "realization/internal/ms/loans.go", "realization/internal/ms/library.go"):
            (start / rel).parent.mkdir(parents=True, exist_ok=True)
            (start / rel).write_text(rel + "\n")
        env = dict(os.environ, GIT_AUTHOR_NAME="r", GIT_AUTHOR_EMAIL="r@x", GIT_COMMITTER_NAME="r", GIT_COMMITTER_EMAIL="r@x")
        subprocess.run(["git", "-C", str(start), "add", "-A"], check=True, env=env)
        subprocess.run(["git", "-C", str(start), "-c", "commit.gpgsign=false", "commit", "-q", "-m", "reference"], check=True, env=env)
        subprocess.run(["git", "-C", str(start), "bundle", "create", "-q", str(self.tmp / "reference.bundle"), "start"], check=True)
        # The reference is one package: the owner's pages name its files, Root keeps library.go, and the model wants modules.
        base = self.model("split-base")
        (base / "registry" / "lending.md").write_text(
            "---\npaths:\n  - realization/internal/ms/loans.go\n  - realization/cmd/main.go\n---\n# Registry: lending\n\n"
            "| File | Realizes | Why |\n|---|---|---|\n"
            "| `realization/internal/ms/loans.go` | [[lending/loan-policy]] | Lending operations |\n"
            "| `realization/cmd/main.go` | [[lending/loan-policy]] | The program |\n")
        (base / "registry.md").write_text("# Registry\n\n| File | Realizes | Why |\n|---|---|---|\n"
                                          "| `realization/internal/ms/library.go` | [[project]] | Records |\n")
        model = self.model("split")
        (model / "registry" / "lending.md").write_text(
            "---\npaths:\n  - realization/lending/**\n  - realization/cmd/**\n  - realization/internal/ms/**\n---\n# Registry: lending\n\n"
            "| File | Realizes | Why |\n|---|---|---|\n")
        record, out = self.run_arm(self.task("split-task", model, self.tmp / "reference.bundle", base), "spec")
        self.assertEqual(record["exitCode"], 0, (out / "variant" / "run.json").read_text())
        page = self.applies()[0]["lendingPage"]
        # A new folder, and a folder that holds only the page's own file, are added to what the page had.
        for pattern in ("realization/internal/ms/loans.go", "realization/cmd/main.go", "realization/lending/**", "realization/cmd/**"):
            self.assertIn(f"  - {pattern}\n", page)
        # This one would take library.go from Root.
        self.assertNotIn("realization/internal/ms/**", page)
        self.assertIn("[[lending/loan-policy]]", page)
        adoption = record["reported"]["adoption"]
        self.assertEqual((adoption["newPaths"], adoption["newPathsLeftOut"], adoption["unlinkedRows"], adoption["pageRows"]), (2, 1, 0, 3))
        # The patterns are part of the adoption, so the base holds them and the Run's impact is the model's change alone.
        args = self.applies()[0]["args"]
        self.assertEqual(args[args.index("--base") + 1], adoption["commit"])
        self.assertFalse([f for f in self.applies()[0]["sinceBase"] if "registry" in f])

    def test_the_arm_reads_patterns_as_the_registry_does(self):
        import markitect_arm
        hit = markitect_arm.pattern_matches
        self.assertTrue(hit("a/**", "a/b/c.go"))
        self.assertFalse(hit("a/**", "a"))
        self.assertTrue(hit("a/*.go", "a/b.go"))
        self.assertFalse(hit("a/*.go", "a/b/c.go"))
        self.assertTrue(hit("**/c.go", "c.go"))
        self.assertTrue(hit("a/**/c.go", "a/b/d/c.go"))
        self.assertTrue(hit("a/?.go", "a/b.go"))
        self.assertFalse(hit("a/b.go", "a/b.go/c"))
        self.assertEqual(markitect_arm.page_patterns("---\npaths: []\n---\n# x\n\n- not/a/pattern\n"), [])
        self.assertEqual(markitect_arm.page_patterns("# No front matter\n\n  - not/a/pattern\n"), [])
        self.assertEqual(markitect_arm.page_patterns("---\npaths:\n  - a/**\n  - b.go\n---\n  - not/a/pattern\n"), ["a/**", "b.go"])
        for other in ("---\npaths: [a, b]\n---\n", "---\npaths:\n- a\n---\n", "---\npaths:\n  - a # why\n---\n", "﻿---\npaths: []\n---\n"):
            self.assertIsNone(markitect_arm.page_patterns(other), other)
        self.assertEqual(markitect_arm.pattern_specificity("a/b.go"), 1 << 30)
        self.assertEqual(markitect_arm.pattern_specificity("a/b/**"), 2)
        pages = {"x": ["src/**"], "y": ["src/b/**"]}
        self.assertEqual([markitect_arm.file_owner(f, pages) for f in ("src/a.go", "src/b/c.go", "docs/d.md")], ["x", "y", None])

    def paged(self, name, **pages):
        """A folder of registry pages, each given as its front matter."""
        folder = self.tmp / name
        (folder / "registry").mkdir(parents=True)
        for page, front in pages.items():
            (folder / "registry" / (page + ".md")).write_text(front + "# Registry: " + page + "\n", encoding="utf-8")
        return folder

    def test_new_paths_never_move_a_file_or_repeat_a_pattern(self):
        import markitect_arm
        block = lambda *patterns: "---\npaths:\n" + "".join(f"  - {p}\n" for p in patterns) + "---\n"
        read = lambda folder, page: markitect_arm.page_patterns((folder / "registry" / (page + ".md")).read_text(encoding="utf-8"))
        # Another page already has the pattern, and no file matches it: adding it would be a compile error.
        dot = self.paged("dup-dot", a=block("x/**"), b="---\npaths: []\n---\n")
        self.assertEqual(markitect_arm.add_new_paths(self.paged("dup-model", b=block("x/**")), dot, []), (0, 1))
        self.assertEqual(read(dot, "b"), [])
        # The new pattern is more specific than the page holding the file, and would take it from there.
        dot = self.paged("move-dot", a=block("src/**"), b=block("src/b/**"))
        files = ["src/b/x/y.go"]
        self.assertEqual(markitect_arm.add_new_paths(self.paged("move-model", a=block("src/b/x/**")), dot, files), (0, 1))
        self.assertEqual(read(dot, "a"), ["src/**"])
        # The same for a file Root owns, and for a folder the page's own file is in, which it may take.
        dot = self.paged("root-dot", a=block("cmd/main.go"))
        files = ["cmd/main.go", "lib/root.go"]
        self.assertEqual(markitect_arm.add_new_paths(self.paged("root-model", a=block("cmd/**", "lib/**", "new/**")), dot, files), (2, 1))
        self.assertEqual(read(dot, "a"), ["cmd/main.go", "cmd/**", "new/**"])
        # A pattern the page already has is neither added nor left out; the first page to claim a new one keeps it.
        dot = self.paged("keep-dot", a=block("a/**"), b="---\npaths: []\n---\n")
        model = self.paged("keep-model", a=block("a/**", "shared/**"), b=block("shared/**", "b/**"))
        self.assertEqual(markitect_arm.add_new_paths(model, dot, []), (2, 1))
        self.assertEqual((read(dot, "a"), read(dot, "b")), (["a/**", "shared/**"], ["b/**"]))
        self.assertIn("# Registry: b\n", (dot / "registry" / "b.md").read_text(encoding="utf-8"))
        # A page in another form cannot be read, so its owners are unknown and nothing changes.
        dot = self.paged("odd-dot", a="---\npaths: [x/**]\n---\n", b="---\npaths: []\n---\n")
        self.assertEqual(markitect_arm.add_new_paths(self.paged("odd-model", b=block("y/**")), dot, []), (0, 0))
        self.assertEqual(read(dot, "b"), [])

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
        record, out = self.run_arm(self.task("bare", self.model("m2"), self.tmp / "reference.bundle", base), "spec")
        self.assertEqual(record["exitCode"], 0, (out / "variant" / "run.json").read_text())
        self.assertEqual(self.applies()[0]["model"].count(".markitect/registry/lending.md"), 1)
        self.assertNotIn(".markitect/registry.md", self.applies()[0]["model"])

    def test_only_sessions_that_finished_name_the_model_and_every_model_that_answered_counts(self):
        import markitect_arm
        sessions = [{"agent": "claude:claude-sonnet-5-5", "reason": "failed"}, {"agent": "claude:claude-sonnet-5-5", "reason": "deadline"},
                    {"agent": "claude:claude-sonnet-5-5-20261001", "models": ["claude-sonnet-5-5-20261001"], "reason": "finished"}]
        reported = lambda result: markitect_arm.reported_model(result, "claude", "claude-sonnet-5-5")
        self.assertEqual(reported({"sessions": sessions}), ("claude", "claude-sonnet-5-5-20261001", []))
        # A session that lists no models stands for the model of its agent.
        self.assertEqual(reported({"sessions": [{"agent": "claude:claude-sonnet-5-5-20261001", "reason": "finished"}]}),
                         ("claude", "claude-sonnet-5-5-20261001", []))
        # Subagents' models count: every model that answered is listed, and the suite holds each against the evaluation's.
        sessions.append({"agent": "claude:claude-sonnet-5-5-20261001", "models": ["claude-haiku-5", "claude-sonnet-5-5-20261001"], "reason": "finished"})
        self.assertEqual(reported({"sessions": sessions}), ("claude", "claude-haiku-5", ["claude-haiku-5", "claude-sonnet-5-5-20261001"]))
        self.assertEqual(markitect_arm.reported_model({"sessions": []}, "claude", "x"), ("claude", "x", []))
        self.assertEqual(markitect_arm.reported_model({}, "claude", "x"), ("claude", "x", []))

    def test_every_model_that_answered_is_reported_and_held_against_the_evaluations(self):
        record, _ = self.run_arm(self.task("models", self.model("model-mm")), "models")
        rep = record["reported"]
        self.assertEqual((rep["model"], rep["modelsUsed"]), ("claude-haiku-5", ["claude-haiku-5", "claude-sonnet-5-5"]))
        self.assertIs(record["modelMatches"], False)
        record, _ = self.run_arm(self.task("one-model", self.model("model-om")))
        self.assertNotIn("modelsUsed", record["reported"])
        self.assertIs(record["modelMatches"], True)

    def test_the_owners_token_reaches_the_host_and_nothing_the_arm_writes(self):
        # Every way a Command can end: landed, a stop, a host that cannot run, an owner's input that does not compile, a verify.
        outs = []
        for mode in ("land", "incomplete", "none"):
            record, out = self.run_arm(self.task("token-" + mode, self.model("model-t-" + mode)), mode)
            outs.append(out)
        os.environ["FAKE_CHECK"] = "fail"
        outs.append(self.run_arm(self.task("token-broken", self.model("model-t-b")))[1])
        del os.environ["FAKE_CHECK"]
        record, out = self.verify(self.task("token-verify", self.model("model-t-v")), "verified")
        outs.append(out)
        calls = self.applies()
        self.assertEqual(len(calls), 4)
        self.assertEqual([c["env"]["MARKITECT_CLAUDE_TOKEN"] for c in calls], [TOKEN] * len(calls))
        files = [f for out in outs for f in out.rglob("*") if f.is_file()]
        self.assertTrue(any(f.name == "run.json" for f in files) and any(f.name == "apply.stderr.log" for f in files))
        for f in files:
            self.assertNotIn(TOKEN.encode(), f.read_bytes(), f)
        # Never on a command line: not the host's, which the stand-in records, nor the arm's own.
        self.assertNotIn(TOKEN, json.dumps([c["args"] for c in calls]))
        self.assertNotIn(TOKEN, json.dumps(record["variant"]["argv"]))

    def test_the_arm_refuses_without_the_owners_token_or_a_model(self):
        for n, token in enumerate((None, "", "  ")):
            with self.subTest(token=token):
                os.environ.pop("MARKITECT_CLAUDE_TOKEN", None)
                if token is not None:
                    os.environ["MARKITECT_CLAUDE_TOKEN"] = token
                record, out = self.run_arm(self.task(f"notoken{n}", self.model(f"model-l{n}")))
                self.assertEqual((record["delivered"], record["exitCode"], record["reported"]["status"]), (False, 2, "could-not-run"))
                self.assertIn("MARKITECT_CLAUDE_TOKEN", record["reported"]["cause"])
                self.assertEqual(self.applies(), [])
                self.assertFalse((out / "variant" / "result.bundle").exists())
        os.environ["MARKITECT_CLAUDE_TOKEN"] = TOKEN
        record, _ = self.run_arm(self.task("nomodel", None))
        self.assertIn("no model/", record["reported"]["cause"])
        self.assertEqual(self.applies(), [])

    def test_the_suites_login_is_not_needed_and_codex_has_no_route(self):
        os.environ.pop("EVALUATION_LOGIN")
        record, _ = self.run_arm(self.task("nologin", self.model("model-nl")))
        self.assertEqual((record["delivered"], record["exitCode"], record["reported"]["status"]), (True, 0, "landed"))
        os.environ["FAKE_APPLY"] = "land"
        record = suite.run(self.task("codex", self.model("model-cx")), self.variant, self.tmp / "run-codex", "codex:gpt-5", None)
        self.assertEqual((record["delivered"], record["exitCode"]), (False, 2))
        self.assertIn("no token route for codex", record["reported"]["cause"])

    def test_required_checks_that_fail_after_the_repair_rounds_are_the_agents_work(self):
        record, out = self.run_arm(self.task("refused", self.model("model-rf")), "refused")
        self.assertEqual((record["delivered"], record["exitCode"], record["reported"]["status"]), (True, 1, "refused"))
        self.assertEqual(record["result"]["changedPaths"], [])
        self.assertEqual(classify.classify(out)["class"], "invalid")  # host mode: for tests only, but not a void
        # A start the host cannot compile ends before any agent: no work to score, so nothing is delivered.
        record, _ = self.run_arm(self.task("gate2", self.model("model-g2")), "gate")
        self.assertEqual((record["delivered"], record["exitCode"], record["reported"]["status"]), (False, 2, "check-failed"))
        # check-failed after a session finished would be the agent's work: the start is delivered.
        record, _ = self.run_arm(self.task("gate3", self.model("model-g3")), "gate-late")
        self.assertEqual((record["delivered"], record["exitCode"], record["reported"]["status"]), (True, 1, "check-failed"))
        self.assertNotIn("noAgentTurn", record["reported"])

    def test_a_removed_packages_page_stays_and_a_new_one_is_added(self):
        import markitect_arm
        dot, step = self.tmp / "dot", self.model("step-two")
        (dot / "registry").mkdir(parents=True)
        (dot / "registry" / "lending.md").write_text("---\npaths:\n  - realization/lending/**\n---\n| `realization/lending/x.go` | | written |\n")
        (dot / "registry" / "gone.md").write_text("---\npaths:\n  - realization/gone/**\n---\n| `realization/gone/y.go` | | written |\n")
        (step / "registry" / "fresh.md").write_text("---\npaths:\n  - realization/fresh/**\n---\n")
        markitect_arm.copy_model(step, dot, keep_registry=True)
        self.assertIn("| `realization/lending/x.go` | | written |", (dot / "registry" / "lending.md").read_text())
        self.assertTrue((dot / "registry" / "gone.md").is_file())
        self.assertTrue((dot / "registry" / "fresh.md").is_file())
        self.assertTrue((dot / "model" / "lending" / "loan-policy.md").is_file())


if __name__ == "__main__":
    unittest.main()
