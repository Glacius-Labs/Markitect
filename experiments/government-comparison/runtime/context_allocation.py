"""One fixed additional diagnostic allocation; study admission is unchanged."""
import hashlib
import json
from pathlib import Path
import sqlite3

from ledger import Ledger, LimitReached
from measurement_profile import OBSERVED

DECISION_PATH = Path(__file__).parents[1] / "public/context-additional-decision.json"
DECISION = json.loads(DECISION_PATH.read_bytes())
ALLOCATION_ID = DECISION["decisionId"]
ALLOCATION_PATH = (Path.home() / "Documents/Scientist-Probes/context-additional-20261007/dispatch.sqlite").resolve()


def allocation_binding():
    from diagnostic_history import read_history
    history = read_history()
    if (list(history["ledgerSha256After"].values()) != DECISION["historicalLedgerSha256"] or
            history["actorStartsConsumed"] != 3 or history["historicalStartsRemaining"] != 0 or
            history["knownTokenSubtotal"] != 10009 or history["allHistoryTokens"] is not None):
        raise ValueError("fixed additional allocation predecessor history differs")
    return {"decisionId": ALLOCATION_ID, "decisionSha256": hashlib.sha256(DECISION_PATH.read_bytes()).hexdigest(),
            "trialId": ALLOCATION_ID, "ledgerPath": str(ALLOCATION_PATH), "predecessors": history,
            "maxNewSessions": 1, "cumulativeSessionCeiling": 4,
            "resourceWindow": "separate one-session diagnostic allocation; historical unknown total remains null"}


def validate_allocation(grant, protocol):
    expected = allocation_binding()
    if (grant.get("contextAllocation") != expected or protocol.get("contextAllocation") != expected or
            grant.get("trialId") != ALLOCATION_ID or grant.get("ledgerPath") != str(ALLOCATION_PATH) or
            grant.get("maxActorSessions") != 1 or grant.get("cumulativeSessionCeiling") != 4 or
            grant.get("maxAdditionalActorSessions") != 1 or len(grant.get("authorizedRequests", [])) != 1 or
            grant.get("maxParallelSessions") != 1 or grant.get("maxSessionWallSeconds") != 180 or
            grant.get("retrospectiveTokenThreshold") != 10000 or
            any(grant.get(key) != DECISION[key] for key in ("wrapperAgentTurns", "wrapperRetries", "children", "continuations", "semanticRepairs", "newPurchases"))):
        raise ValueError("complete fixed additional context allocation binding required")
    return expected


class ContextAllocationLedger(Ledger):
    """The fixed successor file retains predecessors separately from its new window."""
    def __init__(self, path, trial_id, limits, *, binding, profile_id, resume_dispatch_id=None):
        if str(Path(path).resolve()) != str(ALLOCATION_PATH) or trial_id != ALLOCATION_ID or binding != allocation_binding():
            raise ValueError("fixed allocation identity/path/predecessors required")
        if profile_id is None and not Path(path).is_file():
            raise ValueError("allocation resume requires existing ledger; no new file")
        if profile_id is None:
            try:
                db = sqlite3.connect(Path(path).resolve().as_uri() + "?mode=ro", uri=True)
                try:
                    old = db.execute("SELECT id,binding FROM context_allocation").fetchall()
                    record = db.execute("SELECT 1 FROM dispatches WHERE id=?", (resume_dispatch_id,)).fetchone()
                finally:
                    db.close()
                if old != [(ALLOCATION_ID, json.dumps(binding, sort_keys=True))] or not record:
                    raise ValueError("allocation resume requires existing exact dispatch and binding")
            except sqlite3.Error as exc:
                raise ValueError("allocation resume requires existing exact dispatch and binding") from exc
        self.context_binding = json.loads(json.dumps(binding))
        super().__init__(path, trial_id, limits, profile_id=profile_id)
        with self.transaction() as db:
            db.execute("CREATE TABLE IF NOT EXISTS context_allocation(id TEXT PRIMARY KEY, binding TEXT)")
            prior = db.execute("SELECT id,binding FROM context_allocation").fetchall()
            value = json.dumps(binding, sort_keys=True)
            if prior and prior != [(ALLOCATION_ID, value)]:
                raise ValueError("additional allocation history cannot change or refill")
            if not prior:
                if db.execute("SELECT 1 FROM attempts").fetchone():
                    raise ValueError("unbound attempts cannot become a new allocation")
                db.execute("INSERT INTO context_allocation VALUES(?,?)", (ALLOCATION_ID, value))

    def bind_dispatch(self, authority, *, maximum=None, transition=None, adoption=None):
        # The base transaction rechecks authority and adopts the profile atomically.
        # No successor of this one-session allocation is allowed.
        with self.transaction() as db:
            if db.execute("SELECT 1 FROM sqlite_master WHERE type='table' AND name='dispatch_authority'").fetchone():
                prior = db.execute("SELECT value FROM dispatch_authority").fetchone()
                if prior and json.loads(prior[0]) != authority:
                    raise ValueError("fixed additional allocation authority cannot change or refill")
        if maximum != 1 or any(value is not None for value in (transition or {}).values()):
            raise ValueError("exactly one fixed additional allocation; no extension")
        super().bind_dispatch(authority, maximum=maximum, adoption=adoption)

    def _reserve(self, db, task, purpose):
        stored = db.execute("SELECT id,binding FROM context_allocation").fetchall()
        if (stored != [(ALLOCATION_ID, json.dumps(self.context_binding, sort_keys=True))] or
                self.context_binding != allocation_binding()):
            raise ValueError("durable additional allocation predecessors changed")
        if self.profile_id != OBSERVED:
            raise ValueError("additional context allocation requires explicit v2 binding")
        rows = db.execute("SELECT end,tokens FROM attempts").fetchall()
        if any(end is not None and tokens is None for end, tokens in rows):
            raise LimitReached("new context usage tokens unknown; further admission blocked")
        if rows:
            raise LimitReached("additional context allocation exhausted; cumulative four starts maximum")
        if purpose != "context-access":
            raise ValueError("fixed additional allocation permits context-access only")
        return super()._reserve(db, task, purpose)

    def _complete_dispatch(self, db, execution_id, result, requests, tokens):
        result = dict(result, contextAllocation=self.context_binding,
                      cumulativeAccounting={"predecessorStarts": 3,
                          "additionalStarts": db.execute("SELECT COUNT(*) FROM attempts").fetchone()[0],
                          "historicalStartsRemaining": 0, "maximumAllStarts": 4,
                          "allHistoryTokens": None, "allHistoryProviderRequests": None,
                          "knownTokenSubtotal": 10009 + (tokens or 0),
                          "newWindowTokens": tokens, "oldBudgetCompliance": "unknown; not claimed"})
        return super()._complete_dispatch(db, execution_id, result, requests, tokens)

    def snapshot(self):
        value = super().snapshot()
        old = self.context_binding["predecessors"]
        value.update(contextAllocation=self.context_binding, predecessorActorSessions=3,
                     cumulativeActorSessions=3 + value["actorSessions"], cumulativeProviderTokens=None,
                     cumulativeProviderRequests=None, historicalStartsRemaining=0,
                     knownCumulativeTokenSubtotal=old["knownTokenSubtotal"] + sum(a["tokens"] or 0 for a in value["attempts"]))
        return value
