#!/usr/bin/env python3
"""Check local Markdown links in the repository's maintained documentation.

This intentionally supports a bounded Markdown subset: ATX and Setext headings,
explicit HTML id attributes, inline links/images, reference definitions and
full, collapsed, or defined shortcut references. It skips fenced and inline
code, HTML comments, and external URLs. It does not attempt full CommonMark or
HTML parsing. Destinations may be angle-bracketed or bare with balanced
parentheses and may have a Markdown title. Local paths are case-checked even on
case-insensitive filesystems, percent-decoded, and confined to the repository.
"""

from __future__ import annotations

import argparse
import html
from html.parser import HTMLParser
import os
from pathlib import Path
import re
import sys
from urllib.parse import unquote


DEFAULT_DOCUMENTS = (
    "README.md",
    "AGENTS.md",
    "CONTRIBUTING.md",
    "SUPPORT.md",
    "SECURITY.md",
    "docs/README.md",
    "docs/project-workflow.md",
    "docs/project-operations.md",
    "docs/provider-adapters.md",
    "docs/architecture.md",
    "docs/repository-layout.md",
    "docs/implementation-plan.md",
    "docs/vision.md",
    "docs/operating-methodology.md",
    "docs/measurement.md",
    "docs/operations.md",
    "docs/development/README.md",
    "docs/development/parallel-work.md",
    "docs/development/documentation.md",
    "docs/design/README.md",
    "docs/validation/README.md",
    "docs/history/README.md",
    "docs/work-items/README.md",
    "docs/workstreams/README.md",
    "docs/research/README.md",
    "docs/strategy/README.md",
    "docs/design/project-world/README.md",
    "docs/work-items/product-readiness/README.md",
    "examples/README.md",
    "experiments/README.md",
    "integration/README.md",
)


def _strip_fenced(text: str) -> str:
    """Blank fenced blocks while preserving source lines."""
    lines = []
    fence_char = None
    fence_size = 0
    for line in text.splitlines():
        match = re.match(r"^\s{0,3}(`{3,}|~{3,})", line)
        if fence_char is not None:
            closes_fence = (
                match is not None
                and match.group(1)[0] == fence_char
                and len(match.group(1)) >= fence_size
                and not line[match.end():].strip()
            )
            if closes_fence:
                fence_char = None
            lines.append("")
            continue
        if match:
            fence_char = match.group(1)[0]
            fence_size = len(match.group(1))
            lines.append("")
            continue
        lines.append(line)
    return "\n".join(lines)


def _find_matching_backtick_run(text: str, start: int, run_length: int) -> tuple[int, int] | None:
    """Find a closing delimiter whose whole backtick run has the requested length."""
    pos = start
    while pos < len(text):
        found = text.find("`", pos)
        if found < 0:
            return None
        end = found
        while end < len(text) and text[end] == "`":
            end += 1
        if end - found == run_length:
            return found, end
        pos = end
    return None


def _strip_comments_around_inline_code(text: str, preserve_inline_code: bool) -> str:
    """Strip HTML comments outside code spans; optionally keep code-span text."""
    out = []
    i = 0
    in_comment = False
    while i < len(text):
        if in_comment:
            end = text.find("-->", i)
            stop = len(text) if end < 0 else end + 3
            span = text[i:stop]
            out.append("".join("\n" if char == "\n" else " " for char in span))
            i = stop
            in_comment = end < 0
            continue
        if text.startswith("<!--", i):
            in_comment = True
            continue
        if text[i] == "`" and (i == 0 or text[i - 1] != "\\"):
            end_run = i
            while end_run < len(text) and text[end_run] == "`":
                end_run += 1
            run = text[i:end_run]
            closing = _find_matching_backtick_run(text, end_run, len(run))
            if closing is not None:
                close, stop = closing
                span = text[i:stop]
                out.append(span if preserve_inline_code else "".join("\n" if char == "\n" else " " for char in span))
                i = stop
                continue
        out.append(text[i])
        i += 1
    return "".join(out)


def _strip_inline_code(text: str) -> str:
    """Blank inline code spans while preserving every newline and line number."""
    out = []
    i = 0
    while i < len(text):
        if text[i] != "`" or (i and text[i - 1] == "\\"):
            out.append(text[i])
            i += 1
            continue
        end_run = i
        while end_run < len(text) and text[end_run] == "`":
            end_run += 1
        run = text[i:end_run]
        closing = _find_matching_backtick_run(text, end_run, len(run))
        if closing is None:
            out.append(run)
            i = end_run
            continue
        close, stop = closing
        span = text[i:stop]
        out.append("".join("\n" if char == "\n" else " " for char in span))
        i = stop
    return "".join(out)


def _strip_code_and_comments(text: str) -> str:
    """Blank fenced and inline code plus comments, preserving source lines."""
    return _strip_comments_around_inline_code(_strip_fenced(text), preserve_inline_code=False)


def _reference_key(label: str) -> str:
    return " ".join(label.split()).casefold()


def _destination_after_open(text: str, pos: int) -> str | None:
    """Read a destination beginning just after `(`; titles are ignored."""
    while pos < len(text) and text[pos].isspace():
        pos += 1
    if pos >= len(text):
        return None
    if text[pos] == "<":
        end = pos + 1
        while end < len(text):
            if text[end] == ">" and (end == pos + 1 or text[end - 1] != "\\"):
                return text[pos + 1:end]
            end += 1
        return None

    start = pos
    depth = 0
    escaped = False
    while pos < len(text):
        char = text[pos]
        if escaped:
            escaped = False
        elif char == "\\":
            escaped = True
        elif char == "(":
            depth += 1
        elif char == ")":
            if depth == 0:
                break
            depth -= 1
        elif char.isspace() and depth == 0:
            break
        pos += 1
    return text[start:pos]


def extract_destinations(markdown: str) -> list[tuple[str, int]]:
    """Return (destination, one-based line) pairs for the supported syntax."""
    visible = _strip_code_and_comments(markdown)
    lines = visible.splitlines()
    refs: dict[str, tuple[str, int]] = {}
    definition_re = re.compile(r"^\s{0,3}\[([^\]]+)\]:\s*(?:<([^>\n]*)>|(\S+))")
    for line_no, line in enumerate(lines, 1):
        match = definition_re.match(line)
        if match:
            destination = match.group(2) if match.group(2) is not None else match.group(3)
            refs.setdefault(_reference_key(match.group(1)), (destination, line_no))

    found: list[tuple[str, int]] = []
    consumed: list[list[tuple[int, int]]] = [[] for _ in lines]
    for line_index, line in enumerate(lines):
        line_no = line_index + 1
        definition = definition_re.match(line)
        if definition:
            destination = definition.group(2) if definition.group(2) is not None else definition.group(3)
            found.append((destination, line_no))
            consumed[line_index].append((definition.start(), definition.end()))

        # Inline links and images. The `](` opener is sufficient for this
        # deliberately bounded parser; nested label brackets are unsupported.
        for match in re.finditer(r"\]\s*\(", line):
            open_paren = line.find("(", match.start(), match.end())
            destination = _destination_after_open(line, open_paren + 1)
            if destination is not None:
                found.append((destination, line_no))

        # Full and collapsed references.
        for match in re.finditer(r"!?\[([^\]]+)\]\s*\[([^\]]*)\]", line):
            label = match.group(2) or match.group(1)
            ref = refs.get(_reference_key(label))
            if ref:
                found.append((ref[0], line_no))
                consumed[line_index].append(match.span())

    # Reference definitions also have destinations; add any not yet represented.
    # Shortcut references are recognized only when their label is defined.
    for line_index, line in enumerate(lines):
        line_no = line_index + 1
        spans = consumed[line_index]
        for match in re.finditer(r"!?\[([^\]]+)\]", line):
            if any(start <= match.start() < end for start, end in spans):
                continue
            following = line[match.end():].lstrip()
            if following.startswith(("(", "[")):
                continue
            ref = refs.get(_reference_key(match.group(1)))
            if ref:
                found.append((ref[0], line_no))

    # Preserve repeated references because each source location is actionable.
    return found


def _plain_heading(text: str) -> str:
    # Code-span delimiters do not contribute to heading ids, but their content
    # does. Match runs of any backtick length so embedded single ticks survive.
    out = []
    i = 0
    while i < len(text):
        if text[i] != "`":
            out.append(text[i])
            i += 1
            continue
        end = i
        while end < len(text) and text[end] == "`":
            end += 1
        run = text[i:end]
        closing = _find_matching_backtick_run(text, end, len(run))
        if closing is None:
            out.append(run)
            i = end
            continue
        close, stop = closing
        out.append(text[end:close])
        i = stop
    text = "".join(out)
    text = re.sub(r"!?\[([^\]]*)\]\([^)]*\)", r"\1", text)
    text = re.sub(r"!?\[([^\]]*)\]\[[^]]*\]", r"\1", text)
    text = re.sub(r"<[^>]+>", "", text)
    return html.unescape(text).strip().strip("#").strip()


def _explicit_html_ids(text: str) -> set[str]:
    """Read id attributes from simple HTML tags, never from prose or data-id."""
    class IdCollector(HTMLParser):
        def __init__(self) -> None:
            super().__init__(convert_charrefs=True)
            self.ids: set[str] = set()

        def _collect(self, attrs: list[tuple[str, str | None]]) -> None:
            self.ids.update(value for name, value in attrs if name == "id" and value is not None)

        def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
            self._collect(attrs)

        def handle_startendtag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
            self._collect(attrs)

    collector = IdCollector()
    collector.feed(text)
    collector.close()
    return collector.ids


def heading_anchors(markdown: str) -> set[str]:
    """Compute GitHub-style anchors for supported headings plus explicit ids."""
    visible = _strip_comments_around_inline_code(_strip_fenced(markdown), preserve_inline_code=True)
    lines = visible.splitlines()
    result = _explicit_html_ids(_strip_inline_code(visible))
    counts: dict[str, int] = {}
    for i, line in enumerate(lines):
        atx = re.match(r"^\s{0,3}#{1,6}\s+(.+?)\s*#*\s*$", line)
        title = atx.group(1) if atx else None
        if title is None and i + 1 < len(lines) and re.match(r"^\s{0,3}(?:=+|-+)\s*$", lines[i + 1]):
            if line.strip():
                title = line.strip()
        if title is None:
            continue
        raw = _plain_heading(title).lower()
        slug = "".join(ch for ch in raw if ch.isalnum() or ch in " _-")
        # GitHub-style heading ids replace each remaining whitespace character
        # with a hyphen. Punctuation removal can therefore leave repeated `-`.
        slug = re.sub(r"\s", "-", slug).strip("-")
        if not slug:
            continue
        suffix = counts.get(slug, 0)
        candidate = slug if suffix == 0 else f"{slug}-{suffix}"
        while candidate in result:
            suffix += 1
            candidate = f"{slug}-{suffix}"
        counts[slug] = suffix + 1
        result.add(candidate)
    return result


def _has_bad_percent_escape(value: str) -> bool:
    return re.search(r"%(?![0-9a-fA-F]{2})", value) is not None


class DocumentationChecker:
    def __init__(self, repo: Path):
        self.repo = repo.resolve()
        self.errors: list[str] = []
        self._anchors: dict[Path, set[str]] = {}

    def error(self, source: Path, line: int, message: str) -> None:
        try:
            source_label = source.resolve().relative_to(self.repo).as_posix()
        except ValueError:
            source_label = str(source)
        self.errors.append(f"{source_label}:{line}: {message}")

    def _case_checked_path(self, source: Path, line: int, raw_path: str) -> Path | None:
        if _has_bad_percent_escape(raw_path):
            self.error(source, line, f"malformed percent escape in local path {raw_path!r}")
            return None
        decoded = unquote(raw_path)
        if "\\" in decoded:
            self.error(source, line, f"backslash in local URL path {raw_path!r}; use forward slashes")
            return None
        if re.match(r"^[A-Za-z]:", decoded):
            self.error(source, line, f"absolute drive path is outside repository: {raw_path!r}")
            return None
        parts = decoded.split("/")
        current = self.repo if decoded.startswith("/") else source.parent.resolve()
        try:
            current.relative_to(self.repo)
        except ValueError:
            self.error(source, line, f"source path is outside repository: {source}")
            return None

        for part in parts:
            if part in ("", "."):
                continue
            if part == "..":
                current = current.parent
                try:
                    current.relative_to(self.repo)
                except ValueError:
                    self.error(source, line, f"local link escapes repository: {raw_path!r}")
                    return None
                continue
            try:
                safe_parent = current.resolve(strict=True)
                safe_parent.relative_to(self.repo)
            except (OSError, ValueError):
                self.error(source, line, f"local link escapes repository: {raw_path!r}")
                return None
            try:
                entries = os.listdir(safe_parent)
            except OSError:
                self.error(source, line, f"cannot inspect local link parent {current}")
                return None
            if part not in entries:
                case_matches = [name for name in entries if name.casefold() == part.casefold()]
                if case_matches:
                    self.error(source, line, f"path case mismatch for {raw_path!r}: found {case_matches[0]!r}, wrote {part!r}")
                else:
                    self.error(source, line, f"missing local target {raw_path!r}")
                return None
            current = current / part

        try:
            resolved = current.resolve(strict=True)
            resolved.relative_to(self.repo)
        except (OSError, ValueError):
            self.error(source, line, f"local target is missing or escapes repository: {raw_path!r}")
            return None
        return resolved

    def _anchors_for(self, path: Path) -> set[str]:
        try:
            path = path.resolve(strict=True)
            path.relative_to(self.repo)
        except (OSError, ValueError):
            return set()
        if path not in self._anchors:
            try:
                self._anchors[path] = heading_anchors(path.read_text(encoding="utf-8"))
            except (OSError, UnicodeError):
                self._anchors[path] = set()
        return self._anchors[path]

    def check_file(self, source: Path) -> None:
        try:
            source_resolved = source.resolve(strict=True)
            source_resolved.relative_to(self.repo)
            text = source_resolved.read_text(encoding="utf-8")
        except (OSError, UnicodeError, ValueError):
            self.error(source, 1, "source Markdown file is missing, unreadable, or outside repository")
            return
        if source_resolved.suffix.casefold() != ".md":
            return
        for destination, line in extract_destinations(text):
            if not destination:
                continue
            is_drive_path = re.match(r"^[A-Za-z]:", destination) is not None
            if destination.startswith("//") or (not is_drive_path and re.match(r"^[A-Za-z][A-Za-z0-9+.-]*:", destination)):
                continue
            path_part, sep, fragment = destination.partition("#")
            path_part = path_part.split("?", 1)[0]
            if sep and _has_bad_percent_escape(fragment):
                self.error(source_resolved, line, f"malformed percent escape in anchor {fragment!r}")
                continue
            fragment = unquote(fragment) if sep else ""
            target = source_resolved if not path_part else self._case_checked_path(source_resolved, line, path_part)
            if target is None:
                continue
            if target.is_dir() and fragment:
                candidates = [target / "README.md", target / "index.md"]
                document = next((item for item in candidates if item.is_file()), None)
                if document is None:
                    self.error(source_resolved, line, f"directory fragment target has no README.md or index.md: {target.relative_to(self.repo).as_posix()}")
                    continue
                try:
                    target = document.resolve(strict=True)
                    target.relative_to(self.repo)
                except (OSError, ValueError):
                    self.error(source_resolved, line, f"directory fragment document escapes repository: {document.relative_to(self.repo).as_posix()}")
                    continue
            if sep and fragment and target.is_file() and target.suffix.casefold() == ".md":
                anchors = self._anchors_for(target)
                if fragment not in anchors:
                    self.error(source_resolved, line, f"missing anchor #{fragment} in {target.relative_to(self.repo).as_posix()}")

    def check_path(self, path: Path) -> None:
        try:
            resolved = path.resolve(strict=True)
            resolved.relative_to(self.repo)
        except (OSError, ValueError):
            self.error(path, 1, "requested documentation path is missing or outside repository")
            return
        if resolved.is_dir():
            for item in sorted(resolved.rglob("*.md"), key=lambda p: p.as_posix().casefold()):
                self.check_file(item)
        else:
            self.check_file(resolved)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, default=Path(__file__).resolve().parents[1], help="repository root (default: script's repository)")
    parser.add_argument("paths", nargs="*", help="additional Markdown files or directories, relative to --repo or absolute within it")
    args = parser.parse_args(argv)
    repo = args.repo.resolve()
    checker = DocumentationChecker(repo)
    for relative in DEFAULT_DOCUMENTS:
        checker.check_path(repo / relative)
    for value in args.paths:
        candidate = Path(value)
        checker.check_path(candidate if candidate.is_absolute() else repo / candidate)
    if checker.errors:
        for error in sorted(set(checker.errors)):
            print(error, file=sys.stderr)
        print(f"Documentation link check failed: {len(set(checker.errors))} issue(s).", file=sys.stderr)
        return 1
    print(f"Documentation link check passed ({len(DEFAULT_DOCUMENTS)} default documents, {len(args.paths)} explicit path(s)).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
