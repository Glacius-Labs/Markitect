"""Bounded middleware helper for one native agentexec role call.

It accepts the documented Invocation bytes, a validated acyclic controller
bootstrap, and operator role authorization, reserves before one delegate, and
returns response bytes unchanged. It makes no OS isolation claim.
"""
from __future__ import annotations

import hashlib
import argparse
import json
import os
from pathlib import Path
import re
import stat
import sys
import time


def _strict_json(raw: bytes, label: str):
    def object_pairs(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise ValueError(f"{label} contains duplicate JSON keys")
            result[key] = value
        return result
    try:
        return json.loads(raw, object_pairs_hook=object_pairs)
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise ValueError(f"{label} is not valid JSON") from exc


def start_native_diagnostics(argv):
    """Retain bounded wrapper output before dependent imports or bootstrap checks.

    Only the explicitly hashed corrected-case configuration enables this path.
    The native RuntimeFiles/freeze owns that configuration; it grants no delegate
    authority. Every wrapper is counted even if a later import or check fails.
    """
    if "--diagnostics-config" not in argv:
        return None
    import atexit
    import sqlite3
    if (argv.count("--diagnostics-config") != 1 or argv.count("--diagnostics-sha256") != 1 or
            argv.index("--diagnostics-config") + 1 >= len(argv) or
            argv.index("--diagnostics-sha256") + 1 >= len(argv)):
        raise ValueError("exact wrapper diagnostic path and digest arguments required")
    config_path = Path(argv[argv.index("--diagnostics-config") + 1]).resolve(strict=True)
    expected = argv[argv.index("--diagnostics-sha256") + 1]
    raw = config_path.read_bytes()
    if len(raw) > 1024 * 1024 or hashlib.sha256(raw).hexdigest() != expected:
        raise ValueError("wrapper diagnostic configuration digest mismatch")
    config = json.loads(raw)
    if not isinstance(config, dict) or set(config) != {"arm", "correction", "requestPath"}:
        raise ValueError("exact wrapper diagnostic configuration fields required")
    correction = config["correction"]
    if (not isinstance(correction, dict) or set(correction) != {"path", "sha256", "sourceKey"} or
            not Path(correction.get("path", "")).is_absolute() or
            not Path(config.get("requestPath", "")).is_absolute()):
        raise ValueError("exact corrected wrapper diagnostic grant binding required")
    arm = config["arm"]
    request_path = Path(config["requestPath"]).resolve()
    if arm not in {"government", "classic"}:
        raise ValueError("wrapper diagnostic requires an exact native arm")

    if correction["sourceKey"] == "native-s1-corrected-integration-20261008-r2":
        # Preserve the already released R2 wire shape and provenance check.
        request = json.loads(request_path.read_bytes()) if request_path.is_file() else None
        if request is not None and (
                request.get("nativeFixtureR3Grant") is not None or
                request.get("nativeFixtureCorrection") != correction):
            raise ValueError("R2 wrapper diagnostics cannot authorize an R3 Request")
        correction_raw = Path(correction["path"]).read_bytes()
        if hashlib.sha256(correction_raw).hexdigest() != correction["sha256"]:
            raise ValueError("wrapper diagnostic correction grant digest mismatch")
        source = json.loads(correction_raw)
        snapshot_raw = Path(source["sourceCoordinationPath"]).read_bytes()
        if hashlib.sha256(snapshot_raw).hexdigest() != source["sourceCoordinationSha256"]:
            raise ValueError("wrapper diagnostic source snapshot digest mismatch")
        scientist = next(item for item in json.loads(snapshot_raw)["threads"] if item["name"] == "Scientist")
        grant = source["grant"]
        if (source["sourceThreadId"] != "01a11367-a781-7683-a20f-46e12614dcb4" or
                source["sourceJsonPointer"] != "threads[name=Scientist].evidence.correctedNativeIntegrationGrant" or
                scientist["evidence"]["correctedNativeIntegrationGrant"] != grant or
                grant["key"] != "native-s1-corrected-integration-20261008-r2" or
                grant["limits"][arm]["maxAdditionalRoleInvocationsIncludingFailedWrapperStarts"] != 6 or
                any(grant[key] != 0 for key in ("realActorCalls", "providerCalls", "metadataAppServerTrees", "studyCells"))):
            raise ValueError("wrapper diagnostic grant differs from corrected allocation")
        max_starts = grant["limits"][arm]["maxAdditionalRoleInvocationsIncludingFailedWrapperStarts"]
    elif correction["sourceKey"] == "native-s1-contract-corrected-integration-20261008-r3":
        request_path = request_path.resolve(strict=True)
        request = json.loads(request_path.read_bytes())
        r3_binding = request.get("nativeFixtureR3Grant")
        original = request.get("nativeFixtureGrant")
        if (request.get("arm") != arm or request.get("nativeFixtureCorrection") is not None or
                r3_binding != correction or
                not isinstance(original, dict) or set(original) != {"path", "sha256", "sourceKey"}):
            raise ValueError("R3 wrapper diagnostics must bind the exact R3 Request and retain original provenance")
        from native_fixture_budget import validate_r3_grant_binding
        validated = validate_r3_grant_binding(request, original["path"], original["sha256"])
        if (validated.get("grantKey") != correction["sourceKey"] or
                validated.get("maxRoleStarts") != 6 or validated.get("maxDeterministicDelegates") != 6):
            raise ValueError("wrapper diagnostic grant differs from the exact R3 role allocation")
        max_starts = validated["maxRoleStarts"]
    elif correction["sourceKey"] == "government-serialization-native-20261008-r4":
        request_path = request_path.resolve(strict=True)
        request = _strict_json(request_path.read_bytes(), "R4 wrapper Request")
        r4_binding = request.get("nativeFixtureR4Grant") if isinstance(request, dict) else None
        original = request.get("nativeFixtureGrant") if isinstance(request, dict) else None
        if (request.get("dispatchId") != "government-native-serialization-r4" or
                request.get("arm") != "government" or
                request.get("nativeFixtureR3Grant") is not None or
                request.get("nativeFixtureCorrection") is not None or
                r4_binding != correction or
                not isinstance(original, dict) or set(original) != {"path", "sha256", "sourceKey"}):
            raise ValueError("R4 wrapper diagnostics must bind the exact Government Request and retain R1 provenance")
        from native_fixture_budget import validate_r4_grant_binding, validate_r4_entry_gate
        validated = validate_r4_grant_binding(request, original["path"], original["sha256"])
        if (validated.get("grantKey") != correction["sourceKey"] or
                validated.get("maxWrapperAttempts") != 6 or
                validated.get("maxDeterministicDelegates") != 6 or
                validated.get("maxNativeStarts") != 2):
            raise ValueError("wrapper diagnostic grant differs from the exact R4 allocation")
        role_binding = request.get("product", {}).get("government", {}).get("roleAuthorization", {})
        role_path = Path(role_binding.get("path", "")).resolve(strict=True)
        role_raw = role_path.read_bytes()
        if (role_binding.get("sha256") != hashlib.sha256(role_raw).hexdigest() or
                not any(isinstance(item, dict) and item.get("path") and item.get("sha256") == role_binding["sha256"] and
                        Path(item["path"]).resolve(strict=True) == role_path
                        for item in request.get("releasedInputs", []))):
            raise ValueError("R4 role authorization must remain the exact released Request input")
        import native_controller
        native_controller.validate_r4_delegate_authorization(request, role_raw, validated)
        max_starts = validated["maxWrapperAttempts"]
    elif correction["sourceKey"] in {"government-scope-native-20261008-r5",
                                     "government-released-binding-native-20261008-r6"}:
        request_path = request_path.resolve(strict=True)
        request = _strict_json(request_path.read_bytes(), "R5/R6 wrapper Request")
        from government_native_profile import profile, request_profile
        fixture_profile = request_profile(request)
        original = request.get("nativeFixtureGrant") if isinstance(request, dict) else None
        if (fixture_profile not in {profile("r5"), profile("r6")} or
                request.get(fixture_profile.marker) != correction or
                not isinstance(original, dict) or set(original) != {"path", "sha256", "sourceKey"}):
            raise ValueError("R5/R6 wrapper diagnostics must bind the exact Government Request and retain R1 provenance")
        from native_fixture_budget import validate_profile_grant_binding
        validated = validate_profile_grant_binding(request, original["path"], original["sha256"],
                                                   fixture_profile)
        if (validated.get("grantKey") != correction["sourceKey"] or
                validated.get("maxWrapperAttempts") != 6 or
                validated.get("maxDeterministicDelegates") != 6 or
                validated.get("maxNativeStarts") != 2):
            raise ValueError("wrapper diagnostic grant differs from the exact R5/R6 role allocation")
        role_binding = request.get("product", {}).get("government", {}).get("roleAuthorization", {})
        role_path = Path(role_binding.get("path", "")).resolve(strict=True)
        role_raw = role_path.read_bytes()
        if (role_binding.get("sha256") != hashlib.sha256(role_raw).hexdigest() or
                not any(isinstance(item, dict) and item.get("path") and item.get("sha256") == role_binding["sha256"] and
                        Path(item["path"]).resolve(strict=True) == role_path
                        for item in request.get("releasedInputs", []))):
            raise ValueError("R5/R6 role authorization must remain the exact released Request input")
        import native_controller
        native_controller.validate_profile_delegate_authorization(
            request, role_raw, validated, fixture_profile)
        max_starts = validated["maxWrapperAttempts"]
    else:
        raise ValueError("wrapper diagnostic grant key is not authorized")
    directory = config_path.parent / "wrapper-diagnostics"
    directory.mkdir(exist_ok=True)
    db = sqlite3.connect(directory / "starts.sqlite", timeout=5)
    try:
        db.execute("CREATE TABLE IF NOT EXISTS starts(id INTEGER PRIMARY KEY, config_sha TEXT, started REAL)")
        db.execute("BEGIN IMMEDIATE")
        rows = db.execute("SELECT config_sha FROM starts").fetchall()
        if len(rows) >= max_starts or any(row[0] != expected for row in rows):
            raise ValueError("corrected wrapper invocation cap exhausted or configuration changed")
        if correction["sourceKey"] == "government-serialization-native-20261008-r4":
            # Re-read the live activation/slot immediately before the durable claim.
            validate_r4_entry_gate(validated)
        elif correction["sourceKey"] in {"government-scope-native-20261008-r5",
                                         "government-released-binding-native-20261008-r6"}:
            # Re-read the exact profile activation/slot immediately before the durable claim.
            native_controller.validate_profile_live_gate(validated, fixture_profile)
        cursor = db.execute("INSERT INTO starts(config_sha,started) VALUES(?,?)", (expected, time.time()))
        invocation = cursor.lastrowid
        db.commit()
    finally:
        db.close()
    call = directory / f"wrapper-{invocation:06d}"
    call.mkdir()
    streams = {}
    class Tee:
        def __init__(self, original, path):
            self.original, self.path = original, path
            self.file = path.open("xb")
            self.buffer = self
            self.count = 0
        def write(self, value):
            data = value.encode("utf-8") if isinstance(value, str) else value
            if self.count + len(data) > 16 * 1024 * 1024:
                raise ValueError("bounded raw wrapper diagnostics exhausted")
            self.file.write(data)
            self.file.flush()
            self.count += len(data)
            written = self.original.buffer.write(data)
            self.original.flush()
            return len(value) if isinstance(value, str) else written
        def flush(self):
            self.file.flush()
            self.original.flush()
    start = {"arm": arm, "configSha256": expected, "correction": correction,
             "requestPath": config["requestPath"], "invocation": invocation,
             "argvSha256": hashlib.sha256(json.dumps(argv).encode()).hexdigest(),
             "wrapperSha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
             "started": time.time(), "providerUsage": None}
    (call / "start.json").write_text(json.dumps(start, sort_keys=True), encoding="utf-8")
    for name in ("stdout", "stderr"):
        streams[name] = Tee(getattr(sys, name), call / f"{name}.log")
        setattr(sys, name, streams[name])
    def finish():
        receipts = []
        for name, stream in streams.items():
            stream.flush()
            receipts.append({"path": str(stream.path), "sha256": hashlib.sha256(stream.path.read_bytes()).hexdigest()})
        (call / "finished.json").write_text(json.dumps({"start": start, "receipts": receipts}, sort_keys=True), encoding="utf-8")
    atexit.register(finish)
    return {"config": config, "callDirectory": str(call), "finish": finish}


# Execute before ledger/process/controller imports: their earliest exceptions
# must reach the same raw stderr sink and consumed wrapper-start record.
_NATIVE_DIAGNOSTICS = start_native_diagnostics(sys.argv) if __name__ == "__main__" else None

from ledger import Ledger, LimitReached
from process import bounded
import government
import native_controller

INVOCATION_API = "markitect.example.org/agent-execution/v1alpha1"
ROLE_AUTH_API = "markitect.scientist-role-authorization/v1alpha1"
NATIVE_FIXTURE_AUTH = {"sourceGrantKey": "native-s1-integration-fixtures-20261008",
                       "classification": "nativeFixture", "providerUse": "noProvider"}
_GOVERNMENT_FIXTURE_DELEGATE_SHA256 = "e8b8e5087f994a975efc2228301cf7f51dee9de4d64b77077a0941db7a8e98b9"
_CLASSIC_FIXTURE_DELEGATE_SHA256 = "ddac10f2d0cb7884d24d2e31b4933ec500d87857d88a0688ead7a5dcaa450c20"
_HEX256 = re.compile(r"^[0-9a-f]{64}$")
_DIGEST = re.compile(r"^sha256:[0-9a-f]{64}$")
_MAX_INVOCATION_BYTES = 32 * 1024 * 1024
_MAX_AUTH_BYTES = 4 * 1024 * 1024
_MAX_ROLE_LOG_BYTES = 16 * 1024 * 1024


def digest(raw: bytes) -> str:
    return hashlib.sha256(raw).hexdigest()


def encoded(value) -> bytes:
    # Match the existing Scientist/dispatch canonical encoder for ASCII paths
    # and its JSON-compatible escaping behavior.
    return json.dumps(value, sort_keys=True, separators=(",", ":")).encode("utf-8")


def native_digest(raw: bytes) -> str:
    """agentexec and Government RuntimeFile digests use a sha256: prefix."""
    return "sha256:" + digest(raw)


def _native_mode(path: Path) -> str:
    mode = stat.S_IMODE(path.stat().st_mode)
    if os.name == "nt":
        return "0644" if mode & stat.S_IWUSR else "0444"
    return f"{mode:04o}"


def parse_invocation(raw: bytes) -> dict:
    if not isinstance(raw, bytes) or not raw or len(raw) > _MAX_INVOCATION_BYTES:
        raise ValueError("agentexec Invocation bytes are empty or exceed the protocol bound")
    value = _strict_json(raw, "agentexec Invocation")
    expected = {"apiVersion", "runId", "nonce", "inputDigest", "request"}
    if not isinstance(value, dict) or set(value) != expected:
        raise ValueError("agentexec Invocation fields do not match v1alpha1")
    if (value["apiVersion"] != INVOCATION_API or not isinstance(value["runId"], str) or not value["runId"] or
            not isinstance(value["nonce"], str) or not value["nonce"] or
            not isinstance(value["inputDigest"], str) or not _DIGEST.fullmatch(value["inputDigest"]) or
            not isinstance(value["request"], dict)):
        raise ValueError("agentexec Invocation identity is malformed")
    request = value["request"]
    request_fields = {"role", "sourceRevision", "modelDigest", "modulePin", "projectionId", "scopeIds",
                      "policyIds", "context", "artifacts"}
    if set(request) != request_fields:
        raise ValueError("agentexec Request fields do not match v1alpha1")
    if request.get("role") not in {"executor", "verifier"}:
        raise ValueError("only Government Executor/Verifier agentexec roles are supported")
    if not isinstance(request.get("projectionId"), str):
        raise ValueError("agentexec projectionId is required")
    if not all(isinstance(request.get(key), list) for key in ("scopeIds", "policyIds", "artifacts")):
        raise ValueError("agentexec scopes, policies and artifacts must be arrays")
    if not isinstance(request.get("context"), dict):
        raise ValueError("agentexec context must be a JSON object")
    return value


def role_projection(projection_id: str) -> tuple[str, str]:
    parts = projection_id.split("/")
    if len(parts) != 3 or parts[0] != "government" or parts[1] not in {"execute", "review", "vote"} or not parts[2]:
        raise ValueError("agentexec projectionId does not identify a documented Government role slot")
    return parts[1], parts[2]


def wrapper_arguments(script_path: str, authorization_path: str, authorization_sha256: str,
                      evidence_path: str, slot_id: str | None = None) -> list[str]:
    result = [str(Path(script_path).resolve()), "--authorization", str(Path(authorization_path).resolve()),
              "--authorization-sha256", authorization_sha256,
              "--evidence", str(Path(evidence_path).resolve())]
    if slot_id is not None:
        result.extend(["--slot", slot_id])
    return result


def _released(captured: dict, path: str, expected_sha: str, label: str) -> bytes:
    target = Path(path).resolve(strict=True)
    for source, content in captured.items():
        if Path(source).resolve() == target:
            if digest(content) != expected_sha:
                raise ValueError(f"released {label} digest mismatch")
            return content
    raise ValueError(f"{label} is not an exact operator-authorized released input")


def _file_digest(path: str, expected: str, label: str, *, native=False) -> None:
    target = Path(path)
    resolved = target.resolve(strict=True)
    if target.is_symlink() or not resolved.is_file():
        raise ValueError(f"{label} must be a regular non-symlink file")
    actual = resolved.read_bytes()
    value = native_digest(actual) if native else digest(actual)
    if not target.is_absolute() or value != expected:
        raise ValueError(f"{label} digest mismatch")


def _role_auth(raw: bytes, expected_sha: str, request: dict, request_raw: bytes,
               authority, captured: dict, invocation: dict, authorization_path: str,
               expected_slot: str | None, role_resolver=None) -> tuple[dict, dict, dict]:
    if not _HEX256.fullmatch(expected_sha) or digest(raw) != expected_sha:
        raise ValueError("operator-supplied role authorization digest mismatch")
    if len(raw) > _MAX_AUTH_BYTES:
        raise ValueError("role authorization exceeds its size bound")
    if Path(authorization_path).resolve(strict=True).read_bytes() != raw:
        raise ValueError("role authorization path bytes differ from the supplied approved bytes")
    auth = _strict_json(raw, "role authorization")
    if not isinstance(auth, dict) or auth.get("apiVersion") != ROLE_AUTH_API or auth.get("status") != "approved":
        raise ValueError("approved Scientist role authorization required")
    if request.get("mode") == "mechanical" and auth.get("fixtureAuthorization") != NATIVE_FIXTURE_AUTH:
        raise ValueError("mechanical native role authorization requires the explicit no-provider fixture grant")
    ledger_path = getattr(authority, "ledger_path", None)
    if (auth.get("trialId") != request.get("trialId") or auth.get("dispatchId") != request.get("dispatchId") or
            auth.get("taskId") != request.get("task", {}).get("id") or
            str(Path(auth.get("ledgerPath", "")).resolve()) != str(Path(ledger_path).resolve())):
        raise ValueError("role authorization Request, dispatch, task, or ledger binding mismatch")
    if not hasattr(authority, "grant") or not callable(getattr(authority, "validate", None)):
        raise ValueError("an externally validated Coordinator Authority is required")
    limits = request.get("limits", {})
    max_task_calls = limits.get("taskActorCalls") if isinstance(limits, dict) else None
    grant_max_calls = getattr(authority, "grant", {}).get("maxActorSessions")
    if (type(auth.get("maxCalls")) is not int or type(max_task_calls) is not int or
            type(grant_max_calls) is not int or not 1 <= auth["maxCalls"] <= min(max_task_calls, grant_max_calls) or
            type(auth.get("expiresAt")) not in (int, float) or not time.time() < auth["expiresAt"] < float("inf")):
        raise ValueError("finite operator role-call count and expiry required")
    grant_expiry = getattr(authority, "grant", {}).get("expiresAt")
    if type(grant_expiry) not in (int, float) or auth["expiresAt"] > grant_expiry:
        raise ValueError("role authorization cannot outlive the validated Coordinator grant")
    request_path = auth.get("requestPath")
    if not isinstance(request_path, str) or not Path(request_path).is_absolute():
        raise ValueError("exact outer Request path required in role authorization")
    if Path(request_path).resolve(strict=True).read_bytes() != request_raw:
        raise ValueError("outer Request file changed after operator authorization")
    auth_path = str(Path(authorization_path).resolve(strict=True))

    arm = request.get("arm")
    product = request.get("product", {}).get(arm, {})
    runtime_path, runtime_sha = product.get("runtime", {}).get("path"), product.get("runtime", {}).get("sha256")
    if not isinstance(runtime_path, str) or not isinstance(runtime_sha, str):
        raise ValueError("Request-bound Government runtime required")
    runtime_raw = _released(captured, runtime_path, runtime_sha, "Government runtime")
    if auth.get("runtimePath") != str(Path(runtime_path).resolve()):
        raise ValueError("role authorization runtime binding mismatch")
    runtime = _strict_json(runtime_raw, "Government runtime")
    if arm == "government":
        roles = government.configured_roles(runtime)
    elif arm == "classic":
        import classic_integration
        roles = classic_integration.configured_roles(runtime)
    else:
        raise ValueError("role bridge only supports native Government or Classic arms")
    by_slot = {role["slotId"]: role for role in roles}
    if len(by_slot) != len(roles):
        raise ValueError("configured Government role slots are ambiguous")
    auth_slots = auth.get("slots")
    if not isinstance(auth_slots, list) or len(auth_slots) != len(roles):
        raise ValueError("operator authorization must cover every configured native role slot")
    auth_by_slot = {slot.get("slotId"): slot for slot in auth_slots if isinstance(slot, dict)}
    if len(auth_by_slot) != len(auth_slots) or set(auth_by_slot) != set(by_slot):
        raise ValueError("authorized role slots do not exactly match the native runtime")
    if role_resolver is not None:
        selected = role_resolver(invocation, auth, runtime)
        if not isinstance(selected, dict) or set(selected) != {"slotId", "phase", "responseRole"}:
            raise ValueError("native product role resolver returned a malformed slot binding")
        slot_id, phase = selected["slotId"], selected["phase"]
        response_role = selected["responseRole"]
    else:
        if arm != "government":
            raise ValueError("non-Government role bridge requires an adapter role resolver")
        phase, slot_id = role_projection(invocation["request"]["projectionId"])
        response_role = invocation["request"]["role"]
    if expected_slot is not None and slot_id != expected_slot:
        raise ValueError("native invocation arrived at a different configured wrapper slot")
    role_slot = auth_by_slot.get(slot_id)
    configured = by_slot.get(slot_id)
    if not role_slot or not configured or (phase, response_role) != (configured["phase"], configured["responseRole"]):
        raise ValueError("agentexec invocation role/phase is not authorized for this slot")
    if (role_slot.get("phase"), role_slot.get("responseRole")) != (phase, response_role):
        raise ValueError("operator role grant phase/role mismatch")
    delegate = role_slot.get("delegate")
    if not isinstance(delegate, dict):
        raise ValueError("operator delegate runner binding required")
    argv = delegate.get("argv")
    command = delegate.get("command")
    if (not isinstance(command, str) or not command or not Path(command).is_absolute() or
            not isinstance(argv, list) or not argv or not all(isinstance(arg, str) for arg in argv) or
            argv[0] != command):
        raise ValueError("delegate command must be an explicit absolute executable equal to argv[0]")
    wrapper = role_slot.get("wrapper", {})
    wrapper_slot = slot_id if arm == "government" else None
    expected_args = wrapper_arguments(__file__, auth_path, expected_sha,
                                      auth["roleEvidenceDirectory"], wrapper_slot)
    diagnostics = auth.get("diagnostics")
    r3_grant = request.get("nativeFixtureR3Grant")
    r2_correction = request.get("nativeFixtureCorrection")
    r4_grant = request.get("nativeFixtureR4Grant")
    fixture_profile = native_controller.native_profile(request)
    if sum(value is not None for value in (r3_grant, r2_correction, r4_grant,
                                           request.get("nativeFixtureR5Grant"),
                                           request.get("nativeFixtureR6Grant"))) > 1:
        raise ValueError("R2, R3, R4, R5, and R6 additive grants cannot be combined")
    if r4_grant is not None and (request.get("dispatchId") != "government-native-serialization-r4" or arm != "government"):
        raise ValueError("R4 grant is restricted to its exact Government dispatch")
    diagnostic_grant = (request.get(fixture_profile.marker) if fixture_profile is not None else
                        r4_grant if r4_grant is not None else
                        r3_grant if r3_grant is not None else r2_correction)
    if diagnostic_grant is not None:
        if not isinstance(diagnostics, dict) or set(diagnostics) != {"path", "sha256"}:
            raise ValueError("corrected role requires released diagnostic configuration")
        diagnostic_raw = _released(captured, diagnostics["path"], diagnostics["sha256"], "wrapper diagnostics")
        diagnostic_value = json.loads(diagnostic_raw)
        if (diagnostic_value.get("correction") != diagnostic_grant or
                diagnostic_value.get("requestPath") != auth["requestPath"] or diagnostic_value.get("arm") != arm):
            raise ValueError("wrapper diagnostics do not match this corrected Request")
        expected_args.extend(["--diagnostics-config", diagnostics["path"], "--diagnostics-sha256", diagnostics["sha256"]])
    if wrapper.get("command") != configured["command"] or configured["args"] != expected_args:
        raise ValueError("runtime wrapper argv is not the exact Scientist role bridge invocation")
    script_sha = digest(Path(__file__).resolve().read_bytes())
    runtime_files = configured.get("runtimeFiles")
    if not isinstance(runtime_files, list):
        raise ValueError("native role runtime file bindings are required")
    wrapper_files = {}
    for item in runtime_files:
        if (not isinstance(item, dict) or set(item) != {"path", "mode", "digest"} or
                not isinstance(item["path"], str) or not Path(item["path"]).is_absolute() or
                not isinstance(item["mode"], str) or not re.fullmatch(r"0[0-7]{3}", item["mode"]) or
                not isinstance(item["digest"], str) or not _DIGEST.fullmatch(item["digest"])):
            raise ValueError("native role runtime file binding is malformed")
        key = str(Path(item["path"]).resolve())
        if key in wrapper_files:
            raise ValueError("duplicate native role runtime file binding")
        _file_digest(item["path"], item["digest"], "native role runtime file", native=True)
        if _native_mode(Path(item["path"])) != item["mode"]:
            raise ValueError("native role runtime file mode differs from its bound mode")
        wrapper_files[key] = item["digest"]
    controller_script = Path(native_controller.__file__).resolve()
    controller_sha = digest(controller_script.read_bytes())
    required_wrapper_files = {str(Path(__file__).resolve()): "sha256:" + script_sha,
                              str(controller_script): "sha256:" + controller_sha,
                              auth_path: "sha256:" + expected_sha}
    if diagnostic_grant is not None:
        required_wrapper_files[str(Path(diagnostics["path"]).resolve())] = "sha256:" + diagnostics["sha256"]
    if request.get("nativeFixtureGrant") is not None:
        budget_script = Path(__file__).with_name("native_fixture_budget.py").resolve()
        required_wrapper_files[str(budget_script)] = "sha256:" + digest(budget_script.read_bytes())
    if arm == "classic":
        import classic_integration
        classic_pin = Path(__file__).with_name("classic-pin.json")
        classic_source = Path(__file__).with_name("classic.py")
        required_wrapper_files.update({
            str(Path(classic_integration.__file__).resolve()): "sha256:" + digest(Path(classic_integration.__file__).read_bytes()),
            str(classic_pin.resolve()): "sha256:" + digest(classic_pin.read_bytes()),
            str(classic_source.resolve()): "sha256:" + digest(classic_source.read_bytes())})
    if any(wrapper_files.get(path) != value for path, value in required_wrapper_files.items()):
        raise ValueError("wrapper scripts and role authorization must be runtime-file pinned")
    delegate = role_slot.get("delegate")
    if not isinstance(delegate, dict):
        raise ValueError("operator delegate runner binding required")
    argv = delegate.get("argv")
    if (not isinstance(argv, list) or not argv or not all(isinstance(arg, str) for arg in argv) or
            not Path(argv[0]).is_absolute() or type(delegate.get("timeoutSeconds")) is not int or
            not 0 < delegate["timeoutSeconds"] <= 180 or
            type(delegate.get("maxStdoutBytes")) is not int or type(delegate.get("maxStderrBytes")) is not int):
        raise ValueError("operator delegate command and finite bounds are required")
    if not all(0 < delegate[key] <= _MAX_ROLE_LOG_BYTES for key in ("maxStdoutBytes", "maxStderrBytes")):
        raise ValueError("operator delegate output bounds exceed the shared process wrapper")
    command_digest = delegate.get("commandDigest")
    if wrapper_files.get(str(Path(argv[0]).resolve())) != command_digest:
        raise ValueError("delegate executable digest is not pinned by the native role runtime")
    _file_digest(argv[0], delegate.get("commandDigest", ""), "delegate executable", native=True)
    delegate_files = delegate.get("runtimeFiles")
    if not isinstance(delegate_files, list):
        raise ValueError("delegate runtime file bindings required")
    pinned_wrapper_files = {str(Path(item.get("path", "")).resolve()): item.get("digest")
                            for item in configured.get("runtimeFiles", []) if isinstance(item, dict)}
    for item in delegate_files:
        if (not isinstance(item, dict) or set(item) != {"path", "mode", "digest"} or
                not isinstance(item.get("path"), str)):
            raise ValueError("malformed delegate runtime file binding")
        path, sha = item["path"], item.get("digest")
        if (not isinstance(sha, str) or not _DIGEST.fullmatch(sha) or
                pinned_wrapper_files.get(str(Path(path).resolve())) != sha or
                _native_mode(Path(path)) != item["mode"]):
            raise ValueError("delegate runtime file is not included in the native role runtime pin")
        _file_digest(path, sha, "delegate runtime file", native=True)
    if (delegate.get("model") != configured.get("model") or
            delegate.get("modelOptions") != configured.get("modelOptions") or
            delegate.get("providerVersion") != configured.get("providerVersion")):
        raise ValueError("delegate model/provider configuration differs from the native role slot")
    if request.get("arm") not in {"government", "classic"} or request.get("operation") != "run_task":
        raise ValueError("native role bridge supports only an authorized Government or Classic run_task")
    return auth, role_slot, runtime


def preflight_authorization(request: dict, request_raw: bytes, authority, captured: dict,
                            authorization_path: str, authorization_raw: bytes,
                            authorization_sha256: str) -> None:
    """Validate every static slot and runtime pin before starting a native controller."""
    from dispatch import digest as dispatch_digest
    if dispatch_digest(authorization_raw) != authorization_sha256:
        raise ValueError("operator role authorization digest mismatch")
    product = request.get("product", {}).get(request.get("arm"), {})
    binding = product.get("roleAuthorization")
    auth_target = Path(authorization_path).resolve(strict=True)
    captured_auth = next((content for source, content in captured.items()
                          if Path(source).resolve() == auth_target), None)
    if (not isinstance(binding, dict) or Path(binding.get("path", "")).resolve() != auth_target or
            binding.get("sha256") != authorization_sha256 or captured_auth != authorization_raw):
        raise ValueError("operator authorization must be an exact released Request input")
    runtime_path = product.get("runtime", {}).get("path")
    runtime_raw = _released(captured, runtime_path, product.get("runtime", {}).get("sha256", ""),
                            "native role runtime")
    runtime = _strict_json(runtime_raw, "native role runtime")
    roles = (government.configured_roles(runtime) if request.get("arm") == "government" else
             __import__("classic_integration").configured_roles(runtime))
    if request.get("arm") == "classic":
        resolver = __import__("classic_integration").resolve_role
    else:
        resolver = None
    for role in roles:
        invocation = _preflight_invocation(request.get("arm"), role)
        _role_auth(authorization_raw, authorization_sha256, request, request_raw, authority,
                   captured, invocation, authorization_path, role["slotId"], role_resolver=resolver)


def _preflight_invocation(arm: str, role: dict) -> dict:
    """Construct source-shaped synthetic invocation identities for static slot checks."""
    if arm == "government":
        projection = f"government/{role['phase']}/{role['slotId']}"
        scope_ids = ["preflight-scope"]
        context = {}
    elif arm == "classic":
        import classic_integration
        projection = role["projectionId"]
        scope_ids = list(role["nativeScopeIds"])
        definitions = []
        for identity in classic_integration.NATIVE_SUBJECTS:
            api_version, kind, namespace, name = json.loads(identity)
            definitions.append({"apiVersion": api_version, "kind": kind,
                                "metadata": {"namespace": namespace, "name": name}})
        context = {"model": {"projectionId": projection,
                              "scopeIds": list(classic_integration.NATIVE_SUBJECTS),
                              "definitions": definitions}}
    else:
        raise ValueError("preflight invocation requires a supported native arm")
    return {"apiVersion": INVOCATION_API,
            "request": {"role": role["responseRole"], "sourceRevision": "preflight",
                        "modelDigest": "sha256:" + "0" * 64, "modulePin": "sha256:" + "0" * 64,
                        "projectionId": projection, "scopeIds": scope_ids, "policyIds": [],
                        "context": context, "artifacts": []}}


def _execution_sha(request: dict) -> str:
    return digest(encoded({key: value for key, value in request.items() if key != "operation"}))


def _response_usage(raw: bytes, invocation: dict) -> tuple[dict | None, int | None]:
    try:
        response = _strict_json(raw, "delegate response")
    except ValueError:
        return None, None
    request = invocation["request"]
    if not isinstance(response, dict) or any(response.get(key) != value for key, value in (
            ("apiVersion", INVOCATION_API), ("runId", invocation["runId"]), ("nonce", invocation["nonce"]),
            ("role", request["role"]), ("inputDigest", invocation["inputDigest"]))):
        return None, None
    usage = response.get("usage")
    if not isinstance(usage, dict) or usage.get("source") != "provider-reported":
        return response, None
    input_tokens, output_tokens = usage.get("inputTokens"), usage.get("outputTokens")
    for value in (input_tokens, output_tokens):
        if value is not None and (type(value) is not int or value < 0):
            return response, None
    tokens = input_tokens + output_tokens if input_tokens is not None and output_tokens is not None else None
    return response, tokens


def _known_no_provider_fixture_accounting(request, authorization, role_slot, fixture_bounds):
    """Return explicit zero-use accounting only for the two hash-pinned deterministic delegates."""
    if (fixture_bounds is None or request.get("mode") != "mechanical" or
            authorization.get("fixtureAuthorization") != NATIVE_FIXTURE_AUTH or
            fixture_bounds.get("sourceKey") != native_controller.NATIVE_FIXTURE_GRANT_KEY or
            fixture_bounds.get("product") != request.get("arm", "").title()):
        return None
    python = Path(sys.executable).resolve()
    delegate = role_slot.get("delegate") if isinstance(role_slot, dict) else None
    if (not isinstance(delegate, dict) or not isinstance(delegate.get("argv"), list) or
            not delegate["argv"] or Path(delegate["argv"][0]).resolve() != python or
            delegate.get("commandDigest") != native_digest(python.read_bytes())):
        return None
    if request.get("arm") == "government":
        script = Path(__file__).with_name("fixtures") / "government_positive" / "deterministic_delegate.py"
        expected_sha = _GOVERNMENT_FIXTURE_DELEGATE_SHA256
        expected_argv = [str(python), str(script.resolve()), "--phase", role_slot.get("phase")]
    elif request.get("arm") == "classic":
        import classic_integration
        script = (classic_integration.PACKET / "smoke" / "protocol_test_double.py").resolve()
        expected_sha = _CLASSIC_FIXTURE_DELEGATE_SHA256
        expected_argv = [str(python), str(script)]
    else:
        return None
    script = script.resolve(strict=True)
    if (script.is_symlink() or digest(script.read_bytes()) != expected_sha or
            delegate.get("argv") != expected_argv):
        return None
    pinned_files = delegate.get("runtimeFiles")
    if not isinstance(pinned_files, list) or not any(
            isinstance(item, dict) and Path(item.get("path", "")).resolve() == script and
            item.get("digest") == native_digest(script.read_bytes()) for item in pinned_files):
        return None
    return {"knownNoProviderCalls": 0, "knownNoProviderTokens": 0,
            "basis": "approved deterministic no-provider fixture delegate"}


def _reserve(ledger: Ledger, auth: dict, invocation: dict, invocation_sha: str,
             authorization_sha: str, role_slot: dict, evidence_path: Path,
             grant_token_threshold: int, *, fixture_role_limit: int | None = None,
             fixture_parallel_limit: int | None = None,
             fixture_process_limit: int | None = None, requested_timeout: float | None = None) -> tuple[str, str]:
    call_id = digest(encoded({"dispatchId": auth["dispatchId"], "runId": invocation["runId"],
                              "nonce": invocation["nonce"], "inputDigest": invocation["inputDigest"]}))
    with ledger.transaction() as db:
        db.execute("""CREATE TABLE IF NOT EXISTS government_role_calls(
            call_id TEXT PRIMARY KEY, dispatch_id TEXT NOT NULL, attempt_id TEXT UNIQUE NOT NULL,
            authorization_sha256 TEXT NOT NULL, invocation_sha256 TEXT NOT NULL,
            input_digest TEXT NOT NULL, run_id TEXT NOT NULL, nonce_sha256 TEXT NOT NULL,
            slot_id TEXT NOT NULL, phase TEXT NOT NULL, evidence_path TEXT NOT NULL,
            status TEXT NOT NULL, receipt TEXT, reserved_wall_seconds REAL NOT NULL DEFAULT 0,
            wall_seconds REAL)""")
        columns = {row[1] for row in db.execute("PRAGMA table_info(government_role_calls)")}
        if "reserved_wall_seconds" not in columns:
            db.execute("ALTER TABLE government_role_calls ADD COLUMN reserved_wall_seconds REAL NOT NULL DEFAULT 0")
        if "wall_seconds" not in columns:
            db.execute("ALTER TABLE government_role_calls ADD COLUMN wall_seconds REAL")
        prior = db.execute("SELECT 1 FROM government_role_calls WHERE call_id=?", (call_id,)).fetchone()
        if prior:
            raise ValueError("agentexec invocation replay is already reserved; no delegate relaunch")
        count = db.execute("SELECT COUNT(*) FROM government_role_calls WHERE dispatch_id=?", (auth["dispatchId"],)).fetchone()[0]
        if count >= auth["maxCalls"]:
            raise LimitReached("operator role-call authorization exhausted")
        if fixture_role_limit is not None or fixture_process_limit is not None or fixture_parallel_limit is not None:
            if (type(fixture_role_limit) is not int or type(fixture_parallel_limit) is not int or
                    type(fixture_process_limit) is not int or
                    fixture_role_limit <= 0 or fixture_process_limit <= 0 or
                    fixture_parallel_limit <= 0 or
                    type(requested_timeout) not in (int, float) or requested_timeout <= 0):
                raise ValueError("validated source-grant role-count and process-time bounds are required")
            active_calls = db.execute("SELECT COUNT(*) FROM government_role_calls WHERE status='reserved'").fetchone()[0]
            if active_calls >= fixture_parallel_limit:
                raise LimitReached("source grant role parallelism ceiling")
            source_calls = db.execute("SELECT COUNT(*) FROM government_role_calls").fetchone()[0]
            if source_calls >= fixture_role_limit:
                raise LimitReached("source grant deterministic role-start ceiling")
            source_seconds = db.execute("SELECT COALESCE(SUM(CASE WHEN wall_seconds IS NULL "
                                        "THEN reserved_wall_seconds ELSE wall_seconds END),0) "
                                        "FROM government_role_calls").fetchone()[0]
            if source_seconds + requested_timeout > fixture_process_limit:
                raise LimitReached("source grant deterministic role-process seconds ceiling")
        grant = db.execute("SELECT ceiling FROM dispatch_authority_history WHERE authority_sha=?",
                           (ledger.authority_key,)).fetchone()
        if not grant or type(grant[0]) is not int:
            raise ValueError("bound Coordinator grant ceiling is unavailable in ledger history")
        trial_attempts = db.execute("SELECT COUNT(*) FROM attempts").fetchone()[0]
        if trial_attempts >= grant[0]:
            raise LimitReached("grant session limit")
        if type(grant_token_threshold) is not int or grant_token_threshold <= 0:
            raise ValueError("validated Coordinator retrospective token threshold is required")
        dispatch_attempt = db.execute("SELECT attempt FROM dispatches WHERE id=?",
                                      (auth["dispatchId"],)).fetchone()
        if not dispatch_attempt:
            raise ValueError("current outer dispatch booking is absent from the common ledger")
        controller = db.execute("SELECT 1 FROM controller_runs WHERE dispatch_id=?",
                                (auth["dispatchId"],)).fetchone()
        if dispatch_attempt[0] is None and not controller:
            raise ValueError("controller timing record is absent for a nullable outer Actor attempt")
        if dispatch_attempt[0] is None:
            reported_tokens = db.execute("""SELECT COALESCE(SUM(a.tokens),0) FROM attempts a
                WHERE a.id IN (SELECT attempt_id FROM government_role_calls WHERE dispatch_id=?)""",
                                         (auth["dispatchId"],)).fetchone()[0]
        else:
            reported_tokens = db.execute("""SELECT COALESCE(SUM(tokens),0) FROM attempts
                WHERE id=? OR id IN (SELECT attempt_id FROM government_role_calls WHERE dispatch_id=?)""",
                                         (dispatch_attempt[0], auth["dispatchId"])).fetchone()[0]
        if reported_tokens >= grant_token_threshold:
            raise LimitReached("Coordinator retrospective token threshold")
        task_id = auth["taskId"]
        purpose = "task" if role_slot["phase"] == "execute" else "review"
        attempt_id = ledger._reserve(db, task_id, purpose)
        db.execute("""INSERT INTO government_role_calls(
                    call_id,dispatch_id,attempt_id,authorization_sha256,invocation_sha256,input_digest,
                    run_id,nonce_sha256,slot_id,phase,evidence_path,status,receipt,reserved_wall_seconds,wall_seconds)
                    VALUES(?,?,?,?,?,?,?,?,?,?,?,'reserved',NULL,?,NULL)""",
                   (call_id, auth["dispatchId"], attempt_id, authorization_sha, invocation_sha,
                    invocation["inputDigest"], invocation["runId"], digest(invocation["nonce"].encode()),
                    role_slot["slotId"], role_slot["phase"], str(evidence_path),
                    float(requested_timeout or 0)))
    return call_id, attempt_id


def _finish(ledger: Ledger, call_id: str, attempt_id: str, status: str,
            turns: int | None, tokens: int | None, receipt: dict) -> None:
    with ledger.transaction() as db:
        changed = db.execute("UPDATE attempts SET end=?,status=?,turns=?,tokens=?,receipt=? WHERE id=? AND end IS NULL",
                             (time.time(), status, turns, tokens, json.dumps([receipt], sort_keys=True), attempt_id)).rowcount
        if changed != 1:
            raise ValueError("role reservation is unknown or already finalized")
        wall_seconds = receipt.get("wallSeconds")
        if type(wall_seconds) not in (int, float) or wall_seconds < 0:
            wall_seconds = None
        changed = db.execute("UPDATE government_role_calls SET status=?,receipt=?,wall_seconds=? "
                             "WHERE call_id=? AND status='reserved'",
                             (status, json.dumps(receipt, sort_keys=True), wall_seconds, call_id)).rowcount
        if changed != 1:
            raise ValueError("role invocation receipt is unknown or already finalized")


def _effective_timeout(ledger: Ledger, request: dict, authorization: dict, authority,
                      dispatch_record: dict, requested_seconds: int) -> float:
    """Cap one delegate by every already-bound outer/shared wall deadline."""
    now = time.time()
    grant = authority.grant
    if (type(request.get("wallSeconds")) not in (int, float) or
            type(grant.get("maxSessionWallSeconds")) not in (int, float)):
        raise ValueError("outer Request and Coordinator session wall limits are required")
    attempt_id = dispatch_record.get("attempt")
    with ledger.transaction() as db:
        trial = db.execute("SELECT start,stopped FROM trial").fetchone()
        task = db.execute("SELECT start FROM tasks WHERE id=?", (request["task"]["id"],)).fetchone()
        outer = (db.execute("SELECT start FROM attempts WHERE id=?", (attempt_id,)).fetchone()
                 if attempt_id is not None else
                 db.execute("SELECT start FROM controller_runs WHERE dispatch_id=?",
                            (request["dispatchId"],)).fetchone())
    if not trial or not task or not outer or trial[1]:
        raise LimitReached("shared wall-clock authority is stopped or unavailable")
    bounds = [float(requested_seconds), float(grant["maxSessionWallSeconds"]),
              float(authorization["expiresAt"]) - now, float(grant["expiresAt"]) - now,
              float(request["wallSeconds"]) - (now - outer[0]),
              ledger.limits["trialWallSeconds"] - (now - trial[0]),
              ledger.limits["taskWallSeconds"] - (now - task[0])]
    remaining = min(bounds)
    if remaining <= 0:
        raise LimitReached("delegate wall deadline exhausted before launch")
    return min(180.0, remaining)


def run_role(invocation_raw: bytes, authorization_raw: bytes, authorization_sha256: str,
             authorization_path: str, request_raw: bytes, authority, *,
             expected_slot: str | None = None,
             controller_context=None,
             role_resolver=None,
             cwd: str | None = None) -> tuple[bytes, dict]:
    """Validate one per-call operator grant, reserve, then execute one delegate.

    `authority` is the existing validated dispatch.Authority. It revalidates the
    exact outer Request/Grant/Protocol and returns the already captured released
    inputs; its Ledger is the existing shared trial ledger. This helper never
    retries. An interrupted/ambiguous reservation stays occupied.
    """
    invocation = parse_invocation(invocation_raw)
    request, captured = authority.validate(request_raw)
    fixture_bounds = (native_controller.validate_native_fixture_grant(request, captured)
                      if request.get("nativeFixtureGrant") is not None else None)
    auth, role_slot, runtime = _role_auth(authorization_raw, authorization_sha256, request, request_raw,
                                         authority, captured, invocation, authorization_path, expected_slot,
                                         role_resolver=role_resolver)
    if not (time.time() < auth["expiresAt"]):
        raise ValueError("operator role authorization expired before delegate reservation")
    ledger = authority.ledger("run_task", dispatch_id=request["dispatchId"])
    record = ledger.dispatch_record(request["dispatchId"])
    if (not record or record.get("phase") != "launching" or record.get("result") is not None or
            record.get("request") != request_raw or record.get("execution_sha") != _execution_sha(request)):
        raise ValueError("active operator-authorized outer dispatch binding required")
    if record.get("attempt") is None:
        if (controller_context is None or controller_context.request_raw != request_raw or
                controller_context.authority.grant_sha != authority.grant_sha or
                controller_context.authorization_raw != authorization_raw or
                controller_context.authorization_path.resolve() != Path(authorization_path).resolve()):
            raise ValueError("validated controller bootstrap context required for native role calls")
    delegate = role_slot["delegate"]
    argv = delegate["argv"]
    if argv[0] != delegate.get("command"):
        raise ValueError("delegate argv does not begin with its bound command")
    raw_evidence_parent = auth.get("roleEvidenceDirectory")
    if not isinstance(raw_evidence_parent, str) or not Path(raw_evidence_parent).is_absolute():
        raise ValueError("absolute role evidence directory required")
    evidence_parent = Path(raw_evidence_parent).resolve()
    authority_paths = getattr(authority, "paths", [])
    if not isinstance(authority_paths, list):
        raise ValueError("validated Authority path set is malformed")
    product = request.get("product", {}).get(request.get("arm"), {})
    protected_paths = [request["actorRepository"], request["evidenceDirectory"], authority.ledger_path,
                       authorization_path, auth["requestPath"], *authority_paths]
    protected_paths.extend(value for value in (product.get("queueStateDirectory"),
                            runtime.get("stateDirectory"), runtime.get("recordStore"),
                            runtime.get("privateLogs")) if value is not None)
    protected_paths = [str(path) if isinstance(path, Path) else path for path in protected_paths]
    for path in protected_paths:
        if not isinstance(path, str) or not Path(path).is_absolute():
            raise ValueError("protected bridge paths must be absolute")
    for protected in (Path(path).resolve() for path in protected_paths):
        if evidence_parent == protected or protected in evidence_parent.parents or evidence_parent in protected.parents:
            raise ValueError("role evidence directory overlaps Actor, trial evidence, ledger, or authorization")
    # Do not create anything until all authorization and path-overlap checks pass.
    invocation_sha = digest(invocation_raw)
    call_id = digest(encoded({"dispatchId": request["dispatchId"], "runId": invocation["runId"],
                              "nonce": invocation["nonce"], "inputDigest": invocation["inputDigest"]}))
    evidence_path = evidence_parent / call_id
    # Reserve and persist the unique invocation identity before any delegate effect.
    effective_timeout = _effective_timeout(ledger, request, auth, authority, record,
                                          delegate["timeoutSeconds"])
    r3_binding = request.get("nativeFixtureR3Grant")
    if r3_binding is not None:
        validated_r3 = fixture_bounds.get("r3Grant") if isinstance(fixture_bounds, dict) else None
        if (not isinstance(validated_r3, dict) or
                validated_r3.get("grantKey") != r3_binding.get("sourceKey")):
            raise ValueError("R3 role call requires the exact validated additive grant before reservation")
        from native_fixture_budget import validate_r3_entry_gate
        validate_r3_entry_gate(validated_r3)
    r4_binding = request.get("nativeFixtureR4Grant")
    if r4_binding is not None:
        validated_r4 = fixture_bounds.get("r4Grant") if isinstance(fixture_bounds, dict) else None
        if (request.get("dispatchId") != "government-native-serialization-r4" or
                request.get("arm") != "government" or
                request.get("nativeFixtureR3Grant") is not None or
                request.get("nativeFixtureCorrection") is not None or
                not isinstance(validated_r4, dict) or
                validated_r4.get("grantKey") != r4_binding.get("sourceKey")):
            raise ValueError("R4 role call requires the exact separate Government grant before reservation")
        from native_fixture_budget import validate_r4_entry_gate
        validate_r4_entry_gate(validated_r4)
        native_controller.validate_r4_delegate_authorization(request, authorization_raw, validated_r4)
    fixture_profile = native_controller.native_profile(request)
    validated_profile = None
    if fixture_profile is not None:
        profile_binding = request.get(fixture_profile.marker)
        validated_profile = (fixture_bounds.get(f"{fixture_profile.name}Grant")
                             if isinstance(fixture_bounds, dict) else None)
        if (not isinstance(validated_profile, dict) or
                validated_profile.get("grantKey") != profile_binding.get("sourceKey")):
            raise ValueError(f"{fixture_profile.name.upper()} role call requires its exact grant before reservation")
        native_controller.validate_profile_live_gate(validated_profile, fixture_profile)
        native_controller.validate_profile_delegate_authorization(
            request, authorization_raw, validated_profile, fixture_profile)
    reserved_id, attempt_id = _reserve(ledger, auth, invocation, invocation_sha,
                                       authorization_sha256, role_slot, evidence_path,
                                       authority.grant["retrospectiveTokenThreshold"],
                                       fixture_role_limit=fixture_bounds["maxRoleStarts"] if fixture_bounds else None,
                                       fixture_parallel_limit=fixture_bounds["maxRoleParallel"] if fixture_bounds else None,
                                       fixture_process_limit=fixture_bounds["maxRoleProcessSeconds"] if fixture_bounds else None,
                                       requested_timeout=effective_timeout if fixture_bounds else None)
    if reserved_id != call_id:
        raise AssertionError("internal role invocation identity mismatch")
    evidence_parent.mkdir(parents=True, exist_ok=True)
    if r4_binding is not None:
        # The reservation may have taken time; revocation immediately before
        # process creation still stops the delegate without a retry.
        from native_fixture_budget import validate_r4_entry_gate
        validate_r4_entry_gate(validated_r4)
        native_controller.validate_r4_delegate_authorization(request, authorization_raw, validated_r4)
    if fixture_profile is not None:
        # A reservation may take time; the current profile slot and delegate
        # pins must still hold at the final delegate Popen boundary.
        native_controller.validate_profile_live_gate(validated_profile, fixture_profile)
        native_controller.validate_profile_delegate_authorization(
            request, authorization_raw, validated_profile, fixture_profile)
    environment = native_controller.strip_bootstrap_environment(os.environ)
    result = bounded(argv, str(Path(cwd or os.getcwd()).resolve()), evidence_path,
                     effective_timeout, stdin=invocation_raw, env=environment,
                     max_log_bytes=min(delegate["maxStdoutBytes"], delegate["maxStderrBytes"]))
    stdout_path = evidence_path / "stdout.log"
    stderr_path = evidence_path / "stderr.log"
    process_path = evidence_path / "process.json"
    stdout = stdout_path.read_bytes()
    process_raw = process_path.read_bytes()
    response, tokens = _response_usage(stdout, invocation) if result["returnCode"] == 0 else (None, None)
    protocol_ok = response is not None
    status = "protocol-echo-valid" if result["returnCode"] == 0 and protocol_ok else "failed"
    fixture_accounting = (_known_no_provider_fixture_accounting(request, auth, role_slot, fixture_bounds)
                          if tokens is None else None)
    ledger_turns, ledger_tokens = (0, 0) if fixture_accounting is not None else (None, tokens)
    receipt = {"apiVersion": "markitect.scientist-role-receipt/v1alpha1", "callId": call_id,
               "dispatchId": request["dispatchId"], "trialId": request["trialId"],
               "taskId": request["task"]["id"], "authorizationSha256": authorization_sha256,
               "outerRequestSha256": digest(request_raw), "invocationSha256": invocation_sha,
               "inputDigest": invocation["inputDigest"], "runId": invocation["runId"],
               "nonceSha256": digest(invocation["nonce"].encode()), "slotId": role_slot["slotId"],
               "phase": role_slot["phase"], "responseRole": invocation["request"]["role"],
               "delegateArgvSha256": digest(encoded(argv)), "returnCode": result["returnCode"],
               "wallSeconds": result["wallSeconds"],
               "effectiveTimeoutSeconds": effective_timeout,
               "stdoutPath": str(stdout_path.resolve()), "stderrPath": str(stderr_path.resolve()),
               "processReceiptPath": str(process_path.resolve()),
               "responseSha256": digest(stdout), "stderrSha256": digest(stderr_path.read_bytes()),
               "processReceiptSha256": digest(process_raw), "status": status,
               "providerRequests": None, "providerTurns": None,
               "reportedInputPlusOutputTokens": tokens,
               "usageSource": "agentexec-provider-reported" if tokens is not None else "unknown"}
    if fixture_accounting is not None:
        receipt["fixtureAccounting"] = fixture_accounting
    # Missing request/turn telemetry is preserved as unknown; the Ledger's bound
    # measurement profile decides whether another invocation can be admitted,
    # except for the exact source-authorized no-provider fixture branch above.
    _finish(ledger, call_id, attempt_id, status, ledger_turns, ledger_tokens, receipt)
    if result["returnCode"] != 0:
        raise RuntimeError("authorized Government delegate process failed; reservation remains consumed")
    if not protocol_ok:
        raise ValueError("delegate response did not echo the exact agentexec invocation identity")
    return stdout, receipt


def main(argv: list[str] | None = None) -> int:
    """Run one already-reserved native agentexec invocation through the Ledger."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--authorization", required=True)
    parser.add_argument("--authorization-sha256", required=True)
    parser.add_argument("--evidence", required=True)
    parser.add_argument("--slot")
    parser.add_argument("--diagnostics-config")
    parser.add_argument("--diagnostics-sha256")
    args = parser.parse_args(argv)
    invocation_raw = sys.stdin.buffer.read()
    context = native_controller.load_context(invocation_raw=invocation_raw)
    fixture_profile = native_controller.native_profile(context.request)
    diagnostic_grant = (context.request.get(fixture_profile.marker) if fixture_profile is not None else
                        context.request.get("nativeFixtureR4Grant") or
                        context.request.get("nativeFixtureR3Grant") or
                        context.request.get("nativeFixtureCorrection"))
    if diagnostic_grant is not None:
        if (_NATIVE_DIAGNOSTICS is None or
                _NATIVE_DIAGNOSTICS["config"]["correction"] != diagnostic_grant or
                Path(_NATIVE_DIAGNOSTICS["config"]["requestPath"]).resolve() !=
                Path(json.loads(context.authorization_raw)["requestPath"]).resolve()):
            raise ValueError("corrected wrapper requires exact early diagnostic binding")
    if context.request.get("nativeFixtureR3Grant") is not None:
        bounds = context.native_fixture_bounds
        validated = bounds.get("r3Grant") if isinstance(bounds, dict) else None
        if not isinstance(validated, dict) or validated.get("grantKey") != diagnostic_grant.get("sourceKey"):
            raise ValueError("wrapper requires the exact validated R3 fixture grant before delegation")
    if context.request.get("nativeFixtureR4Grant") is not None:
        bounds = context.native_fixture_bounds
        validated = bounds.get("r4Grant") if isinstance(bounds, dict) else None
        if (context.request.get("dispatchId") != "government-native-serialization-r4" or
                context.request.get("arm") != "government" or
                not isinstance(validated, dict) or
                validated.get("grantKey") != diagnostic_grant.get("sourceKey")):
            raise ValueError("wrapper requires the exact validated R4 grant before delegation")
        from native_fixture_budget import validate_r4_entry_gate
        validate_r4_entry_gate(validated)
    if fixture_profile is not None:
        bounds = context.native_fixture_bounds
        validated = bounds.get(f"{fixture_profile.name}Grant") if isinstance(bounds, dict) else None
        if (not isinstance(validated, dict) or
                validated.get("grantKey") != diagnostic_grant.get("sourceKey")):
            raise ValueError(f"wrapper requires the exact validated {fixture_profile.name.upper()} grant before delegation")
        native_controller.validate_profile_live_gate(validated, fixture_profile)
    if (Path(args.authorization).resolve(strict=True) != context.authorization_path or
            args.authorization_sha256 != context.authorization_sha256):
        raise ValueError("wrapper argv differs from the digest-bound controller authorization")
    expected_evidence = json.loads(context.authorization_raw)["roleEvidenceDirectory"]
    if Path(args.evidence).resolve() != Path(expected_evidence).resolve():
        raise ValueError("wrapper evidence path differs from the operator authorization")
    role_resolver = None
    if context.request["arm"] == "classic":
        import classic_integration
        role_resolver = classic_integration.resolve_role
    elif not args.slot:
        raise ValueError("Government role wrapper requires its fixed --slot argument")
    response, receipt = run_role(invocation_raw, context.authorization_raw,
                                 context.authorization_sha256, str(context.authorization_path),
                                 context.request_raw, context.authority, expected_slot=args.slot,
                                 controller_context=context, role_resolver=role_resolver,
                                 cwd=context.request["actorRepository"])
    sys.stdout.buffer.write(response)
    sys.stderr.write(json.dumps({"roleReceipt": receipt}, sort_keys=True) + "\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
