"""The one remaining identity start, explicitly reauthorized after config failure."""
from contextlib import closing
import argparse
import hashlib
import json
import os
from pathlib import Path
import secrets
import sqlite3
import time

from identity_probe import digest, events, summarize
from process import bounded
import runner

ROOT = Path(__file__).resolve().parents[1]
AUTHORIZATION = {
    "maxTotalExternalStarts": 2, "previousStartsConsumed": 1, "remainingStarts": 1,
    "sessionWallSeconds": 180, "maxParallelActors": 1, "wrapperAgentTurns": 1,
    "wrapperRetries": 0, "wrapperContinuation": False, "subagents": False,
    "transportRetries": "unchanged builtin defaults; observed internal count unknown unless receipts expose it",
    "documentedDefaults": {"requestMaxRetries": 4, "streamMaxRetries": 5},
    "documentedDefaultScope": "Official current config reference; exact applied binary values not independently exposed",
    "retrospectiveKnownTokenThreshold": 10000, "oldUnknownUsagePreserved": True,
    "scope": "Explicit remaining tiny diagnostic start only; later common study limits unchanged",
}


def corrected_config():
    config = {key: value for key, value in runner.CONFIG.items() if not key.startswith("model_providers.")}
    return {**config, "approval_policy": "never", "forced_login_method": "chatgpt", "model_provider": "openai",
            "features.shell_tool": False, "features.unified_exec": False, "apps._default.enabled": False}


def reserve_last(database, previous, request_sha, actor_root):
    Path(database).parent.mkdir(parents=True, exist_ok=True)
    with closing(sqlite3.connect(database, isolation_level=None, timeout=10)) as db:
        try:
            db.execute("BEGIN IMMEDIATE")
            db.execute("CREATE TABLE IF NOT EXISTS attempts(number INTEGER PRIMARY KEY,start REAL,end REAL,result TEXT,request_sha TEXT,root TEXT)")
            rows = db.execute("SELECT number,end FROM attempts").fetchall()
            if not rows:
                if previous["number"] != 1 or previous["status"] != "failed" or previous["usage"]["agentTurnsStarted"] != 0:
                    raise ValueError("remaining-start grant only follows the preserved pre-turn config failure")
                db.execute("INSERT INTO attempts VALUES(1,NULL,0,?,?,'historical immutable evidence')",
                           (json.dumps(previous), previous["requestSha256"]))
            rows = db.execute("SELECT number,end FROM attempts").fetchall()
            if rows != [(1, 0.0)]:
                raise ValueError("two-start authorization exhausted or another start active; no refill")
            db.execute("INSERT INTO attempts VALUES(2,?,NULL,NULL,?,?)", (time.time(), request_sha, str(actor_root)))
            db.commit()
        except BaseException:
            db.rollback()
            raise


def run(root, executable):
    root = Path(root).resolve(strict=True)
    checkout = ROOT.parents[1]
    if root == checkout or checkout in root.parents or root in checkout.parents:
        raise ValueError("independent external probe root required")
    previous_dir = ROOT / "evidence/actual-runner-probe-r1"
    previous = json.loads((previous_dir / "result-1.json").read_bytes())
    if previous["requestSha256"] != digest(previous_dir / "request-1.json"):
        raise ValueError("historical failed-attempt binding changed")
    if "reserved built-in provider IDs" not in (previous_dir / "session-1/stderr.log").read_text(encoding="utf-8"):
        raise ValueError("historical native rejection differs from authorized prerequisite")
    preflight = json.loads((root / "preflight-corrected-config/process.json").read_bytes())
    if preflight["returnCode"] != 0 or preflight["stopReason"]:
        raise ValueError("corrected config did not pass non-agent native parsing")
    actor = root / "actor-2"
    actor.mkdir(exist_ok=False)
    sentinel = secrets.token_hex(16)
    (actor / "PUBLIC-SENTINEL.txt").write_text(sentinel + "\n", encoding="utf-8")
    prompt = "Use no tools, subagents, or file access. This is a public identity/usage smoke, no project work. Reply exactly with this sentinel and nothing else:\n" + sentinel + "\n"
    (root / "prompt-2.txt").write_text(prompt, encoding="utf-8")
    pin, config = runner.inspect(executable), corrected_config()
    argv = [pin["path"], "exec", "--ignore-user-config", "--ignore-rules", "--model", runner.MODEL,
            "--sandbox", "read-only", "--ephemeral", "--json", "--skip-git-repo-check"]
    for key, value in sorted(config.items()):
        argv += ["--config", key + "=" + json.dumps(value)]
    argv += ["--cd", str(actor), "-"]
    request = {"number": 2, "authorization": AUTHORIZATION, "argv": argv, "cwd": str(actor), "config": config,
               "runnerPath": pin["path"], "runnerSha256": pin["sha256"], "runnerVersion": pin["version"],
               "requestedModel": runner.MODEL, "requestedReasoning": runner.REASONING,
               "configSha256": hashlib.sha256(json.dumps(config, sort_keys=True, separators=(",", ":")).encode()).hexdigest(),
               "promptSha256": digest(root / "prompt-2.txt"), "sentinelSha256": digest(actor / "PUBLIC-SENTINEL.txt"),
               "priorRequestSha256": previous["requestSha256"], "oldUsage": previous["usage"],
               "processReceiptAutomaticRetriesScope": "wrapper process launches only; never internal transport retries",
               "runtimeSourceDigests": {p.name: digest(p) for p in (Path(__file__), Path(__file__).with_name("identity_probe.py"), Path(__file__).with_name("process.py"), Path(__file__).with_name("runner.py"))}}
    request_path = root / "request-2.json"
    request_path.write_text(json.dumps(request, indent=2) + "\n", encoding="utf-8")
    # This fixed operator-owned path prevents changing the external root to refill quota.
    database = ROOT / ".study-data/actual-runner-authorized-starts.sqlite"
    reserve_last(database, previous, digest(request_path), actor)
    result = {"number": 2, "requestSha256": digest(request_path), "status": "failed", "usage": summarize([]),
              "resolvedModel": None, "liveTrials": 0, "effectiveAccessProbe": "not authorized in this final diagnostic",
              "wrapperRetries": 0, "transportRetriesObserved": None, "oldUsageUnknown": True,
              "allAttemptsTokenTotal": None, "knownObservedTokenSubtotal": None}
    def monitor(directory):
        values = events(directory / "stdout.log")
        usage = summarize(values)
        if usage["agentTurnsStarted"] > 1:
            return "unexpected_second_agent_turn"
        if any(event.get("type") in {"item.started", "item.completed"} and event.get("item", {}).get("type")
               not in {"agent_message", "reasoning"} for event in values):
            return "unexpected_tool_activity"
        if (usage["reportedInputPlusOutputTokens"] or 0) >= 10000:
            return "retrospective_known_token_threshold"
        return None
    env = {key: value for key, value in os.environ.items() if key not in ("OPENAI_API_KEY", "CODEX_API_KEY")}
    try:
        receipt = bounded(argv, str(actor), root / "session-2", 180, stdin=prompt.encode(), env=env,
                          stop_path=root / "STOP", poll_stop=monitor)
        values = events(root / "session-2/stdout.log")
        result["process"] = receipt
        result["usage"] = summarize(values)
        result["knownObservedTokenSubtotal"] = result["usage"]["reportedInputPlusOutputTokens"]
        result["status"] = "stopped" if receipt["stopReason"] else ("completed" if receipt["returnCode"] == 0 and result["usage"]["agentTurnsCompleted"] == 1 else "failed")
        messages = [event.get("item", {}).get("text") for event in values if event.get("type") == "item.completed" and event.get("item", {}).get("type") == "agent_message"]
        result["sentinelEchoMatched"] = messages == [sentinel]
    except BaseException as exc:
        result["error"] = type(exc).__name__ + ": " + str(exc)
        raise
    finally:
        (root / "result-2.json").write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
        with closing(sqlite3.connect(database)) as db:
            with db:
                db.execute("UPDATE attempts SET end=?,result=? WHERE number=2", (time.time(), json.dumps(result)))
    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", required=True)
    parser.add_argument("--runner", required=True)
    args = parser.parse_args()
    result = run(args.root, args.runner)
    print(json.dumps({key: result.get(key) for key in ("status", "resolvedModel", "sentinelEchoMatched", "usage")}))
