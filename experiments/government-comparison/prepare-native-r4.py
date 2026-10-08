"""Prepare one fresh public Government R4 fixture; never start product roles.

The root coordinator must wait for the final runtime/source freeze and then run
this script with --final-runtime-ready. It creates only the new external R4
Government subtree, a disposable Git repository, and static released inputs.
It does not invoke Markitect, the role wrapper, the deterministic delegate, or
any model/provider. Git is used only to author and bind the fixture baseline.
"""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import sys
import time

RUNTIME_ROOT = Path(__file__).resolve().parent / "runtime"
sys.path.insert(0, str(RUNTIME_ROOT))
import government_integration as gi


EXTERNAL_ROOT = Path(
    r"C:\Users\Consiliari\Documents\Scientist-Probes\native-government-serialization-20261008-r4"
)
OUTPUT_ROOT = EXTERNAL_ROOT / "government"
GRANT_PATH = EXTERNAL_ROOT / "released-r4-grant.json"
R3_ROOT = Path(
    r"C:\Users\Consiliari\Documents\Scientist-Probes\native-metadata-fixtures-20261008-r3\government"
)
R1_ROOT = Path(
    r"C:\Users\Consiliari\Documents\Scientist-Probes\native-metadata-fixtures-20261008\government"
)
HISTORICAL_PREPARATION = R1_ROOT / "preparation-final.json"
EXPECTED_GRANT_SHA256 = "e4991c497c1ce086ee9a8bd7708409a6ec3e0de8bd114104eaef319df018e03c"
EXPECTED_DELEGATE_SHA256 = "e8b8e5087f994a975efc2228301cf7f51dee9de4d64b77077a0941db7a8e98b9"
EXPECTED_GOVERNMENT_CONFIG_SHA256 = "816041553684dc5710baf63217d435f62e9f997d1bbba4b74d9a2c4803da4325"
RETAINED_CONSTITUTION_DIGEST = "sha256:c1d7194bd6a8961001c68cbace3f7da40d5daf1b22570026d097e6091cf63e93"
R4_SOURCE_KEY = "government-serialization-native-20261008-r4"
R4_TRIAL_ID = "government-serialization-native-20261008-r4"
R4_DISPATCH_ID = "government-native-serialization-r4"
R4_AUTHORIZATION = gi.FIXTURE_AUTHORIZATION  # R1 fixture provenance remains fixed.


def _sha(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def _write_new(path: Path, raw: bytes) -> None:
    if path.exists() or path.is_symlink():
        raise FileExistsError(f"refusing to overwrite R4 preparation output: {path}")
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(raw)


def _validate_grant() -> dict:
    if GRANT_PATH.is_symlink() or not GRANT_PATH.is_file():
        raise FileNotFoundError(f"exact external R4 grant required at {GRANT_PATH}")
    raw = GRANT_PATH.read_bytes()
    if _sha(GRANT_PATH) != EXPECTED_GRANT_SHA256:
        raise ValueError("external R4 grant SHA-256 differs from the parent binding")
    envelope = json.loads(raw)
    grant = envelope.get("grant")
    expected_limits = {
        "key": R4_SOURCE_KEY,
        "maxFreshCases": 1,
        "maxNativeStarts": 2,
        "maxWrapperAttempts": 6,
        "maxDeterministicDelegates": 6,
        "maxParallelRoles": 2,
        "maxNewReservedSessionSeconds": 300,
        "controllerWindowSeconds": 38,
        "nativeProcessDeadlineSeconds": 38,
        "newModelProviderCalls": 0,
        "classicNativeStarts": 0,
        "fullProductSuites": 0,
        "metadataSessions": 0,
    }
    if not isinstance(grant, dict) or any(grant.get(key) != value for key, value in expected_limits.items()):
        raise ValueError("external R4 grant is not the exact one-case Government allocation")
    product = grant.get("product", {})
    accepted = gi.government.PIN["accepted"]
    if (product.get("name") != "Government" or
            product.get("sourceSha") != accepted["sourceCommit"] or
            product.get("binarySha256") != accepted["binary"]["sha256"] or
            product.get("delegateSha256") != EXPECTED_DELEGATE_SHA256):
        raise ValueError("R4 grant product/delegate pins differ from the held inputs")
    return envelope


def _historical_equivalence() -> dict:
    if not HISTORICAL_PREPARATION.is_file():
        raise FileNotFoundError("historical positive Government preparation is required")
    history = json.loads(HISTORICAL_PREPARATION.read_bytes())
    historical_config_sha = history["productGovernment"]["projectConfig"]["sha256"]
    if (history.get("constitutionDigest") != RETAINED_CONSTITUTION_DIGEST or
            historical_config_sha != EXPECTED_GOVERNMENT_CONFIG_SHA256):
        raise ValueError("historical inspect binding is not the expected config/digest pair")
    fixture_config_sha = _sha(gi.FIXTURE_ROOT / "government.yaml")
    old_actor = Path(history["actorRepository"])
    old_config_sha = _sha(old_actor / "government.yaml")
    if not (fixture_config_sha == old_config_sha == historical_config_sha):
        raise ValueError("Government Constitution input is not byte-equivalent to the historical inspect input")
    return {
        "historicalPreparation": str(HISTORICAL_PREPARATION),
        "historicalPreparationSha256": _sha(HISTORICAL_PREPARATION),
        "historicalActorConfigSha256": old_config_sha,
        "fixtureConfigSha256": fixture_config_sha,
        "retainedConstitutionDigest": RETAINED_CONSTITUTION_DIGEST,
    }


def prepare() -> dict:
    grant = _validate_grant()
    equivalence = _historical_equivalence()
    delegate_path = (gi.FIXTURE_ROOT / "deterministic_delegate.py").resolve(strict=True)
    if _sha(delegate_path) != EXPECTED_DELEGATE_SHA256:
        raise ValueError("corrected R4 deterministic delegate bytes are not present")

    # No historical fixture namespace is reused or modified.
    if OUTPUT_ROOT.exists() or OUTPUT_ROOT.is_symlink():
        raise FileExistsError(f"R4 Government output subtree already exists: {OUTPUT_ROOT}")
    OUTPUT_ROOT.mkdir(parents=True)
    actor = OUTPUT_ROOT / "actor"
    initial = gi.create_disposable_repository(actor)
    if _sha(actor / "government.yaml") != EXPECTED_GOVERNMENT_CONFIG_SHA256:
        raise ValueError("fresh repository config differs from the historical Constitution input")

    # Deliberately no G5 inspect is run. Reuse the digest only after the exact
    # Government config bytes have matched the retained inspect input above.
    bound = gi.bind_order_to_inspection(actor, RETAINED_CONSTITUTION_DIGEST)
    if bound["constitutionDigest"] != RETAINED_CONSTITUTION_DIGEST:
        raise AssertionError("fresh Order was not bound to the retained Constitution")

    released = OUTPUT_ROOT / "released"
    role_evidence = OUTPUT_ROOT / "role-evidence"
    controller_evidence = OUTPUT_ROOT / "controller-evidence"
    results = OUTPUT_ROOT / "results"
    queue_state = results / "queue-state"
    run_state = results / "run-state"
    temporary = results / "temporary"
    for path in (released, role_evidence, queue_state, run_state, temporary):
        path.mkdir(parents=True, exist_ok=False)
    request_path = released / "request.json"  # Root creates the frozen Request later.
    authorization_path = released / "role-auth.json"
    runtime_path = released / "runtime.json"
    backlog_path = released / "backlog.json"
    diagnostics_path = released / "wrapper-diagnostics-config.json"
    ledger_path = OUTPUT_ROOT / "fixture-ledger.sqlite"

    correction = {
        "path": str(GRANT_PATH.resolve(strict=True)),
        "sha256": EXPECTED_GRANT_SHA256,
        "sourceKey": R4_SOURCE_KEY,
    }
    diagnostic_config = {
        "arm": "government",
        "correction": correction,
        "requestPath": str(request_path.resolve()),
    }
    diagnostics_raw = gi.canonical_json(diagnostic_config) + b"\n"
    _write_new(diagnostics_path, diagnostics_raw)
    diagnostics_binding = {"path": str(diagnostics_path.resolve()), "sha256": _sha(diagnostics_path)}

    # The released R4 grant caps role calls at six. Reuse the helper's schema,
    # with the R4 numerical cap and controller wall deadline before building.
    gi.MAX_ROLE_CALLS = 6
    gi.MAX_WALL_SECONDS = 38
    authorization = gi.build_role_authorization(
        trial_id=R4_TRIAL_ID,
        dispatch_id=R4_DISPATCH_ID,
        request_path=request_path,
        ledger_path=ledger_path,
        runtime_path=runtime_path,
        role_evidence_directory=role_evidence,
        expires_at=int(time.time()) + 24 * 60 * 60,
        python_executable=sys.executable,
        delegate_path=delegate_path,
        status="approved",
        fixture_authorization=R4_AUTHORIZATION,
    )
    authorization["diagnostics"] = diagnostics_binding
    authorization_raw = gi.canonical_json(authorization) + b"\n"
    _write_new(authorization_path, authorization_raw)

    runtime = gi.build_runtime(
        repository=actor,
        base_commit=bound["baseCommit"],
        queue_state_directory=queue_state,
        run_state_directory=run_state,
        temporary_directory=temporary,
        runtime_path=runtime_path,
        authorization_path=authorization_path,
        authorization_raw=authorization_raw,
        python_executable=sys.executable,
        delegate_path=delegate_path,
        expected_constitution_digest=RETAINED_CONSTITUTION_DIGEST,
    )
    diagnostics_file = gi.runtime_file(diagnostics_path)
    for slot in (runtime["executor"], runtime["verifier"], runtime["ressorts"][0]["runner"]):
        slot["args"].extend([
            "--diagnostics-config", diagnostics_binding["path"],
            "--diagnostics-sha256", diagnostics_binding["sha256"],
        ])
        runtime_files = {item["path"]: item for item in slot["runtimeFiles"]}
        runtime_files[diagnostics_file["path"]] = diagnostics_file
        slot["runtimeFiles"] = [runtime_files[path] for path in sorted(runtime_files)]
    runtime_raw = gi.canonical_json(runtime) + b"\n"
    _write_new(runtime_path, runtime_raw)

    backlog = gi.build_backlog(runtime_path=runtime_path, queue_state_directory=queue_state)
    backlog_raw = gi.canonical_json(backlog) + b"\n"
    _write_new(backlog_path, backlog_raw)

    product, released_inputs = gi.product_binding(
        repository=actor,
        runtime_path=runtime_path,
        runtime_raw=runtime_raw,
        backlog_path=backlog_path,
        backlog_raw=backlog_raw,
        authorization_path=authorization_path,
        authorization_raw=authorization_raw,
        queue_state_directory=queue_state,
    )
    diagnostics_release = {"path": diagnostics_binding["path"], "sha256": diagnostics_binding["sha256"]}
    r4_release = {"path": correction["path"], "sha256": correction["sha256"]}
    released_inputs.extend([diagnostics_release, r4_release])

    role_slots = gi.government.configured_roles(runtime)
    runtime_files = runtime["executor"]["runtimeFiles"]
    for item in runtime_files:
        if gi.runtime_file(item["path"]) != item:
            raise ValueError("a runtime source changed while R4 preparation was frozen")
    manifest = {
        "apiVersion": "markitect.scientist-government-native-fixture-preparation/v1alpha1",
        "fixtureId": "government-native-serialization-r4",
        "actorRepository": str(actor.resolve()),
        "initialCommit": initial["baseCommit"],
        "baseCommit": bound["baseCommit"],
        "managedRef": gi.MANAGED_REF,
        "constitutionDigest": RETAINED_CONSTITUTION_DIGEST,
        "orderSha256": bound["orderSha256"],
        "inputEquivalence": equivalence,
        "trialId": R4_TRIAL_ID,
        "dispatchId": R4_DISPATCH_ID,
        "taskId": gi.TASK_ID,
        "roleAuthorization": {"path": str(authorization_path.resolve()), "sha256": _sha(authorization_path)},
        "diagnostics": {**diagnostics_binding, "correction": correction},
        "runtime": {"path": str(runtime_path.resolve()), "sha256": _sha(runtime_path)},
        "backlog": {"path": str(backlog_path.resolve()), "sha256": _sha(backlog_path)},
        "nativeFixtureR4Grant": correction,
        "productGovernment": product,
        "releasedInputs": released_inputs,
        "roleSlots": [{"slotId": item["slotId"], "phase": item["phase"],
                       "responseRole": item["responseRole"]} for item in role_slots],
        "limits": {
            "nativeStartsMaximum": 2,
            "wrapperAttemptsMaximum": 6,
            "deterministicDelegatesMaximum": 6,
            "maxParallelRoles": 2,
            "maxWallTimeSeconds": 38,
            "reservedSecondsPerNativeStart": 150,
            "maxNewReservedSessionSeconds": 300,
            "realActorStarts": 0,
            "providerCalls": 0,
            "studyCells": 0,
        },
        "plannedSequence": [
            "one fresh Government Queue",
            "only after complete positive Queue: its associated Resume/Replay verification",
        ],
        "expectedQueueRoleInvocations": 3,
        "resumeExpectedAdditionalRoleInvocations": 0,
        "runtimeSourceFiles": runtime_files,
        "productStartsDuringPreparation": 0,
        "outerRequest": "Root creates and freezes Request/Authority; request.json is intentionally not authored here.",
    }
    manifest_path = OUTPUT_ROOT / "preparation-final.json"
    _write_new(manifest_path, gi.canonical_json(manifest) + b"\n")
    return {
        "preparationPath": str(manifest_path),
        "preparationSha256": _sha(manifest_path),
        "actorRepository": str(actor),
        "baseCommit": bound["baseCommit"],
        "roleAuthorizationSha256": _sha(authorization_path),
        "runtimeSha256": _sha(runtime_path),
        "backlogSha256": _sha(backlog_path),
        "diagnosticsSha256": _sha(diagnostics_path),
        "grantSha256": EXPECTED_GRANT_SHA256,
        "runtimeFileCount": len(runtime_files),
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--final-runtime-ready", action="store_true", required=True,
        help="explicit root signal: runtime/budget/diagnostic source closure is frozen",
    )
    args = parser.parse_args()
    if not args.final_runtime_ready:
        parser.error("wait until Root confirms the final R4 runtime source freeze")
    result = prepare()
    print(json.dumps(result, sort_keys=True, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
