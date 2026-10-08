"""Offline response-to-assessment integration tests for the R3 metadata client."""
import copy
import importlib.util
import json
from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("selected_feature_metadata_r3_client", ROOT / "client.py")
client = importlib.util.module_from_spec(spec)
spec.loader.exec_module(client)

EXPECTED = {
    "apps": False, "goals": False, "hooks": False, "memories": False,
    "multi_agent": False, "plugins": False, "shell_tool": True,
    "unified_exec": True,
}
VERSION = "sha256:" + "a" * 64


def raw_config(*, feature_values=None, origin_type="sessionFlags"):
    config = {
        "default_permissions": ":read-only",
        "approval_policy": "never",
        "model": "gpt-6.1-sol",
        "model_provider": "openai",
        "model_reasoning_effort": "high",
        "features": copy.deepcopy(EXPECTED if feature_values is None else feature_values),
    }
    return {
        "config": config,
        "origins": {"default_permissions": {
            "name": {"type": origin_type}, "version": VERSION,
        }},
        "layers": [{
            "name": {"type": "sessionFlags"}, "version": VERSION,
            "config": {"default_permissions": ":read-only"},
        }],
    }


def raw_response(identifier, result):
    return client.strict_json(json.dumps({"id": identifier, "result": result}).encode("utf-8"))


def chain(*, config_result=None, requirements_result=None):
    config_frame = raw_response(1, raw_config() if config_result is None else config_result)
    req_frame = raw_response(2, {} if requirements_result is None else requirements_result)
    sanitized_config = client.decode_response(config_frame, 1)
    sanitized_requirements = client.decode_response(req_frame, 2)
    observations = [
        {"id": 0, "sanitizedResult": client.decode_response(raw_response(0, {}), 0)},
        {"id": 1, "sanitizedResult": sanitized_config},
        {"id": 2, "method": "configRequirements/read", "responseEnvelopeValidated": True,
         "sanitizedResult": sanitized_requirements},
    ]
    profile = client.strict_json((ROOT / "profile.json").read_bytes())
    sent = [{"method": rpc["method"], "id": rpc.get("id")} for rpc in profile["rpc"]]
    return observations, sent, sanitized_config, sanitized_requirements


class SelectedFeatureMetadataR3IntegrationTests(unittest.TestCase):
    def test_requirements_null_omitted_optional_null_empty_and_typed_map_are_valid_observations(self):
        cases = (
            ("requirements-null", None, {"requirements": None}, "null"),
            ("feature-field-omitted", None, {"requirements": {}}, "missing"),
            ("feature-field-null", None, {"requirements": {"featureRequirements": None}}, "null"),
            ("feature-map-empty", None, {"requirements": {"featureRequirements": {}}}, "present"),
            ("feature-map-compatible", None, {"requirements": {"featureRequirements": EXPECTED}}, "present"),
        )
        for name, config_result, req_result, expected_state in cases:
            with self.subTest(case=name):
                observations, sent, config, requirements = chain(
                    config_result=config_result, requirements_result=req_result)
                self.assertEqual(client.assess_config(config),
                                 ("config-precheck-satisfied-awaiting-requirements", None))
                self.assertEqual(requirements["requirements"]["state"], "present"
                                 if req_result != {"requirements": None} else "null")
                if requirements["requirements"]["state"] == "present":
                    self.assertEqual(requirements["requirements"]["value"]["featureRequirements"]["state"],
                                     expected_state)
                self.assertEqual(client.final_assess(observations, sent),
                                 ("reported-config-and-requirements-observed", None))

    def test_missing_rpc_response_or_provenance_and_missing_requirements_field_are_not_positive(self):
        observations, sent, _, _ = chain(requirements_result={"requirements": None})
        self.assertEqual(client.final_assess(observations[:2], sent),
                         ("insufficient", "actual-requirements-response-required"))
        self.assertEqual(client.final_assess(observations, sent[:-1]),
                         ("insufficient", "actual-requirements-response-required"))
        no_provenance = copy.deepcopy(observations)
        no_provenance[2].pop("responseEnvelopeValidated")
        self.assertEqual(client.final_assess(no_provenance, sent),
                         ("insufficient", "actual-requirements-response-required"))
        no_method = copy.deepcopy(observations)
        no_method[2]["method"] = "other/read"
        self.assertEqual(client.final_assess(no_method, sent),
                         ("insufficient", "actual-requirements-response-required"))

        missing_body, sent, _, missing_requirements = chain(requirements_result={})
        self.assertEqual(missing_requirements["requirements"], {"state": "missing"})
        self.assertNotEqual(client.final_assess(missing_body, sent)[0],
                            "reported-config-and-requirements-observed")

    def test_malformed_response_envelopes_and_malformed_typed_maps_fail_closed(self):
        with self.assertRaises(ValueError):
            client.decode_response(client.strict_json(b'{"id":2,"result":{},"error":{"code":1}}'), 2)
        with self.assertRaises(ValueError):
            client.decode_response(client.strict_json(b'{"id":1,"id":2,"result":{}}'), 1)

        malformed_maps = (
            {"requirements": {"featureRequirements": [True]}},
            {"requirements": {"featureRequirements": {"apps": "false"}}},
        )
        for raw_result in malformed_maps:
            with self.subTest(raw_result=raw_result):
                observations, sent, _, sanitized = chain(requirements_result=raw_result)
                managed = sanitized["requirements"]["value"]["featureRequirements"]
                self.assertEqual(managed["state"], "invalid")
                self.assertNotEqual(client.final_assess(observations, sent)[0],
                                    "reported-config-and-requirements-observed")

    def test_unknown_boolean_and_known_conflict_requirements_do_not_pass(self):
        unknown = {**EXPECTED, "network_proxy": True}
        observations, sent, _, sanitized = chain(requirements_result={
            "requirements": {"featureRequirements": unknown}})
        managed = sanitized["requirements"]["value"]["featureRequirements"]
        self.assertTrue(managed["unresolved"])
        self.assertNotIn("network_proxy", json.dumps(sanitized))
        self.assertEqual(client.final_assess(observations, sent),
                         ("insufficient", "requirement-feature-unresolved"))

        conflict = {**EXPECTED, "apps": True}
        observations, sent, _, _ = chain(requirements_result={
            "requirements": {"featureRequirements": conflict}})
        self.assertEqual(client.final_assess(observations, sent),
                         ("incompatible", "managed-feature-conflict"))

    def test_selected_feature_success_does_not_override_an_independent_origin_gate(self):
        observations, sent, config, _ = chain(
            config_result=raw_config(origin_type="project"),
            requirements_result={"requirements": {"featureRequirements": EXPECTED}})
        self.assertEqual(client.assess_config(config), ("insufficient", "selected-origin-ambiguous"))
        self.assertEqual(client.final_assess(observations, sent),
                         ("insufficient", "selected-origin-ambiguous"))


if __name__ == "__main__":
    unittest.main()
