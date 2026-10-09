"""Reconcile the terminal first pilot attempt and run only the still-unused second trajectory.

No replay of Roombook, native continuation, source repair or permission escalation.
Original bindings and controller-error receipts are retained unchanged.
"""
import argparse
from datetime import datetime
import json
from pathlib import Path
import run_conventional_pilot as pilot


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--destination", required=True, type=Path)
    args = parser.parse_args()
    destination = args.destination.resolve()
    binding = json.loads((destination / "pilot-binding.json").read_text(encoding="utf-8"))
    plan = binding["plan"]
    if plan["trajectories"] != [{"case":"roombook","backend":"codex-cli"},
                                 {"case":"readinglog","backend":"codex-app-server"}]:
        raise ValueError("not the verified two-trajectory order")
    audit = destination / "roombook/audit"
    turn = json.loads((audit / "turn-S1.json").read_text(encoding="utf-8"))
    if turn["state"] != "completed" or (audit / "conventional-execution/active.json").exists():
        raise ValueError("original actor must have authoritative terminal state and released ownership")
    if (destination / "readinglog").exists() or (audit / "reconciliation.json").exists():
        raise ValueError("reconciliation/second trajectory cannot be repeated")
    if pilot.git(pilot.ROOT, "status", "--porcelain"):
        raise ValueError("source must be clean and committed")
    expiry = datetime.fromisoformat(binding["overallExpiresAt"])
    if pilot.now() >= expiry:
        raise ValueError("original finite overall window expired")
    source_sha = pilot.git(pilot.ROOT, "rev-parse", "HEAD")
    pilot.lifecycle.record_execution(audit, metadata={"station":"S1", "wrapperState":turn["state"],
        "wrapperRunId":turn["runId"], "nativeSessionId":turn["nativeSessionId"],
        "reconciliation":"terminal receipt already existed; no redispatch"})
    captured = pilot.lifecycle.snapshot(destination / "roombook/repo", audit)
    assessment = pilot.CHECKER.assess(audit / "snapshot-S1/immutable-main", "roombook", 1)
    pilot.save(audit / "assessment-S1.json", assessment)
    frozen = pilot.lifecycle.freeze(destination / "roombook/repo", audit,
        reason="native shell setup failure; first actor terminated without implementation; no replay")
    receipt = {"case":"roombook", "backend":"codex-cli", "status":"blocked_by_native_shell_setup",
        "nativeState":turn["state"], "originalControllerError":"record_execution keyword-only metadata miscalled",
        "reconciliationSourceCommit":source_sha, "reconciliationControllerSha256":pilot.digest(__file__),
        "originalBindingPreserved":True, "redispatchedStarts":0, "publicChecks":assessment,
        "snapshotBindingSha256":pilot.digest(audit / "snapshot-S1/binding.json"),
        "finalMainCommit":frozen["immutableMain"]["commit"],
        "finalMainPath":str(audit / "final-freeze/immutable-main"),
        "humanAcceptance":"not established", "completedStations":[], "remainingStations":"NOT RUN"}
    pilot.save(audit / "reconciliation.json", receipt)
    print(json.dumps({"event":"first-trajectory-reconciled", "status":receipt["status"]}), flush=True)
    second = pilot.trajectory("readinglog", "codex-app-server", destination, plan,
        Path(binding["executable"]), binding["version"], source_sha, expiry)
    pilot.save(destination / "reconciled-pilot-results.json", {"trajectories":[receipt,second],
        "originalPilotResultsPreserved":True, "interpretation":"case and transport confounded; no comparative winner"})
    print(json.dumps({"event":"unused-second-trajectory-ended", "status":second["status"]}), flush=True)


if __name__ == "__main__":
    for stream in (pilot.sys.stdin, pilot.sys.stdout, pilot.sys.stderr):
        if hasattr(stream, "reconfigure"): stream.reconfigure(encoding="utf-8")
    main()
