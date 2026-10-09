"""Small local CLI wrapper: seed, fixed start/resume, freeze, assessment. No agent SDK."""
from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import time
from datetime import datetime, timezone
from contextlib import contextmanager

HERE = Path(__file__).resolve().parent
PROMPT = "Hier liegt das Backlog. Implementiere die Arbeit, halte die Projektregeln ein, prüfe die Ergebnisse und merge die fertigen Änderungen nach main."
CONTINUE = "Setze denselben Backlog-Auftrag aus dem vorhandenen Projektstand fort. Halte die Projektregeln ein, prüfe die Ergebnisse und merge fertige Änderungen nach main. Dokumentiere einen echten Blocker, falls du nicht fortsetzen kannst."
MODEL = "gpt-6-luna"
EFFORT = "high"


def utc():
    return datetime.now(timezone.utc).isoformat()


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def read(path):
    return json.loads(Path(path).read_text(encoding="utf-8-sig"))


def write(path, value):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_suffix(path.suffix + ".tmp")
    temporary.write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n", encoding="utf-8", newline="\n")
    os.replace(temporary, path)


def git(repo, *args):
    return subprocess.check_output(["git", "-C", str(repo), *args], text=True, encoding="utf-8", timeout=45).strip()


@contextmanager
def locked(state_dir):
    path = Path(state_dir) / "dispatch.lock"
    fd = os.open(path, os.O_CREAT | os.O_EXCL | os.O_WRONLY)
    try:
        os.write(fd, str(os.getpid()).encode("ascii"))
        yield
    finally:
        os.close(fd)
        path.unlink()


def commit(repo, message):
    git(repo, "add", ".")
    git(repo, "diff", "--cached", "--check")
    git(repo, "-c", "user.name=Study Seed", "-c", "user.email=study@example.invalid", "commit", "-m", message)


def inventory(root):
    return {p.relative_to(root).as_posix(): {"sha256": sha(p), "bytes": p.stat().st_size}
            for p in sorted(root.rglob("*")) if p.is_file() and ".git" not in p.relative_to(root).parts}


def seed(case, target):
    """Ordinary files/Git only. Never launches Codex or Markitect."""
    target = Path(target).resolve()
    if target.exists():
        raise ValueError("Seed destination must not exist")
    source = HERE / "cases" / case
    if not source.is_dir():
        raise ValueError("Unknown case")
    target.mkdir(parents=True)
    shutil.copytree(HERE / "common", target, dirs_exist_ok=True)
    shutil.copytree(source, target, dirs_exist_ok=True)
    subprocess.run(["git", "init", "-b", "main", str(target)], check=True, capture_output=True)
    git(target, "config", "core.autocrlf", "false")
    commit(target, f"Original shared {case} seed and public backlog")
    return {"case": case, "seedCommit": git(target, "rev-parse", "HEAD"),
            "seedTree": git(target, "rev-parse", "HEAD^{tree}"), "files": inventory(target)}


def cell(case, arm, seed_repo, target, state_dir):
    target, state_dir = Path(target).resolve(), Path(state_dir).resolve()
    if arm not in ("conventional", "markitect") or target.exists() or state_dir.exists():
        raise ValueError("Fresh cell and external state directories required")
    if state_dir == target or target in state_dir.parents or state_dir in target.parents:
        raise ValueError("State and project must be separate sibling trees")
    if read(Path(seed_repo) / "STATIONS.json")["case"] != case:
        raise ValueError("Seed case does not match")
    subprocess.run(["git", "-c", "core.autocrlf=false", "clone", "--no-hardlinks", str(Path(seed_repo).resolve()), str(target)],
                   check=True, capture_output=True)
    git(target, "remote", "remove", "origin")
    git(target, "config", "core.autocrlf", "false")
    git(target, "checkout", "-b", "work/backlog")
    if arm == "conventional":
        with (target / "AGENTS.md").open("a", encoding="utf-8", newline="\n") as handle:
            handle.write("\n" + (HERE / "conventional" / "AGENTS.fragment.md").read_text(encoding="utf-8"))
    else:
        # These are authored initial model inputs, not simulated installation/tool output.
        shutil.copytree(HERE / "models" / case, target, dirs_exist_ok=True)
    shutil.copytree(HERE / "native", target / ".codex", dirs_exist_ok=True)
    stations = read(target / "STATIONS.json")["stations"]
    write(target / ".study" / "station.json", stations[0])
    commit(target, f"Prepare {arm} method inputs; no application work")
    state = {"case": case, "arm": arm, "repo": str(target), "stateDir": str(state_dir),
             "seedCommit": git(seed_repo, "rev-parse", "HEAD"),
             "preparedCommit": git(target, "rev-parse", "HEAD"), "preparedTree": git(target, "rev-parse", "HEAD^{tree}"),
             "createdUtc": utc(), "status": "prepared", "sessions": [], "interventions": [],
             "productReadiness": None, "initialSetup": [], "trialGrant": None, "sessionId": None,
             "deadlineUtc": None, "implementationDeadlineUtc": None, "evaluationDeadlineUtc": None,
             "tokens": None, "cost": None, "servingModel": None,
             "stationIndex": 0, "stations": stations, "snapshots": []}
    write(state_dir / "state.json", state)
    return state


def command(executable, repo, last_message, session_id=None, assessment=False):
    # Normal project/user configuration, rules and tools remain active. No output DTO.
    cmd = [str(executable), "exec", "--sandbox", "workspace-write"]
    if session_id:
        cmd += ["resume", session_id]
    else:
        cmd += ["--cd", str(repo)]
    cmd += ["--model", MODEL, "--config", 'model_reasoning_effort="high"',
            "--json", "--output-last-message", str(last_message), "-"]
    return cmd


def parse_events(path):
    result = {"sessionId": None, "completedTurns": [], "errors": [], "invalidLines": 0}
    for raw in Path(path).read_text(encoding="utf-8", errors="replace").splitlines():
        try:
            event = json.loads(raw)
        except ValueError:
            result["invalidLines"] += 1
            continue
        if event.get("type") == "thread.started":
            result["sessionId"] = event.get("thread_id")
        if event.get("type") == "turn.completed":
            # Preserve real collector values, including missing/partial data. Never infer zero.
            result["completedTurns"].append({"usage": event.get("usage"), "event": event})
        if event.get("type") in ("turn.failed", "error"):
            result["errors"].append(event)
    return result


def admission(state, grant, executable, phase):
    repo = Path(state["repo"])
    if not grant.get("executionAuthorized") or grant.get("model") != MODEL or grant.get("reasoning") != EFFORT:
        raise ValueError("A separate exact Luna High actual grant is required")
    if grant.get("cell") != f'{state["case"]}/{state["arm"]}':
        raise ValueError("Grant belongs to another cell")
    if grant.get("preparedCommit") != state["preparedCommit"] or grant.get("executableSha256") != sha(executable):
        raise ValueError("Frozen project/executable binding differs")
    if not state["sessions"] and (git(repo, "rev-parse", "HEAD") != state["preparedCommit"] or git(repo, "status", "--porcelain")):
        raise ValueError("Prepared project drift before first admission")
    if grant.get("maxCellWallSeconds") != 7200 or grant.get("maxImplementationSessions") != 8:
        raise ValueError("Unexpected common cell budget")
    config = Path(grant["nativeHome"]) / "config.toml"
    if sha(config) != grant.get("nativeHomeConfigSha256"):
        raise ValueError("Sealed isolated native configuration differs")
    if not grant.get("nativeConfigurationReviewed") or not grant.get("freshContextConfirmed"):
        raise ValueError("Normal native profile/context binding required")
    if state["arm"] == "markitect" and not grant.get("markitectReadinessReceipt"):
        raise ValueError("Design must bind autonomous work-item readiness and actual installed inputs")
    if state["status"] in ("running", "unreconciled_stop"):
        raise ValueError("Unreconciled previous process: inspect its own receipt; do not blindly relaunch")
    if phase == "implementation" and len([s for s in state["sessions"] if s["phase"] == phase]) >= 8:
        raise ValueError("Implementation session envelope consumed")
    if phase == "implementation" and sum(s.get("stationIndex") == state["stationIndex"]
                                         and s["phase"] == phase for s in state["sessions"]) >= 2:
        raise ValueError("Station continuation envelope consumed")
    if phase == "assessment" and (not state.get("freeze") or any(s["phase"] == phase for s in state["sessions"])):
        raise ValueError("Exactly one independent final assessment after freeze")
    if phase == "implementation" and state.get("freeze"):
        raise ValueError("Frozen cells cannot return to implementation")
    return repo


def stop_owned(proc):
    """Only the child handle we created. No process inventory, discovery or foreign kill."""
    if proc.poll() is not None:
        return {"alreadyExited": True, "exitCode": proc.returncode}
    if os.name == "nt":
        try:
            result = subprocess.run(["taskkill", "/PID", str(proc.pid), "/T", "/F"],
                                    capture_output=True, timeout=15)
        except (OSError, subprocess.TimeoutExpired) as error:
            return {"requested": True, "localExitConfirmed": False, "errorType": type(error).__name__}
        try:
            proc.wait(timeout=10)
        except subprocess.TimeoutExpired:
            return {"requested": True, "taskkillExit": result.returncode, "localExitConfirmed": False}
        return {"requested": True, "taskkillExit": result.returncode, "localExitConfirmed": True}
    os.killpg(proc.pid, 15)
    try:
        proc.wait(timeout=10)
    except subprocess.TimeoutExpired:
        os.killpg(proc.pid, 9)
        proc.wait(timeout=5)
    return {"requested": True, "localExitConfirmed": True}


def run(state_dir, grant_file, executable, phase="implementation"):
    with locked(state_dir):
        return run_locked(state_dir, grant_file, executable, phase)


def run_locked(state_dir, grant_file, executable, phase="implementation"):
    state_dir = Path(state_dir).resolve()
    state = read(state_dir / "state.json")
    grant = read(grant_file)
    if state.get("trialGrant") and state["trialGrant"]["sha256"] != sha(grant_file):
        raise ValueError("Actual grant changed; no deadline or quota reset")
    repo = admission(state, grant, executable, phase)
    now = datetime.now(timezone.utc)
    outer_end = datetime.fromisoformat(grant["notAfterUtc"].replace("Z", "+00:00"))
    if state["deadlineUtc"] is None:
        if (outer_end - now).total_seconds() < 7200:
            raise ValueError("A full common cell window must fit before actual admission")
        from datetime import timedelta
        state["startedUtc"] = now.isoformat()
        state["deadlineUtc"] = (now + timedelta(seconds=7200)).isoformat()
        state["implementationDeadlineUtc"] = (now + timedelta(seconds=5400)).isoformat()
        state["evaluationDeadlineUtc"] = (now + timedelta(seconds=6600)).isoformat()
    from datetime import timedelta
    end = min(outer_end, datetime.fromisoformat(state["implementationDeadlineUtc"] if phase == "implementation"
                                              else state["evaluationDeadlineUtc"]))
    if phase == "implementation":
        start = datetime.fromisoformat(state["startedUtc"])
        end = min(end, start + timedelta(seconds=1350 * (state["stationIndex"] + 1)))
    if phase == "assessment":
        end = min(end, now + timedelta(seconds=1200))
    if (end - now).total_seconds() <= 30:
        raise ValueError("Phase window expired; no session or quota refill")
    session_id = state["sessionId"] if phase == "implementation" else None
    if phase == "implementation" and state["sessions"] and not session_id:
        raise ValueError("No actual own session id to resume; no fresh replacement")
    if phase == "assessment":
        repo = Path(state["freeze"]["evaluationRepo"])
        prompt = (HERE / "evaluation" / "assessor-prompt.txt").read_text(encoding="utf-8")
        prompt += "\nGesicherte Zustände (Station, Commit, lesbares Checkout):\n"
        prompt += json.dumps([{k: s[k] for k in ("station", "mainCommit", "evaluationRepo")}
                              for s in state["snapshots"]], ensure_ascii=False) + "\nFinal: " + str(repo)
    else:
        prompt = PROMPT if not session_id else CONTINUE
    number = len(state["sessions"]) + 1
    folder = state_dir / f"session-{number:02d}"
    folder.mkdir()
    (folder / "prompt.txt").write_text(prompt, encoding="utf-8", newline="\n")
    cmd = command(executable, repo, folder / "last-message.txt", session_id)
    record = {"number": number, "phase": phase, "reservedUtc": utc(), "argv": cmd,
              "cwd": str(repo), "promptSha256": sha(folder / "prompt.txt"), "executableSha256": sha(executable),
              "status": "reserved", "usage": None, "providerRequests": None, "servingModel": None,
              "cost": None, "tokensScope": "Raw own JSONL turns only; child/product receipts separately, no overlap sum"}
    record["stationIndex"] = state["stationIndex"]
    record["phaseDeadlineUtc"] = end.isoformat()
    record["nativeHomeConfigSha256"] = grant["nativeHomeConfigSha256"]
    if phase == "assessment":
        record["candidateFilesBefore"] = inventory(repo)
    state["sessions"].append(record)
    state["status"] = "running"
    state["trialGrant"] = {"path": str(Path(grant_file).resolve()), "sha256": sha(grant_file)}
    write(state_dir / "state.json", state)  # before any possible model launch
    proc = None
    started = time.monotonic()
    active_limit = max(0, (end - now).total_seconds() - 30)
    try:
        with (folder / "stdout.jsonl").open("wb") as out, (folder / "stderr.txt").open("wb") as err:
            env = os.environ.copy()
            env["CODEX_HOME"] = str(Path(grant["nativeHome"]).resolve())
            proc = subprocess.Popen(cmd, cwd=repo, env=env, stdin=subprocess.PIPE, stdout=out, stderr=err,
                                    creationflags=subprocess.CREATE_NEW_PROCESS_GROUP if os.name == "nt" else 0,
                                    start_new_session=os.name != "nt")
            record.update(status="started", pid=proc.pid, startedUtc=utc())
            write(state_dir / "state.json", state)
            proc.stdin.write(prompt.encode("utf-8"))
            proc.stdin.close()
            while proc.poll() is None:
                if datetime.now(timezone.utc) >= end - timedelta(seconds=30) or time.monotonic() - started >= active_limit:
                    record["stop"] = stop_owned(proc)
                    record["status"] = "phase_limit"
                    break
                time.sleep(0.25)
            record["exitCode"] = proc.returncode
    except Exception as error:
        record.update(status="launch_or_transport_error", errorType=type(error).__name__)
        # Exception messages may contain provider/host data; raw stderr stays private.
    finally:
        if proc is not None and proc.poll() is None:
            try:
                record["stop"] = stop_owned(proc)
            except Exception as error:
                record["stop"] = {"localExitConfirmed": False, "errorType": type(error).__name__}
        record.update(endedUtc=utc(), observedWallSeconds=time.monotonic() - started)
        if (folder / "stdout.jsonl").exists():
            events = parse_events(folder / "stdout.jsonl")
            record["events"] = events
            record["stdoutSha256"] = sha(folder / "stdout.jsonl")
            record["stderrSha256"] = sha(folder / "stderr.txt")
            if phase == "implementation" and events["sessionId"]:
                if state["sessionId"] not in (None, events["sessionId"]):
                    record["sessionIdentityMismatch"] = True
                else:
                    state["sessionId"] = events["sessionId"]
        if record["status"] == "started":
            record["status"] = "returned" if record.get("exitCode") == 0 else "failed"
        if phase == "assessment":
            record["candidateFilesAfter"] = inventory(repo)
            record["candidateUnchanged"] = record["candidateFilesBefore"] == record["candidateFilesAfter"]
            record["stationCandidatesUnchanged"] = [
                {"station": s["station"], "unchanged": git(s["evaluationRepo"], "status", "--porcelain") == "",
                 "headMatches": git(s["evaluationRepo"], "rev-parse", "HEAD") == s["mainCommit"]}
                for s in state["snapshots"]]
        if record.get("stop", {}).get("localExitConfirmed") is False:
            record["status"] = "unreconciled_stop"
        state["status"] = record["status"]
        write(state_dir / "state.json", state)
    return record


def snapshot(state_dir):
    state_dir = Path(state_dir).resolve()
    state = read(state_dir / "state.json")
    if state["status"] in ("running", "unreconciled_stop"):
        raise ValueError("Cannot snapshot an unconfirmed live process")
    index = state["stationIndex"]
    if any(s["stationIndex"] == index for s in state["snapshots"]):
        raise ValueError("Station already retained")
    repo = Path(state["repo"])
    main = git(repo, "rev-parse", "main")
    folder = state_dir / "stations" / state["stations"][index]["id"]
    folder.mkdir(parents=True)
    raw = folder / "working-tree"
    shutil.copytree(repo, raw, ignore=shutil.ignore_patterns(".git"))
    git(repo, "bundle", "create", str(folder / "history.bundle"), "--all")
    evaluation = folder / "main"
    subprocess.run(["git", "-c", "core.autocrlf=false", "clone", "--no-hardlinks", str(repo), str(evaluation)],
                   check=True, capture_output=True, timeout=45)
    git(evaluation, "remote", "remove", "origin")
    git(evaluation, "checkout", "--detach", main)
    item = {"stationIndex": index, "station": state["stations"][index]["id"],
            "mainCommit": main, "headCommit": git(repo, "rev-parse", "HEAD"),
            "dirtyStatus": git(repo, "status", "--porcelain"), "evaluationRepo": str(evaluation),
            "rawFiles": inventory(raw), "bundleSha256": sha(folder / "history.bundle"), "utc": utc()}
    state["snapshots"].append(item)
    write(state_dir / "state.json", state)
    return item


def advance(state_dir):
    state_dir = Path(state_dir).resolve()
    state = read(state_dir / "state.json")
    index = state["stationIndex"]
    if not any(s["stationIndex"] == index for s in state["snapshots"]):
        raise ValueError("Retain station before advancing")
    if index + 1 >= len(state["stations"]):
        return False
    state["stationIndex"] += 1
    # Only a predeclared public scheduler file is changed; never code or model.
    write(Path(state["repo"]) / ".study" / "station.json", state["stations"][index + 1])
    write(state_dir / "state.json", state)
    return True


def freeze(state_dir):
    state_dir = Path(state_dir).resolve()
    state = read(state_dir / "state.json")
    if state["status"] in ("running", "unreconciled_stop") or state.get("freeze"):
        raise ValueError("Must be terminal and not already frozen")
    repo = Path(state["repo"])
    main = git(repo, "rev-parse", "main")
    head = git(repo, "rev-parse", "HEAD")
    dirty = git(repo, "status", "--porcelain")
    # Keep actual dirty bytes as evidence even when main integration was not completed.
    raw = state_dir / "final-working-tree"
    shutil.copytree(repo, raw, ignore=shutil.ignore_patterns(".git"))
    git(repo, "bundle", "create", str(state_dir / "history.bundle"), "--all")
    evaluation = state_dir / "evaluation" / "repo"
    evaluation.parent.mkdir()
    subprocess.run(["git", "-c", "core.autocrlf=false", "clone", "--no-hardlinks", str(repo), str(evaluation)], check=True, capture_output=True, timeout=45)
    git(evaluation, "remote", "remove", "origin")
    git(evaluation, "checkout", "--detach", main)
    state["freeze"] = {"frozenUtc": utc(), "mainCommit": main, "headCommit": head, "dirtyStatus": dirty,
                       "mergedAndClean": not dirty and head == main, "mainTree": git(repo, "rev-parse", "main^{tree}"),
                       "rawWorkingFiles": inventory(raw), "historyBundleSha256": sha(state_dir / "history.bundle"),
                       "evaluationRepo": str(evaluation), "assessedScope": "isolated main; unfinished tree retained separately"}
    state["status"] = "frozen"
    write(state_dir / "state.json", state)
    return state["freeze"]


def log_effort(state_dir, category, note, seconds=None):
    if category not in ("setup", "model-preparation", "model-upkeep", "repair", "review", "merge", "oversight", "context-resume"):
        raise ValueError("Unknown accounting category")
    state_dir = Path(state_dir)
    state = read(state_dir / "state.json")
    state["interventions"].append({"utc": utc(), "category": category, "note": note,
                                   "measuredActiveSeconds": seconds, "overlapUnresolved": True})
    write(state_dir / "state.json", state)


def setup_seal(state_dir):
    """Record legitimate installed method inputs; does not install or repair a product."""
    state_dir = Path(state_dir).resolve()
    state = read(state_dir / "state.json")
    if state["sessions"] or state.get("freeze"):
        raise ValueError("Setup cannot be replaced after trial activation")
    repo = Path(state["repo"])
    if git(repo, "status", "--porcelain"):
        raise ValueError("Installed setup must be committed by its owner")
    state["initialSetup"].append({"sealedUtc": utc(), "commit": git(repo, "rev-parse", "HEAD"),
                                  "files": inventory(repo), "initialPreparationCost": None})
    state["preparedCommit"] = git(repo, "rev-parse", "HEAD")
    state["preparedTree"] = git(repo, "rev-parse", "HEAD^{tree}")
    write(state_dir / "state.json", state)
    return state["initialSetup"][-1]


def drive(state_dir, grant, executable):
    with locked(state_dir):
        return drive_locked(state_dir, grant, executable)


def drive_locked(state_dir, grant, executable):
    """Fixed scheduler reads only completion metadata and process receipts, never source."""
    state_dir = Path(state_dir).resolve()
    initial = read(state_dir / "state.json")
    admission(initial, read(grant), executable, "implementation")
    if initial.get("trialGrant") and initial["trialGrant"]["sha256"] != sha(grant):
        raise ValueError("Actual grant changed; no retry or quota reset")
    while True:
        state = read(state_dir / "state.json")
        if state.get("freeze"):
            raise ValueError("Cell already frozen; no replay")
        consumed = sum(s["phase"] == "implementation" and s.get("stationIndex") == state["stationIndex"]
                       for s in state["sessions"])
        try:
            receipt = run_locked(state_dir, grant, executable) if consumed < 2 else None
        except ValueError as error:
            # No hidden retry; preserve explicit finite admission failure.
            state["dispatchStop"] = str(error)
            write(state_dir / "state.json", state)
            receipt = None
        if receipt and receipt["status"] == "unreconciled_stop":
            return {"status": "blocked", "reason": "Owned stop not confirmed; no next actor"}
        state = read(state_dir / "state.json")
        completion = Path(state["repo"]) / ".study" / "completion.json"
        if completion.exists():
            try:
                metadata = read(completion)
                status = metadata.get("status") if metadata.get("station") == state["stations"][state["stationIndex"]]["id"] else None
            except (ValueError, OSError):
                status = None
        else:
            status = None
        terminal = status in ("complete", "blocked") or receipt is None or receipt["status"] != "returned" or receipt.get("events", {}).get("errors")
        if terminal:
            snapshot(state_dir)
            if status == "complete" and receipt and receipt["status"] == "returned" and not receipt.get("events", {}).get("errors") and advance(state_dir):
                continue
            break
    final = freeze(state_dir)
    # Only the frozen main is assessed. Completeness/quality is decided independently.
    try:
        result = run_locked(state_dir, grant, executable, "assessment")
    except ValueError as error:
        result = {"status": "NOT RUN", "reason": str(error)}
        state = read(state_dir / "state.json")
        state["assessment"] = result
        write(state_dir / "state.json", state)
    return {"freeze": final, "assessment": result}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="action", required=True)
    p = commands.add_parser("seed"); p.add_argument("case"); p.add_argument("target")
    p = commands.add_parser("cell"); p.add_argument("case"); p.add_argument("arm"); p.add_argument("seed"); p.add_argument("target"); p.add_argument("state")
    p = commands.add_parser("plan"); p.add_argument("state"); p.add_argument("executable")
    p = commands.add_parser("run"); p.add_argument("state"); p.add_argument("grant"); p.add_argument("executable"); p.add_argument("--phase", choices=["implementation", "assessment"], default="implementation")
    p = commands.add_parser("freeze"); p.add_argument("state")
    p = commands.add_parser("seal-setup"); p.add_argument("state")
    p = commands.add_parser("drive"); p.add_argument("state"); p.add_argument("grant"); p.add_argument("executable")
    p = commands.add_parser("effort"); p.add_argument("state"); p.add_argument("category"); p.add_argument("note"); p.add_argument("--seconds", type=float)
    args = parser.parse_args()
    if args.action == "seed": result = seed(args.case, args.target)
    elif args.action == "cell": result = cell(args.case, args.arm, args.seed, args.target, args.state)
    elif args.action == "plan":
        s = read(Path(args.state) / "state.json")
        result = {"executionAuthorized": False, "argv": command(args.executable, s["repo"], Path(args.state) / "last-message.txt", s["sessionId"]), "prompt": PROMPT if not s["sessionId"] else CONTINUE, "cwd": s["repo"]}
    elif args.action == "run": result = run(args.state, args.grant, args.executable, args.phase)
    elif args.action == "freeze": result = freeze(args.state)
    elif args.action == "seal-setup": result = setup_seal(args.state)
    elif args.action == "drive": result = drive(args.state, args.grant, args.executable)
    else: result = log_effort(args.state, args.category, args.note, args.seconds)
    print(json.dumps(result, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
