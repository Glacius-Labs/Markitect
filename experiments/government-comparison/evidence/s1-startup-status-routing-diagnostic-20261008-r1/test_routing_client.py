"""Six synthetic routing and binding cases for the copied diagnostic client."""
import copy
import hashlib
import io
import json
import os
from pathlib import Path
import queue
import tempfile
import types
import unittest
from datetime import datetime
from unittest.mock import patch

import client


HERE = Path(__file__).resolve().parent
REPO = HERE.parents[3]
PROFILE_SOURCE = json.loads((HERE / "profile.json").read_text(encoding="utf-8"))
CONTRACT = json.loads((HERE / "schema-contract.json").read_text(encoding="utf-8"))
STARTUP_MEMBER = "v2/McpServerStatusUpdatedNotification.json"
STARTUP_METHOD = "mcpServer/startupStatus/updated"
SYNTHETIC_CWD = "C:/Synthetic/StartupRoute"
SYNTHETIC_MODEL = "synthetic-model-route"
SYNTHETIC_PROVIDER = "synthetic-provider-route"


def make_profile(external_root):
    profile = copy.deepcopy(PROFILE_SOURCE)
    profile["cwd"] = SYNTHETIC_CWD
    profile["externalEvidence"] = str(external_root)
    profile["argv"] = ["synthetic-app-server", "--stdio"]
    for rpc in profile["rpc"]:
        if rpc.get("method") == "config/read":
            rpc["params"]["cwd"] = SYNTHETIC_CWD
    profile["threadStart"]["params"].update({
        "cwd": SYNTHETIC_CWD,
        "model": SYNTHETIC_MODEL,
        "modelProvider": SYNTHETIC_PROVIDER,
    })
    return profile


def thread_response(profile, **changes):
    params = profile["threadStart"]["params"]
    result = {
        "approvalPolicy": params["approvalPolicy"],
        "approvalsReviewer": "user",
        "cwd": params["cwd"],
        "model": params["model"],
        "modelProvider": params["modelProvider"],
        "sandbox": {"type": "readOnly", "networkAccess": False},
        "activePermissionProfile": {"id": ":read-only", "extends": None},
        "thread": {
            "cliVersion": "synthetic-test-client",
            "createdAt": 1,
            "cwd": params["cwd"],
            "ephemeral": True,
            "id": "synthetic-thread-routing-01",
            "model": params["model"],
            "modelProvider": params["modelProvider"],
            "preview": "",
            "projectId": None,
            "sessionId": "synthetic-session-routing-01",
            "source": "appServer",
            "status": {"type": "active", "activeFlags": []},
            "turns": [],
            "updatedAt": 1,
            "parentThreadId": None,
            "forkedFromId": None,
        },
    }
    result.update(changes)
    return result


class FakeStream:
    def __init__(self, blocking=False):
        self.pending = queue.Queue() if blocking else None

    def read1(self, _limit):
        if self.pending is None:
            return b""
        try:
            return self.pending.get(timeout=2)
        except queue.Empty:
            return b""

    def push(self, raw):
        self.pending.put(raw)


class FakeStdin:
    def __init__(self, stdout, profile, mode, startup_frames=(), thread_changes=None):
        self.stdout = stdout
        self.profile = profile
        self.mode = mode
        self.startup_frames = list(startup_frames)
        self.thread_changes = thread_changes or {}

    def _notification(self, params):
        return {"method": STARTUP_METHOD, "params": params}

    def write(self, raw):
        request = json.loads(raw)
        method = request["method"]
        if method == "initialize" and self.mode == "metadata-refusal":
            self.stdout.push(json.dumps(self._notification(self.startup_frames[0])).encode() + b"\n")
        if method == "thread/start":
            if self.mode == "other-method":
                frame = {"method": "synthetic/unlisted/notification", "params": {}}
                self.stdout.push(json.dumps(frame).encode() + b"\n")
            elif self.mode == "negative-before-response":
                for params in self.startup_frames:
                    self.stdout.push(json.dumps(self._notification(params)).encode() + b"\n")
            result = thread_response(self.profile, **self.thread_changes)
            frame = {"jsonrpc": "2.0", "id": request["id"], "result": result}
            self.stdout.push(json.dumps(frame).encode() + b"\n")
            if self.mode in ("positive-cleanup", "first-failure-drain"):
                for params in self.startup_frames:
                    self.stdout.push(json.dumps(self._notification(params)).encode() + b"\n")
            return len(raw)
        if "id" in request:
            result = {}
            frame = {"jsonrpc": "2.0", "id": request["id"], "result": result}
            self.stdout.push(json.dumps(frame).encode() + b"\n")
        return len(raw)

    def flush(self):
        return None

    def close(self):
        if self.mode in ("positive-cleanup", "first-failure-drain"):
            self.stdout.push(json.dumps({"method":"thread/started","params":{"thread":thread_response(self.profile)["thread"]}}).encode()+b"\n")
            self.stdout.push(json.dumps({"method":"thread/closed","params":{"threadId":"synthetic-thread-routing-01"}}).encode()+b"\n")
        self.stdout.push(b"")


class FakeProcess:
    pid = 424242

    def __init__(self, profile, mode, startup_frames=(), thread_changes=None):
        self.stdout = FakeStream(blocking=True)
        self.stdin = FakeStdin(self.stdout, profile, mode, startup_frames, thread_changes)
        self.stderr = FakeStream()

    def wait(self, timeout=None):
        return 0

    def poll(self):
        return 0

    def kill(self):
        return None


def startup_params(status="ready", thread_id=None, **extra):
    params = {"name": "synthetic-startup-service", "status": status, "threadId": thread_id}
    params.update(extra)
    return params


def run_synthetic_session(mode="positive-cleanup", startup_frames=(), thread_changes=None):
    captured = {}
    with tempfile.TemporaryDirectory(prefix="startup-routing-synthetic-") as temp:
        external = Path(temp)
        profile = make_profile(external)
        assigned = "2026-10-08T19:32:00Z"
        binding = {
            "key": client.KEY,
            "grantIssuedUtc": client.ISSUED,
            "slotAssignedUtc": assigned,
            "sourceCommit": "a" * 40,
            "freezeSha256": "b" * 64,
        }
        process = FakeProcess(profile, mode, startup_frames, thread_changes)
        source_files = {}
        for name in ("thread_gate.py", "protocol_failure.py", "startup_observer.py"):
            path = HERE / name
            source_files[path.relative_to(REPO).as_posix()] = hashlib.sha256(path.read_bytes()).hexdigest()
        job_record = {"workerPid": os.getpid(), "controllerPid": os.getppid(), **binding}
        real_read_bytes = Path.read_bytes
        real_load_module = client.load_module

        class SelectiveProtocol:
            def __init__(self, archive, contract, protocol_type):
                self.actual = protocol_type(archive, contract)
                self.contract = contract

            def validate(self, member, value):
                # The routing and unchanged thread boundary use the real pinned
                # schemas. Metadata reply fixtures stay deliberately minimal.
                if member in (STARTUP_MEMBER, self.contract["responseSchemas"]["thread/start"]):
                    return self.actual.validate(member, value)

        def selected_load_module(path, expected, name):
            if Path(path) == Path(client.PROTOCOL):
                module = real_load_module(path, expected, name)
                return types.SimpleNamespace(
                    Protocol=lambda archive, contract: SelectiveProtocol(archive, contract, module.Protocol)
                )
            return real_load_module(path, expected, name)

        def safe_read_bytes(path):
            resolved = path.resolve()
            if resolved == (HERE / "request.json").resolve():
                return json.dumps({"sourceFiles": source_files}).encode()
            if resolved == (external / "job-assigned.json").resolve():
                return json.dumps(job_record).encode()
            return real_read_bytes(path)

        with (
            patch.object(client, "load_request", return_value=(profile, binding)),
            patch.object(client, "live_authority", return_value=({}, assigned)),
            patch.object(client, "sanitize", return_value={}),
            patch.object(client, "assess_config", return_value=("config-precheck-satisfied-awaiting-requirements", None)),
            patch.object(client, "final_assess", return_value=("reported-config-and-requirements-observed", None)),
            patch.object(client, "validate_cwd", return_value={}),
            patch.object(client, "exclusive_json", side_effect=lambda path, value: captured.update({Path(path).name: copy.deepcopy(value)})),
            patch.object(client, "load_module", side_effect=selected_load_module),
            patch.object(client.subprocess, "Popen", return_value=process) as popen,
            patch.object(Path, "read_bytes", safe_read_bytes),
            patch.object(client.sys, "stdin", types.SimpleNamespace(buffer=io.BytesIO(b"GO\n"))),
        ):
            exit_code = client.session()
        return exit_code, captured["sanitized-result.json"], popen.call_count


class StartupRoutingClientTests(unittest.TestCase):
    def setUp(self):
        self.no_real_process = patch.object(
            client.subprocess, "Popen", side_effect=AssertionError("unexpected-real-process")
        )
        self.no_real_run = patch.object(
            client.subprocess, "run", side_effect=AssertionError("unexpected-subprocess-run")
        )
        self.no_check_output = patch.object(
            client.subprocess, "check_output", side_effect=AssertionError("unexpected-subprocess-check-output")
        )
        self.no_system = patch.object(client.os, "system", side_effect=AssertionError("unexpected-os-system"))
        for guard in (self.no_real_process, self.no_real_run, self.no_check_output, self.no_system):
            guard.start()
            self.addCleanup(guard.stop)

    def test_app_and_correlated_own_startup_are_observed_after_valid_thread_response(self):
        frames = [startup_params(status="starting", thread_id=None, error=None, failureReason=None), startup_params(thread_id="synthetic-thread-routing-01")]
        exit_code, receipt, popen_count = run_synthetic_session("positive-cleanup", frames)
        observation = receipt["startupStatusObservation"]
        self.assertEqual(exit_code, 0)
        self.assertEqual(popen_count, 1)
        self.assertEqual(receipt["status"], "thread-response-observed")
        self.assertEqual(observation["observed"], 2)
        self.assertEqual(observation["statusCounts"]["ready"], 1)
        self.assertEqual(observation["statusCounts"]["starting"], 1)
        self.assertTrue(receipt["threadBoundary"]["threadStartedNotificationCorrelated"])
        self.assertTrue(receipt["threadBoundary"]["threadClosedNotificationCorrelated"])
        self.assertEqual(observation["scopeCounts"]["app"], 1)
        self.assertEqual(observation["scopeCounts"]["own-thread"], 1)
        self.assertEqual(observation["dispositionCounts"]["observation-only"], 2)
        self.assertEqual(observation["routingRuleVersion"], 1)
        self.assertIs(observation["telemetryOnly"], True)
        self.assertNotIn("methodAdmitted", observation)
        self.assertNotIn("mode", observation)
        self.assertEqual(receipt["threadBoundary"]["responseValidated"], True)
        self.assertEqual(receipt["threadBoundary"]["turnsRequested"], 0)
        self.assertEqual(receipt["threadBoundary"]["toolsRequested"], 0)

    def test_negative_startup_classifications_remain_terminal(self):
        cases = [
            (startup_params(status="outside-schema", thread_id=None), "schema-invalid"),
            (startup_params(status="failed", thread_id=None), "terminal-status"),
            (startup_params(thread_id="synthetic-foreign-thread"), "foreign-thread"),
            ({"name": "synthetic-startup-service", "status": "ready"}, "missing-thread-id"),
            (startup_params(thread_id=None, error="synthetic failure"), "error-present"),
            (startup_params(status="cancelled",thread_id=None), "terminal-status"),
            (startup_params(thread_id=None,failureReason="reauthenticationRequired"), "failure-reason-present"),
            (startup_params(thread_id=None,extraSynthetic=True), "local-shape"),
            ({"name":"s"*257,"status":"ready","threadId":None}, "local-size"),
            (startup_params(thread_id="synthetic-unresolved-thread"), "unresolved-thread"),
        ]
        for params, refusal in cases:
            with self.subTest(refusal=refusal):
                mode = "positive-cleanup" if refusal == "foreign-thread" else "negative-before-response"
                exit_code, receipt, _ = run_synthetic_session(mode, [params])
                self.assertEqual(exit_code, 1)
                self.assertEqual(receipt["status"], "stopped")
                self.assertEqual(receipt["firstProtocolFailure"]["refusalCode"], "forbidden-or-unknown-notification")
                self.assertEqual(receipt["startupStatusObservation"]["firstClassification"]["refusalCode"], refusal)
                self.assertEqual(receipt["startupStatusObservation"]["observed"], 1)
                self.assertEqual(receipt["turnStartRequests"], 0)
                self.assertEqual(receipt["taskToolRequests"], 0)

    def test_metadata_phase_startup_observation_keeps_original_lifecycle_refusal(self):
        params = startup_params(thread_id=None)
        exit_code, receipt, _ = run_synthetic_session("metadata-refusal", [params])
        self.assertEqual(exit_code, 1)
        self.assertEqual(receipt["sentRpc"], [{"method": "initialize", "id": 0, "writeCompleted": True}])
        self.assertEqual(receipt["stopReason"], "lifecycle-before-thread-request")
        failure = receipt["firstProtocolFailure"]
        self.assertEqual(failure["stage"], "thread-lifecycle")
        self.assertEqual(failure["refusalCode"], "lifecycle-before-thread-request")
        self.assertEqual(receipt["startupStatusObservation"]["observed"], 0)
        self.assertEqual(receipt["threadBoundary"]["responseValidated"], False)

    def test_unlisted_method_and_thread_response_gate_remain_strict(self):
        exit_code, receipt, _ = run_synthetic_session("other-method")
        self.assertEqual(exit_code, 1)
        self.assertEqual(receipt["firstProtocolFailure"]["refusalCode"], "forbidden-or-unknown-notification")
        self.assertEqual(receipt["startupStatusObservation"]["observed"], 0)
        self.assertEqual(receipt["threadBoundary"]["turnsRequested"], 0)
        self.assertEqual(receipt["threadBoundary"]["toolsRequested"], 0)

        exit_code, receipt, _ = run_synthetic_session(
            "positive-cleanup", [], {"model": "synthetic-model-mismatch"}
        )
        self.assertEqual(exit_code, 1)
        self.assertEqual(receipt["stopReason"], "thread-response-identity-or-policy-mismatch")
        self.assertEqual(receipt["threadBoundary"]["responseValidated"], False)
        self.assertEqual(receipt["threadBoundary"]["threadId"], None)
        self.assertEqual(receipt["threadResponseFieldComparison"]["comparisons"][1]["equal"], False)
        self.assertEqual(receipt["turnStartRequests"], 0)
        self.assertEqual(receipt["taskToolRequests"], 0)

    def test_first_startup_failure_survives_later_cleanup_drain(self):
        frames = [
            startup_params(thread_id="synthetic-foreign-thread"),
            startup_params(status="failed", thread_id=None),
        ]
        exit_code, receipt, _ = run_synthetic_session("first-failure-drain", frames)
        self.assertEqual(exit_code, 1)
        self.assertEqual(receipt["startupStatusObservation"]["observed"], 2)
        self.assertEqual(receipt["startupStatusObservation"]["scopeCounts"]["foreign-thread"], 1)
        self.assertEqual(receipt["startupStatusObservation"]["statusCounts"]["failed"], 1)
        self.assertEqual(receipt["firstProtocolFailure"]["refusalCode"], "forbidden-or-unknown-notification")
        self.assertEqual(receipt["startupStatusObservation"]["firstClassification"]["refusalCode"], "foreign-thread")
        self.assertEqual(receipt["startupStatusObservation"]["dispositionCounts"]["terminal-reject"], 2)
        self.assertTrue(receipt["threadBoundary"]["threadClosedNotificationCorrelated"])
        self.assertEqual(receipt["firstProtocolFailure"]["stage"], "notification-gate")
        self.assertEqual(receipt["turnStartRequests"], 0)
        self.assertEqual(receipt["taskToolRequests"], 0)

    def test_fresh_attempt_two_window_and_request_freeze_binding(self):
        self.assertEqual(hashlib.sha256((HERE/"protocol_failure.py").read_bytes()).hexdigest(),client.PROTOCOL_FAILURE_SHA)
        self.assertEqual(hashlib.sha256((HERE/"startup_observer.py").read_bytes()).hexdigest(),client.STARTUP_OBSERVER_SHA)
        with self.assertRaisesRegex(ValueError,"source-pin-mismatch"):
            client.load_module(HERE/"startup_observer.py","0"*64,"synthetic_rejected_import")
        assigned = "2026-10-08T19:32:00Z"
        grant = {
            "key": client.KEY,
            "issuedUtc": client.ISSUED,
            "status": client.GRANT_STATUS,
            "readinessWindowKey": client.READINESS_KEY,
            "notAfterUtc": client.READINESS_DEADLINE,
            "latestActualStartUtc": client.LATEST_ACTUAL_START,
            "readinessAttemptOrdinal": 2,
        }
        slot = {
            "owner": "Scientist",
            "key": client.KEY,
            "status": client.SLOT_STATUS,
            "grantIssuedUtc": client.ISSUED,
            "assignedUtc": assigned,
        }
        self.assertEqual(client.validate_live_authority(copy.deepcopy(grant), slot, copy.deepcopy(grant)), assigned)
        window = {
            "key": client.READINESS_KEY,
            "authorizedUtc": "2026-10-08T19:30:00Z",
            "deadlineUtc": client.READINESS_DEADLINE,
            "maxFurtherActualReadinessAttempts": 2,
            "consumedFurtherActualAttempts": 1,
            "reservedFurtherActualAttempts": 1,
            "priorActualAppServerTrees": 11,
            "grantKeys": ["s1-final-output-diagnostic-20261008-r1", client.KEY],
            "eachAttemptRequiresFreshConcreteGrant": True,
            "zeroRuntimeAuthorizedByThisWindow": True,
            "noPolicyBypassNoPurchases": True,
            "finalPackageEndsDiagnosisBlock": True,
        }
        client.validate_readiness_window(
            grant, window, datetime.fromisoformat(client.LATEST_ACTUAL_START.replace("Z", "+00:00"))
        )
        request = {
            "key": client.KEY,
            "grantIssuedUtc": client.ISSUED,
            "readinessWindowKey": client.READINESS_KEY,
            "notAfterUtc": client.READINESS_DEADLINE,
            "latestActualStartUtc": client.LATEST_ACTUAL_START,
            "readinessAttemptOrdinal": 2,
            "sourceCommit": "a" * 40,
            "slotAssignedUtc": assigned,
        }
        freeze = {**request, "frozenAtUtc": "2026-10-08T19:32:01Z"}
        client.validate_binding_identity(request, freeze, assigned)
        wrong = copy.deepcopy(freeze)
        wrong["readinessAttemptOrdinal"] = True
        with self.assertRaisesRegex(ValueError, "readiness-freeze-binding-mismatch"):
            client.validate_binding_identity(request, wrong, assigned)
        inactive_slot = {**slot, "status": "synthetic-inactive-slot"}
        with self.assertRaisesRegex(ValueError, "inactive-live-slot"):
            client.validate_live_authority(grant, inactive_slot, grant)


if __name__ == "__main__":
    unittest.main()
