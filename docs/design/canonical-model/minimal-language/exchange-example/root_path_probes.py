"""On-disk root spelling, containment, overlap and redirection probes."""
from __future__ import annotations

import json
import os
import subprocess
import tempfile
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

from name_profile import ProbeLimitError
from root_paths import model_root_path, reject_overlapping_roots, reject_redirect, root_components


def root_path_probes(load_models, read_yaml, discover):
    results = {}

    def rejected(action, fragment):
        try:
            action()
        except ProbeLimitError:
            return False
        except ValueError as error:
            return fragment in str(error)
        return False

    for value in (".", "model", "docs/model", "überweisung/文档"):
        results[f"root_canonical_form_{value}"] = root_components(value) == (() if value == "." else tuple(value.split("/")))
    for value in ("", "/model", "C:/model", "C:model", "\\\\host\\model", "./model", "model/", "docs//model", "a/./b", "a/../b", "..", "a\\b", "Model", "a_b", "a-b", "a.b", "2model", "u\u0308berweisung", "📄"):
        results[f"root_invalid_form_{value}"] = rejected(lambda v=value: root_components(v), "model root")

    # Exercise the generic Windows reparse attribute even on non-Windows hosts.
    with patch.object(Path, "lstat", return_value=SimpleNamespace(st_mode=0, st_file_attributes=0x400)):
        with patch("root_paths.stat.FILE_ATTRIBUTE_REPARSE_POINT", 0x400, create=True):
            results["root_generic_reparse_attribute_rejected"] = rejected(lambda: reject_redirect(Path("probe")), "redirection is forbidden")

    with tempfile.TemporaryDirectory(prefix="markitect-root-paths-") as temporary:
        base = Path(temporary)

        def write_model(folder, identifier, root_value, actual_root, dependencies=None, version=None):
            directory = base / folder
            directory.mkdir(parents=True)
            root = directory if actual_root == "." else directory / actual_root
            root.mkdir(parents=True, exist_ok=True)
            manifest = {"formatVersion": 1, "id": identifier, "root": root_value, "dependencies": dependencies or {}}
            if version is not None:
                manifest["version"] = version
            path = directory / "model.yaml"
            path.write_text(json.dumps(manifest, ensure_ascii=False), encoding="utf-8")
            return path, root

        def load(entry, candidates=()):
            return load_models(entry, list(candidates), [], read_yaml=read_yaml, discover=discover, source_root=base)

        manifest, root = write_model("ordinary", "ordinary", "docs/model", "docs/model")
        package = root / "feature"
        package.mkdir()
        definition = package / "item.yaml"
        definition.write_text("id: item\nkind: Example\nbody: probe\nspec: {}\n", encoding="utf-8")
        before = manifest.read_bytes(), definition.read_bytes()
        results["root_existing_nested_directory_loads"] = set(load(manifest)) == {"feature"}
        results["root_directory_prefix_is_not_package_prefix"] = "docs/model/feature" not in load(manifest)
        results["root_loading_preserves_manifest_and_definition"] = before == (manifest.read_bytes(), definition.read_bytes())
        for value in ("./docs/model", "docs//model", "docs/model/"):
            data = json.loads(before[0])
            data["root"] = value
            manifest.write_text(json.dumps(data), encoding="utf-8")
            try:
                results[f"root_loader_does_not_normalize_{value}"] = rejected(lambda: load(manifest), "canonical relative / components")
            finally:
                manifest.write_bytes(before[0])
        dot, dot_root = write_model("Physical-Parent", "dot-root", ".", ".")
        results["root_dot_allows_free_physical_parent_name"] = model_root_path(dot, ".") == dot_root and not load(dot)
        unicode_manifest, unicode_root = write_model("unicode", "unicode-root", "überweisung/文档", "überweisung/文档")
        results["root_unicode_physical_path_loads"] = model_root_path(unicode_manifest, "überweisung/文档") == unicode_root
        wrong_case, actual = write_model("case", "case-root", "model", "Model")
        results["root_actual_case_must_match_even_on_windows"] = rejected(lambda: load(wrong_case), "physical spelling") and actual.name == "Model"
        wrong_nfc, _ = write_model("nfc", "nfc-root", "überweisung", "u\u0308berweisung")
        results["root_actual_nfc_spelling_must_match"] = rejected(lambda: load(wrong_nfc), "physical spelling")
        collision, collision_root = write_model("collision", "collision-root", "strasse", "strasse")
        (collision_root.parent / "straße").mkdir()
        results["root_caseless_physical_collision_rejected"] = rejected(lambda: load(collision), "root folder collision")
        results["root_missing_directory_is_error"] = rejected(lambda: model_root_path(manifest, "missing"), "does not exist")
        (manifest.parent / "file").write_text("probe", encoding="utf-8")
        results["root_file_is_not_directory"] = rejected(lambda: model_root_path(manifest, "file"), "existing directory")
        parent, _ = write_model("nested", "parent", ".", ".", {"child": "1.0"})
        child, _ = write_model("nested/inner", "child", ".", ".", version="1.0")
        results["root_loaded_nested_models_rejected_before_discovery"] = rejected(lambda: load(parent, [child]), "model roots overlap")
        results["root_same_physical_directory_rejected"] = rejected(
            lambda: reject_overlapping_roots({"one": {"root": root}, "two": {"root": root}}), "model roots overlap")
        other, other_root = write_model("other", "other", "model", "model", version="1.0")
        main_data = json.loads(manifest.read_text(encoding="utf-8"))
        main_data["dependencies"] = {"other": "1.0"}
        manifest.write_text(json.dumps(main_data), encoding="utf-8")
        results["root_disjoint_models_load"] = set(load(manifest, [other]).models) == {"ordinary", "other"}

        # Junctions need no symlink privilege on Windows. Both endpoints stay in
        # this task-owned temporary tree; release the link before cleanup.
        link = root / "linked"
        if os.name == "nt":
            env = dict(os.environ, MARKITECT_PROBE_LINK=str(link), MARKITECT_PROBE_TARGET=str(other_root))
            subprocess.run(["powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
                            "$ErrorActionPreference = 'Stop'; New-Item -ItemType Junction -Path $env:MARKITECT_PROBE_LINK -Target $env:MARKITECT_PROBE_TARGET | Out-Null"],
                           env=env, check=True, capture_output=True, creationflags=subprocess.CREATE_NO_WINDOW)
        else:
            link.symlink_to(other_root, target_is_directory=True)
        try:
            results["root_real_directory_redirection_rejected"] = rejected(lambda: model_root_path(manifest, "docs/model/linked"), "redirection is forbidden")
            results["root_real_tree_redirection_rejected"] = rejected(lambda: load(manifest, [other]), "redirection is forbidden")
            results["root_redirection_rejection_preserves_target"] = other_root.is_dir() and link.exists()
        finally:
            if os.name == "nt":
                link.rmdir()
            else:
                link.unlink()
    return results
