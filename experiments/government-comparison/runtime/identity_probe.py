"""Explicit Overseer probe addendum: at most two one-turn sessions, no retry."""
from __future__ import annotations
import argparse
from contextlib import closing
import hashlib
import json
import os
from pathlib import Path
import secrets
import sqlite3
import time

from process import bounded
import runner

AUTHORIZATION = {
    "maxSessions": 2, "maxParallelSessions": 1, "sessionWallSeconds": 180,
    "wrapperAgentTurnsPerSession": 1, "automaticRetries": 0, "subagents": False,
    "retrospectiveTokenStopThreshold": 10000, "hardProviderRequestLimit": None,
    "scope": "Only this actual public-sentinel readiness probe; later study commonLimits unchanged",
}


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def events(path):
    if not path.exists():
        return []
    values = []
    for line in path.read_bytes().splitlines():
        try:
            values.append(json.loads(line))
        except (ValueError, UnicodeDecodeError):
            pass  # Partial or non-JSON lines remain in unchanged raw output.
    return values


def summarize(values):
    completed = [event for event in values if event.get("type") == "turn.completed"]
    usage = [event.get("usage") for event in completed]
    total = None
    if usage and all(isinstance(item, dict) and all(type(item.get(key)) is int and item[key] >= 0
                         for key in ("input_tokens", "output_tokens")) for item in usage):
        total = sum(item["input_tokens"] + item["output_tokens"] for item in usage)
    return {"agentTurnsStarted": sum(item.get("type") == "turn.started" for item in values),
            "agentTurnsCompleted": len(completed), "rawUsage": usage,
            "reportedInputPlusOutputTokens": total, "providerRequests": None,
            "resolvedProviderModel": None,
            "identitySemantics": "Explicit requested CLI model/config only unless raw runner/provider metadata exposes resolved identity; never use actor self-report",
            "usageSemantics": "exec agent-turn usage input_tokens + output_tokens; cached/reasoning counters retained raw and not added; retrospective, not a hard token ceiling"}


def reserve(database, number):
    db = sqlite3.connect(database, isolation_level=None, timeout=10)
    try:
        db.execute("BEGIN IMMEDIATE")
        db.execute("CREATE TABLE IF NOT EXISTS attempts(number INTEGER PRIMARY KEY, start REAL, end REAL, result TEXT)")
        rows = db.execute("SELECT number,end,result FROM attempts ORDER BY number").fetchall()
        if len(rows) >= 2 or any(row[1] is None for row in rows) or any(row[0] == number for row in rows):
            raise ValueError("probe session/parallel/no-repeat boundary reached")
        if number != len(rows) + 1:
            raise ValueError("probe order cannot change")
        if rows:
            previous = json.loads(rows[-1][2])
            tokens = previous["usage"]["reportedInputPlusOutputTokens"]
            if previous["status"] != "completed" or tokens is None or tokens >= 10000:
                raise ValueError("second probe requires executable first probe and known remaining observed token budget")
        db.execute("INSERT INTO attempts VALUES(?,?,NULL,NULL)", (number, time.time()))
        db.commit()
    except BaseException:
        db.rollback()
        raise
    finally:
        db.close()


def run(root, executable, number=1):
    root = Path(root).resolve()
    checkout = Path(__file__).resolve().parents[3]
    if root == checkout or checkout in root.parents or root in checkout.parents:
        raise ValueError("probe root must be outside study checkout hierarchy")
    root.mkdir(parents=True, exist_ok=True)
    pin = runner.inspect(executable)
    if number != 1:
        raise ValueError("Probe 2 requires a separately reviewed public access plan after executable Probe 1")
    actor = root / "actor-1"
    actor.mkdir(exist_ok=False)
    sentinel = secrets.token_hex(16)
    (actor / "PUBLIC-SENTINEL.txt").write_text(sentinel + "\n", encoding="utf-8")
    prompt = ("This is a public identity/usage smoke, not project work. Use no tools or subagents. "
              "Do not read files. Reply exactly with the following public random sentinel and no other text:\n" + sentinel + "\n")
    (root / "prompt-1.txt").write_text(prompt, encoding="utf-8")
    config = {**runner.CONFIG, "approval_policy": "never", "forced_login_method": "chatgpt",
              "features.shell_tool": False, "features.unified_exec": False,
              "apps._default.enabled": False}
    argv = [pin["path"], "exec", "--ignore-user-config", "--ignore-rules", "--model", runner.MODEL,
            "--sandbox", "read-only", "--ephemeral", "--json", "--skip-git-repo-check"]
    for key, value in sorted(config.items()):
        argv += ["--config", key + "=" + json.dumps(value)]
    argv += ["--cd", str(actor), "-"]
    # Preserve saved account auth; do not allow inherited API-key fallback.
    env = {key: value for key, value in os.environ.items() if key not in ("OPENAI_API_KEY", "CODEX_API_KEY")}
    request = {"authorization": AUTHORIZATION, "number": number, "argv": argv, "cwd": str(actor),
               "runner": pin, "effectiveConfigRequested": config,
               "configSha256": hashlib.sha256(json.dumps(config, sort_keys=True, separators=(",", ":")).encode()).hexdigest(),
               "promptSha256": digest(root / "prompt-1.txt"), "publicSentinelSha256": digest(actor / "PUBLIC-SENTINEL.txt"),
               "authPolicy": "existing saved ChatGPT account auth; API-key environment variables excluded without reading or logging values",
               "runtimeSourceDigests": {p.name: digest(p) for p in (Path(__file__), Path(__file__).with_name("process.py"), Path(__file__).with_name("runner.py"))}}
    request_path = root / "request-1.json"
    request_path.write_text(json.dumps(request, indent=2) + "\n", encoding="utf-8")
    def monitor(directory):
        values = events(directory / "stdout.log")
        observed = summarize(values)
        if observed["agentTurnsStarted"] > 1:
            return "unexpected_second_agent_turn"
        if any(item.get("type") in {"item.started", "item.completed"} and item.get("item", {}).get("type")
               not in {"agent_message", "reasoning"} for item in values):
            return "unexpected_tool_activity"
        if (observed["reportedInputPlusOutputTokens"] or 0) >= 10000:
            return "retrospective_token_threshold"
        return None
    reserve(root / "probe-budget.sqlite", number)  # Failed native start consumes the reserved attempt.
    result = {"requestSha256": digest(request_path), "number": number, "status": "failed", "usage": summarize([]),
              "requestedModel": runner.MODEL, "requestedReasoning": runner.REASONING, "resolvedModel": None,
              "liveTrials": 0, "effectiveAccessProbe": "not executed"}
    try:
        process = bounded(argv, str(actor), root / "session-1", 180, stdin=prompt.encode(), env=env,
                          stop_path=root / "STOP", poll_stop=monitor)
        values = events(root / "session-1/stdout.log")
        result["usage"] = summarize(values)
        result["process"] = process
        result["status"] = "stopped" if process["stopReason"] else ("completed" if process["returnCode"] == 0 and result["usage"]["agentTurnsCompleted"] == 1 else "failed")
        messages = [event.get("item", {}).get("text") for event in values if event.get("type") == "item.completed" and event.get("item", {}).get("type") == "agent_message"]
        result["sentinelEchoMatched"] = messages == [sentinel]
    except BaseException as exc:
        result["error"] = type(exc).__name__ + ": " + str(exc)
        raise
    finally:
        (root / "result-1.json").write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
        with closing(sqlite3.connect(root / "probe-budget.sqlite")) as db:
            with db:
                db.execute("UPDATE attempts SET end=?,result=? WHERE number=?", (time.time(), json.dumps(result), number))
    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", required=True)
    parser.add_argument("--runner", required=True)
    args = parser.parse_args()
    result = run(args.root, args.runner)
    print(json.dumps({key: result[key] for key in ("status", "requestedModel", "resolvedModel", "sentinelEchoMatched", "usage") if key in result}))
