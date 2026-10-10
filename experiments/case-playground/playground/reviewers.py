"""Independent per-wave consistency reviews by a Codex and a Claude reviewer.

Runs inside the assessment container as root; every reviewer process runs as the
unprivileged agent user (as the current user in test mode) through
`codex_agent.run_as_agent`, in a fresh home that holds only its own credential:
`auth.json` in a fresh CODEX_HOME for Codex, the OAuth token in the child's
environment (never in argv, files or logs) and a fresh CLAUDE_CONFIG_DIR for Claude.

Both reviewers get the identical prompt: the fixed `evaluation/common/reviewer-prompt.md`
followed by the wave's inputs (released item text, project rules, ground-truth slice
when present, the agent's final message, the wave diff). Their working directory is
the bundle folder, never the repository, so no project instruction file (AGENTS.md,
CLAUDE.md) is loaded as instructions; the read-only snapshot copy is read by path.

A reviewer never ends the assessment: a crash, timeout or invalid answer is recorded
with its raw output and status `error` or `invalid`.
"""
from __future__ import annotations

import json
import os
import re
import shutil
import sys
import tempfile
import traceback
from pathlib import Path
from typing import Any

from . import codex_agent
from .lifecycle import read_text as _read_text

PROVIDERS = ("codex", "claude")
CATEGORIES = ("missed_obligation", "unnecessary_change", "rule_violation", "contradiction", "regression",
              "false_claim", "escalation_needed", "escalation_unneeded")
SEVERITIES = ("high", "medium", "low")
READ_ONLY_TOOLS = ("Read", "Grep", "Glob")
DEFAULT_TIMEOUT = 2700
VERSION_TIMEOUT = 120
OUTPUT_LIMIT = 4 << 20  # bytes of a reviewer answer that are read
# One argv element is limited to 128 KiB on Linux; the prompt travels as one argument
# (run_as_agent gives the child no stdin). Longer inputs are cut, with a pointer to
# the full file in the bundle.
DEFAULT_PROMPT_MAX_BYTES = 100_000
CLAUDE_ENV = {"DISABLE_AUTOUPDATER": "1", "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1"}
REVIEWER_HOMES = Path("/var/lib/mpg-reviewers")  # container only; see _fresh_home


# --- schema validation (the subset reviewer-schema.json uses) --------------------------

_TYPES = {"object": dict, "array": list, "string": str, "boolean": bool, "null": type(None)}


def _is_type(value: Any, name: str) -> bool:
    if name == "integer":
        return isinstance(value, int) and not isinstance(value, bool)
    if name == "number":
        return isinstance(value, (int, float)) and not isinstance(value, bool)
    return isinstance(value, _TYPES.get(name, ()))


def validate(value: Any, schema: dict, where: str = "$") -> list[str]:
    """Errors of `value` against a JSON schema using type, enum, properties, required,
    additionalProperties, items and minimum; an empty list means valid."""
    errors: list[str] = []
    types = schema.get("type")
    if types is not None:
        names = [types] if isinstance(types, str) else list(types)
        if not any(_is_type(value, name) for name in names):
            return [f"{where}: expected {' or '.join(names)}, got {type(value).__name__}"]
    if "enum" in schema and value not in schema["enum"]:
        errors.append(f"{where}: {value!r} is not one of {', '.join(map(str, schema['enum']))}")
    if isinstance(value, dict):
        properties = schema.get("properties") or {}
        for key in schema.get("required") or []:
            if key not in value:
                errors.append(f"{where}: missing {key}")
        if schema.get("additionalProperties") is False:
            errors += [f"{where}: unexpected {key}" for key in value if key not in properties]
        for key, sub in properties.items():
            if key in value:
                errors += validate(value[key], sub, f"{where}.{key}")
    if isinstance(value, list) and isinstance(schema.get("items"), dict):
        for index, item in enumerate(value):
            errors += validate(item, schema["items"], f"{where}[{index}]")
    if "minimum" in schema and _is_type(value, "number") and value < schema["minimum"]:
        errors.append(f"{where}: {value} is below {schema['minimum']}")
    return errors


def parse_json_text(text: str) -> Any:
    """The JSON value in a reviewer's answer; tolerates a Markdown code fence."""
    stripped = text.strip()
    fenced = re.fullmatch(r"```(?:json)?\s*(.*?)\s*```", stripped, re.S)
    if fenced:
        stripped = fenced.group(1)
    try:
        return json.loads(stripped)
    except ValueError:
        start, end = stripped.find("{"), stripped.rfind("}")
        if 0 <= start < end:
            return json.loads(stripped[start:end + 1])
        raise


# --- reviewer input ---------------------------------------------------------------------

_ITEM_START = r"^\s*(?:[-*+]\s+|#{1,6}\s+|\d+[.)]\s+)?[*_`]*"


def extract_items(backlog: str, ids: list[str], all_ids: list[str]) -> dict[str, str | None]:
    """The public text of each item: its line (bullet or heading) up to the next item or
    heading. None when an item cannot be found."""
    lines = backlog.splitlines()
    known = sorted(set(all_ids) | set(ids), key=len, reverse=True)
    any_item = re.compile(_ITEM_START + "(?:" + "|".join(map(re.escape, known)) + r")\b") if known else None
    found: dict[str, str | None] = {}
    for item in ids:
        own = re.compile(_ITEM_START + re.escape(item) + r"\b")
        start = next((index for index, line in enumerate(lines) if own.match(line)), None)
        if start is None:
            found[item] = None
            continue
        block = [lines[start]]
        for line in lines[start + 1:]:
            if line.lstrip().startswith("#") or (any_item is not None and any_item.match(line)):
                break
            block.append(line)
        found[item] = "\n".join(block).strip()
    return found


def wave_ground_truth(ground_truth: dict | None, station: int) -> dict | None:
    """The ground-truth wave of `station` plus the rule list, or None."""
    if not isinstance(ground_truth, dict):
        return None
    for wave in ground_truth.get("waves") or []:
        if isinstance(wave, dict) and str(wave.get("station", "")).upper().lstrip("S") == str(station):
            return {"case": ground_truth.get("case"), "rules": ground_truth.get("rules") or [], "wave": wave}
    return None


def obligation_count(wave_truth: dict | None) -> int | None:
    if wave_truth is None:
        return None
    return sum(len(item.get("obligations") or []) for item in wave_truth["wave"].get("items") or []
               if isinstance(item, dict))


def _cut(text: str, limit: int, pointer: str) -> str:
    data = text.encode("utf-8")
    if len(data) <= limit:
        return text
    kept = data[:max(limit, 0)].decode("utf-8", errors="ignore")
    return kept + f"\n\n[... cut here: {len(data) - len(kept.encode('utf-8')):,} more bytes; full text in {pointer}]\n"


def compose_prompt(template: str, wave: dict, max_bytes: int = DEFAULT_PROMPT_MAX_BYTES) -> str:
    """The template followed by this wave's inputs. `wave` holds station, items,
    earlierItems, itemTexts, rules ({name: text}), groundTruth, lastMessage, diff,
    backlog, repo and bundle (paths). Long parts are cut to keep the prompt under
    `max_bytes`, diff first, with a pointer to the full file in the bundle."""
    bundle = wave["bundle"]
    items = ", ".join(wave["items"]) or "(none)"
    head = [template.rstrip(), "", "# Inputs for this wave", "",
            f"- Station: S{wave['station']}",
            f"- Released items: {items}",
            f"- Released in earlier waves: {', '.join(wave.get('earlierItems') or []) or '(none)'}",
            f"- Repository snapshot (merged `main` after this wave, read-only): {wave['repo']}",
            f"- Full inputs (diff, item texts, rules, ground truth): {bundle}", "",
            "## Released backlog items", ""]
    for item in wave["items"]:
        text = (wave.get("itemTexts") or {}).get(item)
        head.append(text if text else f"- {item}: (text not found; see the full backlog below)")
    head.append("")
    head += ["## Project rules", ""]
    for name, text in (wave.get("rules") or {}).items():
        head += [f"### {name}", "", text.strip(), ""]
    head += ["## Ground truth for this wave", ""]
    if wave.get("groundTruth") is not None:
        head += ["Use these obligations, areas, rule expectations and must-not-change statements as "
                 "the checklist.", "", "```json", json.dumps(wave["groundTruth"], ensure_ascii=False, indent=2),
                 "```", ""]
    else:
        head += ["No ground truth exists for this case. Derive the obligations from the released items "
                 "and the project rules only.", ""]
    fixed = "\n".join(head)
    sections = [
        ("## Agent's final message for this wave", wave.get("lastMessage") or "(no final message recorded)",
         f"{bundle}/last-message.txt", "text"),
        ("## Wave diff (previous `main` to this `main`)", wave.get("diff") or "(no change)",
         f"{bundle}/wave.diff", "diff"),
        ("## Full public backlog (items of later waves are not released yet)", wave.get("backlog") or "",
         f"{bundle}/backlog.md", "markdown"),
    ]
    budget = max_bytes - len(fixed.encode("utf-8")) - 400
    shares = {"diff": 0.6, "text": 0.15, "markdown": 0.25}
    parts = [fixed]
    for title, text, pointer, kind in sections:
        limit = max(int(budget * shares[kind]), 0)
        parts += ["", title, "", f"```{kind}", _cut(text, limit, pointer).rstrip(), "```"]
    return "\n".join(parts) + "\n"


def write_bundle(bundle: Path, wave: dict, prompt: str, schema: dict) -> None:
    """The full inputs next to the prompt, for the reviewers and for the record."""
    bundle.mkdir(parents=True, exist_ok=True)
    files = {
        "prompt.md": prompt,
        "wave.diff": wave.get("diff") or "",
        "last-message.txt": wave.get("lastMessage") or "",
        "backlog.md": wave.get("backlog") or "",
        "items.md": "\n\n".join(text or f"- {item}: (not found)" for item, text in
                                (wave.get("itemTexts") or {}).items()) + "\n",
        "rules.md": "\n\n".join(f"# {name}\n\n{text.strip()}" for name, text in (wave.get("rules") or {}).items()) + "\n",
        "reviewer-schema.json": json.dumps(schema, indent=2) + "\n",
    }
    if wave.get("groundTruth") is not None:
        files["ground-truth.json"] = json.dumps(wave["groundTruth"], ensure_ascii=False, indent=2) + "\n"
    for name, text in files.items():
        (bundle / name).write_text(text, encoding="utf-8", newline="\n")


# --- commands ---------------------------------------------------------------------------

def codex_command(cfg: dict, *, schema_path: Path, output_path: Path, prompt: str,
                  executable: list[str] | None = None) -> list[str]:
    argv = [*(executable or ["codex"]), "exec", "--json", "--sandbox", "read-only", "--skip-git-repo-check",
            "-m", cfg["model"]]
    if cfg.get("effort"):
        argv += ["-c", f"model_reasoning_effort={json.dumps(cfg['effort'])}"]
    return argv + ["--output-schema", str(schema_path), "-o", str(output_path), "--", prompt]


def claude_command(cfg: dict, *, schema: dict, repo_dir: Path, prompt: str,
                   executable: list[str] | None = None) -> list[str]:
    """`--bare` would ignore OAuth tokens, so settings are switched off with an empty
    `--setting-sources` (no user, project or local settings) and a fresh config dir."""
    tools = ",".join(READ_ONLY_TOOLS)
    argv = [*(executable or ["claude"]), "-p", "--output-format", "json",
            "--json-schema", json.dumps(schema, separators=(",", ":")), "--model", cfg["model"]]
    if cfg.get("effort"):
        argv += ["--effort", cfg["effort"]]
    return argv + ["--tools", tools, "--allowedTools", tools, "--permission-mode", "dontAsk",
                   "--setting-sources", "", "--strict-mcp-config", "--disable-slash-commands",
                   "--no-session-persistence", "--add-dir", str(repo_dir), "--", prompt]


def cli_version(provider: str, executable: list[str] | None = None) -> str | None:
    """`<cli> --version` as the unprivileged user in a throwaway home."""
    home = _fresh_home()
    try:
        env = {"HOME": str(home), "CODEX_HOME": str(home / ".codex"), "CLAUDE_CONFIG_DIR": str(home / ".claude"),
               **(CLAUDE_ENV if provider == "claude" else {})}
        run = codex_agent.run_as_agent([*(executable or [provider]), "--version"], home, VERSION_TIMEOUT,
                                       home / "version.out", home / "version.err", extra_env=env)
        text = _read_text(home / "version.out").strip()
        return text.splitlines()[-1].strip() if run.get("exitCode") == 0 and text else None
    except Exception:
        return None
    finally:
        codex_agent.kill_all_agent_processes()
        remove_tree(home)


# --- running one review -----------------------------------------------------------------

def _fresh_home() -> Path:
    """An empty home for one reviewer process, owned by the unprivileged user. In the
    container it lies outside /tmp: Codex refuses to set up its sandbox helper under the
    temp dir, and every read-only shell command would then fail."""
    base = None
    if codex_agent.container_mode():
        base = REVIEWER_HOMES
        base.mkdir(mode=0o755, parents=True, exist_ok=True)
    home = Path(tempfile.mkdtemp(prefix="mpg-reviewer-home-", dir=base))
    codex_agent.give_to_agent(home)
    return home


def _file_key(path: Path) -> tuple[int, int] | None:
    try:
        info = os.stat(path, follow_symlinks=False)
    except OSError:
        return None
    return info.st_mtime_ns, info.st_size


def _read_capped(path: Path) -> str:
    try:
        with path.open("rb") as handle:
            return handle.read(OUTPUT_LIMIT).decode("utf-8", errors="replace")
    except OSError:
        return ""


def run_reviewer(provider: str, cfg: dict, *, prompt: str, schema: dict, schema_path: Path, repo_dir: Path,
                 bundle_dir: Path, out_dir: Path, codex_auth: Path | None = None,
                 claude_token: str | None = None, executable: list[str] | None = None) -> dict:
    """Run one reviewer on one wave and return its record; raw output stays in `out_dir`."""
    out_dir.mkdir(parents=True, exist_ok=True)
    record: dict[str, Any] = {
        "provider": provider, "model": cfg.get("model"), "effort": cfg.get("effort"), "status": "error",
        "exitCode": None, "timedOut": False, "seconds": None, "usage": None, "resolvedModels": None,
        "loginRefreshed": None, "findings": [], "obligations": None, "notes": None, "error": None,
        "validationErrors": [], "stdout": None, "stderr": None, "raw": None,
    }
    home = None
    try:
        home = _fresh_home()
        env = {"HOME": str(home)}
        login = None
        if provider == "codex":
            codex_home = home / ".codex"
            codex_home.mkdir()
            codex_agent.give_to_agent(codex_home)
            if codex_auth is None or not codex_agent.install_auth(Path(codex_auth), codex_home):
                record["error"] = "Codex login file missing"
                return record
            login = _file_key(codex_home / "auth.json")
            output = home / "review.json"
            env["CODEX_HOME"] = str(codex_home)
            argv = codex_command(cfg, schema_path=schema_path, output_path=output, prompt=prompt,
                                 executable=executable)
            stdout, stderr = out_dir / "events.jsonl", out_dir / "stderr.txt"
        elif provider == "claude":
            if not claude_token:
                record["error"] = "Claude token missing"
                return record
            config_dir = home / ".claude"
            config_dir.mkdir()
            codex_agent.give_to_agent(config_dir)
            env.update(CLAUDE_ENV, CLAUDE_CONFIG_DIR=str(config_dir), CLAUDE_CODE_OAUTH_TOKEN=claude_token)
            argv = claude_command(cfg, schema=schema, repo_dir=repo_dir, prompt=prompt, executable=executable)
            output, stdout, stderr = None, out_dir / "stdout.json", out_dir / "stderr.txt"
        else:
            record["error"] = f"unknown reviewer {provider!r}"
            return record
        # The prompt is long and already saved in the bundle; record the argv without it.
        (out_dir / "argv.json").write_text(json.dumps(argv[:-1] + ["<prompt.md>"], indent=2) + "\n",
                                           encoding="utf-8", newline="\n")
        run = codex_agent.run_as_agent(argv, bundle_dir, timeout_seconds(cfg), stdout, stderr, extra_env=env)
        codex_agent.kill_all_agent_processes()
        record.update(exitCode=run.get("exitCode"), timedOut=bool(run.get("timedOut")), seconds=run.get("seconds"),
                      stdout=stdout.name, stderr=stderr.name)
        if run.get("error"):  # e.g. the CLI is not installed in this image
            record["error"] = f"cannot start the {provider} reviewer: {run['error']}"
        if provider == "codex":
            record["loginRefreshed"] = login is not None and _file_key(home / ".codex" / "auth.json") != login
            raw = _read_capped(output) if output is not None else ""
            _codex_details(record, stdout)
        else:
            raw = _claude_details(record, _read_capped(stdout))
        (out_dir / "output.txt").write_text(raw, encoding="utf-8", newline="\n")
        record["raw"] = "output.txt"
        if record.get("error"):  # the CLI itself reported a failed session
            record["status"] = "error"
        else:
            _judge(record, raw, schema, run)
    except Exception:
        record.update(status="error", error=traceback.format_exc()[-4000:])
    finally:
        try:
            codex_agent.kill_all_agent_processes()
        except Exception:
            pass
        if home is not None:
            remove_tree(home)  # removes the copied Codex login with it
    return record


def _codex_details(record: dict, events_path: Path) -> None:
    events = codex_agent.parse_events(events_path)
    record["usage"] = events.get("tokens")
    record["eventErrors"] = events.get("errors")
    last = None
    for line in _read_text(events_path).splitlines():
        try:
            event = json.loads(line)
        except ValueError:
            continue
        if isinstance(event, dict) and event.get("type") in ("error", "turn.failed"):
            error = event.get("error")
            last = event.get("message") or (error.get("message") if isinstance(error, dict) else None) or last
    record["lastEventError"] = str(last)[:1000] if last else None


def _claude_details(record: dict, stdout: str) -> str:
    """Usage and the structured answer from `claude -p --output-format json`."""
    result = None
    for candidate in (stdout, *reversed(stdout.strip().splitlines()[-1:])):
        try:
            value = json.loads(candidate)
        except ValueError:
            continue
        if isinstance(value, dict):
            result = value
            break
    if result is None:
        return stdout
    usage = result.get("usage") if isinstance(result.get("usage"), dict) else {}
    record["usage"] = {"input": usage.get("input_tokens"), "cachedInput": usage.get("cache_read_input_tokens"),
                       "cacheCreation": usage.get("cache_creation_input_tokens"),
                       "output": usage.get("output_tokens"), "costUsd": result.get("total_cost_usd"),
                       "turns": result.get("num_turns")}
    models = result.get("modelUsage")
    record["resolvedModels"] = sorted(models) if isinstance(models, dict) else None
    if result.get("is_error"):
        record["error"] = str(result.get("result") or result.get("subtype") or "reviewer reported an error")[:4000]
    structured = result.get("structured_output")
    if isinstance(structured, dict):
        return json.dumps(structured, ensure_ascii=False, indent=2)
    return result.get("result") if isinstance(result.get("result"), str) else stdout


def _judge(record: dict, raw: str, schema: dict, run: dict) -> None:
    if not raw.strip():
        reason = ("timed out" if run.get("timedOut") else
                  f"exit {run.get('exitCode')}" if run.get("exitCode") != 0 else "empty answer")
        detail = f": {record['lastEventError']}" if record.get("lastEventError") else ""
        record.update(status="error", error=record.get("error") or f"no answer ({reason}){detail}")
        return
    try:
        value = parse_json_text(raw)
    except ValueError as exc:
        record.update(status="invalid", validationErrors=[f"not JSON: {exc}"])
        return
    errors = validate(value, schema)
    if errors:
        record.update(status="invalid", validationErrors=errors[:50])
        return
    record.update(status="ok", findings=value["findings"], obligations=value["obligations"], notes=value["notes"])
    if value["obligations"]["covered"] > value["obligations"]["total"]:
        record["warnings"] = ["obligations.covered exceeds obligations.total"]


# --- summaries --------------------------------------------------------------------------

def summarize(record: dict) -> dict:
    """Counts of one reviewer record for the report."""
    ok = record.get("status") == "ok"
    findings = record.get("findings") or []
    return {
        "status": record.get("status"), "model": record.get("model"), "effort": record.get("effort"),
        "findings": len(findings) if ok else None,
        "byCategory": {name: sum(1 for f in findings if f.get("category") == name) for name in CATEGORIES}
        if ok else None,
        "bySeverity": {name: sum(1 for f in findings if f.get("severity") == name) for name in SEVERITIES}
        if ok else None,
        "obligations": record.get("obligations") if ok else None,
        "seconds": record.get("seconds"), "usage": record.get("usage"),
        "resolvedModels": record.get("resolvedModels"), "loginRefreshed": record.get("loginRefreshed"),
        "error": record.get("error") or ("; ".join(record.get("validationErrors") or []) or None),
    }


def _keys(findings: list[dict]) -> set[tuple[str, str]]:
    return {(str(f.get("item") or "").strip().upper(), str(f.get("category"))) for f in findings}


def agreement(records: dict[str, dict]) -> dict | None:
    """Agreement of two successful reviewers: findings with the same item and category."""
    ok = [name for name in PROVIDERS if (records.get(name) or {}).get("status") == "ok"]
    if len(ok) != 2:
        return None
    first, second = (_keys(records[name]["findings"]) for name in ok)
    both = first & second
    union = first | second
    return {"reviewers": ok, "both": len(both), f"{ok[0]}Only": len(first - second),
            f"{ok[1]}Only": len(second - first),
            "jaccard": round(len(both) / len(union), 3) if union else None,
            "shared": [{"item": item or None, "category": category} for item, category in sorted(both)]}


def remove_tree(path: Path) -> None:
    """Remove a temporary tree, including read-only entries."""
    def retry(func, target, _error):
        os.chmod(target, 0o700)
        func(target)
    try:
        if sys.version_info >= (3, 12):
            shutil.rmtree(path, onexc=retry)
        else:
            shutil.rmtree(path, onerror=retry)
    except OSError:
        pass


def timeout_seconds(cfg: dict) -> float:
    return float(cfg.get("timeoutSeconds") or DEFAULT_TIMEOUT)
