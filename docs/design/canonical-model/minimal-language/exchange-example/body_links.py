"""CommonMark body-link extension for the bounded design probe, not Go runtime."""

from __future__ import annotations

from importlib.metadata import version
from typing import Any

from markdown_it import MarkdownIt
from markdown_it.rules_inline.image import image

import diagnostic_coordinates as coordinate_profile
from name_profile import WHITE_SPACE

MARKDOWN_IT_VERSION = "4.0.0"
MAX_BODY_BYTES = 2_000_000
MAX_NESTING = 64


class BodyLinkLimitError(ValueError):
    pass


class BodyLinkSyntaxError(ValueError):
    def __init__(self, message: str, byte_start: int | None = None):
        super().__init__(message if byte_start is None else f"{message}; body byte {byte_start}")
        self.byte_start = byte_start


def _normalize(text: str, *, coordinates: bool) -> tuple[str, list[int]]:
    result, positions = [], []
    index = 0
    while index < len(text):
        if coordinates:
            positions.append(index)
        char = text[index]
        result.append("\n" if char == "\r" else char)
        index += 2 if text[index:index + 2] == "\r\n" else 1
    if coordinates:
        positions.append(len(text))
    return "".join(result), positions


def _literal_inline(content, line_map, normalized_lines, literal_lines):
    """Recover literal characters for the extension, independently of diagnostics.

    CommonMark context rules still see U+FFFD in place of NUL. Both streams
    have the same length; only the model-link rule reads the literal stream.
    """
    if literal_lines is None or "\ufffd" not in content:
        return content
    row, end_row = line_map
    parts = []
    for part in content.split("\n"):
        while row < end_row:
            column = normalized_lines[row].find(part)
            if column >= 0:
                break
            row += 1
        if row >= end_row:
            raise BodyLinkLimitError("body-link probe capability limit: cannot recover literal inline text containing NUL")
        parts.append(literal_lines[row][column:column + len(part)])
        row += 1
    return "\n".join(parts)


def _link_at(text: str, start: int, end: int) -> tuple[int, str, str | None]:
    index = start + 2
    target, label = [], []
    has_label = False
    while index < end:
        char = text[index]
        if text.startswith("]]", index):
            name = "".join(target)
            display = "".join(label) if has_label else None
            if not name or name[0] in WHITE_SPACE or name[-1] in WHITE_SPACE:
                raise ValueError("body link requires an exact nonempty target without surrounding whitespace")
            if display is not None and (not display or all(char in WHITE_SPACE for char in display)):
                raise ValueError("body link display text must not be empty or whitespace-only")
            return index + 2, name, display
        if char in "\r\n\x85\u2028\u2029":
            raise ValueError("body links cannot contain line breaks")
        if char == "|":
            if has_label:
                raise ValueError("body link contains an additional unescaped separator")
            has_label = True
            index += 1
            continue
        if char == "\\":
            if not has_label or index + 1 >= end or text[index + 1] not in "|[]\\":
                raise ValueError("invalid body-link escape; escapes are permitted only in display text")
            label.append(text[index + 1])
            index += 2
            continue
        if char in "[]":
            raise ValueError("body links cannot be nested; escape brackets in display text")
        (label if has_label else target).append(char)
        index += 1
    raise ValueError("unclosed active body link")


def parse_body_links(body: str, *, coordinates: bool = False) -> list[dict[str, Any]]:
    if version("markdown-it-py") != MARKDOWN_IT_VERSION:
        raise BodyLinkLimitError(f"body-link probe requires markdown-it-py {MARKDOWN_IT_VERSION}")
    if len(body.encode("utf-8")) > MAX_BODY_BYTES:
        raise BodyLinkLimitError(f"body-link probe resource limit: body exceeds {MAX_BODY_BYTES} UTF-8 bytes")
    # The pinned parser trims Python whitespace in a few CommonMark contexts.
    # Stop before using its context decisions on affected input (upstream #418).
    if any(char in body for char in "\x1c\x1d\x1e\x1f\x85"):
        raise BodyLinkLimitError("body-link probe capability limit: pinned parser whitespace handling for U+001C..U+001F/U+0085")
    literal, original_positions = _normalize(body, coordinates=coordinates)
    normalized = literal.replace("\0", "\ufffd")
    literal_lines = literal.split("\n") if "\0" in literal else None
    normalized_lines = normalized.split("\n") if literal_lines is not None else None
    byte_positions, total = [], 0
    if coordinates:
        for char in body:
            byte_positions.append(total)
            total += len(char.encode("utf-8"))
        byte_positions.append(total)
    records: list[dict[str, Any]] = []
    md = MarkdownIt("commonmark", {"maxNesting": MAX_NESTING + 8})

    def block_limit(state, start_line, end_line, silent):
        if state.level >= MAX_NESTING:
            raise BodyLinkLimitError(f"body-link probe resource limit: nesting reaches {MAX_NESTING}")
        return False

    def inline_limit(state, silent):
        if state.level >= MAX_NESTING:
            raise BodyLinkLimitError(f"body-link probe resource limit: nesting reaches {MAX_NESTING}")
        return False

    def model_link(state, silent):
        if silent or state.linkLevel > 0 or state.env.get("_inside_image") or not state.src.startswith("[[", state.pos):
            return False
        start = state.pos
        try:
            end, target, display = _link_at(state.env["_literal_inline"], start, state.posMax)
        except ValueError as error:
            offset = None
            if coordinates and state.env.get("_inline_positions") is not None:
                try:
                    original = original_positions[state.env["_inline_positions"][start]]
                    offset = byte_positions[original]
                except (IndexError, TypeError):
                    pass
            raise BodyLinkSyntaxError(str(error), offset) from error
        record = {"target": target, "display": display}
        if coordinates:
            try:
                positions = state.env.get("_inline_positions")
                if positions is None:
                    raise coordinate_profile.CoordinateMappingError(state.env.get("_coordinate_issue", "inline source mapping is unavailable"))
                original_start = original_positions[positions[start]]
                original_end = original_positions[positions[end - 1] + 1]
                record.update(startByte=byte_positions[original_start], endByte=byte_positions[original_end])
            except (ValueError, IndexError, TypeError, RecursionError, MemoryError) as error:
                record["coordinate_issue"] = str(error) or type(error).__name__
        records.append(record)
        token = state.push("model_link", "", 0)
        token.content = display if display is not None else target
        state.pos = end
        return True

    def protected_image(state, silent):
        previous = state.env.get("_inside_image", False)
        state.env["_inside_image"] = True
        try:
            return image(state, silent)
        finally:
            state.env["_inside_image"] = previous

    def inlines(state):
        for token in state.tokens:
            if token.type == "inline":
                state.env["_literal_inline"] = _literal_inline(token.content, token.map, normalized_lines, literal_lines)
                if coordinates:
                    try:
                        state.env["_inline_positions"] = coordinate_profile.inline_positions(token.content, token.map, state.src)
                        state.env.pop("_coordinate_issue", None)
                    except (ValueError, KeyError, IndexError, TypeError, RecursionError, MemoryError) as error:
                        state.env["_inline_positions"] = None
                        state.env["_coordinate_issue"] = str(error) or type(error).__name__
                token.children = []
                state.md.inline.parse(token.content, state.md, state.env, token.children)

    md.block.ruler.before("code", "probe_limit", block_limit)
    md.inline.ruler.before("text", "probe_limit", inline_limit)
    md.inline.ruler.before("link", "model_link", model_link)
    md.inline.ruler.at("image", protected_image)
    md.core.ruler.at("inline", inlines)
    try:
        md.parse(normalized, {})
    except RecursionError as error:
        raise BodyLinkLimitError("body-link probe resource limit: parser recursion") from error
    return records
