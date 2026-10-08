"""Offline tests for the narrow selected-feature adapter."""
import copy
import importlib.util
import json
from pathlib import Path
import unittest


PACKAGE = Path(__file__).resolve().parents[1]
MODULE_PATH = PACKAGE / "runtime/selected_feature_contract.py"
spec = importlib.util.spec_from_file_location("selected_feature_contract_under_test", MODULE_PATH)
contract = importlib.util.module_from_spec(spec)
spec.loader.exec_module(contract)


EXPECTED = {
    "apps": False, "goals": False, "hooks": False, "memories": False,
    "multi_agent": False, "plugins": False, "shell_tool": True,
    "unified_exec": True,
}
VERSION = "sha256:" + "a" * 64


def raw_config(features=None, *, include_features=True, origin_type="sessionFlags", layers=None):
    config = {
        "default_permissions": ":read-only",
        "approval_policy": "never",
        "model": "gpt-6.1-sol",
        "model_provider": "openai",
        "model_reasoning_effort": "high",
    }
    if include_features:
        config["features"] = copy.deepcopy(EXPECTED if features is None else features)
    layer_rows = layers if layers is not None else [{
        "name": {"type": "sessionFlags"}, "version": VERSION,
        "config": {"default_permissions": ":read-only"},
    }]
    return {
        "config": config,
        "origins": {"default_permissions": {
            "name": {"type": origin_type}, "version": VERSION,
        }},
        "layers": copy.deepcopy(layer_rows),
    }


def raw_requirements(feature_requirements=None, *, include=True):
    value = {}
    if include:
        value["featureRequirements"] = copy.deepcopy(EXPECTED if feature_requirements is None else feature_requirements)
    return {"requirements": value}


class SelectedFeatureContractTests(unittest.TestCase):
    def assess_raw(self, config, requirements=None):
        sanitized_config = contract.sanitize_config_read(config)
        sanitized_requirements = contract.sanitize_requirements(
            raw_requirements() if requirements is None else requirements)
        return contract.assess(sanitized_config, sanitized_requirements)

    def test_all_eight_requested_boolean_features_satisfy_selected_contract(self):
        raw = raw_config()
        observed = contract.sanitize_config_read(raw)
        features = observed["config"]["features"]
        self.assertEqual(features, {
            "scope": "eight-requested-fields-only", "remainder": "not-evaluated",
            "state": "present", "selected": {
                name: {"state": "present", "value": value} for name, value in EXPECTED.items()}})
        self.assertEqual(self.assess_raw(raw), ("visible-metadata-precondition-satisfied", None))

    def test_unselected_feature_values_are_not_evaluated_or_published(self):
        features = {**EXPECTED, "feature-private-a": "raw string", "feature-private-b": None,
                    "feature-private-c": {"nested": "secret value"}}
        result = contract.sanitize_config_read(raw_config(features))
        selected = result["config"]["features"]
        self.assertEqual(set(selected["selected"]), set(EXPECTED))
        self.assertEqual(selected["remainder"], "not-evaluated")
        serialized = json.dumps(result, sort_keys=True)
        for text in ("feature-private-a", "feature-private-b", "feature-private-c", "raw string", "secret value"):
            self.assertNotIn(text, serialized)
        self.assertEqual(contract.assess(result, contract.sanitize_requirements(raw_requirements()))[0],
                         "visible-metadata-precondition-satisfied")

    def test_unselected_values_in_layers_and_legacy_feature_origins_are_not_published(self):
        extras = {"network_proxy": {"private": "layer object"},
                  "powershell_shell_version": "private layer string",
                  "other_null_feature": None}
        raw = raw_config({**EXPECTED, **extras}, layers=[{
            "name": {"type": "sessionFlags"}, "version": VERSION,
            "config": {"default_permissions": ":read-only",
                       "features": {**EXPECTED, **extras}},
        }])
        raw["origins"].update({
            "features.network_proxy": {"name": {"type": "user"}, "version": VERSION,
                                        "private": "origin object"},
            "features.powershell_shell_version": {"name": {"type": "project"},
                                                   "version": VERSION,
                                                   "private": "origin string"},
        })
        sanitized = contract.sanitize_config_read(raw)
        serialized = json.dumps(sanitized, sort_keys=True)
        for text in ("network_proxy", "powershell_shell_version", "other_null_feature",
                     "layer object", "private layer string", "origin object", "origin string"):
            self.assertNotIn(text, serialized)
        self.assertEqual(contract.assess(sanitized, contract.sanitize_requirements(raw_requirements())),
                         ("visible-metadata-precondition-satisfied", None))

    def test_each_selected_key_requires_its_expected_typed_boolean(self):
        for name, expected in EXPECTED.items():
            cases = []
            missing = dict(EXPECTED)
            del missing[name]
            cases.append(("missing", missing, ("insufficient", "config-feature-field-missing")))
            null = dict(EXPECTED)
            null[name] = None
            cases.append(("null", null, ("insufficient", "config-feature-field-null")))
            string = dict(EXPECTED)
            string[name] = "true"
            cases.append(("string", string, ("insufficient", "config-feature-field-type")))
            integer = dict(EXPECTED)
            integer[name] = 1
            cases.append(("integer", integer, ("insufficient", "config-feature-field-type")))
            conflict = dict(EXPECTED)
            conflict[name] = not expected
            cases.append(("opposite", conflict, ("incompatible", "config-feature-value-conflict")))
            for case, features, assessment in cases:
                with self.subTest(feature=name, case=case):
                    self.assertEqual(self.assess_raw(raw_config(features)), assessment)

    def test_feature_container_missing_null_wrong_type_and_oversize_fail_closed(self):
        null_features = raw_config()
        null_features["config"]["features"] = None
        cases = (
            ("missing", raw_config(include_features=False), "config-features-container-missing"),
            ("null", null_features, "config-features-container-null"),
            ("wrong-type", raw_config([True]), "config-features-container-type"),
            ("oversize", raw_config({**EXPECTED, **{f"extra-{i}": False for i in range(249)}}),
             "config-features-container-size"),
        )
        for name, raw, reason in cases:
            with self.subTest(case=name):
                sanitized = contract.sanitize_config_read(raw)
                self.assertEqual(sanitized["config"]["features"]["code"], reason)
                self.assertEqual(contract.assess(sanitized, contract.sanitize_requirements(raw_requirements())),
                                 ("insufficient", reason))

    def test_managed_feature_map_requires_boolean_values_and_unknown_bools_remain_unresolved(self):
        invalid_maps = (
            {**EXPECTED, "apps": 1},
            {**EXPECTED, "unknown-private-name": "true"},
            {**EXPECTED, "unknown-private-name": None},
            {**EXPECTED, "unknown-private-name": 1},
        )
        for feature_map in invalid_maps:
            with self.subTest(feature_map=feature_map):
                requirements = contract.sanitize_requirements(raw_requirements(feature_map))
                managed = requirements["requirements"]["value"]["featureRequirements"]
                self.assertEqual(managed["state"], "invalid")
                self.assertEqual(managed["code"], "requirement-feature-value-type")
                self.assertEqual(contract.assess(contract.sanitize_config_read(raw_config()), requirements),
                                 ("insufficient", "requirement-feature-value-type"))
        unknown_boolean = {**EXPECTED, "network_proxy": True}
        requirements = contract.sanitize_requirements(raw_requirements(unknown_boolean))
        managed = requirements["requirements"]["value"]["featureRequirements"]
        self.assertEqual(managed["state"], "present")
        self.assertTrue(managed["unresolved"])
        self.assertNotIn("network_proxy", json.dumps(requirements))
        self.assertEqual(contract.assess(contract.sanitize_config_read(raw_config()), requirements),
                         ("insufficient", "requirement-feature-unresolved"))

    def test_managed_requirement_container_absence_null_type_and_size_are_preserved(self):
        cases = (
            ({"requirements": {}}, "missing", "requirement-features-missing"),
            ({"requirements": {"featureRequirements": None}}, "null", "requirement-features-null"),
            ({"requirements": {"featureRequirements": [True]}}, "invalid", "requirement-features-container-type"),
            ({"requirements": {"featureRequirements": {f"f{i}": True for i in range(257)}}},
             "invalid", "requirement-features-container-size"),
        )
        for raw, state, code in cases:
            with self.subTest(code=code):
                result = contract.sanitize_requirements(raw)
                managed = result["requirements"]["value"]["featureRequirements"]
                self.assertEqual((managed["state"], managed.get("code")), (state, code))

    def test_known_managed_conflict_blocks_even_with_unknown_boolean_key(self):
        requirements = {**EXPECTED, "apps": True, "unknown-feature": False}
        sanitized = contract.sanitize_requirements(raw_requirements(requirements))
        self.assertTrue(sanitized["requirements"]["value"]["featureRequirements"]["unresolved"])
        self.assertEqual(contract.assess(contract.sanitize_config_read(raw_config()), sanitized),
                         ("incompatible", "managed-feature-conflict"))

    def test_duplicate_json_warning_and_origin_gates_use_real_adapter_paths(self):
        with self.assertRaisesRegex(ValueError, "duplicate-json-key"):
            contract.strict_json(b'{"features":true,"features":false}')
        with self.assertRaisesRegex(ValueError, "warning-or-provisional-result"):
            contract.sanitize_config_read({**raw_config(), "warning": "private warning text"})
        with self.assertRaisesRegex(ValueError, "warning-or-provisional-result"):
            contract.sanitize_requirements({"requirements": {"warning": "private warning text"}})

        ambiguous = contract.sanitize_config_read(raw_config(origin_type="project"))
        self.assertEqual(contract.assess(ambiguous, contract.sanitize_requirements(raw_requirements())),
                         ("insufficient", "selected-origin-ambiguous"))
        duplicate_layer = {"name": {"type": "sessionFlags"}, "version": VERSION,
                           "config": {"default_permissions": ":read-only"}}
        ambiguous_layers = contract.sanitize_config_read(raw_config(layers=[duplicate_layer, duplicate_layer]))
        self.assertEqual(contract.assess(ambiguous_layers, contract.sanitize_requirements(raw_requirements())),
                         ("insufficient", "selected-origin-layer-ambiguous"))

    def test_historical_receipt_stays_insufficient_without_raw_reconstruction(self):
        path = PACKAGE / "evidence/s1-common-runner-policy-metadata-20261008/run-1/sanitized-result.json"
        receipt = contract.strict_json(path.read_bytes())
        self.assertEqual(receipt["status"], "insufficient")
        config = receipt["observations"][1]["sanitizedResult"]
        self.assertNotIn("scope", config["config"]["features"])
        requirements = {"requirements": {"state": "missing"}}
        self.assertEqual(contract.assess(config, requirements),
                         ("insufficient", "selected-feature-evidence-unresolved"))
        self.assertEqual(receipt["status"], "insufficient")


if __name__ == "__main__":
    unittest.main()
