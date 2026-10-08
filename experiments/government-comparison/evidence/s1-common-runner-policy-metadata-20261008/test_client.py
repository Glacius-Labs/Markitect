"""Offline regressions for the bounded common-runner metadata client."""
import copy
import hashlib
import importlib.util
import json
from pathlib import Path
from types import SimpleNamespace
import tempfile
import unittest
from unittest.mock import patch


ROOT = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("metadata_client_under_test", ROOT / "client.py")
client = importlib.util.module_from_spec(spec)
spec.loader.exec_module(client)


def raw_config(**overrides):
    value = {
        "default_permissions": ":read-only",
        "approval_policy": "never",
        "model": "gpt-6.1-sol",
        "model_provider": "openai",
        "model_reasoning_effort": "high",
        "features": {
            "apps": False, "goals": False, "hooks": False, "memories": False,
            "multi_agent": False, "plugins": False, "shell_tool": True,
            "unified_exec": True, "unrecognized-secret": True,
        },
        "permissions": {"private-profile-name": {"definition": "do not retain"}},
        "windows": {"sandbox": "unelevated", "private_path": "C:/secret"},
    }
    value.update(overrides)
    return value


def config_result(config=None, *, origin_type="sessionFlags", layers_marker=None):
    version = "sha256:" + "a" * 64
    layer = {"name": {"type": "sessionFlags"}, "version": version,
             "config": {"default_permissions": ":read-only"}}
    layers = [layer] if layers_marker is None else layers_marker
    return {"config": raw_config() if config is None else config,
            "origins": {"default_permissions": {
                "name": {"type": origin_type, "profile": "private-origin-name"},
                "version": version, "private_path": "C:/private/origin"}},
            "layers": layers}


def requirements_result(requirements=None):
    return {"requirements": {} if requirements is None else requirements}


class MetadataClientTests(unittest.TestCase):
    def test_live_authority_requires_exact_grant_and_active_slot(self):
        original = json.loads((ROOT / "authorization-grant.json").read_text(encoding="utf-8"))["grant"]
        grant = copy.deepcopy(original)
        slot = {"owner": "Scientist", "key": client.KEY, "assignedUtc": client.ISSUED,
                "status": "Assigned to one bounded prospective metadata operation, not a full product suite."}
        client.validate_live_authority(grant, slot, original)

        for status in ("Revoked", "Pending", "Closed", "Active; expired"):
            with self.subTest(grant_status=status):
                changed = copy.deepcopy(grant)
                changed["status"] = status
                with self.assertRaisesRegex(ValueError, "inactive-live-grant"):
                    client.validate_live_authority(changed, slot, original)
        for status in ("Revoked", "Pending", "Closed", "Unassigned"):
            with self.subTest(slot_status=status):
                changed_slot = {**slot, "status": status}
                with self.assertRaisesRegex(ValueError, "inactive-live-slot"):
                    client.validate_live_authority(grant, changed_slot, original)
        wrong_slot = {**slot, "key": "another-grant"}
        with self.assertRaisesRegex(ValueError, "slot-mismatch"):
            client.validate_live_authority(grant, wrong_slot, original)
        changed_limit = copy.deepcopy(grant)
        changed_limit["maxRawBytes"] += 1
        with self.assertRaisesRegex(ValueError, "live-grant-mismatch"):
            client.validate_live_authority(changed_limit, slot, original)

    def test_presence_states_cover_missing_null_present_and_invalid(self):
        convert = lambda value: client.enum(value, {"never"})
        self.assertEqual(client.field({}, "x", convert), {"state": "missing"})
        self.assertEqual(client.field({"x": None}, "x", convert), {"state": "null"})
        self.assertEqual(client.field({"x": "never"}, "x", convert),
                         {"state": "present", "value": "never"})
        self.assertEqual(client.field({"x": "private-value"}, "x", convert), {"state": "invalid"})

    def test_privacy_aliases_and_allowlisted_metadata_only(self):
        config = client.sanitize_config_read(config_result())
        requirements = client.sanitize_requirements(requirements_result({
            "allowedPermissionProfiles": {"private-profile-name": True},
            "models": {"secret-provider-model": {"path": "C:/secret/model"}},
            "additionalDeveloperInstructions": "private prompt text",
        }))
        serialized = json.dumps({"config": config, "requirements": requirements}, sort_keys=True)
        self.assertNotIn("private-profile-name", serialized)
        self.assertNotIn("private-origin-name", serialized)
        self.assertNotIn("C:/private/origin", serialized)
        self.assertNotIn("C:/secret", serialized)
        self.assertNotIn("secret-provider-model", serialized)
        self.assertNotIn("private prompt text", serialized)
        self.assertNotIn("do not retain", serialized)
        self.assertEqual(config["config"]["permissions"]["value"], {
            "opaque-" + client.sha(b"private-profile-name")[:24]: {"definitionPresent": True}})
        self.assertEqual(config["origins"]["default_permissions"]["value"]["selectedProfile"],
                         {"state": "present", "value": "opaque-" + client.sha(b"private-origin-name")[:24]})
        self.assertEqual(config["config"]["features"]["value"], {
            "apps": False, "goals": False, "hooks": False, "memories": False,
            "multi_agent": False, "plugins": False, "shell_tool": True, "unified_exec": True})
        self.assertEqual(requirements["requirements"]["value"]["models"], {
            "state": "present", "value": {"uninterpretedConstraintPresent": True}})

    def test_default_permission_and_origin_ambiguity_never_satisfy_assessment(self):
        good = client.sanitize_config_read(config_result())
        req = client.sanitize_requirements(requirements_result())
        self.assertEqual(client.assess(good, req)[0], "visible-metadata-precondition-satisfied")
        absent_requirements = client.sanitize_requirements({"requirements": None})
        self.assertEqual(absent_requirements["requirements"], {"state": "null"})
        self.assertEqual(client.assess(good, absent_requirements)[0],
                         "visible-metadata-precondition-satisfied")
        cases = []
        missing_default = copy.deepcopy(good)
        missing_default["config"]["default_permissions"] = {"state": "missing"}
        cases.append((missing_default, "selected-default-or-origin-unobserved"))
        missing_origin = copy.deepcopy(good)
        missing_origin["origins"]["default_permissions"] = {"state": "missing"}
        cases.append((missing_origin, "selected-default-or-origin-unobserved"))
        ambiguous_type = client.sanitize_config_read(config_result(origin_type="project"))
        cases.append((ambiguous_type, "selected-origin-ambiguous"))
        ambiguous_layers = client.sanitize_config_read(config_result(layers_marker=[
            {"name": {"type": "sessionFlags"}, "version": "sha256:" + "a" * 64,
             "config": {"default_permissions": ":read-only"}},
            {"name": {"type": "sessionFlags"}, "version": "sha256:" + "a" * 64,
             "config": {"default_permissions": ":read-only"}},
        ]))
        cases.append((ambiguous_layers, "selected-origin-layer-ambiguous"))
        for observed, reason in cases:
            with self.subTest(reason=reason):
                self.assertEqual(client.assess(observed, req), ("insufficient", reason))

    def test_requirements_validate_boolean_profile_map_and_detect_conflicts(self):
        base = client.sanitize_config_read(config_result())
        allowed = client.sanitize_requirements(requirements_result({
            "allowedPermissionProfiles": {":read-only": True}}))
        self.assertEqual(client.assess(base, allowed)[0], "visible-metadata-precondition-satisfied")
        denied = client.sanitize_requirements(requirements_result({
            "allowedPermissionProfiles": {":read-only": False}}))
        self.assertEqual(client.assess(base, denied), ("incompatible", "managed-profile-allowlist-conflict"))
        wrong_shape = client.sanitize_requirements(requirements_result({
            "allowedPermissionProfiles": [":read-only"]}))
        self.assertEqual(wrong_shape["requirements"]["value"]["allowedPermissionProfiles"]["state"], "invalid")
        self.assertEqual(client.assess(base, wrong_shape), ("insufficient", "invalid-requirement"))
        conflicts = (
            ({"allowedApprovalPolicies": ["on-request"]}, "managed-enum-conflict"),
            ({"defaultPermissions": "private-profile"}, "managed-default-conflict"),
            ({"allowedSandboxModes": ["danger-full-access"]}, "legacy-managed-sandbox-constraint-unresolved"),
            ({"featureRequirements": {"shell_tool": False}}, "managed-feature-conflict"),
            ({"modelProvider": "private provider text"}, "uninterpreted-managed-constraint"),
        )
        for item, reason in conflicts:
            with self.subTest(item=item):
                result = client.sanitize_requirements(requirements_result(item))
                self.assertEqual(client.assess(base, result)[1], reason)

    def test_every_warning_or_provisional_payload_stops(self):
        warning_methods = (("warning", True), ("configWarning", True),
                           ("windows/worldWritableWarning", True), ("deprecationNotice", False))
        for method, provisional in warning_methods:
            with self.subTest(method=method):
                result = client.classify({"method": method, "params": {}}, self._enums(), 0)
                self.assertEqual(result["action"], "stop")
                self.assertEqual(result["provisional"], provisional)
        for key in ("warning", "warnings", "configWarning", "provisional", "provisionalConfig"):
            with self.subTest(key=key), self.assertRaisesRegex(ValueError, "warning-or-provisional-result"):
                client.reject_provisional({"nested": [{key: "raw warning text"}]})

    def test_disabled_remote_status_is_the_only_accepted_status_and_is_sanitized(self):
        accepted = client.classify({"method": "remoteControl/status/changed",
                                    "params": {"status": "disabled", "installationId": "secret",
                                               "serverName": "private host"}}, self._enums(), 0)
        self.assertEqual(accepted["action"], "discard-notification")
        self.assertEqual(accepted["event"], {"method": "remoteControl/status/changed",
                                              "class": "RemoteControl/status/changedNotification",
                                              "status": "disabled"})
        for status in ("connecting", "connected", "errored"):
            with self.subTest(status=status):
                result = client.classify({"method": "remoteControl/status/changed",
                                          "params": {"status": status, "installationId": "private",
                                                     "serverName": "private host"}}, self._enums(), 0)
                self.assertEqual(result["action"], "stop")
                self.assertEqual(result["reason"], "remote-control-status-not-disabled")
        malformed = client.classify({"method": "remoteControl/status/changed", "params": {"status": 4}},
                                    self._enums(), 0)
        self.assertEqual(malformed["reason"], "malformed-remote-control-status")

    def test_malformed_frames_server_requests_and_notification_limit_stop(self):
        enums = self._enums()
        self.assertEqual(client.classify({"method": "", "params": {}}, enums, 0)["action"], "stop")
        request = client.classify({"id": 7, "method": "item/commandExecution/requestApproval", "params": {}}, enums, 0)
        self.assertEqual(request["action"], "stop")
        self.assertEqual(request["reason"], "unexpected-frame-envelope")
        bad_remote = client.classify({"method": "remoteControl/status/changed", "params": {}}, enums, 0)
        self.assertEqual(bad_remote["action"], "stop")
        limited = client.classify({"method": "remoteControl/status/changed", "params": {"status": "disabled"}},
                                  enums, 16)
        self.assertEqual(limited["reason"], "notification-limit")
        self.assertEqual(limited["notificationCount"], 17)
        unexpected_envelope = client.classify({"method": "remoteControl/status/changed", "params": {"status": "disabled"},
                                               "result": {}}, enums, 0)
        self.assertEqual(unexpected_envelope["reason"], "unexpected-frame-envelope")

    def test_duplicate_json_keys_and_nonstandard_numbers_are_rejected(self):
        for raw, reason in ((b'{"x":1,"x":2}', "duplicate-json-key"),
                            (b'{"x":NaN}', "invalid-json-number")):
            with self.subTest(raw=raw), self.assertRaisesRegex(ValueError, reason):
                client.strict_json(raw)

    def test_profile_argv_cwd_limits_and_plan_tampering_guards(self):
        profile = json.loads((ROOT / "profile.json").read_text(encoding="utf-8"))
        grant = json.loads((ROOT / "authorization-grant.json").read_text(encoding="utf-8"))["grant"]
        plan_path = client.PACKAGE / "public/s1-common-runner-readiness-plan-20261008.md"
        plan = plan_path.read_text(encoding="utf-8")
        client.validate_profile(profile, grant, plan)
        cases = (
            ("argv", lambda p: p["argv"].__setitem__(1, "shell")),
            ("cwd", lambda p: p.__setitem__("cwd", p["cwd"] + "/tampered")),
            ("limits", lambda p: p["limits"].__setitem__("maxRawBytes", 1)),
            ("rpc", lambda p: p["rpc"][0].__setitem__("method", "shutdown")),
        )
        for name, mutate in cases:
            changed = copy.deepcopy(profile)
            mutate(changed)
            with self.subTest(name=name), self.assertRaises(ValueError):
                client.validate_profile(changed, grant, plan)
        with self.assertRaisesRegex(ValueError, "rpc-or-cwd-mismatch"):
            client.validate_profile(profile, grant, plan.replace("initialize", "initialize_tampered", 1))
        changed_grant = copy.deepcopy(grant)
        changed_grant["maxRawBytes"] += 1
        with self.assertRaisesRegex(ValueError, "limits-mismatch"):
            client.validate_profile(profile, changed_grant, plan)

    def test_cwd_guard_accepts_exact_temp_snapshot_and_rejects_tampering(self):
        with tempfile.TemporaryDirectory() as directory:
            cwd = Path(directory)
            content = b"fixture only\n"
            (cwd / "README.txt").write_bytes(content)
            profile = {"cwd": str(cwd), "expectedFiles": {
                "README.txt": hashlib.sha256(content).hexdigest()}}
            self.assertEqual(client.validate_cwd(profile), profile["expectedFiles"])
            (cwd / "README.txt").write_text("changed", encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "cwd-content-mismatch"):
                client.validate_cwd(profile)
            (cwd / "extra-directory").mkdir()
            with self.assertRaisesRegex(ValueError, "unexpected-cwd-entry"):
                client.validate_cwd(profile)

    def test_controller_writes_go_only_after_job_assignment(self):
        with tempfile.TemporaryDirectory() as directory:
            external = Path(directory)
            (external / "sanitized-result.json").write_text("{}", encoding="utf-8")
            events = []

            class FakeStdin:
                closed = False

                def write(self, data):
                    events.append(("write", data, (external / "job-assigned.json").is_file()))

                def close(self):
                    self.closed = True

            class FakeWorker:
                pid = 4242

                def __init__(self):
                    self.stdin = FakeStdin()
                    self.code = None

                def poll(self):
                    return self.code

                def wait(self, timeout=None):
                    return self.code

                def kill(self):
                    self.code = -9

            worker = FakeWorker()

            class FakeJob:
                def __init__(self, owned_worker):
                    self.worker = owned_worker
                    events.append(("job-assigned", owned_worker.pid))
                    original_close = owned_worker.stdin.close

                    def complete_on_close():
                        original_close()
                        owned_worker.code = 0
                    owned_worker.stdin.close = complete_on_close

                def close(self):
                    events.append(("job-closed",))

                def kill(self):
                    worker.kill()

            profile = {"externalEvidence": str(external), "cwd": str(external)}
            binding = {"requestSha256": "r", "freezeSha256": "f", "reservationSha256": "s",
                       "sourceCommit": "commit", "freezeCommit": "freeze"}
            fake_process_module = SimpleNamespace(WindowsJob=FakeJob)
            with (patch.object(client, "load_request", return_value=(profile, binding)),
                  patch.object(client, "load_module", return_value=fake_process_module),
                  patch.object(client.subprocess, "Popen", return_value=worker)):
                self.assertEqual(client.controller(), 0)
            self.assertEqual(events[0], ("job-assigned", worker.pid))
            self.assertEqual(events[1], ("write", b"GO\n", True))
            self.assertTrue(worker.stdin.closed)
            receipt = json.loads((external / "controller-receipt.json").read_text(encoding="utf-8"))
            self.assertEqual(receipt["workerReturnCode"], 0)
            self.assertTrue(receipt["terminalResultPresent"])

    def test_controller_never_writes_go_if_job_assignment_fails(self):
        with tempfile.TemporaryDirectory() as directory:
            external = Path(directory)
            events = []

            class FakeStdin:
                def write(self, data):
                    events.append(("write", data))

                def close(self):
                    events.append(("stdin-close",))

            class FakeWorker:
                pid = 4343

                def __init__(self):
                    self.stdin = FakeStdin()
                    self.code = None

                def poll(self):
                    return self.code

                def wait(self, timeout=None):
                    return self.code

                def kill(self):
                    events.append(("worker-kill",))
                    self.code = -9

            worker = FakeWorker()

            class FailingJob:
                def __init__(self, owned_worker):
                    events.append(("job-assignment-failed", owned_worker.pid))
                    raise RuntimeError("simulated assignment failure")

            profile = {"externalEvidence": str(external), "cwd": str(external)}
            binding = {"requestSha256": "r", "freezeSha256": "f", "reservationSha256": "s",
                       "sourceCommit": "commit", "freezeCommit": "freeze"}
            fake_process_module = SimpleNamespace(WindowsJob=FailingJob)
            with (patch.object(client, "load_request", return_value=(profile, binding)),
                  patch.object(client, "load_module", return_value=fake_process_module),
                  patch.object(client.subprocess, "Popen", return_value=worker)):
                self.assertEqual(client.controller(), 1)
            self.assertIn(("job-assignment-failed", worker.pid), events)
            self.assertIn(("worker-kill",), events)
            self.assertFalse(any(event[0] == "write" and event[1] == b"GO\n" for event in events))
            self.assertFalse((external / "job-assigned.json").exists())

    @staticmethod
    def _enums():
        return client.strict_json((ROOT / "frozen-method-enums.json").read_bytes())


if __name__ == "__main__":
    unittest.main()
