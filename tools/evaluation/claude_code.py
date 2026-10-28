"""The baseline variant: classic agentic coding, one Claude Code session with the task.

  python -I -B claude_code.py [--claude BIN] [--credentials FILE] [--effort LEVEL] TASK_DIR OUT_DIR [STATE_DIR]

Run it only through the suite: the suite starts it in its own process group,
which it stops at the end.

The model comes from the suite's EVALUATION_MODEL (claude:MODEL_ID). The
variant clones the task's start, places task.md as TASK.md (never committed),
and runs one headless `claude -p` session in the clone with an isolated
configuration: no user settings, plugins, CLAUDE.md, memory or MCP servers, and
a copy of the login without its refresh token, so a run can never rotate the
user's login. It stops the session a margin before the task's deadline, commits
whatever the session left with a fixed identity and writes result.bundle,
run.json, the session's stream and its transcripts. Every step of a chain is a
fresh session; the repository is its memory, and STATE_DIR is unused.

This is separation, not a sandbox: the session runs as the user on the host.
"""
import argparse
import json
import os
import pathlib
import shutil
import signal
import subprocess
import sys
import tempfile
import time

PROMPT = """Read the documentation in TASK.md and implement it accordingly.

Work as one coding agent in this repository, using the build, test and file tools available here. Deliver the implementation, automated tests and usage documentation that TASK.md requires. Do not change TASK.md.

If an implementation already exists, first reconcile it with TASK.md. Change files only where TASK.md, its required build and test commands, or necessary consequences of that work require it. Preserve conforming code, tests and documentation; do not add optional coverage, refactoring or cosmetic changes solely to produce a diff. Repair genuine defects you find against TASK.md.

When you finish, report the files you changed, the checks you actually ran and any requirement you could not resolve."""
TOOLS = ["Bash", "Read", "Edit", "Write", "Grep", "Glob", "Agent", "TodoWrite"]
DENIED = ["WebFetch", "WebSearch"]
IDENTITY = ("Claude Code Baseline", "baseline@markitect.invalid")
MARGIN, LOGIN_MARGIN, KILL = 60, 300, 15
KEPT = ("CLAUDE_CODE_GIT_BASH_PATH",)


class VariantError(Exception):
    pass


def git(repo, *args, env, binary=False):
    p = subprocess.run(["git", "-C", str(repo), *args], capture_output=True, text=not binary, env=env)
    if p.returncode != 0:
        raise VariantError(f"git {' '.join(args)}: {p.stderr if binary else p.stderr.strip()}")
    return p.stdout if binary else p.stdout.strip()


def base_env(home, hooks):
    """One Git setting for clone, session and capture: no system or user configuration, no line-ending
    conversion, no hooks, no credentials; a fixed identity; a private home; no Claude, Anthropic or Git variables."""
    env = {k: v for k, v in os.environ.items()
           if k in KEPT or not k.startswith(("CLAUDE", "ANTHROPIC", "GIT_", "GH_", "GITHUB_", "EVALUATION_"))}
    name, email = IDENTITY
    settings = {"core.autocrlf": "false", "core.hooksPath": str(hooks), "credential.helper": "", "commit.gpgsign": "false"}
    env.update(HOME=str(home), USERPROFILE=str(home), GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0",
               GIT_AUTHOR_NAME=name, GIT_AUTHOR_EMAIL=email, GIT_COMMITTER_NAME=name, GIT_COMMITTER_EMAIL=email,
               GIT_CONFIG_COUNT=str(len(settings)))
    for i, (key, value) in enumerate(settings.items()):
        env[f"GIT_CONFIG_KEY_{i}"], env[f"GIT_CONFIG_VALUE_{i}"] = key, value
    return env


def session_login(raw, seconds):
    """The session's copy of the Claude login: OAuth access only, valid beyond the session."""
    try:
        login = json.loads(raw)
    except ValueError:
        login = None
    oauth = login.get("claudeAiOauth") if isinstance(login, dict) else None
    if not isinstance(oauth, dict) or not oauth.get("accessToken") or not isinstance(oauth.get("expiresAt"), int):
        raise VariantError("the credentials hold no Claude account login")
    left = oauth["expiresAt"] / 1000 - time.time()
    if left < seconds + LOGIN_MARGIN:
        raise VariantError(f"the Claude login expires in {int(left)} s; use Claude once to refresh it, then start again")
    copy = {"claudeAiOauth": {k: v for k, v in oauth.items() if k not in ("refreshToken", "refreshTokenExpiresAt")}}
    secrets = [oauth[k].encode() for k in ("accessToken", "refreshToken") if oauth.get(k)]
    return json.dumps(copy).encode(), secrets


def stop_children(proc):
    """Stop the session and everything it started.

    The session stays in this process's group, so the suite's deadline kill reaches it too.
    On POSIX this process ignores the group's SIGTERM; on Windows the tree goes with taskkill.
    """
    if os.name == "nt":
        subprocess.run(["taskkill", "/PID", str(proc.pid), "/T", "/F"], capture_output=True)
    else:
        previous = signal.signal(signal.SIGTERM, signal.SIG_IGN)
        try:
            os.killpg(os.getpgrp(), signal.SIGTERM)
        finally:
            signal.signal(signal.SIGTERM, previous)
        if proc.poll() is None:
            proc.kill()
    try:
        proc.wait(timeout=KILL)
    except subprocess.TimeoutExpired:
        pass


def run_session(argv, prompt, cwd, env, stream, stderr, seconds):
    """Run the session with the prompt on stdin: a multi-line argument does not survive a .cmd shim on Windows."""
    started = time.monotonic()
    with open(prompt, "rb") as inp, open(stream, "wb") as out, open(stderr, "wb") as err:
        proc = subprocess.Popen(argv, cwd=cwd, env=env, stdout=out, stderr=err, stdin=inp)
        try:
            code, timed_out = proc.wait(timeout=seconds), False
        except subprocess.TimeoutExpired:
            code, timed_out = None, True
        # Background children outlive a normal exit too, and would keep files and the login open.
        stop_children(proc)
    return {"exitCode": proc.returncode if timed_out else code, "timedOut": timed_out, "sessionSeconds": round(time.monotonic() - started, 1)}


def stream_facts(path):
    """The model, the session and the final result from the session's stream-json output."""
    facts = {"model": None, "sessionId": None, "result": None}
    for line in pathlib.Path(path).read_text(encoding="utf-8", errors="replace").splitlines():
        try:
            event = json.loads(line)
        except ValueError:
            continue
        if not isinstance(event, dict):
            continue
        if event.get("type") == "system" and event.get("subtype") == "init":
            facts["model"], facts["sessionId"] = event.get("model"), event.get("session_id")
        elif event.get("type") == "result":
            facts["result"] = event
    return facts


def usage(result):
    u = (result or {}).get("usage") or {}
    return {"inputTokens": u.get("input_tokens"), "cacheReadTokens": u.get("cache_read_input_tokens"),
            "cacheCreationTokens": u.get("cache_creation_input_tokens"), "outputTokens": u.get("output_tokens"),
            "reasoningTokens": None, "turns": (result or {}).get("num_turns"), "costUsd": (result or {}).get("total_cost_usd")}


def contains_secret(root, secrets):
    for p in pathlib.Path(root).rglob("*"):
        if p.is_file():
            data = p.read_bytes()
            if any(s in data for s in secrets):
                return True
    return False


def capture(repo, start, out, env, secrets):
    """Commit what the session left (TASK.md excluded) and bundle the result as one complete branch.

    Every commit since the start is searched for the login first, uncompressed, because a bundle's
    packed objects hide it from a byte search. On a hit no bundle is written.
    """
    (repo / ".git" / "index.lock").unlink(missing_ok=True)  # left behind when the session was killed mid-commit
    git(repo, "add", "-A", env=env)
    if git(repo, "status", "--porcelain", env=env):
        git(repo, "commit", "-q", "--no-verify", "-m", "Baseline session result", env=env)
    head = git(repo, "rev-parse", "HEAD", env=env)
    history = git(repo, "log", "-p", "--binary", "--format=%H%n%B", f"{start}..{head}", env=env, binary=True)
    if any(s in history for s in secrets):
        return head, True
    git(repo, "branch", "-f", "result", head, env=env)
    git(repo, "bundle", "create", str(pathlib.Path(out).resolve()), "refs/heads/result", env=env)
    return head, False


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--claude", default=shutil.which("claude") or "claude")
    parser.add_argument("--credentials", default=str(pathlib.Path.home() / ".claude" / ".credentials.json"))
    parser.add_argument("--effort")
    parser.add_argument("task")
    parser.add_argument("out")
    parser.add_argument("state", nargs="?")
    a = parser.parse_args(argv)
    task, out = pathlib.Path(a.task), pathlib.Path(a.out)
    out.mkdir(parents=True, exist_ok=True)
    record = {"provider": "claude", "model": None, "status": "error", "cause": None}
    try:
        provider, _, model = os.environ.get("EVALUATION_MODEL", "").partition(":")
        if provider != "claude" or not model:
            raise VariantError(f"EVALUATION_MODEL must be claude:MODEL_ID, not {os.environ.get('EVALUATION_MODEL')!r}")
        record["requestedModel"] = model
        spec = json.loads((task / "task.json").read_text(encoding="utf-8"))
        seconds = spec["deadlineSeconds"] - MARGIN
        if seconds <= 0:
            raise VariantError(f"a deadline of {spec['deadlineSeconds']} s leaves no time for a session")
        login, secrets = session_login(pathlib.Path(a.credentials).read_bytes(), seconds)
        with tempfile.TemporaryDirectory(ignore_cleanup_errors=True) as tmp:
            tmp = pathlib.Path(tmp)
            repo, home, config, hooks = tmp / "repo", tmp / "home", tmp / "config", tmp / "no-hooks"
            for d in (home, config, hooks):
                d.mkdir()
            env = base_env(home, hooks)
            try:
                (config / ".credentials.json").write_bytes(login)
                git(tmp, "clone", "-q", "--no-checkout", str((task / "start.bundle").resolve()), str(repo), env=env)
                git(repo, "checkout", "-q", "-B", "work", spec["start"]["commit"], env=env)
                shutil.copyfile(task / "task.md", repo / "TASK.md")
                with open(repo / ".git" / "info" / "exclude", "a", encoding="utf-8") as f:
                    f.write("\n/TASK.md\n")
                (tmp / "prompt.txt").write_text(PROMPT, encoding="utf-8")
                argv = [a.claude, "-p", "--model", model, "--output-format", "stream-json", "--verbose",
                        "--strict-mcp-config", "--setting-sources", "project",
                        "--allowedTools", *TOOLS, "--disallowedTools", *DENIED]
                if a.effort:
                    argv += ["--effort", a.effort]
                record.update(run_session(argv, tmp / "prompt.txt", repo, dict(env, CLAUDE_CONFIG_DIR=str(config)),
                                          out / "session.stream.jsonl", out / "session.stderr.log", seconds))
            finally:
                # The login copy goes first, whatever happened.
                (config / ".credentials.json").unlink(missing_ok=True)
            facts = stream_facts(out / "session.stream.jsonl")
            record.update(model=facts["model"], sessionId=facts["sessionId"], usage=usage(facts["result"]),
                          sessionResult=(facts["result"] or {}).get("subtype"))
            if (config / "projects").is_dir():
                shutil.copytree(config / "projects", out / "transcripts")
            record["resultCommit"], leaked = capture(repo, spec["start"]["commit"], out / "result.bundle", env, secrets)
        record["status"] = "timed_out" if record["timedOut"] else ("completed" if record["exitCode"] == 0 else "failed")
        if leaked or contains_secret(out, secrets):
            for p in (out / "transcripts", out / "session.stream.jsonl", out / "session.stderr.log", out / "result.bundle"):
                if p.is_dir():
                    shutil.rmtree(p)
                elif p.exists():
                    p.unlink()
            record["quarantine"], record["status"] = True, "quarantined"
    except (VariantError, OSError, ValueError, KeyError) as e:
        record["cause"] = f"{type(e).__name__}: {e}"
    (out / "run.json").write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
    return 0 if record["status"] == "completed" else 1


if __name__ == "__main__":
    sys.exit(main())
