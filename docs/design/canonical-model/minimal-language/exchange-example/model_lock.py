"""Lock schema and binding probes for computed canonical model digests.

Synthetic self-test digests exercise comparison only. Source checking receives
real model-digest/1 digests. Reading or checking never rewrites a lock.
"""
from __future__ import annotations

import copy
import json
import re
import tempfile
from pathlib import Path

from model_manifest import check_bound_models, exact_version
from name_profile import ProbeLimitError, name_key, require_nfc, require_model_label
from root_paths import reject_redirect


DIGEST = re.compile(r"sha256:[0-9a-f]{64}\Z", re.ASCII)
PENDING_DIGEST = "sha256:<pending-canonical-bytes>"


def closed_fields(value, required, label):
    if not isinstance(value, dict):
        raise ValueError(f"{label} must be a mapping")
    for field in sorted(set(value) - required):
        near = next((known for known in required if name_key(field) == name_key(known)), None)
        if near:
            raise ValueError(f"{label}: wrong-case field {field!r}; write {near!r}")
        raise ValueError(f"{label}: unknown field {field!r}")
    if set(value) != required:
        raise ValueError(f"{label}: missing required fields {sorted(required - set(value))!r}")


def check_lock(lock, packages, computed_digests=None, *, allow_pending_template=False):
    """Bind exactly the external closure and compare separately supplied digests.

    computed_digests contains the selected canonicalizer's output. Synthetic
    self-tests can supply artificial values to isolate comparison behavior.
    A pending sentinel is retained only for negative/partial template probes;
    normal source loading never accepts it.
    """
    check_bound_models(packages)
    closed_fields(lock, {"formatVersion", "models"}, "model-lock.yaml")
    if type(lock["formatVersion"]) is not int or lock["formatVersion"] != 1:
        raise ValueError("unsupported lock formatVersion; this probe implements 1")
    if not isinstance(lock["models"], dict):
        raise ValueError("lock models must be a mapping")
    expected = set(packages.models) - {packages.entry_model}
    seen = {}
    pending = []
    for identifier, binding in lock["models"].items():
        require_nfc(identifier, "locked model ID")
        if name_key(identifier) in seen:
            raise ValueError(f"case-insensitive locked model ID collision {seen[name_key(identifier)]!r} vs {identifier!r}")
        seen[name_key(identifier)] = identifier
        near = next((known for known in expected if name_key(known) == name_key(identifier)), None)
        if near and near != identifier:
            raise ValueError(f"wrong-case locked model ID {identifier!r}; write {near!r}")
        require_model_label(identifier, "locked model ID")
        closed_fields(binding, {"version", "digest"}, f"locked model {identifier!r}")
        if not isinstance(binding["version"], str) or not binding["version"]:
            raise ValueError("locked version must be nonempty exact text")
        exact_version(binding["version"])
        digest = binding["digest"]
        if allow_pending_template and digest == PENDING_DIGEST:
            pending.append(identifier)
        elif not isinstance(digest, str) or not DIGEST.fullmatch(digest):
            raise ValueError("locked digest must be sha256: followed by exactly 64 lowercase hexadecimal digits")
    for identifier in sorted(set(lock["models"]) - expected):
        near = next((known for known in expected if name_key(known) == name_key(identifier)), None)
        if near:
            raise ValueError(f"wrong-case locked model ID {identifier!r}; write {near!r}")
    missing = sorted(expected - set(lock["models"]))
    extra = sorted(set(lock["models"]) - expected)
    if missing or extra:
        raise ValueError(f"lock must equal the external dependency closure; missing={missing!r}, extra={extra!r}")
    for identifier in sorted(expected):
        version = packages.models[identifier]["manifest"]["version"]
        if lock["models"][identifier]["version"] != version:
            if name_key(lock["models"][identifier]["version"]) == name_key(version):
                raise ValueError(f"colliding version label spelling for locked model {identifier!r}; write declared label {version!r} exactly")
            raise ValueError(f"locked model {identifier!r} version does not match the manifest requirements and supplied model")
    diagnostics = []
    checked = False
    if computed_digests is None and expected:
        diagnostics.append("No computed canonical model digests were supplied; content verification is incomplete.")
    else:
        computed_digests = computed_digests or {}
        if set(computed_digests) != expected:
            raise ValueError("computed digests must cover exactly the locked external models")
        for identifier in sorted(expected):
            digest = computed_digests[identifier]
            if not isinstance(digest, str) or not DIGEST.fullmatch(digest):
                raise ValueError("computed digest has an invalid representation")
            if identifier not in pending and lock["models"][identifier]["digest"] != digest:
                raise ValueError(f"locked model {identifier!r} content digest mismatch; no automatic lock update")
        checked = not pending
    if pending:
        diagnostics.append("The supplied template has an explicit pending digest and is not a valid completed lock.")
    return {"status": "passed" if checked and not pending else "not_completed",
            "schema_validation": "not_completed" if pending else "passed",
            "manifest_bindings": "passed", "content_digest_checked": checked,
            "external_models": sorted(expected), "pending_templates": sorted(pending),
            "diagnostics": diagnostics, "sources_modified": False}


def load_lock(path, packages, *, read_yaml, source_root, computed_digests=None):
    if path.name != "model-lock.yaml":
        raise ValueError("lock filename must be exactly model-lock.yaml")
    filenames = sorted(item.name for item in path.parent.iterdir() if name_key(item.name) == name_key("model-lock.yaml"))
    if filenames != ["model-lock.yaml"]:
        raise ValueError("lock filename must be exactly model-lock.yaml without case-colliding alternatives; no automatic correction")
    reject_redirect(path)
    lock = read_yaml(path, source_root=source_root)
    return check_lock(lock, packages, computed_digests)


def lock_probes(source, read_yaml, source_root, encode_model):
    """Exercise synthetic binding cases and verify the actual stored source lock."""
    results = {}
    packages = copy.deepcopy(source)
    expected = set(packages.models) - {packages.entry_model}
    digests = {identifier: f"sha256:{index:064x}" for index, identifier in enumerate(sorted(expected), 1)}
    lock = {"formatVersion": 1, "models": {
        identifier: {"version": packages.models[identifier]["manifest"]["version"], "digest": digests[identifier]}
        for identifier in sorted(expected)}}
    before = copy.deepcopy((packages, lock, digests))
    results["lock_synthetic_digest_binding_positive"] = check_lock(lock, packages, digests)["status"] == "passed"
    results["lock_check_does_not_rewrite_inputs"] = (packages, lock, digests) == before
    results["lock_check_preserves_manifest_metadata"] = packages.models == before[0].models
    no_digest = check_lock(lock, packages)
    results["lock_valid_syntax_without_canonicalizer_is_incomplete"] = no_digest["status"] == "not_completed" and not no_digest["content_digest_checked"]

    def rejected(action, fragment):
        try:
            action()
        except ValueError as error:
            return fragment in str(error)
        return False

    def mutate(action, fragment):
        trial = copy.deepcopy(lock)
        action(trial)
        return rejected(lambda: check_lock(trial, packages, digests), fragment)

    identifier = sorted(expected)[0]
    cases = (
        ("missing_model", lambda d: d["models"].pop(identifier), "external dependency closure"),
        ("extra_model", lambda d: d["models"].update(extra={"version": "1.0.0", "digest": digests[identifier]}), "external dependency closure"),
        ("local_model_excluded", lambda d: d["models"].update({packages.entry_model: {"version": "1.0.0", "digest": digests[identifier]}}), "external dependency closure"),
        ("wrong_version", lambda d: d["models"][identifier].update(version="9.0.0"), "version does not match"),
        ("version_label_components_not_padded", lambda d: d["models"][identifier].update(version="1.0"), "version does not match"),
        ("version_label_newer_is_not_compatible", lambda d: d["models"][identifier].update(version="1.0.1"), "version does not match"),
        ("wrong_digest", lambda d: d["models"][identifier].update(digest="sha256:" + "f" * 64), "content digest mismatch"),
        ("numeric_version", lambda d: d["models"][identifier].update(version=1), "version must be nonempty exact text"),
        ("missing_digest", lambda d: d["models"][identifier].pop("digest"), "missing required fields"),
        ("missing_version", lambda d: d["models"][identifier].pop("version"), "missing required fields"),
        ("redundant_edges", lambda d: d["models"][identifier].update(dependencies={}), "unknown field"),
        ("source_path_not_in_lock", lambda d: d["models"][identifier].update(path="installed/core"), "unknown field"),
        ("redundant_root_id", lambda d: d.update(model=packages.entry_model), "unknown field"),
        ("unknown_top_field", lambda d: d.update(other=True), "unknown field"),
        ("wrong_case_field", lambda d: d.update(Models=d.pop("models")), "wrong-case field"),
        ("wrong_case_model_id", lambda d: d["models"].update({identifier.upper(): d["models"].pop(identifier)}), "wrong-case locked model ID"),
        ("case_collision", lambda d: d["models"].update({identifier.upper(): copy.deepcopy(d["models"][identifier])}), "locked model ID collision"),
        ("unsupported_format", lambda d: d.update(formatVersion=2), "unsupported lock formatVersion"),
        ("boolean_format", lambda d: d.update(formatVersion=True), "unsupported lock formatVersion"),
        ("models_not_mapping", lambda d: d.update(models=[]), "models must be a mapping"),
        ("binding_not_mapping", lambda d: d["models"].update({identifier: "1.0.0"}), "must be a mapping"),
    )
    for label, action, fragment in cases:
        results[f"lock_{label}_rejected"] = mutate(action, fragment)
    for version in ("Release", "1..0", "1.0-", "version/name", "a_b", "u\u0308berweisung"):
        results[f"lock_common_version_label_grammar_rejects_{version}"] = mutate(
            lambda d, v=version: d["models"][identifier].update(version=v), "must already be NFC" if version == "u\u0308berweisung" else "lowercase Unicode label sections")
    for version in ("latest", "*", "^1.0", ">=1.0"):
        results[f"lock_version_range_{version}_rejected"] = mutate(lambda d, v=version: d["models"][identifier].update(version=v), "ranges and latest")
    for version in ("x", "1.x", "release.x"):
        exact_packages = copy.deepcopy(packages)
        exact_packages.models[identifier]["manifest"]["version"] = version
        for record in exact_packages.models.values():
            if identifier in record["manifest"]["dependencies"]:
                record["manifest"]["dependencies"][identifier] = version
        exact_lock = copy.deepcopy(lock)
        exact_lock["models"][identifier]["version"] = version
        results[f"lock_x_label_{version}_exact_binding_passes"] = check_lock(exact_lock, exact_packages, digests)["status"] == "passed"
        exact_lock["models"][identifier]["version"] = "1.7"
        results[f"lock_x_label_{version}_is_not_a_wildcard"] = rejected(
            lambda: check_lock(exact_lock, exact_packages, digests), "version does not match")
    unicode_packages = copy.deepcopy(packages)
    unicode_packages.models[identifier]["manifest"]["version"] = "straße"
    for record in unicode_packages.models.values():
        if identifier in record["manifest"]["dependencies"]:
            record["manifest"]["dependencies"][identifier] = "straße"
    unicode_lock = copy.deepcopy(lock)
    unicode_lock["models"][identifier]["version"] = "strasse"
    results["lock_colliding_version_label_is_not_corrected"] = rejected(
        lambda: check_lock(unicode_lock, unicode_packages, digests), "colliding version label spelling")
    unicode_lock["models"][identifier]["version"] = "straße"
    results["lock_unicode_version_label_exact_binding_passes"] = check_lock(unicode_lock, unicode_packages, digests)["status"] == "passed"
    for digest in ("sha256:<Digest>", "sha256:" + "0" * 63, "sha256:" + "A" * 64, "SHA256:" + "0" * 64, "sha512:" + "0" * 64, "sha256:" + "0" * 64 + " "):
        results[f"lock_invalid_digest_{digest}_rejected"] = mutate(lambda d, value=digest: d["models"][identifier].update(digest=value), "64 lowercase hexadecimal digits")
    template = copy.deepcopy(lock)
    template["models"][identifier]["digest"] = PENDING_DIGEST
    results["lock_pending_placeholder_never_conformant"] = rejected(lambda: check_lock(template, packages, digests), "64 lowercase hexadecimal digits")
    partial = check_lock(template, packages, allow_pending_template=True)
    results["lock_pending_template_reports_incomplete_not_verified"] = partial["status"] == partial["schema_validation"] == "not_completed" and not partial["content_digest_checked"]
    changed = {**digests, identifier: "sha256:" + "e" * 64}
    results["lock_changed_content_under_same_version_rejected"] = rejected(lambda: check_lock(lock, packages, changed), "content digest mismatch")
    local_edit = copy.deepcopy(packages)
    local_edit["example/connector/project"]["definitions"]["list-documents"]["body"] += " Local change."
    results["lock_local_definition_change_does_not_require_relock"] = check_lock(lock, local_edit, digests)["status"] == "passed"
    results["lock_missing_computed_digest_rejected"] = rejected(lambda: check_lock(lock, packages, {}), "computed digests must cover exactly")
    results["lock_unrelated_computed_digest_rejected"] = rejected(lambda: check_lock(lock, packages, {**digests, "extra": digests[identifier]}), "computed digests must cover exactly")
    binding = copy.deepcopy(lock["models"][identifier])
    results["lock_unicode_model_collision_rejected"] = mutate(lambda d: d["models"].update({"straße": binding, "strasse": binding}), "locked model ID collision")
    results["lock_non_nfc_model_id_rejected"] = mutate(lambda d: d["models"].update({"u\u0308berweisung": binding}), "must already be NFC")
    # A synthetic manifest adds a transitive dependency, without importing any
    # canonical-byte encoding or claiming a digest for the new source.
    transitive = copy.deepcopy(packages)
    leaf = "lock-probe-leaf"
    transitive.models[identifier]["manifest"]["dependencies"][leaf] = "3.0.0"
    record = copy.deepcopy(transitive.models[identifier])
    record["manifest"] = {"formatVersion": 1, "id": leaf, "version": "3.0.0", "root": "model", "dependencies": {}}
    transitive.models[leaf] = record
    transitive_lock = copy.deepcopy(lock)
    transitive_lock["models"][leaf] = {"version": "3.0.0", "digest": "sha256:" + "c" * 64}
    transitive_digests = {**digests, leaf: "sha256:" + "c" * 64}
    results["lock_transitive_closure_complete_positive"] = check_lock(transitive_lock, transitive, transitive_digests)["status"] == "passed"
    results["lock_missing_transitive_entry_rejected"] = rejected(lambda: check_lock(lock, transitive, digests), "external dependency closure")
    transitive.models[identifier]["manifest"]["dependencies"].pop(leaf)
    transitive.models.pop(leaf)
    results["lock_stale_transitive_entry_rejected"] = rejected(lambda: check_lock(transitive_lock, transitive, transitive_digests), "external dependency closure")
    empty = copy.deepcopy(packages)
    empty.models = {empty.entry_model: copy.deepcopy(empty.models[empty.entry_model])}
    empty.models[empty.entry_model]["manifest"]["dependencies"] = {}
    results["lock_empty_external_graph_has_empty_table"] = check_lock({"formatVersion": 1, "models": {}}, empty)["status"] == "passed"
    with tempfile.TemporaryDirectory(prefix="markitect-lock-") as temporary:
        temporary_root = Path(temporary)
        path = temporary_root / "model-lock.yaml"
        path.write_text(json.dumps(lock), encoding="utf-8")
        results["lock_valid_file_still_requires_canonicalizer"] = load_lock(path, packages, read_yaml=read_yaml, source_root=temporary_root)["status"] == "not_completed"
        intermediate = path.rename(path.with_name("lock-intermediate.yaml"))
        wrong_case = intermediate.rename(path.with_name("Model-lock.yaml"))
        results["lock_actual_filename_case_checked_on_disk"] = rejected(lambda: load_lock(path, packages, read_yaml=read_yaml, source_root=temporary_root), "filename must be exactly model-lock.yaml")
        wrong_case.rename(path.with_name("lock-intermediate.yaml")).rename(path)
    stored = source_root / "model-lock.yaml"
    before_bytes = stored.read_bytes()
    from canonical_bytes import digest
    actual = {identifier: digest(encode_model(packages, identifier)) for identifier in expected}
    result = load_lock(stored, packages, read_yaml=read_yaml, source_root=source_root, computed_digests=actual)
    results["lock_stored_source_digests_verified"] = result["status"] == "passed" and result["content_digest_checked"]
    results["lock_stored_source_not_rewritten"] = stored.read_bytes() == before_bytes
    return results
