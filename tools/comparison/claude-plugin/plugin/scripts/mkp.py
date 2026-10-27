"""Deterministic steps of the Markitect Claude plugin (comparison arm P).

The plugin's agents judge and realize; this script does everything that has
one right answer, the way the native host does: the structural check before
any agent starts, the Manager tree, compact check results for agents, the
check note at every Manager handoff, the host's own record of every agent's
answer, the Bash guard and the publication guard. Publication rests on those
records, not on what the session reports.

  python mkp.py prepare --run-id ID
  python mkp.py check [--revision REV]
  python mkp.py context PACKAGE [--revision REV]
  python mkp.py record          (SubagentStop hook, JSON on stdin)
  python mkp.py guard           (PreToolUse Bash hook, JSON on stdin)
  python mkp.py publish --run-id ID [--result WORKFLOW_RESULT.json]

Environment: MARKITECT_BIN (the markitect binary), MARKITECT_STATE_DIR (run
records, outside the repository), MARKITECT_OWNER_BRANCH (the branch Apply
must not write), MARKITECT_RUN_ID, MARKITECT_CHECK_TIMEOUT (project check
seconds, default 600) and the workflow limits in LIMITS.
"""
import argparse
import json
import os
import pathlib
import re
import shlex
import shutil
import subprocess
import sys
import time

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


HEADER = re.compile(r'Markitect (Assessment|Implement|Verify) round (\d+), run (\S+)\. You are the Manager of .*? \(Package "([^"]*)"\)')


def agent_answer(transcript):
    """The agent's task header and its last structured answer, from its own transcript."""
    header, answer = None, None
    for line in pathlib.Path(transcript).read_text(encoding="utf-8", errors="replace").splitlines():
        try:
            entry = json.loads(line)
        except ValueError:
            continue
        message = entry.get("message") if isinstance(entry, dict) else None
        if not isinstance(message, dict):
            continue
        content = message.get("content")
        items = [{"type": "text", "text": content}] if isinstance(content, str) else content if isinstance(content, list) else []
        for item in items:
            if not isinstance(item, dict):
                continue
            if header is None and entry.get("type") == "user" and item.get("type") == "text":
                match = HEADER.search(item.get("text", ""))
                if match:
                    header = {"mandate": match[1], "round": int(match[2]), "run": match[3], "package": match[4]}
            if entry.get("type") == "assistant" and item.get("type") == "tool_use" and item.get("name") == "StructuredOutput":
                answer = item.get("input")
    return header, answer


def add_note(repo, commit, text, attempts=5):
    """Siblings finishing together race on the notes ref; retry a few times."""
    for attempt in range(attempts):
        r = subprocess.run(["git", "-C", str(repo), "notes", f"--ref={NOTES_REF}", "add", "-f", "-m", text, commit],
                           capture_output=True, text=True, encoding="utf-8")
        if r.returncode == 0:
            return
        time.sleep(0.5 * (attempt + 1))
    raise RuntimeError(f"git notes: {r.stderr.strip()}")


def cmd_record(_):
    """SubagentStop: the host's own record of every Markitect agent's result.

    For each agent it records the mandate, round and Package from the task
    header, the commit its worktree has checked out and its structured answer,
    all read from the agent's own transcript, never from the session's
    summary. A Manager's handoff commit also gets its deterministic check as a
    note that travels to the parent (D11). The hook never sends an agent back
    and never fails.
    """
    data = hook_input()
    try:
        kind = str(data.get("agent_type", ""))
        if not kind.startswith("markitect:"):
            return 0
        repo = toplevel(data.get("cwd") or ".")
        header, answer = agent_answer(data["agent_transcript_path"]) if data.get("agent_transcript_path") else (None, None)
        record = {"agent": data.get("agent_id"), "agentType": kind, "head": git(repo, "rev-parse", "HEAD"),
                  "header": header, "answer": answer}
        if kind == "markitect:manager":
            reported = (answer or {}).get("commit")
            target = reported if reported and git(repo, "cat-file", "-t", reported, check=False) == "commit" else record["head"]
            code, report = markitect("check", repo, target)
            summary = summarize(code, report)
            summary.update({"commit": target, "agent": data.get("agent_id")})
            add_note(repo, target, json.dumps(summary))
            record["check"] = summary
        run_id = os.environ.get("MARKITECT_RUN_ID", "adhoc")
        write(state_dir(run_id) / "agents" / f"{data.get('agent_id', 'unknown')}.json", record)
    except Exception as e:  # a hook must not stop the agent organization
        print(f"markitect record incomplete: {e}", file=sys.stderr)
    return 0


GIT_VALUE_OPTIONS = {"-C", "-c", "--git-dir", "--work-tree", "--namespace", "--exec-path"}
NOTE_WRITES = {"add", "append", "copy", "edit", "merge", "prune", "remove"}
OWNER_VERBS = {"checkout", "switch", "reset", "rebase", "merge", "branch", "update-ref", "worktree"}


def segments(command):
    """Each simple command of a shell line as tokens, quotes respected."""
    for part in re.split(r"\|\||&&|[;|\n]", command):
        try:
            tokens = shlex.split(part, posix=True)
        except ValueError:
            tokens = part.split()
        while tokens and re.fullmatch(r"[A-Za-z_][A-Za-z0-9_]*=.*", tokens[0]):
            tokens = tokens[1:]
        if tokens:
            yield tokens


def refusal(tokens, owner):
    """Why the host refuses this simple command, or None. Only the command
    word and git's own subcommand count, never message or path text."""
    word = pathlib.PurePath(tokens[0].replace("\\", "/")).name.lower()
    if word in ("markitect", "markitect.exe") and tokens[1:2] == ["apply"]:
        return "Agents do not start a native Apply."
    if word not in ("git", "git.exe"):
        return None
    rest = tokens[1:]
    while rest and rest[0].startswith("-"):
        option = rest.pop(0)
        if option in GIT_VALUE_OPTIONS and rest:
            rest.pop(0)
    if not rest:
        return None
    verb, args = rest[0], rest[1:]
    refs = [a.removeprefix("refs/heads/") for a in args]
    if verb == "push":
        return "Apply never pushes; hand off your commit instead."
    if verb == "notes" and NOTE_WRITES & set(args):
        return "Check notes are written by the host only; read them with git notes --ref=markitect-check show COMMIT."
    if verb == "update-ref" or (verb == "worktree" and {"remove", "prune", "move"} & set(args)):
        return "Refs and worktrees are managed by the host."
    if verb in OWNER_VERBS and any(r.startswith("markitect/apply/") for r in refs):
        return "Publication branches are written by the publication guard only."
    if verb == "branch" and {"-D", "-d", "--delete", "-M", "-m", "--move"} & set(args):
        return "Do not delete or rename branches; commit on your own branch."
    if owner and verb in OWNER_VERBS and owner in refs:
        return f"The owner's branch {owner} is the bound input; work on your own branch."
    return None


def cmd_guard(_):
    data = hook_input()
    command = str((data.get("tool_input") or {}).get("command", ""))
    owner = os.environ.get("MARKITECT_OWNER_BRANCH")
    for tokens in segments(command):
        reason = refusal(tokens, owner)
        if reason:
            print(f"Refused by the Markitect host: {reason}", file=sys.stderr)
            return 2
    return 0


def changed_protected(repo, base, candidate):
    # --no-renames: a moved model file must show its old path too.
    paths = git(repo, "diff", "--no-renames", "--name-only", base, candidate).splitlines()
    return [p for p in paths if any(p == x or (x.endswith("/") and p.startswith(x)) for x in PROTECTED)]


def project_checks(repo, candidate, checks, scratch):
    """Run every declared project check on a detached copy of the candidate."""
    tree = scratch / "check-tree"
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
            timeout = int(os.environ.get("MARKITECT_CHECK_TIMEOUT", "600"))
            try:
                r = subprocess.run(["go", *args], cwd=where, capture_output=True, text=True, encoding="utf-8", timeout=timeout)
                results.append({"id": cid, "args": args, "exit": r.returncode, "output": (r.stdout + r.stderr)[-3000:]})
            except subprocess.TimeoutExpired:
                results.append({"id": cid, "args": args, "exit": None, "output": f"timed out after {timeout} s"})
    finally:
        git(repo, "worktree", "remove", "--force", str(tree), check=False)
    return results


def packages(tree):
    return [tree["pkg"]] + [p for c in tree["children"] for p in packages(c)]


def verified_candidate(records, all_packages):
    """The candidate of the latest Verify round in which every Package approved it.

    Each approval counts only for the commit the verifier's worktree actually
    had checked out; a reject or model_blocker in that round on that commit
    stays in force.
    """
    rounds = {}
    for r in records:
        h, answer = r.get("header") or {}, r.get("answer") or {}
        if r.get("agentType") != "markitect:verifier" or h.get("mandate") != "Verify":
            continue
        rounds.setdefault((h["round"], r["head"]), {}).setdefault(h["package"], []).append(answer.get("decision"))
    for (round_, head), by_package in sorted(rounds.items(), reverse=True):
        if all(by_package.get(p) and set(by_package[p]) == {"approve"} for p in all_packages):
            return round_, head
    return None, None


def cmd_publish(a):
    """Publish only a candidate that every Package's Verify approved and every guard accepts (concept §12.2)."""
    repo = toplevel()
    state = state_dir(a.run_id)
    prepared = json.loads((state / "prepare.json").read_text(encoding="utf-8"))
    result = json.loads(pathlib.Path(a.result).read_text(encoding="utf-8")) if a.result and pathlib.Path(a.result).exists() else {}
    agents_dir = state / "agents"
    records = [json.loads(p.read_text(encoding="utf-8")) for p in sorted(agents_dir.glob("*.json"))] if agents_dir.exists() else []
    record = {"runId": a.run_id, "outcome": result.get("outcome"), "at": result.get("at"), "agents": result.get("agents"),
              "recordedAgents": len(records), "published": False, "base": prepared["base"]}
    round_, candidate = verified_candidate(records, packages(prepared["tree"]))
    record.update(round=round_, candidate=candidate)
    if candidate is None:
        record["reason"] = "no Verify round in which every Package approved one candidate"
        return finish(state, record, 1)
    if result.get("candidate") and result["candidate"] != candidate:
        record["reason"] = f"the workflow names {result['candidate']}, the host records approve {candidate}"
        return finish(state, record, 1)
    refs = git(repo, "for-each-ref", "--format=%(objectname)", f"refs/heads/mk/{a.run_id}/i{round_}/_root/").splitlines()
    guards = {
        "candidateIsRootResult": candidate in refs,
        "ownerBranchUnchanged": git(repo, "rev-parse", "refs/heads/" + prepared["branch"]) == prepared["base"],
        "descendsFromBase": subprocess.run(["git", "-C", str(repo), "merge-base", "--is-ancestor", prepared["base"], candidate]).returncode == 0,
    }
    guards["protectedInputsUnchanged"] = guards["descendsFromBase"] and not changed_protected(repo, prepared["base"], candidate)
    if all(guards.values()):
        code, report = markitect("check", repo, candidate)
        record["check"] = summarize(code, report)
        guards["structuralCheck"] = code == 0 and not stale_or_orphaned(report)
        try:
            record["projectChecks"] = project_checks(repo, candidate, prepared.get("checks") or [], state)
            guards["projectChecks"] = all(c["exit"] == 0 for c in record["projectChecks"])
        except Exception as e:
            record["projectChecks"] = {"error": str(e)}
            guards["projectChecks"] = False
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
    sub.add_parser("record")
    sub.add_parser("guard")
    s = sub.add_parser("publish")
    s.add_argument("--run-id", required=True)
    s.add_argument("--result", help="the workflow's own result, kept as a record and cross-checked")
    a = p.parse_args(argv)
    return {"prepare": cmd_prepare, "check": cmd_check, "context": cmd_context, "record": cmd_record,
            "guard": cmd_guard, "publish": cmd_publish}[a.command](a)


if __name__ == "__main__":
    sys.exit(main())
