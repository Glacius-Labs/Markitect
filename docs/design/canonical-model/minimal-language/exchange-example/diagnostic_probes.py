"""Coordinates may be unavailable without weakening semantic link validation."""
from __future__ import annotations

import copy
from unittest.mock import patch

from body_links import parse_body_links
from diagnostic_coordinates import CoordinateMappingError


def diagnostic_probes(baseline, parse_yaml, check_links, coordinate_report, encode, assess):
    results = {}
    project, model = "example/connector/project", baseline.entry_model
    sample = copy.deepcopy(baseline)
    source = {}
    body = parse_yaml("body: >-\n  [[api]] [[api]]\n  [[onprem]]\n", body_source=source, file="coordinates.yaml")["body"]
    sample[project]["definitions"]["list-documents"]["body"] = body
    sample[project]["body_sources"]["list-documents"] = source
    before = copy.deepcopy(sample)
    expected = encode(sample, model)
    records = parse_body_links(body)
    results["diagnostic_semantic_parser_preserves_order_and_duplicates_without_offsets"] = (
        [record["target"] for record in records] == ["api", "api", "onprem"]
        and all(not {"startByte", "endByte", "source"} & record.keys() for record in records))
    results["diagnostic_exact_profile_still_passes"] = coordinate_report(sample)["status"] == "passed"

    missing = copy.deepcopy(sample)
    for package in missing.values():
        package["body_sources"] = {}
    results["diagnostic_missing_yaml_metadata_does_not_change_bytes"] = encode(missing, model) == expected
    results["diagnostic_missing_yaml_metadata_reported_separately"] = coordinate_report(missing)["status"] == "not_completed"

    stale = copy.deepcopy(sample)
    stale[project]["body_sources"]["list-documents"]["decoded"] = "outdated"
    results["diagnostic_stale_source_does_not_change_bytes"] = encode(stale, model) == expected
    results["diagnostic_stale_source_not_given_invented_coordinates"] = (
        coordinate_report(stale)["status"] == "not_completed"
        and all("source" not in record for record in coordinate_report(stale)["occurrences"]
                if record["definition"]["id"] == "list-documents"))

    for label, function, error in (
        ("inline_mapping", "inline_positions", CoordinateMappingError("forced inline mapping gap")),
        ("yaml_mapping", "yaml_body_source", CoordinateMappingError("forced YAML mapping gap")),
        ("coordinate_memory", "inline_positions", MemoryError("coordinate-only memory exhaustion")),
    ):
        with patch(f"diagnostic_coordinates.{function}", side_effect=error) as mapping:
            actual = encode(sample, model)
            results[f"diagnostic_{label}_not_called_by_canonical_projection"] = mapping.call_count == 0 and actual == expected
            record = coordinate_report(sample)
            results[f"diagnostic_{label}_is_independent_incomplete_result"] = (
                record["status"] == "not_completed" and bool(record["diagnostics"]))
            if label == "inline_mapping":
                semantic = check_links(sample)
                results["diagnostic_mapping_gap_keeps_all_resolved_occurrences"] = [
                    item["target"]["id"] for item in semantic if item["definition"]["id"] == "list-documents"
                ] == ["api", "api", "onprem"]
                for name, invalid_body, expected_status in (
                    ("missing_target", "[[absent]]", "failed"),
                    ("invalid_markup", "[[api", "failed"),
                    ("parser_budget", "> " * 80 + "[[api]]", "not_completed"),
                ):
                    invalid = copy.deepcopy(sample)
                    invalid[project]["definitions"]["list-documents"]["body"] = invalid_body
                    outcome, data = assess(invalid, model)
                    results[f"diagnostic_gap_does_not_hide_{name}"] = outcome["status"] == expected_status and data is None
    results["diagnostic_phases_preserve_source_metadata"] = sample == before and sample.models == before.models
    literal = copy.deepcopy(sample)
    literal[project]["definitions"]["list-documents"]["body"] = "[[api|A\0\ufffdB]]"
    literal[project]["body_sources"] = {}
    literal_bytes = encode(literal, model)
    with patch("diagnostic_coordinates.inline_positions", side_effect=CoordinateMappingError("forced diagnostic mapping gap")) as mapping:
        records = check_links(literal)
        results["diagnostic_nul_preservation_independent_of_coordinate_mapping"] = (
            mapping.call_count == 0 and encode(literal, model) == literal_bytes
            and [record["display"] for record in records if record["definition"]["id"] == "list-documents"] == ["A\0\ufffdB"])
        results["diagnostic_nul_mapping_gap_reported_separately"] = coordinate_report(literal)["status"] == "not_completed"
    return results
