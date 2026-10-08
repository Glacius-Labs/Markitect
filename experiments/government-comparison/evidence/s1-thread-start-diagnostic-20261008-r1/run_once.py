"""Reserve once, execute the reviewed closed thread-start-only diagnostic controller, read receipt.

No raw output capture or automatic retries. Importing has no effect.
"""
import json
from pathlib import Path
import subprocess
import time
import client


def main():
    outer_started = time.monotonic()
    root = Path(__file__).resolve().parent
    profile = client.strict_json((root / "profile.json").read_bytes())
    request = client.strict_json((root / "request.json").read_bytes())
    freeze_bytes = (root / "freeze.json").read_bytes()
    freeze = client.strict_json(freeze_bytes)
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=client.REPO).decode().strip()
    client.require(not subprocess.check_output(["git", "status", "--porcelain"], cwd=client.REPO).strip(), "dirty-prestart-tree")
    client.require(freeze["sourceCommit"] == request["sourceCommit"], "source-binding-mismatch")
    external = Path(profile["externalEvidence"])
    # Exclusive, durable reservation before any process capable of launching
    # Codex. Failure closes the grant; this entrypoint may never be retried.
    client.exclusive_json(external / "reservation.json", {"key": client.KEY, "reservedAtUnix": time.time(), "sourceCommit": request["sourceCommit"], "freezeCommit": head, "requestSha256": client.sha((root / "request.json").read_bytes()), "freezeSha256": client.sha(freeze_bytes), "maxTrees": 1, "maxDiagnosticReservations": 1, "maxActorTaskReservations": 0, "priorAppServerTrees": 6, "priorActorReservations": 6, "cumulativeAppServerMaximum": 7})
    code = 2
    reason = None
    try:
        code = client.controller()
    except BaseException:
        reason = "prestart-or-controller-failure"
    receipt_path = external / "controller-receipt.json"
    result_path = external / "sanitized-result.json"
    receipt = client.strict_json(receipt_path.read_bytes()) if receipt_path.is_file() else None
    result = client.strict_json(result_path.read_bytes()) if result_path.is_file() else None
    elapsed = time.monotonic()-outer_started
    summary = {"key": client.KEY, "outerReturnCode": code, "stopReason": reason, "outerThroughReceiptReadbackSeconds": elapsed, "withinOuterBound": elapsed <= 80, "controllerReceiptSha256": client.sha(receipt_path.read_bytes()) if receipt else None, "sanitizedResultSha256": client.sha(result_path.read_bytes()) if result else None, "terminalResultPresent": result is not None, "status": result["status"] if result else "terminal-result-missing", "nativeStartAttemptRecorded": (external / "started-once.json").exists(), "diagnosticReservations": 1, "actorTaskReservations": 0, "nativeProviderRequests": None, "nativeProviderRetries": None, "remainingGrantClosed": True}
    client.exclusive_json(external / "outer-receipt.json", summary)
    print(json.dumps(summary))
    return 0 if code == 0 and elapsed <= 80 else 1


if __name__ == "__main__":
    try:
        code = main()
    except BaseException:
        print('{"status":"preparation-blocked-or-already-reserved","automaticRetries":0}')
        code = 2
    raise SystemExit(code)
