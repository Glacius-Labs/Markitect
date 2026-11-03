"""One fresh checker over a submission that is already made: the planted-bug test of the arms' checker.

  python3 -I -B review_arm.py --markitect BIN --answers new|old --submission FILE [--claude BIN] [--credentials FILE] [--effort LEVEL] TASK_DIR OUT_DIR

Run it through the suite's sandbox, as one_agent_arm.py. The task's start is the submission, and FILE holds its checklist
and the maker's verdicts. No maker starts and nothing lands: the Result delivers the start unchanged.

The host runs the gate's checks first (`markitect check` and the required checks). A submission that does not pass them
starts no agent, since the test measures the checker and not the gate. Then one checker session reviews the submission
as the arm's checker does, with the answers as launched (`new`) or as they were before that change (`old`). The Result
holds every answer with its evidence, under `reviews`.
"""
import argparse
import json
import os
import pathlib
import shutil
import sys
import tempfile
import time

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import claude_code as cc  # noqa: E402
import markitect_arm as arm  # noqa: E402
import one_agent_arm as one  # noqa: E402

OLD_ANSWERS = ("confirmed", "disputed")
REFUSING = ("wrong", "disputed")


def old_prompt(host, items, verdicts, checks, base, commit, file):
    """The checker's prompt as the pilot's first set launched it, at ce16a754f."""
    diff = host.git("diff", base, commit)
    if len(diff) <= one.MAX_INLINE_DIFF:
        marks = one.fence(diff)
        shown = f"{marks}diff\n{diff}\n{marks}\n"
    else:
        names = "\n".join(f"- {p}" for p in host.git("diff", "--name-only", base, commit).splitlines())
        shown = f"The diff is larger than {one.MAX_INLINE_DIFF >> 10} KiB. `git diff {base} {commit}` shows it. Changed files:\n\n{names}\n"
    rows = "".join(f"| `{i['item']}` | {verdicts[i['item']]['verdict']} | {str(verdicts[i['item']]['evidence']).replace('|', ' ')} |\n" for i in items)
    ran = "".join(f"- {c['name']}: {'passed' if c['passed'] else 'failed'}\n" for c in checks) or "- none\n"
    return f"""# Task

Re-check every verdict of the submission `{commit}` against the model and the repository, which holds that commit. Change no file. Dispute a `conforms` or `changed` verdict whose evidence does not show how each example in the item's Definition is met, and one whose realization matches part of the Definition but not its example. Where the Definition and what it links to contradict each other, nothing can conform: the maker's only right verdict is `model-problem`, so dispute a verdict that claims the item conforms. Confirm a `model-problem` verdict only if the model really contradicts itself or leaves open behavior at the project boundary for this item; otherwise dispute it.

{one.table(items)}
## Submitted verdicts

| Item | Verdict | Evidence |
|---|---|---|
{rows}
## Required checks on the whole repository

{ran}
## Diff from `{base}` to `{commit}`

{shown}
# Model

{one.model_text(host.repo, [i['item'] for i in items])}
# Answer

Write your review to `{file}`, which the host reads and never commits, as JSON:

{{"reviews": [{{"item": "ITEM", "answer": "confirmed", "evidence": "..."}}]}}

One entry per verdict, `confirmed` or `disputed`, each with evidence: what you read or ran, and for a dispute what is wrong and where.
"""


def read_answers(path, known, answers):
    """A checker's answers to items of the checklist, with the answer set it was given."""
    try:
        entries = json.loads(pathlib.Path(path).read_text(encoding="utf-8"))["reviews"]
        return {e["item"]: {"answer": e["answer"], "evidence": e.get("evidence") or ""} for e in entries
                if isinstance(e, dict) and e.get("item") in known and e.get("answer") in answers}
    except (OSError, ValueError, KeyError, TypeError):
        return {}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--markitect", required=True, help="the markitect program for this platform, for check only")
    p.add_argument("--answers", choices=("new", "old"), required=True)
    p.add_argument("--submission", required=True, help="a JSON file with the submission's owner commit, checklist and verdicts")
    p.add_argument("--claude", default=shutil.which("claude") or "claude")
    p.add_argument("--credentials", default=os.environ.get("EVALUATION_LOGIN") or str(pathlib.Path.home() / ".claude" / ".credentials.json"))
    p.add_argument("--effort", default="high")
    p.add_argument("task")
    p.add_argument("out")
    a = p.parse_args(argv)
    began = time.monotonic()
    task, out = pathlib.Path(a.task), pathlib.Path(a.out)
    out.mkdir(parents=True, exist_ok=True)
    report = {"arm": "review", "answers": a.answers, "effort": a.effort, "command": "review"}

    def finish(code, **facts):
        report.update(facts)
        (out / "run.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        return code

    try:
        provider, model = arm.provider_model()
        if provider != "claude":
            raise one.ArmError("this test runs on Claude only")
        report.update(provider=provider, model=model)
        spec = json.loads((task / "task.json").read_text(encoding="utf-8"))
        commit, deadline = spec["start"]["commit"], began + spec["deadlineSeconds"]
        name = arm.run_name()
        report["run"] = name
        submission = json.loads(pathlib.Path(a.submission).read_text(encoding="utf-8"))
        items, verdicts, owner = submission["items"], submission["verdicts"], submission["owner"]
        known = {i["item"] for i in items}
        login, secrets = cc.session_login(pathlib.Path(a.credentials).read_bytes(), spec["deadlineSeconds"])
        with tempfile.TemporaryDirectory(prefix="review-arm-", ignore_cleanup_errors=True) as tmp:
            tmp = pathlib.Path(tmp).resolve()
            for folder in ("hooks", "home", "bin"):
                (tmp / folder).mkdir()
            env = arm.git_env(tmp / "hooks")
            program = pathlib.Path(a.markitect)
            local = tmp / "bin" / program.name
            shutil.copyfile(program, local)
            local.chmod(0o755)
            a.markitect = str(local)
            session_env = cc.base_env(tmp / "home", tmp / "hooks")
            session_env["PATH"] = str(tmp / "bin") + os.pathsep + session_env.get("PATH", "")
            repo = tmp / "work"
            arm.git(tmp, "clone", "-q", "--no-checkout", str((task / "start.bundle").resolve()), str(repo), env=env)
            arm.git(repo, "config", "core.autocrlf", "false", env=env)
            arm.git(repo, "checkout", "-q", "-B", "main", commit, env=env)
            host = one.Host(a, tmp, repo, out, env, session_env, login, model, deadline, secrets)
            compiled, code = arm.check(a, repo, env)
            checks = host.run_checks(commit)
            if code != 0 or not all(c["passed"] for c in checks):
                raise one.ArmError(f"the submission does not pass the gate: check exit {code}, checks {[(c['name'], c['passed']) for c in checks]}")
            review, missing, tries, status, why = {}, items, 0, "reviewed", "the checker answered every verdict"
            while missing and tries < 2:
                tries += 1
                path = host.answers / f"checker-1-{tries}.json"
                if a.answers == "new":
                    prompt = one.checker_prompt(host, missing, verdicts, checks, owner, commit, path)
                else:
                    prompt = old_prompt(host, missing, verdicts, checks, owner, commit, path)
                if tries > 1:
                    prompt += "\nThe last review left out these items, and an entry for each is needed: " + ", ".join(f"`{m['item']}`" for m in missing) + ".\n"
                reason = host.session(f"checker-1-{tries}", one.ROOT, "checker", 1, prompt)
                host.tidy()
                if reason == "deadline":
                    status, why = "incomplete", "the deadline passed during the check"
                    break
                if reason == "failed" and host.starts[-1].get("class"):
                    status, why = "incomplete", "a provider failure ended the checker: " + host.starts[-1].get("detail", "")
                    break
                review = {**read_answers(path, known, one.ANSWERS if a.answers == "new" else OLD_ANSWERS), **review}
                missing = [i for i in items if i["item"] not in review]
            if status == "reviewed" and missing:
                status, why = "incomplete", "the checker left no complete review twice in a row"
            result = {"format": 1, "run": name, "command": "review", "outcome": "not-landed", "state": status, "why": why, "answers": a.answers, "submission": commit,
                      "base": owner, "start": commit, "landed": None, "requiredChecks": {"ran": True, "wholeRepository": True, "checks": checks}, "items": [],
                      "reviews": review, "refused": sorted(item for item, r in review.items() if r["answer"] in REFUSING), "notParticipating": [], "findings": [], "starts": host.starts}
            (out / "results").mkdir(exist_ok=True)
            (out / "results" / f"{name}.json").write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
            report["provider"], report["model"] = arm.reported_model(result, provider, model)
            failed = [{k: s.get(k) for k in ("manager", "role", "round", "class", "detail")} for s in host.starts if s.get("reason") == "failed"]
            costs = [(s.get("usage") or {}).get("costUsd") for s in host.starts]
            report["usage"] = {"costUsd": round(sum(c for c in costs if c), 6) if any(costs) else None, "sessions": len(host.starts)}
            report.update(status=status, cause=why, items=len(items), starts=len(host.starts), failedStarts=failed, refused=len(result["refused"]), landed=None,
                          leftovers=[], afterLanding=[], afterLandingReported=True, cleanup=[])
            if not any(s.get("reason") in ("finished", "deadline") for s in host.starts):
                return finish(2, noAgentTurn=True)
            if cc.contains_secret(out, secrets):
                for path in (out / "transcripts", out / "sessions", out / "results"):
                    if path.is_dir():
                        shutil.rmtree(path)
                return finish(1, status="quarantined", quarantine=True)
            arm.bundle(repo, commit, out / "result.bundle", env)
            code = 0 if status == "reviewed" else 1
            report["exit"] = code
            return finish(code)
    except (one.ArmError, arm.ArmError, cc.VariantError, OSError, ValueError, KeyError, TypeError, one.subprocess.SubprocessError) as e:
        if (out / "result.bundle").exists():
            report["recordError"] = f"{type(e).__name__}: {e}"
            return finish(report.get("exit", 1))
        return finish(2, status="could-not-run", cause=f"{type(e).__name__}: {e}")


if __name__ == "__main__":
    sys.exit(main())
