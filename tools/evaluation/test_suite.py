"""Offline tests of the suite runner and the baselines on Claude Code and Codex: real Git, fake variants, no provider."""
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
import claude_code  # noqa: E402
import classify  # noqa: E402
import codex_cli  # noqa: E402
import suite  # noqa: E402
import test_sandbox  # noqa: E402

MODEL = "claude:claude-sonnet-5-5"
CODEX = "codex:gpt-6-luna"

# A fake variant: appends one line per step to work.txt, records its state, and writes result.bundle.
# Its behavior is chosen by the first line of task.md: "ok", "none" (no result), "sleep",
# "orphan" (a commit unrelated to the start) or "wrong-model".
FAKE = textwrap.dedent("""
    import json, os, pathlib, subprocess, sys, time
    task, out = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
    state = pathlib.Path(sys.argv[3]) if len(sys.argv) > 3 else None
    mode = (task / "task.md").read_text().splitlines()[0]
    spec = json.loads((task / "task.json").read_text())
    if mode == "sleep":
        time.sleep(60)
    if mode == "tamper":
        # In a chain the original task sits next to the run directory.
        (out.parent.parent / "task" / "task.md").write_text("changed\\n")
    if state:
        with open(state / "seen.txt", "a") as f:
            f.write(mode + "\\n")
    if mode == "pages":
        (out / "task-files.txt").write_text("\\n".join(sorted(p.relative_to(task).as_posix() for p in task.rglob("*") if p.is_file())))
    if mode == "none":
        sys.exit(1)
    repo = out / "repo"
    env = dict(os.environ, GIT_AUTHOR_NAME="v", GIT_AUTHOR_EMAIL="v@x", GIT_COMMITTER_NAME="v", GIT_COMMITTER_EMAIL="v@x")
    g = lambda *a: subprocess.run(["git", "-C", str(repo), *a], check=True, capture_output=True, env=env)
    if mode == "orphan":
        repo.mkdir()
        g("init", "-q")
    else:
        subprocess.run(["git", "clone", "-q", str(task / "start.bundle"), str(repo)], check=True)
        g("checkout", "-q", "-B", "work", spec["start"]["commit"])
    with open(repo / "work.txt", "a") as f:
        f.write(mode + "\\n")
    g("add", "-A")
    g("-c", "commit.gpgsign=false", "commit", "-q", "-m", "work")
    g("branch", "-f", "result", "HEAD")
    g("bundle", "create", str((out / "result.bundle").resolve()), "refs/heads/result")
    model = "other" if mode == "wrong-model" else os.environ["EVALUATION_MODEL"].split(":", 1)[1]
    (out / "run.json").write_text(json.dumps({"provider": "claude", "model": model}))
""")


def git(repo, *args):
    return subprocess.run(["git", "-C", str(repo), *args], check=True, capture_output=True, text=True).stdout.strip()


class SuiteTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = pathlib.Path(self._tmp.name)
        self.fake = self.tmp / "fake.py"
        self.fake.write_text(FAKE)
        self.variant = [sys.executable, "-B", str(self.fake)]

    def tearDown(self):
        self._tmp.cleanup()

    def step(self, name, mode, model=False):
        d = self.tmp / "steps" / name
        d.mkdir(parents=True)
        (d / "task.md").write_text(mode + "\n")
        if model:
            (d / "model").mkdir()
            (d / "model" / "m.yaml").write_text("kind: Thing\n")
        return d

    def task(self, mode, deadline=120):
        s = self.step(mode, mode, model=True)
        out = self.tmp / f"task-{mode}"
        suite.make_task(s / "task.md", s / "model", None, deadline, out)
        return out

    def test_task_is_complete_and_checked(self):
        task = self.task("ok")
        record = suite.read_task(task)
        self.assertEqual(record["start"]["commit"], suite.bundle_head(task / "start.bundle"))
        (task / "model" / "m.yaml").write_text("kind: Changed\n")
        with self.assertRaisesRegex(suite.SuiteError, "does not match"):
            suite.read_task(task)

    def test_a_base_model_belongs_to_a_start_and_is_part_of_the_task(self):
        s = self.step("based", "based", model=True)
        with self.assertRaisesRegex(suite.SuiteError, "base model"):
            suite.make_task(s / "task.md", s / "model", None, 120, self.tmp / "no-start", s / "model")
        start = self.tmp / "start.bundle"
        suite.empty_start(start)
        task = self.tmp / "task-based"
        suite.make_task(s / "task.md", s / "model", start, 120, task, s / "model")
        self.assertTrue((task / "base-model" / "m.yaml").is_file())
        (task / "base-model" / "m.yaml").write_text("kind: Changed\n")
        with self.assertRaisesRegex(suite.SuiteError, "does not match"):
            suite.read_task(task)

    def test_every_task_ends_with_the_same_stop_rule(self):
        task = self.task("ok")
        text = (task / "task.md").read_text(encoding="utf-8")
        self.assertEqual(text.splitlines()[0], "ok")
        self.assertTrue(text.endswith(suite.STOP_RULE))
        self.assertEqual(text.count("## Stop rule"), 1)

    def test_empty_start_is_reproducible(self):
        a, b = self.tmp / "a.bundle", self.tmp / "b.bundle"
        suite.empty_start(a)
        suite.empty_start(b)
        self.assertEqual(suite.bundle_head(a), suite.bundle_head(b))

    def test_run_collects_a_delivered_result(self):
        task = self.task("ok")
        record = suite.run(task, self.variant, self.tmp / "run", MODEL)
        self.assertTrue(record["delivered"], record["resultError"])
        self.assertFalse(record["timedOut"])
        self.assertTrue(record["modelMatches"])
        self.assertIn("A\twork.txt", record["result"]["changedPaths"])
        self.assertEqual(json.loads((self.tmp / "run" / "suite.json").read_text())["result"], record["result"])

    def test_variant_sees_a_copy_and_the_model(self):
        task = self.task("ok")
        before = suite.intent_digest(task)
        suite.run(task, self.variant, self.tmp / "run", MODEL)
        self.assertEqual(suite.intent_digest(task), before)

    def test_missing_result_is_not_delivered(self):
        record = suite.run(self.task("none"), self.variant, self.tmp / "run", MODEL)
        self.assertFalse(record["delivered"])
        self.assertEqual(record["resultError"], "no result.bundle")
        self.assertEqual(record["exitCode"], 1)

    def test_result_must_descend_from_the_start(self):
        record = suite.run(self.task("orphan"), self.variant, self.tmp / "run", MODEL)
        self.assertFalse(record["delivered"])
        self.assertIn("does not descend", record["resultError"])

    def test_wrong_model_is_flagged(self):
        record = suite.run(self.task("wrong-model"), self.variant, self.tmp / "run", MODEL)
        self.assertTrue(record["delivered"])
        self.assertIs(record["modelMatches"], False)

    def test_deadline_stops_the_variant(self):
        task = self.task("sleep", deadline=2)
        started = time.monotonic()
        record = suite.run(task, self.variant, self.tmp / "run", MODEL)
        self.assertTrue(record["timedOut"])
        self.assertFalse(record["delivered"])
        self.assertLess(time.monotonic() - started, 30)

    def test_refuses_existing_out_and_bad_model(self):
        task = self.task("ok")
        (self.tmp / "exists").mkdir()
        with self.assertRaisesRegex(suite.SuiteError, "exists"):
            suite.run(task, self.variant, self.tmp / "exists", MODEL)
        with self.assertRaisesRegex(suite.SuiteError, "PROVIDER:MODEL_ID"):
            suite.run(task, self.variant, self.tmp / "run", "sonnet")

    def test_chain_builds_on_own_results_and_keeps_state(self):
        steps = [self.step("1", "ok"), self.step("2", "none"), self.step("3", "ok")]
        out = self.tmp / "chain"
        summary = suite.chain(steps, None, 120, self.variant, out, MODEL)
        self.assertEqual([s["delivered"] for s in summary["steps"]], [True, False, True])
        # Step 3 starts from step 2's start (step 1's result), since step 2 delivered nothing.
        self.assertEqual(suite.read_task(out / "step-03" / "task")["start"]["commit"], summary["steps"][0]["result"])
        repo = self.tmp / "final"
        subprocess.run(["git", "clone", "-q", "-b", "result", str(out / "step-03" / "run" / "variant" / "result.bundle"), str(repo)], check=True)
        self.assertEqual((repo / "work.txt").read_text().split(), ["ok", "ok"])
        self.assertEqual((out / "state" / "seen.txt").read_text().split(), ["ok", "none", "ok"])

    def test_a_chain_continues_an_earlier_one_with_its_state(self):
        first = suite.chain([self.step("1", "ok")], None, 120, self.variant, self.tmp / "first", MODEL)
        bundle = self.tmp / "first" / "step-01" / "run" / "variant" / "result.bundle"
        with self.assertRaisesRegex(suite.SuiteError, "does not exist"):
            suite.chain([self.step("2", "ok")], bundle, 120, self.variant, self.tmp / "bad", MODEL, state=self.tmp / "nowhere")
        summary = suite.chain([self.step("3", "ok")], bundle, 120, self.variant, self.tmp / "second", MODEL, state=self.tmp / "first" / "state")
        self.assertEqual(suite.read_task(self.tmp / "second" / "step-01" / "task")["start"]["commit"], first["steps"][0]["result"])
        self.assertEqual((self.tmp / "first" / "state" / "seen.txt").read_text().split(), ["ok", "ok"])
        self.assertFalse((self.tmp / "second" / "state").exists())
        self.assertTrue(summary["steps"][0]["delivered"])

    def test_changed_task_invalidates_the_run_and_stops_the_chain(self):
        steps = [self.step("1", "tamper"), self.step("2", "ok")]
        summary = suite.chain(steps, None, 120, self.variant, self.tmp / "chain", MODEL)
        self.assertEqual([s["delivered"] for s in summary["steps"]], [False])
        record = json.loads((self.tmp / "chain" / "step-01" / "run" / "suite.json").read_text())
        self.assertIn("the task changed", record["resultError"])
        self.assertEqual(summary["stopped"]["step"], 2)
        self.assertTrue((self.tmp / "chain" / "chain.json").is_file())

    def test_chain_from_a_head_only_start_bundle(self):
        repo = self.tmp / "origin"
        repo.mkdir()
        git(repo, "init", "-q")
        (repo / "a.txt").write_text("a\n")
        git(repo, "add", "-A")
        git(repo, "-c", "user.name=o", "-c", "user.email=o@x", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "a")
        bundle = self.tmp / "head.bundle"
        git(repo, "bundle", "create", str(bundle), "HEAD")
        summary = suite.chain([self.step("1", "ok"), self.step("2", "ok")], bundle, 120, self.variant, self.tmp / "chain", MODEL)
        self.assertEqual([s["delivered"] for s in summary["steps"]], [True, True])
        self.assertIsNone(summary["stopped"])

    def test_without_pages_the_variant_gets_the_task_without_its_registry_pages(self):
        s, base = self.step("pages", "pages", model=True), self.tmp / "base"
        for root in (s / "model", base):
            (root / "registry").mkdir(parents=True)
            (root / "registry" / "p.md").write_text("# page\n")
            (root / "registry.md").write_text("# root page\n")
        (base / "m.yaml").write_text("kind: Base\n")
        start, task = self.tmp / "start.bundle", self.tmp / "task-pages"
        suite.empty_start(start)
        suite.make_task(s / "task.md", s / "model", start, 120, task, base)
        before = suite.intent_digest(task)
        code = suite.main(["run", "--task", str(task), "--out", str(self.tmp / "k"), "--model", MODEL, "--without-pages", "--", *self.variant])
        self.assertEqual(code, 0)
        record = json.loads((self.tmp / "k" / "suite.json").read_text())
        self.assertEqual(sorted(record["pagesWithheld"]), ["base-model/registry", "base-model/registry.md", "model/registry", "model/registry.md"])
        seen = (self.tmp / "k" / "variant" / "task-files.txt").read_text().split()
        self.assertEqual(([p for p in seen if "registry" in p], "model/m.yaml" in seen, "base-model/m.yaml" in seen), ([], True, True))
        # The task keeps its pages, and a variant run without the option gets them.
        self.assertEqual(suite.intent_digest(task), before)
        record = suite.run(task, self.variant, self.tmp / "m", MODEL)
        self.assertNotIn("pagesWithheld", record)
        self.assertIn("base-model/registry/p.md", (self.tmp / "m" / "variant" / "task-files.txt").read_text().split())

    def test_cli_runs_a_task(self):
        task = self.task("ok")
        code = suite.main(["run", "--task", str(task), "--out", str(self.tmp / "cli"), "--model", MODEL, "--", *self.variant])
        self.assertEqual(code, 0)
        self.assertTrue(json.loads((self.tmp / "cli" / "suite.json").read_text())["delivered"])


# A fake `claude`: checks its isolation, edits the repository and writes a stream-json result.
FAKE_CLAUDE = textwrap.dedent("""
    import json, os, pathlib, subprocess, sys, time
    cwd = pathlib.Path.cwd()
    mode = (cwd / "TASK.md").read_text().splitlines()[0]
    assert not any(k.startswith(("ANTHROPIC", "EVALUATION_")) for k in os.environ), "leaked variable"
    autocrlf = subprocess.run(["git", "config", "core.autocrlf"], capture_output=True, text=True).stdout.strip()
    assert autocrlf == "false", "line endings not fixed: " + autocrlf
    config = pathlib.Path(os.environ["CLAUDE_CONFIG_DIR"])
    login = json.loads((config / ".credentials.json").read_text())["claudeAiOauth"]
    assert "refreshToken" not in login, "refresh token reached the session"
    argv = sys.argv[1:]
    model = argv[argv.index("--model") + 1]
    assert sys.stdin.read().startswith("Read the documentation in TASK.md"), "prompt not on stdin"
    (cwd / "main.go").write_text("package main\\n")
    if mode == "leak":
        (cwd / "token.txt").write_text(login["accessToken"])
    if mode == "ignore":
        # A binary's name that also matches the source directories, as in the first sandbox smoke.
        (cwd / ".gitignore").write_text("library\\n")
        for p in ("cmd/library/main.go", "internal/library/library.go", "internal/library/README.md"):
            (cwd / p).parent.mkdir(parents=True, exist_ok=True)
            (cwd / p).write_text("package library\\n")
        (cwd / "library").write_bytes(b"\\x7fELF")
    if mode == "sleep":
        time.sleep(120)
    (config / "projects" / "p").mkdir(parents=True)
    (config / "projects" / "p" / "s.jsonl").write_text("{}\\n")
    print(json.dumps({"type": "system", "subtype": "init", "model": model, "session_id": "s1"}))
    print(json.dumps({"type": "result", "subtype": "success", "num_turns": 3, "total_cost_usd": 0.1,
                      "usage": {"input_tokens": 10, "output_tokens": 5, "cache_read_input_tokens": 7}}))
""")


class ClaudeCodeTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = pathlib.Path(self._tmp.name)
        fake = self.tmp / "fake_claude.py"
        fake.write_text(FAKE_CLAUDE)
        if os.name == "nt":
            self.claude = self.tmp / "claude.cmd"
            self.claude.write_text(f'@"{sys.executable}" -B "{fake}" %*\n')
        else:
            self.claude = self.tmp / "claude"
            self.claude.write_text(f'#!/bin/sh\nexec "{sys.executable}" -B "{fake}" "$@"\n')
            self.claude.chmod(0o755)
        self.credentials = self.tmp / "credentials.json"
        expires = int((time.time() + 7200) * 1000)
        self.credentials.write_text(json.dumps({"claudeAiOauth": {"accessToken": "sk-access-SECRET", "refreshToken": "sk-refresh-SECRET",
                                                                  "expiresAt": expires}}))
        self.task = self.make("ok", 600)
        self.variant = [sys.executable, "-B", str(HERE / "claude_code.py"), "--claude", str(self.claude), "--credentials", str(self.credentials)]

    def tearDown(self):
        self._tmp.cleanup()

    def make(self, mode, deadline):
        (self.tmp / f"{mode}.md").write_text(mode + "\n")
        task = self.tmp / f"task-{mode}"
        suite.make_task(self.tmp / f"{mode}.md", None, None, deadline, task)
        return task

    def test_a_committed_login_is_never_delivered(self):
        record = suite.run(self.make("leak", 600), self.variant, self.tmp / "run", MODEL)
        self.assertFalse(record["delivered"])
        self.assertEqual(record["reported"]["status"], "quarantined")
        out = self.tmp / "run" / "variant"
        self.assertFalse((out / "result.bundle").exists())
        for p in out.rglob("*"):
            if p.is_file():
                self.assertNotIn(b"SECRET", p.read_bytes(), p)

    def test_a_timed_out_session_still_delivers_its_work(self):
        # 63 s deadline less the 60 s margin: the session is stopped after 3 s, well before the suite's deadline.
        record = suite.run(self.make("sleep", 63), self.variant, self.tmp / "run", MODEL)
        self.assertFalse(record["timedOut"])
        self.assertEqual(record["reported"]["status"], "timed_out")
        self.assertTrue(record["delivered"], record["resultError"])
        self.assertEqual(record["result"]["changedPaths"], ["A\tmain.go"])

    def test_sources_an_over_broad_gitignore_matches_are_kept(self):
        record = suite.run(self.make("ignore", 600), self.variant, self.tmp / "run", MODEL)
        self.assertTrue(record["delivered"], record["resultError"])
        self.assertEqual(sorted(record["result"]["changedPaths"]),
                         ["A\t.gitignore", "A\tcmd/library/main.go", "A\tinternal/library/README.md",
                          "A\tinternal/library/library.go", "A\tmain.go"])

    def test_login_copy_has_no_refresh_token_and_must_outlive_the_session(self):
        copy, secrets = claude_code.session_login(self.credentials.read_bytes(), 600)
        self.assertNotIn(b"refresh", copy)
        self.assertEqual(len(secrets), 2)
        with self.assertRaisesRegex(claude_code.VariantError, "expires"):
            claude_code.session_login(self.credentials.read_bytes(), 10 * 3600)

    def test_baseline_through_the_suite(self):
        record = suite.run(self.task, self.variant, self.tmp / "run", MODEL)
        self.assertTrue(record["delivered"], (record["resultError"], record["reported"]))
        self.assertTrue(record["modelMatches"], (record["reported"], (self.tmp / "run" / "variant" / "session.stderr.log").read_text()))
        self.assertEqual(record["reported"]["status"], "completed")
        self.assertEqual(record["reported"]["usage"]["cacheReadTokens"], 7)
        self.assertIsNone(record["reported"]["usage"]["reasoningTokens"])
        self.assertEqual(record["result"]["changedPaths"], ["A\tmain.go"])
        out = self.tmp / "run" / "variant"
        self.assertTrue((out / "transcripts" / "p" / "s.jsonl").is_file())
        for p in out.rglob("*"):
            if p.is_file():
                self.assertNotIn(b"SECRET", p.read_bytes(), p)

    def test_baseline_refuses_another_provider(self):
        record = suite.run(self.task, self.variant, self.tmp / "run", "codex:gpt-6-luna")
        self.assertFalse(record["delivered"])
        self.assertIn("EVALUATION_MODEL", record["reported"]["cause"])


# A fake `codex`: checks its isolation and switches, edits the repository and writes exec events. TASK.md's first line
# chooses: "ok", "leak" (writes the access token), "sleep", "limit" (a failed turn on a usage limit) or "silent" (no
# thread).
FAKE_CODEX = textwrap.dedent("""
    import json, os, pathlib, sys, time
    cwd = pathlib.Path.cwd()
    mode = (cwd / "TASK.md").read_text().splitlines()[0]
    leaked = [k for k in os.environ if k.startswith(("ANTHROPIC", "EVALUATION_", "OPENAI_")) or (k.startswith("CODEX_") and k != "CODEX_HOME")]
    assert not leaked, "leaked variable: " + ", ".join(leaked)
    home = pathlib.Path(os.environ["CODEX_HOME"])
    tokens = json.loads((home / "auth.json").read_text())["tokens"]
    assert tokens["refresh_token"] == "", "refresh token reached the session"
    argv = sys.argv[1:]
    assert argv[:2] == ["exec", "--json"] and argv[-1] == "-", argv
    for switch in ("--ignore-user-config", "--ignore-rules", "--dangerously-bypass-approvals-and-sandbox", 'web_search="disabled"',
                   'model_reasoning_effort="high"'):
        assert switch in argv, switch
    model = argv[argv.index("-m") + 1]
    assert sys.stdin.read().startswith("Read the documentation in TASK.md"), "prompt not on stdin"
    if mode == "silent":
        sys.exit(1)
    print(json.dumps({"type": "thread.started", "thread_id": "th-1"}))
    print(json.dumps({"type": "turn.started"}))
    (cwd / "main.go").write_text("package main\\n")
    if mode == "leak":
        (cwd / "token.txt").write_text(tokens["access_token"])
    if mode == "sleep":
        time.sleep(120)
    (home / "sessions" / "2026").mkdir(parents=True)
    (home / "sessions" / "2026" / "rollout-th-1.jsonl").write_text(json.dumps({"type": "session_meta", "model": model}) + "\\n")
    if mode == "limit":
        print(json.dumps({"type": "turn.failed", "error": {"message": "You've hit your usage limit."}}))
        sys.exit(1)
    print(json.dumps({"type": "turn.completed", "usage": {"input_tokens": 30, "cached_input_tokens": 20, "output_tokens": 5,
                                                          "reasoning_output_tokens": 2}}))
""")


class CodexCliTest(unittest.TestCase):
    """The baseline on Codex through the suite, with a fake `codex`."""

    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = pathlib.Path(self._tmp.name)
        fake = self.tmp / "fake_codex.py"
        fake.write_text(FAKE_CODEX)
        if os.name == "nt":
            self.codex = self.tmp / "codex.cmd"
            self.codex.write_text(f'@"{sys.executable}" -B "{fake}" %*\n')
        else:
            self.codex = self.tmp / "codex"
            self.codex.write_text(f'#!/bin/sh\nexec "{sys.executable}" -B "{fake}" "$@"\n')
            self.codex.chmod(0o755)
        self.credentials = self.tmp / "auth.json"
        self.credentials.write_text(test_sandbox.codex_auth(7200))
        self.variant = [sys.executable, "-B", str(HERE / "codex_cli.py"), "--codex", str(self.codex), "--credentials", str(self.credentials),
                        "--effort", "high"]
        self._env = dict(os.environ)
        os.environ.update(OPENAI_API_KEY="sk-proj-host", CODEX_HOME=str(self.tmp / "host-codex"))

    def tearDown(self):
        os.environ.clear()
        os.environ.update(self._env)
        self._tmp.cleanup()

    def run_mode(self, mode, deadline=600, model=CODEX):
        (self.tmp / f"{mode}.md").write_text(mode + "\n")
        suite.make_task(self.tmp / f"{mode}.md", None, None, deadline, self.tmp / f"task-{mode}")
        return suite.run(self.tmp / f"task-{mode}", self.variant, self.tmp / f"run-{mode}", model)

    def no_secret(self, out):
        for p in out.rglob("*"):
            if p.is_file():
                self.assertNotIn(b"SECRET", p.read_bytes(), p)

    def test_baseline_through_the_suite(self):
        record = self.run_mode("ok")
        out = self.tmp / "run-ok" / "variant"
        self.assertTrue(record["delivered"], (record["resultError"], record["reported"], (out / "session.stderr.log").read_text()))
        self.assertTrue(record["modelMatches"])
        reported = record["reported"]
        self.assertEqual((reported["provider"], reported["status"], reported["sessionResult"], reported["sessionId"]),
                         ("codex", "completed", "completed", "th-1"))
        self.assertEqual(reported["usage"], {"inputTokens": 10, "cacheReadTokens": 20, "cacheCreationTokens": None, "outputTokens": 5,
                                             "reasoningTokens": 2, "turns": None, "costUsd": None})
        self.assertEqual(record["result"]["changedPaths"], ["A\tmain.go"])
        self.assertTrue((out / "transcripts" / "2026" / "rollout-th-1.jsonl").is_file())
        self.no_secret(out)

    def test_a_failed_turn_is_reported_for_the_classifier(self):
        record = self.run_mode("limit")
        reported = record["reported"]
        self.assertEqual((reported["status"], reported["sessionResult"], reported["sessionError"]),
                         ("failed", "failed", "You've hit your usage limit."))
        # The session's work is still delivered; the classifier decides whether the run is void.
        self.assertTrue(record["delivered"])
        errors = classify.session_errors(self.tmp / "run-limit" / "variant")
        self.assertEqual([e["error"] for e in errors], ["rate-limit"])

    def test_a_committed_login_is_never_delivered(self):
        record = self.run_mode("leak")
        self.assertFalse(record["delivered"])
        self.assertEqual(record["reported"]["status"], "quarantined")
        self.no_secret(self.tmp / "run-leak" / "variant")

    def test_a_timed_out_session_still_delivers_its_work(self):
        record = self.run_mode("sleep", deadline=63)
        self.assertEqual(record["reported"]["status"], "timed_out")
        self.assertTrue(record["delivered"], record["resultError"])

    def test_a_session_that_never_started_a_thread_reports_no_model(self):
        record = self.run_mode("silent")
        self.assertEqual((record["reported"]["status"], record["reported"]["model"]), ("failed", None))
        self.assertIsNone(record["modelMatches"])

    def test_login_copy_has_an_empty_refresh_token_and_must_outlive_the_session(self):
        copy, secrets = codex_cli.session_login(self.credentials.read_bytes(), 600)
        self.assertEqual(json.loads(copy)["tokens"]["refresh_token"], "")
        self.assertEqual(len(secrets), 3)
        with self.assertRaisesRegex(codex_cli.baseline.VariantError, "expires"):
            codex_cli.session_login(self.credentials.read_bytes(), 10 * 3600)
        with self.assertRaisesRegex(codex_cli.baseline.VariantError, "API key"):
            codex_cli.session_login(test_sandbox.codex_auth(7200, api_key="sk-proj-x").encode(), 600)

    def test_baseline_refuses_another_provider(self):
        record = self.run_mode("ok", model=MODEL)
        self.assertFalse(record["delivered"])
        self.assertIn("EVALUATION_MODEL", record["reported"]["cause"])

    def test_the_suite_copies_the_codex_login_by_default(self):
        a = suite.argparse.Namespace(native=None, sandbox="sha256:" + "a" * 64, variant_code=str(HERE), login=None, lock=None,
                                     hidden_absent=None, model=CODEX, series=str(suite.sandboxing.DEFAULT_SERIES), engine="docker")
        self.assertEqual(suite.sandbox_from(a).login, pathlib.Path.home() / ".codex" / "auth.json")


if __name__ == "__main__":
    unittest.main()
