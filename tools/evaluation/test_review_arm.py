"""Offline tests of the checker's review arm: real Git, a scripted `claude` and `markitect`, no engine and no provider."""
import json
import os
import pathlib
import subprocess
import sys
import tempfile
import time
import unittest

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import suite  # noqa: E402
import test_one_agent_arm as base  # noqa: E402

MODEL = "claude:claude-sonnet-5-5"
ITEMS = [{"item": "project", "package": "", "why": "modified"}, {"item": "lending/loan-policy", "package": "lending", "why": "modified"}]
VERDICTS = {"project": {"verdict": "changed", "evidence": "the maker says it holds"}, "lending/loan-policy": {"verdict": "conforms", "evidence": "a test shows the limit"}}


class ReviewArmTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = pathlib.Path(self._tmp.name).resolve()
        self.fake = self.tmp / "fake"
        self.fake.mkdir()
        self.claude, self.markitect = base.command(self.tmp, "fake_claude", base.FAKE_CLAUDE), base.command(self.tmp, "fake_markitect", base.FAKE_MARKITECT)
        (self.tmp / "login.json").write_text(json.dumps({"claudeAiOauth": {"accessToken": "sk-ant-oat-arm-secret", "refreshToken": "sk-refresh-secret",
                                                                           "expiresAt": int((time.time() + 7200) * 1000)}}))
        self._env = dict(os.environ)
        os.environ.update(FAKE_DIR=str(self.fake), EVALUATION_LOGIN=str(self.tmp / "login.json"), EVALUATION_LABEL="markitect.evaluation.arm=eval-0123456789ab")
        self.script({})

    def tearDown(self):
        os.environ.clear()
        os.environ.update(self._env)
        self._tmp.cleanup()

    def script(self, steps):
        (self.fake / "script.json").write_text(json.dumps(steps))

    def git(self, repo, *args):
        env = dict(os.environ, GIT_AUTHOR_NAME="Owner", GIT_AUTHOR_EMAIL="o@x", GIT_COMMITTER_NAME="Owner", GIT_COMMITTER_EMAIL="o@x")
        return subprocess.run(["git", "-C", str(repo), *args], check=True, capture_output=True, text=True, env=env).stdout.strip()

    def task(self, name, checks="checks: []\n", bad=False):
        """A start commit, the owner's commit with the model, the maker's work on it, as a one-branch bundle and a task."""
        repo = self.tmp / (name + "-repo")
        repo.mkdir()
        self.git(repo, "init", "-q", "-b", "result")
        (repo / "start.txt").write_text("start\n")
        self.git(repo, "add", "-A")
        self.git(repo, "commit", "-qm", "Start")
        dot = repo / ".markitect"
        (dot / "model" / "lending").mkdir(parents=True)
        (dot / "execution.yaml").write_text(checks)
        (dot / "model" / "project.md").write_text("# Project\n\nSee [[lending/loan-policy]].\n")
        (dot / "model" / "lending" / "loan-policy.md").write_text("# Policy\n\nThe limit is 5.\n")
        self.git(repo, "add", "-A")
        self.git(repo, "commit", "-qm", "Owner: the model of this task")
        owner = self.git(repo, "rev-parse", "HEAD")
        (repo / "realization").mkdir()
        (repo / "realization" / "code.txt").write_text("limit = 5\n")
        if bad:
            (repo / "realization" / "BAD").write_text("x\n")
        self.git(repo, "add", "-A")
        self.git(repo, "commit", "-qm", "Work of attempt 1")
        bundle = self.tmp / (name + ".bundle")
        self.git(repo, "bundle", "create", "-q", str(bundle), "refs/heads/result")
        (self.tmp / (name + ".md")).write_text("Review the submission.\n")
        suite.make_task(self.tmp / (name + ".md"), None, bundle, 600, self.tmp / name)
        submission = self.tmp / (name + ".json")
        submission.write_text(json.dumps({"owner": owner, "items": ITEMS, "verdicts": VERDICTS}))
        return self.tmp / name, submission

    def run_arm(self, name, answers, **task):
        path, submission = self.task(name, **task)
        variant = [sys.executable, "-B", str(HERE / "review_arm.py"), "--markitect", str(self.markitect), "--claude", str(self.claude), "--answers", answers, "--submission", str(submission)]
        out = self.tmp / ("run-" + name)
        record = suite.run(path, variant, out, MODEL)
        self.assertNotIn("Traceback", (out / "variant.stderr.log").read_text(errors="replace"))
        return record, out

    def result(self, out):
        return json.loads(next((out / "variant" / "results").glob("*.json")).read_text())

    def prompt(self, name):
        return (self.fake / f"prompt-{name}.txt").read_text(encoding="utf-8")

    def test_the_new_answers_review_and_deliver_the_start_unchanged(self):
        self.script({"checker": ["recorded"]})
        record, out = self.run_arm("new", "new")
        self.assertEqual((record["exitCode"], record["reported"]["status"]), (0, "reviewed"), (out / "variant" / "run.json").read_text())
        self.assertTrue(record["delivered"])
        self.assertEqual(record["result"]["changedPaths"], [])
        self.assertTrue(record["modelMatches"])
        result = self.result(out)
        self.assertEqual({k: v["answer"] for k, v in result["reviews"].items()}, {"project": "not-shown", "lending/loan-policy": "waived"})
        self.assertEqual(result["refused"], [])
        self.assertEqual([(s["role"], s["reason"]) for s in result["starts"]], [("checker", "finished")])
        prompt = self.prompt("checker-1-1")
        self.assertIn("this is the only answer that refuses", prompt)
        self.assertIn("the maker says it holds", prompt)
        self.assertIn("realization/code.txt", prompt)
        self.assertFalse([p for p in out.rglob("*") if p.is_file() and b"sk-ant-oat-arm-secret" in p.read_bytes()])

    def test_the_old_answers_use_the_old_prompt_and_count_a_dispute(self):
        self.script({"checker": ["old-dispute"]})
        record, out = self.run_arm("old", "old")
        self.assertEqual(record["reported"]["status"], "reviewed", (out / "variant" / "run.json").read_text())
        result = self.result(out)
        self.assertEqual(result["refused"], ["project"])
        self.assertEqual(result["reviews"]["project"]["answer"], "disputed")
        prompt = self.prompt("checker-1-1")
        self.assertIn("Dispute a `conforms` or `changed` verdict", prompt)
        self.assertNotIn("not-shown", prompt)

    def test_an_answer_from_the_other_set_does_not_count(self):
        self.script({"checker": ["dispute", "dispute"]})
        record, out = self.run_arm("mixed", "old")
        self.assertEqual(record["reported"]["status"], "incomplete")
        self.assertEqual(record["exitCode"], 1)
        # `wrong` is no answer in the old set; `confirmed` is, so only the first item is asked for again.
        retry = self.prompt("checker-1-2")
        self.assertIn("| `project` |", retry)
        self.assertNotIn("| `lending/loan-policy` |", retry)

    def test_a_submission_that_fails_the_gate_starts_no_agent(self):
        record, out = self.run_arm("failing", "new", bad=True)
        self.assertEqual(record["reported"]["status"], "could-not-run")
        self.assertIn("does not pass the gate", record["reported"]["cause"])
        self.assertFalse((self.fake / "prompt-checker-1-1.txt").exists())
        fail = "checks:\n  - name: test\n    command: python -c \"import sys; sys.exit(3)\"\n    directory: .\n    timeout: 1m\n"
        record, out = self.run_arm("failing-check", "new", checks=fail)
        self.assertEqual(record["reported"]["status"], "could-not-run")
        self.assertFalse((self.fake / "prompt-checker-1-1.txt").exists())

    def test_a_provider_failure_is_named(self):
        self.script({"checker": ["rate-limit"]})
        record, out = self.run_arm("provider", "new")
        self.assertEqual(record["reported"]["status"], "incomplete")
        self.assertEqual(self.result(out)["starts"][0]["class"], "rate-limit")

    def test_the_old_prompt_is_the_one_the_first_set_launched(self):
        """Word for word what commit ce16a754f's `checker_prompt` holds, as far as the two are fixed text."""
        old = subprocess.run(["git", "-C", str(HERE), "show", "ce16a754f:tools/evaluation/one_agent_arm.py"], capture_output=True, text=True, encoding="utf-8")
        if old.returncode != 0:
            self.skipTest("the commit is not in this checkout")
        import review_arm
        source = pathlib.Path(review_arm.__file__).read_text(encoding="utf-8")
        start = old.stdout.index('    return f"""# Task')
        text = old.stdout[start:old.stdout.index('"""\n\n\ndef read_reviews')]
        self.assertIn(text.replace("{table(items)}", "{one.table(items)}").replace("{model_text(", "{one.model_text("), source)


if __name__ == "__main__":
    unittest.main()
