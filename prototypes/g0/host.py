#!/usr/bin/env python3
"""G0 prototype host for one Run: start, then stop. Nothing in between.

Start: create the Run network, the referee with the seeded Run remote, and one
container per Manager. Stop: wait for the referee's outcome (or the hard
deadline), collect evidence, remove everything that carries the Run's label.
Runs on Linux (WSL) with the docker CLI. Throwaway prototype code.

  python3 host.py RUNSPEC.json      prints the Run result as JSON
"""
import json
import os
import re
import secrets
import shutil
import subprocess
import sys
import tempfile
import time

HERE = os.path.dirname(os.path.abspath(__file__))
PATH = "/mk/bin:/usr/local/go/bin:/home/agent/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"


def sh(*args, check=True, capture=True):
    p = subprocess.run(list(args), capture_output=capture, text=True)
    if check and p.returncode != 0:
        raise RuntimeError(f"{' '.join(args)}: {p.stderr.strip()}")
    return (p.stdout or "").strip()


def slug(name):
    return re.sub(r"[^a-z0-9]+", "-", name.lower()).strip("-") or "root"


def cleanup(label, run_net):
    ids = sh("docker", "ps", "-aq", "--filter", f"label=mkproto={label}", check=False).split()
    if ids:
        sh("docker", "rm", "-f", *ids, check=False)
    sh("docker", "network", "rm", run_net, check=False)
    left = sh("docker", "ps", "-aq", "--filter", f"label=mkproto={label}", check=False).split()
    nets = sh("docker", "network", "ls", "-q", "--filter", f"label=mkproto={label}", check=False).split()
    return {"containersLeft": len(left), "networksLeft": len(nets)}


def run(spec):
    rid = spec["runId"]
    label, net = rid, f"mkp-{rid}"
    ev = spec["evidence"]
    os.makedirs(os.path.join(ev, "referee"), exist_ok=True)
    rundir = tempfile.mkdtemp(prefix=f"mkp-{rid}-")
    logindir = tempfile.mkdtemp(prefix=f"mkp-{rid}-login-")
    os.chmod(logindir, 0o755)
    shutil.copy(spec["login"], os.path.join(logindir, "auth.json"))
    os.chmod(os.path.join(logindir, "auth.json"), 0o644)
    creds = {m["name"]: secrets.token_hex(16) for m in spec["managers"]}
    cfg = {k: spec[k] for k in ("phase", "base", "rootTask", "checkCommands", "messageBudget", "deadlineSeconds")}
    cfg["checkTimeout"] = spec.get("checkTimeout", 300)
    cfg["managers"] = [{"name": m["name"], "parent": m["parent"]} for m in spec["managers"]]
    cfg["credentials"] = {t: n for n, t in creds.items()}
    with open(os.path.join(rundir, "config.json"), "w") as f:
        json.dump(cfg, f)
    shutil.copy(spec["startBundle"], os.path.join(rundir, "start.bundle"))
    for d in (rundir,):
        os.chmod(d, 0o755)
        for n in os.listdir(d):
            os.chmod(os.path.join(d, n), 0o644)
    started = time.time()
    result = {"runId": rid, "phase": spec["phase"], "base": spec["base"]}
    try:
        sh("docker", "network", "create", "--label", f"mkproto={label}", net)
        sh("docker", "run", "-d", "--name", f"{net}-referee", "--network", net, "--network-alias", "referee",
           "--label", f"mkproto={label}", "--entrypoint", "python3",
           "-v", f"{rundir}:/run:ro", "-v", f"{HERE}:/mk/bin:ro", "-v", f"{os.path.abspath(ev)}/referee:/state",
           spec["image"], "/mk/bin/referee.py")
        for _ in range(60):
            if os.path.exists(os.path.join(ev, "referee", "score.json")):
                break
            time.sleep(1)
        else:
            raise RuntimeError("referee did not start: " + sh("docker", "logs", f"{net}-referee", check=False))
        for m in spec["managers"]:
            out = os.path.abspath(os.path.join(ev, "managers", slug(m["name"])))
            os.makedirs(out, exist_ok=True)
            sh("docker", "run", "-d", "--name", f"{net}-{slug(m['name'])}", "--network", net,
               "--label", f"mkproto={label}", "--entrypoint", "python3",
               "-e", f"MK_NAME={m['name']}", "-e", f"MK_TOKEN={creds[m['name']]}", "-e", "MK_API=http://referee:8080",
               "-e", f"MK_MODEL={spec['model']}", "-e", f"MK_EFFORT={spec['effort']}",
               "-e", f"MK_MAX_TURNS={spec.get('maxTurns', 40)}", "-e", f"MK_CHECK={json.dumps(spec['checkCommands'])}",
               "-e", f"PATH={PATH}",
               "-v", f"{HERE}:/mk/bin:ro", "-v", f"{os.path.abspath(m['harness'])}:/mk/harness.md:ro",
               "-v", f"{logindir}:/mk/login:ro", "-v", f"{out}:/out",
               spec["image"], "/mk/bin/supervise.py")
        result["started"] = round(time.time() - started, 1)
        hard = started + spec["deadlineSeconds"] + 120
        score = {}
        while time.time() < hard:
            time.sleep(10)
            try:
                with open(os.path.join(ev, "referee", "score.json")) as f:
                    score = json.load(f)
            except (OSError, ValueError):
                continue
            if score.get("final"):
                break
            running = sh("docker", "ps", "-q", "--filter", f"label=mkproto={label}", check=False).split()
            if len(running) <= 1:
                score.setdefault("final", None)
                score["final"] = {"outcome": "failed", "reason": "every Manager supervisor stopped"}
                break
        else:
            score["final"] = {"outcome": "incomplete", "reason": "host hard deadline"}
        result["final"] = score.get("final")
        result["messages"] = score.get("messages")
        result["refusals"] = len(score.get("refusals", []))
        # give supervisors time to push their wip and exit on run_end
        for _ in range(12):
            if len(sh("docker", "ps", "-q", "--filter", f"label=mkproto={label}", check=False).split()) <= 1:
                break
            time.sleep(5)
        sh("docker", "cp", f"{net}-referee:/srv/run.git", os.path.join(ev, "run.git"), check=False)
        for c in sh("docker", "ps", "-a", "--filter", f"label=mkproto={label}", "--format", "{{.Names}}",
                    check=False).split():
            with open(os.path.join(ev, f"{c}.log"), "w") as f:
                f.write(sh("docker", "logs", c, check=False))
    except Exception as e:
        result["final"] = {"outcome": "failed", "reason": f"host error: {e}"}
    finally:
        result["cleanup"] = cleanup(label, net)
        shutil.rmtree(logindir, ignore_errors=True)
        shutil.rmtree(rundir, ignore_errors=True)
    result["seconds"] = round(time.time() - started, 1)
    usage = {}
    for root, _, files in os.walk(os.path.join(ev, "managers")):
        if "usage.jsonl" in files:
            with open(os.path.join(root, "usage.jsonl")) as f:
                for line in f:
                    for k, v in json.loads(line).items():
                        if k not in ("seconds", "exit") and isinstance(v, int):
                            usage[k] = usage.get(k, 0) + v
    result["usage"] = usage
    with open(os.path.join(ev, "result.json"), "w") as f:
        json.dump(result, f, indent=1)
    return result


if __name__ == "__main__":
    with open(sys.argv[1]) as f:
        print(json.dumps(run(json.load(f)), indent=1))
