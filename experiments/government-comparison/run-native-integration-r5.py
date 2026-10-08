"""One bounded Government scope-identity Queue and, only after full success, its Resume."""
from __future__ import annotations

import argparse
import copy
import json
import os
from pathlib import Path
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parent
sys.path.insert(0, str(ROOT / "runtime"))
import dispatch
import government
import government_integration
import government_roles
import classic
import classic_integration
import native_controller
import government_native_profile as native_profiles
import native_fixture_budget as native_budget
from native_fixture_budget import (DEADLINE, FixtureBudget,
                                   validate_r5_entry_gate,
                                   validate_r5_grant_binding)
from process import bounded

ACTIVE_PROFILE_NAME = "r5"
ACTIVE_PROFILE = native_profiles.profile(ACTIVE_PROFILE_NAME)
EXTERNAL = ACTIVE_PROFILE.external_root
EVIDENCE = ACTIVE_PROFILE.evidence_directory
LEGACY = Path(r"C:\Users\Consiliari\Documents\Scientist-Probes\native-metadata-fixtures-20261008")
SOURCE_GRANT = LEGACY / "released-native-grant.json"
SUCCESSOR_GRANT = ACTIVE_PROFILE.envelope_path
COORDINATOR_SNAPSHOT = ACTIVE_PROFILE.snapshot_path
R5_KEY = "government-scope-native-20261008-r5"
GRANT_SHA = "c80edc0e9864c1e332eaee81838dad0f33640aa96b226d30d14228375c493c90"
HISTORY = EVIDENCE / "historical-native-starts.sqlite"
EXTERNAL_HISTORY = EXTERNAL / "history-native-starts.sqlite"
LIVE_HISTORY = LEGACY / "native-starts.sqlite"
R1_GRANT_SHA = "b917f5a5eb99f0a607fd282a81acdf7b085e6a1dba14f89c8d0c32f349c82c3e"
COORDINATION = Path(r"C:\Users\Consiliari\Glacius Labs\Markitect\docs\design\government\coordination-state.json")
_PROFILE_QUEUE_STARTED = None


def select_profile(name):
    """Select one of the closed Government profiles for this CLI call."""
    global ACTIVE_PROFILE_NAME, ACTIVE_PROFILE, EXTERNAL, EVIDENCE
    global SUCCESSOR_GRANT, COORDINATOR_SNAPSHOT, HISTORY, EXTERNAL_HISTORY
    ACTIVE_PROFILE_NAME = name
    ACTIVE_PROFILE = native_profiles.profile(name)
    EXTERNAL = ACTIVE_PROFILE.external_root
    EVIDENCE = ACTIVE_PROFILE.evidence_directory
    SUCCESSOR_GRANT = ACTIVE_PROFILE.envelope_path
    COORDINATOR_SNAPSHOT = ACTIVE_PROFILE.snapshot_path
    HISTORY = ACTIVE_PROFILE.history_path
    EXTERNAL_HISTORY = EXTERNAL / "history-native-starts.sqlite"
    return ACTIVE_PROFILE


def fixture_grant_bindings(original, successor, coordinator_snapshot, profile_name="r5"):
    """Build exact released file bindings and separate Request metadata for one closed profile."""
    selected = native_profiles.profile(profile_name)
    return {
        "releasedInputs": [dict(original), dict(successor), dict(coordinator_snapshot)],
        "nativeFixtureGrant": {
            **original, "sourceKey": "native-s1-integration-fixtures-20261008"},
        selected.marker: {**successor, "sourceKey": selected.key},
    }


def prepare():
    """Bind one selected closed-profile Request/Authority to its prepared fixture."""
    selected = ACTIVE_PROFILE
    base = EXTERNAL / "government"
    released = base / "released"
    preparation = json.loads((base / "preparation-final.json").read_bytes())
    role = json.loads((released / "role-auth.json").read_bytes())
    runtime_binding = binding(released / "runtime.json")
    authorization_binding = binding(released / "role-auth.json")
    actor = Path(preparation["actorRepository"])
    # prepare-native-r5 creates the parent directory; the outer controller gets
    # one fresh child directory so its evidence writer cannot reuse old bytes.
    evidence = base / "controller-evidence" / "outer-run"
    if evidence.exists():
        raise ValueError(f"{selected.name.upper()} outer controller evidence must be a fresh absent directory")
    outer_results = base / "outer-results"
    outer_results.mkdir(exist_ok=False)
    card = write_new(released / "task-card.txt",
        b"Public deterministic native protocol fixture only. Use the selected Government task and configured role slots. No model/provider, semantic or human acceptance claim.\n")
    mechanical = write_new(released / "mechanical_actor.py", (ROOT / "runtime/mechanical_actor.py").read_bytes())
    original = binding(SOURCE_GRANT)
    if original["sha256"] != R1_GRANT_SHA:
        raise ValueError(f"{selected.name.upper()} requires the exact immutable original R1 source grant")
    grants = fixture_grant_bindings(
        original, binding(SUCCESSOR_GRANT), binding(COORDINATOR_SNAPSHOT), selected.name)
    successor = grants[selected.marker]
    if successor["sha256"] != selected.envelope_sha:
        raise ValueError(f"{selected.name} envelope bytes differ from the exact released grant")
    product = preparation["productGovernment"]
    product, inputs = government_integration.product_binding(
        repository=actor, runtime_path=runtime_binding["path"],
        runtime_raw=Path(runtime_binding["path"]).read_bytes(),
        backlog_path=released / "backlog.json",
        backlog_raw=(released / "backlog.json").read_bytes(),
        authorization_path=authorization_binding["path"],
        authorization_raw=Path(authorization_binding["path"]).read_bytes(),
        queue_state_directory=product["queueStateDirectory"])
    common = json.loads((ROOT / "public/resource-proposal.json").read_bytes())["commonLimits"]
    released_inputs = inputs + grants["releasedInputs"] + [mechanical,
                                binding(EXTERNAL_HISTORY), role["diagnostics"], card,
                                binding(Path("C:/Python313/python.exe"))]
    request = {
        "schemaVersion": 1, "mode": "mechanical", "operation": "run_task",
        "trialId": role["trialId"], "dispatchId": selected.dispatch_id,
        "arm": "government", "condition": "brownfield",
        "actorRepository": str(actor.resolve()), "evidenceDirectory": str(evidence.resolve()),
        "limits": common, "wallSeconds": 38, "baseCommit": preparation["baseCommit"],
        "task": {"id": selected.task_id, "card": card}, "purpose": "task",
        "releasedInputs": released_inputs, "prompt": card,
        "mechanicalFixture": mechanical["path"],
        "nativeFixtureGrant": grants["nativeFixtureGrant"],
        "fixtureAuthorization": dict(government_roles.NATIVE_FIXTURE_AUTH),
        selected.marker: successor, "product": {"government": product}}
    request_binding = write_new(released / "request.json", request)
    runner_pin = dispatch.digest(dispatch.encoded(dispatch.mechanical_pin()))
    protocol = {"status": "frozen", "mode": "mechanical", "commonLimits": common,
                "runnerPinSha256": runner_pin, "runtimeSourceSha256": dispatch.runtime_pins(),
                "wrapperPythonSha256": dispatch.digest(Path(sys.executable).read_bytes()),
                "fixtureSourceGrant": original, selected.metadata_field: successor,
                "semanticAcceptance": False, "humanAcceptance": False}
    protocol_binding = write_new(base / "protocol.json", protocol)
    grant = {"schemaVersion": 1, "status": "approved", "purpose": "s1-mechanics",
             "mode": "mechanical", "trialId": role["trialId"], "notBefore": time.time() - 1,
             "expiresAt": role["expiresAt"], "protocolSha256": protocol_binding["sha256"],
             "profileSha256": dispatch.digest(dispatch.encoded(common)),
             "runnerPinSha256": runner_pin, "ledgerPath": role["ledgerPath"],
             "resultDirectory": str(outer_results.resolve()), "maxActorSessions": 6,
             "maxSessionWallSeconds": request["wallSeconds"], "retrospectiveTokenThreshold": 10000,
             "fixtureSourceGrant": original, selected.metadata_field: successor,
             "authorizedRequests": [{"dispatchId": request["dispatchId"],
                                     "executionSha256": dispatch.execution_sha(request),
                                     "initialRequestSha256": request_binding["sha256"]}]}
    grant_binding = write_new(base / "grant.json", grant)
    authority_binding = write_new(base / "authority.json",
                                  {"request": request_binding, "grant": grant_binding,
                                   "protocol": protocol_binding})
    return {"authority": authority_binding, "request": request_binding,
            "grant": grant_binding, "protocol": protocol_binding}


def freeze():
    selected = ACTIVE_PROFILE
    if subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT):
        raise ValueError(f"{selected.name} source must be committed before the final preflight freeze")
    if selected.name == "r5":
        validate()
    else:
        _require_actual_validation_receipt()
    _, request, _, _, _ = authority(validate_request=selected.name == "r5")
    paths = {Path(__file__), ROOT / "prepare-native-r5.py", SUCCESSOR_GRANT,
             COORDINATOR_SNAPSHOT, EXTERNAL_HISTORY, EVIDENCE / "independent-preflight-review.md",
             EVIDENCE / "host-success-contract.md", HISTORY, SOURCE_GRANT,
             Path("C:/Python313/python.exe"), ROOT / "public/resource-proposal.json",
             ROOT / "runtime/mechanical_actor.py", ROOT / "runtime/government_native_profile.py"}
    if selected.name in {"r6", "r7"}:
        paths.update({EVIDENCE / "actual-input-claim.json", EVIDENCE / "actual-input-validation.json"})
    if selected.name == "r6":
        paths.add(EVIDENCE / "independent-preflight-a1-addendum.md")
    if selected.name == "r7":
        paths.add(EVIDENCE / "offline-r7-tests.log")
    paths.update((EXTERNAL / "government").rglob("*.json"))
    paths.update(ROOT / "runtime" / name for name in dispatch.runtime_pins()
                 if (ROOT / "runtime" / name).is_file())
    paths.add(ROOT / "runtime/government_integration.py")
    runtime = json.loads(Path(request["product"]["government"]["runtime"]["path"]).read_bytes())
    for configured_role in government.configured_roles(runtime):
        paths.update(Path(item["path"]) for item in configured_role.get("runtimeFiles", []))
    paths.update(Path(item["path"]) for item in request["releasedInputs"] if isinstance(item, dict))
    paths.add(Path(request["product"]["government"]["executable"]["path"]))
    paths.add(Path(government_integration.FIXTURE_ROOT / "deterministic_delegate.py"))
    paths.update((EXTERNAL / "government/actor").rglob("*"))
    paths = {path for path in paths if path.is_file() and ".git" not in path.parts}
    result = {"grantKey": selected.key, "profile": selected.name,
              "status": "independently-reviewed-mechanical-fixture",
              "sourceCommit": subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
              "runtimeSourceSha256": dispatch.runtime_pins(),
              "inputs": [binding(path) for path in sorted(paths, key=str)],
              "nativeStarts": 0, "semanticAcceptance": False, "humanAcceptance": False}
    if selected.name in {"r6", "r7"}:
        result["actualInputValidation"] = {
            "claim": binding(EVIDENCE / "actual-input-claim.json"),
            "receipt": binding(EVIDENCE / "actual-input-validation.json")}
    return write_new(EVIDENCE / "preflight-freeze.json", result)


def write_new(path, value):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    raw = value if isinstance(value, bytes) else dispatch.encoded(value) + b"\n"
    with path.open("xb") as stream:
        stream.write(raw)
    return {"path": str(path.resolve()), "sha256": dispatch.digest(raw)}


def binding(path):
    path = Path(path).resolve(strict=True)
    return {"path": str(path), "sha256": dispatch.digest(path.read_bytes())}


def _capture_released_inputs(request):
    captured = {}
    for item in request.get("releasedInputs", []):
        if not isinstance(item, dict) or set(item) != {"path", "sha256"}:
            raise ValueError("released input must be an exact path/SHA file binding")
        path = Path(item["path"]).resolve(strict=True)
        raw = path.read_bytes()
        if dispatch.digest(raw) != item["sha256"]:
            raise ValueError("released input changed after successful one-shot profile admission")
        captured[item["path"]] = raw
    return captured


def authority(*, validate_request=None):
    selected = ACTIVE_PROFILE
    if validate_request is None:
        validate_request = selected.name == "r5" or not _r6_validation_receipt_exists()
    doc = json.loads((EXTERNAL / "government/authority.json").read_bytes())
    for item in doc.values():
        if binding(item["path"]) != item:
            raise ValueError(f"frozen {selected.name.upper()} authority entry changed")
    result_authority = dispatch.Authority(
        doc["grant"]["path"], doc["grant"]["sha256"],
        doc["protocol"]["path"], doc["protocol"]["sha256"])
    raw = Path(doc["request"]["path"]).read_bytes()
    if validate_request:
        request, captured = result_authority.validate(raw)
    else:
        request = json.loads(raw)
        captured = _capture_released_inputs(request)
    original = {key: request["nativeFixtureGrant"][key] for key in ("path", "sha256")}
    successor = request.get(selected.marker)
    if (original != {"path": str(SOURCE_GRANT.resolve(strict=True)),
                     "sha256": dispatch.digest(SOURCE_GRANT.read_bytes())} or
            result_authority.grant.get("fixtureSourceGrant") != original or
            result_authority.protocol.get("fixtureSourceGrant") != original or
            result_authority.grant.get(selected.metadata_field) != successor or
            result_authority.protocol.get(selected.metadata_field) != successor or
            successor != {**binding(SUCCESSOR_GRANT), "sourceKey": selected.key} or
            request.get("dispatchId") != selected.dispatch_id or
            request.get("arm") != "government" or request.get("operation") != "run_task" or
            result_authority.mode != "mechanical" or
            result_authority.grant.get("maxActorSessions") != 6 or
            request.get("task", {}).get("id") != selected.task_id or
            request.get("wallSeconds") != 38 or
            result_authority.grant.get("maxSessionWallSeconds") != 38):
        raise ValueError(f"compiled Authority differs from the exact {selected.name.upper()} Government-only grant")
    return result_authority, request, raw, captured, Path(doc["request"]["path"])


def _r6_validation_receipt_exists():
    return ACTIVE_PROFILE.name in {"r6", "r7"} and (EVIDENCE / "actual-input-claim.json").exists()


def _r6_input_bindings(auth, request, captured, request_path):
    base = EXTERNAL / "government"
    review_path = EVIDENCE / "independent-preflight-review.md"
    if not review_path.is_file():
        raise ValueError(f"{ACTIVE_PROFILE.name.upper()} independent preflight review is required before admission")
    paths = {Path(request_path), base / "authority.json", base / "grant.json", base / "protocol.json",
             SUCCESSOR_GRANT, COORDINATOR_SNAPSHOT, EXTERNAL_HISTORY, HISTORY, SOURCE_GRANT,
             Path("C:/Python313/python.exe"), ROOT / "prepare-native-r5.py",
             ROOT / "run-native-integration-r5.py", ROOT / "runtime/government_native_profile.py",
             ROOT / "runtime/mechanical_actor.py", ROOT / "public/resource-proposal.json",
             review_path,
             Path(request["product"]["government"]["executable"]["path"]),
             Path(government_integration.FIXTURE_ROOT / "deterministic_delegate.py")}
    for contract_name in r6_required_success_contracts(ACTIVE_PROFILE.name):
        contract_path = EVIDENCE / contract_name
        if not contract_path.is_file():
            raise ValueError(f"{ACTIVE_PROFILE.name.upper()} required success evidence is missing: {contract_path}")
        paths.add(contract_path)
    if ACTIVE_PROFILE.name == "r7":
        paths.add(EVIDENCE / "offline-r7-tests.log")
    paths.update(ROOT / "runtime" / name for name in dispatch.runtime_pins()
                 if (ROOT / "runtime" / name).is_file())
    paths.add(ROOT / "runtime/government_integration.py")
    runtime = json.loads(Path(request["product"]["government"]["runtime"]["path"]).read_bytes())
    for role in government.configured_roles(runtime):
        paths.update(Path(item["path"]) for item in role.get("runtimeFiles", []))
    paths.update(Path(path) for path in captured)
    paths.update((EXTERNAL / "government/actor").rglob("*"))
    return [binding(path) for path in sorted(
        {path for path in paths if path.is_file() and ".git" not in path.parts}, key=str)]


def r6_required_success_contracts(profile_name):
    """Return the closed success-contract set required for an R6/R7 profile."""
    if profile_name not in {"r6", "r7"}:
        return ()
    required = ["host-success-contract.md"]
    if profile_name == "r6":
        required.append("independent-preflight-a1-addendum.md")
    return tuple(required)


def _require_actual_validation_receipt():
    selected = ACTIVE_PROFILE
    if selected.name not in {"r6", "r7"}:
        raise ValueError("one-shot actual admission receipts apply only to R6/R7")
    claim_path = EVIDENCE / "actual-input-claim.json"
    receipt_path = EVIDENCE / "actual-input-validation.json"
    if not claim_path.is_file() or not receipt_path.is_file():
        raise ValueError(f"{selected.name.upper()} one-shot actual input validation is missing")
    claim = json.loads(claim_path.read_bytes())
    receipt = json.loads(receipt_path.read_bytes())
    if (claim.get("profile") != selected.name or claim.get("grantKey") != selected.key or
            receipt.get("status") != "passed" or receipt.get("profile") != selected.name or
            receipt.get("grantKey") != selected.key or
            receipt.get("sourceCommit") != claim.get("sourceCommit") or
            receipt.get("claimSha256") != binding(claim_path)["sha256"]):
        raise ValueError(f"{selected.name.upper()} actual input validation receipt is not a successful exact one-shot claim")
    source_commit = receipt.get("sourceCommit")
    if not isinstance(source_commit, str) or len(source_commit) != 40:
        raise ValueError(f"{selected.name.upper()} actual validation does not bind its clean source commit")
    subprocess.check_call(["git", "merge-base", "--is-ancestor", source_commit, "HEAD"], cwd=ROOT)
    subprocess.check_call(["git", "merge-base", "--is-ancestor", selected.base_sha, source_commit], cwd=ROOT)
    auth, request, raw, captured, request_path = authority(validate_request=False)
    if (receipt.get("request") != binding(request_path) or receipt.get("requestSha256") != dispatch.digest(raw) or
            receipt.get("authority") != binding(EXTERNAL / "government/authority.json") or
            receipt.get("protocol") != binding(EXTERNAL / "government/protocol.json") or
            receipt.get("grant") != binding(EXTERNAL / "government/grant.json") or
            receipt.get("profileEnvelope") != binding(SUCCESSOR_GRANT) or
            receipt.get("coordinatorSnapshot") != binding(COORDINATOR_SNAPSHOT) or
            receipt.get("runtimeSourceSha256") != dispatch.runtime_pins()):
        raise ValueError(f"{selected.name.upper()} actual validation receipt no longer binds its Request/Authority/source pins")
    expected = {item["path"]: item for item in receipt.get("preFreezeInputs", [])}
    for item in expected.values():
        if binding(item["path"]) != item:
            raise ValueError(f"{selected.name.upper()} pre-freeze actual validation input changed")
    required = _r6_input_bindings(auth, request, captured, request_path)
    if any(expected.get(item["path"]) != item for item in required):
        raise ValueError(f"{selected.name.upper()} actual validation receipt omits or differs from a required frozen input")
    return receipt


def _actual_r6_input_validation():
    selected = ACTIVE_PROFILE
    claim_path = EVIDENCE / "actual-input-claim.json"
    receipt_path = EVIDENCE / "actual-input-validation.json"
    if not r6_validation_claim_available(claim_path.exists(), receipt_path.exists()):
        raise ValueError(f"{selected.name.upper()} actual input validation is one-shot; existing claim closes it")
    status = subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT, text=True).splitlines()
    allowed = f"experiments/government-comparison/evidence/{selected.evidence_directory.name}/"
    if any(not line[3:].replace("\\", "/").startswith(allowed) for line in status):
        raise ValueError(f"{selected.name.upper()} actual validation requires the reviewed clean source commit")
    source_commit = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
    claim = write_new(claim_path, {"profile": selected.name, "grantKey": selected.key,
                                   "sourceCommit": source_commit, "claimedUtc": time.time()})
    receipt = {"profile": selected.name, "grantKey": selected.key, "claimSha256": claim["sha256"],
               "sourceCommit": source_commit, "status": "failed", "nativeStarts": 0,
               "wrapperAttempts": 0, "delegates": 0, "providerCalls": 0}
    try:
        auth, request, raw, captured, request_path = authority(validate_request=True)
        validator = getattr(native_budget, f"validate_{selected.name}_grant_binding")
        entry_gate = getattr(native_budget, f"validate_{selected.name}_entry_gate")
        validated = validator(request, SOURCE_GRANT, dispatch.digest(SOURCE_GRANT.read_bytes()))
        gate = entry_gate(validated)
        bound = government.bind_request(request)
        native_controller.validate_native_fixture_grant(request, captured, bound)
        role = request["product"]["government"]["roleAuthorization"]
        government_roles.preflight_authorization(
            request, raw, auth, captured, role["path"], Path(role["path"]).read_bytes(), role["sha256"])
        if (request.get("dispatchId") != selected.dispatch_id or request.get(selected.marker) is None or
                request.get("task", {}).get("id") != selected.task_id or
                bound["backlogValue"]["limits"]["maxParallelism"] > 2):
            raise ValueError(f"{selected.name.upper()} Request identity or native role parallelism differs from its closed profile")
        ledger = _fixture_budget(request).preflight_snapshot()
        if ledger.get("entryGate", {}).get("ready") is not True:
            raise ValueError(f"{selected.name.upper()} live slot is not ready during actual input validation")
        rows = ledger.get("starts")
        if not isinstance(rows, list) or len(rows) != selected.history_starts:
            raise ValueError(f"{selected.name.upper()} actual validation requires its exact immutable historical ledger prefix")
        bindings = _r6_input_bindings(auth, request, captured, request_path)
        receipt.update(status="passed", request=binding(request_path), requestSha256=dispatch.digest(raw),
                       authority=binding(EXTERNAL / "government/authority.json"),
                       protocol=binding(EXTERNAL / "government/protocol.json"),
                       grant=binding(EXTERNAL / "government/grant.json"),
                       profileEnvelope=binding(SUCCESSOR_GRANT),
                       coordinatorSnapshot=binding(COORDINATOR_SNAPSHOT),
                       validatedGrantSha256=validated["sha256"], sourceCoordinationSha256=validated["sourceCoordinationSha256"],
                       runtimeSourceSha256=dispatch.runtime_pins(), entryGate=gate,
                       ledgerRows=len(rows), preFreezeInputs=bindings)
    except Exception as exc:
        receipt["error"] = f"{type(exc).__name__}: {exc}"
        write_new(receipt_path, receipt)
        raise
    write_new(receipt_path, receipt)
    return receipt


def validate():
    if ACTIVE_PROFILE.name in {"r6", "r7"}:
        return _actual_r6_input_validation()
    auth, request, raw, captured, _ = authority()
    validated = validate_r5_grant_binding(request, SOURCE_GRANT, dispatch.digest(SOURCE_GRANT.read_bytes()))
    validate_r5_entry_gate(validated)
    bound = government.bind_request(request)
    native_controller.validate_native_fixture_grant(request, captured, bound)
    role = request["product"]["government"]["roleAuthorization"]
    government_roles.preflight_authorization(
        request, raw, auth, captured, role["path"], Path(role["path"]).read_bytes(), role["sha256"])
    if bound["backlogValue"]["limits"]["maxParallelism"] > 2:
        raise ValueError("R5 Government native parallelism exceeds its grant")
    ledger = _fixture_budget(request).preflight_snapshot()
    if ledger.get("entryGate", {}).get("ready") is not True:
        raise ValueError("R5 live slot assignment is not ready for a native reservation")
    starts = ledger.get("starts")
    if not isinstance(starts, list):
        raise ValueError("R5 immutable native-start ledger snapshot is malformed")
    r5_labels = [row.get("label") for row in starts
                 if isinstance(row, dict) and row.get("label") in {
                     "government-native-scope-r5/queue", "government-native-scope-r5/resume"}]
    if (r5_labels not in ([], ["government-native-scope-r5/queue"],
                          ["government-native-scope-r5/queue", "government-native-scope-r5/resume"]) or
            len(starts) != 12 + len(r5_labels)):
        raise ValueError("R5 historical ledger prefix or sequential additive consumption is malformed")
    return {"arm": "government", "requestSha256": dispatch.digest(raw),
            "validated": True, "nativeStarts": 0, "providerCalls": 0,
            "sourcePins": dispatch.runtime_pins(),
            "ledgerRows": len(starts), "r5LedgerRows": len(r5_labels),
            "entryGate": ledger["entryGate"]}


def r6_validation_claim_available(claim_exists, receipt_exists):
    return claim_exists is False and receipt_exists is False


def r6_resume_authorized(outer_status, native_queue_status, translated_status):
    return (outer_status == "completed" and native_queue_status == "complete" and
            translated_status == "completed")


def require_freeze():
    selected = ACTIVE_PROFILE
    changes = subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT, text=True).splitlines()
    allowed = f"experiments/government-comparison/evidence/{selected.evidence_directory.name}/"
    if any(not line[3:].replace("\\", "/").startswith(allowed) for line in changes):
        raise ValueError(f"{selected.name.upper()} source and non-profile evidence must remain clean")
    if selected.name in {"r6", "r7"}:
        _require_actual_validation_receipt()
    else:
        validate()
    freeze_path = EVIDENCE / "preflight-freeze.json"
    freeze = json.loads(freeze_path.read_bytes())
    committed = subprocess.check_output([
        "git", "show", f"HEAD:experiments/government-comparison/evidence/{selected.evidence_directory.name}/preflight-freeze.json"], cwd=ROOT)
    if committed != freeze_path.read_bytes():
        raise ValueError(f"{selected.name.upper()} freeze must be committed byte-identically")
    if freeze.get("status") != "independently-reviewed-mechanical-fixture":
        raise ValueError(f"independent final {selected.name.upper()} preflight review is missing")
    if (freeze.get("grantKey") != selected.key or freeze.get("profile") != selected.name or
            freeze.get("runtimeSourceSha256") != dispatch.runtime_pins()):
        raise ValueError(f"{selected.name.upper()} source/grant freeze differs from the admitted candidate")
    source_commit = freeze.get("sourceCommit")
    if not isinstance(source_commit, str) or len(source_commit) != 40:
        raise ValueError(f"{selected.name.upper()} freeze lacks its exact source commit")
    subprocess.check_call(["git", "merge-base", "--is-ancestor", source_commit, "HEAD"], cwd=ROOT)
    if selected.name in {"r6", "r7"}:
        changed = subprocess.check_output(["git", "diff", "--name-only", source_commit, "HEAD"],
                                          cwd=ROOT, text=True).splitlines()
        if not r6_source_changes_confined(changed, EVIDENCE):
            raise ValueError(f"{selected.name.upper()} committed source changed outside its frozen evidence directory")
    expected = {str(Path(item["path"]).resolve()): item for item in freeze.get("inputs", [])}
    required = [Path(__file__), ROOT / "prepare-native-r5.py", EVIDENCE / "independent-preflight-review.md",
                SUCCESSOR_GRANT, COORDINATOR_SNAPSHOT,
                EXTERNAL_HISTORY, HISTORY, SOURCE_GRANT, Path("C:/Python313/python.exe"),
                ROOT / "public/resource-proposal.json", ROOT / "runtime/mechanical_actor.py"]
    required += [EVIDENCE / name for name in r6_required_success_contracts(selected.name)]
    if selected.name == "r7":
        required.append(EVIDENCE / "offline-r7-tests.log")
    required += [EXTERNAL / "government" / name for name in
                 ("authority.json", "grant.json", "protocol.json", "released/request.json")]
    for path in required:
        if str(path.resolve(strict=True)) not in expected:
            raise ValueError(f"required final {selected.name.upper()} input is absent from the freeze: {path.name}")
    for item in expected.values():
        if binding(item["path"]) != item:
            raise ValueError(f"frozen {selected.name.upper()} input changed before native start")
    if selected.name in {"r6", "r7"}:
        validation = _require_actual_validation_receipt()
        receipt_bindings = freeze.get("actualInputValidation", {})
        if (receipt_bindings.get("claim") != binding(EVIDENCE / "actual-input-claim.json") or
                receipt_bindings.get("receipt") != binding(EVIDENCE / "actual-input-validation.json") or
                validation.get("status") != "passed"):
            raise ValueError(f"{selected.name.upper()} freeze does not bind the exact successful one-shot actual validation receipt")


def _check_report_checks(report, runtime):
    expected = runtime.get("checks")
    observed = report.get("checks")
    if (not isinstance(expected, list) or not expected or not isinstance(observed, list) or
            len(expected) != len(observed)):
        raise ValueError("fresh configured Government checks are missing")
    by_key = {}
    for definition in expected:
        if not isinstance(definition, dict) or not isinstance(definition.get("name"), str) or not definition.get("name"):
            raise ValueError("configured fresh check definition is malformed")
        command = definition.get("run")
        tool = definition.get("tool")
        if tool is None and isinstance(command, list) and command:
            tool = command[0]
        timeout_seconds = definition.get("timeoutSeconds", 30)
        if (not isinstance(tool, str) or not tool or type(timeout_seconds) is not int or
                timeout_seconds <= 0):
            raise ValueError("configured fresh check tool or timeout is malformed")
        key = (definition["name"], tool)
        if key in by_key:
            raise ValueError("configured fresh check identity is duplicated")
        by_key[key] = definition
    seen = set()
    for item in observed:
        if not isinstance(item, dict):
            raise ValueError("fresh native GateResult is malformed")
        key = (item.get("Name"), item.get("Tool"))
        definition = by_key.get(key)
        if definition is None or key in seen:
            raise ValueError("fresh native check names/tools differ from the exact configured set")
        timeout_ms = definition.get("timeoutSeconds", 30) * 1000
        if (type(item.get("ExitCode")) is not int or item["ExitCode"] != 0 or
                type(item.get("Milliseconds")) is not int or item["Milliseconds"] < 0 or
                type(item.get("TimeoutMilliseconds")) is not int or
                item["TimeoutMilliseconds"] != timeout_ms or
                item["Milliseconds"] > timeout_ms):
            raise ValueError("fresh native check exit/time receipt is missing or outside its configured bound")
        seen.add(key)
    if seen != set(by_key):
        raise ValueError("fresh native check coverage is incomplete")


def _controller_elapsed(auth, request, process):
    runs = [item for item in auth.ledger().snapshot()["controllerRuns"]
            if item.get("dispatch_id") == request["dispatchId"]]
    if len(runs) != 1 or runs[0].get("end") is None or runs[0].get("status") != "completed":
        raise ValueError("completed Queue requires its exact terminal controller ledger receipt")
    raw_receipt = runs[0].get("process_receipt")
    if not isinstance(raw_receipt, str):
        raise ValueError("terminal controller receipt is missing")
    receipt = json.loads(raw_receipt)
    elapsed = receipt.get("elapsedSeconds") if isinstance(receipt, dict) else None
    if (not isinstance(receipt, dict) or receipt.get("process") != process or
            type(elapsed) not in (int, float) or elapsed < 0 or elapsed > 38 or
            process.get("stopReason") is not None):
        raise ValueError("completed Queue controller receipt is invalid or exceeds the 38-second deadline")
    return elapsed


def r6_queue_completion_is_positive(receipt, request_sha, process_binding,
                                    driver_elapsed_seconds, profile_name="r6"):
    """Pure predicate for the bridge's immutable terminal Queue receipt plus driver window."""
    if not isinstance(receipt, dict):
        return False
    terminal = receipt.get("terminalResult")
    native = receipt.get("nativeProcess")
    controller_elapsed = receipt.get("elapsedSecondsAfterTerminalReceiptAvailable")
    controller_receipt_sha = receipt.get("terminalControllerReceiptSha256")
    return (receipt.get("status") == "completed" and
            receipt.get("dispatchId") == native_profiles.profile(profile_name).dispatch_id and
            receipt.get("requestSha256") == request_sha and
            isinstance(terminal, dict) and isinstance(terminal.get("sha256"), str) and
            len(terminal["sha256"]) == 64 and
            isinstance(controller_receipt_sha, str) and len(controller_receipt_sha) == 64 and
            all(character in "0123456789abcdef" for character in controller_receipt_sha) and
            native == process_binding and
            type(controller_elapsed) in (int, float) and 0 <= controller_elapsed <= 38 and
            type(driver_elapsed_seconds) in (int, float) and 0 <= driver_elapsed_seconds <= 38)


def r6_source_changes_confined(changed_paths, evidence_directory):
    prefix = (Path("experiments/government-comparison/evidence") /
              Path(evidence_directory).name).as_posix() + "/"
    return all(Path(path).as_posix().startswith(prefix) for path in changed_paths)


def r6_resume_completion_is_positive(completion, process, result_status,
                                    driver_elapsed_after_receipt_readback, profile_name=None):
    if not isinstance(completion, dict) or not isinstance(process, dict):
        return False
    elapsed = completion.get("elapsedSecondsAfterTerminalReceiptAvailable")
    wall = process.get("wallSeconds")
    return (completion.get("profile") in {"r6", "r7"} and
            (profile_name is None or completion.get("profile") == profile_name) and
            completion.get("status") == "completed" and
            result_status == "completed" and process.get("returnCode") == 0 and
            process.get("stopReason") is None and type(wall) in (int, float) and 0 <= wall <= 38 and
            type(elapsed) in (int, float) and 0 <= elapsed <= 38 and
            type(driver_elapsed_after_receipt_readback) in (int, float) and
            0 <= driver_elapsed_after_receipt_readback <= 38)


def _r6_queue_completion(auth, request, raw, process_path):
    evidence = Path(request["evidenceDirectory"])
    receipt_path = evidence / "controller-completion.json"
    receipt_raw = receipt_path.read_bytes()
    receipt = json.loads(receipt_raw)
    process_raw = process_path.read_bytes()
    process_binding = {"path": str(process_path.resolve()), "sha256": dispatch.digest(process_raw)}
    result_path = auth.result_directory / f"{ACTIVE_PROFILE.name}-queue.json"
    result_raw = result_path.read_bytes()
    if receipt.get("terminalResult") != {"path": str(result_path.resolve()),
                                         "sha256": dispatch.digest(result_raw)}:
        raise ValueError(f"{ACTIVE_PROFILE.name.upper()} Queue completion receipt does not bind the exact outer terminal result")
    driver_elapsed = time.monotonic() - _PROFILE_QUEUE_STARTED
    if not r6_queue_completion_is_positive(receipt, dispatch.digest(raw),
                                           process_binding, driver_elapsed, ACTIVE_PROFILE.name):
        raise ValueError(f"{ACTIVE_PROFILE.name.upper()} Queue terminal/driver completion receipt is missing, late, or mismatched")
    return {"path": str(receipt_path.resolve()), "sha256": dispatch.digest(receipt_raw),
            "receipt": receipt, "driverElapsedSeconds": driver_elapsed}


def positive_checkpoint(process, queue, report, runtime):
    """Require full native acceptance and its exact positive completion checkpoint."""
    if (process.get("returnCode") != 0 or process.get("stopReason") is not None or
            type(process.get("wallSeconds")) not in (int, float) or
            process["wallSeconds"] > 38 or
            type(process.get("controllerElapsedSeconds")) not in (int, float) or
            process["controllerElapsedSeconds"] > 38 or
            queue.get("status") != "complete" or queue.get("nextStep") != "none" or
            queue.get("error") or queue.get("inFlightActors") != 0):
        raise ValueError("Queue/process is not completely positive")
    jobs = queue.get("jobs", [])
    if (len(jobs) != 1 or jobs[0].get("id") != ACTIVE_PROFILE.task_id or
            jobs[0].get("state") != "accepted-scoped" or jobs[0].get("nextStep") != "complete" or
            jobs[0].get("resultStatus") not in (None, "completed")):
        raise ValueError("Queue job/result is not accepted-scoped and complete")
    if (report.get("status") != "accepted-scoped" or report.get("stage") != "complete" or
            report.get("error") or not report.get("candidateCommit") or not report.get("candidateTree")):
        raise ValueError("run result is not accepted-scoped/complete")
    _check_report_checks(report, runtime)
    promotion = report.get("promotion", {})
    if (promotion.get("status") != "promoted" or
            promotion.get("expectedOld") != runtime["expectedBase"] or
            promotion.get("newCommit") != report["candidateCommit"] or
            promotion.get("actualActive") != report["candidateCommit"] or
            not promotion.get("intentPath") or not promotion.get("completionPath")):
        raise ValueError("exact promotion and durable completion are required")
    roles = government.configured_roles(runtime)
    government._validate_run_report(
        report, report["runId"],
        {role["slotId"]: (role["phase"], role["responseRole"]) for role in roles},
        require_acceptance=True,
        root_review_slots={role["slotId"] for role in roles
                           if role["phase"] == "review" and not role.get("area")})
    return {"candidateCommit": report["candidateCommit"],
            "candidateTree": report["candidateTree"],
            "materialCandidateId": report["evidence"]["materialCandidateId"],
            "evidenceId": report["evidence"]["id"],
            "decisionId": report["decision"]["id"],
            "round": report["evidence"]["round"],
            "scopeIdentities": report["plan"]["integrationReviews"]}


def _fixture_budget(request):
    # The unchanged Classic packet is read only to preserve the original shared
    # allocation's binary map; this helper never launches the Classic product.
    classic_binary = classic.inspect_packet(classic_integration.PACKET)["binary"]["path"]
    return FixtureBudget(LIVE_HISTORY, SOURCE_GRANT, R1_GRANT_SHA,
                         {"government": government.PIN["accepted"]["binary"]["path"],
                          "classic": classic_binary}, request=request)


def _promotion_receipts(result):
    receipts = result.get("receipts") if isinstance(result, dict) else None
    if not isinstance(receipts, list):
        return None
    output = {}
    for kind in ("government-promotion-intent", "government-promotion-completion"):
        selected = [(item.get("path"), item.get("sha256")) for item in receipts
                    if isinstance(item, dict) and item.get("kind") == kind]
        if len(selected) != 1:
            return None
        output[kind] = selected
    return output


def r5_resume_within_controller_deadline(elapsed_seconds):
    """Pure terminal predicate for the complete Resume controller window."""
    return (type(elapsed_seconds) in (int, float) and
            0 <= elapsed_seconds <= 38)


def r5_resume_work_window(elapsed_seconds):
    """Reserve cleanup time inside 38s and bound Resume work to at most 32s."""
    if type(elapsed_seconds) not in (int, float) or elapsed_seconds < 0:
        raise ValueError(f"{ACTIVE_PROFILE.name.upper()} Resume elapsed time must be finite and nonnegative")
    available = min(DEADLINE - 6, 38 - elapsed_seconds - 6)
    if available <= 0:
        raise ValueError(f"{ACTIVE_PROFILE.name.upper()} Resume leaves no work time after cleanup reserve")
    return available


def require_positive_queue():
    auth, request, raw, _, _ = authority()
    evidence = Path(request["evidenceDirectory"])
    process = json.loads((evidence / "process/process.json").read_bytes())
    if ACTIVE_PROFILE.name in {"r6", "r7"}:
        checkpoint_path = EVIDENCE / "government/queue-driver-completion.json"
        driver_checkpoint = json.loads(checkpoint_path.read_bytes())
        bridge_path = evidence / "controller-completion.json"
        bridge_raw = bridge_path.read_bytes()
        bridge = json.loads(bridge_raw)
        process_path = evidence / "process/process.json"
        process_raw = process_path.read_bytes()
        expected_bridge = driver_checkpoint.get("bridgeCompletion")
        if (not isinstance(expected_bridge, dict) or
                expected_bridge != {"path": str(bridge_path.resolve()), "sha256": dispatch.digest(bridge_raw)} or
                driver_checkpoint.get("requestSha256") != dispatch.digest(raw) or
                driver_checkpoint.get("process") != {"path": str(process_path.resolve()),
                                                       "sha256": dispatch.digest(process_raw)} or
                driver_checkpoint.get("profile") != ACTIVE_PROFILE.name or
                not r6_queue_completion_is_positive(
                    bridge, dispatch.digest(raw), driver_checkpoint["process"],
                    driver_checkpoint.get("driverElapsedSeconds"), ACTIVE_PROFILE.name)):
            raise ValueError(f"saved {ACTIVE_PROFILE.name.upper()} Queue completion checkpoint is absent, changed, or nonpositive")
        process["controllerElapsedSeconds"] = bridge["elapsedSecondsAfterTerminalReceiptAvailable"]
    else:
        process["controllerElapsedSeconds"] = _controller_elapsed(auth, request, process)
    translated = government.translate_queue_result(
        request, raw, evidence / "process/stdout.log", process["returnCode"])
    if translated.get("status") != "completed":
        raise ValueError("strict Queue Result translation is not completed")
    queue = json.loads((evidence / "process/stdout.log").read_bytes())
    job = queue["jobs"][0]
    report = json.loads(Path(job["reportPath"]).read_bytes())
    runtime = json.loads(Path(request["product"]["government"]["runtime"]["path"]).read_bytes())
    checkpoint = positive_checkpoint(process, queue, report, runtime)
    for key in ("intentPath", "completionPath"):
        record = json.loads(Path(report["promotion"][key]).read_bytes())
        intent = record.get("intent", {}) if key == "completionPath" else record
        if key == "completionPath" and record.get("observedRef") != report["candidateCommit"]:
            raise ValueError("promotion completion readback does not bind the accepted candidate")
        expected = {"expectedOld": runtime["expectedBase"], "newCommit": report["candidateCommit"],
                    "expectedTreeId": report["candidateTree"],
                    "materialCandidateId": checkpoint["materialCandidateId"],
                    "evidenceId": checkpoint["evidenceId"], "decisionId": checkpoint["decisionId"],
                    "activeRef": runtime["activeRef"], "idempotencyKey": report["runId"]}
        if any(intent.get(name) != value for name, value in expected.items()):
            raise ValueError("promotion intent/completion binding mismatch")
    active = subprocess.check_output(
        ["git", "-C", request["actorRepository"], "rev-parse", runtime["activeRef"]], text=True).strip()
    if active != report["candidateCommit"]:
        raise ValueError("promoted active ref readback mismatch")
    return {"checkpoint": checkpoint, "queue": queue,
            "translatedResult": translated, "runReport": binding(job["reportPath"]),
            "runtime": runtime}


def government_queue():
    require_freeze()
    auth, request, raw, captured, request_path = authority()
    argv = government.plan_request(request)["argv"]
    budget = _fixture_budget(request)
    selected = ACTIVE_PROFILE
    validated_grant = getattr(budget, selected.name)
    getattr(native_budget, f"validate_{selected.name}_entry_gate")(validated_grant)
    label = f"{selected.dispatch_id}/queue"
    global _PROFILE_QUEUE_STARTED
    if selected.name in {"r6", "r7"}:
        _PROFILE_QUEUE_STARTED = time.monotonic()
    budget.reserve("government", label, argv)
    result = dispatch.dispatch(request_path, auth.result_directory / f"{selected.name}-queue.json", auth)
    process_path = Path(request["evidenceDirectory"]) / "process/process.json"
    if process_path.exists():
        budget.finish("government", label,
                      json.loads(process_path.read_bytes()))
    if selected.name in {"r6", "r7"}:
        completion = _r6_queue_completion(auth, request, raw, process_path)
        process_raw = process_path.read_bytes()
        driver_checkpoint = {"profile": selected.name, "dispatchId": selected.dispatch_id,
                             "requestSha256": dispatch.digest(raw),
                             "process": {"path": str(process_path.resolve()),
                                         "sha256": dispatch.digest(process_raw)},
                             "bridgeCompletion": {"path": completion["path"],
                                                  "sha256": completion["sha256"]},
                             "driverElapsedSeconds": completion["driverElapsedSeconds"]}
        write_new(EVIDENCE / "government/queue-driver-completion.json", driver_checkpoint)
    write_new(EVIDENCE / "government/queue-result.json", result)
    snapshots = []
    for index, item in enumerate(result.get("receipts", [])):
        content = Path(item["path"]).read_bytes()
        if dispatch.digest(content) != item["sha256"]:
            raise ValueError(f"{ACTIVE_PROFILE.name.upper()} Queue receipt changed before archival snapshot")
        saved = write_new(EVIDENCE / "government/queue-checkpoint" / str(index), content)
        snapshots.append({"original": item, "snapshot": saved})
    write_new(EVIDENCE / "government/queue-checkpoint.json", snapshots)
    return result


def government_resume():
    require_freeze()
    auth, request, raw, captured, _ = authority()
    original_stdout = Path(request["evidenceDirectory"]) / "process/stdout.log"
    queue_value = json.loads(original_stdout.read_bytes())
    positive = require_positive_queue()
    queue_directory = Path(queue_value["queueDirectory"]).resolve(strict=True)
    if queue_directory.parent != Path(request["product"]["government"]["queueStateDirectory"]).resolve():
        raise ValueError("R5 Resume queue escaped its exact bound parent")
    argv = government.resume_argv(
        request["product"]["government"]["executable"]["path"], request["actorRepository"],
        request["product"]["government"]["backlog"]["path"], queue_directory)
    ledger = auth.ledger()
    before = ledger.snapshot()
    budget = _fixture_budget(request)
    controller_started = time.monotonic()
    label = f"{ACTIVE_PROFILE.dispatch_id}/resume"
    budget.reserve("government", label, argv)
    fresh, fresh_captured = auth.validate(raw)
    getattr(native_controller, f"validate_{ACTIVE_PROFILE.name}_entry_for_launch")(
        fresh, fresh_captured, argv)
    launch_gate_elapsed = time.monotonic() - controller_started
    work_window = r5_resume_work_window(launch_gate_elapsed)
    process = bounded(argv, request["actorRepository"],
                      Path(request["evidenceDirectory"]) / "resume-process", work_window,
                      env=native_controller.strip_bootstrap_environment(os.environ))
    budget.finish("government", label, process)
    resumed_path = Path(request["evidenceDirectory"]) / "resume-process/stdout.log"
    resumed_queue = json.loads(resumed_path.read_bytes())
    after = ledger.snapshot()
    if (type(process.get("wallSeconds")) not in (int, float) or process["wallSeconds"] > 38):
        raise ValueError(f"{ACTIVE_PROFILE.name.upper()} Resume exceeded its 38-second process deadline")
    replay_request = copy.deepcopy(request)
    replay_request["operation"] = "resume"
    replay_request["product"]["government"]["queueDirectory"] = str(queue_directory)
    replay_raw = dispatch.encoded(replay_request)
    translated = government.translate_queue_result(
        replay_request, replay_raw, resumed_path, process["returnCode"])
    same_report = binding(positive["runReport"]["path"]) == positive["runReport"]
    replay_promotions = _promotion_receipts(translated)
    queue_promotions = _promotion_receipts(positive["translatedResult"])
    result = {
        "process": process, "nativeResult": resumed_queue,
        "translatedResult": translated,
        "roleStartsBefore": before["actorSessions"], "roleStartsAfter": after["actorSessions"],
        "sameQueue": resumed_queue.get("queueDirectory") == queue_value.get("queueDirectory"),
        "sameNativeRoleStarts": resumed_queue.get("actorStarts") == queue_value.get("actorStarts"),
        "noAdditionalRoles": before["attempts"] == after["attempts"],
        "samePersistedQueue": resumed_queue == queue_value,
        "sameCandidateEvidenceDecision": all(
            resumed_queue.get("jobs", [{}])[0].get(key) == queue_value.get("jobs", [{}])[0].get(key)
            for key in ("runId", "reportPath", "reportDigest")),
        "sameBoundRunReport": same_report,
        "samePromotionReceipts": (replay_promotions is not None and queue_promotions is not None and
                                  replay_promotions == queue_promotions),
        "semanticAcceptance": False, "humanAcceptance": False}
    active = subprocess.check_output(
        ["git", "-C", request["actorRepository"], "rev-parse", positive["runtime"]["activeRef"]],
        text=True).strip()
    result["samePromotionReadback"] = active == positive["checkpoint"]["candidateCommit"]
    controller_elapsed = time.monotonic() - controller_started
    result["controllerElapsedSeconds"] = controller_elapsed
    result["workWindowSeconds"] = work_window
    result["controllerDeadlineSatisfied"] = r5_resume_within_controller_deadline(controller_elapsed)
    result["passed"] = (
        process["returnCode"] == 0 and process["stopReason"] is None and
        translated.get("status") == "completed" and
        result["controllerDeadlineSatisfied"] and
        all(result[key] for key in ("sameQueue", "sameNativeRoleStarts", "noAdditionalRoles",
                                    "samePersistedQueue", "sameCandidateEvidenceDecision",
                                    "sameBoundRunReport", "samePromotionReceipts",
                                    "samePromotionReadback")))
    resume_result = write_new(EVIDENCE / "government/resume-result.json", result)
    if ACTIVE_PROFILE.name in {"r6", "r7"}:
        if binding(resume_result["path"]) != resume_result:
            raise ValueError(f"{ACTIVE_PROFILE.name.upper()} Resume result failed exact pre-completion readback")
        elapsed_after_result_readback = time.monotonic() - controller_started
        result_path = Path(request["evidenceDirectory"]) / "resume-process/process.json"
        process_binding = binding(result_path)
        result["completion"] = {"profile": ACTIVE_PROFILE.name, "dispatchId": ACTIVE_PROFILE.dispatch_id,
                                "requestSha256": dispatch.digest(raw), "process": process_binding,
                                "resumeResult": resume_result,
                                "elapsedSecondsAfterTerminalReceiptAvailable": elapsed_after_result_readback,
                                "status": "completed" if elapsed_after_result_readback <= 38 else "incomplete"}
        completion_binding = write_new(EVIDENCE / "government/resume-completion.json", result["completion"])
        if binding(completion_binding["path"]) != completion_binding:
            raise ValueError(f"{ACTIVE_PROFILE.name.upper()} Resume completion receipt failed exact readback")
        elapsed_after_completion_readback = time.monotonic() - controller_started
        result["controllerElapsedSeconds"] = elapsed_after_result_readback
        result["completionReceiptAvailableElapsedSeconds"] = elapsed_after_completion_readback
        result["controllerDeadlineSatisfied"] = r6_resume_completion_is_positive(
            result["completion"], process, translated.get("status"), elapsed_after_completion_readback,
            ACTIVE_PROFILE.name)
        result["passed"] = result["passed"] and result["controllerDeadlineSatisfied"]
        result["completionReceipt"] = completion_binding
        result_path_binding = write_new(EVIDENCE / "government/resume-final-result.json", result)
        if binding(result_path_binding["path"]) != result_path_binding:
            raise ValueError(f"{ACTIVE_PROFILE.name.upper()} Resume final result failed exact readback")
    return result


def run_once():
    require_freeze()
    selected = ACTIVE_PROFILE
    write_new(EXTERNAL / "case-claimed.json", {"grantKey": selected.key, "claimedUtc": time.time()})
    outcome = {"grantKey": selected.key, "status": "incomplete", "noRetry": True,
               "semanticAcceptance": False, "humanAcceptance": False, "s1Complete": False}
    try:
        write_new(EVIDENCE / "coordinator-start-snapshot.json", COORDINATION.read_bytes())
        queue_result = government_queue()
        outcome["outerQueueStatus"] = queue_result.get("status")
        if queue_result.get("status") == "completed":
            outcome["positiveQueue"] = require_positive_queue()
            if (selected.name in {"r6", "r7"} and not r6_resume_authorized(
                    queue_result.get("status"), outcome["positiveQueue"]["queue"].get("status"),
                    outcome["positiveQueue"]["translatedResult"].get("status"))):
                raise ValueError(f"{selected.name.upper()} Resume is not authorized by a fully positive native and outer Queue")
            resume = government_resume()
            outcome["resumePassed"] = resume["passed"]
            if resume["passed"]:
                outcome["status"] = "bounded-mechanics-passed"
    except Exception as exc:
        outcome["error"] = f"{type(exc).__name__}: {exc}"
    finally:
        write_new(EVIDENCE / "terminal-result.json", outcome)
    return outcome


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--profile", choices=("r5", "r6", "r7"), default="r5")
    parser.add_argument("action", choices=("prepare", "validate", "freeze", "run-once"))
    args = parser.parse_args()
    select_profile(args.profile)
    print(json.dumps({"prepare": prepare, "validate": validate, "freeze": freeze,
                      "run-once": run_once}[args.action](), sort_keys=True))
