"""Durable lifecycle/admission fixtures with injected offline backends only."""
from datetime import datetime, timedelta, timezone
import hashlib
import json
from pathlib import Path
import sys
import tempfile
import threading
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from conventional.service import Service


def write(path, data):
    path.write_bytes((json.dumps(data, indent=2) + "\n").encode("utf-8"))


class ServiceTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        root = Path(self.temp.name)
        self.repo, self.audit = root / "repo", root / "audit"
        self.repo.mkdir()
        self.audit.mkdir()
        (self.repo / ".study").mkdir()
        (self.repo / ".study/run-id").write_bytes(b"offline-fixture-run\n")
        write(self.repo / ".study/station.json", {"station": "S1", "fixtureOnly": True})
        write(self.audit / "run.json", {"method": "Conventional", "runId": "offline-fixture-run",
                                      "repoPath": str(self.repo), "auditPath": str(self.audit)})
        self.path, self.order_path = root / "config.json", root / "order.json"
        exe = str(Path(sys.executable).resolve())
        self.config = {"schema": 1, "execution_authorized": True,
                       "actualOrderPath": str(self.order_path), "backend": "codex-cli",
                       "command": [exe], "filePins": {exe: hashlib.sha256(Path(exe).read_bytes()).hexdigest()},
                       "cwd": str(self.repo), "audit": str(self.audit), "model": "offline-no-model",
                       "effort": "high", "timeoutSeconds": 5400, "requestTimeoutSeconds": 600,
                       "runtimeBinding": {"fixtureOnly": True},
                       "runtimeOptions": {"sandbox": "workspace-write", "approvalPolicy": "never",
                                          "memoryEnabled": False, "nativeHelperModel": "offline-no-model",
                                          "nativeHelperEffort": "high"}}
        source = Path(__file__).resolve().parents[1]
        for relative in ("conventional/service.py", "conventional/backends.py", "conventional/mcp.py", "conventional_wrapper.py"):
            path = source / relative
            self.config["filePins"][str(path)] = hashlib.sha256(path.read_bytes()).hexdigest()
        self.calls = []
        self.persist()

    def persist(self, *, limit=4, expires=None):
        write(self.path, self.config)
        self.order = {"execution_authorized": True, "actualOrderRef": "OFFLINE-FIXTURE-NOT-A-GRANT",
                      "configSha256": hashlib.sha256(self.path.read_bytes()).hexdigest(),
                      "runId": "offline-fixture-run", "maxTurns": limit,
                      "expiresAt": expires or (datetime.now(timezone.utc) + timedelta(hours=2)).isoformat()}
        write(self.order_path, self.order)

    def backend(self, spec, prompt, resume_id, emit, cancel):
        self.calls.append((spec, prompt, resume_id))
        emit({"kind": "offline", "text": prompt})
        return {"state": "completed", "nativeSessionId": resume_id or "fixture-session",
                "nativeTurnId": "fixture-turn-" + str(len(self.calls)), "usage": None,
                "detail": "scripted lifecycle terminal only"}

    def service(self, runner=None):
        service = Service(self.path, backend_runner=runner or self.backend)
        self.addCleanup(service.close)
        return service

    def test_disabled_start_never_dispatches(self):
        self.config["execution_authorized"] = False
        self.persist()
        with self.assertRaisesRegex(ValueError, "execution disabled"):
            self.service().start("ordinary request")
        self.assertEqual([], self.calls)
        self.assertFalse((self.audit / "conventional-execution").exists())

    def test_resume_uses_owned_native_session_and_same_prompt(self):
        service = self.service()
        first = service.start("first ordinary backlog")
        first = service.wait(first["runId"])
        second = service.resume(first["runId"], "next released work")
        second = service.wait(second["runId"])
        self.assertEqual("fixture-session", self.calls[1][2])
        self.assertEqual("next released work", self.calls[1][1])
        self.assertEqual("NOT RUN", second["taskAssessment"])
        self.assertEqual(first["runId"], second["parentRunId"])
        self.assertIsNone(second["usage"])
        self.assertEqual("not established", second["humanAcceptance"])
        self.assertEqual({"fixtureOnly": True}, second["runtimeBinding"])
        self.assertEqual(self.config["runtimeOptions"], second["runtimeOptions"])
        with self.assertRaisesRegex(ValueError, "already continued"):
            service.resume(first["runId"], "replay forbidden")
        folder = self.audit / "conventional-execution" / second["runId"]
        request = json.loads((folder / "request.json").read_bytes())
        self.assertEqual("next released work", request["prompt"])
        self.assertEqual(self.config["runtimeOptions"], request["runtimeOptions"])
        self.assertEqual(self.config["runtimeOptions"],
                         json.loads((folder / "result.json").read_bytes())["runtimeOptions"])
        self.assertEqual(1, json.loads((folder / "events.jsonl").read_bytes())["sequence"])

    def test_runtime_options_are_validated_before_dispatch(self):
        invalid = [
            {"sandbox": "danger-full-access", "approvalPolicy": "never", "memoryEnabled": False},
            {"sandbox": "workspace-write", "approvalPolicy": "never", "memoryEnabled": "false"},
            {"sandbox": "workspace-write", "approvalPolicy": "never", "memoryEnabled": False,
             "permissionBypass": True},
            {"sandbox": "workspace-write", "approvalPolicy": "never", "memoryEnabled": False,
             "nativeHelperModel": "other-model"},
        ]
        for options in invalid:
            with self.subTest(options=options):
                self.config["runtimeOptions"] = options
                self.persist()
                with self.assertRaises(ValueError):
                    self.service().start("work")
        self.assertEqual([], self.calls)

    def test_legacy_configuration_without_runtime_options_remains_valid(self):
        self.config.pop("runtimeOptions")
        self.persist()
        service = self.service()
        result = service.wait(service.start("legacy config")["runId"])
        self.assertEqual("completed", result["state"])
        self.assertIsNone(result["runtimeOptions"])
        self.assertIsNone(self.calls[0][0]["runtimeOptions"])

    def test_unknown_dispatch_blocks_replay_and_fresh_starts(self):
        def unknown(*args):
            raise RuntimeError("controller transport lost after intent")
        service = self.service(unknown)
        result = service.wait(service.start("work")["runId"])
        self.assertEqual("uncertain", result["state"])
        self.assertTrue((self.audit / "conventional-execution/active.json").exists())
        with self.assertRaisesRegex(ValueError, "uncertain dispatch"):
            service.resume(result["runId"], "retry")
        with self.assertRaisesRegex(ValueError, "unresolved owner"):
            service.start("new work")

    def test_finite_outer_turn_count_includes_failed_attempt(self):
        self.persist(limit=1)
        def failed(*args):
            self.calls.append("failed native fixture")
            return {"state": "failed", "nativeSessionId": "fixture-session"}
        service = self.service(failed)
        service.wait(service.start("work")["runId"])
        with self.assertRaisesRegex(ValueError, "allowance exhausted"):
            service.start("extra")
        self.assertEqual(1, len(self.calls))

    def test_worker_start_failure_preserves_handle_and_never_dispatches(self):
        service = self.service()
        with patch("conventional.service.threading.Thread.start", side_effect=RuntimeError("offline exhaustion")):
            result = service.start("work")
        self.assertEqual("failed", result["state"])
        self.assertIn("worker not dispatched", result["detail"])
        self.assertEqual("failed", service.wait(result["runId"])["state"])
        self.assertEqual([], self.calls)
        self.assertFalse((self.audit / "conventional-execution/active.json").exists())
        folder = self.audit / "conventional-execution" / result["runId"]
        self.assertIs(False, json.loads((folder / "result.json").read_bytes())["dispatchOccurred"])

    def test_resume_worker_failure_does_not_consume_native_parent_continuation(self):
        service = self.service()
        parent = service.wait(service.start("first")["runId"])
        with patch("conventional.service.threading.Thread.start", side_effect=RuntimeError("offline exhaustion")):
            failed = service.resume(parent["runId"], "not delivered")
        self.assertEqual("failed", service.wait(failed["runId"])["state"])
        folder = self.audit / "conventional-execution" / failed["runId"]
        self.assertTrue((folder / "resume-not-dispatched.json").is_file())
        resumed = service.wait(service.resume(parent["runId"], "delivered")["runId"])
        self.assertEqual("completed", resumed["state"])
        self.assertEqual(2, len(self.calls))
        self.assertEqual("fixture-session", self.calls[1][2])

    def test_config_order_binary_and_run_binding_reject_before_dispatch(self):
        self.order["configSha256"] = "wrong"
        write(self.order_path, self.order)
        with self.assertRaisesRegex(ValueError, "bind configuration"):
            self.service().start("work")
        self.persist()
        self.config["filePins"][self.config["command"][0]] = "wrong"
        self.persist()
        with self.assertRaisesRegex(ValueError, "file pin changed"):
            self.service().start("work")
        self.assertEqual([], self.calls)

    def test_expired_order_and_final_freeze_reject_before_dispatch(self):
        self.persist(expires=(datetime.now(timezone.utc) - timedelta(seconds=1)).isoformat())
        with self.assertRaisesRegex(ValueError, "expired"):
            self.service().start("work")
        self.persist()
        (self.audit / "final-freeze").mkdir()
        with self.assertRaisesRegex(ValueError, "frozen"):
            self.service().start("work")
        self.assertEqual([], self.calls)

    def test_other_controller_cancel_is_observed_even_without_cli_wait(self):
        entered = threading.Event()
        def waiting(spec, prompt, resume_id, emit, cancel):
            entered.set()
            if not cancel.wait(5):
                raise RuntimeError("offline cancellation not delivered")
            return {"state": "cancelled", "nativeSessionId": "fixture-session", "usage": None}
        service = self.service(waiting)
        first = service.start("work")
        self.assertTrue(entered.wait(5))
        other = self.service()
        reply = other.cancel(first["runId"])
        self.assertTrue(reply["cancelRequested"])
        self.assertEqual("unverified_in_this_process", reply["controllerOwnership"])
        result = service.wait(first["runId"])
        self.assertEqual("cancelled", result["state"])
        self.assertFalse((self.audit / "conventional-execution/active.json").exists())

    def test_second_controller_cannot_start_over_active_owner(self):
        entered = threading.Event()
        def waiting(spec, prompt, resume_id, emit, cancel):
            entered.set()
            cancel.wait(5)
            return {"state": "cancelled", "nativeSessionId": "fixture-session"}
        service = self.service(waiting)
        first = service.start("work")
        self.assertTrue(entered.wait(5))
        with self.assertRaisesRegex(ValueError, "unresolved owner"):
            self.service().start("overlap")
        service.cancel(first["runId"])
        service.wait(first["runId"])


if __name__ == "__main__":
    unittest.main()
