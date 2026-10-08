"""Offline R6 bridge and completion-receipt checks; no experimental process starts."""
from __future__ import annotations

import ast
import hashlib
import json
from pathlib import Path
import sqlite3
import tempfile
import unittest
from contextlib import contextmanager
from unittest.mock import patch

import dispatch
import government_native_profile
import government_roles
import native_controller
import native_fixture_budget


def sha(raw: bytes) -> str:
    return hashlib.sha256(raw).hexdigest()


class R6BridgeTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.profile = government_native_profile.profile("r6")
        self.executable = self.root / "government.exe"
        self.executable.write_bytes(b"pinned-government-binary")
        self.interpreter = self.root / "python.exe"
        self.interpreter.write_bytes(b"pinned-interpreter")
        self.delegate = self.root / "deterministic_delegate.py"
        self.delegate.write_bytes(b"pinned-deterministic-delegate")
        runtime_files = [
            {"path": str(self.interpreter), "mode": "0644",
             "digest": "sha256:" + sha(self.interpreter.read_bytes())},
            {"path": str(self.delegate), "mode": "0644",
             "digest": "sha256:" + sha(self.delegate.read_bytes())},
        ]
        auth = {"apiVersion": "test-role-auth", "slots": [
            {"slotId": "execute", "phase": "execute", "delegate": {
                "command": str(self.interpreter),
                "commandDigest": "sha256:" + sha(self.interpreter.read_bytes()),
                "argv": [str(self.interpreter), str(self.delegate), "--phase", "execute"],
                "runtimeFiles": runtime_files}},
            {"slotId": "review", "phase": "review", "delegate": {
                "command": str(self.interpreter),
                "commandDigest": "sha256:" + sha(self.interpreter.read_bytes()),
                "argv": [str(self.interpreter), str(self.delegate), "--phase", "review"],
                "runtimeFiles": runtime_files}},
        ]}
        self.auth_raw = json.dumps(auth, sort_keys=True).encode()
        self.auth_path = self.root / "role-auth.json"
        self.auth_path.write_bytes(self.auth_raw)
        self.request = {
            "mode": "mechanical", "dispatchId": self.profile.dispatch_id,
            "arm": "government", "task": {"id": self.profile.task_id},
            "nativeFixtureGrant": {"path": str(self.root / "r1.json"), "sha256": "0" * 64,
                                   "sourceKey": native_controller.NATIVE_FIXTURE_GRANT_KEY},
            self.profile.marker: {"path": str(self.root / "r6.json"), "sha256": "1" * 64,
                                  "sourceKey": self.profile.key},
            "releasedInputs": [{"path": str(self.auth_path), "sha256": sha(self.auth_raw)}],
            "product": {"government": {
                "executable": {"path": str(self.executable), "sha256": sha(self.executable.read_bytes())},
                "roleAuthorization": {"path": str(self.auth_path), "sha256": sha(self.auth_raw)}}}}
        self.captured = {str(self.auth_path.resolve()): self.auth_raw}
        self.validated = {
            "grantKey": self.profile.key, "profileName": self.profile.name,
            "dispatchId": self.profile.dispatch_id,
            "grant": {"key": self.profile.key,
                      "product": {"delegateSha256": sha(self.delegate.read_bytes())},
                      "python": {"path": str(self.interpreter),
                                 "sha256": sha(self.interpreter.read_bytes())}},
            "product": {"name": "Government"}}

    def tearDown(self):
        self.temp.cleanup()

    def test_r6_profile_is_exact_and_rejects_cross_profile_bindings(self):
        self.assertEqual(native_controller.native_profile(self.request), self.profile)
        for changed in (
                dict(self.request, dispatchId=government_native_profile.profile("r5").dispatch_id),
                dict(self.request, nativeFixtureR5Grant={"wrong": True}),
                dict(self.request, nativeFixtureR4Grant={"wrong": True})):
            with self.assertRaisesRegex(ValueError, "profile|Request profile"):
                native_controller.native_profile(changed)

    def _validate(self, request=None, captured=None, argv=None):
        with patch.object(native_controller, "validate_native_fixture_grant",
                          return_value={"r6Grant": self.validated}) as bind, \
             patch.object(native_fixture_budget, "validate_r6_entry_gate",
                          return_value={"slotOwner": "Scientist", "slotKey": self.profile.key}) as gate:
            outcome = native_controller.validate_r6_entry_for_launch(
                self.request if request is None else request,
                self.captured if captured is None else captured,
                [str(self.executable), "queue"] if argv is None else argv)
            return outcome, bind, gate

    def test_r6_entry_binds_request_binary_python_delegate_and_live_slot(self):
        outcome, bind, gate = self._validate()
        bind.assert_called_once_with(self.request, self.captured)
        gate.assert_called_once_with(self.validated)
        self.assertEqual(outcome["entryGate"]["slotKey"], self.profile.key)
        self.assertEqual(outcome["delegatePaths"], [str(self.delegate.resolve())])

    def test_missing_or_changed_authorization_and_binary_fail_before_live_gate(self):
        with patch.object(native_controller, "validate_native_fixture_grant",
                          return_value={"r6Grant": self.validated}), \
             patch.object(native_fixture_budget, "validate_r6_entry_gate") as gate:
            with self.assertRaisesRegex(ValueError, "captured Request input"):
                native_controller.validate_r6_entry_for_launch(
                    self.request, {}, [str(self.executable), "queue"])
            gate.assert_not_called()
        other = self.root / "other.exe"
        other.write_bytes(b"other")
        with patch.object(native_controller, "validate_native_fixture_grant",
                          return_value={"r6Grant": self.validated}), \
             patch.object(native_fixture_budget, "validate_r6_entry_gate") as gate:
            with self.assertRaisesRegex(ValueError, "Request-bound Government executable"):
                native_controller.validate_r6_entry_for_launch(
                    self.request, self.captured, [str(other), "queue"])
            gate.assert_not_called()

    def test_python_and_delegate_mutation_fail_closed(self):
        self.delegate.write_bytes(b"mutated delegate")
        with self.assertRaisesRegex(ValueError, "delegate script differs"):
            native_controller.validate_r6_delegate_authorization(
                self.request, self.auth_raw, self.validated)
        self.delegate.write_bytes(b"pinned-deterministic-delegate")
        self.interpreter.write_bytes(b"mutated Python")
        with self.assertRaisesRegex(ValueError, "Python executable differs"):
            native_controller.validate_r6_delegate_authorization(
                self.request, self.auth_raw, self.validated)

    def test_r6_launch_uses_exact_live_gate(self):
        with patch.object(native_controller, "validate_native_fixture_grant",
                          return_value={"r6Grant": self.validated}), \
             patch.object(native_fixture_budget, "validate_r6_entry_gate",
                          side_effect=ValueError("R6 slot revoked")) as gate:
            with self.assertRaisesRegex(ValueError, "revoked"):
                native_controller.validate_r6_entry_for_launch(
                    self.request, self.captured, [str(self.executable), "queue"])
            gate.assert_called_once_with(self.validated)

    def test_role_gates_precede_reservation_and_delegate_launch(self):
        tree = ast.parse(Path(government_roles.__file__).read_text(encoding="utf-8"))
        run_role = next(node for node in tree.body
                        if isinstance(node, ast.FunctionDef) and node.name == "run_role")
        reserve_line = next(node.lineno for node in ast.walk(run_role)
                            if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and
                            node.func.id == "_reserve")
        bounded_line = next(node.lineno for node in ast.walk(run_role)
                            if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and
                            node.func.id == "bounded")
        gate_lines = [node.lineno for node in ast.walk(run_role)
                      if isinstance(node, ast.Call) and isinstance(node.func, ast.Attribute) and
                      node.func.attr == "validate_profile_live_gate"]
        self.assertEqual(len(gate_lines), 2)
        self.assertLess(gate_lines[0], reserve_line)
        self.assertLess(gate_lines[1], bounded_line)


class R6CompletionReceiptTests(unittest.TestCase):
    @staticmethod
    def _dispatch_id():
        return government_native_profile.profile("r6").dispatch_id

    def test_positive_completion_requires_all_terminal_bindings_and_deadline(self):
        args = ("completed", "finished", "completed", "a" * 64, True, True, True, True, 38)
        self.assertEqual(dispatch.r6_completion_status(*args), "completed")
        for invalid in (
                (*args[:-1], 38.001), (*args[:3], None, *args[4:]),
                (*args[:4], False, *args[5:]), (*args[:5], False, *args[6:]),
                (*args[:6], False, *args[7:])):
            self.assertEqual(dispatch.r6_completion_status(*invalid), "incomplete")

    def test_completion_receipt_is_written_after_terminal_readback_and_measurement(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            result_path = root / "result.json"
            result = {"dispatchId": self._dispatch_id(), "status": "completed"}
            request = {"dispatchId": result["dispatchId"]}
            request_raw = json.dumps(request).encode()
            result_raw = json.dumps(result, sort_keys=True, indent=2).encode()
            result_path.write_bytes(result_raw)
            process_path = root / "process.json"
            process_raw = b"{\"status\":\"completed\"}"
            process_path.write_bytes(process_raw)
            terminal_receipt = '{"status": "completed"}'
            receipt_sha = sha(terminal_receipt.encode())
            conn = sqlite3.connect(":memory:")
            conn.execute("CREATE TABLE controller_runs(dispatch_id TEXT,status TEXT,receipt_sha256 TEXT,end REAL,process_receipt TEXT)")
            conn.execute("INSERT INTO controller_runs VALUES(?,?,?,?,?)",
                         (result["dispatchId"], "completed", receipt_sha, 10.0, terminal_receipt))

            class FakeLedger:
                def dispatch_record(self, dispatch_id):
                    return {"phase": "finished", "result": json.dumps(result),
                            "request": request_raw, "execution_sha": dispatch.execution_sha(request)}

                @contextmanager
                def transaction(self):
                    yield conn

                def controller_elapsed(self, dispatch_id):
                    return 12.5

            timing = dispatch.write_r6_completion_receipt(
                request, request_raw, result_path, root, FakeLedger(),
                process_path, process_raw, True)
            self.assertEqual(timing["status"], "completed")
            self.assertEqual(timing["elapsedSecondsAfterTerminalReceiptAvailable"], 12.5)
            self.assertEqual(timing["terminalResult"]["sha256"], sha(result_raw))
            self.assertEqual(timing["terminalControllerReceiptSha256"], receipt_sha)
            self.assertEqual(json.loads((root / "controller-completion.json").read_bytes()), timing)
            conn.close()

    def test_dispatch_writes_completion_timing_only_after_terminal_result_write(self):
        tree = ast.parse(Path(dispatch.__file__).read_text(encoding="utf-8"))
        dispatch_fn = next(node for node in tree.body
                           if isinstance(node, ast.FunctionDef) and node.name == "dispatch_government")
        calls = [(node.lineno, node.func.id) for node in ast.walk(dispatch_fn)
                 if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and
                 node.func.id in {"write_result", "write_r6_completion_receipt"}]
        terminal_writes = [line for line, name in calls if name == "write_result"]
        completion_calls = [line for line, name in calls if name == "write_r6_completion_receipt"]
        self.assertTrue(terminal_writes)
        self.assertEqual(len(completion_calls), 1)
        self.assertLess(max(terminal_writes), completion_calls[0])

    def test_r6_controller_start_gate_and_cleanup_budget_precede_popen(self):
        self.assertEqual(dispatch.r5_process_timeout(38, 0), 32)
        self.assertLessEqual(dispatch.r5_process_timeout(38, 5), 27)
        self.assertLessEqual(dispatch.r5_process_timeout(20, 32), 0)
        tree = ast.parse(Path(dispatch.__file__).read_text(encoding="utf-8"))
        dispatch_fn = next(node for node in tree.body
                           if isinstance(node, ast.FunctionDef) and node.name == "dispatch_government")
        gate_line = next(node.lineno for node in ast.walk(dispatch_fn)
                         if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and
                         node.func.id == "native_fixture_start_gate")
        timeout_line = next(node.lineno for node in ast.walk(dispatch_fn)
                            if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and
                            node.func.id == "r5_process_timeout")
        reserve_line = next(node.lineno for node in ast.walk(dispatch_fn)
                            if isinstance(node, ast.Call) and isinstance(node.func, ast.Attribute) and
                            node.func.attr == "reserve_controller_dispatch")
        bounded_line = next(node.lineno for node in ast.walk(dispatch_fn)
                            if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and
                            node.func.id == "bounded")
        self.assertLess(gate_line, reserve_line)
        self.assertLess(gate_line, bounded_line)
        self.assertLess(timeout_line, bounded_line)

if __name__ == "__main__":
    unittest.main()
