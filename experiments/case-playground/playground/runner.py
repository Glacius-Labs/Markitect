"""Container side: one trajectory (setup, stations S1..SN, freeze, final assessment, report).

Runs as root inside the run container (`python3 -m playground run --manifest M --out /out`).
On a host without root it runs everything as the current user, which the tests use.

The stations come from the case's STATIONS.json. The time limits count agent time
only: checks, snapshots and other harness work between waves are not charged to the
agent.

The outer agent is Codex (`codex_agent`) or Claude Code (`claude_agent`), or a fake
stand-in for either. Claude Code gets the operator's token only in the environment of
its own process; should the agent print it, every result file outside `audit/` is
redacted.

Exit codes: 0 all stations ran (whatever the quality), 1 stopped early, 2 runner error
(traceback in `<out>/runner-error.txt`). `runner.json` records why a run stopped and
the category of that cause: harness, environment, product, or none (e.g. the time
budget was used up).
"""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import tempfile
import time
import traceback
from pathlib import Path
from typing import Any, Callable

from . import assess, claude_agent, codex_agent, lifecycle, methods, report
from .lifecycle import utc as _utc
from .manifest import CLAUDE_KINDS, ManifestError, load

DEFAULT_STATIONS = 4  # the v1 cases; the host passes the case's own count
SNAPSHOT_ALLOWANCE = 1200  # one capture (Git reads, sweep, copies, clone)
LAST_MESSAGE_LIMIT = 1 << 20
TOKEN_KINDS = ("input", "cachedInput", "output")
# Agent-home folders kept as evidence at the end: Codex's session records (all
# sessions, including subagents and Markitect's inner roles), Claude Code's session
# transcripts and Markitect's cache.
EVIDENCE = {"codex-sessions": ".codex/sessions", "claude-sessions": ".claude/projects",
            "markitect-cache": ".cache/Markitect"}


def overhead_bound_seconds(stations: int = DEFAULT_STATIONS) -> int:
    """Worst-case runner time outside the agent budget: setup, checks and snapshot per
    station, freeze and final assessment, plus a margin. The host adds it to totalSeconds."""
    return (methods.SETUP_BOUND_SECONDS + stations * (assess.STATION_BOUND_SECONDS + SNAPSHOT_ALLOWANCE)
            + SNAPSHOT_ALLOWANCE + assess.FINAL_BOUND_SECONDS + 600)


def _log(message: str) -> None:
    print(f"[runner {_utc()}] {message}", flush=True)


def _save(path: Path, value: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2, default=str) + "\n",
                    encoding="utf-8", newline="\n")


def _save_error(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(traceback.format_exc(), encoding="utf-8")


def _version(command: str, env: dict) -> str | None:
    try:
        proc = subprocess.run([command, "--version"], capture_output=True, timeout=60, env=env,
                              stdin=subprocess.DEVNULL)
    except (OSError, subprocess.SubprocessError):
        return None
    words = proc.stdout.decode("utf-8", errors="replace").split()
    return words if proc.returncode == 0 and words else None


def _codex_version(agent_home: Path) -> str | None:
    env = {"PATH": os.environ.get("PATH", ""), "HOME": str(agent_home),
           "CODEX_HOME": str(agent_home / ".codex"), "LANG": "C.UTF-8"}
    words = _version("codex", env)
    return words[-1] if words else None  # "codex-cli 0.162.0"


def _claude_version() -> str | None:
    """Asked in a throwaway home, so no Claude Code state lands in the agent's home."""
    with tempfile.TemporaryDirectory(prefix="mpg-claude-version-") as home:
        words = _version("claude", {"PATH": os.environ.get("PATH", ""), "HOME": home,
                                    "CLAUDE_CONFIG_DIR": home, "LANG": "C.UTF-8", **claude_agent.ENV})
    return words[0] if words else None  # "2.1.296 (Claude Code)"


def _versions(manifest: dict, agent_home: Path) -> dict[str, Any]:
    kind = manifest["agent"]["kind"]
    in_image = codex_agent.container_mode()  # the image ships both CLIs, so record both there
    return {
        "codex": _codex_version(agent_home) if kind == "codex" or in_image else None,
        "claude": _claude_version() if kind == "claude" or in_image else None,
        "imageId": os.environ.get("MPG_IMAGE_ID") or None,
        "markitectCommit": (os.environ.get("MPG_MARKITECT_COMMIT")
                            or (manifest.get("markitect") or {}).get("commit")),
        "markitectSha256": os.environ.get("MPG_MARKITECT_SHA256") or None,
    }


def _assess(func: Callable[..., dict], *args: Any, **kwargs: Any) -> dict:
    """Assessment problems are recorded, never allowed to end the trajectory."""
    try:
        return func(*args, **kwargs)
    except Exception:
        return {"passed": None, "total": None, "status": "error", "error": traceback.format_exc()}


def _main_commit(repo: Path) -> str | None:
    try:
        return lifecycle.git(repo, "rev-parse", "--verify", "--quiet", "refs/heads/main^{commit}")
    except lifecycle.LifecycleError:
        return None


def _new_main_commits(repo: Path, base: str | None, tip: str | None) -> int | None:
    if not base or not tip:
        return None
    try:
        return int(lifecycle.git(repo, "rev-list", "--count", f"{base}..{tip}"))
    except (lifecycle.LifecycleError, ValueError):
        return None


def _file_key(path: Path) -> tuple[int, int] | None:
    """Change marker (mtime, size) of a file; its content is never read."""
    try:
        info = os.stat(path, follow_symlinks=False)
    except OSError:
        return None
    return info.st_mtime_ns, info.st_size


def _usage(previous: dict, current: dict, outer: str | None) -> dict:
    """Tokens of one station from cumulative per-session totals before and after it.

    `outer` is the session the harness started; every other session that grew is a
    subagent or one of Markitect's inner roles. Unknown stays null, never zero.
    """
    def minus(after: dict, before: dict | None) -> dict:
        return {kind: after[kind] - ((before or {}).get(kind) or 0) if isinstance(after.get(kind), int) else None
                for kind in TOKEN_KINDS}

    deltas = {thread: minus(total, previous.get(thread)) for thread, total in current.items()
              if total != previous.get(thread)}
    if outer not in current:
        nothing = {kind: None for kind in TOKEN_KINDS}
        return {"outer": nothing, "all": nothing, "otherSessions": len(deltas)}
    every = [deltas.get(outer, {kind: 0 for kind in TOKEN_KINDS})]
    every += [delta for thread, delta in deltas.items() if thread != outer]
    return {"outer": every[0],
            "all": {kind: report.sum_known([delta[kind] for delta in every]) for kind in TOKEN_KINDS},
            "otherSessions": len(every) - 1}


def _stop_reason(number: int, session_id: str | None, result: dict, events: dict) -> str | None:
    """Infrastructure failures stop the run; timeouts and ordinary agent failures do not."""
    if session_id is None:
        return f"S{number}: agent never produced a session id (exit {result.get('exitCode')})"
    if result.get("exitCode") is None and not result.get("timedOut"):
        return f"S{number}: agent could not be started: {result.get('error')}"
    idle = not events.get("items") and all(v is None for v in (events.get("tokens") or {}).values())
    if result.get("exitCode") != 0 and not result.get("timedOut") and idle:
        return f"S{number}: agent exited {result.get('exitCode')} without doing any work (see stderr.log)"
    return None


class _Trajectory:
    def __init__(self, manifest: dict, out: Path, state: dict, *, in_dir: Path, work_root: Path,
                 agent_home: Path, auth_src: Path, token_src: Path) -> None:
        self.manifest, self.out, self.state = manifest, out, state
        self.in_dir, self.agent_home, self.auth_src, self.token_src = in_dir, agent_home, auth_src, token_src
        self.codex_home = agent_home / ".codex"
        self.claude_home = agent_home / claude_agent.CONFIG_DIR
        self.case, self.method = manifest["case"], manifest["method"]
        self.kind = manifest["agent"]["kind"]
        self.claude = self.kind in CLAUDE_KINDS
        self.work_root = work_root
        self.repo, self.audit = work_root / self.case, out / "audit"
        self.stations_planned = DEFAULT_STATIONS  # replaced by the case's plan in run()
        self.token: str | None = None  # never saved, logged or put on a command line
        self.mcp_config: Path | None = None
        # The agent's environment always points at the run's own home, also in test mode.
        self.agent_env = {"HOME": str(agent_home), "CODEX_HOME": str(self.codex_home)}
        if self.claude:
            self.agent_env.update(claude_agent.environment(self.claude_home))

    def save_state(self) -> None:
        _save(self.out / "runner.json", self.state)

    def stop(self, category: str, reason: str) -> str:
        """Record why the run stops: harness, environment, product or none."""
        self.state["stopCategory"] = category
        return reason

    def run(self) -> int:
        agent_cfg = self.manifest["agent"]
        # 1. Agent home: only the Codex login and the Claude Code token come from outside.
        self.codex_home.mkdir(parents=True, exist_ok=True)
        self.state["authInstalled"] = codex_agent.install_auth(self.auth_src, self.codex_home)
        if self.claude:
            self.claude_home.mkdir(parents=True, exist_ok=True)
            self.token = claude_agent.read_token(self.token_src)
            self.state["claudeTokenProvided"] = self.token is not None
        codex_agent.give_to_agent(self.agent_home, recursive=True)
        # 2. Fresh case repository, owned by the agent.
        _log(f"prepare {self.case} ({self.method})")
        prepared = lifecycle.prepare(self.in_dir / "cases", self.repo, self.audit, case=self.case,
                                     method=self.method)
        self.stations_planned = self.state["stationsPlanned"] = len(prepared["stationPlan"])
        codex_agent.give_to_agent(self.work_root, recursive=True)
        self.save_state()

        stop, setup = None, {}
        # Markitect's inner roles run on Codex, so Claude Code with Markitect needs both logins.
        needs_codex = self.kind == "codex" or (self.kind == "claude" and self.method == "markitect")
        if needs_codex and not self.state["authInstalled"]:
            stop = self.stop("environment", f"codex auth file missing: {self.auth_src}")
        elif self.kind == "claude" and self.token is None:
            stop = self.stop("environment", f"claude token file missing, empty or malformed: {self.token_src}")
        if stop is None:
            setup = self.setup()
            if setup.get("status") != "ready":
                stop = self.stop(setup.get("blockedBy") or "product",
                                 f"method setup {setup.get('status')}: {setup.get('error')}")
        if stop is None:
            servers = setup.get("mcpServers") or {}
            if self.claude:
                # Codex serves only Markitect's inner roles: no outer model, no MCP servers.
                codex_agent.write_config(self.codex_home, {"maxSubagents": agent_cfg["maxSubagents"]}, {})
                self.mcp_config = claude_agent.write_config(self.claude_home, servers)
            else:
                codex_agent.write_config(self.codex_home, agent_cfg, servers)
            codex_agent.give_to_agent(self.agent_home, recursive=True)
            login = _file_key(self.codex_home / "auth.json")
            stop = self.stations()
            # A changed login file usually means Codex refreshed its token inside the run.
            self.state["codexLoginChanged"] = login is not None and _file_key(self.codex_home / "auth.json") != login
        self.state["stopReason"] = stop
        self.save_state()
        self.keep_evidence()

        reason = stop or f"completed S1-S{self.stations_planned}"
        _log(f"freeze ({reason})")
        frozen = lifecycle.freeze(self.repo, self.audit, reason=reason, sweep=codex_agent.kill_all_agent_processes)
        if self.state["stationsRun"]:
            _log("final assessment")
            folder = self.out / "final"
            if frozen["immutableMain"]:
                final = _assess(assess.final, Path(frozen["immutableMain"]), self.case, self.method,
                                self.in_dir, folder, station=self.stations_planned)
            else:
                final = {"passed": None, "total": None, "status": "error", "error": "no main branch to assess"}
            _save(folder / "final.json", final)
        return 0 if stop is None else 1

    def setup(self) -> dict:
        _log(f"method setup: {self.method}")
        folder = self.out / "setup"
        ctx = {"manifest": self.manifest, "case": self.case, "inDir": self.in_dir, "outDir": folder,
               "agentHome": self.agent_home, "runAsAgent": codex_agent.run_as_agent}
        started = time.monotonic()
        result = methods.setup(self.method, self.repo, ctx)
        if result.get("seconds") is None:
            result["seconds"] = round(time.monotonic() - started, 3)
        result["leftoverProcessesKilled"] = codex_agent.kill_all_agent_processes()
        _save(folder / "setup.json", result)
        return result

    def session_totals(self) -> dict:
        """Cumulative tokens per recorded session: Codex threads (outer Codex, its subagents,
        Markitect's inner roles) and, for Claude Code, its transcripts (`claude:` keys)."""
        totals = codex_agent.session_totals(self.codex_home / "sessions")
        if self.claude:
            totals.update(claude_agent.session_totals(self.claude_home / "projects"))
        return totals

    def agent_command(self, prompt: str, session_id: str | None, last_message: Path) -> tuple[list[str], dict]:
        """The station's command and environment; the token never enters the command."""
        agent_cfg = self.manifest["agent"]
        if not self.claude:
            return codex_agent.build_command(agent_cfg, prompt, session_id, last_message), self.agent_env
        env = {**self.agent_env, claude_agent.TOKEN_ENV: self.token} if self.token else self.agent_env
        return claude_agent.build_command(agent_cfg, prompt, session_id, self.mcp_config), env

    def stations(self) -> str | None:
        limits = self.manifest["limits"]
        prompt = (self.in_dir / "cases" / "task-prompt.txt").read_text(encoding="utf-8").strip()
        messages = self.agent_home / "last-messages"  # Codex writes its last message there itself
        if not self.claude:
            messages.mkdir(parents=True, exist_ok=True)
            codex_agent.give_to_agent(messages)
        used = 0.0  # agent seconds only; the budget starts with S1
        session_id: str | None = None
        totals = self.session_totals()
        main_before = _main_commit(self.repo)
        for number in range(1, self.stations_planned + 1):
            remaining = limits["totalSeconds"] - used
            if remaining <= 0:
                return self.stop("none", f"total time used up before S{number}")
            folder = self.out / "stations" / f"S{number}"
            folder.mkdir(parents=True, exist_ok=True)
            events_path, last_message = folder / "events.jsonl", messages / f"S{number}.txt"
            command, env = self.agent_command(prompt, session_id, last_message)
            timeout = min(limits["stationSeconds"], remaining)
            _log(f"S{number}: agent {'resumes' if session_id else 'starts'} (timeout {timeout:.0f}s)")
            result = codex_agent.run_as_agent(command, self.repo, timeout, events_path,
                                              folder / "stderr.log", extra_env=env)
            used += result.get("seconds") or 0
            leftovers = codex_agent.kill_all_agent_processes()
            if self.claude:
                self.redact(folder)
                events = claude_agent.parse_events(events_path)
                claude_agent.save_last_message(events_path, folder / "last-message.txt", LAST_MESSAGE_LIMIT)
            else:
                events = codex_agent.parse_events(events_path)
                self.copy_last_message(last_message, folder / "last-message.txt")
            resumed = session_id
            session_id = events.get("sessionId") or session_id
            outer = (claude_agent.session_key(session_id) if self.claude else session_id) if session_id else None
            current = self.session_totals()
            tokens = events["tokens"]
            if outer and outer not in current and any(v is not None for v in tokens.values()):
                if self.claude:  # no transcript: the stream holds this wave's usage, not a running total
                    before = totals.get(outer) or {}
                    current[outer] = {kind: (before.get(kind) or 0) + (tokens[kind] or 0) for kind in TOKEN_KINDS}
                else:  # no session record: use the event stream's running total
                    current[outer] = tokens
            record = {
                **result,
                "station": f"S{number}",
                "command": command,
                "resumedSession": resumed,
                "sessionId": session_id,
                "sessionSwitched": bool(resumed and events.get("sessionId") and events["sessionId"] != resumed),
                "timeoutSeconds": round(timeout, 3),
                "leftoverProcessesKilled": leftovers + (result.get("killedOnTimeout") or 0),
                "events": events,
                "usage": _usage(totals, current, outer),
                "mainCommit": None, "newMainCommits": None, "snapshot": None,
            }
            totals = {**totals, **current}
            _save(folder / "agent.json", record)  # saved before anything else can fail
            self.state["stationsRun"] = number
            self.save_state()

            try:
                snap = lifecycle.snapshot(self.repo, self.audit, sweep=codex_agent.kill_all_agent_processes)
            except Exception:
                _save_error(folder / "snapshot-error.txt")
                return self.stop("harness", f"S{number}: snapshot failed (see stations/S{number}/snapshot-error.txt)")
            main_after = snap["record"]["immutableMain"]["commit"]
            record.update(mainCommit=main_after, newMainCommits=_new_main_commits(self.repo, main_before, main_after),
                          snapshot=Path(snap["dir"]).relative_to(self.out.resolve()).as_posix(),
                          snapshotErrors=snap["record"]["errors"])
            _save(folder / "agent.json", record)
            try:
                self.capture_outside(folder)
            except Exception:
                _save_error(folder / "outside-error.txt")
            if self.claude:
                self.redact(folder / "outside")
            if snap["immutableMain"]:
                checks = _assess(assess.station, Path(snap["immutableMain"]), self.case, number,
                                 self.in_dir, folder)
            else:
                checks = {"passed": None, "total": None, "status": "error", "error": "no main branch to check"}
            _save(folder / "checks.json", checks)
            _log(f"S{number}: exit {result.get('exitCode')}, timed out {result.get('timedOut')}, "
                 f"checks {checks.get('passed')}/{checks.get('total')}")
            main_before = main_after or main_before

            stop = _stop_reason(number, session_id, result, events)
            if stop:
                return self.stop("environment", stop)
            if number < self.stations_planned:
                if limits["totalSeconds"] - used <= 0:
                    return self.stop("none", f"total time used up after S{number}")
                try:
                    lifecycle.advance(self.repo, self.audit)
                except Exception:
                    _save_error(folder / "advance-error.txt")
                    return self.stop("harness", f"S{number}: releasing S{number + 1} failed "
                                                f"(see stations/S{number}/advance-error.txt)")
        return None

    def redact(self, root: Path, **kwargs: Any) -> None:
        """Remove the Claude Code token from result files the agent's output reached."""
        if self.token:
            changed = claude_agent.redact_tree(root, self.token, **kwargs)
            self.state["tokenRedactions"] = (self.state.get("tokenRedactions") or 0) + changed

    def copy_last_message(self, source: Path, target: Path) -> None:
        """The agent owns the source: never follow a link, read at most LAST_MESSAGE_LIMIT."""
        handle = lifecycle.open_plain(source, codex_agent.agent_uid())
        if handle is not None:
            with handle:
                target.write_bytes(handle.read(LAST_MESSAGE_LIMIT))

    def capture_outside(self, folder: Path) -> None:
        """Unfinished work outside the case repo: Markitect's candidate workspaces and Git
        worktrees the agent added elsewhere (copied, capped, never following links)."""
        owner = codex_agent.agent_uid()
        sources = {}
        workspaces = self.agent_home / ".cache" / "Markitect" / "workspaces"
        if workspaces.is_dir() and not workspaces.is_symlink():
            sources["markitect-workspaces"] = workspaces
        try:
            listing = lifecycle.git(self.repo, "worktree", "list", "--porcelain")
        except lifecycle.LifecycleError:
            listing = ""
        paths = [Path(line[len("worktree "):]) for line in listing.splitlines() if line.startswith("worktree ")]
        for index, path in enumerate(paths, 1):
            if (path.is_dir() and not path.is_symlink() and path.resolve() != self.repo.resolve()
                    and (owner is None or path.stat().st_uid == owner)):
                sources[f"worktree-{index}"] = path
        if not sources:
            return
        codex_agent.kill_all_agent_processes()
        found = {}
        for name, path in sources.items():
            manifest = lifecycle.copy_tree(path, folder / "outside" / name, owner=owner)
            found[name] = {"path": str(path), "entries": len(manifest)}
        _save(folder / "outside.json", found)

    def keep_evidence(self) -> None:
        try:
            codex_agent.kill_all_agent_processes()
            for name, relative in EVIDENCE.items():
                source = self.agent_home / relative
                if source.is_dir() and not source.is_symlink():
                    lifecycle.copy_tree(source, self.out / "evidence" / name, owner=codex_agent.agent_uid())
        except Exception:
            _save_error(self.out / "evidence" / "error.txt")


def run(manifest: dict, out: Path, *, in_dir: Path = Path("/in"), work_root: Path = Path("/work"),
        agent_home: Path = Path("/home/agent"),
        auth_src: Path = Path("/run/secrets/codex-auth.json"),
        token_src: Path = Path("/run/secrets/claude-token")) -> int:
    """Run one trajectory for an already validated manifest; see the module docstring."""
    out = Path(out).resolve()
    out.mkdir(parents=True, exist_ok=True)
    hide_results = codex_agent.container_mode()
    if hide_results:
        os.chmod(out, 0o700)  # check results and audit stay invisible to the agent user
    started = time.monotonic()
    state: dict[str, Any] = {
        "schema": 1, "manifest": manifest, "status": "running", "exitCode": None, "stopReason": None,
        "stopCategory": None, "startedAt": _utc(), "endedAt": None, "wallSeconds": None,
        "stationsPlanned": None, "stationsRun": 0, "authInstalled": None, "codexLoginChanged": None,
        "claudeTokenProvided": None, "tokenRedactions": 0, "versions": _versions(manifest, agent_home),
    }
    trajectory = _Trajectory(manifest, out, state, in_dir=in_dir, work_root=work_root,
                             agent_home=agent_home, auth_src=auth_src, token_src=token_src)
    trajectory.save_state()
    try:
        code = trajectory.run()
        state["status"] = "completed" if code == 0 else "stopped"
    except Exception:
        (out / "runner-error.txt").write_text(traceback.format_exc(), encoding="utf-8")
        try:
            codex_agent.kill_all_agent_processes()
        except Exception:
            pass
        state["status"], code = "error", 2
    try:
        # Audit snapshots keep their hashed raw bytes; everything else is cleaned.
        trajectory.redact(out, skip=("audit",))
    except Exception:
        with (out / "runner-error.txt").open("a", encoding="utf-8") as handle:
            handle.write("redaction failed:\n" + traceback.format_exc())
        state["status"], code = "error", 2
    state.update(exitCode=code, endedAt=_utc(), wallSeconds=round(time.monotonic() - started, 3))
    trajectory.save_state()
    try:
        report.build(out)
    except Exception:
        with (out / "runner-error.txt").open("a", encoding="utf-8") as handle:
            handle.write("report failed:\n" + traceback.format_exc())
        code = 2
    if hide_results:
        os.chmod(out, 0o755)  # all agent processes are gone; let the host read the results
    _log(f"done: {state['status']} (exit {code})")
    return code


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="python -m playground run",
                                     description="Run one trajectory inside the run container.")
    parser.add_argument("--manifest", type=Path, required=True)
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args(argv)
    try:
        manifest = load(args.manifest)
    except ManifestError as exc:
        print(f"manifest error: {exc}", file=sys.stderr)
        args.out.mkdir(parents=True, exist_ok=True)
        (args.out / "runner-error.txt").write_text(f"manifest error: {exc}\n", encoding="utf-8")
        return 2
    return run(manifest, args.out)


if __name__ == "__main__":
    raise SystemExit(main())
