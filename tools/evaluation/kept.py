"""The lasting store: one folder per launch, with its final repository, transcripts, Result and cost (concept 3.0 §13).

  python -I -B kept.py add --store DIR --name NAME --project P --step S --arm ARM --run RUN_DIR
                           [--judgment DIR] [--weekly BEFORE,AFTER] [--note TEXT]
  python -I -B kept.py check --store DIR
  python -I -B kept.py list --store DIR

Hidden tests cannot tell two results apart when both pass, so what a launch made has to outlast the work folder it ran
in. A kept launch never changes: `add` refuses a name it already holds, and `check` compares every file with the hash
it had when it was kept. `index.jsonl` holds each launch's name and the hash of its `launch.json`, so a changed record or a
deleted launch shows too.

  NAME/launch.json       who, what, on which model, how long and at what cost, and every file's hash
  NAME/repo.bundle       the final repository with its whole history
  NAME/records/          the run's own records: suite.json, run.json, judgment.json
  NAME/transcripts/      the session's stream and every transcript, subagents included

`add` refuses a bundle that is not complete, and any kept text that holds an Anthropic key's shape or the value of
MARKITECT_CLAUDE_TOKEN. The weekly reading is a person's or a runner's reading from before and after the launch; nothing
here can read it.
"""
import argparse
import hashlib
import json
import os
import pathlib
import re
import shutil
import subprocess
import sys
import tempfile

FORMAT = 1
TOKEN_VARIABLE = "MARKITECT_CLAUDE_TOKEN"
KEY_SHAPE = re.compile(rb"sk-ant-[A-Za-z0-9_-]{20,}")
SAFE_NAME = re.compile(r"^[a-z0-9][a-z0-9._-]*$")
INDEX = "index.jsonl"


class KeptError(Exception):
    pass


def git(*args, cwd=None):
    p = subprocess.run(["git", *args], cwd=cwd, capture_output=True, text=True, encoding="utf-8", errors="replace")
    if p.returncode != 0:
        raise KeptError(f"git {' '.join(args)}: {p.stderr.strip()}")
    return p.stdout


def digest(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for block in iter(lambda: f.read(1 << 20), b""):
            h.update(block)
    return h.hexdigest()


def files_of(folder):
    return sorted(p for p in pathlib.Path(folder).rglob("*") if p.is_file())


def holds_secret(folder):
    """The first file under folder with a key's shape or the token's value in it."""
    token = os.environ.get(TOKEN_VARIABLE, "").encode()
    for p in files_of(folder):
        data = p.read_bytes()
        if KEY_SHAPE.search(data) or (token and token in data):
            return p
    return None


def models_in(folder):
    """What the transcripts show: the messages per model that answered, and how many subagents ran."""
    models, subagents = {}, 0
    for p in files_of(folder):
        if p.suffix != ".jsonl" or p.name == "session.stream.jsonl":
            continue
        subagents += "subagents" in p.parts
        with open(p, encoding="utf-8", errors="replace") as f:
            for line in f:
                try:
                    record = json.loads(line)
                except ValueError:
                    continue
                message = record.get("message") if isinstance(record, dict) else None
                model = message.get("model") if isinstance(message, dict) else None
                if record.get("type") == "assistant" and isinstance(model, str) and not model.startswith("<"):
                    models[model] = models.get(model, 0) + 1
    return models, subagents


def cost_in(folder):
    """The sum of what each session's closing record reports, for a Command, whose Result counts tokens only. None without any.

    The provider's figure, kept only as a relative measure against another launch."""
    total, found = 0.0, False
    for p in files_of(folder):
        if p.suffix != ".jsonl" or p.name == "session.stream.jsonl" or "subagents" in p.parts:
            continue
        with open(p, encoding="utf-8", errors="replace") as f:
            for line in f:
                if '"total_cost_usd"' not in line:
                    continue
                try:
                    record = json.loads(line)
                except ValueError:
                    continue
                if isinstance(record, dict) and record.get("type") == "result" and isinstance(record.get("total_cost_usd"), (int, float)):
                    total += record["total_cost_usd"]
                    found = True
    return round(total, 4) if found else None


def read_json(path):
    return json.loads(pathlib.Path(path).read_text(encoding="utf-8"))


def add(store, name, project, step, arm, run_dir, judgment=None, weekly=None, note=None):
    """Keep one suite run. Returns the launch's record."""
    if not SAFE_NAME.match(name):
        raise KeptError(f"{name!r} is not a name for a kept launch")
    store, run_dir = pathlib.Path(store), pathlib.Path(run_dir)
    dest = store / name
    if dest.exists():
        raise KeptError(f"{dest} is already kept, and a kept launch never changes")
    suite = read_json(run_dir / "suite.json")
    variant = run_dir / "variant"
    bundle = variant / "result.bundle"
    commit = (suite.get("result") or {}).get("commit")
    if not suite.get("delivered") or not commit or not bundle.is_file():
        raise KeptError(f"{run_dir} delivered no result to keep")
    with tempfile.TemporaryDirectory(ignore_cleanup_errors=True) as empty:
        # An empty repository can only hold a bundle that needs nothing else.
        git("init", "-q", "--bare", empty)
        verified = subprocess.run(["git", "bundle", "verify", str(bundle.resolve())], cwd=empty, capture_output=True, text=True, encoding="utf-8", errors="replace")
    if verified.returncode != 0 or "complete history" not in verified.stdout + verified.stderr:
        raise KeptError(f"{bundle} is not a complete repository: {verified.stderr.strip()}")
    temporary = store / f".adding-{name}"
    if temporary.exists():
        shutil.rmtree(temporary)
    (temporary / "records").mkdir(parents=True)
    (temporary / "transcripts").mkdir()
    try:
        shutil.copyfile(bundle, temporary / "repo.bundle")
        for source, kept in ((run_dir / "suite.json", "suite.json"), (variant / "run.json", "run.json")):
            if source.is_file():
                shutil.copyfile(source, temporary / "records" / kept)
        if judgment:
            shutil.copyfile(pathlib.Path(judgment) / "judgment.json", temporary / "records" / "judgment.json")
        if (variant / "transcripts").is_dir():
            shutil.copytree(variant / "transcripts", temporary / "transcripts", dirs_exist_ok=True)
        if (variant / "session.stream.jsonl").is_file():
            shutil.copyfile(variant / "session.stream.jsonl", temporary / "transcripts" / "session.stream.jsonl")
        leak = holds_secret(temporary)
        if leak:
            # The name of the file only: the value never reaches a log.
            raise KeptError(f"{leak.relative_to(temporary)} holds a credential, so this launch is not kept")
        run = read_json(variant / "run.json") if (variant / "run.json").is_file() else {}
        # The arm's record names the Command's total `tokens`; a session's names `usage`.
        usage = dict(run.get("usage") or run.get("tokens") or {})
        relative = usage.pop("costUsd", None)
        if relative is None:
            relative = cost_in(temporary / "transcripts")
        models, subagents = models_in(temporary / "transcripts")
        correctness = None
        if judgment:
            areas = read_json(temporary / "records" / "judgment.json").get("areas") or {}
            c = areas.get("correctness") or {}
            correctness = {k: c.get(k) for k in ("planned", "passed", "failed")}
        record = {
            "format": FORMAT, "name": name, "project": project, "step": step, "arm": arm,
            "requestedModel": run.get("requestedModel") or run.get("model"), "modelsAnswered": models, "subagents": subagents,
            "commit": commit, "startedUtc": suite.get("startedUtc"), "endedUtc": suite.get("endedUtc"),
            "wallSeconds": suite.get("wallSeconds"), "usage": usage, "relativeCost": relative,
            "weekly": {"before": weekly[0], "after": weekly[1]} if weekly else None,
            "hiddenCases": correctness, "note": note,
            "files": {p.relative_to(temporary).as_posix(): digest(p) for p in files_of(temporary)},
        }
        (temporary / "launch.json").write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
        os.replace(temporary, dest)
        with open(store / INDEX, "a", encoding="utf-8", newline="\n") as f:
            f.write(json.dumps({"name": name, "launchSha256": digest(dest / "launch.json")}) + "\n")
    except BaseException:
        shutil.rmtree(temporary, ignore_errors=True)
        raise
    return record


def launches(store):
    return sorted(p.parent for p in pathlib.Path(store).glob("*/launch.json"))


def check(store):
    """What no longer matches its hash, per launch, and any launch deleted or added behind the index's back."""
    problems = []
    indexed = {}
    if pathlib.Path(store, INDEX).is_file():
        for line in pathlib.Path(store, INDEX).read_text(encoding="utf-8").splitlines():
            entry = json.loads(line)
            indexed[entry["name"]] = entry["launchSha256"]
    held_launches = {folder.name for folder in launches(store)}
    for name in sorted(set(indexed) - held_launches):
        problems.append(f"{name}: the launch is gone")
    for name in sorted(held_launches - set(indexed)):
        problems.append(f"{name}: not in the index")
    for name in sorted(held_launches & set(indexed)):
        if digest(pathlib.Path(store, name, "launch.json")) != indexed[name]:
            problems.append(f"{name}: launch.json changed")
    for folder in launches(store):
        record = read_json(folder / "launch.json")
        held = {p.relative_to(folder).as_posix() for p in files_of(folder)} - {"launch.json"}
        for name in sorted(held ^ set(record["files"])):
            problems.append(f"{folder.name}: {name} is {'not recorded' if name in held else 'missing'}")
        for name in sorted(held & set(record["files"])):
            if digest(folder / name) != record["files"][name]:
                problems.append(f"{folder.name}: {name} changed")
    return problems


def clone(folder, dest):
    """A working copy of a kept launch's final state, with its history."""
    folder, dest = pathlib.Path(folder), pathlib.Path(dest)
    record = read_json(folder / "launch.json")
    git("init", "-q", str(dest))
    git("fetch", "-q", str((folder / "repo.bundle").resolve()), f"+{record['commit']}:refs/heads/kept", cwd=dest)
    git("-c", "core.autocrlf=false", "checkout", "-q", "kept", cwd=dest)
    return record


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="cmd", required=True)
    a = sub.add_parser("add")
    a.add_argument("--store", required=True)
    a.add_argument("--name", required=True)
    a.add_argument("--project", required=True)
    a.add_argument("--step", required=True)
    a.add_argument("--arm", required=True)
    a.add_argument("--run", required=True)
    a.add_argument("--judgment")
    a.add_argument("--weekly", help="the weekly reading before and after, as BEFORE,AFTER")
    a.add_argument("--note")
    for name in ("check", "list"):
        sub.add_parser(name).add_argument("--store", required=True)
    args = parser.parse_args(argv)
    try:
        if args.cmd == "add":
            weekly = tuple(float(v) for v in args.weekly.split(",")) if args.weekly else None
            pathlib.Path(args.store).mkdir(parents=True, exist_ok=True)
            record = add(args.store, args.name, args.project, args.step, args.arm, args.run, args.judgment, weekly, args.note)
            print(json.dumps({k: record[k] for k in ("name", "commit", "wallSeconds", "modelsAnswered", "subagents", "hiddenCases")}))
        elif args.cmd == "check":
            problems = check(args.store)
            print("\n".join(problems) or f"{len(launches(args.store))} kept launches match their hashes")
            return 1 if problems else 0
        else:
            for folder in launches(args.store):
                r = read_json(folder / "launch.json")
                print(f"{r['name']}  {r['project']} {r['step']}  {r['arm']}  {r['requestedModel']}  {r['wallSeconds']} s")
    except (KeptError, OSError, ValueError, KeyError) as e:
        print(f"kept: {type(e).__name__}: {e}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
