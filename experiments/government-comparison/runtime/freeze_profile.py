"""Freeze exact committed profile sources, local mechanics and unapproved drafts."""
import argparse
import json
from pathlib import Path
import subprocess
import zipfile

from diagnostic_history import read_history
from dispatch import Authority, CHECKOUT, LIVE_GAPS, digest, encoded, execution_sha, runtime_pins, validate_listing
from measurement_profile import OBSERVED, profile_sha


def freeze(evidence_path, drafts_path, output):
    root = Path(__file__).parents[1]
    evidence, drafts = Path(evidence_path).resolve(), Path(drafts_path).resolve()
    observation = json.loads((evidence / "observation.json").read_bytes())
    manifest = json.loads((drafts / "draft-manifest.json").read_bytes())
    head = subprocess.check_output(["git", "-C", str(CHECKOUT), "rev-parse", "HEAD"], text=True).strip()
    if (not observation["successful"] or not observation["historicalLedgersUnchanged"] or
            observation["runtimeSourceSha256"] != runtime_pins() or observation["sourceHeadAtVerification"] != head or
            any(observation[key] != 0 for key in ("actorStarts", "inferenceCalls", "nativeRunnerStarts", "studyCellsExecuted", "metadataRpcCalls"))):
        raise ValueError("passing exact-source zero-Actor mechanics required")
    if (manifest["status"] != "drafts-only-no-live-authorization" or manifest["sourceCandidate"] != head or
            manifest["actorStarts"] != 0 or manifest["inferenceCalls"] != 0 or manifest["liveGrantIssued"] or manifest["ledgerCreated"]):
        raise ValueError("exact-source unapproved zero-Actor drafts required")
    files = {}
    for entry in [manifest[key] for key in ("request", "protocol", "grant", "profileAdoption", "modelListing", "diagnosticHistory")] + manifest["publicInputs"] + manifest["actorOwnedInputs"]:
        path = Path(entry["path"])
        if digest(path.read_bytes()) != entry["sha256"]:
            raise ValueError("draft/input digest mismatch: " + str(path))
        files[str(path)] = entry["sha256"]
    files[str(drafts / "draft-manifest.json")] = digest((drafts / "draft-manifest.json").read_bytes())
    metadata_root = Path(manifest["modelListing"]["path"]).parent
    for name in ("request.json", "process/stdout.log", "process/stderr.log", "process/process.json"):
        path = metadata_root / name
        files[str(path)] = digest(path.read_bytes())
    request, protocol, grant, adoption = [json.loads(Path(manifest[key]["path"]).read_bytes())
                                         for key in ("request", "protocol", "grant", "profileAdoption")]
    if (any(item["status"] != "draft" for item in (protocol, grant, adoption)) or manifest["liveExecutable"] or
            grant["cumulativeLedgerBinding"]["status"] != "blocked-legacy-diagnostic-schema" or
            digest(Path(grant["ledgerPath"]).read_bytes()) != grant["cumulativeLedgerBinding"]["existingLedgerSha256"]):
        raise ValueError("unapproved drafts must preserve existing blocked ledger binding")
    if (execution_sha(request) != manifest["executionSha256"] or grant["authorizedRequests"] != [{
            "dispatchId": request["dispatchId"], "initialRequestSha256": manifest["request"]["sha256"],
            "executionSha256": manifest["executionSha256"]}] or grant["protocolSha256"] != manifest["protocol"]["sha256"]):
        raise ValueError("exact Request/execution/Protocol binding mismatch")
    if (grant["maxActorSessions"] != 4 or grant["maxAdditionalActorSessions"] != 1 or
            grant["maxParallelSessions"] != 1 or grant["maxSessionWallSeconds"] != 180 or
            grant["wrapperAgentTurns"] != 1 or grant["retrospectiveTokenThreshold"] != 10000 or
            any(grant[key] != 0 for key in ("children", "wrapperRetries", "continuations", "semanticRepairs")) or
            grant["newPurchases"] or grant["hardTokenCap"] or grant["providerRequests"] is not None or
            grant["internalTransportRetries"] is not None or grant["notBefore"] is not None or grant["expiresAt"] is not None):
        raise ValueError("finite context proposal bounds mismatch")
    if (any(item["measurementProfileId"] != OBSERVED or item["measurementProfileSha256"] != profile_sha(OBSERVED)
            for item in (request, protocol, grant)) or request["smokeKind"] != "effective-context-only" or
            protocol["requestedModel"] != "gpt-6.1-sol" or protocol["requestedReasoning"] != "high" or
            protocol["sandbox"] != "read-only" or protocol["acceptedObservabilityGaps"] != LIVE_GAPS or
            len(request["allowedReadPaths"]) != 2 or protocol["allowedReadPaths"] != request["allowedReadPaths"]):
        raise ValueError("equal profile or context-only scope mismatch")
    validate_listing(manifest["modelListing"])
    try:
        Authority(manifest["grant"]["path"], manifest["grant"]["sha256"], manifest["protocol"]["path"],
                  manifest["protocol"]["sha256"], allow_live=True)
    except ValueError as exc:
        if "approved Coordinator grant and frozen protocol required" not in str(exc):
            raise
        draft_rejection = str(exc)
    else:
        raise ValueError("unapproved drafts unexpectedly passed authority gate")
    with zipfile.ZipFile(evidence / "mechanical-records.zip") as archive:
        entries = json.loads((evidence / "mechanical-records-manifest.json").read_bytes())
        if sorted(archive.namelist()) != sorted(item["path"] for item in entries):
            raise ValueError("mechanical archive inventory mismatch")
        for item in entries:
            raw = archive.read(item["path"])
            if digest(raw) != item["sha256"] or len(raw) != item["bytes"]:
                raise ValueError("mechanical archive digest mismatch")
    names = [name if name.startswith("public/") or name in {"harness.py", "prepare.py"} else "runtime/" + name
             for name in runtime_pins()]
    names += ["runtime/" + name + ".py" for name in ("test_dispatch", "test_measurement_profile", "test_runtime", "verify_profile",
              "freeze_profile", "freeze_s1", "context_drafts", "diagnostic_history", "metadata_readonly", "selected_metadata")]
    names += ["README.md", "runtime/runner-pin.json", "public/resource-proposal.json", "public/measurement-profile-v2.md",
              "public/measurement-profile-v2-review.md", "public/s1-dispatch.md"]
    sources = {name: digest((root / name).read_bytes()) for name in sorted(set(names))}
    for name, sha in sources.items():
        committed = subprocess.check_output(["git", "-C", str(CHECKOUT), "show", head + ":experiments/government-comparison/" + name])
        if digest(committed) != sha:
            raise ValueError("source differs from committed candidate: " + name)
    original = subprocess.check_output(["git", "-C", str(CHECKOUT), "show",
        "b16d70407016ca25651935c2c9283ed5511212e1:experiments/government-comparison/public/resource-proposal.json"])
    if original != (root / "public/resource-proposal.json").read_bytes():
        raise ValueError("numeric proposal source changed")
    history = read_history()
    if json.loads(Path(manifest["diagnosticHistory"]["path"]).read_bytes()) != history:
        raise ValueError("historical cumulative accounting changed")
    value = {"schemaVersion": 1, "kind": "measurement-profile-v2-and-context-drafts-freeze", "sourceCandidate": head,
             "sources": sources, "evidence": {p.relative_to(root).as_posix(): digest(p.read_bytes()) for p in sorted(evidence.iterdir()) if p.is_file()},
             "draftFiles": files, "draftManifestPath": str(drafts / "draft-manifest.json"), "draftAuthorityRejection": draft_rejection,
             "mechanicalArchiveEntries": len(entries), "testsRun": observation["testsRun"], "measurementProfileSha256": profile_sha(OBSERVED),
             "commonLimitsUnchanged": True, "historicalDiagnosticAccounting": history, "actorStarts": 0, "inferenceCalls": 0,
             "authenticatedMetadataOnly": True, "liveProtocolFrozen": False, "liveRunGrantIssued": False, "s1": "open", "nextAction": "hold"}
    with Path(output).open("xb") as out:
        out.write(encoded(value) + b"\n")
    print(json.dumps({"sourceCandidate": head, "sources": len(sources), "evidence": len(value["evidence"]),
                      "draftFiles": len(files), "mechanicalArchiveEntries": len(entries), "freezeSha256": digest(Path(output).read_bytes())}, indent=2))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--evidence", required=True)
    parser.add_argument("--drafts", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()
    freeze(args.evidence, args.drafts, args.output)
