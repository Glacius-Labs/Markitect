"""Synthetic fake-process tests for actual account-routing client integration."""
import copy
from datetime import datetime
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import queue
import tempfile
import types
import unittest
from unittest.mock import patch


HERE = Path(__file__).resolve().parent
REPO = HERE.parents[3]


def load_module(path, name):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


proposal = load_module(HERE / "client.py", "account_actual_client_tested")
CONTRACT = json.loads((HERE / "schema-contract.json").read_text(encoding="utf-8"))
PROFILE_SOURCE = json.loads((HERE / "profile.json").read_text(encoding="utf-8"))
ACCOUNT_METHOD = "account/updated"
THREAD_ID = "synthetic-account-thread-01"
TURN_ID = "synthetic-account-turn-01"
COMMAND_ID = "synthetic-account-command-01"
SENTINEL = "0123456789abcdef0123456789abcdef"


def make_profile(external_root):
    profile = copy.deepcopy(PROFILE_SOURCE)
    profile["cwd"] = "C:/Synthetic/AccountProbe"
    profile["externalEvidence"] = str(external_root)
    profile["argv"] = ["synthetic-app-server", "--stdio"]
    profile["command"] = "Synthetic-Account-Read"
    profile["turnPrompt"] = "Synthetic prompt for one fixed public-read result."
    profile["expectedFiles"] = {}
    for rpc in profile["rpc"]:
        if rpc.get("method") == "config/read":
            rpc["params"]["cwd"] = profile["cwd"]
    profile["threadStart"]["params"]["cwd"] = profile["cwd"]
    profile["turnStart"]["params"]["input"] = [{"type": "text", "text": profile["turnPrompt"]}]
    profile["limits"]["tokenStopThreshold"] = 10000
    return profile


def expected_value():
    return {"sentinel": SENTINEL, "fileSha256": "0" * 64}


def user_item(profile):
    return {"id": "synthetic-account-user-01", "type": "userMessage",
            "content": [{"type": "text", "text": profile["turnPrompt"]}]}


def thread_record(profile):
    params = profile["threadStart"]["params"]
    return {
        "cliVersion": "synthetic-account-test",
        "createdAt": 1,
        "cwd": profile["cwd"],
        "ephemeral": True,
        "id": THREAD_ID,
        "model": params["model"],
        "modelProvider": params["modelProvider"],
        "preview": "",
        "projectId": None,
        "sessionId": "synthetic-account-session-01",
        "source": "appServer",
        "status": {"type": "active", "activeFlags": []},
        "turns": [],
        "updatedAt": 1,
        "parentThreadId": None,
        "forkedFromId": None,
    }


def thread_response(profile):
    params = profile["threadStart"]["params"]
    return {
        "approvalPolicy": params["approvalPolicy"],
        "approvalsReviewer": "user",
        "cwd": profile["cwd"],
        "model": params["model"],
        "modelProvider": params["modelProvider"],
        "sandbox": {"type": "readOnly", "networkAccess": False},
        "activePermissionProfile": {"id": ":read-only", "extends": None},
        "thread": thread_record(profile),
    }


def command_item(profile, *, status="completed", exit_code=0, output=SENTINEL):
    return {"id": COMMAND_ID, "type": "commandExecution", "command": profile["command"],
            "commandActions": [], "cwd": profile["cwd"], "status": status,
            "source": "agent", "exitCode": exit_code, "aggregatedOutput": output}


def final_item():
    return {"id": "synthetic-account-answer-01", "type": "agentMessage",
            "text": json.dumps({"sentinel": SENTINEL}), "phase": "final_answer"}


def usage_params():
    breakdown = {"inputTokens": 100, "outputTokens": 20, "cachedInputTokens": 0,
                 "reasoningOutputTokens": 0, "totalTokens": 120}
    return {"threadId": THREAD_ID, "turnId": TURN_ID,
            "tokenUsage": {"modelContextWindow": 100000, "total": breakdown, "last": breakdown}}


def turn_response(profile):
    return {"turn": {"id": TURN_ID, "status": "inProgress", "items": [user_item(profile)]}}


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

    def push_raw(self, raw):
        self.pending.put(raw)

    def push_frame(self, frame):
        self.push_raw(json.dumps(frame, separators=(",", ":")).encode("utf-8") + b"\n")

    def close_stream(self):
        if self.pending is not None:
            self.pending.put(b"")


class FakeStdin:
    def __init__(self, process, profile, mode):
        self.process = process
        self.profile = profile
        self.mode = mode
        self.requests = []

    def _reply(self, request, result):
        self.process.stdout.push_frame({"jsonrpc": "2.0", "id": request["id"], "result": result})

    def _notify(self, method, params):
        frame = {"method": method, "params": params}
        if self.mode == "raw-oversize-account" and method == ACCOUNT_METHOD:
            raw = json.dumps(frame, separators=(",", ":")).encode("utf-8")
            padding = max(0, 4200 - len(raw) - 1)
            self.process.stdout.push_raw(raw + (b" " * padding) + b"\n")
        else:
            self.process.stdout.push_frame(frame)

    def _success_events(self):
        profile = self.profile
        self._notify("turn/started", {"threadId": THREAD_ID,
                                       "turn": {"id": TURN_ID, "status": "inProgress", "items": []}})
        self._notify("thread/tokenUsage/updated", usage_params())
        self._notify("item/started", {
            "threadId": THREAD_ID, "turnId": TURN_ID, "startedAtMs": 1,
            "item": command_item(profile, status="inProgress", exit_code=None, output=""),
        })
        self._notify("item/completed", {
            "threadId": THREAD_ID, "turnId": TURN_ID, "completedAtMs": 2,
            "item": command_item(profile),
        })
        self._notify("turn/completed", {
            "threadId": THREAD_ID,
            "turn": {"id": TURN_ID, "status": "completed", "items": [final_item()]},
        })

    def write(self, raw):
        request = json.loads(raw)
        self.requests.append(copy.deepcopy(request))
        method = request["method"]
        if method == "initialize":
            if self.mode == "account-before-thread":
                self._notify(ACCOUNT_METHOD, {"authMode": "chatgpt"})
            self._reply(request, {})
        elif method == "thread/start":
            if self.mode == "account-before-turn":
                self._notify(ACCOUNT_METHOD, {"authMode": "chatgpt", "planType": "plus"})
            self._reply(request, thread_response(self.profile))
            self._notify("thread/started", {"thread": thread_record(self.profile)})
        elif method == "turn/start":
            self._reply(request, turn_response(self.profile))
            if self.mode == "account-success":
                self._notify(ACCOUNT_METHOD, {"authMode": "chatgpt", "planType": "plus"})
                self._success_events()
            elif self.mode == "account-negative":
                self._notify(ACCOUNT_METHOD, {"authMode": "apikey"})
                self._notify("account/login/completed", {"status": "completed"})
            elif self.mode == "unknown-account-family":
                self._notify("account/login/completed", {"status": "completed"})
            elif self.mode == "raw-oversize-account":
                self._notify(ACCOUNT_METHOD, {"authMode": "chatgpt"})
            elif self.mode == "account-overcount":
                for _ in range(9):
                    self._notify(ACCOUNT_METHOD, {"authMode": "chatgpt"})
            else:
                self._success_events()
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

    def __init__(self, profile, mode):
        self.stdout = FakeStream(blocking=True)
        self.stdin = FakeStdin(self, profile, mode)
        self.stderr = FakeStream()

    def wait(self, timeout=None):
        return 0

    def poll(self):
        return 0

    def kill(self):
        return None


def run_synthetic_session(mode):
    captured = {}
    with tempfile.TemporaryDirectory(prefix="account-contract-synthetic-") as temp:
        profile = make_profile(Path(temp))
        binding = {"key": proposal.KEY, "grantIssuedUtc": proposal.ISSUED,
                   "slotAssignedUtc": proposal.ISSUED, "sourceCommit": "a" * 40,
                   "freezeSha256": "b" * 64}
        process = FakeProcess(profile, mode)
        source_files = {}
        for name in ("account_observer.py", "collector.py", "thread_gate.py",
                     "protocol_failure.py", "startup_observer.py"):
            path = HERE / name
            source_files[path.relative_to(REPO).as_posix()] = hashlib.sha256(path.read_bytes()).hexdigest()
        job_record = {"workerPid": os.getpid(), "controllerPid": os.getppid(), **binding}
        real_read_bytes = Path.read_bytes
        real_load_module = proposal.load_module

        class SelectiveProtocol:
            def __init__(self, archive, contract, protocol_type):
                self.actual = protocol_type(archive, contract)

            def validate(self, member, value):
                if member not in {
                    "v1/InitializeResponse.json",
                    "v2/ConfigReadResponse.json",
                    "v2/ConfigRequirementsReadResponse.json",
                }:
                    return self.actual.validate(member, value)

        def selected_load_module(path, expected_sha, name):
            if Path(path) == Path(proposal.PROTOCOL):
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
                return json.dumps(expected_value()).encode()
            if resolved == (Path(temp) / "job-assigned.json").resolve():
                return json.dumps(job_record).encode()
            return real_read_bytes(path)

        with (
            patch.object(proposal, "load_request", return_value=(profile, binding)),
            patch.object(proposal, "live_authority", return_value=(json.loads((HERE / "authorization-grant.json").read_text(encoding="utf-8"))["grant"], proposal.ISSUED)),
            patch.object(proposal, "sanitize", return_value={}),
            patch.object(proposal, "assess_config", return_value=("config-precheck-satisfied-awaiting-requirements", None)),
            patch.object(proposal, "final_assess", return_value=("reported-config-and-requirements-observed", None)),
            patch.object(proposal, "validate_cwd", return_value={}),
            patch.object(proposal, "exclusive_json", side_effect=lambda path, value: captured.update({Path(path).name: copy.deepcopy(value)})),
            patch.object(proposal, "load_module", side_effect=selected_load_module),
            patch.object(proposal.subprocess, "Popen", return_value=process) as popen,
            patch.object(Path, "read_bytes", safe_read_bytes),
            patch.object(proposal.sys, "stdin", types.SimpleNamespace(buffer=io.BytesIO(b"GO\n"))),
        ):
            exit_code = proposal.session()
        return exit_code, captured["sanitized-result.json"], popen.call_count, process.stdin.requests



class AccountRoutingRuntimeTests(unittest.TestCase):
    def setUp(self):
        self.guards = [
            patch.object(proposal.subprocess, "Popen", side_effect=AssertionError("unexpected-real-process")),
            patch.object(proposal.subprocess, "run", side_effect=AssertionError("unexpected-subprocess-run")),
            patch.object(proposal.subprocess, "check_output", side_effect=AssertionError("unexpected-subprocess-check-output")),
            patch.object(proposal.os, "system", side_effect=AssertionError("unexpected-os-system")),
        ]
        for guard in self.guards:
            guard.start()
            self.addCleanup(guard.stop)

    def test_live_account_route_is_telemetry_and_preserves_full_public_read(self):
        exit_code, receipt, popen_count, _ = run_synthetic_session("account-success")
        projection = receipt["accountStatusObservation"]
        self.assertEqual(exit_code, 0)
        self.assertEqual(popen_count, 1)
        self.assertEqual(receipt["status"], "public-read-observed")
        self.assertEqual(projection["observed"], 1)
        self.assertEqual(projection["dispositionCounts"]["observation-only"], 1)
        self.assertFalse(projection["ownThreadOrAccountIdentityEstablished"])
        self.assertTrue(receipt["turnStartResponseValidated"])
        self.assertTrue(receipt["collector"]["commandCompletionEventObserved"])
        self.assertTrue(receipt["collector"]["turnCompletedEventObserved"])
        self.assertEqual(receipt["collector"]["safePublicOutput"], SENTINEL)
        self.assertEqual(receipt["taskToolRequests"], 0)
        serialized = json.dumps(projection)
        self.assertNotIn("chatgpt", serialized)
        self.assertNotIn("plus", serialized)

    def test_negative_account_and_unknown_family_stop_with_fixed_diagnostics(self):
        cases = [("account-negative", 1), ("unknown-account-family", 0)]
        for mode, observed in cases:
            with self.subTest(mode=mode):
                exit_code, receipt, _, _ = run_synthetic_session(mode)
                self.assertEqual(exit_code, 1)
                self.assertEqual(receipt["status"], "stopped")
                self.assertEqual(receipt["firstProtocolFailure"]["refusalCode"],
                                 "forbidden-or-unknown-notification")
                self.assertEqual(receipt["accountStatusObservation"]["observed"], observed)
                self.assertTrue(receipt["interruptSent"])
                self.assertEqual(receipt["interruptReceipt"], {"responseEnvelopeValidated": True})
                self.assertEqual(receipt["interruptRequests"], 1)
                self.assertNotIn("apikey", json.dumps(receipt["accountStatusObservation"]))

    def test_prethread_and_prestart_account_events_remain_terminal_without_turn(self):
        for mode, phase_code in [("account-before-thread", "lifecycle-before-thread-request"),
                                 ("account-before-turn", "forbidden-or-unknown-notification")]:
            with self.subTest(mode=mode):
                exit_code, receipt, _, requests = run_synthetic_session(mode)
                self.assertEqual(exit_code, 1)
                self.assertEqual(receipt["firstProtocolFailure"]["refusalCode"], phase_code)
                self.assertEqual(receipt["accountStatusObservation"]["observed"], 0)
                self.assertNotIn("turn/start", [entry["method"] for entry in requests])
                self.assertEqual(receipt["turnStartRequests"], 0)

    def test_raw_event_size_and_eight_notice_ceiling_stop_through_real_consumer(self):
        for mode, observed, expected_reason in [
            ("raw-oversize-account", 1, "local-size"),
            ("account-overcount", 8, "observation-limit"),
        ]:
            with self.subTest(mode=mode):
                exit_code, receipt, _, _ = run_synthetic_session(mode)
                self.assertEqual(exit_code, 1)
                snapshot = receipt["accountStatusObservation"]
                self.assertEqual(snapshot["observed"], observed)
                if expected_reason == "local-size":
                    self.assertEqual(snapshot["refusalCounts"][expected_reason], 1)
                else:
                    self.assertTrue(snapshot["limitReached"])
                self.assertEqual(receipt["taskToolRequests"], 0)

    def test_stop_cleanup_drains_and_owns_one_validated_interrupt(self):
        exit_code, receipt, _, requests = run_synthetic_session("account-negative")
        self.assertEqual(exit_code, 1)
        self.assertEqual(receipt["stopReason"], "forbidden-or-unknown-notification")
        self.assertEqual(receipt["firstProtocolFailure"]["method"],
                         {"category": "listed", "name": "account/updated"})
        self.assertTrue(receipt["turnStartResponseValidated"])
        self.assertTrue(receipt["interruptSent"])
        self.assertEqual(receipt["interruptReceipt"], {"responseEnvelopeValidated": True})
        self.assertEqual(sum(entry["method"] == "turn/interrupt" for entry in requests), 1)
        self.assertEqual(receipt["nativeReturnCode"], 0)
        self.assertFalse(receipt["rawFramePayloadsPersisted"])
        self.assertFalse(receipt["rawStderrPersisted"])

    def test_final_grant_window_slot_and_freeze_identity_are_bound(self):
        authority = json.loads((HERE / "authorization-grant.json").read_text(encoding="utf-8"))
        grant = authority["grant"]
        slot = {"owner": "Scientist", "key": proposal.KEY, "status": proposal.SLOT_STATUS,
                "grantIssuedUtc": proposal.ISSUED, "assignedUtc": proposal.ISSUED}
        self.assertEqual(proposal.validate_live_authority(grant, slot, copy.deepcopy(grant)), proposal.ISSUED)
        with self.assertRaisesRegex(ValueError, "invalid-slot-activation-time"):
            proposal.validate_live_authority(grant, {**slot, "releasedUtc": proposal.ISSUED}, grant)
        self.assertEqual(grant["readinessAttemptOrdinal"], 2)
        self.assertIs(grant["finalAttemptInResumedBlock"], True)
        window = authority["readinessWindowBinding"]
        now = datetime.fromisoformat("2026-10-08T21:50:00+00:00")
        proposal.validate_readiness_window(grant, window, now=now)
        self.assertEqual(proposal.readiness_binding(window), window)
        with self.assertRaisesRegex(ValueError, "readiness-window-allocation-mismatch"):
            proposal.validate_readiness_window(grant, {**window, "consumedActualAttempts": 2}, now=now)
        self.assertEqual(window["grantKeys"][-1], proposal.KEY)
        self.assertEqual(window["remainingConsideredActualAttempts"], 1)
        request = {"key": proposal.KEY, "grantIssuedUtc": proposal.ISSUED,
                   "slotAssignedUtc": proposal.ISSUED, "readinessWindowKey": proposal.READINESS_KEY,
                   "notAfterUtc": proposal.READINESS_DEADLINE,
                   "latestActualStartUtc": proposal.LATEST_ACTUAL_START,
                   "readinessAttemptOrdinal": 2, "sourceCommit": "a" * 40}
        freeze = {**request, "frozenAtUtc": "2026-10-08T20:52:00Z"}
        proposal.validate_binding_identity(request, freeze, proposal.ISSUED)
        bad = {**freeze, "readinessAttemptOrdinal": True}
        with self.assertRaisesRegex(ValueError, "readiness-freeze-binding-mismatch"):
            proposal.validate_binding_identity(request, bad, proposal.ISSUED)
        self.assertEqual(grant["maxNewAppServerTrees"], 1)
        self.assertEqual(grant["maxNewActorReservations"], 1)
        self.assertEqual(grant["maxTurnStarts"], 1)
        self.assertEqual(grant["maxObservedTaskToolItems"], 1)
        self.assertEqual(grant["maxTurnInterruptsOnStop"], 1)
        self.assertEqual(grant["previousAppServerTrees"], 14)
        self.assertEqual(grant["cumulativeAppServerTreesMaximum"], 15)
        self.assertEqual(grant["previousActorReservations"], 7)
        self.assertEqual(grant["cumulativeActorReservationsMaximum"], 8)


if __name__ == "__main__":
    unittest.main()
