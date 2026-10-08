"""Closed one-shot public read-tool client. Raw server bytes exist only in bounded memory.

Derived from the preserved classified client; no imports start processes.
The controller assigns a Windows Job to a gated Python worker before it can
launch the one app-server. Only that Job's owned process tree can be terminated.
"""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import queue
import re
import subprocess
import sys
import threading
import time
import copy
import math

ROOT = Path(__file__).resolve().parent
PACKAGE = ROOT.parents[1]
REPO = PACKAGE.parents[1]
KEY = "s1-common-runner-read-tool-20261008-r1"
SOURCE_THREAD = "01a11367-a781-7683-a20f-46e12614dcb4"
ISSUED = "2026-10-08T06:05:52Z"
EXE = "C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/5ea220ae823df3d7/codex.exe"
EXE_SHA = "3b8f6e33caa75f232558a3cf76ff9b87bb5ef6dbcf4996372f24e55c78b1b916"
CWD = "C:/Users/Consiliari/Documents/Scientist-Probes/s1-common-runner-read-tool-20261008-r1/actor"
PRIOR = PACKAGE / "evidence/policy-read-disabled-status/run-1/classified-client.py"
PRIOR_SHA = "95c8486ad8962e14ad0c2da2989adcf154fa28f51d19c1fe983c532d1a885926"
PROCESS = PACKAGE / "runtime/process.py"
PROCESS_SHA = "d0cedb71d57be87095d9af119d9769fb0c3d311e1af073cba81f6337743c78ae"
SCHEMA = PACKAGE / "evidence/policy-compatibility/run-1/protocol-schema.zip"
METHOD_ENUMS_SHA = "6553df9ac4a37d11402728992ed5e684ea379fbadeee790700867ae4542594cf"
FLAGS = {"apps", "goals", "hooks", "memories", "multi_agent", "plugins", "shell_tool", "unified_exec", "windows_sandbox_service", "network_proxy", "powershell_shell_version"}
ENUMS = {"approval_policy": {"never", "on-request", "untrusted", "on-failure"}, "approvals_reviewer": {"user", "auto_review"}, "sandbox_mode": {"read-only", "workspace-write", "danger-full-access"}, "windows": {"elevated", "unelevated"}, "model": {"gpt-6.1-sol"}, "model_provider": {"openai"}, "model_reasoning_effort": {"high"}}
LIMITS = dict(metadataHandshakeSeconds=33, threadStartSeconds=15, turnSeconds=120, ownedProcessExecutionSeconds=180, maxCleanupSeconds=10, controllerWithCleanupSeconds=195, outerReceiptSeconds=210, maxQueuedFrames=64, maxSingleFrameBytes=1000000, maxRawBytes=8000000, maxInboundNotifications=2048, tokenStopThreshold=50000)


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def load_module(path, expected, name):
    require(sha(path.read_bytes()) == expected, "source-pin-mismatch")
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def strict_json(raw):
    def pairs(items):
        out = {}
        for k, v in items:
            require(k not in out, "duplicate-json-key")
            out[k] = v
        return out
    return json.loads(raw, object_pairs_hook=pairs, parse_float=lambda x: finite_float(x), parse_constant=lambda _: (_ for _ in ()).throw(ValueError("invalid-json-number")))


def finite_float(value):
    number = float(value)
    require(math.isfinite(number), "invalid-json-number")
    return number


def field(obj, key, convert):
    if key not in obj:
        return {"state": "missing"}
    if obj[key] is None:
        return {"state": "null"}
    try:
        return {"state": "present", "value": convert(obj[key])}
    except (ValueError, TypeError, KeyError):
        return {"state": "invalid"}


def enum(value, allowed):
    require(type(value) is str and value in allowed, "invalid-enum")
    return value


def boolean(value):
    require(type(value) is bool, "invalid-boolean")
    return value


def profile_id(value):
    require(type(value) is str and re.fullmatch(r"[A-Za-z0-9:_-]{1,100}", value), "invalid-profile-id")
    return value if value == ":read-only" else "opaque-" + sha(value.encode())[:24]


def profile_map(value):
    require(type(value) is dict and len(value) <= 32, "invalid-profile-map")
    return {profile_id(k): boolean(v) for k, v in value.items()}


def profile_summary(value):
    require(type(value) is dict and len(value) <= 32, "invalid-profiles")
    require(all(type(v) is dict for v in value.values()), "invalid-profiles")
    return {profile_id(k): {"definitionPresent": True} for k in value}


ADAPTER = PACKAGE / "runtime/selected_feature_contract.py"
ADAPTER_SHA = "985ff1d8f8870de4ac697e030c2abeb543d1bc3e74f475968c5e1097be865550"


def adapter():
    return load_module(ADAPTER, ADAPTER_SHA, "accepted_selected_feature_contract")


def sanitize_config_read(result):
    return adapter().sanitize_config_read(result)


def sanitize_requirements(result):
    return adapter().sanitize_requirements(result)


def assess(config, requirements):
    return adapter().assess(config, requirements)


def assess_config(config):
    # Preliminary only. This placeholder can never satisfy final_assess.
    status, reason = assess(config, {"requirements": {"state": "null"}})
    return ("config-precheck-satisfied-awaiting-requirements", None) if status == "visible-metadata-precondition-satisfied" and reason is None else (status, reason)


def final_assess(observations, sent):
    expected = [("initialize", 0), ("initialized", None), ("config/read", 1), ("configRequirements/read", 2)]
    if [(x.get("method"), x.get("id")) for x in sent] != expected or [x.get("id") for x in observations] != [0, 1, 2]:
        return "insufficient", "actual-requirements-response-required"
    actual = observations[2]
    if actual.get("method") != "configRequirements/read" or actual.get("responseEnvelopeValidated") is not True:
        return "insufficient", "actual-requirements-response-required"
    status, reason = assess(observations[1]["sanitizedResult"], actual["sanitizedResult"])
    return ("reported-config-and-requirements-observed", None) if status == "visible-metadata-precondition-satisfied" and reason is None else (status, reason)


def decode_response(frame, identifier):
    prior = load_module(PRIOR, PRIOR_SHA, "preserved_response_guard")
    return sanitize(identifier, prior.validate_response_envelope(frame, identifier))


def classify(frame, enums, count):
    prior = load_module(PRIOR, PRIOR_SHA, "preserved_classifier")
    result = prior.classify_server_frame(frame, enums, count)
    if not isinstance(frame, dict) or set(frame) - {"method", "params", "emittedAtMs"} or "result" in frame or "error" in frame:
        return {**result, "action": "stop", "reason": "unexpected-frame-envelope"}
    if frame.get("method") != "remoteControl/status/changed":
        return {**result, "action": "stop", "reason": "warning-or-unexpected-method"}
    return result


def reject_provisional(value):
    if isinstance(value, dict):
        for k, v in value.items():
            if k.lower() in {"warning", "warnings", "configwarning", "provisional", "provisionalconfig"} and v not in (None, False, [], {}):
                raise ValueError("warning-or-provisional-result")
            reject_provisional(v)
    elif isinstance(value, list):
        for v in value:
            reject_provisional(v)


def sanitize(identifier, result):
    return adapter().sanitize(identifier, result)


def validate_profile(profile, grant, plan=None):
    previous_path = PACKAGE / "evidence/s1-selected-feature-metadata-20261008-r3/profile.json"
    require(sha(previous_path.read_bytes()) == "daf3f2b775e73cb194f48f8731e73f82a93ca0b7f2976b980ee7ed45110fb416", "prior-profile-pin-mismatch")
    previous = strict_json(previous_path.read_bytes())
    rpc = copy.deepcopy(previous["rpc"])
    rpc[0]["params"]["clientInfo"]["name"] = "scientist_common_runner_read_tool_r1"
    rpc[0]["params"]["capabilities"] = {"experimentalApi": True}
    rpc[2]["params"]["cwd"] = CWD
    require(profile["key"] == grant["key"] == KEY, "grant-key-mismatch")
    require(grant["sourceThreadId"] == SOURCE_THREAD and grant["issuedUtc"] == ISSUED and grant["ownerThreadId"] == "01a1169c-3df6-7571-997d-48ab57365875" and grant["baseSha"] == "f24b7e097564d771d3ad438ab6b5f9f5321b6301", "grant-authority-mismatch")
    require(grant["executable"] == EXE and grant["executableSha256"] == EXE_SHA and grant["selectedFeatureAdapterSha256"] == ADAPTER_SHA, "grant-source-or-binary-mismatch")
    require(profile["argv"] == previous["argv"] and profile["argv"].count("--config") == 17, "argv-mismatch")
    require(profile["rpc"] == rpc and profile["cwd"] == grant["cwd"] == CWD and profile["externalEvidence"] == grant["evidenceDirectory"], "rpc-or-cwd-mismatch")
    require(profile["limits"] == LIMITS and all(grant[k] == v for k, v in LIMITS.items()), "limits-mismatch")
    require(grant["clientMethods"] == ["initialize", "initialized", "config/read", "configRequirements/read", "thread/start", "turn/start"] and grant["initializeCapabilities"] == {"experimentalApi": True}, "method-capability-mismatch")
    require(all(grant[k] == 1 for k in ("maxNewAppServerTrees", "maxNewActorReservations", "maxThreadStarts", "maxTurnStarts", "maxObservedTaskToolItems", "maxTurnInterruptsOnStop", "maxParallel", "maxConfigReads", "maxRequirementsReads")), "grant-count-mismatch")
    require(all(grant[k] == 0 for k in ("retries", "maxAdditionalCliMetadataCalls", "maxProducts", "maxProductWrappers", "maxDelegates", "maxStudyCells", "maxFullProductSuites", "maxWritesByActor")), "grant-forbidden-starts")
    require(grant["previousMetadataOnlyTrees"] == grant["previousActorStarts"] == 5 and grant["cumulativeAppServerTreesMaximum"] == grant["cumulativeActorReservationsMaximum"] == 6, "grant-history-mismatch")
    thread = {"method": "thread/start", "id": 3, "params": {"model": "gpt-6.1-sol", "modelProvider": "openai", "approvalPolicy": "never", "permissions": ":read-only", "ephemeral": True, "cwd": CWD}}
    require(profile["threadStart"] == thread and grant["threadStart"] == {k:v for k,v in thread["params"].items() if k != "cwd"}, "thread-start-mismatch")
    turn = {"method": "turn/start", "id": 4, "params": {"threadId": "$threadId", "effort": "high", "input": [{"type": "text", "text": profile["turnPrompt"]}]}}
    require(profile["turnStart"] == turn, "turn-start-mismatch")
    require(profile["command"] == "Get-Content -LiteralPath '" + CWD + "/probe-input.txt' -Raw" and profile["command"] in profile["turnPrompt"], "frozen-command-mismatch")
    require(profile["interrupt"] == {"method": "turn/interrupt", "id": 5, "params": {"threadId": "$threadId", "turnId": "$turnId"}}, "interrupt-mismatch")


def validate_cwd(profile):
    cwd = Path(profile["cwd"])
    require(cwd.is_dir() and not cwd.is_symlink() and not cwd.is_junction(), "cwd-invalid")
    files = list(cwd.rglob("*"))
    require(all(p.is_file() and not p.is_symlink() and not p.is_junction() for p in files), "unexpected-cwd-entry")
    actual = {p.relative_to(cwd).as_posix(): sha(p.read_bytes()) for p in files}
    require(actual == profile["expectedFiles"], "cwd-content-mismatch")
    return actual


def validate_live_authority(grant, slot, original):
    require(all(grant.get(k) == v for k, v in original.items() if k != "status"), "live-grant-mismatch")
    require(grant.get("status") == "Dispatched once; awaiting direct terminal callback.", "inactive-live-grant")
    require(slot.get("owner") == "Scientist" and slot.get("key") == KEY and slot.get("assignedUtc") == ISSUED, "slot-mismatch")
    require(slot.get("status") == "Assigned to bounded real read-only Actor smoke; no full suite.", "inactive-live-slot")


def load_request():
    request = strict_json((ROOT / "request.json").read_bytes())
    freeze_bytes = (ROOT / "freeze.json").read_bytes(); freeze = strict_json(freeze_bytes)
    profile = strict_json((ROOT / "profile.json").read_bytes())
    authority = strict_json((ROOT / "authorization-grant.json").read_bytes())
    plan_path = PACKAGE / "public/s1-common-runner-readiness-plan-20261008.md"
    validate_profile(profile, authority["grant"], plan_path.read_text())
    require(sha(ADAPTER.read_bytes()) == authority["grant"]["selectedFeatureAdapterSha256"] == ADAPTER_SHA, "adapter-pin-mismatch")
    live = strict_json(Path(authority["sourceCoordinationPath"]).read_bytes())
    scientist = next(x for x in live["threads"] if x["name"] == "Scientist")
    grant = scientist["evidence"]["commonRunnerReadToolGrant"]
    slot = live["fullSuiteSlot"]
    validate_live_authority(grant, slot, authority["grant"])
    reservation_path = Path(profile["externalEvidence"]) / "reservation.json"
    reservation = strict_json(reservation_path.read_bytes())
    git = lambda *a: subprocess.check_output(["git", *a], cwd=REPO)
    head = git("rev-parse", "HEAD").decode().strip()
    require(not git("status", "--porcelain").strip() and reservation["freezeCommit"] == head, "unclean-or-wrong-freeze-commit")
    require(request["key"] == freeze["key"] == reservation["key"] == KEY, "binding-key-mismatch")
    require(reservation["requestSha256"] == sha((ROOT / "request.json").read_bytes()) and reservation["freezeSha256"] == sha(freeze_bytes), "reservation-binding-mismatch")
    require(freeze["requestSha256"] == sha((ROOT / "request.json").read_bytes()), "freeze-request-mismatch")
    require(request["sourceCommit"] == freeze["sourceCommit"] == reservation["sourceCommit"], "source-commit-mismatch")
    require(reservation["maxTrees"] == reservation["maxActors"] == 1 and reservation["priorAppServerTrees"] == reservation["priorActors"] == 5, "reservation-limits-mismatch")
    require(request["profileSha256"] == sha((ROOT / "profile.json").read_bytes()), "profile-pin-mismatch")
    require(request["authorizationGrantSha256"] == sha((ROOT / "authorization-grant.json").read_bytes()), "grant-pin-mismatch")
    require(sha((ROOT / "frozen-method-enums.json").read_bytes()) == METHOD_ENUMS_SHA, "method-schema-pin-mismatch")
    require(sha(SCHEMA.read_bytes()) == authority["grant"]["schemaArchiveSha256"], "schema-archive-pin-mismatch")
    require(Path(sys.executable).resolve() == Path(request["interpreter"]["path"]).resolve() and sha(Path(sys.executable).read_bytes()) == request["interpreter"]["sha256"], "interpreter-pin-mismatch")
    required_sources = {str((ROOT / name).relative_to(REPO).as_posix()) for name in ("client.py", "run_once.py", "collector.py", "protocol.py", "schema-contract.json", "expected.json", "input-binding.json", "profile.json", "authorization-grant.json", "frozen-method-enums.json", "test_client.py", "preflight-review.md", "offline-validation.json")}
    required_sources.add(ADAPTER.relative_to(REPO).as_posix())
    require(required_sources <= set(request["sourceFiles"]), "missing-required-source-pins")
    required_inputs = {str(ROOT / "request.json"), EXE, str(PRIOR), str(PROCESS), str(ADAPTER), str(SCHEMA), str(plan_path), str(ROOT / "frozen-method-enums.json"), str(Path(CWD) / "README.md"), str(Path(CWD) / "probe-input.txt"), request["interpreter"]["path"]}
    require(required_inputs <= set(freeze["files"]), "missing-required-freeze-inputs")
    ledger_pins = strict_json((PACKAGE / "evidence/s1-common-runner-readiness-plan-20261008/source-evidence.json").read_bytes())["liveLedgerPinsReadOnly"]
    require(len(ledger_pins) == 5 and all(freeze["files"].get(p["path"]) == p["sha256"] for p in ledger_pins), "historical-ledger-freeze-mismatch")
    for path, digest in freeze["files"].items():
        require(sha(Path(path).read_bytes()) == digest, "frozen-file-pin-mismatch")
    for rel, digest in request["sourceFiles"].items():
        require(sha((REPO / rel).read_bytes()) == digest, "source-file-pin-mismatch")
        require(sha(git("show", request["sourceCommit"] + ":" + rel)) == digest, "source-git-pin-mismatch")
    require(sha(Path(EXE).read_bytes()) == EXE_SHA, "binary-pin-mismatch")
    inventory = validate_cwd(profile)
    expected = strict_json((ROOT / "expected.json").read_bytes())
    require(re.fullmatch(r"[0-9a-f]{32}", expected["sentinel"]) is not None and expected["sentinel"] not in profile["turnPrompt"] and expected["sentinel"] not in (Path(CWD)/"README.md").read_text(), "sentinel-prompt-leak")
    require((Path(CWD)/"probe-input.txt").read_bytes() == (expected["sentinel"]+"\n").encode("ascii") and expected["fileSha256"] == inventory["probe-input.txt"], "public-input-mismatch")
    return profile, {"requestSha256": sha((ROOT / "request.json").read_bytes()), "freezeSha256": sha(freeze_bytes), "reservationSha256": sha(reservation_path.read_bytes()), "sourceCommit": request["sourceCommit"], "freezeCommit": head}


def exclusive_json(path, value):
    with Path(path).open("x", encoding="utf-8") as f:
        json.dump(value, f, indent=2); f.write("\n"); f.flush(); os.fsync(f.fileno())


def interrupt_request(active_ids, turn_response_validated, already_sent):
    if already_sent or not turn_response_validated or type(active_ids) is not dict or set(active_ids) != {"threadId", "turnId"}:
        return None
    if not all(type(v) is str and re.fullmatch(r"[A-Za-z0-9:_-]{1,128}", v) for v in active_ids.values()):
        return None
    return {"method": "turn/interrupt", "id": 5, "params": dict(active_ids)}


def route_notification(frame, protocol, collector, enums, metadata_only):
    require(type(frame) is dict and set(frame) <= {"method", "params", "emittedAtMs"} and type(frame.get("method")) is str and "params" in frame, "server-request-or-notification-envelope")
    if "emittedAtMs" in frame:
        require(type(frame["emittedAtMs"]) is int and -(2**63) <= frame["emittedAtMs"] < 2**63, "notification-envelope-timestamp")
    if frame["method"] == "remoteControl/status/changed":
        result = classify(frame, enums, 0)
        require(result["action"] == "discard-notification", result["reason"] or "remote-control-status-not-disabled")
        return {"method": "remoteControl/status/changed", "status": "disabled"}
    require(not metadata_only, "lifecycle-before-thread-request")
    reject_provisional(frame["params"])
    collector.accept_notification(frame["method"], frame["params"])
    require(collector.failure_reason is None, collector.failure_reason or "collector-nonpositive")
    return {"method": frame["method"]}


def session():
    require(sys.stdin.buffer.readline() == b"GO\n", "missing-job-gate")
    profile, binding = load_request()
    out = Path(profile["externalEvidence"])
    job_receipt = strict_json((out / "job-assigned.json").read_bytes())
    require(job_receipt["workerPid"] == os.getpid() and job_receipt["controllerPid"] == os.getppid(), "job-parent-or-worker-mismatch")
    require(all(job_receipt.get(k) == v for k, v in binding.items()), "job-binding-mismatch")
    source_pins = strict_json((ROOT / "request.json").read_bytes())["sourceFiles"]
    def packet_module(name):
        path = ROOT / name
        return load_module(path, source_pins[path.relative_to(REPO).as_posix()], "frozen_"+name[:-3])
    contract = strict_json((ROOT / "schema-contract.json").read_bytes())
    protocol = packet_module("protocol.py").Protocol(SCHEMA, contract)
    collector = packet_module("collector.py").Collector(profile, strict_json((ROOT / "expected.json").read_bytes()), protocol.validate)
    prior = load_module(PRIOR, PRIOR_SHA, "preserved_metadata_guard")
    enums = prior.load_method_enums(ROOT / "frozen-method-enums.json")
    prior.validate_disabled_status_gate(enums)
    started = time.monotonic()
    exclusive_json(out / "started-once.json", {"key": KEY, "startedAtUnix": time.time(), **binding})
    messages = queue.Queue(maxsize=64); limit = threading.Event(); lock = threading.Lock()
    counts = {"raw": 0, "stdout": 0, "stderr": 0}; in_flight = 0
    observations = []; sent = []; event_counts = {}; notification_count = 0
    server = None; threads = []; reason = None; status = "stopped"; native = None
    preliminary = None; metadata_final = None; thread_observation = None
    turn_response_validated = False; interrupt_sent = False; interrupt_receipt = None
    phase = "metadata"; phase_times = {}; actor_result = None

    def pump(stream, kind):
        nonlocal in_flight
        pending = b""
        while True:
            with lock:
                allowance = min(8192, 8000000-counts["raw"]-in_flight)
                if allowance <= 0:
                    limit.set(); return
                in_flight += allowance
            raw = stream.read1(allowance)
            with lock:
                in_flight -= allowance
                counts["raw"] += len(raw); counts[kind] += len(raw)
            if kind == "stderr":
                try:
                    messages.put_nowait((kind, raw))
                except queue.Full:
                    limit.set(); return
                if not raw:
                    return
                continue
            pending += raw
            while b"\n" in pending:
                line, pending = pending.split(b"\n", 1)
                if len(line)+1 > 1000000:
                    limit.set(); return
                try:
                    messages.put_nowait((kind, line+b"\n"))
                except queue.Full:
                    limit.set(); return
            if len(pending) > 1000000:
                limit.set(); return
            if not raw:
                try:
                    if pending:
                        messages.put_nowait((kind, pending))
                    messages.put_nowait((kind, b""))
                except queue.Full:
                    limit.set()
                return

    def consume(kind, raw, expected=None):
        nonlocal notification_count, interrupt_receipt
        if kind == "stderr":
            require(not raw, "server-stderr-activity"); return None
        require(raw and len(raw) <= 1000000, "stdout-closed-or-frame-limit")
        frame = strict_json(raw)
        if type(frame) is dict and "method" in frame:
            notification_count += 1
            require(notification_count <= 2048, "notification-limit")
            event = route_notification(frame, protocol, collector, enums, phase == "metadata")
            event_counts[event["method"]] = event_counts.get(event["method"], 0)+1
            return None
        require(expected is not None, "unsolicited-response")
        result = prior.validate_response_envelope(frame, expected["id"])
        reject_provisional(result)
        member = contract["responseSchemas"][expected["method"]]
        protocol.validate(member, result)
        return result

    def send(rpc):
        require(rpc["method"] in {"initialize", "initialized", "config/read", "configRequirements/read", "thread/start", "turn/start", "turn/interrupt"}, "forbidden-client-method")
        require(not any(x["method"] == rpc["method"] for x in sent), "duplicate-client-method")
        member = contract["requestSchemas"].get(rpc["method"])
        if member:
            protocol.validate(member, rpc["params"])
        elif rpc["method"] == "configRequirements/read":
            require("params" not in rpc, "unexpected-requirements-params")
        entry = {"method": rpc["method"], "id": rpc.get("id"), "writeCompleted": False}
        sent.append(entry)
        server.stdin.write((json.dumps(rpc)+"\n").encode()); server.stdin.flush(); entry["writeCompleted"] = True

    def request(rpc, deadline):
        require(not limit.is_set() and time.monotonic() < deadline, "rpc-resource-or-deadline")
        while not messages.empty():
            require(consume(*messages.get_nowait()) is None, "unsolicited-response")
        send(rpc)
        if "id" not in rpc:
            return None
        while True:
            require(not limit.is_set() and time.monotonic() < deadline and time.monotonic() < started+180, "rpc-resource-or-deadline")
            try:
                kind, raw = messages.get(timeout=.05)
            except queue.Empty:
                continue
            result = consume(kind, raw, rpc)
            if result is not None:
                return result

    try:
        env = {k:v for k,v in os.environ.items() if k not in {"OPENAI_API_KEY", "CODEX_API_KEY"}}
        server = subprocess.Popen(profile["argv"], cwd=profile["cwd"], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, shell=False, env=env)
        for stream, kind in ((server.stdout, "stdout"), (server.stderr, "stderr")):
            t = threading.Thread(target=pump, args=(stream, kind), daemon=True);threads.append(t);t.start()
        for rpc in profile["rpc"]:
            raw_result = request(rpc, started+33)
            if raw_result is None:
                continue
            sanitized = sanitize(rpc["id"], raw_result)
            observations.append({"id":rpc["id"], "method":rpc["method"], "responseEnvelopeValidated":True, "sanitizedResult":sanitized})
            if rpc["id"] == 1:
                preliminary, why = assess_config(sanitized)
                require(preliminary == "config-precheck-satisfied-awaiting-requirements", why or "config-precheck-nonpositive")
        metadata_final, why = final_assess(observations, sent)
        require(metadata_final == "reported-config-and-requirements-observed", why or "metadata-precheck-nonpositive")
        phase_times["metadataSeconds"] = time.monotonic()-started
        phase = "thread"; stage_started = time.monotonic()
        raw_thread = request(profile["threadStart"], min(stage_started+15, started+180))
        thread_observation = {"responseEnvelopeValidated":True, "gatePassed":False}
        collector.accept_response("thread/start", raw_thread)
        require(collector.failure_reason is None, collector.failure_reason or "thread-gate-nonpositive")
        active = raw_thread.get("activePermissionProfile")
        thread_observation = {"responseEnvelopeValidated":True, "gatePassed":True, "threadId":collector.active_ids["threadId"], "cwdMatches":True, "model":"gpt-6.1-sol", "modelProvider":"openai", "approvalPolicy":"never", "ephemeral":True, "sandbox":{"type":"readOnly", "networkAccess":{"state":"present", "value":raw_thread["sandbox"]["networkAccess"]} if "networkAccess" in raw_thread["sandbox"] else {"state":"missing"}}, "activePermissionProfile":{"state":"missing"} if "activePermissionProfile" not in raw_thread else {"state":"null"} if active is None else {"state":"present", "value":{"id":":read-only", "extends":active.get("extends")}}}
        phase_times["threadStartSeconds"] = time.monotonic()-stage_started
        phase = "turn"; stage_started = time.monotonic(); turn_deadline = min(stage_started+120, started+180)
        turn_rpc = copy.deepcopy(profile["turnStart"]);turn_rpc["params"]["threadId"] = collector.active_ids["threadId"]
        raw_turn = request(turn_rpc, turn_deadline);turn_response_validated = True
        collector.accept_response("turn/start", raw_turn)
        require(collector.failure_reason is None, collector.failure_reason or "turn-gate-nonpositive")
        while not collector.terminal:
            require(not limit.is_set() and time.monotonic() < turn_deadline, "turn-resource-or-deadline")
            try:
                kind, raw = messages.get(timeout=.05)
            except queue.Empty:
                continue
            consume(kind, raw)
        require(collector.failure_reason is None, collector.failure_reason or "turn-nonpositive")
        phase_times["turnSeconds"] = time.monotonic()-stage_started
        status = "public-read-observed"
    except ValueError as exc:
        reason = str(exc) if re.fullmatch(r"[a-z-]{1,100}", str(exc)) else "invalid-or-unexpected-response"
    except BaseException:
        reason = "local-client-exception"
    finally:
        cleanup = time.monotonic(); cleanup_deadline = min(cleanup+8, started+178)
        if server:
            if reason:
                cancellation = interrupt_request(collector.active_ids, turn_response_validated, interrupt_sent)
                if cancellation:
                    interrupt_sent = True
                    try:
                        send(cancellation)
                        wait_until = min(time.monotonic()+2, cleanup_deadline)
                        while time.monotonic() < wait_until:
                            try:
                                kind, raw = messages.get(timeout=.05)
                            except queue.Empty:
                                continue
                            try:
                                reply = consume(kind, raw, cancellation)
                                if reply is not None:
                                    interrupt_receipt = {"responseEnvelopeValidated":True};break
                            except BaseException:
                                pass  # Original stop stays terminal; no further RPC or raw payload saved.
                    except BaseException:
                        interrupt_receipt = {"responseEnvelopeValidated":False}
            try:
                server.stdin.close();native = server.wait(timeout=max(.001, min(2,cleanup_deadline-time.monotonic())))
            except BaseException:
                reason = reason or "native-cleanup-required";status = "stopped"
                try:
                    server.kill();native = server.wait(timeout=max(.001,min(2,cleanup_deadline-time.monotonic())))
                except BaseException:
                    reason = "native-cleanup-failed"
        for t in threads:
            t.join(timeout=max(0,cleanup_deadline-time.monotonic()))
        if any(t.is_alive() for t in threads):
            reason = reason or "drain-deadline";status = "stopped"
        while not messages.empty():
            kind,raw = messages.get_nowait()
            if raw:
                try:
                    require(consume(kind,raw) is None,"unexpected-final-output")
                except BaseException:
                    reason = reason or "unexpected-final-output";status = "stopped"
        actor_result = collector.finish()
        if actor_result["status"] != "complete":
            reason = reason or actor_result["stopReason"];status = "stopped"
        if limit.is_set() or native != 0 or time.monotonic() >= started+180:
            reason = reason or "native-resource-exit-or-deadline";status = "stopped"
        try:
            validate_cwd(profile)
        except BaseException:
            reason = reason or "public-cwd-changed";status = "stopped"
        if reason:
            status = "stopped"
        result = {"key":KEY,"status":status,"stopReason":reason,**binding,"sentRpc":sent,"metadataObservations":observations,"configPrecheckStatus":preliminary,"metadataFinalStatus":metadata_final,"threadObservation":thread_observation,"turnStartResponseValidated":turn_response_validated,"collector":actor_result,"interruptSent":interrupt_sent,"interruptReceipt":interrupt_receipt,"providerCancellationConfirmed":False,"providerBillingStopped":None,"eventCounts":event_counts,"notificationCount":notification_count,"nativeProcessStarted":server is not None,"nativePid":server.pid if server else None,"nativeReturnCode":native,"terminalAtUnix":time.time(),"nativeSessionWallSeconds":time.monotonic()-started,"cleanupSeconds":time.monotonic()-cleanup,"phaseTimes":phase_times,"rawBytesDiscarded":counts,"rawFramePayloadsPersisted":False,"rawStderrPersisted":False,"nativeProviderRequests":None,"nativeProviderRetries":None,"servingModel":None,"tokenThresholdIsHardInflightCap":False,"automaticRetries":0,"meaning":"Only this prospective public read; no arbitrary reads, OS isolation, writes/build/test/general S1, historical denial cause or comparative quality"}
        exclusive_json(out/"sanitized-result.json",result)
    return 0 if status == "public-read-observed" and reason is None else 1



def controller():
    profile, binding = load_request()
    external = Path(profile["externalEvidence"])
    job_type = load_module(PROCESS, PROCESS_SHA, "existing_process_guard").WindowsJob
    started = time.monotonic(); worker = None; job = None; reason = None; code = None
    exclusive_json(external / "controller-once.json", {"key": KEY, **binding, "startedAtUnix": time.time()})
    try:
        worker = subprocess.Popen([sys.executable, str(Path(__file__).resolve()), "--session"], cwd=profile["cwd"], stdin=subprocess.PIPE, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, shell=False)
        job = job_type(worker)
        exclusive_json(external / "job-assigned.json", {"key": KEY, **binding, "controllerPid": os.getpid(), "workerPid": worker.pid, "processTreeControl": "windows-job-kill-on-close"})
        worker.stdin.write(b"GO\n"); worker.stdin.close()
        while worker.poll() is None:
            if time.monotonic()-started >= 180:
                reason = "owned-process-deadline"; job.kill(); break
            time.sleep(.01)
        code = worker.wait(timeout=10)
    except BaseException:
        reason = "controller-exception"
        if worker and worker.poll() is None:
            try:
                if job:
                    job.kill()
                else:
                    worker.kill()  # Still gated; target cannot have launched.
                code = worker.wait(timeout=10)
            except BaseException:
                reason = "controller-cleanup-failed"
    finally:
        cleanup = time.monotonic()
        if job:
            job.close()
        elapsed = time.monotonic()-started
        receipt = {"key": KEY, **binding, "workerReturnCode": code, "stopReason": reason, "controllerSeconds": elapsed, "jobCloseSeconds": time.monotonic()-cleanup, "processTreeControl": "windows-job-kill-on-close", "deadlineSeconds": 180, "controllerLimitSeconds": 195, "outerLimitSeconds": 210, "rawLogsPersisted": False, "automaticRetries": 0}
        result_path = external / "sanitized-result.json"
        receipt["terminalResultPresent"] = result_path.is_file()
        receipt["terminalResultSha256"] = sha(result_path.read_bytes()) if result_path.is_file() else None
        receipt["withinControllerBound"] = elapsed <= 195
        exclusive_json(external / "controller-receipt.json", receipt)
    return 0 if code == 0 and reason is None and elapsed <= 195 and result_path.is_file() else 1


if __name__ == "__main__":
    try:
        code = session() if sys.argv[1:] == ["--session"] else controller() if sys.argv[1:] == ["--controller"] else 2
    except BaseException:
        # No traceback or raw exception may reach a persisted log.
        code = 2
    raise SystemExit(code)
