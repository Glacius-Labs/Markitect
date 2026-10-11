"""Host side: build the image (and Markitect binary), stage inputs, run one container.

  python -m playground host run --manifest M.json [--out DIR] [--codex-auth PATH]
      [--claude-token PATH] [--keep-container] [--exploratory]
  python -m playground host clean
  python -m playground host image-key

Before the build the evaluation files must be pre-registered (registration.py): no
uncommitted or untracked change in `evaluation/`. host.json records the commit and the
Git tree of `evaluation/` that `assess` will judge the run with. `--exploratory` runs
anyway and records `rules.exploratory: true`; `compare` then refuses the run.

The image is built from container/Dockerfile, the one place for its pins (image.py); a
base without @sha256: is refused. After the build one `docker run --rm --network none`
reads the image's inventory into the run folder's image-inventory.txt. A non-exact npm
version, or a difference from the committed container/inventory/codex-<v>-claude-<v>.txt,
fails the build (an environment failure, like every failed build); without that file the
run only warns. `image-key` prints a cache key of the pinned inputs.

Secrets are only checked for existence and mounted read-only by path: the Codex login
at /run/secrets/codex-auth.json, the Claude Code token at /run/secrets/claude-token.
Their contents are never read, printed, hashed or passed as an environment variable.

The container gets the normalized manifest (defaults filled in, Markitect's sourceRepo
as an absolute path and its full commit); host.json records it with the host platform.
The exit code (outcome.py) comes from the host status and the run report's class; the
runner's own code stays `containerExitCode` in host.json.
"""
from __future__ import annotations

import argparse
import csv
import hashlib
import io
import json
import os
import platform
import re
import shutil
import subprocess
import sys
import tarfile
import tempfile
from datetime import datetime, timezone
from pathlib import Path
from typing import Callable

from . import cases, image, outcome, registration
from . import manifest as manifest_module
from .runner import overhead_bound_seconds

ROOT = Path(__file__).resolve().parent.parent  # experiments/case-playground
LABEL = "markitect-playground=1"
# Codex's Linux sandbox (bubblewrap) needs user namespaces, which Docker's default
# seccomp profile blocks. Markitect's inner roles rely on that sandbox, so every run
# (both methods, for fairness) gets this one relaxation. No --privileged, no added caps.
SECURITY_OPTS = ["--security-opt", "seccomp=unconfined"]
COPY_IGNORE = shutil.ignore_patterns("__pycache__", "*.pyc")
CODEX_AUTH_TARGET = "/run/secrets/codex-auth.json"
CLAUDE_TOKEN_TARGET = "/run/secrets/claude-token"
# A failed `go build` is the product's only when the compiler reports errors in the
# product's own sources (GO_COMPILE_ERROR) and nothing points at the machine: network,
# disk, permissions, memory, a missing or undownloadable toolchain or module
# (GO_ENVIRONMENT_ERROR). When unsure, it is the environment's.
GO_ENVIRONMENT_ERROR = re.compile(
    r"dial tcp|no such host|i/o timeout|connection (?:refused|reset)|TLS handshake|network is unreachable"
    r"|name resolution|proxy\.golang\.org|sum\.golang\.org"
    r"|no space left on device|disk quota exceeded|read-only file system|input/output error"
    r"|permission denied|operation not permitted|too many open files|cannot allocate memory|out of memory"
    r"|signal: killed|resource temporarily unavailable|fork/exec|exec format error|text file busy"
    r"|toolchain not available|go: download go|requires go >= |GOROOT|go: cannot find|cannot find main module"
    r"|missing go\.sum entry|verifying .*: checksum mismatch|reading .*: (?:\d{3}|unexpected)|unrecognized import path"
    r"|go: (?:updates to go\.mod needed|errors parsing go\.mod)", re.I)
# file.go:line[:column]: message, for a file of the product (a relative path), never the
# module cache (path@version) or the toolchain (an absolute path).
GO_COMPILE_ERROR = re.compile(r"(?:^|\s)(?![/\\])(?![A-Za-z]:)[^\s@:]+\.go:\d+(?::\d+)?: \S", re.M)


class HostError(RuntimeError):
    """A clear, user-facing reason why the host could not run the container. `code` is
    the exit code: an environment failure unless the input was refused (outcome.INVALID)."""

    def __init__(self, message: str, code: int = outcome.ENVIRONMENT) -> None:
        super().__init__(message)
        self.code = code


def _now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds")


def _call(cmd: list[str], **kwargs) -> subprocess.CompletedProcess:
    try:
        return subprocess.run(cmd, **kwargs)
    except FileNotFoundError as exc:
        raise HostError(f"{cmd[0]} is not installed or not on PATH") from exc


def _capture(cmd: list[str], *, cwd: Path | None = None, env: dict | None = None) -> str:
    done = _call(cmd, cwd=cwd, env=env, capture_output=True, text=True, encoding="utf-8",
                 errors="replace")
    if done.returncode != 0:
        raise HostError(f"{' '.join(cmd[:3])} ... failed ({done.returncode}): {done.stderr.strip()}")
    return done.stdout.strip()


def _inside_git_checkout(path: Path) -> bool:
    return any((parent / ".git").exists() for parent in (path, *path.parents))


def _retryable(out: Path) -> bool:
    """An earlier attempt that failed before any container was launched may be replaced."""
    try:
        record = json.loads((out / "host.json").read_text(encoding="utf-8"))
    except (OSError, ValueError):
        return False
    return (isinstance(record, dict) and record.get("status") == "setup-failed"
            and record.get("containerLaunched") is False)


def station_count(manifest: dict) -> int:
    """Stations the run plans: the manifest's `stations` (the normalized manifest always
    has it), else every station of the case (the runner checks the plan again)."""
    if manifest.get("stations"):
        return manifest["stations"]
    try:
        return cases.get(manifest["case"], ROOT).stations
    except cases.CaseError as exc:
        raise HostError(str(exc), outcome.INVALID) from exc


def host_platform() -> dict[str, str]:
    """The host's OS and architecture; runs from different platforms are never paired."""
    return {"system": platform.system(), "machine": platform.machine()}


def host_timeout(manifest: dict, stations: int) -> int:
    """Safety net only: the agent budget plus the runner's worst-case own work."""
    return manifest["limits"]["totalSeconds"] + overhead_bound_seconds(stations)


def needs_codex_auth(manifest: dict) -> bool:
    """Codex as the outer agent, or Markitect's inner roles behind Claude Code."""
    kind = manifest["agent"]["kind"]
    return kind == "codex" or (kind == "claude" and manifest["method"] == "markitect")


def _operator() -> tuple[int, int] | None:
    """The host user's uid and gid on a POSIX host that does not run as root."""
    if not hasattr(os, "geteuid") or os.geteuid() == 0:
        return None
    return os.getuid(), os.getgid()


def _foreign(folder: Path, uid: int) -> int:
    """Entries below `folder` not owned by `uid`; a folder that cannot be listed counts."""
    count = 0

    def unreadable(error: OSError) -> None:
        nonlocal count
        count += 1

    for root, dirs, files in os.walk(folder, onerror=unreadable):
        for name in dirs + files:
            try:
                count += os.lstat(os.path.join(root, name)).st_uid != uid
            except FileNotFoundError:
                pass
    return count


def _stopped(container: str) -> bool:
    """True only when Docker confirms that the container has stopped or no longer exists."""
    done = _call(["docker", "container", "inspect", "--format", "{{.State.Running}}", container],
                 capture_output=True, text=True, encoding="utf-8", errors="replace")
    if done.returncode == 0:
        return done.stdout.strip() == "false"
    error = (done.stderr or "").lower()
    return "no such container" in error or "no such object" in error


def hand_back(container: str, image: str, folder: Path) -> str:
    """Give a container's output folder back to the host user: "done", "not-needed",
    "skipped: ..." or "failed: ...".

    Containers write as root and keep snapshots root-only (0700), so on a Linux host the
    operator could neither read nor delete a run. A short root container in the same
    image gives everything below `folder`, without following links, the owner of
    `folder`'s parent, which the host created; that owner maps to the operator under
    rootless and rootful engines alike. Nothing is needed on Windows, as root, or when
    the engine already maps container root to the operator. The host user usually shares
    uid 1000 with the container's agent, so nothing is handed back unless Docker confirms
    that `container` has stopped.
    """
    operator = _operator()
    if operator is None or not _foreign(folder, operator[0]):
        return "not-needed"
    cmd = ["docker", "run", "--rm", "--label", LABEL, "--network", "none", "--user", "0:0",
           "--mount", _mount(folder.parent, "/reference", readonly=True),
           "--mount", _mount(folder, "/handback"), "--entrypoint", "chown", image,
           "-R", "--no-dereference", "--reference=/reference", "/handback"]
    try:
        if not _stopped(container):
            return f"skipped: container {container} may still be running"
        done = _call(cmd, capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=600)
    except (HostError, subprocess.TimeoutExpired) as exc:
        return f"failed: {exc}"
    if done.returncode != 0:
        return f"failed: {(done.stderr or done.stdout).strip() or f'exit {done.returncode}'}"
    left = _foreign(folder, operator[0])
    return f"failed: {left} entries still belong to another user" if left else "done"


def _mount(source: Path, target: str, readonly: bool = False) -> str:
    # --mount is a CSV field list; csv quotes a source containing commas or quotes.
    fields = ["type=bind", f"source={source}", f"target={target}"] + (["readonly"] if readonly else [])
    buffer = io.StringIO()
    csv.writer(buffer, lineterminator="").writerow(fields)
    return buffer.getvalue()


def image_tag(manifest: dict) -> str:
    agent = manifest["agent"]
    return f"markitect-playground:codex-{agent['codexVersion']}-claude-{agent['claudeVersion']}"


PINS_HELP = 'see "Updating the image pins" in the playground README'


def image_pins() -> dict:
    """The pins of container/Dockerfile (image.pins); a missing or malformed pin is refused."""
    try:
        return image.pins(ROOT / "container" / "Dockerfile")
    except image.ImageError as exc:
        raise HostError(f"{exc}; {PINS_HELP}", outcome.INVALID) from exc


def committed_inventory(manifest: dict) -> Path:
    """The committed inventory of the image for the manifest's two CLI versions."""
    agent = manifest["agent"]
    return ROOT / "container" / "inventory" / image.inventory_name(agent["codexVersion"], agent["claudeVersion"])


def read_inventory(image_id: str) -> str:
    """The inventory the image's last build step wrote, read by one short container
    without network."""
    cmd = ["docker", "run", "--rm", "--label", LABEL, "--network", "none", "--entrypoint", "cat", image_id,
           image.INVENTORY_IN_IMAGE]
    try:
        done = _call(cmd, capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=300)
    except subprocess.TimeoutExpired as exc:
        raise HostError(f"reading the image inventory timed out after {exc.timeout} s") from exc
    if done.returncode != 0:
        raise HostError(f"could not read the image inventory {image.INVENTORY_IN_IMAGE} (exit {done.returncode}): "
                        f"{(done.stderr or '').strip()}")
    return done.stdout


INVENTORY_BEGIN = "----- built image inventory for {name} (copy the lines between the markers) -----"
INVENTORY_END = "----- end of built image inventory -----"


def show_inventory(text: str, committed: Path) -> None:
    """Print a built inventory that has no committed match, exactly in the committed format,
    so a build elsewhere (such as CI's smoke job) can deliver the file through its log."""
    print(INVENTORY_BEGIN.format(name=f"container/inventory/{committed.name}"), file=sys.stderr)
    print(text, end="" if text.endswith("\n") else "\n", file=sys.stderr)
    print(INVENTORY_END, file=sys.stderr, flush=True)


def build_image(manifest: dict, log_path: Path, inventory_path: Path) -> dict:
    """Build the image from the pinned Dockerfile, save its inventory as `inventory_path`
    and check it (image.check): exact npm versions, the Dockerfile's pins and, when one is
    committed, the same entries as committed_inventory. Returns the record host.json and
    study.json keep as `image`. Any failure is a failed build: HostError, the environment's."""
    pinned = image_pins()
    tag = image_tag(manifest)
    # Without provenance attestations a cached rebuild keeps its image ID, so both
    # arms can show they ran the same image. SOURCE_DATE_EPOCH (from the snapshot) fixes
    # the image's own timestamps.
    cmd = ["docker", "build", "-t", tag, "--provenance=false",
           "--build-arg", f"SOURCE_DATE_EPOCH={pinned['sourceDateEpoch']}",
           "--build-arg", f"CLAUDE_VERSION={manifest['agent']['claudeVersion']}",
           "--build-arg", f"CODEX_VERSION={manifest['agent']['codexVersion']}", str(ROOT / "container")]
    with open(log_path, "wb") as log:
        done = _call(cmd, stdout=log, stderr=subprocess.STDOUT)
    if done.returncode != 0:
        raise HostError(f"docker build failed; see {log_path}")
    image_id = _capture(["docker", "image", "inspect", "--format", "{{.Id}}", tag])
    text = read_inventory(image_id)
    inventory_path.write_text(text, encoding="utf-8", newline="\n")
    committed = committed_inventory(manifest)
    try:
        expected = committed.read_text(encoding="utf-8") if committed.is_file() else None
        image.check(text, pinned, expected)
    except (OSError, UnicodeError, image.ImageError) as exc:
        show_inventory(text, committed)
        raise HostError(f"image {image_id} failed its inventory check: {exc}; compare {inventory_path} with "
                        f"{committed} and {PINS_HELP}") from exc
    if expected is None:
        show_inventory(text, committed)
        print(f"warning: no committed image inventory {committed} to compare the image with; review {inventory_path} "
              f"and commit it with the Dockerfile ({PINS_HELP}); a study refuses to run without it",
              file=sys.stderr, flush=True)
    return {"tag": tag, "id": image_id, "base": pinned["base"], "snapshot": pinned["snapshot"],
            "dockerfileSha256": pinned["dockerfileSha256"], "inventorySha256": image.sha256(text)}


def resolve_markitect(product: dict) -> dict:
    """The manifest's Markitect block with sourceRepo as an absolute path and the full commit."""
    source = Path(product["sourceRepo"]).resolve()
    if not source.is_dir():
        raise HostError(f"markitect.sourceRepo is not a folder: {source}", outcome.INVALID)
    try:
        commit = _capture(["git", "-C", str(source), "rev-parse", "--verify", "--quiet",
                           f"{product['commit']}^{{commit}}"])
    except HostError as exc:
        raise HostError(f"markitect.commit {product['commit']} is not a commit in {source} "
                        f"(is it a Git checkout, and is the commit fetched?): {exc}", outcome.INVALID) from exc
    return {**product, "sourceRepo": str(source), "commit": commit}


def build_markitect(product: dict, target: Path) -> dict:
    """Build a static Linux binary from sourceRepo at the pinned commit."""
    if shutil.which("go") is None:
        raise HostError("Go is required to build the Markitect binary (go not on PATH)")
    source = product["sourceRepo"]
    commit = _capture(["git", "-C", source, "rev-parse", "--verify", f"{product['commit']}^{{commit}}"])
    target.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="mpg-markitect-") as temp:
        archive, tree = Path(temp) / "source.tar", Path(temp) / "src"
        # No checkout conversion: the binary embeds templates byte for byte.
        _capture(["git", "-C", source, "-c", "core.autocrlf=false", "-c", "core.eol=lf",
                  "archive", "--format=tar", "-o", str(archive), commit])
        with tarfile.open(archive) as tar:
            if hasattr(tarfile, "data_filter"):
                tar.extractall(tree, filter="data")
            else:  # Python < 3.11.4; the archive comes from the pinned product commit
                tar.extractall(tree)
        # Pin what changes the build; keep the rest (proxy, caches) from the host. The
        # toolchain follows the commit's go.mod; the one actually used is recorded.
        env = {**os.environ, "GOOS": "linux", "GOARCH": "amd64", "GOAMD64": "v1", "CGO_ENABLED": "0",
               "GOFLAGS": "", "GOTOOLCHAIN": "auto", "GOWORK": "off", "GOEXPERIMENT": ""}
        try:
            _capture(["go", "build", "-trimpath", "-o", str(target), "./src/cmd/markitect"],
                     cwd=tree, env=env)
        except HostError as exc:  # go ran and failed: the product's only for a compile error
            if isinstance(exc.__cause__, FileNotFoundError) or go_failure_class(str(exc)) != outcome.PRODUCT:
                raise  # no go, or the machine failed the build: the environment's
            raise HostError(f"the Markitect commit {commit} does not build: {exc}", outcome.PRODUCT) from exc
        go_version = _capture(["go", "version", str(target)], cwd=tree, env=env).split()[-1]
    return {"commit": commit, "sha256": hashlib.sha256(target.read_bytes()).hexdigest(), "go": go_version}


def go_failure_class(output: str) -> int:
    """outcome.PRODUCT when a failed `go build`'s output shows compile errors in the
    product's sources and no sign of the machine (network, disk, permissions, memory,
    toolchain or module download), else outcome.ENVIRONMENT."""
    if GO_ENVIRONMENT_ERROR.search(output) or not GO_COMPILE_ERROR.search(output):
        return outcome.ENVIRONMENT
    return outcome.PRODUCT


def stage_inputs(manifest: dict, inputs: Path) -> None:
    """Copy what the container needs to `inputs` (mounted read-only at /in), and the
    normalized manifest as `manifest.json`.

    The agent can read /in, so only this run's case and the files the code uses are
    staged: never another case (or its public checks), the other arm's method files or
    notes for people (methods/markitect/README.md).
    """
    case = manifest["case"]
    folders = ["playground", "cases/common", f"cases/{case}"]
    files = ["cases/task-prompt.txt"]
    if manifest["method"] == "conventional":
        files.append("methods/conventional/AGENTS.fragment.md")
    fake = {"fake": "tests/fake_agent.py", "fake-claude": "tests/fake_claude.py"}.get(manifest["agent"]["kind"])
    if fake:
        files.append(fake)
    for name in folders:
        if not (ROOT / name).is_dir():
            raise HostError(f"missing playground folder: {ROOT / name}")
        shutil.copytree(ROOT / name, inputs / name, ignore=COPY_IGNORE)
    for name in files:
        (inputs / name).parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(ROOT / name, inputs / name)
    _write_json(inputs / "manifest.json", manifest)


def docker_run_argv(manifest: dict, *, name: str, image: str, image_id: str, inputs: Path,
                    results: Path, auth: Path | None, markitect: dict | None,
                    claude_token: Path | None = None, host_os: dict | None = None) -> list[str]:
    limits = manifest["container"]
    argv = ["docker", "run", "--detach", "--name", name, "--label", LABEL, "--init",
            *SECURITY_OPTS,
            "--cpus", str(limits["cpus"]), "--memory", str(limits["memory"]),
            "--pids-limit", str(limits["pidsLimit"]),
            "--mount", _mount(inputs, "/in", readonly=True),
            "--mount", _mount(results, "/out")]
    if auth is not None:
        argv += ["--mount", _mount(auth, CODEX_AUTH_TARGET, readonly=True)]
    if claude_token is not None:  # the runner hands it only to the claude process
        argv += ["--mount", _mount(claude_token, CLAUDE_TOKEN_TARGET, readonly=True)]
    argv += ["--env", f"MPG_IMAGE_ID={image_id}"]
    if host_os:  # the runner records it; inside, platform.system() would name the container
        argv += ["--env", f"MPG_HOST_SYSTEM={host_os['system']}",
                 "--env", f"MPG_HOST_MACHINE={host_os['machine']}"]
    if markitect:  # the resolved full commit and binary hash built on the host
        argv += ["--env", f"MPG_MARKITECT_COMMIT={markitect['commit']}",
                 "--env", f"MPG_MARKITECT_SHA256={markitect['sha256']}"]
    argv += ["--workdir", "/in", image,
             "python3", "-m", "playground", "run", "--manifest", "/in/manifest.json", "--out", "/out"]
    return argv


def wait_container(name: str, timeout: float) -> tuple[int | None, str | None]:
    """Stream the container's output and wait for it; raises TimeoutExpired on the host
    timeout. Returns (exit code, None), or (None, error) when `docker wait` failed."""
    follower = subprocess.Popen(["docker", "logs", "--follow", name])
    try:
        done = subprocess.run(["docker", "wait", name], capture_output=True, text=True,
                              encoding="utf-8", errors="replace", timeout=timeout)
        try:
            follower.wait(timeout=30)
        except subprocess.TimeoutExpired:
            pass  # only the log stream lags; the container has ended
    finally:
        if follower.poll() is None:
            follower.kill()
    lines = done.stdout.split()
    if done.returncode == 0 and lines and lines[-1].lstrip("-").isdigit():
        return int(lines[-1]), None
    return None, (done.stderr or done.stdout).strip() or f"docker wait exited {done.returncode}"


def _container_state(name: str) -> str | None:
    """'running', 'stopped' or None when no such container exists."""
    done = _call(["docker", "container", "inspect", "--format", "{{.State.Running}}", name],
                 capture_output=True, text=True, encoding="utf-8", errors="replace")
    if done.returncode != 0:
        return None
    return "running" if done.stdout.strip() == "true" else "stopped"


def oom_killed(name: str) -> bool | None:
    """Docker's State.OOMKilled of a container (the kernel killed it for memory); None
    when Docker cannot say."""
    done = _call(["docker", "container", "inspect", "--format", "{{.State.OOMKilled}}", name],
                 capture_output=True, text=True, encoding="utf-8", errors="replace")
    value = done.stdout.strip() if done.returncode == 0 else ""
    return {"true": True, "false": False}.get(value)


def _finish_container(name: str, record: dict, out: Path, keep: bool,
                      before_remove: Callable[[str], None] | None = None) -> None:
    """Kill after a host timeout or interrupt, save the log, record Docker's
    `oomKilled`, call `before_remove` once Docker confirms the container has stopped (the
    study copies a login back out), then remove the container unless `keep`, also when
    `before_remove` is interrupted."""
    if record["status"] in ("host-timeout", "host-interrupted"):
        _call(["docker", "kill", name], capture_output=True)
    with open(out / "container.log", "wb") as log:
        _call(["docker", "logs", name], stdout=log, stderr=subprocess.STDOUT)
    record["oomKilled"] = oom_killed(name)
    try:
        if before_remove is not None and _container_state(name) == "stopped":
            try:
                before_remove(name)
            except Exception as exc:
                print(f"warning: after container {name} stopped: {exc}", file=sys.stderr)
    finally:  # never let it keep a container (and its mounted logins) alive
        if not keep:
            _remove_container(name)


def _remove_container(name: str) -> None:
    if _container_state(name) == "running":  # e.g. docker wait failed: never destroy a live run
        print(f"warning: container {name} is still running and was not removed; "
              f"`docker wait {name}` or `docker rm -f {name}`", file=sys.stderr)
        return
    _call(["docker", "rm", "-f", name], capture_output=True)
    if _container_state(name) is not None:
        print(f"warning: could not remove container {name} (it holds the mounted logins); "
              f"run `docker rm -f {name}`", file=sys.stderr)


def _write_json(path: Path, value: dict) -> None:
    path.write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n", encoding="utf-8",
                    newline="\n")


def run_class(results: Path) -> str | None:
    """The class in a run's results/report.json; None when there is no readable report."""
    try:
        report = json.loads((results / "report.json").read_text(encoding="utf-8"))
        value = report["classification"]["class"]
    except (OSError, UnicodeError, ValueError, KeyError, TypeError):
        return None
    return value if isinstance(value, str) else None


def platform_warning() -> str | None:
    """A warning on a non-Linux host: Linux is the reference platform (DEC-013)."""
    system = platform.system()
    if system == "Linux":
        return None
    return (f"this host runs {system or 'an unknown system'}; Linux is the reference platform (DEC-013), "
            "and compare refuses to pair runs from different host platforms")


def run(args: argparse.Namespace) -> int:
    manifest = manifest_module.load(Path(args.manifest), playground=ROOT)
    out = Path(args.out or Path.home() / "markitect-playground-runs" / manifest["id"]).resolve()
    if out.exists():
        if not _retryable(out):
            raise HostError(f"output folder already exists: {out}", outcome.INVALID)
        print(f"replacing {out} (its earlier attempt failed before a container was launched)", flush=True)
        shutil.rmtree(out)
    if _inside_git_checkout(out):
        raise HostError(f"output folder must not be inside a git checkout: {out}", outcome.INVALID)
    pre_registration = registration.observe(ROOT)
    if pre_registration["status"] != "registered":
        if not args.exploratory:
            raise HostError(f"{registration.refusal(pre_registration)}; commit them before the runs they judge, "
                            "or pass --exploratory (compare then refuses the run)", outcome.INVALID)
        print(f"warning: exploratory run: {registration.refusal(pre_registration)}", file=sys.stderr, flush=True)
    elif args.exploratory:
        print("warning: exploratory run: compare refuses it", file=sys.stderr, flush=True)
    auth = token = None
    if needs_codex_auth(manifest):
        auth = Path(args.codex_auth or Path.home() / ".codex" / "auth.json")
        if not auth.is_file():  # existence check only; the file is never opened here
            raise HostError(f"Codex auth file not found: {auth}")
        auth = auth.resolve()
    kind = manifest["agent"]["kind"]
    if kind == "claude" and not args.claude_token:
        raise HostError("--claude-token PATH is required for agent kind claude (create the file once "
                        "with `claude setup-token`)", outcome.INVALID)
    if args.claude_token and kind in manifest_module.CLAUDE_KINDS:  # optional for fake-claude
        token = Path(args.claude_token)
        if not token.is_file():  # existence check only; the file is never opened here
            raise HostError(f"Claude token file not found: {token}")
        token = token.resolve()
    station_count(manifest)  # fails before Docker for a case without a valid plan
    image_pins()  # refuses an unpinned Dockerfile before Docker
    if manifest["method"] == "markitect":
        if shutil.which("go") is None:
            raise HostError("Go is required to build the Markitect binary (go not on PATH)")
        manifest = {**manifest, "markitect": resolve_markitect(manifest["markitect"])}
    warning = platform_warning()
    if warning:
        print(f"warning: {warning}", file=sys.stderr, flush=True)
    exit_code, _record = run_manifest(manifest, out, auth=auth, token=token, keep=args.keep_container,
                                      pre_registration=pre_registration, exploratory=args.exploratory)
    print(f"host record: {out / 'host.json'}")
    print(f"report: {out / 'results' / 'report.md'}")
    return exit_code


def run_manifest(manifest: dict, out: Path, *, auth: Path | None, token: Path | None, keep: bool = False,
                 prebuilt: dict | None = None,
                 before_remove: Callable[[str], None] | None = None,
                 pre_registration: dict | None = None, exploratory: bool = False) -> tuple[int, dict]:
    """Run one validated manifest (Markitect block resolved) into the new folder `out`
    and return (exit code, host record); host.json is written in any case. The exit code
    (outcome.host_run) is also `exitCode` in host.json.

    `auth` and `token` are login files mounted read-only (never opened here). `prebuilt`
    holds what a study built once for all its runs: "image" (build_image's record),
    "inventory" (the path of its image-inventory.txt) and, for the Markitect method,
    "markitect" (the build record) and "binary" (its path). Without it the image and the
    binary are built here. Either way the run folder gets image-inventory.txt.
    `before_remove(container)` runs after the container has stopped and before it is
    removed. `pre_registration` (registration.observe at run start) and `exploratory` go
    to host.json; `assess` judges with that registration.
    """
    stations = station_count(manifest)
    inputs, results = out / "inputs", out / "results"
    results.mkdir(parents=True)
    name = f"mpg-{manifest['id']}"
    timeout = host_timeout(manifest, stations)
    record: dict = {"manifest": manifest, "status": "setup-failed", "container": name,
                    "image": {"tag": image_tag(manifest), "id": None}, "dockerVersion": None,
                    "markitect": None, "hostStartedAt": _now(), "startedAt": None,
                    "endedAt": None, "hostTimeoutSeconds": timeout, "stations": stations,
                    "hostPlatform": host_platform(),
                    "preRegistration": pre_registration, "rules": {"exploratory": bool(exploratory)},
                    "secrets": {"codexAuth": auth is not None, "claudeToken": token is not None},
                    "containerLaunched": False, "failureClass": None,
                    "containerExitCode": None, "oomKilled": None, "exitCode": None, "error": None}
    try:
        record["dockerVersion"] = _capture(["docker", "version", "--format", "{{.Server.Version}}"])
        if _container_state(name) is not None:
            raise HostError(f"a container named {name} already exists; remove it (docker rm -f {name}) "
                            "or use a new manifest id")
        if prebuilt is None:
            print(f"building image {record['image']['tag']} ...", flush=True)
            record["image"] = build_image(manifest, out / "image-build.log", out / "image-inventory.txt")
        else:
            record["image"] = dict(prebuilt["image"])
            shutil.copyfile(prebuilt["inventory"], out / "image-inventory.txt")
        stage_inputs(manifest, inputs)
        if manifest["method"] == "markitect":
            binary = inputs / "bin" / "markitect"
            if prebuilt is None:
                print("building markitect binary ...", flush=True)
                record["markitect"] = build_markitect(manifest["markitect"], binary)
            else:
                binary.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(prebuilt["binary"], binary)
                record["markitect"] = dict(prebuilt["markitect"])
        # A prebuilt image runs by its ID, so a tag rebuilt meanwhile cannot change it.
        argv = docker_run_argv(manifest, name=name, image=record["image"]["tag"] if prebuilt is None
                               else record["image"]["id"],
                               image_id=record["image"]["id"], inputs=inputs, results=results,
                               auth=auth, markitect=record["markitect"], claude_token=token,
                               host_os=record["hostPlatform"])
        record["dockerRun"] = argv
        record["startedAt"] = _now()
        # From here on the container may exist, even if `docker run` is interrupted or fails.
        record["containerLaunched"] = True
        _capture(argv)
        code, error = wait_container(name, timeout)
        if error is None:
            record["status"], record["containerExitCode"] = "completed", code
        else:
            record["status"], record["error"] = "wait-failed", error
            print(f"error: docker wait failed: {error}", file=sys.stderr)
    except subprocess.TimeoutExpired:
        record["status"] = "host-timeout"
    except KeyboardInterrupt:
        record["status"] = "host-interrupted"
    except (HostError, OSError, tarfile.TarError) as exc:
        if record["containerLaunched"]:
            record["status"] = "start-failed"
        elif isinstance(exc, HostError) and exc.code == outcome.PRODUCT:
            record["failureClass"] = "product"  # the Markitect binary does not build
        record["error"] = str(exc)
        print(f"error: {exc}", file=sys.stderr)
    except Exception as exc:  # a bug of ours: recorded with the code __main__ exits with, then raised
        record["status"], record["error"] = "harness-error", f"{type(exc).__name__}: {exc}"
        raise
    finally:
        if record["containerLaunched"]:
            try:
                _finish_container(name, record, out, keep, before_remove)
            except KeyboardInterrupt:  # e.g. during the login copy-out; the container is gone
                record["status"] = "host-interrupted"
        record["endedAt"] = _now()
        record["exitCode"] = exit_code = outcome.host_run(
            record["status"], record["containerExitCode"],
            run_class(results) if record["status"] == "completed" else None,
            failure_class=record["failureClass"], oom_killed=record["oomKilled"])
        _write_json(out / "host.json", record)  # first, so an interrupted hand-back keeps the record
        if record["containerLaunched"]:
            record["handBack"] = hand_back(name, record["image"]["id"] or record["image"]["tag"], results)
            _write_json(out / "host.json", record)
            if record["handBack"].startswith(("failed", "skipped")):
                print(f"warning: {results} stays owned by root ({record['handBack']})", file=sys.stderr)
    return exit_code, record


def clean() -> int:
    """Remove playground containers that are not running."""
    ids = _capture(["docker", "ps", "--all", "--quiet", "--filter", f"label={LABEL}",
                    "--filter", "status=created", "--filter", "status=exited",
                    "--filter", "status=dead"]).split()
    if ids:
        _capture(["docker", "rm", *ids])
    print(f"removed {len(ids)} stopped playground container(s)")
    return 0


def image_key() -> int:
    """Print a stable cache key of the image's pinned inputs (image.key), e.g. for CI caching."""
    try:
        key, files = image.key(ROOT / "container")
    except (OSError, image.ImageError) as exc:
        raise HostError(f"{exc}; {PINS_HELP}", outcome.INVALID) from exc
    if not files:
        print(f"warning: no committed image inventory in {ROOT / 'container' / 'inventory'}; the key covers the "
              f"Dockerfile only ({PINS_HELP})", file=sys.stderr)
    print(key)
    return outcome.OK


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="python -m playground host", description=__doc__,
                                     epilog=outcome.help_text(),
                                     formatter_class=argparse.RawDescriptionHelpFormatter)
    commands = parser.add_subparsers(dest="command", required=True)
    run_parser = commands.add_parser("run", help="build, stage and run one manifest", epilog=outcome.help_text(),
                                     formatter_class=argparse.RawDescriptionHelpFormatter)
    run_parser.add_argument("--manifest", required=True)
    run_parser.add_argument("--out")
    run_parser.add_argument("--codex-auth")
    run_parser.add_argument("--claude-token", help="token file from `claude setup-token` (kind claude)")
    run_parser.add_argument("--keep-container", action="store_true")
    run_parser.add_argument("--exploratory", action="store_true",
                            help="run although the evaluation files are not pre-registered; compare refuses the run")
    commands.add_parser("clean", help="remove stopped playground containers")
    commands.add_parser("image-key", help="print a cache key of the pinned base, the Debian snapshot, the committed "
                                          "image inventories and the Dockerfile's SHA-256")
    args = parser.parse_args(sys.argv[1:] if argv is None else argv)
    try:
        if args.command == "image-key":
            return image_key()
        return run(args) if args.command == "run" else clean()
    except HostError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return exc.code
    except manifest_module.ManifestError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return outcome.INVALID
