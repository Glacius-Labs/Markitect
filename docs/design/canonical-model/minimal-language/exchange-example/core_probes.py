"""Equivalent Core author forms and fixed-seed invariants through full validation."""
from __future__ import annotations

import copy


def core_equivalence_probes(baseline, encode):
    results = {}
    core, model = "markitect/core", "markitect-core"
    original = encode(baseline, model)

    def changed(action):
        trial = copy.deepcopy(baseline)
        action(trial[core]["definitions"])
        return trial

    def equivalent(label, trial, expected=original):
        before = copy.deepcopy(trial)
        results[f"core_semantic_{label}_same_bytes"] = encode(trial, model) == expected
        results[f"core_semantic_{label}_sources_preserved"] = trial == before and trial.models == before.models

    for identifier, definition in baseline[core]["definitions"].items():
        equivalent(f"{identifier}_full_kind", changed(
            lambda d, i=identifier, k=definition["kind"]: d[i].update(kind=f"{core}/{k}")))
        if "properties" in definition["spec"]:
            equivalent(f"{identifier}_full_field_targets", changed(
                lambda d, i=identifier: d[i]["spec"].update(properties=[f"{core}/{ref}" for ref in reversed(d[i]["spec"]["properties"])])))
        if "target" in definition["spec"]:
            equivalent(f"{identifier}_full_target", changed(
                lambda d, i=identifier: d[i]["spec"].update(target=f"{core}/{d[i]['spec']['target']}")))

    def qualify(definitions, alias):
        for definition in definitions.values():
            role = definition["kind"]
            definition["kind"] = f"{alias}.{role}" if alias else f"{core}/{role}"
            definition["imports"] = [{"package": core, "as": alias}] if alias else []
            spec = definition["spec"]
            if role == "Property":
                spec["name"] = definition["id"]
            if "target" in spec:
                spec["target"] = f"{alias}.{spec['target']}" if alias else f"{core}/{spec['target']}"
            if "properties" in spec:
                spec["properties"] = [f"{alias}.{ref}" if alias else f"{core}/{ref}" for ref in reversed(spec["properties"])]
            if "values" in spec:
                spec["values"].reverse()

    equivalent("all_full_references_and_defaults", changed(lambda d: qualify(d, None)))
    equivalent("all_alias_references_and_defaults", changed(lambda d: qualify(d, "Core")))
    equivalent("renamed_alias", changed(lambda d: qualify(d, "Basis")))

    library = copy.deepcopy(baseline)
    auxiliary = {"id": "Auxiliary", "kind": f"{core}/Kind", "body": "An auxiliary type.",
                 "imports": [{"package": core}], "spec": {}}
    directory = library[core]["directory"].parent / "auxiliary"
    library["auxiliary"] = {"directory": directory, "definitions": {"Auxiliary": auxiliary}, "paths": {},
                            "body_sources": {}, "model": model, "locations": {model: directory}}
    library_bytes = encode(library, model)
    library[core]["definitions"]["Kind"]["imports"] = [{"package": "auxiliary", "as": "Aux"}]
    equivalent("valid_unused_import", library, library_bytes)

    cases = (
        ("wrong_case_kind", lambda d: d["Kind"].update(kind=f"{core}/kind"), "wrong-case reference"),
        ("wrong_case_alias", lambda d: d["Kind"].update(kind="core.Kind", imports=[{"package": core, "as": "Core"}]), "wrong-case reference"),
        ("wrong_case_field_target", lambda d: d["Kind"]["spec"].update(properties=[f"{core}/Properties"]), "wrong-case reference"),
        ("missing_target", lambda d: d["properties"]["spec"].update(target=f"{core}/Absent"), "unresolved"),
        ("different_kind_target", lambda d: d["properties"].update(kind=f"{core}/Kind"), "Core definitions differ"),
        ("different_reference_target", lambda d: d["properties"]["spec"].update(target=f"{core}/Kind"), "Core definitions differ"),
        ("duplicate_field_target", lambda d: d["Kind"]["spec"].update(properties=["properties", f"{core}/properties"]), "Core definitions differ"),
        ("missing_field_target", lambda d: d["Property"]["spec"]["properties"].pop(), "Core definitions differ"),
        ("different_enum_value", lambda d: d["type"]["spec"]["values"].append("custom"), "Core definitions differ"),
        ("duplicate_enum_value", lambda d: d["type"]["spec"]["values"].append("string"), "Core definitions differ"),
        ("different_cardinality", lambda d: d["name"]["spec"].update(maxCount=2), "Core definitions differ"),
        ("boolean_not_integer", lambda d: d["name"]["spec"].update(maxCount=True), "expected integer"),
        ("different_effective_name", lambda d: d["name"]["spec"].update(name="otherName"), "Core definitions differ"),
        ("different_body", lambda d: d["Kind"].update(body=d["Kind"]["body"] + " "), "Core definitions differ"),
        ("additional_hull_field", lambda d: d["Kind"].update(extra="data"), "Core definitions differ"),
        ("additional_spec_field", lambda d: d["Kind"]["spec"].update(extra=[]), "Core definitions differ"),
        ("forbidden_list_not_defaulted", lambda d: d["name"]["spec"].update(properties=[]), "Core definitions differ"),
        ("duplicate_import", lambda d: d["Kind"].update(imports=[{"package": core}, {"package": core}]), "duplicate"),
        ("undeclared_alias", lambda d: d["Kind"].update(kind="Core.Kind"), "unresolved"),
        ("missing_definition", lambda d: d.pop("name"), "core must contain exactly"),
    )
    for label, action, fragment in cases:
        try:
            encode(changed(action), model)
        except ValueError as error:
            results[f"core_semantic_{label}_rejected"] = fragment in str(error)
        else:
            results[f"core_semantic_{label}_rejected"] = False

    # Even exact spelling must not hide another visible, case-colliding target.
    ambiguous = copy.deepcopy(baseline)
    other = copy.deepcopy(ambiguous["example/connector/project"])
    other.update(model=model, locations={model: other["directory"]}, paths={}, body_sources={}, definitions={"Kind": {
        "id": "Kind", "kind": f"{core}/Kind", "imports": [{"package": core}], "body": "A different type.", "spec": {}}})
    ambiguous["other"] = other
    ambiguous[core]["definitions"]["Kind"]["imports"] = [{"package": "other", "as": "Other"}]
    try:
        encode(ambiguous, model)
    except ValueError as error:
        results["core_semantic_ambiguous_short_kind_rejected"] = "ambiguous" in str(error)
    else:
        results["core_semantic_ambiguous_short_kind_rejected"] = False
    ambiguous[core]["definitions"]["Kind"]["kind"] = "Other.Kind"
    try:
        encode(ambiguous, model)
    except ValueError as error:
        results["core_semantic_same_local_id_in_other_package_is_different_target"] = "Core definitions differ" in str(error)
    else:
        results["core_semantic_same_local_id_in_other_package_is_different_target"] = False
    ambiguous[core]["definitions"]["Kind"]["kind"] = f"{core}/Kind"
    without_import = copy.deepcopy(ambiguous)
    without_import[core]["definitions"]["Kind"]["imports"] = []
    equivalent("qualified_kind_with_ambiguous_short_name", ambiguous, encode(without_import, model))
    return results
