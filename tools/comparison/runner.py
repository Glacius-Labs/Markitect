"""Arm B of the single-agent comparison: one Codex session, nothing else.

One `codex exec` session reads a repository whose only guidance is
`requirements.md` and works in it until it ends or the deadline expires. Within
a longitudinal chain the next step resumes the same Codex thread, the agent's
normal continuation. The session container gets the hardening, network and
Codex switches of a Markitect Manager session (internal/adapters/container and
internal/adapters/codexappserver), without the Markitect MCP tools and the
Manager harness, and the aggregate resource ceiling of the six Manager
sessions an Apply can run at once. The runner then removes the login, captures
the working tree and the Codex session files, and removes what it created.

Fixtures are Git bundles built with plumbing, so bytes and modes are exact:
  python -I -B runner.py fixture new --requirements FILE --message MSG --out BUNDLE
  python -I -B runner.py fixture continue --from BUNDLE [--requirements FILE] --message MSG --out BUNDLE
  python -I -B runner.py fixture root --from BUNDLE --rev COMMIT --drop .markitect --requirements FILE --message MSG --out BUNDLE
  python -I -B runner.py same-tree --a BUNDLE --a-rev COMMIT --b BUNDLE --b-rev COMMIT --exclude .markitect --exclude requirements.md

One session (one attempt, no retry):
  python -I -B runner.py run --fixture BUNDLE --image sha256:... --login AUTH_JSON --out NEW_DIR --label NAME
      [--resume-sessions CODEX_SESSIONS_TAR --resume-thread THREAD_ID] [--lock FILE]

The out directory must not exist. Agent-produced code and transcripts stay
outside the repository. The login is read once, never printed or written to
the host, and reaches the session without its refresh token.
"""
import argparse
import datetime
import hashlib
import json
import os
import pathlib
import re
import shutil
import subprocess
import sys
import tempfile
import time

HERE = pathlib.Path(__file__).resolve().parent
PROMPT = HERE / "packet" / "prompt.txt"
MODEL, EFFORT = "gpt-6-sol", "high"
LABEL_KEY = "markitect.comparison"
FIXTURE_IDENTITY = ("Library Fixture", "fixture@library.invalid")
FIXTURE_DATE = "2026-10-24T00:00:00Z"

# One Manager session: the example runtime profile's session limits. An Apply
# runs up to MANAGERS such sessions at once; arm B gets that aggregate ceiling.
SESSION_LIMITS = {"cpus": "1", "memory": "2147483648", "pids": "192", "tmp": "268435456"}
MANAGERS = 6
ARM_LIMITS = {key: str(int(value) * MANAGERS) for key, value in SESSION_LIMITS.items()}
# Same switches as providerArgv in internal/adapters/codexappserver/profile.go,
# without the Markitect MCP server, plus the thread settings that adapter
# sends (approval never, danger-full-access; cwd is the container's /work).
CODEX_SWITCHES = [
    "--disable", "apps", "--disable", "memories", "--disable", "multi_agent", "--disable", "multi_agent_v2",
    "-c", "agents.enabled=false", "--disable", "image_generation", "--disable", "browser_use",
    "--disable", "computer_use", "-c", 'cli_auth_credentials_store="file"', "-c", 'model_provider="openai"',
    "-c", "project_doc_max_bytes=0", "-c", "project_doc_fallback_filenames=[]",
    "-c", 'projects."/work".trust_level="untrusted"',
]
THREAD_SETTINGS = ["-m", MODEL, "-c", 'model_reasoning_effort="' + EFFORT + '"',
                   "-c", 'sandbox_mode="danger-full-access"', "-c", 'approval_policy="never"']
LIMITS = {"deadline": 2700, "metadata": 20, "helper": 120, "kill": 8}
LABEL = re.compile(r"^[a-z0-9][a-z0-9-]{2,62}$")
THREAD = re.compile(r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")


class RunError(RuntimeError):
    pass


class native_lock:
    """Exclusive use of the engine and account, shared with the arm-M controller.

    The lock file is created exclusively before the first engine call and
    removed after the final census, only by its holder. A held lock refuses.
    """

    def __init__(self, path, holder, run_id):
        self.path, self.record = (pathlib.Path(path) if path else None), {"holder": holder, "run": run_id}

    def __enter__(self):
        if self.path:
            self.record["started_utc"] = datetime.datetime.now(datetime.timezone.utc).isoformat()
            try:
                fd = os.open(self.path, os.O_CREAT | os.O_EXCL | os.O_WRONLY)
            except FileExistsError:
                raise RunError("the engine is busy: " + self.path.read_text(encoding="utf-8", errors="replace").strip())
            with os.fdopen(fd, "w", encoding="utf-8") as f:
                json.dump(self.record, f)
        return self

    def __exit__(self, *exc):
        if self.path:
            self.path.unlink()
        return False


# --- Fixtures -----------------------------------------------------------------

def git(repo, *args, data=None, env=None, binary=False):
    full = dict(os.environ, **(env or {}))
    result = subprocess.run(["git", "-c", "core.autocrlf=false", "-C", str(repo), *args], input=data,
                            capture_output=True, env=full, timeout=120)
    if result.returncode != 0:
        raise RunError("git " + " ".join(args[:2]) + ": " + result.stderr.decode(errors="replace").strip())
    return result.stdout if binary else result.stdout.decode().strip()


def fixture_env():
    name, email = FIXTURE_IDENTITY
    return {"GIT_AUTHOR_NAME": name, "GIT_AUTHOR_EMAIL": email, "GIT_AUTHOR_DATE": FIXTURE_DATE,
            "GIT_COMMITTER_NAME": name, "GIT_COMMITTER_EMAIL": email, "GIT_COMMITTER_DATE": FIXTURE_DATE}


def scratch_repo(tmp, *bundles):
    repo = pathlib.Path(tmp) / "repo"
    subprocess.run(["git", "init", "-q", str(repo)], check=True, capture_output=True)
    heads = set()
    for bundle in bundles:
        heads |= {line.split()[0] for line in git(repo, "bundle", "unbundle", str(pathlib.Path(bundle).resolve())).splitlines()}
    return repo, heads


def build_fixture(kind, out, message, requirements=None, source=None, rev=None, drop=()):
    """Write a one-branch bundle and return its commit, tree and hashes."""
    out = pathlib.Path(out)
    if out.exists():
        raise RunError("fixture output exists: " + str(out))
    if kind != "new" and not source:
        raise RunError(kind + " needs --from")
    with tempfile.TemporaryDirectory() as tmp:
        repo, heads = scratch_repo(tmp, *([source] if source else []))
        base = None
        if source:
            base = git(repo, "rev-parse", "--verify", (rev or (heads.pop() if len(heads) == 1 else "")) + "^{commit}")
            git(repo, "read-tree", base)
        for path in drop:
            git(repo, "rm", "--cached", "-r", "-q", "--ignore-unmatch", "--", path)
        if requirements:
            blob = git(repo, "hash-object", "-w", "--no-filters", "--stdin", data=pathlib.Path(requirements).read_bytes())
            git(repo, "update-index", "--add", "--cacheinfo", "100644," + blob + ",requirements.md")
        tree = git(repo, "write-tree")
        if kind == "continue" and tree == git(repo, "rev-parse", base + "^{tree}"):
            commit = base
        else:
            parents = ["-p", base] if kind == "continue" else []
            commit = git(repo, "commit-tree", tree, *parents, "-m", message, env=fixture_env())
        git(repo, "update-ref", "refs/heads/main", commit)
        git(repo, "symbolic-ref", "HEAD", "refs/heads/main")
        git(repo, "bundle", "create", str(out.resolve()), "HEAD", "main")
        subtree = subprocess.run(["git", "-C", str(repo), "rev-parse", "--verify", "-q", commit + ":realization"],
                                 capture_output=True, text=True).stdout.strip() or None
    return {"kind": kind, "commit": commit, "tree": tree, "realization_tree": subtree, "base": base,
            "bundle_sha256": sha256_file(out), "requirements_sha256": sha256_file(requirements) if requirements else None}


def same_tree(a, a_rev, b, b_rev, exclude):
    """Compare every path, mode and blob of two commits outside the excluded paths."""
    def entries(repo, rev):
        rows = {}
        for row in git(repo, "ls-tree", "-r", "-z", "--full-tree", rev).split("\0"):
            if row:
                meta, path = row.split("\t", 1)
                if not any(path == e or path.startswith(e.rstrip("/") + "/") for e in exclude):
                    rows[path] = meta
        return rows
    with tempfile.TemporaryDirectory() as tmp:
        repo, _ = scratch_repo(tmp, a, b)
        left, right = entries(repo, a_rev), entries(repo, b_rev)
    differences = sorted(p for p in set(left) | set(right) if left.get(p) != right.get(p))
    return {"equal": not differences, "paths": len(left), "differences": differences}


# --- Session ------------------------------------------------------------------

def sha256_file(path):
    return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()


def session_login(raw):
    """The product's session copy of a dedicated runtime login: refresh token removed."""
    login = json.loads(raw)
    tokens = login.get("tokens") if isinstance(login, dict) else None
    if (not isinstance(tokens, dict) or login.get("auth_mode") != "chatgpt" or login.get("OPENAI_API_KEY") is not None
            or not all(isinstance(tokens.get(k), str) and tokens.get(k) for k in ("access_token", "id_token", "account_id"))):
        raise RunError("the login is not a dedicated ChatGPT runtime login")
    copy = {"auth_mode": "chatgpt", "OPENAI_API_KEY": None,
            "tokens": {"id_token": tokens["id_token"], "access_token": tokens["access_token"],
                       "refresh_token": "", "account_id": tokens["account_id"]},
            "last_refresh": login.get("last_refresh")}
    secrets = [tokens[k].encode() for k in ("access_token", "id_token", "refresh_token") if tokens.get(k)]
    return json.dumps(copy).encode(), secrets


def files_containing(out, secrets):
    """Retained files that contain a login secret, Git objects included.

    Transcripts are not redacted, and the result bundle stores compressed
    objects, so the bundle is scanned through its unpacked object contents.
    """
    found, unscanned = [], []
    for path in sorted(p for p in out.rglob("*") if p.is_file()):
        data = path.read_bytes()
        if path.name == "result.bundle" and data:
            try:
                with tempfile.TemporaryDirectory() as tmp:
                    repo, _ = scratch_repo(tmp, path)
                    data = git(repo, "cat-file", "--batch-all-objects", "--batch", binary=True)
            except RunError:
                unscanned.append(path.relative_to(out).as_posix())
                continue
        if any(secret in data for secret in secrets):
            found.append(path.relative_to(out).as_posix())
    return found, unscanned


def hardening(user="1000:1000", tmp=SESSION_LIMITS["tmp"]):
    return ["--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--user", user,
            "--tmpfs", "/tmp:rw,exec,nosuid,nodev,mode=1777,size=" + tmp]


def codex_argv(prompt, resume=None):
    head = ["exec", "resume"] if resume else ["exec"]
    return [*head, "--json", "--skip-git-repo-check", *CODEX_SWITCHES, *THREAD_SETTINGS, *([resume] if resume else []), prompt]


def session_argv(name, image, work, home, prompt, resume=None):
    return ["run", "--rm", "--init", "--name", name, *hardening(tmp=ARM_LIMITS["tmp"]), "--workdir", "/work",
            "--cpus", ARM_LIMITS["cpus"], "--memory", ARM_LIMITS["memory"], "--pids-limit", ARM_LIMITS["pids"],
            "--label", LABEL_KEY + "=" + name,
            "--mount", "type=volume,src=" + work + ",dst=/work", "--mount", "type=volume,src=" + home + ",dst=/home/agent",
            "-e", "CODEX_HOME=/home/agent/.codex", "--entrypoint", "/usr/local/bin/codex", image, *codex_argv(prompt, resume)]


def helper_argv(image, mounts, script, user="1000:1000"):
    args = ["run", "--rm", "-i", "--network", "none", *hardening(user), "--pids-limit", "64", "--memory", "1073741824"]
    for volume, target in mounts:
        args += ["--mount", "type=volume,src=" + volume + ",dst=" + target]
    return args + ["--entrypoint", "/bin/sh", image, "-c", script]


POPULATE = ("set -e; cat > /tmp/fixture.bundle; git clone -q -b main /tmp/fixture.bundle /work; cd /work; "
            "git remote remove origin; git config user.name '%s'; git config user.email '%s'; "
            "git rev-parse HEAD HEAD^{tree}") % FIXTURE_IDENTITY
RESTORE_SESSIONS = "set -e; umask 077; mkdir -p /home/agent/.codex; tar -x -C /home/agent/.codex"
INSTALL_LOGIN = "set -e; umask 077; mkdir -p /home/agent/.codex; cat > /home/agent/.codex/auth.json"
REMOVE_LOGIN = "rm -f /home/agent/.codex/auth.json && test ! -e /home/agent/.codex/auth.json"
RAW_TAR = "tar -c -C /work --exclude=./.git ."
STATUS = "cd /work && git -c core.fsmonitor=false status --porcelain=v1 --ignored -uall"
# The candidate is the whole working tree the writer left. Files its own
# .gitignore excludes stay out, except Go sources, module files and Markdown,
# which an over-broad pattern (for example `library` matching cmd/library/)
# could otherwise drop silently.
CAPTURE = ("set -e; cd /work; G='git -c core.hooksPath=/dev/null -c core.fsmonitor=false -c commit.gpgsign=false'; "
           "$G add -A; $G ls-files -z -o -i --exclude-standard | grep -z -E '(\\.go|\\.md|(^|/)go\\.(mod|sum))$' "
           "| xargs -0 -r $G add -f --; "
           "$G -c user.name='%s' -c user.email='%s' commit -q --allow-empty --no-verify -m 'Library work'; "
           "$G rev-parse HEAD HEAD^{tree} >&2; $G bundle create -q /tmp/result.bundle HEAD; cat /tmp/result.bundle") % FIXTURE_IDENTITY
SESSIONS_TAR = "cd /home/agent/.codex && if [ -d sessions ]; then tar -c sessions; else tar -c -T /dev/null; fi"


class Recorder:
    """Run engine steps with finite limits; keep every step's record."""

    def __init__(self, engine, out):
        self.engine, self.out, self.steps = engine, out, []

    def __call__(self, name, args, timeout, data=None, stdout_file=None, secret_input=False):
        started = time.monotonic()
        record = {"step": name, "argv": args, "timeout_seconds": timeout, "stdin": "secret" if secret_input else bool(data)}
        out = open(self.out / (name + ".stdout"), "wb") if stdout_file is None else open(stdout_file, "wb")
        stdin = {"input": data} if data is not None else {"stdin": subprocess.DEVNULL}
        try:
            with out:
                result = subprocess.run([*self.engine, *args], stdout=out, stderr=subprocess.PIPE, timeout=timeout, **stdin)
            record.update(exit=result.returncode, timed_out=False)
            (self.out / (name + ".stderr")).write_bytes(result.stderr)
        except subprocess.TimeoutExpired as exc:
            record.update(exit=None, timed_out=True)
            (self.out / (name + ".stderr")).write_bytes(exc.stderr or b"")
        record["seconds"] = round(time.monotonic() - started, 3)
        self.steps.append(record)
        return record

    def text(self, name, suffix="stdout"):
        return (self.out / (name + "." + suffix)).read_text(encoding="utf-8", errors="replace")


def census(rec, name, label):
    """Names of containers and volumes with the label; a failed query refuses."""
    names = []
    for kind, args in (("containers", ["ps", "-a", "--filter", "label=" + label, "--format", "{{.Names}}"]),
                       ("volumes", ["volume", "ls", "--filter", "label=" + label, "--format", "{{.Name}}"])):
        step = rec(name + "-" + kind, args, LIMITS["metadata"])
        if step["exit"] != 0:
            raise RunError("census query failed: " + name + "-" + kind)
        names += rec.text(name + "-" + kind).split()
    return names


def session_absent(rec, label, step_name):
    step = rec(step_name, ["ps", "-a", "--filter", "name=^/?" + label + "$", "--format", "{{.Names}}"], LIMITS["metadata"])
    return step["exit"] == 0 and not rec.text(step_name).split()


def summarize(transcript):
    """Counts, usage and the final message from the `codex exec --json` stream."""
    events, items, usage = {}, {}, {}
    final, thread, failures, errors, malformed = None, None, [], [], 0
    for line in transcript.read_text(encoding="utf-8", errors="replace").splitlines():
        try:
            event = json.loads(line)
        except ValueError:
            malformed += 1
            continue
        if not isinstance(event, dict):
            malformed += 1
            continue
        kind = str(event.get("type"))
        events[kind] = events.get(kind, 0) + 1
        item = event.get("item") if isinstance(event.get("item"), dict) else None
        if kind == "item.completed" and item:
            items[str(item.get("type"))] = items.get(str(item.get("type")), 0) + 1
            if item.get("type") == "agent_message":
                final = item.get("text")
        if kind == "thread.started":
            thread = event.get("thread_id")
        if kind == "turn.completed" and isinstance(event.get("usage"), dict):
            for key, value in event["usage"].items():
                if isinstance(value, int):
                    usage[key] = usage.get(key, 0) + value
        # A turn failure ends the turn; a stream error may be retried by Codex,
        # as the Manager adapter tolerates errors marked willRetry.
        if kind == "turn.failed":
            failures.append(event)
        if kind == "error":
            errors.append(event)
    return {"thread_id": thread, "events": events, "completed_items": items, "usage": usage or None,
            "final_message": final, "failures": failures, "errors": errors, "malformed_lines": malformed,
            "turn_completed": events.get("turn.completed", 0) == 1}


def run_session(engine, fixture, image, login_path, out, label, deadline, resume_sessions=None, resume_thread=None):
    if not LABEL.match(label):
        raise RunError("label must match " + LABEL.pattern)
    if bool(resume_sessions) != bool(resume_thread) or (resume_thread and not THREAD.match(resume_thread)):
        raise RunError("a resume needs both the Codex sessions archive and the thread UUID")
    out = pathlib.Path(out)
    out.mkdir(parents=True, exist_ok=False)
    rec = Recorder(engine, out)
    prompt = PROMPT.read_text(encoding="utf-8")
    result = {"label": label, "arm": "B", "model": MODEL, "effort": EFFORT, "image": image,
              "fixture_bundle_sha256": sha256_file(fixture), "prompt_sha256": sha256_file(PROMPT),
              "runner_sha256": sha256_file(__file__), "limits": {**LIMITS, "deadline": deadline, **ARM_LIMITS},
              "resume_thread": resume_thread, "resume_sessions_sha256": sha256_file(resume_sessions) if resume_sessions else None,
              "started_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(), "status": "incomplete"}
    work, home = label + "-work", label + "-home"
    filter_label = LABEL_KEY + "=" + label
    created, login_installed, secrets, session_gone = [], False, [], True
    try:
        raw_login = pathlib.Path(login_path).read_bytes()
        result["login_sha256_before"] = hashlib.sha256(raw_login).hexdigest()
        login, secrets = session_login(raw_login)
        del raw_login
        rec("image", ["image", "inspect", "--format", "{{.Id}}", image], LIMITS["metadata"])
        if rec.text("image").strip() != image:
            raise RunError("image Id differs or the image is missing")
        if census(rec, "census-before", LABEL_KEY):
            raise RunError("comparison containers or volumes already exist")
        for volume in (work, home):
            if rec("create-" + volume[len(label) + 1:], ["volume", "create", "--label", filter_label, volume], LIMITS["metadata"])["exit"] != 0:
                raise RunError("volume creation failed")
            created.append(volume)
        if rec("populate", helper_argv(image, [(work, "/work")], POPULATE), LIMITS["helper"], data=pathlib.Path(fixture).read_bytes())["exit"] != 0:
            raise RunError("populating /work failed")
        result["input_commit"], result["input_tree"] = rec.text("populate").split()[-2:]
        if resume_sessions:
            if rec("restore-sessions", helper_argv(image, [(home, "/home/agent")], RESTORE_SESSIONS), LIMITS["helper"],
                   data=pathlib.Path(resume_sessions).read_bytes())["exit"] != 0:
                raise RunError("restoring the Codex sessions failed")
        login_installed = True
        if rec("login", helper_argv(image, [(home, "/home/agent")], INSTALL_LOGIN), LIMITS["helper"], data=login, secret_input=True)["exit"] != 0:
            raise RunError("installing the login failed")
        del login
        started = time.monotonic()
        result["session_started_utc"] = datetime.datetime.now(datetime.timezone.utc).isoformat()
        session_gone = False
        step = rec("session", session_argv(label, image, work, home, prompt, resume_thread), deadline, stdout_file=out / "transcript.jsonl")
        result["session_seconds"] = round(time.monotonic() - started, 3)
        result["session_exit"], result["session_timed_out"] = step["exit"], step["timed_out"]
        if step["timed_out"]:
            rec("kill", ["kill", label], LIMITS["kill"])
            rec("remove-session", ["rm", "-f", label], LIMITS["kill"])
        # Never touch /work while the session container could still run.
        session_gone = session_absent(rec, label, "session-container")
        if not session_gone and step["timed_out"]:
            rec("remove-session-again", ["rm", "-f", label], LIMITS["kill"])
            session_gone = session_absent(rec, label, "session-container-again")
        if not session_gone:
            raise RunError("the session container is still present; nothing was captured")
        if rec("logout", helper_argv(image, [(home, "/home/agent")], REMOVE_LOGIN), LIMITS["helper"])["exit"] == 0:
            login_installed = False
        rec("status", helper_argv(image, [(work, "/work")], STATUS), LIMITS["helper"])
        rec("raw-tar", helper_argv(image, [(work, "/work")], RAW_TAR), LIMITS["helper"], stdout_file=out / "work-raw.tar")
        capture = rec("capture", helper_argv(image, [(work, "/work")], CAPTURE), LIMITS["helper"], stdout_file=out / "result.bundle")
        if capture["exit"] == 0:
            result["result_commit"], result["result_tree"] = rec.text("capture", "stderr").split()[-2:]
        rec("codex-sessions", helper_argv(image, [(home, "/home/agent")], SESSIONS_TAR), LIMITS["helper"], stdout_file=out / "codex-sessions.tar")
        result["transcript"] = summarize(out / "transcript.jsonl")
        result["thread_id"] = result["transcript"]["thread_id"] or resume_thread
        technical = (step["exit"] == 0 and not step["timed_out"] and capture["exit"] == 0
                     and result["transcript"]["turn_completed"] and not result["transcript"]["failures"])
        result["status"] = "completed" if technical else "incomplete"
    except (RunError, OSError, ValueError) as exc:
        result["status"] = "incomplete"
        result["error"] = str(exc)
    finally:
        if login_installed:
            rec("logout-final", helper_argv(image, [(home, "/home/agent")], REMOVE_LOGIN), LIMITS["helper"])
        if session_gone:
            for volume in created:
                rec("remove-" + volume[len(label) + 1:], ["volume", "rm", volume], LIMITS["kill"])
        try:
            residue = census(rec, "census-after", filter_label)
        except RunError as exc:
            residue = ["unknown: " + str(exc)]
        result["residue"] = residue
        if residue:
            result["status"] = "incomplete"
            result.setdefault("error", "resources remain: " + ", ".join(residue))
        result["login_sha256_after"] = sha256_file(login_path) if pathlib.Path(login_path).exists() else None
        if result.get("login_sha256_before") and result["login_sha256_after"] != result["login_sha256_before"]:
            result["status"] = "incomplete"
            result.setdefault("error", "the login file changed")
        # A secret in the outputs quarantines the evidence; it does not change the outcome.
        result["secret_in_outputs"], result["secret_scan_unreadable"] = files_containing(out, secrets)
        result["quarantine"] = bool(result["secret_in_outputs"] or result["secret_scan_unreadable"])
        del secrets
        result["ended_utc"] = datetime.datetime.now(datetime.timezone.utc).isoformat()
        result["steps"] = rec.steps
        (out / "result.json").write_text(json.dumps(result, indent=2), encoding="utf-8")
        files = sorted(p for p in out.rglob("*") if p.is_file() and p.name != "files.sha256")
        (out / "files.sha256").write_text("".join(f"{sha256_file(p)}  {p.relative_to(out).as_posix()}\n" for p in files), encoding="utf-8")
    return result


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    sub = p.add_subparsers(dest="command", required=True)
    f = sub.add_parser("fixture")
    f.add_argument("kind", choices=["new", "continue", "root"])
    f.add_argument("--out", required=True)
    f.add_argument("--message", required=True)
    f.add_argument("--requirements")
    f.add_argument("--from", dest="source")
    f.add_argument("--rev")
    f.add_argument("--drop", action="append", default=[])
    s = sub.add_parser("same-tree")
    for name in ("--a", "--a-rev", "--b", "--b-rev"):
        s.add_argument(name, required=True)
    s.add_argument("--exclude", action="append", default=[])
    r = sub.add_parser("run")
    for name in ("--fixture", "--image", "--login", "--out", "--label"):
        r.add_argument(name, required=True)
    r.add_argument("--engine", default=os.environ.get("MARKITECT_ENGINE") or shutil.which("docker") or "docker")
    r.add_argument("--deadline-seconds", type=int, default=LIMITS["deadline"])
    r.add_argument("--resume-sessions", help="codex-sessions.tar of the chain's previous step")
    r.add_argument("--resume-thread", help="thread UUID of the chain's previous step")
    r.add_argument("--lock", help="exclusive engine lock file shared with the arm-M controller")
    a = p.parse_args(argv)
    if a.command == "fixture":
        if a.kind == "new" and not a.requirements:
            p.error("fixture new needs --requirements")
        print(json.dumps(build_fixture(a.kind, a.out, a.message, a.requirements, a.source, a.rev, a.drop), indent=2))
        return 0
    if a.command == "same-tree":
        result = same_tree(a.a, a.a_rev, a.b, a.b_rev, a.exclude)
        print(json.dumps(result, indent=2))
        return 0 if result["equal"] else 1
    if not re.fullmatch(r"sha256:[0-9a-f]{64}", a.image):
        p.error("--image must be an engine image Id")
    with native_lock(a.lock, "comparison arm B", a.label):
        result = run_session([a.engine], a.fixture, a.image, a.login, a.out, a.label, a.deadline_seconds,
                             a.resume_sessions, a.resume_thread)
    print(json.dumps({k: result.get(k) for k in ("label", "status", "error", "session_seconds", "result_commit", "thread_id", "quarantine")}, indent=2))
    return 0 if result["status"] == "completed" else 1


if __name__ == "__main__":
    sys.exit(main())
