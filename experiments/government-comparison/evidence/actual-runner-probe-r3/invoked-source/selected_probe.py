"""Separate one-start grant for the selected 0.160.1 runner; study stays closed."""
import argparse
from contextlib import closing
import hashlib
import json
import os
from pathlib import Path
import secrets
import sqlite3
import subprocess
import sys
import time

from identity_probe import digest, events, summarize
from process import bounded
import runner

ROOT = Path(__file__).resolve().parents[1]
EXE = "C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/5ea220ae823df3d7/codex.exe"
GRANT = "overseer-2026-10-07-1826-runner-01601"
AUTHORIZATION = {
    "grantId": GRANT, "newExternalStarts": 1, "historicalStartsConsumed": 2,
    "diagnosticHistoryMaximum": 3, "wallSeconds": 180, "parallel": 1,
    "wrapperTurns": 1, "wrapperRetries": 0, "continuations": False, "children": False,
    "purpose": "Public sentinel identity/usage only; no study or access probe",
    "transportRetries": "Unchanged built-in defaults permitted; observed internal count unknown unless exposed",
    "knownNewTokenThreshold": 10000, "thresholdIsHardCap": False,
    "oldUnknownUsageRemainsUnknown": True,
    "toolBoundary": "Read-only/approval never; unified exec may remain active; unexpected tools stopped reactively",
}


def write(path, value):
    path.write_text(json.dumps(value, indent=2) + "\n", encoding="utf-8")


def config():
    return {**runner.CONFIG, "approval_policy": "never", "forced_login_method": "chatgpt"}


def overrides(settings):
    return [part for key, value in sorted(settings.items()) for part in ("-c", key + "=" + json.dumps(value))]


def environment():
    return {key: value for key, value in os.environ.items() if key not in ("OPENAI_API_KEY", "CODEX_API_KEY")}


def external_root(value):
    root = Path(value).resolve(strict=True)
    checkout = ROOT.parents[1]
    if root == checkout or checkout in root.parents or root in checkout.parents:
        raise ValueError("independent external public root required")
    return root


def evaluate_metadata(values):
    replies = {value.get("id"): value for value in values if "id" in value}
    account = replies.get(1, {}).get("result", {}).get("account")
    errors = [value["error"] for value in replies.values() if "error" in value]
    records = [record for key, value in replies.items() if isinstance(key, int) and key >= 2
               for record in value.get("result", {}).get("data", [])]
    matches = [record for record in records if record.get("model") == runner.MODEL]
    high = any(item.get("reasoningEffort") == "high" for record in matches
               for item in record.get("supportedReasoningEfforts", []))
    denied = account is None and 1 in replies and "result" in replies[1]
    denied |= any(term in json.dumps(errors).lower() for term in ("unauthorized", "forbidden", "not supported", '"code": 401', '"code": 403'))
    complete = any(isinstance(key, int) and key >= 2 and "result" in value
                   and value["result"].get("nextCursor") is None for key, value in replies.items())
    if complete and not errors and (not matches or not high):
        denied = True  # Current advertised list cannot support the exact requested profile.
    return {"status": "rejected" if denied else ("advertised" if matches and high and account else "unknown"),
            "accountType": account.get("type") if isinstance(account, dict) else None,
            "targetEntries": matches, "requestedHighAdvertised": high, "rpcErrors": errors,
            "scope": "Normal authenticated client metadata; catalog may be cached, not proof of inference or resolved serving model"}


def prepare(root):
    root = external_root(root)
    pin = runner.inspect(EXE)
    write(root / "authorization.json", AUTHORIZATION)
    write(root / "selected-runner.json", pin)
    empty = root / "config-validation-home"
    empty.mkdir(exist_ok=False)
    env = environment()
    version = bounded([EXE, "--version"], str(root), root / "version", 30, env=env)
    if version["returnCode"] != 0 or version["stopReason"] or (root / "version/stdout.log").read_text().strip() != runner.EXPECTED_VERSION:
        raise ValueError("selected native version differs from pin")
    config_env = {**env, "CODEX_HOME": str(empty)}
    receipt = bounded([EXE, "features", "list", *overrides({**config(), "model": runner.MODEL})],
                      str(root), root / "features", 30, env=config_env)
    if receipt["returnCode"] != 0 or receipt["stopReason"]:
        raise ValueError("selected local configuration rejected")
    request = {"argv": [EXE, "app-server", "--stdio", *overrides({**config(), "model": runner.MODEL})],
               "initialize": {"method": "initialize", "id": 0, "params": {
                   "clientInfo": {"name": "scientist_readiness", "version": "1.0"}}},
               "allowedRpc": ["initialize", "initialized", "account/read(refreshToken=false)", "model/list"],
               "auth": "Existing normal client sign-in; no credentials extracted or copied", "actorStarts": 0}
    request_path = root / "metadata-request.json"
    write(request_path, request)
    receipt = bounded([sys.executable, str(Path(__file__).with_name("selected_metadata.py")), str(request_path)],
                      str(root), root / "metadata", 45, env=env, stop_path=root / "STOP")
    outcome = evaluate_metadata(events(root / "metadata/stdout.log"))
    outcome.update(processReturnCode=receipt["returnCode"], processStopReason=receipt["stopReason"],
                   requestSha256=digest(request_path), actorStarts=0)
    write(root / "metadata-result.json", outcome)
    return outcome


def reserve(database, historical, request_hash):
    if [row["number"] for row in historical] != [1, 2] or any(row["status"] != "failed" for row in historical):
        raise ValueError("preserved two-start history prerequisite differs")
    Path(database).parent.mkdir(parents=True, exist_ok=True)
    with closing(sqlite3.connect(database, isolation_level=None, timeout=10)) as db:
        try:
            db.execute("BEGIN IMMEDIATE")
            db.execute("CREATE TABLE IF NOT EXISTS grants(id TEXT PRIMARY KEY,start REAL,end REAL,request_sha TEXT,history TEXT,result TEXT)")
            if db.execute("SELECT COUNT(*) FROM grants").fetchone()[0]:
                raise ValueError("separate one-start grant exhausted or active; no refill")
            db.execute("INSERT INTO grants VALUES(?,?,NULL,?,?,NULL)", (GRANT, time.time(), request_hash, json.dumps(historical)))
            db.commit()
        except BaseException:
            db.rollback()
            raise


def stop_reason(values):
    usage = summarize(values)
    if usage["agentTurnsStarted"] > 1:
        return "unexpected_second_agent_turn"
    if any(value.get("type") in {"item.started", "item.completed"} and value.get("item", {}).get("type")
           not in {"agent_message", "reasoning"} for value in values):
        return "unexpected_tool_activity"
    if (usage["reportedInputPlusOutputTokens"] or 0) >= 10000:
        return "retrospective_new_token_threshold"
    return None


def require_advertised(metadata):
    exact_high = any(record.get("model") == runner.MODEL and any(
        effort.get("reasoningEffort") == "high" for effort in record.get("supportedReasoningEfforts", []))
        for record in metadata.get("targetEntries", []))
    if metadata.get("status") != "advertised" or metadata.get("requestedHighAdvertised") is not True or not exact_high:
        raise ValueError("current metadata does not affirm exact target/high; no inference")


def run(root):
    root = external_root(root)
    metadata = json.loads((root / "metadata-result.json").read_bytes())
    if metadata["status"] == "rejected":
        raise ValueError("current account/model metadata rejected requested profile; no inference")
    require_advertised(metadata)
    if metadata["processReturnCode"] != 0 or metadata["processStopReason"]:
        raise ValueError("metadata subprocess incomplete; no inference")
    if metadata["accountType"] not in {"chatgpt", "chatgptAuthTokens"}:
        raise ValueError("normal existing ChatGPT auth not established; no inference")
    if metadata["requestSha256"] != digest(root / "metadata-request.json"):
        raise ValueError("metadata request binding changed")
    metadata_request = json.loads((root / "metadata-request.json").read_bytes())
    if metadata_request["argv"] != [EXE, "app-server", "--stdio", *overrides({**config(), "model": runner.MODEL})]:
        raise ValueError("prepared configuration differs from selected candidate")
    historical = [json.loads((ROOT / f"evidence/actual-runner-probe-r{number}/result-{number}.json").read_bytes()) for number in (1, 2)]
    for number, result in enumerate(historical, 1):
        if result["requestSha256"] != digest(ROOT / f"evidence/actual-runner-probe-r{number}/request-{number}.json"):
            raise ValueError("historical request binding changed")
    pin = runner.inspect(EXE)
    source_candidate = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT.parents[1], text=True).strip()
    source_digests = {name: digest(Path(__file__).with_name(name)) for name in
                      ("selected_probe.py", "selected_metadata.py", "identity_probe.py", "process.py", "runner.py")}
    for name, expected in source_digests.items():
        committed = subprocess.check_output(["git", "show", source_candidate + ":experiments/government-comparison/runtime/" + name], cwd=ROOT.parents[1])
        if hashlib.sha256(committed).hexdigest() != expected:
            raise ValueError("invoked runtime must match committed source candidate")
    actor = root / "actor-3"
    actor.mkdir(exist_ok=False)
    sentinel = secrets.token_hex(16)
    (actor / "PUBLIC-SENTINEL.txt").write_text(sentinel + "\n", encoding="utf-8")
    prompt = "Use no tools, subagents, commands, or files. Public identity/usage sentinel only, no project work. Reply exactly with this sentinel and nothing else:\n" + sentinel + "\n"
    (root / "prompt-3.txt").write_text(prompt, encoding="utf-8")
    argv = [EXE, "exec", "--ignore-user-config", "--ignore-rules", "--model", runner.MODEL,
            "--sandbox", "read-only", "--ephemeral", "--json", "--skip-git-repo-check",
            *overrides(config()), "--cd", str(actor), "-"]
    old_ledger = ROOT / ".study-data/actual-runner-authorized-starts.sqlite"
    request = {"number": 3, "authorization": AUTHORIZATION, "runner": pin, "argv": argv, "config": config(),
               "promptSha256": digest(root / "prompt-3.txt"), "sentinelSha256": digest(actor / "PUBLIC-SENTINEL.txt"),
               "metadataResultSha256": digest(root / "metadata-result.json"),
               "historicalRequestHashes": [row["requestSha256"] for row in historical],
               "oldLedgerSha256": digest(old_ledger), "historicalTokens": None,
               "sourceCandidate": source_candidate, "runtimeSourceDigests": source_digests}
    write(root / "request-3.json", request)
    database = ROOT / ".study-data/runner-01601-diagnostic.sqlite"
    reserve(database, historical, digest(root / "request-3.json"))
    result = {"number": 3, "status": "failed", "requestSha256": digest(root / "request-3.json"),
              "usage": summarize([]), "resolvedModel": None, "providerRequests": None,
              "internalRetriesObserved": None, "wrapperRetries": 0, "allHistoryTokenTotal": None,
              "oldUsageUnknown": True, "liveTrials": 0, "toolSuppression": "not established"}
    try:
        receipt = bounded(argv, str(actor), root / "session-3", 180, stdin=prompt.encode(), env=environment(),
                          stop_path=root / "STOP", poll_stop=lambda directory: stop_reason(events(directory / "stdout.log")))
        values = events(root / "session-3/stdout.log")
        result.update(process=receipt, usage=summarize(values), observedEventStopReason=stop_reason(values))
        result["status"] = "stopped" if receipt["stopReason"] or stop_reason(values) else (
            "completed" if receipt["returnCode"] == 0 and result["usage"]["agentTurnsCompleted"] == 1 else "failed")
        messages = [value.get("item", {}).get("text") for value in values if value.get("type") == "item.completed"
                    and value.get("item", {}).get("type") == "agent_message"]
        result["sentinelEchoMatched"] = messages == [sentinel]
        result["oldLedgerUnchanged"] = digest(old_ledger) == request["oldLedgerSha256"]
    except BaseException as exc:
        result["error"] = type(exc).__name__ + ": " + str(exc)
        raise
    finally:
        write(root / "result-3.json", result)
        with closing(sqlite3.connect(database)) as db:
            with db:
                db.execute("UPDATE grants SET end=?,result=? WHERE id=?", (time.time(), json.dumps(result), GRANT))
    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("phase", choices=("prepare", "actor"))
    parser.add_argument("--root", required=True)
    args = parser.parse_args()
    result = prepare(args.root) if args.phase == "prepare" else run(args.root)
    print(json.dumps({key: result.get(key) for key in ("status", "accountType", "requestedHighAdvertised", "sentinelEchoMatched", "usage")}))
