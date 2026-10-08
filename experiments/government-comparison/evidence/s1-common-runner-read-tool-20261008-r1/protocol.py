"""Bounded validation of the frozen local protocol schema subset; no I/O launch API."""
import hashlib
import json
import math
import re
import zipfile

ANNOTATIONS = {"$schema", "title", "description", "default", "format", "definitions", "deprecated", "examples"}
ASSERTIONS = {"$ref", "type", "properties", "required", "additionalProperties", "items", "oneOf", "anyOf", "allOf", "enum", "const", "minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "minLength", "maxLength", "pattern", "minItems", "maxItems", "uniqueItems", "minProperties", "maxProperties"}


def require(ok, reason="protocol-schema-invalid"):
    if not ok:
        raise ValueError(reason)


def strict_json(raw):
    def pairs(items):
        result = {}
        for key, value in items:
            require(key not in result, "duplicate-json-key")
            result[key] = value
        return result
    return json.loads(raw, object_pairs_hook=pairs, parse_float=lambda x: finite_float(x), parse_constant=lambda _: (_ for _ in ()).throw(ValueError("invalid-json-number")))


def finite_float(value):
    number = float(value)
    require(math.isfinite(number), "invalid-json-number")
    return number


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), allow_nan=False)


class Protocol:
    def __init__(self, archive, contract):
        require(hashlib.sha256(archive.read_bytes()).hexdigest() == contract["schemaArchiveSha256"], "protocol-archive-pin-mismatch")
        self.documents = {}
        with zipfile.ZipFile(archive) as source:
            for member, digest in contract["memberSha256"].items():
                raw = source.read(member)
                require(hashlib.sha256(raw).hexdigest() == digest, "protocol-member-pin-mismatch")
                schema = strict_json(raw)
                self._supported(schema)
                self.documents[member] = schema

    def _supported(self, schema):
        if type(schema) is bool:
            return
        require(type(schema) is dict, "protocol-schema-shape")
        require(not set(schema) - ANNOTATIONS - ASSERTIONS, "unsupported-schema-keyword")
        if "$ref" in schema:
            require(schema["$ref"].startswith("#/definitions/"), "external-schema-reference")
        for key in ("definitions", "properties"):
            for child in schema.get(key, {}).values():
                self._supported(child)
        for key in ("oneOf", "anyOf", "allOf"):
            for child in schema.get(key, []):
                self._supported(child)
        for key in ("items", "additionalProperties"):
            if key in schema:
                self._supported(schema[key])

    def validate(self, member, value):
        require(member in self.documents, "unfrozen-schema-member")
        root = self.documents[member]
        self._check(root, value, root, 0, [100000])

    def _check(self, schema, value, root, depth, budget):
        budget[0] -= 1
        require(depth <= 64 and budget[0] >= 0, "schema-resource-limit")
        if type(schema) is bool:
            require(schema)
            return
        if "$ref" in schema:
            name = schema["$ref"].split("/")[-1].replace("~1", "/").replace("~0", "~")
            require(name in root.get("definitions", {}), "schema-reference-missing")
            self._check(root["definitions"][name], value, root, depth+1, budget)
            return  # Draft7 ignores assertion siblings of $ref.
        kinds = {"null": type(value) is type(None), "boolean": type(value) is bool,
                 "integer": type(value) is int, "number": type(value) in (int, float),
                 "string": type(value) is str, "object": type(value) is dict,
                 "array": type(value) is list}
        if "type" in schema:
            types = schema["type"] if type(schema["type"]) is list else [schema["type"]]
            require(any(kinds.get(k, False) for k in types))
        if "enum" in schema:
            require(canonical(value) in [canonical(x) for x in schema["enum"]])
        if "const" in schema:
            require(canonical(value) == canonical(schema["const"]))
        for mode in ("allOf", "anyOf", "oneOf"):
            if mode not in schema:
                continue
            matches = 0
            for branch in schema[mode]:
                try:
                    self._check(branch, value, root, depth+1, budget)
                    matches += 1
                except ValueError as exc:
                    if str(exc) != "protocol-schema-invalid":
                        raise
            require(matches == len(schema[mode]) if mode == "allOf" else matches == 1 if mode == "oneOf" else matches >= 1)
        if type(value) is dict:
            require(all(key in value for key in schema.get("required", [])))
            require(schema.get("minProperties", 0) <= len(value) <= schema.get("maxProperties", 1000000))
            properties = schema.get("properties", {})
            for key, item in value.items():
                require(type(key) is str)
                self._check(properties.get(key, schema.get("additionalProperties", True)), item, root, depth+1, budget)
        if type(value) is list:
            require(schema.get("minItems", 0) <= len(value) <= schema.get("maxItems", 1000000))
            if schema.get("uniqueItems"):
                require(len({canonical(x) for x in value}) == len(value))
            for item in value:
                self._check(schema.get("items", True), item, root, depth+1, budget)
        if type(value) is str:
            require(schema.get("minLength", 0) <= len(value) <= schema.get("maxLength", 10000000))
            if "pattern" in schema:
                require(re.search(schema["pattern"], value) is not None)
        if type(value) in (int, float):
            require(schema.get("minimum", float("-inf")) <= value <= schema.get("maximum", float("inf")))
            require(schema.get("exclusiveMinimum", float("-inf")) < value < schema.get("exclusiveMaximum", float("inf")))
