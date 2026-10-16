#!/usr/bin/env python3
"""Check the library fixture with the existing bounded v1 design probe.

This is not a production compiler or a behavioral test of generated code.
Checking is read-only. Only --write-lock explicitly regenerates the M1 lock.
"""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import sys

BASE = Path(__file__).resolve().parent
SOURCES = BASE.parent
PROBE = SOURCES / "minimal-language" / "exchange-example"
sys.path.insert(0, str(PROBE))

import verify as language
from model_lock import check_lock


def assess(*, write_lock: bool = False) -> dict:
    manifest = BASE / "m1" / "model.yaml"
    supplied = [BASE / "vocabulary" / "model.yaml", PROBE / "dependencies" / "core" / "model.yaml"]
    diagnostics = []
    packages = language.load_models(
        manifest, supplied, diagnostics, read_yaml=language.load_yaml,
        discover=language.discover_packages, source_root=SOURCES,
    )
    if diagnostics:
        raise ValueError("; ".join(diagnostics))
    counts = language.validate(packages)
    digests = {
        identifier: language.canonical_digest(language.model_bytes(packages, identifier))
        for identifier in sorted(packages.models)
    }
    external = set(packages.models) - {packages.entry_model}
    external_digests = {identifier: digests[identifier] for identifier in sorted(external)}
    lock_path = manifest.with_name("model-lock.yaml")
    if write_lock:
        if lock_path.exists() or lock_path.is_symlink():
            language.reject_redirect(lock_path)
        lock = {"formatVersion": 1, "models": {
            identifier: {"version": packages.models[identifier]["manifest"]["version"],
                         "digest": external_digests[identifier]}
            for identifier in sorted(external)
        }}
        check_lock(lock, packages, external_digests)
        lock_path.write_text(language.yaml.safe_dump(lock, sort_keys=False), encoding="utf-8")
    lock_result = language.load_lock(lock_path, packages, read_yaml=language.load_yaml,
                                     source_root=SOURCES, computed_digests=external_digests)
    if lock_result["status"] != "passed":
        raise ValueError("External content binding was not completed")

    local = {name: package for name, package in packages.items()
             if package["model"] == packages.entry_model and package["definitions"]}
    links = language.check_body_links(packages)
    cross_package = sorted({
        f"{item['definition']['package']}/{item['definition']['id']} -> "
        f"{item['target']['package']}/{item['target']['id']}"
        for item in links
        if item["definition"]["package"] in local
        and item["target"]["package"] in local
        and item["definition"]["package"] != item["target"]["package"]
    })
    inputs = {Path(__file__).resolve(), language.CONTRACT, lock_path,
              PROBE / "requirements.txt", *PROBE.glob("*.py")}
    for model in packages.models.values():
        inputs.add(model["path"])
    for package in packages.values():
        inputs.update(package["paths"].values())
    return {
        "status": "passed",
        "scope": "Bounded v1 structural, reference, body-link, canonical-content and lock check; no behavioral Apply",
        "unicodeVersion": language.UNICODE_VERSION,
        "pythonVersion": sys.version.split()[0],
        "pyYamlVersion": language.package_version("PyYAML"),
        "markdownItVersion": language.package_version("markdown-it-py"),
        "entryModel": packages.entry_model,
        "localDefinitions": sum(len(package["definitions"]) for package in local.values()),
        "localPackages": sorted(local),
        "checkedDefinitionsIncludingDependencies": counts["definitions"],
        "bodyLinkOccurrencesIncludingDependencies": counts["body_links"],
        "modelDigests": digests,
        "lock": lock_result,
        "crossPackageLinks": cross_package,
        "inputSha256": {
            path.relative_to(SOURCES).as_posix(): hashlib.sha256(path.read_bytes()).hexdigest()
            for path in sorted(inputs)
        },
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--write-lock", action="store_true", help="explicitly regenerate M1's external content lock")
    parser.add_argument("--report", type=Path, help="write the complete JSON evidence to this explicit path")
    args = parser.parse_args()
    try:
        result = assess(write_lock=args.write_lock)
    except language.ProbeLimitError as error:
        result = {"status": "not_completed", "diagnostics": [str(error)]}
    except (OSError, ValueError) as error:
        result = {"status": "failed", "diagnostics": [str(error)]}
    if args.report:
        args.report.write_text(json.dumps(result, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    summary = {key: value for key, value in result.items()
               if key not in {"inputSha256", "crossPackageLinks"}}
    if "crossPackageLinks" in result:
        summary["crossPackageLinkCount"] = len(result["crossPackageLinks"])
    print(json.dumps(summary, indent=2, sort_keys=True))
    return 0 if result["status"] == "passed" else 1


if __name__ == "__main__":
    raise SystemExit(main())
