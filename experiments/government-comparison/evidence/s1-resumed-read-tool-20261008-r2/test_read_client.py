"""Synthetic integration cases for the resumed public-read client."""
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
THREAD_ID = "synthetic-read-thread-01"
TURN_ID = "synthetic-read-turn-01"
COMMAND_ID = "synthetic-native-command-01"
SENTINEL = "0123456789abcdef0123456789abcdef"


def make_profile(external_root):
    profile = copy.deepcopy(PROFILE_SOURCE)
    profile["cwd"] = "C:/Synthetic/ReadProbe"
    profile["externalEvidence"] = str(external_root)
    profile["argv"] = ["synthetic-app-server", "--stdio"]
    profile["command"] = "Synthetic-Read-Probe"
    profile["turnPrompt"] = "Synthetic prompt: perform one public read and return the fixed shape."
    profile["expectedFiles"] = {}
    for rpc in profile["rpc"]:
        if rpc.get("method") == "config/read":
            rpc["params"]["cwd"] = profile["cwd"]
    profile["threadStart"]["params"]["cwd"] = profile["cwd"]
    profile["turnStart"]["params"]["input"] = [{"type": "text", "text": profile["turnPrompt"]}]
    profile["limits"]["tokenStopThreshold"] = 10000
    return profile


def synthetic_expected():
    return {"sentinel": SENTINEL, "fileSha256": "0" * 64}


def user_item(profile):
    return {"id": "synthetic-user-item-01", "type": "userMessage",
            "content": [{"type": "text", "text": profile["turnPrompt"]}]}


def thread_record(profile):
    params = profile["threadStart"]["params"]
    return {
        "cliVersion": "synthetic-client",
        "createdAt": 1,
        "cwd": profile["cwd"],
        "ephemeral": True,
        "id": THREAD_ID,
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
    }


def thread_response(profile, permission=None):
    params = profile["threadStart"]["params"]
    return {
        "approvalPolicy": params["approvalPolicy"],
        "approvalsReviewer": "user",
        "cwd": profile["cwd"],
        "model": params["model"],
        "modelProvider": params["modelProvider"],
        "sandbox": {"type": "readOnly", "networkAccess": False},
        "activePermissionProfile": permission if permission is not None else {"id": ":read-only", "extends": None},
        "thread": thread_record(profile),
    }


def command_item(profile, *, item_id=COMMAND_ID, command=None, status="completed", exit_code=0,
                 output=SENTINEL):
    return {
        "id": item_id,
        "type": "commandExecution",
        "command": profile["command"] if command is None else command,
        "commandActions": [],
        "cwd": profile["cwd"],
        "status": status,
        "source": "agent",
        "exitCode": exit_code,
        "aggregatedOutput": output,
    }


def final_item():
    return {"id": "synthetic-final-item-01", "type": "agentMessage",
            "text": json.dumps({"sentinel": SENTINEL}), "phase": "final_answer"}


def usage_params(*, input_tokens, output_tokens, last_input=None, last_output=None,
                 include_cache_write=True, thread_id=THREAD_ID, turn_id=TURN_ID):
    def breakdown(input_value, output_value, include_write):
        value = {
            "inputTokens": input_value,
            "outputTokens": output_value,
            "cachedInputTokens": 0,
            "reasoningOutputTokens": 0,
            "totalTokens": input_value + output_value,
        }
        if include_write:
            value["cacheWriteInputTokens"] = 0
        return value
    last_input = input_tokens if last_input is None else last_input
    last_output = output_tokens if last_output is None else last_output
    return {
        "threadId": thread_id,
        "turnId": turn_id,
        "tokenUsage": {
            "modelContextWindow": 100000,
            "total": breakdown(input_tokens, output_tokens, include_cache_write),
            "last": breakdown(last_input, last_output, include_cache_write),
        },
    }


def response_turn(profile, turn_id=TURN_ID):
    return {"turn": {"id": turn_id, "status": "inProgress", "items": [user_item(profile)]}}


def notification(method, params):
    # The copied transport's strict notification frame is method/params only.
    return {"method": method, "params": params}


class FakeStream:
    def __init__(self, *, blocking=False):
        self.pending = queue.Queue() if blocking else None

    def read1(self, _limit):
        if self.pending is None:
            return b""
        try:
            return self.pending.get(timeout=2)
        except queue.Empty:
            return b""

    def push_frame(self, frame):
        self.pending.put(json.dumps(frame, separators=(",", ":")).encode("utf-8") + b"\n")

    def close_stream(self):
        if self.pending is not None:
            self.pending.put(b"")


class FakeStdin:
    def __init__(self, process, profile, mode, options):
        self.process = process
        self.profile = profile
        self.mode = mode
        self.options = options
        self.requests = []

    def _reply(self, request, result):
        self.process.stdout.push_frame({"jsonrpc": "2.0", "id": request["id"], "result": result})

    def _notify(self, method, params):
        self.process.stdout.push_frame(notification(method, params))

    def _turn_notifications(self):
        profile = self.profile
        if self.mode in ("success", "usage-latest", "stale-command-snapshot"):
            self._notify("turn/started", {
                "threadId": THREAD_ID,
                "turn": {"id": TURN_ID, "status": "inProgress", "items": []},
            })
            if self.mode == "usage-latest":
                self._notify("thread/tokenUsage/updated", usage_params(input_tokens=2000, output_tokens=500))
                self._notify("thread/tokenUsage/updated", usage_params(
                    input_tokens=2500, output_tokens=700, last_input=1800, last_output=300,
                    include_cache_write=False,
                ))
            else:
                self._notify("thread/tokenUsage/updated", usage_params(input_tokens=2500, output_tokens=700))
            self._notify("item/started", {
                "threadId": THREAD_ID, "turnId": TURN_ID, "startedAtMs": 1,
                "item": command_item(profile, status="inProgress", exit_code=None, output=""),
            })
            self._notify("item/completed", {
                "threadId": THREAD_ID, "turnId": TURN_ID, "completedAtMs": 2,
                "item": command_item(profile),
            })
            terminal_items = [final_item()]
            if self.mode == "stale-command-snapshot":
                terminal_items.insert(0, command_item(profile, status="inProgress", exit_code=None, output=""))
            self._notify("turn/completed", {
                "threadId": THREAD_ID,
                "turn": {"id": TURN_ID, "status": "completed", "items": terminal_items},
            })
        elif self.mode == "structured-error":
            self._notify("turn/started", {
                "threadId": THREAD_ID,
                "turn": {"id": TURN_ID, "status": "inProgress", "items": []},
            })
            self._notify("error", {
                "threadId": THREAD_ID, "turnId": TURN_ID,
                "error": {"message": "synthetic structured failure"}, "willRetry": False,
            })
            if self.options.get("drain_followup"):
                self._notify("thread/tokenUsage/updated", {"threadId": THREAD_ID, "turnId": TURN_ID})
        elif self.mode == "foreign-id":
            self._notify("turn/started", {
                "threadId": THREAD_ID,
                "turn": {"id": TURN_ID, "status": "inProgress", "items": []},
            })
            self._notify("thread/tokenUsage/updated", usage_params(
                input_tokens=10, output_tokens=2, thread_id="synthetic-foreign-thread",
            ))
        elif self.mode == "early-turn":
            self._notify("turn/started", {
                "threadId": THREAD_ID,
                "turn": {"id": "synthetic-early-turn", "status": "inProgress", "items": []},
            })
        elif self.mode == "second-command":
            self._notify("turn/started", {
                "threadId": THREAD_ID,
                "turn": {"id": TURN_ID, "status": "inProgress", "items": []},
            })
            self._notify("item/started", {
                "threadId": THREAD_ID, "turnId": TURN_ID, "startedAtMs": 1,
                "item": command_item(profile, status="inProgress", exit_code=None, output=""),
            })
            self._notify("item/started", {
                "threadId": THREAD_ID, "turnId": TURN_ID, "startedAtMs": 2,
                "item": command_item(profile, item_id="synthetic-command-02", status="inProgress",
                                     exit_code=None, output=""),
            })
        elif self.mode == "wrong-command":
            self._notify("turn/started", {
                "threadId": THREAD_ID,
                "turn": {"id": TURN_ID, "status": "inProgress", "items": []},
            })
            self._notify("item/started", {
                "threadId": THREAD_ID, "turnId": TURN_ID, "startedAtMs": 1,
                "item": command_item(profile, command="Synthetic-Other-Command",
                                     status="inProgress", exit_code=None, output=""),
            })
        elif self.mode == "overshoot":
            self._notify("turn/started", {
                "threadId": THREAD_ID,
                "turn": {"id": TURN_ID, "status": "inProgress", "items": []},
            })
            self._notify("thread/tokenUsage/updated", usage_params(input_tokens=9000, output_tokens=1200))
        elif self.mode == "bad-usage-schema":
            self._notify("thread/tokenUsage/updated", {"threadId": THREAD_ID, "turnId": TURN_ID})
        elif self.mode == "missing-jsonrpc-method":
            self.process.stdout.push_frame({"jsonrpc": "2.0", "method": "turn/started", "params": {
                "threadId": THREAD_ID, "turn": {"id": TURN_ID, "status": "inProgress", "items": []},
            }})

    def write(self, raw):
        request = json.loads(raw)
        self.requests.append(copy.deepcopy(request))
        method = request["method"]
        if method == "thread/start":
            result = thread_response(self.profile, self.options.get("permission"))
            if self.mode == "readonly-gate":
                result["activePermissionProfile"] = {"id": ":workspace-write", "extends": None}
            self._reply(request, result)
            self._notify("thread/started", {"thread": thread_record(self.profile)})
        elif method == "turn/start":
            if self.mode == "early-turn":
                self._turn_notifications()
            self._reply(request, response_turn(self.profile))
            if self.mode != "early-turn":
                self._turn_notifications()
        elif method == "turn/interrupt":
            self._reply(request, {})
        elif "id" in request:
            self._reply(request, {})
        return len(raw)

    def flush(self):
        return None

    def close(self):
        self.process.stdout.close_stream()


class FakeProcess:
    pid = 424242

    def __init__(self, profile, mode, options):
        self.stdout = FakeStream(blocking=True)
        self.stdin = FakeStdin(self, profile, mode, options)
        self.stderr = FakeStream()

    def wait(self, timeout=None):
        return 0

    def poll(self):
        return 0

    def kill(self):
        return None


def run_synthetic_session(mode="success", **options):
    captured = {}
    with tempfile.TemporaryDirectory(prefix="resumed-read-synthetic-") as temp:
        profile = make_profile(Path(temp))
        expected = synthetic_expected()
        binding = {
            "key": client.KEY,
            "grantIssuedUtc": client.ISSUED,
            "slotAssignedUtc": client.ISSUED,
            "sourceCommit": "a" * 40,
            "freezeSha256": "b" * 64,
        }
        process = FakeProcess(profile, mode, options)
        source_files = {}
        for name in ("collector.py", "thread_gate.py", "protocol_failure.py", "startup_observer.py"):
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
                # Keep metadata fixtures concise; every read-tool and lifecycle
                # schema remains the actual pinned validator.
                if member not in {
                    "v1/InitializeResponse.json",
                    "v2/ConfigReadResponse.json",
                    "v2/ConfigRequirementsReadResponse.json",
                }:
                    return self.actual.validate(member, value)

        def selected_load_module(path, expected_sha, name):
            if Path(path) == Path(client.PROTOCOL):
                module = real_load_module(path, expected_sha, name)
                return types.SimpleNamespace(
                    Protocol=lambda archive, contract: SelectiveProtocol(archive, contract, module.Protocol)
                )
            return real_load_module(path, expected_sha, name)

        def safe_read_bytes(path):
            resolved = path.resolve()
            if resolved == (HERE / "request.json").resolve():
                return json.dumps({"sourceFiles": source_files}).encode()
            if resolved == (HERE / "expected.json").resolve():
                return json.dumps(expected).encode()
            if resolved == (Path(temp) / "job-assigned.json").resolve():
                return json.dumps(job_record).encode()
            return real_read_bytes(path)

        with (
            patch.object(client, "load_request", return_value=(profile, binding)),
            patch.object(client, "live_authority", return_value=None),
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
        return exit_code, captured["sanitized-result.json"], popen.call_count, process.stdin.requests


class ResumedReadClientTests(unittest.TestCase):
    def setUp(self):
        self.guards = [
            patch.object(client.subprocess, "Popen", side_effect=AssertionError("unexpected-real-process")),
            patch.object(client.subprocess, "run", side_effect=AssertionError("unexpected-subprocess-run")),
            patch.object(client.subprocess, "check_output", side_effect=AssertionError("unexpected-subprocess-check-output")),
            patch.object(client.os, "system", side_effect=AssertionError("unexpected-os-system")),
        ]
        for guard in self.guards:
            guard.start()
            self.addCleanup(guard.stop)

    def test_full_success_proves_exact_synthetic_read_and_completed_turn(self):
        exit_code, receipt, popen_count, _ = run_synthetic_session("success")
        actor = receipt["collector"]
        self.assertEqual(exit_code, 0)
        self.assertEqual(popen_count, 1)
        self.assertEqual(receipt["status"], "public-read-observed")
        self.assertEqual(actor["status"], "complete")
        self.assertEqual(actor["safePublicOutput"], SENTINEL)
        self.assertEqual(actor["finalSentinel"], SENTINEL)
        self.assertTrue(actor["commandCompletionEventObserved"])
        self.assertTrue(actor["turnCompletedEventObserved"])
        self.assertEqual(actor["commandExecutionItemCount"], 1)
        self.assertEqual(actor["safeCommandExecution"], {
            "command": "Synthetic-Read-Probe", "cwd": "C:/Synthetic/ReadProbe",
            "status": "completed", "exitCode": 0,
            "source": {"state": "present", "value": "agent"},
        })
        self.assertEqual(receipt["turnStartRequested"], True)
        self.assertEqual(receipt["turnStartResponseValidated"], True)
        self.assertEqual(receipt["threadBoundaryScope"],
                         "Unchanged startup gate only; its turnsRequested/toolsRequested fields count that gate, not the whole session.")
        self.assertEqual(receipt["threadBoundary"]["turnsRequested"], 0)
        self.assertEqual(receipt["threadBoundary"]["toolsRequested"], 0)
        self.assertEqual(receipt["startupStatusObservation"]["observed"], 0)
        self.assertEqual(receipt["taskToolRequests"], 0)
        self.assertEqual(receipt["observedNativeCommandItems"], 1)

    def test_structured_error_stops_and_interrupts_without_releasing_error_text(self):
        exit_code, receipt, _, _ = run_synthetic_session("structured-error")
        self.assertEqual(exit_code, 1)
        self.assertEqual(receipt["stopReason"], "rpc-error")
        self.assertEqual(receipt["firstProtocolFailure"]["refusalCode"], "rpc-error")
        self.assertEqual(receipt["firstProtocolFailure"]["method"],
                         {"category": "listed", "name": "error"})
        self.assertEqual(set(receipt["firstProtocolFailure"]), {
            "version", "phase", "stage", "envelopeCategory", "method", "exceptionClass", "refusalCode",
        })
        self.assertTrue(receipt["interruptSent"])
        self.assertEqual(receipt["interruptReceipt"], {"responseEnvelopeValidated": True})
        self.assertEqual(receipt["taskToolRequests"], 0)
        self.assertNotIn("synthetic structured failure", json.dumps(receipt["firstProtocolFailure"]))

    def test_foreign_thread_usage_ids_fail_closed(self):
        exit_code, receipt, _, _ = run_synthetic_session("foreign-id")
        self.assertEqual(exit_code, 1)
        self.assertEqual(receipt["collector"]["stopReason"], "thread-id-mismatch")
        self.assertEqual(receipt["firstProtocolFailure"]["refusalCode"], "unknown")
        self.assertTrue(receipt["turnStartResponseValidated"])
        self.assertTrue(receipt["interruptSent"])
        self.assertEqual(receipt["turnStartRequests"], 1)
        self.assertEqual(receipt["taskToolRequests"], 0)

    def test_early_turn_event_must_correlate_with_turn_start_response(self):
        exit_code, receipt, _, _ = run_synthetic_session("early-turn")
        self.assertEqual(exit_code, 1)
        self.assertEqual(receipt["collector"]["stopReason"], "turn-response-id-conflicts-with-pending-event")
        self.assertTrue(receipt["turnStartRequested"])
        self.assertFalse(receipt["turnStartResponseValidated"])
        self.assertFalse(receipt["interruptSent"])
        self.assertEqual(receipt["sentRpc"][-1]["method"], "turn/start")

    def test_readonly_thread_gate_rejects_conflicting_permission_before_turn(self):
        exit_code, receipt, _, _ = run_synthetic_session("readonly-gate")
        self.assertEqual(exit_code, 1)
        self.assertEqual(receipt["stopReason"], "thread-response-permission-profile-mismatch")
        self.assertEqual(receipt["turnStartRequested"], False)
        self.assertEqual(receipt["turnStartResponseValidated"], False)
        self.assertNotIn("turn/start", [entry["method"] for entry in receipt["sentRpc"]])
        self.assertEqual(receipt["threadBoundary"]["toolsRequested"], 0)

    def test_native_command_is_exact_and_second_command_is_terminal(self):
        for mode, expected_reason in (
            ("wrong-command", "command-or-cwd-mismatch"),
            ("second-command", "multiple-command-items"),
        ):
            with self.subTest(mode=mode):
                exit_code, receipt, _, _ = run_synthetic_session(mode)
                self.assertEqual(exit_code, 1)
                self.assertEqual(receipt["collector"]["stopReason"], expected_reason)
                self.assertEqual(receipt["taskToolRequests"], 0)
                self.assertEqual(receipt["observedNativeCommandItems"], 2 if mode == "second-command" else 1)
                self.assertTrue(receipt["interruptSent"])

    def test_latest_cumulative_usage_is_retained_and_optional_counter_is_not_fabricated(self):
        exit_code, receipt, _, _ = run_synthetic_session("usage-latest")
        usage = receipt["collector"]["usage"]
        raw = receipt["collector"]["rawUsageCounters"]
        self.assertEqual(exit_code, 0)
        self.assertEqual(usage["inputTokens"], 2500)
        self.assertEqual(usage["outputTokens"], 700)
        self.assertEqual(usage["totalTokens"], 3200)
        self.assertNotIn("cacheWriteInputTokens", raw["total"])
        self.assertEqual(raw["last"]["inputTokens"], 1800)
        self.assertEqual(raw["last"]["outputTokens"], 300)
        self.assertEqual(receipt["collector"]["usageOvershoot"], None)

    def test_stale_terminal_command_snapshot_does_not_replace_completed_command_proof(self):
        exit_code, receipt, _, _ = run_synthetic_session("stale-command-snapshot")
        actor = receipt["collector"]
        self.assertEqual(exit_code, 0)
        self.assertEqual(actor["status"], "complete")
        self.assertTrue(actor["commandCompletionEventObserved"])
        self.assertEqual(actor["safeCommandExecution"]["status"], "completed")
        self.assertEqual(actor["safeCommandExecution"]["exitCode"], 0)
        self.assertEqual(actor["safePublicOutput"], SENTINEL)
        self.assertEqual(actor["finalSentinel"], SENTINEL)

    def test_usage_threshold_overshoot_records_delta_and_sends_one_owned_interrupt(self):
        exit_code, receipt, _, requests = run_synthetic_session("overshoot")
        methods = [entry["method"] for entry in receipt["sentRpc"]]
        interrupt = [entry for entry in requests if entry["method"] == "turn/interrupt"]
        self.assertEqual(exit_code, 1)
        self.assertEqual(receipt["collector"]["stopReason"], "token-observation-threshold")
        self.assertEqual(receipt["collector"]["usageOvershoot"], 200)
        self.assertEqual(methods.count("turn/interrupt"), 1)
        self.assertEqual(len(interrupt), 1)
        self.assertEqual(interrupt[0]["params"], {"threadId": THREAD_ID, "turnId": TURN_ID})
        self.assertTrue(receipt["interruptSent"])
        self.assertEqual(receipt["interruptReceipt"], {"responseEnvelopeValidated": True})
        self.assertEqual(receipt["taskToolRequests"], 0)
        self.assertEqual(receipt["turnStartRequests"], 1)

    def test_first_protocol_failure_wins_while_cleanup_drains_later_frames(self):
        exit_code, receipt, _, _ = run_synthetic_session("structured-error", drain_followup=True)
        failure = receipt["firstProtocolFailure"]
        self.assertEqual(exit_code, 1)
        self.assertEqual(failure["method"], {"category": "listed", "name": "error"})
        self.assertEqual(failure["refusalCode"], "rpc-error")
        self.assertEqual(receipt["notificationCount"], 4)
        self.assertTrue(receipt["interruptReceipt"]["responseEnvelopeValidated"])
        self.assertEqual(receipt["taskToolRequests"], 0)
        self.assertFalse(receipt["rawFramePayloadsPersisted"])

    def test_notification_schema_and_strict_envelope_fail_closed(self):
        exit_code, receipt, _, _ = run_synthetic_session("bad-usage-schema")
        self.assertEqual(exit_code, 1)
        self.assertEqual(receipt["firstProtocolFailure"]["stage"], "notification-gate")
        self.assertEqual(receipt["firstProtocolFailure"]["refusalCode"], "protocol-schema-invalid")
        self.assertTrue(receipt["turnStartResponseValidated"])

        exit_code, receipt, _, _ = run_synthetic_session("missing-jsonrpc-method")
        self.assertEqual(exit_code, 1)
        self.assertEqual(receipt["firstProtocolFailure"]["stage"], "notification-envelope")
        self.assertEqual(receipt["firstProtocolFailure"]["refusalCode"], "server-request-or-notification-envelope")

    def test_attempt_one_window_and_request_freeze_binding_are_exact(self):
        grant = {
            "key": client.KEY,
            "issuedUtc": client.ISSUED,
            "status": client.GRANT_STATUS,
            "readinessWindowKey": client.READINESS_KEY,
            "notAfterUtc": client.READINESS_DEADLINE,
            "latestActualStartUtc": client.LATEST_ACTUAL_START,
            "readinessAttemptOrdinal": client.ACTUAL_ATTEMPT_ORDINAL,
        }
        slot = {
            "owner": "Scientist", "key": client.KEY, "status": client.SLOT_STATUS,
            "grantIssuedUtc": client.ISSUED, "assignedUtc": client.ISSUED,
        }
        self.assertEqual(client.validate_live_authority(copy.deepcopy(grant), slot, copy.deepcopy(grant)), client.ISSUED)
        window = {
            "key": client.READINESS_KEY,
            "authorizedUtc": "2026-10-08T20:00:00Z",
            "deadlineUtc": client.READINESS_DEADLINE,
            "maxConsideredActualReadinessAttempts": 2,
            "reservedActualAttempts": 1,
            "consumedActualAttempts": 0,
            "eachNeedsFreshGrant": True,
            "grantKeys": ["s1-resumed-read-tool-20261008-r1", client.KEY],
            "oldWindowsAndGrantsClosedUnchanged": True,
            "studyCellsAuthorized": 0,
            "closedPreparationGrants": [{
                "key": "s1-resumed-read-tool-20261008-r1", "actualStarts": 0,
                "closurePointer": "threads[name=Scientist].evidence.resumedReadToolR1Closure",
            }],
            "activeGrantKey": client.KEY,
        }
        client.validate_readiness_window(
            grant, window, datetime.fromisoformat(client.LATEST_ACTUAL_START.replace("Z", "+00:00"))
        )
        request = {
            "key": client.KEY, "grantIssuedUtc": client.ISSUED,
            "readinessWindowKey": client.READINESS_KEY, "notAfterUtc": client.READINESS_DEADLINE,
            "latestActualStartUtc": client.LATEST_ACTUAL_START,
            "readinessAttemptOrdinal": 1, "sourceCommit": "a" * 40,
            "slotAssignedUtc": client.ISSUED,
        }
        freeze = {**request, "frozenAtUtc": "2026-10-08T20:05:00Z"}
        client.validate_binding_identity(request, freeze, client.ISSUED)
        wrong_window = {**window, "consumedActualAttempts": 1}
        with self.assertRaisesRegex(ValueError, "readiness-window-allocation-mismatch"):
            client.validate_readiness_window(grant, wrong_window,
                datetime.fromisoformat(client.LATEST_ACTUAL_START.replace("Z", "+00:00")))


if __name__ == "__main__":
    unittest.main()
