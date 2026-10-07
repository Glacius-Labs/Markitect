"""Read-only cumulative public diagnostic accounting; unknowns never become zero."""
import json
from pathlib import Path
import sqlite3

from dispatch import digest


def read_history():
    root = Path(__file__).parents[1] / ".study-data"
    paths = [root / "actual-runner-authorized-starts.sqlite", root / "runner-01601-diagnostic.sqlite"]
    before = {str(path): digest(path.read_bytes()) for path in paths}
    attempts = []
    for index, path in enumerate(paths):
        db = sqlite3.connect(path.resolve().as_uri() + "?mode=ro", uri=True)
        try:
            if index == 0:
                rows = db.execute("SELECT number,start,end,request_sha,result,NULL FROM attempts ORDER BY number").fetchall()
            else:
                rows = db.execute("SELECT id,start,end,request_sha,result,history FROM grants ORDER BY start").fetchall()
            for identity, start, end, request_sha, raw, old_history in rows:
                result = json.loads(raw) if raw else {}
                usage = result.get("usage") or {}
                attempts.append({"grant": "historical-two-start" if index == 0 else "selected-runner-one-start",
                                 "id": identity, "finished": end is not None, "status": result.get("status"),
                                 "tokens": usage.get("reportedInputPlusOutputTokens"), "providerRequests": usage.get("providerRequests"),
                                 "startedAt": start, "endedAt": end, "requestSha256": request_sha,
                                 "resultSha256": digest(raw.encode()) if raw else None,
                                 "runnerArgv": result.get("process", {}).get("argv"),
                                 "originalGrantMaxStarts": 2 if index == 0 else 1, "originalGrantRemainingStarts": 0,
                                 "originalGrantHistorySha256": digest(old_history.encode()) if old_history else None})
        finally:
            db.close()
    after = {str(path): digest(path.read_bytes()) for path in paths}
    if before != after or len(attempts) != 3 or not all(a["finished"] for a in attempts):
        raise ValueError("immutable exhausted diagnostic history differs")
    known = [a["tokens"] for a in attempts if type(a["tokens"]) is int]
    return {"actorStartsConsumed": len(attempts), "historicalStartsRemaining": 0, "attempts": attempts,
            "allHistoryTokens": sum(known) if len(known) == len(attempts) else None,
            "knownTokenSubtotal": sum(known), "unknownTokenAttempts": len(attempts) - len(known),
            "allHistoryProviderRequests": None, "internalRetries": None, "ledgerSha256Before": before,
            "ledgerSha256After": after, "historicalLedgersUnchanged": True}


def read_context_history():
    """Append the exhausted fourth start read only; never rewrite its ledger."""
    old = read_history()
    path = (Path.home() / "Documents/Scientist-Probes/context-additional-20261007/dispatch.sqlite").resolve()
    before = digest(path.read_bytes())
    db = sqlite3.connect(path.as_uri() + "?mode=ro", uri=True)
    try:
        bindings = db.execute("SELECT binding FROM context_allocation").fetchall()
        rows = db.execute("SELECT a.id,a.start,a.end,a.status,a.tokens,d.request,d.result,d.phase "
                          "FROM attempts a JOIN dispatches d ON d.attempt=a.id").fetchall()
        attempt_count = db.execute("SELECT COUNT(*) FROM attempts").fetchone()[0]
        dispatch_count = db.execute("SELECT COUNT(*) FROM dispatches").fetchone()[0]
    finally:
        db.close()
    after = digest(path.read_bytes())
    if (before != after or len(bindings) != 1 or json.loads(bindings[0][0])["predecessors"] != old or
            attempt_count != 1 or dispatch_count != 1 or len(rows) != 1):
        raise ValueError("immutable fourth diagnostic history differs")
    identity, start, end, status, tokens, raw_request, raw_result, phase = rows[0]
    result = json.loads(raw_result) if raw_result else {}
    usage = result.get("usage") or {}
    if (end is None or phase != "finished" or status != "stopped" or result.get("status") != status or
            tokens != 20501 or usage.get("reportedInputPlusOutputTokens") != tokens or
            result.get("requestSha256") != digest(raw_request)):
        raise ValueError("exhausted fourth diagnostic result differs")
    attempt = {"grant": "additional-context-one-start", "id": identity, "finished": True,
               "startedAt": start, "endedAt": end, "status": status, "tokens": tokens,
               "providerRequests": usage.get("providerRequests"), "requestSha256": digest(raw_request),
               "resultSha256": digest(raw_result.encode()), "runnerArgv": result.get("process", {}).get("argv"),
               "originalGrantMaxStarts": 1, "originalGrantRemainingStarts": 0,
               "originalGrantHistorySha256": None, "grantSha256": result.get("grantSha256"),
               "protocolSha256": result.get("protocolSha256"), "accountingStopReason": result.get("accountingStopReason")}
    return {**old, "actorStartsConsumed": 4, "attempts": old["attempts"] + [attempt],
            "knownTokenSubtotal": 30510,
            "ledgerSha256Before": {**old["ledgerSha256Before"], str(path): before},
            "ledgerSha256After": {**old["ledgerSha256After"], str(path): after}}
