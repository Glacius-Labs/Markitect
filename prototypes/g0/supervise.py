#!/usr/bin/env python3
"""Per-container supervisor of one Manager (design section 6, "Keeping going").

Clones the Run remote into /work, then keeps the Manager going: while the
referee says this Manager has open work, it starts a provider response that
names what arrived; when nothing is open it waits on the referee. After every
response it pushes uncommitted work to the private mk/<name>/wip ref. It holds
no rules: the referee decides what is open. Throwaway prototype code.

Environment: MK_NAME, MK_TOKEN, MK_API, MK_MODEL, MK_EFFORT, MK_MAX_TURNS.
Mounts: /mk/bin (tools), /mk/harness.md, /mk/login/auth.json, /out (evidence).
"""
import base64
import json
import os
import shutil
import subprocess
import sys
import tempfile
import time
import urllib.request

NAME, TOKEN, API = os.environ["MK_NAME"], os.environ["MK_TOKEN"], os.environ["MK_API"]
WORK, OUT = "/work", "/out"
CODEX_HOME = os.path.expanduser("~/.codex")


def log(text):
    with open(os.path.join(OUT, "supervisor.log"), "a") as f:
        f.write(f"{time.strftime('%H:%M:%S')} {text}\n")


def api(path, timeout=30):
    req = urllib.request.Request(API + path)
    req.add_header("Authorization", "Basic " + base64.b64encode(f"{NAME}:{TOKEN}".encode()).decode())
    with urllib.request.urlopen(req, timeout=timeout) as r:
        return json.load(r)


def git(*args, env=None):
    return subprocess.run(["git", "-C", WORK, *args], capture_output=True, text=True,
                          env={**os.environ, **(env or {})})


def setup():
    os.makedirs(CODEX_HOME, exist_ok=True)
    shutil.copy("/mk/login/auth.json", os.path.join(CODEX_HOME, "auth.json"))
    os.chmod(os.path.join(CODEX_HOME, "auth.json"), 0o600)
    bindir = os.path.expanduser("~/bin")
    os.makedirs(bindir, exist_ok=True)
    with open(os.path.join(bindir, "mk"), "w") as f:
        f.write('#!/bin/sh\nexec python3 /mk/bin/mk "$@"\n')
    os.chmod(os.path.join(bindir, "mk"), 0o755)
    os.environ["PATH"] = bindir + ":" + os.environ["PATH"]
    helper = '!f() { echo "username=$MK_NAME"; echo "password=$MK_TOKEN"; }; f'
    subprocess.run(["git", "config", "--global", "credential.helper", helper], check=True)
    subprocess.run(["git", "config", "--global", "user.name", NAME], check=True)
    subprocess.run(["git", "config", "--global", "user.email", f"{NAME.replace('/', '-')}@run.invalid"], check=True)
    remote = API + "/git/run.git"
    for attempt in range(30):
        if subprocess.run(["git", "clone", "-q", remote, WORK], capture_output=True).returncode == 0:
            break
        time.sleep(2)
    else:
        log("clone failed")
        sys.exit(1)
    git("fetch", "-q", "origin", f"+refs/heads/mk/{NAME}/work:refs/remotes/origin/mk/{NAME}/work")
    git("checkout", "-q", "-B", "work", f"refs/remotes/origin/mk/{NAME}/work")
    log("clone ready")


def push_wip():
    """Save uncommitted work as a private, forward-only wip commit (evidence, never a handoff)."""
    with tempfile.TemporaryDirectory() as d:
        env = {"GIT_INDEX_FILE": os.path.join(d, "index")}
        if git("read-tree", "HEAD", env=env).returncode != 0:
            return
        git("add", "-A", env=env)
        tree = git("write-tree", env=env).stdout.strip()
    ref = f"refs/heads/mk/{NAME}/wip"
    git("fetch", "-q", "origin", f"+{ref}:refs/mk/wip")
    prev = git("rev-parse", "-q", "--verify", "refs/mk/wip").stdout.strip()
    parents = ["-p", "HEAD"] + (["-p", prev] if prev else [])
    c = git("commit-tree", tree, *parents, "-m", f"wip of {NAME}").stdout.strip()
    r = git("push", "-q", "origin", f"{c}:{ref}")
    if r.returncode != 0:
        log(f"wip push refused: {r.stderr.strip()[-300:]}")


def turn(prompt, session):
    if not session:
        with open("/mk/harness.md") as f:
            prompt = f.read() + "\n\n---\n\n" + prompt
    common = ["--json", "--model", os.environ["MK_MODEL"], "-c", f"model_reasoning_effort={os.environ['MK_EFFORT']}",
              "-c", "project_doc_max_bytes=0",
              "--dangerously-bypass-approvals-and-sandbox", "--skip-git-repo-check"]
    if session:
        cmd = ["codex", "exec", "resume", session, *common, prompt]
    else:
        cmd = ["codex", "exec", *common, "-C", WORK, prompt]
    started = time.time()
    with open(os.path.join(OUT, "transcript.jsonl"), "a") as t, open(os.path.join(OUT, "codex.err"), "a") as e:
        p = subprocess.Popen(cmd, cwd=WORK, stdout=subprocess.PIPE, stderr=e, text=True)
        usage = {}
        for line in p.stdout:
            t.write(line)
            try:
                ev = json.loads(line)
            except ValueError:
                continue
            if ev.get("type") == "thread.started":
                session = session or ev.get("thread_id")
            if ev.get("type") == "turn.completed":
                for k, v in (ev.get("usage") or {}).items():
                    usage[k] = usage.get(k, 0) + v
        p.wait()
    log(f"turn exit {p.returncode} in {time.time() - started:.0f}s usage {usage}")
    with open(os.path.join(OUT, "usage.jsonl"), "a") as u:
        u.write(json.dumps({"seconds": round(time.time() - started, 1), "exit": p.returncode, **usage}) + "\n")
    return session


def describe(events, state):
    lines = []
    for e in events:
        lines.append(json.dumps(e, indent=1))
    if state.get("owesHandoff"):
        lines.append("You still owe a handoff for your current task (mk hand-off).")
    return "Events for you:\n" + "\n".join(lines) if lines else "Continue your current task."


def main():
    setup()
    session, turns, max_turns = None, 0, int(os.environ.get("MK_MAX_TURNS", "40"))
    while turns < max_turns:
        state = api("/open")
        if state["ended"]:
            break
        events = api("/wait?timeout=0" if state["open"] else "/wait?timeout=300", timeout=320)
        if any(e.get("kind") == "run_end" for e in events):
            break
        if not events and not api("/open")["open"]:
            continue
        session = turn(describe(events, api("/open")), session)
        turns += 1
        push_wip()
    push_wip()
    log(f"supervisor done after {turns} turns")


if __name__ == "__main__":
    main()
