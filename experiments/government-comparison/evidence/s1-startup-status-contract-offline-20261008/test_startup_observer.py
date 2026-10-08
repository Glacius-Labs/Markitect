"""Six offline cases for hypothetical, schema-validated Startup observations."""
import copy
import json
import os
from pathlib import Path
import subprocess
import sys
import unittest
from unittest.mock import patch


HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import startup_observer
from offline_wiring import OfflineProbe, load_fixture


def payload(status="starting", **fields):
    result = {"name": "synthetic-startup", "status": status}
    result.update(fields)
    return result


class StartupObserverTests(unittest.TestCase):
    def setUp(self):
        self.process_guards = [
            patch.object(subprocess, "Popen", side_effect=AssertionError("process-start-forbidden")),
            patch.object(subprocess, "run", side_effect=AssertionError("process-run-forbidden")),
            patch.object(subprocess, "check_output", side_effect=AssertionError("process-check-forbidden")),
            patch.object(os, "system", side_effect=AssertionError("system-call-forbidden")),
        ]
        for guard in self.process_guards:
            guard.start()
            self.addCleanup(guard.stop)
        self.protocol, self.gate_module, self.failure_module, self.profile, self.admission = load_fixture()

    def test_explicit_app_scope_and_exact_known_own_thread_scope_are_observation_only(self):
        observer = startup_observer.StartupObserver(self.protocol)
        app_start = observer.observe(payload("starting", threadId=None))
        app_ready = observer.observe(payload("ready", threadId=None, error=None, failureReason=None))
        own_start = observer.observe(payload("starting", threadId="synthetic-thread-01"), "synthetic-thread-01")
        own_ready = observer.observe(payload("ready", threadId="synthetic-thread-01"), "synthetic-thread-01")

        self.assertEqual(app_start["hypotheticalDisposition"], "observation-only")
        self.assertEqual(app_start["scope"], "app")
        self.assertEqual(app_ready["status"], "ready")
        self.assertEqual(app_ready["scope"], "app")
        self.assertEqual(own_start["scope"], "own-thread")
        self.assertEqual(own_ready["scope"], "own-thread")
        snapshot = observer.snapshot()
        self.assertFalse(snapshot["methodAdmitted"])
        self.assertEqual(snapshot["observed"], 4)
        self.assertEqual(snapshot["dispositionCounts"]["observation-only"], 4)

    def test_real_schema_wrong_type_is_distinct_from_schema_valid_local_extra_key(self):
        observer = startup_observer.StartupObserver(self.protocol)
        wrong_type = observer.observe({"name": "synthetic", "status": 7, "threadId": None})
        extra_key = observer.observe({
            "name": "synthetic", "status": "ready", "threadId": None,
            "localSyntheticExtra": True,
        })
        self.assertEqual(wrong_type["refusalCode"], "schema-invalid")
        self.assertEqual(wrong_type["status"], "unknown")
        self.assertEqual(extra_key["refusalCode"], "local-shape")
        self.assertEqual(extra_key["status"], "unknown")
        unknown = observer.observe(payload("synthetic-unknown", threadId=None))
        oversized = observer.observe(payload("ready", threadId=None, name="x" * 257))
        self.assertEqual(unknown["refusalCode"], "schema-invalid")
        self.assertEqual(unknown["status"], "unknown")
        self.assertEqual(oversized["refusalCode"], "local-size")
        self.assertEqual(observer.snapshot()["observed"], 4)
        self.assertNotIn("localSyntheticExtra", json.dumps(observer.snapshot()))

    def test_failed_cancelled_nonnull_error_and_failure_reason_remain_terminal(self):
        observer = startup_observer.StartupObserver(self.protocol)
        cases = (
            (payload("failed", threadId=None), "terminal-status"),
            (payload("cancelled", threadId=None), "terminal-status"),
            (payload("ready", threadId=None, error="synthetic-error-value"), "error-present"),
            (payload("starting", threadId=None, failureReason="reauthenticationRequired"),
             "failure-reason-present"),
        )
        for params, refusal in cases:
            with self.subTest(refusal=refusal):
                result = observer.observe(params)
                self.assertEqual(result["hypotheticalDisposition"], "terminal-reject")
                self.assertEqual(result["refusalCode"], refusal)
        rendered = json.dumps(observer.snapshot())
        self.assertNotIn("synthetic-error-value", rendered)
        self.assertNotIn("reauthenticationRequired", rendered)

    def test_foreign_missing_and_unresolved_thread_scopes_are_distinct_rejections(self):
        observer = startup_observer.StartupObserver(self.protocol)
        known_own_id = "synthetic-own-thread"
        foreign = observer.observe(payload("ready", threadId="synthetic-foreign-thread"), known_own_id)
        missing = observer.observe(payload("ready"), known_own_id)
        unresolved = observer.observe(payload("ready", threadId="synthetic-own-thread"), None)
        invalid_context = observer.observe(payload("ready", threadId="synthetic-own-thread"), "bad own id!")

        self.assertEqual((foreign["scope"], foreign["refusalCode"]), ("foreign-thread", "foreign-thread"))
        self.assertEqual((missing["scope"], missing["refusalCode"]), ("missing", "missing-thread-id"))
        self.assertEqual((unresolved["scope"], unresolved["refusalCode"]),
                         ("unresolved-thread", "unresolved-thread"))
        self.assertEqual((invalid_context["scope"], invalid_context["refusalCode"]),
                         ("invalid", "invalid-scope-context"))
        self.assertEqual(observer.snapshot()["scopeCounts"]["missing"], 1)

    def test_original_gate_still_rejects_exact_method_without_admitting_a_turn(self):
        self.assertNotIn(startup_observer.METHOD, self.admission["notifications"])
        self.assertEqual(len(self.failure_module.PUBLIC_METHODS), 94)

        prethread = OfflineProbe(self.profile)
        with self.assertRaisesRegex(ValueError, "lifecycle-before-thread-request"):
            prethread.notification({
                "method": startup_observer.METHOD,
                "params": payload("ready", threadId=None),
            }, phase="metadata")
        prethread_receipt = prethread.finish()
        self.assertEqual(prethread_receipt["firstProtocolFailure"]["phase"], "metadata")
        self.assertEqual(prethread_receipt["firstProtocolFailure"]["stage"], "thread-lifecycle")
        self.assertEqual(prethread_receipt["firstProtocolFailure"]["refusalCode"],
                         "lifecycle-before-thread-request")
        self.assertEqual(prethread_receipt["startupObservation"]["observed"], 1)
        self.assertEqual(prethread_receipt["turnsRequested"], 0)

        probe = OfflineProbe(self.profile)
        with self.assertRaisesRegex(ValueError, "forbidden-or-unknown-notification"):
            probe.notification({
                "method": startup_observer.METHOD,
                "params": payload("ready", threadId=None),
            })
        receipt = probe.finish()
        self.assertEqual(receipt["status"], "stopped")
        self.assertEqual(receipt["startupObservation"]["observed"], 1)
        self.assertFalse(receipt["startupObservation"]["methodAdmitted"])
        self.assertEqual(receipt["firstProtocolFailure"]["phase"], "thread")
        self.assertEqual(receipt["firstProtocolFailure"]["stage"], "notification-gate")
        self.assertEqual(receipt["firstProtocolFailure"]["refusalCode"],
                         "forbidden-or-unknown-notification")
        self.assertEqual(set(receipt["firstProtocolFailure"]), {
            "version", "phase", "stage", "envelopeCategory", "method",
            "exceptionClass", "refusalCode",
        })
        self.assertEqual(receipt["turnsRequested"], 0)
        self.assertEqual(receipt["toolsRequested"], 0)

    def test_cleanup_drain_keeps_first_failure_and_returns_only_value_free_snapshot(self):
        probe = OfflineProbe(self.profile)
        params = self.profile["threadStart"]["params"]
        own_id = "synthetic-owned-thread-canary"
        response = {
            "approvalPolicy": params["approvalPolicy"], "approvalsReviewer": "user",
            "cwd": params["cwd"], "model": params["model"], "modelProvider": params["modelProvider"],
            "sandbox": {"type": "readOnly", "networkAccess": False},
            "activePermissionProfile": {"id": ":read-only", "extends": None},
            "thread": {"cliVersion": "synthetic", "createdAt": 1, "updatedAt": 1,
                "cwd": params["cwd"], "ephemeral": True, "id": own_id,
                "model": params["model"], "modelProvider": params["modelProvider"],
                "preview": "", "projectId": None, "sessionId": "synthetic-session",
                "source": "appServer", "status": {"type": "active", "activeFlags": []},
                "turns": [], "parentThreadId": None, "forkedFromId": None},
        }
        probe.accept_response(response)
        first = {"method": startup_observer.METHOD,
                 "params": payload("ready", threadId=own_id, name="synthetic-name-canary")}
        second = {"method": startup_observer.METHOD,
                  "params": payload("failed", threadId=None, error="synthetic-error-canary")}
        probe.drain([first, second])
        receipt = probe.finish()
        self.assertEqual(receipt["stopReason"], "unexpected-final-output")
        self.assertEqual(receipt["startupObservation"]["observed"], 2)
        self.assertEqual(receipt["startupObservation"]["statusCounts"]["ready"], 1)
        self.assertTrue(receipt["threadResponseValidated"])
        self.assertEqual(receipt["startupObservation"]["firstClassification"]["scope"], "own-thread")
        self.assertEqual((receipt["turnsRequested"], receipt["toolsRequested"]), (0, 0))
        self.assertEqual(receipt["startupObservation"]["statusCounts"]["failed"], 1)
        self.assertEqual(receipt["firstProtocolFailure"]["refusalCode"],
                         "forbidden-or-unknown-notification")
        serialized = json.dumps(receipt)
        for canary in ("synthetic-name-canary", "synthetic-error-canary", own_id):
            self.assertNotIn(canary, serialized)
        snapshot = probe.observer.snapshot()
        snapshot["statusCounts"]["ready"] = 99
        self.assertEqual(probe.observer.snapshot()["statusCounts"]["ready"], 1)


if __name__ == "__main__":
    unittest.main(verbosity=2)
