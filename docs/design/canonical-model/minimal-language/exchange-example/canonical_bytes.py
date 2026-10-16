"""Canonical model-digest/1 projection and exact UTF-8 JSON encoding.

The caller supplies the existing language validator/resolver. A model and its
dependency closure must validate before any own-model bytes are produced.
"""
from __future__ import annotations

import copy
import hashlib
from decimal import Decimal

from model_manifest import ModelPackages


FORMAT = "markitect-model-digest/1"
CORE = "markitect/core"


def integer_text(value: int) -> str:
    """Decimal integer text without Python's configurable digit-string limit."""
    if type(value) is not int:
        raise ValueError("canonical integer requires an exact integer")
    negative = value < 0
    value = abs(value)
    chunks = []
    base = 10 ** 500
    while value >= base:
        value, remainder = divmod(value, base)
        chunks.append(str(remainder).zfill(500))
    return ("-" if negative else "") + str(value) + "".join(reversed(chunks))


def decimal_parts(value) -> list[str]:
    """Unique signed coefficient/exponent, with no decimal-context rounding."""
    if type(value) is int:
        text = integer_text(value)
        negative, digits, exponent = text.startswith("-"), text.lstrip("-"), 0
    elif isinstance(value, Decimal) and value.is_finite():
        parts = value.as_tuple()
        negative, digits, exponent = bool(parts.sign), "".join(chr(48 + digit) for digit in parts.digits), parts.exponent
    else:
        raise ValueError("canonical number requires an exact finite decimal or integer")
    digits = digits.lstrip("0")
    if not digits:
        return ["0", "0"]
    trimmed = digits.rstrip("0")
    exponent += len(digits) - len(trimmed)
    return [("-" if negative else "") + trimmed, integer_text(exponent)]


def json_bytes(value) -> bytes:
    """Sorted UTF-8 keys; fixed escapes; no BOM, whitespace or final newline."""
    escapes = {'"': '\\"', "\\": "\\\\", "\b": "\\b", "\t": "\\t", "\n": "\\n", "\f": "\\f", "\r": "\\r"}

    def string(text):
        return '"' + "".join(escapes.get(char, f"\\u{ord(char):04x}" if ord(char) < 32 else char) for char in text) + '"'

    def encode(item):
        if isinstance(item, str):
            return string(item)
        if type(item) is bool:
            return "true" if item else "false"
        if type(item) is int:
            return integer_text(item)
        if isinstance(item, list):
            return "[" + ",".join(encode(child) for child in item) + "]"
        if isinstance(item, dict) and all(isinstance(key, str) for key in item):
            return "{" + ",".join(string(key) + ":" + encode(item[key]) for key in sorted(item, key=lambda key: key.encode("utf-8"))) + "}"
        raise ValueError("canonical JSON tree contains an unsupported value; no implicit coercion")

    return encode(value).encode("utf-8")


def model_scope(packages: ModelPackages, model_id: str) -> ModelPackages:
    if model_id not in packages.models:
        raise ValueError(f"unknown canonical model {model_id!r}")
    reachable = set()
    pending = [model_id]
    while pending:
        identifier = pending.pop()
        if identifier in reachable:
            continue
        if identifier not in packages.models:
            raise ValueError(f"canonical dependency model {identifier!r} is unavailable")
        reachable.add(identifier)
        pending.extend(packages.models[identifier]["manifest"]["dependencies"])
    models = {identifier: packages.models[identifier] for identifier in sorted(reachable)}
    for record in models.values():
        if record.get("load_diagnostics"):
            raise ValueError("invalid source model before canonicalization: " + "; ".join(record["load_diagnostics"]))
    selected = ModelPackages(models, model_id)
    for name, package in packages.items():
        locations = {owner: path for owner, path in package["locations"].items() if owner in reachable}
        if not locations:
            continue
        if package["model"] in reachable:
            selected[name] = {**package, "locations": locations}
        else:
            selected[name] = {"model": None, "locations": locations, "directory": next(iter(locations.values())),
                              "definitions": {}, "paths": {}, "body_sources": {}}
    return copy.deepcopy(selected)


def canonical_model_bytes(packages, model_id, *, validate, resolve, kind_identity, body_links) -> bytes:
    scope = model_scope(packages, model_id)
    validate(scope)
    links = {}
    for occurrence in body_links(scope):
        key = (occurrence["definition"]["package"], occurrence["definition"]["id"])
        links.setdefault(key, []).append(["reference", occurrence["target"]["package"], occurrence["target"]["id"]])

    def fields(source_package, source, owner_package, owner, values, role=None):
        result = {}
        for prop_ref in owner.get("spec", {}).get("properties", []):
            prop_package, prop_id = resolve(scope, owner_package, owner, prop_ref)
            prop = scope[prop_package]["definitions"][prop_id]
            contract = prop["spec"]
            field_name = contract.get("name", prop_id)
            is_list = contract["maxCount"] != 1
            is_core_field = prop_package == CORE
            if role == (CORE, "Property") and is_core_field:
                if prop_id == "properties" and values["type"] != "object":
                    continue
                if prop_id == "values" and values["type"] != "enum":
                    continue
            if field_name in values:
                value = values[field_name]
            elif role == (CORE, "Property") and (prop_package, prop_id) == (CORE, "name"):
                value = source["id"]
            elif is_list and contract["minCount"] == 0:
                value = []
            else:
                continue

            def scalar(item):
                kind = contract["type"]
                if kind == "reference":
                    return ["reference", *resolve(scope, source_package, source, item)]
                if kind == "object":
                    return ["object", fields(source_package, source, prop_package, prop, item)]
                if kind == "integer":
                    return ["integer", integer_text(item)]
                if kind == "number":
                    return ["number", *decimal_parts(item)]
                return [kind, item]  # Validated string, enum or boolean.

            if is_list:
                items = [scalar(item) for item in value]
                declaration = is_core_field and (
                    (role == (CORE, "Kind") and prop_id == "properties") or
                    (role == (CORE, "Property") and prop_id in {"properties", "values"}))
                if declaration:
                    items.sort(key=json_bytes)
                result[field_name] = ["list", items]
            else:
                result[field_name] = scalar(value)
        return result

    inventory = {}
    for package_name, package in scope.items():
        if package["model"] != model_id or not package["definitions"]:
            continue
        definitions = {}
        for identifier, definition in package["definitions"].items():
            role = kind_identity(scope, package_name, definition)
            owner = scope[role[0]]["definitions"][role[1]]
            record = {"kind": ["reference", *role], "body": definition["body"],
                      "bodyLinks": links.get((package_name, identifier), []),
                      "spec": fields(package_name, definition, role[0], owner, definition["spec"], role)}
            if "title" in definition:
                record["title"] = definition["title"]
            definitions[identifier] = record
        inventory[package_name] = definitions
    manifest = scope.models[model_id]["manifest"]
    metadata = {key: manifest[key] for key in ("formatVersion", "id", "version", "dependencies") if key in manifest}
    return json_bytes({"format": FORMAT, "model": metadata, "packages": inventory})


def digest(data: bytes) -> str:
    return "sha256:" + hashlib.sha256(data).hexdigest()
