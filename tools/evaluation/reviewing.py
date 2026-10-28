"""The blinded code review of one result, the secondary Code quality area.

`suite.py review` makes a blinded copy of the result: the result tree, the
start tree and the diff between them, with `.markitect/` (and the registry in
it) and compiled executables removed; the code itself stays unchanged. The copy
sits in a directory named by a random token. A fresh reviewer session
(reviewer.py, in the evaluation sandbox) judges it against the task's rubric
(the judge folder's rubric.md) and requirements (task.md), with the Phase A-C
review prompt (tools/comparison/review-prompt.md). Every critical finding is
checked mechanically (tools/comparison/rubric_check.py); each valid one goes to
a second, fresh reviewer session with the Phase A-C confirmation prompt and
counts only when that reviewer confirms it. Each reviewer also guesses which of
the four variants produced the result; the report gives how often the guess
was right, which measures what the blinding leaves.
"""
import hashlib
import importlib.util
import json
import os
import pathlib
import re
import shutil
import subprocess
import tempfile
import uuid

import judging
import suite

HERE = pathlib.Path(__file__).resolve().parent
COMPARISON = HERE.parent / "comparison"
PROMPTS = COMPARISON / "review-prompt.md"
REVIEWER = ["python3", "-I", "-B", "/variant/reviewer.py"]
DEADLINE = 1200
VARIANTS = ("K", "M", "P", "G")
GUESS = ('Also add the field "variantGuess" to that JSON object: which of four ways of building software most likely produced '
         'this program, as {"variant": "K", "M", "P" or "G", "why": "one sentence"}. K: one plain coding-agent session working '
         'from the requirements. M: Markitect, a hierarchy of manager agents working from a canonical model. P: a coding-agent '
         'plugin working from a canonical model. G: a Git-based multi-agent prototype working from a canonical model.')


def _load(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


assess_candidate = _load("assess_candidate", COMPARISON / "assess_candidate.py")
rubric_check = _load("rubric_check", COMPARISON / "rubric_check.py")


def prompts():
    """The frozen Phase A-C prompts for the review and the confirmation, each with the variant guess added."""
    text = PROMPTS.read_text(encoding="utf-8")
    blocks = re.findall(r"^## (Review|Confirmation[^\n]*)\n+```text\n(.*?)\n```", text, re.M | re.S)
    found = {title.split()[0]: body for title, body in blocks}
    if set(found) != {"Review", "Confirmation"} or not found["Confirmation"].rstrip().endswith("FINDING"):
        raise judging.JudgeError(f"{PROMPTS}: the review and confirmation prompts are not where they were")
    review = found["Review"].rstrip() + " " + GUESS
    head = found["Confirmation"].rstrip()[:-len("FINDING")].rstrip()
    return review, head + " " + GUESS + "\n\nFINDING"


def blind(task_dir, start, bundle, commit, dest):
    """The blinded copy: candidate/, start/ and diff.patch without .markitect/ or compiled executables."""
    with tempfile.TemporaryDirectory() as tmp:
        repo = pathlib.Path(tmp) / "repo"
        suite.git(tmp, "init", "-q", "--bare", str(repo))
        suite.git(repo, "fetch", "-q", str((pathlib.Path(task_dir) / "start.bundle").resolve()), f"+{start}:refs/review/start")
        suite.git(repo, "fetch", "-q", str(pathlib.Path(bundle).resolve()), f"+{commit}:refs/review/result")
        for rev, name in ((commit, "candidate"), (start, "start")):
            if suite.git(repo, "ls-tree", rev):
                assess_candidate.export(repo, rev, dest / name, (".markitect",), strip_binaries=True)
            else:
                # A new project starts from an empty commit, whose archive tarfile cannot open.
                (dest / name).mkdir()
        hidden = sorted(set(assess_candidate.binaries(repo, commit)) | set(assess_candidate.binaries(repo, start)))
        (dest / "diff.patch").write_bytes(assess_candidate.git(repo, "diff", "--no-renames", start, commit, "--", ".", ":!.markitect",
                                                               *[":!" + p for p in hidden], binary=True))


def session(name, prompt, token, material, out, model, box, reviewer, deadline):
    """One fresh reviewer session on the material; its record, with the answer when it gave one."""
    sdir = out / "sessions" / name
    (sdir / "variant").mkdir(parents=True)
    with tempfile.TemporaryDirectory() as tmp:
        copy = pathlib.Path(tmp) / "material"
        shutil.copytree(material, copy)
        (copy / "review.json").write_text(json.dumps({"prompt": prompt, "token": token, "deadlineSeconds": deadline}), encoding="utf-8")
        stdout, stderr = sdir / "reviewer.stdout.log", sdir / "reviewer.stderr.log"
        record = {"name": name, "conditions": "sandbox" if box else "host", "startedUtc": suite.now()}
        if box:
            timing, record["sandbox"] = box.run(copy, sdir / "variant", None, reviewer, model, deadline, sdir,
                                                lambda argv, stop: suite.start_variant(argv, dict(os.environ), stdout, stderr, deadline, stop))
            record.update(timing)
        else:
            argv = [*reviewer, str(copy), str((sdir / "variant").resolve())]
            record.update(suite.start_variant(argv, dict(os.environ, EVALUATION_MODEL=model), stdout, stderr, deadline))
    record["endedUtc"] = suite.now()
    record["reported"] = suite.reported(sdir / "variant" / "run.json")
    record["modelMatches"] = suite.model_check(model, record["reported"])
    answer = suite.reported(sdir / "variant" / "answer.json")
    record["answer"] = None if answer is None or answer.get("unreadable") else answer
    box_ok = not box or ("error" not in record["sandbox"] and "container" in record["sandbox"] and not record["sandbox"]["residue"]
                         and not record["sandbox"].get("quarantine"))
    record["valid"] = bool(box_ok and record["modelMatches"] and record["answer"] is not None)
    (sdir / "session.json").write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
    return record


def guess(answer):
    g = (answer or {}).get("variantGuess")
    variant = g.get("variant") if isinstance(g, dict) else None
    return {"variant": variant if variant in VARIANTS else None, "why": g.get("why") if isinstance(g, dict) else None}


def review(run_dir, judge_dir, out, model, box=None, reviewer=REVIEWER, deadline=DEADLINE):
    """Review one run's result; writes OUT/review.json and returns it."""
    run_dir, judge_dir, out = pathlib.Path(run_dir), pathlib.Path(judge_dir), pathlib.Path(out)
    record = json.loads((run_dir / "suite.json").read_text(encoding="utf-8"))
    if not record.get("delivered"):
        raise judging.JudgeError(f"{run_dir} delivered no result: there is nothing to review")
    task_dir = pathlib.Path(record["task"]["path"])
    if suite.read_task(task_dir)["intentSha256"] != record["task"]["intentSha256"]:
        raise judging.JudgeError(f"{task_dir} is not the task the run had")
    config = judging.load(judge_dir)
    rubric = judge_dir / "rubric.md"
    if not rubric.is_file():
        raise judging.JudgeError(f"{judge_dir} has no rubric.md")
    if out.exists():
        raise judging.JudgeError(f"{out} exists")
    review_prompt, confirm_prompt = prompts()
    token = uuid.uuid4().hex[:12]
    material = out / "material"
    (material / token).mkdir(parents=True)
    commit = record["result"]["commit"]
    blind(task_dir, record["task"]["start"], run_dir / "variant" / "result.bundle", commit, material / token)
    shutil.copyfile(task_dir / "task.md", material / "requirements.md")
    shutil.copyfile(rubric, material / "rubric.md")
    result = {"format": judging.FORMAT, "run": str(run_dir.resolve()), "commit": commit, "token": token, "model": model,
              "judge": {"name": config["name"], "folderSha256": judging.folder_digest(judge_dir)},
              "rubricSha256": hashlib.sha256(rubric.read_bytes()).hexdigest(),
              "promptsSha256": hashlib.sha256(PROMPTS.read_bytes()).hexdigest(), "problems": []}
    first = session("review", review_prompt, token, material, out, model, box, reviewer, deadline)
    result["review"] = {k: first[k] for k in ("valid", "answer", "reported")}
    sessions, confirmations = [first], []
    if not first["valid"]:
        result["problems"].append("the review session gave no valid answer")
    elif first["answer"].get("token") != token:
        result["problems"].append("the review answers for another token")
    else:
        entry = config.get("reviewEntry") or rubric_check.LIBRARY_ENTRY
        result["check"] = rubric_check.check(material / token, material / "requirements.md", first["answer"], entry=entry)
        for i, critical in enumerate(c for c in result["check"]["criticals"] if c["valid"]):
            finding = {"item": critical["item"], **critical["finding"]}
            s = session(f"confirm-{i + 1}", confirm_prompt.replace("FINDING", json.dumps(finding, ensure_ascii=False)), token,
                        material, out, model, box, reviewer, deadline)
            sessions.append(s)
            confirmed = s["valid"] and (s["answer"] or {}).get("confirmed") is True
            confirmations.append({"finding": finding, "valid": s["valid"], "confirmed": confirmed, "reason": (s["answer"] or {}).get("reason")})
            if not s["valid"]:
                result["problems"].append(f"confirmation {i + 1} gave no valid answer")
    result["confirmations"] = confirmations
    check = result.get("check") or {"criticals": [], "valid_criticals": 0}
    result["criticals"] = {"reported": len(check["criticals"]), "valid": check["valid_criticals"],
                           "confirmed": sum(c["confirmed"] for c in confirmations)}
    result["guesses"] = [{"session": s["name"], **guess(s["answer"])} for s in sessions]
    result["sessions"] = [{k: s.get(k) for k in ("name", "valid", "conditions", "wallSeconds", "timedOut", "exitCode")} for s in sessions]
    result["valid"] = not result["problems"]
    (out / "review.json").write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    return result
