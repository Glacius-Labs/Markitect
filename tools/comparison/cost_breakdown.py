"""Break a Markitect Apply's cost down by Run type, Manager level and tool call.

Addendum C predeclares this analysis so that the cost of arm M is explained
from its own records, not from measures chosen after the data. Every session
transcript of an Apply (state/<project>/transcripts/<applyID>/*.jsonl) is one
Manager session: its harness instructions name the Manager's Package and the
Run's mandate, the provider's last token-usage event gives its tokens, and its
completed items give its tool calls.

  python -I -B cost_breakdown.py TRANSCRIPT_DIR [TRANSCRIPT_DIR ...]

Prints per-session rows and totals by mandate, Manager depth (Root is 0),
Package and tool. Wall time is the span between a session's first and last
recorded event; sessions of one Run overlap, so their sum exceeds the Run's
elapsed time. A session whose instructions lack a section is grouped under
"None" for it, and lines that are not JSON are counted as unreadable_lines.
"""
import argparse
import collections
import datetime
import json
import pathlib
import re
import sys

USAGE_FIELDS = ("totalTokens", "inputTokens", "cachedInputTokens", "outputTokens", "reasoningOutputTokens")


def _section(text, title):
    """The line after `title:` in the harness instructions, parsed as JSON; None when absent or not JSON."""
    lines = text.split("\n")
    for i, line in enumerate(lines[:-1]):
        if line.strip() == title + ":":
            try:
                return json.loads(lines[i + 1])
            except ValueError:
                return None
    return None


def _time(at):
    """A UTC transcript timestamp such as 2026-10-25T09:09:56.5516496Z (any number of fraction digits)."""
    text = re.sub(r"(Z|[+-]00:00)$", "", at)
    if "." in text:
        head, fraction = text.split(".", 1)
        text = head + "." + fraction[:6]
    return datetime.datetime.fromisoformat(text)


def depths(hierarchy):
    parents = {m["Package"]: m["Parent"] for m in hierarchy or []}

    def depth(package):
        steps = 0
        while parents.get(package):
            package, steps = parents[package], steps + 1
        return steps
    return {package: depth(package) for package in parents}


def session(path):
    """One Manager session's Package, mandate, depth, tokens, tool calls and wall time."""
    row = {"transcript": path.name, "package": None, "mandate": None, "depth": None, "turns": 0, "unreadable_lines": 0,
           "tools": collections.Counter(), **{field: 0 for field in USAGE_FIELDS}}
    first = last = None
    with path.open(encoding="utf-8") as stream:
        for line in stream:
            try:
                record = json.loads(line) if line.strip() else None
                message = json.loads(record["data"]) if record and isinstance(record.get("data"), str) else None
            except ValueError:
                row["unreadable_lines"] += 1
                continue
            if record and record.get("at"):
                stamp = _time(record["at"])
                first, last = first or stamp, stamp
            if not isinstance(message, dict):
                continue
            method = message.get("method")
            params = message.get("params") or {}
            if method == "thread/start":
                text = params.get("developerInstructions") or ""
                manager, hierarchy = _section(text, "Manager"), _section(text, "Complete Manager hierarchy")
                row["package"] = (manager or {}).get("Package")
                row["mandate"] = _section(text, "Mandate")
                row["depth"] = depths(hierarchy).get(row["package"])
            elif method == "turn/start":
                row["turns"] += 1
            elif method == "thread/tokenUsage/updated":
                # The total is cumulative; a partial event updates only the fields it carries.
                total = (params.get("tokenUsage") or {}).get("total") or {}
                row.update({field: total[field] for field in USAGE_FIELDS if field in total})
            elif method == "item/completed":
                item = params.get("item") or {}
                kind = item.get("type")
                if kind in ("dynamicToolCall", "mcpToolCall"):
                    row["tools"][item.get("tool") or "?"] += 1
                elif kind in ("commandExecution", "fileChange", "webSearch"):
                    row["tools"][kind] += 1
    row["seconds"] = round((last - first).total_seconds(), 1) if first and last else 0.0
    row["tools"] = dict(row["tools"])
    return row


def breakdown(directories):
    rows = [session(path) for directory in directories for path in sorted(pathlib.Path(directory).glob("*.jsonl"))]

    def totals(key):
        groups = {}
        for row in rows:
            group = groups.setdefault(str(row[key]), {"sessions": 0, "seconds": 0.0, "turns": 0, "tool_calls": 0,
                                                       **{field: 0 for field in USAGE_FIELDS}})
            group["sessions"] += 1
            group["seconds"] = round(group["seconds"] + row["seconds"], 1)
            group["turns"] += row["turns"]
            group["tool_calls"] += sum(row["tools"].values())
            for field in USAGE_FIELDS:
                group[field] += row[field]
        return dict(sorted(groups.items()))
    tools = collections.Counter()
    for row in rows:
        tools.update(row["tools"])
    return {"sessions": rows, "by_mandate": totals("mandate"), "by_depth": totals("depth"), "by_package": totals("package"),
            "tool_calls": dict(sorted(tools.items())), "total": {field: sum(r[field] for r in rows) for field in USAGE_FIELDS}}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    p.add_argument("directories", nargs="+")
    a = p.parse_args(argv)
    print(json.dumps(breakdown(a.directories), indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
