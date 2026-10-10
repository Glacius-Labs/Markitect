"""Build report.json and report.md from the runner's artifacts in one output directory.

Reads `runner.json`, `setup/setup.json`, `stations/S<n>/{agent,checks}.json` and
`final/final.json`.
Unknown values stay null (JSON) / "n/a" (Markdown); they are never turned into 0.

The run outcome is classified (`classification.class`) as `harness` (our code),
`environment` (Docker, logins, provider), `product` (Markitect's setup or commands) or
`none`, with the reason. Only
causes that ended the run or left its final assessment incomplete decide the category;
everything else found on the way is listed under `signals` (for example MCP calls the
product could not run).
"""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

TOKEN_KINDS = ("input", "cachedInput", "output")
CATEGORIES = ("harness", "environment", "product")  # most to least fundamental
TOKEN_NOTE = ("Tokens per wave from Codex's session records (in / cached / out). 'Outer' is the session "
              "the harness started; 'all' adds every other recorded session (subagents, Markitect's "
              "inner roles). Ephemeral sessions keep no record, so 'all' is a lower bound.")
CLAUDE_TOKEN_NOTE = ("Tokens per wave from Claude Code's session transcripts (in incl. cache writes and reads / "
                     "cache reads / out; without a transcript from the wave's stream-json). 'Outer' is the "
                     "session the harness started; 'all' adds subagent transcripts and Codex session records "
                     "(Markitect's inner roles). Subagent transcripts can hold partial output counts, so 'all' "
                     "is a lower bound.")


def _read(path: Path) -> dict[str, Any] | None:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, ValueError):
        return None
    return value if isinstance(value, dict) else None


def _get(data: Any, *keys: str) -> Any:
    for key in keys:
        if not isinstance(data, dict):
            return None
        data = data.get(key)
    return data


def sum_known(values: list[Any]) -> int | float | None:
    """Sum of known numbers; null when nothing is known or any part is unknown."""
    if not values or any(not isinstance(v, (int, float)) or isinstance(v, bool) for v in values):
        return None
    return sum(values)


def _station(out: Path, folder: Path) -> dict[str, Any]:
    agent = _read(folder / "agent.json")
    checks = _read(folder / "checks.json")
    events = _get(agent, "events")
    return {
        "station": folder.name,
        "wallSeconds": _get(agent, "seconds"),
        "exitCode": _get(agent, "exitCode"),
        "timedOut": _get(agent, "timedOut"),
        "leftoverProcessesKilled": _get(agent, "leftoverProcessesKilled"),
        "sessionId": _get(agent, "sessionId"),
        "sessionSwitched": _get(agent, "sessionSwitched"),
        "tokens": {kind: _get(agent, "usage", "outer", kind) for kind in TOKEN_KINDS},
        "tokensAllSessions": {kind: _get(agent, "usage", "all", kind) for kind in TOKEN_KINDS},
        "otherSessions": _get(agent, "usage", "otherSessions"),
        "commands": _get(events, "commands"),
        "fileChanges": _get(events, "fileChanges"),
        "mcpToolCalls": _get(events, "mcpToolCalls"),
        "mcpToolFailures": _get(events, "mcpToolFailures"),
        "mcpCallErrors": _get(events, "mcpCallErrors"),
        "collabToolCalls": _get(events, "collabToolCalls"),
        "agentItems": _get(events, "items"),
        "apiRetries": _get(events, "apiRetries"),
        "agentErrors": _get(events, "errors"),
        "newMainCommits": _get(agent, "newMainCommits"),
        "mainCommit": _get(agent, "mainCommit"),
        "checks": {"passed": _get(checks, "passed"), "total": _get(checks, "total"),
                   "status": _get(checks, "status")},
        "harnessErrors": sorted(path.name for path in folder.glob("*-error.txt")),
        "artifacts": folder.relative_to(out).as_posix(),
    }


def _final_summary(final: dict[str, Any] | None) -> dict[str, Any] | None:
    if final is None:
        return None
    own, conformance = final.get("ownTests"), final.get("conformance")
    return {
        "passed": final.get("passed"), "total": final.get("total"), "status": final.get("status"),
        "station": _get(final, "checks", "station"),
        "ownTests": None if not isinstance(own, dict) else
        {key: own.get(key) for key in ("status", "ran", "failures", "errors", "skipped")},
        "conformance": None if not isinstance(conformance, dict) else
        {key: conformance.get(key) for key in ("status", "reportStatus", "coverageConforming", "findings",
                                               "error", "errorSource")},
        "error": final.get("error"),
        "artifacts": "final",
    }


def _setup_notes(setup: dict[str, Any] | None) -> list[str]:
    """Frictions a normal user would also meet: tolerated step failures, explicit Codex path."""
    if _get(setup, "status") != "ready":
        return []
    notes = [f"{step.get('name')} exited {step.get('exitCode')} (tolerated)"
             for step in setup.get("steps") or [] if isinstance(step, dict) and step.get("exitCode") not in (0, None)]
    provider = _get(setup, "notes", "providerExecutable")
    if provider:
        notes.append(f"the native Codex binary is passed explicitly ({provider}); the product does not "
                     "accept the npm launcher that `codex` on PATH points to")
    return notes


def _roles(setup: dict[str, Any] | None) -> list[dict[str, Any]] | None:
    """Executor, model and effort per Markitect role as the product configured them."""
    roles = _get(setup, "roles")
    if not isinstance(roles, list):
        return None
    keys = ("role", "manager", "executor", "providerVersion", "model", "effort")
    return [{key: role.get(key) for key in keys} for role in roles if isinstance(role, dict)]


def _stratum(manifest: dict[str, Any]) -> str | None:
    """Runs with different outer providers are never pooled; Markitect's inner roles run on Codex."""
    kind = _get(manifest, "agent", "kind")
    if kind is None:
        return None
    return f"outer={kind}" + (", inner=codex" if manifest.get("method") == "markitect" else "")


def _fairness(manifest: dict[str, Any], versions: dict[str, Any], runner: dict[str, Any] | None) -> dict[str, Any]:
    """Fields that must be equal for the two arms of one comparison (the stratum is not:
    only the Markitect arm has inner roles). Runs from different host platforms are never
    paired (DEC-013)."""
    agent = manifest.get("agent") or {}
    return {"case": manifest.get("case"), "stationsPlanned": _get(runner, "stationsPlanned"),
            "outerProvider": agent.get("kind"),
            "codex": versions.get("codex"), "claude": versions.get("claude"), "imageId": versions.get("imageId"),
            "model": agent.get("model"), "effort": agent.get("effort"),
            "maxSubagentsPerSession": agent.get("maxSubagents"),
            "limits": manifest.get("limits"), "container": manifest.get("container"),
            "hostPlatform": _get(runner, "hostPlatform")}


def _last_line(path: Path) -> str:
    try:
        lines = [line for line in path.read_text(encoding="utf-8", errors="replace").splitlines() if line.strip()]
    except OSError:
        return "no traceback"
    return lines[-1].strip()[:500] if lines else "no traceback"


def _classification(out: Path, runner: dict | None, setup: dict | None, stations: list[dict],
                    final: dict | None) -> dict[str, Any]:
    decisive: list[dict[str, str]] = []
    signals: list[dict[str, str]] = []
    if (out / "runner-error.txt").is_file() or _get(runner, "status") == "error":
        decisive.append({"class": "harness", "reason": f"runner error: {_last_line(out / 'runner-error.txt')}"})
    category, reason = _get(runner, "stopCategory"), _get(runner, "stopReason")
    if reason and category in CATEGORIES:
        decisive.append({"class": category, "reason": reason})
    elif _get(setup, "status") == "blocked":  # a runner.json written before stop categories existed
        decisive.append({"class": _get(setup, "blockedBy") or "product",
                         "reason": f"method setup blocked: {_get(setup, 'error')}"})
    if str(_get(final, "error") or "").startswith("Traceback"):
        decisive.append({"class": "harness", "reason": "final assessment failed (see final/final.json)"})
    conformance = _get(final, "conformance")
    if _get(conformance, "errorSource"):
        decisive.append({"class": _get(conformance, "errorSource"),
                         "reason": f"markitect check: {_get(conformance, 'error')}"})
    for station in stations:
        # Timeouts never stop a run, but a whole station without any agent activity means
        # the provider never answered.
        if (station.get("timedOut") and station.get("agentItems") == 0
                and all(value is None for value in station["tokens"].values())):
            retries = f" ({station['apiRetries']} provider request retries)" if station.get("apiRetries") else ""
            decisive.append({"class": "environment",
                             "reason": f"{station['station']}: timed out without any agent activity{retries}"})
        for name in station["harnessErrors"]:
            signals.append({"class": "harness", "reason": f"{station['station']}: {name}"})
        if station.get("mcpCallErrors"):
            signals.append({"class": "product", "reason": f"{station['station']}: {station['mcpCallErrors']} "
                                                             "MCP call(s) to the product could not run"})
    for wanted in CATEGORIES:
        found = next((item for item in decisive if item["class"] == wanted), None)
        if found:
            return {**found, "signals": decisive + signals}
    planned = _get(runner, "stationsPlanned")
    return {"class": "none", "reason": reason or (f"completed S1-S{planned}" if planned else "completed"),
            "signals": signals}


def _station_folders(out: Path) -> list[Path]:
    root = out / "stations"
    if not root.is_dir():
        return []
    folders = [p for p in root.iterdir() if p.is_dir() and p.name[:1] == "S" and p.name[1:].isdigit()]
    return sorted(folders, key=lambda p: int(p.name[1:]))


def build(out: Path) -> dict[str, Any]:
    out = Path(out)
    runner = _read(out / "runner.json")
    setup = _read(out / "setup" / "setup.json")
    final = _read(out / "final" / "final.json")
    stations = [_station(out, folder) for folder in _station_folders(out)]
    manifest = _get(runner, "manifest") or {}
    versions = _get(runner, "versions") or {}
    report = {
        "schema": 1,
        "manifest": _get(runner, "manifest"),
        "status": _get(runner, "status"),
        "exitCode": _get(runner, "exitCode"),
        "stopReason": _get(runner, "stopReason"),
        "stopCategory": _get(runner, "stopCategory"),
        "classification": _classification(out, runner, setup, stations, final),
        "roles": _roles(setup),
        "runnerError": (out / "runner-error.txt").is_file(),
        "codexLoginChanged": _get(runner, "codexLoginChanged"),
        "claudeTokenRedactions": _get(runner, "tokenRedactions"),
        "startedAt": _get(runner, "startedAt"),
        "endedAt": _get(runner, "endedAt"),
        "versions": {key: versions.get(key) for key in
                     ("codex", "claude", "imageId", "markitectCommit", "markitectSha256")},
        "stratum": _stratum(manifest),
        "fairness": _fairness(manifest, versions, runner),
        "setup": {"status": _get(setup, "status"), "seconds": _get(setup, "seconds"),
                  "commit": _get(setup, "commit"), "error": _get(setup, "error"),
                  "blockedBy": _get(setup, "blockedBy"),
                  "leftoverProcessesKilled": _get(setup, "leftoverProcessesKilled"),
                  "claudeRouter": _get(setup, "notes", "claudeRouter"),
                  "onboardProvider": _get(setup, "notes", "onboardProvider"),
                  "roles": _roles(setup),
                  "notes": _setup_notes(setup)},
        "stations": stations,
        "final": _final_summary(final),
        "totals": {
            "tokens": {kind: sum_known([s["tokens"][kind] for s in stations]) for kind in TOKEN_KINDS},
            "tokensAllSessions": {kind: sum_known([s["tokensAllSessions"][kind] for s in stations])
                                  for kind in TOKEN_KINDS},
            "agentSeconds": sum_known([s["wallSeconds"] for s in stations]),
            "wallSeconds": _get(runner, "wallSeconds"),
            "stationsPlanned": _get(runner, "stationsPlanned"),
            "caseStations": _get(runner, "caseStations"),
            "stationsRun": len(stations),
        },
    }
    (out / "report.json").write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n",
                                     encoding="utf-8", newline="\n")
    (out / "report.md").write_text(render_markdown(report), encoding="utf-8", newline="\n")
    return report


# --- Markdown ------------------------------------------------------------------------

def _fmt(value: Any) -> str:
    if value is None:
        return "n/a"
    if isinstance(value, bool):
        return "yes" if value else "no"
    if isinstance(value, float):
        return f"{value:,.1f}"
    if isinstance(value, int):
        return f"{value:,}"
    if isinstance(value, dict):
        return ", ".join(f"{key} {_fmt(item)}" for key, item in value.items())
    return str(value)


def _platform(value: Any) -> str:
    if not isinstance(value, dict):
        return "n/a"
    return f"{_fmt(value.get('system'))} {_fmt(value.get('machine'))}"


def _ratio(block: Any) -> str:
    passed, total = _get(block, "passed"), _get(block, "total")
    status = _get(block, "status")
    if passed is None and total is None:
        return _fmt(status)
    return f"{_fmt(passed)}/{_fmt(total)}" + (f" ({status})" if status else "")


def _tokens(tokens: dict[str, Any]) -> str:
    return " / ".join(_fmt(tokens.get(kind)) for kind in TOKEN_KINDS)


def _role_text(role: dict[str, Any]) -> str:
    who = role["role"] + (" " + role["manager"] if role.get("manager") else "")
    return (f"{who}: {_fmt(role.get('executor'))} ({_fmt(role.get('providerVersion'))}), "
            f"model {_fmt(role.get('model'))}, effort {_fmt(role.get('effort'))}")


def render_markdown(report: dict[str, Any]) -> str:
    manifest = report.get("manifest") or {}
    agent = manifest.get("agent") or {}
    versions = report["versions"]
    claude = agent.get("kind") in ("claude", "fake-claude")
    lines = [f"# Run {manifest.get('id', 'n/a')}", ""]
    lines.append(f"Case **{_fmt(manifest.get('case'))}**, method **{_fmt(manifest.get('method'))}**, "
                 f"agent {_fmt(agent.get('kind'))} (model {_fmt(agent.get('model'))}, "
                 f"effort {_fmt(agent.get('effort'))}, max subagents per session {_fmt(agent.get('maxSubagents'))}"
                 + (", recorded only" if claude else "") + f"); stratum {_fmt(report.get('stratum'))}.")
    outcome = f"Status: {_fmt(report.get('status'))} (exit {_fmt(report.get('exitCode'))})"
    if report.get("stopReason"):
        outcome += f"; stop reason: {report['stopReason']}"
    if report.get("runnerError"):
        outcome += "; see runner-error.txt"
    classification = report.get("classification") or {}
    lines += [outcome + ".",
              f"Classification: **{_fmt(classification.get('class'))}** ({_fmt(classification.get('reason'))}).",
              ""]
    lines.append("| Station | Wall s | Exit | Timed out | Killed | Tokens outer | Tokens all (other sessions) "
                 "| Commands / MCP / subagent calls | New main commits | Public checks |")
    lines.append("|---|---|---|---|---|---|---|---|---|---|")
    for s in report["stations"]:
        lines.append(f"| {s['station']} | {_fmt(s['wallSeconds'])} | {_fmt(s['exitCode'])} "
                     f"| {_fmt(s['timedOut'])} | {_fmt(s['leftoverProcessesKilled'])} "
                     f"| {_tokens(s['tokens'])} | {_tokens(s['tokensAllSessions'])} ({_fmt(s['otherSessions'])}) "
                     f"| {_fmt(s['commands'])} / {_fmt(s['mcpToolCalls'])} / {_fmt(s['collabToolCalls'])} "
                     f"| {_fmt(s['newMainCommits'])} | {_ratio(s['checks'])} |")
    if not report["stations"]:
        lines.append("| (no station ran) | | | | | | | | | |")
    totals = report["totals"]
    setup = report["setup"]
    final = report.get("final") or {}
    fairness = report["fairness"]
    lines += [
        "",
        f"- Stations: {_fmt(totals.get('stationsRun'))} of {_fmt(totals.get('stationsPlanned'))} ran"
        + (f" (the first {totals['stationsPlanned']} of the case's {totals['caseStations']})"
           if isinstance(totals.get("caseStations"), int) and totals.get("stationsPlanned") != totals["caseStations"]
           else "") + ".",
        f"- Setup: {_fmt(setup['status'])} in {_fmt(setup['seconds'])} s"
        + (f", commit {setup['commit'][:12]}" if setup.get("commit") else "")
        + (f", error: {setup['error']}" if setup.get("error") else "") + "."
        + (f" Notes: {'; '.join(setup['notes'])}." if setup.get("notes") else ""),
    ]
    router = setup.get("claudeRouter")
    if isinstance(router, dict):
        lines.append(f"- CLAUDE.md router (`{str(router.get('content') or '').strip()}`, the same for both "
                     "methods): " + ("added." if router.get("added") else
                                     "not added, the repository already had a CLAUDE.md."))
    if setup.get("roles"):
        lines.append(f"- Markitect roles (.markitect/runtime.yaml after setup; onboarding for "
                     f"{_fmt(setup.get('onboardProvider'))}): {'; '.join(_role_text(r) for r in setup['roles'])}.")
    lines += [
        f"- Final assessment: public checks {_ratio(final) if final else 'n/a'}"
        + (f" (S{final['station']})" if final.get("station") else "")
        + (f"; own tests {_fmt(_get(final, 'ownTests', 'status'))} (ran {_fmt(_get(final, 'ownTests', 'ran'))}, "
           f"failures {_fmt(_get(final, 'ownTests', 'failures'))}, errors {_fmt(_get(final, 'ownTests', 'errors'))})"
           if final.get("ownTests") else "")
        + (f"; Markitect conformance {_fmt(_get(final, 'conformance', 'status'))} "
           f"(report {_fmt(_get(final, 'conformance', 'reportStatus'))})" if final.get("conformance") else "")
        + ".",
        f"- Totals: tokens outer {_tokens(totals['tokens'])}; all sessions {_tokens(totals['tokensAllSessions'])}; "
        f"agent {_fmt(totals['agentSeconds'])} s; run wall {_fmt(totals['wallSeconds'])} s.",
        f"- {CLAUDE_TOKEN_NOTE if claude else TOKEN_NOTE}",
    ]
    switched = [s["station"] for s in report["stations"] if s.get("sessionSwitched")]
    if switched:
        lines.append(f"- Resume started a new session at {', '.join(switched)}; the earlier context was not continued.")
    others = [item["class"] + ": " + item["reason"] for item in classification.get("signals") or []
              if item.get("reason") != classification.get("reason")]
    if others:
        lines.append(f"- Other signals: {'; '.join(others)}.")
    if report.get("codexLoginChanged"):
        lines.append("- Codex rewrote its login inside the run (likely a token refresh). Log in again on the "
                     "host before the next run.")
    if report.get("claudeTokenRedactions"):
        lines.append(f"- The Claude Code token appeared in {report['claudeTokenRedactions']} result file(s) and was "
                     "redacted there (the agent printed its environment).")
    lines += [
        f"- Fairness (must match the other arm): case {_fmt(fairness['case'])}; stations "
        f"{_fmt(fairness.get('stationsPlanned'))}; host {_platform(fairness.get('hostPlatform'))}; outer provider "
        f"{_fmt(fairness['outerProvider'])}; codex {_fmt(fairness['codex'])}; "
        f"claude {_fmt(fairness['claude'])}; image {_fmt(fairness['imageId'])}; model {_fmt(fairness['model'])}; "
        f"effort {_fmt(fairness['effort'])}; max subagents per session {_fmt(fairness['maxSubagentsPerSession'])}; "
        f"limits {_fmt(fairness['limits'])}; container {_fmt(fairness['container'])}.",
        f"- Versions: codex {_fmt(versions['codex'])}; claude {_fmt(versions['claude'])}; "
        f"image {_fmt(versions['imageId'])}; "
        f"markitect {_fmt(versions['markitectCommit'])} (sha256 {_fmt(versions['markitectSha256'])}).",
        "",
    ]
    return "\n".join(lines)


if __name__ == "__main__":
    import sys
    build(Path(sys.argv[1]))
