"""Arm P: one Markitect Apply run by the Claude Code plugin in claude-plugin/plugin.

Arm P executes Markitect's Apply method (Assessment, Implement and Verify
through the complete model-derived Manager tree, guarded publication) with
Claude Code agents, a workflow script and git worktrees instead of the native
runtime. Its inputs are arm M's fixtures and its candidates are assessed by
assess_candidate.py exactly like arm M's and arm B's.

  python -I -B armp.py input copy --repo M_INPUT_REPO --rev REV --out NEW.bundle
  python -I -B armp.py input m2 --from CANDIDATE.bundle --commit SHA --out NEW.bundle
  python -I -B armp.py run --input BUNDLE --commit SHA --out NEW_DIR --label LABEL \
      --markitect BIN --work-root DIR [--model MODEL] [--effort LEVEL] [--deadline-seconds N] \
      [--max-agents N] [--branch NAME] [--lock FILE] [--claude BIN] [--credentials FILE]

`input copy` bundles an existing input commit (arm M's inputs/<step>/repo) so
both arms start from the same bytes. `input m2` makes S2's input from the
chain's own previous candidate with the one-line M2 policy edit, as arm M's
controller does. S3 re-applies the previous candidate unchanged.

`run` clones the input, starts one headless `claude -p /markitect:apply` with
an isolated configuration (no user settings, plugins, CLAUDE.md, memory or MCP
servers) and a login copy without refresh token, waits under the deadline,
reads the publication record back, checks the published candidate again,
writes start.bundle, result.bundle (the published candidate), work.bundle
(every Manager branch and check note), the transcripts and result.json, and
removes the worktrees, the login copy and the work repository.
"""
import argparse
import datetime
import hashlib
import importlib.util
import json
import os
import pathlib
import shutil
import signal
import subprocess
import sys
import tempfile
import time

HERE = pathlib.Path(__file__).resolve().parent
PLUGIN = HERE / "claude-plugin" / "plugin"


def _load(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


runner = _load("comparison_runner", HERE / "runner.py")
mkp = _load("markitect_plugin_mkp", PLUGIN / "scripts" / "mkp.py")
RunError = runner.RunError

INPUT_IDENTITY = ("Markitect Library Fixture", "fixture@markitect.invalid")
MANAGER_IDENTITY = ("Markitect Manager", "manager@markitect.invalid")
M2_EDIT = (b"  maxOpenLoans: 5\n", b"  maxOpenLoans: 3\n")
LIMITS = {"deadline": 2700, "max_agents": 150, "max_corrections": 2, "max_returns": 2, "max_questions": 3,
          "check_timeout": 600, "kill": 15, "login_margin": 600}
TOOLS = ["Bash", "Read", "Edit", "Write", "Grep", "Glob", "Workflow"]
DENIED = ["Agent", "WebFetch", "WebSearch", "NotebookEdit"]


def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def sha256_file(path):
    return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()


def plugin_digest():
    """One digest over every plugin file's path and bytes."""
    h = hashlib.sha256()
    for p in sorted(x for x in PLUGIN.rglob("*") if x.is_file() and "__pycache__" not in x.parts):
        h.update(p.relative_to(PLUGIN).as_posix().encode() + b"\0" + p.read_bytes() + b"\0")
    return h.hexdigest()


def git(repo, *args, **kw):
    return runner.git(repo, *args, **kw)


def identity_env(identity, date=None):
    name, email = identity
    env = {"GIT_AUTHOR_NAME": name, "GIT_AUTHOR_EMAIL": email, "GIT_COMMITTER_NAME": name, "GIT_COMMITTER_EMAIL": email}
    if date:
        env.update(GIT_AUTHOR_DATE=date, GIT_COMMITTER_DATE=date)
    return env


# --- Inputs -------------------------------------------------------------------

def describe(repo, ref, bundle):
    commit = git(repo, "rev-parse", ref + "^{commit}")
    return {"commit": commit, "tree": git(repo, "rev-parse", commit + "^{tree}"), "bundle": str(bundle),
            "bundle_sha256": sha256_file(bundle)}


def input_copy(repo, rev, out):
    """Bundle exactly one existing input commit under its branch name."""
    repo, out = pathlib.Path(repo), pathlib.Path(out)
    if out.exists():
        raise RunError(f"{out} exists")
    commit = git(repo, "rev-parse", rev + "^{commit}")
    with tempfile.TemporaryDirectory() as tmp:
        work = pathlib.Path(tmp) / "repo"
        subprocess.run(["git", "init", "-q", str(work)], check=True, capture_output=True)
        git(work, "fetch", "-q", str(repo.resolve()), commit)
        branch = next((b for b in git(repo, "branch", "--format=%(refname:short)", "--points-at", commit).splitlines()), "dev/input")
        git(work, "branch", branch, commit)
        git(work, "bundle", "create", str(out.resolve()), branch)
        return {"kind": "copy", "branch": branch, **describe(work, branch, out)}


def input_m2(source, commit, out):
    """S2: the chain's own candidate plus the one-line M2 edit of arm M's controller."""
    out = pathlib.Path(out)
    if out.exists():
        raise RunError(f"{out} exists")
    with tempfile.TemporaryDirectory() as tmp:
        work, _ = runner.scratch_repo(tmp, source)
        git(work, "config", "core.autocrlf", "false")
        git(work, "checkout", "-q", "-B", "dev/library-m2", commit)
        policy = work / ".markitect" / "model" / "lending" / "loan-policy.yaml"
        data = policy.read_bytes()
        if data.count(M2_EDIT[0]) != 1:
            raise RunError("the M2 edit needs exactly one maxOpenLoans: 5 line")
        policy.write_bytes(data.replace(*M2_EDIT))
        git(work, "add", "--all")
        git(work, "commit", "-q", "-m", "Library M2 model", env=identity_env(INPUT_IDENTITY))
        changed = git(work, "diff", "--name-only", "HEAD~1", "HEAD").splitlines()
        if changed != [".markitect/model/lending/loan-policy.yaml"]:
            raise RunError(f"the M2 input changed more than the policy: {changed}")
        git(work, "bundle", "create", str(out.resolve()), "dev/library-m2")
        return {"kind": "m2", "branch": "dev/library-m2", "from": commit, **describe(work, "dev/library-m2", out)}


# --- Login --------------------------------------------------------------------

def session_login(raw, deadline_seconds, margin):
    """The session's copy of the Claude login: OAuth access only, no refresh token.

    A run must not rotate the user's own login, so the copy cannot refresh;
    the access token must therefore outlive the deadline.
    """
    login = json.loads(raw)
    oauth = login.get("claudeAiOauth") if isinstance(login, dict) else None
    if not isinstance(oauth, dict) or not oauth.get("accessToken") or not isinstance(oauth.get("expiresAt"), int):
        raise RunError("the credentials hold no Claude account login")
    left = oauth["expiresAt"] / 1000 - time.time()
    if left < deadline_seconds + margin:
        raise RunError(f"the Claude login expires in {int(left)} s; use Claude once to refresh it, then start again")
    copy = {"claudeAiOauth": {k: v for k, v in oauth.items() if k not in ("refreshToken", "refreshTokenExpiresAt")}}
    secrets = [oauth[k].encode() for k in ("accessToken", "refreshToken") if oauth.get(k)]
    return json.dumps(copy).encode(), secrets


# --- Session ------------------------------------------------------------------

def claude_argv(claude, model, effort, state):
    argv = [claude, "-p", "/markitect:apply", "--plugin-dir", str(PLUGIN), "--output-format", "stream-json", "--verbose",
            "--strict-mcp-config", "--setting-sources", "project", "--add-dir", str(state),
            "--allowedTools", *TOOLS, "--disallowedTools", *DENIED]
    if model:
        argv += ["--model", model]
    if effort:
        argv += ["--effort", effort]
    return argv


KEPT = ("CLAUDE_CODE_GIT_BASH_PATH",)


def session_env(work, state):
    """The session's environment: no user Claude, Anthropic or Markitect settings,
    a private home, Git without system configuration or credential helper, and
    the user's Go module path so builds need no fresh downloads."""
    gopath = subprocess.run(["go", "env", "GOPATH"], capture_output=True, text=True).stdout.strip()
    env = {k: v for k, v in os.environ.items()
           if k in KEPT or not k.startswith(("CLAUDE", "ANTHROPIC", "MARKITECT_", "GIT_", "GH_", "GITHUB_"))}
    home = work / "home"
    home.mkdir(parents=True, exist_ok=True)
    env.update(identity_env(MANAGER_IDENTITY))
    env.update(HOME=str(home), USERPROFILE=str(home), GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0",
               GIT_CONFIG_COUNT="2", GIT_CONFIG_KEY_0="core.longpaths", GIT_CONFIG_VALUE_0="true",
               GIT_CONFIG_KEY_1="credential.helper", GIT_CONFIG_VALUE_1="")
    if gopath:
        env["GOPATH"] = gopath
    return env


def kill_tree(proc):
    if proc.poll() is not None:
        return
    if os.name == "nt":
        subprocess.run(["taskkill", "/PID", str(proc.pid), "/T", "/F"], capture_output=True)
    else:
        os.killpg(proc.pid, signal.SIGKILL)
    try:
        proc.wait(timeout=LIMITS["kill"])
    except subprocess.TimeoutExpired:
        pass


def run_session(argv, cwd, env, stdout, stderr, deadline):
    started = time.monotonic()
    kw = {"creationflags": subprocess.CREATE_NEW_PROCESS_GROUP} if os.name == "nt" else {"start_new_session": True}
    with open(stdout, "wb") as out, open(stderr, "wb") as err:
        proc = subprocess.Popen(argv, cwd=cwd, env=env, stdout=out, stderr=err, stdin=subprocess.DEVNULL, **kw)
        try:
            code, timed_out = proc.wait(timeout=deadline), False
        except subprocess.TimeoutExpired:
            kill_tree(proc)
            code, timed_out = proc.returncode, True
    return {"exit": code, "timed_out": timed_out, "seconds": round(time.monotonic() - started, 1), "alive_after": proc.poll() is None}


def usage(transcripts):
    """Token use over every main and agent transcript, each assistant message once."""
    seen, total, by_model, agents = set(), {}, {}, 0
    keys = ("input_tokens", "output_tokens", "cache_read_input_tokens", "cache_creation_input_tokens")
    for path in sorted(pathlib.Path(transcripts).rglob("*.jsonl")):
        if path.name.startswith("agent-"):
            agents += 1
        for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
            try:
                entry = json.loads(line)
            except ValueError:
                continue
            message = entry.get("message") if isinstance(entry, dict) else None
            if entry.get("type") != "assistant" or not isinstance(message, dict) or not isinstance(message.get("usage"), dict):
                continue
            key = (message.get("id"), entry.get("requestId"))
            if key in seen:
                continue
            seen.add(key)
            model = by_model.setdefault(message.get("model", "unknown"), dict.fromkeys(keys, 0))
            for k in keys:
                value = message["usage"].get(k) or 0
                total[k] = total.get(k, 0) + value
                model[k] += value
    return {"total": total, "by_model": by_model, "messages": len(seen), "agent_transcripts": agents}


def stream_summary(path):
    """The session's own result events: final answer, errors and reported cost."""
    results, errors, events = [], [], 0
    for line in pathlib.Path(path).read_text(encoding="utf-8", errors="replace").splitlines():
        try:
            event = json.loads(line)
        except ValueError:
            continue
        events += 1
        if event.get("type") == "result":
            results.append({k: event.get(k) for k in ("subtype", "is_error", "duration_ms", "num_turns", "total_cost_usd")}
                           | {"result": str(event.get("result", ""))[-2000:]})
            if event.get("is_error"):
                errors.append(event.get("subtype"))
    return {"events": events, "results": results, "errors": errors}


def remove_tree(path):
    """Remove a work directory, including Git's read-only object files on Windows."""
    def writable(func, target, _):
        os.chmod(target, 0o700)
        func(target)
    if pathlib.Path(path).exists():
        shutil.rmtree(path, onexc=writable)


def worktrees(repo):
    paths = [l[len("worktree "):] for l in git(repo, "worktree", "list", "--porcelain").splitlines() if l.startswith("worktree ")]
    return paths[1:]


def run(a):
    out = pathlib.Path(a.out).resolve()
    if out.exists():
        raise RunError(f"{out} exists")
    if not runner.LABEL.match(a.label):
        raise RunError("the label must match " + runner.LABEL.pattern)
    work_root = pathlib.Path(a.work_root).resolve()
    if work_root.as_posix().lower().startswith("c:/windows/"):
        raise RunError("the work root must not be under C:\\Windows: worktree isolation refuses its path spelling")
    deadline = a.deadline_seconds
    credentials = pathlib.Path(a.credentials or pathlib.Path.home() / ".claude" / ".credentials.json")
    out.mkdir(parents=True)
    work = work_root / a.label
    if work.exists():
        raise RunError(f"{work} exists")
    result = {"label": a.label, "arm": "P", "model": a.model, "effort": a.effort,
              "input_bundle_sha256": sha256_file(a.input), "plugin_sha256": plugin_digest(),
              "runner_sha256": sha256_file(__file__), "markitect_sha256": sha256_file(a.markitect),
              "limits": {**LIMITS, "deadline": deadline, "max_agents": a.max_agents}, "started_utc": now(),
              "status": "incomplete", "technical_pass": False, "error": None, "residue": [], "steps": []}
    secrets = []
    login_before = sha256_file(credentials) if credentials.exists() else None
    with runner.native_lock(a.lock, "comparison arm P", a.label):
        try:
            login, secrets = session_login(credentials.read_bytes(), deadline, LIMITS["login_margin"])
            repo = work / "repo"
            config = work / "claude-config"
            state = out / "state"
            config.mkdir(parents=True)
            (config / ".credentials.json").write_bytes(login)
            subprocess.run(["git", "init", "-q", str(repo)], check=True, capture_output=True)
            git(repo, "config", "core.autocrlf", "false")
            git(repo, "config", "core.eol", "lf")
            heads = git(repo, "bundle", "unbundle", str(pathlib.Path(a.input).resolve())).splitlines()
            if not any(l.split()[0] == a.commit for l in heads):
                raise RunError(f"the input bundle has no head at {a.commit}")
            branch = a.branch or next(l.split()[1] for l in heads if l.split()[0] == a.commit).removeprefix("refs/heads/")
            git(repo, "update-ref", "refs/heads/" + branch, a.commit)
            git(repo, "symbolic-ref", "HEAD", "refs/heads/" + branch)
            git(repo, "reset", "-q", "--hard", a.commit)
            result.update(input_commit=a.commit, input_tree=git(repo, "rev-parse", a.commit + "^{tree}"), owner_branch=branch)
            git(repo, "bundle", "create", str(out / "start.bundle"), branch)
            env = session_env(work, state)
            env.update(CLAUDE_CONFIG_DIR=str(config), MARKITECT_BIN=str(pathlib.Path(a.markitect).resolve()),
                       MARKITECT_STATE_DIR=str(state), MARKITECT_RUN_ID=a.label, MARKITECT_OWNER_BRANCH=branch,
                       MARKITECT_MAX_AGENTS=str(a.max_agents), MARKITECT_MAX_CORRECTIONS=str(LIMITS["max_corrections"]),
                       MARKITECT_MAX_RETURNS=str(LIMITS["max_returns"]), MARKITECT_MAX_QUESTIONS=str(LIMITS["max_questions"]),
                       MARKITECT_CHECK_TIMEOUT=str(LIMITS["check_timeout"]))
            state.mkdir(parents=True)
            argv = claude_argv(a.claude, a.model, a.effort, state)
            result["argv"] = argv
            result["session_started_utc"] = now()
            session = run_session(argv, repo, env, out / "session.stream.jsonl", out / "session.stderr.log", deadline)
            result.update(session=session, session_ended_utc=now())
            result["stream"] = stream_summary(out / "session.stream.jsonl")
            publication_file = state / a.label / "publication.json"
            publication = json.loads(publication_file.read_text(encoding="utf-8")) if publication_file.exists() else None
            result["publication"] = publication
            published = bool(publication and publication.get("published"))
            if published:
                os.environ["MARKITECT_BIN"] = env["MARKITECT_BIN"]
                candidate = git(repo, "rev-parse", "refs/heads/markitect/apply/" + a.label)
                code, report = mkp.markitect("check", repo, candidate)
                readback = {"branch_matches": candidate == publication.get("candidate"), "check": mkp.summarize(code, report),
                            "check_passed": code == 0 and not mkp.stale_or_orphaned(report)}
                result["readback"] = readback
                result.update(result_commit=candidate, result_tree=git(repo, "rev-parse", candidate + "^{tree}"))
                git(repo, "bundle", "create", str(out / "result.bundle"), "markitect/apply/" + a.label)
            git(repo, "bundle", "create", str(out / "work.bundle"), "--all")
            transcripts = config / "projects"
            if transcripts.exists():
                shutil.copytree(transcripts, out / "transcripts")
                result["usage"] = usage(out / "transcripts")
            result["status"] = "completed" if session["exit"] == 0 and not session["timed_out"] else "incomplete"
            result["technical_pass"] = (result["status"] == "completed" and published and result["readback"]["branch_matches"]
                                        and result["readback"]["check_passed"])
        except RunError as e:
            result["error"] = str(e)
        except Exception as e:  # recorded, never hidden
            result["error"] = f"{type(e).__name__}: {e}"
        finally:
            repo = work / "repo"
            if repo.exists():
                for path in worktrees(repo):
                    subprocess.run(["git", "-C", str(repo), "worktree", "remove", "--force", path], capture_output=True)
                subprocess.run(["git", "-C", str(repo), "worktree", "prune"], capture_output=True)
                left = worktrees(repo)
                if left:
                    result["residue"].append({"worktrees": left})
            try:
                remove_tree(work)
            except OSError as e:
                result["residue"].append({"work_removal": str(e)})
            if work.exists():
                result["residue"].append({"work": str(work)})
            # The owner's own Claude use may refresh the login during a run; that is
            # recorded, not a failure: the run's copy cannot refresh.
            result["login_sha256_before"], result["login_sha256_after"] = login_before, (sha256_file(credentials) if credentials.exists() else None)
            result["login_changed_during_run"] = result["login_sha256_before"] != result["login_sha256_after"]
            result["secret_in_outputs"], result["secret_scan_unreadable"] = runner.files_containing(out, secrets)
            result["quarantine"] = bool(result["secret_in_outputs"] or result["secret_scan_unreadable"])
            secrets = None
            if result["residue"]:
                result["technical_pass"] = False
            result["ended_utc"] = now()
            (out / "result.json").write_text(json.dumps(result, indent=1), encoding="utf-8")
            listing = [f"{sha256_file(p)}  {p.relative_to(out).as_posix()}" for p in sorted(out.rglob("*")) if p.is_file()]
            (out / "files.sha256").write_text("\n".join(listing) + "\n", encoding="utf-8")
    print(json.dumps({k: result.get(k) for k in ("label", "status", "technical_pass", "error", "result_commit")}, indent=1))
    return 0 if result["technical_pass"] else 1


def main(argv=None):
    sys.stdout.reconfigure(encoding="utf-8")
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    sub = p.add_subparsers(dest="command", required=True)
    i = sub.add_parser("input")
    isub = i.add_subparsers(dest="kind", required=True)
    c = isub.add_parser("copy")
    c.add_argument("--repo", required=True)
    c.add_argument("--rev", default="HEAD")
    c.add_argument("--out", required=True)
    m = isub.add_parser("m2")
    m.add_argument("--from", dest="source", required=True)
    m.add_argument("--commit", required=True)
    m.add_argument("--out", required=True)
    r = sub.add_parser("run")
    for name in ("--input", "--commit", "--out", "--label", "--markitect", "--work-root"):
        r.add_argument(name, required=True)
    r.add_argument("--model", default="sonnet")
    r.add_argument("--effort", default="high")
    r.add_argument("--deadline-seconds", type=int, default=LIMITS["deadline"])
    r.add_argument("--max-agents", type=int, default=LIMITS["max_agents"])
    r.add_argument("--branch", help="the owner branch name; default the input head's branch")
    r.add_argument("--lock")
    r.add_argument("--claude", default=shutil.which("claude") or "claude")
    r.add_argument("--credentials")
    a = p.parse_args(argv)
    try:
        if a.command == "input":
            value = input_copy(a.repo, a.rev, a.out) if a.kind == "copy" else input_m2(a.source, a.commit, a.out)
            print(json.dumps(value, indent=1))
            return 0
        return run(a)
    except RunError as e:
        print(json.dumps({"error": str(e)}), file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
