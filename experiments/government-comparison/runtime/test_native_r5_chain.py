"""Offline R5 launch admission regressions; no product, wrapper, or delegate starts."""
from __future__ import annotations

import ast
import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import dispatch
import government_roles
import native_controller
import native_fixture_budget


def sha(raw: bytes) -> str:
    return hashlib.sha256(raw).hexdigest()


class R5LaunchAdmissionTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
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
            "mode": "mechanical",
            "dispatchId": native_controller.R5_DISPATCH_ID,
            "arm": "government",
            "nativeFixtureGrant": {"path": str(self.root / "r1.json"), "sha256": "0" * 64,
                                   "sourceKey": native_controller.NATIVE_FIXTURE_GRANT_KEY},
            "nativeFixtureR5Grant": {"path": str(self.root / "r5.json"), "sha256": "1" * 64,
                                     "sourceKey": native_controller.R5_GRANT_KEY},
            "releasedInputs": [{"path": str(self.auth_path), "sha256": sha(self.auth_raw)}],
            "product": {"government": {
                "executable": {"path": str(self.executable), "sha256": sha(self.executable.read_bytes())},
                "roleAuthorization": {"path": str(self.auth_path), "sha256": sha(self.auth_raw)}}}}
        self.captured = {str(self.auth_path.resolve()): self.auth_raw}
        self.validated = {
            "grantKey": native_controller.R5_GRANT_KEY,
            "grant": {"key": native_controller.R5_GRANT_KEY,
                      "product": {"delegateSha256": sha(self.delegate.read_bytes())},
                      "python": {"path": str(self.interpreter),
                                 "sha256": sha(self.interpreter.read_bytes())}},
            "product": {"name": "Government"}}

    def tearDown(self):
        self.temp.cleanup()

    def _validate(self, request=None, captured=None, argv=None, gate=None):
        with patch.object(native_controller, "validate_native_fixture_grant",
                          return_value={"r5Grant": self.validated}) as bind, \
             patch.object(native_fixture_budget, "validate_r5_entry_gate",
                          return_value={"slotOwner": "Scientist", "slotKey": native_controller.R5_GRANT_KEY}
                          if gate is None else gate) as entry_gate:
            outcome = native_controller.validate_r5_entry_for_launch(
                self.request if request is None else request,
                self.captured if captured is None else captured,
                [str(self.executable), "queue"] if argv is None else argv)
            return outcome, bind, entry_gate

    def test_exact_request_binary_delegate_and_live_gate_pass(self):
        outcome, bind, gate = self._validate()
        bind.assert_called_once_with(self.request, self.captured)
        gate.assert_called_once_with(self.validated)
        self.assertEqual(outcome["entryGate"]["slotKey"], native_controller.R5_GRANT_KEY)
        self.assertEqual(outcome["delegatePaths"], [str(self.delegate.resolve())])

    def test_earlier_grant_combination_and_wrong_dispatch_fail_before_binding(self):
        request = dict(self.request, nativeFixtureR4Grant={"unexpected": True})
        with patch.object(native_controller, "validate_native_fixture_grant") as bind:
            with self.assertRaisesRegex(ValueError, "Government-only additive grant"):
                native_controller.validate_r5_entry_for_launch(
                    request, self.captured, [str(self.executable), "queue"])
            bind.assert_not_called()
        request = dict(self.request, dispatchId=native_controller.R4_DISPATCH_ID)
        with patch.object(native_controller, "validate_native_fixture_grant") as bind:
            with self.assertRaisesRegex(ValueError, "Government-only additive grant"):
                native_controller.validate_r5_entry_for_launch(
                    request, self.captured, [str(self.executable), "queue"])
            bind.assert_not_called()

    def test_native_executable_must_match_request_pin(self):
        other = self.root / "other.exe"
        other.write_bytes(b"different")
        with patch.object(native_controller, "validate_native_fixture_grant",
                          return_value={"r5Grant": self.validated}), \
             patch.object(native_fixture_budget, "validate_r5_entry_gate") as gate:
            with self.assertRaisesRegex(ValueError, "Request-bound Government executable"):
                native_controller.validate_r5_entry_for_launch(
                    self.request, self.captured, [str(other), "queue"])
            gate.assert_not_called()

    def test_role_authorization_must_be_captured_and_current(self):
        with patch.object(native_controller, "validate_native_fixture_grant",
                          return_value={"r5Grant": self.validated}), \
             patch.object(native_fixture_budget, "validate_r5_entry_gate") as gate:
            with self.assertRaisesRegex(ValueError, "exact captured Request input"):
                native_controller.validate_r5_entry_for_launch(
                    self.request, {}, [str(self.executable), "queue"])
            gate.assert_not_called()
        self.auth_path.write_bytes(self.auth_raw + b" ")
        with patch.object(native_controller, "validate_native_fixture_grant",
                          return_value={"r5Grant": self.validated}), \
             patch.object(native_fixture_budget, "validate_r5_entry_gate") as gate:
            with self.assertRaisesRegex(ValueError, "current Request-bound file"):
                native_controller.validate_r5_entry_for_launch(
                    self.request, self.captured, [str(self.executable), "queue"])
            gate.assert_not_called()

    def test_interpreter_and_delegate_pins_are_rechecked(self):
        self.delegate.write_bytes(b"changed delegate")
        with patch.object(native_controller, "validate_native_fixture_grant",
                          return_value={"r5Grant": self.validated}), \
             patch.object(native_fixture_budget, "validate_r5_entry_gate") as gate:
            with self.assertRaisesRegex(ValueError, "delegate script differs"):
                native_controller.validate_r5_entry_for_launch(
                    self.request, self.captured, [str(self.executable), "queue"])
            gate.assert_not_called()
        self.delegate.write_bytes(b"pinned-deterministic-delegate")
        self.interpreter.write_bytes(b"changed interpreter")
        with patch.object(native_controller, "validate_native_fixture_grant",
                          return_value={"r5Grant": self.validated}), \
             patch.object(native_fixture_budget, "validate_r5_entry_gate") as gate:
            with self.assertRaisesRegex(ValueError, "Python executable differs"):
                native_controller.validate_r5_entry_for_launch(
                    self.request, self.captured, [str(self.executable), "queue"])
            gate.assert_not_called()

    def test_live_slot_revocation_stops_at_launch_gate(self):
        with patch.object(native_controller, "validate_native_fixture_grant",
                          return_value={"r5Grant": self.validated}), \
             patch.object(native_fixture_budget, "validate_r5_entry_gate",
                          side_effect=ValueError("R5 grant revoked")) as gate:
            with self.assertRaisesRegex(ValueError, "revoked"):
                native_controller.validate_r5_entry_for_launch(
                    self.request, self.captured, [str(self.executable), "queue"])
            gate.assert_called_once_with(self.validated)

    def test_role_and_native_process_gates_precede_effect_boundaries(self):
        roles = ast.parse(Path(government_roles.__file__).read_text(encoding="utf-8"))
        run_role = next(node for node in roles.body
                        if isinstance(node, ast.FunctionDef) and node.name == "run_role")
        reserve_line = next(node.lineno for node in ast.walk(run_role)
                            if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and
                            node.func.id == "_reserve")
        bounded_line = next(node.lineno for node in ast.walk(run_role)
                            if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and
                            node.func.id == "bounded")
        r5_gate_lines = [node.lineno for node in ast.walk(run_role)
                         if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and
                         node.func.id == "validate_r5_entry_gate"]
        self.assertEqual(len(r5_gate_lines), 2)
        self.assertLess(r5_gate_lines[0], reserve_line)
        self.assertLess(r5_gate_lines[1], bounded_line)

        dispatch_tree = ast.parse(Path(dispatch.__file__).read_text(encoding="utf-8"))
        dispatch_fn = next(node for node in dispatch_tree.body
                           if isinstance(node, ast.FunctionDef) and node.name == "dispatch_government")
        gate_calls = [node.lineno for node in ast.walk(dispatch_fn)
                      if isinstance(node, ast.Call) and isinstance(node.func, ast.Attribute) and
                      node.func.attr == "validate_r5_entry_for_launch"]
        self.assertEqual(len(gate_calls), 1)
        reserve_line = next(node.lineno for node in ast.walk(dispatch_fn)
                            if isinstance(node, ast.Call) and isinstance(node.func, ast.Attribute) and
                            node.func.attr == "reserve_controller_dispatch")
        bounded_line = next(node.lineno for node in ast.walk(dispatch_fn)
                            if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and
                            node.func.id == "bounded")
        self.assertLess(gate_calls[0], reserve_line)
        self.assertLess(gate_calls[0], bounded_line)

    def test_r5_process_budget_keeps_cleanup_inside_38_seconds(self):
        self.assertEqual(dispatch.r5_process_timeout(38, 0), 32)
        self.assertEqual(dispatch.r5_process_timeout(20, 5), 20)
        expired_budget = dispatch.r5_process_timeout(20, 32)
        self.assertLessEqual(expired_budget, 0)
        with patch.object(dispatch, "bounded") as mock_popen:
            if expired_budget > 0:
                mock_popen(timeout=expired_budget)
            mock_popen.assert_not_called()
        tree = ast.parse(Path(dispatch.__file__).read_text(encoding="utf-8"))
        dispatch_fn = next(node for node in tree.body
                           if isinstance(node, ast.FunctionDef) and node.name == "dispatch_government")
        bounded_line = next(node.lineno for node in ast.walk(dispatch_fn)
                            if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and
                            node.func.id == "bounded")
        timeout_line = next(node.lineno for node in ast.walk(dispatch_fn)
                            if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and
                            node.func.id == "r5_process_timeout")
        self.assertLess(timeout_line, bounded_line)

    def test_positive_r5_result_must_finish_within_controller_deadline(self):
        dispatch.validate_r5_terminal_deadline(self.request, 38)
        with self.assertRaisesRegex(ValueError, "deadline exceeded"):
            dispatch.validate_r5_terminal_deadline(self.request, 38.001)
        dispatch.validate_r5_terminal_deadline({"dispatchId": "other"}, 39)
        tree = ast.parse(Path(dispatch.__file__).read_text(encoding="utf-8"))
        dispatch_fn = next(node for node in tree.body
                           if isinstance(node, ast.FunctionDef) and node.name == "dispatch_government")
        check_line = next(node.lineno for node in ast.walk(dispatch_fn)
                          if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and
                          node.func.id == "validate_r5_terminal_deadline")
        finish_line = next(node.lineno for node in ast.walk(dispatch_fn)
                           if isinstance(node, ast.Call) and isinstance(node.func, ast.Attribute) and
                           node.func.attr == "finish_controller_dispatch")
        self.assertLess(check_line, finish_line)

if __name__ == "__main__":
    unittest.main()
