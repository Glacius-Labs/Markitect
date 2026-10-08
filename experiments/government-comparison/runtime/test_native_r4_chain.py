"""Offline R4 admission-chain regressions; no product, wrapper, or delegate starts."""
from __future__ import annotations

import hashlib
import ast
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import native_controller
import native_fixture_budget
import government_roles


def sha(raw: bytes) -> str:
    return hashlib.sha256(raw).hexdigest()


class R4LaunchAdmissionTests(unittest.TestCase):
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
            {"path": str(self.interpreter), "mode": "0644", "digest": "sha256:" + sha(self.interpreter.read_bytes())},
            {"path": str(self.delegate), "mode": "0644", "digest": "sha256:" + sha(self.delegate.read_bytes())},
        ]
        auth = {"apiVersion": "test-role-auth", "slots": [
            {"slotId": "execute", "phase": "execute", "delegate": {
                "command": str(self.interpreter), "commandDigest": "sha256:" + sha(self.interpreter.read_bytes()),
                "argv": [str(self.interpreter), str(self.delegate), "--phase", "execute"],
                "runtimeFiles": runtime_files}},
            {"slotId": "verify", "phase": "review", "delegate": {
                "command": str(self.interpreter), "commandDigest": "sha256:" + sha(self.interpreter.read_bytes()),
                "argv": [str(self.interpreter), str(self.delegate), "--phase", "review"],
                "runtimeFiles": runtime_files}},
        ]}
        self.auth_raw = json.dumps(auth, sort_keys=True).encode()
        self.auth_path = self.root / "role-auth.json"
        self.auth_path.write_bytes(self.auth_raw)
        self.request = {
            "mode": "mechanical",
            "dispatchId": native_controller.R4_DISPATCH_ID,
            "arm": "government",
            "nativeFixtureGrant": {"path": str(self.root / "r1.json"), "sha256": "0" * 64,
                                   "sourceKey": native_controller.NATIVE_FIXTURE_GRANT_KEY},
            "nativeFixtureR4Grant": {"path": str(self.root / "r4.json"), "sha256": "1" * 64,
                                     "sourceKey": native_controller.R4_GRANT_KEY},
            "releasedInputs": [{"path": str(self.auth_path), "sha256": sha(self.auth_raw)}],
            "product": {"government": {
                "executable": {"path": str(self.executable), "sha256": sha(self.executable.read_bytes())},
                "roleAuthorization": {"path": str(self.auth_path), "sha256": sha(self.auth_raw)}}}}
        self.captured = {str(self.auth_path.resolve()): self.auth_raw}
        self.validated = {
            "grantKey": native_controller.R4_GRANT_KEY,
            "grant": {"key": native_controller.R4_GRANT_KEY,
                      "product": {"delegateSha256": sha(self.delegate.read_bytes())}},
            "product": {"name": "Government"}}

    def tearDown(self):
        self.temp.cleanup()

    def test_early_diagnostics_parser_precedes_main_hook(self):
        source = Path(government_roles.__file__).read_text(encoding="utf-8")
        tree = ast.parse(source)
        strict_parser = next(node for node in tree.body
                             if isinstance(node, ast.FunctionDef) and node.name == "_strict_json")
        hook = next(node for node in tree.body
                    if isinstance(node, ast.Assign) and any(
                        isinstance(target, ast.Name) and target.id == "_NATIVE_DIAGNOSTICS"
                        for target in node.targets))
        self.assertLess(strict_parser.lineno, hook.lineno)
        self.assertEqual(government_roles._strict_json(b'{"ok":true}', "test"), {"ok": True})
        with self.assertRaisesRegex(ValueError, "duplicate JSON keys"):
            government_roles._strict_json(b'{"x":1,"x":2}', "test")

    def _validate(self, request=None, captured=None, argv=None):
        with patch.object(native_controller, "validate_native_fixture_grant",
                          return_value={"r4Grant": self.validated}) as bind, \
             patch.object(native_fixture_budget, "validate_r4_entry_gate",
                          return_value={"slotOwner": "Scientist", "slotKey": native_controller.R4_GRANT_KEY}) as gate:
            outcome = native_controller.validate_r4_entry_for_launch(
                self.request if request is None else request,
                self.captured if captured is None else captured,
                [str(self.executable), "queue"] if argv is None else argv)
            return outcome, bind, gate

    def test_exact_request_binary_delegate_and_live_gate_pass(self):
        outcome, bind, gate = self._validate()
        bind.assert_called_once_with(self.request, self.captured)
        gate.assert_called_once_with(self.validated)
        self.assertEqual(outcome["entryGate"]["slotKey"], native_controller.R4_GRANT_KEY)
        self.assertEqual(outcome["delegatePaths"], [str(self.delegate.resolve())])

    def test_r2_or_r3_combination_is_rejected_before_binding(self):
        for field in ("nativeFixtureCorrection", "nativeFixtureR3Grant"):
            request = dict(self.request)
            request[field] = {"unexpected": True}
            with patch.object(native_controller, "validate_native_fixture_grant") as bind:
                with self.assertRaisesRegex(ValueError, "Government-only additive grant"):
                    native_controller.validate_r4_entry_for_launch(
                        request, self.captured, [str(self.executable), "queue"])
                bind.assert_not_called()

    def test_wrong_dispatch_is_rejected(self):
        request = dict(self.request, dispatchId="government-native-contract-corrected-r3")
        with patch.object(native_controller, "validate_native_fixture_grant") as bind:
            with self.assertRaisesRegex(ValueError, "Government-only additive grant"):
                native_controller.validate_r4_entry_for_launch(
                    request, self.captured, [str(self.executable), "queue"])
            bind.assert_not_called()

    def test_native_executable_must_match_request_pin(self):
        other = self.root / "other.exe"
        other.write_bytes(b"different")
        with patch.object(native_controller, "validate_native_fixture_grant",
                          return_value={"r4Grant": self.validated}), \
             patch.object(native_fixture_budget, "validate_r4_entry_gate") as gate:
            with self.assertRaisesRegex(ValueError, "Request-bound Government executable"):
                native_controller.validate_r4_entry_for_launch(
                    self.request, self.captured, [str(other), "queue"])
            gate.assert_not_called()

    def test_delegate_change_stops_before_live_gate(self):
        self.delegate.write_bytes(b"changed delegate")
        with patch.object(native_controller, "validate_native_fixture_grant",
                          return_value={"r4Grant": self.validated}), \
             patch.object(native_fixture_budget, "validate_r4_entry_gate") as gate:
            with self.assertRaisesRegex(ValueError, "delegate script differs"):
                native_controller.validate_r4_entry_for_launch(
                    self.request, self.captured, [str(self.executable), "queue"])
            gate.assert_not_called()

    def test_interpreter_change_stops_before_live_gate(self):
        self.interpreter.write_bytes(b"changed interpreter")
        with patch.object(native_controller, "validate_native_fixture_grant",
                          return_value={"r4Grant": self.validated}), \
             patch.object(native_fixture_budget, "validate_r4_entry_gate") as gate:
            with self.assertRaisesRegex(ValueError, "interpreter differs"):
                native_controller.validate_r4_entry_for_launch(
                    self.request, self.captured, [str(self.executable), "queue"])
            gate.assert_not_called()

    def test_live_revocation_stops_exact_launch_gate(self):
        with patch.object(native_controller, "validate_native_fixture_grant",
                          return_value={"r4Grant": self.validated}), \
             patch.object(native_fixture_budget, "validate_r4_entry_gate",
                          side_effect=ValueError("R4 grant revoked")) as gate:
            with self.assertRaisesRegex(ValueError, "revoked"):
                native_controller.validate_r4_entry_for_launch(
                    self.request, self.captured, [str(self.executable), "queue"])
            gate.assert_called_once_with(self.validated)

    def test_role_authorization_must_be_captured_and_pinned(self):
        with patch.object(native_controller, "validate_native_fixture_grant",
                          return_value={"r4Grant": self.validated}), \
             patch.object(native_fixture_budget, "validate_r4_entry_gate") as gate:
            with self.assertRaisesRegex(ValueError, "exact captured Request input"):
                native_controller.validate_r4_entry_for_launch(
                    self.request, {}, [str(self.executable), "queue"])
            gate.assert_not_called()

    def test_role_authorization_file_must_still_match_captured_bytes(self):
        self.auth_path.write_bytes(self.auth_raw + b" ")
        with patch.object(native_controller, "validate_native_fixture_grant",
                          return_value={"r4Grant": self.validated}), \
             patch.object(native_fixture_budget, "validate_r4_entry_gate") as gate:
            with self.assertRaisesRegex(ValueError, "current Request-bound file"):
                native_controller.validate_r4_entry_for_launch(
                    self.request, self.captured, [str(self.executable), "queue"])
            gate.assert_not_called()


if __name__ == "__main__":
    unittest.main()
