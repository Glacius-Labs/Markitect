"""Read-only evidence checks; never invokes a native product or metadata session."""
import hashlib
import json
from pathlib import Path
from datetime import datetime, timezone

STUDY = Path(__file__).resolve().parent
RUN = STUDY / "evidence/policy-read-classified/run-1"


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def main():
    prior = json.loads((STUDY / "evidence/policy-read/run-1/validation.json").read_bytes())
    preserved = {**prior["files"], **prior["preservedEarlierFiles"]}
    preserved["evidence/policy-read/run-1/validation.json"] = "45e04af36ce533d6933f182096163e8497cded11d1d2fddd4558b3376b210428"
    for path, expected in preserved.items():
        if sha(STUDY / path) != expected:
            raise ValueError("historical evidence changed: " + path)
    for path, expected in prior["preservedLedgers"].items():
        if sha(path) != expected:
            raise ValueError("historical ledger changed")
    request = json.loads((RUN / "request.json").read_bytes())
    freeze = json.loads((RUN / "freeze.json").read_bytes())
    reservation = json.loads((RUN / "reservation.json").read_bytes())
    if reservation["freezeSha256"] != sha(RUN / "freeze.json"):
        raise ValueError("reservation/freeze mismatch")
    for path, expected in freeze["files"].items():
        if sha(path) != expected:
            raise ValueError("frozen input changed")
    result = json.loads((RUN / "sanitized-result.json").read_bytes())
    process = json.loads((RUN / "process/process.json").read_bytes())
    if result["sentRpc"] != request["rpc"][:len(result["sentRpc"])]:
        raise ValueError("outbound RPC sequence differs")
    methods = json.loads((RUN / "frozen-method-enums.json").read_bytes())["methods"]
    for event in result["serverMethodEvents"]:
        if set(event) != {"method", "class"} or not any(
                event["method"] in enum and enum[event["method"]]["class"] == event["class"]
                for enum in methods.values()):
            raise ValueError("server event retention exceeds frozen method/class")
    provisional = any(e["method"] in {"warning", "configWarning", "windows/worldWritableWarning"}
                      for e in result["serverMethodEvents"])
    if provisional and result["provisionalConfig"] is not True:
        raise ValueError("warning result lacks provisional flag")
    if not (result["wallSeconds"] < 50 and process["wallSeconds"] < 60
            and process["processTreeControl"] == "windows-job-kill-on-close"
            and result["automaticRetries"] == process["automaticRetries"] == 0
            and result["rawFramePayloadsPersisted"] is False
            and result["rawConfigurationPersisted"] is False):
        raise ValueError("session boundary failed")
    if (RUN / "process/stderr.log").read_bytes():
        raise ValueError("unexpected outer client stderr")
    for receipt in process["receipts"]:
        if sha(receipt["path"]) != receipt["sha256"]:
            raise ValueError("process receipt mismatch")
    mechanics = json.loads((STUDY / "evidence/s1-adapter-mechanics/validation.json").read_bytes())
    if mechanics["exitCode"] != 0 or mechanics["testsRun"] != 23:
        raise ValueError("final focused mechanics gate did not pass")
    if sha(STUDY / "evidence/s1-adapter-mechanics/test.log") != mechanics["testLogSha256"]:
        raise ValueError("mechanics log changed")
    for path, expected in mechanics["sourceFiles"].items():
        if sha(STUDY / "runtime" / path) != expected:
            raise ValueError("mechanics-tested source changed: " + path)
    accepted = json.loads((STUDY / "runtime/government-pin.json").read_bytes())["accepted"]
    for key in ("handoff", "binary"):
        if sha(accepted[key]["path"]) != accepted[key]["sha256"]:
            raise ValueError("accepted Government artifact changed")
    paths = [p for p in RUN.rglob("*") if p.is_file()
             and p.name != "validation.json" and "__pycache__" not in p.parts]
    paths += [p for p in (STUDY / "evidence/s1-adapter-mechanics").rglob("*")
              if p.is_file() and "__pycache__" not in p.parts]
    paths += [STUDY / p for p in (
        "classified-policy-s1-handoff.md", "validate-classified-delivery.py",
        "public/policy-classified-review.md", "public/s1-adapter-review.md",
        "public/s1-adapter-mechanics.md", "runtime/government.py", "runtime/government-pin.json",
        "runtime/classic-pin.json", "runtime/test_government.py", "runtime/government_roles.py",
        "runtime/test_government_roles.py",
        "runtime/adapter.py", "runtime/dispatch.py", "runtime/test_dispatch.py", "runtime/test_runtime.py")]
    delivery = {str(p.relative_to(STUDY)).replace("\\", "/"): sha(p) for p in paths}
    validation = {"kind": "classified-metadata-and-s1-mechanics-validation",
        "validatedAtUtc": datetime.now(timezone.utc).isoformat(), "files": delivery,
        "preservedEarlierFiles": preserved, "preservedLedgers": prior["preservedLedgers"],
        "checks": {"frozenFilesUnchanged": True, "priorFilesAndLedgersUnchanged": True,
                   "allowedRpcPrefixOnly": True, "knownMethodAndClassOnly": True,
                   "warningProvisionalRule": True, "innerUnder50OuterUnder60": True,
                   "WindowsJob": True, "zeroOuterStderr": True, "receiptHashes": True,
                   "final23MechanicsTestsAndSources": True, "acceptedGovernmentArtifactHashes": True},
        "status": {"metadataSession": result["status"], "provisionalConfig": result["provisionalConfig"],
                   "newAppServerTrees": 1, "priorAppServerTrees": 1, "priorCliMetadataCalls": 6,
                   "actorStartsAdded": 0, "actorStartsCumulative": 5,
                   "knownActorInputPlusOutputTokens": 53331, "historicalActorTokenTotal": None,
                   "GovernmentPin": "04e225d5caee78c2a198607143863fca1e829750; Overseer accepted G5 technical mechanics",
                   "S1": "open", "studyCells": 0},
        "preflightArchive": {"originalNoteSha256": reservation["reviewSha256"],
            "originalBytesAvailable": False,
            "evidence": "Reservation command verified original hash before launch; chat recorded clearance; later edited review preserved"},
        "limitations": ["Mechanical helpers do not establish native per-role enforcement or S1 readiness",
            "Original preflight review bytes unavailable after post-run editing; no exact copy fabricated",
            "Metadata route differs from historical exec; warning text discarded and never interpreted",
            "Preparation time/token/cost totals unavailable; no provider/native internal trace"]}
    (RUN / "validation.json").write_text(json.dumps(validation, indent=2)+"\n", encoding="utf-8")
    print(json.dumps({"deliveryFiles": len(delivery), "preservedFiles": len(preserved),
                      "preservedLedgers": len(prior["preservedLedgers"]), "validationSha256": sha(RUN / "validation.json")}))


if __name__ == "__main__":
    main()
