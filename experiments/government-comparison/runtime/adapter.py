"""Scientist-owned v1.1 Request/Result shell. Genuine native probes; no live study."""
import argparse
import hashlib
import json
from pathlib import Path
import sys

import classic
from process import bounded
import runner


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def read_request(raw):
    request = json.loads(raw)
    if request.get("schemaVersion") != 1 or request.get("mode") != "fixture":
        raise ValueError("Live study gate closed; this package accepts fixture readiness requests only")
    if request.get("arm") not in {"conventional", "classic", "government"}:
        raise ValueError("unknown arm")
    if request.get("condition") not in {"greenfield", "brownfield"}:
        raise ValueError("unknown condition")
    if request.get("operation") not in {"probe", "run_task", "resume", "stop"}:
        raise ValueError("unknown operation")
    if request.get("task") is not None or request["limits"].get("maxActorCalls") != 0:
        raise ValueError("readiness accepts no study task or Actor calls")
    if not 0 < request["limits"]["wallSeconds"] <= 30:
        raise ValueError("fixture wall bound must be in (0,30]")
    root = Path(request["actorRepository"])
    if not root.is_absolute() or not root.is_dir():
        raise ValueError("absolute existing actorRepository required")
    study_checkout = Path(__file__).resolve().parents[3]
    if root.resolve() == study_checkout or study_checkout in root.resolve().parents:
        raise ValueError("Actor root must be outside study checkout hierarchy")
    for entry in request["releasedInputs"]:
        if not Path(entry["path"]).is_absolute() or sha(entry["path"]) != entry["sha256"]:
            raise ValueError("released input binding mismatch")
    return request


def handle(request_path, result_path):
    raw = Path(request_path).read_bytes()
    raw_digest = hashlib.sha256(raw).hexdigest()
    request = read_request(raw)
    result = {"schemaVersion": 1, "trialId": request["trialId"], "requestSha256": raw_digest,
              "operation": request["operation"], "mode": request["mode"], "status": "readiness_gap",
              "candidateCommit": None, "capabilities": [], "gaps": [], "receipts": [], "usage": None}
    directory = Path(request["evidenceDirectory"])
    if not directory.is_absolute() or not directory.is_dir():
        raise ValueError("absolute existing evidenceDirectory required")
    if request["operation"] != "probe":
        result["gaps"] = ["No live invocation or persistent product resume is enabled; stop applies only to an active bounded call via its STOP sentinel"]
    elif request["arm"] == "government":
        result["gaps"] = ["Government native handoff and immutable product pins absent", "G1-G5 evidence missing; no Government simulation is permitted"]
    else:
        try:
            if request["arm"] == "classic":
                packet = request["product"]["packetPath"]
                inspection = classic.inspect_packet(packet)
                command = classic.probe_command(inspection["binary"]["path"])
                result["capabilities"] = inspection["capabilities"]
                result["gaps"] = inspection["gaps"] + runner.GAPS
                identity = inspection
            else:
                identity = runner.inspect(request["profile"]["runnerExecutable"])
                command = [identity["path"], "--version"]
                result["capabilities"] = ["native Codex executable version probe", "prospective conventional Actor with ordinary tools; same public requirements and common profile"]
                result["gaps"] = runner.GAPS
            receipt = bounded(command, request["actorRepository"], directory / "native-probe",
                              request["limits"]["wallSeconds"], stop_path=directory / "STOP")
            identity_file = directory / "native-identity.json"
            identity_file.write_text(json.dumps(identity, indent=2) + "\n", encoding="utf-8")
            result["receipts"] = receipt["receipts"] + [
                {"path": str(identity_file), "sha256": sha(identity_file), "kind": "native-identity"},
                {"path": str(directory / "native-probe/process.json"), "sha256": sha(directory / "native-probe/process.json"), "kind": "process"}]
            if receipt["stopReason"]:
                result["status"] = "stopped"
            elif receipt["returnCode"] != 0:
                result["status"] = "failed"
            else:
                result["capabilities"].append("native version probe succeeded; zero provider calls")
        except (OSError, ValueError, KeyError) as exc:
            if isinstance(exc, OSError):
                result["status"] = "failed"
            result["gaps"].append(str(exc))
    Path(result_path).write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--request", required=True)
    parser.add_argument("--result", required=True)
    args = parser.parse_args()
    try:
        handle(args.request, args.result)
    except (OSError, ValueError, KeyError, TypeError) as exc:
        print(str(exc), file=sys.stderr)
        raise SystemExit(1)
