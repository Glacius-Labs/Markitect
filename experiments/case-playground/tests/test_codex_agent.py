import json
import os
import subprocess
import sys
import tempfile
import time
import tomllib
import unittest
from pathlib import Path

from playground import codex_agent

AGENT_CFG = {"kind": "codex", "codexVersion": "0.162.0", "model": "gpt-6-luna",
             "effort": "high", "maxSubagents": 3}


def agent_user_exists() -> bool:
    try:
        codex_agent._agent_ids()
        return True
    except (ImportError, KeyError):
        return False


class ConfigTests(unittest.TestCase):
    def test_minimal_config_parses_with_expected_keys(self):
        with tempfile.TemporaryDirectory() as temp:
            path = codex_agent.write_config(Path(temp) / ".codex", AGENT_CFG, {
                "markitect": {"command": "/usr/local/bin/markitect",
                              "args": ["project", "mcp", "--repo", "/work/readinglog"]}})
            data = tomllib.loads(path.read_text(encoding="utf-8"))
        self.assertEqual(data["model"], "gpt-6-luna")
        self.assertEqual(data["model_reasoning_effort"], "high")
        self.assertEqual(data["features"], {"memories": False})
        self.assertEqual(data["agents"], {"max_concurrent_threads_per_session": 3,
                                          "default_subagent_model": "gpt-6-luna",
                                          "default_subagent_reasoning_effort": "high"})
        self.assertEqual(data["mcp_servers"]["markitect"]["args"][-1], "/work/readinglog")

    def test_strings_and_keys_are_escaped(self):
        tricky = 'a "quoted" \\ path\twith\nnewline\x7f and ümlaut'
        text = codex_agent.render_config(
            {**AGENT_CFG, "model": tricky}, {"odd name.x": {"command": tricky, "args": [tricky, ""]}})
        data = tomllib.loads(text)
        self.assertEqual(data["model"], tricky)
        self.assertEqual(data["mcp_servers"]["odd name.x"], {"command": tricky, "args": [tricky, ""]})

    def test_no_mcp_servers_and_missing_optional_values(self):
        data = tomllib.loads(codex_agent.render_config({"kind": "fake"}, {}))
        self.assertEqual(data, {"features": {"memories": False}, "agents": {}})

    def test_install_auth(self):
        with tempfile.TemporaryDirectory() as temp:
            home = Path(temp) / "home" / ".codex"
            self.assertFalse(codex_agent.install_auth(Path(temp) / "missing.json", home))
            source = Path(temp) / "fixture-auth.json"
            source.write_text('{"fixture": true}', encoding="utf-8")
            self.assertTrue(codex_agent.install_auth(source, home))
            self.assertEqual((home / "auth.json").read_text(encoding="utf-8"), '{"fixture": true}')
            if os.name == "posix":
                self.assertEqual((home / "auth.json").stat().st_mode & 0o777, 0o600)


class CommandTests(unittest.TestCase):
    def test_start_and_resume(self):
        last = Path("/out/stations/S1/last-message.txt")
        self.assertEqual(codex_agent.build_command(AGENT_CFG, "do it", None, last), [
            "codex", "exec", "--json", "--dangerously-bypass-approvals-and-sandbox",
            "--skip-git-repo-check", "-o", str(last), "do it"])
        self.assertEqual(codex_agent.build_command(AGENT_CFG, "do it", "sid-1", last), [
            "codex", "exec", "resume", "sid-1", "--json",
            "--dangerously-bypass-approvals-and-sandbox", "-o", str(last), "do it"])

    def test_fake_kind_uses_staged_fake_agent(self):
        cmd = codex_agent.build_command({"kind": "fake"}, "p", "sid", Path("m.txt"))
        self.assertEqual(cmd[0], sys.executable)
        self.assertEqual(Path(cmd[1]).name, "fake_agent.py")
        self.assertTrue(Path(cmd[1]).is_file())
        self.assertEqual(cmd[2:5], ["exec", "resume", "sid"])

    def test_unknown_kind(self):
        with self.assertRaises(ValueError):
            codex_agent.build_command({"kind": "claude"}, "p", None, Path("m.txt"))


class ParseEventsTests(unittest.TestCase):
    def parse(self, lines):
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / "events.jsonl"
            path.write_text("\n".join(lines) + "\n", encoding="utf-8")
            return codex_agent.parse_events(path)

    def test_real_shaped_events(self):
        events = [
            {"type": "thread.started", "thread_id": "01a11f4e-7bcb-7782-9db7-523a23bead85"},
            {"type": "turn.started"},
            {"type": "item.started", "item": {"id": "item_0", "type": "command_execution"}},
            {"type": "item.completed", "item": {"id": "item_0", "type": "command_execution",
                                                "command": "ls", "exit_code": 0}},
            {"type": "item.completed", "item": {"id": "item_1", "type": "file_change",
                                                "changes": [{"path": "a.py", "kind": "add"}]}},
            {"type": "item.completed", "item": {"id": "item_2", "type": "agent_message", "text": "ok"}},
            {"type": "item.completed", "item": {"id": "item_3", "type": "error", "message": "x"}},
            {"type": "item.completed", "item": {"id": "item_4", "type": "mcp_tool_call", "tool": "t"}},
            {"type": "item.completed", "item": {"id": "item_5", "type": "collab_tool_call"}},
            # Seen in a real Markitect run: the tool answered "not conforming" (status failed, no
            # error) and a call rejected by the tool schema (error set).
            {"type": "item.completed", "item": {"id": "item_6", "type": "mcp_tool_call", "server": "markitect",
                                                "tool": "project_check", "error": None, "status": "failed"}},
            {"type": "item.completed", "item": {"id": "item_7", "type": "mcp_tool_call", "server": "markitect",
                                                "tool": "project_explore", "status": "failed", "error": {
                                                    "message": "Mcp error: -32602: Arguments do not match"}}},
            {"type": "turn.completed", "usage": {"input_tokens": 72406, "cached_input_tokens": 51200,
                                                 "cache_write_input_tokens": 0, "output_tokens": 775,
                                                 "reasoning_output_tokens": 349}},
            {"type": "thread.started", "thread_id": "other"},
            {"type": "turn.completed", "usage": {"input_tokens": 10, "cached_input_tokens": 5,
                                                 "output_tokens": 1}},
            {"type": "turn.failed", "error": {"message": "boom"}},
            {"type": "error", "message": "stream error"},
            {"type": "something.new", "payload": 1},
        ]
        lines = [json.dumps(event) for event in events] + ["", "not json", "[1, 2]", '{"type": "turn.']
        result = self.parse(lines)
        # turn.completed carries the thread's running total: the last value wins, no sum.
        self.assertEqual(result, {
            "sessionId": "01a11f4e-7bcb-7782-9db7-523a23bead85", "items": 8, "commands": 1,
            "fileChanges": 1, "mcpToolCalls": 3, "collabToolCalls": 1, "errors": 3, "malformedLines": 3,
            "mcpToolFailures": 2, "mcpCallErrors": 1, "tokens": {"input": 10, "cachedInput": 5, "output": 1}})

    def test_missing_file_and_no_usage_stay_null(self):
        result = codex_agent.parse_events(Path("does-not-exist.jsonl"))
        self.assertIsNone(result["sessionId"])
        self.assertEqual(result["tokens"], {"input": None, "cachedInput": None, "output": None})
        result = self.parse([json.dumps({"type": "thread.started", "thread_id": "t"})])
        self.assertEqual(result["sessionId"], "t")
        self.assertIsNone(result["tokens"]["input"])


class SessionTotalsTests(unittest.TestCase):
    def test_latest_cumulative_total_per_thread(self):
        def record(thread, *totals):
            lines = [{"type": "session_meta", "payload": {"id": thread, "source": "exec"}},
                     {"type": "response_item", "payload": {"type": "message"}}]
            lines += [{"type": "event_msg", "payload": {"type": "token_count", "info": {"total_token_usage": {
                "input_tokens": i, "cached_input_tokens": c, "output_tokens": o}}}} for i, c, o in totals]
            return "\n".join(json.dumps(line) for line in lines) + "\nnot json\n"

        with tempfile.TemporaryDirectory() as temp:
            day = Path(temp) / "sessions" / "2026" / "10" / "09"
            day.mkdir(parents=True)
            (day / "rollout-a.jsonl").write_text(record("outer", (15145, 11008, 422), (171275, 147712, 1643)),
                                                 encoding="utf-8")
            (day / "rollout-b.jsonl").write_text(record("helper", (100, 0, 10)), encoding="utf-8")
            (day / "rollout-c.jsonl").write_text(record("no-usage"), encoding="utf-8")
            (day / "notes.jsonl").write_text(record("ignored", (1, 1, 1)), encoding="utf-8")
            totals = codex_agent.session_totals(Path(temp) / "sessions")
        self.assertEqual(totals, {"outer": {"input": 171275, "cachedInput": 147712, "output": 1643},
                                  "helper": {"input": 100, "cachedInput": 0, "output": 10}})
        self.assertEqual(codex_agent.session_totals(Path(temp) / "missing"), {})


@unittest.skipIf(codex_agent.container_mode(), "test mode only (not root)")
class RunAsAgentTestModeTests(unittest.TestCase):
    def test_success_and_extra_env(self):
        with tempfile.TemporaryDirectory() as temp:
            out, err = Path(temp) / "o" / "stdout.txt", Path(temp) / "o" / "stderr.txt"
            result = codex_agent.run_as_agent(
                [sys.executable, "-c", "import os; print(os.environ['MPG_TEST'])"],
                Path(temp), 30, out, err, extra_env={"MPG_TEST": "hello"})
            self.assertEqual(result["exitCode"], 0)
            self.assertFalse(result["timedOut"])
            self.assertEqual(out.read_text(encoding="utf-8").strip(), "hello")

    def test_timeout_kills_process(self):
        with tempfile.TemporaryDirectory() as temp:
            started = time.monotonic()
            result = codex_agent.run_as_agent(
                [sys.executable, "-c", "import time; time.sleep(60)"], Path(temp), 1,
                Path(temp) / "stdout.txt", Path(temp) / "stderr.txt")
            self.assertTrue(result["timedOut"])
            self.assertNotEqual(result["exitCode"], 0)
            self.assertLess(time.monotonic() - started, 30)

    def test_missing_binary(self):
        with tempfile.TemporaryDirectory() as temp:
            result = codex_agent.run_as_agent(["mpg-no-such-binary"], Path(temp), 5,
                                              Path(temp) / "stdout.txt", Path(temp) / "stderr.txt")
            self.assertIsNone(result["exitCode"])
            self.assertIn("cannot start agent", (Path(temp) / "stderr.txt").read_text(encoding="utf-8"))

    def test_kill_all_is_noop(self):
        self.assertEqual(codex_agent.kill_all_agent_processes(), 0)


@unittest.skipUnless(codex_agent.container_mode() and agent_user_exists(), "needs root and user agent")
class RunAsAgentContainerTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.dir = Path(self.temp.name)
        uid, gid = codex_agent._agent_ids()
        os.chown(self.dir, uid, gid)

    def tearDown(self):
        codex_agent.kill_all_agent_processes()
        self.temp.cleanup()

    def test_runs_as_agent_with_clean_env(self):
        script = "import os, json; print(json.dumps([os.getuid(), os.getgroups(), dict(os.environ)]))"
        result = codex_agent.run_as_agent(["python3", "-c", script], self.dir, 30,
                                          self.dir / "out.txt", self.dir / "err.txt", {"X": "1"})
        self.assertEqual(result["exitCode"], 0, (self.dir / "err.txt").read_text(encoding="utf-8"))
        uid, groups, env = json.loads((self.dir / "out.txt").read_text(encoding="utf-8"))
        self.assertEqual(uid, codex_agent._agent_ids()[0])
        self.assertNotIn(0, groups)
        self.assertEqual(env["HOME"], "/home/agent")
        self.assertEqual(env["CODEX_HOME"], "/home/agent/.codex")
        self.assertEqual(env["X"], "1")
        self.assertNotIn("MPG_IMAGE_ID", env)

    def test_orphans_are_counted_and_killed(self):
        spawn = ("import subprocess; subprocess.Popen(['sleep', '600'], start_new_session=True, "
                 "stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)")
        result = codex_agent.run_as_agent(["python3", "-c", spawn], self.dir, 30,
                                          self.dir / "out.txt", self.dir / "err.txt")
        self.assertEqual(result["exitCode"], 0)
        self.assertGreaterEqual(codex_agent.kill_all_agent_processes(), 1)
        self.assertEqual(codex_agent.kill_all_agent_processes(), 0)

    def test_timeout_kills_detached_children(self):
        spawn = ("import subprocess, time; subprocess.Popen(['sleep', '600'], start_new_session=True); "
                 "time.sleep(600)")
        result = codex_agent.run_as_agent(["python3", "-c", spawn], self.dir, 2,
                                          self.dir / "out.txt", self.dir / "err.txt")
        self.assertTrue(result["timedOut"])
        self.assertGreaterEqual(result["killedOnTimeout"], 1)
        self.assertEqual(codex_agent.kill_all_agent_processes(), 0)


@unittest.skipUnless(subprocess.run(["git", "--version"], capture_output=True).returncode == 0,
                     "needs git")
class FakeAgentTests(unittest.TestCase):
    def test_start_and_resume_merge_to_main(self):
        with tempfile.TemporaryDirectory() as temp:
            repo, home = Path(temp) / "repo", Path(temp) / "codex-home"
            git_env = {**os.environ, "GIT_CONFIG_NOSYSTEM": "1",
                       "GIT_CONFIG_GLOBAL": str(Path(temp) / "gitconfig"),
                       "CODEX_HOME": str(home), "FAKE_AGENT_NO_ORPHAN": "1"}
            ident = ["-c", "user.name=t", "-c", "user.email=t@t.invalid"]

            def git(*args):
                return subprocess.run(["git", *ident, *args], cwd=repo, env=git_env, check=True,
                                      capture_output=True, text=True, encoding="utf-8").stdout

            repo.mkdir()
            git("init", "-q", "-b", "main")
            (repo / ".study").mkdir()
            (repo / "README.md").write_text("seed\n", encoding="utf-8")
            git("add", "README.md")
            git("commit", "-q", "-m", "seed")
            session = None
            for number in (1, 2):
                (repo / ".study" / "station.json").write_text(
                    json.dumps({"schema": 1, "station": f"S{number}", "items": [f"R0{number}"]}),
                    encoding="utf-8")
                last = Path(temp) / f"last-{number}.txt"
                cmd = codex_agent.build_command({"kind": "fake"}, "prompt", session, last)
                done = subprocess.run(cmd, cwd=repo, env=git_env, capture_output=True, text=True,
                                      encoding="utf-8")
                self.assertEqual(done.returncode, 0, done.stderr + done.stdout)
                events = Path(temp) / f"events-{number}.jsonl"
                events.write_text(done.stdout, encoding="utf-8")
                summary = codex_agent.parse_events(events)
                session = session or summary["sessionId"]
                self.assertEqual(summary["sessionId"], session)
                self.assertEqual(summary["commands"], 5)
                self.assertEqual(summary["fileChanges"], 1)
                self.assertEqual(summary["tokens"]["output"], 150 * number)  # cumulative, like Codex
                totals = codex_agent.session_totals(home / "sessions")
                self.assertEqual(totals[session]["output"], 150 * number)
                self.assertEqual(len(totals), 1 + number)  # plus one helper session per run
                self.assertTrue(last.read_text(encoding="utf-8").startswith("Merged"))
            self.assertEqual(git("ls-tree", "--name-only", "main").split(),
                             ["FAKE_S1.md", "FAKE_S2.md", "README.md"])
            bad = subprocess.run(codex_agent.build_command({"kind": "fake"}, "p", "unknown", last),
                                 cwd=repo, env=git_env, capture_output=True, text=True, encoding="utf-8")
            self.assertEqual(bad.returncode, 1)


if __name__ == "__main__":
    unittest.main()
