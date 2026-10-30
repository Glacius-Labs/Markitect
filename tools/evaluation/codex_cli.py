"""The baseline variant on Codex: classic agentic coding, one Codex CLI session with the task.

  python -I -B codex_cli.py [--codex BIN] [--credentials FILE] [--effort LEVEL] TASK_DIR OUT_DIR [STATE_DIR]

It is claude_code.py with Codex's CLI, for evaluations on Codex. Run it only
through the suite: the suite starts it in its own process group, which it stops
at the end.

The model comes from the suite's EVALUATION_MODEL (codex:MODEL_ID). The
variant clones the task's start, places task.md as TASK.md (never committed),
and runs one `codex exec --json` session in the clone, with claude_code.py's
prompt and Git setting and an isolated configuration: a private CODEX_HOME, no
user configuration, rules, project documents, apps, plugins, memories or MCP
servers, no web search, image generation, browser or computer use, and a copy
of the ChatGPT account login with an empty refresh token, so a run can never
rotate the user's login. Codex's own sandbox and approvals are off, as in a
Manager's Codex session: the evaluation sandbox is the boundary. Codex's native
subagents stay available, as Claude Code's Agent tool does for the Claude
baseline. It stops the session a margin before the task's deadline, commits
whatever the session left with a fixed identity and writes result.bundle,
run.json, the session's event stream and its session files. Every step of a
chain is a fresh session; the repository is its memory, and STATE_DIR is unused.

exec's events do not name the model: run.json reports the one the session was
launched with, as the host's Codex adapter does (internal/agents/codex).

The login is the suite's EVALUATION_LOGIN when it sets one, as the sandbox does
(`suite.py run --sandbox`). In host mode the session runs as the user on the
host: that is separation, not a sandbox.
"""
import argparse
import base64
import json
import os
import pathlib
import shutil
import sys
import tempfile
import time

# Isolated mode (-I) leaves the script's directory off the path.
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import claude_code as baseline  # noqa: E402

# A Manager's Codex switches (internal/agents/codex) without the Manager tools and with the native subagents left on;
# plus no plugins and no web search, as the Claude baseline has no plugins and is denied WebSearch and WebFetch.
SWITCHES = [
    "--disable", "apps", "--disable", "plugins", "--disable", "memories", "--disable", "image_generation",
    "--disable", "browser_use", "--disable", "computer_use",
    "-c", 'cli_auth_credentials_store="file"', "-c", 'model_provider="openai"', "-c", 'web_search="disabled"',
    "-c", "project_doc_max_bytes=0", "-c", "project_doc_fallback_filenames=[]",
    "--ignore-user-config", "--ignore-rules", "--dangerously-bypass-approvals-and-sandbox",
]
EFFORTS = ("minimal", "low", "medium", "high", "xhigh")
# Besides the variables claude_code.base_env clears: the account login only, never an API key or another home.
CLEARED = ("OPENAI_", "CODEX_")


def expiry(token):
    """The exp claim of a JWT access token, in seconds since the epoch."""
    parts = token.split(".")
    try:
        if len(parts) != 3:
            raise ValueError("not three parts")
        exp = json.loads(base64.b64decode(parts[1] + "=" * (-len(parts[1]) % 4), altchars="-_", validate=True))["exp"]
    except (ValueError, KeyError, TypeError):
        raise baseline.VariantError("the Codex access token is no JWT with an expiry")
    if not isinstance(exp, int) or isinstance(exp, bool) or exp <= 0:
        raise baseline.VariantError("the Codex access token is no JWT with an expiry")
    return exp


def session_login(raw, seconds):
    """The session's copy of the ChatGPT account login, valid beyond the session: the ID and access tokens and the
    account with an empty refresh token, as the host's Codex adapter copies it. An API key is refused."""
    try:
        login = json.loads(raw)
    except ValueError:
        login = None
    if isinstance(login, dict) and login.get("OPENAI_API_KEY"):
        raise baseline.VariantError("the credentials hold an OpenAI API key; only a ChatGPT account login is admitted")
    tokens = login.get("tokens") if isinstance(login, dict) else None
    if not isinstance(tokens, dict) or not all(isinstance(tokens.get(k), str) and tokens[k] for k in ("access_token", "id_token")):
        raise baseline.VariantError("the credentials hold no ChatGPT account login")
    left = expiry(tokens["access_token"]) - time.time()
    if left < seconds + baseline.LOGIN_MARGIN:
        raise baseline.VariantError(f"the Codex login expires in {int(left)} s; run codex once to refresh it, then start again")
    account = tokens.get("account_id")
    copy = {"OPENAI_API_KEY": None,
            "tokens": {"id_token": tokens["id_token"], "access_token": tokens["access_token"], "refresh_token": "",
                       "account_id": account if isinstance(account, str) else ""},
            "last_refresh": login.get("last_refresh")}
    secrets = [tokens[k].encode() for k in ("access_token", "id_token", "refresh_token") if isinstance(tokens.get(k), str) and tokens[k]]
    return json.dumps(copy).encode(), secrets


def stream_facts(path):
    """The thread, the completed turns' usage and the errors from `codex exec --json`, read as the host's Codex adapter
    reads them (internal/agents/codex/stream.go)."""
    facts = {"threadId": None, "completed": 0, "usage": {}, "failure": None, "lastError": None}
    for line in pathlib.Path(path).read_text(encoding="utf-8", errors="replace").splitlines():
        try:
            event = json.loads(line)
        except ValueError:
            continue
        if not isinstance(event, dict):
            continue
        kind = event.get("type")
        if kind == "error":
            facts["lastError"] = event.get("message")
            continue
        if isinstance(kind, str) and kind.startswith(("thread.", "turn.", "item.")):
            facts["lastError"] = None  # work after an error event recovered from it, as classify.py reads it
        if kind == "thread.started":
            facts["threadId"] = event.get("thread_id")
        elif kind == "turn.completed":
            facts["completed"] += 1
            for key, value in (event.get("usage") if isinstance(event.get("usage"), dict) else {}).items():
                if isinstance(value, int) and not isinstance(value, bool):
                    facts["usage"][key] = facts["usage"].get(key, 0) + value
        elif kind == "turn.failed" and isinstance(event.get("error"), dict):
            facts["failure"] = event["error"].get("message")
    return facts


def session_end(facts):
    """How the session ended: a completed turn answered, whatever error events came before it; else its failure or
    the error it ended on, which the CLI reports after its own retries."""
    if facts["completed"]:
        return "completed", None
    error = facts["failure"] or facts["lastError"]
    return ("failed", str(error)[:2000]) if error else (None, None)


def usage(facts):
    """The session's tokens as claude_code.py reports them; exec's input tokens include the cached ones."""
    u = facts["usage"]
    total, cached = u.get("input_tokens"), u.get("cached_input_tokens")
    return {"inputTokens": total - (cached or 0) if total is not None else None, "cacheReadTokens": cached, "cacheCreationTokens": None,
            "outputTokens": u.get("output_tokens"), "reasoningTokens": u.get("reasoning_output_tokens"), "turns": None, "costUsd": None}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--codex", default=shutil.which("codex") or "codex")
    parser.add_argument("--credentials", default=os.environ.get("EVALUATION_LOGIN") or str(pathlib.Path.home() / ".codex" / "auth.json"))
    parser.add_argument("--effort", choices=EFFORTS)
    parser.add_argument("task")
    parser.add_argument("out")
    parser.add_argument("state", nargs="?")
    a = parser.parse_args(argv)
    task, out = pathlib.Path(a.task), pathlib.Path(a.out)
    out.mkdir(parents=True, exist_ok=True)
    record = {"provider": "codex", "model": None, "status": "error", "cause": None}
    try:
        provider, _, model = os.environ.get("EVALUATION_MODEL", "").partition(":")
        if provider != "codex" or not model or any(c in model for c in ' "\n'):
            raise baseline.VariantError(f"EVALUATION_MODEL must be codex:MODEL_ID, not {os.environ.get('EVALUATION_MODEL')!r}")
        record["requestedModel"] = model
        spec = json.loads((task / "task.json").read_text(encoding="utf-8"))
        seconds = spec["deadlineSeconds"] - baseline.MARGIN
        if seconds <= 0:
            raise baseline.VariantError(f"a deadline of {spec['deadlineSeconds']} s leaves no time for a session")
        login, secrets = session_login(pathlib.Path(a.credentials).read_bytes(), seconds)
        with tempfile.TemporaryDirectory(ignore_cleanup_errors=True) as tmp:
            tmp = pathlib.Path(tmp)
            repo, home, codex_home, hooks = tmp / "repo", tmp / "home", tmp / "codex", tmp / "no-hooks"
            for d in (home, codex_home, hooks):
                d.mkdir()
            env = {k: v for k, v in baseline.base_env(home, hooks).items() if not k.startswith(CLEARED)}
            try:
                (codex_home / "auth.json").write_bytes(login)
                baseline.git(tmp, "clone", "-q", "--no-checkout", str((task / "start.bundle").resolve()), str(repo), env=env)
                baseline.git(repo, "checkout", "-q", "-B", "work", spec["start"]["commit"], env=env)
                shutil.copyfile(task / "task.md", repo / "TASK.md")
                with open(repo / ".git" / "info" / "exclude", "a", encoding="utf-8") as f:
                    f.write("\n/TASK.md\n")
                (tmp / "prompt.txt").write_text(baseline.PROMPT, encoding="utf-8")
                argv = [a.codex, "exec", "--json", *SWITCHES, "-m", model]
                if a.effort:
                    argv += ["-c", f'model_reasoning_effort="{a.effort}"']
                argv.append("-")  # the prompt comes on stdin
                record.update(baseline.run_session(argv, tmp / "prompt.txt", repo, dict(env, CODEX_HOME=str(codex_home)),
                                                   out / "session.stream.jsonl", out / "session.stderr.log", seconds))
            finally:
                # The login copy goes first, whatever happened.
                (codex_home / "auth.json").unlink(missing_ok=True)
            facts = stream_facts(out / "session.stream.jsonl")
            ended, error = session_end(facts)
            record.update(model=model if facts["threadId"] else None, sessionId=facts["threadId"], usage=usage(facts), sessionResult=ended)
            if error:
                record["sessionError"] = error
            # Without a thread the CLI failed before the model saw the task, on its configuration or its login: the
            # variant's own machinery, which the status `error` marks, as for a failure of this code.
            unstarted = not facts["threadId"] and not record["timedOut"]
            if unstarted:
                record["cause"] = f"the codex session started no thread (exit {record['exitCode']}); see session.stderr.log"
            if (codex_home / "sessions").is_dir():
                shutil.copytree(codex_home / "sessions", out / "transcripts")
            record["resultCommit"], leaked = baseline.capture(repo, spec["start"]["commit"], out / "result.bundle", env, secrets)
        record["status"] = ("timed_out" if record["timedOut"] else "error" if unstarted
                            else "completed" if record["exitCode"] == 0 else "failed")
        if leaked or baseline.contains_secret(out, secrets):
            for p in (out / "transcripts", out / "session.stream.jsonl", out / "session.stderr.log", out / "result.bundle"):
                if p.is_dir():
                    shutil.rmtree(p)
                elif p.exists():
                    p.unlink()
            record["quarantine"], record["status"] = True, "quarantined"
    except (baseline.VariantError, OSError, ValueError, KeyError) as e:
        record["cause"] = f"{type(e).__name__}: {e}"
    (out / "run.json").write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
    return 0 if record["status"] == "completed" else 1


if __name__ == "__main__":
    sys.exit(main())
