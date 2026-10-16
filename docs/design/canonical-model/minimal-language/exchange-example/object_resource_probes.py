"""Finite recursive objects, uninhabitable contracts and budget outcomes."""
from __future__ import annotations

import copy
import tempfile
from pathlib import Path
from unittest.mock import patch

from name_profile import ProbeLimitError
from resources import overall_status


def object_resource_probes(baseline, validate, encode, assess, export, parse_yaml, check_cycles):
    results = {}
    package = "example/connector/project"
    core_import = {"package": "markitect/core", "as": "Core"}
    model = baseline.entry_model

    def recursive_sample(list_values=False):
        sample = copy.deepcopy(baseline)
        definitions = sample[package]["definitions"]
        field = "children" if list_values else "branch"
        definitions["Tree"] = {"id": "Tree", "kind": "Kind", "body": "probe", "imports": [core_import],
                               "spec": {"properties": [field]}}
        definitions[field] = {"id": field, "kind": "Property", "body": "probe", "imports": [core_import],
                              "spec": {"type": "object", "minCount": 0, "maxCount": -1 if list_values else 1,
                                       "properties": [field]}}
        value = [{"children": [{}]}, {}] if list_values else {"branch": {"branch": {}}}
        definitions["tree-one"] = {"id": "tree-one", "kind": "Tree", "body": "probe", "spec": {field: value}}
        return sample

    def rejected(action, fragment):
        try:
            action()
        except ProbeLimitError:
            return False
        except ValueError as error:
            return fragment in str(error)
        return False

    sample = recursive_sample()
    before = copy.deepcopy(sample)
    validate(sample)
    canonical = encode(sample, model)
    results["object_optional_self_recursion_has_finite_values"] = bool(canonical)
    results["object_recursive_validation_does_not_mutate_definitions"] = sample == before and sample.models == before.models
    lists = recursive_sample(True)
    validate(lists)
    list_bytes = encode(lists, model)
    leaf_defaults = copy.deepcopy(lists)
    leaf_defaults[package]["definitions"]["tree-one"]["spec"]["children"][0]["children"][0]["children"] = []
    results["object_recursive_list_terminates_at_empty_or_missing"] = encode(leaf_defaults, model) == list_bytes
    extra_child = copy.deepcopy(lists)
    extra_child[package]["definitions"]["tree-one"]["spec"]["children"].append({})
    results["object_recursive_value_list_multiplicity_affects_bytes"] = encode(extra_child, model) != list_bytes
    mandatory = copy.deepcopy(sample)
    del mandatory[package]["definitions"]["tree-one"]
    mandatory[package]["definitions"]["branch"]["spec"]["minCount"] = 1
    results["object_required_self_cycle_invalid_even_without_instances"] = rejected(lambda: validate(mandatory), "mandatory object schema cycle")
    record, data = assess(mandatory, model)
    results["object_required_cycle_has_no_digest"] = record["status"] == "failed" and "digest" not in record and data is None

    mutual = recursive_sample()
    definitions = mutual[package]["definitions"]
    definitions["branch"]["spec"]["properties"] = ["other"]
    definitions["other"] = {"id": "other", "kind": "Property", "body": "probe", "imports": [core_import],
                            "spec": {"type": "object", "minCount": 1, "maxCount": 1, "properties": ["branch"]}}
    definitions["tree-one"]["spec"] = {"branch": {"other": {"branch": {"other": {}}}}}
    validate(mutual)
    results["object_mutual_cycle_with_optional_edge_is_allowed"] = bool(encode(mutual, model))
    definitions["branch"]["spec"]["minCount"] = 1
    del definitions["tree-one"]
    results["object_mutual_required_cycle_is_rejected"] = rejected(lambda: validate(mutual), "mandatory object schema cycle")

    hidden = copy.deepcopy(mutual)
    hidden[package]["definitions"]["Tree"]["spec"]["properties"] = []
    results["object_unused_required_cycle_still_invalid"] = rejected(lambda: validate(hidden), "mandatory object schema cycle")
    empty_only = recursive_sample(True)
    empty_only[package]["definitions"]["children"]["spec"]["maxCount"] = 0
    empty_only[package]["definitions"]["tree-one"]["spec"] = {"children": []}
    results["object_zero_cardinality_recursion_needs_no_values"] = bool(encode(empty_only, model, max_object_depth=0))

    references = copy.deepcopy(baseline)
    definitions = references[package]["definitions"]
    definitions["Link"] = {"id": "Link", "kind": "Kind", "body": "probe", "imports": [core_import], "spec": {"properties": ["next"]}}
    definitions["next"] = {"id": "next", "kind": "Property", "body": "probe", "imports": [core_import],
                           "spec": {"type": "reference", "minCount": 1, "maxCount": 1, "target": "Link"}}
    definitions["link-one"] = {"id": "link-one", "kind": "Link", "body": "probe", "spec": {"next": "link-two"}}
    definitions["link-two"] = {"id": "link-two", "kind": "Link", "body": "probe", "spec": {"next": "link-one"}}
    results["object_ordinary_reference_cycles_are_not_unfolded"] = bool(encode(references, model, max_object_depth=0))

    wrong_value = copy.deepcopy(sample)
    wrong_value[package]["definitions"]["tree-one"]["spec"]["branch"]["branch"] = {"unknown": "bad"}
    results["object_recursive_values_retain_closed_field_validation"] = rejected(lambda: validate(wrong_value), "unknown field")
    results["object_recursive_failure_preserves_source_inventory"] = not any(identifier.startswith("__object__") for identifier in wrong_value[package]["definitions"])

    limited, data = assess(sample, model, max_object_depth=1)
    results["resource_object_depth_budget_is_not_completed"] = limited["status"] == "not_completed" and "object-depth budget 1" in limited["diagnostics"][0]
    results["resource_incomplete_model_has_no_digest_or_bytes"] = "digest" not in limited and data is None
    with tempfile.TemporaryDirectory(prefix="markitect-incomplete-export-") as temporary:
        output = Path(temporary) / "existing.json"
        output.write_bytes(b"previous artifact")
        result = export(sample, model, output, max_object_depth=1)
        results["resource_incomplete_export_preserves_existing_artifact"] = result["status"] == "not_completed" and output.read_bytes() == b"previous artifact"
        absent = Path(temporary) / "absent" / "output.json"
        result = export(sample, model, absent, max_object_depth=1)
        results["resource_incomplete_export_creates_no_artifact"] = result["status"] == "not_completed" and not absent.parent.exists()
    high, high_data = assess(sample, model, max_object_depth=8)
    higher, higher_data = assess(sample, model, max_object_depth=16)
    results["resource_completed_budgets_yield_same_bytes_and_digest"] = high["status"] == higher["status"] == "passed" and high_data == higher_data == canonical and high["digest"] == higher["digest"]
    results["resource_budget_abort_preserves_all_source_values"] = sample == before and sample.models == before.models
    core, core_bytes = assess(sample, "markitect-core", max_object_depth=0)
    results["resource_unaffected_core_scope_can_still_complete"] = core["status"] == "passed" and core_bytes is not None
    for exception, fragment in ((RecursionError("probe"), "recursion limit"), (MemoryError("probe"), "memory exhaustion")):
        with patch("yaml.compose", side_effect=exception):
            try:
                parse_yaml("value: text")
            except ProbeLimitError as error:
                results[f"resource_yaml_{type(exception).__name__}_is_incomplete"] = fragment in str(error)
            else:
                results[f"resource_yaml_{type(exception).__name__}_is_incomplete"] = False
    results["resource_incomplete_projection_cannot_pass_overall"] = overall_status("passed", {model: limited, "markitect-core": core}, "passed", "passed") == "not_completed"
    results["resource_loading_abort_without_tests_is_incomplete"] = overall_status("not_completed", {}, "not_completed", "not_completed") == "not_completed"
    results["resource_proven_error_is_not_erased_by_other_abort"] = overall_status("failed", {model: limited}, "not_completed", "passed") == "failed"
    cyclic = copy.deepcopy(baseline)
    cyclic.models[model]["manifest"]["version"] = "1.0"
    cyclic.models["markitect-core"]["manifest"]["dependencies"][model] = "1.0"
    record, data = assess(cyclic, model)
    results["audit_manifest_cycle_fails_without_digest"] = record["status"] == "failed" and "digest" not in record and data is None
    self_dependent = copy.deepcopy(baseline)
    self_dependent.models[model]["manifest"]["version"] = "1.0"
    self_dependent.models[model]["manifest"]["dependencies"][model] = "1.0"
    record, data = assess(self_dependent, model)
    results["audit_manifest_self_dependency_fails_without_digest"] = record["status"] == "failed" and "digest" not in record and data is None

    # Required schema depth does not consume the actual-value depth budget.
    # This graph exceeds ordinary Python recursion depth but is traversed iteratively.
    long_graph = copy.deepcopy(baseline)
    definitions = long_graph[package]["definitions"]
    for index in range(1100):
        identifier = f"level{index}"
        definitions[identifier] = {"id": identifier, "kind": "Property", "body": "probe", "imports": [core_import],
                                   "spec": {"type": "object", "minCount": 1, "maxCount": 1,
                                            "properties": [f"level{index + 1}"] if index < 1099 else []}}
    check_cycles(long_graph)
    results["object_large_finite_schema_graph_needs_no_recursive_unfolding"] = True
    return results
