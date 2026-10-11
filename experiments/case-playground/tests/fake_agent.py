"""Deterministic, provider-free stand-in for `codex exec --json`.

Accepts the same argv shape as the real CLI:
  fake_agent.py exec --json <flags> -o LAST_MESSAGE PROMPT
  fake_agent.py exec resume SESSION_ID --json <flags> -o LAST_MESSAGE PROMPT
For the released station it branches, writes FAKE_S<n>.md, commits and merges to
main, prints JSONL events shaped like Codex's, and writes the last message. Like the
real CLI it reports the thread's cumulative usage in turn.completed (also after a
resume) and keeps session records (`rollout-*.jsonl` with `token_count` events) in
its Codex home, plus one helper session per station (like a subagent). On POSIX it
also leaves a detached `sleep 600` behind so orphan cleanup is visible.

Optional per-station behavior for tests: `$CODEX_HOME/fake-plan.json`, e.g.
{"S2": "fail-after-work"}. Modes: crlf (also merges a `.gitattributes` with
`*.md text eol=crlf`), fail-after-work (exit 1 with turn.failed after merging),
no-thread (resume without a thread.started event), timeout (hang after merging).
turn.started carries `fakeResultsVisible`: whether the agent can open /out (None
outside the container), and `fakeLoginGeneration`: like a token refresh, a throwaway
Codex login `{"generation": N, ...}` in its Codex home is rewritten with N + 1 on every
call, and the event names the N it found (None without such a login).
"""
from __future__ import annotations

import json
import os
import subprocess
import sys
import time
import uuid
from pathlib import Path

GIT = ["git", "-c", "user.name=Fake Agent", "-c", "user.email=fake-agent@playground.invalid",
       "-c", "commit.gpgsign=false"]
TURN_USAGE = {"input_tokens": 1200, "cached_input_tokens": 800, "output_tokens": 150,
              "reasoning_output_tokens": 40}
HELPER_USAGE = {"input_tokens": 100, "cached_input_tokens": 0, "output_tokens": 10,
                "reasoning_output_tokens": 0}


class Events:
    def __init__(self) -> None:
        self.count = 0

    def emit(self, event: dict) -> None:
        print(json.dumps(event, ensure_ascii=False), flush=True)

    def item(self, item: dict) -> None:
        item = {"id": f"item_{self.count}", **item}
        self.count += 1
        self.emit({"type": "item.completed", "item": item})


def parse_args(argv: list[str]) -> dict:
    if not argv or argv[0] != "exec":
        raise SystemExit("usage: fake_agent.py exec [resume SESSION_ID] [flags] -o FILE PROMPT")
    rest, session = argv[1:], None
    if rest and rest[0] == "resume":
        if len(rest) < 2:
            raise SystemExit("resume needs a session id")
        session, rest = rest[1], rest[2:]
    last_message, positional, index = None, [], 0
    while index < len(rest):
        arg = rest[index]
        if arg in ("-o", "--output-last-message"):
            last_message = rest[index + 1]
            index += 2
            continue
        if not arg.startswith("-"):
            positional.append(arg)
        index += 1
    if len(positional) != 1 or last_message is None:
        raise SystemExit("expected exactly one prompt and -o FILE")
    return {"session": session, "lastMessage": Path(last_message), "prompt": positional[0]}


def codex_home() -> Path | None:
    # Only inside a Codex home the runner created; never the user's real ~/.codex.
    home = os.environ.get("CODEX_HOME")
    return Path(home) if home else None


def add_usage(home: Path | None, thread: str, usage: dict) -> dict:
    """Add `usage` to the thread's running total, append a session record, return the total."""
    total = dict.fromkeys(usage, 0)
    if home is None:
        return {key: total[key] + value for key, value in usage.items()}
    state = home / "fake-sessions" / f"{thread}.json"
    if state.is_file():
        total.update(json.loads(state.read_text(encoding="utf-8")))
    total = {key: total.get(key, 0) + value for key, value in usage.items()}
    state.parent.mkdir(parents=True, exist_ok=True)
    state.write_text(json.dumps(total), encoding="utf-8")
    record = home / "sessions" / "fake" / f"rollout-{thread}.jsonl"
    record.parent.mkdir(parents=True, exist_ok=True)
    lines = [] if record.exists() else [{"type": "session_meta", "payload": {"id": thread, "source": "exec"}}]
    lines.append({"type": "event_msg", "payload": {"type": "token_count", "info": {"total_token_usage": total}}})
    with record.open("a", encoding="utf-8") as handle:
        handle.writelines(json.dumps(line) + "\n" for line in lines)
    return total


def refresh_login(home: Path | None) -> int | None:
    """Bump the generation of a throwaway login (a smoke test of the study's login hand-over)."""
    path = home / "auth.json" if home else None
    try:
        login = json.loads(path.read_text(encoding="utf-8")) if path and path.is_file() else None
    except (OSError, ValueError):
        return None
    if not isinstance(login, dict) or type(login.get("generation")) is not int:
        return None
    seen = login["generation"]
    path.write_text(json.dumps({**login, "generation": seen + 1}), encoding="utf-8")
    return seen


def git(events: Events, *args: str) -> None:
    done = subprocess.run([*GIT, *args], capture_output=True, text=True, encoding="utf-8")
    events.item({"type": "command_execution", "command": "git " + " ".join(args),
                 "aggregated_output": done.stdout + done.stderr, "exit_code": done.returncode,
                 "status": "completed" if done.returncode == 0 else "failed"})
    if done.returncode != 0:
        raise RuntimeError(f"git {' '.join(args)} failed: {done.stderr.strip()}")


def leave_orphan() -> None:
    # FAKE_AGENT_NO_ORPHAN lets unit tests run the fake without leaking a process.
    if os.name == "posix" and not os.environ.get("FAKE_AGENT_NO_ORPHAN"):
        subprocess.Popen(["sleep", "600"], start_new_session=True, stdin=subprocess.DEVNULL,
                         stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    events = Events()
    home = codex_home()
    session = args["session"] or str(uuid.uuid4())
    if args["session"] and home and not (home / "fake-sessions" / f"{session}.json").is_file():
        events.emit({"type": "error", "message": f"session not found: {session}"})
        return 1
    record = json.loads(Path(".study/station.json").read_text(encoding="utf-8"))
    station = str(record.get("station") or record.get("id"))
    plan = json.loads((home / "fake-plan.json").read_text(encoding="utf-8")) \
        if home and (home / "fake-plan.json").is_file() else {}
    mode = plan.get(station, "")
    if not (mode == "no-thread" and args["session"]):
        events.emit({"type": "thread.started", "thread_id": session})
    # Inside the container the agent must not reach the results under /out.
    visible = os.access("/out", os.R_OK | os.X_OK) if os.path.isdir("/out") else None
    events.emit({"type": "turn.started", "fakeResultsVisible": visible, "fakeLoginGeneration": refresh_login(home)})
    try:
        number = station.removeprefix("S")
        branch, name = f"fake/s{number}", f"FAKE_S{number}.md"
        git(events, "checkout", "-b", branch, "main")
        items = ", ".join(record.get("items", []))
        Path(name).write_text(f"# Fake work for {station}\n\nItems: {items}\n", encoding="utf-8", newline="\n")
        events.item({"type": "file_change", "changes": [{"path": name, "kind": "add"}], "status": "completed"})
        git(events, "add", name)
        if mode == "crlf":
            Path(".gitattributes").write_text("*.md text eol=crlf\n", encoding="utf-8", newline="\n")
            git(events, "add", ".gitattributes")
        git(events, "commit", "-m", f"Fake work for {station}")
        git(events, "checkout", "main")
        git(events, "merge", "--no-ff", "-m", f"Merge {branch}", branch)
    except (OSError, ValueError, RuntimeError) as exc:
        events.emit({"type": "error", "message": str(exc)})
        events.emit({"type": "turn.failed", "error": {"message": str(exc)}})
        return 1
    total = add_usage(home, session, TURN_USAGE)
    add_usage(home, str(uuid.uuid4()), HELPER_USAGE)
    if mode == "fail-after-work":
        events.emit({"type": "turn.failed", "error": {"message": "simulated provider error"}})
        return 1
    if mode == "timeout":
        time.sleep(600)
    message = f"Merged {name} for {station} into main."
    events.item({"type": "agent_message", "text": message})
    args["lastMessage"].parent.mkdir(parents=True, exist_ok=True)
    args["lastMessage"].write_text(message + "\n", encoding="utf-8")
    leave_orphan()
    events.emit({"type": "turn.completed", "usage": total})
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
