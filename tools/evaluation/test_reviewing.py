"""Offline tests of the blinded review and the series report: real Git, a fake `claude`, no provider."""
import contextlib
import io
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile
import textwrap
import time
import unittest

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import judges  # noqa: E402
import report  # noqa: E402
import reviewer  # noqa: E402
import reviewing  # noqa: E402
import suite  # noqa: E402

MODEL = "claude:claude-sonnet-5-5"
REQUIREMENT = "The loan period is 14 calendar days."

# A variant whose result carries Markitect's files, a compiled executable and a README that contradicts the task.
VARIANT = textwrap.dedent('''
    import json, os, pathlib, subprocess, sys
    task, out = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
    repo = out / "repo"
    env = dict(os.environ, GIT_AUTHOR_NAME="v", GIT_AUTHOR_EMAIL="v@x", GIT_COMMITTER_NAME="v", GIT_COMMITTER_EMAIL="v@x")
    g = lambda *a: subprocess.run(["git", "-C", str(repo), *a], check=True, capture_output=True, env=env)
    subprocess.run(["git", "clone", "-q", str(task / "start.bundle"), str(repo)], check=True, capture_output=True)
    g("checkout", "-q", "-B", "work", json.loads((task / "task.json").read_text())["start"]["commit"])
    files = {"realization/go.mod": "module x\\n", "realization/cmd/library/main.go": "package main\\n\\nfunc main() {}\\n",
             "realization/README.md": "# Library\\n\\nThe loan period is 21 calendar days.\\n",
             ".markitect/registry/realization.yaml": "owner: lending\\n", ".markitect/model.yaml": "kind: Model\\n"}
    for rel, text in files.items():
        (repo / rel).parent.mkdir(parents=True, exist_ok=True)
        (repo / rel).write_text(text)
    (repo / "realization" / "library").write_bytes(b"\\x7fELF binary")
    g("add", "-A"); g("-c", "commit.gpgsign=false", "commit", "-q", "-m", "work"); g("branch", "-f", "result", "HEAD")
    g("bundle", "create", str((out / "result.bundle").resolve()), "refs/heads/result")
    (out / "run.json").write_text(json.dumps({"provider": "claude", "model": os.environ["EVALUATION_MODEL"].split(":", 1)[1]}))
''')

# A fake `claude` reviewer: checks it may only read, then answers the review or the confirmation as FAKE_REVIEW says:
# "critical" (one cited R2 contradiction), "clean" (no findings) or "garbage" (no JSON).
FAKE_CLAUDE = textwrap.dedent('''
    import json, os, pathlib, re, sys
    argv = sys.argv[1:]
    allowed = argv[argv.index("--allowedTools") + 1:argv.index("--disallowedTools")]
    assert allowed == ["Read", "Glob", "Grep"], allowed
    assert "Bash" in argv[argv.index("--disallowedTools") + 1:], "Bash is not denied"
    prompt = sys.stdin.read()
    assert "TOKEN_DIR" not in prompt and "variantGuess" in prompt, prompt
    mode = os.environ.get("FAKE_REVIEW", "critical")
    config = pathlib.Path(os.environ["CLAUDE_CONFIG_DIR"])
    (config / "projects" / "p").mkdir(parents=True, exist_ok=True)
    (config / "projects" / "p" / "s.jsonl").write_text(json.dumps({"type": "assistant", "message": {"id": "m1", "usage": {"input_tokens": 5, "output_tokens": 7}}}) + "\\n")
    if prompt.startswith("A reviewer reported"):
        answer = {"confirmed": True, "reason": "The README states 21 days.", "variantGuess": {"variant": "M", "why": "registry"}}
    else:
        token_dir = pathlib.Path(re.search(r"Review the program in (\\S+) against", prompt).group(1))
        assert not (token_dir / "candidate" / ".markitect").exists(), "the copy is not blinded"
        findings = [] if mode == "clean" else [{"cite": "realization/README.md:3", "quote": "The loan period is 21 calendar days.",
                                                "requirement": "The loan period is 14 calendar days."}]
        answer = {"token": token_dir.name, "R1a": True, "R1b": True, "R1c": {"run": True, "test": True, "scope": True}, "R2": findings,
                  "R3a": [], "R4": [], "R5": [], "R7": [], "R6": "", "variantGuess": {"variant": "K", "why": "one session"}}
    if mode == "wrong-shape" and not prompt.startswith("A reviewer reported"):
        answer["R1c"] = ["run"]
    text = "no answer" if mode == "garbage" else "```json\\n" + json.dumps(answer) + "\\n```"
    print(json.dumps({"type": "system", "subtype": "init", "model": argv[argv.index("--model") + 1], "session_id": "s1"}))
    print(json.dumps({"type": "result", "subtype": "success", "result": text, "num_turns": 2, "usage": {"input_tokens": 5, "output_tokens": 7}}))
''')


class ReviewTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = pathlib.Path(self._tmp.name)
        (self.tmp / "fake_claude.py").write_text(FAKE_CLAUDE)
        if os.name == "nt":
            self.claude = self.tmp / "claude.cmd"
            self.claude.write_text(f'@"{sys.executable}" -B "{self.tmp / "fake_claude.py"}" %*\n')
        else:
            self.claude = self.tmp / "claude"
            self.claude.write_text(f'#!/bin/sh\nexec "{sys.executable}" -B "{self.tmp / "fake_claude.py"}" "$@"\n')
            self.claude.chmod(0o755)
        self.credentials = self.tmp / "credentials.json"
        self.credentials.write_text(json.dumps({"claudeAiOauth": {"accessToken": "sk-access-SECRET", "refreshToken": "sk-refresh-SECRET",
                                                                  "expiresAt": int((time.time() + 7200) * 1000)}}))
        (self.tmp / "task.md").write_text("# Library\n\n" + REQUIREMENT + "\n")
        suite.make_task(self.tmp / "task.md", None, None, 600, self.tmp / "task")
        (self.tmp / "variant.py").write_text(VARIANT)
        self.run_dir = self.tmp / "run"
        suite.run(self.tmp / "task", [sys.executable, "-B", str(self.tmp / "variant.py")], self.run_dir, MODEL)
        self.judge = self.tmp / "judge"
        judges.export(HERE.parent / "evaluation-judges" / "library-m1", self.judge)
        self.reviewer = [sys.executable, "-B", str(HERE / "reviewer.py"), "--claude", str(self.claude), "--credentials", str(self.credentials)]
        self._env = dict(os.environ)

    def tearDown(self):
        os.environ.clear()
        os.environ.update(self._env)
        self._tmp.cleanup()

    def reviewed(self, mode, out="review"):
        os.environ["FAKE_REVIEW"] = mode
        return reviewing.review(self.run_dir, self.judge, self.tmp / out, MODEL, None, self.reviewer, 600)

    def test_the_copy_is_blinded_and_the_code_unchanged(self):
        result = self.reviewed("clean")
        token = self.tmp / "review" / "material" / result["token"]
        self.assertRegex(result["token"], r"^[a-z0-9]{12}$")
        self.assertFalse((token / "candidate" / ".markitect").exists())
        self.assertFalse((token / "candidate" / "realization" / "library").exists())
        diff = (token / "diff.patch").read_text()
        self.assertNotIn(".markitect", diff)
        self.assertNotIn("realization/library\n", diff)
        self.assertEqual((token / "candidate" / "realization" / "cmd" / "library" / "main.go").read_text(), "package main\n\nfunc main() {}\n")
        self.assertEqual((self.tmp / "review" / "material" / "requirements.md").read_text(), (self.tmp / "task" / "task.md").read_text())
        self.assertEqual((self.tmp / "review" / "material" / "rubric.md").read_bytes(), (self.judge / "rubric.md").read_bytes())

    def test_a_clean_review_needs_no_confirmation(self):
        result = self.reviewed("clean")
        self.assertTrue(result["valid"], result["problems"])
        self.assertEqual(result["criticals"], {"reported": 0, "valid": 0, "confirmed": 0})
        self.assertEqual([s["name"] for s in result["sessions"]], ["review"])
        self.assertEqual(result["guesses"], [{"session": "review", "variant": "K", "why": "one session"}])

    def test_a_valid_critical_finding_is_confirmed_by_a_second_reviewer(self):
        result = self.reviewed("critical")
        self.assertTrue(result["valid"], result["problems"])
        r2 = [c for c in result["check"]["criticals"] if c["item"] == "R2"]
        self.assertTrue(r2[0]["valid"], r2)
        self.assertEqual(result["criticals"]["confirmed"], 1)
        self.assertEqual([s["name"] for s in result["sessions"]], ["review", "confirm-1"])
        self.assertEqual([g["variant"] for g in result["guesses"]], ["K", "M"])
        session = json.loads((self.tmp / "review" / "sessions" / "confirm-1" / "session.json").read_text())
        self.assertTrue(session["modelMatches"])
        for p in (self.tmp / "review").rglob("*"):
            if p.is_file():
                self.assertNotIn(b"SECRET", p.read_bytes(), p)

    def test_reviewer_sessions_run_in_the_sandbox(self):
        import sandbox
        import test_sandbox
        (self.tmp / "engine.py").write_text(test_sandbox.FAKE_ENGINE)
        os.environ.update(FAKE_DIR=str(self.tmp), FAKE_IMAGE=test_sandbox.IMAGE, FAKE_REVIEW="critical")
        box = sandbox.Sandbox([sys.executable, "-I", str(self.tmp / "engine.py")], test_sandbox.IMAGE, sandbox.load_series(sandbox.DEFAULT_SERIES),
                              HERE, self.credentials, "claude")
        reviewer_argv = [sys.executable, "-B", "/variant/reviewer.py", "--claude", str(self.claude)]
        result = reviewing.review(self.run_dir, self.judge, self.tmp / "boxed", MODEL, box, reviewer_argv, 600)
        self.assertTrue(result["valid"], result["problems"])
        self.assertEqual(result["criticals"]["confirmed"], 1)
        for name in ("review", "confirm-1"):
            session = json.loads((self.tmp / "boxed" / "sessions" / name / "session.json").read_text())
            self.assertEqual(session["conditions"], "sandbox")
            self.assertFalse(session["sandbox"]["residue"])
            self.assertTrue((self.tmp / "boxed" / "sessions" / name / "egress.jsonl").is_file())
        for p in (self.tmp / "boxed").rglob("*"):
            if p.is_file():
                self.assertNotIn(b"SECRET", p.read_bytes(), p)

    def test_an_answer_of_the_wrong_shape_is_invalid_and_still_recorded(self):
        result = self.reviewed("wrong-shape")
        self.assertFalse(result["valid"])
        self.assertTrue(any("rubric's format" in p for p in result["problems"]), result["problems"])
        self.assertTrue((self.tmp / "review" / "review.json").is_file())

    def test_a_review_without_an_answer_is_invalid(self):
        result = self.reviewed("garbage")
        self.assertFalse(result["valid"])
        self.assertIn("the review session gave no valid answer", result["problems"])

    def test_the_prompts_are_the_frozen_phase_a_c_prompts_with_the_guess(self):
        review_prompt, confirm_prompt = reviewing.prompts()
        self.assertTrue(review_prompt.startswith("Review the program in TOKEN_DIR against the requirements in REQUIREMENTS"))
        self.assertTrue(confirm_prompt.startswith("A reviewer reported the finding below"))
        self.assertTrue(confirm_prompt.endswith("\n\nFINDING"))
        for prompt in (review_prompt, confirm_prompt):
            self.assertIn("variantGuess", prompt)

    def test_an_answer_is_found_in_fenced_or_plain_text(self):
        self.assertEqual(reviewer.answer_object('{"a": 1}'), {"a": 1})
        self.assertEqual(reviewer.answer_object('Here:\n```json\n{"a": 2}\n```'), {"a": 2})
        self.assertEqual(reviewer.answer_object('Result {"a": 3} done'), {"a": 3})
        self.assertIsNone(reviewer.answer_object("none"))


def judgment(commit, failed=(), build=True, tests=True, unrequested=(), protected=()):
    return {"valid": True, "commit": commit, "problems": [], "areas": {
        "correctness": {"planned": 19, "passed": 19 - len(failed), "failed": len(failed), "failedCases": list(failed)},
        "buildAndTests": {"build": {"passed": build}, "tests": {"passed": tests}},
        "scope": {"changedPaths": [{"path": "a"}, {"path": "b"}], "unrequested": list(unrequested), "protectedTouched": list(protected)}}}


class ReportTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = pathlib.Path(self._tmp.name)
        failures = HERE / "testdata" / "failures"
        launches = []

        def add(name, source, variant, edit=None, judged=None, reviewed=None):
            shutil.copytree(failures / source, self.tmp / name)
            record = json.loads((self.tmp / name / "suite.json").read_text())
            if edit:
                edit(record)
                (self.tmp / name / "suite.json").write_text(json.dumps(record))
            entry = {"id": name, "variant": variant, "task": "library-m1", "run": name}
            if judged is not None:
                (self.tmp / (name + "-judgment")).mkdir()
                (self.tmp / (name + "-judgment") / "judgment.json").write_text(json.dumps(judgment(record["result"]["commit"], **judged)))
                entry["judgment"] = name + "-judgment"
            if reviewed is not None:
                (self.tmp / (name + "-review")).mkdir()
                (self.tmp / (name + "-review") / "review.json").write_text(json.dumps({
                    "valid": True, "commit": record["result"]["commit"], "criticals": {"reported": 2, "valid": 1, "confirmed": reviewed},
                    "guesses": [{"session": "review", "variant": "K"}, {"session": "confirm-1", "variant": "M"}]}))
                entry["review"] = name + "-review"
            launches.append(entry)

        add("k-a", "result-k03", "K", judged={}, reviewed=0)
        add("k-b", "variant-capture-k01", "K", judged={"failed": ["E04-admission"], "tests": False}, reviewed=1)
        add("k-void", "provider-overloaded", "K")
        add("m-a", "result-k03", "M", judged={"unrequested": [".github/ci.yaml"], "protected": ["TASK.md"]})
        add("m-none", "result-k03", "M", edit=lambda r: r.update(delivered=False, result=None, timedOut=True))
        add("m-unjudged", "result-k03", "M")
        (self.tmp / "ledger.json").write_text(json.dumps({"format": 1, "series": "test", "tasks": {"library-m1": {
            "inputEffort": None, "note": "task.md is rendered from the model"}}, "launches": launches}))

    def tearDown(self):
        self._tmp.cleanup()

    def section(self, text, title):
        part = text.split(f"## {title}\n", 1)[1]
        return part.split("\n## ", 1)[0]

    def row(self, text, title, variant):
        return next(line for line in self.section(text, title).splitlines() if line.startswith(f"| {variant} |"))

    def test_one_table_per_area_and_no_overall_score(self):
        text, data = report.report(self.tmp / "ledger.json")
        for title in ("Human attention (primary)", "Correctness", "Build and tests", "Scope", "Code quality (secondary)", "Delivery",
                      "Void rate", "Cost (descriptive)", "Launches"):
            self.assertIn(f"## {title}\n", text)
        self.assertNotIn("score |", text.lower())
        self.assertEqual(len(data["launches"]), 6)

    def test_areas_count_results_only_and_void_launches_apart(self):
        text, _ = report.report(self.tmp / "ledger.json")
        self.assertEqual(self.row(text, "Void rate", "K"), "| K | 3 | 1/3 | provider 1 | 0 | none |")
        self.assertEqual(self.row(text, "Delivery", "K"), "| K | 2/2 | 0 | 0 |")
        self.assertEqual(self.row(text, "Correctness", "K"), "| K | 18.5 (18-19, n=2) | 19 (19-19, n=2) | 1/2 | E04-admission |")
        self.assertEqual(self.row(text, "Build and tests", "K"), "| K | 2/2 | 1/2 |")
        self.assertEqual(self.row(text, "Code quality (secondary)", "K"), "| K | 2/2 | 2 (2-2, n=2) | 1 (1-1, n=2) | 0.5 (0-1, n=2) | 2/4 |")

    def test_a_missing_or_late_result_fails_correctness_and_delivery(self):
        text, _ = report.report(self.tmp / "ledger.json")
        self.assertEqual(self.row(text, "Delivery", "M"), "| M | 2/3 | 1 | 1 |")
        self.assertEqual(self.row(text, "Correctness", "M"), "| M | 9.5 (0-19, n=2) | 19 (19-19, n=2) | 1/2 | none |")
        # m-none left every case and its step without a result; m-a has no review, so no total; m-unjudged stays out.
        self.assertEqual(self.row(text, "Human attention (primary)", "M"),
                         "| M | 3 | 20 (n=1) | 9.5 (0-19, n=2) | 0 (n=1) | 0 (n=1) | 1 (n=1) | 1 (n=1) | 0 (n=1) | 1 |")
        self.assertIn("m-unjudged: no valid judgment of this result", text)

    def test_human_attention_adds_the_defects_left(self):
        text, _ = report.report(self.tmp / "ledger.json")
        # k-a: nothing left; k-b: one failed case, failed tests and one confirmed finding.
        self.assertTrue(self.row(text, "Human attention (primary)", "K").startswith("| K | 2 | 1.5 (0-3, n=2) |"))
        self.assertIn("unavailable (task.md is rendered from the model)", text)

    def test_a_wrong_run_path_in_the_ledger_is_refused(self):
        ledger = json.loads((self.tmp / "ledger.json").read_text())
        ledger["launches"][0]["run"] = "k-typo"
        (self.tmp / "ledger.json").write_text(json.dumps(ledger))
        with self.assertRaisesRegex(ValueError, "k-typo"):
            report.report(self.tmp / "ledger.json")

    def test_the_cli_writes_the_report_once(self):
        out = self.tmp / "report"
        with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            self.assertEqual(suite.main(["report", "--ledger", str(self.tmp / "ledger.json"), "--out", str(out)]), 0)
            self.assertEqual(suite.main(["report", "--ledger", str(self.tmp / "ledger.json"), "--out", str(out)]), 2)
        self.assertTrue((out / "report.md").is_file())
        self.assertEqual(json.loads((out / "report.json").read_text())["series"], "test")


if __name__ == "__main__":
    unittest.main()
