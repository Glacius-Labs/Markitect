"""Claude Code (or its fake stand-in) as the outer agent.

Processes run through `codex_agent.run_as_agent` (user agent, clean environment,
kill-all), so this module only knows what is specific to Claude Code:

- the command: `claude -p --output-format stream-json --verbose --model M --effort E
  --dangerously-skip-permissions [--resume SESSION] --mcp-config=FILE --strict-mcp-config -- PROMPT`;
- a fresh CLAUDE_CONFIG_DIR under the agent home that holds only the MCP config the
  harness writes (Claude Code adds its own state and session transcripts there);
- the OAuth token from the operator's token file. It goes only into the environment of
  the claude process (`CLAUDE_CODE_OAUTH_TOKEN`), never into a command line or record,
  and `redact_tree` removes it from result files should the agent print it.
"""
from __future__ import annotations

import json
import os
import sys
from pathlib import Path

from . import codex_agent, lifecycle

CONFIG_DIR = ".claude"  # CLAUDE_CONFIG_DIR, relative to the agent home
MCP_CONFIG = "mcp-config.json"
TOKEN_ENV = "CLAUDE_CODE_OAUTH_TOKEN"
TOKEN_LIMIT = 64 << 10
# One line Claude Code reads as an import of AGENTS.md; added when a repo has no CLAUDE.md.
ROUTER = "@AGENTS.md\n"
# The image pins the version; no memory across waves, like Codex's `features.memories = false`.
ENV = {"DISABLE_AUTOUPDATER": "1", "CLAUDE_CODE_DISABLE_AUTO_MEMORY": "1"}
SKIP_PERMISSIONS = "--dangerously-skip-permissions"
REDACTED = b"[redacted:" + TOKEN_ENV.encode("ascii") + b"]"
SESSION_PREFIX = "claude:"  # keeps transcript keys apart from Codex thread ids
SYNTHETIC_MODEL = "<synthetic>"  # Claude Code's own messages (e.g. a rejected login), no model call
SUBAGENT_TOOLS = ("Agent", "Task")
EDIT_TOOLS = ("Edit", "Write", "MultiEdit", "NotebookEdit")
TOKEN_KINDS = ("input", "cachedInput", "output")
# Staged next to the package: /in/playground -> /in/tests/fake_claude.py.
FAKE_CLAUDE = Path(__file__).resolve().parent.parent / "tests" / "fake_claude.py"


def environment(config_dir: Path) -> dict:
    """What the claude process needs beyond the common agent environment (no token)."""
    return {"CLAUDE_CONFIG_DIR": str(config_dir), **ENV}


def read_token(src: Path) -> str | None:
    """The token from the operator's file, or None when it is missing, empty or not one
    word. The value is only ever handed to the claude process environment."""
    handle = lifecycle.open_plain(Path(src))
    if handle is None:
        return None
    with handle:
        data = handle.read(TOKEN_LIMIT + 1)
    token = data.decode("utf-8", errors="replace").strip()
    if not token or len(data) > TOKEN_LIMIT or any(ch.isspace() for ch in token):
        return None
    return token


def write_config(config_dir: Path, mcp_servers: dict) -> Path:
    """Write the MCP configuration loaded with --strict-mcp-config (always, possibly without
    servers, so no other MCP configuration is used); return its path."""
    config_dir.mkdir(parents=True, exist_ok=True)
    servers = {name: {"type": "stdio", "command": server["command"], "args": list(server.get("args", []))}
               for name, server in (mcp_servers or {}).items()}
    path = config_dir / MCP_CONFIG
    path.write_text(json.dumps({"mcpServers": servers}, ensure_ascii=False, indent=2) + "\n",
                    encoding="utf-8", newline="\n")
    codex_agent.give_to_agent(config_dir, path)
    return path


def build_command(agent_cfg: dict, prompt: str, session_id: str | None, mcp_config: Path) -> list[str]:
    args = ["-p", "--output-format", "stream-json", "--verbose", "--model", agent_cfg["model"],
            "--effort", agent_cfg["effort"], SKIP_PERMISSIONS]
    if session_id:
        args += ["--resume", session_id]
    # --mcp-config takes several values; the `=` form and `--` keep it from taking the prompt.
    args += [f"--mcp-config={mcp_config}", "--strict-mcp-config", "--", prompt]
    kind = agent_cfg.get("kind")
    if kind == "claude":
        return ["claude", *args]
    if kind == "fake-claude":
        return [sys.executable, str(FAKE_CLAUDE), *args]
    raise ValueError(f"unsupported agent kind: {kind}")


def session_key(session_id: str) -> str:
    return SESSION_PREFIX + session_id


# --- usage ---------------------------------------------------------------------------

def _int(value) -> int | None:
    return value if isinstance(value, int) and not isinstance(value, bool) else None


def usage(raw) -> dict | None:
    """Anthropic usage in the harness's kinds: input counts fresh input, cache writes and
    cache reads (like Codex's input_tokens), cached input the cache reads."""
    if not isinstance(raw, dict):
        return None
    fresh, out = _int(raw.get("input_tokens")), _int(raw.get("output_tokens"))
    if fresh is None or out is None:
        return None
    written = _int(raw.get("cache_creation_input_tokens")) or 0
    read = _int(raw.get("cache_read_input_tokens")) or 0
    return {"input": fresh + written + read, "cachedInput": read, "output": out}


def _model_usage(raw) -> dict | None:
    """Sum of the result message's `modelUsage` (every model this process called)."""
    if not isinstance(raw, dict) or not raw:
        return None
    total = dict.fromkeys(TOKEN_KINDS, 0)
    for entry in raw.values():
        mapped = usage({"input_tokens": entry.get("inputTokens"), "output_tokens": entry.get("outputTokens"),
                        "cache_creation_input_tokens": entry.get("cacheCreationInputTokens"),
                        "cache_read_input_tokens": entry.get("cacheReadInputTokens")}) \
            if isinstance(entry, dict) else None
        if mapped is None:
            return None
        total = {kind: total[kind] + mapped[kind] for kind in TOKEN_KINDS}
    return total


def _keep(messages: dict, key: str, mapped: dict) -> None:
    """Claude Code repeats one API message per content block; count it once (largest values)."""
    old = messages.get(key)
    messages[key] = mapped if old is None else {kind: max(old[kind], mapped[kind]) for kind in TOKEN_KINDS}


def _sum(messages: dict) -> dict | None:
    if not messages:
        return None
    return {kind: sum(item[kind] for item in messages.values()) for kind in TOKEN_KINDS}


def _text(content) -> str:
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        return "\n".join(block.get("text", "") for block in content
                         if isinstance(block, dict) and isinstance(block.get("text"), str))
    return ""


# --- events ----------------------------------------------------------------------------

def parse_events(jsonl_path: Path) -> dict:
    """Summarize `claude -p --output-format stream-json --verbose` output; same keys as
    `codex_agent.parse_events` plus the result message.

    Every message carries `session_id`; tool calls are `tool_use` blocks of assistant
    messages and their outcomes `tool_result` blocks of user messages. Counters cover the
    outer agent (messages without `parent_tool_use_id`); MCP failures count at every
    level. A failed MCP call carries `is_error`; one the MCP client could not run at all
    also says "MCP error" (e.g. arguments rejected by the tool schema). `tokens` is this
    wave's outer usage from the stream (each message id once). Synthetic messages (no
    model call, e.g. a rejected login) count as no activity; `apiRetries` counts the
    CLI's own retries of failed provider requests.
    """
    nothing = dict.fromkeys(TOKEN_KINDS)
    result = {"sessionId": None, "items": 0, "commands": 0, "fileChanges": 0, "mcpToolCalls": 0,
              "collabToolCalls": 0, "errors": 0, "malformedLines": 0, "mcpToolFailures": 0,
              "mcpCallErrors": 0, "apiRetries": 0, "tokens": dict(nothing), "result": None}
    try:
        text = Path(jsonl_path).read_text(encoding="utf-8", errors="replace")
    except OSError:
        return result
    messages: dict[str, dict] = {}
    mcp_ids: set[str] = set()
    for index, line in enumerate(text.splitlines()):
        if not line.strip():
            continue
        try:
            event = json.loads(line)
        except ValueError:
            event = None
        if not isinstance(event, dict):
            result["malformedLines"] += 1
            continue
        if result["sessionId"] is None and isinstance(event.get("session_id"), str) and event["session_id"]:
            result["sessionId"] = event["session_id"]
        kind = event.get("type")
        message = event.get("message") if isinstance(event.get("message"), dict) else {}
        content = message.get("content") if isinstance(message.get("content"), list) else []
        outer = event.get("parent_tool_use_id") is None
        if kind == "assistant" and message.get("model") != SYNTHETIC_MODEL:
            for block in content:
                if not isinstance(block, dict):
                    continue
                name = block.get("name") if isinstance(block.get("name"), str) else ""
                if block.get("type") == "tool_use" and name.startswith("mcp__"):
                    mcp_ids.add(block.get("id"))
                if not outer or block.get("type") not in ("tool_use", "text"):
                    continue
                result["items"] += 1
                if block.get("type") != "tool_use":
                    continue
                if name == "Bash":
                    result["commands"] += 1
                elif name in EDIT_TOOLS:
                    result["fileChanges"] += 1
                elif name.startswith("mcp__"):
                    result["mcpToolCalls"] += 1
                elif name in SUBAGENT_TOOLS:
                    result["collabToolCalls"] += 1
            mapped = usage(message.get("usage")) if outer else None
            if mapped is not None:
                _keep(messages, str(message.get("id") or f"line-{index}"), mapped)
        elif kind == "system" and event.get("subtype") == "api_retry":
            result["apiRetries"] += 1
        elif kind == "user":
            for block in content:
                if (isinstance(block, dict) and block.get("type") == "tool_result"
                        and block.get("is_error") is True and block.get("tool_use_id") in mcp_ids):
                    result["mcpToolFailures"] += 1
                    result["mcpCallErrors"] += 1 if "MCP error" in _text(block.get("content")) else 0
        elif kind == "result":
            failed = event.get("is_error") is True or str(event.get("subtype", "")).startswith("error")
            result["errors"] += 1 if failed else 0
            cost = event.get("total_cost_usd")
            result["result"] = {
                "subtype": event.get("subtype"), "isError": failed, "numTurns": _int(event.get("num_turns")),
                "costUsd": cost if isinstance(cost, (int, float)) and not isinstance(cost, bool) else None,
                "tokens": usage(event.get("usage")) or dict(nothing),
                "tokensAllModels": _model_usage(event.get("modelUsage")) or dict(nothing),
            }
    result["tokens"] = _sum(messages) or dict(nothing)
    return result


def last_message(jsonl_path: Path) -> str | None:
    """The final answer: the result message's text, else the last outer assistant text."""
    found = None
    try:
        lines = Path(jsonl_path).read_text(encoding="utf-8", errors="replace").splitlines()
    except OSError:
        return None
    for line in lines:
        try:
            event = json.loads(line)
        except ValueError:
            continue
        if not isinstance(event, dict):
            continue
        if event.get("type") == "result" and isinstance(event.get("result"), str):
            found = event["result"]
        elif event.get("type") == "assistant" and event.get("parent_tool_use_id") is None:
            message = event.get("message") if isinstance(event.get("message"), dict) else {}
            if message.get("model") != SYNTHETIC_MODEL:
                found = _text(message.get("content")) or found
    return found


def save_last_message(jsonl_path: Path, target: Path, limit: int) -> None:
    text = last_message(jsonl_path)
    if text is not None:
        target.write_bytes(text.encode("utf-8")[:limit])


def session_totals(projects: Path) -> dict[str, dict]:
    """Token totals per transcript under CLAUDE_CONFIG_DIR/projects: the main session
    `<project>/<session>.jsonl` and its subagents `<project>/<session>/subagents/agent-*.jsonl`
    (sidechain entries). Keys are `session_key(<file stem>)`, with `:sidechain` for
    sidechain entries. Transcripts repeat one API message per content block, so each
    message id counts once. Subagent transcripts can hold partial output counts, so the
    totals are a lower bound."""
    totals: dict[str, dict] = {}
    for current, _dirs, names in os.walk(projects):
        for name in sorted(names):
            if not name.endswith(".jsonl"):
                continue
            handle = lifecycle.open_plain(Path(current) / name)  # only numbers are read
            if handle is None:
                continue
            main: dict[str, dict] = {}
            side: dict[str, dict] = {}
            with handle:
                for index, raw in enumerate(handle):
                    if b'"usage"' not in raw:
                        continue
                    try:
                        entry = json.loads(raw)
                    except ValueError:
                        continue
                    if not isinstance(entry, dict) or entry.get("type") != "assistant":
                        continue
                    message = entry.get("message")
                    if not isinstance(message, dict) or message.get("model") == SYNTHETIC_MODEL:
                        continue
                    mapped = usage(message.get("usage"))
                    if mapped is not None:
                        _keep(side if entry.get("isSidechain") is True else main,
                              str(message.get("id") or f"line-{index}"), mapped)
            stem = session_key(name[:-len(".jsonl")])
            for key, found in ((stem, main), (stem + ":sidechain", side)):
                total = _sum(found)
                if total is not None:
                    totals[key] = total
    return totals


# --- secrets ---------------------------------------------------------------------------

def redact_tree(root: Path, secret: str | None, *, skip: tuple[str, ...] = ()) -> int:
    """Replace `secret` in every regular file under `root` (links are never followed;
    top-level folders named in `skip` are left out); return how many files changed."""
    root = Path(root)
    if not secret or not root.is_dir():
        return 0
    needle, changed = secret.encode("utf-8"), 0
    for current, dirs, names in os.walk(root):
        if Path(current) == root:
            dirs[:] = [name for name in dirs if name not in skip]
        for name in names:
            path = Path(current) / name
            if path.is_symlink() or not path.is_file():
                continue
            try:
                data = path.read_bytes()
                if needle in data:
                    path.write_bytes(data.replace(needle, REDACTED))
                    changed += 1
            except OSError:
                continue
    return changed
