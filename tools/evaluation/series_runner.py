"""The series runner: a counted series or its stability gate, one launch at a time through suite.py, with its judging,
reviews, ledger and report.

  python -I -B series_runner.py plan --protocol counted|gate --machine MACHINE.json [--prepared SPEC.json ...] --out CONFIG.json
  python -I -B series_runner.py units CONFIG.json
  python -I -B series_runner.py run CONFIG.json [--prep-only]
  python -I -B series_runner.py judge CONFIG.json

`plan` writes a series configuration: MACHINE.json (`work`, `rev`, `bin`, `managerImage`, `sandboxImage`, `lock`,
`hidden`, `login`, `model`, optionally `docker`) with the protocol's units in their order and its contrasts, for the
Library, the Makerspace and each `--prepared` series. `units` prints the launch order and its counts and starts nothing.
`run` prepares the public series' inputs and runs every unit not yet done; started again after a stop, it goes on where
it stopped. `judge` judges and reviews what a series whose judge folders stay off the machine during launches left
pending, then writes the report again.

Run it from the frozen revision's tools (`git archive REV tools` into WORK/src) and start it detached (Win32_Process
Create on Windows, setsid nohup on Linux), without a shell's ANTHROPIC_* or CLAUDE_* variables. It writes
WORK/series.jsonl (one line per event), WORK/ledger.json, the reports and WORK/done.json (and WORK/gate.json for the
gate). It holds WORK/runner.lock while it runs, and it ends a window when WORK/STOP exists: before the next launch, or
during any wait.

The runner holds the shared engine lock only for a login refresh, between launches, never while a suite command runs:
suite.py takes the lock for each run, judgment and review, and a busy lock is waited for and not counted. Before each
launch it waits until the engine is quiet (Guard) and the login outlives the launch's deadline by twenty minutes
(Login). Each launch is classed with its arm's variant runtime signatures, as the report classes it. A void launch is
relaunched up to `relaunches` times; a launch invalid only by containers of other work (no `markitect.*` label, as the
run's own engine events show their start) is relaunched once; any other invalid launch stops the series. So do a
refused request or a seven-day use of 90 %, a credential in a run's evidence, a gateway refusal of a provider name,
hidden material on the machine, a void share above one in four for an arm after its first eight launches, a judge that
gives no valid judgment twice, a frozen file, binary or image that changed between windows, the launch bound, a login
that cannot be refreshed, and the runner's own failure; a five-hour use of 85 % waits for that window's reset. In the
stability gate, any launch that is not a result stops it.
"""
import argparse
import contextlib
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
# The protocol's contrasts: the primary one first, each read over the changes' mean differences.
CONTRASTS = [{"variant": "M", "baseline": "K", "endpoint": "casesPassedShare", "minimumDifference": 0.05, "blockBy": "change"},
             {"variant": "M", "baseline": "K", "endpoint": "casesFailed", "minimumDifference": 1, "blockBy": "change"},
             {"variant": "M", "baseline": "K", "endpoint": "defectsLeft", "minimumDifference": 1}]
QUIET_SECONDS = 600
LOGIN_MARGIN = 20 * 60
REVIEW_SECONDS = 3600  # a review and its confirmations, each session up to reviewing.DEADLINE
QUOTA_STOP, QUOTA_WAIT = 0.90, 0.85
VOID_SHARE, VOID_AFTER = 0.25, 8
LEFTOVER_LABELS = ("markitect.run", "markitect.evaluation")
KEY_SHAPES = [re.compile(rb"sk-ant-[A-Za-z0-9_-]{20,}"), re.compile(rb"eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}")]


class Stop(Exception):
    """A stop rule of the protocol: the series stops before its next launch, with this reason in WORK/STOP."""


class Halt(Exception):
    """WORK/STOP exists: the window ends, before the next launch or during a wait."""


def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat(timespec="seconds")


def ours(keys):
    """Whether a container's labels mark it as the suite's or a Run's own."""
    return any(str(k).startswith("markitect.") for k in keys)


def sha256(path):
    return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()


def tree_digest(root):
    """One SHA-256 over every file below root, by relative path and content."""
    h = hashlib.sha256()
    for path in sorted(p for p in pathlib.Path(root).rglob("*") if p.is_file()):
        h.update(path.relative_to(root).as_posix().encode() + b"\0" + sha256(path).encode() + b"\n")
    return h.hexdigest()


def valid(path):
    record = classify.load(path)
    return bool(record) and record.get("valid") is True


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


def prepared_series(spec):
    """A series whose author prepares its inputs, the new project's. The JSON file SPEC names its steps, fixed starts,
    plants with their step (not their kind, which only the judge folders hold), its deadline, the folder of its launch
    inputs (`inputs`) and the folder its judge folders are written to when it is judged (`judges`), both in the runner's
    layout (Runner.task_dir, Runner.judge_dir)."""
    spec = json.loads(pathlib.Path(spec).read_text(encoding="utf-8"))
    return spec["name"], {"role": "confirmatory", "deadlineSeconds": spec["deadlineSeconds"], "steps": spec["steps"],
                          "fixed": spec["fixed"], "plants": {name: {"step": p["step"]} for name, p in spec["plants"].items()},
                          "prepared": spec["inputs"], "judges": spec["judges"], "inputEffort": spec.get("inputEffort") or {}}


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
    """The stability gate: per arm, three Library fixed starts in a row and the Makerspace chain, and the first fixed
    start of each prepared series once, to show that both arms run on its inputs; the first arm alternates."""
    name, steps = GATE_FIXED
    units = []
    for i, step in enumerate(steps):
        units += pair({"id": f"{name}-fixed-{step}-gate", "kind": "fixed", "series": name, "step": step, "repetition": 1}, "MK"[i % 2])
    for s_name, s in series.items():
        if s["role"] == "confirmatory":
            units += pair({"id": f"{s_name}-fixed-{s['fixed'][0]}-gate", "kind": "fixed", "series": s_name, "step": s["fixed"][0],
                           "repetition": 1}, "K")
    units += pair({"id": f"{GATE_CHAIN}-chain-gate", "kind": "chain", "series": GATE_CHAIN, "steps": series[GATE_CHAIN]["steps"],
                   "repetition": 1}, "K")
    return units


def launches(units):
    return sum(len(u["steps"]) if u["kind"] == "chain" else 1 for u in units)


def plan(protocol, machine, prepared=()):
    series = {name: public_series(name) for name in ("library", "makerspace")}
    for path in prepared:
        name, spec = prepared_series(path)
        series[name] = spec
    if protocol == "counted":
        units = counted_units(series)
    elif protocol == "gate":
        units = gate_units(series)
    else:
        raise SystemExit(f"no protocol {protocol!r}")
    planned = launches(units)
    return {**machine, "protocol": protocol, "series": series, "relaunches": 2, "launchBound": planned * 3 // 2,
            "contrasts": CONTRASTS, "reviews": {"repetition": 1, "chains": True} if protocol == "counted" else {}, "units": units}


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
    `docker events` stream kept open for the whole window sees it: the engine keeps only its last 256 events, fewer
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
    """The Claude account login of the series, never an API key. A launch or review needs it to outlive its time by
    twenty minutes. When it cannot, the runner refreshes it first, holding the engine lock for that alone, since a run
    in progress uses a copy of the access token: it sets the login's expiry to now, with no token changed, and lets
    `claude -p` refresh it as the CLI does near expiry. The file is replaced only if no other process changed it in
    between, and its expiry is put back if the CLI did not refresh it."""

    def __init__(self, path, lock, model, log, claude=None, sleep=time.sleep, clock=time.time, halted=lambda: False):
        self.path, self.lock, self.log, self.sleep, self.clock = pathlib.Path(path), pathlib.Path(lock), log, sleep, clock
        self.halted = halted
        provider, _, self.model = model.partition(":")
        if provider != "claude":
            raise SystemExit(f"the series runner refreshes only a Claude login, not {provider!r}")
        self.claude = claude or [shutil.which("claude") or "claude"]

    def oauth(self):
        return json.loads(self.path.read_text(encoding="utf-8"))["claudeAiOauth"]

    def left(self):
        return self.oauth()["expiresAt"] / 1000 - self.clock()

    def ensure(self, seconds):
        need = seconds + LOGIN_MARGIN
        if self.left() >= need:
            return
        renewable = self.oauth().get("refreshTokenExpiresAt")
        if renewable and renewable / 1000 <= self.clock() + 60:
            raise Stop("the login's refresh token has expired: log in again")
        with self.engine_lock():
            if self.left() < need:  # the wait for the lock may have let another process refresh it
                self.refresh()
        if self.left() < need:
            raise Stop(f"the login could not be refreshed: {round(self.left())} s left, {need} s needed")

    @contextlib.contextmanager
    def engine_lock(self):
        waited, denied = False, 0
        while True:
            try:
                fd = os.open(self.lock, os.O_CREAT | os.O_EXCL | os.O_WRONLY)
                break
            except (FileExistsError, PermissionError) as e:  # PermissionError: a lock file being removed on Windows
                denied = denied + 1 if isinstance(e, PermissionError) else 0
                if denied > 5:
                    raise Stop(f"the engine lock cannot be created: {e}")
                if self.halted():
                    raise Halt()
                if not waited:
                    self.log(event="login-refresh-waits", why="another holder has the engine lock; its run uses the access token")
                    waited = True
                self.sleep(60)
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            json.dump({"holder": "series runner: login refresh", "started_utc": now()}, f)
        try:
            yield
        finally:
            os.unlink(self.lock)

    def replace(self, expected, data):
        """Write data over the login file, unless it no longer holds the bytes `expected`: another process refreshed it."""
        temporary = self.path.with_name(self.path.name + ".series-runner")
        temporary.unlink(missing_ok=True)
        fd = os.open(temporary, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600)
        try:
            with os.fdopen(fd, "w", encoding="utf-8") as f:
                json.dump(data, f)
            for _ in range(5):
                if self.path.read_bytes() != expected:
                    return False
                try:
                    os.replace(temporary, self.path)
                    return True
                except PermissionError:  # another process has the file open on Windows: try again
                    self.sleep(1)
            raise Stop("the login could not be marked for refresh: its file stays in use")
        finally:
            temporary.unlink(missing_ok=True)

    def set_expiry(self, value):
        raw = self.path.read_bytes()
        data = json.loads(raw)
        before = data["claudeAiOauth"]["expiresAt"]
        data["claudeAiOauth"]["expiresAt"] = value
        return before if self.replace(raw, data) else None

    def refresh(self):
        before, token = round(self.left()), self.oauth()["accessToken"]
        expiry = self.set_expiry(int(self.clock() * 1000))
        if expiry is None:
            self.log(event="login-changed", why="another process changed the login while the runner marked it")
            return
        env = {k: v for k, v in os.environ.items() if not k.upper().startswith(("ANTHROPIC_", "CLAUDE_", "CLAUDECODE"))}
        with tempfile.TemporaryDirectory() as tmp:
            try:
                p = subprocess.run([*self.claude, "-p", "Reply with the word OK.", "--model", self.model, "--strict-mcp-config",
                                    "--disable-slash-commands", "--no-session-persistence"], cwd=tmp, env=env, capture_output=True,
                                   text=True, timeout=300, stdin=subprocess.DEVNULL)
                code = p.returncode
            except (OSError, subprocess.TimeoutExpired) as e:
                code = type(e).__name__
        refreshed = self.oauth()["accessToken"] != token
        if not refreshed:  # the token is still the one that was valid: put its expiry back
            self.set_expiry(expiry)
        self.log(event="login-refresh", exit=code, refreshed=refreshed, secondsLeftBefore=before, secondsLeft=round(self.left()))


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
        self.docker = config.get("docker") or shutil.which("docker") or "docker"
        self.engine = config.get("engine") or [self.docker]
        self.markitect = pathlib.Path(config["bin"]) / ("markitect.exe" if os.name == "nt" else "markitect")
        self.sleep = sleep
        self.guard = guard
        self.login = login or Login(config["login"], config["lock"], config["model"], self.log, sleep=sleep, halted=self.halted)
        ledger = self.work / "ledger.json"
        self.ledger = json.loads(ledger.read_text(encoding="utf-8")) if ledger.exists() else {
            "format": 1, "series": f"{config['protocol']}-{config['rev'][:8]}", "signatures": "signatures.json", "tasks": {},
            "launches": [], "chains": [], "attempts": [], "done": [], "progress": {}, "pending": [], "waits": [], "windows": [],
            "failures": [], "freeze": None, "contrasts": config["contrasts"]}

    # --- bookkeeping ---
    def log(self, **event):
        event["time"] = now()
        self.work.mkdir(parents=True, exist_ok=True)
        with open(self.work / "series.jsonl", "a", encoding="utf-8") as f:
            f.write(json.dumps(event) + "\n")

    def save(self):
        temporary = self.work / "ledger.json.tmp"
        temporary.write_text(json.dumps(self.ledger, indent=2) + "\n", encoding="utf-8")
        os.replace(temporary, self.work / "ledger.json")

    def rel(self, path):
        return pathlib.Path(path).relative_to(self.work).as_posix()

    def halted(self):
        return (self.work / "STOP").exists()

    def nap(self, seconds):
        """Wait, but end the window as soon as WORK/STOP appears."""
        left = seconds
        while True:
            if self.halted():
                raise Halt()
            if left <= 0:
                return
            self.sleep(min(60, left))
            left -= 60

    def failure(self, name, detail):
        """A judgment, review or report that did not come out: logged, kept in the ledger and in done.json."""
        self.ledger["failures"].append({"name": name, "detail": detail[-500:], "time": now()})
        self.log(event="failure", name=name, detail=detail[-500:])
        self.save()

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
                self.nap(120)
                continue
            return code, err

    @contextlib.contextmanager
    def runner_lock(self):
        """One runner per work folder: a second one would take the first one's launch for an interrupted one."""
        self.work.mkdir(parents=True, exist_ok=True)
        for attempt in range(5):
            try:
                fd = os.open(self.work / "runner.lock", os.O_CREAT | os.O_EXCL | os.O_WRONLY)
                break
            except PermissionError as e:  # a lock file being removed on Windows, or a folder the runner may not write
                if attempt == 4:
                    raise SystemExit(f"{self.work / 'runner.lock'} cannot be created: {e}")
                time.sleep(1)
            except FileExistsError:
                raise SystemExit(f"{self.work / 'runner.lock'} exists: another runner runs here, or one ended without removing it")
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            json.dump({"pid": os.getpid(), "started": now()}, f)
        try:
            yield
        finally:
            (self.work / "runner.lock").unlink()

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
        return [os.path.expanduser(h) for h in (*self.c.get("hidden", []), *(s["judges"] for s in self.c["series"].values() if s.get("judges")))]

    def task_name(self, unit):
        if unit["kind"] == "chain":
            return f"{unit['series']}-chain"
        if unit["kind"] == "plant":
            return "plant-" + unit["plant"]
        return f"{unit['series']}-fixed-{unit['step']}"

    def step_of(self, unit):
        return self.series(unit["series"])["plants"][unit["plant"]]["step"] if unit["kind"] == "plant" else unit["step"]

    def task_dir(self, unit):
        """A fixed start's task, `tasks/STEP/`, or a plant's, `plants/NAME/task/`, in the series' inputs; a chain's steps
        are `steps/STEP/` there."""
        if unit["kind"] == "plant":
            return self.prepared(unit["series"]) / "plants" / unit["plant"] / "task"
        return self.prepared(unit["series"]) / "tasks" / unit["step"]

    def series_judge(self, name):
        return self.judges(name) / "judge"

    def judge_dir(self, unit):
        """The series judge, `judge/`, or a plant's judge with its `plant.json`, `plants/NAME/judge/`, in the series'
        judge folders."""
        if unit["kind"] == "plant":
            return self.judges(unit["series"]) / "plants" / unit["plant"] / "judge"
        return self.series_judge(unit["series"])

    def prep(self):
        """The public series' inputs from the frozen revision, in the layout a prepared series brings: the steps, the series
        judge, fixed reference starts with the registry pages their owner wrote, planted steps and their judges, and a
        task per fixed start and plant. A prepared series must have every input its units launch."""
        for name, s in self.c["series"].items():
            units = [u for u in self.c["units"] if u["series"] == name]
            if s.get("prepared"):
                needed = [self.task_dir(u) for u in units if u["kind"] != "chain"]
                needed += [self.prepared(name) / "steps" / step for u in units if u["kind"] == "chain" for step in u["steps"]]
                missing = [str(p) for p in needed if not p.is_dir()]
                if missing:
                    raise SystemExit(f"the prepared series {name} lacks " + ", ".join(sorted(set(missing))))
                continue
            if not units:
                continue
            prep, src = self.prepared(name), self.ev.parent / "evaluation-judges"
            if not (prep / "steps").exists():
                self.must(f"prep-steps-{name}", [self.py, "-I", "-B", self.ev / "series.py", "steps", "--series", name, "--out", prep / "steps"])
            if not self.series_judge(name).exists():  # from the frozen tree, which is the revision's
                self.must(f"prep-judge-{name}", [self.py, "-I", "-B", self.ev / "judges.py", "export", "--source", src / name,
                                                 "--out", self.series_judge(name)])
            tasks = [{"kind": "fixed", "series": name, "step": step} for step in s["fixed"]]
            tasks += [{"kind": "plant", "series": name, "plant": plant} for plant in s["plants"]]
            for task in tasks:
                step = self.step_of(task)
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
                if task["kind"] == "plant":
                    source = prep / "plants" / task["plant"] / "step"
                    if not source.exists():
                        self.must(f"prep-plant-{task['plant']}", [self.py, "-I", "-B", self.ev / "judges.py", "plant", "--plant",
                                                                  src / "plants" / f"{task['plant']}.json", "--step-out", source,
                                                                  "--judge", self.series_judge(name), "--judge-out", self.judge_dir(task)])
                if not self.task_dir(task).exists():
                    self.must(f"prep-task-{name}-{self.task_dir(task).parent.name}-{self.task_dir(task).name}",
                              [self.py, "-I", "-B", self.suite_py, "task", "--task-md", source / "task.md", "--model", source / "model",
                               "--start", start, "--base-model", base, "--deadline-seconds", s["deadlineSeconds"], "--out", self.task_dir(task)])
        for unit in self.c["units"]:
            entry = self.ledger["tasks"].setdefault(self.task_name(unit), {"inputEffort": None, "note": "task.md is rendered from the model"})
            s = self.series(unit["series"])
            if s["role"] == "confirmatory":
                entry.update(inputEffort=s.get("inputEffort", {}).get(self.task_name(unit)), note="task.md written by the author")
            if unit["kind"] == "plant" and s["plants"][unit["plant"]].get("kind"):
                entry["planted"] = s["plants"][unit["plant"]]["kind"]
        (self.work / "signatures.json").write_text(json.dumps(SIGNATURES, indent=2) + "\n", encoding="utf-8")
        self.save()

    def freeze(self):
        """What must stay the same in every window, and as in the gate when the configuration names its ledger
        (`gateLedger`): the frozen tree, the binaries, a prepared series' launch inputs and the images on the engine."""
        src = self.work / "src"
        found = {"src": tree_digest(src) if src.is_dir() else None,
                 "bin": {p.name: sha256(p) for p in sorted(pathlib.Path(self.c["bin"]).iterdir()) if p.is_file()},
                 "inputs": {name: tree_digest(self.prepared(name)) for name, s in self.c["series"].items() if s.get("prepared")},
                 "images": {}}
        for image in (self.c["managerImage"], self.c["sandboxImage"]):
            try:
                p = subprocess.run([*self.engine, "image", "inspect", "--format", "{{.Id}}", image], capture_output=True, text=True,
                                   stdin=subprocess.DEVNULL, timeout=120)
            except (OSError, subprocess.TimeoutExpired) as e:
                raise Stop(f"the engine does not answer: {type(e).__name__}")
            found["images"][image] = p.stdout.strip() == image
            if not found["images"][image]:
                raise Stop(f"the image {image} is not on the engine, or the engine does not answer: {p.stderr.strip()[-200:]}")
        reference = self.ledger["freeze"]
        if reference is None and self.c.get("gateLedger"):
            reference = {k: v for k, v in json.loads(pathlib.Path(self.c["gateLedger"]).read_text(encoding="utf-8"))["freeze"].items()}
            reference["inputs"] = {k: v for k, v in reference.get("inputs", {}).items() if k in found["inputs"]}
        if reference is not None and found != reference:
            changed = [k for k in found if found[k] != reference.get(k)]
            raise Stop("the frozen " + " and ".join(changed) + (" changed since the first window" if self.ledger["freeze"] else " differ from the gate's"))
        if self.ledger["freeze"] is None:
            self.ledger["freeze"] = found
            self.save()

    # --- conditions before a launch ---
    def arm_args(self, unit):
        if unit["arm"] == "M":
            hidden = [x for h in self.hidden() for x in ("--hidden-absent", h)]
            return (["--native", self.c["sandboxImage"], *hidden, "--login", self.c["login"], "--lock", self.c["lock"], "--engine", self.docker],
                    [self.py, "-I", "-B", self.ev / "markitect_arm.py", "--markitect", self.markitect,
                     "--remote-binary", pathlib.Path(self.c["bin"]) / "markitect-remote", "--manager-image", self.c["managerImage"]])
        return (["--sandbox", self.c["sandboxImage"], "--variant-code", self.ev, "--login", self.c["login"], "--lock", self.c["lock"],
                 "--engine", self.docker],
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
            self.nap(120)

    def ready(self, deadline):
        """Before a launch: no STOP, the bound, no hidden material, the rest of a five-hour quota wait, a quiet engine, a
        login that covers the launch, and the guard once more, since a login refresh can outlast its answer."""
        if self.halted():
            raise Halt()
        if len(self.ledger["attempts"]) >= self.c["launchBound"]:
            raise Stop("the launch bound is reached")
        present = [h for h in self.hidden() if pathlib.Path(h).exists()]
        if present:
            raise Stop("hidden material is on the machine: " + ", ".join(present))
        if self.ledger.get("quotaWaitUntil"):
            self.nap(self.ledger["quotaWaitUntil"] - time.time())
            self.ledger["quotaWaitUntil"] = None
            self.save()
        self.quiet()
        self.login.ensure(deadline)
        self.quiet()

    # --- launches ---
    def attempts_of(self, unit_id, step):
        return [a for a in self.ledger["attempts"] if a["unit"] == unit_id and a["step"] == step]

    def resolved(self, done):
        """Where the recorded launches of a task or chain step leave it: None (none yet), "again" or "done". After a stop
        is cleared, a launch that stopped the series is launched again only if it was void with relaunches left, or the
        first invalid one to stop it; otherwise the series goes on with the next unit rather than repeat a launch that
        counted. In the gate only a result completes one."""
        if not done:
            return None
        last = done[-1]
        if self.c["protocol"] == "gate":
            return "done" if last["class"] == "result" else "again"
        if last.get("answer") in (None, "again"):
            return "again"
        if last["answer"] == "stop":
            if last["class"] == "void" and len(done) <= self.c["relaunches"]:
                return "again"
            if last["class"] == "invalid" and sum(a["class"] == "invalid" and a.get("answer") == "stop" for a in done) == 1:
                return "again"
        return "done"

    def settle(self, unit, verdict, number, foreign_used, run):
        """The protocol's answer to one classed launch, the `number`th of its task or step: "done", "again" (relaunch)
        or a stop."""
        arm = [a for a in self.ledger["attempts"] if a["arm"] == unit["arm"]]
        voids = sum(a["class"] == "void" for a in arm)
        if verdict.get("stopsSeries"):
            raise Stop(f"the classifier stops the series: {verdict.get('evidence')}")
        if self.c["protocol"] == "gate" and verdict["class"] != "result":
            raise Stop(f"the stability gate has a launch classed {verdict['class']}, {self.rel(run)}: {verdict.get('evidence')}")
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
        to record), its class with the arm's signatures, the protocol's answer and its records, saved together; then the
        quota and the evidence, and then its judgment, pending until it is valid. A chain's `progress` keeps a step
        whose launch is a result."""
        code, err = self.suite(f"{unit['id']}-{out.name}", args) if args else (None, "the runner stopped during this launch")
        record = classify.load(run / "suite.json")
        verdict = classify.classify(run, SIGNATURES, unit["arm"]) if record else {
            "class": "void", "kind": "suite", "evidence": err[-500:], "stopsSeries": False}
        done = self.attempts_of(unit["id"], step)
        attempt = {"unit": unit["id"], "step": step, "arm": unit["arm"], "run": self.rel(out), "class": verdict["class"],
                   "kind": verdict.get("kind"), "foreign": verdict["class"] == "invalid" and foreign_escapes(run), "exit": code,
                   "time": now(), "answer": None}
        self.ledger["attempts"].append(attempt)
        stops = []
        try:
            attempt["answer"] = self.settle(unit, verdict, len(done) + 1, any(a["foreign"] for a in done), run)
        except Stop as e:
            attempt["answer"] = "stop"
            stops.append(e)
        entry = None
        if record:
            entry = {"id": f"{unit['id']}-{out.name}", "variant": unit["arm"], "task": self.task_name(unit), "series": unit["series"],
                     "repetition": unit.get("repetition", 1), "step": step, "run": self.rel(run), "class": verdict["class"]}
            self.ledger["launches"].append(entry)
            if progress is not None and verdict["class"] == "result":
                progress["kept"].append([step, self.rel(out)])
                progress["start"] = self.rel(run / "variant" / "result.bundle" if record.get("delivered") else out / "step-01" / "task" / "start.bundle")
            if judge and verdict["class"] == "result" and record.get("delivered"):
                self.ledger["pending"].append({"kind": "launch", "id": entry["id"], "judge": str(judge)})
        self.save()
        self.log(event="ran", unit=unit["id"], step=step, run=self.rel(out), exit=code, answer=attempt["answer"], **verdict)
        if record:
            try:
                self.after(run, wait=not stops)
            except Stop as e:
                stops.append(e)
            if not stops and judge and verdict["class"] == "result" and record.get("delivered"):
                self.judge_launch(entry, judge)
        if stops:
            raise Stop("; ".join(str(s) for s in stops))

    def after(self, run, wait=True):
        """The quota the launch's agents saw and a scan of its evidence for the login. A five-hour wait is kept in the
        ledger, so that a window that ends during it waits out the rest when the series goes on."""
        q = quota(run)
        self.log(event="quota", run=self.rel(run), **q)
        hits, _ = files_with_secrets(run, login_secrets(self.c["login"]))
        if hits:
            raise Stop(f"a credential in the evidence of {self.rel(run)}: {hits}")
        if q["refused"] or q["sevenDay"] >= QUOTA_STOP:
            raise Stop(f"quota: {json.dumps(q)}")
        if q["fiveHour"] >= QUOTA_WAIT and q["fiveHourReset"]:
            self.ledger["quotaWaitUntil"] = q["fiveHourReset"] + 60
            self.save()
            self.log(event="quota-wait", seconds=round(self.ledger["quotaWaitUntil"] - time.time()))
            if wait:
                self.nap(self.ledger["quotaWaitUntil"] - time.time())
                self.ledger["quotaWaitUntil"] = None
                self.save()

    def fresh(self, folder, name, mark):
        """The first of NAME, NAME-MARK2, NAME-MARK3 ... that does not exist yet: a judgment or review made again never
        reuses a folder."""
        n = 1
        while (folder / (name if n == 1 else f"{name}-{mark}{n}")).exists():
            n += 1
        return folder / (name if n == 1 else f"{name}-{mark}{n}")

    def pending_of(self, item):
        """The ledger's pending entry for item, found by its kind and ID, and added when there is none."""
        def key(p):
            return p["kind"], p["id"] if p["kind"] == "launch" else p["chain"]["id"]
        for found in self.ledger["pending"]:
            if key(found) == key(item):
                return found
        self.ledger["pending"].append(item)
        self.save()
        return item

    def judge_launch(self, entry, judge):
        """A delivered result's judgment, pending in the ledger until it is valid: while its judge is off the machine,
        when the window ends during it, or when the runner does not live to record it. A judgment that is not valid is
        made once more; a judge that fails twice in a row stops the series."""
        item = self.pending_of({"kind": "launch", "id": entry["id"], "judge": str(judge)})
        if not judge.exists():
            return
        for _ in (1, 2):
            out = self.fresh(self.work / "judgments", entry["id"], "j")
            self.suite(f"judge-{out.name}", ["judge", "--run", self.work / entry["run"], "--judge", judge, "--step", entry["step"],
                                             "--out", out, "--lock", self.c["lock"], "--engine", self.docker])
            if valid(out / "judgment.json"):
                entry["judgment"] = self.rel(out)
                self.ledger["pending"].remove(item)
                self.save()
                return
            self.failure(f"judge-{out.name}", f"no valid judgment in {self.rel(out)}")
        item["failed"] = True
        self.save()
        raise Stop(f"the judge gave no valid judgment of {entry['id']} twice")

    def run_task(self, unit):
        """A fixed start or plant, launched until a launch counts, within the relaunch limits."""
        flags, variant = self.arm_args(unit)
        base, step = self.work / "runs" / unit["id"], self.step_of(unit)
        while True:
            done = self.attempts_of(unit["id"], step)
            if self.resolved(done) == "done":
                return
            run = base / f"a{len(done) + 1}"
            if run.exists():  # launched before the runner ended and never recorded: recorded as it ended
                self.launch(unit, run, run, None, step, self.judge_dir(unit))
                continue
            self.ready(self.series(unit["series"])["deadlineSeconds"])
            self.launch(unit, run, run, ["run", "--task", self.task_dir(unit), "--model", self.c["model"], "--out", run, *flags,
                                         "--", *variant], step, self.judge_dir(unit))

    def run_chain(self, unit):
        """A chain one step at a time: each step a one-step `suite.py chain` from the step before's result (or its start
        when it delivered none) with the unit's one state folder, which every launch of a step starts from as it was
        before the step. A step without a result after its relaunches, or invalid, ends the chain. The steps that
        counted are then judged as one chain."""
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
                    return
                out = base / f"{step}-a{len(done) + 1}"
                run = out / "step-01" / "run"
                if out.exists():
                    self.launch(unit, out, run, None, step, progress=progress)
                    continue
                self.ready(deadline)
                shutil.rmtree(state)
                shutil.copytree(snapshot, state)
                start = ["--start", self.work / progress["start"]] if progress["start"] else []
                self.launch(unit, out, run, ["chain", "--step", self.prepared(unit["series"]) / "steps" / step, *start, "--state", state,
                                             "--deadline-seconds", deadline, "--model", self.c["model"], "--out", out, *flags, "--", *variant],
                            step, progress=progress)
        self.finish_chain(unit)

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
        self.judge_chain({"id": unit["id"], "variant": unit["arm"], "task": self.task_name(unit), "series": unit["series"],
                          "repetition": unit.get("repetition", 1)}, unit["series"], kept)

    def judge_chain(self, chain, series, kept):
        """The chain's judgment, pending until it is valid like a launch's. The suite judges a chain's delivered steps
        only, so only those need a valid judgment. A judgment that is not valid is made once more; a judge that fails
        twice in a row stops the series."""
        item = self.pending_of({"kind": "chain", "chain": chain, "series": series, "kept": kept})
        if not self.series_judge(series).exists():
            return
        whole = self.work / "runs" / f"{chain['id']}-chain"
        rows = json.loads((whole / "chain.json").read_text(encoding="utf-8"))["steps"]
        for _ in (1, 2):
            judged = self.fresh(self.work / "judgments", f"{chain['id']}-chain", "j")
            self.suite(f"judge-chain-{judged.name}", ["judge-chain", "--chain", whole, "--judge", self.series_judge(series),
                                                      "--steps", ",".join(s for s, _ in kept), "--out", judged, "--lock", self.c["lock"],
                                                      "--engine", self.docker])
            steps = {i: judged / f"step-{i:02d}" for i, row in enumerate(rows, 1) if row.get("delivered")}
            if (judged / "chain-judgment.json").exists() and all(valid(s / "judgment.json") for s in steps.values()):
                for i, (_, out) in enumerate(kept, 1):
                    for entry in self.ledger["launches"]:
                        if entry["run"] == f"{out}/step-01/run" and i in steps:
                            entry["judgment"] = self.rel(steps[i])
                self.ledger["chains"].append({**chain, "judgment": self.rel(judged)})
                self.ledger["pending"].remove(item)
                self.save()
                return
            self.failure(f"judge-chain-{judged.name}", f"no valid chain judgment in {self.rel(judged)}")
        item["failed"] = True
        self.save()
        raise Stop(f"the judge gave no valid judgment of chain {chain['id']} twice")

    # --- the end of a window and of the series ---
    def judge_pending(self, retry_failed=False):
        """Every judgment still pending whose judge folder is on the machine, and each plant's kind, which a prepared
        series names only there (`plant.json`). A pending item leaves the ledger only once its judgment is valid. A judge
        that fails here is recorded and the rest go on: only the window that made a launch stops for its judge. An item
        whose judge already failed twice is tried again only when `retry_failed` asks for it (at the end, and by `judge`)."""
        for unit in self.c["units"]:
            plant = classify.load(self.judge_dir(unit) / "plant.json") if unit["kind"] == "plant" else None
            if plant:
                self.ledger["tasks"][self.task_name(unit)]["planted"] = plant["kind"]
        for item in list(self.ledger["pending"]):
            if item.get("failed") and not retry_failed:
                continue
            try:
                if item["kind"] == "launch":
                    if pathlib.Path(item["judge"]).exists():
                        self.judge_launch(next(e for e in self.ledger["launches"] if e["id"] == item["id"]), pathlib.Path(item["judge"]))
                elif self.series_judge(item["series"]).exists():
                    self.judge_chain(item["chain"], item["series"], item["kept"])
            except Stop as e:
                self.log(event="judge-still-failing", item=item.get("id") or item["chain"]["id"], reason=str(e))
        self.save()

    def reviews(self):
        """The code review of every judged result of the protocol's review repetition (fixed starts and every chain's last
        judged step), blinded, in the sandbox, reviewed again once in a fresh folder if the review is not valid. A
        review needs the judge folder's rubric, so it waits for it like judging."""
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
            judge = self.series_judge(entry["series"])
            if entry.get("review") or not judge.exists():
                continue
            for _ in (1, 2):
                out = self.fresh(self.work / "reviews", entry["id"], "r")
                self.login.ensure(REVIEW_SECONDS)
                self.suite(f"review-{out.name}", ["review", "--run", self.work / entry["run"], "--judge", judge, "--out", out,
                                                  "--sandbox", self.c["sandboxImage"], "--variant-code", self.ev, "--login", self.c["login"],
                                                  "--lock", self.c["lock"], "--engine", self.docker])
                if valid(out / "review.json"):
                    entry["review"] = self.rel(out)
                    self.save()
                    break
                self.failure(f"review-{out.name}", f"no valid review in {self.rel(out)}")

    def report(self):
        """The report over every launch (`report/all`), and per series one over everything (`report/SERIES`), one over its
        fixed starts alone (`report/SERIES-fixed`, where the primary contrast is read) and one over its chains
        (`report/SERIES-chains`), each from its own ledger beside the whole one. A series judged later is withheld from
        every report until `judge` has judged all of it, so that nothing of it is read while launches still run."""
        out = self.work / "report"
        if out.exists():
            shutil.rmtree(out)
        withheld = {name for name, s in self.c["series"].items() if s.get("judges") and not self.ledger.get("released", {}).get(name)}
        whole = self.work / "ledger-all.json"
        whole.write_text(json.dumps({**self.ledger, "launches": [e for e in self.ledger["launches"] if e["series"] not in withheld],
                                     "chains": [c for c in self.ledger["chains"] if c["series"] not in withheld]}, indent=2) + "\n",
                         encoding="utf-8")
        parts = [("all", whole)]
        for name in dict.fromkeys(u["series"] for u in self.c["units"] if u["series"] not in withheld):
            mine = [e for e in self.ledger["launches"] if e["series"] == name]
            chains = [c for c in self.ledger["chains"] if c["series"] == name]
            for part, launches_, chains_ in ((name, mine, chains), (f"{name}-fixed", [e for e in mine if "-fixed-" in e["task"]], []),
                                             (f"{name}-chains", [e for e in mine if e["task"] == f"{name}-chain"], chains)):
                ledger = self.work / f"ledger-{part}.json"
                ledger.write_text(json.dumps({**self.ledger, "series": f"{self.ledger['series']}-{part}", "launches": launches_,
                                              "chains": chains_}, indent=2) + "\n", encoding="utf-8")
                parts.append((part, ledger))
        for part, ledger in parts:
            code, err = self.suite(f"report-{part}", ["report", "--ledger", ledger, "--out", out / part])
            if code != 0:
                self.failure(f"report-{part}", err)

    def final_check(self):
        """Every file of the evidence scanned for the login's tokens and key shapes, and what is left by label."""
        hits, scanned = files_with_secrets(self.work, login_secrets(self.c["login"]), skip=("src",))
        left, errors = {}, []
        for kind, args in (("containers", ["ps", "-a", "-q"]), ("volumes", ["volume", "ls", "-q"]), ("networks", ["network", "ls", "-q"])):
            left[kind] = []
            for label in LEFTOVER_LABELS:
                try:
                    p = subprocess.run([*self.engine, *args, "--filter", f"label={label}"], capture_output=True, text=True,
                                       stdin=subprocess.DEVNULL, timeout=120)
                    left[kind] += p.stdout.split() if p.returncode == 0 else []
                    errors += [] if p.returncode == 0 else [f"{kind} {label}: {p.stderr.strip()[-200:]}"]
                except (OSError, subprocess.TimeoutExpired) as e:
                    errors.append(f"{kind} {label}: {type(e).__name__}")
        check = {"scanned": scanned, "secretHits": hits, "leftovers": left, "engineErrors": errors}
        self.log(event="final-check", **check)
        return check

    def gate_verdict(self, check):
        """The stability gate passes when every launch is a result, every public result and chain has a valid judgment,
        the final check is clean, and M's Makerspace chain landed its first step."""
        public = [e for e in self.ledger["launches"] if self.series(e["series"])["role"] == "public"]
        delivered = [e for e in public if e["class"] == "result" and (classify.load(self.work / e["run"] / "suite.json") or {}).get("delivered")]
        m_chain = [e for e in public if e["variant"] == "M" and e["task"] == f"{GATE_CHAIN}-chain"]
        first = classify.load(self.work / m_chain[0]["run"] / "suite.json") if m_chain else None
        checks = {"everyLaunchAResult": all(a["class"] == "result" for a in self.ledger["attempts"]),
                  "everyResultJudged": all(e.get("judgment") and valid(self.work / e["judgment"] / "judgment.json") for e in delivered),
                  "everyChainJudged": len(self.ledger["chains"]) == sum(u["kind"] == "chain" for u in self.c["units"]),
                  "noCredential": not check["secretHits"], "nothingLeft": not any(check["leftovers"].values()) and not check["engineErrors"],
                  "mLandedTheFirstChainStep": bool(first and first.get("delivered") and ((first.get("result") or {}).get("changedPaths")))}
        verdict = {"passes": all(checks.values()), "checks": checks, "time": now()}
        (self.work / "gate.json").write_text(json.dumps(verdict, indent=2) + "\n", encoding="utf-8")
        self.log(event="gate", **verdict)

    def wrap_up(self, complete):
        """After every window: the report and the final check; after the last unit also the reviews, the gate's verdict
        and done.json."""
        for step in ((lambda: self.judge_pending(retry_failed=True), self.reviews) if complete else ()) + (self.report,):
            try:
                step()
            except Exception as e:  # noqa: BLE001 - recorded; the next one still runs
                self.failure(getattr(step, "__name__", "judge_pending"), f"{type(e).__name__}: {e}")
        try:
            check = self.final_check()
        except Exception as e:  # noqa: BLE001
            check = {"error": f"{type(e).__name__}: {e}", "secretHits": None, "leftovers": {}, "engineErrors": [str(e)]}
            self.failure("final-check", check["error"])
        if complete:
            if self.c["protocol"] == "gate":
                self.gate_verdict(check)
            (self.work / "done.json").write_text(json.dumps({"ended": now(), "launches": len(self.ledger["attempts"]),
                                                             "pending": self.ledger["pending"], "failures": self.ledger["failures"],
                                                             **check}, indent=2) + "\n", encoding="utf-8")

    def run(self, prep_only=False):
        if (self.work / "done.json").exists():
            raise SystemExit("the series is done")
        with self.runner_lock():
            self.log(event="begin", rev=self.c["rev"], protocol=self.c["protocol"], units=len(self.c["units"]), done=len(self.ledger["done"]),
                     runner=sha256(__file__))
            self.prep()
            if prep_only:
                return
            self.guard = self.guard or Guard(self.engine).start()
            window = {"began": now(), "launchesBefore": len(self.ledger["attempts"])}
            self.ledger["windows"].append(window)
            complete = False
            try:
                self.freeze()
                self.judge_pending()  # what an earlier window left pending: a stop or the runner's end during a judgment
                for unit in self.c["units"]:
                    if unit["id"] in self.ledger["done"]:
                        continue
                    self.run_chain(unit) if unit["kind"] == "chain" else self.run_task(unit)
                    self.ledger["done"].append(unit["id"])
                    self.save()
                complete = True
            except Halt:
                self.log(event="halt", reason=(self.work / "STOP").read_text(encoding="utf-8", errors="replace").strip())
            except Exception as e:  # noqa: BLE001 - a stop rule, or the runner's own failure: either way a person looks first
                reason = str(e) if isinstance(e, Stop) else f"the runner failed: {type(e).__name__}: {e}"
                (self.work / "STOP").write_text(reason + "\n", encoding="utf-8")
                self.log(event="stop", reason=reason)
            window.update(ended=now(), launches=len(self.ledger["attempts"]) - window["launchesBefore"])
            self.save()
            self.wrap_up(complete)
            self.log(event="end", complete=complete, launches=len(self.ledger["attempts"]))

    def judge(self):
        """Judge, review and report what waited for judge folders that were off the machine during launches."""
        with self.runner_lock():
            try:
                self.judge_pending(retry_failed=True)
                self.reviews()
            except Stop as e:
                self.failure("judge", str(e))
            waiting = {p["series"] if p["kind"] == "chain" else next(e["series"] for e in self.ledger["launches"] if e["id"] == p["id"])
                       for p in self.ledger["pending"]}
            for name, s in self.c["series"].items():
                ran = all(u["id"] in self.ledger["done"] for u in self.c["units"] if u["series"] == name)
                if s.get("judges") and ran and name not in waiting:
                    self.ledger.setdefault("released", {})[name] = True
            self.save()
            self.report()
            self.log(event="judged", pending=len(self.ledger["pending"]))


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    sub = p.add_subparsers(dest="command", required=True)
    pl = sub.add_parser("plan")
    pl.add_argument("--protocol", required=True, choices=("counted", "gate"))
    pl.add_argument("--machine", required=True)
    pl.add_argument("--prepared", action="append", default=[], help="a prepared series' spec (JSON)")
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
