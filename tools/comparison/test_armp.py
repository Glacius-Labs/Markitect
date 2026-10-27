import importlib.util
import io
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile
import time
import unittest
from contextlib import redirect_stdout
from unittest import mock

HERE = pathlib.Path(__file__).resolve().parent


def load(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


armp = load("armp_under_test", HERE / "armp.py")
mkp = armp.mkp
analyze = load("analyze_under_test", HERE / "analyze.py")
NODE = shutil.which("node")


def run_git(repo, *args):
    return subprocess.run(["git", "-c", "core.autocrlf=false", "-C", str(repo), *args], check=True,
                          capture_output=True, text=True).stdout.strip()


def library_repo(root):
    repo = pathlib.Path(root) / "input"
    subprocess.run(["git", "init", "-q", "-b", "dev/library-m1", str(repo)], check=True, capture_output=True)
    policy = repo / ".markitect" / "model" / "lending" / "loan-policy.yaml"
    policy.parent.mkdir(parents=True)
    policy.write_bytes(b"spec:\n  maxOpenLoans: 5\n  loanPeriodDays: 14\n")
    (repo / ".markitect" / "model.yaml").write_bytes(b"apiVersion: markitect.model/v1\n")
    run_git(repo, "add", "--all")
    run_git(repo, "-c", "user.name=f", "-c", "user.email=f@x.invalid", "commit", "-q", "-m", "M1")
    return repo


class Inputs(unittest.TestCase):
    def test_copy_keeps_the_commit_and_m2_changes_exactly_the_policy(self):
        with tempfile.TemporaryDirectory() as tmp:
            repo = library_repo(tmp)
            head = run_git(repo, "rev-parse", "HEAD")
            copied = armp.input_copy(repo, "HEAD", pathlib.Path(tmp) / "s1.bundle")
            self.assertEqual((copied["commit"], copied["branch"]), (head, "dev/library-m1"))
            m2 = armp.input_m2(pathlib.Path(tmp) / "s1.bundle", head, pathlib.Path(tmp) / "s2.bundle")
            self.assertEqual((m2["from"], m2["branch"]), (head, "dev/library-m2"))
            check = pathlib.Path(tmp) / "check"
            subprocess.run(["git", "clone", "-q", str(pathlib.Path(tmp) / "s2.bundle"), str(check), "-b", "dev/library-m2"],
                           check=True, capture_output=True)
            self.assertEqual(run_git(check, "rev-parse", "HEAD~1"), head)
            self.assertIn(b"maxOpenLoans: 3", (check / ".markitect/model/lending/loan-policy.yaml").read_bytes())
            self.assertEqual(run_git(check, "log", "-1", "--format=%s %an"), "Library M2 model Markitect Library Fixture")
            with self.assertRaises(armp.RunError):
                armp.input_m2(pathlib.Path(tmp) / "s2.bundle", m2["commit"], pathlib.Path(tmp) / "again.bundle")


class Login(unittest.TestCase):
    def test_copy_drops_refresh_and_refuses_a_login_that_expires_before_the_deadline(self):
        later = int((time.time() + 8 * 3600) * 1000)
        raw = json.dumps({"claudeAiOauth": {"accessToken": "acc-secret", "refreshToken": "ref-secret", "expiresAt": later,
                                            "refreshTokenExpiresAt": later, "scopes": ["a"]}, "mcpOAuth": {"x": {"accessToken": "m"}}})
        copy, secrets = armp.session_login(raw.encode(), 2700, 600)
        self.assertEqual(json.loads(copy), {"claudeAiOauth": {"accessToken": "acc-secret", "expiresAt": later, "scopes": ["a"]}})
        self.assertEqual(secrets, [b"acc-secret", b"ref-secret"])
        with self.assertRaises(armp.RunError):
            armp.session_login(raw.encode(), 8 * 3600, 600)
        with self.assertRaises(armp.RunError):
            armp.session_login(b'{"mcpOAuth": {}}', 10, 0)


class Records(unittest.TestCase):
    def test_usage_counts_each_assistant_message_once_over_main_and_agent_transcripts(self):
        with tempfile.TemporaryDirectory() as tmp:
            agent = pathlib.Path(tmp) / "p" / "s" / "subagents" / "workflows" / "wf" / "agent-1.jsonl"
            agent.parent.mkdir(parents=True)
            entry = {"type": "assistant", "requestId": "r1", "message": {"id": "m1", "model": "claude-sonnet-5-5",
                                                                         "usage": {"input_tokens": 10, "output_tokens": 5}}}
            agent.write_text(json.dumps(entry) + "\n" + json.dumps(entry) + "\nnot json\n", encoding="utf-8")
            (pathlib.Path(tmp) / "p" / "s.jsonl").write_text(json.dumps(dict(entry, requestId="r2")) + "\n", encoding="utf-8")
            u = armp.usage(tmp)
            self.assertEqual((u["messages"], u["agent_transcripts"], u["total"]["input_tokens"], u["total"]["output_tokens"]), (2, 1, 20, 10))

    def test_stream_summary_reads_result_events(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = pathlib.Path(tmp) / "s.jsonl"
            path.write_text('{"type":"system"}\n{"type":"result","subtype":"success","is_error":false,"result":"{}"}\n',
                            encoding="utf-8")
            s = armp.stream_summary(path)
            self.assertEqual((s["events"], len(s["results"]), s["errors"]), (2, 1, []))


FAKE_CLAUDE = r'''
import json, os, pathlib, sys
state = pathlib.Path(os.environ["MARKITECT_STATE_DIR"]); state.mkdir(parents=True, exist_ok=True)
login = json.loads(pathlib.Path(os.environ["CLAUDE_CONFIG_DIR"], ".credentials.json").read_text())
seen = {k: v for k, v in os.environ.items() if k.startswith(("MARKITECT_", "GIT_", "HOME"))}
(state / "seen.json").write_text(json.dumps({"argv": sys.argv[1:], "env": seen, "login": sorted(login["claudeAiOauth"]), "cwd": os.getcwd()}))
project = pathlib.Path(os.environ["CLAUDE_CONFIG_DIR"], "projects", "p"); project.mkdir(parents=True)
(project / "s.jsonl").write_text(json.dumps({"type": "assistant", "requestId": "r", "message": {"id": "m", "usage": {"input_tokens": 3, "output_tokens": 4}}}) + "\n")
print(json.dumps({"type": "result", "subtype": "success", "is_error": False, "result": "{}"}))
'''


class Run(unittest.TestCase):
    def test_a_session_without_publication_is_recorded_completed_but_not_a_technical_pass(self):
        with tempfile.TemporaryDirectory() as tmp:
            tmp = pathlib.Path(tmp)
            repo = library_repo(tmp)
            head = run_git(repo, "rev-parse", "HEAD")
            armp.input_copy(repo, "HEAD", tmp / "in.bundle")
            fake = tmp / "fake_claude.py"
            fake.write_text(FAKE_CLAUDE)
            later = int((time.time() + 8 * 3600) * 1000)
            credentials = tmp / "credentials.json"
            credentials.write_text(json.dumps({"claudeAiOauth": {"accessToken": "acc-secret-1", "refreshToken": "ref-secret-1", "expiresAt": later}}))
            args = armp.main.__globals__["argparse"].Namespace(
                input=str(tmp / "in.bundle"), commit=head, out=str(tmp / "out"), label="p-test-1", markitect=str(fake),
                work_root=str(tmp / "work"), model="sonnet", effort="high", deadline_seconds=120, max_agents=7, lock=str(tmp / "lock"),
                claude="claude", credentials=str(credentials), branch=None)
            with mock.patch.object(armp, "claude_argv", lambda *_: [sys.executable, str(fake), "/markitect:apply"]), \
                    redirect_stdout(io.StringIO()):
                self.assertEqual(armp.run(args), 1)
            out = tmp / "out"
            result = json.loads((out / "result.json").read_text())
            self.assertEqual((result["status"], result["technical_pass"], result["error"], result["residue"]), ("completed", False, None, []))
            self.assertEqual((result["input_commit"], result["owner_branch"], result["publication"]), (head, "dev/library-m1", None))
            self.assertEqual(result["usage"]["total"]["output_tokens"], 4)
            self.assertEqual(result["login_sha256_before"], result["login_sha256_after"])
            self.assertEqual(result["secret_in_outputs"], [])
            seen = json.loads((out / "state" / "seen.json").read_text())
            self.assertEqual(seen["login"], ["accessToken", "expiresAt"])
            self.assertEqual((seen["env"]["MARKITECT_RUN_ID"], seen["env"]["MARKITECT_OWNER_BRANCH"], seen["env"]["MARKITECT_MAX_AGENTS"]),
                             ("p-test-1", "dev/library-m1", "7"))
            self.assertEqual(seen["env"]["GIT_AUTHOR_NAME"], "Markitect Manager")
            self.assertEqual(seen["argv"], ["/markitect:apply"])
            self.assertEqual((pathlib.Path(seen["env"]["HOME"]).name, seen["env"]["GIT_CONFIG_NOSYSTEM"], seen["env"]["GIT_CONFIG_VALUE_1"]),
                             ("home", "1", ""))
            self.assertTrue((out / "start.bundle").exists() and (out / "work.bundle").exists() and not (out / "result.bundle").exists())
            self.assertFalse((tmp / "work" / "p-test-1").exists())
            self.assertFalse((tmp / "lock").exists())
            self.assertIn("result.json", (out / "files.sha256").read_text())


def verifier_record(package, round_, head, decision, mandate="Verify"):
    return {"agentType": "markitect:verifier", "head": head, "answer": {"decision": decision},
            "header": {"mandate": mandate, "round": round_, "run": "r1", "package": package}}


class Host(unittest.TestCase):
    def guard(self, command, owner="dev/library-m1"):
        with mock.patch.dict(os.environ, {"MARKITECT_OWNER_BRANCH": owner}), \
                mock.patch.object(sys, "stdin", io.StringIO(json.dumps({"tool_input": {"command": command}}))), \
                mock.patch.object(sys, "stderr", io.StringIO()):
            return mkp.cmd_guard(None)

    def test_guard_judges_the_git_subcommand_not_message_or_path_text(self):
        for refused in ["git push origin x", "git -C ../w push", "cd x && git push", "git notes --ref=markitect-check add -f -m x HEAD",
                        "git update-ref refs/heads/a b", "git worktree remove ../w", "git branch -D mk/r/i1/a",
                        "git branch -f markitect/apply/r x", "git checkout dev/library-m1", "git reset --hard refs/heads/dev/library-m1",
                        "markitect.exe apply --repo .", "C:/bin/markitect apply"]:
            self.assertEqual(self.guard(refused), 2, refused)
        for allowed in ["git checkout -B mk/r/i1/catalog/w3 abc", "git merge --no-ff abc", "git notes --ref=markitect-check show abc",
                        "git diff abc...def", "go test ./...", "git checkout --detach abc",
                        'git commit -m "add push notification handler"', 'git commit -m "markitect apply rules for loans"',
                        "cat .markitect/model/lending/apply-rules.yaml", 'git commit -m "update release notes: add loans"',
                        "git log --grep=push", "echo dev/library-m1", "git log dev/library-m1..HEAD"]:
            self.assertEqual(self.guard(allowed), 0, allowed)

    def test_tree_nesting_and_protected_paths_including_renames(self):
        tree = mkp.nest([{"package": ".", "children": ["a"], "areas": ["./"]},
                         {"package": "a", "children": ["a/b"], "areas": ["x/"]}, {"package": "a/b", "children": [], "areas": []}])
        self.assertEqual(tree["children"][0]["children"][0]["pkg"], "a/b")
        self.assertEqual(mkp.packages(tree), [".", "a", "a/b"])
        self.assertEqual(mkp.slug("."), "_root")
        self.assertEqual(mkp.slug("lending/reminders"), "lending--reminders")
        with tempfile.TemporaryDirectory() as tmp:
            repo = library_repo(tmp)
            base = run_git(repo, "rev-parse", "HEAD")
            (repo / ".markitect" / "registry").mkdir()
            (repo / ".markitect" / "registry" / "registry.yaml").write_text("x\n")
            run_git(repo, "mv", ".markitect/model/lending/loan-policy.yaml", "realization-policy.yaml")
            run_git(repo, "add", "--all")
            run_git(repo, "-c", "user.name=f", "-c", "user.email=f@x.invalid", "commit", "-q", "-m", "c")
            self.assertEqual(mkp.changed_protected(repo, base, "HEAD"), [".markitect/model/lending/loan-policy.yaml"])

    def test_the_record_hook_reads_header_and_answer_from_the_agents_own_transcript(self):
        with tempfile.TemporaryDirectory() as tmp:
            repo = library_repo(tmp)
            head = run_git(repo, "rev-parse", "HEAD")
            transcript = pathlib.Path(tmp) / "agent-a1.jsonl"
            prompt = 'Markitect Verify round 2, run r1. You are the Manager of lending/reminders (Package "lending/reminders").\nMore.'
            transcript.write_text("\n".join(json.dumps(e) for e in [
                {"type": "user", "message": {"role": "user", "content": prompt}},
                {"type": "assistant", "message": {"content": [{"type": "text", "text": "checking"}]}},
                {"type": "assistant", "message": {"content": [{"type": "tool_use", "name": "StructuredOutput",
                                                               "input": {"decision": "approve", "findings": [], "summary": "ok"}}]}},
            ]) + "\n", encoding="utf-8")
            state = pathlib.Path(tmp) / "state"
            hook = {"agent_type": "markitect:verifier", "agent_id": "a1", "cwd": str(repo), "agent_transcript_path": str(transcript)}
            with mock.patch.dict(os.environ, {"MARKITECT_STATE_DIR": str(state), "MARKITECT_RUN_ID": "r1"}), \
                    mock.patch.object(sys, "stdin", io.StringIO(json.dumps(hook))):
                self.assertEqual(mkp.cmd_record(None), 0)
            record = json.loads((state / "r1" / "agents" / "a1.json").read_text())
            self.assertEqual(record["header"], {"mandate": "Verify", "round": 2, "run": "r1", "package": "lending/reminders"})
            self.assertEqual((record["head"], record["answer"]["decision"]), (head, "approve"))
            with mock.patch.dict(os.environ, {"MARKITECT_STATE_DIR": str(state), "MARKITECT_RUN_ID": "r1"}), \
                    mock.patch.object(sys, "stdin", io.StringIO(json.dumps(dict(hook, agent_type="general-purpose", agent_id="a2")))):
                self.assertEqual(mkp.cmd_record(None), 0)
            self.assertFalse((state / "r1" / "agents" / "a2.json").exists())

    def test_only_a_round_in_which_every_package_approved_the_same_commit_counts(self):
        everyone = [".", "a", "a/b"]
        full = [verifier_record(p, 1, "c1", "approve") for p in everyone]
        self.assertEqual(mkp.verified_candidate(full, everyone), (1, "c1"))
        self.assertEqual(mkp.verified_candidate(full[:2], everyone), (None, None))
        self.assertEqual(mkp.verified_candidate(full + [verifier_record("a/b", 1, "c1", "reject")], everyone), (None, None))
        self.assertEqual(mkp.verified_candidate(full[:2] + [verifier_record("a/b", 1, "base", "approve")], everyone), (None, None))
        self.assertEqual(mkp.verified_candidate([verifier_record(p, 0, "c1", "approve", "Assessment") for p in everyone], everyone), (None, None))
        later = [verifier_record(p, 2, "c2", "approve") for p in everyone]
        self.assertEqual(mkp.verified_candidate(full + later, everyone), (2, "c2"))

    def test_publish_needs_the_hosts_records_and_a_root_result_of_this_run(self):
        with tempfile.TemporaryDirectory() as tmp:
            repo = library_repo(tmp)
            base = run_git(repo, "rev-parse", "HEAD")
            state = pathlib.Path(tmp) / "state"
            (state / "r1" / "agents").mkdir(parents=True)
            tree = {"pkg": ".", "areas": ["./"], "children": [{"pkg": "a", "areas": ["a/"], "children": []}]}
            (state / "r1" / "prepare.json").write_text(json.dumps({"base": base, "branch": "dev/library-m1", "baseTree": "t", "tree": tree}))
            result = pathlib.Path(tmp) / "wr.json"
            result.write_text(json.dumps({"outcome": "approved", "candidate": base}))
            publish = lambda: mkp.cmd_publish(mock.Mock(run_id="r1", result=str(result)))
            cwd = os.getcwd()
            try:
                os.chdir(repo)
                with mock.patch.dict(os.environ, {"MARKITECT_STATE_DIR": str(state)}), redirect_stdout(io.StringIO()):
                    self.assertEqual(publish(), 1)
                    record = json.loads((state / "r1" / "publication.json").read_text())
                    self.assertEqual(record["reason"], "no Verify round in which every Package approved one candidate")
                    for i, p in enumerate(("." , "a")):
                        (state / "r1" / "agents" / f"v{i}.json").write_text(json.dumps(verifier_record(p, 1, base, "approve")))
                    self.assertEqual(publish(), 1)
                    record = json.loads((state / "r1" / "publication.json").read_text())
                    self.assertEqual(record["candidate"], base)
                    self.assertFalse(record["guards"]["candidateIsRootResult"])
            finally:
                os.chdir(cwd)
            self.assertEqual(run_git(repo, "branch", "--list", "markitect/apply/*"), "")


TREE = {"pkg": ".", "areas": ["./"], "children": [
    {"pkg": "catalog", "areas": ["realization/catalog/"], "children": []},
    {"pkg": "lending", "areas": ["realization/lending/"], "children": [
        {"pkg": "lending/reminders", "areas": ["realization/lending/reminders/"], "children": []}]}]}
PACKAGES = [".", "catalog", "lending", "lending/reminders"]
APPROVE = {"decision": "approve", "findings": [], "summary": "ok"}
SUBMIT = {"status": "submitted", "commit": "NEW", "account": "done", "briefs": []}
INTEGRATE = {"status": "submitted", "accepted": "CHILDREN", "returned": [], "commit": "NEW", "account": "merged"}


@unittest.skipUnless(NODE, "node is not installed")
class Workflow(unittest.TestCase):
    def simulate(self, rules, **limits):
        scenario = {"args": {"runId": "r1", "base": "b" * 40, "mkp": "C:/x/mkp.py", "tree": TREE, **limits}, "rules": rules}
        with tempfile.TemporaryDirectory() as tmp:
            path = pathlib.Path(tmp) / "scenario.json"
            path.write_text(json.dumps(scenario), encoding="utf-8")
            out = subprocess.run([NODE, str(HERE / "claude-plugin" / "simulate.js"), str(path)], check=True, capture_output=True, text=True)
        return json.loads(out.stdout)

    def labels(self, run, prefix):
        return [c["label"] for c in run["calls"] if c["label"].startswith(prefix)]

    def base_rules(self):
        return [{"label": "assessment:", "answer": APPROVE}, {"label": "work:", "answer": SUBMIT},
                {"label": "integrate:", "answer": INTEGRATE}, {"label": "verify:", "answer": APPROVE}]

    def test_every_package_takes_part_in_every_mandate_children_first(self):
        run = self.simulate(self.base_rules())
        self.assertEqual(run["result"]["outcome"], "approved")
        for mandate in ("assessment:", "work:", "verify:"):
            self.assertEqual(sorted(l.split(":")[1] for l in self.labels(run, mandate)), sorted(PACKAGES), mandate)
        self.assertEqual(sorted(l.split(":")[1] for l in self.labels(run, "integrate:")), [".", "lending"])
        verify = self.labels(run, "verify:")
        self.assertLess(verify.index("verify:lending/reminders:1"), verify.index("verify:lending:1"))
        self.assertEqual(verify[-1], "verify:.:1")
        self.assertTrue(all(c["isolation"] == "worktree" for c in run["calls"]))
        self.assertTrue(all(c["agentType"] == "markitect:verifier" for c in run["calls"] if c["label"].startswith(("assessment", "verify"))))
        self.assertEqual(run["result"]["candidate"], run["result"]["rounds"][0]["implement"]["commit"])

    def test_a_child_reject_stays_in_force_and_starts_a_fresh_implement_with_bound_findings(self):
        reject = {"decision": "reject", "findings": [{"definition": "lending/borrow", "text": "limit ignored"}, {"text": "style"}], "summary": "no"}
        rules = [{"label": "verify:lending/reminders", "times": 1, "answer": reject}] + self.base_rules()
        run = self.simulate(rules)
        self.assertEqual(run["result"]["outcome"], "approved")
        self.assertEqual(len(run["result"]["rounds"]), 2)
        self.assertEqual(run["result"]["rounds"][0]["verify"]["decision"], "reject")
        self.assertEqual(len(self.labels(run, "work:.:i2")), 1)

    def test_a_question_climbs_until_an_ancestor_answers(self):
        question = {"status": "question", "questions": ["which clock?"], "account": "asking"}
        rules = [{"label": "work:lending/reminders:i1", "times": 1, "answer": question},
                 {"label": "answer:lending:", "answer": {"escalate": True}},
                 {"label": "answer:.:", "answer": {"escalate": False, "answer": "the injected clock"}}] + self.base_rules()
        run = self.simulate(rules)
        self.assertEqual(run["result"]["outcome"], "approved")
        self.assertEqual(self.labels(run, "answer:"), ["answer:lending:lending/reminders:1.0", "answer:.:lending/reminders:1.0"])
        self.assertEqual(len(self.labels(run, "work:lending/reminders:i1")), 2)

    def test_an_unanswered_question_is_a_model_blocker(self):
        rules = [{"label": "work:catalog", "answer": {"status": "question", "questions": ["?"], "account": "a"}},
                 {"label": "answer:", "answer": {"escalate": True}}] + self.base_rules()
        run = self.simulate(rules)
        self.assertEqual((run["result"]["outcome"], run["result"]["at"]), ("model_blocker", "Implement"))
        self.assertEqual(self.labels(run, "verify:"), [])

    def test_a_returned_child_is_corrected_then_integrated(self):
        returned = {"status": "submitted", "accepted": ["lending"], "returned": [{"pkg": "catalog", "findings": [{"text": "fix"}]}],
                    "commit": "NEW", "account": "one returned"}
        rules = [{"label": "integrate:.:i1:r0", "answer": returned}] + self.base_rules()
        run = self.simulate(rules)
        self.assertEqual(run["result"]["outcome"], "approved")
        self.assertEqual(len(self.labels(run, "work:catalog:i1")), 2)
        self.assertEqual(self.labels(run, "integrate:.:i1"), ["integrate:.:i1:r0", "integrate:.:i1:r1"])

    def test_an_assessment_blocker_ends_before_implement(self):
        blocker = {"decision": "model_blocker", "findings": [{"text": "contradiction"}], "summary": "blocked"}
        run = self.simulate([{"label": "assessment:catalog", "answer": blocker}] + self.base_rules())
        self.assertEqual((run["result"]["outcome"], run["result"]["at"]), ("model_blocker", "Assessment"))
        self.assertEqual(self.labels(run, "work:"), [])

    def test_a_child_the_review_left_undecided_is_reviewed_again(self):
        partial = {"status": "submitted", "accepted": ["lending"], "returned": [], "commit": "NEW", "account": "forgot catalog"}
        run = self.simulate([{"label": "integrate:.:i1:r0", "answer": partial}] + self.base_rules())
        self.assertEqual(run["result"]["outcome"], "approved")
        self.assertEqual(self.labels(run, "integrate:.:i1"), ["integrate:.:i1:r0", "integrate:.:i1:r1"])
        self.assertEqual(len(self.labels(run, "work:catalog:i1")), 1)

    def test_every_agent_gets_its_own_branch(self):
        run = self.simulate(self.base_rules())
        branches = [c["label"].rsplit(":", 1)[1] for c in run["calls"] if c["label"].startswith("work:")]
        self.assertEqual(len(branches), len(set(branches)))

    def test_an_error_outside_parallel_still_ends_with_its_reason(self):
        run = self.simulate(self.base_rules(), maxAgents=3)
        self.assertEqual((run["result"]["outcome"], run["result"]["allowance"]), ("incomplete", "agent allowance 3 used up"))
        self.assertIn("agent allowance 3 used up", run["result"]["error"])

    def test_the_agent_allowance_is_finite(self):
        run = self.simulate(self.base_rules(), maxAgents=5)
        self.assertEqual((run["result"]["outcome"], run["result"]["allowance"]), ("incomplete", "agent allowance 5 used up"))
        self.assertEqual(len(run["calls"]), 5)


class Analysis(unittest.TestCase):
    def test_any_two_arms_are_read_in_the_m_and_b_slots(self):
        row = {"chain": 1, "scenario": "S4", "counts": True, **{c: True for c in analyze.CRITERIA}}
        rows = [dict(row, arm="P"), dict(row, arm="B"), dict(row, arm="M", C1=False)]
        paired = analyze.pair(rows, "P", "B")
        self.assertEqual([r["arm"] for r in paired], ["M", "B"])
        self.assertEqual(analyze.read(paired, "A")["units"]["M"][:2], [1, 1])
        with tempfile.TemporaryDirectory() as tmp:
            ledger = pathlib.Path(tmp) / "ledger.json"
            ledger.write_text(json.dumps(rows))
            with redirect_stdout(io.StringIO()) as out:
                analyze.main([str(ledger), "--arms", "P,M"])
            result = json.loads(out.getvalue())
            self.assertEqual(result["arms"], {"M": "P", "B": "M"})
            with redirect_stdout(io.StringIO()) as out:
                analyze.main([str(ledger)])
            self.assertNotIn("arms", json.loads(out.getvalue()))


if __name__ == "__main__":
    unittest.main()
