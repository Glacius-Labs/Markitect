"""Deterministic, provider-free stand-in for `claude -p --output-format stream-json`.

Accepts exactly the argv the harness builds (`claude_agent.build_command`):
  fake_claude.py -p --output-format stream-json --verbose --model M --effort E
      --dangerously-skip-permissions [--resume SESSION] --mcp-config=FILE --strict-mcp-config -- PROMPT
and rejects anything else (exit 2), as the real option parser would. For the released
station it branches, writes FAKE_S<n>.md, commits and merges to main (like
fake_agent.py) and prints stream-json messages shaped like Claude Code's: a system init,
assistant messages (one line per content block, repeating the message id and usage),
tool results, one subagent call and a result message with `usage` and `modelUsage`.
It keeps transcripts under $CLAUDE_CONFIG_DIR/projects/<cwd>/<session>.jsonl plus one
subagent transcript per station; a resume keeps the session id. On POSIX it leaves a
detached `sleep 600` behind so orphan cleanup is visible.

Per wave the outer messages use 1200 input tokens (800 of them cache reads) and 150
output tokens, the subagent 100 and 10, the same numbers as fake_agent.py.

When CLAUDE_CODE_OAUTH_TOKEN is set, the init message says only `fakeTokenSeen: true`,
and one tool result then echoes the token the way an agent running `env` would, so the
harness's redaction is exercised end to end.

Optional per-station behavior: $CLAUDE_CONFIG_DIR/fake-plan.json, e.g. {"S2": "fail-after-work"}.
Modes: crlf, fail-after-work, timeout (as in fake_agent.py) and login-error (no work: a
synthetic assistant message and an error result, as with a rejected token).
"""
from __future__ import annotations

import json
import os
import re
import subprocess
import sys
import time
import uuid
from pathlib import Path

GIT = ["git", "-c", "user.name=Fake Claude", "-c", "user.email=fake-claude@playground.invalid",
       "-c", "commit.gpgsign=false"]
TOKEN_ENV = "CLAUDE_CODE_OAUTH_TOKEN"
VALUE_FLAGS = ("--output-format", "--model", "--effort", "--resume")
BOOL_FLAGS = ("-p", "--verbose", "--dangerously-skip-permissions", "--strict-mcp-config")
# Two outer API messages per wave; together 100 + 300 + 800 = 1200 input, 800 cached, 150 output.
WORK_USAGE = {"input_tokens": 50, "cache_creation_input_tokens": 150, "cache_read_input_tokens": 400,
              "output_tokens": 100}
FINAL_USAGE = {"input_tokens": 50, "cache_creation_input_tokens": 150, "cache_read_input_tokens": 400,
               "output_tokens": 50}
SUBAGENT_USAGE = {"input_tokens": 100, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0,
                  "output_tokens": 10}
SYNTHETIC_USAGE = {"input_tokens": 0, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0,
                   "output_tokens": 0}


def usage_error(message: str) -> None:
    print(f"error: {message}", file=sys.stderr)
    raise SystemExit(2)


def parse_args(argv: list[str]) -> dict:
    options: dict = {"flags": set(), "mcpConfig": None}
    index = 0
    while index < len(argv) and argv[index] != "--":
        arg = argv[index]
        if arg in BOOL_FLAGS:
            options["flags"].add(arg)
        elif arg in VALUE_FLAGS and index + 1 < len(argv):
            options[arg] = argv[index + 1]
            index += 1
        elif arg.startswith("--mcp-config="):
            options["mcpConfig"] = arg.split("=", 1)[1]
        else:
            usage_error(f"unexpected argument {arg!r}")
        index += 1
    prompt = argv[index + 1:]
    if len(prompt) != 1 or not prompt[0]:
        usage_error("expected exactly one prompt after --")
    if set(BOOL_FLAGS) - {"--strict-mcp-config"} - options["flags"]:
        usage_error("-p, --verbose and --dangerously-skip-permissions are required")
    if options.get("--output-format") != "stream-json" or not options.get("--model") or not options.get("--effort"):
        usage_error("--output-format stream-json, --model and --effort are required")
    return {"session": options.get("--resume"), "model": options["--model"], "prompt": prompt[0],
            "mcpConfig": options["mcpConfig"], "strictMcp": "--strict-mcp-config" in options["flags"]}


class Stream:
    """Prints stream-json lines and mirrors assistant/user messages into the transcript."""

    def __init__(self, session: str, transcript: Path | None) -> None:
        self.session, self.transcript = session, transcript

    def emit(self, event: dict, *, record: bool = False) -> None:
        print(json.dumps(event, ensure_ascii=False), flush=True)
        if record and self.transcript is not None:
            self.transcript.parent.mkdir(parents=True, exist_ok=True)
            line = {**event, "sessionId": self.session, "isSidechain": False}
            with self.transcript.open("a", encoding="utf-8") as handle:
                handle.write(json.dumps(line, ensure_ascii=False) + "\n")

    def assistant(self, message_id: str, model: str, block: dict, usage: dict) -> None:
        message = {"id": message_id, "type": "message", "role": "assistant", "model": model,
                   "content": [block], "usage": usage}
        self.emit({"type": "assistant", "message": message, "parent_tool_use_id": None,
                   "session_id": self.session}, record=True)

    def tool_result(self, tool_use_id: str, content: str, is_error: bool = False) -> None:
        block = {"type": "tool_result", "tool_use_id": tool_use_id, "content": content, "is_error": is_error}
        self.emit({"type": "user", "message": {"role": "user", "content": [block]},
                   "parent_tool_use_id": None, "session_id": self.session}, record=True)

    def tool(self, message_id: str, model: str, name: str, tool_input: dict, output: str,
             is_error: bool = False) -> None:
        tool_id = f"toolu_{uuid.uuid4().hex[:12]}"
        self.assistant(message_id, model, {"type": "tool_use", "id": tool_id, "name": name, "input": tool_input},
                       WORK_USAGE)
        self.tool_result(tool_id, output, is_error)

    def result(self, *, text: str, is_error: bool, subtype: str, usage: dict, model_usage: dict,
               turns: int) -> None:
        self.emit({"type": "result", "subtype": subtype, "is_error": is_error, "num_turns": turns,
                   "result": text, "session_id": self.session, "total_cost_usd": 0, "usage": usage,
                   "modelUsage": model_usage})


def git(stream: Stream, message_id: str, model: str, *args: str) -> None:
    done = subprocess.run([*GIT, *args], capture_output=True, text=True, encoding="utf-8")
    stream.tool(message_id, model, "Bash", {"command": "git " + " ".join(args)},
                done.stdout + done.stderr, is_error=done.returncode != 0)
    if done.returncode != 0:
        raise RuntimeError(f"git {' '.join(args)} failed: {done.stderr.strip()}")


def write_subagent(transcript: Path | None, session: str, model: str) -> None:
    if transcript is None:
        return
    path = transcript.with_suffix("") / "subagents" / f"agent-{uuid.uuid4().hex[:16]}.jsonl"
    path.parent.mkdir(parents=True, exist_ok=True)
    message = {"id": f"msg_{uuid.uuid4().hex[:20]}", "type": "message", "role": "assistant", "model": model,
               "content": [{"type": "text", "text": "Reviewed."}], "usage": SUBAGENT_USAGE}
    path.write_text(json.dumps({"type": "assistant", "sessionId": session, "isSidechain": True,
                                "message": message}) + "\n", encoding="utf-8")


def leave_orphan() -> None:
    # FAKE_AGENT_NO_ORPHAN lets unit tests run the fake without leaking a process.
    if os.name == "posix" and not os.environ.get("FAKE_AGENT_NO_ORPHAN"):
        subprocess.Popen(["sleep", "600"], start_new_session=True, stdin=subprocess.DEVNULL,
                         stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


def total(*usages: dict) -> dict:
    return {key: sum(item[key] for item in usages) for key in usages[0]}


def model_usage(**by_model: dict) -> dict:
    return {model: {"inputTokens": u["input_tokens"], "outputTokens": u["output_tokens"],
                    "cacheReadInputTokens": u["cache_read_input_tokens"],
                    "cacheCreationInputTokens": u["cache_creation_input_tokens"], "costUSD": 0}
            for model, u in by_model.items()}


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    config = os.environ.get("CLAUDE_CONFIG_DIR")
    home = Path(config) if config else None  # only a config dir the runner created
    session = args["session"] or str(uuid.uuid4())
    state = home / "fake-sessions" / f"{session}.json" if home else None
    if args["session"] and (state is None or not state.is_file()):
        print(f"No conversation found with session ID: {session}", file=sys.stderr)
        return 1
    servers = {}
    if args["mcpConfig"]:
        servers = json.loads(Path(args["mcpConfig"]).read_text(encoding="utf-8")).get("mcpServers") or {}
    slug = re.sub(r"[^A-Za-z0-9]", "-", str(Path.cwd()))
    transcript = home / "projects" / slug / f"{session}.jsonl" if home else None
    stream = Stream(session, transcript)
    token = os.environ.get(TOKEN_ENV, "")
    stream.emit({"type": "system", "subtype": "init", "session_id": session, "cwd": str(Path.cwd()),
                 "model": args["model"], "permissionMode": "bypassPermissions", "apiKeySource": "none",
                 "mcp_servers": [{"name": name, "status": "connected"} for name in sorted(servers)],
                 "fakeTokenSeen": bool(token)})
    record = json.loads(Path(".study/station.json").read_text(encoding="utf-8"))
    station = str(record.get("station") or record.get("id"))
    plan = json.loads((home / "fake-plan.json").read_text(encoding="utf-8")) \
        if home and (home / "fake-plan.json").is_file() else {}
    mode = plan.get(station, "")
    model = args["model"]
    if mode == "login-error":
        message = "Invalid API key · Please run /login"
        stream.assistant(f"msg_{uuid.uuid4().hex[:20]}", "<synthetic>", {"type": "text", "text": message},
                         SYNTHETIC_USAGE)
        stream.result(text=message, is_error=True, subtype="success", usage=SYNTHETIC_USAGE, model_usage={},
                      turns=1)
        return 1
    if state is not None:
        state.parent.mkdir(parents=True, exist_ok=True)
        state.write_text(json.dumps({"session": session}), encoding="utf-8")
    work = f"msg_{uuid.uuid4().hex[:20]}"
    number = station.removeprefix("S")
    branch, name = f"fake/s{number}", f"FAKE_S{number}.md"
    try:
        if token:  # what an agent running `env` would leave in its own output
            stream.tool(work, model, "Bash", {"command": "env | grep CLAUDE_CODE"}, f"{TOKEN_ENV}={token}\n")
        git(stream, work, model, "checkout", "-b", branch, "main")
        items = ", ".join(record.get("items", []))
        Path(name).write_text(f"# Fake work for {station}\n\nItems: {items}\n", encoding="utf-8", newline="\n")
        stream.tool(work, model, "Write", {"file_path": name}, "File created")
        git(stream, work, model, "add", name)
        if mode == "crlf":
            Path(".gitattributes").write_text("*.md text eol=crlf\n", encoding="utf-8", newline="\n")
            git(stream, work, model, "add", ".gitattributes")
        git(stream, work, model, "commit", "-m", f"Fake work for {station}")
        stream.tool(work, model, "Agent", {"description": "review", "prompt": "Review the change."}, "Reviewed.")
        write_subagent(transcript, session, "fake-subagent")
        git(stream, work, model, "checkout", "main")
        git(stream, work, model, "merge", "--no-ff", "-m", f"Merge {branch}", branch)
    except (OSError, ValueError, RuntimeError) as exc:
        stream.result(text=str(exc), is_error=True, subtype="error_during_execution", usage=WORK_USAGE,
                      model_usage=model_usage(**{model: WORK_USAGE}), turns=1)
        return 1
    outer = total(WORK_USAGE, FINAL_USAGE)
    models = model_usage(**{model: outer, "fake-subagent": SUBAGENT_USAGE})
    if mode == "fail-after-work":
        stream.result(text="simulated provider error", is_error=True, subtype="error_during_execution",
                      usage=WORK_USAGE, model_usage=models, turns=2)
        return 1
    if mode == "timeout":
        time.sleep(600)
    message = f"Merged {name} for {station} into main."
    stream.assistant(f"msg_{uuid.uuid4().hex[:20]}", model, {"type": "text", "text": message}, FINAL_USAGE)
    leave_orphan()
    stream.result(text=message, is_error=False, subtype="success", usage=outer, model_usage=models, turns=2)
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
