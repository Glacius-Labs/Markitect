"""Judges: each task's hidden judge, run the same way for every variant.

A judge folder holds `judge.json` and the judge's files (oracle, probes,
rubric). `judge.json` names the judge image by digest, the judge's command, a
time limit and its container limits:

  {"format": 1, "name": "...", "image": "NAME@sha256:..." or "sha256:...", "command": ["python3", "/judge/judge.py"],
   "timeoutSeconds": 2400, "limits": {"cpus": "4", "memoryBytes": N, "pids": N, "tmpBytes": N}}

The suite appends TASK_DIR START_BUNDLE RESULT_BUNDLE OUT_DIR and runs the
command in one container without network: the judge folder at /judge, the
task at /task, the start bundle at /start/start.bundle and the result bundle
at /result/result.bundle, all read-only; /out writable; a read-only root, no
capabilities, a non-root user and a private /tmp. The judge writes
/out/judge.json with three areas:

  correctness     cases planned, passed and failed, and the failed cases' IDs
  buildAndTests   the result's own build and tests, each with "passed"
  scope           every changed path, the unrequested ones and any protected input touched

A series judge judges every step of a change series. Its `judge.json` lists
the `steps`; the suite tells it the step in EVALUATION_STEP, and it answers
for that step and every earlier one, cumulatively: `step`,
`correctness.byStep` (planned, passed and failed per step that introduced the
cases, for every step up to this one) and `drift` (forgottenRules: earlier
steps' cases that fail; staleFiles: retired terms left in the source;
unaccounted: files nothing accounts for, or null when that is unknown).

A planted task's judge folder also holds `plant.json`, written before any run:

  {"format": 1, "kind": "defect" or "gap", "expected": "no-landing" or "landing",
   "description": "...", "recordedUtc": "2026-10-28T12:00:00+00:00"}

A planted model defect expects no landing, a harmless gap a landing. The
suite scores the outcome itself (`stops`): a result equal to the start, with
no changed path, did not land. A folder with `plant.json` and no `judge.json`
judges the outcome only.

A judge folder in this repository is a source: `judges.py export` writes the
self-contained folder the suite runs, with every included file taken from one
frozen revision and its hash recorded.
"""
import datetime
import hashlib
import json
import pathlib
import re
import shutil
import tempfile
import uuid

import sandbox as sandboxing

FORMAT = 1
LABEL_KEY = "markitect.evaluation.judge"
IMAGE = re.compile(r"(sha256:[0-9a-f]{64}|[a-z0-9][a-z0-9._/-]*(:[A-Za-z0-9._-]+)?@sha256:[0-9a-f]{64})")
MOUNTS = ("/task", "/start/start.bundle", "/result/result.bundle", "/out")
KILL_SECONDS = 20


class JudgeError(Exception):
    pass


def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def load(judge_dir):
    """The judge's configuration, checked before anything runs."""
    path = pathlib.Path(judge_dir) / "judge.json"
    try:
        config = json.loads(path.read_text(encoding="utf-8"))
        limits = config["limits"]
        checked = {"cpus": str(limits["cpus"]), "memoryBytes": int(limits["memoryBytes"]), "pids": int(limits["pids"]),
                   "tmpBytes": int(limits["tmpBytes"])}
        float(checked["cpus"])
        seconds = int(config["timeoutSeconds"])
    except (OSError, ValueError, KeyError, TypeError) as e:
        raise JudgeError(f"{path}: not a judge configuration: {e}")
    if config.get("format") != FORMAT:
        raise JudgeError(f"{path}: format {config.get('format')} is not {FORMAT}")
    if "include" in config:
        raise JudgeError(f"{path}: a judge source, not an exported judge folder; export it with judges.py first")
    if not isinstance(config.get("image"), str) or not IMAGE.fullmatch(config["image"]):
        raise JudgeError(f"{path}: the image must be named by digest (NAME@sha256:... or sha256:...)")
    command = config.get("command")
    if not isinstance(command, list) or not command or not all(isinstance(c, str) and c for c in command):
        raise JudgeError(f"{path}: the command must be a non-empty list of strings")
    if seconds <= 0 or min(float(checked["cpus"]), checked["memoryBytes"], checked["pids"], checked["tmpBytes"]) <= 0:
        raise JudgeError(f"{path}: the time limit and every container limit must be positive")
    steps = config.get("steps")
    if steps is not None and (not isinstance(steps, list) or not steps or not all(isinstance(s, str) and s for s in steps)
                              or len(set(steps)) != len(steps)):
        raise JudgeError(f"{path}: steps must be a list of distinct step names")
    return {"name": config.get("name"), "image": config["image"], "command": command, "timeoutSeconds": seconds, "limits": checked,
            "reviewEntry": config.get("reviewEntry"), "steps": steps}


def load_plant(judge_dir):
    """The planted task's expectation, or None when the folder has no plant.json."""
    path = pathlib.Path(judge_dir) / "plant.json"
    if not path.is_file():
        return None
    try:
        plant = json.loads(path.read_text(encoding="utf-8"))
        recorded = datetime.datetime.fromisoformat(plant["recordedUtc"])
    except (OSError, ValueError, KeyError, TypeError) as e:
        raise JudgeError(f"{path}: not a plant record: {e}")
    if plant.get("format") != FORMAT or recorded.tzinfo is None:
        raise JudgeError(f"{path}: format {plant.get('format')} is not {FORMAT}, or recordedUtc has no time zone")
    expected = {"defect": "no-landing", "gap": "landing"}.get(plant.get("kind"))
    if expected is None or plant.get("expected") != expected or not plant.get("description"):
        raise JudgeError(f"{path}: a defect expects no-landing, a gap expects landing, and each needs a description")
    return plant


def stops(plant, run_record):
    """The planted task's outcome: whether the result landed, and whether that is what the plant expects."""
    landed = bool((run_record.get("result") or {}).get("changedPaths"))
    started = datetime.datetime.fromisoformat(run_record["startedUtc"])
    recorded = datetime.datetime.fromisoformat(plant["recordedUtc"])
    return {"kind": plant["kind"], "expected": plant["expected"], "landed": landed,
            "correct": landed == (plant["expected"] == "landing"), "recordedBeforeRun": recorded < started}


def folder_digest(root):
    """One digest over every file of a folder: paths and bytes."""
    h = hashlib.sha256()
    root = pathlib.Path(root)
    for p in sorted(p for p in root.rglob("*") if p.is_file()):
        h.update(p.relative_to(root).as_posix().encode() + b"\0" + p.read_bytes() + b"\0")
    return h.hexdigest()


def checked_series(value, step, steps):
    """Problems with a series judge's step, byStep and drift; an empty list when they hold for this step."""
    problems = []
    if value.get("step") != step:
        problems.append(f"the judge judged step {value.get('step')}, not {step}")
    c = value.get("correctness") if isinstance(value.get("correctness"), dict) else {}
    by = c.get("byStep")
    upto = steps[:steps.index(step) + 1]
    if not isinstance(by, dict) or list(by) != upto or not all(isinstance(by[s], dict) for s in upto):
        problems.append("correctness.byStep needs every step up to this one, in order: the judgment must be cumulative")
    elif any(sum(by[s].get(k, 0) for s in upto) != c.get(k) for k in ("planned", "passed", "failed")) or \
            any(by[s].get("passed", 0) + by[s].get("failed", 0) != by[s].get("planned") for s in upto):
        problems.append("correctness.byStep does not add up to the step's counts")
    d = value.get("drift")
    if not isinstance(d, dict) or not isinstance(d.get("forgottenRules"), list) or not isinstance(d.get("staleFiles"), list) \
            or not (d.get("unaccounted") is None or isinstance(d.get("unaccounted"), list)):
        problems.append("drift needs forgottenRules and staleFiles as lists and unaccounted as a list or null")
    return problems


def checked_output(value, commit):
    """Problems with a judge's judge.json; an empty list when it holds the three areas for this result."""
    if not isinstance(value, dict):
        return ["judge.json is not a JSON object"]
    if "error" in value:
        return ["the judge reports an error: " + str(value["error"])]
    problems = []
    if value.get("commit") != commit:
        problems.append(f"the judge judged {value.get('commit')}, not the result {commit}")
    c = value.get("correctness")
    if not isinstance(c, dict) or not all(isinstance(c.get(k), int) and not isinstance(c.get(k), bool) for k in ("planned", "passed", "failed")) \
            or not isinstance(c.get("failedCases"), list) or c["passed"] + c["failed"] != c["planned"] or len(c["failedCases"]) != c["failed"]:
        problems.append("correctness needs planned, passed and failed counts that add up, and the failed cases")
    b = value.get("buildAndTests")
    if not isinstance(b, dict) or not all(isinstance(b.get(k), dict) and isinstance(b[k].get("passed"), bool) for k in ("build", "tests")):
        problems.append("buildAndTests needs build and tests, each with passed")
    s = value.get("scope")
    if not isinstance(s, dict) or not isinstance(s.get("changedPaths"), list) or \
            not all(isinstance(s.get(k), list) and all(isinstance(p, str) for p in s[k]) for k in ("unrequested", "protectedTouched")):
        problems.append("scope needs changedPaths, and unrequested and protectedTouched as lists of paths")
    return problems


def container_argv(name, label, config, user, mounts, workdir="/tmp", step=None):
    limits = config["limits"]
    argv = ["run", "--name", name, "--label", label, "--init", "--network", "none", *sandboxing.hardening(user),
            "--tmpfs", f"/tmp:rw,exec,nosuid,nodev,mode=1777,size={limits['tmpBytes']}",
            "--cpus", limits["cpus"], "--memory", str(limits["memoryBytes"]), "--memory-swap", str(limits["memoryBytes"]),
            "--pids-limit", str(limits["pids"]), "-e", "HOME=/tmp", "-e", f"EVALUATION_JUDGE_CONTAINER={name}", "--workdir", workdir]
    if step is not None:
        argv += ["-e", f"EVALUATION_STEP={step}"]
    for src, dst, ro in mounts:
        argv += ["--mount", f"type=bind,src={src},dst={dst}" + (",readonly" if ro else "")]
    return argv + [config["image"], *config["command"], *MOUNTS]


def run_judge(engine_argv, judge_dir, task_dir, result_bundle, commit, step=None):
    """Run the judge on one result; returns the suite's judgment record, with the judge's own output under "areas".

    A series judge needs the step it judges; a single-step judge takes none."""
    judge_dir, task_dir = pathlib.Path(judge_dir), pathlib.Path(task_dir)
    config = load(judge_dir)
    if config["steps"] and step not in config["steps"]:
        raise JudgeError(f"the judge judges the steps {', '.join(config['steps'])}; name one with --step, not {step!r}")
    if not config["steps"] and step is not None:
        raise JudgeError("the judge has no steps; it takes no --step")
    engine = sandboxing.Engine(engine_argv)
    run_id = "judge-" + uuid.uuid4().hex[:12]
    label = f"{LABEL_KEY}={run_id}"
    record = {"format": FORMAT, "judge": {**config, "folderSha256": folder_digest(judge_dir)}, "commit": commit, "step": step,
              "startedUtc": now(), "steps": engine.steps, "problems": []}
    output = None
    with tempfile.TemporaryDirectory() as tmp:
        tmp = pathlib.Path(tmp)
        # The container sees copies: the originals stay out of its reach whatever the mounts allow.
        shutil.copytree(judge_dir, tmp / "judge")
        shutil.copytree(task_dir, tmp / "task")
        shutil.copyfile(result_bundle, tmp / "result.bundle")
        (tmp / "out").mkdir()
        try:
            code, image_id = engine("image", ["image", "inspect", "--format", "{{.Id}}", config["image"]])
            if code != 0:
                raise JudgeError(f"the judge image {config['image']} is not on this engine; pull it first")
            record["judge"]["imageId"] = image_id.strip()
            mounts = [(sandboxing.host_path(tmp / "judge"), "/judge", True), (sandboxing.host_path(tmp / "task"), "/task", True),
                      (sandboxing.host_path(tmp / "task" / "start.bundle"), "/start/start.bundle", True),
                      (sandboxing.host_path(tmp / "result.bundle"), "/result/result.bundle", True),
                      (sandboxing.host_path(tmp / "out"), "/out", False)]
            argv = container_argv(run_id, label, config, sandboxing.container_user(), mounts, step=step)
            record["argv"] = argv
            code, _ = engine("judge", argv, config["timeoutSeconds"])
            record["container"] = {"exit": code, "timedOut": code is None and engine.steps[-1].get("stderr") == "timed out"}
            if code is None:
                engine("judge-kill", ["kill", run_id], KILL_SECONDS)
        except (JudgeError, sandboxing.SandboxError) as e:
            record["problems"].append(str(e))
        finally:
            engine("judge-remove", ["rm", "-f", run_id], KILL_SECONDS)
            try:
                record["residue"] = sandboxing.census(engine, "judge-census", label)
            except sandboxing.SandboxError as e:
                record["residue"] = [f"unknown: {e}"]
        result = tmp / "out" / "judge.json"
        if result.is_file() and not result.is_symlink():
            raw = result.read_bytes()
            record["outputSha256"] = hashlib.sha256(raw).hexdigest()
            try:
                output = json.loads(raw)
            except ValueError:
                record["problems"].append("judge.json is not JSON")
        elif "container" in record:
            record["problems"].append("the judge wrote no judge.json")
    if output is not None:
        record["problems"] += checked_output(output, commit)
        if config["steps"] and isinstance(output, dict) and "error" not in output:
            record["problems"] += checked_series(output, step, config["steps"])
    container = record.get("container")
    if container and container["timedOut"]:
        record["problems"].append("the judge timed out")
    elif container and container["exit"] is None:
        record["problems"].append("the engine could not run the judge")
    elif container and container["exit"] != 0 and not record["problems"]:
        record["problems"].append(f"the judge exited {container['exit']}")
    if record["residue"]:
        record["problems"].append("residue: " + ", ".join(record["residue"]))
    record["valid"] = not record["problems"]
    record["areas"] = {k: output.get(k) for k in ("correctness", "buildAndTests", "scope", "drift") if k in output}         if isinstance(output, dict) else None
    record["output"] = output
    record["endedUtc"] = now()
    return record
