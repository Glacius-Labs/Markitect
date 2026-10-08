"""Prepare one fresh public Government R5, R6 or R7 fixture; never start product roles.

The root coordinator must wait for the final runtime/source freeze and then run
this script with --final-runtime-ready. It creates only the selected external
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
from government_native_profile import profile


EXTERNAL_ROOT = Path(
    r"C:\Users\Consiliari\Documents\Scientist-Probes\native-government-scope-20261008-r5"
)
OUTPUT_ROOT = EXTERNAL_ROOT / "government"
GRANT_PATH = EXTERNAL_ROOT / "released-r5-grant.json"
R1_ROOT = Path(
    r"C:\Users\Consiliari\Documents\Scientist-Probes\native-metadata-fixtures-20261008\government"
)
HISTORICAL_PREPARATION = R1_ROOT / "preparation-final.json"
EXPECTED_GRANT_SHA256 = "c80edc0e9864c1e332eaee81838dad0f33640aa96b226d30d14228375c493c90"
EXPECTED_DELEGATE_SHA256 = "e8b8e5087f994a975efc2228301cf7f51dee9de4d64b77077a0941db7a8e98b9"
EXPECTED_GOVERNMENT_CONFIG_SHA256 = "816041553684dc5710baf63217d435f62e9f997d1bbba4b74d9a2c4803da4325"
RETAINED_CONSTITUTION_DIGEST = "sha256:c1d7194bd6a8961001c68cbace3f7da40d5daf1b22570026d097e6091cf63e93"
R5_SOURCE_KEY = "government-scope-native-20261008-r5"
R5_TRIAL_ID = "government-scope-native-20261008-r5"
R5_DISPATCH_ID = "government-native-scope-r5"
R5_AUTHORIZATION = gi.FIXTURE_AUTHORIZATION  # R1 fixture provenance remains fixed.


def _sha(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def _write_new(path: Path, raw: bytes) -> None:
    if path.exists() or path.is_symlink():
        raise FileExistsError(f"refusing to overwrite Government preparation output: {path}")
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(raw)


def _validate_grant(selected) -> dict:
    grant_path = selected.envelope_path
    if grant_path.is_symlink() or not grant_path.is_file():
        raise FileNotFoundError(f"exact selected Government grant required at {grant_path}")
    raw = grant_path.read_bytes()
    if _sha(grant_path) != selected.envelope_sha:
        raise ValueError("selected Government grant SHA-256 differs from the parent binding")
    envelope = json.loads(raw)
    grant = envelope.get("grant")
    expected_limits = {
        "key": selected.key,
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
    if selected.name in {"r6", "r7"}:
        expected_limits["maxActualInputValidations"] = 1
        expected_limits["maxFreshStaticCasePreparations"] = 1
    if not isinstance(grant, dict) or any(grant.get(key) != value for key, value in expected_limits.items()):
        raise ValueError("selected Government grant is not the exact one-case Government allocation")
    expected_python = {"path": "C:/Python313/python.exe",
                       "sha256": "d87063e5597f257004c731b66c59c56c91038861c6877b1a3dca6b8c4e919125"}
    if (grant.get("python") != expected_python or
            Path(sys.executable).resolve() != Path(expected_python["path"]).resolve() or
            _sha(Path(sys.executable)) != expected_python["sha256"]):
        raise ValueError("Government preparation requires the exact existing Python313 binding")
    product = grant.get("product", {})
    accepted = gi.government.PIN["accepted"]
    if (product.get("name") != "Government" or
            product.get("sourceSha") != accepted["sourceCommit"] or
            product.get("binarySha256") != accepted["binary"]["sha256"] or
            product.get("delegateSha256") != EXPECTED_DELEGATE_SHA256):
        raise ValueError("Government grant product/delegate pins differ from the held inputs")
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


def prepare(profile_name="r5") -> dict:
    selected = profile(profile_name)
    output_root = selected.external_root / "government"
    grant = _validate_grant(selected)
    gi.TASK_ID = selected.task_id
    equivalence = _historical_equivalence()
    delegate_path = (gi.FIXTURE_ROOT / "deterministic_delegate.py").resolve(strict=True)
    if _sha(delegate_path) != EXPECTED_DELEGATE_SHA256:
        raise ValueError("pinned deterministic delegate bytes are not present")

    # No historical fixture namespace is reused or modified.
    if output_root.exists() or output_root.is_symlink():
        raise FileExistsError(f"Government output subtree already exists: {output_root}")
    output_root.mkdir(parents=True)
    actor = output_root / "actor"
    initial = gi.create_disposable_repository(actor)
    if _sha(actor / "government.yaml") != EXPECTED_GOVERNMENT_CONFIG_SHA256:
        raise ValueError("fresh repository config differs from the historical Constitution input")

    # Deliberately no G5 inspect is run. Reuse the digest only after the exact
    # Government config bytes have matched the retained inspect input above.
    bound = gi.bind_order_to_inspection(actor, RETAINED_CONSTITUTION_DIGEST)
    if bound["constitutionDigest"] != RETAINED_CONSTITUTION_DIGEST:
        raise AssertionError("fresh Order was not bound to the retained Constitution")

    released = output_root / "released"
    role_evidence = output_root / "role-evidence"
    controller_evidence = output_root / "controller-evidence"
    results = output_root / "results"
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
    ledger_path = output_root / "fixture-ledger.sqlite"

    correction = {
        "path": str(selected.envelope_path.resolve(strict=True)),
        "sha256": selected.envelope_sha,
        "sourceKey": selected.key,
    }
    diagnostic_config = {
        "arm": "government",
        "correction": correction,
        "requestPath": str(request_path.resolve()),
    }
    diagnostics_raw = gi.canonical_json(diagnostic_config) + b"\n"
    _write_new(diagnostics_path, diagnostics_raw)
    diagnostics_binding = {"path": str(diagnostics_path.resolve()), "sha256": _sha(diagnostics_path)}

    # The selected successor grant caps role calls at six. Preserve the exact
    # existing role-authorization schema and its finite controller wall limit.
    gi.MAX_ROLE_CALLS = 6
    gi.MAX_WALL_SECONDS = 38
    authorization = gi.build_role_authorization(
        trial_id=selected.key,
        dispatch_id=selected.dispatch_id,
        request_path=request_path,
        ledger_path=ledger_path,
        runtime_path=runtime_path,
        role_evidence_directory=role_evidence,
        expires_at=int(time.time()) + 24 * 60 * 60,
        python_executable=sys.executable,
        delegate_path=delegate_path,
        status="approved",
        fixture_authorization=R5_AUTHORIZATION,
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
        profile_file = gi.runtime_file(RUNTIME_ROOT / "government_native_profile.py")
        runtime_files[profile_file["path"]] = profile_file
        slot["runtimeFiles"] = [runtime_files[path] for path in sorted(runtime_files)]
    runtime_raw = gi.canonical_json(runtime) + b"\n"
    _write_new(runtime_path, runtime_raw)

    backlog = gi.build_backlog(runtime_path=runtime_path, queue_state_directory=queue_state,
                               task_id=selected.task_id)
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
    grant_release = {"path": correction["path"], "sha256": correction["sha256"]}
    released_inputs.extend([diagnostics_release, grant_release])

    role_slots = gi.government.configured_roles(runtime)
    runtime_files = runtime["executor"]["runtimeFiles"]
    for item in runtime_files:
        if gi.runtime_file(item["path"]) != item:
            raise ValueError("a runtime source changed while Government preparation was frozen")
    manifest = {
        "apiVersion": "markitect.scientist-government-native-fixture-preparation/v1alpha1",
        "fixtureId": selected.dispatch_id,
        "actorRepository": str(actor.resolve()),
        "initialCommit": initial["baseCommit"],
        "baseCommit": bound["baseCommit"],
        "managedRef": gi.MANAGED_REF,
        "constitutionDigest": RETAINED_CONSTITUTION_DIGEST,
        "orderSha256": bound["orderSha256"],
        "inputEquivalence": equivalence,
        "trialId": selected.key,
        "dispatchId": selected.dispatch_id,
        "taskId": gi.TASK_ID,
        "roleAuthorization": {"path": str(authorization_path.resolve()), "sha256": _sha(authorization_path)},
        "diagnostics": {**diagnostics_binding, "correction": correction},
        "runtime": {"path": str(runtime_path.resolve()), "sha256": _sha(runtime_path)},
        "backlog": {"path": str(backlog_path.resolve()), "sha256": _sha(backlog_path)},
        selected.marker: correction,
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
    manifest_path = output_root / "preparation-final.json"
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
        "grantSha256": selected.envelope_sha,
        "runtimeFileCount": len(runtime_files),
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--profile", choices=("r5", "r6", "r7"), default="r5")
    parser.add_argument(
        "--final-runtime-ready", action="store_true", required=True,
        help="explicit root signal: runtime/budget/diagnostic source closure is frozen",
    )
    args = parser.parse_args()
    if not args.final_runtime_ready:
        parser.error("wait until Root confirms the final Government runtime source freeze")
    result = prepare(args.profile)
    print(json.dumps(result, sort_keys=True, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
