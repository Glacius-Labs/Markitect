"""Deterministic steps of the Markitect Claude plugin (comparison arm P).

The plugin's agents judge and realize; this script does everything that has
one right answer, the way the native host does: the structural check before
any agent starts, the Manager tree, compact check results for agents, the
check note at every Manager handoff, the Bash guard and the publication guard.

  python mkp.py prepare --run-id ID
  python mkp.py check [--revision REV]
  python mkp.py context PACKAGE [--revision REV]
  python mkp.py note            (SubagentStop hook, JSON on stdin)
  python mkp.py guard           (PreToolUse Bash hook, JSON on stdin)
  python mkp.py publish --run-id ID --result WORKFLOW_RESULT.json

Environment: MARKITECT_BIN (the markitect binary), MARKITECT_STATE_DIR (run
records, outside the repository), MARKITECT_OWNER_BRANCH (the branch Apply
must not write), MARKITECT_RUN_ID, MARKITECT_CHECK_TIMEOUT (project check
seconds, default 120) and the workflow limits in LIMITS.
"""
import argparse
import json
import os
import pathlib
import re
import shutil
import subprocess
import sys

PROTECTED = (".markitect/model.yaml", ".markitect/model-lock.yaml", ".markitect/model/",
             ".markitect/dependencies/", ".markitect/execution.yaml")
NOTES_REF = "markitect-check"
# Workflow limits, passed through the prepare record (key, environment, default).
LIMITS = (("maxCorrections", "MARKITECT_MAX_CORRECTIONS", 2), ("maxReturns", "MARKITECT_MAX_RETURNS", 2),
          ("maxQuestions", "MARKITECT_MAX_QUESTIONS", 3), ("maxAgents", "MARKITECT_MAX_AGENTS", 150))


def git(repo, *args, check=True):
    r = subprocess.run(["git", "-C", str(repo), *args], capture_output=True, text=True, encoding="utf-8")
    if check and r.returncode != 0:
        raise RuntimeError(f"git {' '.join(args)}: {r.stderr.strip()}")
    return r.stdout.strip()


def toplevel(path="."):
    return pathlib.Path(git(path, "rev-parse", "--show-toplevel"))


def state_dir(run_id):
    root = os.environ.get("MARKITECT_STATE_DIR")
    if not root:
        raise RuntimeError("MARKITECT_STATE_DIR is not set")
    d = pathlib.Path(root) / run_id
    d.mkdir(parents=True, exist_ok=True)
    return d


def slug(package):
    return "_root" if package == "." else package.replace("/", "--")


def dependencies(repo, revision):
    """Every dependency manifest .markitect/dependencies/<name>/model.yaml, in name order."""
    if revision:
        names = git(repo, "ls-tree", "-r", "--name-only", revision, "--", ".markitect/dependencies").splitlines()
    else:
        names = [p.relative_to(repo).as_posix() for p in (repo / ".markitect" / "dependencies").glob("*/model.yaml")]
    found = sorted(n for n in names if re.fullmatch(r"\.markitect/dependencies/[^/]+/model\.yaml", n))
    return [arg for n in found for arg in ("--dependency", n)]


def markitect(verb, repo, revision=None, extra=(), timeout=300):
    binary = os.environ.get("MARKITECT_BIN") or "markitect"
    argv = [binary, verb, *extra, "--repo", str(repo), "--model", ".markitect/model.yaml", *dependencies(repo, revision)]
    if revision:
        argv += ["--revision", revision]
    r = subprocess.run(argv, capture_output=True, text=True, encoding="utf-8", timeout=timeout)
    try:
        report = json.loads(r.stdout)
    except ValueError:
        report = {"status": "not_completed", "error": (r.stdout + r.stderr).strip()[-4000:]}
    return r.returncode, report


def summarize(code, report):
    """What an agent needs from a check: status and every non-empty diagnostic."""
    out = {"exit": code, "status": report.get("status")}
    if report.get("error"):
        out["error"] = report["error"]
    s = (report.get("stages") or {}).get("structural") or {}
    picks = {
        "model": ((s.get("model") or {}).get("diagnostics")),
        "execution": ((s.get("execution") or {}).get("diagnostics")),
        "registry": ((s.get("registry") or {}).get("diagnostics")),
        "registryOrphaned": ((s.get("registry") or {}).get("orphaned")),
        "files": ((s.get("files") or {}).get("diagnostics")),
        "accountingFindings": (((s.get("files") or {}).get("accounting") or {}).get("findings")),
        "stale": (((s.get("files") or {}).get("accounting") or {}).get("stale")),
        "selection": ((s.get("selection") or {}).get("diagnostics")),
    }
    out.update({k: v for k, v in picks.items() if v})
    return out


def stale_or_orphaned(report):
    s = (report.get("stages") or {}).get("structural") or {}
    return bool((((s.get("files") or {}).get("accounting") or {}).get("stale")) or ((s.get("registry") or {}).get("orphaned")))


def nest(managers):
    by = {m["package"]: {"pkg": m["package"], "areas": m["areas"], "children": []} for m in managers}
    for m in managers:
        by[m["package"]]["children"] = [by[c] for c in m["children"]]
    return by["."]


def cmd_prepare(a):
    repo = toplevel()
    dirty = [l for l in git(repo, "status", "--porcelain", "--untracked-files=all").splitlines() if not l[3:].startswith(".claude/")]
    if dirty:
        return fail({"stage": "prepare", "error": "the repository has uncommitted changes", "paths": dirty[:20]})
    base = git(repo, "rev-parse", "HEAD")
    branch = git(repo, "symbolic-ref", "--short", "HEAD")
    code, report = markitect("check", repo, base)
    check = summarize(code, report)
    record = {"runId": a.run_id, "base": base, "baseTree": git(repo, "rev-parse", base + "^{tree}"), "branch": branch,
              "repo": str(repo), "mkp": pathlib.Path(__file__).resolve().as_posix(),
              "stateDir": state_dir(a.run_id).resolve().as_posix(), "check": check}
    for key, env, default in LIMITS:
        record[key] = int(os.environ.get(env, default))
    if code != 0:
        record["error"] = "the model failed the structural check; no agent starts"
        write(state_dir(a.run_id) / "prepare.json", record)
        return fail(record)
    code, tree = markitect("tree", repo, base)
    if code != 0 or not tree.get("managers"):
        record["error"] = "the Manager tree is unavailable"
        record["tree"] = summarize(code, tree)
        write(state_dir(a.run_id) / "prepare.json", record)
        return fail(record)
    execution = ((report.get("stages") or {}).get("structural") or {}).get("execution") or {}
    record["checks"] = ((execution.get("declarations") or {}).get("Checks") or {}).get("Value") or []
    record["managers"] = len(tree["managers"])
    record["tree"] = nest(tree["managers"])
    write(state_dir(a.run_id) / "prepare.json", record)
    print(json.dumps(record, indent=1))
    return 0


def cmd_check(a):
    repo = toplevel()
    code, report = markitect("check", repo, a.revision)
    print(json.dumps(summarize(code, report), indent=1))
    return code


def cmd_context(a):
    repo = toplevel()
    code, report = markitect("context", repo, a.revision, extra=[a.package])
    print(json.dumps({"status": report.get("status"), "error": report.get("error"), "contexts": report.get("contexts")}, indent=1))
    return code


def hook_input():
    try:
        return json.loads(sys.stdin.read() or "{}")
    except ValueError:
        return {}


def cmd_note(_):
    """SubagentStop: attach the deterministic check of a Manager's handoff commit.

    The note travels with the commit to the parent (D11). It never sends the
    agent back and never fails the hook.
    """
    data = hook_input()
    try:
        if not str(data.get("agent_type", "")).startswith("markitect:manager"):
            return 0
        repo = toplevel(data.get("cwd") or ".")
        head = git(repo, "rev-parse", "HEAD")
        code, report = markitect("check", repo, head)
        summary = summarize(code, report)
        summary.update({"commit": head, "branch": git(repo, "rev-parse", "--abbrev-ref", "HEAD", check=False),
                        "agent": data.get("agent_id")})
        git(repo, "notes", f"--ref={NOTES_REF}", "add", "-f", "-m", json.dumps(summary), head)
        run_id = os.environ.get("MARKITECT_RUN_ID", "adhoc")
        write(state_dir(run_id) / "notes" / f"{data.get('agent_id', 'unknown')}.json", summary)
    except Exception as e:  # a hook must not stop the agent organization
        print(f"markitect note skipped: {e}", file=sys.stderr)
    return 0


GUARDS = [
    (r"\bgit\b[^\n;&|]*\bpush\b", "Apply never pushes; hand off your commit instead."),
    (r"\bgit\b[^\n;&|]*\bnotes\b[^\n;&|]*\b(add|append|copy|edit|merge|prune|remove)\b",
     "Check notes are written by the host only; read them with git notes --ref=markitect-check show COMMIT."),
    (r"\bgit\b[^\n;&|]*\b(update-ref|worktree\s+(remove|prune|move))\b", "Refs and worktrees are managed by the host."),
    (r"\bgit\b[^\n;&|]*\bbranch\b[^\n;&|]*\s-(D|f|-force|-delete)\b", "Do not delete or force other branches; commit on your own branch."),
    (r"markitect/apply/", "Publication branches are written by the publication guard only."),
    (r"\bmarkitect(\.exe)?\b[^\n;&|]*\bapply\b", "Agents do not start a native Apply."),
]


def cmd_guard(_):
    data = hook_input()
    command = str((data.get("tool_input") or {}).get("command", ""))
    owner = os.environ.get("MARKITECT_OWNER_BRANCH")
    rules = list(GUARDS)
    if owner:
        rules.append((r"\b(checkout|switch|reset|rebase|merge|branch|update-ref)\b[^\n;&|]*(refs/heads/)?" + re.escape(owner) + r"(\s|$)",
                      f"The owner's branch {owner} is the bound input; work on your own branch."))
    for pattern, reason in rules:
        if re.search(pattern, command):
            print(f"Refused by the Markitect host: {reason}", file=sys.stderr)
            return 2
    return 0


def changed_protected(repo, base, candidate):
    paths = git(repo, "diff", "--name-only", base, candidate).splitlines()
    return [p for p in paths if any(p == x or (x.endswith("/") and p.startswith(x)) for x in PROTECTED)]


def project_checks(repo, candidate, checks, scratch):
    """Run every declared project check on a detached copy of the candidate."""
    tree = scratch / "check-tree"
    if tree.exists():
        git(repo, "worktree", "remove", "--force", str(tree), check=False)
        shutil.rmtree(tree, ignore_errors=True)
    git(repo, "worktree", "add", "--detach", str(tree), candidate)
    results = []
    try:
        for c in checks:
            cid = (c.get("ID") or {}).get("Value")
            directory = ((c.get("Directory") or {}).get("Value")) or "."
            args = (c.get("Args") or {}).get("Value") or []
            where = tree / (directory if isinstance(directory, str) else ".")
            timeout = int(os.environ.get("MARKITECT_CHECK_TIMEOUT", "120"))
            try:
                r = subprocess.run(["go", *args], cwd=where, capture_output=True, text=True, encoding="utf-8", timeout=timeout)
                results.append({"id": cid, "args": args, "exit": r.returncode, "output": (r.stdout + r.stderr)[-3000:]})
            except subprocess.TimeoutExpired:
                results.append({"id": cid, "args": args, "exit": None, "output": f"timed out after {timeout} s"})
    finally:
        git(repo, "worktree", "remove", "--force", str(tree), check=False)
    return results


def cmd_publish(a):
    """Publish only an approved candidate that every guard accepts (concept §12.2)."""
    repo = toplevel()
    state = state_dir(a.run_id)
    prepared = json.loads((state / "prepare.json").read_text(encoding="utf-8"))
    result = json.loads(pathlib.Path(a.result).read_text(encoding="utf-8"))
    record = {"runId": a.run_id, "outcome": result.get("outcome"), "at": result.get("at"), "published": False,
              "base": prepared["base"], "agents": result.get("agents")}
    write(state / "workflow-result.json", result)
    if result.get("outcome") != "approved":
        record["reason"] = "the workflow did not end with Verify approval"
        return finish(state, record, 1)
    candidate = git(repo, "rev-parse", "--verify", str(result.get("candidate", "")) + "^{commit}", check=False)
    record["candidate"] = candidate
    refs = [line.split(" ", 1) for line in git(repo, "for-each-ref", "--format=%(objectname) %(refname)", f"refs/heads/mk/{a.run_id}/").splitlines()]
    root_tips = [sha for sha, name in refs if name.endswith("/_root")]
    guards = {
        "candidateIsRootIntegration": bool(candidate) and candidate in root_tips,
        "ownerBranchUnchanged": git(repo, "rev-parse", "refs/heads/" + prepared["branch"]) == prepared["base"],
        "descendsFromBase": bool(candidate) and subprocess.run(
            ["git", "-C", str(repo), "merge-base", "--is-ancestor", prepared["base"], candidate]).returncode == 0,
    }
    guards["protectedInputsUnchanged"] = guards["descendsFromBase"] and not changed_protected(repo, prepared["base"], candidate)
    if all(guards.values()):
        code, report = markitect("check", repo, candidate)
        record["check"] = summarize(code, report)
        guards["structuralCheck"] = code == 0 and not stale_or_orphaned(report)
        record["projectChecks"] = project_checks(repo, candidate, prepared.get("checks") or [], state)
        guards["projectChecks"] = all(c["exit"] == 0 for c in record["projectChecks"])
    record["guards"] = guards
    if not all(guards.values()):
        record["reason"] = "a publication guard refused the candidate"
        return finish(state, record, 1)
    branch = f"markitect/apply/{a.run_id}"
    git(repo, "branch", "-f", branch, candidate)
    record.update({"published": True, "branch": branch, "candidateTree": git(repo, "rev-parse", candidate + "^{tree}"),
                   "noChange": git(repo, "rev-parse", candidate + "^{tree}") == prepared["baseTree"]})
    return finish(state, record, 0)


def write(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=1), encoding="utf-8")


def fail(value):
    print(json.dumps(value, indent=1))
    return 1


def finish(state, record, code):
    write(state / "publication.json", record)
    print(json.dumps(record, indent=1))
    return code


def main(argv=None):
    sys.stdout.reconfigure(encoding="utf-8")
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    sub = p.add_subparsers(dest="command", required=True)
    s = sub.add_parser("prepare")
    s.add_argument("--run-id", required=True)
    s = sub.add_parser("check")
    s.add_argument("--revision")
    s = sub.add_parser("context")
    s.add_argument("package")
    s.add_argument("--revision")
    sub.add_parser("note")
    sub.add_parser("guard")
    s = sub.add_parser("publish")
    s.add_argument("--run-id", required=True)
    s.add_argument("--result", required=True)
    a = p.parse_args(argv)
    return {"prepare": cmd_prepare, "check": cmd_check, "context": cmd_context, "note": cmd_note,
            "guard": cmd_guard, "publish": cmd_publish}[a.command](a)


if __name__ == "__main__":
    sys.exit(main())
