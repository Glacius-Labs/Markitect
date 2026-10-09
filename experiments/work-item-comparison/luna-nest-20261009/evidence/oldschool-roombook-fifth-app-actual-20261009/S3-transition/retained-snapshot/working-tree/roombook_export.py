"""CSV export helpers for Roombook reservation history."""

from __future__ import annotations

import csv
import os
from pathlib import Path
from typing import Mapping, Sequence


CSV_COLUMNS = ("id", "room", "start", "end", "title", "status")


class ExportError(Exception):
    """An expected failure while exporting reservation data."""


def _normalized_path(path: Path) -> str:
    """Return a comparable absolute path, resolving existing symlink parents."""
    try:
        resolved = path.resolve(strict=False)
    except (OSError, RuntimeError):
        resolved = Path(os.path.abspath(os.fspath(path)))
    return os.path.normcase(os.path.normpath(os.fspath(resolved)))


def _aliases(database: Path, destination: Path) -> bool:
    if _normalized_path(database) == _normalized_path(destination):
        return True
    try:
        return os.path.samefile(database, destination)
    except (FileNotFoundError, NotADirectoryError, OSError):
        return False


def export_csv(
    database_path: str | os.PathLike[str],
    output_path: str | os.PathLike[str],
    reservations: Sequence[Mapping[str, object]],
) -> int:
    """Write reservations in their supplied order and return the row count.

    The database path is checked before opening the destination so a caller
    cannot accidentally truncate the JSON database by choosing it as output.
    """
    database = Path(database_path)
    destination = Path(output_path)
    if _aliases(database, destination):
        raise ExportError("CSV destination must not be the database file")

    try:
        with destination.open("w", encoding="utf-8", newline="") as handle:
            writer = csv.DictWriter(handle, fieldnames=CSV_COLUMNS, extrasaction="ignore", lineterminator="\n")
            writer.writeheader()
            for reservation in reservations:
                writer.writerow({column: reservation[column] for column in CSV_COLUMNS})
    except (OSError, UnicodeError, KeyError, TypeError, ValueError) as error:
        raise ExportError(f"cannot write CSV export: {error}") from error
    return len(reservations)
