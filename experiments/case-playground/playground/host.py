"""Host side: build the image (and Markitect binary), stage inputs, run one container.

  python -m playground host run --manifest M.json [--out DIR] [--codex-auth PATH]
      [--claude-token PATH] [--keep-container]
  python -m playground host clean

Secrets are only checked for existence and mounted read-only by path: the Codex login
at /run/secrets/codex-auth.json, the Claude Code token at /run/secrets/claude-token.
Their contents are never read, printed, hashed or passed as an environment variable.
"""
from __future__ import annotations

import argparse
import csv
import hashlib
import io
import json
import os
import shutil
import subprocess
import sys
import tarfile
import tempfile
from datetime import datetime, timezone
from pathlib import Path

from . import manifest as manifest_module
from .lifecycle import LifecycleError, load_station_plan
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


class HostError(RuntimeError):
    """A clear, user-facing reason why the host could not run the container."""


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
    """Number of stations in the case's STATIONS.json (the runner checks the plan again)."""
    case = manifest["case"]
    try:
        return len(load_station_plan(ROOT / "cases" / case / "STATIONS.json", case))
    except LifecycleError as exc:
        raise HostError(f"case {case} has no valid station plan: {exc}") from exc


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


def hand_back(container: str, image: str, folder: Path) -> str:
    """Give a container's output folder back to the host user: "done", "not-needed",
    "skipped: ..." or "failed: ...".

    Containers write as root and keep snapshots root-only (0700), so on a Linux host the
    operator could neither read nor delete a run. A short root container in the same
    image changes the owner, without following links. Nothing is needed on Windows, as
    root, or when the engine already maps container root to the operator (rootless
    Docker, Docker Desktop on macOS). The host user usually shares uid 1000 with the
    container's agent, so nothing is handed back while `container` still runs.
    """
    operator = _operator()
    if operator is None:
        return "not-needed"
    try:
        foreign = [entry for entry in os.scandir(folder)
                   if entry.stat(follow_symlinks=False).st_uid != operator[0]]
    except OSError as exc:
        return f"failed: {exc}"
    if not foreign:
        return "not-needed"
    if _container_state(container) == "running":
        return f"skipped: container {container} is still running"
    cmd =["docker", "run", "--rm", "--label", LABEL, "--network", "none", "--user", "0:0",
           "--mount", _mount(folder, "/handback"), "--entrypoint", "chown", image,
           "-R", "--no-dereference", f"{operator[0]}:{operator[1]}", "/handback"]
    try:
        done = _call(cmd, capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=600)
    except (HostError, subprocess.TimeoutExpired) as exc:
        return f"failed: {exc}"
    if done.returncode != 0:
        return f"failed: {(done.stderr or done.stdout).strip() or f'exit {done.returncode}'}"
    return "done"


def _mount(source: Path, target: str, readonly: bool = False) -> str:
    # --mount is a CSV field list; csv quotes a source containing commas or quotes.
    fields = ["type=bind", f"source={source}", f"target={target}"] + (["readonly"] if readonly else [])
    buffer = io.StringIO()
    csv.writer(buffer, lineterminator="").writerow(fields)
    return buffer.getvalue()


def image_tag(manifest: dict) -> str:
    agent = manifest["agent"]
    return f"markitect-playground:codex-{agent['codexVersion']}-claude-{agent['claudeVersion']}"


def build_image(manifest: dict, log_path: Path) -> str:
    tag = image_tag(manifest)
    # Without provenance attestations a cached rebuild keeps its image ID, so both
    # arms can show they ran the same image.
    cmd = ["docker", "build", "-t", tag, "--provenance=false",
           "--build-arg", f"CLAUDE_VERSION={manifest['agent']['claudeVersion']}",
           "--build-arg", f"CODEX_VERSION={manifest['agent']['codexVersion']}", str(ROOT / "container")]
    with open(log_path, "wb") as log:
        done = _call(cmd, stdout=log, stderr=subprocess.STDOUT)
    if done.returncode != 0:
        raise HostError(f"docker build failed; see {log_path}")
    return _capture(["docker", "image", "inspect", "--format", "{{.Id}}", tag])


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
        _capture(["go", "build", "-trimpath", "-o", str(target), "./src/cmd/markitect"],
                 cwd=tree, env=env)
        go_version = _capture(["go", "version", str(target)], cwd=tree, env=env).split()[-1]
    return {"commit": commit, "sha256": hashlib.sha256(target.read_bytes()).hexdigest(), "go": go_version}


def stage_inputs(manifest: dict, manifest_path: Path, inputs: Path) -> None:
    """Copy what the container needs to `inputs` (mounted read-only at /in).

    The agent can read /in, so only this run's case and the files the code uses are
    staged: never the other case, the other arm's method files or notes for people
    (methods/markitect/README.md).
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
    shutil.copyfile(manifest_path, inputs / "manifest.json")


def docker_run_argv(manifest: dict, *, name: str, image: str, image_id: str, inputs: Path,
                    results: Path, auth: Path | None, markitect: dict | None,
                    claude_token: Path | None = None) -> list[str]:
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


def _finish_container(name: str, record: dict, out: Path, keep: bool) -> None:
    if record["status"] in ("host-timeout", "host-interrupted"):
        _call(["docker", "kill", name], capture_output=True)
    with open(out / "container.log", "wb") as log:
        _call(["docker", "logs", name], stdout=log, stderr=subprocess.STDOUT)
    if keep:
        return
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


def run(args: argparse.Namespace) -> int:
    manifest = manifest_module.load(Path(args.manifest))
    out = Path(args.out or Path.home() / "markitect-playground-runs" / manifest["id"]).resolve()
    if out.exists():
        if not _retryable(out):
            raise HostError(f"output folder already exists: {out}")
        print(f"replacing {out} (its earlier attempt failed before a container was launched)", flush=True)
        shutil.rmtree(out)
    if _inside_git_checkout(out):
        raise HostError(f"output folder must not be inside a git checkout: {out}")
    auth = token = None
    if needs_codex_auth(manifest):
        auth = Path(args.codex_auth or Path.home() / ".codex" / "auth.json")
        if not auth.is_file():  # existence check only; the file is never opened here
            raise HostError(f"Codex auth file not found: {auth}")
        auth = auth.resolve()
    kind = manifest["agent"]["kind"]
    if kind == "claude" and not args.claude_token:
        raise HostError("--claude-token PATH is required for agent kind claude (create the file once "
                        "with `claude setup-token`)")
    if args.claude_token and kind in manifest_module.CLAUDE_KINDS:  # optional for fake-claude
        token = Path(args.claude_token)
        if not token.is_file():  # existence check only; the file is never opened here
            raise HostError(f"Claude token file not found: {token}")
        token = token.resolve()
    stations = station_count(manifest)
    if manifest["method"] == "markitect" and shutil.which("go") is None:
        raise HostError("Go is required to build the Markitect binary (go not on PATH)")

    inputs, results = out / "inputs", out / "results"
    results.mkdir(parents=True)
    name = f"mpg-{manifest['id']}"
    timeout = host_timeout(manifest, stations)
    record: dict = {"manifest": manifest, "status": "setup-failed", "container": name,
                    "image": {"tag": image_tag(manifest), "id": None}, "dockerVersion": None,
                    "markitect": None, "hostStartedAt": _now(), "startedAt": None,
                    "endedAt": None, "hostTimeoutSeconds": timeout, "stations": stations,
                    "secrets": {"codexAuth": auth is not None, "claudeToken": token is not None},
                    "containerLaunched": False,
                    "containerExitCode": None, "error": None}
    exit_code = 2
    try:
        record["dockerVersion"] = _capture(["docker", "version", "--format", "{{.Server.Version}}"])
        if _container_state(name) is not None:
            raise HostError(f"a container named {name} already exists; remove it (docker rm -f {name}) "
                            "or use a new manifest id")
        print(f"building image {record['image']['tag']} ...", flush=True)
        record["image"]["id"] = build_image(manifest, out / "image-build.log")
        stage_inputs(manifest, Path(args.manifest), inputs)
        if manifest["method"] == "markitect":
            print("building markitect binary ...", flush=True)
            record["markitect"] = build_markitect(manifest["markitect"], inputs / "bin" / "markitect")
        argv = docker_run_argv(manifest, name=name, image=record["image"]["tag"],
                               image_id=record["image"]["id"], inputs=inputs, results=results,
                               auth=auth, markitect=record["markitect"], claude_token=token)
        record["dockerRun"] = argv
        record["startedAt"] = _now()
        # From here on the container may exist, even if `docker run` is interrupted or fails.
        record["containerLaunched"] = True
        _capture(argv)
        code, error = wait_container(name, timeout)
        if error is None:
            record["status"], record["containerExitCode"], exit_code = "completed", code, code
        else:
            record["status"], record["error"] = "wait-failed", error
            print(f"error: docker wait failed: {error}", file=sys.stderr)
    except subprocess.TimeoutExpired:
        record["status"], exit_code = "host-timeout", 124
    except KeyboardInterrupt:
        record["status"], exit_code = "host-interrupted", 130
    except (HostError, OSError, tarfile.TarError) as exc:
        if record["containerLaunched"]:
            record["status"] = "start-failed"
        record["error"] = str(exc)
        print(f"error: {exc}", file=sys.stderr)
    finally:
        if record["containerLaunched"]:
            _finish_container(name, record, out, args.keep_container)
            record["handBack"] = hand_back(name, record["image"]["id"] or record["image"]["tag"], results)
            if record["handBack"].startswith(("failed", "skipped")):
                print(f"warning: {results} stays owned by root ({record['handBack']})", file=sys.stderr)
        record["endedAt"] = _now()
        _write_json(out / "host.json", record)
    print(f"host record: {out / 'host.json'}")
    print(f"report: {results / 'report.md'}")
    return exit_code


def clean() -> int:
    """Remove playground containers that are not running."""
    ids = _capture(["docker", "ps", "--all", "--quiet", "--filter", f"label={LABEL}",
                    "--filter", "status=created", "--filter", "status=exited",
                    "--filter", "status=dead"]).split()
    if ids:
        _capture(["docker", "rm", *ids])
    print(f"removed {len(ids)} stopped playground container(s)")
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="python -m playground host", description=__doc__,
                                     formatter_class=argparse.RawDescriptionHelpFormatter)
    commands = parser.add_subparsers(dest="command", required=True)
    run_parser = commands.add_parser("run", help="build, stage and run one manifest")
    run_parser.add_argument("--manifest", required=True)
    run_parser.add_argument("--out")
    run_parser.add_argument("--codex-auth")
    run_parser.add_argument("--claude-token", help="token file from `claude setup-token` (kind claude)")
    run_parser.add_argument("--keep-container", action="store_true")
    commands.add_parser("clean", help="remove stopped playground containers")
    args = parser.parse_args(sys.argv[1:] if argv is None else argv)
    try:
        return run(args) if args.command == "run" else clean()
    except (HostError, manifest_module.ManifestError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 2
