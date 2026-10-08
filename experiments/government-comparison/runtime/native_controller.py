"""Acyclic bootstrap and accounting helpers for native agentexec wrappers.

The outer controller writes a digest-bound bundle only after validating the
exact v1.2 Request and Coordinator Authority. Native role wrappers require the
explicit path+digest environment pair and an already-claimed common-ledger
controller row. This module makes no provider or product calls.
"""
from __future__ import annotations

import hashlib
import json
import os
from pathlib import Path
import re
import time
from typing import NamedTuple

BOOTSTRAP_API = "markitect.scientist-native-controller-bootstrap/v1alpha1"
BOOTSTRAP_PATH_ENV = "MARKITECT_SCIENTIST_BOOTSTRAP_PATH"
BOOTSTRAP_SHA_ENV = "MARKITECT_SCIENTIST_BOOTSTRAP_SHA256"
MAX_BOOTSTRAP_BYTES = 1024 * 1024
_SHA = re.compile(r"^[0-9a-f]{64}$")
NATIVE_FIXTURE_GRANT_KEY = "native-s1-integration-fixtures-20261008"
NATIVE_FIXTURE_SOURCE_THREAD = "01a11367-a781-7683-a20f-46e12614dcb4"


def digest(raw: bytes) -> str:
    return hashlib.sha256(raw).hexdigest()


def _strict_json(raw: bytes, label: str):
    def pairs(items):
        result = {}
        for key, value in items:
            if key in result:
                raise ValueError(f"{label} has duplicate JSON keys")
            result[key] = value
        return result
    try:
        return json.loads(raw, object_pairs_hook=pairs)
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise ValueError(f"{label} is not valid JSON") from exc


def _bound_file(item, label, *, max_bytes=64 * 1024 * 1024):
    if not isinstance(item, dict) or set(item) != {"path", "sha256"}:
        raise ValueError(f"{label} path and digest binding required")
    path, expected = item["path"], item["sha256"]
    target = Path(path) if isinstance(path, str) else Path()
    if not target.is_absolute() or not _SHA.fullmatch(expected if isinstance(expected, str) else ""):
        raise ValueError(f"{label} requires an absolute path and lowercase SHA-256")
    if target.is_symlink():
        raise ValueError(f"{label} cannot be a symlink")
    resolved = target.resolve(strict=True)
    if not resolved.is_file() or resolved.stat().st_size > max_bytes:
        raise ValueError(f"{label} is not a bounded regular file")
    raw = resolved.read_bytes()
    if digest(raw) != expected:
        raise ValueError(f"{label} digest mismatch")
    return resolved, raw


def validate_native_fixture_grant(request, captured, product_bound=None):
    """Bind an explicitly released, zero-real-Actor fixture grant to a native pin."""
    if request.get("mode") != "mechanical":
        raise ValueError("no-provider native fixture grant cannot authorize live/provider mode")
    expected_r3_arm = {
        "government-native-contract-corrected-r3": "government",
        "classic-native-contract-corrected-r3": "classic",
    }.get(request.get("dispatchId"))
    if expected_r3_arm is not None and (
            request.get("arm") != expected_r3_arm or
            request.get("nativeFixtureR3Grant") is None or
            request.get("nativeFixtureCorrection") is not None):
        raise ValueError("R3 dispatch requires its exact R3 grant binding and cannot reuse the R2 correction binding")
    binding = request.get("nativeFixtureGrant")
    if (not isinstance(binding, dict) or set(binding) != {"path", "sha256", "sourceKey"} or
            binding.get("sourceKey") != NATIVE_FIXTURE_GRANT_KEY):
        raise ValueError("explicit native fixture source grant path/SHA/key required")
    path, raw = _bound_file({"path": binding.get("path"), "sha256": binding.get("sha256")},
                            "native fixture source grant", max_bytes=1024 * 1024)
    released = {str(Path(item["path"]).resolve()): item.get("sha256")
                for item in request.get("releasedInputs", []) if isinstance(item, dict) and item.get("path")}
    captured_raw = next((value for source, value in captured.items()
                         if Path(source).resolve() == path), None)
    if released.get(str(path)) != binding["sha256"] or captured_raw != raw:
        raise ValueError("native fixture source grant must be an exact released Request input")
    grant_doc = _strict_json(raw, "native fixture source grant")
    grant = grant_doc.get("grant")
    per_product = grant.get("perProduct") if isinstance(grant, dict) else None
    if (grant_doc.get("sourceThreadId") != NATIVE_FIXTURE_SOURCE_THREAD or
            grant_doc.get("sourceJsonPointer") != "threads[name=Scientist].evidence.nativeIntegrationPreparationGrant" or
            not isinstance(grant_doc.get("sourceCoordinationPath"), str)):
        raise ValueError("native fixture grant source-thread provenance mismatch")
    source_snapshot = Path(grant_doc["sourceCoordinationPath"])
    if not source_snapshot.is_absolute() or digest(source_snapshot.read_bytes()) != grant_doc.get("sourceCoordinationSha256"):
        raise ValueError("native fixture grant source coordination snapshot mismatch")
    source = _strict_json(source_snapshot.read_bytes(), "native fixture source coordination snapshot")
    scientist = next((item for item in source.get("threads", [])
                      if isinstance(item, dict) and item.get("name") == "Scientist"), None)
    if not scientist or scientist.get("evidence", {}).get("nativeIntegrationPreparationGrant") != grant:
        raise ValueError("native fixture grant differs from its bound source snapshot")
    if (grant.get("key") != NATIVE_FIXTURE_GRANT_KEY or grant.get("realActorStartsAuthorized") != 0 or
            grant.get("providerCallsAuthorized") != 0 or grant.get("studyCellsAuthorized") != 0 or
            not isinstance(per_product, dict) or
            {key: per_product.get(key) for key in ("positiveCases", "nativeCliOrControllerStartsMaximum",
                 "deterministicRoleStartsMaximum", "maxParallel", "totalProcessSecondsMaximum")} !=
            {"positiveCases": 1, "nativeCliOrControllerStartsMaximum": 8,
             "deterministicRoleStartsMaximum": 12, "maxParallel": 2,
             "totalProcessSecondsMaximum": 1200}):
        raise ValueError("native fixture grant limits differ from the exact finite provider-free allocation")
    name = request.get("arm", "").title()
    products = [item for item in grant.get("products", []) if isinstance(item, dict) and item.get("name") == name]
    if len(products) != 1:
        raise ValueError("native fixture source grant must bind exactly one product candidate")
    product = products[0]
    product_request = request.get("product", {}).get(request.get("arm"), {})
    if request["arm"] == "classic":
        from classic import EXPECTED_HELD_SOURCE, EXPECTED_BINARY_SHA256
        expected_source = EXPECTED_HELD_SOURCE
        expected_binary = EXPECTED_BINARY_SHA256
        executable = product_bound.get("executable") if product_bound else product_request.get("executable", {}).get("path")
    else:
        expected_source = product_bound.get("sourceCommit") if product_bound else product_request.get("executable", {}).get("sourceCommit")
        expected_binary = product_request.get("executable", {}).get("sha256")
        executable = product_bound.get("executable") if product_bound else product_request.get("executable", {}).get("path")
    if (product.get("sourceSha") != expected_source or product.get("binarySha256") != expected_binary or
            not executable or digest(Path(executable).read_bytes()) != expected_binary):
        raise ValueError("native fixture source grant product pin differs from the bound executable/source")
    result = {"path": str(path), "sha256": binding["sha256"], "sourceKey": binding["sourceKey"],
              "product": name, "sourceSha": expected_source, "binarySha256": expected_binary,
              "maxRoleStarts": per_product["deterministicRoleStartsMaximum"],
              "maxRoleParallel": per_product["maxParallel"],
              "maxRoleProcessSeconds": per_product["totalProcessSecondsMaximum"]}
    r3_binding = request.get("nativeFixtureR3Grant")
    if request.get("nativeFixtureCorrection") is not None and r3_binding is not None:
        raise ValueError("an R3 Request cannot reuse the R2 correction binding")
    if request.get("nativeFixtureCorrection") is not None:
        from native_fixture_budget import CORRECTION_KEY, validate_correction_binding
        correction = validate_correction_binding(request, path, binding["sha256"])
        result.update(correctionKey=CORRECTION_KEY, correctionSha256=correction["sha256"],
                      maxRoleStarts=correction["grant"]["limits"][name.lower()][
                          "maxAdditionalRoleInvocationsIncludingFailedWrapperStarts"])
    elif r3_binding is not None:
        from native_fixture_budget import validate_r3_grant_binding
        validated = validate_r3_grant_binding(request, path, binding["sha256"])
        expected_native_starts, expected_seconds = ((2, 300) if name == "Government" else (5, 750))
        if (not isinstance(validated, dict) or
                validated.get("grantKey") != r3_binding.get("sourceKey") or
                validated.get("grantKey") != "native-s1-contract-corrected-integration-20261008-r3" or
                validated.get("maxRoleStarts") != 6 or
                validated.get("maxDeterministicDelegates") != 6 or
                validated.get("maxNativeStarts") != expected_native_starts or
                validated.get("maxReservedSessionSeconds") != expected_seconds):
            raise ValueError("validated R3 bounds differ from the exact additive native allocation")
        result.update(grantKey=validated["grantKey"], maxRoleStarts=validated["maxRoleStarts"],
                      maxDeterministicDelegates=validated["maxDeterministicDelegates"],
                      maxNativeStarts=validated["maxNativeStarts"],
                      maxReservedSessionSeconds=validated["maxReservedSessionSeconds"],
                      r3Grant=validated)
    return result


class ControllerContext(NamedTuple):
    authority: object
    request: dict
    request_raw: bytes
    captured_inputs: dict
    authorization_path: Path
    authorization_raw: bytes
    authorization_sha256: str
    bootstrap_path: Path
    bootstrap_sha256: str
    controller_record: dict
    request_path: Path
    native_fixture_bounds: dict | None


def write_bundle(path, *, request_path, request_raw, authority, authorization_path,
                 authorization_sha256, allow_live=False):
    """Write a closed bundle; caller must already have validated Authority/Request."""
    from dispatch import digest as dispatch_digest, execution_sha
    request_target = Path(request_path).resolve(strict=True)
    if request_target.read_bytes() != request_raw:
        raise ValueError("outer Request bytes changed before controller bootstrap")
    authorization_target = Path(authorization_path).resolve(strict=True)
    authorization_raw = authorization_target.read_bytes()
    if dispatch_digest(authorization_raw) != authorization_sha256:
        raise ValueError("operator role authorization digest mismatch")
    request, captured = authority.validate(request_raw)
    binding = request.get("product", {}).get(request.get("arm"), {}).get("roleAuthorization")
    captured_authorization = next((content for source, content in captured.items()
                                   if Path(source).resolve() == authorization_target), None)
    if (not isinstance(binding, dict) or Path(binding.get("path", "")).resolve() != authorization_target or
            binding.get("sha256") != authorization_sha256 or
            captured_authorization != authorization_raw):
        raise ValueError("role authorization is not an exact released Request input")
    payload = {
        "apiVersion": BOOTSTRAP_API,
        "trialId": request["trialId"],
        "dispatchId": request["dispatchId"],
        "executionSha256": execution_sha(request),
        "allowLive": bool(allow_live),
        "request": {"path": str(request_target), "sha256": dispatch_digest(request_raw)},
        "grant": {"path": str(Path(authority.paths[0]).resolve()), "sha256": authority.grant_sha},
        "protocol": {"path": str(Path(authority.paths[1]).resolve()), "sha256": authority.protocol_sha},
        "roleAuthorization": {"path": str(authorization_target), "sha256": authorization_sha256},
    }
    raw = json.dumps(payload, sort_keys=True, separators=(",", ":")).encode()
    target = Path(path)
    if not target.is_absolute():
        raise ValueError("controller bootstrap output path must be absolute")
    target.parent.mkdir(parents=True, exist_ok=True)
    with target.open("xb") as stream:
        stream.write(raw)
        stream.flush()
        os.fsync(stream.fileno())
    return target.resolve(), digest(raw)


def _classic_materialized_workspace_allowed(request, invocation_raw, ledger):
    """Permit dirty workspace binding only for a verifier after this dispatch's completed Apply."""
    if request.get("arm") != "classic" or invocation_raw is None:
        return False
    import classic_integration
    from government_roles import parse_invocation

    invocation = parse_invocation(invocation_raw)
    projection = classic_integration.resolve_role(invocation)
    if projection.get("phase") != "review":
        return False
    with ledger.transaction() as db:
        row = db.execute("SELECT status FROM controller_processes "
                         "WHERE dispatch_id=? AND action='apply'",
                         (request["dispatchId"],)).fetchone()
    return bool(row and row[0] == "completed")


def load_context(*, env=None, invocation_raw=None):
    """Load and validate only an explicitly injected controller bundle."""
    from dispatch import Authority, execution_sha
    env = os.environ if env is None else env
    path_text, expected = env.get(BOOTSTRAP_PATH_ENV), env.get(BOOTSTRAP_SHA_ENV)
    if not path_text or not expected:
        raise ValueError("explicit controller bootstrap path and SHA-256 environment pair required")
    if not _SHA.fullmatch(expected):
        raise ValueError("controller bootstrap SHA-256 malformed")
    path = Path(path_text)
    if not path.is_absolute() or path.is_symlink():
        raise ValueError("controller bootstrap path must be absolute and non-symlink")
    path = path.resolve(strict=True)
    if not path.is_file() or path.stat().st_size > MAX_BOOTSTRAP_BYTES:
        raise ValueError("controller bootstrap must be a bounded regular file")
    raw = path.read_bytes()
    if digest(raw) != expected:
        raise ValueError("controller bootstrap digest mismatch")
    bundle = _strict_json(raw, "controller bootstrap")
    fields = {"apiVersion", "trialId", "dispatchId", "executionSha256", "allowLive",
              "request", "grant", "protocol", "roleAuthorization"}
    if not isinstance(bundle, dict) or set(bundle) != fields or bundle.get("apiVersion") != BOOTSTRAP_API:
        raise ValueError("controller bootstrap fields/API mismatch")
    if type(bundle.get("allowLive")) is not bool:
        raise ValueError("controller live gate must be an explicit boolean")
    grant_path, _ = _bound_file(bundle["grant"], "Coordinator grant")
    protocol_path, _ = _bound_file(bundle["protocol"], "frozen Protocol")
    request_path, request_raw = _bound_file(bundle["request"], "outer Request")
    authorization_path, authorization_raw = _bound_file(bundle["roleAuthorization"], "role authorization",
                                                         max_bytes=4 * 1024 * 1024)
    authority = Authority(grant_path, bundle["grant"]["sha256"], protocol_path,
                          bundle["protocol"]["sha256"], allow_live=bundle["allowLive"])
    request, captured = authority.validate(request_raw)
    if (request["operation"] != "run_task" or request["arm"] not in {"government", "classic"} or
            request["trialId"] != bundle["trialId"] or request["dispatchId"] != bundle["dispatchId"] or
            execution_sha(request) != bundle["executionSha256"]):
        raise ValueError("bootstrap Request identity/operation mismatch")
    ledger = authority.ledger("run_task", dispatch_id=request["dispatchId"])
    record = ledger.dispatch_record(request["dispatchId"])
    if (not record or record.get("attempt") is not None or record.get("phase") != "launching" or
            record.get("result") is not None or record.get("request") != request_raw or
            record.get("execution_sha") != execution_sha(request)):
        raise ValueError("claimed controller dispatch is required before native role execution")
    with ledger.transaction() as db:
        controller = db.execute("SELECT task,status FROM controller_runs WHERE dispatch_id=?",
                                (request["dispatchId"],)).fetchone()
    if not controller or controller[1] not in {"reserved", "running"} or controller[0] != request["task"]["id"]:
        raise ValueError("active common-ledger controller timing record required")

    allow_materialized_workspace = _classic_materialized_workspace_allowed(
        request, invocation_raw, ledger)

    product_bound = None
    if request["arm"] == "classic":
        import classic_integration
        product_bound = classic_integration.bind_request(
            request, request_path=request_path,
            allow_materialized_workspace=allow_materialized_workspace)
    fixture_bounds = None
    if request.get("nativeFixtureGrant") is not None:
        fixture_bounds = validate_native_fixture_grant(request, captured, product_bound)
    product_binding = request.get("product", {}).get(request.get("arm"), {}).get("roleAuthorization")
    captured_authorization = next((content for source, content in captured.items()
                                   if Path(source).resolve() == authorization_path), None)
    if (not isinstance(product_binding, dict) or Path(product_binding.get("path", "")).resolve() != authorization_path or
            product_binding.get("sha256") != bundle["roleAuthorization"]["sha256"] or
            captured_authorization != authorization_raw):
        raise ValueError("bootstrap role authorization is not the released Request-bound file")
    return ControllerContext(authority, request, request_raw, captured, authorization_path,
                             authorization_raw, bundle["roleAuthorization"]["sha256"],
                             path, expected, record, request_path, fixture_bounds)


CLASSIC_ACTIONS = ("execute", "apply", "verify", "audit", "apply-replay")
CLASSIC_MAX_PROCESS_SECONDS = 38


class ClassicControllerSession:
    """One claimed controller booking across a review-gated Classic sequence."""
    def __init__(self, context, ledger, bound, evidence, reports, action_sequence):
        self.context = context
        self.ledger = ledger
        self.bound = bound
        self.evidence = evidence
        self.reports = reports
        self.action_sequence = tuple(action_sequence)
        self.captures = []
        self.process_records = []
        self.review = None
        self.finished = False


def begin_classic_controller(request_path, authority, *, request_raw=None, captured=None):
    """Validate/bind and claim one finite mechanical Classic controller flow.

    No product process starts here. The Request must explicitly bind the exact
    five-action fixture sequence; each later step separately consumes the
    parent's durable FixtureBudget before its process is launched.
    """
    import classic_integration
    import dispatch
    import government_roles

    request_target = Path(request_path).resolve(strict=True)
    raw = request_target.read_bytes() if request_raw is None else request_raw
    if request_target.read_bytes() != raw:
        raise ValueError("Classic Request changed before controller start")
    request, authority_captured = authority.validate(raw)
    if (request.get("arm") != "classic" or request.get("operation") != "run_task" or
            request.get("mode") != "mechanical"):
        raise ValueError("only the explicitly authorized mechanical Classic fixture flow is enabled")
    product = request.get("product", {}).get("classic")
    if not isinstance(product, dict) or product.get("controllerActions") != list(CLASSIC_ACTIONS):
        raise ValueError("Classic Request must bind the exact finite controllerActions sequence")
    if captured is not None and captured != authority_captured:
        raise ValueError("caller captured inputs differ from validated Request inputs")
    bound = classic_integration.bind_request(request, request_path=request_target)
    validate_native_fixture_grant(request, authority_captured, bound)
    role_binding = product.get("roleAuthorization")
    auth_path = Path(role_binding["path"]).resolve(strict=True)
    auth_raw = auth_path.read_bytes()
    if dispatch.digest(auth_raw) != role_binding.get("sha256"):
        raise ValueError("Classic role authorization changed after Request validation")
    government_roles.preflight_authorization(request, raw, authority, authority_captured,
                                            str(auth_path), auth_raw, role_binding["sha256"])
    auth_value = json.loads(auth_raw)
    if auth_value.get("fixtureAuthorization") != government_roles.NATIVE_FIXTURE_AUTH:
        raise ValueError("approved Classic fixture source-grant provenance is required")
    ledger = authority.ledger("run_task", dispatch_id=request["dispatchId"])
    if ledger.dispatch_record(request["dispatchId"]):
        raise ValueError("Classic dispatch is already booked; relaunch/replay through a new authorization is forbidden")
    argv = [str(bound["executable"]), "classic-native-controller-sequence"]
    ledger.reserve_controller_dispatch(request["dispatchId"], dispatch.execution_sha(request), raw, argv,
                                       request["task"]["id"], request["purpose"],
                                       authority.grant["maxActorSessions"])
    if not ledger.claim_dispatch(request["dispatchId"], "launching"):
        raise ValueError("Classic controller claim lost; product launch is forbidden")
    try:
        evidence = Path(request["evidenceDirectory"])
        evidence.mkdir(parents=True, exist_ok=False)
        (evidence / "request.json").write_bytes(raw)
        released = evidence / "released-inputs"
        released.mkdir()
        for index, item in enumerate(request["releasedInputs"]):
            (released / str(index)).write_bytes(authority_captured[item["path"]])
        bundle_path, bundle_sha = write_bundle(
            evidence / "controller-bootstrap.json", request_path=request_target, request_raw=raw,
            authority=authority, authorization_path=auth_path,
            authorization_sha256=role_binding["sha256"], allow_live=False)
        context = load_context(env={BOOTSTRAP_PATH_ENV: str(bundle_path), BOOTSTRAP_SHA_ENV: bundle_sha})
        reports = evidence / "classic-native"
        reports.mkdir()
        return ClassicControllerSession(context, ledger, bound, evidence, reports, product["controllerActions"])
    except BaseException as exc:
        # A durable claim may not be left in `launching` when bootstrap fails before the
        # first product process. Preserve the error and terminalize the same reservation.
        result = {"status": "incomplete", "inferencePerformed": None,
                  "gaps": ["Classic controller bootstrap failed after reservation; no product process was started"]}
        receipt = {"status": "incomplete", "controllerProcesses": [], "productProcessStarted": False,
                   "bootstrapError": f"{type(exc).__name__}: {exc}"}
        try:
            ledger.finish_controller_dispatch(request["dispatchId"], result, receipt)
        except BaseException as finish_error:
            raise RuntimeError("Classic bootstrap failed and its controller reservation could not be terminalized") from finish_error
        raise


def _classic_step_spec(session, action, external_review):
    import classic_integration

    context = session.context
    request = context.request
    product = request["product"]["classic"]
    revision = request["baseCommit"]
    config_path = product["projectConfig"]["path"]
    runtime_path = product["runtime"]["path"]
    execute_report = session.reports / "execute.json"
    apply_result = session.reports / "apply.json"
    if action in {"apply", "apply-replay"}:
        if action == "apply-replay" and session.review != external_review:
            raise ValueError("stale replay must reuse the exact review object that authorized Apply")
        classic_integration.review_execute(execute_report, external_review)
    if action == "verify":
        apply_value = _strict_json(apply_result.read_bytes(), "Classic Apply result")
        if apply_value.get("status") != "materialized-unverified":
            raise ValueError("fresh Verify requires this controller's successful materialized-unverified Apply result")
    plan = classic_integration.plan_native_action(
        action, packet_path=product["packet"]["path"], repo_path=session.bound["repository"],
        runtime_path=runtime_path, reports_directory=session.reports, revision=revision,
        execute_report_path=execute_report if action in {"apply", "apply-replay"} else None,
        review=external_review if action in {"apply", "apply-replay"} else None,
        apply_result_path=apply_result if action == "verify" else None)
    return plan, plan


def run_classic_step(session, action, *, fixture_budget, external_review=None):
    """Run one parent-budgeted Classic process under an already active controller row."""
    import classic
    import classic_integration
    from process import bounded

    if not isinstance(session, ClassicControllerSession) or session.finished:
        raise ValueError("active Classic controller session required")
    next_index = len(session.captures)
    if next_index >= len(session.action_sequence) or action != session.action_sequence[next_index]:
        raise ValueError("Classic controller action order differs from the exact Request sequence")
    if action in {"apply", "apply-replay"}:
        if external_review is None:
            raise ValueError("Apply requires an out-of-band review after the exact Execute report exists")
        if action == "apply" and session.review is not None:
            raise ValueError("Classic Apply review is immutable once recorded")
    elif external_review is not None:
        raise ValueError("external Execute review is accepted only at Apply steps")
    if session.captures and (session.captures[-1].get("returnCode") != 0 or
                             session.captures[-1].get("stopReason") is not None):
        raise ValueError("prior Classic product process did not finish cleanly; stop the flow")
    if action == "verify" and not any(row["action"] == "apply" for row in session.captures):
        raise ValueError("fresh Verify must follow the same controller's Apply")
    if action == "audit" and not any(row["action"] == "verify" for row in session.captures):
        raise ValueError("Audit must follow the same controller's fresh Verify")
    if action == "apply-replay" and not any(row["action"] == "audit" for row in session.captures):
        raise ValueError("targeted stale Apply replay must follow the completed Audit step")
    spec, planned = _classic_step_spec(session, action, external_review)
    if (Path(spec["argv"][0]).resolve() != session.bound["executable"] or
            classic.sha256_file(session.bound["executable"]) != classic.EXPECTED_BINARY_SHA256):
        raise ValueError("Classic executable changed after preflight")
    if action == "execute":
        runtime_binding = session.context.request["product"]["classic"]["runtime"]
        runtime_path = Path(runtime_binding["path"]).resolve(strict=True)
        runtime_raw = next((raw for path, raw in session.context.captured_inputs.items()
                            if Path(path).resolve() == runtime_path), None)
        if (runtime_raw is None or digest(runtime_raw) != runtime_binding.get("sha256")):
            raise ValueError("Classic runtime must remain the exact released Request input before Execute")
        runtime_value = _strict_json(runtime_raw, "Classic runtime")
        classic_integration.assert_record_store_absent(runtime_value)
    from native_fixture_budget import FixtureBudget
    binding = session.context.request.get("nativeFixtureGrant", {})
    if (not isinstance(fixture_budget, FixtureBudget) or
            fixture_budget.grant_sha != binding.get("sha256") or
            Path(fixture_budget.grant_path).resolve() != Path(binding.get("path", "")).resolve() or
            fixture_budget.binaries.get("classic") != str(session.bound["executable"])):
        raise ValueError("exact parent FixtureBudget bound to this grant and Classic binary is mandatory")
    dispatch_id = session.context.request["dispatchId"]
    label = f"{dispatch_id}/{action}"
    remaining, stopped = session.ledger.running_bound(session.context.request["task"]["id"])
    elapsed = session.ledger.controller_elapsed(dispatch_id)
    authorization = _strict_json(session.context.authorization_raw, "Classic role authorization")
    wall_remaining = min(float(session.context.request["wallSeconds"]) - elapsed,
                         float(session.context.authority.grant["maxSessionWallSeconds"]) - elapsed,
                         float(authorization["expiresAt"]) - time.time(),
                         float(session.context.authority.grant["expiresAt"]) - time.time(),
                         float(remaining))
    if stopped or wall_remaining < 1.0:
        raise ValueError(stopped or "no shared controller wall-time remains")
    fixture_budget.reserve("classic", label, spec["argv"])
    from native_fixture_budget import DEADLINE as FIXTURE_PROCESS_DEADLINE
    timeout = min(CLASSIC_MAX_PROCESS_SECONDS, FIXTURE_PROCESS_DEADLINE,
                  int(spec["timeoutSeconds"]), int(wall_remaining))
    step_dir = session.reports / "process" / action
    started = time.time()
    env = strip_bootstrap_environment(os.environ)
    env.pop("OPENAI_API_KEY", None)
    env.pop("CODEX_API_KEY", None)
    env[BOOTSTRAP_PATH_ENV] = str(session.context.bootstrap_path)
    env[BOOTSTRAP_SHA_ENV] = session.context.bootstrap_sha256
    process_result = bounded(spec["argv"], spec["cwd"], step_dir, timeout, env=env)
    process_raw = (step_dir / "process.json").read_bytes()
    _strict_json(process_raw, "Classic product process receipt")
    capture = {"argv": spec["argv"], "cwd": spec["cwd"], "returnCode": process_result["returnCode"],
               "stdout": (step_dir / "stdout.log").read_bytes(), "stderr": (step_dir / "stderr.log").read_bytes(),
               "wallSeconds": process_result["wallSeconds"], "timedOut": process_result["stopReason"] == "wall_deadline",
               "processTreeControl": process_result["processTreeControl"], "stopReason": process_result["stopReason"],
               "productSourceSha": classic.EXPECTED_SOURCE,
               "runtimeSha256": classic.sha256_file(session.bound["executable"])}
    persisted = classic_integration.persist_native_capture(action, planned, capture)
    fixture_receipt = {"status": "completed" if process_result["returnCode"] == 0 and not process_result["stopReason"] else "incomplete",
                       "argv": spec["argv"], "returnCode": process_result["returnCode"],
                       "wallSeconds": process_result["wallSeconds"], "timedOut": capture["timedOut"],
                       "processTreeControl": process_result["processTreeControl"],
                       "stopReason": process_result["stopReason"],
                       "stdoutSha256": digest(capture["stdout"]), "stderrSha256": digest(capture["stderr"]),
                       "processSha256": persisted["processSha256"]}
    controller_receipt = {**fixture_receipt, "action": action, "processPath": persisted["processPath"]}
    ledger_row = session.ledger.record_controller_process(dispatch_id, action, started, controller_receipt)
    persisted["ledgerSequence"] = ledger_row["sequence"]
    session.captures.append(persisted)
    session.process_records.append(controller_receipt)
    if action == "apply":
        session.review = dict(external_review)
    fixture_budget.finish("classic", label, fixture_receipt)
    return persisted


def finalize_classic_controller(session):
    """Normalize all five captures and terminalize the shared controller booking."""
    import classic_integration
    import dispatch

    if not isinstance(session, ClassicControllerSession) or session.finished:
        raise ValueError("active Classic controller session required")
    result = classic_integration.normalize_result(session.context.request_raw, session.captures,
                                                   inference_performed=False)
    result.update(dispatchId=session.context.request["dispatchId"],
                  grantSha256=session.context.authority.grant_sha,
                  protocolSha256=session.context.authority.protocol_sha,
                  runtimeSourceSha256=dispatch.runtime_pins(),
                  controllerProcesses=session.process_records,
                  controllerElapsedSeconds=session.ledger.controller_elapsed(session.context.request["dispatchId"]))
    result["receipts"].append({"path": str(session.context.bootstrap_path),
                               "sha256": session.context.bootstrap_sha256,
                               "kind": "controller-bootstrap"})
    request_receipt = session.evidence / "request.json"
    result["receipts"].append({"path": str(request_receipt.resolve()),
                               "sha256": digest(request_receipt.read_bytes()),
                               "kind": "outer-request"})
    for index, item in enumerate(session.context.request["releasedInputs"]):
        snapshot = session.evidence / "released-inputs" / str(index)
        if digest(snapshot.read_bytes()) != item["sha256"]:
            raise ValueError("Classic released-input snapshot digest mismatch")
        result["receipts"].append({"path": str(snapshot.resolve()), "sha256": item["sha256"],
                                   "kind": "released-input-snapshot"})
    process_receipt = {"actions": session.process_records,
                       "bootstrapSha256": session.context.bootstrap_sha256,
                       "elapsedSeconds": result["controllerElapsedSeconds"]}
    final = session.ledger.finish_controller_dispatch(session.context.request["dispatchId"], result,
                                                      process_receipt)
    session.finished = True
    target = session.context.authority.result_directory / (session.context.request["dispatchId"] + ".json")
    dispatch.write_result(target, final)
    return final


def strip_bootstrap_environment(env):
    """Do not make the outer authority locator visible to delegate subprocesses."""
    cleaned = dict(env)
    for key in (BOOTSTRAP_PATH_ENV, BOOTSTRAP_SHA_ENV):
        cleaned.pop(key, None)
    return cleaned
