"""Bounded local model-manifest loading, separate from definition validation.

Only explicitly supplied local candidates are used. Lock structure/bindings are
checked in model_lock.py; canonical_bytes.py implements the selected own-content
projection. Registry and release operations remain outside this probe.
"""
from __future__ import annotations

import copy
import heapq
import json
import re
import tempfile
from pathlib import Path
from typing import Any

from name_profile import ProbeLimitError, name_key, require_model_label
from root_paths import model_root_path, reject_overlapping_roots, reject_redirect, root_components


class ModelPackages(dict):
    """Package inventory plus the explicit model context needed for imports."""

    def __init__(self, models, entry_model, packages=None):
        super().__init__(packages or {})
        self.models = models
        self.entry_model = entry_model


def package_binding_names(packages) -> set[str]:
    """Cross-model names come only from occupied paths and their prefixes."""
    names = set()
    for name, package in packages.items():
        if package["definitions"]:
            parts = name.split("/")
            names.update("/".join(parts[:end]) for end in range(1, len(parts) + 1))
    return names


def exact_version(value: Any) -> None:
    # Versions are exact text labels: no padding, ordering or compatibility.
    if not isinstance(value, str) or not value or any(char.isspace() for char in value):
        raise ValueError("version must be nonempty text naming an exact version")
    if value.casefold() == "latest" or re.search(r"[\*\^~<>=|]", value):
        raise ValueError("version ranges and latest are forbidden")
    require_model_label(value, "version label")


def check_manifest(manifest: dict[str, Any], *, external: bool = False) -> None:
    allowed = {"formatVersion", "id", "version", "root", "dependencies"}
    required = allowed - {"version"}
    for field in sorted(set(manifest) - allowed):
        near = next((known for known in allowed if name_key(field) == name_key(known)), None)
        if near:
            raise ValueError(f"wrong-case model manifest field {field!r}; write {near!r}")
        raise ValueError(f"unknown model manifest field {field!r}")
    if not required <= set(manifest):
        raise ValueError(f"missing model manifest fields: {sorted(required - set(manifest))!r}")
    if type(manifest["formatVersion"]) is not int or manifest["formatVersion"] != 1:
        raise ValueError("unsupported model formatVersion; this probe implements 1")
    require_model_label(manifest["id"], "model ID")
    root_components(manifest["root"])
    if "version" in manifest:
        exact_version(manifest["version"])
    elif external:
        raise ValueError("a dependency model requires an explicit version")
    dependencies = manifest["dependencies"]
    if not isinstance(dependencies, dict):
        raise ValueError("model dependencies must be a mapping from model ID to exact version")
    seen = {}
    for identifier, version in dependencies.items():
        require_model_label(identifier, "dependency model ID")
        if name_key(identifier) in seen:
            raise ValueError(f"case-insensitive dependency model collision {seen[name_key(identifier)]!r} vs {identifier!r}")
        seen[name_key(identifier)] = identifier
        exact_version(version)


def check_bindings(models, entry_model) -> set[str]:
    """Audit exact identity/version bindings and acyclic model dependencies."""
    seen = {}
    for identifier, record in sorted(models.items()):
        if name_key(identifier) in seen:
            raise ValueError(f"case-insensitive model ID collision {seen[name_key(identifier)]!r} vs {identifier!r}")
        seen[name_key(identifier)] = identifier
        check_manifest(record["manifest"], external=identifier != entry_model)
        if record["manifest"]["id"] != identifier:
            raise ValueError("model inventory ID differs from its manifest")
    if entry_model not in models:
        raise ValueError("entry model is absent")
    visited = set()
    pending = [entry_model]
    while pending:
        identifier = pending.pop()
        if identifier in visited:
            continue
        visited.add(identifier)
        for dependency, version in sorted(models[identifier]["manifest"]["dependencies"].items()):
            if dependency not in models:
                near = next((known for known in models if name_key(known) == name_key(dependency)), None)
                if near:
                    raise ValueError(f"wrong-case dependency model ID {dependency!r}; write {near!r}")
                raise ValueError(f"dependency model {dependency!r} is not explicitly supplied")
            actual = models[dependency]["manifest"].get("version")
            if actual != version:
                if isinstance(actual, str) and name_key(actual) == name_key(version):
                    raise ValueError(f"{identifier}: colliding version label spelling {version!r} for model {dependency!r}; write declared label {actual!r} exactly")
                raise ValueError(f"{identifier}: dependency model {dependency!r} requires exact version {version!r}; supplied {actual!r}")
            pending.append(dependency)
    # The same acyclicity rule rejects self-dependencies and longer cycles.
    incoming = {identifier: 0 for identifier in visited}
    for identifier in visited:
        for dependency in models[identifier]["manifest"]["dependencies"]:
            incoming[dependency] += 1
    ready = [identifier for identifier, count in incoming.items() if count == 0]
    heapq.heapify(ready)
    completed = 0
    while ready:
        identifier = heapq.heappop(ready)
        completed += 1
        for dependency in sorted(models[identifier]["manifest"]["dependencies"]):
            incoming[dependency] -= 1
            if incoming[dependency] == 0:
                heapq.heappush(ready, dependency)
    if completed != len(visited):
        blocked = sorted(identifier for identifier, count in incoming.items() if count > 0)
        raise ValueError(f"model dependencies must be acyclic, including no self-dependencies; blocked by cycle: {blocked!r}")
    return visited


def load_models(entry_manifest, supplied_manifests, diagnostics, *, read_yaml, discover, source_root):
    catalog = {}
    for path in [entry_manifest, *sorted(supplied_manifests)]:
        if path.name != "model.yaml":
            raise ValueError("model manifest filename must be exactly model.yaml")
        reject_redirect(path)
        filenames = sorted(item.name for item in path.parent.iterdir() if name_key(item.name) == name_key("model.yaml"))
        if filenames != ["model.yaml"]:
            raise ValueError("model manifest filename must be exactly model.yaml without case-colliding alternatives; no automatic correction")
        manifest = read_yaml(path, source_root=source_root)
        check_manifest(manifest, external=path != entry_manifest)
        identifier = manifest["id"]
        previous = next((known for known in catalog if name_key(known) == name_key(identifier)), None)
        if previous is not None:
            old_version = catalog[previous]["manifest"].get("version")
            new_version = manifest.get("version")
            if previous == identifier and isinstance(old_version, str) and isinstance(new_version, str) and old_version != new_version and name_key(old_version) == name_key(new_version):
                raise ValueError(f"case-insensitive version label collision {old_version!r} vs {new_version!r} within model {identifier!r}")
            raise ValueError(f"case-insensitive model ID collision {previous!r} vs {identifier!r}")
        catalog[identifier] = {"manifest": manifest, "path": path, "root": model_root_path(path, manifest["root"])}
    entry = next(identifier for identifier, record in catalog.items() if record["path"] == entry_manifest)
    reachable = check_bindings(catalog, entry)
    models = {identifier: catalog[identifier] for identifier in sorted(reachable)}
    reject_overlapping_roots(models)
    packages = ModelPackages(models, entry)
    folded = {}
    for identifier, record in models.items():
        model_diagnostics = []
        discovered = discover(record["root"], model_diagnostics, source_root=source_root,
                              metadata_paths=(record["path"], record["path"].with_name("model-lock.yaml")))
        record["load_diagnostics"] = model_diagnostics
        diagnostics.extend(model_diagnostics)
        binding_names = package_binding_names(discovered)
        for name, package in discovered.items():
            if name in binding_names:
                previous_name = folded.get(name_key(name))
                if previous_name is not None and previous_name != name:
                    raise ValueError(f"case-insensitive package ID collision {previous_name!r} vs {name!r}")
                folded[name_key(name)] = name
            package["model"] = identifier if package["definitions"] else None
            package["locations"] = {identifier: package["directory"]}
            if name in packages:
                previous = packages[name]
                if previous["definitions"] and package["definitions"]:
                    raise ValueError(f"occupied package {name!r} is split across models {previous['model']!r} and {identifier!r}")
                locations = {**previous["locations"], **package["locations"]}
                package = package if package["definitions"] else previous
                package["locations"] = locations
            packages[name] = package
    return packages


def check_bound_models(packages: ModelPackages) -> None:
    if not isinstance(packages, ModelPackages):
        raise ProbeLimitError("probe scope: explicit model context is missing")
    if check_bindings(packages.models, packages.entry_model) != set(packages.models):
        raise ValueError("bound model inventory contains unreachable models")


def import_model_access(packages: ModelPackages, source: str, target: str) -> None:
    """A file can import own-model Packages or direct model dependencies."""
    package = packages[source]
    owner = package["model"]
    if owner is None:
        if len(package["locations"]) != 1:
            raise ProbeLimitError("probe scope: a definition in an ownerless shared package has no source model")
        owner = next(iter(package["locations"]))
    visible_models = {owner, *packages.models[owner]["manifest"]["dependencies"]}
    imported = packages[target]
    if not imported["definitions"]:
        raise ValueError(f"imported package {target!r} has no own definitions; empty and intermediate Packages are not importable")
    providers = {imported["model"]} if imported["model"] is not None else set(imported["locations"])
    if not providers & visible_models:
        raise ValueError(f"imported package {target!r} is not provided by the own model or a direct model dependency")


def manifest_probes(read_yaml, discover):
    results = {}

    def rejected(action, fragment):
        try:
            action()
        except ValueError as error:
            return fragment in str(error)
        return False

    def label_manifest(identifier="example-model", version="1.0"):
        return {"formatVersion": 1, "id": identifier, "version": version, "root": "model", "dependencies": {}}

    for label in ("markitect-core", "überweisungen", "文档", "1.0", "1.0.0-rc.1", "2026-modell", "markitect.core", "٢٠٢٦.١", "q\u0307", "ⅰ", "x", "1.x", "release.x", "x-1"):
        check_manifest(label_manifest(label, label))
        results[f"manifest_common_label_grammar_accepts_{label}"] = True
    for label in ("Markitect-Core", "1.0-RC.1", "über_weisungen", "one\u203ftwo", "modell/name", "modell\\name", "a b", "📄", "1..0", "1.-0", "1--0", ".a", "a.", "-a", "a-", "\u0301a", "a.\u0301b", "1+build", "Ⅰ"):
        for field in ("id", "version"):
            data = label_manifest()
            data[field] = label
            fragment = "version must be nonempty text" if field == "version" and " " in label else "lowercase Unicode label sections"
            results[f"manifest_common_label_grammar_{field}_rejects_{label}"] = rejected(lambda d=data: check_manifest(d), fragment)
    for field in ("id", "version"):
        data = label_manifest()
        data[field] = "u\u0308berweisung"
        results[f"manifest_common_label_grammar_{field}_requires_nfc"] = rejected(lambda d=data: check_manifest(d), "must already be NFC")
    for field, data in (("id", {"1..0": "1.0"}), ("version", {"example-model": "1..0"})):
        manifest = label_manifest()
        manifest["dependencies"] = data
        results[f"manifest_dependency_{field}_uses_common_label_grammar"] = rejected(lambda d=manifest: check_manifest(d), "lowercase Unicode label sections")
    check_manifest(label_manifest("latest"))
    results["manifest_latest_is_reserved_only_as_version"] = True
    for label in ("x", "1.x", "release.x"):
        catalog = {
            "entry": {"manifest": {**label_manifest("entry"), "dependencies": {"library": label}}},
            "library": {"manifest": label_manifest("library", label)},
        }
        results[f"manifest_x_label_{label}_binds_exactly"] = check_bindings(catalog, "entry") == {"entry", "library"}
        catalog["library"]["manifest"]["version"] = "1.7"
        results[f"manifest_x_label_{label}_does_not_select_numeric_version"] = rejected(
            lambda: check_bindings(catalog, "entry"), "requires exact version")
    cycle_cases = (
        ("self", {"one": {"one": "1.0"}}),
        ("mutual", {"one": {"two": "1.0"}, "two": {"one": "1.0"}}),
        ("longer", {"one": {"two": "1.0"}, "two": {"three": "1.0"}, "three": {"one": "1.0"}}),
        ("transitive_self", {"one": {"two": "1.0"}, "two": {"two": "1.0"}}),
    )
    for label, requirements in cycle_cases:
        catalog = {identifier: {"manifest": {**label_manifest(identifier), "dependencies": dependencies}}
                   for identifier, dependencies in requirements.items()}
        try:
            check_bindings(catalog, "one")
        except ProbeLimitError:
            results[f"manifest_{label}_dependency_cycle_is_language_error"] = False
        except ValueError as error:
            results[f"manifest_{label}_dependency_cycle_is_language_error"] = "model dependencies must be acyclic" in str(error)
        else:
            results[f"manifest_{label}_dependency_cycle_is_language_error"] = False
    diamond = {"one": {"two": "1.0", "three": "1.0"}, "two": {"four": "1.0"}, "three": {"four": "1.0"}, "four": {}}
    catalog = {identifier: {"manifest": {**label_manifest(identifier), "dependencies": dependencies}}
               for identifier, dependencies in diamond.items()}
    results["manifest_shared_transitive_dependency_is_acyclic"] = check_bindings(catalog, "one") == set(diamond)
    chain = copy.deepcopy(catalog)
    del chain["three"]
    del chain["one"]["manifest"]["dependencies"]["three"]
    results["manifest_dependency_chain_is_acyclic"] = check_bindings(chain, "one") == {"one", "two", "four"}

    with tempfile.TemporaryDirectory(prefix="markitect-models-") as temporary:
        base = Path(temporary)

        def write_model(folder, identifier, version, dependencies, package):
            directory = base / folder
            directory.mkdir()
            manifest = {"formatVersion": 1, "id": identifier, "root": "content", "dependencies": dependencies}
            if version is not None:
                manifest["version"] = version
            path = directory / "model.yaml"
            path.write_text(json.dumps(manifest, ensure_ascii=False), encoding="utf-8")
            target = directory / "content" / package
            target.mkdir(parents=True)
            (target / "item.yaml").write_text("id: item\nkind: Example\nbody: probe\nspec: {}\n", encoding="utf-8")
            return path

        entry = write_model("unrelated-checkout", "connector-example", None, {"library-middle": "2.0.0"}, "example/project")
        middle = write_model("middle-installation", "library-middle", "2.0.0", {"markitect-core": "1.0.0"}, "markitect/middle")
        core = write_model("core-installation", "markitect-core", "1.0.0", {}, "markitect/core")
        cycle_one = write_model("cycle-one", "cycle-one", "1.0", {"cycle-two": "1.0"}, "cycle/one")
        cycle_two = write_model("cycle-two", "cycle-two", "1.0", {"cycle-one": "1.0"}, "cycle/two")
        try:
            load_models(cycle_one, [cycle_two], [], read_yaml=read_yaml, discover=discover, source_root=base)
        except ProbeLimitError:
            results["manifest_on_disk_dependency_cycle_is_language_error"] = False
        except ValueError as error:
            results["manifest_on_disk_dependency_cycle_is_language_error"] = "model dependencies must be acyclic" in str(error)
        else:
            results["manifest_on_disk_dependency_cycle_is_language_error"] = False

        def load(paths=None):
            diagnostics = []
            result = load_models(entry, [middle, core] if paths is None else paths, diagnostics,
                                 read_yaml=read_yaml, discover=discover, source_root=base)
            if diagnostics:
                raise ValueError("unexpected fixture diagnostics")
            return result

        bound = load()
        before = {path: path.read_bytes() for path in (entry, middle, core)}
        results["manifest_exact_direct_and_transitive_bindings"] = set(bound.models) == {"connector-example", "library-middle", "markitect-core"}
        results["manifest_local_entry_version_optional"] = "version" not in bound.models["connector-example"]["manifest"]
        results["manifest_identity_independent_of_checkout_and_root_name"] = "example/project" in bound and "connector-example/example/project" not in bound
        results["manifest_common_empty_namespace_prefix_allowed"] = bound["markitect"]["model"] is None and set(bound["markitect"]["locations"]) == {"library-middle", "markitect-core"}
        results["manifest_occupied_package_has_single_owner"] = bound["markitect/core"]["model"] == "markitect-core"
        results["manifest_source_bytes_preserved"] = all(path.read_bytes() == value for path, value in before.items())
        import_model_access(bound, "example/project", "markitect/middle")
        results["manifest_direct_model_dependency_can_supply_import"] = True
        results["manifest_transitive_model_dependency_not_directly_importable"] = rejected(
            lambda: import_model_access(bound, "example/project", "markitect/core"), "direct model dependency")

        relocated = entry.parent.rename(base / "moved-checkout")
        entry = relocated / "model.yaml"
        results["manifest_physical_move_keeps_model_and_package_identity"] = set(load()) == set(bound) and load().entry_model == bound.entry_model
        intermediate = entry.rename(entry.with_name("manifest-intermediate.yaml"))
        wrong_case = intermediate.rename(entry.with_name("Model.yaml"))
        try:
            results["manifest_actual_filename_case_checked_on_disk"] = rejected(load, "filename must be exactly model.yaml")
        finally:
            intermediate = wrong_case.rename(entry.with_name("manifest-intermediate.yaml"))
            intermediate.rename(entry)

        def mutate(path, action, fragment):
            original = path.read_bytes()
            data = read_yaml(path, source_root=base)
            action(data)
            path.write_text(json.dumps(data, ensure_ascii=False), encoding="utf-8")
            try:
                return rejected(load, fragment)
            finally:
                path.write_bytes(original)

        cases = (
            ("unknown_field", entry, lambda d: d.update(packages=[]), "unknown model manifest field"),
            ("wrong_case_field", entry, lambda d: d.update(Id=d.pop("id")), "wrong-case model manifest field"),
            ("missing_root", entry, lambda d: d.pop("root"), "missing model manifest fields"),
            ("missing_dependencies", entry, lambda d: d.pop("dependencies"), "missing model manifest fields"),
            ("missing_identity", entry, lambda d: d.pop("id"), "missing model manifest fields"),
            ("unsupported_format", entry, lambda d: d.update(formatVersion=2), "unsupported model formatVersion"),
            ("boolean_format", entry, lambda d: d.update(formatVersion=True), "unsupported model formatVersion"),
            ("dependency_version_missing", core, lambda d: d.pop("version"), "requires an explicit version"),
            ("version_is_text", core, lambda d: d.update(version=1), "version must be nonempty text"),
            ("null_version", core, lambda d: d.update(version=None), "null and empty YAML values"),
            ("empty_identity", entry, lambda d: d.update(id=""), "model ID must be nonempty text"),
            ("non_nfc_identity", entry, lambda d: d.update(id="u\u0308berweisung"), "must already be NFC"),
            ("wrong_identity_case", core, lambda d: d.update(id="Library-middle"), "lowercase Unicode label sections"),
            ("wrong_dependency_case", entry, lambda d: d.update(dependencies={"Library-middle": "2.0.0"}), "lowercase Unicode label sections"),
            ("dependency_not_supplied", entry, lambda d: d.update(dependencies={"missing": "1.0.0"}), "not explicitly supplied"),
            ("exact_version_mismatch", entry, lambda d: d.update(dependencies={"library-middle": "3.0.0"}), "requires exact version"),
            ("version_label_components_not_padded", core, lambda d: d.update(version="1.0"), "requires exact version"),
            ("version_label_newer_is_not_compatible", core, lambda d: d.update(version="1.0.1"), "requires exact version"),
            ("direct_transitive_version_conflict", entry, lambda d: d["dependencies"].update({"markitect-core": "2.0.0"}), "requires exact version"),
            ("absolute_root", entry, lambda d: d.update(root="/absolute"), "root must be relative"),
            ("windows_absolute_root", entry, lambda d: d.update(root="C:/absolute"), "root must be relative"),
            ("parent_root", entry, lambda d: d.update(root="../other"), "canonical relative / components"),
            ("empty_root", entry, lambda d: d.update(root=""), "root must be a nonempty"),
            ("dependency_shape", entry, lambda d: d.update(dependencies=[]), "dependencies must be a mapping"),
            ("dependency_object_shape", entry, lambda d: d.update(dependencies={"library-middle": {"version": "2.0.0"}}), "version must be nonempty text"),
        )
        for label, path, action, fragment in cases:
            results[f"manifest_{label}_rejected"] = mutate(path, action, fragment)
        for value in ("latest", "1.*", "^1.0.0", "~1.0.0", ">=1.0.0"):
            results[f"manifest_version_range_{value}_rejected"] = mutate(entry, lambda d, v=value: d.update(dependencies={"library-middle": v}), "ranges and latest")
        entry_bytes, middle_bytes = entry.read_bytes(), middle.read_bytes()
        short_entry, short_middle = read_yaml(entry, source_root=base), read_yaml(middle, source_root=base)
        short_entry["dependencies"]["library-middle"] = "2.0"
        short_middle["version"] = "2.0"
        entry.write_text(json.dumps(short_entry), encoding="utf-8")
        middle.write_text(json.dumps(short_middle), encoding="utf-8")
        try:
            results["manifest_exact_two_component_version_label_binds"] = load().models["library-middle"]["manifest"]["version"] == "2.0"
        finally:
            entry.write_bytes(entry_bytes)
            middle.write_bytes(middle_bytes)
        original_entry = read_yaml(entry, source_root=base)
        original_middle = read_yaml(middle, source_root=base)
        original_entry["dependencies"]["library-middle"] = "strasse"
        original_middle["version"] = "straße"
        entry.write_text(json.dumps(original_entry), encoding="utf-8")
        middle.write_text(json.dumps(original_middle), encoding="utf-8")
        try:
            results["manifest_colliding_version_reference_is_not_corrected"] = rejected(load, "colliding version label spelling")
            original_entry["dependencies"]["library-middle"] = "straße"
            original_entry["version"] = "strasse"
            entry.write_text(json.dumps(original_entry), encoding="utf-8")
            results["manifest_unicode_version_reference_exact_spelling_binds"] = load().models["library-middle"]["manifest"]["version"] == "straße"
            results["manifest_versions_in_different_models_do_not_collide"] = load().models["connector-example"]["manifest"]["version"] == "strasse"
            duplicate_version = write_model("colliding-version", "library-middle", "strasse", {}, "example/extra")
            results["manifest_versions_collide_within_same_model"] = rejected(lambda: load([middle, core, duplicate_version]), "version label collision")
        finally:
            entry.write_bytes(entry_bytes)
            middle.write_bytes(middle_bytes)
        results["manifest_missing_transitive_candidate_rejected"] = rejected(lambda: load([middle]), "not explicitly supplied")
        results["manifest_duplicate_candidate_identity_rejected"] = rejected(lambda: load([middle, core, core]), "model ID collision")
        original = read_yaml(entry, source_root=base)
        original["dependencies"]["markitect-core"] = "1.0.0"
        entry.write_text(json.dumps(original), encoding="utf-8")
        import_model_access(load(), "example/project", "markitect/core")
        results["manifest_explicit_direct_dependency_enables_transitive_package_import"] = True

        duplicate = core.parent / "content/markitect/middle/other.yaml"
        duplicate.parent.mkdir()
        duplicate.write_text("id: other\nkind: Example\nbody: probe\nspec: {}\n", encoding="utf-8")
        results["manifest_occupied_package_cannot_split_even_with_different_ids"] = rejected(load, "occupied package")
        duplicate.unlink()
        duplicate.parent.rmdir()
        left, right = middle.parent / "content/straße", core.parent / "content/strasse"
        left.mkdir()
        right.mkdir()
        results["manifest_cross_model_empty_unicode_packages_do_not_bind"] = {"straße", "strasse"} <= set(load())
        for path in (left / "item.yaml", right / "item.yaml"):
            path.write_text("id: item\nkind: Example\nbody: probe\nspec: {}\n", encoding="utf-8")
        results["manifest_cross_model_occupied_unicode_package_collision"] = rejected(load, "package ID collision")
        for path in (left / "item.yaml", right / "item.yaml"):
            path.unlink()
        left.rmdir()
        right.rmdir()
        middle_bytes, core_bytes = middle.read_bytes(), core.read_bytes()
        for path, identifier in ((middle, "straße"), (core, "strasse")):
            data = read_yaml(path, source_root=base)
            data["id"] = identifier
            path.write_text(json.dumps(data, ensure_ascii=False), encoding="utf-8")
        results["manifest_unicode_model_identity_collision"] = rejected(load, "model ID collision")
        middle.write_bytes(middle_bytes)
        core.write_bytes(core_bytes)
        snapshot = copy.deepcopy(load())
        snapshot.models["connector-example"]["manifest"]["dependencies"]["markitect-core"] = "9.0.0"
        results["manifest_in_memory_binding_changes_revalidated"] = rejected(lambda: check_bound_models(snapshot), "requires exact version")
        entry_bytes = entry.read_bytes()
        numeric_id = read_yaml(entry, source_root=base)
        numeric_id["id"] = "٢٠٢٦.modell"
        entry.write_text(json.dumps(numeric_id), encoding="utf-8")
        try:
            results["manifest_digit_start_dotted_model_id_binds_without_namespace"] = load().entry_model == "٢٠٢٦.modell"
        finally:
            entry.write_bytes(entry_bytes)

        adjacent = base / "adjacent-metadata"
        adjacent.mkdir()
        adjacent_manifest = adjacent / "model.yaml"
        adjacent_manifest.write_text(json.dumps({"formatVersion": 1, "id": "adjacent-model", "root": ".", "dependencies": {}}), encoding="utf-8")
        adjacent_lock = adjacent / "model-lock.yaml"
        adjacent_lock.write_text("formatVersion: 1\nmodels: {}\n", encoding="utf-8")
        package = adjacent / "example"
        package.mkdir()
        (package / "item.yaml").write_text("id: item\nkind: Example\nbody: probe\nspec: {}\n", encoding="utf-8")

        def load_adjacent():
            return load_models(adjacent_manifest, [], [], read_yaml=read_yaml, discover=discover, source_root=base)

        results["manifest_root_metadata_are_not_definitions"] = set(load_adjacent()) == {"example"}
        (adjacent / "item.yaml").write_text("id: item\nkind: Example\nbody: probe\nspec: {}\n", encoding="utf-8")
        try:
            load_adjacent()
        except ProbeLimitError:
            results["manifest_root_definition_is_language_error"] = False
        except ValueError as error:
            results["manifest_root_definition_is_language_error"] = "forbidden directly in the model root" in str(error)
        else:
            results["manifest_root_definition_is_language_error"] = False
    return results
