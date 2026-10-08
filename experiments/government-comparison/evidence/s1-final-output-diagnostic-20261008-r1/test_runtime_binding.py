"""Four offline synthetic tests for readiness-window and runtime binding."""
import copy
import hashlib
import importlib.util
import io
import json
from datetime import timedelta
import os
from pathlib import Path
import queue
import tempfile
import threading
import types
import unittest
from unittest.mock import patch

import client


HERE = Path(__file__).resolve().parent
REPO = HERE.parents[3]
PROFILE_SOURCE = json.loads((HERE / "profile.json").read_text(encoding="utf-8"))
AUTHORITY = json.loads((HERE / "authorization-grant.json").read_text(encoding="utf-8"))
CONTRACT = json.loads((HERE / "schema-contract.json").read_text(encoding="utf-8"))
PROTOCOL_MODULE = client.load_module(client.PROTOCOL, client.PROTOCOL_SHA, "runtime_binding_protocol")
REAL_PROTOCOL = PROTOCOL_MODULE.Protocol(client.SCHEMA, CONTRACT)
GATE_MODULE = client.load_module(
    HERE / "thread_gate.py", hashlib.sha256((HERE / "thread_gate.py").read_bytes()).hexdigest(),
    "runtime_binding_gate",
)
HELPER_PATH = HERE / "protocol_failure.py"

SYNTHETIC = {
    "cwd": r"C:\Synthetic\RuntimeBindingRoot",
    "model": "synthetic-model-alpha",
    "modelProvider": "synthetic-provider-alpha",
    "approvalPolicy": "never",
}


def synthetic_response(profile):
    params = profile["threadStart"]["params"]
    return {
        "approvalPolicy": params["approvalPolicy"],
        "approvalsReviewer": "user",
        "cwd": params["cwd"],
        "model": params["model"],
        "modelProvider": params["modelProvider"],
        "sandbox": {"type": "readOnly", "networkAccess": False},
        "activePermissionProfile": {"id": ":read-only", "extends": None},
        "thread": {
            "cliVersion": "offline-synthetic",
            "createdAt": 1,
            "cwd": params["cwd"],
            "ephemeral": True,
            "id": "synthetic-runtime-thread",
            "model": params["model"],
            "modelProvider": params["modelProvider"],
            "preview": "",
            "projectId": None,
            "sessionId": "synthetic-runtime-session",
            "source": "appServer",
            "status": {"type": "active", "activeFlags": []},
            "turns": [],
            "updatedAt": 1,
            "parentThreadId": None,
            "forkedFromId": None,
        },
    }


class QueueStream:
    def __init__(self, blocking=False, chunks=()):
        self.pending = queue.Queue() if blocking else None
        self.chunks = iter(chunks)

    def read1(self, _limit):
        if self.pending is not None:
            try:
                return self.pending.get(timeout=2)
            except queue.Empty:
                return b""
        return next(self.chunks, b"")

    def push(self, raw):
        self.pending.put(raw)


class QueueStdin:
    def __init__(self, stdout, replies, after_close):
        self.stdout = stdout
        self.replies = replies
        self.after_close = after_close

    def write(self, raw):
        request = json.loads(raw)
        if "id" in request:
            reply = {"jsonrpc": "2.0", "id": request["id"], "result": self.replies[request["id"]]}
            self.stdout.push(json.dumps(reply).encode() + b"\n")

    def flush(self):
        return None

    def close(self):
        for raw in self.after_close:
            self.stdout.push(raw if raw.endswith(b"\n") else raw + b"\n")
        self.stdout.push(b"")


class FakeProcess:
    pid = 636363

    def __init__(self, replies, after_close):
        self.stdout = QueueStream(blocking=True)
        self.stderr = QueueStream(chunks=(b"",))
        self.stdin = QueueStdin(self.stdout, replies, after_close)

    def wait(self, timeout=None):
        return 0

    def poll(self):
        return 0

    def kill(self):
        return None


def run_mock_session():
    captured = {}
    with tempfile.TemporaryDirectory(prefix="runtime-binding-synthetic-") as temp:
        profile = copy.deepcopy(PROFILE_SOURCE)
        profile["cwd"] = SYNTHETIC["cwd"]
        profile["externalEvidence"] = temp
        profile["argv"] = ["synthetic-app-server", "--stdio"]
        profile["threadStart"]["params"].update(SYNTHETIC)
        for rpc in profile["rpc"]:
            if rpc.get("method") == "config/read":
                rpc["params"]["cwd"] = SYNTHETIC["cwd"]

        binding = {
            "key": client.KEY, "grantIssuedUtc": client.ISSUED,
            "slotAssignedUtc": client.ISSUED, "sourceCommit": "c" * 40,
            "freezeSha256": "d" * 64,
            "readinessWindowKey": client.READINESS_KEY,
            "notAfterUtc": client.READINESS_DEADLINE,
            "latestActualStartUtc": client.LATEST_ACTUAL_START,
            "readinessAttemptOrdinal": 1,
        }
        replies = {}
        for rpc in profile["rpc"] + [profile["threadStart"]]:
            if "id" not in rpc:
                continue
            replies[rpc["id"]] = synthetic_response(profile) if rpc["method"] == "thread/start" else {}
        after_close = [json.dumps({
            "method": "error",
            "params": {"error": {"message": "synthetic-final-cleanup-error"},
                       "threadId": "synthetic-runtime-thread", "turnId": "synthetic-runtime-turn",
                       "willRetry": False},
        }).encode()]
        process = FakeProcess(replies, after_close)
        grant = copy.deepcopy(AUTHORITY["grant"])
        assigned = client.ISSUED
        request_pins = {}
        for name in ("thread_gate.py", "protocol_failure.py"):
            path = HERE / name
            request_pins[path.relative_to(REPO).as_posix()] = hashlib.sha256(path.read_bytes()).hexdigest()
        job = {"workerPid": os.getpid(), "controllerPid": os.getppid(), **binding}
        old_read_bytes = Path.read_bytes

        def synthetic_read_bytes(path):
            resolved = path.resolve()
            if resolved == (Path(temp) / "job-assigned.json").resolve():
                return json.dumps(job).encode()
            if resolved == (HERE / "request.json").resolve():
                return json.dumps({"sourceFiles": request_pins}).encode()
            return old_read_bytes(path)

        old_load_module = client.load_module

        class SelectiveProtocol:
            def __init__(self, _archive, contract):
                self.contract = contract

            def validate(self, member, value):
                if member == self.contract["responseSchemas"]["thread/start"]:
                    REAL_PROTOCOL.validate(member, value)

        protocol_module = types.SimpleNamespace(Protocol=SelectiveProtocol)

        def load_module(path, expected, name):
            if Path(path) == Path(client.PROTOCOL):
                return protocol_module
            return old_load_module(path, expected, name)

        config_status = "config-precheck-satisfied-awaiting-requirements"
        final_status = "reported-config-and-requirements-observed"
        with (
            patch.object(client, "load_request", return_value=(profile, binding)),
            patch.object(client, "load_module", side_effect=load_module),
            patch.object(client, "live_authority", return_value=(grant, assigned)) as live_check,
            patch.object(client, "sanitize", return_value={}),
            patch.object(client, "assess_config", return_value=(config_status, None)),
            patch.object(client, "final_assess", return_value=(final_status, None)),
            patch.object(client, "validate_cwd", return_value={}),
            patch.object(client, "exclusive_json", side_effect=lambda path, value: captured.update({Path(path).name: copy.deepcopy(value)})),
            patch.object(client.subprocess, "Popen", return_value=process) as popen,
            patch.object(Path, "read_bytes", synthetic_read_bytes),
            patch.object(client.sys, "stdin", types.SimpleNamespace(buffer=io.BytesIO(b"GO\n"))),
        ):
            code = client.session()
        return code, captured, popen.call_count, live_check.call_count


class RuntimeBindingTests(unittest.TestCase):
    def setUp(self):
        self.no_process = patch.object(
            client.subprocess, "Popen", side_effect=AssertionError("native-process-forbidden")
        )
        self.no_process.start()
        self.addCleanup(self.no_process.stop)

    def test_helper_pin_precedes_import_and_readiness_window_boundaries_are_exact(self):
        helper_hash = client.sha(HELPER_PATH.read_bytes())
        self.assertEqual(helper_hash, client.PROTOCOL_FAILURE_SHA)
        with patch.object(client.importlib.util, "spec_from_file_location", side_effect=AssertionError("import-after-bad-pin")) as spec:
            with self.assertRaisesRegex(ValueError, "source-pin-mismatch"):
                client.load_module(HELPER_PATH, "0" * 64, "bad_protocol_failure_pin")
            spec.assert_not_called()

        grant = AUTHORITY["grant"]
        window = AUTHORITY["readinessWindowBinding"]
        latest = client.utc(client.LATEST_ACTUAL_START)
        client.validate_readiness_window(grant, window, now=latest)
        self.assertEqual(client.readiness_binding(window), window)
        with self.assertRaisesRegex(ValueError, "readiness-window-expired"):
            client.validate_readiness_window(grant, window, now=latest + timedelta(seconds=1))
        deadline = client.utc(client.READINESS_DEADLINE)
        with self.assertRaisesRegex(ValueError, "readiness-window-expired"):
            client.validate_readiness_window(grant, window, now=deadline)

    def test_authority_drift_is_rejected_but_unrelated_live_cursor_is_ignored(self):
        authority = copy.deepcopy(AUTHORITY)
        grant = copy.deepcopy(authority["grant"])
        window = copy.deepcopy(authority["readinessWindowBinding"])
        slot = {
            "owner": "Scientist", "key": client.KEY, "status": client.SLOT_STATUS,
            "grantIssuedUtc": client.ISSUED, "assignedUtc": client.ISSUED,
        }
        live = {
            "threads": [{"name": "Scientist", "evidence": {"finalOutputDiagnosticGrant": grant}}],
            "fullSuiteSlot": slot,
            "researchMandate": {"finalReadinessWindow": window},
            "cursor": "unrelated synthetic cursor A",
        }
        target = Path(client.COORDINATION).resolve()
        actual_read_bytes = Path.read_bytes

        def read_bytes(path):
            if path.resolve() == target:
                return json.dumps(live).encode()
            return actual_read_bytes(path)

        with patch.object(Path, "read_bytes", read_bytes):
            live["cursor"] = "unrelated synthetic cursor B"
            accepted, assigned = client.live_authority(authority)
            self.assertEqual(accepted, grant)
            self.assertEqual(assigned, client.ISSUED)

            live["researchMandate"]["finalReadinessWindow"]["consumedFurtherActualAttempts"] = 1
            with self.assertRaisesRegex(ValueError, "readiness-window-allocation-mismatch"):
                client.live_authority(authority)

            live["researchMandate"]["finalReadinessWindow"] = window
            live["threads"][0]["evidence"]["finalOutputDiagnosticGrant"] = {**grant, "readinessAttemptOrdinal": True}
            with self.assertRaisesRegex(ValueError, "live-grant-mismatch"):
                client.live_authority(authority)

    def test_request_and_freeze_bind_all_four_readiness_fields(self):
        assigned = client.ISSUED
        request = {
            "key": client.KEY, "grantIssuedUtc": client.ISSUED,
            "sourceCommit": "e" * 40, "slotAssignedUtc": assigned,
            "readinessWindowKey": client.READINESS_KEY,
            "notAfterUtc": client.READINESS_DEADLINE,
            "latestActualStartUtc": client.LATEST_ACTUAL_START,
            "readinessAttemptOrdinal": 1,
        }
        freeze = {**request, "frozenAtUtc": assigned}
        client.validate_binding_identity(request, freeze, assigned)
        stale_values = (
            {"readinessWindowKey": "synthetic-stale-window"},
            {"notAfterUtc": "2026-10-08T20:42:01Z"},
            {"latestActualStartUtc": "2026-10-08T20:40:41Z"},
            {"readinessAttemptOrdinal": True},
        )
        for mutation in stale_values:
            with self.subTest(field=next(iter(mutation))):
                with self.assertRaisesRegex(ValueError, "readiness-freeze-binding-mismatch"):
                    client.validate_binding_identity(request, {**freeze, **mutation}, assigned)

    def test_mocked_session_finally_keeps_first_failure_boundary_and_field_comparison(self):
        code, captured, popen_calls, live_calls = run_mock_session()
        receipt = captured["sanitized-result.json"]
        self.assertEqual(code, 1)
        self.assertEqual((popen_calls, live_calls), (1, 1))
        self.assertEqual(receipt["firstProtocolFailure"]["phase"], "cleanup")
        self.assertEqual(receipt["firstProtocolFailure"]["stage"], "notification-gate")
        self.assertEqual(receipt["firstProtocolFailure"]["method"], {"category": "listed", "name": "error"})
        self.assertTrue(receipt["threadBoundary"]["responseValidated"])
        self.assertEqual(receipt["threadBoundary"]["turnsRequested"], 0)
        self.assertEqual(receipt["threadBoundary"]["toolsRequested"], 0)
        self.assertEqual(
            [row["equal"] for row in receipt["threadResponseFieldComparison"]["comparisons"]],
            [True, True, True, True],
        )
        self.assertNotIn("synthetic-final-cleanup-error", json.dumps(receipt["firstProtocolFailure"]))


if __name__ == "__main__":
    unittest.main(verbosity=2)
