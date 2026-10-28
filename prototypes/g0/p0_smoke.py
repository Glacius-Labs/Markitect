#!/usr/bin/env python3
"""P0: container smoke of the G0 prototype with scripted fake Managers, no provider.

Starts the referee with a tiny Go project as the Run remote, three fake Manager
containers (root, a, b) that run only the team tools, plays one Implement Run
through, including deliberate refusals, and checks the referee's score. Removes
everything by label at the end.

  python p0_smoke.py IMAGE EVIDENCE_DIR
"""
import json
import os
import shutil
import subprocess
import sys
import tempfile
import time

HERE = os.path.dirname(os.path.abspath(__file__))
LABEL = f"p0-{int(time.time())}"
NET = f"mkp-{LABEL}"
TOKENS = {"root": "t-root-0001", "a": "t-a-0002", "b": "t-b-0003"}
results = []


def sh(*args, check=True, input=None):
    p = subprocess.run(list(args), capture_output=True, text=True, input=input)
    if check and p.returncode != 0:
        raise RuntimeError(f"{' '.join(args)}\n{p.stdout}\n{p.stderr}")
    return p


def mk(who, *args, expect_ok=True):
    p = sh("docker", "exec", "-w", "/work", f"{NET}-{who}", "python3", "/mk/bin/mk", *args, check=False)
    out = (p.stdout + p.stderr).strip()
    return p.returncode == 0, out


def ex(who, script):
    return sh("docker", "exec", "-w", "/work", f"{NET}-{who}", "sh", "-c", script, check=False)


def expect(name, cond, detail=""):
    results.append({"case": name, "pass": bool(cond), "detail": detail[-400:]})
    print(("PASS " if cond else "FAIL ") + name + ("" if cond else f"\n  {detail[-400:]}"), flush=True)


def wait_for(who, pred, timeout=120):
    end = time.time() + timeout
    seen = []
    while time.time() < end:
        ok, out = mk(who, "wait", "--timeout", "10")
        if ok and out and not out.startswith("nothing arrived"):
            events = json.loads("[" + out.replace("}\n{", "},\n{") + "]")
            seen += events
            if any(pred(e) for e in events):
                return seen
    return seen


def main(image, evidence):
    os.makedirs(evidence, exist_ok=True)
    work = tempfile.mkdtemp(prefix="p0-")
    proj, rundir, state = (os.path.join(work, d) for d in ("proj", "run", "state"))
    os.makedirs(proj), os.makedirs(rundir), os.makedirs(state)
    g = lambda *a: sh("git", "-C", proj, *a)
    g("init", "-q", "-b", "main")
    g("config", "user.name", "p0"), g("config", "user.email", "p0@run.invalid"), g("config", "core.autocrlf", "false")
    with open(os.path.join(proj, "go.mod"), "w", newline="\n") as f:
        f.write("module example.com/p0\n\ngo 1.22\n")
    with open(os.path.join(proj, "p0_test.go"), "w", newline="\n") as f:
        f.write("package p0\n\nimport \"testing\"\n\nfunc TestBase(t *testing.T) {}\n")
    g("add", "."), g("commit", "-q", "-m", "base")
    base = g("rev-parse", "HEAD").stdout.strip()
    g("bundle", "create", "-q", os.path.join(rundir, "start.bundle"), "main")
    cfg = {"phase": "implement", "base": base, "baseRef": "refs/heads/main", "rootTask": "P0 smoke task",
           "checkCommands": ["go vet ./... && go test ./..."], "messageBudget": 40, "deadlineSeconds": 900,
           "managers": [{"name": "root", "parent": None}, {"name": "a", "parent": "root"},
                        {"name": "b", "parent": "root"}],
           "credentials": {t: n for n, t in TOKENS.items()}}
    with open(os.path.join(rundir, "config.json"), "w") as f:
        json.dump(cfg, f)
    try:
        sh("docker", "network", "create", "--label", f"mkproto={LABEL}", NET)
        sh("docker", "run", "-d", "--name", f"{NET}-referee", "--network", NET, "--network-alias", "referee",
           "--label", f"mkproto={LABEL}", "--entrypoint", "python3", "-v", f"{rundir}:/run:ro",
           "-v", f"{HERE}:/mk/bin:ro", "-v", f"{state}:/state", image, "/mk/bin/referee.py")
        for _ in range(30):
            if os.path.exists(os.path.join(state, "score.json")):
                break
            time.sleep(1)
        expect("referee starts and seeds the Run remote", os.path.exists(os.path.join(state, "score.json")),
               sh("docker", "logs", f"{NET}-referee", check=False).stderr)
        for n, t in TOKENS.items():
            sh("docker", "run", "-d", "--name", f"{NET}-{n}", "--network", NET, "--label", f"mkproto={LABEL}",
               "--entrypoint", "sleep", "-e", f"MK_NAME={n}", "-e", f"MK_TOKEN={t}", "-e", "MK_API=http://referee:8080",
               "-e", 'MK_CHECK=["go vet ./... && go test ./..."]', "-v", f"{HERE}:/mk/bin:ro", image, "infinity")
            setup = ('git config --global credential.helper \'!f() { echo "username=$MK_NAME"; echo "password=$MK_TOKEN"; }; f\' && '
                     'git config --global user.name "$MK_NAME" && git config --global user.email "$MK_NAME@run.invalid" && '
                     'git clone -q http://referee:8080/git/run.git /tmp/c && cp -a /tmp/c/. /work/ && cd /work && '
                     'git fetch -q origin "+refs/heads/mk/$MK_NAME/work:refs/remotes/origin/mk/$MK_NAME/work" && '
                     'git checkout -q -B work "refs/remotes/origin/mk/$MK_NAME/work"')
            p = sh("docker", "exec", f"{NET}-{n}", "sh", "-c", setup, check=False)
            expect(f"{n} clones the Run remote with its own credential", p.returncode == 0, p.stderr)

        p = ex("a", "git config --global credential.helper '!f() { echo username=a; echo password=wrong; }; f' && "
                    "git ls-remote origin 2>&1; git config --global credential.helper "
                    "'!f() { echo \"username=$MK_NAME\"; echo \"password=$MK_TOKEN\"; }; f'")
        expect("a wrong credential is refused", "Authentication failed" in p.stdout + p.stderr or "401" in p.stdout,
               p.stdout + p.stderr)

        ok, out = mk("root", "send", "a", "task", "add package a")
        expect("root sends a task to child a", ok, out)
        ok, out = mk("root", "send", "b", "task", "add package b")
        expect("root sends a task to child b", ok, out)
        ok, out = mk("a", "send", "b", "question", "hi sibling")
        expect("a message to a sibling is refused", not ok and "may write only to its parent or direct children" in out, out)
        p = ex("a", "echo x > x.txt && git add x.txt && git commit -qm x && git push origin HEAD:refs/heads/mk/b/work 2>&1")
        expect("a push to another Manager's ref is refused", p.returncode != 0 and "refused by the referee" in p.stdout + p.stderr,
               p.stdout + p.stderr)
        ex("a", "git reset -q --hard HEAD~1")
        p = ex("a", "git push origin HEAD:refs/heads/mk/base 2>&1")
        expect("a push to mk/base is refused", p.returncode != 0, p.stdout + p.stderr)

        events = wait_for("a", lambda e: e.get("kind") == "task")
        expect("a receives its task through wait", any(e.get("kind") == "task" for e in events), json.dumps(events))
        ex("a", "mkdir -p a && printf 'package a\\n\\nfunc A() int { return 1 }\\n' > a/a.go")
        ok, out = mk("a", "hand-off", "package a done")
        expect("a hands off", ok, out)
        ex("b", "mkdir -p b && printf 'package b\\n\\nfunc B() int { return }\\n' > b/b.go")
        ok, out = mk("b", "hand-off", "package b (broken)")
        expect("b hands off broken code", ok, out)

        ok, out = mk("root", "hand-off", "too early")
        expect("root's handoff before accepting children is refused", not ok and "no accepted latest handoff" in out, out)
        events = wait_for("root", lambda e: e.get("kind") == "handoff" and e.get("from") == "b" and e.get("check") in ("pass", "fail"))
        events += wait_for("root", lambda e: False, timeout=5)
        checks = {e["from"]: e["check"] for e in events if e.get("kind") == "handoff"}
        expect("root sees a's handoff with a passed check", checks.get("a") == "pass", json.dumps(events))
        expect("root sees b's handoff with a failed check", checks.get("b") == "fail", json.dumps(events))
        ok, out = mk("root", "review", "b", "accept")
        expect("accepting a failed handoff is refused", not ok and "failed its check" in out, out)
        ok, out = mk("root", "review", "b", "reject", "--finding", "b.go does not compile", "fix it")
        expect("root rejects b with a Finding", ok, out)
        ok, out = mk("root", "review", "a", "accept")
        expect("root accepts a", ok, out)
        ok, out = mk("root", "take", "a")
        expect("root takes a's accepted handoff", ok, out)
        ok, out = mk("root", "take", "b")
        expect("root cannot take b's rejected handoff", not ok and "not accepted" in out, out)

        ok, out = mk("root", "hand-off", "push integration so far")
        expect("root's handoff is refused while b is not accepted", not ok, out)
        ex("root", "git push -q origin HEAD:refs/heads/mk/root/work")
        events = wait_for("b", lambda e: e.get("kind") == "review")
        expect("b receives the reject with its Finding", any(e.get("kind") == "review" and e.get("accept") is False for e in events),
               json.dumps(events))
        ex("b", "printf 'package b\\n\\nfunc B() int { return 2 }\\n' > b/b.go")
        ok, out = mk("b", "hand-off", "fixed without sync")
        expect("b's correction without the parent's current state is refused", not ok and "merge it first" in out, out)
        ok, out = mk("b", "sync")
        expect("b syncs the parent's current work", ok, out)
        ok, out = mk("b", "hand-off", "fixed")
        expect("b hands off the correction", ok, out)
        events = wait_for("root", lambda e: e.get("kind") == "handoff" and e.get("from") == "b" and e.get("check") == "pass")
        expect("root sees b's corrected handoff pass", any(e.get("from") == "b" and e.get("check") == "pass" for e in events),
               json.dumps(events))
        ok, out = mk("root", "review", "b", "accept")
        expect("root accepts b", ok, out)
        ok, out = mk("root", "take", "b")
        expect("root takes b", ok, out)
        ok, out = mk("root", "hand-off", "integrated a and b")
        expect("root hands off the integrated result", ok, out)
        final = None
        for _ in range(60):
            with open(os.path.join(state, "score.json")) as f:
                final = json.load(f).get("final")
            if final:
                break
            time.sleep(2)
        expect("the referee reports completed", final and final.get("outcome") == "completed", json.dumps(final))
        if final and final.get("commit"):
            files = sh("docker", "exec", f"{NET}-referee", "git", "-C", "/srv/run.git", "ls-tree", "-r", "--name-only",
                       final["commit"]).stdout.split()
            expect("the final commit holds a and b", "a/a.go" in files and "b/b.go" in files, " ".join(files))
        ok, out = mk("a", "send", "root", "question", "after the end")
        expect("pushes after the end are refused", not ok and "has ended" in out, out)
        sh("docker", "cp", f"{NET}-referee:/srv/run.git", os.path.join(evidence, "run.git"), check=False)
    finally:
        ids = sh("docker", "ps", "-aq", "--filter", f"label=mkproto={LABEL}", check=False).stdout.split()
        if ids:
            sh("docker", "rm", "-f", *ids, check=False)
        sh("docker", "network", "rm", NET, check=False)
        left = sh("docker", "ps", "-aq", "--filter", f"label=mkproto={LABEL}", check=False).stdout.split()
        expect("cleanup leaves no container", not left, " ".join(left))
        for n in ("score.json", "events.jsonl"):
            if os.path.exists(os.path.join(state, n)):
                shutil.copy(os.path.join(state, n), evidence)
        if os.path.isdir(os.path.join(state, "checks")):
            shutil.copytree(os.path.join(state, "checks"), os.path.join(evidence, "checks"), dirs_exist_ok=True)
        with open(os.path.join(evidence, "p0-results.json"), "w") as f:
            json.dump(results, f, indent=1)
    failed = [r for r in results if not r["pass"]]
    print(f"\n{len(results) - len(failed)}/{len(results)} passed")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1], sys.argv[2]))
