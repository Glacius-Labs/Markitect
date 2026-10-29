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

A session that left only stream-json output (`session_id` on its lines, no
transcript line with `sessionId`), such as each agent of the 2.0 arm, is
counted from its `result` lines instead: stream-json repeats each response's
usage as it stood when a content block was written, so its output tokens fall
short, while each `claude -p` process ends with one `result` line whose
`modelUsage` totals that process, thinking included. Such sessions add their
stated cost to `costUsd`.

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
    responses, cli, files, unreadable, results, transcript_sessions = {}, {}, 0, 0, {}, set()
    for path in sorted(root.rglob("*.jsonl")):
        if path.is_symlink() or not path.is_file():
            continue
        files += 1
        for entry in lines(path):
            if entry is None:
                unreadable += 1
                continue
            if entry.get("sessionId"):
                transcript_sessions.add(entry["sessionId"])
            if entry.get("type") == "cost-state" and isinstance(entry.get("modelUsage"), dict):
                # The last state of each session holds its totals.
                cli[entry.get("sessionId")] = entry["modelUsage"]
                continue
            if entry.get("type") == "result" and entry.get("session_id") and isinstance(entry.get("modelUsage"), dict):
                results.setdefault(entry["session_id"], []).append(entry["modelUsage"])
                continue
            message = entry.get("message")
            if entry.get("type") != "assistant" or not isinstance(message, dict) or not isinstance(message.get("usage"), dict):
                continue
            key = message.get("id") or entry.get("uuid")
            if not key:
                unreadable += 1
                continue
            usage = message["usage"]
            seen = responses.setdefault(key, {"model": message.get("model"), "subagent": False, "session": entry.get("session_id") or entry.get("sessionId"),
                                              **{name: 0 for name in FIELDS}, "thinkingTokens": 0})
            for name, field in FIELDS.items():
                seen[name] = max(seen[name], number(usage.get(field)))
            details = usage.get("output_tokens_details")
            seen["thinkingTokens"] = max(seen["thinkingTokens"], number((details or {}).get("thinking_tokens") if isinstance(details, dict) else 0))
            seen["subagent"] = seen["subagent"] or bool(entry.get("isSidechain")) or bool(entry.get("parent_tool_use_id"))
    # Stream-only sessions are counted from their result lines; every other response as it was seen.
    stream_only = {s: usages for s, usages in results.items() if s not in transcript_sessions}
    counted = [r for r in responses.values() if r["session"] not in stream_only]
    names = (*FIELDS, "thinkingTokens")
    totals = {name: sum(r[name] for r in counted) for name in names}
    models = {}
    for r in counted:
        m = models.setdefault(r["model"] or "unknown", {"responses": 0, **{name: 0 for name in names}})
        m["responses"] += 1
        for name in names:
            m[name] += r[name]
    for r in responses.values():
        if r["session"] in stream_only:
            models.setdefault(r["model"] or "unknown", {"responses": 0, **{name: 0 for name in names}})["responses"] += 1
    cost = None
    for usages in stream_only.values():
        for usage in usages:
            for model, u in usage.items():
                if not isinstance(u, dict):
                    continue
                m = models.setdefault(model, {"responses": 0, **{name: 0 for name in names}})
                for name in names:
                    value = number(u.get(CLI_FIELDS[name]))
                    m[name] += value
                    totals[name] += value
                stated = u.get("costUSD")
                cost = (cost or 0) + (float(stated) if isinstance(stated, (int, float)) and not isinstance(stated, bool) and stated >= 0 else 0)
    reported = None
    if cli:
        reported = {name: sum(number(u.get(field)) for usage in cli.values() for u in usage.values() if isinstance(u, dict))
                    for name, field in CLI_FIELDS.items()}
    return {"files": files, "responses": len(responses), "subagentResponses": sum(r["subagent"] for r in responses.values()),
            **totals, "byModel": models, "unreadableLines": unreadable, "sessionTotals": reported,
            "streamResults": sum(len(u) for u in stream_only.values()), "costUsd": None if cost is None else round(cost, 6)}


def main(argv=None):
    argv = list(sys.argv[1:] if argv is None else argv)
    if len(argv) != 1:
        print(__doc__, file=sys.stderr)
        return 2
    print(json.dumps(count(argv[0]), indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
