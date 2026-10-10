from __future__ import annotations

import json
from pathlib import Path
import tempfile
import unittest

from playground import report


def write(path: Path, value) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value), encoding="utf-8")


def events(commands=3):
    return {"sessionId": "thread-1", "items": commands + 1, "commands": commands, "fileChanges": 1,
            "mcpToolCalls": 2, "collabToolCalls": 0, "errors": 0, "malformedLines": 0}


def usage(outer, every, others=1):
    names = ("input", "cachedInput", "output")
    return {"outer": dict(zip(names, outer)), "all": dict(zip(names, every)), "otherSessions": others}


class ReportTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.out = Path(temp.name)

    def test_full_run_is_summarized_with_totals(self):
        manifest = {"id": "conv-roombook-001", "case": "roombook", "method": "conventional",
                    "agent": {"kind": "codex", "model": "m", "effort": "high", "maxSubagents": 2}}
        write(self.out / "runner.json", {
            "manifest": manifest, "status": "completed", "exitCode": 0, "stopReason": None,
            "wallSeconds": 100.5, "versions": {"codex": "0.162.0", "imageId": "sha256:abc",
                                               "markitectCommit": None, "markitectSha256": None}})
        write(self.out / "setup" / "setup.json", {
            "status": "ready", "seconds": 1.5, "commit": "a" * 40, "error": None, "leftoverProcessesKilled": 0,
            "steps": [{"name": "git-add", "exitCode": 0}, {"name": "git-diff-check", "exitCode": 2}],
            "notes": {"providerExecutable": "/usr/lib/codex/vendor/bin/codex"}})
        for number, tokens in ((1, (100, 50, 10)), (2, (200, 0, 20))):
            folder = self.out / "stations" / f"S{number}"
            write(folder / "agent.json", {"seconds": 10.0 * number, "exitCode": 0, "timedOut": False,
                                          "leftoverProcessesKilled": number - 1, "events": events(),
                                          "usage": usage(tokens, [t * 2 for t in tokens]),
                                          "sessionSwitched": number == 2,
                                          "newMainCommits": 2, "mainCommit": "c" * 40})
            write(folder / "checks.json", {"passed": number, "total": 5, "status": "fail"})
        write(self.out / "final" / "final.json", {
            "passed": 4, "total": 5, "status": "fail", "checks": {"findings": ["long"]},
            "ownTests": {"status": "pass", "ran": 7, "failures": 0, "errors": 0, "skipped": 1, "runs": []},
            "conformance": None})

        result = report.build(self.out)
        self.assertEqual(result["status"], "completed")
        self.assertEqual([s["station"] for s in result["stations"]], ["S1", "S2"])
        self.assertEqual(result["stations"][1]["checks"], {"passed": 2, "total": 5, "status": "fail"})
        self.assertEqual(result["totals"]["tokens"], {"input": 300, "cachedInput": 50, "output": 30})
        self.assertEqual(result["totals"]["tokensAllSessions"], {"input": 600, "cachedInput": 100, "output": 60})
        self.assertEqual(result["totals"]["agentSeconds"], 30.0)
        self.assertEqual(result["fairness"]["model"], "m")
        self.assertEqual(len(result["setup"]["notes"]), 2)
        self.assertEqual(result["final"]["ownTests"]["ran"], 7)
        self.assertNotIn("checks", result["final"])
        self.assertEqual(json.loads((self.out / "report.json").read_text(encoding="utf-8")), result)
        text = (self.out / "report.md").read_text(encoding="utf-8")
        self.assertIn("# Run conv-roombook-001", text)
        self.assertIn("| S2 | 20.0 | 0 | no | 1 | 200 / 0 / 20 | 400 / 0 / 40 (1) | 3 / 2 / 0 | 2 | 2/5 (fail) |",
                      text)
        self.assertIn("public checks 4/5 (fail); own tests pass (ran 7", text)
        self.assertIn("git-diff-check exited 2 (tolerated)", text)
        self.assertIn("Resume started a new session at S2", text)
        self.assertIn("Fairness", text)

    def test_unknown_values_stay_null_never_zero(self):
        folder = self.out / "stations" / "S1"
        write(folder / "agent.json", {"seconds": 5.0, "exitCode": None, "timedOut": True,
                                      "events": events(commands=0), "usage": usage((None,) * 3, (None,) * 3)})
        write(self.out / "stations" / "S2" / "agent.json", {"seconds": 3.0, "events": events(),
                                                            "usage": usage((10, 0, 1), (10, 0, 1), 0)})
        (folder / "snapshot-error.txt").write_text("boom\n", encoding="utf-8")
        result = report.build(self.out)
        self.assertIsNone(result["stations"][0]["tokens"]["input"])
        self.assertIsNone(result["stations"][0]["checks"]["passed"])
        self.assertIsNone(result["stations"][0]["newMainCommits"])
        self.assertEqual(result["totals"]["tokens"], {"input": None, "cachedInput": None, "output": None})
        self.assertIsNone(result["totals"]["wallSeconds"])
        self.assertEqual(result["totals"]["agentSeconds"], 8.0)
        text = (self.out / "report.md").read_text(encoding="utf-8")
        self.assertIn("| S1 | 5.0 | n/a | yes | n/a | n/a / n/a / n/a | n/a / n/a / n/a (1) | 0 / 2 / 0 | n/a | n/a |",
                      text)
        self.assertIn("S1: snapshot-error.txt", text)

    def test_classification_follows_what_ended_the_run(self):
        claude = {"id": "c", "case": "readinglog2", "method": "markitect",
                  "agent": {"kind": "claude", "model": "claude-opus-5-5", "effort": "high", "maxSubagents": 2}}
        write(self.out / "runner.json", {"manifest": claude, "status": "stopped", "exitCode": 1,
                                         "stopReason": "S1: agent exited 1 without doing any work (see stderr.log)",
                                         "stopCategory": "environment", "stationsPlanned": 6,
                                         "versions": {"codex": "0.162.0", "claude": "2.1.296"}})
        write(self.out / "stations" / "S1" / "agent.json", {"seconds": 1.0, "usage": usage((None,) * 3, (None,) * 3),
                                                            "events": {**events(), "mcpCallErrors": 2}})
        (self.out / "stations" / "S1" / "outside-error.txt").write_text("x\n", encoding="utf-8")
        result = report.build(self.out)
        classification = result["classification"]
        self.assertEqual((classification["class"], classification["reason"]),
                         ("environment", "S1: agent exited 1 without doing any work (see stderr.log)"))
        self.assertEqual([item["class"] for item in classification["signals"]], ["environment", "harness", "product"])
        self.assertEqual((result["fairness"]["outerProvider"], result["stratum"], result["fairness"]["claude"]),
                         ("claude", "outer=claude, inner=codex", "2.1.296"))
        self.assertNotIn("stratum", result["fairness"])  # it differs between the arms by design
        self.assertEqual(result["totals"]["stationsPlanned"], 6)
        text = (self.out / "report.md").read_text(encoding="utf-8")
        self.assertIn("Classification: **environment**", text)
        self.assertIn("product: S1: 2 MCP call(s) to the product could not run", text)
        self.assertIn("1 of 6 ran", text)
        self.assertIn("max subagents per session 2, recorded only); stratum outer=claude, inner=codex.", text)
        self.assertIn("Claude Code's session transcripts", text)

    def test_product_failures_and_completed_runs(self):
        write(self.out / "runner.json", {"status": "stopped", "stopReason": "method setup blocked: init failed",
                                         "stopCategory": "product"})
        write(self.out / "setup" / "setup.json", {"status": "blocked", "blockedBy": "product", "error": "init failed"})
        self.assertEqual(report.build(self.out)["classification"]["class"], "product")
        write(self.out / "runner.json", {"status": "completed", "stationsPlanned": 4})
        write(self.out / "setup" / "setup.json", {"status": "ready", "roles": [
            {"role": "worker", "manager": "project-owner", "executor": "codex-app-server",
             "providerVersion": "codex-cli 0.162.0", "command": "/x/codex", "model": "gpt-6-luna", "effort": "high"},
            {"role": "verifier", "manager": None, "executor": "codex-app-server",
             "providerVersion": "codex-cli 0.162.0", "command": "/x/codex", "model": "gpt-6-luna", "effort": "high"}],
            "notes": {"onboardProvider": "both", "claudeRouter": {"content": "@AGENTS.md\n", "added": True}}})
        write(self.out / "final" / "final.json", {"passed": 1, "total": 2, "status": "fail", "conformance": {
            "status": "fail", "error": "markitect project check did not finish", "errorSource": "product"}})
        result = report.build(self.out)
        self.assertEqual(result["classification"]["class"], "product")
        self.assertIn("did not finish", result["classification"]["reason"])
        self.assertEqual(result["roles"], result["setup"]["roles"])
        self.assertEqual(result["roles"][0], {"role": "worker", "manager": "project-owner",
                                              "executor": "codex-app-server", "providerVersion": "codex-cli 0.162.0",
                                              "model": "gpt-6-luna", "effort": "high"})
        text = (self.out / "report.md").read_text(encoding="utf-8")
        self.assertIn("worker project-owner: codex-app-server (codex-cli 0.162.0), model gpt-6-luna, effort high; "
                      "verifier: codex-app-server", text)
        self.assertIn("CLAUDE.md router (`@AGENTS.md`, the same for both methods): added.", text)
        write(self.out / "final" / "final.json", {"passed": 2, "total": 2, "status": "pass"})
        self.assertEqual(report.build(self.out)["classification"],
                         {"class": "none", "reason": "completed S1-S4", "signals": []})

    def test_idle_timeouts_mean_the_provider_never_answered(self):
        write(self.out / "runner.json", {"status": "completed", "stationsPlanned": 2})
        idle = {**events(commands=0), "items": 0, "apiRetries": 7}
        write(self.out / "stations" / "S1" / "agent.json", {"seconds": 120.0, "timedOut": True, "events": idle,
                                                            "usage": usage((None,) * 3, (None,) * 3)})
        write(self.out / "stations" / "S2" / "agent.json", {"seconds": 120.0, "timedOut": True, "events": events(),
                                                            "usage": usage((5, 0, 1), (5, 0, 1))})
        classification = report.build(self.out)["classification"]
        self.assertEqual((classification["class"], classification["reason"]),
                         ("environment", "S1: timed out without any agent activity (7 provider request retries)"))

    def test_empty_output_still_produces_a_report(self):
        (self.out / "runner-error.txt").write_text("boom\n", encoding="utf-8")
        result = report.build(self.out)
        self.assertEqual(result["stations"], [])
        self.assertIsNone(result["final"])
        self.assertTrue(result["runnerError"])
        self.assertIsNone(result["totals"]["agentSeconds"])
        self.assertEqual((result["classification"]["class"], result["classification"]["reason"]),
                         ("harness", "runner error: boom"))
        text = (self.out / "report.md").read_text(encoding="utf-8")
        self.assertIn("no station ran", text)
        self.assertIn("runner-error.txt", text)


if __name__ == "__main__":
    unittest.main()
