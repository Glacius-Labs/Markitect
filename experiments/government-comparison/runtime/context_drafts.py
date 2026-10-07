"""Concrete one-session context DRAFTS; cannot dispatch or approve a live grant."""
import argparse
import json
from pathlib import Path
import subprocess
import sys

from diagnostic_history import read_history
from dispatch import CHECKOUT, LIVE_GAPS, digest, encoded, execution_sha, external, runtime_pins
from measurement_profile import LEGACY, OBSERVED, limits_sha, profile_sha
from metadata_readonly import collect
import runner
from context_allocation import ALLOCATION_ID, ALLOCATION_PATH, allocation_binding


def write(path, value):
    path.write_bytes(json.dumps(value, indent=2).encode() + b"\n")
    return {"path": str(path), "sha256": digest(path.read_bytes())}


def prepare(destination, executable, *, collect_current_metadata=False):
    """Use a clean committed wrapper. Metadata is explicit and never inference."""
    root = external(destination)
    if root.exists():
        raise ValueError("fresh context-draft directory required")
    revision = subprocess.check_output(["git", "-C", str(CHECKOUT), "rev-parse", "HEAD"], text=True).strip()
    if subprocess.check_output(["git", "-C", str(CHECKOUT), "status", "--porcelain"], text=True).strip():
        raise ValueError("clean committed source required before concrete Draft preparation")
    if collect_current_metadata is not True:
        raise ValueError("explicit metadata-only collection required; no cached substitution")
    pin = runner.inspect(executable)
    root.mkdir(parents=True)
    listing = collect(root / "metadata", executable)
    history = read_history()
    history_receipt = write(root / "diagnostic-history.json", history)
    previous = json.loads(Path(__file__).parents[1].joinpath("evidence/profile-v2/run-1/request.draft.json").read_bytes())
    actor, results = Path(previous["actorRepository"]), root / "results"
    results.mkdir()
    external_file, owned_file = map(Path, previous["allowedReadPaths"])
    def binding(path):
        return {"path": str(path), "sha256": digest(path.read_bytes())}
    reads = [str(external_file), str(owned_file)]
    prompt_path, card = Path(previous["prompt"]["path"]), Path(previous["task"]["card"]["path"])
    for item in previous["releasedInputs"] + previous["actorOwnedInputs"]:
        if digest(Path(item["path"]).read_bytes()) != item["sha256"]:
            raise ValueError("previously named context input changed")
    runner.validate_start(previous)
    base = previous["baseCommit"]
    common = json.loads(Path(__file__).parents[1].joinpath("public/resource-proposal.json").read_bytes())["commonLimits"]
    trial, ledger_path = ALLOCATION_ID, ALLOCATION_PATH
    cumulative = allocation_binding()
    request = {"schemaVersion": 1, "operation": "run_task", "mode": "live", "trialId": trial,
               "dispatchId": "public-context-1", "arm": "conventional", "condition": "greenfield",
               "purpose": "context-access", "smokeKind": "effective-context-only", "actorRepository": str(actor),
               "baseCommit": base, "evidenceDirectory": str(root / "context-evidence"), "limits": common,
               "wallSeconds": 180, "task": {"id": "public-synthetic-context-1", "card": binding(card)},
               "prompt": binding(prompt_path), "releasedInputs": [binding(card), binding(prompt_path), binding(external_file)],
               "actorOwnedInputs": [binding(owned_file)], "toolPolicy": "ordinary-tools", "sandbox": "read-only",
               "allowedReadPaths": reads, "sourceCandidate": revision, "contextAllocationDecisionId": ALLOCATION_ID,
               "measurementProfileId": OBSERVED, "measurementProfileSha256": profile_sha(OBSERVED)}
    request_receipt = write(root / "request.draft.json", request)
    adoption = {"status": "draft", "trialId": trial, "ledgerPath": str(ledger_path),
                "fromProfileSha256": profile_sha(LEGACY), "toProfileSha256": profile_sha(OBSERVED),
                "limitsSha256": limits_sha(common),
                "decisionRef": ALLOCATION_ID + "; resource decision approved, exact Run-Grant and this profile binding remain drafts"}
    adoption_receipt = write(root / "profile-adoption.draft.json", adoption)
    protocol = {"schemaVersion": 1, "status": "draft", "mode": "live", "sourceCandidate": revision,
                "commonLimits": common, "runtimeSourceSha256": runtime_pins(),
                "wrapperPythonSha256": digest(Path(sys.executable).read_bytes()), "wrapperPython": str(Path(sys.executable).resolve()),
                "runnerExecutable": pin["path"], "runnerPinSha256": digest(encoded(pin)), "runnerPin": pin,
                "configSha256": pin["configSha256"], "requestedModel": runner.MODEL, "requestedReasoning": runner.REASONING,
                "authenticatedModelListing": listing, "measurementProfileId": OBSERVED,
                "measurementProfileSha256": profile_sha(OBSERVED), "profileAdoption": adoption_receipt,
                "toolPolicy": "ordinary-tools", "sandbox": "read-only", "allowedReadPaths": reads,
                "contextOnly": True, "forbiddenActions": ["write", "access-probe", "process-probe", "credential-read", "other-cell-read"],
                "toolRuleSemantics": "Cooperative exact-file-read rule; requested read-only sandbox and disabled features are not a proven access barrier",
                "acceptedObservabilityGaps": LIVE_GAPS, "historicalAccounting": history_receipt, "contextAllocation": cumulative,
                "taskCard": binding(card), "prompt": binding(prompt_path), "actorOwnedInputs": request["actorOwnedInputs"]}
    protocol_receipt = write(root / "protocol.draft.json", protocol)
    grant = {"schemaVersion": 1, "status": "draft", "mode": "live", "purpose": "s1-public-smoke", "trialId": trial,
             "sourceCandidate": revision, "notBefore": None, "expiresAt": None, "protocolSha256": protocol_receipt["sha256"],
             "profileSha256": limits_sha(common), "runnerPinSha256": protocol["runnerPinSha256"],
             "measurementProfileId": OBSERVED, "measurementProfileSha256": profile_sha(OBSERVED),
             "ledgerPath": str(ledger_path.resolve()), "resultDirectory": str(results), "maxActorSessions": 1, "cumulativeSessionCeiling": 4,
             "maxAdditionalActorSessions": 1, "maxParallelSessions": 1,
             "maxSessionWallSeconds": 180, "wrapperAgentTurns": 1, "wrapperRetries": 0, "continuations": 0, "children": 0,
             "semanticRepairs": 0, "newPurchases": False, "retrospectiveTokenThreshold": 10000,
             "hardTokenCap": False, "providerRequests": None, "internalTransportRetries": None,
             "acceptedObservabilityGaps": LIVE_GAPS, "allowedReadPaths": reads, "contextOnly": True,
             "authorizedRequests": [{"dispatchId": request["dispatchId"], "initialRequestSha256": request_receipt["sha256"],
                                     "executionSha256": execution_sha(request)}],
             "historicalAccounting": history_receipt, "contextAllocation": cumulative,
             "resourceAllocation": "One additional resource allocation explicitly decided; exact Run-Grant remains draft; old grants exhausted, no refill",
             "approvalStillRequired": ["Coordinator approves exact one-session resource grant and finite validity interval",
                                       "Ledger profile adoption draft approved; protocol frozen and final hashes rebound"]}
    grant_receipt = write(root / "grant.draft.json", grant)
    manifest = {"schemaVersion": 1, "status": "drafts-only-no-live-authorization", "sourceCandidate": revision,
                "request": request_receipt, "protocol": protocol_receipt, "grant": grant_receipt,
                "profileAdoption": adoption_receipt, "executionSha256": execution_sha(request),
                "modelListing": listing, "diagnosticHistory": history_receipt, "publicInputs": request["releasedInputs"],
                "actorOwnedInputs": request["actorOwnedInputs"], "actorBaseCommit": base, "requestedModel": runner.MODEL,
                "requestedReasoning": runner.REASONING, "actorStarts": 0, "inferenceCalls": 0,
                "liveGrantIssued": False, "ledgerCreated": False, "metadataOnly": True,
                "liveExecutable": False, "contextAllocation": cumulative,
                "actorScope": "Only the two named synthetic context files; no separate access/write/process probe"}
    manifest_receipt = write(root / "draft-manifest.json", manifest)
    print(json.dumps({**manifest, "manifest": manifest_receipt}, indent=2))
    return manifest


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--destination", required=True)
    parser.add_argument("--runner", required=True)
    parser.add_argument("--collect-current-metadata", action="store_true")
    args = parser.parse_args()
    prepare(args.destination, args.runner, collect_current_metadata=args.collect_current_metadata)
