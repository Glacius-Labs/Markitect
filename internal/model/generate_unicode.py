#!/usr/bin/env python3
"""Generate the pinned Unicode 15.1 tables and source corpus constants."""

from __future__ import annotations

import argparse
import base64
import gzip
import hashlib
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parent
DATA = ROOT / "testdata" / "unicode-15.1.0"
TABLES = ROOT / "unicode_tables.go"
CORPUS = ROOT / "unicode_corpus_test.go"

SOURCES = {
    "UnicodeData.txt": "2fc713e6a31a87c4850a37fe2caffa4218180fadb5de86b43a143ddb4581fb86",
    "CaseFolding.txt": "4e55acfdc32825a22e87670e9056a3bf94ad7c5400065778e9e10f8314372bcf",
    "DerivedCoreProperties.txt": "f55d0db69123431a7317868725b1fcbf1eab6b265d756d1bd7f0f6d9f9ee108b",
    "DerivedNormalizationProps.txt": "8875dccee2bc1a7c1fe568a3b502a9e78c9e0495afd96b6568b4294d0ed1f7e1",
    "NormalizationTest.txt": "871238e37e3be0696ec2bd0891119a041b052da1a84485eda05a5438724b223e",
}
LICENSE_SHA256 = "e7a93b009565cfce55919a381437ac4db883e9da2126fa28b91d12732bc53d96"


def read_pinned(name: str) -> tuple[bytes, bytes]:
    path = DATA / (name + ".gz")
    compressed = path.read_bytes()
    if len(compressed) < 10 or compressed[:3] != b"\x1f\x8b\x08":
        raise ValueError(f"{path}: not a gzip stream")
    if compressed[3] & 0x08 or compressed[4:8] != b"\0\0\0\0":
        raise ValueError(f"{path}: gzip must have an empty filename and mtime zero")
    data = gzip.decompress(compressed)
    digest = hashlib.sha256(data).hexdigest()
    if digest != SOURCES[name]:
        raise ValueError(f"{path}: SHA-256 {digest}, expected {SOURCES[name]}")
    return data, compressed


def license_notice() -> str:
    data = (DATA / "UNICODE-LICENSE-V3.txt").read_bytes()
    digest = hashlib.sha256(data).hexdigest()
    if digest != LICENSE_SHA256:
        raise ValueError(f"Unicode license SHA-256 {digest}, expected {LICENSE_SHA256}")
    notice = data.decode("utf-8").rstrip("\n")
    if "*/" in notice:
        raise ValueError("license notice cannot be represented as a Go block comment")
    return "/*\n" + notice + "\n*/"


def codepoints(field: str) -> tuple[int, int]:
    parts = field.strip().split("..")
    lo = int(parts[0], 16)
    return lo, int(parts[1], 16) if len(parts) == 2 else lo


def merge_ranges(points: list[tuple[int, int]]) -> list[tuple[int, int]]:
    if not points:
        return []
    points.sort()
    merged = [list(points[0])]
    for lo, hi in points[1:]:
        if lo <= merged[-1][1] + 1:
            merged[-1][1] = max(merged[-1][1], hi)
        else:
            merged.append([lo, hi])
    return [(lo, hi) for lo, hi in merged]


def parse_property_file(data: bytes, wanted: set[str]) -> dict[str, list[tuple[int, int]]]:
    result = {name: [] for name in wanted}
    for raw in data.decode("utf-8").splitlines():
        body = raw.split("#", 1)[0].strip()
        if not body:
            continue
        fields = [part.strip() for part in body.split(";")]
        if len(fields) >= 2 and fields[1] in wanted:
            result[fields[1]].append(codepoints(fields[0]))
    return {name: merge_ranges(ranges) for name, ranges in result.items()}


def parse_unicode_data(data: bytes):
    category_ranges: dict[str, list[tuple[int, int]]] = {}
    decomposition: dict[int, str] = {}
    combining: dict[int, int] = {}
    pending = None
    for raw in data.decode("utf-8").splitlines():
        fields = raw.split(";")
        cp = int(fields[0], 16)
        name, category = fields[1], fields[2]
        ccc = int(fields[3])
        if name.endswith(", First>"):
            pending = (cp, category, ccc)
            continue
        if name.endswith(", Last>"):
            if pending is None or pending[1] != category or pending[2] != ccc:
                raise ValueError(f"malformed UnicodeData range at {raw}")
            start, _, range_ccc = pending
            category_ranges.setdefault(category, []).append((start, cp))
            if range_ccc:
                for value in range(start, cp + 1):
                    combining[value] = range_ccc
            pending = None
            continue
        category_ranges.setdefault(category, []).append((cp, cp))
        if ccc:
            combining[cp] = ccc
        mapping = fields[5]
        if mapping and not mapping.startswith("<"):
            decomposition[cp] = "".join(chr(int(item, 16)) for item in mapping.split())
    if pending is not None:
        raise ValueError("unterminated UnicodeData First range")
    return category_ranges, decomposition, combining


def parse_case_folding(data: bytes) -> dict[int, str]:
    mappings: dict[int, tuple[str, str]] = {}
    for raw in data.decode("utf-8").splitlines():
        body = raw.split("#", 1)[0].strip()
        if not body:
            continue
        fields = [part.strip() for part in body.split(";")]
        cp, status = int(fields[0], 16), fields[1]
        if status in ("C", "F"):
            mapping = "".join(chr(int(item, 16)) for item in fields[2].split())
            previous = mappings.get(cp)
            if previous is None or status == "F":
                mappings[cp] = (status, mapping)
    return {cp: value for cp, (_, value) in mappings.items()}


def parse_full_composition_exclusion(data: bytes) -> set[int]:
    return {
        cp
        for lo, hi in parse_property_file(data, {"Full_Composition_Exclusion"})[
            "Full_Composition_Exclusion"
        ]
        for cp in range(lo, hi + 1)
    }


def go_string(value: str) -> str:
    escapes = {
        "\\": "\\\\",
        '"': '\\"',
        "\a": "\\a",
        "\b": "\\b",
        "\f": "\\f",
        "\n": "\\n",
        "\r": "\\r",
        "\t": "\\t",
        "\v": "\\v",
    }
    escaped = []
    for char in value:
        if char in escapes:
            escaped.append(escapes[char])
        elif 0x20 <= ord(char) < 0x7F:
            escaped.append(char)
        elif ord(char) <= 0xFFFF:
            escaped.append(f"\\u{ord(char):04x}")
        else:
            escaped.append(f"\\U{ord(char):08x}")
    return '"' + "".join(escaped) + '"'


def emit_ranges(name: str, ranges: list[tuple[int, int]]) -> str:
    rows = [f"var {name} = []ucdRange{{"]
    rows.extend(f"\t{{0x{lo:X}, 0x{hi:X}}}," for lo, hi in ranges)
    rows.append("}")
    return "\n".join(rows)


def emit_map(name: str, typ: str, values: dict[int, object], value_fmt) -> str:
    rows = [f"var {name} = map[rune]{typ}{{"]
    rows.extend(
        f"\t0x{cp:X}: {value_fmt(value)}," for cp, value in sorted(values.items())
    )
    rows.append("}")
    return "\n".join(rows)


def generate_tables(inputs: dict[str, bytes]) -> bytes:
    core = parse_property_file(
        inputs["DerivedCoreProperties.txt"], {"XID_Start", "XID_Continue"}
    )
    categories, decomposition, combining = parse_unicode_data(inputs["UnicodeData.txt"])
    normalization = parse_property_file(
        inputs["DerivedNormalizationProps.txt"], {"Full_Composition_Exclusion"}
    )
    excluded = {cp for lo, hi in normalization["Full_Composition_Exclusion"] for cp in range(lo, hi + 1)}
    composition = {}
    for cp, sequence in decomposition.items():
        parts = list(map(ord, sequence))
        if len(parts) == 2 and cp not in excluded:
            composition[(parts[0] << 21) | parts[1]] = cp

    def category_set(prefix: str) -> list[tuple[int, int]]:
        return merge_ranges(
            [interval for category, ranges in categories.items() if category.startswith(prefix) for interval in ranges]
        )

    props = {
        "ucdXIDStart": core["XID_Start"],
        "ucdXIDContinue": core["XID_Continue"],
        "ucdLowercase": parse_property_file(inputs["DerivedCoreProperties.txt"], {"Lowercase"})["Lowercase"],
        "ucdUppercase": parse_property_file(inputs["DerivedCoreProperties.txt"], {"Uppercase"})["Uppercase"],
        "ucdTitlecase": category_set("Lt"),
        "ucdMarks": category_set("M"),
        "ucdDecimalDigits": category_set("Nd"),
        "ucdConnectors": category_set("Pc"),
        "ucdLetters": category_set("L"),
    }
    lines = [
        "// Code generated by generate_unicode.py; DO NOT EDIT.",
        "// Unicode 15.1.0 data derived from the pinned UCD inputs in testdata/unicode-15.1.0.",
        "// Generated tables are part of the Unicode Data Files and carry this notice:",
        license_notice(),
        "package model",
        "",
        'const unicodeVersion = "15.1.0"',
        "",
        "// unicodeSourceSHA256 binds each table input to the checked-in UCD snapshot.",
        "var unicodeSourceSHA256 = map[string]string{",
    ]
    for name, digest in SOURCES.items():
        lines.append(f'\t"{name}": "{digest}",')
    lines.append(f'\t"UNICODE-LICENSE-V3.txt": "{LICENSE_SHA256}",')
    lines += ["}", ""]
    lines += [emit_ranges(name, ranges) for name, ranges in props.items()]
    lines.append(emit_map("ucdDecomposition", "string", decomposition, go_string))
    lines.append(emit_map("ucdCombiningClass", "uint8", combining, lambda x: str(x)))
    lines += [
        "var ucdComposition = map[uint64]rune{",
        *[f"\t0x{key:X}: 0x{value:X}," for key, value in sorted(composition.items())],
        "}",
        emit_map("ucdCaseFold", "string", parse_case_folding(inputs["CaseFolding.txt"]), go_string),
    ]
    return ("\n\n".join(lines) + "\n").encode("utf-8")


def generate_corpus(compressed_inputs: dict[str, bytes]) -> bytes:
    names = [
        ("unicodeNormalizationTestGzipBase64", "NormalizationTest.txt"),
        ("unicodeCaseFoldingTestGzipBase64", "CaseFolding.txt"),
        ("unicodeDerivedCorePropertiesTestGzipBase64", "DerivedCoreProperties.txt"),
        ("unicodeUnicodeDataTestGzipBase64", "UnicodeData.txt"),
    ]
    lines = [
        "// Code generated by generate_unicode.py; DO NOT EDIT.",
        "// Exact, compressed UCD inputs for offline conformance tests.",
        "// Unicode 15.1.0 data derived from the pinned UCD inputs in testdata/unicode-15.1.0.",
        "// This copied data carries the Unicode Data Files notice:",
        license_notice(),
        "package model",
        "",
        'import "bytes"',
        'import "compress/gzip"',
        'import "encoding/base64"',
        'import "io"',
        "",
    ]
    for const, name in names:
        payload = base64.b64encode(compressed_inputs[name]).decode("ascii")
        lines.append(f"const {const} =")
        chunks = [payload[index : index + 100] for index in range(0, len(payload), 100)]
        lines.extend(f'\t"{chunk}"' + (" +" if index + 1 < len(chunks) else "") for index, chunk in enumerate(chunks))
        lines.append("")
    lines += [
        "// decodeUnicodeTestCorpus returns the exact pinned source file for tests.",
        "func decodeUnicodeTestCorpus(encoded string) []byte {",
        "\tcompressed, err := base64.StdEncoding.DecodeString(encoded)",
        "\tif err != nil {",
        '\t\tpanic("invalid generated Unicode corpus encoding: " + err.Error())',
        "\t}",
        "\treader, err := gzip.NewReader(bytes.NewReader(compressed))",
        "\tif err != nil {",
        '\t\tpanic("invalid generated Unicode corpus gzip: " + err.Error())',
        "\t}",
        "\tdefer reader.Close()",
        "\tdata, err := io.ReadAll(reader)",
        "\tif err != nil {",
        '\t\tpanic("invalid generated Unicode corpus payload: " + err.Error())',
        "\t}",
        "\treturn data",
        "}",
    ]
    return ("\n".join(lines) + "\n").encode("utf-8")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true", help="fail if generated files differ")
    args = parser.parse_args()
    try:
        pinned = {name: read_pinned(name) for name in SOURCES}
        inputs = {name: pair[0] for name, pair in pinned.items()}
        compressed_inputs = {name: pair[1] for name, pair in pinned.items()}
        license_notice()  # Validate the vendored license even though it is already in output.
        expected = {
            TABLES: generate_tables(inputs),
            CORPUS: generate_corpus(compressed_inputs),
        }
        changed = []
        for path, data in expected.items():
            formatted = subprocess.run(
                ["gofmt"], input=data, capture_output=True, check=False
            )
            if formatted.returncode != 0:
                raise ValueError(f"gofmt failed for {path.name}: {formatted.stderr.decode().strip()}")
            data = formatted.stdout
            if args.check:
                if not path.exists() or path.read_bytes() != data:
                    changed.append(path.name)
            else:
                path.write_bytes(data)
        if changed:
            print("generated Unicode output differs: " + ", ".join(changed), file=sys.stderr)
            return 1
        if not args.check:
            print("generated " + ", ".join(path.name for path in expected))
    except (OSError, UnicodeError, ValueError) as error:
        print(f"Unicode generation failed: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
