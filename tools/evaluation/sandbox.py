"""The evaluation sandbox: one container per run, on a network whose only route out is the provider.

`suite.py run --sandbox IMAGE` and `chain --sandbox IMAGE` start the variant
command inside one container of the sandbox image (sandbox/Dockerfile), the
same for every variant:

- mounts: the task read-only at /task, the variant's output at /out and its
  state at /state writable, the variant's own code read-only at /variant, and
  an access-only copy of the provider login read-only at /login/credentials.json
  (EVALUATION_LOGIN); nothing else from the host, never the engine socket;
- a read-only root, no capabilities, no privilege gain, a non-root user and a
  private /tmp, where HOME lives;
- the series' one safety limit (CPUs, memory, processes, /tmp);
- an internal network without a route out. An egress gateway, the run's second
  container, sits on that network and on an ordinary one; the sandbox's
  /etc/hosts maps each provider endpoint the series allows to the gateway, which
  forwards TLS on port 443 by server name and refuses every other name. No tool
  needs a proxy setting.

Every engine call is recorded. Afterwards the containers and networks are
removed and counted by label, the login copy is deleted, and the run's outputs
are searched for the login; files that contain it are deleted.
"""
import datetime
import hashlib
import json
import os
import pathlib
import re
import subprocess
import tempfile
import time
import uuid

HERE = pathlib.Path(__file__).resolve().parent
DEFAULT_SERIES = HERE / "sandbox" / "series.json"
LABEL_KEY = "markitect.evaluation"
LOGIN_TARGET = "/login/credentials.json"
GATEWAY = {"cpus": "1", "memory": "268435456", "pids": "256"}
ENGINE_SECONDS, READY_SECONDS, KILL_SECONDS = 60, 30, 20
LOGIN_MARGIN = 300
NO_DNS = "192.0.2.1"  # TEST-NET-1: never routed
IMAGE_ID = re.compile(r"sha256:[0-9a-f]{64}")
HOST = re.compile(r"[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+")


class SandboxError(Exception):
    pass


def sha256_bytes(data):
    return hashlib.sha256(data).hexdigest()


# --- Configuration -------------------------------------------------------------

def load_series(path):
    """The series' sandbox settings: one safety limit for every run and the endpoints each provider may reach."""
    raw = pathlib.Path(path).read_bytes()
    try:
        series = json.loads(raw)
        limits, egress = series["limits"], series["egress"]
        checked = {"cpus": str(limits["cpus"]), "memoryBytes": int(limits["memoryBytes"]), "pids": int(limits["pids"]),
                   "tmpBytes": int(limits["tmpBytes"])}
        float(checked["cpus"])
    except (ValueError, KeyError, TypeError) as e:
        raise SandboxError(f"{path}: not a series configuration: {e}")
    if min(float(checked["cpus"]), checked["memoryBytes"], checked["pids"], checked["tmpBytes"]) <= 0:
        raise SandboxError(f"{path}: every limit must be positive")
    if checked["tmpBytes"] > checked["memoryBytes"]:
        raise SandboxError(f"{path}: /tmp lives in memory and cannot exceed the memory limit")
    for provider, hosts in egress.items():
        if not isinstance(hosts, list) or not hosts or not all(isinstance(h, str) and HOST.fullmatch(h) for h in hosts):
            raise SandboxError(f"{path}: egress for {provider} must be a list of lower-case host names")
    return {"limits": checked, "egress": egress, "sha256": sha256_bytes(raw)}


def host_path(path):
    """A host path as the engine CLI takes it in a --mount value."""
    s = str(pathlib.Path(path).resolve())
    if os.name == "nt":
        if not re.fullmatch(r"[A-Za-z]:\\.*", s):
            raise SandboxError(f"{s}: the engine mounts only drive paths")
        s = s.replace("\\", "/")
    if any(c in s for c in ',"\n'):
        raise SandboxError(f"{s}: a mount path cannot contain a comma, a quote or a line break")
    return s


def container_user():
    """The host user on Linux, so that outputs stay the user's; the image's user 1000 on Docker Desktop."""
    if os.name == "nt":
        return "1000:1000"
    if os.getuid() == 0:
        raise SandboxError("run the suite as a non-root user")
    return f"{os.getuid()}:{os.getgid()}"


def access_login(provider, raw, seconds):
    """The run's copy of the provider login: access only, valid beyond the run; and the secrets to search for."""
    if provider != "claude":
        raise SandboxError(f"the sandbox has no login copy for {provider}")
    try:
        login = json.loads(raw)
    except ValueError:
        login = None
    oauth = login.get("claudeAiOauth") if isinstance(login, dict) else None
    if not isinstance(oauth, dict) or not oauth.get("accessToken") or not isinstance(oauth.get("expiresAt"), int):
        raise SandboxError("the login holds no Claude account login")
    left = oauth["expiresAt"] / 1000 - time.time()
    if left < seconds + LOGIN_MARGIN:
        raise SandboxError(f"the Claude login expires in {int(left)} s; use Claude once to refresh it, then start again")
    copy = {"claudeAiOauth": {k: v for k, v in oauth.items() if k not in ("refreshToken", "refreshTokenExpiresAt")}}
    return json.dumps(copy).encode(), [oauth[k].encode() for k in ("accessToken", "refreshToken") if oauth.get(k)]


# --- Engine ----------------------------------------------------------------------

class Engine:
    """Run engine CLI steps with a finite timeout; keep every step's record."""

    def __init__(self, argv):
        self.argv, self.steps = list(argv), []

    def __call__(self, step, args, timeout=ENGINE_SECONDS):
        started = time.monotonic()
        record = {"step": step, "argv": list(args)}
        try:
            p = subprocess.run([*self.argv, *args], capture_output=True, text=True, errors="replace", timeout=timeout,
                               stdin=subprocess.DEVNULL)
            record["exit"], out, err = p.returncode, p.stdout, p.stderr
        except subprocess.TimeoutExpired:
            record["exit"], out, err = None, "", "timed out"
        except OSError as e:
            record["exit"], out, err = None, "", str(e)
        record["seconds"] = round(time.monotonic() - started, 3)
        if err.strip():
            record["stderr"] = err.strip()[-2000:]
        self.steps.append(record)
        return record["exit"], out

    def must(self, step, args, timeout=ENGINE_SECONDS):
        code, out = self(step, args, timeout)
        if code != 0:
            raise SandboxError(f"engine step {step} failed")
        return out


def hardening(user):
    return ["--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--user", user]


class engine_lock:
    """Exclusive use of the engine and the account, shared with the other evaluation and comparison runners.

    The lock file is created exclusively before the first engine call and removed afterwards, only by its holder.
    """

    def __init__(self, path, holder, run):
        self.path, self.record = (pathlib.Path(path) if path else None), {"holder": holder, "run": run}

    def __enter__(self):
        if self.path:
            self.record["started_utc"] = datetime.datetime.now(datetime.timezone.utc).isoformat()
            try:
                fd = os.open(self.path, os.O_CREAT | os.O_EXCL | os.O_WRONLY)
            except FileExistsError:
                raise SandboxError("the engine is busy: " + self.path.read_text(encoding="utf-8", errors="replace").strip())
            with os.fdopen(fd, "w", encoding="utf-8") as f:
                json.dump(self.record, f)
        return self

    def __exit__(self, *exc):
        if self.path:
            self.path.unlink()
        return False


# --- Secrets ---------------------------------------------------------------------

def files_containing(root, secrets):
    """Files below root that contain a secret; a bundle is searched through its unpacked objects.

    Links and special files the variant left are never followed: they are reported as hits and removed with them.
    """
    found = []
    for path in sorted(pathlib.Path(root).rglob("*")):
        if path.is_symlink() or (path.exists() and not path.is_dir() and not path.is_file()):
            found.append(path)
            continue
        if not path.is_file():
            continue
        data = path.read_bytes()
        if path.suffix == ".bundle" and data:
            with tempfile.TemporaryDirectory() as tmp:
                ok = (subprocess.run(["git", "init", "-q", "--bare", tmp], capture_output=True).returncode == 0
                      and subprocess.run(["git", "-C", tmp, "bundle", "unbundle", str(path.resolve())], capture_output=True).returncode == 0)
                objects = subprocess.run(["git", "-C", tmp, "cat-file", "--batch-all-objects", "--batch"], capture_output=True)
                # A bundle Git cannot read is kept from delivery as if it held a secret.
                data = objects.stdout if ok and objects.returncode == 0 else None
        if data is None or any(s in data for s in secrets):
            found.append(path)
    return found


# --- One run -----------------------------------------------------------------------

class Sandbox:
    """The settings of every sandboxed run of one series."""

    def __init__(self, engine, image, series, variant_code, login, provider):
        if not IMAGE_ID.fullmatch(image):
            raise SandboxError(f"--sandbox must be an engine image ID (sha256:...), not {image!r}")
        self.engine, self.image, self.series, self.provider = list(engine), image, series, provider
        if provider not in series["egress"]:
            raise SandboxError(f"the series allows no egress for {provider}")
        self.hosts = list(series["egress"][provider])
        if not pathlib.Path(variant_code).is_dir():
            raise SandboxError(f"{variant_code}: the variant code is not a directory")
        self.variant_code = host_path(variant_code)
        self.login = pathlib.Path(login)
        self.user = container_user()

    def describe(self):
        return {"image": self.image, "user": self.user, "limits": self.series["limits"], "seriesSha256": self.series["sha256"],
                "egressAllow": self.hosts, "variantCode": self.variant_code}

    def sandbox_argv(self, name, label, network, gateway_ip, task, out, state, login, model, variant):
        limits = self.series["limits"]
        mounts = [("bind", task, "/task", True), ("bind", out, "/out", False)]
        if state:
            mounts.append(("bind", state, "/state", False))
        mounts += [("bind", self.variant_code, "/variant", True), ("bind", login, LOGIN_TARGET, True)]
        # The engine's resolver forwards to an address that has no route from the internal network, so names outside
        # /etc/hosts never resolve.
        argv = ["run", "--name", name, "--label", label, "--init", "--network", network, "--dns", NO_DNS,
                *[f"--add-host={h}:{gateway_ip}" for h in self.hosts], *hardening(self.user),
                "--tmpfs", f"/tmp:rw,exec,nosuid,nodev,mode=1777,size={limits['tmpBytes']}",
                "--cpus", limits["cpus"], "--memory", str(limits["memoryBytes"]), "--memory-swap", str(limits["memoryBytes"]),
                "--pids-limit", str(limits["pids"])]
        for kind, src, dst, ro in mounts:
            argv += ["--mount", f"type={kind},src={src},dst={dst}" + (",readonly" if ro else "")]
        argv += ["-e", f"EVALUATION_MODEL={model}", "-e", f"EVALUATION_LOGIN={LOGIN_TARGET}", "--workdir", "/tmp", self.image,
                 *variant, "/task", "/out", *(["/state"] if state else [])]
        return argv

    conditions = "sandbox"

    def valid(self, record):
        """Whether the run took place under the evaluation's conditions: the sandbox stood up and left nothing behind."""
        return "error" not in record and "container" in record and not record["residue"]

    def run(self, task, out, state, variant, model, deadline, run_dir, launch):
        """Run the variant command in a fresh sandbox.

        launch(argv, on_deadline) runs the engine CLI under the deadline and calls on_deadline() first when it
        expires, so that the container stops before the CLI does. Returns the launch's timing (exitCode, timedOut,
        wallSeconds) and the sandbox record; an "error" in the record means the run did not take place as intended.
        """
        run_id = "eval-" + uuid.uuid4().hex[:12]
        label = f"{LABEL_KEY}={run_id}"
        internal, gateway, box = (f"{run_id}-{n}" for n in ("internal", "gateway", "sandbox"))
        engine = Engine(self.engine)
        record = {**self.describe(), "runId": run_id, "network": internal, "steps": engine.steps}
        timing = {"exitCode": None, "timedOut": False, "wallSeconds": 0.0}
        networks, containers, secrets = [], [], []
        with tempfile.TemporaryDirectory() as tmp:
            login_file = pathlib.Path(tmp) / "credentials.json"
            try:
                login, secrets = access_login(self.provider, self.login.read_bytes(), deadline)
                login_file.write_bytes(login)
                os.chmod(login_file, 0o600)
                del login
                if engine.must("image", ["image", "inspect", "--format", "{{.Id}}", self.image]).strip() != self.image:
                    raise SandboxError("the image ID differs")
                gateway_ip = open_egress(engine, run_id, label, self.image, self.hosts, networks, containers)
                record["gatewayAddress"] = gateway_ip
                mounts = {"task": host_path(task), "out": host_path(out), "state": host_path(state) if state else None,
                          "login": host_path(login_file)}
                containers.append(box)
                argv = self.sandbox_argv(box, label, internal, gateway_ip, mounts["task"], mounts["out"], mounts["state"],
                                         mounts["login"], model, variant)
                record["argv"] = argv

                def stop():
                    # At the deadline the container stops before the CLI, so nothing it writes later can count.
                    if engine("sandbox-kill", ["kill", box], KILL_SECONDS)[0] != 0:
                        record["killFailed"] = True
                timing = launch([*self.engine, *argv], stop)
                code, state_json = engine("sandbox-state", ["inspect", "--format", "{{json .State}}", box])
                s = json.loads(state_json) if code == 0 else None
                if not isinstance(s, dict) or str(s.get("StartedAt", "")).startswith("0001-"):
                    raise SandboxError("the sandbox container never started")
                record["container"] = {"exitCode": s.get("ExitCode"), "oomKilled": s.get("OOMKilled")}
                if record.get("killFailed") and s.get("Running"):
                    raise SandboxError("the sandbox container could not be stopped at the deadline")
                code, gw = engine("gateway-state", ["inspect", "--format", "{{json .State.Running}}", gateway])
                if code != 0 or gw.strip() != "true":
                    raise SandboxError("the egress gateway stopped during the run")
            except Exception as e:  # noqa: BLE001 - whatever failed, the run is recorded and cleaned up
                record["error"] = f"{type(e).__name__}: {e}"
            finally:
                close_egress(engine, run_id, label, run_dir, record, networks, containers)
                login_file.unlink(missing_ok=True)
        quarantine(record, secrets, run_dir, state)
        return timing, record


def open_egress(engine, run_id, label, image, hosts, networks, containers):
    """The run's internal network and its egress gateway; returns the gateway's address on the internal network.

    Refuses while another run's containers or networks exist. Every name it creates is added to networks and
    containers at once, so that close_egress removes it whatever fails later."""
    internal, egress, gateway = (f"{run_id}-{n}" for n in ("internal", "egress", "gateway"))
    leftovers = census(engine, "census-before", LABEL_KEY)
    if leftovers:
        raise SandboxError("evaluation containers or networks of another run exist: " + ", ".join(leftovers))
    # The internal network's bridge carries no host address (isolated gateway mode), so nothing on the
    # engine's host is reachable from it.
    for name, extra in ((internal, ["--internal", "--opt", "com.docker.network.bridge.gateway_mode_ipv4=isolated"]), (egress, [])):
        networks.append(name)
        engine.must("network-" + name[len(run_id) + 1:], ["network", "create", *extra, "--label", label, name])
    containers.append(gateway)
    allow = [a for h in hosts for a in ("--allow", h)]
    engine.must("gateway-create", ["create", "--name", gateway, "--label", label, "--network", egress, *hardening("1000:1000"),
                                   "--sysctl", "net.ipv4.ip_unprivileged_port_start=443", "--cpus", GATEWAY["cpus"],
                                   "--memory", GATEWAY["memory"], "--pids-limit", GATEWAY["pids"], "--entrypoint", "python3", image,
                                   "-I", "-B", "/opt/evaluation/egress.py", *allow])
    engine.must("gateway-connect", ["network", "connect", internal, gateway])
    engine.must("gateway-start", ["start", gateway])
    nets = json.loads(engine.must("gateway-address", ["inspect", "--format", "{{json .NetworkSettings.Networks}}", gateway]))
    gateway_ip = ((nets or {}).get(internal) or {}).get("IPAddress")
    if not gateway_ip:
        raise SandboxError("the gateway has no address on the internal network")
    wait_ready(engine, gateway)
    return gateway_ip


def close_egress(engine, run_id, label, run_dir, record, networks, containers):
    """Keep the gateway's log, remove the run's containers and networks, and count what is left by the run's label."""
    gateway = f"{run_id}-gateway"
    if gateway in containers:
        code, logs = engine("gateway-logs", ["logs", gateway])
        if code == 0:
            record["egress"] = egress_summary(logs)
            try:
                (run_dir / "egress.jsonl").write_text(logs, encoding="utf-8")
            except OSError as e:
                record.setdefault("error", f"the gateway log could not be kept: {e}")
    for name in reversed(containers):
        engine("remove-" + name[len(run_id) + 1:], ["rm", "-f", name], KILL_SECONDS)
    for name in reversed(networks):
        engine("remove-" + name[len(run_id) + 1:], ["network", "rm", name], KILL_SECONDS)
    try:
        record["residue"] = census(engine, "census-after", label)
    except SandboxError as e:
        record["residue"] = [f"unknown: {e}"]


def quarantine(record, secrets, run_dir, state):
    """Delete, unread, every file of the run and the chain's state that holds the login."""
    if secrets:
        # The state directory outlives the run in a chain, so it is searched too.
        hits = [(root, p) for root in (run_dir, state) if root for p in files_containing(root, secrets)]
        for _, p in hits:
            p.unlink()
        record["quarantine"] = [(p.relative_to(run_dir) if root == run_dir else pathlib.Path("state") / p.relative_to(root)).as_posix()
                                for root, p in hits]


def census(engine, step, label):
    """Names of containers and networks with the label; a failed query refuses."""
    names = []
    for kind, args in (("containers", ["ps", "-a", "--filter", "label=" + label, "--format", "{{.Names}}"]),
                       ("networks", ["network", "ls", "--filter", "label=" + label, "--format", "{{.Name}}"])):
        code, out = engine(f"{step}-{kind}", args)
        if code != 0:
            raise SandboxError(f"census query failed: {step}-{kind}")
        names += out.split()
    return names


def wait_ready(engine, gateway):
    ends = time.monotonic() + READY_SECONDS
    while time.monotonic() < ends:
        code, logs = engine("gateway-ready", ["logs", gateway])
        if code == 0 and '"ready": true' in logs:
            return
        time.sleep(0.5)
    raise SandboxError("the egress gateway did not become ready")


def egress_summary(logs):
    """Connections per allowed host and per refused server name, from the gateway's log."""
    allowed, refused = {}, {}
    for line in logs.splitlines():
        try:
            entry = json.loads(line)
        except ValueError:
            continue
        if not isinstance(entry, dict) or "allowed" not in entry:
            continue
        counts, name = (allowed, entry["sni"]) if entry["allowed"] else (refused, entry.get("sni") or "(none)")
        counts[name] = counts.get(name, 0) + 1
    return {"allowed": allowed, "refused": refused}
