"""The baseline variant: classic agentic coding, one Claude Code session with the task.

  python -I -B claude_code.py [--claude BIN] [--credentials FILE] [--effort LEVEL] TASK_DIR OUT_DIR [STATE_DIR]

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


def git(repo, *args, env=None):
    p = subprocess.run(["git", "-C", str(repo), *args], capture_output=True, text=True, env=env)
    if p.returncode != 0:
        raise VariantError(f"git {' '.join(args)}: {p.stderr.strip()}")
    return p.stdout.strip()


def identity_env():
    name, email = IDENTITY
    return {"GIT_AUTHOR_NAME": name, "GIT_AUTHOR_EMAIL": email, "GIT_COMMITTER_NAME": name, "GIT_COMMITTER_EMAIL": email}


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


def session_env(home, config):
    """No user Claude, Anthropic or Git settings; a private home; Git without system configuration or credentials."""
    env = {k: v for k, v in os.environ.items()
           if k in KEPT or not k.startswith(("CLAUDE", "ANTHROPIC", "GIT_", "GH_", "GITHUB_", "EVALUATION_"))}
    env.update(identity_env())
    env.update(HOME=str(home), USERPROFILE=str(home), CLAUDE_CONFIG_DIR=str(config), GIT_CONFIG_NOSYSTEM="1",
               GIT_TERMINAL_PROMPT="0", GIT_CONFIG_COUNT="1", GIT_CONFIG_KEY_0="credential.helper", GIT_CONFIG_VALUE_0="")
    return env


def run_session(argv, prompt, cwd, env, stream, stderr, seconds):
    """Run the session with the prompt on stdin: a multi-line argument does not survive a .cmd shim on Windows."""
    kw = {"creationflags": subprocess.CREATE_NEW_PROCESS_GROUP} if os.name == "nt" else {"start_new_session": True}
    started = time.monotonic()
    with open(prompt, "rb") as inp, open(stream, "wb") as out, open(stderr, "wb") as err:
        proc = subprocess.Popen(argv, cwd=cwd, env=env, stdout=out, stderr=err, stdin=inp, **kw)
        try:
            code, timed_out = proc.wait(timeout=seconds), False
        except subprocess.TimeoutExpired:
            if os.name == "nt":
                subprocess.run(["taskkill", "/PID", str(proc.pid), "/T", "/F"], capture_output=True)
            else:
                os.killpg(proc.pid, signal.SIGKILL)
            try:
                proc.wait(timeout=KILL)
            except subprocess.TimeoutExpired:
                pass
            code, timed_out = proc.returncode, True
    return {"exitCode": code, "timedOut": timed_out, "sessionSeconds": round(time.monotonic() - started, 1)}


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


def capture(repo, out):
    """Commit what the session left (TASK.md excluded) and bundle the result as one complete branch."""
    env = dict(os.environ, **identity_env())
    git(repo, "add", "-A")
    if git(repo, "status", "--porcelain"):
        git(repo, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "Baseline session result", env=env)
    head = git(repo, "rev-parse", "HEAD")
    git(repo, "branch", "-f", "result", head)
    git(repo, "bundle", "create", str(pathlib.Path(out).resolve()), "refs/heads/result")
    return head


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
        with tempfile.TemporaryDirectory() as tmp:
            tmp = pathlib.Path(tmp)
            repo, home, config = tmp / "repo", tmp / "home", tmp / "config"
            home.mkdir()
            config.mkdir()
            (config / ".credentials.json").write_bytes(login)
            git(tmp, "clone", "-q", str((task / "start.bundle").resolve()), str(repo))
            git(repo, "checkout", "-q", "-B", "work", spec["start"]["commit"])
            shutil.copyfile(task / "task.md", repo / "TASK.md")
            with open(repo / ".git" / "info" / "exclude", "a", encoding="utf-8") as f:
                f.write("\n/TASK.md\n")
            (tmp / "prompt.txt").write_text(PROMPT, encoding="utf-8")
            argv = [a.claude, "-p", "--model", model, "--output-format", "stream-json", "--verbose",
                    "--strict-mcp-config", "--setting-sources", "project",
                    "--allowedTools", *TOOLS, "--disallowedTools", *DENIED]
            if a.effort:
                argv += ["--effort", a.effort]
            record.update(run_session(argv, tmp / "prompt.txt", repo, session_env(home, config), out / "session.stream.jsonl", out / "session.stderr.log", seconds))
            facts = stream_facts(out / "session.stream.jsonl")
            record.update(model=facts["model"], sessionId=facts["sessionId"], usage=usage(facts["result"]),
                          sessionResult=(facts["result"] or {}).get("subtype"))
            if (config / "projects").is_dir():
                shutil.copytree(config / "projects", out / "transcripts")
            record["resultCommit"] = capture(repo, out / "result.bundle")
        record["status"] = "timed_out" if record["timedOut"] else ("completed" if record["exitCode"] == 0 else "failed")
        if contains_secret(out, secrets):
            for p in (out / "transcripts", out / "session.stream.jsonl", out / "session.stderr.log"):
                if p.is_dir():
                    shutil.rmtree(p)
                elif p.exists():
                    p.unlink()
            record["quarantine"] = True
    except (VariantError, OSError, ValueError, KeyError) as e:
        record["cause"] = f"{type(e).__name__}: {e}"
    (out / "run.json").write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
    return 0 if record["status"] == "completed" else 1


if __name__ == "__main__":
    sys.exit(main())
