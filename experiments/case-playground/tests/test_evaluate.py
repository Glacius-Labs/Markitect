"""The assessment: stations, holdouts, diff profile, classification, reviews and reports
on a synthetic run folder (built with the lifecycle), plus the host command with a fake
Docker. No Docker, no provider: reviewers are tests/fake_reviewer.py."""

import contextlib
import hashlib
import io
import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from playground import __main__ as entry
from playground import codex_agent, evaluate, host, lifecycle

PLAYGROUND = Path(__file__).resolve().parents[1]
FAKE = [sys.executable, str(Path(__file__).resolve().parent / "fake_reviewer.py")]
TOKEN = "fake-oauth-token-for-assessment-tests"
HOLDOUT = '''import argparse, json, pathlib
parser = argparse.ArgumentParser()
parser.add_argument("--repo")
parser.add_argument("--station", type=int)
parser.add_argument("--deadline", type=float)
args = parser.parse_args()
repo = pathlib.Path(args.repo)
checks = [{"id": "app-exists", "status": "PASS" if (repo / "app.py").is_file() else "FAIL", "item": "B01",
           "rule": None, "detail": ""},
          {"id": "no-git", "status": "PASS" if not (repo / ".git").exists() else "FAIL", "item": None,
           "rule": "R2", "detail": ""}]
if args.station >= 2:
    checks.append({"id": "audit-line", "status": "FAIL", "item": "B02", "rule": "R1", "detail": "no audit line"})
print(json.dumps({"station": args.station, "checks": checks}))
'''


RUNTIME_YAML = """agents:
  '["project-owner"]':
    transport: codex-app-server
    providerVersion: 0.162.0
    model: gpt-6-luna
    appServer:
      reasoningEffort: high
verifier:
  transport: codex-app-server
  model: gpt-6-luna
  effort: high
"""


def quiet():
    stack = contextlib.ExitStack()
    stack.enter_context(contextlib.redirect_stdout(io.StringIO()))
    stack.enter_context(contextlib.redirect_stderr(io.StringIO()))
    return stack


def write(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8", newline="\n")


def jsonl(events: list[dict]) -> str:
    return "".join(json.dumps(event) + "\n" for event in events)


def mcp(item_id: str, tool: str, kind: str, **fields) -> dict:
    return {"type": kind, "item": {"id": item_id, "type": "mcp_tool_call", "server": "markitect", "tool": tool,
                                   "arguments": {"goal": "x"}, **fields}}


S1_EVENTS = [
    {"type": "thread.started", "thread_id": "t1"},
    mcp("i1", "project_check", "item.started"),
    mcp("i1", "project_check", "item.completed", status="completed", result={"content": [], "structured_content": {}}),
    mcp("i2", "project_explore", "item.started"),
    mcp("i2", "project_explore", "item.completed", status="failed", error=None, result={
        "content": [{"type": "text", "text": "{}"}],
        "structured_content": {"diagnostic": {"code": "host_rejected", "message": "The shared Host rejected this."},
                               "data": {"report": {"findings": [{"code": "scope_missing"}]}}}}),
    mcp("i3", "project_explore", "item.completed", status="failed",
        error={"message": "Mcp error: -32602: Arguments do not match closed tool schema"}, result=None),
    {"type": "item.completed", "item": {"id": "i4", "type": "command_execution",
                                        "command": "/bin/bash -lc 'markitect project plan --repo .'",
                                        "aggregated_output": "error: plan rejected", "exit_code": 2}},
    mcp("i5", "project_plan", "item.started"),
    {"type": "turn.completed", "usage": {"input_tokens": 1, "cached_input_tokens": 0, "output_tokens": 1}},
]


def make_run(root: Path, *, method: str = "markitect", stations: int = 2) -> Path:
    """A finished run folder: host.json, inputs/, results/ with audit snapshots and station records."""
    run = root / "run"
    inputs, results = run / "inputs", run / "results"
    shutil.copytree(PLAYGROUND / "cases", inputs / "cases", ignore=shutil.ignore_patterns("__pycache__"))
    repo, audit = root / "work" / "readinglog", results / "audit"
    lifecycle.prepare(inputs / "cases", repo, audit, case="readinglog", method=method)
    write(repo / ".markitect" / "model.yaml", "managers: [root]\n")
    if method == "markitect":  # a v1 setup record holds no roles; the assessment reads them back
        write(repo / ".markitect" / "runtime.yaml", RUNTIME_YAML)
    lifecycle.git(repo, "add", "-A")
    lifecycle.git(repo, "commit", "-q", "-m", "Install workflow")
    setup_commit = lifecycle.git(repo, "rev-parse", "HEAD")
    write(results / "setup" / "05-setup-write.stderr.txt", "warning: cost cap is approximate\n")
    (results / "setup").mkdir(parents=True, exist_ok=True)
    (results / "setup" / "setup.json").write_text(json.dumps({
        "status": "ready", "method": method, "seconds": 1.5, "commit": setup_commit,
        "steps": [{"name": "init-write", "exitCode": 0, "seconds": 0.2, "stdout": "01.txt", "stderr": "01.err"},
                  {"name": "setup-write", "exitCode": 1, "seconds": 0.3, "stdout": "05-setup-write.stdout.txt",
                   "stderr": "05-setup-write.stderr.txt"}]}), encoding="utf-8")
    for number in range(1, stations + 1):
        folder = results / "stations" / f"S{number}"
        if number == 1:
            write(repo / "app.py", (repo / "app.py").read_text(encoding="utf-8") + "\n# finish\n")
            write(repo / "tests" / "test_finish.py", "import unittest\n")
            write(repo / "README.md", (repo / "README.md").read_text(encoding="utf-8") + "\nFinish docs.\n")
            write(repo / ".markitect" / "model.yaml", "managers: [root, finish]\n")
            write(folder / "events.jsonl", jsonl(S1_EVENTS))
            agent = {"exitCode": 0, "timedOut": False, "sessionId": "t1"}
        else:
            write(repo / "app.py", (repo / "app.py").read_text(encoding="utf-8") + f"\n# wave {number}\n")
            write(folder / "events.jsonl", jsonl([{"type": "thread.started", "thread_id": "t1"},
                                                  {"type": "error", "message": "stream disconnected: 429 Too Many Requests"}]))
            write(folder / "stderr.log", "Reading additional input from stdin...\n")
            agent = {"exitCode": 1, "timedOut": False, "sessionId": "t1"}
        lifecycle.git(repo, "add", "-A")
        lifecycle.git(repo, "commit", "-q", "-m", f"Wave {number}")
        write(folder / "agent.json", json.dumps(agent))
        write(folder / "checks.json", json.dumps({"passed": 1, "total": 5, "status": "fail"}))
        write(folder / "last-message.txt", f"S{number} done; all checks pass.")
        lifecycle.snapshot(repo, audit)
        if number < stations:
            lifecycle.advance(repo, audit)
    lifecycle.freeze(repo, audit, reason="completed")
    manifest = {"schema": 1, "id": f"{method[:4]}-readinglog-test", "case": "readinglog", "method": method,
                "agent": {"kind": "codex", "codexVersion": "0.162.0", "model": "m", "effort": "high",
                          "maxSubagents": 2},
                "limits": {"stationSeconds": 60, "totalSeconds": 600},
                "container": {"cpus": 2, "memory": "4g", "pidsLimit": 512}}
    fairness = {"case": "readinglog", "codex": "0.162.0", "imageId": "sha256:img", "model": "m", "effort": "high",
                "maxSubagentsPerSession": 2, "limits": manifest["limits"], "container": manifest["container"]}
    write(run / "host.json", json.dumps({"manifest": manifest, "status": "completed",
                                         "image": {"tag": "markitect-playground:codex-0.162.0", "id": "sha256:img"}}))
    write(results / "runner.json", json.dumps({"manifest": manifest, "stopReason": None}))
    write(results / "report.json", json.dumps({"manifest": manifest, "status": "completed", "fairness": fairness,
                                               "versions": {"markitectCommit": "f12ffb00"},
                                               "totals": {"agentSeconds": 10.0}}))
    return run


def make_evaluation(root: Path, *, case: bool = True) -> Path:
    evaluation = root / "evaluation"
    shutil.copytree(PLAYGROUND / "evaluation" / "common", evaluation / "common")
    config = json.loads((PLAYGROUND / "evaluation" / "config.json").read_text(encoding="utf-8"))
    config["promptMaxBytes"] = 20_000  # Windows limits one command line to 32k characters
    for cfg in config["reviewers"].values():
        cfg["timeoutSeconds"] = 60
    write(evaluation / "config.json", json.dumps(config))
    if case:
        write(evaluation / "readinglog" / "holdout.py", HOLDOUT)
        write(evaluation / "readinglog" / "ground-truth.json", json.dumps({
            "case": "readinglog", "rules": [{"id": "R1", "text": "audit line"}],
            "waves": [{"station": "S1", "items": [{"id": "B01", "obligations": ["finish", "idempotent", "status"],
                                                   "areas": ["finish"], "rules": [], "mustNotChange": []}]},
                      {"station": "S2", "items": [{"id": "B02", "obligations": ["import"]}]}]}))
        write(evaluation / "readinglog" / "reference" / "S1" / "SECRET.md", "reference implementation\n")
    return evaluation


class AssessRunTests(unittest.TestCase):
    """One assessed Markitect-style run with holdouts, ground truth and both fake reviewers."""

    @classmethod
    def setUpClass(cls):
        cls.temp = tempfile.TemporaryDirectory()
        cls.root = Path(cls.temp.name).resolve()
        os.chmod(cls.root, 0o755)
        empty = cls.root / "empty.gitconfig"
        empty.write_text("", encoding="utf-8")
        cls.env = mock.patch.dict(os.environ, {"GIT_CONFIG_GLOBAL": str(empty), "GIT_CONFIG_NOSYSTEM": "1"})
        cls.env.start()
        cls.run_dir = make_run(cls.root)
        cls.evaluation = make_evaluation(cls.root)
        (cls.root / "auth.json").write_text('{"fake": "login"}', encoding="utf-8")
        (cls.root / "token").write_text(TOKEN + "\n", encoding="utf-8")
        cls.out = cls.run_dir / "assessment"
        with quiet():
            cls.report = evaluate.assess_run(
                cls.run_dir, cls.evaluation, cls.out, reviewer_names=["codex", "claude"],
                codex_auth=cls.root / "auth.json", claude_token_file=cls.root / "token",
                executables={"codex": FAKE, "claude": FAKE}, evaluation={"commit": "abc123", "dirty": False},
                image="sha256:img")

    @classmethod
    def tearDownClass(cls):
        cls.env.stop()
        cls.temp.cleanup()

    def station(self, name: str) -> dict:
        return next(entry for entry in self.report["stations"] if entry["station"] == name)

    def test_report_header_and_files(self):
        report = self.report
        self.assertEqual((report["kind"], report["run"]["id"], report["run"]["outerProvider"]),
                         ("assessment", "mark-readinglog-test", "codex"))
        self.assertEqual(report["run"]["fairness"]["imageId"], "sha256:img")
        self.assertEqual(report["evaluation"]["commit"], "abc123")
        truth = self.evaluation / "readinglog" / "ground-truth.json"
        self.assertEqual(report["evaluation"]["files"]["groundTruth"]["sha256"],
                         hashlib.sha256(truth.read_bytes()).hexdigest())
        for key in ("holdout", "reviewerPrompt", "reviewerSchema", "config"):
            self.assertRegex(report["evaluation"]["files"][key]["sha256"], r"^[0-9a-f]{64}$")
        self.assertEqual(report["reviewers"]["codex"]["model"], "gpt-6.1-sol")
        self.assertEqual(report["reviewers"]["claude"]["version"], "fake-reviewer 1.0")
        for name in ("report.json", "report.md", "product-findings.md"):
            self.assertTrue((self.out / name).is_file(), name)
        markdown = (self.out / "report.md").read_text(encoding="utf-8")
        self.assertIn("not blind", markdown)
        self.assertIn("| S2 |", markdown)
        self.assertIn("shared finding", markdown)

    def test_stratum_and_roles_of_a_run_from_before_they_were_recorded(self):
        run = self.report["run"]
        self.assertEqual(run["stratum"], "outer=codex, inner=codex")
        self.assertEqual([(r["role"], r["manager"], r["executor"], r["model"], r["effort"]) for r in run["roles"]],
                         [("worker", "project-owner", "codex-app-server", "gpt-6-luna", "high"),
                          ("verifier", None, "codex-app-server", "gpt-6-luna", "high")])
        self.assertIn("Markitect roles: worker project-owner: codex-app-server",
                      (self.out / "report.md").read_text(encoding="utf-8"))

    def test_holdouts_rerun_cumulatively_and_list_failures_by_item_and_rule(self):
        s1, s2 = self.station("S1")["holdouts"], self.station("S2")["holdouts"]
        self.assertEqual((s1["passed"], s1["total"], s1["status"]), (2, 2, "pass"))  # scratch copy has no .git
        self.assertEqual((s2["passed"], s2["total"], s2["status"]), (2, 3, "fail"))
        self.assertEqual(s2["failures"], [{"id": "audit-line", "status": "FAIL", "item": "B02", "rule": "R1",
                                           "detail": "no audit line"}])
        self.assertEqual(s2["byRule"]["R1"], {"passed": 0, "total": 1, "errors": 0})
        self.assertEqual(self.report["totals"]["holdouts"], {"passed": 4, "total": 5, "errors": 0})
        self.assertEqual(self.station("S1")["groundTruthObligations"], 3)

    def test_public_checks_run_again(self):
        checks = self.station("S1")["publicChecks"]
        self.assertIsNotNone(checks["total"])
        self.assertEqual(checks["duringRun"], {"passed": 1, "total": 5})

    def test_diff_profile_by_category_from_the_bundles(self):
        diff = self.station("S1")["diff"]
        self.assertEqual(diff["status"], "ok")
        cats = diff["categories"]
        self.assertEqual({name: cats[name]["files"] for name in cats},
                         {"model": 1, "code": 1, "tests": 1, "docs": 1, "config/other": 0})
        self.assertEqual(cats["model"], {"files": 1, "added": 1, "deleted": 1})
        s2 = self.station("S2")["diff"]
        self.assertEqual((s2["files"], s2["base"]), (1, self.station("S1")["mainCommit"]))
        self.assertIn("+# wave 2", (self.out / "stations" / "S2" / "wave.diff").read_text(encoding="utf-8"))

    def test_reviews_agreement_and_inputs(self):
        s1 = self.station("S1")
        self.assertEqual(s1["reviewers"]["codex"]["status"], "ok")
        self.assertEqual(s1["reviewers"]["claude"]["findings"], 2)
        self.assertEqual((s1["agreement"]["both"], s1["agreement"]["codexOnly"], s1["agreement"]["claudeOnly"]),
                         (1, 1, 1))
        self.assertEqual(s1["escalations"]["codex"], {"needed": 0, "unneeded": 0})
        review_input = self.out / "stations" / "S1" / "review-input"
        prompt = (review_input / "prompt.md").read_text(encoding="utf-8")
        self.assertIn("- Released items: B01", prompt)
        self.assertIn('"obligations"', prompt)  # the wave's ground truth
        self.assertIn("S1 done; all checks pass.", prompt)
        self.assertIn("+# finish", prompt)
        self.assertNotIn("reference implementation", prompt)
        self.assertNotIn("audit-line", prompt)  # holdouts never reach a reviewer
        self.assertFalse((review_input / "holdout.py").exists())
        truth = json.loads((review_input / "ground-truth.json").read_text(encoding="utf-8"))
        self.assertEqual(truth["wave"]["station"], "S1")
        notes = {name: json.loads((self.out / "stations" / "S1" / "reviewers" / name / "review.json")
                                  .read_text(encoding="utf-8"))["notes"] for name in ("codex", "claude")}
        self.assertIn("home=auth.json ", notes["codex"])
        self.assertIn("token=present", notes["claude"])
        self.assertEqual(notes["codex"].split("prompt=")[1], notes["claude"].split("prompt=")[1])
        totals = self.report["totals"]["reviewers"]["claude"]
        self.assertEqual((totals["wavesReviewed"], totals["findings"], totals["obligations"]),
                         (2, 4, {"covered": 2, "total": 4}))

    def test_token_never_lands_in_any_output(self):
        for path in self.out.rglob("*"):
            if path.is_file():
                self.assertNotIn(TOKEN, path.read_text(encoding="utf-8", errors="replace"), path)

    def test_classification_and_product_findings(self):
        s1, s2 = self.station("S1"), self.station("S2")
        self.assertEqual((s1["classification"]["class"], s1["classification"]["failed"]), ("none", False))
        self.assertIn({"class": "product", "reason": "MCP call markitect/project_plan did not finish"},
                      s1["classification"]["causes"])
        self.assertEqual(s1["productFindings"], {"markitectMcpCalls": 3, "failedMcpCalls": 2,
                                                 "unfinishedMcpCalls": 1, "productCommands": 1})
        self.assertEqual(s2["classification"]["class"], "environment")
        self.assertIn("429", s2["classification"]["reason"])
        self.assertEqual(self.report["classification"]["class"], "environment")
        text = (self.out / "product-findings.md").read_text(encoding="utf-8")
        for expected in ("project_explore host_rejected x1", "The shared Host rejected this.", "scope_missing",
                         "Arguments do not match closed tool schema", "Did not finish: `project_plan`",
                         "error: plan rejected", "`setup-write` exited 1", "cost cap is approximate"):
            self.assertIn(expected, text)


class V1RunTests(unittest.TestCase):
    def test_v1_run_without_evaluation_files_or_reviewers(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp).resolve()
            os.chmod(root, 0o755)
            empty = root / "empty.gitconfig"
            empty.write_text("", encoding="utf-8")
            with mock.patch.dict(os.environ, {"GIT_CONFIG_GLOBAL": str(empty), "GIT_CONFIG_NOSYSTEM": "1"}):
                run = make_run(root, method="conventional", stations=1)
                evaluation = make_evaluation(root, case=False)
                with quiet():
                    report = evaluate.assess_run(run, evaluation, run / "assessment", reviewer_names=[])
            self.assertFalse(report["evaluation"]["groundTruth"])
            self.assertIsNone(report["evaluation"]["files"]["groundTruth"])
            self.assertIsNone(report["stations"][0]["holdouts"])
            self.assertEqual(report["stations"][0]["reviewers"], {})
            self.assertIsNone(report["totals"]["holdouts"])
            self.assertEqual(report["stations"][0]["diff"]["status"], "ok")
            self.assertEqual((report["run"]["stratum"], report["run"]["roles"]), ("outer=codex", None))
            self.assertIn("did not use Markitect",
                          (run / "assessment" / "product-findings.md").read_text(encoding="utf-8"))


class HoldoutVerdictTests(unittest.TestCase):
    def run_checks(self, statuses: list[str], timeout: float = 60) -> dict:
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp).resolve()
            os.chmod(root, 0o755)
            write(root / "candidate" / "app.py", "")
            checks = [{"id": f"c{n}", "status": status, "item": "B01", "rule": "R1", "detail": ""}
                      for n, status in enumerate(statuses)]
            checks.append({"id": "argv", "status": "PASS", "item": None, "rule": None, "detail": "ARGV"})
            write(root / "holdout.py", "import json, sys\n"
                  f"checks = {checks!r}\nchecks[-1]['detail'] = ' '.join(sys.argv[1:])\n"
                  "print(json.dumps({'station': 1, 'checks': checks}))\n")
            return evaluate.run_holdouts(root / "candidate", root / "holdout.py", 1, root / "out", timeout)

    def test_holdout_gets_a_deadline_before_the_hard_timeout(self):
        argv = next(c for c in self.run_checks([], timeout=600)["checks"] if c["id"] == "argv")["detail"]
        self.assertTrue(argv.endswith("--station 1 --deadline 510"), argv)

    def test_error_is_not_judged_and_never_a_candidate_failure(self):
        result = self.run_checks(["PASS", "ERROR"])  # plus the argv check, a PASS without item
        self.assertEqual((result["passed"], result["total"], result["errors"], result["status"]), (2, 2, 1, "error"))
        self.assertEqual(result["byItem"]["B01"], {"passed": 1, "total": 1, "errors": 1})
        self.assertIn("could not judge", result["error"])
        self.assertEqual(self.run_checks(["PASS", "FAIL", "ERROR"])["status"], "fail")
        self.assertEqual(self.run_checks(["PASS", "PASS"])["status"], "pass")
        self.assertEqual(evaluate._pair(result), "2/2 +1 not judged (error)")


class UnitTests(unittest.TestCase):
    def test_category(self):
        for path, expected in ((".markitect/model.yaml", "model"), ("tests/test_app.py", "tests"),
                               ("src/app_test.go", "tests"), ("test_cli.py", "tests"), ("README.md", "docs"),
                               ("docs/x.yaml", "docs"), ("app.py", "code"), ("checks/acceptance.py", "code"),
                               (".gitignore", "config/other"), ("pyproject.toml", "config/other"),
                               (".agents/skills/markitect-x/SKILL.md", "docs")):
            self.assertEqual(evaluate.category(path), expected, path)

    def test_parse_numstat(self):
        raw = b"3\t1\tapp.py\x000\t0\tdocs/a b.md\x00-\t-\tlogo.png\x00"
        entries = evaluate.parse_numstat(raw)
        self.assertEqual([e["path"] for e in entries], ["app.py", "docs/a b.md", "logo.png"])
        self.assertEqual((entries[0]["added"], entries[0]["deleted"]), (3, 1))
        self.assertIsNone(entries[2]["added"])

    def test_parse_reviewers(self):
        self.assertEqual(evaluate.parse_reviewers(None), ["codex", "claude"])
        self.assertEqual(evaluate.parse_reviewers("claude,codex"), ["codex", "claude"])
        self.assertEqual(evaluate.parse_reviewers("none"), [])
        for bad in ("gemini", "", "codex,codex"):
            with self.assertRaises(evaluate.AssessError):
                evaluate.parse_reviewers(bad)

    def test_claude_stream_events(self):
        with tempfile.TemporaryDirectory() as temp:
            folder = Path(temp)
            write(folder / "events.jsonl", jsonl([
                {"type": "assistant", "message": {"content": [
                    {"type": "tool_use", "id": "u1", "name": "mcp__markitect__project_check", "input": {}},
                    {"type": "tool_use", "id": "u2", "name": "Bash", "input": {"command": "markitect project plan"}},
                    {"type": "tool_use", "id": "u3", "name": "mcp__markitect__project_edit", "input": {"a": 1}}]}},
                {"type": "user", "message": {"content": [
                    {"type": "tool_result", "tool_use_id": "u1", "is_error": True,
                     "content": [{"type": "text", "text": json.dumps({"diagnostic": {"code": "host_rejected",
                                                                                   "message": "rejected"}})}]},
                    {"type": "tool_result", "tool_use_id": "u2", "is_error": True, "content": "exit 2: bad"}]}},
                {"type": "result", "is_error": True, "result": "markitect server crashed"}]))
            found = evaluate.analyze_events(folder)
        self.assertEqual(found["markitectMcpCalls"], 1)
        self.assertEqual((found["failedMcpCalls"][0]["code"], found["failedMcpCalls"][0]["message"]),
                         ("host_rejected", "rejected"))
        self.assertEqual([c["tool"] for c in found["unfinishedMcpCalls"]], ["project_edit"])
        self.assertEqual(found["productCommands"][0]["output"], "exit 2: bad")
        self.assertEqual(found["errorMessages"], ["markitect server crashed"])

    def test_classify_harness_beats_environment(self):
        with tempfile.TemporaryDirectory() as temp:
            folder = Path(temp)
            write(folder / "snapshot-error.txt", "boom")
            station = {"folder": folder, "record": None}
            analysis = {"errorMessages": ["rate limit reached"], "unfinishedMcpCalls": []}
            result = evaluate.classify_station(station, {"exitCode": 0, "sessionId": "s"}, analysis, "")
        self.assertEqual((result["class"], result["failed"]), ("harness", True))
        self.assertEqual([c["class"] for c in result["causes"]], ["harness", "harness", "environment"])

    def test_run_classification_prefers_the_runner_stop_category(self):
        with tempfile.TemporaryDirectory() as temp:
            results = Path(temp)
            first = evaluate.classify_run({"status": "completed"}, {"stopReason": "S2: login rejected by provider",
                                                                    "stopCategory": "environment"}, None, results, [])
            v1 = evaluate.classify_run({}, {"stopReason": "S1: agent never produced a session id (exit 1)"},
                                       None, results, [])
            blocked = evaluate.classify_run({}, {}, {"status": "blocked", "error": "init failed"}, results, [])
            outcome = evaluate.classify_run({}, {"stopReason": "total time used up after S3"}, None, results, [])
        self.assertEqual((first["class"], first["reason"]), ("environment", "S2: login rejected by provider"))
        self.assertEqual(v1["class"], "environment")
        self.assertEqual((blocked["class"], blocked["reason"]), ("product", "method setup blocked: init failed"))
        self.assertEqual((outcome["class"], outcome["reason"]), ("none", "stopped: total time used up after S3"))

    def test_timeout_without_cause_is_an_agent_outcome(self):
        with tempfile.TemporaryDirectory() as temp:
            station = {"folder": Path(temp), "record": {}}
            result = evaluate.classify_station(station, {"exitCode": None, "timedOut": True, "sessionId": "s"},
                                               {"errorMessages": [], "unfinishedMcpCalls": []}, "")
        self.assertEqual(result["class"], "none")
        self.assertIn("timed out", result["reason"])

    def test_run_report_stratum_and_roles_win(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp).resolve()
            os.chmod(root, 0o755)
            empty = root / "empty.gitconfig"
            empty.write_text("", encoding="utf-8")
            with mock.patch.dict(os.environ, {"GIT_CONFIG_GLOBAL": str(empty), "GIT_CONFIG_NOSYSTEM": "1"}):
                run = make_run(root, stations=1)
                path = run / "results" / "report.json"
                data = json.loads(path.read_text(encoding="utf-8"))
                roles = [{"role": "verifier", "manager": None, "executor": "x", "providerVersion": None,
                          "model": "m", "effort": "low"}]
                path.write_text(json.dumps({**data, "stratum": "outer=claude, inner=codex", "roles": roles}),
                                encoding="utf-8")
                with quiet():
                    report = evaluate.assess_run(run, make_evaluation(root, case=False), run / "assessment")
        self.assertEqual((report["run"]["stratum"], report["run"]["roles"]), ("outer=claude, inner=codex", roles))

    def test_stage_inputs_never_copies_reference(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            evaluation = make_evaluation(root)
            write(evaluation / "readinglog" / "validate.py", "")
            write(evaluation / "readinglog" / "mutants" / "m.json", "{}")
            (root / "playground").mkdir()
            write(root / "playground" / "x.py", "")
            with mock.patch.object(evaluate, "ROOT", root), mock.patch.object(evaluate, "EVALUATION", evaluation):
                evaluate.stage_inputs("readinglog", root / "staged")
            staged = root / "staged" / "evaluation"
            self.assertTrue((staged / "readinglog" / "holdout.py").is_file())
            self.assertTrue((staged / "readinglog" / "ground-truth.json").is_file())
            for name in ("reference", "mutants", "validate.py"):
                self.assertFalse((staged / "readinglog" / name).exists(), name)
            self.assertFalse((root / "staged" / "tests").exists())
            self.assertTrue((staged / "common" / "reviewer-prompt.md").is_file())
            self.assertTrue((root / "staged" / "playground" / "x.py").is_file())


class FakeDocker:
    """Stands in for subprocess.run/Popen in host; records every argv."""

    def __init__(self, exit_code: int = 0):
        self.calls, self.exit_code, self.container = [], exit_code, None

    def run(self, cmd, **kwargs):
        self.calls.append(list(cmd))
        code, out = 0, ""
        if cmd[:2] == ["docker", "version"]:
            out = "29.4.1"
        elif cmd[:3] == ["docker", "image", "inspect"]:
            out = "sha256:img"
        elif cmd[:3] == ["docker", "container", "inspect"]:
            code, out = (1, "") if self.container is None else (0, "false")
        elif cmd[:2] == ["docker", "run"]:
            out, self.container = "cid", "stopped"
        elif cmd[:2] == ["docker", "wait"]:
            out = f"{self.exit_code}\n"
        elif cmd[:2] == ["docker", "rm"]:
            self.container = None
        return subprocess.CompletedProcess(cmd, code, stdout=out, stderr="")

    def popen(self, cmd, **kwargs):
        self.calls.append(list(cmd))
        return mock.Mock(wait=lambda timeout=None: 0, poll=lambda: 0, kill=lambda: None)


class HostAssessTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name).resolve() / "runs, one"
        self.run_dir = self.root / "conv-x"
        manifest = {"id": "conv-x", "case": "readinglog", "method": "conventional",
                    "agent": {"kind": "codex"}, "container": {"cpus": 2, "memory": "4g", "pidsLimit": 512}}
        write(self.run_dir / "host.json", json.dumps({"manifest": manifest, "image": {"id": "sha256:img"}}))
        for number in (1, 2, 3):
            (self.run_dir / "results" / "stations" / f"S{number}").mkdir(parents=True)
        self.auth = self.root / "auth.json"
        write(self.auth, "LOGIN-CONTENT")
        self.token = self.root / "claude-token"
        write(self.token, TOKEN)

    def assess(self, docker: FakeDocker, *extra: str) -> int:
        argv = ["assess", "--run", str(self.run_dir), "--codex-auth", str(self.auth),
                "--claude-token", str(self.token), *extra]
        with mock.patch.object(host.subprocess, "run", docker.run), \
                mock.patch.object(host.subprocess, "Popen", docker.popen), \
                mock.patch.object(evaluate, "evaluation_identity", lambda: {"commit": "c0ffee", "dirty": True}), \
                quiet():
            return entry.main(argv)

    def test_launches_one_assessment_container(self):
        docker = FakeDocker(exit_code=0)
        self.assertEqual(self.assess(docker), 0)
        out = self.run_dir / "assessment"
        run = next(call for call in docker.calls if call[:2] == ["docker", "run"])
        self.assertEqual(run[run.index("--name") + 1], "mpg-assess-conv-x")
        self.assertIn("markitect-playground=1", run)
        self.assertIn("--init", run)
        for flag, value in (("--cpus", "2"), ("--memory", "4g"), ("--pids-limit", "512"),
                            ("--workdir", "/assess/in")):
            self.assertEqual(run[run.index(flag) + 1], value)
        mounts = [run[i + 1] for i, arg in enumerate(run) if arg == "--mount"]
        self.assertEqual(mounts, [
            host._mount(self.run_dir, "/assess/run", readonly=True),
            host._mount(out / "inputs", "/assess/in", readonly=True),
            host._mount(out, "/assess/out"),
            host._mount(self.auth, evaluate.SECRET_CODEX, readonly=True),
            host._mount(self.token, evaluate.SECRET_CLAUDE, readonly=True)])
        command = run[run.index("sha256:img") + 1:]
        self.assertEqual(command[:6], ["python3", "-B", "-m", "playground", "assess", "--inside"])
        for pair in (["--reviewers", "codex,claude"], ["--codex-auth", evaluate.SECRET_CODEX],
                     ["--claude-token", evaluate.SECRET_CLAUDE], ["--evaluation-commit", "c0ffee"],
                     ["--evaluation-dirty", "yes"]):
            self.assertIn(pair, [command[i:i + 2] for i in range(len(command) - 1)])
        self.assertTrue((out / "inputs" / "playground" / "evaluate.py").is_file())
        self.assertTrue((out / "inputs" / "evaluation" / "common" / "reviewer-prompt.md").is_file())
        self.assertFalse((out / "inputs" / "evaluation" / "readinglog2").exists())
        record = json.loads((out / "host.json").read_text(encoding="utf-8"))
        self.assertEqual((record["status"], record["containerExitCode"]), ("completed", 0))
        self.assertGreater(record["timeoutSeconds"], 3 * 2700)
        everything = json.dumps(docker.calls) + (out / "host.json").read_text(encoding="utf-8")
        self.assertNotIn(TOKEN, everything)
        self.assertNotIn("LOGIN-CONTENT", everything)
        self.assertIn(["docker", "rm", "-f", "mpg-assess-conv-x"], docker.calls)

    def test_assessment_is_handed_back_after_the_container_ends(self):
        docker = FakeDocker()
        with mock.patch.object(host, "hand_back", return_value="done") as hand_back:
            self.assertEqual(self.assess(docker), 0)
        out = self.run_dir / "assessment"
        hand_back.assert_called_once_with("mpg-assess-conv-x", "sha256:img", out)
        self.assertEqual(json.loads((out / "host.json").read_text(encoding="utf-8"))["handBack"], "done")

    def test_refuses_existing_assessment_unless_forced(self):
        (self.run_dir / "assessment").mkdir()
        write(self.run_dir / "assessment" / "old.txt", "old")
        docker = FakeDocker()
        self.assertEqual(self.assess(docker), 2)
        self.assertFalse(any(call[:2] == ["docker", "run"] for call in docker.calls))
        self.assertTrue((self.run_dir / "assessment" / "old.txt").exists())
        self.assertEqual(self.assess(docker, "--force"), 0)
        self.assertFalse((self.run_dir / "assessment" / "old.txt").exists())

    def test_reviewers_none_needs_no_credentials(self):
        docker = FakeDocker()
        argv = ["assess", "--run", str(self.run_dir), "--reviewers", "none", "--codex-auth", str(self.root / "no"),
                "--claude-token", str(self.root / "no")]
        with mock.patch.object(host.subprocess, "run", docker.run), \
                mock.patch.object(host.subprocess, "Popen", docker.popen), \
                mock.patch.object(evaluate, "evaluation_identity", lambda: {"commit": None, "dirty": None}), quiet():
            self.assertEqual(entry.main(argv), 0)
        run = next(call for call in docker.calls if call[:2] == ["docker", "run"])
        self.assertEqual(sum(arg == "--mount" for arg in run), 3)
        self.assertIn("none", run)

    def test_missing_token_is_refused_before_docker(self):
        docker = FakeDocker()
        self.token.unlink()
        self.assertEqual(self.assess(docker), 2)
        self.assertEqual(docker.calls, [])

    def test_fake_reviewers_get_throwaway_credentials_never_the_real_ones(self):
        docker = FakeDocker()
        argv = ["assess", "--run", str(self.run_dir), "--fake-reviewers"]
        with mock.patch.object(host.subprocess, "run", docker.run), \
                mock.patch.object(host.subprocess, "Popen", docker.popen), \
                mock.patch.object(evaluate, "DEFAULT_CLAUDE_TOKEN", self.root / "no-login"), \
                mock.patch.object(evaluate, "evaluation_identity", lambda: {"commit": None, "dirty": None}), quiet():
            self.assertEqual(entry.main(argv), 0)  # no real login needed
            self.assertEqual(entry.main([*argv, "--force", "--claude-token", str(self.token)]), 2)
        run = next(call for call in docker.calls if call[:2] == ["docker", "run"])
        mounts = [run[i + 1] for i, arg in enumerate(run) if arg == "--mount"]
        self.assertEqual(len(mounts), 5)
        self.assertFalse(any(str(self.auth) in m or str(self.token) in m for m in mounts))
        self.assertIn("--fake-reviewers", run)
        out = self.run_dir / "assessment"
        self.assertTrue((out / "inputs" / "tests" / "fake_reviewer.py").is_file())
        self.assertTrue(json.loads((out / "host.json").read_text(encoding="utf-8"))["fakeReviewers"])
        throwaway = Path(mounts[3].split("source=", 1)[1].split(",", 1)[0]).parent
        self.assertFalse(throwaway.exists())  # removed after the container ended

    def test_not_a_run_folder(self):
        with quiet():
            self.assertEqual(entry.main(["assess", "--run", str(self.root), "--reviewers", "none"]), 2)


if __name__ == "__main__":
    unittest.main()
