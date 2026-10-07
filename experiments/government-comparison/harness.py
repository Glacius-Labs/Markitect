"""Bounded public adapter probe and lossless attempt ledger. Live gate stays closed."""
import argparse
from datetime import datetime, timezone
import json
from pathlib import Path
import subprocess
import time

from prepare import digest, git


def utc():
    return datetime.now(timezone.utc).isoformat()


def validate_result(request, result, request_sha256):
    if result.get("requestSha256") != request_sha256:
        raise ValueError("Result belongs to another exact request")
    for field in ("schemaVersion", "trialId", "operation", "mode"):
        if result.get(field) != request[field]:
            raise ValueError(f"Result binding mismatch: {field}")
    statuses = {"ready", "completed", "incomplete", "blocked", "readiness_gap", "failed", "stopped"}
    if result.get("status") not in statuses:
        raise ValueError("Unknown adapter status")
    if request["mode"] == "fixture" and result.get("candidateCommit") is not None:
        raise ValueError("Fixture may not supply a study candidate")
    if result["status"] == "ready" and request["operation"] != "probe":
        raise ValueError("ready is probe-only")
    if not isinstance(result.get("capabilities"), list) or not isinstance(result.get("gaps"), list):
        raise ValueError("capabilities/gaps must be arrays")
    if not isinstance(result.get("receipts"), list) or "usage" not in result:
        raise ValueError("Missing receipt or usage field")


def probe(request, adapter, directory):
    if request["mode"] != "fixture" or request["operation"] != "probe":
        raise ValueError("Live gate closed: frozen products, actual runner, metering and common budget required")
    repository = Path(request["actorRepository"])
    if not repository.is_absolute() or request.get("schemaVersion") != 1:
        raise ValueError("Expected schema v1 and absolute actorRepository")
    if request.get("arm") not in ("conventional", "classic", "government") or request.get("condition") not in ("greenfield", "brownfield"):
        raise ValueError("Unknown study cell")
    if not 0 < request["limits"]["wallSeconds"] <= 30 or request["limits"].get("maxActorCalls") != 0:
        raise ValueError("Fixture probes are bounded to 30 seconds and zero actor calls")
    for entry in request["releasedInputs"]:
        if not Path(entry["path"]).is_absolute() or digest(entry["path"]) != entry["sha256"]:
            raise ValueError("Released input missing or changed")
    if git(repository, "rev-parse", "HEAD") != request["baseCommit"] or git(repository, "status", "--porcelain"):
        raise ValueError("Actor start changed or dirty")
    directory = Path(directory).resolve()
    directory.mkdir(parents=True, exist_ok=False)
    request_file, result_file = directory / "request.json", directory / "result.json"
    request["evidenceDirectory"] = str(directory)
    request_file.write_text(json.dumps(request, indent=2) + "\n", encoding="utf-8")
    argv = [*adapter, "--request", str(request_file), "--result", str(result_file)]
    started, clock_start = utc(), time.monotonic()
    status, code, error = "failed", None, None
    result = None
    try:
        with (directory / "stdout.log").open("wb") as out, (directory / "stderr.log").open("wb") as err:
            process = subprocess.run(argv, cwd=repository, stdout=out, stderr=err,
                                     timeout=request["limits"]["wallSeconds"], check=False)
        code = process.returncode
        if code != 0:
            raise ValueError(f"adapter exit {code}")
        result = json.loads(result_file.read_text(encoding="utf-8"))
        validate_result(request, result, digest(request_file))
        status = result["status"]
    except (OSError, subprocess.TimeoutExpired, ValueError, KeyError) as exc:
        error = str(exc)
    event = {"schemaVersion": 1, "trialId": request["trialId"], "kind": "adapter_attempt",
             "mode": "fixture", "arm": request["arm"], "condition": request["condition"],
             "taskId": None, "startedUtc": started, "endedUtc": utc(),
             "wallSeconds": time.monotonic() - clock_start, "status": status, "argv": argv,
             "exitCode": code, "error": error, "requestSha256": digest(request_file),
             "resultSha256": digest(result_file) if result_file.exists() else None,
             "usage": result.get("usage") if result else None, "billedCost": None,
             "humanActiveSeconds": None, "coordinatorActiveSeconds": None, "waitSeconds": None,
             "agentActiveSeconds": None, "contaminated": False}
    (directory / "events.jsonl").write_text(json.dumps(event) + "\n", encoding="utf-8")
    return event


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--request", required=True)
    parser.add_argument("--evidence", required=True)
    parser.add_argument("adapter", nargs=argparse.REMAINDER, help="Exact argv after --")
    args = parser.parse_args()
    adapter = args.adapter[1:] if args.adapter[:1] == ["--"] else args.adapter
    if not adapter:
        parser.error("adapter argv required")
    event = probe(json.loads(Path(args.request).read_text(encoding="utf-8")), adapter, args.evidence)
    print(json.dumps(event, indent=2))
    raise SystemExit(0 if event["status"] in ("ready", "readiness_gap") else 1)
