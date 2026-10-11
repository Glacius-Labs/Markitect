"""A whole study with one command: preflight, every run, the assessments, the comparisons.

  python -m playground study STUDY.json [--out DIR] [--codex-auth PATH] [--claude-token PATH]
      [--fake-reviewers] [--preflight] [--keep-containers]

The study file (schema 1) names the case, the stations, the arms, the number of pairs and
the parameters both arms share; `expand` turns it into one schema-1 manifest per run. Pair
p runs `firstArm` first when p is odd and the other arm first when p is even. Run ids are
`<id>-p<p>-conv` and `<id>-p<p>-mkt`.

Order: the preflight runs every cheap check and prints every problem at once (exit 3);
then it builds the image and the Markitect binary once (`--preflight` stops before
that). The runs follow in schedule order, one after the other; then every run whose
container finished is assessed, and every pair with both assessments is compared
(A = conventional, B = markitect; a fairness mismatch fails the step, it is never
allowed). A failed or stopped run does not stop the study; the study stops when the host
could not run a container or the image changed under it, and lists the `assess` and
`compare` commands that finish the runs it completed by hand. There is no resume.

Logins: a private folder `~/.markitect-playground/logins/<id>-<random>/` holds the
study's working copy (`current/`), copied once from the source, and one folder of 0600
copies per run and assessment, mounted read-only. Login files are only copied and
stat'ed on the host, never read, printed or hashed. After a container stopped, its Codex
login is streamed out (`docker cp CONTAINER:PATH -`); exactly one regular file of 1 B to
64 KiB becomes the next step's copy, anything else is rejected unread. The agent controls
that file, so nothing copied out of a container is ever written to the source login;
when a step reported a refresh, the study says at the end to run `codex login`. SIGTERM
and SIGHUP count as an interrupt, and the folder is removed when the study ends.
"""
from __future__ import annotations

import argparse
import json
import os
import platform
import secrets
import shlex
import shutil
import signal
import stat
import subprocess
import sys
import tarfile
import tempfile
import time
import traceback
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from . import compare, evaluate, host, reviewers
from . import manifest as manifest_module

SCHEMA = 1
MIN_PYTHON = (3, 11)
ARM_SUFFIX = {"conventional": "conv", "markitect": "mkt"}
MIN_FREE_BYTES = 5 << 30
LOGIN_MAX_BYTES = reviewers.LOGIN_MAX_BYTES
AGENT_LOGIN = "/home/agent/.codex/auth.json"  # codex_agent.AGENT_HOME/.codex in the run container
STATE_HOME = Path.home() / ".markitect-playground"  # logins/ and study.lock
CODEX_SOURCE = Path.home() / ".codex" / "auth.json"
CLAUDE_SOURCE = evaluate.DEFAULT_CLAUDE_TOKEN
RETURNED = "returned-codex-auth.json"
# Host statuses that mean the host could not run a container: the study stops.
HOST_FAILURES = ("setup-failed", "start-failed", "wait-failed", "host-timeout", "host-interrupted")
PROBE_TIMEOUT = 120

# The one table of the study's exit codes (PLAY-05 extends it); --help prints it.
EXIT_OK, EXIT_FAILED, EXIT_ERROR, EXIT_PREFLIGHT, EXIT_TIMEOUT, EXIT_INTERRUPTED = 0, 1, 2, 3, 124, 130
EXIT_CODES = {
    EXIT_OK: "every step was written and every run exited 0",
    EXIT_FAILED: "a step failed, a run stopped early, or the study stopped (the host could not run a "
                 "container, or the image changed)",
    EXIT_ERROR: "invalid study file, or an error of the study itself (study-error.txt)",
    EXIT_PREFLIGHT: "preflight failed; every problem is printed",
    EXIT_TIMEOUT: "a container hit the host safety timeout; the study stopped",
    EXIT_INTERRUPTED: "interrupted; the study stopped",
}


class StudyError(manifest_module.ManifestError):
    """The study file is missing, unreadable or does not match schema 1."""


def _now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds")


def _write_json(path: Path, value: Any) -> None:
    path.write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n", encoding="utf-8", newline="\n")


def _read_json(path: Path) -> Any:
    try:
        return json.loads(Path(path).read_text(encoding="utf-8"))
    except (OSError, UnicodeError, ValueError):
        return None


# --- study file ------------------------------------------------------------------------

def load(path: Path, *, playground: Path | None = None) -> dict:
    path = Path(path)
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError) as exc:
        raise StudyError(f"cannot read study file {path}: {exc}") from exc
    except json.JSONDecodeError as exc:
        raise StudyError(f"study file {path} is not valid JSON: {exc}") from exc
    return validate(data, base=path.parent, playground=playground)


def validate(data: Any, *, base: Path | None = None, playground: Path | None = None) -> dict:
    """A normalized copy of a schema-1 study file. A relative or `~` markitect.sourceRepo
    counts from `base` (the study file's folder). Every run manifest it expands to must
    pass `manifest.validate`; their normalized shared parts are returned."""
    try:
        return _validate(data, base=base, playground=playground)
    except StudyError:
        raise
    except manifest_module.ManifestError as exc:
        raise StudyError(str(exc)) from exc


def _validate(data: Any, *, base: Path | None, playground: Path | None) -> dict:
    root = manifest_module._object(
        data, "study", required={"schema", "id", "case", "arms", "firstArm", "agent", "limits", "container",
                                 "reviewers"},
        optional={"stations", "pairs", "markitect"})
    if type(root["schema"]) is not int or root["schema"] != SCHEMA:
        raise StudyError(f"schema: expected {SCHEMA}, got {root['schema']!r}")
    study_id = manifest_module._match(root["id"], manifest_module._ID, "id")
    arms = root["arms"]
    if (not isinstance(arms, list) or not arms or len(set(map(str, arms))) != len(arms)
            or any(arm not in manifest_module.METHODS for arm in arms)):
        raise StudyError(f"arms: expected a list of distinct methods ({', '.join(manifest_module.METHODS)}), "
                         f"got {arms!r}")
    arms = [arm for arm in manifest_module.METHODS if arm in arms]
    first = root["firstArm"]
    if first not in arms:
        raise StudyError(f"firstArm: expected one of the arms ({', '.join(arms)}), got {first!r}")
    pairs = manifest_module._positive_int(root.get("pairs", 1), "pairs")
    longest = f"{study_id}-p{pairs}-conv"
    if len(longest) > 63:
        raise StudyError(f"id: too long for its run ids ({longest!r} has {len(longest)} characters, at most 63)")
    names = root["reviewers"]
    if (not isinstance(names, list) or len(set(map(str, names))) != len(names)
            or any(name not in reviewers.PROVIDERS for name in names)):
        raise StudyError(f"reviewers: expected a list of distinct reviewers ({', '.join(reviewers.PROVIDERS)}) "
                         f"or [], got {names!r}")
    config = evaluate.load_config(Path(playground or host.ROOT) / "evaluation" / "config.json")
    missing = [name for name in names if name not in config["reviewers"]]
    if missing:
        raise StudyError(f"reviewers: {', '.join(missing)} not in evaluation/config.json, which names the "
                         "pre-registered reviewer models")
    if "markitect" in arms and "markitect" not in root:
        raise StudyError("markitect: required when the arms include markitect")
    if "markitect" not in arms and "markitect" in root:
        raise StudyError("markitect: only allowed when the arms include markitect")
    product = None
    if "markitect" in root:
        product = dict(manifest_module._object(root["markitect"], "markitect", required={"commit"},
                                               optional={"sourceRepo", "innerModel", "innerEffort"}))
        source = product.get("sourceRepo")
        if isinstance(source, str) and source.strip() and base is not None:
            source = Path(source).expanduser()
            product["sourceRepo"] = str(source if source.is_absolute() else Path(base) / source)
    shared = {key: root[key] for key in ("case", "agent", "limits", "container") if key in root}
    if "stations" in root:
        shared["stations"] = root["stations"]
    normalized = {}
    for arm in arms:
        draft = {"schema": manifest_module.SCHEMA, "id": f"{study_id}-p1-{ARM_SUFFIX[arm]}", "method": arm, **shared}
        if arm == "markitect":
            draft["markitect"] = product
        normalized[arm] = manifest_module.validate(draft, playground=playground)
    one = normalized[arms[0]]
    study = {"schema": SCHEMA, "id": study_id, "case": one["case"], "stations": one["stations"], "arms": arms,
             "firstArm": first, "pairs": pairs, "agent": one["agent"], "limits": one["limits"],
             "container": one["container"], "reviewers": [name for name in reviewers.PROVIDERS if name in names]}
    if "markitect" in normalized:
        study["markitect"] = normalized["markitect"]["markitect"]
    return study


def expand(study: dict, *, playground: Path | None = None) -> list[dict]:
    """The runs in schedule order: {"id", "pair", "arm", "position", "manifest"}; pair p
    runs firstArm first when p is odd, the other arm first when p is even."""
    arms, first = study["arms"], study["firstArm"]
    other = [arm for arm in arms if arm != first]
    runs = []
    for pair in range(1, study["pairs"] + 1):
        order = [first, *other] if pair % 2 else [*other, first]
        for position, arm in enumerate(order, 1):
            run_id = f"{study['id']}-p{pair}-{ARM_SUFFIX[arm]}"
            draft = {"schema": manifest_module.SCHEMA, "id": run_id, "case": study["case"],
                     "stations": study["stations"], "method": arm, "agent": study["agent"],
                     "limits": study["limits"], "container": study["container"]}
            if arm == "markitect":
                draft["markitect"] = study["markitect"]
            runs.append({"id": run_id, "pair": pair, "arm": arm, "position": position,
                         "manifest": manifest_module.validate(draft, playground=playground)})
    return runs


# --- logins ----------------------------------------------------------------------------

def run_logins(manifest: dict, *, codex_given: bool, claude_given: bool) -> tuple[bool, bool]:
    """(Codex login, Claude token) a run gets. The fakes take one only when it was given."""
    kind = manifest["agent"]["kind"]
    codex = host.needs_codex_auth(manifest) or (kind == "fake" and codex_given)
    claude = kind == "claude" or (kind == "fake-claude" and claude_given)
    return codex, claude


def login_needs(study: dict, runs: list[dict], *, fake_reviewers: bool, codex_given: bool,
                claude_given: bool) -> dict[str, list[str]]:
    """What needs each login, for the preflight's messages."""
    needs: dict[str, list[str]] = {"codex": [], "claude": []}
    kind = study["agent"]["kind"]
    for arm in study["arms"]:
        manifest = next(entry["manifest"] for entry in runs if entry["arm"] == arm)
        codex, claude = run_logins(manifest, codex_given=codex_given, claude_given=claude_given)
        if codex:
            needs["codex"].append(f"the {arm} arm (agent kind {kind}"
                                  + (", Markitect's inner roles" if arm == "markitect" else "") + ")")
        if claude:
            needs["claude"].append(f"the {arm} arm (agent kind {kind})")
    if not fake_reviewers:
        for name in study["reviewers"]:
            needs[name].append(f"the {name} reviewer")
    return needs


def _copy_private(source: Path, target: Path) -> None:
    """Copy bytes only (never interpreted) into a new file readable by its owner only."""
    fd = os.open(target, os.O_WRONLY | os.O_CREAT | os.O_EXCL | getattr(os, "O_BINARY", 0), 0o600)
    with os.fdopen(fd, "wb") as out, open(source, "rb") as inp:
        shutil.copyfileobj(inp, out)
    os.chmod(target, 0o600)


def _private_dir(path: Path) -> Path:
    path.mkdir(mode=0o700, parents=True, exist_ok=True)
    os.chmod(path, 0o700)
    return path


class Logins:
    """The study's private login copies; on the host they are only copied and stat'ed."""

    def __init__(self, study_id: str, *, codex: Path | None, claude: Path | None, home: Path | None = None) -> None:
        base = _private_dir(Path(home or STATE_HOME) / "logins")
        self.root = Path(tempfile.mkdtemp(prefix=f"{study_id}-", dir=base))
        try:
            os.chmod(self.root, 0o700)
            self.current = _private_dir(self.root / "current")
            self.codex_source = Path(codex).resolve() if codex else None
            self.claude_source = Path(claude).resolve() if claude else None
            self.codex = self.claude = None
            self.promotions = 0
            self.record: dict[str, Any] = {"folder": str(self.root), "codexSource": None, "claudeSource": None,
                                           "promotions": 0, "removed": False}
            if self.codex_source is not None:
                self.codex = self.current / "codex-auth.json"
                _copy_private(self.codex_source, self.codex)
                self.record["codexSource"] = str(self.codex_source)
            if self.claude_source is not None:  # tokens do not refresh: copied, never copied back
                self.claude = self.current / "claude-token"
                _copy_private(self.claude_source, self.claude)
                self.record["claudeSource"] = str(self.claude_source)
        except BaseException:  # no copy outlives a failed or interrupted setup
            reviewers.remove_tree(self.root)
            raise

    def step(self, name: str, *, codex: bool, claude: bool) -> tuple[Path, Path | None, Path | None]:
        """A folder with this step's own 0600 copies of the working copies."""
        folder = _private_dir(self.root / name)
        auth = token = None
        if codex and self.codex is not None:
            auth = folder / "codex-auth.json"
            _copy_private(self.codex, auth)
        if claude and self.claude is not None:
            token = folder / "claude-token"
            _copy_private(self.claude, token)
        return folder, auth, token

    def copy_out(self, container: str, source: str, folder: Path) -> str:
        """Stream a stopped container's Codex login out (`docker cp CONTAINER:PATH -`) and
        keep it next to the step's copies only if the tar stream holds exactly one regular
        file of 1 B to 64 KiB; anything else is rejected without reading further, so
        nothing larger than the cap reaches the disk."""
        target = folder / RETURNED
        try:
            process = subprocess.Popen(["docker", "cp", f"{container}:{source}", "-"], stdin=subprocess.DEVNULL,
                                       stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        except OSError as exc:
            return f"failed: {exc}"
        verdict = "failed: no answer"
        try:
            try:
                verdict = _take_one_file(process.stdout, target)
            except (tarfile.TarError, EOFError, OSError) as exc:
                verdict = f"rejected: not a readable tar stream ({type(exc).__name__})"
            if verdict != "copied" and process.poll() is None:
                process.kill()  # stop reading: the rest of the stream is never consumed
            try:
                _out, err = process.communicate(timeout=PROBE_TIMEOUT)
            except subprocess.TimeoutExpired:
                process.kill()
                _out, err = process.communicate()
            if process.returncode != 0 and not verdict.startswith("rejected"):
                text = (err or b"").decode("utf-8", errors="replace").strip()[:300]
                verdict = f"failed: {text or f'docker cp exited {process.returncode}'}"
            elif process.returncode != 0 and verdict.startswith("rejected: not a readable"):
                text = (err or b"").decode("utf-8", errors="replace").strip()[:300]
                verdict = f"failed: {text}" if text else verdict
        finally:
            if verdict != "copied":
                target.unlink(missing_ok=True)
            if process.poll() is None:
                process.kill()
        return verdict

    def promote(self, folder: Path) -> str:
        """Make a copied-out login the working copy, whatever the step reported (an agent
        can fake a change, and a crashed step may never report a real one)."""
        returned = folder / RETURNED
        try:
            info = os.lstat(returned)
        except OSError:
            return "nothing copied out; kept the working copy"
        if not stat.S_ISREG(info.st_mode) or not 0 < info.st_size <= LOGIN_MAX_BYTES:
            return "not a regular file of 1 B to 64 KiB; kept the working copy"
        temporary = self.current / f".codex-auth.{secrets.token_hex(4)}"
        _copy_private(returned, temporary)
        os.replace(temporary, self.codex)
        self.promotions += 1
        self.record["promotions"] = self.promotions
        return "promoted"

    def finish_step(self, folder: Path) -> None:
        reviewers.remove_tree(folder)

    def close(self) -> None:
        reviewers.remove_tree(self.root)
        self.record["removed"] = not self.root.exists()


def _take_one_file(stream, target: Path) -> str:
    """Write the only member of an uncompressed tar stream to `target` when it is one
    regular file of 1 B to 64 KiB; otherwise a "rejected: ..." reason, with nothing written."""
    with tarfile.open(fileobj=stream, mode="r|") as tar:
        member = tar.next()
        if member is None:
            return "rejected: the copy holds no file"
        if not member.isreg():
            kind = ("a directory" if member.isdir() else "a link" if member.issym() or member.islnk()
                    else "a device or special file")
            return f"rejected: the copy is {kind}, not a regular file"
        if not 0 < member.size <= LOGIN_MAX_BYTES:
            return f"rejected: {member.size} bytes, not 1 B to 64 KiB"
        handle = tar.extractfile(member)
        data = handle.read(LOGIN_MAX_BYTES + 1) if handle is not None else b""
        if len(data) != member.size:
            return "rejected: the copy is truncated"
        if tar.next() is not None:
            return "rejected: the copy holds more than one entry"
    fd = os.open(target, os.O_WRONLY | os.O_CREAT | os.O_EXCL | getattr(os, "O_BINARY", 0), 0o600)
    with os.fdopen(fd, "wb") as out:
        out.write(data)
    return "copied"


def leftover_login_folders(home: Path | None = None) -> list[str]:
    base = Path(home or STATE_HOME) / "logins"
    try:
        return sorted(path.name for path in base.iterdir())
    except OSError:
        return []


# --- preflight -------------------------------------------------------------------------

def _probe(cmd: list[str], cwd: Path | None = None) -> tuple[int | None, str, str]:
    """(exit code, stdout, stderr); exit None when the command could not run."""
    try:
        done = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, encoding="utf-8", errors="replace",
                              timeout=PROBE_TIMEOUT, stdin=subprocess.DEVNULL)
    except FileNotFoundError:
        return None, "", f"{cmd[0]} is not installed"
    except (OSError, subprocess.SubprocessError) as exc:
        return None, "", str(exc)
    return done.returncode, done.stdout.strip(), done.stderr.strip()


def _first_line(text: str) -> str:
    return (text.strip().splitlines() or [""])[0][:300]


def _existing_parent(path: Path) -> Path:
    for folder in (path, *path.parents):
        if folder.exists():
            return folder
    return Path(path.anchor or ".")


def preflight(study: dict, runs: list[dict], out: Path, *, codex_source: Path, claude_source: Path,
              needs: dict[str, list[str]], codex_flag: str, claude_flag: str) -> tuple[list[dict], dict]:
    """Every cheap check: (checks, facts). A check is {"check", "status": ok|fail|warn|skip,
    "message"}; every failing message names its fix."""
    checks: list[dict] = []
    facts: dict[str, Any] = {"python": platform.python_version(), "hostPlatform": host.host_platform(),
                             "docker": None, "go": None, "markitect": None}

    def add(name: str, status: str, message: str) -> None:
        checks.append({"check": name, "status": status, "message": message})

    if sys.version_info[:2] >= MIN_PYTHON:
        add("python", "ok", f"Python {platform.python_version()}")
    else:
        add("python", "fail", f"Python {'.'.join(map(str, MIN_PYTHON))} or newer is needed, this is "
                              f"{platform.python_version()}; run the study with a newer python3")
    warning = host.platform_warning()
    add("host platform", "warn" if warning else "ok", warning or "Linux, the reference platform")

    markitect = "markitect" in study["arms"]
    if shutil.which("docker") is None:
        add("docker", "fail", "docker is not on PATH; install Docker and make sure your user may run it")
        for name in ("docker daemon", "linux engine", "running containers", "container names"):
            add(name, "skip", "needs docker")
    else:
        add("docker", "ok", "docker on PATH")
        code, text, err = _probe(["docker", "version", "--format",
                                  "{{.Server.Version}}|{{.Server.Os}}|{{.Server.Arch}}"])
        parts = text.split("|") if code == 0 else []
        if len(parts) != 3:
            add("docker daemon", "fail", f"the Docker daemon is not reachable ({_first_line(err or text) or 'no answer'}); "
                                         "start Docker, or check that your user may run docker")
            for name in ("linux engine", "running containers", "container names"):
                add(name, "skip", "needs the Docker daemon")
        else:
            facts["docker"] = {"version": parts[0], "os": parts[1], "arch": parts[2]}
            add("docker daemon", "ok", f"Docker {parts[0]} reachable")
            if parts[1] != "linux":
                add("linux engine", "fail", f"Docker runs a {parts[1] or 'unknown'} engine; switch Docker to "
                                            "Linux containers")
            elif markitect and parts[2] not in ("amd64", "x86_64"):
                add("linux engine", "fail", f"the Markitect arm needs a linux/amd64 engine (its binary is built "
                                            f"for linux/amd64 only); this engine is linux/{parts[2]}")
            else:
                add("linux engine", "ok", f"linux/{parts[2]} engine")
            code, text, err = _probe(["docker", "ps", "--filter", f"label={host.LABEL}", "--format", "{{.Names}}"])
            running = text.split() if code == 0 else []
            if code != 0:
                add("running containers", "fail", f"could not list containers ({_first_line(err)}); check Docker")
            elif running:
                add("running containers", "fail", f"playground containers are running ({', '.join(running)}); "
                                                  "run one study at a time: wait for them to end, or remove them "
                                                  "with `docker rm -f NAME`")
            else:
                add("running containers", "ok", "no playground container running")
            code, text, err = _probe(["docker", "ps", "--all", "--format", "{{.Names}}"])
            expected = [f"mpg-{entry['id']}" for entry in runs] + [f"mpg-assess-{entry['id']}" for entry in runs]
            taken = sorted(set(expected) & set(text.split())) if code == 0 else []
            if code != 0:
                add("container names", "fail", f"could not list containers ({_first_line(err)}); check Docker")
            elif taken:
                add("container names", "fail", f"containers with this study's names exist ({', '.join(taken)}); "
                                               f"remove them (`docker rm -f {' '.join(taken)}`) or give the study a "
                                               "new id")
            else:
                add("container names", "ok", "no container uses this study's names")
    lock = Path(STATE_HOME) / "study.lock"
    if lock.exists():
        holder = _read_json(lock) or {}
        add("study lock", "fail", f"another study holds {lock} (study {holder.get('study')!r}, pid {holder.get('pid')}, "
                                  f"since {holder.get('startedAt')}); run one study at a time, and delete the file "
                                  "only if no study is running")
    else:
        add("study lock", "ok", "no other study running")

    if markitect:
        _markitect_checks(study["markitect"], add, facts)

    for kind, label, source, flag, fix in (
            ("codex", "Codex login", codex_source, codex_flag, "run `codex login` or pass --codex-auth PATH"),
            ("claude", "Claude Code token", claude_source, claude_flag,
             "create it once with `claude setup-token` and save the token alone in that file, or pass "
             "--claude-token PATH")):
        if not needs[kind]:
            continue
        who = f"needed by {', '.join(needs[kind])}"
        try:  # stat only; the file is never opened here
            size = source.stat().st_size if source.is_file() else None
        except OSError:
            size = None
        if size is None:
            add(f"{kind} login", "fail", f"{label} missing: {source} ({who}); {fix}")
        elif size == 0:
            add(f"{kind} login", "fail", f"{label} is empty: {source} ({who}); {fix}")
        else:
            add(f"{kind} login", "ok", f"{label} {source}{flag} ({who})")

    if out.exists() or out.is_symlink():
        add("output folder", "fail", f"output folder exists: {out}; remove it or choose another --out")
    elif host._inside_git_checkout(out):
        add("output folder", "fail", f"output folder {out} is inside a Git checkout; choose a folder outside every "
                                     "checkout (run outputs are never committed)")
    else:
        add("output folder", "ok", f"{out} is new and outside a Git checkout")
    parent = _existing_parent(out)
    try:
        free = shutil.disk_usage(parent).free
    except OSError as exc:
        add("free disk", "fail", f"cannot read the free space at {parent} ({exc}); check the --out folder")
    else:
        if free < MIN_FREE_BYTES:
            add("free disk", "fail", f"only {free / (1 << 30):.1f} GiB free at {parent}; a study needs "
                                     f"{MIN_FREE_BYTES >> 30} GiB: free space or choose another --out")
        else:
            add("free disk", "ok", f"{free / (1 << 30):.0f} GiB free at {parent}")

    code, text, _err = _probe(["git", "-C", str(host.ROOT), "status", "--porcelain", "--untracked-files=all", "--",
                               "evaluation"])
    if code == 0 and text:
        add("evaluation files", "warn", "evaluation files have uncommitted changes; they are pre-registered: commit "
                                        "them before the runs they judge")
    elif code == 0:
        add("evaluation files", "ok", "evaluation files committed")
    leftovers = leftover_login_folders()
    if leftovers:
        add("login folders", "warn", f"login folders of an earlier study are left in {Path(STATE_HOME) / 'logins'} "
                                     f"({', '.join(leftovers)}); delete them if no study is running")
    return checks, facts


def _markitect_checks(product: dict, add, facts: dict) -> None:
    """git and go on PATH, sourceRepo a checkout, the commit present (resolved into facts)."""
    source = product["sourceRepo"]
    git = shutil.which("git") is not None
    add("git", "ok" if git else "fail", "git on PATH" if git else
        "git is not on PATH; install Git (the Markitect arm builds from a checkout)")
    if shutil.which("go") is None:
        add("go", "fail", "go is not on PATH; install Go (the Markitect binary is built on the host; the "
                          "commit's go.mod chooses the toolchain)")
    else:
        code, text, _err = _probe(["go", "version"])
        facts["go"] = text.split()[2] if code == 0 and len(text.split()) > 2 else None
        add("go", "ok", f"go on PATH ({facts['go'] or 'version unknown'})")
    if not git:
        add("markitect checkout", "skip", "needs git")
        return
    code, _text, err = _probe(["git", "-C", source, "rev-parse", "--git-dir"])
    if code != 0 or not Path(source).is_dir():
        add("markitect checkout", "fail", f"markitect.sourceRepo {source} is not a Git checkout "
                                          f"({_first_line(err) or 'no folder'}); set it to a Markitect checkout")
        return
    add("markitect checkout", "ok", f"{source} is a Git checkout")
    try:
        resolved = host.resolve_markitect(product)
    except host.HostError:
        add("markitect commit", "fail", f"markitect.commit {product['commit']} is not a commit in {source}; fetch it "
                                        f"(`git -C {source} fetch`) or fix the commit")
        return
    facts["markitect"] = resolved
    add("markitect commit", "ok", f"markitect.commit {resolved['commit'][:12]}")
    code, text, _err = _probe(["git", "-C", source, "rev-list", "--count", f"{resolved['commit']}..HEAD"])
    if code == 0 and text.isdigit() and int(text) > 0:
        add("markitect commit age", "warn", f"markitect.commit {resolved['commit'][:12]} is {text} commit(s) behind "
                                            f"HEAD of {source}; check that this is the product you mean to test")


def print_checks(checks: list[dict]) -> list[str]:
    labels = {"ok": "[ok]  ", "fail": "[FAIL]", "warn": "[warn]", "skip": "[skip]"}
    lines = [f"  {labels[check['status']]} {check['check']}: {check['message']}" for check in checks]
    print("preflight:", flush=True)
    for line in lines:
        print(line, flush=True)
    return lines


# --- the study -------------------------------------------------------------------------

def _acquire_lock(path: Path, value: dict) -> bool:
    _private_dir(path.parent)
    try:
        fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    except FileExistsError:
        return False
    with os.fdopen(fd, "w", encoding="utf-8") as handle:
        json.dump(value, handle)
    return True


class Study:
    """One study after a passed cheap preflight: owns `out`, study.json and the logins."""

    def __init__(self, study: dict, runs: list[dict], args: argparse.Namespace, *, out: Path, study_file: Path,
                 checks: list[dict], facts: dict, lines: list[str], codex_source: Path | None,
                 claude_source: Path | None) -> None:
        self.study, self.runs, self.args, self.out = study, runs, args, out
        self.study_file, self.facts, self.lines = study_file, facts, lines
        self.codex_source, self.claude_source = codex_source, claude_source
        self.fake = bool(args.fake_reviewers and study["reviewers"])
        self.logins: Logins | None = None
        self.prebuilt: dict | None = None
        self.stopped: str | None = None
        self.stop_code = EXIT_FAILED
        self.clock: dict[int, float] = {}
        parameters = dict(study)
        if facts.get("markitect"):
            parameters["markitect"] = facts["markitect"]
        self.record: dict[str, Any] = {
            "schema": SCHEMA, "kind": "study", "id": study["id"], "status": "running", "exitCode": None,
            "startedAt": _now(), "endedAt": None, "stopReason": None,
            "studyFile": {"path": str(study_file.resolve()), "copy": "study-file.json"},
            "parameters": parameters,
            "options": {"out": str(out), "fakeReviewers": self.fake, "keepContainers": bool(args.keep_containers)},
            "logins": {"codex": str(codex_source) if codex_source else None,
                       "claude": str(claude_source) if claude_source else None, "copies": None,
                       "refreshReported": False, "note": None},
            "versions": {"playground": evaluate.evaluation_identity(), "python": facts["python"],
                         "hostPlatform": facts["hostPlatform"], "docker": facts["docker"], "go": facts["go"],
                         "image": None, "markitect": None},
            "preflight": {"status": "passed", "checks": checks},
            "schedule": [{"id": entry["id"], "pair": entry["pair"], "arm": entry["arm"],
                          "position": entry["position"], "manifest": f"manifests/{entry['id']}.json"}
                         for entry in runs],
            "steps": [],
            "finishByHand": None,
        }

    # -- records

    def save(self) -> None:
        _write_json(self.out / "study.json", self.record)
        (self.out / "study.md").write_text(render(self.record), encoding="utf-8", newline="\n")

    def _step(self, kind: str, **fields: Any) -> dict:
        step = {"step": kind, **fields, "status": "running", "exitCode": None, "startedAt": _now(), "endedAt": None,
                "seconds": None, "error": None}
        self.record["steps"].append(step)
        self.clock[id(step)] = time.monotonic()
        self.save()
        return step

    def _end(self, step: dict, status: str, exit_code: int | None = None, **fields: Any) -> None:
        step.update(status=status, exitCode=exit_code, endedAt=_now(), **fields)
        step["seconds"] = round(time.monotonic() - self.clock.pop(id(step)), 1)
        self.save()

    def _stop(self, reason: str, code: int = EXIT_FAILED) -> None:
        if self.stopped is None:
            self.stopped, self.stop_code = reason, code
            self.record["stopReason"] = reason
            print(f"study stops: {reason}", file=sys.stderr, flush=True)

    # -- phases

    def execute(self) -> int:
        lock = Path(STATE_HOME) / "study.lock"
        if not _acquire_lock(lock, {"study": self.study["id"], "pid": os.getpid(), "out": str(self.out),
                                    "startedAt": _now()}):
            print(f"error: another study holds {lock}; run one study at a time", file=sys.stderr)
            return EXIT_PREFLIGHT
        try:
            return self._execute()
        finally:
            lock.unlink(missing_ok=True)

    def _execute(self) -> int:
        self.out.mkdir(parents=True)
        (self.out / "preflight").mkdir()
        (self.out / "preflight" / "preflight.txt").write_text("\n".join(self.lines) + "\n", encoding="utf-8",
                                                               newline="\n")
        shutil.copyfile(self.study_file, self.out / "study-file.json")
        self.save()
        code = EXIT_ERROR
        try:
            if not self.build():
                code = EXIT_PREFLIGHT
                self.record["status"] = "preflight-failed"
                return code
            self.write_manifests()
            self.logins = Logins(self.study["id"], codex=self.codex_source, claude=self.claude_source)
            self.record["logins"]["copies"] = self.logins.record
            self.save()
            run_steps = [self.run_one(entry) for entry in self.runs]
            assess_steps = {entry["id"]: self.assess_one(entry, step) for entry, step in zip(self.runs, run_steps)}
            self.compare_pairs(assess_steps)
            code = self.exit_code()
            self.record["status"] = {EXIT_OK: "completed", EXIT_TIMEOUT: "timeout",
                                     EXIT_INTERRUPTED: "interrupted"}.get(code, "stopped" if self.stopped else "failed")
            return code
        except KeyboardInterrupt:
            self.record["status"], self.record["stopReason"] = "interrupted", "interrupted"
            code = EXIT_INTERRUPTED
            return code
        except Exception:
            (self.out / "study-error.txt").write_text(traceback.format_exc(), encoding="utf-8")
            print(traceback.format_exc(), file=sys.stderr)
            self.record["status"] = "error"
            code = EXIT_ERROR
            return code
        finally:
            try:
                if self.logins is not None:
                    self.logins.close()
            finally:
                logins = self.record["logins"]
                if logins["refreshReported"] and logins["codex"]:
                    logins["note"] = (f"Codex refreshed its login during the study; your {logins['codex']} may be "
                                      "used up: run `codex login` before the next run.")
                    print(logins["note"], file=sys.stderr)
                if self.stopped or self.record["status"] == "interrupted":
                    self.finish_by_hand()
                self.record["exitCode"] = code
                self.record["endedAt"] = _now()
                self.save()
                print(f"study: {self.out / 'study.md'} (exit {code})", flush=True)

    def finish_by_hand(self) -> None:
        """After a stop: the `assess` and `compare` commands that finish the completed runs."""
        steps = self.record["steps"]
        finished = [s["id"] for s in steps if s["step"] == "run" and s["status"] == "completed"]
        assessed = {s["id"] for s in steps if s["step"] == "assess" and s["exitCode"] == 0}
        compared = {s["pair"] for s in steps if s["step"] == "compare" and s["status"] == "written"}
        names = self.study["reviewers"]
        commands = []
        for run_id in finished:
            if run_id in assessed:
                continue
            command = ["python3", "-m", "playground", "assess", "--run", str(self.out / "runs" / run_id),
                       "--reviewers", ",".join(names) or "none"]
            if self.fake:
                command.append("--fake-reviewers")
            else:
                if "codex" in names and self.args.codex_auth:
                    command += ["--codex-auth", str(Path(self.args.codex_auth).expanduser().resolve())]
                if "claude" in names and self.args.claude_token:
                    command += ["--claude-token", str(Path(self.args.claude_token).expanduser().resolve())]
            commands.append(shlex.join(command))
        if len(self.study["arms"]) == 2:
            for pair in range(1, self.study["pairs"] + 1):
                ids = {entry["arm"]: entry["id"] for entry in self.runs if entry["pair"] == pair}
                if pair in compared or not all(ids[arm] in finished for arm in ids):
                    continue
                commands.append(shlex.join(["python3", "-m", "playground", "compare",
                                            str(self.out / "runs" / ids["conventional"]),
                                            str(self.out / "runs" / ids["markitect"]),
                                            "--out", str(self.out / "comparisons" / f"p{pair}.md")]))
        if not commands:
            return
        self.record["finishByHand"] = {"cwd": str(host.ROOT), "commands": commands}
        print(f"finish by hand, from {host.ROOT}:", file=sys.stderr)
        for command in commands:
            print(f"  {command}", file=sys.stderr)

    def build(self) -> bool:
        """The image and the Markitect binary, once for every run (preflight checks)."""
        checks = self.record["preflight"]["checks"]
        manifest = self.runs[0]["manifest"]
        tag = host.image_tag(manifest)
        try:
            print(f"building image {tag} ...", flush=True)
            image_id = host.build_image(manifest, self.out / "preflight" / "image-build.log")
        except (host.HostError, OSError) as exc:
            checks.append({"check": "image build", "status": "fail", "message": f"{exc}; fix the build and run again"})
            self.record["preflight"]["status"] = "failed"
            print(f"  [FAIL] image build: {exc}", file=sys.stderr)
            return False
        checks.append({"check": "image build", "status": "ok", "message": f"{tag} {image_id}"})
        self.prebuilt = {"image": {"tag": tag, "id": image_id}}
        self.record["versions"]["image"] = {"tag": tag, "id": image_id}
        if "markitect" in self.study["arms"]:
            binary = self.out / "preflight" / "bin" / "markitect"
            try:
                print("building markitect binary ...", flush=True)
                built = host.build_markitect(self.facts["markitect"], binary)
            except (host.HostError, OSError) as exc:
                checks.append({"check": "markitect binary", "status": "fail",
                               "message": f"{exc}; check the commit and the Go toolchain"})
                self.record["preflight"]["status"] = "failed"
                print(f"  [FAIL] markitect binary: {exc}", file=sys.stderr)
                return False
            checks.append({"check": "markitect binary", "status": "ok", "message": f"sha256 {built['sha256']}"})
            self.prebuilt.update(markitect=built, binary=binary)
            self.record["versions"]["markitect"] = built
        self.save()
        return True

    def write_manifests(self) -> None:
        folder = self.out / "manifests"
        folder.mkdir()
        for entry in self.runs:
            if entry["arm"] == "markitect":
                entry["manifest"] = {**entry["manifest"], "markitect": self.facts["markitect"]}
            _write_json(folder / f"{entry['id']}.json", entry["manifest"])

    def run_one(self, entry: dict) -> dict:
        step = self._step("run", id=entry["id"], pair=entry["pair"], arm=entry["arm"], folder=f"runs/{entry['id']}")
        if self.stopped:
            self._end(step, "not-run", error=f"study stopped: {self.stopped}")
            return step
        print(f"== run {entry['id']} (pair {entry['pair']}, {entry['arm']})", flush=True)
        codex, claude = run_logins(entry["manifest"], codex_given=bool(self.args.codex_auth),
                                   claude_given=bool(self.args.claude_token))
        folder, auth, token = self.logins.step(entry["id"], codex=codex, claude=claude)
        login = step["login"] = {"codex": auth is not None, "claude": token is not None, "copyOut": None,
                                 "reported": None, "promotion": None}
        seen: dict[str, str | None] = {"image": None}

        def before_remove(container: str) -> None:
            code, text, _err = _probe(["docker", "container", "inspect", "--format", "{{.Image}}", container])
            seen["image"] = text if code == 0 and text else None
            if auth is not None:
                login["copyOut"] = self.logins.copy_out(container, AGENT_LOGIN, folder)

        run_dir = self.out / "runs" / entry["id"]
        record: dict = {}
        code = None
        try:
            code, record = host.run_manifest(entry["manifest"], run_dir, auth=auth, token=token,
                                             keep=self.args.keep_containers, prebuilt=self.prebuilt,
                                             before_remove=before_remove)
        finally:
            if auth is not None:
                login["reported"] = (_read_json(run_dir / "results" / "runner.json") or {}).get("codexLoginChanged")
                self.record["logins"]["refreshReported"] |= login["reported"] is True
                login["promotion"] = self.logins.promote(folder)
            self.logins.finish_step(folder)
        status = record["status"]
        self._end(step, status, code, containerExitCode=record.get("containerExitCode"), imageId=seen["image"],
                  error=record.get("error"))
        if status in HOST_FAILURES:
            self._stop(f"run {entry['id']}: host status {status}" + (f" ({record['error']})" if record.get("error")
                                                                    else ""),
                       {124: EXIT_TIMEOUT, 130: EXIT_INTERRUPTED}.get(code, EXIT_FAILED))
        elif seen["image"] != self.prebuilt["image"]["id"]:
            self._stop(f"run {entry['id']} ran image {seen['image'] or 'unknown'}, not the preflight build "
                       f"{self.prebuilt['image']['id']}; rebuild and start a new study")
        return step

    def assess_one(self, entry: dict, run_step: dict) -> dict:
        step = self._step("assess", id=entry["id"], pair=entry["pair"], arm=entry["arm"],
                          folder=f"runs/{entry['id']}/assessment")
        if self.stopped or run_step["status"] != "completed":
            reason = f"study stopped: {self.stopped}" if self.stopped else "the run's container did not finish"
            self._end(step, "not-run", error=reason)
            return step
        print(f"== assess {entry['id']}", flush=True)
        names = self.study["reviewers"]
        codex = "codex" in names and not self.fake
        claude = "claude" in names and not self.fake
        folder, auth, token = self.logins.step(f"assess-{entry['id']}", codex=codex, claude=claude)
        login = step["login"] = {"codex": auth is not None, "claude": token is not None, "copyOut": None,
                                 "reported": None, "promotion": None}

        def before_remove(container: str) -> None:
            if auth is not None:
                login["copyOut"] = self.logins.copy_out(container, evaluate.REVIEWER_LOGIN, folder)

        run_dir = self.out / "runs" / entry["id"]
        try:
            code, result = evaluate.assess_container(run_dir, names=names, codex_auth=auth, claude_token=token,
                                                     fake=self.fake, keep=self.args.keep_containers,
                                                     before_remove=before_remove)
        except (evaluate.AssessError, host.HostError, OSError) as exc:
            self._end(step, "failed", EXIT_ERROR, error=str(exc))
            return step
        finally:
            if auth is not None:
                report = _read_json(run_dir / "assessment" / "report.json") or {}
                login["reported"] = (((report.get("totals") or {}).get("reviewers") or {}).get("codex") or {}).get(
                    "loginRefreshed")
                self.record["logins"]["refreshReported"] |= login["reported"] is True
                login["promotion"] = self.logins.promote(folder)
            self.logins.finish_step(folder)
        status = result["status"]
        self._end(step, status, code, error=result.get("error"))
        if status in HOST_FAILURES:
            self._stop(f"assessment of {entry['id']}: host status {status}",
                       {124: EXIT_TIMEOUT, 130: EXIT_INTERRUPTED}.get(code, EXIT_FAILED))
        return step

    def compare_pairs(self, assess_steps: dict[str, dict]) -> None:
        if len(self.study["arms"]) != 2:
            return
        for pair in range(1, self.study["pairs"] + 1):
            ids = {entry["arm"]: entry["id"] for entry in self.runs if entry["pair"] == pair}
            target = f"comparisons/p{pair}.md"
            step = self._step("compare", pair=pair, a=ids["conventional"], b=ids["markitect"], file=target)
            if self.stopped:
                self._end(step, "not-run", error=f"study stopped: {self.stopped}")
                continue
            if any(assess_steps[ids[arm]]["exitCode"] != 0 for arm in ("conventional", "markitect")):
                self._end(step, "not-run", error="an assessment of this pair is missing or failed")
                continue
            try:
                a = compare.load(self.out / "runs" / ids["conventional"])
                b = compare.load(self.out / "runs" / ids["markitect"])
            except compare.CompareError as exc:
                self._end(step, "failed", EXIT_ERROR, error=str(exc))
                continue
            problems = compare.mismatches(a, b)
            if problems:
                self._end(step, "fairness-mismatch", EXIT_ERROR,
                          error="the runs are not comparable; these fairness fields differ",
                          mismatches=[[key, compare._show(key, value_a), compare._show(key, value_b)]
                                      for key, value_a, value_b in problems])
                continue
            (self.out / "comparisons").mkdir(exist_ok=True)
            (self.out / target).write_text(compare.render(a, b, []), encoding="utf-8", newline="\n")
            self._end(step, "written", 0)

    def exit_code(self) -> int:
        if self.stopped:
            return self.stop_code
        for step in self.record["steps"]:
            if step["exitCode"] != 0:
                return EXIT_FAILED
        return EXIT_OK


# --- study.md --------------------------------------------------------------------------

def _fmt(value: Any) -> str:
    if value is None:
        return "n/a"
    if isinstance(value, bool):
        return "yes" if value else "no"
    if isinstance(value, (dict, list)):
        return json.dumps(value, sort_keys=True)
    return str(value)


def render(record: dict) -> str:
    p, versions = record["parameters"], record["versions"]
    agent = p["agent"]
    lines = [f"# Study {record['id']}", "",
             f"Status **{record['status']}**, exit code {_fmt(record['exitCode'])}"
             + (f"; stopped: {record['stopReason']}" if record.get("stopReason") else "") + ".", "",
             f"- Case {p['case']}, stations {p['stations']}, arms {', '.join(p['arms'])}, first arm {p['firstArm']}, "
             f"pairs {p['pairs']}.",
             f"- Agent {agent['kind']} (model {agent['model']}, effort {agent['effort']}, codex {agent['codexVersion']}, "
             f"claude {agent['claudeVersion']}, max subagents {agent['maxSubagents']}); limits {_fmt(p['limits'])}; "
             f"container {_fmt(p['container'])}.",
             f"- Reviewers {', '.join(p['reviewers']) or 'none'}" + (" (fake)" if record["options"]["fakeReviewers"]
                                                                    else "")
             + "; models from evaluation/config.json.",
             f"- Markitect: {_fmt(p.get('markitect'))}." if p.get("markitect") else "- Markitect: not in this study.",
             f"- Versions: playground {_fmt((versions.get('playground') or {}).get('commit'))}"
             + (" (uncommitted changes)" if (versions.get("playground") or {}).get("dirty") else "")
             + f"; Python {versions.get('python')}; host {_fmt(versions.get('hostPlatform'))}; Docker "
               f"{_fmt(versions.get('docker'))}; Go {_fmt(versions.get('go'))}; image "
               f"{_fmt((versions.get('image') or {}).get('id'))}; binary sha256 "
               f"{_fmt((versions.get('markitect') or {}).get('sha256'))}.",
             f"- Logins (paths only): Codex {_fmt(record['logins']['codex'])}, Claude token "
             f"{_fmt(record['logins']['claude'])}; never written by the study."
             + (f" {record['logins']['note']}" if record["logins"].get("note") else ""), ""]
    finish = record.get("finishByHand")
    if finish:
        lines += ["## Finish by hand", "", f"The study stopped. These commands, from `{finish['cwd']}`, assess and "
                  "compare the runs it completed:", "", "```", *finish["commands"], "```", ""]
    lines += ["## Preflight", ""]
    for check in record["preflight"]["checks"]:
        lines.append(f"- {check['status']}: {check['check']}: {check['message']}")
    lines += ["", "## Steps", "", "| Step | Run | Pair | Arm | Status | Exit | Seconds | Login | Output |",
              "|---|---|---|---|---|---|---|---|---|"]
    for step in record["steps"]:
        login = step.get("login") or {}
        login_text = (f"codex {login.get('promotion') or ('mounted' if login.get('codex') else 'none')}"
                      if login else "-")
        output = step.get("file") or step.get("folder") or "-"
        lines.append(f"| {step['step']} | {step.get('id') or step.get('a', '') + ' vs ' + step.get('b', '')} | "
                     f"{step.get('pair')} | {step.get('arm') or '-'} | {step['status']} | {_fmt(step['exitCode'])} | "
                     f"{_fmt(step['seconds'])} | {login_text} | {output} |")
        if step.get("error"):
            lines.append(f"|  | error: {str(step['error'])[:300]} |  |  |  |  |  |  |  |")
        for key, value_a, value_b in step.get("mismatches") or []:
            lines.append(f"|  | mismatch `{key}`: {value_a} vs {value_b} |  |  |  |  |  |  |  |")
    lines += ["", "Run folders hold each run's report (`results/report.md`) and assessment "
                  "(`assessment/report.md`); comparisons are in `comparisons/`. One pair shows mechanisms, "
                  "not a general effect.", ""]
    return "\n".join(lines)


# --- CLI -------------------------------------------------------------------------------

def run(args: argparse.Namespace) -> int:
    study_file = Path(args.study)
    study = load(study_file)
    runs = expand(study)
    out = Path(args.out).expanduser() if args.out else Path.home() / "markitect-playground-runs" / study["id"]
    out = out.resolve()
    codex_source = Path(args.codex_auth).expanduser() if args.codex_auth else CODEX_SOURCE
    claude_source = Path(args.claude_token).expanduser() if args.claude_token else CLAUDE_SOURCE
    needs = login_needs(study, runs, fake_reviewers=bool(args.fake_reviewers), codex_given=bool(args.codex_auth),
                        claude_given=bool(args.claude_token))
    checks, facts = preflight(study, runs, out, codex_source=codex_source, claude_source=claude_source, needs=needs,
                              codex_flag=" (--codex-auth)" if args.codex_auth else "",
                              claude_flag=" (--claude-token)" if args.claude_token else "")
    lines = print_checks(checks)
    failed = [check for check in checks if check["status"] == "fail"]
    if failed:
        print(f"preflight failed: {len(failed)} problem(s); fix them and run again", file=sys.stderr)
        return EXIT_PREFLIGHT
    if args.preflight:
        print("preflight passed (--preflight: the image and the Markitect binary are built by the study itself)")
        return EXIT_OK
    return Study(study, runs, args, out=out, study_file=study_file, checks=checks, facts=facts, lines=lines,
                 codex_source=codex_source.resolve() if needs["codex"] else None,
                 claude_source=claude_source.resolve() if needs["claude"] else None).execute()


def main(argv: list[str] | None = None) -> int:
    epilog = "exit codes:\n" + "\n".join(f"  {code:>3}  {text}" for code, text in EXIT_CODES.items())
    parser = argparse.ArgumentParser(prog="python -m playground study", description=__doc__, epilog=epilog,
                                     formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("study", help="study file (schema 1)")
    parser.add_argument("--out", help="study folder (default ~/markitect-playground-runs/<id>)")
    parser.add_argument("--codex-auth", help="Codex login (default ~/.codex/auth.json; the fake agent gets one "
                                             "only when this is given)")
    parser.add_argument("--claude-token", help="Claude Code token file (default ~/.markitect-playground/claude-token)")
    parser.add_argument("--fake-reviewers", action="store_true", help="fake reviewer CLIs, throwaway credentials")
    parser.add_argument("--preflight", action="store_true", help="only the cheap checks; build and run nothing")
    parser.add_argument("--keep-containers", action="store_true")
    args = parser.parse_args(sys.argv[1:] if argv is None else argv)
    previous = _interrupt_on_signals()
    try:
        return run(args)
    except manifest_module.ManifestError as exc:
        print(f"error: invalid study file {args.study}: {exc}", file=sys.stderr)
        return EXIT_ERROR
    except (host.HostError, evaluate.AssessError, OSError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return EXIT_ERROR
    except KeyboardInterrupt:
        print("interrupted", file=sys.stderr)
        return EXIT_INTERRUPTED
    finally:
        for number, handler in previous.items():
            signal.signal(number, handler)


def _raise_interrupt(signum: int, _frame: Any) -> None:
    raise KeyboardInterrupt(f"signal {signum}")


def _interrupt_on_signals() -> dict:
    """SIGTERM and, where it exists, SIGHUP (a closed terminal) raise KeyboardInterrupt, so
    the containers, the login copies and the lock are cleaned up as after Ctrl+C. Returns
    the previous handlers."""
    previous = {}
    for name in ("SIGTERM", "SIGHUP"):
        if hasattr(signal, name):
            number = getattr(signal, name)
            try:
                previous[number] = signal.signal(number, _raise_interrupt)
            except (ValueError, OSError):  # not the main thread
                pass
    return previous
