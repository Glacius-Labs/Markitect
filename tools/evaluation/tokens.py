"""The suite's token count of one run: every transcript the run left, subagents included.

  python -I -B tokens.py DIR

Claude Code writes one transcript line per content block of an API response,
and each line repeats the response's usage. The parser reads every *.jsonl file
below DIR: Claude Code transcripts, subagent transcripts
(<session>/subagents/**/agent-*.jsonl) and stream-json output alike. It keeps
the assistant responses and counts each response once, by its message ID, with
the largest value of each usage field seen for it, so a response that appears
in several files, or in several lines, counts once. Thinking tokens are part of
the output tokens and are shown apart. A session's own `cost-state` totals are
reported next to the count, as Claude Code states them, for comparison.

The count is descriptive: the report shows it, nothing is optimized against it.
"""
import json
import pathlib
import sys

FIELDS = {"inputTokens": "input_tokens", "cacheCreationTokens": "cache_creation_input_tokens",
          "cacheReadTokens": "cache_read_input_tokens", "outputTokens": "output_tokens"}
CLI_FIELDS = {"inputTokens": "inputTokens", "cacheCreationTokens": "cacheCreationInputTokens",
              "cacheReadTokens": "cacheReadInputTokens", "outputTokens": "outputTokens", "thinkingTokens": "thinkingTokens"}


def number(value):
    return value if isinstance(value, int) and not isinstance(value, bool) and value >= 0 else 0


def lines(path):
    for raw in path.read_text(encoding="utf-8", errors="replace").splitlines():
        try:
            entry = json.loads(raw)
        except ValueError:
            yield None
            continue
        yield entry if isinstance(entry, dict) else None


def count(root):
    """Tokens of every response below root, each counted once; the sessions' own totals for comparison."""
    root = pathlib.Path(root)
    responses, cli, files, unreadable = {}, {}, 0, 0
    for path in sorted(root.rglob("*.jsonl")):
        if path.is_symlink() or not path.is_file():
            continue
        files += 1
        for entry in lines(path):
            if entry is None:
                unreadable += 1
                continue
            if entry.get("type") == "cost-state" and isinstance(entry.get("modelUsage"), dict):
                # The last state of each session holds its totals.
                cli[entry.get("sessionId")] = entry["modelUsage"]
                continue
            message = entry.get("message")
            if entry.get("type") != "assistant" or not isinstance(message, dict) or not isinstance(message.get("usage"), dict):
                continue
            key = message.get("id") or entry.get("uuid")
            if not key:
                unreadable += 1
                continue
            usage = message["usage"]
            seen = responses.setdefault(key, {"model": message.get("model"), "subagent": False,
                                              **{name: 0 for name in FIELDS}, "thinkingTokens": 0})
            for name, field in FIELDS.items():
                seen[name] = max(seen[name], number(usage.get(field)))
            details = usage.get("output_tokens_details")
            seen["thinkingTokens"] = max(seen["thinkingTokens"], number((details or {}).get("thinking_tokens") if isinstance(details, dict) else 0))
            seen["subagent"] = seen["subagent"] or bool(entry.get("isSidechain")) or bool(entry.get("parent_tool_use_id"))
    totals = {name: sum(r[name] for r in responses.values()) for name in (*FIELDS, "thinkingTokens")}
    models = {}
    for r in responses.values():
        m = models.setdefault(r["model"] or "unknown", {"responses": 0, **{name: 0 for name in (*FIELDS, "thinkingTokens")}})
        m["responses"] += 1
        for name in (*FIELDS, "thinkingTokens"):
            m[name] += r[name]
    reported = None
    if cli:
        reported = {name: sum(number(u.get(field)) for usage in cli.values() for u in usage.values() if isinstance(u, dict))
                    for name, field in CLI_FIELDS.items()}
    return {"files": files, "responses": len(responses), "subagentResponses": sum(r["subagent"] for r in responses.values()),
            **totals, "byModel": models, "unreadableLines": unreadable, "sessionTotals": reported}


def main(argv=None):
    argv = list(sys.argv[1:] if argv is None else argv)
    if len(argv) != 1:
        print(__doc__, file=sys.stderr)
        return 2
    print(json.dumps(count(argv[0]), indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
