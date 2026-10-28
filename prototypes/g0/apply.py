#!/usr/bin/env python3
"""G0 prototype Apply on the Library fixture: the fixed workflow over host.py Runs.

Assessment (base T) -> Implement (base T) -> Verify (base candidate C)
[-> one correction Implement (base C, Verify Findings) -> Verify] -> publish.
Publication writes one commit (parent T, tree of the approved candidate) onto
the local target branch with compare-and-swap, or nothing. Every Run is a host
start and stop; nothing talks to Managers in between. Throwaway prototype code.

  python apply.py APPLYSPEC.json
APPLYSPEC: markitect (repo), image, login, model, effort, workdir, evidence,
  tools (dir with a linux markitect binary), mandates (e.g. ["implement"] for P1),
  messageBudget, deadlineSeconds, tokenLimit, maxTurns
"""
import json
import os
import shutil
import subprocess
import sys
import time

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import host  # noqa: E402

DEPS = ["--dependency", ".markitect/dependencies/core/model.yaml",
        "--dependency", ".markitect/dependencies/software/model.yaml"]
TASKS = {
    "assessment": "Assessment of the checked Library model: judge its consistency, sufficiency and realizability for your "
                  "area with the full hierarchy, without implementing. Hand off approve or reject with Findings, or blocker.",
    "implement": "Implement the checked Library model in this repository, as your mandate text describes. The base is "
                 "{base}. Hand off the completed, integrated candidate.",
    "verify": "Verify the fixed candidate {base} against the checked model with the full hierarchy, without repairing it. "
              "Its changes against the command's bound input {origin}:\n\n{diff}",
}


def git(repo, *a):
    return subprocess.run(["git", "-C", repo, *a], check=True, capture_output=True, text=True).stdout.strip()


def harness(spec, fixture, mandate, base, origin, findings, out):
    os.makedirs(out, exist_ok=True)
    fpath = ""
    if findings:
        fpath = os.path.join(out, "findings.json")
        with open(fpath, "w") as f:
            json.dump(findings, f)
    git(fixture, "checkout", "-q", "--detach", base)
    cmd = ["go", "run", "-tags", "g0proto", "./prototypes/g0/harnessgen", "-repo", fixture, "-mandate", mandate,
           "-commit", origin, "-tree", git(fixture, "rev-parse", origin + "^{tree}"), "-out", out, *DEPS]
    if fpath:
        cmd += ["-findings", fpath]
    subprocess.run(cmd, cwd=spec["markitect"], check=True)
    with open(os.path.join(out, "managers.json")) as f:
        managers = json.load(f)
    with open(os.path.join(HERE, "rules.md")) as f:
        rules = f.read()
    name = lambda pkg: "root" if pkg == "." else pkg
    result = []
    for m in managers:
        with open(m["harness"]) as f:
            developer = f.read()
        text = rules.format(name=name(m["package"]), parent=name(m["parent"]) if m["parent"] else "none (you are Root)",
                            children=", ".join(name(c) for c in (m["children"] or [])) or "none")
        file = m["harness"].replace(".developer.md", ".harness.md")
        with open(file, "w", newline="\n") as f:
            f.write(text + "\n\n" + developer)
        result.append({"name": name(m["package"]), "parent": name(m["parent"]) if m["parent"] else None,
                       "harness": file})
    return result


def run_once(spec, fixture, mandate, base, origin, findings, n, left):
    rid = f"{spec['label']}-{n}-{mandate}"
    ev = os.path.join(spec["evidence"], f"{n}-{mandate}")
    managers = harness(spec, fixture, mandate, base, origin, findings, os.path.join(ev, "harness"))
    git(fixture, "branch", "-f", "runbase", base)
    bundle = os.path.join(ev, "start.bundle")
    git(fixture, "bundle", "create", "-q", bundle, "runbase")
    diff = git(fixture, "diff", "--stat", origin, base) + "\n\n" + git(fixture, "diff", origin, base) if mandate == "verify" else ""
    task = TASKS[mandate].format(base=base, origin=origin, diff=diff[:200000])
    if findings:
        task += "\n\nFindings from the previous Verify to correct:\n" + json.dumps(findings, indent=1)
    checks = ["cd realization && go build ./cmd/library && go test ./...",
              "/mk/tools/markitect check --repo . --revision HEAD " + " ".join(DEPS) + " > /dev/null"]
    rspec = {"runId": rid, "evidence": ev, "image": spec["image"], "login": spec["login"], "model": spec["model"],
             "effort": spec["effort"], "phase": mandate, "base": base, "baseRef": "refs/heads/runbase",
             "startBundle": bundle, "rootTask": task, "checkCommands": checks, "tools": spec["tools"],
             "messageBudget": left["messages"], "deadlineSeconds": int(left["deadline"] - time.time()),
             "tokenLimit": left["tokens"], "maxTurns": spec.get("maxTurns", 40), "managers": managers}
    with open(os.path.join(ev, "runspec.json"), "w") as f:
        json.dump({k: v for k, v in rspec.items() if k != "rootTask"}, f, indent=1)
    r = host.run(rspec)
    left["messages"] -= r.get("messages") or 0
    left["tokens"] -= r.get("usage", {}).get("total_tokens", 0)
    final = r.get("final") or {}
    if final.get("commit"):
        git(fixture, "fetch", "-q", os.path.join(ev, "run.git"), f"+{final['commit']}:refs/mk/{n}-{mandate}")
    return r


def main(path):
    with open(path) as f:
        spec = json.load(f)
    spec.setdefault("label", time.strftime("g%m%d%H%M"))
    os.makedirs(spec["evidence"], exist_ok=True)
    fixture = os.path.join(spec["workdir"], "fixture")
    target = os.path.join(spec["workdir"], "target.git")
    T = subprocess.run([sys.executable, os.path.join(HERE, "prepare_library.py"), spec["markitect"], fixture],
                       check=True, capture_output=True, text=True).stdout.strip()
    subprocess.run(["git", "clone", "-q", "--bare", fixture, target], check=True)
    started = time.time()
    left = {"messages": spec["messageBudget"], "deadline": started + spec["deadlineSeconds"], "tokens": spec["tokenLimit"]}
    runs, outcome, candidate, n = [], None, None, 0
    plan = spec.get("mandates", ["assessment", "implement", "verify"])
    queue = [(m, T, None) for m in plan[:2]] if "assessment" in plan else [("implement", T, None)]
    corrections = 0
    while queue:
        mandate, base, findings = queue.pop(0)
        n += 1
        r = run_once(spec, fixture, mandate, base, T, findings, n, left)
        final = r.get("final") or {}
        runs.append({"n": n, "mandate": mandate, "base": base, **{k: r.get(k) for k in ("final", "messages", "refusals", "seconds", "usage")}})
        if final.get("outcome") != "completed":
            outcome = {"outcome": final.get("outcome", "failed"), "reason": final.get("reason") or final.get("text")}
            break
        if mandate == "assessment" and final.get("judgment") != "approve":
            outcome = {"outcome": "no_candidate", "reason": "Assessment did not approve"}
            break
        if mandate == "implement":
            candidate = final["commit"]
            if "verify" in plan:
                queue.append(("verify", candidate, None))
            else:
                outcome = {"outcome": "candidate", "commit": candidate}
        if mandate == "verify":
            if final.get("judgment") == "approve":
                outcome = {"outcome": "verified", "commit": base}
            elif corrections < 1:
                corrections += 1
                fnd = [{"manager": "root", "text": f.get("text", "")} for f in final.get("findings", [])] or \
                      [{"manager": "root", "text": final.get("text", "Verify rejected")}]
                queue += [("implement", base, fnd)]
            else:
                outcome = {"outcome": "no_candidate", "reason": "Verify rejected after one correction"}
    published = None
    if outcome and outcome.get("outcome") == "verified":
        tree = git(fixture, "rev-parse", outcome["commit"] + "^{tree}")
        env = {**os.environ, "GIT_AUTHOR_NAME": "Markitect", "GIT_AUTHOR_EMAIL": "apply@markitect.invalid",
               "GIT_COMMITTER_NAME": "Markitect", "GIT_COMMITTER_EMAIL": "apply@markitect.invalid"}
        git(target, "fetch", "-q", fixture, f"{outcome['commit']}:refs/mk/approved")
        pub = subprocess.run(["git", "-C", target, "commit-tree", tree, "-p", T, "-m",
                              f"Markitect Apply {spec['label']}: verified candidate"], env=env,
                             check=True, capture_output=True, text=True).stdout.strip()
        cas = subprocess.run(["git", "-C", target, "update-ref", "refs/heads/dev/library-m1", pub, T],
                             capture_output=True, text=True)
        published = {"commit": pub, "tree": tree, "ok": cas.returncode == 0, "error": cas.stderr.strip()}
    summary = {"label": spec["label"], "base": T, "outcome": outcome, "published": published, "runs": runs,
               "seconds": round(time.time() - started, 1),
               "messages": spec["messageBudget"] - left["messages"],
               "tokens": spec["tokenLimit"] - left["tokens"]}
    with open(os.path.join(spec["evidence"], "apply.json"), "w") as f:
        json.dump(summary, f, indent=1)
    print(json.dumps(summary, indent=1))


if __name__ == "__main__":
    main(sys.argv[1])
