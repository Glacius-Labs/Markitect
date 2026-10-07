"""One fixed, sanitized config-read session with strict server-frame classification.

No raw server payload or stderr text is persisted. Importing this module never
loads the frozen request or starts a process; the classifier is pure and testable.
"""
import hashlib
import json
import os
from pathlib import Path
import queue
import re
import subprocess
import threading
import time

ROOT = Path(__file__).resolve().parent
ENUMS_PATH = ROOT / "frozen-method-enums.json"
FLAGS = {"apps", "goals", "hooks", "memories", "multi_agent", "plugins", "shell_tool", "unified_exec", "network_proxy", "windows_sandbox_service", "powershell_shell_version"}
POLICY_KEYS = {"approval_policy", "approvals_reviewer", "sandbox_mode", "sandbox_workspace_write", "windows", "default_permissions", "permissions", "features"}
ENUM_VALUES = {"never", "on-request", "untrusted", "user", "auto_review", "read-only", "workspace-write", "danger-full-access", "elevated", "unelevated", "mxc", "read", "write", "deny", "allow", "restricted", "enabled"}
PROVISIONAL_METHODS = {"warning", "configWarning", "windows/worldWritableWarning"}
DISCARD_METHODS = {"warning", "configWarning", "deprecationNotice", "windows/worldWritableWarning"}
REMOTE_STATUS_METHOD = "remoteControl/status/changed"
NOTIFICATION_LIMIT = 16
ALLOCATION_ID = "local-policy-read-disabled-status-20261008"
EXPECTED_LIMITS = {
    "appServerProcessTrees": 1, "wallSeconds": 60, "innerWallSeconds": 50,
    "parallelism": 1, "retries": 0, "actorStarts": 0, "queuedFrames": 32,
    "rawByteIngestionThreshold": 4_000_000, "allowedWarningNotifications": 16,
}
EXPECTED_PRIOR_CLIENT_SHA256 = "998480229ae2573264904b00c8077ba2a6bed11349b04749cefdc6cd79724390"
EXPECTED_SOURCE_THREAD_ID = "01a11367-a781-7683-a20f-46e12614dcb4"
EXPECTED_SOURCE_COORDINATION_SHA256 = "04484d4220dd7383af1d2675a95384554380dccc5a14ee8fc5908c3a474ede58"
EXPECTED_ALLOWED_INBOUND = ["warning", "configWarning", "deprecationNotice", "windows/worldWritableWarning", "remoteControl/status/changed"]
EXPECTED_WARNING_POLICY = "warning/configWarning/windows/worldWritableWarning render any config results provisional; never interpret warning text"
EXPECTED_ALL_SERVER_REQUESTS_POLICY = "stop without answering, including currentTime/read; no method or payload response"
EXPECTED_REMOTE_CONTROL_RULE = "Exact frozen notification schema required. Retain only method/class/status enum; discard installationId, environmentId, serverName and all other parameters/identities. Only disabled may pass without reply. connecting/connected/errored/malformed abort; never enable/connect remote control."
EXPECTED_GRANT_BOUNDARY = "Same pinned 0.160.1 binary, cwd and fifteen inline pairs. Preserve old ledgers, grants, raw historical unknowns and sanitizer/process limits. Immutable preflight and postreview in separate files; no fabricated archive recovery. This remains metadata, not historical exec-policy/access proof."
EXPECTED_PRIOR_CONSUMPTION = {
    "localCliMetadataCalls": 6, "appServerPolicyReadTrees": 2, "actorStarts": 5,
    "knownActorInputPlusOutputTokens": 53331, "historicalActorTokenTotal": None,
}
EXPECTED_GRANT_CORE = {
    "key": ALLOCATION_ID,
    "maxAdditionalAppServerSessions": 1,
    "previousAppServerSessionsConsumed": 2,
    "cumulativeAppServerSessionsMaximum": 3,
    "wallSeconds": 60,
    "innerWallSeconds": 50,
    "maxParallel": 1,
    "allowedMethods": ["initialize", "initialized", "config/read", "configRequirements/read"],
    "maxConfigReads": 1,
    "maxRequirementsReads": 1,
    "retries": 0,
    "realActorCallsAuthorized": 0,
    "modelCallsAuthorized": 0,
    "mutations": False,
    "newPurchases": False,
    "allowedInboundNotifications": EXPECTED_ALLOWED_INBOUND,
    "maxInboundNotifications": 16,
    "remoteControlRule": EXPECTED_REMOTE_CONTROL_RULE,
    "boundary": EXPECTED_GRANT_BOUNDARY,
    "warningResultRule": "Existing provisional-result rule retained for non-deprecation warnings.",
    "abortOn": "Every server request and all other unexpected messages/activity, no replies or retries. No further automatic diagnostic cascade.",
    "realActorCallsAuthorized": 0,
    "modelCallsAuthorized": 0,
    "mutations": False,
    "newPurchases": False,
    "remainingSessions": 1,
    "consumedSessions": None,
}


def require(condition, reason):
    if not condition:
        raise RuntimeError(reason)


def validate_authorization_grant(request):
    """Bind the staged request to the exact local copy and original grant."""
    binding = request["authorizationGrant"]
    grant_path = Path(binding["path"])
    grant_bytes = grant_path.read_bytes()
    grant_sha = hashlib.sha256(grant_bytes).hexdigest()
    require(grant_sha == binding["sha256"], "authorization grant hash mismatch")
    document = json.loads(grant_bytes)
    require(document["sourceThreadId"] == EXPECTED_SOURCE_THREAD_ID, "grant source thread mismatch")
    require(document["sourceJsonPointer"] == "threads[name=Scientist].evidence.localDisabledStatusMetadataGrant", "grant source pointer mismatch")
    original_path = Path(document["sourceCoordinationPath"])
    original_bytes = original_path.read_bytes()
    require(document["sourceCoordinationSha256"] == EXPECTED_SOURCE_COORDINATION_SHA256, "grant snapshot pin mismatch")
    require(hashlib.sha256(original_bytes).hexdigest() == document["sourceCoordinationSha256"], "grant source document hash mismatch")
    coordination = json.loads(original_bytes)
    scientist = next((item for item in coordination["threads"] if item.get("name") == "Scientist"), None)
    require(scientist is not None, "grant source thread missing")
    original_grant = scientist["evidence"]["localDisabledStatusMetadataGrant"]
    require(document["grant"] == original_grant, "grant copy differs from coordination source")
    grant = document["grant"]
    require(all(grant.get(key) == value for key, value in EXPECTED_GRANT_CORE.items()), "grant core differs from authorized allocation")
    require(grant.get("status") == "Assigned; independent preflight, freeze and separate one-time reservation required; usage not yet confirmed", "grant status differs from assignment")
    require("provisional" in grant.get("warningResultRule", "").lower(), "grant warning rule missing provisional restriction")
    require("no repl" in grant.get("abortOn", "").lower(), "grant abort rule missing no-reply condition")
    return grant_sha, grant


def validate_prior_consumption(authorization, reservation):
    require(authorization.get("priorConsumption") == EXPECTED_PRIOR_CONSUMPTION, "historical consumption mismatch")
    require(reservation.get("priorConsumption") == EXPECTED_PRIOR_CONSUMPTION, "reservation historical consumption mismatch")
    require(reservation.get("priorConsumption") == authorization.get("priorConsumption"), "reservation history differs from authorization")


def validate_authorization_reference(authorization, grant):
    require(authorization.get("allowedInboundNotifications") == EXPECTED_ALLOWED_INBOUND, "authorization notification allowlist mismatch")
    require(authorization.get("allowedInboundNotifications") == grant.get("allowedInboundNotifications"), "authorization notification allowlist differs from grant")
    require(authorization.get("warningPolicy") == EXPECTED_WARNING_POLICY, "authorization warning policy mismatch")
    require("provisional" in grant.get("warningResultRule", "").lower(), "grant result provisional rule missing")
    require(authorization.get("allServerRequests") == EXPECTED_ALL_SERVER_REQUESTS_POLICY, "authorization server-request policy mismatch")
    require("no repl" in grant.get("abortOn", "").lower(), "grant server-request no-reply rule missing")
    require(authorization.get("remoteControlRule") == EXPECTED_REMOTE_CONTROL_RULE, "authorization remote-control rule mismatch")
    require(authorization.get("remoteControlRule") == grant.get("remoteControlRule"), "authorization remote-control rule differs from grant")


def _is_type(value, expected):
    if expected == "null":
        return value is None
    if expected == "boolean":
        return type(value) is bool
    if expected == "integer":
        return type(value) is int
    if expected == "number":
        return type(value) in (int, float) and type(value) is not bool
    if expected == "string":
        return type(value) is str
    if expected == "object":
        return isinstance(value, dict)
    if expected == "array":
        return isinstance(value, list)
    return False


def _schema_valid(value, schema):
    """Validate the schema keywords present in the frozen allowed shapes."""
    if "anyOf" in schema:
        return any(_schema_valid(value, option) for option in schema["anyOf"])
    types = schema.get("type")
    if types is not None:
        if not any(_is_type(value, t) for t in (types if isinstance(types, list) else [types])):
            return False
    if "enum" in schema and value not in schema["enum"]:
        return False
    if "minimum" in schema and (type(value) not in (int, float) or value < schema["minimum"]):
        return False
    if schema.get("format") == "uint" and (type(value) is not int or value < 0):
        return False
    if schema.get("format") == "int64" and (type(value) is not int or not -9223372036854775808 <= value <= 9223372036854775807):
        return False
    if "required" in schema and (not isinstance(value, dict) or any(k not in value for k in schema["required"])):
        return False
    if "properties" in schema:
        if not isinstance(value, dict):
            return False
        if any(k in value and not _schema_valid(value[k], child) for k, child in schema["properties"].items()):
            return False
    if "items" in schema:
        if not isinstance(value, list) or any(not _schema_valid(item, schema["items"]) for item in value):
            return False
    return True


def load_method_enums(path=ENUMS_PATH):
    document = json.loads(Path(path).read_bytes())
    methods = document["methods"]
    # Reject ambiguous frozen source data before any future process can start.
    overlap = set(methods["ServerNotification"]) & set(methods["ServerRequest"])
    if overlap:
        raise ValueError("overlapping frozen method enums")
    return document


def validate_disabled_status_gate(enums):
    require(enums.get("remoteControlStatusAcceptedValue") == "disabled", "accepted remote-control status differs from grant")
    schema = enums["methods"]["ServerNotification"][REMOTE_STATUS_METHOD]["paramsSchema"]
    status_schema = schema["properties"]["status"]
    require(status_schema.get("enum") == ["disabled", "connecting", "connected", "errored"], "remote-control status enum differs from frozen schema")


def classify_server_frame(frame, enums, notification_count):
    """Return only a safe event record, continuation decision, and provisional bit.

    The returned record contains method/class only for exact known methods. The
    caller must stop on every action other than ``discard-notification``.
    """
    if not isinstance(frame, dict) or type(frame.get("method")) is not str:
        return {"action": "stop", "reason": "malformed-server-frame", "event": None, "notificationCount": notification_count, "provisional": False}
    method = frame["method"]
    requests = enums["methods"]["ServerRequest"]
    notifications = enums["methods"]["ServerNotification"]
    has_id = "id" in frame
    if has_id:
        identifier = frame["id"]
        if type(identifier) is not str and (type(identifier) is not int or not -9223372036854775808 <= identifier <= 9223372036854775807):
            return {"action": "stop", "reason": "malformed-request-id", "event": None, "notificationCount": notification_count, "provisional": False}
        if method not in requests:
            meta = notifications.get(method)
            event = {"method": method, "class": meta["class"]} if meta else None
            return {"action": "stop", "reason": "unexpected-or-malformed-request", "event": event, "notificationCount": notification_count, "provisional": False}
        return {"action": "stop", "reason": "unanswered-server-request", "event": {"method": method, "class": requests[method]["class"]}, "notificationCount": notification_count, "provisional": False}

    if method not in notifications:
        meta = requests.get(method)
        event = {"method": method, "class": meta["class"]} if meta else None
        return {"action": "stop", "reason": "unexpected-or-malformed-notification", "event": event, "notificationCount": notification_count, "provisional": False}

    for key, schema in enums.get("notificationEnvelopeOptionalProperties", {}).items():
        if key in frame and not _schema_valid(frame[key], schema):
            return {"action": "stop", "reason": "malformed-notification-envelope", "event": {"method": method, "class": notifications[method]["class"]}, "notificationCount": notification_count, "provisional": method in PROVISIONAL_METHODS}

    next_count = notification_count + 1
    event = {"method": method, "class": notifications[method]["class"]}
    if next_count > NOTIFICATION_LIMIT:
        return {"action": "stop", "reason": "notification-limit", "event": event, "notificationCount": next_count, "provisional": method in PROVISIONAL_METHODS}
    if method == REMOTE_STATUS_METHOD:
        params = frame.get("params")
        if "params" not in frame or not _schema_valid(params, notifications[method]["paramsSchema"]):
            return {"action": "stop", "reason": "malformed-remote-control-status", "event": event, "notificationCount": next_count, "provisional": False}
        status = params["status"]
        event["status"] = status
        if status != enums["remoteControlStatusAcceptedValue"]:
            return {"action": "stop", "reason": "remote-control-status-not-disabled", "event": event, "notificationCount": next_count, "provisional": False}
        return {"action": "discard-notification", "reason": None, "event": event, "notificationCount": next_count, "provisional": False}
    if method not in DISCARD_METHODS:
        return {"action": "stop", "reason": "non-discardable-notification", "event": event, "notificationCount": next_count, "provisional": False}
    params = frame.get("params")
    if "params" not in frame or not _schema_valid(params, notifications[method]["paramsSchema"]):
        return {"action": "stop", "reason": "malformed-discardable-notification", "event": event, "notificationCount": next_count, "provisional": method in PROVISIONAL_METHODS}
    return {"action": "discard-notification", "reason": None, "event": event, "notificationCount": next_count, "provisional": method in PROVISIONAL_METHODS}


def validate_response_envelope(frame, expected_id):
    """Return result for one plain result envelope; reject mixed/invalid frames."""
    if not isinstance(frame, dict) or "method" in frame:
        raise ValueError("malformed-response-envelope")
    identifier = frame.get("id")
    if type(identifier) is not type(expected_id) or identifier != expected_id:
        raise ValueError("unexpected-response-id-or-shape")
    if ("error" in frame) == ("result" in frame):
        raise ValueError("malformed-response-envelope")
    if "error" in frame:
        raise ValueError("rpc-error")
    if not isinstance(frame["result"], dict):
        raise ValueError("unexpected-result-shape")
    return frame["result"]


def scalar(value):
    return value is None or type(value) in (bool, int) or (type(value) is str and value in ENUM_VALUES)


def identity(value):
    return type(value) is str and re.fullmatch(r"[A-Za-z0-9:_-]{1,100}", value) is not None


def path_value(value):
    return type(value) is str and len(value) < 2048 and "\n" not in value and (value.startswith(("/", "~", ":")) or re.match(r"^[A-Za-z]:[\\/]", value)) is not None


def access_map(value):
    if not isinstance(value, dict):
        return None
    return {k: (v if v in ("read", "write", "deny", "allow") else access_map(v))
            for k, v in value.items() if isinstance(k, str) and isinstance(v, (str, dict))
            and (isinstance(v, dict) or v in ("read", "write", "deny", "allow"))}


def network(value):
    if not isinstance(value, dict):
        return None
    result = {k: v for k, v in value.items() if k in {"enabled", "allowLocalBinding", "allowUpstreamProxy", "dangerouslyAllowAllUnixSockets", "dangerouslyAllowNonLoopbackProxy", "managedAllowedDomainsOnly"} and type(v) is bool}
    for key in ("domains", "unixSockets", "unix_sockets"):
        if key in value:
            result[key] = access_map(value[key])
    for key in ("allowedDomains", "deniedDomains", "allowUnixSockets"):
        if isinstance(value.get(key), list):
            result[key] = [v for v in value[key] if isinstance(v, str) and len(v) < 2048 and not any(c in v for c in ("\n", "@", "?", "#"))]
    return result


def config(value):
    if not isinstance(value, dict):
        return {}
    result = {k: v for k, v in value.items() if k in {"approval_policy", "approvals_reviewer", "sandbox_mode"} and scalar(v)}
    if identity(value.get("default_permissions")):
        result["default_permissions"] = value["default_permissions"]
    for key in ("windows", "sandbox_workspace_write"):
        if isinstance(value.get(key), dict):
            result[key] = {k: v for k, v in value[key].items() if k in {"sandbox", "sandbox_private_desktop", "network_access", "exclude_tmpdir_env_var", "exclude_slash_tmp"} and scalar(v)}
            if key == "sandbox_workspace_write" and isinstance(value[key].get("writable_roots"), list):
                result[key]["writable_roots"] = [v for v in value[key]["writable_roots"] if path_value(v)]
    if isinstance(value.get("features"), dict):
        result["features"] = {k: v for k, v in value["features"].items() if k in FLAGS and type(v) is bool}
    if isinstance(value.get("permissions"), dict):
        profiles = {}
        for name, profile in value["permissions"].items():
            if not identity(name) or not isinstance(profile, dict):
                continue
            selected = {}
            if identity(profile.get("extends")):
                selected["extends"] = profile["extends"]
            if "filesystem" in profile:
                selected["filesystem"] = access_map(profile["filesystem"])
            if "network" in profile:
                selected["network"] = network(profile["network"])
            if isinstance(profile.get("workspace_roots"), dict):
                selected["workspace_roots"] = {k: v for k, v in profile["workspace_roots"].items() if path_value(k) and type(v) is bool}
            profiles[name] = selected
        result["permissions"] = profiles
    return result


def metadata(value):
    if not isinstance(value, dict):
        return {}
    result = {k: v for k, v in value.items() if k in {"type", "version", "profile"} and identity(v)}
    result.update({k: v for k, v in value.items() if k in {"file", "path"} and path_value(v)})
    if isinstance(value.get("name"), dict):
        result["name"] = metadata(value["name"])
    return result


def requirements(value):
    if value is None:
        return None
    if not isinstance(value, dict):
        return {}
    result = {}
    for key in ("allowedApprovalPolicies", "allowedApprovalsReviewers", "allowedSandboxModes", "allowedWindowsSandboxImplementations"):
        if isinstance(value.get(key), list):
            result[key] = [v for v in value[key] if scalar(v)]
    if identity(value.get("defaultPermissions")):
        result["defaultPermissions"] = value["defaultPermissions"]
    for key in ("allowedPermissionProfiles", "featureRequirements"):
        if isinstance(value.get(key), dict):
            result[key] = {k: v for k, v in value[key].items() if type(v) is bool and (identity(k) if key == "allowedPermissionProfiles" else k in FLAGS)}
    if "network" in value:
        result["network"] = network(value["network"])
    if type(value.get("allowLoginShell")) is bool:
        result["allowLoginShell"] = value["allowLoginShell"]
    return result


def sanitize(identifier, value):
    if identifier == 0:
        return {"handshakeAcknowledged": True}
    if identifier == 1:
        result = {"config": config(value.get("config")), "origins": {k: metadata(v) for k, v in value.get("origins", {}).items() if k in POLICY_KEYS or (k.startswith("features.") and k[9:] in FLAGS)}}
        if isinstance(value.get("layers"), list):
            result["layers"] = [{**metadata(v), "config": config(v.get("config"))} for v in value["layers"] if isinstance(v, dict)]
        return result
    return {"requirements": requirements(value.get("requirements"))}


def _load_request():
    if not __debug__:
        raise RuntimeError("optimized Python is prohibited for frozen guard checks")
    request_bytes = (ROOT / "request.json").read_bytes()
    request = json.loads(request_bytes)
    client_sha = hashlib.sha256(Path(__file__).read_bytes()).hexdigest()
    require(client_sha == request["clientSha256"], "client hash mismatch")
    enums = load_method_enums()
    enums_sha = hashlib.sha256(ENUMS_PATH.read_bytes()).hexdigest()
    require(enums_sha == request["methodEnumsSha256"], "method-enum hash mismatch")
    validate_disabled_status_gate(enums)
    request_sha = hashlib.sha256(request_bytes).hexdigest()
    require(request["allocationId"] == ALLOCATION_ID, "allocation id mismatch")
    require(request["limits"] == EXPECTED_LIMITS, "allocation limits mismatch")
    authorization = request["authorizationReference"]
    require(authorization["allocationId"] == ALLOCATION_ID, "authorization allocation mismatch")
    require(authorization["sourceThreadId"] == EXPECTED_SOURCE_THREAD_ID, "authorization source mismatch")
    grant_sha, grant = validate_authorization_grant(request)
    validate_authorization_reference(authorization, grant)

    freeze_bytes = (ROOT / "freeze.json").read_bytes()
    freeze_sha = hashlib.sha256(freeze_bytes).hexdigest()
    freeze = json.loads(freeze_bytes)
    reservation_bytes = (ROOT / "reservation.json").read_bytes()
    reservation = json.loads(reservation_bytes)
    bound = {"allocationId": ALLOCATION_ID, "requestSha256": request_sha,
             "clientSha256": client_sha, "methodEnumsSha256": enums_sha, "maxTrees": 1}
    for key, value in bound.items():
        require(freeze[key] == value, "freeze binding mismatch: " + key)
        require(reservation[key] == value, "reservation binding mismatch: " + key)
    require(reservation["freezeSha256"] == freeze_sha, "reservation freeze mismatch")
    require(reservation["sourceThreadId"] == authorization["sourceThreadId"], "reservation source mismatch")
    validate_prior_consumption(authorization, reservation)
    require(isinstance(freeze.get("files"), dict), "frozen files map missing")
    for source_key, expected_sha in request["sourceHashes"].items():
        source_path = Path(request["sourcePaths"][source_key])
        require(hashlib.sha256(source_path.read_bytes()).hexdigest() == expected_sha, "source hash mismatch: " + source_key)
    require(request["sourceHashes"].get("priorPolicyClient.py") == EXPECTED_PRIOR_CLIENT_SHA256, "prior policy client pin mismatch")
    repo = ROOT.parents[4]
    for relative, expected_sha in freeze["files"].items():
        source_path = Path(relative)
        if not source_path.is_absolute():
            source_path = repo / source_path
        require(hashlib.sha256(source_path.read_bytes()).hexdigest() == expected_sha, "frozen file hash mismatch")

    exe = Path(request["argv"][0])
    require(hashlib.sha256(exe.read_bytes()).hexdigest() == request["executableSha256"], "executable hash mismatch")
    require(request["argv"][1:3] == ["app-server", "--stdio"], "unexpected executable route")
    frozen_bytes = Path(request["frozenExecProcess"]).read_bytes()
    require(hashlib.sha256(frozen_bytes).hexdigest() == request["frozenExecProcessSha256"], "frozen process hash mismatch")
    frozen_process = json.loads(frozen_bytes)
    frozen_argv = frozen_process["argv"]
    overrides = [part for i, value in enumerate(frozen_argv) if value == "--config" for part in ("--config", frozen_argv[i + 1])]
    require(len(overrides) == 30, "expected exactly 15 inline config settings")
    require(request["argv"] == [frozen_argv[0], "app-server", "--stdio", *overrides], "argv differs from frozen inline settings")
    require(request["cwd"] == frozen_process["cwd"], "cwd differs from frozen process")
    require(request["rpc"] == [
        {"method": "initialize", "id": 0, "params": {"clientInfo": {"name": "scientist_policy_read_once", "version": "1.0"}}},
        {"method": "initialized", "params": {}},
        {"method": "config/read", "id": 1, "params": {"includeLayers": True, "cwd": request["cwd"]}},
        {"method": "configRequirements/read", "id": 2},
    ], "RPC sequence differs from the authorized reads")
    return request, enums, {
        "allocationId": ALLOCATION_ID,
        "requestSha256": request_sha,
        "clientSha256": client_sha,
        "methodEnumsSha256": enums_sha,
        "freezeSha256": freeze_sha,
        "reservationSha256": hashlib.sha256(reservation_bytes).hexdigest(),
        "authorizationGrantSha256": grant_sha,
    }


def main():
    request, enums, frozen_binding = _load_request()
    marker = ROOT / "started-once.json"
    with marker.open("x", encoding="utf-8") as file:
        json.dump({"startedAtUnix": time.time(), **frozen_binding}, file)
        file.flush()
        os.fsync(file.fileno())
    started = time.monotonic()
    # The 38-second whole-tree watchdog leaves four seconds for client shutdown/drain.
    deadline = started + 33
    messages = queue.Queue(maxsize=32)
    raw_limit = threading.Event()
    raw_lock = threading.Lock()
    raw_bytes = 0
    stderr_bytes = 0
    response_bytes = 0
    observations = []
    method_events = []
    sent = []
    status = "stopped"
    reason = None
    native = None
    notification_count = 0
    provisional = False
    env = {k: v for k, v in os.environ.items() if k not in ("OPENAI_API_KEY", "CODEX_API_KEY")}
    server = subprocess.Popen(request["argv"], cwd=request["cwd"], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, shell=False, env=env)

    def pump(stream, kind):
        nonlocal raw_bytes
        while True:
            raw = stream.readline(2_000_001)
            with raw_lock:
                raw_bytes += len(raw)
                if raw_bytes > 4_000_000:
                    raw_limit.set()
                    return
            try:
                messages.put_nowait((kind, raw))
            except queue.Full:
                raw_limit.set()
                return
            if not raw:
                return

    threads = [threading.Thread(target=pump, args=(stream, kind), daemon=True) for stream, kind in ((server.stdout, "stdout"), (server.stderr, "stderr"))]
    for thread in threads:
        thread.start()

    class StopSession(Exception):
        pass

    def process_server_frame(frame):
        nonlocal notification_count, provisional
        classified = classify_server_frame(frame, enums, notification_count)
        notification_count = classified["notificationCount"]
        provisional = provisional or classified["provisional"]
        if classified["event"] is not None:
            method_events.append(classified["event"])
        if classified["action"] != "discard-notification":
            raise StopSession(classified["reason"])

    def consume(kind, raw, identifier):
        nonlocal stderr_bytes, response_bytes, notification_count, provisional
        if kind == "stderr":
            stderr_bytes += len(raw)
            if re.search(rb"refresh|login|log.in|authenticat|unauthorized|credential", raw, re.I):
                raise StopSession("login-refresh-auth-signal; raw discarded")
            return None
        response_bytes += len(raw)
        if not raw:
            raise StopSession("server-stdout-closed")
        if len(raw) > 2_000_000:
            raise StopSession("response-size-limit")
        try:
            value = json.loads(raw)
        except Exception:
            raise StopSession("invalid-json; raw discarded") from None
        if isinstance(value, dict) and "method" in value:
            if "result" in value or "error" in value:
                classified = classify_server_frame(value, enums, notification_count)
                notification_count = classified["notificationCount"]
                provisional = provisional or classified["provisional"]
                if classified["event"] is not None:
                    method_events.append(classified["event"])
                raise StopSession("malformed-mixed-response-method")
            process_server_frame(value)
            return None
        if identifier is None:
            raise StopSession("unexpected-response-id-or-shape")
        try:
            return sanitize(identifier, validate_response_envelope(value, identifier))
        except ValueError as exc:
            reason = str(exc)
            raise StopSession(reason if reason != "rpc-error" else "rpc-error; raw discarded") from None

    def drain_queue(identifier):
        while not messages.empty():
            kind, raw = messages.get_nowait()
            result = consume(kind, raw, identifier)
            if result is not None:
                raise StopSession("unsolicited-response")

    try:
        for rpc in request["rpc"]:
            if raw_limit.is_set():
                raise StopSession("raw-output-limit")
            drain_queue(None)
            if time.monotonic() >= deadline:
                raise StopSession("client-deadline")
            server.stdin.write((json.dumps(rpc) + "\n").encode())
            server.stdin.flush()
            sent.append(rpc)
            if "id" not in rpc:
                continue
            while True:
                if raw_limit.is_set():
                    raise StopSession("raw-output-limit")
                if time.monotonic() >= deadline:
                    raise StopSession("client-deadline")
                try:
                    kind, raw = messages.get(timeout=0.1)
                except queue.Empty:
                    continue
                result = consume(kind, raw, rpc["id"])
                if result is not None:
                    observations.append({"id": rpc["id"], "method": rpc["method"], "sanitizedResult": result})
                    break
        status = "read-responses-received"
    except StopSession as exc:
        reason = str(exc)
    except Exception:
        reason = "local-client-exception; details discarded"
    finally:
        try:
            server.stdin.close()
        except Exception:
            pass
        try:
            native = server.wait(timeout=2)
        except subprocess.TimeoutExpired:
            server.terminate()
            native = server.wait(timeout=2)
            reason = reason or "server-terminated-after-stdin-close"
        while any(thread.is_alive() for thread in threads) and time.monotonic() < started + 50:
            for thread in threads:
                thread.join(timeout=0.05)
        if any(thread.is_alive() for thread in threads):
            reason = reason or "final-drain-deadline"
            status = "stopped"
        if raw_limit.is_set():
            reason = reason or "raw-output-limit"
            status = "stopped"
        # Final drain classifies every queued stdout method frame, including
        # frames received after the last reply. No request is ever answered.
        while not messages.empty():
            kind, raw = messages.get_nowait()
            if kind == "stderr":
                stderr_bytes += len(raw)
                if re.search(rb"refresh|login|log.in|authenticat|unauthorized|credential", raw, re.I):
                    reason = reason or "login-refresh-auth-signal; raw discarded"
                    status = "stopped"
            elif raw:
                try:
                    value = json.loads(raw)
                    if isinstance(value, dict) and "method" in value:
                        process_server_frame(value)
                    else:
                        raise StopSession("unexpected-output-after-reads")
                except StopSession as exc:
                    reason = reason or str(exc)
                    status = "stopped"
                except Exception:
                    reason = reason or "invalid-json-after-reads; raw discarded"
                    status = "stopped"
        result = {"status": status, "stopReason": reason, "sentRpc": sent, "observations": observations, "serverMethodEvents": method_events, "notificationCount": notification_count, "provisionalConfig": provisional, "nativeReturnCode": native, "wallSeconds": time.monotonic()-started, "rawResponseBytesDiscarded": response_bytes, "stderrBytesDiscarded": stderr_bytes, "totalRawBytesConsumedInMemory": raw_bytes, "rawByteLimit": 4_000_000, "queueCapacity": 32, "rawConfigurationPersisted": False, "rawFramePayloadsPersisted": False, "automaticRetries": 0}
        (ROOT / "sanitized-result.json").write_text(json.dumps(result, indent=2)+"\n", encoding="utf-8")
        print(json.dumps({"status": status, "stopReason": reason, "nativeReturnCode": native, "wallSeconds": result["wallSeconds"]}))
    return 0 if status == "read-responses-received" and reason is None else 1


if __name__ == "__main__":
    raise SystemExit(main())
