"""Six synthetic integration/authority tests for additive thread-field diagnostics."""
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
PROFILE_SOURCE = json.loads((HERE / "profile.json").read_text(encoding="utf-8"))
CONTRACT = json.loads((HERE / "schema-contract.json").read_text(encoding="utf-8"))
AUTHORITY = json.loads((HERE / "authorization-grant.json").read_text(encoding="utf-8"))
PRIOR = REPO / "experiments/government-comparison/evidence/s1-stderr-observation-diagnostic-20261008-r2"
PRIOR_CLIENT = importlib.util.spec_from_file_location("thread_response_fields_prior_client", PRIOR / "client.py")
PRIOR_CLIENT_MODULE = importlib.util.module_from_spec(PRIOR_CLIENT)
PRIOR_CLIENT.loader.exec_module(PRIOR_CLIENT_MODULE)
REAL_PROTOCOL_MODULE = PRIOR_CLIENT_MODULE.load_module(
    PRIOR_CLIENT_MODULE.PROTOCOL, PRIOR_CLIENT_MODULE.PROTOCOL_SHA,
    "thread_response_fields_real_protocol",
)
REAL_PROTOCOL = REAL_PROTOCOL_MODULE.Protocol(PRIOR_CLIENT_MODULE.SCHEMA, CONTRACT)
GATE_SPEC = importlib.util.spec_from_file_location("thread_response_fields_gate", HERE / "thread_gate.py")
GATE_MODULE = importlib.util.module_from_spec(GATE_SPEC)
GATE_SPEC.loader.exec_module(GATE_MODULE)

SYNTHETIC = {
    "cwd": "C:/Synthetic/ThreadAgentRoot",
    "model": "synthetic-model-alpha",
    "modelProvider": "synthetic-provider-alpha",
    "approvalPolicy": "never",
}
FIELDS = ("cwd", "model", "modelProvider", "approvalPolicy")


def make_profile(external_root):
    profile = copy.deepcopy(PROFILE_SOURCE)
    profile["cwd"] = SYNTHETIC["cwd"]
    profile["externalEvidence"] = str(external_root)
    profile["argv"] = ["synthetic-app-server", "--stdio"]
    for rpc in profile["rpc"]:
        if rpc.get("method") == "config/read":
            rpc["params"]["cwd"] = SYNTHETIC["cwd"]
    profile["threadStart"]["params"].update(SYNTHETIC)
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
            "cliVersion": "offline-synthetic",
            "createdAt": 1,
            "cwd": params["cwd"],
            "ephemeral": True,
            "id": "synthetic-thread-01",
            "model": params["model"],
            "modelProvider": params["modelProvider"],
            "preview": "",
            "projectId": None,
            "sessionId": "synthetic-session-01",
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
    def __init__(self, chunks=None, blocking=False):
        self.chunks = iter(chunks or ())
        self.pending = queue.Queue() if blocking else None

    def read1(self, _limit):
        if self.pending is not None:
            try:
                return self.pending.get(timeout=2)
            except queue.Empty:
                return b""
        return next(self.chunks, b"")

    def push(self, raw):
        self.pending.put(raw)


class FakeStdin:
    def __init__(self, stdout, replies):
        self.stdout = stdout
        self.replies = replies

    def write(self, _raw):
        request = json.loads(_raw)
        if "id" in request:
            reply = {"jsonrpc": "2.0", "id": request["id"], "result": self.replies[request["id"]]}
            self.stdout.push(json.dumps(reply).encode() + b"\n")
        return None

    def flush(self):
        return None

    def close(self):
        self.stdout.push(b"")
        return None


class FakeProcess:
    pid = 424242

    def __init__(self, replies):
        self.stdout = FakeStream(blocking=True)
        self.stdin = FakeStdin(self.stdout, replies)
        self.stderr = FakeStream([b""])

    def wait(self, timeout=None):
        return 0

    def poll(self):
        return 0

    def kill(self):
        return None


def run_synthetic_session(reject_thread_schema=False):
    """Exercise session/finally with fake streams and a mocked process boundary."""
    captured = {}
    with tempfile.TemporaryDirectory(prefix="thread-fields-synthetic-") as temp:
        out = Path(temp)
        profile = make_profile(out)
        binding = {
            "key": client.KEY,
            "grantIssuedUtc": client.ISSUED,
            "slotAssignedUtc": client.ISSUED,
            "sourceCommit": "a" * 40,
            "freezeSha256": "b" * 64,
        }
        replies = {}
        for rpc in profile["rpc"] + [profile["threadStart"]]:
            if "id" not in rpc:
                continue
            payload = thread_response(profile, model="synthetic-model-beta") if rpc["method"] == "thread/start" else {}
            replies[rpc["id"]] = payload
        fake_process = FakeProcess(replies)

        class SelectiveProtocol:
            def __init__(self, _archive, contract):
                self.contract = contract

            def validate(self, member, value):
                if member == self.contract["responseSchemas"]["thread/start"]:
                    if reject_thread_schema:
                        raise ValueError("synthetic-schema-rejected")
                    REAL_PROTOCOL.validate(member, value)

        protocol_module = types.SimpleNamespace(Protocol=SelectiveProtocol)
        source_files = {
            (HERE / "thread_gate.py").relative_to(REPO).as_posix():
                hashlib.sha256((HERE / "thread_gate.py").read_bytes()).hexdigest(),
        }
        job_record = {"workerPid": os.getpid(), "controllerPid": os.getppid(), **binding}
        real_read_bytes = Path.read_bytes

        def safe_read_bytes(path):
            resolved = path.resolve()
            if resolved == (out / "job-assigned.json").resolve():
                return json.dumps(job_record).encode()
            if resolved == (HERE / "request.json").resolve():
                return json.dumps({"sourceFiles": source_files}).encode()
            return real_read_bytes(path)

        real_load_module = client.load_module

        def selected_load_module(path, expected, name):
            if Path(path) == Path(client.PROTOCOL):
                return protocol_module
            return real_load_module(path, expected, name)

        config_ok = "config-precheck-satisfied-awaiting-requirements"
        final_ok = "reported-config-and-requirements-observed"
        with (
            patch.object(client, "load_request", return_value=(profile, binding)),
            patch.object(client, "load_module", side_effect=selected_load_module),
            patch.object(client, "sanitize", return_value={}),
            patch.object(client, "assess_config", return_value=(config_ok, None)),
            patch.object(client, "final_assess", return_value=(final_ok, None)),
            patch.object(client, "validate_cwd", return_value={}),
            patch.object(client, "exclusive_json", side_effect=lambda path, value: captured.update({Path(path).name: copy.deepcopy(value)})),
            patch.object(client.subprocess, "Popen", return_value=fake_process) as popen,
            patch.object(Path, "read_bytes", safe_read_bytes),
            patch.object(client.sys, "stdin", types.SimpleNamespace(buffer=io.BytesIO(b"GO\n"))),
        ):
            exit_code = client.session()
        return exit_code, captured, popen.call_count


class ThreadResponseFieldIntegrationTests(unittest.TestCase):
    def setUp(self):
        self.no_process = patch.object(
            client.subprocess, "Popen", side_effect=AssertionError("unexpected-process-start")
        )
        self.no_process.start()
        self.addCleanup(self.no_process.stop)

    def test_mismatch_flows_through_mocked_session_cleanup_into_additive_receipt(self):
        exit_code, captured, popen_count = run_synthetic_session()
        receipt = captured["sanitized-result.json"]
        self.assertEqual(exit_code, 1)
        self.assertEqual(popen_count, 1)
        self.assertEqual(receipt["stopReason"], "thread-response-identity-or-policy-mismatch")
        self.assertEqual(receipt["threadResponseFieldComparison"]["comparisons"], [
            {"field": "cwd", "expectedType": "string", "actualType": "string", "equal": True},
            {"field": "model", "expectedType": "string", "actualType": "string", "equal": False},
            {"field": "modelProvider", "expectedType": "string", "actualType": "string", "equal": True},
            {"field": "approvalPolicy", "expectedType": "string", "actualType": "string", "equal": True},
        ])
        self.assertNotIn("synthetic-model-beta", json.dumps(receipt["threadResponseFieldComparison"]))
        self.assertIn("threadBoundary", receipt)
        self.assertIn("stderrDiagnostic", receipt)

    def test_original_thread_boundary_fields_remain_unchanged_on_comparison_failure(self):
        _, captured, _ = run_synthetic_session()
        boundary = captured["sanitized-result.json"]["threadBoundary"]
        self.assertEqual(set(boundary), {
            "responseValidated", "threadId", "threadStartedNotificationCorrelated",
            "threadClosedNotificationCorrelated", "pendingLifecycleCount",
            "reportedPermissionObservation", "turnsRequested", "toolsRequested",
        })
        self.assertEqual(boundary, {
            "responseValidated": False,
            "threadId": None,
            "threadStartedNotificationCorrelated": False,
            "threadClosedNotificationCorrelated": False,
            "pendingLifecycleCount": 0,
            "reportedPermissionObservation": None,
            "turnsRequested": 0,
            "toolsRequested": 0,
        })

    def test_schema_rejection_precedes_field_comparison_even_after_cleanup(self):
        exit_code, captured, _ = run_synthetic_session(reject_thread_schema=True)
        receipt = captured["sanitized-result.json"]
        self.assertEqual(exit_code, 1)
        self.assertEqual(receipt["stopReason"], "synthetic-schema-rejected")
        self.assertIsNone(receipt["threadResponseFieldComparison"])
        self.assertFalse(receipt["threadBoundary"]["responseValidated"])

    def test_mismatch_stops_before_turn_start_and_preserves_rpc_order_gate(self):
        _, captured, _ = run_synthetic_session()
        receipt = captured["sanitized-result.json"]
        sent = [item["method"] for item in receipt["sentRpc"]]
        self.assertEqual(sent, client.METHODS)
        self.assertIn("thread/start", sent)
        self.assertNotIn("turn/start", sent)
        self.assertEqual(receipt["turnStartRequests"], 0)
        with self.assertRaisesRegex(ValueError, "forbidden-or-out-of-order-client-method"):
            client.admit_rpc("turn/start", sent, False, False)

    def test_field_comparison_is_copy_isolated_and_original_finish_stays_separate(self):
        params = copy.deepcopy(PROFILE_SOURCE["threadStart"]["params"])
        params.update(SYNTHETIC)
        profile = {"threadStart": {"params": params}}
        gate = GATE_MODULE.ThreadGate(profile, REAL_PROTOCOL, CONTRACT)
        with self.assertRaisesRegex(ValueError, "thread-response-identity-or-policy-mismatch"):
            gate.accept_response(thread_response(profile, model="synthetic-model-beta"))
        comparison = gate.field_comparison()
        comparison["comparisons"][0]["field"] = "altered"
        self.assertEqual(gate.field_comparison()["comparisons"][0]["field"], "cwd")
        self.assertEqual(set(gate.finish()), {
            "responseValidated", "threadId", "threadStartedNotificationCorrelated",
            "threadClosedNotificationCorrelated", "pendingLifecycleCount",
            "reportedPermissionObservation", "turnsRequested", "toolsRequested",
        })
        self.assertFalse(gate.finish()["responseValidated"])

    def test_fresh_grant_slot_and_request_freeze_bindings_are_exact(self):
        grant = copy.deepcopy(AUTHORITY["grant"])
        slot = {
            "owner": "Scientist", "key": client.KEY, "status": client.SLOT_STATUS,
            "grantIssuedUtc": client.ISSUED, "assignedUtc": client.ISSUED,
        }
        self.assertEqual(client.validate_live_authority(grant, slot, copy.deepcopy(grant)), client.ISSUED)
        changed_type = copy.deepcopy(grant)
        changed_type["maxNewAppServerTrees"] = True
        with self.assertRaises(ValueError):
            client.validate_live_authority(changed_type, slot, grant)
        for field, value in (("owner", "Architect"), ("status", "Queued"),
                             ("key", "synthetic-foreign-key"),
                             ("grantIssuedUtc", "2026-10-08T18:07:11Z")):
            with self.subTest(slot_field=field):
                with self.assertRaises(ValueError):
                    client.validate_live_authority(grant, {**slot, field: value}, grant)

        request = {
            "key": client.KEY, "grantIssuedUtc": client.ISSUED,
            "sourceCommit": "a" * 40, "slotAssignedUtc": client.ISSUED,
        }
        freeze = {**request, "frozenAtUtc": client.ISSUED}
        client.validate_binding_identity(request, freeze, client.ISSUED)
        stale_freeze = {**freeze, "slotAssignedUtc": "2026-10-08T18:07:09Z"}
        with self.assertRaisesRegex(ValueError, "slot-freeze-binding-mismatch"):
            client.validate_binding_identity(request, stale_freeze, client.ISSUED)


if __name__ == "__main__":
    unittest.main(verbosity=2)
