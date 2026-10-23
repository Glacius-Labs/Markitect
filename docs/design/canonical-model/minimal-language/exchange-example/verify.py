#!/usr/bin/env python3
"""Bounded structural verifier for the canonical-model Exchange example.

This is a design probe, not Markitect's production loader or compiler.
Requires PyYAML. Packages are derived from folders under the fixture model root.
Manifests bind exact versions from explicitly supplied local models; lock bindings are checked separately; model-digest/1 canonical bytes and SHA-256 verified per valid model scope. PyYAML parses syntax only; the explicit
profile below owns scalar interpretation. The Markitect product remains Go.
"""

from __future__ import annotations

import argparse
import copy
import hashlib
import json
import re
import sys
import tempfile
import unicodedata
from decimal import Decimal, InvalidOperation
from importlib.metadata import version as package_version
from pathlib import Path
from typing import Any

import yaml

from name_profile import UNICODE_VERSION, WHITE_SPACE, ProbeLimitError, check_unicode_version, require_nfc, name_key, word_matches, name_matches, filename_matches
from model_manifest import ModelPackages, load_models, check_bound_models, import_model_access, manifest_probes, package_binding_names
from model_lock import load_lock, lock_probes
from canonical_bytes import canonical_model_bytes, digest as canonical_digest
from canonical_probes import canonical_probes
from core_probes import core_equivalence_probes
from root_paths import reject_redirect
from root_path_probes import root_path_probes
from resources import DEFAULT_OBJECT_DEPTH, check_object_depth, overall_status, resource_checked
from object_resource_probes import object_resource_probes
from package_probes import package_review_probes, empty_package_binding_probes
from body_links import MAX_BODY_BYTES, MAX_NESTING, BodyLinkLimitError, BodyLinkSyntaxError, parse_body_links
from diagnostic_coordinates import attach_yaml_source, yaml_body_source, with_yaml_coordinates
from diagnostic_probes import diagnostic_probes


ROOT = Path(__file__).resolve().parent
MODEL_ROOT = ROOT / "model"
ENTRY_MANIFEST = ROOT / "model.yaml"
LOCK_PATH = ROOT / "model-lock.yaml"
SUPPLIED_MANIFESTS = [ROOT / "dependencies" / "core" / "model.yaml"]
CONTRACT = ROOT.parent / "language-contract.md"
DIAGNOSTIC_CONTRACT = ROOT.parent / "diagnostic-contract.md"
KIND_STYLE = "pascal"
PROPERTY_STYLE = "camel"
CONCRETE_STYLE = "kebab"
CORE_KIND_IDS = {"Kind", "Property"}
CORE_PROPERTY_IDS = {"name", "type", "minCount", "maxCount", "values", "target", "properties"}
SCALAR_TYPES = {"string", "boolean", "integer", "number", "enum", "object", "reference"}
RESOLUTION_CALLS = 0
INTEGER_TOKEN = re.compile(r"-?(?:0|[1-9][0-9]*)\Z", re.ASCII)
DECIMAL_TOKEN = re.compile(r"-?(?:0|[1-9][0-9]*)(?:\.[0-9]+(?:[eE][+-]?[0-9]+)?|[eE][+-]?[0-9]+)\Z", re.ASCII)
# Probe resource ceilings, not language number ranges. No rounding on overflow.
MAX_NUMBER_DIGITS = 100_000
MAX_ABS_EXPONENT = 1_000_000_000


class DuplicateKeyError(ValueError):
    pass


def integer_value(text: str) -> int:
    """Build arbitrarily precise integers without Python's string digit limit."""
    digits = text.lstrip("-")
    if len(digits) > MAX_NUMBER_DIGITS:
        raise ProbeLimitError(f"probe resource limit: number exceeds {MAX_NUMBER_DIGITS} coefficient digits")
    value = 0
    for start in range(0, len(digits), 9):
        chunk = digits[start:start + 9]
        value = value * 10 ** len(chunk) + int(chunk)
    return -value if text.startswith("-") else value


def scalar_value(node: yaml.ScalarNode) -> Any:
    text = node.value
    if any(0xD800 <= ord(char) <= 0xDFFF for char in text):
        fail("YAML string contains an unpaired Unicode surrogate")
    if node.style is not None:  # Quoted and block scalars are always strings.
        return text
    if text in {"", "null", "Null", "NULL", "~"}:
        fail("null and empty YAML values are forbidden; omit an optional field")
    if text in {"true", "false"}:
        return text == "true"
    if INTEGER_TOKEN.fullmatch(text):
        return integer_value(text)
    if DECIMAL_TOKEN.fullmatch(text):
        coefficient, *exponent = re.split("[eE]", text)
        if sum(char.isdecimal() for char in coefficient) > MAX_NUMBER_DIGITS:
            raise ProbeLimitError(f"probe resource limit: number exceeds {MAX_NUMBER_DIGITS} coefficient digits")
        if exponent:
            exponent_digits = exponent[0].lstrip("+-").lstrip("0") or "0"
            if len(exponent_digits) > 10 or int(exponent_digits) > MAX_ABS_EXPONENT:
                raise ProbeLimitError(f"probe resource limit: absolute decimal exponent exceeds {MAX_ABS_EXPONENT}")
        try:
            return Decimal(text)  # Construction retains every decimal digit.
        except InvalidOperation:
            raise ProbeLimitError("probe resource limit: decimal cannot be represented exactly")
    if text.lower() in {".nan", "nan", ".inf", "+.inf", "-.inf", "inf", "+inf", "-inf",
                        "infinity", "+infinity", "-infinity"}:
        fail("non-finite numeric tokens are forbidden; quote them for literal text")
    return text  # Includes on, yes, dates, and nondecimal-looking text.


def decode_yaml_node(node: yaml.Node) -> Any:
    if isinstance(node, yaml.ScalarNode):
        return scalar_value(node)
    if isinstance(node, yaml.SequenceNode):
        return [decode_yaml_node(item) for item in node.value]
    if isinstance(node, yaml.MappingNode):
        result: dict[str, Any] = {}
        normalized: dict[str, str] = {}
        for key_node, value_node in node.value:
            if not isinstance(key_node, yaml.ScalarNode):
                fail("YAML mapping keys must be strings")
            key = scalar_value(key_node)
            if not isinstance(key, str):
                fail("YAML mapping keys must be strings; quote a literal numeric or boolean name")
            require_nfc(key, "YAML field name")
            if key == "<<":
                fail("YAML merge keys are forbidden")
            canonical = name_key(key)
            if canonical in normalized:
                raise DuplicateKeyError(f"case-insensitive duplicate YAML key: {key!r} collides with {normalized[canonical]!r}")
            normalized[canonical] = key
            result[key] = decode_yaml_node(value_node)
        return result
    fail("unsupported YAML node")


def load_yaml(path: Path, body_source: dict | None = None, *, source_root: Path = ROOT) -> Any:
    return parse_yaml(path.read_bytes().decode("utf-8"), body_source=body_source, file=path.relative_to(source_root).as_posix())


@resource_checked("YAML parsing")
def parse_yaml(text: str, *, body_source: dict | None = None, file: str = "<memory>") -> Any:
    check_unicode_version()
    try:
        for token in yaml.scan(text, Loader=yaml.BaseLoader):
            if isinstance(token, (yaml.tokens.AnchorToken, yaml.tokens.AliasToken)):
                fail("YAML anchors and aliases are forbidden")
            if isinstance(token, yaml.tokens.TagToken):
                fail("explicit YAML tags are forbidden; use quoted strings and ordinary values")
            if isinstance(token, yaml.tokens.DirectiveToken):
                if token.name != "YAML" or token.value != (1, 2):
                    fail("only the YAML 1.2 version directive is allowed")
        node = yaml.compose(text, Loader=yaml.BaseLoader)
    except yaml.YAMLError as error:
        fail(f"invalid YAML document: {error}")
    if not isinstance(node, yaml.MappingNode):
        fail("exactly one YAML document with a mapping root is required")
    result = decode_yaml_node(node)
    if body_source is not None and isinstance(result.get("body"), str):
        body_node = next(value for key, value in node.value if key.value == "body")
        body_source.update({"file": file, "text": text, "node": {"start": body_node.start_mark.index,
                            "end": body_node.end_mark.index, "style": body_node.style}, "decoded": result["body"]})
    return result


def fail(message: str) -> None:
    raise ValueError(message)


def check_package_name(value: str) -> None:
    require_nfc(value, "package ID")
    if not name_matches(value, "lower", "/"):
        fail(f"package ID {value!r} must use lowercase components separated by /; no path normalization")


def folder_package(parts: tuple[str, ...]) -> str:
    """One folder component per package component; no path normalization."""
    if not parts:
        fail("Definitions are forbidden directly in the model root; use a named package subfolder")
    for part in parts:
        require_nfc(part, "package folder")
        if not name_matches(part, "lower", None):
            fail(f"package folder {part!r} must be one lowercase name component; no dots or path shorthand")
    return "/".join(parts)


def discover_packages(model_root: Path, diagnostics: list[str], *, source_root: Path | None = None,
                      metadata_paths: tuple[Path, ...] = ()) -> dict[str, dict[str, Any]]:
    """Discover all folders, including parents without Definitions, deterministically.

    The model root is not a Package and cannot contain Definitions. Symlink/reparse
    redirections are language errors and are never traversed.
    """
    reject_redirect(model_root)
    if not model_root.is_dir():
        fail("fixture model root is missing")
    packages: dict[str, dict[str, Any]] = {}
    source_root = source_root or model_root.parent
    pending = [model_root]
    while pending:
        directory = pending.pop()
        entries = sorted(directory.iterdir(), key=lambda path: path.name)
        for entry in entries:
            reject_redirect(entry)
        directories = [entry for entry in entries if entry.is_dir()]
        pending.extend(reversed(directories))
        files = [entry for entry in entries if entry.is_file() and not any(
            entry == metadata and entry.name == metadata.name for metadata in metadata_paths)]
        for path in files:
            if path.name.lower().endswith((".yaml", ".yml")) and not path.name.endswith(".yaml"):
                fail(f"{path.relative_to(source_root).as_posix()}: definition filename must use exact .yaml; .yml and case variants are forbidden")
        yaml_files = [entry for entry in files if entry.name.endswith(".yaml")]
        if directory == model_root:
            if yaml_files:
                folder_package(())
            continue
        package_name = folder_package(directory.relative_to(model_root).parts)
        check_filename_list([entry.name for entry in yaml_files], directory.relative_to(source_root).as_posix())
        definitions: dict[str, dict[str, Any]] = {}
        paths: dict[str, Path] = {}
        body_sources: dict[str, dict] = {}
        for path in yaml_files:
            if not filename_matches(path.name):
                fail(f"{path.relative_to(source_root).as_posix()}: filename must be lowercase kebab-case with .yaml")
            body_source: dict = {}
            definition = load_yaml(path, body_source, source_root=source_root)
            definition_id = definition.get("id")
            if not isinstance(definition_id, str) or not definition_id:
                fail(f"{path.relative_to(source_root).as_posix()}: missing nonempty id")
            require_nfc(definition_id, "definition ID")
            if "." in definition_id:
                fail(f"definition ID {definition_id!r} must not contain dots")
            existing = next((known for known in definitions if name_key(known) == name_key(definition_id)), None)
            if existing is not None:
                diagnostics.append(f"{directory.relative_to(source_root).as_posix()}: case-insensitive definition ID collision {existing!r} vs {definition_id!r}")
            if definition_id in definitions:
                continue
            definitions[definition_id] = definition
            paths[definition_id] = path
            body_sources[definition_id] = body_source
        packages[package_name] = {"directory": directory, "definitions": definitions,
                                  "paths": paths, "body_sources": body_sources}
    check_identity_collisions(packages, definitions=False)
    return packages


@resource_checked("model loading")
def package_data(diagnostics: list[str]) -> dict[str, dict[str, Any]]:
    return load_models(ENTRY_MANIFEST, SUPPLIED_MANIFESTS, diagnostics, read_yaml=load_yaml,
                       discover=discover_packages, source_root=ROOT)


def anchored_section(source: str, name: str) -> str:
    """Read a documented section without depending on its title or numbering."""
    marker = f'<a id="{name}"></a>'
    if source.count(marker) != 1:
        fail(f"documentation must contain exactly one {name!r} section anchor")
    return source.split(marker, 1)[1].split('<a id="', 1)[0]


def core_bootstrap(packages: dict[str, dict[str, Any]]) -> None:
    core = packages["markitect/core"]["definitions"]
    if set(core) != CORE_KIND_IDS | CORE_PROPERTY_IDS:
        fail("core must contain exactly Kind, Property, and the seven selected meta-properties")
    contract = CONTRACT.read_text(encoding="utf-8")
    section = anchored_section(contract, "core-definitions")
    blocks = re.findall(r"\*\*([^*]+)\.yaml\*\*\s+```yaml\s*\n(.*?)\n```", section, re.S)
    documented = {parse_yaml(source)["id"]: parse_yaml(source) for _, source in blocks}
    expected = copy.copy(packages)
    expected["markitect/core"] = {**packages["markitect/core"], "definitions": documented}
    if len(blocks) != 9 or len(documented) != 9 or core_declaration_view(expected) != core_declaration_view(packages):
        fail("fixture Core definitions differ from the nine normative YAML definitions")


def core_declaration_view(packages: dict[str, dict[str, Any]]) -> dict[str, Any]:
    """Compare resolved seed contents without recursively trusting its contracts.

    Keep unknown fields and duplicate declarations so comparison cannot hide
    them. Ordinary validation still checks the complete input afterwards.
    """
    package = "markitect/core"
    result = {}
    for identifier, source in packages[package]["definitions"].items():
        # Resolution validates all imports, including unused ones, before we
        # omit their author spelling from the comparison.
        role = resolve(packages, package, source, source.get("kind"))
        definition = copy.deepcopy(source)
        definition.pop("imports", None)
        definition["kind"] = role
        spec = definition.get("spec")
        if not isinstance(spec, dict):
            fail(f"{identifier}: Core spec must be a mapping")
        if role == (package, "Property"):
            spec.setdefault("name", source["id"])
            if spec.get("type") == "reference" and "target" in spec:
                spec["target"] = resolve(packages, package, source, spec["target"])
        if role == (package, "Kind") or (role == (package, "Property") and spec.get("type") == "object"):
            references = spec.get("properties", [])
            if not isinstance(references, list):
                fail(f"{identifier}: Core properties must be a list")
            spec["properties"] = sorted(resolve(packages, package, source, reference) for reference in references)
        if role == (package, "Property") and spec.get("type") == "enum":
            values = spec.get("values")
            if not isinstance(values, list) or any(not isinstance(value, str) for value in values):
                fail(f"{identifier}: Core enum values must be a list of strings")
            spec["values"] = sorted(values)
        result[identifier] = definition
    return result


def package_imports(packages: dict[str, dict[str, Any]], package_name: str, definition: dict[str, Any]) -> list[dict[str, str]]:
    imports = definition.get("imports", [])
    if not isinstance(imports, list) or any(not isinstance(item, dict) or set(item) - {"package", "as"} or not isinstance(item.get("package"), str) or not item.get("package") for item in imports):
        fail(f"{definition.get('id')}: imports must be a list of mappings with package and optional as")
    names: set[str] = set()
    aliases: dict[str, str] = {}
    pending_aliases: list[tuple[str, str]] = []
    for item in imports:
        imported = item["package"]
        require_nfc(imported, "package import")
        folded = name_key(imported)
        if folded in names:
            fail(f"{definition.get('id')}: duplicate or case-colliding package import {imported!r}")
        names.add(folded)
        alias = item.get("as")
        if "as" in item:
            if not isinstance(alias, str) or not alias:
                fail(f"{definition.get('id')}: import as must be a nonempty alias; omit as for no alias")
            require_nfc(alias, "import alias")
            if name_key(alias) in aliases:
                fail(f"{definition.get('id')}: case-insensitive alias collision {aliases[name_key(alias)]!r} vs {alias!r}")
            aliases[name_key(alias)] = alias
            pending_aliases.append((alias, imported))
        if imported not in packages:
            near = next((name for name in packages if name_key(name) == folded), None)
            if near:
                fail(f"{definition.get('id')}: wrong-case package import {imported!r}; write {near!r}")
            fail(f"{definition.get('id')}: imported package {imported!r} is absent from this bounded fixture")
        import_model_access(packages, package_name, imported)
    for alias, _ in pending_aliases:
        if not name_matches(alias, "pascal", None):
            fail(f"{definition.get('id')}: import alias {alias!r} must be PascalCase")
    return imports


def check_filename_list(listed: Any, label: str = "package directory") -> None:
    if not isinstance(listed, list) or any(not isinstance(name, str) for name in listed):
        fail(f"{label}: expected a list of filenames")
    folded: dict[str, str] = {}
    for name in listed:
        require_nfc(name, "definition filename")
        previous = folded.get(name_key(name))
        if previous:
            fail(f"{label}: case-insensitive filename collision {previous!r} vs {name!r}")
        folded[name_key(name)] = name


def resolve(packages: dict[str, dict[str, Any]], package_name: str, definition: dict[str, Any], name: Any) -> tuple[str, str]:
    global RESOLUTION_CALLS
    if not isinstance(name, str) or not name:
        fail(f"{definition.get('id')}: expected nonempty reference")
    require_nfc(name, "reference")
    imports = package_imports(packages, package_name, definition)
    visible = [(package_name, None), *((item["package"], item.get("as")) for item in imports)]
    exact: set[tuple[str, str]] = set()
    normalized: set[tuple[str, str]] = set()
    spellings: dict[tuple[str, str], set[str]] = {}
    for candidate_package, alias in visible:
        for candidate_id in packages[candidate_package]["definitions"]:
            target = (candidate_package, candidate_id)
            accepted = [candidate_id, f"{candidate_package}/{candidate_id}"]
            if alias:
                accepted.append(f"{alias}.{candidate_id}")
            spellings.setdefault(target, set()).update(accepted)
            if name in accepted:
                exact.add(target)
            if name_key(name) in {name_key(spelling) for spelling in accepted}:
                normalized.add(target)
    if len(exact) > 1 or (exact and len(normalized) > 1):
        fail(f"{definition.get('id')}: reference {name!r} is ambiguous under case-insensitive matching")
    if not exact:
        if len(normalized) == 1:
            target = next(iter(normalized))
            suggestion = sorted(spelling for spelling in spellings[target] if name_key(spelling) == name_key(name))
            fail(f"{definition.get('id')}: wrong-case reference {name!r}; write {suggestion[0]!r}")
        if len(normalized) > 1:
            fail(f"{definition.get('id')}: reference {name!r} is ambiguous under case-insensitive matching")
        fail(f"{definition.get('id')}: reference {name!r} is unresolved")
    RESOLUTION_CALLS += 1
    return next(iter(exact))


def kind_identity(packages: dict[str, dict[str, Any]], package_name: str, definition: dict[str, Any]) -> tuple[str, str]:
    return resolve(packages, package_name, definition, definition.get("kind"))


def validate_spec(packages: dict[str, dict[str, Any]], package_name: str, definition: dict[str, Any], owner_kind: str,
                  spec: Any, *, owner_package: str | None = None, schema_owner: dict | None = None,
                  object_depth: int = 0, max_object_depth: int = DEFAULT_OBJECT_DEPTH) -> None:
    check_object_depth(object_depth, max_object_depth)
    if not isinstance(spec, dict):
        fail(f"{definition.get('id')}: spec/object value must be a mapping")
    owner_package = owner_package or package_name
    owner = schema_owner if schema_owner is not None else packages[owner_package]["definitions"].get(owner_kind)
    if owner is None:
        fail(f"{definition.get('id')}: unresolved owner kind {owner_kind!r}")
    schema_ids = owner.get("spec", {}).get("properties", [])
    if not isinstance(schema_ids, list):
        fail(f"{owner_kind}: properties schema must be a list")
    schema: dict[str, tuple[str, str]] = {}
    for reference in schema_ids:
        target_package, target_id = resolve(packages, owner_package, owner, reference)
        target = packages[target_package]["definitions"][target_id]
        if kind_identity(packages, target_package, target) != ("markitect/core", "Property"):
            fail(f"{owner_kind}: schema target {reference!r} must be a Property definition")
        effective = target.get("spec", {}).get("name", target_id)
        if not isinstance(effective, str) or not effective:
            fail(f"{target_id}: effective property name must be nonempty")
        check_segment_style(effective, PROPERTY_STYLE, "effective property field name")
        collision = next((name for name in schema if name_key(name) == name_key(effective)), None)
        if collision is not None:
            fail(f"{owner_kind}: case-insensitive effective field collision {collision!r} vs {effective!r}")
        schema[effective] = (target_package, target_id)
    for field in spec:
        require_nfc(field, "spec field name")
    extra = set(spec) - set(schema)
    if extra:
        for field in sorted(extra):
            match = next((known for known in schema if name_key(field) == name_key(known)), None)
            if match:
                fail(f"{definition.get('id')}: wrong-case field {field!r}; write {match!r}")
        fail(f"{definition.get('id')}: unknown field(s) {sorted(extra)!r}")
    for field_name, (property_package, property_id) in schema.items():
        prop = packages[property_package]["definitions"][property_id]
        contract = prop.get("spec", {})
        minimum = contract.get("minCount")
        maximum = contract.get("maxCount")
        if field_name not in spec:
            if minimum > 0:
                fail(f"{definition.get('id')}: required field {field_name!r} is missing")
            continue
        value = spec[field_name]
        is_list = maximum != 1
        values = value if is_list else [value]
        if is_list and not isinstance(value, list):
            fail(f"{definition.get('id')}.{field_name}: expected list")
        if not is_list and isinstance(value, list):
            fail(f"{definition.get('id')}.{field_name}: expected scalar")
        if len(values) < minimum or (maximum != -1 and len(values) > maximum):
            fail(f"{definition.get('id')}.{field_name}: cardinality outside {minimum}..{maximum}")
        value_type = contract.get("type")
        if value_type == "string":
            if any(not isinstance(item, str) for item in values):
                fail(f"{definition.get('id')}.{field_name}: expected string")
        elif value_type == "boolean":
            if any(not isinstance(item, bool) for item in values):
                fail(f"{definition.get('id')}.{field_name}: expected boolean")
        elif value_type == "integer":
            if any(not isinstance(item, int) or isinstance(item, bool) for item in values):
                fail(f"{definition.get('id')}.{field_name}: expected integer")
        elif value_type == "number":
            if any(isinstance(item, bool) or not isinstance(item, (int, Decimal)) or
                   (isinstance(item, Decimal) and not item.is_finite()) for item in values):
                fail(f"{definition.get('id')}.{field_name}: expected exact finite number")
        elif value_type == "enum":
            allowed = contract.get("values")
            if not isinstance(allowed, list) or not allowed or len(allowed) != len(set(allowed)):
                fail(f"{property_id}: enum values must be nonempty and unique")
            if any(not isinstance(item, str) or item not in allowed for item in values):
                fail(f"{definition.get('id')}.{field_name}: value outside enum")
        elif value_type == "reference":
            target_kind = contract.get("target")
            if not target_kind:
                fail(f"{property_id}: reference requires target")
            for item in values:
                target_package, target_id = resolve(packages, package_name, definition, item)
                target_definition = packages[target_package]["definitions"][target_id]
                expected_kind_package, expected_kind_id = resolve(packages, property_package, prop, target_kind)
                actual_kind = target_definition.get("kind")
                if not isinstance(actual_kind, str):
                    fail(f"{target_id}: missing kind")
                actual_kind_package, actual_kind_id = resolve(packages, target_package, target_definition, actual_kind)
                if (actual_kind_package, actual_kind_id) != (expected_kind_package, expected_kind_id):
                    fail(f"{definition.get('id')}.{field_name}: target {item!r} has wrong kind")
        elif value_type == "object":
            for item in values:
                nested_owner = {"id": f"{definition.get('id')}.{field_name}", "imports": definition.get("imports", [])}
                nested_schema = contract.get("properties", [])
                if not isinstance(nested_schema, list):
                    fail(f"{property_id}: object properties must be a list")
                validate_spec(packages, package_name, nested_owner, property_id, item,
                              owner_package=property_package, schema_owner=prop,
                              object_depth=object_depth + 1, max_object_depth=max_object_depth)
        else:
            fail(f"{property_id}: unknown core value type {value_type!r}")
        if value_type == "enum" and "values" not in contract:
            fail(f"{property_id}: enum requires values")
        if value_type != "enum" and "values" in contract:
            fail(f"{property_id}: values is only valid for enum")
        if value_type == "reference" and "target" not in contract:
            fail(f"{property_id}: reference requires target")
        if value_type != "reference" and "target" in contract:
            fail(f"{property_id}: target is only valid for reference")
        if value_type != "object" and "properties" in contract and property_id != "properties":
            fail(f"{property_id}: properties is only valid for object")


@resource_checked("structural validation")
def validate(packages: dict[str, dict[str, Any]], *, max_object_depth: int = DEFAULT_OBJECT_DEPTH) -> dict[str, int]:
    global RESOLUTION_CALLS
    RESOLUTION_CALLS = 0
    check_unicode_version()
    core_bootstrap(packages)
    check_identity_collisions(packages)
    check_bound_models(packages)
    for package_name, package in packages.items():
        check_package_name(package_name)
        filenames = [path.name for path in package["paths"].values()]
        check_filename_list(filenames)
        if any(not filename_matches(filename) for filename in filenames):
            fail("definition filename must use lowercase kebab-case with .yaml")
        for identifier, definition in list(package["definitions"].items()):
            path = package["paths"].get(identifier)
            allowed_hull = {"imports", "id", "kind", "body", "spec", "title"}
            if set(definition) - allowed_hull or not {"id", "kind", "body", "spec"} <= set(definition):
                fail(f"{path.relative_to(ROOT) if path else identifier}: invalid definition hull")
            require_nfc(identifier, "definition ID")
            if "." in identifier:
                fail(f"definition ID {identifier!r} must not contain dots")
            if identifier != definition["id"]:
                fail(f"{identifier}: stored ID differs from declaration")
            if not isinstance(definition["body"], str) or not any(char not in WHITE_SPACE for char in definition["body"]):
                fail(f"{identifier}: body must be text containing a character outside Unicode White_Space")
            if "title" in definition and not isinstance(definition["title"], str):
                fail(f"{identifier}: title must be text")
            imports = package_imports(packages, package_name, definition)
            kind_package, kind_id = resolve(packages, package_name, definition, definition["kind"])
            if (kind_package, kind_id) == ("markitect/core", "Kind"):
                check_segment_style(identifier, KIND_STYLE, "Kind ID")
            elif (kind_package, kind_id) == ("markitect/core", "Property"):
                check_segment_style(identifier, PROPERTY_STYLE, "Property ID")
            else:
                check_segment_style(identifier, CONCRETE_STYLE, "concrete definition ID")
            kind_definition = packages[kind_package]["definitions"][kind_id]
            if not isinstance(kind_definition.get("kind"), str):
                fail(f"{identifier}: kind target {kind_id!r} has no kind")
            metakind_package, metakind_id = resolve(packages, kind_package, kind_definition, kind_definition["kind"])
            if (metakind_package, metakind_id) != ("markitect/core", "Kind"):
                fail(f"{identifier}: kind target {kind_id!r} is not itself a Kind definition")
            validate_spec(packages, package_name, definition, kind_id, definition["spec"], owner_package=kind_package,
                          max_object_depth=max_object_depth)
    for package_name, package in packages.items():
        for identifier, prop in package["definitions"].items():
            if kind_identity(packages, package_name, prop) != ("markitect/core", "Property"):
                continue
            spec = prop.get("spec", {})
            value_type = spec.get("type")
            minimum, maximum = spec.get("minCount"), spec.get("maxCount")
            if value_type not in SCALAR_TYPES:
                fail(f"{identifier}: Property type must be a known core value type")
            if not isinstance(minimum, int) or isinstance(minimum, bool) or minimum < 0:
                fail(f"{identifier}: Property minCount must be a nonnegative integer")
            if not isinstance(maximum, int) or isinstance(maximum, bool) or (maximum != -1 and maximum < minimum):
                fail(f"{identifier}: Property maxCount must be -1 or at least minCount")
            if value_type == "enum":
                values = spec.get("values")
                if not isinstance(values, list) or not values or any(not isinstance(v, str) for v in values) or len(values) != len(set(values)):
                    fail(f"{identifier}: enum values must be nonempty unique strings")
            elif "values" in spec:
                fail(f"{identifier}: values is only valid for enum")
            if value_type == "reference":
                if "target" not in spec:
                    fail(f"{identifier}: reference requires target")
                target_package, target_id = resolve(packages, package_name, prop, spec["target"])
                target_def = packages[target_package]["definitions"][target_id]
                metakind_package, metakind_id = resolve(packages, target_package, target_def, target_def.get("kind"))
                if (metakind_package, metakind_id) != ("markitect/core", "Kind"):
                    fail(f"{identifier}: reference target must be a Kind definition")
            elif "target" in spec:
                fail(f"{identifier}: target is only valid for reference")
            if value_type == "object":
                nested = spec.get("properties", [])
                if not isinstance(nested, list):
                    fail(f"{identifier}: object properties must be a list when present")
                names: dict[str, str] = {}
                for reference in nested:
                    nested_package, nested_id = resolve(packages, package_name, prop, reference)
                    nested_definition = packages[nested_package]["definitions"][nested_id]
                    if kind_identity(packages, nested_package, nested_definition) != ("markitect/core", "Property"):
                        fail(f"{identifier}: object schema target {reference!r} is not a Property")
                    effective_name = nested_definition.get("spec", {}).get("name", nested_id)
                    check_segment_style(effective_name, PROPERTY_STYLE, "effective Property field name")
                    previous = names.get(name_key(effective_name))
                    if previous:
                        fail(f"{identifier}: case-insensitive effective field collision {previous!r} vs {effective_name!r}")
                    names[name_key(effective_name)] = effective_name
            elif "properties" in spec:
                fail(f"{identifier}: properties is only valid for object")
            if "name" in spec:
                check_segment_style(spec["name"], PROPERTY_STYLE, "effective Property field name")
    reject_required_object_schema_cycles(packages)
    body_links = len(check_body_links(packages))
    return {"definitions": sum(len(p["definitions"]) for p in packages.values()), "resolution_calls": RESOLUTION_CALLS,
            "body_links": body_links}


@resource_checked("body-link validation")
def check_body_links(packages: dict[str, dict[str, Any]], *, coordinates: bool = False) -> list[dict[str, Any]]:
    occurrences = []
    for package_name, package in sorted(packages.items()):
        for identifier, definition in sorted(package["definitions"].items()):
            body = definition.get("body")
            if not isinstance(body, str):
                fail(f"{identifier}: body must be text")
            try:
                links = parse_body_links(body, coordinates=coordinates)
                if coordinates:
                    links = with_yaml_coordinates(links, package.get("body_sources", {}).get(identifier), body)
                for link in links:
                    try:
                        target_package, target_id = resolve(packages, package_name, definition, link["target"])
                    except ValueError as error:
                        fail(str(error))
                    occurrences.append({**link, "definition": {"package": package_name, "id": identifier},
                                        "writtenTarget": link["target"], "target": {"package": target_package, "id": target_id}})
            except BodyLinkLimitError as error:
                raise ProbeLimitError(str(error)) from error
            except BodyLinkSyntaxError as error:
                location = None
                if coordinates and error.byte_start is not None:
                    located = with_yaml_coordinates(
                        [{"startByte": error.byte_start, "endByte": error.byte_start + 1}],
                        package.get("body_sources", {}).get(identifier), body)
                    location = located[0].get("source")
                suffix = f"; source {location}" if location else ""
                fail(f"{package_name}.{identifier}: {error}{suffix}")
            except ValueError as error:
                file = package.get("paths", {}).get(identifier, "<probe>")
                fail(f"{file}: {package_name}.{identifier}: {error}")
    return occurrences


def diagnostic_coordinate_report(packages) -> dict[str, Any]:
    """An optional tool-profile result, excluded from model validity and hashing."""
    if packages is None:
        return {"status": "not_completed", "diagnostics": ["Models were not loaded."], "occurrences": []}
    try:
        records = check_body_links(packages, coordinates=True)
    except (ValueError, RecursionError, MemoryError) as error:
        return {"status": "not_completed", "diagnostics": [str(error) or type(error).__name__], "occurrences": []}
    issues = []
    for index, record in enumerate(records):
        if record.get("coordinate_issue") or not {"startByte", "endByte", "source"} <= record.keys():
            source = record["definition"]
            issues.append(f"{source['package']}/{source['id']} occurrence {index}: "
                          + (record.get("coordinate_issue") or "exact coordinates are unavailable"))
    return {"status": "not_completed" if issues else "passed", "diagnostics": issues, "occurrences": records}


def reject_required_object_schema_cycles(packages: dict[str, dict[str, Any]]) -> None:
    """Only required object-containment cycles prevent finite values."""
    objects: set[tuple[str, str]] = set()
    edges: dict[tuple[str, str], set[tuple[str, str]]] = {}
    for package_name, package in packages.items():
        for identifier, prop in package["definitions"].items():
            is_property = kind_identity(packages, package_name, prop) == ("markitect/core", "Property") if "kind" in prop else False
            spec = prop.get("spec", {}) if is_property else {}
            if spec.get("type") != "object":
                continue
            node = (package_name, identifier)
            objects.add(node)
            edges[node] = set()
            for reference in spec.get("properties", []):
                target = resolve(packages, package_name, prop, reference)
                target_prop = packages[target[0]]["definitions"][target[1]]
                if (target_prop.get("spec", {}).get("type") == "object"
                        and target_prop["spec"]["minCount"] > 0):
                    edges[node].add(target)
    visiting: set[tuple[str, str]] = set()
    visited: set[tuple[str, str]] = set()

    for object_node in sorted(objects):
        if object_node in visited:
            continue
        visiting.add(object_node)
        stack = [(object_node, iter(sorted(edges[object_node])))]
        while stack:
            node, children = stack[-1]
            child = next(children, None)
            if child is None:
                visiting.remove(node)
                visited.add(node)
                stack.pop()
            elif child in visiting:
                path = [frame[0] for frame in stack]
                cycle = path[path.index(child):] + [child]
                fail("mandatory object schema cycle: " + " -> ".join(f"{package}/{identifier}" for package, identifier in cycle))
            elif child not in visited:
                visiting.add(child)
                stack.append((child, iter(sorted(edges[child]))))


def check_segment_style(value: str, style: str, category: str) -> None:
    require_nfc(value, category)
    if "." in value:
        fail(f"{category} {value!r} must not contain dots")
    if not name_matches(value, style):
        fail(f"{category} {value!r} violates the Unicode naming convention")


def check_identity_collisions(packages: dict[str, dict[str, Any]], *, definitions: bool = True) -> None:
    seen_packages: dict[str, str] = {}
    # Discovery checks every physical folder locally. Global binding ignores
    # purely empty subtrees, including when validating a selected model scope.
    names = package_binding_names(packages) if definitions else set(packages)
    for package_name in sorted(names):
        previous = seen_packages.get(name_key(package_name))
        if previous:
            fail(f"case-insensitive package ID collision {previous!r} vs {package_name!r}")
        seen_packages[name_key(package_name)] = package_name
    if not definitions:
        return
    for package_name, package in packages.items():
        seen_ids: dict[str, str] = {}
        for identifier in package["definitions"]:
            previous = seen_ids.get(name_key(identifier))
            if previous:
                fail(f"case-insensitive definition ID collision {previous!r} vs {identifier!r} in {package_name}")
            seen_ids[name_key(identifier)] = identifier


@resource_checked("historical list comparison")
def list_comparison_view(packages: dict[str, dict[str, Any]]) -> tuple:
    """Bounded comparison probe for the selected list rules, not canonical bytes.

    Validate first; this must never repair an invalid source or hide duplicates.
    Historical list-only comparison; model-digest/1 is the full projection. No writes.
    """
    validate(packages)
    core_kind = ("markitect/core", "Kind")
    core_property = ("markitect/core", "Property")

    def scalar(value):
        if isinstance(value, Decimal):
            return ("decimal", value.as_tuple())
        return (type(value).__name__, value)

    def fields(source_package, source, owner_package, owner, values, role=None):
        result = []
        for prop_ref in owner.get("spec", {}).get("properties", []):
            prop_package, prop_id = resolve(packages, owner_package, owner, prop_ref)
            prop = packages[prop_package]["definitions"][prop_id]
            contract = prop["spec"]
            field_name = contract.get("name", prop_id)
            is_list = contract["maxCount"] != 1
            is_core_field = prop_package == "markitect/core"
            # Conditional meta-fields must not appear on an inapplicable type.
            if role == core_property and is_core_field:
                if prop_id == "properties" and values["type"] != "object":
                    continue
                if prop_id == "values" and values["type"] != "enum":
                    continue
            if field_name not in values:
                if not (is_list and contract["minCount"] == 0):
                    continue
                value = []
            else:
                value = values[field_name]

            def item_view(item):
                if contract["type"] == "reference":
                    return ("reference", *resolve(packages, source_package, source, item))
                if contract["type"] == "object":
                    return ("object", fields(source_package, source, prop_package, prop, item))
                return scalar(item)

            if is_list:
                items = tuple(item_view(item) for item in value)
                declaration = is_core_field and (
                    (role == core_kind and prop_id == "properties") or
                    (role == core_property and prop_id in {"properties", "values"}))
                if declaration:
                    items = tuple(sorted(items))
                result.append((field_name, ("list", items)))
            else:
                result.append((field_name, item_view(value)))
        return tuple(sorted(result))

    model = []
    for package_name, package in sorted(packages.items()):
        definitions = []
        for identifier, definition in sorted(package["definitions"].items()):
            kind_package, kind_id = kind_identity(packages, package_name, definition)
            owner = packages[kind_package]["definitions"][kind_id]
            imports = tuple(sorted(item["package"] for item in package_imports(packages, package_name, definition)))
            title = ("present", definition["title"]) if "title" in definition else ("absent",)
            definitions.append((identifier, (kind_package, kind_id), definition["body"], title, imports,
                                fields(package_name, definition, kind_package, owner, definition["spec"],
                                       (kind_package, kind_id))))
        model.append((package_name, tuple(definitions)))
    bindings = tuple((identifier, record["manifest"]["formatVersion"], record["manifest"].get("version"),
                      tuple(sorted(record["manifest"]["dependencies"].items())))
                     for identifier, record in sorted(packages.models.items()))
    return (packages.entry_model, bindings, tuple(model))


@resource_checked("canonical projection")
def model_bytes(packages, model_id, *, max_object_depth=DEFAULT_OBJECT_DEPTH):
    return canonical_model_bytes(packages, model_id, validate=lambda scope: validate(scope, max_object_depth=max_object_depth),
                                 resolve=resolve, kind_identity=kind_identity, body_links=check_body_links)


def assess_canonical(packages, model_id, *, max_object_depth=DEFAULT_OBJECT_DEPTH):
    try:
        data = model_bytes(packages, model_id, max_object_depth=max_object_depth)
    except ProbeLimitError as error:
        return {"status": "not_completed", "diagnostics": [str(error)]}, None
    except ValueError as error:
        return {"status": "failed", "diagnostics": [str(error)]}, None
    return {"status": "passed", "digest": canonical_digest(data), "byte_count": len(data)}, data


def export_canonical(packages, model_id, output, *, max_object_depth=DEFAULT_OBJECT_DEPTH):
    record, data = assess_canonical(packages, model_id, max_object_depth=max_object_depth)
    if record["status"] == "passed":
        output.parent.mkdir(parents=True, exist_ok=True)
        output.write_bytes(data)
    return record


@resource_checked("self-test execution")
def test_suite(source: dict[str, dict[str, Any]], source_diagnostics: list[str]) -> dict[str, bool]:
    results: dict[str, bool] = {}
    baseline = make_test_only_valid_baseline(source)
    try:
        validate(baseline)
        results["test_only_baseline_valid"] = True
    except ValueError:
        results["test_only_baseline_valid"] = False

    def expect(mutator, fragment: str) -> bool:
        trial = copy.deepcopy(baseline)
        try:
            mutator(trial)
            validate(trial)
        except ValueError as error:
            return fragment in str(error)
        except (TypeError, KeyError):
            return False
        return False

    project, vocab, core = "example/connector/project", "example/connector/vocabulary", "markitect/core"
    list_docs = lambda t: t[project]["definitions"]["list-documents"]
    results["alias_qualified_name_resolves"] = resolve(baseline, project, list_docs(baseline), "Connector.Exchange") == (vocab, "Exchange")
    results["own_package_qualification_without_import_resolves"] = resolve(
        baseline, project, list_docs(baseline), f"{project}/list-documents") == (project, "list-documents")

    results["full_package_qualified_name_resolves"] = resolve(baseline, project, list_docs(baseline), f"{vocab}/Exchange") == (vocab, "Exchange")
    results["package_qualifier_uses_slash_not_dot"] = expect(lambda t: resolve(t, project, list_docs(t), f"{vocab}.Exchange"), "unresolved")
    results["old_dotted_package_reference_is_not_corrected"] = expect(lambda t: resolve(t, project, list_docs(t), "example.connector.vocabulary.Exchange"), "unresolved")
    results["old_dotted_package_import_is_not_corrected"] = expect(
        lambda t: list_docs(t)["imports"][0].update(package="example.connector.vocabulary"), "is absent")
    results["alias_qualifier_stays_dot_not_slash"] = expect(lambda t: resolve(t, project, list_docs(t), "Connector/Exchange"), "unresolved")
    results["full_package_reference_requires_file_import"] = expect(
        lambda t: resolve(t, project, list_docs(t), "markitect/core/Kind"), "unresolved")
    results["wrong_case_full_package_reference"] = expect(
        lambda t: resolve(t, project, list_docs(t), "Example/connector/vocabulary/Exchange"), "wrong-case reference")
    results["full_package_reference_has_no_dotted_local_id"] = expect(
        lambda t: resolve(t, project, list_docs(t), f"{vocab}/exchange.sender"), "unresolved")
    results["alias_reference_has_no_field_dereference"] = expect(
        lambda t: resolve(t, project, list_docs(t), "Connector.requester.name"), "unresolved")
    for malformed in ("/example/connector/vocabulary/Exchange", "example//connector/vocabulary/Exchange",
                      "./example/connector/vocabulary/Exchange", "example/connector/../vocabulary/Exchange",
                      "example\\connector\\vocabulary\\Exchange", "example/connector/vocabulary/Exchange/"):
        results[f"full_package_reference_no_path_normalization_{malformed}"] = expect(
            lambda t, name=malformed: resolve(t, project, list_docs(t), name), "unresolved")

    def alias_collision(t):
        list_docs(t)["imports"].append({"package": core, "as": "connector"})
        package_imports(t, project, list_docs(t))
    results["case_insensitive_alias_collision"] = expect(alias_collision, "case-insensitive alias collision")
    results["explicit_null_alias_rejected"] = expect(
        lambda t: (list_docs(t)["imports"].__setitem__(0, {"package": vocab, "as": None}), package_imports(t, project, list_docs(t))),
        "omit as for no alias")
    results["wrong_case_alias_reference"] = expect(lambda t: resolve(t, project, list_docs(t), "connector.Exchange"), "wrong-case reference")
    results["wrong_case_alias_declaration"] = expect(
        lambda t: (list_docs(t)["imports"].__setitem__(0, {"package": vocab, "as": "connector"}), package_imports(t, project, list_docs(t))),
        "alias 'connector' must be PascalCase")
    results["wrong_case_import_package"] = expect(
        lambda t: (list_docs(t)["imports"].__setitem__(0, {"package": "Example/connector/vocabulary", "as": "Connector"}), package_imports(t, project, list_docs(t))),
        "wrong-case package import")
    results["wrong_case_reference"] = expect(lambda t: list_docs(t).update(kind="exchange"), "wrong-case reference")
    results["wrong_case_field"] = expect(lambda t: list_docs(t)["spec"].update(Requester="api"), "wrong-case field")
    results["synthetic_case_only_duplicate_id"] = expect(
        lambda t: t[vocab]["definitions"].update({"exchange": {"id": "exchange"}}), "case-insensitive definition ID collision")
    results["effective_field_case_collision"] = expect(lambda t: add_duplicate_case(t), "case-insensitive effective field collision")
    results["filename_case_style"] = filename_style_probe()
    results["case_only_filenames_rejected"] = expect_filename_collision()
    results["case_insensitive_package_collision"] = package_collision_probe(baseline)
    results["identifier_word_case_style"] = identifier_style_probe()
    results["duplicate_yaml_case_key"] = duplicate_key_probe()
    results["missing_file_local_import"] = expect(
        lambda t: t[vocab]["definitions"]["Exchange"].pop("imports", None), "unresolved")
    results["unknown_reference"] = expect(lambda t: list_docs(t)["spec"].update(requester="absent"), "unresolved")
    results["wrong_kind"] = expect(lambda t: list_docs(t).update(kind="requester"), "not itself a Kind definition")
    results["wrong_target"] = expect(lambda t: list_docs(t)["spec"].update(requester="requests"), "has wrong kind")
    results["missing_required"] = expect(lambda t: t[project]["definitions"]["api"].update(spec={}), "required field")
    results["unknown_spec"] = expect(lambda t: t[project]["definitions"]["api"]["spec"].update(unknown="x"), "unknown field")
    results["invalid_enum"] = expect(lambda t: t[project]["definitions"]["api"]["spec"].update(location="moon"), "value outside enum")
    results["invalid_cardinality"] = expect(lambda t: t[project]["definitions"]["api"].update(spec={"location": ["cloud"]}), "expected scalar")

    populated = copy.deepcopy(baseline)
    add_object_case(populated, {"messageId": "x"})
    validate(populated)
    results["populated_object_schema_positive"] = True
    empty = copy.deepcopy(baseline)
    add_empty_object_case(empty)
    validate(empty)
    results["empty_object_schema_positive"] = True
    results["object_unknown_field"] = expect(lambda t: add_object_case(t, {"unknown": "x"}), "unknown field")
    results["properties_forbidden_on_string"] = expect(lambda t: add_string_object_case(t), "properties is only valid for object")
    results["mandatory_recursive_object_schema"] = expect(lambda t: add_cycle_case(t), "mandatory object schema cycle")
    results["duplicate_yaml_key"] = duplicate_key_probe()
    results.update(manifest_probes(load_yaml, discover_packages))
    results.update(root_path_probes(load_models, load_yaml, discover_packages))
    results.update(object_resource_probes(baseline, validate, model_bytes, assess_canonical, export_canonical, parse_yaml,
                                         reject_required_object_schema_cycles))
    results.update(lock_probes(baseline, load_yaml, ROOT, model_bytes))
    results.update(canonical_probes(baseline, source, model_bytes, parse_yaml, ROOT))
    results.update(core_equivalence_probes(baseline, model_bytes))
    results.update(contract_example_probes(baseline))
    results.update(package_review_probes(baseline, resolve, validate, package_imports, discover_packages))
    results.update(empty_package_binding_probes(ROOT, load_models, load_yaml, discover_packages, validate, model_bytes))
    results.update(folder_probes(baseline))
    results.update(unicode_probes(baseline))
    results.update(yaml_value_probes(baseline))
    results.update(list_probes(baseline, source))
    results.update(body_link_probes(baseline))
    results.update(diagnostic_probes(baseline, parse_yaml, check_body_links, diagnostic_coordinate_report, model_bytes, assess_canonical))
    return results


def body_link_probes(baseline: dict[str, dict[str, Any]]) -> dict[str, bool]:
    results: dict[str, bool] = {}
    project = "example/connector/project"
    for name, body, targets in (
        ("plain", "[[api]]", ["api"]),
        ("display", "[[api|Cloud-API]]", ["api"]),
        ("formatting_outside", "**[[api|Cloud-API]]**", ["api"]),
        ("containers", "# [[api]]\n\n> [[onprem]]\n\n- [[api]]", ["api", "onprem", "api"]),
        ("nested_containers", "> - 📄 [[api]]\n>   [[onprem]]", ["api", "onprem"]),
        ("code_span", "`[[broken` and [[api]]", ["api"]),
        ("long_code_span", "`` [[broken ` `` [[api]]", ["api"]),
        ("fence", "```text\n[[broken\n```\n\n[[api]]", ["api"]),
        ("tilde_fence", "~~~\n[[broken\n~~~\n\n[[api]]", ["api"]),
        ("indented_code", "    [[broken\n\n[[api]]", ["api"]),
        ("escaped", "\\[[broken", []),
        ("even_backslashes", "\\\\[[api]]", ["api"]),
        ("odd_backslashes", "\\\\\\[[broken", []),
        ("ordinary_link", "[Label [[absent]]](https://example.org) [[api]]", ["api"]),
        ("image", "![[absent]](x) [[api]]", ["api"]),
        ("reference_link", "[Label [[absent]]][dest]\n\n[dest]: https://example.org", []),
        ("reference_definition", "[dest]: https://example.org/[[absent]]\n\n[[api]]", ["api"]),
        ("autolink", "<https://example.org/[[absent]]> [[api]]", ["api"]),
        ("html_block", "<div>[[broken</div>\n\n[[api]]", ["api"]),
        ("html_comment", "Text <!-- [[broken --> [[api]]", ["api"]),
        ("html_attribute", 'Text <span title="[[broken">x</span> [[api]]', ["api"]),
        ("unmatched_html_close", "Text </a> [[api]]", ["api"]),
        ("entities_not_markers", "&#91;&#91;absent]] [[api]]", ["api"]),
        ("repeated", "[[api]] [[api]]", ["api", "api"]),
        ("model_link_precedence", "[[api]](ordinary text)", ["api"]),
    ):
        records = parse_body_links(body, coordinates=True)
        results[f"body_link_{name}"] = [record["target"] for record in records] == targets
        results[f"body_link_{name}_byte_spans"] = all(
            body.encode("utf-8")[record["startByte"]:record["endByte"]].decode("utf-8").startswith("[[") and
            body.encode("utf-8")[record["startByte"]:record["endByte"]].decode("utf-8").endswith("]]") for record in records)
    results["body_link_display_is_plain_text"] = parse_body_links("[[api|**Cloud** &amp;]]")[0]["display"] == "**Cloud** &amp;"
    results["body_link_display_escapes"] = parse_body_links(r"[[api|A \| B \[C\] \\]]")[0]["display"] == "A | B [C] \\"
    for name, body, expected in (
        ("plain", "[[api|A\0B]]", ["A\0B"]),
        ("only_nul", "[[api|\0]]", ["\0"]),
        ("replacement_distinct", "[[api|A\ufffdB]] [[api|A\0B]]", ["A\ufffdB", "A\0B"]),
        ("mixed_and_repeated", "[[api|\0\ufffd\0]] [[api|\ufffd\0\ufffd]]", ["\0\ufffd\0", "\ufffd\0\ufffd"]),
        ("blockquote_and_heading", "> [[api|\0]]\r\n\r\n# [[api|📄\0]]", ["\0", "📄\0"]),
        ("list_and_escapes", "- [[api|A\0 \\| B \\[C\\]]]\n", ["A\0 | B [C]"]),
        ("protected_code", "`[[broken\0` [[api|\0]]", ["\0"]),
        ("protected_autolink", "<https://example.org/\0[[broken]]> [[api|\0]]", ["\0"]),
        ("protected_html_block", "<widget attr=\0>\n[[broken]]\n</widget>\n\n[[api|\0]]", ["\0"]),
        ("protected_reference", "[x[[broken]]][ref\0]\n\n[ref\ufffd]: https://example.org\n\n[[api|\0]]", ["\0"]),
    ):
        semantic = parse_body_links(body)
        located = parse_body_links(body, coordinates=True)
        results[f"body_link_literal_nul_{name}"] = [record["display"] for record in semantic] == expected
        results[f"body_link_literal_nul_{name}_coordinates_agree"] = (
            [{key: record[key] for key in ("target", "display")} for record in located] == semantic
            and all(body.encode("utf-8")[record["startByte"]:record["endByte"]].decode("utf-8").startswith("[[") for record in located))
    for escape in (r"\0", r"\x00", r"\u0000", r"\U00000000"):
        text = f'body: "[[api|A{escape}B]]"'
        source = {}
        body = parse_yaml(text, body_source=source, file="nul-display.yaml")["body"]
        results[f"body_link_yaml_nul_escape_{escape}_preserved"] = parse_body_links(body)[0]["display"] == "A\0B"
        located = with_yaml_coordinates(parse_body_links(body, coordinates=True), source, body)[0]
        results[f"body_link_yaml_nul_escape_{escape}_source_span"] = located["source"] == {
            "file": "nul-display.yaml", "start": {"line": 1, "column": text.index("[[") + 1},
            "end": {"line": 1, "column": text.index("]]") + 3}}
    body = "📄 [[api]]\r\n> [[onprem]]"
    records = parse_body_links(body, coordinates=True)
    results["body_link_utf8_and_crlf_offsets"] = records[0]["startByte"] == 5 and records[1]["startByte"] == len("📄 [[api]]\r\n> ".encode("utf-8"))
    for name, body, fragment in (
        ("empty_target", "[[]]", "nonempty target"),
        ("target_spaces", "[[ api ]]", "surrounding whitespace"),
        ("empty_display", "[[api|]]", "display text"),
        ("blank_display", "[[api| \t]]", "display text"),
        ("nul_is_not_a_model_link_escape", r"[[api|\0]]", "invalid body-link escape"),
        ("unclosed", "[[api", "unclosed"),
        ("separator", "[[api|A|B]]", "separator"),
        ("nested", "[[api|[[onprem]]]]", "nested"),
        ("newline", "[[api|A\nB]]", "line breaks"),
        ("unicode_line_separator", "[[api|A\u2028B]]", "line breaks"),
        ("unsupported_unicode_next_line", "[[api|A\x85B]]", "capability limit"),
        ("bad_escape", r"[[api|A\q]]", "escape"),
        ("target_escape", r"[[a\pi]]", "escape"),
    ):
        try:
            parse_body_links(body)
        except ValueError as error:
            results[f"body_link_{name}_rejected"] = fragment in str(error)
        else:
            results[f"body_link_{name}_rejected"] = False

    def with_body(body):
        trial = copy.deepcopy(baseline)
        trial[project]["definitions"]["list-documents"]["body"] = body
        return trial

    trial = with_body("[[Exchange]] [[requester]] [[api]] [[Connector.Channel]] [[example/connector/vocabulary/Command]]")
    validate(trial)
    records = check_body_links(trial)
    results["body_link_any_definition_kind_resolves"] = [record["target"]["id"] for record in records] == ["Exchange", "requester", "api", "Channel", "Command"]
    for name, body, fragment in (
        ("missing_target", "[[absent]]", "unresolved"),
        ("wrong_case", "[[API]]", "wrong-case reference"),
        ("non_nfc", "[[u\u0308berweisung]]", "must already be NFC"),
        ("no_transitive_import", "[[Kind]]", "unresolved"),
        ("full_package_no_transitive_import", "[[markitect/core/Kind]]", "unresolved"),
        ("old_dotted_package", "[[example.connector.vocabulary/Exchange]]", "unresolved"),
        ("entity_not_target_escape", "[[a&#112;i]]", "unresolved"),
    ):
        try:
            validate(with_body(body))
        except ValueError as error:
            results[f"body_link_{name}_rejected"] = fragment in str(error)
        else:
            results[f"body_link_{name}_rejected"] = False
    trial = with_body("No active links here.")
    trial[project]["definitions"]["list-documents"]["title"] = "[[absent]]"
    trial[project]["definitions"]["list-documents-command"]["spec"]["operationId"] = "[[absent]]"
    validate(trial)
    results["body_link_only_body_is_interpreted"] = not any(record["definition"]["id"] == "list-documents" for record in check_body_links(trial))
    for name, text in (
        ("plain", "body: See [[api]]"),
        ("single_quoted", "body: '[[api|API''s]]'"),
        ("double_escaped", r'body: "\u005B\u005Bapi\u005D\u005D"'),
        ("literal", "body: |-\n  📄 [[api]]\n"),
        ("folded", "body: >-\n  📄 [[api|Cloud\n  API]]\n"),
        ("container", "body: |-\n  > - 📄 [[api]]\n  >\n  >   [[onprem]]\n"),
        ("crlf", "body: |-\r\n  [[api]]\r\n"),
    ):
        origin = {}
        decoded = parse_yaml(text, body_source=origin, file="probe.yaml")["body"]
        locations = yaml_body_source(text, origin["node"], decoded, origin["file"])
        located = [attach_yaml_source(record, locations) for record in parse_body_links(decoded, coordinates=True)]
        results[f"body_link_yaml_{name}_source_mapping"] = bool(located) and all(record["source"]["file"] == "probe.yaml" for record in located)
        if name == "folded":
            results["body_link_yaml_folded_exact_lines_and_columns"] = located[0]["source"]["start"] == {"line": 2, "column": 5} and located[0]["source"]["end"] == {"line": 3, "column": 8}
        if name == "double_escaped":
            results["body_link_yaml_escapes_can_create_markers_before_markdown"] = located[0]["startByte"] == 0 and located[0]["endByte"] == 7
        else:
            cursor = 0
            exact = True
            for record in located:
                first = text.index("[[", cursor)
                last = text.index("]]", first) + 2
                def pos(index):
                    prefix = text[:index].replace("\r\n", "\n").replace("\r", "\n")
                    return {"line": prefix.count("\n") + 1, "column": len(prefix.rsplit("\n", 1)[-1]) + 1}
                exact = exact and record["source"]["start"] == pos(first) and record["source"]["end"] == pos(last)
                cursor = last
            results[f"body_link_yaml_{name}_exact_source_spans"] = exact
    broken_source = "body: |-\n  📄 [[api\n"
    origin = {}
    decoded = parse_yaml(broken_source, body_source=origin, file="broken.yaml")["body"]
    try:
        parse_body_links(decoded, coordinates=True)
    except BodyLinkSyntaxError as error:
        locations = yaml_body_source(broken_source, origin["node"], decoded, origin["file"])
        located = attach_yaml_source({"startByte": error.byte_start, "endByte": error.byte_start + 1}, locations)
        results["body_link_syntax_error_exact_source_position"] = located["source"]["start"] == {"line": 2, "column": 5}
    else:
        results["body_link_syntax_error_exact_source_position"] = False
    try:
        parse_body_links("> " * 80 + "[[api]]")
    except BodyLinkLimitError:
        results["body_link_nesting_limit_is_explicit"] = True
    else:
        results["body_link_nesting_limit_is_explicit"] = False
    try:
        parse_body_links("Text\x85 [[api]]")
    except BodyLinkLimitError:
        results["body_link_known_parser_whitespace_limit_is_explicit"] = True
    else:
        results["body_link_known_parser_whitespace_limit_is_explicit"] = False
    return results


def list_probes(baseline: dict[str, dict[str, Any]], source: dict[str, dict[str, Any]]) -> dict[str, bool]:
    results: dict[str, bool] = {}
    project, vocab, core = "example/connector/project", "example/connector/vocabulary", "markitect/core"
    trial = copy.deepcopy(baseline)
    defs = trial[project]["definitions"]
    defs["ListProbe"] = {"id": "ListProbe", "kind": "Kind", "imports": [core_import()], "body": "probe",
                          "spec": {"properties": ["items"]}}
    defs["items"] = {"id": "items", "kind": "Property", "imports": [core_import()], "body": "probe",
                      "spec": {"type": "string", "minCount": 0, "maxCount": -1}}
    defs["list-probe"] = {"id": "list-probe", "kind": "ListProbe", "body": "probe", "spec": {"items": ["a", "b"]}}

    def changed(model, mutator):
        copy_model = copy.deepcopy(model)
        mutator(copy_model)
        return copy_model

    def instance(t):
        return t[project]["definitions"]["list-probe"]["spec"]

    def rejects(model, fragment):
        try:
            list_comparison_view(model)
        except ValueError as error:
            return fragment in str(error)
        return False

    before = copy.deepcopy(trial)
    base_view = list_comparison_view(trial)
    results["list_comparison_does_not_mutate_sources"] = trial == before
    results["list_value_order_is_significant"] = base_view != list_comparison_view(changed(trial, lambda t: instance(t)["items"].reverse()))
    repeated = changed(trial, lambda t: instance(t).update(items=["a", "a"]))
    results["list_ordinary_duplicates_are_retained"] = list_comparison_view(repeated) != list_comparison_view(changed(trial, lambda t: instance(t).update(items=["a"])))
    absent = changed(trial, lambda t: instance(t).pop("items"))
    empty = changed(trial, lambda t: instance(t).update(items=[]))
    results["list_optional_absent_equals_empty"] = list_comparison_view(absent) == list_comparison_view(empty)
    results["list_nonempty_differs_from_empty"] = base_view != list_comparison_view(empty)
    results["list_unknown_empty_field_is_not_discarded"] = rejects(
        changed(absent, lambda t: instance(t).update(unknown=[])), "unknown field")
    results["list_body_changes_are_preserved"] = list_comparison_view(absent) != list_comparison_view(
        changed(absent, lambda t: t[project]["definitions"]["list-probe"].update(body="changed statement")))
    results["list_missing_imports_equals_empty_imports"] = base_view == list_comparison_view(
        changed(trial, lambda t: t[project]["definitions"]["list-probe"].update(imports=[])))
    for label, model in (("absent", absent), ("empty", empty)):
        required = changed(model, lambda t: t[project]["definitions"]["items"]["spec"].update(minCount=1))
        results[f"list_required_{label}_rejected"] = rejects(required, "required field" if label == "absent" else "cardinality")
    zero_absent = changed(absent, lambda t: t[project]["definitions"]["items"]["spec"].update(maxCount=0))
    zero_empty = changed(empty, lambda t: t[project]["definitions"]["items"]["spec"].update(maxCount=0))
    results["list_zero_maximum_absent_equals_empty"] = list_comparison_view(zero_absent) == list_comparison_view(zero_empty)
    singleton_absent = changed(absent, lambda t: t[project]["definitions"]["items"]["spec"].update(maxCount=1))
    singleton_empty = changed(singleton_absent, lambda t: instance(t).update(items=""))
    results["list_equivalence_does_not_merge_missing_and_empty_string"] = list_comparison_view(singleton_absent) != list_comparison_view(singleton_empty)
    results["list_equivalence_does_not_admit_array_for_singleton"] = rejects(
        changed(singleton_absent, lambda t: instance(t).update(items=[])), "expected scalar")

    nested = copy.deepcopy(trial)
    nested[project]["definitions"]["details"] = {"id": "details", "kind": "Property", "imports": [core_import()], "body": "probe",
                                                "spec": {"type": "object", "minCount": 0, "maxCount": 1, "properties": ["items"]}}
    nested[project]["definitions"]["ListProbe"]["spec"]["properties"] = ["details"]
    instance(nested).clear()
    instance(nested)["details"] = {}
    nested_empty = changed(nested, lambda t: instance(t)["details"].update(items=[]))
    results["list_nested_optional_absent_equals_empty"] = list_comparison_view(nested) == list_comparison_view(nested_empty)
    results["list_equivalence_does_not_merge_missing_and_empty_object"] = list_comparison_view(nested) != list_comparison_view(
        changed(nested, lambda t: instance(t).pop("details")))

    reordered = copy.deepcopy(trial)
    reordered[vocab]["definitions"]["Exchange"]["spec"]["properties"].reverse()
    reordered[core]["definitions"]["Property"]["spec"]["properties"].reverse()
    reordered[core]["definitions"]["type"]["spec"]["values"].reverse()
    results["list_kind_and_fixed_core_declaration_order_ignored"] = base_view == list_comparison_view(reordered)
    enum_reordered = changed(trial, lambda t: t[vocab]["definitions"]["location"]["spec"]["values"].reverse())
    results["list_enum_declaration_order_ignored"] = base_view == list_comparison_view(enum_reordered)
    enum_added = changed(trial, lambda t: t[vocab]["definitions"]["location"]["spec"]["values"].append("elsewhere"))
    results["list_enum_membership_remains_significant"] = base_view != list_comparison_view(enum_added)
    enum_duplicate = changed(trial, lambda t: t[vocab]["definitions"]["location"]["spec"]["values"].append("cloud"))
    results["list_enum_duplicates_still_rejected"] = rejects(enum_duplicate, "unique")
    duplicate_field = changed(trial, lambda t: t[project]["definitions"]["ListProbe"]["spec"]["properties"].append("items"))
    results["list_duplicate_declarations_still_rejected"] = rejects(duplicate_field, "effective field collision")
    nested_more = copy.deepcopy(nested)
    nested_more[project]["definitions"]["labels"] = {"id": "labels", "kind": "Property", "imports": [core_import()], "body": "probe",
                                                    "spec": {"type": "string", "minCount": 0, "maxCount": -1}}
    nested_more[project]["definitions"]["details"]["spec"]["properties"].append("labels")
    nested_reverse = changed(nested_more, lambda t: t[project]["definitions"]["details"]["spec"]["properties"].reverse())
    results["list_object_declaration_order_ignored"] = list_comparison_view(nested_more) == list_comparison_view(nested_reverse)
    empty_schema = copy.deepcopy(trial)
    empty_schema[project]["definitions"]["EmptyProbe"] = {"id": "EmptyProbe", "kind": "Kind", "imports": [core_import()], "body": "probe", "spec": {}}
    results["list_kind_missing_properties_equals_empty_properties"] = list_comparison_view(empty_schema) == list_comparison_view(
        changed(empty_schema, lambda t: t[project]["definitions"]["EmptyProbe"]["spec"].update(properties=[])))
    empty_object_schema = copy.deepcopy(trial)
    add_empty_object_case(empty_object_schema)
    results["list_object_missing_properties_equals_empty_properties"] = list_comparison_view(empty_object_schema) == list_comparison_view(
        changed(empty_object_schema, lambda t: t[project]["definitions"]["payload"]["spec"].update(properties=[])))
    ordinary_values = changed(trial, lambda t: t[project]["definitions"]["items"]["spec"].update(name="values"))
    instance(ordinary_values)["values"] = instance(ordinary_values).pop("items")
    results["list_ordinary_field_named_values_keeps_order"] = list_comparison_view(ordinary_values) != list_comparison_view(
        changed(ordinary_values, lambda t: instance(t)["values"].reverse()))

    catalog = copy.deepcopy(nested_more)
    catalog[project]["definitions"]["ListProbe"]["spec"]["properties"] = ["Core.properties"]
    instance(catalog).clear()
    instance(catalog)["properties"] = ["items", "labels"]
    results["list_shared_properties_on_ordinary_kind_keeps_order"] = list_comparison_view(catalog) != list_comparison_view(
        changed(catalog, lambda t: instance(t)["properties"].reverse()))
    qualified = changed(catalog, lambda t: t[project]["definitions"]["ListProbe"]["spec"].update(properties=["markitect/core/properties"]))
    results["list_declaration_targets_use_canonical_identity"] = list_comparison_view(catalog) == list_comparison_view(qualified)

    imported = copy.deepcopy(trial)
    imported[project]["definitions"]["list-documents"]["imports"].append(core_import())
    import_reverse = changed(imported, lambda t: t[project]["definitions"]["list-documents"]["imports"].reverse())
    results["list_import_order_ignored"] = list_comparison_view(imported) == list_comparison_view(import_reverse)
    without_imports = changed(trial, lambda t: t[project]["definitions"]["list-documents"]["imports"].pop())
    results["list_import_membership_still_validated"] = rejects(without_imports, "unresolved")
    # The source's pending vocabulary collision must never be hidden by normalization.
    if "Command" in source[vocab]["definitions"] and "command" in source[vocab]["definitions"]:
        results["list_comparison_rejects_invalid_source_first"] = rejects(source, "definition ID collision")
    return results


def yaml_value_probes(baseline: dict[str, dict[str, Any]]) -> dict[str, bool]:
    results: dict[str, bool] = {}

    def rejects(text: str, fragment: str) -> bool:
        try:
            parse_yaml(text)
        except ValueError as error:
            return fragment in str(error)
        return False

    values = parse_yaml('enabled: true\ndisabled: false\nlabel: "true"\nonValue: on\nyesValue: yes\n'
                        'dateValue: 2026-10-15\nupperBool: TRUE\nhexValue: 0x10\noctalValue: 012\n')
    results["yaml_only_lowercase_boolean_tokens"] = values["enabled"] is True and values["disabled"] is False and values["upperBool"] == "TRUE"
    results["yaml_quoted_boolean_stays_string"] = values["label"] == "true"
    results["yaml_on_and_yes_stay_strings"] = values["onValue"] == "on" and values["yesValue"] == "yes"
    results["yaml_date_stays_string"] = values["dateValue"] == "2026-10-15"
    results["yaml_nondecimal_tokens_are_text"] = values["hexValue"] == "0x10" and values["octalValue"] == "012"
    numbers = parse_yaml("integerValue: -42\ndecimalValue: 0.10000000000000000000000000000000000001\nexponentValue: 1e300\n")
    results["yaml_integer_is_exact"] = type(numbers["integerValue"]) is int and numbers["integerValue"] == -42
    results["yaml_decimal_preserves_all_digits"] = numbers["decimalValue"].as_tuple() == Decimal("0.10000000000000000000000000000000000001").as_tuple()
    results["yaml_exponent_is_exact_decimal"] = isinstance(numbers["exponentValue"], Decimal) and numbers["exponentValue"] == Decimal("1e300")
    results["yaml_large_integer_exceeds_machine_and_python_digit_limits"] = parse_yaml("value: 1" + "0" * 5000)["value"] == 10 ** 5000
    results["yaml_decimal_exponent_leading_zeros"] = parse_yaml("value: 1e00000000000000000001")["value"] == Decimal("10")
    results["yaml_numeric_probe_limit_is_explicit"] = rejects("value: " + "9" * (MAX_NUMBER_DIGITS + 1), "probe resource limit")
    results["yaml_exponent_probe_limit_is_explicit"] = rejects("value: 1e1000000001", "probe resource limit")
    results["yaml_layout_comments_and_mapping_order_are_irrelevant"] = parse_yaml("a: 1\nb: text\n") == parse_yaml("# comment\n{b: text, a: 1}\n")
    results["yaml_literal_and_folded_text_follow_yaml_rules"] = parse_yaml("literal: |-\n  A\n  B\nfolded: >-\n  A\n  B\n") == {"literal": "A\nB", "folded": "A B"}
    results["yaml_quoted_empty_string_allowed"] = parse_yaml("value: ''")["value"] == ""
    results["yaml_quoted_syntax_markers_are_text"] = parse_yaml("value: '!!str &anchor *alias <<'")["value"] == "!!str &anchor *alias <<"
    results["yaml_escaped_unicode_string_is_preserved"] = parse_yaml('value: "u\\u0308\\U0001F4C4"')["value"] == "u\u0308📄"
    results["yaml_unpaired_surrogate_rejected"] = rejects('value: "\\uD800"', "unpaired Unicode surrogate")
    results["yaml_single_document_marker_allowed"] = parse_yaml("%YAML 1.2\n---\na: 1\n...\n") == {"a": 1}
    for name, text, fragment in (
        ("multiple_documents", "a: 1\n---\nb: 2\n", "invalid YAML document"),
        ("sequence_root", "- a\n- b\n", "mapping root"),
        ("empty_document", "", "mapping root"),
        ("integer_key", "42: value", "keys must be strings"),
        ("boolean_key", "true: value", "keys must be strings"),
        ("object_key", "? [a, b]\n: value", "keys must be strings"),
        ("anchor", "value: &shared text", "anchors and aliases are forbidden"),
        ("alias", "value: *shared", "anchors and aliases are forbidden"),
        ("merge_key", "value: {<<: {a: 1}}", "merge keys are forbidden"),
        ("custom_tag", "value: !custom text", "explicit YAML tags are forbidden"),
        ("explicit_standard_tag", "value: !!str true", "explicit YAML tags are forbidden"),
        ("version_1_1", "%YAML 1.1\n---\na: 1", "only the YAML 1.2 version directive"),
        ("null", "value: null", "null and empty YAML values are forbidden"),
        ("empty_value", "value:", "null and empty YAML values are forbidden"),
        ("nonfinite_nan", "value: .nan", "non-finite numeric tokens"),
        ("nonfinite_infinity", "value: -.inf", "non-finite numeric tokens"),
    ):
        results[f"yaml_{name}_rejected"] = rejects(text, fragment)

    def accepts_property(value_type: str, scalar: str) -> bool:
        trial = copy.deepcopy(baseline)
        defs = trial["example/connector/project"]["definitions"]
        defs["ValueProbe"] = {"id": "ValueProbe", "kind": "Kind", "body": "probe", "imports": [core_import()],
                              "spec": {"properties": ["sampleValue"]}}
        defs["sampleValue"] = {"id": "sampleValue", "kind": "Property", "body": "probe", "imports": [core_import()],
                               "spec": {"type": value_type, "minCount": 1, "maxCount": 1}}
        defs["value-probe"] = {"id": "value-probe", "kind": "ValueProbe", "body": "probe",
                               "spec": {"sampleValue": parse_yaml("value: " + scalar)["value"]}}
        try:
            validate(trial)
        except ValueError:
            return False
        return True

    results["yaml_integer_property_accepts_big_integer"] = accepts_property("integer", "1" + "0" * 5000)
    results["yaml_number_property_accepts_exact_decimal"] = accepts_property("number", "0.100000000000000000000000000001")
    results["yaml_number_property_accepts_integer"] = accepts_property("number", "42")
    results["yaml_integer_property_does_not_coerce_decimal"] = not accepts_property("integer", "42.0")
    results["yaml_integer_property_does_not_coerce_string"] = not accepts_property("integer", '"42"')
    results["yaml_string_property_does_not_coerce_number"] = not accepts_property("string", "42")
    results["yaml_boolean_property_does_not_coerce_string"] = not accepts_property("boolean", '"true"')
    results["yaml_number_property_does_not_coerce_hex_text"] = not accepts_property("number", "0x10")
    return results


def unicode_probes(baseline: dict[str, dict[str, Any]]) -> dict[str, bool]:
    results: dict[str, bool] = {}
    project, vocab, core = "example/connector/project", "example/connector/vocabulary", "markitect/core"

    def rejected(action, fragment: str) -> bool:
        try:
            action()
        except ValueError as error:
            return fragment in str(error)
        return False

    def mutated(mutator, fragment: str) -> bool:
        trial = copy.deepcopy(baseline)
        return rejected(lambda: (mutator(trial), validate(trial)), fragment)

    whitespace_points = (*range(0x09, 0x0E), 0x20, 0x85, 0xA0, 0x1680,
                         *range(0x2000, 0x200B), 0x2028, 0x2029, 0x202F, 0x205F, 0x3000)
    whitespace = "".join(chr(point) for point in whitespace_points)
    results["unicode_white_space_exact_15_1_table"] = WHITE_SPACE == frozenset(whitespace)
    for label, body in (("empty", ""), ("mixed", whitespace),
                        *((f"u{point:04x}", chr(point)) for point in whitespace_points)):
        results[f"body_white_space_only_{label}_rejected"] = mutated(
            lambda t, b=body: t[project]["definitions"]["list-documents"].update(body=b), "outside Unicode White_Space")
    for point in (0x1C, 0x1D, 0x1E, 0x1F, 0x200B, 0xFEFF):
        trial = copy.deepcopy(baseline)
        body = chr(point)
        trial[project]["definitions"]["list-documents"]["body"] = body
        # Structural text validity is separate from the Markdown probe's
        # already documented capability limit for U+001C..U+001F.
        try:
            validate(trial)
        except ProbeLimitError as error:
            if point not in (0x1C, 0x1D, 0x1E, 0x1F):
                raise
            results[f"body_outside_white_space_u{point:04x}_keeps_markdown_capability_limit"] = "pinned parser whitespace handling" in str(error)
        results[f"body_outside_white_space_u{point:04x}_retained"] = trial[project]["definitions"]["list-documents"]["body"] == body

    def replace_names(value, replacements):
        if isinstance(value, dict):
            replaced = {replacements.get(key, key): replace_names(item, replacements) for key, item in value.items()}
            if isinstance(value, ModelPackages):
                return ModelPackages(value.models, value.entry_model, replaced)
            return replaced
        if isinstance(value, list):
            return [replace_names(item, replacements) for item in value]
        return replacements.get(value, value) if isinstance(value, str) else value

    unicode_model = replace_names(baseline, {
        vocab: "beispiel/überweisung", "Exchange": "Überweisung", "requester": "gültigFür",
        "api": "文档", "Connector": "Übertragung", "exchange.yaml": "überweisung.yaml",
        "Connector.Container": "Übertragung.Container",
        f"{vocab}/Command": "beispiel/überweisung/Command",
    })
    sample = unicode_model[project]["definitions"]["list-documents"]
    sample["kind"] = "Übertragung.Überweisung"
    sample["body"] = "Text bleibt exakt: u\u0308, straße, strasse und 📄."
    sample["title"] = "文档 📄"
    unicode_model[project]["definitions"]["list-documents-command"]["spec"]["operationId"] = "u\u0308📄"
    location = unicode_model["beispiel/überweisung"]["definitions"]["location"]["spec"]["values"]
    location.append("u\u0308📄")
    unicode_model[project]["definitions"]["文档"]["spec"]["location"] = "u\u0308📄"
    before = copy.deepcopy(unicode_model)
    validate(unicode_model)
    results["unicode_packages_aliases_ids_fields_filenames_and_references"] = unicode_model == before
    results["unicode_text_values_remain_exact"] = sample["body"] == before[project]["definitions"]["list-documents"]["body"]
    results["unicode_uncased_identifiers"] = all(name_matches("文档", style) for style in (KIND_STYLE, PROPERTY_STYLE, CONCRETE_STYLE))
    results["unicode_combining_marks_allowed_in_nfc"] = name_matches("किताब", PROPERTY_STYLE)
    results["unicode_digits_in_kebab_suffixes"] = name_matches("größe-2", CONCRETE_STYLE) and name_matches("文档-٢", CONCRETE_STYLE)
    dotless = replace_names(baseline, {"requester": "größeWert"})
    validate(dotless)
    results["unicode_dotless_property_default_field"] = True
    dotted = replace_names(baseline, {"requester": "größe.wert"})
    results["unicode_dotted_property_id_rejected"] = rejected(lambda: validate(dotted), "must not contain dots")
    results["unicode_canonical_caseless_key"] = name_key("straße") == name_key("strasse")
    results["unicode_canonical_equivalence_key"] = name_key("ü") == name_key("u\u0308")
    results["unicode_nfc_is_not_compatibility_folding"] = name_key("Ａ") != name_key("A")
    results["unicode_no_transliteration"] = name_key("ä") != name_key("ae")

    def add_ids(t):
        for identifier in ("straße", "strasse"):
            t[project]["definitions"][identifier] = {"id": identifier}
    results["unicode_sharp_s_definition_collision"] = mutated(add_ids, "definition ID collision")
    def exact_collision(t):
        add_ids(t)
        resolve(t, project, t[project]["definitions"]["list-documents"], "straße")
    results["unicode_exact_reference_does_not_hide_collision"] = mutated(exact_collision, "ambiguous")

    def add_packages(t):
        for package_name in ("beispiel/straße", "beispiel/strasse"):
            t[package_name] = copy.deepcopy(t[project])
    results["unicode_sharp_s_bound_package_collision"] = mutated(add_packages, "package ID collision")

    def add_aliases(t):
        t[project]["definitions"]["list-documents"]["imports"] = [
            {"package": vocab, "as": "Straße"}, {"package": core, "as": "Strasse"}]
    results["unicode_sharp_s_alias_collision"] = mutated(add_aliases, "alias collision")

    def field_collision(t):
        add_duplicate_case(t)
        t[project]["definitions"]["first"]["spec"]["name"] = "straße"
        t[project]["definitions"]["second"]["spec"]["name"] = "strasse"
    results["unicode_sharp_s_field_collision"] = mutated(field_collision, "effective field collision")
    results["unicode_sharp_s_filename_collision"] = rejected(
        lambda: check_filename_list(["straße.yaml", "strasse.yaml"]), "filename collision")
    results["unicode_sharp_s_yaml_key_collision"] = rejected(
        lambda: parse_yaml("straße: 1\nstrasse: 2\n"), "duplicate YAML key")

    def add_street(t):
        t[project]["definitions"]["straße"] = {"id": "straße", "kind": "MqttService", "body": "probe", "spec": {}}
        resolve(t, project, t[project]["definitions"]["list-documents"], "strasse")
    results["unicode_folded_reference_is_not_accepted"] = mutated(add_street, "wrong-case reference")
    results["unicode_non_nfc_reference_rejected"] = mutated(
        lambda t: resolve(t, project, t[project]["definitions"]["list-documents"], "u\u0308berweisung"), "must already be NFC")
    results["unicode_non_nfc_definition_rejected"] = mutated(
        lambda t: t[project]["definitions"].update({"u\u0308berweisung": {
            "id": "u\u0308berweisung", "kind": "MqttService", "body": "probe", "spec": {}}}), "must already be NFC")
    results["unicode_non_nfc_alias_rejected"] = mutated(
        lambda t: t[project]["definitions"]["list-documents"]["imports"][0].update({"as": "U\u0308bertragung"}), "must already be NFC")
    results["unicode_non_nfc_field_rejected"] = mutated(
        lambda t: t[project]["definitions"]["list-documents"]["spec"].update({"gu\u0308ltig": "api"}), "must already be NFC")
    results["unicode_non_nfc_explicit_property_name_rejected"] = mutated(
        lambda t: t[vocab]["definitions"]["requester"]["spec"].update({"name": "gu\u0308ltig"}), "must already be NFC")
    results["unicode_non_nfc_package_rejected"] = mutated(
        lambda t: t[project]["definitions"]["list-documents"]["imports"][0].update({"package": "beispiel/u\u0308berweisung"}), "must already be NFC")
    results["unicode_non_nfc_filename_rejected"] = rejected(
        lambda: check_filename_list(["u\u0308berweisung.yaml"]), "must already be NFC")
    results["unicode_emoji_names_rejected"] = not name_matches("📄", KIND_STYLE)
    results["unicode_table_version_mismatch_rejected"] = rejected(
        lambda: check_unicode_version("0.0.0"), "Unicode table version must be")
    results["unicode_existing_styles_remain_strict"] = all(not name_matches(value, style) for value, style in (
        ("überweisung", KIND_STYLE), ("GültigFür", PROPERTY_STYLE), ("überWeisung", CONCRETE_STYLE),
        ("größe_wert", PROPERTY_STYLE), ("文档API", KIND_STYLE)))
    return results


def make_test_only_valid_baseline(source: dict[str, dict[str, Any]]) -> dict[str, dict[str, Any]]:
    baseline = copy.deepcopy(source)
    vocab = baseline["example/connector/vocabulary"]["definitions"]
    if "Command" in vocab and "command" in vocab:
        prop = vocab.pop("command")
        prop["id"] = "commandField"  # Test-only neutral ID; not a resolution of the modeling choice.
        prop.setdefault("spec", {})["name"] = "command"
        vocab["commandField"] = prop
        baseline["example/connector/vocabulary"]["paths"]["commandField"] = baseline["example/connector/vocabulary"]["paths"].pop("command")
        schema = vocab["Exchange"]["spec"]["properties"]
        vocab["Exchange"]["spec"]["properties"] = ["commandField" if item == "command" else item for item in schema]
    record = baseline.models[baseline.entry_model]
    record["load_diagnostics"] = [error for error in record.get("load_diagnostics", [])
                                  if not ("definition ID collision" in error and "'Command'" in error and "'command'" in error)]
    return baseline


def duplicate_key_probe() -> bool:
    try:
        parse_yaml("id: first\nID: second\n")
    except DuplicateKeyError as error:
        return "case-insensitive duplicate YAML key" in str(error)
    return False


def core_import() -> dict[str, str]:
    return {"package": "markitect/core", "as": "Core"}


def add_object_case(packages: dict[str, dict[str, Any]], value: dict[str, Any]) -> None:
    p = packages["example/connector/project"]["definitions"]
    p["object-probe"] = {"id": "object-probe", "kind": "ObjectProbe", "body": "probe", "spec": {"payload": value}}
    p["ObjectProbe"] = {"id": "ObjectProbe", "kind": "Kind", "body": "probe", "spec": {"properties": ["payload"]}, "imports": [core_import()]}
    p["payload"] = {"id": "payload", "kind": "Property", "body": "probe", "spec": {"type": "object", "minCount": 1, "maxCount": 1, "properties": ["messageId"]}, "imports": [core_import()]}
    p["messageId"] = {"id": "messageId", "kind": "Property", "body": "probe", "spec": {"type": "string", "minCount": 1, "maxCount": 1}, "imports": [core_import()]}


def add_empty_object_case(packages: dict[str, dict[str, Any]]) -> None:
    p = packages["example/connector/project"]["definitions"]
    p["empty-object-probe"] = {"id": "empty-object-probe", "kind": "EmptyObjectProbe", "body": "probe", "spec": {"payload": {}}}
    p["EmptyObjectProbe"] = {"id": "EmptyObjectProbe", "kind": "Kind", "body": "probe", "spec": {"properties": ["payload"]}, "imports": [core_import()]}
    p["payload"] = {"id": "payload", "kind": "Property", "body": "probe", "spec": {"type": "object", "minCount": 1, "maxCount": 1}, "imports": [core_import()]}


def add_string_object_case(packages: dict[str, dict[str, Any]]) -> None:
    p = packages["example/connector/project"]["definitions"]
    p["string-probe"] = {"id": "string-probe", "kind": "StringProbe", "body": "probe", "spec": {"label": "text"}}
    p["StringProbe"] = {"id": "StringProbe", "kind": "Kind", "body": "probe", "spec": {"properties": ["label"]}, "imports": [core_import()]}
    p["label"] = {"id": "label", "kind": "Property", "body": "probe", "spec": {"type": "string", "minCount": 1, "maxCount": 1, "properties": []}, "imports": [core_import()]}


def add_duplicate_case(packages: dict[str, dict[str, Any]]) -> None:
    p = packages["example/connector/project"]["definitions"]
    p["duplicate-probe"] = {"id": "duplicate-probe", "kind": "DuplicateProbe", "body": "probe", "spec": {"same": "x"}}
    p["DuplicateProbe"] = {"id": "DuplicateProbe", "kind": "Kind", "body": "probe", "spec": {"properties": ["first", "second"]}, "imports": [core_import()]}
    for name, field in (("first", "sameField"), ("second", "samefield")):
        p[name] = {"id": name, "kind": "Property", "body": "probe", "spec": {"name": field, "type": "string", "minCount": 0, "maxCount": 1}, "imports": [core_import()]}


def add_cycle_case(packages: dict[str, dict[str, Any]]) -> None:
    p = packages["example/connector/project"]["definitions"]
    p["left"] = {"id": "left", "kind": "Property", "body": "probe", "spec": {"type": "object", "minCount": 1, "maxCount": 1, "properties": ["right"]}, "imports": [core_import()]}
    p["right"] = {"id": "right", "kind": "Property", "body": "probe", "spec": {"type": "object", "minCount": 1, "maxCount": 1, "properties": ["left"]}, "imports": [core_import()]}


def filename_style_probe() -> bool:
    return filename_matches("list-documents.yaml") and not filename_matches("list-Documents.yaml")


def expect_filename_collision() -> bool:
    try:
        check_filename_list(["list-documents.yaml", "List-Documents.yaml"])
    except ValueError as error:
        return "case-insensitive filename collision" in str(error)
    return False


def contract_example_probes(packages: dict[str, dict[str, Any]]) -> dict[str, bool]:
    """Check informative excerpts against their executable sources, separately from v1."""
    example = (ROOT / "README.md").read_text(encoding="utf-8")
    section = anchored_section(example, "definition-excerpts")
    definitions = [definition for source in re.findall(r"```yaml\s*\n(.*?)\n```", section, re.S)
                   if {"id", "kind", "body", "spec"} <= set(definition := parse_yaml(source))]
    try:
        for definition in definitions:
            package_imports(packages, "example/connector/project", definition)
        imports_match = bool(definitions)
    except ValueError:
        imports_match = False
    examples = re.findall(r"\*\*([^*]+\.yaml)\*\*\s+```yaml\n(.*?)\n```", section, re.S)
    match = len(examples) == 9
    for filename, source in examples:
        paths = [path for package in packages.values() for path in package["paths"].values() if path.name == filename]
        match = match and len(paths) == 1 and parse_yaml(source) == load_yaml(paths[0])
    metadata = anchored_section(example, "manifest-excerpts")
    manifest_examples = re.findall(r"\*\*(model\.yaml|dependencies/core/model\.yaml)\*\*\s+```yaml\n(.*?)\n```", metadata, re.S)
    manifests_match = len(manifest_examples) == 2 and all(
        parse_yaml(source) == load_yaml(ROOT / path) for path, source in manifest_examples)
    lock_examples = re.findall(r"\*\*model-lock\.yaml\*\*\s+```yaml\n(.*?)\n```", metadata, re.S)
    lock_match = len(lock_examples) == 1 and parse_yaml(lock_examples[0]) == load_yaml(LOCK_PATH)
    return {"example_definition_imports_use_selected_form": imports_match,
            "example_definition_excerpts_equal_fixture_sources": match,
            "example_manifest_excerpts_equal_fixture_sources": manifests_match,
            "example_lock_excerpt_equals_fixture_source": lock_match}


def folder_probes(baseline: dict[str, dict[str, Any]]) -> dict[str, bool]:
    """Exercise discovery on disk, including moves and Unicode collisions."""
    results: dict[str, bool] = {}

    def rejected(action, fragment):
        try:
            action()
        except ValueError as error:
            return fragment in str(error)
        return False

    def write(directory, filename, identifier):
        directory.mkdir(parents=True, exist_ok=True)
        path = directory / filename
        path.write_text(f"id: {identifier}\nkind: Example\nbody: probe\nspec: {{}}\n", encoding="utf-8")
        return path

    results["folder_path_maps_to_slash_package"] = folder_package(("example", "connector", "project")) == "example/connector/project"
    results["folder_root_has_no_package_name"] = rejected(lambda: folder_package(()), "forbidden directly in the model root")
    for name in ("Wrong", "one.two", "one-two", "..", "u\u0308berweisung"):
        results[f"folder_invalid_component_{name}"] = rejected(lambda: folder_package((name,)), "package folder")
    results["folder_unicode_components"] = folder_package(("beispiel", "überweisung", "文档")) == "beispiel/überweisung/文档"

    with tempfile.TemporaryDirectory(prefix="markitect-folders-") as temporary:
        root = Path(temporary) / "model"
        source = write(root / "example" / "connector", "first.yaml", "item")
        write(root / "example" / "connector" / "child", "other.yaml", "item")
        before_bytes = source.read_bytes()
        diagnostics = []
        first = discover_packages(root, diagnostics)
        results["folder_every_parent_is_package_without_manifest"] = set(first) == {"example", "example/connector", "example/connector/child"} and not diagnostics
        results["folder_root_is_excluded_from_package_inventory"] = all(name and name != "." for name in first) and all(package["directory"] != root for package in first.values())
        results["folder_child_is_separate_package"] = all("item" in first[name]["definitions"] for name in ("example/connector", "example/connector/child"))
        results["folder_parent_package_can_be_empty"] = not first["example"]["definitions"]
        results["folder_package_has_no_version_or_manifest"] = all(set(package) == {"directory", "definitions", "paths", "body_sources"} for package in first.values())
        source = source.rename(source.with_name("renamed.yaml"))
        renamed = discover_packages(root, [])
        results["folder_filename_rename_keeps_identity"] = first["example/connector"]["definitions"] == renamed["example/connector"]["definitions"]
        target = root / "example" / "moved"
        target.mkdir()
        source.rename(target / source.name)
        moved = discover_packages(root, [])
        results["folder_move_changes_package_keeps_local_id"] = not moved["example/connector"]["definitions"] and "item" in moved["example/moved"]["definitions"]
        results["folder_discovery_preserves_source_bytes"] = (target / source.name).read_bytes() == before_bytes
        write(target, "additional.yaml", "additional")
        results["folder_new_definition_discovered_without_file_list"] = set(discover_packages(root, [])["example/moved"]["definitions"]) == {"item", "additional"}
        write(root / "beispiel" / "überweisung" / "文档", "größe-2.yaml", "größe-2")
        results["folder_unicode_path_filename_and_id_discovered"] = "größe-2" in discover_packages(root, [])["beispiel/überweisung/文档"]["definitions"]
        (root / "example" / "connector").rename(root / "example" / "renamed")
        renamed = discover_packages(root, [])
        results["folder_parent_rename_changes_descendant_package"] = "example/renamed/child" in renamed and "example/connector/child" not in renamed
        relocated = root.rename(root.with_name("relocated-model"))
        results["folder_physical_model_root_move_keeps_packages"] = set(renamed) == set(discover_packages(relocated, []))
        root = relocated
        (root / "straße").mkdir()
        (root / "strasse").mkdir()
        results["folder_unicode_path_collision_even_when_empty"] = rejected(lambda: discover_packages(root, []), "package ID collision")

    with tempfile.TemporaryDirectory(prefix="markitect-folders-invalid-") as temporary:
        root = Path(temporary) / "model"
        write(root / "example", "Wrong.yaml", "item")
        results["folder_wrong_filename_rejected_without_correction"] = rejected(lambda: discover_packages(root, []), "filename must") and (root / "example" / "Wrong.yaml").exists()
        (root / "example" / "Wrong.yaml").rename(root / "example" / "correct.yaml")
        write(root / "example", "duplicate.yaml", "item")
        diagnostics = []
        discover_packages(root, diagnostics)
        results["folder_duplicate_id_is_diagnostic"] = any("definition ID collision" in error for error in diagnostics)
        write(root, "root.yaml", "root-item")
        root_bytes = (root / "root.yaml").read_bytes()
        try:
            discover_packages(root, [])
        except ProbeLimitError:
            results["folder_root_definition_is_language_error"] = False
        except ValueError as error:
            results["folder_root_definition_is_language_error"] = "forbidden directly in the model root" in str(error)
        else:
            results["folder_root_definition_is_language_error"] = False
        results["folder_root_rejection_does_not_relocate_definition"] = (root / "root.yaml").read_bytes() == root_bytes

    project = "example/connector/project"
    trial = copy.deepcopy(baseline)
    trial["example/connector"]["definitions"]["parent-only"] = {"id": "parent-only"}
    definition = trial[project]["definitions"]["list-documents"]
    results["folder_parent_definitions_not_implicitly_visible"] = rejected(lambda: resolve(trial, project, definition, "parent-only"), "unresolved")
    definition["imports"].append({"package": "example/connector", "as": "Parent"})
    results["folder_parent_explicit_import_resolves"] = resolve(trial, project, definition, "Parent.parent-only") == ("example/connector", "parent-only")
    definition["imports"].append({"package": "absent/package", "as": "Absent"})
    results["folder_absent_package_import_rejected"] = rejected(lambda: package_imports(trial, project, definition), "is absent")
    return results


def package_collision_probe(packages: dict[str, dict[str, Any]]) -> bool:
    trial = copy.deepcopy(packages)
    trial["Markitect/core"] = trial["markitect/core"]
    try:
        check_identity_collisions(trial)
    except ValueError as error:
        return "case-insensitive package ID collision" in str(error)
    return False


def identifier_style_probe() -> bool:
    for value, pattern, category in (("MqttService", KIND_STYLE, "Kind"), ("messageId", PROPERTY_STYLE, "Property")):
        try:
            check_segment_style(value, pattern, category)
        except ValueError:
            return False
    for value, pattern, category in (("MqttSERVICE", KIND_STYLE, "Kind"), ("MQTTService", KIND_STYLE, "Kind"),
                                     ("messageID", PROPERTY_STYLE, "Property"), ("maxATTEMPTS", PROPERTY_STYLE, "Property")):
        try:
            check_segment_style(value, pattern, category)
        except ValueError:
            continue
        return False
    return True


@resource_checked("input provenance")
def hashes() -> dict[str, str]:
    files = [CONTRACT, DIAGNOSTIC_CONTRACT, ROOT / "README.md", ROOT / "requirements.txt", ROOT / "canonical/core.json", *ROOT.glob("*.py"), *ROOT.rglob("*.yaml")]
    return {path.relative_to(ROOT.parent).as_posix(): hashlib.sha256(path.read_bytes()).hexdigest() for path in sorted(files)}


def scan_body_markers(packages: dict[str, dict[str, Any]] | None) -> int:
    if not packages:
        return 0
    return sum(len(re.findall(r"\[\[", definition.get("body", "")))
               for package in packages.values() for definition in package["definitions"].values()
               if isinstance(definition.get("body", ""), str))


@resource_checked("verification/reporting")
def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--report", type=Path, help="write compact JSON evidence report")
    parser.add_argument("--self-test", action="store_true", help="run probes on a labeled in-memory baseline; does not validate or repair source")
    parser.add_argument("--canonical-model", help="export canonical bytes for one validated model and its dependency closure")
    parser.add_argument("--canonical-output", type=Path, help="explicit output file for --canonical-model")
    parser.add_argument("--max-object-depth", type=int, default=DEFAULT_OBJECT_DEPTH,
                        help="probe budget for actual object values; exhaustion is not a language error")
    args = parser.parse_args()
    if bool(args.canonical_model) != bool(args.canonical_output):
        parser.error("--canonical-model and --canonical-output are required together")
    if args.max_object_depth < 0:
        parser.error("--max-object-depth must be nonnegative")
    if args.canonical_model:
        try:
            selected = package_data([])
        except ProbeLimitError as error:
            print(f"NOT_COMPLETED: {error}", file=sys.stderr)
            return 2
        record = export_canonical(selected, args.canonical_model, args.canonical_output, max_object_depth=args.max_object_depth)
        if record["status"] != "passed":
            print(f"{record['status'].upper()}: {'; '.join(record['diagnostics'])}", file=sys.stderr)
            return 2 if record["status"] == "not_completed" else 1
        print(f"{args.canonical_model}: {record['digest']}; {record['byte_count']} canonical bytes; selected model scope only")
        return 0
    diagnostics: list[str] = []
    packages = None
    canonical_records = {}
    counts: dict[str, int] = {}
    test_results: dict[str, bool] = {}
    test_diagnostics: list[str] = []
    source_limited = False
    source_invalid = False
    body_records: list[dict[str, Any]] = []
    body_diagnostics: list[str] = []
    body_limited = False
    lock_record = {"status": "not_completed", "diagnostics": ["Model manifests were not loaded; lock was not checked."], "content_digest_checked": False}
    try:
        packages = package_data(diagnostics)
        source_invalid = bool(diagnostics)
        computed_digests = {}
        for model_id in sorted(packages.models):
            canonical_records[model_id], _ = assess_canonical(packages, model_id, max_object_depth=args.max_object_depth)
            if canonical_records[model_id]["status"] == "passed":
                computed_digests[model_id] = canonical_records[model_id]["digest"]
        try:
            external = set(packages.models) - {packages.entry_model}
            missing = external - set(computed_digests)
            if any(canonical_records[key]["status"] == "failed" for key in missing):
                fail("cannot verify lock: an external model failed canonical validation")
            if missing:
                raise ProbeLimitError("cannot verify lock: external canonical evaluation not completed")
            lock_record = load_lock(LOCK_PATH, packages, read_yaml=load_yaml, source_root=ROOT,
                                    computed_digests={key: computed_digests[key] for key in external})
        except ProbeLimitError as error:
            lock_record = {"status": "not_completed", "diagnostics": [str(error)], "content_digest_checked": False}
        except (ValueError, OSError) as error:
            lock_record = {"status": "failed", "diagnostics": [str(error)], "content_digest_checked": False}
        try:
            body_records = check_body_links(packages)
        except ProbeLimitError as error:
            source_limited = body_limited = True
            body_diagnostics.append(str(error))
        except ValueError as error:
            source_invalid = True
            body_diagnostics.append(str(error))
        try:
            counts = validate(packages, max_object_depth=args.max_object_depth)
        except ProbeLimitError as error:
            source_limited = True
            diagnostics.append(str(error))
        except Exception as error:
            source_invalid = True
            diagnostics.append(str(error))
        try:
            test_results = test_suite(packages, diagnostics)
        except ProbeLimitError as error:
            test_diagnostics.append(str(error))
        except Exception as error:
            test_results["test_suite_exception"] = False
            test_diagnostics.append(f"self-test error: {error}")
    except ProbeLimitError as error:
        source_limited = True
        diagnostics.append(str(error))
    except Exception as error:
        source_invalid = True
        diagnostics.append(str(error))

    diagnostics.extend(error for error in body_diagnostics if error not in diagnostics)
    test_passed = bool(test_results) and all(test_results.values())
    source_status = "failed" if source_invalid else "not_completed" if source_limited or packages is None else "passed"
    source_passed = source_status == "passed"
    body_link_scan = scan_body_markers(packages)
    test_status = "passed" if test_passed else "failed" if test_results else "not_completed"
    result_status = overall_status(source_status, canonical_records, lock_record["status"], test_status)
    overall_passed = result_status == "passed"
    coordinate_record = diagnostic_coordinate_report(packages)
    result = {
        "status": result_status,
        "source_validation": {"status": source_status, "resource_abort": source_limited, "diagnostics": diagnostics},
        "self_test": {"status": test_status, "results": test_results, "diagnostics": test_diagnostics,
                       "baseline_note": "Separate in-memory probe baseline changes the colliding Property ID command to commandField and sets spec.name: command, clearing only this fixture collision from its test-only load diagnostics; this is not a source repair or decision about the model."},
        "runtime": {"python": sys.version.split()[0], "pyyaml": yaml.__version__, "unicode": unicodedata.unidata_version,
                    "required_unicode": UNICODE_VERSION, "markdown_it": package_version("markdown-it-py"), "mdurl": package_version("mdurl")},
        "model_label_profile": {"shared_model_id_version_grammar": True, "separators": [".", "-"],
                                "decimal_digit_start": True, "lowercase": True, "nfc_required": True,
                                "automatic_correction": False, "model_id_collision_scope": "supplied model set",
                                "version_collision_scope": "same model", "version_binding": "exact declared label"},
        "model_dependency_profile": {"acyclic": True, "self_dependencies_allowed": False,
                                     "shared_dependencies_allowed": True, "cycle_status": "failed"},
        "root_path_profile": {"relative_to": "model.yaml directory", "dot_root_allowed": True,
                              "separator": "/", "folder_grammar": "lowercase Unicode Package folder",
                              "existing_directory_required": True, "exact_physical_spelling": True,
                              "automatic_normalization": False, "redirects_allowed": False,
                              "overlapping_bound_roots_allowed": False,
                              "real_redirection_probe": "junction on Windows; symlink on other supported hosts"},
        "object_cycle_profile": {"reference_cycles": "allowed", "object_cycles": "allowed if the required-edge graph is acyclic",
                                 "value_traversal": "actual finite values only; no schema unfolding"},
        "resource_profile": {"language_size_or_depth_maxima": False, "object_depth_budget": args.max_object_depth,
                             "python_recursion_limit": sys.getrecursionlimit(), "memory_byte_budget": None,
                             "runtime_memory_exhaustion": "not_completed", "exhaustion_status": "not_completed",
                             "incomplete_scope_has_digest": False, "budgets_affect_canonical_bytes": False,
                             "source_limits": "body_link_validation.probe_limits and yaml_value_profile.probe_limits",
                             "self_test_object_depth_budget": DEFAULT_OBJECT_DEPTH},
        "package_profile": {"model_root": "from each model.yaml root", "identity_source": "relative folder path", "separator": "/", "full_qualification_separator": "/", "alias_qualification_separator": ".",
                            "per_package_versions": False, "distribution_unit": "model", "distribution_binding": "model.yaml and model-lock.yaml closure/version checks; model-digest/1 canonical bytes and SHA-256 verified per valid model scope"},
        "packages": {name: {"definitions": len(package["definitions"]), "model": package["model"],
                            "directories": {model_id: directory.relative_to(ROOT).as_posix() for model_id, directory in package["locations"].items()}}
                     for name, package in (packages or {}).items()},
        "model_binding_validation": {"status": "passed" if packages is not None else "not_completed" if source_limited else "failed",
                                     "entry_model": packages.entry_model if packages is not None else None,
                                     "models": {identifier: {"manifest": model["path"].relative_to(ROOT).as_posix(),
                                                             **model["manifest"]} for identifier, model in (packages.models if packages is not None else {}).items()},
                                     "content_digest_checked": False,
                                     "scope": "manifest identity/version/ownership only; actual content checks are in lock_validation and canonical_models"},
        "lock_validation": {"file": "model-lock.yaml", **lock_record,
                            "digest_probe_note": "Source lock uses computed model-digest/1 hashes; synthetic digests remain confined to binding self-tests."},
        "canonical_profile": {"format": "markitect-model-digest/1", "hash": "sha256", "own_content_only": True,
                              "byte_artifact": "canonical/core.json", "sources_modified": False},
        "canonical_models": canonical_records,
        "diagnostic_coordinates": {"contract": "../diagnostic-contract.md", "affects_model_validity": False,
                                   "affects_canonical_digest": False, **coordinate_record},
        "body_link_markers": body_link_scan,
        "body_link_validation": {"status": "not_completed" if packages is None or body_limited else "failed" if body_diagnostics else "passed",
                                 "diagnostics": body_diagnostics, "commonmark": "0.31.2", "occurrences": body_records,
                                 "probe_limits": {"body_utf8_bytes": MAX_BODY_BYTES, "nesting": MAX_NESTING,
                                                  "unsupported_parser_whitespace": ["U+001C", "U+001D", "U+001E", "U+001F", "U+0085"]},
                                 "scope": "link syntax and reference resolution only; not a claim of complete source-model validity"},
        "yaml_value_profile": {"syntax": "restricted YAML 1.2.2", "scalar_resolution": "explicit; no library implicit resolvers",
                               "integer": "arbitrary precision", "number": "exact finite decimal", "type_coercion": False,
                               "probe_limits": {"coefficient_digits": MAX_NUMBER_DIGITS, "absolute_exponent": MAX_ABS_EXPONENT}},
        "list_profile": {"ordinary_values": "ordered", "core_declarations_and_imports": "unordered",
                         "optional_missing_equals_empty": True, "sources_modified": False,
                         "comparison": "bounded historical list view; full canonical bytes are checked separately"},
        **counts,
        "scope_limits": ["fixture-only local model loading and folder discovery; selected root spelling, no-redirection and non-overlap rules are checked; no network installation", "Python design probe; Go product implementation is separate", "PyYAML parses syntax only; selected YAML/value profile controls interpretation", "Unicode 15.1.0 XID naming profile, NFC author forms, canonical full caseless matching", "body and title and enum data are not case-normalized", "no automatic case correction", "selected model-ID/version-label, root-path, object-cycle and resource rules are enforced; model dependencies must be acyclic and cycles fail as language errors", "no production compiler/runtime proof", "CommonMark/body-link probes are bounded; no universal parser-conformance or semantic proof", "only explicitly supplied local model loading and exact version equality; no registry or networking; canonical model-digest/1 computation is implemented in the bounded probe", "imports use only the selected package/as mapping form", "canonical model-digest/1 is checked by fixed byte vectors and semantic equivalence/distinction probes; no Go product compiler claim"],
        "sha256": hashes(),
    }
    if args.report:
        args.report.write_text(json.dumps(result, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    for diagnostic in diagnostics:
        print(f"DIAGNOSTIC: {diagnostic}", file=sys.stderr)
    for diagnostic in lock_record.get("diagnostics", []):
        print(f"LOCK {lock_record['status'].upper()}: {diagnostic}", file=sys.stderr)
    for diagnostic in test_diagnostics:
        print(f"SELF-TEST: {diagnostic}", file=sys.stderr)
    if args.self_test:
        status = "PASS" if test_passed else "FAIL" if test_results else "NOT_COMPLETED"
        print(f"{status}: {len(test_results)} self-tests; fixture source status={'invalid' if source_invalid else 'valid' if source_passed else 'not completed'}; lock status={lock_record['status']}")
        for name, passed in test_results.items():
            if not passed:
                print(f"FAILED TEST: {name}", file=sys.stderr)
        return 0 if test_passed else 1 if test_results else 2
    if overall_passed:
        print(f"PASS: {counts.get('definitions', 0)} definitions; {counts.get('resolution_calls', 0)} resolver calls; {len(test_results)} self-tests")
        return 0
    print(f"{result_status.upper()}: source fixture is {'invalid' if source_invalid else 'not validated (resource limit)' if source_limited else 'valid' if source_passed else 'not completed'}; self-tests {test_status}; lock {lock_record['status']}")
    return 2 if result_status == "not_completed" else 1


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ProbeLimitError as error:
        print(f"NOT_COMPLETED: {error}", file=sys.stderr)
        raise SystemExit(2)
