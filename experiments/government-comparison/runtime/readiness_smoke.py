"""Preserve genuine native version probes outside the study checkout. Zero Actors."""
import argparse
import hashlib
import json
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))
from harness import probe
from prepare import git
from process import bounded
import runner


def run(destination, executable, packet):
    destination = Path(destination).resolve()
    checkout = ROOT.parents[1]
    if checkout == destination or checkout in destination.parents or destination in checkout.parents:
        raise ValueError("readiness destination must be independent of study checkout hierarchy")
    destination.mkdir(parents=True, exist_ok=False)
    sentinel = destination / "PUBLIC-SENTINEL.txt"
    sentinel.write_text("PUBLIC readiness sentinel; no private or study task data.\n", encoding="utf-8")
    result = {"kind": "native-runner-adapter-readiness", "liveTrials": 0, "providerProbeSessions": 0,
              "actorRoot": str(destination), "model": None, "requestedModel": runner.MODEL,
              "requestedReasoning": runner.REASONING, "probes": [],
              "providerProbeDisposition": "not launched: no evidenced pre-dispatch six-provider-call limit",
              "accessProbe": "not executed by actual Actor; public sentinel is setup only",
              "isolation": "independent directories and zero Actor data handoff; no OS privacy claim"}
    for arm in ("conventional", "classic", "government"):
        actor = destination / (arm + "-actor")
        actor.mkdir()
        git(actor, "init", "-b", "codex/native-readiness")
        git(actor, "config", "core.autocrlf", "false")
        git(actor, "-c", "user.name=Scientist Fixture", "-c", "user.email=scientist-fixture@example.invalid",
            "commit", "--allow-empty", "-m", "Empty native readiness root; no actor task")
        request = {"schemaVersion": 1, "operation": "probe", "mode": "fixture", "trialId": arm + "-native-readiness",
                   "arm": arm, "condition": "greenfield", "actorRepository": str(actor),
                   "baseCommit": git(actor, "rev-parse", "HEAD"), "task": None,
                   "releasedInputs": [{"path": str(sentinel), "sha256": hashlib.sha256(sentinel.read_bytes()).hexdigest()}],
                   "product": {"packetPath": str(Path(packet).resolve())} if arm == "classic" else None,
                   "profile": {"runnerExecutable": str(Path(executable).resolve()), "model": runner.MODEL,
                               "reasoning": runner.REASONING, "runnerVersion": runner.EXPECTED_VERSION},
                   "limits": {"wallSeconds": 30, "maxActorCalls": 0, "maxParallelActors": 1}, "evidenceDirectory": "pending"}
        event = probe(request, [sys.executable, str(Path(__file__).parent / "adapter.py")], destination / "receipts" / arm)
        if event["status"] != "readiness_gap":
            raise ValueError("native readiness wrapper smoke failed: " + json.dumps(event))
        result["probes"].append(event)
    pin = runner.inspect(executable)
    result["runnerPin"] = pin
    for name, args in (("exec-help", ["exec", "--help"]),
                       ("protocol-schema", ["app-server", "generate-json-schema", "--experimental", "--out", str(destination / "protocol")])):
        receipt = bounded([pin["path"], *args], str(destination), destination / name, 30)
        if receipt["returnCode"] != 0 or receipt["stopReason"]:
            raise ValueError("native introspection failed: " + name)
        result[name] = receipt
    result["protocolSchemaDigests"] = {p.relative_to(destination / "protocol").as_posix(): hashlib.sha256(p.read_bytes()).hexdigest()
                                      for p in sorted((destination / "protocol").rglob("*.json"))}
    result["runtimeSourceDigests"] = {p.relative_to(ROOT).as_posix(): hashlib.sha256(p.read_bytes()).hexdigest()
                                      for p in sorted((ROOT / "runtime").glob("*")) if p.is_file()}
    result["status"] = "wrapper-smoke-passed-live-gate-closed"
    (destination / "readiness-smoke.json").write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--destination", required=True)
    parser.add_argument("--runner", required=True)
    parser.add_argument("--classic-packet", required=True)
    args = parser.parse_args()
    output = run(args.destination, args.runner, args.classic_packet)
    print(json.dumps({"status": output["status"], "providerProbeSessions": 0, "liveTrials": 0, "destination": output["actorRoot"]}))
