"""Six offline synthetic cases for bounded first-protocol-failure diagnostics."""
import copy
import hashlib
import importlib.util
import io
import json
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
CONTRACT = json.loads((HERE / "schema-contract.json").read_text(encoding="utf-8"))
PROFILE_SOURCE = json.loads((HERE / "profile.json").read_text(encoding="utf-8"))
FAILURE_SPEC = importlib.util.spec_from_file_location(
    "final_output_test_failure", HERE / "protocol_failure.py"
)
FAILURE_MODULE = importlib.util.module_from_spec(FAILURE_SPEC)
FAILURE_SPEC.loader.exec_module(FAILURE_MODULE)
REAL_PROTOCOL_MODULE = client.load_module(
    client.PROTOCOL, client.PROTOCOL_SHA, "final_output_test_real_protocol"
)
REAL_PROTOCOL = REAL_PROTOCOL_MODULE.Protocol(client.SCHEMA, CONTRACT)

SYNTHETIC = {
    "cwd": r"C:\Synthetic\FinalOutputRoot",
    "model": "synthetic-model-alpha",
    "modelProvider": "synthetic-provider-alpha",
    "approvalPolicy": "never",
}


def thread_response(profile, **overrides):
    params = profile["threadStart"]["params"]
    value = {
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
            "id": "synthetic-final-thread",
            "model": params["model"],
            "modelProvider": params["modelProvider"],
            "preview": "",
            "projectId": None,
            "sessionId": "synthetic-final-session",
            "source": "appServer",
            "status": {"type": "active", "activeFlags": []},
            "turns": [],
            "updatedAt": 1,
            "parentThreadId": None,
            "forkedFromId": None,
        },
    }
    value.update(overrides)
    return value


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
            result = self.replies[request["id"]]
            self.stdout.push(json.dumps({"jsonrpc": "2.0", "id": request["id"], "result": result}).encode() + b"\n")

    def flush(self):
        return None

    def close(self):
        for raw in self.after_close:
            self.stdout.push(raw if raw.endswith(b"\n") else raw + b"\n")
        self.stdout.push(b"")


class FakeProcess:
    pid = 525252

    def __init__(self, replies, after_close=()):
        self.stdout = QueueStream(blocking=True)
        self.stderr = QueueStream(chunks=(b"",))
        self.stdin = QueueStdin(self.stdout, replies, after_close)

    def wait(self, timeout=None):
        return 0

    def poll(self):
        return 0

    def kill(self):
        return None


def run_mock_session(after_close=(), invalid_thread=False):
    """Run the real nested consume/finally flow over an in-process fake stream."""
    captured = {}
    with tempfile.TemporaryDirectory(prefix="protocol-failure-synthetic-") as temp:
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
            "slotAssignedUtc": client.ISSUED, "sourceCommit": "a" * 40,
            "freezeSha256": "b" * 64,
        }
        replies = {}
        for rpc in profile["rpc"] + [profile["threadStart"]]:
            if "id" not in rpc:
                continue
            payload = thread_response(profile) if rpc["method"] == "thread/start" else {}
            if invalid_thread and rpc["method"] == "thread/start":
                del payload["sandbox"]
            replies[rpc["id"]] = payload
        injected = list(after_close)
        fake_process = FakeProcess(replies, injected)

        class SelectiveProtocol:
            def __init__(self, _archive, contract):
                self.contract = contract

            def validate(self, member, value):
                if member == self.contract["responseSchemas"]["thread/start"]:
                    REAL_PROTOCOL.validate(member, value)
                # Other synthetic metadata frames are shaped only for the
                # session lifecycle; the pinned full thread response is real.

        protocol_module = types.SimpleNamespace(Protocol=SelectiveProtocol)
        source_files = {}
        for name in ("thread_gate.py", "protocol_failure.py"):
            path = HERE / name
            source_files[path.relative_to(REPO).as_posix()] = hashlib.sha256(path.read_bytes()).hexdigest()
        job = {"workerPid": os.getpid(), "controllerPid": os.getppid(), **binding}
        actual_read_bytes = Path.read_bytes

        def read_bytes(path):
            resolved = path.resolve()
            if resolved == (Path(temp) / "job-assigned.json").resolve():
                return json.dumps(job).encode()
            if resolved == (HERE / "request.json").resolve():
                return json.dumps({"sourceFiles": source_files}).encode()
            return actual_read_bytes(path)

        actual_load_module = client.load_module

        def load_module(path, expected, name):
            if Path(path) == Path(client.PROTOCOL):
                return protocol_module
            return actual_load_module(path, expected, name)

        config_status = "config-precheck-satisfied-awaiting-requirements"
        final_status = "reported-config-and-requirements-observed"
        with (
            patch.object(client, "load_request", return_value=(profile, binding)),
            patch.object(client, "load_module", side_effect=load_module),
            patch.object(client, "sanitize", return_value={}),
            patch.object(client, "assess_config", return_value=(config_status, None)),
            patch.object(client, "final_assess", return_value=(final_status, None)),
            patch.object(client, "validate_cwd", return_value={}),
            patch.object(client, "exclusive_json", side_effect=lambda path, value: captured.update({Path(path).name: copy.deepcopy(value)})),
            patch.object(client.subprocess, "Popen", return_value=fake_process) as popen,
            patch.object(Path, "read_bytes", read_bytes),
            patch.object(client.sys, "stdin", types.SimpleNamespace(buffer=io.BytesIO(b"GO\n"))),
        ):
            code = client.session()
        return code, captured, popen.call_count


class FirstProtocolFailureTests(unittest.TestCase):
    def setUp(self):
        self.block_process = patch.object(
            client.subprocess, "Popen", side_effect=AssertionError("real-process-forbidden")
        )
        self.block_process.start()
        self.addCleanup(self.block_process.stop)

    def test_known_forbidden_notification_has_fixed_method_and_refusal_metadata(self):
        first = FAILURE_MODULE.FirstProtocolFailure()
        first.record(
            {"method": "error", "params": {"message": "synthetic-error-value"}},
            "thread", "notification-gate", ValueError("forbidden-or-unknown-notification"),
        )
        self.assertEqual(first.snapshot(), {
            "version": 1,
            "phase": "thread",
            "stage": "notification-gate",
            "envelopeCategory": "notification",
            "method": {"category": "listed", "name": "error"},
            "exceptionClass": "ValueError",
            "refusalCode": "forbidden-or-unknown-notification",
        })

    def test_schema_invalid_thread_is_recorded_at_unchanged_response_schema_gate(self):
        code, captured, _ = run_mock_session(invalid_thread=True)
        receipt = captured["sanitized-result.json"]
        self.assertEqual(code, 1)
        self.assertIsNone(receipt["threadResponseFieldComparison"])
        self.assertEqual(receipt["firstProtocolFailure"]["phase"], "thread")
        self.assertEqual(receipt["firstProtocolFailure"]["stage"], "response-schema")
        self.assertEqual(receipt["firstProtocolFailure"]["refusalCode"], "protocol-schema-invalid")

    def test_unsolicited_response_uses_response_envelope_stage_without_identifier(self):
        first = FAILURE_MODULE.FirstProtocolFailure()
        first.record(
            {"jsonrpc": "2.0", "id": 999, "result": {"synthetic": "payload"}},
            "metadata", "response-envelope", ValueError("unsolicited-response"),
        )
        self.assertEqual(first.snapshot(), {
            "version": 1, "phase": "metadata", "stage": "response-envelope",
            "envelopeCategory": "result-response", "method": {"category": "absent", "name": None},
            "exceptionClass": "ValueError", "refusalCode": "unsolicited-response",
        })

    def test_valid_thread_then_forbidden_error_is_recorded_during_final_cleanup(self):
        after_close = [json.dumps({
            "method": "error",
            "params": {"error": {"message": "synthetic-terminal-error"},
                       "threadId": "synthetic-final-thread", "turnId": "synthetic-final-turn",
                       "willRetry": False},
        }).encode()]
        code, captured, popen_count = run_mock_session(after_close=after_close)
        receipt = captured["sanitized-result.json"]
        self.assertEqual(code, 1)
        self.assertEqual(popen_count, 1)
        self.assertEqual(receipt["stopReason"], "unexpected-final-output")
        self.assertTrue(receipt["threadBoundary"]["responseValidated"])
        self.assertEqual(receipt["threadBoundary"]["turnsRequested"], 0)
        self.assertEqual(
            [row["equal"] for row in receipt["threadResponseFieldComparison"]["comparisons"]],
            [True, True, True, True],
        )
        self.assertEqual(receipt["firstProtocolFailure"], {
            "version": 1, "phase": "cleanup", "stage": "notification-gate",
            "envelopeCategory": "notification", "method": {"category": "listed", "name": "error"},
            "exceptionClass": "ValueError", "refusalCode": "forbidden-or-unknown-notification",
        })
        self.assertNotIn("synthetic-terminal-error", json.dumps(receipt["firstProtocolFailure"]))

    def test_first_cleanup_failure_survives_later_errors_while_drain_continues(self):
        first_frame = {"method": "synthetic-unlisted-method", "params": {"threadId": "synthetic-final-thread"}}
        second_frame = {"method": "error", "params": {"message": "synthetic-later-error"}}
        final_valid = {"method": "thread/started", "params": {"thread": {
            "id": "synthetic-final-thread", "cwd": SYNTHETIC["cwd"],
            "model": SYNTHETIC["model"], "modelProvider": SYNTHETIC["modelProvider"],
            "ephemeral": True, "parentThreadId": None, "forkedFromId": None, "turns": [],
        }}}
        code, captured, _ = run_mock_session(after_close=[
            json.dumps(first_frame).encode(), json.dumps(second_frame).encode(),
            json.dumps(final_valid).encode(),
        ])
        receipt = captured["sanitized-result.json"]
        self.assertEqual(code, 1)
        self.assertEqual(receipt["stopReason"], "unexpected-final-output")
        self.assertEqual(receipt["notificationCount"], 3)
        self.assertEqual(receipt["eventCounts"], {"thread/started": 1})
        self.assertTrue(receipt["threadBoundary"]["threadStartedNotificationCorrelated"])
        self.assertEqual(receipt["threadBoundary"]["turnsRequested"], 0)
        self.assertEqual(receipt["threadBoundary"]["toolsRequested"], 0)
        self.assertEqual(
            [row["field"] for row in receipt["threadResponseFieldComparison"]["comparisons"]],
            ["cwd", "model", "modelProvider", "approvalPolicy"],
        )
        self.assertEqual(receipt["firstProtocolFailure"]["method"], {"category": "unlisted", "name": None})
        self.assertEqual(receipt["firstProtocolFailure"]["phase"], "cleanup")
        self.assertEqual(receipt["firstProtocolFailure"]["refusalCode"], "forbidden-or-unknown-notification")
        self.assertNotIn("synthetic-later-error", json.dumps(receipt["firstProtocolFailure"]))

    def test_unknown_values_and_exception_classes_never_escape_and_snapshot_isolated(self):
        first = FAILURE_MODULE.FirstProtocolFailure()
        first.record(
            {"method": "secret-synthetic-method", "params": {"value": "secret-payload-canary"}},
            "unrecognized-phase", "unrecognized-stage", RuntimeError("secret-exception-canary"),
        )
        snapshot = first.snapshot()
        self.assertEqual(snapshot, {
            "version": 1, "phase": "unknown", "stage": "unknown",
            "envelopeCategory": "notification", "method": {"category": "unlisted", "name": None},
            "exceptionClass": "unknown", "refusalCode": "unknown",
        })
        snapshot["method"]["category"] = "tampered"
        self.assertEqual(first.snapshot()["method"]["category"], "unlisted")
        serialized = json.dumps(first.snapshot())
        for canary in ("secret-synthetic-method", "secret-payload-canary", "secret-exception-canary"):
            self.assertNotIn(canary, serialized)


if __name__ == "__main__":
    unittest.main(verbosity=2)
