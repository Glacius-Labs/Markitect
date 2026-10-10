"""Provider-free stand-in for the Codex and Claude reviewer CLIs.

  fake_reviewer.py --version
  fake_reviewer.py exec --json --sandbox read-only ... --output-schema S -o OUT -- PROMPT   (like codex)
  fake_reviewer.py -p --output-format json --json-schema SCHEMA ... -- PROMPT              (like claude)

It checks the read-only contract the harness promises (sandbox, tools, no settings)
and exits 9 when it is broken. It answers from the prompt: one `missed_obligation` for
the first released item that both modes report, plus one finding of its own, so
agreement is visible. `notes` says what the harness gave it: the files of its fresh
home, whether a token reached its environment, and the SHA-256 of the prompt. Like a
token refresh, the Codex mode rewrites a throwaway login `{"generation": N, ...}` in its
CODEX_HOME with N + 1.

Behavior per mode from `fake-reviewer-plan.json` in the working directory (the bundle;
the reviewer's environment is clean in the container), e.g. {"codex": "invalid"}:
invalid (schema mismatch), garbage (no JSON), fenced (JSON in a code fence), crash
(exit 3, no answer), hang (sleep), error-result (claude reports is_error).
"""
from __future__ import annotations

import hashlib
import json
import os
import re
import sys
import time
from pathlib import Path


def plan(mode: str) -> str:
    try:
        return json.loads(Path("fake-reviewer-plan.json").read_text(encoding="utf-8")).get(mode, "ok")
    except (OSError, ValueError):
        return "ok"


def value_after(args: list[str], flag: str) -> str | None:
    return args[args.index(flag) + 1] if flag in args and args.index(flag) + 1 < len(args) else None


def prompt_of(args: list[str]) -> str:
    if "--" not in args or args.index("--") != len(args) - 2:
        raise SystemExit("contract: the prompt must be the single argument after --")
    return args[-1]


def answer(mode: str, prompt: str, home: Path | None, token: bool) -> dict:
    match = re.search(r"^- Released items: (.*)$", prompt, re.M)
    items = [item.strip() for item in (match.group(1) if match else "").split(",") if item.strip()]
    first = items[0] if items else None
    files = sorted(p.name for p in home.iterdir()) if home and home.is_dir() else []
    shared = {"category": "missed_obligation", "severity": "high", "item": first, "rule": None,
              "evidence": [{"path": "README.md", "line": 1}], "detail": "shared finding"}
    own = ({"category": "contradiction", "severity": "low", "item": first, "rule": None,
            "evidence": [{"path": "app.py", "line": None}], "detail": "codex-only finding"} if mode == "codex" else
           {"category": "rule_violation", "severity": "medium", "item": None, "rule": "R1",
            "evidence": [{"path": "app.py", "line": 3}], "detail": "claude-only finding"})
    notes = (f"mode={mode} home={','.join(files) or '-'} token={'present' if token else 'absent'} "
             f"prompt={hashlib.sha256(prompt.encode('utf-8')).hexdigest()}")
    return {"findings": [shared, own], "obligations": {"covered": 1, "total": 2}, "notes": notes}


def render(value: dict, behavior: str) -> str:
    if behavior == "invalid":
        value = {"findings": [{"category": "made_up", "severity": "high"}], "notes": 1}
    if behavior == "garbage":
        return "I could not decide."
    text = json.dumps(value)
    return f"```json\n{text}\n```" if behavior == "fenced" else text


def refresh_login(home: Path | None) -> None:
    path = home / "auth.json" if home else None
    try:
        login = json.loads(path.read_text(encoding="utf-8")) if path and path.is_file() else None
    except (OSError, ValueError):
        return
    if isinstance(login, dict) and type(login.get("generation")) is int:
        path.write_text(json.dumps({**login, "generation": login["generation"] + 1}), encoding="utf-8")


def codex(args: list[str]) -> int:
    if value_after(args, "--sandbox") != "read-only" or "--output-schema" not in args or "--json" not in args:
        print("contract: read-only sandbox, --json and --output-schema are required", file=sys.stderr)
        return 9
    json.loads(Path(value_after(args, "--output-schema")).read_text(encoding="utf-8"))
    behavior = plan("codex")
    if behavior == "crash":
        print("fake codex crashed", file=sys.stderr)
        return 3
    if behavior == "hang":
        time.sleep(120)
    prompt = prompt_of(args)
    home = Path(os.environ["CODEX_HOME"]) if os.environ.get("CODEX_HOME") else None
    refresh_login(home)
    text = render(answer("codex", prompt, home, False), behavior)
    Path(value_after(args, "-o")).write_text(text, encoding="utf-8")
    for event in ({"type": "thread.started", "thread_id": "fake-review-thread"}, {"type": "turn.started"},
                  {"type": "item.completed", "item": {"id": "item_0", "type": "agent_message", "text": text}},
                  {"type": "turn.completed", "usage": {"input_tokens": 500, "cached_input_tokens": 100,
                                                       "output_tokens": 50}}):
        print(json.dumps(event), flush=True)
    return 0


def claude(args: list[str]) -> int:
    tools = value_after(args, "--tools")
    if (value_after(args, "--output-format") != "json" or tools != "Read,Grep,Glob"
            or value_after(args, "--allowedTools") != tools or value_after(args, "--setting-sources") != ""):
        print("contract: json output, read-only tools and no setting sources are required", file=sys.stderr)
        return 9
    json.loads(value_after(args, "--json-schema"))
    behavior = plan("claude")
    if behavior == "crash":
        print("fake claude crashed", file=sys.stderr)
        return 3
    if behavior == "hang":
        time.sleep(120)
    prompt = prompt_of(args)
    token = bool(os.environ.get("CLAUDE_CODE_OAUTH_TOKEN"))
    config = Path(os.environ["CLAUDE_CONFIG_DIR"]) if os.environ.get("CLAUDE_CONFIG_DIR") else None
    value = answer("claude", prompt, config, token)
    result = {"type": "result", "subtype": "success", "is_error": behavior == "error-result", "num_turns": 3,
              "result": render(value, behavior), "session_id": "fake-session", "total_cost_usd": 0.01,
              "usage": {"input_tokens": 700, "cache_read_input_tokens": 200, "cache_creation_input_tokens": 10,
                        "output_tokens": 70},
              "modelUsage": {"fake-claude-model": {"inputTokens": 700}}}
    if behavior == "ok":
        result["structured_output"] = value
    print(json.dumps(result), flush=True)
    return 1 if behavior == "error-result" else 0


def main(argv: list[str]) -> int:
    if argv == ["--version"]:
        print("fake-reviewer 1.0")
        return 0
    if argv[:1] == ["exec"]:
        return codex(argv)
    if "-p" in argv:
        return claude(argv)
    print(f"usage: {Path(__file__).name} --version | exec ... | -p ...", file=sys.stderr)
    return 2


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
