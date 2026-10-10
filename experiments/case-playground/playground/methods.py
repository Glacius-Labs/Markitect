"""Install one method into the prepared case repository.

`setup` runs in the container as root, but every command it starts goes through
`ctx["runAsAgent"]`, so Git and the product run as the agent user. Each command, its
exit code and duration is returned in `steps`; raw output goes to `ctx["outDir"]`.

With Claude Code as the outer agent, both methods first get the same one-line
`CLAUDE.md` router to AGENTS.md when the repository has none (Claude Code does not read
AGENTS.md by itself). For Markitect the product's role configuration is read back from
`.markitect/runtime.yaml` and returned in `roles`.
"""
from __future__ import annotations

import json
import os
import shutil
import time
from pathlib import Path
from typing import Any, Callable

from .claude_agent import ROUTER as CLAUDE_ROUTER
from .lifecycle import match_owner, open_plain, read_text as _read, sha_file
from .manifest import CLAUDE_KINDS

SETUP_IDENTITY = {
    "GIT_AUTHOR_NAME": "Playground setup",
    "GIT_AUTHOR_EMAIL": "setup@playground.invalid",
    "GIT_COMMITTER_NAME": "Playground setup",
    "GIT_COMMITTER_EMAIL": "setup@playground.invalid",
}
MARKITECT_BINARY = Path("/usr/local/bin/markitect")
MARKITECT_BRANCH = "markitect-setup"
RUNTIME_PATH = ".markitect/runtime.yaml"
RUNTIME_LIMIT = 4 << 20
# `markitect config` requires caller-supplied cost weights and a cost cap.
# Each token weighs one micro-unit (1,000,000 per million), so the product's cost
# counter equals its token count. The cap scales with the manifest's wall-clock
# budget and is set high enough that it cannot bind before that shared time limit.
TOKEN_WEIGHT_MICROS_PER_MILLION = 1_000_000
COST_CAP_TOKENS_PER_SECOND = 50_000
MAX_COST_MICROS_LIMIT = 1_000_000_000_000
GIT_TIMEOUT = 120
PRODUCT_TIMEOUT = 600
# Worst case for the Markitect setup (7 product and 9 Git steps, plus 2 for the CLAUDE.md
# router; conventional needs 4 Git steps), used for the host's safety timeout.
SETUP_BOUND_SECONDS = 7 * PRODUCT_TIMEOUT + 12 * GIT_TIMEOUT


class SetupBlocked(RuntimeError):
    """A setup command failed; the method is not usable as installed.

    `source` classifies the failure: "product" (a Markitect command), "harness" (our Git
    steps or staging) or "environment" (the image lacks something)."""

    def __init__(self, message: str, source: str = "product") -> None:
        super().__init__(message)
        self.source = source


class _Recorder:
    def __init__(self, repo: Path, out_dir: Path, run: Callable[..., dict]):
        self.repo, self.out_dir, self.run = repo, out_dir, run
        self.out_dir.mkdir(parents=True, exist_ok=True)
        self.steps: list[dict[str, Any]] = []

    def step(self, name: str, argv: list[str], *, timeout: float = GIT_TIMEOUT,
             env: dict | None = None, allow_failure: bool = False, source: str = "harness") -> tuple[dict, str]:
        index = len(self.steps) + 1
        stdout_path = self.out_dir / f"{index:02d}-{name}.stdout.txt"
        stderr_path = self.out_dir / f"{index:02d}-{name}.stderr.txt"
        result = self.run(argv, self.repo, timeout, stdout_path, stderr_path, extra_env=env)
        record = {"name": name, "argv": list(argv), "exitCode": result.get("exitCode"),
                  "timedOut": bool(result.get("timedOut")), "seconds": result.get("seconds"),
                  "stdout": stdout_path.name, "stderr": stderr_path.name}
        self.steps.append(record)
        stdout = _read(stdout_path)
        if not allow_failure and (record["timedOut"] or record["exitCode"] != 0):
            detail = _read(stderr_path).strip() or stdout.strip()
            reason = "timed out" if record["timedOut"] else f"exit {record['exitCode']}"
            raise SetupBlocked(f"{name} failed ({reason}): {detail[:2000]}", source)
        return record, stdout

    def git(self, name: str, *args: str, allow_failure: bool = False) -> str:
        return self.step(name, ["git", *args], env=SETUP_IDENTITY, allow_failure=allow_failure)[1]


def setup(method: str, repo: Path, ctx: dict) -> dict:
    """Install `method` into `repo` and commit it. Returns the setup record
    (the runner saves it); step output files are named relative to `ctx["outDir"]`."""
    recorder = _Recorder(Path(repo), Path(ctx["outDir"]), ctx["runAsAgent"])
    result: dict[str, Any] = {"status": "ready", "method": method, "seconds": None, "commit": None,
                              "mcpServers": {}, "steps": recorder.steps, "error": None, "blockedBy": None,
                              "roles": None, "notes": {}}
    started = time.monotonic()
    try:
        if ctx["manifest"]["agent"]["kind"] in CLAUDE_KINDS:
            _claude_router(recorder, result)
        if method == "conventional":
            _conventional(recorder, ctx, result)
        elif method == "markitect":
            _markitect(recorder, ctx, result)
        else:
            raise SetupBlocked(f"unknown method {method!r}", "harness")
    except SetupBlocked as error:
        result.update(status="blocked", error=str(error), blockedBy=error.source, mcpServers={}, commit=None)
    result["seconds"] = round(time.monotonic() - started, 3)
    return result


def _claude_router(recorder: _Recorder, result: dict) -> None:
    """Same one-line CLAUDE.md for both methods, committed before the method's own setup."""
    path = recorder.repo / "CLAUDE.md"
    note = {"path": "CLAUDE.md", "content": CLAUDE_ROUTER, "added": False}
    result["notes"]["claudeRouter"] = note
    if path.exists() or path.is_symlink():
        return
    path.write_text(CLAUDE_ROUTER, encoding="utf-8", newline="\n")
    match_owner(path, recorder.repo)
    recorder.git("git-add-claude-router", "add", "CLAUDE.md")
    recorder.git("git-commit-claude-router", "commit", "-q", "-m", "Add CLAUDE.md router to AGENTS.md")
    note["added"] = True


def _conventional(recorder: _Recorder, ctx: dict, result: dict) -> None:
    fragment_path = Path(ctx["inDir"]) / "methods" / "conventional" / "AGENTS.fragment.md"
    fragment = fragment_path.read_text(encoding="utf-8")
    agents = recorder.repo / "AGENTS.md"
    existing = agents.read_text(encoding="utf-8") if agents.exists() else ""
    text = (existing.rstrip("\n") + "\n\n" if existing.strip() else "") + fragment.strip("\n") + "\n"
    agents.write_text(text, encoding="utf-8", newline="\n")
    match_owner(agents, recorder.repo)
    recorder.git("git-add", "add", "AGENTS.md")
    recorder.git("git-diff-check", "diff", "--cached", "--check", allow_failure=True)
    recorder.git("git-commit", "commit", "-q", "-m", "Install conventional workflow")
    result["commit"] = recorder.git("git-rev-parse", "rev-parse", "HEAD").strip()
    result["notes"]["agentsFragment"] = "methods/conventional/AGENTS.fragment.md"


def _markitect(recorder: _Recorder, ctx: dict, result: dict) -> None:
    manifest, repo = ctx["manifest"], recorder.repo.resolve()
    notes = result["notes"]
    binary = _install_markitect(Path(ctx["inDir"]), Path(ctx.get("markitectBinary", MARKITECT_BINARY)))
    notes["markitectBinary"] = str(binary)
    notes["markitectSha256"] = sha_file(binary)
    codex = ctx.get("codexExecutable") or find_native_codex()
    if not codex:
        raise SetupBlocked("no native Codex executable found (Markitect rejects script shims and symlinks)",
                           "environment")
    agent = manifest["agent"]
    budget = _budget(manifest)
    notes.update(providerExecutable=str(codex), budget=budget)

    def product(name: str, *args: str, allow_failure: bool = False) -> tuple[dict, dict | None]:
        record, stdout = recorder.step(name, [str(binary), *args], timeout=PRODUCT_TIMEOUT,
                                       allow_failure=allow_failure, source="product")
        try:
            return record, json.loads(stdout)
        except ValueError:
            if allow_failure:
                return record, None
            raise SetupBlocked(f"{name} did not return JSON")

    start_branch = recorder.git("git-current-branch", "branch", "--show-current").strip()
    if not start_branch:
        raise SetupBlocked("repository is not on a named branch", "harness")
    # Markitect writes only on a non-protected feature branch.
    recorder.git("git-switch-setup-branch", "switch", "-q", "-c", MARKITECT_BRANCH)

    base = ["--repo", str(repo)]
    init_args = [*base, "--name", ctx["case"]]
    _, preview = product("init-preview", "init", *init_args)
    init_digest = _digest(preview, "init-preview")
    _, written = product("init-write", "init", *init_args, "--expect", init_digest, "--write")
    if _digest(written, "init-write") != init_digest:
        raise SetupBlocked("init plan changed between preview and write")

    # Markitect's inner roles run on Codex, so setup needs the Codex guidance; a Claude Code
    # outer agent also gets the product's own Claude guidance.
    onboard_provider = "both" if agent["kind"] in CLAUDE_KINDS else "codex"
    notes["onboardProvider"] = onboard_provider
    onboard_args = [*base, "--provider", onboard_provider]
    _, preview = product("onboard-preview", "onboard", *onboard_args)
    onboard_digest = _digest(preview, "onboard-preview")
    product("onboard-write", "onboard", *onboard_args, "--expect", onboard_digest, "--write")

    # The inner roles' model: named in the manifest for a Claude Code outer agent, otherwise
    # the outer Codex agent's own.
    inner = manifest.get("markitect") or {}
    inner_model, inner_effort = inner.get("innerModel", agent["model"]), inner.get("innerEffort", agent["effort"])
    setup_args = [*base, "--provider", "codex", "--model", inner_model, "--effort", inner_effort,
                  "--provider-executable", str(codex),
                  "--input-micros-per-million", str(budget["inputMicrosPerMillion"]),
                  "--output-micros-per-million", str(budget["outputMicrosPerMillion"]),
                  "--max-cost-micros", str(budget["maxCostMicros"])]
    _, preview = product("setup-preview", "config", *setup_args)
    setup_digest = _digest(preview, "setup-preview", "editPlan")
    _, written = product("setup-write", "config", *setup_args, "--expect", setup_digest, "--write")
    notes["digests"] = {"init": init_digest, "onboard": onboard_digest, "setup": setup_digest}
    text, notes["runtimeSource"] = _runtime_text(written, recorder.repo)
    result["roles"] = runtime_roles(text) if text is not None else None

    recorder.git("git-add", "add", "-A")
    recorder.git("git-diff-check", "diff", "--cached", "--check", allow_failure=True)
    recorder.git("git-commit", "commit", "-q", "-m", "Install Markitect project workflow")
    recorder.git("git-switch-back", "switch", "-q", start_branch)
    recorder.git("git-merge", "merge", "-q", "--ff-only", MARKITECT_BRANCH)
    recorder.git("git-delete-setup-branch", "branch", "-q", "-d", MARKITECT_BRANCH)
    result["commit"] = recorder.git("git-rev-parse", "rev-parse", "HEAD").strip()

    # Observation only: the initial model owns nothing yet, so coverage is expected
    # to be incomplete until the agent models the project.
    record, report = product("check-after-setup", "check", *base, allow_failure=True)
    notes["initialCheck"] = {"exitCode": record["exitCode"],
                             "status": (report or {}).get("status"),
                             "coverageConforming": ((report or {}).get("coverage") or {}).get("conforming")}
    result["mcpServers"] = {"markitect": {"command": str(binary), "args": ["mcp", *base]}}


def find_native_codex() -> str | None:
    """Return the native Codex binary behind the `codex` command on PATH.

    The npm package puts a Node launcher on PATH; Markitect accepts only a direct
    native executable, which the package ships under its `vendor` directory.
    """
    found = shutil.which("codex")
    if not found:
        return None
    launcher = Path(os.path.realpath(found))
    if _is_elf(launcher):
        return str(launcher)
    package = launcher.parent.parent
    if not (package / "package.json").is_file():
        return None
    candidates = sorted(p for p in package.rglob("codex")
                        if "vendor" in p.parts and p.is_file() and not p.is_symlink() and _is_elf(p))
    return str(candidates[0]) if candidates else None


def _install_markitect(in_dir: Path, target: Path) -> Path:
    source = in_dir / "bin" / "markitect"
    if not source.is_file():
        raise SetupBlocked(f"Markitect binary missing: {source}", "harness")
    target.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(source, target)
    target.chmod(0o755)
    return target


def _budget(manifest: dict) -> dict:
    seconds = int(manifest["limits"]["totalSeconds"])
    cap = min(max(seconds * COST_CAP_TOKENS_PER_SECOND, 1), MAX_COST_MICROS_LIMIT)
    return {"inputMicrosPerMillion": TOKEN_WEIGHT_MICROS_PER_MILLION,
            "outputMicrosPerMillion": TOKEN_WEIGHT_MICROS_PER_MILLION, "maxCostMicros": cap}


def _digest(value: Any, step: str, *path: str) -> str:
    for key in (*path, "digest"):
        value = value.get(key) if isinstance(value, dict) else None
    if not isinstance(value, str) or not value:
        raise SetupBlocked(f"{step} returned no digest")
    return value


def _is_elf(path: Path) -> bool:
    try:
        with path.open("rb") as handle:
            return handle.read(4) == b"\x7fELF"
    except OSError:
        return False


# --- role configuration ----------------------------------------------------------------

def _runtime_text(written: Any, repo: Path) -> tuple[str | None, str | None]:
    """The runtime the product wrote and where it was read: the content its setup output
    printed, else the file in the repository."""
    mutation = written.get("mutation") if isinstance(written, dict) else None
    files = mutation.get("files") if isinstance(mutation, dict) else None
    for change in files if isinstance(files, list) else []:
        if isinstance(change, dict) and change.get("path") == RUNTIME_PATH and isinstance(change.get("content"), str):
            return change["content"], "setup-write output"
    handle = open_plain(repo / RUNTIME_PATH)
    if handle is None:
        return None, None
    with handle:
        return handle.read(RUNTIME_LIMIT).decode("utf-8", errors="replace"), RUNTIME_PATH


def runtime_roles(text: str) -> list[dict]:
    """Executor, model and effort per role of the product's runtime: a worker per Manager
    (`agents`), a reviewer per Manager (`review.agents`) and the `verifier`."""
    data = _yaml_mappings(text)

    def role(name: str, manager: str | None, agent: dict) -> dict:
        app = agent.get("appServer") if isinstance(agent.get("appServer"), dict) else {}
        return {"role": name, "manager": _manager_name(manager), "executor": agent.get("transport"),
                "providerVersion": agent.get("providerVersion"), "command": agent.get("command"),
                "model": agent.get("model"), "effort": app.get("reasoningEffort", agent.get("effort"))}

    review = data.get("review") if isinstance(data.get("review"), dict) else {}
    roles = []
    for name, group in (("worker", data.get("agents")), ("reviewer", review.get("agents"))):
        if isinstance(group, dict):
            roles += [role(name, manager, agent) for manager, agent in group.items() if isinstance(agent, dict)]
    if isinstance(data.get("verifier"), dict):
        roles.append(role("verifier", None, data["verifier"]))
    return roles


def _manager_name(key: str | None) -> str | None:
    """Manager keys are JSON identity tuples; the last element is the Manager's name."""
    try:
        value = json.loads(key) if key else None
    except ValueError:
        return key
    return value[-1] if isinstance(value, list) and value and isinstance(value[-1], str) else key


def _yaml_mappings(text: str) -> dict:
    """The block mappings of a YAML document as nested dicts; scalars stay strings (`null`
    becomes None). Small and tolerant, enough for the product's generated runtime.yaml:
    sequences and block scalars are skipped, flow collections stay text."""
    root: dict = {}
    stack: list[tuple[int, dict]] = [(-1, root)]
    skip: int | None = None
    for raw in text.splitlines():
        stripped = raw.strip()
        if not stripped or stripped.startswith("#") or stripped in ("---", "..."):
            continue
        indent = len(raw) - len(raw.lstrip(" "))
        if skip is not None:
            if indent > skip or (indent == skip and stripped.startswith("-")):
                continue
            skip = None
        if stripped == "-" or stripped.startswith("- "):
            skip = indent
            continue
        key, value = _yaml_entry(stripped)
        if key is None:
            continue
        while indent <= stack[-1][0]:
            stack.pop()
        parent = stack[-1][1]
        if value == "":
            parent[key] = child = {}
            stack.append((indent, child))
        elif value[:1] in ("|", ">"):
            parent[key], skip = None, indent
        else:
            parent[key] = _yaml_scalar(value)
    return root


def _yaml_entry(line: str) -> tuple[str | None, str]:
    """(key, raw value) of one `key: value` line; key None when the line is no entry."""
    if line[:1] in ("'", '"'):
        quote, end = line[0], 1
        while True:
            end = line.find(quote, end)
            if end < 0:
                return None, ""
            if quote == "'" and line[end + 1:end + 2] == "'":
                end += 2
            elif quote == '"' and line[end - 1] == "\\":
                end += 1
            else:
                break
        rest = line[end + 1:].lstrip()
        if not rest.startswith(":"):
            return None, ""
        return str(_yaml_scalar(line[:end + 1])), _uncomment(rest[1:].strip())
    if line.endswith(":"):
        return line[:-1].strip(), ""
    key, separator, value = line.partition(": ")
    return (key.strip(), _uncomment(value.strip())) if separator else (None, "")


def _uncomment(value: str) -> str:
    if value[:1] in ("'", '"'):
        return value
    return "" if value.startswith("#") else value.split(" #", 1)[0].rstrip()


def _yaml_scalar(value: str) -> str | None:
    if len(value) >= 2 and value[0] == value[-1] == "'":
        return value[1:-1].replace("''", "'")
    if len(value) >= 2 and value[0] == value[-1] == '"':
        try:
            return json.loads(value)
        except ValueError:
            return value[1:-1]
    return None if value in ("null", "Null", "NULL", "~") else value
