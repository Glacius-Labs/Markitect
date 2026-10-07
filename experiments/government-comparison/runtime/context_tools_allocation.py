"""Fixed draft-only successor for corrected context tools; no expandable quota."""
import hashlib
import json
from pathlib import Path

from context_allocation import ContextAllocationLedger
from diagnostic_history import read_context_history

DECISION_PATH = Path(__file__).parents[1] / "public/context-tools-decision.json"
DECISION = json.loads(DECISION_PATH.read_bytes())
ALLOCATION_ID = DECISION["decisionId"]
ALLOCATION_PATH = (Path.home() / "Documents/Scientist-Probes/context-tools-20261007/dispatch.sqlite").resolve()


def allocation_binding():
    history = read_context_history()
    if (list(history["ledgerSha256After"].values()) != DECISION["historicalLedgerSha256"] or
            history["actorStartsConsumed"] != 4 or history["historicalStartsRemaining"] != 0 or
            history["knownTokenSubtotal"] != 30510 or history["allHistoryTokens"] is not None):
        raise ValueError("fixed corrected-tools predecessor history differs")
    return {"decisionId": ALLOCATION_ID, "decisionSha256": hashlib.sha256(DECISION_PATH.read_bytes()).hexdigest(),
            "trialId": ALLOCATION_ID, "ledgerPath": str(ALLOCATION_PATH), "predecessors": history,
            "maxNewSessions": 1, "cumulativeSessionCeiling": 5,
            "resourceWindow": "one selected draft-only corrected-tools window; exact Run-Grant still required"}


def validate_allocation(grant, protocol):
    expected = allocation_binding()
    if (grant.get("contextAllocation") != expected or protocol.get("contextAllocation") != expected or
            grant.get("trialId") != ALLOCATION_ID or grant.get("ledgerPath") != str(ALLOCATION_PATH) or
            grant.get("maxActorSessions") != 1 or grant.get("maxAdditionalActorSessions") != 1 or
            grant.get("cumulativeSessionCeiling") != 5 or len(grant.get("authorizedRequests", [])) != 1 or
            any(grant.get(key) != DECISION[key] for key in ("maxParallelSessions", "wrapperAgentTurns",
                    "wrapperRetries", "children", "continuations", "semanticRepairs", "newPurchases")) or
            grant.get("maxSessionWallSeconds") != 180 or grant.get("retrospectiveTokenThreshold") != 50000):
        raise ValueError("complete fixed corrected-tools allocation binding required")
    return expected


class ContextToolsAllocationLedger(ContextAllocationLedger):
    expected_binding = staticmethod(allocation_binding)
