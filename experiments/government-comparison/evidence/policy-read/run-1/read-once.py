"""One fixed, sanitizing config-read session. No reusable runner or raw logs."""
import hashlib
import json
import os
from pathlib import Path
import queue
import re
import subprocess
import sys
import threading
import time

ROOT = Path(__file__).resolve().parent
REQUEST = json.loads((ROOT / "request.json").read_bytes())
assert hashlib.sha256(Path(__file__).read_bytes()).hexdigest() == REQUEST["clientSha256"]
EXE = Path(REQUEST["argv"][0])
assert hashlib.sha256(EXE.read_bytes()).hexdigest() == REQUEST["executableSha256"]
assert REQUEST["argv"][1:3] == ["app-server", "--stdio"]
frozen_bytes = Path(REQUEST["frozenExecProcess"]).read_bytes()
assert hashlib.sha256(frozen_bytes).hexdigest() == REQUEST["frozenExecProcessSha256"]
frozen_process = json.loads(frozen_bytes)
frozen_argv = frozen_process["argv"]
overrides = [part for i, value in enumerate(frozen_argv) if value == "--config"
             for part in ("--config", frozen_argv[i + 1])]
assert REQUEST["argv"] == [frozen_argv[0], "app-server", "--stdio", *overrides]
assert REQUEST["cwd"] == frozen_process["cwd"]
assert REQUEST["rpc"] == [
    {"method": "initialize", "id": 0, "params": {"clientInfo": {"name": "scientist_policy_read_once", "version": "1.0"}}},
    {"method": "initialized", "params": {}},
    {"method": "config/read", "id": 1, "params": {"includeLayers": True, "cwd": REQUEST["cwd"]}},
    {"method": "configRequirements/read", "id": 2},
]
FLAGS = {"apps", "goals", "hooks", "memories", "multi_agent", "plugins", "shell_tool", "unified_exec", "network_proxy", "windows_sandbox_service", "powershell_shell_version"}
POLICY_KEYS = {"approval_policy", "approvals_reviewer", "sandbox_mode", "sandbox_workspace_write", "windows", "default_permissions", "permissions", "features"}
ENUMS = {"never", "on-request", "untrusted", "user", "auto_review", "read-only", "workspace-write", "danger-full-access", "elevated", "unelevated", "mxc", "read", "write", "deny", "allow", "restricted", "enabled"}


def scalar(value):
    return value is None or type(value) in (bool, int) or (type(value) is str and value in ENUMS)


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


def main():
    marker = ROOT / "started-once.json"
    with marker.open("x", encoding="utf-8") as f:
        json.dump({"startedAtUnix": time.time(), "clientSha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest()}, f)
    started = time.monotonic()
    deadline = started + 50
    messages = queue.Queue(maxsize=32)
    raw_limit = threading.Event()
    raw_lock = threading.Lock()
    raw_bytes = 0
    stderr_bytes = 0
    response_bytes = 0
    observations = []
    sent = []
    status = "stopped"
    reason = None
    native = None
    env = {k: v for k, v in os.environ.items() if k not in ("OPENAI_API_KEY", "CODEX_API_KEY")}
    server = subprocess.Popen(REQUEST["argv"], cwd=REQUEST["cwd"], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, shell=False, env=env)

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

    def consume(kind, raw, identifier):
        nonlocal stderr_bytes, response_bytes
        if kind == "stderr":
            stderr_bytes += len(raw)
            if re.search(rb"refresh|login|log.in|authenticat|unauthorized|credential", raw, re.I):
                raise RuntimeError("login-refresh-auth-signal; raw discarded")
            return None
        response_bytes += len(raw)
        if not raw:
            raise RuntimeError("server-stdout-closed")
        if len(raw) > 2_000_000:
            raise RuntimeError("response-size-limit")
        try:
            value = json.loads(raw)
        except Exception:
            raise RuntimeError("invalid-json; raw discarded") from None
        if "method" in value:
            raise RuntimeError("unexpected-server-method; contents discarded")
        if value.get("id") != identifier:
            raise RuntimeError("unexpected-response-id; contents discarded")
        if "error" in value:
            raise RuntimeError("rpc-error; contents discarded")
        if not isinstance(value.get("result"), dict):
            raise RuntimeError("unexpected-result-shape")
        return sanitize(identifier, value["result"])

    try:
        for rpc in REQUEST["rpc"]:
            if raw_limit.is_set():
                raise RuntimeError("raw-output-limit")
            while not messages.empty():
                kind, raw = messages.get_nowait()
                if consume(kind, raw, None) is not None:
                    raise RuntimeError("unsolicited-response")
            if time.monotonic() >= deadline:
                raise RuntimeError("client-deadline")
            server.stdin.write((json.dumps(rpc) + "\n").encode())
            server.stdin.flush()
            sent.append(rpc)
            if "id" not in rpc:
                continue
            while True:
                if raw_limit.is_set():
                    raise RuntimeError("raw-output-limit")
                if time.monotonic() >= deadline:
                    raise RuntimeError("client-deadline")
                try:
                    kind, raw = messages.get(timeout=0.1)
                except queue.Empty:
                    continue
                result = consume(kind, raw, rpc["id"])
                if result is not None:
                    observations.append({"id": rpc["id"], "method": rpc["method"], "sanitizedResult": result})
                    break
        status = "read-responses-received"
    except Exception as exc:
        reason = str(exc) if type(exc) is RuntimeError else "local-client-exception; details discarded"
    finally:
        server.stdin.close()
        try:
            native = server.wait(timeout=2)
        except subprocess.TimeoutExpired:
            server.terminate()
            native = server.wait(timeout=2)
            reason = reason or "server-terminated-after-stdin-close"
        for thread in threads:
            thread.join(timeout=0.2)
        if raw_limit.is_set():
            reason = reason or "raw-output-limit"
            status = "stopped"
        while not messages.empty():
            kind, raw = messages.get_nowait()
            if kind == "stderr":
                stderr_bytes += len(raw)
                if re.search(rb"refresh|login|log.in|authenticat|unauthorized|credential", raw, re.I):
                    reason = reason or "login-refresh-auth-signal; raw discarded"
                    status = "stopped"
            elif raw:
                reason = reason or "unexpected-output-after-reads; raw discarded"
                status = "stopped"
        result = {"status": status, "stopReason": reason, "sentRpc": sent, "observations": observations, "nativeReturnCode": native, "wallSeconds": time.monotonic()-started, "rawResponseBytesDiscarded": response_bytes, "stderrBytesDiscarded": stderr_bytes, "totalRawBytesConsumedInMemory": raw_bytes, "rawByteLimit": 4000000, "queueCapacity": 32, "rawConfigurationPersisted": False, "automaticRetries": 0}
        (ROOT / "sanitized-result.json").write_text(json.dumps(result, indent=2)+"\n", encoding="utf-8")
        print(json.dumps({"status": status, "stopReason": reason, "nativeReturnCode": native, "wallSeconds": result["wallSeconds"]}))
    return 0 if status == "read-responses-received" and reason is None else 1


if __name__ == "__main__":
    raise SystemExit(main())
