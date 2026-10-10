import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

from playground import claude_agent

AGENT_CFG = {"kind": "claude", "claudeVersion": "2.1.296", "codexVersion": "0.162.0",
             "model": "claude-opus-5-5", "effort": "high", "maxSubagents": 3}


def assistant(message_id, blocks, usage=None, *, parent=None, model="claude-opus-5-5", session="s-1"):
    message = {"id": message_id, "model": model, "content": blocks}
    if usage is not None:
        message["usage"] = usage
    return {"type": "assistant", "message": message, "parent_tool_use_id": parent, "session_id": session}


def tool_result(tool_id, content, is_error=False):
    return {"type": "user", "message": {"role": "user", "content": [
        {"type": "tool_result", "tool_use_id": tool_id, "content": content, "is_error": is_error}]},
        "parent_tool_use_id": None, "session_id": "s-1"}


def usage(fresh, written, read, out):
    return {"input_tokens": fresh, "cache_creation_input_tokens": written, "cache_read_input_tokens": read,
            "output_tokens": out}


class CommandTests(unittest.TestCase):
    def test_start_and_resume(self):
        config = Path("/home/agent/.claude/mcp-config.json")
        start = claude_agent.build_command(AGENT_CFG, "do it", None, config)
        self.assertEqual(start, [
            "claude", "-p", "--output-format", "stream-json", "--verbose", "--model", "claude-opus-5-5",
            "--effort", "high", "--dangerously-skip-permissions", f"--mcp-config={config}",
            "--strict-mcp-config", "--", "do it"])
        resume = claude_agent.build_command(AGENT_CFG, "do it", "sid-1", config)
        self.assertEqual(resume[resume.index("--resume") + 1], "sid-1")
        self.assertEqual(resume[-2:], ["--", "do it"])
        # The variadic --mcp-config never stands apart from its value, so it cannot take the prompt.
        self.assertNotIn("--mcp-config", resume)

    def test_fake_and_unknown_kinds(self):
        cmd = claude_agent.build_command({**AGENT_CFG, "kind": "fake-claude"}, "p", "sid", Path("m.json"))
        self.assertEqual(cmd[0], sys.executable)
        self.assertEqual(Path(cmd[1]).name, "fake_claude.py")
        self.assertTrue(Path(cmd[1]).is_file())
        with self.assertRaises(ValueError):
            claude_agent.build_command({**AGENT_CFG, "kind": "codex"}, "p", None, Path("m.json"))

    def test_environment_has_config_dir_and_no_token(self):
        env = claude_agent.environment(Path("/home/agent/.claude"))
        self.assertEqual(env["CLAUDE_CONFIG_DIR"], str(Path("/home/agent/.claude")))
        self.assertEqual((env["CLAUDE_CODE_DISABLE_AUTO_MEMORY"], env["DISABLE_AUTOUPDATER"]), ("1", "1"))
        self.assertNotIn(claude_agent.TOKEN_ENV, env)


class ConfigTests(unittest.TestCase):
    def test_mcp_config_lists_only_the_given_servers(self):
        with tempfile.TemporaryDirectory() as temp:
            path = claude_agent.write_config(Path(temp) / ".claude", {
                "markitect": {"command": "/usr/local/bin/markitect", "args": ["project", "mcp", "--repo", "/w"]}})
            self.assertEqual(path.name, "mcp-config.json")
            self.assertEqual(json.loads(path.read_text(encoding="utf-8")), {"mcpServers": {"markitect": {
                "type": "stdio", "command": "/usr/local/bin/markitect", "args": ["project", "mcp", "--repo", "/w"]}}})
            empty = claude_agent.write_config(Path(temp) / ".claude", {})
            self.assertEqual(json.loads(empty.read_text(encoding="utf-8")), {"mcpServers": {}})

    def test_read_token(self):
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / "token"
            self.assertIsNone(claude_agent.read_token(path))
            path.write_text("  fixture-token-1\n", encoding="utf-8")
            self.assertEqual(claude_agent.read_token(path), "fixture-token-1")
            path.write_text("\n", encoding="utf-8")
            self.assertIsNone(claude_agent.read_token(path))
            path.write_text("two words\n", encoding="utf-8")
            self.assertIsNone(claude_agent.read_token(path))


class ParseEventsTests(unittest.TestCase):
    def parse(self, events, extra_lines=()):
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / "events.jsonl"
            path.write_text("\n".join([json.dumps(e) for e in events] + list(extra_lines)) + "\n", encoding="utf-8")
            return claude_agent.parse_events(path), claude_agent.last_message(path)

    def test_stream_json_shaped_events(self):
        work = usage(50, 150, 400, 100)
        events = [
            {"type": "system", "subtype": "init", "session_id": "s-1", "model": "claude-opus-5-5",
             "mcp_servers": [{"name": "markitect", "status": "connected"}]},
            # One API message, repeated per content block with the same id and usage.
            {"type": "system", "subtype": "api_retry", "attempt": 1, "error": "unknown", "session_id": "s-1"},
            assistant("msg_1", [{"type": "thinking", "thinking": "..."}], work),
            assistant("msg_1", [{"type": "tool_use", "id": "t1", "name": "Bash", "input": {}}], work),
            tool_result("t1", "ok"),
            assistant("msg_1", [{"type": "tool_use", "id": "t2", "name": "Edit", "input": {}}], work),
            assistant("msg_1", [{"type": "tool_use", "id": "t3", "name": "mcp__markitect__project_check"}], work),
            tool_result("t3", [{"type": "text", "text": "{\"status\": \"failed\"}"}], is_error=True),
            assistant("msg_1", [{"type": "tool_use", "id": "t4", "name": "mcp__markitect__project_explore"}], work),
            tool_result("t4", "MCP error -32602: Arguments do not match closed tool schema", is_error=True),
            assistant("msg_1", [{"type": "tool_use", "id": "t5", "name": "Agent", "input": {}}], work),
            # A subagent's own messages: not the outer agent's activity or usage.
            assistant("msg_sub", [{"type": "tool_use", "id": "t6", "name": "Bash"}], usage(9, 9, 9, 9), parent="t5"),
            tool_result("t5", "done"),
            assistant("msg_2", [{"type": "text", "text": "Merged."}], usage(50, 150, 400, 50)),
            {"type": "result", "subtype": "success", "is_error": False, "num_turns": 3, "result": "All merged.",
             "session_id": "s-1", "total_cost_usd": 0.5, "usage": usage(100, 300, 800, 150),
             "modelUsage": {"claude-opus-5-5": {"inputTokens": 100, "outputTokens": 150, "cacheReadInputTokens": 800,
                                                "cacheCreationInputTokens": 300},
                            "claude-haiku": {"inputTokens": 10, "outputTokens": 1, "cacheReadInputTokens": 0,
                                             "cacheCreationInputTokens": 0}}},
            {"type": "something.new"},
        ]
        result, last = self.parse(events, ["", "not json", "[1]"])
        self.assertEqual({key: result[key] for key in ("sessionId", "items", "commands", "fileChanges", "mcpToolCalls",
                                                       "collabToolCalls", "errors", "malformedLines",
                                                       "mcpToolFailures", "mcpCallErrors")},
                         {"sessionId": "s-1", "items": 6, "commands": 1, "fileChanges": 1, "mcpToolCalls": 2,
                          "collabToolCalls": 1, "errors": 0, "malformedLines": 2, "mcpToolFailures": 2,
                          "mcpCallErrors": 1})
        self.assertEqual(result["tokens"], {"input": 1200, "cachedInput": 800, "output": 150})
        self.assertEqual(result["apiRetries"], 1)
        self.assertEqual(result["result"]["tokens"], {"input": 1200, "cachedInput": 800, "output": 150})
        self.assertEqual(result["result"]["tokensAllModels"], {"input": 1210, "cachedInput": 800, "output": 151})
        self.assertEqual((result["result"]["subtype"], result["result"]["costUsd"]), ("success", 0.5))
        self.assertEqual(last, "All merged.")

    def test_rejected_login_counts_as_no_work(self):
        events = [{"type": "system", "subtype": "init", "session_id": "s-2"},
                  assistant("m", [{"type": "text", "text": "Invalid API key"}], usage(0, 0, 0, 0),
                            model="<synthetic>", session="s-2"),
                  {"type": "result", "subtype": "success", "is_error": True, "result": "Invalid API key",
                   "session_id": "s-2", "usage": usage(0, 0, 0, 0), "modelUsage": {}}]
        result, _ = self.parse(events)
        self.assertEqual((result["sessionId"], result["items"], result["errors"]), ("s-2", 0, 1))
        self.assertEqual(result["tokens"], {"input": None, "cachedInput": None, "output": None})

    def test_missing_file_and_fallback_last_message(self):
        result = claude_agent.parse_events(Path("does-not-exist.jsonl"))
        self.assertIsNone(result["sessionId"])
        self.assertIsNone(result["result"])
        _, last = self.parse([assistant("m", [{"type": "text", "text": "partial answer"}], usage(1, 0, 0, 1))])
        self.assertEqual(last, "partial answer")


class SessionTotalsTests(unittest.TestCase):
    def test_transcripts_per_session_and_subagent(self):
        def line(message_id, used, *, side=False, kind="assistant", model="m"):
            return json.dumps({"type": kind, "isSidechain": side, "sessionId": "abc",
                               "message": {"id": message_id, "model": model, "usage": used, "content": []}})

        with tempfile.TemporaryDirectory() as temp:
            project = Path(temp) / "projects" / "-work-roombook"
            (project / "abc" / "subagents").mkdir(parents=True)
            (project / "abc.jsonl").write_text("\n".join([
                line("m1", usage(2, 100, 1000, 40)), line("m1", usage(2, 100, 1000, 40)),
                line("m2", usage(1, 0, 1100, 10)), line("m3", usage(5, 5, 5, 5), side=True),
                line("x", usage(0, 0, 0, 0), model="<synthetic>"), json.dumps({"type": "user", "usage": 1}),
                "not json"]) + "\n", encoding="utf-8")
            (project / "abc" / "subagents" / "agent-1.jsonl").write_text(
                line("s1", usage(100, 0, 0, 6), side=True) + "\n", encoding="utf-8")
            (project / "notes.txt").write_text(line("n", usage(1, 1, 1, 1)), encoding="utf-8")
            totals = claude_agent.session_totals(Path(temp) / "projects")
        self.assertEqual(totals, {
            "claude:abc": {"input": 2203, "cachedInput": 2100, "output": 50},
            "claude:abc:sidechain": {"input": 15, "cachedInput": 5, "output": 5},
            "claude:agent-1:sidechain": {"input": 100, "cachedInput": 0, "output": 6}})
        self.assertEqual(claude_agent.session_totals(Path(temp) / "missing"), {})


class RedactTests(unittest.TestCase):
    def test_secret_is_replaced_everywhere_but_skipped_folders(self):
        secret = "sk-ant-oat01-fixture"
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            (root / "stations" / "S1").mkdir(parents=True)
            (root / "audit").mkdir()
            (root / "stations" / "S1" / "events.jsonl").write_text(f'{{"x": "TOKEN={secret}"}}\n', encoding="utf-8")
            (root / "stations" / "S1" / "clean.txt").write_text("nothing here\n", encoding="utf-8")
            (root / "audit" / "copy.txt").write_text(secret, encoding="utf-8")
            (root / "runner-error.txt").write_text(f"{secret} {secret}", encoding="utf-8")
            self.assertEqual(claude_agent.redact_tree(root, secret, skip=("audit",)), 2)
            events = (root / "stations" / "S1" / "events.jsonl").read_text(encoding="utf-8")
            self.assertNotIn(secret, events)
            self.assertIn("[redacted:CLAUDE_CODE_OAUTH_TOKEN]", events)
            json.loads(events)  # still valid JSON
            self.assertEqual((root / "audit" / "copy.txt").read_text(encoding="utf-8"), secret)
            self.assertEqual(claude_agent.redact_tree(root, secret, skip=("audit",)), 0)
            self.assertEqual(claude_agent.redact_tree(root, None), 0)


@unittest.skipUnless(subprocess.run(["git", "--version"], capture_output=True).returncode == 0, "needs git")
class FakeClaudeTests(unittest.TestCase):
    def test_start_resume_transcripts_and_token_echo(self):
        with tempfile.TemporaryDirectory() as temp:
            repo, config = Path(temp) / "repo", Path(temp) / "claude-config"
            config.mkdir()
            mcp = claude_agent.write_config(config, {})
            env = {**os.environ, "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": str(Path(temp) / "gitconfig"),
                   "CLAUDE_CONFIG_DIR": str(config), "FAKE_AGENT_NO_ORPHAN": "1",
                   claude_agent.TOKEN_ENV: "fixture-token-123"}
            ident = ["-c", "user.name=t", "-c", "user.email=t@t.invalid"]

            def git(*args):
                return subprocess.run(["git", *ident, *args], cwd=repo, env=env, check=True,
                                      capture_output=True, text=True, encoding="utf-8").stdout

            def run(cmd):
                return subprocess.run(cmd, cwd=repo, env=env, capture_output=True, text=True, encoding="utf-8")

            repo.mkdir()
            git("init", "-q", "-b", "main")
            (repo / ".study").mkdir()
            (repo / "README.md").write_text("seed\n", encoding="utf-8")
            git("add", "README.md")
            git("commit", "-q", "-m", "seed")
            cfg = {**AGENT_CFG, "kind": "fake-claude"}
            session = None
            for number in (1, 2):
                (repo / ".study" / "station.json").write_text(
                    json.dumps({"schema": 1, "station": f"S{number}", "items": [f"R0{number}"]}), encoding="utf-8")
                done = run(claude_agent.build_command(cfg, "prompt", session, mcp))
                self.assertEqual(done.returncode, 0, done.stderr + done.stdout)
                self.assertIn("fixture-token-123", done.stdout)  # the agent printed its environment
                self.assertIn('"fakeTokenSeen": true', done.stdout)
                events = Path(temp) / f"events-{number}.jsonl"
                events.write_text(done.stdout, encoding="utf-8")
                summary = claude_agent.parse_events(events)
                session = session or summary["sessionId"]
                self.assertEqual(summary["sessionId"], session)  # resume keeps the session
                self.assertEqual((summary["commands"], summary["fileChanges"], summary["collabToolCalls"]), (6, 1, 1))
                self.assertEqual(summary["tokens"], {"input": 1200, "cachedInput": 800, "output": 150})
                self.assertEqual(summary["result"]["tokensAllModels"]["input"], 1300)
                totals = claude_agent.session_totals(config / "projects")
                self.assertEqual(totals[claude_agent.session_key(session)]["input"], 1200 * number)
                self.assertEqual(len(totals), 1 + number)  # plus one subagent transcript per run
                self.assertTrue(claude_agent.last_message(events).startswith("Merged"))
            self.assertEqual(git("ls-tree", "--name-only", "main").split(),
                             ["FAKE_S1.md", "FAKE_S2.md", "README.md"])
            self.assertEqual(run(claude_agent.build_command(cfg, "p", "unknown", mcp)).returncode, 1)
            space_form = claude_agent.build_command(cfg, "p", None, mcp)
            index = next(i for i, arg in enumerate(space_form) if arg.startswith("--mcp-config="))
            space_form[index:index + 1] = ["--mcp-config", str(mcp)]
            self.assertEqual(run(space_form).returncode, 2)  # the real parser would take the prompt as a config


if __name__ == "__main__":
    unittest.main()
