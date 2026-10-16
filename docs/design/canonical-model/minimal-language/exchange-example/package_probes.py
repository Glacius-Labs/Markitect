"""Regression cases for v1 package resolution, importability and file discovery."""
from __future__ import annotations

import copy
import json
import shutil
import tempfile
from pathlib import Path

from canonical_bytes import digest
from model_lock import check_lock


def empty_package_binding_probes(root, load_models, read_yaml, discover, validate, encode_model):
    """Use physical peer models to test bindings against actual bytes and lock."""
    results = {}

    def rejected(action, fragment):
        try:
            action()
        except ValueError as error:
            return fragment in str(error)
        return False

    with tempfile.TemporaryDirectory(prefix="markitect-empty-bindings-") as temporary:
        base = Path(temporary)
        core = base / "core/model.yaml"
        shutil.copytree(root / "dependencies/core", core.parent)

        def model(identifier, dependencies, package=None):
            directory = base / identifier
            content = directory / "model"
            content.mkdir(parents=True)
            manifest = directory / "model.yaml"
            manifest.write_text(json.dumps({"formatVersion": 1, "id": identifier, "version": "1.0",
                                            "root": "model", "dependencies": dependencies}), encoding="utf-8")
            if package:
                target = content / package
                target.mkdir(parents=True)
                (target / "peer.yaml").write_text(
                    'id: PeerType\nkind: Core.Kind\nimports:\n  - package: markitect/core\n'
                    '    as: Core\nbody: A peer type.\nspec: {}\n', encoding="utf-8")
            return manifest

        left = model("library-a", {"markitect-core": "1.0.0"}, "straße/feature")
        right = model("library-b", {"markitect-core": "1.0.0"}, "unrelated")
        entry = model("entry", {"library-a": "1.0", "library-b": "1.0"})

        def load(paths=None):
            diagnostics = []
            packages = load_models(entry, [core, left, right] if paths is None else paths, diagnostics,
                                  read_yaml=read_yaml, discover=discover, source_root=base)
            if diagnostics:
                raise ValueError(f"unexpected fixture diagnostics: {diagnostics!r}")
            validate(packages)
            return packages

        def encoded(packages):
            return {identifier: encode_model(packages, identifier) for identifier in packages.models}

        before = encoded(load())
        external_digests = {identifier: digest(value) for identifier, value in before.items() if identifier != "entry"}
        lock = {"formatVersion": 1, "models": {identifier: {
            "version": "1.0.0" if identifier == "markitect-core" else "1.0", "digest": value}
            for identifier, value in external_digests.items()}}
        check_lock(lock, load(), external_digests)
        sources = {path: path.read_bytes() for path in base.rglob("*.yaml")}
        empty = right.parent / "model/strasse"
        for label, directory in (("leaf", empty), ("subtree", empty / "empty")):
            directory.mkdir()
            packages = load()
            current = encoded(packages)
            results[f"package_empty_{label}_peer_collision_does_not_bind"] = current == before
            actual = {identifier: digest(value) for identifier, value in current.items() if identifier != "entry"}
            results[f"package_empty_{label}_keeps_actual_digests_and_lock"] = (
                actual == external_digests and check_lock(lock, packages, actual)["status"] == "passed")
            results[f"package_empty_{label}_candidate_order_irrelevant"] = encoded(load([right, left, core])) == before
        # Standalone validation and the selected dependency scope must agree.
        standalone = load_models(right, [core], [], read_yaml=read_yaml, discover=discover, source_root=base)
        validate(standalone)
        results["package_empty_peer_scope_agrees_with_standalone_digest"] = encode_model(
            standalone, "library-b") == before["library-b"]
        occupied = empty / "empty/peer.yaml"
        occupied.write_bytes((right.parent / "model/unrelated/peer.yaml").read_bytes())
        results["package_occupied_descendant_binds_colliding_prefix"] = rejected(load, "package ID collision")
        occupied.unlink()
        # Revalidate a changed logical inventory too, rather than only discovery.
        logical = copy.deepcopy(load())
        logical["strasse/empty"]["definitions"] = copy.deepcopy(logical["unrelated"]["definitions"])
        logical["strasse/empty"]["model"] = "library-b"
        results["package_logical_occupied_descendant_collision_revalidated"] = rejected(
            lambda: validate(logical), "package ID collision")
        (empty / "empty").rmdir()
        empty.rmdir()
        results["package_empty_subtree_removal_keeps_bytes"] = encoded(load()) == before
        local_collision = right.parent / "model/straße"
        local_collision.mkdir()
        empty.mkdir()
        results["package_empty_local_unicode_collision_still_rejected"] = rejected(load, "package ID collision")
        empty.rmdir()
        local_collision.rmdir()
        invalid = right.parent / "model/Wrong"
        invalid.mkdir()
        results["package_empty_local_spelling_still_strict"] = rejected(load, "package")
        invalid.rmdir()
        results["package_empty_binding_checks_preserve_source_files"] = all(
            path.read_bytes() == value for path, value in sources.items())
    return results


def package_review_probes(baseline, resolve, validate, package_imports, discover):
    results = {}
    project, vocab = "example/connector/project", "example/connector/vocabulary"

    def rejected(action, fragment):
        try:
            action()
        except ValueError as error:
            return fragment in str(error)
        return False

    trial = copy.deepcopy(baseline)
    definitions = trial[project]["definitions"]
    definitions["Exchange"] = {"id": "Exchange", "kind": "Kind", "body": "A local type.",
                               "imports": [{"package": "markitect/core", "as": "Core"}], "spec": {}}
    definitions["list-documents"]["kind"] = f"{vocab}/Exchange"
    local = {"id": "local-exchange", "kind": f"{project}/Exchange", "spec": {},
             "body": f"Uses [[{project}/Exchange]] and [[Connector.Exchange]].",
             "imports": [{"package": vocab, "as": "Connector"}]}
    definitions["local-exchange"] = local
    validate(trial)
    results["package_ambiguous_short_names_allowed_when_not_used"] = True
    results["package_own_full_reference_disambiguates"] = resolve(trial, project, local, f"{project}/Exchange") == (project, "Exchange")
    results["package_imported_full_reference_disambiguates"] = resolve(trial, project, local, f"{vocab}/Exchange") == (vocab, "Exchange")
    results["package_ambiguous_short_reference_rejected_at_use"] = rejected(
        lambda: resolve(trial, project, local, "Exchange"), "ambiguous")
    results["package_own_full_reference_spelling_remains_strict"] = rejected(
        lambda: resolve(trial, project, local, f"{project}/exchange"), "wrong-case reference")

    for label, identifier, kind, spec in (
        ("kind", "Messaging.Exchange", "Core.Kind", {}),
        ("property", "exchange.sender", "Core.Property", {"type": "string", "minCount": 0, "maxCount": 1}),
        ("property_with_name", "exchange.sender", "Core.Property", {"type": "string", "minCount": 0, "maxCount": 1, "name": "sender"}),
        ("concrete", "local.item", "Connector.Container", {"location": "cloud"}),
    ):
        invalid = copy.deepcopy(baseline)
        definition = {"id": identifier, "kind": kind, "spec": spec, "body": "An unused declaration.",
                      "imports": [{"package": "markitect/core", "as": "Core"}, {"package": vocab, "as": "Connector"}]}
        invalid[project]["definitions"][identifier] = definition
        results[f"package_dotted_{label}_id_rejected_even_when_unused"] = rejected(
            lambda: validate(invalid), "must not contain dots")
    invalid = copy.deepcopy(baseline)
    invalid[vocab]["definitions"]["requester"]["spec"]["name"] = "exchange.sender"
    results["package_dotted_explicit_field_name_rejected"] = rejected(
        lambda: validate(invalid), "must not contain dots")
    results["package_alias_reference_resolves_with_dotless_local_ids"] = resolve(
        trial, project, local, "Connector.Exchange") == (vocab, "Exchange")

    for label, target in (("own_intermediate", "example/connector"), ("external_intermediate", "markitect")):
        trial = copy.deepcopy(baseline)
        definition = trial[project]["definitions"]["list-documents"]
        definition["imports"].append({"package": target, "as": "Empty"})
        results[f"package_{label}_import_rejected_even_when_unused"] = rejected(
            lambda: package_imports(trial, project, definition), "no own definitions")

    with tempfile.TemporaryDirectory(prefix="markitect-package-review-") as temporary:
        base = Path(temporary)

        def source(root, filename, identifier="item"):
            folder = root / "feature"
            folder.mkdir(parents=True, exist_ok=True)
            path = folder / filename
            path.write_text(f"id: {identifier}\nkind: Example\nbody: Text.\nspec: {{}}\n", encoding="utf-8")
            return path

        for index, filename in enumerate(("item.yml", "item.YML", "item.YAML", "item.Yaml", ".yaml")):
            root = base / str(index)
            path = source(root, filename)
            before = path.read_bytes()
            results[f"package_alternative_yaml_filename_rejected_{filename}"] = rejected(
                lambda: discover(root, []), "filename must") and path.read_bytes() == before

        root = base / "ordinary"
        source(root, "first.yaml", "first")
        source(root, "second.yaml", "second")
        # Names shared with metadata have no special meaning inside a Package.
        source(root, "model.yaml", "part")
        source(root, "model-lock.yaml", "binding")
        (root / "feature" / "README.md").write_text("Not YAML: [", encoding="utf-8")
        (root / "feature" / "diagram.png").write_bytes(b"\x00\xff")
        found = discover(root, [])
        results["package_all_yaml_files_discovered_other_files_ignored"] = set(
            found["feature"]["definitions"]) == {"first", "second", "part", "binding"}
        (root / "feature" / "settings.yaml").write_text("setting: value\n", encoding="utf-8")
        results["package_yaml_data_file_not_silently_ignored"] = rejected(
            lambda: discover(root, []), "missing nonempty id")

        root = base / "dotted-id"
        path = source(root, "item.yaml", "local.item")
        before = path.read_bytes()
        results["package_dotted_source_id_rejected_without_correction"] = rejected(
            lambda: discover(root, []), "must not contain dots") and path.read_bytes() == before

        root = base / "metadata"
        source(root, "item.yaml")
        manifest, lock = root / "model.yaml", root / "model-lock.yaml"
        manifest.write_text("formatVersion: 1\n", encoding="utf-8")
        lock.write_text("formatVersion: 1\n", encoding="utf-8")
        results["package_only_explicit_metadata_excluded"] = set(discover(
            root, [], metadata_paths=(manifest, lock))) == {"feature"}
        results["package_metadata_names_not_globally_reserved"] = rejected(
            lambda: discover(root, []), "forbidden directly in the model root")

        root = base / "wrong-metadata"
        source(root, "item.yaml")
        (root / "Model.yaml").write_text("formatVersion: 1\n", encoding="utf-8")
        results["package_metadata_exclusion_requires_exact_filename"] = rejected(
            lambda: discover(root, [], metadata_paths=(root / "model.yaml",)), "forbidden directly in the model root")
    return results
