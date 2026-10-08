"""Narrow offline integration tests for the selected-feature metadata client."""
import copy
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch


ROOT = Path(__file__).resolve().parent
PACKAGE = ROOT.parents[1]
spec = importlib.util.spec_from_file_location("selected_feature_metadata_client", ROOT / "client.py")
client = importlib.util.module_from_spec(spec)
spec.loader.exec_module(client)

EXPECTED = {
    "apps": False, "goals": False, "hooks": False, "memories": False,
    "multi_agent": False, "plugins": False, "shell_tool": True,
    "unified_exec": True,
}
VERSION = "sha256:" + "a" * 64
SENT = [
    {"method": "initialize", "id": 0},
    {"method": "initialized", "id": None},
    {"method": "config/read", "id": 1},
    {"method": "configRequirements/read", "id": 2},
]


def raw_config(feature_values=None):
    return {
        "config": {
            "default_permissions": ":read-only",
            "approval_policy": "never",
            "model": "gpt-6.1-sol",
            "model_provider": "openai",
            "model_reasoning_effort": "high",
            "features": copy.deepcopy(EXPECTED if feature_values is None else feature_values),
        },
        "origins": {"default_permissions": {
            "name": {"type": "sessionFlags"}, "version": VERSION,
        }},
        "layers": [{
            "name": {"type": "sessionFlags"}, "version": VERSION,
            "config": {"default_permissions": ":read-only"},
        }],
    }


def config_envelope(feature_values=None):
    return {"id": 1, "result": raw_config(feature_values)}


def requirements_envelope(feature_values=None, *, result=None):
    body = {"requirements": {"featureRequirements": copy.deepcopy(
        EXPECTED if feature_values is None else feature_values)}}
    if result is not None:
        body = result
    return {"id": 2, "result": body}


def successful_chain(*, features=None, requirement_features=None):
    config = client.decode_response(config_envelope(features), 1)
    requirements = client.decode_response(requirements_envelope(requirement_features), 2)
    observations = [
        {"id": 0, "sanitizedResult": client.decode_response({"id": 0, "result": {}}, 0)},
        {"id": 1, "sanitizedResult": config},
        {"id": 2, "sanitizedResult": requirements},
    ]
    return observations, config, requirements


class SelectedFeatureClientIntegrationTests(unittest.TestCase):
    def test_response_envelopes_flow_through_sanitizers_and_real_assessment(self):
        features = {**EXPECTED, "foreign_feature_name": "must not appear"}
        observations, config, requirements = successful_chain(features=features)
        self.assertEqual(client.assess_config(config), ("visible-metadata-precondition-satisfied", None))
        self.assertEqual(client.final_assess(observations, SENT),
                         ("visible-metadata-precondition-satisfied", None))
        serialized = json.dumps({"config": config, "requirements": requirements}, sort_keys=True)
        self.assertNotIn("foreign_feature_name", serialized)
        self.assertNotIn("must not appear", serialized)
        self.assertEqual(set(config["config"]["features"]["selected"]), set(EXPECTED))

    def test_final_positive_requires_actual_typed_requirements_response(self):
        observations, _, _ = successful_chain()
        self.assertEqual(client.final_assess(observations, SENT)[0],
                         "visible-metadata-precondition-satisfied")

        missing_response = observations[:2]
        self.assertEqual(client.final_assess(missing_response, SENT),
                         ("insufficient", "actual-requirements-response-required"))
        missing_rpc = SENT[:-1]
        self.assertEqual(client.final_assess(observations, missing_rpc),
                         ("insufficient", "actual-requirements-response-required"))

        placeholder = client.decode_response(requirements_envelope(result={}), 2)
        placeholder_observations = [*observations[:2], {"id": 2, "sanitizedResult": placeholder}]
        self.assertNotEqual(client.final_assess(placeholder_observations, SENT)[0],
                            "visible-metadata-precondition-satisfied")

        with self.assertRaises(ValueError):
            client.decode_response({"id": 9, "result": {"requirements": {}}}, 2)
        with self.assertRaises(ValueError):
            client.decode_response({"id": 2, "result": {}, "error": {"code": -1}}, 2)

    def test_unknown_boolean_requirement_is_unresolved_and_known_conflict_is_not_positive(self):
        unknown = {**EXPECTED, "network_proxy": True}
        observations, _, requirements = successful_chain(requirement_features=unknown)
        managed = requirements["requirements"]["value"]["featureRequirements"]
        self.assertTrue(managed["unresolved"])
        self.assertNotIn("network_proxy", json.dumps(requirements))
        self.assertEqual(client.final_assess(observations, SENT),
                         ("insufficient", "requirement-feature-unresolved"))

        conflict = {**EXPECTED, "apps": True, "network_proxy": True}
        observations, _, _ = successful_chain(requirement_features=conflict)
        self.assertEqual(client.final_assess(observations, SENT),
                         ("incompatible", "managed-feature-conflict"))

    def test_adapter_source_hash_guard_is_fail_closed(self):
        with tempfile.TemporaryDirectory() as directory:
            changed_adapter = Path(directory) / "selected_feature_contract.py"
            changed_adapter.write_text("# different source\n", encoding="utf-8")
            with patch.object(client, "ADAPTER", changed_adapter):
                with self.assertRaisesRegex(ValueError, "source-pin-mismatch"):
                    client.adapter()

    def test_exact_argv_grant_cwd_and_live_status_guards_reject_tampering(self):
        profile = json.loads((ROOT / "profile.json").read_text(encoding="utf-8"))
        authority = json.loads((ROOT / "authorization-grant.json").read_text(encoding="utf-8"))
        grant = authority["grant"]
        plan_path = PACKAGE / "public/s1-common-runner-readiness-plan-20261008.md"
        plan = plan_path.read_text(encoding="utf-8")
        client.validate_profile(profile, grant, plan)

        changed_argv = copy.deepcopy(profile)
        changed_argv["argv"].append("--extra")
        with self.assertRaisesRegex(ValueError, "argv-mismatch"):
            client.validate_profile(changed_argv, grant, plan)
        changed_cwd = copy.deepcopy(profile)
        changed_cwd["cwd"] += "/different"
        with self.assertRaisesRegex(ValueError, "rpc-or-cwd-mismatch"):
            client.validate_profile(changed_cwd, grant, plan)
        changed_grant = copy.deepcopy(grant)
        changed_grant["remainingSessions"] = 2
        with self.assertRaisesRegex(ValueError, "closed-grant"):
            client.validate_profile(profile, changed_grant, plan)

        slot = {"owner": "Scientist", "key": client.KEY, "assignedUtc": client.ISSUED,
                "status": "Assigned to one bounded corrected metadata operation; no product suite."}
        client.validate_live_authority(grant, slot, grant)
        for status in ("Revoked", "Pending", "Closed"):
            with self.subTest(grant_status=status):
                changed = {**grant, "status": status}
                with self.assertRaisesRegex(ValueError, "inactive-live-grant"):
                    client.validate_live_authority(changed, slot, grant)
        for status in ("Revoked", "Pending", "Closed"):
            with self.subTest(slot_status=status):
                changed_slot = {**slot, "status": status}
                with self.assertRaisesRegex(ValueError, "inactive-live-slot"):
                    client.validate_live_authority(grant, changed_slot, grant)


if __name__ == "__main__":
    unittest.main()
