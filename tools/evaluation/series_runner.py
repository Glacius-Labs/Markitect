"""The series runner: a counted series or its stability gate, one launch at a time through suite.py, with its judging,
reviews, ledger and report.

  python -I -B series_runner.py plan --protocol counted|gate --machine MACHINE.json [--prepared DIR ...] --out CONFIG.json
  python -I -B series_runner.py units CONFIG.json
  python -I -B series_runner.py run CONFIG.json [--prep-only]
  python -I -B series_runner.py judge CONFIG.json

`plan` writes a series configuration: MACHINE.json (`work`, `rev`, `bin`, `managerImage`, `sandboxImage`,
`lock`, `hidden`, `login`, `model`, optionally `docker`) with the protocol's units in their order, for the Library, the
Makerspace and each `--prepared` series. `units` prints the launch order and its counts and starts nothing. `run`
prepares the public series' inputs and runs every unit not yet done; started again after a stop, it goes on where it
stopped. `judge` judges and reviews what a series whose judge stays off the machine during launches left pending, then
writes the report again.

Run it from the frozen revision's tools (`git archive REV tools` into WORK/src) and start it detached (Win32_Process
Create on Windows, setsid nohup on Linux), without a shell's ANTHROPIC_* or CLAUDE_* variables. It writes
WORK/series.jsonl (one line per event), WORK/ledger.json, the report and WORK/done.json, and stops before the next launch
when WORK/STOP exists.

The runner never holds the shared engine lock: suite.py takes it for each run, judgment and review, and a busy lock is
waited for and not counted. Before each launch it waits until the engine is quiet (Guard) and the login outlives the
launch's deadline by twenty minutes (Login). Each launch is classed with its arm's variant runtime signatures, as the
report classes it. A void launch is relaunched up to `relaunches` times; a launch invalid only by containers of other
work (no `markitect.*` label, as the run's own engine events show their start) is relaunched once; any other invalid
launch stops the series. So do a refused request or a seven-day use of 90 %, a credential in a run's evidence, a
gateway refusal of a provider name, hidden material on the machine, and a void share above one in four for an arm after
its first eight launches; a five-hour use of 85 % waits for that window's reset.
"""
import argparse
import datetime
import glob
import hashlib
import json
import os
import pathlib
import re
import shutil
import subprocess
import sys
import tempfile
import threading
import time

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import classify  # noqa: E402

DEADLINES = {"library": 2700, "makerspace": 5400}
# The 2.0 arm's variant runtime signatures (markitect_arm.py), written before any launch.
SIGNATURES = {"variants": {"M": [
    {"id": "arm-could-not-run", "path": "reported.status", "pattern": "^(could-not-run|check-failed)$"},
    {"id": "arm-no-agent-turn", "path": "reported.noAgentTurn", "pattern": "^true$"}]}}
REPETITIONS = {"public": {"fixed": 2, "plant": 2, "chain": 1}, "confirmatory": {"fixed": 5, "plant": 3, "chain": 2}}
GATE_FIXED, GATE_CHAIN = ("library", ["M2", "M3", "M4"]), "makerspace"
QUIET_SECONDS = 600
LOGIN_MARGIN = 20 * 60
QUOTA_STOP, QUOTA_WAIT = 0.90, 0.85
VOID_SHARE, VOID_AFTER = 0.25, 8
LEFTOVER_LABELS = ("markitect.run", "markitect.evaluation")
KEY_SHAPES = [re.compile(rb"sk-ant-[A-Za-z0-9_-]{20,}"), re.compile(rb"eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}")]


class Stop(Exception):
    """A stop rule of the protocol: the series stops before its next launch, with this reason in WORK/STOP."""


def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat(timespec="seconds")


def ours(keys):
    """Whether a container's labels mark it as the suite's or a Run's own."""
    return any(str(k).startswith("markitect.") for k in keys)


# --- Plans --------------------------------------------------------------------------------------------------------

def public_series(name, tools=HERE.parent):
    """A series of this repository: its steps, its fixed starts (every step after the first) and its planted tasks."""
    import series
    steps = series.steps(name)
    plants = {}
    for path in sorted((tools / "evaluation-judges" / "plants").glob("*.json")):
        record = json.loads(path.read_text(encoding="utf-8"))
        if record.get("series") == name:
            plants[path.stem] = {"step": record["step"], "kind": record["kind"]}
    return {"role": "public", "deadlineSeconds": DEADLINES[name], "steps": steps, "fixed": steps[1:], "plants": plants}


def prepared_series(folder):
    """A series whose inputs another session prepared (the new project's author): FOLDER/series.json names its steps,
    fixed starts, plants with their step and kind, its deadline, and where its judge folders lie when it is judged."""
    spec = json.loads((pathlib.Path(folder) / "series.json").read_text(encoding="utf-8"))
    return spec["name"], {"role": "confirmatory", "deadlineSeconds": spec["deadlineSeconds"], "steps": spec["steps"],
                          "fixed": spec["fixed"], "plants": spec["plants"], "prepared": str(folder), "judges": spec["judges"],
                          "inputEffort": spec.get("inputEffort") or {}}


def pair(unit, first):
    second = "K" if first == "M" else "M"
    return [{**unit, "id": f"{unit['id']}-{arm}", "arm": arm} for arm in (first, second)]


def counted_units(series):
    """The counted order: by repetition, every fixed start and plant whose count reaches it, both arms, the first arm
    alternating per task and per repetition, the task order rotated per repetition; then the chains, by repetition,
    the first arm alternating per chain."""
    tasks = []
    for name, s in series.items():
        reps = REPETITIONS[s["role"]]
        tasks += [(name, "fixed", step, reps["fixed"]) for step in s["fixed"]]
        tasks += [(name, "plant", plant, reps["plant"]) for plant in s["plants"]]
    top = max(count for *_, count in tasks)
    units = []
    for r in range(1, top + 1):
        due = [(i, t) for i, t in enumerate(tasks) if t[3] >= r]
        shift = (r - 1) * len(due) // top
        for i, (name, kind, what, _) in due[shift:] + due[:shift]:
            key = "step" if kind == "fixed" else "plant"
            units += pair({"id": f"{name}-{kind}-{what}-r{r}", "kind": kind, "series": name, key: what, "repetition": r},
                          "M" if (i + r) % 2 == 1 else "K")
    chains = [(name, r) for r in range(1, max(REPETITIONS[s["role"]]["chain"] for s in series.values()) + 1)
              for name, s in series.items() if REPETITIONS[s["role"]]["chain"] >= r]
    for j, (name, r) in enumerate(chains):
        units += pair({"id": f"{name}-chain-r{r}", "kind": "chain", "series": name, "steps": series[name]["steps"], "repetition": r},
                      "M" if j % 2 == 0 else "K")
    return units


def gate_units(series):
    """The stability gate: three Library fixed starts in a row and one Makerspace chain per arm, alternating."""
    name, steps = GATE_FIXED
    units = []
    for i, step in enumerate(steps):
        units += pair({"id": f"{name}-fixed-{step}-gate", "kind": "fixed", "series": name, "step": step, "repetition": 1}, "MK"[i % 2])
    units += pair({"id": f"{GATE_CHAIN}-chain-gate", "kind": "chain", "series": GATE_CHAIN, "steps": series[GATE_CHAIN]["steps"],
                   "repetition": 1}, "K")
    return units


def launches(units):
    return sum(len(u["steps"]) if u["kind"] == "chain" else 1 for u in units)


def plan(protocol, machine, prepared=()):
    series = {name: public_series(name) for name in ("library", "makerspace")}
    if protocol == "counted":
        for folder in prepared:
            name, spec = prepared_series(folder)
            series[name] = spec
        units = counted_units(series)
    elif protocol == "gate":
        if prepared:
            raise SystemExit("the stability gate runs the Library and the Makerspace only")
        units = gate_units(series)
    else:
        raise SystemExit(f"no protocol {protocol!r}")
    planned = launches(units)
    return {**machine, "protocol": protocol, "series": series, "relaunches": 2, "launchBound": planned * 3 // 2,
            "minimumDifference": 0.05, "reviews": {"repetition": 1, "chains": True} if protocol == "counted" else {},
            "units": units}


def summary(units):
    out = {"launches": launches(units), "byArm": {}, "bySeries": {}}
    for u in units:
        n = len(u["steps"]) if u["kind"] == "chain" else 1
        out["byArm"][u["arm"]] = out["byArm"].get(u["arm"], 0) + n
        key = f"{u['series']} {u['kind']}"
        out["bySeries"][key] = out["bySeries"].get(key, 0) + n
    return out


# --- The engine and the login ---------------------------------------------------------------------------------------

def foreign_running(ps_output):
    """Names of running containers without a `markitect.*` label, from `docker ps --format '{{.Names}}\\t{{.Labels}}'`."""
    found = []
    for line in ps_output.splitlines():
        name, _, labels = line.partition("\t")
        if name and not ours(x.split("=", 1)[0] for x in labels.split(",") if x):
            found.append(name)
    return found


def foreign_escapes(run):
    """Whether a run has escapes and every one was a container without a `markitect.*` label, as the run's engine events
    show its start: started by other work on the machine, not by the arm. An escape the events do not show is not
    taken as foreign."""
    record = classify.load(pathlib.Path(run) / "suite.json") or {}
    escapes = (record.get("sandbox") or {}).get("escapes") or []
    seen = []
    events = pathlib.Path(run) / "engine-events.jsonl"
    for line in events.read_text(encoding="utf-8", errors="replace").splitlines() if events.is_file() else []:
        try:
            event = json.loads(line)
        except ValueError:
            continue
        actor = event.get("Actor") if isinstance(event, dict) else None
        attributes = actor.get("Attributes") if isinstance(actor, dict) else None
        if event.get("Type") == "container" and event.get("Action") == "start" and isinstance(attributes, dict):
            seen.append((str(actor.get("ID") or ""), str(attributes.get("name") or ""), attributes))

    def started(escape):
        name, cid = str(escape.get("name") or ""), str(escape.get("id") or "")
        return [a for i, n, a in seen if (name and n == name) or (cid and i.startswith(cid))]

    return bool(escapes) and all(started(e) and not any(ours(a) for a in started(e)) for e in escapes)


class Guard:
    """No launch while a container without a `markitect.*` label runs, or one started in the last ten minutes, as a
    `docker events` stream kept open for the whole series sees it: the engine keeps only its last 256 events, fewer
    than most native runs write. The ten minutes count from when the stream began, again after it broke."""

    def __init__(self, engine, clock=time.time):
        self.engine, self.clock = engine, clock
        self.since, self.last, self.lock = None, None, threading.Lock()

    def start(self):
        threading.Thread(target=self.watch, daemon=True).start()
        return self

    def watch(self):
        while True:
            p = None
            try:
                p = subprocess.Popen([*self.engine, "events", "--filter", "type=container", "--filter", "event=start", "--format",
                                      "{{json .Actor.Attributes}}"], stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
                                     stdin=subprocess.DEVNULL, encoding="utf-8", errors="replace")
                with self.lock:
                    self.since = self.clock()
                for line in p.stdout:
                    self.seen(line)
            except OSError:
                pass
            finally:
                if p:
                    p.terminate()
                    p.wait()
            with self.lock:
                self.since = None
            time.sleep(10)

    def seen(self, line):
        try:
            attributes = json.loads(line)
        except ValueError:
            return
        if isinstance(attributes, dict) and not ours(attributes):
            with self.lock:
                self.last = (self.clock(), str(attributes.get("name", "?")))

    def running(self):
        """Running containers without a `markitect.*` label; None when the engine does not answer."""
        try:
            p = subprocess.run([*self.engine, "ps", "--format", "{{.Names}}\t{{.Labels}}"], capture_output=True, encoding="utf-8",
                               errors="replace", timeout=60, stdin=subprocess.DEVNULL)
        except (OSError, subprocess.TimeoutExpired):
            return None
        return foreign_running(p.stdout) if p.returncode == 0 else None

    def reasons(self):
        """Why a launch would not be on a quiet engine now; empty when it would."""
        running = self.running()
        out = ["the engine does not answer"] if running is None else [f"{name} (running)" for name in running]
        with self.lock:
            if self.since is None or self.clock() - self.since < QUIET_SECONDS:
                out.append("the engine not yet watched for ten minutes")
            elif self.last and self.clock() - self.last[0] < QUIET_SECONDS:
                out.append(f"{self.last[1]} (started)")
        return out


class Login:
    """The Claude account login of the series, never an API key. A launch needs it to outlive its deadline by twenty
    minutes. When it cannot, the login is refreshed first: its access token is marked expired, with no token changed,
    and `claude -p` refreshes it as it does near expiry. That happens only while no one holds the engine lock, since a
    run in progress uses a copy of the access token."""

    def __init__(self, path, lock, model, log, claude=None, sleep=time.sleep, clock=time.time):
        self.path, self.lock, self.log, self.sleep, self.clock = pathlib.Path(path), pathlib.Path(lock), log, sleep, clock
        provider, _, self.model = model.partition(":")
        if provider != "claude":
            raise SystemExit(f"the series runner refreshes only a Claude login, not {provider!r}")
        self.claude = claude or [shutil.which("claude") or "claude"]

    def oauth(self):
        return json.loads(self.path.read_text(encoding="utf-8"))["claudeAiOauth"]

    def left(self):
        return self.oauth()["expiresAt"] / 1000 - self.clock()

    def ensure(self, deadline):
        need = deadline + LOGIN_MARGIN
        if self.left() >= need:
            return
        renewable = self.oauth().get("refreshTokenExpiresAt")
        if renewable and renewable / 1000 - self.clock() < need:
            raise Stop("the login's refresh token expires before the next launch would end: log in again")
        waited = False
        while self.lock.exists():
            if not waited:
                self.log(event="login-refresh-waits", why="the engine lock is held; that run uses the access token")
                waited = True
            self.sleep(60)
        self.refresh()
        if self.left() < need:
            raise Stop(f"the login could not be refreshed: {round(self.left())} s left, {need} s needed")

    def expire(self):
        """Set the login's expiry to now, leaving every token as it is."""
        data = json.loads(self.path.read_text(encoding="utf-8"))
        data["claudeAiOauth"]["expiresAt"] = int(self.clock() * 1000)
        temporary = self.path.with_name(self.path.name + ".series-runner")
        try:
            temporary.write_text(json.dumps(data), encoding="utf-8")
            for attempt in range(5):
                try:
                    os.replace(temporary, self.path)
                    return
                except PermissionError:  # another process has the file open on Windows: try again
                    self.sleep(1)
            raise Stop("the login could not be marked for refresh: its file stays in use")
        finally:
            if temporary.exists():
                temporary.unlink()

    def refresh(self):
        before = round(self.left())
        self.expire()
        env = {k: v for k, v in os.environ.items() if not k.upper().startswith(("ANTHROPIC_", "CLAUDE_", "CLAUDECODE"))}
        with tempfile.TemporaryDirectory() as tmp:
            try:
                p = subprocess.run([*self.claude, "-p", "Reply with the word OK.", "--model", self.model, "--strict-mcp-config",
                                    "--disable-slash-commands", "--no-session-persistence"], cwd=tmp, env=env, capture_output=True,
                                   text=True, timeout=300, stdin=subprocess.DEVNULL)
                code = p.returncode
            except (OSError, subprocess.TimeoutExpired) as e:
                code = type(e).__name__
        self.log(event="login-refresh", exit=code, secondsLeftBefore=before, secondsLeft=round(self.left()))


def quota(run_dir):
    """The highest five-hour and seven-day use the agents saw, and whether any request was refused."""
    worst = {"fiveHour": 0.0, "sevenDay": 0.0, "refused": False, "fiveHourReset": None}
    for f in glob.glob(str(pathlib.Path(run_dir) / "**" / "*.jsonl"), recursive=True):
        with open(f, encoding="utf-8", errors="replace") as lines:
            found = [line for line in lines if '"rate_limit_event"' in line]
        for line in found:
            try:
                info = json.loads(line)["rate_limit_info"]
            except (ValueError, KeyError, TypeError):
                continue
            windows = info.get("unifiedWindows") or {}
            five, seven = windows.get("five_hour") or {}, windows.get("seven_day") or {}
            if (five.get("utilization") or 0) >= worst["fiveHour"]:
                worst["fiveHour"], worst["fiveHourReset"] = five.get("utilization") or 0, five.get("resetsAt")
            worst["sevenDay"] = max(worst["sevenDay"], seven.get("utilization") or 0)
            worst["refused"] = worst["refused"] or info.get("status") not in (None, "allowed", "allowed_warning")
    return worst


def login_secrets(path):
    """Every token of a login file, at any depth: the values a scan of the evidence looks for."""
    found = []

    def walk(value, key=""):
        if isinstance(value, dict):
            for k, v in value.items():
                walk(v, k)
        elif isinstance(value, str) and "token" in key.lower() and len(value) > 16:
            found.append(value.encode())
    walk(json.loads(pathlib.Path(path).read_text(encoding="utf-8")))
    return found


def files_with_secrets(root, secrets, skip=()):
    hits, scanned = [], 0
    for f in pathlib.Path(root).rglob("*"):
        rel = f.relative_to(root)
        if rel.parts[:1] in [(s,) for s in skip] or f.is_symlink() or not f.is_file():
            continue
        scanned += 1
        data = f.read_bytes()
        if any(s in data for s in secrets) or any(x.search(data) for x in KEY_SHAPES):
            hits.append(rel.as_posix())
    return hits, scanned


# --- The runner -----------------------------------------------------------------------------------------------------

class Runner:
    def __init__(self, config, ev=HERE, suite=None, guard=None, login=None, sleep=time.sleep):
        self.c = config
        self.work = pathlib.Path(config["work"])
        self.ev = pathlib.Path(ev)
        self.suite_py = pathlib.Path(suite) if suite else self.ev / "suite.py"
        self.py = sys.executable
        self.engine = config.get("engine") or [config.get("docker") or shutil.which("docker") or "docker"]
        self.sleep = sleep
        self.guard = guard
        self.login = login or Login(config["login"], config["lock"], config["model"], self.log, sleep=sleep)
        ledger = self.work / "ledger.json"
        self.ledger = json.loads(ledger.read_text(encoding="utf-8")) if ledger.exists() else {
            "format": 1, "series": f"{config['protocol']}-{config['rev'][:8]}", "signatures": "signatures.json", "tasks": {},
            "launches": [], "chains": [], "attempts": [], "done": [], "progress": {}, "pending": [], "waits": [], "windows": [],
            "contrasts": [{"variant": "M", "baseline": "K", "endpoint": "casesPassedShare", "minimumDifference": config["minimumDifference"]},
                          {"variant": "M", "baseline": "K", "endpoint": "defectsLeft", "minimumDifference": 1}]}

    # --- bookkeeping ---
    def log(self, **event):
        event["time"] = now()
        self.work.mkdir(parents=True, exist_ok=True)
        with open(self.work / "series.jsonl", "a", encoding="utf-8") as f:
            f.write(json.dumps(event) + "\n")

    def save(self):
        (self.work / "ledger.json").write_text(json.dumps(self.ledger, indent=2) + "\n", encoding="utf-8")

    def rel(self, path):
        return pathlib.Path(path).relative_to(self.work).as_posix()

    def sh(self, name, argv, cwd=None):
        out = self.work / "logs" / (name + ".out")
        out.parent.mkdir(parents=True, exist_ok=True)
        with open(out, "wb") as so, open(out.with_suffix(".err"), "wb") as se:
            code = subprocess.run([str(a) for a in argv], cwd=cwd, stdout=so, stderr=se, stdin=subprocess.DEVNULL).returncode
        return code, out.with_suffix(".err").read_text(encoding="utf-8", errors="replace")

    def must(self, name, argv, cwd=None):
        code, err = self.sh(name, argv, cwd)
        if code != 0:
            raise RuntimeError(f"{name} exited {code}: {err[-1500:]}")

    def suite(self, name, args):
        """One suite command; while another holder has the engine lock it waits, uncounted, and tries again."""
        while True:
            code, err = self.sh(name, [self.py, "-I", "-B", self.suite_py, *args])
            if code == 2 and "the engine is busy" in err:
                self.log(event="lock-busy", name=name)
                self.sleep(120)
                continue
            return code, err

    # --- inputs ---
    def series(self, name):
        return self.c["series"][name]

    def prepared(self, name):
        s = self.series(name)
        return pathlib.Path(s["prepared"]) if s.get("prepared") else self.work / "prep" / name

    def judges(self, name):
        s = self.series(name)
        return pathlib.Path(s["judges"]) if s.get("judges") else self.prepared(name) / "judges"

    def hidden(self):
        """Every place hidden material could lie: the protocol's, and the judge folders of a series judged later."""
        return [*self.c.get("hidden", []), *(s["judges"] for s in self.c["series"].values() if s.get("judges"))]

    def task_name(self, unit):
        if unit["kind"] == "chain":
            return f"{unit['series']}-chain"
        if unit["kind"] == "plant":
            return "plant-" + unit["plant"]
        return f"{unit['series']}-fixed-{unit['step']}"

    def step_of(self, unit):
        return self.series(unit["series"])["plants"][unit["plant"]]["step"] if unit["kind"] == "plant" else unit["step"]

    def task_dir(self, unit):
        what = f"plant-{unit['plant']}" if unit["kind"] == "plant" else f"fixed-{unit['step']}"
        return self.prepared(unit["series"]) / "tasks" / what

    def judge_dir(self, unit):
        return self.judges(unit["series"]) / (f"plant-{unit['plant']}" if unit["kind"] == "plant" else "series")

    def prep(self):
        """The public series' inputs from the frozen revision: steps, the series judge, fixed reference starts with the
        registry pages their owner wrote, planted steps and their judges, and a task per fixed start and plant."""
        for name, s in self.c["series"].items():
            if s.get("prepared"):
                for unit in self.c["units"]:
                    if unit["series"] == name and not (self.task_dir(unit) if unit["kind"] != "chain" else self.prepared(name) / "steps").is_dir():
                        raise SystemExit(f"the prepared series {name} lacks {self.task_dir(unit)}")
                continue
            prep, src = self.prepared(name), self.ev.parent / "evaluation-judges"
            if not (prep / "steps").exists():
                self.must(f"prep-steps-{name}", [self.py, "-I", "-B", self.ev / "series.py", "steps", "--series", name, "--out", prep / "steps"])
            if not (prep / "judges" / "series").exists():  # from the frozen tree, which is the revision's
                self.must(f"prep-judge-{name}", [self.py, "-I", "-B", self.ev / "judges.py", "export", "--source", src / name,
                                                 "--out", prep / "judges" / "series"])
            tasks = {("fixed", step): step for step in s["fixed"]} | {("plant", p): v["step"] for p, v in s["plants"].items()}
            for (kind, what), step in tasks.items():
                before = s["steps"][s["steps"].index(step) - 1]
                start, base = prep / "starts" / f"{before}.bundle", prep / "bases" / before
                if not start.exists():
                    start.parent.mkdir(parents=True, exist_ok=True)
                    self.must(f"prep-start-{name}-{before}", [self.py, "-I", "-B", self.ev / "judges.py", "start", "--source", src / name,
                                                              "--step", before, "--out", start])
                if not base.exists():
                    self.must(f"prep-base-{name}-{before}", [self.py, "-I", "-B", self.ev / "series.py", "base", "--series", name,
                                                             "--step", before, "--out", base])
                source = prep / "steps" / step
                if kind == "plant":
                    source = prep / "plant-steps" / what
                    if not source.exists():
                        self.must(f"prep-plant-{what}", [self.py, "-I", "-B", self.ev / "judges.py", "plant", "--plant", src / "plants" / f"{what}.json",
                                                         "--step-out", source, "--judge", prep / "judges" / "series",
                                                         "--judge-out", prep / "judges" / f"plant-{what}"])
                task = prep / "tasks" / f"{kind}-{what}"
                if not task.exists():
                    self.must(f"prep-task-{name}-{kind}-{what}", [self.py, "-I", "-B", self.suite_py, "task", "--task-md", source / "task.md",
                                                                   "--model", source / "model", "--start", start, "--base-model", base,
                                                                   "--deadline-seconds", s["deadlineSeconds"], "--out", task])
        for unit in self.c["units"]:
            entry = self.ledger["tasks"].setdefault(self.task_name(unit), {"inputEffort": None, "note": "task.md is rendered from the model"})
            s = self.series(unit["series"])
            if s["role"] == "confirmatory":
                entry.update(inputEffort=s.get("inputEffort", {}).get(self.task_name(unit)), note="task.md written by the author")
            if unit["kind"] == "plant":
                entry["planted"] = s["plants"][unit["plant"]]["kind"]
        (self.work / "signatures.json").write_text(json.dumps(SIGNATURES, indent=2) + "\n", encoding="utf-8")
        self.save()

    # --- conditions before a launch ---
    def arm_args(self, unit):
        if unit["arm"] == "M":
            hidden = [x for h in self.hidden() for x in ("--hidden-absent", h)]
            return (["--native", self.c["sandboxImage"], *hidden, "--lock", self.c["lock"], "--engine", self.engine[0]],
                    [self.py, "-I", "-B", self.ev / "markitect_arm.py", "--markitect", pathlib.Path(self.c["bin"]) / "markitect.exe",
                     "--remote-binary", pathlib.Path(self.c["bin"]) / "markitect-remote", "--manager-image", self.c["managerImage"]])
        return (["--sandbox", self.c["sandboxImage"], "--variant-code", self.ev, "--lock", self.c["lock"], "--engine", self.engine[0]],
                ["python3", "-I", "-B", "/variant/claude_code.py", "--effort", "high"])

    def quiet(self):
        """Wait until the guard finds the engine quiet; the wait goes into the log and the ledger's `waits`."""
        noted = None
        while True:
            found = self.guard.reasons()
            if not found:
                if noted:
                    noted["ended"] = now()
                    self.log(event="engine-quiet-again")
                    self.save()
                return
            if noted is None:
                noted = {"began": now(), "containers": found,
                         "why": "a container without a markitect.* label runs or started in the last ten minutes, or the engine has not "
                                "been watched that long: a launch now could be classed invalid; waiting"}
                self.ledger["waits"].append(noted)
                self.save()
                self.log(event="engine-not-quiet", containers=found)
            self.sleep(120)

    def ready(self, deadline):
        """Before a launch: no STOP, the bound, no hidden material, a quiet engine, a login that covers the launch, and
        the guard once more, since a login refresh can outlast its answer."""
        if (self.work / "STOP").exists():
            return False
        if len(self.ledger["attempts"]) >= self.c["launchBound"]:
            raise Stop("the launch bound is reached")
        present = [h for h in self.hidden() if pathlib.Path(h).exists()]
        if present:
            raise Stop("hidden material is on the machine: " + ", ".join(present))
        self.quiet()
        self.login.ensure(deadline)
        self.quiet()
        return True

    # --- launches ---
    def attempts_of(self, unit_id, step):
        return [a for a in self.ledger["attempts"] if a["unit"] == unit_id and a["step"] == step]

    def resolved(self, done):
        """Where the recorded launches of a task or chain step leave it: None (none yet), "again" or "done". A launch
        that stopped the series is done unless it was void with relaunches left: after the stop is cleared, a series
        goes on with the next unit rather than repeat a launch that counted."""
        if not done:
            return None
        last = done[-1]
        if last["answer"] == "again" or (last["answer"] == "stop" and last["class"] == "void" and len(done) <= self.c["relaunches"]):
            return "again"
        return "done"

    def settle(self, unit, verdict, number, foreign_used, run):
        """The protocol's answer to one classed launch, the `number`th of its task or step: "done", "again" (relaunch)
        or a stop."""
        arm = [a for a in self.ledger["attempts"] if a["arm"] == unit["arm"]]
        voids = sum(a["class"] == "void" for a in arm)
        if verdict.get("stopsSeries"):
            raise Stop(f"the classifier stops the series: {verdict.get('evidence')}")
        if len(arm) >= VOID_AFTER and voids / len(arm) > VOID_SHARE:
            raise Stop(f"arm {unit['arm']}: {voids} of {len(arm)} launches void, above one in four")
        if verdict["class"] == "void":
            return "again" if number <= self.c["relaunches"] else "done"
        if verdict["class"] == "invalid":
            if not foreign_used and foreign_escapes(run):
                return "again"
            raise Stop(f"an invalid launch, {self.rel(run)}: {verdict.get('evidence')}")
        return "done"

    def launch(self, unit, out, run, args, step, judge=None, progress=None):
        """One launch into `out`, whose run folder is `run`: the suite command (none for a launch the runner did not live
        to record), its class with the arm's signatures, its records and judgment, the protocol's answer, and then the
        quota and the evidence. A chain's `progress` keeps a step that counts."""
        code, err = self.suite(f"{unit['id']}-{out.name}", args) if args else (None, "the runner stopped during this launch")
        record = classify.load(run / "suite.json")
        verdict = classify.classify(run, SIGNATURES, unit["arm"]) if record else {
            "class": "void", "kind": "suite", "evidence": err[-500:], "stopsSeries": False}
        done = self.attempts_of(unit["id"], step)
        attempt = {"unit": unit["id"], "step": step, "arm": unit["arm"], "run": self.rel(out), "class": verdict["class"],
                   "kind": verdict.get("kind"), "foreign": verdict["class"] == "invalid" and foreign_escapes(run), "exit": code,
                   "time": now()}
        self.ledger["attempts"].append(attempt)
        self.log(event="ran", unit=unit["id"], step=step, run=self.rel(out), exit=code, **verdict)
        if record:
            entry = {"id": f"{unit['id']}-{out.name}", "variant": unit["arm"], "task": self.task_name(unit),
                     "repetition": unit.get("repetition", 1), "step": step, "run": self.rel(run)}
            self.ledger["launches"].append(entry)
            if judge and verdict["class"] == "result" and record.get("delivered"):
                self.judge_launch(entry, judge)
        stop = None
        try:
            attempt["answer"] = self.settle(unit, verdict, len(done) + 1, any(a["foreign"] for a in done), run)
        except Stop as e:
            attempt["answer"], stop = "stop", e
        if progress is not None and verdict["class"] != "void" and attempt["answer"] != "again":
            delivered = (record or {}).get("delivered")
            progress["kept"].append([step, self.rel(out)])
            progress["start"] = self.rel(run / "variant" / "result.bundle" if delivered else out / "step-01" / "task" / "start.bundle")
        self.save()
        if record:
            self.after(run)
        if stop:
            raise stop

    def after(self, run):
        """The quota the launch's agents saw and a scan of its evidence for the login."""
        q = quota(run)
        self.log(event="quota", run=self.rel(run), **q)
        hits, _ = files_with_secrets(run, login_secrets(self.c["login"]))
        if hits:
            raise Stop(f"a credential in the evidence of {self.rel(run)}: {hits}")
        if q["refused"] or q["sevenDay"] >= QUOTA_STOP:
            raise Stop(f"quota: {json.dumps(q)}")
        if q["fiveHour"] >= QUOTA_WAIT and q["fiveHourReset"]:
            wait = max(0, q["fiveHourReset"] - time.time()) + 60
            self.log(event="quota-wait", seconds=round(wait))
            self.sleep(wait)

    def judge_launch(self, entry, judge):
        """A delivered result's judgment, or a pending one while its judge is off the machine."""
        if not judge.exists():
            self.ledger["pending"].append({"kind": "launch", "id": entry["id"], "judge": str(judge)})
            return
        out = self.work / "judgments" / entry["id"]
        self.suite(f"judge-{entry['id']}", ["judge", "--run", self.work / entry["run"], "--judge", judge, "--step", entry["step"],
                                            "--out", out, "--lock", self.c["lock"], "--engine", self.engine[0]])
        if (out / "judgment.json").exists():
            entry["judgment"] = self.rel(out)

    def run_task(self, unit):
        """A fixed start or plant, launched until a launch counts, within the relaunch limits. False when the series
        stops before it is done."""
        flags, variant = self.arm_args(unit)
        base, step = self.work / "runs" / unit["id"], self.step_of(unit)
        while True:
            done = self.attempts_of(unit["id"], step)
            if self.resolved(done) == "done":
                return True
            run = base / f"a{len(done) + 1}"
            if run.exists():  # launched before a restart and never recorded: recorded as it ended
                self.launch(unit, run, run, None, step, self.judge_dir(unit))
                continue
            if not self.ready(self.series(unit["series"])["deadlineSeconds"]):
                return False
            self.launch(unit, run, run, ["run", "--task", self.task_dir(unit), "--model", self.c["model"], "--out", run, *flags,
                                         "--", *variant], step, self.judge_dir(unit))

    def run_chain(self, unit):
        """A chain one step at a time: each step a one-step `suite.py chain` from the step before's result (or its start
        when it delivered none) with the unit's one state folder, which every launch of a step starts from as it was
        before the step. A step that stays void ends the chain. The steps that counted are then judged as one chain."""
        flags, variant = self.arm_args(unit)
        deadline = self.series(unit["series"])["deadlineSeconds"]
        base = self.work / "runs" / unit["id"]
        state = base / "state"
        state.mkdir(parents=True, exist_ok=True)
        progress = self.ledger["progress"].setdefault(unit["id"], {"kept": [], "start": None})
        for step in unit["steps"]:
            snapshot = base / f"state-before-{step}"
            while not any(kept == step for kept, _ in progress["kept"]):
                if not snapshot.exists():
                    shutil.copytree(state, snapshot)
                done = self.attempts_of(unit["id"], step)
                if self.resolved(done) == "done":
                    self.log(event="chain-stopped", unit=unit["id"], at=step)
                    self.finish_chain(unit)
                    return True
                out = base / f"{step}-a{len(done) + 1}"
                run = out / "step-01" / "run"
                if out.exists():
                    self.launch(unit, out, run, None, step, progress=progress)
                    continue
                if not self.ready(deadline):
                    return False
                shutil.rmtree(state)
                shutil.copytree(snapshot, state)
                start = ["--start", self.work / progress["start"]] if progress["start"] else []
                self.launch(unit, out, run, ["chain", "--step", self.prepared(unit["series"]) / "steps" / step, *start, "--state", state,
                                             "--deadline-seconds", deadline, "--model", self.c["model"], "--out", out, *flags, "--", *variant],
                            step, progress=progress)
        self.finish_chain(unit)
        return True

    def finish_chain(self, unit):
        """The counted steps as one chain folder, judged by the series judge now, or later while it is off the machine."""
        kept = self.ledger["progress"][unit["id"]]["kept"]
        if not kept or any(c["id"] == unit["id"] for c in self.ledger["chains"]) or any(
                p.get("chain", {}).get("id") == unit["id"] for p in self.ledger["pending"]):
            return
        whole = self.work / "runs" / f"{unit['id']}-chain"
        rows = []
        for i, (step, out) in enumerate(kept, 1):
            shutil.copytree(self.work / out / "step-01", whole / f"step-{i:02d}", dirs_exist_ok=True)
            row = json.loads((self.work / out / "chain.json").read_text(encoding="utf-8"))["steps"][0]
            rows.append({**row, "step": i})
        (whole / "chain.json").write_text(json.dumps({"format": 1, "model": self.c["model"], "steps": rows, "stopped": None,
                                                      "assembledFrom": [o for _, o in kept]}, indent=2) + "\n", encoding="utf-8")
        chain = {"id": unit["id"], "variant": unit["arm"], "task": self.task_name(unit), "repetition": unit.get("repetition", 1)}
        if (self.judges(unit["series"]) / "series").exists():
            self.judge_chain(chain, unit["series"], kept)
        else:
            self.ledger["pending"].append({"kind": "chain", "chain": chain, "series": unit["series"], "kept": kept})
        self.save()

    def judge_chain(self, chain, series, kept):
        whole, judged = self.work / "runs" / f"{chain['id']}-chain", self.work / "judgments" / f"{chain['id']}-chain"
        self.suite(f"judge-chain-{chain['id']}", ["judge-chain", "--chain", whole, "--judge", self.judges(series) / "series",
                                                  "--steps", ",".join(s for s, _ in kept), "--out", judged, "--lock", self.c["lock"],
                                                  "--engine", self.engine[0]])
        for i, (_, out) in enumerate(kept, 1):
            for entry in self.ledger["launches"]:
                if entry["run"] == f"{out}/step-01/run" and (judged / f"step-{i:02d}" / "judgment.json").exists():
                    entry["judgment"] = self.rel(judged / f"step-{i:02d}")
        if (judged / "chain-judgment.json").exists():
            self.ledger["chains"].append({**chain, "judgment": self.rel(judged)})
            return True
        return False

    # --- the end of a window and of the series ---
    def judge_pending(self):
        """Every judgment that waited for its judge folder, once that folder is on the machine."""
        left = []
        for item in self.ledger["pending"]:
            if item["kind"] == "launch":
                entry = next(e for e in self.ledger["launches"] if e["id"] == item["id"])
                if pathlib.Path(item["judge"]).exists():
                    self.judge_launch(entry, pathlib.Path(item["judge"]))
                judged = bool(entry.get("judgment"))
            else:
                judged = (self.judges(item["series"]) / "series").exists() and self.judge_chain(item["chain"], item["series"], item["kept"])
            if not judged:
                left.append(item)
        self.ledger["pending"] = left
        self.save()

    def reviews(self):
        """The code review of every judged result of the protocol's review repetition (fixed starts and every chain's last
        judged step), blinded, in the sandbox. A review needs the judge folder's rubric, so it waits for it like judging."""
        wanted = self.c.get("reviews") or {}
        if not wanted:
            return
        picks = [e for e in self.ledger["launches"] if e.get("judgment") and "-fixed-" in e["task"] and e["repetition"] == wanted.get("repetition")]
        if wanted.get("chains"):
            for chain in self.ledger["chains"]:
                steps = [e for e in self.ledger["launches"] if e["task"] == chain["task"] and e["variant"] == chain["variant"]
                         and e["repetition"] == chain["repetition"] and e.get("judgment")]
                picks += steps[-1:]
        for entry in picks:
            series = next(u["series"] for u in self.c["units"] if self.task_name(u) == entry["task"])
            judge = self.judges(series) / "series"
            out = self.work / "reviews" / entry["id"]
            if entry.get("review") or not judge.exists():
                continue
            self.suite(f"review-{entry['id']}", ["review", "--run", self.work / entry["run"], "--judge", judge, "--out", out,
                                                 "--sandbox", self.c["sandboxImage"], "--variant-code", self.ev, "--lock", self.c["lock"],
                                                 "--engine", self.engine[0]])
            if (out / "review.json").exists():
                entry["review"] = self.rel(out)
                self.save()

    def report(self):
        out = self.work / "report"
        if out.exists():
            shutil.rmtree(out)
        self.suite("report", ["report", "--ledger", self.work / "ledger.json", "--out", out])

    def final_check(self):
        """Every file of the evidence scanned for the login's tokens and key shapes, and what is left by label."""
        hits, scanned = files_with_secrets(self.work, login_secrets(self.c["login"]), skip=("src",))
        left = {}
        for kind, args in (("containers", ["ps", "-a", "-q"]), ("volumes", ["volume", "ls", "-q"]), ("networks", ["network", "ls", "-q"])):
            left[kind] = [x for label in LEFTOVER_LABELS
                          for x in subprocess.run([*self.engine, *args, "--filter", f"label={label}"], capture_output=True, text=True,
                                                  stdin=subprocess.DEVNULL).stdout.split()]
        self.log(event="final-check", scanned=scanned, secretHits=hits, leftovers=left)
        return {"scanned": scanned, "secretHits": hits, "leftovers": left}

    def wrap_up(self, complete):
        """After every window: the report and the final check; after the last unit also the reviews and done.json."""
        for step in ((self.judge_pending, self.reviews) if complete else ()) + (self.report,):
            try:
                step()
            except Exception as e:  # noqa: BLE001 - recorded; the next one still runs
                self.log(event="wrap-up-error", step=step.__name__, error=f"{type(e).__name__}: {e}")
        check = self.final_check()
        if complete:
            (self.work / "done.json").write_text(json.dumps({"ended": now(), "launches": len(self.ledger["attempts"]),
                                                             "pending": len(self.ledger["pending"]), **check}, indent=2) + "\n", encoding="utf-8")

    def run(self, prep_only=False):
        if (self.work / "done.json").exists():
            raise SystemExit("the series is done")
        self.work.mkdir(parents=True, exist_ok=True)
        manifest = {p.name: hashlib.sha256(p.read_bytes()).hexdigest()[:16] for p in sorted(self.ev.glob("*.py"))}
        self.log(event="begin", rev=self.c["rev"], protocol=self.c["protocol"], units=len(self.c["units"]), done=len(self.ledger["done"]),
                 tools=manifest)
        self.prep()
        if prep_only:
            return
        self.guard = self.guard or Guard(self.engine).start()
        window = {"began": now(), "launchesBefore": len(self.ledger["attempts"])}
        self.ledger["windows"].append(window)
        complete = True
        try:
            for unit in self.c["units"]:
                if unit["id"] in self.ledger["done"]:
                    continue
                if not (self.run_chain(unit) if unit["kind"] == "chain" else self.run_task(unit)):
                    complete = False
                    break
                self.ledger["done"].append(unit["id"])
                self.save()
        except Exception as e:  # noqa: BLE001 - a stop rule, or the runner's own failure: either way a person looks first
            complete = False
            reason = str(e) if isinstance(e, Stop) else f"the runner failed: {type(e).__name__}: {e}"
            (self.work / "STOP").write_text(reason + "\n", encoding="utf-8")
            self.log(event="stop", reason=reason)
        window.update(ended=now(), launches=len(self.ledger["attempts"]) - window["launchesBefore"])
        self.save()
        self.wrap_up(complete)
        self.log(event="end", complete=complete, launches=len(self.ledger["attempts"]))

    def judge(self):
        """Judge, review and report what waited for a judge folder that was off the machine during launches."""
        self.judge_pending()
        self.reviews()
        self.report()
        self.log(event="judged", pending=len(self.ledger["pending"]))


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    sub = p.add_subparsers(dest="command", required=True)
    pl = sub.add_parser("plan")
    pl.add_argument("--protocol", required=True, choices=("counted", "gate"))
    pl.add_argument("--machine", required=True)
    pl.add_argument("--prepared", action="append", default=[])
    pl.add_argument("--out", required=True)
    for name in ("units", "judge"):
        sub.add_parser(name).add_argument("config")
    r = sub.add_parser("run")
    r.add_argument("config")
    r.add_argument("--prep-only", action="store_true")
    a = p.parse_args(argv)
    if a.command == "plan":
        config = plan(a.protocol, json.loads(pathlib.Path(a.machine).read_text(encoding="utf-8")), [pathlib.Path(x) for x in a.prepared])
        pathlib.Path(a.out).write_text(json.dumps(config, indent=2) + "\n", encoding="utf-8")
        print(json.dumps({**summary(config["units"]), "launchBound": config["launchBound"]}, indent=2))
        return 0
    config = json.loads(pathlib.Path(a.config).read_text(encoding="utf-8"))
    if a.command == "units":
        print(json.dumps({**summary(config["units"]), "order": [u["id"] for u in config["units"]]}, indent=2))
        return 0
    if not HERE.is_relative_to(pathlib.Path(config["work"]).resolve() / "src"):
        raise SystemExit(f"run the runner from the frozen tools: git archive {config['rev']} tools into {config['work']}/src")
    runner = Runner(config)
    if a.command == "run":
        runner.run(prep_only=a.prep_only)
    else:
        runner.judge()
    return 0


if __name__ == "__main__":
    sys.exit(main())
