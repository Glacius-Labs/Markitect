"""Focused R7 profile and completion-receipt checks; no process starts."""
from __future__ import annotations

import ast
import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import dispatch
import government_native_profile
import government_roles
import native_controller


def sha(raw: bytes) -> str:
    return hashlib.sha256(raw).hexdigest()


class R7ProfileBridgeTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.profile = government_native_profile.profile("r7")
        self.executable = self.root / "government.exe"
        self.executable.write_bytes(b"pinned government binary")
        self.python = self.root / "python.exe"
        self.python.write_bytes(b"pinned interpreter")
        self.delegate = self.root / "delegate.py"
        self.delegate.write_bytes(b"deterministic delegate")
        runtime_files = [
            {"path": str(self.python), "digest": "sha256:" + sha(self.python.read_bytes())},
            {"path": str(self.delegate), "digest": "sha256:" + sha(self.delegate.read_bytes())},
        ]
        slots = []
        for phase in ("execute", "review", "vote"):
            slots.append({"slotId": phase, "phase": phase, "delegate": {
                "command": str(self.python),
                "commandDigest": "sha256:" + sha(self.python.read_bytes()),
                "argv": [str(self.python), str(self.delegate), "--phase", phase],
                "runtimeFiles": runtime_files}})
        self.auth_raw = json.dumps({"apiVersion": "test-role-auth", "slots": slots},
                                   sort_keys=True).encode()
        self.auth_path = self.root / "role-auth.json"
        self.auth_path.write_bytes(self.auth_raw)
        self.request = {
            "mode": "mechanical", "dispatchId": self.profile.dispatch_id,
            "arm": "government", "task": {"id": self.profile.task_id},
            "nativeFixtureR7Grant": {"path": str(self.root / "r7.json"), "sha256": "a" * 64,
                                     "sourceKey": self.profile.key},
            "nativeFixtureGrant": {"path": str(self.root / "r1.json"), "sha256": "b" * 64,
                                   "sourceKey": native_controller.NATIVE_FIXTURE_GRANT_KEY},
            "releasedInputs": [{"path": str(self.auth_path), "sha256": sha(self.auth_raw)}],
            "product": {"government": {
                "executable": {"path": str(self.executable), "sha256": sha(self.executable.read_bytes())},
                "roleAuthorization": {"path": str(self.auth_path), "sha256": sha(self.auth_raw)}}}}
        self.validated = {"grantKey": self.profile.key, "profileName": self.profile.name,
                          "dispatchId": self.profile.dispatch_id,
                          "grant": {"key": self.profile.key,
                                    "product": {"delegateSha256": sha(self.delegate.read_bytes())},
                                    "python": {"path": str(self.python),
                                               "sha256": sha(self.python.read_bytes())}}}

    def tearDown(self):
        self.temp.cleanup()

    def test_profile_selection_rejects_cross_profile_dispatch_and_marker(self):
        self.assertEqual(native_controller.native_profile(self.request), self.profile)
        for altered in (
                dict(self.request, dispatchId=government_native_profile.profile("r6").dispatch_id),
                dict(self.request, nativeFixtureR6Grant={"cross": True}),
                dict(self.request, arm="classic")):
            with self.assertRaises(ValueError):
                native_controller.native_profile(altered)

    def test_delegate_authorization_binds_r7_python_delegate_and_released_auth(self):
        self.assertEqual(native_controller.validate_r7_delegate_authorization(
            self.request, self.auth_raw, self.validated), [str(self.delegate.resolve())])
        changed = dict(self.request)
        changed["releasedInputs"] = []
        with self.assertRaisesRegex(ValueError, "released input"):
            native_controller.validate_r7_delegate_authorization(changed, self.auth_raw, self.validated)

    def test_delegate_or_interpreter_mutation_is_rejected(self):
        self.delegate.write_bytes(b"changed delegate")
        with self.assertRaisesRegex(ValueError, "delegate script differs"):
            native_controller.validate_r7_delegate_authorization(self.request, self.auth_raw, self.validated)
        self.delegate.write_bytes(b"deterministic delegate")
        self.python.write_bytes(b"changed interpreter")
        with self.assertRaisesRegex(ValueError, "Python executable differs"):
            native_controller.validate_r7_delegate_authorization(self.request, self.auth_raw, self.validated)

    def test_entry_wrapper_rejects_wrong_profile_before_grant_binding(self):
        changed = dict(self.request, dispatchId=government_native_profile.profile("r6").dispatch_id)
        with patch.object(native_controller, "validate_native_fixture_grant") as bind:
            with self.assertRaisesRegex(ValueError, "exact separate R5, R6 or R7 Government Request profile"):
                native_controller.validate_r7_entry_for_launch(changed, {}, [str(self.executable), "queue"])
            bind.assert_not_called()

    def test_controller_and_role_gates_precede_reservation_and_popen(self):
        tree = ast.parse(Path(dispatch.__file__).read_text(encoding="utf-8"))
        dispatch_fn = next(node for node in tree.body
                           if isinstance(node, ast.FunctionDef) and node.name == "dispatch_government")
        gates = sorted(node.lineno for node in ast.walk(dispatch_fn)
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
        self.assertEqual(len(gates), 2)
        self.assertLess(gates[0], reserve_line)
        self.assertLess(gates[1], bounded_line)
        self.assertLess(timeout_line, bounded_line)
        self.assertEqual(dispatch.r5_process_timeout(38, 0), 32)
        self.assertEqual(dispatch.r5_process_timeout(38, 32), 0)

        roles_tree = ast.parse(Path(government_roles.__file__).read_text(encoding="utf-8"))
        run_role = next(node for node in roles_tree.body
                        if isinstance(node, ast.FunctionDef) and node.name == "run_role")
        role_gates = sorted(node.lineno for node in ast.walk(run_role)
                            if isinstance(node, ast.Call) and isinstance(node.func, ast.Attribute) and
                            node.func.attr == "validate_profile_live_gate")
        reserve_role_line = next(node.lineno for node in ast.walk(run_role)
                                 if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and
                                 node.func.id == "_reserve")
        delegate_line = next(node.lineno for node in ast.walk(run_role)
                             if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and
                             node.func.id == "bounded")
        self.assertEqual(len(role_gates), 2)
        self.assertLess(role_gates[0], reserve_role_line)
        self.assertLess(role_gates[1], delegate_line)


class R7CompletionReceiptTests(unittest.TestCase):
    def test_positive_timing_requires_terminal_receipts_and_deadline(self):
        args = ("completed", "finished", "completed", "a" * 64, True, True, True, True, 38)
        self.assertEqual(dispatch.profile_completion_status(*args), "completed")
        for invalid in (
                (*args[:-1], 38.001), (*args[:3], None, *args[4:]),
                (*args[:4], False, *args[5:]), (*args[:5], False, *args[6:]),
                (*args[:6], False, *args[7:]), (*args[:7], False, *args[8:])):
            self.assertEqual(dispatch.profile_completion_status(*invalid), "incomplete")

    def test_r7_receipt_writer_follows_terminal_result_and_reads_bound_records(self):
        source = Path(dispatch.__file__).read_text(encoding="utf-8")
        tree = ast.parse(source)
        dispatch_fn = next(node for node in tree.body
                           if isinstance(node, ast.FunctionDef) and node.name == "dispatch_government")
        calls = [(node.lineno, node.func.id) for node in ast.walk(dispatch_fn)
                 if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and
                 node.func.id in {"write_result", "write_r7_completion_receipt"}]
        result_lines = [line for line, name in calls if name == "write_result"]
        receipt_lines = [line for line, name in calls if name == "write_r7_completion_receipt"]
        self.assertTrue(result_lines)
        self.assertEqual(len(receipt_lines), 1)
        self.assertLess(max(result_lines), receipt_lines[0])
        writer = next(node for node in tree.body
                      if isinstance(node, ast.FunctionDef) and node.name == "write_profile_completion_receipt")
        writer_text = ast.get_source_segment(source, writer)
        for evidence in ("controller_runs", "execution_sha(request)", "controller_elapsed",
                         "controller-completion.json"):
            self.assertIn(evidence, writer_text)


if __name__ == "__main__":
    unittest.main()
