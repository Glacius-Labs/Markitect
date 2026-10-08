"""Pure selected-feature correction for a future metadata candidate.

Config.features is untyped: only eight requested fields are evaluated. Managed
featureRequirements is a schema-wide boolean map: unknown names remain unresolved.
Unaffected parser/assessment gates come from the hash-pinned archived client;
neither that client nor its terminal evidence is changed. No launch API or CLI.
"""
import hashlib
import importlib.util
from pathlib import Path


EXPECTED = {
    "apps": False, "goals": False, "hooks": False, "memories": False,
    "multi_agent": False, "plugins": False, "shell_tool": True,
    "unified_exec": True,
}
MAP_LIMIT = 256
_PRESERVED_PATH = Path(__file__).resolve().parents[1] / "evidence/s1-common-runner-policy-metadata-20261008/client.py"
_PRESERVED_SHA256 = "8b6a2032dce5468c730885906f030fcfbed3cb4f23b010e95f90592a38b0fdfb"
_PRESERVED = None


def _legacy():
    global _PRESERVED
    if hashlib.sha256(_PRESERVED_PATH.read_bytes()).hexdigest() != _PRESERVED_SHA256:
        raise ValueError("preserved-parser-pin-mismatch")
    if _PRESERVED is None:
        spec = importlib.util.spec_from_file_location("preserved_policy_parser", _PRESERVED_PATH)
        _PRESERVED = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(_PRESERVED)
    return _PRESERVED


def strict_json(raw):
    return _legacy().strict_json(raw)


def selected_features(config):
    """Classify only requested fields; never validate or publish other values."""
    scope = {"scope": "eight-requested-fields-only", "remainder": "not-evaluated"}
    if "features" not in config:
        return {**scope, "state": "missing", "code": "config-features-container-missing"}
    value = config["features"]
    if value is None:
        return {**scope, "state": "null", "code": "config-features-container-null"}
    if type(value) is not dict:
        return {**scope, "state": "invalid", "code": "config-features-container-type"}
    if len(value) > MAP_LIMIT:
        return {**scope, "state": "invalid", "code": "config-features-container-size"}
    selected = {}
    for name in EXPECTED:
        if name not in value:
            selected[name] = {"state": "missing", "code": "config-feature-field-missing"}
        elif value[name] is None:
            selected[name] = {"state": "null", "code": "config-feature-field-null"}
        elif type(value[name]) is not bool:
            selected[name] = {"state": "invalid", "code": "config-feature-field-type"}
        else:
            selected[name] = {"state": "present", "value": value[name]}
    return {**scope, "state": "present", "selected": selected}


def requirement_features(requirements):
    """Validate the whole typed map, publishing no uninterpreted field names."""
    scope = {"scope": "schema-wide-boolean-map"}
    if "featureRequirements" not in requirements:
        return {**scope, "state": "missing", "code": "requirement-features-missing"}
    value = requirements["featureRequirements"]
    if value is None:
        return {**scope, "state": "null", "code": "requirement-features-null"}
    if type(value) is not dict:
        return {**scope, "state": "invalid", "code": "requirement-features-container-type"}
    if len(value) > MAP_LIMIT:
        return {**scope, "state": "invalid", "code": "requirement-features-container-size"}
    if any(type(k) is not str for k in value):
        return {**scope, "state": "invalid", "code": "requirement-feature-field-type"}
    if any(type(v) is not bool for v in value.values()):
        return {**scope, "state": "invalid", "code": "requirement-feature-value-type"}
    unresolved = any(k not in EXPECTED for k in value)
    return {
        **scope, "state": "present", "value": {k: value[k] for k in EXPECTED if k in value},
        "unresolved": unresolved,
        "code": "requirement-feature-unresolved" if unresolved else "requirement-features-observed",
    }


def _without_features(config):
    return {k: v for k, v in config.items() if k != "features"} if type(config) is dict else config


def sanitize_config_read(result):
    """Use the real archived consumer for all unaffected config/origin fields."""
    old = _legacy()
    old.reject_provisional(result)
    old.require(type(result) is dict, "invalid-config-result")
    projected = {**result, "config": _without_features(result.get("config"))}
    layers = result.get("layers")
    if type(layers) is list and len(layers) <= 32:
        projected["layers"] = [
            {**layer, "config": _without_features(layer.get("config"))} if type(layer) is dict else layer
            for layer in layers
        ]
    origins = result.get("origins")
    if type(origins) is dict:
        projected["origins"] = {
            k: v for k, v in origins.items()
            if not (type(k) is str and k.startswith("features.") and k[9:] not in EXPECTED)
        }
    sanitized = old.sanitize_config_read(projected)
    sanitized["config"]["features"] = selected_features(result["config"])
    sanitized["origins"] = {
        k: v for k, v in sanitized["origins"].items()
        if not (k.startswith("features.") and k[9:] not in EXPECTED)
    }
    if sanitized["layers"]["state"] == "present":
        for raw, layer in zip(layers, sanitized["layers"]["value"]):
            layer["config"]["features"] = selected_features(raw["config"])
    return sanitized


def sanitize_requirements(result):
    old = _legacy()
    old.reject_provisional(result)
    old.require(type(result) is dict, "invalid-requirements-result")
    raw = result.get("requirements")
    projected = result
    if type(raw) is dict:
        projected = {**result, "requirements": {k: v for k, v in raw.items() if k != "featureRequirements"}}
    sanitized = old.sanitize_requirements(projected)
    if sanitized["requirements"]["state"] == "present":
        sanitized["requirements"]["value"]["featureRequirements"] = requirement_features(raw)
    return sanitized


def _selected_status(features):
    # An old receipt has no selected-field evidence. Never reconstruct it from
    # layer echoes, populate missing booleans, or upgrade its historical status.
    if type(features) is not dict or features.get("scope") != "eight-requested-fields-only" or features.get("remainder") != "not-evaluated":
        return "insufficient", "selected-feature-evidence-unresolved"
    if features.get("state") != "present":
        codes = {"config-features-container-missing", "config-features-container-null", "config-features-container-type", "config-features-container-size"}
        return "insufficient", features.get("code") if features.get("code") in codes else "selected-feature-evidence-unresolved"
    selected = features.get("selected")
    if type(selected) is not dict or set(selected) != set(EXPECTED):
        return "insufficient", "config-feature-field-set"
    for name, expected in EXPECTED.items():
        observed = selected[name]
        if type(observed) is not dict:
            return "insufficient", "config-feature-field-type"
        state = observed.get("state")
        if state != "present":
            return "insufficient", {"missing": "config-feature-field-missing", "null": "config-feature-field-null"}.get(state, "config-feature-field-type")
        if type(observed.get("value")) is not bool:
            return "insufficient", "config-feature-field-type"
        if observed["value"] is not expected:
            return "incompatible", "config-feature-value-conflict"
    return "selected-feature-precondition-satisfied", None


def assess(config, requirements):
    """Assess parser-derived observations; preserve every unaffected old gate."""
    features = config["config"]["features"]
    status, reason = _selected_status(features)
    if reason:
        return status, reason
    req = requirements["requirements"]
    if req["state"] == "present":
        managed = req["value"]["featureRequirements"]
        if type(managed) is not dict or managed.get("scope") != "schema-wide-boolean-map":
            return "insufficient", "requirement-feature-evidence-unresolved"
        if managed["state"] == "invalid":
            codes = {"requirement-features-container-type", "requirement-features-container-size", "requirement-feature-field-type", "requirement-feature-value-type"}
            return "insufficient", managed.get("code") if managed.get("code") in codes else "requirement-feature-evidence-unresolved"
        if managed["state"] == "present":
            if any(v is not EXPECTED[k] for k, v in managed["value"].items()):
                return "incompatible", "managed-feature-conflict"
            if managed["unresolved"]:
                return "insufficient", "requirement-feature-unresolved"
    # Normalize only values already derived and validated by this contract.
    # The archived consumer still enforces default/origin/layer, model, legacy,
    # other managed constraints and Windows gates. No positive state is invented.
    normalized = {**config, "config": {**config["config"], "features": {
        "state": "present", "value": {k: features["selected"][k]["value"] for k in EXPECTED},
    }}}
    return _legacy().assess(normalized, requirements)


def sanitize(identifier, result):
    if identifier == 1:
        return sanitize_config_read(result)
    if identifier == 2:
        return sanitize_requirements(result)
    return _legacy().sanitize(identifier, result)
