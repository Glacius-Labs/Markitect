"""Optional diagnostic coordinates; never part of canonical model validation."""
from __future__ import annotations

from typing import Any


class CoordinateMappingError(ValueError):
    pass


def inline_positions(content: str, line_map: list[int], source: str) -> list[int]:
    """Map retained inline characters through CommonMark container prefixes."""
    lines = source.split("\n")
    starts, total = [], 0
    for line in lines:
        starts.append(total)
        total += len(line) + 1
    row, end_row = line_map
    positions = []
    parts = content.split("\n")
    for part_index, part in enumerate(parts):
        while row < end_row:
            column = lines[row].find(part)
            if column >= 0:
                break
            row += 1
        if row >= end_row:
            raise CoordinateMappingError("body-link probe cannot prove the inline source mapping")
        positions.extend(range(starts[row] + column, starts[row] + column + len(part)))
        if part_index < len(parts) - 1:
            positions.append(starts[row] + len(lines[row]))
        row += 1
    return positions


def yaml_body_source(text: str, node: Any, decoded: str, file: str) -> dict[str, Any]:
    """Prove non-whitespace scalar origins, sufficient for complete link spans.

    YAML folding/chomping changes whitespace only; flow escapes are decoded with
    their entire original span. Never guess if scalar and source do not agree.
    """
    start, end, style = node["start"], node["end"], node["style"]
    atoms = []
    if style in {"|", ">"}:
        newline = text.find("\n", start, end)
        if newline < 0:
            newline = text.find("\r", start, end)
        index = end if newline < 0 else newline + 1
        stop = end
    elif style in {"'", '"'}:
        index, stop = start + 1, end - 1
    else:
        index, stop = start, end
    replacements = {"0": "\0", "a": "\a", "b": "\b", "t": "\t", "\t": "\t", "n": "\n", "v": "\v",
                    "f": "\f", "r": "\r", "e": "\x1b", " ": " ", '"': '"', "/": "/", "\\": "\\",
                    "N": "\x85", "_": "\xa0", "L": "\u2028", "P": "\u2029"}
    while index < stop:
        origin = index
        char = text[index]
        index += 1
        if style == "'" and char == "'" and text[index:index + 1] == "'":
            index += 1
        elif style == '"' and char == "\\":
            marker = text[index]
            index += 1
            if marker in "\r\n":
                continue
            if marker in {"x", "u", "U"}:
                count = {"x": 2, "u": 4, "U": 8}[marker]
                char = chr(int(text[index:index + count], 16))
                index += count
            else:
                char = replacements[marker]
        if not char.isspace():
            atoms.append((char, origin, index))
    expected = [(char, index) for index, char in enumerate(decoded) if not char.isspace()]
    if "".join(char for char, _, _ in atoms) != "".join(char for char, _ in expected):
        raise CoordinateMappingError("body-link probe cannot prove the YAML scalar source mapping")
    spans = {decoded_index: (origin_start, origin_end)
             for (_, decoded_index), (_, origin_start, origin_end) in zip(expected, atoms)}
    return {"file": file, "text": text, "decoded": decoded, "spans": spans}


def attach_yaml_source(record: dict[str, Any], source: dict[str, Any]) -> dict[str, Any]:
    decoded = source["decoded"]
    prefix = decoded.encode("utf-8")
    start = len(prefix[:record["startByte"]].decode("utf-8"))
    end = len(prefix[:record["endByte"]].decode("utf-8"))
    first = source["spans"][start][0]
    last = source["spans"][end - 1][1]
    text = source["text"]

    def position(index):
        # Source lines/columns are one-based; columns count Unicode code points.
        normalized = text[:index].replace("\r\n", "\n").replace("\r", "\n")
        return {"line": normalized.count("\n") + 1, "column": len(normalized.rsplit("\n", 1)[-1]) + 1}

    return {**record, "source": {"file": source["file"], "start": position(first), "end": position(last)}}


def with_yaml_coordinates(records: list[dict[str, Any]], source: dict | None, body: str) -> list[dict[str, Any]]:
    """Attach proven positions or report their absence, without inventing a span."""
    if not records:
        return records
    try:
        if not source or source.get("decoded") != body:
            raise CoordinateMappingError("matching YAML source metadata is unavailable")
        origins = yaml_body_source(source["text"], source["node"], body, source["file"])
    except (ValueError, KeyError, IndexError, TypeError, RecursionError, MemoryError) as error:
        return [{**record, "coordinate_issue": record.get("coordinate_issue") or str(error) or type(error).__name__}
                for record in records]
    located = []
    for record in records:
        if "startByte" not in record or "endByte" not in record:
            located.append({**record, "coordinate_issue": record.get("coordinate_issue") or "body byte range is unavailable"})
            continue
        try:
            located.append(attach_yaml_source(record, origins))
        except (ValueError, KeyError, IndexError, TypeError, RecursionError, MemoryError) as error:
            located.append({**record, "coordinate_issue": str(error) or type(error).__name__})
    return located
