"""Canonical-byte equivalence, distinction and fixed byte-vector probes."""
from __future__ import annotations

import copy
import json
from decimal import Decimal, localcontext

from canonical_bytes import decimal_parts, digest, integer_text, json_bytes


def canonical_probes(baseline, source, encode_model, parse_yaml, root):
    results = {}
    project, vocab, core = "example/connector/project", "example/connector/vocabulary", "markitect/core"
    model = baseline.entry_model

    def rejected(action, fragment):
        try:
            action()
        except ValueError as error:
            return fragment in str(error)
        return False

    sample = copy.deepcopy(baseline)
    defs = sample[project]["definitions"]
    imports = [{"package": core, "as": "Core"}]
    defs["CanonicalProbe"] = {"id": "CanonicalProbe", "kind": "Kind", "body": "probe", "imports": imports,
                               "spec": {"properties": ["amount", "count", "flag", "label", "items", "details"]}}
    for identifier, kind, minimum, maximum in (("amount", "number", 1, 1), ("count", "integer", 1, 1),
                                              ("flag", "boolean", 0, 1), ("label", "string", 0, 1),
                                              ("items", "string", 0, -1), ("details", "object", 0, 1),
                                              ("labels", "string", 0, -1), ("nestedName", "string", 0, 1)):
        spec = {"type": kind, "minCount": minimum, "maxCount": maximum}
        if kind == "object":
            spec["properties"] = ["labels", "nestedName"]
        defs[identifier] = {"id": identifier, "kind": "Property", "body": "probe", "imports": imports, "spec": spec}
    defs["canonical-probe"] = {"id": "canonical-probe", "kind": "CanonicalProbe", "body": "probe", "spec": {
        "amount": 42, "count": 42, "flag": True, "label": "u\u0308 📄", "items": ["a", "b"], "details": {}}}

    def data(packages=sample):
        return encode_model(packages, model)

    def changed(action):
        trial = copy.deepcopy(sample)
        action(trial)
        return trial

    def values(packages):
        return packages[project]["definitions"]["canonical-probe"]["spec"]

    base = data()
    short_version = changed(lambda t: t.models[model]["manifest"].update(version="1.0"))
    full_version = changed(lambda t: t.models[model]["manifest"].update(version="1.0.0"))
    results["canonical_exact_version_labels_not_padded"] = data(short_version) != data(full_version)
    for label in ("x", "1.x", "release.x"):
        labelled = changed(lambda t: t.models[model]["manifest"].update(version=label))
        encoded = data(labelled)
        results[f"canonical_x_label_{label}_preserved_exactly"] = json.loads(encoded)["model"]["version"] == label
        results[f"canonical_x_label_{label}_differs_from_numeric_version"] = encoded != data(full_version)
    results["canonical_model_id_common_grammar_enforced"] = rejected(
        lambda: data(changed(lambda t: t.models[model]["manifest"].update(id="wrong_id"))), "lowercase Unicode label sections")
    results["canonical_version_label_common_grammar_enforced"] = rejected(
        lambda: data(changed(lambda t: t.models[model]["manifest"].update(version="1..0"))), "lowercase Unicode label sections")
    tree = json.loads(base)
    spec = tree["packages"][project]["canonical-probe"]["spec"]
    results["canonical_typed_integer_number_and_boolean"] = spec["count"] == ["integer", "42"] and spec["amount"] == ["number", "42", "0"] and spec["flag"] == ["boolean", True]
    results["canonical_json_independent_library_encoding_agrees"] = base == json.dumps(tree, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode("utf-8")
    results["canonical_no_bom_or_final_newline"] = not base.startswith(b"\xef\xbb\xbf") and base.endswith(b"}")
    before = copy.deepcopy(sample)
    data()
    results["canonical_sources_and_manifests_preserved"] = sample == before and sample.models == before.models

    padded_body = " \t\u00a0\u2003Beschreibung.\u3000\n "
    padded = changed(lambda t: t[project]["definitions"]["canonical-probe"].update(body=padded_body))
    padded_bytes = data(padded)
    results["canonical_body_surrounding_white_space_preserved"] = (
        json.loads(padded_bytes)["packages"][project]["canonical-probe"]["body"] == padded_body
        and padded[project]["definitions"]["canonical-probe"]["body"] == padded_body)
    unpadded = changed(lambda t: t[project]["definitions"]["canonical-probe"].update(body="Beschreibung."))
    results["canonical_body_surrounding_white_space_affects_digest"] = digest(padded_bytes) != digest(data(unpadded))
    whitespace_only = changed(lambda t: t[project]["definitions"]["canonical-probe"].update(body=" \t\u00a0\n"))
    results["canonical_white_space_only_body_rejected_before_projection"] = rejected(
        lambda: data(whitespace_only), "outside Unicode White_Space")
    nul_body = "[[api|A\0B]]"
    nul = changed(lambda t: t[project]["definitions"]["canonical-probe"].update(body=nul_body))
    nul_bytes = data(nul)
    results["canonical_link_display_nul_keeps_original_body"] = (
        json.loads(nul_bytes)["packages"][project]["canonical-probe"]["body"] == nul_body
        and nul[project]["definitions"]["canonical-probe"]["body"] == nul_body)
    replacement = changed(lambda t: t[project]["definitions"]["canonical-probe"].update(body="[[api|A\ufffdB]]"))
    results["canonical_link_display_nul_distinct_from_replacement_character"] = digest(nul_bytes) != digest(data(replacement))

    for label, action in (
        ("number_decimal_and_integer_equivalent", lambda t: values(t).update(amount=Decimal("42.0"))),
        ("number_exponent_equivalent", lambda t: values(t).update(amount=Decimal("4.2e1"))),
        ("property_default_name_equivalent", lambda t: t[project]["definitions"]["amount"]["spec"].update(name="amount")),
        ("kind_declaration_order_irrelevant", lambda t: t[project]["definitions"]["CanonicalProbe"]["spec"]["properties"].reverse()),
        ("object_declaration_order_irrelevant", lambda t: t[project]["definitions"]["details"]["spec"]["properties"].reverse()),
        ("enum_declaration_order_irrelevant", lambda t: t[vocab]["definitions"]["location"]["spec"]["values"].reverse()),
        ("structural_full_reference_equivalent", lambda t: t[project]["definitions"]["list-documents"].update(kind=f"{vocab}/Exchange")),
        ("structural_own_full_reference_equivalent", lambda t: t[project]["definitions"]["list-documents"]["spec"].update(requester=f"{project}/api")),
        ("kind_own_full_reference_equivalent", lambda t: t[project]["definitions"]["canonical-probe"].update(kind=f"{project}/CanonicalProbe")),
        ("unused_alias_change_irrelevant", lambda t: t[project]["definitions"]["list-documents"]["imports"][0].update({"as": "Wire"})),
        ("unused_import_irrelevant", lambda t: t[project]["definitions"]["list-documents"]["imports"].append({"package": core, "as": "Core"})),
        ("nested_optional_list_empty_equivalent", lambda t: values(t)["details"].update(labels=[])),
        ("physical_root_not_content", lambda t: t.models[model]["manifest"].update(root="relocated")),
    ):
        results[f"canonical_{label}"] = data(changed(action)) == base
    for label, action in (
        ("body_text_significant", lambda t: t[project]["definitions"]["canonical-probe"].update(body="**probe**")),
        ("title_presence_significant", lambda t: t[project]["definitions"]["canonical-probe"].update(title="")),
        ("data_unicode_not_normalized", lambda t: values(t).update(label="ü 📄")),
        ("list_order_significant", lambda t: values(t)["items"].reverse()),
        ("list_duplicates_significant", lambda t: values(t).update(items=["a", "a", "b"])),
        ("number_change_significant", lambda t: values(t).update(amount=Decimal("42.0000000000000000001"))),
        ("local_version_presence_significant", lambda t: t.models[model]["manifest"].update(version="1.0.0")),
        ("object_presence_significant", lambda t: values(t).pop("details")),
    ):
        results[f"canonical_{label}"] = data(changed(action)) != base
    absent = changed(lambda t: values(t).pop("items"))
    empty = changed(lambda t: values(t).update(items=[]))
    results["canonical_optional_absent_empty_list_equivalent"] = data(absent) == data(empty)

    dotted_field = changed(lambda t: t[project]["definitions"]["nestedName"]["spec"].update(name="nested.name"))
    results["canonical_dotted_object_field_rejected_before_projection"] = rejected(
        lambda: data(dotted_field), "must not contain dots")
    dotted_value = changed(lambda t: values(t).update(label="plain.value"))
    results["canonical_dots_in_string_values_preserved"] = json.loads(data(dotted_value))["packages"][project]["canonical-probe"]["spec"]["label"] == ["string", "plain.value"]
    for label, action, fragment in (
        ("wrong_case_not_corrected", lambda t: t[project]["definitions"]["canonical-probe"].update(kind="canonicalProbe"), "wrong-case reference"),
        ("title_type_rejected", lambda t: t[project]["definitions"]["canonical-probe"].update(title=42), "title must be text"),
        ("unknown_field_rejected", lambda t: values(t).update(unknown=[]), "unknown field"),
        ("decimal_not_integer", lambda t: values(t).update(count=Decimal("42.0")), "expected integer"),
        ("nan_rejected", lambda t: values(t).update(amount=Decimal("NaN")), "exact finite number"),
    ):
        results[f"canonical_{label}"] = rejected(lambda: data(changed(action)), fragment)

    relocated = changed(lambda t: None)
    for package in relocated.values():
        package["paths"] = {identifier: root / "moved" / path.name for identifier, path in package["paths"].items()}
        for origin in package["body_sources"].values():
            origin["file"] = "different-provenance.yaml"
    results["canonical_filenames_and_source_positions_excluded"] = data(relocated) == base
    empty_package = changed(lambda t: None)
    empty_package["empty"] = {"directory": root / "empty", "definitions": {}, "paths": {}, "body_sources": {},
                              "model": None, "locations": {model: root / "empty"}}
    results["canonical_empty_packages_excluded"] = data(empty_package) == base
    empty_package[project]["definitions"]["canonical-probe"]["imports"] = [{"package": "empty"}]
    results["canonical_empty_package_import_never_produces_digest"] = rejected(
        lambda: data(empty_package), "no own definitions")
    del empty_package["empty"]
    results["canonical_removed_empty_package_import_still_rejected"] = rejected(
        lambda: data(empty_package), "is absent")

    alias_body = changed(lambda t: t[project]["definitions"]["list-documents"].update(body="[[Connector.Exchange|X]]"))
    rewritten = copy.deepcopy(alias_body)
    rewritten[project]["definitions"]["list-documents"]["imports"][0]["as"] = "Wire"
    rewritten[project]["definitions"]["list-documents"]["body"] = "[[Wire.Exchange|X]]"
    results["canonical_alias_spelling_in_body_remains_significant"] = data(alias_body) != data(rewritten)

    body_target = changed(lambda t: None)
    alt = "example/alternate"
    body_target[alt] = {"directory": root / "alternate", "paths": {}, "body_sources": {}, "model": model,
                        "locations": {model: root / "alternate"}, "definitions": {"Container": {
                            "id": "Container", "kind": "Kind", "imports": imports, "body": "alternate", "spec": {}}}}
    instance = body_target[project]["definitions"]["canonical-probe"]
    instance.update(body="[[Other.Container]]", imports=[{"package": vocab, "as": "Other"}])
    first = data(body_target)
    instance["imports"][0]["package"] = alt
    results["canonical_body_targets_bound_even_when_text_unchanged"] = data(body_target) != first

    ordinary = changed(lambda t: None)
    ordinary[project]["definitions"]["CanonicalProbe"]["spec"]["properties"] = ["properties"]
    values(ordinary).clear()
    values(ordinary)["properties"] = ["items", "labels"]
    first = data(ordinary)
    values(ordinary)["properties"].reverse()
    results["canonical_ordinary_shared_properties_list_keeps_order"] = data(ordinary) != first

    source_core = encode_model(source, "markitect-core")
    results["canonical_valid_dependency_hash_independent_of_invalid_entry"] = source_core == encode_model(baseline, "markitect-core")
    results["canonical_invalid_source_entry_has_no_digest"] = rejected(lambda: encode_model(source, source.entry_model), "invalid source model")
    core_default = copy.deepcopy(baseline)
    for definition in core_default[core]["definitions"].values():
        if definition["kind"] == "Property":
            definition["spec"]["name"] = definition["id"]
    results["canonical_fixed_core_explicit_default_names_equivalent"] = encode_model(core_default, "markitect-core") == source_core
    for identifier in sorted(baseline[core]["definitions"]):
        explicit = copy.deepcopy(baseline)
        explicit[core]["definitions"][identifier]["imports"] = []
        results[f"canonical_core_{identifier}_empty_imports_equivalent"] = encode_model(explicit, "markitect-core") == source_core
    explicit = copy.deepcopy(core_default)
    for definition in explicit[core]["definitions"].values():
        definition["imports"] = []
    before_explicit = copy.deepcopy(explicit)
    results["canonical_core_all_empty_imports_with_default_names_equivalent"] = encode_model(explicit, "markitect-core") == source_core
    results["canonical_core_import_default_does_not_rewrite_sources"] = explicit == before_explicit
    for label, value, fragment in (("null", None, "imports must"), ("mapping", {}, "imports must"),
                                   ("string", "", "imports must"), ("boolean", False, "imports must"),
                                   ("string_list", ["markitect/core"], "imports must"),
                                   ("nonempty", [{"package": "absent/package"}], "is absent")):
        invalid = copy.deepcopy(baseline)
        invalid[core]["definitions"]["Kind"]["imports"] = value
        results[f"canonical_core_{label}_imports_not_discarded"] = rejected(
            lambda t=invalid: encode_model(t, "markitect-core"), fragment)
    golden = (root / "canonical/core.json").read_bytes()
    results["canonical_core_matches_fixed_byte_artifact"] = source_core == golden
    results["canonical_core_digest_matches_sha256_artifact"] = digest(source_core) == "sha256:562a20613c64fb9cd3425ff39b5f9fb826b75deb153cdfac48b00fce6f5c19f1"

    vector = {"文": ["📄", "\"\\\b\t\n\f\r\x00\x1f/<>\u2028"], "a": True}
    expected = '{"a":true,"文":["📄","\\\"\\\\\\b\\t\\n\\f\\r\\u0000\\u001f/<>\u2028"]}'.encode("utf-8")
    results["canonical_exact_unicode_escape_and_sort_vector"] = json_bytes(vector) == expected
    results["canonical_large_integer_without_digit_limit"] = integer_text(10 ** 5000 + 1) == "1" + "0" * 4999 + "1"
    results["canonical_negative_zero_unique"] = decimal_parts(Decimal("-0.000")) == ["0", "0"]
    results["canonical_huge_exponent_not_expanded"] = decimal_parts(Decimal("1e1000000000")) == ["1", "1000000000"]
    with localcontext() as context:
        context.prec = 2
        results["canonical_decimal_not_rounded_by_context"] = decimal_parts(Decimal("123456789.0012300")) == ["12345678900123", "-5"]
    results["canonical_rejects_binary_float_tree"] = rejected(lambda: json_bytes({"x": 0.1}), "unsupported value")
    formatted = parse_yaml('# comment\nlabel: "same"\nflag: true\n')
    alternate = parse_yaml('flag: true\nlabel: same\n')
    results["canonical_yaml_comments_quotes_and_mapping_order_irrelevant"] = json_bytes(formatted) == json_bytes(alternate)
    # Use a valid external vocabulary so a real own-content edit can be hashed
    # under the same version, without changing the immutable Core seed.
    from model_lock import check_lock
    library = copy.deepcopy(baseline)
    library_id = "vocabulary-library"
    record = copy.deepcopy(library.models[model])
    record["manifest"] = {"formatVersion": 1, "id": library_id, "version": "2.0.0", "root": "model",
                          "dependencies": {"markitect-core": "1.0.0"}}
    record["load_diagnostics"] = []
    library.models[library_id] = record
    library.models[model]["manifest"]["dependencies"][library_id] = "2.0.0"
    library[vocab]["model"] = library_id
    library[vocab]["locations"] = {library_id: library[vocab]["directory"]}
    for prefix in ("example", "example/connector"):
        library[prefix]["locations"][library_id] = library[prefix]["directory"]
    digests = {identifier: digest(encode_model(library, identifier)) for identifier in ("markitect-core", library_id)}
    lock = {"formatVersion": 1, "models": {identifier: {
        "version": library.models[identifier]["manifest"]["version"], "digest": value} for identifier, value in digests.items()}}
    results["canonical_real_digest_to_lock_comparison_positive"] = check_lock(lock, library, digests)["status"] == "passed"
    own_before = encode_model(library, model)
    lock_before = copy.deepcopy(lock)
    library[vocab]["definitions"]["Container"]["body"] += " An additional binding statement."
    changed_digests = {identifier: digest(encode_model(library, identifier)) for identifier in digests}
    results["canonical_real_content_change_same_version_rejected_by_lock"] = rejected(
        lambda: check_lock(lock, library, changed_digests), "content digest mismatch") and lock == lock_before
    results["canonical_foreign_content_not_recursively_embedded_in_own_hash"] = encode_model(library, model) == own_before and changed_digests[library_id] != digests[library_id]
    return results
