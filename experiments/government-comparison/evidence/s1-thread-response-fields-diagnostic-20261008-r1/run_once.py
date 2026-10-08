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
    # Full authority, committed fresh binding, pins and inputs before reservation.
    profile,request,freeze,head,assigned=client.validate_bound_inputs()
    external=Path(profile['externalEvidence'])
    client.require(not external.exists() and not external.is_symlink() and not external.is_junction(), 'unexpected-external-root')
    external.mkdir(exist_ok=False)
    client.exclusive_json(external/'output-root-created.json', {'key':client.KEY,'createdAtUnix':time.time(),'grantIssuedUtc':client.ISSUED,'slotAssignedUtc':assigned,'createdExclusively':True})
    # Recheck immediately before durable allocation. Any failure is terminal.
    checked_profile,checked_request,checked_freeze,checked_head,checked_assigned=client.validate_bound_inputs()
    client.require(checked_head==head and checked_assigned==assigned and checked_profile==profile and checked_request==request and checked_freeze==freeze, 'reservation-boundary-drift')
    client.exclusive_json(external/'reservation.json', {'key':client.KEY,'reservedAtUnix':time.time(),'grantIssuedUtc':client.ISSUED,'slotAssignedUtc':assigned,'sourceCommit':request['sourceCommit'],'freezeCommit':head,'requestSha256':client.sha((root/'request.json').read_bytes()),'freezeSha256':client.sha((root/'freeze.json').read_bytes()),'maxTrees':1,'maxDiagnosticReservations':1,'maxActorTaskReservations':0,'priorAppServerTrees':9,'priorActorReservations':6,'priorCliMetadataCalls':8,'cumulativeAppServerMaximum':10})
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
