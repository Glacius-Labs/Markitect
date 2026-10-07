"""Importable bounded middleware helper for one native agentexec role call.

It accepts the documented Invocation bytes, a previously validated Coordinator
Authority and operator authorization, reserves before one delegate, and returns
response bytes unchanged. The native CLI bootstrap stays disabled until the
outer Authority can be supplied without a Request/runtime digest cycle.
"""
from __future__ import annotations

import hashlib
import json
import os
from pathlib import Path
import re
import stat
import time

from ledger import Ledger, LimitReached
from process import bounded
import government

INVOCATION_API = "markitect.example.org/agent-execution/v1alpha1"
ROLE_AUTH_API = "markitect.scientist-role-authorization/v1alpha1"
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
                      evidence_path: str, slot_id: str) -> list[str]:
    return [str(Path(script_path).resolve()), "--authorization", str(Path(authorization_path).resolve()),
            "--authorization-sha256", authorization_sha256,
            "--evidence", str(Path(evidence_path).resolve()), "--slot", slot_id]


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
               expected_slot: str | None) -> tuple[dict, dict, dict]:
    if not _HEX256.fullmatch(expected_sha) or digest(raw) != expected_sha:
        raise ValueError("operator-supplied role authorization digest mismatch")
    if len(raw) > _MAX_AUTH_BYTES:
        raise ValueError("role authorization exceeds its size bound")
    if Path(authorization_path).resolve(strict=True).read_bytes() != raw:
        raise ValueError("role authorization path bytes differ from the supplied approved bytes")
    auth = _strict_json(raw, "role authorization")
    if not isinstance(auth, dict) or auth.get("apiVersion") != ROLE_AUTH_API or auth.get("status") != "approved":
        raise ValueError("approved Scientist role authorization required")
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

    product = request.get("product", {}).get("government", {})
    runtime_path, runtime_sha = product.get("runtime", {}).get("path"), product.get("runtime", {}).get("sha256")
    if not isinstance(runtime_path, str) or not isinstance(runtime_sha, str):
        raise ValueError("Request-bound Government runtime required")
    runtime_raw = _released(captured, runtime_path, runtime_sha, "Government runtime")
    if auth.get("runtimePath") != str(Path(runtime_path).resolve()):
        raise ValueError("role authorization runtime binding mismatch")
    runtime = _strict_json(runtime_raw, "Government runtime")
    roles = government.configured_roles(runtime)
    by_slot = {role["slotId"]: role for role in roles}
    if len(by_slot) != len(roles):
        raise ValueError("configured Government role slots are ambiguous")
    auth_slots = auth.get("slots")
    if not isinstance(auth_slots, list) or len(auth_slots) != len(roles):
        raise ValueError("operator authorization must cover every configured native role slot")
    auth_by_slot = {slot.get("slotId"): slot for slot in auth_slots if isinstance(slot, dict)}
    if len(auth_by_slot) != len(auth_slots) or set(auth_by_slot) != set(by_slot):
        raise ValueError("authorized role slots do not exactly match the native runtime")
    phase, slot_id = role_projection(invocation["request"]["projectionId"])
    if expected_slot is not None and slot_id != expected_slot:
        raise ValueError("native invocation arrived at a different configured wrapper slot")
    role_slot = auth_by_slot.get(slot_id)
    configured = by_slot.get(slot_id)
    if not role_slot or not configured or (phase, invocation["request"]["role"]) != (configured["phase"], configured["responseRole"]):
        raise ValueError("agentexec invocation role/phase is not authorized for this slot")
    if (role_slot.get("phase"), role_slot.get("responseRole")) != (phase, invocation["request"]["role"]):
        raise ValueError("operator role grant phase/role mismatch")
    wrapper = role_slot.get("wrapper", {})
    expected_args = wrapper_arguments(__file__, auth_path, expected_sha,
                                      auth["roleEvidenceDirectory"], slot_id)
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
    if (wrapper_files.get(str(Path(__file__).resolve())) != "sha256:" + script_sha or
            wrapper_files.get(auth_path) != "sha256:" + expected_sha):
        raise ValueError("wrapper script and role authorization must be runtime-file pinned")
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
    if request.get("arm") != "government" or request.get("operation") != "run_task":
        raise ValueError("role bridge only supports an operator-authorized Government run_task")
    return auth, role_slot, runtime


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


def _reserve(ledger: Ledger, auth: dict, invocation: dict, invocation_sha: str,
             authorization_sha: str, role_slot: dict, evidence_path: Path,
             grant_token_threshold: int) -> tuple[str, str]:
    call_id = digest(encoded({"dispatchId": auth["dispatchId"], "runId": invocation["runId"],
                              "nonce": invocation["nonce"], "inputDigest": invocation["inputDigest"]}))
    with ledger.transaction() as db:
        db.execute("""CREATE TABLE IF NOT EXISTS government_role_calls(
            call_id TEXT PRIMARY KEY, dispatch_id TEXT NOT NULL, attempt_id TEXT UNIQUE NOT NULL,
            authorization_sha256 TEXT NOT NULL, invocation_sha256 TEXT NOT NULL,
            input_digest TEXT NOT NULL, run_id TEXT NOT NULL, nonce_sha256 TEXT NOT NULL,
            slot_id TEXT NOT NULL, phase TEXT NOT NULL, evidence_path TEXT NOT NULL,
            status TEXT NOT NULL, receipt TEXT)""")
        prior = db.execute("SELECT 1 FROM government_role_calls WHERE call_id=?", (call_id,)).fetchone()
        if prior:
            raise ValueError("agentexec invocation replay is already reserved; no delegate relaunch")
        count = db.execute("SELECT COUNT(*) FROM government_role_calls WHERE dispatch_id=?", (auth["dispatchId"],)).fetchone()[0]
        if count >= auth["maxCalls"]:
            raise LimitReached("operator role-call authorization exhausted")
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
        reported_tokens = db.execute("""SELECT COALESCE(SUM(tokens),0) FROM attempts
            WHERE id=? OR id IN (SELECT attempt_id FROM government_role_calls WHERE dispatch_id=?)""",
                                     (dispatch_attempt[0], auth["dispatchId"])).fetchone()[0]
        if reported_tokens >= grant_token_threshold:
            raise LimitReached("Coordinator retrospective token threshold")
        task_id = auth["taskId"]
        purpose = "task" if role_slot["phase"] == "execute" else "review"
        attempt_id = ledger._reserve(db, task_id, purpose)
        db.execute("INSERT INTO government_role_calls VALUES(?,?,?,?,?,?,?,?,?,?,?,'reserved',NULL)",
                   (call_id, auth["dispatchId"], attempt_id, authorization_sha, invocation_sha,
                    invocation["inputDigest"], invocation["runId"], digest(invocation["nonce"].encode()),
                    role_slot["slotId"], role_slot["phase"], str(evidence_path)))
    return call_id, attempt_id


def _finish(ledger: Ledger, call_id: str, attempt_id: str, status: str,
            turns: int | None, tokens: int | None, receipt: dict) -> None:
    with ledger.transaction() as db:
        changed = db.execute("UPDATE attempts SET end=?,status=?,turns=?,tokens=?,receipt=? WHERE id=? AND end IS NULL",
                             (time.time(), status, turns, tokens, json.dumps([receipt], sort_keys=True), attempt_id)).rowcount
        if changed != 1:
            raise ValueError("role reservation is unknown or already finalized")
        changed = db.execute("UPDATE government_role_calls SET status=?,receipt=? WHERE call_id=? AND status='reserved'",
                             (status, json.dumps(receipt, sort_keys=True), call_id)).rowcount
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
        outer = db.execute("SELECT start FROM attempts WHERE id=?", (attempt_id,)).fetchone()
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
             cwd: str | None = None) -> tuple[bytes, dict]:
    """Validate one per-call operator grant, reserve, then execute one delegate.

    `authority` is the existing validated dispatch.Authority. It revalidates the
    exact outer Request/Grant/Protocol and returns the already captured released
    inputs; its Ledger is the existing shared trial ledger. This helper never
    retries. An interrupted/ambiguous reservation stays occupied.
    """
    invocation = parse_invocation(invocation_raw)
    request, captured = authority.validate(request_raw)
    auth, role_slot, runtime = _role_auth(authorization_raw, authorization_sha256, request, request_raw,
                                         authority, captured, invocation, authorization_path, expected_slot)
    if not (time.time() < auth["expiresAt"]):
        raise ValueError("operator role authorization expired before delegate reservation")
    ledger = authority.ledger("run_task", dispatch_id=request["dispatchId"])
    record = ledger.dispatch_record(request["dispatchId"])
    if (not record or record.get("phase") != "launching" or record.get("result") is not None or
            record.get("request") != request_raw or record.get("execution_sha") != _execution_sha(request)):
        raise ValueError("active operator-authorized outer dispatch binding required")
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
    protected_paths = [request["actorRepository"], request["evidenceDirectory"], authority.ledger_path,
                       authorization_path, auth["requestPath"], *authority_paths,
                       request["product"]["government"]["queueStateDirectory"], runtime.get("stateDirectory")]
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
    reserved_id, attempt_id = _reserve(ledger, auth, invocation, invocation_sha,
                                       authorization_sha256, role_slot, evidence_path,
                                       authority.grant["retrospectiveTokenThreshold"])
    if reserved_id != call_id:
        raise AssertionError("internal role invocation identity mismatch")
    effective_timeout = _effective_timeout(ledger, request, auth, authority, record,
                                          delegate["timeoutSeconds"])
    evidence_parent.mkdir(parents=True, exist_ok=True)
    environment = dict(os.environ)
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
    receipt = {"apiVersion": "markitect.scientist-role-receipt/v1alpha1", "callId": call_id,
               "dispatchId": request["dispatchId"], "trialId": request["trialId"],
               "taskId": request["task"]["id"], "authorizationSha256": authorization_sha256,
               "outerRequestSha256": digest(request_raw), "invocationSha256": invocation_sha,
               "inputDigest": invocation["inputDigest"], "runId": invocation["runId"],
               "nonceSha256": digest(invocation["nonce"].encode()), "slotId": role_slot["slotId"],
               "phase": role_slot["phase"], "responseRole": invocation["request"]["role"],
               "delegateArgvSha256": digest(encoded(argv)), "returnCode": result["returnCode"],
               "effectiveTimeoutSeconds": effective_timeout,
               "stdoutPath": str(stdout_path.resolve()), "stderrPath": str(stderr_path.resolve()),
               "processReceiptPath": str(process_path.resolve()),
               "responseSha256": digest(stdout), "stderrSha256": digest(stderr_path.read_bytes()),
               "processReceiptSha256": digest(process_raw), "status": status,
               "providerRequests": None, "providerTurns": None,
               "reportedInputPlusOutputTokens": tokens,
               "usageSource": "agentexec-provider-reported" if tokens is not None else "unknown"}
    # Missing request/turn telemetry is preserved as unknown; the Ledger's bound
    # measurement profile decides whether another invocation can be admitted.
    _finish(ledger, call_id, attempt_id, status, None, tokens, receipt)
    if result["returnCode"] != 0:
        raise RuntimeError("authorized Government delegate process failed; reservation remains consumed")
    if not protocol_ok:
        raise ValueError("delegate response did not echo the exact agentexec invocation identity")
    return stdout, receipt


def main(argv: list[str] | None = None) -> int:
    """Native execution is disabled until an acyclic validated-Authority bootstrap exists."""
    raise RuntimeError("role bridge is a library helper; native CLI bootstrap is not integrated")


if __name__ == "__main__":
    raise SystemExit(main())
