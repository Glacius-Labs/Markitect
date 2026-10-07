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
                rows = db.execute("SELECT number,end,result FROM attempts ORDER BY number").fetchall()
            else:
                rows = db.execute("SELECT id,end,result FROM grants ORDER BY start").fetchall()
            for identity, end, raw in rows:
                result = json.loads(raw) if raw else {}
                usage = result.get("usage") or {}
                attempts.append({"grant": "historical-two-start" if index == 0 else "selected-runner-one-start",
                                 "id": identity, "finished": end is not None, "status": result.get("status"),
                                 "tokens": usage.get("reportedInputPlusOutputTokens"), "providerRequests": usage.get("providerRequests")})
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
            "ledgerSha256After": after, "historicalLedgersUnchanged": True,
            "newContextSession": "separate proposed allocation of one; not approved, not a refill or reset"}
