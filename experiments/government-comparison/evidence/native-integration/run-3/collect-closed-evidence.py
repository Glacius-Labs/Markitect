"""Archive closed R3 evidence and reconcile additive counters; no launches."""
import hashlib
import importlib.util
import json
from pathlib import Path
import sqlite3
import sys

ROOT = Path(__file__).resolve().parents[3]
spec = importlib.util.spec_from_file_location("r3_driver", ROOT / "run-native-integration-r3.py")
d = importlib.util.module_from_spec(spec)
spec.loader.exec_module(d)
from native_fixture_budget import R3_HISTORY_PATH, R3_LABELS, _history_rows


def read_table(path, table):
    db = sqlite3.connect(path.resolve().as_uri() + "?mode=ro", uri=True)
    db.row_factory = sqlite3.Row
    try:
        return [dict(row) for row in db.execute('SELECT * FROM "' + table + '"')]
    finally:
        db.close()


native_path = d.LEGACY / "native-starts.sqlite"
historical_allocation, historical_rows, historical_corrections = _history_rows(R3_HISTORY_PATH)
allocation, rows, corrections = _history_rows(native_path)
prior_keys = {(row[0], row[1]) for row in historical_rows}
assert allocation == historical_allocation
assert [row for row in rows if (row[0], row[1]) in prior_keys] == historical_rows
assert [row for row in corrections if row[0] != d.R3_KEY] == historical_corrections
assert all(row[4] is not None for row in rows)
arms = {}
for arm in ("government", "classic"):
    base = d.EXTERNAL / arm
    ledger = base / "fixture-ledger.sqlite"
    controllers = read_table(ledger, "controller_runs")
    assert len(controllers) == 1 and controllers[0]["end"] is not None
    processes = read_table(ledger, "controller_processes")
    attempts = read_table(ledger, "attempts")
    calls = read_table(ledger, "government_role_calls")
    wrappers = read_table(base / "wrapper-diagnostics/starts.sqlite", "starts")
    delegates = sorted((base / "role-evidence").glob("*/process.json"))
    new = [row for row in rows if row[0] == arm and row[1] in R3_LABELS[arm]]
    assert len(delegates) == len(attempts) == len(calls)
    assert len(wrappers) <= 6 and len(delegates) <= 6
    assert len(new) <= (2 if arm == "government" else 5)
    arms[arm] = {
        "newNativeStarts": len(new), "newReservedSessionSeconds": sum(row[5] for row in new),
        "newWrapperAttempts": len(wrappers), "newDeterministicDelegates": len(delegates),
        "nativeLabels": [row[1] for row in new], "roleSlots": [row["slot_id"] for row in calls],
        "roleStatuses": [row["status"] for row in calls],
        "controllerStatus": controllers[0]["status"],
        "controllerElapsedSeconds": controllers[0]["end"] - controllers[0]["start"],
        "controllerProcessRows": len(processes),
        "controllerActions": [row["action"] for row in processes],
        "outerControllerProcessReceipt": json.loads(controllers[0]["process_receipt"]),
        "delegateProcessReceipts": [json.loads(path.read_bytes()) for path in delegates]}

assert len(rows) <= 12 and sum(row[5] for row in rows) <= 1800
history = json.loads((d.EXTERNAL / "history.json").read_bytes())
for name in ("priorGovernmentLedger", "priorClassicLedger"):
    assert d.binding(history[name]["path"]) == history[name]

archived = []
for path in sorted(d.EXTERNAL.rglob("*")):
    relative = path.relative_to(d.EXTERNAL)
    if not path.is_file() or ".git" in relative.parts:
        continue
    assert not path.is_symlink()
    before = d.binding(path)
    saved = d.write_new(d.EVIDENCE / "external-snapshots/r3" / relative, path.read_bytes())
    assert d.binding(path) == before and saved["sha256"] == before["sha256"]
    archived.append({"original": before, "snapshot": saved})
budget_before_copy = d.binding(native_path)
saved = d.write_new(d.EVIDENCE / "external-snapshots/native-starts.sqlite", native_path.read_bytes())
assert d.binding(native_path) == budget_before_copy and saved["sha256"] == budget_before_copy["sha256"]
archived.append({"original": budget_before_copy, "snapshot": saved})
d.write_new(d.EVIDENCE / "external-snapshot-manifest.json", archived)
summary = {
    "grantKey": d.R3_KEY, "amendmentKey": "r3-a1", "arms": arms,
    "historicalConsumed": {"nativeStarts": 5, "wrapperAttempts": 3, "delegates": 0, "reservedSessionSeconds": 750},
    "newConsumed": {"nativeStarts": sum(a["newNativeStarts"] for a in arms.values()),
                    "wrapperAttempts": sum(a["newWrapperAttempts"] for a in arms.values()),
                    "delegates": sum(a["newDeterministicDelegates"] for a in arms.values()),
                    "reservedSessionSeconds": sum(a["newReservedSessionSeconds"] for a in arms.values())},
    "cumulative": {"nativeStarts": len(rows), "reservedSessionSeconds": sum(row[5] for row in rows)},
    "historyPreserved": True, "historicalRoleLedgerHashesPreserved": True,
    "nativeBudget": budget_before_copy, "nativeBudgetArchive": saved,
    "historicalUnknownRealUsage": {"knownReportedTokens": 53331, "totalTokens": None},
    "newRealActorCalls": 0, "newModelProviderCalls": 0, "metadataSessions": 0, "studyCells": 0,
    "semanticAcceptance": False, "humanAcceptance": False,
    "accounting": "150 logical reserved session seconds/native start includes role/cleanup allowance; not an aggregate OS descendant time measurement.",
    "archivedExternalFiles": len(archived)}
d.write_new(d.EVIDENCE / "quota-and-ledger-summary.json", summary)
print(json.dumps({key: summary[key] for key in ("newConsumed", "cumulative", "historyPreserved", "archivedExternalFiles")}))
