"""Run the coding agent (Codex CLI or the fake stand-in) as the unprivileged user.

Inside the container the runner is root; the agent runs as uid/gid ``agent`` with
a clean environment and its own Codex home. Outside the container ("test mode")
the agent runs as the current user so the unit tests work on any host.
"""
from __future__ import annotations

import json
import os
import re
import shutil
import signal
import subprocess
import sys
import time
from pathlib import Path

from . import lifecycle

AGENT_USER = "agent"
AGENT_HOME = Path("/home/agent")
IMAGE_PATH = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
BYPASS_FLAG = "--dangerously-bypass-approvals-and-sandbox"
# Staged next to the package: /in/playground -> /in/tests/fake_agent.py.
FAKE_AGENT = Path(__file__).resolve().parent.parent / "tests" / "fake_agent.py"


def container_mode() -> bool:
    """True when running as root on POSIX, i.e. the runner inside the container."""
    return os.name == "posix" and os.geteuid() == 0


def _agent_ids() -> tuple[int, int]:
    import pwd  # POSIX only
    entry = pwd.getpwnam(AGENT_USER)
    return entry.pw_uid, entry.pw_gid


def agent_uid() -> int | None:
    """uid of the agent user inside the container; None in test mode."""
    return _agent_ids()[0] if container_mode() else None


def give_to_agent(*paths: Path, recursive: bool = False) -> None:
    """chown (-R) to agent:agent without following symlinks; a no-op in test mode."""
    if not container_mode():
        return
    uid, gid = _agent_ids()
    for path in paths:
        os.chown(path, uid, gid, follow_symlinks=False)
        if recursive and path.is_dir() and not path.is_symlink():
            for current, dirs, files in os.walk(path):
                for name in dirs + files:
                    os.chown(os.path.join(current, name), uid, gid, follow_symlinks=False)


def install_auth(src: Path, home: Path) -> bool:
    """Copy the Codex credential file into the agent's Codex home (mode 0600)."""
    if not src.is_file():
        return False
    home.mkdir(parents=True, exist_ok=True)
    target = home / "auth.json"
    shutil.copyfile(src, target)
    os.chmod(target, 0o600)
    give_to_agent(home, target)
    return True


def _toml_str(value: str) -> str:
    escapes = {'"': '\\"', "\\": "\\\\", "\b": "\\b", "\t": "\\t",
               "\n": "\\n", "\f": "\\f", "\r": "\\r"}
    out = []
    for ch in str(value):
        if ch in escapes:
            out.append(escapes[ch])
        elif ord(ch) < 0x20 or ord(ch) == 0x7F:
            out.append(f"\\u{ord(ch):04X}")
        else:
            out.append(ch)
    return '"' + "".join(out) + '"'


def _toml_key(name: str) -> str:
    return name if re.fullmatch(r"[A-Za-z0-9_-]+", name) else _toml_str(name)


def render_config(agent_cfg: dict, mcp_servers: dict) -> str:
    """Return the minimal config.toml text for one run."""
    model, effort = agent_cfg.get("model"), agent_cfg.get("effort")
    lines = []
    if model:
        lines.append(f"model = {_toml_str(model)}")
    if effort:
        lines.append(f"model_reasoning_effort = {_toml_str(effort)}")
    lines += ["", "[features]", "memories = false", "", "[agents]"]
    if agent_cfg.get("maxSubagents") is not None:
        lines.append(f"max_concurrent_threads_per_session = {int(agent_cfg['maxSubagents'])}")
    if model:
        lines.append(f"default_subagent_model = {_toml_str(model)}")
    if effort:
        lines.append(f"default_subagent_reasoning_effort = {_toml_str(effort)}")
    for name, server in (mcp_servers or {}).items():
        args = ", ".join(_toml_str(arg) for arg in server.get("args", []))
        lines += ["", f"[mcp_servers.{_toml_key(name)}]",
                  f"command = {_toml_str(server['command'])}", f"args = [{args}]"]
    return "\n".join(lines) + "\n"


def write_config(home: Path, agent_cfg: dict, mcp_servers: dict) -> Path:
    home.mkdir(parents=True, exist_ok=True)
    path = home / "config.toml"
    path.write_text(render_config(agent_cfg, mcp_servers), encoding="utf-8", newline="\n")
    give_to_agent(home, path)
    return path


def build_command(agent_cfg: dict, prompt: str, session_id: str | None,
                  last_message: Path) -> list[str]:
    if session_id:
        args = ["exec", "resume", session_id, "--json", BYPASS_FLAG,
                "-o", str(last_message), prompt]
    else:
        args = ["exec", "--json", BYPASS_FLAG, "--skip-git-repo-check",
                "-o", str(last_message), prompt]
    kind = agent_cfg.get("kind", "codex")
    if kind == "codex":
        return ["codex", *args]
    if kind == "fake":
        return [sys.executable, str(FAKE_AGENT), *args]
    raise ValueError(f"unsupported agent kind: {kind}")


def agent_env(extra_env: dict | None = None) -> dict:
    env = {"HOME": str(AGENT_HOME), "CODEX_HOME": str(AGENT_HOME / ".codex"),
           "PATH": IMAGE_PATH, "LANG": "C.UTF-8", "PYTHONUTF8": "1", "TERM": "dumb"}
    env.update(extra_env or {})
    return env


def run_as_agent(cmd: list[str], cwd: Path, timeout: float, stdout_path: Path,
                 stderr_path: Path, extra_env: dict | None = None) -> dict:
    """Run cmd as user agent in its own session; kill it all on timeout."""
    kwargs: dict = {"cwd": cwd, "stdin": subprocess.DEVNULL}
    if container_mode():
        uid, gid = _agent_ids()
        kwargs.update(user=uid, group=gid, extra_groups=os.getgrouplist(AGENT_USER, gid),
                      env=agent_env(extra_env), start_new_session=True)
    else:
        kwargs["env"] = {**os.environ, **(extra_env or {})}
        if os.name == "posix":
            kwargs["start_new_session"] = True
        else:
            kwargs["creationflags"] = subprocess.CREATE_NEW_PROCESS_GROUP
    stdout_path.parent.mkdir(parents=True, exist_ok=True)
    stderr_path.parent.mkdir(parents=True, exist_ok=True)
    timed_out, swept = False, 0
    started = time.monotonic()
    with open(stdout_path, "wb") as out, open(stderr_path, "wb") as err:
        try:
            proc = subprocess.Popen(cmd, stdout=out, stderr=err, **kwargs)
        except OSError as exc:
            err.write(f"cannot start agent: {exc}\n".encode("utf-8"))
            return {"exitCode": None, "timedOut": False, "killedOnTimeout": 0,
                    "seconds": round(time.monotonic() - started, 3), "error": str(exc)}
        try:
            proc.wait(timeout=timeout)
        except subprocess.TimeoutExpired:
            timed_out = True
            _kill_group(proc)
            proc.wait()
            _wait_group_gone(proc.pid)
            swept = kill_all_agent_processes()  # only processes that left the group, e.g. via setsid
    return {"exitCode": proc.returncode, "timedOut": timed_out,
            "seconds": round(time.monotonic() - started, 3), "killedOnTimeout": swept}


def _kill_group(proc: subprocess.Popen) -> None:
    try:
        if os.name == "posix":
            os.killpg(proc.pid, signal.SIGKILL)
        else:
            proc.kill()
    except (ProcessLookupError, PermissionError, OSError):
        pass


def _wait_group_gone(pgid: int, timeout: float = 5) -> None:
    """Wait until the killed process group has died, so the sweep counts only escapees."""
    if not container_mode():
        return
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        alive = False
        for entry in Path("/proc").iterdir():
            if not entry.name.isdigit():
                continue
            try:
                fields = (entry / "stat").read_text(encoding="utf-8", errors="replace").rsplit(")", 1)[-1].split()
            except OSError:
                continue
            if len(fields) > 2 and fields[0] != "Z" and fields[2] == str(pgid):
                alive = True
                break
        if not alive:
            return
        time.sleep(0.05)


def _live_agent_pids(uid: int) -> list[int]:
    pids = []
    for entry in Path("/proc").iterdir():
        if not entry.name.isdigit():
            continue
        try:
            status = (entry / "status").read_text(encoding="utf-8", errors="replace")
        except OSError:
            continue  # exited meanwhile
        fields = dict(line.split(":", 1) for line in status.splitlines() if ":" in line)
        if fields.get("State", "").strip().startswith("Z"):
            continue  # already dead, waiting to be reaped
        if int(fields.get("Uid", "-1").split()[0]) == uid:
            pids.append(int(entry.name))
    return pids


def kill_all_agent_processes(timeout: float = 30) -> int:
    """SIGKILL every process of uid agent; return how many were found alive."""
    if not container_mode():
        return 0
    uid, _ = _agent_ids()
    found: set[int] = set()
    deadline = time.monotonic() + timeout
    while True:
        pids = _live_agent_pids(uid)
        if not pids:
            return len(found)
        found.update(pids)
        for pid in pids:
            try:
                os.kill(pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
        if time.monotonic() > deadline:
            print(f"warning: agent processes still alive after {timeout}s: {pids}", file=sys.stderr)
            return len(found)
        time.sleep(0.1)


ITEM_COUNTERS = {"command_execution": "commands", "file_change": "fileChanges",
                 "mcp_tool_call": "mcpToolCalls", "collab_tool_call": "collabToolCalls"}
USAGE_KEYS = {"input": "input_tokens", "cachedInput": "cached_input_tokens", "output": "output_tokens"}


def _usage(raw: dict) -> dict:
    return {key: raw.get(source) if isinstance(raw.get(source), int) else None
            for key, source in USAGE_KEYS.items()}


def parse_events(jsonl_path: Path) -> dict:
    """Summarize `codex exec --json` output; unknown events are ignored.

    Seen in real 0.162 output: thread.started (thread_id), turn.started,
    item.completed, turn.completed (usage), turn.failed, error. turn.completed
    carries the thread's cumulative usage, also after `exec resume` (checked against
    the rollout's total_token_usage), so `tokens` is the last value, not a sum.
    `items` counts completed items of every type, i.e. any activity (shell commands,
    file changes, MCP and subagent calls, messages). An MCP call with status `failed`
    counts in `mcpToolFailures` (the tool may simply have answered "no", e.g. a
    non-conforming check); one that carries an `error` could not run at all and also
    counts in `mcpCallErrors` (e.g. arguments rejected by the tool schema).
    """
    result = {"sessionId": None, "items": 0, "commands": 0, "fileChanges": 0, "mcpToolCalls": 0,
              "collabToolCalls": 0, "errors": 0, "malformedLines": 0, "mcpToolFailures": 0,
              "mcpCallErrors": 0, "tokens": {"input": None, "cachedInput": None, "output": None}}
    try:
        text = Path(jsonl_path).read_text(encoding="utf-8", errors="replace")
    except OSError:
        return result
    for line in text.splitlines():
        if not line.strip():
            continue
        try:
            event = json.loads(line)
        except ValueError:
            event = None
        if not isinstance(event, dict):
            result["malformedLines"] += 1
            continue
        kind = event.get("type")
        if kind == "thread.started" and result["sessionId"] is None:
            if isinstance(event.get("thread_id"), str):
                result["sessionId"] = event["thread_id"]
        elif kind == "turn.completed" and isinstance(event.get("usage"), dict):
            result["tokens"] = _usage(event["usage"])
        elif kind in ("turn.failed", "error"):
            result["errors"] += 1
        elif kind == "item.completed" and isinstance(event.get("item"), dict):
            result["items"] += 1
            item_type = event["item"].get("type")
            if item_type in ITEM_COUNTERS:
                result[ITEM_COUNTERS[item_type]] += 1
                if item_type == "mcp_tool_call" and event["item"].get("status") == "failed":
                    result["mcpToolFailures"] += 1
                    result["mcpCallErrors"] += 1 if event["item"].get("error") else 0
            elif item_type == "error":
                result["errors"] += 1
    return result


def session_totals(sessions: Path) -> dict[str, dict]:
    """Latest cumulative token usage per Codex thread, from the session records
    (`rollout-*.jsonl`, `token_count` events) under `sessions`.

    Subagents and Markitect's inner App Server roles write their own records there;
    ephemeral sessions write none, so the sum is a lower bound.
    """
    totals: dict[str, dict] = {}
    for current, _dirs, names in os.walk(sessions):
        for name in sorted(names):
            if not (name.startswith("rollout-") and name.endswith(".jsonl")):
                continue
            handle = lifecycle.open_plain(Path(current) / name)  # only numbers are read
            if handle is None:
                continue
            thread, usage = None, None
            with handle:
                for raw in handle:
                    if b"session_meta" not in raw and b"token_count" not in raw:
                        continue
                    try:
                        event = json.loads(raw)
                    except ValueError:
                        continue
                    payload = event.get("payload") if isinstance(event, dict) else None
                    if not isinstance(payload, dict):
                        continue
                    if event.get("type") == "session_meta" and isinstance(payload.get("id"), str):
                        thread = thread or payload["id"]
                    elif event.get("type") == "event_msg" and payload.get("type") == "token_count":
                        info = payload.get("info")
                        total = info.get("total_token_usage") if isinstance(info, dict) else None
                        if isinstance(total, dict):
                            usage = _usage(total)
            key = thread or name
            if usage is not None and (key not in totals or (usage["input"] or 0) >= (totals[key]["input"] or 0)):
                totals[key] = usage
    return totals
