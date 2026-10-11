"""End-to-end trajectory on the host with the fake agent (no Docker, no provider)."""

from __future__ import annotations

import contextlib
import io
import json
import os
from pathlib import Path
import shutil
import tempfile
import unittest
from unittest.mock import patch

from playground import __main__ as entry
from playground import codex_agent, lifecycle, manifest, runner

PLAYGROUND = Path(__file__).resolve().parents[1]


def make_manifest(**overrides) -> dict:
    data = {
        "schema": 1, "id": "fake-roombook-test", "case": "roombook", "method": "conventional",
        "agent": {"kind": "fake", "codexVersion": "0.162.0", "model": "fake-model-1", "effort": "high",
                  "maxSubagents": 2},
        "limits": {"stationSeconds": 300, "totalSeconds": 1200},
    }
    data.update(overrides)
    return manifest.validate(data)


def read_json(path: Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8"))


def claude_agent_cfg(kind: str = "fake-claude") -> dict:
    return {"kind": kind, "codexVersion": "0.162.0", "claudeVersion": "2.1.296", "model": "fake-model-1",
            "effort": "high", "maxSubagents": 2}


def files_containing(root: Path, needle: str) -> list[str]:
    found = []
    for path in root.rglob("*"):
        if path.is_file() and not path.is_symlink() and needle.encode("utf-8") in path.read_bytes():
            found.append(path.relative_to(root).as_posix())
    return found


class RunnerTestCase(unittest.TestCase):
    def setUp(self):
        if codex_agent.container_mode():
            try:
                import pwd
                pwd.getpwnam(codex_agent.AGENT_USER)
            except KeyError:
                self.skipTest("root without an 'agent' user")
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name).resolve()
        os.chmod(self.root, 0o755)  # the agent user must reach its repo in container mode
        self.out = self.root / "out"
        self.work = self.root / "work"
        self.home = self.root / "home"
        # Hermetic Git and no orphan from the fake agent outside the container.
        empty = self.root / "empty.gitconfig"
        empty.write_text("", encoding="utf-8")
        env = patch.dict(os.environ, {"GIT_CONFIG_GLOBAL": str(empty), "GIT_CONFIG_NOSYSTEM": "1",
                                      "FAKE_AGENT_NO_ORPHAN": "1"})
        env.start()
        self.addCleanup(env.stop)
        quiet = contextlib.redirect_stdout(io.StringIO())
        quiet.__enter__()
        self.addCleanup(quiet.__exit__, None, None, None)

    def run_trajectory(self, data: dict, in_dir: Path = PLAYGROUND, token_src: Path | None = None) -> int:
        with patch.object(runner, "_claude_version", return_value=None):  # never the host's own CLI
            return runner.run(data, self.out, in_dir=in_dir, work_root=self.work, agent_home=self.home,
                              auth_src=self.root / "no-auth.json",
                              token_src=token_src or self.root / "no-token")

    def six_station_seed(self) -> Path:
        """A copy of the real cases with an extra six-station case (roombook files, own plan)."""
        in_dir = self.root / "in"
        shutil.copytree(PLAYGROUND / "cases", in_dir / "cases", ignore=shutil.ignore_patterns("readinglog2"))
        shutil.copytree(PLAYGROUND / "cases" / "roombook", in_dir / "cases" / "readinglog2")
        checks = in_dir / "cases" / "readinglog2" / "checks"
        (checks / "roombook.py").rename(checks / "readinglog2.py")
        shutil.copytree(PLAYGROUND / "methods" / "conventional", in_dir / "methods" / "conventional")
        ids = iter(f"X{i:02d}" for i in range(1, 30))
        plan = {"schema": 1, "case": "readinglog2", "stations": [
            {"id": f"S{n}", "items": [next(ids) for _ in range(size)]} for n, size in enumerate((1, 3, 6, 2, 2, 1), 1)]}
        (in_dir / "cases" / "readinglog2" / "STATIONS.json").write_text(json.dumps(plan), encoding="utf-8")
        return in_dir


class FakeTrajectoryTests(RunnerTestCase):
    def test_four_stations_with_fake_agent(self):
        code = self.run_trajectory(make_manifest())
        error = self.out / "runner-error.txt"
        self.assertEqual(code, 0, error.read_text(encoding="utf-8") if error.exists() else "")
        state = read_json(self.out / "runner.json")
        self.assertEqual((state["status"], state["exitCode"], state["stationsRun"]), ("completed", 0, 4))
        self.assertIsNone(state["stopReason"])
        self.assertEqual(read_json(self.out / "setup" / "setup.json")["status"], "ready")
        self.assertTrue((self.home / ".codex" / "config.toml").is_file())
        self.assertFalse((self.home / ".codex" / "auth.json").exists())

        session = None
        for number in range(1, 5):
            folder = self.out / "stations" / f"S{number}"
            for name in ("events.jsonl", "stderr.log", "last-message.txt", "checks.json", "agent.json"):
                self.assertTrue((folder / name).is_file(), f"S{number}/{name}")
            agent = read_json(folder / "agent.json")
            self.assertEqual(agent["exitCode"], 0, (folder / "stderr.log").read_text(encoding="utf-8"))
            self.assertFalse(agent["timedOut"])
            self.assertEqual(agent["newMainCommits"], 2)  # feature commit + merge commit
            self.assertEqual(agent["snapshot"], f"audit/snapshot-S{number}")
            # Codex reports running totals; the harness stores what this wave used.
            self.assertEqual(agent["usage"]["outer"], {"input": 1200, "cachedInput": 800, "output": 150})
            self.assertEqual(agent["usage"]["otherSessions"], 1)
            if codex_agent.container_mode():  # the fake leaves a detached `sleep` behind
                self.assertGreaterEqual(agent["leftoverProcessesKilled"], 1)
            if number == 1:
                session = agent["events"]["sessionId"]
                self.assertIsNone(agent["resumedSession"])
                self.assertNotIn("resume", agent["command"])
            else:
                self.assertEqual(agent["resumedSession"], session)
                self.assertIn("resume", agent["command"])
            self.assertIn(f"S{number}", (folder / "last-message.txt").read_text(encoding="utf-8"))
            self.assertIn(read_json(folder / "checks.json")["status"], ("pass", "fail", "error"))
        self.assertIsNotNone(session)

        frozen = self.out / "audit" / "final-freeze"
        self.assertTrue((frozen / "snapshot.json").is_file())
        for number in range(1, 5):
            self.assertTrue((frozen / "immutable-main" / f"FAKE_S{number}.md").is_file())
        self.assertIn("Install conventional workflow",
                      lifecycle.git(self.work / "roombook", "log", "--format=%s", "main"))
        self.assertEqual(json.loads((self.work / "roombook" / ".study" / "station.json")
                                    .read_text(encoding="utf-8"))["station"], "S4")

        if codex_agent.container_mode():
            self.assertEqual(self.out.stat().st_mode & 0o777, 0o755)
        result = read_json(self.out / "report.json")
        self.assertEqual(len(result["stations"]), 4)
        self.assertEqual(result["totals"]["tokens"], {"input": 4800, "cachedInput": 3200, "output": 600})
        self.assertEqual(result["totals"]["tokensAllSessions"], {"input": 5200, "cachedInput": 3200, "output": 640})
        self.assertTrue((self.out / "evidence" / "codex-sessions").is_dir())
        self.assertEqual(result["classification"], {"class": "none", "reason": "completed S1-S4", "signals": []})
        self.assertEqual((state["stationsPlanned"], state["caseStations"], state["stopCategory"]), (4, 4, None))
        self.assertIsNone(state["hostPlatform"])  # only the host knows it; it passes it in
        self.assertEqual((result["fairness"]["stationsPlanned"], result["fairness"]["hostPlatform"]), (4, None))
        self.assertEqual(result["stratum"], "outer=fake")
        self.assertFalse((self.work / "roombook" / "CLAUDE.md").exists())
        self.assertIsNotNone(result["final"])
        self.assertIn(result["final"]["status"], ("pass", "fail", "error"))
        self.assertTrue((self.out / "report.md").is_file())

    def test_failures_timeouts_and_odd_repo_content_do_not_end_the_run(self):
        plan = {"S1": "crlf", "S2": "fail-after-work", "S3": "no-thread", "S4": "timeout"}
        (self.home / ".codex").mkdir(parents=True)
        (self.home / ".codex" / "fake-plan.json").write_text(json.dumps(plan), encoding="utf-8")
        data = make_manifest(limits={"stationSeconds": 8, "totalSeconds": 1200})
        code = self.run_trajectory(data)
        error = self.out / "runner-error.txt"
        self.assertEqual(code, 0, error.read_text(encoding="utf-8") if error.exists() else "")
        agents = [read_json(self.out / "stations" / f"S{n}" / "agent.json") for n in range(1, 5)]
        self.assertEqual([a["exitCode"] for a in agents[:3]], [0, 1, 0])
        self.assertTrue(agents[3]["timedOut"])
        first = agents[0]["sessionId"]
        self.assertEqual((agents[2]["resumedSession"], agents[2]["sessionId"]), (first, first))
        self.assertFalse(any(a["sessionSwitched"] for a in agents))
        # Session records cover waves without a turn.completed event (failure, timeout).
        self.assertEqual([a["usage"]["outer"]["input"] for a in agents], [1200] * 4)
        for number in range(1, 5):
            self.assertEqual(read_json(self.out / "stations" / f"S{number}" / "checks.json")["station"], number)
        frozen = self.out / "audit" / "final-freeze" / "immutable-main"
        self.assertNotIn(b"\r\n", (frozen / "FAKE_S1.md").read_bytes())  # in-tree eol=crlf is ignored

    def test_snapshot_failure_keeps_the_station_and_still_freezes(self):
        real = lifecycle.snapshot
        calls = []

        def failing(*args, **kwargs):
            calls.append(1)
            if len(calls) == 2:
                raise lifecycle.LifecycleError("disk full")
            return real(*args, **kwargs)

        with patch.object(runner.lifecycle, "snapshot", side_effect=failing):
            self.assertEqual(self.run_trajectory(make_manifest()), 1)
        state = read_json(self.out / "runner.json")
        self.assertEqual((state["status"], state["stationsRun"]), ("stopped", 2))
        self.assertIn("S2: snapshot failed", state["stopReason"])
        self.assertEqual(state["stopCategory"], "harness")
        self.assertEqual(read_json(self.out / "report.json")["classification"]["class"], "harness")
        self.assertTrue((self.out / "stations" / "S2" / "agent.json").is_file())
        self.assertIn("disk full", (self.out / "stations" / "S2" / "snapshot-error.txt").read_text(encoding="utf-8"))
        self.assertTrue((self.out / "audit" / "final-freeze").is_dir())
        self.assertIsNotNone(read_json(self.out / "final" / "final.json")["total"])
        self.assertEqual(read_json(self.out / "report.json")["stations"][1]["harnessErrors"], ["snapshot-error.txt"])

    def test_blocked_setup_stops_with_exit_1_and_a_report(self):
        in_dir = self.root / "in"
        shutil.copytree(PLAYGROUND / "cases", in_dir / "cases")  # no bin/markitect
        data = make_manifest(method="markitect", markitect={"sourceRepo": "unused", "commit": "669cecd2"})
        self.assertEqual(self.run_trajectory(data, in_dir), 1)
        state = read_json(self.out / "runner.json")
        self.assertEqual((state["status"], state["stationsRun"]), ("stopped", 0))
        self.assertIn("blocked", state["stopReason"])
        self.assertFalse((self.out / "stations").exists())
        self.assertTrue((self.out / "audit" / "final-freeze").is_dir())
        result = read_json(self.out / "report.json")
        self.assertEqual(result["setup"]["status"], "blocked")
        # The staged inputs lack the binary: our staging failed, not the product.
        self.assertEqual((state["stopCategory"], result["classification"]["class"]), ("harness", "harness"))

    def test_codex_without_auth_stops_before_setup(self):
        data = make_manifest(agent={"kind": "codex", "codexVersion": "0.162.0", "model": "gpt-6-luna",
                                    "effort": "high", "maxSubagents": 1})
        with patch.object(runner, "_codex_version", return_value=None):
            self.assertEqual(self.run_trajectory(data), 1)
        state = read_json(self.out / "runner.json")
        self.assertFalse(state["authInstalled"])
        self.assertIn("auth", state["stopReason"])
        self.assertEqual(state["stopCategory"], "environment")
        self.assertFalse((self.out / "setup").exists())

    def test_results_stay_root_only_even_when_the_agent_owns_the_mount(self):
        if not codex_agent.container_mode():
            self.skipTest("needs root and user agent")
        uid, gid = codex_agent._agent_ids()
        self.out.mkdir()
        os.chown(self.out, uid, gid)  # a Linux host user with uid 1000, like the agent
        seen = []
        with patch.object(runner._Trajectory, "run", lambda trajectory: seen.append(self.out.stat()) or 0):
            self.run_trajectory(make_manifest())
        self.assertEqual((seen[0].st_uid, seen[0].st_mode & 0o777), (0, 0o700))
        after = self.out.stat()
        self.assertEqual((after.st_uid, after.st_gid, after.st_mode & 0o777), (uid, gid, 0o755))

    def test_runner_error_writes_traceback_and_report(self):
        with patch.object(runner.lifecycle, "prepare", side_effect=RuntimeError("seed exploded")):
            self.assertEqual(self.run_trajectory(make_manifest()), 2)
        self.assertIn("seed exploded", (self.out / "runner-error.txt").read_text(encoding="utf-8"))
        self.assertEqual(read_json(self.out / "report.json")["status"], "error")


class ClaudeTrajectoryTests(RunnerTestCase):
    def test_fake_claude_runs_all_stations_and_never_keeps_the_token(self):
        secret = "fixture-oauth-token-0123456789"
        token = self.root / "claude-token"
        token.write_text(secret + "\n", encoding="utf-8")
        code = self.run_trajectory(make_manifest(agent=claude_agent_cfg()), token_src=token)
        error = self.out / "runner-error.txt"
        self.assertEqual(code, 0, error.read_text(encoding="utf-8") if error.exists() else "")
        state = read_json(self.out / "runner.json")
        self.assertEqual((state["status"], state["stationsRun"], state["claudeTokenProvided"]), ("completed", 4, True))
        self.assertGreater(state["tokenRedactions"], 0)  # the fake printed its environment
        self.assertEqual(files_containing(self.out, secret), [])
        self.assertEqual(files_containing(self.work, secret), [])
        self.assertEqual(json.loads((self.home / ".claude" / "mcp-config.json").read_text(encoding="utf-8")),
                         {"mcpServers": {}})
        self.assertEqual((self.work / "roombook" / "CLAUDE.md").read_text(encoding="utf-8"), "@AGENTS.md\n")
        session = None
        for number in range(1, 5):
            folder = self.out / "stations" / f"S{number}"
            agent = read_json(folder / "agent.json")
            self.assertEqual(agent["exitCode"], 0, (folder / "stderr.log").read_text(encoding="utf-8"))
            self.assertIn("--strict-mcp-config", agent["command"])
            self.assertNotIn(secret, json.dumps(agent))
            self.assertIn('"fakeTokenSeen": true', (folder / "events.jsonl").read_text(encoding="utf-8"))
            self.assertEqual(agent["usage"]["outer"], {"input": 1200, "cachedInput": 800, "output": 150})
            self.assertEqual(agent["usage"]["all"], {"input": 1300, "cachedInput": 800, "output": 160})
            self.assertEqual(agent["usage"]["otherSessions"], 1)  # the subagent's transcript
            self.assertEqual((agent["events"]["commands"], agent["events"]["collabToolCalls"]), (6, 1))
            self.assertEqual(agent["newMainCommits"], 2)
            self.assertTrue((folder / "last-message.txt").read_text(encoding="utf-8").startswith("Merged"))
            if number == 1:
                session = agent["sessionId"]
                self.assertNotIn("--resume", agent["command"])
            else:
                self.assertEqual((agent["resumedSession"], agent["sessionId"]), (session, session))
                self.assertEqual(agent["command"][agent["command"].index("--resume") + 1], session)
        self.assertTrue((self.out / "evidence" / "claude-sessions").is_dir())
        result = read_json(self.out / "report.json")
        self.assertEqual(result["totals"]["tokens"], {"input": 4800, "cachedInput": 3200, "output": 600})
        self.assertEqual(result["totals"]["tokensAllSessions"], {"input": 5200, "cachedInput": 3200, "output": 640})
        self.assertEqual((result["classification"]["class"], result["fairness"]["outerProvider"],
                          result["stratum"]), ("none", "fake-claude", "outer=fake-claude"))
        self.assertTrue(result["setup"]["claudeRouter"]["added"])
        text = (self.out / "report.md").read_text(encoding="utf-8")
        self.assertIn("Claude Code's session transcripts", text)
        self.assertIn("redacted", text)

    def test_claude_without_token_stops_before_setup(self):
        self.assertEqual(self.run_trajectory(make_manifest(agent=claude_agent_cfg("claude"))), 1)
        state = read_json(self.out / "runner.json")
        self.assertEqual((state["claudeTokenProvided"], state["stopCategory"]), (False, "environment"))
        self.assertIn("claude token", state["stopReason"])
        self.assertFalse((self.out / "setup").exists())
        self.assertEqual(read_json(self.out / "report.json")["classification"]["class"], "environment")

    def test_claude_with_markitect_needs_the_codex_login_too(self):
        token = self.root / "claude-token"
        token.write_text("fixture-token", encoding="utf-8")
        data = make_manifest(agent=claude_agent_cfg("claude"), method="markitect",
                             markitect={"sourceRepo": "unused", "commit": "669cecd2", "innerModel": "gpt-6-luna",
                                        "innerEffort": "high"})
        self.assertEqual(self.run_trajectory(data, token_src=token), 1)
        state = read_json(self.out / "runner.json")
        self.assertIn("codex auth", state["stopReason"])
        self.assertEqual(state["stopCategory"], "environment")

    def test_rejected_login_stops_as_environment(self):
        (self.home / ".claude").mkdir(parents=True)
        (self.home / ".claude" / "fake-plan.json").write_text(json.dumps({"S1": "login-error"}), encoding="utf-8")
        self.assertEqual(self.run_trajectory(make_manifest(agent=claude_agent_cfg())), 1)
        state = read_json(self.out / "runner.json")
        self.assertIn("without doing any work", state["stopReason"])
        self.assertEqual((state["stopCategory"], state["stationsRun"]), ("environment", 1))


class SixStationTests(RunnerTestCase):
    def test_stations_come_from_the_case_plan(self):
        in_dir = self.six_station_seed()
        code = self.run_trajectory(make_manifest(case="readinglog2"), in_dir)
        error = self.out / "runner-error.txt"
        self.assertEqual(code, 0, error.read_text(encoding="utf-8") if error.exists() else "")
        state = read_json(self.out / "runner.json")
        self.assertEqual((state["stationsPlanned"], state["stationsRun"]), (6, 6))
        frozen = self.out / "audit" / "final-freeze"
        self.assertEqual(read_json(frozen / "snapshot.json")["freezeReason"], "completed S1-S6")
        for number in range(1, 7):
            self.assertTrue((frozen / "immutable-main" / f"FAKE_S{number}.md").is_file())
        self.assertEqual(read_json(self.out / "final" / "final.json")["checks"]["station"], 6)
        result = read_json(self.out / "report.json")
        self.assertEqual((result["totals"]["stationsPlanned"], result["totals"]["stationsRun"]), (6, 6))
        self.assertEqual(result["classification"]["reason"], "completed S1-S6")
        self.assertIn("6 of 6 ran", (self.out / "report.md").read_text(encoding="utf-8"))


class StationSelectionTests(RunnerTestCase):
    def test_the_run_takes_the_first_stations_only(self):
        host_os = {"MPG_HOST_SYSTEM": "Linux", "MPG_HOST_MACHINE": "x86_64"}
        with patch.dict(os.environ, host_os):
            code = self.run_trajectory(make_manifest(stations=2))
        error = self.out / "runner-error.txt"
        self.assertEqual(code, 0, error.read_text(encoding="utf-8") if error.exists() else "")
        state = read_json(self.out / "runner.json")
        self.assertEqual((state["stationsPlanned"], state["caseStations"], state["stationsRun"]), (2, 4, 2))
        self.assertEqual(state["hostPlatform"], {"system": "Linux", "machine": "x86_64"})
        self.assertEqual(sorted(path.name for path in (self.out / "stations").iterdir()), ["S1", "S2"])
        self.assertEqual(read_json(self.out / "audit" / "final-freeze" / "snapshot.json")["freezeReason"],
                         "completed S1-S2")
        self.assertEqual(read_json(self.out / "final" / "final.json")["checks"]["station"], 2)
        result = read_json(self.out / "report.json")
        self.assertEqual((result["fairness"]["stationsPlanned"], result["fairness"]["hostPlatform"]),
                         (2, {"system": "Linux", "machine": "x86_64"}))
        text = (self.out / "report.md").read_text(encoding="utf-8")
        self.assertIn("2 of 2 ran (the first 2 of the case's 4)", text)
        self.assertIn("stations 2; host Linux x86_64", text)


class StopRuleTests(unittest.TestCase):
    def test_infrastructure_failures_stop_but_agent_failures_do_not(self):
        tokens = {"input": 5, "cachedInput": 0, "output": 1}
        idle = {"items": 0, "commands": 0, "tokens": {"input": None, "cachedInput": None, "output": None}}
        busy = {"items": 4, "commands": 4, "tokens": tokens}
        mcp_only = {"items": 2, "commands": 0, "tokens": {"input": None, "cachedInput": None, "output": None}}
        self.assertIn("session id", runner._stop_reason(1, None, {"exitCode": 1}, idle))
        self.assertIn("could not be started", runner._stop_reason(2, "t", {"exitCode": None}, idle))
        self.assertIn("without doing any work", runner._stop_reason(2, "t", {"exitCode": 1}, idle))
        self.assertIsNone(runner._stop_reason(2, "t", {"exitCode": 1}, busy))
        self.assertIsNone(runner._stop_reason(2, "t", {"exitCode": 1}, mcp_only))  # MCP/subagent work counts
        self.assertIsNone(runner._stop_reason(2, "t", {"exitCode": None, "timedOut": True}, idle))
        self.assertIsNone(runner._stop_reason(1, "t", {"exitCode": 0}, idle))


class UsageTests(unittest.TestCase):
    def test_per_wave_deltas_from_running_totals(self):
        def t(i, c, o):
            return {"input": i, "cachedInput": c, "output": o}

        before = {"outer": t(100, 50, 10), "old-helper": t(5, 0, 1)}
        after = {"outer": t(160, 80, 15), "old-helper": t(5, 0, 1), "inner": t(40, 0, 4)}
        self.assertEqual(runner._usage(before, after, "outer"),
                         {"outer": t(60, 30, 5), "all": t(100, 30, 9), "otherSessions": 1})
        self.assertEqual(runner._usage(before, before, "outer")["outer"], t(0, 0, 0))
        unknown = runner._usage({}, {"inner": t(1, 1, 1)}, "outer")
        self.assertEqual((unknown["outer"]["input"], unknown["all"]["input"]), (None, None))

    def test_overhead_bound_covers_setup_and_assessment(self):
        self.assertGreater(runner.overhead_bound_seconds(), 7 * 600 + 4 * 600 + 600 + 2 * 900)
        self.assertGreater(runner.overhead_bound_seconds(6), runner.overhead_bound_seconds(4) + 2 * 600)


class EntryPointTests(unittest.TestCase):
    def test_main_rejects_bad_manifest_with_exit_2(self):
        with tempfile.TemporaryDirectory() as folder:
            bad = Path(folder) / "bad.json"
            bad.write_text('{"schema": 2}', encoding="utf-8")
            with contextlib.redirect_stderr(io.StringIO()):
                code = runner.main(["--manifest", str(bad), "--out", str(Path(folder) / "out")])
            self.assertEqual(code, 2)
            self.assertIn("manifest error", (Path(folder) / "out" / "runner-error.txt").read_text(encoding="utf-8"))

    def test_package_entry_requires_a_subcommand(self):
        with contextlib.redirect_stderr(io.StringIO()):
            self.assertEqual(entry.main([]), 2)
            self.assertEqual(entry.main(["nonsense"]), 2)


if __name__ == "__main__":
    unittest.main()
