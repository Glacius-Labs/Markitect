"""Offline tests of the pilot's arms S and F: real Git, a scripted `claude` and `markitect`, no engine and no provider."""
import json
import os
import pathlib
import re
import subprocess
import sys
import tempfile
import textwrap
import time
import unittest

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import one_agent_arm as arm  # noqa: E402
import results  # noqa: E402
import suite  # noqa: E402

MODEL = "claude:claude-sonnet-5-5"

# A scripted `claude`: the session's name is its config folder's. FAKE_DIR/script.json maps a name, or its role, to the
# steps left for it. Every session records its prompt and its time, and follows one step:
#   work          writes realization/NAME.txt and a verdict for every item of its prompt's checklist
#   none          writes realization/NAME.txt and no verdict file
#   model         also edits the model
#   problem       gives every item the verdict model-problem
#   rewrite       gives every item new evidence
#   approve       confirms every verdict
#   dispute       answers the first verdict `wrong`
#   recorded      answers the first verdict `not-shown`, the second `waived`
#   all-waived    answers every verdict `waived`
#   reopen:ITEM   confirms every verdict and answers ITEM, which is not on its list, `wrong`
#   junk          writes no review
#   rate-limit    ends on a provider error
FAKE_CLAUDE = textwrap.dedent(r"""
    import json, os, pathlib, re, sys, time
    name = pathlib.Path(os.environ["CLAUDE_CONFIG_DIR"]).name
    d = pathlib.Path(os.environ["FAKE_DIR"])
    prompt = sys.stdin.read()
    assert not any(k.startswith(("ANTHROPIC", "EVALUATION_")) for k in os.environ), "leaked variable"
    login = json.loads((pathlib.Path(os.environ["CLAUDE_CONFIG_DIR"]) / ".credentials.json").read_text())["claudeAiOauth"]
    assert "refreshToken" not in login
    (d / f"prompt-{name}.txt").write_text(prompt, encoding="utf-8")
    began = time.time()
    script_path = d / "script.json"
    role = name.split("-")[0]
    lock = d / "lock"
    while True:
        try:
            fd = os.open(lock, os.O_CREAT | os.O_EXCL)
            break
        except OSError:
            time.sleep(0.02)
    script = json.loads(script_path.read_text())
    key = name if script.get(name) else role
    step = script[key].pop(0) if script.get(key) else ("work" if role == "maker" else "approve")
    script_path.write_text(json.dumps(script))
    os.close(fd)
    os.unlink(lock)
    cwd = pathlib.Path.cwd()
    answer = re.search(r"to `([^`]+)`, which the host reads", prompt).group(1)
    items = re.findall(r"^\| `([^`]+)` \|", prompt.split("## Submitted verdicts")[0], re.M)
    if role == "maker":
        (cwd / "realization").mkdir(exist_ok=True)
        (cwd / "realization" / (name + ".txt")).write_text(name + "\n")
        if step == "model":
            (cwd / ".markitect" / "model" / "project.md").write_text("# Changed by the agent\n")
        if step not in ("none", "rate-limit"):
            word = "model-problem" if step == "problem" else "changed"
            said = "again " if step == "rewrite" else "checked "
            pathlib.Path(answer).write_text(json.dumps({"verdicts": [{"item": i, "verdict": word, "evidence": said + i} for i in items]}))
    elif step != "junk":
        reviews = [{"item": i, "answer": "confirmed", "evidence": "read it"} for i in items]
        if step == "dispute":
            reviews[0].update(answer="wrong", evidence="the limit is 6, the Definition says 5")
        elif step == "recorded":
            reviews[0].update(answer="not-shown", evidence="no test, and I could not run it")
            reviews[1].update(answer="waived", evidence="read two ways, this one is fair")
        elif step == "all-waived":
            for r in reviews:
                r.update(answer="waived", evidence="fair enough")
        elif step.startswith("reopen:"):
            reviews.append({"item": step.split(":", 1)[1], "answer": "wrong", "evidence": "the last change broke it"})
        pathlib.Path(answer).write_text(json.dumps({"reviews": reviews}))
    if os.environ.get("FAKE_SLOW"):
        time.sleep(float(os.environ["FAKE_SLOW"]))
    with open(d / "times.jsonl", "a") as f:
        f.write(json.dumps({"name": name, "began": began, "ended": time.time()}) + "\n")
    print(json.dumps({"type": "system", "subtype": "init", "model": "claude-sonnet-5-5", "session_id": name}))
    if step == "rate-limit":
        print(json.dumps({"type": "assistant", "isApiErrorMessage": True, "error": "rate_limit", "apiError": "429"}))
        sys.exit(1)
    print(json.dumps({"type": "result", "subtype": "success", "num_turns": 1, "total_cost_usd": 0.1, "usage": {"input_tokens": 10, "output_tokens": 5}}))
""")
# A scripted `markitect`: `check` fails while realization/BAD exists; `impact` prints FAKE_DIR/impact.json and records its arguments.
FAKE_MARKITECT = textwrap.dedent(r"""
    import json, os, pathlib, sys
    args = sys.argv[1:]
    d = pathlib.Path(os.environ["FAKE_DIR"])
    repo = pathlib.Path(args[args.index("--repo") + 1])
    if args[0] == "check":
        bad = (repo / "realization" / "BAD").exists()
        print(json.dumps({"errors": [{"code": "file.unregistered", "file": "realization/BAD", "message": "not registered"}] if bad else []}))
        sys.exit(1 if bad else 0)
    with open(d / "impact-calls.jsonl", "a") as f:
        f.write(json.dumps(args) + "\n")
    print((d / "impact.json").read_text())
""")


def command(tmp, name, script):
    (tmp / (name + ".py")).write_text(script)
    if os.name == "nt":
        path = tmp / (name + ".cmd")
        path.write_text(f'@"{sys.executable}" -B "{tmp / (name + ".py")}" %*\n')
    else:
        path = tmp / name
        path.write_text(f'#!/bin/sh\nexec "{sys.executable}" -B "{tmp / (name + ".py")}" "$@"\n')
        path.chmod(0o755)
    return path


def impact(**packages):
    """An impact answer: for each Package (`root` for Root), its own items and the Rules it inherits."""
    checklists = []
    for name, (own, inherited) in packages.items():
        checklists.append({"package": "" if name == "root" else name, "own": [{"definition": d, "reasons": ["modified"]} for d in own],
                           "inherited": [{"definition": d, "reasons": ["modified"]} for d in inherited], "findings": [], "stale": []})
    return {"checklists": checklists}


class OneAgentArmTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = pathlib.Path(self._tmp.name).resolve()
        self.fake = self.tmp / "fake"
        self.fake.mkdir()
        self.claude, self.markitect = command(self.tmp, "fake_claude", FAKE_CLAUDE), command(self.tmp, "fake_markitect", FAKE_MARKITECT)
        (self.tmp / "login.json").write_text(json.dumps({"claudeAiOauth": {"accessToken": "sk-ant-oat-arm-secret", "refreshToken": "sk-refresh-secret",
                                                                           "expiresAt": int((time.time() + 7200) * 1000)}}))
        self._env = dict(os.environ)
        os.environ.update(FAKE_DIR=str(self.fake), EVALUATION_LOGIN=str(self.tmp / "login.json"), EVALUATION_LABEL="markitect.evaluation.arm=eval-0123456789ab")
        self.script({})
        self.impact(impact(root=(["project"], []), lending=(["lending/loan-policy"], ["project"])))

    def tearDown(self):
        os.environ.clear()
        os.environ.update(self._env)
        self._tmp.cleanup()

    def script(self, steps):
        (self.fake / "script.json").write_text(json.dumps(steps))

    def impact(self, answer):
        (self.fake / "impact.json").write_text(json.dumps(answer))

    def model(self, name, checks="checks: []\n"):
        model = self.tmp / name
        (model / "model" / "lending").mkdir(parents=True)
        (model / "manifest.yaml").write_text("libraries:\n  std: 1.0.0\n")
        (model / "lock.yaml").write_text("lock\n")
        (model / "execution.yaml").write_text(checks)
        (model / "model" / "project.md").write_text("# Project\n\nSee [[lending/loan-policy]].\n")
        (model / "model" / "lending" / "loan-policy.md").write_text("# Policy\n\nThe limit is 5.\n")
        return model

    def run_arm(self, name, *flags, checks="checks: []\n", state=None, task=None):
        if task is None:
            (self.tmp / (name + ".md")).write_text("Build the library.\n")
            suite.make_task(self.tmp / (name + ".md"), self.model(name + "-model", checks), None, 600, self.tmp / name)
            task = self.tmp / name
        variant = [sys.executable, "-B", str(HERE / "one_agent_arm.py"), "--markitect", str(self.markitect), "--claude", str(self.claude), *flags]
        out = self.tmp / ("run-" + name)
        record = suite.run(task, variant, out, MODEL, state)
        self.assertNotIn("Traceback", (out / "variant.stderr.log").read_text(errors="replace"))
        return record, out

    def result(self, out):
        return json.loads(next((out / "variant" / "results").glob("*.json")).read_text())

    def prompt(self, name):
        return (self.fake / f"prompt-{name}.txt").read_text(encoding="utf-8")

    def names(self, out):
        return [(s["manager"], s["role"]) for s in self.result(out)["starts"]]

    def test_one_maker_and_a_fresh_checker_land_the_work(self):
        record, out = self.run_arm("single")
        self.assertEqual(record["exitCode"], 0, (out / "variant" / "run.json").read_text())
        self.assertEqual(record["reported"]["status"], "landed")
        self.assertTrue(record["modelMatches"])
        result = self.result(out)
        self.assertEqual((result["outcome"], result["state"]), ("landed", "landed"))
        self.assertEqual(self.names(out), [(".", "maker"), (".", "checker")])
        # Each reached Definition once, whoever inherits it; every verdict has evidence and approval.
        self.assertEqual([i["item"] for i in result["items"]], ["project", "lending/loan-policy"])
        self.assertTrue(all(i["approved"] and i["evidence"] for i in result["items"]))
        self.assertEqual(results.check_run(out)["problems"], [])
        # The maker sees the Definitions in full and what they link to; the checker the verdicts and the diff.
        maker, checker = self.prompt("maker-1"), self.prompt("checker-1-1")
        self.assertIn("The limit is 5.", maker)
        self.assertIn("| `project` | modified |", maker)
        self.assertIn("checked project", checker)
        self.assertIn("realization/maker-1.txt", checker)
        # The result holds the work and the owner's model, and no answer file.
        repo = self.tmp / "look"
        subprocess.run(["git", "init", "-q", "--bare", str(repo)], check=True)
        subprocess.run(["git", "-C", str(repo), "fetch", "-q", str(out / "variant" / "result.bundle"), "+refs/heads/*:refs/heads/*"], check=True, capture_output=True)
        tree = suite.git(repo, "ls-tree", "-r", "--name-only", "result").splitlines()
        self.assertIn("realization/maker-1.txt", tree)
        self.assertIn(".markitect/model/project.md", tree)
        self.assertFalse([p for p in tree if "maker-1.json" in p or "verdicts" in p])
        # The login never reaches the evidence.
        self.assertFalse([p for p in out.rglob("*") if p.is_file() and b"sk-ant-oat-arm-secret" in p.read_bytes()])

    def test_a_rejection_starts_a_new_maker_with_the_defects(self):
        self.script({"checker": ["dispute", "approve"]})
        record, out = self.run_arm("rejected")
        self.assertEqual(record["reported"]["status"], "landed")
        self.assertEqual(self.names(out), [(".", "maker"), (".", "checker"), (".", "maker"), (".", "checker")])
        self.assertIn("the limit is 6", self.prompt("maker-2"))
        refusals = self.result(out)["refusals"]
        self.assertEqual([(r["attempt"], r["by"], list(r["items"])) for r in refusals], [(1, "checker", ["project"])])

    def test_only_wrong_refuses_and_the_rest_land_as_findings(self):
        self.script({"checker": ["recorded"]})
        record, out = self.run_arm("recorded")
        self.assertEqual(record["reported"]["status"], "landed", (out / "variant" / "run.json").read_text())
        self.assertEqual(self.names(out), [(".", "maker"), (".", "checker")])
        result = self.result(out)
        self.assertEqual(result["refusals"], [])
        self.assertEqual(result["findings"], [{"definition": "project", "answer": "not-shown", "observed": "no test, and I could not run it"},
                                              {"definition": "lending/loan-policy", "answer": "waived", "observed": "read two ways, this one is fair"}])
        self.assertEqual([i["review"] for i in result["items"]], ["not-shown", "waived"])
        self.assertEqual(record["reported"]["findings"], 2)
        self.assertTrue(results.check_run(out)["complete"], results.check_run(out))
        self.assertIn("only answer that refuses", self.prompt("checker-1-1"))
        self.assertNotIn("What an earlier review ruled", self.prompt("checker-1-1"))

    def test_a_later_round_rechecks_only_what_was_refused(self):
        self.script({"checker": ["dispute", "approve"]})
        record, out = self.run_arm("narrow")
        self.assertEqual(record["reported"]["status"], "landed")
        later = self.prompt("checker-2-1")
        self.assertIn("| `project` |", later)
        self.assertNotIn("| `lending/loan-policy` |", later)
        self.assertIn("The other items: `lending/loan-policy`.", later)
        # It gets the reason, and the diff since the last review with the way to see the whole work.
        self.assertIn("the limit is 6", later)
        self.assertIn("realization/maker-2.txt", later)
        self.assertNotIn("realization/maker-1.txt", later)
        result = self.result(out)
        self.assertEqual([i["review"] for i in result["items"]], ["confirmed", "confirmed"])
        self.assertEqual(result["findings"], [])

    def test_a_verdict_the_maker_changed_is_checked_again(self):
        self.script({"maker": ["work", "rewrite"], "checker": ["dispute", "approve"]})
        record, out = self.run_arm("rewritten")
        self.assertEqual(record["reported"]["status"], "landed")
        later = self.prompt("checker-2-1")
        self.assertIn("| `project` |", later)
        self.assertIn("| `lending/loan-policy` |", later)

    def test_a_later_review_reopens_an_item_the_diff_broke(self):
        self.script({"checker": ["dispute", "reopen:lending/loan-policy", "approve"]})
        record, out = self.run_arm("reopened")
        self.assertEqual(record["reported"]["status"], "landed", (out / "variant" / "run.json").read_text())
        self.assertEqual([list(r["items"]) for r in self.result(out)["refusals"]], [["project"], ["lending/loan-policy"]])
        last = self.prompt("checker-3-1")
        self.assertIn("| `lending/loan-policy` |", last)
        self.assertNotIn("| `project` |", last)
        self.assertIn("the last change broke it", self.prompt("maker-3"))

    def test_an_answer_outside_the_checklist_is_ignored(self):
        self.script({"checker": ["reopen:not/on-the-list"]})
        record, out = self.run_arm("outside")
        self.assertEqual(record["reported"]["status"], "landed")
        self.assertEqual(self.result(out)["refusals"], [])

    def test_a_refusal_by_the_gate_keeps_the_rulings(self):
        self.script({"maker": ["work", "none", "work"], "checker": ["dispute", "approve"]})
        record, out = self.run_arm("gate-between")
        self.assertEqual(record["reported"]["status"], "landed", (out / "variant" / "run.json").read_text())
        self.assertEqual(self.names(out), [(".", "maker"), (".", "checker"), (".", "maker"), (".", "maker"), (".", "checker")])
        later = self.prompt("checker-3-1")
        self.assertIn("| `project` |", later)
        self.assertNotIn("| `lending/loan-policy` |", later)
        self.assertIn("the limit is 6", later)
        self.assertIn("realization/maker-2.txt", later)
        self.assertNotIn("realization/maker-1.txt", later)

    def test_any_other_answer_to_a_model_problem_refuses_it(self):
        self.script({"maker": ["problem", "problem"], "checker": ["all-waived", "approve"]})
        record, out = self.run_arm("waived-problem")
        self.assertEqual((record["reported"]["status"], record["exitCode"]), ("model-defect", 0))
        self.assertEqual(record["result"]["changedPaths"], [])
        result = self.result(out)
        self.assertEqual([(r["by"], sorted(r["items"])) for r in result["refusals"]], [("checker", ["lending/loan-policy", "project"])])
        self.assertEqual(result["findings"], [])

    def test_the_gate_refuses_work_on_the_model_a_missing_verdict_and_a_failing_check(self):
        self.script({"maker": ["model", "work"]})
        record, out = self.run_arm("model-edit")
        self.assertEqual(record["reported"]["status"], "incomplete")
        self.assertIn(".markitect/model/project.md is protected", self.prompt("maker-2"))
        self.assertEqual(record["delivered"], True)
        self.assertEqual(record["result"]["changedPaths"], [])
        self.script({"maker": ["none", "work"]})
        record, out = self.run_arm("no-verdict")
        self.assertEqual(record["reported"]["status"], "landed")
        self.assertIn("project has no verdict", self.prompt("maker-2"))
        self.assertEqual(self.names(out)[:2], [(".", "maker"), (".", "maker")])
        fail = "checks:\n  - name: test\n    command: python -c \"import sys, pathlib; pathlib.Path('left.txt').write_text('x'); sys.exit(3)\"\n    directory: .\n    timeout: 1m\n"
        (self.fake / "times.jsonl").unlink()
        record, out = self.run_arm("failing-check", "--rounds", "2", checks=fail)
        self.assertEqual(record["reported"]["status"], "incomplete")
        self.assertIn("the required check test failed", self.prompt("maker-2"))
        self.assertEqual(self.result(out)["requiredChecks"]["checks"][0]["passed"], False)

    def test_refusals_up_to_the_limit_stop_the_run_and_deliver_the_start(self):
        self.script({"checker": ["dispute", "dispute"]})
        record, out = self.run_arm("limit", "--rounds", "2")
        self.assertEqual(record["reported"]["status"], "incomplete")
        self.assertEqual(record["exitCode"], 1)
        self.assertIn("used its 2 attempts", record["reported"]["cause"])
        self.assertEqual(record["result"]["changedPaths"], [])
        self.assertEqual(self.result(out)["outcome"], "not-landed")

    def test_a_confirmed_model_problem_ends_the_run_without_landing(self):
        self.script({"maker": ["problem"]})
        record, out = self.run_arm("defect")
        self.assertEqual((record["reported"]["status"], record["exitCode"]), ("model-defect", 0))
        self.assertEqual(record["result"]["changedPaths"], [])
        self.assertEqual(self.result(out)["items"][0]["verdict"], "model-problem")

    def test_nothing_reached_starts_no_agent(self):
        self.impact(impact(root=([], [])))
        record, out = self.run_arm("nothing")
        self.assertEqual((record["reported"]["status"], record["exitCode"]), ("nothing-to-do", 0))
        self.assertEqual(self.result(out)["starts"], [])

    def test_federal_runs_the_packages_side_by_side_and_then_root(self):
        self.impact(impact(root=(["project"], []), lending=(["lending/loan-policy"], ["project"]), members=([], ["project"])))
        os.environ["FAKE_SLOW"] = "1"
        self.script({"checker": ["dispute", "approve"]})
        record, out = self.run_arm("federal", "--shape", "federal")
        self.assertEqual(record["reported"]["status"], "landed", (out / "variant" / "run.json").read_text())
        times = {t["name"]: t for t in map(json.loads, (self.fake / "times.jsonl").read_text().splitlines())}
        lending, members, root = times["maker-lending"], times["maker-members"], times["maker-1"]
        self.assertLess(lending["began"], members["ended"])
        self.assertLess(members["began"], lending["ended"])
        self.assertGreaterEqual(root["began"], max(lending["ended"], members["ended"]))
        # A Package's prompt is bounded: its own item as the checklist, the inherited Rule named, nothing of another's.
        prompt = self.prompt("maker-lending")
        self.assertIn("| `lending/loan-policy` |", prompt)
        self.assertNotIn("| `project` |", prompt)
        self.assertIn("`project`", prompt)
        self.assertIn("one of 2 Packages", prompt)
        # Root integrates and gets what they reported. A rejection goes back to it alone.
        self.assertIn("What the Packages reported", self.prompt("maker-1"))
        self.assertIn("checked lending/loan-policy", self.prompt("maker-1"))
        self.assertEqual([s["manager"] for s in self.result(out)["starts"] if s["role"] == "maker"][-2:], [".", "."])
        self.assertIn("the limit is 6", self.prompt("maker-2"))
        self.assertFalse(any((self.fake / f"prompt-maker-{p}-2.txt").exists() for p in ("lending", "members")))

    def test_a_chain_step_compares_with_the_last_landing(self):
        state = self.tmp / "state"
        state.mkdir()
        record, out = self.run_arm("step-1", state=state)
        self.assertEqual(record["reported"]["status"], "landed")
        landed = json.loads((state / "markitect-arm.json").read_text())["landed"]
        (self.tmp / "step-2.md").write_text("Change the library.\n")
        suite.make_task(self.tmp / "step-2.md", self.model("step-2-model"), out / "variant" / "result.bundle", 600, self.tmp / "step-2")
        for p in (self.fake / "prompt-maker-1.txt", self.fake / "prompt-checker-1-1.txt"):
            p.unlink()
        record, out = self.run_arm("step-2", state=state, task=self.tmp / "step-2")
        self.assertEqual(record["reported"]["status"], "landed", (out / "variant" / "run.json").read_text())
        calls = [json.loads(x) for x in (self.fake / "impact-calls.jsonl").read_text().splitlines()]
        self.assertEqual(calls[1][calls[1].index("--base") + 1], landed)

    def test_a_provider_failure_stops_and_is_named(self):
        self.script({"maker": ["rate-limit"]})
        record, out = self.run_arm("provider")
        self.assertEqual(record["reported"]["status"], "incomplete")
        start = self.result(out)["starts"][0]
        self.assertEqual((start["reason"], start["class"]), ("failed", "rate-limit"))

    def test_no_review_twice_stops_the_run(self):
        self.script({"checker": ["junk", "junk"]})
        record, out = self.run_arm("no-review")
        self.assertEqual(record["reported"]["status"], "incomplete")
        self.assertIn("no complete review", record["reported"]["cause"])
        self.assertIn("left out these items", self.prompt("checker-1-2"))

    def test_a_review_that_left_items_out_is_completed_on_the_second_try(self):
        self.script({"checker-1-1": ["junk"], "checker-1-2": ["approve"]})
        record, out = self.run_arm("partial")
        self.assertEqual(record["reported"]["status"], "landed", (out / "variant" / "run.json").read_text())
        self.assertEqual(self.names(out), [(".", "maker"), (".", "checker"), (".", "checker")])

    def test_the_cost_and_a_check_in_a_missing_folder_are_reported(self):
        fail = "checks:\n  - name: build\n    command: go build\n    directory: not-there\n    timeout: 1m\n"
        record, out = self.run_arm("missing-folder", "--rounds", "1", checks=fail)
        self.assertEqual(record["reported"]["status"], "incomplete")
        self.assertIn("the check could not run", json.dumps(self.result(out)["requiredChecks"]))
        record, out = self.run_arm("cost")
        self.assertEqual(record["reported"]["usage"]["costUsd"], 0.2)
        # The answers stay out of Git.
        repo = self.tmp / "look-cost"
        subprocess.run(["git", "init", "-q", "--bare", str(repo)], check=True)
        subprocess.run(["git", "-C", str(repo), "fetch", "-q", str(out / "variant" / "result.bundle"), "+refs/heads/*:refs/heads/*"], check=True, capture_output=True)
        self.assertFalse([f for f in suite.git(repo, "ls-tree", "-r", "--name-only", "result").splitlines() if f.startswith(".arm")])

    def test_small_readers(self):
        with tempfile.TemporaryDirectory() as t:
            repo = pathlib.Path(t)
            (repo / ".markitect").mkdir()
            (repo / ".markitect" / "execution.yaml").write_text("checks:\n  - name: build\n    command: go build ./cmd/x\n    directory: realization\n    timeout: 2m\n    environment: golang:1.27\n"
                                                                 "  - name: test\n    command: go test ./...\n    directory: realization\n    timeout: 90s\n")
            checks = arm.required_checks(repo)
            self.assertEqual([(c["name"], c["command"], c["directory"]) for c in checks],
                             [("build", "go build ./cmd/x", "realization"), ("test", "go test ./...", "realization")])
            self.assertEqual([arm.seconds_of(c["timeout"]) for c in checks], [120, 90])
            model = repo / ".markitect" / "model"
            (model / "a").mkdir(parents=True)
            (model / "a" / "one.md").write_text("# One\n\nSee [[two]] and [[std/Rule]].\n\n```\nx\n```\n")
            (model / "two.md").write_text("# Two\n")
            (model / "three.md").write_text("# Three\n")
            text = arm.model_text(repo, ["a/one", "two"])
            self.assertEqual(re.findall(r"^### `([^`]+)`", text, re.M), ["a/one", "two"])
            self.assertIn("````markdown", text)
            path = repo / "v.json"
            path.write_text(json.dumps({"verdicts": [{"item": "a", "verdict": "changed", "evidence": "x"}, {"item": "b", "verdict": "maybe", "evidence": "x"},
                                                     {"item": "c", "verdict": "changed", "evidence": " "}, {"item": "z", "verdict": "changed", "evidence": "x"}]}))
            found, problems = arm.read_verdicts(path, {"a", "b", "c"})
            self.assertEqual(list(found), ["a"])
            self.assertEqual(len(problems), 3)
            self.assertEqual(arm.read_verdicts(repo / "absent.json", {"a"}), ({}, []))
            path.write_text("not json")
            self.assertEqual(len(arm.read_verdicts(path, {"a"})[1]), 1)
            items = [{"item": "a", "package": "", "why": ""}]
            path.write_text(json.dumps({"reviews": [{"item": "a", "answer": "confirmed"}]}))
            self.assertEqual(arm.read_reviews(path, items, {"a"})[0]["a"]["answer"], "confirmed")
            self.assertEqual(arm.read_reviews(path, items, {"a"})[1], [])
            path.write_text(json.dumps({"reviews": [{"item": "a", "answer": "disputed"}, {"item": "extra", "answer": "confirmed"}]}))
            self.assertEqual(arm.read_reviews(path, items, {"a"}), ({}, ["a"]))
            # An item of the checklist that is not on this round's list counts, and leaves the listed one missing.
            found, missing = arm.read_reviews(path, items, {"a", "extra"})
            self.assertEqual((list(found), missing), (["extra"], ["a"]))
            path.write_text("not json")
            self.assertEqual(arm.read_reviews(path, items, {"a"}), ({}, ["a"]))
            self.assertEqual(arm.findings_of({"a": {"answer": "confirmed", "evidence": "x"}, "b": {"answer": "waived", "evidence": "y", "verdict": {}}}),
                             [{"definition": "b", "answer": "waived", "observed": "y"}])
        listed, inherited = arm.checklist({"checklists": [
            {"package": "", "own": [{"definition": "project", "reasons": ["modified", "linked"]}], "inherited": [], "findings": [],
             "stale": [{"code": "stale.file", "package": "", "pageFile": ".markitect/registry.md", "file": "gone.go"}]},
            {"package": "m", "own": [], "inherited": [{"definition": "project", "reasons": ["modified"]}],
             "findings": [{"definition": "m/x", "observed": "o", "expected": "e"}], "stale": []}]})
        self.assertEqual([(i["item"], i["package"]) for i in listed],
                         [("project", ""), ("stale.file gone.go on .markitect/registry.md", ""), ("finding 1 on m/x", "m")])
        self.assertEqual(inherited, {"": [], "m": ["project"]})


if __name__ == "__main__":
    unittest.main()
