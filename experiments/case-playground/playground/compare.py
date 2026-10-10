"""Compare two assessed runs side by side.

  python -m playground compare RUN_A RUN_B [--allow-mismatch] [--out FILE]

Reads `<run>/assessment/report.json` of both runs. First it checks that the runs are
comparable: the run's fairness fields (case, stations, host platform, image, Codex
version, model, effort, subagent limit, limits, container, as the run report lists
them), the outer provider, the evaluation files (by SHA-256) and the reviewer models.
Mismatched runs are refused unless `--allow-mismatch` is given; the mismatches are then
printed at the top. The comparison goes to `--out` (default: `compare-<A>-vs-<B>.md`
next to RUN_A).
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

from .reviewers import CATEGORIES, PROVIDERS


class CompareError(RuntimeError):
    pass


def load(run_dir: Path) -> dict:
    path = Path(run_dir) / "assessment" / "report.json"
    try:
        report = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, ValueError) as exc:
        raise CompareError(f"no readable assessment report: {path} ({exc})") from exc
    if not isinstance(report, dict) or report.get("kind") != "assessment":
        raise CompareError(f"not an assessment report: {path}")
    return report


def fairness_fields(report: dict) -> dict[str, Any]:
    """Everything that must match for a fair comparison, flattened to field -> value."""
    run = report.get("run") or {}
    fields: dict[str, Any] = {"case": run.get("case"), "outerProvider": run.get("outerProvider")}
    for key, value in sorted((run.get("fairness") or {}).items()):
        fields[f"fairness.{key}"] = value
    for key, value in sorted(((report.get("evaluation") or {}).get("files") or {}).items()):
        fields[f"evaluation.{key}"] = (value or {}).get("sha256")
    for name, cfg in sorted((report.get("reviewers") or {}).items()):
        # The CLI version also tells a real reviewer from the smoke tests' fake one.
        fields[f"reviewer.{name}"] = f"{cfg.get('model')} (effort {cfg.get('effort')}, {cfg.get('version')})"
    return fields


def mismatches(a: dict, b: dict) -> list[tuple[str, Any, Any]]:
    first, second = fairness_fields(a), fairness_fields(b)
    return [(key, first.get(key), second.get(key)) for key in sorted(set(first) | set(second))
            if first.get(key) != second.get(key)]


# --- Markdown --------------------------------------------------------------------------

def _fmt(value: Any) -> str:
    if value is None:
        return "n/a"
    if isinstance(value, bool):
        return "yes" if value else "no"
    if isinstance(value, float):
        return f"{value:,.1f}"
    if isinstance(value, int):
        return f"{value:,}"
    if isinstance(value, (dict, list)):
        return json.dumps(value, sort_keys=True)
    return str(value)


def _pair(block: dict | None) -> str:
    if not block or (block.get("passed") is None and block.get("total") is None):
        return "n/a"
    text = f"{_fmt(block.get('passed'))}/{_fmt(block.get('total'))}"
    return text + (f" +{block['errors']} not judged" if block.get("errors") else "")


def _lines(diff: dict | None, category: str | None = None) -> str:
    if not diff or diff.get("status") != "ok":
        return "n/a"
    block = diff["categories"][category] if category else diff
    return f"+{block['added']}/-{block['deleted']}"


def _findings(entry: dict | None, name: str) -> str:
    summary = ((entry or {}).get("reviewers") or {}).get(name)
    if not summary:
        return "n/a"
    if summary.get("status") != "ok":
        return str(summary.get("status"))
    return f"{summary['findings']} ({summary['bySeverity']['high']} high)"


def _obligations(entry: dict | None, name: str) -> str:
    block = (((entry or {}).get("reviewers") or {}).get(name) or {}).get("obligations")
    return f"{block['covered']}/{block['total']}" if block else "n/a"


def _escalations(entry: dict | None) -> str:
    parts = [f"{name} {block['needed']}/{block['unneeded']}"
             for name, block in ((entry or {}).get("escalations") or {}).items() if block]
    return ", ".join(parts) or "n/a"


def _cell(a: str, b: str) -> str:
    return f"{a} │ {b}"


def render(a: dict, b: dict, problems: list[tuple[str, Any, Any]]) -> str:
    run_a, run_b = a["run"], b["run"]
    label_a = f"{run_a.get('id')} ({run_a.get('method')})"
    label_b = f"{run_b.get('id')} ({run_b.get('method')})"
    lines = [f"# Comparison: {label_a} vs {label_b}", ""]
    if problems:
        lines += ["**Fairness mismatch, compared anyway (--allow-mismatch).** These fields differ, so "
                  "differences below may come from them and not from the method:", ""]
        lines += [f"- `{key}`: {_fmt(value_a)} vs {_fmt(value_b)}" for key, value_a, value_b in problems]
        lines.append("")
    else:
        lines += ["Fairness fields match (case, stations, host platform, image, versions, model, effort, "
                  "limits, container, outer provider, evaluation files, reviewer models).", ""]
    lines += [f"Each cell shows A │ B. A = {label_a}, B = {label_b}. Outer provider "
              f"{_fmt(run_a.get('outerProvider'))} │ {_fmt(run_b.get('outerProvider'))}; case "
              f"{_fmt(run_a.get('case'))}; classification {a['classification']['class']} │ "
              f"{b['classification']['class']}.", ""]
    waves_a = {entry["station"]: entry for entry in a.get("stations") or []}
    waves_b = {entry["station"]: entry for entry in b.get("stations") or []}
    waves = sorted(set(waves_a) | set(waves_b), key=lambda name: int(name[1:]) if name[1:].isdigit() else 0)
    names = [name for name in PROVIDERS if name in (a.get("reviewers") or {}) or name in (b.get("reviewers") or {})]

    lines += ["## Outcome per wave", "",
              "| Wave | Public checks | Holdouts | Diff lines | Model lines | Class |", "|---|---|---|---|---|---|"]
    for wave in waves:
        ea, eb = waves_a.get(wave), waves_b.get(wave)
        lines.append("| " + " | ".join([
            wave,
            _cell(_pair((ea or {}).get("publicChecks")), _pair((eb or {}).get("publicChecks"))),
            _cell(_pair((ea or {}).get("holdouts")), _pair((eb or {}).get("holdouts"))),
            _cell(_lines((ea or {}).get("diff")), _lines((eb or {}).get("diff"))),
            _cell(_lines((ea or {}).get("diff"), "model"), _lines((eb or {}).get("diff"), "model")),
            _cell(((ea or {}).get("classification") or {}).get("class", "n/a"),
                  ((eb or {}).get("classification") or {}).get("class", "n/a"))]) + " |")

    if names:
        header = ["Wave"] + [f"Findings {name}" for name in names] + (["Shared"] if len(names) == 2 else [])
        header += [f"Obligations {name}" for name in names] + ["Escalations needed/unneeded"]
        lines += ["", "## Reviews per wave", "", "| " + " | ".join(header) + " |", "|" + "---|" * len(header)]
        for wave in waves:
            ea, eb = waves_a.get(wave), waves_b.get(wave)
            row = [wave] + [_cell(_findings(ea, name), _findings(eb, name)) for name in names]
            if len(names) == 2:
                row.append(_cell(_fmt(((ea or {}).get("agreement") or {}).get("both")),
                                 _fmt(((eb or {}).get("agreement") or {}).get("both"))))
            row += [_cell(_obligations(ea, name), _obligations(eb, name)) for name in names]
            row.append(_cell(_escalations(ea), _escalations(eb)))
            lines.append("| " + " | ".join(row) + " |")
        lines += ["", "## Findings by category, all waves", "",
                  "| Category | " + " | ".join(f"{name}" for name in names) + " |", "|---|" + "---|" * len(names)]
        for category in CATEGORIES:
            cells = []
            for name in names:
                values = [str((((report.get("totals") or {}).get("reviewers") or {}).get(name) or {})
                              .get("byCategory", {}).get(category, "n/a")) for report in (a, b)]
                cells.append(_cell(*values))
            lines.append(f"| {category} | " + " | ".join(cells) + " |")

    totals_a, totals_b = run_a.get("totals") or {}, run_b.get("totals") or {}
    lines += ["", "## Run totals", "",
              f"- Agent seconds: {_cell(_fmt(totals_a.get('agentSeconds')), _fmt(totals_b.get('agentSeconds')))}.",
              f"- Tokens outer (in/cached/out): {_cell(_tokens(totals_a.get('tokens')), _tokens(totals_b.get('tokens')))}.",
              f"- Tokens all sessions: {_cell(_tokens(totals_a.get('tokensAllSessions')), _tokens(totals_b.get('tokensAllSessions')))}.",
              f"- Public checks: {_cell(_pair(a['totals']['publicChecks']), _pair(b['totals']['publicChecks']))}; "
              f"holdouts: {_cell(_pair(a['totals'].get('holdouts')), _pair(b['totals'].get('holdouts')))}; "
              f"failed Markitect MCP calls: {_cell(_fmt(a['totals'].get('failedMcpCalls')), _fmt(b['totals'].get('failedMcpCalls')))}.",
              "- One pair of runs shows mechanisms, not a general effect. Cost is a secondary observation.", ""]
    return "\n".join(lines)


def _tokens(block: dict | None) -> str:
    if not block:
        return "n/a"
    return " / ".join(_fmt(block.get(kind)) for kind in ("input", "cachedInput", "output"))


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="python -m playground compare", description=__doc__,
                                     formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("run_a")
    parser.add_argument("run_b")
    parser.add_argument("--allow-mismatch", action="store_true")
    parser.add_argument("--out")
    args = parser.parse_args(sys.argv[1:] if argv is None else argv)
    try:
        a, b = load(Path(args.run_a)), load(Path(args.run_b))
    except CompareError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 2
    problems = mismatches(a, b)
    if problems and not args.allow_mismatch:
        print("error: the runs are not comparable; these fairness fields differ "
              "(use --allow-mismatch to compare anyway):", file=sys.stderr)
        for key, value_a, value_b in problems:
            print(f"  {key}: {_fmt(value_a)} vs {_fmt(value_b)}", file=sys.stderr)
        return 2
    out = Path(args.out) if args.out else (
        Path(args.run_a).resolve().parent / f"compare-{a['run'].get('id')}-vs-{b['run'].get('id')}.md")
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(render(a, b, problems), encoding="utf-8", newline="\n")
    print(f"comparison: {out}")
    return 0
