"""Six bounded synthetic tests for passive stderr-observation diagnostics."""

from __future__ import annotations

import copy
import importlib.util
import json
from pathlib import Path
import tempfile
import threading
import time
import unittest
from unittest.mock import patch

import client


HERE = Path(__file__).resolve().parent
profile = json.loads((HERE / "profile.json").read_text(encoding="utf-8"))
contract = json.loads((HERE / "schema-contract.json").read_text(encoding="utf-8"))
protocol_module = client.load_module(client.PROTOCOL, client.PROTOCOL_SHA, "observation_test_protocol")
protocol = protocol_module.Protocol(client.SCHEMA, contract)
gate_spec = importlib.util.spec_from_file_location("observation_test_gate", HERE / "thread_gate.py")
gate_module = importlib.util.module_from_spec(gate_spec)
gate_spec.loader.exec_module(gate_module)
prior = client.load_module(client.PRIOR, client.PRIOR_SHA, "observation_test_response_guard")


def thread(thread_id="thread-observation-test"):
    return {
        "cliVersion": "offline-test", "createdAt": 1, "cwd": profile["cwd"],
        "ephemeral": True, "id": thread_id, "model": "gpt-6.1-sol",
        "modelProvider": "openai", "preview": "", "projectId": None,
        "sessionId": "session-observation-test", "source": "appServer",
        "status": {"type": "active", "activeFlags": []}, "turns": [],
        "updatedAt": 1, "parentThreadId": None, "forkedFromId": None,
    }


def thread_response():
    params = profile["threadStart"]["params"]
    return {
        "approvalPolicy": params["approvalPolicy"], "approvalsReviewer": "user",
        "cwd": params["cwd"], "model": params["model"],
        "modelProvider": params["modelProvider"],
        "sandbox": {"type": "readOnly", "networkAccess": False},
        "activePermissionProfile": {"id": ":read-only", "extends": None},
        "thread": thread(),
    }


def new_gate():
    return gate_module.ThreadGate(profile, protocol, contract)


class FakeStream:
    def __init__(self, chunks):
        self._chunks = iter(chunks)

    def read1(self, _limit):
        return next(self._chunks, b"")


class FakePump:
    def __init__(self, alive=False):
        self.alive = alive

    def is_alive(self):
        return self.alive


class FakeNative:
    def __init__(self, code=0):
        self.code = code

    def poll(self):
        return self.code


class TrackingLock:
    def __init__(self, capture_event, lock):
        self.capture_event = capture_event
        self.lock = lock
        self.event_seen_at_enter = False

    def __enter__(self):
        self.event_seen_at_enter = self.capture_event.is_set()
        self.lock.acquire()
        return self

    def __exit__(self, exc_type, exc, traceback):
        self.lock.release()


class PassiveStderrDiagnosticCases(unittest.TestCase):
    def setUp(self):
        self._popen_guard = patch.object(
            client.subprocess, "Popen", side_effect=AssertionError("process-start-forbidden")
        )
        self._popen_guard.start()
        self.addCleanup(self._popen_guard.stop)
        self.authorization = json.loads((HERE / "authorization-grant.json").read_text(encoding="utf-8"))
        self.grant = self.authorization["grant"]

    def test_passive_capture_allows_fake_thread_reply_then_stops_before_turn(self):
        capture_seen = threading.Event()
        fatal = threading.Event()
        consumer = client.StderrConsumer(capture_seen, fatal)
        cleanup = threading.Event()
        stream = FakeStream([b"progress ", b"continues\n"])
        consumer.observe_read(stream.read1(1024), threading.Lock(), cleanup)
        consumer.observe_read(stream.read1(1024), threading.Lock(), cleanup)
        self.assertTrue(capture_seen.is_set())
        self.assertFalse(fatal.is_set())

        gate = new_gate()
        sent = []
        for method in client.METHODS[:4]:
            client.admit_rpc(method, sent, fatal.is_set(), gate.validated)
            sent.append(method)
        client.admit_rpc("thread/start", sent, fatal.is_set(), gate.validated)
        sent.append("thread/start")
        result = thread_response()
        protocol.validate(contract["responseSchemas"]["thread/start"], result)
        gate.accept_response(result)
        with self.assertRaisesRegex(ValueError, "rpc-after-thread-terminal"):
            client.admit_rpc("turn/start", sent, fatal.is_set(), gate.validated)
        boundary = gate.finish()
        self.assertTrue(boundary["responseValidated"])
        self.assertEqual(boundary["turnsRequested"], 0)
        self.assertEqual(boundary["toolsRequested"], 0)

    def test_structured_rpc_schema_notification_auth_and_config_errors_remain_terminal(self):
        rpc_error = {
            "jsonrpc": "2.0", "id": 3,
            "error": {"code": -32001, "message": "authentication required"},
        }
        with self.assertRaisesRegex(ValueError, "rpc-error"):
            prior.validate_response_envelope(rpc_error, 3)

        malformed_config = {}
        with self.assertRaises(ValueError):
            protocol.validate(contract["responseSchemas"]["config/read"], malformed_config)
        with self.assertRaises(ValueError):
            client.reject_provisional({"configWarning": "offline synthetic warning"})

        notification = {
            "method": "error",
            "params": {
                "error": {"message": "offline synthetic error",
                          "codexErrorInfo": {"responseStreamDisconnected": {"httpStatusCode": 503}}},
                "threadId": "thread-observation-test", "turnId": "turn-synthetic", "willRetry": False,
            },
        }
        protocol.validate("ServerNotification.json", notification)
        with self.assertRaisesRegex(ValueError, "forbidden-or-unknown-notification"):
            new_gate().accept_notification(notification["method"], notification["params"])

        fatal = threading.Event()
        fatal.set()
        with self.assertRaisesRegex(ValueError, "stderr-observation-limit"):
            client.admit_rpc("thread/start", client.METHODS[:4], fatal.is_set(), False)

    def test_total_stderr_ceiling_is_passive_below_and_terminal_at_16384(self):
        seen = threading.Event()
        fatal = threading.Event()
        consumer = client.StderrConsumer(seen, fatal)
        self.assertIsNone(client.consume_stderr(consumer, b"x" * 8192))
        self.assertIsNone(client.consume_stderr(consumer, b"x" * (16383 - 8192), cleanup=True))
        self.assertEqual(consumer.observed_bytes, 16383)
        self.assertTrue(seen.is_set())
        self.assertFalse(fatal.is_set())
        self.assertEqual(client.consume_stderr(consumer, b"x", cleanup=True), "stderr-observation-limit")
        self.assertEqual(consumer.observed_bytes, 16384)
        self.assertTrue(fatal.is_set())

        oversized = client.StderrConsumer(threading.Event(), threading.Event())
        with self.assertRaisesRegex(ValueError, "stderr-observation-limit"):
            oversized.feed(b"z" * 16385)
        self.assertTrue(oversized.fatal_event.is_set())
        self.assertEqual(oversized.observed_bytes, 0)

    def test_authority_slot_and_request_freeze_identity_reject_stale_bindings(self):
        slot = {
            "owner": "Scientist", "key": client.KEY, "status": client.SLOT_STATUS,
            "grantIssuedUtc": client.ISSUED, "assignedUtc": "2026-10-08T17:35:00Z",
        }
        assigned = client.validate_live_authority(copy.deepcopy(self.grant), slot, self.grant)
        self.assertEqual(assigned, slot["assignedUtc"])

        invalid_slots = (
            {**slot, "owner": "Architect"}, {**slot, "status": "Queued"},
            {**slot, "key": "foreign-slot"}, {**slot, "grantIssuedUtc": "2026-10-08T17:31:24Z"},
        )
        for candidate in invalid_slots:
            with self.subTest(slot=candidate["owner"], status=candidate["status"]):
                with self.assertRaises(ValueError):
                    client.validate_live_authority(self.grant, candidate, self.grant)

        for name, value in (("key", "stale-key"), ("issuedUtc", "2026-10-08T17:31:24Z"),
                            ("status", "Revoked")):
            changed = copy.deepcopy(self.grant)
            changed[name] = value
            with self.subTest(grant_field=name):
                with self.assertRaises(ValueError):
                    client.validate_live_authority(changed, slot, changed)
        bool_for_one = copy.deepcopy(self.grant)
        bool_for_one["maxNewAppServerTrees"] = True
        with self.assertRaises(ValueError):
            client.validate_live_authority(bool_for_one, slot, self.grant)

        authority = {
            "sourceCoordinationPath": client.COORDINATION,
            "sourceJsonPointer": client.AUTHORITY_POINTER,
            "grant": copy.deepcopy(self.grant),
        }
        coordination = {"threads": [{"name": "Scientist", "evidence": {
            "stderrObservationDiagnosticR2Grant": copy.deepcopy(self.grant)
        }}], "fullSuiteSlot": slot, "cursor": "unrelated synthetic cursor"}
        actual_read_bytes = Path.read_bytes
        target = Path(client.COORDINATION).resolve()

        def read_bytes(path):
            if path.resolve() == target:
                return json.dumps(coordination).encode("utf-8")
            return actual_read_bytes(path)

        with patch.object(Path, "read_bytes", read_bytes):
            self.assertEqual(client.live_authority(authority)[1], slot["assignedUtc"])
            bad_pointer = {**authority, "sourceJsonPointer": "threads[name=Architect].evidence.grant"}
            with self.assertRaises(ValueError):
                client.live_authority(bad_pointer)

        assigned = "2026-10-08T17:35:00Z"
        request = {"key": client.KEY, "grantIssuedUtc": client.ISSUED,
                   "slotAssignedUtc": assigned, "sourceCommit": "a" * 40}
        freeze = {**request, "frozenAtUtc": "2026-10-08T17:36:00Z"}
        client.validate_binding_identity(request, freeze, assigned)
        stale_freezes = (
            {**freeze, "key": "stale-freeze-key"},
            {**freeze, "grantIssuedUtc": "2026-10-08T17:31:24Z"},
            {**freeze, "slotAssignedUtc": "2026-10-08T17:34:59Z"},
            {**freeze, "sourceCommit": "b" * 40},
            {**freeze, "frozenAtUtc": "2026-10-08T17:34:59Z"},
        )
        for stale in stale_freezes:
            with self.subTest(freeze="stale-identity-or-time"):
                with self.assertRaises(ValueError):
                    client.validate_binding_identity(request, stale, assigned)

    def test_cleanup_writes_once_only_after_native_and_pumps_are_quiescent(self):
        cleanup = threading.Event()
        seen, fatal = threading.Event(), threading.Event()
        consumer = client.StderrConsumer(seen, fatal)
        consumer.feed(b"synthetic lifecycle diagnostic\n")

        with tempfile.TemporaryDirectory(prefix="observation-lifecycle-pump-") as temp:
            cleanup.set()
            blocked = consumer.finish_after_cleanup(Path(temp).resolve(), cleanup, [FakePump(True)], FakeNative(0), time.monotonic() + 20)
            self.assertEqual(blocked["excerptStatus"], "not-written-not-quiescent")
            self.assertFalse(consumer._collector._finished)
            self.assertFalse((Path(temp) / "private-redacted-stderr").exists())

        other = client.StderrConsumer(threading.Event(), threading.Event())
        other.feed(b"synthetic lifecycle diagnostic\n")
        cleanup = threading.Event()
        cleanup.set()
        with tempfile.TemporaryDirectory(prefix="observation-lifecycle-success-") as temp:
            root = Path(temp).resolve()
            calls = []
            original = other._collector.write_private_excerpt

            def count_write(directory):
                calls.append(1)
                return original(directory)

            with patch.object(other._collector, "write_private_excerpt", side_effect=count_write):
                receipt = other.finish_after_cleanup(root, cleanup, [FakePump(False)], FakeNative(0), time.monotonic() + 20)
                cached = other.finish_after_cleanup(root, cleanup, [FakePump(False)], FakeNative(0), time.monotonic() + 20)
            self.assertEqual(receipt["excerptStatus"], "written-private")
            self.assertEqual(cached, receipt)
            self.assertEqual(calls, [1])
            self.assertLessEqual((root / "private-redacted-stderr" / "redacted-stderr.json").stat().st_size, 2048)

        expired = client.StderrConsumer(threading.Event(), threading.Event())
        expired.feed(b"synthetic lifecycle diagnostic\n")
        cleanup = threading.Event()
        cleanup.set()
        with tempfile.TemporaryDirectory(prefix="observation-lifecycle-deadline-") as temp:
            refused = expired.finish_after_cleanup(Path(temp).resolve(), cleanup, [FakePump(False)], FakeNative(0), time.monotonic() - 1)
            self.assertEqual(refused["excerptStatus"], "not-written-cleanup-deadline")

    def test_capture_caps_redaction_fragmentation_and_passive_event_precede_lock(self):
        seen, fatal = threading.Event(), threading.Event()
        consumer = client.StderrConsumer(seen, fatal)
        cleanup = threading.Event()
        stream = FakeStream([
            b"diagnostic from ",
            b"https://example.test/path alice@example.test /home/synthetic/private\n",
            b"Authorization: Bearer synthetic-canary\n",
        ])
        lock = threading.Lock()
        lock.acquire()
        tracking = TrackingLock(seen, lock)
        errors = []

        def capture_first_chunk():
            try:
                consumer.observe_read(stream.read1(1024), tracking, cleanup)
            except Exception as error:
                errors.append(type(error).__name__)

        worker = threading.Thread(target=capture_first_chunk, daemon=True)
        worker.start()
        self.assertTrue(seen.wait(2))
        self.assertTrue(worker.is_alive())
        self.assertFalse(fatal.is_set())
        lock.release()
        worker.join(2)
        self.assertEqual(errors, [])
        self.assertTrue(tracking.event_seen_at_enter)

        consumer.observe_read(stream.read1(1024), threading.Lock(), cleanup)
        consumer.observe_read(stream.read1(1024), threading.Lock(), cleanup)
        client.admit_rpc("initialize", [], fatal.is_set(), False)
        self.assertTrue(seen.is_set())
        self.assertFalse(fatal.is_set())
        with tempfile.TemporaryDirectory(prefix="observation-redaction-") as temp:
            cleanup.set()
            receipt = consumer.finish_after_cleanup(Path(temp).resolve(), cleanup, [FakePump(False)], FakeNative(0), time.monotonic() + 20)
            self.assertEqual(receipt["excerptStatus"], "written-private")
            sanitized = (Path(temp) / "private-redacted-stderr" / "redacted-stderr.json").read_bytes()
            self.assertIn(b"[url]", sanitized)
            self.assertIn(b"[email]", sanitized)
            self.assertNotIn(b"/home/synthetic/private", sanitized)
            self.assertNotIn(b"synthetic-canary", sanitized)

        capped = client.StderrConsumer(threading.Event(), threading.Event())
        exact_input = (b'"' * 250 + b"\n") * 3 + (b'"' * 270 + b"\n")
        self.assertEqual(len(exact_input), 1024)
        capped.feed(exact_input)
        self.assertFalse(capped.fatal_event.is_set())
        metadata = capped._collector.finish()
        output = capped._collector.render_private()
        self.assertEqual(metadata["selectedInputBytes"], 1024)
        self.assertEqual(metadata["completeLines"], 4)
        self.assertTrue(metadata["outputTruncated"])
        self.assertLessEqual(len(output), 2048)
        self.assertLessEqual(metadata["includedLines"], 4)
        with self.assertRaisesRegex(ValueError, "forbidden-or-out-of-order-client-method"):
            client.admit_rpc("turn/start", client.METHODS[:5], fatal.is_set(), False)


if __name__ == "__main__":
    unittest.main(verbosity=2)
