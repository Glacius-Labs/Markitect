"""Selected physical model-root rules; no source-path normalization."""
from __future__ import annotations

import os
import re
import stat
from pathlib import Path

from name_profile import name_key, require_nfc, word_matches


def root_components(value) -> tuple[str, ...]:
    if not isinstance(value, str) or not value:
        raise ValueError("model root must be a nonempty relative directory path")
    if value == ".":
        return ()
    if value.startswith("/") or re.match(r"^[A-Za-z]:", value):
        raise ValueError("model root must be relative to model.yaml")
    parts = value.split("/")
    if "\\" in value or any(part in ("", ".", "..") for part in parts):
        raise ValueError("model root must use canonical relative / components; no empty, . or .. components or backslashes")
    for part in parts:
        require_nfc(part, "model root folder")
        if not word_matches(part, "lower"):
            raise ValueError("model root folder must use the lowercase Unicode Package-folder convention")
    return tuple(parts)


def reject_redirect(path: Path) -> None:
    info = path.lstat()
    reparse = getattr(info, "st_file_attributes", 0) & getattr(stat, "FILE_ATTRIBUTE_REPARSE_POINT", 0)
    if stat.S_ISLNK(info.st_mode) or reparse:
        raise ValueError(f"symlink/reparse redirection is forbidden: {path.name!r}")


def model_root_path(manifest_path: Path, value: str) -> Path:
    parts = root_components(value)
    reject_redirect(manifest_path)
    current = manifest_path.parent
    reject_redirect(current)
    for part in parts:
        matches = sorted((entry for entry in current.iterdir() if name_key(entry.name) == name_key(part)), key=lambda path: path.name)
        if len(matches) > 1:
            raise ValueError(f"case-insensitive model root folder collision for {part!r}")
        if not matches:
            raise ValueError(f"model root directory does not exist: {part!r}")
        candidate = matches[0]
        if candidate.name != part:
            raise ValueError(f"model root folder {part!r} must match physical spelling {candidate.name!r} exactly; no automatic correction")
        reject_redirect(candidate)
        if not candidate.is_dir():
            raise ValueError(f"model root must name an existing directory: {part!r}")
        current = candidate
    return current


def reject_overlapping_roots(models) -> None:
    # Resolve trusted physical anchors for comparison, never the authored root
    # string. Root spelling and redirects have already been checked on disk.
    roots = [(identifier, record["root"].resolve(strict=True)) for identifier, record in sorted(models.items())]
    for index, (left_id, left) in enumerate(roots):
        for right_id, right in roots[index + 1:]:
            if left.is_relative_to(right) or right.is_relative_to(left) or os.path.samefile(left, right):
                raise ValueError(f"model roots overlap: {left_id!r} and {right_id!r}")
