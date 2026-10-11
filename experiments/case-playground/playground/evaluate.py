"""Assess a finished run in its own container, after the run and apart from it.

  python -m playground assess --run DIR [--codex-auth PATH] [--claude-token PATH]
                              [--reviewers codex,claude|none] [--image IMAGE] [--force] [--keep-container]
                              [--fake-reviewers] [--exploratory]

Host side: stage the code and the evaluation files the run was pre-registered with
(registration.py: `evaluation/common`, `config.json` and the case's files except
`reference/`, `mutants/` and `validate.py`, read with `git archive` from the Git tree
recorded in host.json, never from the working tree) into `<run>/assessment/inputs`. The
reviewer models come from that tree's `config.json` and must differ from the arms'
models. A run without a registration or a reviewer clash is refused (exit 2) unless
`--exploratory`: it uses the working tree, records `rules.exploratory: true`, and
`compare` refuses the assessment. The commit of the judging code (playground/, the shared
and the case's public checks) is recorded as `evaluation.code`; local changes in it make
the assessment exploratory too. Then it launches one container `mpg-assess-<id>` from
the run's image and waits for it. Mounts:
the run folder read-only at /assess/run, the staged inputs read-only at /assess/in,
the reviewer credentials read-only under /assess/secrets and `<run>/assessment` at
/assess/out. Credentials are only checked for existence on the host, never read.

Container side (`--inside`), per station on scratch copies of the frozen
`immutable-main` (no `.git`) as the unprivileged user: the public checks again, the
holdouts (when the case has them), the wave's diff profile (from the snapshot bundles),
failure classification and product findings. Then the reviews: every wave by every
selected reviewer (`reviewers.py`). /assess stays root-only, so reviewers see only their
bundle and a read-only snapshot copy, never holdouts, check results or the other
reviewer's answer. Writes `report.json`, `report.md` and `product-findings.md`.

`--fake-reviewers` (smoke tests, no model call) runs `tests/fake_reviewer.py` in place
of both CLIs with throwaway credentials; the real logins are never mounted then.

Exit codes (outcome.py): 0 written; 2 refused (pre-registration, reviewer models); 10
the container did not exit 0 (assess-error.txt); failed holdouts or reviews inside a
written assessment do not change it.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import secrets
import shutil
import subprocess
import sys
import tempfile
import traceback
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Callable

from . import assess, codex_agent, lifecycle, methods, outcome, registration, reviewers
from . import report as report_module
from .lifecycle import read_text as _read_text

ROOT = Path(__file__).resolve().parent.parent  # experiments/case-playground
EVALUATION = ROOT / "evaluation"
CONTAINER_ROOT = Path("/assess")
SECRET_CODEX = "/assess/secrets/codex-auth.json"
SECRET_CLAUDE = "/assess/secrets/claude-token"
REVIEWER_LOGIN = reviewers.LOGIN_COPY.as_posix()  # the reviewers' working copy, copied out by a study
DEFAULT_CLAUDE_TOKEN = Path.home() / ".markitect-playground" / "claude-token"
COPY_PATTERNS = ("__pycache__", "*.pyc")
COPY_IGNORE = shutil.ignore_patterns(*COPY_PATTERNS)
# The reference and the mutants that validate the holdouts are never staged, never shown.
CASE_PATTERNS = ("reference", "mutants", "validate.py", *COPY_PATTERNS)
CASE_IGNORE = shutil.ignore_patterns(*CASE_PATTERNS)
GIT_TIMEOUT = 300
HOLDOUT_TIMEOUT = 600
HOLDOUT_MARGIN = 90  # seconds between holdout.py's own deadline and the hard timeout
EMPTY_TREE = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
DIFF_FLAGS = ("--no-renames", "--no-ext-diff", "--no-textconv", "--no-color")
NOT_AGENT_WORK = ("--", ":(top,exclude).study")  # the harness releases waves there
CATEGORY_NAMES = ("model", "code", "tests", "docs", "config/other")
CODE_SUFFIXES = (".py", ".go", ".js", ".mjs", ".cjs", ".ts", ".tsx", ".jsx", ".sh", ".bash", ".rb", ".java",
                 ".kt", ".rs", ".c", ".h", ".cc", ".cpp", ".cs", ".php", ".pl", ".sql", ".ps1", ".lua", ".swift")
DOC_SUFFIXES = (".md", ".markdown", ".rst", ".txt", ".adoc")
CLASS_ORDER = ("harness", "environment", "product")
ENVIRONMENT_TEXT = re.compile(
    r"\b(?:401|403|429|502|503|529)\b|unauthori[sz]ed|rate[ _-]?limit|usage limit|quota|authenticat"
    r"|refresh token|stream disconnected|reconnecting|connection (?:reset|refused|closed|error)|network error"
    r"|ECONNRESET|ETIMEDOUT|EAI_AGAIN|ENOTFOUND|overloaded|no space left|out of memory", re.I)
BLINDING_NOTE = ("Reviewers get no arm label, and the prompt template is the same for every arm and provider. "
                 "The repository itself can reveal the method (a `.markitect/` folder, Markitect guidance in "
                 "AGENTS.md), so the reviews are not blind to it.")
TEXT_CAP = 600
FAKE_REVIEWER = ROOT / "tests" / "fake_reviewer.py"


class AssessError(RuntimeError):
    """A clear, user-facing reason why an assessment could not run. `code` is the exit
    code: a refused input unless the environment failed (outcome.ENVIRONMENT)."""

    def __init__(self, message: str, code: int = outcome.INVALID) -> None:
        super().__init__(message)
        self.code = code


class GitError(RuntimeError):
    pass


def _utc() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds")


def _read_json(path: Path) -> Any:
    try:
        return json.loads(Path(path).read_text(encoding="utf-8"))
    except (OSError, UnicodeError, ValueError):
        return None


def _write_json(path: Path, value: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2, default=str) + "\n", encoding="utf-8",
                    newline="\n")


def _sha256(path: Path) -> str | None:
    return lifecycle.sha_file(path) if path.is_file() else None


def _short(text: Any, limit: int = TEXT_CAP) -> str:
    value = " ".join(str(text).split())
    return value if len(value) <= limit else value[:limit - 3] + "..."


def load_config(path: Path) -> dict:
    return check_config(_read_json(path), path)


def check_config(config: Any, where: Any) -> dict:
    if not isinstance(config, dict) or not isinstance(config.get("reviewers"), dict):
        raise AssessError(f"evaluation config missing or invalid: {where}")
    for name, cfg in config["reviewers"].items():
        if name not in reviewers.PROVIDERS or not isinstance(cfg, dict) or not cfg.get("model"):
            raise AssessError(f"evaluation config: reviewer {name!r} needs a model")
        reason = registration.alias_reason(cfg["model"])
        if reason:  # the config is pre-registered: the fix is a commit, never an override
            raise AssessError(f"evaluation config ({where}): reviewer {name}: {reason}; commit the full id before "
                              "the runs it judges")
    return config


def parse_reviewers(text: str | None) -> list[str]:
    if text is None:
        return list(reviewers.PROVIDERS)
    names = [part.strip() for part in text.split(",") if part.strip()]
    if names == ["none"]:
        return []
    unknown = [name for name in names if name not in reviewers.PROVIDERS]
    if unknown or not names or len(set(names)) != len(names):
        raise AssessError(f"--reviewers: expected a list of {', '.join(reviewers.PROVIDERS)} or 'none'")
    return [name for name in reviewers.PROVIDERS if name in names]


# --- stations, history, diff --------------------------------------------------------------

def discover_stations(results: Path) -> list[dict]:
    """Every station that ran (a station folder or snapshot), in order; never assumes a count."""
    audit = results / "audit"
    plan = (_read_json(audit / "run.json") or {}).get("stationPlan") or []
    numbers = {int(p.name[len("snapshot-S"):]) for p in audit.glob("snapshot-S*") if p.name[len("snapshot-S"):].isdigit()}
    numbers |= {int(p.name[1:]) for p in (results / "stations").glob("S*") if p.name[1:].isdigit()}
    found = []
    for number in sorted(numbers):
        snapshot = audit / f"snapshot-S{number}"
        record = _read_json(snapshot / "snapshot.json")
        commit = ((record or {}).get("immutableMain") or {}).get("commit")
        main = snapshot / "immutable-main"
        items = (record or {}).get("items") or (plan[number - 1] if number <= len(plan) else [])
        found.append({"number": number, "id": f"S{number}", "items": list(items), "record": record,
                      "snapshot": snapshot if record else None,
                      "main": main if commit and main.is_dir() else None, "mainCommit": commit,
                      "folder": results / "stations" / f"S{number}"})
    return found


def _git(repo: Path | None, *args: str, timeout: int = GIT_TIMEOUT) -> bytes:
    env = lifecycle.git_env({**os.environ, "GIT_CONFIG_GLOBAL": os.devnull, "GIT_CONFIG_NOSYSTEM": "1"})
    command = ["git", *(["-C", str(repo)] if repo is not None else []), *args]
    try:
        done = subprocess.run(command, capture_output=True, timeout=timeout, env=env, stdin=subprocess.DEVNULL)
    except (OSError, subprocess.SubprocessError) as exc:
        raise GitError(f"git {args[0]}: {exc}") from exc
    if done.returncode:
        raise GitError(f"git {args[0]} failed ({done.returncode}): "
                       f"{done.stderr.decode('utf-8', errors='replace').strip()[:500]}")
    return done.stdout


def build_history(audit: Path, target: Path) -> tuple[Path | None, list[str]]:
    """One bare repository holding every snapshot bundle under refs/mpg/<snapshot>/."""
    try:
        _git(None, "init", "-q", "--bare", str(target))
    except GitError as exc:
        return None, [str(exc)]
    errors = []
    for bundle in sorted(audit.glob("*/history.bundle")):
        try:
            _git(target, "fetch", "-q", "--no-tags", str(bundle), f"+refs/*:refs/mpg/{bundle.parent.name}/*")
        except GitError as exc:
            errors.append(f"{bundle.parent.name}: {exc}")
    return target, errors


def category(path: str) -> str:
    """model (.markitect/), tests, docs, code or config/other: a proxy for review effort."""
    parts = path.split("/")
    name = parts[-1].lower()
    if parts[0] == ".markitect":
        return "model"
    if (any(part.lower() in ("tests", "test") for part in parts[:-1]) or name.startswith("test_")
            or name.endswith(("_test.py", "_test.go", ".test.js", ".test.ts", ".spec.js", ".spec.ts"))):
        return "tests"
    if parts[0].lower() == "docs" or name.endswith(DOC_SUFFIXES):
        return "docs"
    if name.endswith(CODE_SUFFIXES):
        return "code"
    return "config/other"


def parse_numstat(raw: bytes) -> list[dict]:
    """`git diff --numstat -z --no-renames` records: binary files have null line counts."""
    entries = []
    for record in raw.split(b"\0"):
        if not record.strip():
            continue
        added, deleted, path = record.decode("utf-8", errors="surrogateescape").split("\t", 2)
        binary = added == "-" or deleted == "-"
        entries.append({"path": path, "category": category(path), "added": None if binary else int(added),
                        "deleted": None if binary else int(deleted)})
    return entries


def diff_profile(history: Path | None, base: str | None, head: str | None, out_dir: Path) -> dict:
    """Files and lines changed from `base` (previous main) to `head` (this main) by category,
    without the harness-owned `.study/`; the full text diff goes to `out_dir/wave.diff`."""
    result: dict[str, Any] = {"status": "error", "base": base, "head": head, "files": None, "added": None,
                              "deleted": None, "binaryFiles": None, "categories": None, "paths": [],
                              "diff": None, "error": None}
    if history is None or not head:
        result["error"] = "no history" if history is None else "no main commit"
        return result
    try:
        for commit in filter(None, (base, head)):
            _git(history, "cat-file", "-e", f"{commit}^{{commit}}")
        numstat = _git(history, "diff", "--numstat", "-z", *DIFF_FLAGS, base or EMPTY_TREE, head, *NOT_AGENT_WORK)
        text = _git(history, "diff", *DIFF_FLAGS, base or EMPTY_TREE, head, *NOT_AGENT_WORK)
    except GitError as exc:
        result["error"] = str(exc)
        return result
    entries = parse_numstat(numstat)
    categories = {name: {"files": 0, "added": 0, "deleted": 0} for name in CATEGORY_NAMES}
    for entry in entries:
        block = categories[entry["category"]]
        block["files"] += 1
        block["added"] += entry["added"] or 0
        block["deleted"] += entry["deleted"] or 0
    out_dir.mkdir(parents=True, exist_ok=True)
    (out_dir / "wave.diff").write_bytes(text)
    result.update(status="ok", files=len(entries), added=sum(e["added"] or 0 for e in entries),
                  deleted=sum(e["deleted"] or 0 for e in entries),
                  binaryFiles=sum(1 for e in entries if e["added"] is None), categories=categories,
                  paths=entries[:500], diff="wave.diff", diffBytes=len(text))
    return result


def setup_roles(history: Path | None, setup: dict | None, method: str | None) -> list[dict] | None:
    """Markitect's roles as its setup configured them: from the setup record, or for runs
    from before it was recorded, from `.markitect/runtime.yaml` at the setup commit."""
    roles = report_module._roles(setup)
    commit = (setup or {}).get("commit")
    if roles is not None or method != "markitect" or history is None or not commit:
        return roles
    try:
        text = _git(history, "show", f"{commit}:{methods.RUNTIME_PATH}").decode("utf-8", errors="replace")
    except GitError:
        return None
    return report_module._roles({"roles": methods.runtime_roles(text)})


# --- holdouts -----------------------------------------------------------------------------

def run_holdouts(candidate: Path, script: Path, station: int, out_dir: Path, timeout: float) -> dict:
    """`holdout.py --repo <scratch copy> --station N` as the unprivileged user."""
    out_dir.mkdir(parents=True, exist_ok=True)
    result: dict[str, Any] = {"passed": None, "total": None, "status": "error", "failures": [], "byItem": {},
                              "byRule": {}, "exitCode": None, "timedOut": False, "seconds": None, "error": None}
    root = Path(tempfile.mkdtemp(prefix="mpg-holdout-"))
    try:
        copy = assess.scratch_copy(candidate, root / "candidate")
        home = root / "home"
        (home / "tmp").mkdir(parents=True)
        codex_agent.give_to_agent(root, recursive=True)
        env = {"HOME": str(home), "TMPDIR": str(home / "tmp"), "GIT_CONFIG_GLOBAL": os.devnull,
               "GIT_CONFIG_NOSYSTEM": "1"}
        stdout, stderr = out_dir / "holdout.stdout.txt", out_dir / "holdout.stderr.txt"
        # The holdout's own deadline leaves room for one more app call and the JSON, so a
        # slow candidate loses only the checks not yet run, not the whole station.
        deadline = int(timeout - min(HOLDOUT_MARGIN, timeout / 4))
        run = codex_agent.run_as_agent([sys.executable, "-I", "-B", str(script), "--repo", str(copy),
                                        "--station", str(station), "--deadline", str(deadline)],
                                       copy, timeout, stdout, stderr, extra_env=env)
        result.update(exitCode=run.get("exitCode"), timedOut=bool(run.get("timedOut")), seconds=run.get("seconds"),
                      leftoverProcessesKilled=codex_agent.kill_all_agent_processes() + (run.get("killedOnTimeout") or 0))
    finally:
        reviewers.remove_tree(root)
    try:
        checks = json.loads(_read_text(stdout))["checks"]
        if not isinstance(checks, list) or not all(isinstance(c, dict) and "status" in c for c in checks):
            raise TypeError("checks must be a list of objects with a status")
    except (ValueError, KeyError, TypeError) as exc:
        result["error"] = "holdouts timed out" if result["timedOut"] else f"holdout output could not be parsed: {exc}"
        return result
    result["checks"] = checks
    # PASS and FAIL are verdicts on the candidate; anything else (ERROR) means the holdout
    # could not judge, so it is counted apart and never as a candidate failure.
    judged = [c for c in checks if c["status"] in ("PASS", "FAIL")]
    result["passed"], result["total"] = sum(1 for c in judged if c["status"] == "PASS"), len(judged)
    result["errors"] = len(checks) - len(judged)
    result["failures"] = [{"id": c.get("id"), "status": c.get("status"), "item": c.get("item"),
                           "rule": c.get("rule"), "detail": _short(c.get("detail") or "")}
                          for c in checks if c["status"] != "PASS"]
    for key, field in (("byItem", "item"), ("byRule", "rule")):
        groups: dict[str, dict] = {}
        for check in checks:
            group = groups.setdefault(str(check.get(field) or "-"), {"passed": 0, "total": 0, "errors": 0})
            if check["status"] in ("PASS", "FAIL"):
                group["total"] += 1
                group["passed"] += check["status"] == "PASS"
            else:
                group["errors"] += 1
        result[key] = dict(sorted(groups.items()))
    crashed = result["exitCode"] != 0
    result["status"] = ("error" if crashed else "fail" if result["passed"] < result["total"]
                        else "error" if result["errors"] else "pass")
    if crashed:
        result["error"] = f"holdout.py exited {result['exitCode']}"
    elif result["errors"]:
        result["error"] = f"{result['errors']} holdout(s) could not judge (ERROR)"
    return result


# --- agent events: product findings and classification ------------------------------------

def _compact(value: Any, limit: int = 300) -> str:
    try:
        return _short(json.dumps(value, ensure_ascii=False, separators=(",", ":")), limit)
    except (TypeError, ValueError):
        return _short(value, limit)


def _detail_excerpt(data: Any, depth: int = 0) -> str | None:
    """The first non-empty findings/errors list in a product answer, compacted."""
    if depth > 4:
        return None
    if isinstance(data, dict):
        for key in ("findings", "errors", "violations", "diagnostics", "issues", "problems", "blockers",
                    "openDecisions"):
            if isinstance(data.get(key), list) and data[key]:
                return _compact(data[key][:3], TEXT_CAP)
        for value in data.values():
            found = _detail_excerpt(value, depth + 1)
            if found:
                return found
    return None


def _content_text(content: Any) -> str:
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        return "".join(part.get("text", "") for part in content if isinstance(part, dict))
    return ""


def _mcp_error(item: dict) -> dict:
    """Code, message and details of a failed MCP call (Codex item)."""
    error = item.get("error")
    if isinstance(error, dict) and error.get("message"):
        return {"code": None, "message": _short(error["message"]), "details": None}
    result = item.get("result") if isinstance(item.get("result"), dict) else {}
    structured = result.get("structured_content") or result.get("structuredContent")
    text = _content_text(result.get("content"))
    if not isinstance(structured, dict):
        try:
            structured = json.loads(text)
        except ValueError:
            structured = None
    diagnostic = structured.get("diagnostic") if isinstance(structured, dict) else None
    if isinstance(diagnostic, dict):
        return {"code": diagnostic.get("code"), "message": _short(diagnostic.get("message") or ""),
                "details": _detail_excerpt(structured.get("data"))}
    return {"code": None, "message": _short(text or "failed without a message"), "details": None}


def analyze_events(folder: Path) -> dict:
    """Markitect MCP calls and commands plus error messages in a station's event streams
    (`codex exec --json` items and Claude Code stream-json messages)."""
    found: dict[str, Any] = {"markitectMcpCalls": 0, "failedMcpCalls": [], "unfinishedMcpCalls": [],
                             "productCommands": [], "errorMessages": []}
    started: dict[str, dict] = {}
    commands: dict[str, str] = {}
    finished: set[str] = set()
    for path in sorted(folder.glob("*.jsonl")) if folder.is_dir() else []:
        for line in _read_text(path).splitlines():
            try:
                event = json.loads(line)
            except ValueError:
                continue
            if isinstance(event, dict):
                _codex_event(event, found, started, finished)
                _claude_event(event, found, started, commands, finished)
    found["unfinishedMcpCalls"] = [call for key, call in started.items() if key not in finished]
    return found


def _codex_event(event: dict, found: dict, started: dict, finished: set) -> None:
    kind, item = event.get("type"), event.get("item")
    if kind in ("error", "turn.failed"):
        error = event.get("error")
        message = event.get("message") or (error.get("message") if isinstance(error, dict) else error)
        found["errorMessages"].append(_short(message or _compact(event)))
        return
    if not isinstance(item, dict):
        return
    key = f"codex:{item.get('id')}"
    if item.get("type") == "mcp_tool_call" and item.get("server") == "markitect":
        if kind == "item.started":
            started[key] = {"tool": item.get("tool"), "arguments": _compact(item.get("arguments"))}
        elif kind == "item.completed":
            finished.add(key)
            found["markitectMcpCalls"] += 1
            result = item.get("result") if isinstance(item.get("result"), dict) else {}
            if item.get("status") == "failed" or item.get("error") or result.get("is_error") or result.get("isError"):
                found["failedMcpCalls"].append({"tool": item.get("tool"), "arguments": _compact(item.get("arguments")),
                                                **_mcp_error(item)})
    elif kind == "item.completed" and item.get("type") == "command_execution":
        command = str(item.get("command") or "")
        if "markitect" in command.lower() and item.get("exit_code") not in (0, None):
            output = str(item.get("aggregated_output") or "")
            found["productCommands"].append({"command": _short(command, 300), "exitCode": item.get("exit_code"),
                                             "output": _short(output[-1500:])})
    elif kind == "item.completed" and item.get("type") == "error":
        found["errorMessages"].append(_short(item.get("message") or _compact(item)))


def _claude_event(event: dict, found: dict, started: dict, commands: dict, finished: set) -> None:
    kind = event.get("type")
    if kind == "result" and event.get("is_error"):
        found["errorMessages"].append(_short(event.get("result") or event.get("subtype") or "error result"))
        return
    message = event.get("message")
    if kind not in ("assistant", "user") or not isinstance(message, dict) or not isinstance(message.get("content"), list):
        return
    for part in message["content"]:
        if not isinstance(part, dict):
            continue
        key = f"claude:{part.get('id') or part.get('tool_use_id')}"
        if part.get("type") == "tool_use":
            name = str(part.get("name") or "")
            if name.startswith("mcp__markitect__"):
                started[key] = {"tool": name[len("mcp__markitect__"):], "arguments": _compact(part.get("input"))}
            elif name == "Bash" and "markitect" in str((part.get("input") or {}).get("command", "")).lower():
                commands[key] = str(part["input"]["command"])
        elif part.get("type") == "tool_result":
            text = _content_text(part.get("content"))
            if key in started:
                finished.add(key)
                found["markitectMcpCalls"] += 1
                if part.get("is_error"):
                    call = started[key]
                    try:
                        diagnostic = (json.loads(text) or {}).get("diagnostic")
                    except (ValueError, AttributeError):
                        diagnostic = None
                    found["failedMcpCalls"].append({
                        "tool": call["tool"], "arguments": call["arguments"],
                        "code": diagnostic.get("code") if isinstance(diagnostic, dict) else None,
                        "message": _short(diagnostic.get("message") if isinstance(diagnostic, dict) else text),
                        "details": None})
            elif key in commands and part.get("is_error"):
                found["productCommands"].append({"command": _short(commands[key], 300), "exitCode": None,
                                                 "output": _short(text[-1500:])})


def classify_station(station: dict, agent: dict | None, analysis: dict, stderr: str) -> dict:
    """`harness`, `environment`, `product` or `none` with the reason; a cause only decides
    the class when the station failed (exit, timeout, no session, harness error)."""
    folder = station["folder"]
    causes = []
    harness_files = sorted(p.name for p in folder.glob("*-error.txt")) if folder.is_dir() else []
    if harness_files:
        causes.append({"class": "harness", "reason": f"harness error files: {', '.join(harness_files)}"})
    if station["record"] is None:
        causes.append({"class": "harness", "reason": "no station snapshot"})
    if agent is None:
        causes.append({"class": "harness", "reason": "no agent record"})
    texts = analysis["errorMessages"] + [line for line in stderr.splitlines() if line.strip()]
    environment = next((text for text in texts if ENVIRONMENT_TEXT.search(text)), None)
    if environment:
        causes.append({"class": "environment", "reason": _short(environment, 300)})
    agent = agent or {}
    if agent and not agent.get("sessionId"):
        causes.append({"class": "environment", "reason": "the agent never produced a session id (login or network)"})
    for call in analysis["unfinishedMcpCalls"]:
        causes.append({"class": "product", "reason": f"MCP call markitect/{call['tool']} did not finish"})
    marked = next((text for text in analysis["errorMessages"] if "markitect" in text.lower()), None)
    if marked:
        causes.append({"class": "product", "reason": _short(marked, 300)})
    failed = (not agent or agent.get("exitCode") != 0 or bool(agent.get("timedOut")) or not agent.get("sessionId")
              or bool(harness_files) or station["record"] is None)
    if not failed:
        return {"class": "none", "reason": "station completed", "failed": False, "causes": causes}
    for name in CLASS_ORDER:
        hit = next((cause for cause in causes if cause["class"] == name), None)
        if hit:
            return {"class": name, "reason": hit["reason"], "failed": True, "causes": causes}
    outcome = "timed out" if agent.get("timedOut") else f"exit {agent.get('exitCode')}"
    return {"class": "none", "reason": f"agent outcome ({outcome}), no infrastructure cause found",
            "failed": True, "causes": causes}


def classify_run(host_record: dict, runner_state: dict, setup: dict | None, results: Path,
                 stations: list[dict]) -> dict:
    causes = []
    status = host_record.get("status")
    if status and status != "completed":
        name = ("product" if host_record.get("failureClass") == "product"
                else "environment" if status in ("start-failed", "wait-failed", "setup-failed") else "harness")
        causes.append({"class": name, "reason": _short(f"host status {status}: {host_record.get('error')}", 300)})
    if host_record.get("containerExitCode") == outcome.KILLED:
        if host_record.get("oomKilled") is True:
            causes.append({"class": "environment", "reason": "the run container was killed (exit 137) and Docker "
                                                             "reports it OOMKilled"})
        else:
            causes.append({"class": "harness", "reason": "the run container was killed (exit 137) and Docker does "
                                                         "not report it OOMKilled"})
    if (results / "runner-error.txt").is_file():
        causes.append({"class": "harness", "reason": "runner error (results/runner-error.txt)"})
    if (setup or {}).get("status") == "blocked":
        causes.append({"class": "product", "reason": _short(f"method setup blocked: {setup.get('error')}", 300)})
    stop = runner_state.get("stopReason") or ""
    if stop and runner_state.get("stopCategory") in CLASS_ORDER:  # the runner's own verdict
        causes.append({"class": runner_state["stopCategory"], "reason": _short(stop, 300)})
    elif re.search(r"auth|session id|could not be started", stop):  # runs before stop categories
        causes.append({"class": "environment", "reason": _short(stop, 300)})
    elif re.search(r"snapshot|releasing", stop):
        causes.append({"class": "harness", "reason": _short(stop, 300)})
    for entry in stations:
        if entry["classification"]["class"] != "none":
            causes.append({"class": entry["classification"]["class"],
                           "reason": f"{entry['station']}: {entry['classification']['reason']}"})
    for name in CLASS_ORDER:
        hit = next((cause for cause in causes if cause["class"] == name), None)
        if hit:
            return {"class": name, "reason": hit["reason"], "causes": causes}
    return {"class": "none", "reason": f"stopped: {stop}" if stop else "run completed", "causes": causes}


# --- the assessment (container side) --------------------------------------------------------

def _stage_tools(in_dir: Path, case_eval: Path | None, target: Path) -> tuple[Path, Path | None]:
    """Agent-readable copies of the public checks and the holdouts (removed before reviews)."""
    target.mkdir(parents=True)
    shutil.copytree(in_dir / "cases", target / "cases", ignore=COPY_IGNORE)
    holdout = None
    if case_eval is not None and (case_eval / "holdout.py").is_file():
        shutil.copytree(case_eval, target / "holdout",
                        ignore=shutil.ignore_patterns("ground-truth.json", "reference", "mutants", "validate.py",
                                                      "__pycache__", "*.pyc"))
        holdout = target / "holdout" / "holdout.py"
    _readable(target)
    return target, holdout


def _readable(root: Path) -> None:
    """Root-owned and read-only for the unprivileged user (container only)."""
    if not codex_agent.container_mode():
        return
    for current, dirs, files in os.walk(root):
        os.chmod(current, 0o755)
        for name in files:
            path = os.path.join(current, name)
            if not os.path.islink(path):
                os.chmod(path, (os.stat(path).st_mode & 0o755) | 0o444)


def _station_phase(station: dict, ctx: dict) -> dict:
    """Checks, holdouts, diff, classification and product findings of one station."""
    folder = ctx["out"] / "stations" / station["id"]
    folder.mkdir(parents=True, exist_ok=True)
    agent = _read_json(station["folder"] / "agent.json")
    during = _read_json(station["folder"] / "checks.json") or {}
    entry: dict[str, Any] = {"station": station["id"], "items": station["items"], "mainCommit": station["mainCommit"],
                             "baseCommit": ctx["base"], "artifacts": f"stations/{station['id']}"}
    if station["main"] is not None:
        try:
            checks = assess.station(station["main"], ctx["case"], station["number"], ctx["tools"], folder / "checks")
        except Exception:
            checks = {"passed": None, "total": None, "status": "error", "error": traceback.format_exc()[-2000:]}
    else:
        checks = {"passed": None, "total": None, "status": "error", "error": "no merged main in the snapshot"}
    entry["publicChecks"] = {"passed": checks.get("passed"), "total": checks.get("total"),
                             "status": checks.get("status"), "error": checks.get("error"),
                             "duringRun": {"passed": during.get("passed"), "total": during.get("total")}}
    holdouts = None
    if ctx["holdout"] is not None:
        if station["main"] is not None:
            try:
                holdouts = run_holdouts(station["main"], ctx["holdout"], station["number"], folder / "holdouts",
                                        ctx["holdoutTimeout"])
            except Exception:
                holdouts = {"passed": None, "total": None, "status": "error", "failures": [],
                            "error": traceback.format_exc()[-2000:]}
        else:
            holdouts = {"passed": None, "total": None, "status": "error", "failures": [], "error": "no merged main"}
        _write_json(folder / "holdouts" / "holdouts.json", holdouts)
    entry["holdouts"] = None if holdouts is None else {key: holdouts.get(key) for key in
                                                      ("passed", "total", "errors", "status", "failures", "byItem",
                                                       "byRule", "error")}
    diff = diff_profile(ctx["history"], ctx["base"], station["mainCommit"], folder)
    entry["diff"] = {key: value for key, value in diff.items() if key != "paths"}
    analysis = analyze_events(station["folder"])
    stderr = _read_text(station["folder"] / "stderr.log")
    analysis["stderrProductLines"] = [_short(line) for line in stderr.splitlines()
                                      if "markitect" in line.lower() and re.search(r"error|panic|fatal|fail", line, re.I)][:20]
    entry["classification"] = classify_station(station, agent, analysis, stderr)
    entry["productFindings"] = {"markitectMcpCalls": analysis["markitectMcpCalls"],
                                "failedMcpCalls": len(analysis["failedMcpCalls"]),
                                "unfinishedMcpCalls": len(analysis["unfinishedMcpCalls"]),
                                "productCommands": len(analysis["productCommands"])}
    entry["groundTruthObligations"] = reviewers.obligation_count(
        reviewers.wave_ground_truth(ctx["groundTruth"], station["number"]))
    _write_json(folder / "assessment.json", {**entry, "checks": checks, "diffPaths": diff.get("paths"),
                                              "events": analysis})
    if station["mainCommit"]:
        ctx["base"] = station["mainCommit"]
    ctx["analyses"][station["id"]] = analysis
    return entry


def _review_phase(station: dict, entry: dict, ctx: dict) -> None:
    folder = ctx["out"] / "stations" / station["id"]
    records: dict[str, dict] = {}
    if not ctx["reviewers"]:
        pass
    elif station["main"] is None:
        records = {name: {"provider": name, "status": "error", "error": "no merged main to review"}
                   for name in ctx["reviewers"]}
    else:
        root = _review_root()
        try:
            repo = assess.scratch_copy(station["main"], root / "repo")
            bundle = root / "bundle"
            wave = {"station": station["number"], "items": station["items"],
                    "earlierItems": [item for earlier in ctx["stations"] if earlier["number"] < station["number"]
                                     for item in earlier["items"]],
                    "itemTexts": reviewers.extract_items(ctx["backlog"], station["items"], ctx["allItems"]),
                    "rules": ctx["rules"],
                    "groundTruth": reviewers.wave_ground_truth(ctx["groundTruth"], station["number"]),
                    "lastMessage": _read_text(station["folder"] / "last-message.txt"),
                    "diff": _read_text(folder / "wave.diff"), "backlog": ctx["backlog"],
                    "repo": str(repo), "bundle": str(bundle)}
            prompt = reviewers.compose_prompt(ctx["template"], wave, ctx["promptMaxBytes"])
            reviewers.write_bundle(bundle, wave, prompt, ctx["schema"])
            shutil.copytree(bundle, folder / "review-input", dirs_exist_ok=True)
            _readable(root)
            for name in ctx["reviewers"]:
                record = reviewers.run_reviewer(
                    name, ctx["config"]["reviewers"][name], prompt=prompt, schema=ctx["schema"],
                    schema_path=bundle / "reviewer-schema.json", repo_dir=repo, bundle_dir=bundle,
                    out_dir=folder / "reviewers" / name, codex_auth=ctx["codexAuth"],
                    claude_token=ctx["claudeToken"], executable=ctx["executables"].get(name),
                    login_copy=ctx["codexAuth"])
                _write_json(folder / "reviewers" / name / "review.json", record)
                records[name] = record
        finally:
            reviewers.remove_tree(root)
    entry["reviewers"] = {name: reviewers.summarize(record) for name, record in records.items()}
    entry["agreement"] = reviewers.agreement(records)
    entry["escalations"] = {name: {"needed": summary["byCategory"]["escalation_needed"],
                                   "unneeded": summary["byCategory"]["escalation_unneeded"]}
                            if summary.get("byCategory") else None for name, summary in entry["reviewers"].items()}
    ctx["findings"][station["id"]] = {name: record.get("findings") or [] for name, record in records.items()
                                      if record.get("status") == "ok"}


def _review_root() -> Path:
    """A fixed path in the container, so the prompt text does not differ by temp names."""
    if codex_agent.container_mode():
        root = Path(tempfile.gettempdir()) / "mpg-review"
        reviewers.remove_tree(root)
        root.mkdir(mode=0o755)
        return root
    return Path(tempfile.mkdtemp(prefix="mpg-review-"))


def assess_run(run_dir: Path, evaluation_dir: Path, out_dir: Path, *, reviewer_names: list[str] | tuple = (),
               codex_auth: Path | None = None, claude_token_file: Path | None = None, in_dir: Path | None = None,
               executables: dict | None = None, evaluation: dict | None = None, image: str | None = None,
               login_copy: Path | None = None, exploratory: bool = False) -> dict:
    """Assess every station of one run folder; writes report.json, report.md and
    product-findings.md into `out_dir` and returns the report. The Codex reviewers share
    one working copy of `codex_auth` (at `login_copy`; default reviewers.LOGIN_COPY in
    the container, else a temporary one), which carries a refreshed login between waves.
    `evaluation` says where `evaluation_dir` came from ({"source": "registered" or
    "working-tree", "tree", "commit", "dirty"}); the report's `rules` say whether the
    assessment counts as pre-registered (registered source, neither it nor the run
    exploratory) and whether the reviewer models differ from the arms' models."""
    run_dir, evaluation_dir, out_dir = (Path(p).resolve() for p in (run_dir, evaluation_dir, out_dir))
    out_dir.mkdir(parents=True, exist_ok=True)
    results = run_dir / "results"
    host_record = _read_json(run_dir / "host.json") or {}
    run_report = _read_json(results / "report.json") or {}
    runner_state = _read_json(results / "runner.json") or {}
    manifest = host_record.get("manifest") or run_report.get("manifest") or runner_state.get("manifest")
    if not isinstance(manifest, dict) or not manifest.get("case"):
        raise AssessError(f"no run manifest in {run_dir}")
    case = manifest["case"]
    in_dir = Path(in_dir or run_dir / "inputs").resolve()
    config = load_config(evaluation_dir / "config.json")
    case_eval = evaluation_dir / case if (evaluation_dir / case).is_dir() else None
    truth_path = case_eval / "ground-truth.json" if case_eval else None
    ground_truth = _read_json(truth_path) if truth_path and truth_path.is_file() else None
    stations = discover_stations(results)
    setup = _read_json(results / "setup" / "setup.json")
    plan = (_read_json(results / "audit" / "run.json") or {}).get("stationPlan") or []
    case_inputs = in_dir / "cases" / case
    rules = {name: _read_text(path) for name, path in (("README.md", case_inputs / "README.md"),
                                                       ("AGENTS.md", in_dir / "cases" / "common" / "AGENTS.md"),
                                                       ("QUALITY.md", in_dir / "cases" / "common" / "QUALITY.md"))
             if path.is_file()}
    names = [name for name in reviewer_names if name in config["reviewers"]]
    evaluation = dict(evaluation or {})
    exploratory = bool(exploratory or (host_record.get("rules") or {}).get("exploratory") is True)
    clashes = registration.clashes(config, names, registration.arm_models(manifest, run_report.get("roles")))
    verdict = {"preRegistered": evaluation.get("source") == "registered" and not exploratory,
               "exploratory": exploratory, "reviewersIndependent": not clashes, "reviewerClashes": clashes}
    token = None
    if "claude" in names and claude_token_file is not None and Path(claude_token_file).is_file():
        token = Path(claude_token_file).read_text(encoding="utf-8").strip() or None
    login = reviewers.working_login(codex_auth, login_copy) if "codex" in names else None
    temporary_login = login is not None and login_copy is None and not codex_agent.container_mode()
    ctx: dict[str, Any] = {
        "out": out_dir, "case": case, "stations": stations, "groundTruth": ground_truth,
        "holdoutTimeout": float(config.get("holdoutTimeoutSeconds") or HOLDOUT_TIMEOUT),
        "base": (setup or {}).get("commit") or ((_read_json(results / "audit" / "run.json") or {}).get("seed") or {}).get("mainCommit"),
        "analyses": {}, "findings": {}, "reviewers": names, "config": config,
        "template": _read_text(evaluation_dir / "common" / "reviewer-prompt.md"),
        "schema": _read_json(evaluation_dir / "common" / "reviewer-schema.json"),
        "promptMaxBytes": int(config.get("promptMaxBytes") or reviewers.DEFAULT_PROMPT_MAX_BYTES),
        "backlog": _read_text(case_inputs / "BACKLOG.md"), "rules": rules,
        "allItems": [item for wave in plan for item in wave] or [i for s in stations for i in s["items"]],
        "codexAuth": login, "claudeToken": token,
        "executables": executables or {}, "roles": None, "verdict": verdict,
    }
    if names and not isinstance(ctx["schema"], dict):
        raise AssessError("reviewer schema missing or invalid")
    entries, errors = [], []
    work = Path(tempfile.mkdtemp(prefix="mpg-evaluate-"))
    try:
        ctx["history"], history_errors = build_history(results / "audit", work / "history")
        errors += history_errors
        ctx["roles"] = setup_roles(ctx["history"], setup, manifest.get("method"))
        tools = Path(tempfile.mkdtemp(prefix="mpg-eval-tools-"))
        try:
            ctx["tools"], ctx["holdout"] = _stage_tools(in_dir, case_eval, tools / "in")
            os.chmod(tools, 0o755)
            entries = [_station_phase(station, ctx) for station in stations]
        finally:
            reviewers.remove_tree(tools)  # holdouts never stay where a reviewer could read them
        versions = {name: reviewers.cli_version(name, ctx["executables"].get(name)) for name in names}
        for station, entry in zip(stations, entries):
            _review_phase(station, entry, ctx)
    finally:
        reviewers.remove_tree(work)
        if temporary_login:
            reviewers.remove_tree(login.parent)
    report = _report(run_dir, out_dir, manifest, host_record, run_report, runner_state, setup, results, entries,
                     ctx, evaluation_dir, case_eval, evaluation, image, versions, errors)
    _write_json(out_dir / "report.json", report)
    (out_dir / "report.md").write_text(render_report(report, ctx["findings"]), encoding="utf-8", newline="\n")
    (out_dir / "product-findings.md").write_text(
        render_product_findings(report, setup, results, ctx["analyses"]), encoding="utf-8", newline="\n")
    return report


def _report(run_dir, out_dir, manifest, host_record, run_report, runner_state, setup, results, entries, ctx,
            evaluation_dir, case_eval, evaluation, image, versions, errors) -> dict:
    agent = manifest.get("agent") or {}
    files = {"groundTruth": case_eval / "ground-truth.json" if case_eval else None,
             "holdout": case_eval / "holdout.py" if case_eval else None,
             "reviewerPrompt": evaluation_dir / "common" / "reviewer-prompt.md",
             "reviewerSchema": evaluation_dir / "common" / "reviewer-schema.json",
             "config": evaluation_dir / "config.json"}
    names = ctx["reviewers"]
    totals_reviewers = {}
    for name in names:
        summaries = [e["reviewers"].get(name) or {} for e in entries]
        ok = [s for s in summaries if s.get("status") == "ok"]
        totals_reviewers[name] = {
            "wavesReviewed": len(ok), "waves": len(summaries),
            "findings": sum(s["findings"] for s in ok),
            "byCategory": {c: sum(s["byCategory"][c] for s in ok) for c in reviewers.CATEGORIES},
            "obligations": {"covered": sum(s["obligations"]["covered"] for s in ok),
                            "total": sum(s["obligations"]["total"] for s in ok)},
            "loginRefreshed": any(s.get("loginRefreshed") for s in summaries)}
    holdout_entries = [e["holdouts"] for e in entries if e.get("holdouts")]
    return {
        "schema": 1, "kind": "assessment", "assessedAt": _utc(),
        "run": {"id": manifest.get("id"), "case": manifest.get("case"), "method": manifest.get("method"),
                "outerProvider": agent.get("kind"), "agent": agent, "status": run_report.get("status"),
                "exitCode": run_report.get("exitCode"), "stopReason": runner_state.get("stopReason"),
                "hostStatus": host_record.get("status"), "stationsRun": len(entries),
                "fairness": run_report.get("fairness"), "versions": run_report.get("versions"),
                "totals": run_report.get("totals"), "setupStatus": (setup or {}).get("status"),
                # Top-level in the run report; runs from before it existed get it from the manifest.
                "stratum": run_report.get("stratum") or report_module._stratum(manifest),
                "roles": run_report.get("roles") or ctx["roles"],
                "classification": run_report.get("classification")},
        "evaluation": {"source": evaluation.get("source"), "tree": evaluation.get("tree"),
                       "commit": evaluation.get("commit"), "dirty": evaluation.get("dirty"),
                       "code": evaluation.get("code"),
                       "groundTruth": ctx["groundTruth"] is not None, "holdouts": ctx["holdout"] is not None,
                       "files": {key: ({"path": path.relative_to(evaluation_dir).as_posix(), "sha256": _sha256(path)}
                                       if path is not None and path.is_file() else None)
                                 for key, path in files.items()}},
        "rules": ctx["verdict"],
        "reviewers": {name: {"model": ctx["config"]["reviewers"][name].get("model"),
                             "effort": ctx["config"]["reviewers"][name].get("effort"),
                             "timeoutSeconds": ctx["config"]["reviewers"][name].get("timeoutSeconds"),
                             "version": versions.get(name)} for name in names},
        "image": image,
        "blinding": BLINDING_NOTE,
        "stations": entries,
        "classification": classify_run(host_record, runner_state, setup, results, entries),
        "totals": {
            "publicChecks": _sum_pairs([e["publicChecks"] for e in entries]),
            "holdouts": ({**_sum_pairs(holdout_entries), "errors": _sum_known([e.get("errors") for e in holdout_entries])}
                         if holdout_entries else None),
            "reviewers": totals_reviewers,
            "diff": {"files": _sum_known([(e["diff"] or {}).get("files") for e in entries]),
                     "added": _sum_known([(e["diff"] or {}).get("added") for e in entries]),
                     "deleted": _sum_known([(e["diff"] or {}).get("deleted") for e in entries]),
                     "modelLines": _sum_known([_category_lines(e["diff"], "model") for e in entries])},
            "failedMcpCalls": sum(e["productFindings"]["failedMcpCalls"] for e in entries),
        },
        "errors": errors,
    }


def _sum_known(values: list) -> int | None:
    if not values or any(not isinstance(v, int) for v in values):
        return None
    return sum(values)


def _sum_pairs(blocks: list[dict]) -> dict:
    return {"passed": _sum_known([b.get("passed") for b in blocks]),
            "total": _sum_known([b.get("total") for b in blocks])}


def _category_lines(diff: dict | None, name: str) -> int | None:
    block = ((diff or {}).get("categories") or {}).get(name)
    return None if block is None else block["added"] + block["deleted"]


# --- Markdown ------------------------------------------------------------------------------

def _fmt(value: Any) -> str:
    if value is None:
        return "n/a"
    if isinstance(value, bool):
        return "yes" if value else "no"
    if isinstance(value, float):
        return f"{value:,.1f}"
    if isinstance(value, int):
        return f"{value:,}"
    return str(value)


def _pair(block: dict | None, status: bool = True) -> str:
    if not block:
        return "n/a"
    text = f"{_fmt(block.get('passed'))}/{_fmt(block.get('total'))}"
    if block.get("errors"):
        text += f" +{block['errors']} not judged"
    return text + (f" ({block['status']})" if status and block.get("status") else "")


def _diff_cell(diff: dict | None, category_name: str | None = None) -> str:
    if not diff or diff.get("status") != "ok":
        return "n/a"
    block = diff["categories"][category_name] if category_name else diff
    return f"{block['files']} (+{block['added']}/-{block['deleted']})"


def _reviewer_cell(summary: dict | None) -> str:
    if not summary:
        return "n/a"
    if summary.get("status") != "ok":
        return str(summary.get("status"))
    high = (summary.get("bySeverity") or {}).get("high", 0)
    return f"{summary['findings']} ({high} high)"


def _obligations_cell(summary: dict | None) -> str:
    block = (summary or {}).get("obligations")
    return f"{block['covered']}/{block['total']}" if block else "n/a"


def _counts(block: dict | None) -> str:
    items = [f"{name} {count}" for name, count in (block or {}).items() if count]
    return ", ".join(items) or "none"


def render_report(report: dict, findings: dict) -> str:
    run, evaluation = report["run"], report["evaluation"]
    agent = run.get("agent") or {}
    names = list(report["reviewers"])
    lines = [f"# Assessment of run {run.get('id')}", "",
             f"Case **{_fmt(run.get('case'))}**, method **{_fmt(run.get('method'))}**, outer agent "
             f"{_fmt(run.get('outerProvider'))} (model {_fmt(agent.get('model'))}, effort {_fmt(agent.get('effort'))}). "
             f"Run status {_fmt(run.get('status'))}, {run['stationsRun']} station(s).",
             f"Classification: **{report['classification']['class']}** ({report['classification']['reason']}).", ""]
    file_notes = [f"{key} {(value or {}).get('sha256') or 'none'}" for key, value in evaluation["files"].items()]
    rules = report.get("rules") or {}
    clashes = rules.get("reviewerClashes") or []
    lines += [f"- Evaluation files: {_evaluation_source(evaluation)}; SHA-256: {'; '.join(file_notes)}.",
              *_judging_code_line(evaluation.get("code")),
              f"- Rules: pre-registered {_fmt(rules.get('preRegistered'))}; exploratory {_fmt(rules.get('exploratory'))}; "
              f"reviewer models differ from the arms' models {_fmt(rules.get('reviewersIndependent'))}"
              + (f" ({registration.describe(clashes)})" if clashes else "")
              + ("." if rules.get("preRegistered") and rules.get("reviewersIndependent")
                 else "; compare refuses this assessment."),
              f"- Ground truth: {_fmt(evaluation['groundTruth'])}; holdouts: {_fmt(evaluation['holdouts'])}.",
              "- Reviewers: " + ("; ".join(f"{name} {cfg['model']} (effort {_fmt(cfg.get('effort'))}, "
                                           f"{_fmt(cfg.get('version'))})" for name, cfg in report["reviewers"].items())
                                 or "none") + ".",
              f"- Image: {_fmt(report.get('image'))}.",
              f"- Stratum: {_fmt(run.get('stratum'))}"
              + (f"; Markitect roles: {'; '.join(report_module._role_text(r) for r in run['roles'])}"
                 if run.get("roles") else "") + ".",
              f"- Blinding: {report['blinding']}", ""]
    header = ["Wave", "Items", "Public checks", "Holdouts", "Diff files (+/-)", "Model files (+/-)"]
    header += [f"Findings {name}" for name in names] + (["Shared"] if len(names) == 2 else [])
    header += [f"Obligations {name}" for name in names] + ["Class"]
    lines += ["| " + " | ".join(header) + " |", "|" + "---|" * len(header)]
    for entry in report["stations"]:
        row = [entry["station"], ", ".join(entry["items"]) or "-", _pair(entry["publicChecks"]),
               _pair(entry.get("holdouts")), _diff_cell(entry["diff"]), _diff_cell(entry["diff"], "model")]
        row += [_reviewer_cell(entry["reviewers"].get(name)) for name in names]
        if len(names) == 2:
            row.append(_fmt((entry.get("agreement") or {}).get("both")))
        row += [_obligations_cell(entry["reviewers"].get(name)) for name in names]
        row.append(entry["classification"]["class"])
        lines.append("| " + " | ".join(row) + " |")
    if not report["stations"]:
        lines.append("| (no station ran) |" + " |" * (len(header) - 1))
    for entry in report["stations"]:
        lines += ["", f"## {entry['station']} ({', '.join(entry['items']) or 'no items'})", ""]
        checks = entry["publicChecks"]
        lines.append(f"- Public checks {_pair(checks)}; during the run {_pair(checks['duringRun'], False)}"
                     + (f"; error: {_short(checks['error'], 200)}" if checks.get("error") else "") + ".")
        holdouts = entry.get("holdouts")
        if holdouts:
            lines.append(f"- Holdouts {_pair(holdouts)}" + (f"; error: {holdouts['error']}" if holdouts.get("error") else "") + ".")
            for failure in holdouts.get("failures") or []:
                lines.append(f"  - {failure.get('status')} `{failure.get('id')}` (item {_fmt(failure.get('item'))}, "
                             f"rule {_fmt(failure.get('rule'))}): {failure.get('detail') or ''}")
        diff = entry["diff"]
        if diff and diff.get("status") == "ok":
            parts = [f"{name} {_diff_cell(diff, name)}" for name in CATEGORY_NAMES if diff["categories"][name]["files"]]
            lines.append(f"- Diff {_diff_cell(diff)} from {str(diff['base'])[:12]} to {str(diff['head'])[:12]}: "
                         + (", ".join(parts) or "no change") + ".")
        else:
            lines.append(f"- Diff: {_fmt((diff or {}).get('error'))}.")
        for name in names:
            summary = entry["reviewers"].get(name) or {}
            if summary.get("status") != "ok":
                lines.append(f"- {name}: {summary.get('status')}: {_short(summary.get('error') or '', 300)}")
                continue
            lines.append(f"- {name} ({_fmt(summary.get('seconds'))} s): {_counts(summary['byCategory'])}; "
                         f"obligations {_obligations_cell(summary)}"
                         + (f" (checklist {entry['groundTruthObligations']})" if entry.get("groundTruthObligations") else "")
                         + ".")
            for finding in (findings.get(entry["station"]) or {}).get(name) or []:
                places = ", ".join(f"{e.get('path')}:{e.get('line') if e.get('line') is not None else '-'}"
                                   for e in finding.get("evidence") or [])
                lines.append(f"  - [{finding.get('severity')}] {finding.get('category')} "
                             f"{finding.get('item') or ''}{'/' + finding['rule'] if finding.get('rule') else ''}: "
                             f"{_short(finding.get('detail') or '', 400)} ({places or 'no evidence'})")
        escalations = [f"{name} {block['needed']}/{block['unneeded']}"
                       for name, block in (entry.get("escalations") or {}).items() if block]
        if escalations:
            lines.append(f"- Escalations needed/unneeded: {'; '.join(escalations)}.")
        agreement = entry.get("agreement")
        if agreement:
            shared = ", ".join(f"{s['item'] or '-'} {s['category']}" for s in agreement["shared"]) or "none"
            only = "; ".join(f"{name} only {agreement.get(name + 'Only')}" for name in agreement["reviewers"])
            lines.append(f"- Agreement (same item and category): {agreement['both']} shared ({shared}); {only}.")
        product = entry["productFindings"]
        if product["markitectMcpCalls"] or product["productCommands"] or product["unfinishedMcpCalls"]:
            lines.append(f"- Markitect: {product['markitectMcpCalls']} MCP calls, {product['failedMcpCalls']} failed, "
                         f"{product['unfinishedMcpCalls']} unfinished; {product['productCommands']} failed product "
                         "commands (see product-findings.md).")
        lines.append(f"- Classification: {entry['classification']['class']} ({entry['classification']['reason']}).")
    totals = report["totals"]
    lines += ["", "## Totals", "",
              f"- Public checks {_pair(totals['publicChecks'], False)}; holdouts {_pair(totals['holdouts'], False)}; "
              f"diff {_fmt(totals['diff']['files'])} files (+{_fmt(totals['diff']['added'])}/-{_fmt(totals['diff']['deleted'])}), "
              f"model lines {_fmt(totals['diff']['modelLines'])}; failed Markitect MCP calls {totals['failedMcpCalls']}."]
    for name, block in totals["reviewers"].items():
        lines.append(f"- {name}: {block['wavesReviewed']}/{block['waves']} waves reviewed, {block['findings']} findings "
                     f"({_counts(block['byCategory'])}); obligations {block['obligations']['covered']}/"
                     f"{block['obligations']['total']}.")
        if block.get("loginRefreshed"):
            lines.append(f"- {name} refreshed its login inside the container; log in on the host again before the "
                         "next run.")
    if report.get("errors"):
        lines.append(f"- Assessment errors: {'; '.join(_short(e, 300) for e in report['errors'])}.")
    lines += ["- Diff categories: model = `.markitect/`; tests = test folders and test files; docs = Markdown/text "
              "and `docs/`; code = source files; config/other = the rest. Changed lines approximate review effort.",
              "- Holdouts and reviews judge only what the public requirements and project rules say. One run shows "
              "mechanisms, not a general effect.", ""]
    return "\n".join(lines)


def _judging_code_line(code: Any) -> list[str]:
    if not isinstance(code, dict):  # an assessment from before the judging code was recorded
        return []
    state = {False: "", True: " with local changes (exploratory)"}.get(code.get("dirty"),
                                                                       ", local changes unknown (exploratory)")
    return [f"- Judging code (playground, public checks): commit {str(code.get('commit') or 'n/a')[:12]}{state}."]


def _evaluation_source(evaluation: dict) -> str:
    commit = str(evaluation.get("commit") or "n/a")[:12]
    if evaluation.get("source") == "registered":
        return f"registered Git tree {str(evaluation.get('tree'))[:12]} (commit {commit})"
    if evaluation.get("source") == "working-tree":
        return f"working tree at commit {commit}" + (" with uncommitted changes" if evaluation.get("dirty") else "")
    return f"commit {commit}" + (" with uncommitted changes" if evaluation.get("dirty") else "")


def render_product_findings(report: dict, setup: dict | None, results: Path, analyses: dict) -> str:
    run = report["run"]
    versions = run.get("versions") or {}
    lines = [f"# Product findings: run {run.get('id')}", "",
             f"Case {_fmt(run.get('case'))}, method {_fmt(run.get('method'))}, Markitect "
             f"{_fmt(versions.get('markitectCommit'))} (binary sha256 {_fmt(versions.get('markitectSha256'))}). "
             "Collected from the run's records with the exact error text; the product side judges them. "
             "A failed call is not necessarily a defect (a nonconforming `check` reports findings by "
             "failing).", ""]
    if run.get("method") != "markitect":
        lines += ["This run did not use Markitect; nothing to report.", ""]
        return "\n".join(lines)
    lines += ["## Setup", "", f"Status {_fmt((setup or {}).get('status'))} in {_fmt((setup or {}).get('seconds'))} s"
              + (f"; error: {_short(setup['error'])}" if (setup or {}).get("error") else "") + ".", "",
              "| # | Step | Exit | Seconds |", "|---|---|---|---|"]
    failed_steps = []
    for index, step in enumerate((setup or {}).get("steps") or [], 1):
        lines.append(f"| {index} | {step.get('name')} | {_fmt(step.get('exitCode'))} | {_fmt(step.get('seconds'))} |")
        if step.get("exitCode") not in (0, None) or step.get("timedOut"):
            failed_steps.append(step)
    for step in failed_steps:
        text = (_read_text(results / "setup" / str(step.get("stderr"))).strip()
                or _read_text(results / "setup" / str(step.get("stdout"))).strip())
        lines += ["", f"`{step.get('name')}` exited {_fmt(step.get('exitCode'))}:", "", "```",
                  text[-2000:] or "(no output)", "```"]
    for entry in report["stations"]:
        analysis = analyses.get(entry["station"]) or {}
        lines += ["", f"## {entry['station']}", ""]
        failed = analysis.get("failedMcpCalls") or []
        groups: dict[str, int] = {}
        for call in failed:
            label = f"{call.get('tool')} {call.get('code') or 'error'}"
            groups[label] = groups.get(label, 0) + 1
        lines.append(f"{analysis.get('markitectMcpCalls', 0)} MCP calls to `markitect`, {len(failed)} failed"
                     + (f" ({', '.join(f'{label} x{count}' for label, count in groups.items())})" if groups else "")
                     + ".")
        for call in failed:
            details = (f" Details: `{call['details']}`." if call.get("details") else
                       " The answer carries no validation details." if call.get("code") else "")
            lines.append(f"- `{call.get('tool')}` {call.get('code') or ''}: {call.get('message')} "
                         f"Arguments: `{call.get('arguments')}`.{details}")
        for call in analysis.get("unfinishedMcpCalls") or []:
            lines.append(f"- Did not finish: `{call.get('tool')}` with `{call.get('arguments')}`.")
        for command in analysis.get("productCommands") or []:
            lines.append(f"- Command `{command['command']}` exited {_fmt(command.get('exitCode'))}: {command['output']}")
        for line in analysis.get("stderrProductLines") or []:
            lines.append(f"- Agent stderr: {line}")
        marked = [text for text in analysis.get("errorMessages") or [] if "markitect" in text.lower()]
        for text in marked:
            lines.append(f"- Agent error event: {text}")
    lines.append("")
    return "\n".join(lines)


# --- host side --------------------------------------------------------------------------------

def evaluation_identity() -> dict:
    """The playground source commit, and whether its folder has uncommitted changes."""
    def git(*args: str) -> str | None:
        try:
            done = subprocess.run(["git", "-C", str(ROOT), *args], capture_output=True, text=True, encoding="utf-8",
                                  errors="replace", timeout=60, stdin=subprocess.DEVNULL)
        except (OSError, subprocess.SubprocessError):
            return None
        return done.stdout if done.returncode == 0 else None
    commit = git("rev-parse", "HEAD")
    status = git("status", "--porcelain", "--untracked-files=all", "--", ".")
    return {"commit": commit.strip() if commit else None, "dirty": None if status is None else bool(status.strip())}


def judging_code(case: str) -> dict:
    """The commit of the code that judges a run (playground/, the shared and the case's
    public checks) and whether it has local changes (registration.code_state)."""
    return registration.code_state(ROOT, ("playground", "cases/common/checks", f"cases/{case}/checks"))


def evaluation_source(record: dict, exploratory: bool) -> tuple[dict, dict]:
    """(evaluation, config) an assessment judges with: the Git tree the run was
    pre-registered with and its config.json, or with `exploratory` the working tree.
    Raises AssessError (exit 2) for a run without a registration."""
    root = EVALUATION.parent
    if exploratory:
        state = registration.observe(root)
        dirty = None if state["status"] == "no-checkout" else state["status"] == "dirty"
        return ({"source": "working-tree", "tree": None, "commit": state.get("commit"), "dirty": dirty},
                load_config(EVALUATION / "config.json"))
    pre = record.get("preRegistration")
    if not isinstance(pre, dict):
        raise AssessError("the run records no pre-registration (it started before pre-registration was enforced); "
                          "assess it with --exploratory (compare then refuses the assessment)")
    if pre.get("status") != "registered" or not pre.get("evaluationTree"):
        raise AssessError(f"the run is not pre-registered: when it started, {registration.refusal(pre)}; assess it "
                          "with --exploratory (compare then refuses the assessment)")
    tree = pre["evaluationTree"]
    try:
        if not registration.tree_exists(root, tree):
            raise AssessError(f"the registered evaluation tree {tree} (commit {pre.get('commit')}) is not in this "
                              "checkout; fetch that commit")
        config = check_config(registration.read_json(root, tree, "config.json"),
                              f"config.json in the registered evaluation tree {tree}")
    except registration.RegistrationError as exc:
        raise AssessError(str(exc)) from exc
    return {"source": "registered", "tree": tree, "commit": pre.get("commit"), "dirty": False}, config


def stage_inputs(case: str, target: Path, *, fake_reviewers: bool = False, tree: str | None = None) -> None:
    """Code and evaluation files for the assessment container; never `reference/`,
    `mutants/` or `validate.py`. With `tree` the evaluation files are the committed
    bytes of that registered Git tree, else the working tree's."""
    shutil.copytree(ROOT / "playground", target / "playground", ignore=COPY_IGNORE)
    if fake_reviewers:
        (target / "tests").mkdir(parents=True)
        shutil.copyfile(FAKE_REVIEWER, target / "tests" / FAKE_REVIEWER.name)
    evaluation = target / "evaluation"
    if tree is not None:
        def keep(parts: tuple[str, ...]) -> bool:
            if parts == ("config.json",):
                return True
            patterns = {"common": COPY_PATTERNS, case: CASE_PATTERNS}.get(parts[0])
            return patterns is not None and len(parts) > 1 and not registration.ignored(parts[1:], patterns)

        evaluation.mkdir(parents=True)
        try:
            registration.extract(registration.archive(EVALUATION.parent, tree), evaluation, keep)
        except registration.RegistrationError as exc:
            raise AssessError(str(exc)) from exc
        return
    shutil.copytree(EVALUATION / "common", evaluation / "common", ignore=COPY_IGNORE)
    shutil.copyfile(EVALUATION / "config.json", evaluation / "config.json")
    if (EVALUATION / case).is_dir():
        shutil.copytree(EVALUATION / case, evaluation / case, ignore=CASE_IGNORE)


def wait_bound(stations: int, config: dict, names: list[str]) -> int:
    """Safety net for the host: the worst case of every station's work plus a margin."""
    review = sum(reviewers.timeout_seconds(config["reviewers"][name]) + reviewers.VERSION_TIMEOUT for name in names)
    holdout = float(config.get("holdoutTimeoutSeconds") or HOLDOUT_TIMEOUT)
    per_station = assess.STATION_BOUND_SECONDS + holdout + assess.SWEEP_SECONDS + 3 * GIT_TIMEOUT + review
    return int(max(stations, 1) * per_station + 1800)


def docker_argv(*, name: str, image: str, limits: dict, run_dir: Path, out: Path, codex_auth: Path | None,
                claude_token: Path | None, command: list[str]) -> list[str]:
    from . import host
    argv = ["docker", "run", "--detach", "--name", name, "--label", host.LABEL,
            "--label", "markitect-playground.role=assessment", "--init", *host.SECURITY_OPTS,
            "--cpus", str(limits["cpus"]), "--memory", str(limits["memory"]),
            "--pids-limit", str(limits["pidsLimit"]),
            "--mount", host._mount(run_dir, "/assess/run", readonly=True),
            "--mount", host._mount(out / "inputs", "/assess/in", readonly=True),
            "--mount", host._mount(out, "/assess/out")]
    if codex_auth is not None:
        argv += ["--mount", host._mount(codex_auth, SECRET_CODEX, readonly=True)]
    if claude_token is not None:
        argv += ["--mount", host._mount(claude_token, SECRET_CLAUDE, readonly=True)]
    return argv + ["--workdir", "/assess/in", image, *command]


def inside_command(names: list[str], evaluation: dict, image_id: str | None, *, codex: bool, claude: bool,
                   fake_reviewers: bool = False, exploratory: bool = False) -> list[str]:
    command = ["python3", "-B", "-m", "playground", "assess", "--inside", "--run", "/assess/run",
               "--evaluation", "/assess/in/evaluation", "--out", "/assess/out",
               "--reviewers", ",".join(names) or "none"]
    if fake_reviewers:
        command.append("--fake-reviewers")
    if exploratory:
        command.append("--exploratory")
    if codex:
        command += ["--codex-auth", SECRET_CODEX]
    if claude:
        command += ["--claude-token", SECRET_CLAUDE]
    if evaluation.get("source"):
        command += ["--evaluation-source", evaluation["source"]]
    if evaluation.get("tree"):
        command += ["--evaluation-tree", evaluation["tree"]]
    if evaluation.get("commit"):
        command += ["--evaluation-commit", evaluation["commit"]]
    if evaluation.get("dirty") is not None:
        command += ["--evaluation-dirty", "yes" if evaluation["dirty"] else "no"]
    code = evaluation.get("code") or {}
    if code.get("commit"):
        command += ["--code-commit", code["commit"]]
    if code.get("dirty") is not None:
        command += ["--code-dirty", "yes" if code["dirty"] else "no"]
    if image_id:
        command += ["--image-id", image_id]
    return command


def host_assess(args: argparse.Namespace) -> int:
    run_dir = Path(args.run).resolve()
    names = parse_reviewers(args.reviewers)
    fake = bool(args.fake_reviewers and names)
    if fake and (args.codex_auth or args.claude_token):
        raise AssessError("--fake-reviewers uses throwaway credentials; do not pass --codex-auth or --claude-token")
    codex_auth = claude_token = None
    if fake:
        pass  # throwaway files are made right before the container starts
    elif "codex" in names:  # existence checks only; the files are never opened on the host
        codex_auth = Path(args.codex_auth or Path.home() / ".codex" / "auth.json")
        if not codex_auth.is_file():
            raise AssessError(f"Codex auth file not found: {codex_auth}", outcome.ENVIRONMENT)
        codex_auth = codex_auth.resolve()
    if "claude" in names and not fake:
        claude_token = Path(args.claude_token or DEFAULT_CLAUDE_TOKEN)
        if not claude_token.is_file():
            raise AssessError(f"Claude token file not found: {claude_token} (create it once with `claude setup-token`)",
                              outcome.ENVIRONMENT)
        claude_token = claude_token.resolve()
    exit_code, _result = assess_container(run_dir, names=names, codex_auth=codex_auth, claude_token=claude_token,
                                          fake=fake, image=args.image, force=args.force, keep=args.keep_container,
                                          exploratory=args.exploratory)
    print(f"assessment: {run_dir / 'assessment' / 'report.md'}")
    return exit_code


def assess_container(run_dir: Path, *, names: list[str], codex_auth: Path | None = None,
                     claude_token: Path | None = None, fake: bool = False, image: str | None = None,
                     force: bool = False, keep: bool = False,
                     before_remove: Callable[[str], None] | None = None,
                     exploratory: bool = False) -> tuple[int, dict]:
    """Assess one run folder in its own container and return (exit code, host record);
    the code (outcome.assess) is also `exitCode` in the assessment's host.json.

    The evaluation files are the run's registered Git tree (evaluation_source); a run
    without a registration, or a reviewer whose model is also an arm's model, is refused
    with AssessError (exit 2) before Docker, unless `exploratory` (or, for a model clash
    only, `fake`): then the clash is a warning and the report records it.

    `codex_auth` and `claude_token` are login files mounted read-only (never opened here);
    with `fake` the reviewers get throwaway credentials instead. `before_remove(container)`
    runs after the container has stopped and before it is removed: the Codex reviewers'
    working copy of the login is then at REVIEWER_LOGIN inside it.
    """
    from . import host
    run_dir = Path(run_dir).resolve()
    record = _read_json(run_dir / "host.json")
    manifest = (record or {}).get("manifest")
    if not isinstance(manifest, dict) or not (run_dir / "results").is_dir():
        raise AssessError(f"not a run folder (needs host.json with a manifest and results/): {run_dir}")
    evaluation, config = evaluation_source(record, exploratory)
    missing = [name for name in names if name not in config["reviewers"]]
    if missing:
        raise AssessError(f"evaluation/config.json has no reviewer {', '.join(missing)}")
    exploratory = exploratory or (record.get("rules") or {}).get("exploratory") is True
    code = judging_code(manifest["case"])
    evaluation["code"] = {"commit": code["commit"], "dirty": code["dirty"]}
    if code["dirty"] is not False:  # judged by code that is not committed: not the registered judgment
        exploratory = True
        state = (f"has local changes ({', '.join(code['changed'][:5])})" if code["dirty"]
                 else f"has no known commit ({code.get('error')})")
        print(f"warning: the judging code {state}; the assessment is exploratory and compare refuses it",
              file=sys.stderr)
    run_report = _read_json(run_dir / "results" / "report.json") or {}
    clashes = registration.clashes(config, names, registration.arm_models(manifest, run_report.get("roles")))
    if clashes and not (exploratory or fake):
        raise AssessError(f"{registration.describe(clashes)}; reviewer models must differ from the arms' models: "
                          "select other reviewers with --reviewers, or pass --exploratory (compare then refuses "
                          "the assessment)")
    if clashes:
        print(f"warning: {registration.describe(clashes)}; compare refuses this assessment", file=sys.stderr)
    out = run_dir / "assessment"
    if out.exists() or out.is_symlink():
        if not force:
            raise AssessError(f"{out} already exists; pass --force to replace it")
        shutil.rmtree(out)
    name = f"mpg-assess-{manifest['id']}"
    image = image or (record.get("image") or {}).get("id") or (record.get("image") or {}).get("tag")
    if not image:
        raise AssessError("host.json names no image; pass --image")
    docker_version = host._capture(["docker", "version", "--format", "{{.Server.Version}}"])
    if host._container_state(name) is not None:
        raise AssessError(f"a container named {name} already exists; remove it (docker rm -f {name})",
                          outcome.ENVIRONMENT)
    try:
        image_id = host._capture(["docker", "image", "inspect", "--format", "{{.Id}}", image])
    except host.HostError as exc:
        raise AssessError(f"image {image} is not available ({exc}); rebuild it or pass --image",
                          outcome.ENVIRONMENT) from exc
    stations = len(discover_stations(run_dir / "results"))
    timeout = wait_bound(stations, config, names)
    out.mkdir()
    try:
        stage_inputs(manifest["case"], out / "inputs", fake_reviewers=fake, tree=evaluation["tree"])
    except (OSError, AssessError):
        shutil.rmtree(out, ignore_errors=True)
        raise
    throwaway = Path(tempfile.mkdtemp(prefix="mpg-fake-credentials-")) if fake else None
    if throwaway is not None:
        codex_auth = claude_token = None
        if "codex" in names:
            codex_auth = throwaway / "auth.json"
            codex_auth.write_text('{"fake": "login"}\n', encoding="utf-8")
        if "claude" in names:
            claude_token = throwaway / "claude-token"
            claude_token.write_text(f"fake-reviewer-token-{secrets.token_hex(16)}\n", encoding="utf-8")
    command = inside_command(names, evaluation, image_id, codex=codex_auth is not None,
                             claude=claude_token is not None, fake_reviewers=fake, exploratory=exploratory)
    argv = docker_argv(name=name, image=image_id, limits=manifest.get("container") or {}, run_dir=run_dir, out=out,
                       codex_auth=codex_auth, claude_token=claude_token, command=command)
    result: dict[str, Any] = {"run": str(run_dir), "status": "start-failed", "container": name,
                              "image": {"ref": image, "id": image_id}, "dockerVersion": docker_version,
                              "reviewers": names, "fakeReviewers": fake, "evaluation": evaluation,
                              "rules": {"exploratory": exploratory, "reviewerClashes": clashes},
                              "judgingCode": code,
                              "timeoutSeconds": timeout, "startedAt": _utc(), "endedAt": None,
                              "containerExitCode": None, "oomKilled": None, "exitCode": None, "dockerRun": argv,
                              "error": None}
    try:
        host._capture(argv)
        code, error = host.wait_container(name, timeout)
        if error is None:
            result["status"], result["containerExitCode"] = "completed", code
        else:
            result["status"], result["error"] = "wait-failed", error
    except subprocess.TimeoutExpired:
        result["status"] = "host-timeout"
    except KeyboardInterrupt:
        result["status"] = "host-interrupted"
    except host.HostError as exc:
        result["error"] = str(exc)
        print(f"error: {exc}", file=sys.stderr)
    except Exception as exc:  # a bug of ours: recorded with the code __main__ exits with, then raised
        result["status"], result["error"] = "harness-error", f"{type(exc).__name__}: {exc}"
        raise
    finally:
        try:
            host._finish_container(name, result, out, keep, before_remove)
        except KeyboardInterrupt:  # e.g. during the login copy-out; the container is gone
            result["status"] = "host-interrupted"
        if throwaway is not None:
            shutil.rmtree(throwaway, ignore_errors=True)
        result["endedAt"] = _utc()
        result["exitCode"] = exit_code = outcome.assess(result["status"], result["containerExitCode"],
                                                        oom_killed=result["oomKilled"])
        _write_json(out / "host.json", result)  # first, so an interrupted hand-back keeps the record
        result["handBack"] = host.hand_back(name, image_id, out)
        _write_json(out / "host.json", result)
        if result["handBack"].startswith(("failed", "skipped")):
            print(f"warning: {out} stays owned by root ({result['handBack']})", file=sys.stderr)
    return exit_code, result


# --- CLI ---------------------------------------------------------------------------------------

def fake_executables(source: Path, names: list[str]) -> dict:
    """The fake reviewer as a copy the unprivileged user can run (/assess stays root-only)."""
    folder = Path(tempfile.mkdtemp(prefix="mpg-fake-reviewer-"))
    os.chmod(folder, 0o755)
    target = folder / source.name
    shutil.copyfile(source, target)
    os.chmod(target, 0o644)
    return {name: [sys.executable, str(target)] for name in names}


def inside(args: argparse.Namespace) -> int:
    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    if codex_agent.container_mode() and CONTAINER_ROOT.is_dir():
        os.chmod(CONTAINER_ROOT, 0o700)  # inputs, results and secrets stay root-only
    try:
        names = parse_reviewers(args.reviewers)
        executables = None
        if args.fake_reviewers:
            executables = fake_executables(Path(args.evaluation).parent / "tests" / FAKE_REVIEWER.name, names)
        assess_run(Path(args.run), Path(args.evaluation), out, reviewer_names=names,
                   codex_auth=Path(args.codex_auth) if args.codex_auth else None,
                   claude_token_file=Path(args.claude_token) if args.claude_token else None,
                   executables=executables,
                   evaluation={"source": args.evaluation_source, "tree": args.evaluation_tree,
                               "commit": args.evaluation_commit,
                               "dirty": None if args.evaluation_dirty is None else args.evaluation_dirty == "yes",
                               "code": {"commit": args.code_commit,
                                        "dirty": None if args.code_dirty is None else args.code_dirty == "yes"}},
                   image=args.image_id, exploratory=args.exploratory)
    except Exception:
        (out / "assess-error.txt").write_text(traceback.format_exc(), encoding="utf-8")
        print(traceback.format_exc(), file=sys.stderr)
        return 2
    print(f"assessment written to {out}")
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="python -m playground assess", description=__doc__,
                                     epilog=outcome.help_text(),
                                     formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--run", required=True, help="run folder (with host.json and results/)")
    parser.add_argument("--codex-auth", help="Codex login file (default ~/.codex/auth.json)")
    parser.add_argument("--claude-token", help="file with a Claude OAuth token from `claude setup-token` "
                                               "(default ~/.markitect-playground/claude-token)")
    parser.add_argument("--reviewers", help="codex,claude (default), one of them, or none")
    parser.add_argument("--image", help="image to assess in (default: the run's image)")
    parser.add_argument("--force", action="store_true", help="replace an existing assessment/ folder")
    parser.add_argument("--keep-container", action="store_true")
    parser.add_argument("--fake-reviewers", action="store_true",
                        help="smoke test: tests/fake_reviewer.py stands in for both CLIs (no model call)")
    parser.add_argument("--exploratory", action="store_true",
                        help="judge with the working tree's evaluation files, also for a run without a "
                             "pre-registration or with a reviewer model clash; compare refuses the assessment")
    parser.add_argument("--inside", action="store_true", help=argparse.SUPPRESS)
    parser.add_argument("--evaluation", help=argparse.SUPPRESS)
    parser.add_argument("--out", help=argparse.SUPPRESS)
    parser.add_argument("--evaluation-source", choices=("registered", "working-tree"), help=argparse.SUPPRESS)
    parser.add_argument("--evaluation-tree", help=argparse.SUPPRESS)
    parser.add_argument("--evaluation-commit", help=argparse.SUPPRESS)
    parser.add_argument("--evaluation-dirty", choices=("yes", "no"), help=argparse.SUPPRESS)
    parser.add_argument("--code-commit", help=argparse.SUPPRESS)
    parser.add_argument("--code-dirty", choices=("yes", "no"), help=argparse.SUPPRESS)
    parser.add_argument("--image-id", help=argparse.SUPPRESS)
    args = parser.parse_args(sys.argv[1:] if argv is None else argv)
    if args.inside:
        if not (args.evaluation and args.out):
            parser.error("--inside needs --evaluation and --out")
        return inside(args)
    from . import host  # host side only; the container never needs it
    try:
        return host_assess(args)
    except (AssessError, host.HostError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return exc.code
