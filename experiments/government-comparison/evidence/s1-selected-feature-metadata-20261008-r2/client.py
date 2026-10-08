"""Closed one-shot metadata client. Raw server bytes exist only in bounded memory.

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

ROOT = Path(__file__).resolve().parent
PACKAGE = ROOT.parents[1]
REPO = PACKAGE.parents[1]
KEY = "s1-selected-feature-metadata-20261008-r2"
SOURCE_THREAD = "01a11367-a781-7683-a20f-46e12614dcb4"
ISSUED = "2026-10-08T05:27:07Z"
EXE = "C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/5ea220ae823df3d7/codex.exe"
EXE_SHA = "3b8f6e33caa75f232558a3cf76ff9b87bb5ef6dbcf4996372f24e55c78b1b916"
CWD = "C:/Users/Consiliari/Documents/Scientist-Probes/s1-common-runner-prospective-20261008/actor"
PRIOR = PACKAGE / "evidence/policy-read-disabled-status/run-1/classified-client.py"
PRIOR_SHA = "95c8486ad8962e14ad0c2da2989adcf154fa28f51d19c1fe983c532d1a885926"
PROCESS = PACKAGE / "runtime/process.py"
PROCESS_SHA = "d0cedb71d57be87095d9af119d9769fb0c3d311e1af073cba81f6337743c78ae"
SCHEMA = PACKAGE / "evidence/policy-compatibility/run-1/protocol-schema.zip"
METHOD_ENUMS_SHA = "6553df9ac4a37d11402728992ed5e684ea379fbadeee790700867ae4542594cf"
FLAGS = {"apps", "goals", "hooks", "memories", "multi_agent", "plugins", "shell_tool", "unified_exec", "windows_sandbox_service", "network_proxy", "powershell_shell_version"}
ENUMS = {"approval_policy": {"never", "on-request", "untrusted", "on-failure"}, "approvals_reviewer": {"user", "auto_review"}, "sandbox_mode": {"read-only", "workspace-write", "danger-full-access"}, "windows": {"elevated", "unelevated"}, "model": {"gpt-6.1-sol"}, "model_provider": {"openai"}, "model_reasoning_effort": {"high"}}
LIMITS = dict(activeRpcSeconds=33, ownedProcessExecutionSeconds=38, controllerWithCleanupSeconds=50, outerReceiptSeconds=60, maxCleanupSeconds=10, maxQueuedFrames=32, maxRawBytes=4000000, maxInboundNotifications=16)


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
    return json.loads(raw, object_pairs_hook=pairs, parse_constant=lambda _: (_ for _ in ()).throw(ValueError("invalid-json-number")))


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
    return assess(config, {"requirements": {"state": "null"}})


def final_assess(observations, sent):
    expected = [("initialize", 0), ("initialized", None), ("config/read", 1), ("configRequirements/read", 2)]
    if [(x.get("method"), x.get("id")) for x in sent] != expected or [x.get("id") for x in observations] != [0, 1, 2]:
        return "insufficient", "actual-requirements-response-required"
    return assess(observations[1]["sanitizedResult"], observations[2]["sanitizedResult"])


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


def validate_profile(profile, grant, plan):
    previous_bytes = (PACKAGE / "evidence/s1-common-runner-policy-metadata-20261008/profile.json").read_bytes()
    require(sha(previous_bytes) == "de8de70e51b66ca4b7d3bf8a9c01ec577c462e258b824926ea9e6f85c1742589", "previous-profile-pin-mismatch")
    previous = strict_json(previous_bytes)
    argv = previous["argv"]
    rpc = previous["rpc"]
    rpc[0]["params"]["clientInfo"]["name"] = "scientist_selected_feature_policy_read_r2"
    require(profile["key"] == grant["key"] == KEY, "grant-key-mismatch")
    require(grant["sourceThreadId"] == SOURCE_THREAD and grant["issuedUtc"] == ISSUED, "grant-authority-mismatch")
    require(grant["ownerThreadId"] == "01a1169c-3df6-7571-997d-48ab57365875" and grant["baseSha"] == "822b5112f138b1921bfdbe01baecdd8944877035", "grant-owner-or-base-mismatch")
    require(grant["executable"] == EXE and grant["executableSha256"] == EXE_SHA and grant["preservedClientSha256"] == "8b6a2032dce5468c730885906f030fcfbed3cb4f23b010e95f90592a38b0fdfb" and grant["adapterSha256"] == ADAPTER_SHA, "grant-binary-or-client-mismatch")
    require(grant["allowedMethods"] == ["initialize", "initialized", "config/read", "configRequirements/read"] and grant["maxConfigReads"] == grant["maxRequirementsReads"] == grant["maxParallel"] == 1 and grant["retries"] == 0, "grant-method-or-count-mismatch")
    require(all(grant[k] == 0 for k in ("maxAdditionalCliMetadataCalls", "maxRealActors", "maxThreads", "maxTurns", "maxRequestedModelCalls", "maxNativeProducts", "maxProductWrappers", "maxDelegates", "maxStudyCells", "maxFullProductSuites")), "grant-forbidden-starts")
    require(profile["argv"] == argv and argv.count("--config") == 17, "argv-mismatch")
    require(profile["rpc"] == rpc and profile["cwd"] == CWD == grant["cwd"], "rpc-or-cwd-mismatch")
    require(profile["limits"] == LIMITS and all(grant[k] == v for k, v in LIMITS.items()), "limits-mismatch")
    require(grant["remainingSessions"] == 1 and grant["consumedSessions"] == 0 and grant["maxAdditionalAppServerSessions"] == 1, "closed-grant")
    require(grant["previousPolicyTreesConsumed"] == 4 and grant["cumulativePolicyTreesMaximum"] == 5, "policy-history-mismatch")
    require(profile["expectedFiles"] == grant["publicCwdInventory"] and profile["externalEvidence"] == grant["newEvidenceDirectory"], "cwd-or-evidence-binding-mismatch")


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
    require(grant.get("status") == "Dispatched once; conditional operation authorized, awaiting terminal callback.", "inactive-live-grant")
    require(slot.get("owner") == "Scientist" and slot.get("key") == KEY and slot.get("assignedUtc") == ISSUED, "slot-mismatch")
    require(slot.get("status") == "Assigned to one bounded corrected metadata operation; no product suite.", "inactive-live-slot")


def load_request():
    request = strict_json((ROOT / "request.json").read_bytes())
    freeze_bytes = (ROOT / "freeze.json").read_bytes(); freeze = strict_json(freeze_bytes)
    profile = strict_json((ROOT / "profile.json").read_bytes())
    authority = strict_json((ROOT / "authorization-grant.json").read_bytes())
    plan_path = PACKAGE / "public/s1-common-runner-readiness-plan-20261008.md"
    validate_profile(profile, authority["grant"], plan_path.read_text())
    require(sha(ADAPTER.read_bytes()) == authority["grant"]["adapterSha256"] == ADAPTER_SHA, "adapter-pin-mismatch")
    live = strict_json(Path(authority["sourceCoordinationPath"]).read_bytes())
    scientist = next(x for x in live["threads"] if x["name"] == "Scientist")
    grant = scientist["evidence"]["selectedFeatureMetadataR2Grant"]
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
    require(reservation["maxTrees"] == 1 and reservation["priorPolicyTrees"] == 4, "reservation-limits-mismatch")
    require(request["profileSha256"] == sha((ROOT / "profile.json").read_bytes()), "profile-pin-mismatch")
    require(request["authorizationGrantSha256"] == sha((ROOT / "authorization-grant.json").read_bytes()), "grant-pin-mismatch")
    require(sha((ROOT / "frozen-method-enums.json").read_bytes()) == METHOD_ENUMS_SHA, "method-schema-pin-mismatch")
    require(sha(SCHEMA.read_bytes()) == authority["grant"]["schemaArchiveSha256"], "schema-archive-pin-mismatch")
    require(Path(sys.executable).resolve() == Path(request["interpreter"]["path"]).resolve() and sha(Path(sys.executable).read_bytes()) == request["interpreter"]["sha256"], "interpreter-pin-mismatch")
    required_sources = {str((ROOT / name).relative_to(REPO).as_posix()) for name in ("client.py", "run_once.py", "profile.json", "authorization-grant.json", "frozen-method-enums.json", "test_client.py", "preflight-review.md", "offline-validation.json")}
    required_sources.add(ADAPTER.relative_to(REPO).as_posix())
    require(required_sources <= set(request["sourceFiles"]), "missing-required-source-pins")
    required_inputs = {str(ROOT / "request.json"), EXE, str(PRIOR), str(PROCESS), str(ADAPTER), str(SCHEMA), str(plan_path), str(ROOT / "frozen-method-enums.json"), str(Path(CWD) / "README.txt"), request["interpreter"]["path"]}
    require(required_inputs <= set(freeze["files"]), "missing-required-freeze-inputs")
    ledger_pins = strict_json((PACKAGE / "evidence/s1-common-runner-readiness-plan-20261008/source-evidence.json").read_bytes())["liveLedgerPinsReadOnly"]
    require(len(ledger_pins) == 5 and all(freeze["files"].get(p["path"]) == p["sha256"] for p in ledger_pins), "historical-ledger-freeze-mismatch")
    for path, digest in freeze["files"].items():
        require(sha(Path(path).read_bytes()) == digest, "frozen-file-pin-mismatch")
    for rel, digest in request["sourceFiles"].items():
        require(sha((REPO / rel).read_bytes()) == digest, "source-file-pin-mismatch")
        require(sha(git("show", request["sourceCommit"] + ":" + rel)) == digest, "source-git-pin-mismatch")
    require(sha(Path(EXE).read_bytes()) == EXE_SHA, "binary-pin-mismatch")
    validate_cwd(profile)
    return profile, {"requestSha256": sha((ROOT / "request.json").read_bytes()), "freezeSha256": sha(freeze_bytes), "reservationSha256": sha(reservation_path.read_bytes()), "sourceCommit": request["sourceCommit"], "freezeCommit": head}


def exclusive_json(path, value):
    with Path(path).open("x", encoding="utf-8") as f:
        json.dump(value, f, indent=2); f.write("\n"); f.flush(); os.fsync(f.fileno())


def session():
    # Controller must assign this worker to the kill-on-close Job before GO.
    require(sys.stdin.buffer.readline() == b"GO\n", "missing-job-gate")
    profile, binding = load_request()
    out = Path(profile["externalEvidence"])
    job_receipt = strict_json((out / "job-assigned.json").read_bytes())
    require(job_receipt["workerPid"] == os.getpid() and job_receipt["controllerPid"] == os.getppid(), "job-parent-or-worker-mismatch")
    require(all(job_receipt.get(k) == v for k, v in binding.items()), "job-binding-mismatch")
    prior = load_module(PRIOR, PRIOR_SHA, "preserved_metadata_client")
    enums = prior.load_method_enums(ROOT / "frozen-method-enums.json")
    prior.validate_disabled_status_gate(enums)
    started = time.monotonic(); deadline = started + 33
    exclusive_json(out / "started-once.json", {"key": KEY, "startedAtUnix": time.time(), **binding})
    messages = queue.Queue(maxsize=32); limit = threading.Event(); lock = threading.Lock()
    counts = {"raw": 0, "stdout": 0, "stderr": 0}; in_flight = 0
    observations = []; sent = []; events = []
    server = None; threads = []; reason = None; status = "stopped"; native = None; notification_count = 0

    def pump(stream, kind):
        nonlocal in_flight
        pending = b""
        while True:
            # Reserve read capacity before blocking: concurrent pipes together
            # can never ingest more than 4M bytes, including unfinished frames.
            with lock:
                allowance = min(8192, 4000000 - counts["raw"] - in_flight)
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
                if len(line) + 1 > 2000000:
                    limit.set(); return
                try:
                    messages.put_nowait((kind, line + b"\n"))
                except queue.Full:
                    limit.set(); return
            if len(pending) > 2000000:
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
        nonlocal notification_count
        if kind == "stderr":
            # Every stderr payload is nonpositive: no warning/auth text interpreted or saved.
            require(not raw, "server-stderr-activity")
            return None
        require(raw and len(raw) <= 2000000, "stdout-closed-or-frame-limit")
        frame = strict_json(raw)
        if isinstance(frame, dict) and "method" in frame:
            classified = classify(frame, enums, notification_count)
            notification_count = classified["notificationCount"]
            if classified["event"]:
                events.append(classified["event"])
            require(classified["action"] == "discard-notification", classified["reason"] or "unexpected-server-method")
            return None
        require(expected is not None, "unsolicited-response")
        return decode_response(frame, expected)

    try:
        env = {k: v for k, v in os.environ.items() if k not in {"OPENAI_API_KEY", "CODEX_API_KEY"}}
        server = subprocess.Popen(profile["argv"], cwd=profile["cwd"], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, shell=False, env=env)
        for stream, kind in ((server.stdout, "stdout"), (server.stderr, "stderr")):
            t = threading.Thread(target=pump, args=(stream, kind), daemon=True); threads.append(t); t.start()
        for rpc in profile["rpc"]:
            require(not limit.is_set() and time.monotonic() < deadline, "rpc-resource-or-deadline")
            while not messages.empty():
                require(consume(*messages.get_nowait()) is None, "unsolicited-response")
            server.stdin.write((json.dumps(rpc) + "\n").encode()); server.stdin.flush(); sent.append({"method": rpc["method"], "id": rpc.get("id")})
            if "id" not in rpc:
                continue
            while True:
                require(not limit.is_set() and time.monotonic() < deadline, "rpc-resource-or-deadline")
                try:
                    kind, raw = messages.get(timeout=.05)
                except queue.Empty:
                    continue
                result = consume(kind, raw, rpc["id"])
                if result is not None:
                    observations.append({"id": rpc["id"], "sanitizedResult": result}); break
            if rpc["id"] == 1:
                preliminary, preliminary_reason = assess_config(result)
                if preliminary != "visible-metadata-precondition-satisfied":
                    status = preliminary
                    raise ValueError(preliminary_reason)
        status, reason = final_assess(observations, sent)
    except ValueError as exc:
        # Only locally chosen fixed reason strings reach evidence.
        reason = str(exc) if re.fullmatch(r"[a-z-]{1,100}", str(exc)) else "invalid-or-unexpected-response"
    except BaseException:
        reason = "local-client-exception"
    finally:
        cleanup = time.monotonic(); cleanup_deadline = min(cleanup + 4, started + 37)
        if server:
            try:
                server.stdin.close(); native = server.wait(timeout=max(.001, min(1, cleanup_deadline-time.monotonic())))
            except BaseException:
                reason = reason or "native-cleanup-required"; status = "stopped"
                # Only own direct child; outer Job owns all descendants.
                try:
                    server.kill(); native = server.wait(timeout=max(.001, min(1, cleanup_deadline-time.monotonic())))
                except BaseException:
                    reason = "native-cleanup-failed"
        for t in threads:
            t.join(timeout=max(0, cleanup_deadline-time.monotonic()))
        if any(t.is_alive() for t in threads):
            reason = reason or "drain-deadline"; status = "stopped"
        while not messages.empty():
            kind, raw = messages.get_nowait()
            if raw:
                try:
                    require(consume(kind, raw) is None, "unexpected-final-output")
                except BaseException:
                    reason = reason or "unexpected-final-output"; status = "stopped"
        if limit.is_set() or native != 0 or time.monotonic() >= started+38:
            reason = reason or "native-resource-exit-or-deadline"; status = "stopped"
        if reason and status == "visible-metadata-precondition-satisfied":
            status = "stopped"
        result = {"key": KEY, "status": status, "stopReason": reason, **binding, "sentRpc": sent, "observations": observations, "serverMethodEvents": events, "notificationCount": notification_count, "nativeReturnCode": native, "nativeSessionWallSeconds": time.monotonic()-started, "cleanupSeconds": time.monotonic()-cleanup, "rawBytesDiscarded": counts, "rawFramePayloadsPersisted": False, "rawStderrPersisted": False, "requestedActors": 0, "requestedTurns": 0, "requestedModelCalls": 0, "nativeInternalNetworkOrInference": None, "activePermissionProfile": None, "actualTools": None, "servingModel": None, "providerRequests": None, "automaticRetries": 0, "meaning": "Reported prospective config/actual requirements only; recognition/enforcement, active permissions, tools, serving and historical cause unproven"}
        exclusive_json(out / "sanitized-result.json", result)
    return 0 if status == "visible-metadata-precondition-satisfied" and not reason else 1


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
            if time.monotonic()-started >= 38:
                reason = "owned-process-deadline"; job.kill(); break
            time.sleep(.01)
        code = worker.wait(timeout=5)
    except BaseException:
        reason = "controller-exception"
        if worker and worker.poll() is None:
            try:
                if job:
                    job.kill()
                else:
                    worker.kill()  # Still gated; target cannot have launched.
                code = worker.wait(timeout=5)
            except BaseException:
                reason = "controller-cleanup-failed"
    finally:
        cleanup = time.monotonic()
        if job:
            job.close()
        elapsed = time.monotonic()-started
        receipt = {"key": KEY, **binding, "workerReturnCode": code, "stopReason": reason, "controllerSeconds": elapsed, "jobCloseSeconds": time.monotonic()-cleanup, "processTreeControl": "windows-job-kill-on-close", "deadlineSeconds": 38, "controllerLimitSeconds": 50, "outerLimitSeconds": 60, "rawLogsPersisted": False, "automaticRetries": 0}
        result_path = external / "sanitized-result.json"
        receipt["terminalResultPresent"] = result_path.is_file()
        receipt["terminalResultSha256"] = sha(result_path.read_bytes()) if result_path.is_file() else None
        receipt["withinControllerBound"] = elapsed <= 50
        exclusive_json(external / "controller-receipt.json", receipt)
    return 0 if code == 0 and reason is None and elapsed <= 50 and result_path.is_file() else 1


if __name__ == "__main__":
    try:
        code = session() if sys.argv[1:] == ["--session"] else controller() if sys.argv[1:] == ["--controller"] else 2
    except BaseException:
        # No traceback or raw exception may reach a persisted log.
        code = 2
    raise SystemExit(code)
